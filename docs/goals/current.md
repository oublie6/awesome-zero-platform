# Goal 0037: Kubernetes 可观测性实际部署与验收

## Status

- State: blocked
- Started: 2026-10-28
- Completed:
- Blockers: 当前唯一 context 为已有 network-learning 集群，目标身份及共享使用范围待确认；项目 Namespace、app-api、MySQL/Redis Service、应用 Secret 均不存在；无 StorageClass/PV；容量余量未完成核实，且本机 vm.max_map_count 不满足要求。未执行集群写入。

## Goal

在用户确认的云服务器 Kubernetes 环境实际部署日志、指标、追踪系统，并验证真实数据可查询及业务可用性。属于可复用技术基础设施的部署与验收，不改变平台架构或业务语义。用户明确授权初始化此目标及必要的最小部署配置修复。

## References

- `AGENTS.md`
- `docs/operations/logging-observability.md`
- `deploy/kubernetes/`
- `deploy/kubernetes-overlays/observability/`
- `scripts/k8s-observability.sh`
- `scripts/k8s-observability-smoke.sh`

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
- 仅全新专用测试集群可在容量足够时补齐最小持久化基础设施；未知用途共享集群须先确认使用范围。
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

- 完整阅读规范、当前模板、用户指定运维文档、Base/Overlay、Dashboard 和两份脚本；未读取归档目标。
- 初始 main 工作区干净；`git pull --ff-only` 返回 Already up to date，起点 `2f4068f`。
- 完成下述只读环境检查和本地渲染。无集群、主机配置、数据库或业务变更。

### In progress

- 无；等待目标和共享环境授权范围确认后恢复。

### Remaining

- 确认 `network-learning` 是否目标，或提供已有可信 SSH/context。
- 若使用现有集群，明确是否允许新建项目 MySQL/Redis 和持久化供应器、节点资源预算及存储路径/回收备份策略；评估两节点真实余量后决定是否适合完整栈。
- 确认节点 sysctl 的批准配置和回退方式、可获取的 app-api 镜像及初始化凭据；不得复用未知数据卷。
- 部署前变更 diff/影响和回退审查、所有实际部署与端到端验收、运维交付均待执行。

### Verification status

2026-10-28 UTC，由本次 Codex 部署执行代理实测：

| 检查 | 结果 |
| --- | --- |
| 执行主机 | 本地云主机，标识脱敏 `ser…6259`；已有 SSH 会话，不是无集群沙箱 |
| 系统 | Ubuntu 22.04 LTS，Linux 5.15.0-198-generic，containerd 2.3.3 |
| 本机资源 | 8 CPU；7936 MiB RAM，可用约 6034 MiB，无 swap；根盘 39 GiB/剩余 29 GiB，/data 独立 ext4 40 GiB/剩余 31 GiB |
| context/版本 | `kubernetes-admin@network-learning`，客户端/服务端 v1.36.4；仅此 context |
| 节点 | 2 Ready，无 MemoryPressure/DiskPressure；各 8 CPU，allocatable 约 7.65 GiB RAM，无 taint |
| 工作节点 | `hk-worker`，Ubuntu 20.04.4、Linux 5.15.0-139-generic、containerd 1.7.24；未取得主机实际空闲资源/挂载/sysctl |
| requests | 工作节点 100m CPU/10Mi RAM；控制节点 950m/250Mi。系统 Pod 多数未声明完整 requests，不能据此认定空闲容量 |
| 实际节点指标 | `kubectl top nodes` 返回 `Metrics API not available`；不据此安装组件 |
| 已有环境 | Cilium/CoreDNS/控制面正常运行，netlab 有 Completed Pod；主机 Docker 的 xray 与 nginx subscription 服务已运行约 6 天，属于需保留的其他业务 |
| 项目基础设施 | Namespace 不存在；指定 app-api Deployment、mysql/redis/app-api Service、应用 Secret 查询均返回 Namespace NotFound；不存在可核实的当前 app-api 副本/镜像/活跃牌局 |
| 存储 | 全集群无 StorageClass、PV、PVC，无法直接绑定 Overlay 所需 8Gi ES/3Gi Prometheus 卷 |
| 内核 | 本机 vm.max_map_count=65530，低于要求 1048576；未修改，工作节点未核实 |
| 权限 | namespaced Deployment create 检查 yes；其余部署权限未穷举 |
| 远程连接 | SSH 配置只有 GitHub 连接，无目标服务器别名；未猜测账号或地址、未尝试未经确认的远程登录 |
| 渲染 | `kubectl kustomize deploy/kubernetes-overlays/observability` 成功，1015 行；脚本 `bash -n` 通过；未执行 apply/diff/运行验收 |

渲染中的计划镜像（全部未部署/未验证拉取）：app-api `ghcr.io/oublie6/awesome-zero-platform/app-api:latest`；Filebeat/Elasticsearch/Kibana `9.1.0`；Prometheus `v3.5.0`；Grafana `12.1.0`；Jaeger `2.20.0`；bootstrap `curlimages/curl:8.11.1`。Base 的 app-api 默认 2 副本，恢复实施时需显式审查单实例要求，禁止直接套用。

现有 smoke 仅检查 ES requestId、Kibana data view、Prometheus UP target 数量、Grafana 配置存在和 Jaeger service 存在，不足以单独证明本目标全部验收项；恢复后还须验证实际 Trace 时间、Ready Pod 身份、面板数据、ILM、认证/业务和故障隔离。

## Completion Report

**未完成，blocked，不归档。** 根因是目标/共享使用范围未确认及基础设施、存储和容量验收前置缺失，尚无证据归因业务代码或镜像网络故障。未运行 `observability-up` 或 `observability-smoke`；三条数据链路均未验收，不声称成功。没有新建 Deployment/StatefulSet/Service/PVC，也未生成 Grafana 密码。已有资源未被本次操作修改；业务功能未做请求验收，不能声称业务回归通过。

只修改本目标记录。自审重新对照用户安全边界、仓库规范、部署文档与渲染内容，确认不能把已有双节点集群视为全新专用测试集群，不能把 allocatable 当实际余量，也不能把静态渲染当端到端验收。未修改业务、Schema、生产代码或测试；未触发本次 GitHub Actions 验证。

恢复后的安全访问方案：在持有目标 kubeconfig 的可信机器运行 `make observability-forward`，脚本仅绑定 127.0.0.1；Kibana 15601、Grafana 13000、Jaeger 16686、Prometheus 19090、ES 19200。若 kubectl 在服务器，通过已确认的 SSH 主机配置用 `ssh -N -L 15601:127.0.0.1:15601 -L 13000:127.0.0.1:13000 -L 16686:127.0.0.1:16686 -L 19090:127.0.0.1:19090 -L 19200:127.0.0.1:19200 <trusted-host>` 转发；这只是待部署后的访问步骤，本次没有启动转发或开放公网端口。

计划保留策略：ES 8Gi/日志 ILM 7 天；Prometheus 3Gi/7 天或 1GB；Jaeger 内存存储重启丢失，Grafana 非预置状态和 app/Filebeat emptyDir 随 Pod 丢失。当前没有实际数据位置可交付。启动、升级和卸载须在部署后基于真实资源制定：保留应用镜像/副本与卷，先检查活跃牌局；禁止直接 delete Overlay 或 Namespace，也不能盲目重应用两副本 Base；逐项审查资源所有权与备份再操作。本次没有部署变更，无需环境回退。

提交与推送结果见承载本记录的 main 提交及本次最终执行报告。
