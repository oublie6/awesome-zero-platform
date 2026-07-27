# Goal 0018: Codex Production Volume HTTPS Reconciliation

## Status

- State: completed
- Started: 2026-07-27
- Completed: 2026-07-27
- Blockers: None.

## Goal

Use Codex CLI on the user's actual Docker host to reconcile the currently running isolated HTTPS test stack with the stable `production` Compose project, select the original `production_mysql-data` and `production_redis-data` volumes without deleting or rewriting them, verify the corrected HTTP redirect and authenticated WSS behavior, and leave the verified production stack running.

This is a host-specific supplementary task. ChatGPT already completed the implementation, tests, failure fixes, and repository verification. Codex must not modify production code, tests, Compose files, Nginx configuration, CI, Makefiles, or non-goal documentation.

## Why This Requires Codex CLI

- ChatGPT cannot access the user's local Docker daemon, Compose projects, named volumes, runtime environment files, host ports, or certificate files.
- The repository implementation passed the full CI gate, including the exact non-standard-port redirect and HTTPS/WSS runtime acceptance.
- The remaining work is operational reconciliation of the user's actual host state.

## Baseline

- Goal 0017 completion checkpoint: `566f9d57502c776f90de31bcca001400428a2fb7`.
- Goal 0017 final GitHub Actions run: `30260057847` with `ci/full: success`.
- Stable production project name: `production`.
- Stable production volumes:

```text
production_mysql-data
production_redis-data
```

- Current isolated test project reported by Goal 0016:

```text
awesome-zero-platform-codex-https
```

- The isolated project and its volumes must not be deleted.
- The Go `app-api` remains internal HTTP/WS; TLS is terminated by `tls-edge`.

## References

- `AGENTS.md`
- `docs/operations/production-deployment.md`
- `scripts/production-compose.sh`
- `deploy/production/docker-compose.yml`
- `deploy/production/docker-compose.tls.yml`
- `deploy/production/tls/nginx.conf.template`
- `docs/operations/realtime-websocket.md`

## Codex Scope

Codex may perform only the following work:

1. Confirm clean synchronized `main`.
2. Inspect Docker, Compose projects, containers, ports, and the production/isolated named volumes.
3. Locate an existing private production environment source or operator-provided production credentials without printing them.
4. Validate that the production credentials can start and authenticate against the existing `production` volumes.
5. Stop the isolated project without removing volumes only when production startup prerequisites are confirmed.
6. Start the stable `production` project with `APP_HTTPS_ENABLED=true`.
7. Verify exact volume mounts, health, HTTP redirect, HTTPS, authentication, and WSS.
8. Leave the verified `production` stack running.
9. Update only the status, Working State, Verification Status, and Completion Report sections of this file.
10. Commit and push only the permitted goal-report update to `main`.

Codex must not rerun the full Go, Vue, race, integration, or CI test suite.

## Safety Rules

- Do not delete, rename, copy, recreate, or migrate any Docker volume.
- Never run `down --volumes`, `docker volume rm`, `docker system prune`, or an equivalent destructive command.
- Do not reset MySQL users or passwords.
- Do not use `--skip-grant-tables`, directly edit MySQL grant tables, or bypass authentication.
- Do not modify source files to accommodate host state.
- Do not expose passwords, tokens, certificate private keys, access tokens, or administrator credentials in output, logs, Git commits, or the goal report.
- Do not stop unrelated containers or services.
- Do not assume that similarly named volumes contain the same data.
- Do not claim data continuity unless the final containers are proven to mount `production_mysql-data` and `production_redis-data`.
- Do not remove the isolated project's volumes after a successful cutover; retain them for rollback until the user explicitly authorizes cleanup.

## Preflight

Run from the repository root:

```bash
git status --short --branch
git pull --ff-only

docker version
docker compose version
docker compose ls
docker volume ls
```

Inspect relevant objects without printing secret environment values:

```bash
docker ps -a --filter label=com.docker.compose.project=production
docker ps -a --filter label=com.docker.compose.project=awesome-zero-platform-codex-https
docker volume inspect production_mysql-data production_redis-data
```

Confirm the following before changing running services:

- `production_mysql-data` and `production_redis-data` exist.
- The current isolated project and its volume names are recorded.
- Ports `8888`, `8080`, the selected HTTP edge port, and the selected HTTPS edge port are owned only by the expected isolated project.
- A usable production environment source is available.
- An authentication path is available for the production database:
  - existing administrator credentials stored privately; or
  - bootstrap is available and a valid production bootstrap token is privately available.

If production credentials or an authentication path are unavailable, leave the isolated stack running, set the goal to `blocked`, and report exactly which non-secret input is missing. Do not attempt password recovery or reset.

## Private Runtime State

Use an ignored private directory:

```text
.runtime/codex-production-https/
```

Set directory mode `0700` and private files to `0600`.

The final production environment must use the existing production MySQL and Redis credentials. Do not substitute the random credentials created for the isolated Goal 0016 stack.

Certificate selection order:

1. Use readable operator-provided trusted certificate paths from `APP_TLS_CERT_FILE` and `APP_TLS_KEY_FILE` when available.
2. Otherwise reuse or generate a private self-signed certificate under `.runtime/codex-production-https/` solely to keep the host HTTPS testable.

When a self-signed certificate is used, report clearly that browser trust remains deferred and record its non-secret expiry date. Do not claim that a public trusted certificate was installed.

## Production Configuration Validation

Validate the stable project before stopping the isolated stack:

```bash
APP_COMPOSE_PROJECT_NAME=production \
APP_HTTPS_ENABLED=true \
  bash scripts/production-compose.sh \
  --env-file /absolute/private/path/to/production-runtime.env \
  config --services
```

The service list must contain:

```text
mysql
schema
redis
app-api
admin-web
tls-edge
```

Confirm the rendered volume names resolve to the `production` namespace. Do not continue if the configuration points at a new or unexpected project name.

## Cutover Procedure

### 1. Preserve rollback information

Record only non-secret state:

- isolated Compose project name;
- isolated environment-file path;
- selected ports;
- isolated certificate type and paths without displaying file contents;
- isolated container health;
- isolated volume names.

### 2. Stop only the isolated containers

After all production prerequisites pass:

```bash
APP_COMPOSE_PROJECT_NAME=awesome-zero-platform-codex-https \
APP_HTTPS_ENABLED=true \
  bash scripts/production-compose.sh \
  --env-file .runtime/codex-https/runtime.env \
  down --remove-orphans
```

Do not add `--volumes`.

### 3. Start the stable production project with HTTPS

```bash
APP_COMPOSE_PROJECT_NAME=production \
APP_HTTPS_ENABLED=true \
  bash scripts/production-compose.sh \
  --env-file /absolute/private/path/to/production-runtime.env \
  up -d --build --wait
```

Use the selected certificate paths and ports from the private production environment.

### 4. Prove volume continuity

Inspect the final mounts and require exact matches:

```text
production_mysql-data -> /var/lib/mysql
production_redis-data -> /data
```

Use `docker inspect` or `docker compose ps` plus container inspection. Record only volume names and mount destinations.

### 5. Verify HTTP and HTTPS

Using the actual selected ports:

- `GET http://127.0.0.1:<http-port>/healthz` returns `200` and `ok`.
- An ordinary HTTP path returns `308` to `https://127.0.0.1:<https-port>/<same-path>`.
- `GET https://127.0.0.1:<https-port>/health` returns `200` and `ok`.
- `GET https://127.0.0.1:<https-port>/` returns the Admin Web.
- Inspect the served certificate subject, SAN, issuer, and expiry without exposing its private key.
- Use normal certificate verification for a trusted certificate; use explicit insecure verification only for a self-signed certificate.

### 6. Verify authentication and WSS

Use the private production authentication path identified during preflight:

- If bootstrap is available and a valid production bootstrap token exists, create the administrator once and then recreate `app-api` using an environment that omits `APP_ADMIN_BOOTSTRAP_TOKEN`.
- Otherwise log in using existing private administrator credentials.
- Obtain an access token without printing it.
- Run the repository realtime healthcheck in browser-authentication mode against:

```text
wss://tls-edge:8443/ws
```

The authenticated handshake, `system.hello`, and ping/pong must pass.

Confirm the final API process has no non-empty bootstrap token and `/admin/bootstrap/status` is unavailable after bootstrap has been completed.

## Rollback

If production startup or verification fails after the isolated stack was stopped:

1. collect redacted production diagnostics;
2. stop only the partial `production` project with `down --remove-orphans` and no volume removal;
3. restart the isolated project using its existing private runtime environment;
4. verify the isolated HTTPS health endpoint is restored;
5. mark the goal `blocked` and return the evidence to ChatGPT.

Do not leave both projects stopped.

## Acceptance Criteria

- Codex works from clean synchronized `main` without branches or pull requests.
- No tracked file changes except the permitted goal-report sections.
- Original production credentials are used; isolated random database credentials are not reused for the production volumes.
- Final MySQL and Redis containers mount exactly `production_mysql-data` and `production_redis-data`.
- Isolated volumes remain present and untouched.
- Final Compose project is `production` and remains running.
- `tls-edge` is running and healthy.
- HTTP health returns `200`.
- HTTP redirect includes the actual configured HTTPS port.
- HTTPS health and Admin Web return `200`.
- Authenticated browser-mode WSS passes handshake, hello, and ping/pong.
- Final bootstrap-token state is safe.
- Certificate trust status is reported honestly as trusted or self-signed/deferred.
- No secrets or certificate private keys are printed or committed.
- On failure, the isolated service is restored and the goal is marked blocked.

## Working State

### Completed

- ChatGPT completed Goal 0017 implementation and repository verification.
- Stable production project naming, port-aware redirects, readiness polling, and trusted-certificate documentation are in `main`.
- Goal 0017 final run `30260057847` reported `ci/full: success`.
- ChatGPT identified actual host volume reconciliation as the only remaining Codex-specific task.
- Codex confirmed clean synchronized `main` at `12d662e`.
- Codex confirmed Docker Engine `29.3.1` and Docker Compose `v5.1.1` are available.
- Codex confirmed original production volumes `production_mysql-data` and `production_redis-data` exist.
- Codex recorded the running isolated project `awesome-zero-platform-codex-https`, its ports, health, and retained volumes before cutover.
- Codex found private production environment files under `.runtime/` and verified the production database, Redis, and administrator credential path without printing secrets.
- Codex created private production HTTPS runtime state under ignored `.runtime/codex-production-https/`.
- Codex validated the stable `production` HTTPS Compose configuration before stopping the isolated stack.
- Codex stopped only the isolated project containers with `down --remove-orphans`; no volumes were removed.
- Codex started the stable `production` project with `APP_HTTPS_ENABLED=true`, verified production volume mounts, HTTP, HTTPS, TLS, authentication, and browser-mode WSS, and left `production` running.

### In progress

- None.

### Remaining

- None.

### Verification status

- Preflight `git status --short --branch`: clean `main` synchronized with `origin/main`.
- `git pull --ff-only origin main`: fast-forwarded to `12d662e`.
- Docker/Compose preflight passed: Docker Engine `29.3.1`, Docker Compose `v5.1.1`.
- Existing Docker state before cutover: isolated project `awesome-zero-platform-codex-https` was running with healthy `mysql`, `redis`, `app-api`, `admin-web`, and `tls-edge`; stable `production` project had no running containers.
- Existing production volumes: `production_mysql-data` and `production_redis-data` exist with Compose project labels for `production`.
- Isolated rollback volumes retained before and after cutover: `awesome-zero-platform-codex-https_mysql-data` and `awesome-zero-platform-codex-https_redis-data`.
- Port ownership before cutover: `8888`, `8080`, `8081`, and `8443` were owned by the expected isolated project.
- Production credentials were sourced from private `.runtime/` files; core production secrets differ from the Goal 0016 isolated runtime secrets.
- Production MySQL credential preflight against `production_mysql-data`: passed using a temporary no-port container; the container was removed and the volume was not removed.
- Production Redis credential preflight against `production_redis-data`: passed using a temporary no-port container; the container was removed and the volume was not removed.
- Private production runtime file: `.runtime/codex-production-https/production-runtime.env` with mode `0600`.
- Certificate type: generated self-signed local certificate under `.runtime/codex-production-https/`; browser trust remains deferred.
- Production Compose validation: `APP_COMPOSE_PROJECT_NAME=production APP_HTTPS_ENABLED=true bash scripts/production-compose.sh --env-file .runtime/codex-production-https/production-runtime.env config --services` rendered `mysql`, `redis`, `schema`, `app-api`, `admin-web`, and `tls-edge`.
- Rendered Compose project name: `production`; rendered named volumes are `mysql-data` and `redis-data`, which resolve to `production_mysql-data` and `production_redis-data` for the stable project.
- Isolated project stop command: `APP_COMPOSE_PROJECT_NAME=awesome-zero-platform-codex-https APP_HTTPS_ENABLED=true bash scripts/production-compose.sh --env-file .runtime/codex-https/runtime.env down --remove-orphans`; no `--volumes` option was used.
- Production startup: `APP_COMPOSE_PROJECT_NAME=production APP_HTTPS_ENABLED=true bash scripts/production-compose.sh --env-file .runtime/codex-production-https/production-runtime.env up -d --build --wait` passed.
- Final mount proof: `production_mysql-data -> /var/lib/mysql`; `production_redis-data -> /data`.
- Final container state: `mysql`, `redis`, `app-api`, `admin-web`, and `tls-edge` are running and healthy; `schema` completed successfully.
- HTTP liveness: `http://127.0.0.1:8081/healthz` returned `200` with body `ok`.
- HTTP redirect: `http://127.0.0.1:8081/admin/` returned `308` with `Location: https://127.0.0.1:8443/admin/`.
- HTTPS liveness: `https://127.0.0.1:8443/health` returned `200` with body `ok` using explicit insecure verification for the generated self-signed certificate.
- HTTPS Admin Web: `https://127.0.0.1:8443/` returned `200` with `text/html; charset=utf-8`.
- Served certificate evidence: self-signed `CN=localhost`, issuer `CN=localhost`, SAN includes `localhost`, `tls-edge`, `127.0.0.1`, the local hostname, and the host primary IPv4 address; validity is 2026-07-27 through 2026-08-03 UTC.
- Production administrator login using private credentials: passed; access token was stored only under ignored `.runtime/codex-production-https/`.
- Browser-mode realtime WSS healthcheck: passed against `wss://tls-edge:8443/ws`, including authenticated handshake, `system.hello`, and ping/pong.
- Final bootstrap-token state: app-api environment contains no non-empty bootstrap token; `/admin/bootstrap/status` reports unavailable.
- Final Compose project: `production` remains running.
- Final resource snapshot while production remained running: host memory available approximately `629MiB`, swap used `0MiB`.
- Final tracked-file diff is limited to this goal file; private runtime files remain ignored.
- `git diff --check`: passed.

## Completion Report

Completed on 2026-07-27.

Codex reconciled the host from the isolated HTTPS test project to the stable `production` Compose project and left `production` running.

Runtime summary:

- Final Compose project: `production`.
- Final volume mounts: `production_mysql-data -> /var/lib/mysql`; `production_redis-data -> /data`.
- Non-sensitive ports: API `127.0.0.1:8888`, Admin Web `127.0.0.1:8080`, HTTP edge `8081`, HTTPS edge `8443`.
- HTTP result: `/healthz` returned `200 ok`; ordinary HTTP `/admin/` returned `308` with `Location: https://127.0.0.1:8443/admin/`.
- HTTPS result: `/health` returned `200 ok`; `/` returned Admin Web HTML.
- WSS result: browser-mode authenticated realtime healthcheck passed through `wss://tls-edge:8443/ws`, including handshake, `system.hello`, and ping/pong.
- Certificate status: self-signed local certificate, expires 2026-08-03 UTC; browser trust remains deferred.
- Container health: `mysql`, `redis`, `app-api`, `admin-web`, and `tls-edge` are healthy; `schema` completed successfully.
- Bootstrap state: final `app-api` has no non-empty `APP_ADMIN_BOOTSTRAP_TOKEN`; `/admin/bootstrap/status` is unavailable.
- Isolated rollback volumes remain present: `awesome-zero-platform-codex-https_mysql-data` and `awesome-zero-platform-codex-https_redis-data`.
- Secrets, private keys, administrator credentials, and access tokens remained only under ignored `.runtime/` paths and were not printed or committed.
- Goal report commit: `7cff1c6692969701d1f37e7166ba81cd95015a42`; pushed to `origin/main`.
