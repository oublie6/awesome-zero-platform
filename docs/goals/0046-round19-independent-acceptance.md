# Goal 0046 — Round 19 无状态恢复与公平模式竞态独立验收

日期：2026-11-07。Codex独立验收；没有修改产品、正式测试、Schema或历史报告。

## 1. 结论与原问题关闭资格

**本轮 FAIL，15项矩阵13 PASS / 2 FAIL / 0 BLOCKED；确认2个新P2恢复缺陷。**

- **R18-N01：原始反例PASS、定位根因已修复；完整关闭资格暂不授予。** 真实新进程仅认证即可发现本人房间，恢复各正常阶段，活跃成员也能恢复；但本轮明确要求的多房间/迟到响应/最终成员状态竞态仍失败，须先修复R19-N01/N02。不是要求重做已通过的发现接口。
- **R18-N02：PASS，已关闭。** 未知模式不渲染公平操作；底层旧操作拒绝托管局，旧H1响应不能生成秘密或污染H2，随后新贡献正常。
- **E03修订后完成资格：FAIL。** 正常的无牌局持久化恢复已得到真实证据，完整可靠恢复承诺仍被两个新P2阻断；不重新强加新模式手工恢复凭证要求。
- **Goal0045/0046继续 `in_progress`**。H05/GOAL8未完成；C09/C10仍DEFERRED，未执行根迁移/退役。原R18-N01不重复计为第三个新缺陷。

## 2. 固定SHA、差异和CI

实际产品：`610c90a0fd6c9827156cafe202658ab6c202e9fc`。初始main干净，经 `git pull --ff-only` 从报告 `6b8ecefbe7c4504a6b6fa005f95d8296b36b738e` 同步到一个产品修复提交；结束前再次核对仍为固定SHA。16文件、546行新增/28行删除，未混入其他产品提交。

已独立读取head SHA及全部job：

| 工作流 | 精确610c90a的结果 |
|---|---|
| [主CI 42372393407](https://github.com/oublie6/awesome-zero-platform/actions/runs/42372393407) | SUCCESS，integration/unit/admin_web/secure_envelope/runtime/report **6/6** |
| [Fair Doudizhu 42372393374](https://github.com/oublie6/awesome-zero-platform/actions/runs/42372393374) | SUCCESS，application/client_foundation/manifest_client/mysql_integration **4/4** |
| [Cocos 42372393388](https://github.com/oublie6/awesome-zero-platform/actions/runs/42372393388) | 总体SUCCESS；**linux_source SUCCESS、linux_native_export SKIPPED** |

差异实读：服务端新增actor作用域GET `/v1/doudizhu/me/rooms`，SQL用真实Go JSON的`Seats/AccountID`及JSON_CONTAINS精确匹配、返回上限32；逐个RoomView重新检查当前成员/关闭状态。客户端增加发现/选择、活跃成员恢复、发牌前先读fairness而不索取私牌；Bootstrap用null区分未知模式与空字符串历史模式，核心旧公平方法拒绝新托管模式。

上述正向实现经过实际行为验证，未以源码保护条件或CI成功直接判整个Goal通过。报告SHA与产品SHA分开，报告提交可由 `git log -1 --format=%H -- docs/goals/0046-round19-independent-acceptance.md` 查询。

## 3. 环境、方法与证据边界

按AGENTS、原则、current、Goal0046、[Round18](0046-round18-independent-acceptance.md)与托管架构追踪受影响链路。复用Linux、Node22.23.4、自有MySQL5.7.44/Redis8.8.0、TLS、drand与稳定保护根；**本轮从固定候选重新编译API**，客户端协议/shell也重建。为避免历史验收账号的多房间记录影响单房间断言，复用认证服务和管理员，在隔离库建立4个新真实账号，不删改旧证据。

私有证据 **`E=/root/.config/awesome-zero-platform/acceptance-0046-r19-610c90a`**。`runs.jsonl`记录argv/cwd/产品SHA/UTC/耗时/退出码，`ci.json`、`provenance.json`、`sha256.json`及`completion-audit.json`索引结果；账号、token、测试贡献及审计数据不提交。历史Round17/18报告和原始FAIL保留。

当前Bootstrap经实际TS转译，`renderScreen('rooms')`实际执行菜单分支，只有Cocos几何节点创建/布局替代；API、controller、FairnessActions、RoomCoordinator、HTTP/WSS、Go认证/规则/数据库真实。不是原生包/手机点击证据。账号认证用隔离账号真实Go登录，不声称本轮重复微信SDK交换；新账号替换旧账号的迟到响应测试在已认证controller层执行。

每个cold phase均为独立Node进程，重新认证并从空roomID/handID的场景调用无参数recoverMyRooms；期望RoomID/HandID仅在验收文件中作为结果断言，不传给恢复入口。没有读取旧客户端秘密、kit或pending来完成新模式恢复。

## 4. R18-N01正向与跨阶段恢复

主手局 **`hand-cmd-bbffc2adadc9eee06ced97c5866e9991`**，`recovery.mjs`八个phase：

| 阶段 | 实际结果 |
|---|---|
| setup | 真实room.create已接受后丢ACK；原请求UNKNOWN，客户端进程退出 |
| lobby | 新进程仅认证，自动发现唯一等待房间，恢复版本/本人座位/RoomClientSequence，无重发create；随后三人入座准备 |
| ready | 三个新controller各自找到唯一READY房间，准备状态与权威版本一致；正常开新托管局，第一人贡献已接受但丢ACK后退出 |
| partial | 新进程自动定位同HandID和已贡献状态，不重生成本人材料；其他两人继续贡献。三人均**0个private手牌同步请求**，无尚未发牌错误 |
| beacon | 新进程恢复WAITING_PUBLIC_BEACON，同样无提前私牌请求；实际公开drand到达并发牌 |
| bidding | 新进程恢复同牌序、三个合法座位和版本，收到真实WSS私牌快照并正常Bid |
| playing | 实际API进程SIGKILL/重启后，新进程仅认证恢复同局和三人原private view；真实WSS重连不改状态，随后完整打完对局 |
| terminal | 新进程从服务器恢复lastHand归档，COMPLETED/version/digest相同；三人本人贡献查询正常。合法离房后发现列表不再包含该房间 |

上述均exit0；**59条真实Bid/Play/Pass，finalVersion60**。这是一次受本轮恢复链路影响的完整正控，不是压力测试。加密贡献仍3条，第一人密文与重启前原记录完全相同；其余随机/存储/Journal/Checkpoint基础复用Round18未变实现证据，不重复完整故障矩阵。

`discovery.mjs active`另创建两个真实活跃手局：不自动猜选，实际房间菜单出现两个恢复入口；明确选B后采用B。已入座成员join恢复HAND_ACTIVE不发room.join、不再打开额外socket；非成员客户端与直接Go命令都拒绝进入活跃局。GET me/rooms忽略伪造actor查询参数，非成员得不到其他账号房间、HandID或私有roster。

32个真实房间全部返回；第33个时接口明确失败，不静默返回部分列表，关闭一个后精确返回其余32个，测试房间随后正常关闭。实际MySQL新增集成测试也验证Go JSON格式、相似账号不匹配和关闭排除。EXPLAIN使用 `idx_doudizhu_rooms_status_updated` range，当前数据集估计扫描9行；JSON成员过滤仍非actor索引，LIMIT只约束结果数，不能据此声称大规模扫描成本有硬上界。本轮未跑容量测试。

## 5. R18-N02模式隔离及历史兼容

`mode-isolation.mjs`复用原H1取消→H2已建立→模式HTTP结果延迟的真实反例，额外扣留一条原H1合法prepare响应：

1. 先确认旧H1模式且实际菜单有旧Prepare入口，发起真实旧prepare并延迟其HTTP结果。
2. 合法取消H1、重新准备、开新托管H2；peer刷新并延迟新H2模式projection。
3. 此时handID已H2、mode=null。实际重新渲染房间菜单，既无旧kit/Commit/Reveal，也无未授权的新提交入口；旧UI方法直接调用被挡。
4. 绕过UI，实际调用prepareRecoveryKit、带正确结构/自身digest的importRecoveryKit、commit、reveal：均拒绝，0个旧命令、无PendingSecret/kit。reveal在有效新会话没有旧secret时拒绝，不伪造不可能的服务器阶段来制造通过。
5. 释放H2模式响应，再释放旧H1响应：旧实例已失效，当前模式/实例和秘密状态不受污染；实际菜单仅有新模式贡献/本人核验入口，新贡献被Go接受，不再出现historical fairness secret still active。

该真实反例exit0。旧模式独立验证：明确不指定mode创建历史局，UI先读权威模式再准备kit；3人commit、Node退出、新Node导入原kit并真实HPKE reveal通过。已有旧completed归档也在当前API上重新读取验证，未迁移历史数据。未变的终局密码学/牌序/BLS及完整重放使用Round18有效依据；本轮重验新旧归档访问和当前场景恢复，不重复基础算法测试。

## 6. 新发现问题

### R19-N01 / P2 — 较早恢复结果覆盖用户较新的房间选择

位置：`GameOnlineController.ts:61–81`只捕获membershipGeneration，两个reconnect恢复之间不推进该代数；`GameBootstrap.ts:284–300`仅比较online对象，没有恢复操作token/最新选择围栏。

最小真实复现：同一账号有两个合法WAITING房间A/B，实际菜单要求明确选择。先恢复A，扣留其真实HTTP房间结果；再恢复B并完成，客户端及RoomCoordinator为B/v1。释放较早A结果，客户端及协调器却改回A/v1，仍显示恢复成功。两次操作是同一认证controller，账号指针检查不能阻止此覆盖。

证据：`recovery-race-stale-selection.log` exit1、`race-stale-selection.json`。A=`room-cmd-cf4dcd30c167664ba2591398a582e71d`，B=`room-cmd-89c5650279d9ea8a1ce54f6e0f31345d`；新的B/v1被较早的A/v1覆盖，0个room变更命令。这里为等待房间、HandID为空，不冒称已经实测跨局私牌泄漏；恢复请求是只读HTTP URL，没有游戏Command RequestID。

同根因附加证据：同目标双击恢复，`same-target-duplicate.log` exit1、`race-duplicate.json`，初始连接加两次恢复共打开3条认证socket，无room命令；缺乏请求合并/连接复用。该连接浪费表现本身较轻，不另计第三个独立P2。

影响：用户明确选择的目标不可靠，后续房间操作可能针对旧目标；没有证明绕过服务端授权或丢失秘密，不升级P1。

最小建议：在controller与scene两层定义当前恢复操作所有权，恢复开始即推进generation/token，并在每个await后及采用room/hand前验证；合并重复恢复，复用可用连接。仅加UI围栏不够，否则controller/RoomCoordinator仍可能回退。保留未知命令精确对账，不以清pending修复。

### R19-N02 / P2 — 最新查询已确认关闭/非成员，仍报告恢复成功

位置：`GameBootstrap.ts:298–300`在refreshRoom之后无条件输出成功，而refreshRoom内部捕获成员校验错误；controller此前已依据延迟旧读采用房间。

最小真实复现：恢复A时扣留真实member=true、WAITING/v1响应；该账号另一合法controller正常退出并关闭A，server为CLOSED/v2、member=false。释放旧读后，恢复过程的后续实际fresh room查询拒绝成员身份，但最终scene仍绑定A、协调器仍v1，显示 **“已通过服务器恢复本人房间和牌局”**。

证据：`recovery-race-closed-during-recovery.log` exit1、`race-closed-during-recovery.json`；A=`room-cmd-0695a11a5446127924d18507425f8d29`，server CLOSED/v2/nonmember，对比client A/v1/success。不是伪造服务端状态：通过实际owner leave关闭，只延迟旧HTTP结果。服务端后续授权仍严格，没有证明非成员取到了私有手牌。

最小建议：让恢复最后的权威校验有明确成功/失败结果；若最新读取证明关闭或已非成员，应终止恢复并使已采用的本地房间状态失效，不覆盖错误为成功。不要无限重试旧版本，不将未确认状态伪造成成功；对比成功ACK后业务采用与纯查询恢复的不同语义。

两项发现后继续完成其他检查，各反例只作必要独立验证。它们不是原R18缺少发现接口/旧模式污染的同一个根因；原定位问题正常反例通过，但本轮规定的完整N01恢复族仍FAIL。

## 7. 完整验收矩阵

| 编号 | 结果 | 独立依据 |
|---|---|---|
| N01-01 账号发现未结束房间 | PASS | 真实等待/READY/ACTIVE、MySQL JSON精确匹配、32/33边界及关闭排除 |
| N01-02 无本地数据冷启动 | PASS | 八个独立Node phase，无locator输入；另查取消发牌前旧局后的等待房间也能恢复 |
| N01-03 成员恢复活跃局 | PASS | 真实HAND_ACTIVE成员恢复，0个join命令；非成员直接命令被拒 |
| N01-04 发牌前公平恢复 | PASS | FAIRNESS_COMMITTING/WAITING_PUBLIC_BEACON不发private sync、不误请求未发牌视图 |
| N01-05 出牌断线/进程恢复 | PASS | 客户端全新进程、实际API重启、WSS重连，原状态/牌序不变并继续到终局 |
| N01-06 多房间明确选择 | **FAIL** | 正常多活跃选择通过，但交错A/B选择被较早响应覆盖：R19-N01 |
| N01-07 非成员/跨账号隔离 | PASS | 真实actor作用域、忽略伪造actor、拒绝活跃新加入；B新controller不接纳A迟到列表，销毁后不采用 |
| N01-08 版本/ACK/WSS竞态 | **FAIL** | create/贡献ACK丢失后冷恢复通过；但CLOSED/v2被旧v1采用且误报成功，重复恢复开多连接 |
| N02-01 未知模式隔离 | PASS | 实际renderScreen菜单分支在null时无旧/新公平操作；权威模式到达后才显示 |
| N02-02 旧凭证底层拒绝 | PASS | 有效新模式的4个核心旧操作全部拒绝，未生成secret/kit或提交旧命令 |
| N02-03 迟到/跨局隔离 | PASS | 真实H1 prepare结果晚于H2模式到达，旧实例/回调不能污染新会话 |
| N02-04 历史Commit/Reveal | PASS | 真实旧模式三人流程、独立Node kit恢复/加密揭示，历史归档可读 |
| N02-05 新贡献正常 | PASS | 冷恢复后继续三人贡献，旧操作拒绝后合法新贡献仍接受 |
| REG-01 服务端/加密记录 | PASS | 当前编译API重启后原private view及3条加密记录不变，59条持久化操作；基础故障/Checkpoint沿用R18 |
| REG-02 新旧终局验证 | PASS | 新模式COMPLETED/v60冷发现及本人材料查询；当前API读取已有旧completed归档，基础验证实现未变 |

**13 PASS / 2 FAIL / 0 BLOCKED，失败条目映射2个新P2（另有同根因重复连接表现），不把断言数量当缺陷数量。**

## 8. 执行汇总、限制和交接

- `npm --prefix clients/packages/doudizhu-client test`：**42/42 PASS**。
- `npm --prefix clients/fair-doudizhu-cocos test`：**66/66 PASS**；static typecheck通过。
- `go test -p 1 -parallel 1`：application/mysqlstore/dispatcher三个受影响包通过。
- `go test -tags integration -p 1 -parallel 1 -run '^TestRecoverableRoomDiscoveryWithRealMySQL$'`：实际MySQL PASS。
- 八阶段冷恢复、模式隔离、多活跃房间/ACL、32/33上限、旧模式两进程与旧completed读取均有独立日志；两个新P2及重复连接负向断言保留FAIL。

夹具修正：创建测试账号正常返回201，初始旧helper只认200，已针对合法成功码修正并保留日志。首次归档补验错误要求“未发牌取消局”已有final archive，实际404不能单独判产品缺陷；另以真实冷恢复确认该等待房间不暴露无效lastHandId且能恢复，再对确实已归档的历史completed行验证通过。未删除业务负向断言、改产品或把404盲目忽略。

本轮只复验两个修复点及列明关联路径；未重跑Round18密码学全套、规模/SIGKILL全矩阵或Cocos导出。当前原生package、实体设备/微信账号切换SDK、手机UX及发布资格没有新证据，仍BLOCKED/未验证，不用Node场景代替。H05内部抖音适配、GOAL8正式交付保持未完成，C09/C10仍DEFERRED。

**是否实现无牌局持久化恢复：** 单房间各阶段已经实际实现，不依赖客户端随机数、字符串或旧房间缓存；但可靠多房间选择与最终成员状态的一致性尚未通过，故不宣称完整恢复资格或E03完成。

交回ChatGPT的最小集合：①恢复操作generation/最新选择围栏和重复请求/连接合并；②最后权威校验失败时不保留失效room或报成功。保留已通过的账号发现、发牌前恢复、模式UNKNOWN隔离、旧核心方法校验与精确重试，修复后仅复验失败反例及必要正常链路。不得通过本地保存locator/秘密来规避。

Goal0045/0046继续 `in_progress`。仅一次提交本报告及current允许状态到main；保留历史报告，私有测试API/TLS/自有数据库服务停止并保存证据，其他部署不改。
