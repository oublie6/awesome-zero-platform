# Engineering Principles — Canonical Decisions

Status: **accepted design direction** (2026-11-01). These documents define architectural constraints for future design, implementation, review and Goal acceptance. They do **not** claim that the corresponding capabilities are already implemented or verified.

## Mandatory reading and precedence

Every agent must read `AGENTS.md`, `docs/goals/current.md`, and the applicable principles below **before designing or changing code**. The root `AGENTS.md` mandates compliance. These principles supersede older aspirational statements in design documents when inconsistent, but **do not silently expand a current Goal**. If a Goal conflicts with them or needs a new backend/data scope, document the conflict and obtain an explicit Goal amendment first. Published protocol contracts and actual shipped implementations are not rewritten by an architectural decision alone.

1. [01 — Modularity and service evolution](01-modularity-and-evolution.md): domain ownership, go-zero first, boundary contracts, service extraction criteria.
2. [02 — Game runtime, performance and durable recovery](02-game-runtime-and-recovery.md): authoritative state, commit-before-ack, log, snapshots, Redis, fencing, performance gates.
3. [03 — Client portability and cross-platform battles](03-client-portability.md): WeChat-first qualification, platform-neutral protocol and state, unified account IDs, fairness and security.
4. [04 — Engineering and decision gates](04-engineering-gates.md): tests, evidence, design review, observability, performance budgets and operational safety.

## Decision categories

- **Required now:** module boundaries, dependency direction, server authority, command idempotency, no secret leakage, honest verification, resource bounds, explicit versioning and lifecycle cleanup.
- **Target architecture requiring dedicated implementation Goals:** durable B-level live-hand recovery, event log/checkpoints, epoch fencing, ownership routing, cross-node hand recovery, full three-player online gameplay, cross-platform account linking.
- **Deliberately deferred:** Redis as an authoritative per-turn database, universal actor/plugin framework, automatic zero-interruption failover, mandatory microservice splitting, advertising/payments/rankings and simultaneous qualification of all clients.

## Change protocol

Propose architectural changes in a dedicated decision section with rationale, alternatives, migration effects, failure modes and verification plan. Keep these principle documents concise and normative; put endpoint/schema details in versioned API docs and measured results in performance/operations reports. Never overwrite historical verification evidence to make it match a new target.
