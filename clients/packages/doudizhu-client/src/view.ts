/** Strict projection of Go's livehand.PrivateView; never fabricate other seats' cards. */
export interface SeatProjection {
  position: number
  accountId: string
  remainingCards: number
}
export interface PublicHandProjection {
  v: 'doudizhu-live-public-view-v1'
  handId: string
  phase: string
  stateVersion: number
  seats: readonly SeatProjection[]
  landlordSeat?: number
  playingSeat?: number
  winnerSeat?: number
  bidding: Readonly<Record<string, unknown>>
  playing?: Readonly<Record<string, unknown>> | null
}
export interface PrivateHandProjection {
  v: 'doudizhu-live-private-view-v1'
  public: PublicHandProjection
  position: number
  cards: readonly string[]
}
function record(v: unknown): v is Record<string, unknown> {
  return v !== null && typeof v === 'object' && !Array.isArray(v)
}
function version(v: unknown): v is number {
  return typeof v === 'number' && Number.isSafeInteger(v) && v >= 1
}
const cardCode = /^(?:[CDHS](?:[3-9]|10|J|Q|K|A|2)|XJ|YJ)$/
function seatNumber(value: unknown): value is number {
  return Number.isSafeInteger(value) && typeof value === 'number' && value >= 1 && value <= 3
}
function optionalSeat(pub: Record<string, unknown>, field: string): boolean {
  return pub[field] === undefined || seatNumber(pub[field])
}
/** Accept only a complete, authorized private view from the trusted HTTP/WSS layer. */
export function parsePrivateHandView(raw: unknown): PrivateHandProjection {
  if (!record(raw) || raw.v !== 'doudizhu-live-private-view-v1' || !record(raw.public))
    throw new Error('invalid private hand protocol')
  const pub = raw.public
  if (pub.v !== 'doudizhu-live-public-view-v1' ||
      typeof pub.handId !== 'string' || !pub.handId ||
      typeof pub.phase !== 'string' || !pub.phase ||
      !version(pub.stateVersion) || !record(pub.bidding) ||
      !Array.isArray(pub.seats) || pub.seats.length !== 3 ||
      !pub.seats.every(seat => record(seat) && seatNumber(seat.position) && typeof seat.accountId === 'string' &&
        seat.accountId.length > 0 && Number.isSafeInteger(seat.remainingCards) &&
        (seat.remainingCards as number) >= 0 && (seat.remainingCards as number) <= 20) ||
      new Set(pub.seats.map(seat => (seat as Record<string, unknown>).position)).size !== 3 ||
      new Set(pub.seats.map(seat => (seat as Record<string, unknown>).accountId)).size !== 3 ||
      !optionalSeat(pub, 'landlordSeat') || !optionalSeat(pub, 'playingSeat') || !optionalSeat(pub, 'winnerSeat') ||
      (pub.playing !== undefined && pub.playing !== null && !record(pub.playing))) {
    throw new Error('invalid public hand protocol')
  }
  if (!seatNumber(raw.position) ||
      !Array.isArray(raw.cards) || raw.cards.length > 20 ||
      !raw.cards.every(card => typeof card === 'string' && cardCode.test(card)) ||
      new Set(raw.cards).size !== raw.cards.length ||
      !pub.seats.some(seat => record(seat) && seat.position === raw.position)) {
    throw new Error('invalid private cards')
  }
  // Strict JSON copy protects the store against caller-side mutations; no secret card logging.
  const copied = JSON.parse(JSON.stringify(raw)) as PrivateHandProjection
  Object.freeze(copied.cards)
  Object.freeze(copied.public.seats)
  Object.freeze(copied.public)
  return Object.freeze(copied)
}
/** Go dispatcher wraps the raw private view as {version,payload}. */
export function parsePrivateViewResult(raw: unknown): { version: number; view: PrivateHandProjection } {
  if (!record(raw) || !version(raw.version)) throw new Error('invalid Go view wrapper')
  const view = parsePrivateHandView(raw.payload)
  if (view.public.stateVersion !== raw.version) throw new Error('private view version mismatch')
  return { version: raw.version, view }
}
