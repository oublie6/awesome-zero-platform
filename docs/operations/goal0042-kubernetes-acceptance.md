# Goal 0042：最终部署与运行验收记录

代码、部署与真实运行验收已通过；归档提交的最终 GitHub Actions 结果在推送后确认。本记录区分本机执行、真实集群证据与 CI 结果。

## 版本与执行范围

- 本次同步前：`890abc8`；同步后的源版本：`698d7608cbecfb7c690a035b7675e5edc9ec0e3e`。
- Goal 0039～0041 已归档；该源版本 CI `41022280151`、Fair Doudizhu `41022280106` 均成功。
- Goal 定义检查点：`399703c`；CI `41043134748`、Fair Doudizhu `41043134806` 均成功。
- 可观测性修复检查点：`0a9ae1c`；该提交 CI `41044799534`、Fair Doudizhu `41044799514` 均成功；后续修复尚需新 CI。
- 本次用户明确授权主执行器执行源码检查、最小修复、测试、文档与安全条件满足后的部署；不是仅执行 Goal 0038 的补充交接。
- 部署前实际运行镜像为 `awesome-zero-platform/app-api:goal0038-gozero-first`；现已升级至 Goal 0042 镜像，前后证据见本报告末尾。

## 本次最小修复与源码验证

1. `foundation/database`：原生事务失败原来只记录事件，没有错误状态；panic 也没有失败状态。补上状态，保留 Begin、回调、Commit/Rollback、错误包装与 panic 传播顺序。
2. `platform/authn/adapter/redissession`：WATCH 失败补上错误状态；刷新轮换、冲突及过期语义不变。
3. `platform/realtime`：接受的消息在 envelope/topic/dispatch/handler 失败时使用固定协议错误码标记 Span；不记录私有 handler 错误或消息内容；响应不变。
4. `foundation/httpmiddleware`：外层访问日志原来拿不到 go-zero 内层路由 Context。使用 go-zero 注入响应的有效 W3C Trace Context 恢复日志关联，不新增请求 Span。
5. `foundation/observability`：HTTP 扩展 method 原来可以无限增加指标标签；归并为 `OTHER`，保留标准 method 和既有指标名称。
6. `server/go.mod`：tidy 将原已选中的 OpenTelemetry v1.40.0 根模块调整为直接依赖，没有升级版本。

相关五包 focused tests 和完整串行 unit suite 已通过；完整 suite 包含现有架构边界和 API 路由清单测试。该检查点的本机完整 Race 检查也已通过；后续指标/所有权/授权事务补丁的五包 focused Race（含严格边界检查）已通过，清理时序断言、后续业务 READ COMMITTED 事务与 HTTP Context 边界变更的 focused Race 及完整 unit 重跑也通过。

REST 继续使用 go-zero `rest.Server`/router/route middleware；外层 CORS、RequestID、脱敏日志、流式 body limit、JSON recovery 保留已记录的兼容语义。JWT 签名与可撤销 Redis Session 仍由既有适配层组合，不能用不可撤销 JWT 中间件直接替代。47 条静态路由清单不等于业务 DTO/handler 全量生成；健康端点仍是当前 goctl 生成入口。

## 补充架构与指标修复（源码与集成检查已完成）

- 原 `casbinmysql/active_superadmin.go` 直接锁读身份表；现改为 identity 的公开 `AnyActiveAccountForUpdate` 契约。SQL、状态解释与游标关闭归所有者；同一事务、排序 ID、版本锁优先及最后活跃超管保护保持不变。新增测试覆盖 disabled/missing、查询/扫描错误、active 行之后的游标错误和关闭时机。
- Admin 实际独占维护角色/资源元数据，authz 维护 Casbin 可执行策略及版本；修正测试中仅按 `authorization_*` 前缀推断全部归 authz 的规则，明确目录表归 Admin，保留实际写入所有者、表名及 DTO。严格架构测试现覆盖 platform 模块间的 SQL 所有权，不能通过跨模块直接查表。
- 提取最小 `database.TraceTransaction` 边界，覆盖原生 identity 和授权事务，保持原 Begin/Commit/Rollback 和 post-commit reload/publish 顺序与 partial-success 返回值；无 Context 的 Casbin `SavePolicy` 使用明确独立的 root Span。原生事务语句不逐条创建 SQL Span，Context-less Casbin 回调不能宣称关联 HTTP。
- Redis 操作/池指标已通过 go-zero 的公开 `core/prometheus.Enable` 启用并选择性合并框架既有指标。没有启动第二个 listener；精确 allowlist 避免重复 Go/进程 collector。新 handler 测试含重复 runtime collector/无关指标，已通过 Race；还需真实 Prometheus 验证。
- 新的 account-state 契约明确依赖进程内共享事务。未来拆分必须重新设计该原子安全不变量；未将显式 SQL 事务契约包装成 RPC 就绪声明。详见 [架构说明](../architecture/go-zero-http-and-module-integration.md)。

## 部署前实测

- context：`kubernetes-admin@network-learning`；两节点 `ser730417996259`、`hk-worker` 均 Ready、amd64；Memory/Disk/PID Pressure 均 False。
- API、MySQL、Redis、Elasticsearch、Prometheus、Grafana、Jaeger、Kibana 均 Ready；四个 PVC 均 Bound。API 为一副本、Recreate、Never；原 server overlay 与集群 `kubectl diff` 为空。
- `/data` 40 GiB，初始约 26 GiB 可用。未修改或清理数据卷。
- 集群没有 Metrics API，`kubectl top` 不可用；两节点 kubelet `stats/summary` 可读取，原始数据已保存。初始工作集约：API Pod 81 MB、MySQL 247 MB、Redis 10 MB、Elasticsearch 1.12 GB、Prometheus 37 MB；另节点 Grafana/Kibana/Jaeger 合计约 778 MB。这些是时间点证据，不是长期容量保证。
- 本机编译期间两秒 `/proc/stat` 样本 steal 约 35%，不能推断稳态性能。今日本机 kernel journal 未匹配 soft lockup/OOM/I/O error；未据此宣称另一节点内核检查通过。
- 升级前健康、既有验收管理员登录、Session、`/admin/me` 及权限基线通过；仅为验收创建一个专用 Session，保留用于升级兼容检查。

## 备份、回滚与安全条件

证据保存在 `/root/.local/state/awesome-zero-platform/goal0042`（目录 0700，含敏感快照的文件 0600，均不入 Git）：

- 全部 15 个应用表均 InnoDB；生成 `--single-transaction` 一致性 MySQL dump。
- Redis 通过复制协议导出 RDB；未 FLUSH、重建 AOF 或改写业务 key。
- 旧 API 镜像已从 containerd 导出；原 Deployment 与只更新镜像的回滚补丁已保存。
- 备份清单记录文件字节数与 SHA256。
- **隔离恢复已通过：** MySQL 8.4.6 恢复全部 15 表，逐表行数与备份前一致；Redis 7.4.5 成功加载 RDB，恢复 2 个 key。演练使用 `network=none`、临时存储，没有挂载线上卷；两容器均已删除。没有执行线上恢复、生产故障注入或应用回滚。部署仍需源码检查、最新 CI 和最后安全核验。

## 验收矩阵

| 门禁 | 结果 |
| --- | --- |
| main 同步、Goal 0039～0041 归档、同步源 CI | 通过 |
| 最小修复 focused/unit、gofmt、tidy | 通过 |
| Race | 完整 platform/foundation/architecture Race 与后续事务/HTTP/Redis focused Race 均通过 |
| API DSL 校验、生成一致性、前端构建 | 通过；Admin build、9 项客户端测试及 TS→Go 互操作通过 |
| 隔离集成、后端不可用/恢复、慢消费者检查 | MySQL 8.4.6/Redis 7.4.5 集成与隔离暂停恢复通过；慢消费者有 deterministic/Race 覆盖 |
| 两套 Kubernetes 渲染校验 | 通过：server 28 resources；默认 observability 23 resources；默认配置未应用 |
| 原集群清单、PVC、部署漂移 | 已实测；仅 API image 变更，Secret/PVC/运行配置不变 |
| 备份可恢复性、部署前最后安全检查 | 全 15 表与 2 Redis keys 隔离恢复匹配；备份 hash、镜像回滚补丁、diff/会话/PVC/Secret 安全门禁通过 |
| 新镜像构建/import/源码对应、Ready | 通过；3fd4cb6 source label/config ID 与实际 Pod imageID 对应 |
| 新版本 REST/WS/缓存/事务/WATCH 真实验收 | 通过；见下方真实请求与 trace 证据 |
| Prometheus/Grafana 真实采集/面板 | 通过；指标查询与浏览器六面板截图 |
| Jaeger HTTP/SQL/transaction/Redis/WATCH/WebSocket message spans | 通过；实测关系、时间和错误状态 |
| Filebeat/ES/Kibana 日志与 RequestID/TraceID | 通过；ES 单条相关记录及 Kibana Discover 浏览器结果 |
| 有界资源观察、泄漏、保留/磁盘水位 | 部署后在线 4 分钟及静默约 2.7 分钟观察通过；保留策略和水位复查通过（非长期容量证明） |
| 平台内数据所有权问题 | 严格边界与实际版本并发集成通过；未来跨服务原子性限制明确记录 |
| 最终自审、归档、push、最终 CI | 自审完成；归档 push 与其 CI 在进行中 |

## 测试失败与自审证据

- 系统 Go 1.18.1 无法解析项目 `go 1.25.8`；已在 `/opt/goal0042-toolchain` 安装并按官方 SHA256 验证匹配工具链。
- 初次调用 K8s validator 缺少渲染文件参数而失败；按 CI 命令传入正确文件后两套均通过。没有更改 validator。
- 首轮测试期间新增 import，导致已完成的依赖扫描失效；终止该轮并重新运行，fresh focused/full unit 全通过。没有跳过断言或修改测试标准。
- Node 官方压缩包 SHA256 已验证；Python 安全解包拒绝 npm symlink，使用 tar 完成解包。版本 22.23.4；前端构建通过，既有 bundle >500 kB 提示未改。
- 已对可观测性补丁做一次独立阶段的**自审**：事务 defer 顺序、panic/error 身份、Context/父 Span、敏感字段、响应兼容、标准指标标签和新增测试误报风险；修正新 WS 测试的无界接收。不是独立第三方审查。后续源码变更仍需重新自审。

以上为过程阶段记录；最终运行验收结果与剩余环境限制以本报告末尾“部署后真实验收”为准。

## 后续源码检查点补充证据

- Goctl 1.10.1 与 CI 一致：API `format ok`、健康代码再生成后 diff 为空；gofmt/tidy 幂等通过。完整 unit、platform/foundation/architecture Race 和后续 focused Race 通过。
- 全量原生事务入口审查补齐业务存储的追踪边界；既有 scripted driver 强制 READ COMMITTED，新增测试验证回调子 Context、错误/panic/重试分类以及 Span.End 前提交/回滚已完成。未在服务器启用该业务。
- 日志测试新增“请求先有一个有效 Span，但 incoming W3C context 指向另一 Trace”的边界；优先使用实际路由响应 Context，保留缺失/无效传播时原 Context。该 focused Race 通过。
- Admin Web 构建成功，存在既有 minified chunk >500 kB 提示；安全信封客户端 9 项测试及 TS→Go 互操作通过。未修改前端源码、依赖版本或警告阈值。
- 在仅 loopback、临时存储的隔离 MySQL 8.4.6 / Redis 7.4.5 上执行原 `make integration-test`，全部通过（117 秒）。这些版本匹配目标集群；既有 5.7/8.8 验证由新提交 CI 另行确认。集成容器已删除。
- 六个新增 observability regression tests 以 Go overlay 对修复前源码执行，均按预期失败，且无编译失败；严格 SQL 所有权测试拒绝临时跨模块读表探针，探针随后移除。反向验证不是新的失败项。
- 真实新镜像 Jaeger、Prometheus、Grafana 与 ELK 验收已在部署后完成；阶段检查点上的“待执行”仅记录当时状态，最终结果见文末。

## 源码检查点 45707c4 与隐私自审

- `45707c4fbe40017f90e61d41a5195aee02959e90`：CI `41050979330`、Fair Doudizhu `41050979382` 均成功。静态二进制构建成功，记录的 Go/go-zero 版本为 1.25.8/1.10.2，VCS dirty=false。
- 该二进制在隔离 fixture 的 MySQL/Redis 暂停约 14.17/9.46 秒后恢复；解除暂停后的健康/认证成功检查约 0.03/0.11 秒。不可用时 readiness 失败、liveness 正常、认证请求 fail closed。没有对生产数据库注入故障；所有 fixture 容器已移除，日志敏感信息扫描通过。
- 资源短采样 goroutine 33～35，但堆内存从 66.46 增至 71.49 MiB。该样本仅说明短时有界，**不能证明内存稳定或无泄漏**；后续延长观察并记录 GC 次数。
- 后续隐私自审发现新事务边界会将任意 callback error 文本复制到异常事件。现使用固定失败文本，保留调用方原错误及 panic/回滚语义。完整 unit、三个事务消费者 Race 通过；旧实现的 overlay 被新增私有错误回归断言拒绝。此改动仍需新检查点 CI 和最新镜像。

## 待部署镜像与最后预检

- 最终代码 `3fd4cb6fcb4983d3d548804f3964dc541c90cb54`：CI `41053084557`、Fair Doudizhu `41053084529` 均成功。
- 延长到三分钟以上的隔离资源观察看到 GC 次数 3→7；堆从最高约 75.10 MiB 回收到约 3.15 MiB，最后 2.50 MiB，goroutine 约 33～34。先前短样本不能证明平台，现在已实际观察到回收；仍不是长期容量/无泄漏保证。隔离故障恢复和日志扫描通过，临时容器均移除。
- 镜像 `awesome-zero-platform/app-api:goal0042-3fd4cb6fcb49`，config ID `sha256:81e0fd668b276c59d8790944ecac4358f5906ff45f45eec858a134d126736dd1`。使用仓库 Dockerfile、精确 Git archive 上下文和 1.5 GiB/2 CPU builder 限额构建；没有打包工作区未跟踪凭据或客户端 node_modules。
- source revision label、linux/amd64、nonroot、entrypoint、内置 Go 1.25.8/go-zero 1.10.2 及原 production 配置模板均核对。镜像二进制 SHA256：`ccf2fca67b191e6354156581def60b0b32620102869fe8aeeacce1435cdd68d6`。
- 已导入目标节点 containerd，crictl image ID 一致；旧镜像与备份仍在。server overlay 仅改该 image tag，28 资源验证通过；实际 diff 仅 API 镜像及 generation。窄镜像补丁 server-side dry-run 没有其他 spec 变化。
- 预检八个组件 Ready、PVC Bound、两节点无 pressure、/data 约 24 GiB 空闲、WS 活跃数为 0、业务仍 disabled；单副本/Recreate/10 秒配置与所有 Secret/卷引用保持原状。主数据备份校验与隔离恢复已复核。
- 已在 4b36269 manifest 检查点 CI 通过后按预审补丁完成 API image-only rollout。真实 REST/WS/Jaeger/Prometheus/Grafana/ELK 与部署后资源检查现已通过，详见最终验收。


## 部署后真实验收（2026-10-30）

### 镜像、状态与保全

- 部署前 Git source：`3fd4cb6fcb4983d3d548804f3964dc541c90cb54`；同步仓库头部 `4b36269378bf0b8e0d17ea32c693e99f8068f059` 仅包括同一镜像 tag 与文档。部署镜像标签 `awesome-zero-platform/app-api:goal0042-3fd4cb6fcb49`、containerd config ID `sha256:81e0fd668b276c59d8790944ecac4358f5906ff45f45eec858a134d126736dd1`，内含二进制 SHA256 `ccf2fca67b191e6354156581def60b0b32620102869fe8aeeacce1435cdd68d6`。
- 部署后 `app-api-669ccbbc56-dqwr5` 在 `ser730417996259` 上 2/2 Ready、0 restart，Pod image ID 与上述 config ID 一致。部署镜像由 Dockerfile 对准确 Git archive 构建、导入所选节点 containerd；`crictl` 显示相同 image ID，旧镜像仍可回滚。
- 实际前后 Pod template 对比只有 image 字段不同；API 维持 1 replica、Recreate、CPU 100m/1、内存 128/512Mi、10 秒挂载生产配置、原 Secrets/卷引用。Secret UID/resourceVersion/data SHA256 均未变，四个 PVC Bound 且 UID/spec 保持；未触碰 MySQL/Redis 卷、未重跑 bootstrap/schema、未启用 Doudizhu。
- 部署后 API、MySQL 8.4.6、Redis 7.4.5、Elasticsearch 9.1.0、Prometheus 3.5.0、Grafana 12.1.0、Jaeger 2.20.0、Kibana 9.1.0 均 Ready；两个节点 Ready、无 Memory/Disk/PIDPressure。`/data` 40 GiB，约 24 GiB 可用；节点 kubelet stats 可用，Metrics API 不可用。只有 mysql/redis 有 35 小时前的一次历史 restart，验收期间 restart 数无增加。
- `OBS_KUSTOMIZE_OVERLAY=deploy/kubernetes-overlays/observability-server make observability-status` 与 `make observability-smoke` 均通过。后者实际检查 Pod 级 Prometheus target、Grafana datasource/dashboard、Jaeger 服务、日志进入 ES 及 Kibana Data View。

### REST、Session、授权、数据访问和 WebSocket

- 升级前取得的 opaque refresh token 在升级后成功轮换，验证既有 Session 兼容；健康与 readiness、有效登录/登出、管理员 API、普通账号认证及未授权拒绝通过。
- 对匿名会话、匿名 Admin、空/无效 JSON、错误密码、缺失路由、普通账号调用 Admin、重复用户名事务回滚、refresh token 重放执行真实请求，分别观察预期 400/401/403/404/409。
- 真实创建、登录、读取、资料更新和后续 cache invalidation 成功；管理员资料更新后，冷读 trace 含 2 个 SQL spans，命中读 trace 只含 1 个 SQL span。Redis Session refresh 使用 WATCH 成功轮换，旧 refresh 重放被拒绝。
- 专用 WebSocket 连接通过 Bearer 认证；收 welcome、ping/pong、topic subscribe/unsubscribe ack、unknown type、invalid envelope、invalid JSON 的固定错误响应。约 24.9 秒收服务器 ping 并回 pong；关闭、重新连接和断开后 active gauge 回到零。`/ws` 匿名握手被拒绝。
- 两个验收专用普通账号都已禁用；管理员和专用 viewer Session 均已撤销。未禁用既有管理员或业务账号。

### Jaeger 实际 trace

- 最新镜像 REST Trace IDs 真实存在 Jaeger：login、Admin、创建账户、重复创建/回滚、cache cold/hit/update/invalidation、refresh/WATCH rotate。检查每条有且仅有预期 HTTP route parent，所有子 span 关联同 Trace 的实际 parent，开始/结束时间处于 parent 时间范围，耗时非负。
- SQL 查询和 Redis 命令 span 实际出现；事务成功与重复冲突回滚 span 存在，失败事务带 ERROR 状态。WATCH rotation 及 Redis 操作实际有 span。更新资料走普通 SQL（不伪称事务），因此验收脚本按实际服务路径要求 SQL span。
- WebSocket 的 6 条 `realtime.message` span 为独立 root trace（遵守消息生命周期，不伪造握手父 Span），context 传入 handler；无效 JSON/envelope 等真实协议失败 span 有受限 ERROR 状态。没有将敏感 payload/原始错误注入 span。
- Jaeger 使用内存存储，Pod 重启会丢历史 trace；本次 trace 实际验证通过，但没有持久化保证。

### Prometheus、Grafana、Filebeat、Elasticsearch、Kibana

- Prometheus 查询有实际 series/value：HTTP request 总速率、状态 counters（200/400/401/403/404/101/201/409）、HTTP 延迟直方图 p95、WebSocket 消息及 active gauge、go_goroutines、process CPU seconds/RSS、go-zero managed Redis requests/pool、native Redis pool gauges。最近探测数据示例：请求累计 2604；400=10、401=16、403=3；HTTP histogram count 2647；WS messages=12、active=0；Redis managed pool 9，native pool 1。计数是集群共享累计数，不归因成测试独占。
- 本次不制造生产 5xx。5xx PromQL 当前无匹配 series；Grafana 的 5xx panel 在实际浏览器中绘制在 0 轴，无 5xx 数据；因此此项只证明面板/采集正常，不声称发生过 5xx 故障恢复。
- 真实 Chromium 浏览器（临时、非特权、结束即删除的 Pod）打开现有 Grafana Dashboard，六个 HTTP/WS/延迟/内存面板均显示实际时间序列；截图和页面文本在私有验收目录。Kibana Discover 按测试 `requestId` 与 `traceId` 找到 1 条 `platform/http.access` 记录，并由实际浏览器渲染为 1 document。
- Filebeat→Elasticsearch 查询以相同 request ID 与 trace ID 命中 1 条日志；应用日志、trace 对测试已知密码/Token/Argon2 内容的扫描通过。临时 browser Pod、CDP port-forward 与所有验收 port-forward 均已终止。

### 后端恢复、资源、 retention 和主机风险

- 相同源码在隔离 Docker fixture 实测 MySQL/Redis 短暂不可用时 readiness 失败、liveness 保持、认证 fail closed，恢复后约 0.03/0.11 秒重新成功；这是 fixture，不是生产故障注入。MySQL/Redis 生产实例在本次部署期间未中断。
- 应用既有 `TestPerAccountLimitAndSlowConsumerBound` 的短队列满载/拒绝测试与 Race 检查通过；真实客户端关闭/重连后 gauge 为 0。未在生产开放真实无限慢读者以免积压风险。
- rollout 后四分钟混合请求观察覆盖 GC、heap、goroutine、RSS、WS 和两类 Redis 池；随后约 2.7 分钟安静采样，goroutine 35–36、heap ~3.5–5.6 MiB、RSS ~32.5–32.8 MiB、GC 332→338、WS 连接始终 0，native Redis pool 始终 1；之前混合样本 managed pool 为 9。只证明短时无异常增长，不是 soak/capacity SLA。
- 部署后只读重查 Elasticsearch green/3-day ILM/12-8-5 GB watermarks、Prometheus 3d/512 MiB、MySQL binlog 259200s/100 MiB、Redis AOF/noeviction/128 MiB，均通过。集群当前没有 off-node backup destination，快照存于本执行环境私有 0700 目录；独立复制/灾备仍是后续运维任务。
- 共享 CPU：此前编译期间本执行环境两秒 steal ~35%，当前空闲时两秒 sample 为 0%；这些短样本不证明稳定宿主机容量。通过 kubelet 节点日志代理只读检查两节点 `kern.log` 及轮转文件：控制节点 `ser730417996259` 有 317 条历史 soft-lockup 日志，分布 Oct 20（57）、Oct 21（71）、Oct 22（32）、Oct 29（157）；Oct 29 02:50 多 CPU 记录 stuck 3702–3709 秒，03:29 又有多个 22–226 秒记录，涉及 etcd/kubelet/kube-apiserver/Cilium/containerd 等。hk-worker 当前及轮转 kernel logs 未发现 soft/hard lockup、OOM 或 I/O error。控制节点当前 Ready、heartbeat 更新，Kubernetes events/pressure 无当前告警；node stats startTime 为 Oct 29 04:48，但现有证据不能证明恢复原因，也不能将日志归因于本次代码。此为严重既有宿主机稳定性风险，原因与运维修复未验证；没有重启/改主机配置。应由节点/虚拟化管理员查明长时间 CPU/调度停顿，并用宿主机监控验证恢复。该残余风险必须在部署完成结论中保留。

### 本次测试/CI、修复和失败记录

- 通过：API DSL/goctl format 与生成无 diff、gofmt、idempotent `go mod tidy`、完整 unit、platform/foundation/architecture Race、native transaction/HTTP context focused Race、模块 SQL ownership、frontend typecheck/build、客户端 9/9、TS→Go envelope interop、实际版本 MySQL 8.4.6/Redis 7.4.5 `make integration-test`、两套 Kubernetes render 28/23、隔离数据库 restore/recovery、镜像 build/import、部署后 observability smoke 和现场 REST/WS/telemetry/UI。
- 代码自审第三次检查还发现并修复 native Doudizhu transaction bypass、早期 request Context 被 go-zero response trace 覆盖时的日志关联、事务失败 Span 可能记录调用方私密错误文本。新增测试按预期拒绝旧实现；自审覆盖事务顺序/隔离级别/rollback/panic/error identity、锁和 post-commit 路径、查询游标关闭、Context/parent、隐私、标签边界、资源清理及模块所有权。属于同一执行者 self-review，不是独立审核。
- 新源码 commit `3fd4cb6fcb4983d3d548804f3964dc541c90cb54` GitHub CI 41053084557 / Fair Doudizhu 41053084529 success；部署 manifest commit `4b36269378bf0b8e0d17ea32c693e99f8068f059` CI 41056434697 / Fair Doudizhu 41056434701 / K8s manifests 41056434663 均 success。归档 commit 的最终 CI 待其 push 后确认。
- 曾有探针遗漏 API DTO 必填字段、把完整 DTO PATCH 当成部分 PATCH、以及把资料更新错误要求为事务；均为验收脚本假设错误。按已声明契约修正探针，并保留失败记录；没有更改 production validation 或降低断言。一次 Prometheus 补充读取复用了旧 port-forward 日志误判启动，按实际本地监听重试后通过。无测试 assertion 因失败被跳过。

### 精确改动文件

- 部署：`deploy/kubernetes-overlays/observability-server/placement.yaml`（仅固定到不可变 goal 镜像 tag）。
- 架构与操作报告：`docs/architecture/go-zero-http-and-module-integration.md`；`docs/operations/goal0042-kubernetes-acceptance.md`。
- 架构边界：`server/architecture/boundaries_test.go`。
- Doudizhu 持久化事务 Trace：`server/business/doudizhu/infrastructure/mysqlstore/store.go`、`transaction_trace_test.go`。
- Foundation 数据库事务与 Trace：`server/foundation/database/database.go`、`transaction_trace_test.go`。
- Foundation HTTP trace 日志：`server/foundation/httpmiddleware/accesslog.go`、`accesslog_observability_test.go`。
- Foundation Prometheus/Redis 指标：`server/foundation/observability/metrics.go`、`metrics_test.go`、`server/go.mod`。
- Authn Redis WATCH Trace：`server/platform/authn/adapter/redissession/store.go`、`tracing_test.go`。
- Authz 模块所有权与事务 Trace：`server/platform/authz/adapter/casbinmysql/active_superadmin.go`、`cluster.go`、`engine.go`、`transaction_trace_test.go`。
- Identity 公共锁查询契约：`server/platform/identity/account_state_guard.go`、`account_state_guard_test.go`。
- Realtime message Trace：`server/platform/realtime/hub.go`、`tracing.go`、`tracing_test.go`。
- 本次新增恢复、runtime、backup、browser、trace、prom、resource 等原始证据位于 repo 外私有目录，不提交凭据、快照或日志。
