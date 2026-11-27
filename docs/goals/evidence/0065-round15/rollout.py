import subprocess,json,time,ssl,urllib.request,hashlib,sys
from pathlib import Path
from preflight import P,k,sql,save
from importlib.machinery import SourceFileLoader
source='efc0699b0d62ae1284db86121619a746ce4f37ba';gate=json.loads((P/'deploy-gate.json').read_text());images=json.loads((P/'images.json').read_text());old=gate['rollbackImages'];events=[];changed=False
ctx=ssl.create_default_context(cafile=str(P/'test-ca.crt'))
def ready():
 for attempt in range(30):
  try:
   with urllib.request.urlopen('https://127.0.0.1:29457/health/ready',context=ctx,timeout=3) as r:
    if r.status==200 and json.load(r).get('status')=='ready':return
  except Exception:pass
  time.sleep(1)
 raise RuntimeError('User29457 health did not recover')
def record(event):events.append({'event':event,'utc':time.strftime('%Y-%m-%dT%H:%M:%SZ',time.gmtime())});save('rollout-events.json',events);print(event,flush=True)
def image(kind):
 resource='app-api' if kind=='api' else 'game-web';return resource,resource+'='+images[kind]['image']
try:
 assert gate['sourceRuntime']=='PASS'
 assert int(sql("SELECT COUNT(*) FROM goal0057.doudizhu_hands WHERE phase NOT IN ('COMPLETED','ABORTED','CANCELLED','EXPIRED');"))==0
 assert int(sql('SELECT COUNT(*) FROM goal0057.doudizhu_turn_deadlines;'))==0
 record('no unfinished hands/deadlines; API cutover begins')
 resource,arg=image('api');changed=True;save('api-set-image.log',k('set','image','deployment/'+resource,arg));save('api-rollout.log',k('rollout','status','deployment/'+resource,'--timeout=120s'));ready();record('API rollout ready before Web update')
 session=SourceFileLoader('cluster_session',str(P/'cluster-session.py')).load_module();token=json.loads((P/'cluster-session-private.json').read_text())['token'];status,j=session.req('/auth/session',token);assert status==200;status,j=session.req('/auth/logout',token,body={});assert status==200 and j['data']['loggedOut'] is True;save('cluster-session-after-api.json',{'preservedSessionHTTP':200,'explicitLogoutConfirmed':True})
 subprocess.run(['python3',str(P/'create-cluster-qa.py')],check=True)
 resource,arg=image('web');save('web-set-image.log',k('set','image','deployment/'+resource,arg));save('web-rollout.log',k('rollout','status','deployment/'+resource,'--timeout=120s'));ready();record('Web rollout ready; user forwarding healthy')
 for name in ['smoke.cjs','cache-smoke.cjs']:
  with (P/'deployed'/name.replace('.cjs','.log')).open('w') as f:subprocess.run(['/opt/goal0042-toolchain/node-v22.23.4-linux-x64/bin/node',str(P/'deployed'/name)],stdout=f,stderr=subprocess.STDOUT,check=True,timeout=160)
  assert json.loads((P/'deployed'/name.replace('.cjs','.json')).read_text())['result']=='PASS',name
 record('Deployed login/load/create/reconnect/leave/logout/switch/cache PASS')
 subprocess.run(['python3',str(P/'verify-running.py')],check=True)
 save('rollout-result.json',{'result':'PASS','source':source,'images':images,'order':['api','web'],'postDeploymentRuntime':'PASS','rollbackExecuted':True,'earlierAttemptRolledBack':True,'finalAttemptRollbackExecuted':False});record('Running image/binary/site identity PASS')
except Exception as e:
 save('rollout-result.json',{'result':'FAIL','source':source,'failureType':type(e).__name__,'failure':str(e),'rollbackAttempted':changed});record('Deployment verification failed; inverse rollback begins')
 if changed:
  for resource in ['game-web','app-api']:
   save(resource+'-rollback-set.log',k('set','image','deployment/'+resource,resource+'='+old[resource]));save(resource+'-rollback-status.log',k('rollout','status','deployment/'+resource,'--timeout=120s'))
  ready();save('rollback-result.json',{'result':'PASS','restored':old,'order':['game-web','app-api']})
 raise
