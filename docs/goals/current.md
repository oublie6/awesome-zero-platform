# Goal 0036: Kubernetes-Native Observability Deployment

## Status

- State: in_progress
- Started: 2026-10-29
- Completed:
- Blockers:

## Goal

Deploy previously implemented three-signal observability using native Kubernetes resources rather than Docker Compose: application JSON logs -> per-Pod Filebeat -> Elasticsearch/Kibana, existing Prometheus metrics -> Prometheus/Grafana, and go-zero OTLP traces -> Jaeger. Keep normal K8s application behavior when observability overlay is absent.

## References

- `AGENTS.md`
- `deploy/kubernetes/base.yaml`
- `docs/operations/logging-observability.md`
- `docs/architecture/module-boundaries.md`

## Deliverables

1. Kubernetes Kustomize overlay for platform Deployment: opt-in file logger, Filebeat sidecar, per-Pod ephemeral application-log volume, OTLP endpoint, preserved non-root/read-only-root restrictions.
2. Kubernetes Services/ConfigMaps/PVCs/Jobs/Deployments for development-only single-node Elasticsearch, Kibana, Prometheus, Grafana and Jaeger, with bounded data retention and pre-provisioned dashboards/data sources.
3. Secure-by-default public surface: ClusterIP only, no Ingress/NodePort, Grafana admin password from existing Kubernetes Secret; explicit warnings that dev Elastic disables auth/TLS.
4. K8s-native rollout/port-forward/smoke tests and CI validation; no dependency on Docker Compose for this deployment path.
5. Update project docs and reduce obsolete Compose-based observability guidance/CI while leaving ordinary legacy application production Compose unchanged.

## Constraints

- No application API changes, game state/DB changes, Kafka, go-stash, or microservice extraction.
- No assumption that running a K8s YAML file starts the application's required MySQL/Redis (pre-existing dependency contract).
- Elasticsearch/Kibana dev settings are never exposed publicly. Do not commit secrets.
- Do not claim a full live K8s observability test without running a real cluster.

## Acceptance Criteria

- Kustomize build parses without unresolved resource/patch reference or duplicate resources; Kubernetes schema/API validation passes if tooling is available.
- Existing Go and game CI suites pass.
- Observability opt-in does not modify default `deploy/kubernetes/base.yaml` behavior.
- Deployment and smoke docs include prerequisites (existing MySQL/Redis/Secret/storage class), per-Pod log behavior, port forwarding, explicit data/trace/metrics assertions, disabling and cleanup.
- All available checks pass, or Goal remains in_progress/blocked with exact limits.

## Working State

### Completed

- Reviewed current Kubernetes base Deployment, Compose-only observability resources and existing three-signal runtime acceptance.

### In progress

- Kubernetes manifests and testing adaptation.

### Remaining

- Verify validation, update docs/CI, record outcome.

### Verification status

- Pending.

## Completion Report

Pending.
