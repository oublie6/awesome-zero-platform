# Infrastructure Foundation Requirements

## Purpose

Provide the reusable MySQL and Redis infrastructure required by later platform modules while preserving the modular-monolith architecture and the existing go-zero HTTP foundation.

MySQL is the platform's default relational database. The runtime foundation must not retain PostgreSQL compatibility layers, dual-database abstractions, or lowest-common-denominator SQL solely for hypothetical portability.

This foundation owns connection lifecycle, configuration validation, health probing, readiness integration, transaction execution, current schema and seed definitions, and repeatable local development dependencies. It must not introduce user, authentication, authorization, or other product semantics.

## Scope

The infrastructure foundation includes:

- MySQL connectivity and lifecycle management.
- Redis connectivity and lifecycle management.
- Startup configuration and validation for both dependencies.
- Dependency health checks and readiness aggregation.
- Current complete database schema and seed locations.
- A narrow transaction execution abstraction.
- Repeatable local MySQL and Redis startup for development and integration testing.
- Focused unit and integration tests.

## Package boundaries

Infrastructure code belongs under focused packages within `server/foundation`:

```text
server/foundation/
├── database/
├── cache/
└── readiness/
```

The database package may expose the concrete MySQL handle or narrow execution interfaces actually required by later modules. Do not create generic `common`, `utils`, `helpers`, repository, CRUD, dependency-injection, driver-plugin, or multi-database frameworks.

`server/database` stores database definitions rather than connection code:

```text
server/database/
├── schema/
└── seed/
```

The repository stores only the current complete MySQL schema and seed definitions during this phase. Incremental migration history and committed environment-upgrade SQL are prohibited.

## MySQL

The MySQL foundation must:

- Use a maintained MySQL driver suitable for Go and the standard `database/sql` lifecycle where appropriate.
- Accept configuration without hard-coded production credentials.
- Validate required connection, character-set, time-zone, timeout, and pool settings before opening the service.
- Establish and verify connectivity with bounded timeouts.
- Use explicit safe defaults suitable for UTF-8 application data, including `utf8mb4`.
- Expose the concrete database handle or narrow execution interfaces needed by later modules without hiding SQL behind a speculative generic repository.
- Close resources safely and idempotently during shutdown or partial startup failure.
- Support transaction execution with explicit context propagation, commit on success, rollback on error or panic, and correct error preservation.

The transaction helper must remain infrastructure-only. It must not encode business retries, distributed transactions, unit-of-work repositories, automatic nesting semantics, or database portability abstractions unless a later requirement explicitly needs them.

## Redis

The Redis foundation must:

- Use a maintained Redis client suitable for Go.
- Accept address, authentication, database index, pool, and timeout configuration without committed production secrets.
- Validate configuration before startup.
- Verify connectivity using a bounded ping or equivalent operation.
- Expose the concrete client or narrowly scoped infrastructure interface required by later platform modules.
- Close resources safely and idempotently during shutdown or partial startup failure.

This phase must not define session, token, distributed lock, rate-limit, queue, pub/sub, cache-key, or business caching semantics.

## Startup and lifecycle

`app-api` owns the lifecycle of the infrastructure resources it uses.

Startup must:

1. Load and validate configuration.
2. Create MySQL and Redis clients.
3. Verify required dependency connectivity with bounded timeouts.
4. Build the service context only after required resources are valid.
5. Close already-created resources if a later startup step fails.
6. Start the HTTP server only after required infrastructure is ready.

Shutdown must stop the HTTP service and close infrastructure resources without panics or avoidable leaks. Close operations should be safe when called after partial initialization.

## Liveness and readiness

`GET /health/live` continues to answer whether the process is alive and must not fail merely because MySQL or Redis is temporarily unavailable.

`GET /health/ready` represents whether the process can serve dependency-backed requests. It must evaluate MySQL and Redis using bounded checks and return:

- HTTP 200 with the existing probe-friendly payload when all required dependencies are ready.
- HTTP 503 with a safe probe-friendly payload when any required dependency is unavailable.

Readiness responses must not expose credentials, DSNs, internal network addresses, raw driver errors, or stack traces. Diagnostic details belong in server-side logs. Readiness checks must not hang indefinitely or create a new unbounded client per request.

## Configuration

Committed development configuration must contain clearly marked local-only defaults or environment-variable placeholders, never real production credentials.

Configuration must cover the fields required for:

- MySQL address, database, user, password, character set, time parsing, time zone, connection and pool behavior.
- Redis connection and pool behavior.
- Startup connectivity timeout.
- Readiness probe timeout.

Invalid required values must produce a clear non-zero startup failure. Sensitive values must not be printed in startup errors or logs.

## Schema and seed policy

Create the schema and seed directories only when they contain real files.

The current complete schema must use MySQL syntax and deterministic engine, character-set, and collation choices. It may remain intentionally minimal while no platform tables exist, but it must prove the initialization workflow without inventing user or business tables. An infrastructure-neutral schema metadata table is acceptable when clearly documented.

Seed data must remain optional and deterministic. Do not add fake users, roles, permissions, or product data in this phase.

## Local development dependencies

Provide a repeatable local development definition for MySQL and Redis under the deployment/local-development area.

It must:

- Pin explicit compatible image versions rather than floating `latest` tags.
- Use development-only credentials clearly marked as non-production.
- Configure MySQL with `utf8mb4` and deterministic local settings.
- Provide health checks.
- Use conservative memory settings suitable for a development machine with approximately 1–2 GB of available memory.
- Avoid committing runtime volumes or database contents.
- Document startup, shutdown, reset, schema application, and test commands.

Do not add Kubernetes, production deployment manifests, secret-management systems, or cloud-specific resources.

## Testing

Tests must run serially by default according to `AGENTS.md` and cover, where practical:

- Configuration validation and secret-safe error reporting.
- MySQL successful connection and failed connection behavior.
- Redis successful connection and failed connection behavior.
- Partial startup cleanup when one dependency succeeds and another fails.
- Idempotent resource close behavior.
- Transaction commit, rollback on returned error, and rollback on panic.
- Liveness remaining healthy during dependency failure.
- Readiness returning 200 when dependencies are healthy.
- Readiness returning 503 when MySQL or Redis is unavailable.
- Readiness responses not leaking sensitive or raw dependency information.
- Current schema initialization against a clean MySQL database.

Integration tests may use the committed local dependency definition or another isolated, repeatable mechanism. Tests must not depend on a developer's pre-existing database contents.

## PostgreSQL replacement policy

The active runtime, local development environment, configuration, tests, schema, documentation, and Go dependencies must use MySQL rather than PostgreSQL.

Historical references inside archived Goal 0003 may remain because they describe completed history. Outside archived history, remove stale PostgreSQL runtime references and dependencies. Do not retain dual support, compatibility switches, unused PostgreSQL code, or PostgreSQL-specific transitive dependencies that are no longer required.

## Deferred capabilities

The following are explicitly deferred:

- User, identity, authentication, sessions, tokens, JWT, and Casbin.
- Business repositories, models, tables, or seed data.
- Generic CRUD or generic repository frameworks.
- Multi-database support and database driver plugin systems.
- Database migration history and committed upgrade patches.
- Distributed transactions, sagas, outbox, event buses, or message queues.
- Redis sessions, locks, rate limiting, pub/sub, or business caching.
- Metrics, tracing backends, dashboards, CI, Kubernetes, and production deployment.
