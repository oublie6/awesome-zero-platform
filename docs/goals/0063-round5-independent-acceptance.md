# Goal0063 Round 5 客户端最终差异化独立验收

2026-11-20。**最终 FAIL。R4-N01 关闭；R4-N02 原按钮文字重叠已修复，但 760×390 的 UNKNOWN 提示仍遮盖公共出牌。暂不满足代码合并前验收条件。**

## 基线与执行范围

固定候选 `5d0f6157cfe724bf960dd66d1f6572787f96a330`，分支 `goal0063-dq-turn-experience`，[PR #18](https://github.com/oublie6/awesome-zero-platform/pull/18) open、未合并。主工作区 `main` 干净并两次 `git pull --ff-only` 确认同步；候选以 `git archive` 只读展开到私有目录，未切换、修改或推送候选分支。

[通用 CI 44720176912](https://github.com/oublie6/awesome-zero-platform/actions/runs/44720176912) 4 个执行 job SUCCESS，runtime/report skipped；[Fair 44720176746](https://github.com/oublie6/awesome-zero-platform/actions/runs/44720176746) 4/4 SUCCESS，33 core / 50 protocol / 144 Cocos 检查通过。实际 checkout 是合并测试 SHA `6f3bd009b77c0a69ca50549dbc1708770971acfb`，与候选 `server/clients/scripts/.github` 输入无差异。[绑定](0063-round5-evidence/binding.json)、[CI 原始记录](0063-round5-evidence/ci-runs.json)。

阅读 AGENTS、current Goal、规范原则、Goal0063、Round4 及相关客户端合同；只审查 Round4 后 6 个客户端文件和必要调用链。归属 Cocos 展示/输入协调，无服务端、协议、schema、部署或跨模块数据边界变更。服务端和构建脚本未变，复用 [Round3](0063-round3-independent-acceptance.md) 终局/恢复及 [Round2](0063-round2-independent-acceptance.md) 调度/故障/完整业务证据，保留各轮原有限制；没有重复服务端矩阵、全量 Race、性能或完整牌局。

本机只导出当前客户端一次：固定官方 Linux CLI `f506fa3…` / engine `d0a696…`（Cocos 4.0.0-beta.0），167.83s；实际生成声明严格检查及 8 个 JS 审计通过。114 个相关 tracked 客户端输入核对无实质差异，CLI 仅重排 package.json。复用前轮 Go 二进制 SHA256 `51cd2a61…5f39c71`，隔离 QA MySQL5.7.44 / Redis / Beanstalkd、真实认证与 HTTPS/WSS。运行配置只补公开地址/验证公钥及生成 HTML 的配置引用，未改产品 JS。私有回合观察期限设为 5m，客户端原 15s 超时不变；不据此重新认定 DQ 时效通过。原始夹具在 `/root/goal0063-round5-5d0f615`。

## 核心结果

| 条件 | 结果与实际证据 |
|---|---|
| R4-N01：建房 UNKNOWN | **PASS**。原生 EditBox 完整中文输入保留，RetryUnknownRoom 即时出现；重试完整 Wire 对象一致，MySQL 同 commandId 仅 1 条记录，实际创建/恢复成功。[现场](0063-round5-evidence/room-create-unknown-1280x720.png)、[操作记录](0063-round5-evidence/native-check.json)。 |
| 准备、退出、开局 UNKNOWN | **PASS（输入保留）**。同上下文原输入不清空；确认退出后清空；开局 UNKNOWN 保留，真正采纳新牌局后旧输入清除。后者另用真实同房间下一手验证，见 followups 最后一项。 |
| 普通刷新、重连、公平刷新 | **PASS**。await 恢复完成后，同手个人字符串在 refreshRoom / refreshFairness / reconnect 均完整保留；localStorage/sessionStorage 无该测试字符串，源码无新增草稿持久化。 |
| 账号、房间、牌局、用途边界 | **PASS（分层证据）**。真实账号切换不带旧草稿；真实 A→B 房间切换后释放 A 的迟到 HTTP 回调，B 草稿不变；真实同房间下一手清空旧个人字符串。用途/handID/离开屏幕后返回另通过真实 Cocos EditBox + 受控元数据重绘检查，不冒充额外服务器牌局。 |
| R4-N02：UNKNOWN 替换普通按钮 | **PASS（已观察三操作/三尺寸）**。仅核对/放弃占用主操作行，提示与这两个按钮分离；实际字号约 20.25 / 15 / 12.02 CSS px。 |
| 真实触摸、重复超时、成功核对 | **所测范围 PASS**。桌面 Bid/Play/Pass、手机 Play/Pass、窄横屏 Play 真实触摸核对成功；重发完整对象一致、SQL 动作各 1 条，无额外普通操作请求；桌面 Bid/手机 Play 再次超时仍有入口，核对后按钮清理、提示更新。[续验](0063-round5-evidence/resume.json)。 |
| R4-N02：提示不遮挡公共出牌 | **FAIL**。760×390 Pass UNKNOWN 提示与公共牌重叠，见下文。桌面/手机对应所测页面通过。 |

关联回归：[followups.json](0063-round5-evidence/followups.json) 7/7 PASS，包括旧控制器延迟 ID 完成后不发请求、不重绘新账号；12 次原生表格重绘节点数不增长，250ms 时钟始终 1 个，离桌后 0 个，之后原生输入仍可编辑。所测 in-flight→UNKNOWN 每次仅新增 1 次场景 render。另执行严格 shell 编译及受影响 stock 测试 **25/25 PASS**；未用这些纯函数/源码断言代替真实界面。

临时通知显示时，LastPass 和 ServerSyncNeeded 被代码主动隐藏，采用提示优先级，并非同时布局通过。公共牌仍显示，因此不能用隐藏辅助提示抵消下面的真实遮挡。

| UNKNOWN 实际截图 | 1280×720 | 390×844 | 760×390 |
|---|---|---|---|
| Bid | [图](0063-round5-evidence/hand-bid-unknown-1280x720.png) | [图](0063-round5-evidence/hand-bid-unknown-390x844.png) | [图](0063-round5-evidence/hand-bid-unknown-760x390.png) |
| Play | [图](0063-round5-evidence/hand-play-unknown-1280x720.png) | [图](0063-round5-evidence/hand-play-unknown-390x844.png) | [图](0063-round5-evidence/hand-play-unknown-760x390.png) |
| Pass | [图](0063-round5-evidence/hand-pass-unknown-1280x720.png) | [图](0063-round5-evidence/hand-pass-unknown-390x844.png) | [FAIL](0063-round5-evidence/hand-pass-unknown-discard-760x390.png) |

## R4-N02 残留 / P2：窄横屏提示遮盖公共牌

定位候选 `bootstrap/TableGeometry.ts:106–111`，调用处 `GameTableView.ts:259–262`。这是 R4-N02 同族残留，不重复报成两个缺陷。

**最小复现：** 官方 Cocos Web 设置 760×390 → 真实地主出一张牌 → 下一座位点击“不出” → 服务器已提交，仅在浏览器接收端丢该响应及后续 hand 通知 → 等原 15s 超时。无需重入或缩放即出现核对按钮，但黄色“操作结果未知，请核对原请求”压在中央公共牌下缘。[原始现场](0063-round5-evidence/hand-pass-unknown-discard-760x390.png)、[定尺寸复核](0063-round5-evidence/overlap-confirmed-760x390.png)、[实际坐标](0063-round5-evidence/overlap-760x390.json)。

实测提示 y 范围 **199.50–218.74 CSS px**，公共牌 **152.73–209.97 CSS px**，矩形交集约 **40.88×10.47 CSS px**，截图亦可见文字覆盖牌面。复核仍使用同一原生 pending 请求，无新建 Mock 投影或再次发牌。

**根因：** `tableNoticePlacement` 用 `height<=480` 判断短横屏，但接收的是 Cocos 逻辑尺寸。该真实页面为 **1280×656.8421、u=1.35**，因此误走 `+51*u / 24*u` 普通分支，而非紧凑间距。新增几何测试直接给 760×390，未覆盖真实相机缩放后的逻辑尺寸；纯函数 PASS 漏掉了这条路径。

**交给 ChatGPT 的最小修复清单：**

1. 根据实际 Cocos 可用空间/缩放或公共牌与按钮边界选择提示间距，避免仅以 CSS 风格的绝对高度阈值判断逻辑坐标；继续保留 UNKNOWN 替换操作行。
2. 回归覆盖真实 1280×656.8421 逻辑尺寸、公共牌已存在的 Pass UNKNOWN，断言提示同时避开公共牌和核对按钮。
3. 用真实 760×390 Cocos 页面复验提示、再次超时及核对/放弃入口，补齐受布局修复影响的尺寸检查。无须重设计服务端或重跑终局、DQ、全量 Race/性能。

## 证据解释、未验证范围与收尾

[夹具记录](0063-round5-evidence/fixture-notes.md) 保留初始 TLS/运行配置准备失败、错误相对路径，以及两类被撤回的夹具 FAIL：开局回执清除早于新手采纳时误断言输入应清空、未经过相机投影的几何误判。原始 native-check JSON 不改标签；该 runner 为纠正坐标而退出 -15。随后独立原生边界检查及正确屏幕坐标续验区分了夹具问题与上述真实产品 FAIL。resume 进程 exit0 仅表示运行结束，其 Pass760 断言明确 FAIL，不当成通过。

本轮手机/窄横屏 Bid 完成了真实提交、UNKNOWN 展示及布局观察，精确核对因早期夹具中断未补跑；窄横屏 Pass 的后续核对/放弃点击未执行。确认真实遮挡后未启动预备的额外 Bid 牌局检查；不将这些未执行项写成 PASS。Round4 已通过的动作语义只在原证据范围复用。

未重验完整牌局、终局/MySQL ACK 故障、DQ、全量 Race、性能、长期资源泄漏、微信 SDK/真机或 Kubernetes 部署。独立测试部署仍需备份与恢复、增量 schema 预检、双队列持久化/故障、升级及回滚、历史数据/密钥/TLS 保留验证；真机触摸、安全区、前后台和真实网络仍需独立完成。

[清理记录](0063-round5-evidence/cleanup.json)：本轮 API/Web/浏览器和 5 个专用依赖容器停止，数据/卷/保护材料保留。两个临时手最终均 ABORTED；第二手在可选清理前已终止，清理请求 accepted=false 的夹具断言保留，未重复操作。未改产品、合并 PR、部署或修改既有 Kubernetes 环境。仅报告和必要证据提交 main，current Goal 与历史报告不变。报告自检核对源/CI/产物、真实截图尺寸、失败归因、覆盖范围及敏感值；不冒充额外独立审阅。

**最终：R4-N01 PASS；R4-N02 FAIL；代码合并前验收 FAIL，部署/发布不放行。ChatGPT 统一修复上述一项 P2 后交回差异化复验。**
