# Goal 0035: Unified Logs, Metrics and Tracing

## Status

- State: in_progress
- Started: 2026-10-29
- Completed:
- Blockers:

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

- Requirements and architecture identified; verified existing JSON logx and Prometheus integration.

### In progress

- Code, optional deployment overlay, tests, docs, CI and runtime smoke artifacts.

### Remaining

- CI review, runtime validation where feasible, and final completion report.

### Verification status

- Not started.

## Completion Report

Pending.
