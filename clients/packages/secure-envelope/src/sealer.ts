import * as HPKE from 'hpke'
import {
  AEAD_AES_256_GCM,
  KDF_HKDF_SHA256,
  KEM_DHKEM_X25519_HKDF_SHA256,
} from '@panva/hpke-noble'

import {
  DEFAULT_MAX_PLAINTEXT,
  MAX_KEY_ID_LENGTH,
  SECURE_ENVELOPE_SUITE,
  SECURE_ENVELOPE_VERSION,
  secureEnvelopeInfo,
  X25519_KEY_SIZE,
} from './constants.js'
import { encodeBase64Url } from './base64url.js'
import { SecureEnvelopeError } from './errors.js'
import { withSecureRandomRuntime } from './random.js'
import type { AsyncRandomSource, SealRequest, SecureEnvelope } from './types.js'

const suite = new HPKE.CipherSuite(
  KEM_DHKEM_X25519_HKDF_SHA256,
  KDF_HKDF_SHA256,
  AEAD_AES_256_GCM,
)

export interface SecureEnvelopeSealerOptions {
  randomSource?: AsyncRandomSource
  maxPlaintextLength?: number
  randomPoolLength?: number
}

export class SecureEnvelopeSealer {
  readonly #randomSource: AsyncRandomSource | undefined
  readonly #maxPlaintextLength: number
  readonly #randomPoolLength: number

  constructor(options: SecureEnvelopeSealerOptions = {}) {
    this.#randomSource = options.randomSource
    this.#maxPlaintextLength = options.maxPlaintextLength ?? DEFAULT_MAX_PLAINTEXT
    this.#randomPoolLength = options.randomPoolLength ?? 4096
    if (!Number.isSafeInteger(this.#maxPlaintextLength) || this.#maxPlaintextLength <= 0) {
      throw new RangeError('maxPlaintextLength must be a positive integer')
    }
    if (
      !Number.isSafeInteger(this.#randomPoolLength) ||
      this.#randomPoolLength < 64 ||
      this.#randomPoolLength > 65_536
    ) {
      throw new RangeError('randomPoolLength must be an integer between 64 and 65536')
    }
  }

  async seal(request: SealRequest): Promise<SecureEnvelope> {
    validateRequest(request, this.#maxPlaintextLength)

    const keyId = request.recipient.keyId
    const publicKeyBytes = request.recipient.publicKey.slice()
    const plaintext = request.plaintext.slice()
    const aad = request.aad?.slice()

    try {
      return await withSecureRandomRuntime(
        this.#randomSource,
        async () => {
          try {
            const publicKey = await suite.DeserializePublicKey(publicKeyBytes)
            const info = secureEnvelopeInfo()
            const options = aad === undefined ? { info } : { aad, info }
            const sealed = await suite.Seal(publicKey, plaintext, options)
            return {
              version: SECURE_ENVELOPE_VERSION,
              keyId,
              suite: SECURE_ENVELOPE_SUITE,
              encapsulatedKey: encodeBase64Url(sealed.encapsulatedSecret),
              ciphertext: encodeBase64Url(sealed.ciphertext),
            }
          } catch (error) {
            throw new SecureEnvelopeError('secure envelope sealing failed', { cause: error })
          }
        },
        this.#randomPoolLength,
      )
    } finally {
      publicKeyBytes.fill(0)
      plaintext.fill(0)
      aad?.fill(0)
    }
  }
}

function validateRequest(request: SealRequest, maxPlaintextLength: number): void {
  const { keyId, publicKey } = request.recipient
  if (
    keyId.length === 0 ||
    keyId.length > MAX_KEY_ID_LENGTH ||
    keyId.trim() !== keyId ||
    /[^\x21-\x7e]/u.test(keyId)
  ) {
    throw new TypeError('recipient.keyId is invalid')
  }
  if (!(publicKey instanceof Uint8Array) || publicKey.length !== X25519_KEY_SIZE) {
    throw new TypeError('recipient.publicKey must be a 32-byte Uint8Array')
  }
  if (!(request.plaintext instanceof Uint8Array) || request.plaintext.length > maxPlaintextLength) {
    throw new RangeError('plaintext exceeds the configured limit')
  }
  if (request.aad !== undefined && !(request.aad instanceof Uint8Array)) {
    throw new TypeError('aad must be a Uint8Array')
  }
}
