import json,ssl,urllib.request
from pathlib import Path
p=Path(__file__).parent;ctx=ssl.create_default_context(cafile=str(p/'test-ca.crt'));a=json.loads((p/'deployed/accounts.json').read_text())[0]
def req(path,token=None,body=None):
 headers={'Content-Type':'application/json'}
 if token:headers['Authorization']='Bearer '+token
 q=urllib.request.Request('https://127.0.0.1:29457'+path,data=json.dumps(body).encode() if body is not None else None,headers=headers)
 with urllib.request.urlopen(q,context=ctx,timeout=20) as r:return r.status,json.load(r)
if __name__=='__main__':
 status,j=req('/auth/login',body={'identifier':a['username'],'password':a['password']});token=j['data']['accessToken'];status,s=req('/auth/session',token);assert status==200 and s['data']['account']['id']==a['id'];f=p/'cluster-session-private.json';f.write_text(json.dumps({'token':token}));f.chmod(0o600);(p/'cluster-session-before.json').write_text(json.dumps({'status':status,'ownedQAIdentityMatches':True},indent=2));print('Existing cluster QA login/session PASS; private captured session retained for upgrade check')
