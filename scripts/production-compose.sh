#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BASE_COMPOSE_FILE="$ROOT_DIR/deploy/production/docker-compose.yml"
TLS_COMPOSE_FILE="$ROOT_DIR/deploy/production/docker-compose.tls.yml"

https_value="${APP_HTTPS_ENABLED:-false}"
https_value="${https_value,,}"

case "$https_value" in
  true|1|yes|on)
    https_enabled=true
    ;;
  false|0|no|off|'')
    https_enabled=false
    ;;
  *)
    printf 'invalid APP_HTTPS_ENABLED value %q; use true/false, 1/0, yes/no, or on/off\n' \
      "${APP_HTTPS_ENABLED}" >&2
    exit 2
    ;;
esac

# Keep the historical production Compose namespace stable so normal production
# commands continue to select production_mysql-data and production_redis-data.
# Docker Compose's explicit --project-name/-p option still has higher precedence.
export COMPOSE_PROJECT_NAME="${APP_COMPOSE_PROJECT_NAME:-${COMPOSE_PROJECT_NAME:-production}}"

compose_args=(-f "$BASE_COMPOSE_FILE")
if [[ "$https_enabled" == true ]]; then
  compose_args+=(-f "$TLS_COMPOSE_FILE")
fi

exec docker compose "${compose_args[@]}" "$@"
