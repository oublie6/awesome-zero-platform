# Goal 0025: Verified Doudizhu Setup, In-Memory Hands, and Final Archive

## Status

- State: ready
- Started: Not yet.
- Completed: Not yet.
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
- `docs/architecture/fair-doudizhu-domain.md`
- `docs/architecture/fair-doudizhu-application-persistence.md`
- `docs/architecture/fair-doudizhu-card-shuffle-v1.md`
- `docs/architecture/reveal-key-lifecycle-v1.md`
- `docs/requirements/fair-doudizhu-v1.md`
- `server/business/gamecore`
- `server/business/doudizhu/application`
- `server/business/doudizhu/domain/carddeck`
- `server/business/doudizhu/domain/randomizedsetup`
- `server/business/doudizhu/infrastructure/mysqlstore`
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

- Goal 0024 was completed, verified, cleaned up, and archived.
- The user confirmed that active private card state is memory-only and the database is used for completed or explicitly aborted game archives.

### In progress

- None.

### Remaining

- Implement the complete Goal 0025 scope and verification.

## Completion Report

Pending.
