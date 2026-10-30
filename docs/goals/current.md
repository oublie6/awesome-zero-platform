# Goal 0042: go-zero architecture final deployment and runtime acceptance

## Status

- State: in_progress
- Started: 2026-10-30
- Completed:
- Blockers:

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

- Read AGENTS/current goal and requested deployment/architecture references completely.
- Clean main fast-forwarded from 890abc8 to 698d760; Goals 0039–0041 archived completed.
- Live context kubernetes-admin@network-learning: both nodes Ready; API and observability/database Pods Ready; four PVCs Bound. API still uses goal0038-gozero-first and does not establish deployment of latest source.
- Local /data: 26 GiB available; 4.8 GiB RAM available, no swap. gh/node/npm absent; Docker/containerd/crictl/Go available. CI must be checked through supported GitHub tools; frontend tooling needs resolution.

### In progress

- Race verification of observability corrections; backup restore validation and remaining architecture/runtime gates.

### Remaining

- Resolve/verify module-data ownership findings; complete remaining checks and self-review; validate backups in isolation; build/import/roll out verified source; run full live acceptance, report and final CI.

### Verification status

- Initial read-only cluster inventory passed; no rollout, fault injection or live observability acceptance performed yet.
- make fmt-check passed. Both Kubernetes render validators passed after correcting invocation to supply rendered YAML (28 server-profile resources, 23 default-observability resources). Initial invocation without the required file argument failed with IndexError; no repository defect inferred.
- make test could not start with system Go 1.18.1 (go.mod requires 1.25.8); downloading isolated matching toolchain, then rerun. No test pass claimed.
- GitHub REST Actions inventory confirms synchronized source 698d7608cbecfb7c690a035b7675e5edc9ec0e3e: CI 41022280151 success, Fair Doudizhu 41022280106 success. New definition checkpoint 399703c triggered live CI 41043134748 and Fair Doudizhu 41043134806, not complete at observation. Short SHA query returned no runs; use full SHA/inventory.
- Goal definition committed and pushed directly to main: 399703c.

## Completion Report

Not complete. This executor made narrowly scoped observability corrections with regression coverage; no Deployment, PVC, schema or production business state was reset. A dedicated acceptance login/session was created for upgrade compatibility testing. Backups require isolated restore validation before rollout.

### Review and progress checkpoint (2026-10-30)

- Code findings/fixes: native SQL transaction and WATCH error events lacked error status; transaction panics lacked failure status; accepted WebSocket failures lacked status; outer access logs lost inner go-zero route trace context; arbitrary HTTP extension methods could grow Prometheus label cardinality. Fixed failure statuses/correlation and bounded extension methods without changing response envelopes, transaction ordering, panic/error identity, JWT/session semantics or standard metric labels.
- Added deterministic regression coverage for transaction begin/callback/commit/rollback/panic outcomes, Redis WATCH dial failure, actual go-zero route middleware/access-log correlation, WebSocket failures and private handler-error redaction, and extension-method metric cardinality. Existing and new tests passed in the full sequential unit suite, including architecture/route-inventory tests. Related five-package focused run passed. Race suite is running (not yet passed).
- Distinct self-review of this patch checked deferred rollback/commit-before-span-end, original panic and error identity, unchanged WebSocket response IDs/codes, no payload attributes, outer 404/405 coverage, no duplicate HTTP root spans, unchanged standard metric names/labels and absence of new module dependencies. Removed an unbounded receive from the new WebSocket test. This is self-review, not independent review; final review remains required after any further changes.
- Go 1.25.8 and Node 22.23.4 downloaded into /opt/goal0042-toolchain and SHA256-verified against their official release indexes. Initial Python Node extraction rejected an npm symlink; extraction with tar succeeded. go mod tidy passed; it correctly promoted the already-selected OpenTelemetry v1.40.0 root module to a direct dependency, with no version change.
- Initial full test invocation was stopped because imports were edited after dependency discovery, producing invalid import-scan results. Fresh focused and full runs subsequently passed; logs retained under /tmp/goal0042-*.log. No failure was bypassed by changing assertions.
- Deployment diff before changes is empty. Both node architectures amd64. Metrics API is absent, so kubectl top fails; kubelet stats/summary succeeded for both nodes and raw evidence is saved. At observation API working set ~81 MB, MySQL ~247 MB, Redis ~10 MB, Elasticsearch ~1.12 GB, Prometheus ~37 MB; second-node observability ~778 MB combined. CPU samples during compilation are not steady-state load measurements.
- Private evidence directory: /root/.local/state/awesome-zero-platform/goal0042 (0700). Created MySQL consistent single-transaction dump (all 15 tables InnoDB), Redis replication RDB snapshot, old image export, manifest SHA256 checksums and original Deployment. No restore/rollback execution claimed yet. Existing admin health/login/session/permission baseline passed; dedicated session retained for upgrade.
- GitHub Actions on 399703c: CI 41043134748 and Fair Doudizhu 41043134806 completed success. New source fixes still need their own CI after checkpoint push.
- Open architecture findings: casbinmysql/active_superadmin.go directly lock-reads identity_accounts for the last-active-super-admin invariant; admin/mysqlstore directly manages authorization catalog tables. Existing architecture tests only forbid platform-to-business SQL, so their green result does not prove platform-to-platform ownership isolation. Native authorization transactions and context-less Casbin callbacks also require explicit tracing review. These are NOT marked passed or silently removed to satisfy checks.
- Frontend/generation/API-tool checks, isolated integration/backend recovery, full live metrics/traces/ELK/WS/reliability acceptance, production rollout and final CI remain outstanding.
