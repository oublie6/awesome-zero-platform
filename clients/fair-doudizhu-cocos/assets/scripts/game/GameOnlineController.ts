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
  // The latest explicit recovery choice owns all post-await room/hand writes.
  // Same-target concurrent recoveries share one operation only while its
  // controller selection, confirmed membership and room commands are current.
  private recoveryGeneration = 0
  private pendingRecovery: {
    key:string;token:number;membership:number;commands:number;
    promise:Promise<RoomRead|null>
  } | null = null
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
  /** Recovery is read-only until its final authoritative membership check.
   * Never adopt an earlier room response over a newer user selection. */
  reconnectAndResync(roomId?: string): Promise<RoomRead | null> {
    if (this.disposed || !this.session.authenticated)
      return Promise.reject(new Error('authenticated session required'))
    const target = roomId || this.rooms.state?.roomId || this.rooms.pendingRoomId || undefined
    if (this.rooms.needsReconciliation && this.rooms.pendingRoomId !== target)
      return Promise.reject(new Error('room outcome unknown for another room; retry original request before switching'))
    const key = target ?? '#discover'
    const membership = this.membershipGeneration
    const commands = this.rooms.commandGeneration
    const pending = this.pendingRecovery
    // A matching locator is not sufficient: join(B), startHand(), an
    // exact-ID retry, or a committed room command can revoke old recovery.
    // Never coalesce a new explicit choice onto that obsolete promise.
    if (pending?.key === key && pending.token === this.recoveryGeneration &&
        pending.membership === membership && pending.commands === commands)
      return pending.promise
    const token = ++this.recoveryGeneration
    const promise = this.performRecovery(target,token,membership,commands)
    this.pendingRecovery = {key,token,membership,commands,promise}
    void promise.finally(()=>{
      if(this.pendingRecovery?.promise===promise)this.pendingRecovery=null
    }).catch(()=>{})
    return promise
  }
  private async performRecovery(target:string|undefined,token:number,membership:number,commands:number):Promise<RoomRead|null>{
    const current=()=>!this.disposed && this.session.authenticated &&
      token===this.recoveryGeneration && membership===this.membershipGeneration &&
      commands===this.rooms.commandGeneration
    await this.shell.connect(this.platform.socket,this.socketEndpoint,this.session.token)
    if(!current())return null
    if(!target){
      const choices=await this.discoverRecoverableRooms()
      if(!current())return null
      const active=choices.filter(view=>view.status==='HAND_ACTIVE')
      if(active.length===1)target=active[0].roomId
      else if(active.length>1)throw new Error('multiple active rooms; choose one to recover')
      else if(choices.length===1)target=choices[0].roomId
      else if(choices.length>1)throw new Error('multiple recoverable rooms; choose one to recover')
      else return null
    }
    const initial=await this.lookupOrRetryPendingCreate(target)
    if(!current())return null
    if(!initial.member || initial.status==='CLOSED')
      throw new Error('room membership not confirmed after reconnect')
    // Establish whether a private view is available, without mutating
    // controller state before final membership verification.
    let phase:string|null=null
    let evidence:Awaited<ReturnType<DoudizhuHTTPClient['finalEvidence']>>|null=null
    if(initial.activeHandId){
      const fairness=await this.api.fairness(initial.activeHandId)
      if(!current())return null
      phase=fairness.phase
    }else if(initial.lastHandId){
      evidence=await this.api.finalEvidence(initial.lastHandId)
      if(!current())return null
    }
    // A delayed A/v1 result can predate a genuine CLOSED/v2 change.
    // An independent fresh read is required BEFORE any authoritative adopt.
    const fresh=await this.api.room(initial.roomId)
    if(!current())return null
    if(!fresh.member || fresh.status==='CLOSED'){
      this.forgetInvalidRecoveredRoom(initial.roomId)
      throw new Error('room membership revoked or room closed during recovery')
    }
    if(fresh.version<initial.version ||
       fresh.activeHandId!==initial.activeHandId ||
       fresh.lastHandId!==initial.lastHandId)
      throw new Error('room changed during recovery; retry authoritative lookup')
    this.adoptAuthorizedRoom(fresh)
    // No suspension between final membership check and authoritative adoption.
    if(fresh.activeHandId && (phase==='BIDDING'||phase==='PLAYING'||phase==='SETTLING')){
      await this.shell.syncHand(fresh.activeHandId)
      if(!current())return null
    }else if(evidence){
      this.shell.acceptVerifiedFinalEvidence(evidence)
    }
    return current()?fresh:null
  }
  /** Scene refresh must obey the same pending-command/fairness/hand barriers.
   * Do not adopt HTTP data if a command changed while the read was in flight. */
  adoptAuthorizedRoomRefresh(view: RoomRead, commandGeneration: number): void {
    if (this.disposed || !this.session.authenticated ||
        commandGeneration !== this.rooms.commandGeneration)
      throw new Error('room changed while refreshing; retry authoritative lookup')
    if (!view.member || view.status === 'CLOSED')
      throw new Error('room membership revoked or room closed during refresh')
    this.adoptAuthorizedRoom(view)
  }

  /** Check all mutating subsystems before adopting any new binding. The
   * reservations and adoption are synchronous: no intermediate HTTP awaits. */
  private adoptAuthorizedRoom(view: RoomRead): void {
    this.rooms.assertCanRestoreTrustedRoom(view.roomId)
    if (this.startingHand !== null && view.activeHandId !== this.confirmedFairnessHand)
      throw new Error('room hand start in progress; wait for authoritative result')
    const releaseHand = this.actions.beginHandTransition()
    try {
      const releaseFairness = this.fairness.beginHandTransition()
      try {
        this.rooms.restoreTrustedRoom(view.roomId, view.version, view.lastClientSequence)
        this.confirmActiveFairnessHand(view.activeHandId || null)
        this.shell.useAuthoritativeHand(view.activeHandId || view.lastHandId || null)
      } finally { releaseFairness() }
    } finally { releaseHand() }
  }

  /** A later authoritative CLOSED/nonmember read must revoke an earlier
   * adopted locator, without discarding unresolved durable commands. */
  forgetInvalidRecoveredRoom(roomId:string):void{
    if(this.disposed || this.rooms.state?.roomId!==roomId)return
    if(this.rooms.needsReconciliation || this.fairness.needsReconciliation ||
       this.actions.needsReconciliation || this.startingHand!==null)
      throw new Error('cannot discard room with an unresolved durable operation')
    this.confirmActiveFairnessHand(null)
    this.rooms.forgetTrustedRoom(roomId)
    this.shell.clearHand()
    this.membershipGeneration++
    this.recoveryGeneration++
  }
  async discoverRecoverableRooms():Promise<RoomRead[]> {
    if(this.disposed || !this.session.authenticated)
      throw new Error('authenticated session required for recovery discovery')
    return this.api.recoverableRooms()
  }
  async roomInfo(roomId: string): Promise<RoomRead> {
    if (this.disposed) throw new Error('online controller disposed')
    return this.api.room(roomId)
  }
  /** Secure join code is a locator; the Go server checks membership and version. */
  async joinRoom(roomId: string): Promise<Readonly<{roomId:string;version:number}>> {
    if (this.disposed || !this.session.authenticated)
      throw new Error('authenticated session required')
    if (this.rooms.needsReconciliation && this.rooms.pendingRoomId !== roomId)
      throw new Error('room outcome unknown for another room; retry original request before switching')
    const token = ++this.recoveryGeneration
    const membership = this.membershipGeneration
    const commands = this.rooms.commandGeneration
    const current = () => !this.disposed && this.session.authenticated &&
      token === this.recoveryGeneration && membership === this.membershipGeneration
    const readCurrent = () => current() && commands === this.rooms.commandGeneration
    const view = await this.roomInfo(roomId)
    if (!readCurrent()) throw new Error('room selection superseded during join')
    if (view.status === 'CLOSED') throw new Error('room not joinable')
    // Current members can resume an already active hand. This does NOT allow
    // outsiders into HAND_ACTIVE or bypass the server's membership check.
    if (view.member) {
      // Re-adopt the authenticated room on the existing socket. Joining a
      // known member must not open a second connection or treat HAND_ACTIVE
      // as a new admission; the Go room query already verified membership.
      // Resolve metadata before the final membership read; stale responses
      // must not mutate controller state after a newer room selection.
      let phase: string | null = null
      let evidence: Awaited<ReturnType<DoudizhuHTTPClient['finalEvidence']>> | null = null
      if (view.activeHandId) {
        const fairness = await this.api.fairness(view.activeHandId)
        if (!readCurrent()) throw new Error('room selection superseded during join')
        phase = fairness.phase
      } else if (view.lastHandId) {
        evidence = await this.api.finalEvidence(view.lastHandId)
        if (!readCurrent()) throw new Error('room selection superseded during join')
      }
      const fresh = await this.roomInfo(roomId)
      if (!readCurrent()) throw new Error('room selection superseded during join')
      if (!fresh.member || fresh.status === 'CLOSED') {
        this.forgetInvalidRecoveredRoom(roomId)
        throw new Error('room membership revoked or room closed during join')
      }
      if (fresh.version < view.version || fresh.activeHandId !== view.activeHandId ||
          fresh.lastHandId !== view.lastHandId)
        throw new Error('room changed during join; retry authoritative lookup')
      // No suspension between the final authorization read and adoption.
      this.adoptAuthorizedRoom(fresh)
      const generation = ++this.membershipGeneration
      if (fresh.activeHandId && (phase === 'BIDDING' || phase === 'PLAYING' ||
                                  phase === 'SETTLING')) {
        await this.shell.syncHand(fresh.activeHandId)
        if (this.disposed || token !== this.recoveryGeneration ||
            generation !== this.membershipGeneration ||
            this.rooms.state?.roomId !== roomId)
          throw new Error('room selection superseded during join')
      } else if (evidence) {
        this.shell.acceptVerifiedFinalEvidence(evidence)
      }
      return {roomId:fresh.roomId,version:fresh.version}
    }
    if (view.status === 'HAND_ACTIVE') throw new Error('room not joinable')
    const joined = await this.rooms.join(roomId, view.version, view.lastClientSequence)
    // Do not discard an accepted durable command if its UI choice is stale.
    if (!current()) throw new Error('room selection superseded after join receipt')
    this.confirmActiveFairnessHand(null)
    this.membershipGeneration++
    this.shell.useAuthoritativeHand(null)
    return joined
  }
  /** Settle only by the original durable request ID, never by a room read. */
  async retryUnknownRoom(): Promise<Readonly<RoomState>> {
    if (this.disposed || !this.session.authenticated)
      throw new Error('authenticated session required')
    if (!this.rooms.needsReconciliation)
      throw new Error('no pending room request to reconcile')
    const selection = ++this.recoveryGeneration
    if (!this.shell.commandTransportReady)
      await this.shell.connect(this.platform.socket, this.socketEndpoint, this.session.token)
    if (this.disposed || !this.session.authenticated || selection !== this.recoveryGeneration)
      throw new Error('room reconciliation superseded')
    return this.rooms.retryUnknown()
  }
  /** Recheck authoritative membership after an exact unknown receipt. */
  async reconcileRoom(roomId: string): Promise<RoomRead> {
    if (this.rooms.needsReconciliation) {
      if (this.rooms.pendingRoomId !== roomId)
        throw new Error('room outcome unknown for another room; retry original request before switching')
      await this.retryUnknownRoom()
    }
    const view = await this.reconnectAndResync(roomId)
    if (!view) throw new Error('room recovery superseded or no authorized room')
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
    ++this.recoveryGeneration
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
    const sourceRoom = this.rooms.state?.roomId
    if (!sourceRoom) throw new Error('room not confirmed by server')
    const token = ++this.recoveryGeneration
    const membership = this.membershipGeneration
    const commands = this.rooms.commandGeneration
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
      if (this.disposed || this.fairness !== previous ||
          token !== this.recoveryGeneration || membership !== this.membershipGeneration ||
          commands !== this.rooms.commandGeneration || this.rooms.state?.roomId !== sourceRoom)
        throw new Error('room selection superseded while reconciling fairness; refresh room')
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
    if (this.disposed || !this.session.authenticated)
      throw new Error('authenticated session required')
    if (this.rooms.state) throw new Error('already managing a room')
    // Reserve the caller's selection before awaiting CSPRNG. Otherwise an
    // older Create can send after a newer member Join/Recover has won.
    const token = ++this.recoveryGeneration
    const membership = this.membershipGeneration
    const commands = this.rooms.commandGeneration
    const roomId = 'room-' + await secureRoomCommandIdentity(this.platform.random).newRequestId()
    if (this.disposed || token !== this.recoveryGeneration ||
        membership !== this.membershipGeneration ||
        commands !== this.rooms.commandGeneration)
      throw new Error('room selection superseded during create')
    const room = await this.rooms.create(roomId)
    // The ACK is durable; preserve its RoomCoordinator result. Do not let
    // its stale controller cleanup clobber a newer accepted room binding.
    if (this.disposed || token !== this.recoveryGeneration ||
        membership !== this.membershipGeneration ||
        this.rooms.state?.roomId !== roomId)
      throw new Error('room selection superseded after create receipt')
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
    // Revoke every pending recovery/join before logout suspends for token
    // revocation. A delayed HTTP reply cannot adopt state after this point.
    this.disposed = true
    ++this.recoveryGeneration
    ++this.membershipGeneration
    this.fairness.dispose()
    this.shell.dispose()
    await this.session.logout()
  }
  dispose(): void {
    if (this.disposed) return
    this.disposed = true
    ++this.recoveryGeneration
    this.session.dispose()
    this.fairness.dispose()
    this.shell.dispose()
  }
}
