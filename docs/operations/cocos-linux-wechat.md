# Linux native Cocos Mini Game builder

## Unified Linux development build baseline (2026-11-05)

The user authorized one Linux-native build line. The Cocos project now
declares engine package **4.0.0-beta.0**, matching the pinned official CLI.
See `scripts/cocos-linux-wechat/toolchain.json` for the exact official CLI
revision and engine checkout. Creator 3.8.8 is historical evidence only.
Windows Creator, Wine and foreign-OS virtual machines are not part of the
current routine build or CI pipeline.

`node scripts/cocos-linux-wechat/verify-toolchain.cjs` checks the pinned
source declarations; when `COCOS_CLI_ROOT` and `COCOS_ENGINE_ROOT` are set it
also validates actual installed revisions. `node
scripts/cocos-linux-wechat/build.cjs /private/native-output` performs an
isolated **real WECHAT export** with the official CLI and the project-owned
extension. The exporter requires an already-built pinned Linux CLI/engine,
`xvfb-run`, Node 22.23.4, actual engine caches, and built portable
packages. It exits nonzero when any are unavailable; no browser or fake
mini-game output is substituted.

The Linux GitHub workflow runs independent static/source validation on
hosted Linux runners and exposes an explicit native export job for an
approved `self-hosted,linux,x64,cocos-cli4` runner with the pinned toolchain
installed. **A hosted static green check is not an actual native export.**
The previous Creator3.8 evidence, below, remains intact for historical
reference. Adoption of the beta exporter for release additionally requires
a successful exact-source native build, real SDK complete game, genuine
device/platform qualification, and independent Goal review.


User authorization (2026-11-04): implement the missing Linux native Mini Game
export path. Do not use Wine or Windows/macOS virtual machines. This is an
experimental client toolchain extension, not independent product acceptance.

## Design and boundaries

- Owner: client build integration, under `scripts/cocos-linux-wechat/`. The
  backend, protocol, authoritative rules, credentials and database are unchanged.
- Contract: extend pinned official cocos-cli `f506fa3de403b48b4833334fa48dddd27bb26364`
  with a `wechatgame` platform. Compile for WECHAT, use the official engine's
  existing runtime adapters, emit `game.js`, `game.json` and project configuration.
  Do not relabel a browser build as a Mini Game build.
- Authority/dependencies: source project remains Creator3.8.8. The separately
  installed CLI uses engine4.0.0-beta.0. Cross-version compatibility must be tested;
  no source version upgrade or accepted-source claim is implied.
- Failure: fail on unsupported CLI revision, absent adapters, unsupported bundle
  modes or generation errors. Initial scope is a local, unsplit package. Remote
  bundles and subpackages require explicit implementation and verification.
- Resource bounds: one build at a time, bounded external process runtime and
  memory, no swap, stop on OOM. Preserve Linux developer-tool profiles.
- Security: no AppSecret or upload key in output; AppID is operator-supplied.
  No upload, preview upload, publication, production changes or relaxed TLS.
- Migration/rollback: install a uniquely owned platform directory in the external
  CLI; remove that directory to roll back. Build an isolated source copy, retain
  original Creator3.8.8 configuration and historical acceptance evidence.
- Evidence: validate platform registration and WECHAT compilation; inspect real
  output, run in the Linux WeChat developer tool, verify rendered scene/touch,
  and rerun source checks. Build success is separate from SDK/device/product
  qualification. Implementation review is self-review, not independent acceptance.

## State

Linux native export and the scoped SDK smoke check passed on 2026-11-04.
This is implementation self-verification, not independent product acceptance.
Previous Round7 product FAIL remains authoritative; Goals0045/0046 remain
`in_progress`. Creator3.8 remains the published baseline; adopting CLI4 as a
release baseline requires explicit architecture/Goal amendment and integrated
qualification. This additional experimental builder does not silently replace it.

## Reproduce on Linux

Prerequisites: pinned official CLI compiled with its real engine caches and
dependencies, Node22.23.4, Xvfb/Mesa; no Wine or virtual machine. The current
installation is `/data/awesome-zero-platform-tools/round7-cocos-cli`.

```sh
export COCOS_CLI_ROOT=/data/awesome-zero-platform-tools/round7-cocos-cli
node scripts/cocos-linux-wechat/install.cjs "$COCOS_CLI_ROOT"
node --test scripts/cocos-linux-wechat/test.cjs
```

Prepare and build client packages with `scripts/verify-client-foundation.sh`,
then use a disposable copy of the source tree with those packages available.
Do not import the primary source checkout into CLI4. In the copy, merge
`scripts/cocos-linux-wechat/engine-profile.json` into `settings/cocos.config.json`
or use it directly if there are no existing settings. This selects the same
2D/UI feature set used by the tested project; it is not full 3D qualification.

Write an external build-options JSON with `platform: "wechatgame"`, `debug`,
`outputName`, `buildPath`, and the actual Entry.scene.meta `startScene` UUID.
Optional `packages.wechatgame` contains `appid` and `orientation` (landscape).
AppID may be filled in the private exported project before importing it into
the developer tool. Never place AppSecret or upload keys in the client.

```sh
export PATH="$COCOS_CLI_ROOT/node_modules/.bin:$PATH"
export NODE_OPTIONS=--max-old-space-size=2048
xvfb-run -a node "$COCOS_CLI_ROOT/dist/cli.js" --no-interactive build \
  --project "$PROJECT_COPY" --platform wechatgame --build-config "$BUILD_OPTIONS"
```

Run under a bounded process supervisor: tested limits are 3GiB/no swap,
one CPU and ten minutes. Use an existing Linux WeChat developer tool profile,
open the exported project and debugger, then execute:

```sh
node scripts/cocos-linux-wechat/smoke.cjs /private/evidence-directory
```

The smoke check requires Playwright (the existing Admin Web dependency or
`COCOS_SMOKE_PLAYWRIGHT_MODULE`), with developer-tool CDP on loopback19227 by
default. It sends actual SDK touch events to rendered buttons and does not log
in, upload or publish. SDK simulation is distinct from physical-device proof.

## Compatibility constraints

The build target preserves native exponentiation using a Chrome90 syntax
target because this application already requires native BigInt for HPKE/BLS.
A parser-based generated-code audit rejects direct/known BigInt Math.pow
lowering, including aliases; it is a narrow regression gate, not a complete
JavaScript type proof. Actual old-phone and iOS compatibility remains unverified.
Generated temporary TypeScript configuration uses real engine declarations and
retains the project's strict settings; there are no fake declarations or skipped
source typechecks. No production source or declared Creator version is migrated.

Upstream CLI attempts to import an unused built-in 3D FBX with a non-Linux
converter and reports an import error. This extension does not supply a foreign
binary or claim FBX/3D support. Remote bundles, subpackages, engine separation,
MD5-cache modes, Douyin, uploads and release qualification remain outside the
initial target. Release output strips only trailing, parsed inline-source-map
comments; source-map text inside string literals is preserved.

## Executed evidence and self-review

- Main started clean and synchronized at `cc28c70db7765ef413d1c4c8854002ff01373afd`.
  Client input binding: 87 tracked files byte-identical in the isolated copy;
  one package.json is JSON-semantically identical after CLI formatting. The
  source Creator version remains3.8.8; actual engine is4.0.0-beta.0.
- Actual Linux WECHAT compilation: initial debug export exit0,391.850s; native
  exponent release export exit0,404.931s; final warm export with generated-code
  audit and map stripping exit0,157.471s. CPU1, MemoryMax3GiB, no swap. A prior
  compressed attempt was intentionally stopped after identifying incompatible
  BigInt lowering; stopping a systemd wrapper is not recorded as build PASS.
- Final release package3074144 bytes (2.932MiB), before any upload. SHA256 of
  the sorted per-file manifest JSON is
  `510688be2290d3630fb02edcbbd3a9923f5377930bc0a39c7a51444eef1db652`.
  AppID and SDK3.17.5 were configured only in the private output. No AppSecret,
  upload key, mock shader compiler, foreign-system VM or Wine was used.
- Actual CLI strict typecheck passed without the previous private tsc shim.
  Platform tests cover installation refusal, startup generation, strict config,
  remote/orientation rejection, BigInt alias/bracket lowering and exact map
  comment stripping. Existing client foundation30+37+35=102 tests passed.
- Linux community developer toolv2.02.2609240-2/SDK3.17.5: actual Cocos scene,
  native BigInt semantics and32-byte UserCryptoManager request passed; actual
  native touches opened Rules, returned Home and opened Rooms. No component
  handlers were invoked directly. Screenshot manually reviewed. Initial smoke
  caught the debugger before SDK existed; that failure is retained and bounded
  readiness waiting corrected before the successful rerun.
- Scope limits: no new login/network/full-hand/device qualification is claimed.
  The Rooms screenshot shows a default `label` on the input and small text;
  input/layout and wider3.8→4.0 UI compatibility need follow-up. The existing
  Round7 timer/replay/product failures are not repaired or closed by this work.
- Second-pass self-review checked platform/engine selection, CommonJS loading,
  output paths, strict type options, native exponent semantics, parser false
  positives, map-comment boundaries, ownership/pin refusal, resources and secret
  boundaries. It found the compressed-code gate and readiness issues above,
  both corrected and rerun, plus an exact-parent output-path guard edge fixed
  with a negative regression. This is self-review, not independent acceptance.
- Owned build/GUI processes were stopped after verification. Both Kubernetes
  nodes remained Ready; no swap; main available memory returned to about4.5GiB.
  Private raw logs, provenance, artifact hashes and screenshots are under
  `/root/.config/awesome-zero-platform/linux-native-minigame` and are not committed.

## Goal0055 G55-N01/N02 concentrated remediation (2026-11-08)

The original candidate `a4551b77d375` failed on WeChat preview despite a
successful Linux Cocos4 export:

- `G55-N01`: raw exported `assets/main/index.js` retained `??`, which
  the real WeChat preview syntax gate rejected.
- `G55-N02`: enabling WeChat developer-tool `es6=true` transformed native
  `2n ** 256n` into invalid `Math.pow(2n,256n)`, so the simulator and the
  user's real device could not enter the game scene.

The owned exporter now targets **Chrome 79**: esbuild lowers `??` and `?.`
for that target while retaining native BigInt exponentiation. The generated
WeChat project config keeps approved-domain enforcement (`urlCheck=true`),
**disables WeChat's second ES6 transform** (`es6=false`), and enables
**WeChat minification** (`minified=true`) for the preview size gate.
This choice is based on the Goal0055 private A/B result: minified,
ES6-disabled local output preserved the actual scene and native HPKE. It is
not yet a claim that the new export has passed a real preview/upload/device
test.

A parser-based audit of every exported JS file now rejects nullish coalescing,
optional chaining, and BigInt passed to `Math.pow`. This fails closed if a
CLI-generated or copied adapter file escapes the chosen syntax target. The
standalone audit can also be invoked on the **actual WeChat-processed**
directory after the tool's build:

```sh
# First build from this exact source SHA with the pinned Linux-native CLI:
node scripts/cocos-linux-wechat/build.cjs /private/verified-export

# Then (after the authorized WeChat developer-tool compilation), audit
# the *processed* src/chunks directory or its parent, not raw output alone:
node scripts/cocos-linux-wechat/audit.cjs /private/actual-wechat-processed-code
```

This is source/build-level remediation and regression checking, not automatic
proof of running the Linux developer tool. Next verification must record an
actual matching-SHA native export, actual WeChat preview syntax/size result,
the compiled BigInt/HPKE execution, actual simulator startup, and a freshly
authorized device preview. WeChat HTTPS/WSS with approved public TLS remains
a distinct external integration blocker; do not weaken `urlCheck` or bundle
AppSecret to bypass it.
