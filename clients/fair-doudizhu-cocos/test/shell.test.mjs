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
 const syncing=s.syncHand('hand-1',7)
 assert.deepEqual(t.tasks[0].c.sent[0],{id:'sync-1',type:'doudizhu.hand.sync',payload:{v:'doudizhu-realtime-sync-v1',handId:'hand-1',knownVersion:7}})
 const receive=(type,payload,id)=>{for(const cb of t.tasks[0].c.messages)cb(JSON.stringify({type,payload,id}))}
 assert.equal(s.hand.canSubmit,false)
 receive('doudizhu.hand.snapshot',{v:'doudizhu-realtime-event-v1',handId:'hand-1',notModified:true,knownVersion:7},'sync-1')
 await syncing
 receive('doudizhu.hand.evidence',{v:'doudizhu-realtime-event-v1',handId:'unrelated',evidence:{}});assert.equal(s.hand.canSubmit,true)
 const before=notices.length
 receive('doudizhu.hand.changed',{v:'doudizhu-realtime-event-v1',handId:'hand-1'})
 assert.equal(notices.length,before+1);assert.equal(s.hand.canSubmit,false)
 receive('doudizhu.hand.snapshot',{v:'doudizhu-realtime-event-v1',handId:'hand-1',notModified:true,knownVersion:6});assert.equal(s.hand.canSubmit,false)
 const next=s.syncHand('hand-1',7)
 receive('doudizhu.hand.snapshot',{v:'doudizhu-realtime-event-v1',handId:'hand-1',notModified:true,knownVersion:7});assert.equal(s.hand.canSubmit,false)
 receive('doudizhu.hand.snapshot',{v:'doudizhu-realtime-event-v1',handId:'hand-1',notModified:true,knownVersion:7},'sync-2')
 await next
 assert.equal(s.hand.canSubmit,true);assert.deepEqual(s.hand.snapshot,snapshot)
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
 assert.equal(s.hand.canSubmit,false,'ACK does not supply a confirmed new private projection')
 await assert.rejects(s.submitCommand({...req,requestId:'early-move'}),/not eligible/)
 s.acceptVerifiedSnapshot({...snapshot,version:8})
 const pending2=s.submitCommand({...req,requestId:'move-2',expectedVersion:8})
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

function viewReply(handId='hand-1',version=2){
 return {v:'doudizhu-realtime-event-v1',handId,notModified:false,view:{version,payload:{
  v:'doudizhu-live-private-view-v1',position:1,cards:['S3'],
  public:{v:'doudizhu-live-public-view-v1',handId,phase:'BIDDING',stateVersion:version,
    bidding:{},seats:[{position:1,accountId:'a',remainingCards:17},{position:2,accountId:'b',remainingCards:17},{position:3,accountId:'c',remainingCards:17}]}}}}
}
async function syncFixture(timeout=1000){
 const s=new GameClientShell(()=>{},timeout),t=transport()
 await s.initialize()
 const open=s.connect(t,'wss://example.org/ws','token');t.tasks[0].resolve();await open
 s.useAuthoritativeHand('hand-1')
 const c=t.tasks[0].c
 const deliver=(type,payload,id)=>{for(const cb of c.messages)cb(JSON.stringify({type,payload,id}))}
 return {s,t,c,deliver}
}
test('G65 R5: send is not completion; delayed correlated private response is required',async()=>{
 const {s,c,deliver}=await syncFixture()
 const first=s.syncHand('hand-1'),duplicate=s.syncHand('hand-1')
 assert.strictEqual(first,duplicate);assert.equal(c.sent.length,1)
 let completed=false;void first.then(()=>{completed=true})
 await tick();assert.equal(completed,false);assert.equal(s.hand.snapshot,null)
 deliver('doudizhu.hand.snapshot',viewReply())
 await tick();assert.equal(completed,false,'broadcast is not correlated read completion')
 assert.equal(s.hand.canSubmit,false)
 deliver('doudizhu.hand.snapshot',viewReply(),'sync-1')
 await first;assert.equal(s.hand.snapshot.version,2);assert.equal(s.hand.status,'connected')
 s.dispose()
})
test('G65 R5: dropped or stalled send times out and late response cannot resurrect recovery',async()=>{
 const {s,c,deliver}=await syncFixture(20)
 c.send=()=>new Promise(()=>{})
 await assert.rejects(s.syncHand('hand-1'),/timed out/)
 assert.equal(s.hand.status,'synchronizing');assert.equal(s.hand.snapshot,null)
 deliver('doudizhu.hand.snapshot',viewReply(),'sync-1')
 assert.equal(s.hand.snapshot,null)
 c.send=async()=>{}
 const retry=s.syncHand('hand-1')
 deliver('doudizhu.hand.snapshot',viewReply(),'sync-2')
 await retry;assert.equal(s.hand.snapshot.handId,'hand-1')
 s.dispose()
})
test('G65 R5: malformed, mismatched, stale, forbidden and invalid conditional replies reject',async()=>{
 for(const fault of ['malformed','identity','stale','forbidden','notModified','router']){
  const {s,deliver}=await syncFixture()
  s.acceptVerifiedSnapshot({...snapshot,version:7})
  const p=s.syncHand('hand-1')
  const failed=assert.rejects(p,/sync|stale|notModified|malformed|identity|invalid Go view wrapper/)
  if(fault==='malformed')deliver('doudizhu.hand.snapshot',{v:'doudizhu-realtime-event-v1',handId:'hand-1',view:{}},'sync-1')
  if(fault==='identity')deliver('doudizhu.hand.snapshot',viewReply('other-hand',8),'sync-1')
  if(fault==='stale')deliver('doudizhu.hand.snapshot',viewReply('hand-1',6),'sync-1')
  if(fault==='forbidden')deliver('doudizhu.error',{v:'doudizhu-realtime-event-v1',requestId:'sync-1',code:'FORBIDDEN'},'sync-1')
  if(fault==='notModified')deliver('doudizhu.hand.snapshot',{v:'doudizhu-realtime-event-v1',handId:'hand-1',notModified:true,knownVersion:7},'sync-1')
  if(fault==='router')deliver('doudizhu.hand.snapshot',{v:'bad',handId:'hand-1'},'sync-1')
  await failed
  assert.equal(s.hand.snapshot.version,7);assert.equal(s.hand.canSubmit,false)
  s.dispose()
 }
})
test('G65 R5: disconnect, disposal, replacement, and hand switch cancel pending reads',async()=>{
 for(const mode of ['close','dispose','replace','switch']){
  const {s,t,c,deliver}=await syncFixture()
  const p=s.syncHand('hand-1'),failed=assert.rejects(p,/connection|disposed|changed/)
  if(mode==='close')for(const cb of c.closes)cb()
  if(mode==='dispose')s.dispose()
  if(mode==='replace'){
   const reopening=s.connect(t,'wss://example.org/ws','token')
   t.tasks[1].resolve();await reopening
  }
  if(mode==='switch')s.useAuthoritativeHand('hand-2')
  await failed
  deliver('doudizhu.hand.snapshot',viewReply(),'sync-1')
  assert.equal(s.hand.snapshot,null)
  s.dispose()
 }
})
test('G65 R5: terminal fallback validates the same archive contract and completes without invented cards',async()=>{
 const {s,deliver}=await syncFixture()
 const p=s.syncHand('hand-1')
 const evidence={handId:'hand-1',status:'completed',finalVersion:8,
  verification:{handId:'hand-1',finalVersion:8,recordDigest:'a'.repeat(64),seatPoints:[6,-3,-3],winnerSeat:1}}
 deliver('doudizhu.hand.evidence',{v:'doudizhu-realtime-event-v1',handId:'hand-1',evidence},'sync-1')
 await p;assert.equal(s.hand.snapshot.phase,'COMPLETED');assert.equal(s.hand.snapshot.version,8)
 assert.equal(s.hand.snapshot.view.cards,undefined);assert.equal(s.hand.canSubmit,false)
 s.dispose()
})

test('G65 R5: a newer committed change invalidates earlier readiness; old change does not',async()=>{
 const {s,deliver}=await syncFixture()
 const first=s.syncHand('hand-1')
 deliver('doudizhu.hand.snapshot',viewReply('hand-1',2),'sync-1')
 await first;assert.equal(s.handProjectionReady('hand-1'),true)
 const changed=version=>deliver('doudizhu.hand.changed',{v:'doudizhu-realtime-event-v1',
   handId:'hand-1',response:{live:{version}}})
 changed(2);assert.equal(s.handProjectionReady('hand-1'),true)
 changed(3);assert.equal(s.handProjectionReady('hand-1'),false)
 const pending=s.syncHand('hand-1',2),rejected=assert.rejects(pending,/stale/)
 deliver('doudizhu.hand.snapshot',viewReply('hand-1',2),'sync-2')
 await rejected;assert.equal(s.handProjectionReady('hand-1'),false)
 const retry=s.syncHand('hand-1')
 deliver('doudizhu.hand.snapshot',viewReply('hand-1',3),'sync-3')
 await retry;assert.equal(s.handProjectionReady('hand-1'),true)
 s.dispose()
})
test('G65 R5: verified same-version archive promotes a completed live projection',async()=>{
 const {s,deliver}=await syncFixture()
 s.acceptVerifiedSnapshot({...snapshot,version:8,phase:'COMPLETED'})
 assert.equal(s.handProjectionReady('hand-1',true),false)
 const pending=s.syncHand('hand-1')
 const evidence={handId:'hand-1',status:'completed',finalVersion:8,
  verification:{handId:'hand-1',finalVersion:8,recordDigest:'a'.repeat(64),seatPoints:[6,-3,-3],winnerSeat:1}}
 deliver('doudizhu.hand.evidence',{v:'doudizhu-realtime-event-v1',handId:'hand-1',evidence},'sync-1')
 await pending
 assert.equal(s.handProjectionReady('hand-1',true),true)
 assert.equal(s.hand.snapshot.view.terminalEvidence.recordDigest,'a'.repeat(64))
 assert.equal(s.hand.snapshot.view.cards,undefined)
 s.dispose()
})

test('G65 R5: an unversioned committed timeout fences an older pending sync',async()=>{
 const {s,deliver}=await syncFixture()
 const initial=s.syncHand('hand-1')
 deliver('doudizhu.hand.snapshot',viewReply('hand-1',2),'sync-1')
 await initial
 const stale=s.syncHand('hand-1',2),failed=assert.rejects(stale,/invalidated/)
 deliver('doudizhu.hand.changed',{v:'doudizhu-realtime-event-v1',handId:'hand-1',
  response:{type:'hand.timeout'}})
 await failed
 deliver('doudizhu.hand.snapshot',viewReply('hand-1',2),'sync-2')
 assert.equal(s.handProjectionReady('hand-1'),false)
 const retry=s.syncHand('hand-1')
 deliver('doudizhu.hand.snapshot',viewReply('hand-1',3),'sync-3')
 await retry
 assert.equal(s.handProjectionReady('hand-1'),true)
 s.dispose()
})
test('G65 R5: a conflicting equal-version phase is not a successful sync',async()=>{
 const {s,deliver}=await syncFixture()
 const initial=s.syncHand('hand-1')
 deliver('doudizhu.hand.snapshot',viewReply('hand-1',2),'sync-1')
 await initial
 const retry=s.syncHand('hand-1'),failed=assert.rejects(retry,/conflicting/)
 const conflicting=viewReply('hand-1',2)
 conflicting.view.payload.public.phase='PLAYING'
 deliver('doudizhu.hand.snapshot',conflicting,'sync-2')
 await failed
 assert.equal(s.hand.snapshot.phase,'BIDDING')
 assert.equal(s.handProjectionReady('hand-1'),false)
 s.dispose()
})

test('G65 R7: unversioned change blocks old broadcasts and conditional reads, not a fresh authoritative equal version',async()=>{
 for(const waiting of [false,true]){
  const {s,deliver,c}=await syncFixture()
  try{
   const initial=s.syncHand('hand-1')
   deliver('doudizhu.hand.snapshot',viewReply('hand-1',2),'sync-1')
   await initial
   const pending=waiting?s.syncHand('hand-1',2):null
   const rejected=pending?assert.rejects(pending,/invalidated/):null
   const timeout=()=>deliver('doudizhu.hand.changed',{v:'doudizhu-realtime-event-v1',
     handId:'hand-1',response:{type:'hand.timeout'}})
   timeout();timeout()
   if(rejected)await rejected
   deliver('doudizhu.hand.snapshot',viewReply('hand-1',2))
   assert.equal(s.handProjectionReady('hand-1'),false)
   assert.equal(s.hand.canSubmit,false)
   const stale=s.syncHand('hand-1',2),failed=assert.rejects(stale,/notModified/)
   const request=c.sent.at(-1)
   assert.equal(request.payload.knownVersion,undefined)
   deliver('doudizhu.hand.snapshot',{v:'doudizhu-realtime-event-v1',handId:'hand-1',
     notModified:true,knownVersion:2},request.id)
   await failed
   assert.equal(s.handProjectionReady('hand-1'),false)
   const retry=s.syncHand('hand-1')
   deliver('doudizhu.hand.snapshot',viewReply('hand-1',2),c.sent.at(-1).id)
   await retry
   assert.equal(s.handProjectionReady('hand-1'),true)
   timeout()
   const again=s.syncHand('hand-1')
   deliver('doudizhu.hand.snapshot',viewReply('hand-1',2),c.sent.at(-1).id)
   await again
   assert.equal(s.handProjectionReady('hand-1'),true)
  }finally{s.dispose()}
 }
})
test('G65 R7: causal freshness fence survives reconnect but resets on a different authoritative hand',async()=>{
 const {s,t,deliver}=await syncFixture()
 const first=s.syncHand('hand-1')
 deliver('doudizhu.hand.snapshot',viewReply('hand-1',2),'sync-1')
 await first
 deliver('doudizhu.hand.changed',{v:'doudizhu-realtime-event-v1',handId:'hand-1',response:{type:'hand.timeout'}})
 const open=s.connect(t,'wss://example.org/ws','token')
 t.tasks[1].resolve();await open
 const send=(payload,id)=>{for(const cb of t.tasks[1].c.messages)cb(JSON.stringify({id,type:'doudizhu.hand.snapshot',payload}))}
 const stale=s.syncHand('hand-1'),failed=assert.rejects(stale,/stale/)
 send(viewReply('hand-1',1),'sync-2');await failed
 assert.equal(s.handProjectionReady('hand-1'),false)
 s.useAuthoritativeHand('hand-2')
 const next=s.syncHand('hand-2')
 send(viewReply('hand-2',1),'sync-3');await next
 assert.equal(s.handProjectionReady('hand-2'),true)
 s.dispose()
})
test('G65 R6: inconsistent command envelope and payload IDs cannot settle a receipt',async()=>{
 const {s,deliver}=await syncFixture()
 s.acceptVerifiedSnapshot(snapshot)
 const input={requestId:'move-r6',type:'hand.pass',aggregateId:'hand-1',clientSeq:1,expectedVersion:7}
 const p=s.submitCommand(input)
 let done=false;void p.then(()=>{done=true},()=>{done=true})
 const result={v:'doudizhu-api-command-result-v1',requestId:input.requestId,type:input.type,
  live:{version:8,payload:{},requiresTermination:false}}
 deliver('doudizhu.command.result',result,'other-id')
 deliver('doudizhu.error',{v:'doudizhu-realtime-event-v1',requestId:input.requestId,code:'FORBIDDEN'},'other-id')
 await tick();assert.equal(done,false)
 deliver('doudizhu.command.result',result,input.requestId)
 assert.equal((await p).live.version,8);s.dispose()
})

// Exact resource-bound integration: real MiniSocketTransport, manager,
// router and shell; the controlled native SDK never completes a send.
test('G65 R6: repeated read retry cannot accumulate native send promises on a stalled task',async()=>{
 const {MiniSocketTransport}=await import('@awesome-zero-platform/game-client-core/minigame')
 const tasks=[]
 const sdk={connectSocket(){
  const listeners={open:new Set(),message:new Set(),close:new Set(),error:new Set()}
  const task={calls:[],closes:0,listeners,
   send(args){this.calls.push(args)},close(){this.closes++},
   onOpen(cb){listeners.open.add(cb);queueMicrotask(cb)},offOpen(cb){listeners.open.delete(cb)},
   onMessage(cb){listeners.message.add(cb)},offMessage(cb){listeners.message.delete(cb)},
   onClose(cb){listeners.close.add(cb)},offClose(cb){listeners.close.delete(cb)},
   onError(cb){listeners.error.add(cb)},offError(cb){listeners.error.delete(cb)}}
  tasks.push(task);return task
 }}
 const native=new MiniSocketTransport(sdk)
 let outstanding=0
 const socket={async connect(...args){
  const c=await native.connect(...args)
  return {...c,send(raw,timeoutMs){outstanding++;return c.send(raw,timeoutMs).finally(()=>outstanding--)}}
 }}
 const s=new GameClientShell(()=>{},10)
 await s.initialize()
 try{
  for(let attempt=0;attempt<4;attempt++){
   await s.connect(socket,'wss://example.org/ws','token')
   s.useAuthoritativeHand('hand-1')
   await assert.rejects(s.syncHand('hand-1'),/timed out|connection lost/)
   // Read and send use separate deadlines; allow the transport close
   // callback from the same deadline to settle before checking ownership.
   for(let i=0;i<20&&outstanding;i++)await new Promise(ok=>setTimeout(ok,1))
   assert.equal(outstanding,0)
   assert.equal(s.commandTransportReady,false)
   assert.equal(tasks.at(-1).calls.length,1)
   assert.equal(tasks.at(-1).closes,1)
   assert.ok(Object.values(tasks.at(-1).listeners).every(set=>set.size===0))
  }
 }finally{s.dispose()}
 tasks.forEach(t=>t.calls.forEach(c=>{c.success();c.fail(Error('late callback'))}))
 assert.equal(outstanding,0)
})

test('G65 R8: deal notice requires private projection and never uses durable version as live floor',async()=>{
 const {s,deliver,c}=await syncFixture()
 try{
  deliver('doudizhu.hand.changed',{v:'doudizhu-realtime-event-v1',handId:'hand-1',
   response:{type:'hand.dealt',durable:{aggregateVersion:17}}})
  assert.equal(s.requiresLiveProjection('hand-1'),true)
  assert.equal(s.handProjectionReady('hand-1'),false)
  const read=s.syncHand('hand-1')
  deliver('doudizhu.hand.snapshot',viewReply('hand-1',1),c.sent.at(-1).id)
  await read
  assert.equal(s.handProjectionReady('hand-1'),true)
  s.useAuthoritativeHand('hand-2')
  assert.equal(s.requiresLiveProjection('hand-1'),false)
 }finally{s.dispose()}
})

test('G65 R11: shell disposal settles pending sync and clears state despite native close fault',async()=>{
 const notices=[],s=new GameClientShell(v=>notices.push(v));await s.initialize()
 const t=transport(),p=s.connect(t,'wss://example.org/ws','token')
 t.tasks[0].resolve();await p
 s.acceptVerifiedSnapshot(snapshot)
 const c=t.tasks[0].c
 c.close=()=>{throw Error('close fault')}
 const pending=assert.rejects(s.syncHand('hand-1',7),/disposed/)
 s.dispose();await pending
 assert.equal(s.commandTransportReady,false)
 assert.equal(s.hand.snapshot,null);assert.equal(s.hand.status,'disconnected')
 assert.equal(c.messages.size,0);assert.equal(c.closes.size,0)
 s.dispose()
})

test('G65 R12: shell cleanup reentry preserves latest connection and transport readiness',async()=>{
 for(const action of ['connect','cancelConnection']){
  const s=new GameClientShell(()=>{}),t=transport();await s.initialize()
  let latest
  const first=s.connect(t,'wss://x/ws','initial');t.tasks[0].resolve();await first
  const c=t.tasks[0].c,off=c.onMessage
  // Replace the registered off function, not the later registration method.
  const manager=s.connection
  manager.unsubscribe.unshift(()=>{latest=s.connect(t,'wss://x/ws','latest')})
  const stale=action==='connect'?s.connect(t,'wss://x/ws','superseded'):s.cancelConnection()
  assert.equal(t.tasks.length,2)
  assert.equal(s.hand.status,'reconnecting')
  t.tasks[1].resolve()
  assert.equal(await latest,true)
  if(action==='connect')assert.equal(await stale,false)
  assert.equal(s.commandTransportReady,true)
  assert.equal(s.connection.isOpen,true)
  assert.equal(c.closed,1)
  assert.equal(c.messages.size,0)
  assert.equal(t.tasks[1].c.messages.size,1)
  await s.connection.send({probe:true})
  assert.deepEqual(t.tasks[1].c.sent,[{probe:true}])
  s.dispose();assert.equal(t.tasks[1].c.closed,1)
 }
})
test('G65 R12: success notice may cancel or replace connection before returning readiness',async()=>{
 for(const replace of [false,true]){
  const t=transport()
  let latest,trigger=true
  const s=new GameClientShell(message=>{
   if(trigger && message.startsWith('已连接')){
    trigger=false
    if(replace)latest=s.connect(t,'wss://x/ws','latest')
    else s.cancelConnection()
   }
  })
  await s.initialize()
  const old=s.connect(t,'wss://x/ws','old');t.tasks[0].resolve()
  assert.equal(await old,false)
  assert.equal(t.tasks[0].c.closed,1)
  if(replace){t.tasks[1].resolve();assert.equal(await latest,true);assert.equal(s.commandTransportReady,true)}
  else assert.equal(s.commandTransportReady,false)
  s.dispose()
 }
})
