# Goal0065 Round14 Vue 存储所有权独立验收

**FAIL：原 G65-R13-N01/P3 的五个反例全部通过；新发现1个P3，存储清理异常会阻断注销撤销或覆盖恢复结果。** 本轮未修改产品代码，条件部署未触发，Goal0065保持 `in_progress`；Goal0045/0046不变。这里完成的是独立验收与报告，不是产品Goal关闭。

固定候选 main `747597181a8497547e4c3b591e73efcd84d1c5e1`；审阅增量 `415468daf2072de3f2a8aff0d3e5529a5f0623be...747597181a8497547e4c3b591e73efcd84d1c5e1`。2026-11-27 UTC执行，干净main经 `git pull --ff-only` 同步。已核对AGENTS、项目原则、当前状态、Goal0065要求、Round13报告与完整四文件增量；测试均在项目自有隔离环境。没有扩展至无关架构、手机黑屏或真机资格调查。

## 固定候选 CI、源码与复用产物

三项适用工作流均为固定SHA、attempt1 **SUCCESS**，所有job成功，详见 [ci.json](evidence/0065-round14/ci.json)：[Vue](https://github.com/oublie6/awesome-zero-platform/actions/runs/45856578888)、[Fair Doudizhu](https://github.com/oublie6/awesome-zero-platform/actions/runs/45856578702)、[CI](https://github.com/oublie6/awesome-zero-platform/actions/runs/45856578738)。按本轮授权复用Round13的Cocos工作流SUCCESS与实际官方本地导出；不冒称当前SHA新执行了Cocos CI或native export。

本轮只改变Vue auth.ts、其测试及两份Goal文档。对Round13产品 eda1a70 至候选的 Cocos源/配置、client packages/游戏协议、server、scripts、deploy及workflows执行完整范围 `git diff --quiet`，均无差异。直接复用Round13唯一官方Cocos4 Web产物：4.0.0-beta.0，CLI f506fa3de403b48b4833334fa48dddd27bb26364、engine d0a696cfe63fd8d19c70d7913ba11aa641c2e3c7；24个导出文件逐个匹配Round13已提交SHA256，见 [artifact-reuse.json](evidence/0065-round14/artifact-reuse.json)。本轮Cocos导出0次。

API继续复用二进制 SHA256 `dc50adc3bdc3fabe04b64e0a66f417df44a1fb56fc35a2f3f225e9cdb714c000`，server tree `a8f14fec9d7703c0fe68ac286bbd4aa9081fefbe`，重核实际运行 `/proc/.../exe` 同hash。没有重建Go或把旧集群API当作匹配候选API。Vue以候选git archive执行 `npm test` 与 `npm run build`（含vue-tsc --noEmit），**33个测试通过、0失败、0跳过**；包括新增8组测试的25个具体分支，均实际循环执行。33是runner测试数，不把内部25分支再次相加。日志及命令见 [checks.json](evidence/0065-round14/checks.json)、vue-tests.log、vue-build.log。

重新组装本轮Vue正式构建与上述Cocos产物，应用原公开配置、CSP hashes及缓存映射。候选archive的1617个普通tracked文件全部字节一致；实际站点30文件的大小/哈希见 [source-site-identity.json](evidence/0065-round14/source-site-identity.json)。隔离环境：Node22.23.4，Chrome156/Xvfb/SwiftShader，自有goal0065r9库、g65r14身份/队列前缀，API18870/HTTPS30471/CDP19181；临时测试CA真实受信任，无证书绕过、HOME覆盖或其他OS工具链。

## 原五个反例复验：PASS

直接复用Round13 `vue-reentry.mjs`，仅替换候选路径/源标记，加载实际生产WebAuth TS模块（Node strip-types），不是复制实现。

| 原反例 | 本轮结果 |
| --- | --- |
| login写入时dispose | 返回null，account null、authenticated false，未提交token清除。 |
| login写入时logout | 返回null，不重新认证，未提交token清除；对确切old凭据发出logout。 |
| restore最终读取时dispose | 返回null，身份无效；保留原已提交存储凭据，符合dispose不等同服务端注销的契约。 |
| restore最终读取时同实例新login | 旧restore返回null，最终只有new身份和存储。 |
| restore最终读取时共享实例新login | 旧restore返回null，旧实例无效，new存储/新实例身份由新增分支测试确认。 |

证据 [vue-reentry.json](evidence/0065-round14/vue-reentry.json)，5PASS/0FAIL。没有将“保留已提交token”误判为dispose失败，也没有把同realm协调解释成跨realm原子存储。

## 全部存储调用及所有权矩阵

Vue集成拥有本地身份生命周期，Go仍是身份和权限权威；没有新后端接口、数据库表、公共storage key、业务规则或常驻轮询。WeakMap每个StoragePort只保留当前登录/临时写入，正常路径HTTP数量不变。取消已派发业务不被解释成服务端回滚。

| 生产调用点 | 实际核验及结论 |
| --- | --- |
| auth.ts:87 currentIdentity.getItem | 捕获generation、activeEpoch、token、identity；读前/后再次核对，读异常fail closed，读结果不能返回另一个任务的身份。两个getter × dispose/logout/同实例/共享实例新登录8分支通过；独立读异常控制通过，故障结束后原已提交身份仍可正常读取。 |
| :125 restore首次getItem | current同时要求本地/共享epoch匹配、无pendingLogin及临时write；读后再核。首次读重入不派发old验证，读异常原样拒绝，之后新登录可成功。 |
| :133 restore最终getItem | 验证await后及外读后重新检查，存储匹配才提交身份。最终读重入旧返回null；一次读取异常可fail closed并清本地凭据。 |
| :190 login.setItem | 先发布精确write lease，再外调，返回后检查代次/epoch/abort；独立测试写入前/后 × dispose/logout/restore/同实例新login/共享实例新login共10分支通过，旧返回null，不误删new。 |
| :192 login提交确认getItem | 保存后再读确认token及current，才提交active/ownedToken；读取异常不发表身份，保留原异常，清除部分写入并释放lease。 |
| :75 clearStored.getItem | 读前、读后都要求调用者仍拥有清理决策，结果相同才remove；恢复失败、注销、临时write补偿使用各自所有者。新任务在比较读取期间开始则旧操作不删除新凭据。**异常隔离仅在login补偿调用者具备，restore/logout仍缺失，见N01。** |
| :76 clearStored.removeItem | 调用前完成精确token比较和所有权核验；适配器内部任意变更/跨realm原子CAS不在承诺内。remove前/后同步启动共享login只撤销old、新身份保留；**清理抛错会逃出restore/logout，见N01。** |

临时write在提交、取消、部分写失败、补偿异常及诊断异常后均释放精确lease；同步restore在未提交write期间被阻断，不能提前认领。写入失败+补偿/console.warn同时抛错时，原始write错误对象仍保留，后续登录成功。恢复失败先清本地owned/active再外部清理，新操作不会被旧catch之后重置。logout先捕获已验证临时write或自己拥有的token，不重新认领别人的存储值；比较读取重入时仍只撤销捕获old。

新增8组25分支均通过；独立相邻专项 [storage-audit.json](evidence/0065-round14/storage-audit.json) **26场景：20PASS/6FAIL**，6个失败为同一清理异常根因（logout与restore各3种故障位置），不是6个独立产品缺陷。原驱动与该独立驱动、源码全7个存储调用点共同构成证据，没有用作者测试通过代替独立检查。

## G65-R14-N01 / P3：存储清理异常阻断服务端注销，并覆盖恢复失败结果

位置：auth.ts:73—76的clearStored未隔离存储异常；:217—219的logout仅finally取消旧attempt，异常会跳过后续 `/auth/logout`；:138—144的restore在catch内再调用可能抛错的clearStored，不能返回其正常失败值null。

Node最小复现：成功login old后，在注销比较getItem、removeItem删除前或删除后注入同一个清理异常。三个场景均本地authenticated=false，但logout拒绝清理异常，实际HTTP撤销调用0。即使remove已经删除本地token再抛错，服务端撤销仍被跳过。对已过期凭据的restore，服务端验证401后在对应三个清理位置注入异常，三个场景均由应返回null变成拒绝该清理异常；pending状态释放，不扩大为新任务已被破坏。

**实际构建产物与真实API也复现logout路径**：[browser-storage-fault.json](evidence/0065-round14/browser-storage-fault.json)。普通Vue登录成功后，刻意让当前realm的Storage.removeItem对SESSION_KEY抛错，点击实际退出按钮：Vue显示“已清理浏览器会话，但服务端注销状态需要确认”；logout请求0、存储仍含原凭据、以原凭据查询真实 `/auth/session` 仍HTTP200。没有iframe参与或另一路Cocos注销掩盖结果。结束后恢复原生方法并显式调用真实注销清理，该反例不修改产品源或构建产物。

这是**新发现的受控清理异常边界，不宣称由本次修复新引入**；也不是原五个同步重入缺陷回归。P3依据是明确的清理/业务顺序与失败结果契约缺口。本轮没有观察到未注入的原生sessionStorage自行抛错、正常浏览器注销失败或Go越权；App确实提示服务端状态待确认，未虚构成界面报告服务端注销成功。存储拒绝删除时不能保证物理清除，但不应因此跳过独立的捕获凭据撤销。建议ChatGPT分别处理本地清理与服务端业务结果，并保留故障证据；本轮未修复。

## 正向业务回归及历史证据

真实候选网站+匹配API [smoke.json](evidence/0065-round14/smoke.json) **PASS**：登录并加载官方游戏、创建一间新自有房间、网站/游戏切换复用同一iframe且新增静态请求0、关闭实际Socket后观察非就绪并重连恢复同房、正常离房关闭、注销iframe0且旧凭据401、换号后无旧room/private hand/fairness。

[vue-lifecycle.json](evidence/0065-round14/vue-lifecycle.json) 3PASS：真实子iframe写入共享sessionStorage触发storage事件，旧frame销毁、旧回调无效、replacement保留；没有擅自撤销另一身份；刷新重新挂载后只恢复new账号及新游戏实例。共享存储置换本身不等同Go注销，旧凭据HTTP200在该控制中是预期边界，最后通过隔离前缀清理遗留测试会话。

[vue-cleanup.json](evidence/0065-round14/vue-cleanup.json) PASS：实际Vue根unmount，三个原生remove与诊断抛错仍尝试全部步骤、root和iframe移除，旧回调HTTP0/watchdog0；底层故意拒绝移除不声称已物理注销。普通重新挂载由上述真实刷新验证。[browser-negative.json](evidence/0065-round14/browser-negative.json) 3PASS：真实API拒绝保存的无效凭据并清除、错误密码401且未认证/不存token、随后有效登录成功。取消pending登录后重新登录/restore、共享存储竞争、超时及迟到完成由本轮Vue全量测试覆盖，没有杜撰不存在的取消按钮。

[cache-smoke.json](evidence/0065-round14/cache-smoke.json) PASS：HTML重验证、config/manifest no-store、hash JS immutable、未指纹Cocos cc.js条件304/0字节、认证no-store。实际加载/身份正确来自真实Go响应，不拿GAME_READY替代SESSION_ACTIVE。

按授权复用 [Round13](0065-round13-independent-acceptance.md) 的Cocos、Node/Creator各19项连接/清理矩阵、原Round12三缺陷复验和组件计时器证据；复用 [Round11](0065-round11-independent-acceptance.md) 完整三账号v61、300秒自动操作、UNKNOWN精确重试、终局/个人字符串，以及Round9—13有效数据库/性能/崩溃恢复/缓存内存证据。对应实现和契约本轮无差异；未重新执行这些未受影响组件、官方Cocos导出、Go构建、完整牌局或压力/内存矩阵，不把复用称为本轮新执行。

## 条件部署、资源及未验证事项

全部适用门禁未通过，**未构建/部署候选镜像，未执行API→Web rollout或Web→API回滚，部署后验收未执行，Goal0065不关闭**。原API镜像仍为goal0063-api@sha256:04614d0bc04c1fe37d2cb4467be23fec728cb0b1cfb63c26b1cd3746943ec82f，原Web仍为goal0064-web@sha256:44995360be6e91a0e7e1c355db4e8e9027633d1fc4d9eb59b15b110990af0a0f，不冒称本候选已上线。

集群6 Pod/2节点Ready；Deployment、Pod UID/镜像/重启数、ConfigMap、Service、StatefulSet、PVC与Secret hash均未变。集群ABORTED14/COMPLETED11/未完成0；隔离库ABORTED21/COMPLETED14/未完成0与开始一致。仅隔离库创建并正常关闭一间房。见preflight-summary.json、[final-audit.json](evidence/0065-round14/final-audit.json)、audit-extra.json。

本轮自有API、浏览器、Nginx与原已停止的4个依赖容器已停止，18870/19181/30471端口关闭，临时CA昵称删除、NSS证书列表恢复。仅删除本轮隔离Redis g65r14*残留1个键（不含其他前缀），不影响用户/集群存储；用户29457服务active、HTTPS健康200。见owned-auth-cleanup.json、owned-cleanup.json及 [post-cleanup.json](evidence/0065-round14/post-cleanup.json)。私有源副本、产物及原始日志保留；公开证据白名单与已知秘密/JWT/PEM扫描通过，无密码、token、密钥、手牌或验证字符串。

未验证：修复后的新候选、故障条件下刷新恢复的最终行为、候选实际集群部署及部署后业务/镜像验收、移动真机与平台资格。普通原生存储及普通刷新已通过，不推断受控故障组合穷尽。

## 第二遍证据自审与交接

重新核对完整增量、所有存储调用及临时lease生命周期、原五个反例、8组25分支、26独立场景的每项结果、真实Go注销路径、源码/产物一致性、CI SHA、复用界限、集群及清理结果。清理异常与重入失权分开判定；N01的一条根因、六个Node观察及一个真实API故障注入不重复计缺陷数。正常API拒绝、正常dispose保留已提交token、共享身份替换及物理清理被拒绝分别按其契约记录。

两处观察器限制保留原JSON：首次故障浏览器已取得旧凭据HTTP200/注销0的直接证据，但额外刷新后等待大厅超时；移除该未能验证的扩展断言，再从同一唯一Vue产物复验最小注销路径成功复现，不宣称故障刷新已验证。首次无效登录等待指定中文文案超时；复验按HTTP4xx+可见错误+未认证/无存储契约检查，实际401及正常重登录通过，不宣称已定位首轮文案超时原因。两者均未更改产品代码，不把观察器超时另计产品缺陷。

此为报告与证据的第二遍自审，不冒称另一位独立审阅者。交回ChatGPT处理G65-R14-N01/P3；修复后仍须固定候选适用门禁及成功条件部署，才可关闭Goal0065。Goal0045/0046状态保持原样。
