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
  /** Authenticated room read: only members receive account seats or activeHandId. */
  async room(roomId: string): Promise<RoomRead> {
    if (!/^[A-Za-z0-9_-]{1,128}$/.test(roomId)) throw new Error('invalid room id')
    const raw = await this.transport.request<unknown>(
      '/v1/doudizhu/rooms/' + encodeURIComponent(roomId),
      {accessToken:this.token()}
    )
    return parseRoomRead(unwrap(raw), roomId)
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

export interface RoomRead {
  roomId: string
  version: number
  status: 'WAITING_PLAYERS'|'READY'|'HAND_ACTIVE'|'CLOSED'
  occupied: number
  member: boolean
  seats?: ReadonlyArray<{position:number;accountId:string;ready:boolean}>
  activeHandId?: string
}
function parseRoomRead(raw: unknown, expectedID: string): RoomRead {
  if (!record(raw) || raw.roomId !== expectedID ||
      !Number.isSafeInteger(raw.version) || (raw.version as number) < 1 ||
      !['WAITING_PLAYERS','READY','HAND_ACTIVE','CLOSED'].includes(raw.status as string) ||
      !Number.isSafeInteger(raw.occupied) || (raw.occupied as number) < 0 || (raw.occupied as number) > 3 ||
      typeof raw.member !== 'boolean') throw new Error('invalid authoritative room view')
  if (raw.member) {
    if (!Array.isArray(raw.seats) || raw.seats.length !== 3 ||
        !raw.seats.every(seat => record(seat) &&
        typeof seat.position === 'number' && Number.isSafeInteger(seat.position) && seat.position >= 1 && seat.position <= 3 &&
        typeof seat.accountId === 'string' && typeof seat.ready === 'boolean') ||
        new Set(raw.seats.map(s => s.position)).size !== 3 ||
        (raw.activeHandId !== undefined && typeof raw.activeHandId !== 'string')) {
      throw new Error('invalid member room view')
    }
    if (raw.seats.filter(s => s.accountId !== '').length !== raw.occupied ||
        new Set(raw.seats.filter(s => s.accountId !== '').map(s=>s.accountId)).size !== raw.occupied) {
      throw new Error('invalid room seat ownership')
    }
  } else if (raw.seats !== undefined || raw.activeHandId !== undefined) {
    throw new Error('room preview leaked private membership fields')
  }
  return JSON.parse(JSON.stringify(raw)) as RoomRead
}
