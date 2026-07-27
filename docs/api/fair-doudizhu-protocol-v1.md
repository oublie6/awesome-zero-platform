# Fair Doudizhu Command and Event Protocol v1

## 1. Purpose

This document defines the engine-independent command and event contract for Fair Doudizhu. It is a business protocol carried over authenticated HTTPS or WSS. It does not replace TLS, authentication, or the secure envelope used for sensitive reveal plaintext.

All names are versioned. Unknown versions or command names fail closed.

## 2. Command envelope

A state-changing command has this shape:

```json
{
  "v": "fair-doudizhu-command-v1",
  "name": "doudizhu.room.ready.set.v1",
  "commandId": "019c0123-4567-7abc-8def-0123456789ab",
  "aggregateType": "room",
  "aggregateId": "room_019c0123",
  "clientSeq": 17,
  "expectedVersion": 6,
  "issuedAt": "2026-07-28T00:00:00Z",
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
- `expectedVersion` — exact aggregate version observed by the client;
- `issuedAt` — auxiliary UTC timestamp used by application expiration policy;
- `payload` — command-specific object.

The envelope must not contain authoritative `accountId`, `actorId`, `ownerId`, `role`, or `seat`. The server derives the actor from the authenticated session and resolves membership from the aggregate.

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
      "name": "doudizhu.room.ready-changed.v1",
      "version": 7
    }
  ]
}
```

When persistence is implemented, a repeated `commandId` for the same authenticated actor returns the original result with `duplicate: true`. The aggregate is not invoked a second time.

A rejected command uses the platform error envelope and one of these stable business codes:

- `DDZ_INVALID_COMMAND`
- `DDZ_FORBIDDEN`
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

A version conflict includes the current aggregate version in non-sensitive error details so the client can reload state.

## 4. Room commands

### `doudizhu.room.join.v1`

Target: `room`.

Payload:

```json
{}
```

The server assigns the first empty fixed seat. A client cannot request a seat.

### `doudizhu.room.leave.v1`

Target: `room`.

Payload:

```json
{}
```

The room resolves the actor's seat. Leaving is rejected while a hand is active.

### `doudizhu.room.ready.set.v1`

Target: `room`.

Payload:

```json
{
  "ready": true
}
```

### `doudizhu.room.hand.start.v1`

Target: `room`.

Payload:

```json
{
  "handId": "hand_019c0123"
}
```

Only the server-recognized room owner may start. Before constructing the hand, the application also locks the server commitment, reveal key ID, and beacon plan from trusted server configuration.

### `doudizhu.room.hand.finish.v1`

Target: `room`.

This is an internal application command, not a direct client command.

Payload:

```json
{
  "handId": "hand_019c0123"
}
```

It is issued only after the corresponding hand is terminal.

## 5. Hand commands

### `doudizhu.hand.fairness.commit.submit.v1`

Target: `hand`.

Payload:

```json
{
  "commitment": "base64url-32-bytes"
}
```

Exactly one commitment is accepted per server-resolved seat in `FAIRNESS_COMMITTING`.

### `doudizhu.hand.fairness.reveal.submit.v1`

Target: `hand`.

Payload:

```json
{
  "secureEnvelope": {
    "v": "secure-envelope-v1",
    "keyId": "reveal-key-2026-07",
    "suite": "DHKEM(X25519,HKDF-SHA256)/HKDF-SHA256/AES-256-GCM",
    "enc": "base64url",
    "ciphertext": "base64url"
  }
}
```

The HPKE associated data binds at least:

- protocol version and command name;
- reveal key ID;
- command ID;
- aggregate/hand ID;
- client sequence;
- expected version;
- prior client commitment;
- server-resolved actor and seat context;
- expiration policy data.

After decryption and validation, the application invokes the domain with the computed contribution digest and encrypted-record reference. Raw plaintext is never placed in a domain event.

### `doudizhu.hand.beacon.lock.v1`

Target: `hand`. Internal application command.

Payload:

```json
{
  "provider": "nist-randomness-beacon",
  "round": "2026-07-28T00:00:00Z",
  "digest": "base64url-32-bytes",
  "proofRef": "beacon-proof-opaque-reference"
}
```

The provider and round must match the plan locked at hand creation.

### Internal phase commands

These are application commands emitted only after their future rule modules have succeeded:

- `doudizhu.hand.dealt.v1` — `DEALING -> BIDDING`;
- `doudizhu.hand.play.start.v1` — `BIDDING -> PLAYING`;
- `doudizhu.hand.settlement.start.v1` — `PLAYING -> SETTLING`;
- `doudizhu.hand.complete.v1` — `SETTLING -> COMPLETED`.

### Terminal commands

- `doudizhu.hand.cancel.v1` — normal pre-deal cancellation;
- `doudizhu.hand.abort.v1` — integrity or operational failure;
- `doudizhu.hand.expire.v1` — timeout policy reached.

Payload:

```json
{
  "reasonCode": "FAIRNESS_REVEAL_TIMEOUT"
}
```

External clients do not supply arbitrary authoritative terminal reasons. Transport maps permitted user actions to server-controlled reason codes.

## 6. Event envelope

Persisted or realtime events use:

```json
{
  "v": "fair-doudizhu-event-v1",
  "name": "doudizhu.hand.phase-changed.v1",
  "eventId": "019c0123-aaaa-7bbb-8ccc-0123456789ab",
  "aggregateType": "hand",
  "aggregateId": "hand_019c0123",
  "version": 8,
  "occurredAt": "2026-07-28T00:00:01Z",
  "causationCommandId": "019c0123-4567-7abc-8def-0123456789ab",
  "payload": {
    "from": "FAIRNESS_COMMITTING",
    "to": "FAIRNESS_REVEALING"
  }
}
```

Events for one aggregate are strictly ordered by `version`. A reconnecting client loads an authoritative snapshot and may then consume events newer than the snapshot version.

Domain event names fixed by Goal 0020 are:

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

Sensitive contribution plaintext, active private cards, full active deck order, access tokens, and private keys are never event payload fields.

## 7. Replay, ordering, and concurrency

WSS does not make a command one-time. Clients retry after reconnect and may accidentally send the same JSON more than once.

The server therefore enforces all of these independently:

1. unique command ID and persisted original result;
2. monotonic client sequence;
3. exact expected aggregate version;
4. phase and membership validation inside the aggregate;
5. database uniqueness and atomic command/event persistence;
6. server-controlled actor and seat resolution.

A new command with stale aggregate state must fail rather than silently overwrite newer state. The client reloads the snapshot and decides whether to issue a new command with a new ID and sequence.

## 8. Transport privacy

Reveal commands use both WSS and `secure-envelope-v1`. HPKE is defense in depth so TLS termination, reverse-proxy access logs, and ordinary middleware do not need to see the reveal plaintext.

The server application necessarily decrypts the contribution to verify it. HPKE does not protect against a compromised application process or stolen private key; key management, memory exposure reduction, encrypted storage, audit access, and rotation remain separate controls.
