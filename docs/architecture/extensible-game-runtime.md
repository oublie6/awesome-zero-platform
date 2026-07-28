# Extensible Game Runtime Architecture

## 1. Purpose

The repository may host more than Fair Doudizhu. The architecture therefore separates reusable game-runtime capabilities from the rules and state of any concrete game.

The goal is not to build a speculative universal game engine. Only capabilities already required by at least one real game and clearly reusable by another game are extracted. Concrete rules remain inside concrete business modules.

## 2. Module boundaries

The intended server layout is:

```text
server/business/
  gamecore/       reusable business-level game primitives
  doudizhu/       Fair Doudizhu rules, state and adapters
  <future-game>/  another concrete game module
```

`gamecore` is a business capability rather than technical infrastructure, so it must not move into `server/foundation` or `server/platform`.

Dependency direction is one-way:

```text
application composition
  -> concrete game module
  -> gamecore

concrete game modules must not import one another
gamecore must not import a concrete game module
```

## 3. What belongs in gamecore

Only semantics that remain valid across different games belong in `gamecore`:

- stable game identity and ruleset version values;
- a compile-time module descriptor and registry;
- participant position/index validation where the count is supplied by the game descriptor;
- versioned fairness-material envelopes;
- deterministic cryptographic byte streams;
- unbiased bounded sampling;
- deterministic index permutations;
- common integrity digest and audit metadata primitives;
- opaque, versioned randomized-setup artifacts and verification contracts;
- an in-memory directory for active game instances;
- per-instance command serialization;
- opaque public/private view routing;
- a terminal final-record archive port.

The generic fairness layer operates on ordered participant contributions and index permutations. It does not know about poker cards, Doudizhu ranks, Mahjong tiles, dice faces, bidding, turns, scoring, or settlement.

The live runtime treats game commands, views, and final records as immutable byte payloads owned by the concrete game. It never inspects a hand, tile wall, hidden board, move, score, or settlement field.

## 4. What remains game-specific

Each concrete game module owns:

- canonical game items such as cards, Mahjong tiles, pieces, boards, or dice semantics;
- participant count and position meaning;
- canonical initial item order;
- how a deterministic permutation is applied;
- dealing, wall construction, board generation, or other randomized setup rules;
- game phases and legal state transitions;
- commands, events, snapshots, scoring, settlement, and replay semantics;
- all active private state, including current hands or other hidden information;
- public and participant-specific state projections;
- game-specific final-record and transcript payloads;
- game-specific client UI.

For example, Doudizhu continues to own its 54-card encoding, three-seat round-robin deal, three landlord cards, each player's current hand, bidding, play-pattern rules, and scoring.

## 5. Stable game identity

Every game instance must bind these values before gameplay begins:

```text
gameId
rulesetVersion
moduleVersion
fairnessSuiteId
participantCount
```

Suggested examples are:

```text
gameId = "doudizhu"
rulesetVersion = "doudizhu-standard-v1"
moduleVersion = "doudizhu-module-v1"
fairnessSuiteId = "fair-randomized-setup-v1"
participantCount = 3
```

These values are authoritative server configuration. Clients cannot choose an arbitrary implementation version after a room or match has started.

A future incompatible rule or encoding change creates a new explicit version. Existing completed transcripts remain verifiable with their original descriptor.

## 6. Compile-time module registry

The modular monolith uses an explicit compile-time registry. Dynamic shared-library loading, reflection-based plugin discovery, downloaded executable plugins, and runtime source compilation are out of scope.

A minimal randomized-setup module contract exposes:

```go
type RandomizedSetupModule interface {
    Descriptor() Descriptor
    GenerateSetup(FairnessMaterial) (SetupArtifact, error)
    VerifySetup(FairnessMaterial, SetupArtifact) error
}
```

The following properties are mandatory:

- selection is by an exact versioned descriptor;
- registration rejects duplicate keys;
- lookup returns a concrete immutable module wrapper;
- the registry contains no game-specific switch statement;
- registration is explicit rather than performed by package-global `init` functions;
- modules do not receive database, HTTP, Redis, clocks, loggers, or global process state through this interface.

## 7. Fairness material

The common fairness envelope binds:

- the exact game and ruleset descriptor;
- match, hand, or game-instance ID;
- disclosed server seed and its precommitted digest;
- an ordered list of accepted participant contribution digests and commitments;
- locked beacon provider, round, digest, and proof reference;
- algorithm-suite version;
- audit references such as reveal-key ID and public-key hash.

The number of contributions is validated against `participantCount`. Their order is defined by the concrete game's stable participant-position order.

The common layer may derive a deterministic seed and index permutation. The concrete module interprets that permutation against its canonical item sequence and produces its own setup artifact.

## 8. Randomized setup artifact

A generic runtime must not persist a fake universal card model. Instead, a module returns a versioned artifact envelope:

```text
gameId
rulesetVersion
moduleVersion
artifactVersion
canonicalPayload
payloadDigest
```

`canonicalPayload` is owned and verified by the concrete module. For Doudizhu it can encode the shuffled deck, three initial hands, and landlord cards. A Mahjong module could encode a tile wall and initial hands without changing the runtime contract.

The common runtime treats the payload as immutable bytes plus validated metadata. It never edits fields inside a game-owned payload.

## 9. Active game authority

An active game is a concrete in-memory object implementing the narrow `LiveGame` boundary. It owns the complete authoritative state required by that game.

For Doudizhu, this eventually includes:

```text
current hand for every seat
landlord cards
played cards and action history
current phase and acting seat
landlord, multiplier and settlement state
fairness/setup references required by the game
```

The database is not consulted for each bid, move, play, pass, or other ordinary command. Redis is not the source of truth for current hands or hidden state.

The generic directory stores only:

```text
instanceId -> concrete LiveGame instance
fixed game descriptor
per-instance serialization guard
optional pending final record
```

The directory never reads or mutates fields inside the concrete game state.

## 10. Command serialization and views

All commands for one instance are serialized through that instance's guard. This prevents two concurrent requests from consuming or modifying the same logical state version at once.

Different game instances have different guards and are not forced through one process-wide gameplay lock.

The concrete module validates:

- whether the actor belongs to the game;
- whether it is that actor's turn;
- expected state version;
- ownership of cards, tiles, pieces, resources, or other private state;
- action legality and resulting phase transitions.

The concrete game also produces public and participant-specific views. `gamecore` validates only the participant-position range and copies the returned bytes. It cannot infer, expose, filter, or accidentally merge another player's private state because it does not understand the payload format.

Transport code must authenticate the requester and ask the live instance for the correct view. A broadcast must never reuse one participant's private view for another recipient.

## 11. Terminal archive policy

Ordinary in-progress commands do not cross a persistence port.

A final archive is attempted only when:

- the concrete game returns a terminal completion; or
- the runtime explicitly aborts the instance with a reason.

The runtime constructs an immutable final-record envelope containing:

```text
instanceId
game descriptor
completed or aborted status
final game version
game-owned final payload
integrity digest
```

The sequence is:

```text
apply terminal command or abort
-> create immutable final record
-> mark the in-memory entry as finalization pending
-> call FinalRecordArchive
-> remove the live instance only after archive success
```

If the archive fails:

- the terminal command or abort is not applied a second time;
- the exact final record remains attached to the in-memory entry;
- additional commands and aborts are rejected;
- an explicit retry sends the same record again;
- removal happens only after a successful retry.

The runtime makes one archive call per attempt. The archive implementation must be idempotent by `instanceId` and final-record digest because network or process failures can make delivery outcomes uncertain.

## 12. Process loss and graceful shutdown

The first runtime version intentionally does not promise recovery of active games after unexpected process loss.

A crash, container eviction, machine failure, or forced deployment may therefore abort active games. This is acceptable while games have no real-money value and the product explicitly treats such games as void or restartable.

A graceful shutdown should later follow this operational sequence:

```text
stop accepting new games
stop or drain new commands
allow active games to finish for a bounded interval
explicitly abort remaining games when possible
archive those abort records
terminate the process
```

Snapshots or action-log replay may be added when a real recovery requirement appears. That future capability must sit behind explicit ports and must not change the rule that concrete game modules own their state format.

## 13. Multi-instance evolution

When horizontal scaling becomes necessary, each active game must be assigned to one server instance. A router may maintain:

```text
game instance ID -> server instance ID
player connection -> gateway/server instance
server instance heartbeat
```

Redis or a coordination service may store those routing facts. It still must not become the authoritative storage for current hands or concrete game state.

Commands must be routed to the owning process. Ownership transfer and live migration are intentionally deferred because they require snapshots or event replay.

## 14. Final persistence strategy

The current version requires no schema change. `FinalRecordArchive` is a port, not a database implementation.

When final persistence is implemented, a narrow common envelope may contain:

- game descriptor;
- completed or aborted status;
- final version;
- module-owned final-record version and bytes;
- fairness/setup artifact version and bytes or references;
- integrity digests and audit timestamps.

Concrete modules may own dedicated tables when indexed game-specific results or invariants require them. A single unrestricted JSON table for every live and completed game is not the default design.

Database storage is therefore limited to completed or explicitly aborted records in the current lifecycle. It does not participate in active command processing.

## 15. Aggregate strategy

The current `doudizhu` Room and Hand aggregates remain concrete Doudizhu aggregates. They should not be renamed into generic game aggregates merely because another game may be added later.

Reusable lobby, matchmaking, table allocation, spectator, or match-session capabilities should be introduced only when a second real game requires them.

The generic live directory is a runtime boundary, not a replacement for concrete domain aggregates.

## 16. Client strategy

Clients should evolve toward:

```text
client shell
  -> authentication, navigation, transport, updates, shared UI primitives
  -> game catalog and exact module/version selection
  -> Doudizhu client module
  -> future game client module
```

The shell must not hard-code Doudizhu as the only game. Concrete game assets, screens, state reducers, input handling, and transcript presentation remain in game-specific client modules.

Shared API envelopes may contain `gameId`, `rulesetVersion`, and `moduleVersion`; game-specific command names and payloads remain versioned by the concrete module.

## 17. Compatibility with Goal 0023

Goal 0023 remains a valid Doudizhu v1 implementation and golden vector. Extensibility work must not silently change its canonical bytes or digests.

The extraction must:

- wrap only genuinely reusable deterministic-stream, rejection-sampling, and permutation behavior;
- preserve all Goal 0023 Doudizhu golden-vector outputs;
- keep Doudizhu card, deck, deal, and transcript semantics in the Doudizhu module;
- prove the generic core with a test-only non-Doudizhu module or sequence, not with a fake production game.

Any change to Goal 0023 canonical bytes requires a new Doudizhu algorithm version rather than modifying v1.

## 18. Non-goals

This architecture does not introduce:

- a visual game editor;
- user-uploaded executable game plugins;
- scripting engines;
- reflection-heavy dependency injection;
- a universal rule DSL;
- a universal card/tile/piece schema;
- a generic event-sourcing framework;
- microservices per game;
- a shared gameplay state machine that tries to model every game;
- database writes for every active command;
- Redis storage of current hands;
- automatic active-game recovery in the first version.

The extension point is a small, versioned Go business interface with explicit compile-time registration, concrete module ownership, in-memory live authority, and terminal-only archival.
