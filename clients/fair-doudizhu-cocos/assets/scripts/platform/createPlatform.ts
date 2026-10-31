import { createMiniPlatform, type MiniPlatformSDK } from '@awesome-zero-platform/game-client-core/platform'
import { WebHttp, WebLifecycle, WebRandom, WebSessionVault, WebShare, WebSocketTransport } from '@awesome-zero-platform/game-client-core/web'
import type { GamePlatform } from '@awesome-zero-platform/game-client-core'
import { detectRuntimeTarget } from './platform-target.ts'

/** Only the runtime composition root may read platform globals. */
export function createPlatform(config: {apiBase: string; sdk?: MiniPlatformSDK}): GamePlatform {
  const target = detectRuntimeTarget()
  if (target === 'wechat' || target === 'douyin') {
    if (!config.sdk) throw new Error('platform SDK must be explicitly injected')
    return createMiniPlatform(target, config.sdk, config.apiBase)
  }
  if (target === 'web') {
    return {
      target: 'web',
      auth: { acquireCode: async () => { throw new Error('Web login requires platform identity UI') } },
      http: new WebHttp(config.apiBase),
      socket: new WebSocketTransport(),
      vault: new WebSessionVault(),
      random: new WebRandom(),
      lifecycle: new WebLifecycle(),
      share: new WebShare()
    }
  }
  throw new Error('unsupported runtime')
}
