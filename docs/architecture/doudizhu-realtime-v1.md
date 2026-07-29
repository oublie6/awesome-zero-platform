# Doudizhu Realtime Protocol v1

## Purpose

Goal 0032 exposes the authenticated Goal 0031 Doudizhu dispatcher through the existing WebSocket hub without moving game rules into transport code or persisting active cards. HTTP and WSS commands share the same dispatcher, replay cache, authorization, lifecycle supervisor, live directory, and terminal archive.

## Message types

Client requests:

- `doudizhu.command` — payload is the exact Goal 0031 `doudizhu-command-request-v1` object.
- `doudizhu.hand.sync` — payload is `doudizhu-realtime-sync-v1` with `handId` and optional `knownVersion`.

Server responses and account-targeted events:

- `doudizhu.command.result` — the exact versioned dispatcher response.
- `doudizhu.hand.changed` — accepted command result identifying the changed hand.
- `doudizhu.hand.snapshot` — the authenticated account's private full snapshot, or `notModified=true` when its supplied version is current.
- `doudizhu.hand.evidence` — immutable participant-authorized terminal evidence after the live hand is gone.
- `doudizhu.error` — stable `INVALID_REQUEST`, `CONFLICT`, `FORBIDDEN`, `NOT_FOUND`, or `INTERNAL` envelope.

## Identity and authorization

The actor is always derived from `realtime.ConnectionContext.AccountID`, which is established by access-token authentication during the WebSocket upgrade. Request payloads contain no trusted actor or seat field. The dispatcher and final-evidence service perform the same membership and version checks used by HTTP.

## Delivery boundary

Private game state is never published to a generic or subscribable topic. After an accepted hand command, the bridge loads the immutable persisted seat membership and sends account-targeted events to exactly those three accounts:

1. each participant receives the same public command-change event;
2. while the game is active, each participant receives a separately generated private snapshot containing only that participant's cards;
3. after completion or abort, each participant receives immutable final evidence instead of recreating a live hand.

The existing hub delivers account events to every active connection for that account. Its bounded send queue and slow-consumer disconnect policy remain authoritative.

## Reconnect

`doudizhu.hand.sync` first asks the authoritative live directory for the caller's private view. When `knownVersion` equals the live version, the response contains `notModified=true` and no duplicate view. When the version is old or absent, the full current private snapshot is returned. If the live game no longer exists, the bridge queries immutable terminal evidence, which remains participant-only and side-effect free.

## Failure and idempotency

HTTP and WSS use the same dispatcher. Durable command IDs retain database idempotency; live request IDs retain the same bounded in-memory replay semantics. A failed account delivery does not roll back or repeat an already accepted command. Clients recover from missed events by issuing `doudizhu.hand.sync` with their last observed version.

## Persistence and process boundary

Active bids, cards, turns, passes, and versions remain memory-authoritative. Redis is not used for cards or live snapshots. MySQL stores durable room/fairness data and immutable terminal archives only. Process-crash restoration, cross-instance live migration, public spectators, and client UI are outside service v1.
