# Goal0065 — Cocos Web 启动反馈与静态资源缓存

Status: **in_progress** — ChatGPT 开发提交；待固定源码 CI 与 Codex 独立实测。

## 现状与问题

Vue 从网站进入游戏时原来每次增加 iframe key；返回网站卸载 iframe。
Cocos index.ejs 在引擎加载前仅显示深色 GameDiv，用户看到黑屏。
目前 Web 资源的静态缓存头依赖部署环境，不能把当前测试镜像当成
已启用新缓存策略的证据。

## 设计和归属

- Vue 只负责账号入口、同源 iframe、启动进度覆盖与同会话 iframe 生命周期；
  Cocos Web HTML 原生加载遮罩在脚本执行前出现；Cocos 引擎/场景和服务端
  会话实际事件决定进度；Go/房间/手牌仍为服务端权威，不修改协议。
- 启动阶段：HTML(1/4) → 引擎(2/4) → 场景(3/4) →
  服务器会话/恢复完成(4/4)。这些数字为**可验证阶段数量**，不是
  网络下载百分比。不能自行定时伪造 90%。
- 15 秒无进展提示慢加载，60 秒无进展展示失败/重试；错误提示不附带
  令牌、URL 或服务端私密响应。双层加载页保证独立 Cocos 页也可用。
- 通过同源 origin/source 检查接收 BOOT_STAGE/BOOT_FAILED/BOOT_RETRY_REQUESTED，
  GAME_READY/SESSION_ACTIVE/SESSION_FAILED 原登录桥保留。
- Vue 在同账号网站导航中仅隐藏 Cocos iframe；明确重试重建，注销强制销毁；
  不存储账号令牌、牌局、验证字符串，不复用不同账号游戏实例。
- HTTP 缓存：指纹化静态资源 immutable；未指纹化 Cocos 文件条件 revalidate；
  HTML revalidate、公开运行时配置 no-store；不得缓存认证响应；不引入 SW。
  缓存配置为待部署验证的运维模板，并非本次自动部署。
- 资源占用有界：每账号仅一个 iframe，计时器可清理，卸载时清零。
  回滚为之前 Vue/Cocos 镜像，不迁移 MySQL/Redis/DQ/协议。

## 验收

1. Vue 与 Cocos HTML 原生加载页在网络慢速/冷启动时先于 Cocos Canvas 可见；
   阶段仅由真实事件推进，失败、超时及重试可操作，无黑屏。
2. 同账号返回网站再进入，iframe 身份不变，无二次加载脚本/静态资源；
   退出账号后旧 iframe 立即销毁，换号重新校验并创建全新实例。
3. 浏览器刷新和再次打开：分别检查资源 HTTP 状态、Cache-Control、ETag/
   Last-Modified、304 和传输字节数；验证新版本不会被旧资源长期污染。
4. 390x844、760x390、1280x900 真实 Cocos 浏览器布局及
   真实 HTTPS/WSS、CSP 审核；错误页/404、脚本失败、超时和断网。
5. 回归三人房间与服务端恢复、UNKNOWN 精确重试、注销会话隔离；
   CI + 实际 Cocos4 native Web 导出 + 独立 Codex 验收。
6. 记录冷/热启动性能、每阶段耗时及 transferSize。未测量之前
   不宣称实际提速或已部署缓存。

## Round 1 FAIL 集中修复（待 Codex Round 2 独立验收）

- N01：启动就绪的判定现在要求账号认证且真实 WSS 传输连接可提交命令；配置尚未确认时不发送 GAME_READY。HTTP 鉴权成功但 socket 失败只能报 SESSION_FAILED。
- N02：网页进入/离开游戏以经过同源鉴别的 BOOT_RESUME/BOOT_PAUSE 协调原生 Cocos 计时器；iframe/tab 隐藏时停止原生 15/60 秒计时器，真实异常延后到恢复可见时处理。
- N03：公开配置改为异步脚本加载和明确的 load/timeout 结果；HTML 解析不再被 game-config.js 阻塞，Cocos 本地启动页会实际创建并显示，GAME_READY 在配置成功装载后才发送。
- 额外状态防护：Vue 拒绝乱序 SESSION_ACTIVE/在失败后迟到的 GAME_READY；隐藏页面错误不再悄悄销毁缓存的 iframe；未改变 Go/房间/牌局协议、密钥或身份。
- 变更后应重新计算和验证实际 Cocos Web 导出页面新增内联脚本 CSP 哈希；无 Web 上线，原 Goal 0064 部署继续保留。完整 CI、浏览器隐藏/恢复与 WSS 故障回归、源码完整三人牌局和独立验收仍未通过前，**Goal0065 必须保持 in_progress**。

## Round 2 FAIL — 集中修复及 Round 3 验收范围

- R2-N01：原生 Cocos ENGINE_INITED 不再使用 window.innerWidth 在隐藏 iframe 时强制写尺寸；改为读取实际 iframe 可见状态、GameDiv 和 Cocos3dGameContainer 的非零几何，使用公有 screen/windowSize API 在可见时应用。ResizeObserver、同源 BOOT_RESUME、浏览器 resize/visibilitychange 用于安全恢复，布局竞态不会中断 engine 初始化。
- R2-N02：原生启动增加不可逆 terminalFailure。60 秒超时、CSS/引擎故障或隐藏期间真实故障进入该终态后，迟到的 stage/done/style onload 不再隐藏错误界面，只有明确重试并创建新页面才重置启动流程。
- R2-N03：移除 HTML head 中阻断解析/脚本执行的外部 stylesheet，首屏启动页与必要 GameDiv/Canvas 尺寸采用内联关键 CSS；启动 watchdog 后以 print 非渲染阻塞方式加载 Cocos 样式，onload 后切换为 all，onerror 进入失败终态；stylesheet 加载完成前不可宣布 Cocos/Vue 游戏就绪。
- 关联修复：Vue 在隐藏 iframe 或后台标签期间不抢先处理 CSS/config 所致的 SESSION_FAILED/BOOT_FAILED；原生错误在恢复可见时统一呈现，避免正常返回网站后重新进入自动销毁缓存 iframe。
- 新增真实画布尺寸门禁：Cocos 真正成功设置非零 windowSize 且外部 CSS 已应用后才完成 __AZP_GEOMETRY_READY__；独立页 done 和嵌入页 GAME_READY 都要等待该信号。CSS 在引擎之后抵达时强制重新测量，既防止 0×0 假就绪，也避免 ResizeObserver 缺失导致卡住。
- 显式点击重新加载立刻冻结原启动代的 timer/stage/done，迟到事件不得在 Vue 替换旧 iframe 前再次完成当前页。
- 新增 VM 行为回归及源码门禁；实际 Chromium CSS stall、hidden-success engine、迟到事件、3 视口与真实 Cocos4 导出、CSP/304、三人游戏均应由 Codex 在 Round 3 独立验证。Goal0065 仍 in_progress；本轮没有测试集群或业务数据变更。

## Round 3 FAIL 集中修复（待 Codex Round 4 独立验收）

- R3-N01 / P2：根因是 Vue 使用 `v-show` / `display:none` 停放 Cocos iframe，使隐藏期间下载完成的引擎在零布局下激活 UI 和摄像机，得到非有限 worldMatrix，单纯恢复 Canvas 像素尺寸无法修复。改为位于真实宽度的 `game-layout-parked` 绝对定位屏外元素，禁用交互与可访问焦点（`inert`、`aria-hidden`、`pointer-events:none`），保留原 iframe 引擎/缓存与真实布局。
- 真实渲染门禁：Cocos 的 `GameBootstrap` 在原生 done、GAME_READY、SESSION_ACTIVE 之前要求 Camera、Canvas、SafeContent、标题和所有菜单按钮均拥有有限的 worldPosition/worldMatrix，以及有效按钮尺寸。通过 rAF 等待真实布局；若不可恢复，显示错误而不是假游戏就绪；不改变 Go 权威牌局数据。
- 后台加载协调：隐藏期间可能已完成资源和 scene，Vue 暂缓发送 `SESSION_AVAILABLE`，直到返回游戏或后台标签恢复前台才启动 Web 会话验证，避免重复连线或无意义的后台业务初始化。
- R3-N02 / P3：Cocos 生成样式在动态加载后会覆盖内联的 `#GameDiv` margin/border/radius/shadow，导致 5px 左偏与画布溢出。内联的视口关键样式加最高优先级 `!important`，保持三个视口无偏移。
- 回归测试：修正源断言为双重（authenticated+transport+render）就绪验证；新增 parked iframe 布局/输入隔离与 Cocos 相机、按钮矩阵及 CSS 边框的定向约束。实际 `cc.js` 隐藏完成、横竖屏、菜单真点击、CSS 慢加载与三人牌局仍需由 Codex 在 Round 4 独立验收后才能部署测试环境。本轮没有修改测试集群。

## Round 4 P2 启动顺序与关联竞态集中修复（待下一轮独立验收）

- 启动闭环由 **身份验证→WSS 连接→房间发现→授权恢复→必要公平/牌局数据同步→有效场景→SESSION_ACTIVE** 组成。过去 `login(true)` 即使 `recoverMyRooms()` 发生 HTTP 503 或自动恢复失败也正常返回，导致 Vue 错报“游戏就绪”；现在 `login(): Promise<boolean>` 只有完整验证才返回 true，错误/陈旧请求返回 false，保留已验证的账号身份但将本次启动判为失败。
- 关联修复：`performAuthorizedRoomRecovery` 对明确目标返回 null 视为失败；调用 `refreshRoom(true)` 不允许吞掉房间/公平阶段/终局证据/实时同步错误，并在最终读取成员资格前验证 `roomView` 已经同步到目标房间。用户主动刷新仍采用原 best-effort 行为，不改变一般操作体验。
- 同源启动握手保存所属 controller/epoch/selection；注销、换号、关闭 iframe 或新选房间使迟到的登录/恢复成功失效，防止旧请求错误发送 `SESSION_ACTIVE`。恢复失败不会再被一次普通 `renderScreen('rooms')` 覆盖提示；展示原生错误界面，由用户明确重试重新进行只读房间查询，不用新建持久化房间命令掩盖失败。
- 已增加两组启动门禁/必需读失败/时序测试。仍需 Codex 在真实 Cocos Web 环境注入 /me/rooms、/room、/fairness、/evidence 的 HTTP 503，验证现有房间可恢复、无房间正常成功、401/403/超时以及中途注销、重试、UNKNOWN 时序；后续完成上一轮未执行的 R3-N01/R3-N02 及部署验收。

## Round 5 私有牌局同步与异步调用链集中修复

设计边界：GameClientShell 负责已有 WSS 读取的完成语义，纯 TS GameMessageRouter 透传已有服务端 envelope.id，DoudizhuHTTPClient 与 WSS 共享终局证据验证；Go/数据库/协议不变。每个 shell 至多一个兼容的在途同步读取，15s 有界超时；同请求复用，切换、断连、销毁、替换及超时清理；直接回应需匹配 ID/hand/连接代次，严格验证版本、私有投影和 HandStore 状态后才 resolve；广播只更新权威版本，不冒充应答。notModified 需具备匹配的可信本地快照。终局回退使用同一归档验证器。收到已提交 live ACK 但私有视图尚未达到 ACK 版本时阻断新动作，保留 UNKNOWN 原始请求身份。房间严格刷新、控制器恢复及普通同步调用者自动继承新的完成契约，刷新后的 UI 再次检查账号/房间选择代次。不新增协议或持久化内容；故障只触发读取重试，不补发写命令。正常同步增加等待真实响应的时间，不新增常驻轮询；通过取消和单一计时器限制资源。回滚只需原客户端产物。新增行为测试覆盖等待、超时/迟到、同请求合并、错误/畸形/错牌局/旧版本/无效条件响应、连接及牌局切换、终局归档和 ACK 与快照版本差异。真实 Cocos 导出、旧缺陷复验及集群部署仍须 Codex 独立验证。

补充审计：恢复成功发布前再次读取当前 HandStore/WSS/权威房间展示状态，避免最后一次等待布局或 HTTP 校验期间新变更/断连使早先已完成的同步失效；广播 live.version 与命令 ACK 一样构成最小确认版本，乱序旧变更不得降级已确认的更新版本。没有房间或处于合法发牌前阶段不强求私有牌局；已结束房间需确认归档。同版本 COMPLETED 私有投影可由已验证终局证据提升为归档投影（普通同版本快照依然不覆盖）。控制器恢复与成员只读重入也验证实际连接状态。增加新版变更失效、终局同版本提升和最终门禁回归。

补充行为验收：实际控制器→WSS→Router→HandStore 的恢复和已有成员重入等待关联响应，未关联广播、匹配的 FORBIDDEN 错误不允许提前成功；不生成房间/牌局持久化命令。无 live.version 的服务端已提交 hand.timeout 通知使更早的在途读取失效；同版本但阶段冲突的响应必须失败。
## Round 6 关联缺陷集中修复（待独立验收）

设计与归属：Socket 生命周期属于共享 game-client-core；状态新鲜性与请求确认属于斗地主 Shell/Controller；身份令牌写入与响应期限属于现有会话/认证适配器。保持 Go 权威、既有 envelope、UNKNOWN 原始请求身份、同源部署和服务端持久化不变。客户端回滚不需要数据库迁移。用户本轮明确不处理手机黑屏，实机故障调查不在本轮范围。

- N01：无版本的已提交 timeout 保存最低确认版本（失效时已知快照版本+1），同步超时/失败及重连均不清零；同版本广播、匹配的旧完整响应、notModified 不得解除，重复通知在更高版本应用前不会持续抬高下限。新 hand 重新绑定后重置。现有 Go 无版本通知无法证明新鲜性时保守拒绝，真实自动操作及乱序正负控制须独立复验。
- N02：共用 openSocket 直接拥有 callback send 的 Promise、超时和32项容量，Mini SDK 只提供回调，Web 同步确认本地发送；发送超时关闭原 task，关闭/替换/超时结算全部发送并释放等待器，迟到回调无效。sync deadline 传递到底层。未确认的写命令仍 UNKNOWN，已确认 ACK 不重新解释。真实 SDK/内存实测不可由受控回调替代。
- N03：已有成员 join 与 reconnect 共享最终连接、房间版本及必要私有/终局投影检查；已接受的持久化 Join ACK 仍是独立确认契约。覆盖合法快照应用后同步断连/权威状态失效的最后异步窗口。
- 关联审计：命令 envelope.id 与 payload.requestId 同时存在时必须一致；错误回复不能消耗其他请求。Vue 身份 HTTP deadline 覆盖响应体读取；旧验证不能恢复被替换的令牌，旧 login/restore 不返回另一个会话的成功。GameSession 串行化本地 vault 写操作、注销先清本地再等待服务端 revoke；Web/Mini 同步存储使用 compare-and-clear，避免旧账号的延迟清理删除新令牌。
- 回归：新增 core 发送容量/超时/关闭/迟到回调、会话旧保存/注销/验证、Shell 新鲜度/关联 ID、真实控制器末次断连/失效、Vue 响应体超时/换号和 SDK→Shell 跨层测试。源码及受控行为测试不等于独立 Cocos Web/真实 Go 回归。保留 Round6 已通过的 R3/R5 原复现和三账号业务证据，仍需复验受影响项并完成缓存、生命周期和部署门禁。Goal0065 保持 in_progress，不改测试集群。

## Round 7 集中修复 — 待 CI 与独立验收
- 版本所有权：已知服务端版本才建立数值下限；无版本 timeout 只设因果失效屏障，废弃在途读取、拒绝未关联广播解除及条件响应，要求通知后发出的完整权威读取。新关联同版本回复允许解除屏障（Go 当前读取是权威；无法仅凭无版本通知区分旧/已涵盖事件），真实低版本仍拒绝，终局归档按不可变证据确认。替换 Round6 对新关联同版本回复的错误假设，不弱化旧请求/广播保护。
- 会话所有权：恢复时捕获、成功保存时认领；注销不重新 load 猜测归属。按共享 vault/存储作用域串行化变更、最新登录代次阻断迟到交换；内置同步 compare-and-clear 保留。异步扩展适配器共享同一 vault 或声明同一 backing store/key；独立 JS realm 的任意异步外部写入不由本地队列保证。
- 启动：最后场景等待结束后重新读取房间/公平阶段，必要时同步私有投影并终查成员、版本、hand，空房发现同样重新核对；并行同房刷新按读代次丢弃过期结果。阶段证明只能覆盖最后一次服务端读取，不能作为永久租约。只读失败显式失败，不能产生新写命令或清除 UNKNOWN。
- 边界：共享 TS 会话、斗地主 shell、Cocos scene/Vue auth 分别拥有各自逻辑；无 Go、协议、数据库、缓存头或集群变更。额外读取只在启动末次门禁，不新增轮询；一次同步请求及单 vault 有序写入，网络交换不占用队列。回滚原客户端产物。新增受控异步测试，实际 Go/Cocos 和真实设备结论待独立验证；按用户要求不调查手机黑屏。

第二遍关联审计：成员重入/reconnect 的 active-hand 读取在最终 membership 后重新检查发牌前阶段并在最后异步同步后核验成员/room version；不改变已经确认的持久化新成员 Join 回执。Cocos 对自动 timeout 通知主动发起有界只读同步，旧选择/重复通知错误反馈受代次约束。Web host 恢复尝试独立代次保护旧 finally/catch；微信解绑/换绑用 session.discardLocal() 仅清除自己的本地令牌，已由 Go 撤销的 token 不重复 revoke，旧完成不覆盖新账号界面。补充实际 Controller 双入口跨阶段、关房和会话本地 detach 测试，以及最后场景门禁结构约束。仍需真实 Cocos/Go 独立复验。

## Round 8 系统性改造（用户授权；待 CI 与独立验收）
- 范围修订：用户授权为解决两个 P2 重构完成语义并评估服务端接口。新增向后兼容 room v1 的成员限定 activeHandAuthority（handId、durable version、phase、fairnessMode、本人 revealed），不改命令协议、数据库 schema、秘密、存储、缓存策略；旧客户端忽略新增字段，新客户端严格恢复要求新 API，先升级 API 再升级 Web，回滚顺序相反。Go 业务模块拥有权威投影，生产 MySQL go-zero 单语句 PK JOIN 同一一致读点取得 room、active hand、actor seq；不锁定写入、不将 durable version 当 live version。非成员不返回证明，缺失/错配 fail closed。旧归档授权保持。
- 共享 Controller.confirmRoomAuthority 由 join/reconnect/场景恢复/最后启动门禁消费同一证明；若观察到已发牌或实时 deal 提示，必须有授权私有投影，必要时一次同步和一次重新读取，最多两轮、超出显式失败。成功代表服务端读取点，不是永久租约；自动发牌提交及追踪成功通过已有账号私有 WSS 通道发布 hand.dealt 提示/快照。无新增轮询、无全局锁、无跨模块数据访问、无额外写命令；失联或推送遗漏仍走恢复。
- Session 登录/恢复返回仅内存的 operation-owned grant 或 null，旧完成不再产生连接许可；Controller 在首个 await 前认领启动代次，三个身份入口共用 grant+连接完成门禁，已建立身份使用新 Controller 换号以隔离业务状态；Shell 显式返回连接是否完成。保持 UNKNOWN 精确身份与已确认 durable receipt。单 realm vault 所有权及双 realm compare-and-clear 原边界保持。
- 回归覆盖旧成功丢弃与 B 连接/待确认命令保留、同 room version 的阶段推进、Go 成员限定证明和独立手牌版本、协议校验、自动发牌通知、已有成员双入口和 Bootstrap 共用门禁；保持 in_progress。真实 Cocos/Go、MySQL 一致读、完整业务与本产物缓存性能仍需 CI/Codex 证据；不调查手机黑屏。

第二遍自审与 CI 修正：同步更新旧 room discovery fixture 和构建断言，修复 TS 跨 await 收窄与 gofmt；补真实 MySQL 联合查询集成测试；连接绑定授权 grant 与连接代次，替换尚未建立身份的登录时立即取消旧握手，即使新登录失败也不会遗留可提交连接；Controller 业务入口要求已建立身份。联合证明检查 durable version 单调、相同版本不得冲突，连接/房间/命令代次变化撤销证明；预发牌恢复转为 live 即使收到无关联广播仍须关联私有读取。

### Round 9 follow-up: session identity and UI ownership (development)

Owner: shared client session lifecycle, Cocos composition and Web host. A current grant, not stored credential presence, authorizes client business continuations. Pending login blocks restoration of the previous stored identity in the same logical scope. Pre-authentication provider binding captures its original attempt; cleanup retains only the previously owned credential. Go authorization and wire/schema contracts remain unchanged. Constant-time checks and bounded existing HTTP/SDK deadlines, no additional normal-path network calls. Same-realm coordination only; cross-realm storage and already dispatched server operations are not cancellation leases. Regression and CI precede independent Round10 acceptance; no deployment in this change.

### Round 10 follow-up: owned operation lifetime (development)

Owner: platform-neutral game-client-core operation scopes and session coordination; Cocos owns navigation and provider operation presentation; Vue owns its browser auth lifecycle. Contract: local cancellation immediately releases only the matching pending owner, and late completion cannot release a replacement. Bounded portable waits for SDK operations; no assumption that a dispatched server mutation can be undone. Provider confirmation is non-cancellable once its final write is dispatched; show a pending-result notice instead. Pre-dispatch cancel permits a fresh operation. Preserve the serialized vault write queue until actual writes settle, and preserve existing command UNKNOWN contracts. No server, database, wire or cache changes, no extra steady-state network calls. Same-realm guarantee, platform callbacks may remain outstanding but inert; independent acceptance and CI required, no deployment during development.

### Round 11 follow-up: exception-isolated teardown (development)

Owner: shared client operation/transport cleanup; scene and browser composition own UI teardown. Contract: synchronous, bounded cleanup steps run independently after local ownership is invalidated; failures are counted and reported with a static non-secret diagnostic, never substitute for the original business outcome. OperationScope exposes cleanupFailures; cancellation/finish/abandon remain idempotent and non-throwing for cleanup callback failures. No server rollback is inferred. Native unregistration/close is best effort; failed native removal cannot guarantee physical release, but late callbacks must be inert. Preserve command UNKNOWN and existing protocol/session contracts. No new normal-path requests, timers or schema changes. Tests cover throwing release/listener/diagnostic, nested waits, registration and unregister faults, old ownership and Creator ES5 output. CI and independent acceptance required before deployment.
