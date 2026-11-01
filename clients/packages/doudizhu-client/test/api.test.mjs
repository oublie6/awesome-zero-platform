import { test } from 'node:test'
import assert from 'node:assert/strict'
import { DoudizhuHTTPClient } from '../dist/api.js'
const cmd={requestId:'req-01',type:'room.create',aggregateId:'room-01',clientSeq:1,expectedVersion:0}
test('authenticated real dispatcher command and wrapped Go response',async()=>{
 let actual
 const client=new DoudizhuHTTPClient({request:async(path,options)=>{actual={path,options};return {code:'OK',data:{v:'doudizhu-api-command-result-v1',requestId:'req-01',type:'room.create',durable:{accepted:true}}}}},()=> 'server-token')
 const result=await client.command(cmd)
 assert.equal(actual.path,'/v1/doudizhu/commands')
 assert.equal(actual.options.accessToken,'server-token')
 assert.equal(actual.options.body.v,'doudizhu-api-command-v1')
 assert.equal(result.durable.accepted,true)
})
test('API rejects missing auth or malformed wrapper',async()=>{
 const client=new DoudizhuHTTPClient({request:async()=>({code:'OK',data:null})},()=> '')
 await assert.rejects(client.command(cmd),/token required/)
 const bad=new DoudizhuHTTPClient({request:async()=>({code:'CONFLICT',data:null})},()=> 't')
 await assert.rejects(bad.command(cmd),/CONFLICT/)
})
test('private hand path does not accept untrusted query injection',async()=>{
 const client=new DoudizhuHTTPClient({request:async()=>({})},()=> 't')
 await assert.rejects(client.privateHand('x?token=leak'),/invalid hand id/)
})
