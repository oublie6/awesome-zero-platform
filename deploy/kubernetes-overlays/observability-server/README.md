# Goal 0037 server deployment

The host/control plane recovered after a reboot outside this execution.
All backends are now Ready, and the repository smoke plus extended API/data
checks pass. Goal 0037 is completing its browser and outage/recovery checks;
see the current goal report for the latest evidence. The previous kernel
soft lockups and shared-vCPU steal remain operational risks to monitor.

This is an explicit installation profile for the confirmed two-node
`network-learning` integration cluster. It does not change the default base.
Do not apply it to another cluster without reviewing node names and disk paths.

- `app-api`: one replica, Recreate, locally built/imported image pinned to the
  source checkpoint in `placement.yaml`; no registry credentials are required.
- Elasticsearch, MySQL, Redis and Prometheus: dedicated directories under
  `/data/awesome-zero-platform` on the named node, statically prebound local PVs,
  `Retain`. `awesome-zero-local` is intentionally not the default StorageClass.
- Kibana, Grafana and Jaeger run on the second node to distribute memory use.
- Local PV size is a binding reservation, not a filesystem quota. Monitor the
  shared disk and back up data off-node. Node failure prevents volume failover.

Before initial installation, create **new** empty directories with ownership
matching the container (ES 1000, MySQL/Redis 999, Prometheus 65534), provision the
application Secret securely (`mysql-user`, `mysql-password`,
`mysql-root-password`, `redis-password`, `access-token-secret`, and
`admin-bootstrap-token` keys), and configure the Elasticsearch node sysctl
`vm.max_map_count=1048576`. Never reuse unknown volumes. Initialize only the new
application database using the repository's complete current schema. No seed
or existing database reset is required.

Use the wrapper with this profile explicitly selected:

```bash
export OBS_KUSTOMIZE_OVERLAY="$PWD/deploy/kubernetes-overlays/observability-server"
# Set OBS_GRAFANA_ADMIN_PASSWORD from a secure source without printing it.
make observability-config
kubectl diff -k "$OBS_KUSTOMIZE_OVERLAY"
make observability-up
make observability-status
make observability-smoke
make observability-forward
```

The forward command binds only localhost. See the Goal 0037 deployment report
in `docs/goals/current.md` for evidence, credential file location (no values),
SSH forwarding, retention, operational limits and safe removal procedures.
Do not run `kubectl delete -k` on this profile or reapply the two-replica base.

Disk policy for this 40 GiB data disk: application log indexes expire after
3 days via `awesome-logs-three-days`; Prometheus retains 3 days or 512 MiB.
Elasticsearch watermarks reserve 12/8/5 GB free (low/high/flood stage), including
on the single data node. Flood stage blocks index writes to protect remaining
disk; investigate growth instead of disabling protection. MySQL binlogs expire
after 3 days and rotate at 100 MiB. Database tables and Redis AOF are durable
business data and are not automatically purged. Jaeger traces remain in memory.
These policies are not filesystem quotas; query actual ILM/TSDB settings and
monitor `df -h /data` and per-directory use. No backup destination has been
provided, so off-node backups remain an operational requirement.


The shared-vCPU server uses a **10 second HTTP request deadline** (the shipped
application default remains 3 seconds). During the controlled backend outage,
the original deadline expired during login at about 3.1 seconds; password
verification finished after that deadline. `app-api-production.yaml` is mounted
read-only through a generated ConfigMap and differs from the shipped production
configuration only in `Timeout: 10000` (milliseconds). The profile validation
checks the complete parsed configuration against the shipped file, preventing
accidental auth/security/default drift. Argon2 parameters and all application
code remain unchanged. See [go-zero's timeout field](https://github.com/zeromicro/go-zero/blob/v1.10.2/rest/config.go).

Changing this ConfigMap reference recreates the single API Pod. Schedule a
maintenance window if in-memory games are enabled. Rollback means restoring
the reviewed previous API Deployment without the `server-app-config` mount;
keep its image and one replica. Do not apply the default two-replica base.
