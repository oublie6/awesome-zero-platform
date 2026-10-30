# Goal 0039: go-zero API and HTTP standardization

## Status

- State: completed
- Started: 2026-10-30
- Completed: 2026-10-30
- Blockers: None for agreed API route inventory and HTTP middleware migration. Full goctl generation of business handler implementations/DTOs remains deliberately out of scope to protect stable client contracts.

## Goal

Migrate shared HTTP route authentication/authorization registration to go-zero middleware and keep established error format, request logging/privacy and route contracts compatible.

## Deliverables

1. All **47 static REST routes** inventoried in `server/apps/app-api/contracts/routes.api`; `make api-contract-check` validates goctl DSL syntax, and architecture tests verify exact parity with registrations.
2. Admin (33 protected routes), authentication and Doudizhu REST endpoints use go-zero `rest.WithMiddleware` / `rest.ToMiddleware` with existing revocable Redis Session checks and Casbin authorization.
3. Corrected swallowed Admin account-role lookup and system overview bootstrap-read errors; preserve response envelopes, statuses and existing external API paths.
4. Retained carefully scoped outer security middlewares for request IDs, redacted logs, panic JSON error handling, streaming body size, CORS and WebSocket Hijack: go-zero v1.10.2 defaults are not semantically equivalent for those contracts.
5. Added architecture and compatibility reference `docs/architecture/go-zero-http-and-module-integration.md`.

## Verification

- Code checkpoint: `f29a53c8331420a63d9ec2f712fdb58bb4068d21`.
- Full main CI (unit, race, MySQL/Redis integration, code generation, formatting, production Compose runtime) **success**: https://github.com/oublie6/awesome-zero-platform/actions/runs/41021397036
- Fair Doudizhu regression **success**: https://github.com/oublie6/awesome-zero-platform/actions/runs/41021396898
- Changes were implemented and self-reviewed by ChatGPT. GitHub Actions independently executed repository checks. No Kubernetes server work was performed by ChatGPT.

## Mandatory Self-Review

Reviewed public routes and aliases, protected/unprotected grouping, Redis-backed JWT session invalidation, middleware ordering, 404/405 behavior, CORS, response envelope compatibility, sensitive request logging, WebSocket upgrade and existing tests. Corrected errors in API regex and restored original observability tests after discovering they were overwritten during implementation. Source and integration CI passed on the final checkpoint.

## Remaining / Important Scope Boundary

`server/apps/app-api/app.api` remains the **goctl generation source for health endpoints only**. The new `contracts/routes.api` is a validated route **inventory** without full request/response DTO definitions. Do not claim that all 45 business routes have generated handlers or types. Complete generator-managed migration only as a future compatible per-module implementation with exhaustive DTO/client tests. This is a known, explicit limitation rather than an unverified claim.

## Completion Report

The scoped API route inventory, HTTP middleware integration and error handling changes are complete and CI verified. No server deployment occurred in this goal.
