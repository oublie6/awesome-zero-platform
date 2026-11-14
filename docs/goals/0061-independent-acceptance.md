# Goal 0061 — independent incremental acceptance

2026-11-14 UTC. **Overall BLOCKED: configured test TLS expired. G60-N01 input repair PASS.** No new confirmed P0/P1/P2 product defect in the executed scope; business/deployment acceptance remains outstanding.

## Source and CI

- Tested product: `1dd54dbf64c3b8375ffa3fe08dfe2a614d34b9f3`, clean synchronized `main` before verification. This report's documentation commit is separate from the tested product.
- Increment from Goal0060 report commit `c8899c71e957947a093254cbb627fd7eca16f269`: four files, EditBox initialization order/single-line configuration and affected smoke/source regressions. No product changes by Codex.
- Exact-SHA Actions: [CI 43583401611](https://github.com/oublie6/awesome-zero-platform/actions/runs/43583401611) SUCCESS (6 jobs); [Fair Doudizhu 43583401606](https://github.com/oublie6/awesome-zero-platform/actions/runs/43583401606) SUCCESS (4 jobs); [Cocos Linux Native Client 43583401624](https://github.com/oublie6/awesome-zero-platform/actions/runs/43583401624) SUCCESS, **native export SKIPPED**. Actual local export below supplies that evidence.

## Executed checks

All private evidence is under `/root/.config/awesome-zero-platform/acceptance-0061-1dd54db` (abbreviated E61). Logs, dummy-input screenshots and artifacts remain private; no real user passwords used.

| Check | Observed result |
| --- | --- |
| `node --test scripts/cocos-linux-web/test.cjs` | 12/12 PASS, real AST parser, no skips |
| `node --test --test-name-pattern='G58-N01\|G59-N01' clients/fair-doudizhu-cocos/test/assets.test.mjs` | 2/2 affected tests PASS |
| `node scripts/cocos-linux-wechat/verify-toolchain.cjs` | Installed pinned toolchain PASS |
| One `node scripts/cocos-linux-web/build.cjs E61/native` | Real Linux Web export PASS; generated strict types PASS; eight generated JS AST/BigInt checks PASS |
| Unmodified `node scripts/cocos-linux-web/smoke.cjs E61/native/fair-doudizhu-web E61/smoke` | One run, 31 checks PASS; no errors/failures; portrait-to-landscape sequence PASS |
| Existing Goal0060 input fixture, restricted to current affected paths | 12 case groups PASS across three sizes; two disposal cycles per size |
| Existing detached-native-editor event fixture | PASS: old strings unchanged and current input unaffected after destruction |

Node22.23.4; CLI `f506fa3de403b48b4833334fa48dddd27bb26364`; engine4.0.0-beta.0 `d0a696cfe63fd8d19c70d7913ba11aa641c2e3c7`. Build options: web-desktop, chrome79, strict=true. No Wine, foreign OS, patched output or repeated build. Root Chromium launch adapter reused from Goal0060; assertions were retained. Input fixture omitted already-covered extra rotation and reduced repetitive disposal from eight cycles to two.

Artifact: **25 files / 16,452,615 bytes**, `E61/native/fair-doudizhu-web`. File hashes/options are in `artifact.json`, SHA256 `5f88b2c5c35d5ea381ecee708c4b16f1e3b504780ef6260a1ddeebaa2a640c12`. The ninth JS, default `game-config.js`, is byte-identical to Goal0060 (SHA256 `953cc4b5ce1e74bd6352c2efa2a6320c5a57fda3301050bc56c59a5b88e8dc26`); its prior AST result is reused. No new native crypto run.

## Input result and evidence

At **1280×720, 390×844, 844×390**, password editing uses actual focused **INPUT/password**, visually masked while typing; username uses INPUT/text and strips pasted newlines. Real typing, Backspace, clear, blur/refocus and re-entry pass. Room input remains functional. Each field has one text/placeholder pair; clicking visible hint text focuses its corresponding editor; input/button bounds do not overlap. Maintained resize smoke passes, resolving the previously observed rotation gate in this run.

Repeated room/login replacement leaves the expected one/two native editors, destroys old nodes and event processors, and detached old-editor events cannot mutate the replacement. Evidence: `smoke/`, `input.json`, `input.log`, `late-dom.json`, `password-editing-{1280,390,844}.png`; focused portrait screenshot also visually inspected. The standalone submit reaches the missing-configuration path; **this is not authenticated business integration evidence**. G60-N01 closes for this tested native Web input scope.

## Reused evidence

Source comparisons confirm unchanged `server`, `clients/packages`, `clients/game-web` and Cocos game controllers. `reuse.json` records source SHAs and independently checked evidence SHA256 values (manifest SHA256 `73de8042b3176def29bde3704a40f3f546ff0d2028deb1ba0963e642bc5ebdb5`).

- Goal0060 `6e7e63c9199b1b6bd1e5b60d88b77801dbf9f43e`: native crypto/HPKE and browser zoom evidence retained. Current affected input/destruction paths were rerun above.
- Goal0058 `c62d57200d69ec86d8012263afece1177e6359a9`: full-game/fairness/durable-recovery and bridge/account-boundary evidence retained in its original scope (`full-game.json`, `bridge.json`, and linked [report](0058-independent-acceptance.md)).
- No unchanged Go full suite, full game, crypto matrix, stress or crash campaign rerun. Historical PASS does not imply fresh candidate business/deployment PASS.

## Deployment blocker and resource state

Context `kubernetes-admin@network-learning`, namespace `awesome-zero-game-test`: actual `g57-tls` leaf **expired 2026-11-10 12:43:04 UTC**, checked on2026-11-14; OpenSSL validity check fails. SHA256 fingerprint `98:D4:7C:D8:83:9A:09:C1:74:44:04:46:3C:B0:79:52:99:96:CE:D3:E9:08:A3:1D:1D:39:FE:6C:A7:EC:00:E8`. See `resources.log`. This is an environment blocker, not a newly confirmed input defect.

Per Goal0061 TLS gate, no bypass/new trust root, image publication, service startup, rollout or data/schema/configuration change was performed. Before and after: app-api, game-web, MySQL and Redis Deployments all **0/0**, no test Pods; both nodes Ready. Existing stopped services were preserved; no scale-to-zero cleanup was run. There is no successful candidate deployment to claim retained running.

**Stage3 minimum Vue/Cocos real login, room create/join, contribution submit/clear/page-switch and account isolation, and stage4 HTTPS/WSS/business smoke, Ready/restart/resource checks are BLOCKED** on valid configured TLS and subsequent test-service startup. Local input evidence does not substitute for these checks.

Build and browser checks ran sequentially: build cap3GiB, browser cap1GiB, swap disabled; sampled build memory about1.30GB, node available memory2.17/4.00GiB. Final local available memory4.33GiB, swap0. No resource-driven service redistribution was necessary. Temporary build/browser units and fixture HTTP servers exited; verified artifacts retained for reuse. No product repair or independent deployment qualification is claimed.

Resume with valid test TLS: reuse this exact source/configuration artifact, execute only outstanding minimum business and deployment gates, then retain successfully deployed test services running. Goals0045/0046 remain `in_progress`; Goal0056 Honor/ADB remains paused; C09/C10 remain DEFERRED. **Goal0061 is not completed.**
