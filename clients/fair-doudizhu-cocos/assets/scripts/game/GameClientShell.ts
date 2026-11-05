import { createWireCommand, parseWireResult, type WireCommand, type WireCommandResult } from '@awesome-zero-platform/doudizhu-client/wire'
import { parsePrivateViewResult } from '@awesome-zero-platform/doudizhu-client/view'
import type { FinalEvidenceRead } from '@awesome-zero-platform/doudizhu-client/api'
import { HandStore, GameMessageRouter, type HandSnapshot } from '@awesome-zero-platform/doudizhu-client'
import { ConnectionManager, type SocketTransport } from '@awesome-zero-platform/game-client-core'

/** UI orchestration over confirmed server messages. No generated fake game state. */
/** This is an explicit, correlated pre-commit rejection from the Go
 * dispatcher. INTERNAL, transport loss and timeouts are never definite. */
export class CommandRejectedError extends Error {
  constructor(readonly code: string) {
    super('server rejected command: ' + code)
    this.name = 'CommandRejectedError'
  }
}
const DEFINITIVELY_REJECTED = new Set(['INVALID_REQUEST','CONFLICT','FORBIDDEN','NOT_FOUND'])
const MAX_PENDING_COMMANDS = 32
const SERVER_ERROR_CODES = new Set(['INVALID_REQUEST', 'CONFLICT', 'FORBIDDEN', 'NOT_FOUND', 'INTERNAL'])
export class GameClientShell {
  readonly hand = new HandStore()
  readonly events = new GameMessageRouter()
  private connection: ConnectionManager | null = null
  private disposed = false
  private initialized = false
  private transportReady = false
  private generation = 0
  private syncSequence = 0
  private pendingCommands = new Map<string, { request: WireCommand; resolve: (value: WireCommandResult) => void; reject: (reason: Error) => void; timer: ReturnType<typeof setTimeout> }>()
  private notify: (message: string) => void
  constructor(notify: (message: string) => void) { this.notify = notify }
  async initialize(): Promise<void> {
    if (this.disposed || this.initialized) return
    this.initialized = true
    this.notify('连接模块就绪，请登录并同步服务端房间状态')
    this.events.on('doudizhu.command.result', payload => {
      if (this.disposed) return
      const id = (payload as {requestId?: unknown})?.requestId
      if (typeof id !== 'string') return
      const pending = this.pendingCommands.get(id)
      if (!pending) return
      try {
        const result = parseWireResult(payload, pending.request)
        this.pendingCommands.delete(id)
        clearTimeout(pending.timer)
        pending.resolve(result)
      } catch (error) {
        this.pendingCommands.delete(id)
        clearTimeout(pending.timer)
        pending.reject(error instanceof Error ? error : new Error('invalid command response'))
      }
    })
    this.events.on('doudizhu.hand.changed', payload => {
      if (this.hand.snapshot && (payload as {handId?: string}).handId !== this.hand.snapshot.handId) return
      if (this.hand.status === 'connected') this.hand.setStatus('synchronizing')
      this.notify('收到牌局状态变化，请同步权威快照')
    })
    this.events.on('doudizhu.hand.snapshot', payload => {
      if (this.disposed) return
      const event = payload as {handId?: string; knownVersion?: number; notModified?: boolean; view?: unknown}
      if (event.notModified) {
        if (this.hand.status === 'synchronizing' &&
            event.handId === this.hand.snapshot?.handId && event.knownVersion === this.hand.snapshot?.version) {
          this.hand.setStatus('connected')
        }
        return
      }
      if (!event.view) return
      try {
        const {version, view} = parsePrivateViewResult(event.view)
        if (event.handId !== view.public.handId) throw new Error('hand identity mismatch')
        if (view.public.phase !== 'BIDDING' && view.public.phase !== 'PLAYING' && view.public.phase !== 'COMPLETED') {
          // Unknown live phases require an explicit protocol upgrade, not an invented state.
          throw new Error('unsupported hand phase')
        }
        this.acceptVerifiedSnapshot({
          handId: view.public.handId,
          version,
          phase: view.public.phase,
          view: view as unknown as Readonly<Record<string, unknown>>
        })
      } catch {
        this.notify('私有牌局数据校验失败，请重新同步')
      }
    })
    this.events.on('doudizhu.hand.evidence', payload => {
      if (this.hand.snapshot && (payload as {handId?: string}).handId !== this.hand.snapshot.handId) return
      // Keep transport status truthful; block actions while terminal DTO mapping is pending.
      if (this.hand.status === 'connected') this.hand.setStatus('synchronizing')
      this.notify('已收到终局证据，牌局操作已停止')
    })
    this.events.on('doudizhu.error', payload => {
      if (this.disposed) return
      const event = payload as {requestId?: unknown; code?: unknown}
      if (typeof event.requestId === 'string' && SERVER_ERROR_CODES.has(event.code as string)) {
        const pending = this.pendingCommands.get(event.requestId)
        if (pending) {
          this.pendingCommands.delete(event.requestId)
          clearTimeout(pending.timer)
          pending.reject(DEFINITIVELY_REJECTED.has(event.code as string)
            ? new CommandRejectedError(event.code as string)
            : new Error('server error; command outcome unknown: ' + event.code))
        }
      }
      this.notify('服务器拒绝操作，请同步后重试')
    })
  }
  async connect(socket: SocketTransport, endpoint: string, token: string): Promise<void> {
    if (this.disposed) throw new Error('client disposed')
    const generation = ++this.generation
    this.transportReady = false
    this.rejectPending('connection replaced; command outcome unknown')
    this.connection?.close()
    if (this.hand.status !== 'disconnected') this.hand.setStatus('disconnected')
    const connection = new ConnectionManager(socket)
    this.connection = connection
    this.hand.setStatus('reconnecting')
    try {
      await connection.open(endpoint, token,
        raw => { try { this.events.receive(raw) } catch { this.notify('收到不合法的服务器消息') } },
        () => { if (!this.disposed && generation === this.generation) { this.transportReady = false; this.rejectPending('connection lost; command outcome unknown'); this.hand.setStatus('disconnected'); this.notify('连接中断，请重连') } })
      if (this.disposed || generation !== this.generation) return
      this.transportReady = true
      this.hand.setStatus('synchronizing')
      this.notify('已连接，等待房间或牌局状态同步')
    } catch (error) {
      if (this.disposed || generation !== this.generation) return
      this.transportReady = false
      this.hand.setStatus('disconnected')
      throw error
    }
  }
  clearHand(): void {
    if (this.pendingCommands.size) throw new Error('hand command still pending')
    this.hand.clearHand()
  }

  /** Only call with authenticated HTTP evidence already validated by
   * DoudizhuHTTPClient.finalEvidence. A terminal archive replaces live
   * presentation state; no private hand/cards are invented after cleanup. */
  acceptVerifiedFinalEvidence(evidence: FinalEvidenceRead): void {
    if (this.disposed) return
    const handId = evidence.handId
    const phase = String(evidence.status).toUpperCase()
    const verification = evidence.verification
    if (!handId || !Number.isSafeInteger(evidence.finalVersion) || evidence.finalVersion < 1 ||
        !['COMPLETED','ABORTED'].includes(phase) ||
        verification.handId !== handId || verification.finalVersion !== evidence.finalVersion ||
        typeof verification.recordDigest !== 'string' || !/^[0-9a-f]{64}$/.test(verification.recordDigest) ||
        !Array.isArray(verification.seatPoints) || verification.seatPoints.length !== 3 ||
        !verification.seatPoints.every(Number.isSafeInteger)) {
      throw new Error('invalid verified final evidence')
    }
    const prior = this.hand.snapshot
    if (prior && prior.handId !== handId) throw new Error('terminal evidence hand mismatch')
    if (prior && evidence.finalVersion < prior.version) throw new Error('stale terminal evidence')
    if (prior && prior.version === evidence.finalVersion && prior.phase === phase) return
    if (prior && prior.version === evidence.finalVersion) throw new Error('terminal evidence version conflict')
    const snapshot: HandSnapshot = {
      handId, version: evidence.finalVersion, phase: phase as 'COMPLETED'|'ABORTED',
      view: { terminalEvidence: {
        winnerSeat: verification.winnerSeat ?? null,
        seatPoints: verification.seatPoints,
        recordDigest: verification.recordDigest,
      } },
    }
    this.acceptVerifiedSnapshot(snapshot)
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
  /** Transmit only actual Go dispatcher commands; timeout is an unknown outcome, never a proof of failure. */
  async submitCommand(input: Omit<WireCommand, 'v'>, exactUnknownRetry = false): Promise<WireCommandResult> {
    if (this.disposed || !this.connection || !this.transportReady) throw new Error('game connection not ready')
    const request = createWireCommand(input)
    if (request.type.startsWith('hand.')) {
      if (exactUnknownRetry) {
        // A previously sent command may have committed before a lost reply.
        // A stale local version MUST NOT block the same-ID verification.
        // Ownership, seat, command fingerprint and accepted result are
        // still checked by the Go application and MySQL receipt journal.
        if (this.hand.snapshot?.handId !== request.aggregateId)
          throw new Error('reconnected hand identity does not match pending command')
      } else if (!this.hand.canSubmit || this.hand.snapshot?.handId !== request.aggregateId ||
                 this.hand.snapshot.version !== request.expectedVersion) {
        throw new Error('hand not eligible for command')
      }
    }
    if (this.pendingCommands.has(request.requestId)) throw new Error('command already pending')
    if (this.pendingCommands.size >= MAX_PENDING_COMMANDS) throw new Error('pending command capacity exceeded')
    return new Promise<WireCommandResult>((resolve, reject) => {
      const generation = this.generation
      const connection = this.connection!
      const entry = {request, resolve, reject, timer: undefined as unknown as ReturnType<typeof setTimeout>}
      entry.timer = setTimeout(() => {
        if (this.pendingCommands.get(request.requestId) !== entry) return
        this.pendingCommands.delete(request.requestId)
        reject(new Error('command timeout; outcome unknown, retry with same requestId'))
      }, 15000)
      this.pendingCommands.set(request.requestId, entry)
      void connection.send({id: request.requestId, type: 'doudizhu.command', payload: request}).catch(() => {
        if (this.generation !== generation || this.pendingCommands.get(request.requestId) !== entry) return
        clearTimeout(entry.timer)
        this.pendingCommands.delete(request.requestId)
        reject(new Error('command send failed; outcome unknown'))
      })
    })
  }
  private rejectPending(reason: string): void {
    for (const pending of this.pendingCommands.values()) {
      clearTimeout(pending.timer)
      pending.reject(new Error(reason))
    }
    this.pendingCommands.clear()
  }
  dispose(): void {
    if (this.disposed) return
    this.disposed = true
    this.generation++
    this.transportReady = false
    this.rejectPending('client disposed; command outcome unknown')
    this.connection?.close()
    this.events.clear()
    this.hand.reset()
  }
}
