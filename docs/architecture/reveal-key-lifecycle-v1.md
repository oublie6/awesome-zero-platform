# Reveal Key Lifecycle and Signed Manifests v1

## Boundary

Reveal transport confidentiality remains an RFC 9180 HPKE concern, while public-key authenticity and lifecycle are owned by `server/foundation/revealkeys`. Fair Doudizhu consumes that foundation through application ports and stores only immutable public context in its hand aggregate.

The three key families remain independent:

- X25519 HPKE reveal keys: frequent rotation, normally every 14–30 days and immediately after suspected exposure;
- Ed25519 manifest signing roots: low-frequency rotation, normally annual or after a security event;
- AES/KMS contribution-storage keys: an independent at-rest lifecycle.

No production private key belongs in Git. Static JSON configuration is an explicit bootstrap adapter for environment variables or mounted Secrets; production deployments can replace it with KMS/HSM providers without changing the domain or client protocol.

## Lifecycle

A key record has one state:

- `active`: the only key selectable for a new hand;
- `retiring`: not selectable for new hands, but a hand bound before `retiringAt` may reveal until `retireAfter`;
- `retired`: unavailable for publication and decryption;
- `revoked`: immediately unavailable, including for unfinished hands.

Exactly one configured key is `active`. Every record has a unique monotonically assigned `manifestVersion`, `notBefore`, and `expiresAt`. A retiring key additionally has `retiringAt` and `retireAfter`, with the grace deadline no later than key expiry.

An unfinished hand bound to a revoked key cannot accept further reveals. The service returns the existing stable reveal-invalid failure; an operator or orchestration layer must abort the hand through the existing terminal command and restart with the current active key. Plaintext is never opened before lifecycle authorization succeeds.

## Hand binding

At hand creation the setup provider obtains the current manifest and records:

- `revealKeyId`;
- `revealPublicKeySha256`, SHA-256 over the raw 32-byte X25519 public key;
- `revealKeyBoundAt`.

The values are immutable aggregate metadata, included in snapshots and the hand-created event, and persisted in MySQL. Reveal execution sends the exact binding to the secure-transport adapter before HPKE open. The adapter rejects an unknown ID, hash mismatch, invalid validity window, post-retirement bind, expired grace period, retired key, or revoked key.

## Canonical signed manifest

The signature covers UTF-8 JSON emitted from a fixed field-order structure without insignificant whitespace. Binary values use unpadded canonical Base64URL. Timestamps use UTC RFC 3339. The signed fields are:

1. `manifestVersion`
2. `protocolVersion`
3. `keyId`
4. `suite`
5. `publicKey`
6. `publicKeySha256`
7. `notBefore`
8. `expiresAt`
9. `status`
10. optional `retiringAt`
11. optional `retireAfter`
12. `signatureKeyId`

`signature` is Ed25519 over those canonical bytes and is appended only after signing.

## Client trust and rollback

The TypeScript package accepts an explicit immutable map from `signatureKeyId` to raw Ed25519 root public key. Product builds may embed roots or inject them through a reviewed build-time boundary; runtime network responses cannot add trust roots.

The verifier checks strict fields, fixed protocol and suite, Base64URL canonical form, Ed25519 signature, X25519 key length, SHA-256 hash, time window, lifecycle state, and optional hand pin. It keeps:

- a global high-water `manifestVersion` for the current-key endpoint;
- a per-`keyId` high-water version for current and historical lookups.

This prevents rollback of the current key while still allowing a client that has already seen a newer current key to fetch an older retiring manifest required by a pinned unfinished hand.
