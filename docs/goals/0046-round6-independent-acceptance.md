# Goal 0046 Round 6 independent acceptance — 2026-11-03

**R5-N05 repair and executed internal regression gates: PASS. Full product/internal qualification: BLOCKED; external-platform qualification: BLOCKED; production release: BLOCKED; sealing NO.** No new blocking product defect was established. Required room/session reclamation, full current native/client qualification and operational/platform evidence remain unproven; no complete internal-product PASS or Goal completion is claimed. Goals0045/0046 remain `in_progress`. This completes an independent acceptance/report assignment, not the product Goals.

## Baseline, ownership and scope

Tested product source **`1ae8ee111f8dbaad2e8312b55a0a1f51603f34da`** on clean synchronized `main`, pulled with `git pull --ff-only`. The report commit is documentation only, explicitly distinct from this tested SHA. Reviewed AGENTS, the four canonical principles, current Goal, original Goal0046 and114 requirements, Round1–5 reports and historical reproducers, relevant actual contracts and complete5-file250-addition/2-deletion repair diff from Round5 report commit4d7b7b7. Historical requirements/reports are preserved. Older principles/Goal paragraphs saying durable recovery is absent are historical direction, not a statement of current source behavior.

Codex did not change product implementation, public protocol, formal tests, schema, CI or deployment configuration. Only this report and permitted current-Goal status/working/verification/completion records are committed. Private probes run through external Go overlays and isolated fixtures. Doudizhu owns journal/authentication/lifecycle semantics, while Dispatcher/bootstrap remain adapters/composition; there is no new platform/business dependency, go-zero component, native-client exception, SQL operation, network call or schema change. The production delta adds a O(1) policy guard before the existing v1 digest check; ordinary v2 HMAC writing, transaction/fencing, bounded replay, checkpoint and receipt paths are unchanged. Full architecture/import/table-boundary suites reran.

`E=/root/.config/awesome-zero-platform/acceptance-0046-r6-1ae8ee1` holds objective/current-before/source.diff, copied scripts but **fresh results**, runs.jsonl, private Go overlays, CI job/step metadata, matrix-round6.json, runtime/fault JSON, logs, supplementary screenshots, summary and SHA256 inventory. Commands record cwd/UTC/exit/duration/source. No prior-round PASS is used as current evidence. Private directories0700/files0600/binaries0700 at closeout; no credentials, tokens, keys, kits, DB contents, binaries or screenshots are committed.

Linux x86_64, ~7.8GiB RAM/no swap; Go1.25.8, Node22.23.4/npm10.9.9, GOMAXPROCS=2; Go `-p 1 -parallel 1`, heavy jobs sequential. Fresh codex0046r6 Docker project/volume: MySQL5.7.44 at1.5CPU/768MiB/64MiB buffer, `innodb_flush_log_at_trx_commit=1`, binlog off; Redis8.8.0 at128MiB. Benchmark SQL pool24, application fixture pool16, scale16workers. Databases awesome_zero_platform (integration), codex0046r6_product (actual product), codex0046r6_upgrade (test-only legacy clone). No production or earlier-round DB/volume was changed.

Actual Go binary, four ordinary server-created/password-authenticated accounts with distinct AccountIDs, three authenticated WSS players through owned Nginx/local HTTPS/WSS and explicitly trusted self-signed SAN certificate. Current original client code performs room1/2/3, leave/rejoin/ready/start, entropy, external private kit backup before commit, new-process import, signed Manifest/AAD/HPKE, future drand Quicknet/BLS,51 disjoint private cards and outsider403. No invented JWT/account/cards/beacon or old HMAC beacon secret. Fault observation uses5-minute bid/play deadlines, unchanged fairness cutoffs. These are local Go identities/TLS, not wx code exchange, approved production domains or mobile qualification.

## R5-N05 independent repair verdict

**Closed at this source within the original defect scope.** Original `TestCodexAuthenticatedJournalDowngradeMustFailClosed`, unchanged from Round5, now passes against real fresh MySQL. It first independently matches the valid stored v2 HMAC, then applies the exact old inconsistency with a legacy checksum and creates a new Coordinator. Recovery now returns integrity refusal. The reused HMAC/multi-field/order suite records26 passing test events with no failure; exact counts are retained below. Newly added developer tests were inspected and rerun, but are not the sole closure evidence.

Source: `server/business/doudizhu/infrastructure/mysqlstore/live_journal.go:318`. A v1 row is rejected if active or retained authentication keys exist, even with clientSeq0. Nil journal also refuses. Explicitly keyless internal environments retain v1/seq0 compatibility; authenticated internal zero-sequence commands continue to write/verify v2. Unknown v0, missing roots, malformed rows and positive public sequence without a configured signing key fail closed. Production config requires a valid256-bit contribution root, and bootstrap unconditionally configures the authenticated keyring before recovery; omitted configuration cannot silently select keyless business recovery.

| Independent check | Current result/evidence |
|---|---|
| Exact Round5 original repro | PASS, hmac-integrity-independent.log, original failure assertion retained |
| Legitimate v2 / internal0 | PASS, new-root/retained-root and real internal signed0 cold receipt checks in serialized-order-root-confirmed.log |
| Explicit keyless legacy compatibility | PASS, TestCodexExplicitKeylessLegacyPolicyInMySQL: actual keyless internal adapter Append/Read preserves v1/seq0 identity and result digest; configuring a root rejects same unsigned data before active/private publication, no auto-retag |
| Original R4-N04 sequence-only91→92 | PASS, sequence-log-integrity.log; valid protected genesis and original request control, new Coordinator refuses changed identity |
| Ordinary integrity fields and ordering | PASS: all12 field/multiple-field/missing/version/tag controls; logical reorder refuses; SQL duplicate constraint rejects; same rows inserted reverse physical order and MySQL JSON key reorder restore correctly |
| Root rotation | PASS for retained-root policy: old/new mixed log authenticates, new writes independently match active root; missing old root refuses actual recovery with seed reader retained to isolate journal behavior |
| Upgrade prerequisite | PASS for fail-closed test controls: old schema refuses/no automatic DDL; explicit additive test-only columns leave seq0/version0/tagNULL; new app refuses unsigned old history rather than guessing identity or authenticating it |

No journal retagging or production migration was performed. Existing signed v2 data is compatible; enabling authenticated recovery over v1/v0 data requires explicit drain/reconciliation. Seed/Checkpoint AES rewrap does not retag old HMAC rows, so an old root remains necessary while its journal/contribution dependencies survive. C09/C10 remain BLOCKED for complete new-only recovery/production retirement inventory, rather than calling narrower AES tests full migration proof.

## Historical regressions and real business flows

| Finding/gate | Current independent result |
|---|---|
| R3-N02 original clocks | PASS under Race: old Bid at4s preserves5s expiry at6s; interleaved phase advance does not replace PLAYING30s timer with BIDDING5s; Play/Pass history at29s preserves30s expiry at31s. Real Service/Dispatcher/Supervisor plus controlled receipt/clock/terminal ports; virtual expiry is not mislabeled SQL persistence. Cache TTL/eviction and fresh-operation controls also pass. |
| Serialized provenance | PASS:16 identical concurrent actual coordinator calls, exactly one fresh/15 Historical, identical payloads/one appended move. Production Dispatcher invokes separate historical reconcilers. |
| SETTLING in-process repair | PASS: real final action/archive durable while controlled SQL failure blocks COMPLETED; remove fault and resend original WSS request without restart, completion reconciles once. Five repeated exact final requests preserve action/archive/command/outbox counts; restart keeps same result. JSON below. |
| R3-N03 | PASS: original wrong sequence refused both hot and first cold request; original then succeeds, no poisoned cache/append. Seven bid/play/pass/checkpoint-prefix/terminal-suffix histories independently kill/restart, changed seq first rejects, full original payload unchanged. Six cold actor/type/score/card/version negatives also pass. |
| F01/F09 | PASS: original signed Manifest/AAD/HPKE clients and actual drand/BLS in all three fair games, no unused HMAC secret. |
| F02/N01 | PASS: actual terminal success deliberately discarded before client, full app SIGKILL, three clients disconnect, completed archive and exact original response restored once. Original score3 bid after PLAYING/restart succeeds, then original harness finishes game. |
| F03/F04 | PASS: twelve immutable Genesis corruption controls; actual completed archive clone/incomplete evidence rejection, seven prune refusal boundaries, actual CLI dry-run/minimum30days and SIGKILL mid-delete rollback. |
| F05/F11 | PASS: actual original room DTO1/2/3 and leave/rejoin; real room.create loss after commit→authoritative reconciliation, and before send→404→same-ID/same locator retry. |
| F06/F07 | PASS: original3-case adversarial client repro plus11-case async lifetime matrix; correct known-rejection/unknown-outcome handling and no secret resurrection after disposal. JS zeroization remains best effort. |
| F08/F10 | PASS: actual production Compose render forwards five required values with nonsecret placeholders; unmodified stock YAML native process reaches ready200 using only documented loopback/port overrides. |
| Active process faults | PASS: actual pre-COMMIT rollback/SIGKILL and receiver-dropped post-COMMIT result/SIGKILL/exact replay, live-action-before-phase-COMMIT kill, epoch takeover and stale-writer controls; no acknowledged duplicate/loss in tested process-loss/storage-intact scope. |
| Concurrency | PASS:16 same-ID requests via3 extra WSS sockets (4/account limit), old history vs new legal move, two distinct IDs at same version yield one commit/one refusal. |

Three fresh fair hands reach actual terminal state/evidence. Original harness action counts exclude actions already executed by probes; direct SQL verifies full count and version. Intentional terminal-loss and SQL-failed-completion child harness exit1 are proven injected controls with expected artifacts/state, not ordinary successful game exits. Cold identity conflicts still map to INTERNAL rather than hot CONFLICT; this existing classification limitation is recorded, not a new private-data bypass or silently repaired API.

Actual SETTLING record:

```json
{"source": "1ae8ee111f8dbaad2e8312b55a0a1f51603f34da", "checks": ["SETTLING old hand.bid exact", "SETTLING old hand.play exact", "SETTLING old hand.pass exact", "SETTLING old hand.pass exact", "five exact final retries in same process: no actions/archive/settlement/outbox duplicates"], "before": {"phase": "SETTLING", "epoch": 3, "actions": 59, "watermark": 60, "commands": 10, "outbox": 14, "archives": 1}, "afterRetry": {"phase": "COMPLETED", "epoch": 3, "actions": 59, "watermark": 60, "commands": 11, "outbox": 15, "archives": 1}, "missingReconciliation": false, "afterRestart": {"phase": "COMPLETED", "epoch": 4, "actions": 59, "watermark": 60, "commands": 11, "outbox": 15, "archives": 1}, "result": "passed"}
```

## CI, automated gates and build qualification

Exact source [CI41687865374](https://github.com/oublie6/awesome-zero-platform/actions/runs/41687865374) and [Fair Doudizhu41687865383](https://github.com/oublie6/awesome-zero-platform/actions/runs/41687865383) completed success; all10 actual jobs/steps inspected: secure_envelope,admin_web,unit,integration,runtime,report,mysql_integration,application,manifest_client,client_foundation. No exact-source Creator run/artifact exists. Green CI is separate from these independent reruns.

Go cwd=server, other commands cwd=repository root; run.py sets stated toolchain/GOMAXPROCS. Counts include parent/subtests and overlap, not additive unique tests. The TS interoperability vector initially skipped by general Go tests is generated and actually rerun separately in crypto-interop. Empty packages are not failures. Current test-event counts:

`{'go-unit': {'pass': 554, 'skip': 1}, 'go-race': {'pass': 554, 'skip': 1}, 'mysql-integration-scale': {'pass': 591, 'skip': 1}, 'hmac-integrity-independent': {'pass': 26}, 'original-clock-races': {'pass': 9}, 'serialized-order-root-confirmed': {'pass': 9}, 'original-mysql-regressions': {'pass': 42}}`

| Gate | Exact command (E is private evidence dir) | Exit / seconds / log |
|---|---|---|
| client-foundation | `bash scripts/verify-client-foundation.sh` | 0 / 30.244 / client-foundation.log |
| go-unit | `go test -json -count=1 -p 1 -parallel 1 ./...` | 0 / 52.743 / go-unit.log |
| go-race | `go test -json -race -count=1 -p 1 -parallel 1 ./...` | 0 / 124.116 / go-race.log |
| go-vet | `go vet -p 1 ./...` | 0 / 7.65 / go-vet.log |
| app-build | `env CGO_ENABLED=0 go build -p 1 -trimpath -ldflags=-s -o E/app-api ./apps/app-api` | 0 / 7.635 / app-build.log |
| maintenance-build | `env CGO_ENABLED=0 go build -p 1 -trimpath -o E/doudizhu-maintenance ./apps/doudizhu-maintenance` | 0 / 2.854 / maintenance-build.log |
| mysql-integration-scale | `env APP_API_INTEGRATION=1 DOUDIZHU_SCALE_PROFILE=1 go test -json -count=1 -p 1 -parallel 1 -tags=integration ./...` | 0 / 87.97 / mysql-integration-scale.log |
| original-clock-races | `go test -json -race -count=1 -p 1 -parallel 1 -overlay E/overlay.json -run ^TestCodex(Historical\|NewBid) ./apps/app-api/internal/doudizhuapi` | 0 / 5.302 / original-clock-races.log |
| cache-clock-extended | `go test -json -race -count=1 -p 1 -parallel 1 -overlay E/overlay.json -run ^TestCodexHistoricalCacheMiss ./apps/app-api/internal/doudizhuapi` | 0 / 5.801 / cache-clock-extended.log |
| serialized-order-root-confirmed | `env APP_API_INTEGRATION=1 CODEX_EVIDENCE=E go test -json -race -count=1 -p 1 -parallel 1 -tags=integration -overlay E/overlay.json -run ^TestCodex(Serialized\|AuthenticatedJournalLogical\|JournalPhysical\|MissingJournalRoot\|RealInternalZero\|RewrappedSeed\|ExplicitKeylessLegacy) ./business/doudizhu/infrastructure/mysqlstore` | 0 / 11.475 / serialized-order-root-confirmed.log |
| original-mysql-regressions | `go test -json -count=1 -p 1 -parallel 1 -tags=integration -overlay E/overlay.json -run ^TestCodex(Completed\|IncompleteArchive\|Genesis\|Rotation\|Keyring\|Candidate\|RealPrune\|Provider\|StockEnvironment) ./business/doudizhu/infrastructure/mysqlstore ./business/doudizhu/domain/livehand ./apps/app-api/internal/config` | 0 / 9.688 / original-mysql-regressions.log |
| sequence-log-integrity | `go test -json -count=1 -p 1 -parallel 1 -tags=integration -overlay E/overlay.json -run ^TestCodexClientSequenceLogTamper ./business/doudizhu/infrastructure/mysqlstore` | 0 / 2.581 / sequence-log-integrity.log |
| receipt-scale-profile | `go test -json -count=1 -p 1 -parallel 1 -tags=integration -overlay E/overlay.json -run ^TestCodexReceiptScaleProfile ./business/doudizhu/infrastructure/mysqlstore` | 0 / 71.604 / receipt-scale-profile.log |
| original-client-repro | `node --test E/client-adversarial.test.mjs` | 0 / 0.503 / original-client-repro.log |
| client-lifetime-matrix | `node --test E/client-lifetime-matrix.test.mjs` | 0 / 2.732 / client-lifetime-matrix.log |
| hpke | `npm --prefix clients/packages/secure-envelope test` | 0 / 3.275 / hpke.log |
| crypto-interop | `python3 E/crypto-regressions.py` | 0 / 7.658 / crypto-interop.log |
| fmt | `make fmt-check` | 0 / 0.3 / fmt.log |
| api-contract | `make api-contract-check` | 0 / 0.052 / api-contract.log |
| compose-switch | `bash scripts/test-production-compose.sh` | 0 / 0.466 / compose-switch.log |
| deploy-wiring | `python3 E/deploy-wiring.py` | 0 / 0.313 / deploy-wiring.log |
| production-compose-runtime | `env DOCKER_BUILDKIT=0 bash E/runtime-isolated.sh` | 0 / 292.526 / production-compose-runtime.log |
| legacy-clone | `python3 E/clone-legacy.py` | 0 / 6.989 / legacy-clone.log |
| legacy-upgrade | `python3 E/legacy-upgrade.py` | 0 / 0.87 / legacy-upgrade.log |
| game1-complete | `python3 E/run-harness.py` | 0 / 2.862 / game1-complete.log |
| active-lifecycle-matrix | `node E/active-lifecycle-matrix.mjs` | 0 / 13.515 / active-lifecycle-matrix.log |
| terminal-recovery | `node E/terminal-recovery.mjs` | 0 / 1.823 / terminal-recovery.log |
| completed-history | `node E/completed-history.mjs` | 0 / 3.514 / completed-history.log |
| restart-binding-negatives | `node E/restart-binding-negatives.mjs` | 0 / 20.878 / restart-binding-negatives.log |
| all-types-sequence-restart | `node E/all-types-sequence-restart.mjs` | 0 / 17.396 / all-types-sequence-restart.log |
| active-process-fault | `node E/active-process-fault-record.mjs` | 0 / 7.425 / active-process-fault.log |
| settling-without-restart | `node E/settling-without-restart.mjs` | 0 / 12.763 / settling-without-restart.log |
| room-after-send-loss | `env NODE_EXTRA_CA_CERTS=E/fixture-cert.pem node E/room-unknown.mjs` | 0 / 15.614 / room-after-send-loss.log |
| room-before-send-loss | `env NODE_EXTRA_CA_CERTS=E/fixture-cert.pem node E/room-before-send-loss.mjs` | 0 / 0.667 / room-before-send-loss.log |
| default-yaml-start | `python3 E/default-start.py` | 0 / 1.284 / default-yaml-start.log |
| prune-cli-atomicity | `python3 E/prune-process.py` | 0 / 7.839 / prune-cli-atomicity.log |
| hmac-integrity-independent | `env APP_API_INTEGRATION=1 CODEX_EVIDENCE=E go test -json -count=1 -p 1 -parallel 1 -tags=integration -overlay E/overlay.json -run ^TestCodex(HMAC\|AuthenticatedJournal\|NoKey\|Sequence) ./business/doudizhu/infrastructure/mysqlstore` | 0 / 7.98 / hmac-integrity-independent.log |
| socket-lifetime | `env NODE_EXTRA_CA_CERTS=E/fixture-cert.pem node E/socket-lifetime.mjs` | 0 / 29.889 / socket-lifetime.log |
| full-three-games | `python3 E/games.py` | 0 / 285.463 / full-three-games.log |
| game1-commit | `node E/prepare-game.mjs commit` | 0 / 4.83 / game1-commit.log |
| game1-reveal | `node E/prepare-game.mjs reveal` | 0 / 64.45 / game1-reveal.log |
| game2-commit | `node E/prepare-game.mjs commit-reuse` | 0 / 2.751 / game2-commit.log |
| game2-reveal | `node E/prepare-game.mjs reveal` | 0 / 61.311 / game2-reveal.log |
| game3-commit | `node E/prepare-game.mjs commit-reuse` | 0 / 2.502 / game3-commit.log |
| game3-reveal | `node E/prepare-game.mjs reveal` | 0 / 64.257 / game3-reveal.log |

Exact-source client source types/build/boundaries102tests, original faults3/3, async lifetime11/11 and HPKE9 rerun. Supplementary actual Chromium smoke uses retained official Creator3.8.8 artifact source06733a93d614b68f897a796b9eb7684a79a4dbf9; verified unchanged client/script/workflow inputs, but this is explicitly not a new1ae8ee1 native build or current rendered complete-game/wx qualification. Three viewport/disposal/browser checks and manual portrait inspection are recorded; old home debug HUD/small text remain. Local qualified Creator and workflow-dispatch capability are unavailable. Existing operator AppID/logged-in community Linux WeChat tool do exist; this does not provide backend AppSecret/domains or three handset qualification. H02–H08/AUTO10 remain BLOCKED, no old-SHA native PASS inherited, no upload/preview/release.

Production runtime builds original repository Go/Admin Dockerfiles and executes all original bootstrap/auth/authz/HTTP/WS/HTTPS/WSS assertions. Private fixture changes only project/output/loopback ports and build network=host for host container-DNS compatibility, preserving original build/assertions. It cleans its own containers/network/volume and leaves unrelated host8080 services alone. Repository deployment settings are unchanged. This is controlled local deployment evidence, not production/Kubernetes release approval.

## Performance, resource and lifetime evidence

Stated VM/fsync/container/pool conditions above; measurements are local runner baselines, not production SLA or a controlled A/B benchmark of the policy guard. New guard adds no SQL/crypto operation to normal v2 handling; aggregate timings cannot establish a causal speedup. P99 tails and maximums remain part of the evidence, not replaced by averages. Total reconstruction time is not per-hand P95.

```text
live_journal_perf_integration_test.go:94: MySQL COMMIT-before-ACK sample=80 p50=8.596531ms p95=18.087165ms p99=71.582779ms max=98.207033ms (CI runner, not production SLA)
live_scale_integration_test.go:39: MySQL live profile hands=100 setup=905.634342ms recovered=1.240606448s heapDelta=299960 bytes commit p50=17.31661ms p95=31.124833ms p99=34.691374ms max=35.055063ms; workers=16 (CI/hardware dependent)
live_scale_integration_test.go:39: MySQL live profile hands=1000 setup=8.617113996s recovered=14.678657051s heapDelta=3034112 bytes commit p50=15.58197ms p95=53.523449ms p99=62.353937ms max=82.265351ms; workers=16 (CI/hardware dependent)
codex_receipt_profile_test.go:19: hands=100 receipt=bid allocationsPerQuery=57
codex_receipt_profile_test.go:19: hands=100 receipt=play allocationsPerQuery=78
codex_receipt_profile_test.go:19: hands=100 receipt=pass allocationsPerQuery=60
codex_receipt_profile_test.go:16: 1000-new-bid samples=1000 p50=7.564813ms p95=13.46619ms p99=27.425357ms max=395.154428ms
codex_receipt_profile_test.go:16: 1000-new-play samples=1000 p50=7.706177ms p95=15.249675ms p99=27.821071ms max=81.340611ms
codex_receipt_profile_test.go:16: 1000-new-pass samples=1000 p50=7.634536ms p95=13.778752ms p99=35.731242ms max=471.043077ms
codex_receipt_profile_test.go:17: hands=1000 recovery=13.233405076s postGCHeapDelta=6929048
codex_receipt_profile_test.go:19: 1000-receipt-bid samples=1000 p50=37.569µs p95=58.499µs p99=91.942µs max=191.182µs
codex_receipt_profile_test.go:19: hands=1000 receipt=bid allocationsPerQuery=57
codex_receipt_profile_test.go:19: 1000-receipt-play samples=1000 p50=29.177µs p95=128.07µs p99=214.109µs max=337.812µs
codex_receipt_profile_test.go:19: hands=1000 receipt=play allocationsPerQuery=78
codex_receipt_profile_test.go:19: 1000-receipt-pass samples=1000 p50=44.663µs p95=69.239µs p99=154.696µs max=16.308189ms
codex_receipt_profile_test.go:19: hands=1000 receipt=pass allocationsPerQuery=60
codex_receipt_profile_test.go:20: 1000-contended-same-id-16workers samples=1600 p50=35.281µs p95=1.616572ms p99=6.759198ms max=13.553794ms
codex_receipt_profile_test.go:20: hands=1000 contendedThroughput=23214 queries/s elapsed=68.924143ms goroutineDelta=0
codex_clone_profile_test.go:58: candidate clone prefix=0 n=1000 p50=4.722µs p95=11.738µs p99=22.251µs allocatedPerClone=2416 postGCHeapDelta=-29664 goroutineDelta=0; domain fixture only, no production SLA
codex_clone_profile_test.go:58: candidate clone prefix=58 n=1000 p50=22.831µs p95=44.55µs p99=156.2µs allocatedPerClone=13416 postGCHeapDelta=-38528 goroutineDelta=0; domain fixture only, no production SLA
```

Receipt performance uses authenticated-journal internal zero-sequence ports; positive public identities are independently verified over actual WSS/MySQL.16-worker1600query controls leave3*N durable rows unchanged. Clone prefix0/58, allocations, post-GC heap and goroutine deltas rerun. Five in-process terminal retries add no side effects, but no isolated microsecond CPU cost is asserted.

Additional real socket lifetime probe:60 warmup connections, then1500 authenticated WSS open/close operations with three identities, actual server Prometheus sampling and15second quiescence. Final active sockets0, exact1500 accepted delta and goroutines within the predeclared+4 background allowance; game persistence unchanged. This discharges G08 within a bounded real-socket/query test scope; it is not an indefinite/device/WAN soak. Heap/RSS are observations without forced GC and not a claim of full room/game-history reclamation; G02/GOAL7 remain BLOCKED for that broader lifetime requirement.

```json
{"source": "1ae8ee111f8dbaad2e8312b55a0a1f51603f34da", "scope": "bounded real authenticated WSS open/close lifetime probe; no product instrumentation or forced GC", "samples": [{"completedConnections": 150, "goroutines": 26, "heapAlloc": 3441064, "heapInUse": 6807552, "rss": 32018432, "gcCycles": 12, "activeSockets": 0, "accepted": 213}, {"completedConnections": 300, "goroutines": 26, "heapAlloc": 4574712, "heapInUse": 7790592, "rss": 32034816, "gcCycles": 14, "activeSockets": 0, "accepted": 363}, {"completedConnections": 450, "goroutines": 26, "heapAlloc": 3871600, "heapInUse": 7438336, "rss": 33316864, "gcCycles": 17, "activeSockets": 0, "accepted": 513}, {"completedConnections": 600, "goroutines": 26, "heapAlloc": 5233424, "heapInUse": 8822784, "rss": 34516992, "gcCycles": 19, "activeSockets": 0, "accepted": 663}, {"completedConnections": 750, "goroutines": 26, "heapAlloc": 5424680, "heapInUse": 8765440, "rss": 34394112, "gcCycles": 21, "activeSockets": 0, "accepted": 813}, {"completedConnections": 900, "goroutines": 26, "heapAlloc": 5426576, "heapInUse": 8929280, "rss": 34775040, "gcCycles": 23, "activeSockets": 0, "accepted": 963}, {"completedConnections": 1050, "goroutines": 26, "heapAlloc": 5263024, "heapInUse": 8855552, "rss": 34746368, "gcCycles": 25, "activeSockets": 0, "accepted": 1113}, {"completedConnections": 1200, "goroutines": 26, "heapAlloc": 5157464, "heapInUse": 8740864, "rss": 35823616, "gcCycles": 27, "activeSockets": 0, "accepted": 1263}, {"completedConnections": 1350, "goroutines": 26, "heapAlloc": 5391656, "heapInUse": 9142272, "rss": 35123200, "gcCycles": 29, "activeSockets": 0, "accepted": 1413}, {"completedConnections": 1500, "goroutines": 26, "heapAlloc": 5421328, "heapInUse": 8798208, "rss": 35549184, "gcCycles": 31, "activeSockets": 0, "accepted": 1563}], "before": {"goroutines": 26, "heapAlloc": 4288408, "heapInUse": 7454720, "rss": 30433280, "gcCycles": 8, "activeSockets": 0, "accepted": 63}, "after": {"goroutines": 26, "heapAlloc": 4947904, "heapInUse": 7766016, "rss": 35217408, "gcCycles": 32, "activeSockets": 0, "accepted": 1563}, "durationMs": 28718, "measuredConnections": 1500, "warmupConnections": 60, "goroutineDelta": 0, "heapDelta": 659496, "rssDelta": 4784128, "result": "passed"}
```

## Complete114 requirements and Round5 changes

{'PASS': 97, 'FAIL': 0, 'BLOCKED': 17}. All original IDs/descriptions retained. No previous PASS directly inherited; current gate exits/runtime artifacts were required before generating this matrix. Audit rows certify audit execution, not absence of every possible defect.

All previous4FAIL/18BLOCKED individually reassessed:

| ID | Round5 → Round6 | Current basis |
|---|---|---|
| B07 | FAIL → **PASS** | Original R5-N05 downgrade and all12 field/multi-field/order/missing/canonical JSON controls pass; keyed v1 refused before hand/receipt/private publication |
| B15 | FAIL → **PASS** | Real HMAC journal/old-root/new-root verification and original91→92 + R5-N05 downgrade fail closed; explicitly keyless compatibility separately verified |
| C09 | BLOCKED → **BLOCKED** | AES Seed/Checkpoint new-only reads pass after rewrap, but full new-only recovery remains unavailable while old-root Journal HMAC rows survive; no journal retagging/retirement qualification |
| C10 | BLOCKED → **BLOCKED** | Old roots explicitly retained for Seed/Checkpoint/contributions/Journal HMAC; premature removal refuses recovery. Complete production inventory/KMS/approved retirement unavailable |
| E03 | BLOCKED → **BLOCKED** | Code controls and explicit private backup-before-commit pass; cannot establish human backup confirmation/device UX from Node harness |
| G02 | BLOCKED → **BLOCKED** | Bounded post-GC heap/copy/1000-hand measurements executed; long room/socket lifetime reclamation not proved |
| G08 | BLOCKED → **PASS** | Actual1500 WSS open/close connections after60 warmup, server Prometheus goroutines/sockets return to baseline within stated allowance; private query/clone controls also rerun; bounded duration only |
| H02 | BLOCKED → **BLOCKED** | No exact-source native Creator artifact; actual current wx SDK/full rendered UI/devices unqualified; older artifact rerender only supplementary |
| H03 | BLOCKED → **BLOCKED** | Old06733a9 Creator artifact rerender passed at three viewports/disposal; manually inspected portrait. It is not an exact1ae8ee1 scene/build pass |
| H04 | BLOCKED → **BLOCKED** | No exact-source native Creator artifact; actual current wx SDK/full rendered UI/devices unqualified; older artifact rerender only supplementary |
| H05 | BLOCKED → **BLOCKED** | No exact-source native Creator artifact; actual current wx SDK/full rendered UI/devices unqualified; older artifact rerender only supplementary |
| H06 | BLOCKED → **BLOCKED** | Current source GameOnlineController↔real Go room/fairness flows pass; current rendered Cocos/wx login-through-result flow unqualified |
| H07 | BLOCKED → **BLOCKED** | Current controller kit import/reconnect and room loss flows pass; actual current SDK/device UI reachability/recovery unqualified |
| H08 | BLOCKED → **BLOCKED** | Supplementary older home view inspected, debug HUD/small portrait text remain; actual current playable-table safe area/layout/device refresh unqualified |
| AUTO10 | BLOCKED → **BLOCKED** | Exact-SHA CI and Fair Doudizhu pass; no Creator workflow/artifact at1ae8ee1, no qualified local Creator/dispatch capability |
| GOAL1 | BLOCKED → **BLOCKED** | Real Go password sessions/isolation pass; server-side wx AppSecret/code exchange/approved domains/device evidence unavailable |
| GOAL2 | BLOCKED → **BLOCKED** | Actual Go rooms1/2/3, leave/rejoin/ready/start and both before/after-send loss branches pass; full current Cocos/platform UI unqualified |
| GOAL4 | BLOCKED → **BLOCKED** | Backend complete games and original lifecycle/settlement defects now pass; full exact-candidate rendered Cocos/platform UI through score/evidence remains unqualified |
| GOAL5 | FAIL → **PASS** | Current full MySQL/fencing/checkpoint/replay, original R4-N04/R5-N05 and actual WSS pre/post-COMMIT/terminal recovery controls pass; stated process-loss/storage-intact scope |
| GOAL6 | FAIL → **PASS** | Three real fair WSS games, phase-gap/pre/post-COMMIT SIGKILL, same-ID concurrency/response loss/original replay and corruption refusals pass |
| GOAL7 | BLOCKED → **BLOCKED** | 100/1000 latency/heap/concurrency measured; sustained room/socket soak, production storage/hardware/WAN evidence unavailable |
| GOAL8 | BLOCKED → **BLOCKED** | Exact-SHA native packages/current SDK gameplay/approved domains/real wx handsets and Douyin qualification unavailable |

| ID | Original requirement | Round6 | Current evidence / limit |
|---|---|---|---|
| A01 | 真实身份认证和账号隔离 | **PASS** | game1/2/3 commit+reveal, original-N01, complete games and active-lifecycle-matrix: real Go accounts/rooms/HPKE/drand/WSS; platform/device UI separately blocked |
| A02 | 创建、加入、退出房间 | **PASS** | game1/2/3 commit+reveal, original-N01, complete games and active-lifecycle-matrix: real Go accounts/rooms/HPKE/drand/WSS; platform/device UI separately blocked |
| A03 | 三人准备和开局 | **PASS** | game1/2/3 commit+reveal, original-N01, complete games and active-lifecycle-matrix: real Go accounts/rooms/HPKE/drand/WSS; platform/device UI separately blocked |
| A04 | 公平承诺与揭示 | **PASS** | game1/2/3 commit+reveal, original-N01, complete games and active-lifecycle-matrix: real Go accounts/rooms/HPKE/drand/WSS; platform/device UI separately blocked |
| A05 | 公共随机源验证及发牌 | **PASS** | game1/2/3 commit+reveal, original-N01, complete games and active-lifecycle-matrix: real Go accounts/rooms/HPKE/drand/WSS; platform/device UI separately blocked |
| A06 | 地主竞叫及确定 | **PASS** | game1/2/3 commit+reveal, original-N01, complete games and active-lifecycle-matrix: real Go accounts/rooms/HPKE/drand/WSS; platform/device UI separately blocked |
| A07 | 出牌、过牌、牌型比较 | **PASS** | game1/2/3 commit+reveal, original-N01, complete games and active-lifecycle-matrix: real Go accounts/rooms/HPKE/drand/WSS; platform/device UI separately blocked |
| A08 | 回合切换 | **PASS** | game1/2/3 commit+reveal, original-N01, complete games and active-lifecycle-matrix: real Go accounts/rooms/HPKE/drand/WSS; platform/device UI separately blocked |
| A09 | 终局、计分及证据归档 | **PASS** | game1/2/3 commit+reveal, original-N01, complete games and active-lifecycle-matrix: real Go accounts/rooms/HPKE/drand/WSS; platform/device UI separately blocked |
| A10 | 断线重连及权威状态同步 | **PASS** | game1/2/3 commit+reveal, original-N01, complete games and active-lifecycle-matrix: real Go accounts/rooms/HPKE/drand/WSS; platform/device UI separately blocked |
| B01 | 已向客户端确认的操作必须先完成持久化提交。 | **PASS** | full MySQL integration+Race, original-mysql-regressions, terminal-recovery, active-process-fault, all-types-sequence-restart; stated process-fault scope |
| B02 | 任何已确认操作不能在进程崩溃后丢失。 | **PASS** | full MySQL integration+Race, original-mysql-regressions, terminal-recovery, active-process-fault, all-types-sequence-restart; stated process-fault scope |
| B03 | 同一 requestId 的重复请求不能重复执行。 | **PASS** | original clocks+cache TTL/eviction pass; exactly 1 fresh/15 Historical serialized duplicates; actual WSS16 same-ID requests/no extra writes; SETTLING reconciles once |
| B04 | 相同 requestId、不同参数或不同操作者必须拒绝。 | **PASS** | full MySQL integration+Race, original-mysql-regressions, terminal-recovery, active-process-fault, all-types-sequence-restart; stated process-fault scope |
| B05 | 新服务接管后，旧 Owner Epoch 必须被正确隔离。 | **PASS** | full MySQL integration+Race, original-mysql-regressions, terminal-recovery, active-process-fault, all-types-sequence-restart; stated process-fault scope |
| B06 | 检查点、日志、水位线和业务版本必须严格一致。 | **PASS** | full MySQL integration+Race, original-mysql-regressions, terminal-recovery, active-process-fault, all-types-sequence-restart; stated process-fault scope |
| B07 | 检查点或日志被篡改时必须拒绝恢复。 | **PASS** | Original R5-N05 downgrade and all12 field/multi-field/order/missing/canonical JSON controls pass; keyed v1 refused before hand/receipt/private publication |
| B08 | 恢复不能泄露其他玩家的私有手牌。 | **PASS** | full MySQL integration+Race, original-mysql-regressions, terminal-recovery, active-process-fault, all-types-sequence-restart; stated process-fault scope |
| B09 | 终局重启必须保留最终证据和历史回执。 | **PASS** | full MySQL integration+Race, original-mysql-regressions, terminal-recovery, active-process-fault, all-types-sequence-restart; stated process-fault scope |
| B10 | MySQL COMMIT 前后发生 SIGKILL、网络断开、响应丢失时，不得出现重复执行或状态分叉。 | **PASS** | full MySQL integration+Race, original-mysql-regressions, terminal-recovery, active-process-fault, all-types-sequence-restart; stated process-fault scope |
| B11 | 真实 MySQL 事务 | **PASS** | full MySQL integration+Race, original-mysql-regressions, terminal-recovery, active-process-fault, all-types-sequence-restart; stated process-fault scope |
| B12 | 并发重复提交 | **PASS** | full MySQL integration+Race, original-mysql-regressions, terminal-recovery, active-process-fault, all-types-sequence-restart; stated process-fault scope |
| B13 | Epoch fencing | **PASS** | full MySQL integration+Race, original-mysql-regressions, terminal-recovery, active-process-fault, all-types-sequence-restart; stated process-fault scope |
| B14 | 检查点加密 | **PASS** | full MySQL integration+Race, original-mysql-regressions, terminal-recovery, active-process-fault, all-types-sequence-restart; stated process-fault scope |
| B15 | 日志完整性 | **PASS** | Real HMAC journal/old-root/new-root verification and original91→92 + R5-N05 downgrade fail closed; explicitly keyless compatibility separately verified |
| B16 | 三人完整对局恢复 | **PASS** | full MySQL integration+Race, original-mysql-regressions, terminal-recovery, active-process-fault, all-types-sequence-restart; stated process-fault scope |
| B17 | 进程级故障注入 | **PASS** | full MySQL integration+Race, original-mysql-regressions, terminal-recovery, active-process-fault, all-types-sequence-restart; stated process-fault scope |
| B18 | 终局证据恢复 | **PASS** | full MySQL integration+Race, original-mysql-regressions, terminal-recovery, active-process-fault, all-types-sequence-restart; stated process-fault scope |
| C01 | 新记录使用当前主密钥加密 | **PASS** | original-mysql-regressions: AES seed/checkpoint rewrap, partial-transaction SIGKILL, key bounds; HMAC root controls in hmac-integrity-independent/serialized-order-root-confirmed |
| C02 | 旧记录可以使用保留的历史密钥读取 | **PASS** | original-mysql-regressions: AES seed/checkpoint rewrap, partial-transaction SIGKILL, key bounds; HMAC root controls in hmac-integrity-independent/serialized-order-root-confirmed |
| C03 | 历史密钥数量和长度严格受限 | **PASS** | original-mysql-regressions: AES seed/checkpoint rewrap, partial-transaction SIGKILL, key bounds; HMAC root controls in hmac-integrity-independent/serialized-order-root-confirmed |
| C04 | 旧密钥不存在时拒绝错误解密 | **PASS** | original-mysql-regressions: AES seed/checkpoint rewrap, partial-transaction SIGKILL, key bounds; HMAC root controls in hmac-integrity-independent/serialized-order-root-confirmed |
| C05 | 重加密必须先认证旧密文 | **PASS** | original-mysql-regressions: AES seed/checkpoint rewrap, partial-transaction SIGKILL, key bounds; HMAC root controls in hmac-integrity-independent/serialized-order-root-confirmed |
| C06 | 种子重加密不得改变原始随机贡献 | **PASS** | original-mysql-regressions: AES seed/checkpoint rewrap, partial-transaction SIGKILL, key bounds; HMAC root controls in hmac-integrity-independent/serialized-order-root-confirmed |
| C07 | 检查点重加密必须校验当前 Owner Epoch | **PASS** | original-mysql-regressions: AES seed/checkpoint rewrap, partial-transaction SIGKILL, key bounds; HMAC root controls in hmac-integrity-independent/serialized-order-root-confirmed |
| C08 | 错误密钥、篡改密文不能恢复 | **PASS** | original-mysql-regressions: AES seed/checkpoint rewrap, partial-transaction SIGKILL, key bounds; HMAC root controls in hmac-integrity-independent/serialized-order-root-confirmed |
| C09 | 重加密后仅保留新密钥仍可正确恢复 | **BLOCKED** | AES Seed/Checkpoint new-only reads pass after rewrap, but full new-only recovery remains unavailable while old-root Journal HMAC rows survive; no journal retagging/retirement qualification |
| C10 | 旧密钥未完成迁移前不得被错误淘汰 | **BLOCKED** | Old roots explicitly retained for Seed/Checkpoint/contributions/Journal HMAC; premature removal refuses recovery. Complete production inventory/KMS/approved retirement unavailable |
| D01 | 活跃牌局不能删除 | **PASS** | original-mysql-regressions + prune-cli-atomicity: real final archive, semantic corruption/7 refusal boundaries/30-day minimum/actual CLI SIGKILL rollback |
| D02 | 未归档牌局不能删除 | **PASS** | original-mysql-regressions + prune-cli-atomicity: real final archive, semantic corruption/7 refusal boundaries/30-day minimum/actual CLI SIGKILL rollback |
| D03 | 不满足 30 天保留期不能删除 | **PASS** | original-mysql-regressions + prune-cli-atomicity: real final archive, semantic corruption/7 refusal boundaries/30-day minimum/actual CLI SIGKILL rollback |
| D04 | 损坏或不完整的归档不能删除 | **PASS** | original-mysql-regressions + prune-cli-atomicity: real final archive, semantic corruption/7 refusal boundaries/30-day minimum/actual CLI SIGKILL rollback |
| D05 | 最终版本和日志水位线不一致不能删除 | **PASS** | original-mysql-regressions + prune-cli-atomicity: real final archive, semantic corruption/7 refusal boundaries/30-day minimum/actual CLI SIGKILL rollback |
| D06 | 历史日志数量不完整不能删除 | **PASS** | original-mysql-regressions + prune-cli-atomicity: real final archive, semantic corruption/7 refusal boundaries/30-day minimum/actual CLI SIGKILL rollback |
| D07 | 不得直接越权读取其他业务模块数据库表 | **PASS** | original-mysql-regressions + prune-cli-atomicity: real final archive, semantic corruption/7 refusal boundaries/30-day minimum/actual CLI SIGKILL rollback |
| D08 | 保留完整终局证据和必要审计信息 | **PASS** | original-mysql-regressions + prune-cli-atomicity: real final archive, semantic corruption/7 refusal boundaries/30-day minimum/actual CLI SIGKILL rollback |
| D09 | 默认 dry-run 不得修改数据库 | **PASS** | original-mysql-regressions + prune-cli-atomicity: real final archive, semantic corruption/7 refusal boundaries/30-day minimum/actual CLI SIGKILL rollback |
| D10 | 必须显式 `--execute` 才能执行清理 | **PASS** | original-mysql-regressions + prune-cli-atomicity: real final archive, semantic corruption/7 refusal boundaries/30-day minimum/actual CLI SIGKILL rollback |
| D11 | 中途崩溃后不能出现违反一致性约束的部分删除 | **PASS** | original-mysql-regressions + prune-cli-atomicity: real final archive, semantic corruption/7 refusal boundaries/30-day minimum/actual CLI SIGKILL rollback |
| E01 | 提交承诺前生成高熵随机贡献 | **PASS** | all three actual fair hands: external kit before commit, new-process import; original-client-repro/client-lifetime-matrix, best-effort zeroization; excludes human/device proof |
| E02 | 恢复凭证包含完整且可验证的信息 | **PASS** | all three actual fair hands: external kit before commit, new-process import; original-client-repro/client-lifetime-matrix, best-effort zeroization; excludes human/device proof |
| E03 | 用户备份后才允许提交承诺 | **BLOCKED** | Code controls and explicit private backup-before-commit pass; cannot establish human backup confirmation/device UX from Node harness |
| E04 | 应用退出后可以导入恢复凭证 | **PASS** | all three actual fair hands: external kit before commit, new-process import; original-client-repro/client-lifetime-matrix, best-effort zeroization; excludes human/device proof |
| E05 | 恢复凭证与当前登录身份、Hand、Seat、Commitment 必须匹配 | **PASS** | all three actual fair hands: external kit before commit, new-process import; original-client-repro/client-lifetime-matrix, best-effort zeroization; excludes human/device proof |
| E06 | 错误凭证必须拒绝 | **PASS** | all three actual fair hands: external kit before commit, new-process import; original-client-repro/client-lifetime-matrix, best-effort zeroization; excludes human/device proof |
| E07 | 不能产生新的随机贡献冒充旧承诺 | **PASS** | all three actual fair hands: external kit before commit, new-process import; original-client-repro/client-lifetime-matrix, best-effort zeroization; excludes human/device proof |
| E08 | 不能通过普通小游戏明文持久化自动泄漏随机数 | **PASS** | all three actual fair hands: external kit before commit, new-process import; original-client-repro/client-lifetime-matrix, best-effort zeroization; excludes human/device proof |
| E09 | 释放对象时敏感数据应尽可能清零 | **PASS** | all three actual fair hands: external kit before commit, new-process import; original-client-repro/client-lifetime-matrix, best-effort zeroization; excludes human/device proof |
| E10 | 请求丢响应时必须保留原 requestId 精确重试 | **PASS** | all three actual fair hands: external kit before commit, new-process import; original-client-repro/client-lifetime-matrix, best-effort zeroization; excludes human/device proof |
| E11 | 不能因为未知结果直接生成新的操作请求 | **PASS** | all three actual fair hands: external kit before commit, new-process import; original-client-repro/client-lifetime-matrix, best-effort zeroization; excludes human/device proof |
| F01 | 验证使用三个真实、不同的服务端认证身份 | **PASS** | three real Go identities/three distinct seats/HTTPS/WSS, actual signed Manifest/AAD/HPKE/drand/private outsider403/full games/evidence; not three wx devices |
| F02 | 验证三人座位各不相同 | **PASS** | three real Go identities/three distinct seats/HTTPS/WSS, actual signed Manifest/AAD/HPKE/drand/private outsider403/full games/evidence; not three wx devices |
| F03 | 验证每次操作使用真实 WSS | **PASS** | three real Go identities/three distinct seats/HTTPS/WSS, actual signed Manifest/AAD/HPKE/drand/private outsider403/full games/evidence; not three wx devices |
| F04 | 验证客户端私有手牌访问权限 | **PASS** | three real Go identities/three distinct seats/HTTPS/WSS, actual signed Manifest/AAD/HPKE/drand/private outsider403/full games/evidence; not three wx devices |
| F05 | 验证游戏完整进行到终局 | **PASS** | three real Go identities/three distinct seats/HTTPS/WSS, actual signed Manifest/AAD/HPKE/drand/private outsider403/full games/evidence; not three wx devices |
| F06 | 验证最终结算及证据 | **PASS** | three real Go identities/three distinct seats/HTTPS/WSS, actual signed Manifest/AAD/HPKE/drand/private outsider403/full games/evidence; not three wx devices |
| G01 | 并发牌局状态隔离 | **PASS** | 100/1000-hand MySQL profile, receipt-scale-profile, Race, clone/query/allocation controls; bounded runner measurements, not production SLA |
| G02 | 内存增长与回收 | **BLOCKED** | Bounded post-GC heap/copy/1000-hand measurements executed; long room/socket lifetime reclamation not proved |
| G03 | MySQL 提交延迟 | **PASS** | 100/1000-hand MySQL profile, receipt-scale-profile, Race, clone/query/allocation controls; bounded runner measurements, not production SLA |
| G04 | P50/P95/P99 | **PASS** | 100/1000-hand MySQL profile, receipt-scale-profile, Race, clone/query/allocation controls; bounded runner measurements, not production SLA |
| G05 | 检查点及故障恢复耗时 | **PASS** | 100/1000-hand MySQL profile, receipt-scale-profile, Race, clone/query/allocation controls; bounded runner measurements, not production SLA |
| G06 | 高并发下连接池和锁竞争 | **PASS** | 100/1000-hand MySQL profile, receipt-scale-profile, Race, clone/query/allocation controls; bounded runner measurements, not production SLA |
| G07 | Race 检查 | **PASS** | 100/1000-hand MySQL profile, receipt-scale-profile, Race, clone/query/allocation controls; bounded runner measurements, not production SLA |
| G08 | Goroutine 泄漏 | **PASS** | Actual1500 WSS open/close connections after60 warmup, server Prometheus goroutines/sockets return to baseline within stated allowance; private query/clone controls also rerun; bounded duration only |
| G09 | 重复请求及失败重试压力 | **PASS** | 1600 same-ID queries at16workers, WSS16 repeats, original timer/cache controls and five in-process final retries leave counts unchanged |
| G10 | 长牌局的候选状态复制开销 | **PASS** | 100/1000-hand MySQL profile, receipt-scale-profile, Race, clone/query/allocation controls; bounded runner measurements, not production SLA |
| H01 | TypeScript 类型检查 | **PASS** | Exact-source strict TypeScript/build/dependency checks, client-foundation.log:30 core+37 Doudizhu+35 Cocos=102 tests |
| H02 | Web 实际构建 | **BLOCKED** | No exact-source native Creator artifact; actual current wx SDK/full rendered UI/devices unqualified; older artifact rerender only supplementary |
| H03 | Chromium 场景冒烟测试 | **BLOCKED** | Old06733a9 Creator artifact rerender passed at three viewports/disposal; manually inspected portrait. It is not an exact1ae8ee1 scene/build pass |
| H04 | 微信小游戏构建 | **BLOCKED** | No exact-source native Creator artifact; actual current wx SDK/full rendered UI/devices unqualified; older artifact rerender only supplementary |
| H05 | 抖音小游戏构建兼容性 | **BLOCKED** | No exact-source native Creator artifact; actual current wx SDK/full rendered UI/devices unqualified; older artifact rerender only supplementary |
| H06 | 游戏 UI 操作与真实网络请求一致 | **BLOCKED** | Current source GameOnlineController↔real Go room/fairness flows pass; current rendered Cocos/wx login-through-result flow unqualified |
| H07 | 重连及恢复凭证操作可触达 | **BLOCKED** | Current controller kit import/reconnect and room loss flows pass; actual current SDK/device UI reachability/recovery unqualified |
| H08 | 游戏页面布局、按钮可见性和状态刷新 | **BLOCKED** | Supplementary older home view inspected, debug HUD/small portrait text remain; actual current playable-table safe area/layout/device refresh unqualified |
| AUTO01 | Go 单元测试 | **PASS** | Exact-source local reruns and exact-source CI, commands/exits below and in runs.jsonl |
| AUTO02 | Go Race 测试 | **PASS** | Exact-source local reruns and exact-source CI, commands/exits below and in runs.jsonl |
| AUTO03 | Go Vet | **PASS** | Exact-source local reruns and exact-source CI, commands/exits below and in runs.jsonl |
| AUTO04 | Go fmt-check | **PASS** | Exact-source local reruns and exact-source CI, commands/exits below and in runs.jsonl |
| AUTO05 | MySQL 集成测试 | **PASS** | Exact-source local reruns and exact-source CI, commands/exits below and in runs.jsonl |
| AUTO06 | 100/1000 牌局专项测试 | **PASS** | Exact-source local reruns and exact-source CI, commands/exits below and in runs.jsonl |
| AUTO07 | 前端 TypeScript 类型检查 | **PASS** | Exact-source local reruns and exact-source CI, commands/exits below and in runs.jsonl |
| AUTO08 | 客户端单元测试 | **PASS** | Exact-source local reruns and exact-source CI, commands/exits below and in runs.jsonl |
| AUTO09 | 安全组件测试 | **PASS** | Exact-source local reruns and exact-source CI, commands/exits below and in runs.jsonl |
| AUTO10 | Cocos Creator Smoke | **BLOCKED** | Exact-SHA CI and Fair Doudizhu pass; no Creator workflow/artifact at1ae8ee1, no qualified local Creator/dispatch capability |
| AUTO11 | Production Compose Runtime | **PASS** | production-compose-runtime: original Go/Admin Dockerfiles plus all original bootstrap/auth/authz/HTTP/WS/HTTPS/WSS assertions in own isolated project |
| AUTO12 | 数据库架构边界测试 | **PASS** | Exact-source local reruns and exact-source CI, commands/exits below and in runs.jsonl |
| AUDIT01 | P0：数据丢失、敏感信息泄露、身份越权、不可恢复的数据破坏 | **PASS** | This independent audit/report executed including the confirmed P2 and unverified gates; not proof of defect absence |
| AUDIT02 | P1：重复出牌、状态分叉、错误结算、恢复失败、死锁及关键链路中断 | **PASS** | This independent audit/report executed including the confirmed P2 and unverified gates; not proof of defect absence |
| AUDIT03 | P2：边界条件、资源泄漏、错误处理、用户操作异常、可维护性问题 | **PASS** | This independent audit/report executed including the confirmed P2 and unverified gates; not proof of defect absence |
| AUDIT04 | 严重级别 | **PASS** | This independent audit/report executed including the confirmed P2 and unverified gates; not proof of defect absence |
| AUDIT05 | 文件及代码位置 | **PASS** | This independent audit/report executed including the confirmed P2 and unverified gates; not proof of defect absence |
| AUDIT06 | 触发条件 | **PASS** | This independent audit/report executed including the confirmed P2 and unverified gates; not proof of defect absence |
| AUDIT07 | 可复现步骤 | **PASS** | This independent audit/report executed including the confirmed P2 and unverified gates; not proof of defect absence |
| AUDIT08 | 实际行为 | **PASS** | This independent audit/report executed including the confirmed P2 and unverified gates; not proof of defect absence |
| AUDIT09 | 预期行为 | **PASS** | This independent audit/report executed including the confirmed P2 and unverified gates; not proof of defect absence |
| AUDIT10 | 建议修复方向 | **PASS** | This independent audit/report executed including the confirmed P2 and unverified gates; not proof of defect absence |
| GOAL1 | Real platform-code login exchange / session validation, token lifetime and secure private credential management; no fake identity, no tokens in URLs; include local authorized dev login path only if correctly access-controlled. | **BLOCKED** | Real Go password sessions/isolation pass; server-side wx AppSecret/code exchange/approved domains/device evidence unavailable |
| GOAL2 | Account-scoped rooms: create/join/leave/ready, seats, start hand, authoritative room refresh/reconnect and timeout/duplicate reconciliation; full Cocos interactive flow. | **BLOCKED** | Actual Go rooms1/2/3, leave/rejoin/ready/start and both before/after-send loss branches pass; full current Cocos/platform UI unqualified |
| GOAL3 | Fair lifecycle: server-bound commit/reveal key/AAD acquisition, secure entropy + HPKE, trusted beacon, deal, private hand and evidence verification, fail closed on missing security context. | **PASS** | All three actual client entropy/Manifest/signature/AAD/HPKE/new-process kit/drand deals/private outsider403 passed at this source |
| GOAL4 | Playable game: bidding, landlord determination, legal play/pass delegated to authoritative Go rules, turn indicators, cards/selection, clear error and version-retry handling, termination, score/evidence/result UI. | **BLOCKED** | Backend complete games and original lifecycle/settlement defects now pass; full exact-candidate rendered Cocos/platform UI through score/evidence remains unqualified |
| GOAL5 | B-level MySQL recovery: append each accepted live bid/play/pass state transition and dedup + owner epoch/fence atomically **before acknowledgement**; bounded encrypted checkpoint and ordered replay restoring game state, ambiguity handling and owner fencing; address seed and other secret continuity; no plaintext player hands in routine logs. | **PASS** | Current full MySQL/fencing/checkpoint/replay, original R4-N04/R5-N05 and actual WSS pre/post-COMMIT/terminal recovery controls pass; stated process-loss/storage-intact scope |
| GOAL6 | Real 3-player repeatable E2E, reconnect and crash-injection: pre-/post-COMMIT kill, response loss, duplicate/out-of-order commands, competing owners, wrong epoch and replay mismatch; preserve every acknowledged action. | **PASS** | Three real fair WSS games, phase-gap/pre/post-COMMIT SIGKILL, same-ID concurrency/response loss/original replay and corruption refusals pass |
| GOAL7 | Performance and stability: bounded queues/workers/room memory, P50/P95/P99 measurements and transaction fsync budget on stated hardware, no unknown target claims. | **BLOCKED** | 100/1000 latency/heap/concurrency measured; sustained room/socket soak, production storage/hardware/WAN evidence unavailable |
| GOAL8 | WeChat Creator package, actual developer tool, approved HTTPS/WSS domains and real handset evidence where access is available. Record platform-owned AppID/domain/device blockers honestly. Preserve Web and Douyin build compatibility. | **BLOCKED** | Exact-SHA native packages/current SDK gameplay/approved domains/real wx handsets and Douyin qualification unavailable |

## Open conditions, evidence review and handoff

No new P0/P1/P2/P3 product defect was established. R5-N05 and the original R3-N02/R3-N03/R4-N04/F01–F11/N01 reproductions pass at this source. No new product-code repair is requested on those findings. Remaining conditions require explicit qualification/operations work: full room/session/history memory reclamation and sustained hardware/storage/WAN stability; current exact-source native builds and rendered UI→backend full game; real wx server credentials/approved HTTPS/WSS/TLS whitelist and three physical identities/devices with lifecycle/share/safe-area/recovery behavior; Douyin SDK/device/entropy; KMS/key custody and all Seed/Checkpoint/contribution/Journal dependencies before root retirement. Local Node/simulator/old native evidence cannot substitute for these.

Second-pass report/self-review, not a second independent agent: inspected source guard/keyring/bootstrap/config and new tests, original downgrade proof, normal v2/internal0/keyless adapter compatibility, receipt publication refusal, source/artifact identity, complete success/failure/restart paths, timer monotonicity/cache misses/serialized duplicates, epoch/COMMIT ordering, schema ownership/rollout and resource limitations. Kept formal assertions unchanged. Initial private TLS startup omitted nginx.conf because of a script-copy filter; removed only the failed own container/empty mount directory and copied unchanged fixture config before any game/account setup. A private config read exposed retained isolated-fixture credentials in tool output; removed them from the copied YAML and regenerated fresh fixture keys/credentials before starting business validation. No production credential was involved; final active fixture/public documents/logs are checked separately. The initial private keyless probe also assumed full durable cold recovery with an unconfigured checkpoint reader; source requires protected checkpoints, so the probe was corrected to verify the actual keyless Append/Read adapter contract and keyed rejection, while normal authenticated internal0 cold recovery remains independently tested. Initial serialized-order-root-final.log failure retained; corrected full probe rerun in serialized-order-root-confirmed.log. No unsupported keyless Coordinator PASS is claimed. These fixture corrections are not product defects, and initial evidence is retained rather than relabeled successful.

Owned Round6 supervisor/app/TLS/MySQL/Redis stopped after final gates. Runtime Compose cleaned its own containers/network/volume; test DB volume retained privately for reproducibility. All owned ports3306/6379/28888/28889/28999/28443/28444/28080/29081/29443 have no listeners. Only unrelated learn-xray/learn-xray-subscription remain; both Kubernetes nodes Ready, kubelet/containerd active (cleanup.json). Earlier-round volumes and production services untouched. Evidence permissions, generated-secret scan scope/results and SHA256 inventory recorded privately; no claim of scanning images/memory/all nested artifacts. Final source/requirements/historical-content/document diff reviewed before publication.


**Final decision:** repair/executed-code regression gates PASS; full product/internal qualification BLOCKED by the remaining evidence gaps, external qualification BLOCKED, production release BLOCKED, sealing NO. Preserve Goals0045/0046 `in_progress`. ChatGPT can proceed with remaining qualification/resource/operational work; no speculative code change or repeated R5-N05 repair is required by this report. A later complete-product acceptance still needs evidence for every required open condition at its own fixed source SHA.


## 2026-11-04 补充：已登录微信工具的实际 SDK 与 Go 网络验证

用户指出已登录的微信工具仍可继续验证。本次重新启动保留的社区 Linux 微信工具，CLI `islogin` 实际返回 `{"login":true}`，不是沿用历史登录结论。原有 AppID 和已登录配置都存在；“微信未登录／没有 AppID／SDK 无法执行”不能作为本次阻塞原因。

产品源码仍为 `1ae8ee111f8dbaad2e8312b55a0a1f51603f34da`；开始时仓库 HEAD 为 Round6 文档提交 `118ae7b3ceb03e8d655d0d9af6ea903df116b127`。未修改产品、正式测试、业务设计、原项目或生产部署。使用此前已校验的 Creator3.8.8 产物06733a9，验证其全部客户端/构建脚本输入与当前代码无差异；只在私有副本 project.config.json 设置已有操作者 AppID 和项目显示名。引擎、应用、资产文件哈希保持一致。这不是一次新的当前 SHA Creator 原生导出。

| 实际补充检查 | 结果与范围 |
|---|---|
| 当前工具登录态 | PASS：CLI islogin=true，真实小游戏上下文所属 AppID 匹配操作者项目 |
| 当前项目适配器调用真实 wx.login | PASS：SDK3.17.5/platform=devtools，返回非空32字符 code；探针仅记录是否非空与长度，不输出 code 内容 |
| 微信安全随机源 | PASS：使用现有 createMiniPlatform 与 UserCryptoManager 路径，两次32字节输出长度正确且不同，随后清零临时数组 |
| 存储、生命周期监听 | PASS：独立测试键保存/读取/移除；当前适配器 onShow/onHide 注册和卸载成功。未验证真实前后台事件触发、分享或手机系统行为 |
| 真实 wx.request→当前 Go | PASS：全新隔离 Go 测试账号通过 HTTPS 登录和会话验证。服务端是 Round6 已验证的同一二进制（哈希一致），使用新数据库/新凭据，不改以前验收库 |
| 真实 wx.connectSocket→当前 Go | PASS：现有 MiniSocketTransport 使用实际 SocketTask 建立认证 WSS，取得关联 system.pong；无效令牌拒绝；退出自有会话后旧令牌被拒绝；未将令牌放在连接 URL |
| 微信 code→Go 业务身份完整换证 | **仍未通过**：真实 SDK 生成的 code 已通过实际 HTTPS 请求发到 `/auth/wechat/code`，返回 HTTP500，Go 日志明确为 `WeChat identity exchange not configured`。隔离服务缺少 APP_WECHAT_APP_ID/APP_WECHAT_APP_SECRET/APP_WECHAT_SUBJECT_HMAC_KEY_HEX。工具登录不会自动配置这组服务端参数。已向用户询问可用测试后端或本地私有配置路径，没有索取明文 AppSecret |
| 完整 Cocos UI 与真机 | **未验证通过**：真实 System.import('cc') 可加载引擎，但 director 尚无活动 Scene/Canvas；SDK/网络通过不等于界面已可玩。没有进行上传、预览、审核、发布或真机操作 |

网络认证对照使用服务端正常创建的隔离密码账号；它证明真实微信 SDK 的 HTTP/WSS 能与当前 Go 认证协议配合，**不表示该对照账号已经通过微信 code 换证成为平台业务身份**。正向平台换证需要有效的服务端测试配置，不能用工具登录态或假身份替代。

第一次完整产物运行触发工具自身2GiB上限（oom-kill）；在主机有余量时以3GiB/无swap/CPU150%重新运行，SDK检查完成。工具均有运行时限。初次手写随机数探针用了错误的 wx.getRandomValues 入口，改为加载项目实际适配器后通过；未据此报告产品缺陷。模块探针补齐实际依赖后复测；这些是私有探针纠错。

本地 HTTPS 首次失败的工具日志为 TLS net_error -202。随后仅对隔离服务证书配置精确 SPKI 信任和 NODE_EXTRA_CA_CERTS，重试实际 SDK 网络检查通过；未主动启用全局忽略证书错误。私有项目使用开发调试 urlCheck=false，故不构成正式域名白名单或生产 TLS 资格。

私有新证据：`/root/.config/awesome-zero-platform/round6-wechat-supplement-20261104`，主要文件 cli-islogin.log、project-provenance.json、sdk-login.json、sdk-network.json、backend-boundary.json、cocos-module-state.json、local-tls-trust.json、cleanup.json、SHA256SUMS，及实际探针脚本。截图、AppID、登录配置、凭据、code 和令牌不提交。Go测试会话已经退出；本次工具/服务/容器均停止，端口清空，已登录配置保留，原验收库与无关服务不动。

本段是定向补充，不是114项完整重跑；2026-11-03的97PASS/0FAIL/17BLOCKED矩阵保留为当时记录。相关阻塞原因应收窄：**工具登录、真实 SDK 登录/随机源/存储监听，以及真实 SDK→Go HTTP/WSS 已有本次实测证据**；GOAL1仍需正向平台换证，H06/H07/GOAL2/GOAL4仍需完整UI流程，当前原生导出、正式域名与真机要求仍未满足。不能再笼统归因于“微信没有登录”。未新增已确认的产品代码缺陷，Goal0045/0046继续保持 in_progress。
