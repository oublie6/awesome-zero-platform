import {test} from 'node:test'
import assert from 'node:assert/strict'
import {GameHandActions} from '../dist/game/GameHandActions.js'
function fixture(phase='PLAYING') {
 const commands=[]
 const snapshot={handId:'h',version:7,phase,view:{}}
 const shell={hand:{snapshot,canSubmit:true},async submitCommand(c){commands.push(c);return {v:'doudizhu-api-command-result-v1',requestId:c.requestId,type:c.type,live:{version:8,payload:{},requiresTermination:false}}}}
 let counter=0
 const random={async bytes(n){return Uint8Array.from({length:n},(_,i)=>(i+counter++)%256)}}
 return {shell,commands,actions:new GameHandActions(shell,random)}
}
test('real hand actions use authoritative ID/version and cryptographic request IDs',async()=>{
 const f=fixture()
 await f.actions.play(['S3','YJ'])
 assert.equal(f.commands[0].type,'hand.play')
 assert.equal(f.commands[0].expectedVersion,7)
 assert.equal(f.commands[0].aggregateId,'h')
 assert.deepEqual(f.commands[0].payload,{cards:['S3','YJ']})
 assert.match(f.commands[0].requestId,/^cmd-[a-f0-9]{32}$/)
})
test('reject malformed cards, incorrect phase, disconnected state and duplicate selection',async()=>{
 const f=fixture()
 await assert.rejects(f.actions.play(['S3','S3']),/invalid card selection/)
 await assert.rejects(f.actions.play(['ZZ']),/invalid card selection/)
 await assert.rejects(f.actions.bid(3),/not ready/)
 f.shell.hand.canSubmit=false
 await assert.rejects(f.actions.pass(),/not ready/)
 assert.equal(f.commands.length,0)
})
test('bid requires bidding phase and rejects invalid score',async()=>{
 const f=fixture('BIDDING')
 await f.actions.bid(3)
 assert.equal(f.commands[0].type,'hand.bid')
 assert.equal(f.commands[0].payload.score,3)
 await assert.rejects(f.actions.bid(4),/invalid bid score/)
})
test('snapshot changes during entropy acquisition invalidate action',async()=>{
 const f=fixture()
 f.actions=new GameHandActions(f.shell,{bytes:async(n)=>{f.shell.hand.snapshot={...f.shell.hand.snapshot,version:8};return new Uint8Array(n)}})
 await assert.rejects(f.actions.pass(),/hand changed/)
 assert.equal(f.commands.length,0)
})

test('unknown live action retains identical request across reconnect and blocks new commands',async()=>{
 const f=fixture('PLAYING')
 let attempted=0
 f.shell.submitCommand=async (command,retry)=>{
   f.commands.push({command:structuredClone(command),retry:!!retry})
   attempted++
   if(attempted===1)throw new Error('command timeout; outcome unknown')
   return {v:'doudizhu-api-command-result-v1',requestId:command.requestId,
     type:command.type,live:{version:8,payload:{},requiresTermination:false}}
 }
 await assert.rejects(f.actions.pass(),/outcome unknown/)
 assert.equal(f.actions.needsReconciliation,true)
 await assert.rejects(f.actions.play(['S3']),/previous hand command outcome unknown/)
 f.shell.hand.snapshot={...f.shell.hand.snapshot,version:8,phase:'PLAYING'}
 const result=await f.actions.retryUnknown()
 assert.equal(result.live.version,8)
 assert.deepEqual(f.commands[0].command,f.commands[1].command)
 assert.equal(f.commands[0].retry,false)
 assert.equal(f.commands[1].retry,true)
 assert.equal(f.actions.needsReconciliation,false)
 assert.equal(f.commands.length,2)
})
test('explicit server command rejection releases pending action but ambiguous failure does not',async()=>{
 const f=fixture()
 f.shell.submitCommand=async()=>{throw new Error('command outcome unknown')}
 await assert.rejects(f.actions.pass(),/outcome unknown/)
 assert.equal(f.actions.needsReconciliation,true)
 await assert.rejects(f.actions.retryUnknown(),/outcome unknown/)
 assert.equal(f.actions.needsReconciliation,true)
 // A fresh authoritative version is required to abandon rather than reissue.
 assert.throws(()=>f.actions.discardUnknownAfterSync(),/not confirmed/)
 f.shell.hand.snapshot={...f.shell.hand.snapshot,version:8}
 f.actions.discardUnknownAfterSync()
 assert.equal(f.actions.needsReconciliation,false)
})

test('Round15 live hand reservation rejects unknown and in-flight commands',async()=>{
 const f=fixture('PLAYING')
 let finish
 f.shell.submitCommand=()=>new Promise(resolve=>{finish=resolve})
 const passing=f.actions.pass()
 assert.throws(()=>f.actions.beginHandTransition(),/in progress/)
 await new Promise(resolve=>setImmediate(resolve))
 finish({v:'doudizhu-api-command-result-v1',requestId:'handled',type:'hand.pass'})
 await passing
 const release=f.actions.beginHandTransition()
 await assert.rejects(f.actions.pass(),/already in flight/)
 release()
 f.shell.submitCommand=async()=>{throw Error('lost response')}
 await assert.rejects(f.actions.pass(),/lost response/)
 assert.equal(f.actions.needsReconciliation,true)
 assert.throws(()=>f.actions.beginHandTransition(),/outcome unknown/)
 assert.equal(f.actions.needsReconciliation,true)
})

test('Round3 unknown bid/play/pass notifies presentation only after transport settles',async()=>{
 for(const [phase,method,arg] of [
   ['BIDDING','bid',1],['PLAYING','play',['S3']],['PLAYING','pass',undefined]
 ]){
   const f=fixture(phase)
   const events=[]
   let rejectWire
   f.actions=new GameHandActions(f.shell,{
     bytes:async(n)=>new Uint8Array(n).fill(1)
   },()=>events.push({
     unknown:f.actions.needsReconciliation,submitting:f.actions.isSubmitting
   }))
   f.shell.submitCommand=()=>new Promise((_,reject)=>{rejectWire=reject})
   const attempt=method==='pass'?f.actions.pass():f.actions[method](arg)
   await new Promise(resolve=>setImmediate(resolve))
   assert.equal(f.actions.isSubmitting,true,method)
   assert.equal(events[0].submitting,true,method)
   assert.equal(events[0].unknown,false,method)
   rejectWire(new Error('command timeout; outcome unknown'))
   await assert.rejects(attempt,/outcome unknown/)
   assert.equal(f.actions.needsReconciliation,true,method)
   assert.equal(f.actions.isSubmitting,false,method)
   assert.deepEqual(events.at(-1),{unknown:true,submitting:false},method)
   const original=f.commands[0]
   // A retry of UNKNOWN is the exact original command, never a fresh ID.
   f.shell.submitCommand=async (command,retry)=>{
     f.commands.push({command:structuredClone(command),retry:!!retry})
     return {v:'doudizhu-api-command-result-v1',requestId:command.requestId,type:command.type,live:{version:8,payload:{},requiresTermination:false}}
   }
   await f.actions.retryUnknown()
   assert.equal(f.actions.needsReconciliation,false)
   assert.equal(f.actions.isSubmitting,false)
   assert.deepEqual(events.at(-1),{unknown:false,submitting:false})
   assert.equal(f.commands.at(-1).retry,true)
   assert.equal(f.commands.at(-1).command.type,method==='bid'?'hand.bid':method==='play'?'hand.play':'hand.pass')
 }
})

test('Round3 explicit rejection and authorized discard notify to hide retry controls',async()=>{
 const f=fixture('PLAYING')
 const events=[]
 f.actions=new GameHandActions(f.shell,{bytes:async(n)=>new Uint8Array(n).fill(3)},
   ()=>events.push({unknown:f.actions.needsReconciliation,submitting:f.actions.isSubmitting}))
 f.shell.submitCommand=async()=>{throw new Error('lost response')}
 await assert.rejects(f.actions.pass(),/lost response/)
 assert.deepEqual(events.at(-1),{unknown:true,submitting:false})
 f.shell.hand.snapshot={...f.shell.hand.snapshot,version:8}
 f.actions.discardUnknownAfterSync()
 assert.deepEqual(events.at(-1),{unknown:false,submitting:false})
 assert.equal(f.actions.needsReconciliation,false)
})
