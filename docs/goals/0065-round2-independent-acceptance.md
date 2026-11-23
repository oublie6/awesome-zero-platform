# Goal0065 Round2 independent acceptance — FAIL

2026-11-23 UTC. Fixed product **`c3232213ec6de5a111c3a31a791a88190afb56f0`**. Initial clean `main` was synchronized with `git pull --ff-only` and matched the requested SHA. Codex independently reviewed and executed the repaired client; **no product source changes**. Goal0065 remains **in_progress**. The original three exact reproducers pass, but three related P2 findings below prevent deployment and acceptance.

## Consolidated findings for ChatGPT repair

### G65-R2-N01 / P2 — successful engine download while hidden breaks initialization

Reproduce: real Vue login → enter Cocos while holding actual `cc.js` → 返回网站 → release that resource while the iframe remains `display:none` → reenter. The same iframe is retained, but it stays at document stage. Independently repeated twice, then confirmed with console/engine observations.

Actual console: `_updateAdaptResult Invalid size`, through the exported HTML `resize` callback and `screen.windowSize` setter. Canvas is **0×0**, `game._inited=false`, no scene; displaying the frame again does not recover initialization. Its watchdog eventually fails; explicit new-iframe retry succeeds. No server/private game state was altered by this startup fault.

Root: [template ENGINE_INITED callback](../../clients/fair-doudizhu-cocos/build-templates/web-desktop/index.ejs#L147) unconditionally writes physical screen size while the engine container has no rendered dimensions. The exception occurs before reporting engine progress/registering resize recovery, and rejects startup. `window.innerWidth` can remain nonzero while the containing iframe is hidden; checking that value alone is insufficient.

Repair: coordinate actual engine initialization/physical resizing with a visible, nonzero container and preserve safe resumability. Avoid throwing from this lifecycle callback; merely restarting the loader timer does not recover an already rejected initialization. Add actual hidden-download-success/visible-resume regression coverage. [Engine/console proof](evidence/0065-round2/hidden-success-confirm.json), [stuck loading screenshot](evidence/0065-round2/hidden-success-stuck.png).

### G65-R2-N02 / P2 — standalone late completion erases terminal failure

Open actual `/game/index.html`, hold `cc.js`, wait for the real60s failure and retry UI, then release the script without clicking retry. Scene initialization subsequently calls native `done()`, which removes the failed overlay. Recorded failure changes from `overlay=true,failed=true` to both false.

Root: [native done](../../clients/fair-doudizhu-cocos/build-templates/web-desktop/index.ejs#L108) checks only `complete`, while `fail()` never makes `complete` true. [Standalone Bootstrap completion](../../clients/fair-doudizhu-cocos/assets/scripts/bootstrap/GameBootstrap.ts#L126) can therefore complete an already failed attempt. Vue's new failed-phase guard protects the hosted parent but does not protect standalone native loading.

Repair: make terminal failure absorbing for that boot attempt, including native done and late initialization callbacks; only explicit retry/new generation should reset it. This is a failure-state UX defect, not evidence of unauthenticated server access. [Actual before/after](evidence/0065-round2/faults-r2.json), [late standalone scene](evidence/0065-round2/standalone-late-engine.png).

### G65-R2-N03 / P2 — stylesheet stall still blocks the native loader

Open actual standalone page and hold `/game/style.css`. At2s and **63,771ms**, body/overlay/boot handler/config promise are all absent and `readyState=loading`. Releasing CSS lets the actual scene load normally. Thus the asynchronous configuration repair does not eliminate all blocking dependencies ahead of the splash/watchdog.

Root: [external stylesheet in head](../../clients/fair-doudizhu-cocos/build-templates/web-desktop/index.ejs#L7) precedes parser-executed inline startup/configuration code. Pending CSS blocks that script's execution and parsing before the body loader exists. No native15/60s fallback can run in this state.

Repair: establish visible critical loading markup/styles and arm its watchdog without waiting on external CSS; load subsequent styling/resources in a deliberately ordered, failure-aware path. Preserve CSP and avoid adding unsafe inline event handlers. Add a real browser stylesheet-stall regression, not just a VM script test. [Delayed resource and live DOM proof](evidence/0065-round2/style-stall.json). No screenshot/first-paint measurement is claimed for the parser-blocked page.

## Original findings and extended runtime gates

| Round1 item | Round2 exact reproduction | Related limitations |
|---|---|---|
| N01 HTTP auth succeeds, WSS fails | PASS: authenticated true/transportReady false emits SESSION_FAILED; both covers retain failure, no false ready; retry works | Constructor failure and actual HTTP502 handshake tested; late current-frame SESSION_ACTIVE/GAME_READY after failure ignored |
| N02 hidden frame exceeds60s | PASS:62.5s CSS-hidden frame resumes the same iframe; rapid five Return/Enter cycles retain it | Real background tab also passes; hidden download success fails new R2-N01 |
| N03 game-config.js stalls | PASS: native page exists, no GAME_READY before config; delayed success,404,20s timeout, late result and retry tested | Other blocking head dependency fails new R2-N03 |

[Fault/event evidence](evidence/0065-round2/faults-r2.json), [actual handshake502→101 recovery](evidence/0065-round2/handshake.json). Normal stage order is document → engine → scene → session boundary → GAME_READY → SESSION_ACTIVE. Stage values reflect real events; no timer-generated download percentage was observed.

- Real background-tab test uses raw CDP without Playwright's focus override: verified `document.hidden=true` for62.5s, no native/parent failure, same iframe after foreground, actual “游戏就绪”. [Evidence](evidence/0065-round2/raw-background.json).
- Hidden real resource404 defers error until visible, retains the iframe on reentry, then exposes failure and explicit retry; a genuine error is not silently discarded.
- Session-boundary runtime instrumentation records cancellation of the scene watchdog and a new15/60s pair at the actual session-restore stage. A2.5s delayed real HTTP validation succeeds; after ready no tracked watchdog remains. [Evidence](evidence/0065-round2/session-boundary.json).
- JS404, HTML404 and offline→online explicit retry behave correctly. Old iframe, wrong-origin and wrong-source success/failure messages do not change the current attempt. [Transport observations](evidence/0065-round2/transport-faults.json).
- Rapid reload, forged lifecycle messages and closing during loading are recorded in [edge checks](evidence/0065-round2/edges.json). These are bounded scenarios; exact same-event-loop hide-versus-expiry ordering was not exhaustively enumerated.

## Fixed source, build and CI

Read AGENTS, canonical principles, current Goal, Goal0065, [Round1 report](0065-round1-independent-acceptance.md) and affected source/tests/diff. Backend/shared protocol have no diff from the accepted Goal0064 product; no unrelated Go audits or pressure tests were repeated.

Exact-SHA Actions independently fetched: [CI6/6 SUCCESS](https://github.com/oublie6/awesome-zero-platform/actions/runs/45127956722), [Fair Doudizhu4/4 SUCCESS](https://github.com/oublie6/awesome-zero-platform/actions/runs/45127956611), [Vue SUCCESS](https://github.com/oublie6/awesome-zero-platform/actions/runs/45127956667), [Cocos linux_source SUCCESS](https://github.com/oublie6/awesome-zero-platform/actions/runs/45127956643). The Cocos `linux_native_export` job was **SKIPPED**. [Job inventory](evidence/0065-round2/ci.json).

Independently exported the fixed archived source **once**, official Linux cocos-cli/Cocos4.0.0-beta.0, pinned CLI `f506fa3…`, engine `d0a696c…`, Node22.23.4. Real web-desktop build50s; generated Cocos4 declaration strict gate and BigInt AST gate passed, eight exported JS files. Vue `vue-tsc`/Vite build passed. Independently rerun: Vue `npm test` **12PASS**, `node --test scripts/cocos-linux-web/test.cjs scripts/game-web/stage.test.cjs` **20PASS**, Cocos `npm test` **163PASS**; no skips/failures. [Build identity](evidence/0065-round2/build-proof.json), [24 native file hashes](evidence/0065-round2/native-export-hashes.json), [staged manifest](evidence/0065-round2/artifact-manifest.json).

Chromium156, SwiftShader, headful Xvfb, scale1, isolated profile with trusted test CA; TLS errors were not bypassed. Real pinned Nginx fixture localhost30465 serves this export and proxies existing Go HTTP/WSS. Actual candidate cache map and four calculated inline CSP hashes are active; `nginx -t` passes. [Fixture identity](evidence/0065-round2/fixture-proof.json). This proves fixture integration, not Kubernetes rollout.

## Three-player real business regression — PASS

Three owned ordinary accounts enter independent real Cocos instances, create/join/ready/start one room and submit all three actual encrypted contributions. Lost ready ACK survives Return/Enter with its exact identity; lost contribution response is retried with the same requestId/clientSeq/expectedVersion/payload, and the accepted result is reconciled. Double Start sends one persistent command.

Waited for the real future public beacon. Mid-hand browser refresh closes the old connection and restores the same hand with the **unchanged authoritative deadline**. Manual legal bid and58 actual UI play/pass actions reach **COMPLETED/v60**. A two-distinct-rank illegal play is rejected with readable feedback and no version change. All three clients verify terminal archive, matching own phrase, rejection of wrong phrase, and room resynchronization; next-round ready views remain valid. The owned completed room is then left/closed through normal UI.

[Redacted game trace](evidence/0065-round2/game.json), [terminal screenshot](evidence/0065-round2/terminal-0.png). Pageerror0. One expected browser `net::ERR_FAILED` console entry comes from the intentionally aborted accepted contribution response; no unexpected console error in this game. No plaintext phrases, live private cards or contribution ciphertext are published.

Goal0064's actual300s DQ/full fault/cryptography proofs are reused for unchanged server code; this round does not rerun300s timeout or crash/fencing suites. The full game here is a fresh candidate run, not a relabeling of that prior evidence.

## Loading and cache observations

Actual loading screenshots: [390×844](evidence/0065-round2/loading-390.png), [760×390](evidence/0065-round2/loading-760.png), [1280×900](evidence/0065-round2/loading-1280.png). Each normal case completes authenticated transport and real Canvas menu input with no pageerror/console warnings/errors.

| Observation | Interactive ms | Cover ms | Requests / static | Wire bytes | HTTP304 |
|---|---:|---:|---:|---:|---:|
| Cold1280 | 7,987 | 33.2 | 26 /22 | 16,656,959 | 0 |
| Full refresh, same context | 7,958 | 19.4 | 26 /22 | 12,641,239 | 20 |
| New tab/login, same context | 9,757 | 74.6 | 26 /22 | 12,641,239 | 20 |
| Cold760 | 7,477 | 45.8 | 26 /22 | 16,656,959 | 0 |
| Cold390 | 6,846 | 40.9 | 26 /22 | 16,656,959 | 0 |

[Events/measurements](evidence/0065-round2/startup.json). Cold1280 actual document→engine3,336ms, engine→scene3,392ms, GAME_READY→SESSION_ACTIVE114ms. Interactive totals include real menu tests and observer work. One sample per scenario: no P50/P95 or statistical regression claim. This round's headful environment differs from Round1's headless measurements.

Same-session Return/Enter: **same iframe, same engine,1,381ms; zero new resource requests/WebSockets/persistent commands**, versus Round1's observed782ms. Persistent-profile standalone guest HTTP cache: cold **16,657,677B/5,202ms**, hot **18,902B/5,310ms**,21 actual304 including engine, versus Round1's16,650,780B/17,660B. These are distinct from iframe DOM reuse; guest timing does not prove authenticated interactivity. [Persistent HTTP records](evidence/0065-round2/persistent-cache.json). Large engine200 in isolated contexts reflects bounded memory caching; persistent-profile revalidation does work.

Actual response headers: hashed Vue assets immutable/max-age31536000; unversioned Cocos max-age0/must-revalidate; HTML no-cache/must-revalidate; config/manifest and authorized session/room reads no-store. ETag/Last-Modified/304 verified. Changed only owned fixture settings bytes, observed newETag/200, restored original bytes. No SW registrations/CacheAPI entries. Normal logout destroys old iframe and token returns401; switching account creates a fresh authorized frame with no previous room.

Five same-session navigation cycles: one iframe, GC heap24,468,956→25,658,080B, documents3→3, nodes293→299, listeners97→110. Bounded observation, not a long-term leak proof. [Privacy/cache/memory records](evidence/0065-round2/privacy-cache.json). Slow network120ms/524,288Bps download/262,144Bps upload showed the16s slow hint at stage1, then actual interaction at45,095ms with16,660,496B; expired/revoked session startup retained a failure cover and fresh login recovered. [Final browser observations](evidence/0065-round2/remaining.json).

## Operational result, limitations and observer corrections

**No candidate Kubernetes image was built/published/deployed.** Predeployment independent acceptance failed. Existing Goal0064 Web `sha256:44995360be6e91a0e7e1c355db4e8e9027633d1fc4d9eb59b15b110990af0a0f` and API `sha256:04614d0bc04c1fe37d2cb4467be23fec728cb0b1cfb63c26b1cd3746943ec82f` remain the rollback/current baseline. No production action, API/MySQL/Redis/DQ configuration, schema, Secret or PVC mutation. The HTTP502 injection changed only the owned local Nginx fixture and was restored.

Final operational audit and cleanup are recorded in [baseline comparison](evidence/0065-round2/final-audit.json). All six Pods are Ready; their UIDs/images/restart counts and Deployment/ConfigMap/Service/PVC/StatefulSet baselines are unchanged, both nodes Ready, no critical runtime log markers. Final hands9ABORTED/7COMPLETED, unfinished hands0 and deadline rows0. The new completed owned hand is retained; no historical hand was deleted. Owned browser19175 and fixture30465 are stopped; user29457 forward remains active. Normal QA identity/room/session history remains; no original user credentials were used. Candidate deployment preflight session inventory, immutable-image import, postdeployment game/cache/fault checks and actual cluster rollback were **not executed**, because the source gate failed. No successful rollout or deployed cache-policy claim is made.

Observer corrections retained privately: (1) Playwright focus emulation invalidated the first background-tab observation; raw CDP verifies real hidden state instead. (2) One missing geometry helper stopped the initial game harness before start; after adding the existing helper, clients restored the same server room and completed it without creating a replacement. (3) Failed502 handshakes expose socket-error events rather than the success-only handshake event; the corrected observer records actual502 failure and101 recovery. (4) A55s HTTP-validation hold correctly hit the existing5s request deadline, so it cannot test the native60s timer; replaced with real timer registration/cancellation and a2.5s validation delay. The aggregate postcheck session later returned143 after recording all final child exit0 results; saved JSON gates and staged file hashes were independently rechecked, without rerunning passed cases. None of these observer failures is claimed as a product defect or a passing runtime gate.

Private restricted evidence/harnesses: `/root/goal0065-round2-c323221`; public files are whitelisted/redacted and [checksummed](evidence/0065-round2/sha256.json). No physical mobile Chrome, WeChat SDK/device, production/release or soak qualification. The combined physical WSS loss during an existing UNKNOWN request was not separately rerun; this round proves exact UNKNOWN navigation/retry and separate WSS failure/recovery, with prior unchanged-shell evidence retained. Exact simultaneous visibility/expiry event order remains unexhausted; direct frame/tab >60s, hidden success/failure, delayed config/engine completion and rapid navigation/reload cases cover the stated observed paths.

**Final: FAIL.** ChatGPT owns R2-N01/N02/N03 repair and regression tests. Reverify at its next fixed SHA; only after all source/runtime gates pass may Web-only test deployment and its full postdeployment checks proceed. Goal0065 is not completed; historical Round1 FAIL and Goal0045/0046 states remain unchanged.
