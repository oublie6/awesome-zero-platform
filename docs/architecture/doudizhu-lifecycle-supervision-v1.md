# Doudizhu Lifecycle Supervision v1

## Purpose

`doudizhu-lifecycle-supervision-v1` coordinates terminal outcomes that are not produced by a normal winning play. It covers all-pass bidding, participant cancellation, bidding timeout, playing timeout, and retry after terminal archive failure.

The supervisor does not own cards or game rules. The active `livehand.Game` remains the only authority for bidding, current hands, turns, and play results. The supervisor owns only volatile deadlines and an immutable pending terminal command.

## Stable reasons

- `no_landlord` — all three seats passed during score bidding;
- `participant_cancelled` — a seated participant requested cancellation;
- `bidding_timeout` — the bidding deadline elapsed;
- `playing_timeout` — the playing deadline elapsed.

Callers cannot supply arbitrary persisted reasons through the supervisor.

## Terminal ordering

Every supervised termination uses the existing application terminal command path:

1. authorize the participant that caused or registered the lifecycle action;
2. load the persisted Hand and bind its current version;
3. create one internal command for actor `system:doudizhu-lifecycle`;
4. atomically mark the persisted Hand terminal, release its Room, write command result and outbox events;
5. archive the in-memory live hand;
6. remove the live hand and release the server seed only after archive success.

If step 5 fails, the supervisor keeps exactly the same command ID, sequence, timestamps, expected version, kind, and reason. Retry replays the completed database command and retries the same pending final record. It does not execute the last bid, play, pass, or terminal domain mutation again.

## Deadlines

Bidding and playing deadlines are memory-only leases. A successful relevant command replaces the previous deadline. A completed hand removes its deadline. The default policy is:

- bidding: 45 seconds;
- playing: 60 seconds;
- sweep: once per second.

Tests use an injected clock and call `Sweep` directly; they do not sleep.

## Concurrency

The supervisor serializes its maps with one mutex but never holds the mutex while calling MySQL, the application service, or the live runtime. Due hands are sorted by Hand ID before processing, which gives deterministic tests and logs. The application command store and `gamecore.LiveDirectory` remain responsible for database and per-hand mutation serialization.

## Process loss

Deadlines and pending archive retries are volatile in v1, matching the active-game memory-authority policy. A future recovery goal may persist lifecycle leases without persisting private hands. Goal 0030 does not add Redis hand state, database card snapshots, or cross-instance live migration.
