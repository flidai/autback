# Operations

Clients use the HTTPS control plane. SSH is reserved for deployment, administration, and
break-glass access.

## Routine checks

```console
outback doctor
ssh <worker> sudo systemctl status outback-server outback-cas outback-buildkit outback-maintenance.timer
ssh <worker> sudo journalctl -u outback-server -u outback-cas -u outback-buildkit --since today
ssh <worker> docker service ls --filter label=outback.managed=true
curl --fail --cacert <ca.pem> https://<worker>/readyz
```

The public endpoints are:

- `443/tcp`: protobuf/Connect control API over HTTPS;
- `50052/tcp`: protocol-transparent REAPI CAS gateway requiring an active job certificate;
- `1235/tcp`: protocol-transparent BuildKit gateway requiring an active build certificate.

The actual bazel-remote and BuildKit daemons listen only on `127.0.0.1:50051` and
`127.0.0.1:1234`. The private CA key, token pepper, SQLite control state, and audit data
live under `/var/lib/outback` with service-user-only permissions.

`/healthz` is process liveness. `/readyz` returns success only when both SQLite and
Docker Swarm respond, so it is the endpoint to use for alerts and deployment verification.

## Capacity

The reused CPX32 has 4 vCPU and 8 GB RAM. Jobs default to 2 CPU and 4 GB RAM; reservations
equal limits. The single-worker service also holds an exclusive host-backed admission lock
while a project command runs. This intentionally serializes Docker/Testcontainers-heavy
jobs across projects instead of relying only on Swarm's memory arithmetic. Jobs may
materialize source concurrently but wait before executing. BuildKit is capped separately
at 3 GB; do not overlap a large image build and a memory-heavy test until measurements
justify a larger or additional worker.

CAS data lives under `/var/lib/outback/cas`; workspaces live under `/var/lib/outback/jobs`;
explicit project caches live under `/var/lib/outback/cache/<project-id>/<cache-name>`;
BuildKit uses `outback-buildkit-state`. CAS and BuildKit content caches are intentionally
shared by mutually trusted projects in the initial single-VM deployment. Control-plane
authorization and accounting remain project-scoped, but the service does not claim cache
confidentiality between these projects.

The hourly maintenance job keeps completed Swarm services for one hour and durable job
logs/workspaces for seven days. Project caches use a 12 GiB high watermark and are pruned
oldest-first to 8 GiB, but only while the exclusive worker lock is idle. At 85% filesystem
use, the same cache pruning runs and BuildKit retention tightens from 10 GiB to 4 GiB.
The shell fallback removes only terminal services older than 24 hours. Its cutoff is
independent because Swarm does not expose the task completion timestamp in `service ls`.
`OUTBACK_SERVICE_FALLBACK_RETENTION_SECONDS`, `OUTBACK_JOB_RETENTION_MINUTES`,
`OUTBACK_CACHE_HIGH_BYTES`, `OUTBACK_CACHE_LOW_BYTES`, and `OUTBACK_DISK_HIGH_PERCENT` override
these defaults for a larger worker.

The CPX32 hardening exercise—including two projects, two device identities, hosted OIDC,
cache isolation, serialized execution, restart convergence, durable logs, closed Docker
ports, and a backup/restore drill—is recorded in
[`../evidence/service/hardening.json`](../evidence/service/hardening.json).

## Deployment and first login

`task deploy:swarm` (also exposed as `task deploy:service`) requires an explicit existing
host and SSH identity. It never provisions or replaces infrastructure. It installs the
server and static job entrypoint, starts pinned CAS/BuildKit images, initializes Swarm,
bootstraps the control database once, copies the private CA locally, stores the one-time
device token in the OS keychain, then removes the bootstrap handoff file from the host.

```console
OUTBACK_SERVER_IP=62.238.54.70 \
OUTBACK_SSH_USER=developer \
OUTBACK_SSH_KEY=~/.ssh/outback-poc \
OUTBACK_PROJECT=leapview \
OUTBACK_PROJECT_NAME=LeapView \
OUTBACK_PROJECT_IMAGE=ghcr.io/example/ci@sha256:... \
task deploy:service
```

Create a separate user and one-time enrollment for every coworker laptop:

```console
outback admin user create --name coworker
outback admin member add --project leapview --user usr...
outback admin enrollment create --user usr... --device coworker-laptop --expires 10m
# On the coworker's laptop; the code is read by a hidden prompt:
outback login
outback token list
outback token revoke <token-id>
```

Use `outback admin user create`, `outback admin project create`, and `outback admin member add`
to create coworker/project identities. No user's device token is shared.

Terraform under `infra/` provisions an optional dedicated Hetzner worker and firewall.
Provisioning remains a separate explicit action; the existing-host deploy path cannot
call HCP or Hetzner APIs.

## Operation credential lifecycle and rotation

Job and build certificates contain exactly one SPIFFE identity of the form
`spiffe://outback/<job|build>/<operation-id>`. The data-plane gateway requires TLS 1.3,
the private outback CA, the expected operation kind, a currently valid certificate, and an
operation that is still active in SQLite. User tokens, GitHub session tokens, missing
certificates, wrong-kind certificates, unknown IDs, and completed/cancelled operations
therefore cannot reach CAS or BuildKit. The CLI writes each issued key bundle to a private
temporary directory and removes it after the operation; it never persists the bundle in
configuration or logs.

Routine revocation is automatic: completing or cancelling the operation makes the live
operation callback reject its certificate immediately, before its normal 15-minute expiry.
For emergency CA rotation, schedule control-plane downtime, back up `/var/lib/outback`, stop
`outback-server`, move `/var/lib/outback/pki` to a dated root-only recovery directory, restart
the service to generate a new CA/server identity, and redistribute the new `ca.pem` to
client configuration. Verify control, CAS, and BuildKit TLS before deleting the recovery
copy. Every old operation certificate becomes invalid at the gateway after the restart.

## Backup and restore

Create a consistent, private recovery bundle without copying SQLite WAL files directly:

```console
sudo install -d -o outback -g outback -m 0700 /var/backups/outback
sudo -u outback outback-server backup \
  --data-dir /var/lib/outback \
  --output /var/backups/outback/2026-08-01T120000Z
```

The bundle contains a SQLite `VACUUM INTO` snapshot, token pepper, private PKI, and a
SHA-256 manifest. Copy it to storage outside the worker and apply separate retention and
encryption there. It deliberately excludes rebuildable CAS, BuildKit layers, job
workspaces, and project caches.

Restore is offline and refuses to overwrite a data directory. Stop the service, move the
old directory to a dated recovery path, restore, fix ownership, and start the service:

```console
sudo systemctl stop outback-server
sudo mv /var/lib/outback /var/lib/outback.failed-20260801
sudo outback-server restore --input /var/backups/outback/2026-08-01T120000Z --data-dir /var/lib/outback
sudo chown -R outback:outback /var/lib/outback
sudo systemctl start outback-server
curl --fail --cacert /var/lib/outback/pki/ca.pem https://localhost/readyz
```

Restore validates every declared path and checksum before writing. Keep the moved state
until device login, OIDC exchange, CAS credentials, and a real job have been verified.

## Troubleshooting

- `outback doctor` fails at TLS: verify the configured URL/CA and server certificate names.
- CAS or BuildKit reports a rejected certificate: operation credentials expire after 15
  minutes and are invalidated when the job/build finishes; retry the CLI operation.
- A job remains queued: inspect `docker service ps --no-trunc <job-id>` for image, mount,
  or resource-placement errors.
- Testcontainers cannot connect: verify the Docker socket and identical
  `/var/lib/outback/jobs` host/container path.
- GitHub OIDC exchange is denied: compare immutable owner/repository IDs, audience,
  workflow ref, ref, environment, and event with `outback trust github list`.
