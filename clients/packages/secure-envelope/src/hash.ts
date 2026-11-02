import { sha256 } from '@noble/hashes/sha2.js'

/** Portable deterministic SHA-256 for non-secret fairness commitments. */
export function sha256Digest(value: Uint8Array): Uint8Array {
  if (!(value instanceof Uint8Array)) throw new TypeError('SHA-256 input must be bytes')
  return sha256(value)
}
