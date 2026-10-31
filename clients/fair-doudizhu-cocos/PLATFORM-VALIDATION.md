# Goal 0045 platform validation status

This iteration implements reusable packages, browser/mini-game adapter *code*, and a code-created Cocos entry shell. It has NOT executed Cocos Creator 3.8.8, WeChat developer tools, Douyin developer tools, real-device tests or backend game login. These must remain verification gates before Goal 0045 can close.

## Build sequence for Codex

1. Node 22: run `npm install && npm test` in each of `clients/packages/game-client-core` and `clients/packages/doudizhu-client`; run `npm run build` in `clients/packages/secure-envelope`.
2. Install the dependencies in `clients/fair-doudizhu-cocos`. Import project using Cocos Creator **3.8.8**. Validate project compiles and inspect missing .meta/scene assets; do not claim scene start until actual editor build or preview proves it.
3. Validate browser transport through a real HTTPS/WSS endpoint. In particular Cocos/browser subprotocol use must match the Go server.
4. Verify WeChat and Douyin SDK support for `connectSocket.protocols`, SocketTask callback semantics and secure entropy; this code is intentionally fail-closed on unsupported entropy. Current mini-game adapter interface is not a proof of SDK signature support.
5. Run real browser/mini-game runtime, capture screenshot and logs, and report blocked prerequisites without marking passed.
6. Do not issue arbitrary game commands or modify production accounts in the absence of authorized test identities.

## Next functional work

Goal 0046 integrates concrete platform login, authenticated HTTP/WSS sessions, typed room/hand DTO mapping and safe-retry sequencing. Goals 0047-0050 provide fully playable room and table views, animations, full game flow and release packaging.

The current GameBootstrap only displays a responsive entry status. There is no genuine lobby or playable card table yet; neither should be described as complete.
