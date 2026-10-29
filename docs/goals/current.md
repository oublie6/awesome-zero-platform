# Goal 0037: Kubernetes 可观测性实际部署与验收

## Status

- State: blocked
- Started: 2026-10-28
- Completed:
- Blockers: 宿主机多 CPU soft lockup（最长超过 1000 秒）、CPU steal 最高实测 83%、磁盘/调度长停顿，Kubernetes API 6443 无监听并拒绝连接；安全减载未能全部经 API 完成。完整 smoke 失败，登录回归出现会话创建超时 500，禁止标记 completed。

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

- 初始 main 干净，`git pull --ff-only` 成功，起点 `2f4068f`；初次环境 blocker 记录提交 `74d2c01` 已推送。用户确认后继续实际部署。
- 阅读规范、指定运维文档、Base/Overlay、两份脚本；未读取归档目标。实现属于 Kubernetes 技术基础设施，未改变 Go 模块边界、业务代码、协议、Schema 或默认应用 Base。
- 实测两节点初始合计约 11.7GiB 可用 RAM，按节点分配负载；创建 Namespace、Secret、专属静态 Retain PV 和非默认 StorageClass。
- 用现有 Dockerfile 顺序低并发构建 app-api，导入本机 containerd；按单副本/Recreate 部署。仅向新空数据库导入 current.sql（15 张表），无 seed、无清库。
- ES、Prometheus、Grafana、Jaeger 曾 Ready；日志确实入库，ILM 管理确实生效，Prometheus 实际 Pod target UP。Kibana 拉取成功但未完成健康验收。
- 用户要求的磁盘保留策略已配置并部分实测；诊断、安全减载、失败证据及当前不完整运行状态见下。

### In progress

- 无。因主机级故障停止部署推进，不继续启动或恢复完整 ELK。

### Remaining

- 先通过云控制台/主机维护流程定位 soft lockup、虚拟 CPU 调度、块设备延迟和控制面故障。不得未经评估重启主机、重装集群或替换 Cilium；这些会影响其他服务。
- API 恢复后核对本项目所有副本、Pod 与 runtime 状态，完成下述安全减载/恢复协调；当前不是干净的 Kubernetes 回退。
- 主机稳定后，单独应用 Prometheus 5 秒 readiness 超时补丁（配置已写入但运行时未验证），逐个恢复组件，重新检查 PVC/ILM/健康。
- 完整重跑 smoke、唯一 requestId 与 Kibana Discover 查询、Grafana 数据源及全部面板、与请求时间匹配的实际 Jaeger Trace、WS、认证与故障隔离。全部通过后才 completed/归档。

### Verification status

执行者：本次用户明确授权的 Codex 部署与运维代理。没有子代理、独立评审或另一个补充测试执行者。

- 最终离线复核通过：默认23资源校验、服务器profile安全校验（复用27个namespaced资源契约检查）、两份Bash脚本语法、git diff --check、实际生成凭据值泄漏扫描；确认server业务/Schema/默认应用Base无diff。
- 此前还完成目标 API 服务端 dry-run、部署前 diff、镜像构建/导入、逐组件真实部署及下述实际查询和业务请求；后期控制面故障使运行验收失败。
- 已执行 `make observability-config`、`make observability-up`（明确选择服务器 profile）。重复 apply 时 app-api 为 unchanged，没有扩大副本或替换镜像。
- 已实际执行 `make observability-smoke`，退出 **2**：首次 rollout 查询即因目标 Kubernetes API connection refused 失败。没有跳过后续检查来宣称 PASS。
- 完整补充验收脚本已准备在服务器私有证据目录，但因环境故障未完整执行；不得把准备脚本视为测试通过。
- 新增 GitHub Actions 的 server profile 安全校验步骤；本次没有取得该最终提交的 CI 运行结果，不能声称 CI 通过。
- 未改 Go/前端代码，未另跑 Go 单测、前端构建、浏览器 UI 或竞态测试。实际构建是现有 Dockerfile 的 `go build -p 1`。

## Completion Report

**结果：blocked，未完成，不归档。服务曾实际部署运行，但三条链路及业务回归没有全部通过。**

### 1. 服务器、集群和资源

| 项目 | 实测 |
| --- | --- |
| 主机/context | 执行主机脱敏 `ser…6259`，`kubernetes-admin@network-learning`；仅此 context。无其他可信目标 SSH 配置，未猜测远程账号/IP |
| Kubernetes | client/server v1.36.4；初始两节点 Ready，无 MemoryPressure/DiskPressure |
| 主节点 | 8 vCPU、7936MiB RAM；Ubuntu 22.04、Linux 5.15.0-198-generic、containerd 2.3.3 |
| 工作节点 | `hk-worker`，8 vCPU、约 7.65GiB allocatable RAM；Ubuntu 20.04.4、Linux 5.15.0-139-generic、containerd 1.7.24 |
| 实际余量 | 初始节点 summary：工作节点约 6.78GiB、本机约 4.88GiB 可用；非仅使用 requests/allocatable 推断。Metrics API 未安装，未为此更改集群 |
| 磁盘 | 本机根盘 39GiB/剩余约29GiB；独立 ext4 `/data` 40GiB，初始剩余31GiB，最终约26GiB（占用32%）；工作节点部署中剩余约73GiB |
| 最终限制 | RAM 尚约5.3GiB 可用，磁盘未满；但 CPU steal 实测31–83%，内核出现多 CPU soft lockup、watchdog 和长 I/O/调度停顿，不能认定为健康可用容量 |
| 已有业务 | 原有 Xray、nginx subscription Docker 容器未修改；最终 `docker ps` 仍运行，Xray 标记 healthy。未对其业务功能作完整回归 |

### 2. 实际组件、镜像和最后确认状态

所有新资源位于 `awesome-zero-platform`，PV/StorageClass 为带项目前缀的集群资源。

| 资源 | 实际镜像 | 曾验证 | 停止推进时的状态 |
| --- | --- | --- | --- |
| Deployment app-api + sidecar | `awesome-zero-platform/app-api:goal0037-74d2c01`；`docker.elastic.co/beats/filebeat:9.1.0` | 单副本，2/2 Ready，JSON 文件和 Filebeat 连接 ES | 未改副本/镜像；API 故障后直接 Pod health 200，但登录500，不能称业务正常 |
| StatefulSet mysql | `mysql:8.4.6` | Ready，新 schema 15 表，binlog参数实测 | 未停止/清理；最终无法经 API 再核实 |
| StatefulSet redis | `redis:7.4.5-alpine` | Ready，认证 PONG | 未停止/清理；后续登录会话创建超时需排查 |
| StatefulSet elasticsearch | `docker.elastic.co/elasticsearch/elasticsearch:9.1.0` | Ready/green，44条日志，索引 managed=true；后期迁移期间 yellow/探针抖动 | API 缩容失败；仅本项目容器已由 ctr 暂停，实际 task=PAUSED |
| StatefulSet prometheus | `prom/prometheus:v3.5.0` | Ready，实际 Pod UP、HTTP查询有值、保留参数 | API缩容失败；runtime暂停请求超时；已撤销残留 freeze 请求，cgroup.freeze=0、task=RUNNING |
| Deployment grafana | `grafana/grafana:12.1.0` | Ready，日志显示预置 Dashboard 加载 | scale replicas=0 请求成功；API故障后未确认 Pod 完全退出 |
| Deployment jaeger | `jaegertracing/jaeger:2.20.0` | Ready，日志确认4317/4318监听 | scale replicas=0 请求成功；未验收到真实 Trace，最终 Pod 退出未经 API 再确认 |
| Deployment kibana | `docker.elastic.co/kibana/kibana:9.1.0` | 镜像拉取成功，启动并执行 saved objects 初始化 | 未 Ready；停止请求未成功确认，可能仍在重试，须 API 恢复后核对 |
| Job elastic-bootstrap | `curlimages/curl:8.11.1` | Completed，三个请求 acknowledged=true | 保持原有24小时TTL；没有删除数据 |

app-api 源码检查点 `74d2c01`，镜像 config digest `sha256:c1ae18d2f24e1a9eac52a9d7fcb0db5e26c84c2ee81e0f7b35c04f34b040a789`；本机构建后导入 containerd，`imagePullPolicy: Never` 并固定本节点，不依赖不可匿名获取的 GHCR tag。

Service 清单：`app-api:8888`、`mysql:3306`、`redis:6379`、`elasticsearch:9200`、`prometheus:9090`、`grafana:3000`、`kibana:5601`、`jaeger:16686/4317/4318`。全部 ClusterIP，无 Ingress/NodePort/LoadBalancer 或新增公网监听。

### 3. 三条链路与业务验收证据

| 项目 | 实际结果，不能扩大解释 |
| --- | --- |
| 应用 JSON → Filebeat → ES | 共享日志文件实际存在；Filebeat 重试后连接成功；`awesome-logs-2026.10.29` 实际44条文档、约120KB、green，ILM managed=true。**未完成唯一 requestId 精确匹配和 Kibana Discover 验收** |
| /metrics → Prometheus | target 对应 `app-api-d5886c97-m7nm9`，health=up；HTTP rate查询有真实正值，修正后的5xx错误率实测0%；flags实际 `3d/512MiB`。后续有 readiness 超时，**未完整验收延迟/WS及所有面板数据** |
| Grafana | Pod曾Ready、预置Dashboard加载；**数据源健康与全部面板真实查询尚未完成** |
| OTLP → Jaeger | Pod曾Ready、OTLP监听日志确认；**没有取得实际 Trace，不能据端口开放宣称链路成功** |
| 初始业务 | 观测后端都尚未启动时，实际 live/ready/login/session/admin/logout 通过；初始化期间重复检查也通过 |
| 故障后业务 | ES暂停/API不可用后，直接Pod请求 `/health/live` 200/0.020s，`/health/ready` 200/0.356s；登录返回500，应用日志 `request failed: create session: context deadline exceeded`。**故障隔离/最终业务回归未通过** |
| 完整 smoke | 对真实目标执行，make退出2，Kubernetes API拒绝连接；未完成后三条端到端断言 |

登录500的复现：在该环境直接请求 app-api Pod 的 `/auth/login`，使用私有凭据目录中本次创建的 `goal0037admin` 账号；此前同一流程成功，主机故障后会话创建超时。先恢复主机、控制面和 Redis 的稳定性再复现，尚不能将其归因于业务代码或单独归因于观测停机。没有修改业务代码或测试语义掩盖错误。

### 4. 问题、分类与最小修复

1. **镜像授权**：默认 GHCR 镜像匿名获取403；使用已有 Dockerfile 从已提交源码构建/导入，未请求或提交注册表凭据。
2. **K8s安全配置**：distroless USER 为字符串 nonroot，runAsNonRoot 无法验证；服务器 profile 显式 UID/GID65532 后正常启动。
3. **保留策略初始化竞态**：Filebeat可能先创建索引；初始化Job同时附加3天ILM到已有本项目日志索引，实际 managed=true。
4. **Dashboard配置**：原5xx/sec表达式在无5xx时为空；最小修改为带零值回退的真实5xx百分比，实际PromQL验证0%，没有制造500来填图。
5. **启动期限/探针**：Kibana镜像拉取8m8s，再初始化超过原600s进度期限；服务器profile设置1200s，仍必须实际Ready。Prometheus1秒探针多次超时，服务器profile改为5秒，**该最后补丁尚未运行验证**。
6. **服务器/内核性能，最终blocker**：ES记录cluster-state写入13–63秒、timer停顿46秒；CPU steal最高83%；内核多CPU soft lockup 22–48秒及1028–1114秒，systemd-networkd/journald watchdog，journal不干净关闭。随后6443无监听、TLS超时/connection refused。容器运行时曾报告apiserver/etcd Running，这不代表API健康。精确的底层内核/虚拟化/存储根因需主机与云侧调查，不能声称仅改YAML即可修复。

### 5. PVC、实际数据位置及保留策略

静态 StorageClass `awesome-zero-local`，`kubernetes.io/no-provisioner`，非默认，WaitForFirstConsumer；以下PV均Retain且初次实际PVC Bound：

| PVC → PV | 容量 | 本机独立数据盘目录 | 保留策略 |
| --- | --- | --- | --- |
| data-mysql-0 → awesome-zero-mysql | 4Gi | /data/awesome-zero-platform/mysql | 表数据不自动清理；binlog259200秒/100MiB轮转，实测开启 |
| data-redis-0 → awesome-zero-redis | 1Gi | /data/awesome-zero-platform/redis | AOF持久化，不自动清空业务数据，maxmemory128MB/noeviction |
| data-elasticsearch-0 → awesome-zero-elasticsearch | 8Gi | /data/awesome-zero-platform/elasticsearch | awesome-logs-three-days，索引生命周期3天；free-space水位12/8/5GB |
| data-prometheus-0 → awesome-zero-prometheus | 3Gi | /data/awesome-zero-platform/prometheus | 3天或512MiB，已查询实际flags；WAL/head额外占空间 |

声明合计16Gi，**local PV容量不是文件系统硬配额**。最终/data约26GiB空闲；业务数据不会因空间紧张被脚本清理。没有提供异机备份目的地，尚无自动异机备份/恢复演练。Jaeger为内存存储，停止/重启丢Trace；Grafana非预置状态、app日志和Filebeat注册表是emptyDir，Pod替换会丢失。3天ILM已配置生效，不代表已等待3天验证删除。

宿主机仅新增 `/etc/sysctl.d/99-awesome-zero-elasticsearch.conf`，vm.max_map_count从65530调整到1048576。ES固定本机，无特权容器修改sysctl。完全停用ES并确认不会自动恢复后，移除该文件并 `sysctl -w vm.max_map_count=65530` 才能回退；当前不回退以免残留Deployment恢复时失败。

### 6. 安全访问（仅在API和组件恢复后）

在持有目标kubeconfig的可信机器运行 `make observability-forward`，只绑定127.0.0.1：Kibana15601、Grafana13000、Jaeger16686、Prometheus19090、ES19200。**当前API不可用/部分组件停止，这些UI尚不能交付为可用地址。**

如果kubectl在服务器，在客户端用已有可信SSH连接参数建立：

```bash
ssh -N -L 15601:127.0.0.1:15601 -L 13000:127.0.0.1:13000 \
  -L 16686:127.0.0.1:16686 -L 19090:127.0.0.1:19090 \
  -L 19200:127.0.0.1:19200 <trusted-host>
```

浏览器访问客户端对应127.0.0.1端口。Grafana用户admin；凭据仅在服务器 `/root/.local/state/awesome-zero-platform/goal0037/credentials.json`（0600、父目录0700），不得打印到普通日志或提交。ES/Kibana仍为开发模式、集群内未认证，不能因此当作生产安全方案。

### 7. 减载、恢复、重启、升级与安全卸载

- **当前不是完整回退**：G/J的scale=0 API请求成功；Kibana/ES/Prom的scale请求失败/未确认。ES通过精准核对namespace和container name后仅暂停其本机task；记录在私有目录 `elasticsearch-paused-container.txt`，ctr实际PAUSED。Prom暂停超时留下freeze=1，已经仅对该容器撤销，freeze=0、task=RUNNING。未暂停或停止DB/app/其他项目。
- API恢复后，先核对所有本项目对象的实际副本、Pod、PVC和runtime状态。若继续保持停载，先把本项目五个观测后端的desired replicas设0，再核验暂停容器ID归属并resume该ES task，使其能正常退出。**runtime PAUSED不是持久化K8s停用配置；宿主机重启可能按残留spec恢复负载。**
- 不擅自重启主机或修改控制面、Cilium、其他项目。先维护共享主机/云侧问题，再判断是否具备重新启动完整栈的条件；不能只看内存、磁盘容量或Running。
- 恢复必须选 `OBS_KUSTOMIZE_OVERLAY=$PWD/deploy/kubernetes-overlays/observability-server`，审查diff、保留app镜像及单副本，逐组件启动。**不要直接套用默认Base/Overlay，它会恢复两副本和另一镜像。** 故障排除前不要运行整套up。
- 日常检查节点CPU steal/soft lockup、磁盘空间和延迟、events、PVC、rollout、ES健康/ILM、Prometheus每Pod target和保留参数；运行手册的只读诊断命令可复用。安排异机备份并实测恢复。
- 升级前备份MySQL/ES/Prometheus数据及配置，固定镜像版本，逐项维护验证。app的Recreate会中断内存牌局，必须安排维护窗口；不得用扩容双实例绕过。
- 安全卸载须先确认活跃业务与备份，按所有权逐项审查、停止本项目观测后端；移除sidecar/观测env会重启app，须审查保留镜像/单副本的专用补丁。禁止delete整个Overlay/Namespace或盲目重应用Base；不删除MySQL、Redis、PVC、PV目录，也不以卸载为名清空数据。当前未执行卸载/资源删除。

### 8. 证据、自审和交付范围

私有证据目录 `/root/.local/state/awesome-zero-platform/goal0037/` 包含渲染结果、部署前验收前inventory/events、kernel-warnings.txt、runtime-tasks.txt、smoke-failed.log、business-after-safe-stop.json、app-errors-redacted.txt及测试脚本。目录同时含credentials.json，**禁止整体打包、上传或提交**。目标仓库仅存非敏感摘要和部署配置，不含kubeconfig、密码、token或原始Secret。

单一主代理进行了**自审**，不是独立评审：重新阅读规范/目标/运维文档，对照副本与镜像、模块无改动、卷所有权和初始化路径、Secret引用、ClusterIP、文件权限、ILM创建竞态、局部成功/超时/控制面失联、减载未完成和恢复顺序。成功流跟踪了HTTP→应用→认证→MySQL/Redis；失败流跟踪了主机停顿→探针/端点/DNS→Kibana重试及登录会话超时。发现并修复上述配置问题；没有将Running、已加载Dashboard或监听端口当成端到端通过。

最终仍未证明真实Discover查询、Grafana全部面板、实际Trace、WS及最终认证/故障隔离，且安全减载需控制面恢复后收尾。保留这些不确定性和运行风险，**不得标记completed或归档**。最终静态检查、提交SHA及push结果见本次最终执行回复；初始checkpoint `74d2c01` 已成功推送main。
