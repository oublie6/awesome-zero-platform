import json,hashlib,base64,subprocess
from preflight import P,k,sql,save
out={}
for resource in ['deployments','services','pvc','configmaps','statefulsets']:
 before=json.loads((P/(resource+'-before.json')).read_text());after=json.loads(k('get',resource,'-o','json'));save(resource+'-final.json',after);old={x['metadata']['name']:x for x in before['items']}
 same=True
 for x in after['items']:
  b=old[x['metadata']['name']];aSpec=x.get('spec',x.get('data'));bSpec=b.get('spec',b.get('data'))
  if resource=='deployments' and x['metadata']['name'] in ['app-api','game-web']:
   aSpec=json.loads(json.dumps(aSpec));bSpec=json.loads(json.dumps(bSpec));aSpec['template']['spec']['containers'][0]['image']=bSpec['template']['spec']['containers'][0]['image']
  same=same and aSpec==bSpec and x['metadata']['uid']==b['metadata']['uid']
 out[resource+'PreservedExceptAuthorizedImages']=same
secrets=json.loads(k('get','secrets','-o','json'));h={x['metadata']['name']:{key:hashlib.sha256(base64.b64decode(value)).hexdigest() for key,value in x.get('data',{}).items()} for x in secrets['items']};out['secretValuesUnchanged']=h==json.loads((P/'secret-hashes-before.json').read_text())
pods=json.loads(k('get','pods','-o','json'));out['pods']=[{'name':x['metadata']['name'],'ready':all(c.get('ready') for c in x['status'].get('containerStatuses',[])),'restarts':[c['restartCount'] for c in x['status'].get('containerStatuses',[])],'images':[c['image'] for c in x['spec']['containers']]} for x in pods['items']];baseline={x['metadata']['name']:x for x in json.loads((P/'pods-before.json').read_text())['items']};out['dependencyPodUIDsAndRestartsUnchanged']=all(x['metadata']['uid']==baseline[x['metadata']['name']]['metadata']['uid'] and x['status']['containerStatuses'][0]['restartCount']==baseline[x['metadata']['name']]['status']['containerStatuses'][0]['restartCount'] for x in pods['items'] if not x['metadata']['name'].startswith(('app-api-','game-web-')))
schema=sql("SELECT table_name,column_name,column_type,is_nullable,column_default,extra FROM information_schema.columns WHERE table_schema='goal0057' ORDER BY table_name,ordinal_position;");out['schemaUnchanged']=hashlib.sha256(schema.encode()).hexdigest()==(P/'schema-predeploy.sha256').read_text();out['handPhases']=sql('SELECT phase,COUNT(*) FROM goal0057.doudizhu_hands GROUP BY phase;');out['unfinishedHands']=int(sql("SELECT COUNT(*) FROM goal0057.doudizhu_hands WHERE phase NOT IN ('COMPLETED','ABORTED','CANCELLED','EXPIRED');"));out['deadlines']=int(sql('SELECT COUNT(*) FROM goal0057.doudizhu_turn_deadlines;'))
nodes=json.loads(subprocess.check_output(['kubectl','--context','kubernetes-admin@network-learning','get','nodes','-o','json']));out['nodesReady']=all(any(c['type']=='Ready' and c['status']=='True' for c in x['status']['conditions']) for x in nodes['items']);out['user29457Service']=subprocess.check_output(['systemctl','is-active','goal0063-game-test-forward-v2.service'],text=True).strip()
assert all(v is True for k,v in out.items() if isinstance(v,bool)),out
assert len(out['pods'])==6 and all(x['ready'] for x in out['pods']) and out['unfinishedHands']==0 and out['deadlines']==0 and out['user29457Service']=='active'
save('final-integrity.json',out);print('Final scope/infrastructure/schema/secret/pod/forward integrity PASS')
