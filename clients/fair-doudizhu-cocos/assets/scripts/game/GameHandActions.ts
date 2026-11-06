import type { WireCommand, WireCommandResult } from '@awesome-zero-platform/doudizhu-client/wire'
import { secureRoomCommandIdentity } from '@awesome-zero-platform/doudizhu-client/room'
import type { SecureRandomSource } from '@awesome-zero-platform/game-client-core'
import { GameClientShell, CommandRejectedError } from './GameClientShell.ts'

/**
 * Explicit player actions against current authoritative hand version.
 * Rules remain entirely server-side. Never apply optimistic card or turn changes.
 */
export class GameHandActions {
  private seq = new Map<string, number>()
  private inFlight = false
  private handTransition = false
  private pending: Omit<WireCommand, 'v'> | null = null
  private readonly identities
  constructor(private readonly shell: GameClientShell, random: SecureRandomSource) {
    this.identities = secureRoomCommandIdentity(random)
  }
  bid(score: number): Promise<WireCommandResult> {
    if (!Number.isSafeInteger(score) || score < 0 || score > 3) return Promise.reject(new Error('invalid bid score'))
    return this.send('hand.bid', {score}, 'BIDDING')
  }
  play(cards: readonly string[]): Promise<WireCommandResult> {
    const code = /^(?:[CDHS](?:[3-9]|10|J|Q|K|A|2)|XJ|YJ)$/
    if (!Array.isArray(cards) || cards.length < 1 || cards.length > 20 ||
        !cards.every(card => typeof card === 'string' && code.test(card)) ||
        new Set(cards).size !== cards.length) return Promise.reject(new Error('invalid card selection'))
    return this.send('hand.play', {cards:[...cards]}, 'PLAYING')
  }
  pass(): Promise<WireCommandResult> { return this.send('hand.pass', {}, 'PLAYING') }
  get needsReconciliation(): boolean { return this.pending !== null }

  /** Prevent live Bid/Play/Pass from racing a new-hand room command.
   * Unknown commands must be reconciled explicitly, never thrown away. */
  beginHandTransition(): () => void {
    if (this.inFlight || this.handTransition)
      throw new Error('live hand action in progress; cannot start next hand')
    if (this.pending)
      throw new Error('live hand outcome unknown; reconcile the original request before starting next hand')
    this.handTransition = true
    return () => { this.handTransition = false }
  }

  /** Retry only the exact originally transmitted request ID, actor hand,
   * sequence, payload and version. Reconnect may already have advanced the
   * local snapshot; the authoritative Go server decides whether this was
   * committed and returns the persisted result without reapplying it. */
  async retryUnknown(): Promise<WireCommandResult> {
    if (!this.pending || this.inFlight) throw new Error('no pending hand command to retry')
    this.inFlight = true
    try {
      const result = await this.shell.submitCommand(this.pending, true)
      this.pending = null
      return result
    } catch (err) {
      if (err instanceof CommandRejectedError) this.pending = null
      throw err
    } finally {
      this.inFlight = false
    }
  }
  /** No local command can prove its own unknown outcome. After a trusted
   * server snapshot advances beyond its expected version, the user may
   * explicitly discard the old request (never resubmit it with a new ID). */
  discardUnknownAfterSync(): void {
    if (this.inFlight || !this.pending) throw new Error('no pending hand command')
    const snapshot = this.shell.hand.snapshot
    if (!snapshot || snapshot.handId !== this.pending.aggregateId ||
      !this.shell.hand.canSubmit || snapshot.version <= this.pending.expectedVersion)
      throw new Error('server has not confirmed a newer hand version')
    this.pending = null
  }

  private async send(type: 'hand.bid'|'hand.play'|'hand.pass', payload: Record<string, unknown>, phase: 'BIDDING'|'PLAYING'): Promise<WireCommandResult> {
    if (this.inFlight || this.handTransition) throw new Error('hand command already in flight; outcome may be unknown')
    if (this.pending) throw new Error('previous hand command outcome unknown; retry exact request or reconcile')
    const snapshot = this.shell.hand.snapshot
    if (!snapshot || snapshot.phase !== phase || !this.shell.hand.canSubmit) throw new Error('hand is not ready for this action')
    this.inFlight = true
    try {
      const seq = (this.seq.get(snapshot.handId) ?? 0) + 1
      if (!Number.isSafeInteger(seq)) throw new Error('hand command sequence exhausted')
      this.seq.set(snapshot.handId, seq)
      const requestId = await this.identities.newRequestId()
      // Reject if a new authoritative version arrived during entropy acquisition.
      if (this.shell.hand.snapshot !== snapshot || !this.shell.hand.canSubmit)
        throw new Error('hand changed before command submission')
      const command: Omit<WireCommand, 'v'> = {
        type, requestId, aggregateId: snapshot.handId, clientSeq: seq,
        expectedVersion: snapshot.version, payload
      }
      this.pending = command
      try {
        const response = await this.shell.submitCommand(command)
        this.pending = null
        return response
      } catch (err) {
        // Transport timeout/disconnect and ambiguous server failures are
        // UNKNOWN, not FAILED. Never allocate a fresh ID until reconciled.
        if (err instanceof CommandRejectedError) this.pending = null
        throw err
      }
    } finally {
      this.inFlight = false
    }
  }
}
