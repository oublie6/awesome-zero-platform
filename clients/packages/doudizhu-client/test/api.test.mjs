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

test('room GET supplies authoritative version while outsiders receive no seats',async()=>{
 let actual
 const client=new DoudizhuHTTPClient({request:async(path,options)=>{
  actual={path,options}
  return {code:'OK',data:{roomId:'r-1',version:4,status:'WAITING_PLAYERS',occupied:2,member:false,lastClientSequence:0}}
 }},()=> 'real-token')
 const room=await client.room('r-1')
 assert.equal(room.version,4)
 assert.equal(room.member,false)
 assert.equal(room.seats,undefined)
 assert.equal(actual.path,'/v1/doudizhu/rooms/r-1')
 assert.equal(actual.options.accessToken,'real-token')
 assert.equal(JSON.stringify(actual).includes('real-token?'),false)
})
test('room GET rejects response identity and membership leaks',async()=>{
 const cases=[
  {roomId:'other',version:1,status:'READY',occupied:2,member:false},
  {roomId:'r-1',version:0,status:'READY',occupied:2,member:false},
  {roomId:'r-1',version:1,status:'READY',occupied:2,member:false,lastClientSequence:0,seats:[]},
  {roomId:'r-1',version:1,status:'READY',occupied:2,member:true,lastClientSequence:0,seats:[{position:1,accountId:'a',ready:true}]}
 ]
 for(const data of cases) {
  const client=new DoudizhuHTTPClient({request:async()=>({code:'OK',data})},()=> 'token')
  await assert.rejects(client.room('r-1'),/room view|room preview/)
 }
})
