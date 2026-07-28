# Goal 0021: Fair Doudizhu Application and Persistence

## Status

- State: completed
- Started: 2026-07-28
- Completed: 2026-07-28
- Blockers: None.

## Goal

Implement the transactional Fair Doudizhu application and persistence layer around the Goal 0020 aggregates, including command idempotency, monotonic client sequences, MySQL snapshots, outbox events, encrypted contribution records, and secure-envelope reveal orchestration before adding transport or card rules.

## Scope

ChatGPT owned architecture, implementation review, tests, failure diagnosis and fixes, schema integration, repository verification, commits, and pushes directly to `main`.

Delivered:

1. Application command and result types enforcing the v1 command envelope, trusted actor context, issue/expiry policy, stable error codes, duplicate-result replay, and aggregate version reporting.
2. Transactional room and hand services covering room creation/join/leave/readiness, atomic hand creation, fairness commit/reveal, beacon locking, lifecycle transitions, terminal handling, and room release after a terminal hand.
3. Repository and unit-of-work ports for command records, client sequences, room/hand snapshots, protected contribution records, and outbox events.
4. MySQL 5.7-compatible infrastructure adapters and complete current schema definitions for aggregate snapshots, command results, sequence state, encrypted contribution records, and outbox delivery.
5. Domain snapshot restoration with invariant validation and no synthetic creation events.
6. A versioned reveal plaintext and canonical associated-data codec binding the authenticated actor, server-resolved seat, hand, commitment, command identity, sequence, version, reveal key, issue time, and expiry.
7. Secure-envelope reveal orchestration with an explicit opener port, NFKC phrase normalization, deterministic contribution digests, and decrypted-buffer clearing where practical.
8. AES-256-GCM protected contribution records with injected versioned keys and random nonces; plaintext is never passed to the MySQL adapter.
9. Unit, race, SQL-adapter, real-MySQL, Redis, Compose, HTTP/WS, and HTTPS/WSS verification.
10. A permanent dedicated Fair Doudizhu GitHub Actions workflow for focused module and MySQL verification.

The following remain intentionally deferred:

- HTTP/WSS handlers, public routes, authentication composition, realtime fan-out, public-key publication, signed key manifests, and Cocos client reducers;
- production KMS/HSM integration, key rotation operations, outbox workers, Redis routing, beacon-provider network adapters, and operational Admin UI;
- card/deck representation, deterministic shuffle, dealing, bidding, hand-pattern validation, turns, scoring, settlement rules, and fairness transcript publication.

## References

- `AGENTS.md`
- `docs/requirements/fair-doudizhu-v1.md`
- `docs/architecture/fair-doudizhu-domain.md`
- `docs/architecture/fair-doudizhu-application-persistence.md`
- `docs/api/fair-doudizhu-protocol-v1.md`
- `docs/api/fair-doudizhu-application-v1.md`
- `docs/architecture/secure-envelope-v1.md`

## Acceptance Results

- Existing command results are loaded before timing, sequence, or aggregate mutation and are returned with `duplicate: true` without re-execution.
- New commands require a positive monotonic `clientSeq` per authenticated actor and aggregate. Business rejections after admission consume the sequence and persist the original rejection; stale sequences fail before aggregate execution.
- Command result, sequence advancement, room/hand snapshots, protected contribution record, and outbox events share one transaction and roll back together on infrastructure failure.
- Start-hand updates the room and inserts the hand atomically. Terminal hand processing releases only the matching active hand from the room in the same transaction.
- MySQL command and sequence streams are serialized, aggregate rows are locked for update, and snapshot updates use optimistic version predicates.
- Restored room and hand snapshots are validated against domain invariants and do not emit synthetic creation events.
- Reveal AAD is canonical and binds protocol/name, command ID, hand, actor, server-resolved seat, sequence, expected version, commitment, reveal key, issued time, and expiry.
- Reveal plaintext is versioned, context-bound, strictly decoded, limited in size, and contains exactly 32 secure-random bytes plus the original UTF-8 phrase.
- Contribution digests are deterministic from hand, seat, secure random, and SHA-256 of the NFKC-normalized phrase. Raw phrase and random bytes are absent from snapshots and outbox payloads.
- Contribution plaintext is protected with AES-256-GCM before persistence. Records contain only metadata, key ID, nonce, ciphertext, and AAD digest and reject invalid keys, modified metadata, modified ciphertext, or wrong AAD.
- The complete MySQL 5.7 schema advanced to version `0009` and includes rooms, hands, command results, client sequences, contribution records, and outbox events.
- Unit tests, race tests, `go vet`, generated-code verification, formatting, secure-envelope interoperability, Admin Web build, real MySQL/Redis integration, and production runtime acceptance all passed.
- The final repository diff contains only the Goal 0021 application/persistence implementation, its tests/docs/schema/CI integration, three focused CI corrections, and this completion report.

## Verification Evidence

- Baseline main: `562b2bdf885d42616c43d6348bf56cebc860bcd2`.
- Goal definition commit: `264c2257b9f3aec96118a95b1ab656b8bf21c9be`.
- Integrated application/persistence implementation checkpoint: `c4cd8c3f8fd6e72c8b47cba1f9306aff6044fa29`.
- Initial CI run `30318515145` exposed three concrete defects: one Go formatting difference, an outdated bootstrap schema-version assertion (`0008` instead of `0009`), and two missing explicit `domain.AccountID` conversions in the MySQL integration test.
- Corrections were committed as `f8b2a25232917cb21c256f4a96340dd254df1b4d`, `cad55a7cc60781f6de8c2a2e33f69022e4c7ccf3`, and `8095c6b2c5bd9402a2372b3471860eb5d47042bc`.
- A temporary latest-`main` verification workflow was used because GitHub App content commits and ordinary `push` workflow status emission were inconsistent during this execution. Its first runtime attempt correctly exposed a verification-harness environment-precedence bug: a step-level bootstrap token overrode the runtime script's generated env-file token. Product code and containers were healthy; the harness was isolated and corrected.
- Verified implementation commit: `864d7d9a7e1ab46c1a410cd12eff47f7d0999c6f`.
- Full verification run `30322515954` succeeded. It passed:
  - `go mod tidy` cleanliness;
  - goctl generated-code repeatability and `gofmt` cleanliness;
  - all repository Go unit tests;
  - Fair Doudizhu `-race` tests and `go vet`;
  - TypeScript secure-envelope build/tests and TypeScript-to-Go HPKE interoperability;
  - Admin Web typecheck/build;
  - clean MySQL 5.7 and Redis schema/seed/integration tests, including concurrent duplicate commands, sequence serialization, optimistic updates, rollback, outbox persistence, and protected contribution storage;
  - production Compose startup, administrator bootstrap and login, HTTP/WS, HTTPS redirect/health, authenticated WSS, bootstrap-token removal, and final login/realtime checks.
- Commit status `ci/goal0021` was recorded as `success` for the verified implementation commit and points to run `30322515954`.
- The temporary verification workflow was deleted in cleanup commit `69eea25b3dca6bac6d41dcb59d4165080bdea765`. That cleanup changed no production, schema, client, or test code.
- GitHub compare from the Goal 0020 baseline through cleanup reported the expected Goal 0021 files only: application/domain restoration, infrastructure adapters, tests, schema, focused CI, README, and protocol/architecture documentation.

## Completion Report

Goal 0021 is complete. Fair Doudizhu now has a tested transactional application boundary and MySQL persistence layer around its room and hand aggregates. Duplicate commands replay their original results, client sequences are serialized, aggregate versions are protected, mutations and outbox records commit atomically, sensitive reveal plaintext is HPKE-opened and NFKC-normalized before deterministic digest verification, and the original contribution is encrypted with AES-256-GCM before database insertion.

The next goal should implement the versioned card/deck model, deterministic shuffle and verification transcript, and dealing result as pure domain/application modules. HTTP/WSS transport should remain separate so card and fairness rules can be tested without network concerns.
