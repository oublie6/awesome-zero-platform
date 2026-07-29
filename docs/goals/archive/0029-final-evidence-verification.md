# Goal 0029: Doudizhu Final Evidence Retrieval and Independent Verification

## Status

- State: completed
- Started: 2026-07-29
- Completed: 2026-07-29
- Blockers: None.

## Goal

Make archived Doudizhu terminal records readable by authorized participants and independently verifiable after the live hand has been removed. Reconstruct the immutable `gamecore.FinalRecord`, validate its digest and descriptor, verify the Doudizhu completed or aborted payload, and return a stable verification report without restoring active gameplay state.

## Delivered

1. Added a read side to the immutable MySQL final-record archive with stable not-found and corrupt-record errors.
2. Reconstructed and revalidated the exact `gamecore.FinalRecord` descriptor, status, version, payload, digest, and archive timestamp.
3. Added strict canonical Doudizhu transcript decoding and byte-for-byte round-trip verification.
4. Added independent verification for completed records by recomputing setup, shuffle/deal, bidding, every accepted play/pass, remaining hands, winner, multipliers, and zero-sum settlement.
5. Added aborted-record verification without manufacturing a winner, settlement, or unrecorded action history.
6. Rejected tampered descriptor, digest, setup, transcript, card ownership, action history, final hands, winner, and settlement evidence.
7. Added a participant-authorized, side-effect-free application query; outsiders cannot retrieve private terminal evidence.
8. Added real MySQL coverage for archive write/read, participant authorization, outsider denial, exact payload recovery, and stored-digest corruption.
9. Documented completed-versus-aborted disclosure, privacy, immutable archive, and no-live-state-restoration boundaries.
10. Kept active gameplay state in memory and added no Redis card state, replayed commands, archive mutations, rankings, balances, prizes, or money-like value.

## Acceptance

- Focused verification run `30426835659` passed Go 1.25.8 formatting, ordinary tests, `-race`, integration-tag compilation, and `go vet`.
- Final verification run `30430719244` passed:
  - module tidy and generated-code checks;
  - all Goal 0029 and repository Go tests;
  - Security, Admin, and realtime race tests;
  - real MySQL 5.7 and Redis integration;
  - Secure Envelope and TypeScript/Go interoperability;
  - signed reveal-key manifest interoperability;
  - Cocos deterministic-randomness policy and Admin Web build;
  - server build and local/production Compose validation;
  - production container startup, HTTP, authenticated WebSocket, HTTPS, WSS, administrator bootstrap/login, and cleanup.
- Final status: `goal0029/final-full: success`.

## Main-only workflow

All implementation, tests, documentation, verification setup, fixes, and cleanup were committed directly to `main`. No feature branch, verification branch, or pull request was created. Temporary verification and source-snapshot workflows were removed after use.

## Completion Report

Goal 0029 is complete. Archived final evidence is immutable, participant-scoped, independently verifiable, and side-effect free. Public HTTP/WSS exposure and lifecycle timeout orchestration remain in Goals 0031 and 0030 respectively.
