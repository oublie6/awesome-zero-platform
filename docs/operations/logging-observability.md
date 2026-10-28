# Logs, Metrics, and Traces (Goal 0035)

This repository exposes three **opt-in developer observability pipelines** without replacing go-zero's standard logger or introducing Kafka/go-stash:

| Signal | Path | UI |
| --- | --- | --- |
| JSON logs | go-zero `logx` file output -> shared app-only volume -> Filebeat -> Elasticsearch | Kibana at `http://127.0.0.1:15601` |
| Metrics | existing `/metrics` (HTTP, WebSocket, process/Go) -> Prometheus | Grafana at `http://127.0.0.1:13000` |
| Traces | go-zero OpenTelemetry -> OTLP/gRPC -> Jaeger v2 | Jaeger at `http://127.0.0.1:16686` |

Prometheus can also be inspected directly at `http://127.0.0.1:19090` and a **development-only, unauthenticated** Elasticsearch API at `http://127.0.0.1:19200`. All published ports are loopback only. There is no externally exposed OTLP receiver; `app-api` sends traces internally to `jaeger:4317`.

## Start locally on a dedicated development machine

Allocate **at least ~8 GB of available memory** and sufficient disk space. This combined stack is unsuitable for the existing 1–2 GB primary development environment. It includes Elasticsearch 9.1, Kibana 9.1, Filebeat 9.1, Prometheus, Grafana and Jaeger v2.20. Java heap is limited to 512 MB, but the total Elasticsearch and Kibana processes consume more than their heap sizes.

Set the same application database/Redis/auth secrets required by [production deployment](production-deployment.md), and a separate Grafana password:

```bash
export APP_MYSQL_PASSWORD="$(openssl rand -hex 24)"
export APP_MYSQL_ROOT_PASSWORD="$(openssl rand -hex 24)"
export APP_REDIS_PASSWORD="$(openssl rand -hex 24)"
export APP_AUTH_ACCESS_TOKEN_SECRET="$(openssl rand -hex 32)"
export OBS_GRAFANA_ADMIN_PASSWORD="$(openssl rand -hex 24)"
export APP_OBSERVABILITY_ENABLED=true
export APP_HTTPS_ENABLED=false
make production-config
make production-up
bash scripts/observability-smoke.sh
```

**Existing persistent deployments:** the example generates **new passwords** and is **only safe on a fresh isolated stack**. For an existing populated Compose deployment, reuse its established application/database secrets and project name, then add the new observability flag and Grafana secret. Do not switch Compose project name or reset volumes. You can use `--env-file` as documented in the production deployment guide.

Log in to Grafana with username `admin` and your `OBS_GRAFANA_ADMIN_PASSWORD`. The preprovisioned `Platform / Awesome Zero Platform Overview` dashboard visualizes HTTP request rate, HTTP 5xx, p95 latency, active WebSockets, process resident memory and realtime message rate. Grafana reads Prometheus directly; no duplicate go-zero Prometheus server is started.

Kibana data view `awesome-logs-*` is created by the smoke script. Use Discover and filter `requestId`, `module`, `eventAction`, `statusCode` and time. For a single request:

```bash
curl -H 'X-Request-Id: diagnostic-123' http://127.0.0.1:8888/health/live
curl -fsS -XPOST http://127.0.0.1:19200/awesome-logs-*/_search \
  -H 'Content-Type: application/json' \
  -d '{"query":{"term":{"requestId.keyword":"diagnostic-123"}}}'
```

For tracing, open Jaeger, select `app-api`, and search for recent spans. go-zero v1.10+ uses `Batcher: otlpgrpc`, **not** the removed `jaeger` batcher. Go HTTP, zrpc and instrumented SQL/Redis calls may produce spans when their respective tracing paths are enabled. Jaeger all-in-one uses **in-memory storage**, so spans do **not** survive Jaeger restarts. Trace IDs can be correlated with log context where the go-zero tracer records them. Do not assume business domain spans automatically exist without explicit domain instrumentation.

## Disable / separate the dependency

`APP_OBSERVABILITY_ENABLED=false` (the default) uses the original Compose deployment without any observability containers, log volume, or OTLP configuration. `APP_LOG_MODE` and `APP_TELEMETRY_ENDPOINT` are absent by default. The API can run whether or not Elastic, Prometheus, Grafana or Jaeger is reachable.

```bash
APP_OBSERVABILITY_ENABLED=false make production-config
# To stop the full optional Compose stack, include the flag so it selects its services:
APP_OBSERVABILITY_ENABLED=true make production-down
```

`make production-down` does **not** delete named volumes. If you disable observability in an existing running environment, execute a full `down` with the flag enabled first to avoid orphaned observability services; then start the default stack.

## Retention, redaction and limits

- Application log volume `app-logs` is writable to the non-root app UID after an isolated one-shot volume ownership setup; Filebeat receives **read-only** access. It does not mount the Docker daemon socket or host-wide log directories.
- Files are rotated/retained according to go-zero logx's file policy; ES has an explicit **7-day index lifecycle delete policy** for `awesome-logs-*`. Filebeat's registry lives on its own named volume to resume tailing after restart. Verify real ingest and log-file rotation in a live environment.
- Prometheus stores at most seven days / 1 GB in its TSDB. Grafana dashboards are provisioned declaratively. Jaeger traces are ephemeral.
- Access logs include `requestId`, `module`, `eventAction=http.access`, `durationMs`, method, path and status, **not** full query strings, request bodies or Authorization headers. Do not log passwords, JWTs, refresh tokens, encryption/reveal private keys, sensitive random materials, personal data or private card hands. Restrict access and retention even for development logs.
- Metrics route labels are bounded to well-known routes or grouped categories to avoid attacker-controlled high-cardinality metric labels.
- There is **no authentication/TLS for the development Elasticsearch and Kibana endpoints**. Loopback binding is only a local safeguard; anyone with access to the host or a compromised browser may be able to access the data. **Never expose or reuse this Compose overlay as a public/production observability deployment.** Production should use authentication, TLS, restricted network ingress, index permissions, audited retention and capacity planning.
- `OBS_GRAFANA_ADMIN_PASSWORD` is a required environment variable for enabling this overlay; it must not be committed.

## Verification evidence and limitations

- `bash scripts/test-observability-compose.sh` checks flag and file-overlay selection.
- `make production-config` with the flag disabled and enabled validates Compose configuration.
- `bash scripts/observability-smoke.sh` tests the **live** developer stack: API readiness, ES indexing by `requestId`, Kibana data-view registration, Prometheus target health, Grafana datasource and Jaeger service trace arrival. A script being present is **not** evidence it has passed; use its actual exit code.
- GitHub CI checks code, deployment configuration and runtime acceptance separately. Only mark an observability-stack runtime integration verified if a CI job or local smoke actually ran all the containers and reported success.

## Future extensions

Add a durable OTLP collector for multi-backend delivery, production-grade secured Elastic, business-span instrumentation, dashboards and alert rules based on actual operational requirements. Kafka/go-stash remains intentionally deferred; avoid introducing a broker solely for a single application logger.
