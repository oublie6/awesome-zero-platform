# Goal 0038: Codex supplementary Kubernetes deployment & runtime verification

This is a **restricted, environment-only Codex CLI handoff** after ChatGPT completes and archives Goal 0038. It does not transfer feature development, source fixes, schema changes, or general refactoring to Codex. Read [AGENTS.md](../../AGENTS.md), [server profile](../../deploy/kubernetes-overlays/observability-server/README.md), [prior runtime report](../goals/archive/0037-kubernetes-observability-deployment.md), and [go-zero First standard](../architecture/go-zero-first-components.md).

## Why supplementary execution is required

ChatGPT verified code and integration tests in GitHub Actions but has no SSH credentials, Kubernetes context, or access to the user's two-node integration cluster. Only an agent running on the user's authorized server can build/import the new container image, perform the controlled application rollout, and verify MySQL/Redis spans inside **the live Jaeger**.

## Scope of user-authorized actions

1. Before any action confirm an existing clean checkout on `main`, `git pull --ff-only`, finalized Goal 0038 archive, latest main SHA, and **successful** main CI and Fair Doudizhu CI on an appropriate final checkpoint. If the final code differs from the verified code, request verification.
2. Inspect current `kubectl config current-context`, `kubectl get nodes`, `kubectl -n awesome-zero-platform get deploy,pods,svc,pvc`, current app image/replica strategy, mounted ConfigMap, imagePullPolicy and events. The target is the existing **two-node** `network-learning` cluster, not the default Kustomize base. Document before state.
3. Preserve **one app replica / Recreate** and every existing MySQL, Redis, Elasticsearch, Prometheus volume, Secret and Service. No database reset, seed, data migration, PV deletion, namespace deletion, or other workload rollout. Detect active games/sessions and choose an approved maintenance window if disruption risk exists. Do not create extra app replicas.
4. The repository profile already declares `awesome-zero-platform/app-api:goal0038-gozero-first` in `deploy/kubernetes-overlays/observability-server/placement.yaml`. Build `deploy/docker/app-api.Dockerfile` from the verified main checkout; inspect architecture and available Docker/Podman/containerd tooling. Build and import image into **the node selected for app-api**, confirming tag, architecture and SHA/image digest. Do not rely on unverified GHCR access, :latest, or imagePullPolicy changes.
5. Explicitly select `OBS_KUSTOMIZE_OVERLAY="$PWD/deploy/kubernetes-overlays/observability-server"`. Render and inspect `make observability-config` / `kubectl diff -k "$OBS_KUSTOMIZE_OVERLAY"`. Do not blindly apply the generic two-replica base, nor `kubectl delete -k`. If the profile applies without unrelated resource drift, use the existing safe deployment mechanism or narrowly update the app Deployment to the **same reviewed image and profile**, preserve the existing ConfigMap/Secret/sidecar. Avoid rewriting other workload resource specs.
6. Wait for `kubectl -n awesome-zero-platform rollout status deployment/app-api`; verify `/health/live`, `/health/ready`, login/session refresh/revoke, normal Redis auth/policy actions, admin endpoints, WebSocket handshakes and current enabled business functions; confirm database and Redis contents preserved.
7. Run `make observability-status` and `make observability-smoke` with securely obtained existing Grafana credentials. Check real Elasticsearch logs, Prometheus app target, Grafana queries and Jaeger traces. **Generate a safe authenticated request that actually touches MySQL and Redis**, and in Jaeger verify a trace with HTTP parent, SQL and Redis child spans and meaningful timings. Distinguish cache hits that bypass MySQL. Verify no credential/hash, token or private payload leaks into SQL/Redis logs. Check resource pressure and warnings on both nodes.
8. Preserve old image in local containerd as rollback. On regression revert app-api to the exact previous image/deployment spec with the existing one-replica Recreate behavior, without PVC/secret/database mutations. Do not roll back observability databases or run destructive commands.
9. Provide evidence: Git SHA, image digest and node, baseline/current image, rollout and image inspection, ready Pods, app health, test requests, Jaeger traceID and list of child Span names/durations, Prometheus target/metrics, Kibana log query, errors and rollback results if needed. Redact secrets and tokens. Record results in a runtime report; **do not mark runtime acceptance passed without observing actual traces.**

**Allowed repository changes:** None to Go source, tests, AGENTS.md, archived goals, database schema, or platform architecture. Any deployment manifest edit beyond the pre-reviewed image tag must be treated as a blocker and handed back to ChatGPT. Follow the repository's authorized workflow for recording supplementary evidence (report/status only if there is a current goal), not independently redefining goals.

## Operational warning

`app-api` uses one replica and `Recreate` because active in-memory game state cannot be safely shared between two pods. An update briefly interrupts HTTP and WebSocket sessions. Existing server profile has a 10-second request deadline and may differ from the default app-api production file. The original version from Goal 0037 was `awesome-zero-platform/app-api:goal0037-74d2c01`. Preserve the configuration while updating only application source/image.

If the target cluster is unavailable, old image cannot be backed up, new image cannot be loaded, a `kubectl diff` reveals unrelated changes, or a live service would be disrupted without consent, stop and report exactly what is blocked. Do not guess credentials.

## Supplementary execution result

The authorized server rollout and runtime acceptance passed on 2026-10-30. See the [supplementary deployment acceptance report](goal0038-kubernetes-acceptance.md) for the deployed image digest, ready Pod, real HTTP/SQL/Redis Trace IDs, regression evidence, preserved resources and rollback readiness. The original source-level Goal archive remains unchanged.
