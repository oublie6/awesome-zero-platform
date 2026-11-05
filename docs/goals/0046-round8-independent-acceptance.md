# Goal 0046 — Round 8 独立质量验收

> **总体 FAIL；99 PASS / 9 FAIL / 6 BLOCKED。** 9个失败验收项对应3个独立产品问题，不能计作9个缺陷。内部代码与完整客户端验收FAIL；外部平台完整适配BLOCKED，正式发布资格不具备。Goal0045/0046继续 `in_progress`。

## 版本、职责及环境

实际产品源码固定为 `e0712d7169e0c69ddeecd59a0a6da30c82693249`。在干净main执行 `git pull --ff-only` 后核对候选；收尾再次同步仍为该SHA。读取AGENTS、四项原则、current、Goal0046、Round7、Linux工具说明，并参考历史问题依据，检查本轮产品差异及原始复现。验收期间不改产品、正式测试、协议、schema、CI或原验收要求。报告提交是文档提交，不冒充重新测试的产品源码。

用户在本轮追加要求：后续只维护Linux原生CLI4.0构建链，避免重复维护3.8.8，并明确要求写入AGENTS。因此本次另外修改AGENTS中的构建政策：官方cocos-cli+固定Cocos4.0引擎、禁用Wine/外国系统虚拟机兼容、旧3.8仅保留历史证据；正式迁移须同步项目版本、工具/扩展固定版本、CI和文档。本轮已完成的3.8证据保留，追加指示后不再新增该链路测试；不在验收中静默升级产品版本。

私有证据目录：
`/root/.config/awesome-zero-platform/acceptance-0046-r8-e0712d7`（下称E）。
保存 `runs.jsonl`、外置Go overlay/探针源码、原始失败、配置、构建元数据、哈希与截图。凭据、JWT、恢复套件、二进制、截图及下载签名不入库。

环境为Ubuntu22.04/Linux x86_64、约7.8GiB内存、无swap，Go1.25.8、Node22.23.4/npm10.9.9。GOMAXPROCS=2，Go使用 `-p 1 -parallel 1`，重型编译/测试/构建顺序执行。新建codex0046r8 MySQL5.7.44/Redis8.8.0隔离项目和独立integration/product数据库，不复用历史测试数据；磁盘MySQL、768MiB上限、64MiB buffer、COMMIT刷盘配置1，规模测试16worker。当前Go二进制用于真实HTTPS/WSS与整进程故障，独立supervisor受512MiB/CPU/运行时限约束。

微信运行环境为现有Linux社区开发工具v2.02.2609240-2、SDK3.17.5与已登录profile；使用用户授权的实际AppID及服务端私有AppSecret/HMAC根。AppSecret/上传密钥没有进入客户端，上传密钥未使用。只信任自有测试TLS证书；没有上传、预览发布、生产配置替换或修改Kubernetes部署。本轮本机不使用Wine或其他系统虚拟机。Creator3.8原构建只使用既有远程Windows CI，下载到本机的是生成的客户端产物。

产品三项ChatGPT修复按原复现独立重跑。Linux微信平台扩展曾由当前代理实现，因此该扩展代码的复查属于实现者复验/自审，不冒充另一位审查者的独立代码背书；本轮新构建及SDK操作是真实执行，不能继承以前的自测结论。

## 三项重点修复

| 问题 | 本轮结果 | 独立依据 |
| --- | --- | --- |
| R7-N03 生命周期计时 | **原始乱序复现PASS；生命周期整项仍未关闭** | 原Service/Dispatcher/Supervisor+真实叫分规则、受控返回port/时钟/终止port重跑；新v3先回调、旧v2后回调不再退回BIDDING。原历史/hand.dealt不续期、新动作续期、恢复及陈旧Sweep复核通过。扩展发现R8-N01。 |
| R7-N04 Creator集合兼容 | **原问题PASS，可关闭** | 新3.8实际生成代码使用forEach复制；真实SDK选牌/出牌、多个消息/关闭回调及解绑、主动重连与整进程重启后的私牌同步通过。Linux4同流程通过。终局状态问题另列R8-N03，不归因于Set编译。 |
| WX-N02 空事件拒绝回执 | **PASS，可关闭** | 原WSS首次拒绝→整个Go进程kill/start→HTTPS精确重放，HTTP200、failure不变、duplicate=true、两次events均为[]；SQL仍1房间/1业务Outbox/2命令结果。对应单元、MySQL冷回放和普通重试也通过。 |

R7-N03不能仅凭新增测试或原复现修复给整个生命周期PASS；R8-N01是另一个确定的提交/回调交错窗口。历史R3-N02原定时/SETTLING问题回归PASS，也不代表全部并发生命周期已正确。

## 新问题、优先级与复现

### R8-N01 / P1 — 已提交有效操作的延迟返回仍可被旧期限终止

位置：`server/business/doudizhu/application/lifecycle/supervisor.go:280`、`:298`、`:380`，`server/apps/app-api/internal/doudizhuapi/dispatcher.go:325`；实际运行时提交/返回边界见 `infrastructure/runtime/live_hand_bidding.go:46`、`durable_live_apply.go:35`。

独立外置探针 `TestCodexCommittedBeforeDeadlineDelayedReturn` 经真实Service、Dispatcher、Supervisor，复用真实叫分结果，控制runtime返回port、存储快照、虚拟时钟与终止port：

1. t0注册5秒BIDDING期限；t4新的有效低叫分到达代表提交成功的runtime返回边界。
2. 正常返回对照在t4完成回调，t6 Sweep终止次数0。
3. 延迟返回组暂停该调用，t6先Sweep，再释放原成功返回；终止port被调用1次，reason=bidding_timeout，随后原首次AfterBid仍执行。
4. `committed-callback-gap.log`，Race开启、退出1，5.393秒；不是Race detector的数据竞争报警。

新liveVersion只记录**已处理回调**，不读取尚未回调的已提交版本；两次due复核不能发现这个窗口。实际runtime把持久化/发布完成后才返回，Dispatcher随后才注册期限，二者没有共同的线性化保护。影响为有效进展中的手牌可能被旧期限提前终止，违反本轮新操作与到期竞争的生命周期要求。

这是有明确正常对照的受控port/时钟/终止复现，**没有声称已在真实SQL牌局中把该交错落库为EXPIRED**。实际MySQL/WSS进程故障是另一组证据。建议ChatGPT协调提交版本、期限更新和过期决策，覆盖“提交完成但回调未处理”和“终止预留/执行期间新动作”的交错；同时保留历史/重复不续期及真正新动作续期。

### R8-N02 / P2 — Cocos房间缺少退出入口

位置：`clients/fair-doudizhu-cocos/assets/scripts/bootstrap/GameBootstrap.ts:99`房间菜单及`:119`返回入口；`clients/packages/doudizhu-client/src/room.ts:74`已有leave能力。

真实3.8 SDK原生触摸完成登录、加入另一个正常认证玩家创建的房间；探针填入实际EditBox的房间号，未验证键盘/IME。当前按钮清单包含创建/加入/准备/返回首页等，没有退出房间；返回首页仅切换screen。已管理房间时建房被现有守卫拒绝 `already managing a room`。后端room.leave和SDK认证补充测试中的公开API退出均通过，缺失的是用户可操作的Cocos退出流程。

证据 `creator388-join-exit-proof.json`、菜单源码及实际按钮清单。探针通过另一个公开API控制器清理自有测试成员关系，不算原生退出PASS；该清理也不会清除原Cocos协调器的本地管理状态，后续创建失败不能误报为服务端建房故障。之后用独立新登录场景继续对局验收，没有代修UI。

影响为完整“创建/加入/退出”客户端要求未满足，用户无法用界面释放席位/正常离开。建议提供明确退出控制和未知结果协调，成功后清理room/hand/选择状态，并验证离开后加入其他房间及成员/房主边界。

### R8-N03 / P2 — 正确终局证据未映射为完成的牌桌状态

位置：`assets/scripts/game/GameClientShell.ts:84`的evidence处理、`assets/scripts/bootstrap/GameBootstrap.ts:122`牌桌渲染及`:349`同步/收尾路径。

两套实际SDK均以原生触摸完成owner的选牌、出牌和过牌，另两席为不同Go正常账号/真实WSS，没有owner后端代打：

- Creator3.8：原生20次出牌、1次过牌；整进程SIGKILL后确认版本20→20、epoch推进；最终真实SQL为COMPLETED、日志水位65、归档65、64条动作。UI仍标题“出牌中”，最后已确认快照停在64，显示“已收到终局证据，牌局操作已停止”。
- 新Linux CLI：原生20次出牌、1次过牌；SIGKILL后18→18；归档版本63，真实最终分数/摘要与原生“核验结算”显示一致，但UI标题仍“出牌中”、phase=PLAYING、version=62。截图人工复核确认。
- 3.8额外用真实微信身份重新登录/重连，原生RoomEvidence与FinalEvidence按钮均显示正确积分6/-3/-3、版本65及摘要；重新打开牌桌却是空“牌桌”/synchronizing，没有完成状态。

证据 `creator388-core-native-failure.json`、`creator388-terminal-independent.json`、`creator388-terminal-controls.json`、`linux4-full-native-ui.json`、`full-native-final.png`。Linux完整探针259.224秒、退出1，明确保留完成状态断言失败。

终局私牌接口目前HTTP404、归档证据接口HTTP200；**不把“终局仍必须存在活跃私牌接口”臆造为新协议要求，也不声称分数/归档丢失**。缺陷是客户端没有采用已验证终局结果完成状态收尾，仍保留旧阶段/牌/操作界面。建议在现有证据契约上完成终局映射、版本与操作门禁、重新进入显示；保留真实分数/证据校验。不得靠伪造私牌或静默放宽状态校验修复。

## 历史回归与真实业务

| 历史项 | 本轮执行结果 |
| --- | --- |
| WX-N01 普通请求重试 | PASS。25/250/1250ms热重试、kill/start后先错seq再原请求、join/ready/leave及后续房间演进后回放；原回执稳定、无额外业务作用。实际wx认证补充亦验证同ID建房重试。 |
| hand.dealt、R3-N02旧请求/计时 | PASS原复现。重复dealt不重置期限、不降PLAYING；历史Bid/Play/Pass不续期；真实SETTLING重试同进程推进COMPLETED，5次最终重试不重复动作/归档/结算。 |
| R3-N03、R4-N04、R5-N05 | PASS原复现及扩展。冷启动错seq先到再原请求、7类历史、actor/type/score/cards/version负例；HMAC字段篡改、降级、逻辑重排、缺根/错误根、内部zero与公共序列隔离均拒绝；合法物理倒序插入恢复通过。 |
| F01/F09 | PASS。真实签名Manifest/AAD/HPKE与实际drand/BLS；不依赖不用的本地HMAC beacon secret。 |
| F02/N01 | PASS。真实终局回复被丢弃、整个Go进程SIGKILL、三客户端断线、精确原回执/归档恢复；历史最高叫分跨PLAYING/重启回放通过。 |
| F03/F04 | PASS。12项Genesis篡改、真实完成归档克隆/7项prune边界、默认dry-run/显式execute/30天边界，以及真实删除事务中SIGKILL全量回滚。 |
| F05/F11 | PASS。真实1/2/3席房间解析、API离开/重入；发送前丢失→权威404→原ID重发，提交后丢回复→权威协调。GUI退出缺失另列R8-N02。 |
| F06/F07 | PASS。3项原客户端反例+11项异步生命周期矩阵，已知拒绝与未知结果分开，不复活已销毁秘密；JS清零仅best effort。 |
| F08/F10 | PASS。实际production Compose渲染转发5项字段，stock YAML仅覆盖绑定地址/端口后原生进程ready200。 |
| 旧schema/密钥 | PASS所测兼容控制。真实产品数据克隆到旧schema，缺字段启动拒绝、无自动DDL；仅隔离副本显式加列后未知seq/v0不猜测补签。AES重包与旧HMAC根保留通过，不等于完整根退休资格。 |

三轮Go真实公平对局流水302.137秒通过：4个正常认证账号中3席操作、独立进程套件恢复、私牌互斥及第四账号403，真实WSS动作直至归档；覆盖真实SQL INSERT/COMMIT前强杀回滚、COMMIT后丢回复/强杀、终局回复丢失、原请求重放、owner takeover与SETTLING补偿。另两轮是实际Cocos owner手势对局，不能把两位Node陪测账号说成三台微信手机。

## 两套构建、实际微信运行与工具条件

### 已执行的Creator3.8链路

独立重跑既有工作流，run41937356536/job126966135979成功，新artifact13376959190，2026-11-05 00:39:45 UTC生成；ZIP SHA256：
`ee8203c5feb7cc40ca6a07d99fc430902686fbd18b031556651f447caec54705`。
[原生Creator构建任务](https://github.com/oublie6/awesome-zero-platform/actions/runs/41937356536/job/126966135979)。

实际checkout是 `1ca40ab75bc6fa430bc80203ffff99e3b231f863`；已独立核对该SHA至e071的 `clients/`、`scripts/`及该Cocos workflow差异为空。不能说整仓e071被该任务checkout；实际全部客户端构建输入一致，后端则使用e071的本轮二进制。两SHA完整差异另有无关的一次性格式化workflow删除，不能谎称整个workflows目录相同。

官方Creator3.8.8-win-121518，Web/Wx/Douyin均exit36；生成配置TS与CI浏览器步骤通过。下载后在Linux本地用原仓库浏览器验证脚本实际运行Web三视口/字体/场景销毁，10.206秒通过；新微信产物实际SDK运行如上。

新工具导入时的增强编译会再次把BigInt幂降为Math.pow，初次启动失败；私有项目显式配置enhance=false/es6=false后使用原生BigInt。20个生成JS文件与下载ZIP逐字相同，没有手改生成代码；urlCheck保持true。该条件是实际运行前提，不是默认导入即通过，也不构成正式域名资格。按用户后续指示，此链路证据存档，不再作为未来日常并行基线。

### Linux原生新链路

官方cocos-cli标签0.2.0-beta.0，commit `f506fa3de403b48b4833334fa48dddd27bb26364`；配套engine package4.0.0-beta.0、engine checkout `d0a696cfe63fd8d19c70d7913ba11aa641c2e3c7`，运行时版本标识4.0.0。官方[发布记录](https://github.com/cocos/cocos-cli/releases/tag/0.2.0-beta.0)与本机固定版本分别核对；不是“Creator3.8.8的Linux原生CLI”。

私有候选副本导入前89个tracked客户端文件逐字一致；导入后88个仍逐字一致，package.json仅格式/键序、JSON语义一致。主仓库creator声明没有被升级。使用既有2D/UI引擎缓存与固定WECHAT扩展，显式调用原生Node/CLI、真实生成引擎声明、项目strict选项；**未使用以前的私有tsc wrapper**。引擎工作目录只有导入器生成的5个meta变化，没有修改引擎实现来通过验收。

实际WECHAT导出exit0，228.202秒；3GiB/无swap/1CPU/600秒上限。导出原始包3074169字节，配置AppID/SDK前按文件排序的manifest SHA256：
`336ea9c2bf0a9355980ed63389f9073ef4e7e221957b94a5741d880f4e24533a`。
4项平台/生成入口/严格配置/BigInt AST审计测试在明确CLI Babel依赖搜索路径后通过。实际SDK4引擎场景、原生熵、微信认证、公平揭示、完整动作与证据及进程故障均执行；发现的是同一R8-N03状态缺陷，没有把运行退出1写成整套产品PASS。

范围限当前本地不拆包的2D/UI项目。上游未用FBX/HDR导入器的Linux支持限制、3D、远程包/分包/引擎分离、抖音Linux导出与真机均未因此获得资格；不靠外系统兼容补齐。官方CLI和项目扩展、正式Creator3.8及未来正式Linux基线不能混为同一产品版本。

### 微信身份与会话补充

实际SDK补充13.916秒通过：wx.login真实code→当前Go→官方换证、两次code稳定同一AccountID、保存后恢复并校验session、真实WSS及pong、同ID建房回执duplicate、正常退出API、全部自有token吊销。另测32字节安全随机数两次不同、SDK自有存储往返/删除、前后台监听注册/移除。没有伪造身份或把缺少AppID/登录当作BLOCKED理由。该补充以当前TS组件在真实SDK调用，和生成产物手势测试的证据边界明确分开。

## 自动化执行结果

常规Go命令均在server下，带 `-count=1 -p 1 -parallel 1`；Race另加 `-race`。详细工作目录、UTC、参数和退出码见E/runs.jsonl。

| 执行 | 实际结果 | 秒 |
| --- | --- | ---: |
| 全量Go JSON | 562项PASS（含父组，520叶），skip1 | 77.351 |
| 全量Go JSON Race | 同562项通过，skip1，无数据竞争告警 | 137.273 |
| go vet -p 1 ./... | exit0 | 7.376 |
| 当前app-api / maintenance构建 | exit0 / exit0 | 11.727 / 2.520 |
| verify-client-foundation.sh | 严格类型与31+37+36=104项通过 | 33.408 |
| APP_API_INTEGRATION=1 DOUDIZHU_SCALE_PROFILE=1 go test -tags=integration ./... | 601项PASS（556叶），skip1；含100/1000手 | 86.747 |
| 原始生命周期overlay + Race | 20项通过（16叶） | 9.294 |
| 扩展MySQL overlay首次全组 | exit1；3个缺产品夹具的失败，其他63项通过；不记整次PASS | 86.077 |
| 真实归档/清理/检查点补跑 | 归档边界与完整性通过；首次检查点读错DB仍exit1，纠正后单项exit0 | 4.297 / 4.611 |
| MySQL扩展最终去重断言 | 73项通过、66叶；由原运行+必要补跑组成，不伪造一次全绿记录 | — |
| TS HPKE / 原始客户端异步反例 | 9项 / 14项通过 | 4.061 / 2.825 |
| TS→Go envelope、Go→TS签名manifest | 实际互操作通过；解除默认skip的相关证据 | — |
| fmt-check / api-contract-check | exit0 | 0.846 |
| 隔离精确源码make generate + go mod tidy | 300个tracked server文件逐字不变 | 3.248 |
| 三轮真实Go/WSS/SQL对局及故障流水 | exit0；含预期丢终局回复的受控子进程exit1 | 302.137 |
| 原普通重试 / 原拒绝冷回执 | exit0 / exit0 | 8.477 / 0.691 |
| 1500 WSS lifetime | exit0，另60预热 | 30.278 |
| 原生产Dockerfile/Compose运行 | exit0，隔离端口/项目，全部原断言 | 338.700 |
| 新3.8实际Web浏览器 | exit0 | 10.206 |
| Linux平台4项 / 原生微信导出 | exit0 / exit0 | 0.958 / 228.202 |
| 实际微信身份/重试补充 | exit0 | 13.916 |
| 新提交/回调期限窗口 | **exit1，R8-N01** | 5.393 |
| 实际3.8核心UI / 终局继续验证 | **exit1，终局快照等待失败；独立核对与原生证据继续完成，R8-N03** | 305.802 / 22.143 |
| 实际Linux核心UI完整执行 | **exit1，24检查完成，20出牌/1过牌/真实归档与证据通过，完成状态断言失败** | 259.224 |

已执行CI元数据独立读取：候选e071的Fair Doudizhu run41938071334四项成功、CI run41938071397六项成功。CI与上表独立实测分别记录，绿色不替代本轮FAIL。

保留的准备/探针失败及纠正：最初相对路径复制、config探针未注入路径/密钥、真实对局前就运行依赖归档的探针、checkpoint误用integration而非product DB、Babel搜索路径缺失、SDK增强编译、重编译时context更换、前一探针清理已吊销陪测session。分别用正确私有配置/夹具、真实重新认证及稳定context重跑，没有修改断言来掩盖产品失败。初次SDK无scene探针exit0但JSON为BLOCKED，**未计PASS**。3.8最后动作的私牌快照等待不符合实际归档后接口路径，随后独立查真实归档并继续实际证据按钮；Linux探针继续验证分数，但仍明确断言终局状态失败。所有原日志保留。

## 性能、资源及稳定性

本机有限样本，不是生产容量或SLA：

| 场景 | P50 / P95 / P99或观测 |
| --- | --- |
| 80次真实COMMIT-before-ACK | 7.107 / 10.511 / 16.767 ms |
| 100手/16worker提交 | 14.339 / 22.212 / 25.793 ms；冷恢复1.144s |
| 1000手/16worker提交 | 18.934 / 46.357 / 60.774 ms；冷恢复14.487s；post-GC heap增约3.04MB |
| 1000手新Bid / Play / Pass P99 | 38.900 / 35.167 / 32.388 ms；新Bid最大688.196ms，不能只报分位数隐藏尾部 |
| 1000手历史Bid / Play / Pass P99 | 80.640 / 168.749 / 93.472 µs；分别57/78/60 allocations/query |
| 1600同ID争用/16worker（1000手配置） | P50 43.217µs / P95 1.234ms / P99 1.810ms，max5.958ms；不增加日志行 |
| 实际v33加密checkpoint、watermark60、76939B、100次写 | 6.616 / 14.009 / 17.500 ms；max17.986ms |
| 1500 WSS + 60预热、15秒冷却 | active=0，goroutine26→27，heap约3.77→5.53MB、RSS约29.72→32.06MB；采样中自然GC回落，未强制GC |

完整clone/长动作及回执profile见原日志；没有用短连接测量证明房间/长历史资源无限期有界。千手恢复显著超过原则中的初始建议目标1秒，提交P99也有超过建议50ms的观测；这些是未确认预算的实测事实，不将建议目标说成已满足。正式容量、fsync/SLA及持续稳定性资格仍列G02/GOAL7。

## 原始114项矩阵

每项按本轮源码、实测与明确证据范围重评。PASS不自动上升为真机/发布PASS；AUDIT的PASS表示完成相应审计交付。计数为**99 PASS / 9 FAIL / 6 BLOCKED**。

相对Round7的94/9/11：A07、H07由FAIL→PASS（真实选牌出牌/过牌与重连恢复）；H04、H05、AUTO10由BLOCKED→PASS（新原生Creator导出、输入身份核对与真实产物验证）；GOAL3由BLOCKED→PASS（实际公平流程和原生归档证据）；A09由PASS→FAIL、H08由BLOCKED→FAIL（实际终局界面未完成）。其他状态不变也重新检查；A02/A08/A10/H06/GOAL2/4/6的失败原因以本轮3个问题为准，不能直接继承Round7原因。

| ID | 原始要求 | Round8 | 本轮依据/边界 |
| --- | --- | --- | --- |
| A01 | 真实身份认证和账号隔离 | **PASS** | 实际wx.login→官方换证、重复稳定账号、session恢复/吊销及4个正常Go账号隔离。 |
| A02 | 创建、加入、退出房间 | **FAIL** | Go房间API及原生加入/创建/准备通过；Cocos没有退出入口，R8-N02。 |
| A03 | 三人准备和开局 | **PASS** | 实际三席/三账号准备，原生StartHand及三轮Go对局。 |
| A04 | 公平承诺与揭示 | **PASS** | 原生安全熵、外部私有备份、commit/import、签名Manifest/AAD/HPKE揭示。 |
| A05 | 公共随机源验证及发牌 | **PASS** | 5轮真实公平流程与drand/BLS验证、51张互斥私牌；临时relay500后正常重试。 |
| A06 | 地主竞叫及确定 | **PASS** | 真实Go规则与原生叫3分确定地主通过，原计时乱序复现通过。 |
| A07 | 出牌、过牌、牌型比较 | **PASS** | 两套SDK产物真实选牌/出牌；Linux20次出牌/1次过牌，无owner代打；规则全回归。 |
| A08 | 回合切换 | **FAIL** | 正常回合和旧回调单调性通过；提交后回调延迟可使旧期限终止，R8-N01。 |
| A09 | 终局、计分及证据归档 | **FAIL** | 服务端终局/归档与原生分数证据通过；客户端仍显示旧PLAYING状态，R8-N03。 |
| A10 | 断线重连及权威状态同步 | **FAIL** | 原生实际SIGKILL后版本20→20/18→18与epoch推进通过；终局重入未映射完成状态，R8-N03。 |
| B01 | 已向客户端确认的操作必须先完成持久化提交。 | **PASS** | 601项Go集成与73项独立MySQL断言；真实前/后COMMIT强杀、原回执、checkpoint/epoch/篡改隔离及终局恢复，存储完好范围。 |
| B02 | 任何已确认操作不能在进程崩溃后丢失。 | **PASS** | 601项Go集成与73项独立MySQL断言；真实前/后COMMIT强杀、原回执、checkpoint/epoch/篡改隔离及终局恢复，存储完好范围。 |
| B03 | 同一 requestId 的重复请求不能重复执行。 | **PASS** | 601项Go集成与73项独立MySQL断言；真实前/后COMMIT强杀、原回执、checkpoint/epoch/篡改隔离及终局恢复，存储完好范围。 |
| B04 | 相同 requestId、不同参数或不同操作者必须拒绝。 | **PASS** | 601项Go集成与73项独立MySQL断言；真实前/后COMMIT强杀、原回执、checkpoint/epoch/篡改隔离及终局恢复，存储完好范围。 |
| B05 | 新服务接管后，旧 Owner Epoch 必须被正确隔离。 | **PASS** | 601项Go集成与73项独立MySQL断言；真实前/后COMMIT强杀、原回执、checkpoint/epoch/篡改隔离及终局恢复，存储完好范围。 |
| B06 | 检查点、日志、水位线和业务版本必须严格一致。 | **PASS** | 601项Go集成与73项独立MySQL断言；真实前/后COMMIT强杀、原回执、checkpoint/epoch/篡改隔离及终局恢复，存储完好范围。 |
| B07 | 检查点或日志被篡改时必须拒绝恢复。 | **PASS** | 601项Go集成与73项独立MySQL断言；真实前/后COMMIT强杀、原回执、checkpoint/epoch/篡改隔离及终局恢复，存储完好范围。 |
| B08 | 恢复不能泄露其他玩家的私有手牌。 | **PASS** | 601项Go集成与73项独立MySQL断言；真实前/后COMMIT强杀、原回执、checkpoint/epoch/篡改隔离及终局恢复，存储完好范围。 |
| B09 | 终局重启必须保留最终证据和历史回执。 | **PASS** | 601项Go集成与73项独立MySQL断言；真实前/后COMMIT强杀、原回执、checkpoint/epoch/篡改隔离及终局恢复，存储完好范围。 |
| B10 | MySQL COMMIT 前后发生 SIGKILL、网络断开、响应丢失时，不得出现重复执行或状态分叉。 | **PASS** | 601项Go集成与73项独立MySQL断言；真实前/后COMMIT强杀、原回执、checkpoint/epoch/篡改隔离及终局恢复，存储完好范围。 |
| B11 | 真实 MySQL 事务 | **PASS** | 601项Go集成与73项独立MySQL断言；真实前/后COMMIT强杀、原回执、checkpoint/epoch/篡改隔离及终局恢复，存储完好范围。 |
| B12 | 并发重复提交 | **PASS** | 601项Go集成与73项独立MySQL断言；真实前/后COMMIT强杀、原回执、checkpoint/epoch/篡改隔离及终局恢复，存储完好范围。 |
| B13 | Epoch fencing | **PASS** | 601项Go集成与73项独立MySQL断言；真实前/后COMMIT强杀、原回执、checkpoint/epoch/篡改隔离及终局恢复，存储完好范围。 |
| B14 | 检查点加密 | **PASS** | 601项Go集成与73项独立MySQL断言；真实前/后COMMIT强杀、原回执、checkpoint/epoch/篡改隔离及终局恢复，存储完好范围。 |
| B15 | 日志完整性 | **PASS** | 601项Go集成与73项独立MySQL断言；真实前/后COMMIT强杀、原回执、checkpoint/epoch/篡改隔离及终局恢复，存储完好范围。 |
| B16 | 三人完整对局恢复 | **PASS** | 601项Go集成与73项独立MySQL断言；真实前/后COMMIT强杀、原回执、checkpoint/epoch/篡改隔离及终局恢复，存储完好范围。 |
| B17 | 进程级故障注入 | **PASS** | 601项Go集成与73项独立MySQL断言；真实前/后COMMIT强杀、原回执、checkpoint/epoch/篡改隔离及终局恢复，存储完好范围。 |
| B18 | 终局证据恢复 | **PASS** | 601项Go集成与73项独立MySQL断言；真实前/后COMMIT强杀、原回执、checkpoint/epoch/篡改隔离及终局恢复，存储完好范围。 |
| C01 | 新记录使用当前主密钥加密 | **PASS** | 真实MySQL密钥/密文/边界/epoch/重包负例通过；完整HMAC根退休另列C09/C10。 |
| C02 | 旧记录可以使用保留的历史密钥读取 | **PASS** | 真实MySQL密钥/密文/边界/epoch/重包负例通过；完整HMAC根退休另列C09/C10。 |
| C03 | 历史密钥数量和长度严格受限 | **PASS** | 真实MySQL密钥/密文/边界/epoch/重包负例通过；完整HMAC根退休另列C09/C10。 |
| C04 | 旧密钥不存在时拒绝错误解密 | **PASS** | 真实MySQL密钥/密文/边界/epoch/重包负例通过；完整HMAC根退休另列C09/C10。 |
| C05 | 重加密必须先认证旧密文 | **PASS** | 真实MySQL密钥/密文/边界/epoch/重包负例通过；完整HMAC根退休另列C09/C10。 |
| C06 | 种子重加密不得改变原始随机贡献 | **PASS** | 真实MySQL密钥/密文/边界/epoch/重包负例通过；完整HMAC根退休另列C09/C10。 |
| C07 | 检查点重加密必须校验当前 Owner Epoch | **PASS** | 真实MySQL密钥/密文/边界/epoch/重包负例通过；完整HMAC根退休另列C09/C10。 |
| C08 | 错误密钥、篡改密文不能恢复 | **PASS** | 真实MySQL密钥/密文/边界/epoch/重包负例通过；完整HMAC根退休另列C09/C10。 |
| C09 | 重加密后仅保留新密钥仍可正确恢复 | **BLOCKED** | AES种子/checkpoint重包通过；全Journal HMAC换根及new-only完整恢复没有合格操作证据。 |
| C10 | 旧密钥未完成迁移前不得被错误淘汰 | **BLOCKED** | 已证实旧Journal HMAC仍依赖旧根；完整密钥库存、迁移/核准退休流程未验证。 |
| D01 | 活跃牌局不能删除 | **PASS** | 真实完整归档克隆、7项清理边界、dry-run/显式execute及删除中SIGKILL事务回滚，模块边界回归。 |
| D02 | 未归档牌局不能删除 | **PASS** | 真实完整归档克隆、7项清理边界、dry-run/显式execute及删除中SIGKILL事务回滚，模块边界回归。 |
| D03 | 不满足 30 天保留期不能删除 | **PASS** | 真实完整归档克隆、7项清理边界、dry-run/显式execute及删除中SIGKILL事务回滚，模块边界回归。 |
| D04 | 损坏或不完整的归档不能删除 | **PASS** | 真实完整归档克隆、7项清理边界、dry-run/显式execute及删除中SIGKILL事务回滚，模块边界回归。 |
| D05 | 最终版本和日志水位线不一致不能删除 | **PASS** | 真实完整归档克隆、7项清理边界、dry-run/显式execute及删除中SIGKILL事务回滚，模块边界回归。 |
| D06 | 历史日志数量不完整不能删除 | **PASS** | 真实完整归档克隆、7项清理边界、dry-run/显式execute及删除中SIGKILL事务回滚，模块边界回归。 |
| D07 | 不得直接越权读取其他业务模块数据库表 | **PASS** | 真实完整归档克隆、7项清理边界、dry-run/显式execute及删除中SIGKILL事务回滚，模块边界回归。 |
| D08 | 保留完整终局证据和必要审计信息 | **PASS** | 真实完整归档克隆、7项清理边界、dry-run/显式execute及删除中SIGKILL事务回滚，模块边界回归。 |
| D09 | 默认 dry-run 不得修改数据库 | **PASS** | 真实完整归档克隆、7项清理边界、dry-run/显式execute及删除中SIGKILL事务回滚，模块边界回归。 |
| D10 | 必须显式 `--execute` 才能执行清理 | **PASS** | 真实完整归档克隆、7项清理边界、dry-run/显式execute及删除中SIGKILL事务回滚，模块边界回归。 |
| D11 | 中途崩溃后不能出现违反一致性约束的部分删除 | **PASS** | 真实完整归档克隆、7项清理边界、dry-run/显式execute及删除中SIGKILL事务回滚，模块边界回归。 |
| E01 | 提交承诺前生成高熵随机贡献 | **PASS** | 原生安全熵/套件导入及三轮新进程恢复；错误身份/seat/hand/commitment与14项异步/未知结果探针通过；JS清零best effort。 |
| E02 | 恢复凭证包含完整且可验证的信息 | **PASS** | 原生安全熵/套件导入及三轮新进程恢复；错误身份/seat/hand/commitment与14项异步/未知结果探针通过；JS清零best effort。 |
| E03 | 用户备份后才允许提交承诺 | **BLOCKED** | 工具确实在commit前外部备份；真实人类备份确认和设备重启UX未验证。 |
| E04 | 应用退出后可以导入恢复凭证 | **PASS** | 原生安全熵/套件导入及三轮新进程恢复；错误身份/seat/hand/commitment与14项异步/未知结果探针通过；JS清零best effort。 |
| E05 | 恢复凭证与当前登录身份、Hand、Seat、Commitment 必须匹配 | **PASS** | 原生安全熵/套件导入及三轮新进程恢复；错误身份/seat/hand/commitment与14项异步/未知结果探针通过；JS清零best effort。 |
| E06 | 错误凭证必须拒绝 | **PASS** | 原生安全熵/套件导入及三轮新进程恢复；错误身份/seat/hand/commitment与14项异步/未知结果探针通过；JS清零best effort。 |
| E07 | 不能产生新的随机贡献冒充旧承诺 | **PASS** | 原生安全熵/套件导入及三轮新进程恢复；错误身份/seat/hand/commitment与14项异步/未知结果探针通过；JS清零best effort。 |
| E08 | 不能通过普通小游戏明文持久化自动泄漏随机数 | **PASS** | 原生安全熵/套件导入及三轮新进程恢复；错误身份/seat/hand/commitment与14项异步/未知结果探针通过；JS清零best effort。 |
| E09 | 释放对象时敏感数据应尽可能清零 | **PASS** | 原生安全熵/套件导入及三轮新进程恢复；错误身份/seat/hand/commitment与14项异步/未知结果探针通过；JS清零best effort。 |
| E10 | 请求丢响应时必须保留原 requestId 精确重试 | **PASS** | 原生安全熵/套件导入及三轮新进程恢复；错误身份/seat/hand/commitment与14项异步/未知结果探针通过；JS清零best effort。 |
| E11 | 不能因为未知结果直接生成新的操作请求 | **PASS** | 原生安全熵/套件导入及三轮新进程恢复；错误身份/seat/hand/commitment与14项异步/未知结果探针通过；JS清零best effort。 |
| F01 | 验证使用三个真实、不同的服务端认证身份 | **PASS** | 3轮正常Go身份/WSS完整公平对局+2套实际SDK owner手势对局；私牌403、归档与分数证据。非3微信手机，公平请求按现有HTTPS契约。 |
| F02 | 验证三人座位各不相同 | **PASS** | 3轮正常Go身份/WSS完整公平对局+2套实际SDK owner手势对局；私牌403、归档与分数证据。非3微信手机，公平请求按现有HTTPS契约。 |
| F03 | 验证每次操作使用真实 WSS | **PASS** | 3轮正常Go身份/WSS完整公平对局+2套实际SDK owner手势对局；私牌403、归档与分数证据。非3微信手机，公平请求按现有HTTPS契约。 |
| F04 | 验证客户端私有手牌访问权限 | **PASS** | 3轮正常Go身份/WSS完整公平对局+2套实际SDK owner手势对局；私牌403、归档与分数证据。非3微信手机，公平请求按现有HTTPS契约。 |
| F05 | 验证游戏完整进行到终局 | **PASS** | 3轮正常Go身份/WSS完整公平对局+2套实际SDK owner手势对局；私牌403、归档与分数证据。非3微信手机，公平请求按现有HTTPS契约。 |
| F06 | 验证最终结算及证据 | **PASS** | 3轮正常Go身份/WSS完整公平对局+2套实际SDK owner手势对局；私牌403、归档与分数证据。非3微信手机，公平请求按现有HTTPS契约。 |
| G01 | 并发牌局状态隔离 | **PASS** | 磁盘MySQL100/1000手、16worker提交/回执、P50/P95/P99/冷恢复及clone观测；562项Go Race，无生产SLA推断。 |
| G02 | 内存增长与回收 | **BLOCKED** | 1500 WSS关闭及自然GC观测完成；完整房间/长历史soak和回收资源上界尚未证明。 |
| G03 | MySQL 提交延迟 | **PASS** | 磁盘MySQL100/1000手、16worker提交/回执、P50/P95/P99/冷恢复及clone观测；562项Go Race，无生产SLA推断。 |
| G04 | P50/P95/P99 | **PASS** | 磁盘MySQL100/1000手、16worker提交/回执、P50/P95/P99/冷恢复及clone观测；562项Go Race，无生产SLA推断。 |
| G05 | 检查点及故障恢复耗时 | **PASS** | 磁盘MySQL100/1000手、16worker提交/回执、P50/P95/P99/冷恢复及clone观测；562项Go Race，无生产SLA推断。 |
| G06 | 高并发下连接池和锁竞争 | **PASS** | 磁盘MySQL100/1000手、16worker提交/回执、P50/P95/P99/冷恢复及clone观测；562项Go Race，无生产SLA推断。 |
| G07 | Race 检查 | **PASS** | 磁盘MySQL100/1000手、16worker提交/回执、P50/P95/P99/冷恢复及clone观测；562项Go Race，无生产SLA推断。 |
| G08 | Goroutine 泄漏 | **PASS** | 1500次+60次预热，活跃连接归零，goroutine26→27；不等于长期全业务无泄漏。 |
| G09 | 重复请求及失败重试压力 | **PASS** | 磁盘MySQL100/1000手、16worker提交/回执、P50/P95/P99/冷恢复及clone观测；562项Go Race，无生产SLA推断。 |
| G10 | 长牌局的候选状态复制开销 | **PASS** | 磁盘MySQL100/1000手、16worker提交/回执、P50/P95/P99/冷恢复及clone观测；562项Go Race，无生产SLA推断。 |
| H01 | TypeScript 类型检查 | **PASS** | 104项基础检查含严格TS；Creator生成类型CI通过、Linux构建实用真实声明且无旧tsc shim。 |
| H02 | Web 实际构建 | **PASS** | 新Creator3.8同输入Web导出exit36；本地实际Chromium运行通过。 |
| H03 | Chromium 场景冒烟测试 | **PASS** | 实际下载的3.8 Web产物三视口、字体、场景/销毁检查，10.206秒通过。 |
| H04 | 微信小游戏构建 | **PASS** | 独立重跑Creator3.8新微信产物及Linux原生WECHAT实际导出；SDK实际动作另评。 |
| H05 | 抖音小游戏构建兼容性 | **PASS** | 独立重跑Creator3.8抖音导出exit36；未声称抖音SDK/设备运行通过。 |
| H06 | 游戏 UI 操作与真实网络请求一致 | **FAIL** | 动作/网络及合法出牌已验证；最后动作归档v65/v63时UI仍是v64/v62，R8-N03。 |
| H07 | 重连及恢复凭证操作可触达 | **PASS** | 原生恢复套件导入、主动重连、多回调清理及真正进程重启后私牌版本恢复通过。 |
| H08 | 游戏页面布局、按钮可见性和状态刷新 | **FAIL** | 截图与真实状态均显示终局标题仍“出牌中”/重入空牌桌，R8-N03；真机布局另未验证。 |
| AUTO01 | Go 单元测试 | **PASS** | 本轮独立实际执行Go/集成/Race/Vet/fmt/API生成/TS104/HPKE9/互操作/模块边界；命令与退出码见执行表。 |
| AUTO02 | Go Race 测试 | **PASS** | 本轮独立实际执行Go/集成/Race/Vet/fmt/API生成/TS104/HPKE9/互操作/模块边界；命令与退出码见执行表。 |
| AUTO03 | Go Vet | **PASS** | 本轮独立实际执行Go/集成/Race/Vet/fmt/API生成/TS104/HPKE9/互操作/模块边界；命令与退出码见执行表。 |
| AUTO04 | Go fmt-check | **PASS** | 本轮独立实际执行Go/集成/Race/Vet/fmt/API生成/TS104/HPKE9/互操作/模块边界；命令与退出码见执行表。 |
| AUTO05 | MySQL 集成测试 | **PASS** | 本轮独立实际执行Go/集成/Race/Vet/fmt/API生成/TS104/HPKE9/互操作/模块边界；命令与退出码见执行表。 |
| AUTO06 | 100/1000 牌局专项测试 | **PASS** | 本轮独立实际执行Go/集成/Race/Vet/fmt/API生成/TS104/HPKE9/互操作/模块边界；命令与退出码见执行表。 |
| AUTO07 | 前端 TypeScript 类型检查 | **PASS** | 本轮独立实际执行Go/集成/Race/Vet/fmt/API生成/TS104/HPKE9/互操作/模块边界；命令与退出码见执行表。 |
| AUTO08 | 客户端单元测试 | **PASS** | 本轮独立实际执行Go/集成/Race/Vet/fmt/API生成/TS104/HPKE9/互操作/模块边界；命令与退出码见执行表。 |
| AUTO09 | 安全组件测试 | **PASS** | 本轮独立实际执行Go/集成/Race/Vet/fmt/API生成/TS104/HPKE9/互操作/模块边界；命令与退出码见执行表。 |
| AUTO10 | Cocos Creator Smoke | **PASS** | 新3.8 Web/Wx/Douyin导出与本地浏览器实测；按实际输入身份绑定，不冒充整仓e071 checkout。 |
| AUTO11 | Production Compose Runtime | **PASS** | 原Dockerfile/Compose隔离部署的bootstrap/auth/HTTP/WS/TLS/WSS全断言通过，338.700秒。 |
| AUTO12 | 数据库架构边界测试 | **PASS** | 本轮独立实际执行Go/集成/Race/Vet/fmt/API生成/TS104/HPKE9/互操作/模块边界；命令与退出码见执行表。 |
| AUDIT01 | P0：数据丢失、敏感信息泄露、身份越权、不可恢复的数据破坏 | **PASS** | 已交付本级审计及3个独立问题的位置、触发、复现、实际/预期、影响与建议；PASS为审计交付，非零缺陷。 |
| AUDIT02 | P1：重复出牌、状态分叉、错误结算、恢复失败、死锁及关键链路中断 | **PASS** | 已交付本级审计及3个独立问题的位置、触发、复现、实际/预期、影响与建议；PASS为审计交付，非零缺陷。 |
| AUDIT03 | P2：边界条件、资源泄漏、错误处理、用户操作异常、可维护性问题 | **PASS** | 已交付本级审计及3个独立问题的位置、触发、复现、实际/预期、影响与建议；PASS为审计交付，非零缺陷。 |
| AUDIT04 | 严重级别 | **PASS** | 已交付本级审计及3个独立问题的位置、触发、复现、实际/预期、影响与建议；PASS为审计交付，非零缺陷。 |
| AUDIT05 | 文件及代码位置 | **PASS** | 已交付本级审计及3个独立问题的位置、触发、复现、实际/预期、影响与建议；PASS为审计交付，非零缺陷。 |
| AUDIT06 | 触发条件 | **PASS** | 已交付本级审计及3个独立问题的位置、触发、复现、实际/预期、影响与建议；PASS为审计交付，非零缺陷。 |
| AUDIT07 | 可复现步骤 | **PASS** | 已交付本级审计及3个独立问题的位置、触发、复现、实际/预期、影响与建议；PASS为审计交付，非零缺陷。 |
| AUDIT08 | 实际行为 | **PASS** | 已交付本级审计及3个独立问题的位置、触发、复现、实际/预期、影响与建议；PASS为审计交付，非零缺陷。 |
| AUDIT09 | 预期行为 | **PASS** | 已交付本级审计及3个独立问题的位置、触发、复现、实际/预期、影响与建议；PASS为审计交付，非零缺陷。 |
| AUDIT10 | 建议修复方向 | **PASS** | 已交付本级审计及3个独立问题的位置、触发、复现、实际/预期、影响与建议；PASS为审计交付，非零缺陷。 |
| GOAL1 | Real platform-code login exchange / session validation, token lifetime and secure private credential management; no fake identity, no tokens in URLs; include local authorized dev login path only if correctly access-controlled. | **PASS** | 实际微信官方换证/稳定身份/session恢复与注销/WSS/普通同ID重试通过，私有凭据。 |
| GOAL2 | Account-scoped rooms: create/join/leave/ready, seats, start hand, authoritative room refresh/reconnect and timeout/duplicate reconciliation; full Cocos interactive flow. | **FAIL** | 后端与原生创建/加入/准备/开局通过；房间退出UI缺失，R8-N02。 |
| GOAL3 | Fair lifecycle: server-bound commit/reveal key/AAD acquisition, secure entropy + HPKE, trusted beacon, deal, private hand and evidence verification, fail closed on missing security context. | **PASS** | 原生Manifest/AAD/HPKE/drand/私牌及实际核验归档分数摘要通过；人类备份UX单列E03。 |
| GOAL4 | Playable game: bidding, landlord determination, legal play/pass delegated to authoritative Go rules, turn indicators, cards/selection, clear error and version-retry handling, termination, score/evidence/result UI. | **FAIL** | 实际手势对局到服务端终局且分数证据可见；期限窗口与终局状态展示仍失败，R8-N01/N03。 |
| GOAL5 | B-level MySQL recovery: append each accepted live bid/play/pass state transition and dedup + owner epoch/fence atomically **before acknowledgement**; bounded encrypted checkpoint and ordered replay restoring game state, ambiguity handling and owner fencing; address seed and other secret continuity; no plaintext player hands in routine logs. | **PASS** | MySQL commit-before-ack、checkpoint、全序日志/HMAC/去重、epoch及真进程故障回归通过。 |
| GOAL6 | Real 3-player repeatable E2E, reconnect and crash-injection: pre-/post-COMMIT kill, response loss, duplicate/out-of-order commands, competing owners, wrong epoch and replay mismatch; preserve every acknowledged action. | **FAIL** | 三Go身份/多局/故障与两套原生SDK主动及冷恢复通过；完整Cocos终局状态收尾仍失败，R8-N03。 |
| GOAL7 | Performance and stability: bounded queues/workers/room memory, P50/P95/P99 measurements and transaction fsync budget on stated hardware, no unknown target claims. | **BLOCKED** | 已测尾延迟/冷恢复/复制/WSS资源；缺房间/长历史资源上界与正式容量/fsync预算资格。 |
| GOAL8 | WeChat Creator package, actual developer tool, approved HTTPS/WSS domains and real handset evidence where access is available. Record platform-owned AppID/domain/device blockers honestly. Preserve Web and Douyin build compatibility. | **BLOCKED** | AppID/Secret/工具登录已具备且实际通过；核准生产域名/TLS/白名单、真机和抖音SDK/设备未具备。 |

## 未完成条件、清理及最终结论

仍未验证：完整Journal HMAC根迁移后的new-only恢复与核准根退休；人类外部备份确认/真实设备重启恢复UX；房间与长历史持续资源上界/正式容量预算；核准生产HTTPS/WSS域名、生产TLS/白名单、实际手机完整交互/生命周期及抖音SDK/设备。AppID、AppSecret、工具登录与安全随机数均已可用，不能再以缺少这些解释BLOCKED。Linux新链路的3D/FBX及未实现导出模式不在本轮已验证2D范围内。

收尾已调用自有supervisor.shutdown，停止codex-r8-wechat-final/backend及codex0046r8 TLS/MySQL/Redis容器；原生产Compose运行自清理测试项目。自有端口均关闭，只剩原先learn-xray容器；两个Kubernetes节点Ready，kubelet/containerd active；本机约4512MiB available、无swap。保留私有证据/测试卷/已安装原生工具与用户profile，不碰旧验收卷、不做全局Docker清理。

报告及AGENTS政策做了独立的文档第二遍自审：核对114唯一ID/状态计数、初次失败和补跑、SQL事实与受控port、构建源码/产物版本、3.8与Linux证据边界、秘密与修改范围；这项文档自审不冒充第二位产品验收者。

**内部代码验收FAIL**：R7原始乱序修复通过，但R8-N01/P1阻断完整生命周期资格；另有R8-N02/P2房间退出UI和R8-N03/P2终局界面状态问题。原R7-N04与WX-N02可关闭。无已证实P0发现，不代表穷尽所有安全风险。

**完整客户端验收FAIL；外部平台完整适配BLOCKED；正式发布不具备资格；Goal0045/0046保持in_progress，不封存。** 实际原生手势已可完成对局及核对分数证据，不能把剩余UI和生命周期失败隐去。交回ChatGPT修复产品，并按用户新政策实施单Linux构建基线迁移、同步版本/CI/文档，再冻结新SHA验收。Codex本轮没有代修产品。

