# Goal 0022: Reveal Key Lifecycle and Signed Manifests

## Status

- State: in_progress
- Started: 2026-07-28
- Completed: Not yet.
- Blockers: None.

## Goal

Implement the production-oriented trust and lifecycle boundary for Fair Doudizhu HPKE reveal keys before card and shuffle rules: versioned X25519 key records, signed Ed25519 public-key manifests, public lookup APIs, client verification and rollback protection, and immutable per-hand reveal-key context.

## Scope

ChatGPT owns architecture, implementation, tests, failure diagnosis and fixes, schema/API/client integration, repository verification, commits, and pushes directly to `main`.

Deliver:

1. A reusable reveal-key registry model with `active`, `retiring`, `retired`, and `revoked` states, validity windows, retirement grace, current-key selection, and exact `keyId` lookup.
2. Canonical versioned public-key manifests containing protocol and suite identifiers, `manifestVersion`, `keyId`, raw X25519 public key, `publicKeySha256`, validity times, `signatureKeyId`, and an Ed25519 signature.
3. Server-side manifest signing and verification primitives with deterministic canonical encoding and no production private keys committed to the repository.
4. TypeScript client verification with an explicit embedded-root/configuration boundary, Ed25519 signature verification, fixed-suite enforcement, time validation, key/hash validation, and monotonic `manifestVersion` rollback protection.
5. Public HTTP contracts for `GET /api/v1/crypto/reveal-keys/current` and `GET /api/v1/crypto/reveal-keys/{keyId}`, including cache headers and rotation-aware behavior.
6. Fair Doudizhu hand metadata that locks both reveal `keyId` and `publicKeySha256` at hand creation and preserves them through snapshots and persistence.
7. Reveal execution that rejects expired, retired-without-grace, revoked, or hash-mismatched key contexts while allowing an already-started hand to continue on a valid `retiring` key during its grace window.
8. Explicit emergency-revocation behavior: unfinished hands bound to a revoked key cannot accept reveals and must be aborted or restarted through the existing terminal-hand flow.
9. Configuration/bootstrap boundaries for X25519 private material, Ed25519 signing material, root public keys, and future Secret/KMS/HSM providers; test fixtures may use deterministic non-production keys.
10. Unit, race, TypeScript, interoperability, HTTP contract, MySQL 5.7/Redis integration, and available full-repository CI verification.

The following remain outside this goal:

- production KMS/HSM network adapters and automated operational rotation jobs;
- certificate pinning or replacement of HTTPS/WSS transport security;
- card/deck representation, deterministic shuffle, dealing, gameplay rules, fairness transcript publication, and Cocos gameplay UI;
- public matchmaking, robots, spectators, voice, payments, withdrawals, or tradable assets.

## References

- `AGENTS.md`
- `docs/requirements/fair-doudizhu-v1.md`
- `docs/architecture/secure-envelope-v1.md`
- `docs/architecture/fair-doudizhu-domain.md`
- `docs/architecture/fair-doudizhu-application-persistence.md`
- `docs/api/fair-doudizhu-protocol-v1.md`
- `docs/api/fair-doudizhu-application-v1.md`
- `docs/architecture/reveal-key-lifecycle-v1.md` (to be created)
- `docs/api/reveal-key-manifest-v1.md` (to be created)

## Acceptance Criteria

- The registry rejects malformed identifiers, invalid X25519 key sizes, duplicate keys, overlapping or contradictory lifecycle data, non-monotonic manifest versions, and configurations without exactly one selectable current key.
- `active` keys are selectable for new hands only inside their validity interval. `retiring` keys are not selected for new hands but remain usable by already-bound hands through the configured grace deadline. `retired` and `revoked` keys are never usable for reveal decryption.
- A manifest has one deterministic canonical byte representation. Signing the same fields produces the same signed content; changing any signed field makes verification fail.
- Manifests use Ed25519 and the fixed Secure Envelope v1 HPKE suite. The public-key hash is SHA-256 over the raw 32-byte X25519 public key and is checked on both server and client.
- The TypeScript verifier trusts only configured root public keys, rejects unknown signing keys, invalid signatures, malformed Base64URL, wrong suite/protocol, invalid time windows, hash mismatches, and lower `manifestVersion` values than the persisted high-water mark.
- Public-key endpoints return signed manifests only, use stable JSON and error contracts, emit `ETag` plus bounded cache headers, support exact historical lookup during retirement grace, and do not expose private material.
- Every new hand locks `revealKeyId` and `revealPublicKeySha256`; restoration and MySQL snapshots preserve both values and reject missing or malformed hashes.
- Reveal orchestration resolves the hand-bound key context and fails closed on unknown, expired, retired, revoked, or hash-mismatched keys. A valid retiring key remains accepted only for hands created before retirement and only until its grace deadline.
- Emergency revocation tests prove that an unfinished hand cannot continue revealing with the revoked key and can enter the documented abort/restart flow without exposing plaintext.
- Production configuration examples contain references or environment-variable names only. No real X25519, Ed25519, database-protection, KMS, or HSM private key is committed.
- Go unit and race tests, TypeScript tests, Go/TypeScript interoperability, schema/adapter tests, real MySQL 5.7 and Redis integration, generated-code checks, formatting, vet, Admin Web build, Compose, HTTP/WS, HTTPS/WSS, and available full CI remain green.
- Final verification evidence records exact commands, actual failures and fixes, CI run IDs, commit SHAs, push result, unavailable checks, and intentionally deferred work.

## Working State

### Completed

- Goal 0019 Secure Envelope HPKE foundation is complete.
- Goal 0020 Fair Doudizhu protocol and pure domain aggregates are complete.
- Goal 0021 application and persistence are complete.
- Repository inspection found no existing signed reveal-key manifest, lifecycle registry, public-key endpoint, or client verification implementation.
- Goal 0022 definition was committed first as `1bdfd4a219d8bbd0476027d1880a16423a2df40e`.

### In progress

- Implementing lifecycle validation, signed manifests, client verification, immutable hand key binding, persistence, public APIs, bootstrap configuration, and tests.

### Remaining

- Run targeted and full verification, inspect failures and fix root causes, complete the Goal report, and remove any temporary verification workflow.

### Verification status

- Baseline main before Goal definition: `7030ba87d30ab5dc45969561e6c829afcbb071b2`.
- Goal definition commit: `1bdfd4a219d8bbd0476027d1880a16423a2df40e`.
- Goal 0021 full verification run `30322515954` was previously successful.
- Goal 0022 implementation verification pending.

## Completion Report

Pending.
