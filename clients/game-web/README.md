# Fair Doudizhu Vue 3 Website

This is the **Web application shell**, not another authoritative Doudizhu
implementation. It uses the existing Go `POST /auth/login`,
`GET /auth/session`, `POST /auth/logout` identity APIs and the same
AccountID as WeChat; registration remains administrator-controlled.

## Boundaries

- Vue 3 owns login, navigation, presentation and the game iframe.
- Cocos4 owns the interactive cards/room/hand scene under `/game/index.html`.
- Only the Go backend authorizes rooms, deals, moves, replay and recovery.
- A Vue login stores the *server-issued access token only* in per-tab
  `sessionStorage['fair-doudizhu-session']`. Password, random contribution,
  card snapshots, room and hand state never enter browser storage.
- The game iframe must be **same-origin** and hosted over HTTPS. Its
  `WebSessionVault` reads that same per-tab storage; the parent sends only
  the `azp-doudizhu-frame-v1` `SESSION_AVAILABLE` control event.
  The Cocos scene independently validates `GET /auth/session` before
  connecting its WSS transport. All incoming messages check both exact
  sender `event.source` and exact `event.origin`.
- Sign-out drops the iframe immediately, clears local token, and requests
  Go server-side revocation. No bearer tokens are posted through the iframe
  bridge, placed into URLs or included in generated assets.

## Build

```sh
npm ci --prefix clients/game-web
npm run test --prefix clients/game-web
npm run build --prefix clients/game-web

# Use the already pinned Linux Cocos CLI4 toolchain, per
# scripts/cocos-linux-web/build.cjs. This is a REAL export, not a shim.
node scripts/cocos-linux-web/build.cjs /private/verified-web
# Locate the actual generated Cocos Web directory containing index.html.
export WEB_API_BASE=https://play.example.org
export WEB_SOCKET_ENDPOINT=wss://play.example.org/ws
node scripts/game-web/stage.cjs \
  /private/verified-web/fair-doudizhu-web clients/game-web/dist
```

The combined output is `clients/game-web/dist`: Vue assets at `/`,
Cocos assets at `/game/`, and identical public `game-config.js` in
both locations. It also emits a SHA256 manifest. Never ship secrets as
public config; `WEB_API_BASE` must be a legal HTTPS origin and WSS
endpoint must be configured through approved routing/TLS.

The Vite dev server intentionally returns 404 for an unstaged Cocos game:
it must never silently substitute its own SPA index for a fake game.
For local graphical testing, stage a real Cocos4 Web export in
`clients/game-web/public/game/`. The Vue shell can be run without
game assets for independent login UI development; browser identity
authentication requires an approved same-origin HTTPS deployment.

## Remaining qualification

Hosted Web CI compiles/tests the **Vue shell only**. It does not prove
a generated Cocos4 Web build exists. The separately gated native Linux
CLI build and Chromium browser smoke must run on real Cocos output.
Then verify authorized three-person live play, recovery and websocket
identity with real Go/MySQL/WSS. HTTP localhost/self-signed TLS and fake
identities cannot substitute for production-style browser authorization.
WeChat device G55-N02 is a separate unresolved gate, not magically fixed
by Web integration. Goal0045/0046 and E03 stay in_progress; C09/C10 DEFERRED.
