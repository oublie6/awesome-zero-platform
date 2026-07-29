# Goal 0032: Authenticated Doudizhu WSS Synchronization and Reconnect

## Status

- State: completed
- Started: 2026-07-29
- Completed: 2026-07-29
- Blockers: None.

## Goal

Expose the Goal 0031 dispatcher through the existing authenticated WebSocket hub, deliver per-participant state changes without leaking private cards, and support reconnect by returning the latest private live snapshot or immutable terminal evidence. Keep active gameplay authoritative only in memory and do not introduce active-hand persistence or generic public game topics.

## References

- `AGENTS.md`
- `docs/goals/archive/0031-authenticated-doudizhu-http-api.md`
- `docs/architecture/doudizhu-final-evidence-v1.md`
- `docs/architecture/doudizhu-realtime-v1.md`
- `server/apps/app-api/internal/doudizhuapi/realtime.go`
- `server/platform/realtime`

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

## Constraints preserved

- All changes went directly to `main`; no branch or pull request was created.
- Clients cannot submit a trusted actor or seat.
- Private state is delivered only with account-targeted sends.
- Active cards and turns remain memory-authoritative; Redis and MySQL do not store active hand snapshots.
- Missed events are recovered through a latest-state sync, not event-log replay.
- Goal 0023 deterministic bytes, Goal 0028 settlement, Goal 0030 lifecycle ordering, and Goal 0031 HTTP contracts remain compatible.
- Public spectator topics, crash restoration, cross-instance migration, client UI, rankings, balances, prizes, and money-like value remain outside service v1.

## Acceptance Criteria

- Authenticated WSS commands execute as the connection account and return the shared dispatcher response.
- Strict request, replay, membership, and live-version checks fail closed.
- Accepted hand changes target the three participants and no outsider.
- Each participant receives only that participant's private snapshot.
- Old reconnect versions receive a full latest snapshot; current versions receive `notModified=true`.
- Completed or aborted reconnects receive immutable participant-authorized evidence.
- Broadcast errors do not change the accepted command result.
- Generic hub tests continue proving multi-connection account delivery and bounded slow-consumer handling.
- Focused and final verification are green.

## Working State

### Completed

- Realtime bridge, handlers, participant audience, bootstrap registration, private broadcasts, reconnect, terminal evidence recovery, stable errors, tests, and architecture documentation.
- Focused and full repository verification.
- Temporary focused and final verification workflows removed.

### In progress

- None.

### Remaining

- None within the recommended maintainable server v1.

### Verification status

Focused run `30443668911` succeeded:

- Go 1.25.8 formatting;
- Doudizhu realtime and bootstrap ordinary tests;
- Doudizhu and generic realtime race tests;
- integration-tag compilation;
- `go vet`.

Final run `30443958893` succeeded for source commit `eaf79ea0c27f324aad046e038694ce741cd41f09`:

- module and generated-code cleanliness;
- repository formatting;
- Goal 0032 ordinary, race, and vet checks;
- all Go tests;
- Security, Admin, and realtime race suites;
- server build and local/production Compose validation;
- Secure Envelope and signed-manifest interoperability;
- Cocos deterministic-randomness policy and Admin Web build;
- complete MySQL 5.7 and Redis integration;
- production HTTP, authenticated WebSocket, HTTPS, WSS, bootstrap/login, and cleanup acceptance.

Final status: `goal0032/final-full: success`.

## Completion Report

Goal 0032 is complete. The recommended maintainable server v1 now supports authenticated HTTP commands and queries, authenticated WSS commands, participant-only live broadcasts, private state synchronization, stale-version reconnect recovery, terminal evidence recovery, bounded realtime delivery, lifecycle termination and timeout supervision, and independently verifiable immutable final archives.

Implementation commits include `3a747948bac8d12f9d3001365d8c81b60ee658d7`, `5a28773deb5badb9ee07965bf86f3b03f46a7430`, `07054fc760057043f9f99c054da3f456840b9f2e`, `2a074ec68776e5eea7fb96bc5e0566b00c4868a5`, `627db9d0a241f7e56a89af6e86b7c7b597a258f4`, and the verified formatting/cleanup commit `4a1371bff2ffd6d3a6e3d6bbd9ee881e7da63974`. The temporary final verifier was removed by `cc9a536fb5ce13ba713894f040d97be09ec2a490` after its single locked run started.
