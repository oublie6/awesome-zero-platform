/** Platform-neutral runtime ports. No platform globals are accessed by this module. */
export type Target = 'web' | 'wechat' | 'douyin'
export type Unsubscribe = () => void
export interface PlatformAuth {
  acquireCode(): Promise<string>
}
export interface HttpTransport {
  request<T>(path: string, options?: { method?: string; body?: unknown; signal?: AbortSignal; timeoutMs?: number }): Promise<T>
}
export interface SocketConnection {
  send(message: string): Promise<void>
  close(code?: number, reason?: string): void
  onMessage(handler: (message: string) => void): Unsubscribe
  onClose(handler: () => void): Unsubscribe
}
export interface SocketOptions {
  /** Opening deadline, default 15 seconds. */
  timeoutMs?: number
  /** Portable cancellation: does not require a browser AbortController in mini-games. */
  onCancel?: (handler: () => void) => Unsubscribe
}
export interface SocketTransport {
  connect(url: string, accessToken: string, options?: SocketOptions): Promise<SocketConnection>
}
export interface SessionVault {
  load(): Promise<string | null>
  save(token: string): Promise<void>
  clear(): Promise<void>
}
export interface SecureRandomSource {
  bytes(length: number): Promise<Uint8Array>
}
export interface AppLifecycle {
  onForeground(handler: () => void): Unsubscribe
  onBackground(handler: () => void): Unsubscribe
}
export interface SharePort {
  /** Initiation/copy result; never proof that another player received an invite. */
  invite(roomCode: string): Promise<boolean>
}
export interface GamePlatform {
  readonly target: Target
  readonly auth: PlatformAuth
  readonly http: HttpTransport
  readonly socket: SocketTransport
  readonly vault: SessionVault
  readonly random: SecureRandomSource
  readonly lifecycle: AppLifecycle
  readonly share: SharePort
}
export async function requireSecureBytes(source: SecureRandomSource, length: number): Promise<Uint8Array> {
  if (!Number.isSafeInteger(length) || length < 1 || length > 65536) throw new RangeError('invalid random byte count')
  const bytes = await source.bytes(length)
  if (!(bytes instanceof Uint8Array) || bytes.byteLength !== length) throw new Error('secure random unavailable')
  return bytes
}
export function websocketURL(endpoint: string): { readonly protocol: string; readonly pathname: string } {
  // Do not assume URL exists in a mini-game JS runtime. This is a credential
  // guard; the native transport still validates the endpoint/host itself.
  if (typeof endpoint !== 'string' || !/^wss?:\/\//.test(endpoint)) throw new Error('WebSocket scheme required')
  const match = /^(wss?):\/\/([^/?#]+)(\/[^?#]*)?$/.exec(endpoint)
  if (!match || /[@\\\s]/.test(endpoint)) throw new Error('credentials in WebSocket URL prohibited')
  return {protocol: match[1] + ':', pathname: match[3] ?? '/'}
}

/** A platform-neutral connection manager; credentials are never part of the URL. */
export class ConnectionManager {
  private connection: SocketConnection | null = null
  private unsubscribe: Unsubscribe[] = []
  private generation = 0
  private cancellations = new Set<() => void>()
  constructor(private readonly socket: SocketTransport) {}
  async open(url: string, token: string, onMessage: (message: string) => void, onClose: () => void): Promise<void> {
    websocketURL(url)
    this.close()
    const generation = this.generation
    const connection = await this.socket.connect(url, token, { onCancel: handler => {
      this.cancellations.add(handler)
      return () => { this.cancellations.delete(handler) }
    } })
    if (generation !== this.generation) { connection.close(); throw new Error('connection cancelled') }
    this.connection = connection
    const offMessage = connection.onMessage(message => { if (this.generation === generation) onMessage(message) })
    this.unsubscribe.push(offMessage)
    const offClose = connection.onClose(() => {
      if (this.generation === generation) { this.close(); onClose() }
    })
    if (generation !== this.generation) { offClose(); throw new Error('connection closed during registration') }
    this.unsubscribe.push(offClose)
  }
  async send(value: unknown): Promise<void> {
    if (!this.connection) throw new Error('socket disconnected')
    await this.connection.send(JSON.stringify(value))
  }
  close(): void {
    this.generation++
    const cancellations = [...this.cancellations]
    this.cancellations.clear()
    cancellations.forEach(cancel => { try { cancel() } catch { /* continue owned-resource cleanup */ } })
    this.unsubscribe.splice(0).forEach(fn => { try { fn() } catch { /* continue owned-resource cleanup */ } })
    const connection = this.connection
    this.connection = null
    connection?.close()
  }
}
