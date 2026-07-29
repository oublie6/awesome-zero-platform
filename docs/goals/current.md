# Goal 0030: Doudizhu Unified Termination and Timeout Supervision

## Status

- State: in_progress
- Started: 2026-07-29
- Completed: Not yet.
- Blockers: None.

## Goal

Unify all non-winning Doudizhu terminal paths around the existing database-first and immutable-final-archive ordering. Automatically terminate all-pass bidding, authorize participant cancellation, expire bidding and playing leases, and retry archive failures with the same terminal command while keeping active hands and deadlines in process memory.

## Scope

Deliver:

1. Stable reason codes for all-pass, participant cancellation, bidding timeout, and playing timeout.
2. A transport-independent lifecycle supervisor with injected clock, ID generator, Hand reader, and terminal service.
3. Automatic `NO_LANDLORD` abort after the final bid result.
4. Participant-only cancellation that uses an internal lifecycle actor after authorization.
5. Memory-only bidding and playing deadlines refreshed after successful live commands.
6. Deterministic due-hand sweep and an optional production ticker loop.
7. Retry with the exact same terminal command when database completion succeeded but final archive failed.
8. No per-bid, per-play, per-pass, current-card, or deadline persistence.
9. Focused ordinary, race, vet, full repository, real integration, build, Compose, and production runtime verification.

Outside this goal:

- public HTTP routes;
- WSS commands or broadcasts;
- active-hand crash restoration or cross-instance migration;
- client UI;
- rankings, balances, prizes, or money-like value.

## Acceptance Criteria

- Three passes produce one automatic `no_landlord` terminal command.
- A terminal archive failure keeps one pending command and retry reuses every command field.
- Only seated participants can request cancellation.
- Bidding and playing use different configurable deadlines.
- Successful live commands replace the appropriate deadline; completed play removes it.
- Due hands expire through the existing `ExpireHand` transaction and final archive path.
- Concurrent access is race-free and no external call occurs while the supervisor mutex is held.
- Full verification remains green.

## Working State

### Completed

- Goal 0029 terminal evidence verification and read side.

### In progress

- Lifecycle supervisor implementation and tests.

### Remaining

- Focused and final verification, completion report, and archive.

## Completion Report

Pending.
