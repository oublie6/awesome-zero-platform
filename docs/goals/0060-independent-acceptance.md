# Goal 0060 — independent input and Kubernetes delivery acceptance

Date:2026-11-14 UTC. Fixed product **`6e7e63c9199b1b6bd1e5b60d88b77801dbf9f43e`**, `oublie6/awesome-zero-platform`, clean synchronized main before execution.

**Overall FAIL. G60-N01/P2: the actual Cocos password editor displays plaintext while editing in all3 required viewports.** Its blurred Cocos Label is masked, but the active native DOM editor is an unmasked textarea. No real credential was used to establish this defect. Mandatory UI acceptance is not green; candidate image construction and Kubernetes deployment were not attempted.

Original settled room/login duplicate-label and visible-hint hitbox reproductions pass, as do settled login button/password separation checks. Those results do not close the complete input acceptance, the real contribution input remains unqualified this round, and the maintained native smoke also fails in a resize sequence. Historical [Goal0058 PASS](0058-independent-acceptance.md) and [Goal0059 FAIL](0059-independent-acceptance.md) remain unchanged. Goals0045/0046 stay `in_progress`; paused Goal0056 Honor/ADB and C09/C10 DEFERRED are preserved.

## Authority, scope and plan

Codex acted as independent acceptance executor, with no subagent or product repair. Reviewed AGENTS, canonical principles, current goal constraints/state, Goal0058/0059 reports, current input/menu source, corresponding tests, native/staging workflow and existing deployment ownership. Verified clean main, then pulled fast-forward directly to the fixed candidate. The current goal file is unchanged from the preceding report checkpoint before this report's permitted status additions.

Owner: Cocos client input and menu presentation. Existing auth, room, fairness, hand and message contracts remain unchanged; Go owns decisions, MySQL owns durable state and Redis is auxiliary. No server/domain/platform dependency or data-ownership change was introduced. This is client-integration acceptance, not a server redesign or key migration.

The written private plan required source/native/types/AST/crypto and actual3-size input/overlap/lifecycle/rotation/zoom gates before any test rollout. Only after all mandatory UI gates and no P0/P1/P2 blocker would the operator restore the isolated4 services, verify immutable images/configuration/TLS, preserve data, review the manifest diff, deploy and run real Go/Vue/Cocos smoke. Successful services must remain running; the old unconditional scale-to-zero script must not run. On UI failure, preserve the original environment and hand defects back to ChatGPT. No new image, production action, schema change, database cleanup, TLS bypass or generated-game-JS patch was authorized as a workaround.

Private evidence root: `/root/.config/awesome-zero-platform/acceptance-0060-6e7e63c` (`E60`). Original private evidence from Goal0058/0059 is separate. Credentials, full raw runtime data and native artifacts are outside Git. Executed facts and unexecuted/future steps are distinguished below.

## Coverage of the10 delivery requirements

| Requirement | Evidence/result |
|---|---|
| 1. Fixed SHA and CI | Exact candidate; all4 exact-SHA workflows SUCCESS, native export skipped in CI |
| 2. Executed tests | Web11/11, client198/198 PASS; maintained native smoke FAIL twice; independent UI cases and new P2 below |
| 3. Real build/hashes | Pinned Linux web-desktop export, strict11-script compiler, final9-JS AST and full25-file inventory |
| 4. Three viewport UI |1280×720,390×844,844×390 actual native pages; screenshots, pointer/geometry, native editing properties |
| 5. Business/crypto | Current native BigInt/WebCrypto/HPKE PASS; fresh Go identity/room/3 contributions/game/proof/recovery gated, not claimed passed |
| 6. Kubernetes | Read-only preflight;0 candidate rollout; unchanged4 Deployments at0/0,0 test Pods |
| 7. Resources/cleanup | Sequential heavy work, explicit cgroup bounds, both nodes Ready, temporary processes stopped |
| 8. Remaining issues | G60-N01/P2, related single-line symptom, unresolved resize smoke failures, expired existing test TLS |
| 9. Historical reuse | Exact Goal0058 source and raw-evidence SHA256 table below; unchanged authority/protocol/controller paths only |
| 10. Verdict | **FAIL**, deployment/post-deployment verification BLOCKED behind mandatory UI failure |

## Source difference and CI

Compared with the Goal0059 product `37a21d059d911e5295a5b0abf40b2ce460cd63b7` / its report checkpoint `7780a0d15dfa78e27625e250a5b0f2be9696f07d`, the candidate changes6 files: GameBootstrap, MenuPagination, their2 tests, Web smoke and its test. Input Labels now use top-left anchors and SHRINK; input sizing/fonts are responsive; createLoginInput sets SINGLE_LINE after createInput returns; Web error wording is platform-specific; login buttons start lower. The tests and maintained smoke add hint geometry/pointer/font and password/menu separation checks.

The actual diff against Goal0058 confirms no changes under `server`, `clients/packages`, `clients/game-web`, or `clients/fair-doudizhu-cocos/assets/scripts/game`, and no API Dockerfile change. `source-reuse.json` records these verified boundaries. This permits historical evidence reuse for unchanged paths, not a fresh current UI or Go-integration PASS.

| Workflow | Exact-SHA run | Final jobs |
|---|---|---|
| CI | [42835870178](https://github.com/oublie6/awesome-zero-platform/actions/runs/42835870178) | SUCCESS,6/6 |
| Fair Doudizhu | [42835870424](https://github.com/oublie6/awesome-zero-platform/actions/runs/42835870424) | SUCCESS,4/4 |
| Fair Doudizhu Vue3 Web | [42835870209](https://github.com/oublie6/awesome-zero-platform/actions/runs/42835870209) | SUCCESS,vue_build |
| Cocos Linux Native Client | [42835870457](https://github.com/oublie6/awesome-zero-platform/actions/runs/42835870457) | linux_source SUCCESS; linux_native_export SKIPPED |

Run metadata/head SHA/conclusion were checked through GitHub's public API; jobs were independently read through the connected GitHub tool. `ci-runs.json`, `ci-selected.json`, `ci-jobs.json` retained. No candidate workflow was manually rerun and no unrelated full Go/race/performance matrix was run locally. Any later report-only commit CI is separate. Green source CI does not override the actual native failures below.

## Actual source/native checks

Executed repository commands with installed Node22.23.4 and actual pinned CLI/engine environment:

```sh
node --test scripts/cocos-linux-web/test.cjs
bash scripts/verify-client-foundation.sh
node scripts/cocos-linux-wechat/verify-toolchain.cjs
node scripts/cocos-linux-web/build.cjs "$E60/native"
node scripts/cocos-linux-web/smoke.cjs "$E60/native/fair-doudizhu-web" "$E60/smoke"
# A second unmodified smoke run preserved the original and wrote smoke-repeat/.
```

Web11/11 PASS,0 skipped, including the real installed Babel parser. Foundation33 core+49 protocol+116 shell/source = **198/198 PASS**,0 skipped, with typecheck/build gates. Portable historical Creator3 types are only a compatibility aid; the actual generated Cocos4 strict gate below is authoritative for this export.

Actual CLI `f506fa3de403b48b4833334fa48dddd27bb26364`; engine `d0a696cfe63fd8d19c70d7913ba11aa641c2e3c7`,4.0.0-beta.0. verify-toolchain checked installed revisions. The actual options in `/tmp/azp-native-cocos-web-ZT9tzh/web-build-options.json` select web-desktop and `buildScriptTargets:"chrome 79"`. All3 isolated shared-package dist directories were deleted and rebuilt before Cocos resolved them; entry hashes and input tree retained. No old dist, foreign Creator, engine replacement or JS repair was used.

Build unit exit0. The CLI reported its export phase completed in2m34s; this excludes package rebuild/startup and the wrapper's later gates. Upstream relative declaration TS2688 messages were handled by the existing explicit generated-path repair and strict post-build compiler, which exited0. Saved effective configuration includes all11 business scripts with strict settings; existing skipLibCheck is not portrayed as checking every engine declaration. Original unused3D importer/tool and Browserslist messages remain in the log.

Output `E60/native/fair-doudizhu-web`:25 files,16,452,311bytes. Inventory `native-artifacts.json`; actual options, generated config, effective config and isolated-package hashes retained. The formal build audited8 JS before public config injection; the final independent audit checked **9 JS,15,165,092bytes,1 native BigInt literal exponent,0 literal BigInt Math.pow calls**, and passed the candidate's broader alias/exponent checks.

| Native JS | Bytes | SHA256 |
|---|---:|---|
| `application.js` | 1269 | `60633140454acc1070f4eac53f89828e86f2c8471bb8f80e77e42fac68b4ff3a` |
| `assets/internal/index.js` | 259686 | `edd79bd3246940d6acd8adcaa6093c8f1c4e635af2c203f59684696b390fdf58` |
| `assets/main/index.js` | 392448 | `7d76f94ce84ac4d83b17fa752fcceb546bcae94ebbd8aae95a4d02b8ba7f2808` |
| `cocos-js/cc.js` | 12622964 | `d0b4d7872199c93f0b86eddf5d19d80152fae2cd9d84a793261d39d0c5d58b9f` |
| `game-config.js` | 44 | `953cc4b5ce1e74bd6352c2efa2a6320c5a57fda3301050bc56c59a5b88e8dc26` |
| `index.js` | 615 | `2e2d7f9d93e071156db972f33f7a7b03b3ab6c65d865c3b83009c261929a6042` |
| `src/chunks/bundle.js` | 1790581 | `cd7b109e9dd917a8a63dddacf4bca7297b3e54e9c1ba2354c7072c0a77a64354` |
| `src/polyfills.bundle.js` | 83 | `77d21eb18bb5296dd7ef0e66a678af45cc5f297bc64b0dbcae29f49577462ab7` |
| `src/system.bundle.js` | 97402 | `40d96cddea8facc054c39ab741ecd373f486ac0e97e1ba770ed7b618c924f311` |

## Current runtime and old reproductions

Actual Playwright1.64.1 / Chromium156.0.8078.4 ran the unmodified native output. Browser launch's root-only no-sandbox adapter was private; no TLS-error bypass or fake Go state was used.

At each required viewport, settled independent pointer/geometry checks passed for room and login hints: actual top-left Label anchors, unique TEXT_LABEL/PLACEHOLDER_LABEL, no RoomCodeText/default white label, readable rendered input fonts, visible glyph inside its own field and no field/button or field/field overlap. Real typing, Backspace and clearing were exercised. Settled password input and login action rectangles do not overlap; clicking Submit while the editor was active reached the real login handler and the expected missing-config failure in this standalone fixture. This is a real pointer-handler result, not successful authentication.

Blurred password Labels show masking, but active native editing **fails**; that distinction is the new finding. The completed diagnostic cases never relabel this failure PASS. All3 required sizes have screenshots `room-empty-*`, `room-typed-*`, `login-empty-*`, `login-typed-*` as produced, and focused password evidence `password-editing-{1280,390,844}.png`; detailed nodes/layout/editor properties in `extended-diagnostic.json`, `extended-remaining.json`, `privacy-summary.json`. Some diagnostic steps stop at a failed assertion, so missing screenshots/checks are not assumed to exist or pass.

Repeated login/room entry:8 cycles per viewport,24 cycles total; current native DOM editor counts stay within1 for room/2 for login; destroyed input nodes/children become invalid and their event processors disappear. This is bounded lifecycle evidence, not proof of zero heap growth under unlimited use. A separate actual detached-editor negative test preserved a reference to an old DOM editor, destroyed its scene inputs, then dispatched synthetic late input/change/focus/blur events. The old strings and new room input were unchanged;0 page errors (`late-dom.json` PASS). This covers current input-event cleanup, not all current cross-account network callbacks.

Actual browser zoom used the already installed **full Chromium executable**, a fresh private persistent profile and Chrome's own default zoom setting.125% changed inner viewport1280×720→1024×576/DPR1.25;80%→1600×900/DPR0.8;100% restored1280×720/DPR1. All3 preserved input value, focus and layout, with screenshots/`zoom.json` PASS. This is actual browser zoom, not a renamed CSS scale or device-only emulation.

Viewport/rotation checks inspected the actual browser SafeArea rectangle and geometry. Hardware notch/IME/device qualification is not claimed; Goal0056 remains paused. A timed independent rotation observer sampled50/180/350/800/1500ms after390→844→390→1280 changes, with settled focus checks; that run passed (`rotation.json`). However, it does not erase the failures described next.

## Unresolved resize verification failures

Both unmodified maintained smoke runs fail identically at `smoke.cjs:147`, after passing desktop and portrait hint checks and switching to844×390: the room hint geometry assertions pass, but waiting5s for an active INPUT/TEXTAREA after the pointer click times out. Original checks/logs retained in `smoke/`, `smoke-repeat/`. Neither run is reported PASS.

An early private300ms rotation check also raised `Field escapes viewport roomInput`; later settled/timed checks and the remaining viewport run with800ms settling passed. These results are reported together. Root cause of the discrepancy is **unresolved**; timing is a possible investigation direction, not an established explanation or a license to ignore the failure. No additional product P2 is assigned without stronger attribution. This remains a verification failure to investigate alongside the confirmed P2, and no all-rotation/native-smoke qualification is claimed.

## G60-N01/P2 — native password plaintext and wrong editor initialization

Owner: Cocos bootstrap input factory; scope: the standalone Cocos Web login form. This does not establish a disclosure in the unchanged Vue parent password field, WSS, server storage or real accounts. **Only dummy local strings were typed; no actual player password or token was exposed by this test.**

All3 sizes reproduce the following:

| Viewport | Cocos logical mode/flag | Active DOM element | CSS text security | Editing password masked | Blurred Label masked |
|---|---|---|---|---|---|
|1280×720|SINGLE_LINE/PASSWORD|TEXTAREA, type textarea|none|**NO**|YES|
|390×844|SINGLE_LINE/PASSWORD|TEXTAREA, type textarea|none|**NO**|YES|
|844×390|SINGLE_LINE/PASSWORD|TEXTAREA, type textarea|none|**NO**|YES|

The native element is visible with opacity1. Focused screenshots clearly show the dummy password characters, while the earlier blurred-state check sees asterisks. `privacy-summary.json` records native properties and geometry. The same native textarea also accepts an internal newline into the username despite the logical SINGLE_LINE setting (`G60\nUser`); this is a related effect of the same initialization problem, not a separate server-auth defect.

Cause verified against source and the pinned engine: GameBootstrap adds the node to active content and calls `node.addComponent(EditBox)` at lines337–338. EditBox.__preload initializes its native implementation immediately, while the mode is still the default ANY. The engine implementation chooses createTextArea at initialization. Only after createInput returns does createLoginInput set SINGLE_LINE at line361. That setter updates Labels, not the existing native element. The engine's `_updateInputType` explicitly returns early for a textarea and notes that it cannot support password type; PASSWORD therefore only masks the Cocos display Label when native editing is hidden.

Minimal reproduction: open this real export, RoomEntry→LoginEntry; click the visible password hint; type any dummy test string; observe the active editor before blur, then blur and compare. At all3 sizes the active textarea is plaintext and the blurred Label is masked. No backend or forged session is necessary.

ChatGPT repair checklist: establish the intended credential input mode before native EditBox initialization/activation, or correctly recreate the editor through a supported lifecycle; preserve engine-owned Label pairs and valid hit regions. Add a browser regression that verifies **active** native password masking, actual single-line behavior, then blur/re-entry/destruction and all3 viewports. Investigate maintained smoke resize readiness without deleting/weakening pointer, geometry or focus assertions. Codex did not patch source, the engine or emitted JS.

## Current crypto and historical business evidence

`native-crypto.json` **PASS** on this export: actual Entry/Canvas/home title, parent-received GAME_READY with only protocol/type keys, correct runtime `2n ** 256n`, available WebCrypto,2 distinct nonzero32-byte WebRandom reads, actual exported HPKE seal, independent Node open and wrong-AAD rejection.0 page errors and failed resources. This local iframe fixture is not Vue authentication, Go state, K8s HTTPS/WSS or a dealt hand.

Historical evidence source: **`c62d57200d69ec86d8012263afece1177e6359a9`**. Source diff checks establish unchanged server/schema/protocol, portable crypto, Vue and game-controller paths. `historical-evidence.json` preserves actual raw-file hashes:

| Private Goal0058 evidence | SHA256 |
|---|---|
| `full-game.json` | `c117ea0dc3423fa22910b00bcaea54f702614cd131b1d55aa8c49f5678e745c4` |
| `rooms-contributions.json` | `1312b706d93023bf9f89b0a47102c854e53a00bbfb75a5d92bf65705286f0089` |
| `bridge.json` | `17ae856113aae60a7769066c62b4f77512f6c000f29c6bcc47c9f097aea6e82c` |
| `ast-summary.json` | `4dedaf23e0dd3ff4b569f62154101d072c21c342a108da7e5349cf67e377c519` |

Those files support the earlier actual3-account full game/finalVersion60, own-phrase proof, genuine UNKNOWN exact retry, HTTP deadline, API Pod cold recovery at34/checkpoint33/epoch2 and account/bridge boundaries within their reported fixture. They are reused only for unaffected behavior. The current password editor and input/viewport paths were rerun rather than covered by that historical PASS.

Fresh Go identity, same-origin HTTPS/WSS, actual Vue/Cocos account switch/late network callbacks, create/join,3 new contributions, basic current game/proof/recovery and post-deployment logs were **not executed** after the blocking UI finding. The current contribution field was not faked by altering scene state. No current business/integration PASS or full G58/G59 repair closure is invented. Hardware/device and excluded C09/C10 work were not resumed.

## Kubernetes preflight, rollback boundary and final state

Actual context `kubernetes-admin@network-learning`, namespace `awesome-zero-game-test`, both nodes Ready. Inspected current resources, CPU/memory/disk and immutable image/configuration references; retained original MySQL data, credentials/protection roots and namespace. Test placement/configuration remain the previously verified topology. No production/shared-system mutation occurred.

| Component | Existing reference | Before→after |
|---|---|---|
| game-web | Goal0058 `sha256:d23435919ab571f552f349f689276fe8658976e095ddfcbf8e976834451ab069` |0/0→0/0 |
| app-api | unchanged Go `sha256:a00c238eb7463929289f16b4675b26d8c13228e2fd27aafcf6e5963c63fe9306` |0/0→0/0 |
| mysql | existing pinned image/goal0057 data |0/0→0/0; not started/reset |
| redis | existing pinned image/configuration |0/0→0/0; not started |

No Goal0060 Web image exists; no candidate assembly/image import, manifest apply, rollout, Kubernetes business smoke or rollback was reached. The planned rollback was only the Web image/configuration to the verified prior test reference, preserving all data; it was not needed. The old unconditional scale-to-zero script **was not executed**: the services started stopped and remained untouched, rather than being shut down after a successful deployment.

`resources-before.json`/`resources-after.json` show identical Deployment/Service/Ingress/configmap UIDs/spec/data. All23 cluster Pod UIDs/specs, including the shared platform Pods, are unchanged;0 test Pods (`unchanged-cluster.json`). Existing Services are ClusterIP and there is no test Ingress or usable new candidate URL. No temporary forward was presented as permanent access.

Additional environment precondition: inspected the **actual current** g57-tls leaf without printing its private key; it expired2026-11-10 12:43:04 UTC (CN127.0.0.1). This is not valid current HTTPS evidence. It was not renewed or ignored after the UI gate failed. A later passing candidate needs valid authorized test TLS before real HTTPS/WSS deployment verification; this is separate from and does not authorize database protection-root migration.

After a future accepted rollout, retain all necessary4 services as requested and stop only transient acceptance processes. That desired persistent-running end state was **not achieved** in this failed acceptance, and is not relabeled PASS merely because old Deployment objects exist.

## Resources, probe integrity and cleanup

All heavy work was sequential. Builder cgroup3GiB/CPU200%, browser trees1GiB/CPU150%, swap0. Minimum sampled node available memory: control-plane2.43GiB, HK4.00GiB. Both nodes remained Ready; no OOM-caused failure was observed. No service rebalance was needed and no dependency container was started. Disk preflight had about18GiB free on root and8.8GiB on data. Raw samples, unit statuses, inventories and original logs retained.

All temporary browser/static-server/build processes stopped; observer stop marker set. No CA trust, TLS secret, proxy or port-forward was installed/changed this round. Private input trees, profiles and evidence are owned by Goal0060 and retained seven days, through2026-11-21 UTC, for reproduction. No automatic shared-resource/data deletion is authorized.

Probe provenance: the first independent extension asserted an INPUT tag and stopped at desktop. The diagnostic then checked actual native masking through type **or** real CSS masking, retained failed results and continued unaffected cases; the actual textarea/none result still FAILS. It did not waive masking or alter source assertions. Initial Chrome zoom probes used an isolated context/headless-shell that cannot host Chrome settings; their original errors remain. The successful run used a fresh persistent profile with the already installed full Chromium and verified actual renderer viewport/DPR changes, without downloading tools or touching a user profile. Formal smoke code/assertions remained unmodified and both failures are retained. Later rotation settling observations do not erase original failures.

## Conclusion and handoff

| Acceptance area | Result |
|---|---|
| Exact CI/source tests/native export/strict types/final AST | PASS within stated scope |
| Current native BigInt/WebCrypto/HPKE | PASS |
| Settled original room/login Label/hint/button reproductions | PASS; contribution/full closure not claimed |
| Active native credential privacy | **FAIL, G60-N01/P2** |
| Maintained native smoke | **FAIL twice**; resize discrepancy unresolved |
| Current real-Go/Vue integration and K8s delivery | BLOCKED/NOT EXECUTED after mandatory UI failure |
| Overall Goal0060 independent acceptance | **FAIL** |

Return the P2 and preserved resize evidence to ChatGPT for a consolidated repair candidate. Only after all mandatory UI gates pass may image/deployment proceed with renewed valid test TLS, preserved database/roots, reviewed rollback and actual post-deployment business verification. No current product completion, release qualification, persistent access or mandatory-device completion is claimed.

Only this report and permitted Status/Working State/Completion Report additions in `current.md` are committed. Source, assertions, schema, deployment resources and historical reports are unchanged. The documentation commit is distinct from the verified product SHA; final diff/secret checks, commit/push and clean-main state are reported in the delivery handoff.
