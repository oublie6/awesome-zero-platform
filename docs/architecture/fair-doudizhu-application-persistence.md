# Fair Doudizhu Application and Persistence Architecture

## 1. Purpose

Goal 0021 surrounds the pure Goal 0020 aggregates with a transaction-oriented application layer and MySQL adapters. Transport remains outside this boundary. HTTP and WSS handlers will later derive the authenticated actor, parse a command, invoke one application method, and map the stable result.

```text
authenticated transport
  -> application command executor
     -> MySQL READ COMMITTED transaction
        -> command-row claim / duplicate replay
        -> expiry and sequence policy
        -> row-locked aggregate restore
        -> domain mutation
        -> protected contribution write when applicable
        -> optimistic snapshot write
        -> outbox append
        -> sequence advance / command completion
  -> commit
```

## 2. Command identity and replay

A command is identified by `(actorAccountId, commandId)`. The MySQL adapter claims this primary-key row inside the same transaction with `INSERT ... ON DUPLICATE KEY UPDATE`, then reads it with `SELECT ... FOR UPDATE`.

Concurrent copies of the same command wait on that row. The winner completes the result in the transaction; waiters then replay the committed result. A failed transaction rolls back the command claim, so a later retry can become the new winner. This avoids connection-scoped `GET_LOCK` calls and does not reserve an extra pooled connection while waiting.

The stored command identity includes:

- protocol and versioned command name;
- aggregate type and ID;
- client sequence and expected aggregate version;
- issue and expiry times;
- a server-computed SHA-256 digest of the command-specific payload.

The payload digest prevents an attacker or buggy client from reusing a command ID with a different ready value, commitment, terminal reason, or secure envelope.

After the command row is claimed, an existing exact command returns its original result with `duplicate: true`. It is returned before expiry, sequence, decryption, or aggregate execution. Rebinding an existing ID to different command metadata or payload is rejected.

## 3. Sequence policy

`clientSeq` is scoped to `(aggregateType, aggregateId, actorAccountId)`. MySQL creates or touches the sequence row with `INSERT ... ON DUPLICATE KEY UPDATE` and locks it with `SELECT ... FOR UPDATE`.

- `clientSeq == 0` is invalid.
- a new sequence must be greater than the stored value;
- stale or reused sequences are rejected without changing the sequence;
- once a sequence is admitted, a domain/business rejection consumes it and stores the original rejection result;
- infrastructure failures roll back the sequence and every other write.

This distinguishes a valid one-time business attempt from an operation that never reached a trustworthy business decision.

## 4. Aggregate persistence

Rooms and hands are stored as complete JSON snapshots plus indexed projection columns such as status, phase, active hand, reveal key ID, and aggregate version.

Repository reads use `SELECT ... FOR UPDATE`. Snapshot JSON is restored through `domain.RestoreRoom` or `domain.RestoreHand`, which validates invariants and does not emit synthetic creation events.

Updates include `WHERE aggregate_version = previousVersion`. A zero-row update is an optimistic persistence conflict and rolls back the command.

Starting a hand atomically:

1. locks and restores the room;
2. validates owner/readiness through the room aggregate;
3. obtains trusted hand setup metadata;
4. creates the hand aggregate;
5. inserts the hand and updates the room;
6. appends both aggregates' events and the command result.

Completing or otherwise terminating a hand atomically updates the hand and releases only the matching active hand from the room.

## 5. Concurrency and lock ordering

The adapter uses `READ COMMITTED` to reduce unnecessary gap and next-key locking. It does not take a process-global or database-global mutex. Contention is limited to the business keys that must be serialized:

1. `(actorAccountId, commandId)` command-result row;
2. `(aggregateType, aggregateId, actorAccountId)` client-sequence row;
3. the target room or hand snapshot row;
4. for terminal hand commands, the matching room row after the hand row.

Every command follows that order. Different rooms and hands can therefore progress on different database connections. Commands for the same aggregate are intentionally serialized at the aggregate row, because their exact `expectedVersion` and domain phase checks must observe a single authoritative order.

Starting a hand locks the room and inserts a new hand; it never locks an existing hand after locking the room. Terminal commands lock the existing hand and then its room. Future commands that touch two existing aggregates must preserve a documented canonical order rather than introducing a reverse room-to-hand path.

MySQL deadlock (`1213`) and lock-wait timeout (`1205`) errors are returned as `ErrRetryableTransaction`. The application does not blindly retry the callback inside the same request because reveal decryption, random nonce generation, setup providers, and ID generation may have observable cost or side effects. A transport or job runner may retry the same command with the same `commandId`; transaction rollback plus command-row idempotency makes that retry safe.

The repository's committed MySQL pool defaults are intentionally small for low-memory development. Production composition must size the pool, request concurrency, lock-wait timeout, and database capacity together. Increasing HTTP goroutines without increasing database capacity only moves the queue into the connection pool.

## 6. Reveal processing

A reveal command carries `secure-envelope-v1`; application code never trusts a client-supplied actor or seat. The hand snapshot resolves the actor's fixed seat and existing commitment.

Canonical HPKE associated data binds:

- command/reveal protocol versions and command name;
- reveal key ID;
- command ID;
- aggregate type and hand ID;
- authenticated actor and server-resolved seat;
- client sequence and expected version;
- prior commitment;
- issued and expiry times.

The decrypted JSON is strict and versioned. It contains:

```json
{
  "v": "fair-doudizhu-reveal-v1",
  "handId": "hand-id",
  "seat": 1,
  "secureRandom": "base64url-32-bytes",
  "phrase": "original user text",
  "normalization": "NFKC-v1"
}
```

Unknown fields, trailing JSON, context mismatches, invalid UTF-8, an empty/oversized phrase, or a random value other than exactly 32 bytes fail closed.

The phrase is normalized with Unicode NFKC. The contribution digest is:

```text
SHA-256(
  UTF8("fair-doudizhu/contribution/v1") || 0x00 ||
  U32BE(len(UTF8(handId))) || UTF8(handId) ||
  U8(seat) ||
  secureRandom[32] ||
  SHA-256(UTF8(NFKC(originalPhrase)))
)
```

The aggregate receives only this digest and an opaque protected-record ID.

## 7. Protected contribution records

The original reveal JSON is encrypted before the SQL adapter receives it. The first provider uses AES-256-GCM with:

- an injected current 32-byte key and versioned key ID;
- a fresh cryptographically secure GCM nonce;
- canonical record AAD binding record, hand, seat, actor, command, and contribution digest;
- a stored SHA-256 digest of that AAD.

The database table contains key ID, nonce, ciphertext, AAD digest, contribution digest, and metadata. It has no plaintext phrase or random columns. The in-memory static keyring is suitable only for tests and explicitly configured small deployments; production KMS/HSM and rotation operations remain deferred.

## 8. Outbox

Domain events are converted to `fair-doudizhu-event-v1` outbox rows in the same transaction as aggregate and command changes. Each row includes authoritative occurrence time, actor, causation command ID, aggregate/version, and JSON payload.

Sensitive reveal plaintext, secure random values, private cards, and complete active deck order are absent from domain event payloads. A unique `(aggregateType, aggregateId, aggregateVersion)` key prevents duplicate logical events even under retries.

A future worker will publish pending rows and mark `published_at`; delivery is at-least-once, so consumers must use `eventId` or aggregate version for deduplication. Parallel workers must preserve order per aggregate, for example by stable aggregate partitioning or a lease/claim protocol. A generic unordered `LIMIT N` publisher would be incorrect for gameplay events.

## 9. MySQL tables

The module owns:

- `doudizhu_rooms`;
- `doudizhu_hands`;
- `doudizhu_command_results`;
- `doudizhu_client_sequences`;
- `doudizhu_contribution_records`;
- `doudizhu_outbox_events`.

The schema remains the repository's complete current MySQL 5.7-compatible schema. The product module does not read identity, authorization, or platform audit tables directly.

## 10. Deferred composition

Later goals will compose:

- authenticated HTTP/WSS handlers and stable error mapping;
- public reveal-key manifests and key rotation;
- a KMS-backed contribution key provider;
- the outbox publisher and realtime routing;
- beacon network verification;
- deterministic shuffle, dealing, bidding, play, settlement, and fairness transcript publication.
