# Fair Doudizhu — Goal 0045/0046 交付缺口审计与集中收口

审计日期：2026-11-08。执行者：ChatGPT（开发者自审，**非 Codex 独立验收**）。

审计代码基线：`2c39db7fc94e476441787b62fbd4f42b2e54f3e2`。
上个已独立验收的**专项产品 SHA**：`fcca57d6f29b724cb193c3d652a1bd2a7cd230ca`（[Goal0054 PASS](../goals/0054-independent-acceptance.md)，只代表该专项通过）。

依据：`AGENTS.md`、`docs/principles/01–04`、`docs/goals/current.md`、`docs/goals/0046-integrated-fair-doudizhu-delivery.md`、Goal0051–0054 及既有 Round 18/19 业务证据。旧报告中的 FAIL/BLOCKED 是**历史固定候选结论**，不会被重写成当前版本 PASS。

## 1. 交付判断方法

严格分开四层：

- **产品源码已实现**：代码与 API/组件契约存在，不意味着运行成功。
- **已有可复用证据**：有明确受验版本、执行方式、真实环境与边界；未修改的 Go 端可沿用历史独立证据，必要时复核提交树与协议。
- **当前版本待验证**：新提交或新的 Cocos4 工具链没有等价的独立构建/行为证据，不计 PASS。
- **外部条件受阻**：AppSecret、域名、真机、受管 Linux 构建节点、正式发布条件不可凭源码或模拟 SDK 替代。

特别约束：新局使用**玩家字符串 + 客户端安全随机数，单次 HPKE 传给服务器加密托管**；服务端负责长期存储与恢复；客户端不保存每局状态或贡献备份。历史两阶段恢复 kit 仅供旧局，不得成为新局开始的必备门槛。

## 2. Goal0046 八项交付映射

| 交付要求 | 当前代码和证据 | 剩余内部/外部门槛 |
|---|---|---|
| GOAL1 身份与会话 | `GameOnlineController`、`GameSession`、`WeChatIdentityExchange`、Go 认证/AccountID 已存在；真实 Go 测试账号与授权隔离在 Goal0054 覆盖 | 真实 `wx.login`→服务端换证需受管 AppSecret/批准域名和 SDK 环境；不能将 Go 测试登录冒充微信账号验收 |
| GOAL2 房间和 UI | Go room API/WSS；`RoomCoordinator`、`GameBootstrap` 的 create/join/ready/leave、发现/恢复、UNKNOWN 精确对账已多轮验证 | 真实 Creator4 导出后的 UI/触控、拥挤菜单和移动安全区仍需运行层验证；本轮修复分页 |
| GOAL3 公平机制 | 新贡献模式：本人字符串、客户端 CSPRNG、HPKE、服务端加密托管；历史两阶段协议兼容；Goal0054 真实三份贡献与旧局交错 | 新模式整套真实 SDK 牌桌流程/终局本人核验需当前客户端端到端覆盖；不回到客户端备份方案 |
| GOAL4 可玩完整牌局 | Go 真实三人局、Bid/Play/Pass、终局证据有历史完整局证据；`GameBootstrap` 实际牌桌操作入口已接入 | 在**当前生成的客户端**中完成从房间到结算的可点击/可读场景；UI 实物布局和终局画面不能从 Go 测试替代 |
| GOAL5 B 级恢复 | MySQL 先提交后 ACK、journal/checkpoint、owner epoch/fence、精确 RequestID 恢复及错误根拒绝有历史 Go/实际 DB 证据 | 对最终候选做证据一致性复核，并验证客户端终端重连后的可继续操作；正式增量升级/运维根备份是部署门槛 |
| GOAL6 三人 E2E 与故障注入 | 多轮 3 实际 Go 身份、真实 WSS、进程故障、响应丢失、完整局终局证明结果可复用（需受验 Go 树一致） | 用当前 Cocos 4 原生场景完成整合产品 E2E，或准确标 BLOCKED；不可将旧 HTTP/WSS 独立客户端测试宣称为原生小游戏 E2E |
| GOAL7 性能稳定性 | 历史真实 MySQL 100/1000 局、P50/P95/P99、Race、checkpoint/fork/内存有记录 | 缺少具体目标设备/生产硬件上的长时在线房间+Socket soak、会话内存和发布 SLA；按环境分开评估 |
| GOAL8 多平台构建及平台资格 | 旧 Creator3.8 三目标构建/Web/部分微信工具 SDK 证据；仓库现已固定 Linux CLI4、Cocos4.0 beta，具备原生 exporter 脚本和静态 CI | **当前** `linux_native_export` 为 SKIPPED（仅 workflow_dispatch+批准的 self-hosted runner）；需要实际导出、工具启动、微信 SDK 全业务、域名/真机/分享/前后台，抖音熵与 SDK 资格未证明 |

GOAL8 的微信优先原则不意味着必须同时完成抖音真机；但不能把历史 Creator3.8 的 Douyin 构建结果冒充目前 Linux CLI4 的实物构建或 SDK 资格。

## 3. 本轮内部改动

当前已提交 `GameBootstrap.ts`、纯 TS `MenuPagination.ts` 和对应 Cocos meta、`tsconfig.shell.json`、`menu-pagination.test.mjs`：

- 将房间大厅和牌桌的全量操作缓存为仅内存中的 UI 动作列表；安全区可见按钮分页，提供上一页/下一页或极矮窗口循环翻页。
- 横/竖屏按目标宽高选择可读列数，保持有限按钮高度与间距，防止原实现随行数增加无限挤压触控区域。
- 翻页/缩放只更新呈现控件；不清除房间码输入、本人字符串、选牌、不发出网络命令，不重置 room/hand/controller。
- 编译的纯 helper 测试覆盖紧凑横屏、17 张牌加其他动作、单页、分页范围、缩放越界页码。

这是**源码/单元验证**，不是 Creator4 实物画面或手机验收。本次未修改后端、协议、MySQL、凭证持久化和随机贡献方案。

## 4. 仍然需要执行的收口检查（按顺序）

1. **同 SHA CI**：源代码、完整 Fair Doudizhu、主 CI 均无失败；有失败则 ChatGPT 集中修复。不要因文档更新后的新 SHA 将原受测代码结论混淆。
2. **批准的 Linux 原生生成**：现有 `.github/workflows/cocos-client-smoke.yml` 的 `workflow_dispatch/native_export=true` 需要标记 `self-hosted,linux,x64,cocos-cli4` 的实际节点和已安装固定工具链。用 `scripts/cocos-linux-wechat/build.cjs` 构建并保存输出 SHA/包体、入口脚本及日志。不拿 hosted source check、旧 Creator3.8 产物代替。
3. **真实可操作的 Cocos 场景**：首页→微信登录→房间发现/创建/加入→三人 ready→本人贡献→开始/叫地主/出牌→终局、本人字符串核验→冷恢复；每步检查按钮触达、提示、safe-area、重复点击/UNKNOWN，以及 HTTP/WSS/数据库的真实回执。无实际微信 AppSecret、批准 HTTPS/WSS 或真实 SDK 则如实 BLOCKED，并使用可控夹具仅验证可执行部分。
4. **重新核验当前目标的长时运行指标**：在指定硬件上记录活动房间、WebSocket/连接池、内存/GC、队列、MySQL 提交 P50/P95/P99、恢复时间；没有目标 SLA/硬件就记录环境限制，避免自创阈值。
5. **一次性整体验收交接**：代码/CI/可执行环境已完成的部分由 ChatGPT 集中开发自审后固定 SHA，再由 Codex 对 Goal0045/0046 进行完整独立验收。没有外部资格证据的 gate 均留 BLOCKED，不把局部 Goal0054 PASS 当作全项目 PASS。

## 5. 明确暂缓和禁止项

- 用户已明确将 **C09/C10**（完整旧根→新根迁移/旧根淘汰）移出本次验收，保持 **DEFERRED**，不是 PASS；`rotate` 仍禁用。稳定保护根需管理员独立保管备份，错根必须拒绝。
- 不重新设计服务端权威、Redis/DB 职责、客户端备份或公平随机模型。
- 不上传/发布微信应用，不伪造开发者工具或真机凭证；不修改生产数据库/管理员密钥。
- Goal0045/0046 仍是 `in_progress`；本轮自审或自动化绿色不替代 Codex 独立产品验收。

## 6. 自审与限制

本轮按 `AGENTS.md` 二次审查 UI 分页的空列表、行列容量、翻页循环、页面切换/缩放、文本输入、卡牌选择和按钮事件清理。旧 UI 的问题来自所有按钮挤入安全区底部半屏、行距小于按钮高度，而不是服务端房间状态。纯 helper 行为测试覆盖范围与目标窗口假设，真实 Creator4 排版和字体缩放仍未执行；需要产物运行后观察长中文字和手指点击精度。未声明当前完整版 UI 或 E03 获独立 PASS。
