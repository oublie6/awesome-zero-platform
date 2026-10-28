# Goal 0035: Unified Logs, Metrics and Tracing

## Status

- State: completed
- Started: 2026-10-29
- Completed: 2026-10-29
- Blockers: None.

## Goal

Deliver an opt-in, single-host developer observability stack: JSON logs via Filebeat -> Elasticsearch -> Kibana, existing Prometheus HTTP/WS metrics via Prometheus -> Grafana, and go-zero OpenTelemetry traces via OTLP -> Jaeger. None of these services may become a core business availability dependency.

## References

- `AGENTS.md`
- `docs/architecture/module-boundaries.md`
- `docs/operations/production-deployment.md`
- `docs/operations/logging-observability.md` (created in this goal)

## Deliverables

1. Keep default console JSON logging and add an explicit, validated file logging opt-in to support a narrowly scoped shared log volume.
2. Improve platform HTTP access logs with stable event/module/duration fields and bounded metric route labels; add unit verification without recording request bodies, tokens or credentials.
3. Provide optional Compose overlay for Elastic/Filebeat/Kibana, Prometheus/Grafana and Jaeger with non-root API filesystem policy preserved, dedicated log volume, localhost-bound dashboards, safe development defaults and bounded retention where appropriate.
4. Configure go-zero OTLP tracing and a Grafana datasource/dashboard without replacing existing application metrics; provide Kibana data-view bootstrap, diagnostic/search commands, automated compose validation, smoke scripts, and updated docs.

## Constraints

- Keep business endpoints, protocols and existing CI/runtime tests unchanged.
- No Docker socket/host-wide container log mount, Kafka, go-stash, distributed tracing backend, new product module or unvalidated MySQL change.
- Elasticsearch security-disabled mode must be documented and explicitly rejected for public/production exposure; no published non-loopback endpoints.
- The collector must be optional; the platform starts without Elasticsearch, Kibana or Filebeat.
- No sensitive request bodies, bearer tokens, secrets, encryption keys or private game hands in logs.
- Follow `AGENTS.md`.

## Acceptance Criteria

- `make test`, `make fmt-check`, `make build`, and existing CI/runtime tests succeed.
- Both normal Compose and observability opt-in Compose render successfully and preserve existing project namespace and TLS configuration.
- Collector accepts JSON records from a dedicated app volume; Prometheus scrapes the existing custom metrics endpoint; Jaeger receives OTLP spans; retention settings and Grafana/Kibana provisioning are documented and tested where available.
- Disable observability and verify service startup does not require it. CI must verify both compose modes and must not claim live Elastic/Jaeger visibility unless observed.
- Documentation explicitly distinguishes CI-verified configuration from a full live Elastic stack acceptance when the latter cannot run.

## Agent Strategy

ChatGPT implements and verifies through GitHub CI; additional heavy Elastic-stack runtime verification is only claimed if actually executed.

## Working State

### Completed

- Existing go-zero JSON logger and custom Prometheus metrics reused; opt-in file logging, contextual access log fields and bounded metric route labels implemented.
- Optional development-only Compose overlay built with Filebeat/Elasticsearch/Kibana, Prometheus/Grafana and Jaeger OTLP/gRPC; non-root writable application log volume, localhost-bound UIs and ILM/TSDB retention included.
- Grafana provisioning and sample dashboard; Kibana data-view registration; live index/metrics/trace acceptance scripts; unit/integration tests and repository documentation.
- Fixed Docker Compose multi-file bind path resolution and Filebeat 9 long-form strict permissions argument based on real CI errors.

### In progress

- None.

### Remaining

- None.

### Verification status

- PASS: Main CI, Go format/generation, all unit tests, race checks, MySQL integration, frontend and crypto checks, binary build, production Compose acceptance: https://github.com/oublie6/awesome-zero-platform/actions/runs/40732503270
- PASS: Fair Doudizhu application, MySQL integration, manifest tests: https://github.com/oublie6/awesome-zero-platform/actions/runs/40732503123
- PASS: Dedicated observability runtime: https://github.com/oublie6/awesome-zero-platform/actions/runs/40732502950; observed live Filebeat/Elasticsearch indexing by requestId, Kibana data view, Prometheus target, Grafana datasource, Jaeger OTLP service traces.
- Docker and Go 1.25 runtime checks were performed by GitHub Actions; the local container lacks Docker and runs Go 1.23.

## Completion Report

The required Goal 0035 implementation and all three workflow acceptances completed successfully at commit ccd8dcf7a39dc5c5555bf5085621ba1b1e7d9366.

Scope remains intentionally limited: this is a **developer-only opt-in** observability overlay, with unauthenticated loopback-bound ES/Kibana, transient Jaeger in-memory storage, one API process and existing metrics. It is not a secured public production deployment; no Kafka/go-stash, tracing collector cluster, external gateway or microservice has been introduced. Existing HTTP/WSS game routes and persistent databases were not altered.

Self-review verified logging avoids bodies/query-string/tokens, no Docker socket is mounted, source log volume is readable by Filebeat while writable to non-root API, metrics series labels are bounded, optional enablement preserves default production Compose, and the smoke validates actual traces and indexed request logs. Follow-up production needs TLS/auth, capacity planning, stable retention/storage, failure alerts and service-specific spans. The heavy observability workflow has been scoped to relevant source/deployment edits to avoid unnecessary redeploys on Goal/documentation-only changes.

