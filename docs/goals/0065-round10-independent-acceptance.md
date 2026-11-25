# Goal 0065 — Round 10 独立功能验收

结论：**FAIL，2 个 P2；不修改产品代码，不部署候选版本。** Goal0065 保持 `in_progress`，Goal0045/0046 状态不变。验收基线为 main `4a5650ac0d2e3f228c6d5ea1eb938dc7aab23e0e`，增量基线 `a0e6741acf7e5bdd88b2b2c389ca1973e0e13c72`。执行日期：2026-11-25 UTC。

## 新发现

### G65-R10-N01 / P2：取消旧登录后，共享 pending 标记仍阻塞新会话恢复

位置：`clients/packages/game-client-core/src/session.ts:126`（loginWith）、`:157`（releaseToken）、`:181`（dispose）；相邻 WebAuth 在 `clients/game-web/src/auth.ts` 的 pendingLogins/restore/logout 链。

复现：同一 MiniVault 后端保留一份服务器仍认可的凭证。A 开始平台登录，SDK 回调暂不返回；随后对 A 执行 dispose、logout 或 discardLocal。新建 B，用同一 vault 显式 restore。实际 B 返回 false，身份验证请求为 0，captureAttempt 也为 false；通过实际 Controller.restoreAndConnect 复现时，身份和连接均未建立。释放已经取消的 A SDK 回调后，A 不再 exchange，但相同 B 恢复立即成功。

原因：pendingLogin 仅在旧 loginWith 的 finally 清除。取消使旧业务所有权失效，却没有释放其共享恢复屏障；新 Controller 的正常启动回退也会被 false attempt 阻止。WebAuth 的取消待定登录具有同类窗口。

预期：取消自己的业务应释放自己的 pending 标记，同时不能清掉另一个较新登录的标记。正在进行且未取消的新登录阻止旧凭证恢复是正确行为，本报告未将其判为缺陷。

影响：取消/销毁旧页面后，合法新页面恢复依赖旧平台任务返回。MiniAuth.acquireCode 未设置客户端超时；本轮只用受控 SDK 延迟证明依赖关系，不声称实际微信 SDK 永不返回。Web HTTP 有 12 秒期限，不将其描述为无限等待。

证据：[session-lifecycle.json](evidence/0065-round10/session-lifecycle.json)（5 PASS / 4 FAIL，四个失败单元属于同一问题）、[controller-cancel-restore.json](evidence/0065-round10/controller-cancel-restore.json)。驱动使用真实编译的 Session、Controller、MiniAuth/MiniVault、WebAuth，控制 SDK/存储/HTTP 回调；没有手机或真实平台 SDK 验收结论。

### G65-R10-N02 / P2：“取消换绑”后，旧 SDK 回调仍提交换绑并清理身份

位置：`clients/fair-doudizhu-cocos/assets/scripts/bootstrap/GameBootstrap.ts:487`（WechatCancelRelink）、`:898`（relinkWeChat，current 仅检查 currentAccount）。

复现：已有有效身份的 A 开始换绑，暂停 SDK 回调。调用生产 Bootstrap 实际构造的“取消换绑”菜单 action，页面回到 rooms；此时 HTTP 请求为 0、身份仍有效。再释放旧 SDK 回调。实际继续发送 `/auth/wechat/prepare` 和 `/auth/wechat/relink`，显示“微信换绑成功”，并 detach 当前身份。

原因：取消菜单只 renderScreen；传入 Controller 的 owned 回调仍认为同一账号有效，没有结束此次换绑业务。Controller 对调用方传入的所有权判断本身有效，但页面取消没有撤销这个所有权。

预期：用户取消后，不再发起尚未派发的换绑步骤，也不因该旧结果清理当前身份。此问题不要求撤回已发送的服务端操作；复现中取消时尚无 HTTP。

证据：[bootstrap-identity.json](evidence/0065-round10/bootstrap-identity.json)（11 PASS / 1 FAIL）及同名驱动。使用实际编译 Bootstrap 菜单构造、Controller、Session 和独立 receiver，控制呈现端口及 SDK/HTTP；未修改真实微信绑定。正常绑定、注册绑定、解绑、换绑均有正向对照，不是靠拒绝全部请求通过测试。

## 本轮实际执行结果

| 检查 | 结果与范围 |
| --- | --- |
| 受影响源码测试 | 287 PASS：core46、Cocos192、Vue20、构建脚本29；无跳过。core/Cocos 类型检查、Vue vue-tsc/Vite 均通过。见 checks.json 和各日志。 |
| 官方 Linux Cocos Web | 只导出一次，CLI f506fa3de403b48b4833334fa48dddd27bb26364、引擎 d0a696cfe63fd8d19c70d7913ba11aa641c2e3c7、4.0.0-beta.0。CLI 报告 1分4秒；含预构建、严格原生声明和8份 JS AST 审计的整个阶段214.185秒。无 Wine，非发布资格认证。 |
| 四项候选 CI | 均 SUCCESS；Cocos linux_source SUCCESS，linux_native_export SKIPPED；本机真实导出是独立证据，不将 skipped 算通过。完整任务链接见 ci.json。 |
| Round9 N01 | 原失效 grant 的 unlink/relink 和换号后旧 SDK 继续请求均已阻断，PASS。 |
| 统一身份服务 | provider-matrix 36 PASS：资料、绑定、注册绑定、解绑、换绑正常路径，重复调用、各 await 的所有权失效、账号替换/销毁、前置绑定及后续真实 Controller 登录。 |
| 四种状态 | 本地凭证不等于身份；验证后 grant 不等于连接；已建立身份后断连仍可读身份资料但不能发游戏命令；WSS 启动失败不发布游戏就绪。 |
| Round8 N01 | 两条原 Controller 登录完成链 PASS，旧任务不替换新连接或清掉其待定命令。 |
| Round8 N02 | joinRoom/reconnectAndResync 两个最终成员读取窗口 PASS；受控私有读取超时不给出就绪。未将历史其余矩阵声称为本轮重跑。 |
| Vue 页面生命周期 | 正常退出网站/重新进入游戏复用同一 iframe，新增静态请求0。真实子 frame 存储事件移除旧 frame，保留新 token、无自动 revoke；重新加载恢复替换账号。真实隐藏→前台通过 raw CDP 验证，同样移除旧 frame，旧消息无效。 |
| 启动失败 | 真实候选浏览器+Go身份，受控关闭 WSS；9027ms 内 SESSION_FAILED/BOOT_FAILED，hostRestoring结束，无 SESSION_ACTIVE；恢复通路后显式重试正常。并非实际 Go502 测试。 |
| 补齐最终场景观察 | 实际最终场景 await、最终权威成员读取两个窗口均等到真实 confirmStartupAuthority 结束，各私有读1次并超时；无错误就绪。随后显式重试采用同一已发牌手牌。见 startup-gates.json。 |
| 补齐上一局取消通知 | 真实 WSS hand.cancel 及其 changed 通知捕获成功；H2 采用后回放 H1 原取消通知，不改变 H2、不新增读取、不抬高版本门槛。两局业务取消并关闭自有房间。 |
| 缓存/版本/重复进入 | 实际 HTTPS：HTML revalidate、配置/manifest no-store、hash JS一年immutable、cc.js revalidate；条件请求304/0字节，身份 no-store。真实 settings 仅增加合法空白后 body/ETag 改变，最后恢复原 SHA。本轮一轮 iframe 复用；完整冷热/内存多轮指标复用历史。 |

证据目录：[evidence/0065-round10](evidence/0065-round10/)。报告中的受控接口测试不替代真实平台账号写入、手机 Chrome、微信真机、键盘输入或发布验收。本轮遵照范围不调查手机黑屏。

## 三账号真实业务回归

**PASS，真实三账号同一牌局 COMPLETED/v61，进程 exit0，90条步骤记录，pageerror0。** 见 [game-summary.json](evidence/0065-round10/game-summary.json)、[r10-live.json](evidence/0065-round10/r10-live.json) 和 game-exit.json。实际通过自动发牌、真实300秒自动操作、关联私有同步、ACK/快照两种顺序、同牌局刷新、原 RequestID 的 UNKNOWN 重试、WSS 断开重连、非法牌型服务端拒绝、三账号终局归档及个人字符串正反校验。已发送的房间 ready UNKNOWN 通过同一 iframe 返回后保持原请求重试。

三个原业务浏览器上下文逐一注销：iframe0，旧 bearer 再请求401；随后换号，房间/私有手牌/公平状态为空。三人下一局 ready 菜单无重复，自有房间按业务流程退出并关闭，未再发新牌。skipDQ 只避免第二次重复等待，真实自动期限已在 live probes 执行。控制台3条401均来自预期的注销后旧凭证拒绝检查，pageerror为0。原始手牌、个人字符串、令牌和截图不发布，保留脱敏投影。

## 产物、环境及历史复用

环境：Node22.23.4，Chrome for Testing156.0.8078.4，Xvfb有头桌面浏览器/SwiftShader、受信任TLS，1280×900。详见 environment.json。所有重型阶段顺序执行，未新增并行验收代理。

执行入口：隔离候选快照下 core/Cocos 的 `npm run typecheck` / `npm test`，Vue的 `npm run build` / `npm test`，两个构建脚本的 `node --test`；完整命令、工作目录及退出码见 checks.py/checks.json。`native-build.cjs` 调用固定官方 CLI `build --platform web-desktop --build-config`，并执行严格声明及AST审计。独立复现入口为 `node session-lifecycle.mjs`、`node controller-cancel-restore.mjs`、`node provider-matrix.mjs`；浏览器驱动 `bootstrap-identity.cjs` 等连接自有CDP。运行需要同候选编译产物及私有自有QA账号配置，发布证据不含这些凭证。

候选源码隔离快照检查1360个普通文件：1359字节一致，Cocos package.json 仅官方 CLI 调整 JSON 排列且语义相同；无其他差异。详见 source-identity.json。产物文件 hash 见 artifact-hashes.json。

真实浏览器使用当前候选唯一 Web 产物及匹配 API18870，不是旧集群 API。Round9 与候选 server tree 同为 `a8f14fec9d7703c0fe68ac286bbd4aa9081fefbe`；复用 API 二进制 SHA256 `dc50adc3bdc3fabe04b64e0a66f417df44a1fb56fc35a2f3f225e9cdb714c000`，没有重新构建 Go。复用自有隔离 MySQL/Redis/队列，独立 auth 前缀 `g65r10:authn:session:`、队列 `g65r10-turn`。数据库是复用的 goal0065r9，不声称新建空库；开始 ABORTED19/COMPLETED12/未完成0，配置和凭证仅留私有目录。

Go、schema、协议与缓存机制增量未变；按本轮授权复用 [Round9报告](0065-round9-independent-acceptance.md) 及其固定来源证据中的 Go单元/真实MySQL联合读与PK/并发、后端压力/性能、崩溃恢复和完整冷热/内存矩阵。本轮没有再次执行这些测试，也不把历史数值记为本轮测量。协议51项未重跑；本轮287不含它们。自动操作仍使用原300秒期限；没有为缩短测试改变后端配置。

## 观察器与第二遍自审

测试驱动是验收工具，不是产品修复。provider-matrix 初始语法括号错误未执行用例；随后替换账号误复用悬停 SDK 造成驱动等待，修正为独立 SDK 后36项完整通过。Vue 初始按钮精确名称遗漏箭头，改为匹配实际按钮；Playwright焦点覆盖使前置 document.hidden 不成立，vue-lifecycle.json 保留该观察器失败，另以独立 raw CDP 同窗口真实隐藏/前台完成替代证据。这些不计为第三项产品缺陷。发布日志仅规范化行尾空白，原始日志私有保留；manifest-sha256.json 固定发布文件哈希。

第二遍审阅范围：候选增量与共享状态、登录取消和旧 finally 所有权、各身份 HTTP await、Bootstrap实际取消菜单、失效 grant 与页面代次、Vue存储/可见性、最终权威等待真实完成、旧手牌通知、正常业务正向对照及证据分类。两项失败均有最小稳定复现及正常对照；未将受控平台接口说成真实平台写入。只写报告、证据和 current.md 允许的状态段，不修改产品代码或验收条件。

## 部署与收尾

因存在2P2，本轮**未部署**，未触发 API→Web 上线或 Web→API 回滚。集群保持此前已验收的 API/Web；部署、Pod UID/镜像/重启数、配置、Service、PVC、StatefulSet对比未变，六Pod均Ready，业务库 ABORTED14/COMPLETED11/未完成0。Secret hash不变、两节点Ready、用户29457服务active。隔离库结束 ABORTED21/COMPLETED13/未完成0；增量为两场取消观察用局和一场完整牌局。自有API、浏览器、临时Nginx及此前停止状态的四个隔离依赖均已停止，18870/19181/30471端口关闭，保留隔离数据以便复现。见 final-audit.json、audit-extra.json、owned-cleanup.json。

未宣告 Goal0065 完成；修复交回 ChatGPT。Goal0045/0046 的既有未验收项、设备/平台凭证和发布门禁保持原状态。
