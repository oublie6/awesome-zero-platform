# Architecture Overview

## Repository boundaries

```text
awesome-zero-platform/
├── server/   reusable server foundation and capability modules
├── clients/  user-facing applications for different platforms
├── deploy/   environment and deployment definitions
├── docs/     architecture and development decisions
└── scripts/  repeatable engineering commands
```

## Server architecture

The server begins as a modular monolith:

```text
server/
├── apps/        runnable processes
├── modules/     business and platform capabilities
├── foundation/  reusable technical infrastructure
└── database/    current schema and seed definitions
```

Initial capability modules are expected to include identity, user, authorization, file, notification, audit, and system configuration. They should be added only when implementation begins.

## Evolution strategy

A module can remain in-process while exposing a stable interface. When independent scaling, ownership, deployment, or reliability requirements appear, the same interface can be backed by RPC or messaging.

Code is organized for possible extraction, but the project does not create microservices before a real need exists.

## Database policy

- The repository stores the current complete schema.
- Development databases may be rebuilt from the current schema.
- Incremental migration history is not maintained during the early foundation phase.
- Upgrade SQL for an existing environment is generated as a temporary deployment artifact and is not committed.

## Client policy

Clients share API contracts and error conventions, not complete UI or runtime implementations. Vue 3, WeChat Mini Program, H5, and app clients are created incrementally as real product needs appear.
