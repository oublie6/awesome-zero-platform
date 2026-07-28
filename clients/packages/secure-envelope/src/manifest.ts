import * as ed25519 from '@noble/ed25519'
import { sha256, sha512 } from '@noble/hashes/sha2.js'
import { decodeBase64Url, encodeBase64Url } from './base64url.js'
import { SECURE_ENVELOPE_SUITE } from './constants.js'

ed25519.hashes.sha512 = sha512

export const REVEAL_KEY_MANIFEST_V1 = 'reveal-key-manifest-v1'

export type RevealKeyStatus = 'active' | 'retiring' | 'retired' | 'revoked'

export interface RevealKeyManifest {
  manifestVersion: number
  protocolVersion: string
  keyId: string
  suite: string
  publicKey: string
  publicKeySha256: string
  notBefore: string
  expiresAt: string
  status: RevealKeyStatus
  retiringAt?: string
  retireAfter?: string
  signatureKeyId: string
  signature: string
}

export interface ManifestVersionStore {
  getCurrentHighWater(): number
  setCurrentHighWater(version: number): void
  getKeyHighWater(keyId: string): number
  setKeyHighWater(keyId: string, version: number): void
}

export class InMemoryManifestVersionStore implements ManifestVersionStore {
  private current = 0
  private readonly keys = new Map<string, number>()

  getCurrentHighWater(): number { return this.current }
  setCurrentHighWater(version: number): void { this.current = Math.max(this.current, version) }
  getKeyHighWater(keyId: string): number { return this.keys.get(keyId) ?? 0 }
  setKeyHighWater(keyId: string, version: number): void {
    this.keys.set(keyId, Math.max(this.getKeyHighWater(keyId), version))
  }
}

export type VerifyManifestOptions = {
  roots: Readonly<Record<string, string>>
  versions: ManifestVersionStore
  now?: Date
  mode: 'current' | 'pinned'
  expectedKeyId?: string
  expectedPublicKeySha256?: string
  boundAt?: Date
}

export type VerifiedRevealKeyManifest = RevealKeyManifest & {
  publicKeyBytes: Uint8Array
  publicKeySha256Bytes: Uint8Array
}

const allowedKeys = new Set([
  'manifestVersion', 'protocolVersion', 'keyId', 'suite', 'publicKey',
  'publicKeySha256', 'notBefore', 'expiresAt', 'status', 'retiringAt',
  'retireAfter', 'signatureKeyId', 'signature',
])

export function canonicalRevealKeyManifest(manifest: RevealKeyManifest): Uint8Array {
  const canonical = {
    manifestVersion: manifest.manifestVersion,
    protocolVersion: manifest.protocolVersion,
    keyId: manifest.keyId,
    suite: manifest.suite,
    publicKey: manifest.publicKey,
    publicKeySha256: manifest.publicKeySha256,
    notBefore: manifest.notBefore,
    expiresAt: manifest.expiresAt,
    status: manifest.status,
    ...(manifest.retiringAt === undefined ? {} : { retiringAt: manifest.retiringAt }),
    ...(manifest.retireAfter === undefined ? {} : { retireAfter: manifest.retireAfter }),
    signatureKeyId: manifest.signatureKeyId,
  }
  return new TextEncoder().encode(JSON.stringify(canonical))
}

export function verifyRevealKeyManifest(
  value: unknown,
  options: VerifyManifestOptions,
): VerifiedRevealKeyManifest {
  const manifest = parseManifest(value)
  const publicKey = decodeSized(manifest.publicKey, 32, 'public key')
  const publicKeyHash = decodeSized(manifest.publicKeySha256, 32, 'public key hash')
  const computedHash = sha256(publicKey)
  if (!equalBytes(publicKeyHash, computedHash)) throw new TypeError('reveal key hash mismatch')

  const rootValue = options.roots[manifest.signatureKeyId]
  if (rootValue === undefined) throw new TypeError('unknown reveal manifest signing key')
  const root = decodeSized(rootValue, 32, 'manifest root public key')
  const signature = decodeSized(manifest.signature, 64, 'manifest signature')
  if (!ed25519.verify(signature, canonicalRevealKeyManifest(manifest), root)) {
    throw new TypeError('invalid reveal key manifest signature')
  }

  const now = options.now ?? new Date()
  const nowMillis = checkedDate(now, 'current time')
  const notBefore = checkedTimestamp(manifest.notBefore, 'notBefore')
  const expiresAt = checkedTimestamp(manifest.expiresAt, 'expiresAt')
  if (expiresAt <= notBefore || nowMillis < notBefore || nowMillis >= expiresAt) {
    throw new TypeError('reveal key manifest is outside its validity window')
  }

  if (options.mode === 'current') {
    if (manifest.status !== 'active' || manifest.retiringAt !== undefined || manifest.retireAfter !== undefined) {
      throw new TypeError('current reveal key manifest is not active')
    }
    if (manifest.manifestVersion < options.versions.getCurrentHighWater()) {
      throw new TypeError('current reveal key manifest rollback')
    }
  } else {
    if (options.expectedKeyId === undefined || options.expectedPublicKeySha256 === undefined || options.boundAt === undefined) {
      throw new TypeError('pinned reveal key expectations are required')
    }
    if (manifest.keyId !== options.expectedKeyId || manifest.publicKeySha256 !== options.expectedPublicKeySha256) {
      throw new TypeError('pinned reveal key context mismatch')
    }
    if (manifest.status !== 'active' && manifest.status !== 'retiring') {
      throw new TypeError('pinned reveal key is unavailable')
    }
    if (manifest.status === 'retiring') {
      if (manifest.retiringAt === undefined || manifest.retireAfter === undefined) {
        throw new TypeError('retiring reveal key lifecycle is incomplete')
      }
      const retiringAt = checkedTimestamp(manifest.retiringAt, 'retiringAt')
      const retireAfter = checkedTimestamp(manifest.retireAfter, 'retireAfter')
      if (retireAfter < retiringAt || retireAfter > expiresAt || checkedDate(options.boundAt, 'boundAt') >= retiringAt || nowMillis >= retireAfter) {
        throw new TypeError('retiring reveal key grace window is invalid')
      }
    }
  }

  if (manifest.manifestVersion < options.versions.getKeyHighWater(manifest.keyId)) {
    throw new TypeError('reveal key manifest version rollback')
  }
  options.versions.setKeyHighWater(manifest.keyId, manifest.manifestVersion)
  if (options.mode === 'current') options.versions.setCurrentHighWater(manifest.manifestVersion)

  return Object.assign({}, manifest, {
    publicKeyBytes: publicKey,
    publicKeySha256Bytes: publicKeyHash,
  })
}

function parseManifest(value: unknown): RevealKeyManifest {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) throw new TypeError('invalid reveal key manifest')
  const record = value as Record<string, unknown>
  for (const key of Object.keys(record)) {
    if (!allowedKeys.has(key)) throw new TypeError(`unknown reveal key manifest field: ${key}`)
  }
  const manifestVersion = record.manifestVersion
  if (!Number.isSafeInteger(manifestVersion) || (manifestVersion as number) <= 0) throw new TypeError('invalid manifestVersion')
  const status = requiredString(record.status, 'status')
  if (!['active', 'retiring', 'retired', 'revoked'].includes(status)) throw new TypeError('invalid reveal key status')
  const manifest: RevealKeyManifest = {
    manifestVersion: manifestVersion as number,
    protocolVersion: requiredString(record.protocolVersion, 'protocolVersion'),
    keyId: identifier(record.keyId, 'keyId'),
    suite: requiredString(record.suite, 'suite'),
    publicKey: requiredString(record.publicKey, 'publicKey'),
    publicKeySha256: requiredString(record.publicKeySha256, 'publicKeySha256'),
    notBefore: requiredString(record.notBefore, 'notBefore'),
    expiresAt: requiredString(record.expiresAt, 'expiresAt'),
    status: status as RevealKeyStatus,
    signatureKeyId: identifier(record.signatureKeyId, 'signatureKeyId'),
    signature: requiredString(record.signature, 'signature'),
  }
  if (record.retiringAt !== undefined) manifest.retiringAt = requiredString(record.retiringAt, 'retiringAt')
  if (record.retireAfter !== undefined) manifest.retireAfter = requiredString(record.retireAfter, 'retireAfter')
  if (manifest.protocolVersion !== REVEAL_KEY_MANIFEST_V1 || manifest.suite !== SECURE_ENVELOPE_SUITE) {
    throw new TypeError('unsupported reveal key manifest protocol or suite')
  }
  return manifest
}

function identifier(value: unknown, name: string): string {
  const result = requiredString(value, name)
  if (result.length > 128 || result.trim() !== result || /[^\x21-\x7e]/u.test(result)) throw new TypeError(`invalid ${name}`)
  return result
}

function requiredString(value: unknown, name: string): string {
  if (typeof value !== 'string' || value.length === 0) throw new TypeError(`invalid ${name}`)
  return value
}

function decodeSized(value: string, size: number, name: string): Uint8Array {
  let decoded: Uint8Array
  try { decoded = decodeBase64Url(value) } catch { throw new TypeError(`invalid ${name}`) }
  if (decoded.length !== size || encodeBase64Url(decoded) !== value) throw new TypeError(`invalid ${name}`)
  return decoded
}

function checkedTimestamp(value: string, name: string): number {
  if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$/u.test(value)) throw new TypeError(`invalid ${name}`)
  const parsed = Date.parse(value)
  if (!Number.isFinite(parsed)) throw new TypeError(`invalid ${name}`)
  return parsed
}

function checkedDate(value: Date, name: string): number {
  const milliseconds = value.getTime()
  if (!Number.isFinite(milliseconds)) throw new TypeError(`invalid ${name}`)
  return milliseconds
}

function equalBytes(left: Uint8Array, right: Uint8Array): boolean {
  if (left.length !== right.length) return false
  let difference = 0
  for (let index = 0; index < left.length; index += 1) difference |= (left[index] ?? 0) ^ (right[index] ?? 0)
  return difference === 0
}
