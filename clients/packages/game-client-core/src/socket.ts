import { networkTimeout } from './network-timeout.js'
import { runCleanup } from './operation.js'
import type { SocketConnection, SocketOptions, Unsubscribe } from './index.js'

/** Shared adapter lifecycle. Drivers supply platform events; core never reads SDK globals. */
export interface SocketDriver {
  bind(events: { open(): void; message(data: string): void; close(): void; error(): void }): Unsubscribe
  /** Callback only: openSocket owns the bounded Promise and its cleanup. */
  send(data: string, complete:(error?:Error)=>void): void
  close(code?: number, reason?: string): void
}
export function openSocket(driver: SocketDriver, options: SocketOptions = {}): Promise<SocketConnection> {
  return new Promise((resolve, reject) => {
    let state: 'connecting' | 'open' | 'closed' = 'connecting'
    let unbind: Unsubscribe = () => {}
    let uncancel: Unsubscribe = () => {}
    const timeout = networkTimeout(options.timeoutMs)
    const timer = setTimeout(() => finish(new Error('socket connection timed out')), timeout)
    const messages = new Set<(data: string) => void>()
    const closes = new Set<() => void>()
    const sends = new Set<(error:Error)=>void>()
    const maxPendingSends=32
    const finish = (error?: Error, code?: number, reason?: string) => {
      if (state === 'closed') return
      const pending = state === 'connecting'
      state = 'closed'
      clearTimeout(timer)
      const pendingSends:Array<(error:Error)=>void>=[]
      sends.forEach(cancel=>pendingSends.push(cancel))
      sends.clear()
      // Snapshot all ownership before invoking any external cleanup.
      const callbacks: Array<() => void> = []
      closes.forEach(cb => callbacks.push(cb))
      messages.clear(); closes.clear()
      const offDriver=unbind,offCancel=uncancel
      unbind=()=>{};uncancel=()=>{}
      runCleanup([
        ...pendingSends.map(cancel=>()=>cancel(error??new Error('socket closed during send'))),
        offDriver,offCancel,()=>driver.close(code,reason)
      ])
      if (pending) reject(error ?? new Error('socket closed before open'))
      runCleanup(callbacks)
    }
    const isClosed = () => state === 'closed'
    const connection: SocketConnection = {
      send(data,timeoutMs) {
        if (state !== 'open') return Promise.reject(new Error('socket not open'))
        if(sends.size>=maxPendingSends)return Promise.reject(new Error('socket send capacity exceeded'))
        let deadline:number
        try{deadline=networkTimeout(timeoutMs)}catch(error){return Promise.reject(error)}
        return new Promise<void>((resolveSend,rejectSend)=>{
          // One Promise owned by the transport. Close a stalled socket so
          // retries cannot accumulate on the same native task.
          let complete:((error?:Error)=>void)|undefined
          const callback=(error?:Error)=>complete?.(error)
          const cancel=(error:Error)=>callback(error)
          const sendTimer=setTimeout(()=>{
            const error=new Error('socket send timed out')
            callback(error)
            finish(error)
          },deadline)
          complete=error=>{
            complete=undefined
            clearTimeout(sendTimer)
            sends.delete(cancel)
            if(error)rejectSend(error);else resolveSend()
          }
          sends.add(cancel)
          try{driver.send(data,callback)}
          catch{callback(new Error('socket send failed'))}
        })
      },
      close(code, reason) { finish(undefined, code, reason) },
      onMessage(cb) {
        if (state === 'closed') return () => {}
        messages.add(cb); return () => { messages.delete(cb) }
      },
      onClose(cb) {
        if (state === 'closed') { runCleanup([cb]); return () => {} }
        closes.add(cb); return () => { closes.delete(cb) }
      }
    }
    try {
      unbind = driver.bind({
        open() { if (state === 'connecting') { state = 'open'; clearTimeout(timer); resolve(connection) } },
        message(data) {
          if (state === 'open') messages.forEach(cb => {
            if(isClosed())return
            try{cb(data)}catch{runCleanup([()=>{throw Error('socket message observer failed')}])}
          })
        },
        close() { finish() },
        error() { finish(new Error('socket connection failed')) }
      })
      if (isClosed()) {
        const off=unbind;unbind=()=>{};runCleanup([off])
        return
      }
      uncancel = options.onCancel?.(() => finish(new Error('socket connection cancelled'))) ?? (() => {})
      if (isClosed()) {const off=uncancel;uncancel=()=>{};runCleanup([off])}
    } catch {
      finish(new Error('socket event registration failed'))
    }
  })
}
