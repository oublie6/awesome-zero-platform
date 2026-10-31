# Fair Doudizhu pure TypeScript client state

Contains a minimal authoritative snapshot holder, connection state and a versioned command envelope guard. It has no dependency on Cocos, browsers, WeChat, Douyin, or business-server internals.

This is **not** a complete protocol adapter or game rules implementation. More precise protocol DTOs and command/event handlers must be added against the server's documented wire structures during Goal 0046.

Run with Node 22: `npm install && npm test`. See [client design](../../../docs/architecture/fair-doudizhu-cross-platform-client.md).

`CommandEnvelope` models the durable application contract in
`fair-doudizhu-protocol-v1.md`. It is **not** the current HTTP/WSS dispatcher DTO:
that uses `doudizhu-api-command-v1`, `requestId` and `type`. No sender currently
casts or submits the durable envelope as a wire request. Goal 0046 must implement
explicit mapping and per-account/aggregate sequence allocation; callers currently
supply command IDs/sequences. `createCommand` freezes a copy so retries can reuse
identical metadata/payload. Its validity window is capped at the current default
two minutes (application default, distinct from dispatcher-generated timestamps); game payload legality is always checked by the server.

HandStore clones/freezes confirmed snapshots, ignores equal/older versions and
prevents submitting terminal hands. Router checks event/result protocol versions;
full private-view DTO mapping is intentionally deferred, never inferred from a
command result or change notification.
