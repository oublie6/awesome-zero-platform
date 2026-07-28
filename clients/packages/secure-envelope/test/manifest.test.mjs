import assert from 'node:assert/strict'
import test from 'node:test'
import * as ed25519 from '@noble/ed25519'
import { sha256, sha512 } from '@noble/hashes/sha2.js'
import {
  REVEAL_KEY_MANIFEST_V1,
  SECURE_ENVELOPE_SUITE,
  InMemoryManifestVersionStore,
  canonicalRevealKeyManifest,
  encodeBase64Url,
  verifyRevealKeyManifest,
} from '../dist/index.js'

ed25519.hashes.sha512 = sha512
const seed = Uint8Array.from({ length: 32 }, (_, index) => index + 1)
const publicRoot = ed25519.getPublicKey(seed)
const publicKey = Uint8Array.from({ length: 32 }, (_, index) => index + 40)
const publicKeyHash = sha256(publicKey)

function signedManifest(overrides = {}) {
  const manifest = {
    manifestVersion: 7,
    protocolVersion: REVEAL_KEY_MANIFEST_V1,
    keyId: 'reveal-2026-07',
    suite: SECURE_ENVELOPE_SUITE,
    publicKey: encodeBase64Url(publicKey),
    publicKeySha256: encodeBase64Url(publicKeyHash),
    notBefore: '2026-07-27T00:00:00Z',
    expiresAt: '2026-08-28T00:00:00Z',
    status: 'active',
    signatureKeyId: 'root-2026',
    signature: '',
    ...overrides,
  }
  manifest.signature = encodeBase64Url(ed25519.sign(canonicalRevealKeyManifest(manifest), seed))
  return manifest
}

const roots = { 'root-2026': encodeBase64Url(publicRoot) }
const now = new Date('2026-07-28T04:00:00Z')

test('verifies a signed current reveal key manifest', () => {
  const versions = new InMemoryManifestVersionStore()
  const verified = verifyRevealKeyManifest(signedManifest(), { roots, versions, mode: 'current', now })
  assert.equal(verified.keyId, 'reveal-2026-07')
  assert.deepEqual(verified.publicKeyBytes, publicKey)
  assert.equal(versions.getCurrentHighWater(), 7)
})

test('rejects replacement, signature, suite, time and unknown-root attacks', () => {
  const cases = [
    { publicKey: encodeBase64Url(new Uint8Array(32)) },
    { keyId: 'attacker-key' },
    { suite: 'attacker-suite' },
    { signatureKeyId: 'unknown-root' },
    { expiresAt: '2026-07-28T03:00:00Z' },
  ]
  for (const override of cases) {
    const manifest = signedManifest(override)
    if ('keyId' in override) manifest.keyId = 'different-after-signing'
    assert.throws(() => verifyRevealKeyManifest(manifest, { roots, versions: new InMemoryManifestVersionStore(), mode: 'current', now }))
  }
})

test('rejects current-manifest rollback but permits pinned historical lookup', () => {
  const versions = new InMemoryManifestVersionStore()
  verifyRevealKeyManifest(signedManifest({ manifestVersion: 8 }), { roots, versions, mode: 'current', now })
  assert.throws(() => verifyRevealKeyManifest(signedManifest({ manifestVersion: 7 }), { roots, versions, mode: 'current', now }), /rollback/u)

  const retiring = signedManifest({
    manifestVersion: 4,
    keyId: 'reveal-old',
    status: 'retiring',
    retiringAt: '2026-07-28T03:00:00Z',
    retireAfter: '2026-07-28T06:00:00Z',
  })
  const verified = verifyRevealKeyManifest(retiring, {
    roots,
    versions,
    mode: 'pinned',
    now,
    expectedKeyId: retiring.keyId,
    expectedPublicKeySha256: retiring.publicKeySha256,
    boundAt: new Date('2026-07-28T02:00:00Z'),
  })
  assert.equal(verified.status, 'retiring')
  assert.equal(versions.getCurrentHighWater(), 8)
})

test('rejects post-retirement binds, expired grace and pinned hash mismatch', () => {
  const retiring = signedManifest({
    manifestVersion: 4,
    keyId: 'reveal-old',
    status: 'retiring',
    retiringAt: '2026-07-28T03:00:00Z',
    retireAfter: '2026-07-28T06:00:00Z',
  })
  const base = {
    roots,
    versions: new InMemoryManifestVersionStore(),
    mode: 'pinned',
    expectedKeyId: retiring.keyId,
    expectedPublicKeySha256: retiring.publicKeySha256,
  }
  assert.throws(() => verifyRevealKeyManifest(retiring, { ...base, now, boundAt: new Date('2026-07-28T03:00:00Z') }))
  assert.throws(() => verifyRevealKeyManifest(retiring, { ...base, now: new Date('2026-07-28T06:00:00Z'), boundAt: new Date('2026-07-28T02:00:00Z') }))
  assert.throws(() => verifyRevealKeyManifest(retiring, { ...base, now, boundAt: new Date('2026-07-28T02:00:00Z'), expectedPublicKeySha256: encodeBase64Url(new Uint8Array(32)) }))
})
