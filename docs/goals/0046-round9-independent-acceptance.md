# Goal 0046 — Round 9 全量独立验收

日期：2026-11-05（UTC）。本轮命令记录07:05:55开始，最后完整runtime门禁08:38:40结束；报告/资源收尾于当日16:37后完成。执行者：Codex。**内部代码/集成产品验收：FAIL；89 PASS / 20 FAIL / 5 BLOCKED = 114。** 新确认8个独立问题（3 P1、5 P2，未建立 P0）。20个失败条目不是20个独立缺陷。外部平台/真机资格仍 BLOCKED，正式发布不合格，Goal0045/0046保持 `in_progress`。

## 1. 固定版本、范围与执行环境

产品候选与全部本轮正式执行记录的源码均为 **`dbfb21475ab5d818913c84bfec63353473a8bb92`**。验收开始及报告前确认 clean `main`，`git pull --ff-only origin main` 无新增提交。报告提交是后续独立的文档提交；可用 `git log -1 --format=%H -- docs/goals/0046-round9-independent-acceptance.md` 查询，不能把它作为重新构建过的产品版本。

已阅读 AGENTS、全部相关原则、current、0046 integrated、Round1–8历史验收（Round1–3在current历史段）、Linux Cocos操作文档和 PLATFORM-VALIDATION。按照原始114条目审计产品，不修改业务实现、正式测试、协议、schema或CI；只有本报告及current允许的状态段更新。未运行旧Creator3.8/Wine/其他OS虚拟机，未自行修复发现的产品缺陷。

| 项目 | 实际环境与边界 |
|---|---|
| 主机 | Linux x86_64，约7.75GiB物理内存；swap=0；所有重型任务顺序执行 |
| 服务端 | Go1.25.8，GOMAXPROCS=2，`-p 1 -parallel 1`；current候选编译的app-api/maintenance |
| 数据库 | 隔离Docker项目codex0046r9；真实MySQL5.7.44，innodb_flush_log_at_trx_commit=1、binlog关闭、64MiB buffer pool；Redis8.8.0辅助 |
| 规模/连接 | MySQL容器768MiB/1.5CPU；专项16 workers、池上限24；产品app池16；资源与延迟不是生产容量认证 |
| 数据所有权 | integration库awesome_zero_platform、实际游戏库codex0046r9_product、旧schema兼容库codex0046r9_upgrade，均在本轮自有隔离容器 |
| HTTP/WSS | 自有SAN证书/CA及明确SPKI信任，HTTPS/WSS真实鉴权；SDK `urlCheck=true`，不关闭TLS验证；不等于核准生产域名/白名单 |
| 客户端 | Node22.23.4、官方cocos-cli与engine固定revision、现有community Linux微信开发工具2.02.2609240-2、SDK3.17.5 |
| 测试身份 | 三局正常服务端认证Go账号；原生局为真实wx账号+两个正常认证Go peers，合计三个不同身份；不是三台微信手机 |
| 微信配置 | 已有用户授权AppID/AppSecret可用、工具已登录；上传密钥未使用，未上传/发布/修改生产配置 |

私有证据根目录记为 **`$E=/root/.config/awesome-zero-platform/acceptance-0046-r9-dbfb214`**。下文短文件名均相对该目录；临时Go overlay只加入外置探针，实际产品包使用固定源码。`runs.jsonl`保存命令、cwd、UTC起始、耗时、退出码和完整源码SHA；不能把脚本存在当执行证据。测试计数包含parent/subtest，互有覆盖，不相加宣称独立用例总数。

候选SHA的[CI / ci/full](https://github.com/oublie6/awesome-zero-platform/actions/runs/41964933529) **6/6 SUCCESS**、[Fair Doudizhu](https://github.com/oublie6/awesome-zero-platform/actions/runs/41964933293) **4/4 SUCCESS**，已检查各job/step，原始元数据在 `ci-exact-source.json`。该候选没有可据以声称通过的Cocos native-export Actions运行；hosted源码检查、缺席/跳过的native job不是导出成功。以下Linux导出和SDK是本轮实际本地执行证据。CI绿不能覆盖下述产品失败。

## 2. R8-N01：提交与生命周期超时竞争

**原始复现 PASS；扩展一致性验收 FAIL（R9-N01、R9-N07）。** 没有复现“已经提交的新操作被旧期限错误终止”的原始问题；不能因此判整个生命周期已通过。

| 真实执行的交错 | 观察与结论 | 证据 |
|---|---|---|
| t4有效动作实际提交，暂停返回/AfterBid，t6旧Sweep，释放回调，t7再Sweep | BIDDING、PLAYING、BIDDING→PLAYING三组均保留新版本、不旧超时、不回退/重复期限；PASS | original-lifecycle.log；expiry-sql-callback-all-phases.log（真实MySQL/Service/runtime/Supervisor） |
| Sweep旧决定之后，新动作先提交 | ErrLiveProgressed回滚，epoch不变，watermark更新；PASS | expiry-sql-final-real-archive.log |
| 域行已锁、writer锁尚未取得时新动作提交 | 新commit成功，随后旧expiry回滚；PASS | expiry-sql-final-real-archive.log |
| Expire先持writer行锁，新Bid并发 | 实际INNODB_LOCK_WAITS确认阻塞；expiry成功后旧epoch Bid拒绝，不产生终局后动作；PASS | expiry-sql-lock-order.log |
| Fence后基础设施失败 | SQL更新全部回滚、epoch不变、下一Bid成功；PASS | expiry-sql-final-real-archive.log |
| 手局行/Writer反向锁竞争 | 实际制造InnoDB死锁，expiry为victim，分类ErrRetryableTransaction、状态/水位/epoch/回执全部回滚；精确重试成功；PASS | expiry-sql-deadlock.log |
| 所有事务修改后、真实Store COMMIT前KILL本轮SQL连接 | 实际COMMIT invalid connection，无ACK、所有变更回滚、原writer可继续Bid；PASS | expiry-sql-connection-loss.log |
| 子进程在终止COMMIT前SIGKILL | tentative fence回滚，仅子进程已提交takeover epoch保留，冷恢复合法新Bid可提交；PASS | expiry-sql-sigkill-boundaries.log |
| 正常实际Sweep终止、同系统requestId重试与再Sweep | domain EXPIRED、一次fence、一个归档、一条终止回执，无额外终止；PASS | expiry-actual-sweep-once.log |
| stale ExpectedVersion、匹配ExpectedLiveVersion的内部Expire | 拒绝DDZ_VERSION_CONFLICT却提交epoch 2→3，domain仍BIDDING，下一Bid被fence；FAIL | expiry-sql-final-real-archive.log，N01 |
| 成功EXPIRED的原动作热/冷回执；终止COMMIT后、Archive前SIGKILL | 热回执正确，冷恢复原Bid失败；缺Archive且同ID重试不补；FAIL | expiry-sql-real-archive-cold.log、expiry-sql-sigkill-boundaries.log，N07 |

实际追踪：`terminateHand`在同一MySQL事务中通过 `LockLiveCommittedVersion` 和 `FenceLiveWriter` 串行化实时journal的writer；`DurabilityEnforced`实际命中，而非空port。新动作的 `LiveJournal.Append` 同锁校验epoch/水位，正确的成功/rollback交错通过。**业务拒绝被execute当成可提交结果**，暴露N01；COMMIT后外部runtime归档和冷恢复跳过EXPIRED暴露N07。受控port的原Dispatcher复现仅作补充，未代替真实数据库测试。

N01使用内部公开application方法及正常 `ClientSeq==ExpectedVersion` 形状，但**尚未证明正常生产Supervisor自然生成这一stale域版本+有效live水位组合**。不声称WSS用户可直接选择Expire（transport没有该type），也不把它称为原始旧超时复现仍失败。明确失败的契约是“被拒绝的终止不得错误fence仍活跃writer”。

历史Bid/Play/Pass不续期、重复hand.dealt不重置、新动作正常续期、旧回调单调性和SETTLING同进程补偿均重验，见第7节。

## 3. R8-N02：原生退出、未知结果与清理

**原始缺少按钮问题 PASS；扩展异步状态清理 FAIL（N06），关闭房间读取另有N05。** 实际生成的Linux微信产物运行在真实SDK，按钮可见并由原生触摸触发。未用后台API退出代替下列原生操作。

实际执行：登录→原生Create→最后owner原生Leave→服务端CLOSED/本地清空；原生Join他人创建房间；第一次Leave已由Go提交后只丢真实ACK→15秒未知提示→保留pending room.leave→第二次原生Leave发送**完整DTO deepEqual/相同requestId**→占用人数只少一人/本地清空；同会话再次Create成功。房主有其他成员时退出、活跃牌局退出明确拒绝；正常最后owner退出清空roomID/handID/选牌和shell。

`leave-fairness-real.json`另以真实三账号HTTPS/WSS执行：公平commit已接受但HTTP结果丢失，公平结果未知时Leave在发送前拒绝（leave帧=0）；原DTO重试解未知后才允许继续。原生退出未知结果测试与此controller测试分开记录。一般pending hand、版本冲突/成员失效等分支有正式客户端/外置生命周期测试，**未分别完成每个分支的独立原生GUI复现**；退出后旧事件/全部重新登录排列也未做穷举。不能将这些统称原生PASS。

原生清空以后释放先前已经真实收到的终局HTTP响应，旧HandStore及分数复活（N06）。最后owner关闭后GET原房间400（N05），退出本身在真实DB中已成功；不能用该400反推退出未提交，也不能用DB成功掩盖查询失败。见 `wechat-full-ui.json`、attempt1/attempt2原始日志和N05/N06步骤。

## 4. R8-N03：首局终态与下一局/重入

**上一轮首局终局仍PLAYING/空桌的原始复现 PASS；完整终态/房间生命周期 FAIL（N02/N04）。**

真实原生owner完成公平准备、私有外部备份、commit/import/reveal/发牌、叫3分、**20次原生出牌+1次原生过牌**；两个peer由实际Go规则通过认证WSS正常操作，owner无后台代打。中途whole-Go-SIGKILL，SDK观察断线后由原生Reconnect/同步恢复**已ACK版本19→19，epoch推进**。最终真实archive **v64、winnerSeat1、seatPoints 6/-3/-3**，digest为 `83321cde28489e0cb3dcb12815302e633452c08a813fabebbf7b87e7995af9ed`。

UI实际显示“公平斗地主 · 已结算”与服务端相同积分/胜者/摘要，无旧私牌、叫分/出牌/过牌按钮。`full-native-final.png`人工查看通过该范围。终局私牌HTTP404不阻止证据显示；BackRooms→Reconnect→OpenTable恢复COMPLETED；真实成员lastHandId和第四账号证据403通过。原Set编译/多socket回调注册、移除和断线重连检查通过。

随后**同一房间实际Start第二局**：Bootstrap指向新hand，但HandStore保留第一局COMPLETED，OpenTable仍显示旧分数（N02）。通过公开授权API取消未发牌第二局后，room.lastHandId指向无归档的ABORTED，原生Reconnect提示HTTP404（N04）。取消是测试清理/异常路径API，不宣称原生Cancel按钮通过。

终态handId/version/分数/digest验证、重复/过时消息的正式单测与本轮原生第一局正例均检查；部分组合仅组件/端口负例。**终局以后完整重启微信客户端进程、三台真机前后台/重登/重入的所有排列未验证**，本轮原生的进程故障指服务端Go进程，不冒充客户端重启。A10/完整产品仍FAIL，不据有限重入正例断言所有恢复通过。

## 5. Linux原生Cocos4实际构建与SDK

固定official CLI `f506fa3de403b48b4833334fa48dddd27bb26364`，engine checkout `d0a696cfe63fd8d19c70d7913ba11aa641c2e3c7`，package `4.0.0-beta.0`（SDK运行时显示4.0.0），Node22.23.4。复用 `/data/awesome-zero-platform-tools/round7-cocos-cli`，直接调用真实dist/cli.js，未通过旧全局cocos/tsc shim。platform-extension所有权与revision/pins、4个extension测试通过。

| 层次 | 实际结果 |
|---|---|
| 正式 `scripts/cocos-linux-wechat/build.cjs` | **FAIL**，82.372秒，wrapper退出1，真实CLI退出34；复制排除所有node_modules后只对Cocos项目npm ci，file-linked portable包缺hpke/@noble依赖。N03 |
| 隔离环境补充 | 只在保留的临时clients副本对三个portable包安装依赖；未改业务、引擎、测试断言。相同真实CLI `--platform wechatgame` **PASS**，261.012秒（内部build计时260.973秒），入口、实际WECHAT adapter、landscape config、资源、BigInt/HPKE/BLS及真实生成引擎strict检查通过 |
| 源码绑定 | `native-current-input-binding.json`核对124个tracked client输入，import后字节相同，只有package JSON格式变化；import metadata/UUID保留，不混用旧产物 |
| 实际微信SDK | 当前产物在Linux工具/SDK3.17.5真实运行、原生触摸、wx.login/安全熵/房间/公平/完整一局/重连均执行；整套扩展GUI因四个问题退出1，不重标整体PASS |
| 严格类型 | 微信真实生成声明检查通过；另用Web真实cc4绝对声明路径+项目原有Bundler/allowTSextensions选项执行strict/strictNull/noImplicit检查通过。初始私有checker漏项目选项失败记录保留，未用伪造cc声明或关闭资产严格检查 |
| Web | 当前官方CLI WEB_DESKTOP真实导出176.616秒退出0，但core CLI打印相对声明TS2688错误并过滤非assets诊断；原始包启动因Math.pow(BigInt)异常94.273秒退出1，不能称原包可运行 |
| Web完整已有配方 | 在独立产物副本运行仓库既有 `scripts/fix-cocos-bigint-power.cjs`，只对一个generated bundle做6处已知修正；原始产物保留。Chromium三viewport/字体/帧缓冲/销毁实际PASS8.676秒。属于已有后处理配方，不是验收方新写产品修复 |

真实微信原始包 **3,077,632B**；逐文件hash清单序列的manifest SHA256：`39fab99eba9f34adf5d32ba8c51ea6da2ac5093a38d17fb404671b521df50996`。哈希对应导出后、操作员SDK配置前，之后仅有已记录project SDK/AppID等操作配置，不冒充内容完全未变。`linux-native-provisioned-provenance.json`含逐文件hash。

Web raw manifest SHA256 `ef02c3f7a2edbaab7e5edcb9cc7a794a918916328b802a7ae79f694fcfaa949e`；已有后处理后的manifest `17c7a305498121204406e395428db0dbc5ab840279a096010e03fbd7cac63b75`。Web为23,529,956B调试产物，存在debug HUD/调试资源，**不具备发布包资格**。已人工查看portrait场景，SDK/缩放画面的小字及真实手机可读性未资格化。资源加载/文本/输入框/触摸/横屏可用不等于所有真机屏幕通过。

experimental Linux平台扩展早期由当前agent在用户授权下编写，对其源码的复查属于**自审**，不能称独立他人审查；本轮固定候选的真实导出/SDK/业务结果是新执行证据，未继承早期自测作为本轮成功。未虚构Douyin、3D、远程分包或beta engine的正式发布资格。

## 6. 真实三人闭环与B级恢复

`full-three-games.log`在313.183秒内完成三轮真实不同认证账号/席位：登录→建房/加入/准备→开局→提交前外部备份kit→新Node进程导入→签名Manifest/AAD/HPKE→真实Quicknet/BLS→互斥发牌→叫分→出牌/过牌→Go结算/归档/证据。另有第4节实际原生局；不是mock三人或三台微信手机。

- 第一局：原N01跨阶段历史Bid score3/clientSeq91，从BIDDING推进到PLAYING后真实强杀重启，精确原accepted receipt不变，正常完成archive v60。
- 第二局：低叫分、16路同ID并发、不同ID同序号只允许一个提交；故意丢最终结果再whole-Go-SIGKILL，精确终局回执和分数/摘要不变，archive v61。重启后Bid/Play/Pass七个序号/身份/类型绑定负例执行。
- 第三局：实际SQL事务COMMIT前强杀回滚、COMMIT后丢实际结果并强杀，精确重试不重复；真实SETTLING完成事务故障后移除注入，同进程必要补偿完成一次、五次重试无额外结算，archive v60。
- checkpoint加密/版本、水位、journal HMAC、genesis成员绑定、epoch takeover/fencing、错误actor/type/aggregate/payload/sequence/根/保留key负例均实际执行。Redis仅辅助，不作为逐动作权威；未引入Kafka替代MySQL。
- 正常COMPLETED冷恢复与终局重复通过，**不能外推EXPIRED**；N07真实Archive/进程边界测试是本轮反例。普通房间/fairness命令durable请求身份与空events[]冷重放也重验。

## 7. 历史问题回归

| 历史问题 | 本轮独立依据与范围 | 结论 |
|---|---|---|
| F01 signed manifest实际HTTP契约 | 当前原生reveal及三轮原客户端Manifest/AAD/HPKE成功；game1-reveal.log、wechat-full-ui.json | 原复现PASS |
| F02 / N01 COMPLETED跨阶段/终局历史回执 | original-N01.log、terminal-recovery.json、completed-history.json、original-mysql-regressions.log | 原COMPLETED复现PASS；EXPIRED扩展N07 FAIL |
| F03 genesis换座/私牌隔离 | original-mysql-regressions.log真实数据库篡改拒绝；实际非成员403 | 原复现PASS |
| F04 不完整归档允许清理 | 真实已完成对局克隆/语义证据破坏后拒绝，原有完整归档正对照与prune CLI强杀回滚 | 原复现PASS |
| F05 房间空seat解析 | 实际1/2/3人房间视图、原生Create/Join/Refresh通过；新增CLOSED读取N05单列 | 原复现PASS |
| F06 已知fairness拒绝锁死 | original-client-repro.log、client-lifetime-matrix.log已知拒绝与未知结果分离 | 原复现PASS |
| F07 dispose后prepare秘密复活 | 原异步prepare/dispose回归及best-effort清理；不等于immutable string物理擦除 | 原复现PASS |
| F08 五个Compose配置丢失 | deploy-wiring.json、真实production Compose runtime | 原复现PASS |
| F09 Quicknet要求legacy HMAC | 原Provider/StockEnvironment启动回归及真实Quicknet使用，无HMAC替代 | 原复现PASS |
| F10 默认YAML不能启动 | default-start.json：stock YAML仅loopback端口覆盖，真实readiness200 | 原复现PASS |
| F11 丢建房响应无法定位 | room-unknown.json、room-before-send-loss.json，真实发送前/后断开两分支 | 原复现PASS |
| R3-N02 历史请求续期/状态推进/SETTLING | original-lifecycle.log 23项及settling-without-restart.json、历史Bid/Play/Pass不续期/必要补偿 | 原复现PASS；完整终止恢复N07 FAIL |
| R3-N03 / R4-N04 请求clientSeq与持久化身份 | all-types-sequence-restart.json、restart-binding-negatives.json、hmac-historical.log，序号篡改/重排冷恢复拒绝 | 原复现PASS |
| R5-N05 authenticated journal downgrade | 35项独立HMAC/版本/根/keyless与真实MySQL测试；legacy-upgrade.json旧v0拒绝恢复 | 原复现PASS |
| R7-N03 回调乱序 / duplicate dealt | original-lifecycle.log，回调单调性和新动作续期/重复hand.dealt通过 | 原复现PASS |
| R7-N04 Set编译/回调清理 | Linux当前真实SDK原生出牌、message/close multiplicity及移除、重连；不重跑旧3.8 | 当前Linux回归PASS，历史Creator仅参照 |
| WX-N01 请求时间变化影响身份 | authentication.json普通同ID请求重试、ordinary-replay.json、重启绑定负例 | 原复现PASS |
| WX-N02 拒绝events[]变null | rejected-replay-contract.json真实MySQL冷重放，空数组保留 | 原复现PASS |
| R8-N01 / N02 / N03 | 原迟回调/原生退出/第一局结算恢复均新执行；扩展发现N01/N07、N06、N02/N04 | 三项扩展均FAIL |

兼容性：从历史04da095 schema构造隔离clone，旧schema启动明确拒绝、无自动DDL；仅测试库显式增加三列后，未知身份保留clientSeq0/version0/NULLtag，配置认证根的当前coordinator仍拒绝未签v0。`legacy-clone.json`、`legacy-upgrade.json`。没有猜测旧request身份、伪造HMAC或执行生产迁移；真实旧业务资产迁移仍需ChatGPT受控方案，不能因fail-closed就声称所有旧数据可无缝恢复。

## 8. 自动化、命令与故障执行记录

所有命令在固定源码执行；本节表格由本轮 `runs.jsonl`生成，时间为各命令耗时（父子脚本有重叠，不能累加为总墙钟时间）。全部日志按label保存为`$E/<label>.log`。Go JSON parent/subtest重复计数：unit与Race各563 PASS/1 SKIP，完整clean MySQL602 PASS/1 SKIP；original lifecycle23、HMAC35、original MySQL42；client31+37+39=107；HPKE9。默认跳过的互操作向量已用真实向量另行运行，不把SKIP算PASS。

可复用的核心命令（认证、私钥、DSN均从私有配置注入，不写在报告）：

```bash
# repo/server; Go1.25.8, GOMAXPROCS=2
 go test -json -count=1 -p 1 -parallel 1 ./...
 go test -json -race -count=1 -p 1 -parallel 1 ./...
 go vet -p 1 ./...
 APP_API_INTEGRATION=1 go test -json -count=1 -p 1 -parallel 1 -tags=integration ./...
 # 独立真实SQL竞争，不替换产品实现
 APP_API_INTEGRATION=1 go test -json -race -count=1 -p 1 -parallel 1 \
   -tags=integration -overlay "$E/overlay-expiry.json" -run '^TestCodexExpirySQL' \
   ./business/doudizhu/infrastructure/mysqlstore
# repo
 make fmt-check
 make api-contract-check
 bash scripts/verify-client-foundation.sh
 npm --prefix clients/packages/secure-envelope test
 node scripts/cocos-linux-wechat/verify-toolchain.cjs
 node scripts/cocos-linux-wechat/build.cjs "$E/native-output"
# 独立证据驱动，需先启动隔离依赖和相应私有fixture
 python3 "$E/games.py"
 python3 "$E/post-business-gates.py"
 NODE_EXTRA_CA_CERTS="$E/fixture-cert.pem" node "$E/wechat-full-ui-attempt2.cjs"
 DOCKER_BUILDKIT=0 COMPOSE_PARALLEL_LIMIT=1 bash "$E/runtime-isolated.sh"
```

上面是入口摘要，不隐藏实际CLI/overlay参数；精确命令、Node路径、SDK运行环境和故障注入顺序在runs、保留的脚本与各原始日志中。特别是正式native构建返回FAIL；补环境导出命令为真实CLI `--no-interactive build --project <retained-copy> --platform wechatgame --build-config <private-options>`。SDK实际运行要求报告记录的project title/engine/source/build-chain环境，不能无配置直接运行脚本后把环境失败算产品失败。

| 命令记录label（同名log） | UTC开始 | 秒 | 退出码 |
|---|---|---:|---:|
| `go-unit` | 07:05:55.189 | 85.067 | 0 |
| `go-race` | 07:09:02.032 | 156.540 | 0 |
| `client-foundation` | 07:18:36.572 | 52.405 | 0 |
| `fixture-setup` | 07:19:15.227 | 26.619 | 0 |
| `expiry-sql-first` | 07:20:47.690 | 13.239 | 1 |
| `expiry-sql-expanded` | 07:21:43.656 | 14.143 | 1 |
| `expiry-sql-lock-order` | 07:22:36.482 | 7.534 | 0 |
| `client-next-hand` | 07:23:54.732 | 0.484 | 1 |
| `client-next-hand-valid` | 07:24:20.002 | 0.328 | 1 |
| `mysql-integration-scale` | 07:23:25.304 | 123.458 | 1 |
| `go-vet` | 07:25:44.625 | 9.558 | 0 |
| `app-build` | 07:25:54.268 | 21.648 | 0 |
| `maintenance-build` | 07:26:16.037 | 3.708 | 0 |
| `fmt` | 07:26:19.832 | 0.565 | 0 |
| `api-contract` | 07:26:20.489 | 0.616 | 0 |
| `hpke` | 07:26:21.207 | 5.389 | 0 |
| `interop-ts-vector` | 07:26:26.822 | 4.386 | 0 |
| `interop-go-open` | 07:26:31.307 | 0.977 | 0 |
| `interop-go-manifest` | 07:26:32.356 | 0.343 | 0 |
| `interop-ts-manifest` | 07:26:32.814 | 4.163 | 0 |
| `crypto-interop` | 07:26:26.679 | 10.332 | 0 |
| `original-lifecycle` | 07:26:37.135 | 11.169 | 0 |
| `hmac-historical` | 07:26:48.394 | 8.183 | 0 |
| `mysql-integration-scale-clean` | 07:27:15.156 | 106.383 | 0 |
| `linux-toolchain-pins` | 07:29:23.415 | 0.571 | 0 |
| `linux-platform-tests` | 07:29:24.073 | 1.199 | 0 |
| `linux-native-wechat-build` | 07:29:25.355 | 82.372 | 1 |
| `native-private-deps-game-client-core` | 07:32:11.340 | 1.874 | 0 |
| `native-private-deps-secure-envelope` | 07:32:13.290 | 3.331 | 0 |
| `native-private-deps-doudizhu-client` | 07:32:16.703 | 1.613 | 0 |
| `linux-native-wechat-provisioned` | 07:32:18.395 | 261.012 | 0 |
| `expiry-sql-deadlock` | 07:37:50.737 | 15.563 | 0 |
| `expiry-sql-callback-all-phases` | 07:38:43.896 | 6.942 | 0 |
| `product-start` | 07:40:05.431 | 1.344 | 0 |
| `game1-commit` | 07:40:20.840 | 8.406 | 0 |
| `game1-reveal` | 07:40:29.325 | 64.590 | 0 |
| `original-N01` | 07:41:34.008 | 1.071 | 0 |
| `game1-complete` | 07:41:35.145 | 4.038 | 0 |
| `game2-commit` | 07:41:39.323 | 4.668 | 0 |
| `game2-reveal` | 07:41:44.086 | 64.707 | 0 |
| `active-lifecycle-matrix` | 07:42:48.869 | 16.562 | 0 |
| `game2-terminal-loss` | 07:43:05.525 | 3.248 | 1 |
| `terminal-recovery` | 07:43:08.858 | 1.857 | 0 |
| `completed-history` | 07:43:10.801 | 3.949 | 0 |
| `restart-binding-negatives` | 07:43:14.824 | 23.095 | 0 |
| `all-types-sequence-restart` | 07:43:38.000 | 20.650 | 0 |
| `game3-commit` | 07:43:58.728 | 4.838 | 0 |
| `game3-reveal` | 07:44:03.629 | 64.767 | 0 |
| `active-process-fault` | 07:45:08.482 | 8.009 | 0 |
| `settling-failed-complete-game` | 07:45:17.073 | 4.292 | 1 |
| `settling-without-restart` | 07:45:16.621 | 17.210 | 0 |
| `full-three-games` | 07:40:20.686 | 313.183 | 0 |
| `original-client-repro` | 07:46:23.770 | 0.645 | 0 |
| `client-lifetime-matrix` | 07:46:24.501 | 2.900 | 0 |
| `original-mysql-regressions` | 07:46:27.511 | 16.996 | 0 |
| `checkpoint-latency` | 07:46:44.589 | 3.934 | 0 |
| `receipt-scale-profile` | 07:46:48.605 | 97.612 | 0 |
| `expiry-sql-connection-loss` | 07:48:26.326 | 8.084 | 0 |
| `prune-cli-atomicity` | 07:48:34.505 | 9.111 | 0 |
| `ordinary-replay` | 07:48:43.705 | 9.356 | 0 |
| `rejected-replay-contract` | 07:48:53.143 | 1.149 | 0 |
| `room-after-send-loss` | 07:48:54.408 | 15.831 | 0 |
| `room-before-send-loss` | 07:49:10.335 | 0.760 | 0 |
| `deploy-wiring` | 07:49:11.175 | 0.476 | 0 |
| `compose-switch` | 07:49:11.753 | 0.488 | 0 |
| `default-yaml-start` | 07:49:12.315 | 1.336 | 0 |
| `generation-check` | 07:49:13.726 | 2.339 | 0 |
| `socket-lifetime` | 07:49:16.134 | 33.766 | 0 |
| `leave-fairness-real` | 07:51:17.947 | 1.248 | 1 |
| `wechat-launch` | 07:52:38.619 | 0.208 | 0 |
| `linux4-native-full-ui` | 07:53:30.497 | 66.165 | 1 |
| `linux4-native-full-ui-attempt2` | 07:57:28.201 | 355.053 | 1 |
| `wx-sdk-authentication` | 08:03:46.900 | 16.536 | 0 |
| `legacy-clone` | 08:06:24.166 | 9.576 | 0 |
| `legacy-upgrade` | 08:08:26.710 | 1.223 | 0 |
| `expiry-sql-real-archive-cold` | 08:09:27.397 | 11.128 | 1 |
| `expiry-sql-sigkill-boundaries` | 08:11:50.599 | 6.178 | 1 |
| `expiry-sql-final-real-archive` | 08:15:00.430 | 10.248 | 1 |
| `expiry-actual-sweep-once` | 08:15:37.245 | 7.694 | 0 |
| `linux-native-web-build` | 08:16:38.188 | 176.616 | 0 |
| `web-runtime-raw` | 08:21:56.519 | 94.273 | 1 |
| `web-independent-strict` | 08:24:16.282 | 7.740 | 2 |
| `web-existing-bigint-postbuild` | 08:24:53.326 | 0.478 | 0 |
| `web-runtime-operational` | 08:25:05.396 | 8.676 | 0 |
| `web-independent-strict-source-settings` | 08:25:47.016 | 3.651 | 0 |
| `terminated-history-retention` | 08:26:59.513 | 45.388 | 0 |
| `production-compose-runtime` | 08:28:28.653 | 1.297 | 1 |
| `production-compose-runtime-configured` | 08:33:00.185 | 340.502 | 0 |

失败归因保持透明：

1. 首次完整MySQL suite被先前外置expiry fixture遗留的6个本轮自有r9-exp行污染，两项正式测试失败；修正**私有fixture清理**并仅删除本轮孤立行后完整602项通过，原失败保留。未改正式测试或降低断言。
2. 初始next-hand探针FinalEvidence DTO嵌套错误不是产品反例；修正外置fixture后原完整合法snapshot仍失败，并由实际GUI确认N02。
3. actualSweep探针初始查询不存在column是私有查询错误，保留日志；按当前schema精确同aggregate查询后实际Sweep一次性与归档正对照通过。
4. Web独立strict checker最初遗漏项目Bundler/allowTSextensions选项；保留原checker/配置，合并现有项目选项与真实引擎声明后检查通过，不是关闭严格模式。
5. Production runtime首次因私有runtime-ports overlay文件未复制在build前退出；补齐原有未改overlay后，原Dockerfile和完整断言运行340.502秒通过。未修改production Compose。
6. 三局中的game2-terminal-loss和settling-failed-complete-game预期以退出1表示受控丢结果/故障；父级exact retry/compensation断言和full-three-games最终退出0。不能将这两次预期故障算产品独立缺陷，也不能隐藏它们。
7. 正式native依赖错误、原始Web运行BigInt错误、GUI四项状态错误及真实SQL不变量失败是**实际失败**，没有由补充通过覆盖。测量脚本退出0（如历史map增长）也不等于产品资源要求PASS。

## 9. 性能、锁等待与资源

下表均为本轮隔离MySQL、本机配额和指定样本，单位ms；数字来自实际日志，不推导生产SLA。

| 样本/路径 | P50 | P95 | P99 | max |
|---|---:|---:|---:|---:|
| 80次真实COMMIT-before-ACK | 14.095 | 44.397 | 48.563 | 61.327 |
| 100 hands / 16workers commit | 18.827 | 56.644 | 58.584 | 65.644 |
| 1000 hands / 16workers commit | 18.460 | 54.859 | 90.972 | 100.785 |
| 真实加密checkpoint100次（payload77035B） | 6.434 | 25.889 | 48.085 | 75.324 |
| 1000新Bid | 8.499 | 27.699 | 51.311 | 348.735 |
| 1000新Play | 8.534 | 27.547 | 46.973 | 80.010 |
| 1000新Pass | 8.502 | 24.551 | 45.504 | 435.588 |
| 1000手局热Bid receipt | 0.036 | 0.061 | 0.108 | 19.477 |
| 1000手局热Play receipt | 0.045 | 0.105 | 0.207 | 2.578 |
| 1000手局热Pass receipt | 0.045 | 0.065 | 0.109 | 2.832 |
| 16workers/1600次同ID receipt争用（1000 hand fixture） | 0.046 | 1.537 | 2.245 | 27.870 |

正式100/1000 hand恢复**总耗时**2.488/18.685秒、post-GC heap增量306040/3029856B；独立含receipt恢复profile为2.253/20.289秒、694872/6930072B（不同fixture，不混为同次结果）。checkpoint样本实际version33/watermark60。真实writer锁等待与InnoDB死锁已观察并验证回滚/重试，不等于不存在所有锁顺序风险。没有把总恢复时间冒充单局P95，也没有因平均值良好声称满足所有早期建议预算。

`socket-lifetime.json`：60次warmup、1500个真实WSS连接、15秒cooldown，active0、goroutine26→27、heap约4.603→4.772MB、RSS29.463→31.994MB、自然GC6→27。该运行未强制GC粉饰生产连接回收。1000局候选复制/receipt profile已测；未做生产硬件长期soak/容量上界/fsync预算认证。

新资源反例：同一runtime、共享一个真实DB pool、100个真实EXPIRED并归档的hand，各一个accepted Bid，无active game。20/40/60/80/100局时 finalized、receiptHands、epoch map分别仍保留20/40/60/80/100；主动GC仅为检查对象可达性，heap delta204336/327120/438560/541824/652000B，goroutineDelta0。结合源码无cap/TTL/完成清理证明**有界历史资源要求未实现**（N08），不声称实测无限时长泄漏或已发生OOM，也不冒充100局正常胜负局。

SDK memcg执行时memory.current1,891,921,920B，memory.events low/high/max/oom/oom_kill均0；实际UI异常不是本次OOM的推测替代。清理后可用内存约4481MiB，swap0；根盘48%（约21GiB余量），data盘68%（约13GiB余量）。未执行全局Docker清理。

## 10. 新确认问题及交回修复建议

以下8项均未由Codex修复。代码位置相对固定候选；源码行仅辅助定位，私有脚本/日志保留实际复现。P1为内部关键契约/恢复或重复游戏阻断，P2为客户端边界/构建/资源问题。没有从受控DB/内部调用前提夸大为互联网越权。

### R9-N01 / P1 — 业务拒绝的Expiry仍提交fence

- 位置：`server/business/doudizhu/application/hand_commands.go:277`（Fence先于域mutation）、`:291`；`application/service.go:146`（business rejection完成回执后提交）。
- 前提：durable实时hand在BIDDING，有效ExpectedLiveVersion=1，内部Expire命令的ExpectedVersion过时；ClientSeq与ExpectedVersion一致且未被sequence guard先拒绝。不是外部WSS Expire调用；正常Supervisor自然出现该组合尚未建立。
- 复现：用真实Store/Coordinator/Service建立accepted Bid v1的hand；读取epoch2/watermark1/domainBIDDING；调用上述形状Expire；读取SQL和结果，再发当前writer合法新Bid。
- 实际：accepted=false/DDZ_VERSION_CONFLICT，却epoch2→3、watermark1、domain仍BIDDING/v12；新Bid `verification failed / live writer epoch fenced`。
- 预期：拒绝终止不得提交writer fence或破坏仍活跃runtime；所有拒绝/rollback须维持一致性。
- 证据：`expiry_sql_test.go` 的 `TestCodexExpirySQLRejectedMustNotFence`、`expiry-sql-final-real-archive.log`；rollback、新commit-wins、成功expiry正对照通过。
- 影响/建议：内部失败终止可阻断后续合法动作，需要恢复接管；先完成域验证再fence，或把副作用与成功终止严格原子化，保留业务拒绝回执契约。应补正常Supervisor可达性分析/回归，不能仅删除此负例。

### R9-N02 / P1 — 同房第二局保留上一局COMPLETED身份

- 位置：`clients/fair-doudizhu-cocos/assets/scripts/bootstrap/GameBootstrap.ts:287`、`:296`；`game/GameClientShell.ts:177`；`clients/packages/doudizhu-client/src/index.ts` HandStore身份保护。
- 前提/步骤：原生完整第一局→BackRooms→三人ready→StartHand真实创建不同hand→OpenTable；独立Shell探针再送合法新hand BIDDING snapshot作对照。
- 实际：Bootstrap.handID为新hand，HandStore仍第一局COMPLETED/v64，合法新hand snapshot被跨hand保护拒绝，UI仍旧分数/无新局操作。初始第一局COMPLETED正例通过。
- 预期：经认证room activeHand确认后显式、安全地切换hand上下文，清旧选牌/终态/未完成请求，之后接受新hand权威snapshot。
- 证据：`wechat-full-ui.json.secondHand.identity`、`client-next-hand-valid.log`（初始错误fixture另存）。
- 影响/建议：同房连续游戏关键链路阻断；提供授权的hand切换生命周期，保留跨房间/旧消息隔离，不能放宽全局HandStore ID检查来绕过。

### R9-N03 / P2 — 正式Linux exporter复制后缺portable包依赖

- 位置：`scripts/cocos-linux-wechat/build.cjs:27`、`:29`、`:35`。
- 前提/步骤：已具备固定CLI/引擎/Node、源portable包已build；直接运行正式build.cjs。它拷贝clients时排除全部node_modules，只对fair-doudizhu-cocos执行npm ci。
- 实际：file-linked secure-envelope等包无法解析hpke/@noble，真实CLI exit34、wrapper exit1。私有副本对三个portable目录npm安装依赖后相同CLI/WECHAT/严格检查真实成功。
- 预期：正式脚本在声明的环境中完整准备复制后依赖，可重复执行真实导出，无人工补目录。
- 证据：`linux-native-wechat-build.log`、`native-build-provisioned.py`、`linux-native-wechat-provisioned.log`、输入绑定及产物hash。
- 影响/建议：单一Linux流水线不能按正式入口交付；在脚本内按锁文件准备各file依赖或正确workspace打包，验证冷副本而非依赖原机node_modules。补环境PASS不清除此缺陷。

### R9-N04 / P2 — 未发牌取消后的lastHandId不可恢复

- 位置：`server/business/doudizhu/domain/room.go:226` FinishHand；`clients/fair-doudizhu-cocos/assets/scripts/game/GameOnlineController.ts:66`、`GameBootstrap.ts:310`。
- 前提/步骤：三真实账号开局但未发牌→公开授权hand.cancel成功→房间activeHand为空/lastHandId为该hand→客户端reconnectAndResync；实际原生下一局取消后Reconnect再次确认。
- 实际：domain ABORTED，未发牌runtime没有可提供的final archive，HTTP404；原生显示“重连失败：HTTP 404”。
- 预期：合法取消后的房间可重入；lastHandId语义与可查询证据一致，明确区分无归档终止和可恢复终态。
- 证据：`leave-fairness-real.json`、`wechat-full-ui.json.preDealCancelledReentry`。
- 影响/建议：合法异常/取消流程阻断房间重连；由owner明确终止摘要/归档或HTTP契约，客户端按真实终止种类处理，不无条件把所有lastHand当有archive的完成局。

### R9-N05 / P2 — 最后owner关闭后Room Restore拒绝合法空座

- 位置：`server/business/doudizhu/domain/restore.go:59`、`domain/room.go:154`。
- 前提/步骤：真实owner创建仅自己在场房间→原生Leave最后一人成功→查本轮DB为CLOSED且空座→GET同room。
- 实际：RestoreRoom无条件要求owner占seat1，而关闭已清空；GET返回400（owner must occupy seat1）。原生attempt1在该检查退出，attempt2记录失败后继续其他验收。
- 预期：合法持久化CLOSED快照可还原，API按明确closed/成员契约响应，不把成功状态当无效snapshot。
- 证据：`wechat-full-ui-attempt1.json`、`linux4-native-full-ui.log`、attempt2检查与真实DB确认。
- 影响/建议：关闭后的权威查询/未知结果协调受损；这是本轮新发现的既有不变量冲突，未证明候选首次引入；按RoomClosed定义验证owner占座规则，并覆盖真实关闭/restore。

### R9-N06 / P2 — Leave后迟到的真实Evidence重新写入旧牌局

- 位置：`clients/fair-doudizhu-cocos/assets/scripts/bootstrap/GameBootstrap.ts:369`/`:374`，leave清理`:268`；`game/GameClientShell.ts:140`。
- 前提/步骤：原生RoomEvidence发出真实已认证请求，取得并通过HTTP客户端校验的旧终局结果，仅在await返回边界延迟交付；完成后续操作/最后owner原生Leave，确认roomID/handID和store已清空；释放原始未篡改response。
- 实际：旧HandStore/COMPLETED分数复活，Bootstrap.roomID/handID仍空；notice显示上一房间分数。
- 预期：离房/换局后的旧异步HTTP响应无权改变当前上下文。
- 证据：`wechat-full-ui-attempt2.cjs`的held actual HTTP response探针、`wechat-full-ui.json.delayedEvidenceAfterLeave`。
- 影响/建议：跨房间状态污染/错误UI；给room/hand上下文及每次await补generation/身份复核，与原有socket generation fence一致。该用户原本仍可读取自己历史归档，**不声称新数据权限泄露**。

### R9-N07 / P1 — EXPIRED冷恢复跳过回执，提交后归档缺口不补偿

- 位置：`server/business/doudizhu/infrastructure/runtime/durable_live_recover.go:39`、`:169`；`application/hand_commands.go:207`/`:217`。
- 前提：真实已发牌durable hand、accepted Bid，实际MySQLarchive适配器；不是noop archive。
- 复现A：正常Expire并真实archive=1，热重放原accepted Bid完全相同；构造冷Coordinator/RecoverActive后再发同一原Bid。
- 实际A：durable action仍在，热回执v2正确，但冷返回WrongPhase/version0，因为EXPIRED不在replay集合；**没有声称action行丢失**。
- 复现B：独立子进程实际终止COMMIT成功后、runtime Archive前SIGKILL；冷恢复后精确原expiry requestId重试。
- 实际B：EXPIRED、epoch4、水位1、成功终止receipt1、archive0；重试accepted=true/duplicate=true，archive仍0。COMMIT前强杀正对照完整回滚；正常Sweep一次归档正对照通过。
- 预期：所有需要保留的已接受历史回执在终止后按身份精确重放；必要未完成归档在恢复/精确重试完成一次，不能凭duplicate成功遗漏finalization。
- 证据：`expiry_sql_test.go`、`expiry-sql-real-archive-cold.log`、`expiry-sql-sigkill-boundaries.log`，实际SQL/child PID kill记录。
- 影响/建议：终止后的B级恢复/历史幂等与证据完整性失败；为非COMPLETED终态恢复必要历史/补偿，区分禁止复活执行器与允许重放授权回执，明确COMMIT和归档间持久恢复进度。

### R9-N08 / P2 — 归档后历史runtime资源无界保留

- 位置：`server/business/doudizhu/infrastructure/runtime/live_hand_coordinator.go:315`、`live_receipts.go:63`/`:71`、durable seed cache。RollbackStart对receipt的delete只属开局失败补偿，不是正常完成清理。
- 复现：共享一个真实SQL池/同一runtime顺序生成100个合法durable手局，每局accepted Bid→真实Expire并实际archive；每20局主动GC后记录map/heap，确认activeGames始终0。
- 实际：finalized/receiptHands/epochs各20→100，100局后heap较起点+652000B，源码无TTL/cap/正常完成回收路径；goroutineDelta0。
- 预期：原始有界资源契约需要可解释的保留/淘汰/按需持久回执恢复策略，而非永久内存保留。
- 证据：`history_resource_test.go`、`terminated-history-retention.log`、`overlay-resource.json`。
- 影响/建议：长生命周期进程随历史局数增长；定义有限热缓存+持久回执查询/保留策略，并覆盖完成/取消/过期与cold retry。该100局观察不能证明无限时长泄漏、全部正常胜负局行为或实际OOM；也不因短时连接回收良好判资源有界PASS。

## 11. 原始114项验收矩阵

要求文本从原始Goal0046矩阵逐项保留，上一轮列取Round8实际status，**不是最早历史verdict**；本轮每行使用本轮源码/行为/日志。PASS按行写明的范围，不扩张为平台或发布资格。BLOCKED行说明缺失条件；FAIL按产品反例判定，不因为环境/CI通过而掩盖。审计项PASS表示审计交付已完成。

| ID | 原始要求 | 本轮 | Round8→Round9 | 本轮方法、证据与边界 |
|---|---|---|---|---|
| A01 | 真实身份认证和账号隔离 | PASS | PASS→PASS | 实际 wx.login 官方换证、两次稳定 AccountID、session 校验/吊销与四个 Go 测试账号隔离；authentication.json、full-three-games.log。 |
| A02 | 创建、加入、退出房间 | FAIL | FAIL→FAIL | 原生创建/加入/Leave 与丢 ACK 精确重试通过；关闭房间 GET 400、退出后迟到证据恢复旧状态失败；wechat-full-ui.json，N05/N06。 |
| A03 | 三人准备和开局 | PASS | PASS→PASS | 三账号真实三席准备、原生 StartHand 与三轮 WSS 开局；wechat-full-ui.json、full-three-games.log。 |
| A04 | 公平承诺与揭示 | PASS | PASS→PASS | 原生准备、外部私有备份、commit/import/HPKE reveal；三轮新 Node 进程导入恢复；wechat-full-ui.json、game1-reveal.log。 |
| A05 | 公共随机源验证及发牌 | PASS | PASS→PASS | 真实 Quicknet/drand BLS、签名 manifest/AAD、51 张互斥牌与三席私牌；game1-reveal.log、wechat-full-ui.json。 |
| A06 | 地主竞叫及确定 | PASS | PASS→PASS | 低叫分、多次叫分、原生叫 3 分进入 PLAYING；active-lifecycle-matrix.json、wechat-full-ui.json。 |
| A07 | 出牌、过牌、牌型比较 | PASS | PASS→PASS | Go 牌型全回归、三轮完整 WSS 对局及原生 20 次出牌/1 次过牌，无 owner 后台代打；go-unit.log、full-three-games.log、wechat-full-ui.json。 |
| A08 | 回合切换 | FAIL | FAIL→FAIL | 原提交/迟回调三个阶段通过；被拒绝 Expire 仍 fence 导致后续合法 Bid 失败；expiry-sql-final-real-archive.log，N01（内部调用前提）。 |
| A09 | 终局、计分及证据归档 | FAIL | FAIL→FAIL | 正常 COMPLETED 分数/UI/归档通过；EXPIRED 在提交后归档前 SIGKILL 缺归档且重试不补偿；expiry-sql-sigkill-boundaries.log，N07。 |
| A10 | 断线重连及权威状态同步 | FAIL | FAIL→FAIL | 原生实际 SIGKILL 后 v19→19 与首次终局重连通过；下一局旧终态和取消后重连 404；wechat-full-ui.json，N02/N04。 |
| B01 | 已向客户端确认的操作必须先完成持久化提交。 | PASS | PASS→PASS | 实际 pre-COMMIT kill/连接 KILL 回滚、post-COMMIT 丢响应及精确重试，正常动作持久化先于 ACK；active-process-fault.json、expiry-sql-connection-loss.log。 |
| B02 | 任何已确认操作不能在进程崩溃后丢失。 | PASS | PASS→PASS | 正常 Bid/Play/Pass 已 ACK 的动作行与版本在真实进程重启后保留；all-types-sequence-restart.json、active-process-fault.json；EXPIRED 历史回执可用性另在 B09 判 FAIL，不声称动作行被删除。 |
| B03 | 同一 requestId 的重复请求不能重复执行。 | PASS | PASS→PASS | 真实相同 ID 并发 16 路不重复业务动作、终局重试不重复结算，实际 Sweep/系统 ID 重试只执行一次；active-lifecycle-matrix.json、expiry-actual-sweep-once.log。 |
| B04 | 相同 requestId、不同参数或不同操作者必须拒绝。 | PASS | PASS→PASS | 重启后篡改 actor/type/aggregate/payload/clientSeq 均拒绝，旧回执正对照通过；restart-binding-negatives.json、all-types-sequence-restart.json。 |
| B05 | 新服务接管后，旧 Owner Epoch 必须被正确隔离。 | PASS | PASS→PASS | 实际新 owner takeover、旧 writer 拒绝；MySQL writer-lock-first 正常终止对照通过；mysql-integration-scale-clean.log、expiry-sql-lock-order.log。 |
| B06 | 检查点、日志、水位线和业务版本必须严格一致。 | FAIL | PASS→FAIL | checkpoint/日志正常恢复及篡改拒绝通过；拒绝 Expire 后 epoch 2→3 而 domain/watermark 未推进，runtime 被 fence；expiry-sql-final-real-archive.log，N01。 |
| B07 | 检查点或日志被篡改时必须拒绝恢复。 | PASS | PASS→PASS | 真实 MySQL checkpoint、genesis、journal HMAC/序号/物理顺序篡改拒绝；original-mysql-regressions.log、hmac-historical.log。 |
| B08 | 恢复不能泄露其他玩家的私有手牌。 | PASS | PASS→PASS | 三账号/非成员实际私牌及归档授权拒绝，genesis 换座篡改拒绝；game1-reveal.log、wechat-full-ui.json、original-mysql-regressions.log。 |
| B09 | 终局重启必须保留最终证据和历史回执。 | FAIL | PASS→FAIL | 实际 EXPIRED 已归档对局热回执正常、冷恢复原 ACK Bid 失败；expiry-sql-real-archive-cold.log，N07。 |
| B10 | MySQL COMMIT 前后发生 SIGKILL、网络断开、响应丢失时，不得出现重复执行或状态分叉。 | FAIL | PASS→FAIL | 正常动作 pre/post-COMMIT 故障通过；终止 COMMIT 后归档前进程强杀形成未补偿缺口；expiry-sql-sigkill-boundaries.log，N07。 |
| B11 | 真实 MySQL 事务 | PASS | PASS→PASS | 真实 MySQL 5.7.44、锁等待/死锁/COMMIT 失败/回滚均实际执行；mysql-integration-scale-clean.log、expiry-sql-deadlock.log、expiry-sql-connection-loss.log；本项指真实事务验证，语义失败另列 B06/B10/B13。 |
| B12 | 并发重复提交 | PASS | PASS→PASS | 三真实 WSS 会话、16 路同 ID 并发原回执一致；不同 ID 同序号冲突；active-lifecycle-matrix.json、receipt-scale-profile.log。 |
| B13 | Epoch fencing | FAIL | PASS→FAIL | 成功终止隔离旧 writer 通过；业务拒绝的终止不应推进 epoch，但实测推进；expiry-sql-final-real-archive.log，N01。 |
| B14 | 检查点加密 | PASS | PASS→PASS | 真实加密 checkpoint 写入/旧密钥/密文篡改和 epoch 负例；checkpoint-latency.log、original-mysql-regressions.log。 |
| B15 | 日志完整性 | PASS | PASS→PASS | 独立 HMAC 降级、客户端序号、重排/截断/错根/显式 keyless 测试 35 项通过；hmac-historical.log。 |
| B16 | 三人完整对局恢复 | PASS | PASS→PASS | 三局正常三身份完整对局，活跃阶段 whole-process kill、终局丢响应恢复和 SETTLING 补偿均执行；full-three-games.log、settling-without-restart.json；异常终止另列 B09/B18。 |
| B17 | 进程级故障注入 | PASS | PASS→PASS | 实际 SIGKILL 及实际 MySQL 连接 KILL，不以返回模拟错误代替所有故障；active-process-fault.json、expiry-sql-sigkill-boundaries.log、expiry-sql-connection-loss.log；执行通过不表示每个恢复不变量通过。 |
| B18 | 终局证据恢复 | FAIL | PASS→FAIL | 正常 COMPLETED 证据恢复通过；EXPIRED 提交后归档前 kill，冷恢复与同 ID retry 仍 archive=0；expiry-sql-sigkill-boundaries.log，N07。 |
| C01 | 新记录使用当前主密钥加密 | PASS | PASS→PASS | 真实 MySQL 新 seed/checkpoint 当前根加密与正常读取；original-mysql-regressions.log 的 Rotation/Keyring 测试。 |
| C02 | 旧记录可以使用保留的历史密钥读取 | PASS | PASS→PASS | 真实 retained AES key 读取旧 seed/checkpoint，原始贡献一致；original-mysql-regressions.log。 |
| C03 | 历史密钥数量和长度严格受限 | PASS | PASS→PASS | keyring 数量/长度/重复/非法 key 边界；original-mysql-regressions.log、go-unit.log。 |
| C04 | 旧密钥不存在时拒绝错误解密 | PASS | PASS→PASS | 删除所需旧 AES key 后拒绝恢复；original-mysql-regressions.log，非静默重新生成。 |
| C05 | 重加密必须先认证旧密文 | PASS | PASS→PASS | 重包前认证旧密文，错误旧 key/篡改 ciphertext 拒绝；original-mysql-regressions.log。 |
| C06 | 种子重加密不得改变原始随机贡献 | PASS | PASS→PASS | seed 重包前后实际随机贡献一致；original-mysql-regressions.log、hmac-historical.log。 |
| C07 | 检查点重加密必须校验当前 Owner Epoch | PASS | PASS→PASS | checkpoint 重包错 epoch 被拒绝；original-mysql-regressions.log。 |
| C08 | 错误密钥、篡改密文不能恢复 | PASS | PASS→PASS | 实际错误密钥/篡改密文/错 HMAC 根均 fail closed；original-mysql-regressions.log、hmac-historical.log。 |
| C09 | 重加密后仅保留新密钥仍可正确恢复 | BLOCKED | BLOCKED→BLOCKED | BLOCKED：AES seed/checkpoint 重包已测；完整 journal HMAC 换根后 new-only 冷恢复缺受支持操作流程与合格证据，不能用 AES 重包替代。 |
| C10 | 旧密钥未完成迁移前不得被错误淘汰 | BLOCKED | BLOCKED→BLOCKED | BLOCKED：旧 journal 仍依赖旧 HMAC 根；完整密钥库存、全量迁移与核准退休流程未验证；hmac-historical.log 只证明错误根会拒绝。 |
| D01 | 活跃牌局不能删除 | PASS | PASS→PASS | 真实 active-hand prune 拒绝，行数保留；original-mysql-regressions.log。 |
| D02 | 未归档牌局不能删除 | PASS | PASS→PASS | 真实未归档手局 prune 拒绝；original-mysql-regressions.log、prune-process.json。 |
| D03 | 不满足 30 天保留期不能删除 | PASS | PASS→PASS | 真实 29 天拒绝/超过 30 天合格正对照；prune-process.json、original-mysql-regressions.log。 |
| D04 | 损坏或不完整的归档不能删除 | PASS | PASS→PASS | 真实已完成对局克隆，破坏 DDZ 证据并重算通用 digest，仍拒绝删除；original-mysql-regressions.log。 |
| D05 | 最终版本和日志水位线不一致不能删除 | PASS | PASS→PASS | 最终版本与 watermark 不符拒绝清理；original-mysql-regressions.log。 |
| D06 | 历史日志数量不完整不能删除 | PASS | PASS→PASS | 删除/缺少历史 action 导致计数不符，prune 拒绝；original-mysql-regressions.log。 |
| D07 | 不得直接越权读取其他业务模块数据库表 | PASS | PASS→PASS | 正式模块/数据库依赖边界及源码导入审计通过；mysql-integration-scale-clean.log、go-unit.log。 |
| D08 | 保留完整终局证据和必要审计信息 | FAIL | PASS→FAIL | 正常 COMPLETED 清理保留证据正对照通过，但 EXPIRED 强杀窗口未留下完整归档；expiry-sql-sigkill-boundaries.log，N07；按完整证据要求判 FAIL。 |
| D09 | 默认 dry-run 不得修改数据库 | PASS | PASS→PASS | 实际 maintenance 默认 dry-run，数据库前后无删除；prune-process.json。 |
| D10 | 必须显式 `--execute` 才能执行清理 | PASS | PASS→PASS | 实际 --execute 才删除符合条件历史，档案保留；prune-process.json。 |
| D11 | 中途崩溃后不能出现违反一致性约束的部分删除 | PASS | PASS→PASS | 实际删除事务中 SIGKILL，连接断开后原记录完整回滚；prune-process.json。 |
| E01 | 提交承诺前生成高熵随机贡献 | PASS | PASS→PASS | 真实 SDK 安全随机数两次 32 字节不同，原生 prepare 及缺熵 fail-closed 测试；authentication.json、wechat-full-ui.json、client-foundation.log。 |
| E02 | 恢复凭证包含完整且可验证的信息 | PASS | PASS→PASS | 真实承诺前导出完整 kit，验证身份/hand/seat/commitment 绑定；game1-commit.log、client-lifetime-matrix.log。 |
| E03 | 用户备份后才允许提交承诺 | BLOCKED | BLOCKED→BLOCKED | BLOCKED：工具确实先私有外部备份再 commit；真实人类备份确认、设备重启后找回与 UX 未验证；不能以自动化写私有文件替代用户行为。 |
| E04 | 应用退出后可以导入恢复凭证 | PASS | PASS→PASS | 三轮 commit 后新 Node 进程导入同一 kit，并由原生 Import 实际导入；game1-reveal.log、wechat-full-ui.json；非手机重装恢复资格。 |
| E05 | 恢复凭证与当前登录身份、Hand、Seat、Commitment 必须匹配 | PASS | PASS→PASS | 错误 account/hand/seat/commitment 绑定拒绝，实际有效 kit 正对照通过；client-lifetime-matrix.log、wechat-full-ui.json。 |
| E06 | 错误凭证必须拒绝 | PASS | PASS→PASS | 损坏/错身份/错承诺 kit 拒绝；client-foundation.log、client-lifetime-matrix.log。 |
| E07 | 不能产生新的随机贡献冒充旧承诺 | PASS | PASS→PASS | 未知 commit/reveal 精确原 DTO 重试、不重新生成秘密；leave-fairness-real.json、client-lifetime-matrix.log。 |
| E08 | 不能通过普通小游戏明文持久化自动泄漏随机数 | PASS | PASS→PASS | SDK 只测试自有存储键，实际 kit 保存在私有外部证据目录；源码与客户端测试无普通 storage 随机贡献自动保存；authentication.json、client-foundation.log。 |
| E09 | 释放对象时敏感数据应尽可能清零 | PASS | PASS→PASS | 原 F07 异步 prepare/dispose 与敏感 byte-array 尽力清零回归；original-client-repro.log、client-lifetime-matrix.log；不声称 JS immutable string 物理清除。 |
| E10 | 请求丢响应时必须保留原 requestId 精确重试 | PASS | PASS→PASS | 实际丢失 fairness/leave/终局结果后原 requestId/DTO 精确重试；leave-fairness-real.json、wechat-full-ui.json、terminal-recovery.json。 |
| E11 | 不能因为未知结果直接生成新的操作请求 | PASS | PASS→PASS | fairness 未知期间 Leave 发包数=0，未知 leave 保留原命令，第二原生点击复用原 DTO；leave-fairness-real.json、wechat-full-ui.json。 |
| F01 | 验证使用三个真实、不同的服务端认证身份 | PASS | PASS→PASS | 四个正常服务端认证 Go 账号，每局三人身份不同；原生局 wx 账号加两个正常认证 Go peers；full-three-games.log、wechat-full-ui.json。 |
| F02 | 验证三人座位各不相同 | PASS | PASS→PASS | 每局三个不同 AccountID/不同 seat，51 私牌互斥；full-three-games.log、wechat-full-ui.json。 |
| F03 | 验证每次操作使用真实 WSS | PASS | PASS→PASS | 游戏业务命令经实际认证 WSS，HTTP 用于文档约定的公平/证据接口；full-three-games.log、active-lifecycle-matrix.json。 |
| F04 | 验证客户端私有手牌访问权限 | PASS | PASS→PASS | 成员只能读本人私牌，第四账号 403，终局私牌 404 不捏造牌；game1-reveal.log、wechat-full-ui.json。 |
| F05 | 验证游戏完整进行到终局 | PASS | PASS→PASS | 三轮 Go 客户端完整游戏及实际原生 owner 完整一局，无代打；full-three-games.log、wechat-full-ui.json。 |
| F06 | 验证最终结算及证据 | PASS | PASS→PASS | 四局正常 archive/分数/digest 验证，原生 v64 胜者1、6/-3/-3；completed-history.json、wechat-full-ui.json。 |
| G01 | 并发牌局状态隔离 | PASS | PASS→PASS | 100/1000 hands、16 worker 真实数据库隔离、三账号并发；mysql-integration-scale-clean.log。 |
| G02 | 内存增长与回收 | FAIL | BLOCKED→FAIL | 同一 runtime 100 个真实归档终止局，active=0，finalized/receiptHands/epochs 各保留100且无 cap/TTL；terminated-history-retention.log，N08；非长期 OOM 证明。 |
| G03 | MySQL 提交延迟 | PASS | PASS→PASS | 真实 fsync=1 MySQL，80 次提交及 100/1000 并发分位数；mysql-integration-scale-clean.log，详见第9节。 |
| G04 | P50/P95/P99 | PASS | PASS→PASS | P50/P95/P99/max 实际样本，含 checkpoint、新动作和重复回执；checkpoint-latency.log、receipt-scale-profile.log。 |
| G05 | 检查点及故障恢复耗时 | PASS | PASS→PASS | 实际 encrypted checkpoint 100 写入与 100/1000 hand 冷恢复计时；checkpoint-latency.log、mysql-integration-scale-clean.log；总耗时不冒充单局 P95。 |
| G06 | 高并发下连接池和锁竞争 | PASS | PASS→PASS | 16 worker/24 DB 连接上限、实际 INNODB_LOCK_WAITS 及 deadlock victim/retry；mysql-integration-scale-clean.log、expiry-sql-lock-order.log、expiry-sql-deadlock.log。 |
| G07 | Race 检查 | PASS | PASS→PASS | 全仓 Go -race、独立生命周期及 SQL 竞争 -race 实际执行；go-race.log、original-lifecycle.log、expiry-sql-final-real-archive.log；语义失败单列。 |
| G08 | Goroutine 泄漏 | PASS | PASS→PASS | 60 WSS warmup 后 1500 次连接关闭、15秒 cooldown，active=0、goroutine 26→27；socket-lifetime.json；只能说明本次观察，无长期零泄漏声明。 |
| G09 | 重复请求及失败重试压力 | FAIL | PASS→FAIL | 16 路重复压力正对照通过；EXPIRED 冷恢复后原 ACK 请求不能重放、精确 expiry retry 不修归档；expiry-sql-real-archive-cold.log、expiry-sql-sigkill-boundaries.log，N07。 |
| G10 | 长牌局的候选状态复制开销 | PASS | PASS→PASS | 实际候选快照复制/长历史开销 profile 与 1000 局 receipt 热路径；original-mysql-regressions.log、receipt-scale-profile.log，未据此推断生产容量。 |
| H01 | TypeScript 类型检查 | PASS | PASS→PASS | 当前 TS 107 测试/类型检查、真实微信导出严格检查；Web 另用真实生成 cc4 declarations 严格检查通过，保留初始 checker 配置错误；client-foundation.log、web-independent-strict-source-settings.log。 |
| H02 | Web 实际构建 | PASS | PASS→PASS | 真实官方 CLI4 WEB_DESKTOP 导出，完整可运行配方还须仓库已有 BigInt 后处理；linux-native-web-build.log、web-existing-bigint-postbuild.log；原始导出启动 FAIL 保留。 |
| H03 | Chromium 场景冒烟测试 | PASS | PASS→PASS | 上述完整配方产物 Chromium 1280×720/844×390/390×844、字体/帧缓冲/销毁检查通过；web-runtime-operational.log；原始包 Math.pow(BigInt) FAIL 未隐藏。 |
| H04 | 微信小游戏构建 | PASS | PASS→PASS | 隔离副本补三 portable package npm 依赖后真实 WECHAT 导出、入口/adapter/资源/严格检查与 SDK 执行；linux-native-wechat-provisioned.log；正式 wrapper 失败另在 AUTO10/N03，不等同一键构建通过。 |
| H05 | 抖音小游戏构建兼容性 | BLOCKED | PASS→BLOCKED | BLOCKED：当前固定 Linux CLI/extension 无可用 Douyin exporter，未执行当前候选抖音构建；旧 Creator 3.8 历史产物不能替代本轮 Linux 基线。 |
| H06 | 游戏 UI 操作与真实网络请求一致 | FAIL | FAIL→FAIL | 原生登录/创建/退出/公平/出牌和首局终态通过；同房第二局、取消重入、迟到证据状态错误；wechat-full-ui.json，N02/N04/N06。 |
| H07 | 重连及恢复凭证操作可触达 | PASS | PASS→PASS | 原生 Import/Retry/Reconnect/Refresh/Evidence 实际按钮可触达，未知 leave 原 DTO 重试通过；wechat-full-ui.json；功能恢复缺陷按 A10/H06 判 FAIL。 |
| H08 | 游戏页面布局、按钮可见性和状态刷新 | FAIL | FAIL→FAIL | 原生按钮可见且点击送达，三 Web viewport 冒烟通过；第二局仍显示上一局已结算、退出后旧结果复现，整体状态刷新 FAIL；wechat-full-ui.json。 |
| AUTO01 | Go 单元测试 | PASS | PASS→PASS | Go 单元563 PASS/1 SKIP；go-unit.log。 |
| AUTO02 | Go Race 测试 | PASS | PASS→PASS | Go Race563 PASS/1 SKIP；go-race.log。 |
| AUTO03 | Go Vet | PASS | PASS→PASS | go vet -p 1 ./... 退出0；go-vet.log。 |
| AUTO04 | Go fmt-check | PASS | PASS→PASS | make fmt-check 退出0；fmt.log。 |
| AUTO05 | MySQL 集成测试 | PASS | PASS→PASS | 真实 MySQL 集成602 PASS/1 SKIP；mysql-integration-scale-clean.log。 |
| AUTO06 | 100/1000 牌局专项测试 | PASS | PASS→PASS | 100/1000 hand 和独立 receipt profile 实际执行退出0；mysql-integration-scale-clean.log、receipt-scale-profile.log。 |
| AUTO07 | 前端 TypeScript 类型检查 | PASS | PASS→PASS | 当前 TS 严格检查与真实引擎检查；client-foundation.log、web-independent-strict-source-settings.log。 |
| AUTO08 | 客户端单元测试 | PASS | PASS→PASS | 31 core+37 doudizhu+39 Cocos=107 测试；client-foundation.log。 |
| AUTO09 | 安全组件测试 | PASS | PASS→PASS | HPKE9测试、实际 TS↔Go envelope 和 manifest；hpke.log、crypto-interop.log。 |
| AUTO10 | Cocos Creator Smoke | FAIL | PASS→FAIL | 正式 Linux wrapper 实际退出1（底层CLI34），缺 portable 包依赖；linux-native-wechat-build.log，N03；补环境后的真实 SDK 冒烟通过不覆盖正式失败。 |
| AUTO11 | Production Compose Runtime | PASS | PASS→PASS | 原 production Dockerfiles/HTTP WS HTTPS WSS hardening，隔离 Compose 退出0/340.502秒；production-compose-runtime-configured.log。 |
| AUTO12 | 数据库架构边界测试 | PASS | PASS→PASS | 数据库/schema/module import 边界正式测试通过；go-unit.log、mysql-integration-scale-clean.log。 |
| AUDIT01 | P0：数据丢失、敏感信息泄露、身份越权、不可恢复的数据破坏 | PASS | PASS→PASS | 本轮独立源码/负例/真实环境审计及8个问题的等级、位置、前提、步骤、实际/预期、证据、影响和建议，见第10节；PASS 表示审计交付完成，非产品无缺陷。 |
| AUDIT02 | P1：重复出牌、状态分叉、错误结算、恢复失败、死锁及关键链路中断 | PASS | PASS→PASS | 本轮独立源码/负例/真实环境审计及8个问题的等级、位置、前提、步骤、实际/预期、证据、影响和建议，见第10节；PASS 表示审计交付完成，非产品无缺陷。 |
| AUDIT03 | P2：边界条件、资源泄漏、错误处理、用户操作异常、可维护性问题 | PASS | PASS→PASS | 本轮独立源码/负例/真实环境审计及8个问题的等级、位置、前提、步骤、实际/预期、证据、影响和建议，见第10节；PASS 表示审计交付完成，非产品无缺陷。 |
| AUDIT04 | 严重级别 | PASS | PASS→PASS | 本轮独立源码/负例/真实环境审计及8个问题的等级、位置、前提、步骤、实际/预期、证据、影响和建议，见第10节；PASS 表示审计交付完成，非产品无缺陷。 |
| AUDIT05 | 文件及代码位置 | PASS | PASS→PASS | 本轮独立源码/负例/真实环境审计及8个问题的等级、位置、前提、步骤、实际/预期、证据、影响和建议，见第10节；PASS 表示审计交付完成，非产品无缺陷。 |
| AUDIT06 | 触发条件 | PASS | PASS→PASS | 本轮独立源码/负例/真实环境审计及8个问题的等级、位置、前提、步骤、实际/预期、证据、影响和建议，见第10节；PASS 表示审计交付完成，非产品无缺陷。 |
| AUDIT07 | 可复现步骤 | PASS | PASS→PASS | 本轮独立源码/负例/真实环境审计及8个问题的等级、位置、前提、步骤、实际/预期、证据、影响和建议，见第10节；PASS 表示审计交付完成，非产品无缺陷。 |
| AUDIT08 | 实际行为 | PASS | PASS→PASS | 本轮独立源码/负例/真实环境审计及8个问题的等级、位置、前提、步骤、实际/预期、证据、影响和建议，见第10节；PASS 表示审计交付完成，非产品无缺陷。 |
| AUDIT09 | 预期行为 | PASS | PASS→PASS | 本轮独立源码/负例/真实环境审计及8个问题的等级、位置、前提、步骤、实际/预期、证据、影响和建议，见第10节；PASS 表示审计交付完成，非产品无缺陷。 |
| AUDIT10 | 建议修复方向 | PASS | PASS→PASS | 本轮独立源码/负例/真实环境审计及8个问题的等级、位置、前提、步骤、实际/预期、证据、影响和建议，见第10节；PASS 表示审计交付完成，非产品无缺陷。 |
| GOAL1 | Real platform-code login exchange / session validation, token lifetime and secure private credential management; no fake identity, no tokens in URLs; include local authorized dev login path only if correctly access-controlled. | PASS | PASS→PASS | 真实 wx.login→官方 exchange→稳定内部账号、session/JWT/WSS 与吊销；authentication.json、go-unit.log；凭据未入库，无 URL token。 |
| GOAL2 | Account-scoped rooms: create/join/leave/ready, seats, start hand, authoritative room refresh/reconnect and timeout/duplicate reconciliation; full Cocos interactive flow. | FAIL | FAIL→FAIL | 原生创建/加入/Leave/精确重试原复现通过；CLOSED 读取和退出后旧状态复活失败；wechat-full-ui.json，N05/N06。 |
| GOAL3 | Fair lifecycle: server-bound commit/reveal key/AAD acquisition, secure entropy + HPKE, trusted beacon, deal, private hand and evidence verification, fail closed on missing security context. | PASS | PASS→PASS | 真实安全熵/manifest/AAD/HPKE/drand、私牌隔离、证据校验与三轮恢复凭证闭环；full-three-games.log、wechat-full-ui.json；人类备份 UX 单列 E03。 |
| GOAL4 | Playable game: bidding, landlord determination, legal play/pass delegated to authoritative Go rules, turn indicators, cards/selection, clear error and version-retry handling, termination, score/evidence/result UI. | FAIL | FAIL→FAIL | 原生完整首局正确结算；下一局旧 terminal/取消重入404及拒绝 Expire fence 仍违约；wechat-full-ui.json、expiry-sql-final-real-archive.log，N01/N02/N04。 |
| GOAL5 | B-level MySQL recovery: append each accepted live bid/play/pass state transition and dedup + owner epoch/fence atomically **before acknowledgement**; bounded encrypted checkpoint and ordered replay restoring game state, ambiguity handling and owner fencing; address seed and other secret continuity; no plaintext player hands in routine logs. | FAIL | PASS→FAIL | 正常 durable log-before-ACK 与 checkpoint/HMAC 回归通过；失败终止错误 epoch、EXPIRED 冷回执/归档缺口未恢复；N01/N07。 |
| GOAL6 | Real 3-player repeatable E2E, reconnect and crash-injection: pre-/post-COMMIT kill, response loss, duplicate/out-of-order commands, competing owners, wrong epoch and replay mismatch; preserve every acknowledged action. | FAIL | FAIL→FAIL | 三局真实 Go 身份/WSS 故障与原生完整一局已执行；异常终止提交后 crash 恢复、第二局状态失败；N02/N07。 |
| GOAL7 | Performance and stability: bounded queues/workers/room memory, P50/P95/P99 measurements and transaction fsync budget on stated hardware, no unknown target claims. | FAIL | BLOCKED→FAIL | 测量门禁已执行，100 个归档终止局历史 map 无上限/TTL，不满足 bounded-memory；terminated-history-retention.log，N08；生产容量/fsync SLA 未资格化。 |
| GOAL8 | WeChat Creator package, actual developer tool, approved HTTPS/WSS domains and real handset evidence where access is available. Record platform-owned AppID/domain/device blockers honestly. Preserve Web and Douyin build compatibility. | BLOCKED | BLOCKED→BLOCKED | BLOCKED：真实 SDK/login/AppID/Secret 已具备；核准生产域名/TLS/平台白名单、实际手机和抖音 SDK/设备及 Linux exporter 资格未具备，不得说微信未登录。 |

统计：**89 PASS + 20 FAIL + 5 BLOCKED = 114**，ID无重复/遗漏。较Round8的99/9/6，原9个FAIL在扩展要求下仍FAIL，新增9个PASS→FAIL（含D08完整证据保留），G02/GOAL7从BLOCKED→FAIL（现有可执行反例），H05从PASS→BLOCKED（当前单一Linux基线不继承旧Creator构建）。因此不是产品通过率降低的简单趋势比较，而是新缺陷和更明确证据/工具链范围。

## 12. 未验证的平台、运维和发布条件

- C09/C10：完整journal HMAC换根、new-only恢复、旧根库存/迁移/安全退休。已验证AES重包和错根拒绝，不冒充完整根轮换。
- E03：自动化确实先备份再commit；真实用户备份确认、实际设备退出/重装找回和UX未验证。
- H05：固定Linux4原生Douyin exporter不存在/未执行，旧3.8产物仅历史。Douyin真实SDK/熵/设备、3D/远程分包未资格化。
- GOAL8：已登录、AppID/AppSecret及官方换证已实测。欠缺的是核准生产域名/TLS/平台白名单和真机实测等；不是以“微信未登录”作为阻挡理由。测试局使用本机信任链HTTPS/WSS，不替代生产域名审核。
- 完整客户端进程重启后的终局恢复、每个成员失效/冲突/pending手局命令排列、实际旧房间事件隔离和三台手机端到端未逐一原生实测；已执行范围在第3/4节，不能从单测推断真机PASS。
- 长期soak、生产硬件容量/SLA、独立fsync预算、手机前后台事件真实投递/分享/移动网络、正式构建资源剔除/上传/体验版/审核/发布未验证。工具的storage与listener注册/移除只覆盖其明确范围。
- 既有扩展源码的agent复查属于自审；未安排另一独立作者审查。既有历史报告/旧产物全部保留，没有用静态检查、mock或历史成功替代本轮可执行门禁。

## 13. 最终结论与交接

**Round9总体 FAIL，内部代码/完整集成产品FAIL，外部平台及真机资格BLOCKED，正式发布不合格，封板NO。** R8-N01、N02、N03的原始问题均已修复并独立重跑通过，但其扩展契约分别仍因N01/N07、N06、N02/N04失败。另有正式构建、CLOSED还原及历史资源问题。不存在把三原问题消除或CI绿直接等同Goal完成的结论。

交回ChatGPT：修复N01–N08及相应测试，尤其拒绝终止原子性、EXPIRED历史回执/归档补偿、连续局hand切换、异步上下文隔离、取消终态契约、CLOSED快照、冷构建依赖与有界缓存；自行完整验证并push一个新固定main候选后，再交Codex独立重验受影响项和全量门禁。N01需保留内部调用前提并进一步明确生产Supervisor可达性，不通过夸大风险代替修复契约。

Goal0045与0046均继续 `in_progress`，未归档，未修改原始要求。验收任务交付完成不等于产品Goal完成。该报告的文档提交只发布本轮结论，不是新的产品验收SHA。

## 14. 证据保护、清理与报告复查

私有证据根目录 `$E`权限0700；其中配置、账号/JWT、原始kits/私牌/签名及HMAC密钥、截图、故障日志和构建产物仅保存在该私有目录或本轮隔离副本，不进入git。已知私有密钥/Secret/JWT值扫描覆盖拟提交文档及当前导出产物；扫描结果和范围在 `secret-scan.json`，不输出真实秘密，不宣称覆盖运行内存/所有潜在编码。客户端允许携带的公开签名key/AppID不与服务端秘密混淆。Web调试资源未被当作可发布包。

`cleanup.json`记录仅关闭本轮codex-r9后端/GUI/build与codex0046r9的TLS/MySQL/Redis；GUI先前已停止，收尾stop得到unit not loaded是既有终止状态，不是重新启动理由。Production runtime脚本清理其自有项目/卷；其余本轮验收数据库卷与证据/工具缓存保留。自有19227/28888/28889/28999/28443/28080/29081/29443等测试监听不再存在；其他服务未清理，生产DB/Kubernetes/既有凭据和旧轮报告未变。未用全局prune或启用swap。

报告自检（文档自审，不冒称额外独立产品审查）：逐条核对114原始ID/要求/状态/本轮证据、8问题的源位置及前提，检查原始失败和fixture错误不混淆、真实SDK与controller/模拟器与真机不混淆、正式构建与环境补充不混淆、固定产品SHA与文档SHA不混淆。`matrix-round9.json`、`closeout-summary.json`和`evidence-sha256.json`保留机器可读索引。最终git diff限定本报告和current允许的Status/Working/Verification/Completion段；历史报告及产品源码未改。提交/推送遵循main-only、无强推。
