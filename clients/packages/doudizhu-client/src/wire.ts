/**
 * Actual Go dispatcher protocol; distinct from the durable application envelope.
 * Source: server/apps/app-api/internal/doudizhuapi/dispatcher.go.
 * No Cocos / platform SDK references.
 */
export const WIRE_REQUEST_VERSION = 'doudizhu-api-command-v1' as const
export const WIRE_RESULT_VERSION = 'doudizhu-api-command-result-v1' as const

export type WireCommandType =
  | 'room.create' | 'room.join' | 'room.leave' | 'room.ready' | 'room.hand.start'
  | 'hand.commit' | 'hand.reveal' | 'hand.contribution' | 'hand.bid' | 'hand.play' | 'hand.pass' | 'hand.cancel'

export interface WireCommand {
  v: typeof WIRE_REQUEST_VERSION
  requestId: string
  type: WireCommandType
  aggregateId: string
  clientSeq: number
  expectedVersion: number
  payload?: Readonly<Record<string, unknown>>
  revealContext?: Readonly<SignedRevealContext>
}
export interface SignedRevealContext {
 claims:{
  v:'doudizhu-reveal-context-v1'|'doudizhu-custodied-context-v1'
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
 mac:string
}
export interface WireCommandResult {
  v: typeof WIRE_RESULT_VERSION
  requestId: string
  type: string
  durable?: Readonly<Record<string, unknown>>
  live?: { version: number; payload: unknown; requiresTermination: boolean }
  termination?: Readonly<Record<string, unknown>>
}

const commandTypes: Readonly<Record<WireCommandType, 'room' | 'hand'>> = {
  'room.create': 'room', 'room.join': 'room', 'room.leave': 'room',
  'room.ready': 'room', 'room.hand.start': 'room',
  'hand.commit': 'hand', 'hand.reveal': 'hand', 'hand.contribution': 'hand',
  'hand.bid': 'hand', 'hand.play': 'hand',
  'hand.pass': 'hand', 'hand.cancel': 'hand'
}

function object(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
}
function id(value: string): boolean {
  return typeof value === 'string' && value.length > 0 && value.length <= 128 && value === value.trim()
}
function integer(value: number, min: number): boolean {
  return Number.isSafeInteger(value) && value >= min
}

/** Validate against the current Go dispatcher and narrowly constrain known client payloads. */
export function createWireCommand(input: Omit<WireCommand, 'v'>): WireCommand {
  if (!id(input.requestId) || !id(input.aggregateId) ||
    !Object.prototype.hasOwnProperty.call(commandTypes, input.type) ||
    !integer(input.clientSeq, 1) || !integer(input.expectedVersion, input.type === 'room.create' ? 0 : 1) ||
    (input.type === 'room.create' && input.expectedVersion !== 0)) {
    throw new Error('invalid Go dispatcher command')
  }
  if (input.type !== 'hand.reveal' && input.type !== 'hand.contribution' && input.revealContext !== undefined)
    throw new Error('unexpected reveal ticket on command')
  const payload = input.payload ?? {}
  if (!object(payload)) throw new Error('invalid command payload')
  const keys = Object.keys(payload)
  const exactKeys = (expected: string[]): boolean =>
    keys.length === expected.length && expected.every(key => Object.prototype.hasOwnProperty.call(payload, key))
  switch (input.type) {
    case 'room.create': case 'room.join': case 'room.leave':
    case 'hand.pass': case 'hand.cancel':
      if (!exactKeys([])) throw new Error('unexpected command payload')
      break
    case 'room.ready':
      if (!exactKeys(['ready']) || typeof payload.ready !== 'boolean') throw new Error('invalid ready payload')
      break
    case 'room.hand.start':
      if ((!exactKeys(['handId']) && !exactKeys(['handId','fairnessMode'])) ||
          !id(payload.handId as string) ||
          (payload.fairnessMode!==undefined && payload.fairnessMode!=='client-contributed-server-custodied-v1'))
        throw new Error('invalid hand start payload')
      break
    case 'hand.commit':
      if (!exactKeys(['commitment']) || typeof payload.commitment !== 'string' ||
          !/^[0-9a-f]{64}$/.test(payload.commitment)) throw new Error('invalid commitment encoding')
      break
    case 'hand.bid':
      if (!exactKeys(['score']) || !integer(payload.score as number, 0) || (payload.score as number) > 3)
        throw new Error('invalid bid score')
      break
    case 'hand.play':
      if (!exactKeys(['cards']) || !Array.isArray(payload.cards) || payload.cards.length < 1 ||
          payload.cards.length > 20 || !payload.cards.every(card => typeof card === 'string' &&
          card.trim() === card && card.length > 0 && card.length <= 3))
        throw new Error('invalid play cards')
      break
    case 'hand.reveal': case 'hand.contribution': {
      const ticket=input.revealContext
      const claims=ticket?.claims
      if(!ticket || !object(claims) || claims.v!==(input.type==='hand.contribution'?'doudizhu-custodied-context-v1':'doudizhu-reveal-context-v1') ||
        typeof ticket.mac!=='string' || !/^[0-9a-f]{64}$/.test(ticket.mac) ||
        claims.handId!==input.aggregateId || claims.requestId!==input.requestId ||
        claims.clientSeq!==input.clientSeq || claims.expectedVersion!==input.expectedVersion ||
        typeof claims.actor!=='string' || !id(claims.actor) ||
        !integer(claims.seat,1) || claims.seat>3 ||
        typeof claims.keyId!=='string' || !id(claims.keyId) ||
        typeof claims.commitment!=='string' ||
        (input.type==='hand.contribution' ? claims.commitment!=='' : !/^[0-9a-f]{64}$/.test(claims.commitment)) ||
        typeof claims.publicKeySha256!=='string' || !/^[0-9a-f]{64}$/.test(claims.publicKeySha256) ||
        !integer(claims.issuedAt,0) || !integer(claims.expiresAt,1) ||
        claims.expiresAt<=claims.issuedAt)
        throw new Error('verified server AAD reveal context required')
      if(!exactKeys(['envelope']) || !object(payload.envelope))throw new Error('missing HPKE envelope')
      const e=payload.envelope
      if(Object.keys(e).length!==5 ||
        e.v!=='secure-envelope-v1' || e.keyId!==claims.keyId ||
        e.suite!=='hpke-x25519-hkdf-sha256-aes-256-gcm' ||
        typeof e.encapsulatedKey!=='string' || !/^[A-Za-z0-9_-]+$/.test(e.encapsulatedKey) ||
        typeof e.ciphertext!=='string' || !/^[A-Za-z0-9_-]+$/.test(e.ciphertext))
        throw new Error('invalid bound HPKE reveal envelope')
      break
    }
  }
  return Object.freeze({
    v: WIRE_REQUEST_VERSION,
    requestId: input.requestId,
    type: input.type,
    aggregateId: input.aggregateId,
    clientSeq: input.clientSeq,
    expectedVersion: input.expectedVersion,
    payload: JSON.parse(JSON.stringify(payload)) as Readonly<Record<string, unknown>>,
    ...(input.revealContext === undefined ? {} : {revealContext:JSON.parse(JSON.stringify(input.revealContext)) as SignedRevealContext})
  })
}

/** Strictly acknowledge only a response correlated with the original request. */
export function parseWireResult(value: unknown, request: WireCommand): WireCommandResult {
  if (!object(value) || value.v !== WIRE_RESULT_VERSION ||
    value.requestId !== request.requestId || value.type !== request.type ||
    (value.durable === undefined && value.live === undefined && value.termination === undefined)) {
    throw new Error('invalid or uncorrelated Go command result')
  }
  if (value.durable !== undefined && !object(value.durable)) throw new Error('invalid durable result')
  if (value.live !== undefined &&
      (!object(value.live) || !integer(value.live.version as number, 1) ||
       typeof value.live.requiresTermination !== 'boolean' || !object(value.live.payload))) {
    throw new Error('invalid live result')
  }
  if (value.termination !== undefined && !object(value.termination)) throw new Error('invalid termination result')
  return value as unknown as WireCommandResult
}
