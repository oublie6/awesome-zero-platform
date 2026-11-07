# Goal 0046 — Round 18 玩家贡献与服务端托管专项独立验收

日期：2026-11-07。Codex 独立验收；未修改产品、正式测试、数据库Schema或历史报告。

## 1. 结论与版本

**本轮 FAIL：M01–M16 为12 PASS / 4 FAIL / 0 BLOCKED；确认2项P2。E03修订后需求尚不能关闭，Goal0045/0046继续 `in_progress`。**

验收产品固定为 `895cf44a2b1540e05ee218752bf5997374c14f90`；初始main工作区干净，`git pull --ff-only` 从Round17报告提交 `31f9f9d27af28ecbb405667fcc678e7298ac5c9a` 同步；报告前再核对无前进。新增一个完整产品提交，36文件、1315行新增/130行删除，未混入其他分支代码。报告文档提交与产品SHA分别记录。

- [CI 42346050009](https://github.com/oublie6/awesome-zero-platform/actions/runs/42346050009)：精确895cf44，SUCCESS，secure_envelope/admin_web/integration/unit/runtime/report **6/6**。
- [Fair Doudizhu 42346050004](https://github.com/oublie6/awesome-zero-platform/actions/runs/42346050004)：精确895cf44，SUCCESS，client_foundation/manifest_client/mysql_integration/application **4/4**。
- [Cocos 42346049985](https://github.com/oublie6/awesome-zero-platform/actions/runs/42346049985)：精确895cf44，总体SUCCESS，**linux_source SUCCESS、linux_native_export SKIPPED**。

已独立读取run SHA及jobs；未重新运行完整CI。源码门槛成功不代表当前原生导出、手机输入或发布资格。

## 2. 本轮范围与环境

按AGENTS、原则及已修订Goal、[新架构](../architecture/doudizhu-client-custodied-fairness-v1.md)、[Round17](0046-round17-independent-acceptance.md)追踪实际链路：Bootstrap→GameFairnessActions→签名context/HPKE→dispatcher→SubmitHandContribution→同一事务中的AEAD记录/Hand/receipt/Outbox→drand→durable live Journal/Checkpoint→终局与本人验证。

实际验收的是 **`client-contributed-server-custodied-v1`**：用户字符串和客户端32字节CSPRNG一起加密上传，服务器知道完整贡献并负责持久化与恢复；不是server-managed-v1，不宣称服务器无法提前知道玩家贡献。未指定模式的历史局仍用两步Commit/Reveal。

复用Round17 Linux、Node22.23.4、自有MySQL5.7.44/Redis8.8.0、TLS与四个真实Go认证账号，三个参与者及一个非参与者。使用真实HTTPS/WSS、MySQL、drand Quicknet和权威Go规则；登录使用已有隔离账号的实际Go认证，不冒称本轮重新验证微信SDK交换。

**重新从候选编译API**（`go build -p 1`），不复用旧后端二进制；客户端协议包/scene shell也由当前候选重建。Bootstrap当前TS源码在Node中转译，Cocos基础节点用替代；业务controller、客户端加密/协议、Go服务和数据库真实运行。所有UI结论限定为当前源码处理流程，不是原生触摸/真实手机输入。

私有证据 **`E=/root/.config/awesome-zero-platform/acceptance-0046-r18-895cf44`**。`runs.jsonl`保存argv/cwd/SHA/UTC/耗时/退出码；`ci.json`、`sha256.json`及`completion-audit.json`索引证据，二进制校验值在私有provenance中。测试账号、原字符串/随机数、测试库解密结果和恢复材料不提交。沿用稳定保护根，只更新隔离测试reveal registry有效期；未运行C09/C10、未下载SDK、未跑无关规模/完整SIGKILL矩阵。

## 3. 真实玩家贡献、加密与持久化

`custody.mjs setup/partial` 使用当前Bootstrap.startHand及submitOwnContribution：

1. 三账号建房、入座、准备，明确启动新模式；三人分别输入不同字符串：含中文、全角/组合Unicode、前后空格，以及256字节ASCII边界。
2. 在实际客户端安全随机源边界观察32字节输出副本，保持真实Node CSPRNG生成，不提供固定假随机；将其与独立AES-GCM解密的MySQL原记录逐一比较，确认随机数来自客户端，原字符串完整保存且三人随机值独立。
3. UI提交前同步清空输入，无手动备份/导入步骤；网络只含HPKE envelope及MAC context，不含明文字符串/随机数。新模式公平对象无recovery secret或prepared kit，仅UNKNOWN时保留内存中的原密文请求。
4. 独立按record ID、HandID、Seat、Actor、CommandID和digest重建AAD，验证aad_digest并用测试服务器根解密真实记录；TS重新计算的承诺等于服务端承诺。保护记录的唯一hand/seat及actor/command约束存在。
5. 实际Snapshot、Outbox、command receipt扫描未发现原字符串或原随机数；实际live Journal均为v2、32字节HMAC，最终59条操作。敏感原材料在贡献记录中为经过认证的密文，不在这些普通JSON记录中明文保存。

`security.mjs`执行 **14类真实负向请求**：改requestId、clientSeq、version、HandID、认证actor、seat claim、MAC、伪造expiry、跨旧Reveal模式、密文、AAD，以及使用合法外层context重新加密但内层HandID/Seat错误或phrase为空。均被拒绝，无accepted contribution、无protected row、Hand保持版本1；随后合法正控成功。另有空白/全角空格及257字节/多字节超界在客户端网络前拒绝。256字节及NFKC正常提交已实测；不把字符数当UTF-8字节数。

`crypto-negatives.json`与security日志记录各状态/拒绝；context的actor/seat/hand/request/seq/version/key/time绑定由真实服务器MAC与AAD共同校验，非只检查存在一个保护条件。

## 4. 原子性、幂等及UNKNOWN

- 真实贡献已COMMIT后丢弃客户端ACK；同一完整原请求、requestId、密文及签名context精确重试得到原accepted receipt，没有新随机替代、重复记录或额外推进。
- 另一参与者已接受贡献但响应丢失后，退出Node进程。新进程在**显式给定房间定位的正控**中只从服务端状态得知已committed/revealed，拒绝重新生成/提交；不依赖旧pending、随机数或字符串恢复。无定位入口的反例见N01。
- 私有Go overlay执行7类应用端口故障：解密、保护加密、插入、Hand更新、提交失败、已提交但提交结果丢失、合法命令过期。失败无部分贡献/状态/Outbox；已提交UNKNOWN按同一请求恢复原记录。
- 独立real-SQL overlay执行4类：记录插入失败、真实加密INSERT后Hand更新失败、事务结束前故障、**真实MySQL COMMIT完成后丢弃返回结果**。前三类检查真实库回滚至Hand v1、零贡献/receipt/Outbox；最后一类保留一条记录，原ID重试duplicate accepted，仍只有一条。

证据：`independent-atomic-faults.log` 7/7、`real-sql-faults.log` 4/4。注入点在应用/事务端口，使用临时overlay而非产品改动；SQL部分使用真实MySQL和真实保护器，但EnvelopeOpener采用受控合法明文。真实HPKE解密另由完整HTTP链路及14类反例验证。**未声称做了MySQL引擎崩溃、磁盘断电或真实网络COMMIT代理中断**；提交失败为确定性端口故障，提交结果丢失正控则已有真实持久化。两者的证据层级分别记录。

## 5. 服务端与客户端进程恢复

主手局 **`hand-cmd-8131700e63b2f2e09cfbe30b6e6879aa`**。每个phase为不同Node进程，重新认证、创建新controller；各phase之间控制真实API进程SIGKILL并重新启动，保留原根和存储：

| 崩溃/重启边界 | 结果 |
|---|---|
| 只有第一人贡献已持久化 | 原Hand、mode、版本、原记录及承诺完全相同；其他两人可继续贡献 |
| 三人完成、公开信标公布前 | 实测future beaconReleaseAt仍在未来，重启后记录及固定round不变，无再次贡献 |
| 已发牌后 | 三个账号private view逐字节/结构一致，HandID、牌序、席位及版本未变 |
| 已叫地主后 | 原PLAYING状态、地主和版本一致 |
| 32条实时操作及checkpoint后 | checkpoint存在（版本33），恢复三人原private view；继续原局 |
| 已结算归档后 | final archive及版本、digest、archivedAt完全相同，无重新结算 |

`setup/partial/dealt/bid/playing/finish/verify`均exit0。整个对局接受 **59条Bid/Play/Pass，finalVersion=60**，真实三人HTTPS/WSS完整进行，非预造牌局。最后27步复用仓库三客户端E2E工具完成；不存在手工备份依赖。

**恢复资格边界：** 为继续其他可执行验收，正控由独立夹具提供已知room locator，再读服务器权威状态；测试文件保留期望值仅用于比较，不被产品读取。此正控证明服务器具备数据和游戏恢复能力，**不证明产品能仅登录自动找到活跃房间**。后者真实失败，不被正控掩盖。

## 6. 客户端无牌局持久化

审计client scene、协议包及platform adapters：新流程未调用localStorage、IndexedDB、文件、平台Storage或vault保存hand、贡献、kit、Journal/Checkpoint；实际存储适配调用仅用于普通认证token生命周期。旧局kit为显式历史流程，不用于新模式恢复。

完整链路跨7个Node进程、新controller及6次服务端重启；新实例无旧secret/原pending，不输入原phrase也能在**已定位的房间**继续。原phrase仅在终局验证阶段重新输入。实例内UNKNOWN密文重试和UI/连接状态是临时内存，不是跨进程游戏缓存。私有E的期望值/账号/审计副本由验收脚本写入，非产品客户端持久化。

没有重新做原生包或实体换手机测试；“换设备仅登录”的产品流程本身已有N01，不标PASS。

## 7. 终局字符串验证A与整局证据验证B

**A — 本人字符串独立复算：PASS。** 三人分别通过own-contribution取得自己的原32字节随机、seat、digest及commitment，无phrase返回；当前Bootstrap.verifyMyPhrase实际调用客户端哈希函数。正确输入和NFKC等价输入通过，错误输入、其他HandID/Seat失败，输入框清空。不是只信任服务器布尔值。

真实授权检查：终局前拒绝读取随机数；nonparticipant、未认证、真实已过期旧token拒绝；A添加seat=2/actor=B查询参数仍仅得到A原随机；终局公开给参与者的归档也不含B原random/phrase。保护记录密文翻转一位后拒绝披露，恢复原行后正常；原数据已恢复。源代码还以record身份/AAD/digest校验约束解密结果。

**B — 整局随机材料/牌序/过程：PASS（独立验收工具层）。** 使用实际归档与三人own proof在独立Go进程：

- 比较本人proof与归档transcript中对应三个digest/commitment；重新计算server commitment、确定性洗牌及deal。
- 用独立预置的Quicknet公钥重新验证实际drand round的BLS签名及digest，不信任服务器返回“verified”布尔值。
- 重放transcript、bidding、playing、结算并比较record digest；改deck、contribution、beacon round/digest及winner后拒绝。结算负例重新构造外层record digest，避免只验证外层checksum的假阳性。

证据：`independent-final-proof-compact.log` exit0、`verify.go`。当前UI只完成A并展示服务端归档摘要，未声称UI已内置全部B验证；B的输入实际可从授权归档取得，本轮由独立工具验证。

服务器提交时已经知道全部玩家原材料，短phrase也不是高熵认证凭据。上述结果不证明服务器不能对未见证commitment作恶或选择性终止；更强不等价承诺仍需独立见证，不以账号token冒充公平性证明。

## 8. 历史兼容与网络竞态

真实未指定mode的新测试局仍走历史Commit/Reveal：三人旧kit备份、commit、Node退出、新进程导入原kit及加密reveal通过。实际当前数据库中已有旧completed归档 `codex-hand-e166de6ac084be93ee47cfb5` 通过新API读取/校验，mode仍缺省，没有强行改写旧证据。

R16-N01原WSS路由前延迟ACK反例复用后，追加submitCustodied：HTTP已读H2而ACK未投递时，旧实例/secret与原协调器pending不变，prepare/commit/新贡献均被阻止、零公平请求；ACK后进入新模式并三人单步贡献成功。`new-mode-start-race` exit0。因此原R16修复仍PASS，**不能据此认为所有模式发现竞态都通过**，N02为另一真实路径。

夹具记录：归档pretty JSON曾改变raw payload空白，独立record digest比较首先失败；规范为原compact JSON后仍保留digest及全部语义负向检查，复验通过。最初Round10归档locator不在当前隔离数据库，改为当前库已有且参与者有权限的旧completed行；未修改旧归档或产品。SQL审计首字段名修正为payload_json。上述为夹具错误/数据集差异，不计产品缺陷。

## 9. 新发现缺陷与最小修复建议

### R18-N01 / P2 — 全新客户端不能仅登录恢复活跃牌局

位置：`GameOnlineController.ts:61–66`无房间locator则return null，`:88–96`先拒绝HAND_ACTIVE，后才处理member；Bootstrap.login不发现活跃房间，reconnect依赖内存roomID。

真实最小复现：在 `security-c2403a9447169f3b` 活跃手局，创建同账号全新认证controller和roomID/handID均空的当前Bootstrap；点击对应reconnect处理方法。服务器仍返回本账号成员身份与该activeHandId，但客户端显示 **“重连成功，等待创建或加入房间”**，两个ID仍空；已知房间号再走joinRoom也被room not joinable拒绝。

证据：`cold-recovery-repro.json`、`security.mjs`；房间 `room-cmd-f16b28e83722e6a7ec389bf55ddc4df7`，开局requestId `cmd-b78be0b9b32edb6db2ac181165271d2c`，房间版本7、Hand版本1。主partial重启同样记录无locator恢复返回null。不是缺凭据或服务端数据丢失；显式调用reconnectAndResync(roomID)的正控能恢复。

影响：客户端死亡/换设备后，只靠账户认证无法找回已有局；即便用户另外知道房间号，现有UI加入入口仍阻断活跃成员。违反本轮无本地牌局数据、仅登录继续的必需条件，M06/M15/M16失败。

最小建议：由业务模块提供认证actor作用域的活跃/可恢复房间发现契约并接入登录/重连；在服务端成员身份确认后允许现有成员恢复HAND_ACTIVE（不放开其他人的新加入）。不以本地保存room/hand/kit替代服务端发现；补全新进程、未知贡献结果与账号切换回归。

### R18-N02 / P2 — 模式发现期间旧凭证入口污染新公平实例

位置：`GameBootstrap.ts:365–382`先按旧mode渲染房间，再更新HandID并await新mode；`GameFairnessActions.ts:158–180`prepareRecoveryKit只查phase/committed、不拒绝托管mode，因而在新局建立legacy secret；新submitCustodied又正确拒绝持有此secret的实例。

真实最小复现：旧H1取消后owner开新托管H2；peer执行当前refreshRoom，先取到H2房间状态，将H2公平projection的真实HTTP结果延迟交付。此时b.handID已H2，但mode仍空，刚渲染的旧PrepareFairness入口可调用。执行其实际处理函数，合法HTTP查询新局后成功建立H2旧式kit/secret。释放原模式响应，UI切新模式但秘密仍留在同一实例。输入正常字符串并提交，出现 **“提交贡献失败：historical fairness secret still active”**；普通刷新仍不能解除。

证据：`real-mode-race.log` exit1、`mode-race.json`，Go服务及数据库真实，Cocos节点替代，不冒称手机物理点击。房间 `room-cmd-c328d935ed35300b0b2980fd42b5d8c7`，H1 `e03-r14-a0ecd9bc70d07b0d`，H2 `mode-race-8a1e02c168256578`；开局requestId `cmd-765f7608638940453d977c9955585db2`、房间版本12、Hand版本1。server mode正确但peer committed=false；本次贡献在发命令前已被阻止，故没有可伪造报告的contribution RequestID。

影响：有实际旧UI回调窗口，新托管局无法提交贡献，可能等待到信标截止而终止。没有证明秘密跨账号泄露或服务端接受了跨模式Commit/Reveal，不升级P1。与原R16开局ACK锁缺陷不同。

最小建议：把模式“未知”与“历史模式”区分，权威模式未就绪前禁用/不渲染公平操作；按新HandID/mode完成采用后再渲染。旧prepare/import/commit/reveal应在核心层校验权威模式，禁止在托管hand创建旧式secret。保留真正旧局及其UNKNOWN秘密，不通过无条件清空secret绕过保护。增加peer跨局、迟到projection和旧按钮回调回归。

两项交回ChatGPT修复；验收方未改产品。失败后仍完成其他可执行验收，没有针对同一缺陷反复重试。

## 10. M01–M16矩阵

| 项目 | 结果 | 本轮依据/限制 |
|---|---|---|
| M01 玩家随机数和字符串 | PASS | 当前TS处理流程+真实三人独立CSPRNG、原文/边界及密文记录对比；非手机输入UX |
| M02 加密传输/身份绑定 | PASS | 实际HPKE/MAC/AAD、14类负例、合法正控 |
| M03 加密持久化 | PASS | 真实AES-GCM解密/AAD、原文/随机一致、普通记录无明文、篡改拒绝 |
| M04 原子事务/幂等 | PASS | 7应用故障+4真实SQL事务端口注入、COMMIT后丢结果、真实ACK丢失精确重试；见证据层级 |
| M05 客户端无牌局持久化 | PASS | 新模式存储调用审计、7个独立Node进程，无旧secret/kit恢复依赖；测试文件不是产品存储 |
| M06 退出/换设备恢复 | **FAIL** | R18-N01；已知room的正控不代表仅登录恢复通过；实体换设备未测 |
| M07 服务重启/Journal/Checkpoint | PASS | 6次真实进程重启覆盖5类边界，原记录/私牌状态/版本/归档不变 |
| M08 drand公开信标 | PASS | immutable future round、三人贡献先完成、实际proof/BLS独立重验 |
| M09 三玩家完整对局 | PASS | 真实HTTPS/WSS/MySQL，59操作到结算、finalVersion60 |
| M10 自己字符串独立校验 | PASS | 实际TS本地复算、正确/错误/Hand/Seat/NFKC及边界 |
| M11 完整洗牌/终局证据 | PASS | 独立工具重算输入、deck、BLS及过程/结算；UI未内置全部B |
| M12 权限隔离 | PASS | 终局前/非参与者/无效过期会话拒绝，不能选B/C，记录篡改拒绝 |
| M13 历史兼容 | PASS | 旧两步流程、新进程kit恢复与已有旧completed归档读取，服务端模式不混用 |
| M14 开局/网络竞态 | **FAIL** | R16原反例仍PASS，但新增R18-N02模式发现窗口阻断新局 |
| M15 E03修订后需求 | **FAIL** | 新局已去掉手工kit，但N01/N02未满足；不沿用旧备份UX门槛冒充新要求 |
| M16 Goal0046内部完成资格 | **FAIL** | 两项P2及仍未完成的内部适配/平台条件 |

**12 PASS / 4 FAIL / 0 BLOCKED，4个失败条目对应2个缺陷，不是4个独立缺陷。** 历史114项报告保留为其原范围证据，不声称本轮重跑或把新M矩阵替换进旧报告。

## 11. 自动化汇总与未验证范围

- 客户端协议 **41/41**、Cocos shell **61/61**、当前静态typecheck通过。
- 受影响Go application/domain/livehand/runtime/mysqlstore/dispatcher六包测试通过。
- 独立应用fault overlay **7/7**、真实MySQL fault overlay **4/4**；完整命令及日志在E。
- 7个主链phase、真实14类安全负例、边界/权限/旧归档补验、旧协议两个独立进程及新模式ACK竞态通过；冷发现与模式竞态反例按FAIL保留。
- 完整drand/确定性牌序/玩法及结算验证另进程通过，不以主CI替代。

当前候选实际原生导出、支持工具上的新字符串控件输入/可读性、真实手机换设备与前后台/网络/分享、正式域名/TLS和发布资格 **BLOCKED/未执行**，没有用Node模拟宣称通过。H05抖音入口/身份/合格导出与SDK适配仍有内部缺口，GOAL8正式交付仍BLOCKED。C09/C10继续DEFERRED，不执行根迁移或退役。

本轮只提交本报告及current允许状态，一次性提交main并push；私有API/TLS及自有测试DB服务停止、证据保留，其他部署不改。修复N01/N02后复验受影响流程及必要正常链路，才重新判断E03与内部资格；Goal0045/0046保持 `in_progress`。
