import type { AppLifecycle, GamePlatform, PlatformAuth, SessionVault, SharePort, Target, Unsubscribe } from './index.js'
import { MiniHttp, MiniRandom, MiniSocketTransport, type MiniSDK } from './minigame.js'

export interface MiniPlatformSDK extends MiniSDK {
  login(args: { success: (result: {code?: string}) => void; fail: (error: unknown) => void }): void
  getStorageSync?(key: string): unknown
  setStorageSync?(key: string, value: string): void
  removeStorageSync?(key: string): void
  onShow?(handler: () => void): void
  offShow?(handler: () => void): void
  onHide?(handler: () => void): void
  offHide?(handler: () => void): void
  shareAppMessage?(args: { title: string; query: string }): void
}
class MiniAuth implements PlatformAuth {
  constructor(private readonly sdk: MiniPlatformSDK) {}
  acquireCode(): Promise<string> {
    return new Promise((resolve, reject) => this.sdk.login({
      success: result => result.code ? resolve(result.code) : reject(new Error('platform login yielded no code')),
      fail: reject
    }))
  }
}
class MiniVault implements SessionVault {
  constructor(private readonly sdk: MiniPlatformSDK, private readonly key: string) {}
  async load(): Promise<string | null> {
    const raw = this.sdk.getStorageSync?.(this.key)
    return typeof raw === 'string' && raw ? raw : null
  }
  async save(token: string): Promise<void> {
    if (!token) throw new Error('empty session token')
    if (!this.sdk.setStorageSync) throw new Error('secure session storage unsupported')
    this.sdk.setStorageSync(this.key, token)
  }
  async clear(): Promise<void> {
    if (!this.sdk.removeStorageSync) throw new Error('session clear unsupported')
    this.sdk.removeStorageSync(this.key)
  }
}
class MiniLifecycle implements AppLifecycle {
  constructor(private readonly sdk: MiniPlatformSDK) {}
  onForeground(handler: () => void): Unsubscribe {
    if (!this.sdk.onShow || !this.sdk.offShow) throw new Error('foreground hooks unsupported')
    this.sdk.onShow(handler)
    return () => this.sdk.offShow?.(handler)
  }
  onBackground(handler: () => void): Unsubscribe {
    if (!this.sdk.onHide || !this.sdk.offHide) throw new Error('background hooks unsupported')
    this.sdk.onHide(handler)
    return () => this.sdk.offHide?.(handler)
  }
}
class MiniShare implements SharePort {
  constructor(private readonly sdk: MiniPlatformSDK) {}
  async invite(roomCode: string): Promise<boolean> {
    if (!this.sdk.shareAppMessage || !/^[A-Za-z0-9_-]{1,64}$/.test(roomCode)) return false
    this.sdk.shareAppMessage({title: '公平斗地主 · 好友房', query: 'room=' + encodeURIComponent(roomCode)})
    return true
  }
}
/** Composition root: platform globals are supplied by the caller, never touched by shared modules. */
export function createMiniPlatform(target: Exclude<Target, 'web'>, sdk: MiniPlatformSDK, apiBase: string): GamePlatform {
  if (!apiBase.startsWith('https://')) throw new Error('mini-game API requires HTTPS')
  return {
    target, auth: new MiniAuth(sdk), http: new MiniHttp(sdk, apiBase),
    socket: new MiniSocketTransport(sdk),
    vault: new MiniVault(sdk, 'fair-doudizhu-' + target + '-session'),
    random: new MiniRandom(sdk), lifecycle: new MiniLifecycle(sdk),
    share: new MiniShare(sdk)
  }
}
