# Doudizhu Final Evidence Verification v1

## 1. Purpose

A live Doudizhu hand is removed from process memory only after its immutable final record is archived. This document defines how a seated participant can later retrieve that record and independently verify the evidence without recreating an active game.

Versioned implementation:

```text
server/business/gamecore/infrastructure/mysqlarchive
server/business/doudizhu/domain/carddeck
server/business/doudizhu/domain/livehand
server/business/doudizhu/application
```

## 2. Read boundary

The MySQL archive remains the write adapter used by `gamecore.LiveDirectory`, and also exposes a side-effect-free read adapter:

```text
LoadFinalRecord(ctx, instanceID) -> FinalRecord, archivedAt
```

The read path reconstructs the original `gamecore.FinalRecord` from persisted descriptor fields, status, version, payload and digest. It rejects:

- missing records;
- invalid descriptors or final statuses;
- zero or invalid final versions;
- malformed payloads;
- any digest mismatch.

Reading never updates the archive and never creates Redis or live-game state.

## 3. Participant authorization

`application.FinalEvidenceService` first loads the persisted Hand coordination snapshot and derives authorization from the server-owned seat assignments.

```text
authenticated account
    -> LoadHand(handID)
    -> account must match one of the three Hand seats
    -> LoadFinalRecord(handID)
    -> VerifyFinalRecord(record)
    -> return immutable payload copy and verification report
```

The caller cannot supply or override a seat number. An outsider is rejected before the archive lookup, so the query does not disclose whether a private final record exists.

## 4. Common evidence verification

Both completed and aborted records must pass all common checks:

1. the final record digest is valid;
2. the descriptor is exactly the versioned Doudizhu descriptor;
3. the JSON payload is strict, with unknown fields and trailing JSON rejected;
4. the payload hand ID, status and state version match the final-record envelope;
5. the setup artifact uses canonical unpadded base64url and its digest reconstructs the exact `SetupArtifact`;
6. the setup payload decodes under `doudizhu-card-deal-artifact-v1`;
7. the fairness transcript uses canonical unpadded base64url;
8. canonical transcript bytes are strictly parsed and byte-for-byte round-tripped;
9. server/client commitments, public beacon binding, shuffle seed, random stream, complete deck, deal and transcript digest are recomputed;
10. transcript identity, deck, hands, landlord cards and deal digest match the setup artifact.

Unknown versions fail closed.

## 5. Completed-record verification

`doudizhu-live-completed-v1` contains enough data to replay the whole game.

The verifier:

1. rebuilds the bidding state from the deterministic first bidder;
2. submits every archived score action under `doudizhu-score-bidding-v1`;
3. requires the recomputed bidding snapshot, landlord and winning score to match exactly;
4. starts the landlord with the original 17 cards plus the three landlord cards;
5. replays every archived play and pass under `doudizhu-playing-state-v1`;
6. checks turn order, physical-card ownership, pattern recognition, comparison and pass resets;
7. requires the replayed final snapshot and final physical hands to match exactly;
8. recomputes bombs, rocket, spring or anti-spring, multiplier, final stake and zero-sum points under `doudizhu-settlement-v1`;
9. requires winner, landlord, score and settlement to match the archived payload.

A successful report includes the record/setup/transcript digests, winner, landlord, seat points and final remaining-card counts.

## 6. Aborted-record verification

`doudizhu-live-terminal-v1` predates durable gameplay history. Its verifier therefore makes only claims supported by the archived bytes:

- status and non-empty bounded reason are valid;
- setup and fairness evidence pass the common verification;
- landlord-card metadata matches the original setup;
- every remaining card is valid and appears in at most one current hand;
- a card remains with its original seat, except that landlord cards may appear with at most one seat;
- no hand exceeds the maximum 20 cards;
- no settlement or points are manufactured.

The v1 aborted payload cannot prove the exact sequence of bids, plays or passes before abort because that history is not stored. A future terminal-payload version may add such history, but verification must not infer it retroactively.

## 7. Privacy and disclosure

Completed and explicitly aborted terminal records disclose the server seed, contribution digests, full shuffled deck and private cards because post-hand verification requires those values. Access through the application boundary is limited to seated participants.

The archive remains ordinary access-controlled persistence. Application field encryption is not added; deployment storage encryption, backup encryption and database access controls remain infrastructure responsibilities.

## 8. Failure behavior

- Archive not found: return the stable archive not-found error.
- Corrupt database row: fail before game-specific verification.
- Invalid Doudizhu payload: return `ErrInvalidFinalEvidence`.
- Verification failure: do not modify the archive or create a live hand.
- Caller cancellation or deadline: propagate the context failure from the read adapter.

## 9. Explicit exclusions

This version does not add:

- public HTTP or WSS evidence routes;
- spectator or administrator access to private evidence;
- active-hand restoration or replay as a live game;
- all-pass replacement-hand orchestration;
- cancellation or timeout scheduling;
- Cocos verification screens;
- rankings, balances, prizes or money-like value.
