# Goal 0026: Versioned Doudizhu Score Bidding and Landlord Transition

## Status

- State: in_progress
- Started: 2026-07-28
- Completed: Not yet.
- Blockers: None.

## Goal

Implement the first real Doudizhu gameplay phase on top of the verified in-memory hand runtime. Define one explicit and reproducible score-bidding ruleset, derive the first bidder from committed deterministic deal evidence, serialize bidding commands inside the live game, select the landlord, reveal and attach the three landlord cards, expose reconnect-safe public/private bidding views, and keep every bid and resulting card mutation in process memory. Do not add per-bid database writes, Redis card state, or Doudizhu semantics to `gamecore`.

## Scope

ChatGPT owns architecture, implementation, tests, failure diagnosis and fixes, documentation, repository verification, commits, and pushes directly to `main`.

Deliver:

1. Define `doudizhu-score-bidding-v1` with exactly three fixed seats and scores `0`, `1`, `2`, and `3`:
   - `0` means pass;
   - a positive score must be strictly greater than the current highest score;
   - each seat acts at most once in circular order;
   - score `3` ends bidding immediately;
   - otherwise bidding ends after all three seats act;
   - the highest positive bidder becomes landlord;
   - three passes produce an explicit `NO_LANDLORD` live state that requires terminal coordination before a replacement hand may start.
2. Derive the first bidder deterministically and without modulo bias:
   - `seed = SHA-256(UTF8("doudizhu/bidding-first-seat/v1") || 0x00 || dealDigest[32])`;
   - initialize the existing `gamecore` HMAC counter stream with that seed;
   - use `gamecore.Uniform(stream, 3) + 1` as seat `1..3`.
3. Add a pure Doudizhu bidding state model with stable snapshots, explicit errors, copy isolation, and no dependency on application, database, Redis, transport, or clocks.
4. Add a strict, versioned live bid command and result payload. Unknown fields, trailing JSON, unsupported versions, invalid scores, wrong turns, stale live versions, repeated seats, and post-bidding commands fail closed.
5. Replace the fixed `[3][17]` live-card representation with game-owned slices so the landlord can hold exactly 20 cards while the two farmers retain 17.
6. On successful landlord selection:
   - append the three existing landlord cards to the selected seat exactly once;
   - transition the live game from `BIDDING` to `PLAYING`;
   - set the landlord as the first playing seat;
   - expose landlord seat, winning score, bidding history, and the now-public landlord cards;
   - preserve the unchanged Goal 0023 deck, deal, and transcript bytes.
7. On three passes:
   - transition to `NO_LANDLORD` without silently reshuffling or discarding the deal;
   - reject further bidding/gameplay commands;
   - retain the live hand until an explicit existing terminal command archives the auditable setup and transcript;
   - return a versioned `requiresTermination` signal to the caller.
8. Add Doudizhu runtime/application methods that:
   - derive the actor's seat from trusted persisted Hand membership;
   - submit the bid through `gamecore.LiveDirectory.Apply`;
   - use only in-memory live versioning and serialization;
   - do not write command rows, aggregate snapshots, outbox events, or current cards for ordinary bids.
9. Extend public/private views for reconnect:
   - public: first/current bidder, highest score/bidder, ordered bid history, bidding completion, landlord seat, winning score, playing seat, and landlord cards only after selection;
   - private: the authenticated viewer's current 17 or 20 cards plus the public projection;
   - no view exposes another player's private cards, the server seed, or the complete deck.
10. Ensure explicit abort after `NO_LANDLORD` remains retry-safe and produces one immutable final archive without replaying the final pass or abort logic.
11. Document the exact bidding algorithm, command/view payloads, no-landlord policy, active-memory authority, and deferred transport behavior.
12. Add deterministic vectors, state-machine tests, malformed-payload tests, property tests, copy-isolation tests, concurrency/race tests, application authorization tests, archive interaction tests, formatting, full-repository, real MySQL/Redis, and production runtime verification.

The following remain outside this goal:

- the public HTTP/WSS gameplay command endpoint and durable network-command replay cache;
- automatic invocation of the terminal database command after `NO_LANDLORD`;
- doubling, bombs, spring rules, legal play-pattern recognition, turn comparison, passes during play, scoring, settlement, and replay UI;
- active-hand snapshots, Redis hand storage, process-crash restoration, live migration, or cross-instance ownership transfer;
- changing Goal 0023 card/deal/transcript bytes or Goal 0024 generic runtime semantics.

## References

- `AGENTS.md`
- `docs/goals/archive/0025-verified-doudizhu-live-hand-runtime.md`
- `docs/requirements/fair-doudizhu-v1.md`
- `docs/architecture/gamecore-v1.md`
- `docs/architecture/doudizhu-live-hand-runtime-v1.md`
- `docs/architecture/fair-doudizhu-card-shuffle-v1.md`
- `server/business/gamecore/random.go`
- `server/business/gamecore/runtime.go`
- `server/business/doudizhu/domain/livehand`
- `server/business/doudizhu/infrastructure/runtime`

## Constraints

- All edits and commits go directly to `main`; do not create a branch or pull request.
- The active live game remains the only authority for bids, landlord selection, and current cards.
- No ordinary bid writes MySQL, Redis, outbox, command-result, or audit rows.
- Authenticated accounts map to seats through trusted server state; clients cannot choose `ActorPosition`.
- Expected live version is mandatory and exact.
- The fixed Goal 0023 deck, shuffle, deal, and transcript algorithms remain byte-for-byte compatible.
- Landlord cards are private before landlord selection and public afterwards.
- All-pass handling must remain auditable; do not silently redeal inside the same hand.
- `gamecore` remains game-agnostic and standard-library-only.
- Run memory-intensive tests sequentially with low Go test concurrency.

## Acceptance Criteria

- The first bidder is deterministic for a fixed deal digest, uses rejection sampling, is always seat `1..3`, and has a committed golden vector.
- Bidding accepts only the current seat and only `0..3`; positive bids strictly increase.
- Every seat acts at most once and history order is stable.
- Score `3` immediately selects the bidder as landlord.
- After three actions, the highest positive bidder becomes landlord.
- After three passes, phase is `NO_LANDLORD`, `requiresTermination` is true, and no landlord cards are revealed.
- Landlord selection appends exactly the original three landlord cards once, producing hand sizes `20/17/17` in the proper seat order.
- The landlord is the first playing seat.
- Public views reveal no private hand cards and reveal landlord cards only after selection.
- Private views contain exactly the authenticated viewer's cards and are copy-isolated.
- Stale expected versions, malformed payloads, unsupported versions, unknown fields, trailing JSON, wrong turns, repeated actions, and post-bidding bids are rejected without mutation.
- Concurrent submissions for one hand are serialized; separate hands remain independent.
- Application bid submission rejects non-seated actors and never trusts a caller-supplied seat.
- Ordinary successful and rejected bids leave database command, Hand snapshot, outbox, archive, and Redis state unchanged.
- Explicit abort after `NO_LANDLORD` archives once and releases the seed only after archive success.
- Focused ordinary, `-race`, and `go vet` checks pass.
- Module tidy, generated code, formatting, all Go tests, Secure Envelope, Admin Web, MySQL/Redis integration, Compose, and production runtime checks remain green.

## Working State

### Completed

- Goal 0025 was completed, fully verified, cleaned up, and archived.
- The score-bidding rules and no-landlord lifecycle boundary are now explicit.

### In progress

- Implementing the pure bidding state and deterministic first-bidder derivation.

### Remaining

- Integrate bidding into the live game, runtime/application boundary, views, tests, documentation, and final verification.

## Completion Report

Pending.
