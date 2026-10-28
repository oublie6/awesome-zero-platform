# Kubernetes logs, metrics and tracing

**Primary deployment:** Kubernetes Kustomize. This guide is for an isolated, private **development / integration** cluster. No Docker Compose deployment is required for this observability stack.

## Architecture

| Signal | Kubernetes flow | UI (via localhost port-forward) |
| --- | --- | --- |
| Logs | `app-api` JSON file logs -> **per-Pod Filebeat sidecar** (shared `emptyDir`) -> Elasticsearch -> Kibana | `127.0.0.1:15601` |
| Metrics | existing go-zero HTTP/WS `/metrics` -> **Prometheus Kubernetes Pod discovery** -> Grafana | `127.0.0.1:13000` |
| Traces | go-zero OTLP/gRPC -> internal `jaeger:4317` -> Jaeger v2 | `127.0.0.1:16686` |

Kustomize base: `deploy/kubernetes/` (existing app, **without observability**). Optional overlay: `deploy/kubernetes-overlays/observability/`. Running `kubectl apply -k deploy/kubernetes-overlays/observability` adds the collectors and the `app-api` sidecar/configuration; it does not change application APIs or business rules.

The `app-api` Deployment currently requests **two replicas**. A Filebeat sidecar is attached to **each Pod**, and Prometheus discovers **each Pod individually** with a namespaced read-only Role (get/list/watch Pods). Scraping a load-balanced Service directly would mix separate counters and is deliberately avoided.

## Prerequisites — important

- A working Kubernetes cluster and `kubectl` access to the **intended context**, with permission to create namespaced objects, Role/RoleBinding, PVC, Secret and a Job.
- The existing `app-api` image from `deploy/kubernetes/base.yaml` must be available to cluster nodes. The manifests do **not** build or publish container images.
- Working `mysql:3306` and `redis:6379` Services, existing schema, and an `awesome-zero-platform-secrets` Secret with the keys already referenced by `base.yaml`. This overlay does **not** install/replace MySQL or Redis.
- A default dynamic StorageClass that can provision ReadWriteOnce PVCs (Elasticsearch **8 GiB**, Prometheus **3 GiB**) and appropriate persistent-volume backup procedures. The Filebeat registry and app logs are ephemeral per Pod. Grafana's preprovisioned dashboards survive restarts, but unprovisioned UI state uses an ephemeral local directory.
- For Elasticsearch the Kubernetes node(s) must meet `vm.max_map_count >= 262144`; configure at the **node** according to your Kubernetes distribution and security policy. Do not add a privileged init container simply to change a host sysctl.
- A comfortably sized **dedicated test cluster** (plan for roughly 8–12 GiB available RAM plus Kubernetes and existing DB/Redis/app overhead; check actual requests/limits and disk/CPU). This stack is **not suitable for a 1–2 GiB server**.
- Never enable multi-replica Doudizhu's current in-memory active game runtime without addressing its existing single-process state and routing constraints; this overlay does not make live hands distributed.

## Render, deploy and verify

First inspect the selected cluster:

```bash
kubectl config current-context
kubectl get storageclass
kubectl -n awesome-zero-platform get svc mysql redis
kubectl -n awesome-zero-platform get secret awesome-zero-platform-secrets
```

Supply a **new secret** for the Grafana admin; do not commit it:

```bash
read -rsp 'Grafana development password: ' OBS_GRAFANA_ADMIN_PASSWORD
echo
export OBS_GRAFANA_ADMIN_PASSWORD

# No cluster mutation: inspect fully resolved YAML, including the app-api patch.
bash scripts/k8s-observability.sh render > /tmp/awesome-observability.yaml

# Installs/updates namespaced observability resources and applies the app overlay.
bash scripts/k8s-observability.sh apply
bash scripts/k8s-observability.sh status

# Runs a real end-to-end probe with temporary localhost-only port-forwards.
bash scripts/k8s-observability.sh smoke
```

The `apply` command first verifies that the platform application Secret exists, then creates/updates a separate namespaced `observability-secrets` Secret from stdin (not from a tracked YAML file), and applies the Kustomize overlay. It may trigger an `app-api` rolling restart because an app container environment/volume and Filebeat sidecar are added. Plan this as a normal application rollout; active **in-memory** games may be interrupted. It does not change MySQL volumes or schemas.

The smoke checks the actual indexed request ID, Kibana data view, Prometheus healthy `app-api` target, Grafana datasource and Jaeger OTLP-reported service. A static YAML check alone does not establish runtime success.

In another terminal, for dashboards:

```bash
bash scripts/k8s-observability.sh forward
```

Then visit these **localhost-only** addresses on the machine running `kubectl`:

| Tool | Address |
| --- | --- |
| Kibana Discover | `http://127.0.0.1:15601` |
| Grafana (user `admin`) | `http://127.0.0.1:13000` |
| Jaeger | `http://127.0.0.1:16686` |
| Prometheus targets | `http://127.0.0.1:19090/targets` |
| Elasticsearch API | `http://127.0.0.1:19200` |

The Kibana `awesome-logs-*` data view is set up by the smoke script, not by clicking buttons. Check `module`, `requestId`, `eventAction` and `statusCode`. The Grafana `Platform / Awesome Zero Platform Overview` dashboard includes HTTP requests, error rate, latency, process memory and WebSocket activity. In Jaeger, select service `app-api`.

### Diagnostics

```bash
kubectl -n awesome-zero-platform get pods,svc,pvc
kubectl -n awesome-zero-platform rollout status deployment/app-api
kubectl -n awesome-zero-platform logs deployment/app-api -c filebeat --tail=100
kubectl -n awesome-zero-platform logs job/elastic-bootstrap --tail=100
kubectl -n awesome-zero-platform logs deployment/jaeger --tail=100
kubectl -n awesome-zero-platform get pod -l app.kubernetes.io/name=elasticsearch
```

If Elasticsearch remains Pending, inspect PVC provisioning/StorageClass and node memory / `vm.max_map_count`. If Grafana is not Ready, confirm `observability-secrets` contains key `grafana-admin-password`. If no log is indexed, inspect Filebeat sidecar and `app-api` file path. Do not solve permissions problems by mounting the Docker socket or the node's entire log directory.

## Retention, risk and production boundaries

- Elasticsearch **single node**, security deliberately disabled for development, exposed only by a `ClusterIP` Service. Kibana is also unauthenticated. No Ingress, LoadBalancer or NodePort is created; `kubectl port-forward` is restricted to 127.0.0.1. **Other workloads inside the cluster may reach these Services**, so this is **not production-secure**. For production use dedicated authentication/TLS, network isolation/NetworkPolicy where supported, appropriate RBAC and storage/snapshot lifecycle.
- Elasticsearch uses a seven-day index lifecycle for `awesome-logs-*`. Prometheus retains seven days or 1 GB, whichever limit is reached first. Jaeger all-in-one uses **in-memory trace storage**, so restart loses spans. Other stack indexes may have independent retention.
- Per-Pod `emptyDir` holds bounded log files and Filebeat state. Both are **lost when the Pod is replaced**, and logging and collection occur independently; logs not yet shipped before termination can be lost. This is acceptable for a development stack, **not a durability guarantee**.
- The application still has bounded-route Prometheus labels; logs should not include request query strings, passwords, refresh/access tokens, private keys, sensitive game cards or random secret contributions. Keep trace sampling and data retention proportional to load.
- No Kafka/go-stash has been introduced, and none of these three systems is required to start the default Kubernetes base (without this overlay).

## Disable without deleting business state

**Never run `kubectl delete -k deploy/kubernetes-overlays/observability`**: that built configuration also contains the base application's Namespace/Deployment/Service and would delete the platform namespace.

For removal, first review any customizations to the `app-api` Deployment (the base file includes its own image/replicas/resources), then reapply the unmodified base and remove only the observability resources:

```bash
kubectl apply -k deploy/kubernetes
NS=awesome-zero-platform
kubectl -n "$NS" delete deployment kibana grafana jaeger --ignore-not-found
kubectl -n "$NS" delete statefulset elasticsearch prometheus --ignore-not-found
kubectl -n "$NS" delete job elastic-bootstrap --ignore-not-found
kubectl -n "$NS" delete service kibana grafana jaeger elasticsearch prometheus --ignore-not-found
kubectl -n "$NS" delete configmap obs-filebeat obs-prometheus obs-grafana-datasources obs-grafana-providers obs-grafana-dashboard --ignore-not-found
kubectl -n "$NS" delete rolebinding observability-prometheus-pods --ignore-not-found
kubectl -n "$NS" delete role observability-prometheus-pods --ignore-not-found
kubectl -n "$NS" delete serviceaccount observability-prometheus --ignore-not-found
```

StatefulSet PVCs and observability Secret are **intentionally retained**; review and delete them separately only after checking data-retention/backup requirements. The default base does not configure Filebeat, Jaeger, Prometheus, Grafana or ELK.

## Validation evidence

`Kubernetes Observability Manifests` GitHub Actions executes `kubectl kustomize`, checks the base remains unchanged, parses 23 K8s resources, validates the security/Pod discovery boundaries, dashboard JSON and script syntax **without a live cluster**. Follow the linked CI run for the exact evidence. This **does not prove** images can be pulled, PVCs bound or collectors run on your particular Kubernetes cluster; the `smoke` command above is the live acceptance test.

The previously tested Compose implementation is kept only for backward compatibility in the repository; this guide and new deployment implementation are Kubernetes-native. Earlier Docker Compose success is not a claim that the Kubernetes deployment has been exercised on a real cluster.
