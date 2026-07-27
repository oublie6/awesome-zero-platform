# Goal 0020: Fair Doudizhu Product Protocol and Domain Core

## Status

- State: in_progress
- Started: 2026-07-28
- Completed: Not yet.
- Blockers: None.

## Goal

Define the first executable Fair Doudizhu product contract and implement a pure Go domain core for three-player private rooms, fairness orchestration, and hand lifecycle before adding persistence or transport adapters.

## Scope

ChatGPT owns architecture, implementation, tests, failure fixes, CI integration, commits, and pushes directly to `main`.

Deliver:

1. A v1 product requirement document for classic three-player private-room Doudizhu with no money, recharge, cash-out, public matchmaking, bots, spectators, or voice.
2. A versioned command/event protocol defining command IDs, client sequence numbers, expected aggregate versions, actor derivation, idempotent replay semantics, and stale-command rejection.
3. A domain architecture document defining room, seat, hand, fairness contribution, public-beacon, and audit boundaries.
4. A pure Go business module under `server/business/doudizhu/domain` with room and hand aggregates, explicit state machines, domain events, validation errors, and repository-facing snapshots.
5. Unit and race tests covering valid lifecycle paths, invalid transitions, fixed three-seat invariants, fairness commit/reveal progression, terminal states, and optimistic-version checks.

The following remain outside this goal:

- card representation, shuffle implementation, dealing, bidding rules, hand-pattern validation, turn rules, scoring, and settlement calculations;
- HTTP or WebSocket handlers, public-key endpoints, secure-envelope decryption integration, authentication composition, database schemas, Redis coordination, and repositories;
- public-beacon provider adapters, production key management, encrypted-at-rest contribution storage, replay-result persistence, Admin UI, and Cocos gameplay UI.

## References

- `AGENTS.md`
- `docs/requirements/fair-doudizhu-v1.md`
- `docs/architecture/fair-doudizhu-domain.md`
- `docs/api/fair-doudizhu-protocol-v1.md`
- `docs/architecture/secure-envelope-v1.md`

## Acceptance Criteria

- The product scope is explicit and does not imply gambling or real-money functionality.
- A room has exactly three stable seats and derives ownership and actor identity from trusted server context rather than client-supplied account or seat fields.
- The protocol requires `commandId`, `clientSeq`, `expectedVersion`, aggregate identifiers, and versioned command names; duplicate commands return the original result once persistence is added, while stale or out-of-order commands are rejected.
- The room aggregate enforces join, leave, ready, and start-hand invariants without depending on transport or storage packages.
- The hand aggregate implements the documented fairness and gameplay lifecycle, including `FAIRNESS_COMMITTING`, `FAIRNESS_REVEALING`, `WAITING_PUBLIC_BEACON`, `DEALING`, `BIDDING`, `PLAYING`, `SETTLING`, `COMPLETED`, `CANCELLED`, `ABORTED`, and `EXPIRED`.
- Fairness commits and reveals are one-per-seat, bound to the active hand, and cannot be submitted in the wrong phase.
- Aggregate versions increase only after accepted state changes, expected-version checks fail closed, and emitted events carry the resulting version.
- Terminal hands cannot return to active states.
- The business-domain package uses only the Go standard library and does not import `foundation`, `platform`, transport, SQL, or Redis packages.
- `go test -count=1 -p 1 -parallel 1 ./business/doudizhu/domain` passes.
- `go test -race -count=1 -p 1 -parallel 1 ./business/doudizhu/domain` passes in an environment with the race detector.
- Existing repository formatting, module, generated-code, Admin Web, integration, HTTPS, WSS, and Compose checks remain unchanged and green when GitHub Actions is available.
- Final verification evidence, commit SHA, push result, intentionally deferred work, and unavailable environment-specific checks are recorded honestly.

## Working State

### Completed

- Goal boundaries, product direction, security assumptions, and repository architecture rules were reviewed.
- Goal 0019 secure-envelope and Cocos foundation is complete on `main`.

### In progress

- Defining the product, protocol, and domain documents.
- Implementing and testing the pure Go room and hand aggregates.

### Remaining

- Integrate files into the repository, inspect the final diff, and run all available checks.
- Record verification evidence and complete the goal.

### Verification status

- Baseline main: `7796e86faf89e0f919e359cb142fab06dfaa598c`.
- Baseline Goal 0019 verification: PR CI run `30287493380` and runtime run `30288017127` succeeded.
- Goal 0020 implementation verification pending.

## Completion Report

Pending.
