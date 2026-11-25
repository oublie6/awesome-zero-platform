# Goal0065 Round9 独立功能验收

**FAIL：新增1项P2，候选不部署。** 固定产品 SHA `e9762f566a1c03bef99f42630f0d4384f2538443`，增量基线 `35c76bf8d6ab4b158d911cd68e643f3f2fed7a89`。Round8两个原始问题通过本轮复验；相邻Cocos解绑/换绑调用链仍在SessionGrant失效后继续发送身份请求。发现后继续完成其他功能检查，未修改产品代码、仓库测试、协议或schema。Goal0065保持 **in_progress**，0045/0046及历史报告不变。

执行者Codex，2026-11-25 UTC。用户将任务更新为独立功能验收后，沿用同一候选已完成的本轮实际证据，没有重建相同产物。本报告区分真实Go/Cocos、生产类受控端口、历史复用和未验证观察项；不调查手机黑屏，不进行手机真机测试。

## 基线、隔离及产物身份

干净main通过 `git pull --ff-only` 同步到固定候选，读取AGENTS、当前Goal、四项规范原则、Round8报告、关联架构契约及增量。客户端所有权属于共享TS会话/斗地主Controller和Cocos调用方；联合查询属于斗地主业务模块及其MySQL适配器，账号私有通知位于app-api组合/传输层。未增加平台对游戏的依赖、跨模块表访问或数据库结构变更；生产查询使用go-zero `QueryRowCtx`。

候选归档于权限受限的 `/root/goal0065-round9-e9762f5/input`，工作区包重新构建、依赖链接指向候选。1275份tracked输入全部核对：1274份字节一致；Cocos package.json仅CLI重排格式/键顺序，JSON语义一致，产品差异0。Go1.25.8单次构建的API运行文件SHA256为 `dc50adc3bdc3fabe04b64e0a66f417df44a1fb56fc35a2f3f225e9cdb714c000`，重启前后及停止前 `/proc/<pid>/exe` 实际文件均匹配。[源码/二进制身份](evidence/0065-round9/source-identity.json)、[实际运行及收尾](evidence/0065-round9/owned-cleanup.json)、[逐文件产物哈希](evidence/0065-round9/artifact-hashes.json)。归档构建没有内嵌Git revision，本报告通过源码逐文件校验、构建命令和实际二进制哈希关联SHA，不伪称内嵌版本证明。

**联合运行使用候选API，不使用集群旧API冒充。** 对现有测试库做只读一致备份，在本机隔离MySQL恢复为 `goal0065r9`；候选API仅监听18869，独立Redis会话前缀及DQ tube，保留300秒业务期限。临时候选Web30469仅代理该API，受信TLS/WSS、精确CSP哈希及候选缓存map。原集群API/Web、schema、Secret、PVC及29457转发未改。[隔离记录](evidence/0065-round9/clone-created.json)、[临时站点摘要](evidence/0065-round9/fixture-proof.json)。备份、凭证、私钥、原始手牌/个人字符串不提交。

Node22.23.4；官方cocos-cli `f506fa3de403b48b4833334fa48dddd27bb26364`、engine `d0a696cfe63fd8d19c70d7913ba11aa641c2e3c7` /4.0.0-beta.0，仓库2D profile。**仅一次真实Linux Web导出**，CLI导出阶段47秒；含工具初始化/包构建/严格声明和AST门禁的完整驱动171秒。严格生成引擎声明、8份JS BigInt AST检查通过。[原日志](evidence/0065-round9/native-build.log)、[顺序构建驱动结果](evidence/0065-round9/pipeline.json)。保留未使用3D导入工具和CLI内部tsc提示，不据此宣称3D/物理设备/发布资格。浏览器为Chromium156、headful/Xvfb/SwiftShader。

## 已执行检查

| 检查 | 本轮结果 |
| --- | --- |
| 受影响客户端包 | **330/330 PASS**：core44、协议51、Cocos188、Vue18、构建脚本29；无跳过。core/协议/Cocos类型、Vue vue-tsc/Vite均通过。 |
| 受影响Go单元 | application、mysqlstore、app-api bootstrap、doudizhuapi四包通过；低并发 `-p 1 -parallel 1`。 |
| 真实MySQL集成 | stock发现/联合查询两项及独立一致性/开销一项通过，无跳过；仅隔离库。 |
| 自动发牌失败次序 | 独立Go overlay三项通过：持久写错误、业务拒绝、生命周期跟踪失败均不发布dealt；前两项重试成功只通知一次，再执行不重复。生产类配受控故障端口，不冒充真实MySQL故障。 |

[命令/耗时](evidence/0065-round9/checks.json)、[Go单元日志](evidence/0065-round9/go-unit-pipeline.log)、[MySQL命令](evidence/0065-round9/mysql-authority-status.json)、[MySQL原日志](evidence/0065-round9/mysql-authority.log)、[通知故障日志](evidence/0065-round9/dealt-notification.log)。独立Go脚本通过 `go test -overlay` 加入临时测试文件，主仓库及候选归档产品文件未改。

固定SHA四项CI均SUCCESS：[CI](https://github.com/oublie6/awesome-zero-platform/actions/runs/45519080990)、[Fair Doudizhu](https://github.com/oublie6/awesome-zero-platform/actions/runs/45519080768)、[Vue](https://github.com/oublie6/awesome-zero-platform/actions/runs/45519080934)、[Cocos](https://github.com/oublie6/awesome-zero-platform/actions/runs/45519080831)。Cocos `linux_source`成功，`linux_native_export` **skipped**；本地真实导出另行执行。[逐job记录](evidence/0065-round9/ci.json)。绿CI不覆盖下述功能缺陷。

## 新问题 G65-R9-N01 / P2：Cocos身份操作绕过已失效的SessionGrant

位置：`clients/fair-doudizhu-cocos/assets/scripts/bootstrap/GameBootstrap.ts:896` 的relinkWeChat、952行unlinkWeChat，以及648行currentAccount。两个身份操作只检查 `session.authenticated`；await后的currentAccount只比较Controller引用、页面代次和场景有效性，均不消费授权grant。它们直接使用 `platform.http` 和 `session.token`，绕过Controller API的grant门禁。

使用**真实编译后的Bootstrap、Controller和GameSession**，独立接收对象及受控同backing/key Vault、SDK/身份HTTP端口，确定复现：

1. A完成登录，B在共享存储作用域完成较新登录。A的grant=false、commandTransportReady=false，但legacy `session.authenticated=true`。
2. 调用A的解绑：仍发出 `/auth/wechat/unlink`，携带旧token；调用换绑：仍发出prepare及 `/auth/wechat/relink`，后者携带旧token。
3. 扩展真实await窗口：A在有效grant下启动换绑，等待acquireCode；B登录使A失效；释放旧SDK结果，A仍继续上述两个HTTP步骤。最新任务优先规则未传递到此调用链。

预期：已经失去操作所有权的任务不能再发起后续身份业务；实际Controller连接/普通业务API已正确拒绝，而同一身份的Cocos直接调用仍继续。有效grant的解绑正控制确实发出一次预期请求，排除观察器把合法路径全部阻断的假通过。旧清理没有删除B的token，这项所有权清理正控制通过。

[初始两项复现](evidence/0065-round9/provider-grant.json)、[迟到SDK复现及有效grant对照](evidence/0065-round9/provider-grant-extra.json)、[驱动](evidence/0065-round9/provider-grant.cjs)。三个失败单元归为**一个P2**。这是失效任务继续执行业务的客户端功能缺陷；身份HTTP在受控端口终止，没有修改真实微信映射，也没有证明Go越权成功、普通Web按钮竞态或物理微信设备行为。修复应交回ChatGPT，本轮不代修。

## Round8复验与统一恢复

**N01 PASS。** 原密码/平台代码两个独立复现均保持连接数1，较新连接不关闭、待确认命令不失效；旧任务返回false。[原始复验](evidence/0065-round9/login-chain.json)。扩展[身份矩阵](evidence/0065-round9/auth-matrix-final.json)31PASS：三个入口交叉、旧成功/失败、握手尚未完成时新任务成功/失败、注销/dispose/discardLocal、建立身份后拒绝直接换号、连接注册/成功通知中关闭、共享存储grant失效后普通HTTP和业务门禁。没有未处理rejection。较新恢复验证失败合法返回false，不要求抛错；首次观察器误要求error，仅修正该单元并局部复测，其余30个通过单元复用，见[复测](evidence/0065-round9/auth-matrix-corrected.json)。

**N02 PASS。** 在候选Go/Cocos真实局中，延迟最终roomInfo调用，先真实完成三人贡献和公开信标发牌，再放行真正的Go房间读取。目标的dealt提示及私有广播被受控丢弃，使最终联合证明承担识别职责；没有伪造成功响应。房间版本始终7，证明从FAIRNESS_COMMITTING/durable1推进到BIDDING/durable11，对照私有投影是live1。目标收到新联合证明后发出关联私有读取；继续丢弃其回应，真实15秒期限后SESSION_FAILED，遮罩保留，**没有SESSION_ACTIVE**。移除故障明确retry后取得同hand私有v1，继续原局。该retry只算恢复对照，不算自动正确启动。[原始等待门证据](evidence/0065-round9/startup-gates.json)、[驱动](evidence/0065-round9/dual-gate.cjs)。等待点在最终调用发出前，非把已经完成的SQL读取解释为永久状态租约。

[联合恢复受控矩阵](evidence/0065-round9/authority-matrix.json)20PASS：已有成员join/reconnect两个原窗口均须关联读取，4次总room读/1次私有读完成；直接confirm最多2次room读/1次私有读。覆盖私有丢包、错ID/hand、拒绝、低live版本、缺失/非法证明、durable回退/同版本冲突、同步后关房/换hand/换连接/失去grant/命令变化/取消选择。受控期限50ms，独立于上面的真实15秒验证。durable9/live1可正常完成，未把durable版本当live下限。

[真实Vue/Cocos生命周期](evidence/0065-round9/post-runtime.json)11PASS：正常无房、最后场景等待时真实关房/断连、多房选择、旧成功/失败与新失败、A-B-A、同房合并、注销换号/旧消息及快速reload。[最终读取扩展](evidence/0065-round9/final-startup.json)6PASS：无房变一房、多房变空、最终读中断连/注销、旧finally不清新任务、选新房取消旧证明；此组为真实Cocos+Go配受控可信最小host，不是Vue UI资格，最后一项只证明取消栅栏。

上轮未完成的强化[编译后Bootstrap原型矩阵](evidence/0065-round9/bootstrap-order.json)9PASS；旧撤销支路使用真实side-effect计数，不再以可被catch吞掉的throw stub作为证明。房间/公平阶段的同目标并行读取、跨读取代次和旧账号回调均通过。这组是实际方法配受控HTTP/展示端口，不冒充Go或触屏实测。

## 通知、完整业务及兼容性

正常自动发牌对照端收到真实 `hand.dealt`，丢弃其无关联私有广播，随后**自动发起一次关联读取取得BIDDING/v1，无手动RefreshFairness**，版本下限0。[实际自动发牌](evidence/0065-round9/automatic-deal.json)。三个实际账号对应私有位置1/2/3，目标端故障观察只保留位置元数据，不发布他人牌面。通知只在写入及跟踪成功后发布的调用次序由源码、stock/独立故障单元及真实运行分别证明；不将跟踪失败后的最终送达当作已经实测。

[生产Shell通知矩阵](evidence/0065-round9/dealt-matrix.json)9PASS：通知/快照先后、重复、迟到、通知丢失、缺快照后关联恢复、合成durable999不抬live下限、旧hand拒绝及新hand清理。协议排列是受控端口；真实乱序/UNKNOWN/timeout见[本局11项运行检查](evidence/0065-round9/r8-live.json)，文件名沿用旧驱动但source为本轮SHA，11项全PASS。真实300秒自动操作后，先取得当前v2再释放原始迟到timeout，自动新关联读取仍以当前v2恢复，wireCommands0，无需其他玩家推进；500ms有限观察中未持续重试。

候选API与Web完成真实三人牌局 **`hand-cmd-60438a7cd76436804fd6bd7cc84399f5` / COMPLETED/v61**。贡献、公开信标、自动发牌/自动操作、正常Bid/Play/Pass、Go拒绝非法组合、PLAYING刷新、重连、准备/实时命令UNKNOWN原ID/clientSeq/version/payload重试、ACK先到与快照先到均执行。已确认新成员Join ACK后真实断开WSS，持久化接受结果未误成UNKNOWN。三个原上下文完成归档及个人字符串正/反验证、下一局准备菜单检查、注销旧token401/iframe0，同页换号无旧房间/快照/公平数据。正常exit0，pageerror0；console3条均为验证已撤销token的预期401，不写成console0。[白名单业务记录](evidence/0065-round9/game-summary.json)、[退出](evidence/0065-round9/game-exit.json)、[日志](evidence/0065-round9/game.log)。legacy skipDQ字段只避免play模块第二次等待，未跳过实际自动操作。

本局还实际重启同一候选API；三个客户端恢复相同hand、live版本、私有牌面及原turnDeadline，未重发deal或重置期限。[进程记录](evidence/0065-round9/api-restart.json)及业务记录中的三项restart断言。它是进程正常重启检查，不扩张成新一轮SIGKILL/存储崩溃资格。

旧客户端兼容采用**此前已验收Goal0064 Web实际产物**，不重新构建：28文件字节不变，仅两份公开配置改为隔离端点30470；同候选18869 API下真实登录/WSS、恢复带新证明的活跃准备房间成功。[旧产物身份](evidence/0065-round9/legacy-identity.json)、[兼容功能原始通过项](evidence/0065-round9/legacy-next-hand-second-observer.json)。另一个实际候选Cocos在同房第一局终止后，经正常“更多→重连”恢复、进入第二hand的FAIRNESS_COMMITTING，私有快照为空，没有沿用上一局；见[下一局原始通过项](evidence/0065-round9/legacy-next-hand.json)。仅准备阶段切换，不冒充第二局完整玩法。

## 查询效率与缓存

生产Store实际执行计划：room、hand、actor sequence三者均 `PRIMARY / const / rows1`。120次真实InnoDB事务同步更新三个投影版本，3个读者累计480次调用生产LoadRoomAuthority，零撕裂观察；不是用mock代替事务。Store计时包含SQL及JSON解码；测量行JSON401/1342 bytes，合成版本元组用于一致性检查，不宣称等同所有真实大手牌数据。整个HTTP还包含身份校验及可能的历史归档鉴别，不是“每次请求只有一条SQL”。[原日志及计划](evidence/0065-round9/mysql-authority.log)、[独立驱动](evidence/0065-round9/authority-independent_test.go)。

| 测量（同机有限样本） | 并发 / 样本 | P50 / P95 / P99，毫秒 |
| --- | --- | --- |
| 生产Store联合读取 | 1 /100 | 2.202 /3.269 /6.267 |
| 生产Store联合读取 | 4 /400 | 2.311 /33.839 /73.439 |
| 实际候选HTTP、真实准备局，含认证与联合查询 | 1 /20 | 13.832 /50.177 /50.177 |
| 同上 | 4 /80 | 30.929 /84.373 /109.877 |

[HTTP观察](evidence/0065-round9/http-authority-cost.json)。并发尾延迟明显变大，记录开销，不宣称达成长期容量或命令提交性能目标；样本量、状态和共享主机负载均限制外推。

本产物独立持久上下文冷/热加载：16,772,833→18,902传输bytes，原生资源阶段7.373→5.193秒，真实cc.js由200变304。[304及字节原始记录](evidence/0065-round9/persistent-cache.json)。这是资源就绪的单次对照，不含完整业务登录，不是手机/P95结论。

HTML重验证、公开配置no-store、指纹资源immutable、无指纹Cocos重验证、认证/房间重复响应no-store；无SW/CacheAPI。资源合法空白更新后响应体SHA/ETag变化，恢复后body SHA回到原值，仅私有临时站点改动并finally还原。同账号5次返回再入复用iframe；GC后heap24,653,392→25,709,536 bytes，documents3→3、nodes299→305、listeners101→114。有限增长样本不证明无泄漏。[头部/存储/内存](evidence/0065-round9/privacy-version.json)、[内容级更新](evidence/0065-round9/version-body.json)。

## 保证边界、观察器自审与历史复用

[Shell同步52PASS](evidence/0065-round9/async-matrix.json)、[Session/Vue/Web18PASS](evidence/0065-round9/session-matrix.json)、[MiniVault3PASS](evidence/0065-round9/mini-vault.json)均使用本轮生产类配受控端口。真实HTTPS头先到、正文停滞的注入期限检查也通过。真实浏览器双realm同步compare-clear保护新值；独立realm任意异步登录写入仍不由单realm队列保证，[边界记录](evidence/0065-round9/realm-storage.json)，不将其宣称为已保护或新增产品缺陷。

第二遍自审保留以下**未验证观察项**，不隐藏原始FAIL字段，也不计入新产品缺陷：

- 额外的旧Round7最终scene延迟对照：原生60秒终态错误已出现，观察器释放门后立即被现有错误唤醒，随后重试销毁旧任务，没有等待其续体完成；reads0不能证明产品遗漏同步。原始case的pass=false不作为P2，也不宣称此额外点独立通过。真正Round8 N02最终成员查询有BIDDING证明、关联请求、15秒SESSION_FAILED及正常retry证据，结论独立成立。
- 下一局切换本身已通过；额外“回放上一局取消通知”未取得原始hand.cancel报文，观察器captured断言失败，故该特定真实取消通知回放未验证。手牌切换后的旧dealt通知已由受控生产Shell矩阵覆盖，不混写成实际Go报文回放通过。
- 旧客户端驱动先前过早读取尚未到达的fairness、以及误假设贡献菜单始终提供RefreshRoom；修正等待条件/使用实际“更多→重连”后分别取得兼容及下一局功能证据。为这些受控准备局共关闭5个隔离hand，未隐藏为只有一条新数据库记录。身份恢复失败观察器修正、provider首次选择到重载中scene的装置失败同样未算成产品问题。

[观察项裁定](evidence/0065-round9/observer-adjudication.json)与原始JSON同时保留。复用[Round8报告](0065-round8-independent-acceptance.md)及其引用的未受影响历史布局、SDK send、密码学、Go压力/崩溃恢复证据；本轮针对已修改的查询/自动发牌/身份调用链新增验证。不把历史压力、物理微信或手机验收冒充本轮执行。并未重新执行所有后端矩阵或追加第二次相同构建。

## 部署门禁、收尾及交接

**不部署候选，未构建部署镜像，也未触发回滚。** N01使功能门禁FAIL；API→Web上线及Web→API回滚、部署后业务/缓存门禁均条件不成立，未执行。隔离二进制和临时静态nginx运行不是候选Kubernetes上线。原Web仍Goal0064镜像 `44995360be6e91a0e7e1c355db4e8e9027633d1fc4d9eb59b15b110990af0a0f`，API仍Goal0063镜像 `04614d0bc04c1fe37d2cb4467be23fec728cb0b1cfb63c26b1cd3746943ec82f`。

[集群审计](evidence/0065-round9/final-audit.json)、[Secret/节点/转发](evidence/0065-round9/protection-audit.json)：6个Pod均Ready、UID/镜像/restart与进入前投影一致，两个节点Ready，deployment、configmap、service、PVC、statefulset及Secret值保持原状；集群ABORTED14/COMPLETED11、未完成hand0。用户29457转发active，其他基础设施未修改。

隔离库完成局11→12、ABORTED14→19，含上述一局完整玩法及5个业务清理的准备局，未完成hand0。自有房间经公开业务调用关闭，没有用SQL删除真实业务来掩盖结果。候选API、专用19179浏览器、两个临时Web容器及本轮启动的隔离MySQL/Redis/两个队列容器已停止；保留私有数据库/备份供复现，未停止用户服务或其他容器。[自有收尾](evidence/0065-round9/owned-cleanup.json)。

提交前自审检查源码/运行身份、SQL读点、durable/live版本分离、grant失效和调用方完成结果、ACK/UNKNOWN身份、故障实际命中、观察器等待完成、原上下文终局/退出、敏感值及文件范围。原始完整手牌、个人字符串、token、私钥和备份不发布；同目录SHA256清单校验证据。本过程是Codex验收报告的第二遍自审，不冒充另一位审查者。

结论 **FAIL，返回G65-R9-N01/P2给ChatGPT修复并提交新候选后复验**。本次独立功能验收报告任务完成，不等于产品Goal完成；Goal0065维持in_progress，0045/0046不关闭。
