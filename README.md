# autback

`autback` moves CPU- and memory-heavy trusted-project work off developer laptops and
GitHub-hosted runners without defining a proprietary execution protocol:

- source transfer uses Remote Execution API v2 CAS/ByteStream;
- test execution, lifecycle, logs, and cancellation use Docker Swarm jobs;
- image builds use an upstream BuildKit daemon through native Docker Buildx;
- test environments run in project-selected, digest-pinned OCI images;
- Testcontainers owns integration-test dependencies.

```console
autback exec --cache go-build=/root/.cache/go-build --cache go-mod=/go/pkg/mod -- go test -count=1 -race ./...
autback exec -- task test
autback build -- --push -t ghcr.io/example/service:sha .
autback image build --tag ghcr.io/example/ci:autback --file Dockerfile.autback
```

Git selects tracked and untracked non-ignored files from the exact worktree. The REAPI CAS
uploads only missing content, so a repeated action transfers zero unchanged file bytes.
Test action results are deliberately not cached because Testcontainers and other external
services make them non-hermetic. Projects may explicitly declare persistent OCI-directory
caches with repeatable `--cache NAME=/absolute/container/path` flags. The server scopes
each writable directory to the immutable project ID; cache names never imply a language
or tool. Docker/BuildKit layers remain persistent on the worker and are governed by the
[worker capacity contract](docs/decisions/0002-worker-capacity-contract.md).

The target service does not supply a language runner. Each project selects a
digest-pinned OCI image, while autback injects only its static CAS/materialization
entrypoint and mounts the Docker socket for sibling Testcontainers. This is a
trusted-code design: Docker access is host control, not a security sandbox.

## Target project contract

autback executes arbitrary argument vectors; repositories keep task composition in their
existing Taskfile, scripts, Makefile, or CI configuration:

```console
autback init --project example
autback image activate --project example --image ghcr.io/example/ci@sha256:...
autback exec --project example -- task test
autback exec --project example -- go test -race ./...
```

The committed `autback.json` contains only the autback project identifier. It is discovered
from the nearest directory up to the Git root and is safe to commit; flags and
`AUTBACK_PROJECT` can override it. Autback accepts no suite/profile file and supplies no
language runner: repositories own commands and project images. See [the architecture](docs/architecture.md) and
[ADR 0001](docs/decisions/0001-shared-service-architecture.md) and
[ADR 0002](docs/decisions/0002-worker-capacity-contract.md). The versioned service
contract is documented in [Control API v1](docs/control-api.md).

## Existing-host deployment

The current proof runs on the existing CPX32 `leapview-development`. Deployment requires
an explicit existing host and never provisions a replacement:

```console
cd autback
task test
AUTBACK_SERVER_IP=62.238.54.70 AUTBACK_SSH_USER=developer \
AUTBACK_SSH_KEY=~/.ssh/id_ed25519 task deploy
AUTBACK_SERVER_IP=62.238.54.70 AUTBACK_SSH_USER=developer \
AUTBACK_SSH_KEY=~/.ssh/id_ed25519 AUTBACK_PROJECT=leapview task e2e:service
```

The control plane serves Connect over HTTPS on 443. Protocol-transparent mTLS gateways on
50052 and 1235 expose upstream REAPI CAS and BuildKit only to active job/build certificates.
The CLI never receives Docker access. Swarm remains private to the server and provides
detached job identity, logs, status, cancellation, and resource-aware queuing.

## Sign in from a developer laptop

An administrator creates the Autback user, grants project membership, and binds the
user's immutable GitHub account ID. The GitHub login is resolved once by the server and
is retained only as display metadata:

```console
autback admin user create --name coworker
autback admin member add --project example --user usr...
autback admin identity github --user usr... --login coworker-github-login
# Later, revoke the binding and every active human credential:
autback admin identity revoke --user usr...
```

The coworker runs `autback login`. The CLI opens Autback's GitHub sign-in and approval page,
then receives one independent device token and stores it in macOS Keychain, Linux Secret
Service, or Windows Credential Manager through the operating-system keyring. GitHub proves
human identity; it never becomes the durable CLI credential. `autback logout` removes the
local entry and `autback token revoke <id>` revokes one laptop server-side. A single-use
enrollment code remains available only as the documented recovery path through
`autback login --recovery-code`.

## Read-only governance console

Open the configured public service URL or run `autback console` to open the live service
console. The public console authenticates through GitHub and keeps an independent,
revocable Autback browser session in a secure HttpOnly cookie. The CLI command remains a
private loopback alternative that injects the device credential into proxied `/app`
requests; the browser never receives the Keychain token in either model and cannot reach
the Connect control API.
The console is deliberately read-only: all execution, cancellation, enrollment, trust,
and image commands remain in the CLI and audit log.

For design-system and browser work without a running worker, use the loopback-only fixture:

```console
go run ./cmd/autback-console-preview
```

See [the console architecture](docs/console.md) for routes, live consistency, and signal
contracts.

The manual GitHub Actions POC exchanges GitHub OIDC directly for a short-lived project
credential; see [GitHub Actions](docs/github-actions.md). It deliberately has no
pull-request trigger until the protected environment and trusted-change policy are proven.

## Proven boundary

The E2E proves dirty and untracked bytes, ignored-file exclusion, Redis through
Testcontainers, same-path sibling bind mounts, zero-byte repeat CAS upload, persisted
logs, immediate cancellation, server-side timeout, durable FIFO queuing, cleanup, and
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
runs the same checked-in argv contract serially across available local, autback, and Depot
candidates while preserving raw logs and an exact source fingerprint.

## Layout

- `cmd/autback`: project-aware CLI.
- `api/rtest/v1`: the frozen deployed v1 protobuf ABI for the Connect control API.
- `internal/control`: projects, credentials, OIDC, authorization, audit, PKI, and scheduling.
- `internal/cas`, `internal/workspace`: standard CAS transfer and exact Git input selection.
- `internal/control/swarmscheduler`: the server-owned Swarm scheduling port and job contract.
- `internal/adapter/docker`: typed, version-negotiated Docker Engine adapters for jobs, capacity, and cleanup.
- `internal/adapter/buildkit`: typed server-side BuildKit health and cache collection.
- `internal/buildkit`: thin client-facing native Buildx remote-builder wrapper.
- `internal/console`, `web`: authenticated Gomponents/Pagestream transport and Lit console UI.
- `cmd/autback-job-entrypoint`: CAS materialization, timeout, process-group, and result boundary.
- `host`: idempotent existing-host installation and systemd units.
- `action/setup-autback`: GitHub composite action that installs and configures the CLI.
- `cmd/autback-benchmark`: generic controlled command-comparison harness.
- `scripts`: deployment and reproducible E2E proof.
- `infra`: vendor-specific Terraform for an optional dedicated Hetzner worker.

Autback is an independent service. LeapView remains its first production consumer and its
checked-in benchmark evidence demonstrates the migration from Depot and laptop-bound Docker.

## Project site

The dependency-free static project site lives in [`site/`](site/). All asset references are
relative so the same artifact works at a GitHub project Pages path or a custom domain.
`go test ./internal/site` checks the page contract, local references, anchors, and Pages workflow.

After the repository's Pages source is set to **GitHub Actions**, pushes to `main` that touch
the site publish the directory through [the Pages workflow](.github/workflows/pages.yml).
The workflow can also be started manually. Preview the exact artifact locally with any
static file server, for example:

```console
python3 -m http.server --directory site 8000
```
