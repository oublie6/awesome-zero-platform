# Secure Envelope TypeScript Client

Engine-independent RFC 9180 HPKE sealer for `secure-envelope-v1`.

```ts
import { SecureEnvelopeSealer } from '@awesome-zero-platform/secure-envelope'

const envelope = await new SecureEnvelopeSealer({ randomSource }).seal({
  recipient: { keyId, publicKey },
  plaintext,
  aad,
})
```

Standard Web-interoperable runtimes use `crypto.getRandomValues`. Constrained hosts such as WeChat can provide an asynchronous `AsyncRandomSource`; the package prefetches true random bytes and fails closed rather than falling back to `Math.random`.

Business code owns canonical AAD construction, authentication, authorization, replay prevention, and state validation.
