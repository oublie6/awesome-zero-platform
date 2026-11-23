# Goal0065 Round1 independent acceptance — FAIL

Date: 2026-11-23 UTC. Fixed product source: `dac83978d4612e7cd50ed13b9b654aaa9719afcc` on clean synchronized `main`. This is independent Codex acceptance; no product source was repaired. Goal0065 remains **in_progress**. Three P2 findings prevent acceptance and candidate Kubernetes deployment.

## Findings and consolidated repair handoff

### G65-R1-N01 / P2: failed WebSocket startup is reported as ready

Reproduce with valid ordinary QA login and a controlled failing WebSocket constructor before entering the real exported Cocos page. HTTP account validation succeeds; socket connection fails. Actual messages are `document → engine → scene → GAME_READY → SESSION_ACTIVE`. Parent displays “游戏就绪”, removes its cover, and the native cover disappears, while `authenticated=true` and `transportReady=false`.

Root: [GameBootstrap.bindWebHost](../../clients/fair-doudizhu-cocos/assets/scripts/bootstrap/GameBootstrap.ts#L147) checks only authenticated after `login(true)`. [login](../../clients/fair-doudizhu-cocos/assets/scripts/bootstrap/GameBootstrap.ts#L556) catches connection failure and resolves; authentication can already have succeeded. [Vue SESSION_ACTIVE](../../clients/game-web/src/App.vue#L132) then completes startup. This violates the explicit failed-server-connection/no-false-ready gate. The scene itself rendered in this reproduction; this is not proof of the user's particular mobile black screen or an authentication bypass.

Repair: propagate an explicit startup success/failure result and require the actual connected/recovered usable scene before emitting SESSION_ACTIVE/native done. Preserve authorization checks and UNKNOWN identities. Add failure-after-successful-HTTP-auth regression coverage. [Observed state/events](evidence/0065-round1/wss-failure.json), [screenshot](evidence/0065-round1/wss-failure.png).

### G65-R1-N02 / P2: hidden loading iframe times out and is replaced

Hold actual `cc.js`, enter game, then click 返回网站 while document loading is visible. The same iframe remains hidden. After 64.37s, native BOOT_FAILED has marked parent startup failed. Reentering replaces the iframe instead of resuming it.

Root: [native arm](../../clients/fair-doudizhu-cocos/build-templates/web-desktop/index.ejs#L60) continues its 15/60s timers while hidden; [Vue page watcher](../../clients/game-web/src/App.vue#L212) clears only parent timers, and its message handler still accepts the hidden child's failure. This violates the explicit hidden-page watchdog requirement.

Repair: coordinate suspension/resumption of both watchdogs through an origin/source-validated lifecycle contract. Re-arm from real progress without inventing completion or allowing hidden stale errors to destroy retained startup. [Evidence](evidence/0065-round1/hidden-startup.json), [reentry screenshot](evidence/0065-round1/hidden-reenter.png).

### G65-R1-N03 / P2: blocking configuration prevents standalone loading UI and watchdog

Open real exported `/game/index.html` independently and hold `/game/game-config.js`. At **221,672ms** after navigation, `readyState=loading`, body absent, native overlay absent and `__AZP_BOOT__` absent. No 15s hint or 60s retry can run.

Root: [configure.cjs](../../scripts/cocos-linux-web/configure.cjs#L32) inserts a synchronous external script before `</head>`. The new native loader markup/script is in body. A stalled configuration request blocks parsing before either exists. This integration behavior predates part of this diff but directly fails this Goal's standalone/no-black-screen gate.

Repair: arrange visible native markup and watchdog before any potentially blocking external startup dependency; then explicitly order configuration before consumers. Preserve CSP hashes and fail-closed configuration. [Live DOM proof](evidence/0065-round1/config-stall-live-proof.json). Screenshot attempts blocked awaiting initial paint/font readiness; no black-screen screenshot is claimed.

## Source, builds and automatic gates

Read AGENTS, canonical principles, current Goal, Goal0065 and [Goal0064 Round2](0064-round2-independent-acceptance.md). Initial clean main was fast-forwarded to the fixed candidate. Backend and shared protocol have no diff from the accepted Goal0064 product; their unrelated audit/crypto/fault tests are reused rather than rerun.

Actual exact-SHA workflows: [CI, six jobs SUCCESS](https://github.com/oublie6/awesome-zero-platform/actions/runs/45046396295), [Fair Doudizhu, four jobs SUCCESS](https://github.com/oublie6/awesome-zero-platform/actions/runs/45046396606), [Vue build SUCCESS](https://github.com/oublie6/awesome-zero-platform/actions/runs/45046396312), [Cocos linux_source SUCCESS](https://github.com/oublie6/awesome-zero-platform/actions/runs/45046396235). That last workflow's **linux_native_export was SKIPPED**, not a CI export pass. [Job inventory](evidence/0065-round1/ci.json).

Independently built the fixed archived source once using pinned Linux official cocos-cli, **Cocos 4.0.0-beta.0**, Node22.23.4; CLI revision `f506fa…`, engine `d0a696…`. Official real web-desktop export completed in32s. Generated actual Cocos declarations strict type gate and BigInt AST gate passed, eight exported JS files. [Build identity and exported file hashes](evidence/0065-round1/build-proof.json), [staged artifact manifest](evidence/0065-round1/artifact-manifest.json). Vue `vue-tsc` + Vite build passed. Independent commands: `npm test` in clients/game-web: **10 pass**; `node --test scripts/cocos-linux-web/test.cjs scripts/game-web/stage.test.cjs`: **15 pass, one skip**. No repeated full Go build.

Private build/logs and controlled browser harnesses: `/root/goal0065-round1-dac8397` (restricted access; includes QA credentials/TLS material and is deliberately not committed). Export `native2/fair-doudizhu-web`; staged site `site`; logs `native-build.log`, `vue-build.log`, `targeted-tests.log`. Public evidence is redacted and [checksummed](evidence/0065-round1/sha256.json).

## Real browser loading, faults and security

Chromium156 headless with SwiftShader, device scale1, trusted isolated test CA; no ignored TLS errors. Real Nginx HTTPS/CSP fixture on localhost30465 served the actual export and proxied existing Go HTTPS/WSS application paths. Candidate cache map was installed in actual Nginx http context and `nginx -t` passed. This proves fixture integration, **not deployed Kubernetes cache headers**.

- Loading visible immediately at all requested viewports: [390×844](evidence/0065-round1/loading-390.png), [760×390](evidence/0065-round1/loading-760.png), [1280×900](evidence/0065-round1/loading-1280.png). All normal cases showed document/engine/scene/GAME_READY/SESSION_ACTIVE, authorized connected rooms and actual Canvas menu clicks; no console warnings/errors or pageerror. The narrow screenshot has outer page scroll; no real mobile-device qualification is inferred.
- Real engine settings I/O held during `game.init`: verified module loaded, engine not initialized, no scene. At16s hint visible, progress remained1; at60,193ms failure/retry appeared, still1. Explicit retry produced authenticated connected scene. [Evidence](evidence/0065-round1/engine-settings-stall.json), [slow](evidence/0065-round1/engine-stall-slow.png), [timeout](evidence/0065-round1/engine-stall-timeout.png).
- JS404, missing game HTML404, network offline/recovery and retry passed in the exercised paths. Offline error wording says same-origin requirement, which is imprecise but does provide failure/retry. Late messages from the old iframe and wrong-source/origin messages could not complete the new startup. Normal completion clears tracked startup watchdogs. [Fault observations](evidence/0065-round1/faults.json), [404 screenshot](evidence/0065-round1/script-404.png).
- Slow network120ms latency, 524,288B/s download, 262,144B/s upload:16s hint, progress1; actual final connection and Canvas input succeeded at39,946ms,16,653,491 measured wire bytes. Portrait canvas/frame dimensions both343×526, CSS matched physical bounds. [Evidence](evidence/0065-round1/remaining.json), [screenshot](evidence/0065-round1/slow-network.png).
- Actual current session revoked with HTTP logout before Cocos restore: failure cover remained; fresh login restored working game. Normal UI logout destroyed iframe, detached old frame, old token returned401; switching QA account created a different authorized iframe with no old room. SESSION_REVOKED handler/source isolation reviewed. No cross-account cards/state caching observed. [Privacy/session evidence](evidence/0065-round1/privacy-cache.json).
- Real lost room.ready ACK followed by Return/Enter retained same iframe and exact UNKNOWN request. Real lost room.leave ACK plus duplicate current-frame GAME_READY retained the same controller, requestId/clientSeq/expectedVersion/payload and RetryUnknownRoom. Original accepted leave was subsequently settled through its exact duplicate, not replaced. [Evidence](evidence/0065-round1/unknown-duplicate.json). No duplicate persistent command from ordinary navigation.

Observer corrections: two earlier System.import/init interception attempts did not intercept the real call and cannot prove engine stall. They were replaced by the verified real settings I/O stall above. Config-stall screenshot observers blocked; an independent live DOM probe supplied the >60s evidence, then closed only that fault page. These observer limitations are not additional product defects.

## Cache and observed performance

All timings below are single observations, not P50/P95. Interactive totals include real menu verification/observer work, so they are not pure CPU benchmarks. Stage deltas derive from actual events; no timer-generated download percentage was observed.

| Context | Interactive ms | Cover visible ms | Requests (22 static) | Wire bytes | HTTP304 |
|---|---:|---:|---:|---:|---:|
| Isolated cold1280 | 7,713 | 30.8 | 26 | 16,650,116 | 0 |
| Same-context refresh | 5,221 | 10.9 | 26 | 12,640,051 | 20 |
| Same-context new tab/login | 6,351 | 21.3 | 26 | 12,640,051 | 20 |
| Isolated cold760 | 5,078 | 22.4 | 26 | 16,650,116 | 0 |
| Isolated cold390 | 5,358 | 23.9 | 26 | 16,650,116 | 0 |

[Measurements/events](evidence/0065-round1/startup.json). Cold1280 document→engine3,429ms; engine→scene3,137ms; GAME_READY→SESSION_ACTIVE113ms. Isolated contexts re-downloaded large engine200 despite other resources304; do not conflate their bounded memory cache with persistent disk-cache behavior.

Independent persistent-profile standalone guest Cocos cold/hot (no request-routing cache disable): cold **16,650,780B /4,617ms**, hot **17,660B /4,448ms**,21 HTTP304 including engine. Guest runtime timings do not prove authenticated interactive startup. [Actual HTTP records](evidence/0065-round1/persistent-cache.json).

Same-session Return/Enter reused **identical iframe and engine**,782ms including input checks, **zero resource requests, zero extra WebSockets, zero persistent commands**. This is DOM reuse, distinct from HTTP cache. Full browser refresh recreated the engine as expected; new tab used explicit fresh login because sessionStorage is tab-scoped.

Actual fixture headers: hashed Vue JS public/max-age31536000/immutable; unversioned Cocos public/max-age0/must-revalidate; HTML no-cache/must-revalidate; configuration/manifest and authorized API reads no-store. ETag/Last-Modified and conditional304 observed. Changing only owned fixture settings bytes produced newETag/200, then original bytes were restored. ServiceWorker registrations0, CacheAPI keys empty. No business authorization/private payload included in static cache evidence.

Five actual same-session navigation cycles retained one iframe. After forced GC, heap24,678,860→26,027,116B; DOM documents3→3, nodes291→297, listeners94→107. This is bounded observation, not proof of zero long-term leakage. Cold JS heap approximately25–31MiB across measured contexts.

## Kubernetes, data and scope limits

**No candidate image was built/published/deployed to Kubernetes.** Findings failed the predeployment gate; existing accepted Goal0064 Web image remains `sha256:44995360be6e91a0e7e1c355db4e8e9027633d1fc4d9eb59b15b110990af0a0f`, API remains `sha256:04614d0bc04c1fe37d2cb4467be23fec728cb0b1cfb63c26b1cd3746943ec82f`. No API/MySQL/Redis/DQ/Secret/PVC mutation or production action. Six Pod identities/images/restart counts and deployment/configmap/service/PVC/statefulset specs match the captured baseline; all six ready. [Final comparison](evidence/0065-round1/final-audit.json).

Initial and final hands:9ABORTED/6COMPLETED, **zero unfinished hands**. Three new owned QA identities and ordinary QA room/session history are retained; no original user credentials were used. Exact pending owned leave was reconciled. No database schema changes. Owned Chromium19175 and temporary Nginx30465 were stopped after testing; both ports are closed. Existing user29457 forward remains active.

Rollback condition is any failed source/runtime/cache/security gate; here rollout never occurred, so no cluster rollback was necessary. After ChatGPT repair and reacceptance, preserve current immutable Web/config before any Web-only rollout, validate CSP/file hashes/cache responses, and roll back to the above accepted Web if rollout checks fail.

Goal0064 Round2 full three-player games, true300s DQ, room/hand recovery and fairness evidence are retained for unchanged business/backend paths. This round reran affected navigation/auth/UNKNOWN/room-create/ready/leave paths; it **did not rerun a full three-player game/start/recovery suite on the candidate**. That and postdeployment three-player/cold/hot/cache checks remain unverified and must be executed after repair before any PASS. No production, physical mobile Chrome, WeChat SDK or native-device qualification. The user's particular reported mobile black screen is not proven by these controlled candidate findings: the deployed environment still serves the prior image. Current public game config uses loopback29457; a different phone URL would need separate origin/endpoint investigation. Available recent server logs contain no identifying reproduction trace; absence of server errors does not establish browser success.

Final result: **FAIL**. ChatGPT owns N01/N02/N03 fixes and focused plus remaining regression; independent acceptance must be repeated at its new fixed SHA. No Goal completion is asserted.
