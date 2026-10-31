import { test } from 'node:test'
import assert from 'node:assert/strict'
import { requireSecureBytes, websocketURL } from '../dist/index.js'
test('entropy source must fail closed', async () => {
  await assert.rejects(requireSecureBytes({bytes: async () => new Uint8Array(2)}, 32), /unavailable/)
  await assert.rejects(requireSecureBytes({bytes: async () => {throw Error('no platform entropy')}}, 32), /no platform entropy/)
  assert.equal((await requireSecureBytes({bytes: async n => new Uint8Array(n)}, 32)).length, 32)
})
test('socket URLs reject token query strings and invalid schemes', () => {
  assert.throws(() => websocketURL('wss://example.org/ws?token=secret'), /prohibited/)
  assert.throws(() => websocketURL('https://example.org/ws'), /scheme/)
  assert.equal(websocketURL('wss://example.org/ws').pathname, '/ws')
})
