'use strict'
const {test}=require('node:test')
const assert=require('node:assert/strict')
const fs=require('node:fs')
const path=require('node:path')
const os=require('node:os')
const {packageSite}=require('./package.cjs')
function fixture(t){const root=fs.mkdtempSync(path.join(os.tmpdir(),'azp-package-'));t.after(()=>fs.rmSync(root,{recursive:true,force:true}));const site=path.join(root,'source'),config=path.join(root,'server.conf'),output=path.join(root,'context');fs.mkdirSync(site);fs.writeFileSync(path.join(site,'index.html'),'<script>const untouched = true;</script>'.repeat(100));fs.writeFileSync(path.join(site,'game-web-assets.json'),'original manifest');fs.writeFileSync(config,'server { # exact CSP/TLS/server configuration\n}\n');return{root,site,config,output}}
test('packaging preserves originals, manifest/CSP/config and supports repeat/update',async t=>{
 const f=fixture(t),first=await packageSite(f.site,f.config,f.output)
 assert.ok(first.entries.length>0);assert.deepEqual(fs.readFileSync(path.join(f.output,'site/index.html')),fs.readFileSync(path.join(f.site,'index.html')))
 assert.equal(fs.readFileSync(path.join(f.output,'site/game-web-assets.json'),'utf8'),'original manifest')
 assert.deepEqual(fs.readFileSync(path.join(f.output,'nginx.conf')),fs.readFileSync(f.config))
 assert.deepEqual(await packageSite(f.site,f.config,f.output),first)
 fs.writeFileSync(path.join(f.site,'new.svg'),'<svg>updated</svg>'.repeat(100));await packageSite(f.site,f.config,f.output);assert.ok(fs.existsSync(path.join(f.output,'site/new.svg.gz')))
 fs.unlinkSync(path.join(f.site,'new.svg'));await packageSite(f.site,f.config,f.output);assert.equal(fs.existsSync(path.join(f.output,'site/new.svg.gz')),false)
})
test('packaging refuses unowned output and overlapping or symlink-alias input/output',async t=>{
 const f=fixture(t);fs.mkdirSync(f.output);fs.writeFileSync(path.join(f.output,'user-data'),'keep')
 await assert.rejects(packageSite(f.site,f.config,f.output),/unowned/);assert.equal(fs.readFileSync(path.join(f.output,'user-data'),'utf8'),'keep')
 await assert.rejects(packageSite(f.site,f.config,path.join(f.site,'context')),/disjoint/)
 fs.symlinkSync(f.site,path.join(f.root,'alias'));await assert.rejects(packageSite(f.site,f.config,path.join(f.root,'alias','nested')),/disjoint/)
})
