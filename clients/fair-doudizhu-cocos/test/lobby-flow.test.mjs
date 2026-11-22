import {test} from 'node:test'
import assert from 'node:assert/strict'
import {planLobby} from '../dist/bootstrap/LobbyFlow.js'

const seats=[
 {position:1,accountId:'account-A',ready:false},
 {position:2,accountId:'account-B',ready:true},
 {position:3,accountId:'',ready:false}
]
const context=(changes={})=>({
 authenticated:true,roomId:'',room:null,roomUnknown:false,fairnessUnknown:false,
 fairnessMode:null,fairness:null,handPhase:null,recoverableRooms:0,...changes
})
const room=(changes={})=>({
 roomId:'room-1',status:'WAITING_PLAYERS',occupied:2,member:true,seats,...changes
})
test('guest, entry and unverified room expose no unauthorized room mutations',()=>{
 assert.deepEqual(planLobby(context({authenticated:false})).primary,[])
 const entry=planLobby(context())
 assert.deepEqual(entry.primary,['create','join'])
 assert.equal(entry.input,'join')
 for(const unverified of [null,room({member:false}),room({roomId:'room-2'}),room({status:'CLOSED'})]){
  const result=planLobby(context({roomId:'room-1',room:unverified}))
  assert.deepEqual(result.primary,['sync-room'])
  assert.equal(result.input,'none')
 }
})
test('room waiting, ready and terminal states show next action, not flat test menu',()=>{
 const waiting=planLobby(context({roomId:'room-1',room:room()}))
 assert.deepEqual(waiting.primary,['ready'])
 assert.equal(waiting.seats.length,3)
 assert.equal(waiting.seats[2].occupied,false)
 const ready=planLobby(context({roomId:'room-1',room:room({
  status:'READY',occupied:3,seats:seats.map(s=>({...s,ready:true,accountId:s.accountId||'account-C'}))
 })}))
 assert.deepEqual(ready.primary,['start'])
 const past=planLobby(context({roomId:'room-1',room:room({lastHandId:'hand-1'})}))
 assert.ok(past.secondary.includes('evidence'))
})
test('new fairness contribution and terminal own phrase verification are distinct inputs',()=>{
 const active=room({status:'HAND_ACTIVE',activeHandId:'hand-1'})
 const ctx=context({roomId:'room-1',room:active,fairnessMode:'client-contributed-server-custodied-v1'})
 const unverified=planLobby(ctx)
 assert.deepEqual(unverified.primary,['sync-fairness'])
 assert.equal(unverified.input,'none')
 const needs=planLobby({...ctx,fairness:{handId:'hand-1',phase:'REVEALING',revealed:false}})
 assert.deepEqual(needs.primary,['contribute'])
 assert.equal(needs.input,'contribute')
 const done=planLobby({...ctx,fairness:{handId:'hand-1',phase:'REVEALING',revealed:true}})
 assert.deepEqual(done.primary,['sync-fairness'])
 assert.equal(done.input,'none')
 const playing=planLobby({...ctx,handPhase:'PLAYING'})
 assert.deepEqual(playing.primary,['table'])
 const final=planLobby({...ctx,handPhase:'COMPLETED'})
 assert.deepEqual(final.primary,['evidence'])
 assert.equal(final.input,'verify')
 assert.ok(final.secondary.includes('verify-own'))
 const otherHand=planLobby({...ctx,fairness:{handId:'old-hand',phase:'REVEALING',revealed:false}})
 assert.equal(otherHand.input,'none','old hand evidence must not show contribution')
})
test('UNKNOWN preserves exact original request path and hides all fresh mutations',()=>{
 for(const flag of ['roomUnknown','fairnessUnknown']){
  const p=planLobby(context({roomId:'room-1',room:room(),[flag]:true}))
  assert.equal(p.stage,'reconcile')
  assert.deepEqual(p.primary,[flag==='roomUnknown'?'retry-room':'retry-fairness'])
  assert.equal(p.input,'none')
  assert.equal(p.secondary.length,0)
 }
})
test('state projection never exposes full AccountID in seat labels',()=>{
 const p=planLobby(context({roomId:'room-1',room:room()}))
 assert.deepEqual(p.seats[0],{position:1,occupied:true,ready:false})
 assert.doesNotMatch(JSON.stringify(p.seats),/account-/)
})

test('terminal last hand retains its own-phrase fairness verification',()=>{
 const p=planLobby(context({roomId:'room-1',room:room({lastHandId:'hand-final'}),
   fairnessMode:'client-contributed-server-custodied-v1'}))
 assert.equal(p.input,'verify')
 assert.ok(p.secondary.includes('evidence'))
 assert.ok(p.secondary.includes('verify-own'))
})

test('only owner at authoritative seat 1 is offered start, ready players are not prompted again',()=>{
 const all=seats.map(s=>({...s,ready:true,accountId:s.accountId||'account-C'}))
 const base=context({roomId:'room-1',room:room({status:'READY',occupied:3,seats:all})})
 assert.deepEqual(planLobby({...base,viewerAccountID:'account-A'}).primary,['start'])
 assert.deepEqual(planLobby({...base,viewerAccountID:'account-B'}).primary,['sync-room'])
 assert.match(planLobby({...base,viewerAccountID:'account-B'}).hint,/等待房主/)
 const waiting=context({roomId:'room-1',room:room()})
 assert.deepEqual(planLobby({...waiting,viewerAccountID:'account-A'}).primary,['ready'])
 assert.deepEqual(planLobby({...waiting,viewerAccountID:'account-B'}).primary,['sync-room'])
 assert.equal(planLobby({...waiting,viewerAccountID:'not-a-member'}).stage,'syncing')
})
