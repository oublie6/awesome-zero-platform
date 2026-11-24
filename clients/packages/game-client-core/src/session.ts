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
  // Serialize local vault mutations, never network exchanges or revocation.
  // A late save/clear must not overwrite a newer login in this session.
  private vaultTail:Promise<void>=Promise.resolve()
  private mutateVault<T>(operation:()=>Promise<T>):Promise<T>{
    const work=this.vaultTail.then(operation)
    this.vaultTail=work.then(()=>{},()=>{})
    return work
  }
  private async clearOwnedToken(token:string):Promise<void>{
    const vault=this.platform.vault
    if(vault.clearIf)await vault.clearIf(token)
    else if(await vault.load()===token)await vault.clear()
  }
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
    this.accessToken=null
    await this.vaultTail
    if(this.disposed || generation!==this.generation)return false
    const stored = await this.platform.vault.load()
    if (!stored || this.disposed || generation!==this.generation) return false
    let valid = false
    try { valid = await validate(stored) } catch { valid = false }
    if (this.disposed || generation !== this.generation) return false
    if (!valid) {
      await this.mutateVault(async()=>{
        if(!this.disposed && generation===this.generation)await this.clearOwnedToken(stored)
      })
      return false
    }
    // Another controller or the parent Web page may have replaced the
    // shared token while the HTTP validation was outstanding.
    const current=await this.platform.vault.load()
    if(this.disposed || generation!==this.generation || current!==stored)return false
    this.accessToken = stored
    return true
  }
  async login(): Promise<void> {
    const expected = this.generation + 1
    return this.loginWith(async () => {
      const code = await this.platform.auth.acquireCode()
      // Do not exchange a platform code after logout or account replacement.
      if (this.disposed || this.generation !== expected) return {accessToken: ''}
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
    await this.mutateVault(async()=>{
      if(this.disposed || generation!==this.generation)return
      await this.platform.vault.save(result.accessToken)
      if(this.disposed || generation!==this.generation){
        await this.clearOwnedToken(result.accessToken)
        return
      }
      this.accessToken=result.accessToken
    })
  }
  async logout(): Promise<void> {
    this.generation++
    const token = this.accessToken
    this.accessToken = null
    // Capture the old storage value at logout, not after remote revoke.
    // Queue local cleanup immediately, before a later login can save.
    const stored=token?Promise.resolve(token):this.platform.vault.load()
    try{
      await this.mutateVault(async()=>{
        const old=await stored
        if(old)await this.clearOwnedToken(old)
      })
    }finally{if(token)await this.identity.revoke?.(token)}
  }
  dispose(): void {
    this.generation++
    this.disposed = true
    this.accessToken = null
  }
  private assertActive(): void { if (this.disposed) throw new Error('session disposed') }
}
