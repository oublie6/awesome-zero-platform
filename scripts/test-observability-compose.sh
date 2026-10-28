#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPT="$ROOT_DIR/scripts/production-compose.sh"
BASE="$ROOT_DIR/deploy/production/docker-compose.yml"
TLS="$ROOT_DIR/deploy/production/docker-compose.tls.yml"
OBS="$ROOT_DIR/deploy/observability/docker-compose.yml"
TEMP="$(mktemp -d)"
trap 'rm -rf "$TEMP"' EXIT
mkdir -p "$TEMP/bin"
cat >"$TEMP/bin/docker" <<'FAKE'
#!/usr/bin/env bash
printf '%s\n' "$@" >"$TEST_ARGS_FILE"
FAKE
chmod +x "$TEMP/bin/docker"

assert_compose() {
  local https="$1" obs="$2"
  shift 2
  TEST_ARGS_FILE="$TEMP/args" PATH="$TEMP/bin:$PATH" \
    APP_HTTPS_ENABLED="$https" APP_OBSERVABILITY_ENABLED="$obs" \
    OBS_GRAFANA_ADMIN_PASSWORD=ci-only-example \
    bash "$SCRIPT" config --services
  local -a actual expected
  mapfile -t actual <"$TEMP/args"
  expected=(compose -f "$BASE")
  if [[ "$https" == true ]]; then expected+=(-f "$TLS"); fi
  if [[ "$obs" == true ]]; then expected+=(-f "$OBS"); fi
  expected+=(config --services)
  if [[ "${actual[*]}" != "${expected[*]}" ]]; then
    printf 'unexpected compose arguments: %s\n' "${actual[*]}" >&2
    exit 1
  fi
}

assert_compose false false
assert_compose true false
assert_compose false true
assert_compose true true

if APP_OBSERVABILITY_ENABLED=invalid PATH="$TEMP/bin:$PATH" \
  TEST_ARGS_FILE="$TEMP/args" bash "$SCRIPT" config >/dev/null 2>&1; then
  echo "invalid observability flag should fail" >&2
  exit 1
fi
echo "optional observability Compose overlay switches passed"
