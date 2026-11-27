from pathlib import Path
import subprocess,json,time
p=Path(__file__).parent;out=[]
for name in ['smoke.cjs','vue-lifecycle.cjs','vue-cleanup.cjs','cache-smoke.cjs','browser-storage-fault.cjs']:
 start=time.time()
 with (p/name.replace('.cjs','.log')).open('w') as f:r=subprocess.run(['/opt/goal0042-toolchain/node-v22.23.4-linux-x64/bin/node',str(p/name)],stdout=f,stderr=subprocess.STDOUT,timeout=180)
 out.append({'name':name,'exit':r.returncode,'seconds':round(time.time()-start,3)});(p/'browser-checks.json').write_text(json.dumps(out,indent=2));print(name,r.returncode,flush=True)
