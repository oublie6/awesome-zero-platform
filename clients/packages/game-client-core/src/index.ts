/** Platform-neutral runtime ports. No platform globals are accessed by this module. */
export type Target = 'web' | 'wechat' | 'douyin'
export type Unsubscribe = () => void
export interface PlatformAuth {
  acquireCode(): Promise<string>
}
export interface HttpTransport {
  request<T>(path: string, options?: { method?: string; body?: unknown; signal?: AbortSignal }): Promise<T>
}
export interface SocketConnection {
  send(message: string): Promise<void>
  close(code?: number, reason?: string): void
  onMessage(handler: (message: string) => void): Unsubscribe
  onClose(handler: () => void): Unsubscribe
}
export interface SocketTransport {
  connect(url: string, accessToken: string): Promise<SocketConnection>
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
export function websocketURL(endpoint: string): URL {
  const url = new URL(endpoint)
  if (url.protocol !== 'wss:' && url.protocol !== 'ws:') throw new Error('WebSocket scheme required')
  if (url.username || url.password || url.searchParams.has('token') || url.searchParams.has('access_token')) {
    throw new Error('credentials in WebSocket URL prohibited')
  }
  return url
}
