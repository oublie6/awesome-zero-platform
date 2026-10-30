# Goal 0038：Kubernetes 补充部署验收报告

结果：**通过**。2026-10-30 UTC。仅执行获授权的服务器部署和补充验收，没有修改业务源码、测试、Schema、AGENTS.md 或归档 Goal。

## 代码与 CI

- 部署时 main 已 `git pull --ff-only` 同步且工作区干净，部署源码 SHA：`2feb934d41404b23bfb4126be822e72ade07729a`。
- 应用代码与最终功能验证提交 `7fed9ed85abc21e474b3c4d8735b20e29cd5a288` 完全相同，后续仅 Goal 文档变化。
- 主 CI 成功：https://github.com/oublie6/awesome-zero-platform/actions/runs/40994258994
- Fair Doudizhu 回归成功：https://github.com/oublie6/awesome-zero-platform/actions/runs/40994258988
- 上述测试由 GitHub Actions 执行；源码实现、自审由 ChatGPT 在原 Goal 完成。本机 Codex 只完成此次交接明确要求的真实集群部署与验收，没有重复整个代码测试套件。

## 部署结果

- context：`kubernetes-admin@network-learning`，两节点 Ready，Kubernetes v1.36.4。
- 目标节点：`ser…6259`，linux/amd64。现有单副本、Recreate、Never 拉取策略保留。
- 原镜像：`awesome-zero-platform/app-api:goal0037-74d2c01`。
- 新镜像：`awesome-zero-platform/app-api:goal0038-gozero-first`。
- 使用仓库 Dockerfile 构建，`go build -p 1`；Docker config / 实际 Pod imageID：`sha256:014bc25551f1cdd6699bcc717f8c702eaaafc9da3a4dbc9683f49661ca4df48a`。
- containerd OCI manifest digest：`sha256:bc25cb4506b6995fc00bab406d2f10c5a9ec5474cefa3aeb63d08eca18a2b843`。
- 新 Pod：`app-api-f7f8f4dcd-swf68`，app-api + Filebeat **2/2 Ready、零重启**。`rollout status` 成功。
- 唯一 Deployment 变更是 app-api 容器镜像。保留服务器 10 秒 HTTP deadline、原 ConfigMap、Secret、Filebeat、安全上下文和所有卷引用。
- 完整 overlay 最终 `kubectl diff -k` 退出 0，没有未处理漂移。

## 初始阻塞及获准处置

原 `elastic-bootstrap` 完成 Job 被 86400 秒 TTL 清理，部署前 smoke 因 NotFound 失败。按交接要求停止，并保留失败证据；用户回复“允许”后，仅从现有 server overlay 抽取原 Job，server dry-run 审查后重建。Job 在约 10 秒内 Complete，按既定配置幂等写入 3 天日志 ILM/模板及对应索引设置，没有删除数据或重启观测组件。随后 Diff 只剩 app-api 镜像，才执行更新。

## 真实业务回归

全部通过：

- `/health/live`、`/health/ready`。
- 部署前创建的专用测试 Session 在升级后仍可查询，并访问 `/admin/me`；证明已有账号和 Redis Session 兼容保留。
- 新登录、Session 查询、管理员账号/角色/权限策略读取；匿名 Session 与管理员接口返回 401。
- Refresh 轮换成功，旧 Refresh 重放 401；Logout 成功，随后 Access 与 Refresh 均返回 401。只撤销专用验收 Session，没有执行账号级批量撤销。
- 真实带认证 WebSocket 返回 101；连接 gauge 从 1 回到 0。
- MySQL 真实业务 SQL 读取和 Redis Session 创建/读取/刷新/撤销均通过；实际调用由下述 Trace 佐证。
- Doudizhu 按原配置 disabled，更新前活跃 WebSocket 为 0，未开启或扩容游戏；没有真实牌局客户端验收的声明。

## Jaeger：真实 HTTP → SQL / Redis

两条 Trace 均来自实际请求的 traceparent，已从目标 Jaeger `/api/traces/<id>` 查询原始数据。服务名为 `app-api`；逐个确认 SQL/Redis 子 Span 引用对应 HTTP Span，duration > 0 且时间包含在 HTTP 请求内。SQL 和 Redis 是 HTTP 下的调用序列，不宣称数据库直接调用 Redis。

| 请求 | Trace ID | Span | 时长 ms |
| --- | --- | --- | --- |
| login | `946c0815809f41768f1af7e9326eb52e` | `sql` | 8.423 |
| login | `946c0815809f41768f1af7e9326eb52e` | `sql` | 1.151 |
| login | `946c0815809f41768f1af7e9326eb52e` | `sql` | 1.373 |
| login | `946c0815809f41768f1af7e9326eb52e` | `redis` | 5.007 |
| login | `946c0815809f41768f1af7e9326eb52e` | `/auth/login` | 785.874 |
| authenticated-admin-accounts | `e172d01ddac4456d9aada7be38a24860` | `redis` | 0.665 |
| authenticated-admin-accounts | `e172d01ddac4456d9aada7be38a24860` | `sql` | 1.381 |
| authenticated-admin-accounts | `e172d01ddac4456d9aada7be38a24860` | `sql` | 0.967 |
| authenticated-admin-accounts | `e172d01ddac4456d9aada7be38a24860` | `sql` | 4.718 |
| authenticated-admin-accounts | `e172d01ddac4456d9aada7be38a24860` | `/admin/accounts` | 8.389 |

认证请求 `/admin/accounts` 是安全的管理员只读业务调用，实际出现 Redis `get` 和 SQL `QueryRow` / `QueryRows`，不是只命中缓存或仅验证 OTLP 端口。登录 Trace 另含 Redis `set` 和 SQL 查询。

原始 Trace：`login-trace.json`、`authenticated-admin-accounts-trace.json`；请求 ID、时间和 HTTP 状态见 `runtime-evidence.json`。保留原有 native WATCH/PubSub/复杂 SQL 事务例外，不宣称这些原生旁路自动有子 Span。

## 日志、指标与可视化

- 目标集群实际 `make observability-status` 成功；完整 `make observability-smoke` 最终退出 **0**。保留最初失败与最终成功的日志。
- 认证请求日志 requestId：`goal0038-d3bb78c2b39847bf9b95b263c3eef80f`，Elasticsearch 和 Kibana 查询均命中。
- Prometheus UP target 精确等于当前 Ready Pod 集合：`app-api-f7f8f4dcd-swf68`，没有把旧 Pod 目标当成通过。
- Grafana Prometheus 数据源健康 OK，Dashboard `awesome-zero-platform` 六面板实际经 Prometheus + Grafana `/api/ds/query` 查询有数据。
- 采样：HTTP 请求率约 0.239/s，5xx 比例 0%，p95 约 0.080s，常驻内存约 98 MiB，WS 活跃数与消息速率为 0，符合测试连接关闭且没有消息流量的实际状态。
- 扫描应用文件日志和 app-api/Filebeat 容器日志：未发现已知凭据、验收 token、Argon2 密码 hash。实际 SQL/Redis Span 属性仅方法/命令名，没有语句参数或私有载荷。没有声称这是对未来所有输入的形式化保密证明。

## 数据与资源保护

- 全部 4 个 PVC UID 与 Spec 保留且 Bound；MySQL、Redis、ES、Prometheus 的 Pod UID、容器状态未变。
- 其他 Deployment/StatefulSet/Service/PVC Spec、ConfigMap 内容和 Secret 内容指纹均与部署前一致；无其他工作负载 rollout。
- MySQL Schema 前后 dump SHA256 完全一致：`4c6d9aaa9b9d4fc61e14b32ab352ff82a772662e4f24f4073a77430570a136a2`。原账号资料及策略版本一致，原 15 表保留。
- Redis 持久化实例没有重启、flush 或删业务键；已有 Session 的跨升级访问已实测，验收后的 Logout 只清理测试会话。
- 最终节点：主节点 working set 4.66 GiB、可用 3.09 GiB；worker working set 2.45 GiB、可用 5.30 GiB（Kubelet 快照，非总 RSS）。两节点 Memory/Disk/PID pressure 均 False。
- `/data` 40G，约 26G 剩余，33% 使用；根盘约 28G 剩余。3 天 ES/Prometheus/MySQL binlog 与 Prometheus 512MiB 保留策略沿用原部署，没有放宽。
- 共享 CPU steal 实测约 42%，仍是服务器性能风险。唯一观察到的新 Warning 是旧 app Pod 关停时健康探针连接拒绝；新 Pod 无重启或 Warning。

## 回滚准备与安全访问

- 旧镜像仍在目标节点 containerd：manifest `sha256:e43911988ddea7e5ed885ee8db8cea0a452b3706998e23802bab0e43384b44d0`。
- 旧镜像导出 `rollback-image.tar` SHA256：`07876bda5f92467ffc529f6900592716660b582f12ae65bae72674a5438f85cb`，最终复核一致。
- 原完整 Deployment：`app-before.json` / `app-rollout-before.json`。镜像回滚补丁 `rollback-image-only-patch.json` 经目标 API server dry-run 通过，未实际回滚；成功部署无需为演练额外重启应用。
- 如后续回归，维护窗口执行：

```bash
kubectl -n awesome-zero-platform patch deployment app-api --type=strategic \
  --patch-file /root/.local/state/awesome-zero-platform/goal0038/rollback-image-only-patch.json
kubectl -n awesome-zero-platform rollout status deployment/app-api --timeout=300s
```

补丁仅恢复原镜像，保持当前 ConfigMap、Secret、Filebeat 和 one/Recreate；没有卷/数据库操作。不要 apply 默认双副本 base。

所有 Service 仍 ClusterIP，测试仅 localhost port-forward，无新增公网端口。安全查看 Jaeger：服务器运行 `kubectl -n awesome-zero-platform port-forward --address 127.0.0.1 svc/jaeger 16686:16686`；客户端通过已有可信 SSH 主机运行 `ssh -N -L 16686:127.0.0.1:16686 <trusted-host>`，再打开 localhost:16686。测试用 port-forward 子进程已结束。

## 自审、交接与限制

本次是 Codex 的环境补充执行，非源码修复。已完成同代理第二遍自审：逐项对照交接文档、受限修改范围、镜像来源、真实 trace 图与时间、Session 生命周期、凭据扫描、数据/配置保留、失败 Job 与恢复、回滚准备和最终 Diff。没有独立审查声明；没有发现需交回 ChatGPT 的业务源码缺陷。

部署验收期间没有修改受 Git 管理的文件。验收后用户明确要求“提交并推送”，本次仅提交本报告及交接文档中的报告链接，不修改源码、测试、Manifest 或已归档 Goal，也不重新定义空白的 current goal。原始诊断、Trace JSON 和回滚材料留在服务器 `/root/.local/state/awesome-zero-platform/goal0038/`（目录权限 0700），不随 Git 发布；本报告不包含凭据值。

本次没有重新执行源码单元/压力/实体游戏客户端测试、观测故障演练或破坏性回滚；这些不在本次补充缺口内。Jaeger 仍为内存存储，重启后 Trace 会消失，所以保留本次 JSON 证据。共享 CPU 与 off-node backup 等既有运维风险未由本次镜像更新解决。

验收完成时间：2026-10-30T04:42:53.435021+00:00。
