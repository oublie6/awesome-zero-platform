# Cross-platform Fair Doudizhu Client Architecture

Status: accepted design baseline (Goal 0045 onward). **Execution priority updated 2026-11-01: WeChat Mini Game first, other platform adapters retained but not release blockers.** Scope: client integration, not a new platform/server framework.

## Purpose and release targets

Build one playable, three-person friend-room Fair Doudizhu client, with **Cocos Creator 3.8.x** rendering and a shared TypeScript core. The **first fully functional and qualified release target is WeChat Mini Game**. Web/H5 remains a development/debugging target; mainland Douyin Mini Game is a future release target, with its adapter boundary retained. Other Cocos-supported targets are future adapters, not promises of binary compatibility; TikTok international products are a separate qualification. Actual platform SDK, Creator export support and account prerequisites must be checked against the installed version and each platform's current documentation before declaring success.

First playable release: login; private room create/join/invite; seats/ready; fair contribution commit/reveal; bidding and landlord selection; card play/pass; settlement/evidence; reconnection and recovery from lost messages. Do not add payments, gambling, rankings, matchmaking, AI opponents, or commercial inventory in the baseline.

## Non-negotiable architecture

- Server remains sole game authority for identity, room/seat membership, randomness verification, active hand, turn, legal play and settlement. The client can predict UI interactions only; success requires the accepted server result/snapshot.
- Backend is the existing modular Go/go-zero platform. Do not duplicate game rules in Cocos or leak Cocos/WeChat/Douyin names into Go foundation/platform modules.
- **Business and platform layers must not import Cocos, `wx`, `tt`, DOM, or platform SDKs.** Resolve platform capabilities through injected ports at composition time.
- Keep Cocos scenes thin: visual nodes, animations, input events and view binding only. Use presenter/view-model classes to connect versioned server snapshots to views.
- Protocol contracts are versioned and type-safe. Do not reinterpret actor, account ID, seat, card ownership, or legality from user-controlled payload.
- Reuse the existing `clients/packages/secure-envelope` implementation. No second HPKE implementation or insecure random fallback.
- Extract shared abstractions *when two real consumers need them*. Avoid giant adapters, speculative plugin registries and empty directory hierarchies.

## Target dependency diagram

```text
Web / WeChat / Douyin bootstrap (platform-specific)
   |                         |
   +--> capability adapters -+--> [shared game-client-core ports and services]
                                     |
                                     v
                              [doudizhu-client]
                         protocol / store / commands /
                       fairness / reconnect coordinator
                                     |
                                     v
                               Presenter / VM
                                     |
                                     v
                            Cocos scenes and UI
```

Arrows express imports/dependencies except that presenters observe state and call use cases. Business core depends on port **interfaces**, not adapter implementations or rendering packages.

Suggested *incremental*, not mandatory all-at-once, layout:

```text
clients/
  packages/
    secure-envelope/              # existing
    game-client-core/             # small reusable session, socket, lifecycle and ports
    doudizhu-client/              # game-only protocol, model, store, application, fairness
  fair-doudizhu-cocos/            # existing Cocos project / common assets
    assets/scripts/bootstrap/
    assets/scripts/presenter/
    assets/scripts/ui/
    assets/scripts/platform/web/
    assets/scripts/platform/wechat/
    assets/scripts/platform/douyin/
```

Never move existing code just to match illustrative paths. Record actual boundaries and migrate with tests.

## Capability contracts

Keep ports small. Extend each only when its first real call site exists:

- `PlatformAuth`: obtain ephemeral platform authorization material; **backend** verifies it and links platform identity to the stable platform AccountID. No client-provided openid/seat is authoritative.
- `HttpTransport`: request/response, cancellation, request timeout and canonical error mapping.
- `SocketTransport`: handshake, send, receive, close, connection status; isolate browser WebSocket versus mini-game SocketTask semantics, including headers/subprotocol limitations.
- `SessionVault`: platform-appropriate secrets/session lifetime and explicit cleanup; avoid logging tokens or storing durable secrets unencrypted in inappropriate storage.
- `SecureRandomSource`: cryptographically secure entropy; explicit availability check; fail closed. Validate WeChat's existing entropy path and independently qualify Douyin; never use `Math.random` for fairness secrets.
- `AppLifecycle`: foreground/background, connectivity changes and orderly cancellation.
- `SharePort`: invite payload, share target and cancellation. Browser copy/share and platform shares can behave differently; room code fallback always available.
- Optional audio/feedback/asset-delivery ports only when actual platform differences appear. Prefer shared Cocos APIs for equivalent rendering/animation/sound.

Example conceptual composition (not a mandate to implement exactly this interface):

```ts
type Target = 'web' | 'wechat' | 'douyin'
interface GamePlatform {
  readonly target: Target
  auth: PlatformAuth
  http: HttpTransport
  socket: SocketTransport
  vault: SessionVault
  random: SecureRandomSource
  lifecycle: AppLifecycle
  share: SharePort
}
```

Use target-specific composition roots; no scattered `typeof wx`/`typeof tt` checks in domain, game coordinators, or presenters. A missing capability is explicit, not silently ignored.

## Protocol and reliability

Use `docs/api/fair-doudizhu-protocol-v1.md`, `docs/architecture/doudizhu-realtime-v1.md`, and `docs/operations/realtime-websocket.md` as contracts.

- HTTP: login/account and non-realtime queries; WSS: `doudizhu.command`, `doudizhu.command.result`, `doudizhu.hand.changed`, `doudizhu.hand.sync`, `doudizhu.hand.snapshot`, `doudizhu.hand.evidence`, `doudizhu.error`.
- Client commands use `commandId`, increasing `clientSeq`, `expectedVersion`, timestamps and exact versioned names. Do not generate a different ID merely because an accepted request's response was lost; recover or idempotently retry first.
- Store contains **last server-confirmed** room/hand snapshot and monotonic version. A changed notification triggers a version-aware sync if needed, not an assumption about private cards. Never put private cards in generic topics.
- Reconnect states: connected → disconnected → reconnecting → synchronizing → connected. Disable irreversible actions until authoritative synchronization completes; backoff with jitter and stop on explicit logout. `notModified` preserves current snapshot; terminal evidence is a distinct read-only state.
- Current live hand is process-memory authoritative on the server; client reconnection does **not** imply recovery from an app-api crash or cross-node migration. Keep this explicit in UX, tests and release notes.
- WeChat/Douyin transport authentication must be validated against their actual Socket API. Browser uses existing `['bearer', accessToken]` subprotocol. Never put long-lived access/refresh tokens in query parameters. If a target cannot use the current handshake safely, design a short-lived single-purpose server-verified socket ticket as a separate reviewed backend change.

## Fairness

The existing client secure-envelope package handles HPKE; secure random is supplied by platform adapters. Contributions must stay consistent across commit/reveal, including a safe interrupted-flow policy and protection of sensitive values. Treat beacon and server adjudication as authoritative. Never include reveals, secrets, hands, auth tokens or private keys in logs, analytics or screenshots. User-facing progress can be simple while a detailed evidence view is available after settlement.

## UX and assets

Prefer landscape for the match table with own cards at bottom and two other seats above; build a readable lobby, room/ready flow, fairness-progress overlay, bidding/play/pass feedback, settlement/evidence and reconnect overlay. Distinguish a server-confirmed move from a locally selected preview. Scale to aspect ratios, cutouts and safe-area insets; accept touch gestures and alternative tap operation. Keep Cocos assets and common scenes shared across targets, with platform-specific build profiles, orientation, launch assets and subpackage/resource policies. Platform sharing must not be a prerequisite for entering a room.

## Testing and release gates

1. Pure TypeScript unit tests run without Cocos, `wx`, `tt` or browser globals: command envelopes, versions, snapshot privacy, idempotent retries, reconnection transitions, cancellation, fairness secret lifecycle.
2. Adapter contract tests cover Web plus WeChat/Douyin mocks for network, tokens, lifecycle, storage and secure random; failing entropy availability must fail closed.
3. Web playable smoke checks run with real backend on approved test accounts; three distinct sessions participate in one hand. Use simulator only as an explicitly labeled substitute for visual checks.
4. Creator Web, WeChat, Douyin build artifacts and editor/runtime importability are separately checked. `build success` != developer tool startup != real device / platform review approval. If SDK or credentials are unavailable, record blockers rather than claim compatibility.
5. End-to-end tests cover normal turn flow, invalid play, out-of-turn, duplicate commands, version conflict, lost response, foreground/background, network drop and reconnect, terminal evidence, and privacy of hidden cards.
6. Add measurable readiness criteria: zero unexpected console errors, no accidental secrets in logs/network query strings, legible controls and safe-area bounds, bounded retained listeners/resources, clean lifecycle shutdown.
7. Do not change existing backend security contracts or deploy production game traffic as part of client scaffolding. Changes to server authentication are separate reviewed goals.

## Milestones

- **Goal 0045**: establish actual import/buildable Cocos project, compact core/adapter seam and Web/WeChat/Douyin target smoke checks; validate secure entropy and socket compatibility; record any dev-tool/SDK blockers. No speculative game feature expansion.
- **Goal 0046**: login/session + HTTP/WSS + typed protocol integration and authoritative client state/reconnect coordinator.
- **Goal 0047**: friend-room creation/join/ready/invitation and multiplayer lobby.
- **Goal 0048**: fairness, bidding, card interactions, server-governed play, animations, one full three-player hand.
- **Goal 0049**: interrupted flows, reconnect/conflict/terminal recovery, settlement/evidence and quality/diagnostics.
- **Goal 0050**: per-platform packaging, real-device checks and release acceptance.

## Design decisions to revisit only with evidence

- Authentication ticket only if platform APIs cannot safely provide the existing handshake.
- A separate games service only if scaling/availability requires it; Cocos code cannot repair server process-owned live state.
- Additional capabilities (cloud save, push, payments, rankings, AI, remote asset CDN, observability events) only when justified by a concrete business scenario.

## 2026-11-01 delivery priority decision (supersedes multi-target simultaneous qualification)

- **P0 WeChat Mini Game playable vertical slice:** approved platform code-to-backend identity exchange and session; secure HTTPS/WSS; private friend room create/join/ready/share; fairness commit/reveal, bidding/play, settlement/evidence, disconnected/foreground recovery, and three real accounts completing one game. Implement only against actual Go wire DTOs. Resolve HPKE AAD issue/expiry context with backend before encrypted reveal; do not guess.
- **P0 WeChat release path:** Creator 3.8.8 real build, deterministic packaging/subpackage under applicable size limits, SDK simulator, user-approved upload/preview and explicit device qualification. Never upload, publish or modify backend production environment without appropriate authorization.
- **Preserved portability:** all game state/commands/fairness/protocol logic in pure TS packages; wx SDK exists solely in injected WeChat adapter/composition; browser retained for development and CI. Keep Douyin adapter source buildable where practical, but do not spend time qualifying tt SDK/devices or block WeChat milestones on Douyin entropy.
- **Milestone sequencing:** conclude Goal 0045 only under its recorded acceptance policy or explicitly amend criteria; Goal 0046 identity/protocol, 0047 room, 0048 full hand, 0049 recovery, 0050 WeChat release qualification. Do not declare any unimplemented feature complete.
- **Avoid duplicating business rules client-side.** Server owns state; never expose all players' cards or turn authority in Cocos. Don't substitute a mocked room or local game for integration acceptance.
