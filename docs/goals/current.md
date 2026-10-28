# Goal 0036: Kubernetes-Native Observability Deployment

## Status

- State: completed
- Started: 2026-10-29
- Completed: 2026-10-29
- Blockers: None for checked-in Kubernetes deployment artifacts; target-cluster access is unavailable.

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

- Preserved default app-only Kustomize base and added a sibling Kubernetes observability overlay for app-api Filebeat sidecar, Elasticsearch/Kibana, Prometheus/Grafana, Jaeger, native Services, ConfigMaps, resource limits, PVCs and ILM Job.
- Prometheus uses namespace-scoped Pod discovery and Role/RoleBinding instead of scraping a load-balanced API Service. Grafana admin password is supplied using a Kubernetes Secret; no credentials are committed.
- Added kubectl render/apply/status/port-forward/live smoke scripts and switched observability Makefile targets away from Docker Compose. The older Compose observability workflow remains manual-only for backward compatibility.
- Updated AGENTS.md, repository/architecture/server/deployment documentation and operational Kubernetes runbook. Cleaned up invalid nested Kustomize overlay and corrected Elasticsearch 9 vm.max_map_count to 1048576, Headless Services for StatefulSets and portable ES_SETTING_* variables.
- Confirmed final diff contains only Goal 0036 observability Kubernetes/source docs and CI changes. Existing API/DB/domain game logic was untouched.

### In progress

- None for repository implementation.

### Remaining

- On the user's intended Kubernetes cluster: verify image availability, StorageClass/PVC provisioning, existing MySQL/Redis and Secret, then run `make observability-up` and `make observability-smoke`. No kubeconfig or access to that cluster is available in this conversation. This is a deployment/operational handoff, not a claim of completed live installation.

### Verification status

- PASS: Kubernetes Kustomize render, 23 resource invariants, app-only base, RBAC/secret/Pod-discovery checks, shell syntax, and real Kind Kubernetes API server-side dry-run: https://github.com/oublie6/awesome-zero-platform/actions/runs/40739186344
- PASS: Main CI (Go unit, race, build, generated code, formatting, MySQL/Redis integration, Vue and crypto, historical production runtime regression): https://github.com/oublie6/awesome-zero-platform/actions/runs/40739518577
- PASS: Fair Doudizhu application, MySQL integration, signed manifest client: https://github.com/oublie6/awesome-zero-platform/actions/runs/40739520362
- NOT EXECUTED: Live K8s workload install, Pod/PVC readiness, Filebeat log indexing, Grafana metric retrieval and Jaeger traces on the user's cluster. Kind API server dry-run does not create running observability Pods.
- Historic prior Goal 0035 Docker Compose end-to-end verification is not evidence of this new Kubernetes installation.

## Completion Report

Delivered Kubernetes-native application observability deployment artifacts on main with tests and documentation; no Docker Compose is needed on the Kubernetes deployment path. The app-base default behavior remains unchanged. Exposed observability UIs are ClusterIP-only and accessed through localhost port-forward. The single-node Elasticsearch/Kibana implementation deliberately disables Elastic authentication/TLS and is appropriate only for a private development/integration cluster, not production.

Self-review checked that the new per-Pod Filebeat volume is isolated from host logs, non-root/read-only-root API constraints remain, no cluster-wide Prometheus RBAC exists, Elasticsearch/Prometheus StatefulSets have governing headless Services/PVCs, Grafana credentials are Secret-backed, Kustomize files have no circular references or duplicate resource names, and no application API or database schema was altered. The user must separately run and record the provided live Kubernetes smoke test on the actual cluster.

Source checked through commit `4c53e40ea3e9a0c93583d1ed3d6b0e45822f1f18` (documentation-only final source checkpoint) and the three passing GitHub Actions runs above.

