# Awesome Zero Platform

A modular Go application platform built on go-zero, designed for reusable server foundations and multiple client applications.

## Server v1 status

The maintainable server v1 baseline is complete. It provides a deployable platform backend and a private-room Fair Doudizhu backend with authenticated HTTP/WSS commands, participant-only realtime synchronization, reconnect, lifecycle supervision, settlement, and independently verifiable terminal evidence.

The platform currently includes:

- a runnable go-zero REST application with consistent HTTP errors, envelopes, request IDs, recovery, logging, CORS, security headers, and body limits;
- MySQL and Redis lifecycle, readiness, deterministic schema, cache behavior, and real integration tests;
- UUIDv7 identity accounts with Argon2id password hashing;
- HMAC JWT access tokens with rotating Redis sessions, refresh, logout, and account-disable enforcement;
- Casbin/MySQL authorization with clustered policy synchronization;
- a complete Admin backend and Vue 3 control plane for accounts, roles, resources, sessions, audit, bootstrap, standard permissions, and expert authorization tooling;
- an authenticated realtime WebSocket foundation with bounded queues, slow-consumer handling, multiple connections per account, and graceful shutdown;
- an RFC 9180 HPKE secure-envelope implementation for Go and TypeScript, signed reveal-key manifests, and key lifecycle support;
- a Fair Doudizhu domain and application stack covering private rooms, fixed seats, ready/start, commit-reveal fairness, deterministic shuffle and deal, bidding, play, settlement, timeout/cancellation, and terminal evidence;
- transactional MySQL persistence with command-row idempotency, monotonic client sequences, optimistic aggregate versions, encrypted contribution records, events, and immutable final archives;
- authenticated Doudizhu HTTP endpoints and WSS handlers sharing one dispatcher and replay policy;
- participant-only private snapshots, account-targeted change delivery, latest-state reconnect, and terminal evidence recovery;
- Prometheus-compatible HTTP, process, and realtime metrics;
- local and production Compose, non-root API and Admin Web images, Kubernetes baselines, and GitHub Actions CI.

Doudizhu is disabled by default and must be enabled with reveal-key, beacon-proof, and contribution-encryption configuration. The Cocos gameplay UI remains a client-stage deliverable.

## Quick start

Install the required Go tool once:

```bash
go install github.com/zeromicro/go-zero/tools/goctl@v1.10.1
```

Start local dependencies, generate temporary development-only Doudizhu keys, and run the server:

```bash
make deps-reset
make schema-apply
make seed-apply

eval "$(cd server && go run ./foundation/revealkeys/cmd/local-env)"
make run
```

In another terminal, start Admin Web:

```bash
cd clients/admin-web
npm install
npm run dev
```

Open `http://localhost:5173/bootstrap`, create the first administrator, and create three player accounts for integration.

The complete startup, login, HTTP/WSS, reconnect, production enablement, and verification procedure is in [Server v1 release baseline](docs/operations/server-v1-release.md).

## Project layout

- `server/apps/` — runnable processes and transport composition
- `server/foundation/` — reusable technical infrastructure without product semantics
- `server/platform/` — reusable identity, authentication, authorization, Admin, and realtime capabilities
- `server/business/doudizhu/domain/` — pure Fair Doudizhu aggregates and state machines
- `server/business/doudizhu/application/` — commands, idempotency, sequence policy, reveal orchestration, lifecycle, and persistence ports
- `server/business/doudizhu/infrastructure/` — MySQL, secure transport, protected contribution, shuffle/game runtime, final archive, and adapters
- `clients/admin-web/` — Vue 3 platform administration client
- `clients/packages/` — engine-independent reusable client packages
- `clients/fair-doudizhu-cocos/` — code-first Cocos Creator client composition root
- `deploy/` — local, container, production Compose, and Kubernetes assets
- `docs/` — architecture, API, operations, requirements, and goal documentation
- `scripts/` — project automation scripts

The project starts as a modular monolith and keeps module boundaries explicit so capabilities can be replaced or extracted only when real scaling or ownership needs appear.

## Server v1 boundary

Active cards, bids, turns, passes, and live versions are authoritative in one running API process. MySQL stores durable setup state and immutable terminal archives; Redis does not store active hands. Process-crash restoration, cross-instance active-game migration, and distributed room ownership are intentionally deferred to a future server version.

The first release also excludes public matchmaking, tournaments, bots, spectators, rankings, balances, prizes, recharge, cash-out, and any transfer of value.

## Documentation

- [Server usage](server/README.md)
- [Server v1 release baseline](docs/operations/server-v1-release.md)
- [Production deployment](docs/operations/production-deployment.md)
- [Realtime WebSocket transport](docs/operations/realtime-websocket.md)
- [Fair Doudizhu requirements](docs/requirements/fair-doudizhu-v1.md)
- [Fair Doudizhu command protocol](docs/api/fair-doudizhu-protocol-v1.md)
- [Doudizhu realtime protocol](docs/architecture/doudizhu-realtime-v1.md)
- [Fair Doudizhu application contract](docs/api/fair-doudizhu-application-v1.md)
- [Secure-envelope architecture](docs/architecture/secure-envelope-v1.md)
- [Authentication API](docs/api/authentication.md)
- [Admin API](docs/api/admin.md)
- [Admin architecture](docs/architecture/admin-platform.md)
- [Admin Web operations](docs/operations/admin-web.md)
- [Security architecture](docs/architecture/security-platform.md)
