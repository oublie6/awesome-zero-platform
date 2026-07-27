# Goal 0014: Realtime WebSocket Transport and TLS Edge

## Status

- State: completed
- Started: 2026-07-27
- Completed: 2026-07-27
- Blockers: None.

## Goal

Add a reusable authenticated WebSocket transport to the platform for future card, board, chat, and collaborative applications, and add a production HTTPS/WSS edge baseline without coupling the transport to a specific game.

## References

- `AGENTS.md`
- `server/apps/app-api/internal/bootstrap/bootstrap.go`
- `server/apps/app-api/internal/config/config.go`
- `server/platform/authn/authn.go`
- `server/platform/realtime`
- `server/foundation/httpmiddleware/accesslog.go`
- `server/foundation/observability/metrics.go`
- `clients/admin-web/nginx.conf`
- `deploy/production/docker-compose.yml`
- `deploy/production/docker-compose.tls.yml`
- `deploy/production/tls/nginx.conf`
- `docs/operations/realtime-websocket.md`
- `docs/operations/production-deployment.md`
- `scripts/ci-runtime-acceptance.sh`

## Deliverables

1. Add `server/platform/realtime` as a reusable transport capability with no card-game business rules.
2. Expose an authenticated WebSocket endpoint on the existing `app-api` HTTP server.
3. Support native clients through `Authorization: Bearer <token>` and browser clients through `Sec-WebSocket-Protocol: bearer, <token>`; do not accept access tokens in query strings.
4. Send a versioned `system.hello` envelope after connection establishment.
5. Provide built-in ping/pong, topic subscribe, and topic unsubscribe message types.
6. Provide application handler registration, account-targeted sends, topic publication, and connection snapshots for future business modules.
7. Enforce bounded per-connection send queues, one writer goroutine, message-size limits, heartbeat deadlines, global connection limits, per-account connection limits, and graceful shutdown.
8. Disconnect slow consumers instead of allowing unbounded memory growth.
9. Add Prometheus counters/gauges for active connections, accepted/rejected connections, inbound/outbound messages, and slow-consumer disconnects.
10. Preserve WebSocket upgrade capabilities through the existing access-log and metrics response-writer wrappers.
11. Proxy `/ws` through Admin Web Nginx with the required HTTP/1.1 Upgrade headers.
12. Add an optional production TLS edge Compose override that terminates HTTPS/WSS and proxies to the existing Admin Web/API stack.
13. Keep local development compatible with HTTP/WS while documenting that public production traffic must use HTTPS/WSS.
14. Add deterministic unit, HTTP handshake, lifecycle, proxy, and runtime acceptance coverage.
15. Pass the complete repository `ci/full` gate.

## Constraints

- Work directly on `main`; do not create branches or pull requests.
- Keep `app-api` stateless and reusable in multi-Pod deployments.
- Do not implement card rules, rooms persisted in MySQL, matchmaking, or game-state storage in this goal.
- Do not use raw TCP as a public browser transport.
- Do not log access tokens, WebSocket payloads, passwords, or TLS private keys.
- Do not put access tokens in URLs or query parameters.
- TLS certificates remain deployment secrets and must not be committed.
- Application containers may continue speaking HTTP internally; TLS terminates at Nginx/Ingress.
- Preserve existing HTTP APIs, authentication semantics, authorization behavior, MySQL 5.7 compatibility, and Admin Web behavior.

## Protocol Baseline

WebSocket path:

```text
/ws
```

Client envelope:

```json
{
  "id": "optional-client-message-id",
  "type": "topic.subscribe",
  "topic": "optional-topic",
  "payload": {}
}
```

Built-in message types:

```text
system.ping
system.pong
topic.subscribe
topic.unsubscribe
system.ack
system.error
system.hello
```

Topic names and message types are canonical, bounded, and strictly validated. The transport does not interpret game semantics.

## Required Verification

```bash
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
docker compose --env-file .runtime/ci-compose-final.env -f deploy/production/docker-compose.yml config >/dev/null
```

GitHub Actions must report `ci/full: success` for the final commit, including WebSocket and TLS runtime acceptance.

## Acceptance Criteria

- Authenticated clients can connect to `/ws` through both the API port and Admin Web reverse proxy.
- Unauthenticated or invalid-token handshakes are rejected before upgrade.
- The browser-compatible subprotocol flow selects only `bearer` and never echoes the token.
- Connections receive `system.hello` and support heartbeat and topic subscription messages.
- Application code can register message handlers and publish by account or topic.
- Slow clients cannot grow unbounded queues.
- Multiple connections for one account are bounded by configuration.
- Existing middleware still records HTTP/WebSocket handshakes without breaking `Hijacker`.
- `/metrics` exposes realtime metrics when metrics are enabled.
- Nginx forwards WebSocket Upgrade and Connection headers correctly.
- An optional TLS edge can serve HTTPS and WSS using externally mounted certificates.
- Runtime acceptance verifies a login token can establish WS and WSS connections, receive hello/pong, and close cleanly.
- No TLS key, access token, query-token support, or game-specific code is committed.
- Final `ci/full` passes.

## Working State

### Completed

- Archived completed Goal 0013.
- Added `github.com/gorilla/websocket` v1.5.3 and synchronized module checksums through the Go toolchain.
- Added the reusable `server/platform/realtime` Hub, connection lifecycle, protocol contracts, Prometheus metrics, authenticated HTTP upgrade handler, and deterministic tests.
- Added native Authorization-header authentication and browser subprotocol authentication; URL query tokens are rejected.
- Reused `authn.AuthenticateAccess`, so handshakes validate the access token, Redis session, and active account status.
- Added versioned `system.hello`, application ping/pong, topic subscribe/unsubscribe acknowledgements, application handler registration, account-targeted sends, local topic publication, and connection snapshots.
- Added global/per-account limits, bounded send queues, message-size limits, heartbeat deadlines, one writer goroutine per connection, slow-consumer disconnects, and graceful shutdown.
- Made message types and topic names canonical and strict so leading/trailing whitespace cannot create split routing keys.
- Preserved `http.Hijacker`, `http.Flusher`, and response-writer unwrapping through access-log and metrics middleware; WebSocket handshakes are recorded as HTTP 101.
- Added realtime Prometheus metrics and runtime assertions for the accepted-connection counter.
- Registered `/ws` on the existing `app-api` port and added an authenticated `-realtime-healthcheck` binary mode.
- Added Admin Web Nginx `/ws` Upgrade proxying with origin-preserving Host headers and disabled proxy buffering.
- Bound the base Compose HTTP ports to loopback only.
- Added a pinned unprivileged Nginx TLS Edge Compose override with externally mounted certificates, HTTPS redirect, TLS 1.2/1.3, HSTS, and WSS proxying.
- Preserved the external host and port through both Nginx layers so browser Origin checks work on standard and nonstandard HTTPS ports.
- Extended production runtime acceptance to verify direct WS, Admin Web proxied WS, HTTPS, WSS, browser Origin, hello/pong, metrics, Bootstrap-token removal, API recreation, and post-recreation WS/WSS connections.
- Added complete realtime and production TLS operational documentation, including the Pod-local routing boundary and future distributed game-node responsibilities.
- Removed all one-shot module/format synchronization workflows after their authoritative outputs were committed.

### In progress

- None.

### Remaining

- None.

### Verification status

- Implementation checkpoint: `207db9eba2abae7ae46eb249a7e7775ca1db95db`.
- GitHub Actions run `30233228724` reported `ci/full: success`.
- Module tidy verification, generated-code verification, formatting, Go unit tests, security/Admin/realtime Race tests, and Go build passed.
- Vue dependency installation, type checking, production build, and clean source verification passed.
- MySQL 5.7 and Redis startup, schema/seed application, integration tests, and clustered Casbin tests passed.
- Production Compose runtime acceptance passed with MySQL 5.7, Redis, `app-api`, `admin-web`, and the unprivileged TLS Edge.
- Runtime authentication used a freshly bootstrapped administrator and validated direct WS, proxied WS, HTTPS, and WSS.
- Browser-style WS/WSS probes used `Sec-WebSocket-Protocol` and a real Origin header; only `bearer` was selected by the server.
- Realtime probes received `system.hello`, sent `system.ping`, received `system.pong`, and closed cleanly.
- Realtime Prometheus metrics were present after accepted connections.
- The API was recreated without `APP_ADMIN_BOOTSTRAP_TOKEN`; login and WS/WSS still worked afterward.
- No password, access token, TLS certificate, TLS private key, runtime environment file, generated synchronization workflow, or game-specific implementation remains committed.

## Completion Report

Completed on 2026-07-27.

The platform now has a reusable authenticated WebSocket transport suitable as the networking baseline for future card, board, chat, notification, and collaborative modules. It deliberately stops at the transport boundary: game rules, authoritative room state, matchmaking, cross-Pod game routing, reconnect replay, and durable event storage remain responsibilities of future business goals.

External client baseline:

```text
Local development: HTTP + WS
Public production: HTTPS + WSS
```

Endpoints:

```text
HTTP API: /api paths or direct app-api routes
WebSocket: /ws
Metrics: /metrics
```

Authentication:

```text
Native client: Authorization: Bearer <access-token>
Browser client: Sec-WebSocket-Protocol: bearer, <access-token>
```

Access tokens in query parameters are rejected. TLS terminates at Nginx/Ingress while application containers continue using internal HTTP/WS.

Operational documentation:

```text
docs/operations/realtime-websocket.md
docs/operations/production-deployment.md
```

The implementation was committed directly to `main` and passed the complete repository gate on implementation commit `207db9eba2abae7ae46eb249a7e7775ca1db95db` in GitHub Actions run `30233228724`.
