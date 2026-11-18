# Goal0063 Round 2 独立验收报告

**结论：FAIL。不能进入合并或部署执行阶段。** 原七项缺陷中五项在下述实测范围通过，N04/P1、N06/P2 未消除；另确认取消牌局后截止游标持续重投的 P2。没有确认 P0。本报告完成独立验收任务，不代表产品 Goal0063 完成。

## 固定版本与环境

- 候选始终为 `099af4eaa117628a6337841928cbbbc5f13697c1`，PR [#18](https://github.com/oublie6/awesome-zero-platform/pull/18)，分支 `goal0063-dq-turn-experience`；末次只读核对仍 open、draft=false、merged=false，head 与候选一致。
- 按本轮明确授权，在 detached worktree `/tmp/goal0063-round2-099af4e` 验收。主工作区 `main` 初始干净、同步于 `16ac3e2f15e9c7e2051e8f945c09dae3908217fa`；候选源码没有修改，没有合并、部署或写入现有 Kubernetes。
- [CI 44225458135](https://github.com/oublie6/awesome-zero-platform/actions/runs/44225458135)：4 个执行 job success，runtime/report 两项 skipped；[Fair Doudizhu 44225458131](https://github.com/oublie6/awesome-zero-platform/actions/runs/44225458131)：4/4 success。实际 checkout 为 merge `de6aa816ae41ae3bc413595f234e9899303c37ed`；与候选的 `server/clients/scripts/.github` 输入无差异。原 CI/Race/完整集成结果复用，另独立执行下述修复差异、真实故障和业务测试。[绑定与 job 证据](0063-round2-evidence/ci-binding.json)。
- Linux x86_64、约 7.8GiB RAM、无 swap；Go1.25.8、Node22.23.4，Go `GOMAXPROCS=2`、`-p 1 -parallel 1` 顺序运行。实际 go-zero1.10.2、go-queue1.2.2、go-beanstalk0.2.0。
- 全新专用 MySQL5.7.44、Redis8.8.0、Beanstalkd1.13 ×3。MySQL768MiB/1.5CPU，Redis128MiB，队列各64MiB；localhost API18864/18865、HTTPS/WSS29464。独立 schema A `goal0063_r2_099af4e` 保留损坏归档现场；B `goal0063_r2b_099af4e` 继续真实业务；perf schema 单独测调度，避免清除 B 的失败游标或被后台 worker 干扰。没有复用生产账号、密钥、数据库或队列。
- 候选实际 Go 二进制 VCS revision/hash、150个 tracked 客户端输入清单及导出文件哈希已保存。官方 Linux CLI 固定 f506fa3…、Cocos4.0.0-beta.0 engine d0a696…，实际重新导出 Web，真实生成声明严格检查与8个JS审计通过。CLI tracked 代码干净；缓存引擎工作区有5个内置导入资产 `.meta` 差异，无引擎代码 diff，未重置共享工具链。默认 skybox/FBX 工具及 non-assets tsc PATH 警告保留；最终独立严格声明检查通过。这是实际2D Web证据，不能扩展为引擎正式发布、WeChat或真机资格。
- 使用全新测试保护根、签名/揭示密钥、TLS信任和真实注册/密码登录 QA 账号。Node/Chromium 正常验证本地测试 TLS；没有 token-in-URL。公平准备使用原客户端 CSPRNG、签名上下文、HPKE、真实未来 drand Quicknet/BLS，无模拟后端、假发牌或延长20s叫地主/30s出牌期限。

## 原七项逐项复验

| 原缺陷 | 结果 | 实际证据与边界 |
|---|---|---|
| N01 DQ发布阻塞/P1 | **PASS（已测范围）** | 真队列静默 TCP、200ms期限、取消、默认2s失败、单节点暂停、三节点两副本仲裁、实际 Put 成功但两个 ACK 丢失、精确重试/Redis去重、新deliveryID均执行。64并发超时后goroutine回基线；真实同批坏牌局后健康牌局约2.11s完成复制。有发布连接且节点暂停时，API约1.044s退出0、连接清零。不是无限压力或所有网络故障资格。 |
| N02 自动操作延迟/P1 | **PASS（已测范围）** | 同一真实牌局的正常自动不叫、最小合法单张领牌、合法不出均写入journal。截止到SQL提交分别1231/1813/1686ms，均不早提交，未见原约10s空窗。真实双队列提前/错版本/错座位/畸形/旧wrapper投递未提前推进；SQL锁等待跨期限、人工先赢、12并发回调仅一次提交通过。小样本不是P95/P99生产SLO，秒级队列仍有可见量化延迟。 |
| N03 倒计时重置/P1 | **PASS（已测范围）** | 实际选牌/取消、局部刷新、返回重进、三尺寸切换、真实WSS同步/重连执行。重进同版本24s→22s；转发真实Go WSS后，将捕获的同版本旧时间戳帧原样重复发送两次，18s→16s，没有回跳。新版本/回合沿真实服务端新截止时间显示。未代替微信后台/真机时钟资格。 |
| N04 终局ACK与收敛/P1 | **FAIL** | NO_LANDLORD：live4后归档旧状态v4，正确应为v5，冷启动冲突退出。COMPLETED：live61/domain完成/标记已删，但归档0，超过一分钟仍不收敛，重启才补出。另真实修复中崩溃、2个SQL锁等待、epoch1→3和定向COMPLETED domain ACK损失执行；第二次启动后数据收敛，首次退出1，不能写成单次启动可用性PASS。 |
| N05 终局界面/P1 | **PASS（功能范围）** | 实际不含live `public`的终局投影正常显示胜者3、积分[-3,-3,6]；出牌/叫地主控件消失。终局推送后的显示、核验按钮真实HTTP请求、手动“同步”HTTP200、返回重进及全新Cocos页面恢复v61均执行，无页面异常。合法非成员对private/evidence各403。竖屏文字问题仍归N06，不能据功能通过宣称完整体验合格。 |
| N07 编码与合法不出/P1 | **PASS（已测范围）** | 使用实际Go HTTP/WSS Base64字段；例如v20、当前座位2、本人座位2、`CQ==`时真实PublicTrick节点可见，不出启用并实际成功提交；领牌时禁用不出。实际原生单张出牌、切换回合、重连继续通过。旧数字数组、54个规范ID及畸形Base64/越界/类型/长度拒绝复用候选CI与源码核对，不冒充旧服务器实际运行或所有牌型逐一原生截图。 |
| N06 竖屏适配/P2 | **FAIL** | 官方实际产物17/20张手牌在1280×720、760×390、390×844留图；真实选牌/取消、按钮触摸命中执行。竖屏按钮面积已增大，但返回/同步/核验文字均仅5.484375 CSS px，实际截图不可读。不能以几何测试或精确CDP命中代替可用性。 |

结构化记录：[正常业务与原生交互](0063-round2-evidence/nominal-business-native.json)、[真实字体/触摸/生命周期](0063-round2-evidence/native-metrics.json)、[真实SQL检查](0063-round2-evidence/sql-independent.log)、[队列Race/真实故障](0063-round2-evidence/queue-final-race.log)。

## 必须交回 ChatGPT 的去重修复清单

### R2-N01 / P1：N04修复未正确恢复终局，且过早退休修复标记

所有者：Doudizhu lifecycle/runtime/journal 协调；涉及 `application/lifecycle/supervisor.go:213`、bootstrap `doudizhu_turn_dq.go`、`gamecore/runtime.go:317` 的 Abort，以及 `mysqlstore/live_turn_deadline.go`。

**场景A，真实三人全部不叫：** 前两次不叫成功，第三次由真实队列自动提交；MySQL代理只丢实际 COMMIT OK。持久化三条bid、writer live4。新修复路径把domain变为ABORTED，却从旧内存v3执行Abort，生成archive v4并清除标记。正常三次不叫是live4，再Abort一次应archive v5；正常domain ACK损失控制局确为archive5。实际冷启动重放到live4后试图生成archive5，报 `game final archive conflicts with an existing record`，进程退出1，API无法就绪。

原因：`ApplyCommitted` 遇未知提交结果会保留旧内存并 quarantine；`LiveDirectory.Abort` 没有对应quarantine检查。`RepairTerminal` 在没有先按journal解决未知结果时直接终止，随后又以domain terminal当作已收敛。一个坏归档足以阻断该应用进程的模块启动，不是只有倒计时显示问题。

**场景B，正常获胜最后一手：** 真实完整牌局推进到live60，最后一张真实牌使live61已提交，丢该COMMIT OK。随后domain已COMPLETED、60条live action仍在，修复标记已清除，archive行数却为0；实际观察超过一分钟没有自行恢复。SIGKILL/冷启动后才生成archive61，epoch1→2，动作数仍60。

原因：COMPLETED修复只推进domain阶段；它没有完成未知终局候选的恢复/归档，却允许清理唯一持久化修复入口。`snapshot.Phase.Terminal()` 的早返回也不能证明archive存在、版本/摘要一致。

**修复要求：** 在现有owner公开合同内，先以稳定命令ID、已提交版本和epoch解决未知结果/恢复候选，再幂等完成归档、domain和房间状态；拒绝从quarantined旧状态生成终局。只有这些持久化状态真正一致后才能条件清除修复入口。不要直接从别的模块读取私表、覆盖冲突归档、延长回合或依赖重启掩盖。增加真实MySQL COMMIT响应损失的两种终局回归，同时保留重复/迟到/并发与启动中未知提交检查。

证据：[NO_LANDLORD语义与冷启动](0063-round2-evidence/no-landlord-faults.json)、[COMPLETED故障/崩溃/接管](0063-round2-evidence/completed-faults.json)、[最终数据库摘要](0063-round2-evidence/final-sql-summary.json)。最早夹具只检查“归档存在”曾输出PASS，该标签已撤回；原始文件私存，不能据其标签覆盖本报告FAIL。

### R2-N02 / P2：N06按钮文字没有随竖屏缩放策略放大

所有者：Cocos `GameTableView.button`、`TableGeometry.tableUnit`。按钮框使用放大的逻辑单位，Caption仍由 `Math.min(18,w*.19,h*.48)` 封顶18逻辑px。390/1280的实际缩放后只有5.48 CSS px；返回、同步和核验结算均复现。手牌、座位和中央结算文字已较大，不能据此忽略操作文字。

实测核验按钮：桌面367.2×51.3 CSS px/字体18；窄横屏218.03×30.46/字体10.69；竖屏210.6×38/字体5.48。三个视口真实触摸均触发服务端证据请求，但这不是文字可读性通过。修复应覆盖叫地主、出牌/不出、导航与核验文字，并重新留真实尺寸截图及命中证据，不只增加几何断言。

### R2-N03 / P2：正常取消牌局后，活跃截止游标永久进入重投循环

所有者同终局协调/调度。真实三账号发牌到BIDDING，参与者通过既有 `hand.cancel` 成功终止；domain ABORTED、archive2、live action0。原BIDDING/v1截止行仍在，过原截止时间后持续更新 `next_attempt_at`、投递新deliveryID；多次观察归档摘要不变，但游标没有退休。

原因：worker读到domain terminal直接返回nil，原行仍是BIDDING，既不转为受保护的终局修复阶段，也不能经当前限定NO_LANDLORD/COMPLETED的清理分支退出。结束牌局持续消耗SQL、双队列和Redis去重资源；本轮只实测单局，不夸称完成长期DoS压测。

应与R2-N01统一设计“真正收敛后退休”的生命周期，覆盖正常完成、全不叫和参与者取消；这是同一修复领域的不同验收条件，不应分别临时加无条件DELETE。[取消游标现场](0063-round2-evidence/cancel-cursor.json) 保留，未手工清除来使测试变绿。

## 完整业务、故障与资源结果

同一真实手牌 `g63-full-4f0d7bf9ebd7f44f0adc05b0` 完成三身份认证、入房/准备、真实贡献与发牌、原生叫地主/底牌展示、排序手牌、实际原生出牌/不出、队列自动动作、重连、COMPLETED/v61、三账号一致的终局证据和原生核验。胜者3，积分[-3,-3,6]。归档后至少32秒，旧任务没有更改版本/摘要，截止行已消失；合法非成员private/evidence403，观察者未收到手牌推送。

这是**真实三协议客户端的完整业务闭环，加实际Cocos关键交互与终局闭环**；剩余动作由协议驱动，不能宣称三个Cocos客户端全手操作或手机真机全部完成。18/19张在对局中经过，未逐个手牌数量、逐牌型、逐视口穷举截图。长触摸序列在双页面软件GPU环境跨过真实30s期限，自动出牌使旧目标节点消失；没有延长期限，后续在同一局用单渲染页完成代表触摸与业务操作。

| 独立检查 | 结果 |
|---|---|
| 真MySQL差异/事务套件 | 7项通过：截止门禁、锁后时间检查、人工/自动胜出、12并发仅1提交、真实COMMIT OK损失与断链恢复；保留SQL状态/退出码 |
| 真队列+Race | 10项通过（含6项真实队列场景）；真实双副本Put ACK损失后精确重试只回调一次；三节点静默第三副本不拖慢两成功仲裁；64并发超时goroutine回基线 |
| 同批失败隔离 | 两真实副本已写入但ACK被丢；该牌局发布报未知，健康后续牌局约2.110s复制完成；没有无限阻塞，但串行失败预算有实际等待成本 |
| 1000条未来调度 | 2轮、11.116s，goroutine8→8，GC后heap693056B；真实MySQL/双队列，合成未来回合，无业务callback |
| 128条集中到期 | 首次复制从调度开始计P50/P95/P99=888.702/1441.025/1463.695ms，max1520.205ms；总1.523s、128条均复制，goroutine8→8，GC后heap695136B；合成到期回合，无真实游戏callback |
| 正常自动动作 | 原截止→SQL提交：不叫1231ms、最小单张1813ms、不出1686ms；原始毫秒时间保留，不是P95/P99容量或零延迟承诺 |
| 进程关闭 | 真节点暂停且出现新的已建立发布TCP连接时，API正常退出0约1044ms，应用队列连接清零 |
| 原生生命周期 | 12轮终局BackToRoom/OpenTable，节点恒25、250ms计时器0/1；GC后JS heap27660500→27874476B。短程测量，非长期无泄漏证明 |
| 相关stock回归 | bootstrap与lifecycle两包独立重跑通过；完整其他Go/身份/集成/客户端/Race复用同源CI。客户端CI为33core+50protocol+135Cocos检查，不把静态几何绿灯当实际竖屏通过 |

压测时API已停止，使用单独perf schema，避免后台扫描竞争；合成行最后计数0。上述排程数字不等于1000局/128局真实游戏吞吐、所有回调按时提交或生产硬件SLO。原子日志、幂等、旧版本约束与权限边界有实际测试；没有凭没有Race报告就断言逻辑正确，N04本身就是反例。

崩溃复合场景中真实domain行锁造成两个等待，终局live/归档已提交、domain仍PLAYING且持久标记存在时SIGKILL。接管后刻意丢COMPLETED domain的实际COMMIT OK；首次启动退出1，第二次启动READY，epoch1→3，59条动作、archive60摘要不变、标记清除。这证明所执行范围的持久化/再次启动恢复，不是无需额外重试或部署自动重启策略已通过。

## 截图

| 真实手牌/状态 | 1280×720 | 760×390 | 390×844 |
|---|---|---|---|
| 17张 | [桌面](0063-round2-evidence/table-17-1280x720.png) | [窄横屏](0063-round2-evidence/table-17-760x390.png) | [竖屏](0063-round2-evidence/table-17-390x844.png) |
| 20张 | [桌面](0063-round2-evidence/table-20-1280x720.png) | [窄横屏](0063-round2-evidence/table-20-760x390.png) | [竖屏](0063-round2-evidence/table-20-390x844.png) |
| 真实COMPLETED | [桌面](0063-round2-evidence/terminal-completed-1280x720.png) | [窄横屏](0063-round2-evidence/terminal-completed-760x390.png) | [竖屏](0063-round2-evidence/terminal-completed-390x844.png) |

## 可复现命令、夹具边界与自检

原始证据和外部夹具位于私有目录 `/root/goal0063-round2-099af4e`，`business/` 为后续真实业务；候选源码保持干净。Go命令从候选 `server/` 执行：

```text
go test -p 1 -parallel 1 -count=1 -tags=integration -overlay=E/sql-overlay.json -run 'TestG63|TestDQTurnDeadlineEnforces' -v ./business/doudizhu/infrastructure/mysqlstore
go test -race -p 1 -parallel 1 -count=1 -overlay=E/queue-overlay.json -v ./platform/delayqueue
go test -p 1 -parallel 1 -count=1 -overlay=E/backlog-overlay.json -run 'TestG63ActualThousand|TestR2RealBadHand|TestR2RealConcentrated' -v ./apps/app-api/internal/bootstrap
go test -p 1 -parallel 1 -count=1 -v ./apps/app-api/internal/bootstrap ./business/doudizhu/application/lifecycle
```

E下 `run.py/run-perf.py` 负责加载私有测试环境、低并发限制、实际退出码及耗时。真实故障脚本为 `terminal-round2.cjs`、`business/completed-faults.cjs`、`business/cancel-cursor.cjs`；代理按真实MySQL协议丢COMMIT OK，复合场景按实际COMPLETED参数定向丢domain ACK。源SHA、动作数、domain、archive、游标和epoch相互核对，没有修改生产代码或屏蔽产品失败。

夹具失败没有抹去：[记录](0063-round2-evidence/fixture-notes.md) 保留缺注册开关403、异步房间恢复等待不足、长触摸跨期限、把WSS刷新误当HTTP、重载后探针计时器变量丢失、终局正常private404的错误断言、临时TableNotice清理导致节点26→25、代理重启后的连接错误与SQL检查表达式错误。原始脚本标签不充当最终结论。

最后补充的终局“同步”及新页面恢复，初次因本地代理/API启动顺序失败，随后多房间发现经故障代理额外时延触及未改变的3s请求期限。原失败保留；补充名义UI检查改为同一测试MySQL/Redis直连，候选、数据、保护根、HTTP预算和20/30s回合设置不变，实际同步HTTP200、新页面v61及非成员403通过。不能据此宣称故障代理多房间场景或生产发现查询性能通过。

第二遍自检由同一验收执行者完成，**不是又一位独立审阅者**：重新追踪正常业务和两种未知终局、检查quarantine/归档/标记顺序、事务与epoch、重复命令/旧任务、权限、Base64真实边界、时钟/视图销毁、测试假阳性和资源范围。对最初错误PASS/FAIL标签逐一以真实数据库及运行证据校正；没有为使验收通过而修产品。

## 环境保护与交接

Kubernetes Deployment/Service/ConfigMap/PVC名称、UID及spec/data，Secret名称/UID/resourceVersion、既有Pod身份/状态/重启数均与只读基线一致；未读Secret值、未对既有业务库/PVC做字节级比较，不夸称数据内容全部hash相同。[保护审计](0063-round2-evidence/protection-audit.json)。本轮没有操作用户转发；收尾本机查询该转发unit为not-found、29457无监听，初始状态未单独留证，因此不能宣称其在本机始终运行，也未擅自创建或启动。没有执行部署、迁移、回滚或合并。

本轮专用API、浏览器、Web、代理及MySQL/Redis/三队列在收尾停止；失败schema、容器存储、密钥和原始证据保留以供修复复现，不drop/flush。发布资料仅含筛选状态、测试日志、哈希和QA截图；账号密码、JWT、私钥、配置、浏览器profile、原始socket和完整终局秘密材料不提交。

未验证/未通过的边界明确保留：真实手机及微信SDK/发布、beta引擎发布资格、长期负载/泄漏、完整帧时间/低端机性能、生产SLO、OTLP动态链路、实际Kubernetes滚动部署/数据升级/回滚和存储级灾难恢复。本轮没有把这些计入PASS。

**交给ChatGPT的两个修复主题：** ①统一终局恢复及真正收敛后的游标退休（R2-N01/P1、R2-N03/P2）；②修正实际竖屏操作文字（R2-N02/P2）。修复后推送新的固定SHA，重新独立验证上述真实故障、取消路径、三视口与完整业务回归。原Round1 FAIL及Goal0045/0046未完成状态保留；本次没有批准合并或部署。
