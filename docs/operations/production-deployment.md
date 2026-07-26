# Production Deployment Baseline

The repository provides deployment baselines rather than prescribing a specific cloud, managed database, ingress controller, certificate issuer, or secret manager.

## Required secrets

Production examples require these values to be supplied outside Git:

- `APP_MYSQL_PASSWORD`
- `APP_MYSQL_ROOT_PASSWORD` for the bundled production Compose MySQL only
- `APP_REDIS_PASSWORD`
- `APP_AUTH_ACCESS_TOKEN_SECRET`

The access-token secret must contain at least 32 characters and should be generated from cryptographically secure random bytes. Rotating it invalidates all existing signed access tokens; Redis sessions remain but cannot be used without a newly issued access token.

## Production Compose

From the repository root, create an uncommitted environment file or export the required variables, then run:

```bash
export APP_MYSQL_PASSWORD='replace-me'
export APP_MYSQL_ROOT_PASSWORD='replace-me'
export APP_REDIS_PASSWORD='replace-me'
export APP_AUTH_ACCESS_TOKEN_SECRET='replace-with-a-long-random-secret'

docker compose -f deploy/production/docker-compose.yml up -d --build
```

The Compose baseline:

- starts persistent MySQL and Redis services;
- waits for dependency health checks;
- applies the complete current schema through a one-shot schema container;
- starts the non-root, read-only application container;
- performs a real HTTP liveness check through the application's `-healthcheck` command.

This is a baseline for a single-host deployment. Backups, TLS termination, firewall policy, log shipping, monitoring retention, and disaster recovery must be chosen for the actual environment.

## Kubernetes

`deploy/kubernetes/base.yaml` contains:

- a namespace;
- a two-replica Deployment;
- readiness and liveness probes;
- non-root and read-only filesystem security settings;
- CPU and memory requests and limits;
- Prometheus scrape annotations;
- a ClusterIP Service;
- a PodDisruptionBudget.

Before applying it:

1. Replace the example image reference with an immutable version tag or digest.
2. Create `awesome-zero-platform-secrets` in the target namespace with `mysql-user`, `mysql-password`, `redis-password`, and `access-token-secret` keys.
3. Provide reachable MySQL and Redis Services or change `APP_MYSQL_ADDR` and `APP_REDIS_ADDR`.
4. Apply `server/database/schema/current.sql` through the deployment system before rolling out a version that requires it.
5. Add the environment-specific ingress, TLS, network policies, backup policy, and monitoring stack.

The committed manifest deliberately does not contain a Secret object or plaintext production credentials.

## Probes and metrics

- Liveness: `GET /health/live`
- Readiness: `GET /health/ready`
- Metrics: `GET /metrics`

Readiness depends on MySQL and Redis. Metrics expose process, Go runtime, request-count, and request-duration series. The metrics endpoint should normally be restricted to the monitoring network rather than exposed publicly.

## Configuration overrides

The production YAML provides non-secret defaults. These environment variables override deployment-specific values:

```text
APP_HOST
APP_PORT
APP_MYSQL_ADDR
APP_MYSQL_DATABASE
APP_MYSQL_USER
APP_MYSQL_PASSWORD
APP_REDIS_ADDR
APP_REDIS_USERNAME
APP_REDIS_PASSWORD
APP_AUTH_ACCESS_TOKEN_SECRET
```

No production credential should be added to `production.yaml`, Compose files, Kubernetes manifests, Docker build arguments, source code, or CI logs.
