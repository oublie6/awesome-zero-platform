# Goal 0038: go-zero First component integration

## Status

- State: completed
- Started: 2026-10-30
- Completed: 2026-10-30
- Blockers: None for source/CI acceptance. Live Kubernetes deployment and Jaeger span proof require separately authorized Codex supplementary execution.

## Goal

Implement the go-zero First standard in existing foundation and business call paths: prefer go-zero sqlx for supported SQL operations, go-zero Redis/cache for supported Redis operations, and existing go-zero components for infrastructure. Keep native clients only where go-zero v1.10.2 lacks an equivalent interface or exact transaction/subscription semantics, without weakening current behavior.

## References

- `AGENTS.md`
- `docs/operations/logging-observability.md`

## Deliverables

1. Document and apply the go-zero First standard in root AGENTS.md.
2. Provide go-zero SQL and Redis clients at the foundation lifecycle boundary; migrate directly supported application operations, preserving native fallback for genuine gaps.
3. Verify authentication session TTL/rotation, Casbin synchronization, MySQL identity/admin/game transactions, connection pools, and context tracing.
4. Add and update focused tests and explain any deliberately retained raw access.
5. Complete repository CI and document evidence. Do not deploy automatically to the live Kubernetes server.

## Constraints

- Follow AGENTS.md and work directly on main.
- Preserve public APIs, existing database schema, active sessions/keys, and game state. No destructive deployment or database migration.
- Redis WATCH/TxPipelined and long-lived Pub/Sub may stay native where no equivalent go-zero API exists.
- Preserve transaction boundaries, lock ordering, TTL precision, error identity, readiness, and shutdown.
- Do not claim tracing for native bypasses without actual instrumentation.
- Avoid expanding this goal into independent WebSocket/JWT redesign or new metrics dashboards without a demonstrated direct go-zero replacement.

## Acceptance Criteria

- AGENTS.md documents the default go-zero preference and explicit native exceptions.
- Supported migrated MySQL/Redis operations use go-zero APIs in real business call paths, not just initialization.
- Existing transactional/session/authorization behavior remains covered by tests, including concurrency and failure paths.
- `go test -p 1 -parallel 1 ./...` and affected CI jobs pass; note anything not executable in the current environment.
- Remaining native call sites are enumerated with technical justification.
- No unintended changes to K8s production resources or stored data.

## Agent Strategy

ChatGPT is primary implementer and reviewer. No concurrent repository agents. Use GitHub Actions for available validation; do not request Codex unless a specific unavailable runtime test remains.

## Working State

### Completed

- Established go-zero First standards in root `AGENTS.md` (commit `1b3b9d8`), matching common usage with go-zero v1.10.2.
- Added `sqlx.SqlConn` to database foundation over the existing `*sql.DB` pool. Migrated supported identity, admin, Casbin policy reads, Doudizhu snapshot reads, and game final-archive SQL call paths.
- Added go-zero Redis to cache foundation, migrated session CRUD/list/revoke, Casbin publish and retained go-zero model cache. Preserved native WATCH/TxPipelined, Pub/Sub, and nonzero-DB compatibility.
- Fixed go-zero JSON blob decoding through a tagged struct; explicitly named COALESCE projections for sqlx; classified expected MySQL constraint conflicts as acceptable breaker outcomes; suppressed sensitive SQL statement/Redis slow-command logs.
- Added real Redis integration coverage for subsecond session TTL, conflict handling, WATCH rotation, stale tokens, and go-zero revoke; added nonzero logical DB model-cache validation and regression coverage.
- Prepared the existing observability-server profile image tag `awesome-zero-platform/app-api:goal0038-gozero-first`; committed `docs/architecture/go-zero-first-components.md` for native exceptions and `docs/operations/goal0038-kubernetes-handoff.md` for Codex-only server rollout and real-trace checks.
- Performed a same-agent second-pass self-review of session lifecycle, Redis DB 0 vs nonzero DB, SQL row mapping, MySQL transaction isolation and locking, error preservation, logging of secret values, tracing/context coverage, Docker/K8s upgrade boundaries and tests; fixed discovered issues before acceptance.

### In progress

- No development tasks. Supplementary on-server deployment and real Jaeger tracing are handed off separately.

### Remaining

- Run the explicit environment-only Codex handoff from `docs/operations/goal0038-kubernetes-handoff.md` on the authorized server; do not conflate with completed source-level acceptance.
- Future separate goals may reduce raw transaction adapters further only with equivalent isolation and dedicated tests.

### Verification status

- Full main CI on commit `7fed9ed85abc21e474b3c4d8735b20e29cd5a288` **success**: https://github.com/oublie6/awesome-zero-platform/actions/runs/40994258994 (unit, race, integration, build, Compose runtime, report).
- Fair Doudizhu on the same SHA **success**: https://github.com/oublie6/awesome-zero-platform/actions/runs/40994258988.
- Kubernetes Observability Manifests CI **success** at relevant profile checkpoint `9765dd5645011af7f263215aa648eeb2dc78310e`: https://github.com/oublie6/awesome-zero-platform/actions/runs/40994081347.
- The Redis Session end-to-end integration test ran against CI Redis and verified interaction of go-zero client calls with native WATCH.
- No direct access to user Kubernetes cluster, SSH, runtime Jaeger UI, or true on-server app image deployment. Actual Jaeger HTTP→SQL→Redis child spans are **not yet verified live**; no in-place production data changes or deployment were made by ChatGPT.

## Completion Report

**Source and repository CI acceptance complete.** The main branch checkpoint `7fed9ed85abc21e474b3c4d8735b20e29cd5a288` passed the required main CI and Doudizhu regression. Most standard database and Redis business operations now use go-zero v1.10.2 with original pools and persisted key/schema semantics preserved.

Native exceptions intentionally retained: `database/sql` driver/startup and `*sql.Tx` identity/admin/Casbin/Doudizhu atomic transactions, Doudizhu READ COMMITTED and FOR UPDATE lock sequences, Redis WATCH+TxPipelined refresh rotation, Redis Pub/Sub subscription, and client DB != 0 fallback. All are documented in `docs/architecture/go-zero-first-components.md`. These raw operations do not automatically create go-zero child spans.

The acceptance of real Jaeger visibility and app functionality **on the user's Kubernetes cluster remains pending** and is a separate Codex-only operational check in `docs/operations/goal0038-kubernetes-handoff.md`. It must verify actual image, Pod status, HTTP/MySQL/Redis child spans, and rollback readiness, not infer them from green CI. No unauthorized server changes performed.

Mandatory self-review was performed by the same primary agent, not described as independent. It identified and corrected raw []byte scanning, SQL projection mapping, Redis DB routing, session-test cleanup order, and sensitive argument logging. Remaining caveats are the internal go-zero Redis pool's independent tuning and native-bypass tracing; no new feature scope was added.

Archiving after successful repository verification is permitted. Server deployment is a distinct supplementary environment-specific step, not completed in this Goal's code acceptance.
