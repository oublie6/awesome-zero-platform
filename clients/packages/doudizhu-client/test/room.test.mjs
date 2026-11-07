import {test} from 'node:test'
import assert from 'node:assert/strict'
import {RoomCoordinator,DurableRoomRejected,secureRoomCommandIdentity} from '../dist/room.js'

function fixture() {
 let n=0
 const calls=[]
 const api={async command(cmd){
   calls.push(cmd)
   return {v:'doudizhu-api-command-result-v1',requestId:cmd.requestId,type:cmd.type,
     durable:{v:'fair-doudizhu-command-result-v1',commandId:cmd.requestId,accepted:true,aggregateId:cmd.aggregateId,aggregateVersion:cmd.expectedVersion+1}}
 }}
 return {calls,api,room:new RoomCoordinator(api,{newRequestId:()=> 'secure-id-'+(++n)})}
}
test('create, ready and start use actual Go DTO and advance only on confirmed durable results',async()=>{
 const {room,calls}=fixture()
 assert.deepEqual(await room.create('room-one'),{roomId:'room-one',version:1})
 assert.deepEqual(await room.ready(true),{roomId:'room-one',version:2})
 assert.deepEqual(await room.startHand('hand-new'),{roomId:'room-one',version:3})
 assert.deepEqual(calls.map(c=>[c.type,c.clientSeq,c.expectedVersion]),[
  ['room.create',1,0],['room.ready',2,1],['room.hand.start',3,2]])
 assert.deepEqual(calls[1].payload,{ready:true})
 assert.deepEqual(calls[2].payload,{handId:'hand-new'})
})
test('join from trusted version and do not expose mutable room state',async()=>{
 const {room,calls}=fixture()
 await room.join('room-one',5)
 assert.deepEqual(room.state,{roomId:'room-one',version:6})
 const old=room.state
 assert.throws(()=>{old.version=99},TypeError)
 await room.leave()
 assert.equal(calls[1].expectedVersion,6)
 assert.equal(room.state,null)
})
test('explicit durable rejection does not mutate confirmed version',async()=>{
 const {room,api}=fixture()
 room.restoreTrustedRoom('r',3)
 api.command=async c=>({v:'doudizhu-api-command-result-v1',requestId:c.requestId,type:c.type,durable:{
  v:'fair-doudizhu-command-result-v1',commandId:c.requestId,aggregateId:'r',aggregateVersion:3,
  accepted:false,failure:{code:'CONFLICT',currentVersion:5}}})
 await assert.rejects(room.ready(true),e=>e instanceof DurableRoomRejected && e.code==='CONFLICT' && e.currentVersion===5)
 assert.equal(room.state.version,3)
})
test('unknown transport outcome is not converted to confirmed state and sequence is not reused',async()=>{
 const {room,api,calls}=fixture()
 let fail=true
 const base=api.command
 api.command=async cmd=>{if(fail){calls.push(cmd);throw Error('network timeout')}return base(cmd)}
 await assert.rejects(room.create('r'),/network timeout/)
 assert.equal(room.state,null)
 fail=false
 await assert.rejects(room.create('r'),/reconcile/)
 room.restoreTrustedRoom('r',1)
 await room.ready(true)
 assert.equal(calls[1].clientSeq,2)
 assert.equal(room.state.version,2)
})
test('reject mismatched durable confirmation',async()=>{
 const {room,api}=fixture()
 api.command=async c=>({requestId:c.requestId,type:c.type,durable:{
 v:'fair-doudizhu-command-result-v1',commandId:'another-command',aggregateId:c.aggregateId,
 aggregateVersion:1,accepted:true}})
 await assert.rejects(room.create('r'),/invalid authoritative room result/)
 assert.equal(room.state,null)
})
test('one in-flight room operation per coordinator',async()=>{
 const {room,api}=fixture()
 let resolve
 api.command=()=>new Promise(r=>{resolve=r})
 const first=room.create('r')
 await assert.rejects(room.create('r'),/outstanding/)
 resolve({v:'doudizhu-api-command-result-v1',requestId:'secure-id-1',type:'room.create',durable:{
  v:'fair-doudizhu-command-result-v1',commandId:'secure-id-1',aggregateId:'r',aggregateVersion:1,accepted:true}})
 await first
})

test('secure IDs use injected entropy, reject missing or short entropy without a fake fallback',async()=>{
 let calls=0
 const identity=secureRoomCommandIdentity({bytes:async n=>{calls++;assert.equal(n,16);return new Uint8Array(Array.from({length:n},(_,i)=>i))}})
 assert.equal(await identity.newRequestId(),'cmd-000102030405060708090a0b0c0d0e0f')
 assert.equal(calls,1)
 await assert.rejects(secureRoomCommandIdentity({bytes:async()=>new Uint8Array(8)}).newRequestId(),/entropy unavailable/)
})
test('room coordinator supports asynchronous secure identities and validates top-level envelope',async()=>{
 const {room,api}=fixture()
 let calls=0
 const secure=new RoomCoordinator(api,{newRequestId:async()=>{calls++;return 'async-secure-id'}})
 await secure.create('room-async')
 assert.equal(calls,1)
 api.command=async c=>({v:'wrong-version',requestId:c.requestId,type:c.type,durable:{
  v:'fair-doudizhu-command-result-v1',commandId:c.requestId,aggregateId:c.aggregateId,aggregateVersion:1,accepted:true}})
 await assert.rejects(room.create('room-invalid'),/invalid authoritative room result/)
 assert.equal(room.needsReconciliation,true)
})

test('lost room.create acknowledgment preserves the exact request ID and locator',async()=>{
 const {room,api,calls}=fixture()
 let first=true
 const original=api.command.bind(api)
 api.command=async command=>{
   if(first){first=false;calls.push(command);throw Error('lost committed reply')}
   return original(command)
 }
 await assert.rejects(room.create('room-lost'),/lost committed reply/)
 assert.equal(room.pendingRoomId,'room-lost')
 assert.equal(room.needsReconciliation,true)
 await assert.rejects(room.create('room-else'),/reconcile/)
 await room.retryUnknown()
 assert.equal(room.pendingRoomId,null)
 assert.deepEqual(calls[0],calls[1])
 assert.equal(room.state.roomId,'room-lost')
 assert.equal(room.state.version,1)
})

test('new custodial hand start keeps mode inside exact durable command identity',async()=>{
 const {room,calls}=fixture()
 await room.create('room-one')
 await room.startHand('hand-new','client-contributed-server-custodied-v1')
 assert.deepEqual(calls[1].payload,{
   handId:'hand-new',fairnessMode:'client-contributed-server-custodied-v1'
 })
})
