# Goal0062 Round4 independent acceptance — PASS

2026-11-16 UTC. Fixed product **`a739e23f315a98ad76fd83c233aae1813415037a`**, clean `main` synchronized once by `git pull --ff-only`; no source switch. **Query optimization is active, the small matched comparison improves aggregate latency, and affected functional checks pass within the boundaries below.** No newly confirmed substantive product defect. This accepts the source optimization checkpoint, not deployment or completion of Goal0062.

Exact-SHA [CI43814460914](https://github.com/oublie6/awesome-zero-platform/actions/runs/43814460914) **6/6 SUCCESS**; [Fair Doudizhu43814460946](https://github.com/oublie6/awesome-zero-platform/actions/runs/43814460946) **4/4 SUCCESS**. Only run/job metadata read, no full CI rerun. Historical reports/conclusions unchanged.

## Diff and actual paths

Reviewed only changes from accepted Round3 `d50c6fb` plus necessary caller/library context. `ValidateRealtimeSession` selects `GetForAccount` and the Identity provider's combined verifier in production. Redis uses go-zero **v1.10.2 `PipelinedCtx`**, delegating to the existing Redis pipeline; native fallback remains for adapters without managed Redis. Identity's one uncached `SELECT EXISTS` checks its own indexed account, provider mapping and/or exact credential fingerprint. Real MySQL **5.7.44** checks confirm SQL SHA2/Go fingerprint equivalence. The query returns a boolean rather than the credential hash.

Account/session match, expiry, epoch and all presented proof predicates remain required; malformed/half-present/noncanonical proofs fail closed. There is no new positive authorization cache. No transport, bootstrap, game, client, schema or configuration change; no new client/native build. Missing combined capability fails closed on the production Identity adapter. Legacy/in-flight/rollout boundaries remain as documented in Round3.

## Matched small performance comparison

Reused Round3 fixture/program/configuration with real MySQL/Redis through the same localhost forwards. Same freshly created isolated account, provider mapping, stored sessions, SQL pool6/idle2, native Redis pool4, unchanged managed defaults,25s heartbeat, one client and identical fresh account-read handler for both paths. Go1.25.8, `GOMAXPROCS=2`, `GOMEMLIMIT=768MiB`, `-p 1 -parallel 1`,1536MiB/zero swap/CPU150% cap; all work serial.

No old checkout/build: interface-hiding test wrappers select the candidate's retained **source-equivalent Round3 fallback** versus optimized adapters in one binary. Both paths perform full authorization; this is not Round3's earlier validation-bypass diagnostic. For each login type, ABBA order,5 warmups then30 measured messages per block: **60 samples per path**,240 measured messages total. Call counters assert the actual selected paths. Nearest-rank percentiles, milliseconds:

| Session | Path | N | P50 | P95 | P99 |
| --- | --- | ---: | ---: | ---: | ---: |
| Password | Round3 fallback | 60 | 83.570 | 340.438 | 1361.863 |
| Password | Round4 optimized | 60 | 50.464 | 232.518 | 437.874 |
| WeChat fixture | Round3 fallback | 60 | 105.945 | 395.048 | 998.604 |
| WeChat fixture | Round4 optimized | 60 | 41.970 | 193.975 | 621.678 |

Observed calls per successful authority check: **2 Redis GETs remain2**, sequential execution batches/network waits **2→1 pipeline**; logical SQL statements **2→1** for either session type. Each30-message block observed60 checks: old60 ordinary session reads plus60 account/60 credential-or-binding reads; new60 pipeline reads/60 combined SQL checks, with zero old-path calls. Including the identical business account read, a request/reply uses logical SQL **5→3** and Redis batches **4→2**. Counts are method instrumentation plus verified query/library paths, not a packet capture; MySQL prepared-statement protocol frames/physical RTT counts were not measured.

Both aggregate distributions improve; there is **no claim that every batch/tail improved**. The first optimized provider batch P99/max621.678ms exceeds the preceding old batch483.861ms. With N60, P99 is the maximum; shared-host/port-forward jitter is uncontrolled. These are observations of the matched diagnostic, not a production SLO, statistical capacity result or comparison against Round3's historical absolute timings. No retries of the performance run or sample trimming. Raw8 blocks and all samples retained; total35.670s.

## Functional evidence and failed-attempt analysis

- **9 affected unit packages PASS**: authn and4 adapters, Identity, realtime, bootstrap, securityhttp. **4 Race packages PASS**: authn, identityprovider, redissession, Identity. No race report; changed production files pass `gofmt -l`.
- **2 current stock real integration tests PASS**: new single-statement authority test (including7 input subcases, link/unlink, reset, disable and proof conjunction) and current Redis round-trip test with new pipelined account/epoch checks and resurrected-key rejection. Identity's unchanged connection-only overlay scopes destructive test cleanup exclusively to `goal0062_accept_2067`; current Redis test/assertions were not replaced by an older overlay.
- Retained original **post-unlink socket reproduction PASS**. Reused unchanged Round3 lifecycle fixture: **7 state cases PASS** (logout/change/reset/disable-enable/unlink/relink/account epoch), with normal messages, revoked inbound/outbound/idle closure1008, map cleanup and fresh login. Independent sessions/refresh/unsubscribe and real access/server-session expiry also PASS. Controlled provider Exchange and role/audit fixtures retain Round3 limitations. No fresh full game/client campaign.
- **Additional real MySQL failure PASS**: valid/invalid combined proofs, canonical digest/account matching, then close only the fixture SQL client. Optimized validation rejects, existing socket receives no queued payload, closes1008 and clears maps; a new client authenticates the retained UUID and resumes messages. Shared MySQL remains running.
- The same reused suite has **one raw FAIL**, retained in `regression.log`: its `redis-client-unavailable` case closes only the native pool. Round4's managed pipeline remains connected and correctly reads fresh state, so this no longer injects failure into the path under test. No standard was weakened or entire suite relabeled green. Existing bootstrap shuts the Hub before cache resources on application stop.
- A dedicated loopback relay then interrupted **both actual managed/native fixture Redis channels**, without inspecting credentials or stopping shared Redis. Validation/push fail closed, old socket closes1008, maps empty and handshake rejected. First attempt's immediate recovery failed because go-zero's existing breaker remained open; log retained. After inspecting its10s rolling window, one bounded replay waited11s after restoring the relay. **PASS19.068s**, same clients/service recover by fresh login and normal messages at the same UUID. No breaker bypass/reset or product edit. Temporary dependency loss is not asserted to permanently revoke surviving server sessions.

## Reuse, unverified scope and delivery

Round3's unchanged queue/timeout/in-flight/slow-consumer and actual game-bridge evidence is reused only in its original scope. Source comparisons confirm transport/apps/game/clients/schema unchanged; preserved client/game/native history is not a new run. This checkpoint does not qualify large concurrency, pool saturation, sustained resource/leak budgets, in-cluster/production latency, actual provider/device login, OTLP linkage, existing-data migration or candidate deployment. Deferred mail/domain/SMS and other product states remain unchanged.

Private E4=`/root/.config/awesome-zero-platform/acceptance-0062-r4-a739e23` (0700): `units.log`, `race.log`, `integrations.log`, `performance.log`/`timings.json`, `regression.log`, `redis-fault.log` and `redis-recovery.log`; source/overlay/commands and `reuse.json` retained. `performance-overlay.json` selects `TestR4SameDataPerformanceComparison`; `regression-overlay.json` selects the named R2/R3 cases plus `TestR4CombinedAuthorityAndMySQLFailure`; `redis-fault-recovery-overlay.json` selects `TestR4ActualManagedRedisFailureAndRecovery`. Run from `server` with the above bounds, `-tags=integration -overlay=... -run ... -count=1 -v ./apps/app-api/internal/securityhttp`, and private `G62_FIXTURE_CONFIG` pointing to the retained E1 fixture. `run-integrations.py` configures the2 stock integrations without printing credentials. Never substitute an existing business DB.

Final snapshots: **all4 Deployment specs/UIDs, Pod UIDs/restart counts unchanged and services1/1**. No existing business data/schema, deployment, Secret or protection-root change; only isolated fixture accounts/sessions used. Owned dependency forwards/relay/test processes stopped; user **29457 frontend forward remains active**. No environment duplication or rebalance needed. Evidence hashes recorded privately in `result.json` (SHA256 `ab273e1a8a06d065bdbd13464a7742ee9a84dd3aae2e9b820644ecba2694a3c0`).

**Final PASS for Round4 software/query-optimization acceptance.** Report-only delivery; Goal0062 remains `in_progress`, prior Round1/2 FAIL and Round3 PASS preserved. Stop here; no automatic development, migration or deployment follows.
