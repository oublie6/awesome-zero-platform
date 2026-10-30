# Goal 0040: Redis resource and request tracing governance

## Status

- State: completed
- Started: 2026-10-30
- Completed: 2026-10-30
- Blockers: None for source and CI acceptance; actual new Jaeger child spans in the user's Kubernetes deployment are pending a supplementary on-server check after rollout.

## Goal

Preserve Goal 0038's go-zero SQL/Redis migration while making the necessary native exceptions observable and keeping established monitoring dashboards stable.

## Deliverables

1. `realtime.message` uses go-zero's configured tracer, creating a short per-message Span and propagating its context to application handlers, instead of relying on an hours-long WebSocket handshake context.
2. Native `database.WithinTransaction` creates an `sql.transaction` span surrounding begin and deferred commit/rollback, and native Redis refresh rotation creates `redis.watch.rotate`. Sensitive keys, messages, passwords and token digests are not attached as trace attributes.
3. Original Prometheus HTTP/WS metric families remain unchanged; new native Redis pool total/idle/stale gauges are added to the same registry and verified. These are explicitly **native pool only**, never represented as go-zero pool totals.
4. Added focused tests for Redis pool metric scrape, WebSocket message span parent propagation and native SQL transaction trace.
5. Documented independent go-zero Redis connection manager/pool behavior and tracing limitations.

## Verification

- Code checkpoint: `f29a53c8331420a63d9ec2f712fdb58bb4068d21`.
- Full main CI (unit, race, MySQL/Redis integration, code generation, formatting, production Compose runtime) **success**: https://github.com/oublie6/awesome-zero-platform/actions/runs/41021397036
- Fair Doudizhu regression **success**: https://github.com/oublie6/awesome-zero-platform/actions/runs/41021396898
- Changes were implemented and self-reviewed by ChatGPT. GitHub Actions independently executed repository checks. No Kubernetes server work was performed by ChatGPT.

## Mandatory Self-Review

Reviewed Redis DB 0 vs nonzero DB selection, retained WATCH/PubSub semantics, span parent context and defer cleanup, per-message scope, secret redaction, metric naming/cardinality and preexisting Grafana dashboard compatibility. The initial metric test overwrite was caught in second-pass self-review and original HTTP tests were restored. CI passed at the final source checkpoint.

## Remaining / Important Scope Boundary

- Real K8s Jaeger checks for the new WebSocket, SQL transaction and Redis WATCH spans require the authorized Codex server-side rollout and suitably exercised operations.
- go-zero v1.10.2's internal Redis shared client pool cannot be correctly represented by the native pool gauges. Its independent connection limits/timeouts remain version-specific operational limitations; monitor Redis connection totals during workload testing.
- Raw individual `*sql.Tx` statements and long-lived native PubSub do not automatically create individual child Spans. This change traces native transaction boundaries rather than asserting full query-by-query coverage.
- Do not replace existing secure JWT semantics or Gorilla WebSocket protocol library solely for component uniformity.

## Completion Report

Source-level observability and native fallback instrumentation are complete and CI verified. No live cluster deployment or in-cluster Jaeger verification performed by ChatGPT.
