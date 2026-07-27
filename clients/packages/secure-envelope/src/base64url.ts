const alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_'

export function encodeBase64Url(bytes: Uint8Array): string {
  let output = ''
  for (let index = 0; index < bytes.length; index += 3) {
    const first = bytes[index] ?? 0
    const second = bytes[index + 1] ?? 0
    const third = bytes[index + 2] ?? 0
    const value = (first << 16) | (second << 8) | third
    output += alphabet[(value >>> 18) & 63]
    output += alphabet[(value >>> 12) & 63]
    if (index + 1 < bytes.length) output += alphabet[(value >>> 6) & 63]
    if (index + 2 < bytes.length) output += alphabet[value & 63]
  }
  return output
}

export function decodeBase64Url(value: string): Uint8Array {
  if (value.length === 0 || value.length % 4 === 1 || /[^A-Za-z0-9_-]/u.test(value)) {
    throw new TypeError('invalid base64url value')
  }
  const output = new Uint8Array(Math.floor((value.length * 6) / 8))
  let buffer = 0
  let bits = 0
  let offset = 0
  for (const character of value) {
    const decoded = alphabet.indexOf(character)
    if (decoded < 0) throw new TypeError('invalid base64url value')
    buffer = (buffer << 6) | decoded
    bits += 6
    if (bits >= 8) {
      bits -= 8
      output[offset++] = (buffer >>> bits) & 0xff
    }
  }
  if (encodeBase64Url(output) !== value) {
    throw new TypeError('non-canonical base64url value')
  }
  return output
}
