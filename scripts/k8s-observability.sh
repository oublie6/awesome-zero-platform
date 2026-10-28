#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OVERLAY="$ROOT/deploy/kubernetes-overlays/observability"
NAMESPACE=awesome-zero-platform

usage() {
  cat <<'EOF'
Usage: bash scripts/k8s-observability.sh <render|apply|status|smoke|forward>
Requires kubectl configured for the intended cluster.
  render   Build opt-in Kustomize manifest without altering the cluster.
  apply    Install/update the Kubernetes development observability overlay.
           Requires preexisting application MySQL/Redis and platform Secret.
           Requires OBS_GRAFANA_ADMIN_PASSWORD (not committed).
  status   Show rollouts, Services and persistent storage.
  smoke    Run end-to-end request log, Prometheus, Grafana and Jaeger checks.
  forward  Port-forward localhost-only UIs (Ctrl-C stops all).
Removal: see docs/operations/logging-observability.md.
EOF
}

ensure_namespace() {
  kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
}

secret() {
  : "${OBS_GRAFANA_ADMIN_PASSWORD:?set OBS_GRAFANA_ADMIN_PASSWORD in the shell}"
  printf %s "$OBS_GRAFANA_ADMIN_PASSWORD" |
    kubectl -n "$NAMESPACE" create secret generic observability-secrets \
      --from-file=grafana-admin-password=/dev/stdin --dry-run=client -o yaml |
    kubectl apply -f -
}

case "${1:-}" in
  render)
    kubectl kustomize "$OVERLAY"
    ;;
  apply)
    # The platform requires an existing MySQL and Redis deployment (or Service)
    # and a separately provisioned application credential Secret.
    ensure_namespace
    if ! kubectl -n "$NAMESPACE" get secret awesome-zero-platform-secrets >/dev/null; then
      echo "application Secret awesome-zero-platform-secrets is missing; see Kubernetes deployment prerequisites" >&2
      exit 1
    fi
    secret
    kubectl apply -k "$OVERLAY"
    ;;
  status)
    kubectl -n "$NAMESPACE" get deployments,statefulsets,pods,services,jobs,pvc -o wide
    ;;
  smoke)
    bash "$ROOT/scripts/k8s-observability-smoke.sh"
    ;;
  forward)
    # Background kubectl processes are scoped to this invoking terminal.
    pids=()
    cleanup() { for pid in "${pids[@]}"; do kill "$pid" 2>/dev/null || :; done; wait 2>/dev/null || :; }
    trap cleanup EXIT INT TERM
    for item in "app-api 8888:8888" "elasticsearch 19200:9200" \
      "kibana 15601:5601" "prometheus 19090:9090" \
      "grafana 13000:3000" "jaeger 16686:16686"; do
      read -r service ports <<< "$item"
      kubectl -n "$NAMESPACE" port-forward --address 127.0.0.1 "service/$service" "$ports" >/dev/null &
      pids+=("$!")
    done
    echo "Local forwards: Kibana :15601 | Grafana :13000 | Jaeger :16686 | Prometheus :19090"
    wait
    ;;
  *)
    usage >&2
    exit 2
    ;;
esac
