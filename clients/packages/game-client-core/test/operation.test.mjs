import {test} from 'node:test'
import assert from 'node:assert/strict'
import {OperationScope,OperationSlot} from '../dist/operation.js'
test('cancel releases the slot synchronously and old completion cannot release new owner',async()=>{
 const slot=new OperationSlot(),first=slot.begin()
 let resolve
 const pending=first.wait(()=>new Promise(ok=>{resolve=ok}))
 const rejected=assert.rejects(pending,/cancelled/)
 assert.equal(slot.begin(),null)
 assert.equal(slot.cancel(),true)
 const second=slot.begin();assert.ok(second)
 await rejected;resolve('late');first.finish()
 assert.equal(slot.begin(),null)
 second.finish();assert.ok(slot.begin())
})
test('dispatched writes cannot be called cancelled; disposal is local abandonment',async()=>{
 const slot=new OperationSlot(),scope=slot.begin()
 scope.markDispatched()
 assert.equal(slot.cancel(),false)
 assert.equal(slot.begin(),null)
 scope.finish();assert.ok(slot.begin())
 slot.abandon();assert.ok(slot.begin())
})
test('bounded waits ignore late success and failure, reject synchronous throws and release',async()=>{
 const scope=new OperationScope()
 let late
 await assert.rejects(scope.wait(()=>new Promise((_ok,fail)=>{late=fail}),2),/timed out/)
 late(Error('late'))
 await assert.rejects(scope.wait(()=>{throw Error('sync')}),/sync/)
 assert.equal(await scope.wait(async()=>42),42)
 scope.finish()
 await assert.rejects(scope.wait(async()=>0),/cancelled/)
})
