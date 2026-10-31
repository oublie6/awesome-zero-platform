import {test} from 'node:test'
import assert from 'node:assert/strict'
import {HandStore, requireValidCommand} from '../dist/index.js'
const view = Object.freeze({cards: ['3S']})
test('server snapshots never regress in version', () => {
  const state = new HandStore()
  assert.equal(state.apply({handId:'h1',version:3,phase:'PLAYING',view}),true)
  assert.equal(state.apply({handId:'h1',version:2,phase:'BIDDING',view}),false)
  assert.equal(state.snapshot.version,3)
  assert.throws(()=>state.apply({handId:'h2',version:5,phase:'PLAYING',view}),/identity/)
  state.reset()
  assert.equal(state.snapshot,null)
})
test('commands require connection and synchronization', () => {
  const state = new HandStore()
  state.apply({handId:'h1',version:3,phase:'PLAYING',view})
  assert.equal(state.canSubmit,false)
  state.setStatus('reconnecting')
  state.setStatus('synchronizing')
  state.setStatus('connected')
  assert.equal(state.canSubmit,true)
  state.setStatus('disconnected')
  assert.equal(state.canSubmit,false)
  assert.throws(()=>state.setStatus('connected'),/invalid reconnect/)
})
test('versioned command envelopes reject malformed data',()=>{
  const command={v:'fair-doudizhu-command-v1',name:'doudizhu.room.ready.set.v1',commandId:'id',
    aggregateType:'room',aggregateId:'room_1',clientSeq:1,expectedVersion:1,
    issuedAt:'2026-10-31T00:00:00.000Z',expiresAt:'2026-10-31T00:01:00.000Z',payload:{ready:true}}
  assert.doesNotThrow(()=>requireValidCommand(command))
  assert.throws(()=>requireValidCommand({...command,clientSeq:0}),/invalid/)
})

test('duplicate snapshots do not replace state and consumers cannot mutate accepted state', () => {
  const state = new HandStore(), input = {handId:'h1',version:3,phase:'PLAYING',view:{cards:['3S']}}
  state.apply(input); input.view.cards[0]='4S'; assert.equal(state.snapshot.view.cards[0],'3S')
  assert.equal(state.apply({...input,view:{cards:['5S']}}),false)
  assert.throws(()=>{state.snapshot.view.cards.push('6S')}, TypeError)
  state.setStatus('reconnecting');state.setStatus('synchronizing');state.setStatus('connected')
  state.apply({...input,version:4,phase:'COMPLETED'});assert.equal(state.canSubmit,false)
})
test('invalid command shapes, names, actor claims and timestamps fail closed', () => {
  const c={v:'fair-doudizhu-command-v1',name:'doudizhu.room.ready.set.v1',commandId:'id',aggregateType:'room',aggregateId:'room_1',clientSeq:1,expectedVersion:1,issuedAt:'2026-10-31T00:00:00.000Z',expiresAt:'2026-10-31T00:01:00.000Z',payload:{ready:true}}
  for(const patch of [{name:'unknown.v1'},{aggregateType:'hand'},{payload:null},{payload:[]},{payload:{actorId:'fake'}},{seat:1},{issuedAt:'2026-02-30T00:00:00.000Z'},{expiresAt:'2026-10-31T00:02:00.000Z'},{clientSeq:NaN},{commandId:' '},{expectedVersion:0}]) assert.throws(()=>requireValidCommand({...c,...patch}),/invalid/)
  assert.throws(()=>requireValidCommand(null),/invalid/)
})

test('router gates wire versions and old unsubscribe cannot remove a replacement listener', async()=>{
  const {GameMessageRouter,createCommand}=await import('../dist/index.js')
  const r=new GameMessageRouter();const old=r.on('doudizhu.error',()=>{});r.clear();let count=0
  r.on('doudizhu.error',()=>count++);old()
  assert.throws(()=>r.receive('{'),/invalid/)
  assert.throws(()=>r.receive(JSON.stringify({type:'doudizhu.error',payload:{v:'future'}})),/version/)
  r.receive(JSON.stringify({type:'doudizhu.error',payload:{v:'doudizhu-realtime-event-v1',code:'FORBIDDEN'}}));assert.equal(count,1)
  r.clear();r.receive(JSON.stringify({type:'doudizhu.error',payload:{v:'doudizhu-realtime-event-v1'}}));assert.equal(count,1)
  const input={name:'doudizhu.room.ready.set.v1',commandId:'retry-id',aggregateType:'room',aggregateId:'room-1',clientSeq:7,expectedVersion:3,issuedAt:'2026-10-31T00:00:00.000Z',expiresAt:'2026-10-31T00:01:00.000Z',payload:{ready:true}}
  const c=createCommand(input);input.payload.ready=false
  assert.equal(c.payload.ready,true);assert.equal(c.commandId,'retry-id');assert.equal(c.clientSeq,7)
  assert.throws(()=>{c.commandId='different'},TypeError)
})

test('snapshot phase coverage matches actual Go owner constants, including beacon wait',async()=>{
  const {readFile}=await import('node:fs/promises')
  const source=await readFile(new URL('../../../../server/business/doudizhu/domain/hand_types.go',import.meta.url),'utf8')
  const phases=[...source.matchAll(/Hand\w+\s+HandPhase\s*=\s*"([A-Z_]+)"/g)].map(m=>m[1])
  assert.ok(phases.length>=10)
  for(const phase of phases)assert.equal(new HandStore().apply({handId:'h1',version:1,phase,view:{}}),true,phase)
})
