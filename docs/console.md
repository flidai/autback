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

The public console redirects unauthenticated `/app` requests through GitHub OAuth with
PKCE. GitHub supplies an immutable account ID; Autback maps that ID to a pre-provisioned
user and creates its own short-lived, revocable browser session. The secure HttpOnly,
SameSite=Lax cookie is valid only for the Autback service and is never a CLI credential.
Signing out revokes that session server-side.

`autback console` remains a private alternative. It reads the named device token from the
operating-system keyring and starts a random loopback proxy. A one-time capability URL
establishes an HttpOnly, SameSite=Strict local session. The proxy accepts only `GET` and
`HEAD` below `/app`, injects the device token upstream, and refuses every Connect API path.
The browser never receives the device token.

The server authenticates and authorizes every document and update stream. Static console
assets are public but contain no state. Except for authentication and device approval,
there are no browser mutation routes, forms, control buttons, or client-side calls to
Connect. Execution and governance remain CLI-only.

## Routes and signals

- `/app`: VM utilization and one ordered Runs list spanning active, queued, and completed work.
- `/app/projects/{project}`: project image/trust posture, utilization, and the same project-scoped Runs list.
- `/app/runs/{kind}/{id}`: provenance, timing, command, and a bounded 64 KiB log tail.
- `/app/audit`: authorized governance events.

Go console structs generate the TypeScript contracts used by Lit. The stable signal roots
are `$session`, `$service`, `$worker`, `$clock`, `$resources`, `$queue`, `$operations`, `$operation`, `$log`, `$audit`,
and `$status`. Canonical route identity remains in server URLs; Lit does not route or fetch.

`<autback-runs-table>` reads `$queue`, `$operations`, and `$clock` directly through the
Datastar Lit adapter. It presents active work first, queued work in FIFO order, and then
completed work newest first. TanStack Table owns the local search and status/kind filters;
filtering does not introduce another data source or browser API.

For an authorized job detail route, the same SSE connection follows scheduler output and
patches a coalesced `$log` tail as new bytes arrive. The browser receives at most the latest
64 KiB; reconnecting hydrates that tail again from the scheduler's durable job log.

The SSE connection also publishes the server-owned `$clock` once per second. Running
durations and relative timestamps derive from that signal, so they advance without browser
timers, polling, or a full document refresh. Durable projections are still re-queried only
after their committed change notification.

The server samples whole-VM CPU, memory, and disk capacity every two seconds by default,
attributes occupied samples to the admitted run, and records the sample in the same SQLite
store. Raw two-second samples are retained for 14 days, minute rollups for 180 days, and
compact per-run averages and peaks permanently. The interval and retention windows can be
set with `AUTBACK_METRICS_INTERVAL`, `AUTBACK_METRICS_RAW_RETENTION`, and
`AUTBACK_METRICS_ROLLUP_RETENTION`.

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
