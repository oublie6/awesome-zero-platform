import { test } from 'node:test'
import assert from 'node:assert/strict'
import { ConnectionManager } from '../dist/index.js'
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
