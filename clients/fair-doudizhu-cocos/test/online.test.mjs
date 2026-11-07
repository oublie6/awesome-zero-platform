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
  assert.equal(paths.length,2)
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
