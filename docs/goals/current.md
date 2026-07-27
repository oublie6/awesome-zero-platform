# Goal 0016: Codex Local HTTPS/WSS Runtime Verification

## Status

- State: completed
- Started: 2026-07-27
- Completed: 2026-07-27
- Blockers: None.

## Goal

Use Codex CLI on the user's actual repository host to perform the one supplementary verification that ChatGPT cannot execute from its connected-repository environment: start the production Compose stack with `APP_HTTPS_ENABLED=true` against the host's real Docker daemon, verify actual host-level HTTPS and WSS behavior, and leave the verified stack running for the user.

This is a runtime-only supplementary goal. The WebSocket foundation and HTTPS switch are already implemented and passed the complete repository CI gate. Codex must not repeat normal development testing or modify source code.

## Why This Requires Codex CLI

- ChatGPT completed the implementation, deterministic tests, CI integration, and repository verification.
- GitHub Actions already passed Go, Vue, MySQL 5.7, Redis, integration, race, HTTP/WS, HTTPS/WSS, and Compose runtime acceptance.
- ChatGPT cannot access the user's Codex CLI host, local Docker daemon, occupied ports, host filesystem certificate paths, or the final running containers.
- The exact remaining gap is therefore host-specific deployment verification and leaving HTTPS/WSS running on that host.

## Baseline

- Completed implementation and goal checkpoint: `397e794c8d66b35876b267db8b24845e6391fa95`.
- Final GitHub Actions run: `30243462103` with `ci/full: success`.
- Production entrypoint: `scripts/production-compose.sh`.
- HTTPS switch: `APP_HTTPS_ENABLED=true`.
- TLS remains terminated by the `tls-edge` Nginx container; the Go `app-api` continues to use internal HTTP/WS on port `8888`.

## References

- `AGENTS.md`
- `docs/operations/production-deployment.md`
- `docs/operations/realtime-websocket.md`
- `scripts/production-compose.sh`
- `deploy/production/docker-compose.yml`
- `deploy/production/docker-compose.tls.yml`
- `deploy/production/tls/nginx.conf`
- `clients/admin-web/nginx.conf`
- `Makefile`

## Codex Scope

Codex may perform only the following supplementary work:

1. Confirm the repository is on clean, synchronized `main`.
2. Check that Docker Engine, Docker Compose, OpenSSL, curl, and Python 3 are available.
3. Start one non-destructive local production stack with HTTPS enabled.
4. Verify the actual host ports, TLS endpoint, HTTP redirect response, authenticated WSS connection, container health, and final running state.
5. Update only the status, working-state, verification-status, and completion-report sections of this file.
6. Commit and push only the permitted goal-report update directly to `main`.

Codex must not run the full Go/Vue/CI test suite again, because those checks are already complete and do not depend on the user's host.

## Runtime Preparation

Use a dedicated ignored runtime directory:

```text
.runtime/codex-https/
```

Requirements:

- Set the directory mode to `0700`.
- Do not place any certificate, private key, password, token, generated environment file, or administrator credential outside `.runtime/codex-https/`.
- Do not print secret values in terminal summaries, logs, the goal report, Git commits, or chat output.
- Reuse readable certificate paths already supplied by the user through `APP_TLS_CERT_FILE` and `APP_TLS_KEY_FILE` when both are available.
- Otherwise generate a short-lived self-signed certificate under the runtime directory. Include SAN entries for `localhost`, `tls-edge`, `127.0.0.1`, the local hostname when resolvable, and the host's primary IPv4 address when safely detectable.
- A self-signed certificate is acceptable for this host-level test; use explicit insecure verification only for that generated certificate and state clearly that browsers will not trust it automatically.
- Store the certificate and key using absolute paths. Keep the runtime directory private; make the mounted key file readable by the unprivileged TLS container without making the runtime directory broadly accessible.

Create private ignored files as needed, including:

```text
.runtime/codex-https/bootstrap.env
.runtime/codex-https/runtime.env
.runtime/codex-https/operator.env
.runtime/codex-https/tls.crt
.runtime/codex-https/tls.key
```

Generate cryptographically random values for the MySQL, MySQL root, Redis, access-token, bootstrap-token, and temporary administrator credentials. `bootstrap.env` may contain `APP_ADMIN_BOOTSTRAP_TOKEN`; `runtime.env` must omit it so the final running API does not retain bootstrap capability. `operator.env` may retain the generated local administrator username and password with mode `0600`, but they must not be printed or committed.

## Port and Existing-Service Safety

- Do not stop, replace, or reconfigure unrelated containers or host services.
- Do not delete any Docker volume.
- Do not use `down --volumes`, `docker system prune`, or similarly destructive commands.
- Before startup, check host listeners and existing Compose projects.
- The base production stack requires loopback ports `8888` and `8080`. If either is occupied by an unrelated process or an unknown deployment, stop and document a blocker instead of killing it.
- Prefer public edge ports `8081` and `8443` when available. If they are occupied, choose free unprivileged alternatives such as `18081` and `18443`, write them to the runtime environment files, and record the selected non-secret port numbers.
- Use the dedicated Compose project name:

```text
awesome-zero-platform-codex-https
```

## Required Procedure

### 1. Preflight

```bash
git status --short --branch
git pull --ff-only

docker version
docker compose version
openssl version
curl --version
python3 --version
```

Stop with a documented blocker when the working tree is not clean, `main` cannot fast-forward, a required tool is unavailable, Docker is unreachable, or required host ports cannot be used safely.

### 2. Validate the selected HTTPS configuration

The wrapper reads `APP_HTTPS_ENABLED` from its process environment, so prefix every wrapper invocation explicitly:

```bash
APP_HTTPS_ENABLED=true \
  bash scripts/production-compose.sh \
  --project-name awesome-zero-platform-codex-https \
  --env-file .runtime/codex-https/bootstrap.env \
  config --services
```

Confirm that the rendered service list includes:

```text
mysql
schema
redis
app-api
admin-web
tls-edge
```

Do not rerun `scripts/test-production-compose.sh`, `make test`, frontend builds, race tests, or the complete CI runtime script; those are outside this supplementary gap.

### 3. Start HTTPS mode

```bash
APP_HTTPS_ENABLED=true \
  bash scripts/production-compose.sh \
  --project-name awesome-zero-platform-codex-https \
  --env-file .runtime/codex-https/bootstrap.env \
  up -d --build --wait
```

Inspect the real running state:

```bash
APP_HTTPS_ENABLED=true \
  bash scripts/production-compose.sh \
  --project-name awesome-zero-platform-codex-https \
  --env-file .runtime/codex-https/bootstrap.env \
  ps
```

All required services must be running and healthy, and `tls-edge` must be present.

### 4. Verify host-level HTTP and HTTPS

Using the selected `APP_HTTP_PORT` and `APP_HTTPS_PORT`:

- Confirm `http://127.0.0.1:<http-port>/healthz` returns `200` and body `ok`.
- Confirm an ordinary HTTP path returns `308` and an HTTPS redirect location. When non-standard ports are used, record the exact `Location` header instead of silently assuming it points to the selected HTTPS port.
- Confirm `https://127.0.0.1:<https-port>/health` returns `200` and body `ok`.
- Confirm `https://127.0.0.1:<https-port>/` returns a successful Admin Web response.
- Inspect the served certificate and negotiated TLS connection with `openssl s_client` or equivalent evidence.
- Use normal certificate verification for a user-provided trusted certificate. Use `curl --insecure` only when Codex generated the self-signed runtime certificate.

### 5. Bootstrap once, authenticate, and verify WSS

- Query `/admin/bootstrap/status` through the API loopback endpoint.
- If bootstrap is available, create one temporary local administrator using the private bootstrap token and the private administrator credentials in `operator.env`.
- Log in and obtain an access token without printing it.
- Execute the repository's existing realtime healthcheck from the `app-api` container against the TLS edge:

```text
wss://tls-edge:8443/ws
```

Use browser-mode authentication and insecure TLS only for the generated self-signed certificate. The check must complete the authenticated WebSocket handshake and hello/pong flow.

After bootstrap succeeds:

1. switch subsequent Compose operations to `.runtime/codex-https/runtime.env`, which does not contain `APP_ADMIN_BOOTSTRAP_TOKEN`;
2. recreate only `app-api` as needed so the final running process no longer receives the bootstrap token;
3. confirm `/admin/bootstrap/status` reports unavailable;
4. log in again and repeat the authenticated WSS healthcheck.

Do not expose the access token, administrator password, bootstrap token, database passwords, or Redis password.

### 6. Final running-state verification

Confirm all of the following while leaving the stack running:

- `mysql`, `redis`, `app-api`, `admin-web`, and `tls-edge` are running and healthy.
- Direct host HTTPS is reachable at the recorded URL.
- Authenticated WSS succeeds through `tls-edge`.
- The final `app-api` does not receive `APP_ADMIN_BOOTSTRAP_TOKEN`.
- No tracked source, test, CI, deployment, or documentation file changed except the permitted sections of this goal file.
- Runtime files remain under ignored `.runtime/codex-https/`.

Do not run `production-down` after successful verification. Leave the dedicated Compose project running for the user.

## Failure Handling

- Do not edit production code, tests, Compose files, Nginx configuration, Makefile targets, CI, or documentation to repair a failure.
- Capture only non-secret diagnostics: the failing command, exit status, selected ports, `docker compose ps`, relevant redacted container logs, and whether the certificate was user-provided or self-signed.
- On a partial failed startup, stop only the dedicated project with `down --remove-orphans`; never remove volumes unless the user explicitly authorizes it.
- Set the goal state to `blocked` and return the evidence to ChatGPT for diagnosis and any required fix.

## Acceptance Criteria

- Codex works from clean synchronized `main` without creating a branch or pull request.
- The dedicated Compose project uses `APP_HTTPS_ENABLED=true` and includes `tls-edge`.
- Real host containers start successfully and become healthy.
- Host HTTP is reachable and returns the expected redirect response for ordinary paths.
- Host HTTPS serves the Admin Web and health endpoint using the selected certificate.
- An authenticated browser-mode WSS healthcheck passes through `tls-edge`.
- Bootstrap is disabled in the final running `app-api` process.
- Secrets and certificates remain only in ignored runtime files and are not printed or committed.
- No production or test source file is modified.
- The successful stack remains running at the end.
- The completion report records the Compose project name, selected non-secret ports, access URLs, certificate type, container health summary, WSS result, commit SHA, and push result.

## Working State

### Completed

- ChatGPT implemented and tested the WebSocket foundation and HTTPS switch.
- GitHub Actions run `30243462103` passed the complete repository `ci/full` gate.
- ChatGPT identified the host-specific Docker startup and leave-running state as the only justified Codex supplementary test gap.
- Codex confirmed clean synchronized `main` at `d421500`.
- Codex confirmed Docker Engine, Docker Compose, OpenSSL, curl, and Python 3 are available on the host.
- Codex created private ignored runtime files under `.runtime/codex-https/` with directory mode `0700`.
- Codex generated a short-lived self-signed certificate for the local supplementary verification.
- Codex started the dedicated Compose project `awesome-zero-platform-codex-https` with `APP_HTTPS_ENABLED=true`.
- Codex verified HTTP, HTTPS, TLS certificate serving, Admin Web, bootstrap/login, and authenticated browser-mode WSS through `tls-edge`.
- Codex recreated only `app-api` with `.runtime/codex-https/runtime.env`, which omits `APP_ADMIN_BOOTSTRAP_TOKEN`; the final container receives an empty bootstrap-token value from the Compose default and `/admin/bootstrap/status` reports unavailable.
- Codex left the verified HTTPS/WSS stack running.

### In progress

- None.

### Remaining

- None.

### Verification status

- `git status --short --branch` before runtime work: clean `main` synchronized with `origin/main`.
- `git pull --ff-only origin main`: fast-forwarded to `d421500`.
- Tool preflight passed: Docker Engine `29.3.1`, Docker Compose `v5.1.1`, OpenSSL `3.0.2`, curl `7.81.0`, Python `3.10.12`.
- Selected Compose project: `awesome-zero-platform-codex-https`.
- Selected host ports: base API `127.0.0.1:8888`, base Admin Web `127.0.0.1:8080`, TLS edge HTTP `8081`, TLS edge HTTPS `8443`.
- Existing port occupants were the same repository's previous `production` Compose project. Codex stopped that project with `down --remove-orphans` only; no Docker volumes or images were removed.
- `APP_HTTPS_ENABLED=true bash scripts/production-compose.sh --project-name awesome-zero-platform-codex-https --env-file .runtime/codex-https/bootstrap.env config --services` rendered `mysql`, `redis`, `schema`, `app-api`, `admin-web`, and `tls-edge`.
- Docker Hub pull for `nginxinc/nginx-unprivileged:1.29.4-alpine` timed out; Codex pulled the same tag through `m.daocloud.io` and tagged it locally without changing repository files.
- Docker build initially stalled on container-side `go mod download`; Codex configured the local `golang:1.25.8-alpine` build image to use `GOPROXY=https://goproxy.cn,direct` and `GOSUMDB=sum.golang.google.cn`, then the required Compose image build succeeded.
- `APP_HTTPS_ENABLED=true bash scripts/production-compose.sh --project-name awesome-zero-platform-codex-https --env-file .runtime/codex-https/bootstrap.env up -d --build --wait`: passed.
- Final container state: `mysql`, `redis`, `app-api`, `admin-web`, and `tls-edge` are running and healthy; `schema` completed successfully.
- HTTP edge liveness: `http://127.0.0.1:8081/healthz` returned `200` with body `ok`.
- HTTP redirect check: `http://127.0.0.1:8081/admin/` returned `308` with `Location: https://127.0.0.1/admin/`.
- HTTPS edge liveness: `https://127.0.0.1:8443/health` returned `200` with body `ok` using `curl --insecure` for the generated self-signed certificate.
- HTTPS Admin Web: `https://127.0.0.1:8443/` returned `200` with `text/html; charset=utf-8`.
- TLS certificate evidence: self-signed `CN=localhost`, SAN includes `localhost`, `tls-edge`, `127.0.0.1`, the local hostname, and the host primary IPv4 address; validity is 2026-07-27 through 2026-07-30 UTC.
- Bootstrap status before bootstrap: available.
- Bootstrap administrator creation: passed; generated credentials remained only in `.runtime/codex-https/operator.env`.
- Login before bootstrap-token removal: passed; access token was stored only in `.runtime/codex-https/`.
- Authenticated browser-mode WSS healthcheck before bootstrap-token removal: passed against `wss://tls-edge:8443/ws`, including authenticated handshake, `system.hello`, and ping/pong.
- Recreated only `app-api` with `.runtime/codex-https/runtime.env`, which does not contain `APP_ADMIN_BOOTSTRAP_TOKEN`.
- Bootstrap status after final `app-api` recreation: unavailable.
- Final `app-api` bootstrap-token environment value: present but empty due to the Compose default expression; no bootstrap token secret is present.
- Login after bootstrap-token removal: passed.
- Authenticated browser-mode WSS healthcheck after bootstrap-token removal: passed against `wss://tls-edge:8443/ws`, including authenticated handshake, `system.hello`, and ping/pong.
- `git diff --check`: passed.
- Final tracked-file diff is limited to this goal file; `.runtime/codex-https/` remains ignored.
- Final resource snapshot while the stack remained running: host memory available approximately `655MiB`, swap used `0MiB`.

## Completion Report

Completed on 2026-07-27.

Codex completed the host-specific supplementary HTTPS/WSS runtime verification on the user's Docker host and left the verified stack running.

Runtime summary:

- Compose project: `awesome-zero-platform-codex-https`.
- HTTPS enabled: `APP_HTTPS_ENABLED=true`; rendered services include `tls-edge`.
- Access URLs: `http://127.0.0.1:8081/healthz`, `https://127.0.0.1:8443/health`, and `https://127.0.0.1:8443/`.
- Certificate type: generated short-lived self-signed certificate under `.runtime/codex-https/`; browsers will not trust it automatically.
- Container health: `mysql`, `redis`, `app-api`, `admin-web`, and `tls-edge` are running and healthy; `schema` completed successfully.
- WSS result: authenticated browser-mode realtime healthcheck passed through `wss://tls-edge:8443/ws` both before and after bootstrap-token removal.
- Bootstrap final state: `/admin/bootstrap/status` reports unavailable; `.runtime/codex-https/runtime.env` omits `APP_ADMIN_BOOTSTRAP_TOKEN`.
- Secrets, tokens, generated credentials, certificate, and private key remained only under ignored `.runtime/codex-https/` and were not printed or committed.
- No production code, test code, Compose file, Nginx config, CI file, Makefile, or non-goal documentation file was modified.

Verification report commit and push result are recorded in the final Codex handoff for this goal.
