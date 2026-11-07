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

test('fairness GET returns only the authenticated seat commitment and sequence',async()=>{
 let observed
 const client=new DoudizhuHTTPClient({request:async(path,options)=>{
  observed={path,options}
  return {code:'OK',data:{
   handId:'hand-1',phase:'FAIRNESS_REVEALING',version:5,seat:2,committed:true,revealed:false,
   lastClientSequence:7,commitment:'01'.repeat(32),serverCommitment:'02'.repeat(32),
   revealKeyId:'key-1',revealPublicKeySha256:'03'.repeat(32),
   revealKeyBoundAt:'2026-11-02T06:00:00Z'
  }}
 }},()=> 'server-auth-token')
 const view=await client.fairness('hand-1')
 assert.equal(view.lastClientSequence,7)
 assert.equal(view.seat,2)
 assert.equal(observed.path,'/v1/doudizhu/hands/hand-1/fairness')
 assert.equal(observed.options.accessToken,'server-auth-token')
})
test('client refuses malformed or mismatched server-signed reveal AAD context',async()=>{
 const request={requestId:'secure-id',clientSeq:3,expectedVersion:5}
 const expected={v:'doudizhu-reveal-context-v1',ticket:{
  mac:'a'.repeat(64),claims:{v:'doudizhu-reveal-context-v1',actor:'account',handId:'hand-1',
   requestId:'secure-id',clientSeq:3,expectedVersion:5,seat:1,commitment:'01'.repeat(32),
   keyId:'reveal-key',publicKeySha256:'02'.repeat(32),issuedAt:100000,expiresAt:160000}
 },aad:'YWJj'}
 const client=new DoudizhuHTTPClient({request:async()=>({code:'OK',data:expected})},()=> 'server-token')
 assert.equal((await client.revealContext('hand-1',request)).ticket.claims.requestId,'secure-id')
 const malicious=new DoudizhuHTTPClient({request:async()=>({code:'OK',data:{
  ...expected,ticket:{...expected.ticket,claims:{...expected.ticket.claims,seat:4}}
 }})},()=> 'server-token')
 await assert.rejects(malicious.revealContext('hand-1',request),/mismatched/)
 const wrongHand=new DoudizhuHTTPClient({request:async()=>({code:'OK',data:{
  ...expected,ticket:{...expected.ticket,claims:{...expected.ticket.claims,handId:'another'}}
 }})},()=> 'server-token')
 await assert.rejects(wrongHand.revealContext('hand-1',request),/mismatched/)
})

test('raw signed Go reveal-key manifest is passed to cryptographic verifier without API-envelope unwrap',async()=>{
 const manifest={manifestVersion:7,protocolVersion:'reveal-key-manifest-v1',keyId:'key-1',
   suite:'hpke-x25519-hkdf-sha256-aes-256-gcm',publicKey:'a',publicKeySha256:'b',
   notBefore:'2026-11-02T00:00:00Z',expiresAt:'2026-11-03T00:00:00Z',
   status:'active',signatureKeyId:'root-1',signature:'signed-by-root'}
 const client=new DoudizhuHTTPClient({request:async()=>manifest},()=> 'token')
 assert.deepEqual(await client.revealManifest('key-1'),manifest)
 const envelope=new DoudizhuHTTPClient({request:async()=>({code:'OK',data:manifest})},()=> 'token')
 await assert.rejects(envelope.revealManifest('key-1'),/signed reveal manifest/)
})
test('real partially occupied Go room has omitted empty-seat account IDs',async()=>{
 for(const occupied of [1,2,3]){
  const seats=[1,2,3].map(i=>i<=occupied
    ? {position:i,accountId:'player-'+i,ready:true}
    : {position:i,ready:false})
  const client=new DoudizhuHTTPClient({request:async()=>({code:'OK',data:{
    roomId:'r-1',version:occupied,status:'WAITING_PLAYERS',occupied,
    member:true,lastClientSequence:0,seats
  }})},()=> 'token')
  const room=await client.room('r-1')
  assert.equal(room.seats.length,3)
  assert.equal(room.seats.filter(s=>s.accountId!=='').length,occupied)
  assert.equal(room.seats[2].accountId,occupied===3?'player-3':'')
 }
 const invalid=new DoudizhuHTTPClient({request:async()=>({code:'OK',data:{
    roomId:'r-1',version:1,status:'WAITING_PLAYERS',occupied:1,member:true,
    lastClientSequence:0,seats:[{position:1,accountId:'a',ready:true},
      {position:2,ready:true},{position:3,ready:false}]
  }})},()=> 'token')
 await assert.rejects(invalid.room('r-1'),/member room/)
})

test('server-issued custodial HPKE ticket is authenticated and version separated',async()=>{
 const request={requestId:'request-custody',clientSeq:2,expectedVersion:3}
 const ticket={v:'doudizhu-custodied-context-v1',ticket:{
  claims:{v:'doudizhu-custodied-context-v1',actor:'alice',handId:'hand-1',
   requestId:'request-custody',clientSeq:2,expectedVersion:3,seat:1,
   commitment:'',keyId:'key-one',publicKeySha256:'b'.repeat(64),
   issuedAt:100000,expiresAt:160000},mac:'a'.repeat(64)},aad:'YWJj'}
 const api=data=>new DoudizhuHTTPClient({request:async()=>({code:'OK',data})},()=> 'auth')
 assert.equal((await api(ticket).revealContext('hand-1',request,'custodied')).v,'doudizhu-custodied-context-v1')
 await assert.rejects(api(ticket).revealContext('hand-1',request),/signed reveal context/)
 await assert.rejects(api({...ticket,ticket:{...ticket.ticket,claims:{
  ...ticket.ticket.claims,commitment:'x'.repeat(64)}}}).revealContext('hand-1',request,'custodied'),/mismatched/)
})
test('account-authorized final own-random endpoint rejects malformed and mismatched material',async()=>{
 const expected={v:'doudizhu-own-contribution-v1',fairnessMode:'client-contributed-server-custodied-v1',
  handId:'hand-1',seat:1,secureRandom:'A'.repeat(43),commitment:'a'.repeat(64),
  contributionDigest:'b'.repeat(64)}
 const calls=[]
 const api=data=>new DoudizhuHTTPClient({request:async(path,opts)=>{
  calls.push({path,opts});return {code:'OK',data}
 }},()=> 'private-access-token')
 assert.equal((await api(expected).ownContribution('hand-1')).handId,'hand-1')
 assert.equal(calls[0].path,'/v1/doudizhu/hands/hand-1/own-contribution')
 assert.equal(calls[0].opts.accessToken,'private-access-token')
 await assert.rejects(api({...expected,handId:'hand-other'}).ownContribution('hand-1'),/invalid own/)
 await assert.rejects(api({...expected,phrase:'should never be returned',secureRandom:'bad'})
   .ownContribution('hand-1'),/invalid own/)
})

test('recoverable rooms are authenticated, membership-scoped and strictly parsed',async()=>{
 const own={roomId:'room-cold',version:8,status:'HAND_ACTIVE',occupied:3,
  member:true,lastClientSequence:6,activeHandId:'hand-cold',seats:[
    {position:1,accountId:'a',ready:false},{position:2,accountId:'b',ready:false},
    {position:3,accountId:'c',ready:false}]}
 const calls=[]
 const api=(data,token='member-token')=>new DoudizhuHTTPClient({
  request:async(path,options)=>{calls.push({path,options});return {code:'OK',data}}
 },()=>token)
 assert.deepEqual(await api({rooms:[own]}).recoverableRooms(),[own])
 assert.equal(calls[0].path,'/v1/doudizhu/me/rooms')
 assert.equal(calls[0].options.accessToken,'member-token')
 await assert.rejects(api({rooms:[{...own,member:false}]}).recoverableRooms(),/invalid|unauthorized|private membership/)
 await assert.rejects(api({rooms:[own,own]}).recoverableRooms(),/invalid/)
 await assert.rejects(api({rooms:[{...own,activeHandId:'hand-cold',seats:undefined}]}).recoverableRooms(),/invalid/)
 await assert.rejects(api({rooms:[{...own,status:'CLOSED'}]}).recoverableRooms(),/unauthorized/)
 await assert.rejects(api({rooms:[own]},'').recoverableRooms(),/token required/)
})
