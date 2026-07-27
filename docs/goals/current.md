# Goal 0021: Fair Doudizhu Application and Persistence

## Status

- State: in_progress
- Started: 2026-07-28
- Completed: Not yet.
- Blockers: None.

## Goal

Implement the transactional Fair Doudizhu application and persistence layer around the Goal 0020 aggregates, including command idempotency, monotonic client sequences, MySQL snapshots, outbox events, encrypted contribution records, and secure-envelope reveal orchestration before adding transport or card rules.

## Scope

ChatGPT owns architecture, implementation, tests, failure fixes, schema integration, commits, and pushes directly to `main`.

Deliver:

1. Application command and result types that enforce the v1 command envelope, trusted actor context, issue/expiry policy, stable error codes, duplicate-result replay, and aggregate version reporting.
2. Transactional room and hand services covering room creation/join/leave/readiness, atomic hand creation, fairness commit/reveal, beacon locking, lifecycle transitions, terminal handling, and room release after a terminal hand.
3. Repository and unit-of-work ports for command records, client sequences, room/hand snapshots, protected contribution records, and outbox events.
4. MySQL 5.7-compatible infrastructure adapters and current complete schema definitions for aggregate snapshots, command results, sequence state, encrypted contribution records, and outbox delivery.
5. Domain snapshot restoration with invariant validation and no synthetic creation events.
6. A versioned reveal plaintext and associated-data codec that binds the authenticated actor, server-resolved seat, hand, commitment, command identity, sequence, version, reveal key, and expiry.
7. Secure-envelope reveal orchestration using an explicit opener port, NFKC phrase normalization, deterministic contribution digests, and zeroization of decrypted buffers where practical.
8. AES-256-GCM protected contribution records using injected versioned keys and random nonces. Production KMS and key distribution remain deferred, but plaintext must never be written to the database adapter.
9. Unit, race, SQL-adapter, and real-MySQL integration tests for transaction atomicity, duplicate replay, concurrent sequence protection, optimistic updates, outbox persistence, protected reveal records, and rollback on failures.

The following remain outside this goal:

- HTTP/WSS handlers, public routes, authentication composition, realtime fan-out, public-key publication, signed key manifests, and Cocos client reducers;
- production KMS/HSM integration, key rotation operations, outbox workers, Redis routing, beacon-provider network adapters, and operational Admin UI;
- card/deck representation, deterministic shuffle, dealing, bidding, hand-pattern validation, turns, scoring, settlement rules, and fairness transcript publication.

## References

- `AGENTS.md`
- `docs/requirements/fair-doudizhu-v1.md`
- `docs/architecture/fair-doudizhu-domain.md`
- `docs/api/fair-doudizhu-protocol-v1.md`
- `docs/architecture/secure-envelope-v1.md`

## Acceptance Criteria

- Existing command results are returned before sequence or aggregate mutation and are marked `duplicate: true` without re-executing the command.
- New commands require a positive `clientSeq` greater than the last admitted sequence for the authenticated actor and aggregate. Business rejections after sequence admission consume the sequence and persist an idempotent original rejection result; stale sequence commands fail before aggregate execution.
- Command result, sequence advancement, aggregate snapshot changes, protected contribution record, and outbox events commit atomically or all roll back.
- Start-hand updates the room and inserts the corresponding hand in one transaction; terminal hand processing can atomically release the room only for the matching active hand.
- Room and hand repositories use row locking plus optimistic version predicates and can restore validated aggregates without emitting creation events.
- Reveal associated data is canonical and includes the command protocol/name, command ID, hand ID, actor, server-resolved seat, client sequence, expected version, commitment, reveal key ID, issued time, and expiry.
- Reveal plaintext is versioned, bound to the same hand and seat, contains exactly 32 secure-random bytes and the original phrase, and is rejected when malformed, expired, oversized, or context-mismatched.
- The contribution digest is deterministic from the hand, seat, secure random, and SHA-256 of the NFKC-normalized phrase. The original phrase and random bytes are absent from domain snapshots and outbox payloads.
- Contribution records are AES-256-GCM encrypted before insertion, store only key ID/nonce/ciphertext/AAD digest and metadata, and fail closed for invalid key sizes or tampering.
- The MySQL schema is the complete current schema, remains compatible with MySQL 5.7, and advances `foundation_schema_meta.schema_version`.
- Unit and race tests pass for `business/doudizhu/domain`, `business/doudizhu/application`, and the contribution-protection package.
- SQL adapter tests verify expected statements and optimistic conflicts; integration tests verify real-MySQL persistence, duplicate replay, sequence serialization, rollback, outbox rows, and absence of plaintext in stored contribution records.
- Existing generated-code, Admin Web, secure-envelope, platform integration, HTTPS, WSS, and Compose checks remain green when GitHub Actions is available.
- Final verification evidence, commits, push result, unavailable checks, and intentionally deferred work are recorded honestly.

## Working State

### Completed

- Goal 0020 product, protocol, and pure domain aggregate foundations are complete on `main`.
- Goal 0021 boundaries, transaction semantics, reveal trust model, and persistence responsibilities were fixed.

### In progress

- Implementing domain restoration, application services, protected contribution storage, MySQL adapters, schema, and tests.

### Remaining

- Run all available verification, inspect the integrated diff, fix failures, complete the Goal report, and push the verified result to `main`.

### Verification status

- Baseline main: `562b2bdf885d42616c43d6348bf56cebc860bcd2`.
- Goal 0020 domain unit, race, vet, formatting, and dependency-boundary checks succeeded.
- Goal 0021 implementation verification pending.

## Completion Report

Pending.
