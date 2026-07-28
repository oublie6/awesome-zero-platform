# Goal 0026: Versioned Doudizhu Score Bidding and Landlord Transition

## Status

- State: completed
- Started: 2026-07-28
- Completed: 2026-07-28
- Blockers: None.

## Goal

Implement the first real Doudizhu gameplay phase on top of the verified in-memory hand runtime. Define one explicit and reproducible score-bidding ruleset, derive the first bidder from committed deterministic deal evidence, serialize bidding commands inside the live game, select the landlord, reveal and attach the three landlord cards, expose reconnect-safe public/private bidding views, and keep every bid and resulting card mutation in process memory. Do not add per-bid database writes, Redis card state, or Doudizhu semantics to `gamecore`.

## Scope

ChatGPT owns architecture, implementation, tests, failure diagnosis and fixes, documentation, repository verification, commits, and pushes directly to `main`.

Delivered:

1. Defined `doudizhu-score-bidding-v1` with exactly three fixed seats and scores `0`, `1`, `2`, and `3`:
   - `0` means pass;
   - a positive score must be strictly greater than the current highest score;
   - each seat acts at most once in circular order;
   - score `3` ends bidding immediately;
   - otherwise bidding ends after all three seats act;
   - the highest positive bidder becomes landlord;
   - three passes produce an explicit `NO_LANDLORD` live state that requires terminal coordination before a replacement hand may start.
2. Derived the first bidder deterministically and without modulo bias:
   - `seed = SHA-256(UTF8("doudizhu/bidding-first-seat/v1") || 0x00 || dealDigest[32])`;
   - initialize the existing `gamecore` HMAC counter stream with that seed;
   - use `gamecore.Uniform(stream, 3) + 1` as seat `1..3`.
3. Added a pure Doudizhu bidding state model with stable snapshots, explicit errors, copy isolation, and no dependency on application, database, Redis, transport, or clocks.
4. Added strict, versioned live bid command and result payloads. Unknown fields, trailing JSON, unsupported versions, invalid scores, wrong turns, stale live versions, repeated seats, and post-bidding commands fail closed.
5. Replaced the fixed `[3][17]` live-card representation with game-owned slices so the landlord can hold exactly 20 cards while the two farmers retain 17.
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
8. Added Doudizhu runtime/application methods that:
   - derive the actor's seat from trusted persisted Hand membership and the immutable live-game seat snapshot;
   - submit the bid through `gamecore.LiveDirectory.Apply`;
   - use only in-memory live versioning and serialization;
   - do not write command rows, aggregate snapshots, outbox events, or current cards for ordinary bids.
9. Extended public/private views for reconnect:
   - public: first/current bidder, highest score/bidder, ordered bid history, bidding completion, landlord seat, winning score, playing seat, and landlord cards only after selection;
   - private: the authenticated viewer's current 17 or 20 cards plus the public projection;
   - no view exposes another player's private cards, the server seed, or the complete deck.
10. Proved explicit abort after `NO_LANDLORD` remains retry-safe and produces one immutable final archive without replaying the final pass or abort logic.
11. Documented the exact bidding algorithm, command/view payloads, no-landlord policy, active-memory authority, deferred transport behavior, concurrency, and process-loss policy.
12. Added deterministic vectors, state-machine tests, malformed-payload tests, property tests, copy-isolation tests, concurrency/race tests, application authorization tests, archive interaction tests, formatting, full-repository, real MySQL/Redis, and production runtime verification.

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
- `docs/architecture/doudizhu-score-bidding-v1.md`
- `docs/architecture/fair-doudizhu-card-shuffle-v1.md`
- `server/business/gamecore/random.go`
- `server/business/gamecore/runtime.go`
- `server/business/doudizhu/domain/bidding`
- `server/business/doudizhu/domain/livehand`
- `server/business/doudizhu/application/hand_bidding.go`
- `server/business/doudizhu/infrastructure/runtime/live_hand_bidding.go`

## Constraints

- All edits and commits went directly to `main`; no branch or pull request was created.
- The active live game remains the only authority for bids, landlord selection, and current cards.
- No ordinary bid writes MySQL, Redis, outbox, command-result, or audit rows.
- Authenticated accounts map to seats through trusted server state; clients cannot choose `ActorPosition`.
- Expected live version is mandatory and exact.
- The fixed Goal 0023 deck, shuffle, deal, and transcript algorithms remain byte-for-byte compatible.
- Landlord cards are private before landlord selection and public afterwards.
- All-pass handling remains auditable; the same hand is not silently redealt.
- `gamecore` remains game-agnostic and standard-library-only.
- Memory-intensive tests ran sequentially with low Go test concurrency.

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
- Ordinary successful and rejected bids do not enter the database command transaction path and do not mutate persisted Hand, outbox, archive, or Redis state.
- Explicit abort after `NO_LANDLORD` archives once through the existing final-record path.
- Focused ordinary, `-race`, and `go vet` checks pass.
- Module tidy, generated code, formatting, all Go tests, Secure Envelope, Admin Web, MySQL/Redis integration, Compose, and production runtime checks remain green.

## Working State

### Completed

- Added the pure score-bidding state and deterministic first-bidder derivation.
- Integrated bidding into the concrete in-memory Doudizhu live game.
- Added strict versioned command/result payloads and reconnect-safe public/private views.
- Added 20-card landlord transition, public landlord-card disclosure, `PLAYING`, and first-playing-seat state.
- Added explicit `NO_LANDLORD` behavior without silent redeal.
- Added identity-scoped application/runtime bid routing with no ordinary database command transaction.
- Added same-hand serialization, separate-hand independence, malformed-command, copy-isolation, authorization, and terminal-archive tests.
- Added formal score-bidding architecture documentation.
- Completed focused and full repository verification, including two successful production runtime runs.
- Removed every temporary Goal 0026 verification workflow.

### In progress

- None.

### Remaining

- Deferred to later goals: public authenticated HTTP/WSS bidding transport, automatic no-landlord terminal coordination, legal play patterns, playing turns/passes, multipliers, scoring, settlement, replay UI, and active-game recovery.

## Completion Report

### Delivered implementation

- Goal start and explicit acceptance boundary: `57fc6c0f7012922e019bc44525cd6943d82382b2`.
- Pure bidding state and deterministic first bidder: `235f50ba8e4d4374372955e469bb9e5fd08ee49b`, `5e0aba9986a68c841cf1be9a2cacb23478c66051`, `4fa24d474382825043f0252ed71888e414366c37`, `34ab17deeb6969a6d4f6cd8865c7d38a53a1afad`.
- Concrete live-game bidding, landlord transition, views, and malformed-command handling: `ea31e8b475f74332a4f9db3617f3f556d1f5a2e2`, `2850b221250148f6d0364a4d10a778eb6f943891`.
- Application/runtime boundary and compatibility fixtures: `fb3af0931226db3dbdbe7fe0a7fa108823b5dd9d`, `5735be5a259100997d43edd3d10b4ee0248191f2`, `9322b2d8142ae903dfea4995a69b5ff3cc0c7bba`, `586a16c5357435bdc1efa9f911788f067a139492`, `4003660bdd42e1591079aefa20daefc507b110a6`, `c3377b9ef964dbe9051c2dbec2360db314e6102a`.
- Live-directory concurrency and no-landlord terminal archive evidence: `80a5fd16c33c59748877347a2b20f904fb52883a`, `7570868b8698bf6b9a96b4adf485090bddf99835`, `93095444eaf13806813e7b07440fb1dd5c0dfa0e`.
- Formal architecture: `4c95f4c5cf577b279313b6554bcafa0270ecabfb`, `73724150c6116a0754d5c7fe08d844e846fb8230`.

### Verification progression

Verification was intentionally split into small latest-main checks so interface and formatting faults could not accumulate:

- bidding core run `30372672529`: Go 1.25.8 formatting, ordinary tests, `-race`, and `go vet` succeeded;
- livehand bidding run `30373670956`: all Doudizhu domain tests, focused race, and domain vet succeeded;
- runtime/application run `30374371381`: formatting, ordinary tests, race, integration-tag MySQL-package compilation, and vet succeeded;
- final domain run `30374960652`: all Doudizhu domain tests, live bidding race, concurrency, no-landlord terminal archive, and vet succeeded.

A code review between the first and second focused runs identified that the bidding package should accept the dedicated `carddeck.DealDigest` type rather than the structurally identical generic digest type. This was corrected by `4fa24d474382825043f0252ed71888e414366c37` before the livehand verification; no formal Goal 0026 verification run failed.

### Final verification

A temporary read-only latest-main verifier locked commit `e951e24d8d7c67bbc84c00a837e5532eb5d3a96d` and ran the repository's actual verification commands.

Both final runs succeeded:

- `30375284834` — server, clients, full MySQL/Redis integration, and production runtime all succeeded;
- `30375439837` — repeated latest-main verification also succeeded and published `goal0026/final-full: success`.

The verified surface included:

- `go mod tidy` cleanliness;
- generated-code consistency and Go 1.25.8 formatting;
- Goal 0026 focused ordinary, race, and vet checks;
- all Go tests;
- Security and Admin race tests;
- server build and local/production Compose validation;
- Secure Envelope tests and TypeScript/Go interoperability;
- signed reveal-key manifest interoperability;
- Cocos deterministic-randomness policy;
- Admin Web build;
- complete MySQL 5.7 and Redis integration;
- production container startup, HTTP, authenticated WebSocket, HTTPS, WSS, administrator bootstrap/login, and cleanup.

No acceptance check remained unavailable.

### Main-only workflow and cleanup

All implementation, tests, documentation, verification setup, and cleanup commits were written directly to `main`. No feature branch, verification branch, or pull request was created.

The temporary final verifier was removed by `a0759e14e6ad3cbeda76b009a04e341f73237a1c` before its repeated run completed, preventing further triggers. Earlier focused verification workflows were also removed immediately after their successful runs. No Goal 0026 temporary workflow remains in the repository.

### Deferred work

Goal 0026 intentionally does not expose a public HTTP/WSS bid command, automatically terminate and replace an all-pass hand, or implement playing-card rules and settlement. These remain explicit follow-on goals and must preserve Goal 0023 deterministic bytes, Goal 0024 generic runtime semantics, and the Goal 0025 active-memory/terminal-archive boundary.
