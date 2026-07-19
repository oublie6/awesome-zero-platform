# Goal 0003: Establish PostgreSQL and Redis infrastructure

## Status

- State: ready
- Started:
- Completed:
- Blockers:

Supported states: `idle`, `ready`, `in_progress`, `completed`, `blocked`. New executable goals start as `ready`.

## Goal

Add the reusable PostgreSQL and Redis infrastructure required by future platform modules, including validated configuration, connection lifecycle, transaction execution, dependency-aware readiness, current schema and seed definitions, repeatable local development dependencies, integration tests, and developer documentation, without introducing product or authentication semantics.

## References

- `AGENTS.md`
- `docs/architecture/overview.md`
- `docs/requirements/server-bootstrap.md`
- `docs/requirements/http-foundation.md`
- `docs/requirements/infrastructure-foundation.md`
- `server/README.md`
- `server/apps/app-api/internal/bootstrap`
- `server/apps/app-api/internal/config`
- `server/apps/app-api/internal/svc`
- `server/apps/app-api/internal/logic/health`

## Deliverables

1. Add focused PostgreSQL infrastructure under `server/foundation` using a maintained Go driver and connection pool, with validated configuration, bounded startup connectivity checks, explicit lifecycle ownership, and safe idempotent cleanup.
2. Add focused Redis infrastructure under `server/foundation` using a maintained Go client, with validated configuration, bounded startup connectivity checks, explicit lifecycle ownership, and safe idempotent cleanup.
3. Add a narrow PostgreSQL transaction runner that propagates context, commits on success, rolls back on returned error, rolls back on panic, and preserves useful errors without introducing repository or unit-of-work frameworks.
4. Integrate PostgreSQL and Redis resources into `app-api` startup and service context while preserving the existing go-zero lifecycle and HTTP foundation.
5. Ensure partial startup failure closes any resources already created before returning an error.
6. Keep `/health/live` process-only and change `/health/ready` to report required PostgreSQL and Redis readiness using bounded checks, HTTP 200 when ready, and HTTP 503 when unavailable, without leaking internal dependency details.
7. Create executable current schema and optional seed definitions under `server/database/schema` and `server/database/seed` without introducing migration history or product tables.
8. Add a pinned local-development PostgreSQL and Redis definition with health checks, development-only credentials, and no committed runtime data.
9. Add repeatable root-level engineering commands for starting and stopping local dependencies, applying the current schema and seed definitions, resetting local data deliberately, and running infrastructure integration tests.
10. Add focused unit and integration tests for configuration, connectivity, partial cleanup, idempotent close, transactions, schema initialization, liveness, and readiness success and failure behavior.
11. Update `server/README.md` and any necessary deployment/local-development documentation with configuration, lifecycle, commands, readiness semantics, schema policy, and troubleshooting guidance.

## Constraints

- Follow `AGENTS.md` and every referenced document.
- Preserve the modular-monolith `app-api`, existing go-zero lifecycle, HTTP middleware chain, response conventions, request ID behavior, and probe-friendly health payload policy.
- Use PostgreSQL and Redis as required dependencies for this goal; startup must fail clearly and safely when required startup connectivity cannot be established.
- Use maintained, concrete PostgreSQL and Redis clients. Do not build broad adapters merely to hide their APIs.
- Do not create generic repositories, generic CRUD services, active-record models, dependency-injection frameworks, plugin systems, or generic cache abstractions.
- Do not add users, identities, roles, permissions, authentication, authorization, JWT, sessions, Casbin, files, notifications, audit business features, or business modules.
- Do not invent business tables or fake user, role, permission, or product seed data.
- Do not add MySQL, Kafka, object storage, message queues, distributed locks, rate limiting, pub/sub, outbox, sagas, or distributed transactions.
- Do not create incremental migration history. Store only the current complete schema and seed definitions. Do not commit temporary environment-upgrade SQL.
- Do not add CI, Kubernetes, production deployment manifests, cloud-specific resources, metrics, or tracing backends.
- Do not commit real credentials, secrets, runtime database files, container volumes, generated binaries, logs, coverage artifacts, or local environment files containing secrets.
- Pin explicit compatible dependency and container image versions rather than floating `latest` tags.
- Keep readiness checks bounded and safe. Do not expose DSNs, passwords, internal addresses, raw driver errors, Redis errors, or stack traces in HTTP responses.
- Do not modify archived goals.
- Do not expand or reinterpret this goal without explicit user instruction.
- Codex may update only the Status, Working State, and Completion Report sections of this file unless a deliverable explicitly requires another documentation update.

## Acceptance Criteria

1. `make generate` succeeds and remains repeatable without unintended diffs.
2. `make fmt` succeeds.
3. `make test` succeeds; equivalently, `cd server && go test ./...` passes.
4. The committed local dependency definition starts pinned PostgreSQL and Redis services and both reach their configured healthy state.
5. The current schema applies successfully to a clean PostgreSQL database and can be applied according to the documented deterministic workflow.
6. Optional seed application succeeds and contains no product, user, role, or permission data.
7. With healthy local PostgreSQL and Redis dependencies, `make run` starts `app-api` successfully.
8. With healthy dependencies, `GET /health/live` returns HTTP 200 with its existing liveness payload and `GET /health/ready` returns HTTP 200 with the documented ready payload.
9. If PostgreSQL becomes unavailable after startup, liveness remains HTTP 200 while readiness becomes HTTP 503 within the configured bounded timeout.
10. If Redis becomes unavailable after startup, liveness remains HTTP 200 while readiness becomes HTTP 503 within the configured bounded timeout.
11. Dependency failure readiness responses remain probe-friendly and do not expose credentials, DSNs, internal addresses, raw driver errors, or stack traces.
12. Invalid PostgreSQL, Redis, pool, timeout, or readiness configuration produces a clear non-zero startup failure without printing sensitive values.
13. A startup sequence in which PostgreSQL opens but Redis initialization fails closes the PostgreSQL resource and returns a safe startup error.
14. PostgreSQL and Redis close operations are safe and idempotent, including after partial initialization.
15. Transaction tests prove commit on success, rollback on returned error, rollback on panic, context propagation, and useful error preservation.
16. Integration tests prove successful PostgreSQL and Redis connectivity and deterministic isolation from pre-existing developer data.
17. No migration-history directory, committed upgrade patch, business table, generic repository framework, credential, runtime volume, or unrelated deferred capability is added.
18. `server/README.md` and local-development documentation accurately describe configuration, dependency lifecycle, schema and seed commands, readiness semantics, testing, reset behavior, and safe credential policy.
19. Final Git inspection shows only Goal 0003 implementation, tests, dependency definitions, current schema/seed content, configuration, and necessary documentation changes.
20. All verified goal-related changes are committed and pushed to the configured upstream without force pushing.

## Agent Strategy

The primary agent owns infrastructure boundaries, lifecycle integration, schema policy, readiness behavior, integration testing, final verification, commit, and push. It may use subagents for independent PostgreSQL client and transaction review, Redis client review, local dependency and schema workflow review, readiness and security review, or test review. Do not allow multiple agents to modify the same files concurrently. The primary agent must inspect and integrate every subagent result.

## Execution Process

1. Synchronize the branch according to `AGENTS.md`, read every reference completely, and inspect the existing bootstrap, configuration, service context, health logic, HTTP foundation, and generated-file boundaries.
2. Produce a concise plan before editing that identifies package boundaries, selected maintained clients, configuration shape, ownership and cleanup order, transaction semantics, readiness flow, schema layout, local dependency location, Makefile commands, and test strategy.
3. Implement configuration and focused PostgreSQL and Redis lifecycle packages first, with secret-safe validation and bounded connectivity checks.
4. Implement and test the narrow transaction runner without introducing repository or business abstractions.
5. Integrate resource ownership into `app-api`, ensuring partial startup failures clean up earlier resources and normal shutdown closes everything safely.
6. Implement dependency-aware readiness while preserving process-only liveness and the existing globally safe HTTP middleware behavior.
7. Add current schema and seed definitions plus pinned local PostgreSQL and Redis services and repeatable engineering commands.
8. Add unit and integration tests incrementally, including dependency outage and partial startup cleanup scenarios.
9. Run generation, formatting, all tests, local dependency startup, health checks, schema and seed application, real application startup, PostgreSQL outage readiness verification, Redis outage readiness verification, graceful shutdown, and local dependency cleanup.
10. Inspect responses and logs to ensure sensitive configuration and raw internal errors are not exposed.
11. Inspect the final Git diff for scope expansion, migration history, runtime artifacts, unpinned images, secrets, generic abstractions, or accidental generated-file edits.
12. Update the permitted sections below with concrete evidence, then commit and push according to `AGENTS.md`.
13. Stop only when all acceptance criteria pass and the verified commit is pushed, or a genuine blocker is documented with evidence while preserving safe local work.

## Working State

### Completed

- Goal 0002 archived.
- Infrastructure foundation requirements prepared.
- Goal 0003 execution contract prepared.

### In progress

- None.

### Remaining

- All implementation deliverables.

### Verification status

- Not started.

## Completion Report

Not started.
