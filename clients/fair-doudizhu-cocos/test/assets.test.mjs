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

test('Creator 3.8 card selections use Set.forEach instead of downlevel-unsafe iteration',async()=>{
 const source=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 assert.doesNotMatch(source,/\[\s*\.\.\.\s*this\.selectedCards\s*\]/)
 assert.doesNotMatch(source,/for\s*\(\s*const\s+\w+\s+of\s+this\.selectedCards\s*\)/)
 assert.match(source,/this\.selectedCards\.forEach\(card\s*=>\s*selectedCards\.push\(card\)\)/)
 assert.match(source,/actions\.play\(selectedCards\)/)
})

test('native UI exposes a room leave control and displays verified archive completion',async()=>{
 const source=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 assert.match(source,/name:'LeaveRoom'/)
 assert.match(source,/online\.leaveRoom\(\)/)
 assert.match(source,/terminalEvidence/)
 assert.match(source,/roomGeneration/)
 assert.match(source,/useAuthoritativeHand/)
 assert.match(source,/api\.finalEvidence\(handID\)/)
})

test('native fairness backup gate requires explicit user-held re-entry before commitment',async()=>{
 const source=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 assert.match(source,/name:'VerifyFairnessBackup'/)
 assert.match(source,/beginFairnessBackupVerification\(\)/)
 assert.match(source,/this\.roomInput\.string=''/)
 assert.match(source,/backupVerificationPendingHand!==handID/)
 assert.match(source,/this\.verifiedBackupController=online/)
 assert.match(source,/this\.verifiedBackupHand!==this\.handID/)
 assert.match(source,/this\.verifiedBackupController!==online/)
 assert.match(source,/online\.fairness\.importRecoveryKit\(kit,handID\)/)
 assert.match(source,/online\.fairness\.commit\(this\.handID\)/)
})

test('delayed fairness kit UI updates are fenced after leave, hand and login replacement',async()=>{
 const source=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 const prepare=source.split('private async prepareFairness():Promise<void>{')[1].split('private beginFairnessBackupVerification()')[0]
 assert.match(prepare,/const roomID=this\.roomID/)
 assert.match(prepare,/const handID=this\.handID/)
 assert.match(prepare,/const generation=this\.roomGeneration/)
 assert.match(prepare,/const fairness=online\?\.fairness/)
 assert.match(prepare,/await fairness\.prepareRecoveryKit\(handID\)/)
 assert.match(prepare,/generation!==this\.roomGeneration/)
 assert.match(prepare,/handID!==this\.handID/)
 assert.match(prepare,/roomID!==this\.roomID/)
 assert.match(prepare,/fairness!==online\.fairness/)
 assert.match(prepare,/input!==this\.roomInput/)
 assert.ok(prepare.indexOf('generation!==this.roomGeneration')<prepare.indexOf('input.string=kit'))
 assert.match(source,/online\.confirmActiveFairnessHand\(handID\)/)
 assert.match(source,/online\.confirmActiveFairnessHand\(info\.activeHandId \|\| null\)/)
})
