import assert from 'node:assert/strict'
import test from 'node:test'

import * as HPKE from 'hpke'
import {
  AEAD_AES_256_GCM,
  KDF_HKDF_SHA256,
  KEM_DHKEM_X25519_HKDF_SHA256,
} from '@panva/hpke-noble'

import {
  decodeBase64Url,
  secureEnvelopeInfo,
  SECURE_ENVELOPE_SUITE,
  SECURE_ENVELOPE_VERSION,
  SecureEnvelopeSealer,
} from '../dist/index.js'

const suite = new HPKE.CipherSuite(
  KEM_DHKEM_X25519_HKDF_SHA256,
  KDF_HKDF_SHA256,
  AEAD_AES_256_GCM,
)

async function fixture() {
  const keyPair = await suite.GenerateKeyPair(true)
  return {
    keyPair,
    publicKey: await suite.SerializePublicKey(keyPair.publicKey),
  }
}

test('seals and opens a versioned envelope', async () => {
  const { keyPair, publicKey } = await fixture()
  const plaintext = new TextEncoder().encode('fair reveal payload')
  const aad = new TextEncoder().encode('game=1;hand=2;seat=0')
  const sealer = new SecureEnvelopeSealer()
  const envelope = await sealer.seal({
    recipient: { keyId: 'reveal-key-1', publicKey },
    plaintext,
    aad,
  })

  assert.equal(envelope.version, SECURE_ENVELOPE_VERSION)
  assert.equal(envelope.suite, SECURE_ENVELOPE_SUITE)
  const opened = await suite.Open(
    keyPair.privateKey,
    decodeBase64Url(envelope.encapsulatedKey),
    decodeBase64Url(envelope.ciphertext),
    { aad, info: secureEnvelopeInfo() },
  )
  assert.deepEqual(opened, plaintext)
})

test('aad tampering fails authentication', async () => {
  const { keyPair, publicKey } = await fixture()
  const sealer = new SecureEnvelopeSealer()
  const envelope = await sealer.seal({
    recipient: { keyId: 'reveal-key-1', publicKey },
    plaintext: new Uint8Array([1, 2, 3]),
    aad: new Uint8Array([4, 5, 6]),
  })

  await assert.rejects(() =>
    suite.Open(
      keyPair.privateKey,
      decodeBase64Url(envelope.encapsulatedKey),
      decodeBase64Url(envelope.ciphertext),
      { aad: new Uint8Array([4, 5, 7]), info: secureEnvelopeInfo() },
    ),
  )
})

test('validates public key and plaintext limits', async () => {
  const sealer = new SecureEnvelopeSealer({ maxPlaintextLength: 2 })
  await assert.rejects(
    () =>
      sealer.seal({
        recipient: { keyId: 'key', publicKey: new Uint8Array(31) },
        plaintext: new Uint8Array([1]),
      }),
    /32-byte/u,
  )
  await assert.rejects(
    () =>
      sealer.seal({
        recipient: { keyId: 'key', publicKey: new Uint8Array(32) },
        plaintext: new Uint8Array([1, 2, 3]),
      }),
    /configured limit/u,
  )
})

test('base64url rejects non-canonical encodings', async () => {
  assert.throws(() => decodeBase64Url('A'), /invalid base64url/u)
  assert.throws(() => decodeBase64Url('AB'), /non-canonical/u)
})

test('uses prefetched asynchronous entropy when Web Crypto is absent', async () => {
  const descriptor = Object.getOwnPropertyDescriptor(globalThis, 'crypto')
  Object.defineProperty(globalThis, 'crypto', {
    configurable: true,
    writable: true,
    value: undefined,
  })
  try {
    const source = {
      async bytes(length) {
        return Uint8Array.from({ length }, (_, index) => index & 0xff)
      },
    }
    const { withSecureRandomRuntime } = await import('../dist/index.js')
    const generated = await withSecureRandomRuntime(source, async () => {
      const target = new Uint8Array(8)
      globalThis.crypto.getRandomValues(target)
      return target
    }, 64)
    assert.deepEqual(generated, new Uint8Array([0, 1, 2, 3, 4, 5, 6, 7]))
    assert.equal(globalThis.crypto, undefined)
  } finally {
    if (descriptor) Object.defineProperty(globalThis, 'crypto', descriptor)
    else delete globalThis.crypto
  }
})
