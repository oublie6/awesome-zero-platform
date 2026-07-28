# Reveal Key Manifest API v1

## Endpoints

### `GET /api/v1/crypto/reveal-keys/current`

Returns the one signed `active` manifest selectable for a new hand.

### `GET /api/v1/crypto/reveal-keys/{keyId}`

Returns an exact signed manifest while the key is `active` or remains inside its `retiring` grace window. Retired, revoked, expired, malformed, and unknown IDs are not returned.

## Success response

```json
{
  "manifestVersion": 12,
  "protocolVersion": "reveal-key-manifest-v1",
  "keyId": "reveal-prod-2026-08",
  "suite": "hpke-x25519-hkdf-sha256-aes-256-gcm",
  "publicKey": "unpadded-base64url-32-bytes",
  "publicKeySha256": "unpadded-base64url-32-bytes",
  "notBefore": "2026-08-01T00:00:00Z",
  "expiresAt": "2026-09-15T00:00:00Z",
  "status": "active",
  "signatureKeyId": "manifest-root-2026",
  "signature": "unpadded-base64url-64-byte-ed25519-signature"
}
```

A retiring manifest additionally contains `retiringAt` and `retireAfter`.

## Cache contract

- The current endpoint emits `Cache-Control: public, max-age=300, must-revalidate`.
- Exact-key lookup emits `Cache-Control: public, max-age=60, must-revalidate`.
- Both emit a strong `ETag` over the response JSON and honor `If-None-Match` with `304`.
- Errors use `Cache-Control: no-store`.

Clients must still enforce manifest time windows and rollback protection. Cache freshness does not establish trust.

## Errors

Errors are stable JSON objects:

```json
{"code":"REVEAL_KEY_NOT_FOUND","message":"reveal key is unavailable"}
```

Relevant statuses are `400` for malformed IDs, `404` for unavailable IDs, `410` for an expired/revoked lifecycle result where distinguishable, and `503` when key publication is not configured or cannot be served. No response exposes private key material, internal cryptographic errors, or plaintext.
