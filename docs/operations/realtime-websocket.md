# Realtime WebSocket Transport

The platform exposes a reusable authenticated WebSocket transport for card games, board games, chat, collaboration, notifications, and other bidirectional applications. The transport contains no game rules, matchmaking logic, durable room state, or database schema.

## Transport selection

Use ordinary HTTPS APIs for request/response operations such as login, account profiles, room discovery, history, configuration, and administration.

Use WebSocket for long-lived bidirectional activity such as:

- joining or leaving an active table;
- ready-state changes;
- turns and card actions;
- countdown and presence events;
- state snapshots and incremental events;
- chat and system notifications;
- reconnect synchronization.

Local development may use `http://` and `ws://`. Public production traffic must use `https://` and `wss://` through a TLS-terminating edge, ingress, gateway, or load balancer.

## Endpoint and authentication

The default endpoint is:

```text
/ws
```

The handshake is authenticated against the existing access-token and Redis-session mechanism. A token with a valid signature but a missing/revoked session or an inactive account is rejected.

Native clients may use an Authorization header:

```http
GET /ws HTTP/1.1
Connection: Upgrade
Upgrade: websocket
Authorization: Bearer <access-token>
```

Browser JavaScript cannot set an arbitrary Authorization header on the WebSocket constructor. Browser clients therefore use the WebSocket subprotocol list:

```ts
const socket = new WebSocket('wss://example.com/ws', ['bearer', accessToken])
```

The server selects and returns only the `bearer` subprotocol. It never echoes the token.

Access tokens are deliberately rejected in URL query parameters. Do not use either of these forms:

```text
/ws?access_token=...
/ws?token=...
```

URLs are commonly retained by browser history, proxies, access logs, tracing systems, and monitoring tools.

## Message envelope

All application messages are JSON text frames using a versioned envelope:

```json
{
  "id": "optional-client-message-id",
  "type": "topic.subscribe",
  "topic": "optional-topic",
  "payload": {},
  "sentAt": "2026-07-27T00:00:00Z"
}
```

Fields:

- `id` correlates a client request with an acknowledgement or response;
- `type` selects a built-in or application-registered handler;
- `topic` identifies the broadcast topic for outbound events;
- `payload` contains type-specific JSON;
- `sentAt` is added to server-generated messages.

## Built-in messages

After a successful connection, the server sends:

```json
{
  "type": "system.hello",
  "payload": {
    "protocolVersion": "1",
    "connectionId": "...",
    "accountId": "...",
    "displayName": "...",
    "serverTime": "..."
  }
}
```

Application-level heartbeat:

```json
{"id":"ping-1","type":"system.ping"}
```

Response:

```json
{"id":"ping-1","type":"system.pong"}
```

Topic subscription:

```json
{
  "id": "subscribe-1",
  "type": "topic.subscribe",
  "payload": {"topic":"game:table-42"}
}
```

Topic unsubscription uses `topic.unsubscribe` with the same payload. Successful subscription changes receive `system.ack`; invalid envelopes, topics, and unknown message types receive `system.error`.

Topic names are limited to 128 characters and the safe character set:

```text
A-Z a-z 0-9 : . _ -
```

## Server integration

Future platform or business modules register application message handlers during startup:

```go
err := hub.RegisterHandler("game.action", func(
    ctx context.Context,
    connection realtime.ConnectionContext,
    message realtime.Envelope,
) (*realtime.Envelope, error) {
    // Validate the command, load authoritative game state, apply it once,
    // persist or publish the resulting event, then return an acknowledgement.
    return &realtime.Envelope{Type: "game.action.accepted"}, nil
})
```

Send to every connection belonging to one account:

```go
_, err := hub.SendAccount(accountID, realtime.Envelope{
    Type:    "account.notice",
    Payload: payload,
})
```

Publish to every local connection subscribed to a topic:

```go
_, err := hub.Publish("game:table-42", realtime.Envelope{
    Type:    "game.state.changed",
    Payload: payload,
})
```

`Snapshot()` exposes bounded diagnostic counts for active connections, accounts, topics, registered handlers, and shutdown state.

## Reliability and limits

Each connection has exactly one reader goroutine and one writer goroutine. All outbound application messages enter a bounded queue. When a client cannot drain its queue, the server disconnects it as a slow consumer rather than allowing unbounded memory growth.

Configuration controls:

- global connection limit;
- per-account connection limit;
- send-queue capacity;
- maximum inbound message size;
- handshake timeout;
- write timeout;
- ping interval and pong timeout;
- allowed browser origins.

The server also uses WebSocket protocol ping/pong frames to detect dead connections. Graceful application shutdown closes active sockets and waits for writer goroutines to finish.

## Multi-instance boundary

The current Hub owns connections inside one API process. `SendAccount` and `Publish` operate on that process's local connections.

This is intentional for the generic transport baseline. A future multi-Pod game implementation must add an explicit distributed routing layer, for example:

1. assign each room/table to an authoritative game node;
2. persist or replicate authoritative state outside a browser connection;
3. route cross-Pod events through Redis Streams, NATS, Kafka, or another selected dispatcher;
4. use message IDs, commands, and state versions for idempotency and reconnect recovery.

Do not treat a WebSocket connection or one API Pod's memory as durable game state.

## Metrics

When application metrics are enabled, `/metrics` includes:

```text
awesome_zero_platform_realtime_active_connections
awesome_zero_platform_realtime_connections_accepted_total
awesome_zero_platform_realtime_connections_rejected_total
awesome_zero_platform_realtime_messages_received_total
awesome_zero_platform_realtime_messages_sent_total
awesome_zero_platform_realtime_slow_consumer_disconnects_total
```

The rejection metric uses a bounded reason label. Access tokens and message payloads are never metric labels.

## Admin Web proxy

The Admin Web Nginx configuration forwards `/ws` using HTTP/1.1 and the required Upgrade headers. A browser opened through the Admin Web origin can therefore connect to:

```ts
const scheme = location.protocol === 'https:' ? 'wss:' : 'ws:'
const socket = new WebSocket(`${scheme}//${location.host}/ws`, ['bearer', accessToken])
```

## HTTPS and WSS edge

The base production Compose file binds its internal HTTP ports to loopback only:

```text
127.0.0.1:8080 -> admin-web
127.0.0.1:8888 -> app-api
```

For a single-host TLS baseline, provide certificate paths outside Git:

```dotenv
APP_TLS_CERT_FILE=/absolute/path/to/fullchain.pem
APP_TLS_KEY_FILE=/absolute/path/to/privkey.pem
APP_HTTP_PORT=80
APP_HTTPS_PORT=443
```

The certificate and private key must be readable by the unprivileged Nginx container user. Prefer group-readable permissions with a controlled group or a deployment secret mechanism; never commit the private key.

Start the base stack plus TLS edge:

```bash
docker compose \
  --env-file .runtime/admin-compose.env \
  -f deploy/production/docker-compose.yml \
  -f deploy/production/docker-compose.tls.yml \
  up -d --build --wait
```

The TLS edge:

- redirects its HTTP listener to HTTPS;
- accepts TLS 1.2 and TLS 1.3;
- adds HSTS;
- proxies HTTPS requests to Admin Web;
- proxies WSS `/ws` upgrades through Admin Web to `app-api`.

For Kubernetes or managed production infrastructure, prefer the environment's ingress controller, gateway, certificate manager, load balancer, and secret store instead of mounting certificates into the application Deployment.

## Realtime probe

The `app-api` binary includes an authenticated probe used by CI and deployment diagnostics:

```bash
APP_REALTIME_HEALTHCHECK_TOKEN='<access-token>' \
  ./app-api \
  -realtime-healthcheck \
  -realtime-healthcheck-url 'wss://example.com/ws' \
  -realtime-healthcheck-browser
```

The probe verifies handshake authentication, `system.hello`, application-level ping/pong, and clean connection closure. The insecure-TLS flag exists only for ephemeral self-signed acceptance certificates and must not be used for public production verification.
