# outback

`outback` moves CPU- and memory-heavy trusted-project work off developer laptops and
GitHub-hosted runners without defining a proprietary execution protocol:

- source transfer uses Remote Execution API v2 CAS/ByteStream;
- test execution, lifecycle, logs, and cancellation use Docker Swarm jobs;
- image builds use an upstream BuildKit daemon through native Docker Buildx;
- test environments run in project-selected, digest-pinned OCI images;
- Testcontainers owns integration-test dependencies.

```console
outback exec --cache go-build=/root/.cache/go-build --cache go-mod=/go/pkg/mod -- go test -count=1 -race ./...
outback exec -- task test
outback build -- --push -t ghcr.io/example/service:sha .
outback image build --tag ghcr.io/example/ci:outback --file Dockerfile.outback
```

Git selects tracked and untracked non-ignored files from the exact worktree. The REAPI CAS
uploads only missing content, so a repeated action transfers zero unchanged file bytes.
Test action results are deliberately not cached because Testcontainers and other external
services make them non-hermetic. Projects may explicitly declare persistent OCI-directory
caches with repeatable `--cache NAME=/absolute/container/path` flags. The server scopes
each writable directory to the immutable project ID; cache names never imply a language
or tool. Docker/BuildKit layers remain persistent on the worker.

The target service does not supply a language runner. Each project selects a
digest-pinned OCI image, while outback injects only its static CAS/materialization
entrypoint and mounts the Docker socket for sibling Testcontainers. This is a
trusted-code design: Docker access is host control, not a security sandbox.

## Target project contract

outback executes arbitrary argument vectors; repositories keep task composition in their
existing Taskfile, scripts, Makefile, or CI configuration:

```console
outback init --project example
outback image activate --project example --image ghcr.io/example/ci@sha256:...
outback exec --project example -- task test
outback exec --project example -- go test -race ./...
```

The committed `outback.json` contains only the outback project identifier. It is discovered
from the nearest directory up to the Git root and is safe to commit; flags and
`OUTBACK_PROJECT` can override it. Outback accepts no suite/profile file and supplies no
language runner: repositories own commands and project images. See [the architecture](docs/architecture.md) and
[ADR 0001](docs/decisions/0001-shared-service-architecture.md). The versioned service
contract is documented in [Control API v1](docs/control-api.md).

## Existing-host deployment

The current proof runs on the existing CPX32 `leapview-development`. Deployment requires
an explicit existing host and never provisions a replacement:

```console
cd outback
task test
OUTBACK_SERVER_IP=62.238.54.70 OUTBACK_SSH_USER=developer \
OUTBACK_SSH_KEY=~/.ssh/id_ed25519 task deploy
OUTBACK_SERVER_IP=62.238.54.70 OUTBACK_SSH_USER=developer \
OUTBACK_SSH_KEY=~/.ssh/id_ed25519 OUTBACK_PROJECT=leapview task e2e:service
```

The control plane serves Connect over HTTPS on 443. Protocol-transparent mTLS gateways on
50052 and 1235 expose upstream REAPI CAS and BuildKit only to active job/build certificates.
The CLI never receives Docker access. Swarm remains private to the server and provides
detached job identity, logs, status, cancellation, and resource-aware queuing.

## Enroll a developer laptop

An administrator creates the user, grants project membership, and generates a ten-minute
single-use code:

```console
outback admin user create --name coworker
outback admin member add --project example --user usr...
outback admin enrollment create --user usr... --device coworker-laptop --expires 10m
```

The coworker runs `outback login` and enters that code at the hidden prompt. outback exchanges
it once and stores the resulting named device token in macOS Keychain, Linux Secret
Service, or Windows Credential Manager through the operating-system keyring. `outback logout`
removes the local entry; `outback token revoke <id>` independently revokes one laptop.

The manual GitHub Actions POC exchanges GitHub OIDC directly for a short-lived project
credential; see [GitHub Actions](docs/github-actions.md). It deliberately has no
pull-request trigger until the protected environment and trusted-change policy are proven.

## Proven boundary

The E2E proves dirty and untracked bytes, ignored-file exclusion, Redis through
Testcontainers, same-path sibling bind mounts, zero-byte repeat CAS upload, persisted
logs, immediate cancellation, server-side timeout, capacity-aware queuing, cleanup, and
native Buildx/BuildKit.
Shared-service local evidence is in
[`evidence/service-local/`](evidence/service-local/); CPX32 evidence is written to
[`evidence/service/`](evidence/service/).

For repeatable latency measurements, `task benchmark -- <command>` records one or more
unmeasured priming runs followed by warm-cache samples. It preserves `-count=1` commands
unchanged, reports end-to-end CLI and remote execution time, and rejects a warm benchmark
if any measured source transfer is non-zero.
The current CPX32 warm-cache baseline and methodology are documented in
[benchmarks](docs/benchmarks.md).
For controlled provider comparisons, `task benchmark:compare -- --spec ... --output ...`
runs the same checked-in argv contract serially across available local, outback, and Depot
candidates while preserving raw logs and an exact source fingerprint.

## Layout

- `cmd/outback`: project-aware CLI.
- `api/rtest/v1`: the frozen deployed v1 protobuf ABI for the Connect control API.
- `internal/control`: projects, credentials, OIDC, authorization, audit, PKI, and scheduling.
- `internal/cas`, `internal/workspace`: standard CAS transfer and exact Git input selection.
- `internal/swarm`: server-private Docker job specifications and lifecycle operations.
- `internal/buildkit`: thin native Buildx remote-builder wrapper.
- `cmd/outback-job-entrypoint`: CAS materialization, timeout, process-group, and result boundary.
- `host`: idempotent existing-host installation and systemd units.
- `action/setup-outback`: GitHub composite action that installs and configures the CLI.
- `cmd/outback-benchmark`: generic controlled command-comparison harness.
- `scripts`: deployment and reproducible E2E proof.
- `infra`: vendor-specific Terraform for an optional dedicated Hetzner worker.

Outback is an independent service. LeapView remains its first production consumer and its
checked-in benchmark evidence demonstrates the migration from Depot and laptop-bound Docker.
