# HTTP Foundation Requirements

## Purpose

Define the reusable HTTP request and response conventions that every future API module in Awesome Zero Platform must follow. This foundation establishes consistent error handling, response envelopes, request identity, recovery behavior, access logging, CORS, security headers, and request-size limits before database, cache, authentication, authorization, or business capabilities are introduced.

## Scope

This requirement applies to the existing `server/apps/app-api` process and reusable technical components under `server/foundation`.

The implementation must remain compatible with go-zero REST conventions and must not introduce a parallel web framework.

## Package boundaries

The initial reusable structure should be organized around focused packages such as:

```text
server/foundation/
├── errors/
├── response/
├── requestid/
└── middleware/
```

Exact package names may be refined during implementation when required by Go naming or go-zero integration, but generic dumping grounds such as `common`, `utils`, or `helpers` are prohibited.

## Response envelope

Successful API responses should use a stable JSON envelope:

```json
{
  "code": "OK",
  "message": "success",
  "requestId": "01J...",
  "data": {}
}
```

Error responses should use the same top-level structure:

```json
{
  "code": "PARAM_INVALID",
  "message": "invalid parameter",
  "requestId": "01J...",
  "data": null
}
```

Requirements:

- `code` is a stable machine-readable application code.
- `message` is safe for clients and must not expose stack traces, secrets, infrastructure details, or raw internal errors.
- `requestId` identifies the request across response headers, response bodies, and logs.
- `data` contains the successful payload and is `null` for errors unless a documented endpoint contract requires structured error details.
- HTTP status codes must remain semantically correct; the application code does not replace HTTP status.

## Error model

The foundation must provide an explicit application error type or equivalent focused abstraction containing at least:

- stable application code
- safe client message
- HTTP status
- wrapped internal cause when present

Initial reusable categories should include at least:

- invalid parameter
- unauthorized
- forbidden
- not found
- conflict
- request too large
- internal error

Unknown errors must map to a generic internal error response while preserving the original cause for server-side logging.

Do not build a speculative hierarchy for every future business error. Platform and business modules will define their own stable codes while reusing the foundation mapping mechanism.

## Request ID

Every request must have exactly one effective request ID.

Behavior:

1. Accept a caller-provided request ID from a documented request header when it is non-empty and within a safe bounded length.
2. Generate a new request ID when the caller does not provide a valid value.
3. Store the effective request ID in the request context.
4. Return it in a documented response header.
5. Include it in the JSON response envelope.
6. Include it in request-related logs.

The implementation should use a compact, collision-resistant identifier suitable for distributed systems, but must not introduce database or Redis dependencies.

## Panic recovery

Unexpected panics in request handling must:

- be recovered at the HTTP boundary
- produce HTTP 500 with the standard internal-error envelope
- include the effective request ID
- log the panic and stack trace server-side
- avoid leaking the panic text or stack trace to clients
- keep the process available for subsequent requests where the runtime permits

Prefer go-zero-supported recovery hooks or middleware integration over replacing the server lifecycle.

## Access logging

Each completed request should produce a structured access log containing at least:

- request ID
- HTTP method
- path
- status code
- elapsed duration
- remote address or trusted client address according to a clearly documented policy

Do not log request or response bodies by default. Sensitive headers such as authorization and cookies must not be logged.

## CORS

CORS behavior must be configurable and disabled or conservative by default.

Configuration should support at least:

- allowed origins
- allowed methods
- allowed headers
- exposed headers
- whether credentials are permitted

Wildcard origins must not be combined with credentialed requests.

## Security headers

The API should return a conservative baseline of configurable security headers appropriate for JSON APIs, including at least:

- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Referrer-Policy: no-referrer`

Do not add browser policies that break legitimate API clients without documenting and testing the behavior.

## Request body limit

The server must enforce a configurable maximum request body size before future upload capabilities are introduced.

Oversized requests must return:

- HTTP 413
- the standard error envelope
- the effective request ID
- a stable application error code

The default should be safe for ordinary JSON APIs and must be documented.

## Health endpoint policy

`GET /health/live` and `GET /health/ready` are infrastructure endpoints. They may retain their minimal existing payloads to remain simple for probes, but they must still receive and return the request ID response header and benefit from recovery, access logging, security headers, and other globally safe middleware.

The implementation must document this deliberate envelope exception if the health payloads remain unchanged.

## Configuration

Middleware-related configuration belongs in the existing application configuration model. Startup validation must reject unsafe or internally inconsistent configuration where practical, including invalid request-size limits and invalid CORS combinations.

Committed configuration must remain safe for local development and contain no secrets.

## Testing expectations

Tests must cover at least:

- generated request ID
- accepted caller request ID
- rejection or replacement of invalid caller request IDs
- propagation to response header and response envelope
- successful response envelope
- application error mapping
- unknown error mapping
- panic recovery and safe client response
- request body size rejection
- security headers
- CORS allowed and denied cases
- access-log fields through an appropriate testable boundary
- startup validation for relevant configuration

Prefer focused unit tests and lightweight HTTP handler tests. Do not introduce external services for this requirement.

## Deferred capabilities

This requirement intentionally excludes:

- PostgreSQL or MySQL
- Redis
- authentication and authorization
- users, sessions, JWT, or Casbin
- object storage and file uploads
- Prometheus metrics
- distributed tracing backends
- CI, Docker, Kubernetes, or deployment manifests
- business modules
