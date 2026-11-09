import {test} from 'node:test'
import assert from 'node:assert/strict'
import {createWebAuth,runtimeConfig,SESSION_KEY} from '../src/auth.ts'

const config={apiBase:'https://play.example.com',socketEndpoint:'wss://play.example.com/ws',revealManifestRoots:{}}
function storage(){
 const data=new Map()
 return {
  getItem:key=>data.get(key)??null,
  setItem:(key,value)=>data.set(key,value),
  removeItem:key=>data.delete(key),
  contents:()=>[...data.entries()]
 }
}
const okay=data=>new Response(JSON.stringify({code:'OK',data}),{
 status:200,headers:{'content-type':'application/json'}
})
const invalid=()=>new Response(JSON.stringify({code:'UNAUTHORIZED'}),{status:401})

test('Vue runtime requires exact same-origin HTTPS/WSS; cannot use cross-origin iframe token bridge',()=>{
 assert.ok(runtimeConfig('https://play.example.com',config))
 assert.equal(runtimeConfig('http://play.example.com',config),null)
 assert.equal(runtimeConfig('https://another.example.com',config),null)
 assert.equal(runtimeConfig('https://play.example.com',{...config,socketEndpoint:'wss://foreign.example.com/ws'}),null)
 assert.equal(runtimeConfig('https://play.example.com',{...config,apiBase:'http://play.example.com'}),null)
 assert.equal(runtimeConfig('https://play.example.com',{...config,apiBase:'https://user:pass@play.example.com'}),null)
})
test('real Go credential exchange and server validation store only server token, never password',async()=>{
 const s=storage(),calls=[]
 const fetcher=async(url,options)=>{
  const path=new URL(url).pathname
  calls.push({path,method:options.method,headers:options.headers,body:options.body})
  if(path==='/auth/login')return okay({tokenType:'Bearer',accessToken:'real-server-access-token',refreshToken:'never-persist-me'})
  if(path==='/auth/session')return okay({account:{id:'server-account-uuid',displayName:'测试玩家'}})
  if(path==='/auth/logout')return okay({loggedOut:true})
  throw Error('unexpected endpoint '+path)
 }
 const web=createWebAuth(config,fetcher,s)
 const actor=await web.login(' alice ','test-password')
 assert.deepEqual(actor,{id:'server-account-uuid',displayName:'测试玩家'})
 assert.deepEqual(calls.map(c=>c.path),['/auth/login','/auth/session'])
 assert.deepEqual(JSON.parse(calls[0].body),{identifier:'alice',password:'test-password'})
 assert.equal(calls[1].headers.Authorization,'Bearer real-server-access-token')
 assert.deepEqual(s.contents(),[[SESSION_KEY,'real-server-access-token']])
 assert.equal(await web.restore(),actor)
 await web.logout()
 assert.equal(s.getItem(SESSION_KEY),null)
 assert.equal(calls.at(-1).headers.Authorization,'Bearer real-server-access-token')
})
test('invalid login does not create local web session or socket credentials',async()=>{
 const s=storage()
 const web=createWebAuth(config,async()=>invalid(),s)
 await assert.rejects(web.login('alice','bad-pass'),/账号或密码错误/)
 assert.deepEqual(s.contents(),[])
 assert.equal(web.authenticated,false)
})
test('server rejection of saved access token clears only old token',async()=>{
 const s=storage();s.setItem(SESSION_KEY,'expired-token')
 const web=createWebAuth(config,async()=>invalid(),s)
 assert.equal(await web.restore(),null)
 assert.equal(s.getItem(SESSION_KEY),null)
 assert.equal(web.authenticated,false)
})
test('late login completion after logout never revives earlier identity',async()=>{
 const s=storage()
 let resolveLogin
 const wait=new Promise(resolve=>{resolveLogin=resolve})
 let calls=0
 const fetcher=async(url)=>{
  calls++
  if(new URL(url).pathname==='/auth/login')return wait
  return okay({account:{id:'A',displayName:'A'}})
 }
 const web=createWebAuth(config,fetcher,s)
 const pending=web.login('alice','secret')
 await web.logout()
 resolveLogin(okay({accessToken:'late-token',tokenType:'Bearer'}))
 await pending
 assert.equal(web.authenticated,false)
 assert.deepEqual(s.contents(),[])
})
test('malformed server token or missing principal fails closed',async()=>{
 const s=storage()
 const web=createWebAuth(config,async(url)=>{
  if(new URL(url).pathname==='/auth/login')return okay({accessToken:'header\r\ninjection',tokenType:'Bearer'})
  return okay({account:{id:'A',displayName:'Alice'}})
 },s)
 await assert.rejects(web.login('alice','p'),/有效访问令牌/)
 assert.deepEqual(s.contents(),[])
})
