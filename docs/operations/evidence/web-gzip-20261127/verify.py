from pathlib import Path
import json,urllib.request,ssl,gzip,hashlib,sys
p=Path(__file__).parent;port=int(sys.argv[1]);ctx=ssl.create_default_context(cafile='/root/goal0065-round15-efc0699/test-ca.crt')
def request(path,encoding='gzip',etag=None,token=None,body=None):
 headers={'Accept-Encoding':encoding}
 if etag:headers['If-None-Match']=etag
 if token:headers['Authorization']='Bearer '+token
 data=None
 if body is not None:headers['Content-Type']='application/json';data=json.dumps(body).encode()
 r=urllib.request.Request('https://127.0.0.1:'+str(port)+path,headers=headers,data=data)
 try:response=urllib.request.urlopen(r,context=ctx,timeout=25)
 except urllib.error.HTTPError as e:
  if e.code!=304:raise
  response=e
 with response:return response.status,dict(response.headers),response.read()
out={'port':port,'cases':[]}
for path in ['/game/cocos-js/cc.js','/assets/index-M5cYNUMW.css','/game/index.html']:
 status,h,data=request(path);assert status==200 and h.get('Content-Encoding')=='gzip';original=Path('/root/goal0065-round15-efc0699/web-context/site'+path).read_bytes();assert hashlib.sha256(gzip.decompress(data)).digest()==hashlib.sha256(original).digest();assert 'Accept-Encoding' in h.get('Vary','');out['cases'].append({'path':path,'status':status,'encoding':h['Content-Encoding'],'gzipBytes':len(data),'originalBytes':len(original),'cache':h.get('Cache-Control'),'vary':h.get('Vary'),'decompressedSHA256':hashlib.sha256(original).hexdigest()})
 if path.endswith('cc.js'):
  assert len(data)<4_000_000;status,x,b=request(path,etag=h['ETag']);assert status==304 and len(b)==0;out['conditional']={'status':status,'bytes':len(b),'etag':h['ETag']};status,x,b=request(path,encoding='identity');assert status==200 and 'Content-Encoding' not in x and b==original;out['identityFallback']={'status':status,'bytes':len(b)}
status,h,b=request('/game/game-config.js');assert status==200 and h.get('Cache-Control')=='no-store'
a=json.loads(Path('/root/goal0065-round15-efc0699/deployed/accounts.json').read_text())[0];token=None
try:
 status,h,b=request('/auth/login',body={'identifier':a['username'],'password':a['password']});assert status==200 and 'Content-Encoding' not in h;token=json.loads(b)['data']['accessToken'];status,h,b=request('/auth/session',token=token);assert status==200 and 'Content-Encoding' not in h and 'no-store' in h['Cache-Control'];assert json.loads(b)['data']['account']['id']==a['id'];out['auth']={'status':status,'compressed':False,'noStore':True,'identityMatches':True}
finally:
 if token:
  status,h,b=request('/auth/logout',token=token,body={});assert status==200 and json.loads(b)['data']['loggedOut'] is True
out['result']='PASS';(p/('verify-'+str(port)+'.json')).write_text(json.dumps(out,indent=2));print(json.dumps(out))
