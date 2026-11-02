#!/usr/bin/env node
// Real endpoint qualification only: three distinct platform-authenticated
// bearer tokens; no faked accounts, hands, cards, HTTP server or websocket.
// Run against a pre-dealt BIDDING hand with three real, seated accounts.
// Secrets and private hands are deliberately never logged.
import assert from 'node:assert/strict'
import {randomBytes} from 'node:crypto'

const api=process.env.GOAL0046_E2E_API
const socket=process.env.GOAL0046_E2E_WSS
const hand=process.env.GOAL0046_E2E_HAND
const tokens=[1,2,3].map(i=>process.env['GOAL0046_E2E_TOKEN_'+i])
if(!api?.startsWith('https://')||!socket?.startsWith('wss://')||
  !/^[A-Za-z0-9_-]{1,128}$/.test(hand??'')||
  tokens.some(t=>!t||/\s/.test(t))||new Set(tokens).size!==3) {
  throw new Error('requires GOAL0046_E2E_API=https://, WSS=wss://, HAND and 3 distinct TOKEN_1..3')
}
const deadline=Date.now()+120000
async function get(token,path) {
  const response=await fetch(api+path,{headers:{Authorization:'Bearer '+token},
    signal:AbortSignal.timeout(12000)})
  if(!response.ok)throw new Error('authenticated HTTP '+response.status+' on '+path)
  const body=await response.json()
  if(body.code!=='OK')throw new Error('server rejected authenticated read: '+String(body.code))
  return body.data
}
class Player {
  constructor(token){this.token=token;this.ws=null;this.inflight=new Map();this.sequence=1}
  async connect(){
    const ws=new WebSocket(socket,['bearer',this.token])
    this.ws=ws
    ws.addEventListener('message',event=>{
      let message
      try{message=JSON.parse(event.data)}catch{return}
      const waiting=this.inflight.get(message.id)
      if(!waiting)return
      if(message.type==='doudizhu.error'){
        waiting.reject(new Error('server rejected live command: '+String(message.payload?.code)))
      }else if(message.type==='doudizhu.command.result'){
        waiting.resolve(message.payload)
      }else return
      this.inflight.delete(message.id)
      clearTimeout(waiting.timer)
    })
    ws.addEventListener('close',()=>{
      for(const [id,entry] of this.inflight) {
        entry.reject(new Error('WSS closed; outcome unknown: '+id))
        clearTimeout(entry.timer)
      }
      this.inflight.clear()
    })
    await new Promise((resolve,reject)=>{
      const timer=setTimeout(()=>reject(new Error('WSS open deadline exceeded')),12000)
      ws.addEventListener('open',()=>{clearTimeout(timer);resolve()},{once:true})
      ws.addEventListener('error',()=>{clearTimeout(timer);reject(new Error('WSS failed'))},{once:true})
    })
  }
  async action(type,version,payload,requestId='cmd-'+randomBytes(16).toString('hex')){
    assert.ok(this.ws?.readyState===WebSocket.OPEN,'player socket not open')
    const seq=this.sequence++
    const command={v:'doudizhu-api-command-v1',requestId,type,
      aggregateId:hand,clientSeq:seq,expectedVersion:version,payload}
    const ack=await new Promise((resolve,reject)=>{
      const timer=setTimeout(()=>{
        this.inflight.delete(requestId)
        reject(new Error('WSS command result timeout; outcome unknown'))
      },12000)
      this.inflight.set(requestId,{resolve,reject,timer})
      this.ws.send(JSON.stringify({id:requestId,type:'doudizhu.command',payload:command}))
    })
    if(ack.requestId!==requestId||ack.type!==type||!ack.live||
       ack.live.version!==version+1)throw new Error('invalid correlated durable live outcome')
    return ack
  }
  close(){this.ws?.close()}
}
const players=[]
try {
  for(const token of tokens) {
    const projection=await get(token,'/v1/doudizhu/hands/'+hand+'/private')
    assert.ok(projection&&projection.version>=1&&
      projection.payload?.public?.handId===hand,'private view bound to hand')
    const seat=projection.payload.position
    assert.ok(Number.isInteger(seat)&&seat>=1&&seat<=3,'authenticated seat')
    assert.ok(!players.some(p=>p.seat===seat),'all three bearer identities must have distinct seats')
    const player=new Player(token)
    player.seat=seat
    await player.connect()
    players.push(player)
  }
  assert.equal(players.length,3)
  let final=null
  let count=0
  while(Date.now()<deadline&&count++<80){
    // Server is the only authority for cards, bids, turns and settlement.
    let view=await get(players[0].token,'/v1/doudizhu/hands/'+hand+'/private')
    const publicView=view.payload.public
    assert.ok(['BIDDING','PLAYING'].includes(publicView.phase),'hand must be active')
    const turn=publicView.phase==='BIDDING'?publicView.bidding.currentBidder:
      publicView.playing?.currentSeat
    const current=players.find(p=>p.seat===turn)
    assert.ok(current,'turn must belong to one of three authenticated clients')
    view=await get(current.token,'/v1/doudizhu/hands/'+hand+'/private')
    assert.equal(view.version,publicView.stateVersion,'all clients use same authority version')
    let type,payload
    if(publicView.phase==='BIDDING') {
      type='hand.bid';payload={score:3}
    }else if(turn===publicView.landlordSeat){
      assert.ok(view.payload.cards.length>0,'landlord must have a real private card')
      type='hand.play';payload={cards:[view.payload.cards[0]]}
    }else{type='hand.pass';payload={}}
    const result=await current.action(type,view.version,payload)
    if(result.live.payload?.phase==='COMPLETED'){
      final=result
      break
    }
  }
  assert.ok(final,'game did not reach authenticated settlement in 80 turns')
  const evidence=await get(players[0].token,'/v1/doudizhu/hands/'+hand+'/evidence')
  assert.equal(evidence.handId,hand)
  assert.equal(evidence.finalVersion,final.live.version)
  assert.equal(evidence.verification?.seatPoints?.length,3)
  assert.match(evidence.verification?.recordDigest??'',/^[0-9a-f]{64}$/)
  process.stdout.write(JSON.stringify({pass:true,handId:hand,threeDistinctSeats:true,
    actions:count,finalVersion:evidence.finalVersion,evidenceDigest:evidence.verification.recordDigest})+'\n')
}finally{for(const p of players)p.close()}
