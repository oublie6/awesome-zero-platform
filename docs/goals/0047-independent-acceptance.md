# Goal 0047 — Fair Doudizhu 最新版本独立验收

日期：2026-11-07；执行者：Codex。只执行独立验收，未修改产品、正式测试、Schema、部署或历史报告。

## 1. 最终结论

**FAIL。** 随机贡献、服务器权威存储、各阶段正常恢复及终局证明在下述证据范围通过；最新修复的两个原始反例通过，但另一个现有成员恢复入口未纳入最新选择围栏，真实迟到响应仍能覆盖较新的房间选择：**G47-N01 / P2**。

这不是服务器丢失牌局或配置缺失。服务器数据和授权在本次反例中保持正确，失败发生在客户端采用服务器响应的顺序。不能据此关闭完整可靠恢复要求或E03。Goal0045/0046继续 `in_progress`；本报告交付不表示整个产品完成。

## 2. 候选、验收基线与CI

初始为干净main；执行 `git pull --ff-only`，由上轮报告 `880938696c525cf2cfb8395ebabf3fd55e2c8336` 同步至远端最新产品 **`cfe8804280c4c5d6adbd4715ac1ce5e47279b369`**。全部本轮执行记录绑定此SHA；报告前 `git ls-remote origin refs/heads/main` 再次确认远端相同。

**不存在可确认的“整个Fair Doudizhu产品完整验收PASS版本”。** [Round17](0046-round17-independent-acceptance.md)的产品 `5412c2a981d067e840be4e76369ecd1d0d92405b` 是最近一个相关生命周期检查点PASS，其内部完成和外部资格仍BLOCKED；报告提交为 `31f9f9d27af28ecbb405667fcc678e7298ac5c9a`。不能将其包装成完整产品基线。

因此分层比较：以Round17已通过检查点作较早参考，核对到当前的实际差异；以[Round18](0046-round18-independent-acceptance.md) `895cf44a2b1540e05ee218752bf5997374c14f90`复用新贡献设计已通过的独立证据；以[Round19](0046-round19-independent-acceptance.md)产品 `610c90a0fd6c9827156cafe202658ab6c202e9fc`及其报告 `8809386`作最新缺陷回归基线。后二者历史总体FAIL保持不变。

已通过GitHub连接独立读取精确候选的runs和全部jobs，没有重跑整套CI：

| 工作流 | 结果 |
|---|---|
| [CI 42381327097](https://github.com/oublie6/awesome-zero-platform/actions/runs/42381327097) | SUCCESS，unit/integration/admin_web/secure_envelope/runtime/report 6/6 |
| [Fair Doudizhu 42381327133](https://github.com/oublie6/awesome-zero-platform/actions/runs/42381327133) | SUCCESS，application/client_foundation/manifest_client/mysql_integration 4/4 |
| [Cocos Linux 42381327147](https://github.com/oublie6/awesome-zero-platform/actions/runs/42381327147) | 总体SUCCESS，linux_source SUCCESS；**linux_native_export SKIPPED** |

CI通过是独立证据之一，不替代业务验收或原生包运行。

## 3. 实际差异、架构与执行环境

Round17产品到当前为44文件、2560行新增/171行删除，包含玩家生成随机材料的托管新模式、账号房间发现、未知公平模式隔离及本次恢复修复，也包含中间验收文档。各段由对应独立报告和本轮差异核对衔接，不把历史执行说成本轮重跑。

本次相对Round19报告仅7文件、305行新增/55行删除：Bootstrap、OnlineController、RoomCoordinator及三份测试/current状态。检查了全部本次差异及相关调用链：恢复generation、同目标Promise合并、最终成员重读、失效locator撤销、未决命令保护、UI错误传播；发现遗漏的join成员恢复路径，见G47-N01。

当前server tree与Round19完全相同：`06209f0fc5a8f9bb8a50682c69cc6f195f354e18`。因此复用Round19从候选编译的API二进制，并校验原provenance/二进制摘要；**没有声称重新编译当前后端**。相对Round18服务器只增加房间发现及其路由/测试，贡献事务、规则、随机洗牌和runtime恢复实现未变。`platform/foundation/database/.github`相对Round17无差异。

模块仍为业务Domain/Application/基础设施与app-api组合；没有新增跨模块数据库访问、域层SDK依赖、通用框架或部署单元。恢复修复未改变外部wire/schema；新增发现契约来自上轮并沿用认证actor作用域。原SQL事务和基础设施适配复用，不以本轮为由重构。

实际环境：Linux、Node22.23.4、现有自有MySQL5.7.44/Redis8.8.0隔离库、稳定测试保护根、loopback HTTPS/WSS、真实drand Quicknet。顺序执行，GOMAXPROCS=2，未下载工具或运行Wine。复用Round19四个真实隔离账号，重新通过Go认证，不改生产配置、不重做微信SDK换证。

当前客户端源码经本轮构建；Bootstrap用当前TS转译，实际renderScreen菜单和处理函数执行，只有Cocos基础节点/几何布局替代。协议、controller、加密、HTTP/WSS、Go规则与MySQL均真实。不是原生触摸、手机输入或实际换设备证据。

私有证据：`E=/root/.config/awesome-zero-platform/acceptance-0047-cfe8804`。`runs.jsonl`记录argv/cwd/SHA/UTC/退出码；`ci.json`、`reused-evidence.json`、`storage-regression.json`、`provenance.json`、`completion-audit.json`和`sha256.json`索引本轮及复用依据。账号、token、phrase/random、加密审计明文和密钥不提交。测试服务已停止、卷及证据保留，其他服务未改。

## 4. 历史问题修复确认

| 对象 | 当前独立判断 |
|---|---|
| R19-N01原A/B恢复反例 | PASS：旧A HTTP晚到不再覆盖B；scene及RoomCoordinator保持B，0个房间变更命令 |
| R19-N01同目标重复连接 | PASS：两次调用共享同一scene Promise；初始连接加一次恢复共2条socket，0个重复room命令 |
| R19-N02原关闭反例 | PASS：旧WAITING/v1晚到时，新的CLOSED/v2/nonmember查询使恢复拒绝，未采用旧房间、未报成功 |
| R19-N02已采用后失效 | PASS：前两次授权读正常，扣留第三次refresh结果；另一controller合法离房关闭后，最终重读撤销scene及controller locator，不报成功 |
| R18-N01正常账号冷发现 | PASS：本轮八进程链路各阶段均仅凭账号发现、恢复；完整恢复竞态资格仍受G47-N01阻断 |
| R18-N02模式交叉污染 | PASS，保持关闭：重新执行真实旧H1→H2模式延迟及旧回调反例，未知模式菜单隔离、旧四操作拒绝、合法新贡献成功 |
| R14/R15/R16生命周期、早到HTTP/ACK与UNKNOWN保护 | 保留已通过历史范围；当前71个shell测试及受影响正常链路回归通过；不声称重跑全部历史实际15秒超时/ACK矩阵 |
| 旧服务端生命周期、请求身份、Journal、fencing和稳定根问题 | 按Round10/11/13/18/19已通过且未变实现复用；不重新开放已修复原问题，也不将延期根迁移视为通过 |

R19-N01的原反例已修复，但“所有成员恢复入口均由最新选择拥有”的完整关闭资格仍不授予，G47-N01是关联残留，不把同一问题重复计数。R19-N02本次原反例及最终重读扩展通过。没有证据表明R18-N02再次出现。

## 5. 随机贡献、公平证明与服务器权威

**随机贡献机制：PASS（既定托管信任模型内）。** 当前submitCustodied从真实安全随机源生成32字节，将玩家原字符串与随机数一起HPKE加密上传，绑定actor、hand、seat、requestId、sequence、version、key和时效。服务器在同一事务中加密原完整材料并写贡献、Hand、receipt和Outbox。没有改用服务器生成的玩家随机数。

Round18实际原文/随机数与AES-GCM记录逐项对比、14类真实绑定/密文负例、7个应用故障和4个真实SQL事务故障证据保留。本轮检查这些日志及验证脚本的SHA256与原manifest完全相同，并核对相关实现未变，见reused-evidence.json；没有只复用开发方结论。

Domain每个seat只接受一次，事务锁住Hand并校验版本；同ID用原receipt，改材料/跨身份重放不能成为另一有效贡献。洗牌输入按固定seat排列，使用hand、server seed、三人digest及不可变future drand round/digest的规范编码，不按请求到达顺序洗牌。三人贡献必须先于信标截止完成。本轮重验已接受贡献丢ACK后的新进程恢复及拒绝重生成，无第二条本人记录。

本轮新对局完成后，三个本人proof都重新取回并本地复算：正确字符串通过、错误字符串/错误HandID拒绝。独立验收Go进程对**本轮新归档**运行已有验证程序：比较三个proof与归档承诺/digest，验证固定公钥的真实drand BLS及round，重建确定性洗牌/发牌，重放叫牌/出牌/结算。篡改deck、贡献、beacon round/digest、winner均拒绝；winner负例重算外层record digest，未仅依赖checksum。`fresh-proof-material`及`final-proof` exit0。

该程序使用仓库公开验证算法，在独立进程执行，不宣称独立实现了另一套密码学算法。当前UI只提供本人字符串复算与归档摘要，完整BLS/牌序/过程验证由工具执行；不冒称UI内置整套证明。

服务器接收时知道玩家完整材料。已接受目标是未来公开信标绑定和可重放证明，**不是服务器永远不知道贡献、无法修改未见证承诺或无法选择性终止**。管理员有根/数据库权限的恶意行为不由当前模型证明排除；更强外部见证未实现，也不是本次新增配置问题。

**服务器权威及无牌局持久化：PASS。** 客户端不持久化hand/room locator、随机贡献、恢复kit或Journal/Checkpoint；普通认证token与界面/UNKNOWN请求的临时内存另论。每个冷恢复进程从空roomID/handID、无旧secret/pending开始，只认证后调用recoverMyRooms；验收文件中的期望ID仅用于事后断言，不传入恢复入口。随机数和phrase不参与恢复，服务器不要求再次提交已接受材料。客户端展示状态不构成另一套游戏规则或权威存储；G47-N01是采用顺序错误。

## 6. 实际恢复、持久化和一致性

本轮新hand：`hand-cmd-f3483caab6e4a51ac22a9a2c185151c5`。八个phase为八个独立Node进程，全部exit0：

| 阶段 | 本轮实际结果 |
|---|---|
| setup | room.create已接受后丢ACK，原客户端UNKNOWN后退出 |
| lobby | 新进程仅认证恢复唯一等待房间、本人座位/版本/sequence，无重复create；三人入座准备 |
| ready | 三个新controller恢复READY，正常开新托管局；第一人贡献接受后丢ACK并退出 |
| partial | 恢复同hand和已贡献状态，原记录完全一致；不重生成第一人材料，其他两人继续。发牌前0次private sync |
| beacon | 三人重新恢复WAITING_PUBLIC_BEACON，真实信标到达后发牌；无提前私牌请求 |
| bidding | 新进程恢复三人原private view、版本/席位，合法Bid(3)进入PLAYING |
| playing | 实际API SIGKILL/原根重启后再仅凭账号恢复；三人牌序/轮次/版本与重启前完全一致，WSS重连保持状态，继续打完 |
| terminal | 新进程发现lastHand并恢复COMPLETED归档，相同版本/digest；本人贡献可读，正常离房后发现列表不再返回该房间 |

每个进程退出会关闭其全部客户端连接，因此包含单人贡献后离线和三人全部离线后再恢复。不是保留旧controller假装冷启动。本轮实时接受59条Bid/Play/Pass，finalVersion60，真实MySQL checkpoint为33；贡献记录恰好3条，第一人原密文及身份绑定记录未变。终局后另进程完整证明验证通过。

出牌重启为真实进程故障。其他部分贡献/等待信标/发牌/叫牌/checkpoint后/结算后重启复用Round18六次真实进程重启的未变后端证据；持久化中断复用7个应用和4个真实SQL故障案例，后者含真实COMMIT完成但返回丢失。事务端口故障不等于MySQL引擎崩溃或磁盘断电；本轮未执行这两种测试，不声称验证了主存储损坏后的恢复。

另以真实延迟HTTP检查账号更换、scene销毁、发现后房间先关闭再选择，均通过。新选择与旧成员join的交叉路径失败，见下节。正常恢复与安全授权通过不抵消该失败。

## 7. 缺陷 G47-N01 / P2 — 旧成员join恢复越过最新选择围栏

位置：`clients/fair-doudizhu-cocos/assets/scripts/game/GameOnlineController.ts`的joinRoom（约157–182行）；关联Bootstrap.joinRoom（约388–403行）。新reconnectAndResync/performRecovery检查recoveryGeneration，但joinRoom仅在开始增加该generation，await roomInfo返回后没有验证其操作是否仍拥有最新选择，直接confirm/restore/useAuthoritativeHand。scene的join处理也没有共享恢复选择token。

最小实际复现：

1. 同一个真实账号拥有两个合法WAITING房间A、B，列表要求明确选择。
2. 当前controller发起已有成员的joinRoom(A)，服务器真实返回A/v1/member=true；只扣留HTTP结果交付。
3. 当前scene执行resumeAuthorizedRoom(B)并完成全部重读校验，scene与RoomCoordinator均为B/v1。
4. 释放旧A响应，joinRoom不报错，却将RoomCoordinator绑定回A/v1；scene仍显示B。整个过程0个room变更命令，未伪造服务端数据。

本次A=`room-cmd-3048fa74a0f13c6805729e211f3dcac4`，B=`room-cmd-05da9fd4c75ce79e99455085a1b5934d`。证据：`race-stale-member-join.log` exit1及`race-stale-member-join.json`，实际error=null、sceneRoom=B、controllerRoom=A。这是读请求，没有游戏Command RequestID；两房间HandID均为空，不虚构已经发生私牌跨局泄漏。

影响：最新选择失去控制器所有权，界面与实际命令目标可能不一致；正常UI的加入入口调用同一方法，源码显示其迟到完成还会采用旧room。已实测的是controller与scene不一致，没有声称执行了错误房间的后续变更、绕过服务器授权或丢失随机材料。按P2记录，不升级P1。

最小修复建议：将已有成员join的恢复分支与显式恢复纳入同一个操作所有权/最终授权机制，并让scene加入入口共享最新选择围栏；在await后及采用房间/hand前检查。旧请求不得覆盖较新目标或报成功。继续保护UNKNOWN原命令，不用清pending或本地保存牌局数据修复。补此交错反例及旧响应遇关闭的针对性检查即可，不需重写公平协议或后端恢复。

这是本轮确认的**一个关联残留P2**，不是将R19-N01、G47-N01重复计为两个，也不把CI成功视为覆盖此入口。验收方未修产品；确认一次后继续其他验收，没有反复尝试同一失败。

## 8. 需求覆盖和证据来源

| 要求 | 判断 | 证据及限制 |
|---|---|---|
| 玩家字符串+32字节安全随机上传 | PASS | 当前源码/本轮真实三人提交；原文与随机来源细查复用R18 |
| 服务端完整加密保存 | PASS | 本轮3条记录/原密文不变；R18完整AAD/AEAD/原文审计 |
| 确定性洗牌、真实验证材料 | PASS | 本轮新归档独立进程、真实drand BLS、牌序与过程重放 |
| 本人字符串独立核验、篡改发现 | PASS | 当前TS正/负例及本轮证明变异；非UI全证明承诺 |
| 重放、身份/seat/时效绑定 | PASS | 未变R18真实14类负例，当前源码/CI及丢ACK恢复 |
| 提交顺序与并发原子性 | PASS | 固定seat规范输入、Hand锁/版本/唯一约束；R18事务故障证据复用 |
| 贡献保密与终局揭示权限 | PASS | 未变R18终局前/非参与者/过期token/actor伪造/密文翻转负例；托管服务器本身知道材料 |
| 无客户端权威备份 | PASS | 当前存储调用审计及八次无locator/秘密的新进程恢复 |
| 服务端不依赖重新贡献恢复 | PASS | 接受后丢ACK进程退出；原记录/状态恢复、重复生成被拒 |
| 正常断线、单人/全员离线 | PASS | 本轮新进程链路及实际WSS重连 |
| 进程重启、持久化重加载 | PASS | 本轮真实SIGKILL；其他边界复用R18未变后端证据 |
| 发牌前、发牌、叫牌、出牌、结算 | PASS | 本轮八phase及59操作；原牌序/版本/归档一致 |
| 重复同目标恢复 | PASS | 一次恢复Promise/一次重连，0个重复room命令 |
| 跨入口并发恢复/最新选择 | **FAIL** | G47-N01，迟到成员join仍覆盖B |
| CLOSED/nonmember最终校验 | PASS（本轮显式恢复路径） | 原反例及已采用后的关闭扩展；不扩大为所有入口均通过 |
| 持久化中断/COMMIT未知结果 | PASS（端口注入范围） | R18应用7+真实SQL4案例复用；非磁盘断电 |
| 操作历史、手牌、轮次与结算一致 | PASS | 本轮重启前后深比较、checkpoint33、final60和完整验证 |
| 业务边界、API及旧模式兼容 | PASS（受检源码范围） | 当前差异、CI、R18/R19历史两步kit/归档证据，模式隔离本轮重验 |
| 完整可靠账号恢复/E03 | **FAIL** | 正常路径通过，跨入口并发缺陷仍阻断 |
| 原生包、真实设备/发布资格 | 未验证/BLOCKED | 本次export仍SKIPPED，未执行当前设备/SDK/发布验收 |

PASS仅对应表内明确范围和复用前提；没有把未执行的设备、存储灾难或容量测试填为通过。未把本表替换历史114项矩阵。

## 9. 新执行、复用与未覆盖风险

新执行命令/记录：

- `npm --prefix clients/packages/doudizhu-client test`：43/43 PASS。
- `npm --prefix clients/fair-doudizhu-cocos test`：71/71 PASS，含当前构建。
- `npm --prefix clients/fair-doudizhu-cocos run typecheck`：exit0。
- `recovery-races.mjs`：原A/B、CLOSED、重复、已采用后关闭、账号切换、销毁、发现后关闭7场景PASS；stale-member-join真实反例FAIL。
- `recovery.mjs setup/lobby/ready/partial/beacon/bidding/playing/terminal`：8/8 PASS，期间真实API kill/start。
- `mode-isolation.mjs`：PASS；`proof.mjs`本人核验及 `go run -p 1 E/verify.go E` 新归档完整验证PASS。
- 独立DB审计：3条保护记录、第一条原值不变、59条操作、checkpoint33及finalVersion60。

完整argv、cwd和结果在runs.jsonl；本轮没有重新跑相同SHA下成功的整套CI、旧压力测试、原生构建或基础密码学测试。旧fault/proof/security文件与原manifest摘要一致且相关实现无变，复用清单在reused-evidence.json。R19发现ACL/32–33上限/真实MySQL JSON查询证据亦沿用，未声称本轮重跑。

尚未验证/未承诺：当前原生字符串控件/键盘与手机UX、实体换设备、平台SDK实际账号切换、正式域名/TLS及发布、MySQL主存储损坏/断电恢复、长期soak与最新大规模容量。发现查询的JSON成员过滤不是actor索引，结果上限不等于数据库扫描成本硬上界，此风险沿用R19。客户端全证明UI尚未实现，不能用独立工具通过冒充它。

H05内部抖音入口/身份/合格导出适配及GOAL8交付条件仍不完整，不能全部归因外部配置。C09/C10根迁移/退役按用户既定修订继续DEFERRED，不执行且不计PASS。旧平台微信登录及凭据已有成功历史，不误写为“微信未登录”。

## 10. 交接与完成资格

本轮可执行工作已完成，当前唯一新确认产品问题为G47-N01/P2；交ChatGPT按第7节最小集合修复，再复验失败交错和必要正常链路。随机贡献与后端恢复无需据此重做。

验收报告自检核对了原始反例/扩展残留的区分、真实与替代边界、证据复用的源码/摘要前提、负例未弱化、故障级别、秘密排除和最终差异。未提交私有夹具，未改历史FAIL，仅本报告及current允许状态。一次提交到main并push，报告提交与受测产品SHA分开。

**最终FAIL；E03尚未完成，Goal0045/0046保持 `in_progress`；不封版、不发布。**
