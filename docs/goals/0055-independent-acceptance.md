# Goal 0055 — Cocos4 Linux 原生、微信真机与完整产品独立验收

日期：2026-11-08 UTC。执行者：Codex。固定产品候选 **`a4551b77d3751bf6afbdeeb14d8cde1be30fb803`**。

## 1. 结论与交回开发方事项

**整体产品验收 FAIL。需要 ChatGPT 修复客户端导出、微信转译和打包兼容性，再提交新的固定候选验收。**

Linux 原生导出、原始配置的模拟器场景、真实 SDK 能力、分段微信换证、原生 HPKE、真实 Go 三人完整局及限定范围的恢复检查通过。但默认产物无法通过微信开发预览的语法检查；启用工具 ES6 转译、压缩后虽能生成开发预览码，却把 BigInt 幂运算改为 `Math.pow`，导致同配置模拟器启动失败。用户实际使用荣耀手机扫码，停留在微信小游戏加载页，未进入产品首页。不能将构建成功、模拟器基线成功或 Go 完整局当作微信客户端完整交付。

本轮仅修改本报告及 current 的允许状态部分。没有修补产品、正式测试、协议、数据库 Schema、CI、生产配置或历史报告。Goal0054 的专项 PASS 保持原义；Goal0045/0046 继续 **`in_progress`**。C09/C10 保持 **DEFERRED**，没有执行生产上传、审核、发布或密钥迁移。

## 2. 版本、环境和证据边界

- clean `main` 执行 `git pull --ff-only`，候选及收尾同步结果均为上述 SHA；报告提交与受测产品 SHA 分开。
- 比较 Goal0054 产品 `fcca57d6f29b724cb193c3d652a1bd2a7cd230ca` 的完整差异：9 文件，新增菜单分页、Bootstrap 呈现逻辑、meta、对应测试、文档及配置；Go、协议包、Hand/Fairness 命令实现没有变化。
- 两版本 server tree 同为 **`06209f0fc5a8f9bb8a50682c69cc6f195f354e18`**。复核 34 项既有证据哈希，全部匹配；另绑定 8 项 Goal0054 直接证据。复用的是对应未变实现的原始证据，不是照搬历史 PASS/FAIL 标签。
- 已复核 AGENTS、原则 01–04、current、Goal0046 综合交付、Goal0054、交付缺口审计、Linux 构建文档、客户端 README/PLATFORM-VALIDATION、固定工具链及相关实现。
- 实际环境：Linux x86_64，约 7.75 GiB RAM，无 swap；Node **22.23.4**。重型构建顺序执行，systemd 限制构建 3 GiB、CPU 100%，未装 Wine、其他 OS 或新模拟器。
- 原生工具：`/data/awesome-zero-platform-tools/round7-cocos-cli`；官方 CLI revision **`f506fa3de403b48b4833334fa48dddd27bb26364`**，其 `packages/engine` revision **`d0a696cfe63fd8d19c70d7913ba11aa641c2e3c7`**。引擎包声明 `4.0.0-beta.0`，实际运行版本字符串为 `4.0.0`，不能将后者解释为正式稳定版。
- Linux 社区微信开发者工具 **2.02.2609240**，现有登录 profile，基础库 **3.17.5**。实际 Electron 是 Linux ELF；工具兼容层错误栈中的 `env: Windows` 不代表使用 Windows/Wine。原始模拟器机型为 iPhone 12/13 (Pro)，844×390，safe-area left/right 47、bottom 369。
- 独立 Go/MySQL/Redis/HTTPS/WSS 测试环境沿用现有专用容器、受控配置和同源码树 API 二进制，二进制 SHA256 为 `e3901e2e4b776da28eb8608affac804598233ebeafdbd975755c1af1dfefc1d1`，与既有受验文件一致。只对该自有 Go 进程注入 SIGKILL。开发工具与 Go 客户端、受控呈现夹具、用户手机分别计证据，不互相替代。

私有证据目录：`/root/.config/awesome-zero-platform/acceptance-0055-a4551b7/`。其中包含命令脚本、原始日志、产物、处理后脚本、截图、失败记录、作用域说明和哈希清单；不将凭据、QR 或私牌截图提交到仓库。

## 3. 精确 SHA 的 CI 与本地门槛

独立查询全部 job 的 `head_sha/status/conclusion`；以下均为本候选，未重触发成功 CI。

| 工作流 | 最终结果 |
| --- | --- |
| [主 CI 42452494407](https://github.com/oublie6/awesome-zero-platform/actions/runs/42452494407) | SUCCESS：integration、unit、admin_web、secure_envelope、runtime、report，6/6 |
| [Fair Doudizhu 42452494459](https://github.com/oublie6/awesome-zero-platform/actions/runs/42452494459) | SUCCESS：application、mysql_integration、client_foundation、manifest_client，4/4 |
| [Cocos Linux 42452494371](https://github.com/oublie6/awesome-zero-platform/actions/runs/42452494371) | linux_source SUCCESS；linux_native_export **SKIPPED** |

本地独立执行 `verify-toolchain.cjs`、`node --test scripts/cocos-linux-wechat/test.cjs`、`bash scripts/verify-client-foundation.sh`，以及真实 `build.cjs`。工具测试 **5/5**、通用客户端 **31/31**、斗地主协议 **49/49**、场景/客户端 **105/105**，合计 **190/190**，相关 typecheck/build exit0。静态 Creator3.8.8 声明只是兼容辅助；本轮另通过真实 CLI4 生成引擎声明的严格类型检查，未用历史 Creator 产物代替。

## 4. 实际原生构建、产物与预览配置对照

执行现有脚本，使用固定 CLI/Engine、缓存依赖、隔离源项目副本，保留构建输入。构建从 06:22:05 至 06:27:51 UTC；CLI 自报实际 wechatgame 阶段 2min10s。`build.exit=0`，验证 `enginePlatform=WECHAT`、`game.js/game.json/project.config.json`、官方 engine/web adapter、真实 Entry.scene 入口，原始输出 BigInt 审计通过。

原始本地产物：`native/fair-doudizhu-wechat/`，**26 文件、3,112,907 字节**；导入前仅填已有授权 AppID 和独立项目名，保持 `urlCheck=true`。收尾逐文件复核初始 manifest，全部匹配。

| 证据 | SHA256 |
| --- | --- |
| 原始原生产物确定性归档 `original-native.tar` | `f734e57e3841e220581e229de103564317ba377f872ae68da199894093ea3cd0` |
| 原始 `game.js` | `4635cf7530900863cd82f6b6221b8d18b8651e490347499a1001fcfc51934f21` |
| 已上传开发预览使用的 `project.config.json` 副本 | `09a84ba347c94e67257503da79c4cc40f519b9a32468baf98c29d78b256e0d44` |
| 实际微信 ES6 处理后的 `src/chunks/bundle.js` | `879a3ca22bee90d7a68fa51cea56db8392b9b0dfb5d79c1daf010fca9b9c0eb3` |
| 用户真机加载失败截图的私有副本 | `e1bfa52887ea301ebe77fb2bca64241e23b5769f113fec60a3c1bcd3ed73acd6` |

CLI 预览信息只返回体积，没有返回远端上传包整体二进制，**没有声称取得该远端包整体 SHA256**。通过固定源代码、未改的本地 JS、上传配置副本、处理后脚本、预览信息和 QR 哈希关联。检查 11 个已知服务端秘密值，两个客户端目录均无匹配；AppSecret/保护根没有进入产物。

构建会尝试导入未使用的 3D 内建资源：cmft 缺失、FBX 转换器选择 darwin 文件而失败；2D 导出最终成功，相关错误没有被抹去。不由此认定 3D 工具链可用。CLI 导入还改写了外部引擎的 5 个内建资源 `.meta`，已记录；没有修改引擎实现代码，也不冒称整个外部 checkout 完全无变化。平台扩展仍是仓库拥有的实验性 2D exporter，分包/远程资源没有实现资格。

| 配置/尝试 | 实际结果 |
| --- | --- |
| exporter 默认 `es6=false,minified=false` | 模拟器正常；开发预览报 `invalid file: assets/main/index.js, 3:6141`、`Unexpected token ?`，定位到 `??` |
| 私有副本只改 `es6=true` | 越过语法门槛；平台拒绝：`source size 6038KB exceed max limit 4MB`。这是本次工具实际返回，不泛化为所有平台模式的限额 |
| 私有副本 `es6=true,minified=true` | 开发预览 SUCCESS，**3,380,495 字节**；生成 QR，用户手机实际扫码；同配置模拟器启动失败 |
| 本地 A/B：同一副本只回退 `es6=false`，保留 `minified=true` | 场景和实际原生 HPKE 恢复。仅为本地对照，没有重新上传或将手机版本冒充修好 |

后续三项都是私有产物配置诊断，**本地源码未改**，也没有修补 exporter。已保存真正上传时的配置，避免后来的本地回退覆盖预览证据。

## 5. Linux 开发工具、SDK、UI 和真机

### 原始配置与本地 A/B 可执行范围

- `islogin` 两次实际返回 `{"login":true}`；现有开发工具会话可用。没有重新安装环境，未将历史桌面微信启动旧体验版的结果计入本轮。
- `smoke.cjs` 对新产物执行真实 SDK 原生 touch：主页→规则→返回→房间大厅；安全随机接口返回 32 字节，原生 `2n ** 256n` 运算正确。
- 实际 `wx.login` 返回 code；SDK 临时 session 存储/清理、前后台监听注册/移除通过。分享只确认接口存在，**没有发送消息，也没有将分享送达或前后台实际事件投递计为通过**。
- 房间 14 个实际操作项均可通过原生翻页访问；前后翻页保留输入组件内容。长文本输入值是程序化 EditBox 标记，不是软键盘输入通过。已检查实际截图；视觉结论限于该模拟器尺寸。
- 在真实 Cocos 场景中，以本轮从真实授权 Go 读取的 17 张牌投影作**呈现专用夹具**，注入 pending 呈现标志以强制分页；17 张牌均通过原生触控选中，跨页保留，旧节点在真实帧后销毁。12 次往返的内容节点数交替 21/11，没有累积。这不是微信账号登录、业务命令、异步网络或完整对局 E2E 的替代证据。
- 最后实际销毁 Bootstrap：组件/content/按钮失效，`canvas-resize` 和 `design-resolution-changed` 的该组件监听由存在变为不存在。`renderer17.json`、`destroy.json` 记录范围。
- 原始输出中的实际 HPKE 模块使用真实微信 SDK 熵，独立 Node 接收端解密成功，错误 AAD 拒绝；本地关闭 ES6 的对照也通过。

### 真机实际执行

用户确认可配合，实际使用**荣耀100 Pro / MAA-AN10、MagicOS10.0、Android16、系统版本 `10.0.0.175(COOE170R7P1)`** 扫描上述 3.2MB 开发预览码。用户反馈进入后停留在小游戏加载页、一直转圈，并提供 `Screenshot_20261108_145440_com_tencent_mm_AppBrandUI1.jpg`；已读取、核对并私有保存其哈希。

**真机启动检查：失败现象已确认，Cocos 首页未出现。** 微信版本、手机基础库、网络类型未取得，不能伪造；截图本身不区分包下载或脚本初始化失败。与该包相同配置的开发工具有明确 BigInt 错误栈，但该错误栈是**模拟器日志，不冒充手机日志**。

首页/规则/房间交互、真实输入与软键盘、长房间码、17 张手牌、翻页选牌、屏幕变化、安全区、后台返回、断网恢复、关闭再进、分享、真机完整对局均 **NOT RUN：被当前预览启动问题阻断**。不再把本轮真机条件写成“没有手机”或“微信未登录”。未取得正式审核/生产发布资格。

## 6. 真实身份与严格网络边界

实际 SDK code 通过独立受控 Go 服务端向微信换证，签发 Bearer 后正常 `/auth/session` 解析统一内部 AccountID；重复使用 code 和未认证请求被拒绝，测试 Token 登出撤销成功。AppSecret 在服务端配置内，未写进客户端。

这段换证由测试控制端通过自有 loopback HTTP 送达 Go，是**分段真实身份验证，不是微信客户端 HTTPS/WSS 全链路 PASS**。小游戏自身对当前 fixture 的 HTTPS 返回 `request:fail`，WSS 失败，登录界面显示 `platform HTTP failed`；工具网络日志有 SSL `net_error -202`。现有公开客户端配置指向 `https://127.0.0.1:28443`，使用自签测试证书，手机也不能直接访问该地址；本次连接集群未发现 ingress。未证明存在可供该 AppID 使用的批准公网域名/TLS 端点。

没有关闭 TLS/合法域名校验、增加 ignore-certificate-errors 参数或把服务端秘密放进客户端。Node Go 客户端通过 `NODE_EXTRA_CA_CERTS` 信任自有测试 CA 并保持证书校验；这不等同于微信平台网络资格。跨账号/私牌隔离与失效 token 等未变协议的更完整矩阵沿用已校验历史证据。

## 7. 当前真实 Go 完整局、托管与恢复

本轮三个不同真实认证 Go 账号经 HTTPS/WSS 创建、加入、准备，使用正常 Controller 发起 `client-contributed-server-custodied-v1` 新局；各自输入测试记忆字符串、客户端生成 32 字节安全随机数，经真实 HPKE 和服务端可信 context 提交。没有新局 recovery kit 或本地随机数备份。

- 一次真实贡献已提交但控制端丢弃响应，精确重试断言请求完全相同；三人完成贡献并等待真实 drand 后，各自获得授权 17 张牌。
- 叫地主命令实际已确认后丢响应，在隔离 API 上执行整进程 SIGKILL（**06:55:48 UTC**）再启动；新认证 Controller 从服务端恢复 PLAYING/live v2，已确认的原 17 张牌保留。继续完成整局，共 **59 个有效 live 操作、最终版本60**。
- MySQL 最终复核：59 条 journal/59 个不同 commandId，版本2–60、integrity_version 全2；贡献记录3条/3个不同 seat及commandId，ciphertext 最少227字节、nonce12字节；领域 Hand 为 COMPLETED。不能将领域 aggregate version 与 live version 混为一种版本。
- 终局三席积分、版本、摘要通过实际 API 契约检查；三位本人分别重新取得自己的随机数证明，在客户端本地复算 commitment，正确字符串通过，错误字符串拒绝。终局摘要 **`82706aa8eda2c0af46ab591a66ffe2fd87f060980937cd21efd506fdfb263083`**。
- **本轮新局没有另行独立重算整套 BLS/牌序/过程证明**；服务端 verified evidence 与客户端 schema/版本检查不等同于该证明。完整独立验证、负例、加密记录原文对比及大规模故障证据沿用同 Go 树的 [Goal0047](0047-independent-acceptance.md) 及其 Round18/19 证据链。
- 清理按设计先客人后房主，通过正常授权 API 关闭本轮两个房间，保留测试数据库及归档。没有通过 SQL 删除或更改业务状态。

**这是一局真实 Go 客户端完整局，不是微信完整局或“真机+后端混合 E2E”。** 17-card 原生呈现夹具使用控制端的只读投影作观察，不参与服务器恢复或提供权威状态。

### UNKNOWN、历史回调与辅助脚本的诚实范围

| 检查 | 本轮依据 |
| --- | --- |
| Fairness UNKNOWN | 当前新局真实接受/丢响应/相同请求重试，3条贡献不增加，PASS |
| Room UNKNOWN、跨选择/账号/晚回调 | 未变实现及 Goal0054 原始真实矩阵和哈希复用；不把本轮一次未命中 WSS 发送路径的辅助注入算成新 PASS |
| Hand UNKNOWN/重启 | 当前已执行 accepted bid 丢响应、整 Go SIGKILL、fresh controller恢复v2；辅助脚本随后错误地先走高层房间重绑定，触发 pending-Hand 保护。**本轮这个新用例没有成功完成原请求跨重启精确重试**；精确重试结论限于同源 Goal0054 既有证据 |
| S1/S2 旧消息、同 ID ACK、回调注销 | 同源 Goal0054 五类真实消息/连接矩阵复用，没有重跑整个旧专项 |
| 新冷启动与完整局 | 当前从服务端恢复、继续59操作至v60，PASS（Go测试端范围） |

私有辅助脚本还纠正了组件查询、运行版本标签、误用 `reconcileHand` 方法、终局 owner-first 清理等夹具问题；旧 EXPIRED fixture 也没有冒充可继续的 live hand。原始失败文件保留。`go-finish.json` 的原始 FAIL 包含手工清理次序错误，业务断言已通过；随后 `cleanup-final.json` 正常 guest-first 关闭通过，汇总为 `go-summary.json`，不把所有辅助脚本说成一次全绿。Cocos A/B HPKE 复用脚本的静态 scope 标签修正在单独 `native-hpke-ab.json` 中说明。

## 8. 问题与建议

| 编号 | 优先级/状态 | 最小依据、影响与交回建议 |
| --- | --- | --- |
| G55-N01 | **P1，OPEN** | 固定 SHA 默认 exporter 输出使用微信预览检查器不接受的 `??`，`assets/main/index.js:3:6141`，preview code10。阻止正常开发预览/设备交付。修复项目拥有的生成目标/语法降级链路，保留 BigInt 原生语义；不得仅以静态或模拟器 green 关闭 |
| G55-N02 | **P1，OPEN** | `es6=true,minified=true` 上传成功，但实际处理后 `src/chunks/bundle.js` 出现 `Math.pow(2n,256n)`。仓库 audit 对实际处理后脚本拒绝；同配置工具控制台报 `TypeError: Cannot convert a BigInt value to a number`，`bundle.js:48`，场景未启动。关闭 ES6 的同源对照恢复场景与HPKE。需要修复转译策略并审计**最终微信处理后代码**，独立接收端HPKE/BigInt及手机启动都必须重验 |
| G55-C01 | P2，配置诊断 | 仅启用 ES6、未压缩时实际源码包6038KB被平台拒绝；启用 minified 后3380495字节可预览。体积这一维已在私有配置验证可解决，但不解除 N01/N02；发布构建应固定并测量真实最终包，分包能力未实现 |
| G55-E01 | 外部/部署 BLOCKED | 现有仅loopback自签fixture；完整小游戏微信换证→合法HTTPS/WSS→房间尚未验证。需要提供可访问的测试后端及批准域名/有效TLS配置；AppID/AppSecret缺失、开发工具未登录不是本轮原因 |

建议 ChatGPT 集中修复 N01/N02 和正式导出配置，使用同一 Linux-native pinned CLI/Engine 流程。候选方向是构建阶段降级不兼容语法并保持 BigInt、避免再次让工具把 BigInt 改成数字 `Math.pow`；具体 target 必须实际验证，本文没有把未经验证的参数当作修复。随后固定新 SHA，执行本地原生导出→最终包语法/体积/BigInt/HPKE→工具启动→荣耀真机→真实身份/网络→完整客户端局/终局/冷恢复。保持原测试标准和服务端权威，不引入另一套 OS/Creator 常规流水线或客户端每局备份。

## 9. 八项完整交付矩阵

PASS 只适用于该行完整声明范围；部分成功不自动升级整项。

| 编号 | 结论 | 现有证据与剩余必需门槛 |
| --- | --- | --- |
| GOAL1 真实平台身份与会话 | **BLOCKED** | 实际SDK取码、可信Go分段换证/统一AccountID/撤销通过；小游戏合法HTTPS/WSS全链路未通过 |
| GOAL2 房间及Cocos交互 | **BLOCKED** | 原始模拟器14项菜单、翻页保留输入及受控17牌呈现通过，真实Go房间通过；当前可预览客户端不能启动，完整SDK房间交互未完成 |
| GOAL3 公平贡献与加密验证 | **BLOCKED** | 当前Go三份贡献/本人终局承诺、原生SDK熵及HPKE通过；实际微信UI贡献→终局验证未执行，且预览转译破坏BigInt |
| GOAL4 完整可玩斗地主 | **BLOCKED** | 当前真实Go三人59操作/v60通过；微信客户端从登录到结算的可玩流程没有完成 |
| GOAL5 B级持久化恢复 | **PASS（服务端已声明恢复范围）** | 同Go树历史故障/epoch/checkpoint/证明证据哈希复核，加当前整进程SIGKILL、fresh授权恢复v2/原发牌及继续终局；不代表微信冷重启UI或生产存储灾难资格 |
| GOAL6 三人E2E及故障 | **BLOCKED** | 当前三真实Go客户端/响应丢失/进程故障及既有原请求矩阵有效；当前原生/真机整合E2E尚未通过 |
| GOAL7 性能与稳定性 | **BLOCKED** | 既有同源100/1000局、Race、checkpoint及percentile证据复核；本轮12次原生翻页/节点和销毁通过。未取得可启动真机的长时会话/网络/内存证据，未自创生产SLA |
| GOAL8 Linux原生、工具与真机 | **FAIL** | 新Linux导出通过；真实预览检查、最终转译兼容和手机场景启动失败，见N01/N02 |

C09/C10：**DEFERRED**，既定范围修订不变。Douyin当前原生 exporter/SDK/真机、正式平台审核与发布资格仍未证明，未用旧Creator3.8结果替代。

## 10. 未执行项、资源与收尾

- 真机可用且实际尝试，不再以“缺设备”挡住测试；启动失败后的全部手机UI、SDK业务、软键盘、安全区、后台/断网/关闭再进、完整对局及本人核验 NOT RUN。手机微信/基础库版本和网络未取得。后端可执行项已继续完成。
- 没有执行新整套历史密码学/故障/压测，也没有重触发成功CI；源树和哈希匹配的原始证据列在私有 `reused-evidence.json`。新局独立BLS/牌序全证明、手机长期稳定性、生产硬件percentile未实测，不计新PASS。
- 构建被限制3GiB；工具进程与Go服务分别受控，退出前采样约1.79GiB/45MiB，cgroup OOM事件为0。这是包含编辑器的进程组采样，不是手机客户端堆或生产Go性能预算。没有取得本轮MySQL提交时延分位数/Go Goroutine长期曲线，不能由短局推导SLA。
- 已停止自启Go监督器、开发工具及自有TLS容器，移除该TLS容器；既有测试MySQL/Redis停止并保留卷、归档和历史证据。旧失效调试上下文留下的自有探针已终止，未从挂起探针推断成功。现有微信登录profile、原始用户截图和原凭据库保留。
- 清理临时凭据副本、私钥、control token，去除私有测试状态中的access token；客户端产物11个已知秘密值检查无匹配。必要日志/截图/产物仅私有保存，清单见 `SHA256SUMS`、`service-cleanup.json`。

**收尾判断：本轮独立验收交回 ChatGPT 修复；不能完成或归档 Goal0045/0046。** 本报告与 current 按 main-only 工作流提交、正常 push；受测版本始终为本文开头候选，提交号以本报告 Git 历史与最终交付摘要为准。
