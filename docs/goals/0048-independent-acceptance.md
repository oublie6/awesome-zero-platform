# Goal 0048 — G47-N01修复独立验收

日期：2026-11-07。执行者：Codex。独立增量验收，无产品、正式测试、Schema或部署配置修改。

## 1. 结论

**最终FAIL。** G47-N01原始反例及同账号A/B交叉选择均PASS，最新选择不再被旧成员join覆盖；但本轮明确要求的UNKNOWN保护及退出登录边界仍有真实失败，候选的客户端自动化门槛也失败，不能给出整轮修复验收PASS。

本轮记录：**G48-N01/P2工程门槛问题、G48-N02/P2未决请求身份丢失、G48-N03/P3退出登录后迟到写回**。后两项是本轮扩展检查发现的既有相关边界，不能仅凭本次差异断言为该提交新引入。没有发现服务器重复接受贡献、持久化记录损坏或越权获取手牌。

G47-N01的原覆盖反例及本轮同账号最新选择路径已修复；完整验收关闭资格暂不授予。E03及Goal0045/0046仍未完成，继续 `in_progress`。不扩大为产品发布资格。

## 2. 固定候选与读取/差异

- 受测产品：**`5d4a3b6fe9f5e563a25de711bfca565335f5a1cc`**。
- 初始干净main，执行 `git pull --ff-only`，从Goal0047报告 `089bac9aa09fbcfde8ac7de030ff01ad798aec6e` 快进至精确候选。验收期间远端main核对仍相同，没有借用其他SHA的CI结果。
- 已复读AGENTS、原则、current状态及[Goal0047报告](0047-independent-acceptance.md)。current及其历史内容相对上轮未变，历史FAIL原样保留。
- 实际差异只有4文件、95行新增/18行删除：OnlineController成员join增加generation围栏、元数据后最终成员重读；Bootstrap.join共享scene恢复token；online/assets新增回归。全差异及相关RoomCoordinator/shell/session路径已审查。
- server tree仍为 `06209f0fc5a8f9bb8a50682c69cc6f195f354e18`。后端、协议包、GameFairnessActions、GameHandActions、GameClientShell相对G47均未变。

## 3. 精确候选的全部Actions

通过GitHub连接读取三个run、全部jobs及两个失败job日志。所有head SHA均为候选，没有反复轮询或重跑CI。

| 工作流 | 实际结果 |
|---|---|
| [主CI 42387205696](https://github.com/oublie6/awesome-zero-platform/actions/runs/42387205696) | SUCCESS，integration/secure_envelope/unit/admin_web/runtime/report 6/6 |
| [Fair Doudizhu 42387205750](https://github.com/oublie6/awesome-zero-platform/actions/runs/42387205750) | **FAILURE**；client_foundation失败，application/mysql_integration/manifest_client成功 |
| [Cocos Linux 42387205707](https://github.com/oublie6/awesome-zero-platform/actions/runs/42387205707) | **FAILURE**；linux_source失败，linux_native_export SKIPPED |

两个失败job与本地同一新增测试失败一致：`G47-N01: late CLOSED recheck prevents stale existing-member join`。详见G48-N01。主CI成功不能抵消这两个失败，也不能将上轮CI SUCCESS记到本候选。

## 4. 环境及证据边界

复用现有Linux、Node22.23.4、真实四账号Go认证、MySQL5.7.44/Redis8.8.0自有隔离库、loopback HTTPS/WSS、真实Quicknet。稳定保护根、既有工具及测试账号沿用，无下载/安装、无Wine、无生产部署。

API二进制复用G47/此前实际编译产物，server tree相同并核对二进制摘要/provenance；**不声称本轮重新编译后端**。客户端由本候选test构建，Bootstrap按当前TS转译；Cocos基础节点/布局替代，实际菜单和处理方法执行，业务controller、认证、HTTP/WSS与数据库真实。不是手机物理点击或当前原生包资格。

私有证据：`E=/root/.config/awesome-zero-platform/acceptance-0048-5d4a3b6`。`runs.jsonl`记录命令、cwd、SHA、UTC和退出码；`ci.json`、CI失败日志、各场景JSON、`durable-receipt-audit.json`、`reused-evidence.json`、`provenance.json`、`completion-audit.json`及`sha256.json`索引证据。秘密、原材料、账号、token及夹具不提交。

## 5. 原始反例及交叉入口

原反例首先在真实服务复跑，`original-g47` exit0：A=`room-cmd-857544ff9c419cef48c690be374842bb`，B=`room-cmd-bcfbb404005d0d6d604c9ef6d0aee073`。扣留合法A成员HTTP结果，完成B全部恢复/重读后再释放A；旧操作返回selection superseded，Controller/RoomCoordinator/Scene仍B/v1，Fairness实例未变、无错误hand绑定、0个房间变更命令，没有A成功提示。

| 本轮实际交错 | 结果与依据 |
|---|---|
| joinRoom(A) → reconnectAndResync(B) → 旧A | PASS，旧请求拒绝，controller保持B |
| joinRoom(A) → resumeAuthorizedRoom(B) → 旧A | PASS，scene/controller保持B |
| joinRoom(A) → joinRoom(B) → 旧A | PASS，最终B，0个变更命令 |
| 实际Bootstrap.joinRoom(A) → scene恢复B → 旧A | PASS，scene/controller均B，不输出旧加入失败或成功覆盖新提示 |
| 相同房间重复join | PASS，较旧操作失去选择权，最新采用；0个room.join、无额外socket |
| 同房间join与恢复 | PASS，同一目标不重复变更；旧join拒绝，最新恢复成功 |
| 原R19同目标重复恢复 | PASS，两次共享Promise，初始连接加一次重连共2条socket |
| 加入最终读在途时Scene销毁 | PASS，旧结果不采用，online=null、controller room=null |
| 加入最终读在途时退出登录 | **FAIL，G48-N03**：auth=false后RoomCoordinator仍被写回A，再抛client disposed |
| B已HAND_ACTIVE，旧A迟到 | PASS，实际冷发现选择唯一active B，scene/controller/authoritativeHand/confirmedFairnessHand均B对应新hand，Fairness实例不被旧A替换、无legacy secret、0个room命令 |

最后一项实际新hand为 `g48-2e4409e9ee57ab01fd58`，A=`room-cmd-ab3dc0b281492dfa691ccea0024b57a2`，B=`room-cmd-438f412bff3f999f98db5869d16d9131`。不是只在等待房间验证空HandID。

## 6. 最终授权与持久化命令

| 场景 | 实际判断 |
|---|---|
| 初始WAITING/v1 → CLOSED/v2/nonmember | PASS，真实owner离房关闭；join最终重读拒绝，未采用旧房间 |
| member=true → false但房间仍开放 | PASS，非owner真实退出，最终查询仍开放且非成员；拒绝采用 |
| 查询间版本更新 | PASS，真实ready命令推进版本，采用最终新版本，不回退v1 |
| activeHandId变化 | PASS，READY/v6初始读扣留，真实开局至HAND_ACTIVE/v7；旧join最终重读拒绝，不采用旧元数据 |
| lastHandId变化 | PASS，真实牌局完成后v7→v8、activeHandId由H变空且lastHandId由空变H；扣留的旧join拒绝，未绑定旧hand/fairness |
| 最终授权查询失败/超时 | PASS（受控故障层）：最终客户端查询边界分别注入HTTP503及timeout错误，无room采用；timeout用异步延迟后抛超时，**不是声称实际等待了生产15秒网络deadline** |
| 已确认join，旧UI选择失效 | PASS，真实room.join receipt已接受，扣留调用返回再恢复B；释放后旧UI拒绝，B不被覆盖，原accepted receipt仍可精确查询，只有1次房间变更发送 |
| UNKNOWN加入后恢复另一房间 | **FAIL，G48-N02**：原ID被清除、needsReconciliation=false、界面提示恢复成功 |
| 明确失败恢复不得伪造成功 | 关闭/非成员/查询错误通过；UNKNOWN跨房间情况因丢身份并报成功仍FAIL |

lastHandId检查在真实新局叫牌后完成59条合法Bid/Play/Pass，finalVersion60，再读取真实终局引用变化；不是伪造HTTP字段。仅为验证本次授权引用转换执行这一条必要链路，没有重做整套drand/加密/证明验收。active和last字段同时转换的实际结果如上，不声称隔离修改单个后端字段。

`durable-receipt-audit.json`用只读MySQL查询独立确认已ACK及UNKNOWN两条room.join的accepted=true。客户端丢失pending不等于服务器丢失receipt。

## 7. 问题清单及最小修复

### G48-N01 / P2（工程门槛）— 新增CLOSED测试夹具违反现有隐私契约

位置：`clients/fair-doudizhu-cocos/test/online.test.mjs:682`起的late CLOSED测试。`closed={...waiting,member:false,...}`保留了waiting.seats。非成员预览禁止成员私有字段，parseRoomRead正确提前拒绝 `room preview leaked private membership fields`，所以断言期待的membership revoked/room closed分支根本未执行。

复现：在本候选执行 `npm --prefix clients/fair-doudizhu-cocos test`，**74项中73 PASS/1 FAIL**；精确候选Fair和Cocos job日志同样失败。不是环境、Go后端或测试标准需要放宽；真实Go CLOSED响应的独立交错已通过。

最小建议：开发方让非成员fixture符合真实Go DTO，移除成员专有字段，保留现有privacy校验和最终授权负向断言，再提交新候选并恢复CI。验收方未改正式测试，也不将这项当作越权产品漏洞。

### G48-N02 / P2 — 恢复无关房间清除UNKNOWN命令身份

位置：`clients/packages/doudizhu-client/src/room.ts:52–59`的restoreTrustedRoom，只防inFlight，却无条件清uncertainOutcome/pendingCommand；OnlineController.performRecovery采用B及Bootstrap恢复会调用它。此实现相对G47未变，是本轮必查保护项发现的既有缺陷，不断言本提交引入。

真实步骤：账号0以非owner加入另一个真实账号创建的A；Go真实接受后夹具仅丢客户端ACK，RoomCoordinator保留原request、needsReconciliation=true。随后实际scene恢复本人的B，B所有授权查询成功；它清掉A pending，未精确对账A，却显示“已通过服务器恢复本人房间和牌局”。

证据：`cross-completed.json`的UNKNOWN-join-cross-room-recovery。原RequestID **`cmd-bcb87a2429b853dae3dacf2321ee7892`**；beforePending同ID、afterPending=null、needsReconciliation=false，B=`room-cmd-6cb77710ef609b3512d3e49323d4e0ab`/v1，recoveryError=null、room命令发送1次。MySQL同ID receipt仍accepted=true，未发生服务器状态损坏或额外房间提交。

预期：未决请求必须保留原身份，恢复B不得替代A命令结论。最小建议：不同聚合恢复前阻止切换或先按原ID完成对账；普通成员读不能无条件消除另一个room的UNKNOWN。保留原requestId/payload及accepted/rejected/unknown语义，不用清pending或改为本地持久化规避。

### G48-N03 / P3 — 退出登录后最终查询仍部分采用旧房间

位置：OnlineController.joinRoom的current（157–163行）未检查会话认证代数；logout（346–349行）不推进恢复/成员代数，也不将controller标为disposed。最终读已在登出前获得合法结果，只延迟交付，登出后current仍成立，194行先restore room，195行shell才因disposed抛错。

真实步骤：同账号成员join A，扣留第二次真实room查询；执行controller.logout至session.authenticated=false，再释放已授权旧结果。观察RoomCoordinator变为 **`room-cmd-1e9790b7e95ebe2d9d7ddee11228dcb2`/v1**，方法返回Error: client disposed。预期是已登出对象不再采用旧房间；实际为部分写入后报错。

证据：`cross-initial.json`的join-logout-final-read。没有成功提示、越权服务端请求或私牌泄漏证据，按P3生命周期一致性记录。测试identity.revoke为既有noop，此项仅验证本地登出生命周期；HTTP响应在登出前已由真实服务器授权，不以它宣称本轮验证了服务器token撤销。

最小建议：登出开始即撤销恢复操作所有权，并在await后按认证/会话代数阻止采用；确保清理与未决持久化身份保护兼容，不通过最后shell抛错充当采用前围栏。此缺陷同样不能仅凭差异认定为新引入。

## 8. 历史回归、新执行与复用

- R18-N01：实际空room/hand的新controller仅认证冷发现active B并正常采用，通过；G47八进程正常阶段、重启/持久化依据按未变入口/后端复用。
- R18-N02：`reg-mode-isolation`重新执行真实H1→H2、未知模式菜单、旧四操作拒绝、旧callback与合法新贡献，通过。
- R19-N01：`reg-stale-selection`及`reg-duplicate`真实原反例均通过。
- R19-N02：`reg-closed-during-recovery`及`reg-closed-after-adoption`原反例/已采用后撤销均通过。
- 客户端shell本轮74项：73通过/1失败；静态typecheck exit0。协议包未变，未重复本地43项，候选CI客户端阶段及历史证据单独记录，不把整个失败job写成通过。
- 原始G47反例、15个cross场景、4个R19场景、模式隔离、3个active/终局场景均有私有日志与结果；cross最终按13 PASS/2 FAIL记账，早期环境中断项由补跑替代，不把连接失败计产品缺陷或通过。
- `reused-evidence.json`核对G47八phase日志、storage审计、本人字符串及最终BLS/deck/settlement验证的原manifest摘要，及未变server/协议/加密/action/shell源码。沿用G47衔接的R18原子性、HPKE/AEAD、重放/权限、服务端各阶段故障证据。没有重新执行全部高成本测试、下载工具或跑无关压力/原生构建。

夹具记录：首个cross脚本累计owner连接触及现有每账号4条上限，使4项及清理未完成；修为每场景清理自有房间并释放临时连接，只补跑4项，不改任何业务负向断言。清理工具最初误读me/rooms外层rooms字段、随后在owner离房顺序上被DDZ_FORBIDDEN拒绝；改为读取真实席位、其他成员先离开，最终正常清理。原日志均保留，既不归为产品缺陷，也不隐藏FAIL。真实logout和UNKNOWN反例分别只确认一次，没有反复尝试获取想要的结果。

## 9. 限制、状态与交接

本轮没有当前原生导出、真机/SDK登出或键盘UX、正式域名/发布、全规模/长期soak、MySQL引擎断电/主存储丢失证据。最终查询超时为明确受控注入，Scene几何为替代。按任务增量范围记录这些限制，不等待不必要的外部条件，也不以替代层级虚报通过。

H05/GOAL8仍未完成、C09/C10继续DEFERRED，Goal0045/0046保持 `in_progress`。本轮不改历史Goal0047 FAIL，也不修改Goal要求或承诺发布。

验收报告自检已核对源码/候选/CI归属、实际交错、已ACK与UNKNOWN区分、失败等级、源码及摘要复用前提、夹具错误、秘密排除和最终差异。私有API/TLS及自有DB服务已停止，卷和证据保留，其他部署未改。只提交本报告及current允许状态到main，一次提交并push。

交回ChatGPT的最小集合：①修正不合法CLOSED fixture并恢复候选CI；②阻止无关房间恢复清掉UNKNOWN原请求；③登出在采用前撤销在途恢复所有权。然后复验三项失败及相关正常路径，无需重做已通过的公平协议、加密托管或整套持久化实现。

**最终FAIL：原始G47覆盖反例PASS不等于本轮所有必需保护与自动化门槛PASS。**
