# Goal0065 Round3 independent acceptance — FAIL

2026-11-23 UTC. Fixed candidate **`52fd6d76317d7023e0e3cc134dbe67a21afcf70f`**; last product commit `b4f34e07c25d9acb2e534ff61639301b338dc75f`. Verified their diff contains only Goal0065 documentation. Initial clean `main` was synchronized with `git pull --ff-only`. Codex changed no product source.

| Qualification | Result |
|---|---|
| Source build, automated checks and boundary review | PASS in executed scope |
| Integrated source / real browser acceptance | **FAIL — R2-N01 remains open** |
| Kubernetes candidate deployment/postdeployment acceptance | NOT EXECUTED: source gate failed |
| Production/mobile-device/release qualification | NOT ASSESSED |

Goal0065 remains **in_progress**. R2-N02 and R2-N03 pass their precise real-browser reproducers. R2-N01 avoids its old exception but still produces a blank, unusable scene after hidden startup. One smaller stylesheet cascade regression is also recorded.

## Blocking finding: G65-R3-N01 / P2 — R2-N01 is only partially repaired

Reproduce with this actual official export: login → hold real `cc.js` → enter game → 返回网站 → release engine request while the iframe is `display:none` → wait7s → reenter → allow another1s for rendering.

Observed together:

- Same iframe, engine initialized, scene exists; CSS/config/geometry promises all resolve true.
- Canvas framebuffer and CSS dimensions are **1207×701**, and the camera render window is also1207×701.
- Real Go account validation and WSS succeed. Parent displays **“游戏就绪”**, both loading covers disappear.
- Actual game area is blank. The menu node exists but its world position and computed input point are nonfinite; a real menu click cannot be targeted.
- Camera node position is nonfinite; all16 entries of its view and view-projection matrices are nonfinite. Projection matrix/aspect/orthographic height are valid, distinguishing this from a merely zero framebuffer.
- An actual portrait resize and another1s wait do not recover the menu. A same-artifact normal-visible-start control has finite input coordinates and a successful real menu click.

[Blank-frame screenshot](evidence/0065-round3/geometry-true.png), [normal-visible control](evidence/0065-round3/geometry-false.png), [plain runtime/control observations](evidence/0065-round3/geometry-confirm.json), [scene/camera and rotation details](evidence/0065-round3/geometry-details.json). Nonfinite numeric values serialize as `null`; matrix bad-field lists were produced with `Number.isFinite`. The plain confirmation does not wrap or modify the engine setter. Pageerror0, but the hidden path also produces `WebGL: INVALID_VALUE: texSubImage2D: no canvas`.

Root/ownership: client startup template and Cocos scene lifecycle. [resize](../../clients/fair-doudizhu-cocos/build-templates/web-desktop/index.ejs#L218) now safely avoids invalid size writes, but [geometry ready](../../clients/fair-doudizhu-cocos/build-templates/web-desktop/index.ejs#L156) proves only that a positive size setter returned. The generated application can initialize/activate its scene while hidden, before that deferred resize. [Bootstrap](../../clients/fair-doudizhu-cocos/assets/scripts/bootstrap/GameBootstrap.ts#L210) then treats the fulfilled geometry boolean as sufficient for GAME_READY. Later positive sizing does not repair the already invalid scene/camera state. This is rendering/input availability, not a server authorization bypass or a Go state defect.

Repair handoff: delay hidden initial scene activation until its viewport is usable, or reliably restore the affected scene/camera/layout state when it becomes visible. Establish actual render/input transform validity before announcing readiness, while preserving the retained iframe/engine contract. Regression must check the visible scene and real menu input, including finite camera/world transforms; buffer dimensions alone do not close R2-N01. Codex did not repair production code.

## Additional finding: G65-R3-N02 / P3 — late CSS restores a border and clips a canvas edge

Portrait slow-load observation: framebuffer343×526, CSS rect `x=5,y=0,width=343,height=526`, iframe viewport343×526. The canvas right edge is348, beyond the343px viewport. [Measured bounds](evidence/0065-round3/remaining.json).

The exported stylesheet contains `#GameDiv { border:5px solid black; border-radius:10px; box-shadow:... }`. Previously its link preceded the project's critical `border:0` override. The new [dynamic stylesheet insertion](../../clients/fair-doudizhu-cocos/build-templates/web-desktop/index.ejs#L168) appends it after that override, reversing cascade priority. This source/cascade inference matches the observed5px inset/edge clipping. [Stylesheet and bounds evidence](evidence/0065-round3/css-order.json). Normal menu tests still work; no gameplay command rejection is attributed to this smaller defect.

Repair: preserve the project's final container styling precedence when applying external CSS, then recheck content bounds at all three viewports. This is separate from the nonfinite-scene blocker.

## Exact previous findings and related races

| Previous item | Observed result |
|---|---|
| R2-N01 hidden engine completion | **FAIL / partial repair**: no `_updateAdaptResult Invalid size`, engine initialization succeeds, but positive canvas/geometry flags still expose a blank scene |
| R2-N02 standalone terminal failure | PASS: real failure at60,294ms, late engine/scene/done retain failed cover; explicit reload recovers |
| R2-N02 hosted terminal failure | PASS: real failure at62,598ms including website login; late engine/scene/done never emit successful SESSION_ACTIVE; explicit new-frame retry works |
| R2-N03 CSS blocks parsing | PASS: native body/loader paint before CSS;15s no-progress hint and60s no-progress failure operate |
| Engine completes before CSS, ResizeObserver absent | PASS: no premature GAME_READY/SESSION_ACTIVE; after CSS, positive geometry and real menu input succeed |
| CSS404/network failure/hidden404 | PASS: visible retry path, real hidden failure deferred until visible, original frame retained until explicit retry |

[Precise runtime records](evidence/0065-round3/r3-runtime.json). CSS stall slow hint is observed at21,616ms and terminal failure at66,661ms from navigation because actual engine/scene progress resets the idle watchdog. Late CSS onload cannot clear the terminal error; explicit retry succeeds. [Slow loading UI](evidence/0065-round3/CSS-slow.png), [timeout UI](evidence/0065-round3/CSS-timeout.png).

[Actual request waterfall](evidence/0065-round3/waterfall.svg), [request/DOM/paint timings](evidence/0065-round3/waterfall.json), [CSS-pending screenshot](evidence/0065-round3/CSS-waterfall-loading.png): CSS uses print media while pending; body/overlay exist and native first-contentful-paint is536ms, before stylesheet completion. Later CSS application leads to actual clickable game UI. No simulated engine or fabricated percentage is used.

Further real checks:

- [Config timeout while hidden](evidence/0065-round3/config-hidden.json): SESSION_FAILED is deferred by Vue, reentry keeps the same iframe and presents native failure; late config cannot activate it, old-window/wrong-origin success messages cannot authorize the replacement, explicit retry works.
- [WSS502 handshake](evidence/0065-round3/handshake.json): HTTP validation200/authenticated true, transport false retains failure; restored handshake101 and retry succeed.
- [Raw-CDP background tab](evidence/0065-round3/raw-background.json): actual `document.hidden=true`, engine request released while background,62.5s without false failure; foreground keeps the iframe and an actual menu click succeeds. This is distinct from `display:none` hidden initialization.
- [Session phase](evidence/0065-round3/session-boundary.json): real timer registration/cancellation resets native15/60s timers at session restore;2.5s real validation delay succeeds, tracked startup timers empty after ready.
- [Edges](evidence/0065-round3/edges.json): forged BOOT_PAUSE/RESUME with invalid source/origin leave timer identities unchanged; rapid reload ends with one connected iframe; logout destroys it; loading-page closure completes safely. [Revoked-session/slow-network observations](evidence/0065-round3/remaining.json) preserve failure cover and allow fresh-login recovery.

The first size-recorder probes encountered nonfinite click coordinates; the plain independent control/scene inspection above establishes the product failure rather than relying on a mouse protocol error. An early CSS-state reader used an absent bare global; it was corrected to a safe property lookup, and affected CSS cases were rerun. Those ReferenceErrors are harness defects, not product findings. Passed60s cases were not repeated.

## Source, CI and actual build identity

Read AGENTS, principles, current Goal, Goal0065 design, [Round1](0065-round1-independent-acceptance.md), [Round2](0065-round2-independent-acceptance.md), changed source/tests and diff. Server/shared protocol are unchanged; no unrelated Go fault/pressure suites were repeated.

Exact candidate runs: [CI6/6 SUCCESS](https://github.com/oublie6/awesome-zero-platform/actions/runs/45145174722), [Fair Doudizhu4/4 SUCCESS](https://github.com/oublie6/awesome-zero-platform/actions/runs/45145174650). [Candidate jobs](evidence/0065-round3/ci.json). Vue/Cocos workflows did not run at the documentation-only candidate commit; verified product-equivalent `b4f34e0` has [Vue SUCCESS](https://github.com/oublie6/awesome-zero-platform/actions/runs/45145038424) and [Cocos linux_source SUCCESS](https://github.com/oublie6/awesome-zero-platform/actions/runs/45145038641), with native export **SKIPPED**. [Equivalent-product jobs](evidence/0065-round3/ci-product-equivalent.json).

Independently built fixed archived candidate once using official pinned Linux cocos-cli/Cocos4.0.0-beta.0, CLI `f506fa3…`, engine `d0a696c…`, Node22.23.4. Real web-desktop export41s, generated actual Cocos4 strict declarations and BigInt AST gate pass, eight exported JS files. Vue `vue-tsc`/Vite build passes. Independent commands: Vue `npm test`13PASS; `node --test scripts/cocos-linux-web/test.cjs scripts/game-web/stage.test.cjs`24PASS; Cocos `npm test`163PASS: **200 tests, zero skips/failures**. [Build proof](evidence/0065-round3/build-proof.json), [native file hashes](evidence/0065-round3/native-export-hashes.json), [staged manifest](evidence/0065-round3/artifact-manifest.json).

Chromium156 with SwiftShader/headful Xvfb, device scale1, isolated profile/trusted test CA; no ignored TLS errors. Actual Nginx localhost30465 serves real export/Vue and proxies existing Go HTTP/WSS; exact inline CSP hashes and candidate cache map applied, `nginx -t` passes. [Fixture proof](evidence/0065-round3/fixture-proof.json). Fixture success is not deployed Kubernetes cache evidence.

## Fresh client business regression — PASS on normal visible startup

Three owned ordinary accounts log in, create/join/ready/start, submit actual fairness contributions and wait for the real future beacon. Lost ready ACK survives Return/Enter with identical request identity. Lost accepted contribution response uses exact requestId/clientSeq/expectedVersion/payload retry; double Start sends one persistent command. Actual browser refresh recovers the same hand and unchanged deadline. An illegal two-distinct-rank play is rejected without version advancement.58 actual UI play/pass actions complete a fresh hand at **v60**. All three clients verify terminal archive, correct own phrase, wrong-phrase rejection and room resync; the completed owned room is left/closed through UI.

[Redacted game trace](evidence/0065-round3/game.json), [terminal screenshot](evidence/0065-round3/terminal-0.png). Pageerror0; one expected `net::ERR_FAILED` from intentional accepted-response loss, no unexpected game console errors. No phrases, contribution ciphertext or live private cards published. Unchanged Go/protocol/cryptographic and real300s DQ evidence from Round2/Goal0064 is reused; crash/fencing and pressure suites are not rerun. This normal-path PASS does not close the hidden-start rendering defect.

## Cache and performance samples

Normal loading/input at [390×844](evidence/0065-round3/loading-390.png), [760×390](evidence/0065-round3/loading-760.png), [1280×900](evidence/0065-round3/loading-1280.png) completes without pageerror/console warnings/errors. Canvas-edge limitation is separately recorded above.

| Scenario | Interactive ms | First cover ms | Requests /static | Wire bytes | HTTP304 |
|---|---:|---:|---:|---:|---:|
| Cold1280 | 9,927 | 49.2 | 26 /22 | 16,663,922 | 0 |
| Same-context refresh | 7,827 | 23.6 | 26 /22 | 12,641,239 | 20 |
| Same-context new tab/login | 10,201 | 60.6 | 26 /22 | 12,641,239 | 20 |
| Cold760 | 6,970 | 60.9 | 26 /22 | 16,663,922 | 0 |
| Cold390 | 6,630 | 37.9 | 26 /22 | 16,663,922 | 0 |

[Raw stage/transfer samples](evidence/0065-round3/startup.json). Actual document/engine/scene/session/GAME_READY/SESSION_ACTIVE order retained. Interactive totals include real menu tests/observer work; one observation per case, no P50/P95 or statistical regression claim.

Same-session Return/Enter: same iframe/engine, **1,127ms, zero resource requests, extra WebSockets or persistent commands**, versus Round2's1,381ms. [Persistent-profile standalone HTTP cache](evidence/0065-round3/persistent-cache.json): cold16,664,640B/5,318ms, hot18,902B/4,711ms,21 actual304 including engine; Round2 cold/hot16,657,677B/18,902B. Separate this from DOM reuse and isolated-context bounded memory cache; large engine200 there does not disprove persistent revalidation.

[Actual headers/version/privacy checks](evidence/0065-round3/privacy-cache.json): hashed Vue assets immutable/max-age31536000; Cocos max-age0/must-revalidate; HTML revalidate; config/manifest and authorized session/room reads no-store. ETag/Last-Modified/304 verified; changed owned fixture settings bytes yield newETag/200, then original bytes restored and all staged manifest hashes rechecked. SW registrations0/CacheAPI empty. Logout detaches old iframe and old token returns401; account switch creates a different authorized frame without old room state.

Five navigation cycles retain one iframe. After forced GC heap24,282,176→25,617,712B, documents3→3, nodes294→300, listeners101→114; bounded observation, not a long-term leak claim. Slow network120ms/524,288Bps down/262,144Bps up shows16s hint at stage1 and actual interaction40,808ms,16,667,459B.

## Kubernetes, rollback, limitations and final handoff

**No candidate immutable Web image build/publish or Kubernetes rollout.** The required browser source gate failed. Existing Web `sha256:44995360be6e91a0e7e1c355db4e8e9027633d1fc4d9eb59b15b110990af0a0f` and API `sha256:04614d0bc04c1fe37d2cb4467be23fec728cb0b1cfb63c26b1cd3746943ec82f` remain current/rollback baseline. Local502 injection was restored; no API/dependency configuration, database schema, Secret, PVC, production or historical-user-data action.

[Final comparison](evidence/0065-round3/final-audit.json): six Ready Pods with unchanged UIDs/images/restarts; Deployment/ConfigMap/Service/PVC/StatefulSet baselines unchanged, two Ready nodes, no critical runtime log markers. Hands9ABORTED/8COMPLETED (one new completed owned hand), unfinished0/deadline rows0. Normal owned QA records retained. Browser19175/fixture30465 stopped; user29457 forward remains active. No cluster rollback was necessary because no rollout occurred.

Predeployment session inventory/backup/image import and postdeployment tests/rollback exercise were not executed after source failure; no PASS is claimed for them. No production, physical mobile Chrome/WeChat or release qualification. Exact visibility/expiry same-tick order and combined physical socket loss during an existing UNKNOWN were not exhaustively rerun; distinct failure/visibility/UNKNOWN cases and unchanged-shell evidence are identified above.

Private restricted harnesses/builds/logs: `/root/goal0065-round3-52fd6d7`; public evidence is whitelisted and [checksummed](evidence/0065-round3/sha256.json). Final evidence/diff review verified report links, artifact hashes, test scopes, sensitive-data exclusion and retained operational baseline. No product implementation/self-acceptance claim.

**Final: FAIL.** ChatGPT owns the remaining R2-N01/R3-N01 repair and P3 cascade correction. Repeat affected and integration gates at its next fixed SHA; only all-PASS source/browser qualification permits Web-only test deployment and its required full postdeployment game/fault/cache checks. Goal0065 and Goal0045/0046 remain in_progress; prior reports are preserved.
