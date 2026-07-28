# Doudizhu Live-Hand Runtime v1

## 1. Purpose

This document defines the first concrete integration of the reusable `gamecore` runtime with the Doudizhu Room and Hand application flow.

The central policy is:

```text
active private game state is authoritative only in process memory
completed or explicitly aborted game state is archived once as an immutable final record
```

The runtime does not persist the active deck, current hands, landlord cards, or server seed in MySQL, Redis, command results, outbox events, or ordinary logs.

This version starts a deterministic Doudizhu live hand and exposes identity-scoped views. Bidding, landlord selection, legal play recognition, scoring, and normal gameplay completion remain deferred.

## 2. Component ownership

```text
application Hand aggregate
  coordination metadata only
  ├── seats and authenticated account ownership
  ├── server commitment
  ├── reveal-key audit binding
  ├── ordered contribution commitments/digests
  ├── verified beacon evidence
  └── phase and aggregate version

runtime SeedVault
  └── pre-deal server seed, memory only

runtime LiveHandCoordinator
  ├── constructs fairness material
  ├── generates and verifies deterministic setup
  ├── creates the concrete live game
  ├── registers it in gamecore.LiveDirectory
  ├── resolves public/private views
  └── coordinates terminal archive and seed release

Doudizhu domain livehand.Game
  ├── complete deterministic setup artifact
  ├── initial deck/deal
  ├── each seat's current cards
  ├── landlord cards
  ├── fairness transcript material
  └── Doudizhu-owned public/private/terminal payload schemas

gamecore.LiveDirectory
  ├── per-instance serialization
  ├── terminal-record construction
  ├── pending-finalization retention
  └── archive retry without replaying game logic

mysqlarchive.Archive
  └── immutable completed/aborted final record only
```

`gamecore` never parses cards, seats, landlord cards, Doudizhu phases, or Doudizhu payloads.

## 3. Server-seed custody

`runtime.SeedVault` obtains exactly 32 bytes from an injected `io.Reader`; production defaults to `crypto/rand.Reader`.

For one hand ID it:

1. rejects an empty, oversized, or duplicate hand ID;
2. reads exactly 32 bytes;
3. computes the unchanged Goal 0023 hand-bound server commitment;
4. stores the seed in an in-process map;
5. returns only the hand ID and commitment;
6. returns seed values by copy;
7. clears the stored bytes before deletion on release.

The seed is not written to the Hand snapshot. Before live-game creation it exists only in the vault. After live-game creation, the concrete live game also retains the fairness material needed for terminal transcript construction. Both copies are cleared or released when terminal cleanup succeeds.

Unexpected process loss can therefore invalidate an active hand. v1 deliberately does not attempt process-crash restoration.

## 4. Hand setup preparation

`SeededHandSetupProvider` binds four trusted inputs before a Hand aggregate is created:

- the new server commitment from `SeedVault`;
- the current signed reveal-key manifest;
- the SHA-256 digest of that manifest's public key;
- one explicitly configured beacon provider and round plan.

The returned `application.HandSetup` contains audit and commitment metadata only. It never contains the server seed.

If room or Hand creation fails after setup preparation, application compensation calls `ReleaseHand` so the unused seed is removed.

## 5. Verified beacon boundary

`application.BeaconVerifier` is an application port. `LockHandBeacon` does not pass caller-supplied evidence directly to the Hand aggregate.

```text
client/request beacon value
  -> BeaconVerifier.Verify(committed plan, supplied value)
  -> verifier-returned authoritative value
  -> Hand.LockPublicBeacon
```

The domain aggregate still verifies that the returned provider and round match the plan committed when the hand was created. A verifier error, altered digest, empty digest, mismatched provider, or mismatched round cannot advance the Hand.

Network fetching and provider-specific cryptographic proof validation remain adapter concerns outside this version.

## 6. Deterministic live-hand startup

A live hand may start only from a persisted Hand snapshot in `DEALING` with:

- three stable seats;
- three accepted commitments;
- three accepted reveals and protected-record references;
- one verified locked beacon;
- one retained server seed;
- the expected reveal-key audit binding.

`LiveHandCoordinator.Start` performs:

```text
read retained seed
-> build gamecore.FairnessMaterial in seat order 1,2,3
-> generate Doudizhu randomized setup artifact
-> verify the generated artifact
-> construct livehand.Game
-> verify the Goal 0023 transcript matches the setup
-> register the game in gamecore.LiveDirectory
-> register the concrete Doudizhu instance in the coordinator
```

The existing Goal 0023 Card/Deck v1 commitment, random stream, shuffle, deal, transcript, and golden bytes remain unchanged.

## 7. Database transaction ordering and compensation

`Service.MarkHandDealt` deliberately creates the live game before persisting the Hand phase transition:

```text
lock and restore persisted DEALING Hand
-> validate domain MarkDealt transition in memory
-> create and register deterministic live game
-> update persisted Hand snapshot to BIDDING
-> append events and complete command
-> commit database transaction
```

If any database operation after live registration fails, the command executor invokes `RollbackStart` outside the failed transaction. Compensation removes the uncommitted live instance without creating a terminal archive.

The persisted Hand snapshot contains only coordination metadata and the new phase/version. It contains no deck, hand cards, landlord cards, setup payload, or server seed.

Command idempotency remains database-backed. A duplicate completed command replays its stored result and does not create another live instance.

## 8. Active authority and concurrency

Once registration succeeds, the concrete `livehand.Game` is the authority for the active private state.

The live directory uses one lock per game instance:

- operations for the same hand are serialized;
- separate hands can progress concurrently;
- payloads are copied at the common boundary;
- an instance pending terminal archival rejects further gameplay or abort operations.

Redis is not a source of truth for cards or deck state. A future multi-instance router may store only instance ownership and connection routing metadata.

## 9. Public and private views

Application query methods first load the trusted persisted Hand snapshot and confirm that the authenticated account occupies a seat. Clients cannot select a seat number.

The public payload contains:

- versioned payload identity;
- hand ID and live phase/version;
- seat positions and account IDs;
- remaining-card counts;
- setup, deck, and deal digests;
- landlord-card count.

It contains no card code or ID, server seed, complete deck, landlord cards, or contribution plaintext.

The private payload contains the same public projection plus exactly the authenticated viewer's current cards. The position is derived from the server-owned seat assignment.

Returned payload bytes are copied. Mutating a caller-owned response cannot mutate live state.

## 10. Terminal disclosure and archive

The concrete Doudizhu terminal payload is versioned and game-owned. For an explicit abort it contains:

- terminal status and reason;
- final live-state version;
- the randomized setup artifact and digest;
- the canonical Goal 0023 transcript and digest;
- each current hand at termination;
- landlord cards.

These private materials are disclosed only after the live game becomes terminal.

The common directory then constructs an immutable `gamecore.FinalRecord` containing:

```text
instance ID
exact game descriptor
status: completed | aborted
final version
game-owned payload
integrity digest
```

Normal gameplay completion is supported by the `gamecore` terminal contract, but the Doudizhu gameplay commands that produce it are deferred.

## 11. Archive idempotency and retry

`mysqlarchive.Archive` writes one row keyed by game instance ID. It stores:

- game/ruleset/module/fairness identities;
- participant count;
- terminal status and version;
- opaque terminal payload;
- final-record digest;
- archive timestamp.

An exact retry of the same instance ID and record is accepted. A retry whose descriptor, status, version, payload, or digest differs returns `ErrArchiveConflict`.

Terminal processing follows:

```text
invoke concrete terminal operation once
-> build immutable FinalRecord once
-> retain the exact record as pending
-> attempt archive
```

On archive failure:

- the live entry remains finalization-pending;
- the same pending record is retained;
- no gameplay command is accepted;
- the concrete abort or terminal command is not executed again;
- `RetryArchive` submits the same record.

Only after archive success does the coordinator remove the active game, remember the finalized result for idempotent retries, and release the retained seed.

## 12. Pre-live termination

A hand can terminate before deterministic live-game creation. In that case there are no cards or setup artifact to archive.

The application calls `ReleasePrepared`, which removes an existing prepared seed and is safe when the seed was already released. It refuses to release a seed while a corresponding live hand is still active.

## 13. Persistence and encryption policy

Active private game state is not persisted, so per-card application-layer database encryption is not part of this design.

The database stores:

- Room and Hand coordination snapshots;
- protected reveal contribution records;
- command/idempotency state;
- outbox events without private game state;
- immutable final records after completion or explicit abort.

Completed-game payloads are ordinary restricted business records. Database access control, least-privilege accounts, storage encryption, backup encryption, and audit controls are deployment responsibilities. Application-field encryption can be added only if a concrete threat model requires it.

## 14. Failure policy

v1 intentionally accepts:

```text
unexpected process loss
-> active in-memory hands are not recoverable
-> affected games are voided or handled by an operational policy
```

Not implemented:

- active-state database snapshots;
- Redis current-hand storage;
- per-action persistence;
- event-log replay recovery;
- transparent live migration;
- cross-instance hand ownership transfer.

These capabilities should be added only when product value, real-money exposure, or availability requirements justify their complexity.

## 15. Compatibility boundary

The following are immutable compatibility contracts:

- Goal 0023 Card/Deck v1 canonical bytes and golden vector;
- Goal 0024 `gamecore` descriptor, setup artifact, live directory, and final-record semantics;
- versioned Doudizhu public, private, setup, transcript, and terminal payload identities.

An incompatible future change must introduce a new explicit ruleset, module, fairness suite, artifact, or payload version rather than silently changing v1 behavior.

## 16. Deferred work

The next gameplay goals may add concrete Doudizhu behavior for:

- bidding and landlord selection;
- doubling and multipliers;
- legal play-pattern recognition;
- turn order, comparison, and pass handling;
- normal completion, scoring, spring rules, and settlement;
- authenticated HTTP/WSS commands and view delivery;
- instance routing for multiple server processes.

Those additions remain inside the Doudizhu module and must preserve the active-memory and terminal-archive boundaries unless a later goal explicitly replaces them.
