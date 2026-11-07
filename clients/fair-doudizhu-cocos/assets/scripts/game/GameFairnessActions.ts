import type { GamePlatform } from '@awesome-zero-platform/game-client-core'
import {
  decodeBase64Url,encodeBase64Url,sha256Digest,
  SecureEnvelopeSealer,verifyRevealKeyManifest,InMemoryManifestVersionStore
} from '@awesome-zero-platform/secure-envelope'
import { secureRoomCommandIdentity } from '@awesome-zero-platform/doudizhu-client/room'
import type { DoudizhuHTTPClient, FairnessRead, RevealContextRead } from '@awesome-zero-platform/doudizhu-client/api'
import type { WireCommand, WireCommandResult } from '@awesome-zero-platform/doudizhu-client/wire'

const text = new TextEncoder()
const contributionDomain='fair-doudizhu/contribution/v1'
const commitDomain='fair-doudizhu/client-commit/v1'
const ticketVersion='doudizhu-reveal-context-v1'
const kitVersion='doudizhu-local-fairness-recovery-v1'
function hex(value: Uint8Array): string {return Array.from(value,b=>b.toString(16).padStart(2,'0')).join('')}
function encodeLength(value:Uint8Array):Uint8Array {
  const b=new Uint8Array(4+value.length)
  new DataView(b.buffer).setUint32(0,value.length)
  b.set(value,4)
  return b
}
function concat(...parts:Uint8Array[]):Uint8Array {
  const b=new Uint8Array(parts.reduce((sum,part)=>sum+part.length,0))
  let i=0;for(const part of parts){b.set(part,i);i+=part.length}
  return b
}
function n64(value:number):Uint8Array {
  if(!Number.isSafeInteger(value)||value<0)throw new Error('invalid signed number')
  const out=new Uint8Array(8)
  new DataView(out.buffer).setBigUint64(0,BigInt(value))
  return out
}
function computeContribution(handId:string,seat:number,random:Uint8Array,phrase:string):Uint8Array {
  if(random.length!==32 || !Number.isInteger(seat)||seat<1||seat>3 ||
    !phrase || !phrase.normalize('NFKC')) throw new Error('invalid fairness contribution')
  const input=concat(text.encode(contributionDomain),new Uint8Array([0]),encodeLength(text.encode(handId)),
    new Uint8Array([seat]),random,sha256Digest(text.encode(phrase.normalize('NFKC'))))
  return sha256Digest(input)
}
export function computeCommitment(handId:string,seat:number,random:Uint8Array,phrase:string):string {
  const contribution=computeContribution(handId,seat,random,phrase)
  return hex(sha256Digest(concat(text.encode(commitDomain),new Uint8Array([0]),
    encodeLength(text.encode(handId)),new Uint8Array([seat]),contribution)))
}
/** Mirror application.BuildRevealAAD exactly; refuse any mismatch from the server. */
export function canonicalRevealAAD(claims:RevealContextRead['ticket']['claims']):Uint8Array {
  if(claims.v!==ticketVersion || !/^[0-9a-f]{64}$/.test(claims.commitment))throw new Error('invalid signed AAD claims')
  const fromHex=(value:string):Uint8Array=>Uint8Array.from(value.match(/../g)!.map(pair=>parseInt(pair,16)))
  const strings=[
    'fair-doudizhu/reveal-aad/v1','fair-doudizhu-command-v1',
    'doudizhu.hand.fairness.reveal.submit.v1',claims.keyId,claims.requestId,
    'hand',claims.handId,claims.actor
  ]
  return concat(...strings.map(v=>encodeLength(text.encode(v))),
    new Uint8Array([claims.seat]),n64(claims.clientSeq),n64(claims.expectedVersion),
    fromHex(claims.commitment),n64(claims.issuedAt),n64(claims.expiresAt))
}
function assertDurable(result:WireCommandResult,expectedType:string):void {
  const d=result.durable
  if(!d || d.v!=='fair-doudizhu-command-result-v1' ||
    d.commandId!==result.requestId || result.type!==expectedType ||
    typeof d.accepted!=='boolean')throw new Error('unverifiable durable fairness result; outcome unknown')
  if(!d.accepted) {
    const failure=d.failure as {code?:unknown}|undefined
    if(!failure || typeof failure.code!=='string' || !failure.code)
      throw new Error('unverifiable fairness rejection; outcome unknown')
    throw new DurableFairnessRejected(failure.code)
  }
}
/** Mirror application.BuildCustodiedAAD exactly. No prior commitment exists:
 * the authenticated one-shot command is committed with its encrypted record. */
export function canonicalCustodiedAAD(claims:RevealContextRead['ticket']['claims']):Uint8Array{
 if(claims.v!=='doudizhu-custodied-context-v1' || claims.commitment!=='')
  throw new Error('invalid signed custodial claims')
 const strings=[
  'fair-doudizhu/custodied-contribution-aad/v1','fair-doudizhu-command-v1',
  'doudizhu.hand.fairness.contribution.submit.v1',claims.keyId,
  claims.requestId,'hand',claims.handId,claims.actor
 ]
 return concat(...strings.map(v=>encodeLength(text.encode(v))),
  new Uint8Array([claims.seat]),n64(claims.clientSeq),n64(claims.expectedVersion),
  n64(claims.issuedAt),n64(claims.expiresAt))
}

/** The user re-enters only their own phrase, while the authenticated server
 * supplies the once-protected random bytes after final archive verification. */
export function verifyOwnContribution(handId:string,seat:number,
 encodedRandom:string,phrase:string,commitment:string):boolean{
 const random=decodeBase64Url(encodedRandom)
 try {
  if(random.length!==32)throw new Error('invalid disclosed contribution random')
  return computeCommitment(handId,seat,random,phrase)===commitment
 }finally{random.fill(0)}
}
export class DurableFairnessRejected extends Error {
  constructor(readonly code:string){super('fairness request rejected: '+code)}
}
type PendingSecret = {handId:string;seat:number;random:Uint8Array;phrase:string;commitment:string}
export class GameFairnessActions {
  private readonly ids
  private readonly sealer
  private readonly manifestVersions=new InMemoryManifestVersionStore()
  private secret:PendingSecret|null=null
  private prepared=false
  private inFlight=false
  // Synchronous lifecycle reservation prevents a new fairness operation from
  // racing the asynchronous room.hand.start command after preflight.
  private handTransition=false
  private disposed=false
  private generation=0
  private pending:Omit<WireCommand,'v'>|null=null
  constructor(private readonly api:DoudizhuHTTPClient,private readonly platform:GamePlatform,
    private readonly trustedManifestRoots:Readonly<Record<string,string>>) {
    this.ids=secureRoomCommandIdentity(platform.random)
    this.sealer=new SecureEnvelopeSealer({randomSource:platform.random})
  }
  private ensureActive(epoch=this.generation):void {
    if(this.disposed || this.generation!==epoch)throw new Error('fairness actions disposed')
  }
  private settle(result:WireCommandResult,expectedType:string):void{
    try{assertDurable(result,expectedType)}
    catch(error){
      if(error instanceof DurableFairnessRejected)this.pending=null
      throw error
    }
    this.pending=null
  }
  get needsReconciliation():boolean{return this.pending!==null}
  get hasRecoverySecret():boolean{return this.secret!==null}
  get recoveryKitPrepared():boolean{return this.prepared}

  /** Reserve an atomic transition from the old fairness session to a new hand.
   * A pending exact-ID command must be reconciled first, not silently erased.
   * The caller MUST release this reservation in a finally block. */
  beginHandTransition():()=>void{
    this.ensureActive()
    if(this.inFlight || this.handTransition)
      throw new Error('fairness action in progress; cannot start another hand')
    if(this.pending)
      throw new Error('fairness result unknown; retry the original request before changing hand')
    this.handTransition=true
    return ()=>{this.handTransition=false}
  }

  /** Recovery material is never written into platform-local plaintext storage.
   * The user must explicitly save this high-entropy secret OUTSIDE the game
   * before sending the corresponding irreversible public commitment. */
  exportRecoveryKit():string {
    this.ensureActive()
    if(!this.secret)throw new Error('fairness recovery secret unavailable')
    const secret=this.secret
    const raw=JSON.stringify({v:kitVersion,handId:secret.handId,seat:secret.seat,
      random:encodeBase64Url(secret.random),phrase:secret.phrase,commitment:secret.commitment})
    if(raw.length>1024)throw new Error('fairness recovery kit exceeds size bound')
    return encodeBase64Url(text.encode(raw))
  }

  async prepareRecoveryKit(handId:string,phrase?:string):Promise<string>{
    this.assertAvailable()
    this.inFlight=true
    try{
      const epoch=this.generation
      const view=await this.api.fairness(handId)
      this.ensureActive(epoch)
      if(view.phase!=='FAIRNESS_COMMITTING'||view.committed)
        throw new Error('fairness preparation phase unavailable')
      if(this.secret && this.secret.handId!==handId)
        throw new Error('unrevealed contribution for a different hand')
      if(!this.secret){
        const random=await this.sealerRandom(32)
        try {
          const chosenPhrase=phrase??encodeBase64Url(await this.sealerRandom(24))
          this.ensureActive(epoch)
          const commitment=computeCommitment(handId,view.seat,random,chosenPhrase)
          this.secret={handId,seat:view.seat,random,phrase:chosenPhrase,commitment}
        } finally {if(this.secret?.random!==random)random.fill(0)}
      }
      if(this.secret.seat!==view.seat)throw new Error('fairness seat changed')
      this.prepared=true
      return this.exportRecoveryKit()
    }finally{this.inFlight=false}
  }

  /** Import requires an authenticated current server projection and an exact
   * commitment match; changing seat/hand or a forged kit fails closed. */
  async importRecoveryKit(encoded:string,expectedHandId:string):Promise<void>{
    this.assertAvailable()
    if(typeof encoded!=='string'||encoded.length<48||encoded.length>2048)
      throw new Error('invalid fairness recovery kit')
    let raw:unknown
    try{raw=JSON.parse(new TextDecoder().decode(decodeBase64Url(encoded)))}catch{
      throw new Error('invalid fairness recovery kit')
    }
    if(!raw||typeof raw!=='object'||Array.isArray(raw))
      throw new Error('invalid fairness recovery kit')
    const kit=raw as Record<string,unknown>
    if(kit.v!==kitVersion||kit.handId!==expectedHandId||
      typeof kit.handId!=='string'||!/^[A-Za-z0-9_-]{1,128}$/.test(kit.handId)||
      !Number.isInteger(kit.seat)||Number(kit.seat)<1||Number(kit.seat)>3||
      typeof kit.random!=='string'||typeof kit.phrase!=='string'||
      kit.phrase.length<1||kit.phrase.length>256||
      typeof kit.commitment!=='string'||!/^[0-9a-f]{64}$/.test(kit.commitment))
      throw new Error('invalid fairness recovery kit')
    let random:Uint8Array
    try{random=decodeBase64Url(kit.random)}catch{throw new Error('invalid fairness recovery kit')}
    if(random.length!==32){random.fill(0);throw new Error('invalid fairness recovery kit')}
    this.inFlight=true
    try{
      const commitment=computeCommitment(expectedHandId,Number(kit.seat),random,kit.phrase)
      if(commitment!==kit.commitment)throw new Error('fairness recovery digest mismatch')
      const epoch=this.generation
      const view=await this.api.fairness(expectedHandId)
      this.ensureActive(epoch)
      if(view.seat!==kit.seat||view.revealed||
        (view.committed ? view.commitment!==commitment :
          view.phase!=='FAIRNESS_COMMITTING')||
        !['FAIRNESS_COMMITTING','FAIRNESS_REVEALING'].includes(view.phase))
        throw new Error('recovery kit does not match authenticated server commitment')
      this.clearSecret()
      this.secret={handId:expectedHandId,seat:Number(kit.seat),random,
        phrase:kit.phrase,commitment}
      this.prepared=true
    }catch(error){random.fill(0);throw error}
    finally{this.inFlight=false}
  }
  /** New mode: player contributes a string and fresh CSPRNG random together.
   * No player phrase, random or external recovery kit is retained or written
   * into local persistent storage. Only the already-encrypted exact-ID request
   * may remain in memory until the original outcome is reconciled. */
  async submitCustodied(handId:string,phrase:string):Promise<WireCommandResult>{
    this.assertAvailable()
    if(this.secret)throw new Error('historical fairness secret still active')
    if(typeof phrase!=='string' || !phrase.normalize('NFKC').trim() ||
      text.encode(phrase).length>256)throw new Error('请输入 1–256 字节的个人随机验证字符串')
    this.inFlight=true
    let random:Uint8Array|null=null
    try{
      const epoch=this.generation
      const view=await this.api.fairness(handId)
      this.ensureActive(epoch)
      if(view.fairnessMode!=='client-contributed-server-custodied-v1' ||
        view.phase!=='FAIRNESS_COMMITTING' || view.committed || view.revealed)
        throw new Error('server does not accept a new player contribution')
      random=await this.sealerRandom(32)
      this.ensureActive(epoch)
      const requestId=await this.ids.newRequestId()
      this.ensureActive(epoch)
      const clientSeq=view.lastClientSequence+1
      const context=await this.api.revealContext(handId,{
        requestId,clientSeq,expectedVersion:view.version
      },'custodied')
      this.ensureActive(epoch)
      const claims=context.ticket.claims
      if(claims.v!=='doudizhu-custodied-context-v1' || claims.commitment!=='' ||
        claims.seat!==view.seat || claims.keyId!==view.revealKeyId ||
        claims.publicKeySha256!==view.revealPublicKeySha256 ||
        Date.now()>=claims.expiresAt || claims.expiresAt-claims.issuedAt>120000)
        throw new Error('custodied signed context does not match hand')
      const localAAD=canonicalCustodiedAAD(claims)
      const receivedAAD=decodeBase64Url(context.aad)
      if(!bytesEqual(localAAD,receivedAAD)){
        localAAD.fill(0);receivedAAD.fill(0)
        throw new Error('signed custodied AAD mismatch')
      }
      if(!Object.keys(this.trustedManifestRoots).length)
        throw new Error('trusted reveal signing roots not configured')
      const signedDocument=await this.api.revealManifest(claims.keyId)
      this.ensureActive(epoch)
      const manifest=verifyRevealKeyManifest(signedDocument,{
        roots:this.trustedManifestRoots,versions:this.manifestVersions,mode:'pinned',
        expectedKeyId:claims.keyId,
        expectedPublicKeySha256:encodeBase64Url(
          Uint8Array.from(view.revealPublicKeySha256.match(/../g)!.map(v=>parseInt(v,16)))),
        boundAt:new Date(view.revealKeyBoundAt)
      })
      const payload={
        v:'fair-doudizhu-reveal-v1',handId,seat:view.seat,
        secureRandom:encodeBase64Url(random),phrase,normalization:'NFKC-v1'
      }
      const plaintext=text.encode(JSON.stringify(payload))
      let envelope
      try{
        envelope=await this.sealer.seal({
          recipient:{keyId:claims.keyId,publicKey:manifest.publicKeyBytes},
          plaintext,aad:receivedAAD
        })
      }finally{
        plaintext.fill(0);localAAD.fill(0);receivedAAD.fill(0)
      }
      this.ensureActive(epoch)
      const command:Omit<WireCommand,'v'>={
        requestId,type:'hand.contribution',aggregateId:handId,
        clientSeq,expectedVersion:view.version,revealContext:context.ticket,
        payload:{envelope:{
          v:envelope.version,keyId:envelope.keyId,suite:envelope.suite,
          encapsulatedKey:envelope.encapsulatedKey,ciphertext:envelope.ciphertext
        }}
      }
      this.pending=command
      const result=await this.api.command(command)
      this.ensureActive(epoch)
      this.settle(result,'hand.contribution')
      return result
    }finally{
      random?.fill(0)
      this.inFlight=false
    }
  }

  /** A network timeout must be retried with exactly the same signed request and payload. */
  async retryUnknown():Promise<WireCommandResult> {
    if(!this.pending || this.inFlight)throw new Error('no pending fairness request')
    this.inFlight=true
    try {
      const epoch=this.generation
      const pending=this.pending
      const result=await this.api.command(pending)
      this.ensureActive(epoch)
      this.settle(result,pending.type)
      if(result.type==='hand.reveal')this.clearSecret()
      return result
    }finally{this.inFlight=false}
  }
  private assertAvailable():void{
    this.ensureActive()
    if(this.inFlight || this.handTransition)throw new Error('fairness action in progress')
    if(this.pending)throw new Error('previous fairness result unknown; retry exact request first')
  }
  async commit(handId:string,phrase?:string):Promise<WireCommandResult> {
    this.assertAvailable()
    this.inFlight=true
    try {
      const epoch=this.generation
      const view=await this.api.fairness(handId)
      this.ensureActive(epoch)
      if(view.phase!=='FAIRNESS_COMMITTING' || view.committed)throw new Error('fairness commit phase unavailable')
      if(this.secret && this.secret.handId!==handId)throw new Error('unrevealed contribution for a different hand')
      if(!this.secret){
        const random=await this.sealerRandom(32)
        try {
          const chosenPhrase=phrase??encodeBase64Url(await this.sealerRandom(24))
          this.ensureActive(epoch)
          const commitment=computeCommitment(handId,view.seat,random,chosenPhrase)
          this.secret={handId,seat:view.seat,random,phrase:chosenPhrase,commitment}
        } finally {if(this.secret?.random!==random)random.fill(0)}
      }
      if(this.secret.seat!==view.seat ||
        (phrase!==undefined && phrase!==this.secret.phrase))
        throw new Error('prepared contribution does not match authenticated seat')
      const commitment=this.secret.commitment
      const command:Omit<WireCommand,'v'>={
        requestId:await this.ids.newRequestId(),type:'hand.commit',aggregateId:handId,
        clientSeq:view.lastClientSequence+1,expectedVersion:view.version,payload:{commitment}
      }
      this.ensureActive(epoch)
      this.pending=command
      const result=await this.api.command(command)
      this.ensureActive(epoch)
      this.settle(result,'hand.commit')
      return result
    }finally{this.inFlight=false}
  }
  async reveal(handId:string):Promise<WireCommandResult> {
    this.assertAvailable()
    this.inFlight=true
    try{
      const secret=this.secret
      if(!secret || secret.handId!==handId)throw new Error('local committed fairness secret unavailable')
      const epoch=this.generation
      const view=await this.api.fairness(handId)
      this.ensureActive(epoch)
      if(view.phase!=='FAIRNESS_REVEALING' || !view.committed || view.revealed ||
        view.seat!==secret.seat || view.commitment!==secret.commitment)
        throw new Error('server commitment does not match local fairness secret')
      const requestId=await this.ids.newRequestId()
      this.ensureActive(epoch)
      const clientSeq=view.lastClientSequence+1
      const context=await this.api.revealContext(handId,{requestId,clientSeq,expectedVersion:view.version})
      this.ensureActive(epoch)
      const claims=context.ticket.claims
      if(claims.seat!==secret.seat || claims.commitment!==secret.commitment ||
        claims.keyId!==view.revealKeyId || claims.publicKeySha256!==view.revealPublicKeySha256 ||
        Date.now()>=claims.expiresAt || claims.expiresAt-claims.issuedAt>120000)
        throw new Error('server reveal context does not match committed seat')
      const computedAAD=canonicalRevealAAD(claims)
      const aad=decodeBase64Url(context.aad)
      if(!bytesEqual(aad,computedAAD))throw new Error('signed reveal AAD mismatch')
      if(!Object.keys(this.trustedManifestRoots).length)
        throw new Error('trusted reveal key signing roots not configured')
      const document=await this.api.revealManifest(claims.keyId)
      this.ensureActive(epoch)
      const manifest=verifyRevealKeyManifest(document,{
        roots:this.trustedManifestRoots,versions:this.manifestVersions,mode:'pinned',
        expectedKeyId:claims.keyId,expectedPublicKeySha256:encodeBase64Url(
          Uint8Array.from(view.revealPublicKeySha256.match(/../g)!.map(v=>parseInt(v,16)))),
        boundAt:new Date(view.revealKeyBoundAt)
      })
      const payload={
        v:'fair-doudizhu-reveal-v1',handId,seat:secret.seat,
        secureRandom:encodeBase64Url(secret.random),phrase:secret.phrase,normalization:'NFKC-v1'
      }
      const bytes=text.encode(JSON.stringify(payload))
      let envelope
      try{envelope=await this.sealer.seal({recipient:{keyId:claims.keyId,
        publicKey:manifest.publicKeyBytes},plaintext:bytes,aad})}
      finally{bytes.fill(0);aad.fill(0);computedAAD.fill(0)}
      this.ensureActive(epoch)
      const command:Omit<WireCommand,'v'>={
        requestId,type:'hand.reveal',aggregateId:handId,clientSeq,
        expectedVersion:view.version,
        revealContext:context.ticket,
        payload:{envelope:{v:envelope.version,keyId:envelope.keyId,
          suite:envelope.suite,encapsulatedKey:envelope.encapsulatedKey,ciphertext:envelope.ciphertext}}
      }
      this.pending=command
      const result=await this.api.command(command)
      this.ensureActive(epoch)
      this.settle(result,'hand.reveal')
      this.clearSecret()
      return result
    }finally{this.inFlight=false}
  }
  private async sealerRandom(n:number):Promise<Uint8Array>{
    const v=await this.platform.random.bytes(n)
    if(!(v instanceof Uint8Array)||v.length!==n)throw new Error('secure random unavailable')
    if(this.disposed){v.fill(0);throw new Error('fairness actions disposed')}
    return v
  }
  private clearSecret():void {
    this.secret?.random.fill(0)
    this.secret=null
    this.prepared=false
  }
  dispose():void{
    this.disposed=true
    this.handTransition=false
    this.generation++
    this.clearSecret()
    this.pending=null
  }
}
function bytesEqual(a:Uint8Array,b:Uint8Array):boolean{
 if(a.length!==b.length)return false
 let diff=0;for(let i=0;i<a.length;i++)diff|=a[i]^b[i]
 return diff===0
}
