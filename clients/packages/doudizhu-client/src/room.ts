import type { WireCommand, WireCommandResult, WireCommandType } from './wire.js'

type RoomCommandType = Extract<WireCommandType, 'room.create' | 'room.join' | 'room.leave' | 'room.ready' | 'room.hand.start'>
export interface RoomCommandPort {
  command(input: Omit<WireCommand, 'v'>): Promise<WireCommandResult>
}
export interface CommandIdentity {
  /** Fresh crypto-secure request ID; same ID must be used when reconciling an unknown outcome. */
  newRequestId(): string | Promise<string>
}
export interface RoomState {
  roomId: string
  version: number
}
export class DurableRoomRejected extends Error {
  constructor(readonly code: string, readonly currentVersion?: number) {
    super('room command rejected: ' + code)
  }
}
/**
 * Coordinates real Go durable room commands. Never assumes a failed transport
 * means the command was rolled back, and never derives state from sent intent.
 * One coordinator instance represents one actor's room command sequence.
 */
export class RoomCoordinator {
  private room: RoomState | null = null
  private lastSequence = new Map<string, number>()
  private inFlight = false
  private uncertainOutcome = false
  private pendingCommand: Omit<WireCommand, 'v'> | null = null
  constructor(private readonly api: RoomCommandPort, private readonly ids: CommandIdentity) {}
  get state(): Readonly<RoomState> | null { return this.room ? Object.freeze({...this.room}) : null }
  get needsReconciliation(): boolean { return this.uncertainOutcome }
  /** Keep the committed-or-unknown room locator after a lost create reply. */
  get pendingRoomId(): string | null { return this.pendingCommand?.aggregateId ?? null }
  get pendingCommandType(): string | null { return this.pendingCommand?.type ?? null }

  /** Exact-ID resend is safe even if the prior request committed. Go owns the
   * durable dedup result; a new room command ID is never synthesized here. */
  async retryUnknown(): Promise<RoomState> {
    if (!this.uncertainOutcome || !this.pendingCommand || this.inFlight)
      throw new Error('no exact pending room request to retry')
    this.inFlight = true
    try {
      const cmd = this.pendingCommand
      const response = await this.api.command(cmd)
      return this.acceptResult(cmd, response)
    } finally { this.inFlight = false }
  }

  /** Adopt only a version from a trusted, authorized server-side room query. */
  restoreTrustedRoom(roomId: string, version: number, lastClientSequence = 0): void {
    if (!validID(roomId) || !Number.isSafeInteger(version) || version < 1 ||
        !Number.isSafeInteger(lastClientSequence) || lastClientSequence < 0) throw new Error('invalid trusted room')
    if (this.inFlight) throw new Error('cannot replace room while command outstanding')
    this.room = Object.freeze({roomId, version})
    this.lastSequence.set(roomId, Math.max(this.lastSequence.get(roomId) ?? 0, lastClientSequence))
    this.uncertainOutcome = false
    this.pendingCommand = null
  }

  create(roomId: string): Promise<RoomState> { return this.perform('room.create', roomId, 0, {}) }
  join(roomId: string, expectedVersion: number, lastClientSequence = 0): Promise<RoomState> {
    if (!Number.isSafeInteger(lastClientSequence) || lastClientSequence < 0)
      return Promise.reject(new Error('invalid client sequence'))
    this.lastSequence.set(roomId, Math.max(this.lastSequence.get(roomId) ?? 0, lastClientSequence))
    return this.perform('room.join', roomId, expectedVersion, {})
  }
  ready(ready: boolean): Promise<RoomState> {
    if (typeof ready !== 'boolean') return Promise.reject(new Error('invalid ready flag'))
    return this.withRoom('room.ready', {ready})
  }
  startHand(handId: string): Promise<RoomState> {
    if (!validID(handId)) return Promise.reject(new Error('invalid hand id'))
    return this.withRoom('room.hand.start', {handId})
  }
  leave(): Promise<RoomState> { return this.withRoom('room.leave', {}) }

  private withRoom(type: RoomCommandType, payload: Record<string, unknown>): Promise<RoomState> {
    if (!this.room) return Promise.reject(new Error('room not confirmed by server'))
    return this.perform(type, this.room.roomId, this.room.version, payload)
  }

  private async perform(type: RoomCommandType, roomId: string, expectedVersion: number, payload: Record<string, unknown>): Promise<RoomState> {
    if (!validID(roomId) || !Number.isSafeInteger(expectedVersion) || expectedVersion < (type === 'room.create' ? 0 : 1) ||
      (type === 'room.create' && expectedVersion !== 0)) throw new Error('invalid room version or ID')
    if (this.inFlight) throw new Error('room command outstanding; outcome may be unknown')
    if (this.uncertainOutcome) throw new Error('room outcome unknown; reconcile with an authoritative server room query before continuing')
    if (type === 'room.create' && this.room) throw new Error('already managing a room')
    if (type !== 'room.create' && this.room && this.room.roomId !== roomId) throw new Error('room identity mismatch')
    this.inFlight = true
    try {
      const seq = (this.lastSequence.get(roomId) ?? 0) + 1
      if (!Number.isSafeInteger(seq)) throw new Error('room sequence exhausted')
      // Persist sequence allocation even when the outcome is unknown; never recycle it.
      this.lastSequence.set(roomId, seq)
      const requestId = await this.ids.newRequestId()
      if (!validID(requestId)) throw new Error('secure command ID unavailable')
      const command: Omit<WireCommand, 'v'> = {
        requestId, type, aggregateId: roomId,
        clientSeq: seq, expectedVersion, payload
      }
      this.pendingCommand = command
      let result: WireCommandResult
      try {
        result = await this.api.command(command)
      } catch (error) {
        this.uncertainOutcome = true
        throw error
      }
      return this.acceptResult(command, result)
    } finally { this.inFlight = false }
  }
  private acceptResult(command: Omit<WireCommand, 'v'>, result: WireCommandResult): RoomState {
    const {roomId, type, expectedVersion, requestId} = {
      roomId:command.aggregateId,type:command.type,expectedVersion:command.expectedVersion,
      requestId:command.requestId
    }
    const d = result?.durable as Record<string, unknown> | undefined
    if (result?.v !== 'doudizhu-api-command-result-v1' || result.requestId !== requestId ||
      result.type !== type || !d || d.v !== 'fair-doudizhu-command-result-v1' ||
      d.aggregateId !== roomId || d.commandId !== requestId ||
      !Number.isSafeInteger(d.aggregateVersion) || (d.aggregateVersion as number) < 0 ||
      typeof d.accepted !== 'boolean') {
      this.uncertainOutcome = true
      throw new Error('invalid authoritative room result; outcome unknown')
    }
    if (!d.accepted) {
      const failure = d.failure
      if (!record(failure) || typeof failure.code !== 'string' || !failure.code.trim()) {
        this.uncertainOutcome = true
        throw new Error('malformed durable rejection; outcome unknown')
      }
      this.uncertainOutcome = false
      this.pendingCommand = null
      throw new DurableRoomRejected(failure.code,
        Number.isSafeInteger(failure.currentVersion) ? failure.currentVersion as number : undefined)
    }
    if ((d.aggregateVersion as number) <= expectedVersion) {
      this.uncertainOutcome = true
      throw new Error('non-monotonic room version; outcome unknown')
    }
    this.room = type === 'room.leave' ? null : Object.freeze({roomId,version:d.aggregateVersion as number})
    this.uncertainOutcome = false
    this.pendingCommand = null
    return Object.freeze({roomId,version:d.aggregateVersion as number})
  }
}
function validID(value: unknown): value is string {
  return typeof value === 'string' && !!value.trim() && value === value.trim() && value.length <= 128
}
function record(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object' && !Array.isArray(value)
}

/** Creates unpredictable, platform-neutral request IDs using only injected secure entropy. */
export function secureRoomCommandIdentity(random: { bytes(length: number): Promise<Uint8Array> }): CommandIdentity {
  return {
    async newRequestId(): Promise<string> {
      const bytes = await random.bytes(16)
      if (!(bytes instanceof Uint8Array) || bytes.length !== 16)
        throw new Error('secure request ID entropy unavailable')
      return 'cmd-' + Array.from(bytes, b => b.toString(16).padStart(2, '0')).join('')
    }
  }
}
