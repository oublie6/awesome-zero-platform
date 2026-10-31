import { _decorator, Color, Component, Label, Layers, Node, SafeArea, UITransform, view } from 'cc'
import { GameClientShell } from '../game/GameClientShell.js'

const { ccclass } = _decorator

/** Thin scene host; Canvas and its camera are serialized in Entry.scene. */
@ccclass('GameBootstrap')
export class GameBootstrap extends Component {
  private shell: GameClientShell | null = null
  private content: Node | null = null
  private title: Label | null = null
  private status: Label | null = null

  override start(): void {
    this.content = new Node('SafeContent')
    this.content.layer = Layers.Enum.UI_2D
    this.content.addComponent(UITransform)
    this.node.addChild(this.content)
    this.content.addComponent(SafeArea)
    this.title = this.createLabel('GameTitle', '公平斗地主', 42, new Color(240, 215, 155, 255))
    this.status = this.createLabel('GameStatus', '客户端初始化中…', 22, new Color(224, 234, 230, 255))
    view.on('canvas-resize', this.layout, this)
    view.on('design-resolution-changed', this.layout, this)
    this.layout()
    this.shell = new GameClientShell(message => {
      if (this.status?.isValid) this.status.string = message
    })
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

  private layout(): void {
    if (!this.content || !this.title || !this.status) return
    this.content.getComponent(SafeArea)!.updateArea()
    const size = this.content.getComponent(UITransform)!
    const width = Math.max(1, Math.min(size.width - 32, 760))
    const height = Math.max(1, size.height)
    this.title.node.getComponent(UITransform)!.setContentSize(width, Math.min(70, height * 0.2))
    this.title.node.setPosition(0, height * 0.23, 0)
    this.status.node.getComponent(UITransform)!.setContentSize(width, Math.min(120, height * 0.35))
    this.status.node.setPosition(0, 0, 0)
  }

  override onDestroy(): void {
    view.off('canvas-resize', this.layout, this)
    view.off('design-resolution-changed', this.layout, this)
    this.shell?.dispose()
    this.shell = null
  }
}
