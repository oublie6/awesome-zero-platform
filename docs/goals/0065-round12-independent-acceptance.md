# Goal0065 Round12 独立收尾验收

**FAIL：Round11 N01/P3 原复现通过；新发现 1P2、2P3。未修改产品代码，未部署候选版本。** Goal0065 保持 `in_progress`，Goal0045/0046 不变。本报告完成独立验收任务，不表示产品 Goal 完成。

固定候选：main `62acd921ce767e13d419dd8021e4d065a84e52f0`。增量：`5d84d69f1363e521bf0b07066db4d90c93578b53...62acd921ce767e13d419dd8021e4d065a84e52f0`，22个文件。执行日期2026-11-26 UTC。已核对AGENTS、项目原则、当前Goal、Round11报告、通用清理设计及全部增量。所有运行限于项目自有隔离环境，故障SDK/事件接口是受控端口，不是真机资格证明。

## 原 P3 复验及通用组件

**Round11 N01 PASS**。实际Node生产模块及唯一官方Creator编译产物均验证：release、普通清理handler、诊断handler抛异常，不再阻断后续步骤；cancel/finish/abandon各两个5000ms等待器均立即以`operation cancelled`结束，不等各自deadline。重复清理不重复执行，结束后注册handler的异常也隔离。静态诊断不带原始SDK错误，诊断自身抛异常不替代业务结果。

Node还验证runCleanup多步骤失败计数、原业务值/原错误对象保留、嵌套取消、迟到回调及新旧slot隔离。普通异常计数正确；**重入计数仍失败，见本轮N03，不能由原P3通过推断整个组件通过。** 证据：cleanup-core.json（6PASS/1FAIL）、game-cleanup.json（3PASS）、native-cleanup.json（5PASS/4FAIL，四个失败观察对应下面三个问题，其中连接问题占manager/shell两个单元）。

## G65-R12-N01 / P2：连接清理重入后，较早操作覆盖较新连接

位置：`clients/packages/game-client-core/src/index.ts:82`（open在close返回后才捕获generation）；`clients/fair-doudizhu-cocos/assets/scripts/game/GameClientShell.ts:233`（旧close返回后未复核本次所有权便安装manager）。

复现：建立initial连接。调用较早的B.open/connect；B清理initial时，其受控unsubscribe同步重入，启动较新的C.open/connect（标记latest）。清理返回后，较早B继续建立superseded连接。

实际Node与Creator结果一致：连接创建顺序`initial, latest, superseded`；旧initial关闭一次，latest和superseded均未关闭。Manager最终选择superseded，两路消息`latest:event`与`superseded:event`都能送达。Shell同时返回`[false,true]`（B过期、C成功），却选择superseded，且`commandTransportReady=true`。较新C的接收路径与实际选中的发送路径不再一致，可能形成错误就绪及UNKNOWN请求。

原因：仅在close内快照/置空资源不足以保护包围该close的启动操作。Manager较早与较新操作捕获同一代次；Shell虽知道B过期，仍留下B覆盖的新manager。预期：外部清理重入后的旧启动不得覆盖最新操作，过时连接及回调必须失效。

这是生产类的受控重入反例，**未声称正常浏览器SDK已自行触发、实际Go泄露了其他账号数据或真实网络句柄已经泄漏**。P2依据是连接所有权、消息隔离和就绪契约的确定性失败。证据：[cleanup-core.json](evidence/0065-round12/cleanup-core.json)、[shell-reentrant.json](evidence/0065-round12/shell-reentrant.json)、[native-cleanup.json](evidence/0065-round12/native-cleanup.json)及相应驱动。

## G65-R12-N02 / P3：MiniHttp 部分注册失败遗漏可用的注销

位置：`clients/packages/game-client-core/src/minigame.ts:108`。signal.addEventListener在负责complete清理的try之外。

受控signal先保存abort handler再抛原始注册异常，且提供能正常移除handler的removeEventListener。request拒绝原异常、SDK请求0，但注销尝试0、残留handler1；退出失败路径没有进入complete。实际Creator工厂创建的MiniHttp复现相同结果。对照WebHttp在同样故障下注销一次、残留0。

此处不是底层拒绝注销：**注销可用但没有尝试**。P3限于部分注册失败的资源清理缺口；没有证明普通原生AbortSignal会对合法调用抛此异常，也没有真实业务请求或凭证泄漏。证据：transport-faults.json（10PASS/1FAIL）、native-cleanup.json。

## G65-R12-N03 / P3：OperationScope 重入清理会覆盖异常计数

位置：`clients/packages/game-client-core/src/operation.ts:47`及`:51`。

release执行时，scope已结束；release同步调用onCancel注册一个会抛异常的立即清理，再抛release异常；另一既有handler也抛异常。三次诊断均执行，但cleanupFailures最终为2，预期3。Node生产模块和实际Creator包一致。

原因：`this.failures += runCleanup(...)`在调用runCleanup之前读取左值；嵌套onCancel已经增加的计数，被外层最后写回覆盖。资源步骤均已尝试，问题是诊断计数失真，故P3。证据：[reentrant-count.json](evidence/0065-round12/reentrant-count.json)、native-cleanup.json。

## 全部受影响调用链及边界

引用全量定位见run-cleanup-callers.txt；对新增调用和相邻销毁逐一检查，不扩大到无关架构。runCleanup仍是纯TS技术能力，模块和服务端权威边界不变。

| 调用链 | 实际检查及结果 |
| --- | --- |
| OperationScope/Slot/runCleanup | 原P3、重复清理、多个waiter、嵌套与旧owner隔离通过；重入计数N03失败。错误数只计失败清理步骤，诊断自身失败隔离，不额外算一个清理步骤。 |
| ConnectionManager | 注册失败保留原错误，即使off/close抛异常仍完成其余清理；重复close不再关闭旧资源。重入启动N01失败。 |
| Socket/openSocket | pending sends全部结算，unbind/uncancel/close/close observer异常相互隔离；bind内同步关闭后仍卸载返回的句柄；message observer异常不阻断后续observer，关闭后停止分发。 |
| MiniSocket/WebSocket | 部分注册失败、四类off中一个抛异常仍尝试其他off和close，close一次；迟到native事件不再产生业务投递。底层off拒绝时不声称已物理移除该handler。 |
| MiniAuth | off异常不覆盖成功code或原登录失败；外部取消hook部分注册后抛异常时保持原错误，SDK调用0，保留hook迟到调用逻辑无效。没有返回off句柄时不保证物理注销。 |
| MiniHttp/WebHttp | 正常成功/原业务失败均不被remove异常替换，timer清理继续；WebHttp部分注册失败对照通过，MiniHttp同类N02失败。 |
| GameClientShell | 受控close/router清理抛异常且close重入dispose时，三个pending command均以UNKNOWN结算，一个sync结束，连接先置空，event/hand reset继续，重复dispose幂等。连接切换重入N01失败。 |
| GameOnlineController | dispose先撤销身份、代次和所有权，session/operation/fairness抛异常不阻断shell；依赖重入dispose不重复业务清理。logout的真实业务revoke错误原样传播，未被清理警告吞掉。 |
| GameBootstrap | 实际Creator方法在window/view off、密码字段、online/shell/table/node销毁抛异常、online销毁重入时，仍清完其他步骤；content/identity/数组在外部调用前已分离，旧页面业务回调失去所有权。 |
| GameTableView | 实际Creator类在root.destroy抛异常前清空原tickHandle/data/countdownLabel，原interval清理一次；更换view不重启共享TurnClockAnchor，同版本30秒时钟已过1秒后仍29秒。未声称被拒绝销毁的node已物理销毁。 |
| Vue App | 实际Vue根组件unmount；message/storage/visibility的原生remove全拒绝、诊断也抛异常，仍尝试全部步骤并移除iframe/root。手动调用保留的旧handler后HTTP0、新watchdog0。**拒绝时未物理移除**；测试结束恢复API并显式注销这些handler。 |

见transport-faults.*、game-cleanup.*、native-cleanup.*、vue-cleanup.*。没有把本地cancel或dispose解释成撤回已发送的服务器命令；UNKNOWN、已确认结果、登录撤销和服务端牌局权威保持原契约。

## 必要回归、冒烟与历史复用

**310项受影响客户端测试PASS、0跳过**：core59、Cocos198、Vue24、构建脚本29；core/Cocos类型检查及Vue vue-tsc/Vite通过。完整目录和命令见checks.py/checks.json。独立专项并未修改仓库产品测试。

Round10 N01原session驱动9项、实际Controller确认通过：旧SDK未回调前恢复已成功，迟到exchange0。Round10 N02通过实际Bootstrap取消菜单，SDK/prepare两个阶段在await旧Promise之前启动新任务，释放旧回调时新slot不丢失、无旧write。

**真实候选Web+匹配API冒烟PASS**：只创建一间自有房间，正常进入/导航复用同一iframe，新增静态请求0；关闭实际Socket后重连恢复同一房间；退出并关闭房间，注销iframe0且旧凭证401；换账号后room/私有hand/fairness为空。创建证据与同房后续连续结果见smoke-original-observer.json、smoke.json。未新开牌局。

WSS启动故障的实际浏览器检查PASS：7423ms内明确失败，无SESSION_ACTIVE，旧回调不覆盖，显式重载可恢复。实际HTTPS缓存检查PASS：HTML重验证、配置no-store、hash JS immutable、无hash cc.js条件304/0字节、身份no-store（cache-smoke.json）。

业务命令、游戏协议、Go/schema、自动操作机制和缓存策略未改。依照本轮授权复用 [Round11](0065-round11-independent-acceptance.md) 在固定产品33c05a7完成的三账号v61完整局、实际自动操作、原RequestID UNKNOWN、终局归档及个人字符串验证，以及 [Round9](0065-round9-independent-acceptance.md)—[Round10](0065-round10-independent-acceptance.md) 的有效SQL/压力/性能/崩溃恢复/缓存内存证据。当前连接及销毁改动以本轮专项和冒烟检查，重入故障单独FAIL，**不将历史完整局称为本轮完整端到端通过**。本轮未重复完整牌局、300秒等待、Go构建或后端性能/崩溃/缓存内存矩阵。

## 固定源码、构建和 CI

三项候选62acd92 CI（CI、Fair Doudizhu、Cocos linux_source）SUCCESS；native export job SKIPPED。Vue复用96f3913a5650017f6df705aa3db313700b92671c的SUCCESS；该提交到候选仅修改turn-clock.test.mjs，产品代码和Vue工作流完全一致。本轮另外重跑候选Vue构建/测试，不用等价关系冒充同SHA CI，见ci.json。

只构建一次官方Linux Cocos4 Web，CLI f506fa3de403b48b4833334fa48dddd27bb26364、engine d0a696cfe63fd8d19c70d7913ba11aa641c2e3c7、4.0.0-beta.0。正确在PATH加入CLI与项目node_modules/.bin，内部tsc已运行，不再not found。CLI报告56秒；内部仍有已知生成声明路径TS2688，保留日志；随后针对实际Cocos4生成声明的严格独立tsc与8份JS AST/BigInt审计PASS。未重复导出或修改产品源码消除提示。

1496个普通源码文件中1495个字节一致，Cocos package.json仅CLI重排、JSON语义不变；无其他源码偏差，见source-identity.json及逐文件artifact-hashes.json。API复用已验证二进制dc50adc3bdc3fabe04b64e0a66f417df44a1fb56fc35a2f3f225e9cdb714c000；候选与Round9—11 server tree均为a8f14fec9d7703c0fe68ac286bbd4aa9081fefbe。不是用旧集群API搭配新Web。

自有隔离环境：goal0065r9复用库、g65r12身份/队列前缀、API18870/HTTPS30471/CDP19181；Node22.23.4、Chrome for Testing156/Xvfb/SwiftShader、私有测试CA真实受信任。无证书错误绕过、HOME覆盖、Wine、手机黑屏调查或真机测试。

## 观察器修正及第二遍证据自审

首个原生专项驱动在收集结果时，错误地用pass:true覆盖返回的pass:false；其实际observed字段已显示四个失败。保留native-cleanup-initial-observer.json，修正**测试驱动**后从同一唯一产物重新执行，得到5PASS/4FAIL，和Node反例一致。未修改产品代码，也未保留错误的全通过结论。

冒烟首轮断线注入误关闭ConnectionManager层，不能据此等待Shell的native onClose，导致观察器超时；保存原结果后改关实际Socket。未新建房间重试，而是恢复同一个首轮房间，完成断连/重连、leave/logout/换号全部检查。该首轮失败不计为第四个产品问题。

第二遍自审核对了源/产物身份、全部cleanup引用、原P3修复与重入计数差异、N01两条链的先后顺序与假就绪、N02的“可注销但未尝试”及物理释放边界、原业务值/错误和UNKNOWN、同房冒烟连续性、诊断脱敏及历史复用范围。新发现只描述实际受控证据，不声称正常设备已触发或真实Go越权。独立验收结论仍FAIL，不修复生产代码。

## 部署与收尾

验收未通过，**未执行候选API→Web部署，也未触发回滚**。此前已验收API/Web镜像保留，见deployment-baseline.json。集群六Pod、两节点Ready；部署、Pod UID/镜像/重启数、ConfigMap、Service、StatefulSet、PVC及Secret hash均未变。集群ABORTED14/COMPLETED11/未完成0；隔离库ABORTED21/COMPLETED14/未完成0，与开始相同。

自有房间业务关闭；自有API/浏览器/临时Nginx及原已停止的四个隔离依赖均停止，18870/19181/30471关闭。临时测试CA昵称已移除，NSS证书列表恢复；用户29457服务active、实际HTTPS健康检查200。私有数据/产物保留便于复现；未公开账号密码、token、密钥、私有手牌或个人字符串。证据目录 [evidence/0065-round12](evidence/0065-round12/)。

Goal0065继续in_progress；Goal0045/0046状态保持原样。修复交回ChatGPT，收尾部署门禁尚未通过。
