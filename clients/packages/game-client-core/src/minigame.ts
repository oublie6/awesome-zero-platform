import type { HttpTransport, SecureRandomSource, SocketConnection, SocketTransport, SocketOptions } from './index.js'
import { websocketURL } from './index.js'
import { openSocket } from './socket.js'

/** Minimal SDK shapes injected by the composition root. No references to global wx or tt. */
export interface MiniSocket {
  onOpen(handler: () => void): void
  offOpen?(handler: () => void): void
  send(args: { data: string; success?: () => void; fail?: (error: unknown) => void }): void
  close(args?: { code?: number; reason?: string }): void
  onMessage(handler: (event: { data: string | ArrayBuffer }) => void): void
  offMessage?(handler: (event: { data: string | ArrayBuffer }) => void): void
  onClose(handler: () => void): void
  offClose?(handler: () => void): void
  onError(handler: (error: unknown) => void): void
  offError?(handler: (error: unknown) => void): void
}
export interface MiniSDK {
  connectSocket(args: {
    url: string
    protocols?: string[]
    header?: Record<string, string>
    success?: () => void
    fail?: (error: unknown) => void
  }): MiniSocket
  request?(args: {
    url: string; method?: string; data?: unknown; header?: Record<string, string>
    success: (value: { statusCode: number; data: unknown }) => void
    fail: (error: unknown) => void
  }): { abort?(): void } | void
  getRandomValues?(args: { length: number; success: (value: { randomValues: ArrayBuffer }) => void; fail: (error: unknown) => void }): void
}
/** The server requires a bearer subprotocol: validate support in developer tools and real devices. */
export class MiniSocketTransport implements SocketTransport {
  constructor(private readonly sdk: MiniSDK) {}
  async connect(url: string, accessToken: string, options?: SocketOptions): Promise<SocketConnection> {
    const endpoint = websocketURL(url)
    if (endpoint.protocol !== 'wss:') throw new Error('mini-game socket requires WSS')
    if (!accessToken) throw new Error('access token required')
    // SDK success means task creation, not handshake. Defer synchronous fail until
    // the returned task and its listeners exist; never read uninitialized bindings.
    let fail: () => void = () => { failed = true }
    let failed = false
    let socket: MiniSocket
    try { socket = this.sdk.connectSocket({url, protocols: ['bearer', accessToken], fail: () => fail()}) }
    catch { throw new Error('mini-game socket creation failed') }
    if (!socket || !['onOpen','onClose','onMessage','onError','send','close'].every(key =>
      typeof (socket as unknown as Record<string, unknown>)[key] === 'function')) {
      try { socket?.close() } catch { /* unsupported task */ }
      throw new Error('mini-game SocketTask capability unsupported')
    }
    if (failed) {
      try { socket.close() } catch { /* task creation already failed */ }
      throw new Error('mini-game socket creation failed')
    }
    return openSocket({
      bind(events) {
        fail = events.error
        const message = (event: {data: string | ArrayBuffer}) => {
          if (typeof event.data === 'string') events.message(event.data)
        }
        // Error/close listeners precede open. Optional off methods are not assumed
        // on Douyin; internal callbacks are made inert and the owned task is closed.
        const cleanup = () => {
          socket.offOpen?.(events.open); socket.offMessage?.(message)
          socket.offClose?.(events.close); socket.offError?.(events.error)
        }
        try {
          socket.onError(events.error); socket.onClose(events.close)
          socket.onMessage(message); socket.onOpen(events.open)
          if (failed) events.error()
        } catch { cleanup(); throw new Error('mini-game socket listener failed') }
        return cleanup
      },
      send: data => new Promise<void>((resolve, reject) => socket.send({data, success: resolve, fail: () => reject(new Error('mini-game socket send failed'))})),
      close: (code, reason) => socket.close({code, reason})
    }, options)
  }
}

export class MiniHttp implements HttpTransport {
  constructor(private readonly sdk: MiniSDK, private readonly base: string) {}
  request<T>(path: string, opts: { method?: string; body?: unknown; signal?: AbortSignal } = {}): Promise<T> {
    if (!this.sdk.request) return Promise.reject(new Error('platform HTTP unsupported'))
    if (!path.startsWith('/') || path.startsWith('//')) return Promise.reject(new Error('relative API path required'))
    return new Promise((resolve, reject) => {
      let finished = false
      let cancelled = false
      let task: {abort?(): void} | void
      const abort = () => {
        if (finished) return
        cancelled = true
        complete(new Error('HTTP request cancelled'))
        task?.abort?.()
      }
      const complete = (error?: Error, data?: unknown) => {
        if (finished) return
        finished = true
        opts.signal?.removeEventListener('abort', abort)
        if (error) reject(error); else resolve(data as T)
      }
      if (opts.signal?.aborted) { abort(); return }
      opts.signal?.addEventListener('abort', abort, {once:true})
      try {
        task = this.sdk.request!({
          url: this.base + path, method: opts.method ?? 'GET', data: opts.body,
          header: opts.body === undefined ? undefined : {'content-type': 'application/json'},
          success: value => value.statusCode >= 200 && value.statusCode < 300
            ? complete(undefined, value.data) : complete(new Error(`HTTP ${value.statusCode}`)),
          fail: () => complete(new Error('platform HTTP failed'))
        })
        if (cancelled) task?.abort?.()
      } catch { complete(new Error('platform HTTP failed')) }
    })
  }
}

export class MiniRandom implements SecureRandomSource {
  constructor(private readonly sdk: Pick<MiniSDK, 'getRandomValues'>) {}
  bytes(length: number): Promise<Uint8Array> {
    if (!Number.isSafeInteger(length) || length < 1 || length > 65536 || !this.sdk.getRandomValues)
      return Promise.reject(new Error('secure random unavailable on this platform'))
    return new Promise((resolve, reject) => this.sdk.getRandomValues!({
      length,
      success: value => {
        if (!(value.randomValues instanceof ArrayBuffer) || value.randomValues.byteLength !== length) {
          reject(new Error('invalid platform random result'))
        } else resolve(new Uint8Array(value.randomValues))
      },
      fail: reject
    }))
  }
}
