# Fair Doudizhu Command and Event Protocol v1

## 1. Purpose

This document defines the engine-independent business protocol for Fair Doudizhu. Authenticated HTTPS or WSS carries the protocol. It does not replace TLS, authentication, authorization, command idempotency, or the secure envelope used for sensitive reveal plaintext.

Every protocol, command, result, and event name is versioned. Unknown versions and names fail closed.

## 2. Command envelope

A state-changing client command has this shape:

```json
{
  "v": "fair-doudizhu-command-v1",
  "name": "doudizhu.room.ready.set.v1",
  "commandId": "019c0123-4567-7abc-8def-0123456789ab",
  "aggregateType": "room",
  "aggregateId": "room_019c0123",
  "clientSeq": 17,
  "expectedVersion": 6,
  "issuedAt": "2026-07-28T00:00:00.000Z",
  "expiresAt": "2026-07-28T00:01:00.000Z",
  "payload": {
    "ready": true
  }
}
```

Required fields:

- `v` — exactly `fair-doudizhu-command-v1`;
- `name` — an allowed versioned command name;
- `commandId` — globally unique opaque ID, normally UUIDv7;
- `aggregateType` — `room` or `hand` as required by the command;
- `aggregateId` — target room or hand ID;
- `clientSeq` — positive monotonic sequence for the authenticated account and aggregate stream;
- `expectedVersion` — exact aggregate version observed by the client, or `0` only for aggregate creation;
- `issuedAt` and `expiresAt` — UTC timestamps with millisecond precision and a bounded validity window;
- `payload` — command-specific object.

The envelope must not contain authoritative `accountId`, `actorId`, `ownerId`, `role`, or `seat`. The server derives the actor from the authenticated session and resolves membership and seat from the aggregate.

The application computes a SHA-256 digest over the versioned command name and canonical payload representation. A `commandId` is bound to the complete command metadata and payload, not only to the outer envelope.

## 3. Command result

An accepted command returns:

```json
{
  "v": "fair-doudizhu-command-result-v1",
  "commandId": "019c0123-4567-7abc-8def-0123456789ab",
  "accepted": true,
  "duplicate": false,
  "aggregateType": "room",
  "aggregateId": "room_019c0123",
  "aggregateVersion": 7,
  "events": [
    {
      "aggregateType": "room",
      "aggregateId": "room_019c0123",
      "name": "doudizhu.room.ready-changed.v1",
      "version": 7
    }
  ]
}
```

A rejected command uses the same result protocol with `accepted: false`, an empty `events` array, and a stable failure object:

```json
{
  "v": "fair-doudizhu-command-result-v1",
  "commandId": "019c0123-4567-7abc-8def-0123456789ab",
  "accepted": false,
  "duplicate": false,
  "aggregateType": "room",
  "aggregateId": "room_019c0123",
  "aggregateVersion": 8,
  "events": [],
  "failure": {
    "code": "DDZ_VERSION_CONFLICT",
    "message": "aggregate version conflict",
    "currentVersion": 8
  }
}
```

Stable business codes are:

- `DDZ_INVALID_COMMAND`
- `DDZ_FORBIDDEN`
- `DDZ_NOT_FOUND`
- `DDZ_NOT_SEATED`
- `DDZ_ROOM_FULL`
- `DDZ_ROOM_NOT_READY`
- `DDZ_HAND_ACTIVE`
- `DDZ_WRONG_PHASE`
- `DDZ_DUPLICATE_CONTRIBUTION`
- `DDZ_COMMITMENT_MISMATCH`
- `DDZ_BEACON_MISMATCH`
- `DDZ_VERSION_CONFLICT`
- `DDZ_SEQUENCE_CONFLICT`
- `DDZ_HAND_TERMINAL`
- `DDZ_REVEAL_INVALID`
- `DDZ_CONFLICT`

Unexpected database, cryptographic-provider, ID-generation, or commit failures are infrastructure errors. They are not persisted as business decisions.

## 4. Idempotency, ordering, and retries

A command is uniquely identified by `(authenticatedActor, commandId)`.

- The first transaction claims the command row and completes the original result atomically with aggregate, sequence, contribution, and outbox writes.
- Concurrent exact copies wait for that row and then return the stored result with `duplicate: true`.
- Exact duplicates are replayed before expiry, sequence, decryption, or aggregate mutation checks, so a reconnect can safely retry an old command.
- Reusing the same ID with changed metadata or payload returns `DDZ_CONFLICT`.
- A new command must have `clientSeq` greater than the last admitted sequence for `(aggregateType, aggregateId, actor)`.
- A business rejection after sequence admission consumes the sequence and is replayable.
- A malformed, expired, or stale-sequence command does not advance the sequence.
- A MySQL deadlock or lock-wait timeout rolls back all writes and is retryable with the same command ID.
- A version conflict is a completed business decision; the client reloads state and creates a newly considered command with a new ID and higher sequence.

WSS delivery does not make a command one-time. All replay and ordering guarantees are server-side and database-backed.

## 5. Room commands

### `doudizhu.room.create.v1`

Target: new `room`; `expectedVersion` must be `0`.

Payload:

```json
{}
```

The authenticated actor becomes owner and seat 1.

### `doudizhu.room.join.v1`

Target: `room`.

```json
{}
```

The server assigns the first empty fixed seat. The client cannot request a seat.

### `doudizhu.room.leave.v1`

Target: `room`.

```json
{}
```

The room resolves the actor's seat. Leaving is rejected while a hand is active.

### `doudizhu.room.ready.set.v1`

Target: `room`.

```json
{
  "ready": true
}
```

### `doudizhu.room.hand.start.v1`

Target: `room`.

```json
{
  "handId": "hand_019c0123"
}
```

Only the server-recognized owner may start. The application locks trusted server commitment, reveal key ID, and public-beacon plan while atomically updating the room and inserting the hand.

Room release after a terminal hand is an internal application operation inside the terminal hand transaction; clients do not send a separate authoritative finish command.

## 6. Hand commands

### `doudizhu.hand.fairness.commit.submit.v1`

Target: `hand`.

```json
{
  "commitment": "base64url-32-bytes"
}
```

Exactly one commitment is accepted per server-resolved seat in `FAIRNESS_COMMITTING`.

### `doudizhu.hand.fairness.reveal.submit.v1`

Target: `hand`.

```json
{
  "secureEnvelope": {
    "version": "secure-envelope-v1",
    "keyId": "reveal-key-2026-07",
    "suite": "hpke-x25519-hkdf-sha256-aes-256-gcm",
    "encapsulatedKey": "base64url-32-bytes",
    "ciphertext": "base64url"
  }
}
```

The HPKE associated data canonically binds:

- reveal AAD protocol version;
- command protocol and command name;
- reveal key ID;
- command ID;
- aggregate type and hand ID;
- authenticated actor and server-resolved seat;
- client sequence and expected version;
- the seat's prior commitment;
- issue and expiry timestamps.

The decrypted plaintext is strict JSON:

```json
{
  "v": "fair-doudizhu-reveal-v1",
  "handId": "hand_019c0123",
  "seat": 1,
  "secureRandom": "base64url-exactly-32-bytes",
  "phrase": "the original user phrase",
  "normalization": "NFKC-v1"
}
```

The server verifies context, normalizes the phrase using Unicode NFKC, recomputes the contribution digest and commitment, encrypts the original plaintext for protected storage, and gives the domain only the digest plus opaque record ID. Plaintext is never written to a domain event or aggregate snapshot.

### `doudizhu.hand.beacon.lock.v1`

Internal application command targeting `hand`:

```json
{
  "provider": "configured-beacon",
  "round": "locked-round-id",
  "digest": "base64url-32-bytes",
  "proofRef": "opaque-proof-reference"
}
```

The provider and round must match immutable hand metadata. Network fetching and proof verification are infrastructure responsibilities.

### Internal lifecycle commands

These are issued by trusted application/rule modules, not accepted directly from arbitrary clients:

- `doudizhu.hand.dealt.v1` — `DEALING -> BIDDING`;
- `doudizhu.hand.play.start.v1` — `BIDDING -> PLAYING`;
- `doudizhu.hand.settlement.start.v1` — `PLAYING -> SETTLING`;
- `doudizhu.hand.complete.v1` — `SETTLING -> COMPLETED` and atomically releases the room.

### Terminal commands

- `doudizhu.hand.cancel.v1` — controlled pre-deal cancellation;
- `doudizhu.hand.abort.v1` — integrity or operational failure;
- `doudizhu.hand.expire.v1` — timeout policy reached.

Payload:

```json
{
  "reasonCode": "FAIRNESS_REVEAL_TIMEOUT"
}
```

Transport maps permitted actions to server-controlled reason codes. External clients do not supply arbitrary authoritative reasons.

## 7. Event envelope

Persisted and realtime events use:

```json
{
  "v": "fair-doudizhu-event-v1",
  "name": "doudizhu.hand.phase-changed.v1",
  "eventId": "019c0123-aaaa-7bbb-8ccc-0123456789ab",
  "aggregateType": "hand",
  "aggregateId": "hand_019c0123",
  "version": 8,
  "occurredAt": "2026-07-28T00:00:01.000Z",
  "causationCommandId": "019c0123-4567-7abc-8def-0123456789ab",
  "actorAccountId": "019c0000-0000-7000-8000-000000000001",
  "payload": {
    "from": "FAIRNESS_COMMITTING",
    "to": "FAIRNESS_REVEALING"
  }
}
```

Events for one aggregate are strictly ordered by `version`. The database enforces uniqueness of `(aggregateType, aggregateId, version)`. Delivery is at-least-once, so consumers deduplicate by `eventId` or aggregate version and must not process later versions before earlier versions for the same aggregate.

Domain event names are:

- `doudizhu.room.created.v1`
- `doudizhu.room.player-joined.v1`
- `doudizhu.room.player-left.v1`
- `doudizhu.room.ready-changed.v1`
- `doudizhu.room.hand-started.v1`
- `doudizhu.room.hand-finished.v1`
- `doudizhu.room.closed.v1`
- `doudizhu.hand.created.v1`
- `doudizhu.hand.fairness-commit-accepted.v1`
- `doudizhu.hand.fairness-reveal-accepted.v1`
- `doudizhu.hand.public-beacon-locked.v1`
- `doudizhu.hand.phase-changed.v1`
- `doudizhu.hand.terminated.v1`

Sensitive contribution plaintext, secure random values, normalized phrases, active private cards, full active deck order, access tokens, and private keys are forbidden event fields.

## 8. Concurrency model

The MySQL implementation uses `READ COMMITTED` and fine-grained row serialization:

1. command-result row for the actor and command ID;
2. client-sequence row for actor and aggregate;
3. target aggregate row;
4. matching room row after the hand row for terminal hand operations.

Unrelated rooms and hands can execute concurrently. Same-aggregate commands intentionally serialize and then use `expectedVersion` and phase rules to establish one authoritative order. Future multi-aggregate operations must preserve the documented lock order to avoid deadlocks.

## 9. Transport privacy

Reveal commands use both WSS and `secure-envelope-v1`. HPKE is defense in depth so ordinary reverse-proxy and transport middleware need not see reveal plaintext.

The application process necessarily decrypts the contribution to verify it. HPKE does not protect against a compromised application process or stolen private key. Key management, memory-exposure reduction, encrypted storage, audit access, and rotation are separate controls.
