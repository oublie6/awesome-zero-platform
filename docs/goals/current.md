# Goal 0014: Realtime WebSocket Transport and TLS Edge

## Status

- State: in_progress
- Started: 2026-07-27
- Completed:
- Blockers:

## Goal

Add a reusable authenticated WebSocket transport to the platform for future card, board, chat, and collaborative applications, and add a production HTTPS/WSS edge baseline without coupling the transport to a specific game.

## References

- `AGENTS.md`
- `server/apps/app-api/internal/bootstrap/bootstrap.go`
- `server/apps/app-api/internal/config/config.go`
- `server/platform/authn/authn.go`
- `server/foundation/httpmiddleware/accesslog.go`
- `server/foundation/observability/metrics.go`
- `clients/admin-web/nginx.conf`
- `deploy/production/docker-compose.yml`
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

Topic names are bounded and restricted to a safe character set. The transport does not interpret game semantics.

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
- Confirmed the existing authentication service exposes access-token/session validation suitable for handshake authentication.
- Confirmed the current response-writer wrappers do not preserve `http.Hijacker`, which must be fixed before WebSocket upgrade can work.
- Selected `github.com/gorilla/websocket` v1.5.3 as the stable WebSocket implementation.
- Defined HTTPS/WSS as an edge-proxy concern while retaining internal HTTP/WS.

### In progress

- Implementing the generic realtime transport and server integration.

### Remaining

- Add middleware upgrade compatibility.
- Add realtime package, tests, configuration, and `/ws` registration.
- Add Nginx WS proxying and optional TLS edge.
- Extend runtime acceptance for WS/WSS.
- Run and fix the complete CI gate.
- Record final completion evidence.

## Completion Report

Not completed.
