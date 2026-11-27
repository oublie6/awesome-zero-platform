'use strict'
const fs=require('node:fs')
const path=require('node:path')
const crypto=require('node:crypto')
const {pipeline}=require('node:stream/promises')
const {Transform,Writable}=require('node:stream')
const {createGzip,createGunzip}=require('node:zlib')
const EXTENSIONS=new Set(['.js','.mjs','.cjs','.css','.html','.htm','.json','.svg','.wasm','.txt','.xml','.map','.csv','.webmanifest'])
function files(root){
 const result=[]
 function visit(dir){for(const item of fs.readdirSync(dir,{withFileTypes:true})){
  const file=path.join(dir,item.name),stat=fs.lstatSync(file)
  if(stat.isSymbolicLink())throw Error('Symlink forbidden: '+file)
  if(stat.isDirectory())visit(file)
  else if(stat.isFile())result.push(file)
  else throw Error('Unsupported asset file: '+file)
 }}
 visit(root);return result.sort()
}
async function hash(file){
 const sha=crypto.createHash('sha256')
 await pipeline(fs.createReadStream(file),new Writable({write(chunk,encoding,done){sha.update(chunk);done()}}))
 return sha.digest('hex')
}
function relativeFile(root,name){
 if(typeof name!=='string'||!name||name.includes('\\')||name.split('/').some(p=>!p||p==='.'||p==='..')||path.isAbsolute(name))throw Error('Invalid compression ownership path')
 return path.join(root,...name.split('/'))
}
async function precompress(directory,manifestPath){
 const root=fs.realpathSync(directory),requested=path.resolve(manifestPath)
 const report=path.join(fs.realpathSync(path.dirname(requested)),path.basename(requested))
 if(report===root||report.startsWith(root+path.sep))throw Error('Compression manifest must be outside public assets')
 const all=files(root),owned=new Set()
 if(fs.existsSync(report)){
  if(fs.lstatSync(report).isSymbolicLink())throw Error('Symlink compression manifest forbidden')
  const previous=JSON.parse(fs.readFileSync(report,'utf8'))
  if(previous.format!=='azp-web-precompression-v1'||!Array.isArray(previous.entries))throw Error('Invalid compression ownership manifest')
  for(const entry of previous.entries){
   relativeFile(root,entry.path)
   if(entry.sidecar!==entry.path+'.gz')throw Error('Invalid sidecar ownership')
   owned.add(relativeFile(root,entry.sidecar))
  }
 }
 const originals=all.filter(file=>!owned.has(file))
 // Check all conflicts before deleting any previously generated sidecars.
 for(const file of originals)if(EXTENSIONS.has(path.extname(file).toLowerCase())&&fs.existsSync(file+'.gz')&&!owned.has(file+'.gz'))throw Error('Unowned gzip sidecar conflict: '+file+'.gz')
 for(const file of owned)if(fs.existsSync(file))fs.unlinkSync(file)
 const result={format:'azp-web-precompression-v1',level:9,minBytes:1024,minSavingBytes:128,minSavingPercent:5,originals:[],entries:[]},created=[]
 try{
  for(const file of originals){
   const stat=fs.statSync(file),name=path.relative(root,file).split(path.sep).join('/'),originalSHA=await hash(file)
   result.originals.push({path:name,bytes:stat.size,sha256:originalSHA})
   if(stat.size<result.minBytes||!EXTENSIONS.has(path.extname(file).toLowerCase()))continue
   const temporary=path.join(path.dirname(file),'.azp-gzip-'+crypto.randomBytes(12).toString('hex')+'.tmp')
   try{
    await pipeline(fs.createReadStream(file),createGzip({level:9}),fs.createWriteStream(temporary,{flags:'wx',mode:0o644}))
    const decoded=crypto.createHash('sha256')
    await pipeline(fs.createReadStream(temporary),createGunzip(),new Writable({write(chunk,encoding,done){decoded.update(chunk);done()}}))
    if(decoded.digest('hex')!==originalSHA||await hash(file)!==originalSHA)throw Error('Asset changed during compression: '+name)
    const bytes=fs.statSync(temporary).size
    if(stat.size-bytes<Math.max(result.minSavingBytes,Math.ceil(stat.size*.05)))continue
    fs.renameSync(temporary,file+'.gz');created.push(file+'.gz')
    fs.utimesSync(file+'.gz',stat.atimeMs/1000,stat.mtimeMs/1000)
    result.entries.push({path:name,sidecar:name+'.gz',originalBytes:stat.size,gzipBytes:bytes,sha256:originalSHA,gzipSHA256:await hash(file+'.gz')})
   }finally{if(fs.existsSync(temporary))fs.unlinkSync(temporary)}
  }
  const temporary=report+'.tmp-'+crypto.randomBytes(8).toString('hex')
  try{fs.writeFileSync(temporary,JSON.stringify(result,null,2)+'\n',{flag:'wx'});fs.renameSync(temporary,report)}
  finally{if(fs.existsSync(temporary))fs.unlinkSync(temporary)}
  return result
 }catch(error){for(const file of created)if(fs.existsSync(file))fs.unlinkSync(file);throw error}
}
if(require.main===module){
 const [site,manifest]=process.argv.slice(2)
 if(!site||!manifest)throw Error('Usage: precompress.cjs <public-site> <outside-site-manifest.json>')
 precompress(site,manifest).then(r=>console.log('Precompressed '+r.entries.length+' of '+r.originals.length+' original assets')).catch(e=>{console.error(e.message);process.exitCode=1})
}
module.exports={precompress,files,hash}
