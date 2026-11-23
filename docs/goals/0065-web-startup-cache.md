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
