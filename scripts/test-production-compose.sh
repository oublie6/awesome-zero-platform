#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPT="$ROOT_DIR/scripts/production-compose.sh"
BASE_COMPOSE_FILE="$ROOT_DIR/deploy/production/docker-compose.yml"
TLS_COMPOSE_FILE="$ROOT_DIR/deploy/production/docker-compose.tls.yml"
TEMP_DIR="$(mktemp -d)"
CAPTURE_FILE="$TEMP_DIR/docker-args"

cleanup() {
  rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

mkdir -p "$TEMP_DIR/bin"
cat >"$TEMP_DIR/bin/docker" <<'FAKE_DOCKER'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$@" >"$PRODUCTION_COMPOSE_CAPTURE"
FAKE_DOCKER
chmod +x "$TEMP_DIR/bin/docker"

assert_capture() {
  local value="$1"
  local expect_tls="$2"
  shift 2

  : >"$CAPTURE_FILE"
  if [[ "$value" == '__unset__' ]]; then
    env -u APP_HTTPS_ENABLED \
      PATH="$TEMP_DIR/bin:$PATH" \
      PRODUCTION_COMPOSE_CAPTURE="$CAPTURE_FILE" \
      bash "$SCRIPT" "$@"
  else
    APP_HTTPS_ENABLED="$value" \
      PATH="$TEMP_DIR/bin:$PATH" \
      PRODUCTION_COMPOSE_CAPTURE="$CAPTURE_FILE" \
      bash "$SCRIPT" "$@"
  fi

  mapfile -t actual <"$CAPTURE_FILE"
  expected=(compose -f "$BASE_COMPOSE_FILE")
  if [[ "$expect_tls" == true ]]; then
    expected+=(-f "$TLS_COMPOSE_FILE")
  fi
  expected+=("$@")

  if [[ "${#actual[@]}" -ne "${#expected[@]}" ]]; then
    printf 'unexpected argument count for APP_HTTPS_ENABLED=%s\n' "$value" >&2
    printf 'actual: %q\n' "${actual[@]}" >&2
    printf 'expected: %q\n' "${expected[@]}" >&2
    exit 1
  fi

  local index
  for index in "${!expected[@]}"; do
    if [[ "${actual[$index]}" != "${expected[$index]}" ]]; then
      printf 'argument %s mismatch for APP_HTTPS_ENABLED=%s: got %q, expected %q\n' \
        "$index" "$value" "${actual[$index]}" "${expected[$index]}" >&2
      exit 1
    fi
  done
}

for value in __unset__ false FALSE 0 no NO off OFF ''; do
  assert_capture "$value" false --env-file example.env config --services
done

for value in true TRUE 1 yes YES on ON; do
  assert_capture "$value" true --env-file example.env up -d --wait
done

set +e
invalid_output="$({
  APP_HTTPS_ENABLED=maybe \
    PATH="$TEMP_DIR/bin:$PATH" \
    PRODUCTION_COMPOSE_CAPTURE="$CAPTURE_FILE" \
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

echo 'production Compose HTTPS switch tests passed'
