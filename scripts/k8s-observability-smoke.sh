#!/usr/bin/env bash
# Kubernetes-only live test. Port-forwards six cluster-internal Services to loopback.
set -euo pipefail
: "${OBS_GRAFANA_ADMIN_PASSWORD:?set OBS_GRAFANA_ADMIN_PASSWORD before smoke}"
NS=awesome-zero-platform
tmp="$(mktemp -d)"
pids=()
cleanup() {
  for pid in "${pids[@]}"; do kill "$pid" >/dev/null 2>&1 || :; done
  wait >/dev/null 2>&1 || :
  rm -rf "$tmp"
}
trap cleanup EXIT

kubectl -n "$NS" rollout status deploy/app-api --timeout=600s
kubectl -n "$NS" rollout status sts/elasticsearch --timeout=600s
kubectl -n "$NS" rollout status sts/prometheus --timeout=600s
kubectl -n "$NS" rollout status deploy/kibana --timeout=600s
kubectl -n "$NS" rollout status deploy/grafana --timeout=600s
kubectl -n "$NS" rollout status deploy/jaeger --timeout=600s
kubectl -n "$NS" wait --for=condition=complete job/elastic-bootstrap --timeout=600s

forward() {
  kubectl -n "$NS" port-forward --address 127.0.0.1 "service/$1" "$2" >"$tmp/$1.log" 2>&1 &
  pids+=("$!")
}
forward app-api 8888:8888
forward elasticsearch 19200:9200
forward kibana 15601:5601
forward prometheus 19090:9090
forward grafana 13000:3000
forward jaeger 16686:16686

wait_http() {
  local url="$1"
  for _ in $(seq 1 90); do
    if curl -fsS "$url" >/dev/null 2>&1; then return 0; fi
    sleep 2
  done
  echo "port-forward/service unavailable: $url" >&2
  return 1
}

wait_http http://127.0.0.1:8888/health/live
wait_http http://127.0.0.1:19200/_cluster/health
wait_http http://127.0.0.1:15601/api/status
wait_http http://127.0.0.1:19090/-/ready
wait_http http://127.0.0.1:13000/api/health
wait_http http://127.0.0.1:16686/

rid="k8s-observability-$(date +%s)"
curl -fsS -H "X-Request-Id: $rid" http://127.0.0.1:8888/health/live > /dev/null

# Repeated smoke runs must reuse the exact expected data view. Kibana 9 can
# return HTTP 400 for duplicate creation, so never treat arbitrary 400s as OK.
view_code="$(curl -sS -o "$tmp/kibana-existing-view.json" -w '%{http_code}' \
  http://127.0.0.1:15601/api/data_views/data_view/awesome-logs)"
if [[ "$view_code" == 404 ]]; then
  code="$(curl -sS -o "$tmp/kibana-data-view.json" -w '%{http_code}' \
    -X POST http://127.0.0.1:15601/api/data_views/data_view \
    -H "kbn-xsrf: awesome-zero-platform" -H "Content-Type: application/json" \
    -d '{"data_view":{"id":"awesome-logs","name":"Awesome Logs","title":"awesome-logs-*","timeFieldName":"@timestamp"}}')"
  if [[ "$code" != 200 && "$code" != 201 && "$code" != 409 ]]; then
    echo "Kibana data view creation error: HTTP $code" >&2
    cat "$tmp/kibana-data-view.json" >&2
    exit 1
  fi
elif [[ "$view_code" != 200 ]]; then
  echo "Kibana data view lookup error: HTTP $view_code" >&2
  cat "$tmp/kibana-existing-view.json" >&2
  exit 1
fi
curl -fsS http://127.0.0.1:15601/api/data_views/data_view/awesome-logs |
  python3 -c 'import json,sys; d=json.load(sys.stdin)["data_view"]; assert d["id"]=="awesome-logs" and d["title"]=="awesome-logs-*" and d["timeFieldName"]=="@timestamp", "unexpected Kibana data view contract"'

seen=false
for _ in $(seq 1 120); do
  if curl -fsS -X POST http://127.0.0.1:19200/awesome-logs-*/_search \
    -H 'Content-Type: application/json' \
    -d "{\"query\":{\"term\":{\"requestId.keyword\":\"$rid\"}},\"size\":1}" |
       python3 -c 'import json,sys; assert json.load(sys.stdin)["hits"]["total"]["value"]>0' 2>/dev/null; then
    seen=true; break
  fi
  sleep 2
done
if [[ "$seen" != true ]]; then
  echo "Filebeat -> Elasticsearch missing requestId $rid" >&2
  exit 1
fi

# Verify every Ready replica is individually discovered; a Service-level scrape
# could return a misleading single healthy target even with two API replicas.
expected="$(kubectl -n "$NS" get deployment app-api -o jsonpath='{.status.readyReplicas}')"
test -n "$expected"
curl -fsS http://127.0.0.1:19090/api/v1/targets |
  python3 -c 'import sys,json; d=json.load(sys.stdin); targets=[t for t in d["data"]["activeTargets"] if t.get("labels",{}).get("job")=="app-api" and t.get("health")=="up"]; want=int(sys.argv[1]); assert len(targets)>=want, f"healthy Pod targets {len(targets)} < {want}"' "$expected"
curl -fsS -u "admin:$OBS_GRAFANA_ADMIN_PASSWORD" \
  http://127.0.0.1:13000/api/datasources/uid/prometheus |
  python3 -c 'import sys,json; assert json.load(sys.stdin)["uid"]=="prometheus"'
curl -fsS -u "admin:$OBS_GRAFANA_ADMIN_PASSWORD" \
  http://127.0.0.1:13000/api/dashboards/uid/awesome-zero-platform |
  python3 -c 'import sys,json; assert json.load(sys.stdin)["dashboard"]["uid"]=="awesome-zero-platform"'

traces=false
for _ in $(seq 1 60); do
  if curl -fsS http://127.0.0.1:16686/api/services |
       python3 -c 'import json,sys; assert "app-api" in json.load(sys.stdin).get("data",[])' 2>/dev/null; then
    traces=true; break
  fi
  curl -fsS -H "X-Request-Id: $rid" http://127.0.0.1:8888/health/live >/dev/null 2>&1 || :
  sleep 2
done
if [[ "$traces" != true ]]; then
  echo "Jaeger missing app-api OTLP spans" >&2
  exit 1
fi

echo "PASS: Kubernetes Filebeat/ES/Kibana, Pod-aware Prometheus/Grafana, Jaeger traces"
