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
- Prometheus-compatible HTTP and process metrics;
- local and production Compose, non-root API and Admin web images, Kubernetes baselines, and GitHub Actions CI.

Product-specific business capabilities, the future Vue user client, WeChat Mini Program, H5, and app clients remain deliberately deferred.

## Project layout

- `server/apps/` — runnable processes and transport composition
- `server/foundation/` — reusable technical infrastructure without product semantics
- `server/platform/` — reusable identity, authentication, authorization, and Admin capabilities
- `clients/admin-web/` — Vue 3 platform administration client
- `clients/` — future user-facing clients and stable shared client packages
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
- [Admin architecture](docs/architecture/admin-platform.md)
- [Admin API](docs/api/admin.md)
- [Admin web operations](docs/operations/admin-web.md)
- [Security architecture](docs/architecture/security-platform.md)
- [Authentication API](docs/api/authentication.md)
- [Production deployment](docs/operations/production-deployment.md)
