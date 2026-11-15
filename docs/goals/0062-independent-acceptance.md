# Goal0062 independent acceptance — FAIL

2026-11-15 UTC. Fixed candidate **`2067ca63441cb1264e57a5014fcb4014d310bb32`**, clean synchronized `main` before inspection. **G62-N01/P2 confirmed**: concurrent verified WeChat login can leave a usable source-AccountID session after successful unlink. Per the mandatory stop gate, no candidate rollout or existing-database migration followed. ChatGPT owns the fix; this report does not complete the product Goal.

## Baseline and executed scope

Exact-SHA push workflows all SUCCESS: [CI43710819354](https://github.com/oublie6/awesome-zero-platform/actions/runs/43710819354),6/6 jobs; [Fair Doudizhu43710819331](https://github.com/oublie6/awesome-zero-platform/actions/runs/43710819331),4/4 jobs. Other native workflows were not triggered at this SHA; no missing native run was relabeled PASS. Only run/job metadata read, no CI reruns/full successful logs.

Private evidence E=`/root/.config/awesome-zero-platform/acceptance-0062-2067ca6` (0700). Actual checks ran sequentially with Go1.25.8, `GOMAXPROCS=2`, `GOMEMLIMIT=768MiB`, `-p 1 -parallel 1`, memory cap1536MiB/zero swap/150%CPU:

- `go test -p 1 -parallel 1 ./platform/identity ./apps/app-api/internal/securityhttp`: both affected packages PASS.
- Real MySQL `TestIdentityIntegrationLifecycle`, `TestExternalIdentityBindingIntegration`, `TestConcurrentProviderSubjectClaimIntegration`: **3/3 PASS**, no skips. These exercise Identity/password lifecycle, normal explicit linking/new-account creation/relink/unlink, consumed-proof rejection, uniqueness and concurrent subject claims.
- One independent `TestG62UnlinkConcurrentVerifiedLogin`: **FAIL in6.046s**, real MySQL/InnoDB and real Redis/go-zero session adapter; actual candidate Identity, Authn, HTTP middleware and handlers. Provider Exchange alone is a controlled verified-subject fixture, **not real WeChat/device evidence**. No credentials, provider subjects or session tokens printed.

Stock Identity integration tests hardcode destructive cleanup against a default database. They were run only through a private Go overlay changing connection configuration to freshly created **`goal0062_accept_2067`**, with a dedicated SQL user granted only that database. All assertions/production sources remained intact. The race is an added virtual test file via `go -overlay`; the repository was never modified to inject it. Existing `goal0057` data/schema were not test cleanup targets. Logs/overlay/fixture and safe connection settings are retained privately; credential file0600.

## G62-N01 — post-unlink session revocation race (P2)

Affected source: `server/apps/app-api/internal/securityhttp/wechat.go:284` revokes source sessions **before** `binder.Unlink` at288 removes the mapping. Concurrent `code` at119–126 can still resolve that committed mapping and issue a new session after the revoke scan has finished. Unlink does not invalidate this newly created session before reporting success.

One deterministic reproduction, without production hooks or fake database/session results:

1. Create an isolated active UUID account with password and verified-provider mapping; issue a normal source session.
2. Hold a real InnoDB `SELECT ... FOR UPDATE` lock on that mapping. Start the actual password-reauthenticated unlink handler through authentication middleware.
3. Confirm the initial session is revoked; its mapping DELETE is blocked by the row lock.
4. Call the actual WeChat code handler with the controlled verified provider identity. MySQL MVCC still exposes the committed source mapping, so real Authn/Redis issue a new session.
5. Release the lock. Assert unlink HTTP200 and `ResolveVerified` now returns unbound.
6. Send the new access token through the real authentication middleware and `/auth/session` handler: **HTTP200**, expected401. It still authorizes the source AccountID after the identity has been unlinked.

This defeats the unlink session cutover guarantee: an actor still controlling the removed provider identity can retain access via the racing session. Normal prior-session revocation and mapping removal both passed, so the failure is not a missing password check, stale proof, test timeout or an unsuccessful unlink. The analogous relink ordering at339–347 is a related review risk; its race was **not separately executed** after the stop gate.

Reproduce from `server`, with the retained private fixture and loopback forwards to test MySQL33362/Redis36362:

```bash
G62_FIXTURE_CONFIG="$E/fixture.json" GOMAXPROCS=2 GOMEMLIMIT=768MiB \
  /opt/goal0042-toolchain/go/bin/go test -p 1 -parallel 1 -tags=integration \
  -overlay="$E/race-overlay.json" -run '^TestG62UnlinkConcurrentVerifiedLogin$' \
  -count=1 -v ./apps/app-api/internal/securityhttp
```

Expand E above; this requires the isolated fixture, never the stock/default or existing game database. `unlink-race.log` SHA256 `496aac01fe08f273c20f79847fd552459e455fc9f0d6878492d4abdefa66291f`; fixture source SHA256 `5154c97136dd902ea065530c8158cad6a38abce95447c6b7521e98d79c0cab20`. `result.json` records other command-log hashes. ChatGPT must establish a consistent session invalidation boundary for successful binding changes and add concurrency regression coverage; Codex must rerun this original case plus affected regressions at the repaired SHA.

## Migration, reuse and unverified gates

Read-only deployed-DB preflight found **0 duplicate provider/account groups**, existing unique primary key `(provider,subject_hmac)`, but missing `uq_identity_external_provider_account` and `identity_external_pending`. Candidate deployment therefore needs a reviewed additive migration with backup/compatibility/rollback. **No migration was applied to the existing database**, no full schema script run against it; the complete candidate schema was used only for the verified-new empty fixture DB. Existing Deployment specs, Pod UIDs/restart counts and account-schema preflight outputs match before/after. Web/API/MySQL/Redis remain **1/1** on their existing images; pre-existing API2/Web1 restarts did not increase. No Secret/protection-key changes or game-data deletion.

Scoped reuse: Git comparisons against Goal0061 product1dd54db show unchanged `server/business`, reveal/secure-envelope infrastructure, shared client packages, GameFairnessActions and GameClientShell. `reuse.json` verifies historical full-game evidence hash `e5f23129cfe21c555e3a639b61390093d9f957f528211385b51d7d91764a3daa` and its retained pinned-check result/hash. This preserves only their previously stated game/crypto/recovery scope; it does not qualify the changed identity screens or candidate deployment.

After confirmed P2, Vue registration/typecheck/build, Cocos binding UI/runtime, complete privileged-registration/email and binding security matrices, expanded proof expiry/relink races, candidate image builds and K8s registration/bind/unbind smoke were **not executed**. Real provider/device acceptance was not attempted or claimed. No unrelated Cocos rebuild/native smoke, full game, crypto, pressure or recovery campaign repeated. Existing services and the user-requested `azp-game-test-forward` remain running; only our MySQL/Redis test forwards and test sessions were cleaned up. Isolated fixture/evidence retained for repair reproduction.

Goals0045/0046 and Goal0062 implementation remain open; Goal0056 Honor pause, C09/C10 DEFERRED, mail/domain/SMS/account-merge and legacy-account recovery exclusions remain unchanged. **Overall FAIL**, one confirmed P2; candidate must return to ChatGPT without product repairs by Codex.
