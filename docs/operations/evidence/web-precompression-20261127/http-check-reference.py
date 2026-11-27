from pathlib import Path
import json,ssl,urllib.request,urllib.error,gzip,hashlib,subprocess,sys,os
p=Path(__file__).parent;port=int(sys.argv[1]);ctx=ssl.create_default_context(cafile='/root/goal0065-round15-efc0699/test-ca.crt');manifest=json.loads((p/'context/precompression-manifest.json').read_text());out={'port':port,'entries':[]}
def get(path,encoding='gzip',etag=None):
 headers={'Accept-Encoding':encoding}
 if etag:headers['If-None-Match']=etag
 try:r=urllib.request.urlopen(urllib.request.Request('https://127.0.0.1:'+str(port)+path,headers=headers),context=ctx,timeout=30)
 except urllib.error.HTTPError as e:
  if e.code!=304:raise
  r=e
 with r:return r.status,dict(r.headers),r.read()
for e in manifest['entries']:
 status,h,b=get('/'+e['path']);assert status==200 and h.get('Content-Encoding')=='gzip' and 'Accept-Encoding' in h.get('Vary','');assert int(h['Content-Length'])==e['gzipBytes'];assert hashlib.sha256(b).hexdigest()==e['gzipSHA256'];assert hashlib.sha256(gzip.decompress(b)).hexdigest()==e['sha256'];out['entries'].append({'path':e['path'],'gzipBytes':len(b),'originalBytes':e['originalBytes'],'mime':h.get('Content-Type'),'cache':h.get('Cache-Control'),'staticFileExactMatch':True,'etag':h['ETag']})
cc=next(e for e in manifest['entries'] if e['path']=='game/cocos-js/cc.js');path='/'+cc['path'];status,gh,gb=get(path);out['negotiation']=[]
for encoding in ['identity','gzip;q=0','gzip;q=0, br;q=1','*;q=0']:
 status,h,b=get(path,encoding);assert status==200 and 'Content-Encoding' not in h and hashlib.sha256(b).hexdigest()==cc['sha256'];out['negotiation'].append({'acceptEncoding':encoding,'status':status,'bytes':len(b),'compressed':False})
status,ih,ib=get(path,'identity');status,h,b=get(path,etag=gh['ETag']);assert status==304 and len(b)==0;status,h,b=get(path,'identity',ih['ETag']);assert status==304 and len(b)==0;assert get(path,'identity',gh['ETag'])[0]==200 and get(path,'gzip',ih['ETag'])[0]==200;out['cacheVariants']={'gzip304':True,'identity304':True,'crossVariantETagsNotReused':True}
for original in manifest['originals']:
 if original['path']=='game-web-assets.json' or original['path'].endswith('index.html'):
  status,h,b=get('/'+original['path']);decoded=gzip.decompress(b) if h.get('Content-Encoding')=='gzip' else b;assert hashlib.sha256(decoded).hexdigest()==original['sha256'];assert 'Content-Security-Policy' in h;out.setdefault('manifestAndHTML',[]).append({'path':original['path'],'hashMatches':True,'cache':h.get('Cache-Control')})
if port==30474:
 prefix='/game/precompression-probes/'
 for name,mime in [('probe.svg','image/svg+xml'),('probe.wasm','application/wasm')]:
  status,h,b=get(prefix+name);assert h['Content-Type'].startswith(mime) and h['Content-Encoding']=='gzip';assert gzip.decompress(b)==(p/'probes'/name).read_bytes()
 status,oldh,b=get(prefix+'update.js');before=gzip.decompress(b);(p/'probes/update.js').write_text('const after = '+repr('different updated data '*2000)+';');st=(p/'probes/update.js').stat();os.utime(p/'probes/update.js',(st.st_atime,st.st_mtime+3));subprocess.run(['/opt/goal0042-toolchain/node-v22.23.4-linux-x64/bin/node','/root/awesome-zero-platform/scripts/game-web/precompress.cjs',str(p/'probes'),str(p/'probes-manifest.json')],check=True,stdout=subprocess.DEVNULL);status,newh,b=get(prefix+'update.js',etag=oldh['ETag']);assert status==200 and newh['ETag']!=oldh['ETag'] and gzip.decompress(b)==(p/'probes/update.js').read_bytes() and gzip.decompress(b)!=before
 (p/'probes/fallback.js.gz').unlink();status,h,b=get(prefix+'fallback.js');assert status==200 and h.get('Content-Encoding')=='gzip' and 'Content-Length' not in h and b[8]!=2 and gzip.decompress(b)==(p/'probes/fallback.js').read_bytes();out['fixtureControls']={'svgAndWasmMIME':True,'updatedResourceNoStaleGzip':True,'dynamicFallbackWithoutSidecar':True}
status,h,b=get('/health/ready');assert status==200 and json.loads(b)['status']=='ready' and 'Content-Encoding' not in h;out['proxyReadiness']=200;out['result']='PASS';(p/('http-'+str(port)+'.json')).write_text(json.dumps(out,indent=2));print('HTTP static bytes/MIME/negotiation/variants304/CSP/manifest/update/fallback PASS',port)
