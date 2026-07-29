# Goal 0032: Authenticated Doudizhu WSS Synchronization and Reconnect

## Status

- State: completed
- Started: 2026-07-29
- Completed: 2026-07-29
- Blockers: None.

## Goal

Expose the Goal 0031 dispatcher through the existing authenticated WebSocket hub, deliver per-participant state changes without leaking private cards, and support reconnect by returning the latest private live snapshot or immutable terminal evidence. Keep active gameplay authoritative only in memory and do not introduce active-hand persistence or generic public game topics.

## Delivered

1. Registered versioned `doudizhu.command` and `doudizhu.hand.sync` handlers on the authenticated realtime hub.
2. Derived command identity exclusively from the authenticated WebSocket connection account.
3. Reused the Goal 0031 dispatcher and replay cache, giving HTTP and WSS identical command, authorization, version, and idempotency semantics.
4. Loaded persisted hand membership after accepted changes and delivered account-targeted events only to the three seated participants.
5. Generated a separate private snapshot for every participant and never published private cards through a generic topic.
6. Added reconnect synchronization with full private snapshots for stale clients and `notModified=true` for a current version.
7. Returned participant-authorized immutable terminal evidence when the live hand had already been removed.
8. Added stable realtime `INVALID_REQUEST`, `CONFLICT`, `FORBIDDEN`, `NOT_FOUND`, and `INTERNAL` error envelopes.
9. Preserved the existing bounded send queue, slow-consumer disconnect behavior, multiple connections per account, and shutdown ordering.
10. Proved that an account-delivery failure does not roll back or replay an already accepted command.
11. Added production bootstrap registration without changing game rules, active-memory authority, or terminal-only persistence.
12. Documented the realtime message, privacy, reconnect, idempotency, and process boundaries.

## Architecture boundary

- Private state is sent only with account-targeted delivery; no generic game topic exists.
- Active cards, bids, turns, passes, and versions remain memory-authoritative.
- Redis does not store active game state; MySQL stores durable setup data and immutable terminal archives.
- HTTP and WSS share one dispatcher and replay policy.
- Reconnect reads the latest authoritative snapshot or final evidence rather than replaying an event log.
- Broadcast failure cannot roll back an accepted game command.

## Verification

Focused run `30443668911` passed formatting, ordinary tests, race tests, integration-tag compilation, and `go vet`.

Final run `30443958893` passed for source commit `eaf79ea0c27f324aad046e038694ce741cd41f09`:

- module and generated-code cleanliness;
- repository formatting;
- Goal 0032 ordinary, race, and vet checks;
- all Go tests;
- Security, Admin, and realtime race suites;
- server build and local/production Compose validation;
- Secure Envelope and signed-manifest interoperability;
- Cocos policy and Admin Web build;
- complete MySQL 5.7 and Redis integration;
- production HTTP, authenticated WebSocket, HTTPS, WSS, bootstrap/login, and cleanup acceptance.

Final status: `goal0032/final-full: success`.

## Main-only workflow

All implementation, tests, documentation, verification setup, and cleanup were committed directly to `main`. No feature branch, validation branch, pull request, or merge workflow was used. The focused verifier removed itself after success. The final verifier was removed by `cc9a536fb5ce13ba713894f040d97be09ec2a490` after its locked run started.

## Completion Report

Goal 0032 is complete. The recommended maintainable server v1 now supports authenticated HTTP and WSS commands, participant-only live broadcasts, private latest-state reconnect, terminal evidence recovery, bounded realtime delivery, supervised lifecycle termination/timeouts, and independently verifiable immutable terminal archives.

Implementation commits include `3a747948bac8d12f9d3001365d8c81b60ee658d7`, `5a28773deb5badb9ee07965bf86f3b03f46a7430`, `07054fc760057043f9f99c054da3f456840b9f2e`, `2a074ec68776e5eea7fb96bc5e0566b00c4868a5`, `627db9d0a241f7e56a89af6e86b7c7b597a258f4`, and verified formatting/cleanup commit `4a1371bff2ffd6d3a6e3d6bbd9ee881e7da63974`.
