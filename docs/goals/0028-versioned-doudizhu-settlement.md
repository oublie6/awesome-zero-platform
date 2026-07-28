# Goal 0028: Versioned Doudizhu Multipliers, Settlement, and Normal Completion

## Status

- State: in_progress
- Started: 2026-07-29
- Completed: Not yet.
- Blockers: None.

## Goal

Complete a normal Doudizhu hand after Goal 0027 detects the winning seat. Define deterministic non-monetary integer scoring, bomb/rocket and spring multipliers, produce a zero-sum settlement, atomically return a completed final payload from the winning play, and use the existing `gamecore.LiveDirectory` retry-safe archive path without replaying the winning command.

## Scope

ChatGPT owns architecture, implementation, tests, failure diagnosis and fixes, documentation, verification, commits, and pushes directly to `main`.

Deliver:

1. Define `doudizhu-settlement-v1` as a social-game integer-points ruleset with no money, recharge, cash-out, prize, or transferable value.
2. Use the winning bid `1`, `2`, or `3` as the base stake.
3. Apply one doubling for every accepted bomb play and one doubling for the accepted rocket play.
4. Apply one spring doubling when the landlord wins and neither farmer made an accepted play.
5. Apply one anti-spring doubling when a farmer wins and the landlord made exactly one accepted play.
6. Do not add voluntary player doubling in v1.
7. Compute `stake = winningBid × 2^(bombs + rockets + springOrAntiSpring)`.
8. Produce zero-sum seat points:
   - landlord win: landlord `+2 × stake`, each farmer `-stake`;
   - farmers win: landlord `-2 × stake`, each farmer `+stake`.
9. Strictly validate landlord, winner, winning bid, completed playing snapshot, action history, bomb/rocket bounds, and overflow.
10. Integrate settlement into the winning live play without an extra client command.
11. Return a versioned completed result and immutable completed final payload containing fairness evidence, original setup, bidding, full accepted play/pass history, final hands, multiplier breakdown, and settlement points.
12. Mark the winning `gamecore.CommandOutcome` terminal so `LiveDirectory.Apply` archives it as `completed` before releasing the in-memory hand.
13. Preserve retry safety:
   - archive failure keeps the exact pending final record;
   - the winning play is not replayed;
   - retry submits the same final-record digest;
   - archive success removes the live hand exactly once.
14. Keep non-winning play and pass commands memory-only with no archive or database gameplay writes.
15. Add reconnect/result projections and formal documentation.
16. Run focused formatting, ordinary tests, `-race`, `go vet`, full repository, real MySQL/Redis, build, Compose, and production runtime verification.

Outside this goal:

- public HTTP/WSS gameplay transport and durable network-command replay cache;
- automatic all-pass replacement-hand orchestration, cancellation/expiration policy, and player-facing verification queries;
- Cocos gameplay and settlement screens;
- persistent active-hand snapshots, Redis card state, crash restoration, or cross-instance migration;
- rankings, balances, prizes, or any money-like value.

## Constraints

- All edits and commits go directly to `main`; do not create a branch or pull request.
- Settlement values are non-monetary integer points only.
- Active gameplay remains process-memory authority until the immutable final record is archived.
- The final archive must be deterministic and idempotent for one completed live version.
- A failed archive must not execute the winning play or settlement calculation a second time.
- `gamecore` remains game-agnostic and unchanged unless a real generic defect is proven.
- Goal 0023 shuffle/deal/transcript bytes and Goal 0026/0027 command semantics remain compatible.

## Acceptance Criteria

- Landlord and farmer wins produce exact zero-sum points.
- Bomb, rocket, spring, and anti-spring multipliers are deterministic and independently tested.
- Invalid or incomplete snapshots and arithmetic overflow are rejected.
- A winning play returns `Terminal=true`, a completed result, and one immutable final payload.
- The final payload contains enough information to verify setup, bidding, play history, winner, multiplier, and points.
- Successful archive removes the hand and releases protected runtime material.
- Failed archive leaves one pending record; `RetryArchive` succeeds without replaying the play and preserves the digest.
- Non-winning play/pass does not archive.
- Focused and full verification remain green.

## Working State

### Completed

- Goal 0027 play rules and in-memory winner detection are completed and archived.
- Goal 0028 settlement rules and terminal boundary are explicit.

### In progress

- Implementing the pure versioned multiplier and zero-sum settlement calculator.

### Remaining

- Integrate normal completion into `livehand`, final payloads, runtime/application behavior, tests, documentation, and final verification.

## Completion Report

Pending.
