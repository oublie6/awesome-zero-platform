import type { GamePlatform } from '@awesome-zero-platform/game-client-core'
import { GameSession, type IdentityExchange } from '@awesome-zero-platform/game-client-core/session'
import { RoomCoordinator, secureRoomCommandIdentity, type RoomState } from '@awesome-zero-platform/doudizhu-client/room'
import { DoudizhuHTTPClient, type RoomRead } from '@awesome-zero-platform/doudizhu-client/api'
import { GameClientShell } from './GameClientShell.ts'
import { GameHandActions } from './GameHandActions.ts'
import { DurableFairnessRejected, GameFairnessActions } from './GameFairnessActions.ts'

/**
 * Real session -> authenticated socket -> server-command composition.
 * Never accepts a raw platform login code as an account token or creates local fake rooms.
 */
export class GameOnlineController {
  readonly session: GameSession
  readonly shell: GameClientShell
  readonly rooms: RoomCoordinator
  readonly api: DoudizhuHTTPClient
  readonly actions: GameHandActions
  fairness: GameFairnessActions
  // Only an authenticated server result can advance this binding.
  private confirmedFairnessHand: string | null = null
  private startingHand: string | null = null
  private disposed = false
  private membershipGeneration = 0
  constructor(
    private readonly platform: GamePlatform,
    identity: IdentityExchange,
    private readonly socketEndpoint: string,
    onNotice: (message: string) => void,
    private readonly trustedManifestRoots: Readonly<Record<string,string>> = {}
  ) {
    this.session = new GameSession(platform, identity)
    this.shell = new GameClientShell(onNotice)
    this.actions = new GameHandActions(this.shell, platform.random)
    this.api = new DoudizhuHTTPClient(platform.http, () => this.session.token)
    this.fairness = new GameFairnessActions(this.api,platform,this.trustedManifestRoots)
    this.rooms = new RoomCoordinator(
      {command: input => this.shell.submitCommand(input)},
      secureRoomCommandIdentity(platform.random)
    )
  }
  async loginAndConnect(): Promise<void> {
    if (this.disposed) throw new Error('online controller disposed')
    await this.shell.initialize()
    await this.session.login()
    if (this.disposed) return
    await this.shell.connect(this.platform.socket, this.socketEndpoint, this.session.token)
  }
  /** Server validation is mandatory even for a previously saved access token. */
  async restoreAndConnect(validate: (token: string) => Promise<boolean>): Promise<boolean> {
    if (this.disposed) throw new Error('online controller disposed')
    await this.shell.initialize()
    if (!await this.session.restore(validate)) return false
    if (this.disposed) return false
    await this.shell.connect(this.platform.socket, this.socketEndpoint, this.session.token)
    return true
  }
  /** Re-establish authenticated transport, then trust only server-owned
   * membership/version and private hand state. Never synthesize a successful
   * result for commands whose response was lost on the old connection. */
  async reconnectAndResync(roomId?: string): Promise<RoomRead | null> {
    if (this.disposed || !this.session.authenticated) throw new Error('authenticated session required')
    const generation = this.membershipGeneration
    await this.shell.connect(this.platform.socket, this.socketEndpoint, this.session.token)
    const target = roomId || this.rooms.state?.roomId || this.rooms.pendingRoomId
    if (!target) return null
    const room = await this.lookupOrRetryPendingCreate(target)
    if (this.disposed || generation !== this.membershipGeneration) return null
    if (!room.member) throw new Error('room membership not confirmed after reconnect')
    this.confirmActiveFairnessHand(room.activeHandId || null)
    this.rooms.restoreTrustedRoom(room.roomId, room.version, room.lastClientSequence)
    this.shell.useAuthoritativeHand(room.activeHandId || room.lastHandId || null)
    if (room.activeHandId) {
      await this.shell.syncHand(room.activeHandId)
    } else if (room.lastHandId) {
      const evidence = await this.api.finalEvidence(room.lastHandId)
      if (this.disposed || generation !== this.membershipGeneration ||
          this.rooms.state?.roomId !== room.roomId) return null
      this.shell.acceptVerifiedFinalEvidence(evidence)
    }
    return room
  }
  async roomInfo(roomId: string): Promise<RoomRead> {
    if (this.disposed) throw new Error('online controller disposed')
    return this.api.room(roomId)
  }
  /** Secure join code is a locator; the Go server checks membership and version. */
  async joinRoom(roomId: string): Promise<Readonly<{roomId:string;version:number}>> {
    const view = await this.roomInfo(roomId)
    if (view.status === 'CLOSED' || view.status === 'HAND_ACTIVE') throw new Error('room not joinable')
    if (view.member) {
      this.confirmActiveFairnessHand(view.activeHandId || null)
      this.rooms.restoreTrustedRoom(roomId, view.version, view.lastClientSequence)
      this.membershipGeneration++
      this.shell.useAuthoritativeHand(view.activeHandId || view.lastHandId || null)
      return this.rooms.state!
    }
    const joined = await this.rooms.join(roomId, view.version, view.lastClientSequence)
    this.confirmActiveFairnessHand(null)
    this.membershipGeneration++
    this.shell.useAuthoritativeHand(null)
    return joined
  }
  /** Resolve an ambiguous command only from an authenticated, authoritative room read. */
  async reconcileRoom(roomId: string): Promise<RoomRead> {
    const view = await this.lookupOrRetryPendingCreate(roomId)
    if (!view.member) throw new Error('room membership not confirmed; outcome still unknown')
    this.rooms.restoreTrustedRoom(roomId, view.version, view.lastClientSequence)
    return view
  }
  get pendingRoomId(): string | null { return this.rooms.pendingRoomId }
  /** While the room.hand.start ACK is outstanding, defer HTTP-derived hand
   * rebinding. A read may see the new hand before the WSS command completes. */
  get handStartInProgress(): boolean { return this.startingHand !== null }
  /** Reuse the original command ID if the previous leave ACK was lost. */
  async leaveRoom(): Promise<void> {
    if (this.disposed || !this.session.authenticated) throw new Error('authenticated session required')
    if (this.actions.needsReconciliation || this.fairness.needsReconciliation)
      throw new Error('hand or fairness outcome unknown; reconcile before leaving')
    if (this.rooms.needsReconciliation) {
      if (this.rooms.pendingCommandType !== 'room.leave') throw new Error('room outcome unknown; reconcile before leaving')
      await this.rooms.retryUnknown()
    } else {
      if (!this.rooms.state) throw new Error('room not confirmed by server')
      await this.rooms.leave()
    }
    // The room.leave receipt is definitive. Invalidate in-flight preparation
    // and erase the old hand's secret, but never discard an unknown outcome.
    this.resetFairnessAfterConfirmedTransition()
    this.confirmedFairnessHand = null
    this.membershipGeneration++
    this.shell.clearHand()
  }

  private async lookupOrRetryPendingCreate(roomId:string):Promise<RoomRead>{
    try {
      return await this.api.room(roomId)
    } catch(error) {
      // Only an authoritative NOT_FOUND for the generated locator permits
      // an exact replay of the original request. Other errors remain unknown.
      if(this.rooms.pendingRoomId!==roomId ||
        !(error instanceof Error) || error.message!=='HTTP 404') throw error
      await this.rooms.retryUnknown()
      return this.api.room(roomId)
    }
  }

  /** Reconcile the prior hand before submitting a durable new-hand command.
   * reserveHandTransition bars new fairness work during the async room send;
   * after the server ACK, local binding and cleanup cannot fail due to an
   * unobserved, newly-started fairness operation. */
  async startHand(handId: string): Promise<Readonly<RoomState>> {
    if (this.disposed || !this.session.authenticated)
      throw new Error('authenticated session required')
    // Guard live Bid/Play/Pass as well as fairness. Either kind of pending
    // command could make a post-ACK hand binding throw and desynchronize UI.
    const releaseLive = this.actions.beginHandTransition()
    try {
      const previous = this.fairness
      if (previous.needsReconciliation) {
        try {
          // Exact-ID retry can return an accepted receipt or a definitive
          // durable rejection. Either settles the old unknown outcome.
          await previous.retryUnknown()
        } catch (error) {
          if (!(error instanceof DurableFairnessRejected) || previous.needsReconciliation)
            throw error
        }
      }
      if (this.disposed || this.fairness !== previous)
        throw new Error('fairness session changed while reconciling; refresh room')
      const releaseFairness = previous.beginHandTransition()
      try {
        if (this.startingHand !== null)
          throw new Error('room hand start already in progress')
        this.startingHand = handId
        try {
          const room = await this.rooms.startHand(handId,'client-contributed-server-custodied-v1')
          // RoomCoordinator already accepted the server's durable ACK. No
          // await between receipt and hand adoption; both old action families
          // are still reserved so they cannot create fresh pending requests.
          if (!this.disposed) {
            // The accepted durable receipt is the transition barrier. Clear
            // the controller-owned guard synchronously, before binding H2;
            // no other JS task can interleave before the next synchronous step.
            this.startingHand = null
            this.confirmActiveFairnessHand(handId)
            this.shell.useAuthoritativeHand(handId)
            this.membershipGeneration++
          }
          return room
        } finally { this.startingHand = null }
      } finally { releaseFairness() }
    } finally { releaseLive() }
  }

  async createRoom(): Promise<Readonly<{roomId:string;version:number}>> {
    if (this.disposed) throw new Error('online controller disposed')
    const roomId = 'room-' + await secureRoomCommandIdentity(this.platform.random).newRequestId()
    const room = await this.rooms.create(roomId)
    this.confirmActiveFairnessHand(null)
    this.membershipGeneration++
    this.shell.useAuthoritativeHand(null)
    return room
  }
  /** Call only after the server has confirmed the active hand identity.
   * A completed/cancelled hand has no active ID even if lastHandId remains
   * available for an authorized historical evidence read. */
  confirmActiveFairnessHand(handId: string | null): void {
    if (this.disposed) throw new Error('online controller disposed')
    // Even an authenticated HTTP read of H2 may arrive BEFORE the room.hand.start
    // WSS ACK. Its freshness does not authorize replacing the reserved fairness
    // instance: the replacement would otherwise have no transition lock.
    // Only startHand() itself may adopt H2 after a validated durable receipt.
    if (this.startingHand !== null && handId !== this.confirmedFairnessHand)
      throw new Error('room hand start in progress; wait for authoritative result')
    if (this.confirmedFairnessHand === handId) return
    this.resetFairnessAfterConfirmedTransition()
    this.confirmedFairnessHand = handId
  }
  private resetFairnessAfterConfirmedTransition(): void {
    if (this.fairness.needsReconciliation)
      throw new Error('fairness result unknown; retry the original request before changing hand')
    const previous = this.fairness
    this.fairness = new GameFairnessActions(this.api, this.platform, this.trustedManifestRoots)
    previous.dispose()
  }

  async logout(): Promise<void> {
    this.fairness.dispose()
    this.shell.dispose()
    await this.session.logout()
  }
  dispose(): void {
    if (this.disposed) return
    this.disposed = true
    this.session.dispose()
    this.fairness.dispose()
    this.shell.dispose()
  }
}
