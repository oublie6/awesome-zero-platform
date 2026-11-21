# Goal0063 Round 6 最终布局差异化独立验收

2026-11-21。**整体 FAIL。R4-N02 原公共牌遮挡已修复并关闭；新增 R6-N01/P2：真实窄横屏 SHRINK 后字号仍低于候选自身采用的 11 CSS px 下限。暂不满足代码合并前验收条件，不合并 PR #18。**

## 固定基线与范围

候选 `4aa6c4e02b1dc6fe98f991955d8c99ccd21a07a0`，分支 `goal0063-dq-turn-experience`，[PR #18](https://github.com/oublie6/awesome-zero-platform/pull/18) open、未合并。干净 main 经 `git pull --ff-only` 同步；候选用 `git archive` 展开到私有 `/root/goal0063-round6-4aa6c4e/source`，未修改候选源码。

[CI 44841715604](https://github.com/oublie6/awesome-zero-platform/actions/runs/44841715604) 4 个执行 job SUCCESS，runtime/report skipped；[Fair 44841715598](https://github.com/oublie6/awesome-zero-platform/actions/runs/44841715598) 4/4 SUCCESS，33 core / 50 protocol / 147 Cocos 检查通过。实际检出合并测试 SHA `6342f29df3ef436a7c748fd508b05e684a350315`，与候选 `server/clients/scripts/.github` 输入一致。[绑定](0063-round6-evidence/binding.json)、[CI](0063-round6-evidence/meta0.json)。

按 AGENTS、规范原则、current Goal、Goal0063 和 [Round5](0063-round5-independent-acceptance.md)，只检查 `TableGeometry.ts`、`GameTableView.ts`、`table-geometry.test.mjs` 及必要 Cocos 渲染链。归属客户端展示，无协议、服务端、schema、依赖边界或部署变更。服务端/scripts/CI 自 Round3 未变，复用 Round1～5 仍适用的已通过证据；房间输入/账号隔离不重跑。

只执行一次官方固定 Linux CLI `f506fa3…` / Cocos4 engine `d0a696…` Web 导出（146.54s），实际生成声明类型检查、8 JS 审计通过。118 个相关 tracked 输入核对，CLI 仅语义不变地重排 package.json。复用此前已校验 Go 二进制与隔离 QA MySQL/Redis/Beanstalkd、测试 TLS/签名根；新建一局真实三账号公平发牌，只走最少 Bid/Play/Pass，不完成整局。私有回合观察期限仍 5m，客户端原 15s 超时未改。公开运行地址/验证公钥经生成 HTML 配置引用加载，产品 JS 未修改。

## 实际检查

| 条件 | 结果 |
|---|---|
| R4-N02 原 760×390 遮挡 | **PASS，关闭**。真实单张出牌→下一座位 Pass→服务端已提交→浏览器接收端丢响应/通知→原 15s 超时，无退出/重入即出现核对入口；公共牌、提示和按钮分离。 |
| 三尺寸真实投影 | **几何 PASS**。实际 Pass 单张/双张公共牌均检查 760×390、1280×720、390×844；空公共区另有真实 Bid/Play UNKNOWN 窄横屏，其他组合通过自动几何筛选。未用 CSS 尺寸代替逻辑画布。 |
| 操作/手牌/提示边界 | **PASS（所测场景）**。按实际相机投影四角检查公共牌、空桌提示、核对/放弃及普通按钮、旋转手牌边界，无交叠/操作越界。LastPass/同步辅助提示仍沿用前轮的通知优先显示规则，不宣称同时展示。 |
| 真实触摸与结果核对 | **PASS**。双张公共牌的 UNKNOWN 精确重试、单张场景再次超时及新快照后主动放弃均执行；原请求对象完全一致，每条 SQL 动作仅 1 条；核对/放弃后入口清除，无额外普通操作请求。 |
| 字号/行高/文本框 | **部分 FAIL，R6-N01**。桌面实际约 20 CSS px、竖屏 14.93；窄横屏配置折算 11.22，但实际渲染仅 10.69，见下文。 |
| 代码与自动检查 | 20/20 受影响 stock 测试、48 组独立几何组合 PASS；公共牌绘制与避让共用 `tableTrickGeometry`，提示根据实际 gap 定位，行高已独立设置。测试仍只计算配置字号，漏了真实 SHRINK 结果。 |
| 短程渲染/生命周期 | **PASS（有界观察）**。6 次重绘节点数不增长，250ms 时钟保持 1 个，离桌清除；不等于长期泄漏或手机性能资格。 |

[原始运行](0063-round6-evidence/native.json) 的五个 FAIL 均指向同一字号预算，不是五个缺陷；它们在首次字号断言处提前退出该案例。随后 [续验](0063-round6-evidence/followups.json) 明确分开几何/操作与字号结果，完成其余必要交互，`functionalResult=PASS / fontBudgetResult=FAIL / result=FAIL`。退出码 0 仅代表 runner 完成。11 个真实捕获状态的 [边界审计](0063-round6-evidence/audit.json) 也保留字号 FAIL，未降低下限或改产品来得到 PASS。

| 实际 UNKNOWN 页面 | 760×390 | 1280×720 | 390×844 |
|---|---|---|---|
| 单张公共牌 | [图](0063-round6-evidence/single-unknown-760x390.png) | [图](0063-round6-evidence/single-unknown-1280x720.png) | [图](0063-round6-evidence/single-unknown-390x844.png) |
| 双张公共牌 | [图](0063-round6-evidence/multi-unknown-760x390.png) | [图](0063-round6-evidence/multi-unknown-1280x720.png) | [图](0063-round6-evidence/multi-unknown-390x844.png) |

单张窄横屏实际逻辑画布 **1280×656.8421，u=1.35**。投影后公共牌 y=152.73–209.97，提示 y=216.33–230.76，核对按钮 y=235.57–264.43 CSS px；上、下净距约 **6.37 / 4.81px**，无重叠。[完整节点边界](0063-round6-evidence/followups.json)、[再次超时](0063-round6-evidence/single-second-timeout-760x390.png)、[放弃后](0063-round6-evidence/single-discarded-760x390.png)。

## R6-N01 / P2：字号保障只验证配置值，实际 SHRINK 仍低于下限

定位候选 `TableGeometry.ts:137–140`、`GameTableView.ts:259–263` 及通用 `label()` 的 `Overflow.SHRINK`；新增测试 `table-geometry.test.mjs:244,275–278`。

**最小复现：** 官方 Web、760×390，任一真实 Bid/Play/Pass 响应丢失并等待 15s；读取实际渲染后的 `TableNotice`。逻辑字号 **18.9**、行高 **21.6**、文本框高 **24.3**，但 Cocos `Label.actualFontSize=18`。通过相机四角投影测得比例 **0.59375**，实际字体 **18×0.59375=10.6875 CSS px**。空桌、单张、双张公共牌均复现。

根因：当前 pinned Cocos 的 `text-processing.ts::_calculateShrinkFont` 对 SHRINK 结果执行整数转换；设置行高并不能避免该路径。新增测试只断言 `notice.font*cssScale>=11`（配置值为 11.221875），以及行高不超过文本框，未验证 `actualFontSize`。**11px 是候选现有回归测试采用的保护门槛，并非本轮新增标准。** 截图无文字裁切，本报告不声称文字完全无法辨认；失败点是实际字号未达到候选声明的下限，不能据配置值测试宣称保障已实现。

**交回 ChatGPT 的最小修复：** 保留已通过的 gap 避让与共用公共牌几何；选取字号/行高/框高时计入真实 SHRINK 整数结果，保证本投影实际字号达到现有下限（此比例至少需实际 19 逻辑 px）；在真实 Cocos 页断言 `actualFontSize×投影比例`，再复验窄横屏空区/单张/多张及核对入口。不要通过降低 11px 断言掩盖差异，也不需要重跑服务端矩阵。

## 范围与交付

未运行完整牌局、MySQL ACK 故障、DQ 调度、终局恢复、全量 Go Race/性能或旧房间输入矩阵。三尺寸完整 Bid/Play/Pass 交叉并非全部重新交互：真实关键场景如上，其余复用未变路径与自动几何筛选，不扩大 PASS 范围。没有微信 SDK/手机真机、长期性能或 Kubernetes 验收。

后续独立测试部署仍需备份/恢复、增量 schema 预检、双队列持久化及故障、升级/回滚、历史数据与密钥/TLS 保留检查；微信真机仍需触摸、安全区、前后台和网络切换。代码验收不等于部署或发布验收。

[清理](0063-round6-evidence/cleanup.json)：仅中止本轮部分 QA 手牌并停止自有 API/Web/浏览器及 5 个专用依赖容器；数据/卷/保护材料保留，未操作现有 Kubernetes。首次会话准备早于 API Ready 导致空响应，等待 Ready 后正常完成；不计产品缺陷。所有其他真实失败均保留。报告自检核对源/CI/产物、原与实际字号、投影/截图尺寸、断言覆盖和敏感信息；仅提交本报告及必要证据到 main，不改 current Goal、历史报告或产品代码。

**最终：R4-N02 原遮挡 PASS；Round6 整体 FAIL，剩余 R6-N01 一项 P2。PR #18 暂不可合并；未合并、未部署。**
