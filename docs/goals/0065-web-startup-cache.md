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
