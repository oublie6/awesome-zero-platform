# Goal 0033: Server v1 Release Baseline

## Status

- State: completed
- Started: 2026-07-29
- Completed: 2026-07-29
- Blockers: None.

## Goal

Turn the completed maintainable server v1 into an accurate, operator- and client-developer-facing release baseline. Remove stale capability descriptions, provide one reproducible local three-player integration path, document the production enablement boundary, and leave the repository with a clear verified server-v1 release anchor.

## References

- `AGENTS.md`
- `README.md`
- `server/README.md`
- `docs/api/authentication.md`
- `docs/api/fair-doudizhu-protocol-v1.md`
- `docs/architecture/doudizhu-realtime-v1.md`
- `docs/operations/realtime-websocket.md`
- `docs/operations/production-deployment.md`
- `server/apps/app-api/etc/main-api.yaml`
- `server/apps/app-api/etc/production.yaml`

## Deliverables

1. Updated the repository README so its capability and deferred-work sections match the completed server v1.
2. Updated the server README with the server-v1 overview, local startup path, Doudizhu enablement requirements, API map, verification commands, and process boundary.
3. Added `docs/operations/server-v1-release.md` with local dependency startup, generated development keys, administrator bootstrap, three-account preparation, authentication, HTTP/WSS integration, reconnect, verification, shutdown, production enablement, and v1 boundaries.
4. Updated production deployment documentation with the optional secrets and configuration required to enable Fair Doudizhu safely.
5. Completed the production enablement path by adding environment overrides and Compose forwarding for the beacon provider, beacon round, and contribution-key identifier.
6. Added a development-only local environment generator and tests that create and validate fresh reveal, manifest-signing, beacon-proof, and contribution-encryption keys without writing secrets into the repository.
7. Verified documentation, configuration, generated local environment, formatting, tests, builds, Compose, integration, and production runtime acceptance.
8. Recorded commit `19744dbbeff27620ce29dec579cab3bcd39c61aa` as the verified Server v1 release anchor.

## Constraints preserved

- All changes were committed directly to `main`.
- No public HTTP/WSS API, domain rule, persistence contract, or client-facing game behavior changed.
- Doudizhu remains disabled by default.
- No real password, access-token secret, contribution key, beacon-proof secret, reveal private key, signing private key, or TLS private key was committed.
- The local environment generator is explicitly development-only and creates fresh ephemeral key material at runtime.
- Active-hand crash restoration, cross-instance game migration, public matchmaking, spectators, rankings, balances, prizes, and value-bearing features remain outside server v1.

## Acceptance Criteria

- The root README no longer says completed Doudizhu transport, rules, play, settlement, or evidence capabilities are deferred.
- A new developer can identify the exact commands and documents needed to start the server and integrate three authenticated players.
- Doudizhu remains disabled by default, and the documentation lists every sensitive value required before enabling it.
- HTTP and WSS responsibilities, authentication, actor derivation, private-state delivery, reconnect behavior, and terminal evidence are described consistently with the implementation.
- Production documentation distinguishes the single-process in-memory active-hand boundary from durable MySQL/Redis state.
- All existing repository verification remains green.
- The completion report records the final verified release-anchor commit and intentionally deferred work.

## Agent Strategy

The primary agent owned documentation changes, limited release-enablement configuration, test design, consistency review, integration, and final verification. No supplementary subagent or Codex phase was required.

## Working State

### Completed

- Accurate repository and server entry documentation.
- Reproducible local three-player server startup and integration guide.
- Complete production Doudizhu environment-variable and Compose forwarding path.
- Development-only fresh-key environment generator with real registry-loading tests.
- Production enablement, secret-handling, deployment, and single-process boundary documentation.
- Focused release verification and full repository/runtime verification.
- Temporary Goal 0033 verification workflow removed after success.

### In progress

- None.

### Remaining

- None within Goal 0033 or the maintainable Server v1 release baseline.

### Verification status

Release-anchor commit: `19744dbbeff27620ce29dec579cab3bcd39c61aa`.

Goal 0033 release run `30446686804` passed:

- Doudizhu configuration tests;
- local environment generator tests using the real reveal-key registry;
- fresh runtime key generation;
- production Compose rendering with Doudizhu and reveal keys enabled;
- required environment-variable checks;
- release-document presence and stale-description checks;
- repository cleanliness after verification.

Full CI run `30446685971` passed for the same commit:

- module tidiness and generated-code cleanliness;
- repository formatting;
- all Go tests;
- security, Admin, realtime, and other race-sensitive suites;
- server build;
- local and production Compose validation;
- Secure Envelope TypeScript/Go interoperability and Cocos randomness policy;
- Admin Web typecheck and build;
- complete MySQL 5.7 and Redis integration;
- production HTTP, HTTPS, authenticated WS/WSS, bootstrap/login, and runtime cleanup acceptance.

Final statuses:

- `goal0033/release: success`
- `ci/full: success`

The temporary release workflow was removed by commit `881d6b13759e59f09430ba59f2df467e205fee0f` after the verified run completed.

## Completion Report

Goal 0033 is complete. The repository now presents Server v1 as the capability that actually exists rather than as an earlier partially implemented state. A developer can start dependencies, generate fresh local-only cryptographic configuration, launch the full backend, bootstrap administration, create three players, authenticate them, and integrate through the documented HTTP and WSS contracts.

Production documentation now states every platform and optional Doudizhu configuration value, distinguishes secret from non-secret values, passes the complete enablement set through Compose, keeps the feature disabled by default, and preserves the single-process active-hand boundary. The local generator closes the development startup gap without committing reusable private keys or weakening production key-management requirements.

Commit `19744dbbeff27620ce29dec579cab3bcd39c61aa` is the verified Server v1 release anchor. It passed both the focused Goal 0033 release verification and the repository's complete CI/runtime acceptance. The following remain intentional future-version work rather than Server v1 defects: active-hand process-crash restoration, distributed room ownership, cross-instance live migration, public matchmaking, spectators, rankings, bots, tournaments, balances, prizes, recharge, cash-out, or any transfer of value.
