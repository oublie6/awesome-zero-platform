# Deploy

Deployment assets live here and must remain independent from business code.

Planned areas:

- `local/` — local development orchestration
- `docker/` — container build definitions
- `kubernetes/` — Kubernetes manifests or Helm charts when needed

Database schema definitions remain under `server/database`. One-off database upgrade SQL is treated as a runtime artifact and must not be committed.
