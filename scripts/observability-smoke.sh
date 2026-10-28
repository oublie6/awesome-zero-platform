#!/usr/bin/env bash
# Local developer smoke test for an explicitly enabled observability stack.
set -euo pipefail

: "${OBS_GRAFANA_ADMIN_PASSWORD:?export OBS_GRAFANA_ADMIN_PASSWORD before running smoke}"
BASE="http://127.0.0.1"
wait_http() {
  local url="$1"
  for _ in $(seq 1 150); do
    if curl --fail --silent "$url" >/dev/null; then return 0; fi
    sleep 2
  done
  printf 'service unavailable: %s\n' "$url" >&2
  return 1
}

wait_http "$BASE:8888/health/ready"
wait_http "$BASE:19200/_cluster/health"
wait_http "$BASE:15601/api/status"
wait_http "$BASE:16686/"
wait_http "$BASE:19090/-/ready"
wait_http "$BASE:13000/api/health"

# A harmless fixed path should be present in both logs and HTTP metrics.
id="observability-smoke-$(date +%s)"
curl --fail --silent --show-error -H "X-Request-Id: $id" "$BASE:8888/health/live" >/dev/null

# Data view creation is idempotent: 200/201 create, 409 already exists.
status="$(curl -sS -o /tmp/awesome-kibana-data-view.json -w '%{http_code}' \
  -X POST "$BASE:15601/api/data_views/data_view" \
  -H 'kbn-xsrf: awesome-zero-platform' -H 'Content-Type: application/json' \
  -d '{"data_view":{"id":"awesome-logs","name":"Awesome Logs","title":"awesome-logs-*","timeFieldName":"@timestamp"}}')"
if [[ "$status" != 200 && "$status" != 201 && "$status" != 409 ]]; then
  echo "Kibana data view API error ($status)" >&2
  cat /tmp/awesome-kibana-data-view.json >&2
  exit 1
fi
rm -f /tmp/awesome-kibana-data-view.json

found=false
for _ in $(seq 1 90); do
  response="$(curl -fsS -XPOST "$BASE:19200/awesome-logs-*/_search" \
    -H 'Content-Type: application/json' \
    -d "{\"query\":{\"term\":{\"requestId.keyword\":\"$id\"}},\"size\":1}" || true)"
  if printf '%s' "$response" | python3 -c 'import json,sys; d=json.load(sys.stdin); assert d["hits"]["total"]["value"]>0' 2>/dev/null; then
    found=true
    break
  fi
  sleep 2
done
if [[ "$found" != true ]]; then
  echo "Elasticsearch did not index requestId=$id" >&2
  exit 1
fi

curl -fsS "$BASE:19090/api/v1/targets" | python3 -c 'import sys,json; d=json.load(sys.stdin); assert any(t.get("labels",{}).get("job")=="app-api" and t.get("health")=="up" for t in d["data"]["activeTargets"])'
curl -fsS -u "admin:$OBS_GRAFANA_ADMIN_PASSWORD" "$BASE:13000/api/datasources/uid/prometheus" | python3 -c 'import json,sys; assert json.load(sys.stdin)["uid"]=="prometheus"'

trace_found=false
for _ in $(seq 1 45); do
  if curl -fsS "$BASE:16686/api/services" | python3 -c 'import json,sys; assert "app-api" in json.load(sys.stdin).get("data",[])' 2>/dev/null; then
    trace_found=true
    break
  fi
  sleep 2
done
if [[ "$trace_found" != true ]]; then
  echo "Jaeger did not receive app-api OTLP spans" >&2
  exit 1
fi

echo "observability smoke successful: Kibana data view, indexed requestId, Prometheus, Grafana and Jaeger"
