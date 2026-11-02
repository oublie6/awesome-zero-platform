import type { WireCommand, WireCommandResult } from '@awesome-zero-platform/doudizhu-client/wire'
import { secureRoomCommandIdentity } from '@awesome-zero-platform/doudizhu-client/room'
import type { SecureRandomSource } from '@awesome-zero-platform/game-client-core'
import { GameClientShell } from './GameClientShell.ts'

/**
 * Explicit player actions against current authoritative hand version.
 * Rules remain entirely server-side. Never apply optimistic card or turn changes.
 */
export class GameHandActions {
  private seq = new Map<string, number>()
  private inFlight = false
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

  private async send(type: 'hand.bid'|'hand.play'|'hand.pass', payload: Record<string, unknown>, phase: 'BIDDING'|'PLAYING'): Promise<WireCommandResult> {
    if (this.inFlight) throw new Error('hand command already in flight; outcome may be unknown')
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
      return await this.shell.submitCommand(command)
    } finally {
      this.inFlight = false
    }
  }
}
