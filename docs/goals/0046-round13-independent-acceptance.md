# Goal 0046 — Round 13 独立验收

2026-11-06 UTC，Codex。**A：代码修复 PASS；B：Goal 0046 内部完成资格 BLOCKED；C：外部发布资格 BLOCKED。** R12-N01/P2 关闭；完整密钥迁移和退役未通过，Goal 0045/0046 保持 `in_progress`。

## 版本与范围

从 clean main 的 Round 12 报告提交 `28eb7a01d5b08ac68c173c113790f51687744b52` 执行 `git pull --ff-only origin main`，实际固定候选为 **`14748096aa34e9d423ffcd48141988646b4bdd41`**。该 SHA 与历史候选恰好一致，但经重新同步和查询确认，未直接采用历史状态。

精确 SHA 的 [CI 42182382588](https://github.com/oublie6/awesome-zero-platform/actions/runs/42182382588) **6/6 SUCCESS**、[Fair Doudizhu 42182381912](https://github.com/oublie6/awesome-zero-platform/actions/runs/42182381912) **4/4 SUCCESS**；独立读取全部 job/step 元数据，见 `ci.json`。没有重跑 CI，也不使用报告提交触发的 CI 代替产品证据。

核对 AGENTS、适用原则、current、0045/0046 正式要求、Round 10/11/12 和保护根运维文档。与 Round 12 产品 `6c158868cfda18c2f2e82c910d89545f7ff9f815` 比较，实际变更为维护 CLI 三个 Go 文件及四个文档，共七文件。产品源码、正式测试和历史报告没有被验收方修改。

| 变更/调用边界 | 影响及本轮验证 |
|---|---|
| maintenance/main.go | `validateMaintenanceMode` 在 DSN/根读取、dry-run 和 DB 连接之前执行；删除 previousRoots 和整个 rotate 重包分支。重编真实二进制，独立验证参数拒绝和零数据库访问 |
| key_policy.go / 正式测试 | 从无锁 inventory 数量判断改为禁用 rotate；inventory 本身不变。独立监听器、真实 SQL 摘要及 writer/cold-recovery 探针覆盖，未只相信新增断言 |
| key-audit / prune | 仍是明确不同模式；读路径和清理安全实现未改。新执行只读审计、活跃局 prune 拒绝及正式保留期/归档校验回归 |
| API、runtime、Journal、schema、客户端、协议、部署/构建 | 无代码/config 变更，但检查到共同依赖的根/恢复链；重新执行启动、HMAC/AES-GCM、真实动作与冷恢复。完整公平局、原生 UI、规模性能等引用经核对仍适用的历史证据 |

## 环境与证据

`E=/root/.config/awesome-zero-platform/acceptance-0046-r13-1474809`；目录0700、证据0600，二进制0700。`runs.jsonl` 记录实际 argv、cwd、SHA、时间/耗时/退出码，`sha256.json` 为索引。测试根、凭据、私牌和临时程序不提交。

复用 Linux、Go1.25.8、Node22.23.4、已有 MySQL5.7.44/Redis8.8.0 自有容器。新建隔离 `codex0046r13_keys`，仅在该库构造/清理 SQL 夹具；专用只读账号仅授 SELECT。API 启动使用 Round12 自有产品测试库及其原 K1，未触碰生产库/根。新编译当前 API/maintenance，`GOMAXPROCS=2 -p 1 -parallel 1` 顺序执行，API探针 `GOMEMLIMIT=192MiB`。

## R12-N01：PASS，封堵入口而非修好旧算法

原问题是无锁 COUNT 与独立 seed/checkpoint 事务之间的 TOCTOU：COUNT=0 后首个 Bid 可提交，继而重包 seed，而 Journal 保留旧 HMAC；seed COMMIT 后中断亦可留下混合根。Round12 的成功退出/真实 SIGKILL及两个单根冷恢复失败证据保留在 [Round12报告 §4](0046-round12-independent-acceptance.md)。

当前控制流为 `flag.Parse → validateMaintenanceMode → 其他参数校验 → dry-run/DSN → sql.Open/Ping`。`rotate` 明确非零拒绝，旧根加载器及重包 case 已删除，命令仅支持 key-audit/prune。旧参数 epoch 只为解析兼容保留，不参与任何重包。

全仓生产调用搜索确认 `ReencryptSeed` / `ReencryptCheckpoint` **只有底层定义，没有生产调用入口**；测试调用不算可用运维流程。当前 maintenance 二进制 `go tool nm` 亦无这两个方法及 `main.previousRoots` 符号（`symbol-check.json`）。API 根初始化未改，没有新增隐式根生成、自动轮换、HTTP 别名或维护分支。底层原语仍存在，不能把它们视为受支持完整迁移；历史证据二进制不是本候选维护入口，本轮没有再调用旧危险二进制。

独立负向验证 `negative-cli.py`：本地 TCP 接收器模拟 DB 连接观察点，18 类真实 CLI 参数覆盖普通 rotate、execute、非执行/dry-run、有无 HandID、legacy epoch/retention、单横线/等号格式、非法组合、位置式调用及 ROTATE/rotate-seed/rotate-checkpoint/rewrap 别名。全部非零退出，**数据库连接0次**，无假成功/敏感 canary 输出。受支持 key-audit 使用同一监听器正对照，实际观察到1次连接后因假服务失败，证明零连接并非观测失效。错误根 hex/JSON 同时注入，普通 rotate 优先报 disabled；未知 flag/非法数值可由 flag parser 更早拒绝，仍不接触 DB。

根读取结论结合控制流、已删除的读取函数及毒化配置验证，不把“不可达 DB 时失败”当成未访问 DB 的证明，也不把不可观测的进程环境复制冒充根解析观测。

真实 MySQL 独立探针 `TestCodexR13DisabledRotateKeepsProtectedGame`：

- 零 Journal、有效 K1/有效 writer 时持有实际 seed 行锁；真实 CLI 的 execute=false/true 分别约55/27ms拒绝，不等待 seed 行。九张相关表的所有行按稳定序列编码求摘要，前后完全一致。
- 原 writer 随后通过真实 Go runtime（不是新一轮HTTP/WSS完整对局）正常提交 **32 次 Bid/Play/Pass**，watermark33，产生实际加密 checkpoint33。已有 Journal 时再执行 rotate 亦拒绝、数据库完全不变。
- SELECT-only key-audit 正确返回 **32条v2 / 0条v1 / seed1 / checkpoint1**，前后数据库摘要不变：`66e8c3e22629aeb81079fb34b5cb99e6af18ffd740e41ea9480850a3fbc54c28`。
- 32条 Journal 均经真实 HMAC 认证，K1解密原 seed/实际 checkpoint；K1-only 新 runtime 恢复三个玩家各自私有视图，逐字节一致，原始 Bid 同 requestId 回执精确重放。错误根恢复失败且不发布部分实例。
- 活跃局 prune 拒绝且摘要不变；正式真实 MySQL 防 v2→v1 降级及归档/30天保留期清理测试通过。

原反例 A 的 writer 不再能与维护重包发生竞争，因为新 CLI 在任何 DB 操作前已结束；原反例 B 的 seed COMMIT/checkpoint 阶段在此二进制不可达。本轮没有绕过禁用或强行调用底层函数再制造旧 SIGKILL 窗口，也不宣称旧两事务算法变得正确。零连接、持锁拒绝、无重包符号、全数据不变及 K1 冷恢复共同证明封堵有效。

## 新执行与合法复用

| 实际新执行 | 命令/结果 |
|---|---|
| 当前二进制 | `go build -p 1 -trimpath ... ./apps/doudizhu-maintenance` 及 `CGO_ENABLED=0 go build ... ./apps/app-api`，均exit0 |
| 正式维护 unit | `go test -json -count=1 -p 1 -parallel 1 ./apps/doudizhu-maintenance`：3 PASS，exit0 |
| 独立 CLI 负向 | `python3 E/negative-cli.py`：18类非零拒绝、0连接、1连接正对照，探针exit0 |
| SQL/恢复/清理 | `APP_API_INTEGRATION=1 APP_MYSQL_DATABASE=codex0046r13_keys go test -json -count=1 -p 1 -parallel 1 -tags=integration -overlay E/overlay.json -run '^Test(CodexR13DisabledRotateKeepsProtectedGame\|AuthenticatedLiveJournalRejectsLegacyDowngradeInMySQL\|LiveArchivePruningRequiresVerifiedFinalAndThirtyDayRetention)$' ./business/doudizhu/infrastructure/mysqlstore`：3 PASS，exit0；实际argv见runs.jsonl |
| 当前 API 启动 | `python3 E/api-startup.py`：同K1启动/正常重启ready约166/240ms；错误根、缺根均exit1且未ready；探针exit0 |

初次隔离建库在容器启动但 MySQL socket 尚未就绪时失败，导致首轮 SQL 三项立即失败；确认 mysqladmin ping 成功并实际建库后，仅重跑受影响组一次通过。原始失败日志保留，不计为产品失败，不以反复重试碰运气。

复用 [Round10](0046-round10-independent-acceptance.md)、[Round11](0046-round11-independent-acceptance.md)、[Round12](0046-round12-independent-acceptance.md) 的完整公平三人局、真实认证/私牌隔离、终局/历史精确回执、Expiry修复、成员替换、闭房/迟到会话隔离、有界缓存、100/1000性能、原生Linux4 WECHAT导出及SDK/客户端证据。相关运行时代码、协议、客户端、部署和根派生实现无变更；本轮又实际检查其共有恢复依赖，不能将上述历史内容描述成本次完整重跑。未下载工具链、重跑完整Cocos/规模/旧故障矩阵，也没有新增真机/正式域名/抖音资格证据。

## C09/C10 与完成资格

| 要求 | 本轮独立判断 |
|---|---|
| C09-危险旧部分轮换入口 | PASS：不可达，R12-N01关闭 |
| C09-受支持的旧根→新根完整迁移 | 不存在，禁用命令不提供替代迁移流程 |
| C09-seed/checkpoint/Journal HMAC完整处理 | 未实现完整运维迁移/重认证，不可只换密文 |
| C09-new-root-only冷恢复及历史证据 | 未具备；本轮K1-only原根恢复不是迁移后new-only资格 |
| C09-原要求整体 | **BLOCKED：内部功能未完成**，不能PASS |
| C10-完整依赖库存/迁移/备份回滚/管理员核准退役 | **BLOCKED：内部功能及合格证据缺失**；key-audit计数仍UNVERIFIED，不能证明依赖归属或安全退役 |

矩阵差异：**C09从FAIL改为BLOCKED**，因为已证明Round12危险维护路径被封堵，当前缺少受支持完整迁移能力；不是把该要求解除或改为通过。其他109项PASS与C10/E03/H05/GOAL8四项BLOCKED沿用经核查适用的Round12矩阵。因此 **109 PASS + 0 FAIL + 5 BLOCKED = 114**。历史Round12 FAIL记录及其原矩阵不改动。矩阵PASS的正常稳定根恢复范围不等于迁移、退役或发布资格。

## 遗留分类与最小交接

- **新代码缺陷：** 本轮未建立新的可达P1/P2；历史R12-N01/P2修复PASS。
- **原Goal功能缺口，高优先级：** C09/C10。ChatGPT需另行实现有明确原子/静默边界、HMAC重认证、全量密钥依赖库存和可恢复中断处理的管理员迁移；补new-only冷恢复、历史回执/终局验证及备份回滚/核准退役证据。现有低层原语不能直接宣称完成。
- **验收证据/用户操作缺口：** E03真实备份确认与恢复UX，脚本保存/导入不能代替。
- **外部平台/发布条件：** H05正式抖音导出/SDK；GOAL8官方工具、批准HTTPS/WSS域名、真实手机及正式发布条件，继续BLOCKED。
- **可提交的设计建议：** 若用户决定暂不要求完整换根，可另行正式修订/拆分Goal；本次没有替用户改范围或删除C09/C10。

Goal0045基础客户端原完成条件尚未全部解除，**保持in_progress**；Goal0046受C09/C10等阻塞，**保持in_progress**；外部发布资格**BLOCKED**。A代码修复PASS不能推导B/C通过，不建议封版。

仅提交本报告与current允许的状态段落；报告提交SHA和产品SHA分开，最终总结给出报告提交，也可 `git log -1 --format=%H -- docs/goals/0046-round13-independent-acceptance.md` 查询。提交前核对证据来源、实际源码、状态差异及敏感信息；停止本轮API子进程和自有数据库容器，保留私有证据和测试库卷，未更改其他部署。
