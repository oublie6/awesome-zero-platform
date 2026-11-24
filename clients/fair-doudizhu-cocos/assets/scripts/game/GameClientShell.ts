import { createWireCommand, parseWireResult, type WireCommand, type WireCommandResult } from '@awesome-zero-platform/doudizhu-client/wire'
import { parsePrivateViewResult } from '@awesome-zero-platform/doudizhu-client/view'
import { parseFinalEvidenceRead, type FinalEvidenceRead } from '@awesome-zero-platform/doudizhu-client/api'
import { HandStore, GameMessageRouter, type HandSnapshot } from '@awesome-zero-platform/doudizhu-client'
import { ConnectionManager, type SocketTransport } from '@awesome-zero-platform/game-client-core'

/** UI orchestration over confirmed server messages. No generated fake game state. */
/** This is an explicit, correlated pre-commit rejection from the Go
 * dispatcher. INTERNAL, transport loss and timeouts are never definite. */
export class CommandRejectedError extends Error {
  constructor(readonly code: string) {
    super('server rejected command: ' + code)
    this.name = 'CommandRejectedError'
  }
}
const DEFINITIVELY_REJECTED = new Set(['INVALID_REQUEST','CONFLICT','FORBIDDEN','NOT_FOUND'])
const MAX_PENDING_COMMANDS = 32
const DEFAULT_SYNC_TIMEOUT_MS = 15000
type PendingSync = {
  id:string; handId:string; generation:number; minimumVersion:number; knownVersion?:number;
  promise:Promise<void>; resolve:()=>void; reject:(error:Error)=>void;
  timer:ReturnType<typeof setTimeout>
}
const SERVER_ERROR_CODES = new Set(['INVALID_REQUEST', 'CONFLICT', 'FORBIDDEN', 'NOT_FOUND', 'INTERNAL'])
export class GameClientShell {
  readonly hand = new HandStore()
  readonly events = new GameMessageRouter()
  private connection: ConnectionManager | null = null
  private disposed = false
  private initialized = false
  private transportReady = false
  get commandTransportReady(): boolean { return !this.disposed && this.transportReady && this.connection !== null }
  /** Query readiness again immediately before a caller publishes success.
   * A newer change or a disconnect can invalidate an earlier resolved read. */
  handProjectionReady(handId:string,archiveOnly=false):boolean {
    const snapshot=this.hand.snapshot
    if(!this.commandTransportReady || this.pendingSync ||
       this.hand.status!=='connected' || !snapshot || snapshot.handId!==handId ||
       snapshot.version<this.requiredHandVersion)return false
    if(archiveOnly)return ['COMPLETED','ABORTED'].includes(snapshot.phase) &&
      !!snapshot.view.terminalEvidence
    return ['BIDDING','PLAYING','SETTLING','COMPLETED','ABORTED'].includes(snapshot.phase)
  }
  private generation = 0
  private syncSequence = 0
  // One bounded, correlated read per controller/connection/hand. Broadcasts
  // may advance state but never acknowledge an explicit recovery request.
  private pendingSync:PendingSync|null=null
  private requiredHandVersion=0
  // Bound only by authenticated room/start-hand evidence, not a realtime
  // event. A stale prior-room snapshot must not populate an empty HandStore.
  private authoritativeHand: string | null = null
  private handBound = false
  private pendingCommands = new Map<string, { request: WireCommand; resolve: (value: WireCommandResult) => void; reject: (reason: Error) => void; timer: ReturnType<typeof setTimeout> }>()
  private notify: (message: string) => void
  constructor(notify: (message: string) => void, private readonly syncTimeoutMs=DEFAULT_SYNC_TIMEOUT_MS) {
    if (!Number.isSafeInteger(syncTimeoutMs) || syncTimeoutMs<1 || syncTimeoutMs>60000)
      throw new Error('invalid hand sync timeout')
    this.notify = message => { try { notify(message) } catch { /* presentation cannot settle protocol operations */ } }
  }
  async initialize(): Promise<void> {
    if (this.disposed || this.initialized) return
    this.initialized = true
    this.notify('连接模块就绪，请登录并同步服务端房间状态')
    this.events.on('doudizhu.command.result', payload => {
      if (this.disposed) return
      const id = (payload as {requestId?: unknown})?.requestId
      if (typeof id !== 'string') return
      const pending = this.pendingCommands.get(id)
      if (!pending) return
      try {
        const result = parseWireResult(payload, pending.request)
        this.pendingCommands.delete(id)
        clearTimeout(pending.timer)
        // A committed live mutation is not a confirmed private projection.
        // Block subsequent new actions until a snapshot reaches its receipt.
        if(result.live && pending.request.aggregateId===this.hand.snapshot?.handId){
          this.requiredHandVersion=Math.max(this.requiredHandVersion,result.live.version)
          if(this.hand.snapshot.version<this.requiredHandVersion && this.hand.status==='connected')
            this.hand.setStatus('synchronizing')
        }
        pending.resolve(result)
      } catch (error) {
        this.pendingCommands.delete(id)
        clearTimeout(pending.timer)
        pending.reject(error instanceof Error ? error : new Error('invalid command response'))
      }
    })
    this.events.on('doudizhu.hand.changed', payload => {
      const changed=payload as {handId?:string;response?:{live?:{version?:unknown}}}
      const id=changed.handId
      if (this.handBound && id !== this.authoritativeHand) return
      if (this.hand.snapshot && id !== this.hand.snapshot.handId) return
      const version=changed.response?.live?.version
      if(typeof version==='number' && Number.isSafeInteger(version) && version>0){
        this.requiredHandVersion=Math.max(this.requiredHandVersion,version)
        // A broadcast may race ahead of its corresponding changed event.
        // Do not invalidate an already confirmed equal/newer projection.
        if(this.hand.snapshot && this.hand.snapshot.version>=this.requiredHandVersion)return
      }
      if (this.hand.status === 'connected') this.hand.setStatus('synchronizing')
      this.notify('收到牌局状态变化，请同步权威快照')
    })
    this.events.on('doudizhu.hand.snapshot', (payload,responseId) => {
      if (this.disposed) return
      const pending=this.pendingSync
      // An expired/replaced request's reply cannot mutate current state.
      if(responseId && pending?.id!==responseId)return
      const reply=responseId && pending?.id===responseId?pending:null
      const event = payload as {handId?: string; knownVersion?: number; notModified?: boolean; view?: unknown}
      try {
        if(reply && event.handId!==reply.handId)throw new Error('hand sync response identity mismatch')
        if (this.handBound && event.handId !== this.authoritativeHand) {
          if(reply)throw new Error('hand sync outside authorized hand context')
          return
        }
        if (event.notModified) {
          // Only a matching conditional read with an existing trusted
          // snapshot can confirm notModified; a broadcast cannot do so.
          if(!reply)return
          const current=this.hand.snapshot
          if(reply.knownVersion===undefined || event.knownVersion!==reply.knownVersion ||
             !current || current.handId!==reply.handId || current.version!==event.knownVersion)
            throw new Error('invalid notModified hand sync response')
          this.completeSync(reply,current.version)
          return
        }
        if (!event.view)throw new Error('missing private hand snapshot')
        const {version, view} = parsePrivateViewResult(event.view)
        if (event.handId !== view.public.handId) throw new Error('hand identity mismatch')
        if (view.public.phase !== 'BIDDING' && view.public.phase !== 'PLAYING' && view.public.phase !== 'COMPLETED')
          throw new Error('unsupported hand phase')
        if(reply && version<Math.max(reply.minimumVersion,this.requiredHandVersion,this.hand.snapshot?.version??0))
          throw new Error('stale private hand sync response')
        this.acceptVerifiedSnapshot({
          handId: view.public.handId,
          version,
          phase: view.public.phase,
          view: view as unknown as Readonly<Record<string, unknown>>
        })
        if(reply)this.completeSync(reply,version)
      } catch(error) {
        if(reply)this.failSync(reply,error instanceof Error?error:new Error('invalid hand sync response'))
        this.notify('私有牌局数据校验失败，请重新同步')
      }
    })
    this.events.on('doudizhu.hand.evidence', (payload,responseId) => {
      const event=payload as {handId?:string;evidence?:unknown}
      const id=event.handId
      const pending=this.pendingSync
      if(responseId && pending?.id!==responseId)return
      const reply=responseId && pending?.id===responseId?pending:null
      if(reply){
        try {
          if(id!==reply.handId || (this.handBound && id!==this.authoritativeHand))
            throw new Error('terminal sync identity mismatch')
          const evidence=parseFinalEvidenceRead(event.evidence,reply.handId)
          if(evidence.finalVersion<Math.max(reply.minimumVersion,this.requiredHandVersion,this.hand.snapshot?.version??0))
            throw new Error('stale terminal sync response')
          this.acceptVerifiedFinalEvidence(evidence)
          this.completeSync(reply,evidence.finalVersion)
        }catch(error){
          this.failSync(reply,error instanceof Error?error:new Error('invalid terminal sync response'))
        }
        return
      }
      if (this.handBound && id !== this.authoritativeHand) return
      if (this.hand.snapshot && id !== this.hand.snapshot.handId) return
      // Uncorrelated terminal notifications remain hints for the HTTP
      // evidence path; they cannot complete a pending explicit sync.
      if (this.hand.status === 'connected') this.hand.setStatus('synchronizing')
      this.notify('已收到终局证据，牌局操作已停止')
    })
    this.events.on('doudizhu.error', (payload,responseId) => {
      if (this.disposed) return
      const event = payload as {requestId?: unknown; code?: unknown}
      const sync=this.pendingSync
      if(sync && responseId===sync.id){
        this.failSync(sync,new Error('hand sync rejected: '+(SERVER_ERROR_CODES.has(event.code as string)?event.code:'invalid response')))
        return
      }
      if(responseId?.startsWith('sync-'))return // stale read error
      if (typeof event.requestId === 'string' && SERVER_ERROR_CODES.has(event.code as string)) {
        const pending = this.pendingCommands.get(event.requestId)
        if (pending) {
          this.pendingCommands.delete(event.requestId)
          clearTimeout(pending.timer)
          pending.reject(DEFINITIVELY_REJECTED.has(event.code as string)
            ? new CommandRejectedError(event.code as string)
            : new Error('server error; command outcome unknown: ' + event.code))
        }
      }
      this.notify('服务器拒绝操作，请同步后重试')
    })
  }
  async connect(socket: SocketTransport, endpoint: string, token: string): Promise<void> {
    if (this.disposed) throw new Error('client disposed')
    const generation = ++this.generation
    this.transportReady = false
    this.rejectSync('connection replaced during hand sync')
    this.rejectPending('connection replaced; command outcome unknown')
    this.connection?.close()
    if (this.hand.status !== 'disconnected') this.hand.setStatus('disconnected')
    const connection = new ConnectionManager(socket)
    this.connection = connection
    this.hand.setStatus('reconnecting')
    try {
      await connection.open(endpoint, token,
        raw => {
          if (this.disposed || generation !== this.generation) return
          try { this.events.receive(raw) } catch {
            // Router rejects malformed envelopes before event dispatch. If
            // the malformed reply is correlated, fail that read immediately.
            try {
              const id=(JSON.parse(raw) as {id?:unknown}|null)?.id
              if(this.pendingSync && id===this.pendingSync.id)
                this.failSync(this.pendingSync,new Error('malformed hand sync response'))
            }catch { /* uncorrelated invalid JSON cannot settle another read */ }
            this.notify('收到不合法的服务器消息')
          }
        },
        () => { if (!this.disposed && generation === this.generation) { this.transportReady = false; this.rejectSync('connection lost during hand sync'); this.rejectPending('connection lost; command outcome unknown'); this.hand.setStatus('disconnected'); this.notify('连接中断，请重连') } })
      if (this.disposed || generation !== this.generation) return
      this.transportReady = true
      this.hand.setStatus('synchronizing')
      this.notify('已连接，等待房间或牌局状态同步')
    } catch (error) {
      if (this.disposed || generation !== this.generation) return
      this.transportReady = false
      this.hand.setStatus('disconnected')
      throw error
    }
  }
  /** Called only after an authenticated room read or accepted start/leave.
   * It permits a NEW hand identity without weakening HandStore's normal
   * cross-hand rejection, and keeps previous realtime messages fenced. */
  useAuthoritativeHand(handId: string | null): void {
    if (this.disposed) throw new Error('client disposed')
    if (handId !== null && (!/^[A-Za-z0-9_-]{1,128}$/.test(handId)))
      throw new Error('invalid authoritative hand identity')
    if (this.pendingCommands.size) throw new Error('hand command still pending')
    if (this.handBound && this.authoritativeHand === handId) return
    this.rejectSync('authoritative hand changed during synchronization')
    this.requiredHandVersion=0
    this.authoritativeHand = handId
    this.handBound = true
    this.hand.clearHand()
  }
  clearHand(): void { this.useAuthoritativeHand(null) }

  /** Only call with authenticated HTTP evidence already validated by
   * DoudizhuHTTPClient.finalEvidence. A terminal archive replaces live
   * presentation state; no private hand/cards are invented after cleanup. */
  acceptVerifiedFinalEvidence(evidence: FinalEvidenceRead): void {
    if (this.disposed) return
    const handId = evidence.handId
    if (this.handBound && handId !== this.authoritativeHand)
      throw new Error('terminal evidence outside authorized hand context')
    const phase = String(evidence.status).toUpperCase()
    const verification = evidence.verification
    if (!handId || !Number.isSafeInteger(evidence.finalVersion) || evidence.finalVersion < 1 ||
        !['COMPLETED','ABORTED'].includes(phase) ||
        verification.handId !== handId || verification.finalVersion !== evidence.finalVersion ||
        typeof verification.recordDigest !== 'string' || !/^[0-9a-f]{64}$/.test(verification.recordDigest) ||
        !Array.isArray(verification.seatPoints) || verification.seatPoints.length !== 3 ||
        !verification.seatPoints.every(Number.isSafeInteger)) {
      throw new Error('invalid verified final evidence')
    }
    const prior = this.hand.snapshot
    if (prior && prior.handId !== handId) throw new Error('terminal evidence hand mismatch')
    if (prior && evidence.finalVersion < prior.version) throw new Error('stale terminal evidence')
    if (prior && prior.version === evidence.finalVersion && prior.phase !== phase)
      throw new Error('terminal evidence version conflict')
    const snapshot: HandSnapshot = {
      handId, version: evidence.finalVersion, phase: phase as 'COMPLETED'|'ABORTED',
      view: { terminalEvidence: {
        winnerSeat: verification.winnerSeat ?? null,
        seatPoints: verification.seatPoints,
        recordDigest: verification.recordDigest,
      } },
    }
    this.applyVerifiedSnapshot(snapshot,true)
  }
  acceptVerifiedSnapshot(snapshot: HandSnapshot): void {
    this.applyVerifiedSnapshot(snapshot,false)
  }
  private applyVerifiedSnapshot(snapshot:HandSnapshot,terminalArchive:boolean):void {
    if (this.disposed) return
    if (this.handBound && snapshot.handId !== this.authoritativeHand)
      throw new Error('snapshot outside authorized hand context')
    const applied = terminalArchive?this.hand.applyTerminalArchive(snapshot):this.hand.apply(snapshot)
    const confirmed = this.hand.snapshot
    if (!this.pendingSync && this.hand.status === 'synchronizing' &&
        confirmed?.handId === snapshot.handId && confirmed.version === snapshot.version &&
        confirmed.version>=this.requiredHandVersion) {
      this.hand.setStatus('connected')
    }
    if (applied) this.notify(`牌局 ${snapshot.handId} · 版本 ${snapshot.version} · ${snapshot.phase}`)
  }
  /** Await server reply, strict projection and HandStore adoption, not send.
   * One in-flight read per shell; same compatible request shares the promise.
   * The deadline starts before transport.send, including a stalled driver. */
  syncHand(handId:string,knownVersion?:number):Promise<void> {
    if(this.disposed || !this.connection || !this.commandTransportReady ||
       !['synchronizing','connected'].includes(this.hand.status))
      return Promise.reject(new Error('not connected'))
    if(this.handBound && handId!==this.authoritativeHand)
      return Promise.reject(new Error('hand sync outside authorized hand context'))
    if(!/^[A-Za-z0-9_-]{1,128}$/.test(handId) ||
       (knownVersion!==undefined && (!Number.isSafeInteger(knownVersion) || knownVersion<0)))
      return Promise.reject(new Error('invalid hand sync'))
    const current=this.hand.snapshot
    // Do not advertise a known version without actually possessing it.
    const conditional=current?.handId===handId && current.version===knownVersion &&
      current.version>=this.requiredHandVersion?knownVersion:undefined
    const minimumVersion=Math.max(this.requiredHandVersion,
      current?.handId===handId?current.version:0,knownVersion??0)
    const old=this.pendingSync
    if(old && old.handId===handId && old.generation===this.generation &&
       old.knownVersion===conditional && old.minimumVersion>=minimumVersion)return old.promise
    this.rejectSync('hand sync superseded by a newer read')
    if(this.hand.status==='connected')this.hand.setStatus('synchronizing')
    const id='sync-'+(++this.syncSequence)
    const connection=this.connection
    let resolve!:()=>void,reject!:(error:Error)=>void
    const promise=new Promise<void>((ok,fail)=>{resolve=ok;reject=fail})
    const entry:PendingSync={id,handId,generation:this.generation,minimumVersion,
      knownVersion:conditional,promise,resolve,reject,timer:setTimeout(()=>{
        this.failSync(entry,new Error('hand sync response timed out'))
      },this.syncTimeoutMs)}
    this.pendingSync=entry
    void connection.send({id,type:'doudizhu.hand.sync',
      payload:{v:'doudizhu-realtime-sync-v1',handId,
        ...(conditional===undefined?{}:{knownVersion:conditional})}}).catch(()=>{
      this.failSync(entry,new Error('hand sync send failed'))
    })
    return promise
  }
  private completeSync(entry:PendingSync,version:number):void {
    if(this.pendingSync!==entry)return
    const snapshot=this.hand.snapshot
    if(entry.generation!==this.generation || !this.commandTransportReady ||
       !snapshot || snapshot.handId!==entry.handId || snapshot.version!==version ||
       version<Math.max(entry.minimumVersion,this.requiredHandVersion)){
      this.failSync(entry,new Error('authoritative hand sync not confirmed'))
      return
    }
    clearTimeout(entry.timer)
    this.pendingSync=null
    if(this.hand.status==='synchronizing')this.hand.setStatus('connected')
    entry.resolve()
  }
  private failSync(entry:PendingSync,error:Error):void {
    if(this.pendingSync!==entry)return
    clearTimeout(entry.timer)
    this.pendingSync=null
    entry.reject(error)
  }
  private rejectSync(reason:string):void {
    if(this.pendingSync)this.failSync(this.pendingSync,new Error(reason))
  }
  /** Transmit only actual Go dispatcher commands; timeout is an unknown outcome, never a proof of failure. */
  async submitCommand(input: Omit<WireCommand, 'v'>, exactUnknownRetry = false): Promise<WireCommandResult> {
    if (this.disposed || !this.connection || !this.transportReady) throw new Error('game connection not ready')
    const request = createWireCommand(input)
    if (request.type.startsWith('hand.')) {
      if (exactUnknownRetry) {
        // A previously sent command may have committed before a lost reply.
        // A stale local version MUST NOT block the same-ID verification.
        // Ownership, seat, command fingerprint and accepted result are
        // still checked by the Go application and MySQL receipt journal.
        if (this.hand.snapshot?.handId !== request.aggregateId)
          throw new Error('reconnected hand identity does not match pending command')
      } else if (!this.hand.canSubmit || this.hand.snapshot?.handId !== request.aggregateId ||
                 this.hand.snapshot.version !== request.expectedVersion) {
        throw new Error('hand not eligible for command')
      }
    }
    if (this.pendingCommands.has(request.requestId)) throw new Error('command already pending')
    if (this.pendingCommands.size >= MAX_PENDING_COMMANDS) throw new Error('pending command capacity exceeded')
    return new Promise<WireCommandResult>((resolve, reject) => {
      const generation = this.generation
      const connection = this.connection!
      const entry = {request, resolve, reject, timer: undefined as unknown as ReturnType<typeof setTimeout>}
      entry.timer = setTimeout(() => {
        if (this.pendingCommands.get(request.requestId) !== entry) return
        this.pendingCommands.delete(request.requestId)
        reject(new Error('command timeout; outcome unknown, retry with same requestId'))
      }, 15000)
      this.pendingCommands.set(request.requestId, entry)
      void connection.send({id: request.requestId, type: 'doudizhu.command', payload: request}).catch(() => {
        if (this.generation !== generation || this.pendingCommands.get(request.requestId) !== entry) return
        clearTimeout(entry.timer)
        this.pendingCommands.delete(request.requestId)
        reject(new Error('command send failed; outcome unknown'))
      })
    })
  }
  private rejectPending(reason: string): void {
    for (const pending of this.pendingCommands.values()) {
      clearTimeout(pending.timer)
      pending.reject(new Error(reason))
    }
    this.pendingCommands.clear()
  }
  dispose(): void {
    if (this.disposed) return
    this.disposed = true
    this.generation++
    this.transportReady = false
    this.rejectSync('client disposed during hand sync')
    this.rejectPending('client disposed; command outcome unknown')
    this.connection?.close()
    this.events.clear()
    this.hand.reset()
  }
}
