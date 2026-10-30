# Goal 0042：最终部署与运行验收记录（进行中）

**尚未通过最终验收。** 本记录区分源码检查、自动化测试、既有 CI 和本次真实集群证据。未完成项不得解释为通过。

## 版本与执行范围

- 本次同步前：`890abc8`；同步后的源版本：`698d7608cbecfb7c690a035b7675e5edc9ec0e3e`。
- Goal 0039～0041 已归档；该源版本 CI `41022280151`、Fair Doudizhu `41022280106` 均成功。
- Goal 定义检查点：`399703c`；CI `41043134748`、Fair Doudizhu `41043134806` 均成功。
- 可观测性修复检查点：`0a9ae1c`；该提交 CI `41044799534`、Fair Doudizhu `41044799514` 均成功；后续修复尚需新 CI。
- 本次用户明确授权主执行器执行源码检查、最小修复、测试、文档与安全条件满足后的部署；不是仅执行 Goal 0038 的补充交接。
- 目前没有执行 API 升级。运行镜像仍为 `awesome-zero-platform/app-api:goal0038-gozero-first`，旧版本源码对应关系见 [Goal 0038 部署记录](goal0038-kubernetes-acceptance.md)。

## 本次最小修复与源码验证

1. `foundation/database`：原生事务失败原来只记录事件，没有错误状态；panic 也没有失败状态。补上状态，保留 Begin、回调、Commit/Rollback、错误包装与 panic 传播顺序。
2. `platform/authn/adapter/redissession`：WATCH 失败补上错误状态；刷新轮换、冲突及过期语义不变。
3. `platform/realtime`：接受的消息在 envelope/topic/dispatch/handler 失败时使用固定协议错误码标记 Span；不记录私有 handler 错误或消息内容；响应不变。
4. `foundation/httpmiddleware`：外层访问日志原来拿不到 go-zero 内层路由 Context。使用 go-zero 注入响应的有效 W3C Trace Context 恢复日志关联，不新增请求 Span。
5. `foundation/observability`：HTTP 扩展 method 原来可以无限增加指标标签；归并为 `OTHER`，保留标准 method 和既有指标名称。
6. `server/go.mod`：tidy 将原已选中的 OpenTelemetry v1.40.0 根模块调整为直接依赖，没有升级版本。

相关五包 focused tests 和完整串行 unit suite 已通过；完整 suite 包含现有架构边界和 API 路由清单测试。该检查点的本机完整 Race 检查也已通过；后续指标/所有权/授权事务补丁的五包 focused Race（含严格边界检查）已通过，清理时序断言、后续业务 READ COMMITTED 事务与 HTTP Context 边界变更的 focused Race 及完整 unit 重跑也通过。

REST 继续使用 go-zero `rest.Server`/router/route middleware；外层 CORS、RequestID、脱敏日志、流式 body limit、JSON recovery 保留已记录的兼容语义。JWT 签名与可撤销 Redis Session 仍由既有适配层组合，不能用不可撤销 JWT 中间件直接替代。47 条静态路由清单不等于业务 DTO/handler 全量生成；健康端点仍是当前 goctl 生成入口。

## 补充架构与指标修复（集成验收待完成）

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

| 门禁 | 当前结果 |
| --- | --- |
| main 同步、Goal 0039～0041 归档、同步源 CI | 通过 |
| 最小修复 focused/unit、gofmt、tidy | 通过 |
| Race | 0a9ae1c 完整通过；后续修复 focused 通过，最终重跑待执行 |
| API DSL 校验、生成一致性、前端构建 | 通过；Admin build、9 项客户端测试及 TS→Go 互操作通过 |
| 隔离集成、后端不可用/恢复、慢消费者检查 | 8.4.6/7.4.5 完整集成通过；受控不可用/恢复待执行，慢消费者已有 deterministic/race coverage |
| 两套 Kubernetes 渲染校验 | 通过：server 28 resources；默认 observability 23 resources；默认配置未应用 |
| 原集群清单、PVC、部署漂移 | 已实测；无漂移 |
| 备份可恢复性、部署前最后安全检查 | 隔离恢复通过；最后安全检查待执行 |
| 新镜像构建/import/源码对应、Ready | 未部署 |
| 新版本 REST/WS/缓存/事务/WATCH 真实验收 | 待执行 |
| Prometheus/Grafana 真实采集/面板 | 待执行 |
| Jaeger 六类真实 Span、关系/耗时/错误 | 待执行 |
| Filebeat/ES/Kibana 日志与 RequestID/TraceID | 待执行 |
| 稳态资源、泄漏观察、保留/磁盘水位 | 待执行 |
| 平台内数据所有权问题 | 严格边界与实际版本并发集成通过；未来跨服务原子性限制明确记录 |
| 最终自审、归档、push、最终 CI | 待完成 |

## 测试失败与自审证据

- 系统 Go 1.18.1 无法解析项目 `go 1.25.8`；已在 `/opt/goal0042-toolchain` 安装并按官方 SHA256 验证匹配工具链。
- 初次调用 K8s validator 缺少渲染文件参数而失败；按 CI 命令传入正确文件后两套均通过。没有更改 validator。
- 首轮测试期间新增 import，导致已完成的依赖扫描失效；终止该轮并重新运行，fresh focused/full unit 全通过。没有跳过断言或修改测试标准。
- Node 官方压缩包 SHA256 已验证；Python 安全解包拒绝 npm symlink，使用 tar 完成解包。版本 22.23.4；前端构建尚未运行。
- 已对可观测性补丁做一次独立阶段的**自审**：事务 defer 顺序、panic/error 身份、Context/父 Span、敏感字段、响应兼容、标准指标标签和新增测试误报风险；修正新 WS 测试的无界接收。不是独立第三方审查。后续源码变更仍需重新自审。

尚无最终通过结论；完整原始证据与尚未执行项将随实际结果更新。

## 后续源码检查点补充证据

- Goctl 1.10.1 与 CI 一致：API `format ok`、健康代码再生成后 diff 为空；gofmt/tidy 幂等通过。完整 unit、platform/foundation/architecture Race 和后续 focused Race 通过。
- 全量原生事务入口审查补齐业务存储的追踪边界；既有 scripted driver 强制 READ COMMITTED，新增测试验证回调子 Context、错误/panic/重试分类以及 Span.End 前提交/回滚已完成。未在服务器启用该业务。
- 日志测试新增“请求先有一个有效 Span，但 incoming W3C context 指向另一 Trace”的边界；优先使用实际路由响应 Context，保留缺失/无效传播时原 Context。该 focused Race 通过。
- Admin Web 构建成功，存在既有 minified chunk >500 kB 提示；安全信封客户端 9 项测试及 TS→Go 互操作通过。未修改前端源码、依赖版本或警告阈值。
- 在仅 loopback、临时存储的隔离 MySQL 8.4.6 / Redis 7.4.5 上执行原 `make integration-test`，全部通过（117 秒）。这些版本匹配目标集群；既有 5.7/8.8 验证由新提交 CI 另行确认。集成容器已删除。
- 六个新增 observability regression tests 以 Go overlay 对修复前源码执行，均按预期失败，且无编译失败；严格 SQL 所有权测试拒绝临时跨模块读表探针，探针随后移除。反向验证不是新的失败项。
- 最终运行门禁仍待执行；没有将源码/隔离结果替代真实新镜像的 Jaeger、Prometheus、Grafana 或 ELK 证据。

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
- **尚未滚动部署。** 待 manifest 检查点 CI 后，仅应用审查过的 API 镜像补丁。真实 REST/WS/Jaeger/Prometheus/Grafana/ELK 与部署后资源检查仍待执行。
