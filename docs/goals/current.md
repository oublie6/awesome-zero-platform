# Goal 0009: Cached Data Models and Clustered Authorization Consistency

## Status

- State: completed
- Started: 2026-07-26
- Completed: 2026-07-26
- Blockers: None.

## Goal

Establish a go-zero Redis cache baseline for ordinary entity models, preserve explicit fresh and transactional database reads, make Casbin authorization safe and self-synchronizing across multiple `app-api` replicas, complete the carried-forward Admin runtime verification, and use a low-memory MySQL 5.7 Compose baseline suitable for the development machine.

## References

- `AGENTS.md`
- `docs/goals/archive/0008-admin-verification-and-container-runtime-acceptance.md`
- `docs/architecture/admin-platform.md`
- `docs/api/admin.md`
- `docs/api/authentication.md`
- `docs/operations/admin-web.md`
- `docs/operations/production-deployment.md`
- `deploy/local/docker-compose.yml`
- `deploy/production/docker-compose.yml`
- `deploy/kubernetes/base.yaml`
- `deploy/kubernetes/admin-web.yaml`

## Deliverables

1. A reusable go-zero cache wrapper backed by the existing Redis deployment, with configurable key prefix, positive TTL, and not-found TTL.
2. Cached primary/unique-key reads for ordinary entity models, plus explicit fresh reads and transaction/`FOR UPDATE` reads. Production account, role, and authorization-resource detail reads must use the cache baseline; password credentials, sessions, Casbin policy rows, audit logs, list/search queries, and transaction reads must not use ordinary row caching.
3. One authoritative write path per entity. Successful writes invalidate primary and old/new unique-index cache keys after database success; transaction cache invalidation occurs only after commit.
4. Continued use of the custom Casbin Adapter with MySQL as the policy source of truth and the in-process Enforcer as the authorization cache. Normal `Enforce` calls must not query Redis or MySQL.
5. A policy-version row, cross-replica serialized policy writes, Redis policy-change notifications, periodic database-version reconciliation, duplicate/old notification suppression, startup synchronization, retry behavior, readiness integration, diagnostics, and clean shutdown.
6. Cluster-safe protection for the last active `platform_super_admin`, its role membership, and the wildcard permission, including raw-policy writes.
7. Multi-instance tests proving that policy grants/revocations become visible across two engines sharing MySQL and Redis, missed notifications reconcile from the version row, and concurrent writes do not silently overwrite each other.
8. Kubernetes corrections for API/Admin Web namespace and service routing, plus instance identity injection suitable for multiple replicas.
9. MySQL Compose images changed from 8.x to `mysql:5.7.44`, with low-memory settings and schema compatibility retained.
10. Carried-forward Goal 0008 Admin, Vue, integration, Compose, and container runtime acceptance, including removal of temporary hardening artifacts.
11. Truthful completion evidence with tests, failures, fixes, CI/runtime results, commit SHA, push result, container status, URLs, and local credential-file path without secret values.

## Data cache rules

- Ordinary entity detail models are cache-capable by default.
- Cached reads follow Redis -> MySQL -> Redis using go-zero cache singleflight, not-found caching, and TTL jitter.
- `Fresh` methods always bypass Redis.
- Transaction and `FOR UPDATE` methods always bypass Redis.
- Lists, fuzzy search, dynamic pagination, audit queries, aggregates, password credentials, sessions, and Casbin policy rows bypass ordinary row caching.
- Cache keys are namespaced and deterministic.
- Writes update MySQL first and then invalidate every affected primary/unique key.
- A unique-key change invalidates the primary key, old unique key, and new unique key.
- Cache implementation details remain inside persistence adapters and are not exposed to application services.

## Clustered Casbin rules

- MySQL remains the sole durable policy source.
- The custom Casbin Adapter remains the only persistence boundary used by the authorization engine.
- Each Pod keeps a `SyncedEnforcer` in memory for normal authorization checks.
- Add `authorization_policy_state` with one `global` version row.
- Policy mutations are serialized across replicas using the database policy-state lock and are committed with a monotonically increasing version.
- After commit, the local Enforcer reloads and a Redis message publishes the new version and source instance.
- Other replicas reload only when the received/database version is newer than their local version.
- Periodic version polling repairs missed Pub/Sub messages.
- Redis notification failure does not roll back a committed MySQL policy mutation; polling provides eventual repair.
- Reload failure preserves the prior Enforcer state, records the error, and makes authorization synchronization fail readiness until recovery.
- Redis subscription reconnects and immediately checks the database version after recovery.
- Standard and expert Admin authorization writes use the same coordinator.
- Do not add ordinary go-zero row caching to `authorization_casbin_rules` and do not enable authorization-result caching without benchmark evidence.

## MySQL 5.7 baseline

- Both local and production Compose use `mysql:5.7.44`.
- Retain `utf8mb4`, UTC, health checks, and schema application.
- Use low-memory development settings such as a 64 MiB buffer pool, disabled performance schema, reduced connection limits, and disabled binary logging where appropriate.
- Keep the schema compatible with MySQL 5.7. JSON and fractional timestamps may remain; application validation remains authoritative because 5.7 does not enforce `CHECK` constraints.
- Do not weaken production application invariants merely because MySQL 5.7 ignores `CHECK` constraints.

## Required verification

Run sequentially with low concurrency:

```bash
git status --short
git pull --ff-only origin main
cd server && go mod tidy && git diff --exit-code -- go.mod go.sum && cd ..
make generate
git diff --exit-code -- server/apps/app-api
make fmt-check
make test
cd server && go test -count=1 -p 1 -parallel 1 ./foundation/cache/... ./platform/identity/... ./platform/admin/... ./platform/authn/... ./platform/authz/... && cd ..
cd server && go test -race -count=1 -p 1 -parallel 1 ./foundation/cache/... ./platform/identity/... ./platform/admin/... ./platform/authn/... ./platform/authz/... && cd ..
make build
make deps-reset
make schema-apply
make seed-apply
make integration-test
make deps-down
cd clients/admin-web && npm install --no-audit --no-fund && npm run build && cd ../..
docker compose -f deploy/local/docker-compose.yml config >/dev/null
APP_MYSQL_PASSWORD='development-only-value' \
APP_MYSQL_ROOT_PASSWORD='development-only-root-value' \
APP_REDIS_PASSWORD='development-only-value' \
APP_AUTH_ACCESS_TOKEN_SECRET='development-only-access-token-secret-32chars-minimum' \
APP_ADMIN_BOOTSTRAP_TOKEN='development-only-bootstrap-token-32chars-minimum' \
docker compose -f deploy/production/docker-compose.yml config >/dev/null
```

Add focused repeated and two-engine integration tests where useful. Preserve exact failures, make the narrowest correct fix, rerun the focused command, and then rerun the complete verification set.

## Container runtime acceptance

- Generate strong development-only secrets and first-admin credentials in an untracked mode-`0600` file.
- Start MySQL 5.7, schema, Redis, at least two API instances (or an equivalent reproducible two-engine cluster test), and Admin Web from clean container state.
- Verify liveness/readiness, Bootstrap exactly once, login/session lifecycle, account/role/resource management, standard/expert authorization consistency, audit/system diagnostics, protected-super-admin paths, and cross-instance policy visibility.
- Remove `APP_ADMIN_BOOTSTRAP_TOKEN`, recreate API containers, and verify bootstrap remains disabled while the administrator remains usable.
- Leave the final healthy production stack running for user inspection only after successful acceptance.

## Acceptance criteria

- All required checks pass and generated/module files are clean.
- Cached, fresh, and transaction read behavior is deterministic and tested.
- Cache invalidation covers primary and old/new unique-index keys.
- MySQL 5.7 local and production Compose configurations validate and real schema/integration tests pass.
- Two authorization engines sharing MySQL/Redis converge after grants and revocations.
- Missed notifications reconcile from the database version.
- Concurrent policy writes cannot silently lose a committed update.
- Authorization synchronization failure affects readiness and later recovers.
- Normal `Enforce` remains an in-memory operation.
- K8s manifests support multiple API/Admin Web replicas in the same namespace with correct service routing.
- Temporary self-modifying Admin hardening artifacts are removed.
- No password, token, secret, local env file, runtime data, build output, or unrelated change is committed.
- All implementation changes are committed and pushed directly to `origin/main` without force pushing.

## Working state

### Completed

- Implemented the reusable go-zero Redis model-cache baseline with deterministic namespaced keys, positive/not-found TTL behavior, singleflight reads, and explicit disabled-cache behavior.
- Added cached account, role, and authorization-resource detail reads while retaining fresh, transaction, `FOR UPDATE`, list/search, credential, session, audit, and Casbin-policy database paths.
- Added post-write invalidation for primary and old/new unique-index cache keys, including commit-aware transaction invalidation.
- Added clustered Casbin policy versioning, serialized writes, Redis notifications, polling reconciliation, readiness diagnostics, duplicate suppression, retry/reconnect behavior, and clean Pub/Sub shutdown.
- Protected the last active `platform_super_admin`, its membership, and wildcard permission across standard and raw-policy writes using a shared reentrant MySQL advisory lock.
- Added two-engine integration coverage for grants, revocations, missed notifications, polling recovery, concurrent writes, and convergence.
- Updated local and production Compose to MySQL 5.7.44 with low-memory settings and retained schema compatibility.
- Corrected Kubernetes namespace, service routing, Admin Web placement, and API instance identity injection.
- Completed the carried-forward Admin/Vue verification and removed temporary self-modifying hardening artifacts.
- Fixed verification-discovered defects: module checksums, formatting, stale test fakes, MySQL 5.7 native authentication, clustered test DSN construction, Redis Pub/Sub shutdown, production YAML scalar/CORS completeness, and pre-validation environment-secret injection.
- Added a four-stage `ci/full` gate covering Go/unit/race/build, Vue build, MySQL 5.7 and clustered integration, and production Compose runtime acceptance.

### In progress

- None.

### Remaining

- None in repository scope.

### Verification status

- Full `ci/full` passed on implementation commit `8b49ab1e5949ac0b423723e17fc701cff4063f13`, GitHub Actions run `30205591909`.
- Module tidy, generated-code cleanliness, gofmt, all Go unit tests, focused race tests, and the production API build passed.
- Admin Web dependency installation, type checking, production build, and clean-source verification passed.
- MySQL 5.7 and Redis startup, schema/seed application, full integration tests, and clustered authorization tests passed.
- Production Compose built and started MySQL 5.7, schema, Redis, API, and Admin Web; liveness/readiness, one-time Bootstrap, replay rejection, login/session, Admin endpoints, authorization synchronization, Bootstrap-token removal, API recreation, and post-removal login all passed.
- Runtime secrets and first-admin credentials existed only in mode-`0600` files under the ephemeral CI runner's ignored `.runtime` directory and were removed by cleanup without printing secret values.
- The successful GitHub-hosted runtime was intentionally torn down when the ephemeral runner completed; no claim is made that a stack remains running on the user's local machine.

## Completion report

Completed on 2026-07-26.

The repository now has cache-capable ordinary entity models with explicit fresh and transactional paths, cluster-safe Casbin synchronization with MySQL as the durable source of truth, last-super-admin safety across all write surfaces, MySQL 5.7 low-memory development and production Compose baselines, corrected Kubernetes deployment routing, and a verified Vue/Admin control plane.

Verification exposed and resolved concrete defects rather than weakening tests: missing dependency checksums, formatting drift, stale test interfaces, MySQL 5.7 authentication compatibility, a clustered integration DSN construction error, a blocking Redis Pub/Sub shutdown path, invalid/incomplete production YAML, and environment secrets being applied after go-zero's first validation pass.

The implementation checkpoint `8b49ab1e5949ac0b423723e17fc701cff4063f13` passed the complete four-stage `ci/full` workflow in run `30205591909`. All changes were committed directly to `main`; no feature branch, pull request, force push, credential, token, runtime file, or generated secret was used or committed. The final goal-closure and CI-cleanup commits are documentation/verification-only and are subject to the same full workflow before handoff.
