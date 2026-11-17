/** Server-clock-based visual countdown only; never authorizes a game action. */
export interface TurnTime {
  turnDeadlineMillis?: number
  serverTimeMillis?: number
}
function ms(value:unknown):value is number{
  return typeof value==='number' && Number.isSafeInteger(value) && value>0
}
export function acceptTurnTime(raw:TurnTime):TurnTime|null {
  if(!ms(raw.turnDeadlineMillis)||!ms(raw.serverTimeMillis))return null
  if(raw.turnDeadlineMillis<raw.serverTimeMillis-24*60*60*1000 ||
     raw.turnDeadlineMillis>raw.serverTimeMillis+24*60*60*1000)return null
  return {turnDeadlineMillis:raw.turnDeadlineMillis,serverTimeMillis:raw.serverTimeMillis}
}
export function remainingSeconds(time:TurnTime,monotonicElapsedMs:number):number|null{
  const accepted=acceptTurnTime(time)
  if(!accepted || !Number.isFinite(monotonicElapsedMs))return null
  const elapsed=Math.max(0,monotonicElapsedMs)
  const remaining=Math.max(0,accepted.turnDeadlineMillis!-accepted.serverTimeMillis!-elapsed)
  return Math.ceil(remaining/1000)
}
