import type { GamePlatform } from '@awesome-zero-platform/game-client-core'
import type { IdentityExchange } from '@awesome-zero-platform/game-client-core/session'

function record(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object' && !Array.isArray(value)
}
function unwrap(value: unknown): Record<string, unknown> {
  if (!record(value) || value.code !== 'OK' || !record(value.data)) throw new Error('identity exchange rejected by Go')
  return value.data
}
/** The temporary wx.login code is never treated as a token or account ID. */
export function createWeChatIdentityExchange(platform: GamePlatform): IdentityExchange {
  return {
    async exchange(target, code): Promise<{accessToken:string}> {
      if (target !== 'wechat' || platform.target !== 'wechat' || typeof code !== 'string' ||
          code.length < 4 || code.length > 128 || code.trim() !== code) throw new Error('verified WeChat code required')
      const raw=await platform.http.request<unknown>('/auth/wechat/code',{
        method:'POST',body:{code},timeoutMs:12000
      })
      const data=unwrap(raw)
      if (typeof data.accessToken !== 'string' || !data.accessToken.trim() ||
          data.tokenType !== 'Bearer' || data.accessToken.includes('\n')) throw new Error('invalid authenticated token')
      return {accessToken:data.accessToken}
    },
    async revoke(token:string):Promise<void> {
      if (!token || /[\r\n]/.test(token)) throw new Error('invalid token')
      await platform.http.request<unknown>('/auth/logout',{method:'POST',accessToken:token,body:{},timeoutMs:5000})
    }
  }
}
export async function validateWeChatAccess(platform:GamePlatform,token:string):Promise<boolean> {
  if (!token || /[\r\n]/.test(token)) return false
  try {
    const raw=await platform.http.request<unknown>('/auth/session',{accessToken:token,timeoutMs:5000})
    const data=unwrap(raw)
    return record(data.account) && typeof data.account.id === 'string' && !!data.account.id
  } catch { return false }
}
