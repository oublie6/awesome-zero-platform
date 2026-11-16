# Goal0062 — Round3 established WebSocket session consistency repair

Date: 2026-11-16. ChatGPT development checkpoint after Codex Round2 FAIL `G62-R2-N01/P2`.

## Background

Codex verified that a WebSocket established before provider unlink could continue dispatching using the cached AccountID after HTTP access/refresh were already rejected. Handshake-only authorization did not cover the lifetime of the upgraded connection. The report `0062-round2-independent-acceptance.md` remains intact.

## Owner and authority

- `platform/authn` owns the authoritative session and account state, including current Redis revocation epoch and password/provider mapping validation.
- `platform/realtime` owns the long-lived transport, bounded send queues, connection lifecycle and message routing. It must not invent an independent identity policy.
- The API composition adapter translates a previously authenticated WebSocket `Identity{AccountID, SessionID}` into an `authn.ValidateRealtimeSession` call; the peer cannot supply these values on subsequent frames.
- No bearer token or refresh token is retained on the established WebSocket. Authentication uses the trusted server-captured session ID and AccountID, with a fresh Redis session lookup and active account/provider/password checks.
- Existing `AccountID` and game command authority do not change. No game records, schema, secrets or clients are migrated.

## Implemented source behavior, pending acceptance

1. Validate every incoming WebSocket frame before parsing or dispatching it, including topic subscription and application handlers. Deny and terminate on any invalid/unavailable account/session.
2. Validate every queued outbound frame at the writer before transmission (including account-targeted and topic-based events); an invalid connection cannot continue receiving private events from its send queue.
3. Revalidate on idle heartbeat so a revoked but silent socket is eventually removed and unsubscribed.
4. Send a bounded standard WebSocket close frame consistently when validation closes the connection. Unregister from account/topic maps; drain no stale event payload.
5. `authn.ValidateRealtimeSession` uses the *server-side session expiry*, allowing an established WebSocket to remain valid after the initial short-lived JWT's access expiry while its server session remains active. A fresh connection still requires a valid access JWT. Credential rotation, provider unlink/relink, account disable and global revocation must all invalidate this established session as normal.
6. The underlying transport uses a per-call context deadline from the existing handshake timeout, does not hold Hub locks over Redis/MySQL queries, and avoids persistent positive validation caching so a just-revoked session is not authorized on a later frame.

## Known trade-offs and independent checks

- Fresh validation on every inbound and outbound application frame is intentionally conservative; it adds Redis and active-account database work to each real-time operation. Measure affected latency under representative normal traffic before claiming the transport performance gate; no cached authorization time window should silently weaken account cutover.
- In-flight handlers that began while a session was valid may finish computation after a concurrent revocation. Outbound responses are checked again, but this design does not claim database mutations already in progress are retroactively rolled back. Independent acceptance should explicitly distinguish post-revocation *new* dispatch from an already-running handler.
- Server-side session validity is an internal application-authority API only, never an unauthenticated network endpoint.
- Legacy deployed WebSocket sessions and untagged legacy provider sessions require a controlled rolling rollout with old API pods drained and old sessions invalidated; this source change does not perform a deployment.
- Basic topic subscriptions in the platform realtime layer are generic public channels. Private game state uses account-directed delivery and per-account private views; introducing protected topic classes requires an explicit authorization contract in a later change.
- Compare with the original Round2 socket reproduction (live MySQL/Redis + real Hub + actual authn). Independent Codex must execute its preserved fixture, verify denied application dispatch, no outbound private push after logout/unlink/relink/reset/disable, new handshake rejection and connection cleanup, then cover Vue/Cocos/HTTP regressions as applicable. Do not claim game-data mutation was reproduced by the original QA handler.

## Test and rollout gates

- [ ] Final candidate Go format, relevant units and real Redis/MySQL integration pass on same SHA.
- [ ] Final candidate Fair Doudizhu CI passes.
- [ ] Codex independently reruns Round2 original socket regression plus affected cases and records PASS/FAIL/BLOCKED.
- [ ] Database additive migration and isolated K8s deployment separately reviewed/authorized.
- [ ] Real WeChat credential/device verification remains deferred.
- [ ] No email verification, sender, domain purchase or phone/SMS in this scope.

No independent PASS or deployed-websocket qualification is claimed here.
