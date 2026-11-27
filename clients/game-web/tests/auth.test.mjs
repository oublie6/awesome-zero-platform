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
 assert.deepEqual(await web.restore(),actor)
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


test('registration reuses server identity, requires email and never creates session',async()=>{
 const s=storage(), calls=[]
 const web=createWebAuth(config,async(url,options)=>{
  calls.push({path:new URL(url).pathname,body:JSON.parse(options.body)})
  return new Response(JSON.stringify({code:'OK',data:{account:{id:'uuid-1',displayName:'玩家'},emailVerified:false}}),{status:201})
 },s)
 await assert.rejects(web.register('alice','','玩家','password123'),/有效用户名/)
 assert.deepEqual(await web.register('alice','alice@example.com','玩家','password123'),{id:'uuid-1',displayName:'玩家'})
 assert.equal(calls.length,1)
 assert.equal(calls[0].path,'/auth/register')
 assert.deepEqual(s.contents(),[])
 assert.equal(web.authenticated,false)
})

test('identity timeout includes reading the response body after headers arrived',async()=>{
 const s=storage()
 let aborted=false
 const web=createWebAuth(config,async(_url,{signal})=>({
  ok:true,json:()=>new Promise((_resolve,reject)=>signal.addEventListener('abort',()=>{
    aborted=true;reject(Error('body aborted'))
  },{once:true}))
 }),s,10)
 await assert.rejects(web.login('alice','secret'),/body aborted/)
 assert.equal(aborted,true)
 assert.equal(web.authenticated,false)
 assert.deepEqual(s.contents(),[])
})
test('late validation cannot adopt a token replaced outside the Web auth controller',async()=>{
 const s=storage()
 s.setItem(SESSION_KEY,'old')
 let release
 const web=createWebAuth(config,()=>new Promise(ok=>{release=ok}),s)
 const pending=web.restore()
 s.setItem(SESSION_KEY,'new')
 release(okay({account:{id:'old-id',displayName:'Old'}}))
 assert.equal(await pending,null)
 assert.equal(web.authenticated,false)
 assert.equal(s.getItem(SESSION_KEY),'new')
})

test('G65 R7: old restore logout cannot remove or revoke replacement token',async()=>{
 const s=storage(),revoked=[]
 s.setItem(SESSION_KEY,'old')
 let release
 const old=createWebAuth(config,async(url,options)=>{
  if(new URL(url).pathname==='/auth/logout'){
   revoked.push(options.headers.Authorization);return okay({loggedOut:true})
  }
  return new Promise(ok=>{release=ok})
 },s)
 const pending=old.restore()
 const fresh=createWebAuth(config,async url=>new URL(url).pathname==='/auth/login'
  ?okay({tokenType:'Bearer',accessToken:'new'}):okay({account:{id:'new-id',displayName:'New'}}),s)
 await fresh.login('new','password')
 await old.logout()
 release(okay({account:{id:'old-id',displayName:'Old'}}))
 assert.equal(await pending,null)
 assert.equal(s.getItem(SESSION_KEY),'new')
 assert.deepEqual(revoked,['Bearer old'])
 assert.equal(fresh.authenticated,true)
})
test('G65 R7: later login owns shared storage despite older exchange finishing last',async()=>{
 const s=storage()
 let release
 const old=createWebAuth(config,async url=>new URL(url).pathname==='/auth/login'
  ?new Promise(ok=>{release=ok}):okay({account:{id:'old-id',displayName:'Old'}}),s)
 const pending=old.login('old','password')
 const fresh=createWebAuth(config,async url=>new URL(url).pathname==='/auth/login'
  ?okay({tokenType:'Bearer',accessToken:'new'}):okay({account:{id:'new-id',displayName:'New'}}),s)
 await fresh.login('new','password')
 release(okay({tokenType:'Bearer',accessToken:'old'}))
 assert.equal(await pending,null)
 assert.equal(s.getItem(SESSION_KEY),'new')
 assert.equal(old.authenticated,false)
})

test('G65 R9: superseded established Web identity and late login continuation are inactive',async()=>{
 const s=storage(),calls=[]
 const good=async(url,options)=>{
  const path=new URL(url).pathname
  calls.push(path)
  return path==='/auth/login'?okay({tokenType:'Bearer',accessToken:'old'}):okay({account:{id:'old',displayName:'Old'}})
 }
 const old=createWebAuth(config,good,s)
 await old.login('old','password')
 let release
 const fresh=createWebAuth(config,async url=>new URL(url).pathname==='/auth/login'
  ?new Promise(ok=>{release=ok}):okay({account:{id:'new',displayName:'New'}}),s)
 const pending=fresh.login('new','password')
 assert.equal(old.authenticated,false);assert.equal(old.account,null)
 const restore=createWebAuth(config,good,s)
 assert.equal(await restore.restore(),null)
 release(okay({tokenType:'Bearer',accessToken:'new'}))
 await pending
 assert.equal(fresh.authenticated,true)
 s.setItem(SESSION_KEY,'external')
 assert.equal(fresh.authenticated,false);assert.equal(fresh.account,null)
})

test('G65 R10: cancelled Web login releases restore barrier before old HTTP returns',async()=>{
 for(const method of ['logout','dispose']){
  const s=storage();s.setItem(SESSION_KEY,'stored')
  let release,signal
  const a=createWebAuth(config,async(_url,options)=>{
   signal=options.signal
   return new Promise(ok=>{release=ok})
  },s)
  const pending=a.login('alice','password')
  await a[method]()
  assert.equal(signal.aborted,true)
  const b=createWebAuth(config,async()=>okay({account:{id:'stored-id',displayName:'Stored'}}),s)
  assert.equal((await b.restore()).id,'stored-id')
  release(okay({tokenType:'Bearer',accessToken:'late'}))
  assert.equal(await pending,null)
  assert.equal(b.authenticated,true)
  assert.equal(s.getItem(SESSION_KEY),'stored')
 }
})
test('G65 R10: old Web cancellation/finally cannot clear newer pending login',async()=>{
 const s=storage();s.setItem(SESSION_KEY,'stored')
 let first,second
 const a=createWebAuth(config,()=>new Promise(ok=>{first=ok}),s)
 const b=createWebAuth(config,async url=>new URL(url).pathname==='/auth/login'
  ?new Promise(ok=>{second=ok}):okay({account:{id:'new-id',displayName:'New'}}),s)
 const c=createWebAuth(config,async()=>okay({account:{id:'new-id',displayName:'New'}}),s)
 const old=a.login('old','password'),fresh=b.login('new','password')
 a.dispose()
 assert.equal(await c.restore(),null)
 first(okay({tokenType:'Bearer',accessToken:'old'}))
 assert.equal(await old,null)
 assert.equal(await c.restore(),null)
 second(okay({tokenType:'Bearer',accessToken:'new'}))
 assert.equal((await fresh).id,'new-id')
 assert.equal((await c.restore()).id,'new-id')
})
test('G65 R10: same Web controller restore releases its pending login',async()=>{
 const s=storage();s.setItem(SESSION_KEY,'stored')
 let release
 const a=createWebAuth(config,async url=>new URL(url).pathname==='/auth/login'
  ?new Promise(ok=>{release=ok}):okay({account:{id:'stored-id',displayName:'Stored'}}),s)
 const pending=a.login('old','password')
 assert.equal((await a.restore()).id,'stored-id')
 release(okay({tokenType:'Bearer',accessToken:'late'}))
 assert.equal(await pending,null)
 assert.equal(a.authenticated,true)
})

test('G65 R12: synchronous abort observer starts latest login without stale dispatch or release',async()=>{
 for(const same of [true,false]){
  const s=storage(),calls=[]
  let a,b,latest,oldReply
  const fetcher=async(url,opts)=>{
   const path=new URL(url).pathname
   if(path==='/auth/login'){
    const name=JSON.parse(opts.body).identifier
    calls.push(name)
    if(name==='old')return new Promise(ok=>{
     oldReply=ok
     opts.signal.addEventListener('abort',()=>{latest=b.login('latest','password')},{once:true})
    })
    return okay({tokenType:'Bearer',accessToken:name})
   }
   return okay({account:{id:'latest-id',displayName:'Latest'}})
  }
  a=createWebAuth(config,fetcher,s);b=same?a:createWebAuth(config,fetcher,s)
  const old=a.login('old','password')
  const stale=a.login('stale','password')
  assert.equal(await stale,null)
  assert.equal((await latest).id,'latest-id')
  oldReply(okay({tokenType:'Bearer',accessToken:'old'}))
  assert.equal(await old,null)
  assert.deepEqual(calls,['old','latest'])
  assert.equal(s.getItem(SESSION_KEY),'latest');assert.equal(b.authenticated,true)
 }
})

function reentrantStorage(){
 const s=storage(),hooks={get:null,set:null,remove:null}
 const port={
  getItem(key){const value=s.getItem(key);hooks.get?.(key,value);return value},
  setItem(key,value){s.setItem(key,value);hooks.set?.(key,value)},
  removeItem(key){s.removeItem(key);hooks.remove?.(key)}
 }
 return {s,port,hooks}
}
function identityFetcher(calls){
 return async(url,opts)=>{
  const path=new URL(url).pathname,credential=opts.headers.Authorization?.slice(7)
  calls.push({path,credential})
  if(path==='/auth/login')return okay({tokenType:'Bearer',accessToken:JSON.parse(opts.body).identifier})
  if(path==='/auth/session')return okay({account:{id:credential,displayName:credential}})
  if(path==='/auth/logout')return okay({loggedOut:true})
  throw Error('unexpected endpoint')
 }
}
test('G65 R13: canceled provisional storage write cannot publish identity; logout revokes exact token',async()=>{
 for(const mode of ['dispose','logout']){
  const {s,port,hooks}=reentrantStorage(),calls=[]
  const a=createWebAuth(config,identityFetcher(calls),port)
  let action
  hooks.set=()=>{hooks.set=null;action=a[mode]()}
  assert.equal(await a.login('old','password'),null)
  await action
  assert.equal(a.authenticated,false);assert.equal(a.account,null)
  assert.equal(s.getItem(SESSION_KEY),null)
  assert.deepEqual(calls.filter(x=>x.path==='/auth/logout').map(x=>x.credential),mode==='logout'?['old']:[])
 }
})
test('G65 R13: restore first/final storage read reentry rejects obsolete identity',async()=>{
 for(const read of [1,2])for(const mode of ['dispose','logout','same-login','peer-login']){
  const {s,port,hooks}=reentrantStorage(),calls=[]
  const fetcher=identityFetcher(calls)
  const a=createWebAuth(config,fetcher,port),b=createWebAuth(config,fetcher,port)
  s.setItem(SESSION_KEY,'old')
  let reads=0,next
  hooks.get=()=>{
   if(++reads!==read)return
   hooks.get=null
   next=mode==='same-login'?a.login('new','password'):mode==='peer-login'?b.login('new','password'):a[mode]()
  }
  assert.equal(await a.restore(),null)
  await next
  if(mode.endsWith('-login')){
   assert.equal(s.getItem(SESSION_KEY),'new')
   assert.equal((mode==='same-login'?a:b).account.id,'new')
  }else assert.equal(a.authenticated,false)
  if(read===1)assert.equal(calls.filter(x=>x.path==='/auth/session'&&x.credential==='old').length,0)
 }
})
test('G65 R13: identity getters recheck local and shared ownership after storage reads',async()=>{
 for(const property of ['account','authenticated'])for(const mode of ['dispose','logout','same-login','peer-login']){
  const {s,port,hooks}=reentrantStorage(),calls=[],fetcher=identityFetcher(calls)
  const a=createWebAuth(config,fetcher,port),b=createWebAuth(config,fetcher,port)
  await a.login('old','password')
  let next
  hooks.get=()=>{
   hooks.get=null
   next=mode==='same-login'?a.login('new','password'):mode==='peer-login'?b.login('new','password'):a[mode]()
  }
  assert.equal(a[property],property==='account'?null:false)
  await next
  if(mode.endsWith('-login')){
   assert.equal(s.getItem(SESSION_KEY),'new')
   assert.equal((mode==='same-login'?a:b).account.id,'new')
  }else assert.equal(a.authenticated,false)
 }
})
test('G65 R13: failed restore cleanup cannot delete newer credential or reset newer attempt',async()=>{
 for(const phase of ['get','remove']){
  const {s,port,hooks}=reentrantStorage(),calls=[]
  const fetcher=identityFetcher(calls)
  const a=createWebAuth(config,async(url,opts)=>new URL(url).pathname==='/auth/session'&&opts.headers.Authorization==='Bearer expired'?invalid():fetcher(url,opts),port)
  s.setItem(SESSION_KEY,'expired')
  let reads=0,next
  const start=()=>{hooks.get=null;hooks.remove=null;next=a.login('new','password')}
  if(phase==='get')hooks.get=()=>{if(++reads===2)start()}
  else hooks.remove=start
  assert.equal(await a.restore(),null)
  assert.equal((await next).id,'new')
  assert.equal(s.getItem(SESSION_KEY),'new');assert.equal(a.account.id,'new')
 }
})
test('G65 R13: logout compare read cannot erase replacement credentials; revoke captured old only',async()=>{
 const {s,port,hooks}=reentrantStorage(),calls=[],fetcher=identityFetcher(calls)
 const a=createWebAuth(config,fetcher,port),b=createWebAuth(config,fetcher,port)
 await a.login('old','password')
 let next,removes=0
 hooks.remove=()=>removes++
 hooks.get=()=>{hooks.get=null;next=b.login('new','password')}
 await a.logout();await next
 assert.equal(removes,0);assert.equal(s.getItem(SESSION_KEY),'new')
 assert.equal(b.account.id,'new');assert.equal(a.authenticated,false)
 assert.deepEqual(calls.filter(x=>x.path==='/auth/logout').map(x=>x.credential),['old'])
})
test('G65 R13: provisional write cannot be restored reentrantly; late write after logout is compensated',async()=>{
 for(const before of [true,false]){
  const s=storage(),calls=[]
  let a,b,restore,logout,armed=true
  const port={getItem:s.getItem,removeItem:s.removeItem,setItem(key,value){
   if(!armed){s.setItem(key,value);return}
   armed=false
   if(!before)s.setItem(key,value)
   logout=a.logout();restore=b.restore()
   if(before)s.setItem(key,value)
  }}
  const fetcher=identityFetcher(calls)
  a=createWebAuth(config,fetcher,port);b=createWebAuth(config,fetcher,port)
  assert.equal(await a.login('old','password'),null)
  await logout;assert.equal(await restore,null)
  assert.equal(s.getItem(SESSION_KEY),null)
  assert.equal(a.authenticated,false);assert.equal(b.authenticated,false)
  assert.deepEqual(calls.filter(x=>x.path==='/auth/logout').map(x=>x.credential),['old'])
 }
})
test('G65 R13: newer shared login during provisional write keeps new storage and ownership',async()=>{
 const {s,port,hooks}=reentrantStorage(),calls=[],fetcher=identityFetcher(calls)
 const a=createWebAuth(config,fetcher,port),b=createWebAuth(config,fetcher,port)
 let next
 hooks.set=()=>{hooks.set=null;next=b.login('new','password')}
 assert.equal(await a.login('old','password'),null)
 assert.equal((await next).id,'new')
 assert.equal(s.getItem(SESSION_KEY),'new');assert.equal(b.account.id,'new')
 assert.equal(a.authenticated,false)
})
test('G65 R13: storage write failure preserves original error and clears partial write',async()=>{
 const {s,port,hooks}=reentrantStorage(),calls=[]
 const fault=Error('storage write failed')
 hooks.set=()=>{hooks.set=null;throw fault}
 const a=createWebAuth(config,identityFetcher(calls),port)
 await assert.rejects(a.login('old','password'),error=>error===fault)
 assert.equal(s.getItem(SESSION_KEY),null);assert.equal(a.authenticated,false)
 assert.equal((await a.login('new','password')).id,'new')
})

function cleanupFaultFixture(where){
 const s=storage(),calls=[],diagnostics=[]
 const state={fault:null,revoked:new Set(),logoutError:null,logoutReply:null}
 const port={
  getItem(key){if(state.fault?.where==='get')throw state.fault.error;return s.getItem(key)},
  setItem:s.setItem,
  removeItem(key){
   if(state.fault?.where==='remove-before')throw state.fault.error
   s.removeItem(key)
   if(state.fault?.where==='remove-after')throw state.fault.error
  }
 }
 const fetcher=async(url,opts)=>{
  const path=new URL(url).pathname,credential=opts.headers.Authorization?.slice(7)
  calls.push({path,credential})
  if(path==='/auth/logout'){
   if(state.logoutError)throw state.logoutError
   if(state.logoutReply)return state.logoutReply()
   state.revoked.add(credential)
   return okay({loggedOut:true})
  }
  if(path==='/auth/session'&&state.revoked.has(credential))return invalid()
  return identityFetcher([])(url,opts)
 }
 const arm=()=>{state.fault={where,error:Error('fixture raw storage failure')}}
 const create=()=>createWebAuth(config,fetcher,port)
 return {s,port,calls,state,diagnostics,arm,create}
}
test('G65 R14: cleanup get/remove faults cannot skip captured-token revocation; retained revoked token fails refresh',async()=>{
 for(const where of ['get','remove-before','remove-after']){
  const f=cleanupFaultFixture(where),a=f.create()
  await a.login('old','password')
  f.arm()
  assert.deepEqual(await a.logout(),{storageCleanupFailed:true})
  assert.equal(a.authenticated,false)
  assert.equal(a.account,null)
  assert.deepEqual(f.calls.filter(x=>x.path==='/auth/logout').map(x=>x.credential),['old'])
  assert.equal(f.state.revoked.has('old'),true)
  assert.equal(f.s.getItem(SESSION_KEY),where==='remove-after'?null:'old')
  f.state.fault=null
  const refreshed=f.create()
  assert.equal(await refreshed.restore(),null)
  assert.equal(refreshed.authenticated,false)
  assert.equal(f.s.getItem(SESSION_KEY),null)
  assert.equal((await refreshed.login('new','password')).id,'new')
 }
})
test('G65 R14: failed validation returns null despite compare-read/remove cleanup exceptions',async()=>{
 for(const where of ['get','remove-before','remove-after']){
  const f=cleanupFaultFixture(where)
  f.s.setItem(SESSION_KEY,'expired');f.state.revoked.add('expired')
  let reads=0
  const originalGet=f.port.getItem
  f.port.getItem=key=>{
   if(++reads===2)f.arm()
   return originalGet(key)
  }
  const a=f.create()
  assert.equal(await a.restore(),null)
  assert.equal(a.authenticated,false)
  assert.equal(a.account,null)
  f.state.fault=null
  assert.equal((await a.login('new','password')).id,'new')
 }
})
test('G65 R14: cleanup and diagnostic failure preserve original server logout error or confirmed result',async(t)=>{
 const warn=console.warn
 t.after(()=>{console.warn=warn})
 const warnings=[]
 console.warn=(...args)=>{warnings.push(args);throw Error('diagnostic fault')}
 for(const where of ['get','remove-before','remove-after'])for(const failure of [false,true]){
  const f=cleanupFaultFixture(where),a=f.create(),cause=Error('server logout failure')
  await a.login('old','password')
  f.arm()
  if(failure){
   f.state.logoutError=cause
   await assert.rejects(a.logout(),e=>e===cause)
  }else assert.deepEqual(await a.logout(),{storageCleanupFailed:true})
  assert.equal(a.authenticated,false)
  assert.deepEqual(f.calls.filter(x=>x.path==='/auth/logout').map(x=>x.credential),['old'])
 }
 assert.equal(warnings.length,6)
 assert.ok(warnings.every(args=>args.length===1&&args[0]==='web session storage cleanup failed'))
})
test('G65 R14: retained revoked credential cannot restore even if deletion still fails',async()=>{
 const f=cleanupFaultFixture('remove-before'),a=f.create()
 await a.login('old','password');f.arm()
 await a.logout()
 assert.equal(f.s.getItem(SESSION_KEY),'old')
 const refreshed=f.create()
 assert.equal(await refreshed.restore(),null)
 assert.equal(refreshed.authenticated,false)
 assert.equal(f.s.getItem(SESSION_KEY),'old')
 f.state.fault=null
 assert.equal(await refreshed.restore(),null)
 assert.equal(f.s.getItem(SESSION_KEY),null)
})
test('G65 R14: actual logout confirmation requires Go success envelope and loggedOut true',async()=>{
 for(const reply of [()=>okay({loggedOut:false}),()=>okay({}),()=>new Response(JSON.stringify({code:'ERROR',data:{loggedOut:true}}),{status:200}),invalid]){
  const f=cleanupFaultFixture('remove-before'),a=f.create()
  await a.login('old','password')
  f.arm();f.state.logoutReply=reply
  await assert.rejects(a.logout())
  assert.equal(a.authenticated,false)
  assert.equal(f.state.revoked.has('old'),false)
 }
 const f=cleanupFaultFixture('get'),a=f.create()
 assert.deepEqual(await a.logout(),{storageCleanupFailed:false})
 assert.equal(f.calls.length,0)
 await a.login('old','password')
 assert.deepEqual(await a.logout(),{storageCleanupFailed:false})
 assert.equal(f.state.revoked.has('old'),true)
})
test('G65 R14: cleanup exception after starting a replacement login never revokes or clears new identity',async()=>{
 const {s,port,hooks}=reentrantStorage(),calls=[],fetcher=identityFetcher(calls)
 const a=createWebAuth(config,fetcher,port),b=createWebAuth(config,fetcher,port)
 await a.login('old','password')
 let next
 hooks.get=()=>{hooks.get=null;next=b.login('new','password');throw Error('cleanup fault')}
 assert.deepEqual(await a.logout(),{storageCleanupFailed:true})
 assert.equal((await next).id,'new')
 assert.equal(b.account.id,'new');assert.equal(s.getItem(SESSION_KEY),'new')
 assert.deepEqual(calls.filter(x=>x.path==='/auth/logout').map(x=>x.credential),['old'])
})
