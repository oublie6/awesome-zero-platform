# Production Deployment Baseline

The repository provides deployment baselines rather than prescribing a specific cloud, managed database, ingress controller, certificate issuer, or secret manager.

## Required secrets

Production examples require these values to be supplied outside Git:

- `APP_MYSQL_PASSWORD`
- `APP_MYSQL_ROOT_PASSWORD` for the bundled production Compose MySQL only
- `APP_REDIS_PASSWORD`
- `APP_AUTH_ACCESS_TOKEN_SECRET`

`APP_ADMIN_BOOTSTRAP_TOKEN` is required only for the first-administrator bootstrap. Remove it and recreate the API containers after bootstrap succeeds.

The access-token secret must contain at least 32 characters and should be generated from cryptographically secure random bytes. Rotating it invalidates all existing signed access tokens; Redis sessions remain but cannot be used without a newly issued access token.

## Production Compose

From the repository root, create an uncommitted environment file or export the required variables, then run:

```bash
export APP_MYSQL_PASSWORD='replace-me'
export APP_MYSQL_ROOT_PASSWORD='replace-me'
export APP_REDIS_PASSWORD='replace-me'
export APP_AUTH_ACCESS_TOKEN_SECRET='replace-with-a-long-random-secret'
export APP_ADMIN_BOOTSTRAP_TOKEN='replace-with-a-temporary-long-random-token'

docker compose -f deploy/production/docker-compose.yml up -d --build --wait
```

The Compose baseline:

- uses `mysql:5.7.44` with a 64 MiB InnoDB buffer pool, disabled Performance Schema, reduced connections, and disabled binary logging so it can run on a small development machine;
- starts persistent MySQL and Redis services;
- waits for dependency health checks;
- applies the complete current schema through a one-shot schema container;
- starts the non-root, read-only application container;
- performs a real HTTP liveness check through the application's `-healthcheck` command;
- enables go-zero Redis caching for ordinary entity detail reads;
- enables clustered Casbin synchronization even though the baseline Compose file starts one API instance by default.

MySQL data directories are not downgrade-compatible across major versions. If an existing Compose volume was initialized by MySQL 8.x, remove or migrate that volume before starting the 5.7 baseline:

```bash
docker compose -f deploy/production/docker-compose.yml down --volumes --remove-orphans
```

Do not run that command against data that has not been backed up. The bundled database is a low-memory deployment baseline, not a recommendation to downgrade a managed production database.

This is a baseline for a single-host deployment. Backups, TLS termination, firewall policy, log shipping, monitoring retention, and disaster recovery must be chosen for the actual environment.

## Data cache behavior

Ordinary entity detail reads use the shared Redis cache through the persistence adapter. Account, role, and authorization-resource models provide:

- cached primary or unique-key reads;
- explicit fresh reads that bypass Redis;
- transaction and `FOR UPDATE` reads that bypass Redis;
- one write path that modifies MySQL first and then invalidates all affected primary and old/new unique-index keys.

Passwords, sessions, audit logs, lists, searches, aggregates, and Casbin policy rows do not use ordinary row caching. Authentication status checks use fresh account reads.

## Clustered authorization

MySQL remains the durable Casbin policy source. Each API replica keeps a `SyncedEnforcer` in memory, so normal `Enforce` calls do not query MySQL or Redis.

Policy writes:

1. lock the global row in `authorization_policy_state`;
2. read and validate the latest policy, including protected active-super-administrator invariants;
3. commit the policy and increment its version in one MySQL transaction;
4. reload the local Enforcer;
5. publish the committed version through Redis Pub/Sub.

Other replicas reload when they observe a newer version. Periodic MySQL version polling repairs missed Pub/Sub messages. A stale or failed policy reload makes `/health/ready` fail while the previous in-memory policy remains intact.

## Kubernetes

`deploy/kubernetes/base.yaml` and `deploy/kubernetes/admin-web.yaml` contain:

- the shared `awesome-zero-platform` namespace;
- two-replica API and Admin Web Deployments;
- readiness and liveness probes;
- non-root and read-only filesystem security settings;
- CPU and memory requests and limits;
- Prometheus scrape annotations;
- ClusterIP Services;
- an API PodDisruptionBudget;
- Pod-name injection through `APP_INSTANCE_ID` for authorization synchronization diagnostics.

The Admin Web Nginx container proxies `/api/` to `app-api:8888`; the Kubernetes API Service exposes the matching port and balances requests across API replicas. Authentication does not require sticky sessions because sessions are stored in Redis.

Before applying the manifests:

1. Replace the example image references with immutable version tags or digests.
2. Create `awesome-zero-platform-secrets` in the target namespace with `mysql-user`, `mysql-password`, `redis-password`, and `access-token-secret` keys.
3. Provide reachable MySQL and Redis Services or change `APP_MYSQL_ADDR` and `APP_REDIS_ADDR`.
4. Apply `server/database/schema/current.sql` through a deployment Job or CI/CD database stage before rolling out a version that requires it. API Pods must not modify schema during startup.
5. Add the environment-specific ingress, TLS, network policies, backup policy, and monitoring stack.

The committed manifests deliberately do not contain a Secret object or plaintext production credentials.

## Probes and metrics

- Liveness: `GET /health/live`
- Readiness: `GET /health/ready`
- Metrics: `GET /metrics`

Readiness depends on MySQL, Redis, and current Casbin policy synchronization. Metrics expose process, Go runtime, request-count, and request-duration series. The metrics endpoint should normally be restricted to the monitoring network rather than exposed publicly.

## Configuration overrides

The production YAML provides non-secret defaults. These environment variables override deployment-specific values:

```text
APP_HOST
APP_PORT
APP_INSTANCE_ID
APP_MYSQL_ADDR
APP_MYSQL_DATABASE
APP_MYSQL_USER
APP_MYSQL_PASSWORD
APP_REDIS_ADDR
APP_REDIS_USERNAME
APP_REDIS_PASSWORD
APP_AUTH_ACCESS_TOKEN_SECRET
APP_ADMIN_BOOTSTRAP_TOKEN
```

No production credential should be added to `production.yaml`, Compose files, Kubernetes manifests, Docker build arguments, source code, or CI logs.
