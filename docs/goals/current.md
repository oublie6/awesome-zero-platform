# Goal 0002: Establish the reusable HTTP foundation

## Status

- State: ready
- Started:
- Completed:
- Blockers:

## Goal

Establish the reusable HTTP request and response foundation for Awesome Zero Platform by adding consistent application errors, response envelopes, request IDs, panic recovery, access logging, configurable CORS, baseline security headers, and request body limits to the existing go-zero REST application.

## References

- `AGENTS.md`
- `docs/architecture/overview.md`
- `docs/requirements/server-bootstrap.md`
- `docs/requirements/http-foundation.md`
- `server/README.md`
- `server/apps/app-api/app.api`
- `server/apps/app-api/internal/bootstrap`

## Deliverables

1. Create focused reusable packages under `server/foundation` for application errors, response writing, request ID handling, and HTTP middleware, refining package boundaries only when Go or go-zero integration requires it.
2. Define a stable application error abstraction with machine-readable code, safe client message, HTTP status, and wrapped internal cause support.
3. Define and use a standard JSON response envelope for ordinary API success and error responses.
4. Implement request ID intake, validation, generation, request-context propagation, response-header propagation, response-envelope propagation, and access-log correlation.
5. Integrate panic recovery that returns the standard internal-error envelope while logging the panic and stack trace without exposing them to clients.
6. Add structured access logging containing request ID, method, path, status code, elapsed duration, and the documented client-address field.
7. Add configurable CORS with startup validation for unsafe or contradictory combinations.
8. Add baseline security response headers suitable for JSON APIs.
9. Add a configurable request body limit whose overflow maps to HTTP 413 and the standard error envelope.
10. Integrate these capabilities into `server/apps/app-api` using go-zero-supported extension points without replacing the framework or server lifecycle.
11. Decide and document the health endpoint response-envelope exception while ensuring health endpoints still receive globally safe middleware and the request ID response header.
12. Add focused tests for the error model, response envelopes, request IDs, recovery, access logging, CORS, security headers, request-size enforcement, and related configuration validation.
13. Update `server/README.md` with the implemented HTTP conventions, configuration, request ID header, response examples, and health endpoint policy.

## Constraints

- Follow `AGENTS.md` and every referenced document.
- Keep the existing modular-monolith `app-api` process and go-zero REST lifecycle.
- Do not replace go-zero routing, server startup, configuration loading, recovery lifecycle, or handler generation with a parallel custom framework.
- Do not create generic dumping grounds such as `common`, `utils`, `helpers`, or an oversized catch-all middleware package.
- Do not create speculative business error codes, generic CRUD abstractions, repository abstractions, dependency-injection frameworks, or plugin systems.
- Do not add PostgreSQL, MySQL, Redis, Kafka, object storage, users, authentication, authorization, sessions, JWT, Casbin, metrics, distributed tracing backends, CI, Docker, Kubernetes, clients, or business modules.
- Do not log request bodies, response bodies, authorization headers, cookies, secrets, or stack traces to clients.
- Preserve semantically correct HTTP status codes; application codes must not flatten all responses to HTTP 200.
- Prefer standard library and existing go-zero facilities. Add a dependency only when its benefit is explicit, narrow, tested, and documented.
- Keep generated goctl files distinguishable from handwritten code. Prefer `.api` changes and regeneration when the API contract changes.
- Preserve the existing minimal health payload unless a referenced requirement clearly justifies changing it; document the chosen policy.
- Do not modify archived goals.
- Do not expand or reinterpret this goal without explicit user instruction.
- Codex may update only the Status, Working State, and Completion Report sections of this file unless a deliverable explicitly requires another documentation update.

## Acceptance Criteria

1. `make generate` succeeds and is repeatable without unintended diffs.
2. `make fmt` succeeds.
3. `make test` succeeds; equivalently, `cd server && go test ./...` passes.
4. `make run` starts the API with the committed safe development configuration.
5. An ordinary successful API response uses the documented response envelope and includes the effective request ID in both the response body and documented response header.
6. A valid caller-provided request ID is preserved; a missing or invalid value is replaced with a generated valid ID.
7. Application errors map to their configured HTTP status, stable application code, safe message, `data: null`, and effective request ID.
8. An unknown internal error maps to HTTP 500 with the generic internal-error envelope while retaining the internal cause for server-side logging.
9. A deliberate handler panic is recovered, returns HTTP 500 with the safe standard envelope and request ID, logs diagnostic details server-side, and does not terminate the server process.
10. An oversized request body returns HTTP 413 with the stable request-too-large application code and request ID.
11. Security headers include at least `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, and `Referrer-Policy: no-referrer`.
12. Configured allowed CORS origins receive the expected headers, denied origins do not, preflight behavior is tested, and wildcard-plus-credentials configuration is rejected or safely prevented.
13. Access logging exposes a testable structured record containing request ID, method, path, status code, elapsed duration, and the documented client-address field without logging sensitive content.
14. `/health/live` and `/health/ready` continue to return HTTP 200 with their documented minimal payloads, include the request ID response header, and receive globally safe middleware.
15. Invalid middleware configuration, including an invalid request-size limit or contradictory CORS configuration, produces a clear startup validation failure.
16. `server/README.md` accurately documents the response format, error behavior, request ID header and validation policy, body-size limit, CORS configuration, security headers, access logging, and health endpoint exception.
17. No secrets, generated binaries, editor files, test artifacts, or unrelated infrastructure are committed.
18. Final Git inspection shows only Goal 0002 implementation, tests, configuration, and necessary documentation changes.

## Agent Strategy

The primary agent owns package boundaries, go-zero integration, integration testing, and final verification. It may use subagents for independent go-zero extension-point analysis, API/error design review, middleware security review, or test review. Do not allow multiple agents to modify the same files concurrently. The primary agent must inspect and integrate every subagent result.

## Execution Process

1. Synchronize the branch according to `AGENTS.md`, then read all references completely and inspect the current code and generated-file boundaries.
2. Produce a concise plan identifying reusable packages, go-zero integration points, middleware order, configuration additions, and tests before editing.
3. Define the response and application-error contracts first, then request ID context behavior.
4. Integrate middleware in a deliberate order and document why that order preserves request ID, recovery, access logging, CORS, security headers, and body-limit behavior.
5. Add tests incrementally, including a small test-only or clearly non-production route mechanism when needed to verify success, application errors, unknown errors, panic recovery, and body limits without creating speculative public APIs.
6. Run generation, formatting, unit tests, startup, live HTTP checks, CORS checks, request-ID checks, oversized-body checks, and panic-survival checks.
7. Inspect logs and client responses to confirm internal details are retained server-side but not leaked.
8. Inspect the final Git diff for scope expansion, unsafe defaults, sensitive logging, or accidental edits to generated files.
9. Update the permitted sections below with concrete implementation and verification evidence.
10. Commit and push according to `AGENTS.md`, stopping only when all acceptance criteria pass or a genuine blocker is documented with evidence.

## Working State

### Completed

- Goal 0001 archived.
- HTTP foundation requirements prepared.
- Goal 0002 execution contract prepared.

### In progress

- None.

### Remaining

- All implementation deliverables.

### Verification status

- Not started.

## Completion Report

Not started.
