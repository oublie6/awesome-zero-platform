import { HandStore, GameMessageRouter, type HandSnapshot } from '@awesome-zero-platform/doudizhu-client'
import { ConnectionManager, type SocketTransport } from '@awesome-zero-platform/game-client-core'

/** UI orchestration over confirmed server messages. No generated fake game state. */
export class GameClientShell {
  readonly hand = new HandStore()
  readonly events = new GameMessageRouter()
  private connection: ConnectionManager | null = null
  private disposed = false
  private initialized = false
  private generation = 0
  private syncSequence = 0
  private notify: (message: string) => void
  constructor(notify: (message: string) => void) { this.notify = notify }
  async initialize(): Promise<void> {
    if (this.disposed || this.initialized) return
    this.initialized = true
    this.notify('请选择 Web / 微信 / 抖音平台，并完成安全登录后进入房间')
    this.events.on('doudizhu.hand.changed', payload => {
      if (this.hand.snapshot && (payload as {handId?: string}).handId !== this.hand.snapshot.handId) return
      if (this.hand.status === 'connected') this.hand.setStatus('synchronizing')
      this.notify('收到牌局状态变化，请同步权威快照')
    })
    this.events.on('doudizhu.hand.snapshot', payload => {
      // The server's private snapshot is validated and mapped in the protocol
      // integration layer in Goal 0046; never assume the wire shape here.
      if (!this.disposed) this.notify('已收到服务端私有牌局快照')
      const event = payload as {handId?: string; knownVersion?: number; notModified?: boolean}
      if (event.notModified && this.hand.status === 'synchronizing' &&
          event.handId === this.hand.snapshot?.handId && event.knownVersion === this.hand.snapshot?.version) {
        this.hand.setStatus('connected')
      }
    })
    this.events.on('doudizhu.hand.evidence', payload => {
      if (this.hand.snapshot && (payload as {handId?: string}).handId !== this.hand.snapshot.handId) return
      // Keep transport status truthful; block actions while terminal DTO mapping is pending.
      if (this.hand.status === 'connected') this.hand.setStatus('synchronizing')
      this.notify('已收到终局证据，牌局操作已停止')
    })
    this.events.on('doudizhu.error', () => {
      if (!this.disposed) this.notify('服务器拒绝操作，请同步后重试')
    })
  }
  async connect(socket: SocketTransport, endpoint: string, token: string): Promise<void> {
    if (this.disposed) throw new Error('client disposed')
    const generation = ++this.generation
    this.connection?.close()
    if (this.hand.status !== 'disconnected') this.hand.setStatus('disconnected')
    const connection = new ConnectionManager(socket)
    this.connection = connection
    this.hand.setStatus('reconnecting')
    try {
      await connection.open(endpoint, token,
        raw => { try { this.events.receive(raw) } catch { this.notify('收到不合法的服务器消息') } },
        () => { if (!this.disposed) { this.hand.setStatus('disconnected'); this.notify('连接中断，请重连') } })
      if (this.disposed || generation !== this.generation) return
      this.hand.setStatus('synchronizing')
      this.notify('已连接，等待房间或牌局状态同步')
    } catch (error) {
      if (this.disposed || generation !== this.generation) return
      this.hand.setStatus('disconnected')
      throw error
    }
  }
  acceptVerifiedSnapshot(snapshot: HandSnapshot): void {
    if (this.disposed) return
    const applied = this.hand.apply(snapshot)
    const confirmed = this.hand.snapshot
    if (this.hand.status === 'synchronizing' && confirmed?.handId === snapshot.handId && confirmed.version === snapshot.version) {
      this.hand.setStatus('connected')
    }
    if (applied) this.notify(`牌局 ${snapshot.handId} · 版本 ${snapshot.version} · ${snapshot.phase}`)
  }
  async syncHand(handId: string, knownVersion?: number): Promise<void> {
    if (this.disposed || !this.connection || !['synchronizing','connected'].includes(this.hand.status)) throw new Error('not connected')
    if (!handId.trim() || handId !== handId.trim() || handId.length > 128 ||
        (knownVersion !== undefined && (!Number.isSafeInteger(knownVersion) || knownVersion < 0))) throw new Error('invalid hand sync')
    await this.connection.send({
      id: `sync-${++this.syncSequence}`,
      type: 'doudizhu.hand.sync',
      payload: { v: 'doudizhu-realtime-sync-v1', handId, ...(knownVersion === undefined ? {} : {knownVersion}) }
    })
  }
  dispose(): void {
    if (this.disposed) return
    this.disposed = true
    this.generation++
    this.connection?.close()
    this.events.clear()
    this.hand.reset()
  }
}
