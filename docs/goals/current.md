# Goal 0019: Secure Envelope and Cocos Client Foundation

## Status

- State: completed
- Started: 2026-07-27
- Completed: 2026-07-27
- Blockers: None.

## Goal

Create the first reusable client-to-server secure-envelope capability and the minimum code-first Cocos Creator client skeleton before implementing Fair Doudizhu business logic.

## Scope

ChatGPT owned the architecture, implementation, tests, failure fixes, CI integration, commits, and pushes.

Delivered:

1. A versioned HPKE protocol document using X25519, HKDF-SHA256, and AES-256-GCM in RFC 9180 Base mode.
2. A reusable Go server opener under `server/foundation/secureenvelope`.
3. A reusable TypeScript client sealer under `clients/packages/secure-envelope`.
4. A secure asynchronous-random adapter suitable for `wx.getRandomValues` without a `Math.random` fallback.
5. A cross-language TypeScript-seal to Go-open CI test.
6. A minimal code-first Cocos Creator 3.8 LTS project skeleton under `clients/fair-doudizhu-cocos`.

Rooms, card rules, game state, fairness Commit-Reveal orchestration, public-key HTTP endpoints, signed key manifests, database envelope encryption, and production key management remain outside this goal.

## References

- `AGENTS.md`
- `docs/architecture/secure-envelope-v1.md`
- RFC 9180

## Acceptance Results

- Go accepts only the fixed v1 protocol and suite, enforces metadata and plaintext limits, selects keys by key ID, clears temporary private-key copies, and collapses cryptographic failures.
- TypeScript validates raw X25519 public keys, emits the exact v1 JSON shape, uses canonical unpadded base64url, enforces limits, and supports Web Crypto or prefetched asynchronous secure entropy.
- No implementation path uses `Math.random` or fairness contribution bytes as encryption-key material.
- TypeScript package build and unit tests passed.
- Go unit and race tests passed, including tampered AAD, ciphertext, and encapsulated-key rejection.
- A TypeScript-generated HPKE envelope was successfully opened by the Go implementation in CI.
- The Cocos skeleton targets Creator 3.8.8, is code-first, isolates WeChat entropy access, and contains no product business logic.
- Existing Admin Web, MySQL/Redis integration, production Compose, HTTPS, and authenticated browser-mode WSS verification remained green.

## Verification Evidence

- Baseline main: `e7272658a93d9387167569a5ae3dfa4124044f92`.
- Baseline CI: run `30265067913`, `ci/full: success`.
- Foundation implementation commit: `4fba26f9dda542950dd97788b94370b6bf215670`.
- Secure-envelope CI integration commit: `076d4b3cb6527c075975d7c421cb722cdce55a08`.
- Verified implementation merge on `main`: `c23b413c27c5616d1702f6236ee4026f5722d3af`.
- Pull-request CI run `30287493380`: `unit`, `secure_envelope`, `admin_web`, and `integration` all succeeded.
- Dedicated runtime run `30288017127`: production Compose, HTTPS, and authenticated WSS acceptance succeeded.
- The linked GitHub App merge did not emit a `push` workflow run or a new `ci/full` commit status, so the same acceptance coverage was executed as the successful PR jobs and dedicated runtime job above rather than represented by one combined status.

## Completion Report

Goal 0019 is complete. The reusable client and server secure-envelope modules, protocol documentation, interoperability verification, CI coverage, and code-first Cocos skeleton are committed and merged to `main`.

Cocos Creator editor import/preview and real WeChat-device validation were not run because this execution environment has neither Cocos Creator nor a WeChat host. The repository records this limitation explicitly; it does not affect the cryptographic module or cross-language acceptance results.
