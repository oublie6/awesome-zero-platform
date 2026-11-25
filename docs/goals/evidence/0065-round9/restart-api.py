from pathlib import Path
import json,os,signal,time,subprocess,urllib.request,hashlib
p=Path(__file__).parent;old=json.loads((p/'api-process.json').read_text());pid=old['pid'];assert Path('/proc/'+str(pid)+'/exe').resolve()==p/'app-api';assert hashlib.sha256((p/'app-api').read_bytes()).hexdigest()==old['binarySHA256'];os.kill(pid,signal.SIGTERM)
for _ in range(100):
 try:
  if Path('/proc/'+str(pid)+'/stat').read_text().split()[2]=='Z':break
 except FileNotFoundError:break
 time.sleep(.1)
else:raise RuntimeError('owned API shutdown incomplete; do not overlap')
subprocess.run(['python3',str(p/'start-api.py')],check=True)
for _ in range(100):
 try:
  with urllib.request.urlopen('http://127.0.0.1:18869/health/ready',timeout=1) as r:
   if r.status==200:break
 except Exception:pass
 time.sleep(.1)
else:raise RuntimeError('candidate API restart not Ready')
(p/'api-restart.json').write_text(json.dumps({'previousPid':pid,'newPid':json.loads((p/'api-process.json').read_text())['pid'],'sameBinary':True,'ready':True}));print('Same candidate restarted and Ready')
