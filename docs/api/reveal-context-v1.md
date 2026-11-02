# Doudizhu server-signed reveal AAD v1 (2026-11-02)

## Why this is required

The Go application protects each player contribution with HPKE. Its original AAD includes the application command issuedAt and expiresAt, but the public WebSocket command used to assign them **only after receiving the request**. Without a server-issued context, the client cannot seal plaintext to the exact AAD; guessing clock times would cause decryption failure. This protocol removes the clock guessing without removing context bindings.

## Authorized flow

1. Three authenticated players join the same durable room. Its owner starts the hand. Every seated player reads `GET /v1/doudizhu/hands/:handId/fairness` (Bearer token) and obtains only the caller's seat, phase, aggregate version, its own committed hex digest (if present), the bound reveal key hash, key binding timestamp and its own lastClientSequence. No other player's secret is returned.
2. During `FAIRNESS_COMMITTING`, each client generates at least 32 cryptographically secure bytes, chooses a phrase, computes:
   `SHA256("fair-doudizhu/contribution/v1" || 00 || lengthPrefixed(handId) || seat || entropy32 || SHA256(NFKC(phrase)))`.
   Then it computes `SHA256("fair-doudizhu/client-commit/v1" || 00 || lengthPrefixed(handId) || seat || contributionDigest)`. The latter is sent as lowercase hex using an authenticated `hand.commit` command. No entropy or phrase appears in the commit, logs or telemetry.
3. Once **all three** commitments are accepted, each client reads the updated fairness state (`FAIRNESS_REVEALING`) and calls `POST /v1/doudizhu/hands/:handId/reveal-context` with `{requestId,clientSeq,expectedVersion}` using the same authenticated Bearer token.
4. The Go issuer checks the caller's seat, un-revealed committed contribution and current aggregate version; it generates a `doudizhu-reveal-context-v1` ticket with actor, hand, request ID, sequence, version, seat, commitment, reveal-key ID, reveal public-key SHA256, UTC millisecond issuedAt/expiry. The issuer HMACs the entire canonical ticket. The HMAC root is derived with domain separation from the production contribution key and is never returned. The response includes a base64url encoding of the exact `application.BuildRevealAAD`.
5. The client independently reconstructs the canonical AAD, requires exact equality, fetches the signed key manifest from the existing `/api/v1/crypto/reveal-keys/:keyId`, verifies its Ed25519 signature with the operator-provisioned public signing root and checks the pinned key ID/hash and original binding time. Without a trusted signing root or secure entropy the operation **fails closed**.
6. It creates a `fair-doudizhu-reveal-v1` plaintext, seals it via `SecureEnvelopeSealer` (X25519/HKDF-SHA256/AES-256-GCM), and submits `hand.reveal` with the exact `requestId`, `clientSeq`, `expectedVersion`, envelope and the signed reveal context. No plaintext, secret random, normalised phrase, HPKE keys or token appears in the URL.
7. Before executing, the dispatcher validates the HMAC, authenticated actor, hand identity, request ID, sequence, version, seat, signed time window and commitment shape. It replaces the internal command's issuedAt/expiry with the signed values, so the application decrypts with *exactly* the AAD that the client saw. MySQL command idempotency and aggregate version checks continue to enforce durable single-use semantics.

## Security/availability constraints

- The signed context is not a substitute for TLS, authenticated session, authorization, committed hand snapshot, a trusted manifest root or HPKE itself.
- Code can only issue contexts while the requesting seat is eligible to reveal. Wrong seat/actor/hand/version, missing signer, tampered MAC and expired tickets are rejected.
- Development configuration `__FAIR_DOUDIZHU_CONFIG__` must contain HTTPS `apiBase`, WSS `socketEndpoint` and the public `revealManifestRoots` mapping (`signatureKeyId` → base64url Ed25519 public key). The signing root is public; **do not** put private keys, HMAC roots, WeChat AppSecret or protected contribution keys into this config.
- The client retains its unrevealed contribution only in process memory in this checkpoint; destroying the process can lose the ability to complete a committed contribution. Secure, sealed per-hand local recovery of the secret is still required before release.
- An unknown result is retried with the **exact same** request ID, signed ticket and ciphertext. Ticket expiry before a known server receipt is a pending ambiguity; do not silently create a second contribution.
- The beacon/deal transition still requires **independently verifiable public randomness**, not a self-declared HMAC attestation. The existing test HMAC proof verifier is not evidence of an independent public beacon. No complete fair match is claimed without an authenticated public beacon, three-person end-to-end gameplay and post-commit fault tests.
