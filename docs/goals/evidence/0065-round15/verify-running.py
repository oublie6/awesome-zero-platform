import json,subprocess,hashlib,ssl,urllib.request
from pathlib import Path
from preflight import P,k,save
images=json.loads((P/'images.json').read_text());pods=json.loads(k('get','pods','-o','json'));out={'source':'efc0699b0d62ae1284db86121619a746ce4f37ba','pods':[]}
for kind,prefix in [('api','app-api-'),('web','game-web-')]:
 pod=next(x for x in pods['items'] if x['metadata']['name'].startswith(prefix) and x['status'].get('phase')=='Running' and all(c.get('ready') for c in x['status'].get('containerStatuses',[])))
 assert pod['spec']['containers'][0]['image']==images[kind]['image']
 container=pod['status']['containerStatuses'][0];cid=container['containerID'].split('://')[1];data=json.loads(subprocess.check_output(['crictl','inspect',cid],stderr=subprocess.DEVNULL));pid=data['info']['pid'];root=Path('/proc/'+str(pid)+'/root');row={'kind':kind,'pod':pod['metadata']['name'],'uid':pod['metadata']['uid'],'image':pod['spec']['containers'][0]['image'],'imageID':container['imageID'],'ready':container['ready'],'restarts':container['restartCount']}
 manifest=json.loads(subprocess.check_output(['ctr','-n','k8s.io','content','get',images[kind]['digest']]));assert manifest['config']['digest']==images[kind]['configID'];config=json.loads(subprocess.check_output(['ctr','-n','k8s.io','content','get',images[kind]['configID']]));assert config['config']['Labels']['org.opencontainers.image.revision']==out['source'];row['revision']=out['source']
 if kind=='api':
  row['runningBinarySHA256']=hashlib.sha256(Path('/proc/'+str(pid)+'/exe').read_bytes()).hexdigest();assert row['runningBinarySHA256']=='dc50adc3bdc3fabe04b64e0a66f417df44a1fb56fc35a2f3f225e9cdb714c000'
 else:
  row['nginxSHA256']=hashlib.sha256((root/'etc/nginx/conf.d/default.conf').read_bytes()).hexdigest();assert row['nginxSHA256']==hashlib.sha256((P/'web-context/nginx.conf').read_bytes()).hexdigest();assert (root/'etc/nginx/conf.d/00-cache.conf').read_bytes()==(P/'web-context/cache.conf').read_bytes()
 out['pods'].append(row)
ctx=ssl.create_default_context(cafile=str(P/'test-ca.crt'));files=[]
for f in sorted((P/'web-context/site').rglob('*')):
 if not f.is_file():continue
 path=str(f.relative_to(P/'web-context/site'));expected=hashlib.sha256(f.read_bytes()).hexdigest()
 with urllib.request.urlopen('https://127.0.0.1:29457/'+path,context=ctx,timeout=30) as response:actual=hashlib.sha256(response.read()).hexdigest();assert actual==expected,path
 files.append({'path':path,'sha256':expected})
out['servedFiles']=files;save('running-image-proof.json',out);print('Actual running image/revision/API binary/nginx/cache and30 served file hashes PASS')
