# External System Integration Boundaries

## Scope and current implementation

The existing platform supports authenticated human account HTTP/WS access, internal module wiring, and game-specific versioned commands. **It does not yet provide a general third-party developer portal, service-account identity provider, API keys, webhook delivery service, or public integration gateway.** This document establishes the contract for implementing those capabilities when a specific external system is selected.

An internal business module and an external system integration are different extension modes:

- **Internal module:** compiled into the Go process; typed in-process service calls, its own model/store, explicit startup and shutdown.
- **External integration:** an independent process or organization; communicates over supported, explicitly versioned network contracts, never by directly sharing database tables, Redis keys, Go domain objects, or Cocos runtime code.

## Outbound/inbound connection policy

Select the smallest adapter that matches a real use case:

| Need | Candidate | Not implied |
| --- | --- | --- |
| Synchronous request/reply | HTTPS REST with versioned resource paths | sharing platform account passwords |
| Long-lived client updates | authenticated WSS channels with account/permission scoping | cross-node delivery guarantees |
| External event notifications | signed webhooks with persistent delivery and retries | exactly-once delivery |
| Internal asynchronous integration | domain events and transactional outbox **with a worker** | that outbox rows alone imply working publication |
| Organization identity federation | explicit OIDC/OAuth2 integration when required | using human refresh tokens as server credentials |

HTTP endpoints currently use the platform's established response envelopes, request IDs, error handling, and authentication. Any new machine-to-machine integration must design and verify separate client identity/credential issuance, scopes, revocation, rotation, auditing, and authorization. Do not assume existing account JWT or the Admin bootstrap token is a production API-key service.

## Required contract per external integration

Define and test at least:

1. System owner, trust boundary, permitted resources, identity mapping and least-privilege authorization.
2. API/event schema and explicit version; validation, size limits, pagination, timeouts and stable error codes.
3. Authentication method, key lifecycle, TLS/WSS domain and certificate requirements, replay mitigation, clock tolerance and audit redaction.
4. Idempotency keys and conflict semantics for mutable requests; retriable vs non-retriable outcomes.
5. Delivery ordering, deduplication, backoff and dead-letter/operational repair for events and webhooks where needed.
6. Quotas, rate limits, request tracing/correlation and operational dashboards for the actual traffic pattern.
7. Contract tests, backward-compatibility policy and consumer rollout/rollback.
8. What happens when the remote dependency is down: timeouts, circuit breaking only where measured, and graceful failure isolation.

Do not expose `doudizhu_*` or `identity_*` tables directly as an integration surface. External clients must not supply authoritative account, seat, ownership, or role claims. The receiving adapter validates identity and resolves those values from trusted services.

## Incremental roadmap

1. First external-system requirement: write a one-page integration contract and implement one dedicated adapter, not a generic gateway.
2. A second genuinely different consumer: extract common identity, request and telemetry code only where repetition is demonstrated.
3. Reliable async delivery requirement: add an outbox relay/consumer with durable retry, ordering and idempotency, and prove recovery in tests.
4. Multiple owners or independent scale: evaluate service boundaries according to [module boundaries](module-boundaries.md).

Existing `doudizhu_outbox_events` are persisted in the Doudizhu transaction; the publisher/consumer infrastructure is explicitly not part of the present runtime.
