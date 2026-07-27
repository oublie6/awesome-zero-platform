# Goal 0015: Independent Realtime Transport Verification

## Status

- State: ready
- Started:
- Completed:
- Blockers:

## Goal

Independently verify the completed realtime WebSocket and HTTPS/WSS edge implementation from a clean synchronized `main`, strengthen execution coverage where useful, and make only the narrowest fixes for failures that verification actually exposes.

## References

- `AGENTS.md`
- `server/platform/realtime`
- `server/apps/app-api/internal/bootstrap/bootstrap.go`
- `server/apps/app-api/internal/config/config.go`
- `server/foundation/httpmiddleware/accesslog.go`
- `server/foundation/observability/metrics.go`
- `clients/admin-web/nginx.conf`
- `deploy/production/docker-compose.yml`
- `deploy/production/docker-compose.tls.yml`
- `deploy/production/tls/nginx.conf`
- `docs/operations/realtime-websocket.md`
- `docs/operations/production-deployment.md`
- `scripts/ci-runtime-acceptance.sh`

## Baseline

- Implementation checkpoint: `207db9eba2abae7ae46eb249a7e7775ca1db95db`.
- Completed goal checkpoint: `40b6a6bea332fa58cbbc318ef10ffdaf35603e8f`.
- GitHub Actions run `30233535644` reported `ci/full: success` for the completed goal checkpoint.

## Deliverables

1. Start from a clean `main` synchronized with `origin/main` using `git pull --ff-only`.
2. Rerun every required verification command listed below sequentially under the repository memory constraints.
3. Add stronger verification variants only when they provide concrete independent evidence, such as shuffled tests, repeated race runs, bounded stress counts, or low-resource builds.
4. Review authentication extraction, browser subprotocol selection, origin validation, connection registration/unregistration, bounded queues, heartbeat shutdown, topic membership, graceful close, middleware `Hijacker` preservation, Nginx Upgrade forwarding, and TLS edge host preservation.
5. If every check passes, do not manufacture code changes or speculative refactors; record the verification evidence only.
6. If a check fails, identify the root cause, make the narrowest correct fix, add deterministic regression coverage, rerun the narrow failing command, and then rerun the complete verification set.
7. If production behavior, public contracts, configuration, architecture, or significant test semantics change, clearly flag the change for ChatGPT review before the next feature goal.
8. Commit and push any verification report or failure-driven fix directly to `main`; do not create a branch or pull request.

## Constraints

- Work directly on `main`; do not create branches or pull requests.
- Do not broaden the realtime protocol or add card-game, room, matchmaking, persistence, reconnect replay, or distributed-routing behavior.
- Do not redesign architecture when tests pass.
- Do not weaken, skip, delete, or make assertions less strict to obtain a passing result.
- Do not add access-token query parameters or log tokens, WebSocket payloads, passwords, certificates, or private keys.
- Preserve existing HTTP APIs, authentication and authorization behavior, MySQL 5.7 compatibility, Admin Web behavior, and production deployment semantics.
- Run memory-intensive commands sequentially and prefer `-p 1 -parallel 1` for Go tests.
- Codex may update the status, working-state, verification-status, and completion-report sections of this goal. Source and test changes are permitted only to fix failures actually exposed by verification.

## Required Verification

```bash
git status --short --branch
git pull --ff-only

cd server
go mod tidy
git diff --exit-code -- go.mod go.sum
go test -p 1 -parallel 1 ./platform/realtime/... ./foundation/httpmiddleware/... ./foundation/observability/... ./apps/app-api/internal/bootstrap/...
go test -shuffle=on -count=5 -p 1 -parallel 1 ./platform/realtime/...
go test -race -count=3 -p 1 -parallel 1 ./platform/realtime/... ./foundation/httpmiddleware/... ./foundation/observability/...
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
docker compose --env-file .runtime/ci-compose-final.env -f deploy/production/docker-compose.yml config >/dev/null

git diff --check
git status --short
```

When the local environment supports the repository runtime gate, also run the same MySQL 5.7, Redis, HTTP/WS, HTTPS/WSS, browser-Origin, hello/pong, metrics, API-recreation, and graceful-close acceptance flow used by `ci/full`.

If a commit is produced, GitHub Actions must report `ci/full: success` for that final commit.

## Acceptance Criteria

- All required verification commands pass from a clean synchronized `main`, or a genuine environment blocker is documented with evidence.
- Shuffled repeated realtime tests pass.
- Repeated focused race tests pass without data races, goroutine lifecycle failures, or flaky timeouts.
- Native Authorization-header authentication and browser `Sec-WebSocket-Protocol` authentication remain valid.
- The server selects only the `bearer` subprotocol and never echoes the access token.
- URL query tokens remain rejected.
- Origin checks preserve the external TLS host and port through both Nginx layers.
- Connection and per-account limits remain race-safe and deterministic.
- Slow consumers are disconnected without unbounded queue growth.
- Topic subscribe/unsubscribe and unregister cleanup remain consistent.
- Middleware preserves WebSocket upgrade interfaces and records handshakes without breaking HTTP behavior.
- Runtime acceptance, when available, validates direct WS, proxied WS, HTTPS, WSS, hello/pong, metrics, recreation, and clean shutdown.
- No speculative production changes are made when verification passes.
- Every actual failure, root cause, fix, regression test, final command result, commit SHA, push result, and unavailable integration is recorded.

## Working State

### Completed

- Completed Goal 0014 implemented the reusable authenticated WebSocket transport and HTTPS/WSS edge.
- Confirmed the completed goal checkpoint `40b6a6bea332fa58cbbc318ef10ffdaf35603e8f` currently reports `ci/full: success`.
- Performed a primary-agent source review of the protocol contracts, Hub routing, bounded client lifecycle, authentication wiring, and completed-goal evidence.
- Prepared this verification-only goal for Codex independent execution.

### In progress

- None.

### Remaining

- Codex must synchronize a clean `main` and set the goal state to `in_progress`.
- Run the required and stronger verification commands.
- Fix only failures actually exposed by verification.
- Record final evidence and set the goal to `completed`, or document a genuine blocker.

### Verification status

- Not started by Codex.

## Completion Report

Not completed.
