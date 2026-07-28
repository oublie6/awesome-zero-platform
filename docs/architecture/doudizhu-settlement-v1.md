# Doudizhu Settlement v1

## 1. Product boundary

`doudizhu-settlement-v1` defines non-monetary integer points for the social-game release. Points are evidence of one hand's result only. They are not money, stored value, prizes, withdrawable balances, tradable assets, or a stake.

The pure calculator lives in:

```text
server/business/doudizhu/domain/settlement
```

It consumes the immutable completed `playing.Snapshot`, the server-owned landlord seat, and the accepted winning bid. It does not access databases, Redis, clocks, accounts, network connections, or mutable live hands.

## 2. Base score and multipliers

The winning bid `1`, `2`, or `3` is the base score.

The multiplier starts at `1` and doubles for each of these independently:

- every accepted `BOMB` play;
- the accepted `ROCKET` play;
- spring or anti-spring, when applicable.

There is no voluntary player doubling in v1.

The final stake is:

```text
finalStake = winningBid × 2^(bombCount + rocketCount + springFlag + antiSpringFlag)
```

Spring and anti-spring cannot both be true.

## 3. Spring

Spring is true when:

- the landlord wins; and
- neither farmer made any accepted `PLAY` action.

Farmer pass actions do not prevent spring. Only an accepted farmer play does.

## 4. Anti-spring

Anti-spring is true when:

- a farmer wins; and
- the landlord made exactly one accepted `PLAY` action during the hand.

The landlord may have received turns and passed later; the rule counts accepted plays, not turn opportunities.

## 5. Zero-sum points

For a landlord win:

```text
landlord = +2 × finalStake
each farmer = -finalStake
```

For a farmer win:

```text
landlord = -2 × finalStake
each farmer = +finalStake
```

The three seat points always sum to zero.

## 6. Input verification

The calculator rejects:

- landlord or winner seats outside `1..3`;
- winning bid outside `1..3`;
- a non-completed playing snapshot;
- a completed snapshot that still has a current seat;
- missing, reordered, or incorrectly numbered actions;
- a final action that is not the winner's accepted play;
- malformed play patterns;
- pass actions carrying cards or patterns;
- reuse of one physical card in multiple accepted plays;
- impossible bomb or rocket counts;
- multiplier or point overflow.

Each stored play pattern is recomputed from the physical cards. A caller cannot obtain a multiplier by labeling ordinary cards as a bomb or rocket.

## 7. Atomic normal completion

The winning play is processed on a clone of the authoritative `playing.State` and cloned hands.

Before the live game commits that clone, it successfully creates:

1. the completed playing snapshot;
2. the settlement result;
3. the client command result;
4. the immutable completed final payload.

Only after all four exist does the live game replace its authoritative in-memory state, clear protected runtime fairness material, and return a terminal `gamecore.CommandOutcome`.

This prevents partial states such as cards being removed without a settlement or final record.

## 8. Completed final payload

`doudizhu-live-completed-v1` contains:

- hand ID, completed status, and final live version;
- original setup artifact and digest;
- canonical fairness transcript and digest;
- complete bidding snapshot;
- complete accepted play/pass history;
- final hands and public landlord cards;
- landlord seat, winning bid, and winner;
- complete multiplier breakdown and zero-sum seat points.

The payload is deterministic for one completed live version and contains enough evidence for a later verifier to reproduce shuffle/deal evidence, inspect accepted gameplay, and recompute settlement.

## 9. Archive ordering and retry

`gamecore.LiveDirectory.Apply` receives the terminal outcome and creates a `FinalStatusCompleted` record.

```text
winning play accepted
  -> settlement and final payload frozen
  -> completed FinalRecord created
  -> archive succeeds
  -> live instance removed
```

If the archive fails:

```text
winning play is not replayed
  -> same FinalRecord remains pending
  -> all further commands are rejected as finalization pending
  -> RetryArchive submits the same digest
  -> success removes the instance
```

The application must not report ordinary gameplay success before this archive boundary succeeds. A retry is a final-record retry, not a repeat of the winning play.

## 10. Non-terminal persistence policy

Non-winning play and pass commands remain memory-only. They do not write command rows, Hand snapshots, outbox rows, archives, or Redis card state.

Goal 0029 will add player-facing final-evidence verification and complete non-normal terminal orchestration. Goal 0030 will expose authenticated HTTP/WSS gameplay routes without changing this archive ordering.
