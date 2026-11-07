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

test('client-created custodial hand version and HPKE contribution have strict wire bindings',()=>{
 const req={...base,expectedVersion:3,type:'room.hand.start',
   payload:{handId:'hand-2',fairnessMode:'client-contributed-server-custodied-v1'}}
 assert.deepEqual(createWireCommand(req).payload,req.payload)
 assert.throws(()=>createWireCommand({...req,payload:{handId:'hand-2',fairnessMode:'server-managed-v1'}}),/invalid/)
 const contribution={...base,aggregateId:'hand-2',expectedVersion:1,type:'hand.contribution',
   revealContext:{claims:{
     v:'doudizhu-custodied-context-v1',actor:'alice',handId:'hand-2',
     requestId:base.requestId,clientSeq:1,expectedVersion:1,seat:1,
     commitment:'',keyId:'key-1',publicKeySha256:'0'.repeat(64),
     issuedAt:100000,expiresAt:160000},mac:'a'.repeat(64)},
   payload:{envelope:{v:'secure-envelope-v1',keyId:'key-1',
     suite:'hpke-x25519-hkdf-sha256-aes-256-gcm',
     encapsulatedKey:'YWJj',ciphertext:'YWJj'}}}
 assert.equal(createWireCommand(contribution).type,'hand.contribution')
 assert.throws(()=>createWireCommand({...contribution,revealContext:undefined}),/AAD/)
 assert.throws(()=>createWireCommand({...contribution,revealContext:{
   ...contribution.revealContext,
   claims:{...contribution.revealContext.claims,v:'doudizhu-reveal-context-v1'}}}),/AAD/)
 assert.throws(()=>createWireCommand({...contribution,payload:{...contribution.payload,phrase:'secret'}}),/envelope/)
})
