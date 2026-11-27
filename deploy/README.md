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

## Static Web gzip

Include `game-web/gzip-static.nginx.conf` inside the public `/game/` and
`/assets/` locations of the static Web server. Package it as
`/etc/nginx/static-gzip.conf` and use `include /etc/nginx/static-gzip.conf;`
in those locations. Preserve the existing TLS, CSP and cache configuration.
Do not include the fragment at server/http scope or in API/auth/WebSocket proxies.

This enables negotiated gzip at level 6 for JS, CSS, JSON and the default HTML
type, with a 1024-byte minimum and `Vary: Accept-Encoding`. Only the Web image
needs repackaging; application assets and API binaries remain unchanged.
Verify gzip and identity responses, decompressed hashes, cache revalidation and
authenticated API behavior before rollout. Keep the previous immutable image
for rollback. With curl, use `--compressed` to request and decode compression.

The 2026-11-27 test deployment reduced `cc.js` from 12,622,964 to 3,375,311
transmitted bytes. This reduces transfer size; it does not measure or guarantee
the user's SSH bandwidth. [Runtime evidence](../docs/operations/evidence/web-gzip-20261127/rollout-result.json).
