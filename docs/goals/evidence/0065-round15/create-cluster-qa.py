from pathlib import Path
import json,secrets,time
from importlib.machinery import SourceFileLoader
p=Path(__file__).parent;m=SourceFileLoader('cluster_session',str(p/'cluster-session.py')).load_module();accounts=[]
for i in range(2):
 username='g65r15_'+str(int(time.time()))+'_'+str(i);password=secrets.token_urlsafe(24)
 status,j=m.req('/auth/register',body={'username':username,'email':username+'@example.test','displayName':'Round15 QA '+str(i+1),'password':password});assert status in [200,201] and j['code']=='OK';accounts.append({'id':j['data']['account']['id'],'username':username,'password':password})
 f=p/'deployed/accounts.json';f.write_text(json.dumps(accounts));f.chmod(0o600)
(p/'cluster-qa-selection.json').write_text(json.dumps({'accountsCreatedThroughPublicRegistration':2,'historicalQARecoverableRoomsPreserved':True,'historicalAccountsChanged':False,'purpose':'Post-deployment owned room and account-switch smoke'},indent=2));print('Created two owned QA accounts; preserved all historical rooms/accounts')
