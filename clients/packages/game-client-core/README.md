# Platform-neutral game client ports

The implementation defines small capability contracts for Web, WeChat Mini Game and Douyin Mini Game, without importing their runtime globals or Cocos. The runtime is supplied by a platform-specific composition root.

Run `npm install && npm test` with Node 22. This is a contract and security foundation, **not** a working Cocos/mini-game build. Never pass long-lived bearer tokens in URLs or synthesize secure entropy with Math.random.

See [cross-platform design](../../../docs/architecture/fair-doudizhu-cross-platform-client.md).

The root entry and internal connection driver contain no platform globals.
Adapters are explicit `./web`, `./minigame` and `./platform` ESM exports. The
connection manager cancels opening attempts via a portable callback (no dependency
on a global AbortController), and ignores callbacks from replaced connections.
Mini-game sockets wait for `SocketTask.onOpen`, never `connectSocket.success`.
Do not assume optional `off*` task methods exist; closing makes callbacks inert.

WebSocket endpoints reject credentials, queries and fragments; authentication uses
`['bearer', accessToken]`. WeChat entropy uses UserCryptoManager. Douyin entropy
still requires target qualification; absence fails closed. HTTP cancellation uses
an injected AbortSignal and native RequestTask.abort where provided. Session
storage is platform storage, not an encrypted vault. Share results indicate UI
invocation/copy (WeChat/Web) or the Douyin callback, never recipient delivery.
