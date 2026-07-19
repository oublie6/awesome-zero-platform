# AGENTS.md

## Project intent

Awesome Zero Platform is a modular application platform built on go-zero. It provides reusable server capabilities and supports multiple clients without coupling business modules to a specific frontend.

## Architecture rules

- Start as a modular monolith; extract services only when real scaling or ownership needs appear.
- `server/apps` contains runnable processes.
- `server/platform` contains reusable platform capabilities shared by different products.
- `server/business` contains product-specific business modules and should be created only when real business implementation begins.
- `server/foundation` contains reusable technical infrastructure without business semantics.
- Platform and business modules must not access another module's database tables or repository implementation directly.
- Cross-module calls must use explicit public interfaces or events.
- Keep transport, application logic, and persistence concerns separated.
- Do not create generic dumping grounds such as `common`, `utils`, or `helpers`.
- The repository stores the current complete database schema, not incremental migration history during the early development phase.
- Temporary database upgrade SQL must not be committed.

## Change rules

- Keep generated go-zero files distinguishable from handwritten code.
- Every public API change must update the relevant API documentation or schema.
- Every database structure change must update `server/database/schema`.
- Add tests for reusable foundation capabilities and module-level business rules.
- Prefer small, reviewable changes over broad speculative abstractions.
