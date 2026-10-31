import {test} from 'node:test'
import assert from 'node:assert/strict'
import {readFile,readdir} from 'node:fs/promises'
const json=async path=>JSON.parse(await readFile(new URL(path,import.meta.url),'utf8'))
test('entry scene has valid references, Canvas camera and stable bootstrap script UUID',async()=>{
 const scene=await json('../assets/scenes/Entry.scene'),meta=await json('../assets/scenes/Entry.scene.meta')
 assert.equal(scene[0].__type__,'cc.SceneAsset');assert.equal(scene[1]._id,meta.uuid)
 const check=value=>{if(value&&typeof value==='object'){if('__id__' in value)assert.ok(scene[value.__id__]);Object.values(value).forEach(check)}};check(scene)
 const canvas=scene.find(x=>x.__type__==='cc.Canvas');assert.equal(scene[canvas._cameraComponent.__id__].__type__,'cc.Camera')
 const hex=(await json('../assets/scripts/bootstrap/GameBootstrap.ts.meta')).uuid.replaceAll('-','')
 const b64='ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/'
 let compressed=hex.slice(0,5);for(let i=5;i<32;i+=3){const n=parseInt(hex.slice(i,i+3),16);compressed+=b64[n>>6]+b64[n&63]}
 const bootstrap=scene.find(x=>x.__type__===compressed);assert.ok(bootstrap);assert.equal(bootstrap.node.__id__,canvas.node.__id__)
})
test('every source asset and directory has a unique stable meta UUID',async()=>{
 const uuids=new Set()
 async function visit(url){for(const entry of await readdir(url,{withFileTypes:true})){
  if(entry.name.endsWith('.meta'))continue
  const path=new URL(entry.name+(entry.isDirectory()?'/':''),url)
  const meta=JSON.parse(await readFile(new URL(entry.name+'.meta',url),'utf8'))
  assert.equal(uuids.has(meta.uuid),false);uuids.add(meta.uuid)
  if(entry.isDirectory())await visit(path)
 }}
 await visit(new URL('../assets/',import.meta.url))
})
