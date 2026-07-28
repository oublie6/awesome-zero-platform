import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import {
  InMemoryManifestVersionStore,
  verifyRevealKeyManifest,
} from '../dist/index.js'

if (process.argv.length !== 3) throw new Error('usage: verify-manifest-interop VECTOR')
const vector = JSON.parse(await readFile(process.argv[2], 'utf8'))
const verified = verifyRevealKeyManifest(vector.manifest, {
  roots: { [vector.rootKeyId]: vector.rootPublicKey },
  versions: new InMemoryManifestVersionStore(),
  mode: 'current',
  now: new Date(vector.now),
})
assert.equal(verified.keyId, vector.manifest.keyId)
assert.equal(verified.manifestVersion, vector.manifest.manifestVersion)
console.log(`verified Go-signed reveal key manifest ${verified.keyId}`)
