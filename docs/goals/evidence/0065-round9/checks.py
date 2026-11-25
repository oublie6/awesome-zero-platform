import subprocess,os,time,json
from pathlib import Path
p=Path(__file__).parent
os.environ['PATH']='/opt/goal0042-toolchain/node-v22.23.4-linux-x64/bin:'+os.environ['PATH']
os.environ['COCOS_CLI_ROOT']='/data/awesome-zero-platform-tools/round7-cocos-cli'
r=[]
for name,args,cwd in [('core-types',['npm','run','typecheck'],p/'input/clients/packages/game-client-core'),('core-tests',['npm','test'],p/'input/clients/packages/game-client-core'),('protocol-types',['npm','run','typecheck'],p/'input/clients/packages/doudizhu-client'),('protocol-tests',['npm','test'],p/'input/clients/packages/doudizhu-client'),('cocos-static-types',['npm','run','typecheck'],p/'input/clients/fair-doudizhu-cocos'),('vue-build',['npm','run','build'],p/'input/clients/game-web'),('vue-tests',['npm','test'],p/'input/clients/game-web'),('script-tests',['node','--test','scripts/cocos-linux-web/test.cjs','scripts/game-web/stage.test.cjs'],p/'input'),('cocos-tests',['npm','test'],p/'input/clients/fair-doudizhu-cocos')]:
 start=time.time()
 with (p/(name+'.log')).open('w') as f: result=subprocess.run(args,cwd=cwd,stdout=f,stderr=subprocess.STDOUT)
 r.append({'name':name,'command':args,'exit':result.returncode,'seconds':round(time.time()-start,3)})
 print(name,result.returncode,flush=True)
 (p/'checks.json').write_text(json.dumps(r,indent=2))
 if result.returncode:raise SystemExit(result.returncode)
