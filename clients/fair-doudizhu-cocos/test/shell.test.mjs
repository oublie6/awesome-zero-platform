import {test} from 'node:test'
import assert from 'node:assert/strict'
import {GameClientShell} from '../dist/game/GameClientShell.js'
const tick=()=>new Promise(r=>setImmediate(r))
function transport(){
 const tasks=[]
 return {tasks,connect(){return new Promise((resolve,reject)=>{
  const c={sent:[],messages:new Set(),closes:new Set(),closed:0,
   send:async function(raw){this.sent.push(JSON.parse(raw))},
   close(){this.closed++},onMessage(cb){this.messages.add(cb);return()=>this.messages.delete(cb)},onClose(cb){this.closes.add(cb);return()=>this.closes.delete(cb)}}
  tasks.push({c,resolve:()=>resolve(c),reject})
 })}}
}
const snapshot={handId:'hand-1',version:7,phase:'PLAYING',view:{cards:['3S']}}
test('destroy during connect cancels state changes and disposes listeners',async()=>{
 const notices=[],s=new GameClientShell(v=>notices.push(v));await s.initialize();const t=transport()
 const p=s.connect(t,'wss://example.org/ws','token');s.dispose();const count=notices.length;t.tasks[0].resolve();await p
 assert.equal(s.hand.status,'disconnected');assert.equal(notices.length,count);assert.equal(t.tasks[0].c.closed,1)
 s.events.receive(JSON.stringify({type:'doudizhu.error',payload:{v:'doudizhu-realtime-event-v1'}}));assert.equal(notices.length,count)
})
test('late success/failure from old attempts cannot pollute a newer connection',async()=>{
 for(const fail of [false,true]){
 const s=new GameClientShell(()=>{}),t=transport();await s.initialize()
 const first=s.connect(t,'wss://example.org/ws','token'),second=s.connect(t,'wss://example.org/ws','token')
 t.tasks[1].resolve();await second;assert.equal(s.hand.status,'synchronizing')
 if(fail)t.tasks[0].reject(Error('old failure'));else t.tasks[0].resolve()
 await first;assert.equal(s.hand.status,'synchronizing');s.dispose()
 }
})
test('real shell emits exact Go sync shape, preserves notModified and stops terminal actions',async()=>{
 const notices=[],s=new GameClientShell(v=>notices.push(v));await s.initialize();await s.initialize()
 const t=transport(),p=s.connect(t,'wss://example.org/ws','token');t.tasks[0].resolve();await p
 s.acceptVerifiedSnapshot(snapshot);assert.equal(s.hand.canSubmit,true)
 await s.syncHand('hand-1',7)
 assert.deepEqual(t.tasks[0].c.sent[0],{id:'sync-1',type:'doudizhu.hand.sync',payload:{v:'doudizhu-realtime-sync-v1',handId:'hand-1',knownVersion:7}})
 const receive=(type,payload)=>{for(const cb of t.tasks[0].c.messages)cb(JSON.stringify({type,payload}))}
 receive('doudizhu.hand.evidence',{v:'doudizhu-realtime-event-v1',handId:'unrelated',evidence:{}});assert.equal(s.hand.canSubmit,true)
 const before=notices.length
 receive('doudizhu.hand.changed',{v:'doudizhu-realtime-event-v1',handId:'hand-1'})
 assert.equal(notices.length,before+1);assert.equal(s.hand.canSubmit,false)
 receive('doudizhu.hand.snapshot',{v:'doudizhu-realtime-event-v1',handId:'hand-1',notModified:true,knownVersion:6});assert.equal(s.hand.canSubmit,false)
 receive('doudizhu.hand.snapshot',{v:'doudizhu-realtime-event-v1',handId:'hand-1',notModified:true,knownVersion:7});assert.equal(s.hand.canSubmit,true);assert.deepEqual(s.hand.snapshot,snapshot)
 receive('doudizhu.hand.evidence',{v:'doudizhu-realtime-event-v1',handId:'hand-1',evidence:{}});assert.equal(s.hand.canSubmit,false)
 s.dispose();assert.equal(t.tasks[0].c.messages.size,0);assert.equal(t.tasks[0].c.closes.size,0)
 await assert.rejects(s.syncHand('hand-1'),/not connected/)
})
test('a wire view is never cast into local authoritative state without mapping',async()=>{
 const s=new GameClientShell(()=>{});await s.initialize()
 s.events.receive(JSON.stringify({type:'doudizhu.hand.snapshot',payload:{v:'doudizhu-realtime-event-v1',handId:'hand-1',notModified:false,view:{version:7,payload:{cards:['3S']}}}}))
 assert.equal(s.hand.snapshot,null);assert.equal(s.hand.canSubmit,false);s.dispose()
})
test('same-version full snapshot confirms reconnect without replacing confirmed state',async()=>{
 const s=new GameClientShell(()=>{}),t=transport();await s.initialize()
 let p=s.connect(t,'wss://example.org/ws','token');t.tasks[0].resolve();await p;s.acceptVerifiedSnapshot(snapshot)
 p=s.connect(t,'wss://example.org/ws','token');t.tasks[1].resolve();await p
 assert.equal(s.hand.status,'synchronizing')
 s.acceptVerifiedSnapshot({...snapshot,view:{cards:['duplicate must not overwrite']}})
 assert.equal(s.hand.status,'connected');assert.deepEqual(s.hand.snapshot.view,snapshot.view)
 s.hand.setStatus('synchronizing');s.acceptVerifiedSnapshot({...snapshot,version:6});assert.equal(s.hand.status,'synchronizing')
 s.dispose()
})

test('real Go private hand snapshot is applied after strict version and identity checks',async()=>{
 const s=new GameClientShell(()=>{}),t=transport();await s.initialize()
 const pending=s.connect(t,'wss://example.org/ws','token');t.tasks[0].resolve();await pending
 const wire={v:'doudizhu-live-private-view-v1',position:1,cards:['S3'],
   public:{v:'doudizhu-live-public-view-v1',handId:'hand-1',phase:'BIDDING',stateVersion:2,
     bidding:{},seats:[{position:1,accountId:'a',remainingCards:17},{position:2,accountId:'b',remainingCards:17},{position:3,accountId:'c',remainingCards:17}]}}
 const deliver=(view)=>{for(const cb of t.tasks[0].c.messages)cb(JSON.stringify({type:'doudizhu.hand.snapshot',payload:{v:'doudizhu-realtime-event-v1',handId:'hand-1',view}}))}
 deliver({version:2,payload:wire})
 assert.equal(s.hand.status,'connected');assert.equal(s.hand.snapshot?.version,2);assert.deepEqual(s.hand.snapshot?.view.cards,['S3'])
 deliver({version:3,payload:wire})
 assert.equal(s.hand.snapshot?.version,2)
 s.dispose()
})

test('real WSS command response matches request and disconnect leaves outcome unknown',async()=>{
 const s=new GameClientShell(()=>{}),t=transport();await s.initialize()
 const connected=s.connect(t,'wss://example.org/ws','token');t.tasks[0].resolve();await connected
 s.acceptVerifiedSnapshot(snapshot)
 const req={requestId:'move-1',type:'hand.pass',aggregateId:'hand-1',clientSeq:1,expectedVersion:7}
 const pending=s.submitCommand(req)
 await tick()
 assert.deepEqual(t.tasks[0].c.sent[0],{id:'move-1',type:'doudizhu.command',payload:{v:'doudizhu-api-command-v1',...req,payload:{}}})
 const deliver=payload=>{for(const cb of t.tasks[0].c.messages)cb(JSON.stringify({type:'doudizhu.command.result',payload}))}
 deliver({v:'doudizhu-api-command-result-v1',requestId:'move-1',type:'hand.pass',live:{version:8,payload:{},requiresTermination:false}})
 assert.equal((await pending).live.version,8)
 const pending2=s.submitCommand({...req,requestId:'move-2'})
 await tick()
 for(const cb of t.tasks[0].c.closes)cb()
 await assert.rejects(pending2,/outcome unknown/)
 s.dispose()
})

test('P1 old send failure does not erase same-id command on replacement connection',async()=>{
 const s=new GameClientShell(()=>{}),t=transport();await s.initialize()
 const first=s.connect(t,'wss://example.org/ws','token');t.tasks[0].resolve();await first
 s.acceptVerifiedSnapshot(snapshot)
 let failOld
 t.tasks[0].c.send=()=>new Promise((_,reject)=>{failOld=reject})
 const cmd={requestId:'same-id',type:'hand.pass',aggregateId:'hand-1',clientSeq:1,expectedVersion:7}
 const old=s.submitCommand(cmd);await tick()
 const second=s.connect(t,'wss://example.org/ws','token');await assert.rejects(old,/outcome unknown/)
 t.tasks[1].resolve();await second;s.acceptVerifiedSnapshot(snapshot)
 const fresh=s.submitCommand(cmd);await tick()
 failOld(Error('old socket rejected'));await tick()
 for(const callback of t.tasks[1].c.messages) callback(JSON.stringify({type:'doudizhu.command.result',payload:{v:'doudizhu-api-command-result-v1',requestId:'same-id',type:'hand.pass',live:{version:8,payload:{},requiresTermination:false}}}))
 assert.equal((await fresh).live.version,8)
 s.dispose()
})
test('P2 server rejection settles matching request without waiting for timeout',async()=>{
 const s=new GameClientShell(()=>{}),t=transport();await s.initialize()
 const start=s.connect(t,'wss://example.org/ws','token');t.tasks[0].resolve();await start;s.acceptVerifiedSnapshot(snapshot)
 const cmd={requestId:'rejected',type:'hand.pass',aggregateId:'hand-1',clientSeq:1,expectedVersion:7}
 const pending=s.submitCommand(cmd);await tick()
 for(const cb of t.tasks[0].c.messages)cb(JSON.stringify({type:'doudizhu.error',payload:{v:'doudizhu-realtime-event-v1',requestId:'rejected',code:'CONFLICT',message:'Doudizhu state conflict'}}))
 await assert.rejects(pending,/server rejected command: CONFLICT/)
 s.dispose()
})
test('P2 terminal hand blocks any hand mutation while room commands remain separately controlled',async()=>{
 const s=new GameClientShell(()=>{}),t=transport();await s.initialize()
 const start=s.connect(t,'wss://example.org/ws','token');t.tasks[0].resolve();await start
 s.acceptVerifiedSnapshot({...snapshot,phase:'COMPLETED'})
 await assert.rejects(s.submitCommand({requestId:'end',type:'hand.pass',aggregateId:'hand-1',clientSeq:1,expectedVersion:7}),/not eligible/)
 assert.equal(t.tasks[0].c.sent.length,0)
 s.dispose()
})
test('P2 outstanding commands have finite capacity',async()=>{
 const s=new GameClientShell(()=>{}),t=transport();await s.initialize()
 const start=s.connect(t,'wss://example.org/ws','token');t.tasks[0].resolve();await start
 s.acceptVerifiedSnapshot(snapshot)
 const pendings=[]
 for(let i=0;i<32;i++) pendings.push(s.submitCommand({requestId:'cap-'+i,type:'hand.pass',aggregateId:'hand-1',clientSeq:i+1,expectedVersion:7}))
 await assert.rejects(s.submitCommand({requestId:'overflow',type:'hand.pass',aggregateId:'hand-1',clientSeq:33,expectedVersion:7}),/capacity/)
 assert.equal(t.tasks[0].c.sent.length,32)
 s.dispose()
 await Promise.all(pendings.map(p=>assert.rejects(p,/outcome unknown/)))
})

test('R1 malformed empty error code cannot consume a valid pending operation',async()=>{
 const s=new GameClientShell(()=>{}),t=transport();await s.initialize()
 const opening=s.connect(t,'wss://example.org/ws','token');t.tasks[0].resolve();await opening
 s.acceptVerifiedSnapshot(snapshot)
 const input={requestId:'r1',type:'hand.pass',aggregateId:'hand-1',clientSeq:1,expectedVersion:7}
 const pending=s.submitCommand(input);await tick()
 const deliver=(type,payload)=>{for(const cb of t.tasks[0].c.messages)cb(JSON.stringify({type,payload}))}
 deliver('doudizhu.error',{v:'doudizhu-realtime-event-v1',requestId:'r1',code:'',message:'invalid'})
 deliver('doudizhu.command.result',{v:'doudizhu-api-command-result-v1',requestId:'r1',type:'hand.pass',live:{version:8,payload:{},requiresTermination:false}})
 assert.equal((await pending).live.version,8)
 s.dispose()
})
test('R2 fresh lobby can create room without any hand snapshot',async()=>{
 const s=new GameClientShell(()=>{}),t=transport();await s.initialize()
 const opening=s.connect(t,'wss://example.org/ws','token');t.tasks[0].resolve();await opening
 assert.equal(s.hand.snapshot,null);assert.equal(s.hand.status,'synchronizing')
 const pending=s.submitCommand({requestId:'room-first',type:'room.create',aggregateId:'room-new',clientSeq:1,expectedVersion:0})
 await tick()
 assert.equal(t.tasks[0].c.sent.length,1)
 assert.equal(t.tasks[0].c.sent[0].payload.type,'room.create')
 for(const cb of t.tasks[0].c.messages)cb(JSON.stringify({type:'doudizhu.command.result',payload:{v:'doudizhu-api-command-result-v1',requestId:'room-first',type:'room.create',durable:{accepted:true}}}))
 assert.equal((await pending).durable.accepted,true)
 await assert.rejects(s.submitCommand({requestId:'illegal-hand',type:'hand.pass',aggregateId:'hand-1',clientSeq:2,expectedVersion:7}),/not eligible/)
 s.dispose()
})
test('R2 socket not ready or disconnected still refuses lobby commands',async()=>{
 const s=new GameClientShell(()=>{}),t=transport();await s.initialize()
 const input={requestId:'early',type:'room.create',aggregateId:'room-new',clientSeq:1,expectedVersion:0}
 await assert.rejects(s.submitCommand(input),/not ready/)
 const opening=s.connect(t,'wss://example.org/ws','token')
 await assert.rejects(s.submitCommand(input),/not ready/)
 t.tasks[0].resolve();await opening
 for(const cb of t.tasks[0].c.closes)cb()
 await assert.rejects(s.submitCommand(input),/not ready/)
 s.dispose()
})

test('authenticated archive evidence changes terminal UI state without forging live cards',async()=>{
  const s=new GameClientShell(()=>{}),t=transport();await s.initialize()
  const opening=s.connect(t,'wss://example.org/ws','token');t.tasks[0].resolve();await opening
  s.acceptVerifiedSnapshot(snapshot)
  const evidence={
    handId:'hand-1',status:'completed',finalVersion:8,
    verification:{handId:'hand-1',finalVersion:8,recordDigest:'a'.repeat(64),seatPoints:[6,-3,-3],winnerSeat:1},
  }
  s.acceptVerifiedFinalEvidence(evidence)
  assert.equal(s.hand.snapshot.phase,'COMPLETED')
  assert.equal(s.hand.snapshot.version,8)
  assert.equal(s.hand.canSubmit,false)
  assert.equal(s.hand.snapshot.view.terminalEvidence.recordDigest,'a'.repeat(64))
  assert.equal(s.hand.snapshot.view.cards,undefined)
  await assert.rejects(s.submitCommand({requestId:'end',type:'hand.pass',aggregateId:'hand-1',clientSeq:1,expectedVersion:8}),/not eligible/)
  assert.throws(()=>s.acceptVerifiedFinalEvidence({...evidence,finalVersion:7,verification:{...evidence.verification,finalVersion:7}}),/stale/)
  assert.throws(()=>s.acceptVerifiedFinalEvidence({...evidence,handId:'other-hand',verification:{...evidence.verification,handId:'other-hand'}}),/mismatch/)
  s.clearHand()
  assert.equal(s.hand.snapshot,null)
  assert.equal(s.hand.status,'connected')
  s.dispose()
})

test('Round9 explicitly bound second hand rejects old finished snapshots and stale evidence',async()=>{
 const s=new GameClientShell(()=>{}),t=transport()
 await s.initialize()
 const opening=s.connect(t,'wss://example.org/ws','token');t.tasks[0].resolve();await opening
 s.useAuthoritativeHand('hand-1')
 s.acceptVerifiedSnapshot(snapshot)
 s.useAuthoritativeHand('hand-2')
 assert.equal(s.hand.snapshot,null)
 assert.equal(s.hand.status,'connected')
 assert.throws(()=>s.acceptVerifiedSnapshot({...snapshot,version:10}),/outside authorized hand/)
 const second={handId:'hand-2',version:2,phase:'BIDDING',view:{cards:['5H']}}
 s.acceptVerifiedSnapshot(second)
 assert.equal(s.hand.snapshot.handId,'hand-2')
 const evidence={handId:'hand-1',status:'completed',finalVersion:10,verification:{
  handId:'hand-1',finalVersion:10,recordDigest:'a'.repeat(64),seatPoints:[6,-3,-3],winnerSeat:1}}
 assert.throws(()=>s.acceptVerifiedFinalEvidence(evidence),/outside authorized hand/)
 s.clearHand()
 assert.equal(s.hand.snapshot,null)
 assert.throws(()=>s.acceptVerifiedFinalEvidence(evidence),/outside authorized hand/)
 assert.throws(()=>s.acceptVerifiedSnapshot(second),/outside authorized hand/)
 s.useAuthoritativeHand('hand-3')
 s.acceptVerifiedSnapshot({handId:'hand-3',version:2,phase:'PLAYING',view:{cards:['K']}})
 assert.equal(s.hand.snapshot.handId,'hand-3')
 s.dispose()
})

test('audit: buffered messages from a replaced socket cannot mutate the current hand or notices',async()=>{
 const notices=[],shell=new GameClientShell(message=>notices.push(message))
 await shell.initialize()
 const t=transport()
 const first=shell.connect(t,'wss://example.org/ws','token')
 t.tasks[0].resolve();await first
 shell.acceptVerifiedSnapshot(snapshot)
 const oldMessage=[...t.tasks[0].c.messages][0]
 assert.equal(typeof oldMessage,'function')
 const second=shell.connect(t,'wss://example.org/ws','token')
 t.tasks[1].resolve();await second
 shell.acceptVerifiedSnapshot(snapshot)
 assert.equal(shell.hand.status,'connected')
 const priorNotices=notices.length
 oldMessage(JSON.stringify({type:'doudizhu.hand.changed',payload:{
  v:'doudizhu-realtime-event-v1',handId:'hand-1'}}))
 oldMessage(JSON.stringify({type:'doudizhu.error',payload:{
  v:'doudizhu-realtime-event-v1',code:'INTERNAL',message:'obsolete socket'}}))
 assert.equal(shell.hand.status,'connected','obsolete socket must not force syncing')
 assert.equal(notices.length,priorNotices,'obsolete socket must not notify current session')
 shell.dispose()
})
