# Goal 0043: persistent Jaeger traces on the server overlay

This opt-in change applies **only** to `deploy/kubernetes-overlays/observability-server/`; it does not change the default Kubernetes observability overlay or the application's OTLP endpoint. It retains Jaeger v2.20.0, ClusterIP UI :16686, and OTLP gRPC/HTTP :4317/:4318. Configuration follows the tagged upstream `jaegertracing/jaeger` v2.20.0 `cmd/jaeger/config-badger.yaml` example.

Jaeger switches from volatile memory storage to embedded Badger with a **72h span TTL** and a dedicated, single-node, 2Gi requested RWO local PV/PVC. The PVC is bound to `hk-worker`. This is **not HA**, a disk quota, a guarantee of immediate disk reclaim after TTL, or an off-node backup. Old in-memory traces do **not** migrate.

## Required preflight (before any apply)

1. Confirm `kubectl config current-context` is `kubernetes-admin@network-learning`. Confirm the actual node named `hk-worker` is Ready, inspect current Jaeger Deployment and recent failures.
2. On **hk-worker**, verify real disk free space and inode availability for `/data/awesome-zero-platform/jaeger`: `df -h /data`; `df -i /data`; `findmnt -T /data`. **Do not assume** this node has the same 40Gi data disk as the control-plane node. If the path or capacity is unavailable, STOP and select/provision a reviewed dedicated data path before continuing (including updating the PV manifest).
3. Check host path does not already contain unrelated data: `sudo ls -ld /data /data/awesome-zero-platform /data/awesome-zero-platform/jaeger`. Never overwrite or delete an existing directory. If the directory is new, create it under a reviewed volume using `sudo install -d -o 10001 -g 10001 -m 0750 /data/awesome-zero-platform/jaeger`. Validate ownership and write access for UID/GID 10001. Do **not** chmod 777.
4. Verify the image actually runs as non-root UID 10001. If its user is incompatible, adjust the Deployment security context and path ownership consistently **before** rollout.
5. Inspect current PV/PVC, Secrets, app-api image/replica, all observation services, and available disk. Record hashes/old Jaeger manifest and the existing no-data Jaeger configuration for rollback. Do not reset existing volumes or modify database schema.
6. Run `OBS_KUSTOMIZE_OVERLAY="$PWD/deploy/kubernetes-overlays/observability-server" make observability-config`, `kubectl diff -k deploy/kubernetes-overlays/observability-server` and any existing manifest validation. Verify the diff affects only intended Jaeger storage/PV/PVC/ConfigMap/Deployment. The existing `observability-server` profile remains the only permitted apply target. Do not apply the generic two-replica base.
7. Confirm Jaeger 2.20.0 accepts the generated complete v2 config in an isolated disposable container or equivalent validation, with the mounted path writable. No existing Jaeger memory traces are guaranteed to survive the switch. Plan a brief trace-ingestion interruption.

## Apply and verify

Using the existing explicitly selected server profile and the existing deploy wrapper, apply only after the above checks pass, then:

1. Verify `jaeger-badger` PVC is Bound to `awesome-zero-jaeger`, the PV's node affinity matches `hk-worker`, and the Jaeger Pod is Ready with a single instance and its data volume attached.
2. Confirm OTLP 4317/4318 ingestion and UI 16686 still function; run `make observability-smoke` with `OBS_KUSTOMIZE_OVERLAY` set as above.
3. Generate a uniquely identified new HTTP request, wait until its TraceID and span count are present in Jaeger, and save the ID and query result.
4. Recreate **only** the Jaeger Pod (no PV/PVC deletion). Verify the replacement Pod reattaches the same volume, becomes Ready, and the **identical TraceID and spans** remain queryable. Verify a new trace is also ingested afterward.
5. Observe Badger disk growth and logs, memory, restart counts, possible write errors, filesystem free space, and TTL/garbage-collection behavior over time. A 72h TTL setting alone does not guarantee physical disk reclamation; monitor and investigate growth before it becomes an outage. Record any unverified retention/GC behaviors explicitly.
6. Recheck app-api, MySQL, Redis, Elasticsearch, Prometheus, Grafana, Kibana readiness. Confirm none of their PVCs or configuration were changed.

## Rollback

Rollback is a **reviewed Jaeger-only** Deployment change to its previous memory-backed configuration and original args/mounts; preserve all other deployments and storage resources. Keep the new Badger PV/PVC `Retain` and **do not delete** either resource or its host data. A rollback to memory storage stops persisting new traces; it does not convert Badger data back to memory. Do not use `kubectl delete -k` or reapply the generic base as a rollback.

## Completion gate

Complete and archive Goal 0043 only with documented exact Kubernetes manifest output, live ready status, same-TraceID-after-Pod-recreation evidence, disk/permissions checks, and CI results. Without host access or successful verification, leave the Goal in `ready` or `blocked` with explicit missing acceptance evidence.
