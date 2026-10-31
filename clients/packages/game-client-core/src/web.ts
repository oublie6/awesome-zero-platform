import type { AppLifecycle, HttpTransport, SecureRandomSource, SessionVault, SharePort, SocketConnection, SocketTransport, Unsubscribe } from './index.js'
import { websocketURL } from './index.js'

export class WebHttp implements HttpTransport {
  constructor(private readonly base = '/api') {}
  async request<T>(path: string, options: { method?: string; body?: unknown; signal?: AbortSignal } = {}): Promise<T> {
    if (!path.startsWith('/') || path.startsWith('//')) throw new Error('relative API path required')
    const response = await fetch(this.base + path, {
      method: options.method ?? 'GET',
      signal: options.signal,
      headers: options.body === undefined ? undefined : { 'Content-Type': 'application/json' },
      body: options.body === undefined ? undefined : JSON.stringify(options.body)
    })
    if (!response.ok) throw new Error(`HTTP ${response.status}`)
    return response.json() as Promise<T>
  }
}
export class WebSocketTransport implements SocketTransport {
  async connect(endpoint: string, accessToken: string): Promise<SocketConnection> {
    websocketURL(endpoint)
    if (!accessToken) throw new Error('access token required')
    return new Promise((resolve, reject) => {
      const ws = new WebSocket(endpoint, ['bearer', accessToken])
      const onOpen = () => {
        cleanup()
        const messages = new Set<(value: string) => void>()
        const closes = new Set<() => void>()
        ws.addEventListener('message', event => {
          if (typeof event.data === 'string') messages.forEach(cb => cb(event.data))
        })
        ws.addEventListener('close', () => closes.forEach(cb => cb()))
        resolve({
          send(message) {
            if (ws.readyState !== WebSocket.OPEN) return Promise.reject(new Error('socket not open'))
            ws.send(message)
            return Promise.resolve()
          },
          close(code, reason) { ws.close(code, reason) },
          onMessage(cb) { messages.add(cb); return () => { messages.delete(cb) } },
          onClose(cb) { closes.add(cb); return () => { closes.delete(cb) } }
        })
      }
      const onFailure = () => { cleanup(); reject(new Error('WebSocket connection failed')) }
      const cleanup = () => { ws.removeEventListener('open', onOpen); ws.removeEventListener('error', onFailure); ws.removeEventListener('close', onFailure) }
      ws.addEventListener('open', onOpen, { once: true })
      ws.addEventListener('error', onFailure, { once: true })
      ws.addEventListener('close', onFailure, { once: true })
    })
  }
}
export class WebRandom implements SecureRandomSource {
  async bytes(length: number): Promise<Uint8Array> {
    if (!Number.isSafeInteger(length) || length < 1 || length > 65536 || !globalThis.crypto?.getRandomValues) throw new Error('secure random unavailable')
    const output = new Uint8Array(length)
    globalThis.crypto.getRandomValues(output)
    return output
  }
}
export class WebSessionVault implements SessionVault {
  constructor(private readonly key = 'fair-doudizhu-session') {}
  async load() { return sessionStorage.getItem(this.key) }
  async save(token: string) { sessionStorage.setItem(this.key, token) }
  async clear() { sessionStorage.removeItem(this.key) }
}
export class WebLifecycle implements AppLifecycle {
  onForeground(cb: () => void): Unsubscribe {
    const listener = () => { if (document.visibilityState === 'visible') cb() }
    document.addEventListener('visibilitychange', listener)
    return () => document.removeEventListener('visibilitychange', listener)
  }
  onBackground(cb: () => void): Unsubscribe {
    const listener = () => { if (document.visibilityState === 'hidden') cb() }
    document.addEventListener('visibilitychange', listener)
    return () => document.removeEventListener('visibilitychange', listener)
  }
}
export class WebShare implements SharePort {
  async invite(roomCode: string): Promise<boolean> {
    if (!roomCode) return false
    if (!navigator.clipboard?.writeText) return false
    await navigator.clipboard.writeText(roomCode)
    return true
  }
}
