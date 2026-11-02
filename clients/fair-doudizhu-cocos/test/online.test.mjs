import {test} from 'node:test'
import assert from 'node:assert/strict'
import {GameOnlineController} from '../dist/game/GameOnlineController.js'

function fixture({failLogin=false}={}) {
 const commands=[], messages=new Set(), closes=new Set(), auth=[]
 const platform={
  target:'wechat',
  auth:{acquireCode:async()=> 'temporary-wx-code'},
  vault:{load:async()=>null,save:async()=>{},clear:async()=>{}},
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
