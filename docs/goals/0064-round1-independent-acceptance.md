# Goal0064 Round1 独立验收 — FAIL

日期：2026-11-22。实际验收源码：`61429bd67367b47bc7541a29f492a60708344e76`，与参考候选一致；干净 `main` 经 `git pull --ff-only` 确认同步。仅验收和发布报告，未修改业务代码、协议、数据库结构或现有部署。

## 证据与环境

[脱敏运行记录](evidence/0064-round1/result.json)保存阶段、版本、原截止时间、按钮投影、字体测量和 durable 回执。原始私密脚本/日志留在开发机 `/root/goal0064-61429bd`，账号、令牌、TLS 私钥和贡献字符串不提交。

候选使用官方 Linux Cocos CLI/4.0.0-beta.0 引擎真实导出、生成声明严格类型检查及 BigInt AST 检查，再与 Vue3 产物一起提供可信本地 HTTPS/WSS。Chromium156 headless/SwiftShader，三个独立 BrowserContext、三个新普通 QA 账号，实际鼠标/触摸事件操作。前端预览端口30464代理现有测试命名空间 `awesome-zero-game-test` 的真实 Go API、MySQL5.7、Redis、两个 Beanstalk/DQ；不是模拟后端。

源码相对已验收 R9 `9581630` 的 `server`、`clients/packages` 无差异，故复用其基础恢复/故障测试及本候选 CI，不重复全量 Go 测试。候选导出和 Vue 构建独立执行成功；第一次引擎子进程512MiB V8堆不足，串行1024MiB重试成功，不隐瞒失败。CLI未使用的导入工具/生成声明告警由已有原生类型验证步骤处理，未修改产物业务逻辑。

精确 SHA CI：

- [CI](https://github.com/oublie6/awesome-zero-platform/actions/runs/45003628952)：6/6 SUCCESS。
- [Fair Doudizhu](https://github.com/oublie6/awesome-zero-platform/actions/runs/45003628932)：4/4 SUCCESS。
- [Cocos Linux](https://github.com/oublie6/awesome-zero-platform/actions/runs/45003628937)：源码检查 SUCCESS，CI native export **SKIPPED**；真实导出证据来自本地独立运行。
- [Vue3](https://github.com/oublie6/awesome-zero-platform/actions/runs/45003628951)：SUCCESS。

## 结果

| 项目 | 结果与实测范围 |
| --- | --- |
| Vue登录→唯一Cocos入口 | PASS；Vue仅“进入游戏大厅”，三个独立会话均进入真实画布 |
| 建房、加入、准备、房主开始 | PASS；三席与本人准备状态正确，非房主没有开始权限；重复刷新按钮另列缺陷 |
| 随机贡献与发牌 | PASS；三人真实提交、服务器确认 revealed，贡献输入清除，公开信标后进入同一牌局 |
| 三人完整牌局 | PASS；同一 hand 完成 v1→v61：一次自动不叫、59次真实UI操作（一次3分叫地主、58次出牌/过牌），三客户端一致 COMPLETED，获胜座位2，积分-6/12/-6 |
| 排序、选牌、轮次 | PASS于已执行单张出牌/过牌路径；v53三个玩家实际 HandCard 节点与本人完整手牌按强度/花色排序一致；点击选牌后合法出牌推进权威版本。复杂牌型及无效组合提示没有新增独立运行用例，不记PASS |
| 倒计时、刷新恢复 | PASS；三客户端关闭后新会话恢复同一手牌v1，原截止时间始终为1795351435008；真实300秒到期自动提交seat1/score0至v2，未通过改时钟或缩短配置代替 |
| 结算、返回、归档与个人核验 | PASS；三客户端同一v61/摘要，均返回大厅、取最终证据、本人字符串正例匹配/反例拒绝，更多→重连后取得服务端房间状态。未将归档/个人匹配当作独立全量洗牌验算 |
| UNKNOWN精确核对 | PASS；实际丢弃room.ready ACK/变更消息，超时后仅RetryUnknownRoom；恢复消息并点击核对，requestId/clientSeq/expectedVersion/aggregateId/payload完全一致 |
| 退出、恢复、重复点击 | 单次真实触摸退出及授权发现恢复PASS；双击加入FAIL，见N03。另一玩家准备后用旧version退出被DDZ_VERSION_CONFLICT拒绝，重新授权查询后退出成功，不当作服务故障 |
| 布局/操作去重 | FAIL；1280×900、760×390、390×844实测，见N01/N02 |
| 浏览器异常 | 首轮三客户端登录/大厅采集pageerror为0；未连续采集完整牌局的全部console事件，不据此声称全程控制台无异常 |
| 候选Kubernetes部署 | BLOCKED于独立验收FAIL，未构建/替换候选镜像；既有测试服务健康及真实通信通过 |

房间：`room-cmd-bf9e892a998d061443d43401e105deeb`；手牌：`hand-cmd-343e09a067908d8bf49fb1979abd2577`。保留该QA归档和房间；无历史数据清理。

## 集中修复清单

### G64-R1-N01 / P2 — 等待玩家操作重复

复现：三人加入同一房间并准备；查看seat2或seat3。界面同时出现两个“刷新房间状态”。本人已准备、仍等待他人准备时也可进入同一分支。

证据：[非房主截图](evidence/0064-round1/duplicate-refresh.png)，运行记录 `lobby/all-ready`。根因：`LobbyFlow.ts:92` 将`sync-room`作为主操作，`:99`又无条件加入次操作；`GameBootstrap.ts:290`直接拼接两组。需确保最终操作唯一，并增加真实READY非房主/本人已准备场景回归。

### G64-R1-N02 / P2 — 新大厅竖屏文字过小

复现：进入有三席的房间，将外层网页设为390×844。座位、提示和主操作明显过小；该问题出现在大厅而非旧牌桌UNKNOWN布局。

证据：[竖屏大厅](evidence/0064-round1/portrait-lobby.png)，实际投影字体：seat约8.31 CSS px，提示约8.84 CSS px，长房间标题约10.72 CSS px；760×390座位约10.73 CSS px。根因：`GameBootstrap.ts:1509-1564`用逻辑画布字体/三列布局，未保证嵌入式Web画布缩放后的阅读及触摸尺寸；操作字体仍受18–26逻辑单位限制。需按最终CSS投影校准，并重新验证窄横屏/竖屏触摸与文字。

### G64-R1-N03 / P2 — 双击加入成功但界面停留已退出状态

复现：从已结束牌局的房间退出，输入同一房间号，快速双击“加入房间”（两次实际鼠标点击，间隔30ms）。服务端接受一次join，界面roomId仍为空并提示“已退出房间，可重新创建或加入其他房间”；等待25秒不恢复。点击“查询我的房间”后才正确恢复。

证据：[错误界面](evidence/0064-round1/double-join.png)及运行记录`doubleJoin`：leave接受v12，join接受v13/duplicate=false，协调器已有roomId/v13而界面roomId为空；授权发现后同一房间v13/三成员。此前一次双击同样导致等待超时，重新登录发现其实已入房；后以独立双击用例再次确认。

根因：`GameBootstrap.ts:973-995`的join未设置create/leave同类互斥标记，每次点击都递增`recoveryAttempt`；第二次选择使首次已提交join的回执失去当前选择资格，相关superseded异常又被抑制。`GameOnlineController.ts:222-281`也以选择代数隔离异步join。需保留防止陈旧账号/房间写回的保护，同时让同一在途加入被重复点击时不丢失已接受结果。不得通过更换requestId重新提交或弱化UNKNOWN保护修复。

## 环境保全与结论

未部署候选；现有API仍为R9不可变摘要`04614d0b…`，Web仍为既有`990da695…`（完整摘要在JSON）。验收前后六Pod的UID、镜像、Ready、重启次数，以及Deployment和PVC状态一致。数据库从ABORTED6/COMPLETED3变为ABORTED6/COMPLETED4，仅新增本轮QA完整牌局；最终无未结束手牌、截止任务0。没有修改原账号、密钥或历史业务记录。服务器29457转发保持active，HTTPS入口200；已关闭本轮30464预览及专用Chromium。

验收脚本的菜单专用等待曾不适用于牌桌，且一次并发刷新使观察器Frame detached；修正观察器后用新会话恢复同一手牌，原始失败保留在私密日志，未归因为产品或冒充通过。双击加入的接受回执/界面不一致则经独立复现确认为N03。

**Goal0064本轮不满足完成条件：FAIL，交由ChatGPT集中修复N01/N02/N03，再按新固定SHA复验受影响路径及三人回归。** 无效牌型提示、全程console采集及修复后的候选Kubernetes集成仍需补证。真正跨房间历史分页、独立全量洗牌重算、微信/物理设备与生产发布不在本轮已实现范围，不作为伪造PASS或新增功能要求。Goal0045/0046及既有历史结论保持不变。
