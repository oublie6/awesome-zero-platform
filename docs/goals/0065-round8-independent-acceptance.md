# Goal0065 Round8 独立验收

**FAIL：2项P2，候选不得部署。** 固定产品 SHA `9325366e65f792364d5935493f95288764e8fb2a`。原Round7的迟到timeout、跨控制器清理/保存及原最终场景等待复现点通过；扩展异步调用链仍存在旧登录继续连接、最后成员读取跨发牌阶段的缺陷。用户最终指示“不修复，直接写报告”，因此停止追加回归，仅整理证据和收尾。未修改产品代码、仓库测试、协议或数据库模式，未部署候选。Goal0065保持 **in_progress**；0045/0046不关闭。

## 基线、执行与证据身份

执行者Codex；2026-11-24，UTC。干净main经 `git pull --ff-only` 同步到上述固定SHA，审查增量 `4955eeed76306d2f7af7224465427dac135e8560...9325366e65f792364d5935493f95288764e8fb2a`，遵守AGENTS、当前Goal及四项规范原则。产品所属为客户端集成；Go保持状态权威，UNKNOWN保留原请求身份，不改变平台/业务模块边界。

源码归档、工作区依赖及构建位于独立私有目录 `/root/goal0065-round8-9325366`。本次对Cocos、game-web和clients/packages下归档中存在的118份tracked输入逐一复核：117份字节一致，Cocos package.json仅CLI格式/键顺序改变、JSON语义相同，产品差异0。[身份校验](evidence/0065-round8/source-identity.json)；[实际导出/暂存产物逐文件SHA256](evidence/0065-round8/artifact-hashes.json)。报告提交SHA与受测产品SHA分开，不将文档提交冒充重新构建的产品。

Node22.23.4；官方cocos-cli `f506fa3de403b48b4833334fa48dddd27bb26364`；engine `d0a696cfe63fd8d19c70d7913ba11aa641c2e3c7` /4.0.0-beta.0；采用仓库2D engine profile。**仅一次真实Linux web-desktop导出，47秒成功**，严格生成引擎声明检查、8份JS BigInt AST检查通过。原日志保留未使用3D资源导入工具和CLI内部类型工具提示，不据此宣称3D/发布资格。[导出日志](evidence/0065-round8/native-build.log)、[构建驱动](evidence/0065-round8/native-build.cjs)。无Wine或其他OS虚拟机。

Chromium156、headful/Xvfb/SwiftShader，独立可信TLS，真实HTTPS/WSS连接现有Go测试服务。临时站点30465和专用CDP19177；未替换用户29457服务。受控延迟/丢包、真实Go业务和模拟SDK分别记录，不宣称手机或物理微信设备资格；本轮按任务排除手机黑屏调查和手机真机验收。

## 已执行的构建与检查

| 检查 | 结果 |
| --- | --- |
| core类型/测试 | PASS，43测试 |
| 协议类型/测试 | PASS，50测试 |
| Cocos类型/测试 | PASS，184测试 |
| Vue vue-tsc、Vite构建/测试 | PASS，18测试 |
| Linux导出及Web暂存脚本测试 | PASS，29测试 |
| 合计 | **324/324 PASS**，无跳过；不包含独立矩阵 |

[命令、退出码、耗时](evidence/0065-round8/checks.json)、[顺序检查驱动](evidence/0065-round8/checks.py)；同目录各包原日志保留。串行执行构建和重测试，复用依赖缓存，不重复导出同一产物。

固定SHA四项实际CI均SUCCESS：[Vue](https://github.com/oublie6/awesome-zero-platform/actions/runs/45436658246)、[Cocos](https://github.com/oublie6/awesome-zero-platform/actions/runs/45436657988)、[Fair Doudizhu](https://github.com/oublie6/awesome-zero-platform/actions/runs/45436658297)、[CI](https://github.com/oublie6/awesome-zero-platform/actions/runs/45436658451)。Cocos `linux_native_export` **skipped**，仅linux_source执行成功；本轮本地真实导出另行完成。[CI逐job记录](evidence/0065-round8/ci.json)。绿CI不覆盖下述缺陷。

## 两项失败与复现

### G65-R8-N01 / P2：失效登录完成被外层当成有效连接许可

位置：`clients/fair-doudizhu-cocos/assets/scripts/game/GameOnlineController.ts:57` 的 `loginWithPassword` 与79行 `loginAndConnect`，以及GameSession登录代次的丢弃返回契约。

实际生产Controller、Session、Shell和ConnectionManager，配受控身份与Socket端口；两个入口均确定复现：

1. 同一Controller启动旧登录A，延迟身份结果。
2. 新登录B成功，建立连接1；在连接1提交一个尚未确认的命令。
3. 释放旧A成功结果。GameSession正确丢弃过期结果，但其Promise正常返回void。
4. 外层只检查disposed，继续用当前B令牌调用connect，建立连接2并关闭B正在使用的连接1。新命令被结算为 `connection replaced; command outcome unknown`。

两个失败单元均观测 `connectionCount=2`、`newSelectedConnectionClosed=true`、`pendingCommandInvalidated=true`；预期旧任务不产生连接替换或业务副作用。[结果](evidence/0065-round8/login-chain.json)、[确定复现脚本](evidence/0065-round8/login-chain.mjs)。测试没有真实账号凭证。

这是**公开Controller接口并发契约**缺陷；正常Bootstrap通常通过替换/销毁Controller抑制这一调用组合，本轮没有在普通Vue登录按钮上复现同一竞态。不声称错账号登录、私有手牌泄漏或真实用户会话遭吊销。Session层原缺陷通过，并不证明上层await完成的业务意义也正确。

### G65-R8-N02 / P2：最后成员读取期间发牌，旧公平阶段仍足以发布就绪

位置：`clients/fair-doudizhu-cocos/assets/scripts/bootstrap/GameBootstrap.ts:679` 的 `confirmStartupAuthority`，特别是702行最后roomInfo之后仍使用缓存fairnessView；同类窗口在 `game/GameOnlineController.ts:183` 的 `completeActiveRecovery`：phase读取后等待196行最后roomInfo，再用旧phase验证。

**真实Go/Cocos复现：** 同一真实三人局，目标端已读取FAIRNESS_COMMITTING。延迟最终权威成员读取的调用，原有场景渲染已真实完成；三账号提交真实贡献，等待真实信标发牌，对照端确认同hand **BIDDING/v1**。随后放行真正的Go成员读取，room仍是同版本HAND_ACTIVE、同activeHandId、member=true；客户端没有再确认公平阶段，也没有请求私有投影，却发出 **SESSION_ACTIVE**、移除遮罩并显示“游戏就绪”。实际目标仍FAIRNESS_COMMITTING、无快照，`reads=0 / dropped=0`。

此项证明跨最后await的阶段陈旧，**不是“丢失快照响应”复现**：预设快照丢弃故障没有命中，因为客户端根本未发读取。客户端read generation只能检测本地并行读取，room版本不随hand阶段每次推进，不能作为hand阶段新鲜证明。

同局对照端延迟的是原Round7最终scene等待点：释放后新阶段检查识别BIDDING，确实请求私有快照；受控丢弃该响应后15秒失败关闭遮罩，没有发布成功，原复现点PASS。移除故障并显式retry后两个目标均取得同hand私有v1，再继续完成同一局；**手动retry是失败恢复对照，不算自动正确就绪**。[真实对照记录](evidence/0065-round8/startup-gates.json)、[两个真实等待门驱动](evidence/0065-round8/dual-gate.cjs)。

另用真实Controller及受控HTTP推进阶段，在最后成员读取前将FAIRNESS_COMMITTING切到BIDDING、保持房间元数据不变；已有成员 `joinRoom` 和 `reconnectAndResync` 均返回成功，`phaseReads=2 / roomReads=3 / syncRequests=0`，私有投影缺失。两个入口均FAIL，与真实Bootstrap失败归为同一阶段窗口。[结果](evidence/0065-round8/controller-phase-window.json)、[复现脚本](evidence/0065-round8/controller-phase-window.mjs)。未把受控两入口声称为分别完成的真实Go复现。

## 通过的修复与相邻异步链

| 范围与证据 | 本轮实际结果及边界 |
| --- | --- |
| [Shell矩阵](evidence/0065-round8/async-matrix.json) | **52PASS/0FAIL**，实际Shell/Router/HandStore及GameHandActions配受控协议。旧关联、广播、无效notModified不清屏障；通知后新关联完整读取同版本可清，真正较低已知版本仍拒绝；调用者knownVersion不能凭空制造下限；丢包、超时、迟到、合并、重连/替换/销毁、A-B-A/hand切换、终局下限、ACK/UNKNOWN均覆盖。未处理rejection0。 |
| [Session/Vue/Web矩阵](evidence/0065-round8/session-matrix.json) | **18PASS/0FAIL**，原跨控制器logout和延迟save复现通过。同显式backing/key跨Vault协调、不同key独立、慢load/validate/exchange/save/clear、队列失败后可用、dispose/discardLocal所有权通过。真实HTTPS先flush200头、正文停滞，确认头到达后约166ms在正文读取中abort，未保存身份；注入期限150ms，不冒充生产12秒耗时。 |
| [MiniVault](evidence/0065-round8/mini-vault.json) | **3PASS**，真实factory/Vault/Session配受控SDK；相同SDK/key的不同Vault写序、旧恢复/清理和provider discard、同步存储异常后队列可用通过。不是物理微信SDK资格。 |
| [Controller最后完成门禁](evidence/0065-round8/controller-final-close.json) | **16PASS**，join/reconnect各正常+7种失效时序；断连、版本/hand变化、timeout、身份失效、dispose等。reconnect返回null仅算正确取消，不算成功恢复。最后成员读取期间阶段推进仍FAIL见N02。 |
| [真实Vue/Cocos运行](evidence/0065-round8/post-runtime.json) | **11PASS**：正常无房、真实房间关闭跨scene、多个房间选择、真实WSS断开跨scene、旧成功/失败、新错误、A-B-A、同房合并、注销换号与快速reload。含真实API与受控延迟，不将每个HTTP结果都称为未注入故障。 |
| [最后权威读取的扩展运行](evidence/0065-round8/final-startup.json) | **6PASS**：无房变一房失败关闭、多房变空采用最新发现、最终读中真实断连/注销、旧catch/finally不清新恢复任务、选择新房取消旧证明。真实导出Cocos+Go，使用受控可信同源最小host，**不是Vue UI资格**。最后一项只证明取消栅栏，不证明新选择自动重新发布host-ready。 |
| [浏览器双realm](evidence/0065-round8/realm-storage.json) | 分别加载真实Session/WebVault，在真实共享sessionStorage上旧restore/logout不清新值、不撤销新令牌。独立realm队列不能协调任意延迟异步login写入：受控边界确实观察到旧写覆盖存储，明确**不在本轮约定保证内、没有资格通过**。没有把单realm WeakMap效果推广到跨realm。 |

同名mjs/cjs为复现驱动，运行需私有临时目录内提供固定候选input归档及已构建dist、工作区依赖；Session脚本使用Node22 `--experimental-strip-types`及操作员TLS信任配置。浏览器脚本还需操作员自有accounts.json、受信TLS、30465临时站点及19177专用CDP；不要直接在提交的证据目录运行覆盖记录。私有凭证、原始手牌/贡献/个人验证字符串未提交。

## 真实迟到timeout与完整三人业务

[真实运行时11项全部PASS](evidence/0065-round8/r8-live.json)。等待真实Go自动操作期限（未调短服务配置），捕获其原始无版本timeout；另两客户端自动同步到v2。目标先读到实际v2，再释放该迟到原通知，精确建立Round7的先快照后timeout反向时序。目标**自动发起新的关联完整读取，仍为当前v2即恢复**：causalBarrier=false、connected、canSubmit=true、wireCommands=0。通知后未手动刷新、未要求其他玩家推进牌局；500ms后读取计数不再增加。明确执行的有限稳定观察，不宣称长时无重试/无泄漏。

同一三账号真实牌局 **`hand-cmd-52d72ec465f313d1740c5447ec02f7af` / COMPLETED/v61**，房间 `room-cmd-ba1b87237483961b3046050a826d2d99`。真实登录、创建/加入/准备、三人贡献、HPKE/信标、自动操作、正常Bid/Play/Pass和Go拒绝非法两张不同点数组合；ACK先到bid/play/pass、快照先到play/pass；PLAYING刷新、WSS重连、UNKNOWN原RequestID/clientSeq/expectedVersion/payload精确重试。已确认新成员Join ACK后真实关闭WSS，已接受加入没有变为UNKNOWN。同账号返回再进入复用同iframe，保留准备命令UNKNOWN并原身份重试。未重新执行bid快照先到的真实Go交叉单元，相关协议排列见受控矩阵。

三个原浏览器上下文完成终局归档、各自正确字符串匹配和错误字符串拒绝、终局同步、下一局准备菜单无重复。重放本局同一迟到timeout后，已验证归档v61/最低版本61均不变。三个原上下文注销后均iframe0、旧token401；同页换另一账号无旧房间/快照/公平数据，并再次注销。驱动正常 **exit0**，本轮没有Round7的exit143或fresh-context替代尾部。pageerror0；console3条均为注销验证访问已撤销token的预期401，**不写成console0**。

[仅白名单业务元数据](evidence/0065-round8/game-summary.json)、[进程退出](evidence/0065-round8/game-exit.json)、[运行日志](evidence/0065-round8/game.log)。旧驱动字段skipDQ=true仅跳过后续play模块的第二次等待，实际自动timeout已在前置live probes完整执行，不代表跳过自动操作。原始完整结果及含个人字符串的驱动仅私有保留。

## 历史复用、停止点与未验证范围

本轮正常启动、进度事件、iframe复用、换号隔离已有上述真实执行；临时站点使用候选缓存map与精确CSP哈希，[配置摘要](evidence/0065-round8/fixture-proof.json)。但用户要求直接报告时，尚未追加执行本产物的独立冷/热304、内容更新响应体和五次进出内存测量；**这些本轮未验证**，不能把配置哈希当成真实HTTP缓存行为通过。

缓存策略/静态暂存、SDK send实现、历史布局/后台几何与动态CSS、后端压力/崩溃恢复/密码学均不在本轮产品增量中。复用[Round7报告及证据](0065-round7-independent-acceptance.md)中相应已执行部分，以及其明确引用的历史证据；不重跑Go/压力全矩阵，不将旧产物缓存耗时/内存数值移植成本候选测量。历史通过不抵消本轮失败，也不等于未完成的设备/发布资格通过。

另有编译后Bootstrap方法原型的9项受控读取排序初次观察；第二遍证据自审发现旧撤销支路的throw stub可能被产品catch吞掉，已准备加强side-effect计数，但用户指示报告前未完成重跑。**该9项不纳入已验收PASS，不提交为确定通过证据**。同房/同hand并发刷新仍有已列真实运行和源码检查证据，完整强化矩阵待下轮。

测试装置修正均未改产品：共享写队列测试释放旧save前不能先await新login（否则装置死锁）；最小host最初导航health被当成下载，改为实际HTML页面后6项通过；双realm断言由“必须revoke一次”改为“不能revoke新token”（旧restore已放弃所有权，0次是合法）；保留初始私有记录，不把这些装置失败算成产品缺陷。未靠改判新旧版本语义来消除真实较低版本拒绝检查。

## 部署门禁与环境收尾

**未构建候选Web部署镜像、未部署、未触发回滚**：两项P2使条件部署门禁失败。临时nginx仅挂载隔离构建，不是Kubernetes候选上线。现有Web仍Goal0064已验收镜像 `44995360be6e91a0e7e1c355db4e8e9027633d1fc4d9eb59b15b110990af0a0f`；API仍Goal0063镜像 `04614d0bc04c1fe37d2cb4467be23fec728cb0b1cfb63c26b1cd3746943ec82f`。

[收尾审计](evidence/0065-round8/final-audit.json)：deployment、pod UID/镜像/restart、configmap、service、PVC、statefulset与进入前投影一致；6个Pod均Ready、无新增重启。未写Secret。数据库ABORTED14不变、COMPLETED10→11，本轮仅新增上述完成局，未完成hand0。自有测试等待房间通过公开业务调用清理，完整局房间已关闭，未SQL删除业务数据。用户 `goal0063-game-test-forward-v2.service` 实测active，29457保留。

自有CDP浏览器已关闭，临时容器 `codex0065-r8-web-9325366` 已停止；未停止用户转发及其他服务。凭证、TLS私钥及完整原始证据保留在权限受限的私有目录。

## 第二遍证据自审与交接

报告提交前重新核对固定SHA、归档输入、产物哈希、Go实际阶段/版本、请求关联身份、已确认ACK与UNKNOWN、原与新等待窗口及故障是否实际命中；核对真实完整局exit0和原上下文尾部，区分生产类受控端口、可信最小host、普通Vue UI、真实Go与物理设备。发现原型矩阵观察器不足后撤回其确定PASS，不追加产品修复。发布文件按凭证值与JWT/私钥模式扫描，原始手牌和个人字符串不发布。该过程是Codex验收报告的第二遍自审，未冒充另一位独立审查者。

返回ChatGPT处理N01/N02，并补齐本轮明确未执行项后交新的固定候选；本报告不提供代修或替代验收。最终结论 **FAIL**，Goal0065 **in_progress**；此次独立验收报告任务完成，不是产品Goal完成，0045/0046及历史FAIL全部保留。
