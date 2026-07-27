# Fair Doudizhu v1 Product Requirements

## 1. Product intent

Fair Doudizhu is a private-room, three-player classic Doudizhu game whose shuffle inputs and completed-hand evidence can be independently verified after a hand finishes.

The first release is a social game, not a gambling product. It has no real-money stake, recharge, cash-out, redeemable balance, prize pool, paid matchmaking, or transfer of value between players.

## 2. v1 scope

The first playable product supports:

- exactly three human players;
- invite-only private rooms;
- one room owner and three fixed seats numbered `1`, `2`, and `3`;
- classic Doudizhu room and hand flow;
- ready/unready, owner-controlled hand start, reconnect, and completed-hand replay data;
- Commit-Reveal contributions from all three players;
- a server commitment fixed before client contributions;
- a public randomness beacon plan fixed before client reveals;
- post-hand verification evidence for accepted, cancelled, aborted, and expired hands.

The first release does not support:

- public matchmaking, ranking queues, tournaments, bots, spectators, or voice chat;
- money, recharge, cash-out, coupons redeemable for money, or tradable game assets;
- client-selected seats, client-supplied authoritative account IDs, or client-supplied permissions;
- more or fewer than three active seats;
- changing players or seat assignments during an active hand.

## 3. Room rules

### 3.1 Creation and ownership

Creating a room places the authenticated creator in seat `1` and makes that account the room owner. Ownership is a server-side property and is never accepted from a client payload.

The owner may start a hand only when:

1. all three seats are occupied;
2. all three players are ready;
3. no hand is active;
4. the supplied command version matches the current room version.

The owner cannot leave while guests remain. An owner leaving an otherwise empty room closes it. A guest may leave only while no hand is active.

### 3.2 Stable seats

Seats are stable for the lifetime of a hand. The server resolves a player's seat from authenticated room membership. Commands that contain an `accountId`, `actorId`, `ownerId`, or authoritative `seat` field are invalid at the transport boundary.

### 3.3 Hand boundary

Starting a hand snapshots the three room seats into the hand aggregate. Later room membership changes cannot alter that hand snapshot. When the hand reaches a terminal state, the room may clear its active-hand reference and begin a new ready cycle.

## 4. Hand lifecycle

A new hand begins in `FAIRNESS_COMMITTING` only after the server has locked:

- the hand ID and room ID;
- the three seat-to-account assignments;
- the server seed commitment;
- the secure reveal key ID;
- the public-beacon provider and round.

The lifecycle is:

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

Terminal alternatives are:

- `CANCELLED` — a normal pre-deal cancellation, such as a player failing to contribute before timeout;
- `ABORTED` — an integrity or operational failure requiring the hand to stop;
- `EXPIRED` — a configured deadline elapsed without an allowed continuation;
- `COMPLETED` — normal settlement completed.

Terminal hands never return to an active phase.

## 5. Fairness contribution flow

### 5.1 Client material

Each client generates:

- exactly 32 bytes from a cryptographically secure random source;
- the player's original phrase as UTF-8 text.

The original phrase and random bytes are uploaded only inside the secure envelope defined by `secure-envelope-v1`. The fairness random bytes are never reused as an encryption key, nonce, HPKE seed, session key, or database key.

### 5.2 Commit phase

During `FAIRNESS_COMMITTING`, each seat submits exactly one client commitment. Individual commits are not enough to advance the hand. The third accepted commitment advances the hand to `FAIRNESS_REVEALING`.

A commitment is bound to the hand ID, fixed seat, and contribution digest, so it cannot be replayed for another hand or seat.

### 5.3 Reveal phase

During `FAIRNESS_REVEALING`, each client submits its secure envelope containing the original phrase and 32-byte random value. The application layer will:

1. authenticate the connection and derive the actor;
2. resolve room membership and seat;
3. validate command replay metadata and expected aggregate version;
4. decrypt the secure envelope;
5. normalize and validate the plaintext contribution;
6. compute the contribution digest and commitment;
7. pass only verified domain evidence and an encrypted-record reference to the hand aggregate.

The domain aggregate stores the contribution digest and encrypted-record reference, not the raw phrase or random bytes. Raw contribution storage, access control, retention, and encryption at rest are separate infrastructure responsibilities.

The server must not broadcast another player's raw phrase, random bytes, contribution digest, server seed, or complete deck while a hand is active.

### 5.4 Public beacon

The provider and round are locked when the hand is created, before client reveals. After all three reveals are accepted, the server obtains the exact planned beacon value and proof. A value from another provider or round is rejected.

The public beacon prevents the server from knowing the final shuffle input when choosing its committed seed, provided the planned beacon round is generated after the commitments are locked.

### 5.5 Completion evidence

A completed hand will eventually expose enough non-sensitive evidence to reproduce the final shuffle and verify the deal. A cancelled, aborted, or expired hand must also remain auditable so the server cannot silently discard unfavorable commitments or beacon outcomes.

The exact deck, shuffle, bidding, play, and scoring algorithms are deliberately deferred from this goal and will be versioned separately.

## 6. Command safety and reconnect

Every state-changing command carries:

- a globally unique `commandId`;
- a monotonically increasing `clientSeq` for the authenticated account and aggregate stream;
- the target aggregate ID;
- `expectedVersion`;
- a versioned command name.

Transport security uses HTTPS/WSS. WSS protects the connection but does not replace application-level idempotency or ordering.

When persistence is added:

- a duplicate `commandId` returns the original result without a second state change;
- a reused or lower `clientSeq` is rejected unless it is the same persisted command result;
- a stale `expectedVersion` is rejected with the current aggregate version;
- accepted domain events and the command result are persisted atomically.

Reconnect restores the authoritative room and hand snapshot from the server. The client never advances a hand locally based only on an unacknowledged command.

## 7. Audit and privacy

The system will maintain an append-only logical history of accepted commands, domain events, fairness evidence, terminal reasons, and aggregate versions.

Default player-facing verification exposes hashes, commitments, beacon evidence, shuffle versions, and post-hand material needed for verification. Raw phrases are private contribution data and are not public by default. A player may be allowed to view their own original contribution under a separate authenticated policy.

Operational logs must not contain access tokens, raw phrases, client random bytes, decrypted secure-envelope plaintext, server seeds before disclosure, complete active decks, or private keys.

## 8. Deferred decisions

The following require later versioned decisions:

- exact card encoding and deck order;
- deterministic random stream and shuffle algorithm;
- dealing order and landlord-card placement;
- bidding, doubling, spring, scoring, and settlement rules;
- contribution plaintext schema and phrase normalization implementation;
- public-beacon provider implementation and timeout policy;
- database schema, encrypted storage, key rotation, retention, and deletion policy;
- HTTP/WSS route composition and Cocos gameplay screens.
