import type { GamePlatform } from '@awesome-zero-platform/game-client-core'
import { GameSession, type IdentityExchange } from '@awesome-zero-platform/game-client-core/session'
import { RoomCoordinator, secureRoomCommandIdentity } from '@awesome-zero-platform/doudizhu-client/room'
import { DoudizhuHTTPClient, type RoomRead } from '@awesome-zero-platform/doudizhu-client/api'
import { GameClientShell } from './GameClientShell.ts'
import { GameHandActions } from './GameHandActions.ts'

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
  private disposed = false
  constructor(
    private readonly platform: GamePlatform,
    identity: IdentityExchange,
    private readonly socketEndpoint: string,
    onNotice: (message: string) => void
  ) {
    this.session = new GameSession(platform, identity)
    this.shell = new GameClientShell(onNotice)
    this.actions = new GameHandActions(this.shell, platform.random)
    this.api = new DoudizhuHTTPClient(platform.http, () => this.session.token)
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
    const view = await this.roomInfo(roomId)
    if (!view.member) throw new Error('room membership not confirmed; outcome still unknown')
    this.rooms.restoreTrustedRoom(roomId, view.version, view.lastClientSequence)
    return view
  }
  async createRoom(): Promise<Readonly<{roomId:string;version:number}>> {
    if (this.disposed) throw new Error('online controller disposed')
    const roomId = 'room-' + await secureRoomCommandIdentity(this.platform.random).newRequestId()
    return this.rooms.create(roomId)
  }
  async logout(): Promise<void> {
    this.shell.dispose()
    await this.session.logout()
  }
  dispose(): void {
    if (this.disposed) return
    this.disposed = true
    this.session.dispose()
    this.shell.dispose()
  }
}
