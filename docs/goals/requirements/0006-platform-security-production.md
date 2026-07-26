# Goal 0006 Requirements: Platform Security and Production Foundation

## Objective

Complete the non-business, non-client platform foundation required to run Awesome Zero Platform as a secure, observable, production-oriented modular monolith.

## Scope

1. Public authentication endpoints for login, refresh, logout, and current-session inspection.
2. Redis-backed rotating sessions with short-lived signed access tokens.
3. Pluggable authentication contracts so token signing and session persistence can be replaced independently.
4. Pluggable authorization contracts with Casbin as the first policy-engine adapter.
5. HTTP authentication and authorization middleware that depends only on platform contracts.
6. Role and permission management at the platform-service layer without exposing speculative business APIs.
7. Prometheus-compatible HTTP metrics and health-safe operational endpoints.
8. Production container image, production Compose baseline, Kubernetes baseline manifests, environment-based secret injection, and graceful startup/shutdown compatibility.
9. GitHub Actions baseline CI for generation drift, formatting, unit tests, and integration tests.
10. Documentation, deterministic schema, and focused tests.

## Explicitly Deferred

- Product-specific business modules.
- Generic business CRUD, file, notification, workflow, dictionary, or audit-log products.
- Vue 3, WeChat Mini Program, H5, or app clients.
- Public self-registration and password recovery.
- Social login, MFA, SSO, OIDC provider behavior, and multi-tenancy.
- A production ingress controller, certificate issuer, managed database, or managed Redis vendor choice.

## Architecture Constraints

- Follow SOLID principles through narrow interfaces and dependency inversion.
- Domain/application code must not import Casbin, JWT libraries, Redis drivers, or HTTP-framework-specific types.
- `server/platform/authn` owns authentication application behavior and ports.
- `server/platform/authz` owns authorization application behavior and ports.
- Vendor implementations live under explicit adapter packages.
- Transport code performs request/response mapping only.
- Existing identity tables remain owned by `server/platform/identity`.
- Authorization policy persistence must not be queried directly by other modules.
- No generic `common`, `utils`, `helpers`, repository framework, or service-locator package.

## Security Decisions

- Password authentication delegates verification to the identity capability.
- Access tokens are HMAC-SHA256 signed, intentionally short lived, and carry only subject, session ID, issued-at, expiry, and issuer claims.
- Refresh tokens are opaque random values. Only SHA-256 digests are stored in Redis.
- Refresh rotates the refresh token and invalidates the previous digest.
- Logout revokes the server-side session; access-token validation always checks that the referenced session remains active.
- Disabled accounts cannot create or continue sessions.
- Authentication errors do not reveal whether an account or password was incorrect.
- Authorization uses subject, resource, and action contracts. Casbin is an adapter, not a platform-wide dependency.
- Secrets must come from environment variables in production examples and must not be committed.

## Acceptance Criteria

- `make generate`, `make fmt-check`, and `make test` pass.
- Authentication and authorization packages have deterministic unit coverage.
- Real MySQL/Redis integration tests cover login, refresh rotation, logout revocation, disabled-account rejection, Casbin enforcement, and health behavior.
- Public routes include login, refresh, logout, current session, and metrics; no registration endpoint is added.
- Protected-route middleware validates signed tokens, active Redis sessions, and account state.
- Authorization middleware depends on an `Authorizer` interface and works with the Casbin adapter.
- Schema remains a complete rebuildable current schema.
- Docker, Compose, Kubernetes, and CI assets contain no real credentials.
