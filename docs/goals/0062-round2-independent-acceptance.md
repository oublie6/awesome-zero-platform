# Goal0062 Round2 independent software acceptance — FAIL

2026-11-16 UTC. Fixed product **`406f156b8f1fa851042eaff5f0fc1fcd889bad3e`**, clean synchronized `main` at entry. **Original G62-N01 HTTP cutover reproduction now passes; a new P2, G62-R2-N01, remains: an already-established WebSocket continues dispatching with the revoked session's cached AccountID after successful unlink.** Stop gate applied; no product repair or candidate deployment by Codex. Goal0062 implementation remains `in_progress`.

## Source, CI and evidence boundaries

Exact-SHA push workflows completed SUCCESS: [CI43790731066](https://github.com/oublie6/awesome-zero-platform/actions/runs/43790731066), [Fair Doudizhu43790730683](https://github.com/oublie6/awesome-zero-platform/actions/runs/43790730683). Run metadata reused; no workflow reruns or claim of a new native export. The original [Round1 FAIL](0062-independent-acceptance.md), [delivery contract](0062-unified-account-identity-delivery.md) and [repair notes](0062-security-repair.md) remain intact. Read the current Goal, AGENTS and applicable canonical principles before review; no architecture or acceptance amendment.

Private evidence E2=`/root/.config/awesome-zero-platform/acceptance-0062-r2-406f156` (0700), prior E1=`/root/.config/awesome-zero-platform/acceptance-0062-2067ca6`. Reused the isolated **`goal0062_accept_2067`** MySQL fixture/dedicated scoped SQL user and private Redis prefixes; never the existing `goal0057` business database. Provider Exchange is controlled verified-subject input; Identity, InnoDB, Authn, Redis/go-zero adapter, middleware and binding handlers are real. This is not official WeChat exchange/device qualification.

Commands ran sequentially with Go1.25.8 (`/opt/goal0042-toolchain/go/bin/go`), `GOMAXPROCS=2`, `GOMEMLIMIT=768MiB`, `-p 1 -parallel 1`, systemd memory1536MiB/zero swap/CPU150%. Client checks had a1GiB Node heap bound. No concurrent build, stress, full game, native engine export or unrelated crypto campaign. Review introduced only private Go overlays and this documentation; production source and stock assertions were unchanged.

## Original G62-N01 regression

Replayed the prior real InnoDB row-lock schedule with the original assertions. The only bootstrap addition is the now-required `ConfigureExternalBindingVerifier(binder)` wiring. First run (`original.log`) failed its five-second scheduling precondition, “unlink did not reach session revocation”; it did **not** establish a product failure or successful race execution. A single diagnostic rerun adding timeout HTTP diagnostics, without weakening assertions, **PASS4.040s** (`original-diagnostic.log`): initial session revoked, competing verified-provider login during blocked DELETE, unlink200/mapping absent, replacement bearer rejected. No established cause is assigned to the first scheduling timeout.

Additional observer-synchronized real row-lock cases, separately for **unlink and relink**, avoid repeated expensive password polling: they observe the actual first Redis revoke, issue a competing provider session while the mapping is still committed, verify it works before cutover, release the lock and await successful HTTP completion. Both access **and refresh** then reject; a fresh password login remains usable. Both subcases PASS (`additional.log`). The original HTTP session-issuance defect is resolved in these executed schedules; this does not cover already-open sockets, below.

## Executed regression matrix

| Check | Actual result and scope |
| --- | --- |
| Affected Go units | **8 packages PASS**: Identity; Authn and identityprovider/jwthmac/redissession/wechatcode adapters; Admin; securityhttp. Admin mysqlstore has no test files. `units.log`. |
| Stock real integrations | **4 Identity tests PASS**: lifecycle, external binding, concurrent subject claim, credential fingerprint changes on reset/change. **1 Redis integration PASS**: session round trip/rotation/revoke plus missed stale-row rejection. `integrations.log`. |
| Registration and linking | **PASS** public201/UUID/unverified email/separate login; verified unbound pending flow; bind-existing; provider-register new account; consumed proof rejection; relink toB and old token rejection; unlink/password recovery/logout/fresh login. `TestR2RegistrationBindingRelinkAndUnlink`. |
| Credential/account lifecycle | **PASS** actual password change, administrator reset, disable/re-enable; old password/access/refresh rejected, enabled fresh login retains UUID and old sessions stay rejected. `TestR2PasswordAdminAndAccountState`. |
| Delayed password authentication | **PASS** pause after real credential verification, reset in MySQL, resume login: no usable tokens returned. `TestR2PasswordVerificationConcurrentWithReset`. |
| Redis epoch and refresh | **PASS** stale session row reinserted after account revoke remains rejected; two independent clients see the same epoch fence; concurrent refresh/revoke leaves old and rotated tokens invalid, fresh login works; epoch has no TTL. `TestR2RedisEpochAndConcurrentRefresh`. |
| Redis failure/recovery | **PASS** close only fixture Redis client: access/refresh/login fail, reset fails before changing credential, disable returns failure and remains disabled, enable refuses; new client recovery allows enable/fresh login without reviving old tokens. `TestR2RedisFailureAndRecovery`. Shared Redis server was not stopped. |
| Pending/failed relink | **PASS** occupied target400 with mappings preserved and fresh login usable; expired pending proof rejected; simultaneous create-from-one-proof creates exactly one account with no orphan. `TestR2AdditionalPendingAndFailedRelink`. |
| Registration privileges | **PASS** fresh username plus role injection400/no account; ordinary registration201/emailVerifiedfalse/UUID and zero assigned Casbin roles. `TestR2AdditionalRegistrationRoles`. This strengthens the earlier reused-username rejection control. |
| Mapping database failure | **PASS** close separate mapping-validator DB pool: external access/refresh fail closed while password login's existing session remains valid via the live Identity pool. `TestR2BindingDatabaseFailure`. Shared MySQL server was not stopped. |
| Vue game Web | **7/7 tests PASS**,0 skipped; `vue-tsc --noEmit` and Vite build PASS to private E2 output. Registration/session/origin/logout checks; not a new deployed browser acceptance. |
| Cocos compatibility | Static source typecheck and shell compilation PASS; **120/120 existing tests PASS**,0 skipped, including bound/unbound WeChat session contracts. No native export, device UI interaction or new browser gameplay claimed. |
| Established WebSocket | **FAIL**, one new P2 detailed below. |

The five `TestR2*` lifecycle/epoch/failure groups passed in18.126s (`regression.log`); the three additional top-level groups, including both cutover subcases, passed in13.162s (`additional.log`); separate mapping DB failure passed in2.535s. Admin service tests use real Identity/MySQL/Redis but stub role/audit repositories; they do not establish end-to-end administrator UI/audit delivery. Existing Admin units cover their own authorization contracts. No claim that every arbitrary interleaving or dependency outage was tested.

Stock Identity integration cleanup hardcodes another database. A **connection-only private overlay**, with dedicated-user permissions, redirects it to E1's isolated DB. Redis fixture configuration likewise uses private connection settings; all original assertions remain unchanged. No destructive stock test was run against existing game data. Credentials, passwords, provider subjects and tokens are neither included here nor printed as test results.

Representative commands from `server` (expand E1/E2 above; recreate only the isolated loopback dependency forwards if reproducing):

```bash
GOMAXPROCS=2 GOMEMLIMIT=768MiB /opt/goal0042-toolchain/go/bin/go test \
  -p 1 -parallel 1 -count=1 ./platform/identity ./platform/authn/... \
  ./platform/admin/... ./apps/app-api/internal/securityhttp

G62_FIXTURE_CONFIG="$E1/fixture.json" GOMAXPROCS=2 GOMEMLIMIT=768MiB \
  python3 "$E2/run-integrations.py"

G62_FIXTURE_CONFIG="$E1/fixture.json" GOMAXPROCS=2 GOMEMLIMIT=768MiB \
  /opt/goal0042-toolchain/go/bin/go test -p 1 -parallel 1 -tags=integration \
  -overlay="$E2/socket-overlay.json" -run '^TestR2ExistingSocketAfterUnlink$' \
  -count=1 -v ./apps/app-api/internal/securityhttp
```

Other executed overlays: `original-diagnostic-overlay.json`, `regression-overlay.json`, `additional-overlay.json`, `binding-failure-overlay.json`; exact test names/results above. Client commands: `npm test` and `npm run build -- --outDir "$E2/vue-dist"` in `clients/game-web`; `npm run typecheck` and `npm test` in `clients/fair-doudizhu-cocos`. Raw output is `clients.log`.

## G62-R2-N01 — existing socket retains revoked identity (P2)

**Observed, not hypothetical:** a real platform Hub connection authenticated by the real Authn service before unlink still invokes a registered handler with the original AccountID after unlink200 and access/refresh rejection. A new handshake using the same bearer correctly returns401. The handler performs a real fresh MySQL account read; it is a QA handler, not a mocked successful game command.

Reproduction (`TestR2ExistingSocketAfterUnlink`, `socket-cutover.go`):

1. Create an isolated password account and verified WeChat mapping; issue a normal provider session.
2. Open an actual loopback WebSocket through the platform Hub with Bearer header; receive hello. Send a baseline QA account-read message and verify the handler receives the original UUID.
3. Execute the real password-reauthenticated unlink handler through real authentication middleware. Assert HTTP200, old access/refresh rejection and a fresh socket handshake401.
4. Send another account-read message on the **original** socket. The handler is still invoked as the original account and reads its account record: **FAIL**,2.765s. Expected: a revoked session cannot continue authorized application dispatch.

Source cause at the fixed SHA:

- `server/apps/app-api/internal/bootstrap/bootstrap.go:348` delegates to `AuthenticateAccess` during authentication.
- `server/platform/realtime/http.go:39` authenticates only the handshake, then caches the resulting Identity on the connection.
- `server/platform/realtime/hub.go:278` dispatches later envelopes through `handler(ctx, connection.context(), envelope)` without session/provenance/epoch revalidation or a revoke-triggered disconnect.
- The real game adapter `server/apps/app-api/internal/doudizhuapi/realtime.go:100` uses cached `connection.AccountID` for dispatcher execution; its sync path likewise uses cached account identity. **Source-based inference:** the same stale identity can reach those production operations. This probe did not execute a real game command or mutate game records; its demonstrated runtime scope is the real Hub and fresh Identity read.

Thus successful HTTP token cutover alone does not establish the promised login-state cutover for existing clients. This is a gap in integrating the repaired session semantics with long-lived transport, even though the Hub source itself did not change. Password reset/disable/relink effects on existing sockets are related risks, **not separately reproduced** after the stop gate. ChatGPT owns a bounded design/fix and regression tests covering established connections; Codex made no product change or prescribed a second infrastructure framework.

Private socket setup had two compiler failures before the actual run: an attempted cross-package import cycle and then an unused import after removing that dependency. These are fixture errors, not product defects. `socket-setup-error.log`, `socket.log` and the initial segment of `socket-check.log` preserve them. Removing the unused private fixture import allowed the real failing test above; no product source/assertion was relaxed.

Reproducer SHA256: `3439b18cf57192bcf5ae3c0817881ea038350885868ebe9706be0938cb15e193`. Combined setup/actual `socket-check.log` SHA256: `8f263dea50e714c678b25f49fea0b9762d0f869030ebf0664e67899a55d2a116`. E2 `result.json` hashes the retained logs/overlays and records environment comparison; SHA256 `6541088b92e14bdfafe5aff162c0694cf3fc996c020c856675335fe589fc5c39`.

## Data compatibility, historical reuse and cleanup

Real fixture checks retain the original UUID through credential/account lifecycle; relink changes mapping to the destination UUID, leaves both account credentials usable and does not transfer business ownership. Source review shows no change to game tables/data owners or account ID format. **No existing business records were edited or game-history migration exercised**; fixture UUID checks and source comparison do not assert that a newly deployed candidate has passed historical-game runtime acceptance.

`git diff 2067ca6 406f156 -- server/business server/foundation/revealkeys server/foundation/secureenvelope clients` is empty. Round1's recorded comparison to Goal0061 product1dd54db and E1 `reuse.json` therefore still apply only to unchanged game/crypto/recovery behavior: retained business evidence hash `e5f23129cfe21c555e3a639b61390093d9f957f528211385b51d7d91764a3daa`, pinned check hash `20cccf76c687732cc53777bf861b7cffca3194984f6a5e5adeee89f6bf49b01d`. No repeated full-game/export/fault/crypto claim; changed session-to-socket integration is explicitly FAIL despite those historical passes.

Final real Kubernetes snapshots show **all four Deployment specs/UIDs and Pod UIDs/restart counts unchanged** from this round's baseline. API2/Web1 pre-existing restarts did not increase; API/Web/MySQL/Redis remain1/1 on their original images. No deployment, migration, Secret/protection-root change, existing business-data update, service redistribution or shutdown. Stopped only owned `codex-g62r2-mysql` and `codex-g62r2-redis` dependency forwards. Temporary tests exited; private evidence and isolated fixture retained. User's **`azp-game-test-forward.service` remains active on127.0.0.1:29457** for manual testing.

## Final disposition

**FAIL — G62-R2-N01/P2.** Original HTTP race repair and the enumerated regression checks pass within their stated scopes. Related further testing stopped after the confirmed socket failure; return to ChatGPT development, then independently reverify the repaired fixed SHA. This completes the Round2 review/report assignment, not the product's acceptance requirements.

Candidate rollout/additive existing-schema migration and deployed registration/binding smoke were not performed; existing schema prerequisites remain as in Round1. Real WeChat/device qualification, native exporter/device interaction, full administrator UI/audit integration and arbitrary fault/interleaving coverage are unverified here. Mail/email ownership, domain, phone/SMS and legacy WeChat-only recovery remain deferred. Goals0045/0046 and Goal0062 stay open; prior Goal0061 evidence, Goal0056 Honor pause and C09/C10 DEFERRED remain unchanged.
