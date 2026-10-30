# Goal 0037 server deployment

The stack has passed real log/metric/trace, browser Discover, authentication,
WebSocket, Grafana panel and controlled backend-outage/recovery checks. The
confirmed host recovered after a reboot outside this execution. Shared-vCPU
steal and the earlier kernel soft lockups remain operational risks to monitor.
See the Goal 0037 report for point-in-time evidence and deployment limits.

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

For browser access from your own computer, follow the [security group and SSH tunnel steps](../../../docs/operations/logging-observability.md#cloud-server-access-through-an-ssh-tunnel). Only your existing SSH port needs a public access rule restricted to your IP.

The forward command binds only localhost. See the [Goal 0037 deployment report](../../../docs/goals/archive/0037-kubernetes-observability-deployment.md)
for evidence, credential file location (no values),
SSH forwarding, retention, operational limits and safe removal procedures.
Do not run `kubectl delete -k` on this profile or reapply the two-replica base.

Disk policy for this 40 GiB data disk: application log indexes expire after
3 days via `awesome-logs-three-days`; Prometheus retains 3 days or 512 MiB.
Elasticsearch watermarks reserve 12/8/5 GB free (low/high/flood stage), including
on the single data node. Flood stage blocks index writes to protect remaining
disk; investigate growth instead of disabling protection. MySQL binlogs expire
after 3 days and rotate at 100 MiB. Database tables and Redis AOF are durable
business data and are not automatically purged. These policies are not
filesystem quotas; query actual ILM/TSDB settings and monitor `df -h /data` and
per-directory use. No backup destination has been provided, so off-node backups
remain an operational requirement.

The default observability overlay keeps Jaeger traces in memory. This server
profile uses Badger on `/data/awesome-zero-platform/jaeger`, backed by the
2 GiB-requested Retain local PV on `hk-worker`, with a configured 72-hour span
TTL. The PV request is not a filesystem quota. See the [Goal 0043 rollout and
recovery procedure](../../../docs/operations/goal0043-jaeger-badger-persistence.md)
and its [completed acceptance report](../../../docs/goals/archive/0043-jaeger-badger-persistence.md).


The shared-vCPU server uses a **10 second HTTP request deadline** (the shipped
application default remains 3 seconds). During the controlled backend outage,
the original deadline expired during login at about 3.1 seconds, followed by
a session-creation context deadline error. `app-api-production.yaml` is mounted
read-only through a generated ConfigMap and differs from the shipped production
configuration only in `Timeout: 10000` (milliseconds). The profile validation
checks the complete parsed configuration against the shipped file, preventing
accidental auth/security/default drift. Argon2 parameters and all application
code remain unchanged. See [go-zero's timeout field](https://github.com/zeromicro/go-zero/blob/v1.10.2/rest/config.go).

Changing this ConfigMap reference recreates the single API Pod. Schedule a
maintenance window if in-memory games are enabled. Rollback means restoring
the reviewed previous API Deployment without the `server-app-config` mount;
keep its image and one replica. Do not apply the default two-replica base.
