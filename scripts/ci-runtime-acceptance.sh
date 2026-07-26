#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/deploy/production/docker-compose.yml"
RUNTIME_DIR="$ROOT_DIR/.runtime"
BOOTSTRAP_ENV="$RUNTIME_DIR/ci-compose-bootstrap.env"
FINAL_ENV="$RUNTIME_DIR/ci-compose-final.env"

mkdir -p "$RUNTIME_DIR"
chmod 700 "$RUNTIME_DIR"

MYSQL_PASSWORD="$(openssl rand -hex 16)"
MYSQL_ROOT_PASSWORD="$(openssl rand -hex 16)"
REDIS_PASSWORD="$(openssl rand -hex 16)"
ACCESS_SECRET="$(openssl rand -hex 32)"
BOOTSTRAP_TOKEN="$(openssl rand -hex 32)"
ADMIN_USERNAME="ci-admin"
ADMIN_PASSWORD="Ci-$(openssl rand -hex 16)"

write_env() {
  local path="$1"
  local include_bootstrap="$2"
  {
    printf 'APP_MYSQL_PASSWORD=%s\n' "$MYSQL_PASSWORD"
    printf 'APP_MYSQL_ROOT_PASSWORD=%s\n' "$MYSQL_ROOT_PASSWORD"
    printf 'APP_REDIS_PASSWORD=%s\n' "$REDIS_PASSWORD"
    printf 'APP_AUTH_ACCESS_TOKEN_SECRET=%s\n' "$ACCESS_SECRET"
    if [[ "$include_bootstrap" == "yes" ]]; then
      printf 'APP_ADMIN_BOOTSTRAP_TOKEN=%s\n' "$BOOTSTRAP_TOKEN"
    fi
  } >"$path"
  chmod 600 "$path"
}

write_env "$BOOTSTRAP_ENV" yes
write_env "$FINAL_ENV" no

compose_bootstrap() {
  docker compose --env-file "$BOOTSTRAP_ENV" -f "$COMPOSE_FILE" "$@"
}

compose_final() {
  docker compose --env-file "$FINAL_ENV" -f "$COMPOSE_FILE" "$@"
}

cleanup() {
  local status=$?
  if [[ "$status" -ne 0 ]]; then
    echo "runtime acceptance failed; preserving final container diagnostics" >&2
    compose_bootstrap ps >&2 || true
    compose_bootstrap logs --no-color --tail=300 mysql schema redis app-api admin-web >&2 || true
  fi
  compose_final down --volumes --remove-orphans >/dev/null 2>&1 || true
  rm -f "$BOOTSTRAP_ENV" "$FINAL_ENV"
}
trap cleanup EXIT

json_assert() {
  local expression="$1"
  python3 -c "import json,sys; data=json.load(sys.stdin); assert ($expression), data"
}

json_value() {
  local expression="$1"
  python3 -c "import json,sys; data=json.load(sys.stdin); value=$expression; print(value)"
}

wait_http() {
  local url="$1"
  for _ in $(seq 1 60); do
    if curl --fail --silent --show-error "$url" >/dev/null; then
      return 0
    fi
    sleep 2
  done
  echo "endpoint did not become healthy: $url" >&2
  return 1
}

compose_bootstrap down --volumes --remove-orphans
compose_bootstrap up -d --build --wait
compose_bootstrap ps

curl --fail --silent --show-error http://127.0.0.1:8888/health/live >/dev/null
curl --fail --silent --show-error http://127.0.0.1:8888/health/ready >/dev/null
curl --fail --silent --show-error http://127.0.0.1:8080/health | grep -q ok
curl --fail --silent --show-error http://127.0.0.1:8080/ >/dev/null

BOOTSTRAP_STATUS="$(curl --fail --silent --show-error http://127.0.0.1:8888/admin/bootstrap/status)"
printf '%s' "$BOOTSTRAP_STATUS" | json_assert "data['data']['available'] is True"

BOOTSTRAP_BODY="$(python3 -c 'import json,sys; print(json.dumps({"username":sys.argv[1],"displayName":"CI Administrator","password":sys.argv[2]}))' "$ADMIN_USERNAME" "$ADMIN_PASSWORD")"
curl --fail --silent --show-error \
  -H 'Content-Type: application/json' \
  -H "X-Admin-Bootstrap-Token: $BOOTSTRAP_TOKEN" \
  -d "$BOOTSTRAP_BODY" \
  http://127.0.0.1:8888/admin/bootstrap >/dev/null

SECOND_BOOTSTRAP_CODE="$(curl --silent --show-error -o "$RUNTIME_DIR/second-bootstrap.json" -w '%{http_code}' \
  -H 'Content-Type: application/json' \
  -H "X-Admin-Bootstrap-Token: $BOOTSTRAP_TOKEN" \
  -d "$BOOTSTRAP_BODY" \
  http://127.0.0.1:8888/admin/bootstrap)"
if [[ "$SECOND_BOOTSTRAP_CODE" -lt 400 ]]; then
  echo "replayed bootstrap unexpectedly succeeded" >&2
  exit 1
fi
rm -f "$RUNTIME_DIR/second-bootstrap.json"

LOGIN_BODY="$(python3 -c 'import json,sys; print(json.dumps({"identifier":sys.argv[1],"password":sys.argv[2]}))' "$ADMIN_USERNAME" "$ADMIN_PASSWORD")"
LOGIN_RESPONSE="$(curl --fail --silent --show-error \
  -H 'Content-Type: application/json' \
  -d "$LOGIN_BODY" \
  http://127.0.0.1:8888/auth/login)"
ACCESS_TOKEN="$(printf '%s' "$LOGIN_RESPONSE" | json_value "data['data']['accessToken']")"

curl --fail --silent --show-error \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  http://127.0.0.1:8888/auth/session |
  json_assert "data['code'] == 'OK' and bool(data['data']['sessionId'])"
curl --fail --silent --show-error \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  http://127.0.0.1:8888/admin/me |
  json_assert "'platform_super_admin' in data['data']['roles']"
curl --fail --silent --show-error \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  http://127.0.0.1:8888/admin/authorization/engine |
  json_assert "data['data']['syncHealthy'] is True and data['data']['localPolicyVersion'] == data['data']['databasePolicyVersion']"
curl --fail --silent --show-error \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  http://127.0.0.1:8888/admin/roles |
  json_assert "len(data['data']) >= 4"
curl --fail --silent --show-error \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  http://127.0.0.1:8888/admin/authorization/resources |
  json_assert "len(data['data']) >= 1"
curl --fail --silent --show-error \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  http://127.0.0.1:8888/admin/system/configuration |
  json_assert "data['code'] == 'OK'"

compose_final up -d --no-deps --force-recreate app-api
wait_http http://127.0.0.1:8888/health/ready

FINAL_BOOTSTRAP_STATUS="$(curl --fail --silent --show-error http://127.0.0.1:8888/admin/bootstrap/status)"
printf '%s' "$FINAL_BOOTSTRAP_STATUS" | json_assert "data['data']['available'] is False"

FINAL_LOGIN_RESPONSE="$(curl --fail --silent --show-error \
  -H 'Content-Type: application/json' \
  -d "$LOGIN_BODY" \
  http://127.0.0.1:8888/auth/login)"
FINAL_ACCESS_TOKEN="$(printf '%s' "$FINAL_LOGIN_RESPONSE" | json_value "data['data']['accessToken']")"
curl --fail --silent --show-error \
  -H "Authorization: Bearer $FINAL_ACCESS_TOKEN" \
  http://127.0.0.1:8888/admin/me |
  json_assert "'platform_super_admin' in data['data']['roles']"
curl --fail --silent --show-error http://127.0.0.1:8080/health | grep -q ok

compose_final ps
compose_final logs --no-color --tail=200 app-api admin-web

echo "production container runtime acceptance passed"
