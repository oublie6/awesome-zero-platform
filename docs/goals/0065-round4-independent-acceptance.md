# Goal0065 Round4 — prioritized failure report / FAIL

2026-11-23 UTC. Fixed candidate `5564253ae3d4e6296c846cdd1133a72bb50a2d6a`, clean synchronized `main`. User subsequently requested **“先报告失败情况吧”**. This report prioritizes the first confirmed blocking failure; it is **not a completed full Round4 acceptance sweep**. No product source was modified. Goal0065 remains **in_progress**, and no candidate deployment occurred.

## G65-R4-N01 / P2 — failed room restoration still completes startup

The Round4 objective explicitly requires that successful login with failed room restoration must not display “游戏就绪”. Actual fixed-candidate Cocos/Chromium behavior violates that requirement.

Reproduction:

1. Log in with an owned ordinary QA account and enter the real exported Cocos client. Confirm its existing room is successfully recovered: room `room-cmd-94b10a02027fdfa28b1fe8e570d8ca22`, authenticated/WSS ready.
2. In an isolated browser context, intercept only `/v1/doudizhu/me/rooms` and return controlled HTTP503. Reload the website and enter game again. HTTP account verification and the actual WSS connection remain healthy.
3. Observe startup messages and both loading covers.
4. Remove the fault and explicitly reload game; the same original room is recovered successfully.

Observed:

- The actual discovery fault was hit once. Account/session requests return200; `authenticated=true`, `transportReady=true`.
- Client sends `document → engine → scene → session → GAME_READY → SESSION_ACTIVE`.
- Parent displays **“游戏就绪”**, parent cover is absent and native overlay count0.
- Client `roomId` is empty. It renders the generic create/join lobby and replaces the restoration error with “和好友开一桌，或输入房间号加入”.
- With the fault removed, explicit reload recovers the same room, with usable transport. Server-owned room data was not lost.

Expected: surface the failed required restoration and offer retry; do not announce SESSION_ACTIVE/remove the startup failure UI until that restoration succeeds. Successful identity, socket and finite presentation state do not prove room restoration success.

[Runtime events, control and recovery](evidence/0065-round4/first-fail.json), [working recovered-room control](evidence/0065-round4/room-restore-control.png), [failed restoration reported as ready](evidence/0065-round4/room-restore-failed-ready.png). Pageerror0; one expected resource503 console entry from the fault. The503 is a controlled browser fault injection, not an assertion that the deployed Go service is returning503.

Root: [login restoration](../../clients/fair-doudizhu-cocos/assets/scripts/bootstrap/GameBootstrap.ts#L701) catches `recoverMyRooms` failure, displays a temporary notice, and resolves `login(true)` normally. [Host handshake](../../clients/fair-doudizhu-cocos/assets/scripts/bootstrap/GameBootstrap.ts#L198) checks authentication, WSS transport and scene geometry, then renders rooms and emits SESSION_ACTIVE without the room-recovery outcome. Rendering rooms also overwrites the temporary recovery-failure notice.

Repair handoff: propagate an explicit restore result/error to the startup owner and include successful required room restoration in the readiness decision. Retain a visible error and a targeted recovery/retry path; preserve current authenticated identity and existing UNKNOWN request identity. Do not allocate replacement persistent commands to conceal a failed read. Add a real valid-account/WSS-success plus room-read-failure regression with an existing-room positive control. Codex did not repair production code.

## Executed source/build verification

Read AGENTS, current Goal, canonical principles and Round1–3 findings/evidence; inspected the Round3→Round4 source diff. Initial clean `main` fast-forwarded to exactly the fixed candidate. Go server/shared protocol/schema were not changed by this candidate or this acceptance work.

Verified exact-SHA workflow jobs: [CI6/6 SUCCESS](https://github.com/oublie6/awesome-zero-platform/actions/runs/45156899093), [Fair Doudizhu4/4 SUCCESS](https://github.com/oublie6/awesome-zero-platform/actions/runs/45156898926), [Vue SUCCESS](https://github.com/oublie6/awesome-zero-platform/actions/runs/45156898905), [Cocos linux_source SUCCESS](https://github.com/oublie6/awesome-zero-platform/actions/runs/45156898936), native export **SKIPPED**. [Job inventory](evidence/0065-round4/ci.json).

Independently built the fixed archive once with official Linux cocos-cli/Cocos4.0.0-beta.0, pinned CLI `f506fa3…`, engine `d0a696c…`, Node22.23.4. Real Web export40s; actual generated declaration strict gate and BigInt AST gate pass, eight exported JS files. Vue `vue-tsc`/Vite build passes. Independent Vue `npm test`14PASS, `node --test scripts/cocos-linux-web/test.cjs scripts/game-web/stage.test.cjs`26PASS, Cocos `npm test`163PASS: **203 tests, zero skips/failures**. [Build proof](evidence/0065-round4/build-proof.json), [actual native hashes](evidence/0065-round4/native-export-hashes.json), [staged manifest](evidence/0065-round4/artifact-manifest.json).

Runtime: Chromium156/headful Xvfb/SwiftShader with trusted isolated CA, no ignored TLS errors. Actual fixed-candidate Cocos/Vue served by temporary pinned Nginx on localhost30465, same-origin real Go HTTPS/WSS proxy. CSP hashes calculated from the staged HTML and cache map installed; `nginx -t` passes. [Fixture proof](evidence/0065-round4/fixture-proof.json). Only owned QA credentials used; no private tokens, phrases, cards or ciphertext in public evidence.

## Scope deliberately not claimed

Following the user's request to report failure first, no full suite was continued after the confirmed blocker. **R3-N01 offscreen parked initialization and R3-N02 border correction have not yet received complete Round4 runtime reacceptance; neither is declared PASS or still FAIL based solely on source.** Camera/UI matrix sampling, all three viewport checks, the expanded120-frame gate/corruption/destruction scenarios, full hidden/resize/input isolation cases and complete retry-race sweep remain unexecuted for this candidate.

No fresh Round4 full three-player game, UNKNOWN fault suite, cold/hot/cache/performance comparison or complete account-isolation sweep was run. Prior Round3 business/cache evidence is preserved as historical evidence, not relabeled as a Round4 pass. No unrelated Go pressure/crash suite was repeated.

## Environment, deployment and rollback

No Kubernetes candidate image build/publish/rollout. No Go API/dependency configuration, schema, Secret, PVC, production or historical-user-data mutation. The browser-only503 route was removed and successful restoration positively verified before cleanup. Existing Web `sha256:44995360be6e91a0e7e1c355db4e8e9027633d1fc4d9eb59b15b110990af0a0f` and API `sha256:04614d0bc04c1fe37d2cb4467be23fec728cb0b1cfb63c26b1cd3746943ec82f` remain current/rollback baseline.

[Final baseline comparison](evidence/0065-round4/final-audit.json): six Ready Pods with unchanged identities/images/restart counts and unchanged Deployment/ConfigMap/Service/PVC/StatefulSet baselines. Hands remain9ABORTED/8COMPLETED, unfinished0; no new hand was started. Temporary browser19175 and Nginx30465 stopped; existing user29457 forward retained. No rollback was required because no rollout occurred. Deployment/postdeployment/production acceptance is **NOT EXECUTED**.

Private restricted build/harness/log evidence: `/root/goal0065-round4-5564253`; public evidence is whitelisted and [checksummed](evidence/0065-round4/sha256.json). Report/source links, hashes, sensitive-data exclusion and final diff are reviewed before publication.

**Prioritized result: FAIL (one confirmed P2).** ChatGPT owns repair and regression; Goal0065 remains in_progress. This early failure report does not close the remaining Round4 checklist. Resume complete acceptance at the agreed candidate after repair before any Web-only deployment or Goal completion.
