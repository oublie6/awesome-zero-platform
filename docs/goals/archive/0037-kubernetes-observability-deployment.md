# Goal 0037: Kubernetes 可观测性实际部署与验收

## Status

- State: completed
- Started: 2026-10-28
- Completed: 2026-10-29
- Blockers: 无；运行验收、最终功能CI与自审通过。

## Goal

在用户确认的云服务器 Kubernetes 环境实际部署日志、指标、追踪系统，并验证真实数据可查询及业务可用性。属于可复用技术基础设施的部署与验收，不改变平台架构或业务语义。用户明确授权初始化此目标及必要的最小部署配置修复。

## References

- `AGENTS.md`
- `docs/operations/logging-observability.md`
- `deploy/kubernetes/`
- `deploy/kubernetes-overlays/observability/`
- `scripts/k8s-observability.sh`
- `scripts/k8s-observability-smoke.sh`
- `deploy/kubernetes-overlays/observability-server/README.md`
- `deploy/kubernetes-overlays/observability-server/`

## Deliverables

1. 目标服务器、集群、容量、已有业务及安全检查证据。
2. Kubernetes Filebeat/Elasticsearch/Kibana、Prometheus/Grafana、OTLP/Jaeger 实际部署。
3. 三条真实数据链路、所有 Ready app-api Pod 指标发现、Dashboard、Trace 和业务回归验收。
4. 实际资源及镜像清单、持久化与保留策略、安全访问、运维与回退报告。

## Constraints

- 遵守 main 工作流；仅作此次部署必要的最小修改，不改业务规则、协议或数据库 Schema。
- 不删除/重建现有业务数据库，不清空 Redis/MySQL/PVC，不删除 Namespace，不作未审查的批量删除，不修改其他项目资源。
- 不提交凭据，不公开观测端口，不使用 Compose，不以静态检查替代运行验收。
- 保留实际 app-api 镜像和副本数；不擅自扩容内存牌局实例。影响业务前提供影响分析和回退方案。
- 初始授权仅允许在专用测试集群补齐基础设施；发现共享集群后已询问使用范围，用户回复“可以部署”，据此在当前集群创建本项目专属基础设施，不复用未知卷。
- 不随意安装/替换集群，不以特权容器更改宿主配置，不强行在低资源环境启动完整 ELK。

## Acceptance Criteria

- 确认目标 context、现有应用镜像/副本/活跃业务、节点实际容量、存储、依赖、Secret、镜像拉取和内核配置。
- 渲染并审查目标 Namespace、资源限制、镜像、PVC、Secret 和变更 diff 后才部署。
- ES Ready、PVC Bound、健康与 ILM 正常；Kibana、Filebeat、Prometheus、Grafana、Jaeger、app-api 实际健康。
- 在目标集群执行 `make observability-smoke`；补充验证其未覆盖的验收项。
- 唯一 requestId 请求日志可由 ES 与 Kibana 查询；每个 Ready app-api Pod 指标 UP，HTTP 计数/延迟及 WS 指标符合实际；Grafana 面板取得真实数据；Jaeger 可查询与请求时间匹配的实际 Trace。
- 安全验证观测异常不阻止业务，验证健康、基础认证及现有业务；禁止破坏数据模拟故障。
- 提供部署前后状态及安全运维交付，自审并提交推送；仅全部通过后 completed，按仓库流程归档。

## Agent Strategy

单一主执行代理顺序执行资源检查、部署、验收及自审。无需子代理；本次角色由用户明确指定为部署与运维执行工程师。

## Working State

### Completed

- 在 main 同步并顺序部署实际系统；只新增本项目资源。应用始终单副本，未改业务代码、默认Base、HTTP/WSS协议或Schema文件。
- 完成环境/镜像/存储/内核参数检查及最小部署修复，4个PVC Bound，全部后端恢复Ready；最终 live/profile diff 为0。
- 完成真实 smoke、连续重复 smoke、补充日志/指标/Trace/认证/WS/Grafana验收及真实Discover浏览器查询。
- 五个观测后端全部停机时，业务健康/认证/管理/metrics通过；app UID/重启数及PVC UID不变。已逐个恢复，随后完整验收再通过。
- 修复服务器HTTP deadline和Kibana data view重复创建问题；保留旧失败及本轮成功证据，未降低断言或密码哈希强度。

### In progress

- 无。

### Remaining

- 无验收项剩余；按main流程提交完成报告，再归档。

### Verification status

- 本机部署执行代理：默认/服务器profile渲染与安全校验、Bash语法、目标API dry-run/diff、真实运行部署、下述端到端和故障恢复检查均已执行。
- `make observability-smoke` 最终连续两次退出0；补充验收最终退出0。Kibana Discover浏览器实际显示Documents(1)，有截图及DOM证据。
- GitHub Actions 对最终功能提交 `be09244` 的 Kubernetes清单校验、Fair Doudizhu和完整CI全部成功。CI分别执行单测/边界、构建、认证/管理race、前端和跨语言、集成/运行回归。本机未重复整个Go/前端测试套件，未调用第二个测试执行代理。
- 无真实牌局正在运行：本次部署前Namespace/app-api均不存在，当前Doudizhu仍按原配置disabled；没有启用或扩容游戏实例。实际业务回归覆盖当前启用的认证、管理与WebSocket平台功能。

## Completion Report

**completed：实际部署、三条链路、业务与故障恢复验收及最终功能CI全部通过。** 本目标为Kubernetes技术基础设施部署，不是架构重设计或业务功能开发。

### 1. 实际服务器、集群与资源

- context：`kubernetes-admin@network-learning`；Kubernetes client/server v1.36.4，两节点Ready。执行主机脱敏为`ser…6259`，工作节点`hk-worker`。没有安装/替换集群、修改其他项目资源或猜测SSH地址。
- 主节点：8 vCPU、7.75GiB RAM，Ubuntu22.04、Linux5.15.0-198、containerd2.3.3；worker：8 vCPU、7.75GiB RAM，Ubuntu20.04、Linux5.15.0-139、containerd1.7.24。
- 07:18左右Kubelet实际快照：主节点工作集3.42GiB/可用4.33GiB，worker工作集2.37GiB/可用5.38GiB。含可回收缓存的usage不当作不可用内存。ES约952MiB、Kibana559MiB、MySQL224MiB、Grafana80MiB。
- 根盘约39G/可用29G；独立ext4数据盘`/data`标称40G/可用约26G，使用约32%。没有swap，当前未见MemoryPressure或新的soft lockup。
- 先前主机曾多CPU soft lockup（最长1028–1114秒）、CPU steal最高83%、控制面6443拒绝连接。后续检测到主机已重启（非本代理执行），旧暂停runtime已被替换。本轮恢复后功能检查通过，但共享CPU steal仍约30–50%，不能宣称云侧根因已经解决。

### 2. 实际资源和镜像

Namespace为`awesome-zero-platform`。下列Deployment/StatefulSet最终均1/1 Ready，app Pod为2/2；app为单副本/Recreate，镜像始终保留：

| 类型/名称 | 实际镜像 | 最终验证 |
| --- | --- | --- |
| Deployment app-api | awesome-zero-platform/app-api:goal0037-74d2c01 | live/ready/认证/管理/WS/metrics通过 |
| app-api sidecar filebeat | docker.elastic.co/beats/filebeat:9.1.0 | 读取JSON文件，唯一requestId入库，停机日志恢复投递 |
| StatefulSet mysql | mysql:8.4.6 | 新建空库导入当前完整schema，15表；认证实际读写通过 |
| StatefulSet redis | redis:7.4.5-alpine | 认证PING、AOF状态、会话创建/查询/撤销通过 |
| StatefulSet elasticsearch | docker.elastic.co/elasticsearch/elasticsearch:9.1.0 | Ready/green、ILM、持久化及停机恢复日志查询通过 |
| StatefulSet prometheus | prom/prometheus:v3.5.0 | Ready、每Ready Pod独立UP，真实指标和保留参数通过 |
| Deployment kibana | docker.elastic.co/kibana/kibana:9.1.0 | Ready、数据view/API/真实Discover查询通过 |
| Deployment grafana | grafana/grafana:12.1.0 | Ready、数据源健康OK、预置Dashboard六面板真实数据通过 |
| Deployment jaeger | jaegertracing/jaeger:2.20.0 | Ready、OTLP/gRPC4317、实际Trace与请求时间匹配 |
| Job elastic-bootstrap | curlimages/curl:8.11.1 | Complete，ILM/模板/已有日志索引策略创建成功 |

app镜像使用既有Dockerfile从代码检查点`74d2c01`顺序`go build -p 1`构建并导入containerd，config digest为`sha256:c1ae18d2f24e1a9eac52a9d7fcb0db5e26c84c2ee81e0f7b35c04f34b040a789`。固定本机、imagePullPolicy Never，没有更换为不可匿名拉取的GHCR latest。

Service清单（全部ClusterIP）：app-api8888、mysql3306、redis6379、elasticsearch9200、prometheus9090、kibana5601、grafana3000、jaeger16686/4317/4318。没有新增公网端口、Ingress、NodePort或LoadBalancer。

其他资源：5个原有观测ConfigMap及新增生成的`obs-server-app-config-86gthfkk4d`；Prometheus只读namespaced Pod发现SA/Role/RoleBinding；app PDB；专属非默认StorageClass与4个Retain PV。最终`kubectl diff -k deploy/kubernetes-overlays/observability-server`退出0。

### 3. 真实链路与业务验收

| 检查 | 证据 |
| --- | --- |
| 日志端到端 | HTTP唯一requestId `goal0037-1b4d7b3a10e3422fb77cc53164cceb59`；应用结构化JSON→Filebeat→ES精确命中1条，Kibana API查询同一日志 |
| Discover真实界面 | 查询`goal0037-7768ae5e98ea476fbd1370f167b3bfd0`，浏览器Documents(1)，行内platform/http.access/200；截图已人工查看 |
| 指标 | /metrics返回真实计数；唯一Ready app Pod `app-api-7fc545f689-88mzh`与Prometheus UP target身份集合相等；HTTP计数及延迟count均186，进程内存31166464 bytes；不是Service负载均衡混采 |
| WebSocket | 真实认证握手101，直接metrics连接数1→0；空闲时连接数和消息率0，符合实际无消息流量 |
| Grafana | Prometheus数据源health=OK，Awesome Zero Platform Overview存在；请求数、5xx错误率、p95延迟、WS连接、内存、WS消息率六个表达式均有数据，并通过Grafana /api/ds/query实际查询 |
| Trace | Trace ID `9654672f50a640abbb556c01a9aa2766`，app-api服务实际Span1条，startTime与该真实请求时间`2026-10-29T07:20:01.003864+00:00`相符；不是只检查服务列表或端口 |
| 认证与授权 | 登录/会话/管理接口/登出成功，匿名session被401拒绝；没有打印token或密码 |
| 全部观测后端停止 | ES/Prometheus/Kibana/Grafana/Jaeger全为0，观测Pod全部退出；10轮live/ready、认证/管理/登出、/metrics通过；登录1.353秒；app UID/重启数和所有PVC UID不变 |
| 恢复后再验收 | 后端逐个恢复Ready；旧日志`goal0037-7768...`仍可查；停机期间登录日志`JVTVD3XWNHLNEQQB2NVB5JLDM4`由Filebeat补投后命中；完整smoke连续两次退出0，最终补充验收退出0 |

临时浏览器Pod使用已缓存镜像、受限资源、非root、无host挂载/PVC/Service/ServiceAccount token；取证后仅删除该临时Pod。所有测试port-forward进程已退出，业务数据卷未删除或重建。

### 4. 故障、修复和验证边界

1. GHCR匿名获取403：使用已有Dockerfile本地构建/导入，未提交注册表凭据。
2. nonroot字符串USER导致K8s拒启：profile显式UID/GID65532，保持非root和只读根文件系统。
3. 第一个日志索引可能早于模板创建：初始化Job为已有本项目日志索引附加三天ILM；实际managed=true。
4. 原5xx/sec表达式无5xx时为空：改为真实5xx百分比并提供零值回退，实际查询验证；未制造错误填图。
5. 低速拉取导致Kibana原600秒进度期限不足：profile1200秒；Prometheus readiness从1秒调整5秒，实际重启后通过。
6. 首次全停机认证在3.1秒500，随后日志为create session: context deadline exceeded。go-zero默认HTTP deadline3000ms；仅服务器profile挂载`Timeout:10000`的配置副本，测试比较全部其余字段与镜像生产配置相同，密码哈希/认证参数、业务代码均未改变。重建单个app Pod后，按原停机断言重测通过。它是共享CPU环境的配置缓解，不能证明任何负载下都不会超时。[go-zero配置定义](https://github.com/zeromicro/go-zero/blob/v1.10.2/rest/config.go)
7. 重复smoke遇到Kibana Duplicate data view HTTP400：改为先GET已有view、校验ID/索引模式/时间字段，仅404时创建；任何其他错误仍失败。修复后连续两次真实smoke通过，没有放行任意400或删view来规避。
8. 验收工具适配：Kibana内部查询按浏览器头部要求发送；补充脚本曾端口重叠，该次无效结果保留并顺序重跑；浏览器取证改为真实页面就绪/CDP。未把工具失败算作业务通过，也未弱化验收标准。

### 5. 数据位置、保留与磁盘边界

`awesome-zero-local`为非默认静态StorageClass（no-provisioner/WaitForFirstConsumer），PV均Retain，PVC最终全部Bound：

| PVC → PV | 声明容量 | 本机路径 | 生效策略 |
| --- | --- | --- | --- |
| data-mysql-0 → awesome-zero-mysql | 4Gi | /data/awesome-zero-platform/mysql | 表数据不自动删除；binlog259200秒/100MiB轮转 |
| data-redis-0 → awesome-zero-redis | 1Gi | /data/awesome-zero-platform/redis | AOF；maxmemory128MB/noeviction；不清空业务数据 |
| data-elasticsearch-0 → awesome-zero-elasticsearch | 8Gi | /data/awesome-zero-platform/elasticsearch | awesome-logs-three-days，3天ILM；low/high/flood_stage为12gb/8gb/5gb，已读取实际cluster settings |
| data-prometheus-0 → awesome-zero-prometheus | 3Gi | /data/awesome-zero-platform/prometheus | 实际flags 3d/512MiB，WAL/head额外占空间 |

声明合计16Gi不是文件系统硬配额；数据盘仍约26G可用，需监测实际目录增长、容量及延迟。ILM已创建并管理实际索引，未等待三天验证过期删除。Jaeger为内存Trace存储，重启/停止会丢旧Trace；Grafana非预置UI状态、app日志和Filebeat注册表为emptyDir，Pod替换可能丢未投递数据；本次停机保持app Pod不变，所以验证了日志恢复补投，不能扩大为任意崩溃下零丢失。

本机`/etc/sysctl.d/99-awesome-zero-elasticsearch.conf`将vm.max_map_count由65530设为1048576，重启后实测仍1048576。未使用特权容器。回退须先完全停用ES并防止自动恢复，再移除文件并恢复65530。

### 6. 安全访问

在服务器运行`make observability-forward`，仅绑定127.0.0.1。另一终端通过已有可信SSH参数建立隧道：

```bash
ssh -N -L 15601:127.0.0.1:15601 -L 13000:127.0.0.1:13000 \
  -L 16686:127.0.0.1:16686 -L 19090:127.0.0.1:19090 \
  -L 19200:127.0.0.1:19200 <trusted-host>
```

客户端访问Kibana `http://127.0.0.1:15601`、Grafana `http://127.0.0.1:13000`、Jaeger `http://127.0.0.1:16686`、Prometheus `http://127.0.0.1:19090`、ES `http://127.0.0.1:19200`。若kubectl在客户端可直接port-forward，不需SSH二次转发。Grafana用户admin，密码只在服务器root私有凭据文件，不写Git/普通日志；Ctrl-C停止转发，不开放公网UI。

### 7. 日常维护、重启、升级和安全卸载

- 必须设置`OBS_KUSTOMIZE_OVERLAY=$PWD/deploy/kubernetes-overlays/observability-server`，加载现有安全保存的Grafana密码；先render/diff再up/status/smoke。不得盲目用默认Base/Overlay，它会恢复两副本和不同镜像。
- 日常检查节点Ready/CPU steal/soft lockup、磁盘空间及I/O、events、PVC、rollout、ES health/ILM、每Pod Prometheus target和保留参数。初始化Job有24小时TTL；被清理后重新up会按同一声明幂等重建，再执行smoke。
- 观测后端逐个rollout/restart并检查数据，保留PVC。app的Recreate/config变更会中断内存牌局，启用游戏后必须维护窗口，不允许为更新扩容为两个进程。未来更新镜像须固定版本、先构建导入/确认可拉取，保留旧镜像和reviewed profile便于回退。
- 升级前导出配置和逻辑备份/快照到异机并验证恢复。本次尚无用户提供的异机备份目的地，不声称完成备份或灾备演练。
- 卸载先确认业务窗口及备份，按所有权逐项审查；先为app准备保留镜像/单副本的补丁，移除Filebeat、观测env和相关卷时安排重启。服务器专属HTTP配置挂载应保留或单独评估回退，不应随观测停用意外恢复3秒deadline。
- 然后仅停止/移除审查过的本项目观测Deployment/StatefulSet/Job/Service/ConfigMap和Prometheus只读RBAC。禁止delete整个Overlay/Namespace，禁止清理MySQL/Redis/PVC/PV目录，禁止盲目重应用两副本Base；本次未执行系统卸载。
- 本轮没有重启主机、替换集群/Cilium、改其他项目资源。Xray和nginx原容器保留；没有对其他项目做完整功能测试。

### 8. 证据、CI、自审和剩余风险

私有根目录`/root/.local/state/awesome-zero-platform/goal0037/`保存凭据（credentials.json，0600，父目录0700）及前轮失败记录，**不要整体打包/上传**。本轮非敏感证据在其`resume-20261029/`：smoke-final/repeat、extended-final、outage日志/JSON、旧3秒失败、log-recovery、Discover截图/DOM、final-inventory/events/nodes；截图副本`.runtime/goal0037/discover.png`可在本机查看，不入Git。

- 本部署代理实际执行了所有上述服务器运行验收和本机窄检查；未用Pod Running或静态YAML代替端到端。
- 最终功能提交`be09244`：[Kubernetes清单校验](https://github.com/oublie6/awesome-zero-platform/actions/runs/40834833203)、[Fair Doudizhu回归](https://github.com/oublie6/awesome-zero-platform/actions/runs/40834833235)已成功；[完整CI](https://github.com/oublie6/awesome-zero-platform/actions/runs/40834833568)也已成功。CI的Kind环节是API dry-run，不是本服务器的运行验收；实际K8s数据查询由本代理执行。
- 已完成单一主代理的独立第二遍**自审**（不是独立评审）：重读AGENTS、目标及引用运维文档，逐行检查配置/脚本/测试diff；核对无业务/Schema/Base改动、单副本与镜像、完整配置一致性、Secret引用与泄漏扫描、Retain卷所有权、索引初始化竞态、data view重复/错误配置失败分支、有限超时与取消、全部后端停机及逐项恢复顺序、临时Pod/forward清理和点验收的可证明边界。
- 自审追踪成功HTTP→认证→MySQL/Redis及JSON→Filebeat→ES/Kibana、HTTP→OTLP→Jaeger、metrics→Prometheus→Grafana；失败路径以首次3秒认证deadline及重复view400为反例，修复后完整回归。额外发现旧卸载文档会盲目重应用两副本Base，已改为保留当前镜像/单副本/服务器配置的审查流程；同时删除了无法仅凭日志证明的“密码验证结束时间”因果表述。
- 未发现剩余阻断缺陷。未等待三天验证ILM删除、未做物理游戏客户端/长期容量或异机灾备演练，不将这些扩大成已验证能力。
- 剩余风险：共享宿主机CPU steal与旧soft lockup底层根因未消除；这是开发/集成栈，不承诺生产SLA。ES/Kibana集群内未认证且无NetworkPolicy隔离，必须保持无公网暴露；真正生产使用需要另外设计TLS/认证/隔离和异机备份。固定节点local PV不支持自动跨节点容灾；Grafana/Jaeger及emptyDir耐久性有限。未启用真实斗地主牌局，物理客户端和长期压测不属于本次部署验收。

全部验收已通过；按main顺序提交完成报告并推送，然后使用archive-goal.sh归档。功能提交`f17afcb`（服务器deadline）和`be09244`（smoke幂等）均已推送；完成报告提交`f718e25`亦已成功推送main。本归档提交SHA及push结果见最终交付。
