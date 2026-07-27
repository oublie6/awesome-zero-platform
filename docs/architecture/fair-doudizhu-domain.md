# Fair Doudizhu Domain Architecture

## 1. Boundary

Fair Doudizhu is the first product-specific module under `server/business`. It remains inside the modular monolith and may use platform capabilities only through explicit application-layer interfaces.

The intended future layout is:

```text
server/business/doudizhu/
  domain/          pure aggregates, value objects, rules, events
  application/     commands, queries, idempotency, transactions, ports
  infrastructure/  MySQL, Redis, secure contribution storage, beacon adapters
  transport/       HTTP and realtime mapping
```

Goal 0020 implements only `domain`. The package imports only the Go standard library.

## 2. Trust boundaries

```text
Cocos client
  -> HTTPS/WSS authentication and command envelope
  -> application command validation and idempotency
  -> secure-envelope decryption for reveal commands
  -> Doudizhu domain aggregate
  -> repository/event transaction
```

The domain trusts that the application supplies an authenticated `actor` account. It does not trust or accept a client-declared actor, owner, or seat. The aggregate resolves seats from its own membership snapshot.

The domain does not decrypt HPKE messages, parse transport JSON, access SQL/Redis, read system time, call a beacon service, or write logs.

## 3. Aggregate boundaries

### 3.1 Room aggregate

The room aggregate owns:

- room ID;
- room owner;
- three fixed seats;
- ready flags;
- room status;
- current active-hand ID;
- optimistic version.

Room statuses are:

```text
WAITING_PLAYERS
READY
HAND_ACTIVE
CLOSED
```

A room reaches `READY` only when all three seats are occupied and ready. Starting a hand snapshots the seats, resets ready flags, records the active hand ID, and moves the room to `HAND_ACTIVE`.

The room aggregate does not own hand fairness or gameplay state. The application creates a hand from `Room.HandSeats()` and later calls `Room.FinishHand()` after the hand is terminal.

### 3.2 Hand aggregate

The hand aggregate owns:

- hand and room IDs;
- immutable seat-to-account assignments;
- server commitment;
- secure reveal key ID;
- locked beacon provider and round;
- one client commitment and reveal record per seat;
- accepted beacon value reference;
- lifecycle phase and terminal reason;
- optimistic version.

The hand lifecycle is:

```text
FAIRNESS_COMMITTING
  -> FAIRNESS_REVEALING
  -> WAITING_PUBLIC_BEACON
  -> DEALING
  -> BIDDING
  -> PLAYING
  -> SETTLING
  -> COMPLETED
```

Alternative terminal transitions lead to `CANCELLED`, `ABORTED`, or `EXPIRED`. Terminal phases reject every later mutation.

## 4. Fairness values

### 4.1 Contribution digest

The future application layer will derive a `contributionDigest` from decrypted plaintext using a separately versioned canonical schema. The planned inputs include:

- protocol domain string;
- hand ID;
- server-resolved seat;
- 32-byte client secure random value;
- hash of the normalized original phrase.

The raw phrase and random value are not held by the domain aggregate. They are stored by an encrypted contribution repository, and the aggregate keeps only the digest plus an opaque record reference.

### 4.2 Client commitment

Goal 0020 fixes the domain commitment binding used by `ComputeClientCommitment`:

```text
SHA-256(
  UTF8("fair-doudizhu/client-commit/v1") || 0x00 ||
  U32BE(len(UTF8(handId))) || UTF8(handId) ||
  U8(seat) ||
  contributionDigest[32]
)
```

The commitment is therefore bound to the hand, seat, and contribution digest. The hand aggregate uses constant-time comparison when accepting a reveal.

### 4.3 Server commitment and beacon plan

A hand cannot be constructed without a non-zero server commitment, reveal key ID, beacon provider, and beacon round. These values become immutable hand metadata.

The aggregate accepts a beacon only in `WAITING_PUBLIC_BEACON` and only when provider and round exactly match the locked plan. Provider fetching and proof verification are application/infrastructure responsibilities; the domain records the verified digest and proof reference.

## 5. Version and event model

Each aggregate starts at version `1` after its creation event. Every accepted state change records one domain event and increments the aggregate version exactly once for that event.

A single command may produce multiple events. For example, the third accepted client commitment produces:

1. `doudizhu.hand.fairness-commit-accepted.v1`;
2. `doudizhu.hand.phase-changed.v1` to `FAIRNESS_REVEALING`.

Those events receive consecutive versions. Rejected commands produce no event and do not change the version.

Every mutation requires `expectedVersion`. A mismatch returns a typed version-conflict error with expected and actual versions before changing state.

Domain events contain no timestamp or delivery metadata. The application layer adds command ID, causation ID, correlation ID, actor, and authoritative occurrence time when persisting or publishing them.

## 6. Replay and transaction design

The application layer will eventually execute commands in one transaction:

1. authenticate and derive actor;
2. validate command envelope and expiration policy;
3. look up an existing result by `(actor, commandId)`;
4. validate monotonic `clientSeq` for the actor and aggregate;
5. load the aggregate snapshot with its version;
6. invoke the domain method with `expectedVersion`;
7. persist the new snapshot, events, sequence, and response atomically;
8. publish events after commit through an outbox.

Recommended uniqueness constraints are conceptually:

```text
UNIQUE(actor_account_id, command_id)
UNIQUE(aggregate_type, aggregate_id, actor_account_id, client_seq)
UNIQUE(hand_id, seat, contribution_phase)
```

A duplicate command returns its original stored result. It is not executed again. A new command with a reused sequence is rejected.

## 7. Snapshots and repository ports

`RoomSnapshot` and `HandSnapshot` are repository-facing values. Repository adapters may map them to database rows but must not expose their table implementation to another module.

Snapshots copy fixed arrays and beacon values so callers cannot mutate aggregate internals. Sensitive raw contribution plaintext is intentionally absent.

Future repository interfaces belong in the application layer, not in `domain`, because transaction and idempotency requirements span aggregate loading, command records, and event persistence.

## 8. Deferred rule modules

Later goals will add explicit versioned modules for:

- card and deck representation;
- deterministic shuffle and verification transcript;
- deal result;
- bidding state machine;
- legal hand-pattern and turn comparison;
- scoring and settlement;
- application command handlers and persistence;
- HTTP/WSS transport contracts and Cocos client reducers.

These additions must not weaken the fixed actor, seat, version, event, or fairness boundaries defined here.
