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

## Development checkpoint — 2026-11-02 integrated implementation (not final acceptance)

The current implementation checkpoint includes:
- Authenticated, member-scoped durable room GET /v1/doudizhu/rooms/:roomId, caller-specific lastClientSequence and client recovery from trustworthy room reads.
- Optional encrypted persistence of server fairness seeds, wired to the production bootstrap; restart can load the same key-bound seed instead of silently choosing a new deck.
- MySQL live writer epoch and contiguous action journal, with genesis, writer takeover, conflict checks and action/result digests. The runtime uses clone → commit → publish and quarantines ambiguous commits. Startup recovery replays the journal and checks every result digest before admitting a hand. A preliminary real-MySQL journal/seed/fencing integration test has been added.
- WeChat wx.login code exchange over the official HTTPS endpoint with server-only app credentials, HMAC-hashed OpenID account binding and normal revocable platform sessions. The endpoint requires APP_WECHAT_APP_ID, APP_WECHAT_APP_SECRET and APP_WECHAT_SUBJECT_HMAC_KEY_HEX; absent credentials fail closed.
- Cocos entry now provides real login, room create/join/ready/start/refresh and a versioned private-hand action table; production HTTPS/WSS public configuration must be injected by the operator. No fake players/hand/cards are generated.

**Still not complete and not eligible for final Codex acceptance:**
- B-grade exact request-ID replay receipts after failover, a bounded protected checkpoint/snapshot, full crash/kill and wrong-epoch fault matrix, realistic latency/throughput measurements. Existing recovery tests do not establish all B-level invariants yet.
- Client fairness commitment/reveal HPKE path needs a server-issued signed AAD context. The current hand.reveal command correctly remains fail-closed; consequently a complete three-player fair game cannot yet be played from this UI.
- Complete scoring/evidence UI, verified three-person end-to-end, real WeChat mobile SDK/domain/TLS credentials/devices and Douyin runtime qualification.
- All checks must be rerun on one final source SHA; intermediate CI green/failure/cancellation is not a completed Goal. Codex has **not** been asked to perform final acceptance.


## Development checkpoint — 2026-11-02 bounded recovery and process-crash tests (CI pending)

- The live MySQL journal now rejects overlarge action payloads (16 KiB), more than 512 actions per hand, invalid/empty command or actor identities, invalid or backwards owner epochs, missing versions, invalid JSON and incorrect digest lengths before replay can build its in-memory receipt index. The 512-action cap deliberately exceeds a legal 54-card game's possible turns; if rules evolve, review this limit before release. This is a bounded **genesis-log replay**, **not** the encrypted checkpoint feature still required by this Goal.
- Added dedicated metadata/epoch validation unit tests and real-MySQL corruption checks. Tampered log rows must fail closed with `ErrLiveReplayIntegrity`.
- Added real child-process SIGKILL integration coverage around the journal boundary: kill before append, kill immediately after successful durable append, read committed action after process loss, acquire a higher owner epoch, reject the stale writer and admit the successor. This does **not** yet exercise the complete Cocos/Go three-player gameplay recovery or every COMMIT ambiguity.
- Source checkpoint `f6e9b65fd3cb664a69f9eba230ef1010e59c91aa`; automated CI is still to be confirmed on this exact revision. No Codex final product acceptance has been requested. Goal remains **in_progress**.
