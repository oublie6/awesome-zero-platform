# Goal 0046 — Round 15 增量独立验收

日期：2026-11-06。执行者：Codex 独立验收；没有修改产品、正式测试或数据库 Schema。

## 1. 结论与版本

| 独立判断 | 结果 | 依据 |
|---|---|---|
| A — R14-N01 修复 | **PASS** | 原真实迟到响应反例通过；离房后输入为空，旧实例不再持有秘密 |
| B — R14-N02 修复 | **PASS** | 原取消→合法离房→同控制器新局反例通过；新贡献、重入校验及承诺成功 |
| C — E03 当前范围完成资格 | **FAIL** | 新 R15-N01/P2；人工长凭证备份/键盘/粘贴仍未资格化 |
| D — Goal0046 内部完成资格 | **FAIL** | 真实开局成功后本地状态未推进；H05 内部适配也未完成 |
| E — 外部平台与发布资格 | **BLOCKED** | 正式设备、域名/TLS、支持工具与发布条件证据不全 |

实际产品候选：`15e39e6b5d7c90c4783b1bcfd54b7a91cf20cc3a`。开始和提交前均确认干净 `main`、`git pull --ff-only` 无新增提交。报告提交与产品候选分开；可用 `git log -1 --format=%H -- docs/goals/0046-round15-independent-acceptance.md` 查询本报告提交。

精确候选的 [CI 42210813497](https://github.com/oublie6/awesome-zero-platform/actions/runs/42210813497) SUCCESS：secure_envelope、integration、unit、admin_web、runtime、report 共6/6；[Fair Doudizhu 42210813776](https://github.com/oublie6/awesome-zero-platform/actions/runs/42210813776) SUCCESS：application、mysql_integration、client_foundation、manifest_client 共4/4。已独立读取 run head SHA 和各 job 结论，不把绿色 CI 当作产品通过。

用户指定的 [Cocos Linux Native Client 42210482909](https://github.com/oublie6/awesome-zero-platform/actions/runs/42210482909) 对应 `55ce3cdaee1a28684072de8e6231641f896cfbcb`，总体 SUCCESS，但 **linux_source SUCCESS、linux_native_export SKIPPED**。其 clients tree 与本候选相同（`2c8298d4dd1d5e5ba624cb079d0ff3d07af7d2ac`），可以支持源码门槛，**不能证明当前候选已实际导出或运行**。

## 2. 范围、环境和证据

按本轮明确授权执行增量验收，不重新跑114项全量、100/1000局、完整 SIGKILL 矩阵或下载/重建引擎。复读 AGENTS、原则、current、Goal0046 及 Round13–14；保留历史 FAIL 原件。相对 Round14 报告提交 `04cec17e43022f272e75097492b7d30c7637bda9`，产品差异限于 GameBootstrap、GameOnlineController，两份正式客户端测试及 current 状态。风险集中于异步写入围栏、公平实例替换、未知结果与房间命令先后顺序。

Bootstrap.prepare 新增 online/room/hand/generation/screen/input/fairness/销毁状态检查；controller 在已确认离房或 active hand 改变时替换公平实例，保留 needsReconciliation 检查。GameFairnessActions 与后端未变。整个 server tree 与 Round13 相同：`bafabbb8cb75baec79bf94b8b968b297391e83e2`；本轮复用编译自 `14748096aa34e9d423ffcd48141988646b4bdd41` 的 API 二进制，**不称它是重新编译的15e39e6产物**。

Linux、Node22.23.4、现有 MySQL5.7.44/Redis8.8.0、独立 HTTPS/WSS 代理和自有隔离产品库；真实 Go 认证账号、房间和加密公平协议。重型命令顺序执行。当前源码 Bootstrap 方法经 TS 转译运行，Cocos 节点在 Node 探针中替代；真实 GameOnlineController/GameFairnessActions、HTTP/WSS 和数据库不替代。故这些是客户端控制流加真实服务集成证据，不能称为原生触摸/手机复现。

私有证据：`E=/root/.config/awesome-zero-platform/acceptance-0046-r15-15e39e6`。`runs.jsonl` 记录 argv、cwd、SHA、UTC、耗时和退出码；`ci.json`、`native-ci.json` 记录核对结果；脚本、响应摘要、受控外部备份及配置留在 E，不提交秘密。复用 Round14 脚本时仅适配 rooms 页面前提及真实 controller 新接口，保留负向断言和原始 FAIL。

## 3. 原缺陷回归和新边界

| 检查 | Round14 | Round15 实测 |
|---|---|---|
| R14-N01：H1 prepare 响应延迟至取消、其他成员退出及 owner 最后权威退出后到达 | 回填356字符旧kit、旧secret保留 | PASS：当前房间/手局为空，staleKitLength=0、oldSecretRetained=false |
| R14-N02：H1只生成未commit，合法取消/离房，同账号同controller建H2 | different-hand secret 阻塞 | PASS：旧实例释放，新实例/贡献与H1不同，H2生成、导入校验、commit均成功，无需重新登录 |
| 迟到 prepare：新hand、controller、页面、输入实例、销毁 | 扩展负向 | PASS：不能污染当前输入/状态；旧账号不继承新实例秘密 |
| 重复生成、失败/超时重试、备份门槛、非法kit/seat/hand/digest | 沿用独立反例 | PASS：未验证不能commit；失败不设置proof；旧异步import不设置当前proof |
| 同active hand刷新/WSS重连 | 必须保留 | PASS：实例和合法秘密保留；另有真实结果未知commit后的WSS重连及刷新 |
| 明确拒绝/结果未知的离房 | 必须保留 | PASS：不清秘密；原离房重试得到确认后才清理 |
| commit/reveal飞行中切换 | 必须阻断 | PASS：pending保护；不以新hand确认绕过 |
| 仅有lastHandId | 不应视为active | PASS：不用归档ID继续持有活跃公平实例 |

实际 `ui-probe.mjs` **12/12 PASS**；新增 `lifecycle-edges.mjs` **10/10 PASS**。两组为22项独立逻辑检查，不和真实链路重复合计为另一套覆盖率。

## 4. 真实恢复与未知结果

`real-recovery.mjs commit` / `restore` 在两个独立 Node 进程 exit0：三账号真实建房/加入/准备/开局，当前 Bootstrap 门槛生成凭证并保存到进程外文件，清空后重入验证才commit；全部旧controller dispose并退出；新进程从权威房间恢复同局并导入外部副本；三人真实 Manifest/AAD/HPKE reveal 成功，确认后清除秘密。不是保留旧内存完成恢复；文件副本也不等于真实用户长期安全备份。

真实 `unknown-commit`、`unknown-reveal`：正常服务端接受请求后主动丢弃客户端响应。pending、完整请求与相关secret保留，同hand刷新不替换实例；null/不同hand切换及离房被阻断；精确重试对比原 requestId 和全部请求内容（reveal含密文），成功后才解除未知状态。真实 `unknown-reconnect` 额外重建 WSS 并刷新，仍保持旧pending且精确重试成功。未引入新 requestId 盲重试，未观察到恢复材料丢失。

上述保护通过不代表随后房间操作的顺序正确：下一节为真实反例。createRoom 另被 RoomCoordinator 的“已管理房间”前置约束拒绝，未发生服务器建房，**不列为实际缺陷**；join等未经实测路径不按理论追加FAIL。

## 5. R15-N01 / P2：服务器已开新局，本地清理检查抛错后仍停旧局

位置：`GameBootstrap.ts:306` startHand，特别是先 `await online.rooms.startHand(handID)`、后 `confirmActiveFairnessHand(handID)`、再更新 handID/generation 的顺序；`GameOnlineController.ts:156` 的 confirm/reset 检查。

最小真实复现：

1. 三账号合法开H1，owner生成/验证凭证；服务端接受 owner commit，测试只丢弃响应，owner保留未知请求和secret。
2. 另一合法成员取消H1；三人按实际房间版本刷新并重新准备，owner旧pending保持未对账。
3. 调用当前源码 Bootstrap.startHand。真实 `room.hand.start` 成功，服务端和 RoomCoordinator 都推进到版本12、activeHandId=H2。
4. 随后的 `confirmActiveFairnessHand(H2)` 因H1未知结果抛错；Bootstrap捕获后显示 **“开局失败：fairness result unknown; retry the original request before changing hand”**，但自己的 handID 仍是H1，未完成本地新局采用。
5. 旧请求/secret仍在且原commit精确重试成功；故不是不可逆密钥丢失或P1越权。已成功的新局被错误表现为失败，正常流程和UI权威状态不一致，按P2阻断内部验收。

证据：`real-post-ack-start-fixture-corrected.log` exit1、`real-unknown-start.json`。摘要中 serverConfirmedHand != oldHand、uiHand == oldHand、confirmedRoomVersion == coordinatorRoomVersion == 12、oldPendingAndSecretRetained=true、exactOriginalRequestReplay=true。注入仅丢失响应，不伪造服务端ACK；测试结束已正常取消H2并清理成员。

首次该测试的 loader 缺少实际 `secureRoomCommandIdentity` 导入，导致请求未发送，是**夹具失败**，不据此判产品FAIL。修正 loader 为真实 room 模块后重跑，才得到上述实际开局成功后反例；没有删除负向断言，也没有修改产品。

最小修复建议交回ChatGPT：在发送变更房间/开局命令之前检查和对账旧公平操作；同时处理检查后状态变化的并发边界。对已经权威确认成功的操作，应正确采用/恢复新状态，不能统一报告为业务失败。不得无条件删除pending/secret、跳过对账或使用新ID重试旧请求。增加“旧commit未知→他人取消→重准备→start”的真实回归及已ACK后的状态采用检查。

## 6. 自动化结果与历史复用

| 本轮真实执行 | 结果 |
|---|---|
| `npm --prefix clients/fair-doudizhu-cocos test` | 46 PASS / 0 FAIL，exit0 |
| `npm --prefix clients/fair-doudizhu-cocos run typecheck` | exit0 |
| 独立 UI / 生命周期逻辑检查 | 12/12、10/10 PASS |
| 真实 commit / 新进程restore / 原N01 / 原N02 | 各exit0 |
| 真实unknown commit / reveal / WSS reconnect | 各exit0 |
| 真实post-ACK start，夹具修正后 | **exit1，R15-N01** |

未重跑不受本轮更改影响的完整后端单测/race/SIGKILL/规模矩阵；相关现有实现及有效依据继承 [Round10](0046-round10-independent-acceptance.md)、[Round11](0046-round11-independent-acceptance.md)、[Round12](0046-round12-independent-acceptance.md)、[Round13](0046-round13-independent-acceptance.md)，不是本轮新执行。稳定根、HMAC、AES-GCM、错误根失败关闭、原根恢复、禁用rotate均无实现变化。

[Round14](0046-round14-independent-acceptance.md) 的Linux原生导出与16项SDK操作仅保留其原SHA证据层级；本次没有新增SDK调用，但改动了控制流，**没有将旧包等同于当前包**。本轮没有重新原生导出/运行当前15e39e6，也未做人工键盘、长凭证剪贴板或真实手机UX；当前源码/真实服务探针不能替代这些检查。原生CI实际跳过导出的限制见§1。E03不能因两个原问题关闭而自动PASS。

## 7. 114项矩阵的增量记账与资格

以Round14完整记账为基线，未受影响项沿用原始逐行证据，本轮不是114项全量重跑。

| 条目 | Round14 | Round15 | 判定 |
|---|---|---|---|
| A04、GOAL3 | FAIL | PASS | 原N02关闭，三人真实备份/commit/新进程加密reveal及历史公平链路证据 |
| A03、GOAL2 | PASS | FAIL | R15-N01：真实准备/开局ACK后客户端仍停H1 |
| E03、H08 | FAIL | FAIL | 原N01/N02关闭，但新状态不一致缺陷；E03另有人类备份UX未验证 |
| H05、GOAL8 | BLOCKED | BLOCKED | 内部适配/实际平台/设备发布条件未完成 |
| C09、C10 | DEFERRED | DEFERRED | 用户已正式延期的可选根迁移/退役；不是PASS，不阻塞本期 |
| 其他104项 | PASS | PASS（历史依据） | 源差异影响核查后复用历史结果；非本轮重新逐项测试 |

**当前112项=106 PASS + 4 FAIL（A03/GOAL2/E03/H08）+ 2 BLOCKED（H05/GOAL8）；另2 DEFERRED（C09/C10），总114项。** 四个FAIL条目映射到**一个已确认P2缺陷**，不是四个独立新缺陷。原R14两项问题均关闭；历史报告保持不变。

H05不是全由外部凭据导致：实际抖音入口/身份交换、可用导出及SDK适配仍缺；本轮不尝试重复发布测试。GOAL8还需正式HTTPS/WSS域名/TLS、受支持工具/真实设备、多用户前后台/网络/分享与发布授权。微信登录与AppID/AppSecret已有历史真实成功依据，不能又描述为“未登录”或“缺AppID”。beta/实验Linux导出不是发布资格。

下一轮最小开发清单：①修复R15-N01的对账/房间命令顺序和已ACK状态采用，保留全部未知请求/secret保护；②补E03实际输入、复制粘贴、长凭证可读性/丢失指引及用户备份恢复证据；③分开推进H05内部适配与GOAL8外部资格，不开发已延期C09/C10。

Goal0045、0046均继续 **in_progress**，不封版、不归档。本轮验收报告交付完成不等于产品通过。只提交本报告及current允许状态；测试私有服务停止、证据保留，未修改其他部署或清理全局Docker。
