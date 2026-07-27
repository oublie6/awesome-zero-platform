# Goal 0020: Fair Doudizhu Product Protocol and Domain Core

## Status

- State: completed
- Started: 2026-07-28
- Completed: 2026-07-28
- Blockers: None.

## Goal

Define the first executable Fair Doudizhu product contract and implement a pure Go domain core for three-player private rooms, fairness orchestration, and hand lifecycle before adding persistence or transport adapters.

## Scope

ChatGPT owned architecture, implementation, tests, failure fixes, repository integration, commits, and pushes directly to `main`.

Delivered:

1. A v1 product requirement document for classic three-player private-room Doudizhu with no money, recharge, cash-out, public matchmaking, bots, spectators, or voice.
2. A versioned command/event protocol defining command IDs, client sequence numbers, expected aggregate versions, actor derivation, idempotent replay semantics, stale-command rejection, secure reveal envelopes, and event ordering.
3. A domain architecture document defining room, seat, hand, fairness contribution, public-beacon, audit, replay, and repository boundaries.
4. A pure Go business module under `server/business/doudizhu/domain` with room and hand aggregates, explicit state machines, domain events, validation errors, fairness commitments, repository-facing snapshots, and optimistic-version checks.
5. Unit and race tests covering valid lifecycle paths, invalid transitions, fixed three-seat invariants, owner constraints, fairness commit/reveal progression, commitment mismatch, public-beacon plan mismatch, terminal states, and stale versions.

The following remain intentionally deferred:

- card representation, deterministic shuffle, dealing, bidding rules, hand-pattern validation, turn rules, scoring, and settlement calculations;
- HTTP or WebSocket handlers, public-key endpoints, secure-envelope decryption integration, authentication composition, database schemas, Redis coordination, idempotency persistence, and repositories;
- public-beacon provider adapters, production key management, encrypted-at-rest contribution storage, Admin UI, and Cocos gameplay UI.

## References

- `AGENTS.md`
- `docs/requirements/fair-doudizhu-v1.md`
- `docs/architecture/fair-doudizhu-domain.md`
- `docs/api/fair-doudizhu-protocol-v1.md`
- `docs/architecture/secure-envelope-v1.md`

## Acceptance Results

- Product scope is explicitly social and non-gambling; no real-money or value-transfer capability is implied.
- A room owns exactly three stable seats. Creation fixes the owner in seat 1, joining assigns the first empty seat, and no aggregate method trusts a client-declared seat or actor.
- The protocol requires `commandId`, `clientSeq`, `expectedVersion`, aggregate identifiers, and versioned command names, and documents original-result replay plus stale/out-of-order rejection for the future application layer.
- The room aggregate enforces join, leave, ready, owner, full-room, active-hand, start-hand, and finish-hand invariants without transport or persistence dependencies.
- The hand aggregate implements `FAIRNESS_COMMITTING`, `FAIRNESS_REVEALING`, `WAITING_PUBLIC_BEACON`, `DEALING`, `BIDDING`, `PLAYING`, `SETTLING`, `COMPLETED`, `CANCELLED`, `ABORTED`, and `EXPIRED`.
- Client commitments are bound to the active hand ID, server-resolved seat, and contribution digest. Commits and reveals are one-per-seat and phase constrained; reveal verification uses constant-time comparison.
- Public-beacon provider and round are immutable hand metadata. A mismatched beacon is rejected without changing phase, version, or snapshot data.
- Every accepted domain event increments the aggregate version exactly once; rejected commands leave state and version unchanged; multi-event commands use consecutive versions.
- Completed, cancelled, aborted, and expired hands reject later mutations.
- The domain package imports only the Go standard library and contains no foundation, platform, transport, SQL, or Redis dependency.
- Formatting, unit tests, race tests, and `go vet` passed for the domain package.
- The committed diff contains only Goal 0020 documents, the new business-domain package, tests, and README navigation updates.

## Verification Evidence

- Baseline main: `7796e86faf89e0f919e359cb142fab06dfaa598c`.
- Goal definition commit on `main`: `e59383dc8731accfd57b9eea99e17efe91487cf6`.
- Implementation commit on `main`: `bf79259d710d2befaf0456fcdf100724cd35f4a6`.
- Local verification environment: `go version go1.23.2 linux/amd64`. The new package uses only standard-library APIs compatible with the repository's Go 1.25.8 module.
- `go test -count=1 -p 1 -parallel 1 ./business/doudizhu/domain` succeeded.
- `go test -race -count=1 -p 1 -parallel 1 ./business/doudizhu/domain` succeeded.
- `go vet ./business/doudizhu/domain` succeeded.
- `gofmt` reported no unformatted domain files.
- A dependency-boundary scan found no imports of `foundation`, `platform`, SQL, or Redis packages.
- GitHub compare reported the implementation exactly one commit ahead of the Goal definition and listed only the expected 16 changed files.
- The connected GitHub App moved `main` by fast-forward successfully. As with the previous connector Git-object updates, this ref update emitted no `push` Actions run and no combined commit status. The repository-wide generated-code, Admin Web, integration, HTTPS, WSS, and Compose jobs were therefore not re-executed in this turn; none of their source, configuration, schema, or deployment inputs changed.

## Completion Report

Goal 0020 is complete. Fair Doudizhu now has an explicit non-gambling v1 product boundary, a versioned command/event contract, documented trust and persistence boundaries, and a tested pure Go domain core for three-player rooms and the complete fairness-to-terminal hand lifecycle.

The next implementation goal should add the application and persistence layer around these aggregates: transactional command idempotency, monotonic client sequences, room/hand repositories, encrypted contribution record references, event outbox, and secure-envelope reveal orchestration. Card, shuffle, bidding, play, and scoring rules should remain a separate versioned goal rather than being mixed into persistence work.
