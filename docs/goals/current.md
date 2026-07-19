# Goal 0004: Replace PostgreSQL with MySQL

## Status

- State: completed
- Started: 2026-07-19
- Completed: 2026-07-19
- Blockers:

Supported states: `idle`, `ready`, `in_progress`, `completed`, `blocked`. New executable goals start as `ready`.

## Goal

Replace the active PostgreSQL infrastructure with MySQL as the platform's single default relational database while preserving Redis, the existing lifecycle and readiness behavior, the narrow transaction boundary, deterministic schema and seed workflows, low-memory execution constraints, and all previously established HTTP behavior.

## References

- `AGENTS.md`
- `docs/architecture/overview.md`
- `docs/requirements/server-bootstrap.md`
- `docs/requirements/http-foundation.md`
- `docs/requirements/infrastructure-foundation.md`
- `server/README.md`
- `deploy/README.md`
- `deploy/local/docker-compose.yml`
- `server/database/schema/current.sql`
- `server/database/seed/development.sql`
- `server/apps/app-api/etc/main-api.yaml`
- `server/apps/app-api/internal/bootstrap`
- `server/apps/app-api/internal/config`
- `server/apps/app-api/internal/svc`
- `server/apps/app-api/internal/logic/health`
- `server/foundation/database`
- `server/foundation/cache`
- `server/foundation/readiness`
- `Makefile`
- `server/go.mod`
- `server/go.sum`

## Deliverables

1. Replace the PostgreSQL implementation under `server/foundation/database` with a focused MySQL implementation using a maintained Go driver and an explicit `database/sql` lifecycle where appropriate.
2. Replace PostgreSQL configuration with validated MySQL configuration, including address, database, credentials, `utf8mb4`, time parsing, time zone, connection timeout, pool sizes, connection lifetime, startup timeout, and readiness timeout.
3. Preserve the narrow transaction runner contract: context propagation, commit on success, rollback on returned error, rollback on panic, and useful error preservation.
4. Update `app-api` bootstrap, service context, partial-startup cleanup, shutdown, and readiness wiring so MySQL and Redis are the required dependencies.
5. Preserve process-only liveness and dependency-aware readiness: MySQL or Redis failure must produce readiness HTTP 503 without changing liveness HTTP 200.
6. Replace the local PostgreSQL service with a pinned MySQL service, including health checks, development-only credentials, `utf8mb4`, transient data, and conservative settings suitable for a 1–2 GB development machine.
7. Convert the current complete schema and optional development seed to deterministic MySQL syntax without introducing business tables, migration history, or compatibility SQL.
8. Update root commands for local dependency lifecycle, schema application, seed application, reset, and serial integration testing against MySQL and Redis.
9. Replace PostgreSQL-focused unit and integration tests with equivalent MySQL coverage, including connectivity, startup failure, partial cleanup, transactions, schema initialization, liveness, readiness, and dependency outage behavior.
10. Remove active PostgreSQL driver dependencies, code, configuration, compose definitions, tests, and documentation references outside archived historical goals.
11. Update `server/README.md`, `deploy/README.md`, and any other active documentation needed to state that MySQL is the single default relational database.
12. Verify the final repository has no active dual-database abstraction, PostgreSQL compatibility switch, unused PostgreSQL code, or stale active documentation.

## Constraints

- Follow `AGENTS.md` and every referenced document.
- Respect the low-memory execution rules: run code generation, compilation, tests, Docker verification, and agent work sequentially; use `go test -p 1 -parallel 1`; do not run Docker verification concurrently with Go compilation or tests.
- Preserve the modular-monolith architecture, go-zero lifecycle, HTTP middleware order, response conventions, request ID behavior, and probe-friendly health payload policy.
- MySQL must be the only active relational database after this goal. Do not retain runtime PostgreSQL support, dual-driver configuration, driver factories, portability layers, or lowest-common-denominator SQL.
- Preserve Redis without adding session, lock, rate-limit, queue, pub/sub, or business caching semantics.
- Do not create generic repositories, generic CRUD services, active-record models, unit-of-work frameworks, dependency-injection frameworks, or plugin systems.
- Do not add users, identities, roles, permissions, authentication, authorization, JWT, sessions, Casbin, files, notifications, audit business features, or business modules.
- Do not invent business tables or fake user, role, permission, or product seed data.
- Do not create migration history or commit temporary environment-upgrade SQL. Store only the current complete MySQL schema and optional seed.
- Do not add Kafka, object storage, message queues, distributed locks, rate limiting, outbox, sagas, distributed transactions, metrics, tracing backends, CI, Kubernetes, or production deployment resources.
- Do not commit real credentials, runtime database data, container volumes, generated binaries, logs, coverage artifacts, or local environment files containing secrets.
- Pin explicit compatible dependency and container image versions rather than floating `latest` tags.
- Do not expose credentials, DSNs, internal addresses, raw driver errors, Redis errors, or stack traces in HTTP responses or unsafe startup messages.
- Archived Goal 0003 is historical and must not be modified merely to replace PostgreSQL wording.
- Do not expand or reinterpret this goal without explicit user instruction.
- Codex may update only the Status, Working State, and Completion Report sections of this file unless a deliverable explicitly requires another documentation update.

## Acceptance Criteria

1. `make generate` succeeds and is repeatable without unintended diffs.
2. `make fmt` succeeds.
3. `make test` succeeds serially; equivalently, `cd server && go test -p 1 -parallel 1 ./...` passes.
4. The pinned local MySQL and Redis services start sequentially and reach healthy state on the low-memory development machine.
5. The current MySQL schema applies successfully to a clean database using the documented root command.
6. Optional seed application succeeds and contains no product, user, role, or permission data.
7. Serial integration tests pass against an isolated clean MySQL and Redis environment.
8. With healthy MySQL and Redis, `make run` starts `app-api`; liveness returns HTTP 200 with `{"status":"ok"}` and readiness returns HTTP 200 with `{"status":"ready"}`.
9. If MySQL becomes unavailable after startup, liveness remains HTTP 200 and readiness becomes HTTP 503 with `{"status":"unready"}` within the configured timeout.
10. If Redis becomes unavailable after startup, liveness remains HTTP 200 and readiness becomes HTTP 503 with `{"status":"unready"}` within the configured timeout.
11. Dependency failure responses and logs follow the existing safe disclosure policy and do not expose credentials, DSNs, raw errors, or internal addresses to clients.
12. Invalid MySQL address, database, pool, timeout, character-set, or readiness configuration produces a clear non-zero startup failure without printing sensitive values.
13. A startup sequence in which MySQL opens but Redis initialization fails closes the MySQL resource and returns a safe startup error.
14. MySQL and Redis close operations are safe and idempotent, including after partial initialization.
15. Transaction tests prove commit on success, rollback on returned error, rollback on panic, context propagation, and useful error preservation using MySQL.
16. The local MySQL service uses pinned versioning, health checks, transient storage, `utf8mb4`, development-only credentials, and conservative memory settings.
17. `server/database/schema/current.sql` contains deterministic MySQL syntax and no PostgreSQL-specific types, casts, extensions, sequences, or statements.
18. Active code and configuration contain no PostgreSQL driver imports, PostgreSQL configuration blocks, PostgreSQL compose service, or PostgreSQL-specific integration setup.
19. Active documentation outside `docs/goals/archive/` contains no stale claim that PostgreSQL is the default or required database.
20. `server/go.mod` and `server/go.sum` no longer retain direct PostgreSQL dependencies that are unnecessary after the replacement.
21. No dual-database abstraction, migration-history directory, business table, generic repository framework, runtime volume, secret, or unrelated capability is introduced.
22. Final Git inspection shows only MySQL replacement work, necessary tests, configuration, local dependency definitions, schema/seed changes, and documentation updates.
23. All verified changes are committed and pushed to the configured upstream without force pushing.

## Agent Strategy

The primary agent owns the replacement design, database lifecycle, transaction semantics, readiness behavior, integration testing, cleanup of PostgreSQL remnants, final verification, commit, and push. Use at most one implementation subagent or one lightweight review subagent at a time because the development machine has approximately 1–2 GB of available memory. Do not run subagents concurrently with Docker-backed tests, Go compilation, or code generation. Do not allow multiple agents to modify the same files concurrently.

## Execution Process

1. Synchronize according to `AGENTS.md`, read every reference completely, and inspect the current PostgreSQL implementation, dependencies, configuration, compose service, schema, tests, and documentation.
2. Produce a concise replacement plan before editing, including chosen MySQL driver and version, DSN construction, validation, pool behavior, transaction mapping, cleanup order, compose settings, schema conversion, dependency removal, and serial verification sequence.
3. Set the Goal state to `in_progress` and implement the MySQL database foundation without creating a generic driver abstraction.
4. Replace application configuration and lifecycle wiring while preserving Redis and existing HTTP behavior.
5. Convert schema, seed, local compose, Makefile commands, tests, and documentation.
6. Search active files for PostgreSQL references and remove all stale runtime, dependency, configuration, test, and documentation remnants while leaving archived goals unchanged.
7. Verify sequentially to control memory: generation, formatting, unit tests, dependency startup, schema, seed, integration tests, real startup, healthy probes, MySQL outage, Redis outage, graceful shutdown, and dependency cleanup. Never run these heavy stages concurrently.
8. Inspect client responses and logs for sensitive-data leakage and inspect the final dependency graph for unnecessary PostgreSQL packages.
9. Inspect the final Git diff for scope expansion, dual-database abstractions, migration history, runtime artifacts, unpinned images, secrets, or accidental edits to archived goals.
10. Update the permitted sections below with concrete evidence, commit, and push according to `AGENTS.md`.
11. Stop only when every acceptance criterion passes and the verified commit is pushed, or a genuine blocker is documented with evidence while preserving safe local work.

## Working State

### Completed

- Goal 0003 accepted and archived.
- Infrastructure requirements updated to make MySQL the single default relational database.
- Goal 0004 execution contract prepared.
- PostgreSQL runtime support was replaced with a MySQL `database/sql` foundation using `github.com/go-sql-driver/mysql` `v1.10.0`.
- `app-api` configuration, bootstrap lifecycle, service context, readiness wiring, tests, schema, seed, local dependency definition, and active documentation were converted from PostgreSQL to MySQL.
- Active PostgreSQL driver dependencies, code paths, SQL syntax, configuration blocks, compose service definitions, and non-archived documentation references were removed.
- Sequential low-memory verification completed against pinned MySQL and Redis containers.

### In progress

- None.

### Remaining

- None.

### Verification status

- `make generate` succeeded and remained repeatable without unintended diffs.
- `make fmt` succeeded.
- `make test` succeeded serially with `go test -p 1 -parallel 1 ./...`.
- `make deps-reset` started pinned MySQL `8.4.10` and Redis `8.8.0-alpine3.23` sequentially and both reached healthy state on the low-memory development machine.
- `make schema-apply` succeeded against a clean local MySQL database.
- `make seed-apply` succeeded and applied only deterministic non-business seed content.
- `make integration-test` succeeded serially against the isolated MySQL and Redis environment after schema and seed initialization.
- `make run` started `app-api` successfully against healthy MySQL and Redis.
- `GET /health/live` returned HTTP 200 with `{"status":"ok"}` and `GET /health/ready` returned HTTP 200 with `{"status":"ready"}` against healthy dependencies.
- After stopping MySQL, liveness remained HTTP 200 and readiness returned HTTP 503 with `{"status":"unready"}`.
- After restoring MySQL and stopping Redis, liveness remained HTTP 200 and readiness returned HTTP 503 with `{"status":"unready"}`.
- Safe disclosure was verified: client readiness responses stayed minimal, startup failure with Redis unavailable returned `failed to initialize app-api: redis startup connectivity check failed`, and runtime readiness logs were reduced to dependency names without raw MySQL or Redis driver details.
- Graceful shutdown was verified with `SIGINT`, and local dependency cleanup succeeded with `make deps-down`.
- Active repository files outside `docs/goals/archive/` were searched for PostgreSQL runtime remnants; none remained after replacement.

## Completion Report

Completed on Sunday, July 19, 2026.

Goal 0004 replaced PostgreSQL with MySQL as the platform's single active relational database while preserving Redis, go-zero lifecycle behavior, request/response conventions, process-only liveness, dependency-aware readiness, partial-startup cleanup, idempotent close behavior, deterministic schema/seed workflows, and the narrow transaction boundary.

The final implementation uses MySQL through `database/sql`, validated MySQL configuration, a pinned low-memory local MySQL container, deterministic MySQL schema and seed files, serial unit and integration tests, sanitized readiness and startup failure behavior, and active documentation that now states MySQL is the default relational database.
