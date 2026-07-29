# Goal 0031: Authenticated Doudizhu HTTP API

## Status

- State: in_progress
- Started: 2026-07-29
- Completed: Not yet.
- Blockers: None.

## Goal

Expose the completed Doudizhu application, live runtime, lifecycle supervisor, and final-evidence verifier through a strict authenticated HTTP contract. Keep transport concerns separate from game rules, derive the actor from the access token, preserve durable and live-command idempotency, and return live payloads as structured JSON rather than Base64.

## Scope

Deliver:

1. A shared versioned command dispatcher usable by HTTP and later WSS.
2. Durable room/fairness commands backed by the existing MySQL command protocol.
3. Live bid/play/pass/cancel commands with short-lived in-memory request replay protection.
4. Authenticated public view, private view, and final evidence endpoints.
5. Strict JSON decoding, trim-clean identifiers, supported command whitelist, and bounded replay configuration.
6. Raw JSON live payload transport and copy isolation.
7. A production Doudizhu composition root with MySQL store/archive, live directory, seed vault, reveal-key opener, contribution protection, HMAC beacon proof verification, lifecycle supervisor, and shutdown cancellation.
8. Configuration and environment validation with the feature disabled by default.
9. Focused, real integration, full repository, Compose, and production runtime verification.

Outside this goal:

- WSS commands, broadcasts, and reconnect snapshots;
- client UI;
- active-hand persistence, process-crash restoration, or cross-instance migration;
- rankings, balances, prizes, or money-like value.

## Acceptance Criteria

- Every endpoint requires a valid authenticated account and never accepts a caller-supplied seat or actor.
- Unknown fields, trailing JSON, unsupported versions/types, whitespace-padded IDs, stale live versions, and replay conflicts fail closed.
- Durable duplicate command IDs use existing database idempotency.
- Concurrent duplicate live requests execute once and return copy-isolated identical results.
- Public/private views and live results encode their payload as JSON objects, not Base64 strings.
- All-pass and cancellation continue through Goal 0030 lifecycle supervision.
- Final evidence remains participant-only and side-effect free.
- The production composition root shuts down the lifecycle goroutine before database resources.
- Full verification remains green.

## Working State

### Completed

- Goals 0029 and 0030.

### In progress

- Shared dispatcher, HTTP handlers, configuration, production composition, and tests.

### Remaining

- Focused and final verification, completion report, and archive.

## Completion Report

Pending.
