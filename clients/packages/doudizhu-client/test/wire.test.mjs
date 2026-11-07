import { test } from 'node:test'
import assert from 'node:assert/strict'
import { createWireCommand, parseWireResult } from '../dist/wire.js'

const base = { requestId: 'req-001', aggregateId: 'room-001', clientSeq: 1, expectedVersion: 0 }
test('real dispatcher room create wire shape', () => {
  assert.deepEqual(createWireCommand({ ...base, type: 'room.create' }), {
    v: 'doudizhu-api-command-v1', requestId: 'req-001', type: 'room.create',
    aggregateId: 'room-001', clientSeq: 1, expectedVersion: 0, payload: {}
  })
})
test('non-creation requests must observe a nonzero version', () => {
  assert.throws(() => createWireCommand({ ...base, type: 'room.join' }), /invalid/)
})
test('room ready payload matches actual Go DTO', () => {
  const command = createWireCommand({ ...base, expectedVersion: 3, type: 'room.ready', payload: { ready: true } })
  assert.deepEqual(command.payload, { ready: true })
  assert.throws(() => createWireCommand({ ...base, expectedVersion: 3, type: 'room.ready', payload: { ready: 1 } }))
})
test('commit uses 32-byte lowercase hex, not base64url', () => {
  assert.ok(createWireCommand({ ...base, expectedVersion: 1, type: 'hand.commit', payload: { commitment: 'a'.repeat(64) } }))
  assert.throws(() => createWireCommand({ ...base, expectedVersion: 1, type: 'hand.commit', payload: { commitment: 'a'.repeat(43) } }))
})
test('reveal fails closed until actual AAD is attainable', () => {
  assert.throws(() => createWireCommand({ ...base, expectedVersion: 1, type: 'hand.reveal', payload: { envelope: {} } }), /AAD/)
})
test('live commands map to actual dispatcher command types', () => {
  assert.equal(createWireCommand({ ...base, expectedVersion: 8, type: 'hand.bid', payload: { score: 2 } }).type, 'hand.bid')
  assert.deepEqual(createWireCommand({ ...base, expectedVersion: 9, type: 'hand.play', payload: { cards: ['S3'] } }).payload.cards, ['S3'])
  assert.equal(createWireCommand({ ...base, expectedVersion: 10, type: 'hand.pass' }).type, 'hand.pass')
})
test('correlation required before accepting server command result', () => {
  const command = createWireCommand({ ...base, type: 'room.create' })
  const result = { v: 'doudizhu-api-command-result-v1', requestId: 'req-001', type: 'room.create', durable: { accepted: true } }
  assert.deepEqual(parseWireResult(result, command), result)
  assert.throws(() => parseWireResult({ ...result, requestId: 'other' }, command), /uncorrelated/)
})

test('server-managed hand mode is explicit, versioned and immutable within a command',()=>{
 const original={...base,expectedVersion:4,type:'room.hand.start',
   payload:{handId:'hand-new',fairnessMode:'server-managed-v1'}}
 const valid=createWireCommand(original)
 assert.deepEqual(valid.payload,original.payload)
 assert.throws(()=>createWireCommand({...original,payload:{handId:'hand-new',fairnessMode:'unknown'}}),/invalid/)
 assert.throws(()=>createWireCommand({...original,payload:{handId:'hand-new',fairnessMode:'server-managed-v1',secret:'leak'}}),/invalid/)
 assert.deepEqual(createWireCommand({...original,payload:{handId:'hand-legacy'}}).payload,{handId:'hand-legacy'})
})
