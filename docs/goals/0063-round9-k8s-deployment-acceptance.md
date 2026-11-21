# Goal0063 Round9 — P1 修复验证与 Kubernetes 增量部署验收

**结论：PASS（本轮测试环境范围）。G63-R8-N01 / P1 已独立复验关闭。** 固定产品 `9581630174eea848f44580cc4c416a7073442a88` 已部署到 `kubernetes-admin@network-learning` / `awesome-zero-game-test`，DQ 保持启用。真实自动不叫、最小合法单张、Pass，以及实际 Cocos Bid/Play/Pass UNKNOWN 核对与精确重试均通过。最终六个 Pod、两个节点 Ready，无活跃牌局、deadline cursor 或队列积压。

日期：2026-11-21 UTC。Codex 只执行独立验收、测试部署和文档交付，未修改产品源码、合并代码或操作生产环境。保留 [Round8 FAIL](0063-round8-k8s-deployment-acceptance.md) 与 [Round7 代码/客户端 PASS](0063-round7-independent-acceptance.md) 的原始结论。[本轮脱敏证据](evidence/0063-round9/result.json)；原始日志、账号、备份、配置、截图和详细行哈希仅保存在执行机私有目录。

## 1. 固定源码、CI 与镜像

开始时 clean `main`，`git pull --ff-only` 同步到完整产品 SHA。阅读 `AGENTS.md`、current goal、四篇原则、Goal0063、Round8 及其引用证据。先在隔离环境验证修复及持久化保护，再部署共享测试环境。

| 固定 SHA 的工作流 | 独立读取的结果 |
| --- | --- |
| [CI / 44929912373](https://github.com/oublie6/awesome-zero-platform/actions/runs/44929912373) | unit、secure_envelope、integration、admin_web、runtime、report，6/6 SUCCESS |
| [Fair Doudizhu / 44929912358](https://github.com/oublie6/awesome-zero-platform/actions/runs/44929912358) | mysql_integration、client_foundation、manifest_client、application，4/4 SUCCESS |

API：Go 1.25.8，`CGO_ENABLED=0`，`GOMAXPROCS=2`、`GOMEMLIMIT=768MiB`，`go build -p 1 -trimpath -ldflags='-s -w'`。build info 的 `vcs.revision` 为固定 SHA，`vcs.modified=false`。

- API 二进制 SHA256：`2d125dac9becc1f11d56c1b6f71b6c0827363fee4b23ce604854758e82ced707`。
- 新 API：`docker.io/awesome-zero-platform/goal0063-api@sha256:04614d0bc04c1fe37d2cb4467be23fec728cb0b1cfb63c26b1cd3746943ec82f`。
- 复用 Web/Cocos：`docker.io/awesome-zero-platform/goal0063-web@sha256:990da69543f3501283a0958fe26919ba93eb289ad139924791e6988421ed4e12`。
- 保留 Beanstalkd：`docker.io/awesome-zero-platform/goal0063-beanstalk@sha256:77b1bfeb33adf4f591f50e090be8ab56441ac3f7f6ed25aa6986d6a0dc3092fe`。

镜像通过原已验证 distroless 运行层打包、导入目标节点 containerd，并按不可变 manifest 部署；未宣称上传公共 registry。二进制复制时保留 0755 执行权限。

`git diff 9474b9a 9581630 -- clients scripts deploy` 为空。复用 Round8 已部署验证的 Web 镜像，其中官方 Linux Cocos 导出来自 Round7 的相同客户端输入；没有重建未变化的 Cocos、Vue、Beanstalkd，也没有把历史 native_export skipped 改称本轮构建成功。最终通过实际容器进程根文件系统读取运行中的 API 文件哈希，并核对 Pod imageID；30 个部署中的 Web/Cocos 文件全部匹配已验证产物。

## 2. 原 P1 的真实运行时独立复验

隔离环境使用真实 MySQL 5.7.44、Redis、两份持久日志 Beanstalkd 和固定 API 二进制，单独的 `goal0063r9` 数据库与认证/队列前缀。复用 Round8 自建依赖和备份，未重置旧故障库、共享数据库、Redis 或队列。

每项都使用真实公平贡献和公共信标开局；副本回合配置为 20 秒。为明确观察“拒绝后、自动动作前”的状态，在原截止时间附近短暂暂停**仅副本的两个队列**，不修改 MySQL deadline，发送当前 seat 的合法普通命令，再恢复队列并验证真实 DQ 动作。

| 晚到人工命令 | 发出时间相对真实 deadline | 返回 | 拒绝后的验证 | 后续自动动作与下一玩家 |
| --- | --- | --- | --- | --- |
| Bid | +24ms | HTTP409 / CONFLICT | PublicView、PrivateView 均200；v1不变；无该 action、无已接受回执；deadline不变 | DQ提交不叫到v2；下一玩家合法叫3分到v3 |
| Play | +28ms | HTTP409 / CONFLICT | 两视图200；v3不变；无该 action/已接受回执；deadline不变 | DQ提交确定性最小单张到v4；下一玩家合法Pass到v5 |
| Pass | +25ms | HTTP409 / CONFLICT | 两视图200；v5不变；无该 action/已接受回执；deadline不变 | DQ提交Pass到v6；下一玩家合法出牌到v7 |

三条自动动作都对应唯一的 expectedVersion/seat/action，提交不早于 MySQL 截止时间。由于人为暂停队列和逐项检查，副本自动动作相对 deadline 的观测延迟约 4.1–4.4 秒；这不是正常共享部署的延迟结果。随后仅结束自建副本牌局并保留归档。

源码复核：只有 MySQL adapter 在 action INSERT/COMMIT 之前确定发生的 `ErrTurnExpired` 被标记为 `gamecore.ErrCommitRejected`；gamecore 丢弃候选但保持原实例可用；HTTP/WSS 映射为既有冲突协议。没有对 SQL 故障、watermark 冲突、writer fencing 或未知 COMMIT 结果套用该安全标记。

## 3. COMMIT 歧义、fencing 与定向回归

先完成以下隔离运行时检查，全部通过后才更新共享 API：

- 合法提前 DQ 票据、旧版本、错误 seat 和不存在 hand 的票据均不推进当前状态；普通错误 seat/版本请求返回冲突，视图仍可读。
- 已接受请求按原 requestId/clientSeq 精确重试，重启后仍只有一条 action；换 requestId 重用 clientSeq、改变同 requestId 的 payload 均拒绝。终态票据安全忽略。
- MySQL TCP relay **实际丢弃 live action 事务已提交后的 COMMIT OK**：API500、不返回已接受结果；数据库已提交一条 action，但旧内存视图404，保持 fail-closed。冷恢复取得更高 epoch，精确重试返回原成功结果，action仍一条、原 deadline 不变。
- MySQL 连接暂时不可用：不产生已接受的未持久化动作，恢复/重放后视图可读。
- 仅对自建副本牌局注入 ownership fence：旧 epoch 写入被拒绝，旧视图隔离，无旧 action；冷恢复取得更高 epoch。未更改共享 writer 行。

定向命令均独立执行且通过，未重跑全量 Go/客户端验收：

```bash
go test -p 1 -parallel 1 ./business/gamecore ./apps/app-api/internal/doudizhuapi \
  -run 'TestCommitted|TestCertifiedPreCommitTurnRejection' -count=1 -v

# APP_API_INTEGRATION=1，其他数据库环境变量来自私有副本配置。
go test -p 1 -parallel 1 -tags=integration ./business/doudizhu/infrastructure/mysqlstore \
  -run 'TestDQTurnDeadlineEnforcesHumanTimeoutAndAtomicTurnAdvance|TestDQExpiredHumanBidKeepsCoordinatorLiveAndAutoBidCommits|TestLiveJournalFenceContiguousLogAndProtectedSeedInMySQL' \
  -count=1 -v
```

## 4. 共享环境门禁、备份和增量部署

确认 Round8 回滚基线 API/Web 健康，原 MySQL/Redis、两个队列及 PVC 正常；没有不兼容活跃牌局。原备份/恢复、5.7 additive DDL 和双 PVC 重启证据继续有效。本轮没有对共享库执行 DDL。

另取部署前当前完整 22 表的一致性备份，实际恢复到独立副本中的 baseline 数据库；逐表按主键排序比较原始字节，22/22 相同。补存原有 7 个会话键、值哈希和 TTL；敏感内容不进入报告。保存当前资源、原镜像及回滚配置。

执行顺序：

1. API 以新 digest、DQ=false、Recreate 单实例切换；Web 更新为已验收的相同客户端镜像。
2. 通过升级前建立的同一历史会话检查账号、AccountID、历史最终证据及 readiness，均200。
3. 再检查全库未完成牌局为零、双队列 Ready；启用 DQ，以 Recreate 避免同一活跃牌局被新旧调度器同时处理。
4. 核对运行 Pod digest、API 实际文件和30个客户端文件；检查旧会话和历史业务。

```text
APP_DOUDIZHU_TURN_DQ_ENABLED=true
APP_DOUDIZHU_TURN_DQ_ENDPOINTS=beanstalkd-a.awesome-zero-game-test.svc:11300,beanstalkd-b.awesome-zero-game-test.svc:11300
APP_DOUDIZHU_TURN_DQ_TUBE=doudizhu-turn
```

保留原 A/B StatefulSet、Service、两个独立 1Gi/RWO/Retain PVC。没有清队列、删 PVC、重建 MySQL/Redis 或改动其他 Namespace。Round8 每队列遗留的2条终态消息与1条 cursor 经实际消费/repair 安全退休，未复活旧牌局；新 QA 开局前已确认队列空、cursor为零。

## 5. Kubernetes 真实自动回合业务

使用新普通 QA 账号、新房间与新牌局。真实 Vue 登录、Cocos 创建/加入/三方准备/开局/三方贡献通过。共享配置保持 **BiddingTimeout=300s、PlayingTimeout=300s**。

| 动作 | MySQL deadline（UTC） | 实际提交（UTC） | 误差 | 持久化与客户端 |
| --- | --- | --- | --- | --- |
| 自动不叫 | 18:30:39.661 | 18:30:40.087 | +426ms | v1→v2，score=0，仅一条action；Cocos更新 |
| 自动最小合法单张 | 18:35:48.221 | 18:35:49.110 | +889ms | v3→v4，严格匹配该玩家持牌中最小 card ID；一条action；Cocos更新 |
| 自动Pass | 18:41:03.099 | 18:41:04.066 | +967ms | v5→v6，存在合法上家牌；一条action；Cocos更新 |

逐项核对持久化 expectedVersion、当前 seat、系统 commandId、payload 和提交时间，未用固定等待时长推断成功。后续玩家能正常继续叫分、出牌/Pass；没有错误终止、重复结算或版本回退。

实际页面倒计时随时间变化；刷新、重新进入、WSS 重连及 DQ 同模式 API Pod 重建后，原 v1 MySQL deadline 和客户端投影保持一致。测试结束仅通过业务接口结束自建牌局，保留 ABORTED 最终归档。

## 6. DQ 可靠性和故障恢复

以下危险故障均在专用副本进行，未对共享 MySQL/Redis 做故障注入。票据采用正确的 **纳秒时间戳 + `/` + JSON** go-queue 信封和合法 Ticket 字段：

| 用例 | 实际证据 |
| --- | --- |
| deadline边界人工命令与8份重复合法票据竞态 | 同一版本只有一次有效推进；无隔离、重复动作或回退 |
| 单个队列A停机 | B/repair仍提交最小合法单张，观测延迟943ms；恢复A后不重复 |
| 双份真实延迟消息丢失 | 只删除当前自建 hand/version 的3份真实投递，不清空队列；API重启保留deadline，MySQL repair补投后只提交一次Pass，延迟3558ms |
| 消费回调时 MySQL relay 不可用 | 观察到真实回调失败；不可用期间无对应action；恢复后 durable repair提交一次自动不叫，故障下延迟6325ms |
| 旧版本/错误seat/旧hand、重启重消费、ACK丢失 | 第2、3节及实际终态退休检查通过 |

首次回调恢复观察碰到配置中的15分钟访问令牌自然到期（HTTP401），不计为产品失败或该项通过。只续建自有 QA 会话、结束该局并单独重跑受影响检查；重跑通过，原始失败与其他三项已通过证据均保留。

最终没有手动清空队列。等待所有本轮已终态牌局的原始延迟票据自然到期，由应用安全忽略/清理；最终 A/B ready、delayed、buried 均为0，deadline表为0，未出现无法恢复积压。

## 7. 必要 Web/Cocos 回归

- 实际 Vue/Cocos 新账号、创建、加入、准备、开局、贡献、自动倒计时及自动回合更新通过。
- **Bid、Play、Pass** 的实际已接受 WSS ACK 被观察器丢弃，界面进入稳定 UNKNOWN；通过真实 RetryUnknown 控件核对/重试。每项都观察到相同 requestId、clientSeq、expectedVersion、payload 至少发送两次，但数据库对应 action 只有一条；界面恢复到已提交版本。
- 在 **760×390 窄横屏、390×844 竖屏** 检查部署后的 UNKNOWN 标签、字体、控件边界和手牌/按钮分离，实际点击通过。复用 Round7 完整布局矩阵，不重复全部客户端测试。
- 历史账号在实际 Vue 登录、Cocos 完成牌桌和“查看结算证据”入口通过，显示“已核验归档”。HTTP证据读取在升级前、DQ关闭、DQ启用和最终会话刷新后也均200。
- 浏览器使用原私有测试 CA 的正常信任库；Node/curl 指定 CA；没有 ignoreHTTPSErrors、关闭证书校验或把 Token 放入 WS 查询参数。

**观察脚本纠正及边界：** 首轮复用 Round8 QA 账号时恢复了已有 managed room；CreateRoom 的 existing-room guard 拒绝新建，脚本又过早采用旧 lastHandId。该自有房间发生真实准备/开局操作，缺贡献的新 QA 局自然到期 ABORTED；未涉及真实用户牌局。随后用新账号、新 roomId 和服务端 activeHandId 重跑完整流程通过。

UNKNOWN 脚本还先后遗漏 pending 字段、未区分 submitting 与稳定 UNKNOWN、未等待窗口恢复后的布局稳定，且未确认重试点击是否实际发送。原始失败保留；修正私有观察器后，实际事件、同请求身份、重试结果和单条持久化 action 均重新验证。Bid 单独补跑，Play/Pass 在原主 QA 局补跑。没有修改产品源码、注入客户端成功状态或把原脚本失败改写成成功；不将缩放瞬间的输入竞态定性为已验证产品缺陷或设备资格。

## 8. 历史数据、会话、密钥与 TLS

对当前备份 baseline 里的全部原主键记录做独立比较：**22表中18表原记录字节相同**。另外四表的变化逐行限定为：

- `doudizhu_live_writers`：4行仅 owner_epoch/updated_at 的单调 recovery/fencing 元数据变化。
- `doudizhu_rooms`：仅首轮复用的原自有 QA 房间1行，经真实 UI 操作合法推进 aggregate_version/snapshot_json/updated_at；owner、原历史牌局没有改写。
- `doudizhu_client_sequences`：仅该自有 QA 房间三个原 QA actor 的 last_sequence/updated_at 单调推进。
- `doudizhu_turn_deadlines`：原 Round8 已归档 QA 牌局的 cursor 正常退休；对应手牌与最终归档保留。

没有意外原记录变化或缺行。原账号、密码凭据、历史 hand/action、贡献保护记录、seed/checkpoint 和最终归档保持一致；新增自有 QA 记录保留。没有回写备份来伪造原 QA 房间哈希相同。

原7个持久化 session 值字节相同，TTL 未延长；升级前建立的历史会话在两次切换后可用，最终原 refresh session 也成功刷新。没有全局撤销会话或重置 Redis。

Secret 全字段与部署前、Round8 相同，原 ConfigMap data、Service spec 不变。当前 Reveal `g61-reveal-v2-20261115` 有效至2026-12-15，历史 retiring key和签名/贡献保护根保留。TLS leaf 保留，有效期2026-11-15至2036-11-12；没有轮换或替换历史材料。

## 9. 回滚、最终运行状态与资源

在没有活跃牌局的独立副本，实际启动旧 `a739e23` API（DQ=false），再启动修复后 `9581630` API（DQ=true），两次 readiness 均正常，新增表、原日志/保护材料保留。共享完整回滚和历史业务验证复用 Round8；本轮没有在活跃共享 DQ 牌局上切换旧调度器。

回滚镜像仍保留：API `sha256:9b8bf32b89c384c7f6760473c0f29b72999ed3b2ffff36876693ff2e6832906f`，Web `sha256:ab52cee0608128afef4af6466c90bb7141b92d4b43e51bbc34ab165a537caab2`。原 spec、备份和配置私存于 `/root/goal0063-round9-9581630`。回滚前必须阻止新开局并安全收敛已知自有 QA 局，再核对全库无活跃/在途牌局；使用 Recreate 单 writer 切换。禁止在活跃 DQ 牌局上直接 DQ=false/切旧 API，禁止共享库备份覆盖、删表/PVC/密钥或清队列。

| 最终 Pod | 状态 |
| --- | --- |
| app-api-99f88b95-dxrwx | 修复后 digest，1/1、0 restart，DQ=true |
| game-web-7f596dc6d6-6jgcq | 复用已验收 Web digest，1/1、0 restart |
| beanstalkd-a-0 / beanstalkd-b-0 | 各1/1、0 restart；原 PVC Bound |
| mysql-5cc48fd9b4-5hz6j / redis-67bdd89687-krvb4 | 各1/1；原 UID、原2次restart均未增加 |

四个 Deployment、两个 StatefulSet 各一副本，两个节点 Ready。原 MySQL/Redis/队列 Pod 未重建。应用内存快照 MiB：API约15.1→14.2，MySQL195.8→201.4，Redis7.0→9.3，Web11.1→29.4，队列各约0.6–0.7；均在原资源限制内。没有 OOM/异常重启或不可恢复积压证据；这是本轮有界观测，不是长期 soak/SLO 或任意节点容量资格。

原入口 `https://127.0.0.1:29457` 仍由 `goal0063-game-test-forward-v2.service` 提供。Web Pod 替换时旧隧道连接关闭，自动重连后验证200，门禁期间未因此强行启用DQ。该 unit 仍为 transient，未验证宿主机重启恢复。两个队列仍在同一控制节点、独立进程/Service/PVC；未宣称跨物理节点容灾。新镜像仅导入实际调度节点。

所有自建副本 API、relay、浏览器和 Docker 依赖已停止，数据卷及私有备份保留；共享测试环境继续健康运行。

## 10. 结论与交付复核

**Round9 PASS；G63-R8-N01/P1 CLOSED；修复后 Kubernetes 增量部署成功。** Round8 未完成的重要自动操作、合法票据、故障/repair、ACK/UNKNOWN 和必要客户端路径已有实际证据。无待修复的新阻塞产品缺陷。

对验收脚本及报告做第二遍自检：核对固定 SHA、CI、manifest/运行文件、deadline/版本/seat/单action、真实冲突与不确定提交的不同语义、故障隔离、原数据差异的逐行范围、会话/密钥、终态票据自然退休，以及观察器原始失败与实际补跑证据。没有用观察器失败推断产品通过，也没有依赖 mocked success。

这不授予真机、微信平台、生产域名/发布、跨宿主机容灾或长期容量资格。Goal0045/0046 历史状态和此前报告不改写，不封存其他 Goal。提交范围仅本报告、脱敏证据和 current.md 允许的 Status 记录；未提交账号、Token、密码、私钥、原始备份、临时 SQL 或截图。
