# Goal0063 Round 7 最终字号与布局独立验收

2026-11-21。**PASS。R6-N01/P2 已关闭，R4-N02 遮挡未复发；本轮未发现新增阻塞问题。固定候选满足 Goal0063 代码合并前验收条件，PR #18 可以进入合并；本轮未合并、未部署。**

## 基线与证据复用

候选 `8b9465d87ea3d3ca6893de61aeb33e21a7c78dae`，分支 `goal0063-dq-turn-experience`，[PR #18](https://github.com/oublie6/awesome-zero-platform/pull/18) open、未合并。main 初始干净，已 `git pull --ff-only`；候选仅以 `git archive` 展开到私有 `/root/goal0063-round7-8b9465d/source`，未修改候选代码。

[通用 CI 44846863573](https://github.com/oublie6/awesome-zero-platform/actions/runs/44846863573) 4 个执行 job SUCCESS，runtime/report skipped；[Fair 44846863630](https://github.com/oublie6/awesome-zero-platform/actions/runs/44846863630) 4/4 SUCCESS，33 core / 50 protocol / 149 Cocos 检查通过。实际 checkout 为合并测试 SHA `de4cefda5e934b5a283b5aaeca48d48506ba877a`，与候选 `server/clients/scripts/.github` 输入一致。[CI](0063-round7-evidence/runs.json)、[源码/产物绑定](0063-round7-evidence/binding.json)。

依 AGENTS、current Goal、Goal0063、规范原则及 [Round6](0063-round6-independent-acceptance.md)，仅检查三个客户端修改文件及必要 Cocos 字体装配链。归属客户端展示，无协议、服务端、schema、数据权威或依赖边界变更。服务端/scripts/CI 自 Round3 未变，复用 Round1～6 仍适用的已通过证据，不重跑故障、DQ、MySQL ACK、完整牌局、全量 Race、性能或房间输入隔离。

只导出一次当前官方 Linux Web：固定 CLI `f506fa3…` / engine `d0a696…`（Cocos4.0.0-beta.0），130.81s，实际生成声明类型检查及 8 JS 审计通过。118 个相关 tracked 输入核对，CLI 仅语义不变地重排 package.json；复用已验证 Go 二进制及隔离 QA MySQL/Redis/Beanstalkd、测试 TLS/签名根。仅给生成 HTML 加既有公开运行配置引用，产品 JS 未修改。新建一局真实三账号公平发牌，最少 Bid/Play/Pass，不完成整局；私有回合观察期限 5m，客户端原 15s 超时不变。

## R6-N01：真实字号 PASS

真实 Cocos 节点渲染完成后读取 `fontSize / actualFontSize / lineHeight / UITransform`，将四角经实际相机投影到 CSS 坐标，使用 **actualFontSize×投影比例**，不以配置值或 Mock 代替。13 个 UNKNOWN 捕获状态均达标；空公共区 Bid/Play、单张和双张公共牌 Pass 均覆盖，三视口均实际检查。

| 视口 | fontSize / actualFontSize | lineHeight / 框高（逻辑） | 投影比例 | 实际 CSS 字号 |
|---|---:|---:|---:|---:|
| 760×390 | 21 / 21 | 22.95 / 24.30 | 0.59375 | **12.46875** |
| 1280×720 | 21 / 21 | 24.30 / 32.40 | 1 | **21.00** |
| 390×844 | 50 / 50 | 59.0769 / 78.7692 | 0.3046875 | **15.234375** |

760×390 实际逻辑画布 **1280×656.8421，u=1.35**。UNKNOWN 全文“操作结果未知，请核对原请求”保留；CLAMP 下实际字号等于配置字号，无意外 SHRINK。目标文字未换行，字形宽高在文本框内，结合实际截图复核未见 CLAMP 剪字或缺失。11px 下限未降低。[字号与字形测量](0063-round7-evidence/font-summary.json)、[节点证据](0063-round7-evidence/resume.json)。

| 真实 UNKNOWN | 760×390 | 1280×720 | 390×844 |
|---|---|---|---|
| Bid / 空公共区 | [图](0063-round7-evidence/hand.bid-0cards-unknown-760x390.png) | [图](0063-round7-evidence/hand.bid-0cards-unknown-1280x720.png) | [图](0063-round7-evidence/hand.bid-0cards-unknown-390x844.png) |
| 单张公共牌 Pass | [图](0063-round7-evidence/single-pass-unknown-760x390.png) | [图](0063-round7-evidence/single-pass-unknown-1280x720.png) | [图](0063-round7-evidence/single-pass-unknown-390x844.png) |
| 双张公共牌 Pass | [图](0063-round7-evidence/hand.pass-2cards-unknown-760x390.png) | [图](0063-round7-evidence/hand.pass-2cards-unknown-1280x720.png) | [图](0063-round7-evidence/hand.pass-2cards-unknown-390x844.png) |

## 同类控件与交互

以下为所测 UNKNOWN 场景中的**最小实际 CSS 字号**。RoomVersion 的编号缩略是有意规则，生成字符串及版本后缀完整落在框内；不要求把完整房间 ID 塞进该信息栏。KittyLabel、“核对上次操作”“按新快照放弃”全文完整。相关 EmptyTrick / LeadingSeat / OwnSeatInfo 均量测并复核，无本轮新裁切或可见遮挡。

| 控件 | 760×390 | 1280×720 | 390×844 |
|---|---:|---:|---:|
| RoomVersion | 11.875 | 20.00 | 14.3203 |
| KittyLabel | 11.875 | 20.00 | 14.9297 |
| 核对 / 放弃按钮文字 | 14.84375 | 25.00 | 18.5859 |

真实 Bid/Play/Pass 均保持普通操作进行中禁用、UNKNOWN 核对入口出现；三类型精确重试对象一致，SQL 每条动作仅 1 条，没有额外普通操作请求。单张 Pass 再次超时仍有入口，取得新权威快照后实际点击放弃；双张 Pass 核对成功后控件/提示正常更新。[初段操作](0063-round7-evidence/native.json)、[续验](0063-round7-evidence/resume.json)。提示、公共牌、手牌和操作按钮的实际投影边界无交叠，R4-N02 不复发；辅助提示沿用前轮的通知优先级，不宣称被隐藏的提示同时展示。

22/22 受影响 stock 测试、48 组独立几何检查 PASS，只作辅助证据。真实完成装配的 6 次重绘节点 **61→61**、250ms 时钟 **1→1**，离桌后 **0**；[资源检查](0063-round7-evidence/resource.json) 与 [正常状态截图](0063-round7-evidence/settled-normal-status-760x390.png)。不将短程检查扩展为长期泄漏/性能结论。

## 运行边界与最终交付

原 runner 在两个已通过案例后以 **143** 结束，未写完成状态，原因未确认；API Ready200，原 Cocos 页面及原 Pass pending/requestId 保留。按 [现场状态](0063-round7-evidence/interruption-state.json) 续验同一请求，未重建牌局或重跑已通过案例。续验末次资源断言在标签刚重建、装配未完成时读到 actualFontSize=0，原始 FAIL 保留；随后在同一页面等待当前标签装配完成再测，六次均正常完成并达到字号要求，计时器释放通过。未降低断言、伪造渲染值或修改产品；不能把该预渲染采样当成实际显示字号为零，也不把原 runner/续验进程的退出码直接当整体 PASS。

报告自检重新核对固定源/CI/构建、实际与配置字号、CLAMP 字形范围、原始失败归因、三尺寸截图及操作唯一性；公共 JSON 只去除重复截图子树及无关背景节点，原始私有数据哈希保留，无测量/结果改写。没有额外独立审阅者，不冒充第二次产品验收。

结合 Round1～6 的已通过且源码仍适用的证据，本轮关闭最后一项 P2，**固定候选满足代码合并前验收条件，PR #18 可合并；没有新增阻塞缺陷或待修复清单。** 这不完成整个部署/发布 Goal，也不改变现有 current Goal 或历史报告状态。

后续须独立完成 Kubernetes 测试部署的备份/恢复、增量 schema 预检、双队列持久化/故障、升级及回滚、历史数据/密钥/TLS 保留检查；微信真机仍须 SDK/真实账号、触摸、安全区、前后台及网络切换验收。本轮不将 Web/CDP 触摸等同真机，不验证完整牌局、服务器故障矩阵、生产容量或长期稳定性。

[清理](0063-round7-evidence/cleanup.json)：本轮部分 QA 手牌中止为 ABORTED，自有 API/Web/浏览器及 5 个专用依赖容器停止，数据/卷/保护材料保留；未改产品、候选分支、PR 或 Kubernetes。仅本报告及必要证据提交 main。
