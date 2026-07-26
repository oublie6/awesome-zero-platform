# Goal 0006: Platform Security and Production Foundation

## Status

- State: completed
- Started: 2026-07-26
- Completed: 2026-07-26
- Blockers:

## Goal

Implement the remaining non-business and non-client platform capabilities required for secure authentication, replaceable authorization, observability, continuous integration, and production-oriented deployment.

## References

- `AGENTS.md`
- `docs/goals/requirements/0006-platform-security-production.md`
- `docs/architecture/security-platform.md`
- `docs/api/authentication.md`
- `docs/operations/production-deployment.md`

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
- Do not modify previous archived goals.

## Architecture Outcome

Authentication and authorization are separate platform capabilities with inward-facing dependency direction.

`server/platform/authn` owns the authentication use cases and depends only on three ports:

- `IdentityProvider`
- `AccessTokenCodec`
- `SessionStore`

Concrete adapters provide identity account access, HMAC JWT access tokens, and Redis-backed rotating sessions. Replacing JWT or Redis requires a new adapter and bootstrap composition change, not a rewrite of authentication use cases or HTTP handlers.

`server/platform/authz` owns authorization use cases and depends only on:

- `Authorizer`
- `PolicyManager`

Casbin with MySQL persistence is the first adapter. Casbin types do not leak into HTTP transport, authentication, identity, or callers. Another policy engine can replace it by implementing the same contracts.

The application bootstrap is the composition root. Transport packages map HTTP requests and responses but contain no token signing, session persistence, password verification, or Casbin policy implementation.

## Completed Implementation

### Authentication

- Added login, refresh, logout, and current-session routes.
- Added short-lived HMAC-SHA256 access tokens with issuer, subject, session ID, issued-at, and expiry claims.
- Added opaque 256-bit refresh tokens while storing only SHA-256 digests.
- Added Redis sessions with compare-and-swap refresh rotation.
- Added server-side session checks for every authenticated request.
- Added active-account checks so disabled accounts lose access immediately.
- Added logout revocation before access-token expiry.
- Kept invalid account identifiers, missing accounts, disabled accounts, and wrong passwords indistinguishable to clients.
- Preserved infrastructure failures as internal errors rather than misreporting them as invalid credentials.
- Added constant-time refresh-digest comparison.
- Deliberately did not add public registration, password recovery, MFA, social login, SSO, or account-management endpoints.

### Authorization

- Added subject/resource/action authorization contracts.
- Added role assignment and revocation application methods.
- Added permission grant and revocation application methods.
- Added Casbin RBAC with path-pattern and action-pattern matching.
- Added MySQL policy persistence owned exclusively by the authorization adapter.
- Added bounded SHA-256 rule fingerprints to enforce uniqueness without exceeding InnoDB utf8mb4 index limits.
- Kept role and permission administration internal until a real administration product is designed.

### Observability and Operations

- Added Prometheus-compatible request counters and duration histograms.
- Added Go runtime and process collectors.
- Preserved process-only liveness and dependency-aware readiness.
- Added a real executable HTTP liveness check for container health checks.
- Added environment overrides for deployment-specific addresses, users, passwords, ports, and access-token secrets.

### Deployment and CI

- Added a multi-stage, non-root, distroless application image.
- Added persistent production Compose services, schema application, health checks, read-only application filesystem, and external secret injection.
- Added Kubernetes Deployment, Service, probes, resource limits, security context, Prometheus annotations, and PodDisruptionBudget.
- Added strict GitHub Actions checks for module drift, generated-code drift, formatting, unit tests, focused race tests, build, production Compose validation, current schema application, and real MySQL/Redis integration tests.

### Documentation

- Added security architecture documentation.
- Added authentication HTTP API documentation.
- Added production deployment documentation.
- Updated the repository overview with current platform capabilities and explicit deferred scope.

## Verification Evidence

GitHub Actions run `30186202199` completed successfully on 2026-07-26.

The unit job passed:

- module-file verification through `go mod tidy` with no diff;
- repeatable goctl generation with no generated-code diff;
- `make fmt-check`;
- `make test` using low-concurrency package execution;
- focused `go test -race` for authentication and authorization packages;
- `make build` with a static application binary;
- production Compose interpolation and schema validation through `docker compose config`.

The integration job passed:

- clean MySQL and Redis startup;
- complete schema and development seed application;
- existing HTTP, database, cache, readiness, and identity integration tests;
- authentication login and access validation;
- refresh-token rotation and previous-token rejection;
- disabled-account session revocation;
- logout revocation;
- Casbin role assignment, permission enforcement, MySQL persistence, reload, duplicate prevention, and permission revocation;
- dependency teardown.

No production credentials, generated binaries, runtime databases, temporary SQL patches, business modules, or client applications were committed.

## Completion Report

Goal 0006 completed the reusable security and production foundation while preserving the modular-monolith architecture and explicit package ownership rules. Authentication, authorization, storage, policy engines, token formats, transport, and deployment are separated behind narrow contracts. Casbin, JWT, Redis, and go-zero remain adapters or composition concerns rather than irreversible dependencies of the platform application layer.

Product-specific business capabilities and Vue 3, WeChat Mini Program, H5, and app clients remain intentionally deferred.
