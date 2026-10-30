# go-zero API, HTTP, realtime, and modular-extension conventions

This document records the migration decisions following Goal 0038. The source of truth remains the actual route adapter and `AGENTS.md`; use the matching code and integration tests, not assumptions about the availability of a same-named go-zero component.

## API contract lifecycle

- `server/apps/app-api/app.api` is the **goctl-generated health endpoint** input used by `make generate`. Do not replace it with the full business API inventory without first migrating and verifying all corresponding generated handlers and DTOs.
- `server/apps/app-api/contracts/routes.api` is the **machine-checked inventory of 47 static HTTP routes**. It has no request/response schema and is **not** yet the authoritative generator input for business endpoints. `make api-contract-check` validates the syntax with goctl; `server/architecture/api_contract_test.go` checks inventory-to-implementation route parity. Dynamic `/ws` and `/metrics` are configured by runtime settings and remain excluded.
- When adding an endpoint, register its HTTP adapter using `rest.Route` under the owning module, update the route inventory and the client contract, and verify the API contract check + existing API integration tests. Do not claim full goctl-generated DTO or handler coverage until it actually exists.
- Future migration to generator-managed business handlers should occur per bounded module, preserving established `{code,message,requestId,data}` envelopes, case-sensitive JSON keys, HTTP codes, paths, and Casbin authorization. Generated handlers should delegate to application services, not hold new business logic.

## go-zero route middleware and justified custom behavior

- Protected authentication routes, protected Admin routes and Doudizhu HTTP routes register through go-zero `rest.WithMiddleware` / `rest.ToMiddleware`, while retaining **revocable Redis-backed Session semantics** and Casbin authorization. `rest.WithJwt` is not interchangeable with this contract.
- Existing outer router middlewares implement request ID, redacted structured access logs, JSON panic recovery, actual streaming body-size enforcement, CORS origin/credential policy and configurable security headers. They intentionally wrap 404/405 as well as normal go-zero routes.
- go-zero v1.10.2 `LogHandler` can include request contents or error dumps, including credentials; do **not** activate it blindly. Native `RecoverHandler` and `MaxBytesHandler` do not preserve current JSON-error and body stream contracts. The existing custom implementations are narrower compatibility adapters around go-zero REST, not a second independent HTTP framework.
- Other go-zero REST built-in middleware (Trace, Breaker, Shedding, Timeout, Metrics) remains available under normal `RestConf`. Avoid double registration and check interaction with WebSocket Upgrade / long polling before changing ordering.

## Realtime trace semantics

- Keep Gorilla WebSocket for protocol compliance and existing Hub/queue behavior. Incoming accepted JSON messages create a short `realtime.message` span through `core/trace.TracerFromContext`, and dispatch propagates that Context to business handlers and their SQL/Redis calls.
- The WebSocket connection can outlive the original HTTP handshake. Do not attach all messages to a finished handshake span. Without an explicit safe wire-level trace-context field, message spans are separate roots and **not** magically linked to the initial handshake or browser actions.
- Never use bearer tokens, private game payloads, user input, or arbitrary high-cardinality topic IDs as trace attributes. Prometheus realtime metrics remain on the existing low-cardinality registry.

## Redis connection resource boundary

- DB 0 normal commands use go-zero Redis; native go-redis remains for WATCH, subscriptions, special DB configurations, and lifecycle health checks. go-zero v1.10.2 internally shares connections managed by its own resource manager and does not automatically honor `Redis.PoolSize` or the native client's read/write timeouts.
- The existing `/metrics` handler exposes native Redis connection pool counts as `*_redis_native_pool_*`. This **excludes** go-zero managed pool connections; never add both pools together or treat native-only counts as total Redis traffic/usage.
- Keep existing Prometheus names and dashboard queries stable; observe actual Redis connection counts and request errors during load tests. Enforce compatible cancellations/timeouts at business call sites. Do not implement a second unmanaged Redis pool per module.

## Adding a business module

1. Add `server/business/<module>/domain`, `application` (public ports), and `infrastructure` only when the use case exists. Business modules do not initialize shared MySQL/Redis pools.
2. Add an `apps/app-api/internal/<module>api` adapter, declare only its real HTTP and WebSocket routes, then update `contracts/routes.api`; use go-zero route middleware for authentication/authorization.
3. Register explicitly in the `bootstrap/moduleSpec` list, supply a synchronous start and reverse-order shutdown. Do not copy other module's repositories, raw SQL tables or private game state.
4. Keep SQL access inside the owning module, prefer go-zero `sqlx` / Redis for supported operations and explain any native fallback. Do not add cross-module database transactions as an extraction strategy.
5. Use focused tests for module lifetime, init failure rollback, requests, authorization, persistence ownership, protocol backward compatibility and dependency boundaries. `server/architecture/boundaries_test.go` prevents platform access to business tables and limits the gamecore exception to the real Doudizhu module.
6. Extract an RPC service only after establishing independent deployment need, stable public contract, state authority, idempotency, errors, timeouts, trace propagation and rollback behavior.

## Verification before deployment

- `make generate` followed by a clean Git diff for generated files.
- `make api-contract-check && make fmt-check && make test`.
- Full MySQL/Redis integration and Fair Doudizhu CI, followed by the production Compose acceptance. Preserve existing K8s one-replica/Recreate baseline until a new rollout is explicitly authorized.
- A successful `realtime.message` span in CI with a synthetic parent does not substitute for server-side Jaeger verification of real message processing. That is a separately scoped supplementary runtime check.

## Goal 0042 observability corrections

- Native `sql.transaction` and `redis.watch.rotate` failures set OpenTelemetry error status. Native transaction events record a fixed failure message because callbacks may return private business payloads; original errors still reach callers unchanged. Transaction panic paths retain rollback and panic propagation while marking the ended span as failed. Raw transaction statements remain covered by the transaction boundary, without claiming individual SQL spans.
- Accepted WebSocket messages that fail validation, topic handling, dispatch or a business handler mark their message span with a bounded protocol error code. Private handler errors are not copied to spans; wire responses are unchanged.
- The outer access logger obtains the go-zero route's W3C trace context from the response headers injected by go-zero. This restores trace correlation without creating a duplicate request span or replacing the outer middleware's coverage of 404/405.
- HTTP metrics preserve standard method labels and group client-defined extension methods into `OTHER`, bounding both request-counter and latency-series cardinality.

- The application metrics endpoint also gathers the exact managed Redis operation/pool families verified in go-zero v1.10.2. `core/prometheus.Enable` activates the existing instruments without a second HTTP listener. Only the allowlisted Redis families are merged from the framework registry, avoiding duplicate Go/process collectors and unrelated global metrics. Native pool gauges retain their separate meaning. Redis command labels are command names, and the framework error label uses bounded categories rather than raw error text.

## Module-owned data and coordinated account-state locking

- Admin owns `authorization_roles` and `authorization_resources`: these are its administrator-facing metadata catalog. The authorization engine owns executable `authorization_casbin_rules` and `authorization_policy_state`; it does not query catalog tables. Existing table names, cache keys, public Admin DTOs and schemas remain unchanged. Ownership is explicit rather than inferred solely from a table-name prefix.
- Identity owns all `identity_*` SQL. Its narrow public infrastructure contract `identity.AnyActiveAccountForUpdate` lets the authorization adapter validate active accounts using the caller's existing query/transaction context. The owner performs the `FOR UPDATE` read and fully consumes/closes the result before returning a boolean; the authorization adapter decides the protected-role invariant. The caller preserves its existing sorted account IDs and policy-version-before-account lock order.
- This account-state contract is deliberately in-process and transactional. A future split must redesign the cross-module atomic invariant; passing the existing SQL transaction is explicit, and must not be represented as an independently deployable RPC contract. Removing the lock or substituting an eventually consistent account lookup would weaken last-active-administrator protection.
- Architecture tests now check SQL ownership between platform modules, in addition to platform/business separation, and pin Admin catalog ownership separately from authz policy ownership.
- `database.TraceTransaction` is the small shared tracing boundary for existing native transactions. Both identity transactions and clustered authorization mutations retain their own begin/commit/rollback behavior and propagate the span context. Authorization reload/publish happens after the transaction span ends; a post-commit reload failure still returns the existing partial-success result. Casbin's context-less `SavePolicy` transaction has an independent root span, not a falsely correlated HTTP child. Individual raw statements remain a documented native tracing limit.

- The native business command transaction also uses that same tracing boundary while retaining its explicit READ COMMITTED isolation, primary-key deduplication, callback context, error joins, panic propagation and retryable-deadlock classification. No business-specific symbols were introduced into foundation/platform, and the server deployment keeps that business disabled. Deterministic scripted-driver tests assert the isolation and that commit/rollback completes before the span ends.
- Access-log correlation prefers the route's injected response context even if an earlier request context already contains a different span; tests cover incoming W3C context overriding that earlier span. Invalid/missing response propagation retains the original context.
