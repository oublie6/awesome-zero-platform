# Goal 0045: Cross-platform Fair Doudizhu client foundation

## Status

- State: ready
- Started:
- Completed:
- Blockers:

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

- Cross-platform design accepted and documented.
- ChatGPT committed first-pass pure TypeScript packages: `clients/packages/game-client-core` (capability ports, secure entropy and socket URL guards) and `clients/packages/doudizhu-client` (server-versioned hand store, reconnect-state guard and command envelope validation). Added standalone Node test suites and README files for both. These are foundations, not full Cocos runtime or platform adapters.

### In progress

- ChatGPT implementation phase: added browser HTTP/WebSocket/secure-random/session/lifecycle/share adapters; injected mini-game HTTP/WebSocket/secure-random adapters; added portable connection lifecycle and Doudizhu event router/command factory. Platform SDK support and actual Cocos import have not yet been verified.

### Remaining

- Complete platform bootstrap/composition, Cocos scene integration, and actual game-shell flows; reconcile API and SDK signatures against real server and platform implementations.
- After development, Codex independent test execution (`npm install`, `npm test`, typecheck for both packages), dependency audit, fixes, Creator build, simulator and real runtime validation. Final archive only after required validation.

### Verification status

- Source committed; runtime/CI verification of new packages not yet established. No claim of Web/WeChat/Douyin build compatibility yet.

## Completion Report

Not started.
