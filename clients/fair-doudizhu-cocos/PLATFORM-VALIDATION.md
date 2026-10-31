# Goal 0045 platform validation status

This is a phase checkpoint, not full Goal 0045 acceptance. Do not archive the goal.
See the [phase review](../../docs/operations/goal0045-client-foundation-review.md)
for findings, source commits, evidence, SDK differences and handoff tasks.

| Layer | Result |
| --- | --- |
| game-client-core strict typecheck/build/tests | Passed, 26 tests; real modules with controlled SDKs |
| doudizhu-client strict typecheck/build/tests | Passed, 7 tests, including actual Go phase constants |
| Existing secure-envelope | Unchanged; existing 9 tests passed |
| All Cocos scripts vs official 3.8.8 declarations | Passed; source check, not engine execution |
| Shell ESM + scene/meta references | Passed, 7 tests; Entry.scene now committed |
| Chromium Web adapters + real Go HTTPS/WSS | Passed via temporary local self-signed TLS bridge: login, hello/ping/pong, invalid token, entropy and session storage; no game commands |
| Creator 3.8.8 native CLI import/build | Passed on Windows: Web, WeChat and Douyin, exit 36; both mini-game outputs explicitly landscape |
| Creator generated-config strict TypeScript check | Passed; actual generated configuration, Bundler package resolution |
| Real built Web scene in Chromium | Passed at 1280×720, 844×390 and 390×844: active bootstrap/UI, viewport/framebuffer ratio, rendered font size, component disposal/restart and parent disposal; screenshots manually reviewed |
| Interactive Creator GUI preview | Not qualified: Linux Wine startup failed/stopped; native CI used CLI import/build and real built Web runtime |
| WeChat/Douyin developer tools and device tests | Official Windows tools installed; Wine startup attempted, no usable simulator result; supported OS/app IDs/devices still required |
| Full game client / three-player hand | Not implemented by this foundation; later goals |

## Reproducible checks and editor handoff

1. Run `bash scripts/verify-client-foundation.sh` at repository root with Node 22.23.4.
2. Import this directory in Creator **3.8.8**, open `assets/scenes/Entry.scene`,
   inspect Canvas/camera/bootstrap binding, preview and capture logs/screenshots.
   After import, `npm run typecheck:creator` uses the real generated configuration.
3. Build Web, WeChat Mini Game and mainland Douyin Mini Game separately. Validate
   landscape/safe areas, repeated start/dispose and actual package resolution.
4. Configure approved HTTPS/WSS domains and platform app IDs, then verify bearer
   subprotocol negotiation in each developer tool and on devices. No long-lived
   JWT-in-URL fallback; a server-verified short-lived ticket needs a separate review
   if an actual target cannot support the current secure handshake.
5. WeChat entropy is on `getUserCryptoManager().getRandomValues`. Douyin Mini Game
   entropy is still unqualified; Mini App documentation is not sufficient. Missing
   capability must fail closed. Current injected interfaces are not proof of SDK
   availability or device compatibility.
6. Use explicitly approved test identities. Do not modify production accounts or
   issue arbitrary game commands. Temporary probe sessions must be logged out.

Goal 0046 still needs real identity/session integration, authorized HTTP, explicit
Go wire DTO/private snapshot/evidence mapping, stable command IDs/sequences and
safe retries. The durable CommandEnvelope is not the current WSS request shape.
Resolve the reveal AAD time context before a real encrypted reveal flow.

Accepted source: `7da85a9cf6b4cc3c0404dbecdd40b2e7d6cc2539`.
[Cocos Creator Smoke 41208033225](https://github.com/oublie6/awesome-zero-platform/actions/runs/41208033225),
[CI 41208033131](https://github.com/oublie6/awesome-zero-platform/actions/runs/41208033131)
and [Fair Doudizhu 41208033122](https://github.com/oublie6/awesome-zero-platform/actions/runs/41208033122)
all succeeded. Evidence artifact `13044679501` retains logs, three target archives,
reports and screenshots for seven days; a private local evidence copy is retained.
Earlier runs with weaker viewport checks are superseded and do not qualify layout.
Creator build logs retain non-blocking default-compression/plugin-migration warnings;
the final browser report has no errors, warnings or failed requests.

GameBootstrap displays entry status only. No playable lobby/card table is claimed.
