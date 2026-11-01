# 01 — Modularity, Ownership and Service Evolution

**Rule: modular monolith now; independent deployment only when proven useful.** Awesome Zero Platform is a general-purpose Go/go-zero platform, **not** a Doudizhu-specific backend.

## Dependency direction and ownership

- `server/apps` is the composition/transport boundary, `server/platform` reusable application capabilities, `server/foundation` business-neutral technical primitives, and `server/business/<bounded-context>` business ownership. Reuse existing packages; do not reorganize solely to resemble this description.
- Within a business: **Adapter/Transport → Application → Domain**. Infrastructure implements narrow ports owned by the consuming business/application. Domain must not import go-zero, MySQL/Redis drivers, Kafka, WebSocket, Cocos or platform SDKs.
- Foundation/platform must not depend on Doudizhu or a specific game; `gamecore` is game-domain shared code, not required by unrelated businesses. Do not elevate features to `gamecore` without genuine cross-game demand.
- Every module owns its tables, invariants, schemas and writes. Other modules use explicit typed public APIs/events/read models, not direct joins or imports of another module's repository/private state.
- Prefer go-zero's **verified** public, version-compatible facilities for infrastructure call paths; isolate native-client exceptions to adapters where functionality or semantics differ. Never sacrifice transaction semantics, cancellation, timeout, trace context or performance just to remove a native call.
- No generic `common/utils` dumping ground, dynamic discovery system, universal plugin SDK, premature queues or redundant frameworks. Abstraction needs actual consumers and tests.

## Extraction-ready interfaces

Each public module operation defines ownership and authorization, input/output and version, deadline, cancellation, error class, idempotency, consistency boundary, observability IDs and retry rules. An in-process interface can later receive an RPC adapter, but RPC introduces partial failure; it is **not** transparently interchangeable.

Extract a service only for evidence of independent scalability, isolation, deployment cadence, security or operational ownership. Before extraction verify data migration/rollback, identity propagation, version compatibility, cross-module transaction handling, health/shutdown behavior, and performance cost.

## Game module positioning

Keep gameplay-specific rules in `business/doudizhu/domain`, use cases and commit coordination in `application`, storage/runtime implementations in `infrastructure`. Shared game execution mechanics may live in `business/gamecore` only where an established consumer requires them. API/WSS adapters do not own business rules. Keep the initial deploy as a module-composed Go process; do not create a hypothetical `game-service` executable until a goal justifies it.

## Acceptance gate

Every Goal touching a boundary identifies the owner, public contract, prohibited imports, data ownership and extraction implications. Add import-boundary and contract tests where practical. A directory diagram alone is not evidence of modularity.
