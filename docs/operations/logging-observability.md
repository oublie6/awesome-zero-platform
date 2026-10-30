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
- For Elasticsearch 9.1 the Kubernetes node(s) must meet `vm.max_map_count >= 1048576`; configure at the **node** according to your Kubernetes distribution and security policy. Do not add a privileged init container simply to change a host sysctl.
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

### Cloud server access through an SSH tunnel

For the confirmed server installation, keep the existing ClusterIP Services and
use two forwarding steps: Kubernetes Service to the server's loopback address,
then SSH to your own computer. No dashboard needs a public listener.

**Security group:** allow inbound TCP **22** (or the existing custom SSH port)
from your computer's current public IPv4 address with a `/32` source mask. For
IPv6, restrict access to your address with `/128`. Reuse an existing trusted SSH
connection if it already works. Dashboard access does not require public rules
for 3000, 5601, 9090, 9200, 16686, OTLP 4317/4318, MySQL 3306, Redis 6379 or the
Kubernetes API 6443. These are browser-access instructions, not a replacement
for the cluster's existing internal networking rules.

On the **server with the configured kubectl context**, run:

```bash
cd /root/awesome-zero-platform
kubectl config current-context
make observability-forward
```

For the confirmed deployment the context is
`kubernetes-admin@network-learning`. Keep this terminal running. The command
only forwards Services to `127.0.0.1`; it does not apply manifests, change
replicas or need the Grafana password.

In another terminal on **your own computer**, run:

```bash
ssh -N -o ExitOnForwardFailure=yes \
  -L 127.0.0.1:15601:127.0.0.1:15601 \
  -L 127.0.0.1:13000:127.0.0.1:13000 \
  -L 127.0.0.1:16686:127.0.0.1:16686 \
  -L 127.0.0.1:19090:127.0.0.1:19090 \
  -L 127.0.0.1:19200:127.0.0.1:19200 \
  trusted-ssh-host
```

Replace `trusted-ssh-host` with your existing SSH configuration alias or
`username@server-address`; use your existing key and host verification. Add
`-p YOUR_SSH_PORT` before the host argument if your connection uses a custom port. Keep this terminal
running too. All local listeners are explicitly bound to `127.0.0.1`.

Open the localhost addresses in the table above in **your computer's browser**:

- Kibana: open Discover and select the `Awesome Logs` data view.
- Grafana: log in as `admin` with the existing securely stored deployment
  password, then open `Platform / Awesome Zero Platform Overview`.
- Jaeger: select service `app-api` and search for traces.
- Prometheus: use `/targets` to check collection or the query page for PromQL.
- Elasticsearch: the URL is a REST API returning JSON, not a separate dashboard.

Kibana, Elasticsearch, Prometheus and Jaeger in this development installation
have no separate login protection; keep their access behind the SSH tunnel.
Do not bind the Kubernetes forwards to `0.0.0.0`. Close both forwarding commands
with `Ctrl+C` when finished; this does not stop the Kubernetes workloads.

If SSH reports a local port is already in use, stop the conflicting local
forward or choose another local port. For example,
`-L 127.0.0.1:23000:127.0.0.1:13000` makes Grafana available at
`http://127.0.0.1:23000`. After an app/backend Pod replacement or a disconnected
SSH session, restart the affected forwarding command if access stops working.

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
- Elasticsearch uses a seven-day index lifecycle for `awesome-logs-*`. Prometheus retains seven days or 1 GB, whichever limit is reached first. The default observability overlay keeps Jaeger traces in memory, so its restart loses spans. The confirmed server-specific profile uses Badger/PVC storage with a configured 72-hour span TTL; see the [Goal 0043 completion report](../goals/archive/0043-jaeger-badger-persistence.md). Other stack indexes may have independent retention.
- Per-Pod `emptyDir` holds bounded log files and Filebeat state. Both are **lost when the Pod is replaced**, and logging and collection occur independently; logs not yet shipped before termination can be lost. This is acceptable for a development stack, **not a durability guarantee**.
- The application still has bounded-route Prometheus labels; logs should not include request query strings, passwords, refresh/access tokens, private keys, sensitive game cards or random secret contributions. Keep trace sampling and data retention proportional to load.
- No Kafka/go-stash has been introduced, and none of these three systems is required to start the default Kubernetes base (without this overlay).

## Confirmed server installation profile

Goal 0037 uses `deploy/kubernetes-overlays/observability-server/` for its confirmed
shared two-node cluster. Set `OBS_KUSTOMIZE_OVERLAY` to its absolute path before
`make observability-config` or `make observability-up`. This profile pins one
API replica with Recreate and a local image, places workloads explicitly, and
uses dedicated prebound Retain local PVs with a non-default StorageClass. **Do
not reapply the default overlay/base to that installation:** it would restore
the default two API replicas and image. See the [server profile](../../deploy/kubernetes-overlays/observability-server/README.md).

On that 40 GiB data disk the server profile shortens log ILM to **3 days**, adds
12/8/5 GB free-space Elasticsearch watermarks, retains Prometheus data for
**3 days or 512 MiB**, and retains MySQL binlogs for **3 days**. Local PV capacity
is not a filesystem quota. Durable database/Redis business data is never
purged by these policies. Monitor actual disk use and arrange off-node backups.
The default development overlay's seven-day policy remains unchanged.

## Disable without deleting business state

**Never run `kubectl delete -k deploy/kubernetes-overlays/observability`**: that built configuration also contains the base application's Namespace/Deployment/Service and would delete the platform namespace.

For removal, first prepare a reviewed application manifest from the live
`app-api` Deployment. Preserve its exact image, replica count, business settings
and any server-specific configuration mount. Remove only the Filebeat sidecar,
observability environment and their unused volumes/mounts. Apply that reviewed
manifest during a maintenance window: recreating a Pod can interrupt in-memory
games. **Do not use a generic base reapply as rollback**; the base's image and
two replicas may differ from the running installation.

After checking resource ownership, remove only the named observability objects:

```bash
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

`Kubernetes Observability Manifests` GitHub Actions executes `kubectl kustomize`, checks the base remains unchanged, parses 23 K8s resources, validates the security/Pod discovery boundaries, dashboard JSON and script syntax, then starts a disposable Kind cluster and performs **Kubernetes API server-side dry-run admission** of the complete overlay. See [the passing CI run](https://github.com/oublie6/awesome-zero-platform/actions/runs/40739186344). Kind's dry-run checks Kubernetes API acceptance but **does not start the workload containers**. It does not prove images can be pulled, PVCs bound or collectors run on your actual cluster; the `smoke` command above is the live acceptance test.

Goal 0037 subsequently deployed this stack on the confirmed two-node cluster
and passed actual request-ID indexing/Discover, per-Pod metrics/Grafana panels,
request-correlated traces and controlled all-backend outage/recovery tests.
See the [deployment acceptance report](../goals/archive/0037-kubernetes-observability-deployment.md)
for image/resource inventory, retention, evidence, CI and remaining host risks.

The previously tested Compose implementation is kept only for backward compatibility in the repository; this guide and new deployment implementation are Kubernetes-native. Earlier Docker Compose success is not a claim that the Kubernetes deployment has been exercised on a real cluster.
