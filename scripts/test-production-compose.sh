#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPT="$ROOT_DIR/scripts/production-compose.sh"
BASE_COMPOSE_FILE="$ROOT_DIR/deploy/production/docker-compose.yml"
TLS_COMPOSE_FILE="$ROOT_DIR/deploy/production/docker-compose.tls.yml"
TEMP_DIR="$(mktemp -d)"
CAPTURE_FILE="$TEMP_DIR/docker-args"
PROJECT_CAPTURE_FILE="$TEMP_DIR/compose-project"

cleanup() {
  rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

mkdir -p "$TEMP_DIR/bin"
cat >"$TEMP_DIR/bin/docker" <<'FAKE_DOCKER'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "${COMPOSE_PROJECT_NAME-}" >"$PRODUCTION_COMPOSE_PROJECT_CAPTURE"
printf '%s\n' "$@" >"$PRODUCTION_COMPOSE_CAPTURE"
FAKE_DOCKER
chmod +x "$TEMP_DIR/bin/docker"

assert_capture() {
  local https_value="$1"
  local expect_tls="$2"
  local app_project_name="$3"
  local compose_project_name="$4"
  local expected_project="$5"
  shift 5

  : >"$CAPTURE_FILE"
  : >"$PROJECT_CAPTURE_FILE"

  local -a env_args=(
    PATH="$TEMP_DIR/bin:$PATH"
    PRODUCTION_COMPOSE_CAPTURE="$CAPTURE_FILE"
    PRODUCTION_COMPOSE_PROJECT_CAPTURE="$PROJECT_CAPTURE_FILE"
  )

  if [[ "$https_value" == '__unset__' ]]; then
    env_args+=(-u APP_HTTPS_ENABLED)
  else
    env_args+=(APP_HTTPS_ENABLED="$https_value")
  fi

  if [[ "$app_project_name" == '__unset__' ]]; then
    env_args+=(-u APP_COMPOSE_PROJECT_NAME)
  else
    env_args+=(APP_COMPOSE_PROJECT_NAME="$app_project_name")
  fi

  if [[ "$compose_project_name" == '__unset__' ]]; then
    env_args+=(-u COMPOSE_PROJECT_NAME)
  else
    env_args+=(COMPOSE_PROJECT_NAME="$compose_project_name")
  fi

  env "${env_args[@]}" bash "$SCRIPT" "$@"

  local actual_project
  actual_project="$(cat "$PROJECT_CAPTURE_FILE")"
  if [[ "$actual_project" != "$expected_project" ]]; then
    printf 'unexpected COMPOSE_PROJECT_NAME: got %q, expected %q\n' \
      "$actual_project" "$expected_project" >&2
    exit 1
  fi

  mapfile -t actual <"$CAPTURE_FILE"
  expected=(compose -f "$BASE_COMPOSE_FILE")
  if [[ "$expect_tls" == true ]]; then
    expected+=(-f "$TLS_COMPOSE_FILE")
  fi
  expected+=("$@")

  if [[ "${#actual[@]}" -ne "${#expected[@]}" ]]; then
    printf 'unexpected argument count for APP_HTTPS_ENABLED=%s\n' "$https_value" >&2
    printf 'actual: %q\n' "${actual[@]}" >&2
    printf 'expected: %q\n' "${expected[@]}" >&2
    exit 1
  fi

  local index
  for index in "${!expected[@]}"; do
    if [[ "${actual[$index]}" != "${expected[$index]}" ]]; then
      printf 'argument %s mismatch for APP_HTTPS_ENABLED=%s: got %q, expected %q\n' \
        "$index" "$https_value" "${actual[$index]}" "${expected[$index]}" >&2
      exit 1
    fi
  done
}

# Stable default preserves the historical production_* named-volume namespace.
assert_capture __unset__ false __unset__ __unset__ production --env-file example.env config --services
assert_capture false false __unset__ __unset__ production config

# APP_COMPOSE_PROJECT_NAME is the supported environment override.
assert_capture true true isolated __unset__ isolated --env-file example.env up -d --wait

# Existing COMPOSE_PROJECT_NAME remains supported when the APP-specific value is absent.
assert_capture true true __unset__ compose-env compose-env config

# APP_COMPOSE_PROJECT_NAME intentionally takes precedence over COMPOSE_PROJECT_NAME.
assert_capture false false app-project compose-env app-project ps

# Docker Compose still receives an explicit CLI project name unchanged; the CLI has
# higher precedence than COMPOSE_PROJECT_NAME when Docker executes it.
assert_capture true true __unset__ __unset__ production --project-name cli-project config --services
assert_capture true true __unset__ __unset__ production -p short-cli-project config

for value in false FALSE 0 no NO off OFF ''; do
  assert_capture "$value" false __unset__ __unset__ production --env-file example.env config --services
done

for value in true TRUE 1 yes YES on ON; do
  assert_capture "$value" true __unset__ __unset__ production --env-file example.env up -d --wait
done

set +e
invalid_output="$({
  APP_HTTPS_ENABLED=maybe \
    PATH="$TEMP_DIR/bin:$PATH" \
    PRODUCTION_COMPOSE_CAPTURE="$CAPTURE_FILE" \
    PRODUCTION_COMPOSE_PROJECT_CAPTURE="$PROJECT_CAPTURE_FILE" \
    bash "$SCRIPT" config
} 2>&1)"
invalid_status=$?
set -e

if [[ "$invalid_status" -eq 0 ]]; then
  echo 'invalid APP_HTTPS_ENABLED unexpectedly succeeded' >&2
  exit 1
fi
if [[ "$invalid_output" != *'invalid APP_HTTPS_ENABLED value'* ]]; then
  printf 'invalid value error was not clear: %s\n' "$invalid_output" >&2
  exit 1
fi

echo 'production Compose HTTPS switch and project continuity tests passed'
