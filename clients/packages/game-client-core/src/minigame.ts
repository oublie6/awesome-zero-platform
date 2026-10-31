import type { HttpTransport, SecureRandomSource, SocketConnection, SocketTransport } from './index.js'
import { websocketURL } from './index.js'

/** Minimal SDK shapes injected by the composition root. No references to global wx or tt. */
export interface MiniSocket {
  send(args: { data: string; success?: () => void; fail?: (error: unknown) => void }): void
  close(args?: { code?: number; reason?: string }): void
  onMessage(handler: (event: { data: string | ArrayBuffer }) => void): void
  offMessage(handler: (event: { data: string | ArrayBuffer }) => void): void
  onClose(handler: () => void): void
  offClose(handler: () => void): void
  onError(handler: (error: unknown) => void): void
  offError(handler: (error: unknown) => void): void
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
    url: string; method?: string; data?: unknown
    success: (value: { statusCode: number; data: unknown }) => void
    fail: (error: unknown) => void
  }): void
  getRandomValues?(args: { length: number; success: (value: { randomValues: ArrayBuffer }) => void; fail: (error: unknown) => void }): void
}
/** The server requires a bearer subprotocol: validate support in developer tools and real devices. */
export class MiniSocketTransport implements SocketTransport {
  constructor(private readonly sdk: MiniSDK) {}
  connect(url: string, accessToken: string): Promise<SocketConnection> {
    websocketURL(url)
    if (!accessToken) return Promise.reject(new Error('access token required'))
    return new Promise((resolve, reject) => {
      let settled = false
      let socket: MiniSocket
      socket = this.sdk.connectSocket({
        url, protocols: ['bearer', accessToken],
        success: () => {
          if (settled) return
          settled = true
          socket.offError(onError)
          resolve({
            send: data => new Promise<void>((done, fail) => socket.send({ data, success: done, fail })),
            close: (code, reason) => socket.close({ code, reason }),
            onMessage(cb) {
              const fn = (evt: { data: string | ArrayBuffer }) => { if (typeof evt.data === 'string') cb(evt.data) }
              socket.onMessage(fn)
              return () => socket.offMessage(fn)
            },
            onClose(cb) { socket.onClose(cb); return () => socket.offClose(cb) }
          })
        },
        fail: error => onError(error)
      })
      const onError = (error: unknown) => {
        if (settled) return
        settled = true
        reject(error instanceof Error ? error : new Error('mini-game socket failed'))
      }
      socket.onError(onError)
    })
  }
}
export class MiniHttp implements HttpTransport {
  constructor(private readonly sdk: MiniSDK, private readonly base: string) {}
  request<T>(path: string, opts: { method?: string; body?: unknown } = {}): Promise<T> {
    if (!this.sdk.request) return Promise.reject(new Error('platform HTTP unsupported'))
    if (!path.startsWith('/') || path.startsWith('//')) return Promise.reject(new Error('relative API path required'))
    return new Promise((resolve, reject) => this.sdk.request!({
      url: this.base + path, method: opts.method ?? 'GET', data: opts.body,
      success: value => value.statusCode >= 200 && value.statusCode < 300
        ? resolve(value.data as T) : reject(new Error(`HTTP ${value.statusCode}`)),
      fail: reject
    }))
  }
}
export class MiniRandom implements SecureRandomSource {
  constructor(private readonly sdk: MiniSDK) {}
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
