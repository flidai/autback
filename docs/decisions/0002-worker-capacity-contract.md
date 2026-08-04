# ADR 0002: Worker capacity and lifecycle contract

- Status: Accepted
- Date: 2026-08-03
- Owners: autback project

## Context

Autback deliberately admits one bounded operation at a time. The operation may use most
of the worker's CPU and memory, but it shares finite resources with Docker images and
logs, BuildKit, the REAPI CAS, project caches, job workspaces, and the control plane. The
original hourly shell janitor used independent age and byte thresholds, parsed Docker Go
timestamps with platform-specific `date` syntax, and could report success after failing to
reclaim anything. The worker consequently reached 91% filesystem use and required manual
image pruning.

Mature systems converge on the same lifecycle primitives:

- Nomad collects on a timer, terminal transitions, pressure, and pre-admission, deleting
  the oldest eligible terminal allocation and remeasuring after each deletion.
- Kubernetes uses byte and inode pressure, soft and hard thresholds, and a minimum reclaim
  beyond the threshold to prevent oscillation.
- BuildKit and Dagger combine reserved space, maximum used space, minimum free space,
  serialized collection, in-use protection, and tiered LRU policies.
- bazel-remote provides an eviction limit plus a hard write limit so upload bursts cannot
  exhaust the host before eviction catches up.

Autback does not need Kubernetes or Nomad to obtain these properties. It already owns the
single-worker FIFO, operation state, and trusted Docker lifecycle.

## Decision

### Ownership boundary

An Autback worker is a closed ownership domain. Its Docker daemon and `/var/lib/autback`
are used only by Autback infrastructure and trusted Autback jobs. They are not shared with
unrelated manually managed applications. This permits safe removal of unused Docker
objects while Docker itself continues to protect anything referenced by a container.
The installed worker declares this boundary with `AUTBACK_WORKER_OWNERSHIP=exclusive`.
Without that explicit declaration—most importantly during local development—the same
controller runs in observation/dry-run mode and cannot prune a shared Docker daemon.

Every Swarm service carries Autback project and job labels. Active and previous project
images are protected from pressure GC using durable project state; other images use the
most recent durable job reference as their LRU timestamp. A five-minute creation grace
prevents removal races. Testcontainers resources and arbitrary unused job-created Docker
objects are disposable after the operation terminates.

### Global thresholds

The controller derives thresholds from the filesystem containing Autback and Docker:

```text
soft free-space floor = max(20% of total, 20 GiB)
post-reclaim target   = soft floor + max(5% of total, 5 GiB)
hard emergency floor = max(5% of total, 8 GiB)
minimum free inodes   = 10%
pressure observation  = every 5 seconds
pressure GC throttle  = 30 seconds
routine reconciliation = every minute
```

On the current approximately 150 GiB worker this means a 30 GiB soft floor, a 37.5 GiB
post-reclaim target, and an 8 GiB hard floor.

Autback durably appends job and build preparations to the FIFO before measuring capacity.
Before issuing CAS or BuildKit credentials and before leasing the FIFO head, Autback
measures capacity. Admission and maintenance share an inter-process gate through the FIFO
reservation, so there is no idle-check/admission race. Below the soft floor it
synchronously reclaims toward the post-reclaim target only while the worker is idle. If
an operation is admitting or active, routine and soft-pressure collection is deferred;
Docker, BuildKit, and project-cache resources are never pruned underneath live work. If
the soft floor still cannot be restored once idle, the operation remains queued and the
background dispatcher retries; the accepted client request does not become a terminal CI
failure. Below the hard floor, Autback
atomically terminalizes the one admitting or active operation, revokes its data-plane
access through the durable operation lease, stops its Swarm runtime when applicable, and
only then reclaims.

### Reclaim order

The serialized controller uses the same inter-process lock for admission and maintenance
and remeasures after each tier:

1. terminal job workspaces older than seven days, selected from SQLite rather than file
   timestamps;
2. stopped containers, unused networks, and unused volumes;
3. unprotected Docker images ordered by durable last use;
4. project caches ordered by last use, from a 10% high watermark to an 8% low watermark;
5. BuildKit through its native collector, retaining a 2 GiB floor under pressure.

Terminal job and build transitions commit and acknowledge before this work begins. The
durable operation then moves through `terminalizing` and `cleaning`; those states retain
the worker reservation until idempotent teardown reaches the `released` tombstone. Cleanup
and FIFO advancement run asynchronously and retry transient cleanup, capacity, or runtime
failures, so a successful image push cannot be reported as failed merely because teardown
or reclaiming enough space for the next queued operation exceeds an RPC deadline.

Routine collection uses a 24-hour object age. Pressure collection uses a five-minute
creation grace. Docker image removal is never forced, so images referenced by any
container remain protected even when application metadata is incomplete.

### Bounded producers

- The Docker daemon's default cgroup parent is `autback-workloads.slice`. That outer
  slice bounds the Swarm job and every sibling container a trusted integration test
  creates through the mounted Docker socket. CAS and BuildKit explicitly use
  `autback-infrastructure.slice`; the server runs in its own bounded systemd service.
- On the documented four-core, 8 GiB worker, defaults reserve one CPU and 2 GiB of memory
  from the workload slice. An individual job receives a three-CPU/roughly-5-GiB limit,
  one CPU and 1 GiB reservations, and 4096 PIDs. The installer derives these limits from
  the host and accepts `AUTBACK_WORKLOAD_*`, `AUTBACK_JOB_*`, and
  `AUTBACK_BUILDKIT_*` overrides for other worker shapes. Reservations may not exceed
  limits and an unbounded job is rejected.
- `/proc/pressure` CPU, memory, and I/O PSI values, cgroup v2 memory-high/OOM events and
  PID use, and filesystem byte/inode use are sampled durably. Counter deltas are assigned
  to the active FIFO operation; pressure observed while idle is explicitly reported as
  host-only rather than falsely attributed.
- CAS normal size is `min(25% of the filesystem, 40 GiB)`. Its hard write limit is 5%
  larger, causing uploads to fail instead of exhausting the host.
- BuildKit uses a generated `buildkitd.toml`: 2 GiB reserved, maximum
  `min(10% of the filesystem, 10 GiB)`, 20% minimum free space, targeted 48-hour source
  and cache-mount collection, followed by an all-record LRU policy.
- Project caches use 10%/8% high and low watermarks.
- Docker infrastructure and job logs use the rotating `local` driver with two 10 MiB
  files.
- The durable Autback job log is capped at 256 MiB and records an explicit truncation
  marker while live stdout/stderr streaming continues.

### One implementation

The Go capacity controller is the authoritative implementation for admission, background
pressure monitoring, routine maintenance, emergency handling, and manual operation.
`autback-server maintain --dry-run --json` exercises the same code. The hourly systemd
timer is only a recovery invocation of that command; it contains no cleanup policy.

The latest state is atomically written to `/var/lib/autback/capacity.json`. `/healthz`
remains process liveness. `/readyz` becomes unavailable below the hard floor or when the
capacity filesystem cannot be measured.

## Consequences

- Jobs retain access to most of the worker while cgroup envelopes preserve CPU, memory,
  and process headroom for Docker and the control plane. Disk and inode safety remains a
  host-wide capacity boundary rather than a per-job quota.
- Normal cleanup may wait for a long operation to finish. Bounded producers and the hard
  emergency floor keep that delay safe without risking corruption of active work.
- Warm project images, rollback images, CAS content, and BuildKit records have explicit
  protection or native LRU behavior.
- Every persistent or Docker-backed producer has a bound, so no manual janitor is required
  during normal operation.
- A worker hosting unrelated Docker workloads violates the deployment contract. Such
  workloads require a separate daemon or VM before they can share a physical host.
- A separate data disk can protect the OS filesystem on future deployments, but is not
  required by the controller; its thresholds apply to the configured capacity path.

## References

- [Nomad garbage collection](https://developer.hashicorp.com/nomad/docs/manage/garbage-collection)
- [Kubernetes node-pressure eviction](https://kubernetes.io/docs/concepts/scheduling-eviction/node-pressure-eviction/)
- [BuildKit garbage collection](https://docs.docker.com/build/cache/garbage-collection/)
- [Dagger Engine configuration](https://docs.dagger.io/reference/configuration/engine/)
- [bazel-remote hard size limit](https://github.com/buchgr/bazel-remote)
- [Docker local logging driver](https://docs.docker.com/engine/logging/drivers/local/)

## Rejected alternatives

- **Kubernetes or Nomad solely for cleanup:** adds a second scheduler and control plane
  without improving the trusted single-worker execution model.
- **VM-per-job:** provides stronger isolation but discards the persistent warm caches that
  make Autback competitive with Depot and is unnecessary for the accepted trust boundary.
- **Independent shell janitor:** duplicates lifecycle policy, cannot participate in
  admission, and previously hid timestamp parsing failures.
- **Fixed byte budgets only:** do not scale across vendor-neutral VPS sizes.
