# Goal 0045 platform validation status

**New development baseline (2026-11-05):** Linux-native official
Cocos CLI4 / engine package 4.0.0-beta.0. The tests and Creator 3.8.8
artifacts described below are retained as historical evidence only; no
routine Windows/Wine build is maintained. See
[the pinned Linux toolchain](../../docs/operations/cocos-linux-wechat.md).
Native simulator and physical-device acceptance remain separate.


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
| Linux community WeChat developer tool | Passed on community v2.02.2609240-2 / SDK 3.17.5 with a user-owned Mini Game AppID and scan login: actual 3.8.8 Entry.scene rendered in landscape. Actual adapter + wx SDK probes passed for login code, entropy, storage and lifecycle registration/removal. HTTP/WSS, lifecycle event delivery, share and devices remain unqualified |
| Douyin developer tool and device tests | Windows/Wine attempt unqualified; unused Windows tools/Wine removed. Platform SDK/device validation remains open |
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

Accepted source: `5ef378f62f5d81992a569af99c4141ee7432e7c2`.
[Cocos Creator Smoke 41264441151](https://github.com/oublie6/awesome-zero-platform/actions/runs/41264441151),
[CI 41264441148](https://github.com/oublie6/awesome-zero-platform/actions/runs/41264441148)
and [Fair Doudizhu 41264441201](https://github.com/oublie6/awesome-zero-platform/actions/runs/41264441201)
all succeeded. Evidence artifact `13066896631` retains logs, three target archives,
reports and screenshots for seven days; a private local evidence copy is retained.
Earlier runs with weaker viewport checks are superseded and do not qualify layout.
Creator build logs retain non-blocking default-compression/plugin-migration warnings;
the final browser report has no errors, warnings or failed requests.

GameBootstrap displays entry status only. No playable lobby/card table is claimed.

## Linux WeChat simulator checkpoint

The final Windows-built WeChat artifact from source `7da85a9` was imported using
`/data/awesome-zero-platform-tools/goal0045/wechat-cli.sh open --project
/data/awesome-zero-platform-tools/goal0045/mini-game-projects/wechatgame --lang zh`
(run as one shell command from the installed tool's `bin` directory).
Only the local artifact's AppID was overridden. A regular Mini Program AppID led
to Mini Program mode and missing `app.json`; the user-owned Mini Game AppID plus
scan login selected Mini Game mode and successfully loaded `Entry.scene`.

Launch the installed Linux tool with `COMPILE_THREADS=1 MULTI_TASK_WORKERS=1`.
These environment settings are supported by its bundled compiler source. Default
parallel compilation hit the bounded session's memory-high limit; the single
worker run completed with no memory-high/OOM events at the measured checkpoint.
Do not interpret the earlier black loading screen as an application failure.

A private diagnostic bundle transpiled the actual `platform.ts`, `minigame.ts`,
`socket.ts`, `network-timeout.ts` and `index.ts` sources with TypeScript into a
small CommonJS loader, injected into the real simulator's context. It called
`createMiniPlatform('wechat', wx, 'https://example.invalid')`: two 32-byte entropy
requests returned distinct results, with the first checked as nonzero, vault save/load/clear passed with
previous storage restored, lifecycle hooks registered/unregistered, and login
returned a nonempty code. No HTTP call, WSS connection, backend token exchange or
share was made. Returned login codes and random bytes were not recorded.

Private evidence: `wechat-sdk-probe.json`, `wechat-clean-console.json` and
`wechat-clean-scene.png` in the phase evidence directory. After an earlier DOM
inspection probe failed against the mini-game document shim, a fresh compile
produced 13 informational console entries, including successful scene loading,
with no captured warning/error entries (a console DOM capture is not a complete event stream). The final screenshot was manually
reviewed. This is community Linux simulator evidence, not official-host or
real-device qualification. The tool was stopped after testing; no upload or
publication was performed.

## User-authorized device upload

At the user's explicit request, development version **0.0.45** was uploaded and a
preview QR was generated with the logged-in tool. No review submission or formal
release was made. Original debug output was rejected: source size 8171 KB exceeded
the 4 MB main-package limit. A separate local `wechatgame-device` copy now declares
`{"name":"cocos-engine","root":"cocos-js/"}` in `game.json`'s `subpackages`.
Its root `game.js` wraps the complete original bootstrap in
`wx.loadSubpackage({name: 'cocos-engine', success: function () { /* original */ },
fail: function (error) { console.error('Cocos engine subpackage failed to load', error) }})`.
An inert `cocos-js/game.js` is the subpackage entry; existing SystemJS module
registration still loads the original engine modules after download succeeds.
`projectname` is changed to `fair-doudizhu-device` in this copy only.

To reproduce the packaging from an untouched export (set the owned Mini Game
AppID in the copied project config before import):

```python
# Run in a Python script with SOURCE and TARGET set to separate absolute paths.
from pathlib import Path
import json, shutil
src, dst = Path(SOURCE), Path(TARGET)
assert not dst.exists(), 'Preserve any existing target; select a new directory'
config = json.loads((src / 'game.json').read_text())
assert not config.get('subpackages'), 'This recipe is for the unpartitioned export'
shutil.copytree(src, dst)
config['subpackages'] = [{'name': 'cocos-engine', 'root': 'cocos-js/'}]
(dst / 'game.json').write_text(json.dumps(config, indent=2) + '\n')
original = (src / 'game.js').read_text()
(dst / 'game.js').write_text(
    'wx.loadSubpackage({name:"cocos-engine",success:function(){\n' + original +
    '\n},fail:function(error){console.error("Cocos engine subpackage failed to load",error)}});\n')
(dst / 'cocos-js/game.js').write_text('// Original engine modules load after subpackage success.\n')
```

The uploaded package retains the **unminified original engine**, not the separate
Terser experiment. Hash comparison against the source artifact found only three
changed files (`game.js`, `game.json`, `project.config.json`) and one new file
(`cocos-js/game.js`); no removed files or engine/resource changes. WeChat's successful
upload and preview reports agree: main **915134 bytes**, engine subpackage **7472470
bytes**, total **8387604 bytes**. The simulator screenshot after preview confirms
the landscape entry still renders. The compiler warns it skips processing engine
files above 2000 KB; this is not a warning-free production build qualification.
This is an explicit post-export device-test package, not a new native Creator
release-build result. Integrate appropriate Creator release/feature trimming or
subpackage settings before production distribution.

Run the installed CLI's `upload` with `--project <device-directory> --version
0.0.45 --desc <description> --info-output <private-json>`, and `preview` with
`--project <device-directory> --qr-format image --qr-output <private-png>
--info-output <private-json>` (each as one command). Upload requires user
authorization; do not upload again merely to repeat verification. Private evidence
includes `wechat-device-package-manifest.json`, upload/preview info, QR and
`wechat-device-scene-after-preview.png`. The user subsequently reported **the home page opens in WeChat, but nothing is
clickable**. GameBootstrap creates only title/status Labels and has no Button or
click handlers; interactive login/room UI is not implemented. Record this as
user-reported basic device launch, not verified touch input or a playable game.
The user later confirmed landscape, no anomalies and complete text. Model/OS,
safe areas and lifecycle remain independently unverified. An
uploaded package or QR alone is not proof of successful device execution.

## Closeout runtime choice

Daily cloud execution uses the existing community Linux developer-tool simulator.
A user phone/USB connection is not required for routine simulator checks. The
Android emulator trial is stopped and its SDK/AVD/APK removed after preserving
private evidence. Android boot with official `-lowram` and 1 GB guest RAM passed,
but WeChat installation/login did not qualify; that trial is not device evidence.

The old entry text incorrectly asked users to select a platform and log in with
no controls to do so. Source 5ef378f replaces it with an accurate development
status. The new `verify-wechat-simulator.cjs` check rejects the old artifact's text.
Actual touch delivery was separately verified against source 7da85a9's native
artifact by selecting the game's real execution context and injecting a touch
through the debugger Input protocol. Plain mouse attempts and outer-context
probes are not qualification evidence. The new native artifact from run
41264441151 now passes the repository smoke script: title/status/devtools checked,
one native touch received by Cocos and temporary listener removed. Only the local
project AppID was overridden; engine/application files are the native export.
The operator must open the debugger and wait for compilation; checks before those
prerequisites failed explicitly. Screenshots were self-reviewed. Private report
`wechat-5ef378f/wechat-runtime.json` SHA256:
`4008d3f97dd90e4840752da33da64c79439311914a7f8853d991575a110f23bf`.
The tool is stopped after verification. This new artifact was not uploaded again;
the existing user development package remains 0.0.45.

The smoke script checks rendering/text/touch delivery and removes its listener.
It does not implement or certify login/room UI. Login and full Go wire/session
integration belong to Goal 0046; room/lobby UI belongs to Goal 0047.
