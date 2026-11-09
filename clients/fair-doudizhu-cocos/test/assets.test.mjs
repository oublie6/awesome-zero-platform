import {test} from 'node:test'
import assert from 'node:assert/strict'
import {readFile,readdir} from 'node:fs/promises'
const json=async path=>JSON.parse(await readFile(new URL(path,import.meta.url),'utf8'))
test('entry scene has valid references, Canvas camera and stable bootstrap script UUID',async()=>{
 const scene=await json('../assets/scenes/Entry.scene'),meta=await json('../assets/scenes/Entry.scene.meta')
 assert.equal(scene[0].__type__,'cc.SceneAsset');assert.equal(scene[1]._id,meta.uuid)
 const check=value=>{if(value&&typeof value==='object'){if('__id__' in value)assert.ok(scene[value.__id__]);Object.values(value).forEach(check)}};check(scene)
 const canvas=scene.find(x=>x.__type__==='cc.Canvas');assert.equal(scene[canvas._cameraComponent.__id__].__type__,'cc.Camera')
 const hex=(await json('../assets/scripts/bootstrap/GameBootstrap.ts.meta')).uuid.replaceAll('-','')
 const b64='ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/'
 let compressed=hex.slice(0,5);for(let i=5;i<32;i+=3){const n=parseInt(hex.slice(i,i+3),16);compressed+=b64[n>>6]+b64[n&63]}
 const bootstrap=scene.find(x=>x.__type__===compressed);assert.ok(bootstrap);assert.equal(bootstrap.node.__id__,canvas.node.__id__)
})
test('every source asset and directory has a unique stable meta UUID',async()=>{
 const uuids=new Set()
 async function visit(url){for(const entry of await readdir(url,{withFileTypes:true})){
  if(entry.name.endsWith('.meta'))continue
  const path=new URL(entry.name+(entry.isDirectory()?'/':''),url)
  const meta=JSON.parse(await readFile(new URL(entry.name+'.meta',url),'utf8'))
  assert.equal(uuids.has(meta.uuid),false);uuids.add(meta.uuid)
  if(entry.isDirectory())await visit(path)
 }}
 await visit(new URL('../assets/',import.meta.url))
})

test('Creator 3.8 card selections use Set.forEach instead of downlevel-unsafe iteration',async()=>{
 const source=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 assert.doesNotMatch(source,/\[\s*\.\.\.\s*this\.selectedCards\s*\]/)
 assert.doesNotMatch(source,/for\s*\(\s*const\s+\w+\s+of\s+this\.selectedCards\s*\)/)
 assert.match(source,/this\.selectedCards\.forEach\(card\s*=>\s*selectedCards\.push\(card\)\)/)
 assert.match(source,/actions\.play\(selectedCards\)/)
})

test('native UI exposes a room leave control and displays verified archive completion',async()=>{
 const source=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 assert.match(source,/name:'LeaveRoom'/)
 assert.match(source,/online\.leaveRoom\(\)/)
 assert.match(source,/terminalEvidence/)
 assert.match(source,/roomGeneration/)
 assert.match(source,/adoptAuthorizedRoomRefresh/)
 assert.match(source,/api\.finalEvidence\(handID\)/)
})

test('native fairness backup gate requires explicit user-held re-entry before commitment',async()=>{
 const source=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 assert.match(source,/name:'VerifyFairnessBackup'/)
 assert.match(source,/beginFairnessBackupVerification\(\)/)
 assert.match(source,/this\.roomInput\.string=''/)
 assert.match(source,/backupVerificationPendingHand!==handID/)
 assert.match(source,/this\.verifiedBackupController=online/)
 assert.match(source,/this\.verifiedBackupHand!==handID/)
 assert.match(source,/this\.verifiedBackupController!==online/)
 assert.match(source,/await fairness\.importRecoveryKit\(kit,handID\)/)
 assert.match(source,/fairness!==online\.fairness/)
 assert.match(source,/await fairness\.commit\(handID\)/)
})

test('delayed fairness kit UI updates are fenced after leave, hand and login replacement',async()=>{
 const source=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 const prepare=source.split('private async prepareFairness():Promise<void>{')[1].split('private beginFairnessBackupVerification()')[0]
 assert.match(prepare,/const roomID=this\.roomID/)
 assert.match(prepare,/const handID=this\.handID/)
 assert.match(prepare,/const generation=this\.roomGeneration/)
 assert.match(prepare,/const fairness=online\?\.fairness/)
 assert.match(prepare,/await fairness\.prepareRecoveryKit\(handID\)/)
 assert.match(prepare,/generation!==this\.roomGeneration/)
 assert.match(prepare,/handID!==this\.handID/)
 assert.match(prepare,/roomID!==this\.roomID/)
 assert.match(prepare,/fairness!==online\.fairness/)
 assert.match(prepare,/input!==this\.roomInput/)
 assert.ok(prepare.indexOf('generation!==this.roomGeneration')<prepare.indexOf('input.string=kit'))
 assert.match(source,/await online\.startHand\(handID\)/)
 assert.match(source,/generation!==this\.roomGeneration/)
 assert.match(source,/online\.adoptAuthorizedRoomRefresh\(info,commandGeneration\)/)
})

test('accepted room.hand.start receipt is adopted only after exact fairness preflight',async()=>{
 const source=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 const start=source.split('private async startHand():Promise<void> {')[1].split('private async refreshRoom():Promise<void>')[0]
 assert.match(start,/const generation=this\.roomGeneration/)
 assert.match(start,/await online\.startHand\(handID\)/)
 assert.match(start,/this\.currentRoom\(online,epoch,roomID,generation,selection\)/)
 assert.ok(start.indexOf('await online.startHand(handID)')<start.indexOf('this.handID=handID'))
 assert.doesNotMatch(start,/await online\.rooms\.startHand\(handID\)/)
 assert.doesNotMatch(start,/online\.confirmActiveFairnessHand\(handID\)/)
})

test('Round16 HTTP H2 refresh defers rebinding until durable WSS start receipt',async()=>{
 const ui=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 const controller=await readFile(new URL('../assets/scripts/game/GameOnlineController.ts',import.meta.url),'utf8')
 const refresh=ui.split('private async refreshRoom():Promise<void> {')[1].split('private async refreshFairness():Promise<void>')[0]
 assert.match(refresh,/if\(online\.handStartInProgress\)return/)
 assert.ok(refresh.indexOf('if(online.handStartInProgress)return')<refresh.indexOf('online.adoptAuthorizedRoomRefresh(info,commandGeneration)'))
 assert.doesNotMatch(refresh,/online\.rooms\.restoreTrustedRoom\(/)
 const guard=controller.split('confirmActiveFairnessHand(handId: string | null): void {')[1].split('private resetFairnessAfterConfirmedTransition()')[0]
 assert.match(guard,/this\.startingHand !== null && handId !== this\.confirmedFairnessHand/)
 assert.doesNotMatch(guard,/handId !== this\.startingHand/)
 const start=controller.split('async startHand(handId: string): Promise<Readonly<RoomState>> {')[1].split('async createRoom()')[0]
 assert.ok(start.indexOf('this.startingHand = null\n            this.confirmActiveFairnessHand(handId)') > start.indexOf('await this.rooms.startHand(handId)'))
})

test('new hands use player text and server custody without game-local backup',async()=>{
 const ui=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 const actions=await readFile(new URL('../assets/scripts/game/GameFairnessActions.ts',import.meta.url),'utf8')
 const online=await readFile(new URL('../assets/scripts/game/GameOnlineController.ts',import.meta.url),'utf8')
 assert.match(ui,/name:'SubmitOwnContribution'/)
 assert.match(ui,/name:'VerifyOwnContribution'/)
 assert.match(ui,/if\(this\.fairnessMode==='client-contributed-server-custodied-v1'\)/)
 assert.match(ui,/if\(this\.roomInput\)this\.roomInput\.string=''/)
 assert.match(ui,/await fairness\.submitCustodied\(handID,phrase\)/)
 assert.match(ui,/verifyOwnContribution\(handID,own\.seat,own\.secureRandom,phrase,own\.commitment\)/)
 assert.match(online,/this\.rooms\.startHand\(handId,'client-contributed-server-custodied-v1'\)/)
 assert.match(actions,/async submitCustodied\(handId:string,phrase:string\)/)
 assert.match(actions,/random\?\.fill\(0\)/)
 assert.doesNotMatch(actions.split('async submitCustodied(')[1].split('async retryUnknown(')[0],/localStorage|wx\.setStorage|exportRecoveryKit/)
})

test('R18 recovery UI gates unknown H2 mode and discovers rooms on fresh login',async()=>{
 const bootstrap=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 const core=await readFile(new URL('../assets/scripts/game/GameFairnessActions.ts',import.meta.url),'utf8')
 assert.match(bootstrap,/private fairnessMode:'client-contributed-server-custodied-v1'\|''\|null=null/)
 assert.match(bootstrap,/else if\(this\.handID && this\.fairnessMode===''/)
 assert.ok(bootstrap.indexOf("this.fairnessMode=null\n        this.selectedCards.clear()")<
    bootstrap.indexOf("if(this.screen==='rooms')this.renderScreen('rooms')",bootstrap.indexOf('private async refreshRoom')))
 assert.match(bootstrap,/await this\.recoverMyRooms\(token=>\{selection=token\}\)/)
 assert.match(bootstrap,/online\.discoverRecoverableRooms\(\)/)
 assert.match(bootstrap,/online\.reconnectAndResync\(roomID\)/)
 for(const op of ['prepareRecoveryKit','importRecoveryKit','commit','reveal']){
  assert.match(core,new RegExp('legacy (recovery|commit|reveal) forbidden for server-custodied hand'))
 }
})

test('Round19 latest recovery choice and final membership verification are mandatory',async()=>{
 const ui=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 const controller=await readFile(new URL('../assets/scripts/game/GameOnlineController.ts',import.meta.url),'utf8')
 const part=ui.split('private async performAuthorizedRoomRecovery(')[1].split('private async reconnect():')[0]
 assert.match(part,/token===this\.recoveryAttempt/)
 assert.match(part,/await online\.roomInfo\(roomID\)/)
 assert.match(part,/if\(!verified\.member \|\| verified\.status==='CLOSED'\)/)
 assert.match(part,/this\.revokeInvalidRecoveredRoom\(online,roomID\)/)
 assert.ok(part.indexOf('await online.roomInfo(roomID)')<part.indexOf("this.notice('已通过服务器恢复本人房间和牌局')"))
 assert.match(controller,/private recoveryGeneration = 0/)
 assert.match(controller,/const pending = this\.pendingRecovery/)
 assert.match(controller,/pending\?\.key === key/)
 assert.match(controller,/pending\.token === this\.recoveryGeneration/)
 assert.match(controller,/pending\.membership === membership && pending\.commands === commands/)
 assert.match(controller,/token===this\.recoveryGeneration/)
 assert.match(controller,/const fresh=await this\.api\.room\(initial\.roomId\)/)
 assert.ok(controller.indexOf('const fresh=await this.api.room(initial.roomId)') <
  controller.indexOf('this.adoptAuthorizedRoom(fresh)'))
 const adoption=controller.split('private adoptAuthorizedRoom(view: RoomRead): void {')[1]
  .split('/** A later authoritative CLOSED')[0]
 assert.match(adoption,/this\.rooms\.assertCanRestoreTrustedRoom\(view\.roomId\)/)
 assert.match(adoption,/this\.actions\.beginHandTransition\(\)/)
 assert.match(adoption,/this\.fairness\.beginHandTransition\(\)/)
 assert.ok(adoption.indexOf('this.rooms.assertCanRestoreTrustedRoom') <
  adoption.indexOf('this.rooms.restoreTrustedRoom'))
 assert.ok(adoption.indexOf('this.fairness.beginHandTransition') <
  adoption.indexOf('this.rooms.restoreTrustedRoom'))
})

test('G47-N01: scene join shares latest room selection token',async()=>{
 const source=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 const join=source.split('private async joinRoom():Promise<void> {')[1].split('private async leaveRoom():')[0]
 assert.ok(join.includes('const token=++this.recoveryAttempt'))
 assert.ok(join.includes('token!==this.recoveryAttempt'))
 assert.ok(join.indexOf('token!==this.recoveryAttempt')<join.indexOf('this.roomID=room.roomId'))
})

test('G48-N03: online join rejects stale results after logout',async()=>{
 const source=await readFile(new URL('../assets/scripts/game/GameOnlineController.ts',import.meta.url),'utf8')
 const join=source.split('async joinRoom(roomId: string):')[1].split('async reconcileRoom(')[0]
 assert.ok(join.includes('this.session.authenticated'))
 const logout=source.split('async logout(): Promise<void> {')[1].split('dispose(): void')[0]
 assert.ok(logout.indexOf('this.disposed = true')<logout.indexOf('await this.session.logout()'))
 assert.ok(logout.includes('++this.recoveryGeneration'))
})

test('G50-N02: scene offers exact room UNKNOWN reconciliation before discovery',async()=>{
 const ui=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 const controller=await readFile(new URL('../assets/scripts/game/GameOnlineController.ts',import.meta.url),'utf8')
 assert.match(ui,/name:'RetryUnknownRoom'/)
 const reconnect=ui.split('private async reconnect():Promise<void>{')[1].split('private async retryUnknownFairness():')[0]
 assert.ok(reconnect.indexOf('online.retryUnknownRoom()')<reconnect.indexOf('this.recoverMyRooms('))
 assert.match(controller,/return this\.rooms\.retryUnknown\(\)/)
})
test('audit: Scene refresh does not bypass command generation or preflight barriers',async()=>{
 const ui=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 const controller=await readFile(new URL('../assets/scripts/game/GameOnlineController.ts',import.meta.url),'utf8')
 const refresh=ui.split('private async refreshRoom():Promise<void> {')[1].split('private async refreshFairness():')[0]
 assert.match(refresh,/const commandGeneration=online\.rooms\.commandGeneration/)
 assert.match(refresh,/online\.adoptAuthorizedRoomRefresh\(info,commandGeneration\)/)
 assert.doesNotMatch(refresh,/online\.rooms\.restoreTrustedRoom\(/)
 const member=controller.split('async joinRoom(roomId: string):')[1].split('async retryUnknownRoom(')[0]
 assert.match(member,/const readCurrent/)
 assert.match(member,/commands === this\.rooms\.commandGeneration/)
})

test('audit: pending Scene refresh is fenced when newer room selection begins',async()=>{
 const source=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 const refresh=source.split('private async refreshRoom():Promise<void> {')[1].split('private async refreshFairness():')[0]
 assert.match(refresh,/const roomID=this\.roomID,selection=this\.recoveryAttempt/)
 assert.match(refresh,/selection!==this\.recoveryAttempt/)
 assert.match(refresh,/online\.rooms\.state\.roomId!==roomID/)
})

test('G51-N01: stale account reconciliation callbacks cannot replace another account notice',async()=>{
 const ui=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 const reconnect=ui.split('private async reconnect():Promise<void>{')[1].split('private async retryUnknownFairness():')[0]
 assert.match(reconnect,/const owner=this\.online,epoch=this\.requestGeneration/)
 assert.ok(reconnect.indexOf('if(!this.currentAccount(owner,epoch)') <
   reconnect.indexOf("this.notice('原房间操作已明确拒绝"))
 assert.match(reconnect,/if\(this\.currentAccount\(owner,epoch\) && ownedSelection===this\.recoveryAttempt\)\s*this\.notice\('重连失败：/)
 const login=ui.split('private async login(restoreOnly=false):Promise<void> {')[1].split('private async logoutWeb():Promise<void>{')[0]
 assert.match(login,/text=>\{if\(this\.currentAccount\(online,generation\)\)this\.notice\(text\)\}/)
 assert.match(login,/if\(this\.currentAccount\(online,generation\) && this\.screen==='table'\)/)
})
test('G51 audit: async room and hand callbacks require their original session and room',async()=>{
 const ui=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 const section=(a,b)=>ui.split(a)[1]?.split(b)[0]||''
 for(const [start,end] of [
  ['private async createRoom():Promise<void> {','private async joinRoom():'],
  ['private async joinRoom():Promise<void> {','private async leaveRoom():'],
  ['private async leaveRoom():Promise<void> {','private async readyRoom():'],
  ['private async readyRoom():Promise<void> {','private async startHand():'],
  ['private async startHand():Promise<void> {','private async refreshRoom():'],
  ['private async refreshRoom():Promise<void> {','private async refreshFairness():'],
  ['private async refreshFairness():Promise<void>{','private async submitOwnContribution():'],
  ['private async retryUnknownHand():Promise<void>{','private discardUnknownHand():'],
  ['private async bid(score:number):Promise<void> {','private async play():'],
  ['private async play():Promise<void> {','private async pass():'],
  ['private async pass():Promise<void> {','private requireOnline():'],
  ['private async importFairness():Promise<void>{','private async commitFairness():'],
  ['private async commitFairness():Promise<void>{','private async revealFairness():'],
  ['private async revealFairness():Promise<void>{','private async showEvidence():'],
  ['private async showEvidence():Promise<void>{','private async refreshHand():']
 ])assert.match(section(start,end),/this\.current(Account|Room)\(/,start+' must fence stale callbacks')
})
test('G51 audit: repeat create/leave taps and menu recovery errors are controlled',async()=>{
 const ui=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 assert.match(ui,/private roomMutation:/)
 for(const [start,end] of [
  ['private async createRoom():Promise<void> {','private async joinRoom():'],
  ['private async leaveRoom():Promise<void> {','private async readyRoom():']
 ]) {
  const body=ui.split(start)[1].split(end)[0]
  assert.match(body,/if\(this\.roomMutation\?\.owner===owner\)return/)
  assert.ok(body.indexOf('this.roomMutation=mutation')<body.indexOf('++this.recoveryAttempt'))
  assert.match(body,/if\(this\.roomMutation===mutation\)this\.roomMutation=null/)
 }
 assert.match(ui,/action:\(\)=>void this\.discoverMyRoomsSafely\(\)/)
 assert.match(ui,/action:\(\)=>void this\.resumeSelectedRoom\(room\.roomId\)/)
})

test('G51 audit: failed login preflight leaves the previous account event epoch active',async()=>{
 const ui=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 const login=ui.split('private async login(restoreOnly=false):Promise<void> {')[1].split('private async logoutWeb():Promise<void>{')[0]
 assert.match(login,/let generation=this\.requestGeneration/)
 assert.ok(login.indexOf('if(!cfg)throw')<login.indexOf('generation=++this.requestGeneration'))
 assert.ok(login.indexOf("if(target==='wechat' && !globals.wx)throw")<login.indexOf('generation=++this.requestGeneration'))
 assert.ok(login.indexOf('online=new GameOnlineController')<login.indexOf('generation=++this.requestGeneration'))
 assert.ok(login.indexOf('generation=++this.requestGeneration')<login.indexOf('this.online=online'))
})

test('G52-N01: stale same-account failures compare the same room/selection lifetime as successes',async()=>{
 const src=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 const section=(a,b)=>src.split(a)[1].split(b)[0]
 const discover=section('private async discoverMyRoomsSafely():Promise<void>{','private async resumeSelectedRoom(')
 assert.match(discover,/this\.recoverMyRooms\(token=>\{selection=token\}\)/)
 assert.match(discover,/this\.currentAccount\(owner,epoch\) && selection===this\.recoveryAttempt/)
 const reconnect=section('private async reconnect():Promise<void>{','private async retryUnknownFairness():')
 assert.match(reconnect,/let ownedSelection=this\.recoveryAttempt/)
 assert.match(reconnect,/ownedSelection=this\.recoveryAttempt/)
 assert.match(reconnect,/this\.recoverMyRooms\(token=>\{ownedSelection=token\}\)/)
 assert.match(reconnect,/this\.currentAccount\(owner,epoch\) && ownedSelection===this\.recoveryAttempt/)
 const refresh=section('private async refreshRoom():Promise<void> {','private async refreshFairness():')
 assert.match(refresh,/let generation=this\.roomGeneration/)
 assert.match(refresh,/generation=this\.roomGeneration \/\/ Our own adopted hand/)
 assert.match(refresh,/this\.currentRoom\(owner,epoch,roomID,generation,selection\)/)
 assert.match(refresh,/selection!==this\.recoveryAttempt \|\| !this\.currentAccount\(owner,epoch\)/)
})
test('G52: same-Hand A-B-A stale errors and successful reads share selection barrier',async()=>{
 const src=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 const section=(a,b)=>src.split(a)[1].split(b)[0]
 for(const [start,end] of [
  ['private async retryUnknownFairness():Promise<void>{','private async createRoom():'],
  ['private async refreshFairness():Promise<void>{','private async submitOwnContribution():'],
  ['private async importFairness():Promise<void>{','private async commitFairness():'],
  ['private async showEvidence():Promise<void>{','private async refreshHand():']
 ]){
  const body=section(start,end)
  assert.match(body,/selection=this\.recoveryAttempt/)
  assert.ok(body.includes('generation,selection)'),start)
  assert.match(body,/catch\(error\)\{/)
  assert.match(body.split('catch(error){').at(-1),/this\.currentRoom\([^)]*generation,selection\)/)
 }
})
test('G52: recovery coalescing honors owner and selection, so stale A is not reused after B',async()=>{
 const src=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 const sub=src.split('private resumeAuthorizedRoom(roomID:string):Promise<void>{')[1]
  .split('private async performAuthorizedRoomRecovery(')[0]
 assert.match(sub,/this\.sceneRecovery\.owner===online/)
 assert.match(sub,/this\.sceneRecovery\.selection===this\.recoveryAttempt/)
 assert.match(sub,/this\.sceneRecovery=\{roomID,owner:online,selection:token,promise\}/)
 assert.match(sub,/if\(this\.sceneRecovery\?\.promise===promise\)this\.sceneRecovery=null/)
})
test('G52: changed selection stops delayed Hand/Fairness commands from writing UI',async()=>{
 const src=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 const section=(a,b)=>src.split(a)[1].split(b)[0]
 for(const [start,end] of [
  ['private async readyRoom():Promise<void> {','private async startHand():'],
  ['private async startHand():Promise<void> {','private async refreshRoom():'],
  ['private async submitOwnContribution():Promise<void>{','private async verifyMyPhrase():'],
  ['private async verifyMyPhrase():Promise<void>{','private async prepareFairness():'],
  ['private async commitFairness():Promise<void>{','private async revealFairness():'],
  ['private async revealFairness():Promise<void>{','private async showEvidence():'],
  ['private async refreshHand():Promise<void> {','private async retryUnknownHand():'],
  ['private async retryUnknownHand():Promise<void>{','private discardUnknownHand():'],
  ['private async bid(score:number):Promise<void> {','private async play():'],
  ['private async play():Promise<void> {','private async pass():'],
  ['private async pass():Promise<void> {','private requireOnline():']
 ]){
  const body=section(start,end)
  assert.match(body,/selection=this\.recoveryAttempt/,start)
  assert.ok(body.includes('generation,selection)'),start)
 }
})

test('paged Cocos UI does not depend on downlevel Array.entries iterator',async()=>{
 const source=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 assert.match(source,/for\(let index=0;index<visible\.length;index\+\+\)/)
 assert.doesNotMatch(source,/visible\.entries\(\)/)
 assert.match(source,/this\.menuActions\.push\(\.\.\.actions\)/)
})

test('G55-N01/N02: exported Mini Game syntax target and preview settings preserve native BigInt',async()=>{
 const hooks=await readFile(new URL('../../../scripts/cocos-linux-wechat/hooks.cjs',import.meta.url),'utf8')
 const audit=await readFile(new URL('../../../scripts/cocos-linux-wechat/audit.cjs',import.meta.url),'utf8')
 const build=await readFile(new URL('../../../scripts/cocos-linux-wechat/build.cjs',import.meta.url),'utf8')
 assert.match(hooks,/options\.buildScriptParam\.targets = \{ chrome: '79' \}/)
 assert.match(hooks,/setting: \{ urlCheck: true, enhance: false, es6: false, minified: true \}/)
 assert.match(audit,/expr\.node\.operator === '\?\?'/)
 assert.match(audit,/OptionalMemberExpression/)
 assert.match(audit,/OptionalCallExpression/)
 assert.match(audit,/Generated Math\.pow receives BigInt/)
 assert.match(build,/preview\.setting\.es6,false/)
 assert.match(build,/preview\.setting\.minified,true/)
 assert.match(build,/require\('\.\/audit\.cjs'\)\.audit\(pkg\)/)
})

test('web login form uses explicit password input and server credential exchange',async()=>{
 const src=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 assert.match(src,/screen === 'login'/)
 assert.match(src,/name:'SubmitWebLogin'/)
 assert.match(src,/EditBox\.InputFlag\.PASSWORD/)
 assert.match(src,/this\.loginPassword\.string=''/)
 assert.match(src,/await online\.loginWithPassword\(identifier,password\)/)
 assert.match(src,/createWebIdentityExchange\(platform\)/)
 assert.match(src,/target==='web'/)
 assert.doesNotMatch(src,/当前尚未配置网页账号登录/)
})

test('web session revalidation and logout do not bypass current command reconciliation',async()=>{
 const src=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 assert.match(src,/name:'RestoreWebSession'/)
 assert.match(src,/name:'LogoutWebSession'/)
 assert.match(src,/await online\.restoreAndConnect\(token=>validateWebAccess\(platform,token\)\)/)
 assert.match(src,/owner\.rooms\.needsReconciliation \|\| owner\.fairness\.needsReconciliation/)
 assert.match(src,/owner\.actions\.needsReconciliation/)
 assert.match(src,/await owner\.logout\(\)/)
})

test('embedded Vue game bridge validates origin/source and never trusts a supplied token or account',async()=>{
 const src=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 const bridge=src.split('private bindWebHost():void {')[1].split('private createLabel(')[0]
 assert.match(bridge,/event\.source!==window\.parent/)
 assert.match(bridge,/event\.origin!==origin/)
 assert.match(bridge,/data\.protocol!=='azp-doudizhu-frame-v1'/)
 assert.match(bridge,/data\.type==='SESSION_AVAILABLE'/)
 assert.match(bridge,/void this\.login\(true\)/)
 assert.match(bridge,/data\.type==='SESSION_REVOKED'/)
 assert.match(bridge,/this\.online\?\.dispose\(\)/)
 assert.match(bridge,/data\.type==='OPEN_ROOMS'/)
 assert.doesNotMatch(bridge,/data\.(token|accountId|password|hand|room)/)
 assert.match(src,/window\.removeEventListener\('message',this\.webHostListener\)/)
})

test('Vue-embedded Cocos does not duplicate account login or logout controls',async()=>{
 const src=await readFile(new URL('../assets/scripts/bootstrap/GameBootstrap.ts',import.meta.url),'utf8')
 assert.match(src,/if\(this\.webHostListener\)\{/)
 assert.match(src,/findIndex\(item=>item\.name==='LoginEntry'\)/)
 assert.match(src,/else if\(detectRuntimeTarget\(\)==='web'\)/)
 assert.match(src,/StandAlone|Standalone Web exports/)
})
