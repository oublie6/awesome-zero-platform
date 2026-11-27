from pathlib import Path
import subprocess,json,os,signal,time,urllib.request
p=Path(__file__).parent;images=json.loads((p/'images.json').read_text());state=json.loads((p/'api-process.json').read_text());pid=state['pid']
try:
 assert str(p/'app-api').encode() in Path('/proc/'+str(pid)+'/cmdline').read_bytes();os.kill(pid,signal.SIGTERM)
except FileNotFoundError:pass
for _ in range(50):
 if not Path('/proc/'+str(pid)).exists():break
 time.sleep(.1)
env=os.environ.copy();private=json.loads((p/'clone-runtime-env.json').read_text());env.update(private);cfg=p/'clone-container.yaml';cfg.write_bytes((p/'clone-app.yaml').read_bytes());cfg.chmod(0o444)
args=['docker','run','-d','--name','codex0065-r15-api-image-smoke','--network','host','--entrypoint','/app/app-api','-v',str(cfg)+':/app/clone-app.yaml:ro']
for key in private:args+=['--env',key]
args +=[images['api']['tag'],'-f','/app/clone-app.yaml'];subprocess.run(args,env=env,check=True,stdout=subprocess.DEVNULL)
for _ in range(40):
 try:
  with urllib.request.urlopen('http://127.0.0.1:18870/health/ready',timeout=2) as r:
   if r.status==200:break
 except Exception:time.sleep(1)
else:raise RuntimeError('Nonroot candidate API image did not become ready')
info=json.loads(subprocess.check_output(['docker','inspect','codex0065-r15-api-image-smoke']))[0];assert info['Config']['User']=='nonroot:nonroot';assert info['Image']==images['api']['configID'];pid=info['State']['Pid'];state['pid']=pid;state['container']='codex0065-r15-api-image-smoke';(p/'api-process.json').write_text(json.dumps(state));(p/'container-api-smoke.json').write_text(json.dumps({'image':images['api']['image'],'readyHTTP':200,'nonroot':True,'binarySHA256':__import__('hashlib').sha256(Path('/proc/'+str(pid)+'/exe').read_bytes()).hexdigest(),'sameIsolatedDatabaseConfigAndDependencies':True},indent=2));print('Candidate API image nonroot isolated readiness PASS')
