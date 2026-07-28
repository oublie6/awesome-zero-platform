# Doudizhu Score Bidding v1

## 1. Purpose

This document defines the first concrete gameplay phase for the in-memory Doudizhu live hand.

The phase starts after verified deterministic dealing has created a live game and the persisted coordination Hand has moved to `BIDDING`. It ends in one of two live states:

```text
BIDDING -> PLAYING
BIDDING -> NO_LANDLORD
```

Bids, bidding history, landlord selection, the landlord's 20-card hand, and the current playing seat are active private game state. They are authoritative only in the process-memory live game.

## 2. Version identities

```text
rules:             doudizhu-score-bidding-v1
first-seat derivation: doudizhu-bidding-first-seat-v1
command:           doudizhu-live-bid-command-v1
result:            doudizhu-live-bid-result-v1
public view:       doudizhu-live-public-view-v1
private view:      doudizhu-live-private-view-v1
```

The existing public/private view identities are retained because the payloads are additive and were not yet exposed by a production transport. Any future incompatible transport contract must introduce a new explicit version.

## 3. Score rules

Three fixed seats use integer scores:

```text
0 = pass
1 = one point
2 = two points
3 = three points
```

Rules:

1. each seat acts at most once;
2. actions follow circular seat order;
3. a positive score must be strictly greater than the current highest score;
4. pass never changes the current highest bid;
5. score `3` ends bidding immediately;
6. otherwise bidding ends after all three seats act;
7. the highest positive bidder becomes landlord;
8. three passes produce `NO_LANDLORD`.

There is no second bidding round, rob-landlord subphase, random retry, or implicit redeal in this version.

## 4. First bidder derivation

The first bidder is derived from the already committed deal digest. It is not selected by wall clock, UUID, process randomness, room owner, or client request.

```text
seed = SHA-256(
    UTF8("doudizhu/bidding-first-seat/v1")
    || 0x00
    || dealDigest[32]
)

stream = gamecore HMAC counter stream(seed)
firstBidder = gamecore.Uniform(stream, 3) + 1
```

`gamecore.Uniform` uses rejection sampling, so the mapping does not introduce modulo bias.

The committed golden input whose deal-digest bytes are `00,01,...,1f` selects seat `3`.

The derivation does not consume or alter the Goal 0023 shuffle stream. Card/deck/shuffle/deal/transcript bytes remain unchanged.

## 5. Pure bidding state

`server/business/doudizhu/domain/bidding` owns:

- first bidder;
- current bidder;
- highest score and bidder;
- ordered actions;
- completion flag;
- no-landlord flag;
- selected landlord.

The package depends only on Doudizhu card-deal digest types and generic deterministic-random primitives. It has no application, SQL, Redis, transport, clock, or logging dependency.

Rejected operations do not mutate the state. Snapshot action slices are copied.

## 6. Live command

The in-memory command payload is:

```json
{
  "v": "doudizhu-live-bid-command-v1",
  "score": 2
}
```

The authoritative actor position is not a JSON field. The runtime resolves it from the authenticated account and the server-owned Hand seat assignment.

The surrounding `gamecore.Command` supplies:

- authoritative actor position;
- exact expected live version;
- copied opaque payload bytes.

The live game rejects:

- empty or malformed JSON;
- unknown fields;
- trailing JSON values;
- unsupported command versions;
- scores outside `0..3`;
- stale live versions;
- actors other than the current bidder;
- non-increasing positive scores;
- commands after bidding has ended.

A rejected command leaves phase, live version, action history, hands, landlord cards, and playing seat unchanged.

## 7. Live result

A successful bid returns:

```json
{
  "v": "doudizhu-live-bid-result-v1",
  "handId": "hand_...",
  "stateVersion": 4,
  "phase": "PLAYING",
  "bidding": {
    "v": "doudizhu-score-bidding-v1",
    "firstBidder": 1,
    "currentBidder": 0,
    "highestScore": 2,
    "highestBidder": 3,
    "actions": [
      {"position": 1, "score": 1},
      {"position": 2, "score": 0},
      {"position": 3, "score": 2}
    ],
    "complete": true,
    "noLandlord": false,
    "landlord": 3
  },
  "landlordSeat": 3,
  "winningScore": 2,
  "playingSeat": 3,
  "requiresTermination": false
}
```

Every accepted bid increments the live-game version exactly once.

## 8. Landlord transition

When bidding selects a landlord:

1. the exact three Goal 0023 landlord cards are appended to that seat's current hand once;
2. the landlord hand size becomes `20`;
3. each farmer remains at `17`;
4. the live phase becomes `PLAYING`;
5. the landlord becomes the first playing seat;
6. the winning score is retained;
7. landlord cards become public.

No new shuffle, deal, card copy, or card identity is generated.

A post-bidding command cannot append the landlord cards again because `Apply` accepts bid commands only in `BIDDING`.

## 9. No-landlord transition

After three passes:

```text
phase = NO_LANDLORD
landlord = none
all hand sizes = 17
landlord cards remain hidden in active views
requiresTermination = true
```

The same deterministic deal remains attached to the live hand. The game does not silently reshuffle or start a replacement hand under the same ID.

The caller must invoke an existing server-controlled terminal operation. Explicit abort then publishes the setup artifact, Goal 0023 fairness transcript, current 17-card hands, and landlord cards in the immutable terminal record. The terminal reason is server-controlled, for example `NO_LANDLORD`.

If terminal archival fails, the existing `gamecore.LiveDirectory` finalization-pending behavior retains the same final record and retries without replaying the final pass or abort operation.

## 10. Public view

The public live view contains:

- hand ID, phase, and live version;
- fixed seat identities and remaining-card counts;
- setup/deck/deal digests;
- complete bidding snapshot;
- selected landlord and winning score when available;
- current playing seat when available;
- landlord cards only after landlord selection.

Before selection it contains neither landlord-card codes nor another player's hand cards.

After `NO_LANDLORD`, the landlord cards remain hidden until the terminal disclosure operation.

## 11. Private view

A private view contains the public projection plus exactly the authenticated viewer's current cards.

The seat is derived twice from trusted state:

1. the application loads the persisted coordination Hand and confirms membership;
2. the live coordinator maps the authenticated account through the concrete live game's immutable seat snapshot.

The client never supplies authoritative `ActorPosition` or viewer seat.

After landlord selection, the landlord sees 20 cards and each farmer sees 17.

## 12. Application and runtime boundary

The application method is intentionally not a database command transaction:

```text
authenticated account + hand ID + expected live version + score
-> load trusted Hand coordination snapshot
-> verify actor membership and persisted BIDDING phase
-> LiveHandRuntime.Bid
-> resolve actor position from live game
-> LiveDirectory.Apply
-> return copied versioned result
```

It does not call `Store.WithinCommand` and does not append a Room/Hand aggregate event.

Normal bid success or rejection therefore writes none of:

- `doudizhu_command_results`;
- `doudizhu_client_sequences`;
- `doudizhu_hands` current-card state;
- `doudizhu_outbox_events`;
- `game_final_records`;
- Redis card or bidding state.

The one database read is authorization and coordination lookup, not active-game authority.

## 13. Concurrency

`gamecore.LiveDirectory` owns one lock per active game instance.

- concurrent commands for one hand are serialized;
- two commands carrying the same expected version cannot both succeed;
- commands for separate hands use separate locks and can progress independently;
- returned command and view payloads are copied.

This protects the in-memory state even when multiple WebSocket handlers submit concurrently inside one server process.

## 14. Network retry boundary

Public HTTP/WSS bidding transport remains deferred.

When added, the transport may need a bounded, short-lived request ID cache so a client reconnect does not accidentally submit the same live command twice. That cache must not become the authority for cards or bids and must not convert ordinary bidding into continuous database persistence.

Exact expected live version, turn validation, and phase validation remain mandatory even when a transport retry cache exists.

## 15. Process-loss policy

Bidding state follows the Goal 0025 active-memory policy:

```text
unexpected process loss
-> active bidding/playing state is not restored
-> the hand is voided or handled by operational policy
```

No active-game snapshot, Redis reconstruction, action-log replay, or cross-instance migration is introduced by this version.

## 16. Deferred work

Deferred to later goals:

- public authenticated HTTP/WSS bidding endpoint;
- automatic terminal coordination and replacement-hand creation after `NO_LANDLORD`;
- doubling;
- legal play patterns and comparison;
- playing passes and turn advancement;
- bombs, spring rules, scoring, and settlement;
- durable replay UI and active-game crash recovery.
