# Goal 0015: WebSocket Baseline and HTTPS Switch

## Status

- State: in_progress
- Started: 2026-07-27
- Completed:
- Blockers: None.

## Goal

Keep the completed reusable WebSocket transport as the platform realtime baseline and add one explicit production HTTPS switch so operators can use the same deployment entrypoint for HTTP/WS or HTTPS/WSS without manually assembling Compose files.

## References

- `AGENTS.md`
- `server/platform/realtime`
- `deploy/production/docker-compose.yml`
- `deploy/production/docker-compose.tls.yml`
- `deploy/production/tls/nginx.conf`
- `docs/operations/realtime-websocket.md`
- `docs/operations/production-deployment.md`
- `scripts/ci-runtime-acceptance.sh`
- `.github/workflows/ci.yml`
- `Makefile`

## Deliverables

1. Preserve the existing authenticated `/ws` transport, bounded connection lifecycle, topic routing, metrics, reverse proxying, and WSS edge.
2. Add `APP_HTTPS_ENABLED` as the explicit production deployment switch, defaulting to disabled.
3. Add one production Compose wrapper that always loads the base stack and conditionally loads the TLS edge override.
4. Accept clear boolean values and fail fast for invalid switch values.
5. Keep HTTP/WS available when the switch is disabled and HTTPS/WSS available when enabled.
6. Require the existing external TLS certificate and key inputs only when HTTPS is enabled through Compose interpolation.
7. Add Makefile production targets that use the wrapper instead of requiring operators to remember Compose file combinations.
8. Add deterministic shell coverage for switch parsing, selected Compose files, argument forwarding, and invalid values.
9. Update CI Compose validation and runtime acceptance to exercise both switch states through the same wrapper.
10. Update production deployment documentation with the final operator commands and configuration variables.
11. Pass the complete repository `ci/full` gate.

## Constraints

- Work directly on `main`; do not create branches or pull requests.
- ChatGPT owns implementation, tests, fixes, verification, commits, and pushes.
- Do not delegate routine testing or implementation to Codex.
- Do not redesign or duplicate the existing realtime transport.
- Do not terminate TLS inside the Go application; TLS remains an edge responsibility.
- Do not commit certificates, private keys, passwords, access tokens, or runtime environment files.
- Preserve existing HTTP APIs, authentication behavior, Admin Web behavior, MySQL 5.7 compatibility, and Kubernetes deployment semantics.
- Keep local development compatible with HTTP/WS.

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
bash scripts/production-compose.sh config >/dev/null

cd server
go mod tidy
git diff --exit-code -- go.mod go.sum
go test -p 1 -parallel 1 ./platform/realtime/... ./foundation/httpmiddleware/... ./foundation/observability/... ./apps/app-api/internal/bootstrap/...
go test -race -count=1 -p 1 -parallel 1 ./platform/realtime/... ./foundation/httpmiddleware/... ./foundation/observability/...
cd ..

make generate
git diff --exit-code -- server/apps/app-api
make fmt-check
make test
make build

cd clients/admin-web
npm install --no-audit --no-fund
npm run build
cd ../..

docker compose -f deploy/local/docker-compose.yml config >/dev/null
git diff --check
```

GitHub Actions must report `ci/full: success` for the final commit. Its runtime job must exercise HTTP/WS with the switch disabled and HTTPS/WSS with the switch enabled.

## Acceptance Criteria

- `APP_HTTPS_ENABLED` defaults to false.
- Accepted false values select only `deploy/production/docker-compose.yml`.
- Accepted true values select both the base and TLS Compose files.
- Invalid switch values exit nonzero with a clear error.
- All remaining arguments, including `--env-file`, are forwarded to `docker compose` unchanged.
- `make production-up`, `make production-down`, and `make production-config` use the unified wrapper.
- HTTP/WS startup does not require certificate variables.
- HTTPS/WSS startup uses the existing TLS edge and requires certificate/key paths.
- CI validates both generated Compose configurations.
- Runtime acceptance verifies direct WS, proxied WS, HTTPS, WSS, hello/pong, metrics, API recreation, and clean shutdown through the unified deployment entrypoint.
- Documentation clearly distinguishes local HTTP/WS, production switch-off behavior, and public HTTPS/WSS behavior.
- Final `ci/full` passes.

## Working State

### Completed

- Confirmed Goal 0014 already provides the reusable authenticated WebSocket transport and TLS edge.
- Identified that HTTPS currently requires operators to manually append `deploy/production/docker-compose.tls.yml`, so there is no explicit unified switch yet.
- Defined `APP_HTTPS_ENABLED` and a wrapper-based deployment interface as the narrow completion scope.

### In progress

- Implementing the production HTTPS switch, wrapper, tests, CI integration, and documentation.

### Remaining

- Add the production Compose wrapper and deterministic switch tests.
- Add Makefile targets.
- Route CI and runtime acceptance through the wrapper.
- Update production documentation.
- Run and fix the complete verification gate.
- Record final evidence and completion status.

### Verification status

- Not started for Goal 0015.

## Completion Report

Not completed.
