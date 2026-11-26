import {test} from 'node:test'
import assert from 'node:assert/strict'
import {OperationScope,OperationSlot,runCleanup} from '../dist/operation.js'
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

test('G65 R11: throwing release and listeners never delay other waits or cleanup',async()=>{
 for(const method of ['cancel','abandon','finish']){
  const order=[]
  const scope=new OperationScope(()=>{order.push('release');throw Error('private detail')})
  scope.onCancel(()=>{order.push('first');throw Error('private detail')})
  const p=scope.wait(()=>new Promise(()=>{}),5000)
  const q=scope.wait(()=>new Promise(()=>{}),5000)
  const rejected=Promise.all([assert.rejects(p,/cancelled/),assert.rejects(q,/cancelled/)])
  scope.onCancel(()=>order.push('last'))
  scope[method]()
  assert.deepEqual(order,['release','first','last'])
  assert.equal(scope.cleanupFailures,2)
  assert.equal(scope.active,false)
  await rejected
  scope.finish();scope.abandon();scope.cancel()
  assert.equal(scope.cleanupFailures,2)
  scope.onCancel(()=>{throw Error('late fault')})
  assert.equal(scope.cleanupFailures,3)
 }
})
test('cleanup diagnostic failure is isolated and each step is attempted in order',()=>{
 const order=[]
 const count=runCleanup([()=>{order.push(1);throw Error('fault')},()=>order.push(2)],()=>{throw Error('reporter fault')})
 assert.deepEqual(order,[1,2]);assert.equal(count,1)
})
