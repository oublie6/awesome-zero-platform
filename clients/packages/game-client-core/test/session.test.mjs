import { test } from 'node:test'
import assert from 'node:assert/strict'
import { GameSession } from '../dist/session.js'
function platform() {
  let saved = null
  return {
    target:'wechat',
    auth:{acquireCode:async()=> 'temporary-wechat-code'},
    vault:{ load:async()=>saved, save:async value=>{saved=value}, clear:async()=>{saved=null} }
  }
}
test('login uses server exchange and never treats temporary code as an access token', async()=>{
  const p=platform()
  let observed
  const session=new GameSession(p,{exchange:async(t,code)=>{observed={t,code};return {accessToken:'server-issued-jwt'}}})
  await session.login()
  assert.deepEqual(observed,{t:'wechat',code:'temporary-wechat-code'})
  assert.equal(session.token,'server-issued-jwt')
  assert.equal(await p.vault.load(),'server-issued-jwt')
  await session.logout()
  assert.equal(session.authenticated,false)
  assert.equal(await p.vault.load(),null)
})
test('restored sessions require server validation and invalid tokens are cleared',async()=>{
  const p=platform();await p.vault.save('expired')
  const session=new GameSession(p,{exchange:async()=>({accessToken:'valid'})})
  assert.equal(await session.restore(async()=>false),false)
  assert.equal(await p.vault.load(),null)
})
test('disposed session refuses new logins',async()=>{
  const session=new GameSession(platform(),{exchange:async()=>({accessToken:'valid'})})
  session.dispose()
  await assert.rejects(session.login(),/disposed/)
})
test('logout during pending exchange cannot resurrect an authenticated session',async()=>{
  const p=platform()
  let release
  const gate=new Promise(resolve=>{release=resolve})
  const session=new GameSession(p,{exchange:async()=>{await gate;return {accessToken:'late-token'}}})
  const attempt=session.login()
  await Promise.resolve()
  await session.logout()
  release()
  await attempt
  assert.equal(session.authenticated,false)
  assert.equal(await p.vault.load(),null)
})

test('web credential login only persists server access token, not user/password',async()=>{
 const p=platform()
 p.target='web'
 const calls=[]
 const s=new GameSession(p,{exchange:async()=>{throw Error('platform login should not be called')}})
 await s.loginWith(async()=>{
  calls.push('server credential exchange')
  return {accessToken:'verified-server-access-token'}
 })
 assert.deepEqual(calls,['server credential exchange'])
 assert.equal(await p.vault.load(),'verified-server-access-token')
 assert.equal(s.token,'verified-server-access-token')
 await s.logout()
 assert.equal(await p.vault.load(),null)
})
test('web login response arriving after logout cannot revive session',async()=>{
 const p=platform()
 p.target='web'
 const s=new GameSession(p,{exchange:async()=>{throw Error('not used')}})
 let release
 const pending=new Promise(resolve=>{release=resolve})
 const attempt=s.loginWith(()=>pending)
 await s.logout()
 release({accessToken:'old-web-token'})
 await attempt
 assert.equal(s.authenticated,false)
 assert.equal(await p.vault.load(),null)
})

test('slow remote logout cannot clear a newer confirmed token',async()=>{
 const p=platform()
 let release
 const s=new GameSession(p,{exchange:async()=>({accessToken:'unused'}),revoke:()=>new Promise(ok=>{release=ok})})
 await s.loginWith(async()=>({accessToken:'old'}))
 const logout=s.logout()
 for(let i=0;i<20&&!release;i++)await new Promise(ok=>setImmediate(ok))
 assert.ok(release)
 assert.equal(await p.vault.load(),null)
 await s.loginWith(async()=>({accessToken:'new'}))
 release();await logout
 assert.equal(s.token,'new');assert.equal(await p.vault.load(),'new')
})
test('late save cleanup is ordered before a newer login persists its token',async()=>{
 const p=platform()
 const normalSave=p.vault.save
 let release,started=false
 p.vault.save=async value=>{
  if(value==='old'){started=true;await new Promise(ok=>{release=ok})}
  await normalSave(value)
 }
 const s=new GameSession(p,{exchange:async()=>({accessToken:'unused'})})
 const old=s.loginWith(async()=>({accessToken:'old'}))
 for(let i=0;i<20&&!started;i++)await new Promise(ok=>setImmediate(ok))
 assert.equal(started,true)
 const fresh=s.loginWith(async()=>({accessToken:'new'}))
 release();await Promise.all([old,fresh])
 assert.equal(s.token,'new');assert.equal(await p.vault.load(),'new')
})
test('a stale validation cannot clear or restore a token replaced by another controller',async()=>{
 for(const valid of [false,true]){
  const p=platform()
  await p.vault.save('old')
  let release
  const s=new GameSession(p,{exchange:async()=>({accessToken:'unused'})})
  const pending=s.restore(()=>new Promise(ok=>{release=ok}))
  for(let i=0;i<20&&!release;i++)await new Promise(ok=>setImmediate(ok))
  assert.ok(release)
  await p.vault.save('new')
  release(valid)
  assert.equal(await pending,false)
  assert.equal(s.authenticated,false)
  assert.equal(await p.vault.load(),'new')
 }
})

test('G65 R7: logout owns the token observed by restore, not a newer shared token',async()=>{
 const p=platform()
 await p.vault.save('old')
 const revoked=[]
 const identity={exchange:async()=>({accessToken:'unused'}),revoke:async t=>{revoked.push(t)}}
 const old=new GameSession(p,identity), fresh=new GameSession(p,identity)
 let release
 const pending=old.restore(()=>new Promise(ok=>{release=ok}))
 for(let i=0;i<20&&!release;i++)await new Promise(ok=>setImmediate(ok))
 assert.ok(release)
 await fresh.loginWith(async()=>({accessToken:'new'}))
 await old.logout()
 release(true)
 assert.equal(await pending,false)
 assert.equal(await p.vault.load(),'new')
 assert.deepEqual(revoked,['old'])
 assert.equal(fresh.token,'new')
 await new GameSession(p,identity).logout()
 assert.equal(await p.vault.load(),'new')
})
test('G65 R7: shared async vault serializes delayed save and newer login',async()=>{
 const p=platform(),save=p.vault.save
 let release
 p.vault.save=async value=>{if(value==='old')await new Promise(ok=>{release=ok});await save(value)}
 const identity={exchange:async()=>({accessToken:'unused'})}
 const old=new GameSession(p,identity),fresh=new GameSession(p,identity)
 const pending=old.loginWith(async()=>({accessToken:'old'}))
 for(let i=0;i<20&&!release;i++)await new Promise(ok=>setImmediate(ok))
 assert.ok(release)
 const next=fresh.loginWith(async()=>({accessToken:'new'}))
 old.dispose()
 release()
 await Promise.all([pending,next])
 assert.equal(await p.vault.load(),'new')
 assert.equal(fresh.token,'new')
 assert.equal(old.authenticated,false)
})
test('G65 R7: late exchange cannot replace a newer login across controllers',async()=>{
 const p=platform(),identity={exchange:async()=>({accessToken:'unused'})}
 const a=new GameSession(p,identity),b=new GameSession(p,identity)
 let release
 const pending=a.loginWith(()=>new Promise(ok=>{release=ok}))
 await b.loginWith(async()=>({accessToken:'new'}))
 release({accessToken:'old'});await pending
 assert.equal(await p.vault.load(),'new')
 assert.equal(a.authenticated,false)
})

test('G65 R7: provider detach clears only owned local token and never revokes again',async()=>{
 const p=platform()
 let revokes=0
 const identity={exchange:async()=>({accessToken:'unused'}),revoke:async()=>{revokes++}}
 const old=new GameSession(p,identity),fresh=new GameSession(p,identity)
 await old.loginWith(async()=>({accessToken:'old'}))
 await fresh.loginWith(async()=>({accessToken:'new'}))
 await old.discardLocal()
 old.dispose()
 assert.equal(await p.vault.load(),'new')
 assert.equal(revokes,0)
 await fresh.discardLocal()
 assert.equal(await p.vault.load(),null)
 assert.equal(revokes,0)
})

test('G65 R8: explicit session completion is null when superseded and grants expire with ownership',async()=>{
 const p=platform(),identity={exchange:async()=>({accessToken:'unused'})}
 const s=new GameSession(p,identity)
 let release
 const old=s.loginWith(()=>new Promise(ok=>{release=ok}))
 const fresh=await s.loginWith(async()=>({accessToken:'new'}))
 assert.ok(fresh?.isCurrent())
 release({accessToken:'old'})
 assert.equal(await old,null)
 assert.equal(fresh.token,'new')
 await s.logout()
 assert.equal(fresh.isCurrent(),false)
})

test('G65 R9: shared grant is the authenticated state and stale SDK completion stops before exchange',async()=>{
 const p=platform(),a=new GameSession(p,{exchange:async()=>({accessToken:'old'})})
 const b=new GameSession(p,{exchange:async()=>({accessToken:'new'})})
 await a.login()
 const old=a.captureAttempt()
 assert.equal(a.authenticated,true)
 let release,calls=0
 const slow=new GameSession({...p,auth:{acquireCode:()=>new Promise(ok=>{release=ok})}},
  {exchange:async()=>{calls++;return {accessToken:'late'}}})
 const pending=slow.login()
 assert.equal(a.authenticated,false)
 assert.equal(old(),false)
 assert.throws(()=>a.token,/not authenticated/)
 const restore=new GameSession(p,{exchange:async()=>({accessToken:'unused'})})
 assert.equal(await restore.restore(async()=>true),false)
 const skippedRestore=restore.captureAttempt()
 await b.login()
 assert.equal(skippedRestore(),false)
 release('late-code')
 assert.equal(await pending,null)
 assert.equal(calls,0)
 assert.equal(b.authenticated,true)
 await a.discardLocal()
 assert.equal(await p.vault.load(),'new')
})
test('G65 R9: unbound attempt owner expires on shared replacement even without an authenticated grant',async()=>{
 const p=platform()
 const a=new GameSession(p,{exchange:async()=>{throw Error('binding required')}})
 const b=new GameSession(p,{exchange:async()=>({accessToken:'new'})})
 await assert.rejects(a.login(),/binding required/)
 const owner=a.captureAttempt()
 assert.equal(owner(),true)
 assert.equal(a.authenticated,false)
 await b.login()
 assert.equal(owner(),false)
})

test('G65 R10: canceled login releases its restore barrier before old SDK returns',async()=>{
 for(const method of ['dispose','logout','discardLocal','cancelAttempt']){
  const p=platform()
  await p.vault.save('stored')
  let release,exchanges=0
  p.auth.acquireCode=()=>new Promise(ok=>{release=ok})
  const identity={exchange:async()=>{exchanges++;return {accessToken:'late'}}}
  const a=new GameSession(p,identity),b=new GameSession(p,identity)
  const pending=a.login()
  await a[method]()
  assert.equal(await b.restore(async t=>t==='stored'),true)
  assert.equal(await pending,null)
  release('late')
  await Promise.resolve()
  assert.equal(exchanges,0)
  assert.equal(b.token,'stored')
 }
})
test('G65 R10: old cancellation cannot release a newer active login barrier',async()=>{
 const p=platform();await p.vault.save('stored')
 const identity={exchange:async()=>({accessToken:'unused'})}
 const a=new GameSession(p,identity),b=new GameSession(p,identity),c=new GameSession(p,identity)
 let old,newer
 const first=a.loginWith(()=>new Promise(ok=>{old=ok}))
 const second=b.loginWith(()=>new Promise(ok=>{newer=ok}))
 a.dispose()
 assert.equal(await c.restore(async()=>true),false)
 old({accessToken:'old'});assert.equal(await first,null)
 assert.equal(await c.restore(async()=>true),false)
 newer({accessToken:'new'});assert.ok(await second)
 assert.equal(await c.restore(async()=>true),true)
 assert.equal(c.token,'new')
})
test('G65 R10: same session restore cancels its pending login before reading saved identity',async()=>{
 const p=platform();await p.vault.save('stored')
 const a=new GameSession(p,{exchange:async()=>({accessToken:'unused'})})
 let release
 const pending=a.loginWith(()=>new Promise(ok=>{release=ok}))
 assert.equal(await a.restore(async()=>true),true)
 assert.equal(await pending,null)
 release({accessToken:'late'})
 assert.equal(a.token,'stored')
})
