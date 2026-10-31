# Admin Web server profile

This profile is for the existing `network-learning` integration cluster. It
contains only the `awesome-zero-admin-web` Deployment and Service from the
Admin Web base. It runs one replica on `hk-worker` because the image is imported
into that node's containerd image store; it uses `Recreate` and
`imagePullPolicy: Never` and does not require a registry. The current image tag
is the complete source commit SHA `776be5e4e478a85016fb8b8caff57d94505dcab8`.

The cluster node has one filesystem-backed disk. Image layers, container
writable data, logs and node ephemeral storage all consume that node's existing
filesystem; this profile creates no data-disk path, PV, PVC, or separate data
mount. The Admin Web process serves static files and is stateless. Do not apply
the default two-replica base directly to this cluster.

Review the exact resource set and diff before applying:

```bash
kubectl kustomize deploy/kubernetes/overlays/admin-web-server
kubectl diff -k deploy/kubernetes/overlays/admin-web-server
kubectl apply -k deploy/kubernetes/overlays/admin-web-server
kubectl -n awesome-zero-platform rollout status deployment/awesome-zero-admin-web
```

Rollback removes only the two resources this profile owns. Before the first
apply, both were absent, so there is no previous image to restore:

```bash
kubectl delete -k deploy/kubernetes/overlays/admin-web-server
```

For a later update, record the prior Deployment image and manifest first. This
rollout replaced the prior image tag
`awesome-zero-platform/admin-web:30db007875f7318fdcc748366042d63e3e3ddbf0`;
its ReplicaSet and node-local image are retained. Roll back with
`kubectl -n awesome-zero-platform rollout undo deployment/awesome-zero-admin-web` while that revision is
retained, or restore the recorded image in this profile and apply it. Never
delete the namespace or apply the base manifest as a rollback.
