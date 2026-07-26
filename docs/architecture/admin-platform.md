# Admin platform architecture

## Purpose

The Admin platform is the reusable control plane for Awesome Zero Platform. It owns platform administration, not product business workflows. Future Vue user clients and Mini Program clients stay separate and may reuse only stable authentication and API contracts.

## Two information densities, one source of truth

Admin modules expose a standard view for routine administrators and a technical view for backend and security engineers. These are presentation modes over the same application services:

```text
Vue standard view ─┐
                   ├─ Admin HTTP API ─ Admin application service ─ platform ports
Vue expert view ───┘                                      ├─ identity
                                                         ├─ authn sessions
                                                         └─ authz plugin
```

The expert view never writes adapter tables directly. Casbin policies are validated and applied through the authorization plugin, which persists through its adapter and refreshes its in-memory enforcer.

## Server packages

- `server/platform/admin` coordinates account, role, resource, session, audit, bootstrap, and policy-administration use cases.
- `server/platform/admin/mysqlstore` owns role metadata, resource catalog, and audit persistence.
- `server/platform/identity` remains the owner of accounts and password credentials.
- `server/platform/authn` exposes a narrow session-administration port.
- `server/platform/authz` exposes a separate `Administrator` port. Request authorization still depends only on `Authorizer`.
- `server/apps/app-api/internal/adminhttp` maps the Admin HTTP contract and enforces authentication plus the current request path and method.

This separation keeps normal request paths independent from expert tooling and keeps Casbin replaceable.

## Authorization views

The standard authorization view manages a resource catalog and role permissions. A resource defines a stable code, module, display name, Casbin-compatible resource pattern, and allowed actions. Saving a role permission produces `p` rules through the plugin.

The expert authorization view exposes engine capabilities, the read-only model, raw `p` and `g` rules, batch validation/replacement, and an Enforce explanation endpoint. Replacement is blocked unless it preserves:

- the `platform_super_admin, /*, .*` wildcard permission;
- at least one `g` membership for `platform_super_admin`.

## Bootstrap

No default account or password is committed. When `APP_ADMIN_BOOTSTRAP_TOKEN` is configured and no super administrator exists, a one-time public bootstrap endpoint can create the first account. The token is compared by SHA-256 digest using constant-time comparison. After bootstrap, remove the environment variable.

## Client structure

`clients/admin-web` is a Vue 3, TypeScript, Vite, Pinia, Vue Router, and Element Plus application. Its HTTP client encapsulates token storage and single-flight refresh. Access and refresh tokens are stored in session storage because the current server returns JSON tokens; this vault can be replaced when refresh tokens move to HttpOnly cookies.

## Safety boundaries

- Backend authorization is authoritative; hidden menus and buttons are usability only.
- Passwords and tokens are never written to audit events.
- Secrets are never returned by configuration diagnostics.
- System resources and the last super administrator cannot be removed.
- Account disable and password reset revoke active Redis sessions.
- Raw policy replacement is validated before persistence and retains the previous in-memory policy if saving fails.
