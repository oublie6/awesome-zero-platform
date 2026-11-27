import subprocess,json,os,hashlib,base64
from pathlib import Path
P=Path(__file__).parent;K=['kubectl','--context','kubernetes-admin@network-learning','-n','awesome-zero-game-test']
def cmd(args,inp=None):return subprocess.run(args,input=inp,stdout=subprocess.PIPE,stderr=subprocess.PIPE,check=True).stdout
def k(*args):return cmd(K+list(args))
def save(name,data):
 p=P/name;p.write_bytes(data if isinstance(data,bytes) else json.dumps(data,indent=2).encode());p.chmod(0o600)
def sql(q):return cmd(K+['exec','-i','deployment/mysql','--','sh','-c','MYSQL_PWD="$MYSQL_ROOT_PASSWORD" exec mysql -uroot --batch --skip-column-names'],q.encode()).decode()
if __name__=='__main__':
 assert cmd(['kubectl','config','current-context']).decode().strip()=='kubernetes-admin@network-learning'
 for resource in ['deployments','pods','services','pvc','configmaps','secrets','statefulsets']:
  save(resource+'-before.json',k('get',resource,'-o','json'))
 save('nodes-before.json',cmd(K[:3]+['get','nodes','-o','json']))
 for node in ['ser730417996259','hk-worker']:save('resources-before-'+node+'.json',cmd(K[:3]+['get','--raw','/api/v1/nodes/'+node+'/proxy/stats/summary']))
 tables=sql("SELECT TABLE_NAME,ENGINE FROM information_schema.tables WHERE TABLE_SCHEMA='goal0057' ORDER BY TABLE_NAME;")
 counts={line.split('\t')[0]:int(sql('SELECT COUNT(*) FROM goal0057.`'+line.split('\t')[0]+'`;')) for line in tables.splitlines()}
 phases=sql('SELECT phase,COUNT(*) FROM goal0057.doudizhu_hands GROUP BY phase;');active=sql("SELECT hand_id,phase FROM goal0057.doudizhu_hands WHERE phase NOT IN ('COMPLETED','ABORTED','EXPIRED','CANCELLED');")
 save('active-hands-private.txt',active.encode());save('table-counts-before.json',counts);save('table-engines-before.txt',tables.encode());save('phases-before.txt',phases.encode());save('schema-before.sql',sql('SHOW CREATE DATABASE goal0057;').encode())
 secrets=json.loads((P/'secrets-before.json').read_text());hashes={x['metadata']['name']:{key:hashlib.sha256(base64.b64decode(v)).hexdigest() for key,v in x.get('data',{}).items()} for x in secrets['items']};save('secret-hashes-before.json',hashes)
 dep=json.loads((P/'deployments-before.json').read_text());summary={'deployments':[{'name':x['metadata']['name'],'replicas':x['spec']['replicas'],'images':[c['image'] for c in x['spec']['template']['spec']['containers']]} for x in dep['items']],'tableCount':len(counts),'rows':counts,'phases':phases.strip(),'unfinishedHandCount':len(active.splitlines()),'secrets':{x['metadata']['name']:list(x.get('data',{})) for x in secrets['items']}}
 save('preflight-summary.json',summary);print(json.dumps(summary,indent=2))
