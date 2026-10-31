# Goal 0045: Cross-platform Fair Doudizhu client foundation

## Status

- State: in_progress
- Started: 2026-10-31
- Completed:
- Blockers: Full-goal runtime gates still require Creator 3.8.8, WeChat/Douyin developer tools and target devices; none are installed in this Linux environment.

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

### In progress

- Codex phase handoff: complete report, final self-review, commit/push and verify Actions. This is a phase check, not completion or archival of Goal 0045.

### Remaining

- Run Creator 3.8.8 actual import, generated-config compile, preview and target builds. Static official declarations and scene-reference tests do not prove engine execution, visual layout or absence of engine warnings.
- Qualify WeChat/Douyin in developer tools and devices: app IDs/domain whitelist, bearer subprotocol selection, entropy, lifecycle/storage/share and resource cleanup. Do not infer Mini Game entropy from Douyin Mini App documentation.
- Continue planned protocol integration in Goal 0046: separate durable envelope from current doudizhu-api-command-v1 request DTO, implement mapping, per-account/aggregate sequence/ID allocation and safe retries, private snapshot and terminal-evidence mapping. Do not submit the durable CommandEnvelope directly as a wire request or infer game state from notifications.

### Verification status

- Node 22.23.4: dependency installs, strict shared-package typecheck/build/tests, official @cocos/creator-types@3.8.8 script typecheck and renderer-free Shell build/tests passed. scripts/verify-client-foundation.sh completed successfully.
- Source and controlled Fake Adapter tests are separate from the real Chromium HTTP/WSS transport probe. Private runtime evidence and logs: /root/.config/awesome-zero-platform/goal0045-evidence (0700, files 0600), no password/token values.
- npm run typecheck:creator cannot run successfully: temp/tsconfig.cocos.json is absent because Creator is not installed. No Creator import/preview/build, mini-game developer-tool or device result is marked passed.
- Related Actions for the forthcoming source checkpoint and final report commit will be checked after pushing; no CI success claim is made yet.

## Completion Report

Phase review and fixes only. Goal 0045 remains in_progress and must not be archived or reset. Full editor/target runtime acceptance remains open; the phase report will record exact source commit, CI evidence, capability differences and the next ChatGPT tasks.
