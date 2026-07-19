# Server

The server starts as a modular monolith built on go-zero.

## Structure

- `apps/` — runnable API, RPC, worker, or scheduler processes
- `modules/` — identity, user, authorization, file, notification, audit, and future business capabilities
- `foundation/` — database, cache, logging, errors, response, middleware, storage, tracing, and other technical building blocks
- `database/` — current complete schema and seed data

Modules should expose explicit interfaces so an in-process implementation can later be replaced by RPC without rewriting business logic.
