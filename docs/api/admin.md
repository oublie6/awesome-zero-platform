# Admin API

All responses use the platform envelope `{code,message,requestId,data}`. Except for bootstrap status and bootstrap creation, endpoints require a Bearer access token and a matching authorization policy for the concrete request path and HTTP method.

## First administrator

- `GET /admin/bootstrap/status`
- `POST /admin/bootstrap` with header `X-Admin-Bootstrap-Token`

Bootstrap is available only while a deployment token is configured and no member of `platform_super_admin` exists.

## Current administrator

- `GET /admin/me` returns account, role codes, and effective role permissions.

## Accounts

- `GET /admin/accounts`
- `POST /admin/accounts`
- `GET /admin/accounts/:id`
- `PATCH /admin/accounts/:id`
- `POST /admin/accounts/:id/enable`
- `POST /admin/accounts/:id/disable`
- `POST /admin/accounts/:id/reset-password`
- `GET /admin/accounts/:id/sessions`
- `POST /admin/accounts/:id/revoke-sessions`
- `GET /admin/accounts/:id/roles`
- `PUT /admin/accounts/:id/roles`

Account deletion is deliberately absent. Disable preserves references and audit history. Disable and password reset revoke sessions.

## Roles and standard permissions

- `GET /admin/roles`
- `POST /admin/roles`
- `GET /admin/roles/:code`
- `PATCH /admin/roles/:code`
- `DELETE /admin/roles/:code`
- `GET /admin/roles/:code/permissions`
- `PUT /admin/roles/:code/permissions`

Permissions are `{resource,action}` values and currently map to Casbin `p` rules.

## Resource catalog

- `GET /admin/authorization/resources`
- `POST /admin/authorization/resources`
- `GET /admin/authorization/resources/:code`
- `PATCH /admin/authorization/resources/:code`
- `DELETE /admin/authorization/resources/:code`

The catalog drives the standard UI. It does not duplicate policy state.

## Authorization engine expert API

- `GET /admin/authorization/engine`
- `GET /admin/authorization/engine/model`
- `GET /admin/authorization/engine/policies`
- `POST /admin/authorization/engine/policies/validate`
- `PUT /admin/authorization/engine/policies`
- `POST /admin/authorization/engine/explain`

Raw rules use `{ptype,values}` so other plugins can expose native policies without changing normal authorization APIs. The model is read-only in this release.

## Operations

- `GET /admin/audit/events`
- `GET /admin/system/overview`
- `GET /admin/system/configuration`

Configuration diagnostics return effective non-secret settings and booleans for secret presence.
