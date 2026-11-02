import {test} from 'node:test'
import assert from 'node:assert/strict'
import {computeCommitment,canonicalRevealAAD,GameFairnessActions} from '../dist/game/GameFairnessActions.js'

test('commitment matches canonical Go contribution and commitment hashing vector',()=>{
 const random=Uint8Array.from({length:32},(_,i)=>i+1)
 assert.equal(computeCommitment('hand-1',1,random,'NFKC-test'),
  '1e708cefcaf99ec3905723528f8aa7a5ee2b1d72b922cabc5f9057910c2544f5')
 assert.notEqual(computeCommitment('hand-1',2,random,'NFKC-test'),
  computeCommitment('hand-1',1,random,'NFKC-test'))
 assert.equal(computeCommitment('hand-1',1,random,'ＡＢＣ'),
  computeCommitment('hand-1',1,random,'ABC'))
})
test('reveal AAD canonical bytes match Go binary length and milliseconds layout',()=>{
 const claims={v:'doudizhu-reveal-context-v1',actor:'account-1',handId:'hand-1',
   requestId:'reveal-001',clientSeq:12,expectedVersion:7,seat:1,
   keyId:'key-001',commitment:'01'.repeat(32),publicKeySha256:'02'.repeat(32),
   issuedAt:100000,expiresAt:160000}
 assert.equal(Buffer.from(canonicalRevealAAD(claims)).toString('hex'),
 '0000001b666169722d646f7564697a68752f72657665616c2d6161642f763100000018666169722d646f7564697a68752d636f6d6d616e642d763100000027646f7564697a68752e68616e642e666169726e6573732e72657665616c2e7375626d69742e7631000000076b65792d3030310000000a72657665616c2d3030310000000468616e640000000668616e642d31000000096163636f756e742d3101000000000000000c0000000000000007010101010101010101010101010101010101010101010101010101010101010100000000000186a00000000000027100')
})
test('real fairness commit stays attached to authoritative seat/version and uses secure random',async()=>{
 const sent=[]
 const api={
  fairness:async()=>({handId:'hand-1',phase:'FAIRNESS_COMMITTING',version:3,seat:2,
    committed:false,revealed:false,lastClientSequence:4}),
  command:async c=>{
   sent.push(c)
   return {v:'doudizhu-api-command-result-v1',requestId:c.requestId,type:c.type,
    durable:{v:'fair-doudizhu-command-result-v1',commandId:c.requestId,accepted:true}}
  }
 }
 let count=0
 const platform={random:{bytes:async n=>{count++;return Uint8Array.from({length:n},(_,i)=>i+count)}}}
 const flow=new GameFairnessActions(api,platform,{})
 const result=await flow.commit('hand-1','NFKC-test')
 assert.equal(result.durable.accepted,true)
 assert.equal(sent.length,1)
 assert.equal(sent[0].type,'hand.commit')
 assert.equal(sent[0].expectedVersion,3)
 assert.equal(sent[0].clientSeq,5)
 assert.match(sent[0].payload.commitment,/^[0-9a-f]{64}$/)
 assert.ok(count>=2)
 flow.dispose()
})
test('unknown commit outcome is not retried under a new ID and does not lose secret',async()=>{
 let fail=true,commands=[]
 const api={
  fairness:async()=>({handId:'hand-1',phase:'FAIRNESS_COMMITTING',version:3,seat:2,committed:false,
   revealed:false,lastClientSequence:4}),
  command:async c=>{
   commands.push(c)
   if(fail)throw Error('network timeout')
   return {v:'doudizhu-api-command-result-v1',requestId:c.requestId,type:c.type,
    durable:{v:'fair-doudizhu-command-result-v1',commandId:c.requestId,accepted:true}}
  }
 }
 const flow=new GameFairnessActions(api,{random:{bytes:async n=>new Uint8Array(n).fill(1)}},{})
 await assert.rejects(flow.commit('hand-1','phrase'),/network timeout/)
 assert.equal(flow.needsReconciliation,true)
 await assert.rejects(flow.commit('hand-1','other'),/unknown/)
 fail=false
 await flow.retryUnknown()
 assert.equal(flow.needsReconciliation,false)
 assert.deepEqual(commands[0],commands[1])
 flow.dispose()
})
