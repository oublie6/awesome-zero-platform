# Goal0062 Round5 — Kubernetes test delivery and business acceptance

2026-11-16 UTC; evidence manifest recorded09:19:41 UTC (point-in-time observations). **PASS for the authorized test deployment and bounded business regression**, with the operator credential-logging incident disclosed in section 8. No substantive product defect confirmed. This does not complete Goal0062 or Goals0045/0046, qualify production/device use, or resolve that logging incident by assertion.

## 1. Fixed source and CI

Product **`a739e23f315a98ad76fd83c233aae1813415037a`**, documentation baseline `00935c057fdb5a5e3900e2152e63b21616648da2`; clean synchronized `main`, `git pull --ff-only`. Final comparison of `server`, `clients`, and `scripts` against the fixed product is empty. No product repair, new feature, branch, full CI rerun, or full game/crypto/recovery campaign.

Reuse exact-product [CI43814460914](https://github.com/oublie6/awesome-zero-platform/actions/runs/43814460914) **6/6 SUCCESS**, [Fair Doudizhu43814460946](https://github.com/oublie6/awesome-zero-platform/actions/runs/43814460946) **4/4 SUCCESS**, and the [Round3](0062-round3-independent-acceptance.md)/[Round4](0062-round4-independent-acceptance.md) software acceptance within their stated scopes. Historical FAIL reports remain unchanged. This report's eventual documentation commit is distinct from the tested product and image revisions.

## 2. Predeployment environment and database

Only context `kubernetes-admin@network-learning`, namespace `awesome-zero-game-test`; existing API, Web, MySQL5.7.44 and Redis. Snapshotted Deployment/Pod/image identities, Secret references/material, ConfigMaps, Services, PVCs, schema, data and both nodes' resource state before changes. Existing API/Web had baseline2/1 restarts; MySQL/Redis0. Existing scheduling and resource limits retained: API512Mi/1CPU, Web128Mi/500m, MySQL512Mi/750m, Redis128Mi/250m, on `ser730417996259`.

Database `goal0057`:20 InnoDB tables,4 historical accounts/credentials, no external mappings or conflicting provider/account duplicates,2 completed hands/archives and no unfinished hand. Missing requirements were the provider/account unique index and pending-proof table. Original account UUIDs, room/hand membership, action/checkpoint/archive/protected data were inventoried. Counts below come from `COUNT(*)`; binary TSV line counts are not database row counts.

Owner/authority: Identity owns its additive schema and account invariants; apps own composition/deployment; Doudizhu retains its existing authority and module-owned records. No public contract, business data ownership, persistence model, dependency direction or architecture changed. Public registration remains explicitly test-gated, email ownership unverified. Read [principles](../principles/README.md), current Goal, delivery/security repair and prior TLS/acceptance documents before operations.

## 3. Backup, additive migration and verification

Created a full transactional backup of the existing database before DDL, `goal0057-before.sql.gz`; uncompressed494553 bytes, SQL SHA256 `bbca6e992be848c684ff2af1fafffceb4a92cf8714bf084f221230468aade6de`. It contains no database-selection/recreation command. **Actually restored** into the new, owned `goal0062_r5_restore_a739e23` database. All20 ordered table byte hashes matched the original inventory; restore exit0. Existing database was never cleared, rebuilt, overwritten or restored over.

Applied only:

- `identity_external_subjects`: unique `uq_identity_external_provider_account(provider,account_id)`.
- New `identity_external_pending` table with the canonical source definition.

DDL succeeded after conflict/backup checks; all20 original table hashes were unchanged immediately afterward. Final schema inspection confirms the index and pending table. Temporary migration SQL stays private, outside Git. No unrelated schema upgrade or account/game rewrite.

## 4. Images, safe cutover and runtime

API built serially from an archive of the fixed SHA using Go1.25.8, `CGO_ENABLED=0`, `-p 1`, bounded memory/CPU. Runtime binary extracted and verified: SHA256 `e42bdc82590fa14ca4ec6d13085724de083cda0624666e8a29c04354783fd9bc`.

Reused the previously built Vue output because `clients/game-web` is unchanged from its accepted build. Cocos differs from the old deployed client, so made a fresh official Linux native Web export using pinned CLI `f506fa3de403b48b4833334fa48dddd27bb26364`, engine4.0.0-beta.0 revision `d0a696cfe63fd8d19c70d7913ba11aa641c2e3c7`, Node22.23.4. This beta build is test evidence, not release qualification. Strict generated declarations and emitted-JS AST/native-BigInt gates pass. First bounded build timed out at600s during the import/build workflow, without OOM; one analyzed continuation reused its workspace/cache and completed the exporter in2m59s. An initial standalone AST check omitted `COCOS_CLI_ROOT`; supplying the pinned toolchain environment corrected that fixture failure. Raw logs retained, no source workaround or Wine.

Pinned runtime images, with revision labels and actual Pod imageIDs verified:

| Process | Manifest digest |
| --- | --- |
| `docker.io/awesome-zero-platform/goal0062-api` | `sha256:9b8bf32b89c384c7f6760473c0f29b72999ed3b2ffff36876693ff2e6832906f` |
| `docker.io/awesome-zero-platform/goal0062-web` | `sha256:ab52cee0608128afef4af6466c90bb7141b92d4b43e51bbc34ab165a537caab2` |

Actual Web container contents match all30 manifest files; scan of11 known sensitive values found0 matches. Kept existing Nginx/CSP, public signing root, TLS certificate and protection/signing material. Existing `g57-secrets` and `g57-tls` data are byte-identical. Added only related `g62-wechat` from existing private test configuration; no new HMAC/root rotation. Enabled test registration and added its three WeChat Secret references to API; original ConfigMaps, Services, PVCs and limits unchanged.

Safe single-API replacement: observe a real historical authenticated socket, scale old API to0, wait for deletion and verify0 API Pods, revoke15 legacy sessions across4 historical accounts using the real store/epoch owner, then start the new API. No old/new API overlap. Old socket closes1012; old access and refresh return401; readiness200. Invalid WeChat code is rejected400; no official exchange success is claimed. Web replaced with the pinned image and the existing user frontend forward restarted.

Final Pods **API `app-api-f94454b76-9g26z`, Web `game-web-996b46c84-dbfwq`, MySQL `mysql-5cc48fd9b4-5hz6j`, Redis `redis-67bdd89687-krvb4`: all1/1 Ready, zero restarts**. MySQL/Redis Pod UIDs and volumes unchanged. Trusted HTTPS `/` and `/readyz` return200 using the retained test CA, without TLS bypass. Final10-minute log sample: API218 info/30 stat/0 error entries; Web/MySQL no error-level lines; no Redis lines. Expected negative-test HTTP responses are not hidden as failures.

## 5. Actual deployed business results

| Requirement | Observed result |
| --- | --- |
| Registration/login and privilege | Real Vue registration201, `emailVerified=false`, separate password login, new platform UUID. Real Admin HTTP creation of second ordinary account201. New self-registered account has0 roles and `/admin/accounts`403. |
| Controlled provider lifecycle | Production bind/relink/unlink/register HTTP routes against actual MySQL/Redis pass. Relink invalidates source provider and destination password sessions; both sockets close1008. Unlink denies old access/refresh and closes1008. Provider-first registration returns a new platform UUID, unverified email; password login remains usable after unlink. |
| Password and administration | Only owned new QA accounts changed. Identity-owner password change, real Admin HTTP reset, disable/enable pass; old access/refresh denied, sockets1008, fresh password login works at the same UUID. Reenable does not revive old sessions. |
| Refresh/logout/relogin | Rotation succeeds, old refresh replay401; existing connection remains valid through rotation. Logout denies tokens and closes1008; subsequent fresh login retains original account/room membership. |
| Historical accounts/data | All3 retained player passwords log in at their original UUIDs. Real browser restores an existing completed game; both historical evidence HTTP reads200/finalVersion60. Historical admin password was not reset. |
| Room commands/recovery | Actual Vue/Cocos create roomv1; second account joinsv2; ready acknowledgmentsv3/v4. Page reload recovers same room/v3; real WebSocket reconnect recovers same state. Serial scenes, no page errors, no new hand. |
| Account-directed messages |3 real historical authenticated sockets; one exact existing terminal `hand.play` receipt replay returns live version60 and delivers3 account-directed `doudizhu.hand.changed` notifications plus3 final-evidence messages through the actual Hub/game bridge. No new action, hand or full game. |
| Ownership and cleanup | Account lifecycle leaves the QA room row byte-identical. Historical ownership unchanged; final active-WebSocket gauge0 after owned sockets close. Owned operator/forwards/relay/test processes stopped. User29457 forward and all4 services retained. |

Provider proof generation and external-session issuance use an authenticated, loopback-only operator fixture invoking the actual Identity/AuthN owner contracts with one controlled verified subject. They do **not** test WeChat's official exchange or device SDK. Historical admin password is unavailable; a controlled trusted admin session at the existing UUID exercises the deployed authorizer, permissions, HTTP handlers and audit writes. It is **not** an admin-password-login pass. Password change has no public HTTP route: its mutation used the actual owner contract; subsequent invalidation was tested against deployed HTTP/WSS. Reset/disable/enable used real deployed Admin HTTP. No mock response, product endpoint or product source was introduced.

Raw failures and bounded continuations remain separate:

- Initial admin-create fixture omitted required `phone`; adding `phone:""` matches the existing DTO. Initial role fixture expected an object; actual route returns list/null. Both corrected before the affected mutations, without changing the API.
- Second simultaneous Cocos scene exceeded45s startup wait, without page errors. A single-scene diagnostic joined the already-created room in24.751s, then serial reload/reconnect/ready flow passed. Host CPU steal was sampled at38.9% during build. Resource contention is plausible, **not proven as the timeout's sole cause**; simultaneous-scene startup performance remains unqualified.
- Initial reload fixture used exact label without the actual arrow suffix; corrected selector restored the existing room without repeating ReadyA or creating another room.
- First relink reached HTTP200 and rejected both old token pairs, but the shared `kubectl port-forward` exited on a documented broken pipe, yielding socket1006. Preserved failure and journal; no API/Web restart. A separate bounded loopback TCP relay to the same TLS Web Pod isolated channel failures. Reverse relink between the same owned accounts and all remaining lifecycle checks then observed1008. User29457 forwarding remains active with restart policy. This transport failure is not relabeled a product1008 pass.

## 6. Bounded performance and resources

One client,5 warmup WSS pings, then30 measured sequential request/replies;20 deployed `/auth/session` reads. TLS/WSS through the isolated local TCP relay, real Web proxy/API/MySQL/Redis, retained CA, Node22.23.4. Nearest-rank milliseconds:

| Operation | N | P50 | P95 | P99 |
| --- | ---: | ---: | ---: | ---: |
| `system.ping`/`system.pong` |30|7.733|38.187|45.936|
| HTTP session validation |20|26.736|64.407|67.802|

Small samples: P99 equals maximum, no production SLO, large-load or soak claim. This environment/path differs from Round4's matched ABBA fixture; absolute numbers are **not** a new before/after speedup comparison. Round4 remains the query-reduction evidence.

Actual API metrics during this measured interval: managed Redis pipeline129→190 calls, duration sum183→227ms (61 calls/44ms aggregate, including final revoked check); managed GET346→368, sum978→1073ms. These are client histogram deltas, not packet-level RTT counts or exact per-message SQL counts. Managed pool10 total/10 idle/max10 after sample, native pool2/2; pool timeouts0, no new miss/stale count in the measured window. Existing startup `client` error counter1 unchanged; not a claim of lifetime zero errors.

SQL server processlist filtered by the actual API Pod IP shows3 idle application connections before/after, configured max16/idle4. Global `Slow_queries` stays0; total connected4/running1 including inspector. Global `Com_select`121740→121895 includes background/lifecycle work, so is not attributed solely to pings. `Aborted_connects`28495→28502 also rose; unchanged MySQL readiness is a5s TCP probe, consistent with connection-only probe churn, not proof all aborts have that cause. No API SQL error in final log sample. Statement-digest query yielded no rows; **SQL-only latency and driver wait counters are not exported/measured**. HTTP-session timings cover actual combined authority reads and transport, not isolated SQL latency.

Kubelet stats,32–35s sampled windows around the bounded work:

| Pod | Working set MiB | Average CPU mCPU |
| --- | ---: | ---: |
| API |150.42|29.40|
| Web |12.81|2.70|
| MySQL |207.54|11.93|
| Redis |7.33|8.22|

Primary/HK node available memory2450/4017MiB; both Ready and MemoryPressure=False. No service redistribution needed. Builds/tests serial with bounded cgroups; no concurrent heavy work or OOM observed. Samples are not peak-memory or leak/long-stability evidence. Metrics API unavailable; used actual kubelet stats summaries instead.

## 7. Final integrity and operational recovery

Final original-row comparison joins every table to the restored predeployment copy by its primary key, orders columns/rows identically, and compares **bytes**, preserving binary columns. All original row counts retained. **19/20 original table subsets byte-identical**. Only `doudizhu_live_writers` differs:2 `owner_epoch` and2 `updated_at` fields advance during cold ownership recovery; epochs monotonic, committed versions and genesis snapshots unchanged. Original account/credential, role/policy, hand/action/checkpoint, contribution/seed, room and final archive records otherwise identical. Restore-copy hashes still match the preflight hashes.

Actual original→current counts: accounts/credentials4→7, rooms2→3, command receipts33→37, client sequences14→16, outbox47→51, audit4→8. These additions belong to the3 QA accounts and room/lifecycle checks. Hands2→2, actions118→118, checkpoints2→2, contribution records6→6, protected seeds2→2, final archives2→2, provider mappings0→0. No old record deleted or moved. New QA data and tested restore clone retained. First inventory attempt stopped on UTF-8 decoding of binary data; byte comparison corrected the inspector, without database mutation.

Recovery assets: original Deployment specs/image digests, original configuration/Secrets and full transactional backup retained privately. `rollback.py` passed **server-side dry-run for both Deployment spec replacements**, not an actual rollback exercise. To roll back: restart the owned MySQL/Redis forwards and retained operator using its private config; stop API and verify0 API Pods; invoke the real bounded all-account session drain (refuses >100 accounts); restore old API spec/image, await Ready, restore old Web spec/image and restart user forward; recheck health and force fresh login. Script refuses to apply while API Pods exist. Retain additive DDL/new data/keys; do not automatically restore the old SQL backup over newer writes. Full disaster restore requires a separate stopped-writer, preserve-new-writes decision; actual restore here was only to the owned clone. No compatibility-safe rolling overlap is promised.

Operator cleanup's raw result reports10 failures/1 revoked. `Revoke` also counts already-absent SIDs as session-not-found; the fixture did not record individual error identities, so all10 cannot be conclusively classified from that counter. Separate Redis inspection confirms0 remaining admin sessions; removed1 remaining session belonging strictly to the3 new QA UUIDs. No legacy/user session sweep at final cleanup. Operator SQL/Redis connections closed; owned dependency forwards/relay/browser jobs stopped. All4 business services and existing storage remain running. Existing frontend `azp-game-test-forward.service` stays on127.0.0.1:29457.

## 8. Limits and operator incident

No production deployment/domain/public exposure, email verification/sender, SMS, real WeChat-device binding, release qualification, full new game, fault-injected live-hand recovery, capacity/soak campaign, distributed failover or peak-resource guarantee. Prior software/native/game evidence is reused only within its original scope. Serial browser functionality passes; the simultaneous-scene timeout is retained and not dismissed as a measured performance pass. Goals0045/0046 remain open; Goal0056 Honor pause and C09/C10 DEFERRED unchanged.

**Operator error:** while inspecting a private metadata file, the new related test Secret's encoded AppID/AppSecret/subject-HMAC content was inadvertently emitted into this conversation's tool output. Encoding is not redaction. No values are repeated here, no known match was found in the runtime Web artifact, and the report/diff is checked for known secret values before commit. Treat the conversation/tool log as sensitive; credential owner may review its access and arrange provider-credential rotation separately. Existing keys remain unchanged under the explicit preservation constraint. No claim that the output was erased, that credential rotation occurred, or that transcript confidentiality was independently verified. This is an operator/log-hygiene incident, not a product-secret-response defect; the functional PASS does not certify its remediation.

## 9. Evidence, delivery and conclusion

Private E5=`/root/.config/awesome-zero-platform/acceptance-0062-r5-a739e23` (0700; credentials/config snapshots0600). `result.json` SHA256 **`0264b261d2f18d6217bf11d2a90c94f73a30294a92160f1a5ad78c48affe8166`** inventories42 evidence hashes: backup/restore/migration, runtime files/images, cutover/drain, raw and resumed browser/account results, historical replay, all timings/metrics/resources, original-data comparison, final state/cleanup and log-hygiene record. Private scripts/configuration and failed logs retained; no secret or migration SQL committed.

Reproduction entry points: retained `browser-business*.cjs`, `join-diagnostic.cjs`, `room-continuation*.cjs`, `account-lifecycle*.cjs`, `historical-push.cjs`, `deployed-performance.cjs`, `snapshot.py`, `integrity-bytes.py` and `rollback.py`. Node tasks use the pinned Node, retained CA through `NODE_EXTRA_CA_CERTS`, isolated browser trust home and bounded systemd units. Inspect saved progress/QA state before running a continuation; **do not blindly rerun account registration/mutations or migration**. `kubectl --context kubernetes-admin@network-learning -n awesome-zero-game-test get pods -o wide`, trusted HTTPS health probes and the private snapshot scripts provide read-only final-state checks.

Codex independently executed this deployment acceptance. Second-pass operator review checked exact source/artifact provenance, source-vs-runtime boundaries, DDL/restore evidence, old/new session fencing, partial fixture failures, historical ownership, byte/count inventory semantics, pool/resource limits, cleanup and secret handling. Corrected inspectors/fixtures are explicitly distinguished from production repairs; no production repair made. The logging error remains disclosed above.

**PASS for Round5's scoped Kubernetes test delivery/business checks.** Broader product state remains `in_progress`; only this report and permitted current-goal state/report sections are delivered on `main`. Keep the validated test environment and user tunnel running; no automatic next development round.
