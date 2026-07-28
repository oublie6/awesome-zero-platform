# Fair Doudizhu Application Contract v1

This document refines the engine-independent command protocol with the Goal 0021 application semantics. It is authoritative for persistence and reveal processing; transport endpoints remain deferred.

## Command envelope

Every state-changing client command contains:

```json
{
  "v": "fair-doudizhu-command-v1",
  "name": "doudizhu.room.ready.set.v1",
  "commandId": "opaque-unique-id",
  "aggregateType": "room",
  "aggregateId": "room-id",
  "clientSeq": 17,
  "expectedVersion": 6,
  "issuedAt": "2026-07-28T01:00:00.000Z",
  "expiresAt": "2026-07-28T01:01:00.000Z",
  "payload": {"ready": true}
}
```

Times use UTC millisecond precision. The default accepted TTL is at most two minutes, with a bounded future-clock skew. A previously stored exact duplicate is replayed even when it is now expired.

The server computes and persists a payload digest. The client does not submit this digest. Reusing a command ID with any changed envelope or payload field returns `DDZ_CONFLICT`.

## Room create

`doudizhu.room.create.v1` targets a new room aggregate, uses `expectedVersion: 0`, and has an empty payload. The authenticated actor becomes owner and seat 1.

## Result persistence

Accepted and rejected application decisions use `fair-doudizhu-command-result-v1`. An exact duplicate returns the same accepted/rejected decision and event references with `duplicate: true`.

A new admitted command that reaches a domain rejection consumes its `clientSeq`. A command rejected before admission because it is malformed, expired, or stale does not advance the sequence. Every original result is persisted for deterministic retries.

## Concurrency and retries

The server serializes only the rows that represent the same command, sequence stream, or aggregate. Different room and hand IDs may execute concurrently. Same-command retries wait for the command-result primary key and then replay the committed original result.

MySQL deadlock and lock-wait-timeout failures are transient infrastructure failures. They do not consume `clientSeq`, do not persist a result, and roll back aggregate/outbox/contribution changes. The caller may retry the exact same request with the same `commandId`; it must not invent a new command ID until it knows whether the original transaction committed.

A version conflict is different: it is a durable business decision after sequence admission. The client must reload the aggregate and, when appropriate, send a newly considered command with a new `commandId`, a higher `clientSeq`, and the new `expectedVersion`.

## Reveal envelope

The reveal payload uses the actual `secure-envelope-v1` JSON shape:

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

HPKE AAD is canonical binary data and binds command version/name, key ID, command ID, hand, actor, resolved seat, sequence, expected version, prior commitment, issued time, and expiry. A changed field causes decryption failure.

The decrypted plaintext shape is:

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

The server recomputes normalization, phrase hash, contribution digest, and commitment. It stores the original plaintext only as an AES-256-GCM protected contribution record and exposes only the digest and opaque record ID to the aggregate.

## Stable application codes

Goal 0021 returns these stable business codes in command failures:

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

Unexpected SQL, cryptographic-provider, ID-generation, or commit failures are infrastructure errors and are not converted into durable business decisions; the transaction rolls back. MySQL `1205` and `1213` failures are specifically classified as retryable transaction conflicts.
