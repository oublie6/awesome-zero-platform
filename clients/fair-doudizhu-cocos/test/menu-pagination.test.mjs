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
test('17 cards and game commands are reachable without compressing full hand into tiny labels',()=>{
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
