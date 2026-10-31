# Fair Doudizhu Cocos Client

Cocos Creator **3.8.8** foundation with an entry status screen, not a playable table.
Business state and transports are in small engine-independent packages; HPKE remains
in the existing `@awesome-zero-platform/secure-envelope` package.

## Reproducible source checks

Use Node 22.23.4 from the repository root:

```bash
bash scripts/verify-client-foundation.sh
```

This installs/builds packages in dependency order, checks shared contracts and
all Cocos scripts against official `@cocos/creator-types@3.8.8`, and tests the
actual Shell and scene/meta references. `typecheck:shell` compiles only the
renderer-free Shell/composition code. Neither check starts the Cocos engine.
The local `file:` dependencies resolve built `dist` ESM files; build the packages
before importing the project. Do not point Creator at unbuilt TypeScript packages.
Local asset imports use their actual `.ts` extension for Creator; the standalone
TypeScript build rewrites these to `.js` for Node ESM.

## Creator import and preview (requires installed editor)

1. Run the source checks above.
2. Import this directory as an existing project in Creator **3.8.8**.
3. Open `assets/scenes/Entry.scene`. It has a Canvas, orthographic camera and
   `GameBootstrap` attached through a stable script UUID; no manual scene creation
   or component attachment is required.
4. Preview the scene. Expect the Chinese title and initialization status, not a
   lobby/game table. Inspect console, landscape/portrait resizing and safe areas.
5. Set Entry as the start/included scene in the Build panel; build Web, WeChat
   Mini Game and mainland Douyin Mini Game separately. Configure the actual app IDs,
   landscape orientation and approved HTTPS/WSS domains in their target settings.
   No production app ID, network endpoint or generated editor/build cache is committed.
6. After editor import, `npm run typecheck:creator` uses Creator's generated
   `temp/tsconfig.cocos.json`. It requires the editor; it is not the standalone check.

Stable `.meta` files are committed; `library`, `temp`, `local`, `profiles` and
build outputs remain ignored. Entry scene structure follows the official 3.8.8
[2D template](https://github.com/cocos/cocos-engine/blob/v3.8.8/editor/assets/default_file_content/scene/scene-2d.scene).
Template attribution is retained in [third-party notices](THIRD-PARTY-NOTICES.md).
Actual editor/build/runtime results are tracked in
[platform validation](PLATFORM-VALIDATION.md) and the
[phase review](../../docs/operations/goal0045-client-foundation-review.md).

## Native Windows build and runtime check

The `Cocos Creator Smoke` workflow installs the official pinned 3.8.8 Windows
archive, imports/builds this project, checks the generated TypeScript configuration,
starts the real Web scene in Chromium, and exports both mini-game targets in sequence.
Logs, per-target build reports, artifacts and browser screenshots are retained as
`cocos-creator-web-evidence`. Mini-game exports use Creator's default test app IDs;
developer-tool qualification needs project-owned IDs and domain configuration.
Exports do not prove SDK/device compatibility.

To use an installed editor from repository root in PowerShell:

```powershell
./scripts/verify-cocos-runtime.ps1 -CreatorExe 'C:/tools/Creator/CocosCreator.exe' -OutputRoot 'C:/temp/cocos-smoke'
# Optional -Platform: wechatgame or bytedance-mini-game
```

The check requires Creator's documented success exit code 36 and a target entry
artifact. It does not provide app credentials or bypass editor/platform login.

## Entropy and platform differences

WeChat uses `wx.getUserCryptoManager().getRandomValues` (minimum documented base
library 2.17.3), explicitly injected at composition. `WechatRandomSource` adapts
that existing entropy port to the unchanged HPKE package. It does not access a
hidden global `wx` or fall back to `Math.random`.

Douyin Mini Game entropy support has not been established by a mini-game SDK
runtime; the similarly named **mini-app** API is not proof. Missing/incompatible
entropy fails closed. Platform login codes are ephemeral authorization material,
not game identities; server login/session integration remains Goal 0046.
