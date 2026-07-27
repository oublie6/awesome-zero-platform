import fs from 'node:fs/promises'
import process from 'node:process'

import * as HPKE from 'hpke'
import {
  AEAD_AES_256_GCM,
  KDF_HKDF_SHA256,
  KEM_DHKEM_X25519_HKDF_SHA256,
} from '@panva/hpke-noble'

import { encodeBase64Url, SecureEnvelopeSealer } from '../dist/index.js'

const outputPath = process.argv[2]
if (!outputPath) throw new Error('usage: generate-interop-vector.mjs <output-path>')

const suite = new HPKE.CipherSuite(
  KEM_DHKEM_X25519_HKDF_SHA256,
  KDF_HKDF_SHA256,
  AEAD_AES_256_GCM,
)
const keyPair = await suite.GenerateKeyPair(true)
const recipientPublicKey = await suite.SerializePublicKey(keyPair.publicKey)
const recipientPrivateKey = await suite.SerializePrivateKey(keyPair.privateKey)
const plaintext = new TextEncoder().encode(
  JSON.stringify({ secureRandom: 'interop-random', phraseRaw: 'interop phrase' }),
)
const aad = new TextEncoder().encode(
  'protocol=1;game=game-interop;hand=hand-interop;seat=0;command=cmd-interop',
)
const envelope = await new SecureEnvelopeSealer().seal({
  recipient: { keyId: 'interop-key-1', publicKey: recipientPublicKey },
  plaintext,
  aad,
})
await fs.writeFile(
  outputPath,
  `${JSON.stringify(
    {
      generatedBy: '@awesome-zero-platform/secure-envelope',
      recipientPublicKey: encodeBase64Url(recipientPublicKey),
      recipientPrivateKey: encodeBase64Url(recipientPrivateKey),
      aad: encodeBase64Url(aad),
      plaintext: encodeBase64Url(plaintext),
      envelope,
    },
    null,
    2,
  )}\n`,
)
