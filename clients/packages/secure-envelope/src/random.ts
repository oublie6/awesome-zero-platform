import { SecureRandomUnavailableError } from './errors.js'
import type { AsyncRandomSource } from './types.js'

type IntegerArray =
  | Int8Array
  | Uint8Array
  | Uint8ClampedArray
  | Int16Array
  | Uint16Array
  | Int32Array
  | Uint32Array

type CryptoLike = {
  getRandomValues<T extends IntegerArray>(array: T): T
}

type Runtime = {
  crypto?: Partial<CryptoLike>
}

let runtimeLock: Promise<void> = Promise.resolve()

export async function withSecureRandomRuntime<T>(
  source: AsyncRandomSource | undefined,
  operation: () => Promise<T>,
  poolLength = 4096,
): Promise<T> {
  const runtime = globalThis as unknown as Runtime
  if (typeof runtime.crypto?.getRandomValues === 'function') {
    return operation()
  }
  if (!source) {
    throw new SecureRandomUnavailableError(
      'crypto.getRandomValues is unavailable and no asynchronous secure random source was provided',
    )
  }
  if (!Number.isSafeInteger(poolLength) || poolLength < 64 || poolLength > 65_536) {
    throw new RangeError('poolLength must be an integer between 64 and 65536')
  }

  const previousLock = runtimeLock
  let release!: () => void
  runtimeLock = new Promise<void>((resolve) => {
    release = resolve
  })
  await previousLock

  let pool: Uint8Array
  try {
    pool = await source.bytes(poolLength)
  } catch (error) {
    release()
    throw new SecureRandomUnavailableError('secure random source failed', { cause: error })
  }
  if (!(pool instanceof Uint8Array) || pool.length !== poolLength) {
    if (pool instanceof Uint8Array) pool.fill(0)
    release()
    throw new SecureRandomUnavailableError('secure random source returned an unexpected byte length')
  }

  let offset = 0
  const cryptoLike: CryptoLike = {
    getRandomValues<T extends IntegerArray>(array: T): T {
      const bytes = new Uint8Array(array.buffer, array.byteOffset, array.byteLength)
      if (bytes.byteLength > 65_536 || offset + bytes.byteLength > pool.byteLength) {
        throw new SecureRandomUnavailableError('prefetched secure random pool is exhausted')
      }
      bytes.set(pool.subarray(offset, offset + bytes.byteLength))
      offset += bytes.byteLength
      return array
    },
  }

  const hadOwnCrypto = Object.prototype.hasOwnProperty.call(runtime, 'crypto')
  const previousCrypto = runtime.crypto
  let installed = false
  try {
    Object.defineProperty(runtime, 'crypto', {
      configurable: true,
      enumerable: true,
      writable: true,
      value: cryptoLike,
    })
    installed = true
    return await operation()
  } finally {
    pool.fill(0)
    if (installed) {
      if (hadOwnCrypto) {
        Object.defineProperty(runtime, 'crypto', {
          configurable: true,
          enumerable: true,
          writable: true,
          value: previousCrypto,
        })
      } else {
        delete runtime.crypto
      }
    }
    release()
  }
}
