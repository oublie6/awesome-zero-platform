import subprocess,json,time
from pathlib import Path
p=Path(__file__).parent;r=[]
for n in ['bootstrap-immediate.cjs','bootstrap-adjacent.cjs','smoke.cjs','vue-cleanup.cjs','startup-failure.cjs','cache-smoke.cjs']:
 start=time.time()
 with (p/(n.replace('.cjs','.log'))).open('w') as f:c=subprocess.run(['/opt/goal0042-toolchain/node-v22.23.4-linux-x64/bin/node',str(p/n)],stdout=f,stderr=subprocess.STDOUT,timeout=180)
 r.append({'name':n,'exit':c.returncode,'seconds':round(time.time()-start,3)});(p/'browser-checks.json').write_text(json.dumps(r,indent=2));print(n,c.returncode,flush=True)
