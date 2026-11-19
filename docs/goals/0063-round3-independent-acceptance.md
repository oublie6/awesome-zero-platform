# Goal0063 Round 3 独立验收报告

日期：2026-11-19。**结论：FAIL。Round 2 三项遗留问题在本轮执行范围内均已修复；额外发现 R3-N01/P2：未知结果后核对按钮不随当前 pending 状态刷新。暂不进入合并或部署准备。**

## 基线、范围与证据复用

- 固定候选 `3502b6acedc128d37b63a34b4f5286a9a9e642e2`，PR [#18](https://github.com/oublie6/awesome-zero-platform/pull/18)，分支 `goal0063-dq-turn-experience`。独立 detached worktree `/tmp/goal0063-round3-3502b6a`；主工作区 `main` 初始干净，`git pull --ff-only` 已同步。没有修改候选代码、合并 PR 或部署。
- [CI 44429663523](https://github.com/oublie6/awesome-zero-platform/actions/runs/44429663523)：4 个执行 job success，runtime/report skipped；[Fair 44429663545](https://github.com/oublie6/awesome-zero-platform/actions/runs/44429663545)：4/4 success。实际 checkout 是 merge `d8fe9e1d7565d847ce62d320a52436cc87a57fbb`，与候选 `server/clients/scripts/.github` 输入相同。[源码绑定](0063-round3-evidence/ci-binding.json)。
- 按 AGENTS、规范原则及 Goal0063 检查 Round2→Round3 的15文件差异和必要调用链。归属为 Doudizhu lifecycle/runtime/MySQL、gamecore 归档合同及 Cocos 显示；没有新增跨业务私表访问、依赖或部署单元。持久日志/epoch 是恢复权威，Redis/队列仍是辅助；重放校验和冲突归档拒绝保留，未用旧内存覆盖已提交结果。
- 复用同候选 CI 的 Go/架构/客户端、Race、MySQL集成结果；队列发布、游戏规则和 portable packages 与 Round2 无差异，复用[上轮](0063-round2-independent-acceptance.md)的真实队列故障、性能及完整协议检查。不重复全量构建/CI/性能测试；本轮另执行下述真实故障、竞争与原生页面。
- 复用专用 MySQL5.7.44、Redis8.8.0、Beanstalkd1.13×3；只使用 Round2 的隔离 B/perf schema、QA账号、保护根和测试TLS。损坏现场 A 保留。Go1.25.8、Node22.23.4，`GOMAXPROCS=2`、Go `-p 1 -parallel 1`；私有夹具及原始日志在 `/root/goal0063-round3-3502b6a`（E）。不含生产账号或集群故障注入。

## Round2 遗留项

| 项目 | 结论 | 本轮实际证据 |
|---|---|---|
| R2-N01 / P1 终局持久化恢复 | **PASS（所测范围）** | 真 MySQL COMMIT OK 丢失的全不叫和正常获胜均自动收敛；冷启动不改变归档版本/摘要/动作数。没有依赖重启补齐缺失归档。 |
| R2-N03 / P2 终止游标清理 | **PASS（所测范围）** | 真取消且归档 INSERT 被条件触发器拒绝时，domain已ABORTED、archive0、游标仍在；解除故障后自动归档并退休。真实旧双队列任务及9s空闲观察没有恢复牌局或继续重投。 |
| R2-N02 / P2 竖屏字号 | **PASS（字号/已显示控件）** | 官方实际 Web 导出，三视口实际标签/截图/交互。手机按钮约17.68～20 CSS px；核对按钮出现后可实际精确重试。其“何时出现”的额外缺陷单列 R3-N01，不能由字号通过抵消。 |

**终局故障：** 三人全部不叫局 `g63-full-d67dd34e97e984d75980fe7b`：前两次正常提交，第三次真实自动不叫的 COMMIT OK 被代理丢弃；三条动作/live4，自动得到 ABORTED/archive5、游标0。后续冷启动仍是同一v5和摘要，未重现 Round2 的错误v4/启动冲突。[故障与SQL](0063-round3-evidence/no-landlord.json)、[冷启动核对](0063-round3-evidence/no-landlord-restart.json)。

正常三账号局 `g63-full-2ce4dae28ed6a7409af9bdfa`：真实认证/入房/准备、CSPRNG贡献、签名上下文/HPKE/未来drand发牌、bid/play/pass推进；最后一手 COMMIT OK 丢失，客户端得到未知结果错误。无需重启即 COMPLETED/live60/archive60、59条动作、游标0；归档摘要 `b2efa8b32534af59dfd83634121d52208b062d5c617c2c2812af2b21244e9b24`。之后实际停止/启动候选，epoch4→5、版本/摘要/动作数不变；真实 Cocos 终局显示胜者1、积分[12,-6,-6]并请求核验。这是三协议客户端完整业务局及原生关键操作，未声称三个Cocos客户端全手操作。[故障/恢复](0063-round3-evidence/completed-faults.json)。

缺失终局归档的启动补偿另复用固定候选CI的 `TestExpiredHandColdJournalReceiptAndArchiveRecovery`：`mysql_integration` 在 `APP_API_INTEGRATION=1` 下运行该包的全部integration测试；生产 `durable_live_recover.go` 与该测试相对Round2均无变化。本轮新增的是两种真实末操作ACK损失的自动恢复，以及恢复后实际冷启动的版本/摘要核对；未重跑Round2的修复中SIGKILL+启动domain ACK丢失复合矩阵。

**归档失败与取消：** 同一个既有房间的新手 `g63-full-1ae471ce039f71c7e1c4464a`，仅该hand的 archive INSERT 被真实 MySQL SIGNAL 拒绝。原回合仅剩不足2s，一条合法自动不叫先提交（live1→2/actions0→1），随后参与者取消赢得终止、epoch1→2。实际观察 ABORTED/live2/archive0、BIDDING/v2游标保留。恢复后自动 archive3、游标0、动作仍1。没有为了测试抹去已提交自动动作或延长期限。两个真实节点接受旧任务后，归档及动作不变；空闲9s两节点 total-jobs 不增长。[取消观测及恢复](0063-round3-evidence/cancel-retry.json)、[队列空闲](0063-round3-evidence/queue-idle.json)。

## 终局竞争

外部 Go overlay 使用真实 service/runtime/journal/MySQL archive，测试数据为独立确定性夹具，不把它冒充真实玩家公平准备。没有修改仓库测试或生产源码。

| 顺序/场景 | 实际结果 |
|---|---|
| 最后三次不叫已提交，再中止/过期 | 两种冲突终止均拒绝；domain保持BIDDING待合法收敛，writer epoch不增加 |
| 获胜最后一手已提交，再中止/过期 | 保持真实COMPLETED日志/归档；冲突终止拒绝，epoch不增加 |
| 参与者中止先赢，正常bid并发 | 真实writer行锁造成2个等待者，中止先提交；旧live Append返回fenced，无动作提交；终止仅一次，精确重复不增加epoch |
| 内部过期先赢，正常bid并发 | 同样2个实际锁等待、合法过期提交一次、旧writer拒绝、重复幂等 |

前三子例：[日志](0063-round3-evidence/final-terminal-guard.log)（exit0）；过期子例：[日志](0063-round3-evidence/expiry-guard.log)（exit0）。实际参与者 `hand.cancel` 对已发牌调用的是 `AbortHand`，不是禁止在已发牌阶段使用的 domain `CancelHand`。本轮最终测试已按真实适配路径核对，未把早期 WrongPhase 拒绝误当新writer guard的证明。超时自动不叫→全不叫归档另有真实队列场景；没有宣称普通回合超时应该直接令整局 EXPIRED。

## 原生 Cocos 与新增问题

一次实际官方 Linux cocos-cli `f506fa3…` / engine `d0a696…`（4.0.0-beta.0）Web导出，实际生成声明严格检查及8个JS审计通过。150个tracked客户端输入已核对；CLI仅重排package.json字段/缩进，JSON语义相同。共享引擎仍有上轮已记录的5个内置资产.meta差异，无引擎代码变更。保留默认skybox/FBX及non-assets tsc警告；没有重建外国OS工具链。[构建日志](0063-round3-evidence/native-build.log)、[来源/产物哈希](0063-round3-evidence/provenance.json)。Go二进制未嵌入VCS字段，来源绑定是干净候选工作区的实际构建命令/cwd和产物哈希，不声称二进制自报SHA。

| 390×844按钮 | 实测CSS字号 |
|---|---:|
| 返回 / 同步 | 18.20 / 17.68 |
| 不叫 / 1～3分 | 19.76 |
| 出牌 / 不出 | 20.00 |
| 核对上次操作 / 按新快照放弃 | 18.72 |
| 核验结算 | 19.76 |

760×390导航约14.17～14.59、核对15.01、出牌16.03px；三尺寸操作文字可读，没有按钮文字裁剪或按钮互相遮挡。真实触摸提交叫地主、精确重试、选牌出牌和合法不出；返回/重入/同步执行。终局三个尺寸触摸均实际发出证据请求。续验曾使用鼠标返回/重入以恢复视图，最终原生操作记录无鼠标fallback；不冒充手机硬件。首张桌面竞叫的metric取样碰到resize过渡，桌面字号结论以已稳定的unknown/playing/terminal测量和截图为准。检查限于本轮按钮目标，非所有提示、动画和长期界面资格。

| 状态 | 1280×720 | 760×390 | 390×844 |
|---|---|---|---|
| 叫地主 | [图](0063-round3-evidence/bidding-1280x720.png) | [图](0063-round3-evidence/bidding-760x390.png) | [图](0063-round3-evidence/bidding-390x844.png) |
| 核对控件（重入后） | [图](0063-round3-evidence/unknown-1280x720.png) | [图](0063-round3-evidence/unknown-760x390.png) | [图](0063-round3-evidence/unknown-390x844.png) |
| 出牌 | [图](0063-round3-evidence/playing-1280x720.png) | [图](0063-round3-evidence/playing-760x390.png) | [图](0063-round3-evidence/playing-390x844.png) |
| 真终局 | [图](0063-round3-evidence/terminal-1280x720.png) | [图](0063-round3-evidence/terminal-760x390.png) | [图](0063-round3-evidence/terminal-390x844.png) |

**R3-N01 / P2：未知结果不会刷新核对操作控件。** 新发现，**不是本轮新引入**：GameBootstrap/GameHandActions在修复差异中未改变。实际原生页真实叫地主已提交；在浏览器消息接收边界丢弃该稳定请求的响应及后续hand通知，等待原15s期限，无伪造服务端投影。随后 `actions.needsReconciliation=true`，页面显示“command timeout; outcome unknown”，但root没有 `RetryUnknown` / `DiscardUnknown`，仍是旧BIDDING交互。缩放也没有解决。返回房间再进入，正常full render才出现两按钮；点击核对成功、同步权威状态后可继续实际出牌/不出。

根因：`GameBootstrap.bid/play/pass`异常路径只调用`notice`；`GameTableView.setNotice`用先前缓存的`TableDisplay`做layout，其`needsReconciliation`仍为false。`GameHandActions.pending`变化没有触发新TableDisplay。于是按钮字体修好，却不能在需要时可靠显示核对入口。[状态前后](0063-round3-evidence/unknown-render-gap.json)、[缺失按钮现场，实际1280×757](0063-round3-evidence/unknown-missing-controls-1280x757.png)、[原始失败](0063-round3-evidence/native-targeted-initial.json)。独立消息损失检查与普通字号检查分别报告，不将重入后的成功覆盖最初失败。

**交给ChatGPT的最小、去重修复清单只有一项：** pending/unknown状态变化时刷新牌桌当前显示数据，让核对入口即时出现；保留原请求ID/参数和未知结果保护，不能直接清pending或换ID重发。增加Bid/Play/Pass回包丢失/断线、明确拒绝、无需重入即显示及精确重试后的控件清理回归。终局恢复、取消游标和字号不用因本报告重新设计。

## 命令、限制与收尾

本轮命令从候选server执行，E表示上述私有目录：

```text
go build -p 1 -trimpath -o E/app-api ./apps/app-api
go test -p 1 -parallel 1 -count=1 -tags=integration -overlay=E/guard-overlay.json -run TestR3CommittedTerminalAndWriterRace -v ./business/doudizhu/infrastructure/mysqlstore
go test -p 1 -parallel 1 -count=1 -tags=integration -overlay=E/guard-overlay.json -run TestR3CommittedTerminalAndWriterRace/expire-wins -v ./business/doudizhu/infrastructure/mysqlstore
```

`no-landlord.cjs`、`completed-faults.cjs`运行真实故障/业务；`native-build.cjs`执行官方导出，`native-terminal.cjs`、`native-targeted*.cjs`执行真实页面；取消续验与`queue-idle.cjs`保留完整SQL及消息计数。构建、测试退出码和外部竞争测试源码随证据保存。按用户要求只检查变化及关键缺口，没有再跑全仓审计或性能矩阵。

夹具的错误、SIGTERM中断、缺PROCESS观察权限、已发牌Cancel/Abort区别、直接连接绕过代理、CDP续验触摸问题、resize取样以及Beanstalk stats帧解析问题均保留在[夹具记录](0063-round3-evidence/fixture-notes.md)。可选的domain-ACK选择性注入因prepared statement复用识别失败，**实际丢失0个，不计通过**；没有为此再跑新故障矩阵。原生UI的后续原始文件带BLOCKED标签，是该额外注入断言失败，已完成的字号/关键交互与真正UI缺陷分别按观测列出，不改写原始标签。取消发生时的合法自动bid以取消后观测为基线，未把正常竞争误报为终止后动作。

未重新验证真机/微信SDK、生产容量SLO、长期资源泄漏、整个客户端所有界面、Kubernetes滚动部署、迁移/回滚及存储灾难恢复。本轮所测是隔离进程/MySQL完整时的终局恢复，不是跨节点零中断或引擎正式发布资格。将来批准测试部署仍须备份/历史数据和密钥保护、迁移预检、节点故障与回滚验证；本报告未授权执行。

只提交本报告及必要证据到main；不修改current Goal定义/完成状态，Goal0045/0046及历史报告保持原状。所有本轮专用进程和复用的专用依赖容器在收尾停止，数据库/卷/密钥/原始证据保留；未操作现有Kubernetes工作负载、PVC、用户转发或生产配置。最终资源状态见[清理记录](0063-round3-evidence/cleanup.json)。本报告经过一次范围/证据自检；没有另一位审阅者，不冒充额外独立复核。
