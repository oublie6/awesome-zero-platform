# Goal 0046 — Integrated playable Fair Doudizhu and B-grade recovery

## State
in_progress — authorized by user 2026-11-02. **This is a development plan, not completed functionality or acceptance.** Goal 0045 remains separately in_progress, with its historical Codex checkpoint PASS and remaining platform qualifications intact in `docs/goals/current.md`.

## Owner, boundaries, authority
- Go `server/business/doudizhu`: room, fairness, live hand, gameplay and B-grade durable state authority. MySQL is the durable truth for acknowledged mutations; Redis is not authoritative. Reuse go-zero infrastructure in existing module ownership boundaries.
- `server/apps/app-api` and `server/platform/authn`: authenticated identity-to-AccountID exchange, API/WSS composition; do not trust client-supplied account IDs or platform login codes.
- `clients/packages/*`: platform-agnostic transport/protocol/game coordination; keep Cocos/wx/tt dependencies in adapters.
- `clients/fair-doudizhu-cocos`: single playable room and table UI, Web/WeChat-first, future Douyin compatibility.
- Public contracts: versioned existing Go dispatcher `doudizhu-api-command-v1`, realtime events, private views; all wire additions explicitly versioned and tested.

## Complete authorized deliverables
1. Real platform-code login exchange / session validation, token lifetime and secure private credential management; no fake identity, no tokens in URLs; include local authorized dev login path only if correctly access-controlled.
2. Account-scoped rooms: create/join/leave/ready, seats, start hand, authoritative room refresh/reconnect and timeout/duplicate reconciliation; full Cocos interactive flow.
3. Fair lifecycle: server-bound commit/reveal key/AAD acquisition, secure entropy + HPKE, trusted beacon, deal, private hand and evidence verification, fail closed on missing security context.
4. Playable game: bidding, landlord determination, legal play/pass delegated to authoritative Go rules, turn indicators, cards/selection, clear error and version-retry handling, termination, score/evidence/result UI.
5. B-level MySQL recovery: append each accepted live bid/play/pass state transition and dedup + owner epoch/fence atomically **before acknowledgement**; bounded encrypted checkpoint and ordered replay restoring game state, ambiguity handling and owner fencing; address seed and other secret continuity; no plaintext player hands in routine logs.
6. Real 3-player repeatable E2E, reconnect and crash-injection: pre-/post-COMMIT kill, response loss, duplicate/out-of-order commands, competing owners, wrong epoch and replay mismatch; preserve every acknowledged action.
7. Performance and stability: bounded queues/workers/room memory, P50/P95/P99 measurements and transaction fsync budget on stated hardware, no unknown target claims.
8. WeChat Creator package, actual developer tool, approved HTTPS/WSS domains and real handset evidence where access is available. Record platform-owned AppID/domain/device blockers honestly. Preserve Web and Douyin build compatibility.

## Integration and release gates
- ChatGPT owns implementation and *continuous* CI + independent-of-mocks unit, property, race, MySQL integration, protocol and fault tests. Fix all known regressions before handing off.
- Checkpoint pushes to clean `main` are permitted; no Codex micro-handoffs. **Once all deliverables are implemented and self-tested**, give Codex one final complete-product acceptance SHA. Codex remains independent verifier; defects go back to ChatGPT for repair.
- Do not claim a playable product on a placeholder lobby or fake account; do not claim B-grade recovery without commit-before-ack and kill/replay proof.
- Hardware, production credential and release gates not available to the implementer are recorded as BLOCKED, not silently ignored.
- No destructive production data or unauthorized third-party account operations.

## Current verified facts and remaining work
- Goal 0045 protocol/Socket defects independently passed at `956ab2ab` (historical report retained).
- Existing Go application has durable room/fairness command execution; live bid/play/pass still use volatile `LiveDirectory.Apply` and are **not B-grade durable**.
- Existing Cocos entry has a menu and placeholder rooms; authenticated production login, playable table, fair reveal, and on-device full game are **not yet wired**.
- `clients/packages/doudizhu-client/src/room.ts` is an initial checkpoint with versioned durable room command coordination; integration and final acceptance outstanding.
- Plan: implement auth/room UI composition, complete fairness, add live action persistence/replay, build playable table, verify three-player + fault/latency, then hand off final SHA.

## Development checkpoint — 2026-11-02 (pending CI/independent acceptance)
- Added `secureRoomCommandIdentity`: cryptographically sourced 128-bit injected entropy, asynchronous platform port, fail-closed on missing/wrong-length random bytes. Room coordinator validates both transport and durable versioned Go response envelopes; unknown outcome requires authoritative reconciliation. Source tests added for entropy and malformed acknowledgments.
- Added `GameOnlineController` at `clients/fair-doudizhu-cocos/assets/scripts/game/`, composing trusted `GameSession` platform-code exchange, authenticated WSS `GameClientShell`, and secure real `RoomCoordinator` command forwarding. Added a socket-level integration fixture that verifies actual versioned room.create, correlation and source of authentication; negative test forbids connection on failed identity exchange.
- These are **integration foundations only**, not a working production identity endpoint, actual configured WeChat SDK login, visible room/game table, complete fairness workflow, or B-grade crash recovery. CI and independent Codex acceptance are not claimed at this checkpoint.
