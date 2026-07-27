# Awesome Zero Platform

A modular Go application platform built on go-zero, designed for reusable server foundations and multiple client applications.

## Current foundation

The platform currently provides:

- a runnable go-zero REST application with consistent HTTP errors, envelopes, request IDs, recovery, logging, CORS, security headers, and body limits;
- MySQL and Redis lifecycle, readiness, deterministic schema, and integration tests;
- internal identity accounts with UUIDv7 identifiers and Argon2id password hashing;
- pluggable authentication with HMAC JWT access tokens and rotating Redis sessions;
- pluggable authorization with Casbin/MySQL as the first adapter;
- a complete Admin backend for accounts, roles, resources, sessions, audit, one-time bootstrap, standard permissions, and expert authorization tooling;
- a Vue 3 Admin control plane with standard and backend-engineer views over the same APIs;
- an authenticated realtime WebSocket foundation with bounded queues and graceful shutdown;
- a reusable RFC 9180 HPKE secure-envelope opener for Go and matching engine-independent TypeScript sealer;
- a code-first Cocos Creator 3.8 LTS client skeleton for the Fair Doudizhu product;
- a pure Fair Doudizhu domain core for three-player private rooms, fairness phases, versioned events, and terminal hand states;
- Prometheus-compatible HTTP and process metrics;
- local and production Compose, non-root API and Admin web images, Kubernetes baselines, and GitHub Actions CI.

Fair Doudizhu persistence, transport handlers, shuffle/card/bidding/play/scoring rules, production key publication, beacon adapters, and gameplay UI remain deliberately deferred.

## Project layout

- `server/apps/` — runnable processes and transport composition
- `server/foundation/` — reusable technical infrastructure without product semantics
- `server/platform/` — reusable identity, authentication, authorization, Admin, and realtime capabilities
- `server/business/` — product-specific business modules with explicit domain boundaries
- `clients/admin-web/` — Vue 3 platform administration client
- `clients/packages/` — engine-independent reusable client packages
- `clients/fair-doudizhu-cocos/` — code-first Cocos Creator client composition root
- `deploy/` — local, container, production Compose, and Kubernetes assets
- `docs/` — architecture, API, operations, requirements, and goal documentation
- `scripts/` — project automation scripts

The project starts as a modular monolith and keeps module boundaries explicit so capabilities can be replaced or extracted when real scaling or ownership needs appear.

## Start the Admin platform

Start dependencies and API using the server instructions, then run:

```bash
cd clients/admin-web
npm install
npm run dev
```

For the first administrator, configure a random `APP_ADMIN_BOOTSTRAP_TOKEN` of at least 32 characters, open `/bootstrap`, create the account, and remove the token from the environment.

See:

- [Server usage](server/README.md)
- [Fair Doudizhu requirements](docs/requirements/fair-doudizhu-v1.md)
- [Fair Doudizhu domain architecture](docs/architecture/fair-doudizhu-domain.md)
- [Fair Doudizhu command protocol](docs/api/fair-doudizhu-protocol-v1.md)
- [Secure-envelope architecture](docs/architecture/secure-envelope-v1.md)
- [Admin architecture](docs/architecture/admin-platform.md)
- [Admin API](docs/api/admin.md)
- [Admin web operations](docs/operations/admin-web.md)
- [Security architecture](docs/architecture/security-platform.md)
- [Authentication API](docs/api/authentication.md)
- [Production deployment](docs/operations/production-deployment.md)
