# Goal0065 Round6 独立验收失败报告

结论：**FAIL；验收未全部完成。** 固定产品源码 `9942db7258e14d5342783263d8fe00a74f92045f`。已确认一个 P2、两个 P3；未修复产品源码，未部署候选版本，Goal0065 保持 `in_progress`。用户最新指示“**不修复，直接写报告**”优先于原先继续综合验收的执行安排，剩余回归停止，未执行项目单列，不能算通过。

## 用户当前黑屏事件与证据边界

用户报告：手机 Chrome 可以访问并登录，经 SSH 只映射 `29457`，加载大厅持续黑屏，多次刷新；之前登录 `fetch failed` 的 SSH 隧道故障已由用户确认。报告不包含用户密码。

**本报告没有取得该手机当次黑屏的前端控制台、网络瀑布或设备渲染证据，不能确认其根因。** 下述新问题是固定候选版本的独立验收发现，不能直接等同于该手机故障。现有 Kubernetes Web 仍为 Goal0064 基线，并非本轮候选；本轮真实浏览器验证通过隔离的本机 TLS 静态站点连接现有 Go 测试服务。Pod 健康也不能证明手机 Cocos 渲染正常；仅映射一个端口本身不足以证明发生了资源或 WSS 端口错误。

## 基线、构建与 CI

- 独立执行者：Codex；只提交本报告、脱敏证据及 `current.md` 状态记录。遵守 `docs/principles/`：服务端权威、持久化命令 UNKNOWN/幂等身份、客户端生命周期与有界资源、真实证据和设备证据分开。
- 干净 `main` 上 `git pull --ff-only`，固定上述源码；未创建分支，未修改 Go、平台接口、数据库或客户端产品代码。
- Node `22.23.4`；官方 cocos-cli `f506fa3de403b48b4833334fa48dddd27bb26364`；Cocos4 engine `d0a696cfe63fd8d19c70d7913ba11aa641c2e3c7` / `4.0.0-beta.0`。Linux/Xvfb，仓库 2D profile，真实 Web 导出约43秒；生成引擎声明严格类型检查及8份 JS 的 BigInt AST 检查通过。CLI 内部 TS2688 提示未作为严格检查替代品；导出后独立严格检查通过。
- Chromium156，headful/Xvfb/SwiftShader，信任隔离测试 TLS 证书，未关闭 TLS 校验。390×844 等是桌面浏览器视口/触摸模拟，**不是手机实机验收**。
- 实际 SHA 对应 CI：[CI](https://github.com/oublie6/awesome-zero-platform/actions/runs/45311445383) 六项成功；[Fair Doudizhu](https://github.com/oublie6/awesome-zero-platform/actions/runs/45311445588) 四项成功；[Cocos](https://github.com/oublie6/awesome-zero-platform/actions/runs/45311445437) `linux_source` 成功，`linux_native_export` **skipped**。本地真实导出独立完成，不能将 CI 的 skipped 写成执行成功。
- [CI明细](evidence/0065-round6/ci.json)、[构建日志](evidence/0065-round6/native-build.log)、[产物逐文件哈希](evidence/0065-round6/artifact-hashes.json)、[CSP/缓存配置摘要](evidence/0065-round6/fixture-proof.json)。未发布候选镜像。

## 已确认缺陷

### G65-R6-N01 / P2：无版本 timeout 的失效状态可被同版本旧快照消除

位置：`clients/fair-doudizhu-cocos/assets/scripts/game/GameClientShell.ts:89–108,295–306`。

复现使用真实 Shell、Router、HandStore、ConnectionManager，**受控协议端口和合成合法 DTO**：

1. 应用 handA/v2 的正常私有快照；保持连接。
2. 投递没有 `live.version` 的已提交 `hand.timeout` 变更；分别覆盖正在等待同步及上一读取已完成两种时序。
3. 再投递旧的同 hand/v2 无关联 ID 普通快照。另一路在 timeout 后发起新同步并返回同版本旧完整快照。
4. 前两种恢复 `connected`/可操作/投影就绪；第三种读取 resolve，而预期保持失效或拒绝陈旧结果。

处理器只取消当时的 `pendingSync`，未保存跨请求的失效屏障；`requiredHandVersion` 未因无版本通知提升，后续同版本确认可恢复状态。影响：已知发生权威变更后，陈旧视图重新具备操作资格。不能以“发送成功”或旧快照重新应用代替新鲜性证明。

[矩阵](evidence/0065-round6/async-matrix.json) 中三项失败属同一缺陷；[复现脚本](evidence/0065-round6/async-matrix.mjs)。真实 Go 的自动 timeout 已观察到，并正确拒绝当时的同步等待；但未捕获可用于延迟回放的真实无关联快照广播。因此**真实 Go 端到端的旧广播回放未验证**，不把受控 DTO 结果宣称为真实服务端或用户手机复现。

### G65-R6-N02 / P3：SDK send 未回调时，底层未完成 Promise 随读取累积

位置：`clients/packages/game-client-core/src/minigame.ts:77` 及 Shell 的同步超时/取消路径。

真实 MiniSocketTransport/openSocket/ConnectionManager/Shell，受控 SDK 不调用 send success/fail；注入10ms读超时，串行16次读取。Shell 等待器均结束，但仍有16个未完成 transport.send Promise；dispose 后监听器0、close一次，send Promise 仍16；释放受控 SDK 回调后才归零。

预期生命周期关闭和超时应约束底层未完成发送，避免反复重试无界保留。这里证明的是**回调式 SDK 的资源边界缺口**，不是 Chrome 已发生内存泄漏、RSS 增长或真实微信 SDK 故障。[结果](evidence/0065-round6/sdk-send-resource.json)、[脚本](evidence/0065-round6/sdk-send-resource.mjs)。

### G65-R6-N03 / P3：成员重新进入的 joinRoom 可在最后同步后断连仍 resolve

位置：`clients/fair-doudizhu-cocos/assets/scripts/game/GameOnlineController.ts:292–302`；对照 `performRecovery` 的最终连接检查约177行。

真实 Controller/Shell/Router/ConnectionManager，受控已认证 HTTP/SDK：现有成员进入活动房间，合法私有快照应用后、调用者 await 继续前同步触发 close。`joinRoom` 返回成功，实际 `transportReady=false`、`handStatus=disconnected`；同样时序下 `reconnectAndResync` 正确拒绝 `socket disconnected before recovery completion`。

预期公开恢复 Promise 应反映最终可用状态。join 分支 await 后仅复核销毁、代次及房间选择，缺少最终连接有效性检查。本项是**受控回调时序复现**；未证明真实 Go/手机发生该顺序，也未证明该处必然发布错误 SESSION_ACTIVE。[结果](evidence/0065-round6/controller-final-close.json)、[脚本](evidence/0065-round6/controller-final-close.mjs)。

三个受控脚本按原执行路径保留：同目录需有候选源码的 `input/` 归档、已构建 `dist` 和正确工作区依赖链接；执行 `node async-matrix.mjs` 等生成同名 JSON。它们不是直接调用线上环境的脚本，也不应当作实机测试。

## 已执行验证

| 范围 | 结果与限制 |
| --- | --- |
| 源码类型/构建/测试 | core33、doudizhu协议50、Cocos176、Vue14、构建脚本29，共302项通过；core/协议/Cocos类型检查及Vue vue-tsc/Vite构建通过。详见[命令及耗时](evidence/0065-round6/checks.json)及同目录测试日志。 |
| 私有同步协议矩阵 | 39项：36 PASS、3 FAIL，未处理 rejection 0；覆盖延迟/丢失/停滞send/拒绝、ID、版本/HandID、notModified、重复/迟到、合并/抢占、断连/替换/销毁/Hand切换、终局提升、ACK乱序等。受控协议证据，三项失败归入N01。 |
| R5-N01精确真实复验 | 真实发牌 BIDDING 后刷新，保持认证/HTTP/WSS，丢弃真实私有快照。无快照不发布SESSION_ACTIVE；约15,540ms自sync发出起超时，等待器清理，显示失败/重试；持久化命令0。移除故障后恢复同房同hand/v1。**原问题PASS**。 |
| 真实ACK/快照顺序 | bid/play/pass ACK-first，play/pass snapshot-first通过；每次仅一条真实命令。ACK先到时旧视图不可提交，不误报UNKNOWN；快照先到仍等待关联ACK。bid snapshot-first仅受控矩阵通过，未在本局真实重复。 |
| 真实自动timeout | 无live.version通知确实到达，较早的pending同步拒绝，状态保持synchronizing/不可提交。延迟真实无关联广播回放未执行，见N01证据边界。 |
| 必要HTTP恢复失败 | fairness503、member-room503真实端点各命中一次；无虚假会话就绪，移除故障可恢复。Round5已通过的十项HTTP故障保留历史证据，未宣称本轮全部重跑。 |
| 终局归档故障 | 503/401/403/404/畸形200/连接中断/持有请求超时，共7项PASS；均有真实端点命中和重试恢复。同hand/v62的真实WSS evidence回退PASS。各耗时含引擎重启，不能当作纯HTTP超时精度。 |
| R3-N01隐藏启动 | 实际阻塞cc.js→返回网站→iframe屏外停放→释放引擎→重新进入。三尺寸均实际非零布局、Camera及UI矩阵有限，真实菜单鼠标/触摸命中；反复导航复用同一iframe，新增资源/WSS/持久化命令均0。 |
| R3-N02动态CSS | 1280×900、390×844、760×390通过；移动视口延迟CSS，无提前ready，Canvas边界无5px边框裁切；旋转后矩阵有限。详见[完整几何](evidence/0065-round6/layout-runtime.json)。 |
| 原生加载页 | 慢提示约15,295ms、无进展失败约60,610ms；迟到真实引擎成功未消除失败终态。 |

[R5/ACK/HTTP/timeout/加载页运行时记录](evidence/0065-round6/live-probes.json)、[R5失败遮罩及重试截图](evidence/0065-round6/sync-timeout.png)、[终局故障矩阵](evidence/0065-round6/archive-probes.json)。`GAME_READY` 是引擎/场景阶段消息，单独出现不等于 `SESSION_ACTIVE`，判定时按各消息语义区分。

## 真实三账号业务回归

三个自有普通 QA 账号，经候选 Cocos4 Web、真实 HTTP/WSS 和 Go 权威服务完成 hand `hand-cmd-2949724b32e023274a943a76ca008789`，终局 **COMPLETED/v62**。

创建/加入/准备、真实贡献/HPKE/信标发牌、叫地主/出牌/过牌、自动超时、BIDDING和PLAYING刷新、WSS断开恢复均执行。实际非法两张不同点数出牌收到 Go `INVALID_REQUEST`；贡献响应丢失和活动牌局ACK丢失均以原RequestID/clientSeq/expectedVersion/payload精确重试。三人个人公平验证：本人正确口令通过、错误口令拒绝。完成后按成员先退/房主后退关闭自有房间。

[脱敏业务步骤](evidence/0065-round6/business-summary.json)只保留状态/版本/结果；不提交原始私有手牌、口令、贡献、Token或TLS密钥。

执行有夹具中断，未伪装成一次无中断通关：旧房房主先退被正确拒绝；成员退房未刷新版本被正确拒绝；真实第二次自动操作令读版本变化；重连后界面回到房间页，私有驱动误点旧牌桌节点。修正仅限外部测试驱动，同一主牌局继续至v62，未修改产品或重造权威牌局数据。另一次最终门禁探针的自有牌局通过公开命令取消，不直接删库。

## 未完成与不可判定项目

- 发牌前→已发牌跨越最终场景await的探针未到达预定第四次房间读取等待点，**夹具时序未验证**，不是新P2；准备的替代探针未运行。
- 后续无房间/房间关闭最终门禁夹具报 `No UI action CreateRoom on rooms`，无房间前置条件未证实；其 `pass:false` 不能直接定为产品缺陷。未执行依赖它的房间关闭、多个恢复房间、真实A→B→A、最终await断连、注销/换号/快速reload场景。
- 冷/热启动缓存统计驱动在 `page.goto(...networkidle)` 超时，未取得完整结果。新一轮304、版本失效、业务响应缓存隐私实测、冷/热可交互时间/传输量/内存分位统计未完成。布局场景的同iframe复用通过不替代这些检查。
- 本轮隐藏计时暂停/真实后台挂起、配置握手边界剩余项未完成；原生可见状态15/60秒看门狗通过不代表后台暂停通过。
- 手机Chrome当次黑屏、微信小游戏实际SDK/真机验收未完成。不能将桌面视口或受控SDK当作实机结论。
- 不重复未受影响的Go/SQL/Redis/密码学压力与崩溃测试；历史证据可复用，不作为本轮新执行记录。
- 上述项目现依用户指示停止，不修改验收标准、不补写PASS。[未执行清单](evidence/0065-round6/unexecuted.json)。

## 部署、环境与交付

有P2且必需门禁未完成，**不部署候选**，所以没有候选部署后验证或候选回滚动作。保留原Web Goal0064镜像摘要 `44995360be6e91a0e7e1c355db4e8e9027633d1fc4d9eb59b15b110990af0a0f`，API Goal0063摘要 `04614d0bc04c1fe37d2cb4467be23fec728cb0b1cfb63c26b1cd3746943ec82f`。

[最终只读审计](evidence/0065-round6/final-audit.json)：Deployment/Pod UID、镜像/重启、ConfigMap、Service、PVC、StatefulSet与基线一致，六Pod均Ready，未增加重启。数据库ABORTED 11→12、COMPLETED 8→9，未结束牌局0；这不是数据库零写入，而是授权QA业务命令的结果。未执行SQL删除、迁移或集群配置修改。用户 `goal0063-game-test-forward-v2.service` / `29457` 保持active。本轮专用 Chromium/CDP19175 与 `codex0065-r6-web-9942db7` 临时容器已停止；未停止用户服务。

交付自审：按证据重核每项严重性、真实与受控复现边界、失败/未执行计数及固定SHA；剔除夹具假阴性，未把未验证的最终门禁探针作为产品问题；仅白名单字段和无敏感信息截图入库。报告自审不是第二次独立产品验收。测试源修复留给开发方，Goal0065、0045、0046不宣告完成。
