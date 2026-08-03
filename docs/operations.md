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

- `443/tcp`: protobuf/Connect control API over HTTPS;
- `50052/tcp`: protocol-transparent REAPI CAS gateway requiring an active job certificate;
- `1235/tcp`: protocol-transparent BuildKit gateway requiring an active build certificate.

The actual bazel-remote and BuildKit daemons listen only on `127.0.0.1:50051` and
`127.0.0.1:1234`. The private CA key, token pepper, SQLite control state, and audit data
live under `/var/lib/autback` with service-user-only permissions.

`/healthz` is process liveness. `/readyz` returns success only when both SQLite and
Docker Swarm respond, so it is the endpoint to use for alerts and deployment verification.

## Capacity

The reused CPX32 has 4 vCPU and 8 GB RAM. Builds and commands enter one durable FIFO, and
the control plane admits exactly one operation at a time. The admitted operation can use
the VM's available CPU and memory; there are no per-job Swarm reservations/limits or a
separate BuildKit memory cap. A command may run its own tasks concurrently, so repositories
control useful parallelism without allowing independent submissions to oversubscribe the
worker.

Queued operations consume control-plane state only. Swarm services and BuildKit credentials
are created only after admission. A one-second reconciliation loop releases completed or
lost detached jobs and immediately admits the next FIFO entry. Queue state and the active
lease are persisted in `/var/lib/autback/control.db` and survive a server restart.
Queued and running builds use a renewable lease so a killed or disconnected client cannot
block the FIFO indefinitely. The CLI renews the lease while waiting and while Buildx runs;
the server cancels a build after two minutes without a heartbeat by default
(`AUTBACK_BUILD_LEASE_TIMEOUT`).

CAS data lives under `/var/lib/autback/cas`; workspaces live under `/var/lib/autback/jobs`;
explicit project caches live under `/var/lib/autback/cache/<project-id>/<cache-name>`;
BuildKit uses `autback-buildkit-state`. CAS and BuildKit content caches are intentionally
shared by mutually trusted projects in the initial single-VM deployment. Control-plane
authorization and accounting remain project-scoped, but the service does not claim cache
confidentiality between these projects.

The hourly maintenance job keeps completed Swarm services for one hour and durable job
logs/workspaces for seven days. Project caches use a 12 GiB high watermark and are pruned
oldest-first to 8 GiB, but only while no managed job is active. At 85% filesystem
use, the same cache pruning runs and BuildKit retention tightens from 10 GiB to 4 GiB.
The host janitor removes only terminal services older than 24 hours. Its cutoff is
independent because Swarm does not expose the task completion timestamp in `service ls`.
It also removes unused anonymous Docker volumes and dangling images older than 24 hours.
BuildKit pruning targets the dedicated `autback-buildkit` daemon rather than Docker's
unrelated default builder.
`AUTBACK_ORPHAN_RETENTION_SECONDS`, `AUTBACK_JOB_RETENTION_MINUTES`,
`AUTBACK_CACHE_HIGH_BYTES`, `AUTBACK_CACHE_LOW_BYTES`, and `AUTBACK_DISK_HIGH_PERCENT` override
these defaults for a larger worker.

The CPX32 hardening exercise—including two projects, two device identities, hosted OIDC,
cache isolation, serialized execution, restart convergence, durable logs, closed Docker
ports, and a backup/restore drill—is recorded in
[`../evidence/service/hardening.json`](../evidence/service/hardening.json).

## Deployment and first login

`task deploy` (also exposed as `task deploy:service`) requires an explicit existing
host and SSH identity. It never provisions or replaces infrastructure. It installs the
server and static job entrypoint, starts pinned CAS/BuildKit images, initializes Swarm,
bootstraps the control database once, copies the private CA locally, stores the one-time
device token in the OS keychain, then removes the bootstrap handoff file from the host.

```console
AUTBACK_SERVER_IP=62.238.54.70 \
AUTBACK_SSH_USER=developer \
AUTBACK_SSH_KEY=~/.ssh/id_ed25519 \
AUTBACK_PROJECT=leapview \
AUTBACK_PROJECT_NAME=LeapView \
AUTBACK_PROJECT_IMAGE=ghcr.io/example/ci@sha256:... \
task deploy:service
```

Create a separate user and one-time enrollment for every coworker laptop:

```console
autback admin user create --name coworker
autback admin member add --project leapview --user usr...
autback admin enrollment create --user usr... --device coworker-laptop --expires 10m
# On the coworker's laptop; the code is read by a hidden prompt:
autback login
autback token list
autback token revoke <token-id>
```

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
workspaces, and project caches.

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
