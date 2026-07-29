# Goal 0033: Server v1 Release Baseline

## Status

- State: completed
- Started: 2026-07-29
- Completed: 2026-07-29
- Blockers: None.

## Goal

Turn the completed maintainable server v1 into an accurate, operator- and client-developer-facing release baseline. Remove stale capability descriptions, provide one reproducible local three-player integration path, document the production enablement boundary, and leave the repository with a clear verified server-v1 release anchor.

## Delivered

1. Replaced the stale repository overview with an accurate Server v1 capability, quick-start, documentation, and deferred-boundary description.
2. Reworked `server/README.md` into the primary server usage guide covering startup, structure, authentication, administration, HTTP/WSS Doudizhu integration, configuration, deployment, persistence, and verification.
3. Added `docs/operations/server-v1-release.md` with a reproducible local path from dependency startup through three-player HTTP/WSS integration and reconnect.
4. Updated production deployment documentation with every optional Doudizhu and reveal-key value, secret-handling requirements, Compose usage, TLS, Kubernetes, and the single-process active-hand boundary.
5. Added environment overrides and production Compose forwarding for beacon provider, beacon round, and contribution-key identifier, completing the existing enablement path without changing public API or game behavior.
6. Added `server/foundation/revealkeys/cmd/local-env`, which generates fresh development-only X25519 reveal keys, an Ed25519 manifest-signing key, a beacon-proof secret, a contribution-encryption key, and the required non-secret identifiers.
7. Added tests that load the generated JSON through the real reveal-key registry and validate generated secret formats and shell output.
8. Removed the temporary Goal 0033 verification workflow after successful focused and full verification.

## Security and architecture boundaries

- Doudizhu remains disabled by default.
- No reusable private key or production secret is committed.
- The local generator writes nothing to the repository and is explicitly not a production key-management process.
- The authenticated account remains the only actor identity; no public API or game rule changed.
- MySQL stores durable setup and terminal data; Redis does not store active hands.
- Active cards, bids, turns, passes, and live versions remain authoritative in one API process.
- Process-crash restoration, distributed room ownership, cross-instance live migration, public matchmaking, spectators, rankings, bots, tournaments, and value-bearing features remain outside Server v1.

## Verification

Release-anchor commit:

```text
19744dbbeff27620ce29dec579cab3bcd39c61aa
```

Goal 0033 release run `30446686804` passed:

- focused Doudizhu configuration tests;
- local environment generator tests using the real reveal-key registry;
- fresh runtime key generation;
- production Compose rendering with Doudizhu enabled;
- environment-variable and release-document checks;
- repository cleanliness.

Full CI run `30446685971` passed for the same commit:

- module and generated-code cleanliness;
- formatting and all Go tests;
- race-sensitive security, Admin, realtime, and platform suites;
- server build and Compose validation;
- Secure Envelope TypeScript/Go interoperability;
- Cocos randomness policy and Admin Web build;
- complete MySQL 5.7 and Redis integration;
- production HTTP, HTTPS, authenticated WS/WSS, bootstrap/login, and cleanup acceptance.

Final statuses:

```text
goal0033/release: success
ci/full: success
```

Temporary verification cleanup commit:

```text
881d6b13759e59f09430ba59f2df467e205fee0f
```

## Completion Report

Goal 0033 is complete. Server v1 now has an accurate repository entrypoint, a reproducible local integration path, a closed production configuration path, explicit secret-handling guidance, and a verified release anchor. Commit `19744dbbeff27620ce29dec579cab3bcd39c61aa` is the Server v1 source baseline proven by both focused release verification and the complete repository/runtime CI.
