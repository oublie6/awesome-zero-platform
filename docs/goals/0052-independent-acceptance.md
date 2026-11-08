# Goal 0052 — Scene 异步生命周期与账号隔离独立验收

执行日期：2026-11-08 UTC。执行者：Codex。独立增量验收；只提交报告及允许的Goal状态，不修改产品、正式测试、Schema、协议、生产配置或历史报告。

## 1. 结论

**最终 FAIL。G51-N01 原始跨账号缺陷 PASS，可按本轮证据关闭。** 本轮扩展验证仍确认一类 **G52-N01/P3：同账号的旧错误回调缺少选择/房间代次检查，切房间或A→B→A后覆盖当前成功提示**。三条真实HTTP反例及相关转译行为矩阵证明该遗漏；没有观察到业务状态、权限或持久化记录损坏，也没有永久阻塞。

重复create/leave、跨账号互斥finally隔离、登录前置失败、主要账号/销毁围栏及正常业务通过。协议49/49、shell89/89、typecheck通过；精确候选三套CI总体SUCCESS，原生导出SKIPPED。不能用这些通过项覆盖尚未满足的同账号旧回调必需条件。

ChatGPT负责G52-N01修复。Goal0045/0046保持 `in_progress`，E03不解除，不封版、不发布；独立验收任务交付不等于产品Goal完成。

## 2. 固定版本及完整差异

受测产品：**`74601aa0022e9241c9fd2858fbd80f982872c865`**。初始main干净，执行 `git pull --ff-only`，由上轮报告提交 `2f20673622a636781b53f696c15598d0cd9ebb99` 快进至候选；报告前远端main再次核对相同。

相对上轮产品 `e57f95af6baea6019890cae967c3e5a3eb937150`：4文件、419行新增/67行删除，包括原样保留的Goal0051报告及current状态记录。**产品/正式测试实际差异仅2文件，281行新增/67行删除**：GameBootstrap.ts新增currentAccount/currentRoom、异步围栏、create/leave owner互斥、安全菜单入口及登录代次调整；assets.test.mjs相应调整并增加4项源码断言测试。已审查完整差异，源码匹配测试只作为结构检查，未视为行为验收。

已复核AGENTS、原则、current及[Goal0050](0050-independent-acceptance.md)、[Goal0051](0051-independent-acceptance.md)。server tree仍为 `06209f0fc5a8f9bb8a50682c69cc6f195f354e18`。OnlineController、RoomCoordinator、GameClientShell、GameFairnessActions、GameHandActions均未改，重点复核调用契约而非重审其整个内部实现。平台边界、服务器权威、原请求身份和无客户端权威备份的约束保持。

## 3. CI与本地测试

已独立读取固定SHA三个run的全部job，全部completed；没有重触发CI或用其他SHA结果替代。

| 工作流 | 最终结果 |
|---|---|
| [主CI 42434786036](https://github.com/oublie6/awesome-zero-platform/actions/runs/42434786036) | SUCCESS，admin_web/integration/secure_envelope/unit/runtime/report 6/6 |
| [Fair Doudizhu 42434786074](https://github.com/oublie6/awesome-zero-platform/actions/runs/42434786074) | SUCCESS，client_foundation/application/mysql_integration/manifest_client 4/4 |
| [Cocos Linux 42434786007](https://github.com/oublie6/awesome-zero-platform/actions/runs/42434786007) | 总体SUCCESS；linux_source SUCCESS；**linux_native_export SKIPPED** |

本轮独立执行：

```sh
npm --prefix clients/packages/doudizhu-client test       # 49/49
npm --prefix clients/fair-doudizhu-cocos test             # 89/89
npm --prefix clients/fair-doudizhu-cocos run typecheck     # exit 0
```

## 4. 环境、夹具与证据边界

复用Linux、Node22.23.4、四个真实Go认证测试账号、现有MySQL5.7.44/Redis8.8.0私有验收库、HTTPS/WSS、稳定保护根及真实Quicknet；没有重建后端或下载依赖。API二进制SHA256重新核对为 `e3901e2e4b776da28eb8608affac804598233ebeafdbd975755c1af1dfefc1d1`，与相同server tree历史构建一致。本轮只复制私有测试配置，无生产配置改动。

Bootstrap使用候选实际TS转译，真实菜单和处理方法执行；基础Cocos节点/布局由私有夹具替代。真实场景使用实际Controller、Room/Hand/Fairness模块、Go API、MySQL及HTTP/WSS；ACK丢失/迟到在真实服务器结果到客户端业务层的交付边界注入，不伪造accepted或房间版本。

为执行实际 `login()` 账号替换流程，私有构造绑定返回真实测试Controller和原生Web传输platform；随后实际restoreAndConnect再次调用Go验证真实会话并连接WSS。真实账号来自Go登录，**不冒称真实wx.login/code exchange、平台SDK或真机通过**。登录构造异常、无服务端变化的旧公平/证据UI路径另用受控异步端口执行实际Bootstrap；这些不代表真实业务/密码学验收。

私有证据 `E=/root/.config/awesome-zero-platform/acceptance-0052-74601aa`：runs.jsonl记录完整argv/cwd/SHA/UTC/退出码；各场景JSON/日志、ci.json、provenance.json、reused-evidence.json、retained-hand-state.json、shutdown.json、completion-audit.json及sha256.json保存结果。账号、令牌、密钥、贡献和手牌不提交。

## 5. G51-N01原始反例及扩展

**PASS，6/6。** X真实create/ready接受并持久化，丢ACK后由实际Scene reconnect精确重试原RequestID；扣留真实重复receipt，执行实际Scene login替换至Y并恢复Y的真实等待房间B，再释放X的结果。逐项比较Y的status、roomID、handID、fairnessMode、recoverableRooms、选牌、Controller/Fairness引用、Hand快照、认证、连接状态及命令/同步/连接计数，全部保持；Y随后正常ready至v2。

独立覆盖原网络异常、accepted成功、明确rejected、再次丢ACK仍UNKNOWN、Scene销毁、另一个房间选择已开始。账号替换的四个结果场景全部走实际login；销毁后status/online为空。另房间选择在原UNKNOWN未结算时正确拒绝，旧结果不改该提示；原请求结算后可重新选择B。

每个原RequestID的MySQL整行摘要前后相同；没有向Y发送X的原命令。明确拒绝也以真实durable拒绝receipt产生，而非客户端发送前异常。证据original-expanded.json；原始异常探针A=`room-cmd-310a760c9503491d7855fd36399ef528`，RequestID=`cmd-a67cab8e463ebf05283556735d619150`。

## 6. 全部主要异步入口矩阵

实际转译的25个非login入口分别执行成功/异常交付与账号替换、房间切换、A→B→A、销毁四类交错：**200个受控行为格，193 PASS / 7 FAIL**。它们不是200项完整服务器业务测试，也不合并计入138项正式测试。7个失败格归入同一P3，未重复计缺陷。login另执行8项检查。

| 必查入口 | 结果与实际范围 |
|---|---|
| login | PASS：实际Scene替换/Go会话恢复、四种前置失败、合法重试、替换后失败及重叠登录；见第8节 |
| recoverMyRooms | PASS：当前发现/冷恢复，账号/选择/销毁后不采用旧成功结果 |
| discoverMyRoomsSafely | **部分FAIL**：当前错误可见、账号及销毁隔离通过；旧选择的迟到错误覆盖新提示 |
| resumeAuthorizedRoom、resumeSelectedRoom | PASS：旧选择失效、重复/拒绝/关闭后恢复、错误收口；实际安全菜单无未捕获拒绝 |
| reconnect | 原跨账号/UNKNOWN分支PASS；**非UNKNOWN恢复的迟到异常跨新选择FAIL** |
| createRoom、leaveRoom | PASS：重复点击、账号替换、finally owner隔离、失败释放及UNKNOWN恢复；已接受receipt保留 |
| joinRoom、readyRoom | PASS：旧账号/房间/销毁成功与错误不采用；真实加入、ready/unready及退出登录交错通过 |
| refreshRoom | 正常刷新、命令版本及选择围栏PASS；**A→B→A旧异常提示FAIL** |
| startHand | PASS：真实三人Scene开局、旧账号/房间/销毁围栏，不因UI切换抹除已接受结果 |
| submitOwnContribution | PASS：真实贡献已接受回调迟到后换账号，Y不变、原receipt不变；H/room代次受控交错通过 |
| verifyMyPhrase | PASS（受控UI边界）：两次读取绑定原online/hand；旧上下文不写UI，不读取Y的handID；未重验终局密码学 |
| refreshFairness、retryUnknownFairness | 账号/销毁及真实精确贡献重试PASS；**返回相同H1标识但新代次时，旧异常提示FAIL** |
| prepareFairness | PASS（受控）：账号/Hand/room/screen/input变化后不重显旧凭证，不操作销毁UI |
| importFairness | 账号、销毁、正常屏幕替换保护PASS（受控）；**A→B→A同Hand标识旧异常提示FAIL** |
| commitFairness、revealFairness | PASS（受控）：捕获原fairness/hand/room/epoch，旧成功不启动新账号查询，旧错误不写新上下文 |
| bid、play、pass | PASS：真实Go三类命令接受、ACK迟到/丢失、换账号/销毁与正常恢复；选牌不越界清理 |
| refreshHand | PASS：使用原online及快照handId；旧错误隔离，当前真实同步/恢复正常 |
| retryUnknownHand | PASS：真实精确原Play请求重试中换账号，不用Y的handID继续同步；当前Pass原请求重试可继续 |
| showEvidence | 成功结果与账号/销毁隔离PASS（受控）；**A→B→A同Hand标识旧异常提示FAIL** |

`currentAccount`只证明Controller引用、登录epoch和content有效；`currentRoom`另比较roomID与roomGeneration。不能用仅ID相等替代代次：A→B→A仍是新上下文。成功分支大多使用完整围栏，若干catch退化为currentAccount加字符串ID相等，形成第11节问题。

检查了所有相关await后的写入、catch、finally和二次调用。verifyMyPhrase在第一旧读取返回后仍会尝试**原账号、原hand**的ownContribution读取，但不会改用新账号ID或写新UI；未将该绑定原上下文的只读调用另报为权限/业务缺陷。create/leave finally按mutation对象引用释放，新账号owner不会被旧finally清除。

仅屏幕变化不一律等于取消已接受业务。额外3项受控屏幕测试通过：旧prepare/import结果在home→rooms输入框替换后不重显凭证；同账号同房间ready跨屏仍正确采用确认结果且不强制跳回房间屏幕。账号/Hand替换或销毁则旧结果失效。证据ui-matrix.json、menu-matrix.json、screen-semantics.json；不以源码字符串匹配替代行为。

## 7. 重复create/leave及互斥状态

**8/8通过**，见room-mutations-initial.json中的6个有效通过场景及room-mutations.json中的2个UNKNOWN补验。

- create/leave第一次真实接受但ACK扣留时再次点击，只产生一个原命令身份；不递增第二次选择代次，第一成功仍采用，create最终绑定新房间，leave最终清空Scene。
- X操作在途时实际login替换为Y；Y可立即持有自己的操作，X结果与finally不改Y状态或释放Y的mutation。Y再点也不产生新冲突请求，完成后互斥释放、正常命令可用。
- create/leave受控发送前失败后互斥释放，随后真实合法命令成功。此两项证明UI失败释放，不冒称真实服务端create拒绝。
- 接受后丢ACK：create保留原UNKNOWN，不能新ID建房；leave再次点击按原Controller契约直接精确重试原leave并清空Scene。原MySQL整行摘要不变，后续建房、准备、取消准备、退出正常。

夹具已有Goal0050/0051留存active房间：无Scene locator的发现按既有规则优先唯一active，可能先恢复留存局；本轮UNKNOWN create结算后通过实际Scene选择原创建房间继续，未删除留存局或改产品发现规则。不能将“存在其他active时始终自动优先新waiting房间”当作本轮已证明行为。

## 8. 登录失败及旧会话保护

login-lifecycle.json **8/8 PASS**。缺配置、缺SDK、platform构造异常、Controller构造异常均发生在递增requestGeneration和替换旧Controller之前；旧账号认证、Room、Hand、Fairness不变，原通知闭包和已注册snapshot事件仍有效，随后合法重试成功。构造异常由私有绑定注入，不声称实际SDK提供方出现过该故障。

新Controller已经建立并替换旧会话后，受控SDK登录失败：旧Controller已dispose，新Controller未认证，Scene room/hand清空并显示登录失败，没有错误宣称旧账号仍有效。重叠登录中，旧restore迟到返回false时，旧disposed Controller的loginAndConnect守卫阻止继续认证，不影响已成功的新账号。

## 9. Hand/Fairness迟到结果与正常业务

只创建**一个新真实三人局**，通过实际Scene开局、三份服务器托管贡献和真实信标推进，执行Bid(3)、一张合法Play及一次合法Pass，不重打完整游戏。

- 已接受贡献回调迟到后实际login换账号：新账号状态/选牌/查询不变，原贡献receipt不变；原账号新Controller从服务器确认本人revealed。
- Fairness UNKNOWN精确重试真实收到原receipt后扣留，再换账号：不刷新新账号公平UI，不向新账号重发；另一玩家当前账号正常Scene公平精确对账也通过。
- 实际Bid接受后销毁Scene再放ACK：无旧UI写入，live receipt不变，原账号新Controller可从服务器恢复正确PLAYING状态。
- Play接受丢ACK，Scene retryUnknownHand用完全相同原请求；真实重复receipt扣留后换账号，不清新选牌、不对新账号Hand同步，live记录不变。
- Pass接受丢ACK，当前账号正常Scene原请求重试成功、pending清除、仍可同步PLAYING。
- 只带账号认证的空Controller发现/选择本人active房间，恢复H并真实断开WSS再重连，零新room命令，不使用本地牌局备份。

证据hand-lifecycle.json。保留私有A=`room-cmd-c3e7c008bcb13f700e8ea523b9317bf3`、H=`hand-cmd-0574deb90fba5d90a07c2e837ce12c59`；只读MySQL核对3条贡献、3条live action，见retained-hand-state.json。仅保留验收卷证据，不为清理再跑整局；后续测试须考虑此局及原历史留存局。

正常路径要求全部在上述真实流程中覆盖：Go登录/Scene会话恢复、自动发现、create/join、ready/unready、三人开局、本人字符串/托管贡献、断线恢复、Room/Fairness/Hand原请求对账、Bid/Play/Pass、等待房退出/切换和仅服务器状态冷恢复。当前错误仍可见，未出现永久互斥或已接受命令丢失。旧公平凭证、终局证明UI的异步边界为受控验证，非本轮完整历史模式/密码学重验。

## 10. 菜单异常与历史定向回归

实际菜单DiscoverMyRooms及Resume action分别触发当前发现异常、已关闭房间恢复拒绝：错误可理解，无unhandledRejection，保持合法B，随后恢复/ready正常。账号切换、选择乱序和销毁另由行为矩阵及真实三反例覆盖；旧发现错误提示越界另FAIL，见下节。没有放宽Go成员授权。

| 历史问题 | 本轮证据与结论 |
|---|---|
| G48-N03 | 新执行6个logout/销毁/新账号交错，全PASS，原Controller状态不能迟到复活 |
| G49-N01 | 原始同房间旧读开始后ready接受丢ACK反例新执行PASS；原UNKNOWN完整保留，精确重试wire与receipt不变，恢复v2 |
| G49-N02 | 未改的采用前置契约及Goal0051真实跨组件矩阵证据复用；本轮贡献/Hand UNKNOWN、换账号和正常恢复新执行，不另造一局重复原子切换验收 |
| G50-N01 | 受本次Bootstrap影响的普通刷新防ACK回退、旧刷新与新选择成功响应交错新执行PASS；原member join/命令先开始两个反例沿用Goal0051未变核心证据 |
| G50-N02 | 本轮Room原请求Scene对账、create/leave UNKNOWN及正常恢复新执行PASS；可见菜单原始行为沿用Goal0051证据，未冒称本轮重跑该完整原始菜单探针 |
| G51-N01 | 原始跨账号异常及扩展新执行PASS，见第5节 |

## 11. 新发现 G52-N01 / P3 — 同账号旧异常缺少选择/房间代次围栏

**真实最小反例一：** A=`room-cmd-b6bc65e173656efa0b47ebbfdf5904d6`的普通刷新得到真实Go结果后扣留；实际恢复B再恢复A，当前提示“已通过服务器恢复本人房间和牌局”；将旧HTTP交付以网络异常结束，提示变成“刷新失败：old room HTTP response lost”。`refreshRoom`成功路径检查roomGeneration/recoveryAttempt，**622–624行catch只检查账号及originalRoom字符串**，将重入A错误当作旧A仍当前。

**真实反例二：** 旧discoverMyRoomsSafely读取被扣留后选择本人B成功，再释放旧发现网络异常，B提示被改成“查询房间失败：old discovery response lost”。293–296行错误路径只检查currentAccount，未比较该次发现选择。A=`room-cmd-ed6e25416070c62d5b82b86f0060ecf6`，B=`room-cmd-8fcc01cb2b76d35dc3161f87d902ddfd`。

**真实反例三：** 无Room UNKNOWN时旧reconnect(A)在真实HTTP边界扣留；后选B完成，再释放旧A异常，B提示被改为“重连失败：old room HTTP response lost”。397–426行只在UNKNOWN内部检查initialChoice，外层catch仅检查currentAccount。A=`room-cmd-eb7dac54de40cdf9ba7527e36d7dcab1`，B=`room-cmd-abbf3d7b8a782de1b99c512d45923c6f`。

三例均逐项验证除status外业务状态/引用/连接/认证不变，原create MySQL receipt摘要不变，随后真实ready成功。证据stale-errors.json。没有数据回滚、额外业务命令、权限泄漏或永久阻塞；不是G51原跨账号反例继续失败。

受控实际方法矩阵进一步复现相同代次遗漏：refreshFairness catch647–650、retryUnknownFairness catch440–443、importFairness catch783–786、showEvidence catch849–852，在A→B→A恢复相同Hand标识但不同generation后，旧异常写当前status。只将其记为已执行的UI行为，不冒称另外执行了真实终局或旧版凭证业务；Fairness/import真实切换还受Controller在途/UNKNOWN约束，受控格不独立证明全部底层前提允许该完整业务时序。最终产品FAIL依据上面三条真实可达反例，同类条件遗漏合并一项P3，不重复计数。

预期：旧选择/旧代次的失败和成功遵守相同生命周期资格。最小建议：在入口捕获账号、选择及适用的room/hand代次；catch使用与该操作成功采用路径一致的上下文验证。不能直接比较reconnect入口initialChoice与所有后续结果——它自身可能启动新选择，应明确子恢复的归属。保持当前合法错误可见，保持已接受receipt/UNKNOWN精确身份，不通过静默全部错误、清UNKNOWN或新ID规避。补充本节三条真实时序与同ID重入异常的行为测试；现有仅源码currentAccount字符串断言无法发现该缺陷。

## 12. 执行纠偏、复用和未验证风险

夹具错误未算产品FAIL：API首次先于MySQL就绪而退出，确认数据库healthy后通过原supervisor启动同一个API；首次原始测试未到业务前提即结束，原始场景随后全通过。重复操作夹具曾错误要求UNKNOWN create在已有唯一active时自动选择新waiting，以及错误要求leave第二次点击仍保持UNKNOWN；已按既有发现/原请求重试契约纠正，只补跑两个UNKNOWN。未使用的延迟门超时曾在脚本结束时产生unhandled rejection，改为按访问创建超时；初次6项通过数据保留，后续菜单异常专用监听确认零产品未捕获拒绝。产品新FAIL只来自隔离后的真实/受控时序，非这些前提错误。

复用相同server tree及五个未变核心模块契约，重新核对API二进制和20个历史证据文件摘要。HPKE/AEAD、drand验证、洗牌、完整游戏、最终证明、持久化故障恢复沿用既有报告，不声称本轮重新执行。实际贡献/信标用于生命周期与正常路径验证，不等于重做密码学审计。未重复全Go/race/性能/架构测试或完整原生构建。

本候选Linux原生导出SKIPPED；新的原生SDK、微信code exchange、真机及正式发布未验证，不能由受控构造绑定/节点夹具或绿CI替代。这些外部资格与本轮代码专项FAIL分开记录。

全部可执行专项检查完成后，自有API/TLS/MySQL/Redis验收服务停止、私有卷和证据保留。最终仅本报告及current允许状态段提交main；不修产品、不修改历史FAIL。ChatGPT应修复G52-N01并提供新固定SHA，再独立复验相关异常围栏和正常恢复。
