import { _decorator, Color, Component, Label, Layers, Node, SafeArea, UITransform, view, EventTouch, EditBox } from 'cc'
import { GameClientShell } from '../game/GameClientShell.ts'
import { GameOnlineController } from '../game/GameOnlineController.ts'
import { createWeChatIdentityExchange, validateWeChatAccess } from '../game/WeChatIdentityExchange.ts'
import { createPlatform } from '../platform/createPlatform.ts'
import type { GamePlatform } from '@awesome-zero-platform/game-client-core'
import type { MiniPlatformSDK } from '@awesome-zero-platform/game-client-core/platform'
import { secureRoomCommandIdentity } from '@awesome-zero-platform/doudizhu-client/room'
import type { PrivateHandProjection } from '@awesome-zero-platform/doudizhu-client/view'

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
  private inputNode: Node | null = null
  private roomInput: EditBox | null = null
  private screen: Screen = 'home'
  private selectedCards = new Set<string>()
  private roomID = ''
  private handID = ''
  // A kit shown on this screen is not proof that the user kept a recovery copy.
  private backupVerificationPendingHand = ''
  private verifiedBackupHand = ''
  private requestGeneration = 0
  // Any leave, new hand or room switch invalidates older awaited HTTP replies.
  private roomGeneration = 0

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
    this.screen = screen
    for (const button of this.buttons) if (button.isValid) button.destroy()
    this.buttons = []
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
      this.addMenu([
        {name:'LoginEntry',caption:'微信登录',action:()=>void this.login()},
        {name:'ReconnectRoom',caption:'重连并同步',action:()=>void this.reconnect()},
        {name:'CreateRoom',caption:'创建房间',action:()=>void this.createRoom()},
        {name:'JoinRoom',caption:'加入房间',action:()=>void this.joinRoom()},
        {name:'ReadyRoom',caption:'准备',action:()=>void this.readyRoom()},
        {name:'LeaveRoom',caption:'退出当前房间',action:()=>void this.leaveRoom()},
        {name:'StartHand',caption:'开始本局',action:()=>void this.startHand()},
        {name:'RefreshRoom',caption:'刷新房间',action:()=>void this.refreshRoom()},
        {name:'PrepareFairness',caption:'生成并备份恢复凭证',action:()=>void this.prepareFairness()},
        {name:'VerifyFairnessBackup',caption:'已保存备份，清空输入并校验',action:()=>this.beginFairnessBackupVerification()},
        {name:'ImportFairness',caption:'从备份粘贴并验证恢复凭证',action:()=>void this.importFairness()},
        {name:'CommitFairness',caption:'提交洗牌承诺',action:()=>void this.commitFairness()},
        {name:'RevealFairness',caption:'加密提交随机贡献',action:()=>void this.revealFairness()},
        {name:'RetryUnknownFairness',caption:'重试未确认提交',action:()=>void this.retryUnknownFairness()},
        {name:'RefreshFairness',caption:'刷新洗牌阶段',action:()=>void this.refreshFairness()},
        {name:'RoomEvidence',caption:'查看结算证据',action:()=>void this.showEvidence()},
        {name:'OpenTable',caption:'查看牌桌',action:()=>this.renderScreen('table')},
        {name:'BackHome',caption:'返回首页',action:()=>this.renderScreen('home')}
      ])
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
    // The same field also accepts the explicit, user-held 2048-character
    // fairness recovery kit. Room IDs are validated independently by API.
    input.maxLength=2048
    input.string=this.roomID
    const labelNode=new Node('RoomCodeText')
    labelNode.layer=Layers.Enum.UI_2D
    labelNode.addComponent(UITransform)
    node.addChild(labelNode)
    const label=labelNode.addComponent(Label)
    label.fontSize=24
    label.color=new Color(220,235,245,255)
    label.string=this.roomID || '输入好友房间号'
    input.textLabel=label
    this.inputNode=node
    return input
  }
  private addMenu(actions: MenuAction[]): void {
    for (const item of actions) {
      const label=this.createLabel(item.name,item.caption,24,new Color(245,221,165,255))
      const node=label.node
      node.on(Node.EventType.TOUCH_END,(_event:EventTouch)=>{
        if(this.content?.isValid)item.action()
      },this)
      this.buttons.push(node)
    }
  }
  private notice(message:string):void {if(this.status?.isValid)this.status.string=message}
  private async login():Promise<void> {
    const generation=++this.requestGeneration
    try {
      const cfg=publicConfig()
      if(!cfg)throw new Error('缺少 HTTPS/WSS 公网服务端配置')
      const globals=globalThis as typeof globalThis & {wx?:MiniPlatformSDK}
      if(!globals.wx)throw new Error('当前尚未配置网页账号登录，请使用微信小游戏')
      const platform=createPlatform({apiBase:cfg.apiBase,sdk:globals.wx})
      const online=new GameOnlineController(platform,createWeChatIdentityExchange(platform),cfg.socketEndpoint,
        text=>this.notice(text),cfg.revealManifestRoots)
      this.platform=platform
      this.online?.dispose()
      this.online=online
      const restored=await online.restoreAndConnect(token=>validateWeChatAccess(platform,token))
      if(!restored)await online.loginAndConnect()
      if(generation!==this.requestGeneration || !this.content?.isValid)return
      online.shell.events.on('doudizhu.hand.snapshot',()=>{
        if(this.screen==='table')this.renderScreen('table')
      })
      online.shell.events.on('doudizhu.hand.evidence',payload=>{
        const id=(payload as {handId?: string})?.handId
        if(id && id===this.handID)void this.showEvidence()
      })
      this.notice('已验证微信账号并连接游戏服务')
    } catch(error) {if(generation===this.requestGeneration)this.notice('登录失败：'+this.message(error))}
  }
  private async reconnect():Promise<void> {
    try {
      const view=await this.requireOnline().reconnectAndResync(this.roomID || undefined)
      if(view) {
        const currentHand=view.activeHandId || view.lastHandId || ''
        if(this.roomID!==view.roomId || this.handID!==currentHand) {
          this.roomGeneration++
          this.selectedCards.clear()
        }
        this.roomID=view.roomId
        this.handID=currentHand
      }
      this.notice(view?'重连成功，已向服务器请求房间与牌局同步':'重连成功，等待创建或加入房间')
    }catch(error){this.notice('重连失败：'+this.message(error))}
  }
  private async retryUnknownFairness():Promise<void>{
    try{
      const online=this.requireOnline()
      if(!online.fairness.needsReconciliation)throw new Error('没有待确认的公平提交')
      await online.fairness.retryUnknown()
      await this.refreshFairness()
    }catch(error){this.notice('未确认提交重试失败：'+this.message(error))}
  }
  private async createRoom():Promise<void> {
    try {
      const online=this.requireOnline()
      const room=await online.createRoom()
      this.roomGeneration++
      this.roomID=room.roomId
      this.handID=''
      this.selectedCards.clear()
      await this.refreshRoom()
    }catch(error){this.notice('建房失败：'+this.message(error))}
  }
  private async joinRoom():Promise<void> {
    try {
      const online=this.requireOnline()
      const roomID=this.roomInput?.string?.trim() || this.roomID
      if(!roomID)throw new Error('请填写房间号')
      const room=await online.joinRoom(roomID)
      this.roomGeneration++
      this.roomID=room.roomId
      this.handID=''
      this.selectedCards.clear()
      await this.refreshRoom()
    }catch(error){this.notice('加入失败：'+this.message(error))}
  }
  private async leaveRoom():Promise<void> {
    try {
      const online=this.requireOnline()
      if(!this.roomID || online.rooms.state?.roomId!==this.roomID)
        throw new Error('请先确认当前房间成员身份')
      await online.leaveRoom()
      this.roomGeneration++
      this.roomID=''
      this.handID=''
      this.selectedCards.clear()
      this.renderScreen('rooms')
      this.notice('已退出房间，可重新创建或加入其他房间')
    }catch(error){this.notice('退出房间失败：'+this.message(error))}
  }
  private async readyRoom():Promise<void> {
    try {
      await this.requireOnline().rooms.ready(true)
      await this.refreshRoom()
    }catch(error){this.notice('准备失败：'+this.message(error))}
  }
  private async startHand():Promise<void> {
    try {
      const online=this.requireOnline()
      if(!this.platform)throw new Error('游戏平台未初始化')
      const handID='hand-'+await secureRoomCommandIdentity(this.platform.random).newRequestId()
      await online.rooms.startHand(handID)
      this.roomGeneration++
      this.handID=handID
      this.selectedCards.clear()
      online.shell.useAuthoritativeHand(handID)
      await this.refreshRoom()
    }catch(error){this.notice('开局失败：'+this.message(error))}
  }
  private async refreshRoom():Promise<void> {
    try {
      const online=this.requireOnline()
      if(!this.roomID)throw new Error('还没有房间')
      const roomID=this.roomID
      const generation=this.roomGeneration
      const info=await online.roomInfo(roomID)
      if(generation!==this.roomGeneration || roomID!==this.roomID || online!==this.online)return
      if(!info.member)throw new Error('当前账号不属于该房间')
      online.rooms.restoreTrustedRoom(info.roomId,info.version,info.lastClientSequence)
      if(this.screen==='rooms')this.renderScreen('rooms')
      const seats=(info.seats||[]).map(s=>s.accountId ? `座位${s.position} ${s.ready?'已准备':'未准备'}`:`座位${s.position} 空`).join(' / ')
      this.notice(`房间版本 ${info.version} · ${seats}`)
      const authorizedHand=info.activeHandId || info.lastHandId || ''
      if(authorizedHand!==this.handID){
        this.roomGeneration++
        this.handID=authorizedHand
        this.selectedCards.clear()
      }
      online.shell.useAuthoritativeHand(authorizedHand || null)
      if(info.activeHandId){
        const handGeneration=this.roomGeneration
        const fairness=await online.api.fairness(info.activeHandId)
        if(handGeneration!==this.roomGeneration || info.activeHandId!==this.handID || online!==this.online)return
        if(fairness.phase==='BIDDING' || fairness.phase==='PLAYING' || fairness.phase==='SETTLING'){
          await online.shell.syncHand(info.activeHandId)
        } else {
          this.notice(`公平洗牌阶段：${fairness.phase} · 座位${fairness.seat} · ${fairness.committed?'已承诺':'待承诺'} / ${fairness.revealed?'已贡献':'待贡献'}`)
        }
      } else if(info.lastHandId){
        const handGeneration=this.roomGeneration
        const evidence=await online.api.finalEvidence(info.lastHandId)
        if(handGeneration!==this.roomGeneration || info.lastHandId!==this.handID || online!==this.online)return
        online.shell.acceptVerifiedFinalEvidence(evidence)
        if(this.screen==='table')this.renderScreen('table')
      }
    }catch(error){this.notice('刷新失败：'+this.message(error))}
  }
  private async refreshFairness():Promise<void>{
    try {
      const online=this.requireOnline()
      if(!this.handID)throw new Error('尚未开始牌局，请刷新房间')
      const view=await online.api.fairness(this.handID)
      const remaining=view.beaconReleaseAt ? Math.max(0,Math.ceil((Date.parse(view.beaconReleaseAt)-Date.now())/1000)) : null
      const cutoff=remaining!==null ? ` · 公共随机数公布前剩余 ${remaining} 秒` : ''
      this.notice(`洗牌阶段 ${view.phase} · 座位${view.seat} · 版本${view.version} · ${view.committed?'承诺已确认':'待承诺'} / ${view.revealed?'贡献已确认':'待提交贡献'}${cutoff}`)
      if(view.phase==='BIDDING'||view.phase==='PLAYING')await online.shell.syncHand(this.handID)
    }catch(error){this.notice('公平阶段查询失败：'+this.message(error))}
  }
  private async prepareFairness():Promise<void>{
    try{
      if(!this.handID)throw new Error('尚未开始牌局')
      const kit=await this.requireOnline().fairness.prepareRecoveryKit(this.handID)
      if(!this.roomInput)throw new Error('恢复凭证输入框不可用')
      this.verifiedBackupHand=''
      this.backupVerificationPendingHand=''
      this.roomInput.string=kit
      this.notice('请先将恢复凭证私密保存到游戏外，再点“已保存备份，清空输入并校验”，从备份重新粘贴验证。不要发给其他玩家')
    }catch(error){this.notice('准备恢复凭证失败：'+this.message(error))}
  }
  private beginFairnessBackupVerification():void {
    try {
      if(!this.handID)throw new Error('尚未开始牌局')
      const online=this.requireOnline()
      if(!online.fairness.recoveryKitPrepared || !online.fairness.hasRecoverySecret)
        throw new Error('请先生成本局恢复凭证')
      if(!this.roomInput)throw new Error('恢复凭证输入框不可用')
      // Erase the displayed copy. Re-entry must come from a user-held copy;
      // no automatic clipboard, localStorage, or plaintext persistence.
      this.roomInput.string=''
      this.verifiedBackupHand=''
      this.backupVerificationPendingHand=this.handID
      this.notice('已清空显示的凭证。请从游戏外的私密备份重新粘贴，再点击“从备份粘贴并验证恢复凭证”')
    }catch(error){this.notice('备份校验无法开始：'+this.message(error))}
  }
  private async importFairness():Promise<void>{
    try{
      if(!this.handID)throw new Error('尚未开始牌局')
      const online=this.requireOnline()
      const handID=this.handID
      const generation=this.roomGeneration
      // If this session generated a secret, the visible export must first be
      // cleared. A fresh/restarted session can import its external backup.
      if(online.fairness.hasRecoverySecret &&
        this.backupVerificationPendingHand!==handID)
        throw new Error('请先点击“已保存备份，清空输入并校验”')
      const kit=this.roomInput?.string?.trim()
      if(!kit)throw new Error('请从游戏外备份粘贴原先保存的恢复凭证')
      await online.fairness.importRecoveryKit(kit,handID)
      if(generation!==this.roomGeneration || handID!==this.handID || online!==this.online)return
      this.roomInput!.string=''
      this.backupVerificationPendingHand=''
      this.verifiedBackupHand=handID
      this.notice('恢复凭证已与当前账号、座位及服务端承诺匹配。请继续在游戏外妥善保存备份；现在可提交承诺或揭示')
    }catch(error){this.notice('恢复贡献失败：'+this.message(error))}
  }
  private async commitFairness():Promise<void>{
    try{
      if(!this.handID)throw new Error('尚未开始牌局')
      const online=this.requireOnline()
      if(!online.fairness.recoveryKitPrepared || this.verifiedBackupHand!==this.handID)
        throw new Error('请先在游戏外保存恢复凭证，清空输入框并从备份重新粘贴验证；未经验证不能提交承诺')
      await online.fairness.commit(this.handID)
      await this.refreshFairness()
    }catch(error){this.notice('公平承诺失败：'+this.message(error))}
  }
  private async revealFairness():Promise<void>{
    try{
      if(!this.handID)throw new Error('尚未开始牌局')
      await this.requireOnline().fairness.reveal(this.handID)
      await this.refreshFairness()
    }catch(error){this.notice('加密随机贡献失败：'+this.message(error))}
  }
  private async showEvidence():Promise<void>{
    try{
      if(!this.handID)throw new Error('尚未创建牌局')
      const online=this.requireOnline()
      const generation=this.roomGeneration
      const handID=this.handID
      const roomID=this.roomID
      const data=await online.api.finalEvidence(handID)
      // A valid, previously authorized HTTP response is nevertheless stale
      // after Leave, a subsequent StartHand, or a session/controller switch.
      if(generation!==this.roomGeneration || handID!==this.handID ||
          roomID!==this.roomID || online!==this.online)return
      online.shell.acceptVerifiedFinalEvidence(data)
      this.selectedCards.clear()
      if(this.screen==='table')this.renderScreen('table')
      const verified=data.verification
      const winner=verified.winnerSeat?('获胜座位'+verified.winnerSeat):'已终止'
      this.notice(`已核验归档 · ${winner} · 积分${verified.seatPoints.join('/') } · 版本${data.finalVersion}\n校验摘要：${verified.recordDigest.slice(0,24)}…`)
    }catch(error){this.notice('结算证据暂不可用：'+this.message(error))}
  }
  private async refreshHand():Promise<void> {
    try {
      const snapshot=this.requireOnline().shell.hand.snapshot
      if(!snapshot)throw new Error('尚无当前牌局 ID，请先刷新房间')
      if(snapshot.phase==='COMPLETED' || snapshot.phase==='ABORTED'){
        await this.showEvidence()
      }else{
        await this.requireOnline().shell.syncHand(snapshot.handId,snapshot.version)
      }
    }catch(error){this.notice('牌局同步失败：'+this.message(error))}
  }
  private async retryUnknownHand():Promise<void>{
    try {
      const online=this.requireOnline()
      await online.actions.retryUnknown()
      await online.shell.syncHand(this.handID)
      this.notice('原请求已确认，正在同步权威牌局状态')
    } catch(error) { this.notice('原请求重试失败：'+this.message(error)) }
  }
  private discardUnknownHand():void{
    try {
      this.requireOnline().actions.discardUnknownAfterSync()
      this.notice('已基于新的服务端牌局版本放弃旧请求，不会重复执行')
      if(this.screen==='table')this.renderScreen('table')
    } catch(error) { this.notice('不能放弃未确认操作：'+this.message(error)) }
  }
  private async bid(score:number):Promise<void> {
    try {await this.requireOnline().actions.bid(score);this.selectedCards.clear();await this.refreshHand()}
    catch(error){this.notice('叫地主失败：'+this.message(error))}
  }
  private async play():Promise<void> {
    try {
      // Set spread is not supported by Creator 3.8's mini-game downlevel output.
      const selectedCards: string[] = []
      this.selectedCards.forEach(card => selectedCards.push(card))
      await this.requireOnline().actions.play(selectedCards)
      this.selectedCards.clear();await this.refreshHand()
    }
    catch(error){this.notice('出牌失败：'+this.message(error))}
  }
  private async pass():Promise<void> {
    try {await this.requireOnline().actions.pass();await this.refreshHand()}
    catch(error){this.notice('过牌失败：'+this.message(error))}
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
    const columns=this.screen==='home'||this.screen==='rules'?1:this.screen==='table'?(portrait?5:10):(portrait?3:4)
    const rows=Math.max(1,Math.ceil(this.buttons.length/columns))
    // Both a 15-control room lobby and a full 20-card table must fit inside
    // the safe-area bottom half, including on compact landscape viewports.
    const rowPitch=Math.min(height*0.106,height*0.34/Math.max(1,rows-1))
    const buttonHeight=Math.max(28,Math.min(58,rowPitch-6))
    this.buttons.forEach((button,index)=>{
      const transform=button.getComponent(UITransform)!
      const cellWidth=width/columns
      transform.setContentSize(Math.max(20,cellWidth-8),buttonHeight)
      button.getComponent(Label)!.fontSize=portrait?Math.max(18,Math.min(30,cellWidth*0.16)):Math.max(18,Math.min(26,cellWidth*0.17))
      const col=index%columns,row=Math.floor(index/columns)
      button.setPosition(-width/2+cellWidth*(col+0.5),-height*0.105-row*rowPitch,0)
    })
  }
  override onDestroy(): void {
    this.requestGeneration++
    view.off('canvas-resize',this.layout,this)
    view.off('design-resolution-changed',this.layout,this)
    for(const button of this.buttons)if(button.isValid)button.off(Node.EventType.TOUCH_END)
    this.buttons=[]
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
