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

test('replacement room member reconnects without reading a previous hand archive',async()=>{
  const f=fixture()
  const paths=[]
  f.platform.http.request=async path=>{
    paths.push(path)
    if(path.includes('/evidence'))throw new Error('prior hand evidence must never be requested')
    return {code:'OK',data:{
      roomId:'room-rejoin',version:11,status:'WAITING_PLAYERS',occupied:3,
      member:true,lastClientSequence:2,seats:[
        {position:1,accountId:'account-A',ready:false},
        {position:2,accountId:'account-B',ready:false},
        {position:3,accountId:'account-D',ready:false}
      ]
      // The previous hand exists, but server deliberately omits lastHandId
      // because account D was not seated during its completed game.
    }}
  }
  const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
  await online.loginAndConnect()
  await online.joinRoom('room-rejoin')
  const confirmed=await online.reconnectAndResync()
  assert.equal(confirmed.member,true)
  assert.equal(confirmed.lastHandId,undefined)
  assert.equal(online.shell.hand.snapshot,null)
  assert.equal(online.rooms.state.roomId,'room-rejoin')
  assert.equal(paths.length,4)
  assert.ok(paths.every(path=>path.endsWith('/rooms/room-rejoin')))
  online.dispose()
})

test('confirmed leave securely invalidates pending old fairness preparation and permits next hand',async()=>{
  const f=fixture()
  const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
  await online.loginAndConnect()
  await online.joinRoom('room-rejoin')
  online.confirmActiveFairnessHand('hand-aborted')
  const old=online.fairness
  let deliverOld
  online.api.fairness=handId=>handId==='hand-aborted'
    ? new Promise(resolve=>{deliverOld=resolve})
    : Promise.resolve({handId,phase:'FAIRNESS_COMMITTING',version:1,seat:1,
       committed:false,revealed:false,lastClientSequence:0})
  const late=old.prepareRecoveryKit('hand-aborted')
  const pending=online.leaveRoom()
  await new Promise(resolve=>setImmediate(resolve))
  const req=f.commands.at(-1).payload
  assert.equal(req.type,'room.leave')
  for(const cb of f.messages)cb(JSON.stringify({type:'doudizhu.command.result',payload:{
    v:'doudizhu-api-command-result-v1',requestId:req.requestId,type:'room.leave',
    durable:{v:'fair-doudizhu-command-result-v1',commandId:req.requestId,
      aggregateId:'room-rejoin',aggregateVersion:10,accepted:true}
  }}))
  await pending
  assert.notEqual(online.fairness,old)
  assert.equal(old.hasRecoverySecret,false)
  deliverOld({handId:'hand-aborted',phase:'FAIRNESS_COMMITTING',version:2,seat:1,
    committed:false,revealed:false,lastClientSequence:0})
  await assert.rejects(late,/disposed/)
  assert.equal(old.hasRecoverySecret,false)
  online.confirmActiveFairnessHand('hand-new')
  const newKit=await online.fairness.prepareRecoveryKit('hand-new')
  assert.ok(newKit.length>48)
  assert.equal(online.fairness.hasRecoverySecret,true)
  online.dispose()
})

test('authoritatively changed active hand releases prior cancelled secret, same hand preserves it',async()=>{
  const f=fixture()
  const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
  online.api.fairness=async handId=>({handId,phase:'FAIRNESS_COMMITTING',version:1,seat:1,
    committed:false,revealed:false,lastClientSequence:0})
  online.confirmActiveFairnessHand('hand-cancelled')
  const first=online.fairness
  await first.prepareRecoveryKit('hand-cancelled')
  assert.equal(first.hasRecoverySecret,true)
  online.confirmActiveFairnessHand('hand-cancelled')
  assert.equal(online.fairness,first)
  assert.equal(first.hasRecoverySecret,true)
  online.confirmActiveFairnessHand(null) // authenticated room read: no active hand
  assert.equal(first.hasRecoverySecret,false)
  assert.notEqual(online.fairness,first)
  online.confirmActiveFairnessHand('hand-next')
  const kit=await online.fairness.prepareRecoveryKit('hand-next')
  assert.ok(kit.length>48)
  online.dispose()
})

test('unknown fairness command is never erased by a hand switch or leave attempt',async()=>{
  const f=fixture()
  const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
  await online.loginAndConnect()
  await online.joinRoom('room-rejoin')
  online.confirmActiveFairnessHand('hand-unknown')
  online.api.fairness=async()=>({handId:'hand-unknown',phase:'FAIRNESS_COMMITTING',
    version:1,seat:1,committed:false,revealed:false,lastClientSequence:0})
  online.api.command=async()=>{throw Error('network timeout')}
  await assert.rejects(online.fairness.commit('hand-unknown','secret'),/network timeout/)
  assert.equal(online.fairness.needsReconciliation,true)
  const original=online.fairness
  await assert.rejects(online.leaveRoom(),/fairness outcome unknown/)
  assert.throws(()=>online.confirmActiveFairnessHand('hand-new'),/result unknown/)
  assert.equal(online.fairness,original)
  assert.equal(original.hasRecoverySecret,true)
  assert.equal(original.needsReconciliation,true)
  assert.equal(f.commands.length,0)
  online.dispose()
})

test('Round15 exact fairness reconciliation precedes ACKed new hand and adopts new state',async()=>{
  const f=fixture()
  const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
  await online.loginAndConnect()
  await online.joinRoom('room-rejoin')
  online.confirmActiveFairnessHand('hand-cancelled')
  const old=online.fairness
  const fairnessCalls=[]
  online.api.fairness=async()=>({handId:'hand-cancelled',phase:'FAIRNESS_COMMITTING',
    version:3,seat:1,committed:false,revealed:false,lastClientSequence:4})
  online.api.command=async command=>{
    fairnessCalls.push(command)
    if(fairnessCalls.length===1)throw Error('lost fairness response')
    return {v:'doudizhu-api-command-result-v1',requestId:command.requestId,type:command.type,
      durable:{v:'fair-doudizhu-command-result-v1',commandId:command.requestId,accepted:true}}
  }
  await assert.rejects(old.commit('hand-cancelled','offline phrase'),/lost fairness response/)
  assert.equal(old.needsReconciliation,true)
  const pending=online.startHand('hand-next')
  await new Promise(resolve=>setImmediate(resolve))
  assert.equal(fairnessCalls.length,2)
  assert.deepEqual(fairnessCalls[0],fairnessCalls[1])
  assert.equal(f.commands.length,1)
  assert.equal(f.commands[0].payload.type,'room.hand.start')
  assert.equal(f.commands[0].payload.payload.handId,'hand-next')
  assert.equal(online.rooms.state.version,9)
  assert.equal(online.fairness,old)
  await assert.rejects(old.prepareRecoveryKit('hand-cancelled'),/in progress/)
  const request=f.commands[0].payload
  for(const cb of f.messages)cb(JSON.stringify({type:'doudizhu.command.result',payload:{
    v:'doudizhu-api-command-result-v1',requestId:request.requestId,type:'room.hand.start',
    durable:{v:'fair-doudizhu-command-result-v1',commandId:request.requestId,
      aggregateId:'room-rejoin',aggregateVersion:10,accepted:true}
  }}))
  const result=await pending
  assert.equal(result.version,10)
  assert.equal(online.rooms.state.version,10)
  assert.equal(old.hasRecoverySecret,false)
  assert.notEqual(online.fairness,old)
  assert.equal(online.fairness.needsReconciliation,false)
  online.dispose()
})

test('Round15 unresolved fairness response blocks room.hand.start before transmission',async()=>{
  const f=fixture()
  const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
  await online.loginAndConnect()
  await online.joinRoom('room-rejoin')
  online.confirmActiveFairnessHand('hand-old')
  const old=online.fairness
  online.api.fairness=async()=>({handId:'hand-old',phase:'FAIRNESS_COMMITTING',
    version:1,seat:1,committed:false,revealed:false,lastClientSequence:0})
  online.api.command=async()=>{throw Error('no authoritative response')}
  await assert.rejects(old.commit('hand-old','backup'),/no authoritative response/)
  const secret=old.exportRecoveryKit()
  await assert.rejects(online.startHand('hand-new'),/no authoritative response/)
  assert.equal(f.commands.length,0)
  assert.equal(online.rooms.state.version,9)
  assert.equal(online.fairness,old)
  assert.equal(old.exportRecoveryKit(),secret)
  assert.equal(old.needsReconciliation,true)
  online.dispose()
})

test('Round15 definitive prior fairness rejection permits a new ACKed hand without losing isolation',async()=>{
  const f=fixture()
  const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
  await online.loginAndConnect()
  await online.joinRoom('room-rejoin')
  online.confirmActiveFairnessHand('hand-cancelled')
  const old=online.fairness
  online.api.fairness=async()=>({handId:'hand-cancelled',phase:'FAIRNESS_COMMITTING',
    version:1,seat:1,committed:false,revealed:false,lastClientSequence:0})
  let count=0
  online.api.command=async command=>{
    if(++count===1)throw Error('network timeout')
    return {v:'doudizhu-api-command-result-v1',requestId:command.requestId,type:command.type,
      durable:{v:'fair-doudizhu-command-result-v1',commandId:command.requestId,
        accepted:false,failure:{code:'DDZ_VERSION_CONFLICT'}}}
  }
  await assert.rejects(old.commit('hand-cancelled','secret'),/network timeout/)
  const pending=online.startHand('hand-new')
  await new Promise(resolve=>setImmediate(resolve))
  assert.equal(count,2)
  assert.equal(old.needsReconciliation,false)
  assert.equal(f.commands.length,1)
  const request=f.commands[0].payload
  for(const cb of f.messages)cb(JSON.stringify({type:'doudizhu.command.result',payload:{
    v:'doudizhu-api-command-result-v1',requestId:request.requestId,type:'room.hand.start',
    durable:{v:'fair-doudizhu-command-result-v1',commandId:request.requestId,
      aggregateId:'room-rejoin',aggregateVersion:10,accepted:true}
  }}))
  await pending
  assert.equal(old.hasRecoverySecret,false)
  online.dispose()
})

test('Round15 active fairness operation blocks irreversible new-hand command',async()=>{
  const f=fixture()
  const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
  await online.loginAndConnect()
  await online.joinRoom('room-rejoin')
  online.confirmActiveFairnessHand('hand-old')
  let respond
  online.api.fairness=()=>new Promise(resolve=>{respond=resolve})
  const preparing=online.fairness.prepareRecoveryKit('hand-old')
  await assert.rejects(online.startHand('hand-new'),/action in progress/)
  assert.equal(f.commands.length,0)
  respond({handId:'hand-old',phase:'FAIRNESS_COMMITTING',version:1,
    seat:1,committed:false,revealed:false,lastClientSequence:0})
  await preparing
  assert.equal(online.fairness.hasRecoverySecret,true)
  online.dispose()
})

test('Round15 concurrent old authoritative refresh cannot invalidate an ACKed new hand',async()=>{
  const f=fixture()
  const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
  await online.loginAndConnect()
  await online.joinRoom('room-rejoin')
  online.confirmActiveFairnessHand('hand-aborted')
  const old=online.fairness
  online.api.fairness=async handId=>({handId,phase:'FAIRNESS_COMMITTING',
    version:2,seat:1,committed:false,revealed:false,lastClientSequence:0})
  await old.prepareRecoveryKit('hand-aborted','external user backup')
  const start=online.startHand('hand-new')
  await new Promise(resolve=>setImmediate(resolve))
  assert.equal(f.commands.length,1)
  // A stale room read with no active hand must not replace the session
  // reserved for the already-transmitted hand-start command.
  assert.throws(()=>online.confirmActiveFairnessHand(null),/start in progress/)
  assert.equal(online.fairness,old)
  assert.equal(old.hasRecoverySecret,true)
  const request=f.commands[0].payload
  for(const cb of f.messages)cb(JSON.stringify({type:'doudizhu.command.result',payload:{
    v:'doudizhu-api-command-result-v1',requestId:request.requestId,type:'room.hand.start',
    durable:{v:'fair-doudizhu-command-result-v1',commandId:request.requestId,
      aggregateId:'room-rejoin',aggregateVersion:10,accepted:true}
  }}))
  await start
  assert.equal(online.rooms.state.version,10)
  assert.notEqual(online.fairness,old)
  assert.equal(old.hasRecoverySecret,false)
  online.confirmActiveFairnessHand('hand-new')
  online.dispose()
})

test('Round15 failed room start retains original fairness secret and lifts reservation',async()=>{
  const f=fixture()
  const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
  await online.loginAndConnect()
  await online.joinRoom('room-rejoin')
  online.confirmActiveFairnessHand('hand-aborted')
  const old=online.fairness
  online.api.fairness=async handId=>({handId,phase:'FAIRNESS_COMMITTING',
    version:2,seat:1,committed:false,revealed:false,lastClientSequence:0})
  const secret=await old.prepareRecoveryKit('hand-aborted','safe backup')
  // Preflight succeeds, but no room command is accepted.
  online.rooms.startHand=async()=>{throw Error('pre-ACK room failure')}
  await assert.rejects(online.startHand('hand-new'),/pre-ACK room failure/)
  assert.equal(old.exportRecoveryKit(),secret)
  assert.equal(online.fairness,old)
  assert.equal(online.rooms.state.version,9)
  assert.equal(f.commands.length,0)
  assert.equal(await old.prepareRecoveryKit('hand-aborted','safe backup'),secret)
  online.dispose()
})

test('Round15 live pending preflight blocks new room.hand.start before ACK',async()=>{
  const f=fixture()
  const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
  await online.loginAndConnect()
  await online.joinRoom('room-rejoin')
  online.confirmActiveFairnessHand('hand-old')
  // Reproduce the state that GameHandActions obtains on an actual lost
  // Bid/Play/Pass response; the GameHandActions unit suite drives the
  // preceding send/timeout to establish the pending state independently.
  const pendingAction={requestId:'cmd-original',type:'hand.pass',
    aggregateId:'hand-old',clientSeq:1,expectedVersion:7,payload:{}}
  online.actions.pending=pendingAction
  await assert.rejects(online.startHand('hand-new'),/live hand outcome unknown/)
  assert.equal(f.commands.length,0)
  assert.equal(online.rooms.state.version,9)
  assert.equal(online.actions.pending,pendingAction)
  online.dispose()
})

test('Round16 early authoritative H2 HTTP refresh cannot replace reserved fairness before WSS ACK',async()=>{
  const f=fixture()
  const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
  await online.loginAndConnect()
  await online.joinRoom('room-rejoin')
  online.confirmActiveFairnessHand('hand-cancelled')
  const previous=online.fairness
  let projections=0
  online.api.fairness=async handId=>{
    projections++
    return {handId,phase:'FAIRNESS_COMMITTING',version:1,seat:1,
      committed:false,revealed:false,lastClientSequence:0}
  }
  const oldKit=await previous.prepareRecoveryKit('hand-cancelled','user-held backup')
  assert.ok(oldKit.length>48)
  const start=online.startHand('hand-H2')
  await new Promise(resolve=>setImmediate(resolve))
  assert.equal(f.commands.length,1)
  assert.equal(f.commands[0].payload.type,'room.hand.start')
  assert.equal(online.handStartInProgress,true)
  // The real authenticated HTTP room projection can expose H2 before the
  // corresponding WSS durable ACK is delivered to RoomCoordinator.
  f.platform.http.request=async()=>({code:'OK',data:{
    roomId:'room-rejoin',version:10,status:'HAND_ACTIVE',occupied:3,member:true,
    activeHandId:'hand-H2',lastClientSequence:13,seats:[
      {position:1,accountId:'actor',ready:false},
      {position:2,accountId:'friend',ready:false},
      {position:3,accountId:'third',ready:false}
    ]
  }})
  const ahead=await online.roomInfo('room-rejoin')
  assert.equal(ahead.activeHandId,'hand-H2')
  assert.throws(()=>online.confirmActiveFairnessHand(ahead.activeHandId),/room hand start in progress/)
  assert.equal(online.fairness,previous,'HTTP H2 must not replace reserved old session')
  assert.equal(previous.exportRecoveryKit(),oldKit,'old secret remains held until ACK')
  const before=projections
  await assert.rejects(online.fairness.prepareRecoveryKit('hand-H2'),/in progress/)
  await assert.rejects(online.fairness.commit('hand-H2'),/in progress/)
  assert.equal(projections,before,'no H2 fairness projection may start before ACK')
  assert.equal(online.rooms.state.version,9,'in-flight room coordinator remains untouched')
  assert.equal(online.handStartInProgress,true)
  const req=f.commands[0].payload
  for(const cb of f.messages)cb(JSON.stringify({type:'doudizhu.command.result',payload:{
    v:'doudizhu-api-command-result-v1',requestId:req.requestId,type:'room.hand.start',
    durable:{v:'fair-doudizhu-command-result-v1',commandId:req.requestId,
      aggregateId:'room-rejoin',aggregateVersion:10,accepted:true}
  }}))
  const accepted=await start
  assert.equal(accepted.version,10)
  assert.equal(online.rooms.state.version,10)
  assert.equal(online.handStartInProgress,false)
  assert.notEqual(online.fairness,previous)
  assert.equal(previous.hasRecoverySecret,false)
  assert.equal(online.fairness.hasRecoverySecret,false)
  const newKit=await online.fairness.prepareRecoveryKit('hand-H2')
  assert.ok(newKit.length>48)
  assert.notEqual(newKit,oldKit)
  online.dispose()
})

test('Round16 failed hand start releases controller lock without clearing original secret',async()=>{
  const f=fixture()
  const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
  await online.loginAndConnect()
  await online.joinRoom('room-rejoin')
  online.confirmActiveFairnessHand('hand-cancelled')
  online.api.fairness=async handId=>({handId,phase:'FAIRNESS_COMMITTING',
    version:1,seat:1,committed:false,revealed:false,lastClientSequence:0})
  const first=online.fairness
  const backup=await first.prepareRecoveryKit('hand-cancelled','external backup')
  online.rooms.startHand=async()=>{throw Error('authoritative start rejected')}
  await assert.rejects(online.startHand('hand-H2'),/authoritative start rejected/)
  assert.equal(online.handStartInProgress,false)
  assert.equal(online.fairness,first)
  assert.equal(first.exportRecoveryKit(),backup)
  online.confirmActiveFairnessHand(null)
  assert.equal(first.hasRecoverySecret,false)
  online.dispose()
})

test('cold controller discovers the authenticated active room without any device game locator',async()=>{
 const f=fixture()
 const room={roomId:'room-cold',version:18,status:'HAND_ACTIVE',occupied:3,
  member:true,lastClientSequence:9,activeHandId:'hand-cold',seats:[
   {position:1,accountId:'actor',ready:false},{position:2,accountId:'friend',ready:false},
   {position:3,accountId:'third',ready:false}]}
 const paths=[]
 f.platform.http.request=async path=>{
  paths.push(path)
  return {code:'OK',data:path==='/v1/doudizhu/me/rooms'?{rooms:[room]}:room}
 }
 const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
 await online.loginAndConnect()
 const synced=[]
 let phase='FAIRNESS_COMMITTING'
 online.api.fairness=async()=>({handId:'hand-cold',phase})
 online.shell.syncHand=async id=>{synced.push(id)}
 assert.equal(online.rooms.state,null)
 const result=await online.reconnectAndResync()
 assert.equal(result.roomId,'room-cold')
 assert.equal(online.rooms.state.roomId,'room-cold')
 assert.equal(online.rooms.state.version,18)
 assert.deepEqual(synced,[],'predeal fairness is recoverable without an unavailable private hand')
 assert.deepEqual(paths,['/v1/doudizhu/me/rooms','/v1/doudizhu/rooms/room-cold','/v1/doudizhu/rooms/room-cold'])
 const joined=await online.joinRoom('room-cold')
 assert.equal(joined.roomId,'room-cold','already seated HAND_ACTIVE player must be able to resume')
 assert.deepEqual(synced,[])
 phase='PLAYING'
 await online.reconnectAndResync('room-cold')
 assert.deepEqual(synced,['hand-cold'],'dealt private hand is synced after the authoritative phase advances')
 online.dispose()
})
test('cold recovery refuses ambiguous active rooms instead of choosing wrong game',async()=>{
 const f=fixture()
 const v=id=>({roomId:id,version:1,status:'HAND_ACTIVE',occupied:3,member:true,
  lastClientSequence:1,activeHandId:'hand-'+id,seats:[
  {position:1,accountId:'actor',ready:false},{position:2,accountId:'b',ready:false},
  {position:3,accountId:'c',ready:false}]})
 f.platform.http.request=async()=>({code:'OK',data:{rooms:[v('first'),v('second')]}})
 const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
 await online.loginAndConnect()
 await assert.rejects(online.reconnectAndResync(),/multiple active rooms/)
 assert.equal(online.rooms.state,null)
 assert.equal((await online.discoverRecoverableRooms()).length,2)
 online.dispose()
})
test('nonmember may not join an already active hand through a stale locator',async()=>{
 const f=fixture({roomMember:false})
 f.platform.http.request=async()=>({code:'OK',data:{
  roomId:'room-rejoin',version:12,status:'HAND_ACTIVE',occupied:3,
  member:false,lastClientSequence:0}})
 const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
 await online.loginAndConnect()
 await assert.rejects(online.joinRoom('room-rejoin'),/not joinable/)
 assert.equal(online.rooms.state,null)
 online.dispose()
})

test('R19-N01: older A HTTP response cannot overwrite newer explicit room B recovery',async()=>{
 const f=fixture()
 const room=id=>({roomId:id,version:5,status:'WAITING_PLAYERS',occupied:2,member:true,
  lastClientSequence:3,seats:[{position:1,accountId:'actor',ready:false},
   {position:2,accountId:'friend',ready:false},{position:3,accountId:'',ready:false}]})
 let releaseA,firstA=true
 const urls=[]
 f.platform.http.request=async path=>{
  urls.push(path)
  if(path==='/v1/doudizhu/rooms/A'&&firstA){
   firstA=false
   return new Promise(resolve=>{releaseA=()=>resolve({code:'OK',data:room('A')})})
  }
  return {code:'OK',data:room(path.endsWith('/B')?'B':'A')}
 }
 const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
 await online.loginAndConnect()
 const pendingA=online.reconnectAndResync('A')
 await new Promise(resolve=>setImmediate(resolve))
 assert.equal(typeof releaseA,'function')
 const b=await online.reconnectAndResync('B')
 assert.equal(b.roomId,'B')
 assert.equal(online.rooms.state.roomId,'B')
 releaseA()
 assert.equal(await pendingA,null,'older result must lose ownership')
 assert.equal(online.rooms.state.roomId,'B')
 assert.equal(online.rooms.state.version,5)
 assert.ok(urls.filter(url=>url.endsWith('/rooms/A')).length===1)
 online.dispose()
})

test('R19-N01: duplicate same-room recovery coalesces and creates one reconnect socket',async()=>{
 const f=fixture()
 const view={roomId:'room-single',version:4,status:'WAITING_PLAYERS',member:true,
   occupied:2,lastClientSequence:1,seats:[
    {position:1,accountId:'actor',ready:false},{position:2,accountId:'b',ready:true},
    {position:3,accountId:'',ready:false}]}
 const urls=[]
 let release
 f.platform.http.request=async path=>{
  urls.push(path)
  if(urls.length===1)return new Promise(resolve=>{release=()=>resolve({code:'OK',data:view})})
  return {code:'OK',data:view}
 }
 const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
 await online.loginAndConnect()
 const a=online.reconnectAndResync('room-single')
 const b=online.reconnectAndResync('room-single')
 assert.strictEqual(a,b,'same target must share the exact pending operation')
 assert.equal(f.auth.length,2,'login + one recovery connection')
 await new Promise(resolve=>setImmediate(resolve))
 assert.equal(typeof release,'function','first room read is now pending')
 release()
 const result=await Promise.all([a,b])
 assert.equal(result[0].roomId,'room-single')
 assert.equal(urls.length,2,'a second fresh authorization read, not duplicate recoveries')
 assert.equal(f.auth.length,2)
 online.dispose()
})

test('R19-N02: fresh CLOSED/nonmember read rejects stale WAITING and never adopts it',async()=>{
 const f=fixture()
 const waiting={roomId:'room-closed',version:1,status:'WAITING_PLAYERS',member:true,occupied:1,
   lastClientSequence:1,seats:[{position:1,accountId:'actor',ready:false},
     {position:2,accountId:'',ready:false},{position:3,accountId:'',ready:false}]}
 const closed={roomId:'room-closed',version:2,status:'CLOSED',member:false,occupied:0,lastClientSequence:0}
 let n=0
 f.platform.http.request=async()=>({code:'OK',data:++n===1?waiting:closed})
 const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
 await online.loginAndConnect()
 await assert.rejects(online.reconnectAndResync('room-closed'),/room closed|membership revoked/)
 assert.equal(n,2,'must verify twice against actual authoritative room endpoint')
 assert.equal(online.rooms.state,null)
 assert.equal(online.shell.hand.snapshot,null)
 online.dispose()
})

test('R19-N02: later definitive CLOSED revokes an already adopted room',async()=>{
 const f=fixture()
 const waiting={roomId:'room-revoked',version:5,status:'WAITING_PLAYERS',member:true,occupied:1,
   lastClientSequence:2,seats:[{position:1,accountId:'actor',ready:false},
     {position:2,accountId:'',ready:false},{position:3,accountId:'',ready:false}]}
 const closed={roomId:'room-revoked',version:6,status:'CLOSED',member:false,occupied:0,lastClientSequence:0}
 let reads=0
 f.platform.http.request=async()=>({code:'OK',data:++reads<4?waiting:closed})
 const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
 await online.loginAndConnect()
 await online.reconnectAndResync('room-revoked')
 assert.equal(online.rooms.state.roomId,'room-revoked')
 await assert.rejects(online.reconnectAndResync('room-revoked'),/room closed|membership revoked/)
 assert.equal(online.rooms.state,null)
 online.dispose()
})

test('G47-N01: delayed member join A cannot overwrite newer recovery B',async()=>{
 const f=fixture()
 const room=id=>({roomId:id,version:1,status:'WAITING_PLAYERS',member:true,occupied:1,lastClientSequence:1,seats:[{position:1,accountId:'actor',ready:false},{position:2,accountId:'',ready:false},{position:3,accountId:'',ready:false}]})
 let releaseA,firstA=true
 f.platform.http.request=async path=>{
  if(path==='/v1/doudizhu/rooms/A' && firstA){
   firstA=false
   return new Promise(resolve=>{releaseA=()=>resolve({code:'OK',data:room('A')})})
  }
  return {code:'OK',data:room(path.endsWith('/B')?'B':'A')}
 }
 const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
 await online.loginAndConnect()
 const old=online.joinRoom('A')
 await new Promise(resolve=>setImmediate(resolve))
 assert.equal(typeof releaseA,'function')
 const newer=await online.reconnectAndResync('B')
 assert.equal(newer.roomId,'B')
 releaseA()
 await assert.rejects(old,/selection superseded/)
 assert.equal(online.rooms.state.roomId,'B')
 assert.equal(f.commands.length,0)
 online.dispose()
})
test('G47-N01: late CLOSED recheck prevents stale existing-member join',async()=>{
 const f=fixture()
 const waiting={roomId:'A',version:1,status:'WAITING_PLAYERS',member:true,occupied:1,lastClientSequence:1,seats:[{position:1,accountId:'actor',ready:false},{position:2,accountId:'',ready:false},{position:3,accountId:'',ready:false}]}
 const closed={roomId:'A',version:2,status:'CLOSED',member:false,occupied:0,lastClientSequence:0}
 let reads=0
 f.platform.http.request=async()=>({code:'OK',data:++reads===1?waiting:closed})
 const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
 await online.loginAndConnect()
 await assert.rejects(online.joinRoom('A'),/membership revoked|room closed/)
 assert.equal(reads,2)
 assert.equal(online.rooms.state,null)
 online.dispose()
})

test('G48-N02: controller refuses cross-room recovery while exact join result is UNKNOWN',async()=>{
 const f=fixture()
 let lostRequest
 const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
 await online.loginAndConnect()
 const original=online.shell.submitCommand.bind(online.shell)
 online.shell.submitCommand=async command=>{lostRequest=command;throw Error('ACK lost')}
 await assert.rejects(online.rooms.join('A',5),/ACK lost/)
 assert.equal(online.rooms.pendingRoomId,'A')
 assert.equal(online.rooms.needsReconciliation,true)
 await assert.rejects(online.reconnectAndResync('B'),/unknown for another room/)
 await assert.rejects(online.joinRoom('B'),/unknown for another room/)
 assert.equal(online.rooms.pendingRoomId,'A')
 assert.equal(online.rooms.state,null)
 assert.equal(online.rooms.needsReconciliation,true)
 assert.equal(lostRequest.aggregateId,'A')
 online.shell.submitCommand=original
 online.dispose()
})
test('G48-N03: logout revokes delayed member join before any room adoption',async()=>{
 const f=fixture()
 const waiting={roomId:'A',version:1,status:'WAITING_PLAYERS',member:true,occupied:1,
   lastClientSequence:1,seats:[{position:1,accountId:'actor',ready:false},
     {position:2,accountId:'',ready:false},{position:3,accountId:'',ready:false}]}
 let release
 let reads=0
 f.platform.http.request=async()=> {
   if(++reads===2)return new Promise(resolve=>{release=()=>resolve({code:'OK',data:waiting})})
   return {code:'OK',data:waiting}
 }
 const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
 await online.loginAndConnect()
 const pending=online.joinRoom('A')
 await new Promise(resolve=>setImmediate(resolve))
 assert.equal(typeof release,'function')
 await online.logout()
 assert.equal(online.session.authenticated,false)
 release()
 await assert.rejects(pending,/selection superseded/)
 assert.equal(online.rooms.state,null)
})

test('G49-N01: delayed same-room recovery cannot settle a newer unknown ready',async()=>{
 const f=fixture()
 const view={roomId:'A',version:1,status:'WAITING_PLAYERS',member:true,occupied:1,lastClientSequence:1,
  seats:[{position:1,accountId:'actor',ready:false},{position:2,accountId:'',ready:false},
    {position:3,accountId:'',ready:false}]}
 let release,reads=0
 f.platform.http.request=async()=>{
  if(++reads===2)return new Promise(resolve=>{release=()=>resolve({code:'OK',data:view})})
  return {code:'OK',data:view}
 }
 const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
 await online.loginAndConnect()
 online.rooms.restoreTrustedRoom('A',1,1)
 const recovery=online.reconnectAndResync('A')
 await new Promise(resolve=>setImmediate(resolve))
 assert.equal(typeof release,'function')
 online.shell.submitCommand=async()=>{throw Error('lost ACK')}
 await assert.rejects(online.rooms.ready(true),/lost ACK/)
 const requestRoom=online.rooms.pendingRoomId
 release()
 assert.equal(await recovery,null)
 assert.equal(online.rooms.pendingRoomId,requestRoom)
 assert.equal(online.rooms.needsReconciliation,true)
 assert.equal(online.rooms.state.version,1)
 online.dispose()
})
test('G49-N02: fairness transition rejection cannot partially adopt another room',async()=>{
 const f=fixture()
 const room=id=>({roomId:id,version:1,status:'WAITING_PLAYERS',member:true,occupied:1,
  lastClientSequence:1,seats:[{position:1,accountId:'actor',ready:false},
    {position:2,accountId:'',ready:false},{position:3,accountId:'',ready:false}]})
 f.platform.http.request=async path=>({code:'OK',data:room(path.endsWith('/B')?'B':'A')})
 const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
 await online.loginAndConnect()
 online.rooms.restoreTrustedRoom('A',1,1)
 const before=online.fairness
 before.beginHandTransition=()=>{throw Error('fairness result unknown; retry the original request before changing hand')}
 await assert.rejects(online.reconnectAndResync('B'),/fairness result unknown/)
 assert.equal(online.rooms.state.roomId,'A')
 assert.equal(online.fairness,before)
 online.dispose()
})

test('G50-N01: delayed member join cannot overwrite a newer confirmed ready ACK',async()=>{
 const f=fixture()
 const waiting={roomId:'A',version:1,status:'WAITING_PLAYERS',member:true,occupied:1,
  lastClientSequence:1,seats:[{position:1,accountId:'actor',ready:false},
    {position:2,accountId:'',ready:false},{position:3,accountId:'',ready:false}]}
 let release,reads=0
 f.platform.http.request=async()=> {
  if(++reads===2)return new Promise(resolve=>{release=()=>resolve({code:'OK',data:waiting})})
  return {code:'OK',data:waiting}
 }
 const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
 await online.loginAndConnect()
 online.rooms.restoreTrustedRoom('A',1,1)
 const joining=online.joinRoom('A')
 await new Promise(resolve=>setImmediate(resolve))
 assert.equal(typeof release,'function')
 const ready=online.rooms.ready(true)
 await new Promise(resolve=>setImmediate(resolve))
 const cmd=f.commands.at(-1).payload
 for(const listener of f.messages)listener(JSON.stringify({type:'doudizhu.command.result',payload:{
  v:'doudizhu-api-command-result-v1',requestId:cmd.requestId,type:'room.ready',
  durable:{v:'fair-doudizhu-command-result-v1',commandId:cmd.requestId,
   aggregateId:'A',aggregateVersion:2,accepted:true}}}))
 await ready
 release()
 await assert.rejects(joining,/selection superseded/)
 assert.equal(online.rooms.state.version,2)
 online.dispose()
})
test('G50-N02: room UNKNOWN can be retried from authenticated controller with same request ID',async()=>{
 const f=fixture()
 const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
 await online.loginAndConnect()
 let oldRequest
 const original=online.shell.submitCommand.bind(online.shell)
 online.shell.submitCommand=async cmd=>{oldRequest=cmd;throw Error('ACK lost')}
 await assert.rejects(online.rooms.create('room-pending'),/ACK lost/)
 assert.equal(online.rooms.needsReconciliation,true)
 online.shell.submitCommand=original
 const retry=online.retryUnknownRoom()
 await new Promise(resolve=>setImmediate(resolve))
 const replay=f.commands.at(-1).payload
 assert.deepEqual({...replay,v:undefined},{...oldRequest,v:undefined})
 for(const listener of f.messages)listener(JSON.stringify({type:'doudizhu.command.result',payload:{
  v:'doudizhu-api-command-result-v1',requestId:replay.requestId,type:'room.create',
  durable:{v:'fair-doudizhu-command-result-v1',commandId:replay.requestId,
   aggregateId:'room-pending',aggregateVersion:1,accepted:true}}}))
 assert.deepEqual(await retry,{roomId:'room-pending',version:1})
 assert.equal(online.rooms.needsReconciliation,false)
 online.dispose()
})
test('audit: direct scene refresh cannot partially adopt room when fairness preflight rejects',async()=>{
 const f=fixture()
 const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
 await online.loginAndConnect()
 online.rooms.restoreTrustedRoom('A',2,1)
 const fairness=online.fairness
 const original=fairness.beginHandTransition.bind(fairness)
 fairness.beginHandTransition=()=>{throw Error('fairness result unknown; retry exact request')}
 const view={roomId:'B',version:2,status:'WAITING_PLAYERS',member:true,occupied:1,
  lastClientSequence:1,seats:[{position:1,accountId:'actor',ready:false},
   {position:2,accountId:'',ready:false},{position:3,accountId:'',ready:false}]}
 await assert.rejects(Promise.resolve().then(()=>online.adoptAuthorizedRoomRefresh(view,online.rooms.commandGeneration)),/fairness result unknown/)
 assert.equal(online.rooms.state.roomId,'A')
 assert.equal(online.fairness,fairness)
 fairness.beginHandTransition=original
 online.dispose()
})

test('G53-N01: a new A recovery after a real member join B must not reuse obsolete core A',async()=>{
 const f=fixture()
 const room=id=>({roomId:id,version:1,status:'WAITING_PLAYERS',member:true,occupied:1,
  lastClientSequence:1,seats:[{position:1,accountId:'actor',ready:false},
   {position:2,accountId:'',ready:false},{position:3,accountId:'',ready:false}]})
 let aReads=0,releaseOld
 const urls=[]
 f.platform.http.request=async path=>{
  urls.push(path)
  const id=path.endsWith('/B')?'B':'A'
  if(id==='A' && ++aReads===2)
   return new Promise(resolve=>{releaseOld=()=>resolve({code:'OK',data:room('A')})})
  return {code:'OK',data:room(id)}
 }
 const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
 await online.loginAndConnect()
 const old=online.reconnectAndResync('A')
 await new Promise(resolve=>setImmediate(resolve))
 assert.equal(aReads,2)
 assert.equal(typeof releaseOld,'function')
 await online.joinRoom('B')
 assert.equal(online.rooms.state.roomId,'B')
 const newer=online.reconnectAndResync('A')
 assert.notStrictEqual(newer,old,'different membership epoch must create a new recovery')
 const fresh=await newer
 assert.equal(fresh.roomId,'A')
 assert.equal(aReads,4,'new A must perform two new authority reads without old Promise')
 assert.equal(online.rooms.state.roomId,'A')
 releaseOld()
 assert.equal(await old,null,'old A must remain invalid after B join')
 assert.equal(online.rooms.state.roomId,'A')
 assert.equal(f.commands.length,0,'member rejoin and recovery are read-only')
 online.dispose()
})

test('audit: a room command epoch revokes an in-flight same-key recovery while retaining exact room identity',async()=>{
 const f=fixture()
 let version=1,reads=0,releaseOld
 const view=v=>({roomId:'A',version:v,status:'WAITING_PLAYERS',member:true,occupied:1,
  lastClientSequence:v,seats:[{position:1,accountId:'actor',ready:v>1},
   {position:2,accountId:'',ready:false},{position:3,accountId:'',ready:false}]})
 f.platform.http.request=async()=>{
  if(++reads===2)return new Promise(resolve=>{releaseOld=()=>resolve({code:'OK',data:view(1)})})
  return {code:'OK',data:view(version)}
 }
 const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
 await online.loginAndConnect()
 online.rooms.restoreTrustedRoom('A',1,1)
 const old=online.reconnectAndResync('A')
 await new Promise(resolve=>setImmediate(resolve))
 assert.equal(reads,2)
 const ready=online.rooms.ready(true)
 await new Promise(resolve=>setImmediate(resolve))
 const cmd=f.commands.at(-1).payload
 version=2
 for(const cb of f.messages)cb(JSON.stringify({type:'doudizhu.command.result',payload:{
  v:'doudizhu-api-command-result-v1',requestId:cmd.requestId,type:'room.ready',
  durable:{v:'fair-doudizhu-command-result-v1',commandId:cmd.requestId,
   aggregateId:'A',aggregateVersion:2,accepted:true}}}))
 await ready
 const newer=online.reconnectAndResync('A')
 assert.notStrictEqual(newer,old,'command start and settlement invalidate stale same-key reads')
 assert.equal((await newer).version,2)
 assert.equal(reads,4)
 releaseOld()
 assert.equal(await old,null)
 assert.equal(online.rooms.state.version,2)
 online.dispose()
})

test('audit: delayed create CSPRNG cannot steal a newer member room selection',async()=>{
 const f=fixture()
 let resolveRandom
 f.platform.random.bytes=async n=>new Promise(resolve=>{resolveRandom=()=>resolve(new Uint8Array(n).fill(3))})
 const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
 await online.loginAndConnect()
 const older=online.createRoom()
 await new Promise(resolve=>setImmediate(resolve))
 assert.equal(typeof resolveRandom,'function')
 await online.joinRoom('room-rejoin')
 assert.equal(online.rooms.state.roomId,'room-rejoin')
 resolveRandom()
 await assert.rejects(older,/selection superseded/)
 assert.equal(online.rooms.state.roomId,'room-rejoin')
 assert.equal(f.commands.length,0,'no room.create is sent after newer member choice')
 online.dispose()
})

test('audit: current create still commits after CSPRNG and accepted server receipt',async()=>{
 const f=fixture()
 const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
 await online.loginAndConnect()
 const pending=online.createRoom()
 await new Promise(resolve=>setImmediate(resolve))
 const cmd=f.commands.at(-1).payload
 assert.equal(cmd.type,'room.create')
 assert.equal(cmd.aggregateId,'room-cmd-000102030405060708090a0b0c0d0e0f')
 for(const cb of f.messages)cb(JSON.stringify({type:'doudizhu.command.result',payload:{
  v:'doudizhu-api-command-result-v1',requestId:cmd.requestId,type:cmd.type,
  durable:{v:'fair-doudizhu-command-result-v1',commandId:cmd.requestId,
   aggregateId:cmd.aggregateId,aggregateVersion:1,accepted:true}}}))
 const room=await pending
 assert.equal(room.roomId,cmd.aggregateId)
 assert.equal(online.rooms.state.roomId,cmd.aggregateId)
 online.dispose()
})

test('audit: superseded startHand after waiting for old fairness reconciliation does not send hand start to B',async()=>{
 const f=fixture()
 const view=id=>({roomId:id,version:1,status:'WAITING_PLAYERS',member:true,occupied:1,
  lastClientSequence:1,seats:[{position:1,accountId:'actor',ready:false},
   {position:2,accountId:'',ready:false},{position:3,accountId:'',ready:false}]})
 let releaseB
 f.platform.http.request=async path=>path.endsWith('/B')
  ? new Promise(resolve=>{releaseB=()=>resolve({code:'OK',data:view('B')})})
  : {code:'OK',data:view('A')}
 const online=new GameOnlineController(f.platform,f.identity,'wss://example.org/ws',()=>{})
 await online.loginAndConnect()
 online.rooms.restoreTrustedRoom('A',1,1)
 const oldFairness=online.fairness
 let releaseFairness,unknown=true
 Object.defineProperty(oldFairness,'needsReconciliation',{get:()=>unknown})
 oldFairness.retryUnknown=()=>new Promise(resolve=>{releaseFairness=()=>{unknown=false;resolve({})}})
 const start=online.startHand('hand-H2')
 await new Promise(resolve=>setImmediate(resolve))
 assert.equal(typeof releaseFairness,'function')
 const join=online.joinRoom('B')
 await new Promise(resolve=>setImmediate(resolve))
 assert.equal(typeof releaseB,'function')
 releaseFairness()
 await assert.rejects(start,/selection superseded/)
 // Both B reads must be authorized before it can be adopted.
 releaseB()
 await new Promise(resolve=>setImmediate(resolve))
 releaseB()
 await join
 assert.equal(online.rooms.state.roomId,'B')
 assert.equal(f.commands.length,0,'no room.hand.start can be sent to the newer room')
 online.dispose()
})

test('web login exchanges identifier/password at existing Go route and connects only after verified access token',async()=>{
 const f=fixture()
 f.platform.target='web'
 const requests=[]
 f.platform.http.request=async(path,options)=>{
  requests.push({path,options})
  return {code:'OK',data:{accessToken:'verified-web-token',tokenType:'Bearer'}}
 }
 const online=new GameOnlineController(f.platform,{
  exchange:async()=>{throw Error('wechat exchange must not run for web')},
  revoke:async()=>{}
 },'wss://example.org/ws',()=>{})
 await online.loginWithPassword('web-user','confidential-pass')
 assert.deepEqual(requests.map(x=>x.path),['/auth/login'])
 assert.deepEqual(requests[0].options.body,{identifier:'web-user',password:'confidential-pass'})
 assert.deepEqual(f.auth,[{url:'wss://example.org/ws',token:'verified-web-token'}])
 assert.equal(online.session.token,'verified-web-token')
 online.dispose()
})
test('invalid or rejected web login cannot connect any WSS transport',async()=>{
 const f=fixture()
 f.platform.target='web'
 let called=0
 f.platform.http.request=async()=>{called++;return {code:'OK',data:{tokenType:'Bearer',accessToken:''}}}
 const online=new GameOnlineController(f.platform,{
  exchange:async()=>{throw Error('not called')}
 },'wss://example.org/ws',()=>{})
 await assert.rejects(online.loginWithPassword('alice','test'),/invalid web authentication token/)
 await assert.rejects(online.loginWithPassword('','test'),/valid web credentials required/)
 assert.equal(called,1)
 assert.equal(f.auth.length,0)
 assert.equal(online.rooms.state,null)
 online.dispose()
})
