# Goal 0038: go-zero First component integration

## Status

- State: in_progress
- Started: 2026-10-30
- Completed:
- Blockers:

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

- Added go-zero First guidance in AGENTS.md (commit 1b3b9d8).

### In progress

- Inspecting and migrating directly supported data operations.

### Remaining

- Implementation, tests, CI, review, archival.

### Verification status

- Not started.

## Completion Report

Not started.
