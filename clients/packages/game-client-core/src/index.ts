import { runCleanup } from './operation.js'
export { OperationScope, OperationSlot, OperationCancelled, runCleanup } from './operation.js'

/** Platform-neutral runtime ports. No platform globals are accessed by this module. */
export type Target = 'web' | 'wechat' | 'douyin'
export type Unsubscribe = () => void
export interface PlatformAuth {
  acquireCode(timeoutMs?:number,onCancel?:(handler:()=>void)=>Unsubscribe): Promise<string>
}
export interface HttpTransport {
  request<T>(path: string, options?: { method?: string; body?: unknown; signal?: AbortSignal; timeoutMs?: number; accessToken?: string }): Promise<T>
}
export interface SocketConnection {
  /** Completion means local transport acceptance, not a server ACK. */
  send(message: string, timeoutMs?: number): Promise<void>
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
  /** Adapters sharing a backing store/key share in-process mutation ordering. */
  readonly coordinationScope?: { readonly storage: object; readonly key: string }
  load(): Promise<string | null>
  save(token: string): Promise<void>
  clear(): Promise<void>
  /** Atomic compare-and-clear on the built-in synchronous storage adapters. */
  clearIf?(token:string):Promise<void>
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
  private ready=false
  private cancellations = new Set<() => void>()
  constructor(private readonly socket: SocketTransport) {}
  get isOpen():boolean { return this.ready && this.connection!==null }
  async open(url: string, token: string, onMessage: (message: string) => void, onClose: () => void): Promise<void> {
    websocketURL(url)
    // Reserve before old cleanup: unsubscribe/close can synchronously start
    // a newer open. Never adopt the generation that the callback created.
    const generation=++this.generation
    this.detach()
    if(generation!==this.generation)throw new Error('connection cancelled')
    const cancellations=this.cancellations
    const connection = await this.socket.connect(url, token, { onCancel: handler => {
      if(generation!==this.generation){runCleanup([handler]);return ()=>{}}
      cancellations.add(handler)
      return () => { cancellations.delete(handler) }
    } })
    if (generation !== this.generation) { runCleanup([()=>connection.close()]); throw new Error('connection cancelled') }
    this.connection = connection
    try{
      const offMessage = connection.onMessage(message => { if (this.generation === generation) onMessage(message) })
      if(generation!==this.generation){runCleanup([offMessage]);throw new Error('connection closed during registration')}
      this.unsubscribe.push(offMessage)
      const offClose = connection.onClose(() => {
        if(this.generation!==generation)return
        const closing=++this.generation
        this.detach()
        if(this.generation===closing)onClose()
      })
      if (generation !== this.generation) { runCleanup([offClose]); throw new Error('connection closed during registration') }
      this.unsubscribe.push(offClose)
      this.ready=true
    }catch(error){
      if(generation===this.generation)this.close()
      throw error
    }
  }
  async send(value: unknown, timeoutMs?:number): Promise<void> {
    const connection=this.connection,generation=this.generation
    if (!connection || !this.ready) throw new Error('socket disconnected')
    const encoded=JSON.stringify(value)
    if(generation!==this.generation || connection!==this.connection || !this.ready)
      throw new Error('connection cancelled')
    await connection.send(encoded,timeoutMs)
  }
  close(): void {
    this.generation++
    this.detach()
  }
  private detach():void{
    this.ready=false
    // Snapshot via forEach: Creator 3.8 may lower Set spread to concat(Set).
    const cancellations: Unsubscribe[] = []
    this.cancellations.forEach(cancel => cancellations.push(cancel))
    this.cancellations.clear()
    this.cancellations=new Set()
    const subscriptions=this.unsubscribe.splice(0)
    const connection = this.connection
    this.connection = null
    // Snapshot and detach before external callbacks can re-enter open/close.
    runCleanup([...cancellations,...subscriptions,()=>connection?.close()])
  }
}
