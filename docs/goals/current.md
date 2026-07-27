# Goal 0017: Production HTTPS Hardening and Data Continuity

## Status

- State: completed
- Started: 2026-07-27
- Completed: 2026-07-27
- Blockers: None.

## Goal

Harden the completed HTTPS/WSS deployment path so the normal production entrypoint preserves the existing production data volumes, HTTP redirects remain correct on both standard and non-standard ports, runtime acceptance waits for actual readiness instead of relying on startup timing, and operators have a clear trusted-certificate path.

## References

- `AGENTS.md`
- `scripts/production-compose.sh`
- `scripts/test-production-compose.sh`
- `scripts/ci-runtime-acceptance.sh`
- `deploy/production/docker-compose.yml`
- `deploy/production/docker-compose.tls.yml`
- `deploy/production/tls/nginx.conf.template`
- `docs/operations/production-deployment.md`
- `.github/workflows/ci.yml`

## Deliverables

1. Make the unified production wrapper use the stable Compose project name `production` by default so existing `production_mysql-data` and `production_redis-data` volumes remain selected.
2. Allow an explicit isolated project through `APP_COMPOSE_PROJECT_NAME`, while preserving Docker Compose command-line `--project-name` precedence.
3. Document that changing the Compose project name changes the named-volume namespace and therefore selects different MySQL and Redis data.
4. Convert the TLS Edge Nginx configuration to a runtime template that receives the externally published `APP_HTTPS_PORT`.
5. Redirect HTTP requests to `https://<request-host>:<APP_HTTPS_PORT><request-uri>` so default `8081 -> 8443`, custom non-standard ports, and standard `80 -> 443` all resolve correctly.
6. Preserve the HTTP `/healthz` exception, HTTPS health route, Admin Web proxy, WSS Upgrade forwarding, TLS 1.2/1.3, and HSTS behavior.
7. Strengthen deterministic wrapper tests for stable project naming, environment overrides, explicit CLI project names, HTTPS selection, argument forwarding, and invalid values.
8. Replace immediate initial runtime probes with bounded readiness polling.
9. Add runtime assertions for HTTP edge health, exact non-standard-port redirect location, HTTPS health, Admin Web, and authenticated WSS.
10. Document trusted certificate requirements and clearly distinguish self-signed local verification from a browser-trusted production certificate.
11. Pass the complete repository `ci/full` gate.

## Constraints

- Work directly on `main`; do not create branches or pull requests.
- ChatGPT owns implementation, testing, fixes, verification, commits, and pushes.
- Codex must not repeat normal tests or modify source code.
- Do not delete, rename, or migrate a user's Docker volumes automatically.
- Do not terminate TLS inside the Go application.
- Do not commit certificates, private keys, passwords, access tokens, or environment files.
- Preserve existing HTTP APIs, WebSocket protocol, authentication, Admin Web behavior, MySQL 5.7 compatibility, and Kubernetes semantics.
- A real trusted certificate cannot be provisioned without an operator-controlled domain and certificate source; provide the secure configuration path without claiming that a trusted certificate was issued.

## Required Verification

```bash
bash scripts/test-production-compose.sh

APP_MYSQL_PASSWORD=test \
APP_MYSQL_ROOT_PASSWORD=test-root \
APP_REDIS_PASSWORD=test-redis \
APP_AUTH_ACCESS_TOKEN_SECRET=test-access-token-secret-0123456789abcdef \
APP_HTTPS_ENABLED=false \
bash scripts/production-compose.sh config >/dev/null

touch /tmp/awesome-zero-platform-ci.crt /tmp/awesome-zero-platform-ci.key
APP_MYSQL_PASSWORD=test \
APP_MYSQL_ROOT_PASSWORD=test-root \
APP_REDIS_PASSWORD=test-redis \
APP_AUTH_ACCESS_TOKEN_SECRET=test-access-token-secret-0123456789abcdef \
APP_HTTPS_ENABLED=true \
APP_TLS_CERT_FILE=/tmp/awesome-zero-platform-ci.crt \
APP_TLS_KEY_FILE=/tmp/awesome-zero-platform-ci.key \
APP_HTTP_PORT=18081 \
APP_HTTPS_PORT=18443 \
bash scripts/production-compose.sh config >/dev/null

make fmt-check
make test
make build

cd clients/admin-web
npm install --no-audit --no-fund
npm run build
cd ../..

git diff --check
```

GitHub Actions runtime acceptance must verify the full HTTP/WS and HTTPS/WSS flow, including the exact redirect to the configured non-standard HTTPS port.

## Acceptance Criteria

- The wrapper defaults `COMPOSE_PROJECT_NAME` to `production`.
- `APP_COMPOSE_PROJECT_NAME` overrides the default without changing tracked Compose files.
- An explicit Docker Compose `--project-name` argument remains accepted and takes precedence at execution time.
- Default production commands continue selecting the original production-namespaced MySQL and Redis volumes.
- The TLS Edge template receives `APP_HTTPS_PORT` without substituting Nginx runtime variables such as `$host` or `$request_uri`.
- HTTP `/healthz` returns `200` without redirect.
- An ordinary HTTP request redirects with `308` to the configured HTTPS port.
- HTTPS health and Admin Web requests succeed.
- Authenticated browser-mode WSS still completes handshake, hello, and ping/pong.
- Initial runtime probes use bounded retries and no longer fail solely because readiness lags liveness.
- Documentation explains project-name volume continuity and trusted certificate deployment.
- No secret or certificate is committed.
- Final `ci/full` passes.

## Working State

### Completed

- Updated `scripts/production-compose.sh` to default `COMPOSE_PROJECT_NAME` to `production` while supporting `APP_COMPOSE_PROJECT_NAME`, existing `COMPOSE_PROJECT_NAME`, and explicit Docker Compose CLI project options.
- Extended deterministic wrapper tests to cover project-name precedence, stable volume namespace selection, HTTPS switch values, Compose-file selection, argument forwarding, and invalid values.
- Replaced the static TLS Edge configuration with `deploy/production/tls/nginx.conf.template`.
- Passed `APP_HTTPS_PORT` into the unprivileged Nginx template renderer while preserving Nginx runtime variables.
- Added a writable UID/GID-scoped tmpfs for `/etc/nginx/conf.d` while keeping the TLS Edge root filesystem read-only.
- Corrected HTTP redirects to include the actual externally published HTTPS port.
- Isolated CI runtime containers and volumes under `awesome-zero-platform-ci-runtime` instead of the production project namespace.
- Changed initial liveness, readiness, Admin Web, and TLS probes to bounded polling.
- Added exact HTTP edge health, redirect location, HTTPS health, Admin Web, and WSS runtime assertions.
- Removed a `pipefail`/`grep -q` metrics SIGPIPE failure mode by reading the complete metrics response before assertion.
- Documented stable production volume naming, isolated project behavior, trusted certificate requirements, and the fact that self-signed certificates are local-test-only.

### In progress

- None.

### Remaining

- None for repository implementation and verification.
- Host-specific reconciliation of the currently running isolated Codex stack with the original `production` volumes remains a separate supplementary Codex task because ChatGPT cannot access the user's Docker host or its credentials.

### Verification status

- ChatGPT executed the deterministic Compose wrapper test successfully in its available shell environment.
- ChatGPT rendered the TLS template with `APP_HTTPS_PORT=18443`, confirmed `$host`, `$request_uri`, and WebSocket variables remained intact, and passed local `nginx -t`.
- Initial implementation run `30258685422` exposed an actual runtime-script SIGPIPE failure after metrics retrieval; ChatGPT fixed it without changing application behavior.
- Follow-up run `30259158659` exposed the unprivileged Nginx template output directory permission failure; ChatGPT fixed the tmpfs ownership and mode.
- Final implementation checkpoint: `ad79d60f3364c0428292ab1ad79583dc62674c8a`.
- Final GitHub Actions run `30259648737` reported `ci/full: success`.
- Module and generated-code checks passed.
- Go formatting, unit tests, focused race tests, and build passed.
- Production HTTPS switch and project-continuity tests passed.
- Both HTTP/WS and HTTPS/WSS Compose configurations passed validation.
- Vue type checking and production build passed.
- MySQL 5.7, Redis, schema, seed, integration, and clustered authorization tests passed.
- Runtime acceptance passed for readiness polling, HTTP edge health, exact `18081 -> 18443` redirect, HTTPS health, Admin Web, direct and proxied WS, authenticated WSS, hello/pong, metrics, API recreation, bootstrap removal, and clean teardown.
- No secret, certificate, private key, password, access token, or runtime environment file was committed.
- No Codex supplementary test was used for implementation or repository verification.

## Completion Report

Completed on 2026-07-27.

The normal production entrypoint now retains the stable `production` Compose namespace and therefore continues selecting:

```text
production_mysql-data
production_redis-data
```

HTTPS can be enabled without changing that namespace:

```bash
APP_HTTPS_ENABLED=true \
APP_TLS_CERT_FILE=/absolute/path/to/fullchain.pem \
APP_TLS_KEY_FILE=/absolute/path/to/privkey.pem \
APP_HTTP_PORT=80 \
APP_HTTPS_PORT=443 \
make production-up
```

Non-standard ports now redirect correctly, for example:

```text
http://host:8081/path -> https://host:8443/path
```

A browser-trusted certificate still requires the operator's real domain and certificate files. The repository now provides the correct secure configuration path but does not claim to issue or own those credentials.
