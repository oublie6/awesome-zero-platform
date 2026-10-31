import type { AppLifecycle, HttpTransport, SecureRandomSource, SessionVault, SharePort, SocketConnection, SocketTransport, SocketOptions, Unsubscribe } from './index.js'
import { websocketURL } from './index.js'
import { openSocket } from './socket.js'

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
  async connect(endpoint: string, accessToken: string, options?: SocketOptions): Promise<SocketConnection> {
    websocketURL(endpoint)
    if (!accessToken) throw new Error('access token required')
    const ws = new WebSocket(endpoint, ['bearer', accessToken])
    return openSocket({
      bind(events) {
        const message = (event: MessageEvent) => { if (typeof event.data === 'string') events.message(event.data) }
        ws.addEventListener('open', events.open)
        ws.addEventListener('message', message)
        ws.addEventListener('close', events.close)
        ws.addEventListener('error', events.error)
        return () => {
          ws.removeEventListener('open', events.open)
          ws.removeEventListener('message', message)
          ws.removeEventListener('close', events.close)
          ws.removeEventListener('error', events.error)
        }
      },
      async send(data) { if (ws.readyState !== WebSocket.OPEN) throw new Error('socket not open'); ws.send(data) },
      close(code, reason) { ws.close(code, reason) }
    }, options)
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
  async load() { const value = sessionStorage.getItem(this.key); return value?.trim() ? value : null }
  async save(token: string) { if (!token.trim()) throw new Error('empty session token'); sessionStorage.setItem(this.key, token) }
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
