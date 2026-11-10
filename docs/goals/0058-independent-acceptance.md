# Goal 0058 — independent Cocos4 Web and Kubernetes acceptance

Verified product: `c62d57200d69ec86d8012263afece1177e6359a9`, `oublie6/awesome-zero-platform`, clean synchronized `main`. Native checks ran on 2026-11-09; real business and recovery checks completed on 2026-11-10 UTC against the same source and artifact hashes. The eventual report commit is documentation, not a new product candidate.

**G57-N01/P1: PASS, closed for this candidate. Goal0058 stated A/B checkpoint: PASS.** The original [Goal0057 FAIL report](0057-independent-acceptance.md) remains unchanged. One nonblocking visual defect, G58-N01/P3, is recorded below. Goals0045/0046 remain `in_progress`; this is not release, device, production-domain or whole-product completion. Goal0056 Honor/ADB diagnosis remains user-paused; C09/C10 remain DEFERRED.

## Scope, authority and evidence

Codex independently exported and ran the real candidate, exercised actual Vue and Cocos UI against isolated Kubernetes Go/MySQL/Redis, and injected faults only into its own traffic/Deployment. No production code, assertions, generated game JS, protocol, crypto implementation or schema was repaired. No subagent was used. This is independent acceptance, not implementation self-review.

The owner is the client integration/acceptance operator. Public contracts are the existing same-origin auth, versioned room/hand HTTP/WSS and `azp-doudizhu-frame-v1` messages. Go owns game decisions; MySQL owns committed recovery material; Redis remains auxiliary. Vue carries authenticated user intentions, not cards, phrases or authority. Module dependencies and data ownership were not changed. Test-only configuration, keys and MySQL data were reused. Bounded sequential operation, retained evidence, unique temporary trust and scale-to-zero rollback define the operational/security boundary. These are executed facts within this fixture; broader architecture/release targets remain separate.

Private evidence root: `/root/.config/awesome-zero-platform/acceptance-0058-c62d572` (`E58` below). The reused private Goal0057 directory is retained separately and unchanged. Raw browser frames, test credentials, keys, cards, phrases and database evidence are deliberately not committed.

## Coverage index

| Required report item | Evidence/result |
|---|---|
| 1. Fixed product SHA | Exact candidate above; report SHA is separate |
| 2. Previous-candidate diff | Reviewed against `1301bb1df6f9521474f64d8dc0ce35431c7242ab`; details below |
| 3. CI baseline | Exact 3 runs checked; native export skipped in CI |
| 4–6. CLI/engine, target, shared packages | Real pinned Linux toolchain, Chrome79 options and clean rebuilds PASS |
| 7. Strict generated types | Actual generated declaration gate PASS |
| 8–9. Export and JS audit | Fresh output, all9 JS audited and hashed, native exponent preserved PASS |
| 10–11. Chromium/closure | Actual Entry/GAME_READY/UI/HPKE PASS; G57-N01 closed |
| 12. Kubernetes deployment | Existing namespace/data/API reused; new Web digest and Pod bytes verified |
| 13. Message bridge | All8 message types observed at their actual send/receive boundaries; positive and negative runtime checks PASS |
| 14. Accounts/rooms | 3 independent real Go accounts; create/join/ready/start PASS |
| 15. Full game | Actual Cocos UI, Go settlement, all3 COMPLETED/v60 PASS |
| 16. Crypto/fairness | Native HPKE, drand deal, own phrase positive/negative and Go final verification PASS |
| 17. Pod recovery | Checkpoint33/log through34, new epoch2, identical private state, continued game PASS |
| 18. Logs/resources | Raw logs/screenshots/metrics and memory bounds retained; details below |
| 19. Findings | G58-N01/P3 visual issue; original probe errors separately retained |
| 20. Verdict | PASS for the stated A/B checkpoint; no new P0/P1 blocker found |

## Source difference and CI

The actual diff changes Web target/options, build/test/type gates, BigInt AST auditing, Cocos documentation and CI, plus historical acceptance documentation. The runtime server, Docker API build inputs, schemas, client business/crypto/protocol sources and Vue implementation are unchanged. `git diff --quiet 1301bb1df6f9521474f64d8dc0ce35431c7242ab HEAD -- server deploy/docker/app-api.Dockerfile` passed. Reusing the verified Go image is therefore intentional, not an unverified rebuild shortcut.

Exact head SHA and job results were independently obtained and saved in `ci-run-*.json` and `ci-jobs.json`:

| Workflow | Run | Result |
|---|---|---|
| CI | [42634588406](https://github.com/oublie6/awesome-zero-platform/actions/runs/42634588406) | SUCCESS,6/6 |
| Fair Doudizhu | [42634588288](https://github.com/oublie6/awesome-zero-platform/actions/runs/42634588288) | SUCCESS,4/4 |
| Cocos Linux Native Client | [42634588261](https://github.com/oublie6/awesome-zero-platform/actions/runs/42634588261) | linux_source SUCCESS; linux_native_export SKIPPED |

These are the existing source baseline. No candidate CI workflow was manually retriggered or represented as a native build; any later report-commit CI is separate. Unrelated full Go/race/client suites and historical Creator3/WeChat/device work were not repeated.

## Real Linux export, strict types and AST

Used existing Node22.23.4; CLI `f506fa3de403b48b4833334fa48dddd27bb26364`; engine `d0a696cfe63fd8d19c70d7913ba11aa641c2e3c7`, version4.0.0-beta.0. `verify-toolchain.cjs` checked the actual installed Git revisions. No Wine, foreign Creator, engine replacement or tool reinstall.

Executed the unmodified repository commands with the installed CLI/engine environment:

```sh
node scripts/cocos-linux-wechat/verify-toolchain.cjs
node --test scripts/cocos-linux-web/test.cjs
node scripts/cocos-linux-web/build.cjs "$E58/native"
node scripts/cocos-linux-web/smoke.cjs "$E58/native" "$E58/smoke"
npm run build --prefix clients/game-web
```

Web tests:9/9 PASS,0 skipped, using the actual installed Babel parser. Build exit0. Fresh isolated root `/tmp/azp-native-cocos-web-mC3Y6B`; actual platform `web-desktop`. The CLI logged reading its `web-build-options.json`, which contains `buildScriptTargets: "chrome 79"`. The selected CLI source passes this option into its module transpilation configuration. All3 file-linked packages (`game-client-core`, `doudizhu-client`, `secure-envelope`) had their private isolated `dist` removed and were compiled from this candidate; rebuild logs and entry hashes are retained in `isolated-package-rebuilds.json` and the build inputs. No historical `dist` was accepted as the build input.

The CLI initially printed the upstream TS2688 relative-type path failures (`cc`, custom macro, env, jsb). They were **handled by the candidate's explicit post-export path repair and mandatory strict compiler gate**, not silently ignored. Actual declarations came from this current generated engine/project tree. The saved original/final generated configs and `effective-tsconfig.json` show all11 business scripts, including GameBootstrap, with strict/null/implicit-any/property-initialization checks active. Final compiler exit0; the wrapper explicitly printed `Real generated Cocos4 TypeScript declaration gate passed`. Existing `skipLibCheck` remains in effect; this does not claim strict checking of every engine library declaration. Unused built-in 3D cmft/FBX importer failures and a Browserslist warning are retained in `build.log`; they did not prevent the actual 2D scene, strict business check or export.

Output `E58/native`:25 files,16,449,600 bytes, full per-file SHA256 inventory `native-artifacts.json`. The formal build audited8 generated JS before adding the public default config; the independent final audit checked **all9 JS,15,162,381 bytes**. Both required native BigInt exponent presence. Independent AST counts:1 literal BigInt exponent,0 literal BigInt `Math.pow` calls; the candidate's broader alias/exponent audit and its real-parser tests also passed. No exported JS was patched.

| Native JS | Bytes | SHA256 |
|---|---:|---|
| `application.js` | 1269 | `60633140454acc1070f4eac53f89828e86f2c8471bb8f80e77e42fac68b4ff3a` |
| `assets/internal/index.js` | 259686 | `edd79bd3246940d6acd8adcaa6093c8f1c4e635af2c203f59684696b390fdf58` |
| `assets/main/index.js` | 389737 | `c434603a494ad08ab753567564fb46ef70dd7e6246605601c02ec5289ecbab15` |
| `cocos-js/cc.js` | 12622964 | `d0b4d7872199c93f0b86eddf5d19d80152fae2cd9d84a793261d39d0c5d58b9f` |
| `game-config.js` | 44 | `953cc4b5ce1e74bd6352c2efa2a6320c5a57fda3301050bc56c59a5b88e8dc26` |
| `index.js` | 615 | `2e2d7f9d93e071156db972f33f7a7b03b3ab6c65d865c3b83009c261929a6042` |
| `src/chunks/bundle.js` | 1790581 | `93fe2851965d8c4b8b4ea216b312d67fd6e851465af9ec0cf651bb925bcc3733` |
| `src/polyfills.bundle.js` | 83 | `77d21eb18bb5296dd7ef0e66a678af45cc5f297bc64b0dbcae29f49577462ab7` |
| `src/system.bundle.js` | 97402 | `40d96cddea8facc054c39ab741ecd373f486ac0e97e1ba770ed7b618c924f311` |

The staged public `game-config.js` is the documented deployment-config replacement only:202 bytes, SHA256 `b07431f2418710b4cd0e74e5a661ab2f87c6f17e87d2c42c1dc0c639ae1240ec`. All other game JS bytes are identical. All9 deployed Pod hashes match the staged files.

Actual Chromium156.0.8078.4 / Playwright1.64.1 executed the export. The official smoke passed RoomEntry, LoginEntry and CancelWebLogin with no page errors/failures. A separate actual iframe/parent fixture observed Entry, Canvas, Chinese home title and genuine `GAME_READY`. Runtime `2n ** 256n` produced the expected BigInt. Actual exported WebRandom produced distinct nonzero32-byte draws; actual exported HPKE sealer encrypted, an independent Node receiver opened the payload, and wrong AAD was rejected. This local crypto fixture is explicitly separate from Kubernetes gameplay; the latter also submitted3 actual native HPKE contributions. Browser root launch used `--no-sandbox`, not a TLS bypass. **AST plus actual scene/interaction/crypto execution close G57-N01.**

## Kubernetes, TLS and deployed bytes

Revalidated context `kubernetes-admin@network-learning`, both nodes Ready, namespace `awesome-zero-game-test`, retained isolated database `goal0057`, existing configuration and all15 secret values against the private baseline without exposing values. Initial retained DB had4 accounts,0rooms,0hands. No schema apply, reset, new shared pool, PVC or shared namespace mutation. Test components retain the previously stable control-plane placement; HK was not presumed OOM-faulty.

Vue production build succeeded using existing pinned dependencies. Standard staging used the fresh native export, preserved exact auth/game paths and WSS forwarding, and recomputed CSP hashes for the actual HTML. `kubectl kustomize`, saved diff (expected exit1), apply and sequential MySQL→Redis→API→Web rollout passed. The only persistent manifest difference was the new Web image. API Pod replacement later was a deliberately scoped fault injection.

| Image | Verified identity |
|---|---|
| Reused Go API | `docker.io/awesome-zero-platform/goal0057-api@sha256:a00c238eb7463929289f16b4675b26d8c13228e2fd27aafcf6e5963c63fe9306` |
| New Web | `docker.io/awesome-zero-platform/goal0058-web@sha256:d23435919ab571f552f349f689276fe8658976e095ddfcbf8e976834451ab069` |
| Web Docker config | `sha256:3a7cd730a3e384ceac79f1ff16686855b359cdf4eadd1f8a2e96e5d78d7ff0c3` |

Actual HTTPS/WSS traffic traversed the test Web Service and API Service through loopback29457 forwarding, not Vite. Reused valid private CA/leaf, hostname/IP verification and Chromium trust passed; no `ignoreHTTPSErrors`, ignored certificate errors or fake successful responses. The uniquely named temporary trust was added to both user NSS stores. Real Go identity, Bearer-authenticated WSS, path forwarding and all game assets worked; WS URLs have no tokens. No cross-origin sessionStorage sharing was needed.

## Real bridge, three players and fairness

Real passive parent/frame message listeners observed GAME_READY, SESSION_AVAILABLE, SESSION_ACTIVE, OPEN_ROOMS, CREATE_ROOM and JOIN_ROOM. The positive flow called real Go identity restoration before authenticated business WSS. Actual Cocos/room state and wire ACKs prove the room intentions reached the existing coordinator. The default labels issue below does not prevent these interactions.

The final bridge matrix passed5/5 checks: actual CREATE intent plus recovered existing-room duplication guard; real same-origin sibling and opaque-origin messages rejected by parent/child boundaries; real Go logout/revocation followed by a native SESSION_FAILED; actual Vue SESSION_REVOKED send and immediate iframe destruction/token clearing; and a real successful old iframe auth response held across logout/account switch and released after B established a fresh authenticated frame. Parent account and new frame stayed B. The old request's cancellation is an iframe-destruction boundary check, not a claim that a destroyed controller executed a late callback. Native Vue sends were observed with CDP breakpoints in the unchanged built asset; no postMessage wrapper or manufactured SESSION_ACTIVE was used in the accepted run. Legitimate SESSION_REVOKED is qualified as a native send plus iframe destruction, not presumed receipt by an already removed child. Deliberate revocation produced2 expected401 resource errors; no additional unexpected bridge error was found.

The exported message keys contain no token, password or private game material. Identity is validated via the existing authenticated owner APIs, not accepted from bridge claims. Separate-context logins, original/current source checks, actual disposal, real delayed response cancellation and current B authority are runtime evidence; earlier source-only historical tests are not substituted for these checks.

Three independent BrowserContexts used three retained real Go player accounts; each identity was checked against its authenticated Go session. A’s actual Vue CREATE produced room version1; B/C actual Vue JOIN produced versions2/3. Each actual Cocos refresh saw3members; ready advanced4/5/6; owner start adopted a real custodied hand. No direct Go client substituted for these UI actions. Existing-room create replay was rejected by the recovered client coordinator without allocating a second room.

Each player entered its own distinct phrase into the actual Cocos EditBox, clicked submit and received a successful native HPKE contribution confirmation; input length became0. The real future drand source initially returned a transient relay500, then verified successfully without fallback or fabricated beacon. Three actual authorized projections reached BIDDING with3 distinct positions and51 distinct private dealt cards. The actual current bidder called3 through Cocos, producing PLAYING. Browser storage evidence contains only the session token key and no localStorage keys; fresh-context recovery demonstrably used server discovery, not saved cards. Messages carry protocol/type and the nonsecret room ID where needed, not token/password/phrase/cards. Private material remains within each authorized game projection and encrypted server custody.

The final UI evidence is backed by the existing Go `FinalEvidenceService.Get` → `livehand.VerifyFinalRecord`, which validates the archived game rather than returning an unchecked record. The browser validates the returned contract and displays that server verification; this is **not** a claim that Chromium independently reimplemented BLS/shuffle/game replay. Actual local own-contribution verification also checked each reentered phrase against that player's authenticated final-only random/commitment. Original crypto/protocol sources are unchanged.

## Complete game and real failures/recovery

`rooms-contributions.json` and `full-game.json` contain the actual UI sequence and private wire evidence. A real lead-pass was rejected, leaving the acknowledged version unchanged and no UNKNOWN action. A real legal play was then forwarded to Go, its genuine committed result was dropped by a WebSocket proxy, and the socket was closed. The client entered UNKNOWN. Actual room reconnect and RetryUnknownHand resent the **identical** wire payload/RequestID/clientSeq/expectedVersion. Go returned the retained receipt; a contemporaneous isolated MySQL query found exactly1 action row for that ID. No fake business result was supplied.

The game used one legal single-card lead from the landlord and actual opponent pass buttons until completion; Go chose turns, checked legality, maintained cards and settled. Including the bid,59 accepted live actions produced final version60. All3 clients obtained COMPLETED/v60, winner seat3, points `[-6,-6,12]`, and the identical verified record digest:

`67cd8da2737540f225828a10e631a79edb207f37a73f9fb15378e3bf27e5c01e`.

Each player's correct original phrase matched; a wrong phrase was rejected; the input was cleared. All3 screenshots and raw proof statuses are retained. Gameplay/full-game report has0 unexpected browser errors and0 HTTP error statuses; injected socket loss and aborted timeout are intentional fault paths, not successful fake responses.

At live version34, an actual room HTTP read was held past the unchanged15s client deadline. The client reported failure; subsequent real refresh succeeded. Then only `deployment/app-api` in the test namespace was restarted. Before restart: owner_epoch1, committed_version34,33 contiguous action rows (versions2–34), checkpoint33. Original Pod `app-api-548f7676fc-rk8xx` was replaced by `app-api-985c8f9f5-nj6wc`; UIDs differ. New process recovered epoch2 and watermark34. Each Vue page was refreshed, its iframe recreated, and the existing server hand rediscovered. The exact private state digests (hand, version, position, cards, public projection) matched for all3 players. They continued from35 to the same terminal60: no ACK loss, duplicate execution, new deal or seat reassignment. The full sequential rollout/three-UI-reload comparison took158.386s under the browser CPU cap; this is not a production RTO benchmark or zero-interruption failover claim.

## Finding G58-N01/P3 — Cocos4 EditBox default labels remain visible

Owner: Cocos client bootstrap; no backend/data impact. Fresh actual native room screenshots show overlapping room-ID labels; an empty custodied phrase field shows a large white `label`. Read-only inspection found `PLACEHOLDER_LABEL("label")`, `TEXT_LABEL`, and handwritten `RoomCodeText` together. `createRoomInput()` adds EditBox to an active node and then supplies a new textLabel. The pinned engine creates default text/placeholder nodes when these are missing; replacing the reference does not remove the earlier node, and no intended placeholder is configured. Source: `clients/fair-doudizhu-cocos/assets/scripts/bootstrap/GameBootstrap.ts`, createRoomInput (around308); current engine EditBox textLabel setter and label initialization corroborate the runtime nodes.

Minimal reproduction: open actual Vue→Cocos room, inspect the input before typing; after joining/creating, two room strings can overlap; after submitting/clearing a custodied phrase, `label` is visible. Evidence: original `room-phase-0.png`, current `full-game-{0,1,2}.png`, `bridge-created-room.png`, `bridge.json` inputLabels. The original room screenshot is from the earlier retained probe attempt, not mislabeled as a final table screenshot. This was not repaired. Input, HPKE submission, all card controls and own-phrase checks still work. It is recorded as a nonblocking visual defect under Goal0058's explicit no-new-P0/P1 blocking threshold, not silently treated as full client/release completion.

## Probe corrections, resources and cleanup

Private probe errors were retained, not relabeled as product failures: (1) first trusted-browser launch lacked the CA in the legacy NSS store; trusting the same private CA in both user stores resolved it without bypass; (2) immediate iframe lookup raced navigation, fixed by waiting for the actual iframe URL while retaining the created room; (3) an initial postMessage wrapper changed the incumbent source realm, so it was removed entirely and native Vue sends were observed with CDP breakpoints instead; (4) attempted new create ignored the valid retained-room precondition and correctly received `already managing a room`; (5) a srcdoc probe used location.origin=`null` as targetOrigin, corrected to an actual sibling/opaque-frame message with a valid target. Original scripts/logs/JSON are retained. The current bidder uses the published `public.bidding.currentBidder` field. No formal assertion or product implementation was weakened to fix a fixture.

All memory-intensive work ran sequentially: export, Vue/image build and each browser phase. One Chromium process tree at a time was capped at1GiB, swap0, CPU150%; no build overlapped the three-context game. Full-game browser samples reached about0.90GB, below the cap; saved cgroup memory events showed0 max/oom/oom_kill, and the unit exited successfully. Kubelet stats (the Metrics API was unavailable) sampled both nodes: minimum available memory about2.20GiB on control-plane and4.45GiB on HK. Both remained Ready in this round. No rebalancing was necessary; no shared services were moved. Original HK interruption is historical environment evidence, not asserted as OOM or a new incident.

Raw source/AST/types inventories, CI metadata, images/manifests/diff, UI/protocol JSON, screenshots, before/after API and Web logs, resource samples and metrics remain in E58. API endpoint absence during deliberate replacement is an expected observer error, not a successful health sample. The same9 shared platform Pods retain their names, UIDs and specs compared with the retained baseline.

Cleanup verified: all browser units stopped; owned29457 forward stopped; both Goal0058-Reused-CA NSS registrations removed; all4 test Deployments scaled0 via the reviewed existing stop script;0 test Pods remain. Namespace, isolated data/protection root, images, manifests and evidence are retained for seven days, through2026-11-17 UTC, for diagnosis/reproduction. E58/k8s records the new Web image; the previous evidence and old image are preserved separately. Reproduction must recheck context/ownership, trust expiry and available resources before starting. No automatic deletion or shared resource cleanup is authorized by this report.

A further **operator/probe security incident** is recorded separately from product findings: the first Node-side route.fetch did not trust the internal CA, and its raw error inadvertently surfaced one isolated test access token in tool output. The session was immediately revoked through real HTTPS (200), and subsequent session validation returned401; `probe-token-revocation.json` retains the result without the token. NODE_EXTRA_CA_CERTS then enabled normal CA verification for the successful final real-response probe; no TLS bypass was introduced. Raw failure headers stay private and are not committed. No signing/protection root or production credential was exposed by that incident.

## Verdict and remaining work

**G57-N01 PASS and closed. Goal0058 stated A/B checkpoint PASS, with nonblocking G58-N01/P3 returned to ChatGPT.** Real pinned export/strict types/AST/Entry/crypto, all8 bridge messages at their stated boundaries, three actual UI players/full game/final verification, exact UNKNOWN retry, HTTP deadline and dedicated Pod cold recovery have execution evidence. No new P0/P1 blocker was found. The P3 display defect is disclosed under the objective's explicit blocking threshold; it is not repaired or represented as complete client polish.

This verdict applies to the fixed product SHA and the isolated Linux/Chromium/Kubernetes fixture. It does not complete or archive Goal0045/0046, close paused Goal0056 device startup, qualify real mobile/WeChat domains or production recovery/performance, or change C09/C10. ChatGPT owns the P3 client fix and subsequent evidence. Historical FAIL reports retain their original conclusions.

Only this report and the permitted Status/Working State/Completion Report text in `current.md` are changed. Final diff/source/historical-report/secret checks and commit/push are recorded in the delivery handoff. No product code fix, schema change, branch or PR is part of this acceptance delivery.
