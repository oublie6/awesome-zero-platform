import type { GamePlatform, SessionVault } from './index.js'
import { OperationScope } from './operation.js'

/**
 * Platform-code exchange is an explicit backend adapter: never trust wx.login code
 * or platform user IDs as the application's authenticated account identity.
 */
export interface IdentityExchange {
  exchange(target: GamePlatform['target'], temporaryCode: string): Promise<{ accessToken: string }>
  revoke?(accessToken: string): Promise<void>
}

// One mutation queue per logical vault, not per controller. The weak key
// releases the coordinator with its backing store. Async adapters sharing a
// store must expose the same scope or reuse the same SessionVault instance.
type VaultOrder={tail:Promise<void>;loginEpoch:number;pendingLogin:OperationScope|null}
const orders=new WeakMap<object,Map<string,VaultOrder>>()
function orderFor(vault:SessionVault):VaultOrder {
  const scope=vault.coordinationScope
  const storage=scope?.storage??vault,key=scope?.key??''
  let keys=orders.get(storage)
  if(!keys){keys=new Map();orders.set(storage,keys)}
  let order=keys.get(key)
  if(!order){order={tail:Promise.resolve(),loginEpoch:0,pendingLogin:null};keys.set(key,order)}
  return order
}
/** An identity operation owns exactly this credential and epoch.
 * Superseded operations return null; recheck ownership after caller awaits. */
export interface SessionGrant {
  readonly token:string
  isCurrent():boolean
}
export class GameSession {
  private accessToken: string | null = null
  private generation = 0
  private disposed = false
  private ownedToken:string|null=null
  private activeGrant:SessionGrant|null=null
  private attemptOwner:()=>boolean=()=>false
  private pendingLogin:OperationScope|null=null
  private readonly order:VaultOrder
  /** Stop unfinished authentication without claiming or revoking a vault token.
   * Invalidate local state before invoking cancellation callbacks. */
  cancelAttempt():void{
    const previous=this.pendingLogin
    ++this.generation
    this.activeGrant=null
    this.attemptOwner=()=>false
    previous?.abandon()
  }
  private mutateVault<T>(operation:()=>Promise<T>):Promise<T>{
    const work=this.order.tail.then(operation)
    this.order.tail=work.then(()=>{},()=>{})
    return work
  }
  private async clearOwnedToken(token:string):Promise<void>{
    const vault=this.platform.vault
    if(vault.clearIf)await vault.clearIf(token)
    else if(await vault.load()===token)await vault.clear()
  }
  constructor(private readonly platform: GamePlatform, private readonly identity: IdentityExchange) {
    this.order=orderFor(platform.vault)
  }
  /** Business identity is the current grant, not mere credential presence.
   * Cleanup separately owns the captured token even after supersession. */
  get authenticated(): boolean { return this.activeGrant?.isCurrent()??false }
  get token(): string {
    const grant=this.activeGrant
    if (!grant?.isCurrent()) throw new Error('not authenticated')
    return grant.token
  }
  /** Capture the specific identity attempt for a pre-authentication binding
   * continuation. This is not an authenticated business grant. */
  captureAttempt():()=>boolean { return this.attemptOwner }
  /** Does not interpret local storage as a verified identity. Authentication must be confirmed by the server. */
  async restore(validate: (accessToken: string) => Promise<boolean>): Promise<boolean> {
    const grant=await this.restoreOwned(validate)
    return grant?.isCurrent()??false
  }
  async restoreOwned(validate: (accessToken: string) => Promise<boolean>): Promise<SessionGrant|null> {
    this.assertActive()
    const previous=this.pendingLogin
    const generation = ++this.generation
    this.accessToken=null
    this.activeGrant=null
    this.ownedToken=null
    const loginEpoch=this.order.loginEpoch
    const current=()=>!this.disposed && generation===this.generation && loginEpoch===this.order.loginEpoch &&
      this.order.pendingLogin===null
    this.attemptOwner=current
    previous?.abandon()
    // A restore skipped because a newer login is in flight cannot become a
    // fresh provider login when that newer login finishes.
    if(!current()){
      if(generation===this.generation)this.attemptOwner=()=>false
      return null
    }
    await this.order.tail
    if(!current())return null
    const stored = await this.platform.vault.load()
    if (!stored || !current()) return null
    // Capture ownership at restore, never by reloading during logout.
    this.ownedToken=stored
    let valid = false
    try { valid = await validate(stored) } catch { valid = false }
    if (!current()) return null
    if (!valid) {
      await this.mutateVault(async()=>{
        if(current())
          await this.clearOwnedToken(stored)
      })
      if(generation===this.generation)this.ownedToken=null
      return null
    }
    // Another controller or the parent Web page may have replaced the
    // shared token while the HTTP validation was outstanding.
    const saved=await this.platform.vault.load()
    if(!current() || saved!==stored){
      if(generation===this.generation){
        this.ownedToken=null
        this.attemptOwner=()=>false
      }
      return null
    }
    this.accessToken = stored
    return this.activeGrant=this.grant(stored,generation,loginEpoch)
  }
  private grant(token:string,generation:number,loginEpoch:number):SessionGrant {
    return Object.freeze({token,isCurrent:()=>!this.disposed && generation===this.generation &&
      loginEpoch===this.order.loginEpoch && this.accessToken===token})
  }
  async login(): Promise<SessionGrant|null> {
    return this.loginWith(async (current,scope) => {
      const code = await this.platform.auth.acquireCode(undefined,handler=>scope.onCancel(handler))
      // Do not exchange a platform code after logout or account replacement.
      if (!current()) return {accessToken: ''}
      return this.identity.exchange(this.platform.target, code)
    })
  }
  /** Credential and platform-code login share the same epoch, token and
   * logout semantics. Credential material is never passed to the vault. */
  async loginWith(exchange: (current:()=>boolean,scope:OperationScope) => Promise<{accessToken: string}>): Promise<SessionGrant|null> {
    this.assertActive()
    const previous=this.pendingLogin,sharedPrevious=this.order.pendingLogin
    const generation = ++this.generation
    const loginEpoch=++this.order.loginEpoch
    const scope=new OperationScope(()=>{
      if(this.pendingLogin===scope)this.pendingLogin=null
      if(this.order.pendingLogin===scope)this.order.pendingLogin=null
    })
    this.pendingLogin=scope
    this.order.pendingLogin=scope
    this.activeGrant=null
    const current=()=>!this.disposed && generation===this.generation && loginEpoch===this.order.loginEpoch
    // Keep attempt ownership after a normal binding-required response, but
    // never after explicit cancellation or replacement.
    this.attemptOwner=current
    // Publish the new owner before cancelling the old one: callbacks may
    // start another login on this or a different session sharing the vault.
    previous?.abandon()
    if(sharedPrevious!==previous)sharedPrevious?.abandon()
    try{
      if(!current() || !scope.active)return null
      let result:{accessToken:string}
      try{result=await scope.wait(()=>exchange(()=>scope.active && current(),scope),30_000)}catch(error){if(!scope.active || !current())return null;throw error}
      if (!current()) return null
      if (!result || typeof result.accessToken !== 'string' || !result.accessToken.trim()) {
        throw new Error('identity exchange returned no access token')
      }
      const grant=await this.mutateVault(async()=>{
        if(!current())return null
        await this.platform.vault.save(result.accessToken)
        if(!current()){
          await this.clearOwnedToken(result.accessToken)
          return null
        }
        this.ownedToken=result.accessToken
        this.accessToken=result.accessToken
        return this.activeGrant=this.grant(result.accessToken,generation,loginEpoch)
      })
      scope.finish()
      return grant?.isCurrent()?grant:null
    }finally{
      scope.finish()
    }
  }
  private releaseToken():string|null {
    const token=this.accessToken || this.ownedToken
    this.accessToken=null
    this.ownedToken=null
    this.cancelAttempt()
    return token
  }
  /** Provider relink/unlink already revoked the session on Go. Erase only
   * this controller's local credential; never claim another vault value. */
  async discardLocal():Promise<void>{
    const token=this.releaseToken()
    if(token)await this.mutateVault(()=>this.clearOwnedToken(token))
  }
  async logout(): Promise<void> {
    const token=this.releaseToken()
    // An untouched controller owns no token. Never claim the current vault
    // value on logout; it may belong to a newer controller.
    if(!token)return
    try{
      await this.mutateVault(()=>this.clearOwnedToken(token))
    }finally{await this.identity.revoke?.(token)}
  }
  dispose(): void {
    this.disposed = true
    this.accessToken = null
    this.ownedToken = null
    this.cancelAttempt()
  }
  private assertActive(): void { if (this.disposed) throw new Error('session disposed') }
}
