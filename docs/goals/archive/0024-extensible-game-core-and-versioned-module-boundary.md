# Goal 0024: Extensible Game Core and Versioned Module Boundary

## Status

- State: completed
- Started: 2026-07-28
- Completed: 2026-07-28
- Blockers: None.

## Goal

Extract a small reusable business-level game core from the verified Doudizhu fairness implementation so the platform can support additional games without coupling common runtime contracts to Doudizhu cards, three seats, landlord dealing, bidding, scoring, or client UI. Establish the agreed lifecycle in which an active game instance is authoritative in process memory, commands are serialized per instance, and only completed or explicitly aborted final records are passed to persistence. Preserve every Goal 0023 Doudizhu v1 canonical byte and golden-vector output.

## Scope

ChatGPT owned architecture, implementation, tests, failure diagnosis and fixes, documentation, repository verification, commits, and pushes directly to `main`.

Delivered:

1. Created `server/business/gamecore` as reusable business code under `server/business`, independent of platform and technical-foundation packages.
2. Added validated immutable `GameID`, `RulesetVersion`, `ModuleVersion`, `FairnessSuiteID`, `ArtifactVersion`, `InstanceID`, `Descriptor`, and exact `DescriptorKey` values.
3. Added an explicit compile-time `RandomizedSetupModule` registry with duplicate rejection, exact lookup, fixed registered identity, and cloned mutable inputs.
4. Added a common fairness-material envelope binding the exact descriptor, instance ID, server seed and commitment, ordered participant contribution digests and commitments, beacon evidence, and reveal-key audit metadata.
5. Added standard-library-only deterministic primitives:
   - HMAC-SHA256 counter byte stream;
   - unbiased bounded sampling with rejection;
   - Fisher–Yates index permutation for game-selected item counts.
6. Added an immutable versioned `SetupArtifact` envelope whose canonical payload remains owned by the concrete game module.
7. Added `server/business/doudizhu/domain/randomizedsetup`, which adapts the existing Card/Deck v1 shuffle and deal implementation to `gamecore` without changing any Goal 0023 algorithm or canonical bytes.
8. Added a test-only non-Doudizhu sequence module with four participants and eleven items, proving the common core has no three-seat or 54-card assumption.
9. Added the game-owned `LiveGame` contract for opaque commands, public/private views, completion records, and abort records.
10. Added an in-memory `LiveDirectory` that:
    - is authoritative for active instances;
    - stores a fixed descriptor for each registered instance;
    - serializes commands per instance;
    - allows different instances to execute concurrently;
    - never persists ordinary in-progress commands;
    - rejects commands while finalization is pending;
    - removes an instance only after final archive success.
11. Added immutable completed/aborted `FinalRecord` envelopes and the narrow `FinalRecordArchive` port.
12. Added exact retry semantics: a terminal command or abort creates one logical final record and invokes the game once; archive delivery may retry the same record, so the archive adapter must be idempotent by instance ID and digest.
13. Documented the v1 failure policy: unexpected process loss may void active games; Redis is not a hand store; snapshots, action-log replay, live migration, cross-instance routing, and database adapters remain future explicit capabilities.
14. Updated the Fair Doudizhu workflow to include `gamecore` in focused unit, race, and vet checks.
15. Added architecture and contract documentation covering dependency direction, module ownership, fairness, artifacts, live authority, private views, final archive behavior, failure policy, multi-instance evolution, and deferred integration.

The following remain intentionally deferred:

- integrating randomized setup or `LiveDirectory` into the existing Doudizhu Room/Hand application flow;
- production Doudizhu bidding, current-hand mutation, play patterns, turns, scoring, settlement, and replay;
- production server-seed generation/custody and public-beacon proof adapters;
- a concrete database implementation of `FinalRecordArchive` and any resulting current-schema update;
- active-game snapshots, command event-log recovery, ownership transfer, or live migration;
- generic lobby, matchmaking, table, spectator, or gateway routing capabilities;
- HTTP/WSS contracts and concrete client game-module loading;
- production Mahjong, poker, dice, board-game, or other game implementations;
- dynamic plugins, reflection-based discovery, scripting engines, universal rule DSLs, and universal card/tile/state schemas.

## References

- `AGENTS.md`
- `docs/architecture/overview.md`
- `docs/architecture/extensible-game-runtime.md`
- `docs/architecture/gamecore-v1.md`
- `docs/architecture/fair-doudizhu-card-shuffle-v1.md`
- `docs/architecture/fair-doudizhu-domain.md`
- `docs/requirements/fair-doudizhu-v1.md`
- `server/business/gamecore`
- `server/business/doudizhu/domain/randomizedsetup`
- `server/business/doudizhu/domain/carddeck/testdata/golden-v1.json`
- `.github/workflows/doudizhu.yml`

## Acceptance Results

- `gamecore` production files import only the Go standard library. A static import/vocabulary boundary test rejects concrete-game imports and Doudizhu-specific concepts.
- Descriptor tests reject empty, whitespace-padded, oversized, invalid participant-count, and mismatched identities.
- The registry rejects duplicates, resolves exact versioned keys, returns `ErrModuleNotFound` for unknown keys, and prevents mutable module identity from changing the registered contract.
- Fairness material rejects count/order mismatches and zero cryptographic evidence; its canonical digest changes when any bound contribution changes.
- The generic HMAC counter stream has a committed stable 64-byte vector.
- Rejection sampling has a scripted test proving a biased candidate below the threshold is discarded.
- Permutations are validated for counts `1`, `2`, `7`, `54`, and `137`.
- The test-only sequence module uses four participants and eleven items and generates/verifies through the same registry contract without importing Doudizhu.
- The Doudizhu adapter recomputes server/client commitments using the existing Goal 0023 functions and delegates shuffle/deal to the unchanged `carddeck` package.
- Golden compatibility fixes the same Goal 0023 server commitment, client commitments, first 64 random bytes, shuffle seed, complete deck, three hands, landlord cards, deck digest, and deal digest.
- Adapter tests reject altered participant commitments, altered payloads, unsupported artifact versions, and descriptor mismatches.
- Live-directory tests prove duplicate rejection, fixed descriptor identity, opaque copied command/view payloads, and public/private view routing.
- Twelve concurrent commands for one instance never overlap inside the concrete game.
- Two different instances enter their game implementations concurrently, proving there is no process-wide gameplay lock.
- Ordinary commands make zero archive calls.
- Archive failure retains one pending final record, blocks further commands, and retries the identical digest without reapplying the terminal command.
- Abort failure similarly retains one aborted record and does not call the concrete abort operation again.
- A concurrent second terminal command observes the removed/closed entry and cannot mutate an already archived game.
- No MySQL schema, migration, Redis state, HTTP/WSS contract, or frontend production file was added.

## Verification Evidence

### Formal implementation commits

- Goal lifecycle decision: `a5d0d7792cc6038e24fec4beeb84372869c39597`.
- Error and canonical encoding boundaries: `77711e1abef00fd3058e8224179a0ddf77427ccc`, `721f21e4ff87f03888d34023e645f4fc189f2af7`.
- Descriptor and fairness material: `a761e858ae6e9d92dbc9739e1a32fa3cf1eb5ff5`, `ee37f3ef9fb3f167685007bf10cf74c3f06750ef`.
- Deterministic primitives and setup registry: `b0433640987b24fc975c38ad5c760647bd2e64a4`, `c8b3f73c1c6e1dd66c63f67b9f38612f6b103689`.
- In-memory live lifecycle: `4f1ffb1586952e82ff4d0cf56959b111978fe491`.
- Core, import-boundary, and concurrency/lifecycle tests: `ebf1969a446880c8b975fa5fe82c87a4094dc0f8`, `54e370fe0fd266655022ac183cc59915805b996e`, `092c54543f91f12e422b32b91263eece2a716bd6`.
- Doudizhu adapter and golden compatibility: `814fd83bbc66263f0f0eea198688e5249a8e6ec8`, `10edf1fa572b42bf53cb832c0bfbf6d9ed4dc2b4`.
- Runtime architecture and contract documentation: `344f3e62fb1068fb9eb85b0689bc62e943693e57`, `7b030ab2236766fa7b50bee186f3c7f5a0d05cfa`.
- Focused workflow coverage: `04d8f66f405f6db09875535e8fdaf54219125832`.

### Preliminary local verification

A local isolated Go module was used before repository writes to compile the new packages and run focused ordinary and race tests. The repository's authoritative verification remained GitHub Actions with Go `1.25.8`.

### Real repository verification

- Fair Doudizhu run `30357225829`: success.
  - focused ordinary unit tests including `gamecore`, all Doudizhu domains, application, and infrastructure packages;
  - focused `-race` tests;
  - focused `go vet`;
  - signed reveal-key manifest TypeScript tests and Go-to-TypeScript interoperability;
  - real MySQL 5.7 Doudizhu integration.
- Full CI run `30357226064`: success.
  - `go mod tidy` cleanliness;
  - generated-code repeatability;
  - exact Go `1.25.8` formatting;
  - all Go unit tests;
  - Security/Admin race tests;
  - all Go builds;
  - local and production Compose validation;
  - production HTTPS mode checks;
  - Secure Envelope client and interoperability tests;
  - Cocos skeleton policy;
  - Admin Web typecheck/build;
  - full real MySQL 5.7 and Redis integration.
- Production runtime acceptance run `30357610609`: success.
  - production containers and dependencies;
  - HTTP and authenticated WebSocket behavior;
  - HTTPS and WSS behavior;
  - administrator bootstrap and login;
  - graceful acceptance cleanup.
- A repeated Fair Doudizhu run `30357610683` and repeated full CI run `30357610627` also succeeded after the temporary runtime workflow was introduced.

### Failures and fixes

- No implementation, unit, race, vet, formatting, module, build, integration, or runtime failure occurred in the formal repository verification.
- The first temporary CI PR closed automatically when its head was aligned exactly with the updated base before a new marker commit. A second CI-only PR was opened from the marker commit; no functional code changed.
- The first branch-cleanup workflow listened only for `pull_request.closed` and did not run in this connector-driven close path. It was changed to run when a cleanup-only PR opened; cleanup run `30357929126` then successfully deleted `goal0024-ci-trigger`.

### Cleanup evidence

- CI-only PRs `#9`, `#10`, and `#11` are closed and were never merged.
- Branch `goal0024-ci-trigger` was deleted; direct file access by that ref returns `404`.
- Temporary runtime and cleanup workflow files were deleted from `main`.
- Final cleanup commit before this completion report: `16e8131b06f9903dc104acf5f1a80da93368309d`.

## Working State

### Completed

- Versioned game identity, fairness material, deterministic random primitives, setup artifacts, explicit registry, Doudizhu compatibility adapter, test-only alternate module, in-memory live directory, terminal archive contract, failure policy, documentation, and CI coverage are complete.
- Active game state is now architecturally fixed as concrete game-owned in-memory state; only completed or explicitly aborted records cross the persistence port.
- All required focused, full, integration, and production runtime checks passed.
- Temporary verification resources were removed.

### In progress

- None.

### Remaining

- None within Goal 0024.

## Completion Report

Goal 0024 is complete. The repository now has a small reusable game business core that supports exact versioned game modules without turning Doudizhu concepts into platform-wide assumptions. Doudizhu remains the first concrete adapter and preserves every Goal 0023 deterministic result. Active game instances own their private state in memory, commands are serialized per instance, different games can run concurrently, and only immutable completed or aborted final records cross a narrow archive port.

The next stage should integrate this foundation into a real Doudizhu lifecycle: generate and protect the server seed behind its commitment, consume adapter-verified beacon evidence, construct the existing deterministic setup artifact, create the concrete in-memory Doudizhu game state with each player's current hand, expose authenticated private/public views, implement bidding and gameplay commands incrementally, and archive the final record and fairness transcript only after completion or explicit abort. The generic core should remain unchanged unless a second real game exposes a genuinely shared requirement.
