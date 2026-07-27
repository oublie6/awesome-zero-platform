import { _decorator, Component, Label, Node, UITransform, Vec3 } from 'cc'

const { ccclass } = _decorator

@ccclass('GameBootstrap')
export class GameBootstrap extends Component {
  start(): void {
    const title = new Node('Title')
    title.addComponent(UITransform).setContentSize(800, 80)
    const label = title.addComponent(Label)
    label.string = 'Fair Doudizhu'
    label.fontSize = 52
    title.setPosition(new Vec3(0, 180, 0))
    this.node.addChild(title)
  }
}
