# Platform-neutral game client ports

The implementation defines small capability contracts for Web, WeChat Mini Game and Douyin Mini Game, without importing their runtime globals or Cocos. The runtime is supplied by a platform-specific composition root.

Run `npm install && npm test` with Node 22. This is a contract and security foundation, **not** a working Cocos/mini-game build. Never pass long-lived bearer tokens in URLs or synthesize secure entropy with Math.random.

See [cross-platform design](../../../docs/architecture/fair-doudizhu-cross-platform-client.md).
