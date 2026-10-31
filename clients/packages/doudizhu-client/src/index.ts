/**
 * Fair Doudizhu client-side state coordination.
 * Only server-confirmed versions enter this store. The server owns rules and cards.
 */
export type HandPhase = 'FAIRNESS_COMMITTING' | 'FAIRNESS_REVEALING' | 'DEALING' |
  'BIDDING' | 'PLAYING' | 'SETTLING' | 'COMPLETED' | 'ABORTED' | 'EXPIRED'
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
  get canSubmit(): boolean { return this.statusValue === 'connected' && this.current !== null }
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
    if (!snapshot.handId || !Number.isSafeInteger(snapshot.version) || snapshot.version < 0) {
      throw new Error('invalid authoritative hand snapshot')
    }
    if (this.current && snapshot.handId === this.current.handId && snapshot.version < this.current.version) {
      return false
    }
    if (this.current && snapshot.handId !== this.current.handId) {
      throw new Error('hand identity changed without explicit reset')
    }
    this.current = snapshot
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
export function requireValidCommand(command: CommandEnvelope): void {
  if (command.v !== 'fair-doudizhu-command-v1' || !command.name.endsWith('.v1') ||
      !command.commandId || !command.aggregateId ||
      !Number.isSafeInteger(command.clientSeq) || command.clientSeq < 1 ||
      !Number.isSafeInteger(command.expectedVersion) || command.expectedVersion < 0 ||
      !Number.isFinite(Date.parse(command.issuedAt)) || !Number.isFinite(Date.parse(command.expiresAt)) ||
      Date.parse(command.expiresAt) <= Date.parse(command.issuedAt)) {
    throw new Error('invalid command envelope')
  }
}
