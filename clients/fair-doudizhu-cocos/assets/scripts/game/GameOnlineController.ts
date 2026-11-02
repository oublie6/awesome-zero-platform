import type { GamePlatform } from '@awesome-zero-platform/game-client-core'
import { GameSession, type IdentityExchange } from '@awesome-zero-platform/game-client-core/session'
import { RoomCoordinator, secureRoomCommandIdentity } from '@awesome-zero-platform/doudizhu-client/room'
import { DoudizhuHTTPClient, type RoomRead } from '@awesome-zero-platform/doudizhu-client/api'
import { GameClientShell } from './GameClientShell.ts'
import { GameHandActions } from './GameHandActions.ts'
import { GameFairnessActions } from './GameFairnessActions.ts'

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
  readonly fairness: GameFairnessActions
  private disposed = false
  constructor(
    private readonly platform: GamePlatform,
    identity: IdentityExchange,
    private readonly socketEndpoint: string,
    onNotice: (message: string) => void,
    trustedManifestRoots: Readonly<Record<string,string>> = {}
  ) {
    this.session = new GameSession(platform, identity)
    this.shell = new GameClientShell(onNotice)
    this.actions = new GameHandActions(this.shell, platform.random)
    this.api = new DoudizhuHTTPClient(platform.http, () => this.session.token)
    this.fairness = new GameFairnessActions(this.api,platform,trustedManifestRoots)
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
    await this.shell.connect(this.platform.socket, this.socketEndpoint, this.session.token)
    const target = roomId || this.rooms.state?.roomId || this.rooms.pendingRoomId
    if (!target) return null
    const room = await this.lookupOrRetryPendingCreate(target)
    if (!room.member) throw new Error('room membership not confirmed after reconnect')
    this.rooms.restoreTrustedRoom(room.roomId, room.version, room.lastClientSequence)
    if (room.activeHandId) {
      await this.shell.syncHand(room.activeHandId)
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
      this.rooms.restoreTrustedRoom(roomId, view.version, view.lastClientSequence)
      return this.rooms.state!
    }
    return this.rooms.join(roomId, view.version, view.lastClientSequence)
  }
  /** Resolve an ambiguous command only from an authenticated, authoritative room read. */
  async reconcileRoom(roomId: string): Promise<RoomRead> {
    const view = await this.lookupOrRetryPendingCreate(roomId)
    if (!view.member) throw new Error('room membership not confirmed; outcome still unknown')
    this.rooms.restoreTrustedRoom(roomId, view.version, view.lastClientSequence)
    return view
  }
  get pendingRoomId(): string | null { return this.rooms.pendingRoomId }

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

  async createRoom(): Promise<Readonly<{roomId:string;version:number}>> {
    if (this.disposed) throw new Error('online controller disposed')
    const roomId = 'room-' + await secureRoomCommandIdentity(this.platform.random).newRequestId()
    return this.rooms.create(roomId)
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
