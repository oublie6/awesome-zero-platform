# Goal 0059 — input repair acceptance and persistent test-deployment gate

Date:2026-11-10 UTC. Fixed product **`37a21d059d911e5295a5b0abf40b2ce460cd63b7`**, repository `oublie6/awesome-zero-platform`, clean synchronized `main` before verification. **Overall FAIL: G59-N01/P2, visible EditBox text and actual input hit regions are misaligned.** The mandatory visual gate failed, so no candidate image was built/applied and no deployment success or persistent-running qualification is claimed.

This is independent Codex acceptance; no product code, test assertion, generated game JS, schema or deployment manifest was repaired. The [Goal0058 report](0058-independent-acceptance.md) and its fixed-source PASS remain unchanged. Goals0045/0046 remain `in_progress`; paused Goal0056 Honor/ADB diagnosis and C09/C10 DEFERRED are unaffected.

## Ownership and execution boundary

Owner: Cocos client input presentation in GameBootstrap. The small public room/hand/auth contracts, module dependencies and data authority are unchanged: Go owns game decisions, MySQL owns durable state, Redis is auxiliary. This task permits test deployment after acceptance; it does not permit product repair. The current change is client presentation, not foundation/platform infrastructure or a new business protocol.

Before execution, the operator read the objective, AGENTS, canonical principles, current goal/history, the previous acceptance report, client READMEs, actual changed source/tests/smoke, native/staging scripts and Kubernetes baseline/profile. The repository was clean on main and `git pull --ff-only` advanced directly to the exact candidate. The current goal and server/shared business sources were unchanged from the preceding checkpoint; no archived-goal investigation was added.

Plan: sequential source gates→real native build→actual UI/crypto→real-Go contribution regression→image→test rollout→post-deployment smoke→retain necessary services. UI correctness and the absence of a P0/P1/P2 blocker are mandatory gates. On visual FAIL, leave the existing Web image/data/configuration intact and return defects. Normal success would retain all4 services; the old scale-to-zero script was explicitly not authorized for a successful Goal0059 deployment and **was not executed**.

Security/resource constraints: private test credentials/evidence stay outside Git; no TLS bypass, fake identities, client-authoritative cards or fake contribution state. Builds and browser phases run sequentially; each browser capped1GiB/swap0/CPU150%, builder3GiB/swap0/CPU200%. Temporary processes stop afterward. Existing game-test namespace/data and shared services remain untouched if the gate fails. These are current executed facts, not broader release qualification.

Private evidence: `/root/.config/awesome-zero-platform/acceptance-0059-37a21d0` (`E59` below), mode0700. Original Goal0058 private evidence is retained separately. Raw screenshots, generated artifacts and detailed JSON are not committed.

## Required coverage index

| Item | Actual evidence and result |
|---|---|
| 1. Product SHA | Fixed37a21d059d911e5295a5b0abf40b2ce460cd63b7 |
| 2. CI final state | All4 exact-SHA runs SUCCESS; native export SKIPPED in CI |
| 3. Native build | Real pinned Cocos4 Linux web-desktop export PASS |
| 4. Strict types | Actual generated Cocos4 declarations/11 scripts, strict compiler PASS |
| 5. BigInt/HPKE | Final9-JS audit, real browser crypto/independent open/wrong-AAD PASS |
| 6. Label nodes | Room/login unique pairs, correct empty/typed/clear and destruction PASS; geometry FAIL |
| 7. Room screenshots | All3 sizes, empty/typed screenshots; actual pointer hit failure |
| 8. Contribution screenshots | NOT EXECUTED: real-Go phase follows the failed visual gate; no fake state used |
| 9. Login screenshots | All3 sizes, empty/masked typed screenshots; actual hint hit failure |
| 10. Viewports | 1280×720,390×844,844×390 all exercised; positioning/click-area requirement FAIL |
| 11. Basic business regression | Source196/196; fresh identity/room/contribution E2E gated, not claimed PASS |
| 12. Assets/SHA256 |25 native files/16,449,589bytes; per-file inventory and all9-JS hashes retained |
| 13. New image digest | NONE: goal0059-web was not built after the mandatory FAIL |
| 14. K8s rollout | NOT EXECUTED; no candidate manifest/diff/apply or rollback was needed |
| 15. Deployed HTTPS/WSS | NOT EXECUTED on new candidate; prior evidence is historical only |
| 16. Real login/rooms | NOT EXECUTED this round; existing data/evidence preserved |
| 17. Contributions | NOT EXECUTED this round; no claimed closure for the real contribution field |
| 18. Final Pods | Test Deployments remain0/0,0 test Pods, both nodes Ready; unchanged initial state |
| 19. Access | Existing ClusterIP services only; no test Ingress or continuous external HTTPS entry verified |
| 20. Retained running services | No successful deployment to retain; no services were started or scaled down |
| 21. Remaining issues | G59-N01/P2; small portrait text observation; real contribution/deployment gates unexecuted |
| 22. Verdict | FAIL; deployment/post-smoke/persistent access blocked behind mandatory visual FAIL |

## Diff and CI

Compared with the Goal0058 product `c62d57200d69ec86d8012263afece1177e6359a9`, and the report-only commit `acf0a836e587763ef3a75473aed65b08648e93fb`: the product candidate changes only GameBootstrap input creation, its source contract test, Web smoke and its test. The new createInput factory creates TEXT_LABEL and PLACEHOLDER_LABEL before adding EditBox, assigns both references/hint, then applies limits, password mode and initial string. This addresses duplicate-node ownership but does not establish correct geometry.

`git diff --quiet c62d57200d69ec86d8012263afece1177e6359a9 HEAD -- server deploy/docker/app-api.Dockerfile clients/packages clients/game-web` passed. Server protocols, HPKE, durable logs/recovery, identity implementation, Vue shell and shared controllers are unchanged. Goal0058 full-game/UNKNOWN/cold-recovery evidence can therefore be cited for those unchanged paths; `reused-evidence.json` saves original evidence hashes. It is not a substitute for current UI/contribution/deployment checks.

| Workflow | Exact candidate run | Final result |
|---|---|---|
| CI | [42745839801](https://github.com/oublie6/awesome-zero-platform/actions/runs/42745839801) | SUCCESS,6/6 |
| Fair Doudizhu | [42745839765](https://github.com/oublie6/awesome-zero-platform/actions/runs/42745839765) | SUCCESS,4/4 |
| Fair Doudizhu Vue3 Web | [42745839601](https://github.com/oublie6/awesome-zero-platform/actions/runs/42745839601) | SUCCESS,vue_build |
| Cocos Linux Native Client | [42745839604](https://github.com/oublie6/awesome-zero-platform/actions/runs/42745839604) | SUCCESS; linux_source success/native export skipped |

Read jobs through the connected GitHub tool and independently checked run metadata/head_sha/conclusion through GitHub's public API. Saved `ci-jobs.json` and4 `ci-run-*.json`. No candidate CI rerun or unrelated full Go matrix was initiated.

## Source gates and genuine native export

Commands executed from the repository with installed Node22.23.4 and the pinned CLI/engine environment:

```sh
node --test scripts/cocos-linux-web/test.cjs
bash scripts/verify-client-foundation.sh
node scripts/cocos-linux-wechat/verify-toolchain.cjs
node scripts/cocos-linux-web/build.cjs "$E59/native"
node scripts/cocos-linux-web/smoke.cjs "$E59/native/fair-doudizhu-web" "$E59/smoke"
```

Web tests10/10 PASS,0 skipped with the real CLI Babel parser. Foundation checks:33 core,49 protocol,114 shell/source tests, **196/196 PASS**,0 skipped, and required typechecks/builds PASS. An initial Web test run omitted COCOS_CLI_ROOT and skipped the parser case; that log was retained and the full10 were rerun with the actual toolchain. It is not reported as an initial10/10 pass.

Actual CLI revision `f506fa3de403b48b4833334fa48dddd27bb26364`; engine `d0a696cfe63fd8d19c70d7913ba11aa641c2e3c7`,4.0.0-beta.0. verify-toolchain checked installed revisions. Actual options `/tmp/azp-native-cocos-web-gTvLYL/web-build-options.json`: web-desktop, Chrome79. All3 isolated shared-package dist directories were removed/rebuilt from the fixed source before Cocos resolution. Entry SHA256s are retained in `isolated-package-rebuilds.json` with the build log/input tree. No stale dist, engine replacement, post-export JS patch or foreign Creator was used.

Build unit exited0. Initial upstream relative generated-type errors were handled by the existing explicit declaration-path repair and strict post-build gate; the final actual engine declaration compiler passed. `effective-tsconfig.json` includes all11 business scripts with strict settings. Existing skipLibCheck remains; this is not strict checking of every engine library. Unused built-in 3D import/tool and Browserslist messages remain in the original log rather than being disguised as product defects or erased.

Output: `E59/native/fair-doudizhu-web`,25 files,16,449,589bytes. Full `native-artifacts.json` plus build log are retained. The build audited8 generated JS before public configuration; the final independent audit covered all9 JS, preserving native BigInt exponent and rejecting lowered BigInt pow. Per-JS bytes/SHA256 are recorded below.

| Native JS | Bytes | SHA256 |
|---|---:|---|
| `application.js` | 1269 | `60633140454acc1070f4eac53f89828e86f2c8471bb8f80e77e42fac68b4ff3a` |
| `assets/internal/index.js` | 259686 | `edd79bd3246940d6acd8adcaa6093c8f1c4e635af2c203f59684696b390fdf58` |
| `assets/main/index.js` | 389726 | `bbbe108cff5c1bec3ae623c1281176aa98efff58fc9e3efb42d10c9f21c71123` |
| `cocos-js/cc.js` | 12622964 | `d0b4d7872199c93f0b86eddf5d19d80152fae2cd9d84a793261d39d0c5d58b9f` |
| `game-config.js` | 44 | `953cc4b5ce1e74bd6352c2efa2a6320c5a57fda3301050bc56c59a5b88e8dc26` |
| `index.js` | 615 | `2e2d7f9d93e071156db972f33f7a7b03b3ab6c65d865c3b83009c261929a6042` |
| `src/chunks/bundle.js` | 1790581 | `f0a0d061238700605e2088f62ceadbddef826c39ed852892ced73fc144bebb6c` |
| `src/polyfills.bundle.js` | 83 | `77d21eb18bb5296dd7ef0e66a678af45cc5f297bc64b0dbcae29f49577462ab7` |
| `src/system.bundle.js` | 97402 | `40d96cddea8facc054c39ab741ecd373f486ac0e97e1ba770ed7b618c924f311` |

## Actual Chromium, nodes, input and lifecycle evidence

Playwright1.64.1 / Chromium156.0.8078.4 loaded actual generated files. Official smoke passed RoomEntry, unique-input-labels for roomInput/loginIdentifier/loginPassword, LoginEntry, CancelWebLogin; no page errors or failed resources. Root's launch-only --no-sandbox adapter is private; no TLS bypass or game-bundle replacement was used.

Independent UI typing tests ran one context at a time at1280×720,390×844,844×390. Each room/login field has exactly one TEXT_LABEL and one PLACEHOLDER_LABEL; no RoomCodeText or default white `label`. Initial placeholder strings match the intended Chinese hints. Real pointer focus plus actual HTML input events changed the Cocos string; after blur the real text was visible and placeholder hidden. Clearing restored the correct placeholder. Password mode and displayed mask passed. Room→login→room destroys the old nodes/children and their node event processors; no accumulated input nodes were found. `visual.json` records these **label/lifecycle** passes, not overall visual acceptance.

Screenshots for each width: `room-empty-{width}.png`, `room-typed-{width}.png`, `login-empty-{width}.png`, `login-typed-{width}.png`; extra geometry/login screenshots and JSON retained. The independent reviewer inspected actual screenshots and then checked projected node geometry and real hint/center clicks. That second step found the blocker missed by name/visibility assertions.

Actual native crypto fixture observed real Entry/home title, Canvas and parent-received GAME_READY. Native `2n ** 256n` was correct; exported WebRandom returned distinct nonzero32-byte draws; the actual exported HPKE sealer encrypted, an independent Node receiver decrypted, and wrong AAD was rejected. `native-crypto.json` PASS,0 errors/failed resources. This local fixture is not Go identity, game state or a deployed HTTPS/WSS test. Current real-Go contribution/SESSION_ACTIVE checks were not reached after the visual gate failed.

## G59-N01/P2 — visible text lies outside actual input hit regions

**Confirmed in all3 viewports for room code, username and password.** Clicking a point on the visible placeholder produces0 visible native input elements; clicking the actual field center immediately produces1. This is functional pointer evidence, not a source-only judgment or a screenshot guessed from layout. It affects the normal first action needed to type a room code or login credential.

Representative real CSS geometry (read without changing business state):

| Viewport | Room input rectangle x/y/w/h | Visible hint click x/y | Hint click inputs | Center click inputs | Hint font CSS px |
|---|---|---|---:|---:|---:|
| 1280×720 | 410.00,322.20,460.00,54.00 | 352.00,322.20 | 0 | 1 | 24.00 |
| 390×844 | 124.92,377.69,140.16,63.30 | 107.25,377.69 | 0 | 1 | 7.31 |
| 844×390 | 270.34,173.32,303.31,31.65 | 232.10,173.32 | 0 | 1 | 15.82 |

At1280×720, the room hit rectangle begins at x410 while its visible placeholder starts around x328; the tested text point x352 is outside the input. Username hint point(316,293.4) and password hint point(412,358.2) likewise do not activate their intended field; center clicks do. All three widths show the same hint0/center1 result for all3 fields in `geometry.json`. At390×844 the hint is only about7.31 CSS px; the screenshot shows tiny text. This sizing observation also needs attention under the explicitly required viewport check, but the P2 decision already rests on repeatable failed pointer focus.

Root-cause evidence: the new createInput factory creates label UITransforms with the default anchors(0.5,0.5). The pinned EditBox `_updateLabelPosition` positions labels at the input's left/top offsets; it does not correct those anchors. Actual nodes retain(0.5,0.5), and their projected text/placeholder rectangles extend outside the field. Placeholder Label auto-sizing makes the discrepancy depend on its text. Source: `clients/fair-doudizhu-cocos/assets/scripts/bootstrap/GameBootstrap.ts`, createInput around315–340; corroborated by the actual current engine `cocos/ui/editbox/edit-box.ts` positioning code and runtime geometry.

Minimal reproduction: export this candidate, open native Web at any required viewport, enter room lobby, click the visible “输入好友房间号” text. No text-entry element appears. Click the actual field center and it appears. Open the real Web login screen and repeat for its visible username/password hints. No account or backend is necessary to reproduce the defect.

ChatGPT repair checklist: make text/placeholder anchors, sizing and positioning agree with EditBox's real geometry; keep the visible input/hint inside the pointer region; verify readable actual viewport font sizes; add actual pointer-on-visible-hint tests and geometry assertions for room/login/contribution fields. Preserve unique node ownership, masking, clearing and disposal. Codex did not implement a fix. Existing source/smoke tests check names/visibility and center-click success, so their green result does not cover this failure.

## G58-N01 status and gated work

The original duplicate-label/default-white-label subcases PASS for actual room and login fields. The **complete G58-N01 visual repair acceptance remains FAIL/not closed in this round**: normal pointer areas fail, and the contribution input has not been qualified through a real Go hand. No fake phase was injected to obtain that evidence. The source factory covers it, but source resemblance is not an executed contribution-field pass.

Following the objective's gate, real-Go startup/contribution fixture, Vue final assembly, new goal0059-web image, candidate manifest diff/apply, rollout and post-deployment smoke were **not executed**. This is an intentional stop after a confirmed P2, not an attempt to count missing gates as success. No full-game, UNKNOWN or recovery stress rerun was needed; unchanged Goal0058 provenance is separately retained. No historical room/hand/archive was modified.

## Kubernetes, access and final retained state

Verified context `kubernetes-admin@network-learning`; cluster-info and both Ready nodes; owned namespace `awesome-zero-game-test` remains goal0057-labelled. Inspected existing Deployments/Services/configmap/secret names and ingress state without printing secret values. Existing4 Deployments were0/0 before acceptance and remain0/0;0 test Pods. All23 cluster Pod UIDs/specs match the before snapshot, including the9 shared platform Pods. Test Deployment/Service specs are unchanged. `resources-before.json`, `resources-after.json`, `unchanged-cluster.json` prove the boundary.

| Component | Existing image/configuration | Final state |
|---|---|---|
| game-web | Goal0058 `sha256:d23435919ab571f552f349f689276fe8658976e095ddfcbf8e976834451ab069` |0/0; unchanged; candidate not deployed |
| app-api | Verified unchanged server image `sha256:a00c238eb7463929289f16b4675b26d8c13228e2fd27aafcf6e5963c63fe9306` |0/0; not rebuilt/started |
| MySQL | Retained goal0057 data and existing pinned image |0/0; not started/reset |
| Redis | Existing pinned image/configuration |0/0; not started |

No new Web image Digest exists. The existing stopped deployment is **not** relabeled Goal0059. No rollout failure, rollback, old cleanup script, scale action, namespace/PVC/PV deletion, network change or data repair occurred.

The existing test Web Service is ClusterIP10.105.112.89:443, API10.100.68.238:8888; no test Ingress was configured and there are no ready test endpoints. No usable external continuous HTTPS address or new candidate deployment is claimed. No port-forward was created in this round and no temporary URL is presented as permanent. If a later candidate passes, restore the4 owned services, use a verified immutable Web digest and legitimate TLS/access configuration, and **retain them running** as Goal0059 requests. Do not resurrect the old automatic scale-to-zero behavior. Access should use real test accounts through existing auth, with credentials provided privately; this report contains none.

## Resource cost, probe corrections and cleanup

Builds/browsers ran sequentially. Minimum sampled available memory: control-plane1.75GiB, HK4.36GiB. Both remained Ready; successful build/browser unit exits and resource samples showed no OOM-driven failure. No services were moved and no local dependency containers were started. Extra local browsers/servers/build workers stopped; the observer stop marker was set. No new CA trust, TLS secret, external proxy or test gateway was installed. The retained native build/isolated inputs and private evidence are owned by Goal0059 and retained seven days, through2026-11-17 UTC, for ChatGPT reproduction. No shared-resource deletion is authorized.

Private probe corrections were preserved separately: the initial Web test lacked CLI env and skipped the parser, then10/10 were rerun; the first disposal probe called hasEventListener on an already destroyed node, which correctly had a null processor, and was changed to directly inspect destroyed validity/processor absence; the first geometry probe requested a roomInput while still at home and was restricted to existing fields. Geometry uses each node's own transformed corners, not getBoundingBoxToWorld's union with active children. These are probe errors, not product fixes or failures. Original scripts/logs/JSON remain. The accepted UI and geometry runs have no unexpected page error. No credential or token was printed by this round's probes.

## Verdict and handoff

| Gate | Result |
|---|---|
| G58-N01 complete visual repair | **FAIL/not closed**; unique pairs PASS, pointer geometry FAIL, real contribution unverified |
| Cocos4 native export/strict types/BigInt/HPKE | **PASS** |
| Cocos Web required visual interaction | **FAIL — G59-N01/P2** |
| New Kubernetes deployment | **BLOCKED/NOT EXECUTED** by mandatory visual gate |
| Post-deployment business smoke | **BLOCKED/NOT EXECUTED** |
| Continuous access/services retained running | **BLOCKED/NOT DEPLOYED**; original0/0 state preserved |
| Overall Goal0059 acceptance | **FAIL** |

ChatGPT owns the P2 repair and a new pushed candidate. Then rerun affected native visual/real-contribution gates before image/deployment; only after post-deployment smoke passes mark it current and retain required services. This report does not authorize any production/source fix by Codex or invent a completed deployment.

Only this report and permitted Status/Working State/Completion Report text in `current.md` are delivered on main. The report commit is distinct from the tested product SHA. Final staged diff, secret checks, commit/push and clean-main results are reported in the handoff.
