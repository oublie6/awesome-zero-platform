#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PRODUCTION_COMPOSE="$ROOT_DIR/scripts/production-compose.sh"
RUNTIME_DIR="$ROOT_DIR/.runtime"
BOOTSTRAP_ENV="$RUNTIME_DIR/ci-compose-bootstrap.env"
FINAL_ENV="$RUNTIME_DIR/ci-compose-final.env"
TLS_CERT="$RUNTIME_DIR/ci-tls.crt"
TLS_KEY="$RUNTIME_DIR/ci-tls.key"
COMPOSE_PROJECT="awesome-zero-platform-ci-runtime"
TLS_HTTP_PORT=18081
TLS_HTTPS_PORT=18443

mkdir -p "$RUNTIME_DIR"
chmod 700 "$RUNTIME_DIR"

MYSQL_PASSWORD="$(openssl rand -hex 16)"
MYSQL_ROOT_PASSWORD="$(openssl rand -hex 16)"
REDIS_PASSWORD="$(openssl rand -hex 16)"
ACCESS_SECRET="$(openssl rand -hex 32)"
BOOTSTRAP_TOKEN="$(openssl rand -hex 32)"
ADMIN_USERNAME="ci-admin"
ADMIN_PASSWORD="Ci-$(openssl rand -hex 16)"

openssl req -x509 -newkey rsa:2048 -sha256 -nodes -days 1 \
  -subj '/CN=tls-edge' \
  -addext 'subjectAltName=DNS:tls-edge,DNS:localhost,IP:127.0.0.1' \
  -keyout "$TLS_KEY" \
  -out "$TLS_CERT" >/dev/null 2>&1
# The key is ephemeral, lives under a mode-0700 directory, and is deleted on exit.
# Bind-mounted files must still be readable by the unprivileged Nginx UID in the container.
chmod 644 "$TLS_CERT" "$TLS_KEY"

write_env() {
  local path="$1"
  local include_bootstrap="$2"
  {
    printf 'APP_MYSQL_PASSWORD=%s\n' "$MYSQL_PASSWORD"
    printf 'APP_MYSQL_ROOT_PASSWORD=%s\n' "$MYSQL_ROOT_PASSWORD"
    printf 'APP_REDIS_PASSWORD=%s\n' "$REDIS_PASSWORD"
    printf 'APP_AUTH_ACCESS_TOKEN_SECRET=%s\n' "$ACCESS_SECRET"
    printf 'APP_TLS_CERT_FILE=%s\n' "$TLS_CERT"
    printf 'APP_TLS_KEY_FILE=%s\n' "$TLS_KEY"
    printf 'APP_HTTP_PORT=%s\n' "$TLS_HTTP_PORT"
    printf 'APP_HTTPS_PORT=%s\n' "$TLS_HTTPS_PORT"
    if [[ "$include_bootstrap" == "yes" ]]; then
      printf 'APP_ADMIN_BOOTSTRAP_TOKEN=%s\n' "$BOOTSTRAP_TOKEN"
    fi
  } >"$path"
  chmod 600 "$path"
}

write_env "$BOOTSTRAP_ENV" yes
write_env "$FINAL_ENV" no

compose_bootstrap() {
  APP_COMPOSE_PROJECT_NAME="$COMPOSE_PROJECT" \
    APP_HTTPS_ENABLED=false \
    bash "$PRODUCTION_COMPOSE" --env-file "$BOOTSTRAP_ENV" "$@"
}

compose_final() {
  APP_COMPOSE_PROJECT_NAME="$COMPOSE_PROJECT" \
    APP_HTTPS_ENABLED=false \
    bash "$PRODUCTION_COMPOSE" --env-file "$FINAL_ENV" "$@"
}

compose_tls() {
  APP_COMPOSE_PROJECT_NAME="$COMPOSE_PROJECT" \
    APP_HTTPS_ENABLED=true \
    bash "$PRODUCTION_COMPOSE" --env-file "$FINAL_ENV" "$@"
}

cleanup() {
  local status=$?
  if [[ "$status" -ne 0 ]]; then
    echo "runtime acceptance failed; preserving final container diagnostics" >&2
    compose_tls ps >&2 || true
    compose_tls logs --no-color --tail=300 mysql schema redis app-api admin-web tls-edge >&2 || true
  fi
  compose_tls down --volumes --remove-orphans >/dev/null 2>&1 || true
  rm -f "$BOOTSTRAP_ENV" "$FINAL_ENV" "$TLS_CERT" "$TLS_KEY"
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
  shift || true
  for _ in $(seq 1 60); do
    if curl --fail --silent --show-error "$@" "$url" >/dev/null; then
      return 0
    fi
    sleep 2
  done
  echo "endpoint did not become healthy: $url" >&2
  return 1
}

assert_body() {
  local url="$1"
  local expected="$2"
  shift 2
  local actual
  actual="$(curl --fail --silent --show-error "$@" "$url")"
  if [[ "$actual" != "$expected" ]]; then
    printf 'unexpected response body from %s: got %q, expected %q\n' \
      "$url" "$actual" "$expected" >&2
    return 1
  fi
}

realtime_probe() {
  local service_url="$1"
  local token="$2"
  shift 2
  compose_bootstrap exec -T \
    -e APP_REALTIME_HEALTHCHECK_TOKEN="$token" \
    app-api /app/app-api \
    -realtime-healthcheck \
    -realtime-healthcheck-url "$service_url" \
    "$@"
}

compose_bootstrap down --volumes --remove-orphans
compose_bootstrap up -d --build --wait
compose_bootstrap ps

# Compose health and application readiness are related but not identical. Poll
# every externally asserted endpoint so a brief liveness/readiness lag cannot
# create a false runtime failure.
wait_http http://127.0.0.1:8888/health/live
wait_http http://127.0.0.1:8888/health/ready
wait_http http://127.0.0.1:8080/health
wait_http http://127.0.0.1:8080/
assert_body http://127.0.0.1:8080/health ok

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

realtime_probe ws://127.0.0.1:8888/ws "$ACCESS_TOKEN"
realtime_probe ws://admin-web:8080/ws "$ACCESS_TOKEN" -realtime-healthcheck-browser
METRICS="$(curl --fail --silent --show-error http://127.0.0.1:8888/metrics)"
grep -q '^awesome_zero_platform_realtime_connections_accepted_total ' <<<"$METRICS"

compose_tls up -d tls-edge --wait
wait_http "http://127.0.0.1:${TLS_HTTP_PORT}/healthz"
assert_body "http://127.0.0.1:${TLS_HTTP_PORT}/healthz" ok

read -r redirect_code redirect_url < <(
  curl --silent --show-error --output /dev/null \
    --write-out '%{http_code} %{redirect_url}\n' \
    "http://127.0.0.1:${TLS_HTTP_PORT}/admin/"
)
expected_redirect="https://127.0.0.1:${TLS_HTTPS_PORT}/admin/"
if [[ "$redirect_code" != "308" || "$redirect_url" != "$expected_redirect" ]]; then
  printf 'unexpected HTTPS redirect: status=%q location=%q expected=%q\n' \
    "$redirect_code" "$redirect_url" "$expected_redirect" >&2
  exit 1
fi

wait_http "https://127.0.0.1:${TLS_HTTPS_PORT}/health" --insecure
assert_body "https://127.0.0.1:${TLS_HTTPS_PORT}/health" ok --insecure
wait_http "https://127.0.0.1:${TLS_HTTPS_PORT}/" --insecure
realtime_probe wss://tls-edge:8443/ws "$ACCESS_TOKEN" -realtime-healthcheck-browser -realtime-healthcheck-insecure-tls

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
realtime_probe ws://admin-web:8080/ws "$FINAL_ACCESS_TOKEN" -realtime-healthcheck-browser
realtime_probe wss://tls-edge:8443/ws "$FINAL_ACCESS_TOKEN" -realtime-healthcheck-browser -realtime-healthcheck-insecure-tls
assert_body http://127.0.0.1:8080/health ok

compose_tls ps
compose_tls logs --no-color --tail=200 app-api admin-web tls-edge

echo "production container HTTP/WS and HTTPS/WSS hardening runtime acceptance passed"
