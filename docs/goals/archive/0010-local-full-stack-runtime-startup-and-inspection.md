# Goal 0010: Local Full-Stack Runtime Startup and Inspection

## Status

- State: completed
- Started: 2026-07-26
- Completed: 2026-07-26
- Blockers: None.

## Goal

Start the already verified production Docker Compose stack on the current development machine, complete first-administrator bootstrap, verify login/session/Admin/Casbin behavior, recreate the API without the bootstrap token, and leave the final healthy services running for user inspection.

## References

- `AGENTS.md`
- `docs/goals/archive/0009-cached-data-models-and-clustered-authorization-consistency.md`
- `docs/operations/production-deployment.md`
- `docs/operations/admin-web.md`
- `deploy/production/docker-compose.yml`
- `scripts/ci-runtime-acceptance.sh`

## Deliverables

1. Archive completed Goal 0009 and make this Goal 0010 the active `docs/goals/current.md`.
2. Confirm `main` is synchronized with `origin/main` and the worktree starts clean.
3. Create ignored local runtime files under `.runtime/` with mode `0600`:
   - `.runtime/admin-bootstrap.env`;
   - `.runtime/admin-compose.env`;
   - `.runtime/admin-login.env`.
4. Use strong random local-only secrets and do not print or commit passwords, tokens, JWT secrets, runtime files, generated keys, build output, or Compose volumes.
5. Validate Docker, Docker Compose, relevant host ports, and production Compose interpolation.
6. Start the production Compose stack using MySQL 5.7.44, Redis, schema, `app-api`, and `admin-web`.
7. Bootstrap the first administrator exactly once and verify replay is rejected.
8. Verify login, session, `/admin/me`, roles, authorization resources, authorization engine sync, system configuration, API health, Admin Web health, and Admin Web reverse proxy behavior.
9. Recreate `app-api` using the final environment file without `APP_ADMIN_BOOTSTRAP_TOKEN`, then verify bootstrap remains unavailable and the administrator can still log in.
10. Leave the final production Compose stack running and healthy on:
    - Admin Web: `http://127.0.0.1:8080`;
    - API: `http://127.0.0.1:8888`.
11. Record truthful completion evidence, commit directly to `main`, and push to `origin/main`.

## Constraints

- Follow `AGENTS.md` completely.
- Do not create branches, pull requests, or force pushes.
- Do not discard, overwrite, or stage pre-existing user changes.
- Run memory-intensive work sequentially and set `COMPOSE_PARALLEL_LIMIT=1`.
- Do not kill unrelated user or project processes.
- Do not globally prune Docker images, volumes, networks, or build cache.
- Use the repository production Compose baseline and keep MySQL at `mysql:5.7.44`.
- Do not clear existing local database data merely to work around unknown credentials. If existing data prevents bootstrap and credentials are unavailable, record a blocker instead.
- After successful bootstrap, remove the bootstrap token from the final runtime environment and recreate `app-api`.
- Keep the final healthy stack running after completion.

## Required Verification

Run the following checks, correcting only details proven necessary by repository behavior:

```bash
git switch main
git pull --ff-only origin main
git status --short
docker version
docker compose version
docker info
docker compose --env-file .runtime/admin-bootstrap.env -f deploy/production/docker-compose.yml config >/dev/null
docker compose --env-file .runtime/admin-bootstrap.env -f deploy/production/docker-compose.yml down --volumes --remove-orphans
docker compose --env-file .runtime/admin-bootstrap.env -f deploy/production/docker-compose.yml build app-api
docker compose --env-file .runtime/admin-bootstrap.env -f deploy/production/docker-compose.yml build admin-web
docker compose --env-file .runtime/admin-bootstrap.env -f deploy/production/docker-compose.yml up -d --no-build --wait
docker compose --env-file .runtime/admin-bootstrap.env -f deploy/production/docker-compose.yml ps
docker compose --env-file .runtime/admin-bootstrap.env -f deploy/production/docker-compose.yml logs --no-color --tail=200 mysql redis schema app-api admin-web
curl --fail --silent --show-error http://127.0.0.1:8888/health/live
curl --fail --silent --show-error http://127.0.0.1:8888/health/ready
curl --fail --silent --show-error http://127.0.0.1:8080/health
curl --fail --silent --show-error http://127.0.0.1:8080/ >/dev/null
docker compose --env-file .runtime/admin-compose.env -f deploy/production/docker-compose.yml up -d --no-deps --force-recreate app-api
docker compose --env-file .runtime/admin-compose.env -f deploy/production/docker-compose.yml config >/dev/null
git status --short
git diff --check
docker compose --env-file .runtime/admin-compose.env -f deploy/production/docker-compose.yml ps
```

If code or configuration changes beyond goal documentation are required, rerun the full repository verification required by the objective before committing.

## Acceptance Criteria

- Goal 0009 is archived and Goal 0010 is active.
- The runtime credential files are present only under ignored `.runtime/`, have restrictive permissions, and are not committed.
- The production Compose configuration validates with MySQL 5.7.44.
- MySQL, Redis, `app-api`, and `admin-web` are healthy, and `schema` completed successfully.
- Backend health, Admin Web health, and the Admin Web root endpoint pass.
- Bootstrap succeeds once, replay is rejected, and later bootstrap status is unavailable.
- The final API container runs without `APP_ADMIN_BOOTSTRAP_TOKEN`.
- Administrator login still works after API recreation.
- `/auth/session`, `/admin/me`, `/admin/roles`, `/admin/authorization/resources`, `/admin/authorization/engine`, and `/admin/system/configuration` return expected protected results.
- Casbin reports healthy synchronization with matching local and database policy versions.
- The final stack remains running on this development machine.
- `docs/goals/current.md` contains truthful completion evidence without secrets.
- Goal-related changes are committed and pushed directly to `origin/main`.

## Working State

### Completed

- Goal 0009 was present as completed on synchronized `main`.
- Started Goal 0010 on 2026-07-26.
- Archived Goal 0009 to `docs/goals/archive/0009-cached-data-models-and-clustered-authorization-consistency.md`.
- Created ignored local runtime credentials under `.runtime/` with mode `0600`.
- Enabled a temporary local 2 GiB swap file at `/swapfile-codex` to avoid memory pressure during build/startup on the 1.6 GiB development machine.
- Built `production-app-api:latest` and `production-admin-web:latest` sequentially. The local build used temporary build arguments for domestic Go and npm mirrors because the default external registries timed out; no Dockerfile or runtime configuration change is committed for this.
- Removed an obsolete local MySQL 8.4.10 image before the final run.
- Pulled `mysql:5.7.44` through a domestic mirror and tagged it locally as `mysql:5.7.44`.
- Started the production Compose stack with `--no-build --wait`.
- Bootstrapped the first administrator exactly once; replay was rejected with HTTP 409.
- Verified login, `/auth/session`, `/admin/me`, roles, authorization resources, authorization engine synchronization, system configuration, Admin Web health, and Admin Web `/api/health/live` reverse proxy behavior.
- Recreated `app-api` using `.runtime/admin-compose.env` without `APP_ADMIN_BOOTSTRAP_TOKEN`.
- Verified bootstrap remained unavailable, the administrator could still log in, and Casbin local/database policy versions matched after API recreation.
- Committed and pushed the Goal 0010 runtime documentation directly to `origin/main`.
- Added follow-up UTF-8 fixes for Admin Web responses and MySQL schema/seed imports, then verified the latest repository state through the full CI gate.

### In progress

- None.

### Remaining

- None.

### Verification status

- Docker, Docker Compose, ports, ignored runtime credentials, production Compose interpolation, image builds, clean startup, health endpoints, Bootstrap, login/session/Admin/Casbin smoke checks, reverse proxying, Bootstrap-token removal, and API recreation all passed on the development machine.
- Final container status recorded on the development machine: MySQL 5.7.44 healthy, Redis healthy, `app-api` healthy, `admin-web` healthy, and `schema` exited 0.
- The latest implementation/configuration checkpoint was `4391c54019fa94568a52cc2229acade1a62ed86f`; `ci/full` passed in GitHub Actions run `30209605993`, including unit, race, build, Admin Web, MySQL 5.7 integration, clustered authorization, and production Compose runtime jobs.

## Completion Report

Completed on 2026-07-26.

The local production Compose stack was started and verified healthy for inspection at completion time:

- Admin Web: `http://127.0.0.1:8080`
- API: `http://127.0.0.1:8888`
- API liveness: `http://127.0.0.1:8888/health/live`
- API readiness: `http://127.0.0.1:8888/health/ready`

Runtime credentials are stored only in ignored local files:

- Administrator login: `.runtime/admin-login.env`
- Final Compose environment: `.runtime/admin-compose.env`

Bootstrap was executed once and replay was rejected. The final `app-api` container was recreated without `APP_ADMIN_BOOTSTRAP_TOKEN`; bootstrap remained unavailable and the administrator could still log in.

Casbin authorization synchronization was healthy. The final observed local policy version and database policy version were both `2`.

The final service state recorded on the development machine was:

- `production-mysql-1`: `mysql:5.7.44`, healthy
- `production-redis-1`: `redis:8.8.0-alpine3.23`, healthy
- `production-schema-1`: `mysql:5.7.44`, exited 0
- `production-app-api-1`: `production-app-api`, healthy, exposed on `0.0.0.0:8888`
- `production-admin-web-1`: `production-admin-web`, healthy, exposed on `0.0.0.0:8080`

No password, token, JWT secret, runtime env file, Compose volume, or build artifact was committed. Goal documentation and the follow-up UTF-8 fixes were committed and pushed directly to `origin/main`.
