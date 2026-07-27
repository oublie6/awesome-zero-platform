import type {
  SECURE_ENVELOPE_SUITE,
  SECURE_ENVELOPE_VERSION,
} from './constants.js'

export interface RecipientPublicKey {
  keyId: string
  publicKey: Uint8Array
}

export interface SecureEnvelope {
  version: typeof SECURE_ENVELOPE_VERSION
  keyId: string
  suite: typeof SECURE_ENVELOPE_SUITE
  encapsulatedKey: string
  ciphertext: string
}

export interface SealRequest {
  recipient: RecipientPublicKey
  plaintext: Uint8Array
  aad?: Uint8Array
}

export interface AsyncRandomSource {
  bytes(length: number): Promise<Uint8Array>
}
