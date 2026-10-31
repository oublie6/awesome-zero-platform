# Goal 0045: Cross-platform Fair Doudizhu client foundation

## Status

- State: in_progress
- Started: 2026-10-31
- Completed:
- Blockers: Native Creator builds, real Web checks and Linux community WeChat scene runtime passed. User-owned Mini Game AppID/scan login resolved the import blocker; actual wx SDK adapter probes passed for login code, entropy, storage and lifecycle registration/removal. WeChat HTTP/WSS, lifecycle event delivery, share and devices remain unqualified; Douyin SDK/devices remain unqualified.

## Goal

Implement the first *verified* cross-platform Cocos client foundation with a small reusable TypeScript core/port seam and separate Web, WeChat Mini Game and mainland Douyin Mini Game adapters/build targets, without changing the existing Go server or business rules.

## References

- `AGENTS.md`
- `docs/architecture/fair-doudizhu-cross-platform-client.md` (accepted design)
- `docs/api/fair-doudizhu-protocol-v1.md`
- `docs/architecture/doudizhu-realtime-v1.md`
- `docs/operations/realtime-websocket.md`
- `clients/fair-doudizhu-cocos/README.md`
- `clients/packages/secure-envelope/`

## Deliverables

1. Audit actual Cocos 3.8 project layout, current build scripts and SDK/tool availability; preserve code-first scenes and existing code.
2. Introduce minimal pure TypeScript client-core and Doudizhu-specific boundaries as justified by working code; platform interfaces/adapters for Web, WeChat, Douyin with runtime capability checks. Keep Cocos and platform SDK imports out of shared core.
3. Implement a minimal but working landscape entry shell and lifecycle for each supported build target, with safe-area/resizing checks. No invented production APIs or false game completion claims.
4. Reuse existing HPKE package and verify secure random source capability for each target. Never use `Math.random` as secret entropy. Fail closed when secure random is missing.
5. Validate transport authentication capabilities across platforms against existing WebSocket contract; if a platform cannot support the secure handshake, document a specific future server-side ticket contract rather than weaken security in this goal.
6. Automated contract/dependency tests and the strongest available Creator/target build and runtime smoke tests, clearly labeling simulator, developer-tool and real-device coverage.
7. Update docs with actual paths, commands, verified results and blockers; CI, self-review, archive/reset current on full acceptance.

## Constraints

- Follow `AGENTS.md`; work on clean synchronized `main`, no PR/feature branch.
- No backend changes, database schema changes, live business enablement, new generic game framework or heavy speculative abstraction.
- Gameplay and server-side authority remain unchanged. Don't duplicate business rules in client core.
- Do not claim a target is validated based solely on a mocked adapter. Report missing Cocos editor or platform SDK capabilities as blockers.
- Do not commit credentials, editor caches or generated binaries. Avoid modifying unrelated Admin Web or observability deployments.
- Do not archive when an essential goal acceptance check is blocked.

## Acceptance Criteria

- Existing Cocos project remains importable, and one documented bootstrap/build path works for available tooling.
- Pure TypeScript core import graph is free of Cocos, browser globals and platform SDK dependencies; dependency checks run in CI.
- Platform adapter contract tests pass with explicit capability/failure checks and no token-in-URL fallback.
- Existing secure-envelope package used unchanged; negative tests prove missing entropy does not silently fall back.
- At least one target is exercised end-to-end through its real supported runtime; other targets have clearly classified artifact/build/dev-tool/real-device verification results or explicit blockers.
- Common scene/entry and safe-area layout smoke evidence captured; current Go server, data and deployments unchanged.
- Goal report records actual source tree, tests, environment limits and future goals. Archive only after verifiable acceptance.

## Agent Strategy

Single primary implementer; use subagents only for independent review or API documentation checks. Avoid concurrent writes.

## Working State

### Completed

- ChatGPT's accepted cross-platform design and first-pass implementation retained. Codex phase review examined Goal 0045 client history through 1642121f01b4d94f1de90a282d461522ff934549 and actual imports, owner APIs and Go dispatcher/realtime structures.
- Reproduced five missing transport-lifecycle regressions before fixes. Corrected MiniSocketTask creation versus onOpen, synchronous callbacks, early failures/close, cancellation and old callback isolation. Shared connection cleanup is used by Web and injected mini-game adapters.
- Corrected WeChat entropy to UserCryptoManager; missing Douyin Mini Game entropy still fails closed. Corrected native HTTP abort, capability errors, session empty values and target-specific share callback handling.
- Corrected same-version full-snapshot confirmation after reconnect and bounded HTTP/socket opening waits with cancellable 15-second defaults. The duplicate-confirmation regression failed before the correction and passed afterward.
- Corrected duplicate/mutable snapshots, terminal action gating, command envelope/version validation and router unsubscribe lifecycle. Shell initialization is idempotent, late connection completions cannot change disposed/replaced state, and hand.sync includes the exact required protocol version. Cross-hand notifications do not mutate the selected hand.
- Added Entry.scene with Canvas/camera/bootstrap binding and stable Meta assets; responsive UI uses SafeArea and unregisters resize events. Corrected missing Component override modifiers discovered against official 3.8.8 declarations.
- Added a sequential source verification script and Fair Doudizhu client_foundation CI job. New standalone checks pass: game-client-core 26 tests, doudizhu-client 7, Shell/assets 7. Existing unchanged secure-envelope 9 tests also passed separately. Production backend code, data schema and deployments are unchanged.
- Real Chromium exercised WebHttp login, WebSocketTransport bearer authentication and ping/pong against the real Go service through a temporary local self-signed TLS bridge. Invalid token was rejected; secure entropy and session storage passed; no page errors or tokens in query strings. The existing read-only acceptance account was used and its owned session logged out. TLS bridge/port forwarding/browser processes stopped.

- Installed official Creator 3.8.8, WeChat 2.01.2510260 and Douyin 4.5.7 on the data disk; attempted Wine 6/11 startup sequentially. Creator Wine was stopped at low available memory; WeChat/Douyin did not yield qualified simulators. All owned processes stopped; available memory recovered to ~4.3 GiB. Removed ~2.4 GiB redundant archives/payload/old prefix; data disk ~64%, ~14 GiB free.
- Native Windows Creator CLI import and all three target builds passed (exit 36); both mini-game game.json outputs verified landscape. Actual generated-config strict TypeScript check passed. Fixed source .js/TS resolution, package exports/library configuration, viewport cropping/framebuffer distortion, portrait font size and UI ownership cleanup.
- Actual built Web entry passed at 1280×720, 844×390 and 390×844 with framebuffer/display ratio and actual rendered-font thresholds. Component disposal, replacement and parent destruction clear owned UI and view listeners. Final screenshots manually self-reviewed; final browser errors/warnings/request failures are empty. Earlier weak layout passes are superseded, with their defects and regression failures preserved.

- At the user’s request, removed failed Windows Creator/WeChat/Douyin tool directories, all Wine prefixes and 150 verified session-installed Wine/i386 packages; removed the Wine apt source/key and foreign architecture. Preserved required Linux GTK/Xvfb/fonts. Kubelet/containerd remain active; root disk ~45% (~22 GiB free), data disk ~47–48% (~20 GiB free).
- Installed and checksum-verified msojocs community Linux WeChat v2.02.2609240-2 (Electron 36.6.0) on the data disk. It runs without Wine, shows guest mode and the mini-game project manager. Worked around the packaged CLI’s extra --electron argument and non-XDG default path using a dedicated launcher and an app-specific config symlink; local CLI connected to the running tool. No production source change was needed.

### In progress

- At the user’s explicit request, development version 0.0.45 uploaded successfully and preview QR delivered. Original export exceeded WeChat’s 4 MB main-package limit; a separate post-export device-test copy loads unchanged cocos-js through wx.loadSubpackage before the original bootstrap. WeChat reports main 915134 bytes / engine 7472470 / total 8387604. Its simulator entry rendered after preview; device execution awaits user feedback. No review submission or formal release.

- Linux community WeChat imported the final native artifact after user scan login and Mini Game AppID configuration. Actual Entry.scene rendered in landscape; SDK 3.17.5 probes using actual adapter source passed for login code, two 32-byte entropy requests, vault save/load/clear with restoration, and lifecycle registration/removal. Full networking, lifecycle delivery, sharing and device qualification remain open. Goal 0045 stays in_progress, not archived/reset.

### Remaining

- Complete interactive Creator GUI preview and real mobile safe-area/orientation/background checks. Native CLI import, generated-config compile, three target exports and built Web scene runtime have passed; they do not prove interactive editor or hardware qualification.
- Continue WeChat domain whitelist/HTTP/WSS bearer, lifecycle event delivery, share and real-device qualification; current Linux simulator evidence does not cover them. Qualify Douyin SDK/device behavior and Mini Game entropy separately.
- Continue planned protocol integration in Goal 0046: separate durable envelope from current doudizhu-api-command-v1 request DTO, implement mapping, per-account/aggregate sequence/ID allocation and safe retries, private snapshot and terminal-evidence mapping. Do not submit the durable CommandEnvelope directly as a wire request or infer game state from notifications.

### Verification status

- Latest pre-upload report commit acf6041d8f32eb66cf3eae75b144b7d2da41dd92: CI 41229934812 and Fair Doudizhu 41229934872 both succeeded. Upload/preview success and package-file hashes are separate local evidence, not tests performed by CI. The device package changes only game.js, game.json and project.config.json and adds cocos-js/game.js; raw engine/resources remain unchanged. Reproducible packaging procedure and warnings are recorded in PLATFORM-VALIDATION.md.

- WeChat simulator evidence: wechat-sdk-probe.json SHA256 8ef91c295bb67dfa7db1442abdbc2eb8e72576d5ab9b692d9aad77e3edb80e4e; fresh clean console SHA256 c730c7f04c30b268e667be3612eba586ee55a439bb543c61b2bf90723704c145. Final screenshot manually reviewed. Default compiler concurrency hit the session memory-high limit; bundled-source-supported COMPILE_THREADS=1 / MULTI_TASK_WORKERS=1 resolved the loading bottleneck. Tool stopped; available memory ~4.3 GiB, no swap, root/data disks 45%/47%, kubelet/containerd active.

- Node 22.23.4: dependency installs, strict shared-package typecheck/build/tests, official @cocos/creator-types@3.8.8 script typecheck and renderer-free Shell build/tests passed. scripts/verify-client-foundation.sh completed successfully.
- Source and controlled Fake Adapter tests are separate from the real Chromium HTTP/WSS transport probe. Private runtime evidence and logs: /root/.config/awesome-zero-platform/goal0045-evidence (0700, files 0600), no password/token values.
- Accepted implementation source 7da85a9cf6b4cc3c0404dbecdd40b2e7d6cc2539: Cocos Creator Smoke 41208033225, CI 41208033131 and Fair Doudizhu 41208033122 all succeeded. Native job 124557119115 generated Web/WeChat/Douyin artifacts and actual Web screenshots; artifact 13044679501 is retained for seven days with a private local copy. Creator build logs contain documented non-blocking default-compression/plugin-migration warnings; browser warnings are empty.
- Final native browser report SHA256: c94f2815c3628237525cfe2b8bc689d6b9f384d5008734eac2d995fea6ca0250. Web transport probe remains separately tied to unchanged adapter source 0fc7ebb; it is not a playable-game or production TLS qualification.
- Phase review: docs/operations/goal0045-client-foundation-review.md. Local/source tests total 49 (26 core, 7 protocol, 7 Shell/assets, 9 unchanged HPKE), plus native builds and real browser lifecycle/layout checks. Official Windows/Wine attempts remain unqualified; subsequent community Linux WeChat scene and bounded SDK probes passed, separately from device qualification.

- Mandatory self-review completed: SDK timing, cancellation/deadlines, stale callbacks, duplicate full-snapshot confirmation, Go phase/TTL compatibility, version/identity boundaries, secret handling and cleanup. Reproduced regressions before correction; additional runtime self-review found and fixed false-ready polling, cropped/stretched canvas, small portrait text and retained UI; engine parent-destruction order and Web host initialization/page lifetime were checked against actual 3.8.8 source. Remaining interactive editor/SDK/device and long-term memory uncertainty is explicitly recorded in the phase report.

## Completion Report

Phase review and fixes delivered in docs/operations/goal0045-client-foundation-review.md and clients/fair-doudizhu-cocos/PLATFORM-VALIDATION.md. Verified implementation source 7da85a9 is pushed directly to main with green CI, Fair Doudizhu and native Cocos Creator Smoke. The final publication updates only evidence/docs/notices; its Actions result is checked after pushing and recorded in the closing handoff. Goal 0045 remains in_progress, not archived/reset: interactive editor preview, supported developer-tool SDK/device qualification, Douyin Mini Game entropy and later Goal 0046 identity/protocol integration remain open. The WeChat entry scene and bounded SDK probes now pass in the community Linux simulator; no playable game, complete Mini Game integration or real-device qualification is claimed. Existing Go services, schema, deployments and Admin Web are unchanged.
