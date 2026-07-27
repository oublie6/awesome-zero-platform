export class SecureEnvelopeError extends Error {
  override readonly name: string = 'SecureEnvelopeError'
}

export class SecureRandomUnavailableError extends SecureEnvelopeError {
  override readonly name: string = 'SecureRandomUnavailableError'
}
