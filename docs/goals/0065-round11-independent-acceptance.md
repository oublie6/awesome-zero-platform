# Goal0065 Round11 独立功能验收

结论：**FAIL，Round10 两个 P2 原复现通过；新发现 1 个 P3（通用操作清理异常隔离）。** 不修改产品代码、不部署候选版本。Goal0065 保持 `in_progress`，Goal0045/0046 不变。本报告完成独立验收任务，不表示产品 Goal 完成。

基线：main `33c05a74a98315b2d90a51e817b3cf1fcde122a6`；审查增量 `5bf4e2ab6cec01695acffa2ed98bf824ab9d750d...33c05a74a98315b2d90a51e817b3cf1fcde122a6`。2026-11-26 UTC。已读取 AGENTS、原则、当前 Goal、Round10 报告及本轮19文件增量；只有客户端、测试及文档变化。

## Round10 缺陷复验

| 原问题 | 本轮证据及结果 |
| --- | --- |
| N01/P2 取消旧登录仍阻塞合法恢复 | **PASS**。实际 GameSession/MiniAuth/MiniVault/WebAuth 原驱动9项全通过；dispose/logout/discardLocal 后，旧SDK/HTTP尚未返回，新恢复已调用validator并成功。实际Controller恢复立即建立身份和传输，旧SDK回调exchange0。 |
| N02/P2 取消换绑仍继续发HTTP | **PASS**。实际官方Creator包的Bootstrap构造 WechatCancelRelink 菜单，调用真实action；SDK等待和prepare阶段取消后，旧操作结束且不再发最终请求。立即重新进入同页的新操作成功，旧finally/迟到SDK不影响新操作。另2项实际菜单检查在await旧Promise之前就启动新操作，新write等待期间释放旧SDK/prepare结果仍保留新slot（bootstrap-immediate.json）。实际renderScreen A-B-A同样通过。 |

同实例logout/discardLocal/cancelAttempt/restore恢复、WebAuth logout/dispose/restore、旧finally与新pending隔离、正常进行中的新登录阻止旧恢复、正常失败恢复后的登录回退，均有正向和负向对照。已开始的异步vault写仍严格依次 `save old → saved old → own clear → save new → saved new`，没有以取消强行打断写入。正常绑定所需attempt在身份交换拒绝后仍有效，后续Controller登录成功。实际Creator包另通过完整的首次绑定、注册绑定两条链：生产Bootstrap.login → 原生WeChatBindingRequiredError → 实际绑定表单 → 最终write → 再次平台登录 → 有效身份/WSS/账号资料/房间大厅；每条2次平台登录、1次绑定write、1次socket，所有SDK/HTTP仍受控（binding-chain.json）。

见 [session-lifecycle.json](evidence/0065-round11/session-lifecycle.json)、controller-cancel-restore.json、session-extra.json、web-auth-extra.json、bootstrap-lifecycle.json。受控平台接口不是实际微信映射修改或真机证据。

## G65-R11-N01 / P3：清理回调抛异常会中断其他等待任务的取消

位置：`clients/packages/game-client-core/src/operation.ts:26`，`end()` 先清空listeners，直接调用release，再逐一调用快照中的handler，没有异常隔离。

复现条件：一个OperationScope注册的release或前置onCancel handler同步抛异常，同时存在另一个尚未返回的wait及后续清理handler。调用cancel/abandon/finish所共用的end后，抛出的异常中断遍历；后续wait的cancel没有调用，后续清理handler也未执行。scope已结束且listeners集合已清空，重复cancel/finish不会补做这些清理。

预期：即便向调用方继续报告清理异常，也应尝试其余清理handler并及时结算其他wait，不应让首个异常跳过这些工作。

实际结果：受控清理handler抛出 `cleanup failed` 后，事件只有 `throwing`，没有后续 `last`；等待任务直到自身deadline才报 `operation timed out`，而不是立即 `operation cancelled`。release抛异常也有同类结果。测试用30ms期限避免长等，未修改生产实现。正常等待默认15秒；不能把这个观察说成永久泄漏。

**实际官方Creator降级产物中也复现**：从真实Controller的OperationSlot获得生产构造器，运行同一清理异常场景，仍只有前置handler，等待最后超时。不是只看源码或Node构建。

严重程度为P3：这是新增公共操作组件在异常边界的清理契约缺口；本轮审查的正常生产调用方release/handler未观察到抛异常，正常SDK取消、旧任务隔离和业务流程通过。**不声称已经造成真实微信账号错误、牌局损坏或现网资源持续泄漏，也不升级为P2。** 但本轮明确要求通用组件异常和多等待任务统一清理，此门禁尚未通过，故不部署。

证据：[operation-independent.json](evidence/0065-round11/operation-independent.json)（10 PASS / 2 FAIL，两个失败为同一根因）、[bootstrap-lifecycle.json](evidence/0065-round11/bootstrap-lifecycle.json)（22 PASS / 1 FAIL，唯一失败同此根因）。只报告，不修复。

## 通用组件及全部调用方

生产引用经 `rg` 全量定位，清单与证据映射见 callsite-review.json。OperationScope为纯TS技术能力，未引入SDK/DOM/AbortController依赖；OperationSlot只由Controller持有。Go/业务数据权威和UNKNOWN契约没有改变。

| 组件/调用方 | 实际检查 |
| --- | --- |
| OperationScope / OperationSlot | 正常同步开始和值、同步throw/异步reject、重复cancel/finish/abandon、立即重入、旧finally不释放新owner、多waiter、嵌套scope、截止期限及迟到成败、非法期限不派发、start内同步取消、dispatched不可取消与abandon只结束观察。正常项通过；异常清理隔离失败如上。 |
| GameSession.login/loginWith | 共享pending以scope所有权释放；取消、同实例恢复、新登录替换及存储串行化通过。正常认证所需attempt未被普通finish误清。 |
| MiniAuth | 默认15s、实际短期限、主动取消、observer卸载、无code/失败/同步throw、迟到callback。受控SDK接口通过；无真实SDK资格声明。 |
| MiniRandom | 默认15s的受控timer触发、精确长度、缺少安全SDK拒绝、失败/畸形数据拒绝、迟到回调；无Math.random降级。 |
| MiniShare | Douyin默认15s无回调返回false、迟到成功无效；正常success/fail及合法房间query。WeChat true只表示同步调用分享UI，不表示送达。 |
| Controller.providerOperation / completeWeChatBinding / providerWrite | 单slot、重复请求不派发、每阶段owned/grant/dispose、前置绑定、SDK/prepare取消后重入。最终write标记dispatched后实际取消菜单保持原表单并提示等待；正常完成更新身份/页面。 |
| Vue WebAuth / App | 使用独立Abort所有权；销毁、同实例恢复、替换账号、旧finally隔离通过。真实Vue根组件unmount时取消在途登录，迟到回复不保存token、不发后续session读取；重新挂载后真实Go登录成功。 |

小游戏Node侧13项适配器检查通过；实际官方Creator包通过工厂创建微信/抖音适配器，登录、随机数、分享正向，以及登录期限/取消/迟到callback共4项通过。部分15秒适配器期限用受控timer触发并核对原期限值，另有真实5ms期限对照；不是声称每次都实等15秒。原生bundle的scope多等待/嵌套/旧owner检查也通过。

相邻审查覆盖Shell、ConnectionManager、RoomCoordinator、HandActions、FairnessActions及Bootstrap/Vue的异步入口、finally/reservation、dispose、timer/listener、账号/房间/hand代次和精确请求UNKNOWN。已有对应客户端测试重跑；Round8登录旧完成及最后权威读取两条入口独立复验通过。房间发现/重连结果在grant失效后不再进入旧恢复、采用旧room/hand或调用后续refresh。WebShare未被当前Bootstrap业务路径调用，不能把本轮小游戏分享证明扩展成系统剪贴板权限行为证明。

## 账号、页面和启动实际行为

- 真实编译Bootstrap及Controller的绑定、注册绑定、解绑、换绑正向通过；provider-matrix36项通过，保留正常请求而非拦截全部操作。
- 四种最终身份write已派发时，“取消”均保持当前表单并提示“正在等待服务端结果”，重复提交0；正常回复后绑定继续登录、解绑/换绑detach旧身份，无额外revoke。未将本地取消解释成服务端撤销。
- 四种write响应丢失时均提示结果待确认，并阻止直接重复提交；没有谎报已取消或已失败回滚。另5项实际Controller检查覆盖SDK/prepare取消与超时后的新操作、final write超时后的待确认锁；受控timer把请求的15秒期限加速至5ms，并明确记录原期限值（provider-timeouts.json），不声称实等15秒。
- 实际独立Web登录CancelWebLogin菜单结束旧登录，迟到HTTP不能建立旧身份或改写原存储；真实Controller/Session链执行，非只验证按钮源码。
- Vue正常网站/游戏切换复用同一iframe，新增静态请求0；真实子frame storage事件和真实hidden→foreground均移除旧iframe、保留替换token、自动revoke0，重新加载只恢复新账号，无旧房间/私有牌/公平状态。旧frame消息不影响当前页。
- WSS启动失败：真实浏览器/Go身份、受控关闭握手，8920ms内SESSION_FAILED/BOOT_FAILED，hostRestoring结束，无SESSION_ACTIVE；移除故障后显式重载新frame恢复“游戏就绪”。不是实际Go502或手机测试。

## 必要回归和产物

**303项源码测试全部通过，无跳过**：core54、Cocos196、Vue24、构建脚本29。core/Cocos类型检查、Vue vue-tsc/Vite通过。执行命令和目录见checks.py/checks.json及日志。新增独立驱动不是仓库产品测试的修改。

四项固定SHA的CI SUCCESS：Vue、Fair Doudizhu四jobs、CI六jobs、Cocos linux_source；Cocos linux_native_export **SKIPPED**，未算作导出通过。见ci.json。

只执行一次官方Linux Cocos4 Web导出：CLI `f506fa3de403b48b4833334fa48dddd27bb26364`，引擎 `d0a696cfe63fd8d19c70d7913ba11aa641c2e3c7`，4.0.0-beta.0。CLI报告1分5秒；真实生成声明严格类型检查及8份JS AST/BigInt审计通过。CLI内部非资源tsc子命令因本轮驱动PATH未带其bin报告not found，保留原日志；随后针对实际生成声明的独立严格tsc门禁成功。没有重导出来抹掉这条观察，也不把CLI内部提示当产品通过证据。

实际浏览器使用唯一候选产物与API18870、HTTPS30471、CDP19181；Chrome for Testing156、Node22.23.4、Xvfb有头/SwiftShader、受信任私有测试CA。测试CA以独立NSS昵称临时导入并在收尾移除，不使用ignoreHTTPSErrors，不改HOME或系统根证书。无Wine、手机黑屏调查、真机或真实微信账号写入资格声明。

源码隔离快照1420个普通文件中1419个字节一致，唯一差异是CLI重排Cocos package.json且JSON语义相同；无产品源码差异，详见source-identity.json。原生与最终托管文件逐一SHA256见artifact-hashes.json。

**真实三账号完整牌局 PASS：COMPLETED/v61，exit0，90条步骤记录，pageerror0。** 自动发牌通知触发关联私有读取；原300秒自动操作触发，其余客户端无需手动刷新同步到v2，迟到同版本timeout自动修复。实际Bid/Play/Pass的ACK先/快照先、页面刷新同局、原RequestID的UNKNOWN重试、WSS断开重连11项live检查全通过。非法双张牌型由服务器拒绝并有可见提示，三账号终局归档及个人字符串正反校验各3次通过。

三人原上下文注销后iframe均为0、旧凭证均返回401；随后换号无旧room/手牌/fairness。下一局ready菜单无重复，自有房间退出关闭，没有再开第二局。旧timeout在终局回放不抬高门槛或破坏归档。控制台3条401来自预期的注销后旧凭证拒绝检查。game-summary.json只发布白名单脱敏投影，不发布牌、短语、令牌或截图。R10已补齐的上一局取消通知链未改动，本轮复用其真实WSS证据，未重复开取消用局。

## 历史复用、观察器和第二遍自审

候选server tree与Round9/10同为 `a8f14fec9d7703c0fe68ac286bbd4aa9081fefbe`，匹配API二进制SHA256 `dc50adc3bdc3fabe04b64e0a66f417df44a1fb56fc35a2f3f225e9cdb714c000`。只复用API，不用旧部署API搭配新Web。Go/schema/协议/缓存机制未变，按授权复用 [Round9](0065-round9-independent-acceptance.md) 和 [Round10](0065-round10-independent-acceptance.md) 的有效MySQL、性能/压力、崩溃恢复、协议及缓存内存证据，不重复执行，也不将历史指标称为本轮测量。

复用自有隔离数据库goal0065r9，开始ABORTED21/COMPLETED13/未完成0；独立auth前缀g65r11与队列前缀g65r11。自有账号、令牌、密钥、手牌和个人字符串只留私有目录；公开证据脱敏。发布日志仅规范化行尾空白，原始日志私有保留；manifest-sha256.json固定发布文件哈希。

观察器修正：Round10 Controller驱动在“立即恢复已成功”后还尝试同一已认证Controller再次认证，违反既有契约；本轮将其改为检查既有成功状态。旧Bootstrap驱动仍期望已提交bind/register可以取消，其两个失败符合本轮明确修订后的不可取消write契约；初始结果保留bootstrap-initial-contract-observer.json，并用实际renderScreen/菜单及等待/未知结果的新矩阵验证。复制API时初始执行位丢失，恢复同一二进制执行权限后启动；无产品修改。未将这些观察器问题计为产品缺陷。

提交前已进行第二遍证据自审，核对：固定源码/产物、全部scope调用方、清理异常根因与严重程度、原P2正向复验、提交边界与UNKNOWN、正常业务完整终局、证据脱敏/哈希及实际资源收尾。新P3经Node生产模块及官方Creator产物交叉复现；正常调用方未观察到该清理异常，因此维持P3范围，不夸大为真实账号/牌局故障。原两个P2及所有其他实际执行项目均通过；该异常清理门禁仍FAIL。已核对公开证据与原始日志、固定源码/产物及预期失败单元，未用单个绿色测试替代完整门禁。

## 部署和状态

新P3使本轮通用生命周期门禁未通过，因此不部署；未执行API→Web候选上线或Web→API回滚。Goal0065保持in_progress，Goal0045/0046原状态保留。集群六Pod/两节点Ready，部署、Pod UID/镜像/重启、ConfigMap、Service、PVC、StatefulSet及Secret hash未改变；集群库ABORTED14/COMPLETED11/未完成0。隔离库结束ABORTED21/COMPLETED14/未完成0，仅新增本轮完整局。部署原镜像见deployment-baseline.json；用户29457转发及其他基础设施保留。自有API、浏览器、临时Nginx和四个原先已停止的隔离依赖均已停止，18870/19181/30471关闭。临时CA昵称已移除，NSS证书列表与测试前一致；29457服务active且实际HTTPS /health返回200。私有产物/数据库保留以便复现；见owned-cleanup.json。
