# Goal 0057 — Kubernetes + Vue3/Cocos4 Web 独立验收

2026-11-09，结论：**FAIL**。固定产品源码为 `1301bb1df6f9521474f64d8dc0ce35431c7242ab`。

真实 Linux `web-desktop` 导出产生了破坏 BigInt 的代码；仓库审计拒绝该产物。在隔离 Kubernetes 中部署其**未修复的诊断副本**后，三个真实 Chromium Context 均复现 `Cannot convert a BigInt value to a number`，不能进入 Entry 场景。Vue 登录、同源 HTTPS/WSS、真实账号隔离和 API Pod 重建后的身份校验通过，**不能代替**登录到三人完整牌局、终局核验及牌局持久化恢复。

Codex 未修改产品代码、仓库测试、协议、数据库 schema 或共享部署。仅新增本报告及 `current.md` 允许的报告/状态内容。Goal0045/0046 保持 `in_progress`；Goal0056 用户暂停的手机诊断未恢复。本报告完成本轮独立验收交接，不是产品完成或发布批准。

## 来源、环境和边界

- 初始工作树干净、分支 `main`；`git pull --ff-only` 从 `39cbf0b` 快进到固定候选，随后核对 HEAD。生产构建使用该 SHA 的 `git archive`。报告提交与被测产品 SHA 分开。
- 已阅读 AGENTS、当前 Goal、Goal0046、四篇原则，以及本轮要求的客户端、部署和构建资料；按现有真实配置/路由检查，没有把旧文档里的目标架构当成实现证据。
- 所有临时清单、Dockerfile、探针、账号、密钥和原始证据均在私有目录 `E=/root/.config/awesome-zero-platform/acceptance-0057-1301bb1`；不提交凭证、截图、原始数据库或构建产物。
- 本轮属于客户端集成与临时部署适配。房间/手牌/公平性及 journal/checkpoint 仍由 Go/MySQL 负责；Redis 为辅助能力。保持单副本 API、既有模块接口和稳定 AccountID，没有引入平台游戏语义或新的分布式架构。
- Node22.23.4、Go1.25.8、Playwright1.64.1、Chromium156.0.8078.4；两节点各约8vCPU/7.8GiB，无 swap。构建顺序执行，Go `-p 1`/`GOMAXPROCS=2`；浏览器为一个 Chromium、三个独立 Context，systemd 明确 `--uid=root`，限制1GiB内存/150% CPU。
- CLI `f506fa3de403b48b4833334fa48dddd27bb26364`，engine `d0a696cfe63fd8d19c70d7913ba11aa641c2e3c7` / `4.0.0-beta.0`。复用既有 Linux 工具，没有重装 CLI/engine、使用 Wine 或历史 Creator3.8.8。导入器留下5个 engine 内置资源 `.meta` 修改，未修改其引擎实现源码；未擅自清理工具目录。

## 固定 CI 基线

读取 GitHub jobs/steps，并另取 run 元数据确认四个 run 的 `head_sha` 均为上述候选。没有重新运行无关的大型 Go 矩阵。

| 工作流 | Run | 实际结果 |
|---|---|---|
| CI | [42566995725](https://github.com/oublie6/awesome-zero-platform/actions/runs/42566995725) | 6个 job SUCCESS |
| Fair Doudizhu | [42566995769](https://github.com/oublie6/awesome-zero-platform/actions/runs/42566995769) | 4个 job SUCCESS |
| Fair Doudizhu Vue3 Web | [42566995792](https://github.com/oublie6/awesome-zero-platform/actions/runs/42566995792) | Vue job SUCCESS |
| Cocos Linux Native Client | [42566995702](https://github.com/oublie6/awesome-zero-platform/actions/runs/42566995702) | `linux_source` SUCCESS；`linux_native_export` **SKIPPED** |

CI 不证明本次真实 Web 导出、K8s 或浏览器业务通过。

## 实际构建与产物

本机按顺序实际执行：

```sh
node scripts/cocos-linux-wechat/verify-toolchain.cjs
bash scripts/verify-client-foundation.sh
node scripts/cocos-linux-web/build.cjs "$E/cocos-current"
npm ci --prefix clients/game-web
npm run test --prefix clients/game-web
npm run build --prefix clients/game-web
node --test scripts/game-web/stage.test.cjs scripts/cocos-linux-web/test.cjs
# 使用 E/public-config.json 的公开 HTTPS/WSS 端点和验证根装配诊断副本
node scripts/game-web/stage.cjs "$E/cocos-current/fair-doudizhu-web" clients/game-web/dist
```

命令使用上述固定 Node 和 `COCOS_CLI_ROOT`/`COCOS_ENGINE_ROOT`，详细 argv/配置保留在 E。

| 类别 | 结果 |
|---|---|
| 源码检查 | portable core33、protocol49、Cocos113，共195项通过，类型检查/构建通过；secure-envelope 构建通过。本轮没有重跑其无关大矩阵 |
| Vue 正式构建 | `npm ci` 成功，6/6 auth 单测通过，`vue-tsc` 和 Vite 正式构建成功 |
| Web 配置/装配检查 | 7/7 通过；stage manifest29项，另有 manifest 文件，共30个公开文件 |
| 原生 Web 导出 | CLI 自身报告 `web-desktop` 成功，fresh-input 导出使用缓存引擎，CLI构建约44秒；**外层审计退出1，验收 FAIL** |
| 原生产物保全 | 24文件/16,933,866 bytes；原始 tar.gz SHA256 `4710b4d5cd91b593b688990d204575f062ba193b490a81bd5131a866a6958440` |
| 错误 bundle | `src/chunks/bundle.js` 1,875,933 bytes，SHA256 `b9f38aa9a403cda4d2c74ad9755c1c963c963cbe875b9f4b67541911e3e49fce` |

首轮导出期间发现本地 package `dist` 旧于候选（缺少新 `GameSession.loginWith`）；该次日志和产物保留，不作合格候选证据。随后完整重建客户端包，确认 fresh 输入 `session.js` 与当前重新构建文件逐字节一致，再正式重导，仍独立触发同一 BigInt 审计错误。

fresh 导出还记录四个 `TS2688`（缺 `temp/declarations/cc`、`cc.custom-macro`、`cc.env`、`jsb`），CLI 随后仍报告脚本构建成功。**生成声明下的原生严格类型检查未合格**，不能被静态兼容类型检查替代。内置 skybox/cmft 缺失、FBX 调用 Darwin 二进制也产生导入日志；本轮没有把未使用3D资源导入错误误报为独立游戏功能缺陷。

诊断装配仅注入既有公开配置：`https://127.0.0.1:29457` / `wss://127.0.0.1:29457/ws` 和公开签名验证根。Vue/Cocos 配置一致；所有导出的 Cocos JS 与原生产物逐字节一致。没有替换 `Math.pow`、修改密码学依赖、跳过标准审计后宣布 PASS，或用假引擎替代产物。11个实际敏感值对30个公开文件的精确扫描无命中。

## 容器与 Kubernetes 实施

Go 使用固定 SHA 源码归档、仓库 Dockerfile 的编译步骤，私有副本仅固定基础镜像 Digest 并设置 `GOMAXPROCS=2`；真实 `CGO_ENABLED=0 ... go build -p 1 -trimpath` 成功。Web 使用固定 Digest 的 nginx-unprivileged1.29.4，将实际 Vue/Cocos 静态产物与临时代理配置打包。构建日志、Dockerfile、基础镜像信息和镜像归档均保留。

| 组件 | Docker image ID | 实际部署的 containerd manifest Digest |
|---|---|---|
| API | `717e6fadd26eda2eb93a73d803b6786a0273945e1e8a9a09b9447b538416ff3e` | `a00c238eb7463929289f16b4675b26d8c13228e2fd27aafcf6e5963c63fe9306` |
| Web诊断副本 | `9d0721185da7bfccdfe73ffc4bf9c4510eec79b2f81ada0fd5d24c8baabedd2a` | `c23a8cb7024e5de3e57a9fa8a12d597308740eaffcb9b30c338e91412e4f93fb` |
| MySQL5.7.44 | `5107333e08a87b836d48ff7528b1e84b9c86781cc9f1748bbc1b8c42a870d933` | `0b14b8e5c533fb43d07327fa8fe6c7393358002310d9f0d01976d387822a1bd3` |
| Redis8.8.0 | `55563f45704d59e891b6aa643014bb6b562abf21909a44b3b297502415b68e29` | `961f172bb2427dc11746904b17d853e59ee654b991343ea3955342b65157c907` |

表中均为 SHA256；Docker config ID 与 manifest Digest 是不同身份，不混用。API/Web tag 包含完整候选 SHA；清单最终使用 `@sha256:`，无 `latest`。通过本机 Docker archive 导入实际 containerd；HK 的 Web/Redis manifest Digest 也核对一致，无注册表权限假设。

- context：`kubernetes-admin@network-learning`；API server `https://10.0.72.27:6443`。初始两节点 Ready；权限确认允许创建测试 namespace。
- 独立 namespace `awesome-zero-game-test`，初始不存在。共13个对象：namespace、4 Deployment、4 ClusterIP Service、`g57-config`、Secret `g57-secrets`/`g57-tls`、`g57-isolation` NetworkPolicy。没有 Ingress/Gateway 控制器；测试 Web 自带 Nginx TLS 网关，通过仅绑定 localhost 的 `kubectl port-forward` 访问实际 K8s Service。
- MySQL 使用新专用目录 `/data/awesome-zero-platform/goal0057/mysql`；初始化前确认路径不存在、数据库 `goal0057` 无表，再应用完整 schema/seed，得到20表。实测 `innodb_flush_log_at_trx_commit=1`、`log_bin=0`。没有共享数据、PVC/PV 的升级、重建或清理。
- 配置由当前源码确认：启用 Doudizhu/RevealKeys、配置 drand Quicknet/auto、独立稳定保护根、独立 MySQL/Redis/认证凭据。敏感值仅 Secret/私有证据；ConfigMap 不含敏感值。为受控观察设置 bidding/playing300秒，不改公平协议或后端权威模型。
- 每轮实际变更前均运行 `kubectl kustomize`、`kubectl diff -k`，核对只涉及该 namespace。首次 diff 因 namespace 不存在退出2，创建该 namespace 后 diff 预期退出1；原失败保留。先启动 MySQL/应用 schema，再依次启动 Redis、API、Web。
- 私有清单修正过 named `nonroot` 无法通过 kubelet 非 root 验证的问题：显式 UID/GID65532，镜像/产品未改。首次 rollout timeout 保留；修正后四组件实际 rollout/Ready，通过真实 MySQL/Redis 连接。
- 限额：API512Mi/1CPU、Web128Mi/0.5CPU、MySQL512Mi/0.75CPU、Redis128Mi/0.25CPU。全部有健康探针，内部数据库无公开端口。

### 两节点资源与一次环境中断

遵照用户内存提示，先将测试 Web/Redis 放 HK、MySQL/API 放控制平面；当时分别约4.59/3.68GiB可用内存。真实双节点部署及镜像导入成功。约13:08–13:11 UTC，HK 的 kubelet 心跳中断，节点成为 `NotReady/NodeStatusUnknown`，导致跨节点 Redis/入口不可达。

立即停止自己的浏览器并仅缩容自己的 HK Web/Redis。SSH 实测 HK5.6GiB可用、无swap、load0.41，kubelet/containerd active；**不能归因为 OOM，也未确诊网络根因**。未更改共享节点、网络恢复通道或共享服务。HK 自行恢复 Ready，kubelet stats 确认其本轮 Pod 已清空。

为隔离环境波动，把仅本轮 Web/Redis 改放控制平面，重新 diff/apply；该机约4.2GiB可用。Redis 辅助会话随专用实例重建而重置，测试账号仍在独立 MySQL。最终浏览器证据来自四组件均 Ready 的控制平面部署，两个节点当时均 Ready；中断期间的超时不作产品渲染缺陷证据。**BigInt 缺陷是在稳定回退环境重新复现的独立结果。**

外部环境边界：未提供获批准的公网游戏域名/证书，因此公网入口资格仍未验证；本目标明确允许的内部受信任CA访问已实测，公网条件不被误列为阻止本次内部浏览器验收的原因。HK跨节点持续稳定性及此次心跳中断根因仍未资格化。真实手机/微信域名资格不属于这些内部Web控制项的PASS，Goal0056设备诊断继续按用户要求暂停。完整业务的当前直接阻塞是产品G57-N01，而不是缺少本轮测试账号或可用K8s入口。

## HTTPS/WSS 与真实浏览器矩阵

浏览器实际访问 `https://127.0.0.1:29457`，流量经 port-forward 到 K8s 中的 TLS 网关；不是 Vite、本地静态服务或 Mock。专用测试 CA 被导入 root 的 NSS 信任库，证书 SAN 为127.0.0.1。OpenSSL `-verify_return_error -verify_ip` 返回0，Chromium securityDetails 记录 issuer `Goal0057-Test-CA`、TLS1.3、安全上下文，WebCrypto和原生BigInt均存在。

没有 `ignoreHTTPSErrors`、证书忽略参数、关闭校验或修改浏览器密码学实现。首次受限 systemd unit 缺 HOME/USER，Chromium 未读取用户信任库；环境探针确认后改为显式 `--uid=root`，保持内存限额和正常证书检查，重新实测通过。首次证书失败原样保留。

网关逐字保留 `/auth/*`、`/v1/*`、`/api/v1/*`，`/ws` 使用 HTTP1.1/Upgrade/90秒超时；没有套用 Admin Web 的 `/api/` 去前缀规则。CSP 对导出页面的实际内联启动脚本使用哈希；未改变游戏 JS。

| 项目/证据层级 | 结果与实际边界 |
|---|---|
| 网站启动，浏览器 | **PASS**：Vue首页/CSS/JS正常；三个 Context HTTPS 页面加载204.6/129.2/184.2ms，无静态资源404；故意请求的四个不存在 API 返回404 |
| 真实账号，浏览器+Go | **PASS**：正常管理员 API 创建三个不同账号；真实密码登录、错误密码401、`/auth/session`、三个账号隔离、Vue刷新会话恢复 |
| 浏览器存储 | **PASS**：每个Context仅 `sessionStorage['fair-doudizhu-session']` 保存其服务器访问令牌，localStorage为空；无密码/refresh token/牌局快照持久化 |
| 注销/换账号 | **PASS（补充探针8/8）**：等待真实 `/auth/logout` 200/OK后，三个原令牌均401、存储清空、无残留iframe；补充探针从大厅注销，运行中游戏场景注销仍未验收。A退出后B登录返回B身份且无旧iframe。首次探针在异步注销完成前读到200，是私有探针时序错误，保留原FAIL并纠正后重验 |
| 认证WSS，浏览器+Go | **PASS**：三个Context各自真实 bearer-subprotocol握手及 `system.ping/pong`，服务器仅选择`bearer`；无URL令牌；未授权握手拒绝 |
| 路由/身份边界 | **PASS**：未授权 session/rooms401，真实crypto manifest200、缺key404、其他不存在auth/v1/api/v1路径404；错误API不返回Vue SPA HTML |
| API Pod重建，K8s+浏览器 | **PASS，仅身份基础链路**：专用Deployment实际替换Pod，UID改变；三个原令牌在新Pod再次读到相同账号。最终回退环境耗时14.104秒（早期双节点窗口18.642秒）。没有一手牌，因此不是 B级牌局恢复PASS |
| Cocos4引擎/Entry，浏览器 | **FAIL**：真实引擎/资源返回200，三个Context均在bundle执行阶段发生同一BigInt异常，Entry/按钮就绪等待超时，截图是Vue内的空白游戏区域 |
| Vue/Cocos消息桥，源码+浏览器 | 源码检查存在同源/source校验和无Token控制消息；运行时三个Context均无`GAME_READY`。`SESSION_AVAILABLE/ACTIVE/FAILED/REVOKED`、`OPEN_ROOMS/CREATE_ROOM/JOIN_ROOM`及重放/旧iframe/延迟跨账号矩阵 **未完成**，不能以源码测试替代实测 |
| Canvas输入/房间完整链路 | **未完成，受G57-N01阻塞**：Vue创建意图无法到达已就绪游戏；真实创建/加入/三人准备/开局、鼠标触摸牌桌、刷新/iframe重建后恢复房间均不作PASS |
| 三人完整业务E2E | **未完成，受G57-N01阻塞**：个人字符串、32字节CSPRNG贡献、HPKE托管、发牌、竞叫、出/过牌、结算、本人字符串和完整公平性证据均没有本轮完整浏览器证明 |
| 牌局故障恢复 | **未完成，受G57-N01阻塞**：不存在真实浏览器创建的手牌，未注入该手牌的Pod重启/响应丢失/UNKNOWN精确重试；未证明ACK动作、journal/checkpoint和相同手牌恢复 |

主浏览器探针原始记录为14PASS/4FAIL：三个场景失败，加一次过早注销查询失败。后者由独立的8PASS补充探针解决；没有改写原始JSON。更早私有探针错误地将 Fetch `response.status` 属性当成 Playwright `response.status()`，相关检查在回退环境使用修正探针重验。收集所有结果的主进程退出0仅说明探针正常收尾，不表示矩阵全绿。

最终数据库实读：4账号（管理员+3玩家）、0房间、0手牌、0 live actions、0 checkpoints。这直接反证完整业务或牌局恢复已经通过。没有用Go HTTP客户端或历史三人对局证据替代本轮浏览器。

## G57-N01 / P1 — Web导出把BigInt幂运算转成Math.pow，阻断启动

位置：`scripts/cocos-linux-web/build.cjs:40`的正式Web导出配置/实际生成链路；`:70`标准审计正确拦截。原依赖 `@noble/ed25519/index.js:222` 是合法 `const B256 = 2n ** 256n`，本轮实际bundle变为 `Math.pow(2n,256n)`。依赖源码未被Codex修改。

最小复现：在固定候选构建portable包后运行本报告原生Web命令，得到审计 `Generated Math.pow receives BigInt in src/chunks/bundle.js`、退出1。保留该原生产物，以标准stage注入公开配置后部署诊断镜像；真实浏览器登录Vue并点击创建房间。三个Context均出现：

```text
TypeError: Cannot convert a BigInt value to a number
    at Math.pow (<anonymous>)
    at Object.execute (.../game/src/chunks/bundle.js:50:2727)
```

预期：现代HTTPS浏览器中的合法BigInt操作保持语义，Entry正常启动并发出GAME_READY。实际：模块初始化终止、无GAME_READY，无法开始任何真实房间或牌局。标准审计FAIL与稳定K8s浏览器FAIL相互印证，不依赖HK故障，也不是浏览器缺少BigInt/WebCrypto。

交回ChatGPT修复Web目标的转译/构建配置、补足真实生成声明下的类型门槛和原生Web回归，再推送新的固定候选重新完整验收。不能修改审计标准、用补丁产物或将本次诊断镜像当成可交付版本。现有WeChat exporter/模拟器通过不关闭Web问题；Honor设备启动仍是另一未完成门槛。

## 资源、可观测性与未测指标

复用现有设施的只读能力：Prometheus `up`查询、Grafana `/api/health`数据库ok、Jaeger `/api/services`含app-api。没有部署另一套监控或公开日志面板。现有Prometheus配置未修改；本轮Pod/进程指标来自kubelet stats和专用API原有`/metrics`直接采样，不声称共享Prometheus已抓取测试namespace或Jaeger证明了本轮SQL子span。

| 测量 | 本轮实测 |
|---|---|
| Pod工作集最大采样值 | API162.668MiB、Web26.598MiB、MySQL185.992MiB、Redis12.035MiB |
| Pod CPU最大采样值 | API0.1924核、Web0.0164核、MySQL0.0167核、Redis0.0074核；约5秒采样，不是瞬时峰值 |
| 节点可用内存最小有效采样 | 控制平面2.920GiB、HK4.533GiB；不可达采样不被当成0字节/OOM |
| 浏览器 | 1GiB硬上限；运行中观察约490MiB，未测精确峰值；最终受限浏览器正常结束，无OOM证据 |
| WebSocket连接/Go资源 | 另用三个真实浏览器Context同时连接，active连接0→3→0、goroutine21→27→21，原生HTTP指标实读；该探针不是游戏E2E |
| Go heap采样 | 2,592,088–83,119,680 bytes；包含真实密码认证分配，不能推论长期牌局无泄漏 |
| Pod重启 | 最终四容器restartCount均0；API另有两次有记录的受控Pod替换，不能用restartCount0抹去重建 |
| 页面加载/身份恢复 | Vue三次加载见上表；最终API身份链路重建恢复14.104秒 |
| 游戏首次渲染/命令延迟/断线牌局恢复 | **未测得，G57-N01阻塞**，不填入伪造值 |
| MySQL游戏提交耗时 | **未测得**：没有手牌命令提交；fsync配置和账号持久化不等价于游戏COMMIT延迟证据 |

这是轻量集成诊断，不是生产压力测试、SLA或多Pod故障切换资格。没有扩容API假装高可用。

## 收尾、保留与回退

已停止所有本轮浏览器/采样/port-forward，撤销两个NSS库中唯一的 `Goal0057-Test-CA`；未动其他信任项。按专用 `stop-test.sh` 先停API/Web再停MySQL/Redis，四Deployment均0副本、测试Pod为空；两节点均Ready，控制平面约4.4GiB可用/无swap。共享Admin Web、数据库、可观测性及SSH网络恢复通道未修改。

保留清单和数据归属：仅上述namespace内13对象（含两Secret名称，不公开值）、私有E约948MiB、专用MySQL目录约201MiB、fresh构建输入约129MiB、候选镜像/归档及其构建缓存。HK只导入过本轮Web/Redis，未删除任何共享镜像。收尾root/data约54%/77%使用，约18GiB/8.9GiB剩余。没有运行中的测试体验环境或新增公网端口；保留资源不再消耗Pod CPU/内存。

诊断保留窗口至 **2026-11-16**，未安装自动销毁任务。E中保留受控恢复/停止脚本、精确manifest、Secret备份、原始输出、截图和哈希索引；恢复只能使用同一专用数据库/保护根，重新审查差异并重新建立CA信任。完全移除时只能逐项删除本轮四Deployment/四Service、`g57-config`、`g57-secrets`、`g57-tls`、`g57-isolation`及已核验的专用数据/镜像；**不得以删除共享namespace/PV/PVC或清空共享MySQL/Redis作为回退**。

报告二次自检重新核对固定SHA、真实CI/构建/部署/浏览器/故障证据层级，复查探针UID、HOME/CA、Fetch属性、注销完成时序和HK中断归因；保留初始失败，没有自修产品或缩窄游戏验收条件。运行时不确定项、生成声明类型门槛、桥接矩阵、完整业务和故障恢复均明确未通过。只有本轮**独立验收交接完成**；整体产品判定仍为 **FAIL，交回ChatGPT修复G57-N01**。
