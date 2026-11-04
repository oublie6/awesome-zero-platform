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


## Development checkpoint — 2026-11-02 full encrypted recovery prefix and reconnect integration (final CI pending)

This checkpoint supersedes the older statements above describing checkpoints, accepted request-ID receipts and server-issued reveal AAD as *unimplemented*. Those statements are preserved as historical snapshots, not today's source status.

**Implemented in source (not independently accepted):**
- `doudizhu_live_checkpoints` schema and independently derived AES-256-GCM key; hand/version-bound AAD, protected private views, ciphertext/plaintext size limits, SHA-256 integrity, epoch-fenced monotonic save against the authoritative MySQL journal watermark.
- Bounded asynchronous checkpoint writing (32-accepted-action interval, 8 concurrent checkpoint writes globally per process, 3-second bounded storage calls); a failed optional checkpoint cannot undo or falsely acknowledge a committed live action. Genesis+journal fallback remains available if no checkpoint exists; a corrupt checkpoint is rejected.
- Versioned encrypted replay-prefix checkpoint with exact accepted command identities/results plus public and each seat's private-view digests. Recovery validates checkpoint rows against *every* committed MySQL action, replays the sealed prefix and ordered suffix, verifies deterministic outputs, and publishes no receipt until the full hand validates. This is an event-sourced **replayable checkpoint**, not an O(1) materialized in-memory-state restoration.
- Real MySQL integration cases for encrypted payload non-disclosure, tampering, stale owner fences and future checkpoint watermark rejection. SIGKILL process tests cover pre-transaction, open uncommitted transaction, post-COMMIT survival, new owner acquisition and stale writer rejection. The open-transaction case demonstrates atomic rollback, not an arbitrary network-failure COMMIT outcome.
- Real deterministic 3-player **Go rules** fixture: bidding to terminal version 60, protecting the midpoint version 33 and replaying through final settlement with exactly matching final evidence and private views. These are not three independently authenticated production clients.
- Client authenticated reconnect and authoritative room-version/sequence and hand resync, with negative membership test. Cocos exposes explicit reconnect and exact pending fairness request retry. Room/table layout was adjusted for the expanded controls.

**Open acceptance / not proven:**
- End-to-end process-kill/restart of the entire app and actual three independently authenticated HTTPS/WSS clients, including lost reply immediately around COMMIT, stale/fenced owners, reconnect and complete fairness/settlement flows; existing process-kill proof is limited to the journal's MySQL transaction boundary.
- Measured 100/1,000 active hands, real MySQL commit and checkpoint/recovery percentiles under stated hardware and security/operational resource budgets; the original journal sample is not a production SLA.
- WeChat official domain/AppID credentials, TLS/WSS real-device interaction and release evidence; Douyin runtime/device qualification cannot be claimed from source or build checks.
- Normal client contribution recovery after the app itself is terminated mid-fairness remains fail closed; platform hardware-backed protected storage or an explicit authorized re-contribution protocol would need separate design and qualification.
- Checkpoint encryption root rotation/re-encryption and operational retention/compaction need deliberate deployment review. The current journal is the full durable history.
- Source commit on this checkpoint must pass general CI, Doudizhu CI, Creator smoke and final Codex independent whole-product acceptance before the Goal can be marked complete.

Do not archive Goal 0046, change the historic Goal 0045 verdicts or request final Codex acceptance until these release/verification gates are actually met.

## Development checkpoint — 2026-11-02 real-MySQL three-seat and scale delivery (final CI pending)

- Replaced quadratic speculative candidate replay with a deep clone of the already verified authoritative game, bidding and playing state, private card arrays, command history and material. A candidate cannot leak uncommitted bottom cards or a speculative turn to the live game. Durable commit-before-publish and full journal verification at startup are unchanged.
- Added a complete, deterministic **59-action / three distinct server-account** real-MySQL game test: bidding, legal play/pass, complete settlement, authenticated checkpoint at version 33, fenced owner takeover, stale-writer rejection, exact prior command replay, final restart, final record integrity and terminal lost-response duplicate replay. This does not substitute for three physically independent authenticated HTTPS/WSS devices.
- Added measured 100/1,000 hand real-MySQL scale profiling: protected seed storage, consistent startup reconstruction, 16 concurrent committers across distinct hands, P50/P95/P99 and post-GC memory delta. The CI now runs both sizes; these are runner baselines, not production SLA validation.
- Removed the misleading Cocos initialization placeholder message; login/room/table state still comes from authenticated remote server responses.
- Checkpoint `e9a744861757ad1c2a2bf06ecd7a4d6ad8d1a5e5` passed Doudizhu real-MySQL integration (including 100-hand test), while general CI identified and reported only a gofmt whitespace difference. This next commit fixes that and enables the 1,000-hand CI scenario. Full new-sha CI and independent acceptance must be confirmed before updating the Goal status.

## Implementation review — concurrent exact retries and recovery actor binding

- One bounded 256-stripe command lock is shared by Bid/Play/Pass per hand; the lock covers both receipt lookup and commit/publish, preventing accepted identical concurrent request IDs from receiving spurious version conflicts in the post-COMMIT pre-receipt window. No global lock and no unbounded per-attacker-ID mutex allocations.
- Protected checkpoint replay now verifies each claimed actor is the actual owner of the replayed seat, even when a forged checkpoint and log claim the same false identity. Added adversarial checkpoint test.
- Real MySQL integration launches 16 synchronized requests with one stable ID, verifies exactly one durable action and all identical authoritative outcomes, then performs a fenced coordinator takeover and repeats the same ID. This is a server-side concurrency proof, not three physically separate clients or a hardware-qualified WeChat session.

## Frontend unknown-outcome safety — explicit same-ID recovery

- GameHandActions now retains the exact requested hand mutation, including original command ID, client sequence, expected version and card payload, on socket timeout, disconnect or ambiguous server outcome. It forbids unrelated new moves until an exact retry or explicit after-authoritative-version reconciliation; a verified rejection can safely clear the pending request.
- GameClientShell accepts a narrowly scoped explicit exact-ID retry even when the local hand version has advanced, while checking the hand identity and leaving all authority to the existing authenticated server. INTERNAL errors cannot be treated as confirmed rejections.
- Cocos table adds explicit retry / confirmed-resync discard controls. No secret persistence or misleading auto-acceptance is introduced. These controls are in-memory; app-termination secret recovery and real device qualification remain separate gates.
- Frontend regression tests exercise response loss, changed local version, exact-ID replay, blocking divergent new commands and user-controlled abandonment after trusted resync.

## Consolidated development batch — 2026-11-02 (not yet CI verified / no Codex verdict)

**Code changes staged as one atomic release candidate; this paragraph is not a claim of CI success:**

1. Key rotation: `APP_DOUDIZHU_PREVIOUS_KEYS_JSON` accepts an explicit map of at most seven former 256-bit root key IDs to hex strings, alongside the active `ContributionKeyID/ContributionKeyHex`. New seeds, checkpoints and contribution records use the active root. Existing AES-GCM seed/checkpoint records can be read using retained roots while preserving identical hand/version AAD, authenticated plaintext and bounded ciphertext. `ReencryptSeed` authenticates and rewraps the exact original seed under a SQL row lock. `ReencryptCheckpoint` additionally requires the current owner epoch and verified watermark; it does not modify gameplay history. A real-MySQL integration test attempts both valid rotation and stale-epoch tampering. Contribution ciphertext retains its own key ID: **do not retire a previous root while any old contribution records or other protected data still require it**. Root configuration remains a development static keyring; managed KMS is a deployment security dependency.
2. Lifecycle and retention: `LiveJournal.PruneArchivedHand` requires an explicit selected hand, at least 30 days of completed archive retention, matching completed domain phase and independently reconstructed final record digest, exact final version/watermark and complete acknowledged action count. Only protected server seed, bounded checkpoint and live journal/writer rows are pruned; final archive, durable fairness/room command results and audit outbox remain. Rejected/dead/unverified hands cannot be pruned. No automatic destructive scheduler exists.
3. Authorized maintenance: `go run ./apps/doudizhu-maintenance --mode rotate|prune --hand HAND_ID` is dry-run by default; `--execute` is required to mutate a selected hand after authorized maintenance checks. Rotation additionally requires `--epoch` and the active plus retained keyring; pruning accepts `--retention-days` with a hard minimum of 30. The DB credential comes from `APP_MYSQL_DSN`, never from command-line flags. Each key rewrap has its own transaction; a partial result must be retried while retaining all prior roots.
4. Explicit client-controlled crash recovery: players generate a high-entropy, hand/seat/commitment-bound recovery kit **before** sending their public fairness commit. They must privately save the kit outside the application. A freshly launched client can import it only after querying the currently authenticated server's hand/seat/commitment; fraudulent/mismatched kits fail closed. The mini-game never stores raw random seeds or recovery kits automatically in its plaintext SDK storage. Loss of a kit cannot be magically repaired; the protocol must abort/restart or await an agreed future recovery flow.
5. End-to-end external qualification harness: `scripts/verify-doudizhu-three-client-e2e.mjs` requires a real prepared/dealt hand, three distinct already-authenticated JWT access tokens, HTTPS and WSS endpoints. It obtains private hands under each authenticated identity, sends legal bid/play/pass operations over real WSS, verifies server-owned version evolution, terminal settlement and authenticated archived evidence. It refuses missing tokens and fabricated accounts. CI can syntax-check the harness but **cannot execute an independently authenticated real deployment without supplied secrets**.
6. Automated regression tests: real MySQL key change / protected checkpoint rewrap and stale epoch; archive retention refusal for live/young/corrupt states and preservation of final evidence; client kit export/import, changed seat/account rejection and zeroization on disposal.

**Required operator and external acceptance still BLOCKED unless independently performed:**
- End-to-end WeChat `wx.login` with real AppID, approved HTTPS/WSS production domains, developer-tool interaction and 3 real/mobile device sessions. Do not equate a Creator build or Node harness syntax check with this evidence. Douyin device compatibility remains outside WeChat-first qualification.
- Fully credentialed 3-client fairness commit/reveal through terminal with actual key manifest, drand endpoint and production access tokens; the bounded Node harness starts from an already dealt hand and does **not** replace the full fairness run.
- True application-process SIGKILL during the three connected clients' session and physical rejoin is an operator-run fault gate. The existing process kill and real MySQL server coordinator takeover tests are partial but meaningful evidence.
- Independent Codex whole-product acceptance must still verify one exact post-CI commit SHA. Never mark Goal complete solely from static source and CI.

**No self-declared acceptance:** All these features remain pending automated validation and later external/device evidence until the exact release candidate CI is green.


## Consolidated F01–F11 repair candidate — awaiting new CI and independent Codex recheck

The 2026-11-02 independent acceptance at 0cc90a5 was **FAIL** (five P1,
six P2); its unmodified report is retained in docs/goals/current.md.

ChatGPT has staged a single repair candidate addressing:
- F01 raw signed manifest response vs wrapped business API
- F02 terminal-domain exact accepted receipt reconstruction and retry
- F03 immutable domain/genesis fairness + seat/account integrity comparison
- F04 domain-specific final evidence verification before destructive pruning
- F05 omitted empty-seat accountId in actual Go JSON
- F06 known rejected fairness response clears pending exact request, secret retained
- F07 disposal generation fence on asynchronous fairness continuations
- F08 Compose provisioned Quicknet, previous key roots and WeChat identity fields
- F09 Quicknet no longer depends on unused HMAC proof secret
- F10 quote the colon-ending development YAML scalar
- F11 lost room.create response retains command ID and generated room locator for
  authenticated read / exact Go dedup replay

Regression work includes client manifest/room/rejected-result/lifetime/retry
tests; actual-MySQL tampered-genesis test; completed-domain terminal receipt
test; rejection of incomplete archival material; provider config test.

**Verification not asserted here:** green CI, independent local process/network
reproducers, authenticated end-to-end web/UI and final Codex acceptance must
be rerun on the final commit. Never change Goal 0046 to completed while
critical findings or platform qualifications remain open.


## Round 2 N01/P1 historical bid replay fix candidate (awaiting CI/Codex Round 3)

The 2026-11-03 Codex Round 2 at `da0ed72fc9b49ffc0546589ac649ad83c9510001`
remains **FAIL**: original F01–F11 reproducers passed, but a bid acknowledged
under BIDDING could not be confirmed after the durable hand advanced to PLAYING
and the process restarted. The immutable original report is in
`docs/goals/current.md`. The original acceptance verdict is not changed here.

The single-owner repair uses a read-only `LiveHandReceiptReader` application
port implemented by the verified `LiveHandCoordinator` replay index. After
authorizing the current seated participant and validating command shape, the
application resolves exact previous Bid/Play/Pass receipts BEFORE rejecting a
request for an incompatible current lifecycle phase. The index checks original
actor, request ID, exact typed command payload and original expected live
version. An unknown ID is not executed in the wrong phase; an inconsistent ID
fails closed. In-phase commands still use the original owner-fenced durable
mutation path.

For out-of-phase accepted historical requests only, an internal
`LiveHandCommandResult.Historical` flag suppresses duplicate Dispatcher
lifecycle side effects. The public wire response format is unchanged. An
unrelated new bid after PLAYING must still fail with a conflict, not succeed or
generate an INTERNAL error. Normal phase advancement remains driven by fresh
accepted commands and in-phase reconciliation.

Regression scope: application-phase matrix and authorization/changed-request
negatives, Dispatcher lifecycle-once test, runtime archived Bid/Play/Pass
receipt inspection, and real MySQL end-to-end Service+Coordinator recovery from
BIDDING to PLAYING (with request replay and no duplicate durable action).
There are no schema, migration, client API or deployment changes. Extra
lookups are bounded in-memory receipt-map reads, synchronized with each hand's
fixed-stripe command mutex; no per-retry database scan is added. MySQL owns
durable evidence and fencing, which startup replay verifies before exposing
receipts. Rollback is the prior verified commit; no database migration exists.

**Not self-accepted:** new CI, exact-SHA process SIGKILL/three-WSS replication
of the original N01 scenario, adversarial payload/actor checks and Codex
Round 3 independent acceptance still gate closure. Until then Goal 0046
remains in_progress.

N01 implementation checkpoint: the new real-MySQL recovery regression and
Doudizhu client/manifest jobs passed on source `39c2c08f`; a one-shot
workflow formatted Go sources and removed itself in `cc90e1a6`.
The pre-format CI's gofmt gate failed as expected. The follow-on exact
formatted-source CI is the authoritative build gate; do not interpret
a successful pre-format integration job as complete acceptance.


## Round 3 R3-N02/P1 and R3-N03/P2 repair candidate — not independently accepted

The 2026-11-03 Round 3 independent acceptance at `04da095c269509a2e4061901ee3f45c92d885d22`
remains **FAIL** (read the unchanged report in `docs/goals/current.md`).
Its original N01/F01–F11 reproducers passed, but the following expanded
safety requirements remained open.

### R3-N02 — source-verified history and monotonic lifecycle reconciliation

**Owner:** Doudizhu application/runtime and the existing Dispatcher adapter,
not the platform or gamecore. Every exact receipt hit is now marked Historical
regardless of a concurrently read domain phase; the internal mutation path
also marks duplicates found while the operation was being serialized.
The normal AfterBid/AfterPlay side effects execute only on freshly committed
commands. Historical requests invoke a separate lifecycle reconciliation port.
That port does not reschedule normal bidding/playing deadlines or downgrade
a playing timer to bidding, but durably completes a previously omitted
BIDDING→PLAYING or PLAYING→SETTLING→COMPLETED transition if actual authority
is behind the already-committed result. Recovery still executes before serving
requests. No external response or client protocol version changes.

**Security/concurrency:** seat authorization and exact original actor/type/
payload/version binding are performed before any historical result is returned.
The read-only receipt index shares the per-hand fixed-stripe lock with receipt
publication; normal durable writes remain protected by MySQL epoch fencing.
A historical request does not create a new live operation or refresh an
existing timer. This does not claim that all fault-injected lifecycle races
have been independently requalified; Codex must rerun its original controlled
virtual-clock and real WSS tests.

### R3-N03 — persistent authenticated clientSeq identity

**Owner/data authority:** the Doudizhu-owned `doudizhu_live_actions` MySQL
journal, which now contains `client_sequence BIGINT UNSIGNED NOT NULL DEFAULT 0`.
The public Dispatcher passes the validated `clientSeq` through explicitly
sequenced application/runtime ports; the number is appended in the *same*
owner-fenced SQL transaction as the action and result digest. Exact duplicate
reads compare sequence along with the original actor, command ID/type,
expected version and typed payload. Cold-start replay restores sequence from
the verified journal for both checkpoint prefixes and suffix actions;
encrypted checkpoints record and cross-check it. A changed sequence cannot
become the successful cached identity or displace the original request.
Direct internal test-only/nontransport legacy paths retain sequence=0 and
cannot silently be used as a public transport substitute.

**Schema compatibility / operational rollout:** the repository is an early
complete-schema model and only `server/database/schema/current.sql` is
versioned. `CREATE TABLE IF NOT EXISTS` does *not* add the field to an
existing database. A controlled DB schema upgrade or a fresh disposable
development DB is **mandatory before rolling out this code**; no automatic
DDL, production migration, credentials, or destructive operation was
performed. For existing acknowledged live actions whose old journals lacked
a saved client sequence, the original value cannot be reconstructed safely:
the new reader treats a nonzero public retry against a legacy zero identity
as an **integrity mismatch (fail closed)**, never guesses or rebinds it.
Production upgrade must therefore first drain/finish or explicitly reconcile
these active legacy hands under a separate operator-approved migration plan.
Rollback to the previous app on a DB with the new additive column is
structurally possible; rollback of already-bound sequenced hands sacrifices
their new sequence guard and must **not** be done while such hands are active.
No release or full B-level history claim is made for unconverted legacy
hand identities.

**Performance and regression:** one extra unsigned integer per durable
action/receipt, no additional DB query per live command; existing 512-action
bound and fixed-stripe contention remain. Added real MySQL cold-restart
sequence 91→92 refusal→91 original acceptance, all-three-command complete
59-action takeover/checkpoint/terminal coverage, in-memory changed-sequence
checks, Dispatcher sequencing probe, virtual-clock historical timeout tests
and missing lifecycle stage repair controls. The exact post-format source
must pass CI and independent Codex Round 4 with the original R3 evidence
before Goal 0046 can be considered internally accepted. External WeChat
credentials, devices, domains and release qualifications remain BLOCKED.

Formatted implementation checkpoint: `f8abaac531a4af6eca9900dc9e12da21019434e9`
(one-shot gofmt succeeded and removed itself). A subsequent documentation-only
commit triggers the authoritative GitHub Actions on the formatted source,
since the GitHub Actions token's own formatting push does not start ordinary
push-triggered CI. No acceptance is asserted until those exact-source
workflows complete, and the independent Codex reproducers still gate closure.

## Round 4 FAIL remediation candidate — awaiting exact-SHA CI and Codex Round 5

ChatGPT repair candidate addresses R3-N02/P1 and R4-N04/P2 without changing
the public wire protocol. Verified receipts now have a stable historical marker,
including same-phase lookups and serialized duplicate calls. Dispatcher calls
AfterHistoricalBid/AfterHistoricalPlay rather than the ordinary timer-resetting
handlers. Historic retries can reconcile omitted BIDDING→PLAYING or terminal
lifecycle work without extending prior timeouts. Codex must independently
reproduce the original virtual-clock races and real SETTLING retry.

The live action journal adds integrity_version/integrity_tag. Version 2 is an
HMAC-SHA256 with a separate key derived from the existing checkpoint
protection root; it binds hand, version, epoch, command ID, account, seat,
original clientSeq, expected version, canonical typed JSON payload and
result digest. Retained checkpoint roots also validate retained journal tags.
Missing configured roots reject any nonzero public clientSeq. Version 1
(unkeyed digest) is reserved only for explicitly zero-sequence nontransport
legacy callers; public replay never accepts it. Unknown pre-upgrade version-0
records fail closed. The tag is inserted in the same fenced transaction as
the accepted live action. There is no extra database read per command.

Existing databases require operator-approved additive schema migration and
explicit drain/reconciliation of legacy active hands before rollout; the
current complete-schema SQL is not a migration. An old root cannot be retired
while journal tags signed under that root remain. No automatic live journal
retagging or production migration was performed. This is implementation only,
NOT a CI PASS or independent Codex acceptance.

Formatted Round 4 repair checkpoint: `f3f8e66b7bcaf45939dd49a310acc01fd1453f8d`. The one-shot formatter removed itself after applying gofmt. This documentation commit triggers ordinary Actions on that formatted source; successful CI or independent Round 5 acceptance must be established separately.

## Round 5 R5-N05/P2 remediation candidate — pending CI and independent Round 6

The Round 5 independent report confirms the original R3-N02 lifecycle races,
SETTLING in-process compensation, and R4-N04 client-sequence corruption
regressions now PASS. It identifies a separate authenticated-log recovery
policy gap: an untrusted journal row could choose the unkeyed legacy v1
verification path in a keyed runtime. The MySQL journal now rejects every
v1 row when an active or retained authentication key is configured, regardless
of the row's claimed sequence. Authenticated internal zero-sequence operations
continue to use and verify keyed v2 records; explicitly keyless legacy-only
internal environments still support v1. Pre-existing unsigned v1 rows are
NOT silently authenticated or migrated. A deployment enabling protection must
first drain or reconcile such active hands under an operator-approved plan.

This code-only recovery-policy change adds no SQL, RPC, schema, or gameplay
behavior changes to normal authenticated v2 operations, and adds unit plus
real-MySQL regression coverage including cold coordinator recovery. Product
implementation and tests are staged for CI; **no CI success, independent
acceptance, or Goal completion is claimed by this note**. Round 5 FAIL remains
the latest authoritative acceptance verdict until the next Codex review.

Formatted R5-N05 implementation checkpoint: `3f14b15cdfc8528cfd89b5290fdf4b486934bd06`. The one-shot gofmt job succeeded and removed itself. This documentation-only commit requests the ordinary exact-SHA CI/Fair Doudizhu workflows on the formatted source; do not claim their success until results are verified.

## WX-N01/P2 retry repair candidate — pending CI and independent replay verification

The 2026-11-04 real WeChat SDK + backend supplement established that identical
room.create WSS retransmission was rejected because the adapter regenerated
IssuedAt/ExpiresAt and the business-layer durable identity compared both.
Trusted API adapters now mark timestamps synthesized on receipt as non-client
identity metadata. The application still stores the first timestamps for
freshness/expiry enforcement on new commands and preserves strict comparisons
for every real caller-controlled field: actor (transaction scope), command ID,
type, aggregate, expected version, sequence and typed payload digest. Direct
application commands and signed hand.reveal context retain exact timestamp
binding. The temporary marker is never serialized into durable command JSON,
and new adapters do not change public request/response formats, SQL or schemas.

Unit and actual MySQL regression coverage checks replay across different
clock milliseconds and fresh service instances, original accepted result,
no extra room/outbox records and mismatched identity/payload rejection.
This is an implementation candidate, NOT independent acceptance. The
Round 6 supplemental WX-N01 FAIL stays authoritative until independently
reproduced against a fixed SHA. Goals 0045 and 0046 remain in_progress.

Formatted WX-N01 repair checkpoint: `15e28757f0c6e913edc60d2d137ae5a1dbe30a42`.
The temporary gofmt workflow formatted the updated Go files and removed
itself. This documentation-only commit triggers normal GitHub Actions against
the exact formatted code tree; final success and independent acceptance remain
separate gates.
