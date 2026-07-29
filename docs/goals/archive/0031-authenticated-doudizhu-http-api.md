# Goal 0031: Authenticated Doudizhu HTTP API

## Status

- State: completed
- Started: 2026-07-29
- Completed: 2026-07-29
- Blockers: None.

## Goal

Expose the completed Doudizhu application, live runtime, lifecycle supervisor, and final-evidence verifier through a strict authenticated HTTP contract while keeping transport concerns separate from game rules.

## Delivered

1. Added one versioned command dispatcher shared by HTTP and the following realtime stage.
2. Exposed durable room and fairness commands through the existing MySQL command protocol.
3. Exposed live bid, play, pass, and cancellation commands with bounded in-memory request replay protection.
4. Added authenticated public view, private view, and participant-only final-evidence endpoints.
5. Enforced strict JSON, trim-clean identifiers, supported command types, exact expected versions, and replay conflict rejection.
6. Returned live payloads as structured JSON instead of Base64 strings and preserved copy isolation.
7. Added the production Doudizhu composition root: MySQL store and archive, live directory, seed vault, reveal-key opener, contribution protection, HMAC beacon proof verification, lifecycle supervision, and orderly shutdown.
8. Added environment/configuration validation and kept the feature disabled by default until explicitly configured.
9. Preserved active-hand in-memory authority and terminal-only persistence.

## Verification

Focused run `30442093320` passed:

- Go 1.25.8 formatting;
- Doudizhu HTTP dispatcher/config/bootstrap/runtime ordinary tests;
- focused race tests;
- integration-tag compilation;
- `go vet`.

Final run `30442521770` passed:

- module and generated-code cleanliness;
- full formatting checks;
- Goal 0031 and all repository Go tests;
- Security, Admin, and realtime race tests;
- server build and local/production Compose validation;
- Secure Envelope and signed-manifest interoperability;
- Cocos deterministic-randomness policy and Admin Web build;
- real MySQL 5.7 and Redis integration;
- production HTTP, authenticated WebSocket, HTTPS, WSS, bootstrap/login, and cleanup acceptance.

Final status: `goal0031/final-full: success`.

## Main-only workflow

All implementation, tests, documentation, verification setup, and cleanup were committed directly to `main`. No feature branch, validation branch, pull request, or merge workflow was used. Temporary focused, final, staging, and source-snapshot files were removed before archive.

## Deferred

WSS Doudizhu commands, per-participant broadcasts, versioned reconnect snapshots, and terminal evidence recovery are delivered by Goal 0032. Active-hand persistence, process-crash restoration, cross-instance migration, client UI, rankings, balances, prizes, and money-like value remain outside the service v1 boundary.

## Completion Report

Goal 0031 is complete. The service now exposes one authenticated, strict, idempotent HTTP boundary over the durable and in-memory Doudizhu application model.
