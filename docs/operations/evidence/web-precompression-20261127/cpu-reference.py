from pathlib import Path
import subprocess,json,ssl,urllib.request,time
p=Path(__file__).parent;phase=__import__('sys').argv[1]
pods=json.loads(subprocess.check_output(['kubectl','--context','kubernetes-admin@network-learning','-n','awesome-zero-game-test','get','pods','-o','json']));pod=next(x for x in pods['items'] if x['metadata']['name'].startswith('game-web-') and all(c.get('ready') for c in x['status'].get('containerStatuses',[])));cid=pod['status']['containerStatuses'][0]['containerID'].split('://')[1];pid=json.loads(subprocess.check_output(['crictl','inspect',cid],stderr=subprocess.DEVNULL))['info']['pid'];group=Path('/proc/'+str(pid)+'/cgroup').read_text().strip().split('::')[1];stat=Path('/sys/fs/cgroup'+group+'/cpu.stat')
def stats():return {k:int(v) for k,v in [line.split() for line in stat.read_text().splitlines()]}
ctx=ssl.create_default_context(cafile='/root/goal0065-round15-efc0699/test-ca.crt');start=stats();rows=[]
for i in range(3):
 t=time.monotonic();q=urllib.request.Request('https://127.0.0.1:29457/game/cocos-js/cc.js',headers={'Accept-Encoding':'gzip'})
 with urllib.request.urlopen(q,context=ctx,timeout=30) as r:body=r.read();rows.append({'bytes':len(body),'seconds':time.monotonic()-t,'encoding':r.headers.get('Content-Encoding'),'length':r.headers.get('Content-Length'),'etag':r.headers.get('ETag'),'mime':r.headers.get('Content-Type'),'vary':r.headers.get('Vary'),'cache':r.headers.get('Cache-Control')})
end=stats();out={'phase':phase,'pod':pod['metadata']['name'],'image':pod['spec']['containers'][0]['image'],'samples':rows,'nginxCPUSeconds':(end['usage_usec']-start['usage_usec'])/1e6,'throttledPeriods':end.get('nr_throttled',0)-start.get('nr_throttled',0),'scope':'Container cgroup includes all Nginx workers during three sequential cc gzip HTTP requests; shared test host background traffic may affect counts'};(p/('cpu-'+phase+'.json')).write_text(json.dumps(out,indent=2));print(json.dumps(out))
