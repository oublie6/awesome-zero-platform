# Doudizhu Play Rules v1

## 1. Boundary

`doudizhu-play-rules-v1` defines card-pattern recognition, comparison, playing turns, passes, remaining-card mutation, and gameplay winner detection for the concrete Doudizhu module.

The rules live under:

```text
server/business/doudizhu/domain/playing
```

The package is pure domain code. It does not know authenticated accounts, JSON transports, databases, Redis, clocks, sockets, or terminal persistence. The concrete `livehand.Game` owns physical hands and invokes the pure playing state only after it has verified live version, actor position, and card ownership.

## 2. Rank order

Ranks compare in this order:

```text
3 < 4 < 5 < 6 < 7 < 8 < 9 < 10 < J < Q < K < A < 2 < small joker < big joker
```

Suit never affects Doudizhu pattern strength.

## 3. Legal patterns

| Kind | Structure | Main rank |
|---|---|---|
| `SINGLE` | one card | that card rank |
| `PAIR` | two equal ranks | pair rank |
| `TRIPLE` | three equal ranks | triple rank |
| `TRIPLE_WITH_SINGLE` | triple plus one card | triple rank |
| `TRIPLE_WITH_PAIR` | triple plus a pair | triple rank |
| `STRAIGHT` | at least five consecutive singles | highest sequence rank |
| `CONSECUTIVE_PAIRS` | at least three consecutive pairs | highest sequence rank |
| `AIRPLANE` | at least two consecutive triples | highest body rank |
| `AIRPLANE_WITH_SINGLES` | consecutive triples plus one single wing per body triple | highest body rank |
| `AIRPLANE_WITH_PAIRS` | consecutive triples plus one distinct pair per body triple | highest body rank |
| `FOUR_WITH_TWO_SINGLES` | four equal cards plus two individual physical cards | four-card rank |
| `FOUR_WITH_TWO_PAIRS` | four equal cards plus two distinct pairs | four-card rank |
| `BOMB` | four equal cards | bomb rank |
| `ROCKET` | both jokers | big-joker rank marker |

### 3.1 Sequence restrictions

`2`, small joker, and big joker cannot appear in the body of a straight, consecutive pairs, or airplane.

All sequence ranks must be gap-free. A candidate can beat an existing sequence only when the pattern kind, card count, and sequence length are identical.

### 3.2 Airplane restrictions

Each airplane body rank contributes exactly three cards. A wing cannot use a body rank.

For `AIRPLANE_WITH_SINGLES`, v1 permits two physical single wings of the same non-body rank when those cards actually exist. For example, `33344455` is valid: the two fives are treated as two individual wings.

For `AIRPLANE_WITH_PAIRS`, every wing must be a complete pair and the pair ranks must be distinct.

### 3.3 Four-with-two restrictions

For `FOUR_WITH_TWO_SINGLES`, the two attached physical cards may have the same rank, so `333344` is valid.

For `FOUR_WITH_TWO_PAIRS`, the two pair ranks must be distinct. Eight cards containing two four-of-a-kinds are not interpreted as four-with-two-pairs.

## 4. Comparison

Comparison is deterministic:

1. `ROCKET` beats every non-rocket pattern.
2. No pattern beats `ROCKET`.
3. `BOMB` beats every ordinary pattern.
4. Bombs compare by bomb rank.
5. Ordinary patterns compare only when kind, sequence length, and card count match.
6. Matching ordinary patterns compare by main rank.

An invalid or fabricated pattern value is rejected rather than trusted.

## 5. Playing state

A `playing.State` owns only public turn state:

- current acting seat;
- current leading seat;
- current leading physical cards and recognized pattern;
- consecutive pass count;
- ordered public action history;
- gameplay completion flag and winner seat.

The physical private hands remain owned by `livehand.Game`.

### 5.1 Play

A play is accepted only when:

- gameplay is not complete;
- the actor is the current seat;
- the submitted cards form a legal pattern;
- when a leader exists, the pattern beats the leading pattern.

The concrete live game additionally proves every submitted physical card is currently held by the actor. It computes the remaining hand before invoking the state and commits the card removal only after the state accepts the play.

After a non-winning play, the next fixed seat acts. A play resets the pass count and becomes the new leader.

### 5.2 Pass

A player cannot pass when the trick is empty. The current leader therefore cannot open a trick by passing.

After one pass, the next seat acts. After both opponents pass:

- the leading play is cleared;
- the pass count resets;
- initiative returns to the prior leading seat.

The historical play and pass actions remain in the ordered history.

### 5.3 Gameplay completion

When an accepted play removes the actor's final card, the playing state records that seat as winner and clears the current acting seat. The live game enters `GAMEPLAY_COMPLETE` and rejects further play or pass commands.

`GAMEPLAY_COMPLETE` is not yet financial or score settlement. Goal 0028 will apply multipliers, determine landlord/farmer outcome, and move through settlement to normal terminal completion.

## 6. Live command protocol

### Play

```json
{
  "v": "doudizhu-live-play-command-v1",
  "cards": ["C3", "D3"]
}
```

### Pass

```json
{
  "v": "doudizhu-live-pass-command-v1"
}
```

The command envelope supplied by `gamecore.LiveDirectory` also carries the server-resolved actor position and exact expected live version. Client-supplied authoritative account or seat fields are not accepted.

Unknown JSON fields, trailing JSON, unsupported versions, malformed card codes, stale versions, wrong turns, cards not held, illegal patterns, and insufficient plays fail without mutation.

## 7. Views

The public view exposes:

- current live phase and version;
- remaining card count for each seat;
- landlord and bidding result;
- current seat, leader, leading cards and pattern;
- pass count;
- ordered play/pass history;
- winner after gameplay completion.

A private view adds only the authenticated viewer's current physical cards. It does not reveal another player's hand, server seed, or full deck.

Every returned view is copy-isolated from authoritative in-memory state.

## 8. Persistence boundary

Ordinary play and pass commands do not write:

- MySQL command rows;
- persisted Hand snapshots;
- outbox events;
- archive records;
- Redis card or turn state.

The active `livehand.Game` is the sole authority. The application loads persisted Hand membership only to authorize that the account belongs to the hand. The live game determines phase, turn, card ownership, and version.

The existing explicit abort path remains available and archives the current hands and fairness evidence. Normal gameplay completion and its complete replay/settlement record are intentionally completed in Goal 0028 and Goal 0029.

## 9. Concurrency

All commands for one hand pass through `gamecore.LiveDirectory.Apply` and are serialized by that live instance. Two concurrent commands using the same expected version cannot both mutate one hand. Separate hand instances remain independent.

## 10. Deferred transport

Goal 0027 does not expose public HTTP or WSS routes. Goal 0030 will provide authenticated network command envelopes, short-lived replay protection, broadcasts, private projections, and reconnect behavior without changing the in-memory gameplay authority established here.
