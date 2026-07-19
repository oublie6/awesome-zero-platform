# Server

The server starts as a modular monolith built on go-zero.

## Structure

- `apps/` — runnable API, RPC, worker, or scheduler processes
- `platform/` — reusable platform capabilities such as identity, user, authorization, file, notification, audit, and system configuration
- `business/` — product-specific business modules; create this directory only when real business implementation begins
- `foundation/` — database, cache, logging, errors, response, middleware, storage, tracing, and other technical building blocks without business semantics
- `database/` — current complete schema and seed data

Platform and business modules should expose explicit interfaces so an in-process implementation can later be replaced by RPC or messaging without rewriting business logic.
