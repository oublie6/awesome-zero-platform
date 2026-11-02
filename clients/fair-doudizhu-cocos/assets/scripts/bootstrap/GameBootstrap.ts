import { _decorator, Color, Component, Label, Layers, Node, SafeArea, UITransform, view, EventTouch, EditBox } from 'cc'
import { GameClientShell } from '../game/GameClientShell.ts'
import { GameOnlineController } from '../game/GameOnlineController.ts'
import { createWeChatIdentityExchange, validateWeChatAccess } from '../game/WeChatIdentityExchange.ts'
import { createPlatform } from '../platform/createPlatform.ts'
import type { GamePlatform } from '@awesome-zero-platform/game-client-core'
import type { MiniPlatformSDK } from '@awesome-zero-platform/game-client-core/platform'
import { secureRoomCommandIdentity } from '@awesome-zero-platform/doudizhu-client/room'

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
  private requestGeneration = 0

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
        {name:'CreateRoom',caption:'创建房间',action:()=>void this.createRoom()},
        {name:'JoinRoom',caption:'加入房间',action:()=>void this.joinRoom()},
        {name:'ReadyRoom',caption:'准备',action:()=>void this.readyRoom()},
        {name:'StartHand',caption:'开始本局',action:()=>void this.startHand()},
        {name:'RefreshRoom',caption:'刷新房间',action:()=>void this.refreshRoom()},
        {name:'CommitFairness',caption:'提交洗牌承诺',action:()=>void this.commitFairness()},
        {name:'RevealFairness',caption:'加密提交随机贡献',action:()=>void this.revealFairness()},
        {name:'RefreshFairness',caption:'刷新洗牌阶段',action:()=>void this.refreshFairness()},
        {name:'OpenTable',caption:'查看牌桌',action:()=>this.renderScreen('table')},
        {name:'BackHome',caption:'返回首页',action:()=>this.renderScreen('home')}
      ])
    } else if (screen === 'table') {
      this.title!.string = '公平斗地主 · 牌桌'
      const snapshot=this.online?.shell.hand.snapshot
      this.status!.string = snapshot
        ? `牌局 ${snapshot.handId} · ${snapshot.phase} · 版本 ${snapshot.version}`
        : '等待服务端权威私有牌局快照，不展示模拟手牌'
      const cards=Array.isArray(snapshot?.view.cards) ? snapshot!.view.cards as string[] : []
      for (const card of cards) {
        this.addMenu([{name:'Card_'+card,caption:(this.selectedCards.has(card)?'✓ ':'')+card,
          action:()=>{if(this.selectedCards.has(card))this.selectedCards.delete(card);else this.selectedCards.add(card);this.renderScreen('table')}}])
      }
      this.addMenu([
        {name:'BidZero',caption:'不叫',action:()=>void this.bid(0)},
        {name:'BidOne',caption:'叫 1 分',action:()=>void this.bid(1)},
        {name:'BidTwo',caption:'叫 2 分',action:()=>void this.bid(2)},
        {name:'BidThree',caption:'叫 3 分',action:()=>void this.bid(3)},
        {name:'PlayCards',caption:'出牌',action:()=>void this.play()},
        {name:'PassTurn',caption:'不出',action:()=>void this.pass()},
        {name:'RefreshHand',caption:'同步牌局',action:()=>void this.refreshHand()},
        {name:'BackRooms',caption:'返回房间',action:()=>this.renderScreen('rooms')}
      ])
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
    input.maxLength=128
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
      this.notice('已验证微信账号并连接游戏服务')
    } catch(error) {if(generation===this.requestGeneration)this.notice('登录失败：'+this.message(error))}
  }
  private async createRoom():Promise<void> {
    try {
      const online=this.requireOnline()
      const room=await online.createRoom()
      this.roomID=room.roomId
      await this.refreshRoom()
    }catch(error){this.notice('建房失败：'+this.message(error))}
  }
  private async joinRoom():Promise<void> {
    try {
      const online=this.requireOnline()
      const roomID=this.roomInput?.string?.trim() || this.roomID
      if(!roomID)throw new Error('请填写房间号')
      const room=await online.joinRoom(roomID)
      this.roomID=room.roomId
      await this.refreshRoom()
    }catch(error){this.notice('加入失败：'+this.message(error))}
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
      await this.refreshRoom()
    }catch(error){this.notice('开局失败：'+this.message(error))}
  }
  private async refreshRoom():Promise<void> {
    try {
      const online=this.requireOnline()
      if(!this.roomID)throw new Error('还没有房间')
      const info=await online.roomInfo(this.roomID)
      if(!info.member)throw new Error('当前账号不属于该房间')
      online.rooms.restoreTrustedRoom(info.roomId,info.version,info.lastClientSequence)
      if(this.screen==='rooms')this.renderScreen('rooms')
      const seats=(info.seats||[]).map(s=>s.accountId ? `座位${s.position} ${s.ready?'已准备':'未准备'}`:`座位${s.position} 空`).join(' / ')
      this.notice(`房间版本 ${info.version} · ${seats}`)
      if(info.activeHandId){
        this.handID=info.activeHandId
        const fairness=await online.api.fairness(info.activeHandId)
        if(fairness.phase==='BIDDING' || fairness.phase==='PLAYING' || fairness.phase==='SETTLING'){
          await online.shell.syncHand(info.activeHandId)
        } else {
          this.notice(`公平洗牌阶段：${fairness.phase} · 座位${fairness.seat} · ${fairness.committed?'已承诺':'待承诺'} / ${fairness.revealed?'已贡献':'待贡献'}`)
        }
      }
    }catch(error){this.notice('刷新失败：'+this.message(error))}
  }
  private async refreshFairness():Promise<void>{
    try {
      const online=this.requireOnline()
      if(!this.handID)throw new Error('尚未开始牌局，请刷新房间')
      const view=await online.api.fairness(this.handID)
      this.notice(`洗牌阶段 ${view.phase} · 座位${view.seat} · 版本${view.version} · ${view.committed?'承诺已确认':'待承诺'} / ${view.revealed?'贡献已确认':'待提交贡献'}`)
      if(view.phase==='BIDDING'||view.phase==='PLAYING')await online.shell.syncHand(this.handID)
    }catch(error){this.notice('公平阶段查询失败：'+this.message(error))}
  }
  private async commitFairness():Promise<void>{
    try{
      if(!this.handID)throw new Error('尚未开始牌局')
      await this.requireOnline().fairness.commit(this.handID)
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
  private async refreshHand():Promise<void> {
    try {
      const snapshot=this.requireOnline().shell.hand.snapshot
      if(!snapshot)throw new Error('尚无当前牌局 ID，请先刷新房间')
      await this.requireOnline().shell.syncHand(snapshot.handId,snapshot.version)
    }catch(error){this.notice('牌局同步失败：'+this.message(error))}
  }
  private async bid(score:number):Promise<void> {
    try {await this.requireOnline().actions.bid(score);this.selectedCards.clear();await this.refreshHand()}
    catch(error){this.notice('叫地主失败：'+this.message(error))}
  }
  private async play():Promise<void> {
    try {await this.requireOnline().actions.play([...this.selectedCards]);this.selectedCards.clear();await this.refreshHand()}
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
    const columns=this.screen==='home'||this.screen==='rules'?1:(portrait?3:4)
    this.buttons.forEach((button,index)=>{
      const transform=button.getComponent(UITransform)!
      const cellWidth=width/columns
      transform.setContentSize(Math.max(20,cellWidth-8),Math.max(30,Math.min(72,height*0.095)))
      button.getComponent(Label)!.fontSize=portrait?Math.max(18,Math.min(30,cellWidth*0.16)):Math.max(18,Math.min(26,cellWidth*0.17))
      const col=index%columns,row=Math.floor(index/columns)
      button.setPosition(-width/2+cellWidth*(col+0.5),-height*(0.13+row*(portrait?0.091:0.12)),0)
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
