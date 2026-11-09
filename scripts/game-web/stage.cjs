'use strict';
// Assemble verified Cocos4 web-desktop output with Vue3 assets on one origin.
// No Cocos engine substitution, no frontend account ID, no credential bundling.
const fs=require('node:fs')
const path=require('node:path')
const crypto=require('node:crypto')
const {configure}=require('../cocos-linux-web/configure.cjs')
function files(directory){
  const results=[]
  function visit(root){
    for(const entry of fs.readdirSync(root,{withFileTypes:true})){
      const location=path.join(root,entry.name)
      const st=fs.lstatSync(location)
      if(st.isSymbolicLink())throw Error('Symlink forbidden in public web assets: '+location)
      if(st.isDirectory())visit(location)
      else if(st.isFile())results.push(location)
      else throw Error('Unsupported public web file type: '+location)
    }
  }
  visit(directory)
  return results
}
function stage(cocosSource,siteDestination,config={}){
  const source=fs.realpathSync(cocosSource)
  const site=fs.realpathSync(siteDestination)
  if(source===site || site.startsWith(source+path.sep) || source.startsWith(site+path.sep))
    throw Error('Cocos export and Vue build must be disjoint')
  for(const target of [path.join(source,'index.html'),path.join(site,'index.html')])
    if(!fs.statSync(target).isFile())throw Error('Expected real built HTML: '+target)
  const raw=fs.readFileSync(path.join(source,'index.html'),'utf8')
  if(!/GameCanvas|Cocos3dGameContainer|System\.import/.test(raw))
    throw Error('Cocos web export missing engine entrypoint')
  const inputs=files(source)
  if(inputs.filter(name=>name.endsWith('.js')).length===0)throw Error('Cocos export has no generated JavaScript')
  const dest=path.join(site,'game')
  if(fs.existsSync(dest))throw Error('Refusing to overwrite an already staged /game export')
  fs.cpSync(source,dest,{recursive:true,errorOnExist:true,force:false})
  // The Vue shell and embedded frame receive exactly the same public endpoints.
  // Incomplete config deliberately leaves auth disabled; never infer localhost.
  configure(dest,config)
  fs.copyFileSync(path.join(dest,'game-config.js'),path.join(site,'game-config.js'))
  const names=files(site).sort()
  const manifest={format:'azp-doudizhu-web-assets-v1',
    configured:!!(config.apiBase&&config.socketEndpoint),
    files:names.map(file=>({path:path.relative(site,file).split(path.sep).join('/'),
      bytes:fs.statSync(file).size,sha256:crypto.createHash('sha256').update(fs.readFileSync(file)).digest('hex')}))}
  const manifestName=path.join(site,'game-web-assets.json')
  fs.writeFileSync(manifestName,JSON.stringify(manifest,null,2)+'\n')
  return manifest
}
if(require.main===module){
 const cocos=process.argv[2],site=process.argv[3]
 if(!cocos||!site)throw Error('Usage: stage.cjs <real-Cocos4-Web-export-root> <Vue3-dist-root>')
 const roots=process.env.WEB_REVEAL_MANIFEST_ROOTS?JSON.parse(process.env.WEB_REVEAL_MANIFEST_ROOTS):{}
 const result=stage(cocos,site,{apiBase:process.env.WEB_API_BASE,
   socketEndpoint:process.env.WEB_SOCKET_ENDPOINT,revealManifestRoots:roots})
 console.log('Staged real Cocos Web artifact with Vue shell: '+result.files.length+' assets')
}
module.exports={stage}
