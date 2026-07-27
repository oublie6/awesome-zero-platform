export const SECURE_ENVELOPE_VERSION = 'secure-envelope-v1' as const
export const SECURE_ENVELOPE_SUITE = 'hpke-x25519-hkdf-sha256-aes-256-gcm' as const
export const SECURE_ENVELOPE_INFO_TEXT = 'awesome-zero-platform/secure-envelope/v1' as const
export const X25519_KEY_SIZE = 32
export const MAX_KEY_ID_LENGTH = 128
export const DEFAULT_MAX_PLAINTEXT = 64 * 1024

export function secureEnvelopeInfo(): Uint8Array {
  return new TextEncoder().encode(SECURE_ENVELOPE_INFO_TEXT)
}
