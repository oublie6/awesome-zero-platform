# Authentication HTTP API

All ordinary responses use the standard platform envelope:

```json
{
  "code": "OK",
  "message": "success",
  "requestId": "request-id",
  "data": {}
}
```

Authentication endpoints intentionally do not provide public registration, password recovery, account administration, role administration, or permission administration.

## Login

`POST /auth/login`

Request:

```json
{
  "identifier": "alice@example.com",
  "password": "user-password"
}
```

`identifier` may be a configured username, email address, or explicit E.164 phone number.

Successful response data:

```json
{
  "accessToken": "signed-access-token",
  "refreshToken": "opaque-refresh-token",
  "tokenType": "Bearer",
  "accessExpiresAt": "2026-07-26T03:00:00Z",
  "refreshExpiresAt": "2026-08-25T02:45:00Z",
  "account": {
    "id": "01984f63-ec7f-7a4a-b908-33e8ff14d465",
    "displayName": "Alice"
  }
}
```

Invalid account identifiers, passwords, disabled accounts, and missing accounts all return the same HTTP `401` response and do not reveal which value was incorrect.

## Refresh

`POST /auth/refresh`

Request:

```json
{
  "refreshToken": "opaque-refresh-token"
}
```

A successful refresh returns the same data shape as login. Refresh tokens rotate: after a successful refresh, the previous refresh token is invalid and must be discarded.

## Current session

`GET /auth/session`

Header:

```http
Authorization: Bearer signed-access-token
```

Successful response data:

```json
{
  "sessionId": "01984f63-ec7f-7a4a-b908-33e8ff14d466",
  "expiresAt": "2026-07-26T03:00:00Z",
  "account": {
    "id": "01984f63-ec7f-7a4a-b908-33e8ff14d465",
    "displayName": "Alice"
  }
}
```

Access validation checks all three conditions:

1. The signed access token is valid and unexpired.
2. The referenced Redis session still exists.
3. The identity account is still active.

Therefore logout and account disablement take effect before the signed token's nominal expiry.

## Logout

`POST /auth/logout`

Header:

```http
Authorization: Bearer signed-access-token
```

Successful response data:

```json
{
  "loggedOut": true
}
```

Logout revokes the server-side session. Reusing the access token then returns HTTP `401`.

## Error behavior

- Invalid request shape: HTTP `400`, code `PARAM_INVALID`.
- Missing, invalid, expired, revoked, or unavailable authentication: HTTP `401`, code `UNAUTHORIZED`.
- Authorization denied by a protected route: HTTP `403`, code `FORBIDDEN`.
- Dependency or internal failure: HTTP `500`, code `INTERNAL_ERROR`, without leaking database, Redis, token, or stack details.

## Transport boundary

Authentication routes are registered in the handwritten `securityhttp` transport package rather than generated from the bootstrap health-only goctl contract. This keeps generated health scaffolding stable while the reusable authentication application service remains independent from go-zero and HTTP types.
