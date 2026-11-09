import type { GamePlatform } from '@awesome-zero-platform/game-client-core'
import type { IdentityExchange } from '@awesome-zero-platform/game-client-core/session'

/** Web passwords are handled by /auth/login in GameOnlineController.
 * Platform-code exchange is deliberately unavailable for the browser. */
export function createWebIdentityExchange(platform:GamePlatform): IdentityExchange {
  if(platform.target!=='web')throw new Error('web auth requires a web platform')
  return {
    async exchange():Promise<{accessToken:string}> {
      throw new Error('web login must use the Go password exchange endpoint')
    },
    async revoke(token:string):Promise<void>{
      if(!token || /[\r\n]/.test(token))throw new Error('invalid access token')
      await platform.http.request('/auth/logout',{method:'POST',accessToken:token,body:{},timeoutMs:5000})
    }
  }
}
export async function validateWebAccess(platform:GamePlatform,token:string):Promise<boolean>{
  if(platform.target!=='web' || !token || /[\r\n]/.test(token))return false
  try {
    const raw=await platform.http.request<unknown>('/auth/session',{accessToken:token,timeoutMs:5000})
    const envelope=raw as {code?:unknown;data?:unknown}|null
    if(!envelope || envelope.code!=='OK' || !envelope.data || typeof envelope.data!=='object')return false
    const account=(envelope.data as {account?:unknown}).account as {id?:unknown}|null
    return !!account && typeof account.id==='string' && !!account.id
  }catch{return false}
}
