# ADR 0003: First-class, externally resolved job secrets

- Status: Accepted
- Date: 2026-08-04
- Owners: autback project

## Context

Job commands need registry tokens, signing material, and similar credentials. Treating those
values as ordinary command arguments or environment metadata would copy them into protobuf
messages, SQLite, audit/debug output, Swarm service specifications, and recovery bundles.
Autback needs a credential boundary that remains compatible with the additive `rtest.v1`
wire contract and the trusted single-worker architecture.

## Decision

1. A job persists only a project-scoped secret reference name and either an environment-key
   target or an absolute file target below `/run/secrets`.
2. Values live in an external store mounted at `AUTBACK_SECRET_ROOT` (default
   `/run/autback/secret-store`). The initial adapter reads
   `<root>/<project-id>/<reference-name>` without following links below the configured root.
   The store is an operator-managed tmpfs or secret-provider mount, not Autback state.
3. FIFO admission resolves all references and atomically publishes an operation-scoped
   snapshot below `/var/lib/autback/jobs/<job-id>/secrets`. A missing or revoked reference
   permanently fails that admission, cleans its resources, and advances the queue.
4. Swarm receives read-only bind-mount paths, never values. The entrypoint reads the
   snapshot, injects requested process environment, and masks the exact resolved set in
   both live stdout/stderr and the durable job log. File targets are read-only mounts.
5. Cleanup removes operation-scoped material before Docker resources are released. Secret
   access audit events contain project, job, and reference name only.
6. Rotation is atomic replacement in the external store; revocation is removal. Jobs not
   yet admitted resolve the current value. A running job intentionally retains its admitted
   snapshot so rotation cannot mutate a process mid-run.
7. Control-state backups exclude both the external store and operation workspaces. Restores
   recover references only; the operator restores or remounts the external provider
   separately before admitting jobs.
8. Generic command/environment values containing `secret://...` or GitHub-style secret
   interpolation are rejected and must migrate to the dedicated fields. `AUTBACK_*`
   environment keys are server-owned and cannot be supplied by a job.

## Consequences

- `rtest.v1` gains additive `JobSecret` fields while retaining its route and existing field
  numbers.
- The control plane can authorize and audit use without becoming a secret-value database.
- Redaction is defense in depth, not authorization: trusted job code can still read its own
  declared credentials. Untrusted repositories remain outside the Swarm trust model.
- Operators must provision and back up the external provider independently, use restrictive
  ownership/modes, and keep it unavailable to clients and job containers except through
  admitted read-only snapshots.

## Rejected alternatives

- **Docker Swarm secrets as the source of truth:** values become Swarm manager state and
  couple project credential lifecycle and backup behavior to the scheduler implementation.
- **Encrypted values in SQLite:** encryption still makes control backups a credential store
  and adds key rotation/recovery complexity without improving the execution boundary.
- **Docker-in-Docker per job:** duplicates the host daemon, complicates Testcontainers and
  cache behavior, and does not solve control-plane secret persistence.
- **Best-effort masking of generic environment values:** admission cannot reliably infer
  which arbitrary strings are credentials; references must be explicit.
