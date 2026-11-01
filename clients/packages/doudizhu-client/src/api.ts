/** Structural port avoids coupling this package to a specific platform transport. */
export interface DoudizhuHttpTransport {
  request<T>(path: string, options?: { method?: string; body?: unknown; accessToken?: string }): Promise<T>
}
import { createWireCommand, parseWireResult, type WireCommand, type WireCommandResult } from './wire.js'
import { parsePrivateViewResult, type PrivateHandProjection } from './view.js'

interface APIEnvelope { code: string; data: unknown; message?: string }
function record(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
}
/** Actual platform response.Envelope; fail closed on malformed or non-success responses. */
function unwrap(value: unknown): unknown {
  if (!record(value) || typeof value.code !== 'string' || !('data' in value)) {
    throw new Error('invalid Go API response envelope')
  }
  if (value.code !== 'OK') throw new Error('Go API rejected request: ' + value.code)
  return (value as unknown as APIEnvelope).data
}
/** Injectable transport; no platform SDK or implicit credentials in the business layer. */
export class DoudizhuHTTPClient {
  constructor(private readonly transport: DoudizhuHttpTransport, private readonly accessToken: () => string) {}
  private token(): string {
    const token = this.accessToken()
    if (typeof token !== 'string' || !token.trim() || /[\r\n]/.test(token)) throw new Error('authenticated token required')
    return token
  }
  async command(input: Omit<WireCommand, 'v'>): Promise<WireCommandResult> {
    const request = createWireCommand(input)
    const raw = await this.transport.request<unknown>('/v1/doudizhu/commands', {
      method: 'POST', body: request, accessToken: this.token()
    })
    return parseWireResult(unwrap(raw), request)
  }
  async privateHand(handId: string): Promise<{version: number; view: PrivateHandProjection}> {
    if (!/^[A-Za-z0-9_-]{1,128}$/.test(handId)) throw new Error('invalid hand id')
    const raw = await this.transport.request<unknown>(
      '/v1/doudizhu/hands/' + encodeURIComponent(handId) + '/private',
      {accessToken: this.token()}
    )
    return parsePrivateViewResult(unwrap(raw))
  }
}
