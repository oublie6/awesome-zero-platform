import {test} from 'node:test'
import assert from 'node:assert/strict'
import {GameOnlineController} from '../dist/game/GameOnlineController.js'

function fixture({failLogin=false,roomMember=true}={}) {
 const commands=[], messages=new Set(), closes=new Set(), auth=[]
 const platform={
  target:'wechat',
  auth:{acquireCode:async()=> 'temporary-wx-code'},
  vault:{load:async()=>null,save:async()=>{},clear:async()=>{}},
  http:{request:async()=>({code:'OK',data:roomMember
   ? {roomId:'room-rejoin',version:9,status:'WAITING_PLAYERS',occupied:2,member:true,lastClientSequence:12,seats:[{position:1,accountId:'actor',ready:false},{position:2,accountId:'friend',ready:true},{position:3,accountId:'',ready:false}]}
   : {roomId:'room-rejoin',version:9,status:'WAITING_PLAYERS',occupied:2,member:false,lastClientSequence:0}})},
  random:{bytes:async n=>new Uint8Array(Array.from({length:n},(_,i)=>i))},
  socket:{connect:async(url,token)=>{auth.push({url,token});return {
   send:async raw=>commands.push(JSON.parse(raw)),close:()=>{},
   onMessage:cb=>{messages.add(cb);return()=>messages.delete(cb)},
   onClose:cb=>{closes.add(cb);return()=>closes.delete(cb)}
  }}}
 }
 const identity={exchange:async(target,code)=>{
  assert.equal(target,'wechat');assert.equal(code,'temporary-wx-code')
  if(failLogin)throw Error('exchange failed')
  return {accessToken:'server-validated-token'}
 }}
 return {platform,identity,commands,messages,auth,closes}
}
test('online coordinator requires server identity exchange before connecting socket and sends true room command',async()=>{
 const f=fixture()
 const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
 await online.loginAndConnect()
 assert.deepEqual(f.auth,[{url:'wss://example.org/ws',token:'server-validated-token'}])
 const p=online.rooms.create('room-real')
 await new Promise(resolve=>setImmediate(resolve))
 assert.equal(f.commands.length,1)
 assert.equal(f.commands[0].type,'doudizhu.command')
 assert.equal(f.commands[0].payload.requestId,'cmd-000102030405060708090a0b0c0d0e0f')
 const req=f.commands[0].payload
 for(const cb of f.messages)cb(JSON.stringify({type:'doudizhu.command.result',payload:{
  v:'doudizhu-api-command-result-v1',requestId:req.requestId,type:'room.create',durable:{
   v:'fair-doudizhu-command-result-v1',commandId:req.requestId,aggregateId:'room-real',
   aggregateVersion:1,accepted:true}}}))
 assert.deepEqual(await p,{roomId:'room-real',version:1})
 online.dispose()
 assert.equal(f.messages.size,0)
 assert.equal(f.closes.size,0)
})
test('failed server identity exchange cannot create socket or local room',async()=>{
 const f=fixture({failLogin:true})
 const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
 await assert.rejects(online.loginAndConnect(),/exchange failed/)
 assert.equal(f.auth.length,0)
 assert.equal(online.rooms.state,null)
 online.dispose()
})

test('authenticated room query restores version and monotonic sequence for rejoin',async()=>{
 const f=fixture()
 const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
 await online.loginAndConnect()
 const room=await online.joinRoom('room-rejoin')
 assert.deepEqual(room,{roomId:'room-rejoin',version:9})
 const pending=online.rooms.ready(true)
 await new Promise(resolve=>setImmediate(resolve))
 assert.equal(f.commands[0].payload.clientSeq,13)
 assert.equal(f.commands[0].payload.expectedVersion,9)
 const req=f.commands[0].payload
 for(const cb of f.messages) cb(JSON.stringify({type:'doudizhu.command.result',payload:{
  v:'doudizhu-api-command-result-v1',requestId:req.requestId,type:'room.ready',
  durable:{v:'fair-doudizhu-command-result-v1',commandId:req.requestId,aggregateId:'room-rejoin',aggregateVersion:10,accepted:true}
 }}))
 assert.equal((await pending).version,10)
 online.dispose()
})

test('reconnect reauthenticates the transport and fetches authoritative room version',async()=>{
 const f=fixture()
 const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
 await online.loginAndConnect()
 await online.joinRoom('room-rejoin')
 const latest=await online.reconnectAndResync()
 assert.equal(f.auth.length,2)
 assert.equal(f.auth[1].token,'server-validated-token')
 assert.equal(latest.roomId,'room-rejoin')
 assert.equal(online.rooms.state.version,9)
 assert.equal(f.commands.length,0)
 const pending=online.rooms.ready(true)
 await new Promise(resolve=>setImmediate(resolve))
 assert.equal(f.commands[0].payload.clientSeq,13)
 assert.equal(f.commands[0].payload.expectedVersion,9)
 const request=f.commands[0].payload
 for(const cb of f.messages)cb(JSON.stringify({type:'doudizhu.command.result',payload:{
   v:'doudizhu-api-command-result-v1',requestId:request.requestId,type:'room.ready',
   durable:{v:'fair-doudizhu-command-result-v1',commandId:request.requestId,aggregateId:'room-rejoin',aggregateVersion:10,accepted:true}
 }}))
 assert.equal((await pending).version,10)
 online.dispose()
})
test('reconnect cannot silently restore an unconfirmed room membership',async()=>{
 const f=fixture({roomMember:false})
 const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
 await online.loginAndConnect()
 await assert.rejects(online.reconnectAndResync('room-rejoin'),/membership not confirmed/)
 assert.equal(online.rooms.state,null)
 online.dispose()
})

test('room exit is confirmed before local room identity is cleared',async()=>{
  const f=fixture()
  const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
  await online.loginAndConnect()
  online.rooms.restoreTrustedRoom('room-rejoin',9,12)
  const pending=online.leaveRoom()
  await new Promise(resolve=>setImmediate(resolve))
  assert.equal(f.commands.length,1)
  const cmd=f.commands[0].payload
  assert.equal(cmd.type,'room.leave')
  assert.equal(cmd.clientSeq,13)
  assert.equal(online.rooms.state.roomId,'room-rejoin')
  for(const cb of f.messages)cb(JSON.stringify({type:'doudizhu.command.result',payload:{
    v:'doudizhu-api-command-result-v1',requestId:cmd.requestId,type:'room.leave',
    durable:{v:'fair-doudizhu-command-result-v1',commandId:cmd.requestId,
      aggregateId:'room-rejoin',aggregateVersion:10,accepted:true}
  }}))
  await pending
  assert.equal(online.rooms.state,null)
  assert.equal(online.shell.hand.snapshot,null)
  online.dispose()
})
