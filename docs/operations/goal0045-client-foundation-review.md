# Goal 0045 客户端基础阶段性验收

日期：2026-10-31。结论：**已完成源码审查/修复、49 项自动化测试、Creator 3.8.8 原生三目标构建、真实 Web 场景运行与适配器通信验证；小游戏 SDK/真机资格尚未完成。** Goal 0045 保持 `in_progress`，不归档、不恢复空白模板。本报告不表示完整斗地主客户端或三个发布目标已经可用。

## 范围、提交和责任

这是用户明确授权的阶段性独立验收与修复，保留 ChatGPT 的模块设计和现有 Go 服务。审查了 Goal 0045 从 `415ce34` 到 `1642121f01b4d94f1de90a282d461522ff934549` 的客户端提交历史、累积实现及最新重连修正；不是只依据提交标题或原来的五个测试作结论。

| 提交 | 内容 |
| --- | --- |
| `1642121f01b4d94f1de90a282d461522ff934549` | 本次接收的 ChatGPT 基线；只重置重连前状态，仍存在下述异步生命周期缺陷 |
| `1af0c07ddb5d94c4a49866d10a094b623cd518a4` | 连接/SDK/状态修复、最小场景与 Meta、严格编译和回归测试、CI job |
| `fb9768b20635d49dc1c41e54cbf5afe6b1089a43` | 应用命令两分钟 TTL 边界及测试；修正文档中错误的 WSS 请求版本名 |
| `0fc7ebb9a31f51ddf1830c6f05021eac474e3a09` | 同版本完整快照确认重连；HTTP/握手默认 15 秒超时、外部取消和定时器清理 |
| `fead97d` / `7bdad4f` / `38e46c5` | Windows Creator 实际构建/浏览器验收工作流；修复 Windows npm 安装工作目录 |
| `1ab48ea` | 真实 Creator 构建发现的本地脚本 .js 路径问题：改用 .ts，Node 编译重写为 .js |
| `a5b5ff1` | 原生生成配置类型检查、双小游戏导出、下载超时和重试 |
| `8d6e3fe` | Creator 生成配置下的包 exports/标准库解析；修正浏览器 Promise 等待假阳性 |
| `7677218` | 响应式 Web 构建模板与真实浏览器画布边界断言 |
| `087d3d3` / `96d697c` | 释放动态 UI；横屏小游戏构建配置/产物断言；重挂载和父节点销毁顺序检查 |
| `0acc8ad` | 实测 CDN 约 2.5 MB/s 后调长下载上限、断点续传并缓存已校验的官方归档 |
| `b1b55f0` | Web 子窗口使用 screen.windowSize 同步物理渲染缓冲区及 DPR；加入真实显示比例检查 |
| `7da85a9` | 竖屏字号/文本区域适配；检查实际渲染字号，保留横屏布局 |

本阶段的源码检查、测试执行、修复和修复后的自审由当前 Codex 执行；GitHub Actions 负责仓库 CI。修复代码的第二轮检查称为**自审**，不冒充另一位独立审查者。最后的报告提交只更新验收文档、模板来源声明和 Working State；其 SHA/Actions 状态在交接消息中给出。

变更属于客户端集成：可复用技术端口在 game-client-core，游戏状态在 doudizhu-client，渲染和组合在 Cocos 工程。没有修改 Go 后端、数据库结构、部署配置、生产账号权限或牌局数据；真实通信仅使用既有只读验收账号登录、心跳及退出本次会话。

## 实际目录和导入边界

```text
clients/packages/game-client-core/
  src/index.ts          # 端口、熵检查、URL 凭据守卫、ConnectionManager
  src/socket.ts         # Web/mini 共享的适配器连接生命周期
  src/network-timeout.ts # 明确的网络等待期限策略
  src/web.ts            # 浏览器 HTTP/WSS/随机源/存储/生命周期/复制邀请
  src/minigame.ts       # 注入 SDK 的 HTTP/SocketTask/随机源适配
  src/platform.ts       # 小游戏能力组合，显式处理微信随机源与分享差异
  test/                 # 真实模块 + 可控 Fake SDK、导入边界、ESM exports
clients/packages/doudizhu-client/
  src/index.ts          # HandStore、命令守卫/不可变工厂、消息路由
  test/state.test.mjs   # 状态/协议/后端阶段枚举对照
clients/packages/secure-envelope/  # 原 HPKE 实现，未修改
clients/fair-doudizhu-cocos/
  assets/scenes/Entry.scene       # Scene/Canvas/正交相机/Bootstrap
  assets/scripts/bootstrap/GameBootstrap.ts
  assets/scripts/game/GameClientShell.ts
  assets/scripts/platform/{createPlatform,platform-target}.ts
  assets/scripts/foundation/crypto/WechatRandomSource.ts
  assets/**/*.meta                # 稳定 UUID；不提交编辑器缓存
  build-templates/web-desktop/index.ejs # 随窗口尺寸布局的 Web 宿主模板
  test/{assets,shell}.test.mjs
  tsconfig.static.json            # 官方 3.8.8 声明检查全部脚本
  tsconfig.shell.json             # 不含 cc 的 Node ESM 编译
scripts/verify-client-foundation.sh
scripts/verify-cocos-{runtime.ps1,browser.cjs}
.github/workflows/doudizhu.yml    # 新增 client_foundation job
.github/workflows/cocos-client-smoke.yml # 原生 Windows 导入/三目标构建/浏览器
```

检查实际导入，而非仅看目录名：根端口/连接管理和 doudizhu-client 不导入 `cc`、`wx`、`tt`、DOM 或平台适配器，不访问浏览器运行时全局。`AbortSignal` 只作为请求端口类型使用。浏览器全局只位于显式 `/web` 适配入口，SDK 经 `/minigame`、`/platform` 注入，目标检测限于 Cocos 组合根。没有运行时循环依赖。移除了无人引用、与真实 GamePlatform 重复的旧 `Platform.ts` 占位接口。

NodeNext/严格模式、声明生成、四个 ESM exports 以及 Cocos 本地 `file:` 依赖均有检查；先构建共享包的 `dist`，再安装/检查 Cocos 工程。未引入另一套 HPKE 或大型客户端框架。新增的 `@cocos/creator-types@3.8.8` 仅为开发期官方声明，不能替代引擎执行；[官方说明](https://github.com/cocos/creator-types)也明确区分两者。

## 问题、严重程度和修复

| 严重程度 | 基线问题 | 已完成修复/证据 |
| --- | --- | --- |
| 高 | MiniSocketTransport 同步 success 读取未赋值 socket，同步 fail 读取暂时性死区 onError；把任务创建成功当作握手成功 | 等待 SocketTask.onOpen；创建失败先安全关闭返回任务；回调初始化顺序修正。新增测试在修复前实际失败 |
| 高 | 连接前 error/close 无完整收尾；连接中关闭 Manager 无法及时取消；旧连接消息仍能进入新消费者 | 共享生命周期收尾、可移植取消订阅、失败/取消关闭、代际守卫；同步/异步失败与旧回调测试通过 |
| 高 | 微信安全随机源错误假定 wx.getRandomValues；旧包装还隐式读取全局 wx | 使用注入的 getUserCryptoManager().getRandomValues，校验长度，复用 MiniRandom 作为 HPKE 熵端口；缺失即拒绝 |
| 高 | 仓库无可供打开的 Scene/Meta；strict noImplicitOverride 下 Bootstrap 两个生命周期方法不通过 | 提交 Entry.scene、相机/Canvas/脚本 UUID 绑定与 Meta；补 override，官方 3.8.8 类型检查通过。Windows 真实导入后发现本地脚本 .js 路径不可解析，已改为 .ts 并由独立 Node 构建重写扩展名；最终原生导入/构建通过 |
| 中 | Shell 重复初始化积累监听；旧 connect 完成/失败在新连接或销毁后修改状态 | 初始化幂等、连接代际检查、dispose 清理；测试覆盖旧成功/失败和销毁后完成 |
| 中 | hand.sync 未带 v，真实 Go bridge 将拒绝请求；长 handId 还会生成过长的关联 ID | 发送 doudizhu-realtime-sync-v1，校验 handId/knownVersion，用短递增关联 ID；断言完整发包结构 |
| 中 | 同版本快照可替换状态；调用者可修改已确认视图；终局仍可提交；缺 WAITING_PUBLIC_BEACON | 忽略相同/旧版本，复制并冻结快照，终局禁操作；直接对照 Go owner 全部阶段的回归测试 |
| 中 | 命令守卫允许未知名称、错误 aggregateType、数组 payload、伪造 actor 字段和宽松时间格式 | 增加名称/聚合匹配、对象/身份/UTC 毫秒边界检查，创建时冻结副本；TTL 与应用默认两分钟一致，边界及越界测试通过 |
| 中 | 同版本完整快照在重连后被去重，却未完成同步状态转换 | 保留原快照，同时将匹配 handId/version 的完整确认用于结束同步；新回归修复前失败，修复后通过 |
| 中 | HTTP/握手在 SDK 不回调时可能永久等待 | 增加默认 15 秒、可配置 timeoutMs 的期限，主动 abort/close；超时、外部取消及非法期限不创建原生任务的测试通过 |
| 中 | Router 未校验协议版本；clear 后旧 unsubscribe 可删除新监听 | 检查当前真实结果/事件版本；卸载只删除自己所属的 listener set |
| 中 | SDK HTTP 忽略 AbortSignal，清理/失败信息不一致；缺失存储能力被当作空会话 | 接入 RequestTask.abort，处理创建期间同步 abort、移除监听、拒绝缺失能力与空白 token；成功/失败/取消测试通过 |
| 中 | 微信与抖音分享结果被同样立即视为成功 | 微信仅表示调用分享 UI；抖音等待成功/失败/取消回调，不声称邀请已送达；不支持能力返回 false |
| 中 | Creator 使用 .js 导入本地 .ts 资源，实际构建失败；生成配置无法解析 package exports/缺少新标准库 | 本地资源导入 .ts，Node 构建重写 .js；工程指定 Bundler/ESNext 解析、ES2022/DOM 类型库，strict 保留；只跳过引擎外部 .d.ts 自身一致性检查。原生生成配置检查已在 8d6e3fe 通过 |
| 中 | 默认 Web 桌面模板固定 1280×960，节点逻辑边界断言会漏掉窄屏裁切 | 新增 viewport 大小的 Web 模板，检查实际 DOM canvas 边界；旧真实产物新增断言实际失败，截图确认标题被裁掉 |
| 中 | 响应式 CSS 改变显示尺寸但未同步 Cocos framebuffer，首轮绿色检查仍漏掉文字拉伸 | 自审逐张截图发现；使用 3.8.8 公开 screen.windowSize/屏幕 DPR API，限定 Web 宿主；保留原生生成脚本。新增实际 canvas 像素尺寸/显示比例检查在旧真实产物失败 |
| 中 | 竖屏虽然不再裁切/变形，但固定设计分辨率下状态字号过小 | 增大竖屏标题/状态字号和文本区域，不改横屏布局。实际 Label.actualFontSize × view scale / DPR 检查（标题至少 18 CSS px、状态至少 12 CSS px）旧产物失败；不是只读配置字号 |
| 中 | 单独销毁 Bootstrap 后仍残留 SafeContent/UI | 销毁其拥有的节点并清空引用；检查节点有效性兼容父节点先销毁子节点。真实引擎旧产物实际失败 contentCleared=false，保留回归并检查重挂载、两种销毁及监听释放 |
| 中 | 首次小游戏默认构建方向为 portrait，违反横屏入口要求 | 使用类型正确的 JSON configPath，分别指定 packages.<target>.orientation=landscape；检查生成 game.json，不只检查进程成功码 |
| 文档 | Realtime 文档写成不存在的 doudizhu-command-request-v1 | 改为 Go dispatcher 当前真实 doudizhu-api-command-v1，并明确与应用命令不同 |

最初新增的五个连接测试在未修复基线全部失败：同步 success、同步 fail、早期错误清理、连接中关闭、旧消息污染。修复后通过。自审另发现并修复了注册 close 监听时同步关闭的收尾、clear 后旧卸载误删新订阅、跨手牌事件影响当前手牌，以及 beacon-wait 阶段遗漏。未删掉失败断言来得到通过结果。

## 构建、测试和执行证据

本地 Node **22.23.4**；资源较重的步骤顺序执行。可复现入口：

```bash
bash scripts/verify-client-foundation.sh
# 现有 HPKE 包完整测试另行执行过：
npm --prefix clients/packages/secure-envelope test
```

| 验证层次 | 实际结果 | 证明范围 |
| --- | --- | --- |
| game-client-core 安装/typecheck/build/Node tests | 26/26 通过 | 实际适配模块、熵失败、URL、HTTP、SocketTask、取消、旧事件、能力缺失、生命周期/存储/分享、导入边界与 ESM |
| doudizhu-client 安装/typecheck/build/Node tests | 7/7 通过 | 快照版本/不可变、重连、终局、命令边界、版本路由、监听清理、Go 阶段枚举 |
| secure-envelope 原有测试 | 9/9 通过，源码未改 | 原 HPKE/清单、异步熵预取等既有测试；不是小游戏真机密码学证明 |
| Cocos 全部脚本静态 typecheck | 通过 | 官方 3.8.8 声明、strict/noImplicitOverride；不是 Creator 编译器运行 |
| Shell ESM 编译/测试和 Scene/Meta 引用检查 | 7/7 通过 | 真实 Shell、取消/重连/销毁、sync 结构、notModified、场景引用一致性；不是引擎加载 |
| 真实 Chromium + HTTPS/WSS + Go 服务 | 通过 | 见下文 Web 通信探针；不包含牌局或 Cocos 渲染 |
| Creator 3.8.8 原生 CLI 导入/三目标导出 | **通过** | Web、微信、抖音均退出 36 且生成入口产物；两个 game.json 明确 landscape |
| Creator 实际生成配置的严格 tsc | **通过** | 保留 strict/noImplicitOverride，解析真实本地包 exports；不是伪造 temp 配置 |
| 实际构建 Web 场景/Chromium | **通过** | 1280×720、844×390、390×844；实际 canvas/渲染比例、字号、节点边界、销毁/重挂载/父节点销毁、监听清理；逐张自审截图 |
| Creator 交互式 GUI 编辑器预览 | **未取得合格结果** | Linux Wine 失败/主动停止；Windows CI 执行 CLI 导入构建，随后实际 Web 引擎运行，未冒充交互式编辑器操作 |
| 微信/抖音开发者工具、真机、平台审核 | **工具安装并尝试启动，模拟器/真机未通过** | Wine 下未取得可用工具界面/小游戏执行证据，缺平台应用配置和设备；不标记通过 |

CI 增加 `Fair Doudizhu / client_foundation`，执行同一顺序脚本；原 manifest_client 继续运行 HPKE/签名清单测试和互操作。网络等待修复检查点 `0fc7ebb9a31f51ddf1830c6f05021eac474e3a09` 的 [CI 41193878945](https://github.com/oublie6/awesome-zero-platform/actions/runs/41193878945) 和 [Fair Doudizhu 41193878962](https://github.com/oublie6/awesome-zero-platform/actions/runs/41193878962) 均已成功，包括新增 client_foundation、现有 manifest_client/互操作、单元/竞态/集成及运行时门禁。最终报告提交的 Actions 另在推送后核对，并在交接消息中给出。

真实 Web 通信使用本仓库实际 `WebHttp`、`WebSocketTransport`、`WebRandom`、`WebSessionVault`，由 Chromium 加载编译后的 ESM。既有只读测试身份经真实 Nginx/Go 登录；TLS/WSS 在本机临时自签名桥终止，再经端口转发到既有服务，没有修改集群 TLS、Origin 或授权配置。仅此隔离浏览器上下文忽略临时证书错误；**不能据此宣称生产证书链已经验证**。

真实 WSS 收到 `system.hello`，发送 `system.ping` 后收到对应 `system.pong`；错误 JWT 握手被拒绝。两个捕获到的实际 socket URL 都是 `wss://172.17.0.1:18445/ws`，无 token；合法令牌只走 bearer subprotocol。安全随机源、sessionStorage 存取/清除通过；pageErrors 为空。无效令牌检查是预期负例，不将浏览器的预期握手失败输出冒充“零控制台错误”。未向服务发送牌局命令。

Web 探针在增加网络超时后针对网络等待修复源码 `0fc7ebb` 重跑；协议和 Shell 的可控测试另外执行，不混同为现场牌局验证。报告、源文件哈希、探针脚本与本地日志保存在 `/root/.config/awesome-zero-platform/goal0045-evidence/`，目录 0700、文件 0600，不入 Git、不含密码/token 值。`web-runtime.json` SHA256：`ed57af7ec9caca653d70ce279f1331e09404175908cd7cebf3a4799525a8da89`。测试会话已退出，浏览器/转发/TLS 桥已停止，临时私钥和证书已删除；保留已有缓存。

## Web / 微信 / 抖音能力矩阵

“文档核对”和 Fake Adapter 测试不代表开发者工具或真机通过。

| 能力 | Web | 微信小游戏 | 大陆抖音小游戏 |
| --- | --- | --- | --- |
| 登录 | 浏览器 HTTP 测试账号登录通过；正式平台身份 UI 未实现 | wx.login 临时 code；后端换证/绑定待 Goal 0046 | tt.login 可能只返回 anonymousCode；适配器拒绝把它当已认证 code，后端换证待实现 |
| HTTP/错误/取消 | 真实登录 + 状态/AbortSignal 自动化通过 | wx.request/RequestTask 注入；可控 SDK 测试通过，未跑平台 | tt.request 要求 HTTPS/合法域名；测试了状态与 abort，未跑平台 |
| SocketTask/失败/关闭 | 真实 WebSocket、错误认证通过；取消/旧事件测试通过 | connectSocket 返回任务；等 onOpen，不能等 success；不假定存在所有 off 方法 | 独立核对 SocketTask 文档；同样等 onOpen；不假定 off 方法，关闭后回调失效 |
| 握手认证 | ['bearer', accessToken] 与当前服务兼容，真实 WSS 通过 | 文档列出 protocols/header；真机是否正确保留/协商 bearer 待验 | 文档列出 protocols/header 并要求服务返回所选协议；真机待验，不把 Header 能力从微信推断过来 |
| 安全随机源 | HTTPS 浏览器 crypto.getRandomValues 实际通过 | UserCryptoManager.getRandomValues，文档最低 2.17.3；注入/长度/缺失测试通过，真机未验 | **未确认 Mini Game 安全熵 API**；mini-app 同名方法不能作证；缺失即拒绝，不用 Math.random |
| 存储 | sessionStorage 实际存取/清除通过，按标签页隔离 | 同步存储接口；可控 SDK 测试，未实机；不是加密安全存储 | 同步存储端口/缺失能力测试；实际 SDK/生命周期组合仍待工具验证 |
| 前后台 | document visibilitychange 注册/卸载自动化通过；未测移动 OS 挂起 | onShow/onHide 和对应卸载需同时存在，否则明确失败；工具/真机待验 | 同样进行能力检查而不假定完全一致；工具/真机待验 |
| 分享 | Clipboard 能力/缺失单元测试；不是系统分享送达证明 | 调用 UI 不等于分享送达；素材/用户手势和真机待验 | 使用 success/fail/cancel；PC 宿主有不支持分享的明确差异，失败不能当成功；工具/真机待验 |

主要官方依据（核对日期同本报告）：

- 微信：[connectSocket](https://developers.weixin.qq.com/minigame/dev/api/network/websocket/wx.connectSocket.html)、[SocketTask](https://developers.weixin.qq.com/minigame/dev/api/network/websocket/SocketTask.html)、[UserCryptoManager.getRandomValues](https://developers.weixin.qq.com/minigame/dev/api/base/crypto/UserCryptoManager.getRandomValues.html)、[login](https://developers.weixin.qq.com/minigame/dev/api/open-api/login/wx.login.html)、[request](https://developers.weixin.qq.com/minigame/dev/api/network/request/wx.request.html)、[生命周期](https://developers.weixin.qq.com/minigame/dev/api/base/app/life-cycle/wx.onShow.html)、[存储](https://developers.weixin.qq.com/minigame/dev/api/storage/wx.getStorageSync.html)、[分享](https://developers.weixin.qq.com/minigame/dev/api/share/wx.shareAppMessage.html)。微信站部分页面不能经浏览工具解析时，以直接 HTTPS 下载的官方正文核对随机源/SocketTask；未用社区镜像当 SDK 依据。
- 抖音：[小游戏 connectSocket](https://partner.open-douyin.com/docs/resource/zh-CN/mini-game/develop/api/javascript-api/network/websocket/tt-connect-socket)、[小游戏 SocketTask](https://partner.open-douyin.com/docs/resource/zh-CN/mini-game/develop/api/javascript-api/network/websocket/socket-task/socket-task)、[小游戏 login](https://partner.open-douyin.com/docs/resource/zh-CN/mini-game/develop/api/javascript-api/open-capacity/log-in/tt-login)、[小游戏 request](https://partner.open-douyin.com/docs/resource/zh-CN/mini-game/develop/api/javascript-api/network/initiate-a-request/tt-request)、[小游戏分享](https://developer.open-douyin.com/docs/resource/zh-CN/mini-game/develop/api/javascript-api/open-capacity/retweet/tt-share-app-message)、[PC 差异](https://developer.open-douyin.com/docs/resource/zh-CN/mini-game/develop/guide/open-ability/pc-game/game-tutorial)。[getRandomValues 的已找到页面](https://developer.open-douyin.com/docs/resource/zh-CN/mini-app/develop/api/device/crypto/tt-get-random-valus)属于小程序，不据此认定小游戏支持。部分旧导航链接已失效，未将空正文页面当作签名验证成功。

如果某宿主不能安全携带当前 subprotocol，停止该目标的连接资格确认。未来短期 socket ticket 需求：服务端签发、单用途、短有效期、一次性消费，绑定当前账户/会话和连接受众，可随会话撤销，明确重放/超时/错误语义；评审后另行修改后端。当前没有实现或绕过该协议，也没有长期 JWT URL 回退。

## 当前协议兼容性与未完成项

依据：`server/apps/app-api/internal/doudizhuapi/{dispatcher,realtime}.go`、`server/business/doudizhu/application/{service,reveal_codec}.go`、实际 owner 阶段常量和本 Goal 引用文档。

| 项目 | 本阶段结果 / 后续要求 |
| --- | --- |
| durable Command Envelope | 守卫针对 fair-doudizhu-command-v1；限定已知公开命令、聚合、版本、身份字段和 UTC 毫秒时间。载荷业务合法性仍由后端判断；不是完整 wire DTO |
| 实际发送命令 | 当前 Go HTTP/WSS 要 doudizhu-api-command-v1 / requestId / type；客户端工厂不是此对象。尚无游戏命令发送链；Goal 0046 必须显式映射，不能直接 JSON.stringify durable envelope 当 wire 请求 |
| commandId/clientSeq/重试 | 调用者仍负责安全唯一 ID 和每账户/聚合的单调序号；工厂冻结副本，方便原 ID/序号/载荷重试，但尚无分配器或重试协调器，不声称完成 |
| expectedVersion | 创建为 0，其他命令为正整数；旧/相同快照不能回退或改写 store；大于 JS 安全整数的值拒绝而非悄悄取整 |
| hand.sync | 已补 v=doudizhu-realtime-sync-v1，handId/knownVersion 与实际 Go 结构一致；测试逐字段断言 |
| notModified | 不覆盖当前快照；仅当前 handId/knownVersion 都匹配且正在同步时恢复可提交状态。首次无快照不会凭 notModified 生成状态 |
| command.result / hand.changed | 校验真实结果/事件版本；通知不生成手牌，不以 accepted 结果伪造私有快照；当前手牌变更先停止操作等待同步 |
| hand.snapshot | 只通知接收到已版本化 payload；完整 view.payload 到 HandSnapshot 的严格映射尚待 Goal 0046。测试证明不能直接强转 wire view 注入 store |
| hand.evidence / error | 终局证据通知使当前手牌操作暂停；完整证据只读模型和错误恢复协调待实现；其他 hand 的通知不改变当前手牌 |
| 揭示 HPKE/AAD | 复用旧包，但真实 hand.reveal 尚未端到端验证。Go dispatcher 的 envelope/v 外形与应用 secureEnvelope/version 不同，且 dispatcher 在接收时生成 AAD 绑定的 issue/expiry。这属于静态兼容性风险，未做真实 reveal 复现；客户端如何获得完全一致的 AAD 时间上下文必须先由 ChatGPT 澄清/验证，不能自行猜时钟；本阶段未改 Go 协议 |
| 断线恢复 | 正常/失败/旧连接/销毁状态转换有测试；尚无完整退避重连、掉线恢复、房间/牌局协调或服务进程崩溃恢复承诺 |

## Creator 实际可运行状态与交接

当前工程已有最小宿主场景，不再要求开发者手动创建场景才能引用 Bootstrap；采用代码创建状态 UI，并使用引擎 SafeArea、Label 换行/缩字及 resize 监听清理。场景结构参考官方 [3.8.8 2D 模板](https://github.com/cocos/cocos-engine/blob/v3.8.8/editor/assets/default_file_content/scene/scene-2d.scene)，静态测试检查相机、脚本 UUID、引用和 Meta 唯一性。

用户随后要求“安装环境测试”，因此继续安装并实际运行，不能沿用“工具未安装”的旧结论。工具均在数据盘 `/data/awesome-zero-platform-tools/goal0045/`，未提交二进制或缓存：

| 工具 | 安装来源/版本 | 实际执行结果 |
| --- | --- | --- |
| Cocos Creator | 官方 Windows 3.8.8 `win-121518`；归档 SHA256 `e365030aa4f24b515f499cf093cd86fdf38a0f763b5fbcacb20e96253e0fcc0b` | Wine 6 启动崩溃；Wine 11 进入构建管理器，但出现登录服务超时且本机可用内存降至约 370 MiB，主动停止并恢复至约 4.3 GiB。此超时不能单独证明必须登录 |
| Creator 原生环境 | GitHub Actions windows-2022，官方相同归档并校验 SHA256 | `41203283718` 真实导入了场景和脚本，构建退出 34：`GameClientShell.js` / `platform-target.js` 不能映射到 .ts。修复 `1ab48ea` 使用真实 .ts 路径，独立 Node ESM 构建启用 rewriteRelativeImportExtensions；本地类型检查和 7 项测试通过，`41204313900` 的三个目标均返回 36 并生成真实产物；随后发现默认 Web 裁切、小游戏默认竖屏和生成配置类型解析问题，已继续修复；最终 7da85a9 的检查 41208033225 已通过（96d697c 那轮在下载前置步骤超时，见下文） |
| 抖音开发者工具 | 官方 Windows 4.5.7 | Wine 11 启动并加载主程序，更新服务确认当前版本；出现 Windows COM/图形/网络兼容性错误。补齐 EGL 后限内存重试，CDP 未能连接且显示未取得可用 UI。没有导入小游戏、调用 tt SDK 或真机验证 |
| 微信开发者工具 | 官方备份稳定 Windows x64 2.01.2510260；下载 MD5 与官方 `cc80edf59019e01341399699c4247ec2` 一致；不称最新版 | Wine 11 启动 NW.js 窗口，但捕获界面为空灰色，日志有 Windows API/图像解码错误；未取得可用模拟器或 wx SDK 执行结果 |

安装 WineHQ 11、Xvfb、中文字体和图形库；所有 IDE 按顺序运行，后续小游戏启动置于 MemoryMax=3G、MemorySwapMax=0、180–300 秒运行上限的独立 systemd 单元。未启用 swap、重启服务或更改 Kubernetes/后端。官方安装包只提供 Windows/macOS 版本，Wine 尝试不等于官方支持环境资格。

自定义 Web 模板遵循官方 [3.8 构建模板机制](https://docs.cocos.com/creator/3.8/manual/zh/editor/publish/custom-project-build-template.html)，保留引擎生成脚本注入，不在测试中伪造 cc。小游戏的横屏参数使用 [3.8 命令行配置](https://docs.cocos.com/creator/3.8/manual/zh/editor/publish/publish-in-command-line.html) 的 packages 字段；使用编辑器默认测试 AppID，只证明导出，不能据此认定平台账号/审核/SDK 已通过。

官方安装依据：[Creator CLI/退出码](https://docs.cocos.com/creator/3.8/manual/en/editor/publish/publish-in-command-line.html)、[Creator 版本服务](https://creator-api.cocos.com/api/cocoshub/editor_version_list?lang=zh&ver=2.3.2)、[微信官方备份下载](https://developers.weixin.qq.com/miniprogram/dev/devtools/download_backup.html)、[抖音小游戏开发工具](https://partner.open-douyin.com/docs/resource/zh-CN/mini-game/develop/dev-tools/developer-instrument-update-and-download)。安装/启动日志位于数据盘 `logs/`；截图及原生 Windows 构建日志保留在上述私有证据目录。黑色/灰色启动截图不作场景或模拟器通过证据。

运行检查本身也经过纠错：原先 async waitForFunction 返回 Promise，被轮询当作已就绪，随后读取空场景；现先加载真实 cc 模块，再同步轮询实际 Canvas/SafeContent/GameStatus。又通过逐张截图发现逻辑节点边界检查漏掉固定 DOM 画布，加入真实 viewport 边界。此前宽松运行检查虽然返回 passed，**不作为最终布局验收证据**。单独销毁的旧产物验证为隔离生命周期问题暂时省略 viewport 断言，只记失败证据，不记资格通过。

新增 `Cocos Creator Smoke` 工作流与 `scripts/verify-cocos-runtime.ps1`、`scripts/verify-cocos-browser.cjs`：Windows 原生实际导入/构建必须退出 36 且存在 Web 产物；随后用真实 Chromium 加载引擎、检查 Bootstrap/UI、三个尺寸和销毁。失败必须保留日志，不能将 ordinary exit 0 或静态声明检查当成功。生成配置下的 tsc 检查和微信/抖音产物导出也已加入，按序执行；原生运行 `41206496385` 在 `0acc8ad` 上三个目标导出、严格生成配置检查及当时的浏览器断言均通过，两个 game.json 确认为 landscape。但逐张自审截图发现 CSS 拉伸，而当时的断言只验证画布边界，**该次不作为最终布局资格通过**。继续新增渲染比例、实际字号检查：未修改的该次原生产物复现比例失败；同一真实引擎/场景加候选宿主修复脚本的本地预验证复现字号失败，后者不是 Creator 重建资格。最新 `7da85a9cf6b4cc3c0404dbecdd40b2e7d6cc2539` 的完整原生验证 `41208033225` 已通过，最终三张截图均已逐张复核。`41205713103` 在代码执行前因 240 秒下载上限过短而失败：第二次传输收到约 611 MiB/980 MiB，速率约 2.5 MB/s；不是编辑器或客户端失败。修正为每次最多 600 秒、断点续传且只重试一次，使用官方 cache restore/save，SHA256 每次解压前仍校验；修复失败不再重复下载已验证归档。

最终源码验证为 `7da85a9cf6b4cc3c0404dbecdd40b2e7d6cc2539`：[Cocos Creator Smoke 41208033225](https://github.com/oublie6/awesome-zero-platform/actions/runs/41208033225)、[CI 41208033131](https://github.com/oublie6/awesome-zero-platform/actions/runs/41208033131)、[Fair Doudizhu 41208033122](https://github.com/oublie6/awesome-zero-platform/actions/runs/41208033122) 全部成功。Windows job `124557119115` 使用缓存恢复且每次校验官方 Creator 归档；真实生成配置 tsc 通过，三个目标分别退出 36。构建日志保留 `merge_dep` 默认压缩选项回退、小游戏配置无版本号跳过迁移等非阻塞提示，未宣称编辑器所有日志零警告。

最终 browser-runtime.json 为 passed，errors/warnings/failed requests 均为空。实际 canvas 显示与缓冲区尺寸逐一匹配：1280×720、844×390、390×844；标题/状态实际 CSS 字号分别约 43/23、28.35/15.17、22.24/14.93，竖屏自动换行。单独销毁后 UI/两类 view 监听清除、Shell 断开；重新挂载只创建一个 SafeContent，父节点销毁后组件/子 UI/监听均清理。截图已进行独立于自动断言的第二遍自审；这是当前实现者的自审，不是另一位审查者。

[证据 artifact 13044679501](https://github.com/oublie6/awesome-zero-platform/actions/runs/41208033225/artifacts/13044679501) 含三目标 zip、构建日志/结果、配置、浏览器 JSON 和三张截图，保留至 2026-11-07。下载归档 SHA256 已核对 API digest：`43272cd7b6667083a5db2d16c33d0672ecd6b90f3cf76478a6347bcd4f963f0a`；browser-runtime.json SHA256 为 `c94f2815c3628237525cfe2b8bc689d6b9f384d5008734eac2d995fea6ca0250`。另保留私有本地副本 `/root/.config/awesome-zero-platform/goal0045-evidence/native-windows-41208033225/`（0700/0600），无凭据或二进制入 Git。

本机最终可用内存约 4.3 GiB，无 swap；根盘约 51%，数据盘约 64%，可用约 14 GiB。已停止全部本次 IDE/Wine/浏览器/转发进程，清理约 2.4 GiB 重复安装归档/解包中间文件/旧 Wine 前缀，保留数据盘已安装工具和证据。未进行全局镜像/缓存/PV 清理或服务重启。

交回 ChatGPT 的具体任务：

1. 复用已通过的 Creator **3.8.8** 原生 CLI/生成配置/三目标构建和 Web 入口路径。继续交互式编辑器预览、真实移动设备/刘海安全区/系统旋转和长期资源检查；不要再把已通过的导入/导出列为“编辑器未安装”。
2. 配置合法小游戏 app ID/域名和工具；分别实测 wx/tt 登录、WSS bearer 协商、存储/前后台/分享和熵。尤其确认抖音 **Mini Game** 密码学安全随机源，缺失时保持拒绝。不要仅依据本报告里的可控 SDK 测试放行。
3. 按 Goal 0046 实现有据可查的身份换证/会话、授权 HTTP、准确 wire DTO/结果/私有快照/终局映射、ID/序号与原请求重试，解决揭示 AAD 上下文可获得性；会话寿命、撤销/退出清理和小游戏持久存储的敏感信息策略也需明确；本阶段不展开完整牌桌或 Go 重构。
4. 最终三人真实牌局、房间/准备/公平性/出牌/结算属于既定后续目标，尚未开发/验收。本阶段不开放游戏生产流量；按用户要求，本次交接保持 Goal 0045，不归档、不重置。

新增 Web 宿主尺寸适配由 Cocos 3.8.8 的公开 API 完成，未读取/修改引擎私有状态，也没有伪造 cc。检查了 Game.once 在初始化前/后注册的真实语义、screen.windowSize 的物理像素单位/桌面子窗口限制、引擎 DPR 上限；Web 宿主 resize 监听在永久 pagehide 时清理，bfcache 暂存时保留。仅修改 HTML 模板脚本的本地预验证使用真实已构建引擎加提议脚本，**不替代原生 Creator 重建资格**。真机刘海/旋转/移动系统挂起和长期堆内存分析没有运行。

自审覆盖了边界输入、SDK 同步回调和抛错、close/error/cancel 先后、旧成功/失败/监听、重复消息和版本、不可变状态、身份字段、协议层次、秘密与 URL、启动/销毁、构建依赖/导出、资源范围及验证证据强度。修复后复跑受影响测试和严格类型检查；剩余不确定性均按上述运行层次列明，不以单元测试、声明文件或绿色 CI 替代 Creator/工具/真机结果。
