# Goal 0039: go-zero API and HTTP platform standardization

## Status

- State: in_progress
- Started: 2026-10-30
- Completed:
- Blockers:

## Goal

Use go-zero First for API contracts and HTTP infrastructure while preserving the deployed Goal 0038 API, authentication, session, error payload, metrics, WebSocket upgrade and business semantics. Follow up with separately verified Goal 0040 observability/Redis governance and Goal 0041 module extension boundary improvements.

## References

- `AGENTS.md`
- `docs/architecture/go-zero-first-components.md`
- `docs/operations/goal0038-kubernetes-acceptance.md`

## Deliverables

1. Establish machine-checked API route inventory and typed contract boundaries, keeping goctl generated health files stable; avoid overwriting hand-written business handlers.
2. Use go-zero REST middleware interfaces consistently, preserving custom security-sensitive semantics where native handlers are incompatible.
3. Fix verified HTTP error handling defects and add regression coverage for affected authentication, error and route paths.
4. Document native/Go-zero compatibility decisions and maintain CI coverage.
5. Complete self-review, CI and archival before advancing other goals.

## Constraints

- Work on main and verify CI.
- No server deployment or in-place schema/Redis key migrations by ChatGPT.
- Do not turn on go-zero request-body logging for secrets or change existing session JWT semantics.
- Preserve status codes, JSON envelopes, 101 WebSocket hijack, CORS, tracing, metrics and per-route body limit.
- Do not claim all business routes are safely generated unless goctl equivalence has been proven.

## Acceptance Criteria

- Route contract drift checked in CI.
- No swallowed critical admin read errors.
- Same authentication/error/status behavior as Goal 0038.
- Tests pass and native exceptions explained.

## Agent Strategy

ChatGPT owns implementation/self-review/CI. Codex only for supplementary server-specific verification after an authorized handoff.

## Working State

### Completed

- Inspected existing go-zero v1.10.2 REST middleware and initial route registration.

### In progress

- Implement route contracts and safe error handling.

### Remaining

- Tests, CI, self-review, archival.

### Verification status

- Pending.

## Completion Report

Pending.
