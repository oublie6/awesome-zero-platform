#!/usr/bin/env python3
"""Validate rendered Kustomize resources without a live Kubernetes cluster."""
import json
import pathlib
import sys

import yaml


def main() -> None:
    path = pathlib.Path(sys.argv[1])
    objects = [item for item in yaml.safe_load_all(path.read_text()) if item]
    assert objects, "empty rendered Kustomize output"
    resources = {}
    for obj in objects:
        meta = obj["metadata"]
        key = (obj["kind"], meta["name"])
        assert key not in resources, f"duplicate Kubernetes resource: {key}"
        assert obj["kind"] == "Namespace" or meta.get("namespace") == "awesome-zero-platform", key
        resources[key] = obj

    def get(kind, name):
        return resources[(kind, name)]

    for obj in objects:
        assert obj["kind"] not in {
            "Ingress", "Gateway", "ClusterRole", "ClusterRoleBinding"
        }, "developer observability cannot install a public ingress or cluster RBAC"
        if obj["kind"] == "Service":
            assert obj["spec"].get("type", "ClusterIP") == "ClusterIP", obj["metadata"]["name"]

    app = get("Deployment", "app-api")
    spec = app["spec"]["template"]["spec"]
    containers = {c["name"]: c for c in spec["containers"]}
    assert "app-api" in containers and "filebeat" in containers
    env = {e["name"]: e for e in containers["app-api"]["env"]}
    assert env["APP_LOG_MODE"]["value"] == "file"
    assert env["APP_LOG_PATH"]["value"] == "/app/logs"
    assert env["APP_TELEMETRY_ENDPOINT"]["value"] == "jaeger:4317"
    assert containers["app-api"]["securityContext"]["readOnlyRootFilesystem"]
    filebeat = containers["filebeat"]
    assert filebeat["securityContext"]["runAsNonRoot"] is True
    assert filebeat["securityContext"]["readOnlyRootFilesystem"] is True
    assert "--strict.perms=false" in filebeat["command"]
    assert {m["name"] for m in spec["volumes"]}.issuperset(
        {"app-logs", "filebeat-config", "filebeat-state", "tmp"}
    )
    assert spec["securityContext"]["fsGroup"] == 65532

    for name in ("elasticsearch", "prometheus"):
        sts = get("StatefulSet", name)
        assert sts["spec"]["volumeClaimTemplates"], f"{name} needs a PVC template"
        assert get("Service", name)["spec"]["clusterIP"] == "None", f"{name} StatefulSet requires a Headless Service"
    for name in ("grafana", "kibana", "jaeger", "elasticsearch", "prometheus"):
        get("Service", name)
    for name in ("grafana", "kibana", "jaeger"):
        get("Deployment", name)
    get("Job", "elastic-bootstrap")

    prometheus = get("StatefulSet", "prometheus")
    assert prometheus["spec"]["template"]["spec"]["serviceAccountName"] == "observability-prometheus"
    role = get("Role", "observability-prometheus-pods")
    assert role["rules"] == [
        {"apiGroups": [""], "resources": ["pods"], "verbs": ["get", "list", "watch"]}
    ]
    get("RoleBinding", "observability-prometheus-pods")
    get("ServiceAccount", "observability-prometheus")

    cm = get("ConfigMap", "obs-prometheus")
    prometheus_config = yaml.safe_load(cm["data"]["prometheus.yml"])
    jobs = prometheus_config["scrape_configs"]
    assert len(jobs) == 1 and jobs[0]["job_name"] == "app-api"
    assert jobs[0]["kubernetes_sd_configs"][0]["role"] == "pod"
    assert jobs[0]["kubernetes_sd_configs"][0]["namespaces"]["names"] == ["awesome-zero-platform"]
    assert any(p.get("regex") == "8888" for p in jobs[0]["relabel_configs"])
    json.loads(get("ConfigMap", "obs-grafana-dashboard")["data"]["platform-overview.json"])
    assert "filestream" in get("ConfigMap", "obs-filebeat")["data"]["filebeat.yml"]
    grafana = get("Deployment", "grafana")["spec"]["template"]["spec"]["containers"][0]
    password = next(e for e in grafana["env"] if e["name"] == "GF_SECURITY_ADMIN_PASSWORD")
    assert password["valueFrom"]["secretKeyRef"]["name"] == "observability-secrets"
    assert "value" not in password

    print(f"Kubernetes observability manifest validated: {len(objects)} resources")


if __name__ == "__main__":
    main()
