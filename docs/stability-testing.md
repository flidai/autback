# Stability fault matrix

Autback's stability claims are executable through one deterministic runner. The fast tier
uses component boundaries to inject failures without privileged host mutation; the full
tier adds real Docker and Swarm resource creation/removal. Both tiers run scenarios in a
stable order, set `AUTBACK_FAULT_SEED`, bound every scenario with a context and Go test
timeout, continue after a failed scenario, and retain one log plus `manifest.json` under
`.tmp/stability/<tier>`.

```sh
task stability:fast
task stability:full

# Reproduce a recorded configuration
task stability:fast -- --seed 20260804
```

The runner prints the same evidence into CI logs. Pull requests and `main` run the fast
tier. The CI workflow's `full_stability` dispatch input runs the privileged tier on an
isolated GitHub runner; tests skip with an explicit reason when Docker is unavailable.

| Scenario | Injection boundary | Required evidence |
| --- | --- | --- |
| CAS restart during upload/download | transient preparation heartbeat plus rotating loopback proxy | retry recovers, refreshed certificate is published, reconnect does not duplicate output |
| BuildKit restart during build | transient build heartbeat and cancelled Buildx cleanup | build connection refreshes and builder cleanup remains bounded |
| Docker daemon loss/recovery | typed Docker API outage around check/list/cancel | running job is not falsely lost and concurrent recovery converges |
| Swarm node drain | exhaustive task-state injection | `shutdown`, `remove`, `orphaned`, and unknown states terminalize and release capacity |
| Server `SIGKILL` in queued/running/upload/cleanup | close/reopen durable SQLite state plus cancelled coordinator | active lease and cleanup reservation survive; stale upload and runtime converge after restart |
| Disk soft/full and inode exhaustion | deterministic capacity snapshots | soft reclaim, hard stop-before-reclaim, terminal reason, and inode pressure are distinct |
| Credential expiry/rotation | expiring control session and invalid/valid proxy updates | repeated renewal works; invalid updates do not interrupt the last valid connection |
| Partial cleanup/restart | one-shot Docker failure and cancellation during durable cleaning | remaining resources retry idempotently and FIFO stays held until cleanup succeeds |
| Workload ignores `TERM` | real process group with an event barrier and ignored `TERM` | grace expiry kills the complete group within the scenario deadline |
| Memory/PID exhaustion | Swarm limit spec, terminal runtime errors, and cgroup event fixtures | limits are present, terminal reasons are actionable, and OOM/PID evidence is operation-attributed |

The fast tier deliberately injects dependency loss at Autback's typed client boundary
instead of stopping the CI host's Docker daemon or networking. This makes the state
transition deterministic and proves the protection fails if its error handling is removed.
The full tier's real-Docker scenario verifies that the same ownership and cleanup contract
holds against actual containers, networks, volumes, services, and Swarm infrastructure.
