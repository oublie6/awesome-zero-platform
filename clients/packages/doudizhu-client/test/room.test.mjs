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
 assert.throws(()=>room.restoreTrustedRoom('r',1),/outcome unknown/)
 await room.retryUnknown()
 await room.ready(true)
 assert.equal(calls[2].clientSeq,2)
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

test('authoritative CLOSED clears only a confirmed room; never destroys unknown durable create',async()=>{
 const {room,api}=fixture()
 room.restoreTrustedRoom('room-valid',3,5)
 room.forgetTrustedRoom('unrelated')
 assert.equal(room.state.roomId,'room-valid')
 room.forgetTrustedRoom('room-valid')
 assert.equal(room.state,null)
 let rejectSend
 api.command=()=>new Promise((_,reject)=>{rejectSend=reject})
 const create=room.create('room-unknown')
 await new Promise(resolve=>setImmediate(resolve))
 assert.throws(()=>room.forgetTrustedRoom('room-unknown'),/outstanding|unresolved/)
 rejectSend(new Error('connection lost'))
 await assert.rejects(create,/connection lost/)
 assert.equal(room.needsReconciliation,true)
 assert.throws(()=>room.forgetTrustedRoom('room-unknown'),/unresolved/)
 assert.equal(room.pendingRoomId,'room-unknown')
})

test('G48-N02: recovering a different room cannot discard UNKNOWN request identity',async()=>{
 const {room,api,calls}=fixture()
 let fail=true
 const original=api.command
 api.command=async command=>{
  if(fail){calls.push(command);throw Error('lost ACK')}
  return original(command)
 }
 await assert.rejects(room.join('A',3),/lost ACK/)
 const originalRequest=calls[0]
 assert.equal(room.pendingRoomId,'A')
 assert.equal(room.pendingCommandType,'room.join')
 assert.throws(()=>room.restoreTrustedRoom('B',5),/outcome unknown/)
 assert.equal(room.state,null)
 assert.equal(room.pendingRoomId,'A')
 assert.equal(room.needsReconciliation,true)
 fail=false
 await room.retryUnknown()
 assert.deepEqual(calls[1],originalRequest)
 assert.equal(room.needsReconciliation,false)
 room.restoreTrustedRoom('B',5)
 assert.equal(room.state.roomId,'B')
})

test('G49-N01: same-room UNKNOWN survives old authorized room reads until exact retry',async()=>{
 const {room,api,calls}=fixture()
 room.restoreTrustedRoom('A',1,1)
 const before=room.commandGeneration
 let fail=true
 const original=api.command
 api.command=async cmd=>{if(fail){calls.push(cmd);throw Error('ACK lost')}return original(cmd)}
 await assert.rejects(room.ready(true),/ACK lost/)
 const originalCmd=calls[0]
 assert.ok(room.commandGeneration>before)
 assert.throws(()=>room.restoreTrustedRoom('A',1,1),/outcome unknown/)
 assert.throws(()=>room.restoreTrustedRoom('B',1,1),/outcome unknown/)
 assert.equal(room.needsReconciliation,true)
 assert.equal(room.pendingRoomId,'A')
 fail=false
 await room.retryUnknown()
 assert.deepEqual(calls[1],originalCmd)
 room.restoreTrustedRoom('A',2,2)
 assert.equal(room.state.version,2)
})

test('G50-N01: an older authorized read cannot roll back a confirmed room version',async()=>{
 const {room}=fixture()
 room.restoreTrustedRoom('A',1,1)
 const initial=room.commandGeneration
 await room.ready(true)
 assert.ok(room.commandGeneration>=initial+2)
 assert.deepEqual(room.state,{roomId:'A',version:2})
 assert.throws(()=>room.restoreTrustedRoom('A',1,1),/stale room version/)
 assert.equal(room.state.version,2)
 room.restoreTrustedRoom('A',3,3)
 assert.equal(room.state.version,3)
})
test('audit: rejected join must not poison a different room command sequence',async()=>{
 const {room,calls}=fixture()
 room.restoreTrustedRoom('A',3,2)
 await assert.rejects(room.join('B',3,800),/room identity mismatch/)
 await assert.rejects(room.join('A',0,900),/invalid room version/)
 await room.ready(true)
 assert.equal(calls.at(-1).clientSeq,3)
 assert.equal(room.state.version,4)
})
test('audit: command generation advances after definite rejection and exact UNKNOWN retry',async()=>{
 const {room,api}=fixture()
 room.restoreTrustedRoom('A',3,1)
 const accepted=api.command
 api.command=async c=>({v:'doudizhu-api-command-result-v1',requestId:c.requestId,type:c.type,
  durable:{v:'fair-doudizhu-command-result-v1',commandId:c.requestId,aggregateId:'A',aggregateVersion:3,
   accepted:false,failure:{code:'CONFLICT',currentVersion:4}}})
 const before=room.commandGeneration
 await assert.rejects(room.ready(true),/CONFLICT/)
 assert.equal(room.commandGeneration,before+2)
 api.command=accepted
 await room.ready(true)
 assert.equal(room.state.version,4)
})
