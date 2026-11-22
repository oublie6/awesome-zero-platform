# Goal0064 Round2 独立验收与测试部署 — PASS

日期：2026-11-22 UTC。固定源码 **`ea1e1823f35cd17c37c524a378e6fe6e61b3164a`**；clean `main` 经 `git pull --ff-only` 同步，未切换候选。**N01/N02/N03 均关闭，第一阶段 Web/Cocos 玩家流程及 Kubernetes 测试部署通过；Goal0064 满足本阶段完成条件。** 保留 [Round1 FAIL](0064-round1-independent-acceptance.md)，不修改业务源码、协议、数据库结构或历史验收记录。[脱敏证据与截图](evidence/0064-round2/result.json)。

## 固定源码与构建

已阅读 AGENTS、原则、当前 Goal、0064 设计、Round1 和 0063 Round9。相对 Round1，生产改动仅为四个 Cocos TS 文件的菜单去重、布局和操作防重入；后端、共享协议、Vue、构建脚本均未变。复用 Round1/0063 Round9 已通过的基础、恢复、持久化和故障证据，不重跑无关全量 Go 审计。

| 精确 SHA CI | 独立读取结果 |
| --- | --- |
| [CI / 45030052028](https://github.com/oublie6/awesome-zero-platform/actions/runs/45030052028) | 6/6 SUCCESS |
| [Fair Doudizhu / 45030052054](https://github.com/oublie6/awesome-zero-platform/actions/runs/45030052054) | 4/4 SUCCESS |
| [Cocos Linux / 45030052077](https://github.com/oublie6/awesome-zero-platform/actions/runs/45030052077) | linux_source SUCCESS；native_export SKIPPED |
| [Vue3 / 45030052045](https://github.com/oublie6/awesome-zero-platform/actions/runs/45030052045) | SUCCESS |

独立执行一次官方 Linux Cocos4.0.0-beta.0 导出、真实生成声明严格类型检查和 BigInt AST 检查，均 PASS（8 JS 文件）。Vue 复用 Round1 的源码等价产物，未重复构建。独立编译 shell 并串行执行 `lobby-flow.test.mjs`、`menu-pagination.test.mjs`、`online.test.mjs`：**63 PASS / 0 FAIL**。实际产物绑定固定 SHA，未修改生成业务 JS 或用模拟引擎替代。

## 缺陷与相关回归

真实 Chromium156/SwiftShader、三个独立 BrowserContext/普通 QA 账号，可信 HTTPS/WSS，真实鼠标及 CDP 触摸事件，连接既有 Go/MySQL/Redis/双 Beanstalk/DQ。

| 项目 | 结果和必要证据 |
| --- | --- |
| N01 操作去重 | **PASS**：本人已准备/他人未准备、三人准备、非房主等开始、刷新/恢复/更多、终局后再次准备均无重复；仅 seat1 房主显示开始。状态投影和 UI 最终菜单两层去重，权限/版本检查保留 |
| N02 实际布局 | **PASS**：1280×900、760×390、390×844 的座位、完整房间号、提示、主操作和更多菜单均检查。390竖屏座位约14.74 CSS px、提示14.20、主操作16.35、房间号13.67，按钮高41.80 CSS px；760横屏按钮约14.49 px/高27.91 px。完整房间号逐字符核对，文字行高/边界和组件无交叠；竖屏真实触摸展开更多成功。[竖屏](evidence/0064-round2/portrait-more.png)、[窄横屏](evidence/0064-round2/landscape-more.png)。分页组合及所有条目可达由定向测试补证 |
| N03 同目标重复/延迟 Join | **PASS**：真实双击、在服务端已接受且 ACK 被暂扣期间再次点击，均只有一条持久化 Join，释放原回执后正确入房，不再停留“已退出” |
| N03 连续不同目标 | **PASS**：A Join 在途时填写/点击B，UI互斥保留A，不发B命令；A成功回执正确展示A。未将尚未接受的点击当成已成功切房 |
| N03 退出/账号隔离 | **PASS**：暂扣旧账号 Join 的实际授权查询，Vue退出、换账号后只恢复新账号自己的B房间；旧查询不覆盖新身份。原账号重新登录及入房、准备正常 |
| N03 UNKNOWN | **PASS**：丢弃实际 Join ACK 后只显示原请求核对入口；真实重试保持 requestId/clientSeq/expectedVersion/aggregateId/payload 完全一致，没有替换请求 |
| 其他房间写入/刷新竞态 | **PASS**：创建、准备、开始、退出快速重复点击；开局成功 ACK 暂扣期间并发 HTTP 刷新，最终只启动一个新 hand，并采用已接受结果。陈旧版本准备/退出受既有版本冲突拒绝，授权刷新后可继续 |
| 公平 UNKNOWN 隔离 | **PASS**：经显式 CA 校验的 HTTPS 转发取得真实已接受贡献回执后丢弃应答，UI仅提供 RetryUnknownFairness；原身份、原加密载荷精确重试通过，不开放其他房间写操作 |

源码复核确认 `GameBootstrap` 的 owner/epoch/选择代数隔离、控制器 membership/command generation、服务端授权和 UNKNOWN 原请求机制均保留，没有通过移除隔离或生成替代请求解决问题。

## 完整玩家流程

源码预览同一局 `hand-cmd-ec31fd1e88c81cb4091abb146d7a6a59`：创建/加入/准备/开局 → 三人贡献 → 公开信标与发牌 → 叫分、选牌、出牌/过牌 → **COMPLETED/v61**。一次真实300秒自动不叫和59次合法UI命令对应60条 live action；自动动作 expectedVersion1→v2，仅一条，提交晚于原截止时间 **1014ms**。刷新、重新登录和同步恢复同一 hand，原截止时间未重置；修正观察器后新动作自动推进三客户端一致版本。

两个不同点数的非法组合被拒绝，版本不变且不进入 UNKNOWN，牌桌实际显示 `出牌失败：server rejec…`；这是现有通用失败提示。三方最终 handId、版本、摘要及积分一致；均通过最终证据、本人字符串正例匹配/反例拒绝、返回大厅、重连及下一局准备。[源码终局](evidence/0064-round2/source-terminal.png)。所有实际页面创建前安装 pageerror/console 采集：**pageerror0**；console仅1条有意丢弃贡献应答产生的 ERR_FAILED。

结算归档和个人承诺匹配不等同于独立全量洗牌重算。跨房间历史分页等继续按0064既定延期范围处理。

## Kubernetes 部署及部署后验证

仅在源码/真实交互验收全部通过后更新 `kubernetes-admin@network-learning` / `awesome-zero-game-test` 的 **Web静态镜像**：

- 新镜像：`docker.io/awesome-zero-platform/goal0064-web@sha256:44995360be6e91a0e7e1c355db4e8e9027633d1fc4d9eb59b15b110990af0a0f`。
- OCI revision 为固定SHA；沿用已验证的固定 Nginx 基底，按实际内联内容计算 CSP 哈希，保留原安全策略。镜像导入现有调度节点 containerd，未宣称推送公共 registry。
- API 保持已验收 R9 镜像 `sha256:04614d0b…`，其源码与本轮后端相同；MySQL/Redis/DQ、Service、ConfigMap、Secret、PVC和StatefulSet不变。
- 部署前确认无活跃手牌、自有 UNKNOWN 全部核对、deadline0；保存资源、原镜像、完整一致性备份及22表有序行哈希。备份实际解压校验，恢复能力复用同一未变结构的Round9实际恢复证据，没有声称本轮另做恢复。原Web `sha256:990da695…`仍在containerd，可执行静态Recreate回滚；本轮未触发回滚。
- 运行中 Pod 的 digest 与 **30/30 文件哈希**全部匹配候选；切换后、业务复测前，**22/22表行数及字节哈希相同**。仅Web Pod更换，其他Pod UID/重启次数不变。旧端口转发随旧Pod退出短暂TLS EOF，原systemd unit自动重连后可信HTTPS200，未关闭证书校验。

部署后通过实际 `https://127.0.0.1:29457` 再执行三个独立客户端的新局 `hand-cmd-60367433d0ba585fead96560976da25b`：三人登录、开局、贡献、发牌、非法组合拒绝、中途刷新及原deadline恢复、叫分/出牌/过牌、**COMPLETED/v60**、三方归档/个人字符串正反核验和下一局准备，全部PASS。全程 **pageerror0 / console异常0**，覆盖真实生产式 CSP、HTTPS/WSS 与 Go 依赖通信。[部署后终局](evidence/0064-round2/deployed-terminal.png)。未改API/DQ，因此复用本轮源码运行的真实300秒自动动作及Round9故障证明，不重复等待所有自动动作。

最终六Pod/两节点Ready，无未结束手牌、deadline0，API无错误级日志或panic/OOM/隔离标记。测试Pod工作集MiB约API19.5、MySQL204.9、Redis9.0、Web11.5、两队列各0.5–0.6，低于既有限制；不是长期容量资格。两个队列最终 ready/delayed/buried/reserved 均为0，终态延迟票据由原应用自然退休，未清队列。7个历史会话值及原全部Secret字段与Round9一致，TTL未延长。原ABORTED6/COMPLETED4变为ABORTED9/COMPLETED6：三个观察器未完成贡献的自有QA局自然终止，两个自有完整牌局保留。部署后正常业务复测推进的是本轮QA房间/序列和新增牌局，不将其伪报为全库完全不变。

## 观察器纠正与交付边界

保留私密原始失败及补跑记录：陈旧version、隐藏输入null和阶段专属菜单的观察假设已纠正；贡献初次误按WSS注入而非HTTP，三个QA局超时归档；ACK观察钩子漏了“已配置holdId”条件而拦截广播，实际收到的帧定位后修正，再以真实同步及新动作验证三方自动更新；MySQL哈希读取改用原始字节；队列观察器改用Beanstalk长度帧。均未改产品或注入成功状态。

**操作日志事件：** Playwright `route.fetch` 的TLS异常细节曾在工具日志打印一个本轮新QA访问令牌；核对归属后仅撤销该会话，原7会话未动。后续使用显式信任测试CA、失败不打印请求头的HTTPS转发。令牌、贡献、私钥、备份及完整原始日志均未提交；未宣称可删除既有工具日志。

已第二遍核对固定源码、CI跳过项、真实导出、63测试、请求身份/单命令、角色门禁、实际字体/触摸、两局终态/个人证明、故障观察器纠正、镜像文件、回滚条件、数据/会话/密钥和最终服务。专用浏览器及30464预览已关闭，29457用户入口和测试服务继续运行。**Round2 PASS，N01/N02/N03 CLOSED，Goal0064第一阶段 completed。** 微信/物理设备、生产发布、跨节点容灾、独立全量公平重算及历史分页不在本轮资格范围；Goal0045/0046保持in_progress。
