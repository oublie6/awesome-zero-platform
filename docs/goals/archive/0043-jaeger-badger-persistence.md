# Goal 0043: Jaeger Badger persistence (server overlay)

## Status

- State: completed
- Started: 2026-10-30
- Completed: 2026-10-30
- Blockers: none

## Goal

Replace the existing server-profile Jaeger v2.20.0 memory trace store with a single-instance, PVC-backed Badger trace store (72-hour span TTL), preserving existing OTLP/UI services and all application/business/data configuration.

## References

- `AGENTS.md`
- `docs/operations/logging-observability.md`
- `deploy/kubernetes-overlays/observability-server/README.md`
- `deploy/kubernetes-overlays/observability-server/kustomization.yaml`
- `deploy/kubernetes-overlays/observability/jaeger.yaml`
- Jaeger v2.20.0 upstream `cmd/jaeger/config-badger.yaml`.

## Deliverables

1. Add a Jaeger v2.20.0 Badger configuration, mounted exclusively in the server-specific observability overlay.
2. Add a Retain local PV / prebound RWO PVC on `hk-worker`, with a dedicated host path and no changes to existing PVs.
3. Override only the Jaeger Deployment with a single replica, configuration mount, Badger data mount, and safe restart behavior. Preserve existing service ports, probes and resources.
4. Document host-path provisioning/permissions and free-space preflight, restart-based same-TraceID persistence acceptance, retention observations and rollback.
5. Add automated manifest/config checks where appropriate; run existing Kubernetes validation/CI.

## Constraints

- Do not alter the default observability overlay or app-api deployment. Do not change Elasticsearch, MySQL, Redis, Secrets, images, or the existing volumes.
- Main only, clean working tree, `git pull --ff-only` before execution. No feature branches.
- A local PV request is NOT a hard disk quota. Confirm space on hk-worker and provision the exact mount directory with appropriate UID/GID before applying changes. Never chmod 777.
- Never delete a Retain PV/PVC or preexisting trace storage without explicit approval. The old in-memory traces will not migrate automatically.
- Before rollout confirm sufficient writable storage and a rollback plan. Stop and mark blocked on missing preconditions. A Jaeger restart may briefly interrupt tracing ingestion.
- Trace persistence is not verified by YAML alone; make a known test trace, recreate the Pod while retaining the PVC, and query the exact TraceID again.
- Do not claim production trace durability, high availability, or exact 72h disk reclamation from TTL setting alone.

## Acceptance Criteria

- Rendered server overlay includes Jaeger v2 Badger config, dedicated prebound local PV/PVC, one replica, unchanged OTLP 4317/4318 and UI 16686; default overlay remains untouched.
- K8s rendered manifest validates; Jaeger 2.20.0 accepts full configuration and starts Ready.
- PVC Bound to correct hk-worker local path; Badger writable with restricted permissions and nonroot container user.
- Same TraceID and spans are queryable after Pod recreation, plus new traces still ingest.
- 72h TTL and storage usage observations documented; logs/metrics/App API remain healthy.
- CI passes, real environment evidence and outstanding risks recorded; only then archive goal and reset current.md.

## Agent Strategy

Single primary executor with sequential file edits and mandatory self-review.

## Working State

### Completed

- Server-only Badger configuration, Retain local PV/PVC, Jaeger Deployment patch, and rollout/rollback documentation are committed through `ef5d7b59`.
- Confirmed cluster context `kubernetes-admin@network-learning`; both `hk-worker` and the control-plane node Ready. Existing app-api, MySQL, Redis, Elasticsearch, Prometheus, Grafana, Kibana and their PVCs were Ready/Bound before rollout.
- Read-only worker inspection showed `/data` on the 97 GiB `/dev/vda1` root filesystem, not a separate disk; 73 GiB was free, with target parent/path absent. Created only `/data/awesome-zero-platform/jaeger`, owned by `10001:10001`, mode `0750`; UID/GID 10001 write test passed. Kubernetes fsGroup mounted the directory as `10001:10001` mode `2770` with Badger subdirectories mode `2770`.
- Jaeger v2.20.0 accepted the complete config in an isolated non-root Pod and started Ready with Badger `InMemory:false`; the server overlay deployed with Jaeger one replica/Recreate, unchanged Service ports 16686/4317/4318, and a bound 2 Gi RWO Retain PV/PVC on `hk-worker`.
- Generated a real trace using HTTP login and refresh requests with one W3C TraceID. Before restart it had four spans: `/auth/login` -> SQL, and `/auth/refresh` -> Redis. After deleting only Jaeger Pod, the same TraceID and all four spans had an identical semantic fingerprint. A second SQL/Redis trace was ingested after restart.
- `make observability-smoke` passed before and after Pod recreation. REST readiness returned 200; the unauthenticated WebSocket handshake remained rejected with 401. Existing application Pod/generation and Secret resourceVersion remained unchanged.
- Added explicit server-profile manifest assertions for Badger config, TTL, ports, PV/PVC placement, permissions, mounts, replicas, restart strategy, and preserved Jaeger resources. Local default/server manifest checks and server-side dry-run passed.

### In progress

- None. Runtime acceptance, final self-review, and required GitHub Actions workflows are complete.

### Remaining

- Archive this completed goal and push the final report checkpoint.

### Verification status

- Jaeger v2.20.0 runtime configuration: passed; Badger initialized on both initial deployment and replacement Pod.
- PVC persistence: passed with same TraceID, four spans and identical semantic fingerprint across Pod recreation; replacement Pod Ready on `hk-worker`, same PVC UID and PV.
- Storage observation: 97 GiB `/dev/vda1`, 72.24 GiB free at latest sample; Badger allocated size `du -sh` 200 KiB while apparent size was 2.2 GiB (sparse vlog files); kubelet volume accounting reported approximately 24.5 MiB. The PV 2 Gi request is not a disk quota.
- Jaeger requests/limits: 50m/128 MiB and 1 CPU/512 MiB; latest observed CPU about 24.6m, working set 13.9 MiB, restart count 0.
- Smoke, default overlay (23 resources), server overlay (30 resources), Python syntax, shell syntax, post-deploy zero-diff and Git diff checks: passed.
- GitHub Actions for commit `d6632e78e5c98347cbcbc8adf84c20c8cfca01e7`: CI run `41096176612` passed; Kubernetes Observability run `41096176514` passed; Fair Doudizhu run `41096176603` passed, including application unit tests, race tests, vet, MySQL integration, and manifest client checks.
- TTL is configured as 72h; physical expiry/GC and long-term node disk growth cannot be verified in this short run. No off-node backup is configured.
## Completion Report

### Implementation and rollout

- The server observability overlay now mounts the Jaeger v2.20.0 Badger configuration and the dedicated local PV/PVC. Jaeger runs as one replica with `Recreate`, non-root UID/GID 10001, and the existing UI/OTLP service ports 16686/4317/4318. The default overlay and application/business deployments were not changed.
- Provisioned `/data/awesome-zero-platform/jaeger` on `hk-worker`, owned by `10001:10001` with mode `0750`; the UID 10001 write test succeeded. The 2 Gi RWO PVC bound to the Retain PV `awesome-zero-jaeger`, with node affinity to `hk-worker`. The path is on the node's 97 GiB `/dev/vda1` root filesystem, not a separate disk; 72.24 GiB was free at the latest sample. PV capacity is not a quota, so node disk monitoring remains necessary.
- Jaeger accepted the complete configuration and started Ready with Badger `InMemory:false`. The configured span TTL is 72 hours. Badger's files occupied about 200 KiB allocated (`du`) while sparse files represented 2.2 GiB apparent size; kubelet volume accounting was about 24.5 MiB. Latest observed Jaeger use was about 24.6m CPU and 13.9 MiB working set, with zero restarts.

### Persistence acceptance

- Queried TraceID `b1b88dc41e01b05a7f1577ceb55e5072` before Pod recreation and again after deleting only the Jaeger Pod. The same four spans remained queryable with the same semantic fingerprint `aa7879e38b3ebbbdc657ee2367750f9dcbda351cde0618ad3c29e2435c7a26f7`: `/auth/login` with its SQL child and `/auth/refresh` with its Redis child. The replacement Pod was Ready on `hk-worker` and reused the same PVC UID and PV. A new four-span SQL/Redis trace (`cf8632b1bca0deb36193931fab2004e4`) was also queryable after recreation.
- `make observability-smoke` passed before and after recreation. App readiness returned HTTP 200. The live unauthenticated WebSocket handshake returned the expected 401; an authenticated WebSocket session was not exercised directly against the cluster because no safe existing test credential was available. The Fair Doudizhu CI workflow passed its application, race, vet, MySQL, and manifest-client jobs.
- Existing application Pod and Deployment generation remained unchanged, the observability Secret resourceVersion remained unchanged, and post-deploy `kubectl diff -k deploy/kubernetes-overlays/observability-server` was empty. Existing PV/PVCs and business data were left untouched.

### Verification and review

- Local checks passed: default overlay render/validation (23 resources), server overlay render/validation and Goal 0043 assertions (30 resources), Python and shell syntax, server-side dry-run, `git diff --check`, and post-deploy zero diff.
- GitHub Actions passed for implementation commit `d6632e78e5c98347cbcbc8adf84c20c8cfca01e7`: CI `41096176612`, Kubernetes Observability `41096176514`, and Fair Doudizhu `41096176603`.
- Mandatory primary-agent self-review re-read the goal and referenced architecture/operations docs, reviewed the final implementation and tests, and checked server-only scope, unchanged ports/resources, PVC/PV binding and Retain behavior, local-disk capacity caveat, non-root ownership/modes, configuration acceptance, exact TraceID continuity, replacement-Pod lifecycle, rollback path, and test assertions. No defect was found.

### Remaining risks

- A 72-hour TTL setting does not prove exact physical reclamation time. Long-term Badger GC behavior and disk growth require monitoring; no off-node backup or high availability is configured. The local path shares the worker root filesystem and has no hard quota.
- Authenticated WebSocket behavior was not directly tested against the live cluster. The existing Kubernetes smoke check, live readiness check, and Fair Doudizhu CI passed.
