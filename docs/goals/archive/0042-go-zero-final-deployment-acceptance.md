# Goal 0042: go-zero architecture final deployment and runtime acceptance

## Status

- State: completed
- Started: 2026-10-30
- Completed: 2026-10-30
- Blockers: No implementation/deployment blocker. Residual operational risk: historical severe soft lockups on the control-plane host; documented for operator follow-up, with no host intervention performed.

## Goal

Complete the user-requested final code, deployment consistency, runtime, observability and reliability acceptance of Goals 0039–0041. Preserve existing correct business behavior and the modular monolith baseline.

## References

- `AGENTS.md`
- `docs/operations/logging-observability.md`
- `docs/operations/production-deployment.md`
- `deploy/kubernetes-overlays/observability-server/README.md`
- `docs/architecture/go-zero-http-and-module-integration.md`
- `docs/goals/archive/0039-go-zero-api-http-standardization.md`
- `docs/goals/archive/0040-redis-and-trace-governance.md`
- `docs/goals/archive/0041-module-contract-guardrails.md`

## Deliverables

1. Final code review of REST routing/middleware, JWT/session/Casbin compatibility, go-zero SQL/Redis usage and scoped native adapters, WebSocket context/tracing, metrics and module/data boundaries. Minimal fixes with regression tests only for demonstrated defects.
2. Kubernetes inventory, source/image provenance, configuration drift and resource/disk/PVC/restart evidence. Backup, verified rollback and deployment preflight before updating the API through the existing build/import process.
3. Real REST health/login/refresh/logout/Admin/authentication/authorization/error checks; WebSocket authentication/message/heartbeat/disconnect/reconnect checks; SQL/transaction/cache/WATCH/recovery checks.
4. Real Prometheus HTTP/error/latency/WS/Redis/runtime/CPU/memory collection, Grafana panels, Jaeger HTTP/SQL/transaction/Redis/WATCH/message spans with relationships/timing/errors, and Filebeat/Elasticsearch/Kibana request/trace correlation evidence.
5. Bounded reliability observations for goroutines/connections/memory, slow consumers/cleanup, backend recovery, host CPU/soft-lockup risks and retention/watermarks.
6. Formatting/tidy/generation/API contract/unit/race/integration/architecture/frontend/Kubernetes and GitHub Actions checks, distinct self-review and detailed final evidence report.

## Constraints

- User explicitly authorizes this Goal creation, code review, minimal fixes, tests, deployment and documentation changes; this expands the usual supplementary-executor role for this task.
- Scope is reusable technical infrastructure/platform integration and deployment verification; no speculative abstraction or unrelated business behavior changes.
- Main only; low-concurrency sequential resource-intensive work; no concurrent file writers.
- Use only observability-server profile, one replica and Recreate; preserve runtime settings and MySQL/Redis volumes. Never apply the default two-replica base or bootstrap schema over existing data.
- Stop deployment if backup/data safety or rollback cannot be confirmed. No disruptive fault injection in an unconfirmed environment. Never output credentials or sensitive payloads.
- Evidence must distinguish this executor, previous ChatGPT checks, CI, unavailable checks, failures/fixes and remaining uncertainty. No source-only claim of live trace success.

## Acceptance Criteria

- Every deliverable above has concrete observed evidence, including deployed immutable source/image identity before and after and actual component status.
- Verify latest source CI before deployment and final CI after push. Run available repository checks without weakening assertions; document exact unavailable checks or failures.
- Complete a distinct self-review covering contracts, state/order/concurrency, cleanup/recovery, privacy/security, resource use and boundaries; trace success and failure flows.
- Report changed files/reasons, code review, command/CI outcomes, deployed Git SHA/image, live signals, fixes, residual risks and required follow-up.
- Only after all criteria pass: mark completed, archive with scripts/archive-goal.sh, reset current template, commit/push main and confirm final CI. Otherwise retain incomplete evidence and document genuine blockers; never fabricate completion.

## Agent Strategy

Single primary executor, sequential verification and mandatory self-review.

## Working State

### Completed

- Read required instructions/references; clean main synchronized from 890abc8 to 698d760; Goals 0039–0041 archived completed; synchronized source CI verified.
- Goal definition 399703c and observability checkpoint 0a9ae1c committed/pushed on main; both CI and Fair Doudizhu workflows passed on each checkpoint.
- Reviewed go-zero routes/middleware, JWT/Session/Casbin compatibility, normal SQL/Redis paths, native transaction/WATCH/PubSub exceptions, context/tracing, metric registries and module/data ownership.
- Corrected span failure statuses/panic handling, WebSocket failure status, HTTP extension-method cardinality, access-log trace correlation (including an earlier context overridden by incoming W3C context), framework Redis metrics exposure and platform SQL ownership. Added transaction tracing to native authorization and business-command paths while preserving READ COMMITTED, lock order, commit/rollback, retryable errors and partial-success behavior.
- Identity now owns the account-state FOR UPDATE query through a documented public in-process transaction contract. Admin catalog ownership is explicit; stricter platform-to-platform SQL checks and owner-query/cursor/error tests added. No schema or business enablement changes.
- API DSL validation, generated-code clean diff, gofmt and idempotent tidy passed using Go 1.25.8 / goctl 1.10.1. Fresh complete unit suite passed on the current patch. Full platform/foundation/architecture Race passed, plus focused Race after subsequent HTTP-context and native command-trace changes.
- Admin Web typecheck/build passed (existing >500 kB chunk warning). Secure-envelope client 9/9 tests and TypeScript-to-Go interoperability passed using SHA256-verified Node 22.23.4. No frontend source/dependency-version changes.
- Unchanged make integration-test passed with isolated MySQL 8.4.6 / Redis 7.4.5, matching the deployed backend versions; 117 seconds. Only loopback ports and temporary storage used; dependencies removed. Baseline MySQL 5.7 / Redis 8.8 checks still require new-checkpoint CI.
- Negative tests: all six new observability regression tests reject pre-fix production code via Go overlay without worktree mutations. A temporary cross-module SQL probe is rejected by the strict ownership test and was removed.
- Real cluster inventory/PVC/node/strategy/image and initial diff verified; API still runs goal0038-gozero-first, one replica/Recreate. Both nodes Ready; no pressure conditions. Kubelet stats obtained because Metrics API is absent. /data initially has 26 GiB available. Shared CPU steal remains a measured operational risk (~35% in a two-second compile-time sample).
- Private backups: consistent MySQL dump (15 InnoDB tables), Redis RDB and old image/Deployment/rollback patch with SHA256 manifest. Isolated no-network restore passed: all MySQL table counts matched; Redis loaded 2 keys. Both restore containers and disposable restored-data copy removed; primary snapshots retained outside Git.

### In progress

- Acceptance evidence and detailed report are complete. Archive this goal, reset the template, commit/push on main, and confirm CI for the archival commit.

### Remaining

- Final CI for the archival commit; report its result to the user. No code, rollout, data, or runtime acceptance item remains open.

### Verification status

- All source, runtime and deployment acceptance criteria passed. Full unit and required focused/full Race checks, API validation/generation, gofmt/tidy, frontend/client, integration, strict module-boundary checks, Kubernetes rendering, isolated restore/recovery, live REST/WS, Prometheus/Grafana, Jaeger and Filebeat/Elasticsearch/Kibana checks are recorded in the detailed report.
- Runtime rollout used the immutable 3fd4cb6 image. One API replica/Recreate, runtime configuration, Secrets, PVCs and business disabled state were preserved. No live database/Redis outage or data-volume reset was performed.
- Source CI before rollout: 3fd4cb6 CI 41053084557 and Fair Doudizhu 41053084529 succeeded. Manifest checkpoint 4b36269 CI 41056434697, Fair Doudizhu 41056434701 and Kubernetes manifests 41056434663 all succeeded. The archival commit is the only pending CI confirmation.
- Evidence is under `/root/.local/state/awesome-zero-platform/goal0042` (directory 0700; sensitive snapshots 0600; excluded from Git). No credentials or secret payloads are included in repository documentation.

## Completion Report

Acceptance complete; goal archived. The archival commit CI is pending confirmation after push. See the [detailed deployment and runtime report](../../operations/goal0042-kubernetes-acceptance.md).

### Changes and self-review

- Changes are reusable technical infrastructure/platform integration plus tracing of the existing native business persistence exception. Foundation/platform gained no game-specific semantics. No public HTTP, DB schema, persisted format, client contract, pool configuration or deployment replica/strategy changes.
- Distinct second-pass self-review re-read the goal/architecture/public contracts and challenged defer ordering, rollback/commit/panic/error identity, READ COMMITTED, command deduplication, super-admin lock ordering and post-commit reload errors, cursor closure after active rows, context cancellation/parent propagation, secret redaction, label bounds and duplicate registries.
- Self-review found and fixed another native business transaction bypass and the case where an earlier valid request context could hide go-zero's actual response trace. Added meaningful regression tests, including cleanup assertions at Span.End, full-result cursor errors, and the existing driver's mandatory isolation assertion. Fixed a potentially unbounded receive in a new WS test. New observability and ownership tests were shown to reject original defects. This is self-review, not independent review.
- Explicit native limits remain documented: raw transaction statements do not each emit SQL spans; context-less Casbin callbacks use separate roots; WebSocket messages use short independent roots and propagate context to handlers; managed Redis has its own version-specific pool/timeouts, exposed separately from native gauges. Future service extraction must redesign the explicitly shared transaction invariant.
- Earlier failed invocations are retained: system Go 1.18 could not parse go.mod; K8s validator initially lacked its input argument; an initial Go run was invalidated by imports edited after scanning; Python rejected the Node npm symlink. Matching tools/correct invocations/fresh tests resolved these; no assertions were removed or tests silently skipped.
- No off-node backup destination has been provided. Existing shared-vCPU risk, private-only/unauthenticated observation stack, Jaeger in-memory storage, frontend chunk-size warning and limited-duration resource checks must remain explicit in the final report.

### Checkpoint 45707c4 and final privacy correction

- Source checkpoint 45707c4fbe40017f90e61d41a5195aee02959e90 pushed; CI 41050979330 and Fair Doudizhu 41050979382 both success (including baseline integration/runtime). Local make build passed; Go 1.25.8, go-zero 1.10.2 and vcs.modified=false recorded in binary metadata.
- Isolated same-source API recovery passed on fixture-only MySQL/Redis: paused ~14.17 / 9.46 seconds; recovered successful checks after ~0.03 / 0.11 seconds following unpause. Readiness failed, liveness stayed healthy, authenticated requests failed closed, then recovered. All test containers/processes removed; no production outage injected. Known-secret/Argon2-hash scan passed.
- Short-window resource samples: goroutines 34/33/33/33/33/35; heap 66.46→71.49 MiB while requests ran. This does NOT prove stable memory or absence of a leak. Extend observation to include GC cycles; do not present the short test's generous bound as a plateau.
- Third-pass privacy self-review reproduced a new generic-transaction callback-error leak into exception events. Fixed by recording a constant failure event while returning original/joined errors unchanged. Complete unit and three-consumer Race passed; overlaying pre-redaction production code makes the strengthened privacy regression test fail for the intended reason, without a compile failure. This one-field observability change needs a new source checkpoint/CI and deployable image.
- Next: new source build, longer isolated recovery/resource observation with GC counters, immutable image build/import, latest CI, final cluster rollout gates and full live acceptance. Goal remains in_progress.

### Immutable-image and rollout-preflight checkpoint

- Final code source 3fd4cb6fcb4983d3d548804f3964dc541c90cb54 pushed; CI 41053084557 and Fair Doudizhu 41053084529 both success. Latest make build metadata records that revision and vcs.modified=false.
- Repeated isolated recovery/resource test on that source with >3-minute observation and GC counters passed. Heap rose to ~75.10 MiB before GC, then dropped to ~3.15 MiB and settled at 2.50 MiB; GC count 3→7, goroutines ~33–34. This resolves the earlier short-window ambiguity, not a long-duration/capacity proof. Fixture-only outages and secret scans passed; all fixture containers/processes removed.
- Built existing Dockerfile from an exact git-archive context (server + Dockerfile only), with 1.5 GiB / 2-CPU builder limits; no local credentials/node_modules sent as context. Image awesome-zero-platform/app-api:goal0042-3fd4cb6fcb49; Docker config ID sha256:81e0fd668b276c59d8790944ecac4358f5906ff45f45eec858a134d126736dd1. Label pins source revision; amd64/nonroot/entrypoint verified. Embedded binary Go 1.25.8/go-zero 1.10.2 and shipped production template verified; binary SHA256 ccf2fca67b191e6354156581def60b0b32620102869fe8aeeacce1435cdd68d6.
- Image saved and imported into selected node containerd; crictl config ID matches. Old rollback image remains present. All backup hashes and isolated restore result rechecked.
- observability-server manifest updated only to this immutable image tag. Render/server-profile validator passed (28 objects). Real diff contains only API image/generation. Narrow image-only patch server dry-run matches original Deployment spec except that image; no other resource changes.
- Final preflight: intended context, both nodes Ready/no pressure, eight component Pods Ready, PVCs Bound, ~24 GiB /data free, zero active WS, Doudizhu disabled, 10-second profile unchanged, one replica/Recreate; original Deployment/PVC/ConfigMap and secret fingerprints preserved privately.
- Rollout completed after manifest checkpoint CI. The image-only API patch passed readiness and full live runtime acceptance; see the final live acceptance section below.


### Final live acceptance and residual limits

- Actual rollout on `kubernetes-admin@network-learning`: old `goal0038-gozero-first` image replaced with `awesome-zero-platform/app-api:goal0042-3fd4cb6fcb49`, config ID `sha256:81e0fd668b276c59d8790944ecac4358f5906ff45f45eec858a134d126736dd1`; source revision 3fd4cb6fcb4983d3d548804f3964dc541c90cb54. Pod `app-api-669ccbbc56-dqwr5` ran 2/2 Ready, zero restarts. Before/after Pod template diff proves only the image changed; replica count 1 and Recreate remain. Secret UID/resourceVersion/data fingerprints, PVC UID/specs, mounted 10-second configuration and disabled Doudizhu state were preserved.
- REST verified old opaque refresh continuity after rollout; health, login/logout, malformed/invalid/unauthorized/not-found requests, Admin reads, account creation/duplicate rollback, privilege denial, account profile update/cache invalidation, refresh rotation/replay rejection and Redis WATCH recovery. Two dedicated probe accounts were disabled and dedicated sessions revoked.
- WebSocket verified authentication, welcome, ping/pong, subscribe/unsubscribe, unknown/invalid envelope and JSON errors, server ping at ~25 seconds, close/reconnect and zero active-connection gauge after cleanup. Jaeger held actual HTTP-parent SQL/transaction/Redis/WATCH traces plus six independent WebSocket message traces with timing, references and bounded error statuses. Trace/request IDs matched one indexed Filebeat→Elasticsearch document and the browser-rendered Kibana Discover row.
- Prometheus target and live queries showed HTTP total count and status counters (including 400/401/403), request latency histogram, WebSocket messages/active connections, managed and native Redis pools, goroutine, CPU-time and resident-memory metrics. No 5xx sample was created; no production fault injection was used. A real Chromium browser captured all six populated Grafana dashboard panels and one correlated Kibana Discover result. Repository `make observability-smoke` passed after rollout.
- Post-rollout 4-minute online and ~2.7-minute quiet samples observed: quiet goroutine range 35–36, RSS ~32.5–32.8 MiB, heap ~3.5–5.6 MiB, GC count 332→338, active WebSockets 0 and native Redis pool 1; managed Redis pool stayed 9 in the online sample. Existing isolated same-source recovery had already shown fail-closed readiness and successful recovery after fixture-only MySQL/Redis pauses. These are bounded observations, not long-term capacity proof.
- Final read-only retention recheck passed: Elasticsearch green, 3-day log ILM, 12/8/5 GB watermarks, Prometheus 3-day/512 MiB, MySQL 3-day binlogs, Redis AOF/noeviction/128 MiB. Post-deployment status showed API/MySQL/Redis/Elasticsearch/Prometheus/Grafana/Jaeger/Kibana Ready, four PVCs Bound, nodes Ready, `/data` ~24 GiB available. Kubelet stats were used because Metrics API is absent. Node log proxy inspection found 317 historical control-plane soft-lockup lines on Oct 20/21/22/29, including multi-CPU stalls reported at 3702–3709 seconds on Oct 29; no matching soft/hard lockup, OOM or I/O errors were found in hk-worker logs. Current cluster nodes are Ready, but the cause and host-level mitigation remain unverified and outside this code rollout.
- Goal source/build/deployment CI was green before archive (IDs above). Full test and self-review detail and exact changed-file list are in the operations report. The archival commit needs its own final CI result before the user-facing completion claim.
