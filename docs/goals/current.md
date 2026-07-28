# Goal 0025: Verified Doudizhu Setup, In-Memory Hands, and Final Archive

## Status

- State: completed
- Started: 2026-07-28
- Completed: 2026-07-28
- Blockers: None.

## Goal

Integrate the verified fairness and gamecore foundations into a real Doudizhu hand-start lifecycle. Generate and retain the server seed only in process memory, require public-beacon evidence to pass an application verifier, build the existing deterministic setup artifact after all reveals, create an in-memory Doudizhu live game that owns every player's current hand, expose identity-scoped public/private views, and persist only immutable completed or aborted final records through an idempotent MySQL archive. Do not persist the active deck, current hands, landlord cards, or server seed in the live Hand snapshot, Redis, command results, outbox events, or ordinary logs.

## Scope

ChatGPT owns architecture, implementation, tests, failure diagnosis and fixes, documentation, repository verification, commits, and pushes directly to `main`.

Deliver:

1. Add a Doudizhu server-seed custodian that:
   - obtains exactly 32 bytes from an injected cryptographic entropy source;
   - computes the unchanged Goal 0023 hand-bound server commitment;
   - retains the seed only in process memory by hand ID;
   - rejects duplicate preparation and unknown release/read operations;
   - returns copies rather than mutable internal seed storage;
   - clears retained seed bytes when a hand is discarded or finalised.
2. Add a production-oriented `HandSetupProvider` implementation that combines:
   - the in-memory server-seed custodian;
   - the currently active signed reveal-key manifest;
   - an explicitly configured public-beacon plan;
   - the existing `application.HandSetup` contract.
3. Add an application `BeaconVerifier` port. `LockHandBeacon` must never pass caller-supplied evidence directly into the Hand aggregate; the verified value returned by the port is authoritative and must still match the committed provider and round.
4. Add a concrete Doudizhu `LiveGame` implementation that owns:
   - the immutable randomized setup artifact;
   - the full initial deck and deal;
   - each seat's current cards in memory;
   - landlord cards in memory;
   - the fairness material needed to construct a terminal transcript;
   - the live-game version and phase.
5. Keep Doudizhu-specific cards, seats, hands, landlord cards, payloads, and transcript construction inside the Doudizhu module. Do not add card or hand semantics to `gamecore`.
6. Add canonical, versioned live-view payloads:
   - public view: hand ID, phase, version, seat identities/positions, remaining-card counts, public setup digests, and no private cards;
   - private view: the authenticated viewer's exact current cards plus the public fields;
   - no view may expose another seat's cards, the server seed, the complete deck, unrevealed landlord cards, or raw contribution plaintext.
7. Add a Doudizhu live-hand runtime coordinator that:
   - retrieves the retained seed only after the Hand snapshot has three accepted reveals and one verified locked beacon;
   - builds `gamecore.FairnessMaterial` in stable seat order;
   - generates and verifies the existing `randomizedsetup` artifact;
   - constructs and registers the concrete live game in `gamecore.LiveDirectory`;
   - supports transaction compensation without archiving an uncommitted live game;
   - releases pre-deal secrets when a hand terminates before live-game creation.
8. Integrate `Service.MarkHandDealt` so that `DEALING -> BIDDING` succeeds only when the deterministic live game has been created. The persisted Hand snapshot records phase/commitment/evidence metadata only; it must not contain the deck or current cards.
9. Add authenticated application query methods for public and private live-hand views. Private view resolution must derive the viewer's seat from the persisted Hand seat assignment rather than trusting a client-supplied seat number.
10. Add lifecycle compensation hooks to the command executor so in-memory preparation/registration is rolled back when the surrounding database transaction fails, without changing command idempotency semantics.
11. Add immutable Doudizhu terminal payloads that include the final status/reason, setup artifact, current/initial card state as appropriate, and a verifiable Goal 0023 fairness transcript. Transcript disclosure occurs only for completed or explicitly aborted live hands.
12. Add an idempotent MySQL implementation of `gamecore.FinalRecordArchive` and update the complete current schema. The archive must:
    - insert one final record per game instance;
    - accept an exact retry of the same instance ID and digest;
    - reject a conflicting retry;
    - store the opaque final payload and integrity digest;
    - never store an active-game snapshot or current-hand update.
13. Integrate explicit live-hand abort with the archive path. Normal completed-game archival remains callable by the live game contract, while bidding/play/settlement commands that produce a normal completion remain deferred.
14. Document exact ownership, lifecycle, failure, disclosure, transaction-compensation, and persistence boundaries.
15. Add focused unit, property, race, vet, import-boundary, SQL-mock/real-MySQL integration, schema, formatting, full-repository, and production runtime verification.

The following remain outside this goal:

- bidding decisions, landlord selection, doubling, legal play-pattern recognition, turn comparison, passes, scoring, spring rules, settlement calculations, and replay UI;
- public HTTP/WSS game-command or view endpoints;
- automatic public-beacon network fetching or provider-specific cryptographic proof algorithms; this goal defines and enforces the verifier boundary and supplies deterministic adapters for tests;
- active-game snapshots, Redis hand storage, command event-log recovery, process-crash restoration, live migration, or cross-instance ownership transfer;
- dynamic game plugins, rule DSLs, universal card/tile schemas, or a shared gameplay state machine;
- encrypting ordinary completed-game payloads at the application field level; database access control and storage/backup encryption remain deployment concerns.

## References

- `AGENTS.md`
- `docs/architecture/overview.md`
- `docs/architecture/extensible-game-runtime.md`
- `docs/architecture/gamecore-v1.md`
- `docs/architecture/doudizhu-live-hand-runtime-v1.md`
- `docs/architecture/fair-doudizhu-domain.md`
- `docs/architecture/fair-doudizhu-application-persistence.md`
- `docs/architecture/fair-doudizhu-card-shuffle-v1.md`
- `docs/architecture/reveal-key-lifecycle-v1.md`
- `docs/requirements/fair-doudizhu-v1.md`
- `server/business/gamecore`
- `server/business/gamecore/infrastructure/mysqlarchive`
- `server/business/doudizhu/application`
- `server/business/doudizhu/domain/carddeck`
- `server/business/doudizhu/domain/livehand`
- `server/business/doudizhu/domain/randomizedsetup`
- `server/business/doudizhu/infrastructure/mysqlstore`
- `server/business/doudizhu/infrastructure/runtime`
- `server/foundation/revealkeys`
- `server/database/schema/current.sql`
- `.github/workflows/doudizhu.yml`

## Constraints

- Follow `AGENTS.md`; all edits and commits go directly to `main`. Do not create a feature branch, verification branch, or pull request.
- Existing Goal 0023 canonical bytes, commitments, random stream, shuffle, deal, transcript, and golden vectors are immutable.
- Existing Goal 0024 gamecore descriptor, artifact, directory, and final-record contracts remain generic. Any small gamecore change must be justified by the first real integration and must remain free of Doudizhu vocabulary.
- The server seed, complete deck, landlord cards, and players' current cards must not be added to `domain.HandSnapshot`, MySQL active-hand rows, Redis, outbox payloads, command-result payloads, or normal logs.
- Active private game state is authoritative only in memory. The database may retain room/hand coordination metadata and protected commit-reveal records, but not live card state.
- Database persistence is terminal-record-only for the live game. Do not write a database row for every bid, play, or view.
- Private view authorization derives seat ownership from trusted server state.
- The public-beacon verifier result, not raw request data, is passed to the domain aggregate.
- Randomness uses `crypto/rand` or an injected `io.Reader`; do not use `math/rand`, time, UUID bytes, process-global mutable randomness, or client-only entropy.
- Do not add frontend production code or public transport contracts.
- Update the complete current schema only; do not add migration history or committed patch SQL.
- Run memory-intensive verification sequentially with low-concurrency Go commands.

## Acceptance Criteria

- Seed-custody tests prove exact 32-byte generation, commitment compatibility, duplicate rejection, copy isolation, release zeroisation, entropy failure handling, and concurrency safety.
- The setup provider binds one active reveal-key manifest, its decoded SHA-256 public-key digest, one configured beacon plan, and one newly retained server seed to the returned Hand setup.
- Beacon-lock application tests prove unverified, mismatched, altered, zero-digest, and verifier-failure evidence cannot advance the Hand; only the verifier-returned value is locked.
- The concrete live game starts with the exact Goal 0023 deterministic deck, three 17-card hands, and three landlord cards for the golden material.
- Public view contains no card codes/IDs, seed, full deck, landlord cards, or contribution plaintext.
- Each private seat view contains exactly that seat's 17 cards and no other seat's cards.
- Mutating a returned view payload cannot mutate live state.
- Runtime setup rejects missing seed, wrong Hand phase, incomplete contributions, missing/mismatched beacon, descriptor mismatch, tampered artifact, and duplicate live instance IDs.
- `MarkHandDealt` does not advance the persisted Hand to `BIDDING` unless live registration succeeds.
- A database transaction failure after live registration invokes compensation and leaves no uncommitted live instance.
- A successful `MarkHandDealt` leaves the seed and card state only inside approved in-memory components and persists no deck/hand fields.
- Private query resolution rejects non-seated users and ignores any caller attempt to choose another seat.
- Explicit abort builds and verifies a fairness transcript, archives one final record, removes the live instance only after archive success, and clears retained seed material.
- Archive retry with the same instance ID and digest is idempotent; a conflicting digest is rejected.
- The archive schema contains final metadata, opaque payload, digest, and timestamps, but no active-hand/current-card table.
- `gamecore` still imports only the standard library and contains no Doudizhu/card/landlord vocabulary.
- `go test -count=1 -p 1 -parallel 1 ./business/gamecore/... ./business/doudizhu/domain/... ./business/doudizhu/application/... ./business/doudizhu/infrastructure/runtime/... ./business/gamecore/infrastructure/mysqlarchive/...` succeeds.
- The corresponding `-race` and `go vet` checks succeed.
- Real MySQL 5.7 integration proves schema application and archive idempotency/conflict behavior.
- Existing module tidy, generated-code, formatting, all Go unit, Secure Envelope, Admin Web, MySQL/Redis integration, Compose, HTTP/WS, HTTPS/WSS, and production runtime checks remain green.
- Final evidence records exact commits, actual failures/fixes, main-only verification method, unavailable checks, and intentionally deferred work.

## Working State

### Completed

- Added in-memory server-seed custody with cryptographic entropy, Goal 0023-compatible commitment generation, copy isolation, duplicate protection, concurrency safety, and zeroising release.
- Added a setup provider that binds the active signed reveal-key manifest and configured beacon plan without persisting the server seed.
- Added the application `BeaconVerifier` boundary and required the verifier-returned value before domain beacon lock.
- Added a concrete in-memory Doudizhu live game containing the deterministic setup, all current hands, landlord cards, transcript material, version, and phase.
- Added a runtime coordinator that validates complete fairness evidence, generates and verifies the existing setup artifact, registers the live game, compensates failed database transactions, and releases pre-live secrets.
- Integrated deterministic live-game creation into `MarkHandDealt` before persistence advances the Hand to `BIDDING`.
- Added server-derived, identity-scoped public and private views with copy isolation and no cross-seat card disclosure.
- Added versioned terminal payloads and retry-safe explicit abort handling.
- Added the idempotent MySQL final-record archive and schema version `0011`; active card state remains absent from MySQL and Redis.
- Added formal architecture documentation for seed custody, active-memory authority, transaction compensation, terminal disclosure, and final archive behavior.
- Completed focused, race, vet, full-repository, real MySQL/Redis, client interoperability, Compose, and production runtime verification.
- Removed every temporary repair/final-verification workflow, script, trigger file, and temporary Issue.

### In progress

- None.

### Remaining

- Deferred to later goals: bidding and landlord selection, doubling, legal play patterns, turn comparison and passes, scoring and settlement, authenticated HTTP/WSS gameplay transport, automatic beacon adapters, and active-game crash recovery.

## Completion Report

### Delivered implementation

- Server-seed custody: `8f6b9f5a7c5903338be919be6b2d9d1ebe02d22f`, `0489082670bee5560ff6fb88b9ea2818d78589c2`.
- Concrete live hand and common integration ports: `06156273b1af588aab5af8d0f0b3c275789a24c7`, `d2a81cca1bb476375baeb4db406c17b8576a7f14`, `bf012ef489212e48750ce43b445791391b1a09a9`.
- Verified beacon, transaction compensation, and application lifecycle: `8ffff50f64c335b4c888804ff6fa1029e2bc4753`, `4e9e8f90cd17c725e6dd840c2345f40a60568f63`, `2351c256499627de752333840ff33edc951ba38e`.
- Trusted identity-scoped live views: `12b0e4a2e8f5004279200a54abc22d798a3c42ee`, `243e4a73f7afa679d0407acd918869d95ea16f39`, `d310dc1dd4485d1985d097423c6bae8d4404ca24`.
- Immutable MySQL final archive and retry-safety: `9bcf15d2198e6b5e0c27a97a54cee04e39512d19`, `70d88fc7292738a317634a213bcda427deef53cb`, `8fffc4e0ca030a4738b07bc9bb023ea4629c6ac8`.
- Formal architecture: `9f6a38faa04ea60ab2c3451199ad6c59ca381f9e`, `0e0c931b027e538365eec68b99dbdb50c555009c`.

### Actual failure and repair

The first full-main verification, `CI 30361241969`, exposed three concrete integration defects:

1. four newly added Go files had not been committed with the repository's Go 1.25.8 `gofmt` output;
2. the bootstrap integration test still expected schema version `0010` after `game_final_records` raised the schema to `0011`;
3. two real-MySQL service fixtures still called the old `NewService` signature without `BeaconVerifier` and `LiveHandRuntime` dependencies.

The dependent focused run `30361429871` was skipped because the upstream full CI failed; it was not an additional code failure.

Commit `831084b412812d352d998fcb308eb4e626c26969` applied the exact formatting, schema assertion, table assertion, and fixture repairs. Main-only repair run `30368046405` then passed focused ordinary tests, `-race`, `go vet`, MySQL/Redis startup, schema and seed application, and real MySQL integration.

### Final verification

A temporary read-only `workflow_run` verifier locked commit `8d19f901ba1481b2e62c964f5da768da2087b4e9` and ran the repository's real verification commands against latest `main`.

Both final runs succeeded:

- `30370110270` — server, clients, full integration, and production runtime all succeeded;
- `30370288259` — repeated latest-main verification also succeeded and published `goal0025/final-full: success`.

The verified surface included:

- `go mod tidy` cleanliness;
- generated-code and Go 1.25.8 formatting checks;
- Goal 0025 focused ordinary, race, and vet checks;
- all Go unit tests;
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

All implementation, repair, documentation, and cleanup commits were written directly to `main`. No feature branch, verification branch, or pull request was created.

Temporary repair automation was removed by `fce05d4267b173ccaed04e24f54f5fbae5b975a5`; temporary Issues #12 and #13 were closed. The final verifier was removed by `db898ed4848f14bb9ced8739f0df64d2dab0d2ba` after its first run started, preventing further repeated triggers. Repository fetches confirmed the temporary workflow, script, and trigger paths no longer exist.

### Deferred work

Goal 0025 intentionally does not implement bidding, landlord selection, gameplay commands, settlement, public gameplay transport, automatic beacon-provider verification, or active-hand crash restoration. These remain versioned follow-on goals and must preserve the Goal 0023 deterministic bytes, Goal 0024 generic gamecore contracts, and the active-memory/terminal-archive boundary documented in `doudizhu-live-hand-runtime-v1.md`.
