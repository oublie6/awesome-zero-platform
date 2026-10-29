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
