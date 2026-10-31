import {test} from 'node:test'
import assert from 'node:assert/strict'
import { MiniHttp, MiniRandom, MiniSocketTransport } from '../dist/minigame.js'
import { createMiniPlatform } from '../dist/platform.js'
import { WebSocketTransport, WebHttp, WebRandom, WebSessionVault, WebLifecycle, WebShare } from '../dist/web.js'
import { websocketURL } from '../dist/index.js'
import {Task} from './fakes.mjs'
const override = (t, key, value) => {
  const original=Object.getOwnPropertyDescriptor(globalThis,key)
  Object.defineProperty(globalThis,key,{value,configurable:true,writable:true})
  t.after(()=>{ if(original)Object.defineProperty(globalThis,key,original);else delete globalThis[key] })
}
test('Web bearer protocols never enter URL; failed and cancelled attempts close', async t=>{
  class WS extends Task {
    static OPEN=1; static all=[]; readyState=0
    constructor(url,protocols){super();this.url=url;this.protocols=protocols;WS.all.push(this)}
    addEventListener(name,cb){this.add(name,cb)} removeEventListener(name,cb){this.off(name,cb)}
    send(data){this.sent.push(data)}
    close(){this.readyState=3;super.close()}
  }
  override(t,'WebSocket',WS)
  const transport=new WebSocketTransport()
  let pending=transport.connect('wss://example.org/ws','secret-token')
  let ws=WS.all.at(-1)
  assert.deepEqual(ws.protocols,['bearer','secret-token']);assert.equal(ws.url,'wss://example.org/ws')
  ws.readyState=1;ws.emit('open');const c=await pending
  let received=0,closes=0;const off=c.onMessage(()=>received++);c.onClose(()=>closes++)
  ws.emit('message',{data:'hello'});off();ws.emit('message',{data:'again'});assert.equal(received,1)
  await c.send('out');c.close();assert.equal(closes,1);assert.equal(ws.listeners,0)
  await assert.rejects(c.send('after close'),/not open/)
  for(const event of ['error','close']){
    pending=transport.connect('wss://example.org/ws','secret-token');ws=WS.all.at(-1);ws.emit(event)
    await assert.rejects(pending,/socket/);assert.equal(ws.listeners,0);assert.equal(ws.closes,1)
  }
  let cancel
  pending=transport.connect('wss://example.org/ws','secret-token',{onCancel(cb){cancel=cb;return()=>{}}})
  ws=WS.all.at(-1);cancel();await assert.rejects(pending,/cancelled/);assert.equal(ws.closes,1)
})
test('URL security guard works without URL globals and rejects every query/fragment', t=>{
  override(t,'URL',undefined)
  for(const url of ['wss://x/ws?access_token=s','wss://x/ws?jwt=s','wss://x/ws#s','wss://s@x/ws','wss://x/\nfoo','wss://x\\s']) assert.throws(()=>websocketURL(url),/prohibited/)
  assert.equal(websocketURL('wss://example.org/ws').pathname,'/ws')
})
test('mini tasks without undocumented off methods are supported; callbacks become inert on close', async()=>{
  const task=new Task();task.offOpen=task.offClose=task.offError=task.offMessage=undefined
  const promise=new MiniSocketTransport({connectSocket(){return task}}).connect('wss://example.org/ws','token')
  task.emit('open');const c=await promise;let calls=0;c.onMessage(()=>calls++);c.close();task.emit('message',{data:'stale'})
  assert.equal(calls,0);assert.equal(task.closes,1)
})
test('unsupported SocketTask and synchronous SDK exceptions fail explicitly', async()=>{
  for(const sdk of [{connectSocket(){return {close(){}}}},{connectSocket(){throw Error('sdk')}}]) await assert.rejects(new MiniSocketTransport(sdk).connect('wss://x/ws','token'),/unsupported|creation failed/)
})
test('mini HTTP status, cancellation and unsupported capability are explicit', async()=>{
  let args,aborts=0
  const http=new MiniHttp({request(a){args=a;return{abort(){aborts++}}}},'https://example.org/api')
  const control=new AbortController();const p=http.request('/read',{signal:control.signal});control.abort()
  await assert.rejects(p,/cancelled/);assert.equal(aborts,1);args.success({statusCode:200,data:{late:true}})
  const p2=http.request('/write',{method:'POST',body:{x:1}});assert.equal(args.header['content-type'],'application/json');args.success({statusCode:403,data:{}});await assert.rejects(p2,/403/)
  const p3=http.request('/read');args.success({statusCode:200,data:{ok:true}});assert.deepEqual(await p3,{ok:true})
  await assert.rejects(new MiniHttp({},'https://example.org').request('/read'),/unsupported/)
})
test('Web HTTP preserves method, JSON, abort signal and rejects non-2xx',async t=>{
  let args;override(t,'fetch',async (url,options)=>{args={url,options};return{ok:true,json:async()=>({ok:true})}})
  const signal=new AbortController().signal
  assert.deepEqual(await new WebHttp('/api').request('/read',{method:'POST',body:{a:1},signal}),{ok:true})
  assert.equal(args.url,'/api/read');assert.equal(args.options.signal,signal);assert.equal(args.options.body,'{"a":1}')
  override(t,'fetch',async()=>({ok:false,status:500}));await assert.rejects(new WebHttp().request('/read'),/500/)
  await assert.rejects(new WebHttp().request('//host'),/relative/)
})
test('secure entropy fails closed for missing/wrong-sized SDK responses and missing browser crypto',async t=>{
  await assert.rejects(new MiniRandom({}).bytes(32),/unavailable/)
  await assert.rejects(new MiniRandom({getRandomValues(a){a.success({randomValues:new ArrayBuffer(2)})}}).bytes(32),/invalid/)
  override(t,'crypto',undefined);await assert.rejects(new WebRandom().bytes(32),/unavailable/)
})
test('WeChat uses UserCryptoManager and preserves method receiver',async()=>{
  let calls=0;const manager={getRandomValues(a){assert.equal(this,manager);calls++;a.success({randomValues:new ArrayBuffer(a.length)})}}
  const sdk={getUserCryptoManager(){return manager}}
  const platform=createMiniPlatform('wechat',sdk,'https://example.org/api')
  assert.equal((await platform.random.bytes(32)).length,32);assert.equal(calls,1)
  await assert.rejects(createMiniPlatform('wechat',{},'https://example.org').random.bytes(32),/unsupported/)
  await assert.rejects(createMiniPlatform('douyin',{},'https://example.org').random.bytes(32),/unavailable/)
})
test('mini login/storage/lifecycle/share capability failures do not silently succeed',async()=>{
  const p=createMiniPlatform('douyin',{},'https://example.org')
  await assert.rejects(p.auth.acquireCode());await assert.rejects(p.vault.load(),/unsupported/);await assert.rejects(p.vault.save('t'),/unsupported/);await assert.rejects(p.vault.clear(),/unsupported/)
  assert.throws(()=>p.lifecycle.onForeground(()=>{}),/unsupported/);assert.equal(await p.share.invite('ROOM'),false)
})
test('mini session, lifecycle disposal and ephemeral login code',async()=>{
  const store=new Map(),listeners=new Set()
  const p=createMiniPlatform('wechat',{login(a){a.success({code:'ephemeral-code'})},getStorageSync:k=>store.get(k),setStorageSync:(k,v)=>store.set(k,v),removeStorageSync:k=>store.delete(k),onShow:cb=>listeners.add(cb),offShow:cb=>listeners.delete(cb)},'https://example.org')
  assert.equal(await p.auth.acquireCode(),'ephemeral-code');await p.vault.save('session');assert.equal(await p.vault.load(),'session');await p.vault.clear();assert.equal(await p.vault.load(),null)
  const off=p.lifecycle.onForeground(()=>{});assert.equal(listeners.size,1);off();off();assert.equal(listeners.size,0)
})
test('Web lifecycle unregisters; tab storage clears; unsupported share returns false',async t=>{
  const doc=new EventTarget();doc.visibilityState='visible';override(t,'document',doc)
  const map=new Map();override(t,'sessionStorage',{getItem:k=>map.get(k)??null,setItem:(k,v)=>map.set(k,v),removeItem:k=>map.delete(k)})
  const v=new WebSessionVault();await v.save('token');assert.equal(await v.load(),'token');await v.clear();assert.equal(await v.load(),null)
  let calls=0;const off=new WebLifecycle().onForeground(()=>calls++);doc.dispatchEvent(new Event('visibilitychange'));off();doc.dispatchEvent(new Event('visibilitychange'));assert.equal(calls,1)
  override(t,'navigator',{});assert.equal(await new WebShare().invite('ROOM'),false)
})
test('Douyin anonymous codes are not accepted as authenticated login codes; share rejection is explicit',async()=>{
 const p=createMiniPlatform('douyin',{login(a){a.success({anonymousCode:'not-an-identity'})},shareAppMessage(a){a.fail()}},'https://example.org')
 await assert.rejects(p.auth.acquireCode(),/no code/);assert.equal(await p.share.invite('ROOM'),false)
})
test('synchronous abort during native request creation aborts its returned task',async()=>{
 const control=new AbortController();let aborts=0
 const http=new MiniHttp({request(){control.abort();return{abort(){aborts++}}}},'https://example.org')
 await assert.rejects(http.request('/read',{signal:control.signal}),/cancelled/);assert.equal(aborts,1)
})
