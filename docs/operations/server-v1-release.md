# Server v1 Release Baseline

## Release scope

The server v1 baseline is the completed maintainable single-process backend for the private-room Fair Doudizhu product and the reusable platform around it. It includes:

- identity accounts, login, rotating Redis sessions, logout, and account disablement;
- Casbin/MySQL authorization and the complete administrator backend;
- authenticated HTTP and WebSocket transports;
- three-player private rooms, fixed seats, ready/start flow, fairness commit/reveal, deterministic deal, bidding, play, settlement, cancellation, timeout supervision, and immutable terminal evidence;
- durable command idempotency and client-sequence ordering for persisted commands;
- bounded in-memory replay protection for live commands;
- participant-only private snapshots, account-targeted realtime delivery, reconnect synchronization, and terminal evidence recovery;
- MySQL, Redis, Prometheus metrics, Docker Compose, Kubernetes baselines, and GitHub Actions verification.

The Cocos gameplay UI is not part of this server release. Active-hand crash restoration, cross-instance game migration, public matchmaking, spectators, rankings, balances, prizes, and money-like value are also outside server v1.

## Prerequisites

- Go 1.25.8
- goctl 1.10.1
- Docker with Docker Compose
- Node.js 22 when running the Admin Web locally

Install goctl once:

```bash
go install github.com/zeromicro/go-zero/tools/goctl@v1.10.1
```

## Local server startup

Run these commands from the repository root in one shell:

```bash
make deps-reset
make schema-apply
make seed-apply

eval "$(cd server && go run ./foundation/revealkeys/cmd/local-env)"
make run
```

`local-env` generates fresh development-only values for:

- one X25519 reveal-key pair and its signed public manifest;
- one Ed25519 manifest-signing key;
- the local beacon-proof secret;
- the protected-contribution encryption key;
- all non-secret Doudizhu identifiers required by configuration validation.

The generated reveal key expires after 24 hours. Run the command again for a later local session. The command prints shell exports only; it does not write secrets into the repository. Do not use this generator for production key management.

The committed `main-api.yaml` keeps Doudizhu disabled. The generated environment values enable it only in the shell that starts `make run`.

Verify the process:

```bash
curl -sS http://127.0.0.1:8888/health/live
curl -sS http://127.0.0.1:8888/health/ready
curl -sS http://127.0.0.1:8888/metrics | head
```

Expected health payloads are `{"status":"ok"}` and `{"status":"ready"}`.

## Administrator bootstrap and three accounts

Start Admin Web in a second terminal:

```bash
cd clients/admin-web
npm install
npm run dev
```

Open `http://localhost:5173/bootstrap`. The committed local configuration contains a development-only bootstrap token. Create the first administrator, then use the Accounts screen to create three active player accounts with distinct usernames and passwords.

The same operations can be automated through:

- `GET /admin/bootstrap/status`;
- `POST /admin/bootstrap` with `X-Admin-Bootstrap-Token`;
- `POST /admin/accounts` with a super-administrator Bearer token.

See `docs/api/admin.md` for the administrator route catalog.

## Authenticate each player

Each player logs in independently:

```bash
curl -sS http://127.0.0.1:8888/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"identifier":"player-one","password":"replace-me"}'
```

Store the returned `data.accessToken` and `data.refreshToken` separately for each account. HTTP game routes use:

```http
Authorization: Bearer <access-token>
```

Native WebSocket clients may send the same header during upgrade. Browser clients use subprotocol authentication:

```ts
const socket = new WebSocket(
  'ws://127.0.0.1:8888/ws',
  ['bearer', accessToken],
)
```

Tokens are never accepted in a WebSocket URL query parameter.

## HTTP game boundary

All Doudizhu HTTP endpoints require an authenticated account:

```text
POST /v1/doudizhu/commands
GET  /v1/doudizhu/hands/:handId/public
GET  /v1/doudizhu/hands/:handId/private
GET  /v1/doudizhu/hands/:handId/evidence
```

A command request uses the transport contract:

```json
{
  "v": "doudizhu-api-command-v1",
  "requestId": "019d-local-command-0001",
  "type": "room.create",
  "aggregateId": "room-local-001",
  "clientSeq": 1,
  "expectedVersion": 0,
  "payload": {}
}
```

Supported transport command types are:

```text
room.create
room.join
room.leave
room.ready
room.hand.start
hand.commit
hand.reveal
hand.beacon
hand.dealt
hand.bid
hand.play
hand.pass
hand.cancel
```

The authenticated session is the only actor identity. Clients must not add authoritative account, owner, role, permission, or seat fields. The server resolves membership and seat from persisted room/hand state.

For a three-player integration:

1. player one sends `room.create`;
2. players two and three send `room.join` against the created room;
3. all three send `room.ready` with the current room version;
4. the owner sends `room.hand.start` with a new hand ID;
5. each client follows commit/reveal and the later bidding/play flow using the exact current aggregate version from the previous response or snapshot.

Every new durable command uses a new `requestId` and a higher `clientSeq`. Exact retries reuse the original request unchanged. A stale version is not retried blindly: reload the latest state and create a newly considered command.

The complete business command names, payloads, fairness rules, event ordering, and privacy requirements are defined in `docs/api/fair-doudizhu-protocol-v1.md`. The HTTP transport intentionally uses shorter transport type names while dispatching into that same application protocol.

## WSS commands, broadcasts, and reconnect

A WSS command wraps the same HTTP command request in the generic realtime envelope:

```json
{
  "id": "socket-command-1",
  "type": "doudizhu.command",
  "payload": {
    "v": "doudizhu-api-command-v1",
    "requestId": "019d-local-command-0002",
    "type": "room.ready",
    "aggregateId": "room-local-001",
    "clientSeq": 2,
    "expectedVersion": 3,
    "payload": {"ready": true}
  }
}
```

Responses and account-targeted events use:

```text
doudizhu.command.result
doudizhu.hand.changed
doudizhu.hand.snapshot
doudizhu.hand.evidence
doudizhu.error
```

Reconnect with the last observed live version:

```json
{
  "id": "sync-1",
  "type": "doudizhu.hand.sync",
  "payload": {
    "v": "doudizhu-realtime-sync-v1",
    "handId": "hand-local-001",
    "knownVersion": 12
  }
}
```

A current version returns `notModified=true`. A stale or omitted version returns the caller's complete private snapshot. Once the live hand has been removed, a seated participant receives immutable terminal evidence. Private cards are delivered only through account-targeted snapshots; there is no generic public game topic.

See:

- `docs/architecture/doudizhu-realtime-v1.md` for game-specific WSS behavior;
- `docs/operations/realtime-websocket.md` for authentication, envelope, limits, metrics, and transport behavior.

## Verification and shutdown

Run the repository checks serially:

```bash
make generate
make fmt-check
make test
make integration-test
make build

docker compose -f deploy/local/docker-compose.yml config >/dev/null
bash scripts/test-production-compose.sh
```

Stop local dependencies when finished:

```bash
make deps-down
```

## Production enablement

Doudizhu remains disabled by default. A production deployment that enables it must provide all of these values through an external secret/configuration mechanism:

```text
APP_REVEAL_KEYS_ENABLED=true
APP_REVEAL_KEYS_STATIC_JSON=<strict reveal-key registry JSON containing private key material>
APP_DOUDIZHU_ENABLED=true
APP_DOUDIZHU_BEACON_PROVIDER=<configured provider identifier>
APP_DOUDIZHU_BEACON_ROUND=<locked round identifier or configured plan value>
APP_DOUDIZHU_BEACON_PROOF_SECRET=<at least 32 characters>
APP_DOUDIZHU_CONTRIBUTION_KEY_ID=<key identifier>
APP_DOUDIZHU_CONTRIBUTION_KEY_HEX=<exactly 64 lowercase hex characters>
```

`APP_REVEAL_KEYS_STATIC_JSON` contains an Ed25519 signing private key and one or more X25519 private keys. Treat it as high-sensitivity private-key material. It must not be committed, printed in CI logs, placed in an image layer, or exposed through ordinary configuration diagnostics.

The production Compose baseline passes these variables to `app-api` but leaves them empty and disabled unless the operator supplies them. The ordinary production secrets listed in `docs/operations/production-deployment.md` are still required.

Public production traffic must use HTTPS and WSS. Use an environment-specific secret manager, certificate manager, backup policy, firewall/network policy, monitoring retention, and disaster-recovery procedure.

## Server v1 process boundary

The server v1 game runtime is intentionally single-process authoritative for active cards, bids, turns, passes, and live versions. MySQL stores durable room/fairness state, command results, events, and immutable terminal archives. Redis stores sessions, cache data, and authorization synchronization data; it does not store active hands.

Consequences:

- reconnect to the same running process is supported;
- terminal evidence survives normal live-hand removal;
- an application-process crash does not restore an active hand;
- multiple API replicas do not provide transparent cross-instance game routing or active-hand migration.

Do not deploy active games across multiple interchangeable API replicas until a later version defines authoritative node ownership, distributed routing, and recoverable active state. These are v2 concerns rather than incomplete v1 work.
