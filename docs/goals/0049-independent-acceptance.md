# Goal 0049 — 恢复一致性修复独立验收

日期：2026-11-07；Codex独立增量验收。未修改产品、正式测试、Schema、部署配置或历史报告。

## 1. 结论

**最终FAIL。** G48-N01测试门槛、G48-N02原跨房间反例、G48-N03登出反例均通过；但本轮必查的同房间UNKNOWN交错仍丢失请求身份，并发现本次采用顺序调整造成的房间/公平会话部分采用问题：**G49-N01/P2、G49-N02/P2**。

上轮三个原始反例的修复得到独立确认，不重复计为三个旧缺陷；完整UNKNOWN及恢复一致性资格仍未通过。Goal0045/0046继续 `in_progress`，E03未完成；本报告交付不表示产品或发布资格完成。

## 2. 固定候选与完整差异

受测产品 **`42f3e0c296a50445465660f2c7d577ba521e696a`**。初始干净main，经 `git pull --ff-only` 从上轮报告 `ff075b7c17db60660b4fb35ab8a7fae4fd4320c4` 快进至候选；验收期间远端main再核对仍相同，没有混用其他SHA。

已读取AGENTS、相关原则、current及[Goal0047](0047-independent-acceptance.md)、[Goal0048](0048-independent-acceptance.md)；current和历史报告相对ff075b7未变，历史FAIL保留。

全部差异为5文件、99行新增/7行删除，已逐项审查：

- RoomCoordinator.restoreTrustedRoom新增跨聚合UNKNOWN保护，但同聚合仍会清pending。
- OnlineController恢复/加入入口拒绝无关房间切换，await围栏加入认证检查；logout先设置disposed并推进恢复/成员代数。
- performRecovery及成员join将restoreTrustedRoom移到公平会话采用之前；这项顺序变更触发G49-N02。
- CLOSED夹具改为符合非成员DTO；新增RoomCoordinator/online/assets回归，没有修改parseRoomRead或跳过原测试。

server tree仍为 `06209f0fc5a8f9bb8a50682c69cc6f195f354e18`。Bootstrap、GameFairnessActions、GameHandActions、GameClientShell及后端相对G48未变；没有API、数据库或部署变更。共享协议包仅上述room协调逻辑及其测试变化。

## 3. 精确SHA的Actions与客户端门槛

已通过GitHub连接读取三个run及全部jobs，全部head SHA与候选相同，没有重跑CI或反复轮询。

| 工作流 | 最终结果 |
|---|---|
| [主CI 42392836263](https://github.com/oublie6/awesome-zero-platform/actions/runs/42392836263) | SUCCESS，unit/secure_envelope/admin_web/integration/runtime/report 6/6 |
| [Fair Doudizhu 42392836287](https://github.com/oublie6/awesome-zero-platform/actions/runs/42392836287) | SUCCESS，mysql_integration/manifest_client/application/client_foundation 4/4 |
| [Cocos Linux 42392836332](https://github.com/oublie6/awesome-zero-platform/actions/runs/42392836332) | 总体SUCCESS，linux_source SUCCESS，**linux_native_export SKIPPED** |

本轮独立执行协议包44/44、shell77/77及静态typecheck均通过。新增CLOSED fixture不再带seats，原parseRoomRead隐私校验及真实CLOSED最终拒绝仍通过。**G48-N01 PASS/关闭**；CI成功不抵消后文真实业务失败，也不是当前原生导出PASS。

## 4. 环境和执行方法

复用Linux、Node22.23.4、已有四个真实Go认证账号、MySQL5.7.44/Redis8.8.0自有隔离库、稳定测试保护根与loopback HTTPS/WSS。API二进制沿用此前编译产物，核对相同server tree及原二进制摘要；**不声称本轮重新编译后端**。候选客户端由本轮test实际构建。

Bootstrap按当前TS转译，基础Cocos节点/布局替代；实际controller、协调器、菜单处理、客户端加密、Go规则/认证、HTTP/WSS及MySQL真实。延迟的是已经由真实服务器生成的HTTP结果或已接受命令的客户端ACK，不伪造授权状态。登出测试的identity.revoke为既有noop或受控等待，仅确认客户端会话隔离，不宣称重验微信SDK或服务器token撤销。

私有证据 **`E=/root/.config/awesome-zero-platform/acceptance-0049-42f3e0c`**：runs.jsonl记录argv/cwd/SHA/UTC/退出码；ci.json、unknown.json、logout.json、edges.json、same-room.json、fairness-atomic.json、各历史反例JSON、negative-receipt-audit.json、reused-evidence.json、provenance.json、completion-audit.json和sha256.json索引证据。账号、token、贡献和夹具不提交。顺序执行，无下载/安装、无全量压力、无原生构建或完整故障矩阵。

## 5. G48-N02原始反例与五类命令精确对账

原始join首先验证：真实服务接受room.join后，夹具丢ACK，保留原请求；尝试scene恢复B及显式join B均拒绝“unknown for another room”，无B成功提示、无新room命令。原RequestID/payload/sequence/expectedVersion逐字段不变，needsReconciliation保持true，MySQL整条原receipt摘要不变。

随后真实断开/重新认证WSS，调用产品RoomCoordinator.retryUnknown，实际wire请求与原请求深比较相同；得到原accepted receipt，MySQL原行仍不变。对账后正常scene恢复B成功。此流程对五类命令都执行：

| UNKNOWN来源 | 原RequestID | 跨房间保护、精确重试、后续B恢复 |
|---|---|---|
| room.join | cmd-93b9c66f0eaad7dc80fafb72f4df8ea4 | PASS |
| room.create | cmd-65807fad0fec0d66e9d307be5d76220e | PASS |
| room.ready | cmd-5a1730ef13618d45da178bba11773b4b | PASS |
| room.hand.start | cmd-289a900362100d6a8ee935b0676a409b | PASS |
| room.leave | cmd-d0ac69570e902291e60b08eb25ec750a | PASS |

每项都实际接受后丢ACK，不以“发送前直接throw”的单测代替。MySQL只读oracle把binary digest转HEX后，对包含actor、command、aggregate、sequence、payload digest、command/result JSON、accepted和时间的整行做摘要，重试前后相同；不是只核对行数。

额外通过：重复ready的真实明确拒绝不进入UNKNOWN、pending清空且可恢复B；UNKNOWN leave的原房间因已非成员而拒绝恢复，但原ID保留且可精确重试；全新认证实例无旧pending/locator即可发现并恢复服务器已ready/v2的房间，原旧实例pending不受影响。

**G48-N02原跨房间反例PASS；完整UNKNOWN关闭资格FAIL，原因是同房间在途旧读，见G49-N01。** 不把五类跨房间通过扩大为它们所有同房间时序组合都通过。

## 6. G48-N03登出与多账号生命周期

以下六个真实HTTP交错全部PASS，logout.json记录：

1. 原始最终join授权结果已完成但被扣留，logout后释放：selection superseded，room/hand不采用。
2. 第一次成员结果迟到时logout：拒绝旧操作，未写回房间。
3. reconnectAndResync最终读在途时logout：返回null，未恢复旧用户状态。
4. 同一controller的join A与恢复B两项同时在途，logout后释放：两者均失去采用权，没有后续私牌同步。
5. logout在等待revoke期间与实际Scene.onDestroy交错：认证已失效，旧响应不写回，scene.online=null。
6. logout后以另一个真实认证账号的新controller替换Scene；释放旧账号结果：新账号发现列表/room/hand不受影响。

对旧对象核对rooms、confirmedFairnessHand、shell.authoritativeHand、HandStore、Fairness secret及认证状态，不以最后抛错作为无部分写入的证据。新旧账号隔离在认证controller/Scene层验证，不冒称实际wx账号切换SDK。

另执行UNKNOWN ready存在时logout：原请求完整保留、needsReconciliation=true，旧恢复HTTP晚到不改原room状态；新账号无继承pending或房间。**G48-N03 PASS/关闭**，不存在上轮“先写A再抛client disposed”的原反例。

## 7. 新问题与最小修复

### G49-N01 / P2 — 同房间旧恢复读清除后来出现的UNKNOWN

位置：OnlineController.performRecovery（约120–128行）仅比较两次读之间的版本，未针对读期间发生的room命令重新确认；RoomCoordinator.restoreTrustedRoom（52–63行）只拒绝另一个aggregate的UNKNOWN，同房间仍无条件清uncertainOutcome/pendingCommand。

最小真实步骤：

1. A为本人WAITING/v1。发起同房间恢复，并重复点击确认共享同一Promise；初读和最终读均在真实服务器读到v1，扣留最终读结果。
2. 此时用同一controller发送room.ready(true)，Go实际提交至v2；仅丢其ACK，使请求进入UNKNOWN并保留原ID、seq2、expectedVersion1及payload。
3. 释放较早的真实v1结果。恢复返回v1，清pending、needsReconciliation=false；当前服务器仍v2/ready=true/lastSequence2。产品retryUnknown随后报“no exact pending room request to retry”。

证据：same-room.json及same-room-inflight-unknown.log exit1。A=`room-cmd-a873953e65eaa6186e62567e4d3fdad9`，RequestID=`cmd-ea37943840dd39e0cabee59f9fbf703d`。客户端room版本1，服务端版本2；命令计数2是原create和原ready，无恢复产生的额外命令。原MySQL行存在且摘要不变，没有服务器回滚或存储损坏。

这不是争论“新进程能否仅凭状态恢复”：同一实例已有明确的未决原请求，且采用的读真实早于该命令，不能证明它已对账。服务器已接受的操作因此失去客户端精确重试身份。此缺口不由跨房间guard修复，未据此重复运行其他同根因失败组合。

最小建议：恢复读在采用前必须与当前room命令生命周期/水位重新核对；精确命令结论未取得前保留原request，不因相同roomId就解除UNKNOWN。可以拒绝/重新对账旧恢复，或只更新展示而保留未决命令；不得以清pending或新ID重发解决。保留无旧pending的新账号冷恢复能力。

### G49-N02 / P2 — 公平会话拒绝切换前已采用另一个房间

位置：本次改动的OnlineController.performRecovery:126–128及成员join:201–203，先restoreTrustedRoom，再confirmActiveFairnessHand。后者可能因fairness.needsReconciliation拒绝，但前者已修改RoomCoordinator，导致部分采用。真实复现使用scene恢复路径；不声称另外重跑了join的同根因反例。

步骤：三人真实开新托管H，owner提交一个真实随机贡献；Go accepted后丢ACK，Fairness保持UNKNOWN及原加密请求。另建本人B，实际scene恢复B。方法正确报“fairness result unknown; retry the original request before changing hand”，但RoomCoordinator已变B，Scene和Hand仍A/H。

本次A=`room-cmd-7c164baa4fe384b3ffb28e1f33fb8789`，B=`room-cmd-d63f422d7a6a3e26990af1cf58a6363a`，H=`g49-fair-de09ae498d505a9c`，公平RequestID=`cmd-a30997e0fdd2d24dd4f9cf741c9d3ab5`。fairness-atomic.json及日志exit1记录scene=A、controller=B、confirmedFairnessHand/authoritativeHand=H，公平pending完全保留。accepted receipt已由MySQL只读审计另行确认存在。

影响是客户端房间/hand目标不一致；没有成功提示、秘密丢失、跨账号泄漏或错误后续命令已执行的证据，按P2，不升级P1。对比本次差异可定位到采用顺序调整，不是服务端公平算法/配置错误。

最小建议：先以不修改状态的方式验证room UNKNOWN及fairness/hand切换的全部前提，随后一致采用room/fairness/hand。保留“无关房间不能解除room UNKNOWN”的新保护；不要仅交换两个会修改状态的方法、清公平pending或抛错后留下部分采用。

## 8. 历史回归、执行与证据复用

| 对象 | 本轮依据及结论 |
|---|---|
| R18-N01账号发现 | 新认证实例的正常冷发现/ready状态恢复实际PASS；未增加游戏凭证持久化 |
| R18-N02模式隔离 | 实跑mode-isolation：旧H1迟到callback、未知模式菜单、四旧方法拒绝与合法新贡献PASS |
| R19-N01选择/重复恢复 | 原stale-selection及duplicate实跑PASS |
| R19-N02授权撤销 | 原closed-during-recovery、closed-after-adoption实跑PASS |
| G47-N01旧成员join | 原stale-member-join实跑PASS，不覆盖新房间 |
| G48-N01 | CLOSED夹具/隐私校验/77个shell与精确CI PASS |
| G48-N02 | 原跨房间及五类精确重试PASS；同房间扩展FAIL，G49-N01 |
| G48-N03 | 六个登出场景及UNKNOWN/新账号扩展PASS |
| 正常恢复与对账后继续 | 对账后B恢复、明确拒绝释放、无旧pending冷恢复PASS；公平UNKNOWN切换的原子采用FAIL，G49-N02 |

新执行：`npm --prefix clients/packages/doudizhu-client test` 44/44；`npm --prefix clients/fair-doudizhu-cocos test` 77/77；shell typecheck；五类UNKNOWN真实命令/WSS断开及精确重试；六个logout；四个edges；五个R19/G47原反例及一次模式隔离；两个新负例。不同证据层级不合成虚假覆盖率。

复用：G48真实activeHandId/lastHandId转换及其59操作终局；G47八进程正常恢复/重启、原随机贡献记录、本人核验及完整BLS/洗牌/结算证明，再衔接既有R18原子性/权限证据。reused-evidence.json核对原manifest摘要和未变server/crypto/action/shell/Bootstrap实现。当前发生变化的pending/生命周期未直接继承旧PASS，而是重新实际检查。

初次只读oracle直接以文本读取BINARY payload_digest而解码失败，五项在oracle处未完成，日志及JSON保留；修成HEX后只补齐这些未完成检查，原命令字段、持久化行摘要、精确重试负向要求不变。它是私有取证工具错误，不计产品缺陷。两个真实产品负例各确认一次，未为获得PASS重复尝试。

## 9. 风险与状态

未执行本候选原生导出、真机/平台SDK登出、完整游戏/证明全套、规模/长期soak、服务端故障全矩阵或存储断电测试；按任务只复用其未变有效历史证据。不等待设备/发布条件，不开展C09/C10延期根迁移/退役。

没有改变随机贡献或服务器唯一权威模型，没有新增客户端牌局备份。两项新发现均是客户端恢复采用与未决请求的一致性问题。H05/GOAL8未完成、C09/C10仍DEFERRED；E03及Goal0045/0046保持 `in_progress`，不封版、不发布。

## 10. 交接

交回ChatGPT的最小修复：①同房间旧读不得解除后来出现的UNKNOWN及丢失原RequestID；②room/fairness/hand切换必须先完整校验、再一致采用，失败不留部分状态。保留本轮已通过的跨房间guard、登出围栏、真实DTO及隐私校验。修复后复验这两个反例和必要正常链路，不需重做服务器加密、洗牌或完整故障验收。

报告自检核对了候选/CI归属、原反例与新问题区分、真实异步时序、服务端accepted记录、证据复用摘要、秘密排除和最终差异。私有API/TLS/自有DB服务已停止，卷和证据保留，其他部署未改。仅本报告和current允许状态一次提交到main并push；历史报告原样保留。

**最终FAIL。上轮三个原始修复点通过，不能替代本轮全部必需恢复一致性检查。**
