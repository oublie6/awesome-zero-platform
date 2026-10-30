# Goal 0043: Jaeger Badger persistence (server overlay)

## Status

- State: in_progress
- Started: 2026-10-30
- Completed:
- Blockers:

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

- Final self-review, final report/goal archive, and GitHub Actions verification for the manifest-test update.

### Remaining

- Commit and push the test/report checkpoint, confirm its Kubernetes Observability CI run passes, then archive Goal 0043 and restore the current-goal template.

### Verification status

- Jaeger v2.20.0 runtime configuration: passed; Badger initialized on both initial deployment and replacement Pod.
- PVC persistence: passed with same TraceID, four spans and identical semantic fingerprint across Pod recreation; replacement Pod Ready on `hk-worker`, same PVC UID and PV.
- Storage observation: 97 GiB `/dev/vda1`, 72.24 GiB free at latest sample; Badger allocated size `du -sh` 200 KiB while apparent size was 2.2 GiB (sparse vlog files); kubelet volume accounting reported approximately 24.5 MiB. The PV 2 Gi request is not a disk quota.
- Jaeger requests/limits: 50m/128 MiB and 1 CPU/512 MiB; latest observed CPU about 24.6m, working set 13.9 MiB, restart count 0.
- Smoke, default overlay (23 resources), server overlay (30 resources), Python syntax, shell syntax, post-deploy zero-diff and Git diff checks: passed.
- Existing CI for `ef5d7b59` was green. CI for the newly added manifest assertions is pending the next push.
- TTL is configured as 72h; physical expiry/GC and long-term node disk growth cannot be verified in this short run. No off-node backup is configured.
## Completion Report

Not started.
