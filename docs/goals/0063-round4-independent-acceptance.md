# Goal0063 Round 4 客户端差异化独立验收

2026-11-20。**最终 FAIL。R3-N01 的 Bid/Play/Pass 即时核对修复通过；同类验收发现两项 P2：建房 UNKNOWN 丢失输入、超时提示遮盖核对按钮文字。暂不满足 Goal0063 代码合并前条件。**

## 基线与复用

固定候选 `871ac82be1bce7d3e08e5068d7f46e84f920a76d`，PR [#18](https://github.com/oublie6/awesome-zero-platform/pull/18) 未合并。主工作区main干净并已`git pull --ff-only`；候选在独立detached worktree `/tmp/goal0063-round4-871ac82`，无源码修改。

[CI 44678654759](https://github.com/oublie6/awesome-zero-platform/actions/runs/44678654759) 的4个执行job成功、runtime/report skipped；[Fair 44678654772](https://github.com/oublie6/awesome-zero-platform/actions/runs/44678654772) 4/4成功，客户端33 core/50 protocol/139 Cocos检查通过。实际checkout `6cca82f1b22d91626c43baddaae5ecb36a5441f7` 与候选server/clients/scripts/.github输入一致。[绑定证据](0063-round4-evidence/binding.json)。

仅审查Round3后的6个客户端文件及必要调用链，归属Cocos展示/客户端操作协调，无协议、服务端、schema或部署变更。遵守规范原则和Goal0063；服务端源码/scripts/CI相对Round3不变，复用[Round3](0063-round3-independent-acceptance.md)服务端、终局、调度及760×390字号证据，复用其已校验二进制。没有重跑完整牌局、Go/Race/SQL故障矩阵或全量CI。

本地仅有旧SHA的Web产物，故用现有固定官方Linux CLI/engine缓存导出当前客户端**一次**；150个tracked输入核对、实际Cocos4生成声明检查及8个JS审计通过。package.json仅被CLI重排格式。缓存引擎的既有5个资产.meta差异及skybox/FBX工具警告沿用前轮边界；没有新引擎代码或外国OS工具链。真实场景使用Round3专用MySQL/Redis/三队列、QA账号及受信测试HTTPS/WSS，Go后台二进制hash与前轮一致。原始夹具与私密配置位于 `/root/goal0063-round4-871ac82`（E），未对Kubernetes部署或既有业务注入故障。

## 验收结果

| 条件 | 结果与证据 |
|---|---|
| R3-N01：Bid/Play/Pass UNKNOWN即时显示 | **PASS**。真实操作已提交，浏览器接收边界丢该requestId响应及hand通知，保留原15s期限；`needsReconciliation=true/isSubmitting=false`后直接出现两个入口，全程无需离桌、重入、缩放。 |
| 请求进行中保护 | **PASS**。`isSubmitting=true`，普通Bid/Play/Pass无点击处理器，尚不显示核对入口；真实额外点击不增加请求数。超时后普通操作继续禁用。 |
| 精确重试及再次超时 | **PASS**。Bid共3次、Play/Pass各2次传输，去掉观测时间后完整Wire对象deepEqual，requestId/clientSeq/payload/expectedVersion均相同；每个原命令的真实MySQL动作行均为1。Bid再次超时仍有入口；最终真实hand.sync恢复新水位，pending及核对按钮消失。 |
| 新快照后主动放弃 | **PASS**。另一次真实Pass已提交但丢响应；同步新的Go私有投影后pending仍保留，用户实际点击DiscardUnknown才清理，不重复执行动作。 |
| 明确拒绝 | **PASS**。保留一个原生Play的真实出站帧，另一合法请求先推进版本，再放行原帧；Go实际返回CONFLICT，原帧未落动作，pending/isSubmitting清理且不显示UNKNOWN。 |
| 房间UNKNOWN及输入 | **部分FAIL**。已提交的ready/start/leave的RetryUnknownRoom及时加入既有分页菜单、可点击核对清理，UNKNOWN期间输入保留；已提交的create同样有入口，但文字被清空，见R4-N01。实际SQL分别有唯一accepted记录。 |
| 旧异步回调/取消 | **PASS（分层证据）**。真实页面延迟旧控制器的安全ID获取，恢复另一QA账号后释放；旧请求未发送，旧完成通知render次数0，新控制器无pending/in-flight。熵取消、抛异常的展示回调及已授权放弃另有4项独立状态检查。房间/牌局切换的既有约束复用同源CI的跨房间恢复、迟到快照、UNKNOWN与新手互斥用例，并核对beginHandTransition及currentAccount/currentRoom；未重跑全部跨上下文真实矩阵。 |
| 过度渲染 | **所测范围PASS**。独立通知检查每个send/retry首尾各一次；真实三种操作in-flight→UNKNOWN各增加1次场景render，非按帧重绘；权威同步造成的后续render另计。 |
| 核对控件可读性 | **FAIL / R4-N02**。入口现在及时显示，点击也能命中，但错误提示与按钮caption重叠，桌面和竖屏均见。 |

主数据：[原生状态与请求检查](0063-round4-evidence/native-check.json)、[实际拒绝/账号隔离/建房复现](0063-round4-evidence/followups.json)、[房间SQL](0063-round4-evidence/room-sql.json)。新增测试中的源码正则不能证明真实UI状态，故本轮补了上述实际交互和Wire比较；没有用Mock投影替代它们。单独4项受控端口测试的范围见[日志](0063-round4-evidence/action-probes.log)。

| 实际页面 | 桌面1280×720 | 竖屏390×844 |
|---|---|---|
| UNKNOWN入口及遮盖 | [Bid](0063-round4-evidence/hand-bid-unknown.png) | [Pass](0063-round4-evidence/hand-pass-unknown.png) |
| 精确核对后入口消失 | [Bid](0063-round4-evidence/hand-bid-confirmed.png) | [Pass](0063-round4-evidence/hand-pass-confirmed.png) |
| 明确拒绝 | — | [实际CONFLICT](0063-round4-evidence/explicit-rejection-390x844.png) |

## 最小修复清单

**R4-N01 / P2：建房UNKNOWN的直接render绕过输入保留。** 定位 `GameBootstrap.createRoom` catch及新增`notice`。真实原生EditBox先填入并校验 `R4-create-draft-保留`，点击创建，Go已接受该requestId（SQL accepted=1/count=1），响应被丢弃。15s后RetryUnknownRoom出现，但roomInput变为`''`；未切账号、房间或手动刷新。[现场](0063-round4-evidence/create-unknown-1280x720.png)、[状态](0063-round4-evidence/followups.json)。

原因：createRoom catch先直接`renderScreen('rooms')`，销毁EditBox并以roomID重建；随后notice看到roomRetryRendered已经等于当前UNKNOWN，不再执行新增的保存/恢复逻辑。此直接render是既有路径，本轮修复没有覆盖。应在当前owner/room/hand仍有效的范围内保留输入，统一处理UNKNOWN引起的结构刷新；不能持久化个人字符串或把旧账号草稿恢复到新账号。增加真实create UNKNOWN与精确重试的输入回归。

**R4-N02 / P2：即时核对入口与超时提示占用同一行。** 定位 `GameTableView.ts:243–258`。Retry/Discard中心在`controlsY+46*u`、高度36*u；TableNotice中心在`controlsY+51*u`、高度24*u，两者纵向大幅重叠。实际中文caption上叠加了黄色`command timeout; outcome unknown...`，上述Bid/Pass截图均复现；触摸能提交不代表文字可读。布局代码虽未在本轮改坐标，新的即时UNKNOWN刷新使提示和入口同时出现，不能沿用前轮“重入后提示已清空”的图片认定合格。应给提示与核对控件分开的位置/空间，在Bid/Play/Pass及再次失败状态验证两目标视口；不需要重设计服务端。

## 限制、复现与收尾

实际命令：`COCOS_CLI_ROOT=... COCOS_ENGINE_ROOT=... node E/native-build.cjs`；使用其真实生成声明检查；当前客户端shell单次编译后`node --test E/action-probes.mjs`（4/4）。真实页面为`native-check-resume.cjs`及`followups.cjs`，正常退出0；其中后者的产品断言明确记录FAIL，退出0仅代表测试运行结束。没有把进程退出码当验收结论。

[夹具记录](0063-round4-evidence/fixture-notes.md)保留启动CDP未就绪、把Room UNKNOWN观察时间误设3s、自动恢复旧房间导致未发出create、以及peer准备推进版本后首个start被明确拒绝等前置问题。修正观察/前置条件后只续验缺失项，没有修改产品、原15s期限或服务端状态来屏蔽失败。最终已提交的start另有独立accepted记录；最早create sent0的失败不计产品缺陷，R4-N01来自随后有效的真实create。

本轮只做一局最小发牌/少量动作，未完成整局。没有复验终局、容量、长期泄漏、真机/微信SDK或部署迁移/回滚；760×390只复用未变化部分，不宣称本轮重新测试该视口。软件验收与部署验收分别处理；当前FAIL不能进入代码合并前放行，更不能视为部署许可。

[清理记录](0063-round4-evidence/cleanup.json)：收尾仅中止本轮部分QA牌局，停止本轮API/Web/浏览器和复用的5个专用容器，数据/卷/密钥及原始证据保留。未合并PR、未改业务代码、未部署Kubernetes，Goal0045/0046及原报告保持原状。一次收尾自检核对了原条件、真实断言、夹具误判、源/产物绑定及敏感值；这是报告自检，不冒充另一位独立审阅者。
