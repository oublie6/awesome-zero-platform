# Goal0065 Round5 independent acceptance — FAIL

2026-11-24 UTC. Fixed and actually tested source: **`5849b7d062f24a3f8a8c8121282d9aea623e0797`**. Initial clean `main` was synchronized using `git pull --ff-only`. This is independent acceptance, with **no product source changes**. Goal0065 remains **in_progress**. The objective's stop-on-blocker rule applies: no candidate deployment and no unrelated regression sweep after the finding below.

## Result

- **Round4 G65-R4-N01 original discovery503 reproduction: PASS**, including explicit retry restoring the same existing room.
- **Round5 integrated startup/recovery: FAIL**, one residual P2: an active hand's private snapshot can be absent while startup announces SESSION_ACTIVE.
- Source/build checks pass in their stated scope; the remaining full acceptance and Kubernetes deployment/postdeployment gates are **NOT EXECUTED**, not PASS.

## G65-R5-N01 / P2 — sending hand.sync is mistaken for completing private-hand recovery

Use the actual Cocos4 Web export, real Go HTTP/WSS service and three owned ordinary QA accounts. The browser fault changes delivery only; it does not fabricate cards, server state, authentication or successful backend responses.

Reproduction:

1. Recover existing owned room `room-cmd-94b10a02027fdfa28b1fe8e570d8ca22`. Join/ready three different players, start a hand, submit real encrypted contributions and wait for the actual future public beacon/deal.
2. Positive control: all three real clients have BIDDING snapshots. Target hand is `hand-cmd-3dcc7738a100890a8f50d5ed14f8dd0d`, live version1.
3. Reload the owner's website. A browser WebSocket route forwards actual client requests and server traffic, but drops only server `doudizhu.hand.snapshot` messages. Keep authenticated WSS open and real room/fairness/member reads successful.
4. Enter the game and inspect startup events, both loading covers and the actual HandStore after initialization.
5. Remove the fault and explicitly click the website's reload button. Confirm the original room, HandID and version1 snapshot return.

Observed together:

| Observation | Actual value |
|---|---|
| Authentication / WSS transport | true / true |
| Authorized room and HandID | same original room / same original hand |
| Actual hand.sync requests / dropped snapshot responses | 2 / 2 |
| Private snapshot present | **false** |
| HandStore status | **synchronizing** |
| Host messages | document → engine → scene → session → GAME_READY → **SESSION_ACTIVE** |
| Vue status / loading cover | **游戏就绪 / absent** |
| Native loading overlay | absent |
| New WSS durable commands during the faulted restoration | 0 |
| After fault removal and explicit retry | same room, same hand, snapshot version1 |

The screenshot displays a generic waiting-for-contributions/beacon lobby, although the server has already dealt the hand. There is no restored private snapshot or continue-hand action. Pageerror count0. This is a false startup-success boundary, not proof of server data loss or authorization bypass; it does not establish the cause of the user's physical-phone black screen.

[Redacted runtime/control/retry evidence](evidence/0065-round5/private-sync.json), [actual failed-ready screen](evidence/0065-round5/missing-private-ready.png).

Expected: a live hand's required private recovery must be confirmed before SESSION_ACTIVE. A missing, rejected, malformed or timed-out required snapshot must retain a truthful pending/failure state and usable retry. Successful identity, socket, room metadata and renderable menu alone are insufficient. GAME_READY may describe the base scene; it cannot certify business recovery.

Root/ownership: this is game-client integration. [GameClientShell.syncHand](../../clients/fair-doudizhu-cocos/assets/scripts/game/GameClientShell.ts#L204) resolves after `connection.send`; it does not await receipt/validation/adoption of a snapshot. [Controller recovery](../../clients/fair-doudizhu-cocos/assets/scripts/game/GameOnlineController.ts#L167) and [strict startup refresh](../../clients/fair-doudizhu-cocos/assets/scripts/bootstrap/GameBootstrap.ts#L1273) both await that send-only Promise. Later room-version verification can succeed while HandStore remains empty. The [host startup gate](../../clients/fair-doudizhu-cocos/assets/scripts/bootstrap/GameBootstrap.ts#L205) then treats the resolved login plus authentication/WSS/render checks as complete recovery. The new strict HTTP error propagation does not establish this asynchronous realtime completion contract. This is a residual gap; no claim is made that the latest patch introduced syncHand's pre-existing send-only behavior.

Repair handoff to ChatGPT: require a bounded, current-account/controller/selection/HandID/version-fenced confirmation of the required authoritative private view, with cancellation, disconnect, invalid-response and timeout outcomes propagated to startup. Preserve exact UNKNOWN requests and server authority. Empty-room, multi-choice and legitimate pre-deal states must remain successful without inventing a private hand. Add behavioral tests with real delayed/dropped/error/malformed/stale sync responses; source-regex assertions and awaiting send are insufficient. Codex did not implement a repair.

## Round4 reacceptance and fault matrix

[Ten actual HTTP fault cells and recovery control](evidence/0065-round5/restore-matrix.json). Every injected cell was hit once and blocked SESSION_ACTIVE/游戏就绪 while retaining the authenticated identity and WSS transport. No empty-list success was inferred. The visible failure cover and explicit reload path remained usable. Removing the fault restored the exact original room. These are browser-controlled faults, not assertions that the deployed Go service spontaneously returned those errors.

| Recovery step | Actual Round5 fault coverage | Result / remaining scope |
|---|---|---|
| Account-scoped room discovery | HTTP503,401,403,404; malformed200 envelope; connection reset | **PASS** for six cells; timeout not executed |
| Initial room details | first target-room GET503 | PASS; other fault classes not executed |
| Fresh membership confirmation | second target-room GET503 | PASS; membership=false/changed authorization and other fault classes not executed |
| Required presentation-room read | third target-room GET503 | PASS; other fault classes not executed |
| Fairness stage read | successful real pre-deal/BIDDING reads during control | Fault matrix not executed |
| Private hand synchronization | actual WSS snapshot responses dropped while transport stays open | **FAIL / G65-R5-N01**; other sync fault classes not executed |
| Terminal evidence read | not fault-injected | Not executed |
| Final authority/version recheck | fourth target-room GET503 | PASS; other fault/version-change classes not executed |

The original Round4 P2 is closed **for its exact discovery503 reproduction**. This does not qualify all room/hand restoration paths. The two new controller tests cover HTTP throw/retry; the startup script tests primarily inspect source structure. Their success cannot prove that a required realtime snapshot was actually adopted.

## Source, CI, build and environment evidence

Read AGENTS, current Goal, canonical principles, Goal0065 design and Round3/Round4 reports; reviewed all product/test changes since Round4. No server, shared protocol, database schema or deployment product changes appear in that diff. No unrelated Go/SQL/Redis/cryptographic stress or crash suite was rerun. Historical reports remain unchanged.

Actual exact-SHA jobs: [CI6/6 SUCCESS](https://github.com/oublie6/awesome-zero-platform/actions/runs/45164565236), [Fair Doudizhu4/4 SUCCESS](https://github.com/oublie6/awesome-zero-platform/actions/runs/45164565270), [Cocos linux_source SUCCESS](https://github.com/oublie6/awesome-zero-platform/actions/runs/45164565403). Cocos linux_native_export is **SKIPPED**; no Vue workflow ran at this candidate. [Inspected job inventory](evidence/0065-round5/ci.json). These are CI evidence, separate from the independent local checks below.

| Independent command / check | Observed result |
|---|---|
| Official Linux cocos-cli Web export from fixed git archive, repository 2D engine profile | PASS, CLI reports33s, eight exported JS files |
| Actual generated Cocos4 strict declaration gate / BigInt AST audit | PASS / PASS |
| Vue `npm run build` (`vue-tsc` + Vite) | PASS |
| Vue `npm test` | 14PASS |
| `node --test scripts/cocos-linux-web/test.cjs scripts/game-web/stage.test.cjs`, explicit approved CLI environment | 28PASS, zero skips |
| Cocos `npm test` | 165PASS |
| Total final local source tests | **207PASS, zero failures/skips** |

[Build proof](evidence/0065-round5/build-proof.json), [native file hashes](evidence/0065-round5/native-export-hashes.json), [staged manifest](evidence/0065-round5/artifact-manifest.json), [fixture CSP/cache configuration proof](evidence/0065-round5/fixture-proof.json).

Pinned CLI `f506fa3de403b48b4833334fa48dddd27bb26364`, engine `d0a696cfe63fd8d19c70d7913ba11aa641c2e3c7`, Cocos4.0.0-beta.0, Node22.23.4; both actual tool Git revisions checked. Chromium156/headful Xvfb/SwiftShader, 1280×900, isolated trusted CA, no ignored TLS errors. Temporary pinned Nginx localhost30465 serves real Vue/export with exact inline CSP hashes and the candidate cache map, proxies real Go HTTPS/WSS; nginx configuration test passes. This fixture is not a Kubernetes candidate deployment.

Fixture corrections are retained privately, not misreported as product failures: initial copied input lacked the standard2D profile, so only that owned export was stopped and restarted with the repository profile; CLI's internal `tsc` was missing from PATH, but the separate mandatory generated-declaration strict gate actually passed. Initial script tests skipped the AST case without CLI environment; the explicit-CLI rerun passes all28. The first gameplay probe tried a room-refresh button absent from the contributing-stage menu; the resumed contribution then legitimately exceeded its submission window. A cleanup request with expectedVersion0 was correctly rejected before a corrected nonzero-version public cancellation succeeded. The final probe starts a fresh hand after all clients are ready and completes real contributions/deal before applying the snapshot fault. No product/assertion repair was made to conceal these fixture mistakes.

## Unexecuted acceptance after the blocker

No Round5 PASS is claimed for: the full seven-step × HTTP401/403/404/503/timeout/malformed/disconnect matrix; manual-refresh compatibility; complete no-room/single/multiple-room/terminal success matrix; account/logout/iframe-destruction/A→B→A and stale success/failure races; exact UNKNOWN retries; immediate WSS disconnection; all response order permutations.

**R3-N01 and R3-N02 have not received the required complete Round5 runtime reacceptance.** Hidden real-cc.js release, all Camera/UI world/view matrices, rotation/input isolation, three viewports, delayed CSS,15/60s watchdogs and late-event/retry sweep remain unexecuted. Normal1280 UI clicks and real contribution/deal are not substitutes for those gates.

Three real accounts joined/readied/contributed and received an actual deal, with same-hand refresh recovery after fault removal. There was **no fresh complete game to COMPLETED or personal final-proof verification**, illegal-action/UNKNOWN/account-isolation sweep, cache304/version/privacy qualification, cold/hot/request/transfer/interactive-time benchmark or sustained resource comparison. Earlier accepted backend/crypto reports remain historical evidence, not a Round5 end-to-end PASS. No postdeployment test was run.

## Kubernetes baseline, cleanup and rollback

No image build/publish/rollout or production operation. Existing test Web `sha256:44995360be6e91a0e7e1c355db4e8e9027633d1fc4d9eb59b15b110990af0a0f` and API `sha256:04614d0bc04c1fe37d2cb4467be23fec728cb0b1cfb63c26b1cd3746943ec82f` remain the current/rollback baseline. No Go API, dependency, Secret, PVC, schema or historical-user-data change was performed. No rollback was required because no deployment occurred.

[Before/after audit](evidence/0065-round5/final-audit.json): all six Pods Ready with unchanged UIDs/images/restarts, unchanged Deployment/ConfigMap/Service/PVC/StatefulSet baselines, both nodes Ready. Initial hands9ABORTED/8COMPLETED; after the two owned QA hands are business-cancelled,11ABORTED/8COMPLETED, **unfinished0**. One is the expired probe fixture; the other is the actual fault/control hand, cancelled only after same-hand/version restoration succeeded. QA room/member/ready/contribution/archive records are retained. The cancelled live hand initially retained one deadline cursor until its normal worker deadline; the final read confirms **deadline cursors0**. No database cleanup/delete was substituted for the normal business lifecycle.

Owned browser19175 and Nginx30465 stopped; user29457 forward remains active. Private evidence/builds/scripts and original failed fixture logs: `/root/goal0065-round5-5849b7d` (restricted). Public evidence excludes credentials, tokens, phrases, random contributions and private cards; screenshot and final diff are reviewed and [public files checksummed](evidence/0065-round5/sha256.json).

Second-pass **report self-review** checked the fixed-source/build identity, each HTTP fault hit, actual missing snapshot versus send completion, valid same-hand/version retry, preservation of UNKNOWN/server authority, fixture-error attribution, screenshot privacy, evidence hashes/links and permitted documentation-only diff. Unexecuted client/race/performance/deployment checks above remain explicit; this self-review is not a second independent product acceptance or a product implementation review.

**Final: FAIL.** Return G65-R5-N01 to ChatGPT for implementation and a new fixed-SHA acceptance. Report-only delivery does not complete Goal0065 or the interrupted Round5 checklist. Deployment remains prohibited until the complete required acceptance passes.
