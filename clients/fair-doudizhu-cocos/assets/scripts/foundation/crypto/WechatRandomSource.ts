declare const wx: unknown

import type { AsyncRandomSource } from '@awesome-zero-platform/secure-envelope'

type WechatRandomValuesResult = { randomValues: ArrayBuffer }
type WechatAPI = {
  getRandomValues(options: {
    length: number
    success(result: WechatRandomValuesResult): void
    fail(error: unknown): void
  }): void
}

export class WechatRandomSource implements AsyncRandomSource {
  constructor(private readonly api: WechatAPI = wx as unknown as WechatAPI) {}

  bytes(length: number): Promise<Uint8Array> {
    if (!Number.isSafeInteger(length) || length <= 0 || length > 1_048_576) {
      return Promise.reject(new RangeError('length must be between 1 and 1048576'))
    }
    return new Promise<Uint8Array>((resolve, reject) => {
      this.api.getRandomValues({
        length,
        success: (result) => resolve(new Uint8Array(result.randomValues)),
        fail: reject,
      })
    })
  }
}
