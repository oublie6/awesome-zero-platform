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
