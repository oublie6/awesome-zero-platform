#!/usr/bin/env bash
# Isolated GitHub runner acceptance. Never use this command against a populated stack.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export APP_COMPOSE_PROJECT_NAME=awesome-zero-platform-ci-observability
export APP_OBSERVABILITY_ENABLED=true
export APP_HTTPS_ENABLED=false
export APP_MYSQL_PASSWORD="$(openssl rand -hex 20)"
export APP_MYSQL_ROOT_PASSWORD="$(openssl rand -hex 20)"
export APP_REDIS_PASSWORD="$(openssl rand -hex 20)"
export APP_AUTH_ACCESS_TOKEN_SECRET="$(openssl rand -hex 32)"
export OBS_GRAFANA_ADMIN_PASSWORD="$(openssl rand -hex 24)"

compose() { bash "$ROOT/scripts/production-compose.sh" "$@"; }
cleanup() {
  status=$?
  if [[ $status -ne 0 ]]; then
    echo "observability acceptance failure: status=$status" >&2
    compose ps >&2 || true
    compose logs --no-color --tail=100 app-api elasticsearch elastic-bootstrap filebeat kibana jaeger prometheus grafana >&2 || true
  fi
  compose down --volumes --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT

# Required for Elasticsearch mmap bootstrap checks in Linux runners.
if command -v sudo >/dev/null 2>&1; then
  sudo sysctl -w vm.max_map_count=262144
fi
compose config >/dev/null
compose up -d --build --wait
bash "$ROOT/scripts/observability-smoke.sh"
echo "Goal 0035 full three-signal runtime acceptance passed"
