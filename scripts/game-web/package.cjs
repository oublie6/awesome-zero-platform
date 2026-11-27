'use strict'
const fs=require('node:fs')
const path=require('node:path')
const {precompress,files}=require('./precompress.cjs')
async function packageSite(siteDirectory,nginxConfig,outputDirectory){
 const site=fs.realpathSync(siteDirectory),config=fs.realpathSync(nginxConfig)
 let ancestor=path.resolve(outputDirectory),missing=[]
 while(!fs.existsSync(ancestor)){missing.unshift(path.basename(ancestor));ancestor=path.dirname(ancestor)}
 const output=path.join(fs.realpathSync(ancestor),...missing)
 if(output===path.parse(output).root)throw Error('Filesystem root cannot be a package output')
 if(fs.existsSync(path.resolve(outputDirectory))&&fs.lstatSync(path.resolve(outputDirectory)).isSymbolicLink())throw Error('Symlink package output forbidden')
 if(output===site||output.startsWith(site+path.sep)||site.startsWith(output+path.sep)||config.startsWith(output+path.sep))throw Error('Package inputs and output must be disjoint')
 files(site)
 if(fs.existsSync(output)){
  if(fs.lstatSync(output).isSymbolicLink())throw Error('Symlink package output forbidden')
  files(output)
  const marker=path.join(output,'.azp-package.json')
  if(!fs.existsSync(marker)||JSON.parse(fs.readFileSync(marker,'utf8')).format!=='azp-web-package-v1')throw Error('Refusing to replace unowned package directory')
 }
 const parent=path.dirname(output);fs.mkdirSync(parent,{recursive:true})
 const temporary=fs.mkdtempSync(path.join(parent,'.azp-web-package-'))
 try{
  fs.cpSync(site,path.join(temporary,'site'),{recursive:true,preserveTimestamps:true})
  const manifest=await precompress(path.join(temporary,'site'),path.join(temporary,'precompression-manifest.json'))
  fs.copyFileSync(config,path.join(temporary,'nginx.conf'))
  const repo=path.resolve(__dirname,'../..')
  for(const name of ['cache-policy.nginx.conf','gzip-static.nginx.conf'])fs.copyFileSync(path.join(repo,'deploy/game-web',name),path.join(temporary,name))
  fs.copyFileSync(path.join(repo,'deploy/docker/game-web.Dockerfile'),path.join(temporary,'Dockerfile'))
  fs.writeFileSync(path.join(temporary,'.azp-package.json'),JSON.stringify({format:'azp-web-package-v1'})+'\n')
  if(fs.existsSync(output))fs.rmSync(output,{recursive:true})
  fs.renameSync(temporary,output)
  return manifest
 }finally{if(fs.existsSync(temporary))fs.rmSync(temporary,{recursive:true,force:true})}
}
if(require.main===module){
 const [site,config,output]=process.argv.slice(2)
 if(!site||!config||!output)throw Error('Usage: package.cjs <verified-public-site> <nginx-server.conf> <owned-build-context>')
 packageSite(site,config,output).then(r=>console.log('Packaged '+r.originals.length+' unchanged originals and '+r.entries.length+' gzip sidecars')).catch(e=>{console.error(e.message);process.exitCode=1})
}
module.exports={packageSite}
