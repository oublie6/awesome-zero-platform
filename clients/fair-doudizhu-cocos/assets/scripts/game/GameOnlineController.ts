import type { GamePlatform } from '@awesome-zero-platform/game-client-core'
import { GameSession, type IdentityExchange } from '@awesome-zero-platform/game-client-core/session'
import { RoomCoordinator, secureRoomCommandIdentity } from '@awesome-zero-platform/doudizhu-client/room'
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
