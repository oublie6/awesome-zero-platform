# Goal 0043: Jaeger Badger persistence (server overlay)

## Status

- State: ready
- Started:
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

- Goal scoped; v2.20.0 upstream Badger sample and server overlay design reviewed.

### In progress

- None.

### Remaining

- Manifest implementation, host preflight, live rollout, persistence verification, CI, archiving.

### Verification status

- Not started for the new persistence deployment.

## Completion Report

Not started.
