import {test} from 'node:test'
import assert from 'node:assert/strict'
import {sortHandCards,faceOfCard,publicCardCode,publicCardCodes,relativeOpponents,handFan}
 from '../dist/bootstrap/TableGeometry.js'

test('Dou Dizhu sorts ranks descending with deterministic suit ties',()=>{
 const original=['D3','S2','HJ','C10','YJ','H3','XJ','DA','SK','D10','S3','H2']
 const copy=[...original]
 assert.deepEqual(sortHandCards(original),
   ['YJ','XJ','S2','H2','DA','SK','HJ','C10','D10','S3','H3','D3'])
 assert.deepEqual(original,copy,'presentational sorting cannot mutate server view')
 assert.deepEqual(sortHandCards([]),[])
 assert.deepEqual(sortHandCards(['D5']),['D5'])
 assert.equal(faceOfCard('H10').red,true)
 assert.equal(faceOfCard('S10').suit,'♠')
 assert.equal(faceOfCard('YJ').rank,'大王')
 assert.throws(()=>faceOfCard('bad'))
})
test('public numeric card IDs map exactly to Go canonical deck',()=>{
 assert.equal(publicCardCode(0),'C3')
 assert.equal(publicCardCode(3),'S3')
 assert.equal(publicCardCode(48),'C2')
 assert.equal(publicCardCode(51),'S2')
 assert.equal(publicCardCode(52),'XJ')
 assert.equal(publicCardCode(53),'YJ')
 const deck=Array.from({length:54},(_,n)=>publicCardCode(n))
 assert.equal(new Set(deck).size,54)
 assert.deepEqual(publicCardCodes([0,53,52,51]),['YJ','XJ','S2','C3'])
 for(const invalid of [-1,54,1.4,'52',null,undefined])assert.equal(publicCardCode(invalid),null)
 assert.deepEqual(publicCardCodes([1,'2']),[])
 assert.deepEqual(publicCardCodes(Array(21).fill(1)),[])
})
test('three seats rotate consistently relative to the local player',()=>{
 assert.deepEqual(relativeOpponents(1),{left:3,right:2})
 assert.deepEqual(relativeOpponents(2),{left:1,right:3})
 assert.deepEqual(relativeOpponents(3),{left:2,right:1})
 assert.throws(()=>relativeOpponents(0))
})
test('full 20-card hand fits narrow portrait and short landscape with no paging',()=>{
 const cards=Array.from({length:20},(_,n)=>publicCardCode(n))
 for(const [width,height] of [[360,750],[390,844],[760,390],[1280,720],[1216,2770]]){
   const fan=handFan(cards.length,width,height,new Set(),cards)
   assert.equal(fan.places.length,20)
   assert.ok(fan.cardWidth>=50,'readable hand cards at '+width+'x'+height)
   assert.ok(fan.step>0)
   for(const {x,y} of fan.places){
     assert.ok(x-fan.cardWidth/2>=-width/2-0.01,'left overflow '+width+' '+x)
     assert.ok(x+fan.cardWidth/2<=width/2+0.01,'right overflow '+width+' '+x)
     assert.ok(y-fan.cardHeight/2>=-height/2,'bottom overflow '+height+' '+y)
   }
   const picked=handFan(cards.length,width,height,new Set([cards[7]]),cards)
   assert.ok(picked.places[7].y>fan.places[7].y+10)
   assert.deepEqual(picked.places.map(p=>p.x),fan.places.map(p=>p.x))
 }
})
test('table view is a distinct authority-preserving scene, not a paged menu',async()=>{
 const {readFile}=await import('node:fs/promises')
 const scene=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 const table=await readFile(new URL('../assets/scripts/bootstrap/GameTableView.ts',import.meta.url),'utf8')
 const branch=scene.split("} else if (screen === 'table') {")[1].split("} else {\n      this.title!.string = '玩法介绍'")[0]
 assert.match(branch,/new GameTableView\(/)
 assert.doesNotMatch(branch,/addMenu\(/)
 assert.match(scene,/this\.selectedCards\.forEach/)
 assert.match(table,/sortHandCards\(view!\.cards\)/)
 assert.match(table,/this\.callbacks\.selectCard\(code\)/)
 assert.match(table,/this\.callbacks\.play/)
 assert.match(table,/publicCardCodes\(playing\.leadingCards\)/)
})
