# Goal 0053 — 同账号异步生命周期与恢复一致性独立验收

日期：2026-11-08 UTC。执行者：Codex。独立增量验收，仅报告及允许的状态更新，无产品、正式测试、Schema、协议、部署配置或历史报告修改。

## 1. 结论

**最终 FAIL。G52-N01 三条真实原始反例及同类异常围栏均PASS，可按本轮证据关闭。** 新发现 **G53-N01/P2：Scene新建了当前A恢复Promise，但Controller仍按房间key复用已失效的旧A恢复**。通过实际成员join入口切到本人B后再恢复A，最新A没有新权威读取，最终仍停在B；旧请求结束后再次操作才恢复A。没有服务端数据损坏或额外业务命令。

自有子恢复代次、Scene同代次合并、Scene旧finally、跨账号同名房间隔离、334格受控异步行为和正常真实业务均通过；不能据此覆盖上述必需恢复条件失败。ChatGPT负责修复。Goal0045/0046保持 `in_progress`，E03不解除，不封版、不发布。

## 2. 固定版本及差异

产品 **`c722de2916aacbe348b17c9e82f77af547ace3e3`**。干净main执行 `git pull --ff-only`，由 `d07d2385a053cb30a238bb8610453a1f793c743d` 快进至候选；收尾再次核对远端main相同。文档提交与受测产品SHA分开。

已复核AGENTS、原则、current及[Goal0051](0051-independent-acceptance.md)、[Goal0052](0052-independent-acceptance.md)，这些既有要求/报告未被候选修改。完整差异仅2文件，175行新增/78行删除：

- GameBootstrap.ts：currentRoom增加可选selection；成功/错误分支统一代次；发现向父调用传回自行启动的子恢复token；reconnect跟踪ownedSelection；sceneRecovery增加owner/selection缓存身份，finally仍按Promise引用释放；refreshRoom自行采用Hand后更新本次generation以保留合法错误反馈。
- assets.test.mjs：既有源码断言适配及4项新增源码结构测试。未将正则匹配当作行为证明。

server tree仍为 `06209f0fc5a8f9bb8a50682c69cc6f195f354e18`；OnlineController、RoomCoordinator、Shell、Hand/Fairness模块均未变。按调用契约审查这些模块，未开展无关架构、密码学或性能审计。第7节缺陷位于该实际恢复调用链，不能仅因Controller未改而忽略。

## 3. 精确候选CI及客户端门槛

独立读取三个run及全部job，SHA均为候选、均completed；未重触发CI或重复轮询。

| 工作流 | 最终结果 |
|---|---|
| [主CI 42437681322](https://github.com/oublie6/awesome-zero-platform/actions/runs/42437681322) | SUCCESS，6/6 |
| [Fair Doudizhu 42437681312](https://github.com/oublie6/awesome-zero-platform/actions/runs/42437681312) | SUCCESS，4/4 |
| [Cocos Linux 42437681257](https://github.com/oublie6/awesome-zero-platform/actions/runs/42437681257) | 总体SUCCESS；linux_source SUCCESS；**linux_native_export SKIPPED** |

本地新执行 `npm --prefix clients/packages/doudizhu-client test` **49/49**，`npm --prefix clients/fair-doudizhu-cocos test` **93/93**，`npm --prefix clients/fair-doudizhu-cocos run typecheck` exit0。共142项正式客户端测试通过，不等于专项整体PASS或本候选原生导出成功。

## 4. G52-N01三个真实原始反例

实际转译Bootstrap、真实Go认证账号、MySQL、HTTP/WSS执行，扣留真实HTTP结果后在交付边界结束为网络异常，不伪造服务端版本或业务结果。

| 原反例 | 本轮结果 |
|---|---|
| 旧refresh(A)失败，A→B→A | PASS：新A成功提示保持；Room/Hand/Fairness、账号、连接均不变；最新刷新正常，当前合法刷新失败仍显示错误 |
| 旧discover失败，显式恢复B成功 | PASS：B状态、recoverableRooms和成功提示不变，不启动旧自动恢复；当前新发现失败仍可见 |
| 非UNKNOWN旧reconnect(A)失败，B已恢复 | PASS：B提示和成员状态保持，不产生额外RequestID，正常ready随后成功 |

原create receipt整行摘要前后不变。当前菜单发现异常/已关闭房间恢复拒绝亦通过，无未捕获Promise拒绝，之后恢复及命令可继续。证据stale-errors.json记录交付顺序、请求路径、前后提示及状态检查。

例如首例A=`room-cmd-0004c488e806569e42d9691382bf25c9`、B=`room-cmd-b5e4d616cf6b8966fd0bb299c5ede416`，顺序为“旧A真实结果扣留→B恢复→新A恢复→旧结果网络异常”；前后均为“已通过服务器恢复本人房间和牌局”，随后ready至v2。

## 5. 自有选择代次与正常错误反馈

owned-selection.json **10/10 PASS**：当前发现失败、发现后自动恢复A成功、自动子恢复失败、登录自动子恢复失败、自动恢复中用户改选B、当前reconnect成功/失败、Room UNKNOWN精确对账与普通刷新并发、重连中换账号、Scene销毁。

为避免历史留存active房间干扰唯一候选，本组自动发现把**真实Go发现列表过滤到本人真实A**；之后子恢复的权威读取、成员校验、WSS及错误交付仍为真实产品调用。此受控发现输入不冒充完整发现端点的列表资格验收；第4节原始反例和正常冷发现使用未过滤端点。

父发现由onOwnedSelection获取自身token及子恢复token：当前子失败仍可见；后来用户B选择则使旧父/子结果失效。UNKNOWN与非UNKNOWN重连均通过；真实ready接受丢ACK后保留原RequestID/clientSeq/expectedVersion/payload，原请求重试等待期间普通刷新不能清UNKNOWN，原receipt结算后正确恢复A，整行摘要不变。原RequestID=`cmd-13f04984295142c4fd309da1e1973962`。

这组通过只证明已执行的选择归属条件，不代表所有入口间的Controller缓存身份正确；后者另见第7节。

## 6. Promise合并与finally

| 实际场景 | 结果 |
|---|---|
| 同Controller、同房间、同选择的重复恢复 | PASS：同一Scene Promise；无重复业务命令，重复点击不增加连接；总opens2包含初始连接和首次恢复连接 |
| A旧读未结束，经Scene恢复B，再Scene恢复A | PASS：新A进行新权威读取；旧A结束时不清新A在途记录；重复新A仍合并；最后A成功 |
| A旧读未结束，经实际成员join恢复B，再Scene恢复A | **FAIL，G53-N01**：Scene缓存已更新，Controller旧缓存仍被复用 |
| X旧A在途，Y为A合法成员并恢复同名A | PASS：实际Scene登录替换，不跨Controller/账号复用，X迟到不改Y状态 |

证据recovery-identity.json。以上直接使用产品入口和真实成员关系；没有手工塞入pendingRecovery或修改Scene生命周期字段来制造失败。

## 7. G53-N01 / P2 — Controller按旧key复用失效恢复

真实最小反例：

1. X是两个waiting房间的合法成员。A=`room-cmd-5a12d7df6787fc70c20d4c8bf63b515e`，B=`room-cmd-2cb32aa94bfa38c519bf44188c159937`。
2. 实际Scene恢复A，扣留Controller第二次、最终真实成员读取；A已读取2次，原恢复尚未采用。
3. 在实际房间输入框填B并调用Scene joinRoom。Go确认已有成员，正常采用B。此入口会推进Controller恢复代次，却不覆盖Scene/Controller原A在途记录。
4. 再调用Scene恢复A。新的Scene Promise/记录与原A不同，但Controller `reconnectAndResync(A)`复用原core Promise。期间A读取仍为2，没有新A权威读取。
5. 释放旧A结果：旧Controller恢复因代次过期返回null，新的Scene调用也随之返回，**Scene及Controller均仍为B**。再显式操作一次A才执行4次新读取并成功。

根因：`GameOnlineController.ts:70–71`只按pendingRecovery.key合并，校验缓存资格发生在递增新recoveryGeneration之前；其旧performRecovery在83–85行检查旧token，最终返回null。`GameBootstrap.ts:348–354`虽拒用旧Scene缓存，365–366行仍接收到复用旧core Promise的null并结束，因此Scene修复没有贯通Controller。

预期最新A必须用当前有效身份重新权威读取并恢复A；实际合法选择静默落空，需再次操作。**影响P2：恢复功能失败，非单纯旧错误提示覆盖；未造成服务器回滚、权限泄漏或不可恢复数据损坏，也未证实永久阻塞。** 失败过程零room命令，初始两个accepted create receipt仍在MySQL，见failure-receipts.json（事后存在/摘要核对，不冒称无基线的前后摘要比较）。

请求路径和失败结束状态在recovery-identity.json单独记录，后一次显式重试的读取另列；没有把补救成功混写为第一次新A成功。普通Scene B恢复路径通过，不意味着经其他合法选择入口切B也通过。本例直接落在本轮同房间再次恢复身份要求内。

最小建议：Controller的在途恢复缓存同样绑定有效恢复/成员/命令生命周期，资格失效时不能仅因key相同返回旧Promise。保留真正同代次请求合并及按Promise引用释放的finally；不要简单禁止join、吞掉null、清UNKNOWN或新建业务RequestID规避。由ChatGPT补正式行为回归并提交新SHA，Codex不修产品。

## 8. 合并异步矩阵与过度围栏对照

实际候选Bootstrap通过私有受控异步端口执行：25入口×成功/异常×账号替换、room变化、ABA、销毁、仅selection变化、仅selection ABA，**300/300 PASS**；另17个当前上下文入口分别成功/失败，**34/34 PASS**。二者与142项正式测试分开统计，不冒充334项完整Go业务测试。

四个同类ABA入口refreshFairness、retryUnknownFairness、importFairness、showEvidence的旧成功及旧异常均不写新上下文；当前合法异常仍可见。ready/startHand/贡献/个人核验/历史prepare、commit、reveal/refreshHand/Hand重试/Bid/Play/Pass同样检查了选择资格、成功后续、错误提示及选牌不被旧结果清空。当前操作没有被新selection判断无故全部静默。

这些受控格验证Scene行为，不独立证明历史凭证/终局密码学或每个底层inFlight/UNKNOWN约束均允许对应真实跳转；最终FAIL只依据第7节真实可达反例。实际账号替换的构造绑定、基础Cocos节点夹具沿用Goal0052方法，不冒称真实微信SDK/真机。

## 9. 真实正常业务及复用限制

新执行：Go登录/Scene会话恢复、未过滤自动发现和手动选择、create/join/ready、退房再加入、实际Scene三人公平开局、本人随机字符串/托管贡献、Room/Fairness/Hand原请求对账、Bid/Play/Pass、空Controller仅凭服务器恢复active H及真实WSS断开后重连。当前refresh自行采用新Hand后，后续fairness查询失败仍显示错误；下一次正常刷新可继续。

normal53.json前两组通过真实建房到开局及贡献接受丢ACK/Scene精确重试，贡献receipt不变；该局仅一份贡献后正常取消、关闭房间。随后尝试复用Goal0052 live局，真实phase已为EXPIRED，故其后续动作未执行，脚本exit1明确保留，**不把过期夹具当产品缺陷或已通过的新出牌证据**。

为补齐未执行动作，live53.json仅启动一个最小新局：真实三份贡献/信标到BIDDING、Bid(3)、一次合法Play、一次Pass；Play接受丢ACK后用完全相同原命令精确重试，live整行摘要不变；当前Hand同步及冷恢复继续可用。**6/6 PASS，无完整游戏/终局证明重跑。** 保留A=`room-cmd-8e4277041d10f292e66779310009d7e2`、H=`hand-cmd-2c1c5ae607b4365b967d9beec193bcbb`于私有验收卷，MySQL核对3份贡献、3条live action。其他自有测试等待/取消房间正常关闭；历史留存局不删除。

## 10. 历史复验、环境与最终交付

G52-N01三个真实反例新执行PASS；G51-N01/G48-N03的账号隔离由本轮真实换账号及300格加强，原六场景/登出报告保留并核对摘要。G49-N01/N02、G50-N01/N02未变核心矩阵/版本水位/完整原菜单探针复用历史证据；本轮另新执行旧成功/异常隔离、Room UNKNOWN+刷新、精确身份及正常Scene恢复，不声称完整重跑所有历史原反例。

复用Linux、Node22.23.4、真实Go测试账号、原MySQL5.7.44/Redis8.8.0隔离库、HTTPS/WSS及稳定保护根；不重编后端、不下载依赖。API二进制SHA256仍为 `e3901e2e4b776da28eb8608affac804598233ebeafdbd975755c1af1dfefc1d1`，相同server tree/五模块契约核对，28个历史证据摘要重新核验。HPKE/AEAD、drand验证、完整洗牌/游戏/最终证明、故障恢复沿用已有可信结果，不重复做整套审计。

私有目录 `E=/root/.config/awesome-zero-platform/acceptance-0053-c722de2` 保存runs.jsonl（argv/cwd/SHA/UTC/退出码）、ci/provenance/reused-evidence、各场景JSON/日志、failure-receipts、retained-hand-state、shutdown、completion-audit及sha256索引。秘密、令牌、贡献和手牌不提交。

全部可执行专项内容完成；自有API/TLS/MySQL/Redis验收服务停止，证据和私有卷保留。未执行本候选原生导出、真实平台SDK/真机、生产发布或全Go/race/性能复审，不将SKIPPED/受控夹具当作这些门槛通过。仅本报告/current允许状态提交main，保留所有历史FAIL；产品修复交回ChatGPT。
