import { _decorator, Color, Component, Label, Layers, Node, SafeArea, UITransform, view, EventTouch, EditBox } from 'cc'
import { GameClientShell } from '../game/GameClientShell.ts'
import { GameOnlineController } from '../game/GameOnlineController.ts'
import { verifyOwnContribution } from '../game/GameFairnessActions.ts'
import { createWeChatIdentityExchange, validateWeChatAccess } from '../game/WeChatIdentityExchange.ts'
import { createPlatform } from '../platform/createPlatform.ts'
import type { GamePlatform } from '@awesome-zero-platform/game-client-core'
import type { MiniPlatformSDK } from '@awesome-zero-platform/game-client-core/platform'
import { secureRoomCommandIdentity } from '@awesome-zero-platform/doudizhu-client/room'
import { menuWindow } from './MenuPagination.ts'
import type { PrivateHandProjection } from '@awesome-zero-platform/doudizhu-client/view'
import type { RoomRead } from '@awesome-zero-platform/doudizhu-client/api'

const { ccclass } = _decorator
type Screen = 'home' | 'rooms' | 'rules' | 'table'
interface MenuAction { name: string; caption: string; action: () => void }

type PublicConfig = { apiBase: string; socketEndpoint: string; revealManifestRoots: Record<string,string> }
function publicConfig(): PublicConfig | null {
  const env=globalThis as typeof globalThis & {__FAIR_DOUDIZHU_CONFIG__?:unknown}
  const config=env.__FAIR_DOUDIZHU_CONFIG__
  if (!config || typeof config !== 'object') return null
  const raw=config as Record<string,unknown>
  if (typeof raw.apiBase !== 'string' || !/^https:\/\/[^\s@/]+/.test(raw.apiBase) ||
      typeof raw.socketEndpoint !== 'string' || !/^wss:\/\/[^\s@/]+/.test(raw.socketEndpoint)) return null
  const roots:Record<string,string>={}
  if(raw.revealManifestRoots && typeof raw.revealManifestRoots==='object' && !Array.isArray(raw.revealManifestRoots)){
    for(const [key,value] of Object.entries(raw.revealManifestRoots)){
      if(/^[A-Za-z0-9_-]{1,128}$/.test(key) && typeof value==='string' && /^[A-Za-z0-9_-]+$/.test(value))roots[key]=value
    }
  }
  return {apiBase:raw.apiBase,socketEndpoint:raw.socketEndpoint,revealManifestRoots:roots}
}

/**
 * Cocos Creator entry for actual remote room/hand actions.
 * No local fake players, cards, login success or settlement are manufactured.
 * The public endpoint configuration is injected by deployment, never hard-coded
 * with application secrets or platform login codes.
 */
@ccclass('GameBootstrap')
export class GameBootstrap extends Component {
  private shell: GameClientShell | null = null
  private online: GameOnlineController | null = null
  private platform: GamePlatform | null = null
  private content: Node | null = null
  private title: Label | null = null
  private status: Label | null = null
  private buttons: Node[] = []
  // Keep the full accessible action set even when it does not fit in one view.
  private menuActions: MenuAction[] = []
  private menuPage = 0
  private inputNode: Node | null = null
  private roomInput: EditBox | null = null
  private screen: Screen = 'home'
  private selectedCards = new Set<string>()
  private roomID = ''
  private handID = ''
  // null is UNKNOWN; '' is a confirmed legacy hand.
  private fairnessMode:'client-contributed-server-custodied-v1'|''|null=null
  private recoverableRooms:RoomRead[]=[]
  // A kit shown on this screen is not proof that the user kept a recovery copy.
  private backupVerificationPendingHand = ''
  private verifiedBackupHand = ''
  private verifiedBackupController: GameOnlineController | null = null
  private requestGeneration = 0
  // Any leave, new hand or room switch invalidates older awaited HTTP replies.
  private roomGeneration = 0
  // Explicit room recovery choices are distinct from ordinary UI refreshes.
  private recoveryAttempt = 0
  private sceneRecovery: {
    roomID:string;owner:GameOnlineController;selection:number;promise:Promise<void>
  }|null=null
  // Coalesce duplicate create/leave taps so an already-accepted receipt
  // cannot lose its scene adoption to a later local conflict.
  private roomMutation:{owner:GameOnlineController|null;kind:'create'|'leave'}|null=null

  override start(): void {
    this.content = new Node('SafeContent')
    this.content.layer = Layers.Enum.UI_2D
    this.content.addComponent(UITransform)
    this.node.addChild(this.content)
    this.content.addComponent(SafeArea)
    this.title = this.createLabel('GameTitle', '公平斗地主', 42, new Color(240, 215, 155, 255))
    this.status = this.createLabel('GameStatus', '客户端初始化中…', 22, new Color(224, 234, 230, 255))
    this.shell = new GameClientShell(message => {
      if (this.status?.isValid && this.screen === 'home') this.status.string = message
    })
    view.on('canvas-resize', this.layout, this)
    view.on('design-resolution-changed', this.layout, this)
    this.renderScreen('home')
    void this.shell.initialize()
  }
  private createLabel(name: string, text: string, fontSize: number, color: Color): Label {
    const node = new Node(name)
    node.layer = Layers.Enum.UI_2D
    node.addComponent(UITransform)
    this.content!.addChild(node)
    const label = node.addComponent(Label)
    label.string = text
    label.fontSize = fontSize
    label.lineHeight = fontSize + 8
    label.color = color
    label.enableWrapText = true
    label.overflow = Label.Overflow.SHRINK
    return label
  }
  private renderScreen(screen: Screen): void {
    if(this.screen!==screen)this.menuPage=0
    this.screen = screen
    for (const button of this.buttons) if (button.isValid) button.destroy()
    this.buttons = []
    this.menuActions = []
    this.inputNode?.destroy()
    this.inputNode = null
    this.roomInput = null
    if (screen === 'home') {
      this.title!.string = '公平斗地主'
      this.status!.string = '三人好友房 · 公平洗牌 · 跨平台联机'
      this.addMenu([
        {name:'RoomEntry',caption:'房间大厅',action:()=>this.renderScreen('rooms')},
        {name:'RulesEntry',caption:'游戏说明',action:()=>this.renderScreen('rules')}
      ])
    } else if (screen === 'rooms') {
      this.title!.string = this.roomID ? '好友房间 · '+this.roomID : '房间大厅'
      this.status!.string = this.online?.session.authenticated
        ? '连接服务端，可创建或加入真实好友房间' : '登录后才能创建或加入真实房间'
      this.roomInput=this.createRoomInput()
      const actions:MenuAction[]=[
        {name:'LoginEntry',caption:'微信登录',action:()=>void this.login()},
        {name:'ReconnectRoom',caption:'重连并同步',action:()=>void this.reconnect()},
        {name:'CreateRoom',caption:'创建房间',action:()=>void this.createRoom()},
        {name:'JoinRoom',caption:'加入房间',action:()=>void this.joinRoom()},
        {name:'ReadyRoom',caption:'准备',action:()=>void this.readyRoom()},
        {name:'LeaveRoom',caption:'退出当前房间',action:()=>void this.leaveRoom()},
        {name:'StartHand',caption:'开始本局',action:()=>void this.startHand()},
        {name:'RefreshRoom',caption:'刷新房间',action:()=>void this.refreshRoom()},
        {name:'DiscoverMyRooms',caption:'查询我的房间',action:()=>void this.discoverMyRoomsSafely()}
      ]
      if(this.online?.rooms.needsReconciliation)
        actions.push({name:'RetryUnknownRoom',caption:'核对未确认房间命令',action:()=>void this.reconnect()})
      for(const room of this.recoverableRooms)actions.push({name:'Resume_'+room.roomId,
        caption:(room.status==='HAND_ACTIVE'?'继续牌局 ':'恢复房间 ')+room.roomId,
        action:()=>void this.resumeSelectedRoom(room.roomId)})
      if(this.fairnessMode==='client-contributed-server-custodied-v1'){
        actions.push(
          {name:'SubmitOwnContribution',caption:'提交我的随机验证字符串',action:()=>void this.submitOwnContribution()},
          {name:'VerifyOwnContribution',caption:'终局核验我的字符串',action:()=>void this.verifyMyPhrase()})
      }else if(this.handID && this.fairnessMode===''){
        // Only confirmed legacy hands expose recovery controls.
        actions.push(
          {name:'PrepareFairness',caption:'生成并备份恢复凭证（旧局）',action:()=>void this.prepareFairness()},
          {name:'VerifyFairnessBackup',caption:'已备份并清空输入（旧局）',action:()=>this.beginFairnessBackupVerification()},
          {name:'ImportFairness',caption:'导入恢复凭证（旧局）',action:()=>void this.importFairness()},
          {name:'CommitFairness',caption:'提交承诺（旧局）',action:()=>void this.commitFairness()},
          {name:'RevealFairness',caption:'揭示贡献（旧局）',action:()=>void this.revealFairness()})
      }
      actions.push(
        {name:'RetryUnknownFairness',caption:'核对上一次未确认提交',action:()=>void this.retryUnknownFairness()},
        {name:'RefreshFairness',caption:'刷新洗牌阶段',action:()=>void this.refreshFairness()},
        {name:'RoomEvidence',caption:'查看结算证据',action:()=>void this.showEvidence()},
        {name:'OpenTable',caption:'查看牌桌',action:()=>this.renderScreen('table')},
        {name:'BackHome',caption:'返回首页',action:()=>this.renderScreen('home')})
      this.addMenu(actions)
    } else if (screen === 'table') {
      const snapshot=this.online?.shell.hand.snapshot
      const pub=(snapshot?.view as PrivateHandProjection | undefined)?.public
      const me=snapshot?.view.position
      const final = snapshot?.phase === 'COMPLETED' || snapshot?.phase === 'ABORTED'
      const terminal = snapshot?.view.terminalEvidence as
        {winnerSeat:number|null;seatPoints:number[];recordDigest:string}|undefined
      this.title!.string = final ? (snapshot?.phase === 'COMPLETED' ? '公平斗地主 · 已结算' : '公平斗地主 · 已结束') :
        pub?.phase==='BIDDING' ? '公平斗地主 · 叫地主' :
        pub?.phase==='PLAYING' ? '公平斗地主 · 出牌中' : '公平斗地主 · 牌桌'
      const seatInfo=pub?.seats?.map(seat=>`座位${seat.position}${seat.position===me?'（我）':''} ${seat.remainingCards}张`).join(' / ')
      this.status!.string = final && terminal
        ? `牌局 ${snapshot!.handId} · 已核验归档版本${snapshot!.version}\n积分${terminal.seatPoints.join('/')} · ${terminal.winnerSeat ? '胜者座位'+terminal.winnerSeat : '已结束'}\n证据摘要 ${terminal.recordDigest.slice(0,24)}…`
        : snapshot
        ? `牌局 ${snapshot.handId} · 版本${snapshot.version}\n${seatInfo??''}\n${pub?.phase==='BIDDING'?'等待当前玩家叫地主':pub?.phase==='PLAYING'?('轮到座位'+String(pub.playingSeat??'?')):'等待服务端状态'}`
        : '等待服务端权威私有牌局快照，不展示模拟手牌'
      const cards=final?[]:Array.isArray(snapshot?.view.cards) ? snapshot!.view.cards as string[] : []
      this.selectedCards.forEach(selected => { if(!cards.includes(selected))this.selectedCards.delete(selected) })
      for (const card of cards) {
        this.addMenu([{name:'Card_'+card,caption:(this.selectedCards.has(card)?'✓ ':'')+card,
          action:()=>{if(this.selectedCards.has(card))this.selectedCards.delete(card);else this.selectedCards.add(card);this.renderScreen('table')}}])
      }
      const controls:MenuAction[]=[]
      if(pub?.phase==='BIDDING'){
        controls.push(
          {name:'BidZero',caption:'不叫',action:()=>void this.bid(0)},
          {name:'BidOne',caption:'叫 1 分',action:()=>void this.bid(1)},
          {name:'BidTwo',caption:'叫 2 分',action:()=>void this.bid(2)},
          {name:'BidThree',caption:'叫 3 分',action:()=>void this.bid(3)})
      }else if(pub?.phase==='PLAYING'){
        controls.push({name:'PlayCards',caption:'出牌',action:()=>void this.play()},
                      {name:'PassTurn',caption:'不出',action:()=>void this.pass()})
      }
      if(this.online?.actions.needsReconciliation){
        controls.push(
          {name:'RetryUnknownHand',caption:'重试未确认操作',action:()=>void this.retryUnknownHand()},
          {name:'DiscardUnknownHand',caption:'按新快照放弃旧操作',action:()=>this.discardUnknownHand()})
      }
      controls.push(
        {name:'RefreshHand',caption:'同步牌局',action:()=>void this.refreshHand()},
        {name:'FinalEvidence',caption:'核验结算',action:()=>void this.showEvidence()},
        {name:'BackRooms',caption:'返回房间',action:()=>this.renderScreen('rooms')})
      this.addMenu(controls)
    } else {
      this.title!.string = '玩法介绍'
      this.status!.string = '三人一桌，服务端执行公平洗牌、叫地主、出牌和结算；客户端只显示权威状态。'
      this.addMenu([{name:'BackHome',caption:'返回首页',action:()=>this.renderScreen('home')}])
    }
    this.layout()
  }
  private createRoomInput(): EditBox {
    const node=new Node('RoomCodeInput')
    node.layer=Layers.Enum.UI_2D
    node.addComponent(UITransform)
    this.content!.addChild(node)
    const input=node.addComponent(EditBox)
    // The active new hand accepts only an ephemeral personal phrase. The
    // field is never saved to a platform vault or restored from storage.
    input.maxLength=this.fairnessMode==='client-contributed-server-custodied-v1'?256:2048
    input.string=this.fairnessMode==='client-contributed-server-custodied-v1'?'':this.roomID
    const labelNode=new Node('RoomCodeText')
    labelNode.layer=Layers.Enum.UI_2D
    labelNode.addComponent(UITransform)
    node.addChild(labelNode)
    const label=labelNode.addComponent(Label)
    label.fontSize=24
    label.color=new Color(220,235,245,255)
    label.string=this.fairnessMode==='client-contributed-server-custodied-v1'
      ? '输入本人字符串（只用于参与及终局核验）'
      : (this.roomID || '输入好友房间号')
    input.textLabel=label
    this.inputNode=node
    return input
  }
  private addMenu(actions: MenuAction[]): void {
    // Collect first; layout() materializes only the current viewport. Large
    // room menus and 17-card hands must remain reachable, not shrink to 9px.
    this.menuActions.push(...actions)
  }
  private notice(message:string):void {if(this.status?.isValid)this.status.string=message}
  /** Every async UI callback belongs to one authenticated controller and one
   * Scene login epoch. Disposed/replaced accounts may finish their requests,
   * but must never modify the current account's UI. */
  private currentAccount(online:GameOnlineController|null,epoch:number):boolean{
    return online===this.online && epoch===this.requestGeneration && !!this.content?.isValid
  }
  private currentRoom(online:GameOnlineController|null,epoch:number,roomID:string,
    generation:number,selection?:number):boolean{
    return this.currentAccount(online,epoch) && roomID===this.roomID &&
      generation===this.roomGeneration &&
      (selection===undefined || selection===this.recoveryAttempt)
  }
  private async login():Promise<void> {
    // Failed preflight/login construction must not invalidate notifications
    // from the still-active prior authenticated controller.
    let generation=this.requestGeneration
    try {
      const cfg=publicConfig()
      if(!cfg)throw new Error('缺少 HTTPS/WSS 公网服务端配置')
      const globals=globalThis as typeof globalThis & {wx?:MiniPlatformSDK}
      if(!globals.wx)throw new Error('当前尚未配置网页账号登录，请使用微信小游戏')
      const platform=createPlatform({apiBase:cfg.apiBase,sdk:globals.wx})
      let online:GameOnlineController
      online=new GameOnlineController(platform,createWeChatIdentityExchange(platform),cfg.socketEndpoint,
        text=>{if(this.currentAccount(online,generation))this.notice(text)},cfg.revealManifestRoots)
      generation=++this.requestGeneration
      this.platform=platform
      ++this.recoveryAttempt
      this.sceneRecovery=null
      this.online?.dispose()
      this.online=online
      this.roomGeneration++
      this.roomID=''
      this.handID=''
      this.fairnessMode=null
      this.recoverableRooms=[]
      this.selectedCards.clear()
      const restored=await online.restoreAndConnect(token=>validateWeChatAccess(platform,token))
      if(!restored)await online.loginAndConnect()
      if(!this.currentAccount(online,generation))return
      online.shell.events.on('doudizhu.hand.snapshot',()=>{
        if(this.currentAccount(online,generation) && this.screen==='table')this.renderScreen('table')
      })
      online.shell.events.on('doudizhu.hand.evidence',payload=>{
        const id=(payload as {handId?: string})?.handId
        if(this.currentAccount(online,generation) && id && id===this.handID)void this.showEvidence()
      })
      this.notice('已验证微信账号并连接游戏服务')
      let selection=this.recoveryAttempt
      try{await this.recoverMyRooms(token=>{selection=token})}
      catch(error){if(this.currentAccount(online,generation) && selection===this.recoveryAttempt)
        this.notice('已登录，自动发现暂不可用：'+this.message(error))}
    } catch(error) {if(generation===this.requestGeneration && this.content?.isValid)
      this.notice('登录失败：'+this.message(error))}
  }
  private async discoverMyRoomsSafely():Promise<void>{
    const owner=this.online,epoch=this.requestGeneration
    let selection=this.recoveryAttempt
    try{await this.recoverMyRooms(token=>{selection=token})}
    catch(error){
      if(this.currentAccount(owner,epoch) && selection===this.recoveryAttempt)
        this.notice('查询房间失败：'+this.message(error))
    }
  }
  private async resumeSelectedRoom(roomID:string):Promise<void>{
    const owner=this.online,epoch=this.requestGeneration
    let selection=this.recoveryAttempt
    try{
      const pending=this.resumeAuthorizedRoom(roomID)
      selection=this.recoveryAttempt
      await pending
    }catch(error){
      if(this.currentAccount(owner,epoch) && selection===this.recoveryAttempt)
        this.notice('恢复房间失败：'+this.message(error))
    }
  }
  /** No game room identifier or player entropy is persisted on the device.
   * A slower account discovery must not supersede a later explicit selection. */
  private async recoverMyRooms(onOwnedSelection?:(token:number)=>void):Promise<void>{
    const online=this.requireOnline()
    const token=++this.recoveryAttempt
    onOwnedSelection?.(token)
    const rooms=await online.discoverRecoverableRooms()
    if(online!==this.online || token!==this.recoveryAttempt || !this.content?.isValid)return
    this.recoverableRooms=rooms
    const active=rooms.filter(room=>room.status==='HAND_ACTIVE')
    const candidate=active.length===1?active[0]:
      active.length===0 && rooms.length===1?rooms[0]:null
    if(candidate){
      // Auto-recovery is a child of this discovery. Hand its selection token
      // back to the caller so a legitimate failure remains visible, while a
      // different user's later choice still supersedes it.
      const pending=this.resumeAuthorizedRoom(candidate.roomId)
      onOwnedSelection?.(this.recoveryAttempt)
      await pending
    }else{
      if(this.screen==='rooms')this.renderScreen('rooms')
      this.notice(rooms.length?'发现多个未结束房间，请选择一个恢复':'当前账号没有未结束房间')
    }
  }
  private resumeAuthorizedRoom(roomID:string):Promise<void>{
    const online=this.requireOnline()
    // An obsolete A recovery must not be reused when A is selected again
    // after B (A -> B -> A). Coalesce only the still-current attempt.
    if(this.sceneRecovery?.roomID===roomID &&
       this.sceneRecovery.owner===online &&
       this.sceneRecovery.selection===this.recoveryAttempt)
      return this.sceneRecovery.promise
    const token=++this.recoveryAttempt
    const promise=this.performAuthorizedRoomRecovery(online,roomID,token)
    this.sceneRecovery={roomID,owner:online,selection:token,promise}
    void promise.finally(()=>{
      if(this.sceneRecovery?.promise===promise)this.sceneRecovery=null
    }).catch(()=>{})
    return promise
  }
  private async performAuthorizedRoomRecovery(
    online:GameOnlineController,roomID:string,token:number
  ):Promise<void>{
    const current=()=>online===this.online && token===this.recoveryAttempt && !!this.content?.isValid
    try{
      const room=await online.reconnectAndResync(roomID)
      if(!current()||!room)return
      const handID=room.activeHandId||room.lastHandId||''
      if(this.roomID!==room.roomId || this.handID!==handID){
        this.roomGeneration++
        this.fairnessMode=null
        this.selectedCards.clear()
      }
      this.roomID=room.roomId
      this.handID=handID
      this.recoverableRooms=[]
      this.renderScreen('rooms')
      // The normal screen refresh is best-effort. NEVER infer recovery
      // success from its swallowed HTTP errors; verify membership again.
      await this.refreshRoom()
      if(!current())return
      const verified=await online.roomInfo(roomID)
      if(!current())return
      if(!verified.member || verified.status==='CLOSED'){
        this.revokeInvalidRecoveredRoom(online,roomID)
        throw new Error('room closed or membership revoked during recovery')
      }
      if(verified.version<room.version ||
         verified.activeHandId!==room.activeHandId ||
         verified.lastHandId!==room.lastHandId)
        throw new Error('room changed during recovery; refresh authoritative state')
      if(online.rooms.state?.roomId!==roomID)
        throw new Error('room changed while applying recovery')
      if(online.rooms.state.version!==verified.version)
        throw new Error('room version changed after recovery; retry authoritative lookup')
      if(current())this.notice('已通过服务器恢复本人房间和牌局')
    }catch(error){
      if(current() && this.recoveryAuthorityRevoked(error))
        this.revokeInvalidRecoveredRoom(online,roomID)
      throw error
    }
  }
  private recoveryAuthorityRevoked(error:unknown):boolean{
    if(!(error instanceof Error))return false
    return /membership (not confirmed|revoked)|room closed|HTTP 403|HTTP 404|not a member/i.test(error.message)
  }
  private revokeInvalidRecoveredRoom(online:GameOnlineController,roomID:string):void{
    if(online!==this.online || this.roomID!==roomID)return
    // First revoke the controller's authoritative locator, leaving any
    // unknown durable command intact instead of discarding its retry.
    online.forgetInvalidRecoveredRoom(roomID)
    this.roomGeneration++
    this.roomID=''
    this.handID=''
    this.fairnessMode=null
    this.recoverableRooms=[]
    this.selectedCards.clear()
    this.renderScreen('rooms')
  }
  private async reconnect():Promise<void>{
    const owner=this.online,epoch=this.requestGeneration
    let ownedSelection=this.recoveryAttempt
    const initialChoice=ownedSelection
    try{
      const online=this.requireOnline()
      if(online.rooms.needsReconciliation){
        const commandType=online.rooms.pendingCommandType
        try{await online.retryUnknownRoom()}
        catch(error){
          if(!this.currentAccount(owner,epoch) || initialChoice!==this.recoveryAttempt)return
          if(online.rooms.needsReconciliation)throw error
          // A durable rejection settles the original ID; do not tell a
          // replacement account about an old account's result.
          this.notice('原房间操作已明确拒绝：'+this.message(error))
        }
        if(!this.currentAccount(owner,epoch) || initialChoice!==this.recoveryAttempt)return
        if(commandType==='room.leave' && !online.rooms.state){
          this.roomGeneration++
          this.roomID=''
          this.handID=''
          this.fairnessMode=null
          this.selectedCards.clear()
        }
        if(this.screen==='rooms')this.renderScreen('rooms')
      }
      if(!this.currentAccount(owner,epoch) || ownedSelection!==this.recoveryAttempt)return
      if(this.roomID){
        const pending=this.resumeAuthorizedRoom(this.roomID)
        ownedSelection=this.recoveryAttempt
        await pending
      }else{
        await this.recoverMyRooms(token=>{ownedSelection=token})
      }
    }catch(error){
      if(this.currentAccount(owner,epoch) && ownedSelection===this.recoveryAttempt)
        this.notice('重连失败：'+this.message(error))
    }
  }
  private async retryUnknownFairness():Promise<void>{
    const owner=this.online,epoch=this.requestGeneration
    const roomID=this.roomID,handID=this.handID
    const generation=this.roomGeneration,selection=this.recoveryAttempt
    try{
      const online=this.requireOnline()
      const fairness=online.fairness
      if(!fairness.needsReconciliation)throw new Error('没有待确认的公平提交')
      await fairness.retryUnknown()
      if(!this.currentRoom(owner,epoch,roomID,generation,selection) ||
         handID!==this.handID || fairness!==online.fairness)return
      await this.refreshFairness()
    }catch(error){
      if(this.currentRoom(owner,epoch,roomID,generation,selection) && handID===this.handID)
        this.notice('未确认提交重试失败：'+this.message(error))
    }
  }
  private async createRoom():Promise<void> {
    const owner=this.online,epoch=this.requestGeneration
    if(this.roomMutation?.owner===owner)return
    const mutation:{owner:GameOnlineController|null;kind:'create'|'leave'}={owner,kind:'create'}
    this.roomMutation=mutation
    const selection=++this.recoveryAttempt
    try {
      const online=this.requireOnline()
      const room=await online.createRoom()
      if(!this.currentAccount(owner,epoch) || selection!==this.recoveryAttempt)return
      this.roomGeneration++
      this.roomID=room.roomId
      this.handID=''
      this.fairnessMode=null
      this.recoverableRooms=[]
      this.selectedCards.clear()
      await this.refreshRoom()
    }catch(error){
      if(!this.currentAccount(owner,epoch) || selection!==this.recoveryAttempt)return
      if(this.online?.rooms.needsReconciliation && this.screen==='rooms')this.renderScreen('rooms')
      this.notice('建房失败：'+this.message(error))
    }finally{
      if(this.roomMutation===mutation)this.roomMutation=null
    }
  }
  private async joinRoom():Promise<void> {
    const owner=this.online,epoch=this.requestGeneration
    if(this.roomMutation?.owner===owner)return
    const token=++this.recoveryAttempt
    try {
      const online=this.requireOnline()
      const roomID=this.roomInput?.string?.trim() || this.roomID
      if(!roomID)throw new Error('请填写房间号')
      const room=await online.joinRoom(roomID)
      if(!this.currentAccount(owner,epoch) || token!==this.recoveryAttempt)return
      this.roomGeneration++
      this.roomID=room.roomId
      this.handID=''
      this.fairnessMode=null
      this.recoverableRooms=[]
      this.selectedCards.clear()
      await this.refreshRoom()
    }catch(error){
      if(this.currentAccount(owner,epoch) && token===this.recoveryAttempt &&
         !(error instanceof Error && /selection superseded/.test(error.message)))
        this.notice('加入失败：'+this.message(error))
    }
  }
  private async leaveRoom():Promise<void> {
    const owner=this.online,epoch=this.requestGeneration
    if(this.roomMutation?.owner===owner)return
    const mutation:{owner:GameOnlineController|null;kind:'create'|'leave'}={owner,kind:'leave'}
    this.roomMutation=mutation
    const roomID=this.roomID
    const selection=++this.recoveryAttempt
    try {
      const online=this.requireOnline()
      if(!roomID || online.rooms.state?.roomId!==roomID)
        throw new Error('请先确认当前房间成员身份')
      await online.leaveRoom()
      if(!this.currentAccount(owner,epoch) || selection!==this.recoveryAttempt ||
         roomID!==this.roomID)return
      this.roomGeneration++
      this.roomID=''
      this.handID=''
      this.fairnessMode=null
      this.recoverableRooms=[]
      this.selectedCards.clear()
      this.renderScreen('rooms')
      this.notice('已退出房间，可重新创建或加入其他房间')
    }catch(error){
      if(this.currentAccount(owner,epoch) && selection===this.recoveryAttempt &&
         roomID===this.roomID)this.notice('退出房间失败：'+this.message(error))
    }finally{
      if(this.roomMutation===mutation)this.roomMutation=null
    }
  }
  private async readyRoom():Promise<void> {
    const owner=this.online,epoch=this.requestGeneration
    const roomID=this.roomID,generation=this.roomGeneration,selection=this.recoveryAttempt
    try {
      await this.requireOnline().rooms.ready(true)
      if(!this.currentRoom(owner,epoch,roomID,generation,selection))return
      await this.refreshRoom()
    }catch(error){
      if(this.currentRoom(owner,epoch,roomID,generation,selection))
        this.notice('准备失败：'+this.message(error))
    }
  }
  private async startHand():Promise<void> {
    const online=this.online
    const epoch=this.requestGeneration
    const roomID=this.roomID
    const generation=this.roomGeneration
    const selection=this.recoveryAttempt
    try {
      if(!online?.session.authenticated)throw new Error('请先完成微信登录')
      if(!this.platform)throw new Error('游戏平台未初始化')
      if(!roomID || online.rooms.state?.roomId!==roomID)
        throw new Error('请先确认当前房间成员身份')
      const handID='hand-'+await secureRoomCommandIdentity(this.platform.random).newRequestId()
      if(!this.currentRoom(online,epoch,roomID,generation,selection))return
      // Reconcile any prior unknown fairness receipt and reserve the old
      // session before issuing the irreversible room.hand.start command.
      await online.startHand(handID)
      if(!this.currentRoom(online,epoch,roomID,generation,selection))return
      // A validated durable room.hand.start receipt must be adopted locally.
      // Later HTTP refresh failures must not turn the accepted start into FAIL.
      this.roomGeneration++
      this.handID=handID
      this.fairnessMode='client-contributed-server-custodied-v1'
      this.selectedCards.clear()
      await this.refreshRoom()
    }catch(error){
      if(this.currentRoom(online,epoch,roomID,generation,selection))
        this.notice(online?.rooms.needsReconciliation
          ? '开局结果尚未确认，请重连并同步服务端状态，勿重复发起开局：'+this.message(error)
          : '开局失败：'+this.message(error))
    }
  }
  private async refreshRoom():Promise<void> {
    const owner=this.online,epoch=this.requestGeneration
    const roomID=this.roomID,selection=this.recoveryAttempt
    let generation=this.roomGeneration
    try {
      const online=this.requireOnline()
      if(!this.roomID)throw new Error('还没有房间')
      const commandGeneration=online.rooms.commandGeneration
      const info=await online.roomInfo(roomID)
      if(generation!==this.roomGeneration || selection!==this.recoveryAttempt ||
        roomID!==this.roomID || online!==this.online ||
        (online.rooms.state!==null && online.rooms.state.roomId!==roomID))return
      if(!info.member)throw new Error('当前账号不属于该房间')
      // The HTTP read can see a newly accepted H2 while the WSS start ACK is
      // still withheld. Do not rebind fairness or restore a room coordinator
      // that still owns an in-flight command. startHand() will refresh on ACK.
      if(online.handStartInProgress)return
      online.adoptAuthorizedRoomRefresh(info,commandGeneration)
      const authorizedHand=info.activeHandId || info.lastHandId || ''
      if(authorizedHand!==this.handID){
        this.roomGeneration++
        generation=this.roomGeneration // Our own adopted hand remains eligible for error feedback.
        this.handID=authorizedHand
        this.fairnessMode=null
        this.selectedCards.clear()
      }
      // Adopt new hand as UNKNOWN before rendering. The authenticated mode
      // read below determines which fairness buttons may be displayed.
      if(this.screen==='rooms')this.renderScreen('rooms')
      const seats=(info.seats||[]).map(s=>s.accountId ? `座位${s.position} ${s.ready?'已准备':'未准备'}`:`座位${s.position} 空`).join(' / ')
      this.notice(`房间版本 ${info.version} · ${seats}`)
      if(info.activeHandId){
        const handGeneration=this.roomGeneration
        const fairness=await online.api.fairness(info.activeHandId)
        if(handGeneration!==this.roomGeneration || info.activeHandId!==this.handID ||
           selection!==this.recoveryAttempt || !this.currentAccount(owner,epoch))return
        if(this.fairnessMode !== (fairness.fairnessMode||'')){
          this.fairnessMode=fairness.fairnessMode||''
          if(this.screen==='rooms')this.renderScreen('rooms')
        }
        if(fairness.phase==='BIDDING' || fairness.phase==='PLAYING' || fairness.phase==='SETTLING'){
          await online.shell.syncHand(info.activeHandId)
        } else {
          this.notice(fairness.fairnessMode==='client-contributed-server-custodied-v1'
            ? `玩家随机贡献由服务器加密保存 · ${fairness.phase} · ${fairness.revealed?'本人已提交':'等待本人输入字符串并提交'}`
            : `公平洗牌阶段：${fairness.phase} · 座位${fairness.seat} · ${fairness.committed?'已承诺':'待承诺'} / ${fairness.revealed?'已贡献':'待贡献'}`)
        }
      } else if(info.lastHandId){
        const handGeneration=this.roomGeneration
        const evidence=await online.api.finalEvidence(info.lastHandId)
        if(handGeneration!==this.roomGeneration || info.lastHandId!==this.handID ||
           selection!==this.recoveryAttempt || !this.currentAccount(owner,epoch))return
        online.shell.acceptVerifiedFinalEvidence(evidence)
        if(this.fairnessMode !== (evidence.fairnessMode||'')){
          this.fairnessMode=evidence.fairnessMode||''
          if(this.screen==='rooms')this.renderScreen('rooms')
        }
        if(this.screen==='table')this.renderScreen('table')
      }
    }catch(error){
      if(this.currentRoom(owner,epoch,roomID,generation,selection))
        this.notice('刷新失败：'+this.message(error))
    }
  }
  private async refreshFairness():Promise<void>{
    const owner=this.online,epoch=this.requestGeneration
    const roomID=this.roomID,handID=this.handID
    const generation=this.roomGeneration,selection=this.recoveryAttempt
    try {
      const online=this.requireOnline()
      if(!this.handID)throw new Error('尚未开始牌局，请刷新房间')
      const view=await online.api.fairness(handID)
      if(!this.currentRoom(owner,epoch,roomID,generation,selection) || handID!==this.handID)return
      if(this.fairnessMode !== (view.fairnessMode||'')){
        this.fairnessMode=view.fairnessMode||''
        if(this.screen==='rooms')this.renderScreen('rooms')
      }
      const remaining=view.beaconReleaseAt ? Math.max(0,Math.ceil((Date.parse(view.beaconReleaseAt)-Date.now())/1000)) : null
      const cutoff=remaining!==null ? ` · 公共随机数公布前剩余 ${remaining} 秒` : ''
      this.notice(view.fairnessMode==='client-contributed-server-custodied-v1'
        ? `服务端已加密保存玩家贡献 · ${view.phase} · ${view.revealed?'本人已提交':'等待本人提交'}${cutoff}`
        : `洗牌阶段 ${view.phase} · 座位${view.seat} · 版本${view.version} · ${view.committed?'承诺已确认':'待承诺'} / ${view.revealed?'贡献已确认':'待提交贡献'}${cutoff}`)
      if(view.phase==='BIDDING'||view.phase==='PLAYING')await online.shell.syncHand(this.handID)
    }catch(error){
      if(this.currentRoom(owner,epoch,roomID,generation,selection) && handID===this.handID)
        this.notice('公平阶段查询失败：'+this.message(error))
    }
  }
  private async submitOwnContribution():Promise<void>{
    const online=this.online
    const epoch=this.requestGeneration
    const handID=this.handID
    const roomID=this.roomID
    const generation=this.roomGeneration
    const selection=this.recoveryAttempt
    const fairness=online?.fairness
    const phrase=this.roomInput?.string||''
    // Clear the field before the first asynchronous suspension; no client
    // storage or backup of the phrase/random is created.
    if(this.roomInput)this.roomInput.string=''
    try{
      if(this.fairnessMode!=='client-contributed-server-custodied-v1' ||
        !handID || !online?.session.authenticated || !fairness)
        throw new Error('当前牌局尚未进入个人贡献阶段')
      await fairness.submitCustodied(handID,phrase)
      if(!this.currentRoom(online,epoch,roomID,generation,selection) ||
        handID!==this.handID || fairness!==online.fairness)return
      this.notice('个人随机贡献已由服务器加密保存，可正常退出或换设备恢复牌局。请记住自己的字符串用于终局核验')
      await this.refreshFairness()
    }catch(error){
      if(this.currentRoom(online,epoch,roomID,generation,selection) && handID===this.handID)
        this.notice(fairness?.needsReconciliation
          ? '提交结果未知，请核对原请求或重新登录查询服务端，不要盲目重复提交：'+this.message(error)
          : '提交贡献失败：'+this.message(error))
    }
  }

  private async verifyMyPhrase():Promise<void>{
    const online=this.online
    const epoch=this.requestGeneration
    const roomID=this.roomID
    const handID=this.handID
    const generation=this.roomGeneration
    const selection=this.recoveryAttempt
    const phrase=this.roomInput?.string||''
    if(this.roomInput)this.roomInput.string=''
    try{
      if(!online?.session.authenticated || !handID ||
        this.fairnessMode!=='client-contributed-server-custodied-v1' || !phrase)
        throw new Error('请在终局后输入自己当初提交的字符串')
      // Retrieve authenticated final evidence before the own-random record;
      // independent cryptographic proof still depends on a verifiable transcript.
      const evidence=await online.api.finalEvidence(handID)
      const own=await online.api.ownContribution(handID)
      if(!this.currentRoom(online,epoch,roomID,generation,selection) || handID!==this.handID)return
      if(evidence.fairnessMode!==own.fairnessMode ||
        !verifyOwnContribution(handID,own.seat,own.secureRandom,phrase,own.commitment))
        throw new Error('个人字符串与本局已归档的承诺不一致')
      this.notice('个人字符串与服务器归档承诺匹配，终局记录校验摘要：'+
        evidence.verification.recordDigest.slice(0,24)+'…（可另行核验公开信标和牌序）')
    }catch(error){
      if(this.currentRoom(online,epoch,roomID,generation,selection) && handID===this.handID)
        this.notice('个人字符串校验失败：'+this.message(error))
    }
  }

  private async prepareFairness():Promise<void>{
    const online=this.online
    const epoch=this.requestGeneration
    const roomID=this.roomID
    const handID=this.handID
    const generation=this.roomGeneration
    const selection=this.recoveryAttempt
    const screen=this.screen
    const input=this.roomInput
    const fairness=online?.fairness
    try{
      if(this.fairnessMode!=='')throw new Error('公平模式尚未确认或不是历史模式')
      if(!handID)throw new Error('尚未开始牌局')
      if(!online?.session.authenticated || !fairness)throw new Error('请先完成微信登录')
      const kit=await fairness.prepareRecoveryKit(handID)
      // The reply may arrive after an authoritative cancel/leave, hand switch,
      // session replacement or UI destruction. Never re-display old secrets.
      if(generation!==this.roomGeneration || selection!==this.recoveryAttempt || roomID!==this.roomID ||
        handID!==this.handID || screen!==this.screen || this.screen!=='rooms' ||
        !this.currentAccount(online,epoch) || fairness!==online.fairness ||
        input!==this.roomInput)return
      if(!input)throw new Error('恢复凭证输入框不可用')
      this.verifiedBackupHand=''
      this.verifiedBackupController=null
      this.backupVerificationPendingHand=''
      input.string=kit
      this.notice('请先将恢复凭证私密保存到游戏外，再点“已保存备份，清空输入并校验”，从备份重新粘贴验证。不要发给其他玩家')
    }catch(error){
      if(generation===this.roomGeneration && selection===this.recoveryAttempt && handID===this.handID &&
        this.currentAccount(online,epoch) && fairness===online?.fairness)
        this.notice('准备恢复凭证失败：'+this.message(error))
    }
  }
  private beginFairnessBackupVerification():void {
    try {
      if(this.fairnessMode!=='')throw new Error('公平模式尚未确认或不是历史模式')
      if(!this.handID)throw new Error('尚未开始牌局')
      const online=this.requireOnline()
      if(!online.fairness.recoveryKitPrepared || !online.fairness.hasRecoverySecret)
        throw new Error('请先生成本局恢复凭证')
      if(!this.roomInput)throw new Error('恢复凭证输入框不可用')
      // Erase the displayed copy. Re-entry must come from a user-held copy;
      // no automatic clipboard, localStorage, or plaintext persistence.
      this.roomInput.string=''
      this.verifiedBackupHand=''
      this.verifiedBackupController=null
      this.backupVerificationPendingHand=this.handID
      this.notice('已清空显示的凭证。请从游戏外的私密备份重新粘贴，再点击“从备份粘贴并验证恢复凭证”')
    }catch(error){this.notice('备份校验无法开始：'+this.message(error))}
  }
  private async importFairness():Promise<void>{
    const owner=this.online,epoch=this.requestGeneration
    const originalRoom=this.roomID,originalHand=this.handID
    const generation=this.roomGeneration,selection=this.recoveryAttempt
    try{
      if(this.fairnessMode!=='')throw new Error('公平模式尚未确认或不是历史模式')
      if(!this.handID)throw new Error('尚未开始牌局')
      const online=this.requireOnline()
      const handID=this.handID
      // If this session generated a secret, the visible export must first be
      // cleared. A fresh/restarted session can import its external backup.
      if(online.fairness.hasRecoverySecret &&
        this.backupVerificationPendingHand!==handID)
        throw new Error('请先点击“已保存备份，清空输入并校验”')
      const kit=this.roomInput?.string?.trim()
      if(!kit)throw new Error('请从游戏外备份粘贴原先保存的恢复凭证')
      const fairness=online.fairness
      const input=this.roomInput
      await fairness.importRecoveryKit(kit,handID)
      if(!this.currentRoom(owner,epoch,originalRoom,generation,selection) || handID!==this.handID ||
        fairness!==online.fairness || input!==this.roomInput)return
      if(input)input.string=''
      this.backupVerificationPendingHand=''
      this.verifiedBackupHand=handID
      this.verifiedBackupController=online
      this.notice('恢复凭证已与当前账号、座位及服务端承诺匹配。请继续在游戏外妥善保存备份；现在可提交承诺或揭示')
    }catch(error){
      if(this.currentRoom(owner,epoch,originalRoom,generation,selection) &&
         originalHand===this.handID)this.notice('恢复贡献失败：'+this.message(error))
    }
  }
  private async commitFairness():Promise<void>{
    const owner=this.online,epoch=this.requestGeneration
    const roomID=this.roomID,handID=this.handID,generation=this.roomGeneration,selection=this.recoveryAttempt
    try{
      if(this.fairnessMode!=='')throw new Error('公平模式尚未确认或不是历史模式')
      if(!handID)throw new Error('尚未开始牌局')
      const online=this.requireOnline()
      const fairness=online.fairness
      if(!fairness.recoveryKitPrepared || this.verifiedBackupHand!==handID ||
        this.verifiedBackupController!==online)
        throw new Error('请先在游戏外保存恢复凭证，清空输入框并从备份重新粘贴验证；未经验证不能提交承诺')
      await fairness.commit(handID)
      if(!this.currentRoom(owner,epoch,roomID,generation,selection) ||
        handID!==this.handID || fairness!==online.fairness)return
      await this.refreshFairness()
    }catch(error){
      if(this.currentRoom(owner,epoch,roomID,generation,selection) && handID===this.handID)
        this.notice('公平承诺失败：'+this.message(error))
    }
  }
  private async revealFairness():Promise<void>{
    const owner=this.online,epoch=this.requestGeneration
    const roomID=this.roomID,handID=this.handID,generation=this.roomGeneration,selection=this.recoveryAttempt
    try{
      if(this.fairnessMode!=='')throw new Error('公平模式尚未确认或不是历史模式')
      if(!handID)throw new Error('尚未开始牌局')
      const online=this.requireOnline()
      const fairness=online.fairness
      await fairness.reveal(handID)
      if(!this.currentRoom(owner,epoch,roomID,generation,selection) ||
        handID!==this.handID || fairness!==online.fairness)return
      await this.refreshFairness()
    }catch(error){
      if(this.currentRoom(owner,epoch,roomID,generation,selection) && handID===this.handID)
        this.notice('加密随机贡献失败：'+this.message(error))
    }
  }
  private async showEvidence():Promise<void>{
    const owner=this.online,epoch=this.requestGeneration
    const originalHand=this.handID,originalRoom=this.roomID
    const generation=this.roomGeneration,selection=this.recoveryAttempt
    try{
      if(!this.handID)throw new Error('尚未创建牌局')
      const online=this.requireOnline()
      const handID=this.handID
      const roomID=this.roomID
      const data=await online.api.finalEvidence(handID)
      // A valid, previously authorized HTTP response is nevertheless stale
      // after Leave, a subsequent StartHand, or a session/controller switch.
      if(!this.currentRoom(owner,epoch,roomID,generation,selection) || handID!==this.handID)return
      online.shell.acceptVerifiedFinalEvidence(data)
      this.selectedCards.clear()
      if(this.screen==='table')this.renderScreen('table')
      if(this.fairnessMode !== (data.fairnessMode||'')){
        this.fairnessMode=data.fairnessMode||''
        if(this.screen==='rooms')this.renderScreen('rooms')
      }
      const verified=data.verification
      const winner=verified.winnerSeat?('获胜座位'+verified.winnerSeat):'已终止'
      this.notice(`已核验归档 · ${winner} · 积分${verified.seatPoints.join('/') } · 版本${data.finalVersion}\n校验摘要：${verified.recordDigest.slice(0,24)}…`)
    }catch(error){
      if(this.currentRoom(owner,epoch,originalRoom,generation,selection) &&
         originalHand===this.handID)this.notice('结算证据暂不可用：'+this.message(error))
    }
  }
  private async refreshHand():Promise<void> {
    const owner=this.online,epoch=this.requestGeneration
    const roomID=this.roomID,generation=this.roomGeneration,selection=this.recoveryAttempt
    try {
      const online=this.requireOnline()
      const snapshot=online.shell.hand.snapshot
      if(!snapshot)throw new Error('尚无当前牌局 ID，请先刷新房间')
      if(snapshot.phase==='COMPLETED' || snapshot.phase==='ABORTED'){
        await this.showEvidence()
      }else{
        await online.shell.syncHand(snapshot.handId,snapshot.version)
      }
    }catch(error){
      if(this.currentRoom(owner,epoch,roomID,generation,selection))
        this.notice('牌局同步失败：'+this.message(error))
    }
  }
  private async retryUnknownHand():Promise<void>{
    const owner=this.online,epoch=this.requestGeneration
    const roomID=this.roomID,handID=this.handID,generation=this.roomGeneration,selection=this.recoveryAttempt
    try {
      const online=this.requireOnline()
      await online.actions.retryUnknown()
      if(!this.currentRoom(owner,epoch,roomID,generation,selection) || handID!==this.handID)return
      await online.shell.syncHand(handID)
      if(this.currentRoom(owner,epoch,roomID,generation,selection) && handID===this.handID)
        this.notice('原请求已确认，正在同步权威牌局状态')
    } catch(error) {
      if(this.currentRoom(owner,epoch,roomID,generation,selection) && handID===this.handID)
        this.notice('原请求重试失败：'+this.message(error))
    }
  }
  private discardUnknownHand():void{
    try {
      this.requireOnline().actions.discardUnknownAfterSync()
      this.notice('已基于新的服务端牌局版本放弃旧请求，不会重复执行')
      if(this.screen==='table')this.renderScreen('table')
    } catch(error) { this.notice('不能放弃未确认操作：'+this.message(error)) }
  }
  private async bid(score:number):Promise<void> {
    const owner=this.online,epoch=this.requestGeneration
    const roomID=this.roomID,handID=this.handID,generation=this.roomGeneration,selection=this.recoveryAttempt
    try{
      await this.requireOnline().actions.bid(score)
      if(!this.currentRoom(owner,epoch,roomID,generation,selection) || handID!==this.handID)return
      this.selectedCards.clear()
      await this.refreshHand()
    }catch(error){
      if(this.currentRoom(owner,epoch,roomID,generation,selection) && handID===this.handID)
        this.notice('叫地主失败：'+this.message(error))
    }
  }
  private async play():Promise<void> {
    const owner=this.online,epoch=this.requestGeneration
    const roomID=this.roomID,handID=this.handID,generation=this.roomGeneration,selection=this.recoveryAttempt
    try {
      // Set spread is not supported by Creator 3.8's mini-game downlevel output.
      const selectedCards: string[] = []
      this.selectedCards.forEach(card => selectedCards.push(card))
      await this.requireOnline().actions.play(selectedCards)
      if(!this.currentRoom(owner,epoch,roomID,generation,selection) || handID!==this.handID)return
      this.selectedCards.clear()
      await this.refreshHand()
    }catch(error){
      if(this.currentRoom(owner,epoch,roomID,generation,selection) && handID===this.handID)
        this.notice('出牌失败：'+this.message(error))
    }
  }
  private async pass():Promise<void> {
    const owner=this.online,epoch=this.requestGeneration
    const roomID=this.roomID,handID=this.handID,generation=this.roomGeneration,selection=this.recoveryAttempt
    try{
      await this.requireOnline().actions.pass()
      if(!this.currentRoom(owner,epoch,roomID,generation,selection) || handID!==this.handID)return
      await this.refreshHand()
    }catch(error){
      if(this.currentRoom(owner,epoch,roomID,generation,selection) && handID===this.handID)
        this.notice('过牌失败：'+this.message(error))
    }
  }
  private requireOnline():GameOnlineController {
    if(!this.online?.session.authenticated)throw new Error('请先完成微信登录')
    return this.online
  }
  private message(error:unknown):string{return error instanceof Error ? error.message : '未知错误'}
  private layout(): void {
    if(!this.content?.isValid || !this.title || !this.status)return
    this.content.getComponent(SafeArea)!.updateArea()
    const size=this.content.getComponent(UITransform)!
    const height=Math.max(1,size.height)
    const portrait=height>size.width
    const width=Math.max(1,Math.min(size.width-32,portrait?1216:760))
    this.title.fontSize=portrait?72:42
    this.title.lineHeight=this.title.fontSize+8
    this.status.fontSize=portrait?42:22
    this.status.lineHeight=this.status.fontSize+8
    this.title.node.getComponent(UITransform)!.setContentSize(width,Math.min(portrait?100:70,height*0.2))
    this.title.node.setPosition(0,height*0.32,0)
    this.status.node.getComponent(UITransform)!.setContentSize(width,Math.min(portrait?180:100,height*0.23))
    this.status.node.setPosition(0,height*0.15,0)
    if(this.inputNode) {
      this.inputNode.getComponent(UITransform)!.setContentSize(Math.min(width,460),Math.max(48,height*0.075))
      this.inputNode.setPosition(0,height*0.015,0)
    }
    const window=menuWindow(this.menuActions.length,width,height,portrait,this.screen,this.menuPage)
    this.menuPage=window.page
    const visible=this.menuActions.slice(window.start,window.end)
    if(window.pages>1){
      if(window.rows===1){
        // Very short viewports have room for only one navigation control.
        visible.push({name:'NextMenuPage',caption:`下一页 ${window.page+1}/${window.pages}`,
          action:()=>{this.menuPage=(this.menuPage+1)%window.pages;this.layout()}})
      }else{
        if(window.page>0)visible.push({name:'PreviousMenuPage',caption:`上一页 ${window.page+1}/${window.pages}`,
          action:()=>{this.menuPage--;this.layout()}})
        if(window.page<window.pages-1)visible.push({name:'NextMenuPage',caption:`下一页 ${window.page+1}/${window.pages}`,
          action:()=>{this.menuPage++;this.layout()}})
      }
    }
    // Resizing and paging replace only presentation buttons. Never reset
    // the text input, selected cards, account, room or in-flight commands.
    for(const button of this.buttons)if(button.isValid)button.destroy()
    this.buttons=[]
    for(const [index,item] of visible.entries()){
      const label=this.createLabel(item.name,item.caption,24,new Color(245,221,165,255))
      const button=label.node
      button.on(Node.EventType.TOUCH_END,(_event:EventTouch)=>{
        if(this.content?.isValid)item.action()
      },this)
      this.buttons.push(button)
      const transform=button.getComponent(UITransform)!
      const cellWidth=width/window.columns
      transform.setContentSize(Math.max(20,cellWidth-8),window.buttonHeight)
      label.fontSize=Math.max(18,Math.min(26,cellWidth*0.14))
      const col=index%window.columns,row=Math.floor(index/window.columns)
      button.setPosition(-width/2+cellWidth*(col+0.5),window.firstY-row*window.rowPitch,0)
    }
  }
  override onDestroy(): void {
    this.requestGeneration++
    view.off('canvas-resize',this.layout,this)
    view.off('design-resolution-changed',this.layout,this)
    for(const button of this.buttons)if(button.isValid)button.off(Node.EventType.TOUCH_END)
    this.buttons=[]
    this.menuActions=[]
    this.online?.dispose()
    this.online=null
    this.platform=null
    this.shell?.dispose()
    this.shell=null
    if(this.content?.isValid)this.content.destroy()
    this.content=null
    this.title=null
    this.status=null
  }
}
