import { networkTimeout } from './network-timeout.js'
import type { SocketConnection, SocketOptions, Unsubscribe } from './index.js'

/** Shared adapter lifecycle. Drivers supply platform events; core never reads SDK globals. */
export interface SocketDriver {
  bind(events: { open(): void; message(data: string): void; close(): void; error(): void }): Unsubscribe
  send(data: string): Promise<void>
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
    const finish = (error?: Error, code?: number, reason?: string) => {
      if (state === 'closed') return
      const pending = state === 'connecting'
      state = 'closed'
      clearTimeout(timer)
      try { unbind() } catch { /* callback guards still prevent delivery */ }
      try { uncancel() } catch { /* owned socket must still close */ }
      // Close even a failed opening attempt; late native callbacks are inert.
      try { driver.close(code, reason) } catch { /* the native socket may already be closed */ }
      if (pending) reject(error ?? new Error('socket closed before open'))
      const callbacks = [...closes]
      messages.clear(); closes.clear()
      callbacks.forEach(cb => cb())
    }
    const isClosed = () => state === 'closed'
    const connection: SocketConnection = {
      async send(data) {
        if (state !== 'open') throw new Error('socket not open')
        await driver.send(data)
      },
      close(code, reason) { finish(undefined, code, reason) },
      onMessage(cb) {
        if (state === 'closed') return () => {}
        messages.add(cb); return () => { messages.delete(cb) }
      },
      onClose(cb) {
        if (state === 'closed') { cb(); return () => {} }
        closes.add(cb); return () => { closes.delete(cb) }
      }
    }
    try {
      unbind = driver.bind({
        open() { if (state === 'connecting') { state = 'open'; clearTimeout(timer); resolve(connection) } },
        message(data) { if (state === 'open') messages.forEach(cb => cb(data)) },
        close() { finish() },
        error() { finish(new Error('socket connection failed')) }
      })
      if (isClosed()) unbind()
      uncancel = options.onCancel?.(() => finish(new Error('socket connection cancelled'))) ?? (() => {})
      if (isClosed()) uncancel()
    } catch {
      finish(new Error('socket event registration failed'))
    }
  })
}
