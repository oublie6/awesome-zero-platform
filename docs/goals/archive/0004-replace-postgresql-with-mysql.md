# Goal 0004: Replace PostgreSQL with MySQL

## Status

- State: completed
- Started: 2026-07-19
- Completed: 2026-07-19
- Blockers:

## Goal

Replace PostgreSQL with MySQL as the platform's single default relational database while preserving Redis, lifecycle and readiness behavior, the narrow transaction boundary, deterministic schema and seed workflows, low-memory execution constraints, and established HTTP behavior.

## Delivered

- Replaced PostgreSQL with MySQL using `database/sql` and `github.com/go-sql-driver/mysql` v1.10.0.
- Converted application configuration, bootstrap lifecycle, service context, readiness, tests, schema, seed, Docker Compose, Makefile commands, and active documentation.
- Preserved Redis, process-only liveness, dependency-aware readiness, partial-startup cleanup, idempotent close behavior, and transaction semantics.
- Removed active PostgreSQL code, dependencies, configuration, compose definitions, tests, and non-archived documentation references.
- Kept MySQL as the only active relational database without driver factories, dual-database compatibility, generic repositories, or CRUD frameworks.
- Added conservative MySQL settings and serial test execution for the 1–2 GB development machine.

## Verification

- `make generate` succeeded and remained repeatable.
- `make fmt` succeeded.
- `make test` succeeded serially with `go test -p 1 -parallel 1 ./...`.
- Pinned MySQL 8.4.10 and Redis 8.8.0-alpine3.23 started and reached healthy state.
- Schema and seed application succeeded against a clean MySQL database.
- Serial integration tests succeeded.
- Real application startup succeeded with healthy MySQL and Redis.
- `/health/live` returned HTTP 200 with `{"status":"ok"}` and `/health/ready` returned HTTP 200 with `{"status":"ready"}`.
- MySQL or Redis outage kept liveness at HTTP 200 and changed readiness to HTTP 503 with `{"status":"unready"}`.
- Startup and readiness failures did not expose credentials, DSNs, internal addresses, or raw driver errors to clients.
- Graceful shutdown and dependency cleanup succeeded.
- Explicit `ParseTime: false` is rejected rather than silently overwritten.

## Completion Report

Completed on July 19, 2026. Goal 0004 established MySQL as the platform's single default relational database while preserving Redis and all previously established infrastructure and HTTP behavior.
