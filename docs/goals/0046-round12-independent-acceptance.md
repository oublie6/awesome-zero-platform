# Goal 0046 — Round 12 综合独立验收

日期：2026-11-06 UTC。执行者：Codex。产品固定 **`6c158868cfda18c2f2e82c910d89545f7ff9f815`**。

| 判定 | 结论 |
|---|---|
| A：本轮稳定密钥/维护代码变更 | **FAIL**：稳定根、HMAC/AES-GCM、只读审计通过；不完整轮换的并发/中断保护失败，新增 **R12-N01/P2** |
| B：Goal 0046 内部产品完成资格 | **FAIL，不能封版**；C09 有实际失败反例，C10 仍缺完整库存、迁移及退役资格证据 |
| C：外部发布资格 | **BLOCKED**；用户备份体验、抖音及官方工具/域名/真机/发布条件未全部完成 |

**114 项：109 PASS / 1 FAIL / 4 BLOCKED。** C09 从 BLOCKED 改为 FAIL，其他状态不变；不是新增一个独立矩阵 ID。Goal 0045/0046 继续 `in_progress`。验收报告的完成不代表产品通过。

## 1. 固定版本、范围和环境

开始时 clean main，在 Round 11 报告提交 `b55b86c6ca8014f9ef188182f6b814654711edba` 执行 `git pull --ff-only origin main`，恰好同步到上述候选。核查 AGENTS、相关原则、current、集成交付 Goal、Round 10/11 报告、密钥与生产运维文档及直接源码/测试。实际 diff 仅 7 文件：维护 CLI 的审计/拒绝保护及测试、运维和 Goal 文档；**游戏运行时、客户端、schema、协议、构建/部署及 CI 未改动**。没有修改产品或正式测试。

精确 SHA 的 [CI 42166634436](https://github.com/oublie6/awesome-zero-platform/actions/runs/42166634436) **6/6 SUCCESS**、[Fair Doudizhu 42166634442](https://github.com/oublie6/awesome-zero-platform/actions/runs/42166634442) **4/4 SUCCESS** 已独立读取 job/step 元数据，见 `ci.json`。未把 CI 绿色代替下面的真实反例及集成测试。

私有证据根 **`E=/root/.config/awesome-zero-platform/acceptance-0046-r12-6c15886`**。`runs.jsonl` 记录命令、cwd、源码 SHA、UTC、耗时和退出码；`sha256.json` 为文件摘要索引。目录 0700、敏感证据 0600，二进制仅本用户执行。真实密码、JWT、根密钥、私牌和恢复套件不提交。

复用 Linux x86_64、Go 1.25.8、Node 22.23.4、已安装原生工具；自有 MySQL 5.7.44/Redis 8.8.0 容器。本轮新建 **`codex0046r12_product` / `codex0046r12_keys`** 隔离库，使用当前 schema 和新测试根；不读写生产库或生产密钥。专用 `codex_r12_ro` 仅授予这两库 SELECT，真实 UPDATE 被拒绝。各独立 SQL 夹具的根与完整局的随机根隔离，K1/K2 仅为逻辑标签。专属故障夹具按前缀创建/清理；生产式完整局使用独立新库，审计前保存私有整库备份，回滚/重建只涉及这两库。

Go 使用 `GOMAXPROCS=2 -p 1 -parallel 1`，重型工作顺序执行。当前候选实际编译 API 和 maintenance；API 隔离服务 512 MiB/75% CPU，TLS 代理走回环 HTTPS/WSS。未重新下载 Cocos 或重跑全量 CI、100/1000 局及旧 SIGKILL 全矩阵；逐项复核后复用适用的 [Round 10](0046-round10-independent-acceptance.md) 和 [Round 11](0046-round11-independent-acceptance.md) 原始证据。

## 2. 稳定密钥、加密与恢复：通过范围

- 管理员显式配置 K1；真实四账号登录、三人完成一局。启动/重启/第二局均未产生新根或自动切换 KeyID。种子熵和 AEAD nonce 的随机生成不是根轮换；实际运行时没有自动根生成/轮换任务。
- 当前完整局产生 **59 条 v2 Journal、1 份加密种子、1 份 Checkpoint**。独立 Python 按 Go 固定绑定字段和 canonical payload 重新计算全部 **59 条 HMAC-SHA256**，全部一致；另以不同派生用途的密钥成功认证种子 AES-GCM（48 字节密文）和 Checkpoint AES-GCM（v33，77018 字节密文），校验明文摘要。根、seed key、checkpoint key、journal key 四者互异。未输出解密内容，见 `crypto-inventory.py/.json`。
- 真实 MySQL 定向篡改 payload、version、epoch、actor、position、sequence、digest、tag 及 v2→v1/zero-sequence 共 **9 类**均拒绝；每次恢复原行后摘要一致、认证读取成功。正式真实 MySQL 降级攻击以及 Checkpoint/私有视图恢复测试亦通过，未允许用裸 SHA 降级绕过 HMAC。
- 当前 API 子进程实测缺根、短根、非 hex、uppercase 非规范根、缺/空白/超长 KeyID 及错误根：**8 类均 exit 1**。错误根在恢复阶段拒绝，未监听一个可提供部分私牌的成功服务。相同 K1 两次正常 SIGTERM 重启后，实际已确认 v4 的三人业务投影精确保持，ready+恢复约 **193/141 ms**。实际进程接管、旧 epoch 拒绝及加密旧根读取另有本轮 SQL 正对照。
- 错误根/混合根恢复失败后，独立 runtime `Contains=false`，不发布部分手实例。运行日志未出现测试根和私有 seed canary。普通 Journal 保留已公开动作字段，不把所有字段伪装成加密；保密性由私有投影、受保护 seed/checkpoint 和访问控制共同承担。
- 临时 root/key/plaintext byte slices 有清理；环境字符串、进程所需派生密钥/AEAD 调度仍驻留必要内存，不声称锁页、硬件密钥或彻底可证明清零。运维文档正确要求根与数据库分别保护并配套备份；丢失根不能用完整数据库替代。没有在客户端加入服务端根密钥。

## 3. key-audit 与普通轮换保护

实际执行以下用户指定入口（SELECT-only DSN 由私有环境注入，不进入 argv）：

```sh
cd server
go run ./apps/doudizhu-maintenance --mode key-audit
go run ./apps/doudizhu-maintenance --mode key-audit --hand <test-hand-id>
```

两条均 exit 0，完整局全局/指定统计均为 **journal_actions=59 / authenticated_v2=59 / legacy_v1=0 / seeds=1 / checkpoints=1**，且始终 `old_root_retirement=UNVERIFIED (C09/C10)`。直接二进制审计约 **30/40 ms**；整库 `mysqldump --single-transaction --skip-comments --compact` 前后完全一致，SHA256 `c13777692a7139fbc9b5dde38529887c214cb54b84b645055895543fdd062647`。见 `audit-go-run.log`、`crypto-inventory.json` 和私有备份。

隔离 SQL 夹具还验证空 hand、seed-only、无 checkpoint、带 checkpoint、混合 v1/v2 元数据计数；混合版本夹具只用于分类统计，不冒充可认证有效历史。`--execute`、SQL 注入样式 HandID、129 字符 HandID 均明确拒绝；只读账号无写权限；错误密码/数据库不可达均非零退出，不输出成功清单。审计不需要根密钥，既不检查全部记录 HMAC，也不能推断记录已换到哪一个根。

源码使用参数绑定，表名为编译期常量；COUNT 结果有界、没有逐行装入内存，数据库操作有 45 秒 context 上限。全局 COUNT 在大库仍有扫描成本，三次查询不是同一时间点的事务快照；不能把本轮几十毫秒小库样本扩大为任意规模 SLA。审计为非锁定读，没有新增游戏热路径调用。

已有 Journal 时，v2、v1、未知 v0 及混合元数据的普通 `rotate --execute` 均明确拒绝；同一操作重复两次，相关完整数据库行摘要不变，旧根冷恢复仍成功。无 Journal 的合法显式重包成功，新根可解密且 seed 值保持不变、旧根单独读取拒绝，输出仅称 `PARTIAL_REWRAP`，未宣称旧根可退役。**这些正向保护不覆盖下一节的并发窗口。**

## 4. 新缺陷 R12-N01 / P2：检查后首个 Journal 写入可绕过拒绝保护

**位置：** `server/apps/doudizhu-maintenance/main.go:123–151` 的独立 inventory/seed/checkpoint 步骤，`key_policy.go:21–60` 的无锁统计及计数判断，`infrastructure/mysqlstore/key_rotation.go` 的两个独立重包事务。

**触发条件：** 一局已受 K1 保护、writer 有效但还没有实时 action；操作员以 K2+保留 K1 运行真实 CLI，期间仍有一个合法 writer。文档要求先静默所有 writer，但命令不验证或强制这个前提。本轮任务明确要求并发/异常退出不能绕过保护，不能用手工静默约定代替该检查。未证明正常不运行维护命令的游戏路径会自行换根。

最短确定性复现（只在隔离库）：

1. 用仓库域/runtime 建立真实 BIDDING、protected seed、writer，Journal actions=0。
2. 单独 SQL 事务锁住该 seed 行，启动实际 `maintenance --mode rotate --hand H --epoch E --execute`（根由私有环境提供）。通过实际 PROCESSLIST 看到 CLI 已通过统计，阻塞在 seed `SELECT ... FOR UPDATE`。
3. 原 runtime 提交首个合法、已确认 **Bid v2**，产生 K1 HMAC Journal，然后释放 seed 锁。
4. 实际 CLI **exit 0**，输出 `PARTIAL_REWRAP completed for a no-journal hand`；但该 hand 此时已有已确认 Journal。
5. K1-only 不能解密被换根的 seed；K2-only 可读 seed，但不能认证原 Journal。两个独立真实 runtime 冷恢复分别因 protected seed authentication / live log integrity 失败，均 `Contains=false`。

另一个确定性变体在 Bid 提交后锁住 checkpoint 的 writer 行；CLI seed 已 COMMIT、checkpoint 尚被锁时 **SIGKILL 实际 CLI**，同样留下混合根，两个单根冷恢复均失败。不是端口 mock 或把回执丢失模拟成 SQL 失败。

**预期：** 发现 Journal 后拒绝不完整重包，相关 seed/checkpoint 不变，K1 仍能恢复；并发或退出不能穿过这个保护边界。

**实际与影响：** inventory 到修改之间没有共同锁/事务边界，计数检查失效；即使不删除任何旧根，仍需人工正确配置混合 keyring 才可能恢复。未发现已确认动作被删除或未经授权者拿到私牌，不把该受限操作员场景夸大为普通远程攻击/P1。它违反本轮新增安全保护要求，按 P2 阻止本轮代码验收 PASS。

**建议交回 ChatGPT：** 在模块所有的维护适配器中建立与 Append/writer 一致的锁顺序及原子检查/修改边界，强制验证静默/owner 条件、重新检查 Journal，并保证拒绝/中断时没有误导的部分根迁移；保留真实并发、重复和中断反例。不要通过取消 HMAC、接受 v1 降级、删除旧根或放宽断言修复。

证据：`key_acceptance_test.go`、`key-race-cold-and-tamper.log`（同一反例补全两个真实冷恢复）及 `maintenance-interrupted-race.log`。前者 **11 PASS / 1 FAIL**（含子测试），后者 **1 FAIL**；是同一缺陷的两个调度，不是两个新缺陷。最初临时探针用了不匹配的旧根而提前认证失败，纠正为仓库夹具 K1 后反例成立；未采用那个提前失败作为保护通过依据。

## 5. C09/C10 独立结论

**C09：FAIL（Round 11 为 BLOCKED）。** 受支持的完整 Journal HMAC 重认证迁移仍未实现；本轮进一步建立“CLI 成功重包而有 Journal 保留旧 HMAC、新根-only 实际恢复失败”的反例，因此从仅缺证据升级为实际 FAIL。没有声称新迁移能力已实现，也没有修改原“新根-only 恢复”标准。稳定密钥能降低日常换根需求，不能关闭该原始条件。

**C10：BLOCKED。** key-audit 只按完整性版本/表统计，没有全量根依赖归属库存、完整迁移、核准退役门槛、可操作且已验证的整体回滚/备份恢复流程。所有输出仍明确 UNVERIFIED；没有在生产环境删除任何根。解除条件为受支持迁移工具、全库存 new-only 冷恢复/精确历史回执/终局证据及故障回滚验收，随后才可考虑管理员核准退役。

## 6. 真实业务、客户端与历史回归

本轮完整局不使用确定性公平替代夹具：四个新正常 Go 认证账号，三人创建/加入/准备 → 外部保存恢复套件 → 公平承诺 → 新 Node 进程导入套件 → 正常 Manifest/AAD/HPKE 加密揭示 → **drand Quicknet 独立验证** → 51 张互异私牌/三个不同 Seat → 真实叫地主/出牌/过牌 → 结算/归档/证据。第四身份私牌 403；恢复凭证由脚本外部保存并导入不替代人类备份 UX，E03 不升 PASS。

已确认实时动作共 **59 条、终态 v60、归档 1 份**；完整局继续部分实际执行 56 条 WSS 动作，之前故障段已执行其余动作。seatPoints **[12,-6,-6]**，和为 0；终局 digest 与冷恢复证据保持一致。`full-game-final.log`、`game-commit.log`、`game-reveal.log`、`crypto-inventory.json`。

- 当前 K1 下真实三人 WSS 观察两次 API SIGKILL：SQL COMMIT 前中断回滚、原请求精确重试只写一条；COMMIT 后主动不接收回复，重启后同 requestId 原参数精确重放；改参数和不同 actor 拒绝。冷恢复+三个 WSS 约 **161/163 ms**。
- COMPLETED 冷重启约 **385 ms**，原始 Bid/Play/Pass 及最终请求回执精确重放；actions59、watermark60、commands11、outbox15、archives1 均无重复。writer epoch 9→10 为正常接管，非重复业务效果。
- 真实 A/B/C 完成 H1 后 C 离、D 进：D 房间不含无权旧 locator，实际重连成功、0 次旧 archive 请求，直接旧 archive 403；A/B/C 权限保留。同房 A/B/D 再准备/start H2，再实际 pre-deal cancel，原 HTTP 响应同 requestId deepEqual；按既定 lifecycle contract 为 ABORTED，无虚假 archive locator，旧 H1 archive 仍可按历史参与者读取。
- 原成员最后退出后 CLOSED 持久化查询 200；客户端正确拒绝非成员恢复、local room state 为空，历史权限仍保留。相关脚本最初额外假定“取消保留旧 locator / 状态名 CANCELLED / 非成员 closed reconnect 返回 null”，与现有契约不同；保留原失败日志，核对实际 lifecycle/controller 后改为断言归档权限保留、ABORTED 和明确 membership 拒绝。未改产品/正式测试或降低既定标准。成员替换成功断言、cancel 精确重试和 closed 最终验证分别保留，不能把最初整段脚本说成 exit 0。
- 本轮 SQL 定向复验 R9-N01：拒绝 Expiry 的 epoch2→2、水位1→1、BIDDING/v12 不变，拒绝回执持久化、合法 Bid 继续；合法 Sweep epoch2→3 恰好一次、同 ID 重试不 refence，旧 writer 阻塞后正确拒绝。EXPIRED 冷恢复/归档补偿及加密旧 keyring 正对照亦通过。

迟到 HTTP/WSS 会话隔离、有界历史热缓存、完整原生第二局等复杂场景复用 Round 10/11 原始独立证据；已检查实现无改动，当前稳定根下正常冷恢复/终局重放重新通过，没有把此次维护变化接入这些热路径。客户端生产代码、HTTP/WSS DTO 和 Linux 原生官方 CLI4 策略未变，复用 Round 10 正式 Linux WECHAT 导出、真实 SDK/两局 UI 证据；不下载引擎、Wine 或其他 OS，不声称本轮重新原生构建/真机测试。

## 7. 性能、资源与自动化

Go 集成验证的公共参数为 `APP_API_INTEGRATION=1 APP_MYSQL_DATABASE=codex0046r12_keys GOMAXPROCS=2 go test -json -count=1 -p 1 -parallel 1 -tags=integration`；独立用例加 `-overlay $E/overlay.json`，包 `./business/doudizhu/infrastructure/mysqlstore`，正式恢复回归另含 `./business/doudizhu/infrastructure/runtime`。单独复现本缺陷可加 `-run '^TestCodexR12RotationConcurrentFirstAction$'`；该探针预期在候选上 exit1。各组实际完整 argv、时间/退出码保留于 `runs.jsonl`，不把不同批次或重复子测试数相加为独立总数。

| 新执行 | 结果/边界 |
|---|---|
| maintenance/config 定向 Go unit | exit0，27 PASS（含子测试） |
| 直接安全/恢复关联 Go 两包 | exit0，17 PASS：HMAC 绑定/降级、真实 SQL/加密/epoch、EXPIRED、Checkpoint/完整私有视图恢复、Expiry |
| 独立审计/普通拒绝/无 Journal 正维护 | 正向通过；组合日志 `independent-key-final.log` 为 2 PASS/1 FAIL，唯一 FAIL 是并发轮换反例 |
| 原始 HMAC/AES-GCM 独立 Python 验证及整库只读审计 | exit0；59 条认证、2 种密文、全库摘要不变 |
| 80 次 keyed Journal COMMIT-before-ACK | P50/P95/P99 **8.49/49.16/66.74 ms**，max365.52 ms；因 P99 超建议50ms，追加一次同配置定向复测 **9.34/37.31/45.45 ms**，max51.67 ms；两次数据均保留 |
| 56 次实际游戏 WSS 命令 | 客户端观测 P50/P95/P99 **34.08/64.56/123.61 ms**，包含网络、应用及提交，不能冒充纯 SQL 或 server CPU |
| MySQL 锁 | 本轮真实锁竞争/旧 writer 拒绝可观察；实例统计 waits5、总321ms、avg64ms、max215ms，包含各隔离探针，非生产逐局统计 |
| 内存/goroutine | 同一进程采样 goroutine25→25，heap72.83→4.37MB、RSS176.46→99.03MB；未强制 GC，非长期零泄漏证明；见 metrics-after/final |

Keyed commit 复测用于解释实测离散，不删首次慢值；热路径源码与旧基线一致，新维护查询没有自动挂到游戏操作链。当前小样本不能确定生产尾延迟或代码性能回归，未声称所有样本达成建议目标。100/1000 手、Race/Vet、连接/长历史压力等复用 Round 10 的真实原始证据及当前 CI；未无意义再压测。drand relay 曾报一次500，正常验证重试后完成发牌，未跳过 BLS 校验。

初始探针编译的 unused import、错误测试根、正常关闭等待不完整及客户端契约假设错误均保留在原始日志，和 **R12-N01 的有效产品失败**分开；最终必要正向场景及反例有明确独立结果。没有把任一整体 exit1 日志改写为 PASS。

## 8. 114 项完整复核矩阵

下表每项保留原始 ID/要求，状态以上轮 Round 11 为基线；“新”表示该要求的有关路径本轮重跑，“部分”表示结合新证据及历史不变路径，“否”表示经代码兼容核查后复用历史原始结果。证据代码：**N1** unit、**N2** key SQL/反例、**N3** 安全恢复 SQL、**N4** 真实完整局/故障/终局重放、**N5** 成员替换/第二局开始取消/closed、**N6** 独立密码验证/审计、**N7** 性能/metrics；具体命令及退出码见上述节及 `runs.jsonl`。**R10/R11** 直接引用对应历史报告原始证据，而非声称在本轮重跑。AUDIT 项表示审计交付完成，不表示没有缺陷。

| ID / 原要求 | Round11 → Round12 | 证据来源 | 本轮执行 | 当前源码兼容核查 | 变化原因 / 未完成条件 |
|---|---|---|---|---|---|
| A01 真实身份认证和账号隔离 | PASS → PASS | N4/N5 + R10/R11 | 部分 | 运行时/协议不变；本轮稳定根关联路径验证 | — |
| A02 创建、加入、退出房间 | PASS → PASS | N4/N5 + R10/R11 | 部分 | 运行时/协议不变；本轮稳定根关联路径验证 | — |
| A03 三人准备和开局 | PASS → PASS | N4/N5 + R10/R11 | 部分 | 运行时/协议不变；本轮稳定根关联路径验证 | — |
| A04 公平承诺与揭示 | PASS → PASS | N4/N5 + R10/R11 | 部分 | 运行时/协议不变；本轮稳定根关联路径验证 | — |
| A05 公共随机源验证及发牌 | PASS → PASS | N4/N5 + R10/R11 | 部分 | 运行时/协议不变；本轮稳定根关联路径验证 | — |
| A06 地主竞叫及确定 | PASS → PASS | N4/N5 + R10/R11 | 部分 | 运行时/协议不变；本轮稳定根关联路径验证 | — |
| A07 出牌、过牌、牌型比较 | PASS → PASS | N4/N5 + R10/R11 | 部分 | 运行时/协议不变；本轮稳定根关联路径验证 | — |
| A08 回合切换 | PASS → PASS | N4/N5 + R10/R11 | 部分 | 运行时/协议不变；本轮稳定根关联路径验证 | — |
| A09 终局、计分及证据归档 | PASS → PASS | N4/N5 + R10/R11 | 部分 | 运行时/协议不变；本轮稳定根关联路径验证 | — |
| A10 断线重连及权威状态同步 | PASS → PASS | N4/N5 + R10/R11 | 部分 | 运行时/协议不变；本轮稳定根关联路径验证 | — |
| B01 已向客户端确认的操作必须先完成持久化提交。 | PASS → PASS | N3/N4/N6 + R10 | 部分 | 正常稳定根恢复通过；换根资格另见 C09/C10 | 稳定根路径通过；不包含 C09/C10 换根资格 |
| B02 任何已确认操作不能在进程崩溃后丢失。 | PASS → PASS | N3/N4/N6 + R10 | 部分 | 正常稳定根恢复通过；换根资格另见 C09/C10 | 稳定根路径通过；不包含 C09/C10 换根资格 |
| B03 同一 requestId 的重复请求不能重复执行。 | PASS → PASS | N3/N4/N6 + R10 | 部分 | 正常稳定根恢复通过；换根资格另见 C09/C10 | 稳定根路径通过；不包含 C09/C10 换根资格 |
| B04 相同 requestId、不同参数或不同操作者必须拒绝。 | PASS → PASS | N3/N4/N6 + R10 | 部分 | 正常稳定根恢复通过；换根资格另见 C09/C10 | 稳定根路径通过；不包含 C09/C10 换根资格 |
| B05 新服务接管后，旧 Owner Epoch 必须被正确隔离。 | PASS → PASS | N3/N4/N6 + R10 | 部分 | 正常稳定根恢复通过；换根资格另见 C09/C10 | 稳定根路径通过；不包含 C09/C10 换根资格 |
| B06 检查点、日志、水位线和业务版本必须严格一致。 | PASS → PASS | N3/N4/N6 + R10 | 部分 | 正常稳定根恢复通过；换根资格另见 C09/C10 | 稳定根路径通过；不包含 C09/C10 换根资格 |
| B07 检查点或日志被篡改时必须拒绝恢复。 | PASS → PASS | N3/N4/N6 + R10 | 部分 | 正常稳定根恢复通过；换根资格另见 C09/C10 | 稳定根路径通过；不包含 C09/C10 换根资格 |
| B08 恢复不能泄露其他玩家的私有手牌。 | PASS → PASS | N3/N4/N6 + R10 | 部分 | 正常稳定根恢复通过；换根资格另见 C09/C10 | 稳定根路径通过；不包含 C09/C10 换根资格 |
| B09 终局重启必须保留最终证据和历史回执。 | PASS → PASS | N3/N4/N6 + R10 | 部分 | 正常稳定根恢复通过；换根资格另见 C09/C10 | 稳定根路径通过；不包含 C09/C10 换根资格 |
| B10 MySQL COMMIT 前后发生 SIGKILL、网络断开、响应丢失时，不得出现重复执行或状态分叉。 | PASS → PASS | N3/N4/N6 + R10 | 部分 | 正常稳定根恢复通过；换根资格另见 C09/C10 | 稳定根路径通过；不包含 C09/C10 换根资格 |
| B11 真实 MySQL 事务 | PASS → PASS | N3/N4/N6 + R10 | 部分 | 正常稳定根恢复通过；换根资格另见 C09/C10 | 稳定根路径通过；不包含 C09/C10 换根资格 |
| B12 并发重复提交 | PASS → PASS | R10/R11 + 精确SHA CI | 否 | 相关实现无diff，历史原始证据仍适用 | 稳定根路径通过；不包含 C09/C10 换根资格 |
| B13 Epoch fencing | PASS → PASS | N3/N4/N6 + R10 | 部分 | 正常稳定根恢复通过；换根资格另见 C09/C10 | 稳定根路径通过；不包含 C09/C10 换根资格 |
| B14 检查点加密 | PASS → PASS | N3/N4/N6 + R10 | 部分 | 正常稳定根恢复通过；换根资格另见 C09/C10 | 稳定根路径通过；不包含 C09/C10 换根资格 |
| B15 日志完整性 | PASS → PASS | N3/N4/N6 + R10 | 部分 | 正常稳定根恢复通过；换根资格另见 C09/C10 | 稳定根路径通过；不包含 C09/C10 换根资格 |
| B16 三人完整对局恢复 | PASS → PASS | N3/N4/N6 + R10 | 部分 | 正常稳定根恢复通过；换根资格另见 C09/C10 | 稳定根路径通过；不包含 C09/C10 换根资格 |
| B17 进程级故障注入 | PASS → PASS | N3/N4/N6 + R10 | 部分 | 正常稳定根恢复通过；换根资格另见 C09/C10 | 稳定根路径通过；不包含 C09/C10 换根资格 |
| B18 终局证据恢复 | PASS → PASS | N3/N4/N6 + R10 | 部分 | 正常稳定根恢复通过；换根资格另见 C09/C10 | 稳定根路径通过；不包含 C09/C10 换根资格 |
| C01 新记录使用当前主密钥加密 | PASS → PASS | N1/N2/N3/N6 + R10 | 部分 | 根/密文实现未变；新配置、审计及密钥正负对照 | — |
| C02 旧记录可以使用保留的历史密钥读取 | PASS → PASS | N1/N2/N3/N6 + R10 | 部分 | 根/密文实现未变；新配置、审计及密钥正负对照 | — |
| C03 历史密钥数量和长度严格受限 | PASS → PASS | N1/N2/N3/N6 + R10 | 部分 | 根/密文实现未变；新配置、审计及密钥正负对照 | — |
| C04 旧密钥不存在时拒绝错误解密 | PASS → PASS | N1/N2/N3/N6 + R10 | 部分 | 根/密文实现未变；新配置、审计及密钥正负对照 | — |
| C05 重加密必须先认证旧密文 | PASS → PASS | N1/N2/N3/N6 + R10 | 部分 | 根/密文实现未变；新配置、审计及密钥正负对照 | — |
| C06 种子重加密不得改变原始随机贡献 | PASS → PASS | N1/N2/N3/N6 + R10 | 部分 | 根/密文实现未变；新配置、审计及密钥正负对照 | — |
| C07 检查点重加密必须校验当前 Owner Epoch | PASS → PASS | N1/N2/N3/N6 + R10 | 部分 | 根/密文实现未变；新配置、审计及密钥正负对照 | — |
| C08 错误密钥、篡改密文不能恢复 | PASS → PASS | N1/N2/N3/N6 + R10 | 部分 | 根/密文实现未变；新配置、审计及密钥正负对照 | — |
| C09 重加密后仅保留新密钥仍可正确恢复 | BLOCKED → FAIL | N2；§4/§5；两个实际单根冷恢复失败 | 新 | 历史仅缺证据，当前建立实际反例 | R12-N01；成功/中断重包留下旧HMAC，new-only冷恢复失败；完整迁移仍缺失 |
| C10 旧密钥未完成迁移前不得被错误淘汰 | BLOCKED → BLOCKED | R10/R11 + 本轮能力/环境复核 | 复核 | 缺失条件未因维护CLI或政策文字改变 | 完整依赖库存、Journal重认证迁移、核准退役及整体回滚尚未实现/验证 |
| D01 活跃牌局不能删除 | PASS → PASS | R10/R11 + 精确SHA CI | 否 | 相关实现无diff，历史原始证据仍适用 | — |
| D02 未归档牌局不能删除 | PASS → PASS | R10/R11 + 精确SHA CI | 否 | 相关实现无diff，历史原始证据仍适用 | — |
| D03 不满足 30 天保留期不能删除 | PASS → PASS | R10/R11 + 精确SHA CI | 否 | 相关实现无diff，历史原始证据仍适用 | — |
| D04 损坏或不完整的归档不能删除 | PASS → PASS | R10/R11 + 精确SHA CI | 否 | 相关实现无diff，历史原始证据仍适用 | — |
| D05 最终版本和日志水位线不一致不能删除 | PASS → PASS | R10/R11 + 精确SHA CI | 否 | 相关实现无diff，历史原始证据仍适用 | — |
| D06 历史日志数量不完整不能删除 | PASS → PASS | R10/R11 + 精确SHA CI | 否 | 相关实现无diff，历史原始证据仍适用 | — |
| D07 不得直接越权读取其他业务模块数据库表 | PASS → PASS | R10/R11 + 精确SHA CI | 否 | 相关实现无diff，历史原始证据仍适用 | — |
| D08 保留完整终局证据和必要审计信息 | PASS → PASS | R10/R11 + 精确SHA CI | 否 | 相关实现无diff，历史原始证据仍适用 | — |
| D09 默认 dry-run 不得修改数据库 | PASS → PASS | dry-run.json + R10 | 部分 | CLI变更入口新核对；清理实现不变 | — |
| D10 必须显式 `--execute` 才能执行清理 | PASS → PASS | dry-run.json + R10 | 部分 | CLI变更入口新核对；清理实现不变 | — |
| D11 中途崩溃后不能出现违反一致性约束的部分删除 | PASS → PASS | R10/R11 + 精确SHA CI | 否 | 相关实现无diff，历史原始证据仍适用 | — |
| E01 提交承诺前生成高熵随机贡献 | PASS → PASS | N4 + R10 | 部分 | 真实新进程凭证/精确重试；负向边界引用历史 | — |
| E02 恢复凭证包含完整且可验证的信息 | PASS → PASS | N4 + R10 | 部分 | 真实新进程凭证/精确重试；负向边界引用历史 | — |
| E03 用户备份后才允许提交承诺 | BLOCKED → BLOCKED | R10/R11 + 本轮能力/环境复核 | 复核 | 缺失条件未因维护CLI或政策文字改变 | 真实用户备份确认与恢复UX未验收；脚本保存/导入不替代用户体验 |
| E04 应用退出后可以导入恢复凭证 | PASS → PASS | N4 + R10 | 部分 | 真实新进程凭证/精确重试；负向边界引用历史 | — |
| E05 恢复凭证与当前登录身份、Hand、Seat、Commitment 必须匹配 | PASS → PASS | N4 + R10 | 部分 | 真实新进程凭证/精确重试；负向边界引用历史 | — |
| E06 错误凭证必须拒绝 | PASS → PASS | R10/R11 + 精确SHA CI | 否 | 相关实现无diff，历史原始证据仍适用 | — |
| E07 不能产生新的随机贡献冒充旧承诺 | PASS → PASS | N4 + R10 | 部分 | 真实新进程凭证/精确重试；负向边界引用历史 | — |
| E08 不能通过普通小游戏明文持久化自动泄漏随机数 | PASS → PASS | R10/R11 + 精确SHA CI | 否 | 相关实现无diff，历史原始证据仍适用 | — |
| E09 释放对象时敏感数据应尽可能清零 | PASS → PASS | R10/R11 + 精确SHA CI | 否 | 相关实现无diff，历史原始证据仍适用 | — |
| E10 请求丢响应时必须保留原 requestId 精确重试 | PASS → PASS | N4 + R10 | 部分 | 真实新进程凭证/精确重试；负向边界引用历史 | — |
| E11 不能因为未知结果直接生成新的操作请求 | PASS → PASS | N4 + R10 | 部分 | 真实新进程凭证/精确重试；负向边界引用历史 | — |
| F01 验证使用三个真实、不同的服务端认证身份 | PASS → PASS | N4/N6 | 部分 | 运行时/协议不变；本轮稳定根关联路径验证 | — |
| F02 验证三人座位各不相同 | PASS → PASS | N4/N6 | 部分 | 运行时/协议不变；本轮稳定根关联路径验证 | — |
| F03 验证每次操作使用真实 WSS | PASS → PASS | N4/N6 | 部分 | 运行时/协议不变；本轮稳定根关联路径验证 | — |
| F04 验证客户端私有手牌访问权限 | PASS → PASS | N4/N6 | 部分 | 运行时/协议不变；本轮稳定根关联路径验证 | — |
| F05 验证游戏完整进行到终局 | PASS → PASS | N4/N6 | 部分 | 运行时/协议不变；本轮稳定根关联路径验证 | — |
| F06 验证最终结算及证据 | PASS → PASS | N4/N6 | 部分 | 运行时/协议不变；本轮稳定根关联路径验证 | — |
| G01 并发牌局状态隔离 | PASS → PASS | R10/R11 + 精确SHA CI | 否 | 相关实现无diff，历史原始证据仍适用 | — |
| G02 内存增长与回收 | PASS → PASS | N3/N7 + R10 | 部分 | 新小规模观测；旧100/1000与长时资源证据兼容 | — |
| G03 MySQL 提交延迟 | PASS → PASS | N3/N7 + R10 | 部分 | 新小规模观测；旧100/1000与长时资源证据兼容 | — |
| G04 P50/P95/P99 | PASS → PASS | N3/N7 + R10 | 部分 | 新小规模观测；旧100/1000与长时资源证据兼容 | — |
| G05 检查点及故障恢复耗时 | PASS → PASS | N3/N7 + R10 | 部分 | 新小规模观测；旧100/1000与长时资源证据兼容 | — |
| G06 高并发下连接池和锁竞争 | PASS → PASS | N3/N7 + R10 | 部分 | 新小规模观测；旧100/1000与长时资源证据兼容 | — |
| G07 Race 检查 | PASS → PASS | R10/R11 + 精确SHA CI | 否 | 相关实现无diff，历史原始证据仍适用 | — |
| G08 Goroutine 泄漏 | PASS → PASS | N3/N7 + R10 | 部分 | 新小规模观测；旧100/1000与长时资源证据兼容 | — |
| G09 重复请求及失败重试压力 | PASS → PASS | R10/R11 + 精确SHA CI | 否 | 相关实现无diff，历史原始证据仍适用 | — |
| G10 长牌局的候选状态复制开销 | PASS → PASS | R10/R11 + 精确SHA CI | 否 | 相关实现无diff，历史原始证据仍适用 | — |
| H01 TypeScript 类型检查 | PASS → PASS | R10/R11 + 精确SHA CI | 否 | 相关实现无diff，历史原始证据仍适用 | — |
| H02 Web 实际构建 | PASS → PASS | R10/R11 + 精确SHA CI | 否 | 相关实现无diff，历史原始证据仍适用 | — |
| H03 Chromium 场景冒烟测试 | PASS → PASS | R10/R11 + 精确SHA CI | 否 | 相关实现无diff，历史原始证据仍适用 | — |
| H04 微信小游戏构建 | PASS → PASS | R10/R11 + 精确SHA CI | 否 | 相关实现无diff，历史原始证据仍适用 | — |
| H05 抖音小游戏构建兼容性 | BLOCKED → BLOCKED | R10/R11 + 本轮能力/环境复核 | 复核 | 缺失条件未因维护CLI或政策文字改变 | 正式Linux抖音导出与SDK资格仍无合格证据 |
| H06 游戏 UI 操作与真实网络请求一致 | PASS → PASS | N4/N5 + R10 | 部分 | 新生产controller集成；旧原生GUI/SDK代码一致 | — |
| H07 重连及恢复凭证操作可触达 | PASS → PASS | N4/N5 + R10 | 部分 | 新生产controller集成；旧原生GUI/SDK代码一致 | — |
| H08 游戏页面布局、按钮可见性和状态刷新 | PASS → PASS | N4/N5 + R10 | 部分 | 新生产controller集成；旧原生GUI/SDK代码一致 | — |
| AUTO01 Go 单元测试 | PASS → PASS | N1/N3/N6 + CI/R10 | 部分 | 新定向验证；未改动模块CI及历史全套 | — |
| AUTO02 Go Race 测试 | PASS → PASS | R10/R11 + 精确SHA CI | 否 | 相关实现无diff，历史原始证据仍适用 | — |
| AUTO03 Go Vet | PASS → PASS | R10/R11 + 精确SHA CI | 否 | 相关实现无diff，历史原始证据仍适用 | — |
| AUTO04 Go fmt-check | PASS → PASS | R10/R11 + 精确SHA CI | 否 | 相关实现无diff，历史原始证据仍适用 | — |
| AUTO05 MySQL 集成测试 | PASS → PASS | N1/N3/N6 + CI/R10 | 部分 | 新定向验证；未改动模块CI及历史全套 | — |
| AUTO06 100/1000 牌局专项测试 | PASS → PASS | R10/R11 + 精确SHA CI | 否 | 相关实现无diff，历史原始证据仍适用 | — |
| AUTO07 前端 TypeScript 类型检查 | PASS → PASS | R10/R11 + 精确SHA CI | 否 | 相关实现无diff，历史原始证据仍适用 | — |
| AUTO08 客户端单元测试 | PASS → PASS | R10/R11 + 精确SHA CI | 否 | 相关实现无diff，历史原始证据仍适用 | — |
| AUTO09 安全组件测试 | PASS → PASS | N1/N3/N6 + CI/R10 | 部分 | 新定向验证；未改动模块CI及历史全套 | — |
| AUTO10 Cocos Creator Smoke | PASS → PASS | R10/R11 + 精确SHA CI | 否 | 相关实现无diff，历史原始证据仍适用 | — |
| AUTO11 Production Compose Runtime | PASS → PASS | R10/R11 + 精确SHA CI | 否 | 相关实现无diff，历史原始证据仍适用 | — |
| AUTO12 数据库架构边界测试 | PASS → PASS | R10/R11 + 精确SHA CI | 否 | 相关实现无diff，历史原始证据仍适用 | — |
| AUDIT01 P0：数据丢失、敏感信息泄露、身份越权、不可恢复的数据破坏 | PASS → PASS | 本报告 §4 / 本轮证据索引 | 新 | 本轮独立记录缺陷与边界，不代表产品无缺陷 | — |
| AUDIT02 P1：重复出牌、状态分叉、错误结算、恢复失败、死锁及关键链路中断 | PASS → PASS | 本报告 §4 / 本轮证据索引 | 新 | 本轮独立记录缺陷与边界，不代表产品无缺陷 | — |
| AUDIT03 P2：边界条件、资源泄漏、错误处理、用户操作异常、可维护性问题 | PASS → PASS | 本报告 §4 / 本轮证据索引 | 新 | 本轮独立记录缺陷与边界，不代表产品无缺陷 | — |
| AUDIT04 严重级别 | PASS → PASS | 本报告 §4 / 本轮证据索引 | 新 | 本轮独立记录缺陷与边界，不代表产品无缺陷 | — |
| AUDIT05 文件及代码位置 | PASS → PASS | 本报告 §4 / 本轮证据索引 | 新 | 本轮独立记录缺陷与边界，不代表产品无缺陷 | — |
| AUDIT06 触发条件 | PASS → PASS | 本报告 §4 / 本轮证据索引 | 新 | 本轮独立记录缺陷与边界，不代表产品无缺陷 | — |
| AUDIT07 可复现步骤 | PASS → PASS | 本报告 §4 / 本轮证据索引 | 新 | 本轮独立记录缺陷与边界，不代表产品无缺陷 | — |
| AUDIT08 实际行为 | PASS → PASS | 本报告 §4 / 本轮证据索引 | 新 | 本轮独立记录缺陷与边界，不代表产品无缺陷 | — |
| AUDIT09 预期行为 | PASS → PASS | 本报告 §4 / 本轮证据索引 | 新 | 本轮独立记录缺陷与边界，不代表产品无缺陷 | — |
| AUDIT10 建议修复方向 | PASS → PASS | 本报告 §4 / 本轮证据索引 | 新 | 本轮独立记录缺陷与边界，不代表产品无缺陷 | — |
| GOAL1 Real platform-code login exchange / session validation, token lifetime and secure private credential management; no fake identity, no tokens in URLs; include local authorized dev login path only if correctly access-controlled. | PASS → PASS | N3/N4/N5/N6/N7 + R10/R11 | 部分 | 正常稳定K1交付路径；完整迁移/外部资格单独不通过 | — |
| GOAL2 Account-scoped rooms: create/join/leave/ready, seats, start hand, authoritative room refresh/reconnect and timeout/duplicate reconciliation; full Cocos interactive flow. | PASS → PASS | N3/N4/N5/N6/N7 + R10/R11 | 部分 | 正常稳定K1交付路径；完整迁移/外部资格单独不通过 | — |
| GOAL3 Fair lifecycle: server-bound commit/reveal key/AAD acquisition, secure entropy + HPKE, trusted beacon, deal, private hand and evidence verification, fail closed on missing security context. | PASS → PASS | N3/N4/N5/N6/N7 + R10/R11 | 部分 | 正常稳定K1交付路径；完整迁移/外部资格单独不通过 | — |
| GOAL4 Playable game: bidding, landlord determination, legal play/pass delegated to authoritative Go rules, turn indicators, cards/selection, clear error and version-retry handling, termination, score/evidence/result UI. | PASS → PASS | N3/N4/N5/N6/N7 + R10/R11 | 部分 | 正常稳定K1交付路径；完整迁移/外部资格单独不通过 | — |
| GOAL5 B-level MySQL recovery: append each accepted live bid/play/pass state transition and dedup + owner epoch/fence atomically **before acknowledgement**; bounded encrypted checkpoint and ordered replay restoring game state, ambiguity handling and owner fencing; address seed and other secret continuity; no plaintext player hands in routine logs. | PASS → PASS | N3/N4/N5/N6/N7 + R10/R11 | 部分 | 正常稳定K1交付路径；完整迁移/外部资格单独不通过 | 稳定根路径通过；不包含 C09/C10 换根资格 |
| GOAL6 Real 3-player repeatable E2E, reconnect and crash-injection: pre-/post-COMMIT kill, response loss, duplicate/out-of-order commands, competing owners, wrong epoch and replay mismatch; preserve every acknowledged action. | PASS → PASS | N3/N4/N5/N6/N7 + R10/R11 | 部分 | 正常稳定K1交付路径；完整迁移/外部资格单独不通过 | — |
| GOAL7 Performance and stability: bounded queues/workers/room memory, P50/P95/P99 measurements and transaction fsync budget on stated hardware, no unknown target claims. | PASS → PASS | N3/N4/N5/N6/N7 + R10/R11 | 部分 | 正常稳定K1交付路径；完整迁移/外部资格单独不通过 | — |
| GOAL8 WeChat Creator package, actual developer tool, approved HTTPS/WSS domains and real handset evidence where access is available. Record platform-owned AppID/domain/device blockers honestly. Preserve Web and Douyin build compatibility. | BLOCKED → BLOCKED | R10/R11 + 本轮能力/环境复核 | 复核 | 缺失条件未因维护CLI或政策文字改变 | 官方工具、核准域名、实际手机及发布条件未全部完成 |

## 9. 交付与未完成条件

**R12-N01/P2 交回 ChatGPT 修复，复测以上真实 CLI 并发/中断反例后再判断本轮代码 PASS。** C09 的完整 HMAC 换根/new-only能力和 C10 的整体迁移/退役仍不能以稳定根政策替代；需按原标准完成，或另经正式需求变更，Codex 未修改标准。E03、H05、GOAL8分别需用户操作体验、正式抖音环境和官方/域名/真机/发布证据，未通过上传或真实用户身份操作擅自补证。

矩阵109 PASS/1 FAIL/4 BLOCKED=114；已发现失败后继续执行其余可行检查。只提交本报告和 current 允许的验收状态段落，保留历史报告；候选源码不变，报告提交SHA由最终总结和 `git log -1 --format=%H -- docs/goals/0046-round12-independent-acceptance.md` 获取。文档交付复核包括逐项证据/状态映射、完整矩阵计数、失败日志不隐藏、源码与文档diff和敏感材料排除，未把文档检查称为产品自修复。

结束后停止本轮API/TLS及复用自有数据库容器，保留专用库卷、私有备份和受限证据；未清理全局Docker、改动Kubernetes或其他部署。Goal0045/0046继续 `in_progress`，不封版、不发布。
