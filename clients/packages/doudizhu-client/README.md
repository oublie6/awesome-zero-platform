# Fair Doudizhu pure TypeScript client state

Contains a minimal authoritative snapshot holder, connection state and a versioned command envelope guard. It has no dependency on Cocos, browsers, WeChat, Douyin, or business-server internals.

This is **not** a complete protocol adapter or game rules implementation. More precise protocol DTOs and command/event handlers must be added against the server's documented wire structures during Goal 0046.

Run with Node 22: `npm install && npm test`. See [client design](../../../docs/architecture/fair-doudizhu-cross-platform-client.md).
