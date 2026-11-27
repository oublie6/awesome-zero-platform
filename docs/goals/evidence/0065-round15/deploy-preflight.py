from pathlib import Path
import json,gzip,hashlib,subprocess
from preflight import P,K,k,cmd,sql,save
source='efc0699b0d62ae1284db86121619a746ce4f37ba'
assert subprocess.check_output(['git','rev-parse','HEAD'],text=True).strip()==source
assert not subprocess.check_output(['git','status','--porcelain'],text=True)
for n in ['storage-audit','exception-audit','vue-reentry']:assert json.loads((P/(n+'.json')).read_text())['fail']==0
for n in ['smoke','cache-smoke']:assert json.loads((P/(n+'.json')).read_text())['result']=='PASS'
a=json.loads((P/'browser-logout.json').read_text());assert a['fail']==0 and a['cases'][0]['refreshDeniedWithPersistentDeletionFault'] and a['cases'][0]['normalStorageAndFreshLogin']
b=json.loads((P/'browser-logout-initial-observer.json').read_text());assert all(x['pass'] for x in b['cases'] if x['name']!='confirmed')
assert '# pass 40' in (P/'vue-tests.log').read_text() and '# fail 0' in (P/'vue-tests.log').read_text()
assert 'built in' in (P/'vue-build.log').read_text()
assert int(sql("SELECT COUNT(*) FROM goal0057.doudizhu_hands WHERE phase NOT IN ('COMPLETED','ABORTED','CANCELLED','EXPIRED');"))==0
assert int(sql('SELECT COUNT(*) FROM goal0057.doudizhu_turn_deadlines;'))==0
for r in ['deployments','pods','services','pvc','configmaps','secrets','statefulsets']:save(r+'-predeploy.json',k('get',r,'-o','json'))
d=json.loads((P/'deployments-predeploy.json').read_text());before=json.loads((P/'deployments-before.json').read_text());assert {x['metadata']['name']:x['spec'] for x in d['items']}=={x['metadata']['name']:x['spec'] for x in before['items']}
old={x['metadata']['name']:x['spec']['template']['spec']['containers'][0]['image'] for x in d['items'] if x['metadata']['name'] in ['app-api','game-web']};rows=cmd(['ctr','-n','k8s.io','images','ls']).decode();assert all(x in rows for x in old.values())
backup=cmd(K+['exec','deployment/mysql','--','sh','-c','MYSQL_PWD="$MYSQL_ROOT_PASSWORD" exec mysqldump -uroot --single-transaction --quick --hex-blob --routines --triggers --skip-add-locks --set-gtid-purged=OFF goal0057']);assert b'Dump completed' in backup and b'CREATE TABLE' in backup;save('goal0057-predeploy.sql.gz',gzip.compress(backup));assert gzip.decompress((P/'goal0057-predeploy.sql.gz').read_bytes())==backup
schema=sql("SELECT table_name,column_name,column_type,is_nullable,column_default,extra FROM information_schema.columns WHERE table_schema='goal0057' ORDER BY table_name,ordinal_position;");save('schema-predeploy.sha256',hashlib.sha256(schema.encode()).hexdigest().encode())
save('deploy-gate.json',{'source':source,'sourceRuntime':'PASS','unfinishedHands':0,'deadlines':0,'rollbackImages':old,'rollbackImagesPresent':True,'backupBytes':len(backup),'backupSHA256':hashlib.sha256(backup).hexdigest(),'schemaSHA256':hashlib.sha256(schema.encode()).hexdigest(),'restoreEvidenceReused':'Round9-14 unchanged schema and API binary; no DDL/data migration','permittedMutation':'Only app-api then game-web container image; inverse rollback','candidateSourceClean':True});print('Gate PASS: source/runtime, zero unfinished/deadlines, fresh private backup, rollback images present')
