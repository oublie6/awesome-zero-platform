# Goal0065 Round13 独立收尾验收

**FAIL：Round12 N01/P2、N02/P3、N03/P3 均复验通过；新增 1 个 P3 存储同步重入问题。** 未修改产品代码，未部署候选，不关闭 Goal0065；其状态保持 `in_progress`，Goal0045/0046 不变。本报告更新原 Round13，不另开 Round14。

固定候选 `eda1a70b1ec504d6588d79203c50d9a08416bd8c`，main；执行日期 2026-11-26 UTC。恢复时 main 为仅含验收文档/证据的 `b35c8596e5effb1ff6106a7ecbb35206975d7797`，干净、`git pull --ff-only` 已同步，clients/server/scripts/deploy 相对候选无差异。实际增量审阅基线为 `3cc38fca62952dfd9832992f5112ee5cd4ee6c68`（任务文字中的该 SHA 少了一段字符，以仓库实际提交为准）。读取 AGENTS、原则、当前 Goal、Round12 报告及本轮产品改动；独立验收限定项目自有隔离环境。

## CI 门禁及原阻塞记录

恢复验收前直接核对固定候选的四项工作流均 **SUCCESS**，才开始本轮测试。见 [ci-success.json](evidence/0065-round13/ci-success.json)。

| 固定候选工作流 | 最终结果 |
| --- | --- |
| [Cocos Linux Native Client](https://github.com/oublie6/awesome-zero-platform/actions/runs/45850121871) | attempt1 SUCCESS；linux_source 成功，linux_native_export 跳过，本地另作实际导出。 |
| [Fair Doudizhu Vue3 Web](https://github.com/oublie6/awesome-zero-platform/actions/runs/45850121819) | attempt1 SUCCESS。 |
| [Fair Doudizhu](https://github.com/oublie6/awesome-zero-platform/actions/runs/45850121767) | attempt1 SUCCESS，4 个 job 成功。 |
| [CI](https://github.com/oublie6/awesome-zero-platform/actions/runs/45850121928) | attempt2 SUCCESS，runtime、secure_envelope、unit、admin_web、integration、report 成功。 |

保留原历史：15:35:25 UTC 原门禁观察时 CI 尚在运行，按指令停止；当时未执行产品测试/构建/运行，也没有产品验收结论。[ci-precondition.json](evidence/0065-round13/ci-precondition.json) 是该时刻快照。随后推送文档提交 b35c859 的时机不当：同分支 CI 的 `cancel-in-progress` 取消了候选原 runtime job。已明确说明并仅重跑被取消的 job（140016373449），形成同一固定 SHA 的 attempt2（[重跑启动历史快照](evidence/0065-round13/ci-retry-history.json)，当时pending，最终结果见ci-success）；不修改代码、不换候选，也没有拿文档提交 CI 冒充候选 CI。用户本次明确恢复后继续全部剩余任务。原 BLOCKED 是历史前置门禁结果，现已解除；当前 FAIL 来自下述实测缺陷。

## Round12 三个缺陷复验

| 原缺陷 | Node 生产模块及实际官方 Creator 产物结果 |
| --- | --- |
| N01/P2 连接清理重入 | **PASS**。原 Manager/Shell 驱动均只创建 initial/latest，过时 superseded 不再创建。Manager 旧 open 取消、最新 open 成功；Shell 返回 `[false,true]` 且选中 latest。扩展 offMessage/offClose/close 三处重入：只在 latest 发送/接收，旧 close 不撤销新连接，新 close 撤销就绪，各连接最终关闭一次。 |
| N02/P3 MiniHttp 注册异常 | **PASS**。部分注册后抛原异常，注销一次、残留监听器0、SDK请求0；注册前/后同步取消均不派发请求。额外验证抛 `undefined` 仍以原值拒绝。结束路径在创建 deadline timer 前返回，未留下请求等待。 |
| N03/P3 清理重入计数 | **PASS**。原三次失败现计数3；扩展 release、onCancel、诊断回调内嵌套重入，均执行全部步骤、计数3，正常业务结果42不变。诊断自身异常隔离，不算额外清理失败。 |

原驱动复用时只调整已改变的期望：旧 Manager open 应拒绝取消，Shell 不应再创建 superseded；没有改变原故障注入。Node 证据 [cleanup-core.json](evidence/0065-round13/cleanup-core.json) 7PASS、shell-reentrant.json、transport-faults.json 11PASS、reentrant-count.json；实际 Creator [native-cleanup.json](evidence/0065-round13/native-cleanup.json) 9PASS。新增同一矩阵在 [reentry-matrix.json](evidence/0065-round13/reentry-matrix.json) 与 [native-matrix.json](evidence/0065-round13/native-matrix.json) 各19PASS/0FAIL。

## G65-R13-N01 / P3：Vue WebAuth 在存储读写同步重入后仍发布过时身份成功

位置：`clients/game-web/src/auth.ts:164` 的 `storage.setItem` 后无所有权复核；`:112` 的最终 `storage.getItem` 返回后也未重新检查 generation/共享 loginEpoch，直接在115—116行写入身份并返回成功。对应七项不变量中的“外部调用后再确认所有权”“过期任务不得覆盖新任务”“成功必须持有有效结果”未满足。

最小复现使用实际生产 `createWebAuth`，只注入合法形状的受控同步 StoragePort 与身份 HTTP：

1. login 已收到有效 token 并完成 `/auth/session` 验证。
2. `storage.setItem` 写入 old 后，同步调用同实例 `dispose()` 或 `logout()`，使 generation 失效并取消旧请求所有者。
3. 存储调用返回，旧 login 仍设置 token/active 并返回 old；两种情形均观察 `authenticated=true`，使刚完成的本地销毁/注销失效。
4. restore 的最终存储读取同步 dispose，同样返回 old 并重新认证；最终读取同步启动同实例或共享存储的另一实例 login 时，过时 restore 仍返回 old 成功，而最新登录结果为 new。共享实例例中旧实例最终 authenticated=false，但错误的旧成功返回仍成立，不夸大为覆盖了新存储。

[vue-reentry.json](evidence/0065-round13/vue-reentry.json) 记录5个反例，全部不满足应返回 null 的期望；[vue-reentry.mjs](evidence/0065-round13/vue-reentry.mjs) 可复现。执行 Node22 `--experimental-strip-types`，直接加载固定候选 Vue 生产源模块，不是复制实现或 mock WebAuth。正向生产 Vue 的25项测试与真实浏览器登录/恢复/注销/换号均通过。

**影响边界：P3，受控存储适配器的同步回调缺口。** 未观察到原生 sessionStorage 在普通读写时自行触发该回调，未声称当前普通浏览器注销已失效、真实 Go 接受了撤销凭据、存在跨账号数据泄漏或手机黑屏原因。原 Round12 的正常 abort 回调重入修复仍通过。此问题仍属于本轮明确要求的存储重入及成功所有权契约，不能把正常路径通过当作专项通过。建议修复时在这些外部存储调用返回后重核本地/共享所有权，并保证过期补偿不删除新凭据；本轮没有实施修复。

## 同步重入检查矩阵及七项不变量

对全部本轮改动及直接关联链逐一检查；foundation/platform 无游戏语义变化，纯 TS 生命周期/传输能力与客户端集成的模块归属保持原样，Go 权威、协议、持久化及部署契约未变。未作无关架构扩展。

| 调用链 | 实际检查、证据与边界 |
| --- | --- |
| ConnectionManager | open 在旧清理前保留 generation，detach 先分离集合/句柄；清理、注册、connect await 后重新核对。两个 listener 注册期间重入后旧 open 拒绝，最新连接仍 ready。JSON.toJSON 同步 close 时发送0。矩阵与原驱动 Node/Creator 均通过。 |
| GameClientShell | 旧连接清理前保留本次所有者，owner() 返回后再次检查；success notify 重入 connect，旧返回 false，新返回 true；commandTransportReady 二次核验、消息/close 代次隔离。close/router 清理错误时3个 pending command 均 UNKNOWN、sync结束、其余清理继续，game-cleanup.json 3PASS。HandStore.setStatus 是本地赋值，不虚构其存在外部订阅回调。 |
| GameOnlineController | authentication attempt 在取消前建立，各外部取消/initialize/auth/connect/readiness 后检查；恢复 pending 仅由仍有效代次发布。201项 Cocos 测试含取消重入、旧 owner、membership/command/recovery 屏障及UNKNOWN；provider-matrix.json 36PASS 覆盖正常/拒绝/迟到/重复与身份置换，实际 Bootstrap 菜单额外复验。 |
| GameSession | 本地与共享 vault 新 scope/epoch 在旧 abandon 前发布；同实例及共享 vault 清理回调启动最新 login，旧 login 无派发且返回 null，最终只保存 latest，Node/Creator 矩阵通过。save/load await 后检查、scope.finish 后验证 grant；logout/dispose 先捕获/失效本地凭据再清理。session-lifecycle.json 9PASS 与 controller-cancel-restore.json 保留 Round10 复验。 |
| OperationScope/Slot | 结束状态与 listener 快照在外部 release 前分离；重复清理幂等、旧 finally 不释放新 slot。cancel/finish/abandon、多个 waiter、业务原值/原错误保留；普通和嵌套计数通过，诊断静态脱敏。 |
| MiniHttp / 相邻 SDK | 注册异常/同步取消/原错误/可用 remover通过；SDK与socket同步 close/error、部分注册、移除异常、迟到回调、多个 send 等待全部结算，transport-faults.json 11PASS。底层拒绝注销时只保证尝试与逻辑无效，不声称物理移除。 |
| Vue WebAuth / Vue App | 正常与 abort 回调触发同/不同实例登录通过；存储读写重入新N01失败。实际Vue unmount，message/storage/visibility注销及诊断均抛异常仍移除root/iframe；手动调用保留handler后 HTTP0、watchdog0，vue-cleanup.json PASS，物理移除受控失败不冒充成功。 |
| 实际 Creator UI/生命周期 | Bootstrap重复/重入销毁、多个组件清理错误仍尝试其余步骤；GameTableView先清原timer/data再销毁节点，view替换不重置同一TurnClock。native-cleanup.json通过；实际取消菜单与旧登录/房间回调 fencing 通过。 |

七项不变量在上述实际检查范围内：资源/操作先占有，外部调用后复核，旧操作不覆盖/释放新操作、不再派发，成功有有效结果，清理错误不改变业务/后续清理。连接、会话及通用清理专项通过；Vue存储外调返回后的复核/成功契约仍FAIL。没有把表格的源码检查冒充所有可能回调组合的穷尽证明。

## 必要回归、构建与真实业务

**321项受影响自动化测试 PASS、0 FAIL、0跳过**：core66、Cocos201、Vue25、构建脚本29；core/Cocos静态类型检查、Vue vue-tsc/Vite通过。命令及退出码见 [checks.py](evidence/0065-round13/checks.py)、[checks.json](evidence/0065-round13/checks.json)，日志保留。

只完成一次官方 Linux Cocos4 Web 导出：Node22.23.4，CLI `f506fa3de403b48b4833334fa48dddd27bb26364`，engine `d0a696cfe63fd8d19c70d7913ba11aa641c2e3c7`，4.0.0-beta.0。CLI报告46秒，最终退出0；保留非致命 polyfill/生成声明 TS2688 提示，随后使用实际生成声明的严格类型检查通过，8份JS AST/原生BigInt审计通过。没有把 beta 导出等同正式发布资格。[native-pipeline.log](evidence/0065-round13/native-pipeline.log)。

实际构建输入来自候选 `git archive`：1553个tracked普通文件逐一比较，除CLI重排Cocos package.json（JSON内容完全相同）外字节一致；按仓库官方脚本将 engine-profile 安装到隔离项目生成配置。源/24个导出文件/30个实际站点文件的路径、大小与SHA256见 [source-artifact-identity.json](evidence/0065-round13/source-artifact-identity.json)。不是旧产物改标。

API未重复Go构建：候选与已验证Round12的 server tree同为 `a8f14fec9d7703c0fe68ac286bbd4aa9081fefbe`，复用二进制 `dc50adc3bdc3fabe04b64e0a66f417df44a1fb56fc35a2f3f225e9cdb714c000`，并读取运行进程 `/proc/.../exe` 重核。未以旧集群API替代匹配隔离API。环境为自有 goal0065r9库、g65r13身份/队列前缀、API18870/HTTPS30471/CDP19181，Chrome156/Xvfb/SwiftShader，临时测试CA真实受信任；无证书绕过、HOME覆盖、Wine或其他OS工具链。

[smoke.json](evidence/0065-round13/smoke.json) **PASS**：真实Vue账号登录、官方产物加载并恢复身份；创建一间新自有房间（正常关联业务命令），网站/大厅切换复用同一iframe，新增静态请求0；关闭实际Socket，观察非就绪后重连恢复同房；正常leave关闭房间；注销iframe0、旧凭据HTTP401；换号后无旧room/private hand/fairness。没有新开牌局。

实际WSS启动失败在7237ms内明确显示失败，没有SESSION_ACTIVE，establishedIdentity/transport均false；旧callback无效，显式重载换frame后恢复。`GAME_READY`只表示引擎/场景，不冒充已认证会话成功。实际HTTPS缓存：HTML重验证、config no-store、hash JS immutable、非hash cc.js条件304/0字节、身份响应no-store，cache-smoke.json PASS。

Round10原取消/立即restore驱动、实际Bootstrap取消菜单SDK/prepare两个阶段均在await旧Promise前启动新任务，旧write0，新slot与业务成功保留（bootstrap-immediate.json 2PASS）；实际standalone取消登录、旧discovery/reconnect回复失效（bootstrap-adjacent.json 3PASS）。微信bind/register/unlink/relink/profile的正常及取消/UNKNOWN边界通过受控端口，不冒充真实平台账号绑定资格。

本轮无牌局命令协议、Go/schema/自动操作实现或缓存策略变更；发送前序列化专项、UNKNOWN清理与正常房间命令未显示契约变化，因此按授权复用 [Round11完整三账号v61牌局](0065-round11-independent-acceptance.md) 的300秒自动操作、原RequestID UNKNOWN恢复、终局归档与个人字符串验证，以及 [Round9](0065-round9-independent-acceptance.md)—[Round12](0065-round12-independent-acceptance.md) 有效SQL、性能、崩溃恢复、缓存/内存证据。不重复完整Go构建/压力/heap/300秒等待，也不将历史完整局称为本轮完整端到端PASS。未调查手机黑屏、未执行手机真机测试。

## 部署、资源及未验证项

全部适用门禁未通过，**条件部署未触发**：未部署候选API或Web，未执行回滚、部署后版本/镜像验收，也未关闭Goal0065。既有API镜像仍为 `goal0063-api@sha256:04614d0bc04c1fe37d2cb4467be23fec728cb0b1cfb63c26b1cd3746943ec82f`，Web仍为 `goal0064-web@sha256:44995360be6e91a0e7e1c355db4e8e9027633d1fc4d9eb59b15b110990af0a0f`，不是本候选已部署。

集群6 Pod、2节点Ready；Deployment/Pod UID、镜像/重启数、ConfigMap、Service、StatefulSet、PVC及Secret hash均未变。集群 ABORTED14/COMPLETED11/未完成0；隔离库 ABORTED21/COMPLETED14/未完成0，与开始一致。只在隔离库创建并正常关闭一间自有房间。[final-audit.json](evidence/0065-round13/final-audit.json)、audit-extra.json。

本轮API、浏览器、临时Nginx及原已停止的4个依赖容器均停止，18870/19181/30471端口关闭。临时CA昵称删除，NSS列表恢复；用户29457转发服务active且HTTPS健康200。[post-cleanup.json](evidence/0065-round13/post-cleanup.json)。隔离工作目录及原始证据私有保留。公开证据采用白名单与已知秘密/JWT/PEM扫描，未发布账号密码、token、密钥、私有手牌或个人字符串。测试/构建日志仅规范化行尾空白，原日志私有保留；[manifest.json](evidence/0065-round13/manifest.json) 固定公开证据哈希。

## 第二遍证据自审与修复交接

重新核对候选CI attempt/SHA、源与运行产物哈希、全部增量及相邻外调顺序、19项矩阵在Node和实际产物的一致性、N01五个反例的成功返回与实际当前身份差异、UNKNOWN与真实冒烟、历史复用界限、资源和脱敏。反例中同实例dispose/logout确实使旧操作失效；共享实例只证明过时成功返回，不扩写成新凭据被覆盖。正常原生存储没有受控同步回调，故评级P3且不声称已发生真实注销事故。此为报告与证据的第二遍自审，不冒充另一独立审阅者。

驱动准备阶段一次全局字符串替换使 native-cleanup.cjs 出现语法错误，未执行浏览器测试；修正驱动后使用同一唯一产物完成9PASS，未重建或改产品。其余驱动均逐项核对结果字段，不以进程exit0代替case通过。最初CI推送导致的取消单列历史，不隐藏为环境自然故障。

交回ChatGPT的唯一新修复项为 G65-R13-N01：补足WebAuth存储返回后的代次/共享所有权检查与确定性回归，保留本报告FAIL及历史。后续仍需同候选完整适用验收及条件部署才能关闭Goal0065。当前仅完成本次独立验收和报告，Goal0045/0046保持原状态。
