import {test} from 'node:test'
import assert from 'node:assert/strict'
import {menuWindow} from '../dist/bootstrap/MenuPagination.js'

test('landscape crowded room menu remains paged with readable, non-overlapping targets',()=>{
 const width=760,height=390,count=24
 const first=menuWindow(count,width,height,false,'rooms',0)
 assert.equal(first.columns,3)
 assert.ok(first.pages>1)
 assert.ok(first.rowPitch>=first.buttonHeight+2)
 assert.ok(first.buttonHeight>=34)
 assert.ok(first.end-first.start+2<=first.columns*first.rows)
 const last=menuWindow(count,width,height,false,'rooms',999)
 assert.equal(last.page,last.pages-1)
 assert.equal(last.end,count)
 const seen=[]
 for(let p=0;p<first.pages;p++){
  const window=menuWindow(count,width,height,false,'rooms',p)
  for(let i=window.start;i<window.end;i++)seen.push(i)
  const bottom=window.firstY-(window.rows-1)*window.rowPitch-window.buttonHeight/2
  assert.ok(bottom>=-height/2-0.01,`outside safe area: ${bottom}`)
 }
 assert.deepEqual(seen,Array.from({length:count},(_,i)=>i))
})
test('historical menu paging stays compatible for legacy layout consumers',()=>{
 for(const portrait of [false,true]){
  const viewport=menuWindow(26,portrait?360:760,portrait?750:390,portrait,'table',0)
  assert.ok(viewport.pages>1)
  assert.equal(viewport.columns,portrait?4:6)
  assert.ok(viewport.buttonHeight>=34)
  const last=menuWindow(26,portrait?360:760,portrait?750:390,portrait,'table',viewport.pages-1)
  assert.equal(last.end,26)
 }
})
test('ordinary small screens retain a single visible menu page',()=>{
 for(const screen of ['home','rules']){
  const v=menuWindow(2,760,390,false,screen,0)
  assert.equal(v.pages,1)
  assert.equal(v.start,0)
  assert.equal(v.end,2)
 }
})
test('visible-page clamping supports resize without losing action access',()=>{
 const small=menuWindow(33,700,320,false,'rooms',50)
 assert.equal(small.page,small.pages-1)
 const big=menuWindow(33,1000,900,false,'rooms',small.page)
 assert.ok(big.page<=big.pages-1)
 assert.equal(menuWindow(0,760,390,false,'rooms',0).end,0)
})

test('portrait room captions use a full-width target rather than 9px shrink-to-fit',()=>{
 const narrow=menuWindow(20,358,750,true,'rooms',0)
 assert.equal(narrow.columns,1)
 assert.ok(narrow.pages>1)
 assert.ok(narrow.buttonHeight>=34)
 const wide=menuWindow(20,760,750,true,'rooms',0)
 assert.equal(wide.columns,2)
 const smallLandscape=menuWindow(15,600,320,false,'rooms',0)
 assert.equal(smallLandscape.columns,2)
})

test('G59 follow-up: login actions do not overlap the password input hitbox',()=>{
 // The password editor is centered at -height*0.035 and is at least 48
 // design units tall. Its bottom must clear the first menu button's top.
 for(const [width,height,portrait] of [
   [760,720,false],[760,592,false],[1216,2770,true]
 ]){
   const menu=menuWindow(2,width,height,portrait,'login',0)
   const passwordBottom=-height*0.035-Math.max(48,height*0.075)/2
   const buttonTop=menu.firstY+menu.buttonHeight/2
   assert.ok(buttonTop<passwordBottom-8,
     'login menu and password field overlap at '+width+'x'+height)
   assert.equal(menu.pages,1)
 }
 const room=menuWindow(9,760,592,false,'rooms',0)
 assert.equal(room.firstY,-592*0.103,'other screens keep their layouts')
})

test('room drafts survive only the exact in-place owner/room/hand/input-purpose refresh',async()=>{
 const {sameRoomInputContext}=await import('../dist/bootstrap/MenuPagination.js')
 const original={owner:{},roomID:'room-A',handID:'hand-1',fairnessMode:'client-contributed-server-custodied-v1'}
 assert.equal(sameRoomInputContext(original,{...original}),true)
 assert.equal(sameRoomInputContext(null,original),false)
 assert.equal(sameRoomInputContext(original,{...original,owner:{}}),false,
   'old account draft must never appear after relogin')
 assert.equal(sameRoomInputContext(original,{...original,roomID:'room-B'}),false)
 assert.equal(sameRoomInputContext(original,{...original,handID:'hand-2'}),false,
   'private contribution phrase must never leak to a new hand')
 assert.equal(sameRoomInputContext(original,{...original,fairnessMode:''}),false)
 assert.equal(sameRoomInputContext(original,{...original,fairnessMode:null}),false)
 assert.equal(sameRoomInputContext({...original,owner:null},{...original,owner:null}),true,
   'the user may continue editing a room code while unauthenticated')
})
test('all room rerenders use the same scoped draft protection, including create UNKNOWN',async()=>{
 const {readFile}=await import('node:fs/promises')
 const source=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 assert.match(source,/sameRoomInputContext\(this\.roomInputContext,nextRoomInputContext\)/)
 assert.match(source,/if\(draft!==undefined\)this\.roomInput\.string=draft/)
 assert.match(source,/this\.roomInputContext=nextRoomInputContext/)
 const create=source.split('private async createRoom()')[1].split('private async joinRoom()')[0]
 assert.doesNotMatch(create,/if\(this\.online\?\.rooms\.needsReconciliation[^\n]*renderScreen/)
 assert.match(create,/this\.notice\('建房失败：'/)
})
