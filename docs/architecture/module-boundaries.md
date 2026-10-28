# Module Boundaries and Evolution Contract

## Product purpose

Awesome Zero Platform is a **general-purpose backend platform**, not a game framework. Fair Doudizhu is the first domain used to validate reusable capabilities and the end-to-end delivery workflow. Both non-game business modules and independent external systems are first-class future consumers.

## Ownership and allowed dependencies

| Area | Owns | Must not own |
| --- | --- | --- |
| `server/foundation` | technical infrastructure, HTTP middleware, MySQL/Redis handles, metrics and reusable crypto primitives | accounts, Doudizhu rules, business-specific policies |
| `server/platform` | reusable identity, authentication, authorization, Admin and realtime services | game rules or product-specific state |
| `server/business/<name>` | domain model, application use cases, repositories, internal adapters and contracts for one business | another business's state or tables |
| `server/business/gamecore` | reusable **game-domain** primitives | general platform capabilities, required dependencies for non-game businesses |
| `server/apps/app-api` | runtime configuration, composition, transport adapters, process lifecycle | concrete business rules or cross-module database queries |
| `clients` | platform-specific shells, UI and SDK adapters | server-side authority over business state |

Allowed flow:

```text
apps (composition root) -> platform / business modules -> foundation
                           concrete games -> gamecore (optional)
```

`foundation` must not import `platform`, `business`, or `apps`. `platform` must not import `business` or `apps`. A concrete business may use public platform contracts, but must not import platform persistence adapters, another business's implementation, or `apps`. Concrete business modules do not import each other. Cross-module access goes through explicitly documented interfaces or versioned events.

## Business composition today

`apps/app-api/internal/bootstrap/modules.go` holds a narrow **compile-time** module startup/shutdown contract. `bootstrap/doudizhu.go` owns the Doudizhu assembly, HTTP/WSS registration, and lifecycle cancellation. The composition root lists the enabled concrete modules in one place and shuts them down in reverse startup order. Disabled modules are not initialized; duplicate/invalid module registrations fail startup. An initialization failure stops modules that already started before closing shared resources.

This is an application composition detail, **not** a dynamic plugin SDK. A second real business can implement its own module adapter without importing Doudizhu; only the explicit composition list and any needed runtime configuration should change. Do not introduce a universal module context that exposes every repository or secret. Pass the narrow capabilities required by each module.

## Data ownership today

One MySQL instance and one complete `server/database/schema/current.sql` are intentionally shared for deployment. Logical ownership is separate:

| Table prefix | Owner |
| --- | --- |
| `foundation_*` | `foundation` schema metadata |
| `identity_*` | `platform/identity` |
| `authorization_*` | `platform/authz` |
| `platform_audit_*` | `platform/admin` |
| `game_*` | `business/gamecore` |
| `doudizhu_*` | `business/doudizhu` |

Modules own writes and schema invariants. No business module directly queries another module's table, even when running inside the same process and database. A table can reference the shared account identifier as data without granting permission to mutate or read the identity owner's tables. A cross-module read should use the owner’s public API or a deliberately maintained projection. Do not depend on shared SQL transactions for future inter-service operations.

`server/architecture/boundaries_test.go` automatically checks dependency direction, business SQL references to known owned tables, and explicit classification of schema tables. These are regression guards, not a complete static data-flow or security audit. Dynamic SQL, reflective calls and cross-module semantics still require code review.

## Public contracts and future extraction

Internal synchronous calls are ordinary typed Go interfaces. Avoid network hops, service discovery, messaging brokers or generic RPC wrappers solely to prepare for microservices. When a business genuinely needs independent scaling or deployment, first confirm: owned data, compatible versioned contracts, error/timeout/idempotency semantics, service-to-service identity, tracing/correlation, independent lifecycle and migration/rollback.

The live Doudizhu hand runtime is deliberately single-process authoritative. Adding extra API replicas or extracting it as a microservice does not automatically provide recovery or routing. Keep its existing state and reliability contract until a separate design and tests justify changing it.

## New-module acceptance checklist

1. Place new domain types and persistence in its own `business/<name>` directory.
2. Define the minimum public module contract, business IDs and versioned transport/event APIs.
3. Give the module its own data prefix and update the table ownership tests and this document.
4. Register a narrow adapter in the application composition root with deterministic startup and shutdown.
5. Check that unrelated business modules remain unchanged, and that authentication and permissions are correctly scoped.
6. Add unit, integration, architecture and failure-path tests; update API and deployment documents.
7. Run `make test`, `make integration-test`, and existing CI/runtime acceptance before release.

See [External system integrations](external-integrations.md) and [Database evolution](../operations/database-evolution.md).
