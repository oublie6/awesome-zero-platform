# Awesome Zero Platform

A modular Go application platform built on go-zero, designed for reusable server foundations and multiple client applications.

## Current foundation

The server currently provides:

- a runnable go-zero REST application;
- consistent HTTP errors, envelopes, request IDs, recovery, logging, CORS, security headers, and body limits;
- MySQL and Redis lifecycle, readiness, deterministic schema, and integration tests;
- internal identity accounts with UUIDv7 identifiers and Argon2id password hashing;
- pluggable authentication contracts with HMAC JWT access tokens and rotating Redis sessions;
- pluggable authorization contracts with Casbin and MySQL as the first adapter;
- Prometheus-compatible HTTP and process metrics;
- local and production Compose, a non-root application image, Kubernetes baseline manifests, and GitHub Actions CI.

Product-specific business capabilities and client applications remain deliberately deferred.

## Project layout

- `server/apps/` — runnable processes and transport composition
- `server/foundation/` — reusable technical infrastructure without product semantics
- `server/platform/` — reusable identity, authentication, and authorization capabilities
- `clients/` — planned Vue 3, WeChat Mini Program, H5, and app clients
- `deploy/` — local, container, production Compose, and Kubernetes assets
- `docs/` — architecture, API, operations, requirements, and goal documentation
- `scripts/` — project automation scripts

The project starts as a modular monolith and keeps module boundaries explicit so capabilities can be replaced or extracted when real scaling or ownership needs appear.

See:

- [Server usage](server/README.md)
- [Security architecture](docs/architecture/security-platform.md)
- [Authentication API](docs/api/authentication.md)
- [Production deployment](docs/operations/production-deployment.md)
