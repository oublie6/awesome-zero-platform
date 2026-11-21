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

/** A monotonic clock anchor survives temporary Cocos view destruction.
 * It is presentation-only: neither a game deadline authority nor persistence. */
export class TurnClockAnchor {
  private handID=''
  private version=0
  private deadline=0
  private serverAtCapture=0
  private capturedAt=0
  private static now():number{
    return typeof performance==='undefined'?Date.now():performance.now()
  }
  observe(handID:string,version:number,raw:TurnTime,now=TurnClockAnchor.now()):void {
    const accepted=acceptTurnTime(raw)
    if(!accepted || !handID || !Number.isSafeInteger(version)){
      if(this.handID!==handID || this.version!==version){
        this.handID=handID;this.version=version;this.deadline=0
      }
      return
    }
    const deadline=accepted.turnDeadlineMillis!
    const serverTime=accepted.serverTimeMillis!
    if(this.handID!==handID||this.version!==version||this.deadline!==deadline){
      this.handID=handID;this.version=version;this.deadline=deadline
      this.serverAtCapture=serverTime;this.capturedAt=now
    }else{
      // A repeated/stale projection must never extend the shown countdown.
      const estimated=this.serverAtCapture+Math.max(0,now-this.capturedAt)
      if(serverTime>estimated){
        this.serverAtCapture=serverTime;this.capturedAt=now
      }
    }
  }
  remainingSeconds(now=TurnClockAnchor.now()):number|null{
    if(!this.deadline)return null
    return remainingSeconds({turnDeadlineMillis:this.deadline,
      serverTimeMillis:this.serverAtCapture},Math.max(0,now-this.capturedAt))
  }
}
