# Goal 0029: Doudizhu Final Evidence Retrieval and Independent Verification

## Status

- State: in_progress
- Started: 2026-07-29
- Completed: Not yet.
- Blockers: None.

## Goal

Make archived Doudizhu terminal records readable by authorized participants and independently verifiable after the live hand has been removed. Reconstruct the immutable `gamecore.FinalRecord`, validate its digest and descriptor, verify the Doudizhu completed or aborted payload, and return a stable verification report without restoring active gameplay state.

## Scope

ChatGPT owns architecture, implementation, tests, failure diagnosis and fixes, documentation, verification, commits, and pushes directly to `main`.

Deliver:

1. Add a read side to the immutable MySQL final-record archive.
2. Reconstruct a validated `gamecore.FinalRecord` from persisted descriptor, status, version, payload, and digest.
3. Return a stable not-found error and reject corrupted or conflicting stored rows.
4. Add a pure Doudizhu terminal verifier for:
   - `doudizhu-live-completed-v1`;
   - `doudizhu-live-terminal-v1` aborted records.
5. Verify completed evidence:
   - final-record status, descriptor, version, and digest;
   - setup artifact and setup digest;
   - canonical fairness transcript and transcript digest;
   - transcript/setup identity and deterministic shuffle/deal consistency;
   - bidding landlord and winning score;
   - accepted physical-card history and final hands;
   - winner and settlement recomputation;
   - zero-sum points and completed-payload identity.
6. Verify aborted evidence without pretending it is a completed hand:
   - status and reason;
   - setup and fairness evidence;
   - current hands contain exactly the remaining physical cards represented by the aborted payload;
   - no settlement is manufactured.
7. Add an application query that authorizes only seated participants before returning evidence or a verification report.
8. Keep final-record reads side-effect free: no live-hand recreation, no Redis card state, no command replay, and no archive mutation.
9. Document privacy and disclosure boundaries for completed versus aborted records.
10. Run focused formatting, ordinary tests, `-race`, `go vet`, real MySQL integration, full repository, build, Compose, and production runtime verification.

Outside this goal:

- public HTTP/WSS gameplay and verification routes;
- all-pass replacement-hand orchestration;
- room cancellation, bidding/play timeout policy, or scheduled cleanup;
- Cocos verification screens;
- active-hand persistence, crash restoration, or cross-instance migration;
- rankings, balances, prizes, or money-like value.

## Constraints

- All edits and commits go directly to `main`; do not create a branch or pull request.
- Archived records remain immutable.
- Verification uses the archived bytes and existing versioned rules; it must not trust caller-supplied winner, score, pattern, digest, or seat claims.
- Only participants may retrieve private terminal evidence through the application boundary.
- A verifier failure must not modify the archive or any live game.
- Goal 0023 deterministic vectors and Goal 0028 settlement rules remain unchanged.

## Acceptance Criteria

- MySQL archive read reconstructs the exact original final-record digest.
- Missing rows return a stable not-found error.
- Tampered descriptor, status, version, payload, or digest is rejected.
- A completed Goal 0028 archive verifies successfully and recomputes the same settlement.
- Tampered setup, transcript, action history, final hands, winner, or settlement is rejected.
- An aborted Goal 0025 archive verifies under aborted semantics and does not produce points.
- Outsiders cannot retrieve terminal evidence through the application service.
- Reads and verification make no gameplay or archive writes.
- Focused and full verification remain green.

## Working State

### Completed

- Goal 0028 normal settlement and completed archive are completed and archived.
- MySQL final-record retrieval and exact reconstruction are implemented.
- Canonical fairness transcript decoding and verification are implemented.
- Completed and aborted Doudizhu evidence verification is implemented.
- Participant-authorized application retrieval is implemented.
- Focused formatting, ordinary tests, `-race`, integration-tag compilation, and `go vet` passed in run `30426835659`.

### In progress

- Real MySQL integration and final full-repository verification.

### Remaining

- Full repository, build, Compose, production runtime verification, completion report, and archive.

## Completion Report

Pending.
