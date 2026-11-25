import json,subprocess,time,os,yaml,hashlib,gzip
from pathlib import Path
P=Path(__file__).parent
from preflight import K,cmd,save
assert not (P/'clone-created.json').exists()
# Read-only live backup; all imported writes remain in isolated local MySQL.
q=K+['exec','deployment/mysql','--','sh','-c','MYSQL_PWD="$MYSQL_ROOT_PASSWORD" exec mysqldump -uroot --single-transaction --skip-lock-tables --set-gtid-purged=OFF goal0057']
backup=cmd(q);save('cluster-backup.sql.gz',gzip.compress(backup))
for name in ['codex0063r8-restore','codex0063r8-redis','codex0063r8-queue-a','codex0063r8-queue-b']:cmd(['docker','start',name])
C=['docker','exec','-i','codex0063r8-restore','sh','-c','MYSQL_PWD="$MYSQL_ROOT_PASSWORD" exec mysql -uroot --batch --skip-column-names']
for i in range(50):
 x=subprocess.run(C,input=b'SELECT 1;',capture_output=True)
 if x.returncode==0:break
 time.sleep(.2)
else:raise RuntimeError('isolated mysql not ready')
cmd(C,b'CREATE DATABASE goal0065r9 CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;');cmd(C[:-1]+[C[-1]+' goal0065r9'],backup)
old=Path('/root/goal0063-round9-9581630');env=json.loads((old/'clone-runtime-env.json').read_text());d=json.loads(cmd(['docker','inspect','codex0063r8-restore']))[0];rootpw=next(v.split('=',1)[1] for v in d['Config']['Env'] if v.startswith('MYSQL_ROOT_PASSWORD='))
env.update(APP_MYSQL_ADDR='127.0.0.1:38306',APP_MYSQL_DATABASE='goal0065r9',APP_MYSQL_USER='root',APP_MYSQL_PASSWORD=rootpw,APP_REDIS_ADDR='127.0.0.1:38379',APP_DOUDIZHU_TURN_DQ_TUBE='g65r9-turn',APP_DOUDIZHU_TURN_DQ_ENABLED='true',APP_DOUDIZHU_TURN_DQ_ENDPOINTS='127.0.0.1:38300,127.0.0.1:38301')
save('clone-runtime-env.json',env)
c=yaml.safe_load((old/'clone-app.yaml').read_text());c['Port']=18869;c['Host']='127.0.0.1';c['Doudizhu']['BiddingTimeout']='300s';c['Doudizhu']['PlayingTimeout']='300s';c['Authentication']['SessionKeyPrefix']='g65r9:authn:session:';c['Observability']['Metrics']['Enabled']=False
save('clone-app.yaml',yaml.safe_dump(c).encode());save('clone-created.json',{'database':'goal0065r9','liveBackupSHA256':hashlib.sha256(backup).hexdigest(),'isolated':True,'schemaChanges':False,'queuePrefix':'g65r9-turn','authPrefix':'g65r9:authn:session:'})
print('Live backup restored to new isolated database; original schema/config/storage unchanged')
