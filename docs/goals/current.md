# Goal 0019: Secure Envelope and Cocos Client Foundation

## Status

- State: in_progress
- Started: 2026-07-27
- Completed: Not yet.
- Blockers: None.

## Goal

Create the first reusable client-to-server secure-envelope capability and the minimum code-first Cocos Creator client skeleton before implementing Fair Doudizhu business logic.

## Scope

ChatGPT owns all architecture, implementation, tests, failure fixes, CI integration, commits, and pushes directly to `main`.

Deliver:

1. A versioned HPKE protocol document using X25519, HKDF-SHA256, and AES-256-GCM in RFC 9180 Base mode.
2. A reusable Go server opener under `server/foundation/secureenvelope`.
3. A reusable TypeScript client sealer under `clients/packages/secure-envelope`.
4. A secure asynchronous-random adapter suitable for `wx.getRandomValues` without `Math.random` fallback.
5. A cross-language TypeScript-seal to Go-open CI test.
6. A minimal code-first Cocos Creator 3.8 LTS project skeleton under `clients/fair-doudizhu-cocos`.

Do not implement rooms, card rules, game state, fairness Commit-Reveal orchestration, public-key HTTP endpoints, signing manifests, database envelope encryption, or production key management in this goal.

## References

- `AGENTS.md`
- `docs/architecture/secure-envelope-v1.md`
- RFC 9180

## Acceptance Criteria

- Go accepts only the fixed v1 protocol and suite, enforces field and plaintext limits, selects keys by key ID, clears temporary key copies, and hides cryptographic error details.
- TypeScript accepts only valid raw X25519 public keys, returns the exact v1 JSON shape, uses unpadded base64url, enforces limits, and supports standard Web Crypto or a prefetched asynchronous secure-random source.
- No path uses `Math.random` or fairness contribution bytes as encryption-key material.
- Go and TypeScript unit tests pass.
- A TypeScript-generated envelope is opened by the Go implementation in CI.
- Tampered AAD, ciphertext, and encapsulated-key tests fail closed.
- The Cocos skeleton targets 3.8 LTS, is code-first, isolates WeChat entropy access, and contains no product business logic.
- Existing Go, Admin Web, integration, HTTPS, WSS, and Compose CI remains green.
- The final goal state, verification evidence, commit SHA, push result, and any unavailable editor/device verification are recorded honestly.

## Working State

### Completed

- Protocol and module boundaries were agreed with the user.
- The repository workflow, current baseline, architecture rules, and CI were inspected.

### In progress

- Implementing protocol, Go opener, TypeScript sealer, cross-language test, Cocos skeleton, and CI integration.

### Remaining

- Run GitHub Actions, fix failures, and record final verification.
- Run Cocos Creator editor import/preview later only if an actual host with Creator is available; this is not required for cryptographic module acceptance.

### Verification status

- Baseline main: `e7272658a93d9387167569a5ae3dfa4124044f92`.
- Baseline CI: `30265067913`, `ci/full: success`.
- Implementation verification pending.

## Completion Report

Pending.
