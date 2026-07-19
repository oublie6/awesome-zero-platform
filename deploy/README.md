# Deploy

Deployment assets live here and must remain independent from business code.

Current areas:

- `local/` — local development orchestration
- `docker/` — container build definitions
- `kubernetes/` — Kubernetes manifests or Helm charts when needed

Database schema definitions remain under `server/database`. One-off database upgrade SQL is treated as a runtime artifact and must not be committed.

## Local development dependencies

`local/docker-compose.yml` pins the MySQL and Redis images used for Goal 0004 local development:

- MySQL `8.4.10`
- Redis `8.8.0-alpine3.23`

The committed credentials are development-only and exist only to support deterministic local startup:

- MySQL database: `awesome_zero_platform`
- MySQL user: `app_local`
- MySQL password: `local-dev-only-mysql-password`
- MySQL root password: `local-dev-only-mysql-root-password`
- Redis password: `local-dev-only-redis-password`

The compose file uses `tmpfs` mounts so committed runtime data and container volumes are not preserved.

Recommended lifecycle from the repository root:

```bash
make deps-up
make schema-apply
make seed-apply
make integration-test
make deps-down
```

Use `make deps-reset` when you deliberately want to recreate the local dependency containers and drop their transient data.
