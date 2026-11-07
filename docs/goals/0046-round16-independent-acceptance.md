# Goal 0046 — Round 16 增量独立验收

日期：2026-11-07。Codex 独立验收，仅新增本报告与 current 状态；未修改产品、正式测试或 Schema。

## 1. 五项结论

| 判断 | 结论 | 边界 |
|---|---|---|
| A — R15-N01 修复 | **PASS** | 原真实反例重跑成功；原请求对账先于开局，ACK后各层采用H2 |
| B — 未知请求及生命周期安全回归 | **FAIL** | 未知请求保护、实时动作隔离和ACK采用通过；但新增必要互斥门槛有R16-N01/P3反例 |
| C — E03 当前验收范围 | **BLOCKED** | 真实备份/重启恢复/加密揭示通过；人工长凭证输入、复制粘贴及备份UX仍未资格化 |
| D — Goal0046 内部完成资格 | **FAIL** | 本轮明确要求的“开局期间不能插入新的公平操作”未满足；另有H05内部适配缺口 |
| E — 外部平台与正式交付资格 | **BLOCKED** | 官方支持环境、正式设备、域名/TLS及发布条件证据仍不全 |

**原P2已关闭，不是原状态错乱仍然存在。** 本轮唯一新发现为P3互斥约束缺口，没有证明秘密丢失、错误新ID重试或最终H2采用失败。B/D的FAIL表示本轮必需检查未全通过，不把这个较轻问题夸大为新的P1/P2业务损害，也不把未执行的设备测试算PASS。Goal0045/0046保持 `in_progress`。

## 2. 产品与CI身份

验收产品：`ff82def9485769664dbddd185cee755f53121c2f`。初始工作区干净，main由 `bcbd235b8657c8ab60466f0b6d1c72a2de3f2cf3` 经 `git pull --ff-only` 同步到固定候选；报告前再次同步无新增提交。

- [主CI 42255849395](https://github.com/oublie6/awesome-zero-platform/actions/runs/42255849395)：精确ff82def，SUCCESS，admin_web/integration/unit/secure_envelope/runtime/report **6/6**。
- [Fair Doudizhu 42255849223](https://github.com/oublie6/awesome-zero-platform/actions/runs/42255849223)：精确ff82def，SUCCESS，application/mysql_integration/client_foundation/manifest_client **4/4**。
- [Cocos Linux Native Client 42255627035](https://github.com/oublie6/awesome-zero-platform/actions/runs/42255627035)：`0c938bc40e7f171a1bc773bdd5978bb3cb313361`，总体SUCCESS；**linux_source SUCCESS，linux_native_export SKIPPED**。其clients tree与候选相同：`733fe7d4e4c8c107fc0d5902eec388ace8ade4b9`。只支持相同客户端源码门槛，不是当前包导出、SDK运行或手机UX证据。

已独立读取run SHA、结论和各job，不重跑已有成功CI。报告文档提交不冒充产品版本；报告提交SHA可由 `git log -1 --format=%H -- docs/goals/0046-round16-independent-acceptance.md` 查询。

## 3. 差异、环境及证据边界

阅读AGENTS、适用原则、current、Goal0046及Round14–15，核对调用链和状态转换。相对Round15：Bootstrap改为调用controller.startHand并区别未知开局；controller先精确对账旧fairness，再预留live与fairness生命周期，发出room.hand.start，随后同步采用hand；GameHandActions与GameFairnessActions增加实例级handTransition。GameClientShell和RoomCoordinator未变，单独审查其pending、序列、ACK、restore和WSS身份围栏。新增正式测试也独立重跑。

Linux、Node22.23.4；复用现有MySQL5.7.44/Redis8.8.0、自有隔离产品库、真实Go认证账号及HTTPS/WSS，不涉及生产部署。server tree与Round13相同：`bafabbb8cb75baec79bf94b8b968b297391e83e2`；复用编译源 `14748096aa34e9d423ffcd48141988646b4bdd41` 的API二进制，**不声称重新编译了ff82def后端**。

当前客户端dist由本轮test构建；Bootstrap按当前TS源码转译，Node中的Cocos基础节点为替代。Online/Fairness/HandActions、RoomCoordinator、HTTP/WSS与真实数据库业务不替代。这里的实际客户端控制流/协议证据不等于当前原生包触摸或真机证据。

私有证据 `E=/root/.config/awesome-zero-platform/acceptance-0046-r16-ff82def`：复用Round15脚本到新目录，历史FAIL及原文件不改；`runs.jsonl`记录argv/cwd/SHA/UTC/耗时/退出码，`ci.json`/`native-ci.json`记录核对结果，`sha256.json`索引。配置、账号、原请求密文和恢复副本不提交。测试按低资源顺序运行，不下载引擎、不跑无关规模/SIGKILL矩阵。

环境/夹具记录：复用证书及测试reveal registry已过期，先更新隔离测试有效期/自签证书；MySQL刚启动未就绪时API初始化失败，待依赖可用后重启。初次cancel断言误读durable（实际为termination，改查权威ABORTED）；WSS负向夹具初次缺v；旧Object.create夹具未初始化新字段startingHand=null。修正这些夹具后重跑，初始失败日志保留；没有修改产品、删除安全断言或将这些错误列为产品缺陷。

## 4. 原R15-N01与正常业务

`real-scenarios.mjs original`（修正cancel响应夹具后）exit0，真实三账号：

1. 启动H1，owner生成外部副本并重新导入验证；真实server接受commit后仅丢弃客户端响应。
2. 另一合法玩家取消H1，三人按真实版本重新准备；owner仍持有原pending和secret。
3. 当前Bootstrap.startHand自动精确重试H1请求。原requestId、序列、版本、payload全量相同；记录顺序为 **commit原发送→commit精确重试→确认→room.hand.start**。
4. H2成功后，Bootstrap.handID、controller.confirmedFairnessHand、shell.authoritativeHand和server.activeHandId相同；RoomCoordinator版本等于权威房间版本。旧pending已结算、旧secret清除，无“开局失败”。
5. H2生成新凭证，H1kit导入H2被拒绝；H2外部副本重新导入成功，三人commit/reveal，经真实Manifest/AAD/HPKE和公开beacon进入BIDDING，再实际bid(3)进入PLAYING。

`real-live-isolation`在另一H2同链路追加真实实时动作；并在H2备份后实际重连WSS及刷新，合法实例/秘密保留。最后合法取消测试手局、成员退出，确认秘密清除。

此外复用脚本的 `commit` / `restore` 在两个不同Node进程exit0：三人外部备份后commit；旧进程dispose并退出；新进程恢复同局、导入外部副本并完成三人加密揭示。不是保留旧实例内存恢复。Round14两个真实反例也重新通过：迟到prepare在合法退出后输入长度0、旧secret=false；取消后同controller新房/H2可生成、验证、commit。

## 5. 未知结果、竞态和失败检查

| 场景 | 实测结果与证据 |
|---|---|
| commit执行中、结果未知、精确重试再次超时 | 实例逻辑与真实unknown-again通过；后者新room命令0次，原secret/pending保留，恢复后仍能精确重试 |
| 旧公平请求明确持久化拒绝 | 真实prior-rejected：另一玩家先commit导致版本冲突，丢弃原拒绝响应；原请求精确重试得到同一拒绝并结算，之后H2成功采用 |
| reveal执行中/响应丢失/精确重试再次超时 | 真实reveal-transition：服务端已revealed但响应扣留；飞行中start不发出；取消后重试再丢响应仍不发出；第三次原请求与原密文相同，结算后才开H2 |
| Bid飞行中 | 真实bid已接受但调用未返回，同controller.startHand被阻断，新增room.hand.start=0；释放后正常完成 |
| Play、Pass已接受但响应丢失 | 真实各一次；pending保留，start发出0次，完整请求深比较精确重试成功，无新ID或重复业务推进 |
| 开局期间重复点击、旧实例公平及实时动作插入 | 真实ack-race与独立race检查通过：旧fairness prepare/commit/reveal和Bid/Play/Pass均拒绝，开局仅1次 |
| 旧HTTP刷新、旧WSS消息 | 真实旧房间响应在ACK后释放被generation隔离；旧H1 WSS changed及独立snapshot检查不能回绑H1。**这不覆盖下一节权威H2新刷新缺口** |
| 开局自身响应丢失 | 真实room-loss：提示“尚未确认”，RoomCoordinator保留原命令，再点击不发送新命令；实际重连查询权威H2并采用 |
| 开局自身原命令精确重试 | 真实room-exact：服务端成功后丢ACK，完整原room请求精确重试；权威刷新后各层H2一致 |
| 同手刷新、拒绝/未知离房、旧lastHandId、迟到prepare/import、销毁 | 复用独立UI12项及生命周期10项通过；未知结果不能被归档/本地推断清除 |

正常失败解除reservation且保留秘密由正式测试及独立对账失败测试支持。所有未知结果与明确拒绝分开记账，未观察到原requestId/内容/secret丢失。上述“旧实例动作被阻断”不代表所有可替换实例都受同一锁约束。

## 6. R16-N01 / P3：权威H2刷新可替换预留实例，提前放开公平操作

**影响级别：互斥约束缺口，未证实业务或安全损害。** 原R15-N01的最终状态错乱未复发；本反例最终hand仍匹配H2，新秘密仍保留。不能据此宣称丢密钥、跨局污染、越权或新的P2。

对应本轮任务§5.3明确要求：**“开局期间新的公平操作不能插入。”**

源码位置：`GameOnlineController.ts:169`只锁住previous实例；`:207–212`允许handId等于startingHand的刷新；`:218`创建不带reservation的新GameFairnessActions。实际UI调用在 `GameBootstrap.ts:344–345`，先替换fairness，再调用拒绝飞行中restore的RoomCoordinator。

最小真实复现（`early-refresh.mjs`）：

1. 三账号H1生成/验证凭证，合法取消H1并重新准备。
2. 调用真实Bootstrap.startHand；server真实提交H2。测试在 **WSS消息进入GameMessageRouter之前**扣留实际room.hand.start成功ACK，客户端room命令仍pending；不伪造响应。
3. 此时合法HTTP房间刷新已读到权威H2；Bootstrap调用confirmActiveFairnessHand(H2)，换掉原被锁公平实例，然后RoomCoordinator.restore因room命令仍飞行而拒绝。
4. 对当前controller.fairness调用prepareRecoveryKit(H2)，真实发出1次fairness projection请求并成功生成凭证；此时start尚未完成。实例级锁没有覆盖替换后的实例。
5. 释放真实WSS ACK，开局正常完成，最终各层H2一致且新secret保留；正常取消/退出清理。负向断言“start返回前不能插入公平操作”失败。

证据：`real-early-refresh-WSS-boundary.log` exit1；`early-refresh.json`：fairnessReplacedWhileStartAwaiting=true、newFairnessOperationInserted=true、projectionRequestsWhileStartAwaiting=1、finalHandIDMatches=true、newSecretRetainedAfterACK=true。更早submitCommand返回边界版本也复现同样结果；最终结论采用更强的实际WSS入口延迟版本。

**不声称现有按钮已造成新局误提交**：插入动作通过当前公平对象的公共方法触发，UI刷新替换则为实际Bootstrap控制流。证明的是本轮明文规定的动作隔离不变量未满足，而非已经发生实际损害。B不能按全项PASS记账；原P2的A仍PASS。

最小建议：把transition所有权放到controller，覆盖整个开局和fairness实例替换；或在命令结算前推迟权威新hand刷新对公平实例的替换/对新实例继承reservation。保持H2权威采用和同局秘密，不回退为无条件清理pending，不削弱精确对账。加入“真实ACK未投递、HTTP先看到H2、新实例尝试prepare/commit”的回归。交回ChatGPT，不由验收方修实现。

## 7. 自动化结果及矩阵

| 实际执行 | 最终有效结果 |
|---|---|
| `npm --prefix clients/fair-doudizhu-cocos test` | **56 PASS / 0 FAIL**，含受影响dist构建 |
| `npm --prefix clients/fair-doudizhu-cocos run typecheck` | exit0 |
| 独立旧UI / 生命周期 / 本轮竞态 | **12/12 + 10/10 + 12/12 PASS**；各组单独记账，不把真实链路重复计作另一套覆盖率 |
| 原反例与H2正常链路、真实实时隔离 | original、live均exit0 |
| 拒绝/再超时/ACK竞态/未知开局/精确开局重试 | prior-rejected、unknown-again、ack-race、room-loss、room-exact均exit0 |
| 真实reveal保护、外部备份新进程恢复、原R14两反例 | 均exit0 |
| 新H2刷新替换锁对象 | **exit1，R16-N01/P3** |

### 原114项的增量记账

基线为[Round15](0046-round15-independent-acceptance.md)，未受影响证据沿用[Round10](0046-round10-independent-acceptance.md)至[Round14](0046-round14-independent-acceptance.md)，不宣称逐项重跑。

| 原矩阵条目 | Round15 | 本轮 | 依据 |
|---|---|---|---|
| A03、GOAL2、H08 | FAIL | PASS（已测缺陷关闭及历史证据） | 原ACK后停旧局反例关闭、真实当前源状态采用及恢复通过 |
| E03 | FAIL | BLOCKED | 原生命周期P2关闭，实际备份/恢复通过；人工UX/当前原生运行证据不足 |
| H05、GOAL8 | BLOCKED | BLOCKED | 内部适配和正式平台/设备资格未完成 |
| C09、C10 | DEFERRED | DEFERRED | 用户正式延期的可选根迁移/退役；不是PASS，不阻塞本期 |
| 其他106项 | PASS | PASS（历史适用证据） | 源差异影响核查，无本轮重跑声明 |

**原114项：109 PASS / 0 FAIL / 3 BLOCKED（E03/H05/GOAL8）/ 2 DEFERRED（C09/C10）。** 另将**本轮明文增加的§5.3互斥门槛单列FAIL（R16-N01）**；不把一个未造成已知原业务断言失败的P3强行映射成原矩阵四项FAIL，也不以原矩阵0 FAIL掩盖本轮必要检查失败。当前114项不是Goal所有扩展安全门槛的替代品。

稳定管理员根、Journal HMAC、AES-GCM、错误根失败关闭及禁用rotate未变，复用Round13有效依据；历史报告不覆盖。未重跑后端完整SIGKILL、100/1000局、其他平台构建；没有下载Cocos或重新运行原生导出。Round14原生/SDK证据只保留其原SHA层级，不把旧包当成ff82def新包。

## 8. 未完成项及交接

- **E03 BLOCKED**：受控外部文件回填、真实加密恢复不等于人类安全保管、系统键盘/长凭证复制粘贴和手机UX；当前源码仅有source gate，实际native export SKIPPED。
- **H05 BLOCKED**：实际抖音入口/身份交换、合格导出/SDK适配仍有内部缺口；不能全部归为外部凭据。
- **GOAL8/外部资格 BLOCKED**：正式HTTPS/WSS域名/TLS、支持工具、真实手机、多用户前后台/网络/分享与发布证据不足。微信登录/AppID/AppSecret已有历史成功证据，不重新标为“未登录/缺AppID”。

下一轮最小清单：①处理R16-N01跨实例transition reservation并复测该窗口，保留现有P2修复和全部未知保护；②补E03真实输入/备份恢复UX及当前原生运行证据；③分开推进H05内部适配与GOAL8外部资格。不开发已延期C09/C10、不重做无关规模/故障测试。

本轮报告交付完成不等于产品完成。Goal0045/0046继续 `in_progress`。只提交本报告与current允许状态；自有API/TLS/测试DB服务停止、私有证据保留，不修改其他部署。
