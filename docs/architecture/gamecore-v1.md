# Gamecore v1 Contracts

## 1. Scope

`server/business/gamecore` provides small reusable business contracts for versioned randomized setup and active in-memory game execution. It is not a universal rule engine and does not define cards, tiles, moves, scoring, settlement, rooms, matchmaking, transport, or database schemas.

Production `gamecore` code uses only the Go standard library and has no dependency on a concrete game module.

## 2. Descriptor

A descriptor binds one exact server-side game implementation:

```go
type Descriptor struct {
    // immutable private fields
}

func NewDescriptor(
    gameID GameID,
    ruleset RulesetVersion,
    module ModuleVersion,
    fairness FairnessSuiteID,
    participantCount uint8,
) (Descriptor, error)
```

Every identifier:

- is non-empty UTF-8;
- has no surrounding whitespace;
- is at most 128 bytes.

Participant count is `1..64`.

The registry key is exactly:

```text
gameId + rulesetVersion + moduleVersion
```

`fairnessSuiteId` and participant count remain part of the immutable descriptor and are checked when a module or artifact is used.

## 3. Randomized setup registry

A concrete game implements:

```go
type RandomizedSetupModule interface {
    Descriptor() Descriptor
    GenerateSetup(FairnessMaterial) (SetupArtifact, error)
    VerifySetup(FairnessMaterial, SetupArtifact) error
}
```

Registration is explicit:

```go
registry, err := gamecore.NewRegistry(
    doudizhuSetupModule,
    futureGameSetupModule,
)
```

The registry:

- validates each descriptor;
- rejects duplicate exact keys;
- returns `ErrModuleNotFound` for unknown keys;
- remembers the descriptor observed during registration;
- rejects fairness material or artifacts whose descriptor differs from that registered identity;
- clones mutable participant slices before calling a concrete module.

There is no package-global automatic registration.

## 4. Fairness material

`FairnessMaterial` contains the cross-game evidence required by a randomized setup module:

```go
type FairnessMaterial struct {
    Descriptor       Descriptor
    InstanceID       InstanceID
    ServerSeed       Seed
    ServerCommitment Digest
    Participants     []ParticipantFairness
    Beacon           BeaconEvidence
    RevealKey        RevealKeyAudit
}
```

Participants must be present exactly once in stable positions `1..participantCount`. A concrete game defines what those positions mean.

The common canonical digest uses:

```text
domain  = "gamecore/fairness-material/v1"
version = "gamecore-fairness-material-v1"
```

It binds the full descriptor, instance ID, server material, ordered participant material, beacon evidence, and reveal-key audit reference.

This common digest is available to future games. The Doudizhu v1 adapter intentionally continues to use the already published Goal 0023 Doudizhu seed derivation so its golden vector does not change.

## 5. Deterministic primitives

### 5.1 Random byte stream

`NewStream(seed)` constructs an HMAC-SHA256 counter stream using:

```text
domain  = "gamecore/random-block/v1"
version = "gamecore-hmac-counter-v1"
counter = unsigned 64-bit big-endian
```

A zero seed is rejected. The stream is deterministic and does not use time, process entropy, floating point, global state, `math/rand`, or platform endianness.

### 5.2 Unbiased bounded integer

`Uniform(source, bound)` rejects `bound == 0` and uses rejection sampling:

```text
threshold = (-bound) mod bound
reject candidate < threshold
return candidate mod bound
```

### 5.3 Permutation

`Permutation(count, source)` builds indices `0..count-1` and runs Fisher–Yates from the final index down to one. Counts from 1 through 1,000,000 are supported.

The result is a permutation of indices only. The concrete game decides how those indices map to cards, tiles, positions, map cells, or other items.

## 6. Setup artifact

A game-owned randomized setup is wrapped by:

```go
type SetupArtifact struct {
    // immutable descriptor, version, payload and digest
}
```

Construction copies the payload. Accessors also return copies.

Its digest binds:

- full descriptor;
- participant count;
- artifact version;
- exact payload bytes.

The common layer does not parse the payload.

The Doudizhu adapter uses artifact version:

```text
doudizhu-card-deal-artifact-v1
```

Its payload includes the unchanged Goal 0023 algorithm versions, shuffle seed, deck digest, complete deck, deal digest, three hands, and landlord cards. Generation and verification call the existing `carddeck` package, preserving all Goal 0023 outputs.

## 7. LiveGame contract

A concrete active game implements:

```go
type LiveGame interface {
    Descriptor() Descriptor
    InstanceID() InstanceID
    Apply(Command) (CommandOutcome, error)
    View(ViewRequest) (GameView, error)
    Abort(reason string) (AbortOutcome, error)
}
```

The concrete implementation owns all mutable state and all payload schemas.

`Command` includes an authenticated actor position, optional expected version, and an opaque payload. The game validates action legality and state-version semantics.

`ViewRequest` requests either:

- a public projection; or
- a participant-specific private projection.

`gamecore` only validates the participant position range and copies the resulting payload. It does not inspect or redact game fields.

A non-terminal command must not include a final payload. A terminal command supplies the game-owned final payload used to construct the immutable final record.

## 8. LiveDirectory lifecycle

`NewLiveDirectory` requires a `FinalRecordArchive` implementation.

Adding an instance:

```go
err := directory.Add(expectedDescriptor, concreteGame)
```

The directory rejects:

- a nil game;
- an invalid instance ID;
- a descriptor mismatch;
- a duplicate active instance ID.

It stores the descriptor observed at registration rather than trusting a mutable value returned later by a faulty implementation.

### 8.1 Active command

```text
lookup instance
-> acquire that instance's lock
-> reject closed or finalization-pending state
-> validate actor position
-> copy command payload
-> call concrete game Apply
-> copy outcome payloads
-> return without archive when non-terminal
```

Locks are per instance. Separate active games can process simultaneously.

### 8.2 Terminal command

```text
call concrete game Apply once
-> construct completed FinalRecord
-> retain it as pending
-> call archive
-> on success remove and close the entry
-> on failure retain the exact pending record
```

While a final record is pending, further commands and aborts return `ErrFinalizationPending`.

`RetryArchive(instanceID)` retries the existing final record without invoking the game again.

### 8.3 Explicit abort

Abort follows the same lifecycle, but produces status `aborted`. The concrete game receives the reason once and produces its final abort payload.

## 9. Final record

A `FinalRecord` contains immutable:

```text
instanceId
descriptor
status: completed | aborted
final version
game-owned payload
digest
```

The digest uses domain:

```text
gamecore/final-record/v1
```

The archive port is:

```go
type FinalRecordArchive interface {
    Archive(FinalRecord) error
}
```

The runtime does not require SQL, Redis, HTTP, a clock, or a logging interface. A future database adapter must implement idempotency by instance ID and final-record digest.

## 10. Failure policy

In v1, active games exist only in process memory. Unexpected process loss may void them.

The following are intentionally absent:

- per-command database writes;
- current-hand storage in Redis;
- automatic snapshots;
- action-log replay;
- ownership transfer between processes;
- transparent live migration.

When recovery becomes a product requirement, snapshot or event-log ports can be added around concrete module-owned state encodings. They must not introduce a universal game-state schema.

## 11. Error categories

Reusable sentinel categories include:

- `ErrInvalidArgument`;
- `ErrUnsupportedVersion`;
- `ErrVerificationFailed`;
- `ErrDuplicateRegistration`;
- `ErrModuleNotFound`;
- `ErrInstanceNotFound`;
- `ErrInstanceExists`;
- `ErrFinalizationPending`;
- `ErrNotFinalizing`;
- `ErrArchiveFailed`.

Callers should use `errors.Is` rather than matching error text.

## 12. Deferred integration

Goal 0024 does not connect the generic live directory to the existing Doudizhu Room/Hand application flow. That requires real Doudizhu bidding and gameplay state, server-seed custody, a verified beacon adapter, final-record persistence, and transport-level view delivery.

Those pieces remain concrete integrations layered above the contracts documented here.
