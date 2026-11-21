import {test} from 'node:test'
import assert from 'node:assert/strict'
import {acceptTurnTime,remainingSeconds} from '../dist/bootstrap/TurnClock.js'
test('countdown uses trusted server time, never local wall clock',()=>{
 const t={serverTimeMillis:1_700_000_000_000,turnDeadlineMillis:1_700_000_020_000}
 assert.deepEqual(acceptTurnTime(t),t)
 assert.equal(remainingSeconds(t,0),20)
 assert.equal(remainingSeconds(t,1001),19)
 assert.equal(remainingSeconds(t,19000),1)
 assert.equal(remainingSeconds(t,20000),0)
 assert.equal(remainingSeconds(t,1e9),0)
})
test('missing or unreasonable time is not a fabricated countdown',()=>{
 assert.equal(remainingSeconds({},0),null)
 assert.equal(remainingSeconds({turnDeadlineMillis:200},0),null)
 assert.equal(remainingSeconds({serverTimeMillis:200},0),null)
 assert.equal(remainingSeconds({turnDeadlineMillis:1,serverTimeMillis:1e12},0),null)
 assert.equal(remainingSeconds({serverTimeMillis:1e12,turnDeadlineMillis:1e12+24*3600*1000+1},0),null)
 assert.equal(remainingSeconds({serverTimeMillis:1e12,turnDeadlineMillis:1e12+30000},NaN),null)
})

test('table preserves countdown origin when selecting cards without a new snapshot',async()=>{
 const {readFile}=await import('node:fs/promises')
 const source=await readFile(new URL('../assets/scripts/bootstrap/GameTableView.ts',import.meta.url),'utf8')
 assert.match(source,/this\.turnClock\.observe\(data\.handID,data\.version/)
 assert.match(source,/this\.turnClock\.remainingSeconds\(\)/)
 assert.match(source,/clearInterval\(this\.tickHandle\)/)
})
