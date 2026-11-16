# Goal0062 — Round4 fresh realtime authority-query optimization

2026-11-16 — ChatGPT development checkpoint, **pending exact-SHA CI and Codex independent acceptance**. Round3 accepted source SHA `d50c6fb0f56c94280264d32a13a516dc3653f131`; its PASS and historical Round1/2 FAILs stay unchanged.

## Why this optimization

Round3 solved stale WebSocket authorization by validating on each received envelope, outbound message and idle heartbeat. The independent real-MySQL/Redis small serial measurement showed P95 approx. 26.155ms (diagnostic baseline without fresh validation) vs. 139.758ms (with fresh validation). This is **not** a real production capacity benchmark. Reducing round trips rather than accepting a stale authorization window is the primary design requirement.

## Production changes and dependency boundaries

- **Authn** owns `ValidateRealtimeSession`. The server-side SessionID and AccountID were already verified during the WebSocket handshake. A new optional `RealtimeSessionReader` fetches session plus account revocation epoch in a single Redis pipeline. It still checks the exact account, current revocation epoch and the session's expiry on every call. Without the optimized adapter (e.g. existing memory unit test stores) the original `SessionStore.Get` path is retained.
- **Identity** owns account/provider/credential state. The production `identityprovider.Provider` implements `RealtimeAuthorityVerifier`. It delegates to the Identity service and a **single fresh MySQL statement** for active AccountID plus provider subject and/or the SHA-256 fingerprint of the previously verified Argon2id credential hash. Only a boolean decision is returned. SQL checks indexed AccountID/provider mappings and does not transmit password hashes. The production provider fails closed if this capability is unavailable.
- **No authorization cache:** status, provider association, password credential and Redis account epoch are checked on each authorization decision; no cross-replica invalidation or TTL-stale positive grants. The SQL query is a single read, not a cross-Redis/MySQL transaction. Previously authorized in-flight work is still subject to the business handler's transaction semantics.
- **go-zero first:** the Redis pipeline uses go-zero's `PipelinedCtx` when configured, or the existing native Redis client for local adapter tests. Identity uses the existing `sqlx.SqlConn` with the application's existing MySQL pool.
- **Compatibility:** no protocol, schema, Go HTTP route, game domain, or client change. Existing WebSocket Authenticator methods and connection identity are preserved. Legacy untagged third-party sessions still require the controlled rollout/drain documented in Round3.

### Expected remote-call reduction per realtime liveness check

| Path | Previous | Candidate |
| --- | --- | --- |
| Redis session + account epoch | Two sequential GET commands | Two GET commands in one pipeline / one network round trip |
| MySQL account + password version | Separate account-status and password-credential reads | One indexed boolean statement |
| MySQL account + provider binding | Separate account-status and provider-mapping reads | One indexed boolean statement |
| Additional positive cache lifetime | None | None |

This is a reduction in database/network round trips, **not** a proven P95 improvement until checked on the actual representative environment. SQL SHA2 equivalence with the application's Go credential fingerprint and behavior on the existing MySQL 5.7 configuration are covered by the isolated integration test.

## Test gates

- [ ] Exact frozen SHA: all CI, game workflow, relevant Go Race and isolated MySQL/Redis integration tests pass.
- [ ] Combined-query account status, password reset, link/unlink and incorrect proof checks pass in MySQL 5.7.
- [ ] Redis pipelined reader enforces exact AccountID/epoch, including a resurrected stale Redis key after account revocation.
- [ ] Existing WebSocket authority validation still fails closed when Redis or MySQL is unavailable.
- [ ] Codex independently reruns preserved Round3 actual socket + provider lifecycle fixture and compares representative before/after latency at the same environment, data and concurrency.
- [ ] No performance SLO claimed from micro/small-serial samples; compare P50/P95/P99 and connection count/DB pool saturation where environment permits.
- [ ] Do not update the user's existing K8s test workloads, alter passwords or migrate business MySQL schema during this code optimization.

Goal0062 remains `in_progress` pending separate rollout and provider/device qualifications. Future email sender/domain/SMS TODOs remain out of scope.
