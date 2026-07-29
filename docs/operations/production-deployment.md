# Production Deployment Baseline

The repository provides deployment baselines rather than prescribing a cloud, managed database, ingress controller, certificate issuer, backup product, or secret manager.

## Required platform secrets

Every production deployment requires values supplied outside Git:

```text
APP_MYSQL_PASSWORD
APP_REDIS_PASSWORD
APP_AUTH_ACCESS_TOKEN_SECRET
```

The bundled production Compose MySQL additionally requires:

```text
APP_MYSQL_ROOT_PASSWORD
```

`APP_ADMIN_BOOTSTRAP_TOKEN` is required only while creating the first administrator. It must contain at least 32 characters. Remove it and recreate the API containers after bootstrap succeeds.

The access-token secret must contain at least 32 characters and should be generated from cryptographically secure random bytes. Rotating it invalidates existing signed access tokens.

Never put production credentials in committed YAML, Compose files, Kubernetes manifests, Docker build arguments, image layers, source code, CI logs, or ordinary configuration diagnostics.

## Optional Fair Doudizhu enablement

Fair Doudizhu is disabled by default. A production environment that enables it must provide all of the following:

```text
APP_REVEAL_KEYS_ENABLED=true
APP_REVEAL_KEYS_STATIC_JSON=<strict reveal-key registry JSON>
APP_DOUDIZHU_ENABLED=true
APP_DOUDIZHU_BEACON_PROVIDER=<provider identifier>
APP_DOUDIZHU_BEACON_ROUND=<configured round identifier>
APP_DOUDIZHU_BEACON_PROOF_SECRET=<at least 32 characters>
APP_DOUDIZHU_CONTRIBUTION_KEY_ID=<key identifier>
APP_DOUDIZHU_CONTRIBUTION_KEY_HEX=<exactly 64 lowercase hex characters>
```

`APP_REVEAL_KEYS_STATIC_JSON` contains an Ed25519 manifest-signing private key and one or more X25519 reveal private keys. Treat the whole value as high-sensitivity private-key material. The beacon-proof secret and contribution key are also secrets.

The production Compose baseline passes these variables to `app-api`, but leaves reveal keys and Doudizhu disabled unless the operator supplies them. The local `revealkeys/cmd/local-env` generator is development-only and must not be used as a production key-management process.

The provider and round are locked into hand metadata. Changing them does not rewrite or migrate existing hands. Key rotation, retiring-key grace periods, retention, backup, and emergency revocation must be handled as an explicit operational procedure.

## Production Compose

The supported entrypoint is:

```text
scripts/production-compose.sh
```

The Makefile delegates to it:

```bash
make production-config
make production-up
make production-down
```

The wrapper always loads `deploy/production/docker-compose.yml`. It conditionally adds `deploy/production/docker-compose.tls.yml` according to `APP_HTTPS_ENABLED`.

Accepted enabled values are `true`, `1`, `yes`, and `on`. Accepted disabled values are `false`, `0`, `no`, and `off`. Matching is case-insensitive. The default is `false`, and invalid values fail before Docker Compose starts.

Example loopback-only HTTP/WS deployment:

```bash
export APP_MYSQL_PASSWORD='replace-me'
export APP_MYSQL_ROOT_PASSWORD='replace-me'
export APP_REDIS_PASSWORD='replace-me'
export APP_AUTH_ACCESS_TOKEN_SECRET='replace-with-a-long-random-secret'
export APP_ADMIN_BOOTSTRAP_TOKEN='temporary-bootstrap-token-at-least-32-characters'
export APP_HTTPS_ENABLED=false

make production-config
make production-up
```

An external environment file may be used:

```bash
APP_HTTPS_ENABLED=false \
  bash scripts/production-compose.sh \
  --env-file /absolute/path/to/production.env \
  up -d --build --wait
```

`APP_HTTPS_ENABLED` controls which Compose files the wrapper selects, so set it in the shell before invoking the script. Other values may come from `--env-file`.

The baseline:

- uses pinned MySQL and Redis images;
- starts persistent MySQL and Redis services;
- waits for dependency health checks;
- applies the complete current schema through a one-shot schema container;
- starts a non-root, read-only API container;
- starts a non-root, read-only Admin Web container;
- enables authentication, authorization synchronization, realtime `/ws`, and metrics;
- proxies API and WebSocket requests through Admin Web Nginx;
- binds base plaintext ports to loopback only.

This is a low-memory single-host baseline, not a recommendation to run bundled databases instead of managed production services.

## Compose project name and data continuity

The wrapper defaults the project name to:

```text
production
```

Default volumes are therefore:

```text
production_mysql-data
production_redis-data
```

Keeping the same project name while enabling or disabling HTTPS preserves the existing data. The HTTPS switch changes only whether the TLS edge is included.

Use an alternate project name only for an intentionally isolated stack:

```bash
APP_COMPOSE_PROJECT_NAME=isolated-test \
APP_HTTPS_ENABLED=true \
make production-up
```

That creates different volumes; it does not copy data from the default production stack.

Before changing project names or volume policy on a host with data:

```bash
docker compose ls
docker volume ls | grep -E '(^|_)mysql-data$|(^|_)redis-data$'
```

Do not delete, rename, or replace production volumes without a verified backup and an explicit migration plan.

MySQL data directories are not downgrade-compatible across major versions. Never remove a volume merely to make an image start unless its data is backed up or intentionally disposable.

## HTTPS and WSS

Local development may use HTTP and WS. Public production traffic must use HTTPS and WSS through a TLS-terminating edge, ingress, gateway, or load balancer.

The optional single-host edge requires paths supplied outside Git:

```text
APP_TLS_CERT_FILE
APP_TLS_KEY_FILE
```

Enable it with:

```bash
export APP_HTTPS_ENABLED=true
export APP_TLS_CERT_FILE='/absolute/path/to/fullchain.pem'
export APP_TLS_KEY_FILE='/absolute/path/to/privkey.pem'
export APP_HTTP_PORT=80
export APP_HTTPS_PORT=443

make production-config
make production-up
```

The edge:

- redirects HTTP to the configured HTTPS port;
- accepts TLS 1.2 and TLS 1.3;
- adds HSTS;
- proxies Admin Web and API traffic;
- preserves WebSocket Upgrade headers for `/ws`.

The certificate SAN must contain the deployed domain, and the full chain must be supplied. The Compose edge consumes existing certificate files; it does not request or renew certificates. Prefer a managed certificate service, ingress controller, or ACME process.

Never commit a TLS private key. The key must be readable by the unprivileged edge container through a controlled permission or secret-mount mechanism.

## Authentication bootstrap

After the stack becomes healthy:

1. open the Admin Web bootstrap route;
2. create the first `platform_super_admin` account using `APP_ADMIN_BOOTSTRAP_TOKEN`;
3. verify administrator login and `/admin/me`;
4. remove `APP_ADMIN_BOOTSTRAP_TOKEN` from the deployment environment;
5. recreate the API container so bootstrap is no longer configured.

Authentication does not require sticky sessions because sessions are stored in Redis.

## Probes and metrics

- Liveness: `GET /health/live`
- Readiness: `GET /health/ready`
- Metrics: `GET /metrics`
- Authenticated realtime probe: `app-api -realtime-healthcheck`

Readiness depends on MySQL, Redis, and current authorization policy synchronization. Metrics expose process, Go runtime, HTTP request, and realtime WebSocket series. Restrict `/metrics` to the monitoring network.

The runtime acceptance script verifies production HTTP, HTTPS, authenticated WS/WSS, bootstrap/login, and cleanup behavior.

## Cache and authorization behavior

Ordinary entity detail reads use the shared Redis model cache. Transactions, `FOR UPDATE` reads, authentication account-state checks, lists, searches, password data, sessions, audit events, Doudizhu aggregates, and Casbin policy rows bypass ordinary row caching.

MySQL remains the durable Casbin policy source. Each API process keeps a synchronized in-memory enforcer. Policy writes commit a version in MySQL, reload the local enforcer, and publish the version through Redis Pub/Sub. Periodic MySQL polling repairs missed notifications. A failed or stale reload makes readiness fail while the previous in-memory policy remains intact.

## Fair Doudizhu process boundary

Active cards, bids, turns, passes, and live versions are authoritative in one API process. MySQL stores durable room/fairness data, command results, events, protected contribution references, and immutable terminal archives. Redis does not store active game snapshots.

Consequences:

- participant reconnect is supported while the authoritative process is alive;
- terminal evidence remains available after normal live-hand removal;
- a process crash does not restore an active hand;
- multiple interchangeable API replicas do not provide transparent active-game migration or cross-Pod delivery.

A multi-instance game deployment must add explicit authoritative room ownership, cross-node command/event routing, and recoverable active state. Do not assume the generic local WebSocket hub provides distributed game routing.

## Kubernetes baseline

`deploy/kubernetes/base.yaml` and `deploy/kubernetes/admin-web.yaml` provide namespace, API/Admin Deployments, Services, probes, resource requests/limits, security contexts, Prometheus annotations, and an API PodDisruptionBudget.

Before applying them:

1. replace example image references with immutable tags or digests;
2. create the required Secret outside Git;
3. provide reachable MySQL and Redis endpoints;
4. apply `server/database/schema/current.sql` through a deployment Job or CI/CD database stage;
5. configure ingress, TLS, WebSocket timeouts, network policies, backups, logs, and monitoring;
6. keep Doudizhu disabled across interchangeable replicas unless the later distributed-game boundary has been implemented.

API Pods must not modify schema during startup.

## Environment overrides

Supported production overrides include:

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
APP_REVEAL_KEYS_ENABLED
APP_REVEAL_KEYS_STATIC_JSON
APP_DOUDIZHU_ENABLED
APP_DOUDIZHU_BEACON_PROVIDER
APP_DOUDIZHU_BEACON_ROUND
APP_DOUDIZHU_BEACON_PROOF_SECRET
APP_DOUDIZHU_CONTRIBUTION_KEY_ID
APP_DOUDIZHU_CONTRIBUTION_KEY_HEX
APP_COMPOSE_PROJECT_NAME
APP_HTTPS_ENABLED
APP_TLS_CERT_FILE
APP_TLS_KEY_FILE
APP_HTTP_PORT
APP_HTTPS_PORT
```

See `docs/operations/server-v1-release.md` for the local three-player integration path and exact server-v1 feature boundary.
