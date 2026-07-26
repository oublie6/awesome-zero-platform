# Goal: Platform Security and Production Foundation

## Status

- State: in_progress
- Started: 2026-07-26
- Completed:
- Blockers:

Supported states: `idle`, `ready`, `in_progress`, `completed`, `blocked`. New executable goals start as `ready`.

## Goal

Implement the remaining non-business and non-client platform capabilities required for secure authentication, replaceable authorization, observability, continuous integration, and production-oriented deployment.

## References

- `AGENTS.md`
- `docs/goals/requirements/0006-platform-security-production.md`

## Deliverables

1. Pluggable authentication application contracts and adapters for signed access tokens and Redis sessions.
2. Login, refresh, logout, and current-session HTTP endpoints without public registration.
3. Pluggable authorization contracts and a Casbin adapter.
4. Authentication and authorization HTTP middleware with narrow dependencies.
5. Prometheus-compatible HTTP metrics and operational configuration.
6. Production Docker, Compose, and Kubernetes baselines with environment-injected secrets.
7. GitHub Actions baseline CI.
8. Complete schema, documentation, and deterministic tests.

## Constraints

- Follow `AGENTS.md` and the referenced requirements.
- Do not add product-specific business modules or client applications.
- Do not create generic CRUD or generic repository infrastructure.
- Do not let platform application code depend directly on Casbin, JWT, Redis, or HTTP framework types.
- Keep work sequential and low-concurrency.
- Do not modify archived goals.

## Acceptance Criteria

- `make generate`, `make fmt-check`, and `make test` pass.
- Authentication and authorization unit tests pass.
- Integration coverage proves session rotation/revocation and authorization enforcement where dependencies are available.
- Public API and server documentation describe the new contracts and deferred capabilities.
- Production assets contain no committed real credentials.
- GitHub Actions performs deterministic low-concurrency verification.

## Agent Strategy

ChatGPT owns architecture, implementation, tests, documentation, and the initial pushed checkpoint. GitHub Actions performs baseline verification. Codex may later perform independent failure-driven verification.

## Working State

### Completed

- Goal requirements defined.
- Dedicated implementation branch created.

### In progress

- Authentication, authorization, observability, CI, and deployment implementation.

### Remaining

- Production code, tests, documentation, verification, and completion report.

### Verification status

- Not started.

## Completion Report

Not completed.
