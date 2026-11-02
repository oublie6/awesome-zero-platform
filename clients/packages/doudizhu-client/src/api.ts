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
  /** Pre-deal state must come from Go; the live-hand endpoint is unavailable until dealing. */
  async fairness(handId: string): Promise<FairnessRead> {
    if (!/^[A-Za-z0-9_-]{1,128}$/.test(handId)) throw new Error('invalid fairness hand id')
    const raw=await this.transport.request<unknown>(
      '/v1/doudizhu/hands/'+encodeURIComponent(handId)+'/fairness',
      {accessToken:this.token()})
    const data=unwrap(raw)
    if (!record(data) || data.handId!==handId ||
      !['FAIRNESS_COMMITTING','FAIRNESS_REVEALING','WAITING_PUBLIC_BEACON','DEALING','BIDDING','PLAYING','SETTLING','COMPLETED','CANCELLED','ABORTED','EXPIRED'].includes(data.phase as string) ||
      !Number.isSafeInteger(data.version) || (data.version as number)<1 ||
      !Number.isSafeInteger(data.seat) || (data.seat as number)<1 || (data.seat as number)>3 ||
      typeof data.committed!=='boolean' || typeof data.revealed!=='boolean' ||
      !Number.isSafeInteger(data.lastClientSequence) || (data.lastClientSequence as number)<0 ||
      typeof data.revealKeyId!=='string' || !data.revealKeyId ||
      typeof data.revealKeyBoundAt!=='string' ||
      typeof data.serverCommitment!=='string' || !/^[0-9a-f]{64}$/.test(data.serverCommitment) ||
      typeof data.revealPublicKeySha256!=='string' || !/^[0-9a-f]{64}$/.test(data.revealPublicKeySha256) ||
      (data.committed && (typeof data.commitment!=='string' || !/^[0-9a-f]{64}$/.test(data.commitment))) ||
      (!data.committed && data.commitment!==undefined) || (data.revealed && !data.committed)) {
      throw new Error('invalid authoritative fairness projection')
    }
    return JSON.parse(JSON.stringify(data)) as FairnessRead
  }
  async revealContext(handId: string,request:RevealContextInput):Promise<RevealContextRead> {
    if (!/^[A-Za-z0-9_-]{1,128}$/.test(handId) ||
      typeof request.requestId!=='string' || !request.requestId || request.requestId.length>128 ||
      !Number.isSafeInteger(request.clientSeq) || request.clientSeq<1 ||
      !Number.isSafeInteger(request.expectedVersion) || request.expectedVersion<1)
      throw new Error('invalid reveal context request')
    const raw=await this.transport.request<unknown>(
      '/v1/doudizhu/hands/'+encodeURIComponent(handId)+'/reveal-context',
      {method:'POST',body:request,accessToken:this.token()})
    const data=unwrap(raw)
    if(!record(data) || data.v!=='doudizhu-reveal-context-v1' || !record(data.ticket) ||
       !record(data.ticket.claims) || typeof data.ticket.mac!=='string' ||
       !/^[0-9a-f]{64}$/.test(data.ticket.mac) ||
       typeof data.aad!=='string' || !/^[A-Za-z0-9_-]+$/.test(data.aad)) throw new Error('invalid signed reveal context')
    const claims=data.ticket.claims
    if(claims.v!=='doudizhu-reveal-context-v1' ||
       claims.handId!==handId || claims.requestId!==request.requestId ||
       claims.clientSeq!==request.clientSeq || claims.expectedVersion!==request.expectedVersion ||
       typeof claims.actor!=='string' || !claims.actor ||
       !Number.isSafeInteger(claims.seat) || (claims.seat as number)<1 || (claims.seat as number)>3 ||
       typeof claims.commitment!=='string' || !/^[0-9a-f]{64}$/.test(claims.commitment) ||
       typeof claims.publicKeySha256!=='string' || !/^[0-9a-f]{64}$/.test(claims.publicKeySha256) ||
       typeof claims.keyId!=='string' || !claims.keyId ||
       !Number.isSafeInteger(claims.issuedAt) || !Number.isSafeInteger(claims.expiresAt) ||
       (claims.expiresAt as number)<=(claims.issuedAt as number))
       throw new Error('mismatched server reveal context')
    return JSON.parse(JSON.stringify(data)) as RevealContextRead
  }
  async revealManifest(keyId: string):Promise<unknown> {
    if(!/^[A-Za-z0-9_-]{1,128}$/.test(keyId))throw new Error('invalid signed manifest key id')
    const raw=await this.transport.request<unknown>('/api/v1/crypto/reveal-keys/'+encodeURIComponent(keyId),
      {accessToken:this.token()})
    return unwrap(raw)
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
  lastClientSequence: number
  seats?: ReadonlyArray<{position:number;accountId:string;ready:boolean}>
  activeHandId?: string
}
function parseRoomRead(raw: unknown, expectedID: string): RoomRead {
  if (!record(raw) || raw.roomId !== expectedID ||
      !Number.isSafeInteger(raw.version) || (raw.version as number) < 1 ||
      !['WAITING_PLAYERS','READY','HAND_ACTIVE','CLOSED'].includes(raw.status as string) ||
      !Number.isSafeInteger(raw.occupied) || (raw.occupied as number) < 0 || (raw.occupied as number) > 3 ||
      typeof raw.member !== 'boolean' || !Number.isSafeInteger(raw.lastClientSequence) ||
      (raw.lastClientSequence as number) < 0) throw new Error('invalid authoritative room view')
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

export interface FairnessRead {
 handId:string
 phase:string
 version:number
 seat:number
 committed:boolean
 revealed:boolean
 commitment?:string
 serverCommitment:string
 revealKeyId:string
 revealPublicKeySha256:string
 revealKeyBoundAt:string
 lastClientSequence:number
}
export interface RevealContextInput {requestId:string;clientSeq:number;expectedVersion:number}
export interface RevealContextRead {
 v:'doudizhu-reveal-context-v1'
 aad:string
 ticket:{
  mac:string
  claims:{
   v:'doudizhu-reveal-context-v1'
   actor:string
   handId:string
   requestId:string
   clientSeq:number
   expectedVersion:number
   seat:number
   commitment:string
   keyId:string
   publicKeySha256:string
   issuedAt:number
   expiresAt:number
  }
 }
}
