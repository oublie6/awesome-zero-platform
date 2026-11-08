# Goal 0054 — 恢复缓存与异步命令生命周期独立验收

日期：2026-11-08 UTC。执行者：Codex。独立增量验收，无产品、正式测试、协议、Schema、部署配置或历史报告修改。

## 1. 结论

**Goal0054本次代码修复检查点 PASS。G53-N01/P2原始真实反例通过，可按本轮证据关闭。** 恢复缓存、异步建房、旧Fairness对账后开局、旧连接消息与正常业务对照均在以下明确范围内通过，没有新确认的产品缺陷。

这不是Goal0045/0046完成或发布资格：两者继续 `in_progress`，E03不解除；原生导出、平台SDK/真机及正式发布条件单独评估。历史FAIL原文保留，不倒改为PASS。

## 2. 固定候选与完整差异

产品 **`fcca57d6f29b724cb193c3d652a1bd2a7cd230ca`**。main初始干净，执行 `git pull --ff-only` 从上轮报告提交 `94e778193c6679a1a6b8ee20d3ea84e5641a2133` 快进至候选；收尾再次核对远端main相同。文档提交与受测产品SHA分开。

已复核AGENTS、原则、current、[Goal0053](0053-independent-acceptance.md)及相关历史证据。完整差异5文件，214行新增/12行删除：

| 文件 | 变更及独立审查 |
|---|---|
| GameOnlineController.ts | pendingRecovery绑定key/token/membership/commands；create在CSPRNG前预留选择、发送前及durable ACK后校验归属；startHand在旧Fairness重试后复核原room/代次/命令上下文 |
| GameClientShell.ts | ConnectionManager交付消息后、路由前检查Shell generation/disposed |
| online.test.mjs | 5项新增行为测试；审查其夹具/断言，另执行真实Go反例和交叉验证 |
| shell.test.mjs | 新增旧消息行为测试；独立补足5类消息及实际同ID ACK对照 |
| assets.test.mjs | 缓存身份源码断言适配，仅作为结构门槛 |

Bootstrap、RoomCoordinator、Hand/Fairness及ConnectionManager未改，重点复核调用契约与锁/未决身份，不重做整个内部审计。没有新增后端/平台依赖或客户端牌局备份。server tree仍为 `06209f0fc5a8f9bb8a50682c69cc6f195f354e18`。

## 3. 精确候选CI与本地门槛

独立读取以下固定SHA run及全部job，均completed；未重触发成功工作流。

| 工作流 | 最终结论 |
|---|---|
| [主CI 42442450445](https://github.com/oublie6/awesome-zero-platform/actions/runs/42442450445) | SUCCESS：integration/secure_envelope/admin_web/unit/runtime/report 6/6 |
| [Fair Doudizhu 42442450441](https://github.com/oublie6/awesome-zero-platform/actions/runs/42442450441) | SUCCESS：mysql_integration/client_foundation/application/manifest_client 4/4 |
| [Cocos Linux 42442450503](https://github.com/oublie6/awesome-zero-platform/actions/runs/42442450503) | 总体SUCCESS；linux_source SUCCESS；**linux_native_export SKIPPED** |

本轮独立执行协议测试49/49、shell测试99/99、typecheck exit0：

```sh
npm --prefix clients/packages/doudizhu-client test
npm --prefix clients/fair-doudizhu-cocos test
npm --prefix clients/fair-doudizhu-cocos run typecheck
```

148项正式客户端测试与CI是支持证据，不单独证明整个Goal。SKIPPED不计本候选原生导出成功。

## 4. G53-N01真实原始反例

**PASS。** 使用实际转译Bootstrap、真实Go账号、MySQL和HTTP/WSS；X为A/B已有成员。扣留Controller旧A最终成员读，经实际Scene member join(B)正常采用B，再Scene恢复A，**新A在旧A仍扣留时完成新的权威读取并成功，无须再次点击**；最后放行旧A不改当前状态。

A=`room-cmd-648d4bf5a2957feb4d136dc1017375f5`，B=`room-cmd-03c9052d988115104b65a6afcebdb97c`。新Scene及core Promise均不同于旧A；A读取从旧2次增至6次（新Controller两次权威读取，加Scene刷新/最终核验）；旧结果释放后仍为A，提示成功，Room/Hand/Fairness一致，零业务命令。原create receipt整行摘要前后相同。

证据recovery-identity.json记录请求路径、读取计数、Promise身份及状态。私有观察只读取字段；没有手改pendingRecovery、代次或Controller状态。额外补救性重复恢复成功另记录，不用它替代第一次新A的成功。

## 5. Controller缓存生命周期矩阵

| 场景 | 结果及依据 |
|---|---|
| 同账号/同房间/同有效代次 | PASS：同Scene Promise，重复不增加连接/读取或业务命令；首次恢复仍只一次连接尝试 |
| 旧A→Scene恢复B→新A | PASS：新权威读取，旧finally不清新A记录，最新A正确采用 |
| 旧A→member join(B)→新A | PASS：第4节原始反例 |
| 旧A在途，ready成功 | PASS：新core Promise不复用旧缓存，恢复v2；旧A返回null不回退 |
| room命令实际已提交、ACK扣留 | PASS：新恢复有新身份，但采用被真实inFlight保护拒绝；ACK后可新读恢复v2 |
| room命令明确rejected | PASS：拒绝完成代次改变，新恢复正确启动，UNKNOWN不滞留，原receipt不变 |
| room UNKNOWN | PASS：新恢复不能读清pending；原RequestID/payload/clientSeq/expectedVersion保留，精确重试后恢复v2，receipt摘要不变 |
| 旧完成而新仍在途 | PASS：Scene及Controller源码均按Promise引用释放；真实旧A放行时新Scene记录保留，重复新A仍合并 |
| logout/dispose旧返回 | PASS：6个实际登出/销毁/换账号交错，无迟到状态复活 |
| Y使用同名A | PASS：Y为真实合法成员，实际账号替换后新实例恢复A，不与X共用缓存，X迟到不改Y |

证据recovery-identity.json、cache-epochs54.json、logout.json。源码缓存命中同时比较key、当前recoveryGeneration、membershipGeneration及rooms.commandGeneration；实际read/cmd交错观察到这些身份随合法操作更新，不靠禁用合并获得通过。RoomCoordinator、Hand/Fairness是单个原请求+inFlight/transition保护，没有同类仅按房间key的Promise缓存；本轮原请求/锁释放对照也通过。

## 6. 异步建房：发送前、UNKNOWN、已知ACK分开判断

**发送前过期 PASS。** 在真实CSPRNG已生成但尚未交付room ID时扣留，实际Scene恢复本人active A/H1。释放旧结果后零room.create，Controller/Scene/Hand/Fairness、连接和提示逐项不变；当前公平刷新正常。不是发送前throw替代真实随机等待。证据fairness-create54.json。

**正常/重复/失败释放 PASS。** 实际create持久化后采用正确room/version，ready/unready、恢复和leave可用。重复create/leave、X操作中替换Y、旧finally不能释放Y锁、发送前受控失败后真实重试、两类UNKNOWN合计8/8通过。发送前受控失败两项不冒称真实Go拒绝。证据room-mutations.json。

**发送后选择交错 PASS。** 两个真实create已接受、ACK分别正常扣留/丢失的场景中，B恢复在room命令在途时被现有保护拒绝，未伪造已采用B。已知ACK保留A的RoomCoordinator结果；丢ACK保留完整原pending，Scene原ID重试结算。随后正常选择B并ready成功，A仍可由服务器发现，原receipt前后相同，仅一个create业务RequestID。证据create-after-send54.json。被拒绝的在途切换和已确认采用新房间不是同一事实。

## 7. 旧Fairness对账与开局

**真实交错 PASS。** A/H1的本人贡献真实接受但丢ACK；调用Controller startHand(Hstale)，内部精确重试原贡献并扣留真实重复receipt。用户经实际member join(B)开始新选择，将B首个真实读取暂缓。先放行旧Fairness：原请求正确结算，但旧start因选择资格变化拒绝，没有发送新的room.hand.start；锁释放后放行B读取，B正常成员采用，Hand/Fairness/version不受旧开局污染。原Fairness整行receipt摘要不变。

A=`room-cmd-31ef0ffd0f0f46a3f7ad91b3a3949b2f`，B=`room-cmd-64b1c56791ee1d7790e5cdc7a6c0b116`，原贡献RequestID=`cmd-e7438ea27f4ae2b19d63b51e48877090`。B在旧UNKNOWN尚未结算时没有被采用；未直接改pending/锁或绕过Hand/Fairness资格。startingHand及Hand transition均释放，B后续ready/unready正常。

**合法后续 PASS。** 原请求结算后通过真实恢复返回A，正常取消旧预发牌H1，权威读取/准备后由当前稳定选择的Scene启动H2，获得durable ACK并采用Hand/version，三份本人字符串/托管贡献继续成功。旧H1仍active时不能把下一局直接伪造为合法；本轮如实遵循取消、刷新、准备的业务顺序，不声称“旧局active时一次start直接成功”。证据fairness-create54.json。只有新H2推进至BIDDING及3个live动作，没有重跑完整游戏或终局证明。

## 8. WebSocket旧消息与当前连接对照

**PASS，五类均覆盖。** 私有夹具捕获真实Go帧，在ConnectionManager过滤之后、Shell路由入口之前重放已排队的旧回调，直接验证Shell新增generation/disposed检查。4类active帧来自本轮真实S1：hand.changed、hand.snapshot、command.result、error；terminal evidence另用历史已完成局的真实授权Go WSS/HTTP读取，在独立S1/S2对上验证同Hand标识。没有修改帧的账号、Hand、版本或ACK身份。

| 对照 | 实际结果 |
|---|---|
| 旧changed/snapshot/error | 不进入当前路由，不改S2 HandStore、connected状态、UI提示或私有牌 |
| S2相同RequestID命令等待真实ACK | 先扣留S2的真实duplicate ACK；旧S1 ACK/error不能结算/拒绝新pending，原entry保持 |
| 放行S2真实ACK | 正常解析/结算，pending移除，MySQL原receipt不变；当前真实snapshot/同步正常 |
| 旧S1关闭回调 | 不改S2连接、状态或pending |
| 当前真实断线、dispose后旧帧 | 正确进入disconnected；dispose后两代回调均无UI/状态写入 |
| 同Hand旧terminal evidence | S2已认证并映射真实COMPLETED证据后，S1旧evidence不使其退回同步、不改终局/提示 |
| 当前terminal evidence | S2仍正常处理授权evidence及HTTP终局映射，保留合法COMPLETED状态/传输资格，没有屏蔽所有事件 |
| 旧open成功/失败，新连接已就绪；dispose后成功/失败 | 4项实际Shell/ConnectionManager受控socket端口对照通过，旧结果不改新连接；不冒称真实SDK握手 |

证据ws54-initial.json、ws54.json、ws-initial-frames.json、ws-stage2-frames.json、ws-private-frames.json及connection-callbacks54.json。current pending原RequestID=`cmd-69ab407941e578ad0e2e491050bd4ac4`。terminal局为 `hand-cmd-f3483caab6e4a51ac22a9a2c185151c5`，以真实有座位账号读取；不声称从已关闭房间执行了Scene active恢复或重新完整验算终局密码学。

## 9. 正常回归、执行纠偏及历史复用

本轮新执行真实账号/会话与空Controller恢复、自动发现/手选、已有成员重入、create/join/ready、旧局正常取消后新公平开局与3份贡献、Room/Fairness/Hand精确对账、实际Bid/Play/Pass、WSS同步及断线、退房/再操作。Hand Play接受丢ACK后原wire精确重试、live receipt前后不变。normal-live-summary54.json按真实Go ACK及MySQL核对3条唯一live动作；没有为修夹具重复已通过动作。

两次WS夹具纠偏完整保留，不计产品FAIL：首次误选仍有live snapshot的历史局等待evidence而超时，后改用已有真实COMPLETED archive；第二次把同期合法S2快照计进“旧消息路由应为0”，修为旧回调同步交付前后计数差，保留所有状态/pending断言。失败后夹具未消费的pending在dispose时引发本地Promise拒绝，已为夹具清理附处理；没有改正式测试或产品。补验只重放消息及剩余对照，未重复Bid/Play/Pass。

G53-N01新执行关闭；G48登出原例、G49/G50原请求及命令代次、G51账号隔离、G52/53旧Scene回调与自有选择证据按受影响部分新执行或复核。Bootstrap未改，未重复300+34格全矩阵；保留原报告和可核验结果。HPKE/AEAD、drand、完整洗牌/牌局/最终证明、服务端故障恢复及性能/部署沿用可信历史，不做无关全量审计。

## 10. 环境、限制及交付

复用Linux、Node22.23.4、四个真实Go认证测试账号、原MySQL5.7.44/Redis8.8.0私有库、HTTPS/WSS和稳定保护根，无重新安装依赖/重编后端。API二进制SHA256仍为 `e3901e2e4b776da28eb8608affac804598233ebeafdbd975755c1af1dfefc1d1`，相同server tree、未变契约及34个历史文件摘要重新核对。一次H1预发牌局正常取消，一次H2仅Bid/Play/Pass后保留私有卷；其他自有等待测试房间正常关闭。

实际候选TS转译Bootstrap和真实客户端模块执行；基础节点/布局及socket迟到open端口为受控夹具，真实Go认证不冒充微信code exchange。没有本候选原生导出、真机/实际平台SDK或发布验证；linux_native_export的SKIPPED如实保留。未重复本地全Go/race/完整性能测试，候选CI结果与历史证据范围分开。

私有 `E=/root/.config/awesome-zero-platform/acceptance-0054-fcca57d` 保存runs.jsonl（argv/cwd/SHA/UTC/退出码）、ci/provenance/reused-evidence、场景JSON/日志、真实私有帧、normal-live-summary、retained-hand、shutdown、completion-audit及sha256索引。秘密、令牌和手牌不提交。完成后自有API/TLS/MySQL/Redis服务停止，私有卷/证据保留。最终仅本报告和current允许状态段提交main；本次PASS不归档或完成Goal0045/0046。
