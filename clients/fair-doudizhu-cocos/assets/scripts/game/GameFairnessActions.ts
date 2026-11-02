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
  if(!d.accepted)throw new Error('fairness request rejected: '+String((d.failure as {code?:unknown}|undefined)?.code??'unknown'))
}
type PendingSecret = {handId:string;seat:number;random:Uint8Array;phrase:string;commitment:string}
export class GameFairnessActions {
  private readonly ids
  private readonly sealer
  private readonly manifestVersions=new InMemoryManifestVersionStore()
  private secret:PendingSecret|null=null
  private inFlight=false
  private pending:Omit<WireCommand,'v'>|null=null
  constructor(private readonly api:DoudizhuHTTPClient,platform:GamePlatform,
    private readonly trustedManifestRoots:Readonly<Record<string,string>>) {
    this.ids=secureRoomCommandIdentity(platform.random)
    this.sealer=new SecureEnvelopeSealer({randomSource:platform.random})
  }
  get needsReconciliation():boolean{return this.pending!==null}
  /** A network timeout must be retried with exactly the same signed request and payload. */
  async retryUnknown():Promise<WireCommandResult> {
    if(!this.pending || this.inFlight)throw new Error('no pending fairness request')
    this.inFlight=true
    try {
      const result=await this.api.command(this.pending)
      assertDurable(result,this.pending.type)
      this.pending=null
      if(result.type==='hand.reveal')this.clearSecret()
      return result
    }finally{this.inFlight=false}
  }
  private assertAvailable():void{
    if(this.inFlight)throw new Error('fairness action in progress')
    if(this.pending)throw new Error('previous fairness result unknown; retry exact request first')
  }
  async commit(handId:string,phrase?:string):Promise<WireCommandResult> {
    this.assertAvailable()
    this.inFlight=true
    try {
      const view=await this.api.fairness(handId)
      if(view.phase!=='FAIRNESS_COMMITTING' || view.committed)throw new Error('fairness commit phase unavailable')
      if(this.secret && this.secret.handId!==handId)throw new Error('unrevealed contribution for a different hand')
      const random=await this.sealerRandom(32)
      const chosenPhrase=phrase??encodeBase64Url(await this.sealerRandom(24))
      const commitment=computeCommitment(handId,view.seat,random,chosenPhrase)
      this.clearSecret()
      this.secret={handId,seat:view.seat,random,phrase:chosenPhrase,commitment}
      const command:Omit<WireCommand,'v'>={
        requestId:await this.ids.newRequestId(),type:'hand.commit',aggregateId:handId,
        clientSeq:view.lastClientSequence+1,expectedVersion:view.version,payload:{commitment}
      }
      this.pending=command
      const result=await this.api.command(command)
      assertDurable(result,'hand.commit')
      this.pending=null
      return result
    }finally{this.inFlight=false}
  }
  async reveal(handId:string):Promise<WireCommandResult> {
    this.assertAvailable()
    this.inFlight=true
    try{
      const secret=this.secret
      if(!secret || secret.handId!==handId)throw new Error('local committed fairness secret unavailable')
      const view=await this.api.fairness(handId)
      if(view.phase!=='FAIRNESS_REVEALING' || !view.committed || view.revealed ||
        view.seat!==secret.seat || view.commitment!==secret.commitment)
        throw new Error('server commitment does not match local fairness secret')
      const requestId=await this.ids.newRequestId()
      const clientSeq=view.lastClientSequence+1
      const context=await this.api.revealContext(handId,{requestId,clientSeq,expectedVersion:view.version})
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
      const manifest=verifyRevealKeyManifest(await this.api.revealManifest(claims.keyId),{
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
      const command:Omit<WireCommand,'v'>={
        requestId,type:'hand.reveal',aggregateId:handId,clientSeq,
        expectedVersion:view.version,
        revealContext:context.ticket,
        payload:{envelope:{v:envelope.version,keyId:envelope.keyId,
          suite:envelope.suite,encapsulatedKey:envelope.encapsulatedKey,ciphertext:envelope.ciphertext}}
      }
      this.pending=command
      const result=await this.api.command(command)
      assertDurable(result,'hand.reveal')
      this.pending=null
      this.clearSecret()
      return result
    }finally{this.inFlight=false}
  }
  private async sealerRandom(n:number):Promise<Uint8Array>{
    const v=await this.platformRandom.bytes(n)
    if(!(v instanceof Uint8Array)||v.length!==n)throw new Error('secure random unavailable')
    return v
  }
  private get platformRandom():GamePlatform['random']{return this.random}
  private random:GamePlatform['random']=undefined as never
  private clearSecret():void {
    this.secret?.random.fill(0)
    this.secret=null
  }
  dispose():void{this.clearSecret();this.pending=null}
}
function bytesEqual(a:Uint8Array,b:Uint8Array):boolean{
 if(a.length!==b.length)return false
 let diff=0;for(let i=0;i<a.length;i++)diff|=a[i]^b[i]
 return diff===0
}
