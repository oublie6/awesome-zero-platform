# Goal 0024: Extensible Game Core and Versioned Module Boundary

## Status

- State: in_progress
- Started: 2026-07-28
- Completed: Not yet.
- Blockers: None.

## Goal

Extract a small reusable business-level game core from the verified Doudizhu fairness implementation so the platform can support additional games without coupling common runtime contracts to Doudizhu cards, three seats, landlord dealing, bidding, scoring, or client UI. Establish the agreed lifecycle in which an active game instance is authoritative in process memory, commands are serialized per instance, and only completed or explicitly aborted final records are passed to persistence. Preserve every Goal 0023 Doudizhu v1 canonical byte and golden-vector output.

## Scope

ChatGPT owns architecture, implementation, tests, failure diagnosis and fixes, documentation, repository verification, commits, and pushes directly to `main`.

Deliver:

1. Create `server/business/gamecore` as a reusable business module, not a platform or technical-foundation package.
2. Define validated, versioned game identity values:
   - `GameID`;
   - `RulesetVersion`;
   - `ModuleVersion`;
   - `FairnessSuiteID`;
   - participant count.
3. Define an immutable game `Descriptor` and an explicit compile-time registry with exact descriptor lookup and duplicate-registration rejection.
4. Define a narrow `RandomizedSetupModule` boundary for generating and verifying a game-owned, versioned setup artifact from common fairness material.
5. Define a common fairness-material envelope that binds the exact game descriptor, match/hand ID, server seed and commitment, ordered participant contribution digests and commitments, beacon context, and reveal-key audit metadata.
6. Extract or wrap only genuinely reusable deterministic primitives:
   - HMAC-SHA256 counter byte stream;
   - unbiased bounded sampling with rejection;
   - deterministic index permutation.
7. Keep canonical item definitions, deck/tile construction, dealing/wall/board construction, game phases, legal actions, scoring, settlement, and transcript payload semantics inside concrete game modules.
8. Add a Doudizhu randomized-setup adapter that implements the gamecore contract using the existing Card/Deck v1 implementation.
9. Preserve the Goal 0023 Doudizhu golden vector byte-for-byte, including commitments, first 64 random bytes, shuffled deck, hands, landlord cards, and all digests.
10. Add a test-only non-Doudizhu module or sequence with a different participant count and item count to prove the common core contains no three-seat or 54-card assumption.
11. Add architecture and API documentation for descriptor selection, registration, dependency direction, artifact ownership, compatibility, and deferred runtime integration.
12. Define a small game-owned `LiveGame` contract whose concrete implementation owns private state such as hands, tile walls, hidden boards, or other gameplay state; `gamecore` treats commands, views, and final records as opaque immutable byte payloads.
13. Implement an in-memory live-game directory that:
   - is the only runtime authority for active instances;
   - serializes commands per game instance;
   - exposes public/private views only through the concrete game implementation;
   - does not persist on every command;
   - removes an instance only after a completed or aborted record has been accepted by an archive port.
14. Define a narrow `FinalRecordArchive` port. The runtime calls it exactly once for a terminal completion or explicit abort and never for an ordinary in-progress command.
15. Document and test the failure policy: an unexpected process loss may abort active games in this version; no snapshot/event-log recovery is promised, Redis is not a hand store, and future recovery can be added behind explicit ports without changing game-specific state ownership.
16. Add focused unit, property, race, vet, import-boundary, golden-compatibility, live-runtime concurrency/lifecycle, and full-repository verification.

The following remain outside this goal:

- integrating randomized setup or the generic live directory into the current Doudizhu Room/Hand application command flow;
- implementing Doudizhu bidding, play patterns, turns, scoring, settlement, or replay required for a production `LiveGame`;
- generating, encrypting, persisting, or disclosing production server seeds;
- public-beacon network adapters and proof validation;
- generic lobby, matchmaking, room, table, match-session, spectator, or cross-instance routing capabilities;
- database schema changes, in-progress game snapshots, command event-log recovery, or generic JSON persistence for every game;
- HTTP/WSS contracts, Cocos/client module loading, game catalog UI, or downloadable game plugins;
- Mahjong, poker, dice, board-game, or other production game implementations;
- dynamic shared libraries, reflection-based plugin discovery, scripting engines, a rule DSL, or runtime code downloads.

## References

- `AGENTS.md`
- `docs/architecture/overview.md`
- `docs/architecture/extensible-game-runtime.md`
- `docs/architecture/fair-doudizhu-card-shuffle-v1.md`
- `docs/architecture/fair-doudizhu-domain.md`
- `docs/requirements/fair-doudizhu-v1.md`
- `server/business/doudizhu/domain/carddeck`
- `server/business/doudizhu/domain/carddeck/testdata/golden-v1.json`
- `.github/workflows/doudizhu.yml`

## Constraints

- Follow `AGENTS.md` and commit directly to `main`.
- `gamecore` is reusable business code under `server/business`; it must not be placed under `foundation` or `platform`.
- Dependency direction must be `concrete game -> gamecore`. `gamecore` must never import `doudizhu` or another concrete game.
- Do not rename the existing Doudizhu Room and Hand aggregates into generic aggregates in this goal.
- Do not create a universal card, tile, piece, board, action, event, state-machine, scoring, or persistence model.
- The common randomized-setup artifact must be an immutable versioned envelope whose canonical payload remains owned by the concrete module.
- Module registration is compile-time and explicit. Do not use dynamic plugins, reflection-based discovery, global init registration, or runtime source loading.
- Common deterministic primitives must use only the Go standard library and must not depend on time, global entropy, `math/rand`, floating point, map iteration, platform endianness, database, HTTP, Redis, or framework state.
- Existing Doudizhu v1 domains, canonical encodings, and test vectors are immutable. Incompatible changes require a new explicit version.
- Active game state remains owned by the concrete in-memory game instance. `gamecore` must not inspect or persist game-specific private state.
- The live directory must use per-instance serialization rather than one process-wide gameplay lock.
- Final archival is at-most-once from the runtime's perspective; an archive implementation is responsible for idempotency by game instance ID.
- Do not add production fake games. A non-Doudizhu implementation may exist only in tests to prove the extension boundary.
- Do not modify the MySQL schema or add migration files.
- Do not add frontend production code.
- Run memory-intensive verification sequentially with low-concurrency Go commands.

## Acceptance Criteria

- `server/business/gamecore` exists and imports only the Go standard library.
- `gamecore` contains no `doudizhu`, card rank, suit, joker, landlord, bid, play, score, or three-seat constant.
- Valid descriptors round-trip and invalid/empty/oversized identifiers, unsupported participant counts, and contradictory versions are rejected.
- The registry supports exact lookup by game ID and ruleset/module version and rejects duplicate descriptor keys.
- Registry results are immutable and deterministic; registration order does not alter exact lookup behavior.
- The randomized-setup module contract passes common fairness material to a concrete module without exposing SQL, HTTP, Redis, clocks, loggers, or global process state.
- The common fairness envelope validates the participant contribution count and exact ordered participant positions against the descriptor.
- Generic deterministic stream vectors are stable.
- Generic bounded sampling rejects zero bounds and proves rejection of biased candidates.
- Generic permutation produces valid permutations for multiple item counts, including counts other than 54.
- The Doudizhu adapter reproduces the Goal 0023 golden vector byte-for-byte.
- The existing Doudizhu package retains ownership of Card, Deck, DealResult, transcript payload, and every Doudizhu-specific algorithm version.
- A test-only non-Doudizhu module with a different participant count and item count registers, generates an artifact, and verifies it without importing Doudizhu.
- Tests reject descriptor mismatch, participant-order mismatch, altered fairness material, unsupported artifact versions, duplicate registration, and module/artifact identity mismatch.
- The live directory rejects duplicate instance IDs and descriptor mismatches, routes commands and views to the correct game-owned instance, and keeps active state exclusively in memory.
- Concurrent commands for one instance are serialized; commands for different instances are not forced through one process-wide gameplay lock.
- Ordinary commands never call the final archive. A terminal command archives once and removes the instance only after archive success; archive failure leaves the terminal instance available for retry without reapplying the command.
- Explicit abort archives one final aborted record and removes the instance only after archive success; repeated completion/abort attempts cannot create duplicate runtime archival calls.
- Public/private views are copies returned by the concrete game and the runtime never exposes another participant's private state by inspecting payloads.
- There is no database, Redis, HTTP, WebSocket, logger, clock, or framework dependency in `gamecore`.
- Import-boundary tests or static inspection prove `gamecore` does not import any concrete game and concrete modules do not import one another.
- Architecture documentation clearly distinguishes reusable gamecore, concrete modules, in-memory live authority, terminal archive persistence, future recovery/routing capabilities, and client modules.
- `go test -count=1 -p 1 -parallel 1 ./business/gamecore/... ./business/doudizhu/domain/...` succeeds.
- `go test -race -count=1 -p 1 -parallel 1 ./business/gamecore/... ./business/doudizhu/domain/...` succeeds.
- `go vet ./business/gamecore/... ./business/doudizhu/domain/...` succeeds.
- Existing generated-code, formatting, full Go unit, Secure Envelope, Admin Web, MySQL 5.7/Redis integration, Compose, HTTP/WS, HTTPS/WSS, and production runtime verification remain green.
- Final evidence records exact commits, real failures and fixes, CI run IDs, unavailable checks, and deferred work.

## Working State

### Completed

- Goal 0023 was completed, verified, cleaned up, and archived.
- The extension requirement was converted into `docs/architecture/extensible-game-runtime.md`.
- The architecture fixes a one-way dependency from concrete games to `gamecore`, an explicit compile-time registry, game-owned setup artifacts, and no universal rule framework.
- The active-game storage decision is fixed: concrete live instances are authoritative in memory; only completed or explicitly aborted final records cross the archive port.

### In progress

- Implementing `server/business/gamecore`, the Doudizhu setup adapter, and the in-memory live-game lifecycle while preserving all Doudizhu v1 outputs.

### Remaining

- Implement descriptors, registry, common fairness primitives, randomized-setup contract, Doudizhu adapter, test-only alternate module, live directory, terminal archive contract, documentation, and full verification.

### Verification status

- Goal 0023 final main CI `30342057452`, Fair Doudizhu `30342060323`, and runtime acceptance `30342370234` succeeded.
- Goal 0024 implementation verification pending.

## Completion Report

Pending.
