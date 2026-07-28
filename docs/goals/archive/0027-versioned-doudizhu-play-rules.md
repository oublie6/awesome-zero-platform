# Goal 0027: Versioned Doudizhu Play Rules and In-Memory Turn Engine

## Status

- State: completed
- Started: 2026-07-29
- Completed: 2026-07-29
- Blockers: None.

## Goal

Implement the complete versioned Doudizhu playing phase on top of the verified in-memory live hand. Recognize and compare legal card patterns, validate ownership, serialize play and pass commands, advance turns, detect the winning seat, and expose reconnect-safe public/private views without writing ordinary gameplay state to MySQL or Redis.

## Delivered

1. Defined `doudizhu-play-rules-v1` with:
   - single, pair, triple;
   - triple with single and triple with pair;
   - straight and consecutive pairs;
   - airplane, airplane with single wings, and airplane with pair wings;
   - four with two singles and four with two pairs;
   - bomb and rocket.
2. Fixed all sequence and attachment ambiguities, including exclusion of `2` and jokers from sequence bodies, repeated physical single wings, distinct pair wings, and explicit four-with-two behavior.
3. Added strict physical-card and pattern validation:
   - invalid cards and repeated physical cards fail;
   - fabricated pattern values with inconsistent kind, count, sequence length, or rank fail;
   - rocket, bomb, ordinary-kind, and equal-structure comparison is deterministic.
4. Added a pure `playing.State` that owns current seat, leading play, pass count, public history, gameplay completion, and winner.
5. Added strict versioned play and pass commands to `livehand.Game`.
6. Added exact live-version, turn, physical-card ownership, legal-pattern, and leader-beating validation.
7. Added copy-isolated public/private reconnect views with remaining counts, leading pattern, ordered action history, and winner.
8. Added two-pass trick reset and initiative return to the previous leading seat.
9. Added `GAMEPLAY_COMPLETE` when a player empties their hand; settlement remains Goal 0028.
10. Routed authenticated play/pass application calls through `gamecore.LiveDirectory.Apply` without ordinary MySQL, Redis, outbox, Hand-snapshot, command-result, or archive writes.
11. Proved same-hand serialization and separate-hand independence under concurrency.
12. Added `docs/architecture/doudizhu-play-rules-v1.md` and linked it from the architecture overview.

## Constraints Preserved

- All edits and commits went directly to `main`; no branch or pull request was created.
- Active cards, turns, plays, and passes remain process-memory authority.
- Clients cannot choose an authoritative seat.
- Goal 0023 shuffle/deal/transcript bytes are unchanged.
- Goal 0024 `gamecore` semantics remain game-agnostic.
- Goal 0025 terminal archive behavior and Goal 0026 bidding behavior remain compatible.

## Acceptance Results

- Every listed pattern has positive and negative deterministic tests.
- Duplicate, invalid, non-held, malformed, stale, wrong-turn, non-beating, and post-completion commands fail without mutation.
- Two passes reset the trick and return initiative correctly.
- Winning play empties the exact hand, records the winner, and stops gameplay commands.
- Same-hand concurrent commands serialize; separate hands remain independent.
- Application tests prove ordinary play/pass uses trusted membership and the live runtime only.
- Full repository and production runtime verification passed.

## Completion Report

### Implementation commits

- Goal boundary: `ce98099a`.
- Pattern analyzer and comparison: `ba15c5b3`, `72ccbd1e`, `9b7635d1`.
- Pure turn state: `d6ca4456`, `2a0877c5`, `44b9dafa`.
- Concrete livehand integration: `e0d85089`.
- Application/runtime routing and compatibility fixtures: `e9f30938`, `17756d87`, `5978c6b7`, `7dee2ac5`, `b6b3cd25`, `e99d381a`, `fc3b04aa`.
- Concurrency and strict pattern-value validation: `89a52e72`, `d524d398`.
- Architecture documentation: `d7487afd`, `181e8918`.

### Focused verification

- `30378393088` — pattern formatting, ordinary tests, race, and vet succeeded.
- `30378748668` — turn-state formatting, ordinary tests, race, and vet succeeded.
- `30379210039` — livehand play/pass integration, ordinary tests, race, and vet succeeded.
- `30379791708` — application/runtime tests, integration-tag compilation, race, and vet succeeded.
- `30380384125` — all Doudizhu domain tests, concurrency, race, and vet succeeded.

### Final verification

Run `30380620548` locked commit `6854af796de6d2113ec7c346ef37fdfb0bd9259d` and succeeded across:

- module tidy, generated-code consistency, and Go 1.25.8 formatting;
- Goal 0027 focused ordinary, race, and vet checks;
- all Go tests and Security/Admin race tests;
- server build and local/production Compose validation;
- Secure Envelope and signed-manifest TypeScript/Go interoperability;
- Cocos deterministic-randomness policy and Admin Web build;
- full MySQL 5.7 and Redis integration;
- production container startup, HTTP, authenticated WebSocket, HTTPS, WSS, administrator bootstrap/login, and cleanup.

No acceptance check remained unavailable.

### Cleanup

All temporary Goal 0027 scripts and focused workflows were removed after their successful runs. The temporary final verifier was removed by `753e114274477394e83e9896ee2e15e7e6c2c1b8`, preventing additional triggers. No Goal 0027 temporary workflow remains in the repository.

### Deferred to Goal 0028 and later

- bomb, rocket, spring, and anti-spring multipliers;
- landlord/farmer settlement and `SETTLING -> COMPLETED` orchestration;
- normal terminal archive with complete play and settlement evidence;
- public HTTP/WSS gameplay transport and Cocos screens;
- active-hand crash recovery or cross-instance migration.
