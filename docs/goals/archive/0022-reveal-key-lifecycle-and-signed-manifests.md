# Goal 0022: Reveal Key Lifecycle and Signed Manifests

## Status

- State: completed
- Started: 2026-07-28
- Completed: 2026-07-28
- Blockers: None.

## Goal

Implement the production-oriented trust and lifecycle boundary for Fair Doudizhu HPKE reveal keys before card and shuffle rules: versioned X25519 key records, signed Ed25519 public-key manifests, public lookup APIs, client verification and rollback protection, and immutable per-hand reveal-key context.

## Scope

ChatGPT owned architecture, implementation, tests, failure diagnosis and fixes, schema/API/client integration, repository verification, commits, and pushes directly to `main`.

Delivered:

1. A reusable reveal-key registry model with `active`, `retiring`, `retired`, and `revoked` states, validity windows, retirement grace, current-key selection, and exact `keyId` lookup.
2. Canonical versioned public-key manifests containing protocol and suite identifiers, `manifestVersion`, `keyId`, raw X25519 public key, `publicKeySha256`, validity times, `signatureKeyId`, and an Ed25519 signature.
3. Server-side manifest signing and verification primitives with deterministic canonical encoding and no production private keys committed to the repository.
4. TypeScript client verification with an explicit embedded-root/configuration boundary, Ed25519 signature verification, fixed-suite enforcement, time validation, key/hash validation, and monotonic `manifestVersion` rollback protection.
5. Public HTTP contracts for `GET /api/v1/crypto/reveal-keys/current` and `GET /api/v1/crypto/reveal-keys/{keyId}`, including ETag and bounded rotation-aware caching.
6. Fair Doudizhu hand metadata that locks both reveal `keyId` and `publicKeySha256` at hand creation and preserves them through snapshots and persistence.
7. Reveal execution that rejects unknown, expired, retired, revoked, or hash-mismatched key contexts while allowing an already-started hand to continue on a valid `retiring` key during its grace window.
8. Explicit emergency-revocation behavior for unfinished hands.
9. Configuration and bootstrap boundaries for X25519 private material, Ed25519 signing material, root public keys, and future Secret/KMS/HSM providers.
10. Unit, race, TypeScript, interoperability, HTTP contract, MySQL 5.7/Redis integration, and full-repository verification.

The following remain intentionally deferred:

- production KMS/HSM network adapters and automated operational rotation jobs;
- certificate pinning or replacement of HTTPS/WSS transport security;
- card/deck representation, deterministic shuffle, dealing, gameplay rules, fairness transcript publication, and Cocos gameplay UI;
- public matchmaking, robots, spectators, voice, payments, withdrawals, or tradable assets.

## Acceptance Results

- Registry validation rejects malformed identifiers, invalid key sizes, duplicate keys, contradictory lifecycle data, non-monotonic manifest versions, and invalid current-key configurations.
- Active keys are selectable only inside their validity interval. Retiring keys are unavailable for new hands but remain usable by already-bound hands until their grace deadline. Retired and revoked keys are rejected.
- Manifests use deterministic canonical encoding, Ed25519 signatures, the fixed Secure Envelope v1 suite, and SHA-256 over the raw X25519 public key.
- TypeScript verification rejects unknown signing roots, invalid signatures, malformed Base64URL, wrong protocols or suites, invalid times, key/hash mismatches, and manifest-version rollback.
- Public endpoints expose signed manifests only and support current and exact-key lookup with stable errors, ETag, and bounded caching.
- Every new hand locks `revealKeyId`, `revealPublicKeySha256`, and binding time. Domain restoration and MySQL snapshots preserve and validate the binding.
- Reveal orchestration fails closed for unknown, expired, retired, revoked, or mismatched key contexts. Retiring-key grace and emergency revocation are covered by tests.
- Production configuration contains references and environment boundaries only; no production X25519, Ed25519, database-protection, KMS, or HSM private key is committed.
- The permanent Fair Doudizhu workflow verifies reveal-key Go tests, race tests, vet, TypeScript signed-manifest tests, Go-to-TypeScript interoperability, and real MySQL 5.7 integration.

## Verification Evidence

- Baseline main before Goal definition: `7030ba87d30ab5dc45969561e6c829afcbb071b2`.
- Goal definition commit: `1bdfd4a219d8bbd0476027d1880a16423a2df40e`.
- Goal start commit: `850feb52d72980aa780efc35753fb340e2b41fb9`.
- Verified implementation commit: `814767aeab3298be9141dca1b9c7f45bb750aeea`.
- Permanent Fair Doudizhu CI update: `00d860eda1966659d04a979d7a96b05d7883754d`.
- Real MySQL 5.7 compatibility correction: `7d7147bfe783fd1c5d6a12d239248fb95fa4e634`.
- Targeted implementation runs exposed and corrected three concrete defects:
  - a test fixture violated the registry manifest-version high-water invariant;
  - TypeScript implementation and tests referenced the nonexistent `SUITE_V1` export instead of `SECURE_ENVELOPE_SUITE`;
  - an integration-test `mysql.Config` constructed from its zero value did not enable MySQL native-password authentication.
- Final clean verification succeeded:
  - main CI run `30335660784`;
  - Fair Doudizhu run `30335660790`;
  - production Compose runtime acceptance run `30335660783`.
- Those runs passed module cleanliness, generated-code repeatability, formatting, all Go unit tests, race tests, vet, Go builds, TypeScript tests, HPKE interoperability, signed-manifest interoperability, Admin Web build, real MySQL 5.7 and Redis integration, Compose validation, HTTP/WS, HTTPS/WSS, administrator bootstrap and login, and production runtime acceptance.
- Cleanup run `30335996685` closed temporary PR `#7` and deleted the temporary trigger branch. Temporary apply, MySQL-fix, runtime-verification, and cleanup workflows were removed from `main` after use.

## Completion Report

Goal 0022 is complete. Fair Doudizhu now has a production-oriented reveal-key trust boundary: short-lived X25519 transport keys can rotate independently of long-lived Ed25519 signing roots; clients verify signed manifests and prevent rollback; every hand binds the exact reveal key and public-key hash; and the server enforces key lifecycle state, time windows, hash integrity, retirement grace, and emergency revocation before opening reveal ciphertext.

The next goal should implement the versioned 54-card Doudizhu card/deck model, deterministic random stream and unbiased shuffle, immutable dealing result, and a reconstruction-oriented fairness transcript as pure Go modules. HTTP/WSS transport, Cocos UI, bidding, play-pattern validation, turns, scoring, and settlement should remain separate.
