# Web 静态资源预压缩、性能及部署报告（2026-11-27）

本次授权范围已完成：固化 level-9 预压缩、静态优先／动态兜底、Web 镜像打包、针对性测试、Web-only 部署及清理。复用 Goal0065 已验收业务产物，未重开 Goal0065，未修改或重新构建 Vue、Cocos、Go 业务代码，未重启 API。本报告属于运维优化交付及实施者自测／自审，不冒充新的独立产品验收。

实际基线已是此前动态 Gzip level-6 部署。cc.js 原始 12,622,964 bytes，基线 HTTP gzip 3,375,311 bytes，本次静态 gzip 3,358,403 bytes（3.36 MB / 3.20 MiB）：相对原始减少 73.39%，相对既有动态压缩再减少 16,908 bytes／0.50%。主要新增收益是消除请求时重复压缩 CPU，而非再次从 12 MB 降到 3 MB。浏览器冷启动总耗时未显示明显改善。

所有原始数据和检查结果位于 [证据目录](evidence/web-precompression-20261127/)。本机 HTTPS、固定限速结果不代表用户 Windows→SSH 隧道链路速度；未调查手机黑屏，未执行完整牌局、身份故障注入、连接管理专项或历史完整回归。

## 设计、归属及现状

遵循 AGENTS.md、principles 索引及 01/04：所有权为可复用静态 Web 构建／部署基础设施，位于 scripts/game-web 与 deploy；无 foundation/platform/business 依赖变化。公开契约是原始 URL／内容／CSP／缓存策略不变，仅按 Accept-Encoding 提供等价 gzip 表示；数据权威仍为原 Go 服务。依赖 Node 内置 fs/zlib/crypto 和原 Nginx 基础镜像。详细设计记录见 [design.json](evidence/web-precompression-20261127/design.json)，写于实施前。

读取实际部署 Nginx 配置、容器进程文件及 nginx -V：Nginx 1.29.4，支持 http_gzip_static_module。静态 JS 为 application/javascript; charset=utf-8，CSS text/css、HTML text/html、JSON application/json；SVG/WASM 在隔离实测分别 image/svg+xml/application/wasm。基线动态 gzip 的 cc.js 有 Content-Encoding:gzip、Vary:Accept-Encoding、弱 ETag、无 Content-Length；缓存 public,max-age=0,must-revalidate。优化后预压缩响应具有精确 Content-Length 和静态表示 ETag。两种编码都能重验证，跨变体 ETag 不混用。

[实际运行 server 配置](evidence/web-precompression-20261127/nginx.conf) 保留原 TLS1.2/1.3、证书挂载、CSP、安全头、proxy_pass、超时、WebSocket Upgrade。静态 /game/、/assets/ 和 / fallback 包含 gzip 片段；exact /ws 与原 regex API 路由保持优先级，未在 http/server 或 API/auth 位置开启动态 gzip。

## 固化实现及重复打包

- [precompress.cjs](../../scripts/game-web/precompress.cjs)：顺序流式 level-9，允许 JS/CSS/HTML/JSON/SVG/WASM 等文本格式；至少 1024 bytes，节省同时达到 128 bytes 和 5% 才保留 .gz。PNG/JPEG/WebP/音视频/WOFF2 不在允许列表。每份解压 SHA256 必须等于原始 SHA，保留全部原文件及有意义的 mtime。
- 压缩清单在 public root 外，记录所属 sidecar、原始／压缩哈希及大小。每次重做并清理所属旧 sidecar，覆盖更新、删除、收益下降；未知 .gz 冲突、符号链接及路径越界失败退出，临时文件／本轮生成文件失败清理。打包输入应是已核验的原始只读产物；同一源不要并发写入。
- [package.cjs](../../scripts/game-web/package.cjs) 以临时目录生成独立 context，仅替换带自身 marker 的旧输出，拒绝输入／输出重叠。复用原始站点；复制既有服务器配置、缓存及压缩片段和 [正式 Dockerfile](../../deploy/docker/game-web.Dockerfile)，不调用 Vue/Cocos/Go 构建。镜像 /etc/nginx/precompression-manifest.json 不公开服务。
- [gzip-static.nginx.conf](../../deploy/game-web/gzip-static.nginx.conf)：gzip_static on、gzip_vary on；动态 level6/min1024 兜底。HTML 由 Nginx 默认类型覆盖，显式类型无重复。
- [打包说明](../../deploy/README.md) 记录可重复命令、范围、原产物保留、所有权及回滚。Web CI 执行 scripts/game-web/*.test.cjs。

已逐文件核验基线／新镜像 30 个原文件 SHA 相同，新增 15 个 .gz，总 gzip sidecar 4,440,651 bytes。原 game-web-assets.json、HTML 内联脚本和 CSP 哈希均未改变。原始资源清单无须包含编码副本。

## 压缩及实际 HTTP

以下每项 gzip 大小均等于线上实际 HTTP body 长度及镜像 .gz 大小，响应 Content-Encoding:gzip、Vary:Accept-Encoding，解压 SHA 一致。完整 SHA 见 [预压缩清单](evidence/web-precompression-20261127/precompression-manifest.json)。

| 资源 | 原始 bytes | Gzip/HTTP body bytes | 减少 |
| --- | ---: | ---: | ---: |
| `assets/index-D3BBqkGY.js` | 85,157 | 32,664 | 61.64% |
| `assets/index-M5cYNUMW.css` | 11,419 | 3,506 | 69.30% |
| `game-web-assets.json` | 5,060 | 1,764 | 65.14% |
| `game/application.js` | 1,269 | 506 | 60.13% |
| `game/assets/internal/config.json` | 2,439 | 1,061 | 56.50% |
| `game/assets/internal/import/07/078d1fd7a.json` | 197,930 | 18,785 | 90.51% |
| `game/assets/internal/index.js` | 259,686 | 64,782 | 75.05% |
| `game/assets/main/import/e9/e9a5f2e4-6d64-4363-814e-dab771327440.json` | 1,350 | 625 | 53.70% |
| `game/assets/main/index.js` | 676,467 | 204,224 | 69.81% |
| `game/cocos-js/cc.js` | 12,622,964 | 3,358,403 | 73.39% |
| `game/index.html` | 14,457 | 4,817 | 66.68% |
| `game/src/chunks/bundle.js` | 1,843,720 | 564,457 | 69.38% |
| `game/src/settings.json` | 206,243 | 151,351 | 26.62% |
| `game/src/system.bundle.js` | 97,282 | 33,169 | 65.90% |
| `game/style.css` | 1,091 | 537 | 50.78% |

[隔离 30474 检查](evidence/web-precompression-20261127/http-30474.json) 与 [实际 29457 检查](evidence/web-precompression-20261127/http-29457.json) 均 PASS：

- 支持 gzip 得到与磁盘 level9 sidecar 完全相同的字节与 Content-Length，验证静态副本实际被使用；解压一次就是原文件，无双重压缩。
- identity、gzip;q=0、gzip;q=0,br;q=1、*;q=0 返回原 12,622,964 bytes，无 Content-Encoding。
- gzip 与 identity 各自 If-None-Match 返回无 body 的 304；交叉 ETag 返回 200。Vary 区分缓存表示。HTML 保持 no-cache,must-revalidate，非指纹游戏资源保持 revalidate，Vue 指纹资源保持一年 immutable。
- 运行时 game-config.js 和身份响应保持 no-store；登录、session、logout 均访问真实 Go，Accept-Encoding:gzip 下身份响应无压缩，身份匹配并正常注销。[身份检查](evidence/web-precompression-20261127/auth-29457.json)。
- 隔离探针实际验证 SVG/WASM MIME；更新原文件、重生成 .gz、带旧 ETag 获取新内容；移除某探针 .gz 后响应切换动态 level6（无 Content-Length、gzip 头及解压内容验证）。探针未打进部署镜像。
- 原始资源/CSP/HTML 清单哈希不变；/health/ready 实际代理 Go JSON ready，而非静态 /health 的 200。

Nginx 原有 ETag 基于 mtime/size，并非内容哈希。更新同长度文件时仍应改变有意义的源 mtime；本次保留此缓存契约。打包重生成可消除旧副本，但不声称改变了 Nginx 在人为保持相同 mtime/size 时的既有 ETag 限制。

## 真实浏览器性能

Chrome for Testing 156.0.8078.4，Playwright CDP，1280×900，SwiftShader 软件渲染；相同主机、HTTPS 127.0.0.1:29457，CDP 配置 latency20ms、上下行各 1,048,576 bytes/s。每版本 3 个独立 browser context 的冷 HTTP 缓存首次登录／进入，同 context reload 后热缓存恢复／再次进入，共前后各 6 次。使用本轮拥有的 QA 账号，未使用用户账号。

计时等待真实 GAME_READY 与 SESSION_ACTIVE 消息和真实 WSS 建连。宿主身份耗时含页面、Vue、交互及 Go 往返，不是纯服务端认证耗时。GAME_READY 表示场景就绪，SESSION_ACTIVE 表示最终身份／连接可用。本次未创建房间或完整牌局。

以下为各指标独立中位数；它们可能来自不同一次，不能直接相减推导细分阶段。

| 指标 | 冷：前 | 冷：后 | 热：前 | 热：后 |
| --- | ---: | ---: | ---: | ---: |
| 页面打开→宿主身份可用 (s) | 2.497 | 2.541 | 0.794 | 0.775 |
| cc.js requestStart→responseEnd (s) | 4.081 | 3.983 | 0.401 | 0.454 |
| 点击进入→GAME_READY (s) | 14.183 | 14.360 | 8.094 | 7.482 |
| 点击进入→SESSION_ACTIVE (s) | 14.510 | 14.716 | 8.469 | 7.670 |
| 打开／重载→SESSION_ACTIVE (s) | 16.974 | 17.257 | 10.163 | 8.444 |
| cc 下载完→GAME_READY，混合残余 (s) | 9.810 | 10.064 | 6.610 | 6.693 |
| cc.js ResourceTiming transfer bytes | 3,375,611 | 3,358,703 | 300 | 300 |
| 静态资源总 transfer bytes | 5,342,487 | 5,322,868 | 7,604 | 7,604 |

cc.js 冷下载约 4.081→3.983 秒，总冷 transfer 5,342,487→5,322,868 bytes，仅少 19,619 bytes；打开至最终就绪 16.974→17.257 秒（约 +1.7%），无法据此宣称整体变快。热缓存总耗时 10.163→8.445 秒，但 cc 下载后残余未下降、样本仅3次，不能将热缓存差异归因为预压缩。热缓存 cc transfer=300 是 ResourceTiming 对 304 响应的名义头部大小，encodedBodySize 仍显示缓存实体长度，不代表又下载了 3 MB。

cc 下载完成→场景这一混合残余冷约 9.810→10.064 秒、热 6.610→6.693 秒，包含 JS 执行、引擎初始化、其他资源和场景／布局等待。它与网络阶段分开记录，**没有执行独立 JS parse profiler**，不能称为纯解析耗时。共享 Chromium/GPU/操作系统缓存及主机调度仍可能影响结果；新 context 仅保证冷 HTTP 缓存，非进程／OS 全冷。

[Go 访问日志](evidence/web-precompression-20261127/server-auth-timing.json) 在各浏览器时窗内统计：前／后各3次 login 中位数 546／508ms，前／后各18次 session 中位数均5ms，均200。durationMs 是服务端 handler 墙钟耗时，非 CPU；按时间窗关联而非 requestId，不能完全排除其他共享流量。对比最终就绪采用上表实际浏览器端到端时间，不把 gzip 的字节下降等同于认证加速。

全部原始样本如下，完整事件与指标见 [before](evidence/web-precompression-20261127/benchmark-before.json)、[after](evidence/web-precompression-20261127/benchmark-after.json)。

| 版本 | 组 | 缓存 | 宿主身份 ms | cc 下载 ms | 场景 ms | 最终会话 ms | 总耗时 ms | 静态 transfer bytes |
| --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: |
| before | 1 | cold | 2583.0 | 3949.7 | 14022.9 | 14390.8 | 16973.8 | 5342487 |
| before | 1 | hot | 1694.0 | 400.6 | 8093.7 | 8469.0 | 10163.0 | 7604 |
| before | 2 | cold | 2237.0 | 4080.6 | 14183.4 | 14509.8 | 16746.8 | 5342487 |
| before | 2 | hot | 716.0 | 736.6 | 10167.4 | 10372.6 | 11088.6 | 7604 |
| before | 3 | cold | 2497.0 | 4089.2 | 14660.7 | 15123.1 | 17620.1 | 5342487 |
| before | 3 | hot | 794.0 | 339.8 | 7144.4 | 7370.7 | 8164.7 | 7604 |
| after | 1 | cold | 2498.0 | 4250.3 | 15481.2 | 15747.0 | 18245.0 | 5322868 |
| after | 1 | hot | 561.0 | 453.6 | 8231.2 | 8452.9 | 9013.9 | 7604 |
| after | 2 | cold | 2541.0 | 3939.8 | 14360.1 | 14716.0 | 17257.0 | 5322868 |
| after | 2 | hot | 801.0 | 506.5 | 6854.7 | 7086.0 | 7887.0 | 7604 |
| after | 3 | cold | 2730.0 | 3983.1 | 13484.2 | 13850.4 | 16580.4 | 5322868 |
| after | 3 | hot | 775.0 | 353.4 | 7482.0 | 7669.5 | 8444.5 | 7604 |

最初一次计时工具因按钮 accessible name 包含箭头、exact selector 不匹配而终止；修正观察器后重新采集完整3组，失败工具尝试未作为成功样本或产品故障。未对已完成数据选择性重跑。before/after 按部署顺序测量，非随机交叉试验。

## Nginx CPU 与镜像大小

浏览器采集结束后分别做3次顺序 cc.js gzip HTTP 完整读取；相同 Web 500m CPU／128Mi 限制。读取容器 cgroup cpu.stat，包含所有 Nginx worker 与时窗内背景流量，不含客户端解析或 Go CPU。

| 指标 | 动态 level6 基线 | 静态 level9 |
| --- | ---: | ---: |
| 三次 Nginx CPU 总秒 | 3.455468 | 0.074343 |
| throttled periods 增量 | 68 | 0 |
| 三次 HTTP 秒 | 2.538 / 2.101 / 2.255 | 0.091 / 0.105 / 0.100 |
| HTTP body bytes／次 | 3,375,311 | 3,358,403 |
| Docker 展开镜像 bytes | 70,663,304 | 75,112,681 |
| 本地导入 manifest layer bytes | 72,120,320 | 76,580,352 |

CPU 总量约降低97.85%，无明显增加，符合将压缩成本移至打包的目标。上述直连 HTTP 未加 CDP 限速，与浏览器表不能混为同一网络测试。镜像展开增加4,449,377 bytes（约4.24MiB），本地层增加4,460,032 bytes；这是保留原始文件和增加 sidecar 的存储成本，**非远程 registry 压缩传输量**。证据 [CPU before](evidence/web-precompression-20261127/cpu-before.json)、[CPU after](evidence/web-precompression-20261127/cpu-after.json)、[运行镜像](evidence/web-precompression-20261127/running-image.json)。

## 部署版本、回滚与验证

- 已验收业务源：`efc0699b0d62ae1284db86121619a746ce4f37ba`。打包实现及镜像标签 PACKAGING_REVISION：`91b180ecf1590dcf0c7f227984f1584f8d438e16`。本报告提交仅补文档／证据，不生成另一套业务产物。
- 实际 Web：`docker.io/awesome-zero-platform/game-web-precompressed@sha256:5d600a065401d985679d1cbc6e42cfdda07a174422b3704e66211c14b2150c47`；Pod game-web-7f869b45fd-qzz4x，Ready、restart0，运行文件30 original+15 gzip及配置逐项匹配。
- 保留原 Web 回滚镜像：`docker.io/awesome-zero-platform/goal0065-web@sha256:75ca57f31aecf11539876ee6166841c3ed883fe26bb524c1c0818750b43aa461`。本轮未触发回滚；故障时仅将 deployment/game-web 中 game-web container image 改回此 digest 并复检 readiness/29457。
- API 仍为 `goal0065-api@sha256:78083cc8ede1b176b179de55e7bdde7680714cd76fc131c83430a6c5a2d82dd6`，UID及 restart count 保持原值。

隔离环境 nginx -t／协商／缓存／探针通过后，最终生产配置再次 nginx -t 成功，才执行 game-web image 更新；检查 rollout、实际代理 API readiness、线上 HTTP、真实登录及场景／最终就绪／WSS。仅 Web 镜像字段变化，部署保护及回滚分支记录见 [rollout-result](evidence/web-precompression-20261127/rollout-result.json)。未做故意线上故障回滚演练。

[资源审计](evidence/web-precompression-20261127/integrity.json)：Deployment（允许 Web image 单字段）、Service、StatefulSet、ConfigMap、Secret 内容、PVC 及 API／依赖 Pod UID/restart 均保持；6 Pod Ready、unfinished hands=0。无 DDL/数据库迁移，测试仅使用已有 QA 身份产生正常会话。29457 systemd forward service active，未修改或重启其服务配置。

[清理结果](evidence/web-precompression-20261127/cleanup.json)：仅移除本轮隔离容器、浏览器及临时 CA；NSS trust store 与基线完全一致，30474/19181 关闭；用户29457代理 Go readiness继续200。原镜像及私有原产物留作回滚／重复打包。

## 测试、CI 与自审

`node --test scripts/game-web/*.test.cjs`（Node22.23.4）10 PASS、0 FAIL。覆盖确定性 level9／解压一致性／mtime，删除更新及不可压缩退化后的陈旧副本清理，tiny/压缩格式跳过，未知副本／路径／符号链接拒绝，打包重复及清单／配置字节保留，原有 stage/CSP/cache 规则。[完整输出](evidence/web-precompression-20261127/tests.log)。运行时还执行上文隔离及生产 HTTP、原文件完整性、浏览器与 CPU 检查。

实现 SHA 的三个自动 CI 均 SUCCESS：[Vue Web](https://github.com/oublie6/awesome-zero-platform/actions/runs/45945877802)、[Fair Doudizhu](https://github.com/oublie6/awesome-zero-platform/actions/runs/45945877795)、[CI](https://github.com/oublie6/awesome-zero-platform/actions/runs/45945877837)。正常 push 自动 CI 的业务检查不作为本次部署输入；本机／部署流程未重建业务代码，复用已验收全部30个原文件。

实施者进行了独立第二遍**自审**（同一实施者，不是独立验收）：重读任务及原则、逐行 diff、跟踪打包成功／异常清理／重复更新／删除／未知文件冲突、路径别名／marker 符号链接／根目录保护，检查 MIME/304/表示缓存隔离、CSP/TLS、API/WSS路由优先级和gzip隔离、内存顺序流式资源成本、Web-only rollback。自审期间调整 marker 读取前的 symlink 检查及禁止根输出，再完成10项测试及运行验证；最终无新缺陷。无新业务模块、接口或数据权威变化。

本任务没有另建执行 Goal、没有新的独立代理验收，Goal0065 保持已完成，Goal0045/0046 保持原状态。未声称移动设备、外部平台或用户真实 SSH 加载速度已验证；未执行专门 parse profiler、远程 registry传输测量、长期压测或故意线上回滚故障。压缩／兼容／缓存／CPU／必要业务冒烟验收范围全部完成。
