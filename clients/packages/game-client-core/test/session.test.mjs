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
