# Goal 0046 — Round 17 增量独立验收

日期：2026-11-07。执行者：Codex。仅独立验收及报告，不修改产品、正式测试或数据库Schema。

## 1. 最终判断

| 判断 | 结论 | 依据 |
|---|---|---|
| A — R16-N01 原始真实反例 | **PASS** | 真实WSS ACK扣留、HTTP先读H2时实例不替换；prepare/commit被阻止、零公平请求，释放ACK后正常采用H2 |
| B — 开局、刷新和公平会话生命周期 | **PASS（本轮检查点）** | 成功/明确拒绝/真实超时/未发送/重连/销毁/注销及交错刷新通过；未发现新产品缺陷 |
| C — 原R15-N01/P2回归保护 | **PASS** | 原commit精确对账先于开局，各层H2一致，正常公平及Bid/Play/Pass可继续 |
| D — Goal0046内部完成资格 | **BLOCKED** | E03人工备份/恢复UX、H05内部平台适配仍未完成，不将检查点PASS等同整个Goal完成 |
| E — 外部平台、真机与正式交付资格 | **BLOCKED** | 正式工具、域名/TLS、手机/平台及发布条件证据不足 |

R16-N01/P3关闭；R14-N01、R14-N02、R15-N01/P2的历史PASS保留。Goal0045、0046继续 `in_progress`，不封版或归档。

## 2. 固定版本与工作流

产品候选：`5412c2a981d067e840be4e76369ecd1d0d92405b`。初始main及工作区干净，经 `git pull --ff-only` 从Round16报告提交 `0dded647098b6f140b024a8f5397aae1e89cc167` 同步到固定候选；报告前再次同步，仍是同一SHA，未混入其他产品提交。

已独立读取各run的head SHA与job结论：

| 工作流 | 候选与结果 |
|---|---|
| [CI 42329547590](https://github.com/oublie6/awesome-zero-platform/actions/runs/42329547590) | 精确5412c2a，SUCCESS，unit/secure_envelope/admin_web/integration/runtime/report **6/6** |
| [Fair Doudizhu 42329547560](https://github.com/oublie6/awesome-zero-platform/actions/runs/42329547560) | 精确5412c2a，SUCCESS，manifest_client/mysql_integration/client_foundation/application **4/4** |
| [Cocos Linux Native Client 42329547559](https://github.com/oublie6/awesome-zero-platform/actions/runs/42329547559) | 精确5412c2a，总体SUCCESS；**linux_source SUCCESS、linux_native_export SKIPPED** |

没有重复完整CI。source门槛成功不等于实际原生导出、SDK运行或真机资格；本轮没有重新导出或下载Cocos。

报告提交与产品候选分开；报告SHA由 `git log -1 --format=%H -- docs/goals/0046-round17-independent-acceptance.md` 查询，最终交接提供实际提交。

## 3. 差异与环境

按AGENTS、原则、current、Goal0046、Round16及必要Round15依据检查。实际产品差异只有：

- `GameOnlineController.ts`：公开handStartInProgress；开局未结算时，任何不同于旧confirmedFairnessHand的绑定都拒绝，包括已被HTTP读到的新H2；在真实durable ACK后同步清startingHand并立即绑定H2，finally仍覆盖失败释放。
- `GameBootstrap.ts`：读取、校验房间后，若handStartInProgress则返回，早于公平会话替换和RoomCoordinator.restore。
- 两份客户端测试：增加早到H2刷新、锁释放及源契约检查。没有更改公平协议、GameFairnessActions、GameHandActions、GameClientShell、RoomCoordinator或后端。

审查了旧公平请求对账→预留动作→发送room命令→ACK校验→采用H2→finally释放的顺序，以及RoomCoordinator pending/序列/幂等、shell消息身份隔离、Bootstrap generation围栏。同步清guard到绑定之间没有await；失败路径保留未决请求而不是清pending求PASS。

私有证据 **`E=/root/.config/awesome-zero-platform/acceptance-0046-r17-5412c2a`**。从Round16复制脚本/测试配置到新目录，保留原始失败目录及报告不变；`runs.jsonl`保存argv、cwd、固定SHA、UTC、耗时、退出码，`ci.json`保存工作流核对，`sha256.json`和`completion-audit.json`索引证据。秘密、账号及恢复副本不提交。

复用Linux、Node22.23.4、MySQL5.7.44/Redis8.8.0、既有自有隔离产品库和HTTPS/WSS服务。等待MySQL健康后启动API；仅在私有副本延长测试reveal registry有效期，不修改保护根。server tree仍与Round13一致：`bafabbb8cb75baec79bf94b8b968b297391e83e2`，复用编译源 `14748096aa34e9d423ffcd48141988646b4bdd41` 的API二进制，**不声称重新编译了5412c2a后端**。

当前client dist由本轮test构建；Bootstrap由当前TS源码转译，Cocos基础节点在Node探针替代，实际Online/Fairness/HandActions、RoomCoordinator、Go认证/协议/HTTP/WSS/数据库不替代。故这是当前源客户端控制流与真实服务证据，不是原生触摸/手机UX。

## 4. R16-N01 原反例：时序和负向证据

复用 `early-refresh.mjs`，保持真实WSS消息进入GameMessageRouter之前的ACK扣留机制与原最终负向断言。旧脚本的“实例已替换”缺陷观察断言改为要求保留原实例，**没有删除或弱化“不得插入新公平操作”的负向断言**；新增commit、零请求、秘密保留、协调器及成功后业务检查。原历史FAIL不改。

真实三账号H1准备恢复副本并验证，另一合法成员取消H1，房间重新准备；owner发起H2，服务端真实提交成功。本次记录：

| ACK尚未处理时 | 实测 |
|---|---|
| H1 | `e03-r14-4d878063299d0b38` |
| HTTP权威H2 | `hand-cmd-a42f6df87b4b9faf66a0fb2c4644227c` |
| 开局requestId | `cmd-494e06adcfb2ff380f3c00abcc8dc63a` |
| 房间版本 | RoomCoordinator仍11，server已12；原pending内容与ID不变 |
| Bootstrap/controller | 均保留H1；handStartInProgress=true |
| 公平实例/秘密 | 与开局前同一实例，H1恢复副本一致，secret保留 |
| 多次HTTP读H2及旧H1响应交错 | 当前Bootstrap.refreshRoom提前返回，不替换会话、不恢复飞行中协调器 |
| prepareRecoveryKit(H2)、commit(H2)、reveal(H2) | 均拒绝in progress；**公平projection=0、command=0，无H2新秘密** |
| 直接confirmH2、reconcile、重复开局、Bid/Play/Pass | 不能绕过锁；协调器原pending不变，只有原开局命令 |

这里扣留的是真实成功ACK，没有伪造响应或直接修改server权威数据。释放同一WSS ACK后：Bootstrap、controller、shell、server均采用H2，RoomCoordinator版本12；旧H1secret清除，互斥解除，无“开局失败”。另一条旧H1 HTTP响应在ACK后返回，被generation隔离；迟到H1事件不能重新绑定。H2再生成新凭证，三人重新备份验证、commit及真实Manifest/AAD/HPKE reveal均成功，之后多次正常刷新保持同一合法实例。

结果：`real-early-refresh-WSS-boundary.log` exit0、`early-refresh.json`的earlyPrepareAndCommitBlocked/H2CommitRevealAfterACK/finalHandIDMatches/originalNegativeAssertionPassed均true。

## 5. 关联失败与资源生命周期

| 场景 | 实际验证 |
|---|---|
| 明确拒绝开局 | 实际有一席未ready；server拒绝后startingHand及两类reservation解除，旧kit不变，无未知pending被静默丢弃；该席合法准备后再次开局成功 |
| H2未发送成功 | 在socket写入前注入断线错误；原命令ID/版本/内容保留，提示“尚未确认”，再次点击不发新命令；真实重连和权威HTTP确认H1仍活跃，保留同一H1公平实例和kit |
| 实际15秒超时 | 成功H2的真实ACK一直扣留到shell原生15000ms timer触发；未知请求完整保留、旧secret保留、guard解除；权威刷新采用实际H2后才清旧secret，迟到原ACK不回退 |
| 丢ACK后的精确重试 | 真实room-exact：完整原始room.hand.start请求深比较一致，返回已提交结果，最终各层H2一致；不是新ID重开 |
| 开局飞行期间离房 | RoomCoordinator拒绝outstanding，原pending和旧secret不变，没有额外离房成功 |
| 销毁/注销时开局ACK未交付 | 实际Bootstrap.onDestroy或controller.logout关闭原shell，开局变UNKNOWN，startingHand释放，原room请求仍保留；迟到ACK不能被已销毁路由采用；新认证controller按真实H2恢复且不继承旧秘密 |
| 新旧异步请求交错 | 旧H1刷新分别在ACK前/后返回；新H2刷新在ACK前多次到达及确认后重复到达，均没有回绑、重复公平实例或永久锁 |

销毁/注销测试验证**本地controller/shell生命周期**；测试identity的revoke为既有noop，不把它声称为本轮服务器token撤销验收。此时H1已合法终止，本地退出清理旧secret属既有语义；不是清除尚未确认的公平commit/reveal。原未决room命令身份保留。

初次明确拒绝场景已通过秘密/锁检查，但后续夹具对已ready席位又发ready(true)，被server正确拒绝DDZ_INVALID_COMMAND。修正为仅令未ready席位ready，再核对权威版本；独立重跑全部该场景exit0。原日志保留，非产品缺陷，未改产品或降低断言。

## 6. 正常链路与原P2回归

复用 `real-scenarios.mjs live` 当前候选实跑exit0：

`H1 commit接受但响应丢失 → 他人取消H1 → 重新准备 → 原commit精确重试并确认 → H2开局 → 外部备份重入验证 → 三人commit/reveal → 真实公开beacon发牌 → Bid(3) → PLAYING → Play → Pass`。

记录commit完整请求深比较，先commit-confirmed后room-start，各层H2一致。H1kit不能用于H2；新kit独立；同H2正常WSS重连及HTTP刷新保留合法实例和secret。真实Bid飞行中不能启动新局；真实Play/Pass已接受但响应丢失，原请求保留、开局发送0次，之后原指令精确重试成功。

另复用 `real-reveal-transition.mjs` exit0：真实reveal接受后扣留响应，飞行中start被阻止；取消后原密文/请求精确重试再超时仍不发start；第三次精确请求结算后才开新局。正常终止/退出成功后安全清理。没有观察到新ID盲重试、秘密丢失、永久锁或已ACK仍停H1。

## 7. 执行结果、历史适用性与矩阵

| 本轮实际执行 | 最终结果 |
|---|---|
| `npm --prefix clients/fair-doudizhu-cocos test` | **59 PASS / 0 FAIL**，含当前dist构建 |
| `npm --prefix clients/fair-doudizhu-cocos run typecheck` | exit0 |
| 复用的独立 `races.mjs` | **12/12 PASS**，含公平精确对账接受/拒绝/再超时及Bid/Play/Pass飞行/未知 |
| 原真实early-refresh及正常live链路 | 各exit0 |
| rejected/unsubmitted/timeout/destroy/logout | 修正ready夹具后5场景均exit0 |
| 真实reveal-transition、room-exact | 各exit0 |

九条真实集成场景分别记录，不把其中断言重复计作另一套覆盖率。未重跑Round16未受影响的UI/恢复全套、100/1000局、完整SIGKILL、其他平台或原生导出；历史证据只在相应未变实现范围内沿用，不冒称本轮执行。

[Round16](0046-round16-independent-acceptance.md)与[Round15](0046-round15-independent-acceptance.md)的备份/新进程恢复、终局、历史receipt、公开/私牌授权及后端持久化依据在本轮未变代码范围仍适用；R14-N01/N02无新回归证据，保留PASS。稳定管理员根、HMAC、AES-GCM、错误根失败关闭、原根恢复与禁用rotate没有变化；C09/C10继续正式DEFERRED，不改历史报告。

| 记账对象 | Round16 | Round17 |
|---|---|---|
| 原114项 | 109PASS / 0FAIL / 3BLOCKED / 2DEFERRED | **不变**：109PASS / 0FAIL / 3BLOCKED(E03/H05/GOAL8) / 2DEFERRED(C09/C10) |
| 单列的开局全窗口互斥门槛 | FAIL，R16-N01/P3 | **PASS，本轮真实原反例关闭** |
| 原R15-N01/P2 | PASS | PASS，新候选实际回归 |
| 新产品缺陷 | 1项P3 | **未发现** |

不是重新验收114项；不把原矩阵0 FAIL等同全部Goal已完成，也不把DEFERRED计作PASS。

## 8. 尚未完成与最小后续建议

- **E03 BLOCKED**：本轮真实受控备份/验证/揭示并不证明用户长期安全保管；真实系统键盘、长凭证复制粘贴、备份找回及手机UX证据仍缺。原生export仍SKIPPED。
- **H05 BLOCKED**：实际抖音入口、身份交换及合格导出/SDK适配仍有内部缺口，不能全归为外部配置。
- **GOAL8/外部资格 BLOCKED**：正式HTTPS/WSS域名/TLS、支持工具、真实手机、多用户前后台/网络/分享及发布证据仍不完整。微信登录与AppID/AppSecret已有历史成功证据，不重标为“没登录/缺凭据”。

本轮没有新缺陷修复清单。后续仅推进上述剩余范围：E03实际用户输入/备份恢复资格、H05内部适配、GOAL8外部/设备发布条件，保留已通过的未知结果保护；不实施已延期C09/C10，不重复无关规模与故障矩阵。

Goal0045/0046保持 `in_progress`。只提交本报告和current状态；报告前后检查源码身份与工作区，私有测试API/TLS/自有DB服务停止，证据保留，其他部署不改。
