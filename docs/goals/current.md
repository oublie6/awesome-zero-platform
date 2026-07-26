# Goal 0008: Admin Verification and Container Runtime Acceptance

## Status

- State: ready
- Started:
- Completed:
- Blockers:

## Goal

Independently verify the completed Admin backend and Vue 3 control plane, add deterministic regression coverage for material gaps, fix only failures or safety defects that verification actually exposes, and leave the complete production Docker Compose stack running and healthy for user inspection.

## References

- `AGENTS.md`
- `docs/architecture/admin-platform.md`
- `docs/api/admin.md`
- `docs/api/authentication.md`
- `docs/operations/admin-web.md`
- `docs/operations/production-deployment.md`
- `deploy/production/docker-compose.yml`

## Deliverables

1. A coverage review of the Admin application service, HTTP API, authentication/session behavior, authorization standard view, authorization expert view, audit behavior, configuration diagnostics, and Vue 3 authentication/permission flows.
2. Deterministic tests for material uncovered behavior, prioritizing:
   - first-administrator bootstrap lifecycle and replay rejection;
   - protection of the last active `platform_super_admin` account and wildcard permission;
   - account disable and password reset session revocation;
   - role membership and standard permission replacement;
   - raw Casbin `p`/`g` policy validation, replacement, persistence, and Enforce explanation;
   - consistency between standard permission APIs and expert policy APIs;
   - frontend token refresh, concurrent unauthorized requests, logout/session clearing, cross-tab authentication synchronization, permission-derived menu/route visibility, and forbidden-route behavior.
3. The narrowest correct production or test fixes for defects that the new or existing verification exposes. No speculative refactoring is required when behavior already passes.
4. Removal of temporary self-modifying hardening artifacts before completion. Review `scripts/apply-admin-hardening.py` and `.github/workflows/admin-hardening.yml`; implement any still-valid intended behavior directly with regression coverage, then delete both files. They are not product deliverables.
5. Complete repository verification, including Go formatting, module/generated-code drift, unit tests, focused race tests, integration tests, Vue type checking/build, and production Compose validation.
6. A real container runtime acceptance run using `deploy/production/docker-compose.yml` for MySQL, schema application, Redis, `app-api`, and `admin-web`.
7. End-to-end smoke evidence against the running containers, covering health, bootstrap, login, current administrator, account/role/resource management, standard authorization, expert authorization, audit, system overview, and configuration diagnostics.
8. A completion report recording tests added, failures found, root causes, fixes, final command results, container status, exposed URLs, local credential handoff location, commit SHA, and push result.

## Constraints

- Follow `AGENTS.md` completely.
- Start from a clean synchronized `agent/admin-platform` branch and inspect the current PR diff before editing.
- Work sequentially and use low-concurrency Go commands because the development machine has limited memory.
- Do not broaden the work into business capabilities, a future user client, visual redesign, dependency upgrades, or architectural replacement.
- Tests must assert externally meaningful behavior rather than implementation details wherever practical.
- Do not weaken, skip, delete, or loosen an existing test merely to obtain a passing result.
- Production code changes are permitted only for a verified defect, missing safety invariant, runtime failure, or testability seam required by a meaningful deterministic test.
- Preserve the pluggable authorization boundary. Standard Admin APIs must not depend on Casbin types, and expert writes must pass through the authorization plugin rather than modify its database table directly.
- Do not commit passwords, bootstrap tokens, JWT secrets, database passwords, Redis passwords, generated runtime data, build output, Compose volumes, browser artifacts, or local environment files.
- Use strong development-only runtime secrets. Store temporary first-admin credentials in a local untracked file with restrictive permissions and report only its path, not the secret values, in the completion report.
- After the first administrator is created, recreate the API container without `APP_ADMIN_BOOTSTRAP_TOKEN` and confirm bootstrap is unavailable while the administrator account remains usable.
- Container startup is mandatory. A local `go run`, locally installed MySQL/Redis, or `npm run dev` does not satisfy runtime acceptance.
- After successful verification, leave the production Compose stack running for user inspection. This goal explicitly overrides the normal teardown rule for this verified stack only. On failure, tear down partial containers unless preserving them is necessary to diagnose a documented blocker.
- Codex may update the status, working-state, verification-status, and completion-report sections of this goal. It may update referenced API or operations documentation only when verification proves the documented behavior or command is inaccurate.

## Required Verification

Run the following sequentially, correcting command details only when the repository provides an authoritative equivalent:

```bash
# Repository and generated-code checks
git status --short
git pull --ff-only
cd server && go mod tidy && git diff --exit-code -- go.mod go.sum && cd ..
make generate
git diff --exit-code -- server/apps/app-api
make fmt-check

# Go verification
make test
cd server && go test -count=1 -p 1 -parallel 1 ./platform/admin/... ./platform/authn/... ./platform/authz/... && cd ..
cd server && go test -race -count=1 -p 1 -parallel 1 ./platform/admin/... ./platform/authn/... ./platform/authz/... && cd ..
make build

# Real MySQL and Redis integration verification
make deps-reset
make schema-apply
make seed-apply
make integration-test
make deps-down

# Vue Admin verification
cd clients/admin-web
npm install --no-audit --no-fund
npm run build
cd ../..
git diff --exit-code -- clients/admin-web/package.json clients/admin-web/src

# Production Compose interpolation
APP_MYSQL_PASSWORD='development-only-value' \
APP_MYSQL_ROOT_PASSWORD='development-only-root-value' \
APP_REDIS_PASSWORD='development-only-value' \
APP_AUTH_ACCESS_TOKEN_SECRET='development-only-access-token-secret-32chars-minimum' \
APP_ADMIN_BOOTSTRAP_TOKEN='development-only-bootstrap-token-32chars-minimum' \
docker compose -f deploy/production/docker-compose.yml config >/dev/null
```

If frontend regression tests are added, add a stable package script and run it before `npm run build`. If a command fails, preserve the exact failure, fix the root cause narrowly, rerun the focused failing command, and then rerun the complete verification set.

## Container Runtime Acceptance

1. Generate strong temporary development-only values for:
   - `APP_MYSQL_PASSWORD`;
   - `APP_MYSQL_ROOT_PASSWORD`;
   - `APP_REDIS_PASSWORD`;
   - `APP_AUTH_ACCESS_TOKEN_SECRET`;
   - `APP_ADMIN_BOOTSTRAP_TOKEN`;
   - first administrator username and password.
2. Store the values in an untracked local runtime file with mode `0600`. Do not add the file to the repository or print secret values into committed reports.
3. Start from an isolated clean Compose state:

```bash
docker compose -f deploy/production/docker-compose.yml down --volumes --remove-orphans
docker compose --env-file <local-runtime-env-file> -f deploy/production/docker-compose.yml up -d --build --wait
```

4. Prove the runtime state with at least:

```bash
docker compose --env-file <local-runtime-env-file> -f deploy/production/docker-compose.yml ps
docker compose --env-file <local-runtime-env-file> -f deploy/production/docker-compose.yml logs --no-color --tail=200 app-api admin-web
curl --fail --silent http://127.0.0.1:8888/health/live
curl --fail --silent http://127.0.0.1:8888/health/ready
curl --fail --silent http://127.0.0.1:8080/health
curl --fail --silent http://127.0.0.1:8080/ >/dev/null
```

5. Use HTTP calls against the containerized API to:
   - confirm bootstrap is initially available;
   - create the first administrator using `X-Admin-Bootstrap-Token`;
   - log in and obtain a real access/refresh token pair;
   - call `/admin/me`;
   - exercise representative account, role, role-member, resource, standard-permission, raw-policy, Enforce-explain, audit, overview, and configuration endpoints;
   - verify unauthorized, forbidden, invalid-policy, protected-super-admin, and revoked-session paths;
   - confirm the standard permission view and raw policy view describe the same effective authorization state.
6. Recreate `app-api` without the bootstrap token, wait for health, and verify:
   - bootstrap creation is no longer available;
   - the created administrator can still log in;
   - Admin web and API remain healthy.
7. Keep the final healthy stack running. Report:
   - Admin web: `http://localhost:8080`;
   - API: `http://localhost:8888`;
   - exact `docker compose ps` state;
   - the path of the local credential file;
   - the command the user can later run to stop and remove the stack.

## Acceptance Criteria

- Every existing CI-equivalent check and every verification command required above passes.
- New tests cover each material gap found during review and are deterministic under repeated execution.
- At least one container-backed end-to-end test or reproducible smoke script validates the Admin lifecycle against real MySQL, Redis, API, and Admin web containers.
- The five production Compose services reach their expected terminal state: MySQL, Redis, `app-api`, and `admin-web` are healthy, and the schema service completes successfully.
- The first administrator can be bootstrapped exactly once, log in, access `/admin/me`, and use protected Admin APIs.
- Removing the bootstrap token and recreating the API does not break the administrator account and prevents further bootstrap creation.
- The last active super administrator cannot be disabled or stripped of the protected role or wildcard permission.
- Disabling an account and resetting its password revoke existing sessions as documented.
- Standard permission changes are visible in expert policy output, and expert policy changes are reflected by standard authorization behavior.
- Raw policy validation and Enforce explanation work through the authorization plugin boundary.
- Admin web is served from the container on port `8080`, API is served on port `8888`, and both health endpoints pass.
- No runtime secret, credential, local environment file, Compose volume, build artifact, temporary self-modifying workflow, or unrelated change is committed.
- `docs/goals/current.md` contains truthful completion evidence, including failures encountered and fixes made.
- All goal-related changes are committed and pushed to the configured upstream without force pushing.
- The healthy production Compose stack remains running at completion for user inspection.

## Agent Strategy

The primary Codex agent owns review, test design, failure diagnosis, narrow fixes, runtime verification, final diff inspection, commit, and push. Use subagents only for independent read-only review or isolated test analysis. Do not allow multiple agents to modify the same files concurrently, and do not run memory-intensive verification tasks in parallel.

## Working State

### Completed

- ChatGPT defined the independent verification and container runtime acceptance goal.

### In progress

- None.

### Remaining

- Review current Admin coverage and temporary hardening artifacts.
- Add missing deterministic tests.
- Apply only failure-driven fixes.
- Run complete repository verification.
- Build and start the full production Compose stack.
- Execute container-backed end-to-end smoke verification.
- Remove the bootstrap token from the running API after first-admin creation.
- Leave the healthy stack running.
- Update completion evidence, commit, and push.

### Verification status

- Not started.

## Completion Report

Not started.
