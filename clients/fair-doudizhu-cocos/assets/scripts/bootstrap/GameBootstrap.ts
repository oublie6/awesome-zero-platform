import { _decorator, Color, Component, Label, Node, UITransform, Vec3, view } from 'cc'
import { GameClientShell } from '../game/GameClientShell'

const { ccclass } = _decorator

/** Code-created scene; no platform SDK or business rules in Cocos components. */
@ccclass('GameBootstrap')
export class GameBootstrap extends Component {
  private shell: GameClientShell | null = null
  private title: Label | null = null
  private status: Label | null = null

  start(): void {
    const root = this.node
    const transform = root.getComponent(UITransform) ?? root.addComponent(UITransform)
    const width = Math.max(320, view.getVisibleSize().width)
    const height = Math.max(240, view.getVisibleSize().height)
    transform.setContentSize(width, height)
    const title = new Node('GameTitle')
    title.addComponent(UITransform).setContentSize(Math.min(width - 24, 760), 70)
    this.title = title.addComponent(Label)
    this.title.string = '公平斗地主'
    this.title.fontSize = 42
    this.title.color = new Color(240, 215, 155, 255)
    title.setPosition(new Vec3(0, Math.min(150, height * 0.23), 0))
    root.addChild(title)
    const status = new Node('GameStatus')
    status.addComponent(UITransform).setContentSize(Math.min(width - 24, 760), 100)
    this.status = status.addComponent(Label)
    this.status.string = '客户端初始化中…'
    this.status.fontSize = 22
    this.status.lineHeight = 30
    this.status.color = new Color(224, 234, 230, 255)
    status.setPosition(new Vec3(0, 0, 0))
    root.addChild(status)
    this.shell = new GameClientShell(message => {
      if (this.status?.isValid) this.status.string = message
    })
    void this.shell.initialize()
  }

  onDestroy(): void {
    this.shell?.dispose()
    this.shell = null
  }
}
