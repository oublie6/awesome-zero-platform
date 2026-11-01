import { test } from 'node:test'
import assert from 'node:assert/strict'
import { parsePrivateViewResult } from '../dist/view.js'
const view = {
  v: 'doudizhu-live-private-view-v1',
  public: {
    v: 'doudizhu-live-public-view-v1', handId: 'hand-1', phase: 'BIDDING',
    stateVersion: 4, bidding: { v: 'bidding-state-v1' },
    seats: [
      {position:1, accountId:'acct-a', remainingCards:17},
      {position:2, accountId:'acct-b', remainingCards:17},
      {position:3, accountId:'acct-c', remainingCards:17}
    ]
  }, position: 2, cards: ['S3','D4']
}
test('accept Go versioned private-view wrapper', () => {
  const result = parsePrivateViewResult({version:4, payload:view})
  assert.equal(result.view.position, 2)
  assert.deepEqual(result.view.cards,['S3','D4'])
})
test('reject wrapper and actual view version mismatch',()=>{
  assert.throws(()=>parsePrivateViewResult({version:3,payload:view}),/version mismatch/)
})
test('reject client-generated or incomplete private hands',()=>{
  assert.throws(()=>parsePrivateViewResult({version:4,payload:{ ...view, cards:undefined }}),/private cards/)
  assert.throws(()=>parsePrivateViewResult({version:4,payload:{ ...view, public:{...view.public,seats:[]} }}),/public hand/)
})
test('parsed private cards do not reference untrusted input',()=>{
  const result = parsePrivateViewResult({version:4,payload:view})
  view.cards[0]='H9'
  assert.deepEqual(result.view.cards,['S3','D4'])
  view.cards[0]='S3'
})
