# Goal 0046 — Round 11 增量独立验收

2026-11-06 UTC，执行者 Codex。**两项修复的增量验收 PASS；114 项增量汇总 109 PASS / 0 FAIL / 5 BLOCKED。完整内部产品验收完成条件仍未满足**：C09/C10 是内部密钥运维验证缺口，不能归为纯外部条件。外部平台、真机及正式发布资格继续未完成，Goal 0045/0046 保持 `in_progress`。

## 版本、范围与复用依据

产品候选固定为 **`e83f311b7305b0ae22d4a221bd75b0acef825779`**。clean main 从上轮报告提交 `e2b5361ad87ad54a1cbdc8014da22ad75f6a13a9` 执行 `git pull --ff-only origin main`，同步到候选；结束验证时仍为该 SHA。报告提交 SHA 与产品 SHA 分开，报告 SHA 由交付总结及 `git log -1 --format=%H -- docs/goals/0046-round11-independent-acceptance.md` 获取。

本轮按用户指定低额度策略，阅读 AGENTS、相关原则、current、修复交接及 [Round 10 报告](0046-round10-independent-acceptance.md) 两项缺陷，核对实际差异。产品改动仅涉及 `application/hand_commands.go` 终止事务顺序和 `application/room_queries.go` 历史 locator 可见性；其余为相关测试及文档。未更改产品、正式测试、协议、schema、CI 或客户端构建配置。

独立查询并复用本候选的 [CI 42153250351](https://github.com/oublie6/awesome-zero-platform/actions/runs/42153250351) **6/6 SUCCESS**、[Fair Doudizhu 42153250338](https://github.com/oublie6/awesome-zero-platform/actions/runs/42153250338) **4/4 SUCCESS**，见私有 `ci.json`。未重跑全量 CI、114 项完整测试、Cocos/引擎导出、规模性能、完整 SIGKILL 矩阵、公平协议全流程或两局完整游戏；未变更模块复用 Round 10 已有证据，CI 仅作支持。

环境复用 Linux x86_64、Go 1.25.8、Node 22.23.4、自有 MySQL 5.7.44/Redis 8.8.0 容器及数据库；Go `GOMAXPROCS=2 -p 1 -parallel 1` 顺序执行。本候选 API 重新编译，隔离进程上限 512 MiB，回环 HTTPS/WSS 使用原私有测试证书。直接使用未变更生产客户端代码的现有 dist，未声称本轮原生 GUI/SDK 复测。

私有证据根 **`$E=/root/.config/awesome-zero-platform/acceptance-0046-r11-e83f311`**，目录 0700、证据文件 0600；`runs.jsonl` 记录实际命令、源码 SHA、时间、耗时和退出码，`sha256.json` 为证据索引。复用上轮完成局/账号/配置，重新正常密码登录四个真实 Go 账号并验证 session；没有伪造微信身份。密钥、JWT、私牌及原始数据库内容未提交。

## R9-N01 / P1：PASS

独立阅读实际调用顺序：加载/恢复 Hand → 域终止校验 → 加载/恢复 Room → `Room.FinishHand` 校验 → 锁实时 writer 行、检查 committed version → fence → 更新 Hand/Room → 同事务提交。已知业务拒绝仍按原契约持久化拒绝回执，但不再提前 fence。

| 必需场景 | 本轮真实 MySQL 观察 |
|---|---|
| 过期 ExpectedVersion，正确 ExpectedLiveVersion | `DDZ_VERSION_CONFLICT`；epoch **2→2**、watermark **1→1**、Hand **BIDDING/v12**、Room **HAND_ACTIVE/v7** 不变；outbox **0→0**、receipt **0→1**、storedAccepted=false；后续合法 Bid 成功至实时 v2 |
| 合法到期与同 ID 重试 | 实际生命周期 Sweep 成功，epoch **2→3** 恰好一次；同 commandId 重试 accepted=true/duplicate=true；再次 Sweep 无额外 fence，归档恰好一份 |
| 旧 writer 与并发提交 | 实际 `INNODB_LOCK_WAITS` 可观察；writer 被终止事务行锁阻塞，提交后旧 writer 写入被拒绝；若合法实时提交先赢，旧到期判断回滚、不 fence |
| Room.FinishHand 拒绝 | 使用不匹配 activeHand 的有效 Room 快照；拒绝后 epoch **2→2**、watermark **1→1**、Hand 不变、archive/outbox 均 0，只有一份拒绝回执；恢复房间夹具后原 writer Bid 可继续 |
| fence 后故障回滚 | 真实 SQL 事务内注入错误后，无部分 epoch/业务状态/事件更新；原合法 writer 可继续 |

本轮重跑上轮独立外置 SQL 反例并检查数据库实际值，未只采用开发方新增测试。证据：`expiry-independent.log`、`expiry-room-corrected.log`、`expiry-old-writer.log`，相关新增正式测试见 `associated-official.log`。保留上轮关于内部调用前提的限定，不宣称曾证明正常生产 Supervisor 自然触发所有拒绝组合。

## R10-N01 / P2：PASS

使用 Round 10 真正完成的 A/B/C 历史局 H1，当前候选真实 API、正常认证和生产客户端 controller 执行：

- A/B/C 均可读 H1；C 实际退出、D 实际加入后，D 为当前三人房间成员，room view 不含 H1 的 `lastHandId`。
- D 实际 WSS `reconnectAndResync` 成功，计数确认 **0 次历史归档请求**；D 直接读取 H1 仍 **HTTP 403**。A/B 及已离房的 C 仍可读 H1，C 的非成员 room view 不含私有席位/历史 locator。
- A/B/D 实际准备并开始 H2。为避免重跑无关公平协议/完整两局，在既有仓库确定性 DEALING/BIDDING 夹具处接入该实际账号/房间；随后执行生产 `AbortHand`、真实 MySQL 终局归档。H2 终止后 A/B/D 可读自身终局证据，C 读 H2 **403**，A/B/C 读 H1 仍成功，D 自身终局重连成功。
- `lastHandId` 指向不存在的历史 Hand：省略 locator，重连成功；历史快照损坏：实际接口显式报错，未伪装成功。故障注入数据均恢复，最后恢复 A/B/C 原房间成员关系。

H2 是**权限与终局查询集成夹具**，使用真实域/runtime/archive，但不启用 durable journal，不作为新一轮完整公平、正常打牌结算或恢复资格证明；这些未变更路径引用 Round 10。证据：`history-prepare.json`、`history-verify.json`、`history-restart.log`、`own-terminal-fixture-final.log`、`history-verify-final.log`。

额外关联观察保留：只临时移走归档行、保留有效终态 Hand 时，room view 仍含该参与者有权的 locator，重连显式 **404**；探针的更强“归档单独丢失仍可无错重连”指标 `prunedArchivePass=false`，不能声称该能力通过。现有正式 `live_retention.go` 清理仅删除恢复材料/日志，明确保留最终归档及房间/公平记录；该故障不是正式 prune 行为，也不是新成员越权/403 原问题。本任务的异常读取不得伪装成功条件满足，未将该额外可用性期望升级为既定需求。归档恢复后再次读回成功。本轮不增加归档丢失修复或承诺降级恢复能力。

## 最小关联回归与执行结果

Go 命令公共参数：`APP_API_INTEGRATION=1 GOMAXPROCS=2 go test -json -count=1 -p 1 -parallel 1 -tags=integration`，包 `./business/doudizhu/infrastructure/mysqlstore`；独立探针另加 `-overlay $E/overlay.json`。完整实际命令见 `runs.jsonl`。

| 执行范围 | 有效结果 |
|---|---|
| 独立 SQL：RejectedMustNotFence、RejectedExpiryStateAudit、ActualSweepTerminatesOnce、RollbackAfterFence、CommitBetweenDecisionAndWriterLock、RoomRejectionBeforeFence、WriterLockBlocksThenFences | **7 PASS**，只执行这七项直接关联用例 |
| 正式定向 Go：RejectedExpiryDoesNotFenceLiveWriterInRealMySQL、HistoricalArchiveLocatorDoesNotLeakToReplacementMemberWithRealMySQL，以及 application 的成员替换/非成员投影两项 | **4 PASS**，application/mysqlstore 两包定向执行 |
| `node --test clients/fair-doudizhu-cocos/test/online.test.mjs` | **7 PASS / 0 FAIL / 0 SKIP**，包含新成员无旧 locator 重连 |
| 生产 API `CGO_ENABLED=0 go build -p 1 -trimpath ... ./apps/app-api` | 成功；随后实际 HTTPS/WSS 权限/重连场景通过 |
| H2 终局权限夹具 | **1 PASS**；边界和证据限制见上一节 |

保留初始无效探针记录：Room 拒绝探针误要求“拒绝回执也为 0”，已纠正为正常拒绝回执 1，其余不变性断言保留；H2 夹具初次使用错误阶段名称、下一次已被正常公平超时终止，重新实际 start 并立即运行后通过；边界 SQL 初次连接 collation 不匹配，显式匹配后通过。均为隔离探针/编排问题，无产品修改，不把这些初始退出码抹去或计作产品缺陷。

## 114 项矩阵增量

以 Round 10 原始 114 ID 为基线。下表 10 项本轮关闭对应失败原因；其他 **99 PASS** 引用 Round 10 原始逐项证据及当前未变更模块 CI，**5 BLOCKED** 保留。不是声称本轮重新执行了 114 项。客户端相关行结合 Round 10 原生 UI 正向证据及本轮相同生产 controller 的成员替换修复验证；未重新授予原生/真机资格。

| ID | Round 10 → Round 11 | 本轮关闭依据 |
|---|---|---|
| A02 | FAIL → PASS | 实际退出/加入、历史权限隔离 |
| A08 | FAIL → PASS | 拒绝终止不 fence，合法 Bid 继续 |
| A10 | FAIL → PASS | D 实际重连不请求无权旧归档 |
| B06 | FAIL → PASS | SQL epoch/水位/业务版本不变性、回滚 |
| B13 | FAIL → PASS | 合法 fence 一次、旧 writer 拒绝、精确重试 |
| H06 | FAIL → PASS | 生产 controller 与真实 HTTPS/WSS 操作一致 |
| H08 | FAIL → PASS | 新成员状态恢复无旧归档 403 阻断 |
| GOAL2 | FAIL → PASS | 关联房间加入/准备/开始/重连修复闭环 |
| GOAL4 | FAIL → PASS | 生命周期拒绝后合法动作继续，原可玩性证据复用 |
| GOAL5 | FAIL → PASS | 拒绝/成功终止的 durable writer 原子边界修复 |

**109 PASS + 0 FAIL + 5 BLOCKED = 114**；私有 `matrix-delta.json` 保留全部 ID、历史来源和本轮更新标记，原始描述/历史日志仍在 Round 10 报告。

| 保留 BLOCKED | 原因与完成资格影响 |
|---|---|
| C09 | 完整 journal HMAC 换根后的 new-only 冷恢复缺受支持操作流程及证据；属于内部运维验证缺口 |
| C10 | 旧 HMAC 根依赖、完整密钥库存/全量迁移/核准退休流程未验证；属于内部运维验证缺口 |
| E03 | 人类备份确认及实际备份恢复 UX 资格尚未完成 |
| H05 | Linux 抖音导出/SDK 兼容资格尚未完成 |
| GOAL8 | 官方工具、核准 HTTPS/WSS 域名、实际手机/发布资格未全部完成 |

## 结论与交付

**R9-N01、R10-N01 本轮独立增量修复验收 PASS，必要关联回归无有效产品 FAIL，未建立新的 P1/P2 缺陷。** 不把额外归档丢失故障中的 404 说成无错恢复。本轮报告任务通过，不等于 Goal 整体验收或正式发布通过；C09/C10 尚未满足“剩余阻塞仅限外部平台/真机/发布”的内部完成条件，完整内部资格继续 **BLOCKED**。不修改或降低原有标准，Goal 0045/0046 保持 `in_progress`。

只提交本报告及 current 允许的验收状态文字，历史报告保留。文档提交前复核候选 SHA、矩阵计数/证据映射、最终 diff 和敏感信息排除。本轮启动的 API/TLS 进程及复用的自有数据库容器已停止，数据库卷及历史证据保留；未改动其他部署。
