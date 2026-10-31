import { type MiniPlatformSDK } from '@awesome-zero-platform/game-client-core/platform'
import type { AsyncRandomSource } from '@awesome-zero-platform/secure-envelope'
import { MiniRandom } from '@awesome-zero-platform/game-client-core/minigame'

/** Explicit SDK injection; reuse the entropy adapter, never a second HPKE implementation. */
export class WechatRandomSource implements AsyncRandomSource {
  private readonly random: MiniRandom
  constructor(api: Pick<MiniPlatformSDK, 'getUserCryptoManager'>) {
    this.random = new MiniRandom({getRandomValues: args => {
      const manager = api.getUserCryptoManager?.()
      if (!manager?.getRandomValues) throw new Error('WeChat secure random unsupported')
      manager.getRandomValues(args)
    }})
  }
  bytes(length: number): Promise<Uint8Array> { return this.random.bytes(length) }
}
