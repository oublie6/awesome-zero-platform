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
