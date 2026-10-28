# Goal 0035: Optional JSON Log Search and Kibana Visibility

## Status

- State: in_progress
- Started: 2026-10-29
- Completed:
- Blockers:

## Goal

Provide opt-in centralized search and visualization for the modular monolith's existing go-zero structured logs using Filebeat -> Elasticsearch -> Kibana, without Kafka/go-stash or making logs a business availability dependency.

## References

- `AGENTS.md`
- `docs/architecture/module-boundaries.md`
- `docs/operations/production-deployment.md`
- `docs/operations/logging-observability.md` (created in this goal)

## Deliverables

1. Keep default console JSON logging and add an explicit, validated file logging opt-in to support a narrowly scoped shared log volume.
2. Improve platform HTTP access logs with stable event/module/duration fields and add unit verification without recording request bodies, tokens or credentials.
3. Provide optional Compose overlay for one developer-only Elastic/Filebeat/Kibana stack with non-root API filesystem policy preserved, dedicated log volume, localhost-bound dashboards and bounded retention.
4. Provide Kibana data-view bootstrap, diagnostic/search commands, automated compose validation, an end-to-end smoke script, and updated docs.

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
- Collector accepts JSON records from dedicated app volume, and configuration includes bounded retention; documented smoke proves indexed request logs can be found by requestId and Kibana data view is configured.
- Disable observability and verify service startup does not require it.
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
