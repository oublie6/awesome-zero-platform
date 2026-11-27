from pathlib import Path
import subprocess,json,hashlib,os
p=Path(__file__).parent;assert json.loads((p/'deploy-gate.json').read_text())['sourceRuntime']=='PASS'
assert json.loads(subprocess.check_output(['docker','image','inspect','awesome-zero-platform/goal0063-api:9581630']))[0]['Id']=='sha256:4748dd85707ac7deeae073a27ed38953a830c81406fd05c0c95b4233cfa29a8e'
os.environ['DOCKER_BUILDKIT']='0'
source='efc0699b0d62ae1284db86121619a746ce4f37ba';out=json.loads((p/'images.json').read_text())
for kind in ['api']:
 tag='docker.io/awesome-zero-platform/goal0065-'+kind+':efc0699-runtime'
 with (p/(kind+'-image-build.log')).open('w') as f:subprocess.run(['docker','build','--pull=false','-t',tag,str(p/(kind+'-context'))],stdout=f,stderr=subprocess.STDOUT,check=True)
 info=json.loads(subprocess.check_output(['docker','image','inspect',tag]))[0];assert info['Config']['Labels']['org.opencontainers.image.revision']==source
 archive=p/(kind+'-image.tar');subprocess.run(['docker','save','-o',str(archive),tag],check=True)
 with (p/(kind+'-image-import.log')).open('w') as f:subprocess.run(['ctr','-n','k8s.io','images','import',str(archive)],stdout=f,stderr=subprocess.STDOUT,check=True)
 rows=subprocess.check_output(['ctr','-n','k8s.io','images','ls'],text=True);row=next(x.split() for x in rows.splitlines() if x.split()[0]==tag);digest=row[2];assert digest.startswith('sha256:');image=tag.split(':')[0]+'@'+digest;subprocess.run(['ctr','-n','k8s.io','images','tag',tag,image],check=True,stdout=subprocess.DEVNULL);archive.unlink();out[kind]={'tag':tag,'image':image,'digest':digest,'configID':info['Id'],'revision':source,'localImportOnly':True};(p/'images.json').write_text(json.dumps(out,indent=2));print(kind,digest,flush=True)
