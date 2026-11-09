import type { GamePlatform } from './index.js'

/**
 * Platform-code exchange is an explicit backend adapter: never trust wx.login code
 * or platform user IDs as the application's authenticated account identity.
 */
export interface IdentityExchange {
  exchange(target: GamePlatform['target'], temporaryCode: string): Promise<{ accessToken: string }>
  revoke?(accessToken: string): Promise<void>
}

export class GameSession {
  private accessToken: string | null = null
  private generation = 0
  private disposed = false
  constructor(private readonly platform: GamePlatform, private readonly identity: IdentityExchange) {}
  get authenticated(): boolean { return !!this.accessToken && !this.disposed }
  get token(): string {
    if (!this.accessToken || this.disposed) throw new Error('not authenticated')
    return this.accessToken
  }
  /** Does not interpret local storage as a verified identity. Authentication must be confirmed by the server. */
  async restore(validate: (accessToken: string) => Promise<boolean>): Promise<boolean> {
    this.assertActive()
    const generation = ++this.generation
    const stored = await this.platform.vault.load()
    if (!stored) return false
    let valid = false
    try { valid = await validate(stored) } catch { valid = false }
    if (this.disposed || generation !== this.generation) return false
    if (!valid) { await this.platform.vault.clear(); return false }
    this.accessToken = stored
    return true
  }
  async login(): Promise<void> {
    return this.loginWith(async () => {
      const code = await this.platform.auth.acquireCode()
      return this.identity.exchange(this.platform.target, code)
    })
  }
  /** Credential and platform-code login share the same epoch, token and
   * logout semantics. Credential material is never passed to the vault. */
  async loginWith(exchange: () => Promise<{accessToken: string}>): Promise<void> {
    this.assertActive()
    const generation = ++this.generation
    const result = await exchange()
    if (this.disposed || generation !== this.generation) return
    if (!result || typeof result.accessToken !== 'string' || !result.accessToken.trim()) {
      throw new Error('identity exchange returned no access token')
    }
    await this.platform.vault.save(result.accessToken)
    if (this.disposed || generation !== this.generation) {
      await this.platform.vault.clear()
      return
    }
    this.accessToken = result.accessToken
  }
  async logout(): Promise<void> {
    this.generation++
    const token = this.accessToken
    this.accessToken = null
    try { if (token) await this.identity.revoke?.(token) }
    finally { await this.platform.vault.clear() }
  }
  dispose(): void {
    this.generation++
    this.disposed = true
    this.accessToken = null
  }
  private assertActive(): void { if (this.disposed) throw new Error('session disposed') }
}
