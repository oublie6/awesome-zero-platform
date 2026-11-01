import { _decorator, Color, Component, Label, Layers, Node, SafeArea, UITransform, view, EventTouch } from 'cc'
import { GameClientShell } from '../game/GameClientShell.ts'

const { ccclass } = _decorator
type Screen = 'home' | 'rooms' | 'rules'
interface MenuAction { name: string; caption: string; action: () => void }

/**
 * Lightweight interactive front-end shell. No simulated login, room or cards.
 * All business operations must be wired to authenticated Go endpoints before enabled.
 */
@ccclass('GameBootstrap')
export class GameBootstrap extends Component {
  private shell: GameClientShell | null = null
  private content: Node | null = null
  private title: Label | null = null
  private status: Label | null = null
  private buttons: Node[] = []
  private screen: Screen = 'home'

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
    if (screen === 'home') {
      this.title!.string = '公平斗地主'
      this.status!.string = '三人好友房 · 公平洗牌 · 跨平台联机'
      this.addMenu([
        { name: 'RoomEntry', caption: '房间大厅', action: () => this.renderScreen('rooms') },
        { name: 'RulesEntry', caption: '游戏说明', action: () => this.renderScreen('rules') }
      ])
    } else if (screen === 'rooms') {
      this.title!.string = '房间大厅'
      this.status!.string = '真实账号与服务端房间接口尚未接通；不创建虚假房间'
      this.addMenu([{ name: 'BackHome', caption: '返回首页', action: () => this.renderScreen('home') }])
    } else {
      this.title!.string = '玩法介绍'
      this.status!.string = '三人一桌；房主创建好友房，玩家准备后按服务端公平发牌、叫地主、出牌和结算流程进行。'
      this.addMenu([{ name: 'BackHome', caption: '返回首页', action: () => this.renderScreen('home') }])
    }
    this.layout()
  }

  private addMenu(actions: MenuAction[]): void {
    for (const item of actions) {
      const label = this.createLabel(item.name, item.caption, 24, new Color(245, 221, 165, 255))
      const node = label.node
      node.on(Node.EventType.TOUCH_END, (_event: EventTouch) => {
        if (this.content?.isValid) item.action()
      }, this)
      this.buttons.push(node)
    }
  }

  private layout(): void {
    if (!this.content?.isValid || !this.title || !this.status) return
    this.content.getComponent(SafeArea)!.updateArea()
    const size = this.content.getComponent(UITransform)!
    const height = Math.max(1, size.height)
    const portrait = height > size.width
    const width = Math.max(1, Math.min(size.width - 32, portrait ? 1216 : 760))
    this.title.fontSize = portrait ? 72 : 42
    this.title.lineHeight = this.title.fontSize + 8
    this.status.fontSize = portrait ? 42 : 22
    this.status.lineHeight = this.status.fontSize + 8
    this.title.node.getComponent(UITransform)!.setContentSize(width, Math.min(portrait ? 100 : 70, height * 0.2))
    this.title.node.setPosition(0, height * 0.28, 0)
    this.status.node.getComponent(UITransform)!.setContentSize(width, Math.min(portrait ? 260 : 120, height * 0.3))
    this.status.node.setPosition(0, height * 0.02, 0)
    this.buttons.forEach((button, index) => {
      const transform = button.getComponent(UITransform)!
      transform.setContentSize(Math.min(width, 460), Math.max(52, Math.min(86, height * 0.13)))
      button.getComponent(Label)!.fontSize = portrait ? 36 : 25
      button.setPosition(0, -height * (0.22 + index * 0.12), 0)
    })
  }

  override onDestroy(): void {
    view.off('canvas-resize', this.layout, this)
    view.off('design-resolution-changed', this.layout, this)
    for (const button of this.buttons) if (button.isValid) button.off(Node.EventType.TOUCH_END)
    this.buttons = []
    this.shell?.dispose()
    this.shell = null
    if (this.content?.isValid) this.content.destroy()
    this.content = null
    this.title = null
    this.status = null
  }
}
