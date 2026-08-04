# Operations

Clients use the HTTPS control plane. SSH is reserved for deployment, administration, and
break-glass access.

## Routine checks

```console
autback doctor
ssh <worker> sudo systemctl status autback-server autback-cas autback-buildkit autback-maintenance.timer
ssh <worker> sudo journalctl -u autback-server -u autback-cas -u autback-buildkit --since today
ssh <worker> docker service ls --filter label=autback.managed=true
curl --fail --cacert <ca.pem> https://<worker>/readyz
```

The public endpoints are:

- `443/tcp`: protobuf/Connect control API, GitHub login, and read-only console over HTTPS;
- `50052/tcp`: protocol-transparent REAPI CAS gateway requiring an active job certificate;
- `1235/tcp`: protocol-transparent BuildKit gateway requiring an active build certificate.

The actual bazel-remote and BuildKit daemons listen only on `127.0.0.1:50051` and
`127.0.0.1:1234`. The private CA key, token pepper, SQLite control state, and audit data
live under `/var/lib/autback` with service-user-only permissions.

Job secret values are deliberately outside that tree. Mount an operator-managed tmpfs or
secret-provider filesystem at `/run/autback/secret-store` (or set
`AUTBACK_SECRET_ROOT`) and create private regular files at
`<root>/<project-id>/<reference-name>`. Directories should be `0700` and values `0600`,
owned by the Autback service user. Clients submit only reference names:

```console
autback exec \
  --secret-env registry-token=REGISTRY_TOKEN \
  --secret-file signing-key=/run/secrets/signing-key \
  -- task ci
```

Replace a value atomically to rotate it and remove the file to revoke it. Queued jobs read
the value current at admission; running jobs keep their operation-scoped snapshot. A
revoked reference fails admission without blocking the next FIFO entry. Audit records name
the project, job, and reference but never the value. Inspect `job.secret.access` events when
investigating use.

`/healthz` is process liveness. `/readyz` returns success only when the process is accepting
work and both SQLite and Docker Swarm respond, so it is the endpoint to use for alerts and
deployment verification. It returns `503` as soon as shutdown starts, before listeners or
durable state are closed.

Autback uses one coordinated 15-second shutdown budget. It first marks readiness as
draining and rejects new FIFO admission, then cancels and joins HTTP, both mTLS proxies,
metrics, reconciliation, capacity maintenance, dispatcher admission, and cleanup workers.
SQLite closes last. Active Swarm jobs are detached and are not cancelled by process
shutdown; after restart, reconciliation recovers their status and resumes durable cleanup.

## Capacity

The reused CPX32 has 4 vCPU and 8 GB RAM. Builds and commands enter one durable FIFO, and
the control plane admits exactly one operation at a time. The admitted operation can use
the VM's available CPU and memory; there are no per-job Swarm reservations/limits or a
separate BuildKit memory cap. A command may run its own tasks concurrently, so repositories
control useful parallelism without allowing independent submissions to oversubscribe the
worker.

Queued operations consume control-plane state only. CAS and BuildKit credentials and Swarm
services are created only after admission. Job preparation itself is durable: the CLI waits
for a CAS connection, uploads while renewing its preparation lease, and only then starts the
runtime. A one-second reconciliation loop terminalizes completed
or lost detached jobs. A durable cleanup coordinator then releases the reservation and
admits the next FIFO entry. Queue state, cleanup attempts/errors, and the active lease are
persisted in `/var/lib/autback/control.db` and survive a server restart.
Job preparations and queued/running builds use renewable leases so a killed or disconnected
client cannot block the FIFO indefinitely. The CLI renews them while waiting, uploading, or
running Buildx. The server cancels either after two minutes without a heartbeat by default
(`AUTBACK_JOB_PREPARATION_LEASE_TIMEOUT` and `AUTBACK_BUILD_LEASE_TIMEOUT`).

CAS data lives under `/var/lib/autback/cas`; workspaces live under `/var/lib/autback/jobs`;
explicit project caches live under `/var/lib/autback/cache/<project-id>/<cache-name>`;
BuildKit uses `autback-buildkit-state`. CAS and BuildKit content caches are intentionally
shared by mutually trusted projects in the initial single-VM deployment. Control-plane
authorization and accounting remain project-scoped, but the service does not claim cache
confidentiality between these projects.

The Go capacity controller implements
[Worker Capacity Contract v1](decisions/0002-worker-capacity-contract.md). It observes free
bytes and inodes every five seconds, runs routine reconciliation every minute, and checks
synchronously before upload/build credentials or a FIFO lease are issued. Submissions are
already durable at that point, so an unrecoverable soft-floor check leaves them queued for
background retry instead of returning a terminal capacity error. The soft floor
is `max(20%, 20 GiB)`, collection targets another `max(5%, 5 GiB)` beyond that floor, and
the hard emergency floor is `max(5%, 8 GiB)`. Capacity pressure therefore blocks new
admission before it can threaten SQLite, the OS, or an in-flight source upload.
The installer explicitly sets `AUTBACK_WORKER_OWNERSHIP=exclusive`. Without that ownership
declaration the controller is observation-only, preventing local Autback development from
pruning unrelated resources on a shared Docker Desktop or OrbStack daemon.

Maintenance and FIFO admission use one inter-process gate. Routine and soft-pressure
cleanup reports `deferred` while a job or build is admitting or running, then retries once
the worker is idle. Only the hard emergency floor may stop active work, and it does so
before any Docker, cache, or BuildKit reclaim begins.

Terminal operation acknowledgements are independent of queue advancement. After durable
completion is recorded, Autback responds to the client, converges idempotent cleanup, and
then advances the FIFO in coalesced background loops. `terminalizing` and `cleaning` remain
worker-busy states; `released` is the tombstone proving teardown completed. Transient
cleanup, capacity, or scheduler errors are recorded/logged and retried; they do not turn a
completed build into a client-visible failure.

Admission also stores a baseline of non-Autback services, containers, networks, and volumes before creating the
operation runtime. Cleanup waits ten seconds for Ryuk by default, removes every unprotected
resource added during the exclusive operation window, and verifies the resulting inventory
before releasing the lease. Set `AUTBACK_RESOURCE_CLEANUP_GRACE` to tune the Ryuk window and
`AUTBACK_RESOURCE_CLEANUP_TIMEOUT` to tune the two-minute per-attempt bound. Docker outages
or partial removals leave the operation in `cleaning`; the coordinator retries and resumes
from the same baseline after restart. Images and BuildKit cache are deliberately excluded.

Runtime reconciliation is failure-isolated. A malformed Swarm service remains visible as
an actionable reconciliation error while healthy terminal jobs and stale builds continue
to converge. A daemon-wide list failure is never interpreted as proof that every service
was lost; durable job state is preserved and retried when Docker recovers. Docker cleanup
uses negotiated Engine APIs and treats typed not-found responses as idempotent success.

The CLI removes its ephemeral native Buildx remote-driver record with an independent
15-second deadline and three bounded attempts, even when the build context was cancelled.
An already-missing builder is success; a persistent removal failure is returned instead of
being silently discarded, so automation can retry without leaking local builder records.

The controller keeps terminal Swarm services for one hour through the ordinary reconciler
and job workspaces/logs for seven days. It cleans unused Docker objects, protects active and
rollback project images, applies recorded image last use, and reduces project caches from
10% to 8% of the filesystem. CAS is capped at `min(25%, 40 GiB)` with a 5% hard write limit.
BuildKit uses native reserved/max/min-free GC policies generated for the worker. Container
logs rotate at 20 MiB and durable job logs are capped at 256 MiB.

Runner containers execute as root and can therefore leave root-owned files in bind-mounted
project caches and job workspaces. The server and maintenance units receive only
`CAP_DAC_OVERRIDE`, constrained by `ProtectSystem=strict` and
`ReadWritePaths=/var/lib/autback`, so the lifecycle controller can remove those owned trees
without running the control plane as root or gaining write access outside its data root.

Inspect or exercise exactly the same implementation used by admission and background
monitoring:

```console
sudo -u autback autback-server maintain --dry-run --json
sudo systemctl start autback-maintenance.service
sudo cat /var/lib/autback/capacity.json
```

Use the systemd unit for a destructive manual run so it receives the same narrowly scoped
filesystem capability as scheduled and in-process maintenance. The unprivileged command is
appropriate for dry-run inspection only.

The hourly `autback-maintenance.timer` is a recovery invocation of the Go command, not an
independent janitor. An inter-process lock serializes it with the resident controller.

The CPX32 hardening exercise—including two projects, two device identities, hosted OIDC,
cache isolation, serialized execution, restart convergence, durable logs, closed Docker
ports, and a backup/restore drill—is recorded in
[`../evidence/service/hardening.json`](../evidence/service/hardening.json).

## Public console deployment and first login

`task deploy` (also exposed as `task deploy:service`) requires an explicit existing
host and SSH identity. It never provisions or replaces infrastructure. It installs the
server and static job entrypoint, starts pinned CAS/BuildKit images, initializes Swarm,
bootstraps the control database once, copies the private CA locally, stores the one-time
device token in the OS keychain, then removes the bootstrap handoff file from the host.
An upgrade stops the maintenance timer, any active maintenance invocation, and the old
control plane before pulling infrastructure images, preventing the outgoing lifecycle
controller from pruning a layer being installed for the incoming version.
When a public domain is configured, the server obtains and renews its control-plane
certificate with ACME TLS-ALPN on port 443. CAS and BuildKit continue using Autback's
private operation-scoped PKI.

Before deployment:

1. Create an `A` record for `console.autback.dev` pointing to the worker public IP.
2. Create a GitHub OAuth App with homepage `https://autback.dev` and callback
   `https://console.autback.dev/auth/github/callback`. The app requests no OAuth scopes.
3. Export the client ID and client secret only in the deploying shell. The deploy script
   transfers them through a mode-0600 temporary file and installs root-owned
   `/etc/autback/auth.env`; secrets never appear in process arguments or service logs.

```console
AUTBACK_SERVER_IP=62.238.54.70 \
AUTBACK_SSH_USER=developer \
AUTBACK_SSH_KEY=~/.ssh/id_ed25519 \
AUTBACK_SERVER_NAMES=console.autback.dev,62.238.54.70 \
AUTBACK_PUBLIC_URL=https://console.autback.dev \
AUTBACK_ACME_DOMAIN=console.autback.dev \
AUTBACK_ACME_EMAIL=owner@example.com \
AUTBACK_GITHUB_CLIENT_ID=... \
AUTBACK_GITHUB_CLIENT_SECRET=... \
AUTBACK_PROJECT=leapview \
AUTBACK_PROJECT_NAME=LeapView \
AUTBACK_PROJECT_IMAGE=ghcr.io/example/ci@sha256:... \
task deploy:service
```

Provision the existing owner and each coworker by immutable GitHub identity:

```console
autback admin user create --name coworker
autback admin member add --project leapview --user usr...
autback admin identity github --user usr... --login coworker-github-login
# On the coworker's laptop:
autback login
autback token list
autback token revoke <token-id>
```

The initial owner must also be bound once. Its user ID is available on the existing
administrator token:

```console
owner_user_id=$(autback token list | jq -r '.[0].user_id')
autback admin identity github --user "$owner_user_id" --login Yacobolo
```

Keep one working administrator device token until public login and device approval have
both been verified. For recovery without GitHub, create a one-time enrollment with
`autback admin enrollment create` and consume it with `autback login --recovery-code`.
Use `autback admin user create`, `autback admin project create`, and `autback admin member add`
to create coworker/project identities. No user's device token is shared.

Terraform under `infra/` provisions an optional dedicated Hetzner worker and firewall.
Provisioning remains a separate explicit action; the existing-host deploy path cannot
call HCP or Hetzner APIs.

## Operation credential lifecycle and rotation

Job and build certificates contain exactly one SPIFFE identity of the form
`spiffe://autback/<job|build>/<operation-id>`. The data-plane gateway requires TLS 1.3,
the private autback CA, the expected operation kind, a currently valid certificate, and an
operation that is still active in SQLite. User tokens, GitHub session tokens, missing
certificates, wrong-kind certificates, unknown IDs, and completed/cancelled operations
therefore cannot reach CAS or BuildKit. The CLI writes each issued key bundle to a private
temporary directory and removes it after the operation; it never persists the bundle in
configuration or logs.

Routine revocation is automatic: completing or cancelling the operation makes the live
operation callback reject its certificate immediately, before its normal 15-minute expiry.
For emergency CA rotation, schedule control-plane downtime, back up `/var/lib/autback`, stop
`autback-server`, move `/var/lib/autback/pki` to a dated root-only recovery directory, restart
the service to generate a new CA/server identity, and redistribute the new `ca.pem` to
client configuration. Verify control, CAS, and BuildKit TLS before deleting the recovery
copy. Every old operation certificate becomes invalid at the gateway after the restart.

## Backup and restore

Create a consistent, private recovery bundle without copying SQLite WAL files directly:

```console
sudo install -d -o autback -g autback -m 0700 /var/backups/autback
sudo -u autback autback-server backup \
  --data-dir /var/lib/autback \
  --output /var/backups/autback/2026-08-01T120000Z
```

The bundle contains a SQLite `VACUUM INTO` snapshot, token pepper, private PKI, and a
SHA-256 manifest. Copy it to storage outside the worker and apply separate retention and
encryption there. It deliberately excludes rebuildable CAS, BuildKit layers, job
workspaces, project caches, and `/run/autback/secret-store`. Back up or regenerate the
external secret provider under its own access and retention policy; restore/remount it
before allowing new admission. Restored control state contains references only.

Restore is offline and refuses to overwrite a data directory. Stop the service, move the
old directory to a dated recovery path, restore, fix ownership, and start the service:

```console
sudo systemctl stop autback-server
sudo mv /var/lib/autback /var/lib/autback.failed-20260801
sudo autback-server restore --input /var/backups/autback/2026-08-01T120000Z --data-dir /var/lib/autback
sudo chown -R autback:autback /var/lib/autback
sudo systemctl start autback-server
curl --fail --cacert /var/lib/autback/pki/ca.pem https://localhost/readyz
```

Restore validates every declared path and checksum before writing. Keep the moved state
until device login, OIDC exchange, CAS credentials, and a real job have been verified.

## Troubleshooting

- `autback doctor` fails at TLS: verify the configured URL/CA and server certificate names.
- CAS or BuildKit reports a rejected certificate: operation credentials expire after 15
  minutes and are invalidated when the job/build finishes; retry the CLI operation.
- A job remains queued: inspect `docker service ps --no-trunc <job-id>` for image, mount,
  or resource-placement errors.
- Testcontainers cannot connect: verify the Docker socket and identical
  `/var/lib/autback/jobs` host/container path.
- GitHub OIDC exchange is denied: compare immutable owner/repository IDs, audience,
  workflow ref, ref, environment, and event with `autback trust github list`.
