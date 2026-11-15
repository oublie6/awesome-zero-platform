# 03 — WeChat-First Delivery, Cross-Platform Battles

## Product invariant

All supported clients — WeChat Mini Game, Web/H5, future mainland Douyin Mini Game, Android, iOS and others — play in **the same backend rooms and hands**. Platform selection must never determine game rules, authority, seat or game instance. First qualified playable target: **WeChat Mini Game**. Web remains an integration/debug target. Do not block WeChat on unqualified Douyin tooling.

## Boundaries

- Go backend authenticates a platform code through the platform's trusted exchange, maps to stable internal `AccountID`, and authorizes operations using only that identity and server-owned membership. WeChat openid or a client-supplied seat is not authority.
- Store cross-platform identity binding as an explicit account capability; future secure linking is a separate flow with proofs and anti-takeover checks. A player may authenticate on multiple clients, but each live hand has a defined operation-control policy (one controlling session recommended) and deterministic duplicate-command handling.
- Game protocol, command state, fairness, retry and reconnect coordinators live in platform-neutral TypeScript packages. Cocos scenes own input, UI, animation and resource display. `wx`, `tt`, DOM/native APIs belong only in dedicated platform adapters/composition roots. Common Cocos UI/assets can be reused across targets subject to actual SDK/build qualification.
- Server owns cards, bids, turns, rules and settlement. Clients only maintain confirmed versions and local presentation state (selected cards, animation). Separate public events from each player's authorized private hand view.
- Durable room/fairness commands and live `Bid/Play/Pass` commands may use distinct actual Go wire DTOs and consistency semantics. Never send a conceptual client envelope directly without verified mapping. Version all wire contracts; define backward compatibility and errors.
- Use verified cryptographic entropy, never `Math.random` for fairness contributions. Reuse existing HPKE implementation; resolve server/client AAD issue/expiry and seat/identity context against **actual** backend before integrating reveal. No token in WS query parameters; fail closed where secure platform APIs are unavailable.
- Share adapters handle WeChat friend share or room-code copy; cross-platform joining relies on server room identifier/universal invite semantics, not a platform-specific link.
- Reconnect resynchronizes server state by version, retains stable command IDs when response is lost, never fabricates active cards. App foreground, background, network loss, duplicate message and cross-device control are explicit test cases.

## Performance and delivery

Cocos: sprite atlases, reuse/pooled card nodes when profiling supports it, dirty-part updates instead of whole-table reconstruction, animation not blocking command processing, lazy-loaded bundles, bounded assets and cleanup on scene disposal. Measure startup, peak memory, rendering frame time, UI event delivery and low-end device behavior.

WeChat qualification path: Creator real build → package-size/subpackage validation → simulator SDK/login/socket tests → user-authorized preview/upload → mobile interaction and lifecycle verification. Record artifact SHA/source commit binding. Simulated gesture tests are not real-user behavior, preview upload is not publishing, and an editor build is not a playable game.

Commercialization, coins, rankings, payments, matchmaking and AI are deferred. Do not introduce their assumptions into the game rules or platform interfaces.

## Unified platform account decision (2026-11-15)

- One server-issued immutable UUID `AccountID` owns all business data across Web, WeChat and future clients; login names, emails, third-party subjects and devices never become parallel account IDs or direct authorization authority.
- Public self-registration and administrator account creation share the Identity owner and database. Username and email are required and unique for self-registration; email ownership verification, real mail delivery and a public domain are deferred with explicit TODOs. Unverified emails cannot authorize recovery, email changes, unlinking or takeover-sensitive actions.
- An unbound verified third-party identity must choose **bind an existing authenticated account** or **create a new platform account** before receiving an ordinary session. No silent account auto-creation or automatic merging. The present WeChat auto-provision path is legacy behavior requiring replacement before this principle is considered implemented.
- External identity links are unique by `(provider, verified subject)` and `(provider, AccountID)` in the provider's application scope. Linking/relinking requires fresh proofs for both existing account ownership and provider identity; target AccountID must not already have that provider. Unlink/relink affects only mapping; past hands, achievements and all business data remain attached to original AccountID. No unauthenticated request may self-declare an OpenID or AccountID.
- Unlink needs a remaining usable login method and a recent strong authentication; email-based recovery or verification must not be pretended while email remains unverified. Guard binding changes atomically under concurrency and preserve existing history.
