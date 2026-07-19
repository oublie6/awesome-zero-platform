# Server Bootstrap Requirements

## Purpose

This document defines the minimum runnable server foundation for Awesome Zero Platform. It establishes the project conventions that later platform and business modules will build on without introducing database, cache, authentication, authorization, client, or deployment complexity prematurely.

## Scope

The first server implementation must provide a single go-zero REST process under `server/apps/app-api` and the smallest supporting project structure required to build, test, configure, run, and stop it reliably.

## Target structure

```text
server/
├── apps/
│   └── app-api/
│       ├── etc/
│       ├── internal/
│       ├── app.api
│       └── main.go
├── platform/
├── foundation/
├── database/
├── go.mod
└── README.md
```

Only directories required by implemented files should be committed. Do not create empty platform, business, database, or client scaffolding merely to match the target diagram.

## Runtime behavior

The API process must:

- start from a YAML configuration file;
- bind to a configurable host and port;
- expose `GET /health/live` for process liveness;
- expose `GET /health/ready` for service readiness;
- return deterministic JSON responses for both health endpoints;
- stop gracefully when receiving `SIGINT` or `SIGTERM`;
- return a non-zero exit status when startup configuration is invalid;
- avoid committed secrets and environment-specific credentials.

For this phase, readiness means that the HTTP service has initialized successfully. Database, Redis, message queue, and downstream dependency checks are intentionally deferred.

## go-zero conventions

- Use go-zero REST and its API DSL.
- Keep generated go-zero files clearly distinguishable from handwritten files.
- Treat the `.api` file as the HTTP contract source for this service.
- Do not hand-edit generated files when the same result can be obtained by changing the `.api` definition and regenerating.
- Document the exact `goctl` generation command.
- Pin module dependencies through `go.mod` and `go.sum`.

## Configuration

The development configuration should live under `server/apps/app-api/etc` and contain only safe local defaults.

Configuration must support at least:

- service name;
- listen host;
- listen port;
- go-zero REST service settings needed by the generated application.

Do not add a custom configuration framework in this phase. Use go-zero's standard configuration loading unless a concrete limitation is demonstrated.

## Health API contract

### `GET /health/live`

Indicates that the process is running and capable of serving HTTP requests.

Expected HTTP status: `200 OK`.

Minimum response fields:

```json
{
  "status": "ok"
}
```

### `GET /health/ready`

Indicates that server initialization has completed and the process is ready to receive traffic.

Expected HTTP status: `200 OK`.

Minimum response fields:

```json
{
  "status": "ready"
}
```

The implementation may add stable fields such as service name or version only when they are useful and tested. Do not invent a general response envelope in this goal; unified response and error conventions belong to a later foundation goal.

## Engineering commands

The repository root should provide a Makefile with stable entry points for at least:

- `make generate`
- `make run`
- `make test`
- `make fmt`

Commands must work from the repository root and delegate into `server` as needed.

## Testing

At minimum, tests must verify:

- health handlers return the intended HTTP status;
- health response bodies contain the expected status value;
- configuration or service construction code fails clearly for invalid required settings when practical;
- `go test ./...` passes from `server`.

Prefer focused tests over introducing a broad testing framework.

## Documentation

Update `server/README.md` with:

- required Go and goctl prerequisites;
- dependency installation instructions;
- API generation command;
- local run command;
- test and formatting commands;
- health endpoint examples;
- the distinction between generated and handwritten files.

Update the repository root README only when necessary to point to the runnable server instructions.

## Explicit non-goals

This bootstrap must not add:

- PostgreSQL, MySQL, Redis, Kafka, or other infrastructure clients;
- authentication, sessions, JWT, Casbin, users, roles, or permissions;
- platform or business modules;
- Vue, miniapp, H5, or app clients;
- Docker, Kubernetes, CI workflows, metrics, tracing, or custom logging abstractions;
- generic repository, CRUD, service, middleware, response, or error frameworks;
- microservices, RPC services, workers, schedulers, or message consumers.

These capabilities will be introduced by later goals after the runnable server boundary is stable.
