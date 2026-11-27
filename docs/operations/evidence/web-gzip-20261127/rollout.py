from pathlib import Path
import json,subprocess,time,ssl,urllib.request
from preflight import P,k,sql,save
j=json.loads((P/'image-input.json').read_text());assert json.loads((P/'verify-30473.json').read_text())['result']=='PASS';before=json.loads((P/'deployments-before.json').read_text());old=next(x['spec']['template']['spec']['containers'][0]['image'] for x in before['items'] if x['metadata']['name']=='game-web');assert old==j['baseManifest'];assert int(sql("SELECT COUNT(*) FROM goal0057.doudizhu_hands WHERE phase NOT IN ('COMPLETED','ABORTED','CANCELLED','EXPIRED');"))==0
archive=P/'web.tar';subprocess.run(['docker','save','-o',str(archive),j['tag']],check=True)
with (P/'import.log').open('w') as f:subprocess.run(['ctr','-n','k8s.io','images','import',str(archive)],stdout=f,stderr=subprocess.STDOUT,check=True)
rows=subprocess.check_output(['ctr','-n','k8s.io','images','ls'],text=True);digest=next(line.split()[2] for line in rows.splitlines() if line.split()[0]==j['tag']);image=j['tag'].split(':')[0]+'@'+digest;subprocess.run(['ctr','-n','k8s.io','images','tag',j['tag'],image],stdout=subprocess.DEVNULL,check=True);archive.unlink();j.update(image=image,digest=digest);save('image.json',j)
try:
 save('set-image.log',k('set','image','deployment/game-web','game-web='+image));save('rollout.log',k('rollout','status','deployment/game-web','--timeout=120s'))
 ctx=ssl.create_default_context(cafile='/root/goal0065-round15-efc0699/test-ca.crt')
 for attempt in range(30):
  try:
   with urllib.request.urlopen('https://127.0.0.1:29457/health/ready',context=ctx,timeout=3) as r:
    if r.status==200 and json.load(r)['status']=='ready':break
  except Exception:pass
  time.sleep(1)
 else:raise RuntimeError('User forwarding/API proxy did not become ready')
 subprocess.run(['python3',str(P/'verify.py'),'29457'],check=True,stdout=(P/'verify-live.log').open('w'))
 save('rollout-result.json',{'result':'PASS','oldImage':old,'newImage':image,'apiChanged':False,'unfinishedHandsBefore':0,'rollbackExecuted':False});print('Web-only rollout and live compression/auth/cache checks PASS',flush=True)
except Exception:
 save('rollback-set.log',k('set','image','deployment/game-web','game-web='+old));save('rollback-status.log',k('rollout','status','deployment/game-web','--timeout=120s'));save('rollout-result.json',{'result':'FAIL','restored':old});raise
