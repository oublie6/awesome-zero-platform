# Infrastructure Foundation Requirements

## Purpose

Provide the reusable PostgreSQL and Redis infrastructure required by later platform modules while preserving the modular-monolith architecture and the existing go-zero HTTP foundation.

This foundation owns connection lifecycle, configuration validation, health probing, readiness integration, transaction execution, current schema and seed definitions, and repeatable local development dependencies. It must not introduce user, authentication, authorization, or other product semantics.

## Scope

The first infrastructure phase includes:

- PostgreSQL connectivity and lifecycle management.
- Redis connectivity and lifecycle management.
- Startup configuration and validation for both dependencies.
- Dependency health checks and readiness aggregation.
- Current complete database schema and seed locations.
- A narrow transaction execution abstraction.
- Repeatable local PostgreSQL and Redis startup for development and integration testing.
- Focused unit and integration tests.

## Package boundaries

Infrastructure code belongs under focused packages within `server/foundation`. Prefer concrete names such as:

```text
server/foundation/
├── database/
├── cache/
└── readiness/
```

The exact package split may be refined when Go ownership or lifecycle concerns require it, but do not create generic `common`, `utils`, `helpers`, repository, CRUD, or dependency-injection frameworks.

`server/database` stores database definitions rather than connection code:

```text
server/database/
├── schema/
└── seed/
```

The repository stores only the current complete schema and seed definitions during this phase. Incremental migration history and committed environment-upgrade SQL are prohibited.

## PostgreSQL

The PostgreSQL foundation must:

- Use a maintained PostgreSQL driver and connection pool suitable for Go.
- Accept configuration without hard-coded credentials.
- Validate required connection and pool settings before opening the service.
- Establish and verify connectivity with bounded timeouts.
- Expose the concrete pool or narrow execution interfaces needed by later modules without hiding SQL behind a speculative generic repository.
- Close resources safely and idempotently during shutdown or partial startup failure.
- Support transaction execution with explicit context propagation, commit on success, rollback on error or panic, and correct error propagation.

The transaction helper must remain infrastructure-only. It must not encode business retries, distributed transactions, unit-of-work repositories, or automatic nesting semantics unless a later requirement explicitly needs them.

## Redis

The Redis foundation must:

- Use a maintained Redis client suitable for Go.
- Accept address, authentication, database index, pool, and timeout configuration without committed secrets.
- Validate configuration before startup.
- Verify connectivity using a bounded ping or equivalent operation.
- Expose the concrete client or narrowly scoped infrastructure interface required by later platform modules.
- Close resources safely and idempotently during shutdown or partial startup failure.

This phase must not define session, token, distributed lock, rate-limit, queue, pub/sub, cache-key, or business caching semantics.

## Startup and lifecycle

`app-api` owns the lifecycle of the infrastructure resources it uses.

Startup must:

1. Load and validate configuration.
2. Create PostgreSQL and Redis clients.
3. Verify required dependency connectivity with bounded timeouts.
4. Build the service context only after required resources are valid.
5. Close already-created resources if a later startup step fails.
6. Start the HTTP server only after required infrastructure is ready.

Shutdown must stop the HTTP service and close infrastructure resources without panics or avoidable leaks. Close operations should be safe when called after partial initialization.

## Liveness and readiness

`GET /health/live` continues to answer whether the process is alive and must not fail merely because PostgreSQL or Redis is temporarily unavailable.

`GET /health/ready` represents whether the process can serve dependency-backed requests. It must evaluate PostgreSQL and Redis using bounded checks and return:

- HTTP 200 with the existing probe-friendly payload when all required dependencies are ready.
- HTTP 503 with a safe probe-friendly payload when any required dependency is unavailable.

Readiness responses must not expose credentials, DSNs, internal network addresses, raw driver errors, or stack traces. Diagnostic details belong in server-side logs. Readiness checks must not hang indefinitely or create a new unbounded client per request.

## Configuration

Committed development configuration must contain safe local defaults or environment-variable placeholders, never real credentials.

Configuration must cover the fields required for:

- PostgreSQL connection and pool behavior.
- Redis connection and pool behavior.
- Startup connectivity timeout.
- Readiness probe timeout.

Invalid required values must produce a clear non-zero startup failure. Sensitive values must not be printed in startup errors or logs.

## Schema and seed policy

Create the schema and seed directories only when they contain real files.

The initial complete schema may be intentionally minimal because no platform tables exist yet. It should still provide a deterministic, executable definition that proves the database initialization workflow without inventing user or business tables. A schema metadata table or similarly infrastructure-neutral definition is acceptable when clearly documented.

Seed data must remain optional and deterministic. Do not add fake users, roles, permissions, or product data in this phase.

## Local development dependencies

Provide a repeatable local development definition for PostgreSQL and Redis, preferably using Docker Compose under an appropriate deployment or local-development location.

It must:

- Pin explicit compatible image versions rather than floating `latest` tags.
- Use development-only credentials clearly marked as non-production.
- Provide health checks.
- Avoid committing runtime volumes or database contents.
- Document startup, shutdown, reset, schema application, and test commands.

Do not add Kubernetes, production deployment manifests, secret-management systems, or cloud-specific resources.

## Testing

Tests must cover, where practical:

- Configuration validation and secret-safe error reporting.
- PostgreSQL successful connection and failed connection behavior.
- Redis successful connection and failed connection behavior.
- Partial startup cleanup when one dependency succeeds and another fails.
- Idempotent resource close behavior.
- Transaction commit, rollback on returned error, and rollback on panic.
- Liveness remaining healthy during dependency failure.
- Readiness returning 200 when dependencies are healthy.
- Readiness returning 503 when PostgreSQL or Redis is unavailable.
- Readiness responses not leaking sensitive or raw dependency information.
- Current schema initialization against a clean PostgreSQL database.

Integration tests may use the committed local dependency definition or another isolated, repeatable mechanism. Tests must not depend on a developer's pre-existing database contents.

## Deferred capabilities

The following are explicitly deferred:

- User, identity, authentication, sessions, tokens, JWT, and Casbin.
- Business repositories, models, tables, or seed data.
- Generic CRUD or generic repository frameworks.
- Database migration history and committed upgrade patches.
- Distributed transactions, sagas, outbox, event buses, or message queues.
- Redis sessions, locks, rate limiting, pub/sub, or business caching.
- Metrics, tracing backends, dashboards, CI, Kubernetes, and production deployment.
