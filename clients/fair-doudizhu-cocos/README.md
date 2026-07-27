# Fair Doudizhu Cocos Client

Minimal code-first Cocos Creator 3.8 LTS skeleton for the Fair Doudizhu client.

## Intent

- TypeScript owns application, domain, networking, security, layout, and animation logic.
- Cocos scenes remain thin composition roots instead of business-logic containers.
- Platform APIs are isolated behind adapters so Web and other mini-game targets can be added later.
- The reusable HPKE client lives in `clients/packages/secure-envelope` and is not coupled to Cocos.

## First local import

1. Install Cocos Creator 3.8.8 from the Cocos Dashboard.
2. Import this directory as an existing project.
3. Let Creator generate ignored `library/`, `temp/`, `local/`, and `profiles/` directories.
4. Create one empty 2D scene with a Canvas root.
5. Attach `GameBootstrap` to a root node and run Preview.

The repository intentionally does not commit editor-generated caches or a binary scene assembled by drag-and-drop. The first scene is a minimal host for code-created UI.

## Secure random on WeChat

`WechatRandomSource` wraps the asynchronous `wx.getRandomValues` API. The secure-envelope package prefetches those true random bytes before a seal operation and exposes them synchronously only for the duration of that operation, because HPKE libraries expect the Web Crypto `getRandomValues` contract. It fails closed when secure entropy is unavailable or exhausted and never falls back to `Math.random`.
