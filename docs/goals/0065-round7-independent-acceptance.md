# Goal0065 Round7 独立验收

**FAIL。** 固定候选 `9ab7963dedfc4c194c8b5796db1388d17ffcdf12`。本轮发现 **3项P2、1项P3**；原Round6三个缺陷的精确复现通过，但正常前进、跨控制器会话所有权和最终启动门禁仍有缺陷。发现后继续其余独立场景，未修改产品源码，未部署候选，Goal0065保持 `in_progress`。本轮按新任务明确排除用户手机黑屏调查及手机真机验收。

## 基线与构建

执行者Codex。干净 `main` 上 `git pull --ff-only` 同步到固定候选，独立归档源码到隔离目录，重建所有本地工作区包，确认依赖链接指向本轮源码。154份tracked客户端输入中，153份字节一致，另1份package.json仅被CLI重排格式/键顺序，JSON语义一致；产品源码差异0，见[输入身份校验](evidence/0065-round7/source-identity.json)。遵守 `docs/principles/` 的Go权威、模块边界、UNKNOWN身份、有界资源和真实证据要求；未变更Go、SQL模式、平台接口或客户端产品实现。

- Node `22.23.4`；官方 cocos-cli `f506fa3de403b48b4833334fa48dddd27bb26364`；engine `d0a696cfe63fd8d19c70d7913ba11aa641c2e3c7` / Cocos `4.0.0-beta.0`；仓库Linux 2D engine profile，真实web-desktop导出，严格生成引擎声明检查及8份JS BigInt AST检查通过。
- CLI日志保留未使用的3D内置资源导入器/工具缺失与内部类型工具提示；这些不作为3D能力通过证据。最终2D导出成功，另行严格声明检查通过。没有Wine或其他OS虚拟机。
- Chromium156 headful/Xvfb/SwiftShader，隔离受信TLS证书，真实HTTPS/WSS接现有Go测试服务。本机临时站点30465，不替换现有29457部署；视口模拟不宣称实机。
- [CI](https://github.com/oublie6/awesome-zero-platform/actions/runs/45351843237) 六job成功；[Fair Doudizhu](https://github.com/oublie6/awesome-zero-platform/actions/runs/45351843177) 四job成功；[Cocos](https://github.com/oublie6/awesome-zero-platform/actions/runs/45351843297) `linux_source`成功，`linux_native_export` **skipped**。本地真实导出独立完成，未将skipped写成执行成功。Vue同源码本地 `vue-tsc`/Vite构建通过。
- 独立源码测试 **315/315**：core39、协议50、Cocos181、Vue16、构建脚本29，均无失败/跳过；core、协议、Cocos类型检查及Vue构建通过。

[CI明细](evidence/0065-round7/ci.json)、[检查命令与耗时](evidence/0065-round7/checks.json)、[导出原日志](evidence/0065-round7/native-build.log)、[产物哈希](evidence/0065-round7/artifact-hashes.json)、[CSP/缓存配置摘要](evidence/0065-round7/fixture-proof.json)。测试日志同目录保留。未重复未受影响的Go/MySQL/Redis/密码学压力和崩溃测试，复用历史验收；不把历史证据当作本轮新执行。

## Round6精确复验与受控矩阵

| 项目 | 本轮结果 |
| --- | --- |
| 原N01三个失败单元 | 全PASS：无版本timeout后，同版本旧广播/旧完整响应不能消除屏障。旧notModified拒绝、重复通知在新视图前不累加、重连保留屏障、新hand重置、合法新版本恢复均通过。新增反向时序FAIL，见R7-N01。 |
| 原N02 SDK send累积 | PASS：真实MiniSocketTransport/openSocket/ConnectionManager/Shell，16轮失败/重连后未完成send归零；32容量/第33拒绝、发送期限关闭并结算兄弟等待器、替换、active send期间dispose、同步异常、迟到/重复回调均通过。并发写命令保留UNKNOWN，已确认ACK不被迟到send错误覆盖。 |
| 原N03最终断连 | PASS：两个只读恢复入口各8种时序，共16项；正常、断连、新版本、无版本timeout、房间变化、HandID变化、身份失效、销毁。join拒绝失效；reconnect的null取消结果未误算成功恢复。 |
| 命令关联 | PASS：不一致的envelope/payload ID、错误拒绝不消耗其他请求；迟到/重复ACK无副作用；UNKNOWN原ID精确重试；新成员持久化Join ACK与只读恢复完成契约分开。 |
| 身份HTTP响应体期限 | PASS：真实HTTPS服务器先flush 200头并写JSON前缀，正文停滞；已确认头到达，注入150ms期限仍在正文读取中abort，无身份/令牌保存。非仅模拟fetch头部。 |
| 会话生命周期 | 同控制器延迟保存、慢revoke与新登录、旧验证成功/失败晚于另一控制器写入、快速20轮登录注销、Web旧验证均PASS。跨控制器注销/异步保存失败见R7-N02/N04。 |

独立受控矩阵：同步 **45PASS/3FAIL（48项）**；Controller **16PASS**；SDK **9PASS**；命令 **3PASS**；会话 **7PASS/3FAIL（10项）**。总86项，80PASS/6FAIL；六个失败单元按下述根因归组，不等于六项独立缺陷。同步/SDK记录未处理rejection为0，进程正常退出，SDK关闭后监听器及发送等待器归零。这些是生产类配受控端口，不是物理SDK、设备内存或真实Go认证资格证据。

[同步矩阵](evidence/0065-round7/async-matrix.json)、[SDK矩阵](evidence/0065-round7/send-matrix.json)、[Controller矩阵](evidence/0065-round7/controller-final-close.json)、[命令矩阵](evidence/0065-round7/command-matrix.json)、[会话矩阵](evidence/0065-round7/session-matrix.json)。各同名mjs为独立复现脚本；需在同目录提供固定候选 `input/` 归档、已构建dist和工作区依赖；会话真实HTTPS用操作员自建TLS证书及 `NODE_EXTRA_CA_CERTS`。其中令牌字符串均为合成占位值，不是QA凭证。复现应复制到私有临时目录运行，勿覆盖已提交的证据；runtime脚本还需操作员自有accounts.json、TLS、30465隔离站点和19176专用CDP，ui/hook/cleanup及构建驱动同目录保留，不提交这些私有输入。

## 新缺陷与修复交接

### G65-R7-N01 / P2：迟到timeout将最低版本抬到不存在的未来版本

位置：`clients/fair-doudizhu-cocos/assets/scripts/game/GameClientShell.ts:100–110`，及同步/归档的最小版本校验。

受控精确步骤：已有v3新快照→投递它对应的迟到无版本timeout→客户端要求v4→新建读取返回当前v3仍拒绝。另一单元：v2→timeout→合法v3恢复→同一旧timeout重复到达，版本下限再次升至4。第三单元：普通COMPLETED/v3尚未提升归档时，迟到timeout使同版本合法终局归档也被拒绝。三项均FAIL。

预期：既要拒绝陈旧视图，也不能要求服务端并不存在的未来版本；晚到或重复通知不能令正常牌局永久等待另一次状态变更。实际实现仅忽略已有已验证terminalEvidence的终局，对其他新鲜视图一律 `snapshot.version+1`，无法确认通知是否已被当前版本涵盖。

真实Go精确复验：捕获本局真实自动timeout并暂留，先读取真实最新BIDDING/v2，再回放原通知；客户端将下限设为3，新关联读取 `sync-2` 实际收到Go的BIDDING/v2，仍报 `stale private hand sync response`，WSS保持可用，canSubmit=false。由另一名真实玩家正常Bid1推进Go到v3后，同hand恢复；未修改客户端状态来消除屏障。结果在[运行时记录](evidence/0065-round7/r7-live.json)中单列；只回放从该局真实连接捕获的原通知，不伪造自动操作或成功快照。普通COMPLETED→归档的特定受控单元与实际Go常用终局evidence回退路径区分，不能把全部受控分支都称为真实Go复现。

### G65-R7-N02 / P2：旧的未认证控制器logout认领并清除新账号token

位置：`clients/packages/game-client-core/src/session.ts:89–101`；`clients/game-web/src/auth.ts:121–126`。

步骤：共享真实WebSessionVault/同步Storage；旧控制器restore读取旧token，验证尚未返回；新控制器登录成功并保存新token；再调用旧控制器logout；最后释放旧验证。GameSession因尚无accessToken而重新load共享vault，把新token当作旧token清掉；新控制器仍authenticated但vault为空。WebAuth同类路径还把新token传给revoke适配器，观测 `newTokenRevoked=true`。

预期旧控制器只清理自己持有的会话；compare-and-clear必须比较真正属于旧会话的值，不能比较临时重新load的新值。证据使用实际会话/vault实现、受控认证回调和合成token，未吊销真实其他用户会话，未证明跨用户手牌泄漏。普通真实注销换号控制通过不消除此并发契约缺陷。

### G65-R7-N03 / P2：最终场景await后仍用过期房间/公平阶段作为就绪证明

位置：`clients/fair-doudizhu-cocos/assets/scripts/bootstrap/GameBootstrap.ts:648–664` 及SESSION_ACTIVE前最后一次scene await。

**真实房间关闭复现：** 新建自有空房间，真实授权恢复读取完成；封装原 `waitForWebScene()`，先执行真实原函数并保留其结果，再延迟最终返回。在等待期间用同账号另一个已认证公开HTTP调用执行room.leave，Go返回accepted，随后真实读取明确 `CLOSED / occupied=0 / member=false / version=2`。释放原场景结果后，页面仍发送SESSION_ACTIVE，移除遮罩并显示旧的等待好友房间。无伪造HTTP成功或渲染成功。

**真实发牌跨阶段复现：** 合法FAIRNESS_COMMITTING恢复后同样暂停最终场景返回；三个QA账号真实提交贡献、等待真实信标；两个对照客户端实际取得同hand BIDDING/v1。目标端仍缓存FAIRNESS_COMMITTING、无私有快照，释放等待后仍SESSION_ACTIVE。目标未收到变更/未请求快照，预设的快照丢弃故障**没有实际命中（dropped=0）**；此项证明陈旧的发牌前证明被继续使用，不称作“丢失快照响应”复现，也不把无效的注入计为通过。

最终门禁仅重新读取客户端缓存，并未证明跨越最后await后缓存仍有效。预期房间关闭/成员资格撤销不能继续成功，已经发牌则需要当前私有投影。源Controller的末次检查虽修复原N03，却不等于Bootstrap的最后门禁已具备当前授权/阶段证明。

[房间关闭及生命周期结果](evidence/0065-round7/post-runtime.json)、[发牌对照结果](evidence/0065-round7/predeal-gate.json)、[房间关闭截图](evidence/0065-round7/room-closed-final-gate.png)、[发牌前陈旧界面截图](evidence/0065-round7/predeal-stale-ready.png)。房间关闭脚本同目录保留；需要操作员自有QA账号，创建/关闭的都是测试房间。

### G65-R7-N04 / P3：每实例vault写队列不能保护另一控制器的新保存

位置：`clients/packages/game-client-core/src/session.ts:18–27,79–86`。

在SessionVault允许的异步save端口上延迟旧控制器的save；另一控制器完成新登录并保存新token；dispose旧控制器，再释放旧save。旧save覆盖新token，随后的旧清理把vault清空，新控制器仍认为自己authenticated。期望不同控制器共享vault时仍维护会话所有权。

这是实际GameSession配受控异步vault的扩展契约测试；内置Web/Mini同步存储不直接复现延迟写入，故单列P3，不声称真实浏览器存储或设备内存泄漏。[会话矩阵](evidence/0065-round7/session-matrix.json)记录 `delayedCrossControllerSave` 的布尔观测，不含真实token。

## 补齐启动、缓存和资源检查

- 无房间正常进入、多恢复房间显式选择、同房恢复合并、A→B→A及旧成功/旧失败覆盖保护、最终场景等待中真实WSS断开、恢复中注销/换号/旧窗口消息、快速reload均PASS；房间关闭FAIL见N03。[11个真实场景](evidence/0065-round7/post-runtime.json)，10PASS/1FAIL，未将房间关闭失败作为停止其他独立场景的理由。
- 实际配置加载停滞、隐藏期间失败延后呈现、失败后迟到配置成功不抹掉终态、显式retry、伪造origin/source的BOOT_PAUSE/RESUME、快速重建及加载中关页PASS。[配置](evidence/0065-round7/config-hidden.json)、[边界](evidence/0065-round7/edges.json)。
- 临时站点真实WSS握手502，HTTP认证200但transport不可用时不READY；恢复代理后明确retry成功。[握手](evidence/0065-round7/handshake.json)。只修改自有临时nginx并在finally还原，未动集群或用户转发。
- 用raw CDP真实切换标签页，确认 `document.hidden=true`；隐藏时释放引擎，超过62.5秒无误超时，回前台同iframe恢复“游戏就绪”，真实Cocos输入点击成功。[后台](evidence/0065-round7/raw-background.json)。非只设置JS可见性字段。
- 静态策略：HTML `no-cache,must-revalidate`；公开config/manifest `no-store`；指纹Vue JS一年immutable；未指纹Cocos文件 `max-age=0,must-revalidate`。真实认证和房间响应重复读取均no-store；无SW或CacheAPI条目。[隐私/版本/占用](evidence/0065-round7/privacy-version.json)。
- 独立持久浏览器上下文：首次200，刷新21个304，含真实cc.js；传输16,726,338→18,902 bytes；原生资源就绪约6.046→4.829秒。这是同机同产物的一次冷/热对照，不是P95或手机性能结论。[304明细](evidence/0065-round7/persistent-cache.json)。
- 业务就绪后的真实输入验证：三个隔离视口冷启动1280/760/390约9.404/6.415/6.213秒，各26请求；静态传输各16,725,620 bytes。隔离上下文刷新/新tab约7.924/9.562秒，各26请求、12,641,239 bytes，不能将其误称为上面的持久磁盘缓存结果。未用networkidle作为就绪判定。[阶段、网络与交互](evidence/0065-round7/cache-startup.json)。
- 同账号返回再入：同iframe、同engine，新增资源/WSS/持久化命令0，含实际输入操作约1.318秒。五次进出并GC：JS heap24,565,580→25,762,576 bytes；documents3→3，nodes294→300，listeners101→114。记录有限样本，不宣称证明无内存泄漏或长时稳定；未观察严重退化。
- 资源更新后实际响应体SHA与修改后的文件一致，恢复后SHA与原文件一致，旧缓存未掩盖更新；只在私有fixture修改合法空白并finally恢复。[内容级校验](evidence/0065-round7/version-body.json)。
- Round6已经通过且本轮未改动的R3隐藏几何/三个视口矩阵及动态CSS、R5快照丢失15秒恢复、七项终局HTTP故障/个人验证等历史矩阵按任务复用；本轮对受影响的共享传输、身份和版本链执行真实业务回归，未冒充重新执行全部历史单元。

## 真实业务与环境收尾

三个自有QA账号经候选Cocos4和真实Go完成 **hand-cmd-d5fba24c7705fce31a4b97ecfea2324f / COMPLETED/v62**。真实创建/加入/准备/贡献、信标发牌、自动操作、正常Bid/Play/Pass、非法组合拒绝、BIDDING和PLAYING刷新、WSS断线恢复、准备/贡献/实时命令的UNKNOWN原身份重试均执行。新成员真实Join接受ACK后关闭WSS，已确认加入未变成UNKNOWN。

真实执行bid/play/pass的ACK先到、play/pass的快照先到；bid快照先到仅受控矩阵覆盖，未把完整跨积称为Go运行通过。自动timeout的迟到回放复现N01失败，另一名玩家合法Bid1使v3恢复，然后继续同局完成；[真实运行时10项](evidence/0065-round7/r7-live.json)为9PASS/1FAIL，唯一FAIL归入N01。非法两张不同点数出牌真实收到Go INVALID_REQUEST。三人正确个人验证字符串匹配、错误字符串拒绝；已验证终局归档后回放同一真实旧timeout，v62/最低版本不变、归档仍可用。

客户端存储实测三人均只有 `fair-doudizhu-session`，localStorage为空；不保存私有牌局、贡献或个人验证字符串。主业务pageerror0。[脱敏业务证据](evidence/0065-round7/business-summary.json)不包含私有牌、贡献、字符串、Token或原始私有响应。

**驱动中断透明记录：** 主驱动在保存终局/三人验证证据后以exit143结束，确切原因未证实；进程和原浏览器上下文已确认不存在。没有重开牌局。续测公开读取确认同一原房间已经CLOSED/member=false，清理检查无待退成员；用新上下文完成三账号正常登录/注销/换号，三者Cocos与Go账号ID匹配、旧iframe销毁、注销令牌HTTP401、无旧房间/手牌/公平数据。此处401对应续测新登录的会话，不冒充原已销毁上下文中token的精确续测。连续同一上下文的房间恢复中注销及换号已有post-runtime/privacy-version独立证据。业务流程完成与整体验收FAIL分开记录。

[最终只读审计](evidence/0065-round7/final-audit.json)：六Pod均Ready，Deployment/Pod UID、镜像/重启数、ConfigMap、Service、PVC、StatefulSet与基线一致；两节点Ready。MySQL ABORTED12→14、COMPLETED9→10，未结束牌局0；两次增加的ABORTED是两个自有跨阶段探针的公开业务取消，没有直接删库。业务写入不等于环境零数据变化，未做Go/数据库/Redis/DQ/Secret/PVC或生产配置修改。用户29457转发service保持active。

**未部署候选，无候选回滚动作。** 保留Web Goal0064摘要 `44995360be6e91a0e7e1c355db4e8e9027633d1fc4d9eb59b15b110990af0a0f` 和API Goal0063摘要 `04614d0bc04c1fe37d2cb4467be23fec728cb0b1cfb63c26b1cd3746943ec82f`。临时localhost站点只是验收fixture，不是Kubernetes部署。本轮专用Chromium/CDP19176与 `codex0065-r7-web-9ab7963` 已停止，用户29457服务保留。

## 证据边界、自审与结论

源代码315项成功、CI成功与真实构建不替代行为验收。受控协议/SDK/身份回调、真实HTTPS/WSS/Go业务、桌面视口与设备证据分别列出；没有实机、生产部署或本轮后端压力/进程崩溃通过声明。所有本轮适用独立客户端场景已执行；失败单元保留失败，不降低标准。候选部署及部署后验收因上述P2明确禁止，属于条件未满足，不写PASS。

外部夹具修正单列：首次归档未先构建secure-envelope而静态类型检查找不到包；首次导出漏复制2D profile，停止该次并保留原日志，补齐后重建；新成员受控Join初始夹具包含了非成员不应收到的seats，产品正确拒绝，修正预览DTO及请求观察后通过；跨发牌探针初始判定错误要求必有丢弃响应，修正为同hand的两份真实BIDDING对照及本轮消息后重新执行。均只修外部驱动，不改产品，不把夹具失败算成产品缺陷。

本报告与脱敏证据做第二遍自审：逐项核对固定SHA、执行命令、阶段/版本/ID、真假端口、失败计数、因果与未执行部署条件，检查证据哈希、链接和凭证泄漏；复查最终Git差异只含本报告、证据及current.md状态段。这是报告自审，不是产品独立接受之外的另一轮验收。开发方需要针对N01–N04修复后提交新固定SHA，才能重新接受；Goal0065不关闭，0045/0046不宣告完成。
