# Goal 0042: go-zero architecture final deployment and runtime acceptance

## Status

- State: in_progress
- Started: 2026-10-30
- Completed:
- Blockers:

## Goal

Complete the user-requested final code, deployment consistency, runtime, observability and reliability acceptance of Goals 0039–0041. Preserve existing correct business behavior and the modular monolith baseline.

## References

- `AGENTS.md`
- `docs/operations/logging-observability.md`
- `docs/operations/production-deployment.md`
- `deploy/kubernetes-overlays/observability-server/README.md`
- `docs/architecture/go-zero-http-and-module-integration.md`
- `docs/goals/archive/0039-go-zero-api-http-standardization.md`
- `docs/goals/archive/0040-redis-and-trace-governance.md`
- `docs/goals/archive/0041-module-contract-guardrails.md`

## Deliverables

1. Final code review of REST routing/middleware, JWT/session/Casbin compatibility, go-zero SQL/Redis usage and scoped native adapters, WebSocket context/tracing, metrics and module/data boundaries. Minimal fixes with regression tests only for demonstrated defects.
2. Kubernetes inventory, source/image provenance, configuration drift and resource/disk/PVC/restart evidence. Backup, verified rollback and deployment preflight before updating the API through the existing build/import process.
3. Real REST health/login/refresh/logout/Admin/authentication/authorization/error checks; WebSocket authentication/message/heartbeat/disconnect/reconnect checks; SQL/transaction/cache/WATCH/recovery checks.
4. Real Prometheus HTTP/error/latency/WS/Redis/runtime/CPU/memory collection, Grafana panels, Jaeger HTTP/SQL/transaction/Redis/WATCH/message spans with relationships/timing/errors, and Filebeat/Elasticsearch/Kibana request/trace correlation evidence.
5. Bounded reliability observations for goroutines/connections/memory, slow consumers/cleanup, backend recovery, host CPU/soft-lockup risks and retention/watermarks.
6. Formatting/tidy/generation/API contract/unit/race/integration/architecture/frontend/Kubernetes and GitHub Actions checks, distinct self-review and detailed final evidence report.

## Constraints

- User explicitly authorizes this Goal creation, code review, minimal fixes, tests, deployment and documentation changes; this expands the usual supplementary-executor role for this task.
- Scope is reusable technical infrastructure/platform integration and deployment verification; no speculative abstraction or unrelated business behavior changes.
- Main only; low-concurrency sequential resource-intensive work; no concurrent file writers.
- Use only observability-server profile, one replica and Recreate; preserve runtime settings and MySQL/Redis volumes. Never apply the default two-replica base or bootstrap schema over existing data.
- Stop deployment if backup/data safety or rollback cannot be confirmed. No disruptive fault injection in an unconfirmed environment. Never output credentials or sensitive payloads.
- Evidence must distinguish this executor, previous ChatGPT checks, CI, unavailable checks, failures/fixes and remaining uncertainty. No source-only claim of live trace success.

## Acceptance Criteria

- Every deliverable above has concrete observed evidence, including deployed immutable source/image identity before and after and actual component status.
- Verify latest source CI before deployment and final CI after push. Run available repository checks without weakening assertions; document exact unavailable checks or failures.
- Complete a distinct self-review covering contracts, state/order/concurrency, cleanup/recovery, privacy/security, resource use and boundaries; trace success and failure flows.
- Report changed files/reasons, code review, command/CI outcomes, deployed Git SHA/image, live signals, fixes, residual risks and required follow-up.
- Only after all criteria pass: mark completed, archive with scripts/archive-goal.sh, reset current template, commit/push main and confirm final CI. Otherwise retain incomplete evidence and document genuine blockers; never fabricate completion.

## Agent Strategy

Single primary executor, sequential verification and mandatory self-review.

## Working State

### Completed

- Read AGENTS/current goal and requested deployment/architecture references completely.
- Clean main fast-forwarded from 890abc8 to 698d760; Goals 0039–0041 archived completed.
- Live context kubernetes-admin@network-learning: both nodes Ready; API and observability/database Pods Ready; four PVCs Bound. API still uses goal0038-gozero-first and does not establish deployment of latest source.
- Local /data: 26 GiB available; 4.8 GiB RAM available, no swap. gh/node/npm absent; Docker/containerd/crictl/Go available. CI must be checked through supported GitHub tools; frontend tooling needs resolution.

### In progress

- Establish source CI evidence and preflight inventory before any deployment change.

### Remaining

- All implementation review, verification, backup/rollback, rollout and live acceptance gates.

### Verification status

- Initial read-only cluster inventory passed; no rollout, fault injection or live observability acceptance performed yet.
- make fmt-check passed. Both Kubernetes render validators passed after correcting invocation to supply rendered YAML (28 server-profile resources, 23 default-observability resources). Initial invocation without the required file argument failed with IndexError; no repository defect inferred.
- make test could not start with system Go 1.18.1 (go.mod requires 1.25.8); downloading isolated matching toolchain, then rerun. No test pass claimed.
- GitHub REST Actions inventory confirms synchronized source 698d7608cbecfb7c690a035b7675e5edc9ec0e3e: CI 41022280151 success, Fair Doudizhu 41022280106 success. New definition checkpoint 399703c triggered live CI 41043134748 and Fair Doudizhu 41043134806, not complete at observation. Short SHA query returned no runs; use full SHA/inventory.
- Goal definition committed and pushed directly to main: 399703c.

## Completion Report

Not complete. No production code or cluster resources changed.
