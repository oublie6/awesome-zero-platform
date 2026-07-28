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
- opaque, versioned randomized-setup artifacts and verification contracts.

The generic fairness layer operates on ordered participant contributions and index permutations. It does not know about poker cards, Doudizhu ranks, Mahjong tiles, dice faces, bidding, turns, scoring, or settlement.

## 4. What remains game-specific

Each concrete game module owns:

- canonical game items such as cards, Mahjong tiles, pieces, boards, or dice semantics;
- participant count and position meaning;
- canonical initial item order;
- how a deterministic permutation is applied;
- dealing, wall construction, board generation, or other randomized setup rules;
- game phases and legal state transitions;
- commands, events, snapshots, scoring, settlement, and replay semantics;
- game-specific transcript payload and presentation;
- game-specific client UI.

For example, Doudizhu continues to own its 54-card encoding, three-seat round-robin deal, three landlord cards, bidding, play-pattern rules, and scoring.

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

A minimal module contract is expected to expose:

```go
type Descriptor struct {
    GameID            GameID
    RulesetVersion    RulesetVersion
    ModuleVersion     ModuleVersion
    FairnessSuiteID   FairnessSuiteID
    ParticipantCount  uint8
}

type RandomizedSetupModule interface {
    Descriptor() Descriptor
    GenerateSetup(FairnessMaterial) (SetupArtifact, error)
    VerifySetup(FairnessMaterial, SetupArtifact) error
}
```

The exact Go API may be refined during implementation, but these properties must remain:

- selection is by an exact versioned descriptor;
- registration rejects duplicate keys;
- lookup returns a concrete immutable module;
- the registry contains no game-specific switch statement;
- modules do not receive database, HTTP, Redis, or global process state through this interface.

## 7. Fairness material

The common fairness envelope should bind:

- exact game and ruleset descriptor;
- match or hand ID;
- disclosed server seed and its precommitted digest;
- an ordered list of accepted participant contribution digests;
- locked beacon provider, round, digest, and proof reference;
- algorithm-suite version;
- optional audit references such as reveal-key ID and public-key hash.

The number of contributions is validated against `participantCount`. Their order is defined by the concrete game's stable participant-position order.

The common layer may derive a deterministic seed and index permutation. The concrete module interprets that permutation against its canonical item sequence and produces its own setup artifact.

## 8. Randomized setup artifact

A generic runtime must not persist a fake universal card model. Instead, a module returns a versioned artifact envelope:

```text
gameId
rulesetVersion
artifactVersion
canonicalPayload
payloadDigest
```

`canonicalPayload` is owned and verified by the concrete module. For Doudizhu it can encode the shuffled deck, three hands, and landlord cards. A Mahjong module could encode a tile wall and initial hands without changing the runtime contract.

The common runtime treats the payload as immutable bytes plus validated metadata. It never edits fields inside a game-owned payload.

## 9. Aggregate and persistence strategy

The current `doudizhu` Room and Hand aggregates remain concrete Doudizhu aggregates. They should not be renamed into generic game aggregates merely because another game may be added later.

Reusable lobby, matchmaking, table allocation, or generic match-session capabilities should be introduced only when a second real game requires them.

When common persistence becomes necessary, use a narrow common envelope for:

- game descriptor;
- lifecycle status;
- module-owned snapshot version and bytes;
- fairness/setup artifact version and bytes;
- common audit timestamps and digests.

Concrete modules may still own dedicated tables when indexed game-specific state or invariants require them. A single unrestricted JSON table for every game is not the default design.

## 10. Client strategy

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

## 11. Compatibility with Goal 0023

Goal 0023 remains a valid Doudizhu v1 implementation and golden vector. Extensibility work must not silently change its canonical bytes or digests.

The next extraction should:

- move or wrap only genuinely reusable deterministic-stream, rejection-sampling, and permutation behavior;
- preserve all Goal 0023 Doudizhu golden-vector outputs;
- keep Doudizhu card, deck, deal, and transcript semantics in the Doudizhu module;
- prove the generic core with a test-only non-Doudizhu module or sequence, not with a fake production game.

Any change to Goal 0023 canonical bytes requires a new Doudizhu algorithm version rather than modifying v1.

## 12. Non-goals

This architecture does not introduce:

- a visual game editor;
- user-uploaded executable game plugins;
- scripting engines;
- reflection-heavy dependency injection;
- a universal rule DSL;
- a universal card/tile/piece schema;
- a generic event-sourcing framework;
- microservices per game;
- a shared gameplay state machine that tries to model every game.

The extension point is a small, versioned Go business interface with explicit compile-time registration and concrete module ownership.
