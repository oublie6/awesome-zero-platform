/**
 * Fair Doudizhu client-side state coordination.
 * Only server-confirmed versions enter this store. The server owns rules and cards.
 */
export type HandPhase = 'FAIRNESS_COMMITTING' | 'FAIRNESS_REVEALING' | 'WAITING_PUBLIC_BEACON' | 'DEALING' |
  'BIDDING' | 'PLAYING' | 'SETTLING' | 'COMPLETED' | 'CANCELLED' | 'ABORTED' | 'EXPIRED'
export interface HandSnapshot {
  handId: string
  version: number
  phase: HandPhase
  /** Server-authorized private view; never broadcast it to a generic topic. */
  view: Readonly<Record<string, unknown>>
}
export type ReconnectStatus = 'disconnected' | 'reconnecting' | 'synchronizing' | 'connected'
export class HandStore {
  private current: HandSnapshot | null = null
  private statusValue: ReconnectStatus = 'disconnected'
  get snapshot(): HandSnapshot | null { return this.current }
  get status(): ReconnectStatus { return this.statusValue }
  get canSubmit(): boolean { return this.statusValue === 'connected' && this.current !== null &&
    !['COMPLETED','CANCELLED','ABORTED','EXPIRED'].includes(this.current.phase) }
  setStatus(next: ReconnectStatus): void {
    const allowed: Record<ReconnectStatus, readonly ReconnectStatus[]> = {
      disconnected: ['reconnecting'],
      reconnecting: ['synchronizing', 'disconnected'],
      synchronizing: ['connected', 'disconnected'],
      connected: ['disconnected', 'synchronizing']
    }
    if (this.statusValue !== next && !allowed[this.statusValue].includes(next)) {
      throw new Error(`invalid reconnect transition: ${this.statusValue} -> ${next}`)
    }
    this.statusValue = next
  }
  apply(snapshot: HandSnapshot): boolean {
    if (!snapshot || !snapshot.handId || !Number.isSafeInteger(snapshot.version) || snapshot.version < 1 ||
        !['FAIRNESS_COMMITTING','FAIRNESS_REVEALING','WAITING_PUBLIC_BEACON','DEALING','BIDDING','PLAYING','SETTLING','COMPLETED','CANCELLED','ABORTED','EXPIRED'].includes(snapshot.phase) ||
        !snapshot.view || typeof snapshot.view !== 'object' || Array.isArray(snapshot.view)) {
      throw new Error('invalid authoritative hand snapshot')
    }
    if (this.current && snapshot.handId === this.current.handId && snapshot.version <= this.current.version) {
      return false
    }
    if (this.current && snapshot.handId !== this.current.handId) {
      throw new Error('hand identity changed without explicit reset')
    }
    this.current = freezeJSON(snapshot)
    return true
  }
  reset(): void { this.current = null; this.statusValue = 'disconnected' }
}
export interface CommandEnvelope {
  v: 'fair-doudizhu-command-v1'
  name: string
  commandId: string
  aggregateType: 'room' | 'hand'
  aggregateId: string
  clientSeq: number
  expectedVersion: number
  issuedAt: string
  expiresAt: string
  payload: Readonly<Record<string, unknown>>
}
const commandTargets: Readonly<Record<string, 'room' | 'hand'>> = {
  'doudizhu.room.create.v1': 'room', 'doudizhu.room.join.v1': 'room',
  'doudizhu.room.leave.v1': 'room', 'doudizhu.room.ready.set.v1': 'room',
  'doudizhu.room.hand.start.v1': 'room',
  'doudizhu.hand.fairness.commit.submit.v1': 'hand',
  'doudizhu.hand.fairness.reveal.submit.v1': 'hand',
  'doudizhu.hand.cancel.v1': 'hand'
}
function record(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object' && !Array.isArray(value)
}
function freezeJSON<T>(value: T): T {
  const copy = JSON.parse(JSON.stringify(value)) as T
  const freeze = (item: unknown): void => {
    if (item && typeof item === 'object') { Object.values(item).forEach(freeze); Object.freeze(item) }
  }
  freeze(copy); return copy
}
/** Durable application envelope, NOT the Go HTTP/WSS dispatcher request DTO. */
export function requireValidCommand(command: CommandEnvelope): void {
  if (!record(command) || command.v !== 'fair-doudizhu-command-v1' ||
      typeof command.name !== 'string' || !Object.hasOwn(commandTargets, command.name) ||
      commandTargets[command.name] !== command.aggregateType ||
      typeof command.commandId !== 'string' || !command.commandId.trim() || command.commandId !== command.commandId.trim() || command.commandId.length > 128 ||
      typeof command.aggregateId !== 'string' || !command.aggregateId.trim() || command.aggregateId !== command.aggregateId.trim() || command.aggregateId.length > 128 ||
      !Number.isSafeInteger(command.clientSeq) || command.clientSeq < 1 ||
      !Number.isSafeInteger(command.expectedVersion) || command.expectedVersion < 0 ||
      (command.name === 'doudizhu.room.create.v1' ? command.expectedVersion !== 0 : command.expectedVersion === 0) ||
      !record(command.payload) ||
      ['accountId','actorId','ownerId','role','seat'].some(key => key in command || key in command.payload)) {
    throw new Error('invalid command envelope')
  }
  const utcMillis = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$/
  const issued = Date.parse(command.issuedAt), expires = Date.parse(command.expiresAt)
  if (!utcMillis.test(command.issuedAt) || !utcMillis.test(command.expiresAt) ||
      !Number.isFinite(issued) || !Number.isFinite(expires) ||
      new Date(issued).toISOString() !== command.issuedAt || new Date(expires).toISOString() !== command.expiresAt ||
      expires <= issued || expires - issued > 120_000) throw new Error('invalid command envelope time window')
}

export type GameMessageType =
  | 'doudizhu.command.result'
  | 'doudizhu.hand.changed'
  | 'doudizhu.hand.snapshot'
  | 'doudizhu.hand.evidence'
  | 'doudizhu.error'

export interface RealtimeEnvelope {
  id?: string
  type: GameMessageType | 'system.hello' | 'system.pong'
  payload?: unknown
}

/** Routes only recognized protocol messages, never fabricates authoritative state. */
export class GameMessageRouter {
  private listeners = new Map<string, Set<(payload: unknown) => void>>()
  on(type: GameMessageType, handler: (payload: unknown) => void): () => void {
    const set = this.listeners.get(type) ?? new Set<(payload: unknown) => void>()
    set.add(handler)
    this.listeners.set(type, set)
    return () => {
      set.delete(handler)
      if (set.size === 0 && this.listeners.get(type) === set) this.listeners.delete(type)
    }
  }
  receive(raw: string): void {
    let value: unknown
    try { value = JSON.parse(raw) } catch { throw new Error('invalid realtime JSON') }
    if (!value || typeof value !== 'object' || Array.isArray(value)) throw new Error('invalid realtime envelope')
    const msg = value as Record<string, unknown>
    if (typeof msg.type !== 'string') throw new Error('missing realtime type')
    const recognized = ['doudizhu.command.result','doudizhu.hand.changed','doudizhu.hand.snapshot','doudizhu.hand.evidence','doudizhu.error']
    if (!recognized.includes(msg.type)) return
    if (!record(msg.payload) || msg.payload.v !== (msg.type === 'doudizhu.command.result'
      ? 'doudizhu-api-command-result-v1' : 'doudizhu-realtime-event-v1')) throw new Error('invalid realtime payload version')
    if (msg.type === 'doudizhu.hand.snapshot' && msg.payload.notModified === true &&
        (typeof msg.payload.handId !== 'string' || !Number.isSafeInteger(msg.payload.knownVersion) ||
         (msg.payload.knownVersion as number) < 1 || msg.payload.view !== undefined)) throw new Error('invalid notModified snapshot')
    const callbacks = this.listeners.get(msg.type)
    callbacks?.forEach(callback => callback(msg.payload))
  }
  clear(): void { this.listeners.clear() }
}

/** Pure client command creation; caller supplies a secure unique ID and monotonic sequence. */
export function createCommand(input: Omit<CommandEnvelope, 'v'>): CommandEnvelope {
  const result: CommandEnvelope = { v: 'fair-doudizhu-command-v1', ...input }
  requireValidCommand(result)
  return freezeJSON(result)
}
