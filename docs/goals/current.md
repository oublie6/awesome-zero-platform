# Goal 0031: Authenticated Doudizhu HTTP API

## Status

- State: completed
- Started: 2026-07-29
- Completed: 2026-07-29
- Blockers: None.

## Goal

Expose the completed Doudizhu application, live runtime, lifecycle supervisor, and final-evidence verifier through a strict authenticated HTTP contract. Keep transport concerns separate from game rules, derive the actor from the access token, preserve durable and live-command idempotency, and return live payloads as structured JSON rather than Base64.

## Scope

Delivered:

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

- Shared dispatcher and authenticated HTTP transport.
- Durable and live command routing with database and bounded in-memory idempotency.
- Authenticated public/private views and participant-only final evidence.
- Strict request validation and raw structured JSON payloads with copy isolation.
- Production composition, validated configuration, lifecycle shutdown ordering, and default-disabled feature switch.
- Focused, race, vet, full repository, client interoperability, real MySQL/Redis integration, build, Compose, and production runtime verification.

### In progress

- None.

### Remaining

- None within Goal 0031.

### Verification status

- Focused Goal 0031 verification passed on commit `ce39f5535b06f3e0f724346065a8baa644a483dd`.
- Final full verification passed for source commit `dc0492cb8f32dbde4f3e3a5608fc76eac49b805b` in GitHub Actions run `30442521770`.
- Module tidiness, generated-code cleanliness, formatting, focused ordinary tests, Goal 0031 race tests, and `go vet` passed.
- All Go tests and the security, Admin, and realtime race suites passed.
- Secure Envelope TypeScript/Go interoperability, Cocos client policy, and Admin Web build passed.
- The complete MySQL/Redis integration suite passed.
- Server build, local and production Compose validation, and production runtime acceptance passed.

## Completion Report

Goal 0031 is complete. The Fair Doudizhu application is now exposed through an authenticated versioned HTTP contract without moving product rules into transport code. Actors are derived exclusively from authenticated principals; durable commands retain database idempotency; live commands use bounded replay protection; public/private views and command results remain structured JSON; and final evidence stays participant-only and side-effect free.

The production composition root wires persistence, live runtime, secure contribution handling, reveal-key opening, beacon-proof verification, lifecycle supervision, and ordered shutdown. Configuration is validated and the feature remains disabled by default until explicitly enabled. All focused and repository-wide acceptance checks passed. Deferred WSS transport, client gameplay UI, crash restoration, cross-instance migration, and value-bearing features remain outside this goal.
