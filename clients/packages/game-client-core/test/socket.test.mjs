import { test } from 'node:test'
import assert from 'node:assert/strict'
import { ConnectionManager } from '../dist/index.js'
import { readFileSync } from 'node:fs'
import ts from 'typescript'
import { MiniSocketTransport } from '../dist/minigame.js'

import { Task } from './fakes.mjs'
const tick = () => new Promise(resolve => setImmediate(resolve))
test('mini creation success is not handshake; synchronous success cannot use an unassigned task', async () => {
  const task = new Task(); let ready = false
  const transport = new MiniSocketTransport({ connectSocket(args) { args.success?.(); return task } })
  const pending = transport.connect('wss://example.org/ws', 'test-token').then(c => { ready = true; return c })
  // Attach rejection handling before advancing the event loop.
  const result = pending.then(value => ({value}), error => ({error}))
  await tick(); assert.equal(ready, false)
  task.emit('open'); const {value, error} = await result
  assert.equal(error, undefined); assert.ok(value)
  await value.send('hello'); assert.deepEqual(task.sent, ['hello'])
  value.close(); assert.equal(task.listeners, 0)
})
test('mini synchronous fail rejects safely and closes returned task', async () => {
  const task = new Task()
  const pending = new MiniSocketTransport({connectSocket(args) { args.fail?.({errMsg:'failure'}); return task }}).connect('wss://example.org/ws','test-token')
  await assert.rejects(pending, /socket.*fail/i); assert.equal(task.closes,1); assert.equal(task.listeners,0)
})
test('mini error/close before open rejects and removes all listeners', async () => {
  for (const event of ['error','close']) {
    const task = new Task()
    const pending = new MiniSocketTransport({connectSocket() { return task }}).connect('wss://example.org/ws','test-token')
    task.emit(event, {errMsg:'failure'})
    await assert.rejects(pending, /socket/i); assert.equal(task.listeners,0)
  }
})
test('closing manager cancels the pending real mini adapter immediately', async () => {
  const task = new Task()
  const manager = new ConnectionManager(new MiniSocketTransport({connectSocket() { return task }}))
  const pending = manager.open('wss://example.org/ws','test-token',()=>{},()=>{})
  const handled = pending.catch(()=>{})
  manager.close(); await tick(); assert.equal(task.closes,1); assert.equal(task.listeners,0)
  await handled
})
test('old callbacks cannot deliver messages or close a newer manager connection', async () => {
  const connections = []
  const manager = new ConnectionManager({async connect() {
    const c = {send:async()=>{},close(){},onMessage(cb){this.message=cb;return()=>{}},onClose(cb){this.closed=cb;return()=>{}}}
    connections.push(c); return c
  }})
  let messages=0, closed=0
  await manager.open('wss://example.org/ws','test-token',()=>messages++,()=>closed++)
  await manager.open('wss://example.org/ws','test-token',()=>messages++,()=>closed++)
  connections[0].message('stale'); connections[0].closed()
  assert.equal(messages,0); assert.equal(closed,0)
  connections[1].message('fresh'); assert.equal(messages,1); manager.close()
})
test('close in listener registration cannot leave a manager connected',async()=>{
  let removed=0,closed=0
  const manager=new ConnectionManager({async connect(){return {send:async()=>{},close(){closed++},onMessage(){return()=>removed++},onClose(cb){cb();return()=>removed++}}}})
  await assert.rejects(manager.open('wss://x/ws','token',()=>{},()=>{}),/closed/)
  assert.equal(closed,1);assert.equal(removed,2);await assert.rejects(manager.send({}),/disconnected/)
})

test('Creator 3.8 compatible ES5 output invokes all Set-managed socket cleanup callbacks', async () => {
  // Exercise the actual platform-neutral TypeScript sources after downlevel
  // compilation rather than trusting the ES2022 Node test build.
  const compile = (url, load) => {
    const source = readFileSync(new URL(url, import.meta.url), 'utf8')
    const emitted = ts.transpileModule(source, { compilerOptions: {
      target: ts.ScriptTarget.ES5, module: ts.ModuleKind.CommonJS, downlevelIteration: false,
    } }).outputText
    const module = { exports: {} }
    new Function('module', 'exports', 'require', emitted)(module, module.exports, load)
    return module.exports
  }
  const operations=compile('../src/operation.ts',()=>{throw Error('unexpected operation dependency')})
  const { ConnectionManager: LegacyManager } = compile('../src/index.ts', path => {
    if(path==='./operation.js')return operations
    throw Error('unexpected dependency '+path)
  })
  // Exercise the new shared operation listener cleanup in the same downlevel
  // runtime as the socket cleanup, not a stubbed dependency.
  const slot=new operations.OperationSlot(),first=slot.begin()
  const p1=first.wait(()=>new Promise(()=>{})),p2=first.wait(()=>new Promise(()=>{}))
  const rejected=Promise.all([assert.rejects(p1,/cancelled/),assert.rejects(p2,/cancelled/)])
  first.onCancel(()=>{throw Error('cleanup fault')})
  let last=0;first.onCancel(()=>last++)
  assert.equal(slot.cancel(),true)
  assert.equal(first.cleanupFailures,1);assert.equal(last,1)
  const replacement=slot.begin()
  await rejected
  first.finish()
  assert.equal(slot.begin(),null)
  replacement.finish()
  assert.ok(slot.begin())
  const manager = new LegacyManager({connect: async () => { throw Error('no connection required') }})
  let cancelled = 0
  manager.cancellations.add(() => { cancelled++ })
  manager.cancellations.add(() => { cancelled++ })
  manager.close()
  assert.equal(cancelled, 2)
  assert.equal(manager.cancellations.size, 0)

  const { openSocket } = compile('../src/socket.ts', name => {
    if(name==='./operation.js')return operations
    assert.equal(name, './network-timeout.js')
    return { networkTimeout: value => value ?? 15000 }
  })
  let events
  let closed = 0
  const driver = {
    bind(handlers) { events = handlers; return () => {} },
    send: async () => {},
    close() { closed++ },
  }
  const opening = openSocket(driver, { timeoutMs: 5000 })
  events.open()
  const connection = await opening
  let callbacks = 0
  connection.onClose(() => { callbacks++ })
  connection.onClose(() => { callbacks++ })
  connection.close()
  assert.equal(callbacks, 2)
  assert.equal(closed, 1)
})

test('stalled native sends are bounded, close cancels every pending send and late callbacks are inert',async()=>{
 const task=new Task(),calls=[]
 task.send=args=>calls.push(args)
 const opening=new MiniSocketTransport({connectSocket:()=>task}).connect('wss://example.org/ws','token')
 task.emit('open')
 const connection=await opening
 const promises=[]
 for(let i=0;i<32;i++)promises.push(connection.send('frame-'+i,1000))
 const settled=Promise.all(promises.map(p=>assert.rejects(p,/closed during send/)))
 await assert.rejects(connection.send('overflow'),/capacity/)
 assert.equal(calls.length,32)
 connection.close()
 await settled
 assert.equal(task.closes,1);assert.equal(task.listeners,0)
 for(const call of calls){call.success();call.fail(Error('late'))}
 await assert.rejects(connection.send('after-close'),/not open/)
})
test('native send deadline closes task and releases all sends without SDK callbacks',async()=>{
 const task=new Task(),calls=[]
 task.send=args=>calls.push(args)
 const opening=new MiniSocketTransport({connectSocket:()=>task}).connect('wss://example.org/ws','token')
 task.emit('open')
 const connection=await opening
 const first=assert.rejects(connection.send('stalled',10),/timed out/)
 const second=assert.rejects(connection.send('also stalled',1000),/timed out/)
 await Promise.all([first,second])
 assert.equal(task.closes,1);assert.equal(task.listeners,0)
 calls.forEach(call=>call.success())
 await assert.rejects(connection.send('no more queued work'),/not open/)
})
test('native send failure and repeated callbacks cannot retain capacity or close a healthy connection',async()=>{
 const task=new Task(),calls=[]
 task.send=args=>calls.push(args)
 const opening=new MiniSocketTransport({connectSocket:()=>task}).connect('wss://example.org/ws','token')
 task.emit('open')
 const connection=await opening
 for(let i=0;i<40;i++){
  const failed=assert.rejects(connection.send('frame',1000),/send failed/)
  calls.at(-1).fail(Error('native failure'))
  await failed
  calls.at(-1).success()
 }
 assert.equal(task.closes,0)
 const pending=connection.send('ok',1000)
 calls.at(-1).success();await pending
 connection.close()
})

test('G65 R11: manager registration failure and throwing native close still release owned listeners',async()=>{
 for(const phase of ['register','close']){
  let removed=0,closed=0
  const manager=new ConnectionManager({async connect(){return {
   send:async()=>{},onMessage:()=>()=>{removed++;throw Error('off fault')},
   onClose:()=>{if(phase==='register')throw Error('registration fault');return()=>removed++},
   close:()=>{closed++;throw Error('close fault')}
  }}})
  if(phase==='register')await assert.rejects(manager.open('wss://x/ws','token',()=>{},()=>{}),/registration fault/)
  else {await manager.open('wss://x/ws','token',()=>{},()=>{});manager.close()}
  assert.equal(manager.isOpen,false)
  assert.equal(removed,phase==='register'?1:2);assert.equal(closed,1)
  manager.close();assert.equal(closed,1)
 }
})
test('G65 R11: mini off failure does not skip remaining listeners, sends or close observers',async()=>{
 const task=new Task(),calls=[]
 task.offOpen=()=>{calls.push('offOpen');throw Error('off fault')}
 task.send=()=>{}
 const opening=new MiniSocketTransport({connectSocket:()=>task}).connect('wss://example.org/ws','token')
 task.emit('open');const connection=await opening
 const pending=assert.rejects(connection.send('pending',5000),/closed during send/)
 connection.onClose(()=>{throw Error('observer fault')})
 connection.onClose(()=>calls.push('closed'))
 connection.close();await pending
 assert.equal(task.closes,1);assert.equal(task.listeners,1)
 assert.deepEqual(calls,['offOpen','closed'])
})

test('G65 R12: synchronous unsubscribe/close reentry cannot install a superseded manager connection',async()=>{
 for(const phase of ['unsubscribe','close','native-close']){
  const seen=[],connections=[],messages=[],closed=[]
  let latest,manager
  const startLatest=()=>{if(!latest)latest=manager.open('wss://x/ws','latest',x=>messages.push(x),()=>closed.push('latest'))}
  manager=new ConnectionManager({async connect(_url,token){
   seen.push(token)
   const c={token,closes:0,sent:[],send:async raw=>c.sent.push(raw),
    onMessage(cb){c.message=cb;return()=>{if(token==='initial'&&(phase==='unsubscribe'||phase==='native-close'))startLatest()}},
    onClose(cb){c.closed=cb;return()=>{}},
    close(){c.closes++;if(token==='initial'&&phase==='close')startLatest()}}
   connections.push(c);return c
  }})
  await manager.open('wss://x/ws','initial',x=>messages.push(x),()=>closed.push('initial'))
  if(phase==='native-close')connections[0].closed()
  else await assert.rejects(manager.open('wss://x/ws','superseded',()=>messages.push('stale'),()=>closed.push('stale')),/cancelled/)
  await latest
  assert.deepEqual(seen,['initial','latest'])
  connections[0].message('old');connections[1].message('new')
  await manager.send({hello:true})
  assert.deepEqual(messages,['new']);assert.deepEqual(closed,[])
  assert.equal(manager.isOpen,true);assert.equal(connections[0].closes,1)
  assert.equal(connections[1].sent.length,1)
  manager.close();assert.equal(connections[1].closes,1)
 }
})
test('G65 R12: late cancel registration does not affect replacement callback or throw',async()=>{
 let oldOptions,latestOptions,resolveOld,calls=0
 const c=()=>({send:async()=>{},close(){},onMessage(){return()=>{}},onClose(){return()=>{}}})
 const manager=new ConnectionManager({connect(_url,token,options){
  if(token==='old'){oldOptions=options;return new Promise(ok=>{resolveOld=ok})}
  latestOptions=options;return Promise.resolve(c())
 }})
 const old=manager.open('wss://x/ws','old',()=>{},()=>{})
 const rejected=assert.rejects(old,/cancelled/)
 await manager.open('wss://x/ws','new',()=>{},()=>{})
 const handler=()=>{calls++;throw Error('cancel fault')}
 oldOptions.onCancel(handler)()
 latestOptions.onCancel(handler)
 resolveOld(c());await rejected
 assert.equal(calls,1)
 manager.close();assert.equal(calls,2)
})
