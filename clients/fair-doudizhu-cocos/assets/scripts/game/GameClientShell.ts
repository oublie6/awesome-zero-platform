import { HandStore, GameMessageRouter, type HandSnapshot } from '@awesome-zero-platform/doudizhu-client'
import { ConnectionManager, type SocketTransport } from '@awesome-zero-platform/game-client-core'

/** UI orchestration over confirmed server messages. No generated fake game state. */
export class GameClientShell {
  readonly hand = new HandStore()
  readonly events = new GameMessageRouter()
  private connection: ConnectionManager | null = null
  private disposed = false
  private notify: (message: string) => void
  constructor(notify: (message: string) => void) { this.notify = notify }
  async initialize(): Promise<void> {
    if (this.disposed) return
    this.notify('请选择 Web / 微信 / 抖音平台，并完成安全登录后进入房间')
    this.events.on('doudizhu.hand.changed', () => {
      this.notify('收到牌局状态变化，请同步权威快照')
    })
    this.events.on('doudizhu.hand.snapshot', payload => {
      // The server's private snapshot is validated and mapped in the protocol
      // integration layer in Goal 0046; never assume the wire shape here.
      if (!this.disposed) this.notify('已收到服务端私有牌局快照')
      void payload
    })
    this.events.on('doudizhu.error', () => {
      if (!this.disposed) this.notify('服务器拒绝操作，请同步后重试')
    })
  }
  async connect(socket: SocketTransport, endpoint: string, token: string): Promise<void> {
    if (this.disposed) throw new Error('client disposed')
    this.connection?.close()
    this.connection = new ConnectionManager(socket)
    this.hand.setStatus('reconnecting')
    try {
      await this.connection.open(endpoint, token,
        raw => { try { this.events.receive(raw) } catch { this.notify('收到不合法的服务器消息') } },
        () => { if (!this.disposed) { this.hand.setStatus('disconnected'); this.notify('连接中断，请重连') } })
      this.hand.setStatus('synchronizing')
      this.notify('已连接，等待房间或牌局状态同步')
    } catch (error) {
      this.hand.setStatus('disconnected')
      throw error
    }
  }
  acceptVerifiedSnapshot(snapshot: HandSnapshot): void {
    if (this.disposed) return
    if (this.hand.apply(snapshot)) {
      if (this.hand.status === 'synchronizing') this.hand.setStatus('connected')
      this.notify(`牌局 ${snapshot.handId} · 版本 ${snapshot.version} · ${snapshot.phase}`)
    }
  }
  async syncHand(handId: string, knownVersion?: number): Promise<void> {
    if (!this.connection) throw new Error('not connected')
    await this.connection.send({
      id: `sync-${handId}-${knownVersion ?? 'initial'}`,
      type: 'doudizhu.hand.sync',
      payload: { handId, ...(knownVersion === undefined ? {} : {knownVersion}) }
    })
  }
  dispose(): void {
    if (this.disposed) return
    this.disposed = true
    this.connection?.close()
    this.events.clear()
    this.hand.reset()
  }
}
