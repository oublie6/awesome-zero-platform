# Goal0062 Round3 independent software acceptance

2026-11-16 UTC. Fixed product **`d50c6fb0f56c94280264d32a13a516dc3653f131`**. Final disposition: **PASS for the Round3 software checkpoint**. Original G62-R2-N01 is resolved in the reproduced and expanded scenarios; no new substantive product defect was confirmed. This is an independent account/realtime software checkpoint, not deployment, native/device qualification or completion of the entire Goal0062 product.

## Source, CI and authority

Entered on clean `main`, synchronized with `git pull --ff-only` to exactly the requested candidate; no later product update. Read AGENTS, canonical principles, current Goal and the unified-account/Round2/Round3 repair contracts. Inspected the complete candidate production diff and affected call paths. Only this report and permitted current-goal status sections are deliverables; no product source, stock assertions, schema, client, image or deployment edits by Codex.

Exact-SHA push workflows SUCCESS: [CI43808181819](https://github.com/oublie6/awesome-zero-platform/actions/runs/43808181819), **6/6 jobs**; [Fair Doudizhu43808181828](https://github.com/oublie6/awesome-zero-platform/actions/runs/43808181828), **4/4 jobs**. Read run/job metadata only, no rerun or download of successful full logs. CI is supporting evidence, distinct from the independent checks below.

Private E3=`/root/.config/awesome-zero-platform/acceptance-0062-r3-d50c6fb` (0700), prior E2=`/root/.config/awesome-zero-platform/acceptance-0062-r2-406f156`, E1=`/root/.config/awesome-zero-platform/acceptance-0062-2067ca6`. Real dependency probes reuse the **isolated `goal0062_accept_2067` fixture**, scoped SQL credentials and unique Redis prefixes, never existing `goal0057` business records. New fixture accounts/sessions are test data; this is not a claim of zero writes to the dedicated acceptance fixture. No existing business database cleanup, DDL, migration or data transfer.

Sequential Go1.25.8, `GOMAXPROCS=2`, `GOMEMLIMIT=768MiB`, `-p 1 -parallel 1`, systemd1536MiB/zero swap/CPU150% caps. No parallel build/agents or local dependency stack startup. Controlled provider-code Exchange is retained from Round2; actual Identity, Authn, InnoDB, Redis/go-zero session adapter and binding handlers are used. No official WeChat or physical-device qualification.

## Original Round2 defect

**G62-R2-N01 regression PASS**: preserved `TestR2ExistingSocketAfterUnlink`, with only the new mandatory `Authenticator.ValidateSession` adapter delegating to real `authn.ValidateRealtimeSession`. Same real Hub/WebSocket, account read handler, MySQL/Redis fixture, provider session and actual password-reauthenticated unlink. Baseline message works; unlink200, old access/refresh rejected, new handshake401, post-unlink old socket no longer dispatches to the account handler. **PASS5.261s**, `original-socket.log`.

The preserved original assertion checks dispatch suppression; the stronger lifecycle cases below additionally require a standard1008 close and empty connection/account/topic maps. Round1 and [Round2 FAIL](0062-round2-independent-acceptance.md) remain historical records, not overwritten or relabeled as previously successful.

## Independently executed checks

| Requirement | Result and evidence |
| --- | --- |
| Affected units | **11 packages PASS**: realtime, authn, its4 adapters, admin, identity, bootstrap, securityhttp and doudizhuapi. Admin mysqlstore has no test files. `units.log`. |
| Race checking | **4 packages PASS**, no race report: realtime, authn, bootstrap, doudizhuapi. `race.log`. First-use race compilation completed within the bounded unit; it was not restarted or overlapped with another build. |
| Real account/session cutover | **8 subcases PASS**: logout, password change, administrator reset, disable/re-enable, WeChat unlink, relink, account epoch revocation and fixture Redis-client unavailability. Each begins with normal message delivery; revoked inbound handler dispatch and outbound account/topic events are denied, a separate silent connection closes via heartbeat, all maps empty, old handshake401 during the invalid state, fresh login/new browser connection succeeds. Actual middleware/handlers/service mutations, not cause labels on a fake validator. `TestR3RealLifecycleSocketCutover`,38.42s. |
| Multiple sessions and token rotation | **PASS** two separately issued sessions for the same UUID receive account-directed pushes. Refresh rotates credentials and rejects refresh replay while the existing valid socket works; logout of one session closes only that session. The other continues handling messages, unsubscribes successfully, receives no later topic event, closes cleanly and reconnects after fresh login. `TestR3IndependentSessionsRefreshAndTopics`. |
| Healthy heartbeat | **PASS** real protocol ping/pong while an application reader waits for a delayed account push; healthy connection remains registered. Fifty serial account-read request/replies also succeed with accelerated heartbeat. `TestR3RealLatencyAndIdleMaintenance`. |
| Access/session expiry | **PASS** actual short TTLs: expired JWT denied by access validation/new handshake, established socket still works while server session is valid, refresh/new handshake works, extended server-session expiry eventually closes old socket1008 and clears maps. `TestR3AccessExpiryAndSessionExpiry`. |
| Events queued before revocation | **2 subcases PASS**, account and topic. A gate pauses immediately before the writer's real authoritative validation, events queue before logout, then validation resumes against real revoked Redis state; no payload delivered, close1008 and cleanup. Snapshot/enqueue remain usable while validation is blocked: no Hub lock held across the gate. |
| Validation timeout | **PASS** the same gate honors its one-second context deadline; silent validator failure closes1008 and cleans all maps. It simulates a stalled validation boundary, not a shared DB/Redis outage. Together with the queued cases: `TestR3QueuedBeforeRevokeAndValidationDeadline`,6.00s. |
| Handler already in progress | **PASS** handler starts while valid, logout completes, handler is released; no stale response reaches peer and socket closes1008. No claim that an already-started business transaction is rolled back. `TestR3InFlightHandlerResponseIsSuppressed`. |
| Actual network game adapter | **PASS under Race**: actual WebSocket/Hub, production game bridge and Dispatcher process a room-create envelope, preserve connection-owned actor and emit the correct result; after revocation, close1008 and no second backend call. Identity validator and game backend are controlled test doubles: not a fresh real game or durable room commit. `TestR3ActualHubGameBridgeAndDispatcher`, `game-bridge.log`. |
| Blocked writer/backpressure | **PASS** actual socket writer pauses at validation; capacity1 queue accepts one waiting frame, rejects overflow, cancels blocked validation, closes1013 and clears resources. `TestR3RealBlockedWriterQueueOverflow`, `extra.log`. |
| Default-heartbeat cost | **PASS measurement**, not a performance-SLO qualification. Two30-request serial samples use the default25s heartbeat. Controlled validation-bypass baseline RTT P50/P95/max **6.104/26.155/28.252ms**; actual fresh validation **41.736/139.758/260.121ms**,60 calls totaling1517.835ms. Same fresh account-read handler/real dependency forwards; only the test authenticator bypasses validation for the diagnostic baseline, never production. `TestR3ValidationCostAtDefaultHeartbeat`, `extra.log`. |

Both extra tests PASS in6.771s; the game-bridge Race check PASS in1.102s. The six lifecycle/queue top-level groups, including8 lifecycle and3 queue/deadline subcases, all PASS in **67.287s**, `lifecycle-final.log`. Baseline handshakes cover native Authorization header and browser `bearer` subprotocol; server returns only `bearer`, never a query token. Stock realtime units additionally cover malformed/invalid handshakes, topic validation, generic messages/custom handlers, account limits, bounded send queues, error redaction and trace helper contracts. Game adapter units exercise command identity, private per-account snapshots, sync/evidence, malformed/forbidden requests and broadcast failure handling.

Admin mutations above use real Identity/MySQL/Redis with the preserved role/audit repository test doubles. They prove lifecycle behavior, not administrator UI/authz/audit delivery end to end. Closed-client dependency failure does not shut down the shared Redis server: the old connections fail closed; recovery creates a new client/service/Hub and fresh login succeeds. The probe does not claim that temporary dependency loss permanently revokes otherwise-valid tokens. Network peer close and reconnect are exercised without modifying cluster networking.

## Source review and tested invariants

- `platform/authn` owns the server session lifetime and authoritative account/provider/password state. `ValidateRealtimeSession` requires the captured session ID/account pair, live expiry, active account, current provider mapping and credential fingerprint; the peer cannot replace this identity through a frame.
- `platform/realtime/client.go` validates text frames before JSON/application dispatch and outbound queued payloads before writing; idle heartbeat also validates. `hub.go` uses a bounded context and releases Hub locks before authority checks. Any invalid/unavailable validation terminates the connection and unregisters account/topic ownership. No independent authorization cache or game-specific platform policy was introduced.
- Bootstrap delegates the new typed interface to Authn. Existing server-owned AccountID and game adapter contracts remain unchanged. Validations are not a distributed transaction with game handlers; operations authorized before revocation may already be running, while responses are rechecked.
- Reviewed cancellation/close ordering, reader/writer ownership, close codes, full queues, stale provider/password data, session rotation/expiry, retry boundaries and cleanup. `gofmt -l` on changed production files returned empty. No product fixes were made during review.

## Fixture corrections and limitations

Raw failed attempts are retained, not counted as product failures or silently discarded:

1. Initial systemd invocations lacked the root user environment: kubectl could not locate its context; Go overlay rejected an empty module-cache configuration. Both were terminal setup failures. Recreated only owned units with explicit root user/known kubeconfig; existing business services were unaffected.
2. The added lifecycle fixture initially mishandled the two return values of `RevokeByAccount`, causing a compile error. Corrected only the private fixture before execution.
3. Initial multi-socket run produced socket-read failures. A single labeled password-change diagnostic proved that incoming/account/topic checks passed and the silent socket failed while its client tried to write a pong to the already-closed peer (`broken pipe`). The test had paused reading during mutation and accumulated control pings. The final **close-drain helper only** ignores such queued pings, still strictly rejecting any application payload and requiring the standard1008 close plus empty maps. Normal healthy-heartbeat checks retain actual automatic pong behavior. All lifecycle/expiry cases then pass. `lifecycle.log`, `diagnostic.log` and `lifecycle-final.log` preserve the distinction.

Loopback `ws://` is used for local software checks; no new TLS/device/deployed qualification. Dependency/validation boundaries are deliberately controlled; no shared-server stop, process crash campaign, general network partition, long soak or large client-count capacity claim. Actual OTLP linkage of newly added authority queries is not qualified by the trace-helper units. Tests and private overlays were reviewed for genuine controls and false-positive PASS paths; this fixture review is not a second independent product acceptance.

## Historical reuse and final environment

Comparison from Round2 product406f156 to this candidate is empty for `server/business`, clients, database schema, Identity/Admin/authn adapters and reveal/secure-envelope infrastructure. E3 `reuse.json` records these comparisons and hashes retained Round2 logs. Reuse **4 stock real MySQL +1 Redis integration passes**, registration/binding/concurrent-proof/HTTP cutover/failure evidence and **Vue7/Cocos120 tests plus typechecks/Web build** only within their originally stated scopes. They were not rerun or called fresh native/device evidence. Existing game/crypto/full-hand/recovery evidence remains as recorded in Round1/Round2; no unaffected full game or native export repeated. This round separately tests the changed realtime boundary; unchanged game logic alone does not prove that boundary correct.

Final before/after snapshots show **all4 Deployment specs/UIDs and Pod UIDs/restart counts unchanged**, all4 services1/1. API2/Web1 existing restarts did not increase. Original images, Secrets/protection roots, existing business database/schema and data were not changed; no rebalance or shutdown was needed. All test processes exited. Stopped only owned `codex-g62r3-mysql-live`/`codex-g62r3-redis-live` forwards; user **`azp-game-test-forward.service` stays active at127.0.0.1:29457**. Private evidence and isolated fixture retained.

E3 `result.json` records log/source/overlay hashes and `environment-result.json` records the unchanged-state comparison. Key SHA256 values: original adapted socket source `04c9ab2c1ed8ac1fae71ddc90d9ab0eb1d84a17246fa9fd852a6fc06dde215f6`; final lifecycle source `99ac9fe79b6a594a2e6e1f460ea1c98a547cc0233b5df9ba2f3863fd8904dbe2`; final lifecycle log `1890ac3cfd1e1f121d557f6834b84bb6e1683e24fbd5c2866d3f8f0b9901ed82`; evidence manifest `a39b3d35c5f33a23dd63587950d78ccf57c87b8c59f7bd6b35d06108f95d927d`. No credentials or tokens are committed.

## Disposition and next stage

**PASS for this source-level software checkpoint.** Original defect and expanded realtime/account lifecycle checks pass. No new P0/P1/P2 defect was established. Fresh validation has a measurable latency cost; the small forwarded serial samples do not qualify in-cluster latency, production capacity, per-connection memory or long-term leak budgets. Those operational performance checks remain open before a broader deployment/performance acceptance. A passing source checkpoint permits moving to separately reviewed schema/rollout work; it does not authorize that work or establish deployed account/binding behavior. Goal0062 remains `in_progress`, as do Goals0045/0046. Existing migration prerequisites, controlled legacy-session drain/old API replacement, real provider/device checks and other product gates remain open. Goal0056 Honor pause, C09/C10 DEFERRED and mail/email ownership/domain/phone/SMS exclusions are unchanged.

## Reproduction commands

From `server`, expand E1/E3 above and restore only the isolated fixture forwards if needed. The original fixture uses `original-socket-overlay.json`; final lifecycle uses `lifecycle-final-overlay.json`; extra tests use `extra-overlay.json`. Never substitute the default or existing game DB.

```bash
GOMAXPROCS=2 GOMEMLIMIT=768MiB /opt/goal0042-toolchain/go/bin/go test \
  -p 1 -parallel 1 -count=1 ./platform/realtime ./platform/authn/... \
  ./platform/admin/... ./platform/identity ./apps/app-api/internal/bootstrap \
  ./apps/app-api/internal/securityhttp ./apps/app-api/internal/doudizhuapi

GOMAXPROCS=2 GOMEMLIMIT=768MiB /opt/goal0042-toolchain/go/bin/go test \
  -race -p 1 -parallel 1 -count=1 ./platform/realtime ./platform/authn \
  ./apps/app-api/internal/bootstrap ./apps/app-api/internal/doudizhuapi

G62_FIXTURE_CONFIG="$E1/fixture.json" GOMAXPROCS=2 GOMEMLIMIT=768MiB \
  /opt/goal0042-toolchain/go/bin/go test -p 1 -parallel 1 -tags=integration \
  -overlay="$E3/lifecycle-final-overlay.json" -run '^TestR3' \
  -count=1 -v ./apps/app-api/internal/securityhttp

GOMAXPROCS=2 GOMEMLIMIT=768MiB /opt/goal0042-toolchain/go/bin/go test \
  -race -p 1 -parallel 1 -overlay="$E3/game-bridge-overlay.json" \
  -run '^TestR3ActualHubGameBridgeAndDispatcher$' -count=1 -v \
  ./apps/app-api/internal/doudizhuapi
```

For the original reproduction use `-run '^TestR2ExistingSocketAfterUnlink$'` with `original-socket-overlay.json`. For the extra overlay use `-run '^(TestR3ValidationCostAtDefaultHeartbeat|TestR3RealBlockedWriterQueueOverflow)$'`; same fixture environment and package as the lifecycle command. All commands above were run with the stated external resource bounds. Documentation delivery commit is distinct from the fixed tested product SHA.
