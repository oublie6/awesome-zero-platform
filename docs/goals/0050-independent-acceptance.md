# Goal 0050 — 恢复一致性与原子切换独立验收

执行时间：2026-11-07至2026-11-08 UTC。执行者：Codex。独立增量验收，无产品、正式测试、Schema、部署配置或历史报告修改。

## 1. 结论

**最终FAIL。** G49-N01、G49-N02原始真实反例均PASS，原请求保留并能精确对账，公平UNKNOWN拒绝切换时不再部分采用。扩展检查仍发现两类P2：**G50-N01，已确认版本可被旧加入/恢复结果覆盖；G50-N02，Scene缺少房间UNKNOWN精确对账恢复入口**。

124项客户端测试及三套候选CI成功，不代表本轮业务门槛全部通过。原G49两个反例可按本轮证据关闭；完整命令代次与正常恢复资格未通过。Goal0045/0046继续 `in_progress`，E03未完成，不封版、不发布。

## 2. 固定SHA与完整差异

受测产品：**`99f5a0ede6ef94929c91b99a6ae1582073e880b8`**。干净main执行 `git pull --ff-only`，由上轮报告 `7efed3d9f904716b2141e970e1ee5b3d4a4a2389` 快进至候选；报告前远端main再次核对相同。

已读取AGENTS、原则、current及[Goal0048](0048-independent-acceptance.md)、[Goal0049](0049-independent-acceptance.md)。current和历史报告相对上轮未变，原FAIL保留。

本次5文件、116行新增/20行删除，完整差异已检查：RoomCoordinator增加commandGeneration和所有UNKNOWN的采用前检查；OnlineController的reconnect捕获命令代次，新增adoptAuthorizedRoom并由成员join复用；三份测试增加/调整相应断言。没有GameFairnessActions、GameHandActions、GameClientShell或Bootstrap实现变更。

server tree仍为 `06209f0fc5a8f9bb8a50682c69cc6f195f354e18`，API、数据库、随机贡献、加密、规则和恢复实现未变。本轮不进行无关架构审查或重构。

## 3. 精确候选的全部CI与本地门槛

已独立读取三个run及全部jobs，head SHA均为固定候选，无重跑或重复轮询。

| 工作流 | 结果 |
|---|---|
| [主CI 42398602820](https://github.com/oublie6/awesome-zero-platform/actions/runs/42398602820) | SUCCESS，unit/integration/secure_envelope/admin_web/runtime/report 6/6 |
| [Fair Doudizhu 42398602859](https://github.com/oublie6/awesome-zero-platform/actions/runs/42398602859) | SUCCESS，mysql_integration/client_foundation/manifest_client/application 4/4 |
| [Cocos Linux 42398602842](https://github.com/oublie6/awesome-zero-platform/actions/runs/42398602842) | 总体SUCCESS；linux_source SUCCESS；**linux_native_export SKIPPED** |

本轮独立执行：协议45/45、shell79/79、静态typecheck全部通过。未把SKIPPED记为真实原生导出成功。

## 4. 环境与证据

复用Linux、Node22.23.4、现有四个真实Go认证账号、MySQL5.7.44/Redis8.8.0自有隔离库、稳定保护根、HTTPS/WSS及真实Quicknet。沿用相同server tree的原编译API，并核对二进制摘要，不声称重新编译后端。仅延长复制夹具的reveal测试registry有效期，未替换保护根或修改生产配置。

候选客户端经test构建，Bootstrap按当前TS转译；Cocos基础节点/布局替代，实际菜单/处理方法、controller、HTTP/WSS、加密和Go/MySQL真实。登出验证在认证controller层，identity.revoke采用既有noop或受控等待，不冒称微信SDK换号、服务器token撤销或真机证据。

私有证据：`E=/root/.config/awesome-zero-platform/acceptance-0050-99f5a0e`。runs.jsonl包含完整argv/cwd/SHA/UTC/退出码；ci.json、各场景JSON/日志、version-failure-receipts.json、reused-evidence.json、retained-hand.json、provenance.json、completion-audit.json及sha256.json索引结果。账号、密钥、贡献、手牌和夹具不提交。

## 5. 两个原始反例

**G49-N01 PASS。** 真实A/v1恢复的最终HTTP结果被扣留，随后真实ready接受至v2但丢ACK。释放旧读后恢复返回null，原RequestID/payload/clientSeq/expectedVersion完全保留，needsReconciliation=true，无过期成功状态或替代命令。产品retryUnknown发送原wire请求，MySQL原整行摘要不变；随后正常恢复至v2。证据：same-room.json、original-N01.log。A=`room-cmd-6068f57c7e5daf9cd6bbd828df5fc46e`，原RequestID=`cmd-1a1283031ebf1ed48d6dbc96e7f17bef`，命令代次由原ready至精确重试递增，最终为3。

**G49-N02 PASS。** 三人真实托管局H中，贡献已接受后丢ACK，Fairness保留原加密请求。恢复另一本人B被拒绝，Room/Scene仍A，authoritativeHand/confirmedFairnessHand仍H，原Fairness实例和pending不变。原公平命令精确重试后，原持久化行摘要不变，恢复B正常成功，未留下永久reservation。证据：fairness-atomic.json、original-N02.log。A=`room-cmd-03808c8382de5b3eb7642578ad33d710`，B=`room-cmd-2af84228240b1b5ec367f3b0a7228282`，H=`g49-fair-def0692c8c018793`，RequestID=`cmd-4f8f87e91587f2537886cffeab668259`。

这两项都在真实服务接受/持久化后丢ACK，不用发送前throw的单元模拟替代。

## 6. 命令代次与采用前置检查

| 检查 | 实际判断 |
|---|---|
| 命令开始即推进代次 | PASS：源码在inFlight后、请求ID异步获取前递增；真实延迟CSPRNG结果期间已观察到代次增加 |
| 恢复先读、随后命令ACK迟到 | PASS：实际ready已提交但扣留ACK，旧恢复返回null；释放ACK后为v2 |
| 命令明确拒绝 | PASS：真实重复ready拒绝推进代次，UNKNOWN不滞留，旧恢复不采用 |
| 精确重试期间旧恢复 | PASS：原UNKNOWN请求重试推进代次，旧读返回null，原wire一致 |
| 两次同房间恢复与命令交错 | PASS（原反例）：共享同一恢复Promise，命令使旧结果失效，之后可重新恢复 |
| 普通member join的命令围栏 | **FAIL，G50-N01(a)**：没有捕获commandGeneration，旧结果覆盖已确认v2 |
| 命令先开始、恢复后开始 | **FAIL，G50-N01(b)**：两者捕获同一代次，命令完成后仍采用提交前的v1 |
| 失败/重连/重复导致代次复位 | 未见复位：字段只在构造时为0，perform/retry单调递增；本轮重连和精确重试通过 |

adoptAuthorizedRoom先检查room未决、startingHand，再预留Hand与Fairness，全部前提通过后才修改room/fairness/hand，并在finally释放。分析了实际可能抛错的同步调用：Room采用的ID/version/sequence先由API解析及restore验证；Hand/Fairness inFlight、UNKNOWN和transition异常在修改前抛出；正常公平实例构造使用未变合法配置，不调用异步熵源；合法公开命令路径的shell pending由room/Hand在途状态覆盖，生命周期和选择权在await后检查。不是仅凭“没有await”判原子性。

真实拒绝及后续重试验证了reservation释放和状态不变。但这些前置校验**没有证明读结果比当前已确认房间更新**，也没有替Scene完成精确对账；故不能把原子采用的已测PASS扩大为整体恢复PASS。

## 7. 九项状态矩阵与正常回归

| Room | Hand | Fairness | 结果及实际依据 |
|---|---|---|---|
| 无未决 | 无未决 | 无未决 | PASS：正常waiting/active冷恢复、成员加入、版本更新及切换 |
| UNKNOWN | 无未决 | 无未决 | PASS（身份保护）：五类房间命令的同/跨房间拒绝、精确重试；Scene可用性另FAIL |
| 无未决 | UNKNOWN | 无未决 | PASS：实际Bid接受但丢ACK，保留原Hand请求；对账后正常恢复 |
| 无未决 | 无未决 | UNKNOWN | PASS：G49-N02原反例及精确公平对账后恢复B |
| UNKNOWN | UNKNOWN | 无未决 | PASS：实际Hand UNKNOWN期间room.ready被服务器拒绝但响应丢失，客户端两请求均UNKNOWN；阻止切换并分别精确对账 |
| 无未决 | UNKNOWN | UNKNOWN | PASS：贡献ACK丢失后仍合法发牌，实际Bid ACK再丢失，两请求均保留，未部分采用 |
| 执行中 | 无未决 | 无未决 | **FAIL（版本围栏）**：命令先于恢复开始，提交并ACK后旧读仍采用v1，见G50-N01(b) |
| 无未决 | 执行中 | 无未决 | PASS：真实Play接受后扣留返回，同房间恢复被阻止；释放ACK后正常恢复 |
| 无未决 | 无未决 | 执行中 | PASS：真实贡献接受后扣留返回，恢复B被阻止，不抢占Fairness |

组合均通过公开产品方法与真实请求产生，没有手工塞入非法pending。双Room/Hand UNKNOWN的room命令最终是**拒绝receipt**，客户端在响应丢失期间仍是真实UNKNOWN，不能混写为两条都accepted。另验证Hand执行中同时Fairness UNKNOWN的更强组合。

Hand矩阵只使用一个真实新局，接受一次Bid(3)及一次合法Play；没有重打完整对局或重验洗牌证明。live原请求实际wire精确比较、MySQL整条live action摘要重试前后相同；公平请求先精确对账，再完成Hand对账及A/B恢复。实际新认证实例从空room/hand自动发现并恢复active H，测试oracle只用于选择当前合法操作者及结果断言，不向恢复入口传旧牌局数据。

五类room.create/join/ready/hand.start/leave均实测接受后丢ACK，同房间及跨房间恢复不消除pending，原请求精确重试后允许正常恢复B；leave后已非成员的原房间仍拒绝恢复，但请求可精确结算。结合本次变更，在同一条新命令链中追加同房间检查，没有另跑整套高成本跨房间验收。

R18-N01 waiting/active账号冷恢复、R18-N02真实模式隔离、R19选择/重复/CLOSED/nonmember、G47旧join覆盖原反例均实跑PASS。G48-N03六个登出/销毁/换账号交错复跑PASS。没有新增客户端权威备份。

## 8. 新发现，交回ChatGPT

### G50-N01 / P2 — 命令代次未覆盖全部入口和执行时序

**(a) 普通成员join：** A/v1发起joinRoom(A)，扣留第二次真实成员读；同controller正常ready(true)，Go与RoomCoordinator均确认v2。释放旧v1，join正常返回v1，RoomCoordinator从已确认v2变回v1，服务器仍v2/ready=true。证据：generation.json，A=`room-cmd-a9694e3e8c9d26aa35da260f3fb848ad`。joinRoom:181–184只捕获membership/recovery代数，未捕获命令代数；212行仅比较两次HTTP读而非当前已确认版本。

**(b) 恢复在命令执行中开始：** A/v1发起ready，真实CSPRNG字节已生成但暂缓交付，请求尚未发送，commandGeneration已递增为2。此时开始reconnectAndResync(A)，两次真实HTTP读均为v1，扣留最终读；释放熵源，原命令经真实WSS接受并ACK，RoomCoordinator确认v2。再释放旧读，恢复正常返回v1并覆盖为v1。证据：inflight-version.json，A=`room-cmd-334d886d4f19e827ac22a1fc6c2c7817`，resultVersion=1、clientVersion=1、serverVersion=2、error=null。延迟的是真实熵结果，没有制造假随机、假HTTP或未提交状态。

两种时序对应同一必需不变量失败，合并一项P2，不把相同症状重复计数。普通join缺代次检查；reconnect虽捕获代次，但命令已经开始时捕获的是相同代次，ACK完成又没有使该读失效。采用时inFlight已false、无UNKNOWN，所以新前置检查不能挡住旧版本。

影响为Controller/RoomCoordinator已确认版本被回退，可能导致后续命令使用过期expectedVersion。两个探针直接检查controller方法，**未声称Scene最终一定停在v1或已执行了错误后续业务命令**；Scene的额外刷新可能纠正版本，但不能使原方法的过期采用成为通过。无未决请求被删除；此时ready已明确ACK，MySQL仍是原accepted/v2，不是服务端回滚。version-failure-receipts.json保存对应原RequestID、sequence、accepted receipt。

最小建议：统一join/reconnect采用前的命令生命周期/版本水位检查；覆盖“读先开始”和“命令先开始”两种顺序。执行中开始的恢复必须等待可对账结果或重新读取，不能只比较相同epoch；同一房间不得采用低于当前已确认版本的读。保留G49-N01已通过的UNKNOWN保护，不清pending或新ID替代。

### G50-N02 / P2 — 新UNKNOWN保护没有可用的Scene房间对账入口

位置：RoomCoordinator.assertCanRestoreTrustedRoom:43–46及OnlineController.adoptAuthorizedRoom开始拒绝全部room UNKNOWN；但Bootstrap.reconnect/recoverMyRooms/resumeAuthorizedRoom只走读取/采用，没有调用room.retryUnknown。当前菜单只有Hand/Fairness的显式精确重试入口，room.leave只处理已知房间的特定leave UNKNOWN。

真实步骤：通过实际Bootstrap.createRoom发送建房，Go accepted后丢ACK。Scene尚无roomID，原RequestID仍保留。恢复网络后执行实际Scene重连、手填A加入、退出房间，分别得到：

- 重连失败：room outcome unknown; retry original request before restoring room。
- 加入失败：同上，原pending仍在。
- 退出失败：请先确认当前房间成员身份。

A=`room-cmd-6572e64a4bcccb4c8ad7b148715b81df`，RequestID=`cmd-ec56869f4b070b10b51983c5a5fe9168`。scene-room-retry.json记录Scene仍空、UNKNOWN=true、仅原create一次变更发送。验收工具再手动调用底层rooms.retryUnknown，随后的相同Scene重连立即成功，证明原请求与服务端状态健康，缺的是产品可达的对账流程。

预期是用户能在当前会话通过正常恢复完成原请求对账；实际常规重连/加入/退出路径被保护阻断。不能说数据丢失或底层API永远不可用：精确retry方法可用，重新登录/新实例也可能绕开旧pending，但不是当前Scene提供的正常处理流程。未丢身份、未发替代RequestID；原命令可精确对账。

最小建议：将room UNKNOWN的原RequestID精确对账接入正常重连或明确用户操作，正确区分accepted/rejected/仍UNKNOWN，结算后重新读权威状态再采用。保留当前保护，不能恢复旧的读状态清pending行为来使UI通过。此缺口由本次收紧恢复规则显露，原先Scene读取即可继续的路径不再成立。

## 9. 复用、限制及资源收尾

reused-evidence.json检查G49及G47原manifest摘要，确认server、Bootstrap、Fairness/Hand actions和shell未变。服务器随机贡献、HPKE/AEAD、完整drand/洗牌/牌局/结算证明、持久化与故障恢复沿用其有效范围；本轮只为新前置Hand状态取得真实发牌并执行两条必要操作，不重复完整高成本验收。所有新增场景均绑定候选SHA。

保留一个私有活动测试局：A=`room-cmd-da25a22702d512168144458d344029e0`，H=`g50-hand-680df16e1cfb1f86`，服务端PLAYING、live actions=2，记录在retained-hand.json；它不是产品发布数据，保留以避免为清理而重打整局。测试服务已停止，卷保留。后续同夹具发现列表须考虑该测试局。本轮waiting房间历史回归改用第四个真实测试账号作owner，原参与者/权限/断言不变，避免该活动夹具污染“仅A/B”前提；不是忽略产品返回的额外房间。

未执行真机、当前原生导出、全WSS故障排列、长期soak/规模或C09/C10延期专项。不以替代Cocos节点、内存夹具或源码门槛声称设备/发布通过。同步采用分析限于合法公开调用与列明真实状态组合，未对内存耗尽等进程失效作跨对象事务保证。

## 10. 最终交接

只交回两项最小问题：完整的命令代次/版本采用围栏，以及可达的Scene房间精确对账流程。不要重做已通过的服务器加密、洗牌或B级恢复，也不要取消已通过的UNKNOWN/跨组件保护。

已核对固定SHA、全部差异、CI归属、两项原反例、九项矩阵、命令字段/服务端记录、精确重试及失败后的继续能力；负例只确认一次，未改产品/正式测试。报告自检区分了Controller探针与Scene、真实与替代、私有保留局与生产状态。只提交本报告及current允许状态到main，保留历史FAIL；私有API/TLS/自有DB服务停止，其他部署不改。

**最终FAIL。G49两项原反例PASS；本轮全部必需恢复一致性与可用性门槛尚未通过。Goal0045/0046继续 `in_progress`。**
