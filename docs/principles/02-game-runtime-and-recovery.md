# 02 — Game Runtime, Performance and B-Level Recovery

Status: **target architecture**. Today's live-hand state is in process memory and does **not yet guarantee** recovery after a process crash. Implementing this document requires a separate, explicitly authorized Goal, schema changes and failure-injection tests.

## Non-negotiable invariants

1. **One authoritative state per hand.** The server owns rules, private cards, turn order, settlement and fairness; never trust client-reported results.
2. **Different hands may execute concurrently; one hand's state transitions are serialized.** Use measured lock granularity; do not mandate one permanent goroutine per hand or introduce a speculative actor runtime.
3. **Commit before acknowledge:** an accepted command may be reported successful only when its corresponding recoverable transition/decision is durably committed. Treat ambiguous COMMIT responses as *unknown*, not failed; query by stable command ID before retrying.
4. **No memory/database split brain:** compute a candidate transition without exposing it, commit a version/epoch-checked operation record atomically with command deduplication and Outbox, then publish the committed state in memory. If commit fails, the candidate is discarded. A paused executor must not accept dependent operations until commit outcome is resolved.
5. **Confirmed operation recovery guarantee (B):** after application-process failure, with durable storage intact, reconstruct at least the last acknowledged committed transition. This is not a claim of seamless migration or survival of unrecoverable primary-storage loss.

## Storage responsibilities

- **Go memory:** normal-path hand execution and active views; no database read on every state query.
- **MySQL:** durable operation/event log, monotonic committed version, idempotent command results, owner/fencing epoch, transactional Outbox, authoritative recoverable snapshot/checkpoint metadata. Keep live mutations short, with no network fan-out or snapshot compression inside transactions. Align with existing room/fairness MySQL ownership; don't create conflicting commit mechanisms.
- **Redis:** optional hand-owner discovery/leases, presence, bounded hot caches and *discardable* snapshot copies. Redis is **not** authoritative for each play, nor sufficient by itself for the B recovery guarantee. Redis failure must not invalidate MySQL-committed operations.
- **Snapshots:** versioned, immutable, complete-enough-to-rebuild **server-side** state, encrypted where sensitive, checksummed and tied to rules/schema versions and committed version. These are different from per-player private client views. Create asynchronously from a consistent version; restore by replaying contiguous durable events from the latest trusted checkpoint. A trusted durable checkpoint or full genesis log must always remain available.
- **Fairness secrets:** recover necessary encrypted/controlled seed and reveal context, commitments, event version and protected references; do not regenerate a different deal or leak seeds/ciphertext into client broadcasts or ordinary Redis cache keys.

## Proposed log/checkpoint fields

Operation log: handId, stateVersion, stable commandId, authenticated account/seat resolved server-side, ownerEpoch, rulesVersion, event schemaVersion/type/payload, stateHash and commit timestamp. Store only minimal protected recovery material, not plaintext secrets in regular logs.

Checkpoint: handId, version, ownerEpoch, state/schema/rules version, encrypted full state or protected secret references, integrity checksum/hash, commit watermark and creation metadata.

Each `(handId, version)` must be unique; command dedupe scope and rejection semantics must agree with existing durable and live protocols. A command may generate multiple events: distinguish command sequence, event sequence and committed state version rather than assuming they are identical.

## Recovery state machine

`ACTIVE → SUSPECT → FENCED → RECOVERING → REPLAYING → ACTIVE`, with `QUARANTINED` on integrity/version failure. New writer acquires a strictly greater durable epoch; **every commit is fenced in the same durable transaction**. Redis TTL alone is not fencing. Verify snapshot schema/hash, replay contiguous committed records and validate resulting state hash. Recreate timer, channel, mutex and socket resources instead of serializing them; preserve logical deadline policy. Only allow new commands after the committed-watermark catch-up.

Lost responses: client resends the **same commandId and identical request** or queries outcome; an accepted-but-unanswered command cannot be blindly reissued with a different ID. Transaction timeout or network partition cannot turn a committed operation into an assumed failure. Reconnect sync returns an authorized private view; do not send another user's hand. If integrity cannot be established, quarantine rather than silently recreate a game.

## Performance and test gates

Record baseline and regressions for Go `Bid/Play/Pass` CPU/allocations, per-hand contention, 100/1,000 active hands, socket queues, state serialization, commit P50/P95/P99 and recovery replay. Suggested **initial targets, not measured promises**: pure rules P99 <1ms; committed server operation P99 <50ms; 1,000 concurrent hands under bounded memory and no cross-hand global lock bottleneck; recovery replay P95 <1s excluding fault detection/network reconnect. Benchmark real MySQL transaction durability settings and run fault injection before, during and after COMMIT, after memory publication and before push. Verify old-owner fencing, idempotency, hash integrity, corruption quarantine, rollback and timeouts.

A transaction acknowledged by the primary DB is not proof of no data loss during total storage failure. Higher guarantees require dedicated replication/failover durability analysis.

## Migration from current code

Existing `business/doudizhu/application.Service.execute` already has durable command/Outbox semantics for its domain commands. `LiveHandRuntime` / `LiveHandCoordinator` and in-memory `livehand.Game` require explicit adaptation for reliably recorded bidding/play/pass, deterministic candidate transitions, complete reconstructible state and protected SeedVault recovery. Do not falsely claim the current live-hand runtime already implements B-level recovery. Preserve existing API/protocol compatibility or version changes explicitly.
