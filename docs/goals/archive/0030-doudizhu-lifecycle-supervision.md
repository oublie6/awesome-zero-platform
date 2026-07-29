# Goal 0030: Doudizhu Unified Termination and Timeout Supervision

## Status

- State: completed
- Started: 2026-07-29
- Completed: 2026-07-29
- Blockers: None.

## Goal

Unify all non-winning Doudizhu terminal paths around the existing database-first and immutable-final-archive ordering. Automatically terminate all-pass bidding, authorize participant cancellation, expire bidding and playing leases, and retry archive failures with the same terminal command while keeping active hands and deadlines in process memory.

## Delivered

1. Added stable terminal reasons: `no_landlord`, `participant_cancelled`, `bidding_timeout`, and `playing_timeout`.
2. Added a transport-independent lifecycle supervisor with injected Hand reader, terminal service, clock, ID generator, and configuration.
3. Converted a successful all-pass bid result into one automatic terminal command instead of leaving the live hand indefinitely in `NO_LANDLORD`.
4. Added participant-only cancellation; the external participant is authorized first and the persisted terminal command uses the fixed internal actor `system:doudizhu-lifecycle`.
5. Added separate memory-only bidding and playing leases, deadline refresh after successful commands, completion cleanup, deterministic due-hand ordering, and a production ticker loop.
6. Reused the existing `AbortHand` and `ExpireHand` database-first terminal ordering and the existing immutable final archive.
7. Preserved the exact pending command ID, name, actor, sequence, expected version, timestamps, and reason when final archive fails; retry does not repeat the final bid, play, pass, or terminal mutation.
8. Kept all active cards, turns, deadlines, and pending supervisor state out of MySQL and Redis.
9. Added strict result decoding, outsider rejection, timeout, retry, completion cleanup, concurrency, ordinary, race, and vet coverage.
10. Documented process-loss, locking, terminal ordering, and future recovery boundaries.

## Acceptance

- Focused run `30431704083` passed Go 1.25.8 formatting, ordinary tests, `-race`, and `go vet`.
- Final run `30431952503` passed:
  - module and formatting checks;
  - Goal 0030 and full repository Go tests;
  - client builds and Secure Envelope tests;
  - real MySQL 5.7 and Redis integration;
  - server build and Compose validation;
  - production HTTP, authenticated WebSocket, HTTPS, WSS, bootstrap/login, and cleanup acceptance.
- Final status: `goal0030/final-full: success`.

## Main-only workflow

All code, tests, documentation, validation setup, and cleanup were committed directly to `main`. No feature branch, validation branch, or pull request was created. Temporary focused and final workflows were removed after their successful runs.

## Completion Report

Goal 0030 is complete. The service now has one retry-safe lifecycle boundary for all-pass, participant cancellation, bidding timeout, and playing timeout while preserving in-memory gameplay authority and terminal-only persistence.
