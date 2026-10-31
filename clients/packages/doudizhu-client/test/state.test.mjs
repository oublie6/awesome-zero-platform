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
    aggregateType:'room',aggregateId:'room_1',clientSeq:1,expectedVersion:0,
    issuedAt:'2026-10-31T00:00:00.000Z',expiresAt:'2026-10-31T00:01:00.000Z',payload:{ready:true}}
  assert.doesNotThrow(()=>requireValidCommand(command))
  assert.throws(()=>requireValidCommand({...command,clientSeq:0}),/invalid/)
})
