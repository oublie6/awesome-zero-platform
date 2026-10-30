#!/usr/bin/env python3
"""Guard deployment safety for the explicit Goal 0037 server profile."""
import pathlib
import subprocess
import sys
import tempfile

import yaml

objects = list(yaml.safe_load_all(pathlib.Path(sys.argv[1]).read_text()))
resources = {(o['kind'], o['metadata']['name']): o for o in objects}
assert len(resources) == len(objects), 'duplicate resource'
app = resources['Deployment', 'app-api']['spec']
assert app['replicas'] == 1, 'active game state requires a single process'
assert app['strategy']['type'] == 'Recreate', 'rollout must not overlap API processes'
assert not app['strategy'].get('rollingUpdate')
api = next(c for c in app['template']['spec']['containers'] if c['name'] == 'app-api')
assert api['securityContext']['runAsUser'] == 65532, 'non-numeric image USER requires an explicit UID'
# A server timeout override must not silently change auth/security or other defaults.
volume = next(v for v in app['template']['spec']['volumes'] if v['name'] == 'server-app-config')
config = yaml.safe_load(resources['ConfigMap', volume['configMap']['name']]['data']['production.yaml'])
shipped = yaml.safe_load((pathlib.Path(__file__).resolve().parents[1] / 'server/apps/app-api/etc/production.yaml').read_text())
assert config.pop('Timeout') == 10000
assert config == shipped, 'server config drifted beyond the authorized request timeout'
assert any(m['mountPath'] == '/app/etc/production.yaml' and m.get('readOnly') for m in api['volumeMounts'])
sc = resources['StorageClass', 'awesome-zero-local']
assert sc['provisioner'] == 'kubernetes.io/no-provisioner'
assert sc['reclaimPolicy'] == 'Retain'
assert sc['metadata'].get('annotations', {}).get('storageclass.kubernetes.io/is-default-class') != 'true'
paths = set()
for name in ('mysql', 'redis', 'elasticsearch', 'prometheus'):
    pv = resources['PersistentVolume', 'awesome-zero-' + name]['spec']
    assert pv['persistentVolumeReclaimPolicy'] == 'Retain'
    assert pv['local']['path'].startswith('/data/awesome-zero-platform/')
    assert pv['local']['path'] not in paths, 'volumes must not share data directories'
    paths.add(pv['local']['path'])
    assert pv['claimRef'] == {'namespace': 'awesome-zero-platform', 'name': f'data-{name}-0'}
    sts = resources['StatefulSet', name]['spec']
    claim = sts['volumeClaimTemplates'][0]['spec']
    assert claim['volumeName'] == 'awesome-zero-' + name
    assert claim['storageClassName'] == pv['storageClassName'] == sc['metadata']['name']
    terms = pv['nodeAffinity']['required']['nodeSelectorTerms']
    host = sts['template']['spec']['nodeSelector']['kubernetes.io/hostname']
    assert terms[0]['matchExpressions'][0]['values'] == [host]
prom = resources['StatefulSet', 'prometheus']['spec']['template']['spec']['containers'][0]
assert '--storage.tsdb.retention.time=3d' in prom['args']
assert '--storage.tsdb.retention.size=512MB' in prom['args']
setup = resources['Job', 'elastic-bootstrap']['spec']['template']['spec']['containers'][0]['args'][0]
assert 'awesome-logs-three-days' in setup and '"min_age":"3d"' in setup

# Goal 0043 keeps Badger opt-in to this server profile and preserves the
# existing Jaeger Service contract while binding its data to hk-worker.
jaeger = resources['Deployment', 'jaeger']['spec']
assert jaeger['replicas'] == 1
assert jaeger['strategy']['type'] == 'Recreate'
jaeger_pod = jaeger['template']['spec']
assert jaeger_pod['securityContext']['runAsNonRoot'] is True
assert jaeger_pod['securityContext']['runAsUser'] == 10001
assert jaeger_pod['securityContext']['runAsGroup'] == 10001
assert jaeger_pod['securityContext']['fsGroup'] == 10001
jaeger_container = next(c for c in jaeger_pod['containers'] if c['name'] == 'jaeger')
assert jaeger_container['image'] == 'jaegertracing/jaeger:2.20.0'
assert jaeger_container['args'] == ['--config=/etc/jaeger/config.yaml']
assert jaeger_container['securityContext']['allowPrivilegeEscalation'] is False
assert jaeger_container['securityContext']['capabilities']['drop'] == ['ALL']
assert jaeger_container['resources'] == {
    'requests': {'cpu': '50m', 'memory': '128Mi'},
    'limits': {'cpu': '1', 'memory': '512Mi'},
}
assert jaeger_container['readinessProbe']['tcpSocket']['port'] == 'ui'
jaeger_ports = {(p['name'], p['containerPort']) for p in jaeger_container['ports']}
assert jaeger_ports == {('ui', 16686), ('otlp-grpc', 4317), ('otlp-http', 4318)}
jaeger_service = resources['Service', 'jaeger']['spec']
service_ports = {(p['name'], p['port'], p['targetPort']) for p in jaeger_service['ports']}
assert service_ports == {
    ('ui', 16686, 'ui'), ('otlp-grpc', 4317, 'otlp-grpc'),
    ('otlp-http', 4318, 'otlp-http'),
}
jaeger_pv = resources['PersistentVolume', 'awesome-zero-jaeger']['spec']
assert jaeger_pv['persistentVolumeReclaimPolicy'] == 'Retain'
assert jaeger_pv['capacity']['storage'] == '2Gi'
assert jaeger_pv['accessModes'] == ['ReadWriteOnce']
assert jaeger_pv['local']['path'] == '/data/awesome-zero-platform/jaeger'
assert jaeger_pv['claimRef'] == {
    'namespace': 'awesome-zero-platform', 'name': 'jaeger-badger'
}
assert jaeger_pv['nodeAffinity']['required']['nodeSelectorTerms'][0]['matchExpressions'][0] == {
    'key': 'kubernetes.io/hostname', 'operator': 'In', 'values': ['hk-worker']
}
jaeger_pvc = resources['PersistentVolumeClaim', 'jaeger-badger']['spec']
assert jaeger_pvc['volumeName'] == 'awesome-zero-jaeger'
assert jaeger_pvc['storageClassName'] == 'awesome-zero-local'
assert jaeger_pvc['resources']['requests']['storage'] == '2Gi'
config_volume = next(v for v in jaeger_pod['volumes'] if v['name'] == 'jaeger-config')
config_map = resources['ConfigMap', config_volume['configMap']['name']]['data']['config.yaml']
config = yaml.safe_load(config_map)
assert config['service']['extensions'] == ['jaeger_storage', 'jaeger_query', 'healthcheckv2']
assert config['service']['pipelines']['traces'] == {
    'receivers': ['otlp'], 'processors': ['batch'],
    'exporters': ['jaeger_storage_exporter'],
}
assert config['service']['telemetry']['logs']['level'] == 'info'
assert config['extensions']['healthcheckv2']['use_v2'] is True
assert config['extensions']['jaeger_query']['storage']['traces'] == 'primary_store'
badger = config['extensions']['jaeger_storage']['backends']['primary_store']['badger']
assert badger['directories'] == {'keys': '/badger/keys', 'values': '/badger/values'}
assert badger['ephemeral'] is False
assert badger['ttl']['spans'] == '72h'
assert config['exporters']['jaeger_storage_exporter']['trace_storage'] == 'primary_store'
assert config['processors']['batch'] == {}
assert config['receivers']['otlp']['protocols']['grpc']['endpoint'] == '0.0.0.0:4317'
assert config['receivers']['otlp']['protocols']['http']['endpoint'] == '0.0.0.0:4318'
mounts = {m['name']: m for m in jaeger_container['volumeMounts']}
assert mounts['jaeger-config']['mountPath'] == '/etc/jaeger/config.yaml'
assert mounts['jaeger-config']['readOnly'] is True
assert mounts['jaeger-data']['mountPath'] == '/badger'
data_volume = next(v for v in jaeger_pod['volumes'] if v['name'] == 'jaeger-data')
assert data_volume['persistentVolumeClaim']['claimName'] == 'jaeger-badger'
for o in objects:
    assert o['kind'] not in ('Secret', 'Ingress', 'ClusterRole', 'ClusterRoleBinding')
    if o['kind'] == 'Service':
        assert o['spec'].get('type', 'ClusterIP') == 'ClusterIP'
# Reuse the original stack contract checks after excluding new cluster resources.
with tempfile.NamedTemporaryFile(mode='w', suffix='.yaml') as f:
    yaml.safe_dump_all([o for o in objects if o['kind'] not in ('PersistentVolume', 'StorageClass')], f)
    f.flush()
    subprocess.run([sys.executable, str(pathlib.Path(__file__).with_name('test-k8s-observability-render.py')), f.name], check=True)
print('Server profile safety checks passed')
