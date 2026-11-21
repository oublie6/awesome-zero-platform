# Goal0063 Round8 — Kubernetes 测试部署与业务验收

**最终结论：FAIL。** 固定产品 `9474b9a4b2309cded0fc1dc1664650341d3d03e3` 实际部署后，隔离副本复现 **G63-R8-N01 / P1：过期人工操作使内存实例隔离，DQ 无法继续推进**。停止扩大测试，通过业务接口结束唯一共享环境 QA 牌局，确认未完成牌局为零后，实际回滚 API/Web 到部署前镜像。现有服务健康，原数据、密钥和新增持久卷保留。ChatGPT 负责产品修复；Codex 未修改产品源码。

验收日期：2026-11-21 UTC。操作范围仅 `kubernetes-admin@network-learning` / `awesome-zero-game-test`，以及本机自建故障副本。没有生产环境、其他 Namespace 或其他业务服务变更。证据：[脱敏结果 JSON](evidence/0063-round8/result.json)。数据库备份、账号、配置、Secret 快照、故障数据库和原始日志均保存在执行机私有目录，未提交 Git。

## 1. 固定基线、阅读范围与 CI

开始时检查 clean `main`，执行 `git pull --ff-only`，同步至产品 SHA `9474b9a4b2309cded0fc1dc1664650341d3d03e3`；发布报告前再次同步，无新增远程提交。PR #18 已合并。报告提交与被测产品 SHA 分开。

阅读 `AGENTS.md`、`docs/goals/current.md`、`docs/principles/README.md` 及四篇原则、Goal0063、[Round7 报告](0063-round7-independent-acceptance.md)、DQ 部署 overlay，以及 Goal0061/0062 已有 K8s 报告与保护材料。复用 Round7 代码验收，不重复全量 Go、Race 或无关功能测试。

以下运行均绑定完整产品 SHA，状态为 SUCCESS：

| 工作流 | 实际结果 |
| --- | --- |
| [CI / 44864235370](https://github.com/oublie6/awesome-zero-platform/actions/runs/44864235370) | integration、secure_envelope、unit、admin_web、runtime、report，6/6 success |
| [Fair Doudizhu / 44864235366](https://github.com/oublie6/awesome-zero-platform/actions/runs/44864235366) | app、manifest、client、MySQL，4/4 success |
| [Cocos Linux / 44864235645](https://github.com/oublie6/awesome-zero-platform/actions/runs/44864235645) | linux_source success；native_export **skipped** |
| [Vue3 Web / 44864235323](https://github.com/oublie6/awesome-zero-platform/actions/runs/44864235323) | vue_build success |
| [Kubernetes manifests / 44864235376](https://github.com/oublie6/awesome-zero-platform/actions/runs/44864235376) | success |

native_export skipped 不计为导出成功。实际使用 Round7 已通过的官方 Linux Cocos 导出缓存：`git diff 8b9465d 9474b9a -- clients scripts` 无差异；Vue 资源对应的 `clients/game-web`、`clients/packages/game-client-core` 与已验收 `a739e23` 无差异。没有执行 Creator 3.8.8、Wine 或另一套系统工具链。

## 2. 镜像、构建与实际 Pod

API 用 Go 1.25.8、`CGO_ENABLED=0`、`GOMAXPROCS=2`、`GOMEMLIMIT=768MiB`、`go build -p 1 -trimpath -ldflags='-s -w'` 编译固定源码。`go version -m` 记录 `vcs.revision=9474b9a…`、`vcs.modified=false`；二进制 SHA256 为 `1003f3cb3b0c2973d5f38b6bc294c17da24d9351a573c104a57516b06056f2d8`。沿用现有已验证 distroless 运行层，覆盖二进制与固定源码配置。Web 沿用原 Nginx/TLS/CSP 配置，装入上述已验证客户端缓存。

Beanstalkd 直接构建仓库 `deploy/docker/beanstalkd.Dockerfile`，实际安装 `beanstalkd 1.13-r0`。构建、导入和验证顺序执行，镜像留在目标节点 containerd，Deployment/StatefulSet 使用不可变 manifest 引用；未宣称上传公共镜像仓库。

| 新镜像仓库（前缀 `docker.io/awesome-zero-platform/`） | manifest digest |
| --- | --- |
| goal0063-api | `sha256:5e7f3a4b142556130ac0edd651124585a5e3d68b4c8c7c4cc9669224de6e27bc` |
| goal0063-web | `sha256:990da69543f3501283a0958fe26919ba93eb289ad139924791e6988421ed4e12` |
| goal0063-beanstalk | `sha256:77b1bfeb33adf4f591f50e090be8ab56441ac3f7f6ed25aa6986d6a0dc3092fe` |

实际业务测试使用 `app-api-55d9b88b9d-rx6mh`、`game-web-cd7598589-8lmh6`；Pod imageID 与对应 manifest 一致。先以 DQ=false 升级并验证历史业务，再在未完成牌局为零、队列就绪时启用 DQ。API 用 Recreate 单实例切换，未并行运行新旧调度器。

首次私有打包遗漏二进制执行权限，Pod 出现 `RunContainerError / permission denied`，未提供健康服务；补齐打包文件的 0755 权限后重建并使用表中摘要。失败镜像未计入成功部署，也未修改产品源码来绕过问题。

## 3. 部署前检查、备份与 MySQL 5.7 增量

部署前 API、Web、MySQL、Redis 各一副本，均 Ready；无 StatefulSet/PVC。MySQL 原 hostPath、Redis 生命周期、原 Secret/ConfigMap/Service 均有私有快照。原始镜像摘要见第 9 节。两个节点均 Ready：`ser730417996259`、`hk-worker`。

原库 `goal0057` 共 21 张 InnoDB 表，10 个账号、5 个房间、3 个历史牌局，3 局均 COMPLETED，未完成牌局 0。保存原 schema、表计数、主键排序字节哈希及原资源配置。

用 `mysqldump --single-transaction --quick --hex-blob --routines --triggers --skip-add-locks --set-gtid-purged=OFF` 一致性备份。SQL 为 580721 字节，SHA256：

`1ec1d1cb18f558e8d8168c84ce0d46c8250bbcb9283eba2e2eab824950f2d84b`

实际恢复到独立 Docker MySQL 5.7.44；21 张表按主键排序后的原始字节全部相同。随后仅在副本执行固定完整 schema 中提取的 `CREATE TABLE IF NOT EXISTS doudizhu_turn_deadlines`，验证 5.7.44 兼容，再对共享测试库执行同一 additive DDL。执行后、API 切换前，21 张原表哈希仍全部相同。未执行 DROP、ALTER、数据回填或共享库备份覆盖恢复；临时增量 SQL 未提交。

资源检查：部署前两个节点可用内存约 3041/3897 MiB，最终快照约 3098/3900 MiB；无 swap。`/data` 可用空间由约 8.2 GiB 至 7.6 GiB。未并行执行 Go 构建或重型测试。镜像仅导入实际调度的控制节点，不能据此宣称任意节点重调度已具备镜像条件。

程序性缺口：**没有保存部署前 Redis 活跃会话键的完整计数快照**，不补写或推算该证据。最终只读 SCAN 观察到 7 个 session 键；通过跨切换实际使用旧会话、新会话验证保留，但不能替代完整部署前后会话清单一致性证明。

## 4. 双队列、PVC 与配置

从仓库 overlay 渲染两个独立 ClusterIP Service、StatefulSet，各一副本；镜像替换为已验证摘要，无 UNBUILT 占位符。测试环境只有非默认 `awesome-zero-local`、无动态 provisioner，故增量创建两个专用 local Retain PV，绑定本 Namespace 的 PVC：

| 队列 Pod | PVC | PV | 本机独立日志目录 |
| --- | --- | --- | --- |
| beanstalkd-a-0 | journal-beanstalkd-a-0 | goal0063-beanstalk-a | `/data/awesome-zero-platform/goal0063/beanstalk-a` |
| beanstalkd-b-0 | journal-beanstalkd-b-0 | goal0063-beanstalk-b | `/data/awesome-zero-platform/goal0063/beanstalk-b` |

各 PVC 1 GiB、RWO、Retain；10001 用户、只读根文件系统，日志目录独立。local PV 的容量声明不等于文件系统配额。两队列均位于 `ser730417996259`，**这是独立进程/端点/PVC 验证，不是跨物理节点容灾证明**。没有改变其他 Namespace 已绑定的 PV。访问 HK 的既有 SSH 路径未通过主机密钥校验，未绕过校验或改变其信任配置。

实际启用配置：

```text
APP_DOUDIZHU_TURN_DQ_ENABLED=true
APP_DOUDIZHU_TURN_DQ_ENDPOINTS=beanstalkd-a.awesome-zero-game-test.svc:11300,beanstalkd-b.awesome-zero-game-test.svc:11300
APP_DOUDIZHU_TURN_DQ_TUBE=doudizhu-turn
```

对每个节点写入独立测试 tube 的 600 秒延迟消息，等待日志落盘，实际删除并重建该节点 Pod；同 PVC 恢复后 `peek` 内容一致，只删除本轮测试消息。A/B 均通过。B 的首次复连碰到 Service EndpointSlice 传播窗口，未再重启，等待后成功读取原消息。队列未公开到公网。

## 5. 真实业务与截止时间

共享 K8s 配置保持原值：**BiddingTimeout=300s、PlayingTimeout=300s**；没有用测试中的短超时冒充实际部署结果。MySQL 为回合/截止时间权威，Redis/Beanstalkd 只承担辅助通知与投递。

| 检查 | 结果与边界 |
| --- | --- |
| DQ 关闭状态的旧账号登录、AccountID、历史最终证据 | PASS；正常 HTTPS/CA 校验 |
| DQ 启用前安全门禁 | PASS；未完成牌局 0、双队列 Ready，未运行新旧两套调度器 |
| 三名新普通 QA 账号 | 实际 HTTP 注册/登录成功，随后完成真实 Vue 登录和 Cocos 房间流程 |
| 新房间、两名玩家加入、三方 ready、真实公平贡献及开局 | PASS；真实 drand 公共信标，非后端模拟 |
| 当前版本/seat/截止时间 | 私有投影与 MySQL 行一致 |
| 同模式 API 实际重启 | PASS；原 v1 截止时间未延长，恢复后仍为原回合 |
| 真实自动不叫 | PASS；截止 `16:36:07.856Z`，提交 `16:36:08.856Z`，延迟 1000ms，v1→v2、score=0，只有一条对应 action，未提前提交 |
| 人工叫 3 分进入 PLAYING | PASS；v3 已由私有投影/持久状态确认 |
| 自动最小合法单牌、自动 Pass | **未完成**；确认阻塞缺陷后停止等待，以业务取消结束 QA 牌局，不计 PASS |

本轮唯一共享环境 QA 牌局经真实 `hand.cancel` 收敛为 **ABORTED**，最终归档 1 条；共享库未完成牌局为 0。没有结束其他用户牌局、改写历史牌局或直接删除业务行。

## 6. G63-R8-N01 / P1：过期人工请求隔离实例

**实际故障，无需停 MySQL、Redis 或队列。** 使用独立恢复副本、同一固定产品 API 二进制，副本回合超时 20 秒。先在人工/超时竞态中观察到停滞，再做不注入任何新队列消息的最小复现。该副本与共享库、Redis、队列完全分离。

最小复现：

1. 三个真实认证账号开局至 BIDDING，读取其私有投影及 MySQL 当前 `live_version`、`seat`、`deadline_at`。
2. 由当前 seat 的合法账号发送普通 `hand.bid {score:0}`，使用新 requestId、非零 clientSeq、当前 expectedVersion；在真实 MySQL 截止时间后约 30ms 发出，赶在队列整秒延迟任务之前。
3. `POST /v1/doudizhu/commands` 返回 **500 / INTERNAL_ERROR**。
4. 约 5 秒后，私有牌面返回 **404**；域仍 BIDDING，MySQL 回合仍 v2，action 数仍 1。DQ repair 的 `next_attempt_at` 持续更新，但每次回调报 `gamecore: live instance not found`，回合不推进。

干净复现关键观测：

| 字段 | 值 |
| --- | --- |
| 截止时间 | `2026-11-21T16:38:09.061Z` / `1795279089061` |
| 请求发出 | `2026-11-21T16:38:09.091Z`，截止后 30ms |
| 操作 | hand.bid，expectedVersion=2，seat=1，score=0 |
| 返回 | HTTP500 / INTERNAL_ERROR |
| 后续读取 | HTTP404，BIDDING/v2，action 仍 1 条 |
| 数据正确性 | 未确认未提交操作；没有重复或错误持久化，但可用性失败 |

先前被隔离的同一副本牌局经过冷重启/重放恢复到 v2，然后再次被上述干净请求隔离。副本故障状态已再次一致性备份，原数据库卷保留；没有在原状态上写“修复”。

原因追踪（固定 SHA）：

- [`live_turn_deadline.go:230`](https://github.com/oublie6/awesome-zero-platform/blob/9474b9a4b2309cded0fc1dc1664650341d3d03e3/server/business/doudizhu/infrastructure/mysqlstore/live_turn_deadline.go#L230) 对晚到人工命令返回 `ErrTurnExpired`，此时尚未 INSERT action 或 COMMIT。
- [`live_journal.go:218`](https://github.com/oublie6/awesome-zero-platform/blob/9474b9a4b2309cded0fc1dc1664650341d3d03e3/server/business/doudizhu/infrastructure/mysqlstore/live_journal.go#L218) 将该确定性拒绝作为 persist 错误返回。
- [`gamecore/runtime.go:262`](https://github.com/oublie6/awesome-zero-platform/blob/9474b9a4b2309cded0fc1dc1664650341d3d03e3/server/business/gamecore/runtime.go#L262) 对任意 persist 错误设置 `entry.quarantined=true`。后续 View 返回 instance-not-found，DQ 不能取得当前私有牌面，持续重投也无法推进。

修复交给 ChatGPT：区分确定性未提交拒绝与真实 COMMIT 歧义，保留后者的 fail-closed/fencing；让正常过期人工拒绝不破坏当前可用实例；增加服务层真实 MySQL/DQ 的截止后请求及边界竞态回归。修复后须覆盖 bid/play/pass、版本/seat 竞争、重复票据、故障/ACK-loss 和全套适用部署验收，不能只验证 repository 返回 ErrTurnExpired。

未继续执行或未形成有效证据：完整合法重复票据矩阵、单队列不可用、丢消息修复、MySQL 回调故障、ACK 丢失精确重试及重启后重复投递。初始故障脚本的直接队列消息未采用 go-queue 时间戳信封，**不计为合法 DQ 重复/过期票据验证**；上述干净复现依赖真实普通 HTTP 命令，不依赖这些消息。共享 MySQL/Redis 从未用于故障注入。

## 7. Web/Cocos 实际回归及未完成项

实际部署的 Vue/Cocos 产物上，三账号分别登录，创建房间、加入、恢复认证/WSS、显示真实房间通过；无 pageerror。浏览器使用私有测试 CA 的正常信任库，Node/curl 使用 CA 文件，未设置 ignoreHTTPSErrors 或关闭 TLS 校验。

实际牌桌的本地倒计时从 46s 变为 44s，期间服务端版本未变化；刷新/重载页面并重新建立 WSS 后仍使用同一 MySQL 截止时间。该两项 PASS。

完整 UNKNOWN、出牌/Pass、三种尺寸布局、历史牌局的本轮视觉回归没有完成；保留 Round7 已验收历史证据，不将其改称本轮已部署业务 PASS。布局脚本误将仅 UNKNOWN 状态的 TableNotice 作为普通牌桌必需节点，断言失败不构成产品布局缺陷，也不计布局通过。

其他已纠正的私有验收脚本问题：旧 readiness 端口、开局尚待公共信标时过早读取 live private view、把重启后剩余时间误当作必须重新满 300 秒、将 live ACK 当作 durable ACK，以及误期望取消结果为 CANCELLED。取消实际返回 termination，域归档为 ABORTED；通过持久状态独立确认后没有重复取消。均未通过修改产品源码补救，未将原脚本失败隐藏为测试成功。

## 8. 历史数据、密钥及保护材料

原 Secret 全字段字节比较一致，原 ConfigMap data、四个原 Service spec 一致。MySQL/Redis 原 Pod UID 和重启计数均未变化。没有清空 Redis、撤销所有会话、轮换或删除密钥。

TLS：原测试 CA/leaf 保留，leaf 有效期 2026-11-15 至 2036-11-12。Reveal 当前 `g61-reveal-v2-20261115` 在有效期内（截至 2026-12-15），历史 retiring 的 `g57-reveal` 与原签名根、贡献保护根保留。没有把已过期历史 key 误设为当前 key。配置/签名私钥均未进入公共证据。

最终将原备份再次恢复到**本机独立副本中的 baseline 数据库**，按原主键选取共享库原有记录逐表比较：**20/21 张表原记录字节完全一致**；只有 `doudizhu_live_writers.owner_epoch/updated_at` 发生预期重启 fencing 元数据变化。原账号、密码凭据、房间、历史牌局、保护记录与最终归档均无意外更改或缺行。新增 QA 数据保留，未覆盖原数据。

回滚后，通过升级前签发的同一历史会话读取 `/auth/session` 和历史 `/evidence` 均 200，三个新 QA 会话仍可访问各自账号。该观察在访问令牌自然到期前完成，不承诺延长会话 TTL。

## 9. 实际回滚、最终健康与操作步骤

发现缺陷后停止扩大故障测试。先停止仅属本轮的等待脚本，通过认证业务命令取消共享 QA 牌局，独立确认 ABORTED/最终归档、全库未完成牌局数 0；随后 Recreate API/Web，恢复部署前不可变镜像，显式 DQ=false。**没有在活跃 DQ 牌局上切换旧调度器。** 没有用备份覆盖共享数据库。

| 最终 Pod | 镜像 manifest / 状态 |
| --- | --- |
| app-api-5ffd45d6b8-lwfxl | goal0062-api `sha256:9b8bf32b89c384c7f6760473c0f29b72999ed3b2ffff36876693ff2e6832906f`，1/1、0 restart |
| game-web-996b46c84-s54v7 | goal0062-web `sha256:ab52cee0608128afef4af6466c90bb7141b92d4b43e51bbc34ab165a537caab2`，1/1、0 restart |
| mysql-5cc48fd9b4-5hz6j | 原 `sha256:0b14b8e5c533fb43d07327fa8fe6c7393358002310d9f0d01976d387822a1bd3`，1/1；原 2 restart 未增加 |
| redis-67bdd89687-krvb4 | 原 `sha256:961f172bb2427dc11746904b17d853e59ee654b991343ea3955342b65157c907`，1/1；原 2 restart 未增加 |
| beanstalkd-a-0 / beanstalkd-b-0 | 本轮队列摘要，各 1/1、0 restart；各自 PVC Bound |

两个 K8s 节点 Ready。原四个 Deployment 各 1 副本；两个新增 StatefulSet 各 1 副本。API/Web 回滚后的健康、会话及历史业务读取验证通过。新增 deadline 表与其 1 条终态 QA cursor 保留，关联活跃牌局的 cursor 为 0。每个队列保留 2 条已归属终态 QA 牌局的 ready 投递；旧 API 不消费 DQ，未直接清除这些恢复材料。再次启用 DQ 前应按原门禁核对并通过修复后的终态 reconciliation 验证。

本轮开始时 29457 未监听，历史 forward unit 不存在。现已恢复 `https://127.0.0.1:29457` → `service/game-web:443`，对应 `goal0063-game-test-forward-v2.service`，显式指定原 kubeconfig，自动重连，回滚期间连接中断后恢复；最终 active。该 unit 为 systemd transient，**未验证主机重启后自动恢复**，不是公网入口或生产域名资格。

回滚命令与完整原 spec 私存于 `/root/goal0063-round8-9474b9a`（0700；敏感文件 0600）。以下是本次实际执行顺序；重复执行前必须先停止新开局入口并再次确认不存在活跃/在途牌局，不能把 SQL 计数当作并发 admission lock：

```bash
# 先通过业务接口收敛牌局；确认全库未完成牌局为 0 后才进行以下切换。
kubectl --context kubernetes-admin@network-learning -n awesome-zero-game-test \
  patch deployment app-api --type=json \
  --patch-file /root/goal0063-round8-9474b9a/app-api-rollback-patch.json
kubectl --context kubernetes-admin@network-learning -n awesome-zero-game-test \
  rollout status deployment/app-api --timeout=60s
kubectl --context kubernetes-admin@network-learning -n awesome-zero-game-test \
  patch deployment game-web --type=json \
  --patch-file /root/goal0063-round8-9474b9a/game-web-rollback-patch.json
kubectl --context kubernetes-admin@network-learning -n awesome-zero-game-test \
  rollout status deployment/game-web --timeout=60s
```

不得回放含活跃旧状态的数据库备份到共享库，不删除新增表/PVC/密钥。后续修复部署仍需单 writer、安全收敛、兼容性和恢复门禁。本轮在受控 QA 活动下实测回滚成功；没有演练多用户持续新开局时的流量隔离自动化。

本机隔离 API、浏览器及四个自建 Docker 依赖已停止，卷/故障备份保留；共享测试环境六个 Pod 继续运行。

## 10. 交付、复核与剩余风险

**FAIL / G63-R8-N01-P1 交给 ChatGPT 修复。** CI、备份、增量 DDL、队列持久化、部分真实业务及实际安全回滚通过，不等于完整 Goal0063 部署验收。自动最小出牌/Pass、完整合法票据/故障/ACK-loss、UNKNOWN 和布局回归等尚未完成，不可标 PASS。真机、微信平台、生产域名/发布资格、跨宿主机队列容灾均未获本轮资格。

最终对报告和脱敏证据做第二遍复核：区分实际部署与副本、原始脚本错误与产品复现、代码历史 PASS 与本轮业务未完成，核对镜像/Pod/源码 SHA、归档/截止时间、原始记录保留和回滚安全条件；扫描发布文件避免账号、密码、Token、私钥和原始 SQL。未修复生产代码、未扩大架构、未将 Goal0045/0046/0063 标为 completed，历史报告不改写。

提交范围仅本报告、脱敏结果及 `current.md` 允许的 Status 记录。数据库备份与临时升级 SQL、Secret、原始日志和截图不提交。
