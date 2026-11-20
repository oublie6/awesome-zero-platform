import {test} from 'node:test'
import assert from 'node:assert/strict'
import {sortHandCards,faceOfCard,publicCardCode,publicCardCodes,relativeOpponents,handFan,tableHeaderSlots,tableActionCenterY,tableUnit,tableCaptionFont}
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

test('header title, clock and room buttons do not overlap on target screens',()=>{
  for(const [width,height] of [[240,320],[360,750],[390,844],[760,390],[1280,720]]){
    const u=Math.max(.72,Math.min(1.35,Math.min(width/760,height/480)))
    const slots=tableHeaderSlots(width,u)
    const leftEdge=slots.leftX-slots.leftWidth/2
    const leftEnd=slots.leftX+slots.leftWidth/2
    const clockLeft=-slots.clockWidth/2, clockRight=slots.clockWidth/2
    const rightStart=slots.rightX-slots.rightWidth/2
    const rightEnd=slots.rightX+slots.rightWidth/2
    assert.ok(leftEdge>=-width/2,'header left overflow '+width)
    assert.ok(leftEnd+4<=clockLeft,'header collides with clock '+width)
    assert.ok(clockRight+4<=rightStart,'clock collides with back button '+width)
    assert.ok(rightEnd<=width/2,'header right overflow '+width)
  }
})
test('the selected hand stays clear of play and bid buttons',()=>{
  const cards=Array.from({length:20},(_,n)=>publicCardCode(n))
  for(const [width,height] of [[360,750],[390,844],[760,390],[1280,720]]){
    const u=tableUnit(width,height)
    const selected=handFan(cards.length,width,height,new Set(cards),cards)
    const actionY=tableActionCenterY(selected,width,height,u)
    const neutral=handFan(cards.length,width,height,new Set(),cards)
    assert.equal(actionY,tableActionCenterY(neutral,width,height,u),
      'selecting a card shifted action hitboxes at '+width+'x'+height)
    const highestCardTop=Math.max(...selected.places.map(pos=>pos.y+selected.cardHeight/2))
    assert.ok(actionY-21*u>=highestCardTop+2*u,
      'selected card intercepts action touch target at '+width+'x'+height)
  }
})

test('the pass action is disabled until a public trick exists',async()=>{
  const {readFile}=await import('node:fs/promises')
  const table=await readFile(new URL('../assets/scripts/bootstrap/GameTableView.ts',import.meta.url),'utf8')
  assert.match(table,/myTurn && playing\.cards\.length>0,this\.callbacks\.pass/)
})

test('real Go []uint8 Base64 wire tricks preserve legality and card identities',()=>{
 assert.deepEqual(publicCardCodes('Ag=='),['H3'])
 assert.deepEqual(publicCardCodes('AAECAw=='),['S3','H3','C3','D3'])
 assert.deepEqual(publicCardCodes('NDU='),['YJ','XJ'])
 assert.deepEqual(publicCardCodes(''),[])
 for(const invalid of ['AA=','A===','Af8=','AP==','____', 'A'.repeat(32)]){
   assert.deepEqual(publicCardCodes(invalid),[],'invalid wire '+invalid)
 }
})
test('portrait design resolution retains physical touch sizes and playable fan',()=>{
 for(const [width,height,cssRatio] of [[390,844,1],[1216,2770,390/1216],[1280,2770,390/1280]]){
   const u=tableUnit(width,height)
   assert.ok(42*u*cssRatio>=40,'portrait operation height too small: '+width)
   assert.ok(35*u*cssRatio>=34,'portrait navigation height too small: '+width)
   const cards=Array.from({length:20},(_,n)=>publicCardCode(n))
   const fan=handFan(20,width,height,new Set(cards),cards)
   const y=tableActionCenterY(fan,width,height,u)
   const top=Math.max(...fan.places.map(p=>p.y+fan.cardHeight/2))
   assert.ok(y-21*u>top,'portrait selected hand collides with buttons '+width)
 }
})
test('table view consumes terminal evidence without a live public projection',async()=>{
 const {readFile}=await import('node:fs/promises')
 const scene=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 const table=await readFile(new URL('../assets/scripts/bootstrap/GameTableView.ts',import.meta.url),'utf8')
 assert.match(scene,/snapshotView\?\.terminalEvidence/)
 assert.match(scene,/!final && snapshotView\?\.public/)
 assert.match(table,/!view && data\.handID && !final/)
 assert.match(table,/this\.callbacks\.evidence/)
})
test('monotonic turn clock survives view recreation and ignores stale timestamps',async()=>{
 const {TurnClockAnchor}=await import('../dist/bootstrap/TurnClock.js')
 const c=new TurnClockAnchor()
 const snapshot={turnDeadlineMillis:130000,serverTimeMillis:100000}
 c.observe('h',7,snapshot,500)
 assert.equal(c.remainingSeconds(26500),4)
 c.observe('h',7,snapshot,26500)
 assert.equal(c.remainingSeconds(26500),4,'view reopen extended countdown')
 c.observe('h',7,{turnDeadlineMillis:130000,serverTimeMillis:110000},26500)
 assert.equal(c.remainingSeconds(27500),3,'stale server clock moved anchor backwards')
 c.observe('h',8,{turnDeadlineMillis:180000,serverTimeMillis:150000},27500)
 assert.equal(c.remainingSeconds(27500),30,'new committed turn must reset')
})

test('table button captions remain readable in real CSS portrait scaling',async()=>{
 const actions=[
   ['返回',75,35],['同步',75,34],['不叫',110,38],
   ['3 分',110,38],['不出',150,42],['出牌 (20)',150,42],
   ['核验结算',272,38],['核对上次操作',150,36],
 ]
 for(const [w,h,cssRatio] of [[760,440,1],[1280,720,1],[1216,2770,390/1216]]){
   const u=tableUnit(w,h)
   for(const [caption,logicalWidth,logicalHeight] of actions){
     const pixelFont=tableCaptionFont(caption,logicalWidth*u,logicalHeight*u,u)*cssRatio
     assert.ok(pixelFont>=11,
       'unreadable caption '+caption+' at '+w+'x'+h+': '+pixelFont+' CSS px')
   }
 }
 const {readFile}=await import('node:fs/promises')
 const table=await readFile(new URL('../assets/scripts/bootstrap/GameTableView.ts',import.meta.url),'utf8')
 assert.match(table,/tableCaptionFont\(text,w,h,this\.unit\)/)
 assert.doesNotMatch(table,/Math\.min\(18,w\*\.19,h\*\.48\)/)
})

test('Round3 UNKNOWN action and room UI are refreshed from current owner state',async()=>{
 const {readFile}=await import('node:fs/promises')
 const boot=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 const table=await readFile(new URL('../assets/scripts/bootstrap/GameTableView.ts',import.meta.url),'utf8')
 const online=await readFile(new URL('../assets/scripts/game/GameOnlineController.ts',import.meta.url),'utf8')
 assert.match(boot,/onHandActionStateChange|this\.renderScreen\('table'\)/)
 assert.match(boot,/this\.online\?\.actions\.needsReconciliation/)
 assert.match(boot,/!this\.online\?\.actions\.isSubmitting/)
 assert.match(boot,/this\.roomRetryRendered!==!!this\.online\?\.rooms\.needsReconciliation/)
 assert.match(boot,/const input=this\.roomInput\?\.string/)
 assert.match(boot,/this\.roomInput\.string=input/)
 assert.match(table,/!data\.needsReconciliation && !data\.actionInFlight/)
 assert.match(online,/new GameHandActions\(this\.shell, platform\.random, onHandActionStateChange\)/)
})
