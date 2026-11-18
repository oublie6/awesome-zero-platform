# Goal0063 Round 1 独立验收报告

**结论：FAIL。不能合并或部署。** 验收执行期间未修改生产代码、main、已有 Goal 或已有验收报告。报告完成不代表产品 Goal0063 完成。

**发布说明（2026-11-18）**：用户后续授权按原 main 流程提交、推送本报告，并取消 PR #18 的 Draft 状态。该流程调整不改变 FAIL 结论，也不表示候选代码已经修复、合并或获准部署。

## 版本与环境

- 候选：`74125d6595652ffdd19c6b5b785a8e871fede314`；基线：`e9cfd77e409b85aad1f37e351cc7e0e3d63a5211`。
- PR：https://github.com/oublie6/awesome-zero-platform/pull/18 。末次只读核对为 open、draft=true、merged=false，head 与候选一致，见 `pr-final.json`。
- 候选使用只读 detached worktree `/tmp/goal0063-acceptance-74125d6`；验收执行期间，共享主工作区保持原 main SHA `3304d43ed6e4cc099a26d22aa00d048f8a13be8d`，当时未 pull、提交、推送或切换分支。本次文档发布按用户授权先通过 `git pull --ff-only` 快进到基线 e9cfd77；没有把候选生产代码带入 main。
- CI 44022126927：4 job success，runtime/report 两项 skipped；Fair Doudizhu 44022126964：4/4 success。CI checkout merge SHA `03bc4d66bd6e52898ddb089e5dcd1e1a98fbe48b` 与候选 tree 一致：`1b87b15bfb42e40f6955eb9d382522c10b5208d8`。证据 `ci.json`、`fair.json`、`ci-checkout.json`。未把 skipped 算作通过，未重复运行整套 CI。
- Go 1.25.8、Node 22.23.4；官方 cocos-cli 固定版本 f506fa3…，Cocos 4.0.0-beta.0 engine d0a696…，Linux 原生重新导出。源码客户端150个文件核对后构建；严格声明检查与原生 BigInt 检查通过。见 `native-build.log`、`native-input-inventory.json`、`runtime-file-manifest.json`。这是 Web 运行证据，不是引擎正式发布资格或手机真机资格。
- 独立 localhost API 18863、HTTPS/WSS 29463；全新私有测试密钥、四个 QA 账号、独立 MySQL schema `goal0063_accept_74125d6`；MySQL5.7.44、Redis、两个独立 Beanstalkd1.13 节点。队列各64MiB，Redis128MiB，MySQL768MiB；编译/测试低并发顺序执行，无 swap。未使用现有 Kubernetes 数据或密钥。
- 完整牌局使用3个真实认证 HTTP/WSS 客户端、同服务端的官方 Cocos 页面及真实私有投影。早期三 Cocos 页面执行过入房/准备；**不宣称三台 Cocos 客户端完整游玩通过**。

## 验收矩阵

| 范围 | 结果 | 实际证据与边界 |
|---|---|---|
| A：真实 go-queue/dq、双独立 Beanstalk、Redis 去重/新投递ID | PASS | `queue-independent.log` 中复制去重、新ID重试通过；整个队列测试套件因 N01 退出1 |
| A：MySQL唯一截止时间权威、事务竞争、版本/座位约束 | PASS | `sql-independent.log`：7项通过，包含真实 InnoDB 锁等待跨截止时间、人工先到获胜、12并发写者仅1成功、重开/epoch takeover、实际 COMMIT OK 丢失与提交前断链 |
| A：提前、旧版本、错误座位、畸形、过期投递不改状态 | PASS | `advisory-ticket-injection.json`：两真实队列，5类消息，版本保持不变；31分钟旧 wrapper 是构造时间戳，不是31分钟实际断网 |
| A：自动不叫、不出、最小合法单张与现有日志/推送 | PASS（功能）；FAIL（时效） | `actual-auto-actions.json`、`live-outages.json`、`abandoned-no-bid-observation.json`；实际自动操作写日志、推进版本，但 N02 延迟明显 |
| A：单任务失败隔离、取消/有界运行 | FAIL | N01：一个静默队列连接可卡住串行调度 |
| A：批量调度/资源 | PASS（有界测量） | `backlog-result.json`：1000条合成未来回合，真实MySQL与双队列，4轮/36.937秒；goroutine8→8，GC后堆711256B。不是1000真实牌局或长时性能资格；无完整P95/P99/锁竞争压测结论 |
| B：实际进程 SIGKILL/冷启动保留原截止时间 | PASS | `full-game-protocol.json` 的 before/after deadline 精确毫秒与版本一致 |
| B：单队列、双队列、Redis、MySQL故障恢复 | PASS（所执行场景） | `live-outages.json` 四项；版本4→5→6→7→8，每次仅1条有效操作；双队列停止期间原版本和截止时间不变；实际 consumer 在MySQL中断时报告 invalid connection，恢复后修复 |
| B：提交成功但终局 ACK 丢失 | FAIL | N04，真实MySQL COMMIT OK丢失后BIDDING卡住；冷重启可恢复，但不能算自动恢复通过 |
| B：重复/迟到任务、终局后旧任务 | PASS（已测范围） | 真实重复投递/SQL并发；完整牌局归档后32秒 digest与版本不变，deadline不存在。没有独立双API进程争主资格结论 |
| C：三视口17/20张牌、排序、座位、地主/底牌 | PASS（已截图项） | 6张截图与 `visual-metrics.json`，真实牌、全部节点数量和降序；18/19张在对局推进中出现，未逐尺寸独立留图 |
| C：真实触摸抬起、重复取消、按钮稳定、短程节点/定时器 | PASS（有界探测） | `touch-probe.json`：三视口；12次额外重绘，节点122→122，250ms计时器始终1个。不是长时监听器/堆泄漏资格 |
| C：当前出牌、合法操作按钮 | FAIL | N07：Base64出牌解析为空，真实跟牌座位“不出”禁用 |
| C：竖屏可用性 | FAIL（P2） | N06，390×844实际按钮高度约17.28 CSS px，文字与控件过小；不援引未规定的像素合规阈值 |
| D：真实三账号完整发牌/叫地主/轮转/自动操作/服务端结算归档 | PASS（服务端闭环） | `continue-actual.json`：牌局 g63-full-e2738f0713f4f99386149f78，finalVersion61，地主/胜者1，积分[12,-6,-6]；三账号证据一致；旧任务未改归档 |
| D：重连、倒计时不因本地界面重置 | FAIL | 服务端重启保留时间，但 N03 本地离开/重进04s→30s，同版本同截止时间 |
| D：完整原生界面出牌/不出/终局体验 | FAIL | N07阻断合法不出，N05终局异常；不能用协议客户端完整牌局替代原生界面通过 |
| D：私有手牌/非成员隔离 | PASS（所测边界） | 私有读取非成员403；第四账号0条牌局推送；3个玩家各54条私有快照的position分别匹配本人。`socket-privacy-audit.json`。活跃公开投影仅含规则/座位/公开牌/摘要/时钟等；不是完整信息流形式化证明 |
| 部署、迁移/回滚与混合版本运行 | BLOCKED | 明确未授权改现有K8s；只审查清单和禁用旧机制方案，没有应用DDL、滚动混部或PVC恢复实验，不能算部署PASS |
| 长时断网/高负载/真机 | BLOCKED/未验证 | 本轮实际故障为短时并跨回合截止时间；无31分钟墙钟断网、长期负载或真实手机触摸资格。已有明确P1，不继续扩张破坏性验证 |

## 可直接交给 ChatGPT 的修复清单

没有发现已证实的 P0；以下6个P1阻止通过。修复应保持模块边界、MySQL截止时间权威、提交前后幂等与epoch保护，不以延长时间或屏蔽异常掩盖问题。

### N01 — P1：发布操作不遵守 context，可阻断其他牌局

- 文件：`server/platform/delayqueue/dq.go` 的 `Publisher.At`；`server/apps/app-api/internal/bootstrap/doudizhu_turn_dq.go` 的 `dispatch/startDoudizhuTurnDQ`。
- 复现：两个真实节点运行时暂停其中一个，保持TCP连接；使用200ms context发布。750ms后仍未返回，恢复节点后才返回。`queue-independent.log`/`queue-independent-status.json`。
- 预期：有界失败/取消；一节点静默不阻断整个修复扫描。实际只在进入上游At前检查一次ctx，上游网络Put没有对应操作deadline，外层10秒context不能中断它。
- 修复验收：验证真实静默连接、部分副本成功、取消与关闭；失败任务与其他hand隔离，不能只加调用前/后ctx检查。

### N02 — P1：正常超时在00s后约10秒才执行

- 文件：上述 `Publisher.At`、worker `process`，以及 `server/business/doudizhu/infrastructure/mysqlstore/live_turn_deadline.go` 的 `MarkTurnDelivery`。
- 复现：正常运行20秒叫地主回合。原截止1794946344449，首次自动提交1794946355968，晚11.519秒；随后自动不叫间隔约32秒。证据 `abandoned-no-bid-observation.json`。
- 原因链：固定上游秒级延迟向下取整使任务略早到达；process忽略提前任务，而成功投递修复游标已设到deadline+10秒；用户此时已被截止时间门禁拒绝。
- 修复验收：处理提前投递并保持绝不早提交；测真实队列正常情况下的截止到提交延迟，不通过重设截止时间或放宽人工门禁隐藏空窗。

### N03 — P1：本地返回/重进牌桌重置倒计时

- 文件：`clients/fair-doudizhu-cocos/assets/scripts/bootstrap/GameTableView.ts` 的 render/countdownCapture；`GameBootstrap.ts` 的页面创建/销毁与OpenTable。
- 复现：真实回合退出牌桌再进入；显示04s→30s，版本均7，MySQL deadline均1794947272561。`reentry-clock.json`。
- 预期：同回合绝对时间持续减少。实际新view以旧serverTimeMillis重新捕获performance.now。
- 修复验收：时钟锚点跨view生命周期保持正确，覆盖BackToRoom/OpenTable、resize、选牌、同版本刷新、重连；只测试同实例render不充分。

### N04 — P1：终局提交响应丢失后没有持久化收敛路径

- 文件：`live_turn_deadline.go` 的 updateTurnDeadline；`doudizhu_turn_dq.go` 的 process/ErrNoRows分支；相关 lifecycle AfterBid/AfterPlay 完成链路。
- 复现：真实三人新局前两人不叫，第三人自动不叫前让MySQL代理丢弃服务端实际COMMIT OK。日志writer已到4、三条bid均已落盘，deadline已删；恢复网络并等12秒，domain仍BIDDING且archive0。SIGKILL冷启动后才ABORTED/archive1。`terminal-ack-fault.json`、`api-terminal-ack-before-kill.log`。
- 预期：未知提交结果/生命周期失败能持久化重试并自动收敛。实际终局先删唯一扫描游标，ErrNoRows被视作已归档，无自动重试入口。
- 修复验收：持久化完成/生命周期修复，覆盖NO_LANDLORD以及获胜终局提交、归档/房间完成前后故障；保留原操作ID、exactly-once有效状态和epoch fencing。实际获胜终局的同类ACK故障本轮未单独注入，不能冒称已复现。

### N05 — P1：终局投影触发Cocos异常，结算/核验入口消失

- 文件：`GameBootstrap.ts` 的终局projection强转与table render；`GameTableView.ts` 的 publicPlaying（`view?.public.playing`）。
- 复现：完整真实对局结束后同步牌桌。投影仅有terminalEvidence，没有public；报 `Cannot read properties of undefined (reading 'playing')`。布局止于NoHandSeats，无结算摘要和VerifyFinalEvidence。`terminal-ui-probe.json`、`terminal-ui-stable.png`。
- 修复验收：区分live/terminal投影，实际显示胜者、积分和可操作核验入口；覆盖终局推送、重新查询与重进页面。
- 归因：关键危险读取/投影强转在基线已存在，是候选完整集成验收中证实的遗留阻断项；不声称全部由本PR新引入。

### N07 — P1：真实Go出牌编码与UI不匹配，合法“不出”被禁用

- 文件：`TableGeometry.ts` 的 publicCardCodes；`GameTableView.ts` 的 publicPlaying与PassTurn启用条件；真实Go playing Snapshot/carddeck.Card JSON边界。
- 复现：真实自动领牌后，private/public里的leadingCards为Base64字符串（如`Ag==`），不是数字数组。玩家position2、playingSeat2，已有合法单张领先，但0个PublicTrick节点，PassTurn touch=false。`trick-ui-probe.json`、`public-trick-wire.json`、`trick-ui-actual.png`。
- 预期：当前出牌显示，合法跟牌时可不出。实际数组专用转换返回空，界面误判无领先牌。
- 修复验收：在明确版本兼容的协议边界正确规范/解析真实Go数据；使用实际HTTP/WSS序列化结果做契约和Cocos交互回归，不能只造数字数组测试。检验领牌不能pass、跟牌能pass和真实出牌区。
- 归因：当前出牌转换问题已存在于基线；本PR新增 `myTurn && playing.cards.length>0` 把它进一步变成合法不出禁用，属于本PR新增交互回归。

### N06 — P2：竖屏逻辑画布缩放导致文字/触控区域过小

- 文件：Cocos画布/视口适配与GameTableView/TableGeometry尺寸策略。
- 复现：390×844，1280逻辑宽缩至390；出牌/不出约61.70×17.28 CSS px，返回30.85×14.40；见 `visual-metrics.json`、`table-17-390x844.png`、`table-20-390x844.png`。
- 预期：竖屏可读、正常手指可操作。实际空间大量空置而控件/文字极小。精确CDP命中通过不等于真实手指可用。
- 修复验收：按实际屏幕尺寸检验字体、触控区域、完整手牌和固定操作行；保留横屏资格。不把未规定的某一像素阈值当成既有合规要求。此项包含基线适配遗留问题。

## 第二轮反向审查与测试可信度

主要集成后，再检查了队列先删除再回调、终局删除游标、MySQL锁后时间检查、未知COMMIT、deadline毫秒/微秒、旧版本修复更新条件、客户端投影形状、视图销毁重建和旧/新超时机制切换。N04、N05、N07均由实际状态和代码相互印证；不是仅凭mock推断。范围也覆盖startup恢复顺序、有限64×64扫描、部署禁混机制和当前模块边界。未把未复现的额外风险扩大为确定缺陷。

证据存在以下测试脚本问题，全部保留原日志，不隐瞒：早期HTTPS代理漏转manifest路由；多页面导致贡献错过未来信标窗口；异步房间恢复门禁过早；重连CDP后未重新开启触摸模拟；一个脚本被SIGTERM中断；SQL探测误用不存在字段；首次backlog脚本误把正常10秒批次预算当测试失败，且cleanup在db.Close之后。随后仅修改外部fixture并核对真实状态，未修产品代码。第一次backlog残留1000条合成数据已清理，最终1000条测量通过且合成行计数归零。

另一次故障准备阶段实际HTTP命令超过3秒返回500，新局正常ABORTED；保存 `live-outages-attempt1.*`。后续测试将独立API CPUQuota从1提高到2，并在客户端以完全相同命令ID/内容精确重试未知结果；不是生成新随机贡献覆盖已确认状态。该准备超时不算DQ故障通过，也未据此确定新的候选回归。

原脚本 `continue-actual.json` 有“terminal native render”PASS，仅检验phase并截图，后来稳定探测证实UI失败，故本报告明确判N05 FAIL；早期触摸FAIL后来确认模拟未启用，最终以显式触摸探测结果判定。不能直接把某一原始脚本总标签当产品结论。

## 命令与主要证据

外部fixture根目录 `/root/goal0063-acceptance-74125d6`，不在候选源码中。Go检查均 `-p 1 -parallel 1 -count=1`：

- `go test -tags=integration -overlay=.../independent-overlay.json -run 'TestG63|TestDQTurnDeadlineEnforces' -v ./business/doudizhu/infrastructure/mysqlstore`：exit0。
- `go test -overlay=.../independent-overlay.json -run TestG63 -v ./platform/delayqueue`：exit1，N01。
- `go test -overlay=.../backlog-overlay.json -run TestG63ActualThousandRowDispatchBacklog -v ./apps/app-api/internal/bootstrap`：exit0；合成未来回合、真实SQL/双队列，无业务callback，不冒充1000局完整压测。
- 真运行脚本：`full-game-protocol.cjs`、`controlled-full.cjs`、`continue-actual.cjs`、`terminal-ack-fault.cjs`、`live-outages.cjs`、`native-17.cjs`、`touch-probe.cjs`、`terminal-ui-probe.cjs`、`trick-ui-probe.cjs`。失败尝试和后续有效证据分别保存。私有配置、账号和原始socket记录不应公开传播。

截图：

- [20张桌面](0063-round1-evidence/table-20-1280x720.png)、[20张窄横屏](0063-round1-evidence/table-20-760x390.png)、[20张竖屏](0063-round1-evidence/table-20-390x844.png)。
- [17张桌面](0063-round1-evidence/table-17-1280x720.png)、[17张窄横屏](0063-round1-evidence/table-17-760x390.png)、[17张竖屏](0063-round1-evidence/table-17-390x844.png)。
- [合法不出被禁用](0063-round1-evidence/trick-ui-actual.png)、[终局异常稳定截图](0063-round1-evidence/terminal-ui-stable.png)。

## 环境保护、收尾和重新验收前置条件

`protection-audit.json`：现有K8s Deployment/Service/ConfigMap/PVC名称、UID及spec/data一致；Secret名称/UID/resourceVersion一致，未读取其值；4个原Pod同UID、Running、0重启。没有对业务数据库/PVC内容做字节级比对，不能夸称数据哈希完全相同。没有向K8s执行写操作。

验收专用API、Web、浏览器和故障代理已停止；专用双队列及原本停止、为本轮临时启动的本地MySQL/Redis在收尾恢复为停止。保留专用schema、binlog、构建和证据用于复现，不drop/flush已有数据。最后一个独立故障手牌可保持持久化PLAYING用于后续恢复调查，不属于现有K8s业务。用户既有 `azp-game-test-forward.service` 保持运行。

ChatGPT应按上述清单修复，更新必要协议/回归测试，提交新的明确候选SHA后申请受影响项与整体回归验收；本报告不代替开发；本次只授权报告资料提交main及取消Draft，不将其解释为候选代码合并或部署授权。至少所有P1消除、原生Cocos出牌/不出/终局闭环通过后，才可讨论合并。部署另需不可变镜像、资源/PVC、备份与加性DDL预检、旧活跃局处理、禁混超时机制和可验证回滚；本轮未授权执行这些部署动作。

## 已推送资料与本机原始证据边界

公开给仓库协作者的筛选资料位于 [0063-round1-evidence](0063-round1-evidence/README.md)，包含8张截图、关键结构化故障/交互/性能结果和CI job状态。正文中未附链接的原始日志及脚本名称指本机验收目录；没有把所有原始目录上传。账号、密码、令牌、测试私钥、私有服务配置、浏览器profile和原始socket明文记录均未提交。历史Draft状态快照 pr-final.json 保留原始事实；发布时取消Draft由用户另行授权，不回写历史证据。
