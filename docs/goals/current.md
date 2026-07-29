# Goal 0032: Authenticated Doudizhu WSS Synchronization and Reconnect

## Status

- State: in_progress
- Started: 2026-07-29
- Completed: Not yet.
- Blockers: None.

## Goal

Expose the Goal 0031 dispatcher through the existing authenticated WebSocket hub, deliver per-participant state changes without leaking private cards, and support reconnect by returning the latest private live snapshot or immutable terminal evidence. Keep active gameplay authoritative only in memory and do not introduce active-hand persistence or generic public game topics.

## References

- `AGENTS.md`
- `docs/goals/archive/0031-authenticated-doudizhu-http-api.md`
- `docs/architecture/doudizhu-final-evidence-v1.md`
- `server/apps/app-api/internal/doudizhuapi`
- `server/platform/realtime`

## Deliverables

1. Register versioned `doudizhu.command` and `doudizhu.hand.sync` handlers on the existing authenticated realtime hub.
2. Derive the actor exclusively from the authenticated WebSocket connection context.
3. Reuse the exact Goal 0031 dispatcher and replay cache so HTTP/WSS retries share command semantics.
4. Broadcast accepted hand changes only to the three persisted participant accounts.
5. Send each participant an account-targeted private snapshot; never publish private state on a subscribable topic.
6. Return versioned full reconnect snapshots and a `notModified` result when the client already has the current live version.
7. Return participant-authorized immutable final evidence after the live hand has been removed.
8. Return stable realtime error envelopes for invalid requests, conflicts, forbidden access, missing resources, and internal failures.
9. Preserve bounded send queues, slow-consumer behavior, multiple connections per account, and orderly hub shutdown.
10. Add focused ordinary/race/vet tests, full repository and real integration verification, and production HTTP/WS/HTTPS/WSS acceptance.

## Constraints

- All changes go directly to `main`; no branch or pull request.
- Clients cannot submit actor or seat identity.
- Private snapshots are sent only with account-targeted delivery.
- Broadcast failure must not roll back or repeat an already accepted game command.
- Reconnect always derives current state from the authoritative live directory or immutable final archive.
- Do not add active-hand database/Redis snapshots, spectator topics, crash restoration, cross-instance migration, rankings, balances, prizes, or money-like value.
- Goal 0023 deterministic bytes, Goal 0028 settlement, Goal 0030 lifecycle ordering, and Goal 0031 HTTP contracts remain compatible.

## Acceptance Criteria

- An authenticated WSS command executes as the connection account and returns the same versioned command response as HTTP.
- Malformed, unknown-field, whitespace-padded, stale-version, conflicting-replay, outsider, and unsupported requests fail closed with stable error codes.
- Every accepted hand command sends a change event to all three participants and no outsider.
- Active participants receive distinct private snapshots containing only their own cards.
- A reconnect with an old version receives the latest full private snapshot.
- A reconnect with the current version receives `notModified=true` without duplicating the snapshot.
- A reconnect after completion or abort receives participant-authorized final evidence.
- Multiple connections for one account receive the same account event through the existing hub.
- Slow consumers remain bounded by the existing send queue behavior.
- Focused and final verification remain green.

## Agent Strategy

The primary agent owns architecture, integration, and final verification. Implement the bridge without modifying generic game rules or exposing private state through topics.

## Working State

### Completed

- Goals 0029, 0030, and 0031.
- Existing authenticated realtime transport, bounded queues, account sends, custom handlers, and production WSS acceptance.

### In progress

- Doudizhu realtime bridge, participant broadcasts, reconnect protocol, bootstrap registration, tests, and documentation.

### Remaining

- Focused verification, full repository/runtime verification, completion report, and archive.

### Verification status

- Not started for Goal 0032.

## Completion Report

Pending.
