# Goal 0027: Versioned Doudizhu Play Rules and In-Memory Turn Engine

## Status

- State: in_progress
- Started: 2026-07-29
- Completed: Not yet.
- Blockers: None.

## Goal

Implement the complete versioned Doudizhu playing phase on top of the verified in-memory live hand. Recognize and compare legal card patterns, validate ownership, serialize play and pass commands, advance turns, detect the winning seat, and expose reconnect-safe public/private views without writing ordinary gameplay state to MySQL or Redis.

## Scope

ChatGPT owns architecture, implementation, tests, failure diagnosis and fixes, documentation, verification, commits, and pushes directly to `main`.

Deliver:

1. Define `doudizhu-play-rules-v1` with explicit legal patterns:
   - single, pair, triple;
   - triple with single and triple with pair;
   - straight and consecutive pairs;
   - airplane, airplane with single wings, and airplane with pair wings;
   - four with two singles and four with two pairs;
   - bomb and rocket.
2. Define exact v1 ambiguities:
   - straights, consecutive pairs, and airplane bodies cannot contain rank `2` or jokers;
   - airplane body ranks are consecutive and each contributes exactly three body cards;
   - airplane wings cannot use a body rank;
   - single wings may repeat a non-body rank when the physical cards exist;
   - pair wings must be distinct pairs;
   - four-with-two-singles permits the two attachments to form a pair;
   - four-with-two-pairs requires two distinct pairs.
3. Add pure pattern analysis with invalid-card, duplicate-card, malformed-pattern, and copy-isolation protection.
4. Add deterministic comparison:
   - rocket beats every non-rocket pattern;
   - bombs beat every non-bomb, non-rocket pattern;
   - ordinary patterns compare only when kind and structure length match;
   - the higher main rank wins.
5. Add strict versioned play and pass commands for the concrete live game.
6. Require the server-resolved current playing seat and exact expected live version.
7. Validate that every submitted physical card is currently held by the actor, then remove each exactly once.
8. Track the leading play, leading seat, current seat, pass count, ordered public play history, and remaining card counts.
9. Reject pass when there is no outstanding leading play and clear the trick only after both opponents pass.
10. When a player empties their hand, enter an explicit gameplay-complete state with the winning seat; settlement remains Goal 0028.
11. Keep ordinary play/pass state exclusively in the live in-memory game and route application calls through `gamecore.LiveDirectory.Apply`.
12. Preserve Goal 0023 shuffle/deal/transcript bytes, Goal 0024 generic runtime semantics, Goal 0025 terminal archive behavior, and Goal 0026 bidding state.
13. Document the exact rules, command payloads, views, concurrency, and deferred transport behavior.
14. Run focused formatting, ordinary tests, `-race`, `go vet`, full repository, real MySQL/Redis, build, Compose, and production runtime verification.

Outside this goal:

- bombs, rocket, spring, and other score multipliers;
- settlement amounts and `SETTLING -> COMPLETED` orchestration;
- public HTTP/WSS gameplay transport and durable network-command replay cache;
- active-hand persistence, Redis card state, crash restoration, live migration, or cross-instance ownership transfer;
- Cocos gameplay screens.

## Constraints

- All edits and commits go directly to `main`; do not create a branch or pull request.
- Active cards, turns, plays, and passes remain process-memory authority.
- Ordinary gameplay must not write database command rows, Hand snapshots, outbox rows, archives, or Redis state.
- Clients cannot choose an authoritative seat; application/runtime code derives it from trusted membership.
- Expected live version is mandatory and exact.
- Rejected commands must leave all live state unchanged.
- `gamecore` remains game-agnostic and standard-library-only.

## Acceptance Criteria

- Every listed pattern has positive and negative deterministic tests.
- Duplicate or invalid physical cards are rejected.
- Same-type comparison, length mismatch, bomb override, and rocket override are tested.
- Play commands reject wrong turn, stale version, malformed JSON, unsupported versions, cards not held, illegal patterns, and plays that do not beat the leader without mutation.
- Pass commands reject the leading player and an empty trick without mutation.
- Two passes reset the trick and return initiative to the prior leading seat.
- A legal play removes exact cards, updates remaining counts, and advances the turn.
- Emptying a hand records the winner and stops further gameplay commands.
- Same-hand concurrent commands serialize; separate hands remain independent.
- Ordinary gameplay does not enter the database transaction path.
- Focused and full verification remain green.

## Working State

### Completed

- Goal 0026 score bidding and landlord transition is completed and archived.
- Goal 0027 rules and lifecycle boundary are explicit.

### In progress

- Implementing the pure versioned card-pattern analyzer and comparator.

### Remaining

- Integrate the pure rules into `livehand`, runtime/application boundaries, views, tests, documentation, and final verification.

## Completion Report

Pending.
