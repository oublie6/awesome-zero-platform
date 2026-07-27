# Secure Envelope Protocol v1

## Purpose

`secure-envelope-v1` is a reusable client-to-server confidentiality envelope. Its first consumer will be Fair Doudizhu fairness reveals, but the protocol contains no card-game semantics.

It is defense in depth on top of HTTPS/WSS. It does not replace transport security, authentication, authorization, replay protection, state-machine validation, command idempotency, or database encryption.

## Cipher suite

The protocol uses RFC 9180 HPKE Base mode with one fixed suite:

```text
KEM:  DHKEM(X25519, HKDF-SHA256)  0x0020
KDF:  HKDF-SHA256                 0x0001
AEAD: AES-256-GCM                 0x0002
```

Wire suite name:

```text
hpke-x25519-hkdf-sha256-aes-256-gcm
```

Fixed HPKE `info` bytes, encoded as UTF-8:

```text
awesome-zero-platform/secure-envelope/v1
```

Algorithms are not client-negotiated. Unsupported versions or suite strings are rejected before key use.

## JSON envelope

```json
{
  "version": "secure-envelope-v1",
  "keyId": "reveal-key-2026-01",
  "suite": "hpke-x25519-hkdf-sha256-aes-256-gcm",
  "encapsulatedKey": "unpadded-base64url",
  "ciphertext": "unpadded-base64url"
}
```

- `keyId` is printable ASCII without surrounding whitespace and is at most 128 bytes.
- X25519 public and private keys are raw 32-byte values outside the envelope.
- `encapsulatedKey` decodes to exactly 32 bytes.
- `ciphertext` contains the AES-GCM ciphertext and 16-byte authentication tag.
- Binary fields use RFC 4648 base64url without padding.
- Default plaintext limit is 64 KiB. Callers may configure a smaller positive limit.

## AAD ownership

The foundation module treats AAD as opaque bytes. Each business protocol must define one canonical encoder and bind all replay-sensitive context. The Fair Doudizhu reveal AAD is expected to include at least:

```text
protocolVersion, keyId, gameId, handId, seat, accountId,
commandId, clientSeq, expectedGameVersion, clientCommit, expiresAt
```

Changing one AAD byte must make decryption fail.

## Key handling

- Public keys may be distributed openly but must be authenticated by HTTPS and, before production rollout, a signed key manifest.
- Private keys never enter client code or logs.
- `keyId` selects a private-key version and is locked into the business transaction before reveal.
- Key providers return caller-owned key bytes; the Go opener clears temporary copies after use.
- Transport HPKE keys and database envelope-encryption keys have independent lifecycles.

## Secure randomness in constrained clients

Standard runtimes use `crypto.getRandomValues`. Some WeChat runtimes expose only asynchronous `wx.getRandomValues`. The client package supports an asynchronous secure-random adapter by prefetching true random bytes, serializing the seal operation, and exposing a temporary synchronous `getRandomValues` view while HPKE runs.

The adapter:

- never uses `Math.random`;
- does not invent a deterministic PRNG;
- fails closed if the pool is unavailable, malformed, or exhausted;
- clears the prefetched pool after use;
- restores the previous global runtime state.

## Error and logging policy

Cryptographic open failures are intentionally collapsed into one public error. Handlers must not log keys, plaintext, ciphertext, AAD, raw random contributions, or library error internals. Operational metrics may record only version, key ID, result category, and bounded counters.

## Interoperability verification

CI performs both directions needed for the client-to-server contract:

1. TypeScript creates a recipient key pair and seals an envelope using the noble-backed HPKE suite.
2. Go imports the raw private key and opens the generated envelope using Cloudflare CIRCL.
3. Unit tests independently verify Go round trips, TypeScript round trips, metadata validation, limits, and AAD/ciphertext tamper rejection.
