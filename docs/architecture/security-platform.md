# Security Platform Architecture

## Purpose

The security foundation is split into authentication (`authn`) and authorization (`authz`) platform capabilities. Both are designed around narrow application-facing contracts so storage engines, token formats, identity sources, and policy engines can be replaced without rewriting transport or business logic.

## Package boundaries

```text
server/platform/
  identity/                       # account profile and password ownership
  authn/
    authn.go                      # authentication use cases and ports
    adapter/
      identityprovider/           # identity.Service adapter
      jwthmac/                    # HMAC JWT access-token adapter
      redissession/               # Redis rotating-session adapter
  authz/
    authz.go                      # authorization use cases and ports
    adapter/
      casbinmysql/                # Casbin + MySQL policy adapter

server/apps/app-api/internal/
  securityhttp/                   # HTTP request mapping and route middleware
  bootstrap/                      # concrete adapter composition
```

## Dependency direction

The core dependency direction is inward:

```text
HTTP transport -> authn/authz contracts <- adapters
                     ^
                     |
                 bootstrap
```

`authn` does not import JWT, Redis, MySQL, go-zero, or the identity implementation. It depends on:

- `IdentityProvider`
- `AccessTokenCodec`
- `SessionStore`

`authz` does not import Casbin, MySQL, or HTTP types. It depends on:

- `Authorizer`
- `PolicyManager`

Only `bootstrap` chooses the concrete adapters. Replacing Casbin therefore requires a new adapter implementing `Authorizer` and `PolicyManager`, followed by a composition change in `bootstrap`; authentication middleware and callers remain unchanged.

## Authentication flow

### Login

1. The HTTP transport validates the request shape.
2. `authn.Service` calls `IdentityProvider.Authenticate`.
3. A UUIDv7 session identifier and opaque random refresh token are generated.
4. Only the SHA-256 refresh-token digest is persisted.
5. A short-lived access token is signed by `AccessTokenCodec`.
6. The response returns the access token, refresh token, expirations, and minimal account identity.

### Access validation

1. The access-token adapter validates signature, issuer, algorithm, and expiry.
2. `authn.Service` verifies that the referenced server-side session still exists.
3. The identity adapter confirms the account is still active.
4. The resulting `Authentication` is attached to the request context.

This means logout and account disablement take effect without waiting for access-token expiry.

### Refresh rotation

1. The opaque token is parsed into its session identifier and random secret.
2. Its digest must match the active session record.
3. The account must still be active.
4. A new refresh token and access token are created.
5. The session store performs compare-and-swap rotation so the previous refresh token cannot be reused.

### Logout

Logout validates the access token and deletes the corresponding server-side session. Subsequent access validation fails even if the signed token has not expired.

## Authorization model

The application contract is deliberately small:

```text
Enforce(subject, resource, action) -> allowed
```

The first adapter uses Casbin with an RBAC model:

- account IDs are subjects
- roles are grouping targets
- resources are path-like identifiers
- actions are verb-like identifiers
- `keyMatch2` supports resource patterns
- regular-expression matching supports action sets

Policy storage is owned by the authorization adapter through `authorization_casbin_rules`. Other modules must not query that table directly.

## Operational foundation

- Prometheus-compatible request counters and latency histograms are exposed through a configurable metrics route.
- Liveness remains process-only.
- Readiness checks MySQL and Redis.
- Production examples inject secrets through environment variables or Kubernetes Secrets.
- The application container runs as a non-root user with a read-only root filesystem baseline.
- GitHub Actions runs generated-code drift checks, formatting checks, unit tests, a low-concurrency build, and real MySQL/Redis integration tests.

## Deliberately deferred

The platform does not yet provide public registration, password recovery, MFA, SSO, OIDC provider behavior, social login, multi-tenancy, or product-specific administration APIs. Those capabilities should be introduced as separate goals against the existing contracts instead of broadening the core packages speculatively.
