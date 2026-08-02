# Read-only governance console

Autback's web UI is a read-only projection of the shared service. It is not a second
control plane.

```text
SQLite durable state
  -> scoped control_changes record in the same transaction
  -> SQLite commit hook coalesces an in-process notification
  -> authorized read service re-queries a stable revision
  -> Pagestream emits Datastar signal patches on /app/updates
  -> Lit renders the console
```

Reconnects always hydrate from SQLite. The in-process notification is only a wake-up;
losing it cannot lose committed state. Snapshot reads compare the durable revision before
and after projection and retry if a concurrent transaction crossed the read, preventing a
mixed revision from being published. The journal stores only sequence, project scope,
entity kind, and entity ID, and retains the newest 10,000 records.

## Access boundary

`autback console` reads the named device token from the operating-system keyring and starts
a random loopback proxy. A one-time capability URL establishes an HttpOnly, SameSite=Strict
local session. The proxy accepts only `GET` and `HEAD` below `/app`, injects the device token
upstream, and refuses every Connect API path. The browser never receives the device token.

The server authenticates and authorizes every document and update stream. Static console
assets are public but contain no state. There are no browser mutation routes, forms,
control buttons, or client-side calls to Connect.

## Routes and signals

- `/app`: worker lease, strict global FIFO, authorized projects, and recent operations.
- `/app/projects/{project}`: project image/trust posture, global queue positions, and work.
- `/app/operations/{kind}/{id}`: provenance, timing, command, and a bounded 64 KiB log tail.
- `/app/audit`: authorized governance events.

Go console structs generate the TypeScript contracts used by Lit. The stable signal roots
are `$session`, `$service`, `$worker`, `$queue`, `$operations`, `$operation`, `$log`, `$audit`,
and `$status`. Canonical route identity remains in server URLs; Lit does not route or fetch.

## Development

`task console:check` regenerates contracts, tests formatting, typechecks TypeScript, and
builds the self-hosted Datastar and Lit assets embedded by `autback-server`. The Railway-
referenced design system is implemented as semantic tokens and reusable primitives in
`web/console-styles.ts`.

Run the exact production UI against a safe fixture with:

```console
go run ./cmd/autback-console-preview
```

The preview command refuses non-loopback listeners and never opens SQLite, Docker, or the
control API.
