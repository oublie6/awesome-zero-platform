# Goal 0034: General-Purpose Platform Module Boundaries

## Status

- State: completed
- Started: 2026-10-28
- Completed: 2026-10-28
- Blockers: None.

## Goal

Harden the existing modular monolith so additional non-game business modules can join without importing Doudizhu, directly accessing foreign tables, or moving domain logic into the platform. Reconcile public documentation with the general-purpose platform intent.

## References

- `AGENTS.md`
- `docs/architecture/overview.md`
- `docs/architecture/module-boundaries.md`
- `docs/architecture/external-integrations.md`
- `docs/operations/database-evolution.md`

## Deliverables

1. Move Doudizhu startup, HTTP/WSS registration, and shutdown behind a narrow compile-time business module adapter, keeping the shared application bootstrap generic.
2. Add deterministic tests for module startup, disabled modules, duplicate registration, failure cleanup, and reverse-order shutdown.
3. Add platform-wide Go import direction checks and business-owned SQL table checks to the ordinary `make test` path.
4. Document module/data ownership, external-system integration boundaries, monolith-to-microservice extraction conditions, safe database evolution, and the existing game-specific exceptions.
5. Update repository, server, architecture, deployment, and goal-workflow descriptions to match actual responsibilities.

## Constraints

- Preserve all existing HTTP/WSS routes and Doudizhu Server v1 behavior.
- No speculative universal plugin framework, dynamic code loading, new broker, RPC or microservice.
- Do not change MySQL major version or mutate existing databases without backup, migration and integration proof.
- Do not add external integration endpoints or credentials without a real provider and an explicit authentication/authorization contract.
- Follow `AGENTS.md`; keep edits scoped to this goal.

## Acceptance Criteria

- `make fmt-check`, `make test`, `make integration-test` and `make build` pass in an environment with the required Go version and Docker dependencies.
- Existing runtime, HTTP/WSS, Admin, security and game CI verifications remain green.
- Module startup tests prove disabled modules stay off, duplicate names fail, and failed startup unwinds already-started modules.
- A production module cannot introduce an import from platform/foundation into a concrete business, nor a known cross-business SQL table reference without failing the architecture tests.
- Docs clearly distinguish implemented capabilities from future integration/DB migration work.

## Agent Strategy

ChatGPT owns implementation, review and available verification. CI supplies Go 1.25 and container-backed checks. Codex is only used for a specific test gap if one exists.

## Working State

### Completed

- Architecture/source review, proposed changes and module/data boundaries captured.
- Implementation and documents prepared for main-branch CI validation.

### In progress

- None.

### Remaining

- None.

### Verification status

- Passed: main CI https://github.com/oublie6/awesome-zero-platform/actions/runs/40668262257 (unit, formatting, generated code, build, integration, race, secure envelope, Vue and production Compose runtime).
- Passed: Fair Doudizhu CI https://github.com/oublie6/awesome-zero-platform/actions/runs/40668262278 (application, manifest client, MySQL integration).
- Environment limitation: direct local compilation and Docker integration were unavailable (Go 1.23 and no Docker); GitHub Actions used Go 1.25 with containers.

## Completion Report

- Refactored the application composition root to register enabled business modules through a narrow compile-time adapter and reverse-order shutdown.
- Encapsulated Doudizhu wiring and HTTP/WSS handler registration without changing published routes or protocol semantics.
- Added Go architecture tests for import direction, registered schema-table ownership and business SQL access to known foreign tables, plus startup-failure/disabled-module tests.
- Updated server/repository/architecture/operations docs and recorded external integration and database evolution requirements.
- Verified only the 14 intended source/doc files changed since the previous `main` baseline; no databases, production secrets, or persistent volumes were altered.
- Self-review: examined initialization/error cleanup, disabled-module behavior, duplicate registration, cross-module data access, existing single-process gameplay authority, documentation claims and scope limits. No known defect found; indirect/dynamic SQL and new external providers remain subject to future review.
- Deferred intentionally: production MySQL major upgrade and versioned migration implementation, generic external gateway/credential provider, message publishing workers, distributed room ownership, and service extraction. These require their own evidence-driven goals.
- Verified implementation head: `6fda524d45e3c5e0c362497e65e3c98f17fe6d62`; all required CI suites succeeded.

