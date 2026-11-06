# Goal 0046 — Round 10 全量独立验收

日期：2026-11-06（UTC）。执行者：Codex。**总体/内部产品验收 FAIL；99 PASS / 10 FAIL / 5 BLOCKED = 114。** Round9八项原缺陷中7项完成可靠复测并通过，**R9-N01/P1仍失败**；新增 **R10-N01/P2**。未建立新的P0/P1/P3。矩阵10个FAIL不是10个独立缺陷。外部平台/真机资格BLOCKED，正式发布NO，Goal0045/0046继续 `in_progress`。

## 1. 固定版本、工作流与阅读范围

全部产品检查固定在 **`8f3f90c1ff02a80f167a6dc89dfbf390be3bfd5e`**。开始时clean main，从前轮报告提交f1067fa执行 `git pull --ff-only origin main`，恰好快进到候选；未切分支、混入新源码或自行修复。原有AGENTS/原则、current、0046 integrated、Round7/8/9及相关历史证据、Linux构建/PLATFORM-VALIDATION要求均纳入审查；本轮新增修复说明与实际差异独立核对，未采用开发方“八项均修复”的结论。历史报告保留。

只有本报告及current允许的Status/Working/Verification/Completion状态文字提交main。**文档提交SHA不是产品候选SHA**；文档提交可用 `git log -1 --format=%H -- docs/goals/0046-round10-independent-acceptance.md`查询，最终交付同时给出两者。

候选SHA的[CI / ci/full 42074767696](https://github.com/oublie6/awesome-zero-platform/actions/runs/42074767696) **6/6 SUCCESS**，[Fair Doudizhu 42074767729](https://github.com/oublie6/awesome-zero-platform/actions/runs/42074767729) **4/4 SUCCESS**。已独立读取全部job/step元数据，见 `ci-exact-source.json`。该SHA没有可据以声称成功的Cocos native-export Actions运行；hosted源检查/缺席或跳过的导出job不是原生构建。以下本地导出、SDK与故障运行是新证据，CI绿不覆盖产品失败。

## 2. 环境、隔离与证据约定

私有证据根记为 **`$E=/root/.config/awesome-zero-platform/acceptance-0046-r10-8f3f90c`**，下文短文件名均相对该目录。`runs.jsonl`记录命令、cwd、UTC开始、耗时、退出码和完整源码SHA；配置、kits、JWT、私牌、日志和截图不提交。未复制上轮执行结果；复用的是外置测试脚本及已安装工具，结果均由当前候选新执行。

| 项目 | 本轮实际值与边界 |
|---|---|
| 主机 | Linux5.15.0-202-generic/x86_64，8 vCPU，Xeon E5-2699C v4@2.20GHz，物理内存约7.75GiB，swap0 |
| 编译/并发 | Go1.25.8、Node22.23.4；GOMAXPROCS2、Go `-p 1 -parallel 1`；重型任务顺序执行 |
| 数据库 | 自有Docker项目codex0046r10，真实MySQL5.7.44/Redis8.8.0；实际fsync设置1、buffer pool67108864B、max_connections64、binlog0 |
| 配额/池 | MySQL768MiB/1.5CPU；规模测试16worker、测试池上限24；CLI3GiB/1CPU/no swap，GUI3GiB/1.5CPU/no swap |
| 数据/服务 | 自有integration库awesome_zero_platform、product库codex0046r10_product和旧schema库codex0046r10_upgrade；仅本轮容器暴露loopback测试端口 |
| 密钥/账号 | 新生成本轮local密钥和正常测试账号密码；仅沿用用户授权的APP_WECHAT_*换证配置；没有伪造JWT/玩家身份 |
| 网络 | 自有SAN证书/CA与明确SPKI pin，实际HTTPS/WSS鉴权、SDK urlCheck=true；不是关闭TLS校验，也不代表生产域名审核通过 |
| 微信 | 已有community Linux工具2.02.2609240-2，SDK3.17.5，工具已登录，AppID/AppSecret确实可用 |
| 客户端层级 | 三个不同合法Go认证账号；原生局为真实wx账号+两个Go认证peer；不等于三台真实手机 |

44个Go overlay映射经 `overlay-scope-audit.json`核对，全部为候选不存在的新 `_test.go`及私有fixture，未覆盖生产文件或正式测试。SQL事务、journal、protected seed、runtime、archive和故障是真实执行；部分测试使用仓库确定性的genesis/clock/公平fixture，未将其当作真实平台登录或drand证据。真实公平/平台证据由五局实际对局和SDK提供。

## 3. Round9八项缺陷逐项结论

| 问题 | Round10 | 原始失败路径与本轮有效证据 |
|---|---|---|
| R9-N01/P1 拒绝Expiry错误fence | **FAIL** | 独立真实MySQL原探针仍observed epoch2→3、domain BIDDING/v12、水位1，后续合法Bid被fence；priority-expiry-real-sql.log |
| R9-N02/P1 第二局沿用旧终态 | **PASS** | 实际同房两完整原生局；第二局新hand从空store进入BIDDING/PLAYING并完成；未授权/旧snapshot、changed、evidence拒绝，当前合法事件接受；wechat-full-ui.json、priority-client-event-isolation.log |
| R9-N03/P2 正式Linux构建缺依赖 | **PASS** | 正式build.cjs自动安装复制目录portable依赖，真实WECHAT导出成功，无人工补目录；linux-native-wechat-build.log |
| R9-N04/P2 未发牌取消指向假归档 | **PASS（原问题）** | 三人实际未发牌取消后不返回错误lastHandId，原生第三局取消后Reconnect成功；实际丢取消结果后完整原响应deepEqual精确重试、重复重连及旧完成归档保留；leave-fairness-real.json、wechat-full-ui.json、cancel-repeat-closed-round10.json（前两成功检查）；另有不同的成员变更问题R10-N01 |
| R9-N05/P2 CLOSED空座Restore失败 | **PASS** | 实际最后owner原生退出，真实MySQL CLOSED空座及GET正常；join/ready及原owner start拒绝；非CLOSED不变量正式单测保持严格；wechat-full-ui.json、cancel-repeat-closed-round10.json、closed-start-focused-round10.json、go-unit.log |
| R9-N06/P2 迟到Evidence污染 | **PASS** | 真实认证HTTP结果在return边界暂停，分别在第二局BIDDING、Leave、换房、原生登录/控制器替换后释放；均不回写旧store/分数，合法当前证据仍可用；wechat-full-ui.json |
| R9-N07/P1 EXPIRED冷回执/归档缺口 | **PASS** | 真实archive热/冷原回执相同，实际终止COMMIT后归档前SIGKILL，恢复/精确retry仅archive1；ABORTED/NO_LANDLORD双冷恢复、篡改/身份/未知ID拒绝；priority-expiry-real-sql.log、priority-terminal-integrity-and-aborted.log、terminal-cache-120.log |
| R9-N08/P2 结束历史资源无界 | **PASS（本轮范围）** | 同runtime120个真实归档终止局，热缓存64、淘汰后/新runtime原回执仍准确；active/epochs/seed RAM0，无效ID不扩大缓存；terminal-cache-120.log、terminal-history-throughput.log；不是长期零泄漏/容量资格 |

R9-N01仍是内部application调用条件：过时ExpectedVersion、有效ExpectedLiveVersion且sequence guard未先拒绝，外置探针采用正常ClientSeq==ExpectedVersion形状。**没有证明正常生产Supervisor必然自然生成该组合**；WSS无可直接选择的Expire类型，不声称远程攻击或所有正常对局会随机超时。任务明确要求“被拒绝终止不错误fence”，此契约已被真实SQL反例否定，不能降标准判PASS。

## 4. 真实MySQL竞争、故障与B级恢复

本轮重跑Round9原始外置SQL/race探针：**15 PASS / 1 FAIL**，唯一失败为R9-N01。并发交错不以port模拟代替数据库。

| 交错/故障 | 观察与判定 |
|---|---|
| t4已提交、返回/AfterBid延迟；t6旧Sweep；释放后t7 | BIDDING、BIDDING→PLAYING、PLAYING三组实际SQL/Service/runtime/Supervisor均不错误终止/回退，PASS |
| 新动作先提交，旧expiry后拿writer锁 | ErrLiveProgressed回滚、不留终止回执/epoch改变，PASS |
| 域行已锁，writer锁前新commit | 新commit获胜，旧expiry回滚，PASS |
| expiry先锁writer、新Bid实际锁等待 | INNODB_LOCK_WAITS确认，成功expiry fence旧writer，PASS |
| Fence之后基础设施失败/真实InnoDB死锁 | 事务完整回滚、epoch/水位/域/回执恢复，deadlock分类可重试、原请求retry成功，PASS |
| 全部事务修改后、真正Store COMMIT前KILL自有连接 | COMMIT invalid connection、无ACK、无半提交、旧writer仍可合法继续，PASS |
| 终止COMMIT前子进程SIGKILL | 只保留先前takeover，tentative fence/终止回执回滚，冷恢复新动作成功，PASS |
| 终止COMMIT后、Archive前SIGKILL | 实际EXPIRED/receipt1/archive0；冷恢复与原ID retry后archive1，未复活可写游戏，PASS |
| 正常Sweep、精确系统ID重试、再Sweep | 一次终止/fence/archive，无额外业务影响，PASS |
| 业务版本拒绝 | 仍提交fence且合法后续Bid失败，**FAIL R9-N01** |

三真实Go身份局的 pre-/post-COMMIT进程强杀、接收端丢真实结果、同ID精确重试、16路重复/不同ID同seq、跨actor/type/aggregate/payload/seq绑定、错误epoch及损坏checkpoint/journal均执行。domain版本与live stateVersion属于不同命名空间，不机械强制相等；按其已发布语义核对水位/回执/最终记录。

EXPIRED原探针用真实MySQL archive，不能只靠新 `live_terminal_history_integration_test.go`通过；已审查该测试的fixture和noop archive边界，并额外验证实际存储和真实子进程SIGKILL。ABORTED/无人叫地主独立案例两次冷恢复都只有一份archive、原actions数量不变，不接受新的未知动作。冷历史查询的错误actor/version/score/clientSeq和实际SQL篡改序号/digest/payload均拒绝，恢复原行后正向精确读取仍成功。

## 5. 正式Linux Cocos构建、严格类型与产物

官方CLI revision **`f506fa3de403b48b4833334fa48dddd27bb26364`**；engine checkout **`d0a696cfe63fd8d19c70d7913ba11aa641c2e3c7`**、package **4.0.0-beta.0**（SDK runtime显示4.0.0）；Node22.23.4。本轮实查pins/extension所有权并运行extension测试；没有使用Wine、其他OS虚拟机、旧Creator3.8或全局tsc shim。

正式命令 `node scripts/cocos-linux-wechat/build.cjs "$E/native-output"` 通过，真实CLI单位耗时223.346秒（包含准备的父入口225.335秒），自动安装干净复制目录依赖，真实WECHAT入口/adapter/资源/landscape/BigInt审计和引擎strict检查通过。不是Web改名或补环境后冒充正式脚本成功。

原始微信包 **3,079,988B**，sorted per-file manifest SHA256 **`6aa0fc424b01604115034e7491f3872b4fe5a98fbc814ac1db70c0dc2d3dd4ed`**。`linux-native-provenance.json`保留逐文件hash；`native-current-input-binding.json`核对124个tracked client输入，无意外差异（package JSON允许仅格式相同）。哈希在操作员SDK/AppID/project配置前记录，后续配置变化另记，不能冒称整个目录始终字节不变。SDK测试通过产品公开配置入口注入本轮HTTPS/WSS URL和公开manifest验证根，未改应用/engine JS或鉴权逻辑；不代表生产端点已配置。

Linux平台扩展早期由当前agent在用户授权下实现，对那部分源码复查属于**自审**，不包装成另一作者独立审查；本轮固定候选正式入口/生成严格检查/真实SDK操作结果均为新执行证据。

Web另用同一当前隔离源码和真实官方CLI导出：158.318秒退出0。原始包浏览器因已知Math.pow(BigInt)启动错误退出1/93.467秒；原包保留。独立副本执行**仓库既有** `scripts/fix-cocos-bigint-power.cjs`后，三viewport/字体/帧缓冲/销毁冒烟通过8.219秒；当前真实cc4生成声明+原项目strict选项独立检查通过4.084秒。没有新写产品修复或关闭严格检查，H02/H03仅接受这套既有完整构建配方，**不称原始Web启动PASS**。原始/运行包hash及仅generated文件变化见 `web-provenance.json`。CLI过滤非assets声明诊断的边界保留，不用其退出0替代独立strict检查。

## 6. 实际SDK、三人完整局与同房连续局

`full-three-games.log` **PASS301.139秒**：四个正常服务端认证账号，每局三人不同席位；提交前外部私有备份→新Node进程导入→manifest/AAD/HPKE→真实Quicknet/drand/BLS→51互斥私牌→叫分/出牌/过牌→结算/归档/证据。覆盖原N01跨阶段Bid、终局丢响应重启精确回执、真实活跃SQL故障及SETTLING同进程补偿；五次原终局retry无重复action/archive/settlement/outbox。

正式导出的微信产物在真实community Linux工具/SDK3.17.5执行，`linux4-native-two-games.log` **PASS647.643秒、69检查、0失败**：

| 原生局 | 终局版本 | 原生出牌/过牌 | 真实Go强杀恢复 | 最终digest |
|---|---:|---|---|---|
| 1 | 63 | 20/1 | 18→18，epoch推进 | `7d23eb22a73fd2b34960cd0bcc957827bc997be2540947e0fe609b4e137cf915` |
| 2 | 63 | 20/1 | 18→18，epoch推进 | `96fe070d0b0c25ac164a28e0669499fbdcf72fdc46172a08b4caa8f349f73022` |

两局均为原生owner操作，无后台代打；两个peer通过正常认证WSS按权威Go规则操作。第二局切到新hand，旧COMPLETED清空，低版本1可正常进入BIDDING/PLAYING；授权前跨hand注入和旧WSS snapshot/changed/evidence被拒绝。两次真实Go SIGKILL恢复后继续完成，终态显示实际胜座/积分/摘要，无旧可操作手牌/叫分/出牌/过牌。两张终局截图已人工查看；SDK缩放画面字体较小、存在GL ReadPixels/SDK提示，不称零告警或手机可读性资格。

原生Create/Join/Leave、最后owner关闭、他人在场拒绝owner离开、活跃手局拒绝离开、真实丢Leave ACK→15秒未知提示→同requestId完整DTO精确重试/一次席位变更均通过。第三未发牌局取消后原生重连成功；取消是公开授权API异常路径，不冒充原生Cancel按钮。独立controller真实公平commit结果未知时Leave发包0、原DTO重试解未知后继续，见 `leave-fairness-real.json`。

四个真实HTTP终局结果在解析校验后、返回边界受控延迟：第二局BIDDING、离房清空、换到另一房间、原生LoginEntry替换控制器后释放，均无旧store/旧分数回写；当前合法证据仍接受。原生登录的控制器替换不是完整微信客户端进程重启，后者仍未做终局恢复穷举。

SDK独立补充 **PASS97.011秒**：当前源码生成的真实platform/session/identity组件，wx.login→Go→官方换证、两次code稳定AccountID、session restore/revoke、WSS/pong、同ID普通命令重放、安全随机32B两次不同、storage自有key往返和listener注册/移除；自有tokens注销。该补充取得SDK上下文时Cocos scene未ready，仅证明SDK层；两原生局另有完整scene证据，二者不互相冒充。无上传/预览发布/正式发布，没有三手机证据。

## 7. 历史与兼容性回归

| 问题/范围 | 本轮证据与结论 |
|---|---|
| F01实际signed manifest/reveal契约 | 三Go局+两原生局当前manifest/AAD/HPKE成功，原复现PASS |
| F02/N01跨阶段及COMPLETED历史回执 | original-N01.log、terminal-recovery/completed-history及原MySQL外置测试，PASS；EXPIRED新增修复另见第4节 |
| F03 genesis换座/私牌隔离 | 真实MySQL篡改拒绝和第四账号403，PASS |
| F04完整归档/安全prune | 实际完整归档clone、语义破坏拒绝/30天/dry-run/execute/删除中SIGKILL原子回滚，PASS |
| F05/F11房间空座、丢创建结果协调 | 原生1/2/3席位解析、发送前后丢结果两分支实际执行，原复现PASS；新增成员403另列 |
| F06/F07公平已知拒绝与dispose | original-client-repro、client-lifetime-matrix当前源码回归通过；JS清零为best effort |
| F08/F09/F10部署配置/Quicknet/YAML | 五配置转发、无legacy HMAC真实provider配置、stock YAML readiness及production runtime（330.883秒），PASS；私有provider fixture漏文件的修正见第11节 |
| R3-N02/R7-N03生命周期 | 原历史请求不续期/不重调度、乱序回调单调、重复dealt不回退/延长期限和SETTLING补偿，23外置测试+实际游戏通过；R9-N01另FAIL |
| R3-N03/R4-N04/R5-N05 | 全类型重启seq/actor/指纹绑定、HMAC根/降级/序号/物理顺序/keyless/旧数据拒绝35项通过 |
| R7-N04/WX-N01/WX-N02 | 当前Linux SDK实际Set/callback清理/重连，两次正式平台换证及普通同ID重放、拒绝空events[]冷重放通过；不重跑历史Creator3.8 |
| 旧schema升级边界 | legacy-clone/upgrade实际执行：旧schema启动拒绝且无自动DDL；仅测试库显式加三列后，未知身份仍seq0/v0/NULLtag，认证journal拒绝未签旧v0，不猜测身份或伪造HMAC |

代码边界、go-zero/公开协议、数据库schema/import约束通过本轮全仓unit/integration及源码审查；生成/go.mod tidy复制检查未改变tracked server文件。未执行生产数据库迁移。历史报告和旧工具链证据保留，不能作为当前未执行平台的PASS。

## 8. 性能、安全与资源

本轮实际MySQL/配额见第2节。以下原始测量行来自当前日志；总恢复耗时不冒充单局P95，runtime直接调用吞吐不冒充HTTP/WSS吞吐或生产容量。

`mysql-integration-scale.log`：

```text
live_journal_perf_integration_test.go:94: MySQL COMMIT-before-ACK sample=80 p50=7.087368ms p95=18.583771ms p99=26.270745ms max=30.055192ms (CI runner, not production SLA)
live_scale_integration_test.go:39: MySQL live profile hands=100 setup=720.921408ms recovered=1.079180901s heapDelta=305848 bytes commit p50=19.241092ms p95=45.753203ms p99=51.800827ms max=60.828276ms; workers=16 (CI/hardware dependent)
```

`official-scale-1000.log`：

```text
live_scale_integration_test.go:39: MySQL live profile hands=1000 setup=9.399041769s recovered=15.00683489s heapDelta=3063056 bytes commit p50=22.823118ms p95=81.176728ms p99=189.948853ms max=206.389794ms; workers=16 (CI/hardware dependent)
```

`checkpoint-latency.log`：

```text
codex_crypto_bounds_test.go:113: actual game encrypted checkpoint version=33 watermark=60 payloadBytes=76959 writes=100 p50=3.816794ms p95=5.883736ms p99=6.467606ms max=6.942601ms; includes real MySQL transaction/fsync, not production SLA
```

`receipt-scale-profile.log`：

```text
codex_receipt_profile_test.go:16: 100-new-bid samples=100 p50=8.10617ms p95=27.089583ms p99=34.128791ms max=36.328554ms
codex_receipt_profile_test.go:16: 100-new-play samples=100 p50=8.247932ms p95=20.510279ms p99=54.801118ms max=55.492143ms
codex_receipt_profile_test.go:16: 100-new-pass samples=100 p50=8.13181ms p95=33.80974ms p99=47.479515ms max=258.406169ms
codex_receipt_profile_test.go:17: hands=100 recovery=1.603619385s postGCHeapDelta=692440
codex_receipt_profile_test.go:19: 100-receipt-bid samples=100 p50=23.52µs p95=45.435µs p99=123.695µs max=246.744µs
codex_receipt_profile_test.go:19: 100-receipt-play samples=100 p50=28.612µs p95=52.164µs p99=62.994µs max=91.248µs
codex_receipt_profile_test.go:19: 100-receipt-pass samples=100 p50=28.885µs p95=59.167µs p99=376.924µs max=415.269µs
codex_receipt_profile_test.go:20: 100-contended-same-id-16workers samples=1600 p50=43.239µs p95=1.568171ms p99=10.468182ms max=17.553486ms
codex_receipt_profile_test.go:16: 1000-new-bid samples=1000 p50=7.665354ms p95=23.455162ms p99=36.641483ms max=509.742584ms
codex_receipt_profile_test.go:16: 1000-new-play samples=1000 p50=7.665503ms p95=25.425511ms p99=46.033826ms max=207.132029ms
codex_receipt_profile_test.go:16: 1000-new-pass samples=1000 p50=7.675113ms p95=20.271685ms p99=39.779136ms max=103.38502ms
codex_receipt_profile_test.go:17: hands=1000 recovery=14.959981465s postGCHeapDelta=6918528
codex_receipt_profile_test.go:19: 1000-receipt-bid samples=1000 p50=31.006µs p95=44.389µs p99=85.515µs max=1.046759ms
codex_receipt_profile_test.go:19: 1000-receipt-play samples=1000 p50=39.746µs p95=65.965µs p99=110.497µs max=15.138903ms
codex_receipt_profile_test.go:19: 1000-receipt-pass samples=1000 p50=26.449µs p95=52.033µs p99=76.912µs max=214.44µs
codex_receipt_profile_test.go:20: 1000-contended-same-id-16workers samples=1600 p50=37.529µs p95=1.232778ms p99=4.256631ms max=20.449725ms
```

`terminal-history-throughput.log`：

```text
codex_terminal_throughput_r10_test.go:9: hot-valid samples=100 elapsed=4.038523ms completedRequestsPerSecond=24761.53 p50=20.002µs p95=58.078µs p99=1.307979ms max=1.307979ms (sequential fixture, not capacity/SLA)
codex_terminal_throughput_r10_test.go:9: cold-valid samples=100 elapsed=835.869037ms completedRequestsPerSecond=119.64 p50=7.470834ms p95=13.698707ms p99=24.736136ms max=24.736136ms (sequential fixture, not capacity/SLA)
codex_terminal_throughput_r10_test.go:9: cold-wrong-actor samples=100 elapsed=1.098713152s completedRequestsPerSecond=91.02 p50=7.95255ms p95=23.339477ms p99=111.544752ms max=111.544752ms (sequential fixture, not capacity/SLA)
codex_terminal_throughput_r10_test.go:9: cold-unknown-id samples=100 elapsed=116.731947ms completedRequestsPerSecond=856.66 p50=994.93µs p95=1.975109ms p99=5.436315ms max=5.436315ms (sequential fixture, not capacity/SLA)
```

120个真实归档终止局：20/40/60局时热finalized/receipt为20/40/60，80/100/120局稳定64；active/epochs/seed RAM均0。全量冷恢复120局2.184秒；240个混合热/淘汰/冷回执P50 41.642µs/P95 10.469ms/P99 13.392ms/max13.866ms。200个未知ID及错误身份/版本/参数/seq不扩张缓存。heap增量含刻意保留的fixture/history对象，不能把其上升曲线认定为生产缓存永久持有；也不能仅凭本次64上限声称长期零泄漏。

1500真实WSS（60 warmup、cooldown）active0、goroutine26→27、自然GC6→26、heap3.720→7.127MB、RSS29.303→32.236MB；未强制GC粉饰连接回收，长期soak未资格化。已实际观察writer锁等待/死锁victim/回滚和16worker同ID争用。原生两局GUI内存约1.96GB，memory.events low/high/max/oom/oom_kill均0；结束已停GUI。

冷历史查询先检查ID存在，再按至多512条/单条16KiB的认证日志完整重放，无效ID不加载无限历史或永久扩张seed/receipt缓存。已知有效ID但错误actor仍会付出完整冷重放成本，实测分位数/串行吞吐如上；权限仍拒绝，不声称已认证抗滥用生产容量。没有用AES重包成功替代journal HMAC换根/退休证据。

## 9. 未关闭问题，交回ChatGPT

### R9-N01 / P1（残留）— 业务拒绝的Expiry仍推进epoch

- 位置：`server/business/doudizhu/application/hand_commands.go:277` Fence在`:287`域mutation之前；`application/service.go:146`将business rejection完成回执后提交事务。
- 前提：真实durable BIDDING、ExpectedLiveVersion匹配，但ExpectedVersion已过时且序号检查未先拒绝；内部application调用。未建立正常Supervisor自然可达性，不宣称公开WSS Expire攻击。
- 步骤：外置 `TestCodexExpirySQLRejectedMustNotFence`构建合法真实SQL hand/runtime；读epoch2/live watermark1/domain BIDDING-v12；发送stale域版本/正常seq形状Expire；读回持久化和返回值，再调用合法当前writer Bid。
- 实际：accepted=false/DDZ_VERSION_CONFLICT，却epoch2→3；watermark1和domain BIDDING-v12未变；下一Bid `verification failed / live writer epoch fenced`。追加真实SQL审计：room仍HAND_ACTIVE/v7，outbox0→0，receipt0→1且storedAccepted=false、acceptedReceiptCount0；异常副作用确实是writer epoch。独立原始与focused负例均FAIL，见rejected-expiry-state-audit.log。
- 预期：拒绝终止不提交fence、不破坏仍活跃writer；成功终止、rollback、死锁/提交失败各自维持正确线性化。
- 影响：内部失败终止可能使后续合法动作中断，需恢复接管；不是已证实丢失持久化动作。证据 `priority-expiry-real-sql.log`及 `expiry_sql_test.go`，其余15项真实SQL正对照通过。
- 建议：先完成域验证或把fence严格放在成功终止原子边界；保留业务拒绝回执语义、真实SQL反例，并补正常Supervisor可达性分析。文档修复说明不能代替实际代码。

### R10-N01 / P2（新增发现）— 新房间成员重连读取无权查看的旧局归档

- 位置：`server/business/doudizhu/application/room_queries.go:81`向任何当前成员返回lastHandId；`clients/fair-doudizhu-cocos/assets/scripts/game/GameOnlineController.ts:71`/`:72`无条件取该archive；`bootstrap/GameBootstrap.ts:343`刷新分支同样依赖它。
- 前提：A/B/C正常完成一局H，原房间仍开放；C退出，未参加H的合法账号D加入该房间。
- 精确步骤：使用本轮三个正常Go身份完成局；C真实leave；D真实join后GET room确认member=true/occupied3/lastHandId=H；D直接读H evidence，再调用真实 `reconnectAndResync(roomID)`；原A读H正对照成功。脚本finally使D离开并恢复C原成员关系。
- 实际：D加入成功，但归档按历史手局成员授权正确返回HTTP403；客户端重连同样抛HTTP403。新成员不应因合法加入而被不可用历史引用阻断。
- 预期：保持旧归档权限严格，房间只提供当前账号有权/有意义读取的历史locator，或客户端按明确契约处理无权限历史，不误作当前可恢复牌局。
- 证据：`new-member-history-round10.mjs`、`.json`、`new-member-history.log`；这是当前源码controller+真实认证HTTPS/WSS/数据库对局的反例，**没有声称原生新成员GUI复现**。其他两原生同房局PASS不覆盖成员替换。
- 影响：合法新成员的刷新/重连异常，属于产品契约/客户端处理问题，不是AppID/Secret或工具登录问题，也不是数据越权泄露。此次新发现，未证明该候选首次引入。
- 建议：明确room成员与historical hand成员的不同授权语义；修正lastHandId可见性/恢复契约，并加A/B/C完成、C离、D进、D重连和隐私403正对照，不能通过放宽归档权限修复。

## 10. 原始114项逐项矩阵

每行原始ID/要求保持不变，上一轮取Round9实际status。本轮列的源码缩写 **8f3f90c**均指第1节完整固定SHA；所有短证据路径为本轮新目录，不能指回旧运行。PASS只覆盖写明的执行范围，FAIL列明反例，BLOCKED说明缺失条件。AUDIT项PASS表示审计交付完成，不表示产品零缺陷。

| ID | 原始要求 | Round9 | Round10 | 源码 | 本轮方法、证据、范围/限制 |
|---|---|---|---|---|---|
| A01 | 真实身份认证和账号隔离 | PASS | PASS | 8f3f90c | 实际 wx.login 官方换证、两次稳定 AccountID、session 校验/吊销与四个 Go 测试账号隔离；authentication.json、full-three-games.log。 |
| A02 | 创建、加入、退出房间 | FAIL | FAIL | 8f3f90c | 本轮原生创建/加入/Leave、丢ACK原DTO重试通过；完成局更换成员后当前新成员重连HTTP403，R10-N01；wechat-full-ui.json、new-member-history-round10.json。 |
| A03 | 三人准备和开局 | PASS | PASS | 8f3f90c | 三账号真实三席准备、原生 StartHand 与三轮 WSS 开局；wechat-full-ui.json、full-three-games.log。 |
| A04 | 公平承诺与揭示 | PASS | PASS | 8f3f90c | 原生准备、外部私有备份、commit/import/HPKE reveal；三轮新 Node 进程导入恢复；wechat-full-ui.json、game1-reveal.log。 |
| A05 | 公共随机源验证及发牌 | PASS | PASS | 8f3f90c | 真实 Quicknet/drand BLS、签名 manifest/AAD、51 张互斥牌与三席私牌；game1-reveal.log、wechat-full-ui.json。 |
| A06 | 地主竞叫及确定 | PASS | PASS | 8f3f90c | 低叫分、多次叫分、原生叫 3 分进入 PLAYING；active-lifecycle-matrix.json、wechat-full-ui.json。 |
| A07 | 出牌、过牌、牌型比较 | PASS | PASS | 8f3f90c | Go 牌型全回归、三轮完整 WSS 对局及原生 20 次出牌/1 次过牌，无 owner 后台代打；go-unit.log、full-three-games.log、wechat-full-ui.json。 |
| A08 | 回合切换 | FAIL | FAIL | 8f3f90c | 正常三Go局/两原生局回合切换与迟回调三阶段对照通过；内部被拒绝Expire仍fence合法writer，后续Bid失败，R9-N01；priority-expiry-real-sql.log。 |
| A09 | 终局、计分及证据归档 | FAIL | PASS | 8f3f90c | 三Go局及同房两原生局正确COMPLETED/分数/digest，真实EXPIRED归档前SIGKILL后补一份归档；wechat-full-ui.json、priority-expiry-real-sql.log。 |
| A10 | 断线重连及权威状态同步 | FAIL | FAIL | 8f3f90c | 原生两局均真实GoSIGKILL v18→18，换局/终局/取消后重连通过；新成员接收不可访问旧archive引用后重连403，R10-N01；wechat-full-ui.json、new-member-history-round10.json。 |
| B01 | 已向客户端确认的操作必须先完成持久化提交。 | PASS | PASS | 8f3f90c | 实际 pre-COMMIT kill/连接 KILL 回滚、post-COMMIT 丢响应及精确重试，正常动作持久化先于 ACK；active-process-fault.json、priority-expiry-real-sql.log。 |
| B02 | 任何已确认操作不能在进程崩溃后丢失。 | PASS | PASS | 8f3f90c | 正常 Bid/Play/Pass 已 ACK 的动作行与版本在真实进程重启后保留；all-types-sequence-restart.json、active-process-fault.json；EXPIRED 历史回执可用性另在 B09 判 FAIL，不声称动作行被删除。 |
| B03 | 同一 requestId 的重复请求不能重复执行。 | PASS | PASS | 8f3f90c | 真实相同 ID 并发 16 路不重复业务动作、终局重试不重复结算，实际 Sweep/系统 ID 重试只执行一次；active-lifecycle-matrix.json、priority-expiry-real-sql.log。 |
| B04 | 相同 requestId、不同参数或不同操作者必须拒绝。 | PASS | PASS | 8f3f90c | 重启后篡改 actor/type/aggregate/payload/clientSeq 均拒绝，旧回执正对照通过；restart-binding-negatives.json、all-types-sequence-restart.json。 |
| B05 | 新服务接管后，旧 Owner Epoch 必须被正确隔离。 | PASS | PASS | 8f3f90c | 实际新 owner takeover、旧 writer 拒绝；MySQL writer-lock-first 正常终止对照通过；mysql-integration-scale.log、priority-expiry-real-sql.log。 |
| B06 | 检查点、日志、水位线和业务版本必须严格一致。 | FAIL | FAIL | 8f3f90c | 独立真实SQL原失败重现：known version rejection仍epoch2→3、domain BIDDING/v12/watermark1不变，合法writer后续Bid被fence；R9-N01，priority-expiry-real-sql.log；成功终止/死锁/连接失败回滚正对照通过。 |
| B07 | 检查点或日志被篡改时必须拒绝恢复。 | PASS | PASS | 8f3f90c | 真实 MySQL checkpoint、genesis、journal HMAC/序号/物理顺序篡改拒绝；original-mysql-regressions.log、hmac-historical.log。 |
| B08 | 恢复不能泄露其他玩家的私有手牌。 | PASS | PASS | 8f3f90c | 三账号/非成员实际私牌及归档授权拒绝，genesis 换座篡改拒绝；game1-reveal.log、wechat-full-ui.json、original-mysql-regressions.log。 |
| B09 | 终局重启必须保留最终证据和历史回执。 | FAIL | PASS | 8f3f90c | 真实已归档EXPIRED热/冷回执逐字节一致；ABORTED/无人叫地主两次冷恢复、120局淘汰后/新runtime原回执与篡改拒绝通过；priority-expiry-real-sql.log、priority-terminal-integrity-and-aborted.log、terminal-cache-120.log。 |
| B10 | MySQL COMMIT 前后发生 SIGKILL、网络断开、响应丢失时，不得出现重复执行或状态分叉。 | FAIL | PASS | 8f3f90c | 实际MySQL pre/post-COMMIT SIGKILL、响应丢失、事务连接KILL、真实死锁/回滚，以及终止COMMIT后归档前SIGKILL补偿均通过；active-process-fault.json、priority-expiry-real-sql.log。业务拒绝fence另按B13 FAIL，不把故障正例扩大为全部事务语义通过。 |
| B11 | 真实 MySQL 事务 | PASS | PASS | 8f3f90c | 真实 MySQL 5.7.44、锁等待/死锁/COMMIT 失败/回滚均实际执行；mysql-integration-scale.log、priority-expiry-real-sql.log、priority-expiry-real-sql.log；本项指真实事务验证，语义失败另列 B06/B10/B13。 |
| B12 | 并发重复提交 | PASS | PASS | 8f3f90c | 三真实 WSS 会话、16 路同 ID 并发原回执一致；不同 ID 同序号冲突；active-lifecycle-matrix.json、receipt-scale-profile.log。 |
| B13 | Epoch fencing | FAIL | FAIL | 8f3f90c | 独立真实SQL原失败重现：known version rejection仍epoch2→3、domain BIDDING/v12/watermark1不变，合法writer后续Bid被fence；R9-N01，priority-expiry-real-sql.log；成功终止/死锁/连接失败回滚正对照通过。 |
| B14 | 检查点加密 | PASS | PASS | 8f3f90c | 真实加密 checkpoint 写入/旧密钥/密文篡改和 epoch 负例；checkpoint-latency.log、original-mysql-regressions.log。 |
| B15 | 日志完整性 | PASS | PASS | 8f3f90c | 独立 HMAC 降级、客户端序号、重排/截断/错根/显式 keyless 测试 35 项通过；hmac-historical.log。 |
| B16 | 三人完整对局恢复 | PASS | PASS | 8f3f90c | 三Go完整对局及两原生完整局，活跃/终局真实进程强杀、丢响应、精确回执和SETTLING补偿通过；full-three-games.log、wechat-full-ui.json、settling-without-restart.json。 |
| B17 | 进程级故障注入 | PASS | PASS | 8f3f90c | 实际 SIGKILL 及实际 MySQL 连接 KILL，不以返回模拟错误代替所有故障；active-process-fault.json、priority-expiry-real-sql.log、priority-expiry-real-sql.log；执行通过不表示每个恢复不变量通过。 |
| B18 | 终局证据恢复 | FAIL | PASS | 8f3f90c | 正常完成归档保留、EXPIRED真实归档前SIGKILL后补偿一次、ABORTED/NO_LANDLORD重复恢复不重复归档；旧原生第二局在第三局取消/离房后新HTTP查询仍同digest；priority-expiry-real-sql.log、priority-terminal-integrity-and-aborted.log、cancel-repeat-closed-round10.json（前两检查成功）。 |
| C01 | 新记录使用当前主密钥加密 | PASS | PASS | 8f3f90c | 真实 MySQL 新 seed/checkpoint 当前根加密与正常读取；original-mysql-regressions.log 的 Rotation/Keyring 测试。 |
| C02 | 旧记录可以使用保留的历史密钥读取 | PASS | PASS | 8f3f90c | 真实 retained AES key 读取旧 seed/checkpoint，原始贡献一致；original-mysql-regressions.log。 |
| C03 | 历史密钥数量和长度严格受限 | PASS | PASS | 8f3f90c | keyring 数量/长度/重复/非法 key 边界；original-mysql-regressions.log、go-unit.log。 |
| C04 | 旧密钥不存在时拒绝错误解密 | PASS | PASS | 8f3f90c | 删除所需旧 AES key 后拒绝恢复；original-mysql-regressions.log，非静默重新生成。 |
| C05 | 重加密必须先认证旧密文 | PASS | PASS | 8f3f90c | 重包前认证旧密文，错误旧 key/篡改 ciphertext 拒绝；original-mysql-regressions.log。 |
| C06 | 种子重加密不得改变原始随机贡献 | PASS | PASS | 8f3f90c | seed 重包前后实际随机贡献一致；original-mysql-regressions.log、hmac-historical.log。 |
| C07 | 检查点重加密必须校验当前 Owner Epoch | PASS | PASS | 8f3f90c | checkpoint 重包错 epoch 被拒绝；original-mysql-regressions.log。 |
| C08 | 错误密钥、篡改密文不能恢复 | PASS | PASS | 8f3f90c | 实际错误密钥/篡改密文/错 HMAC 根均 fail closed；original-mysql-regressions.log、hmac-historical.log。 |
| C09 | 重加密后仅保留新密钥仍可正确恢复 | BLOCKED | BLOCKED | 8f3f90c | BLOCKED：AES seed/checkpoint 重包已测；完整 journal HMAC 换根后 new-only 冷恢复缺受支持操作流程与合格证据，不能用 AES 重包替代。 |
| C10 | 旧密钥未完成迁移前不得被错误淘汰 | BLOCKED | BLOCKED | 8f3f90c | BLOCKED：旧 journal 仍依赖旧 HMAC 根；完整密钥库存、全量迁移与核准退休流程未验证；hmac-historical.log 只证明错误根会拒绝。 |
| D01 | 活跃牌局不能删除 | PASS | PASS | 8f3f90c | 真实 active-hand prune 拒绝，行数保留；original-mysql-regressions.log。 |
| D02 | 未归档牌局不能删除 | PASS | PASS | 8f3f90c | 真实未归档手局 prune 拒绝；original-mysql-regressions.log、prune-process.json。 |
| D03 | 不满足 30 天保留期不能删除 | PASS | PASS | 8f3f90c | 真实 29 天拒绝/超过 30 天合格正对照；prune-process.json、original-mysql-regressions.log。 |
| D04 | 损坏或不完整的归档不能删除 | PASS | PASS | 8f3f90c | 真实已完成对局克隆，破坏 DDZ 证据并重算通用 digest，仍拒绝删除；original-mysql-regressions.log。 |
| D05 | 最终版本和日志水位线不一致不能删除 | PASS | PASS | 8f3f90c | 最终版本与 watermark 不符拒绝清理；original-mysql-regressions.log。 |
| D06 | 历史日志数量不完整不能删除 | PASS | PASS | 8f3f90c | 删除/缺少历史 action 导致计数不符，prune 拒绝；original-mysql-regressions.log。 |
| D07 | 不得直接越权读取其他业务模块数据库表 | PASS | PASS | 8f3f90c | 正式模块/数据库依赖边界及源码导入审计通过；mysql-integration-scale.log、go-unit.log。 |
| D08 | 保留完整终局证据和必要审计信息 | FAIL | PASS | 8f3f90c | 正常完成归档保留、EXPIRED真实归档前SIGKILL后补偿一次、ABORTED/NO_LANDLORD重复恢复不重复归档；旧原生第二局在第三局取消/离房后新HTTP查询仍同digest；priority-expiry-real-sql.log、priority-terminal-integrity-and-aborted.log、cancel-repeat-closed-round10.json（前两检查成功）。 |
| D09 | 默认 dry-run 不得修改数据库 | PASS | PASS | 8f3f90c | 实际 maintenance 默认 dry-run，数据库前后无删除；prune-process.json。 |
| D10 | 必须显式 `--execute` 才能执行清理 | PASS | PASS | 8f3f90c | 实际 --execute 才删除符合条件历史，档案保留；prune-process.json。 |
| D11 | 中途崩溃后不能出现违反一致性约束的部分删除 | PASS | PASS | 8f3f90c | 实际删除事务中 SIGKILL，连接断开后原记录完整回滚；prune-process.json。 |
| E01 | 提交承诺前生成高熵随机贡献 | PASS | PASS | 8f3f90c | 真实 SDK 安全随机数两次 32 字节不同，原生 prepare 及缺熵 fail-closed 测试；authentication.json、wechat-full-ui.json、client-foundation.log。 |
| E02 | 恢复凭证包含完整且可验证的信息 | PASS | PASS | 8f3f90c | 真实承诺前导出完整 kit，验证身份/hand/seat/commitment 绑定；game1-commit.log、client-lifetime-matrix.log。 |
| E03 | 用户备份后才允许提交承诺 | BLOCKED | BLOCKED | 8f3f90c | BLOCKED：工具确实先私有外部备份再 commit；真实人类备份确认、设备重启后找回与 UX 未验证；不能以自动化写私有文件替代用户行为。 |
| E04 | 应用退出后可以导入恢复凭证 | PASS | PASS | 8f3f90c | 三轮 commit 后新 Node 进程导入同一 kit，并由原生 Import 实际导入；game1-reveal.log、wechat-full-ui.json；非手机重装恢复资格。 |
| E05 | 恢复凭证与当前登录身份、Hand、Seat、Commitment 必须匹配 | PASS | PASS | 8f3f90c | 错误 account/hand/seat/commitment 绑定拒绝，实际有效 kit 正对照通过；client-lifetime-matrix.log、wechat-full-ui.json。 |
| E06 | 错误凭证必须拒绝 | PASS | PASS | 8f3f90c | 损坏/错身份/错承诺 kit 拒绝；client-foundation.log、client-lifetime-matrix.log。 |
| E07 | 不能产生新的随机贡献冒充旧承诺 | PASS | PASS | 8f3f90c | 未知 commit/reveal 精确原 DTO 重试、不重新生成秘密；leave-fairness-real.json、client-lifetime-matrix.log。 |
| E08 | 不能通过普通小游戏明文持久化自动泄漏随机数 | PASS | PASS | 8f3f90c | SDK 只测试自有存储键，实际 kit 保存在私有外部证据目录；源码与客户端测试无普通 storage 随机贡献自动保存；authentication.json、client-foundation.log。 |
| E09 | 释放对象时敏感数据应尽可能清零 | PASS | PASS | 8f3f90c | 原 F07 异步 prepare/dispose 与敏感 byte-array 尽力清零回归；original-client-repro.log、client-lifetime-matrix.log；不声称 JS immutable string 物理清除。 |
| E10 | 请求丢响应时必须保留原 requestId 精确重试 | PASS | PASS | 8f3f90c | 实际丢失 fairness/leave/终局结果后原 requestId/DTO 精确重试；leave-fairness-real.json、wechat-full-ui.json、terminal-recovery.json。 |
| E11 | 不能因为未知结果直接生成新的操作请求 | PASS | PASS | 8f3f90c | fairness 未知期间 Leave 发包数=0，未知 leave 保留原命令，第二原生点击复用原 DTO；leave-fairness-real.json、wechat-full-ui.json。 |
| F01 | 验证使用三个真实、不同的服务端认证身份 | PASS | PASS | 8f3f90c | 四个正常服务端认证 Go 账号，每局三人身份不同；原生局 wx 账号加两个正常认证 Go peers；full-three-games.log、wechat-full-ui.json。 |
| F02 | 验证三人座位各不相同 | PASS | PASS | 8f3f90c | 每局三个不同 AccountID/不同 seat，51 私牌互斥；full-three-games.log、wechat-full-ui.json。 |
| F03 | 验证每次操作使用真实 WSS | PASS | PASS | 8f3f90c | 游戏业务命令经实际认证 WSS，HTTP 用于文档约定的公平/证据接口；full-three-games.log、active-lifecycle-matrix.json。 |
| F04 | 验证客户端私有手牌访问权限 | PASS | PASS | 8f3f90c | 成员只能读本人私牌，第四账号 403，终局私牌 404 不捏造牌；game1-reveal.log、wechat-full-ui.json。 |
| F05 | 验证游戏完整进行到终局 | PASS | PASS | 8f3f90c | 三轮 Go 客户端完整游戏及实际原生 owner 完整一局，无代打；full-three-games.log、wechat-full-ui.json。 |
| F06 | 验证最终结算及证据 | PASS | PASS | 8f3f90c | 四局正常 archive/分数/digest 验证，原生 v63 胜者1、6/-3/-3；completed-history.json、wechat-full-ui.json。 |
| G01 | 并发牌局状态隔离 | PASS | PASS | 8f3f90c | 100/1000 hands、16 worker 真实数据库隔离、三账号并发；mysql-integration-scale.log + official-scale-1000.log。 |
| G02 | 内存增长与回收 | FAIL | PASS | 8f3f90c | 实际同一runtime120个真实已归档终止局；80/100/120局时finalized/receiptHands均64，active/epochs/seed缓存0；淘汰后/冷恢复240个原回执正确、无效ID无扩张；terminal-cache-120.log。heap含刻意保留的私有fixture，非长期零泄漏声明。 |
| G03 | MySQL 提交延迟 | PASS | PASS | 8f3f90c | 真实 fsync=1 MySQL，80 次提交及 100/1000 并发分位数；mysql-integration-scale.log + official-scale-1000.log，详见第9节。 |
| G04 | P50/P95/P99 | PASS | PASS | 8f3f90c | P50/P95/P99/max 实际样本，含 checkpoint、新动作和重复回执；checkpoint-latency.log、receipt-scale-profile.log。 |
| G05 | 检查点及故障恢复耗时 | PASS | PASS | 8f3f90c | 实际 encrypted checkpoint 100 写入与 100/1000 hand 冷恢复计时；checkpoint-latency.log、mysql-integration-scale.log + official-scale-1000.log；总耗时不冒充单局 P95。 |
| G06 | 高并发下连接池和锁竞争 | PASS | PASS | 8f3f90c | 16 worker/24 DB 连接上限、实际 INNODB_LOCK_WAITS 及 deadlock victim/retry；mysql-integration-scale.log、priority-expiry-real-sql.log、priority-expiry-real-sql.log。 |
| G07 | Race 检查 | PASS | PASS | 8f3f90c | 全仓 Go -race、独立生命周期及 SQL 竞争 -race 实际执行；go-race.log、original-lifecycle.log、priority-expiry-real-sql.log；语义失败单列。 |
| G08 | Goroutine 泄漏 | PASS | PASS | 8f3f90c | 本轮60 warmup+1500真实WSS+cooldown，active0，goroutine26→27、自然GC6→26，heap3.720→7.127MB/RSS29.303→32.236MB；socket-lifetime.json。不强制GC、不声称长期零泄漏。 |
| G09 | 重复请求及失败重试压力 | FAIL | PASS | 8f3f90c | 16worker/1600同ID压力、终局/取消/公平/Leave丢结果精确重试、淘汰后/冷回执及200无效ID均执行；receipt-scale-profile.log、wechat-full-ui.json、terminal-cache-120.log、cancel-repeat-closed-round10.json（原响应deepEqual）。 |
| G10 | 长牌局的候选状态复制开销 | PASS | PASS | 8f3f90c | 实际候选快照复制/长历史开销 profile 与 1000 局 receipt 热路径；original-mysql-regressions.log、receipt-scale-profile.log，未据此推断生产容量。 |
| H01 | TypeScript 类型检查 | PASS | PASS | 8f3f90c | 本轮当前TS类型检查/108客户端测试、正式真实WECHAT引擎strict检查、Web实际cc4声明独立strict；client-foundation.log、linux-native-wechat-build.log、web-independent-strict-source-settings.log。 |
| H02 | Web 实际构建 | PASS | PASS | 8f3f90c | 真实官方 CLI4 WEB_DESKTOP 导出，完整可运行配方还须仓库已有 BigInt 后处理；linux-native-web-build.log、web-existing-bigint-postbuild.log；原始导出启动 FAIL 保留。 |
| H03 | Chromium 场景冒烟测试 | PASS | PASS | 8f3f90c | 上述完整配方产物 Chromium 1280×720/844×390/390×844、字体/帧缓冲/销毁检查通过；web-runtime-operational.log；原始包 Math.pow(BigInt) FAIL 未隐藏。 |
| H04 | 微信小游戏构建 | PASS | PASS | 8f3f90c | 正式build.cjs从去node_modules的隔离副本自动准备依赖，固定真实LinuxCLI/engine WECHAT导出退出0；实际SDK场景/69原生GUI检查、同房两完整局通过；linux-native-wechat-build.log、linux-native-provenance.json、wechat-full-ui.json。无人工补环境。 |
| H05 | 抖音小游戏构建兼容性 | BLOCKED | BLOCKED | 8f3f90c | BLOCKED：当前固定 Linux CLI/extension 无可用 Douyin exporter，未执行当前候选抖音构建；旧 Creator 3.8 历史产物不能替代本轮 Linux 基线。 |
| H06 | 游戏 UI 操作与真实网络请求一致 | FAIL | FAIL | 8f3f90c | 原生两完整局/换局/四种迟到响应/退出清理通过，已人工查看两局终局截图；当前新成员房间lastHand不可访问，controller重连403，Cocos同一refresh分支调用该归档；R10-N01，new-member-history-round10.json及GameBootstrap.ts:343。新成员反例为真实controller+网络，未称原生新成员复现。 |
| H07 | 重连及恢复凭证操作可触达 | PASS | PASS | 8f3f90c | 原生 Import/Retry/Reconnect/Refresh/Evidence 实际按钮可触达，未知 leave 原 DTO 重试通过；wechat-full-ui.json；功能恢复缺陷按 A10/H06 判 FAIL。 |
| H08 | 游戏页面布局、按钮可见性和状态刷新 | FAIL | FAIL | 8f3f90c | 原生两完整局/换局/四种迟到响应/退出清理通过，已人工查看两局终局截图；当前新成员房间lastHand不可访问，controller重连403，Cocos同一refresh分支调用该归档；R10-N01，new-member-history-round10.json及GameBootstrap.ts:343。新成员反例为真实controller+网络，未称原生新成员复现。 |
| AUTO01 | Go 单元测试 | PASS | PASS | 8f3f90c | Go 单元571 PASS/1 SKIP；go-unit.log。 |
| AUTO02 | Go Race 测试 | PASS | PASS | 8f3f90c | Go Race571 PASS/1 SKIP；go-race.log。 |
| AUTO03 | Go Vet | PASS | PASS | 8f3f90c | go vet -p 1 ./... 退出0；go-vet.log。 |
| AUTO04 | Go fmt-check | PASS | PASS | 8f3f90c | make fmt-check 退出0；fmt.log。 |
| AUTO05 | MySQL 集成测试 | PASS | PASS | 8f3f90c | 真实 MySQL 集成610 PASS/1 SKIP；mysql-integration-scale.log。 |
| AUTO06 | 100/1000 牌局专项测试 | PASS | PASS | 8f3f90c | 100/1000 hand 和独立 receipt profile 实际执行退出0；mysql-integration-scale.log + official-scale-1000.log、receipt-scale-profile.log。 |
| AUTO07 | 前端 TypeScript 类型检查 | PASS | PASS | 8f3f90c | 当前 TS 严格检查与真实引擎检查；client-foundation.log、web-independent-strict-source-settings.log。 |
| AUTO08 | 客户端单元测试 | PASS | PASS | 8f3f90c | 本轮31 core+37 doudizhu+40 Cocos=108；client-foundation.log。 |
| AUTO09 | 安全组件测试 | PASS | PASS | 8f3f90c | HPKE9测试、实际 TS↔Go envelope 和 manifest；hpke.log、crypto-interop.log。 |
| AUTO10 | Cocos Creator Smoke | FAIL | PASS | 8f3f90c | 正式build.cjs从去node_modules的隔离副本自动准备依赖，固定真实LinuxCLI/engine WECHAT导出退出0；实际SDK场景/69原生GUI检查、同房两完整局通过；linux-native-wechat-build.log、linux-native-provenance.json、wechat-full-ui.json。无人工补环境。 |
| AUTO11 | Production Compose Runtime | PASS | PASS | 8f3f90c | 本轮原production Dockerfiles/HTTP WS HTTPS WSS hardening隔离运行；production-compose-runtime.log；未修改production或降低断言。 |
| AUTO12 | 数据库架构边界测试 | PASS | PASS | 8f3f90c | 数据库/schema/module import 边界正式测试通过；go-unit.log、mysql-integration-scale.log。 |
| AUDIT01 | P0：数据丢失、敏感信息泄露、身份越权、不可恢复的数据破坏 | PASS | PASS | 8f3f90c | 已进行本轮源码/实际反例/权限/性能审计；记录残留R9-N01/P1及新增R10-N01/P2的前提、位置、步骤、实际/预期、原始证据、影响与建议；PASS为审计交付完成，不表示产品无缺陷。 |
| AUDIT02 | P1：重复出牌、状态分叉、错误结算、恢复失败、死锁及关键链路中断 | PASS | PASS | 8f3f90c | 已进行本轮源码/实际反例/权限/性能审计；记录残留R9-N01/P1及新增R10-N01/P2的前提、位置、步骤、实际/预期、原始证据、影响与建议；PASS为审计交付完成，不表示产品无缺陷。 |
| AUDIT03 | P2：边界条件、资源泄漏、错误处理、用户操作异常、可维护性问题 | PASS | PASS | 8f3f90c | 已进行本轮源码/实际反例/权限/性能审计；记录残留R9-N01/P1及新增R10-N01/P2的前提、位置、步骤、实际/预期、原始证据、影响与建议；PASS为审计交付完成，不表示产品无缺陷。 |
| AUDIT04 | 严重级别 | PASS | PASS | 8f3f90c | 已进行本轮源码/实际反例/权限/性能审计；记录残留R9-N01/P1及新增R10-N01/P2的前提、位置、步骤、实际/预期、原始证据、影响与建议；PASS为审计交付完成，不表示产品无缺陷。 |
| AUDIT05 | 文件及代码位置 | PASS | PASS | 8f3f90c | 已进行本轮源码/实际反例/权限/性能审计；记录残留R9-N01/P1及新增R10-N01/P2的前提、位置、步骤、实际/预期、原始证据、影响与建议；PASS为审计交付完成，不表示产品无缺陷。 |
| AUDIT06 | 触发条件 | PASS | PASS | 8f3f90c | 已进行本轮源码/实际反例/权限/性能审计；记录残留R9-N01/P1及新增R10-N01/P2的前提、位置、步骤、实际/预期、原始证据、影响与建议；PASS为审计交付完成，不表示产品无缺陷。 |
| AUDIT07 | 可复现步骤 | PASS | PASS | 8f3f90c | 已进行本轮源码/实际反例/权限/性能审计；记录残留R9-N01/P1及新增R10-N01/P2的前提、位置、步骤、实际/预期、原始证据、影响与建议；PASS为审计交付完成，不表示产品无缺陷。 |
| AUDIT08 | 实际行为 | PASS | PASS | 8f3f90c | 已进行本轮源码/实际反例/权限/性能审计；记录残留R9-N01/P1及新增R10-N01/P2的前提、位置、步骤、实际/预期、原始证据、影响与建议；PASS为审计交付完成，不表示产品无缺陷。 |
| AUDIT09 | 预期行为 | PASS | PASS | 8f3f90c | 已进行本轮源码/实际反例/权限/性能审计；记录残留R9-N01/P1及新增R10-N01/P2的前提、位置、步骤、实际/预期、原始证据、影响与建议；PASS为审计交付完成，不表示产品无缺陷。 |
| AUDIT10 | 建议修复方向 | PASS | PASS | 8f3f90c | 已进行本轮源码/实际反例/权限/性能审计；记录残留R9-N01/P1及新增R10-N01/P2的前提、位置、步骤、实际/预期、原始证据、影响与建议；PASS为审计交付完成，不表示产品无缺陷。 |
| GOAL1 | Real platform-code login exchange / session validation, token lifetime and secure private credential management; no fake identity, no tokens in URLs; include local authorized dev login path only if correctly access-controlled. | PASS | PASS | 8f3f90c | 本轮原生wx登录、SDK独立官方换证/稳定AccountID/session恢复吊销/WSS，四正常Go认证账号隔离；wechat-full-ui.json、authentication.json、wx-sdk-authentication-corrected.log。 |
| GOAL2 | Account-scoped rooms: create/join/leave/ready, seats, start hand, authoritative room refresh/reconnect and timeout/duplicate reconciliation; full Cocos interactive flow. | FAIL | FAIL | 8f3f90c | 原生房间全流程及取消/关闭/退出未知结果通过；已完成房间新成员被不可访问lastHand引用阻断重连；R10-N01，new-member-history-round10.json。 |
| GOAL3 | Fair lifecycle: server-bound commit/reveal key/AAD acquisition, secure entropy + HPKE, trusted beacon, deal, private hand and evidence verification, fail closed on missing security context. | PASS | PASS | 8f3f90c | 真实安全熵/manifest/AAD/HPKE/drand、私牌隔离、证据校验与三轮恢复凭证闭环；full-three-games.log、wechat-full-ui.json；人类备份 UX 单列 E03。 |
| GOAL4 | Playable game: bidding, landlord determination, legal play/pass delegated to authoritative Go rules, turn indicators, cards/selection, clear error and version-retry handling, termination, score/evidence/result UI. | FAIL | FAIL | 8f3f90c | 两原生局均真实可玩并正确终态，旧终态和迟到响应不污染；内部拒绝Expiry错误fence仍能阻断后续合法Bid，完整生命周期不合格；R9-N01，priority-expiry-real-sql.log。 |
| GOAL5 | B-level MySQL recovery: append each accepted live bid/play/pass state transition and dedup + owner epoch/fence atomically **before acknowledgement**; bounded encrypted checkpoint and ordered replay restoring game state, ambiguity handling and owner fencing; address seed and other secret continuity; no plaintext player hands in routine logs. | FAIL | FAIL | 8f3f90c | 正常log-before-ACK、终止冷回执/归档补偿、日志/密钥/身份/篡改/epoch故障回归通过；业务拒绝终止仍提交fence违约，R9-N01，priority-expiry-real-sql.log。 |
| GOAL6 | Real 3-player repeatable E2E, reconnect and crash-injection: pre-/post-COMMIT kill, response loss, duplicate/out-of-order commands, competing owners, wrong epoch and replay mismatch; preserve every acknowledged action. | FAIL | PASS | 8f3f90c | 三Go身份完整多局/两原生同房完整局与真实COMMIT前后/终局丢响应/SIGKILL/接管/精确回执组合均执行并通过；full-three-games.log、wechat-full-ui.json、active-process-fault.json。内部异常Expiry契约按GOAL5/B13判FAIL，不将此E2E正例等同完整产品验收。 |
| GOAL7 | Performance and stability: bounded queues/workers/room memory, P50/P95/P99 measurements and transaction fsync budget on stated hardware, no unknown target claims. | FAIL | PASS | 8f3f90c | 本轮100/1000局、120归档局热缓存64上限/淘汰/按需冷历史、1500WSS、P50/P95/P99/max/串行吞吐与锁竞争实测；terminal-cache-120.log、terminal-history-throughput.log、mysql-integration-scale.log + official-scale-1000.log、socket-lifetime.json；非生产容量/SLA或长期soak认证。 |
| GOAL8 | WeChat Creator package, actual developer tool, approved HTTPS/WSS domains and real handset evidence where access is available. Record platform-owned AppID/domain/device blockers honestly. Preserve Web and Douyin build compatibility. | BLOCKED | BLOCKED | 8f3f90c | BLOCKED：真实 SDK/login/AppID/Secret 已具备；核准生产域名/TLS/平台白名单、实际手机和抖音 SDK/设备及 Linux exporter 资格未具备，不得说微信未登录。 |

统计：**99 PASS + 10 FAIL + 5 BLOCKED = 114**；较Round9，10个FAIL获得本轮有效正证据而转PASS，10个关联条目因残留R9-N01或新增R10-N01仍FAIL，5个BLOCKED保留。没有按“修复一个P1就自动关联全PASS”，也没有把10矩阵FAIL计成10个独立缺陷。

## 11. 命令、原始退出与私有fixture修正

入口及环境：Go1.25.8/GOMAXPROCS2，仓库server运行 `go test -json -count=1 -p 1 -parallel 1 ./...`、全仓 `-race`、`go vet -p 1 ./...`及 `APP_API_INTEGRATION=1 ... -tags=integration ./...`；根目录运行 `make fmt-check`、`make api-contract-check`、`bash scripts/verify-client-foundation.sh`、secure-envelope测试/真实TS↔Go向量与manifest互操作。独立SQL使用 `$E/overlay-expiry.json`新增TestCodex，120局/吞吐分别用overlay-cache-round10/overlay-throughput-round10。

unit与Race各571PASS/1SKIP，MySQL610PASS/1SKIP，另显式开启DOUDIZHU_SCALE_PROFILE完成官方1000局定向用例（parent/subtest计数，不能跨组相加宣称唯一用例总数）。默认skip的互操作向量另以真实向量单独执行。客户端31+37+40=108、HPKE9、原生命周期23、HMAC35；原MySQL外置组38PASS+私有provider缺文件1FAIL，修正后仅补跑provider父/子4PASS。全部实际执行记录如下，精确argv/cwd在runs.jsonl。

| 命令label（同名log/精确argv见runs） | UTC开始 | 秒 | 退出码 |
|---|---|---:|---:|
| `setup-fixture` | 00:12:17.222 | 19.882 | 0 |
| `priority-expiry-real-sql` | 00:12:47.077 | 15.946 | 1 |
| `client-foundation` | 00:13:23.034 | 35.952 | 0 |
| `priority-client-hand-switch` | 00:14:08.892 | 0.328 | 0 |
| `go-unit` | 00:14:09.292 | 65.332 | 0 |
| `go-race` | 00:15:14.723 | 134.611 | 0 |
| `mysql-integration-scale` | 00:17:29.418 | 62.412 | 0 |
| `go-vet` | 00:18:47.599 | 7.493 | 0 |
| `app-build` | 00:18:55.169 | 13.652 | 0 |
| `maintenance-build` | 00:19:08.897 | 2.753 | 0 |
| `fmt` | 00:19:11.720 | 0.367 | 0 |
| `api-contract` | 00:19:12.157 | 0.333 | 0 |
| `hpke` | 00:19:12.571 | 3.628 | 0 |
| `interop-ts-vector` | 00:19:16.410 | 3.028 | 0 |
| `interop-go-open` | 00:19:19.511 | 0.869 | 0 |
| `interop-go-manifest` | 00:19:20.447 | 0.254 | 0 |
| `interop-ts-manifest` | 00:19:20.774 | 2.945 | 0 |
| `crypto-interop` | 00:19:16.278 | 7.467 | 0 |
| `original-lifecycle` | 00:19:23.824 | 9.191 | 0 |
| `hmac-historical` | 00:19:33.101 | 6.986 | 0 |
| `code-gates` | 00:18:47.482 | 52.634 | 0 |
| `terminal-cache-120` | 00:19:40.190 | 26.398 | 0 |
| `linux-toolchain-pins` | 00:20:06.804 | 0.133 | 0 |
| `linux-platform-tests` | 00:20:07.023 | 1.455 | 0 |
| `linux-native-wechat-build` | 00:20:08.557 | 223.346 | 0 |
| `native-formal-build` | 00:20:06.664 | 225.335 | 0 |
| `priority-terminal-integrity-and-aborted` | 00:24:58.090 | 14.990 | 0 |
| `priority-client-event-isolation` | 00:25:13.174 | 0.317 | 0 |
| `product-start` | 00:25:13.592 | 1.580 | 0 |
| `game1-commit` | 00:25:15.429 | 8.202 | 0 |
| `game1-reveal` | 00:25:23.704 | 61.893 | 0 |
| `original-N01` | 00:26:25.684 | 1.418 | 0 |
| `game1-complete` | 00:26:27.178 | 3.218 | 0 |
| `game2-commit` | 00:26:30.487 | 4.136 | 0 |
| `game2-reveal` | 00:26:34.697 | 64.370 | 0 |
| `active-lifecycle-matrix` | 00:27:39.130 | 13.492 | 0 |
| `game2-terminal-loss` | 00:27:52.703 | 2.986 | 1 |
| `terminal-recovery` | 00:27:55.780 | 1.679 | 0 |
| `completed-history` | 00:27:57.529 | 3.845 | 0 |
| `restart-binding-negatives` | 00:28:01.455 | 21.451 | 0 |
| `all-types-sequence-restart` | 00:28:22.984 | 18.593 | 0 |
| `game3-commit` | 00:28:41.668 | 4.135 | 0 |
| `game3-reveal` | 00:28:45.875 | 64.337 | 0 |
| `active-process-fault` | 00:29:50.306 | 8.089 | 0 |
| `settling-failed-complete-game` | 00:29:58.774 | 4.625 | 1 |
| `settling-without-restart` | 00:29:58.480 | 17.895 | 0 |
| `full-three-games` | 00:25:15.269 | 301.139 | 0 |
| `original-client-repro` | 00:30:16.629 | 0.554 | 0 |
| `client-lifetime-matrix` | 00:30:17.252 | 2.892 | 0 |
| `original-mysql-regressions` | 00:30:20.238 | 9.938 | 1 |
| `checkpoint-latency` | 00:30:30.277 | 3.289 | 0 |
| `receipt-scale-profile` | 00:30:33.633 | 82.347 | 0 |
| `expiry-sql-connection-loss` | 00:31:56.099 | 5.349 | 0 |
| `prune-cli-atomicity` | 00:32:01.534 | 7.898 | 0 |
| `ordinary-replay` | 00:32:09.502 | 9.376 | 0 |
| `rejected-replay-contract` | 00:32:18.943 | 0.707 | 0 |
| `room-after-send-loss` | 00:32:19.721 | 15.686 | 0 |
| `room-before-send-loss` | 00:32:35.488 | 0.685 | 0 |
| `deploy-wiring` | 00:32:36.242 | 0.696 | 0 |
| `compose-switch` | 00:32:37.022 | 0.455 | 0 |
| `default-yaml-start` | 00:32:37.537 | 1.315 | 0 |
| `generation-check` | 00:32:38.924 | 2.043 | 0 |
| `socket-lifetime` | 00:32:41.063 | 31.988 | 0 |
| `post-business-gates` | 00:30:16.494 | 176.595 | 1 |
| `leave-fairness-real` | 00:33:13.179 | 1.188 | 0 |
| `new-member-history` | 00:33:14.432 | 0.847 | 1 |
| `legacy-clone` | 00:33:15.373 | 7.759 | 0 |
| `legacy-upgrade` | 00:33:23.206 | 0.913 | 0 |
| `provider-private-fixture-corrected` | 00:34:50.916 | 3.574 | 0 |
| `terminal-history-throughput` | 00:34:54.562 | 6.239 | 0 |
| `refresh-real-sessions` | 00:35:00.875 | 3.207 | 0 |
| `wechat-launch` | 00:35:04.147 | 0.212 | 0 |
| `wechat-open-current-artifact` | 00:35:04.433 | 32.240 | 255 |
| `linux4-native-two-games` | 00:35:36.750 | 647.643 | 0 |
| `wx-sdk-authentication` | 00:46:24.479 | 6.227 | 1 |
| `cancel-repeat-closed` | 00:47:59.569 | 4.167 | 1 |
| `cancel-repeat-closed-corrected` | 00:49:29.699 | 3.514 | 1 |
| `wechat-auth-retry-launch` | 00:49:33.279 | 0.173 | 0 |
| `wechat-auth-retry-open` | 00:49:40.540 | 9.111 | 0 |
| `wx-sdk-authentication-corrected` | 00:49:49.970 | 97.011 | 0 |
| `linux-native-web-build` | 00:51:27.581 | 158.318 | 0 |
| `web-runtime-raw` | 00:54:06.222 | 93.467 | 1 |
| `web-existing-bigint-postbuild` | 00:55:39.833 | 0.485 | 0 |
| `web-runtime-operational` | 00:55:40.406 | 8.219 | 0 |
| `web-independent-strict-source-settings` | 00:55:48.707 | 4.084 | 0 |
| `production-compose-runtime` | 00:55:53.066 | 330.883 | 0 |
| `web-and-production-runtime` | 00:51:27.452 | 596.558 | 0 |
| `closed-start-focused` | 01:03:51.999 | 1.377 | 0 |
| `rejected-expiry-state-audit` | 01:03:53.176 | 10.093 | 1 |
| `official-scale-1000` | 01:06:42.422 | 29.132 | 0 |

以下失败保留，区分产品与fixture，未修改正式测试/断言绕过问题：

1. R9-N01真实SQL FAIL、R10-N01新成员真实403 FAIL均为产品问题，未修复/隐藏。
2. 原MySQL外置组缺私有config-provider.yaml；用本轮新有效配置补齐后focused provider检查通过，未重复跑完整组或伪造原命令退出0。
3. 微信CLI首次从repo cwd运行，spawn `./wechat-devtools` ENOENT/timeout退出255；同一GUI进程保留，正确tool-bin cwd打开成功。该次记录器在非git cwd抓SHA又报错，原日志/`operator-cli-correction.json`保留；后续记录器明确git -C，正确打开及独立SDK执行均有完整记录。
4. SDK补充漏复制纯测试orchestration `auth-expression.js`，首次退出1；补齐不含旧产品二进制的同一探针后，使用**当前dist新生成的组件bundle**只重跑SDK补充成功。未重跑已通过两局GUI。
5. 取消补充探针最初错误要求live replay结果duplicate=true；源码契约为完整原响应（可保持duplicate=false），修正为更精确的完整response deepEqual原值。后续CLOSED start探针又误用hand.start，client正确在发送前拒绝；其前两取消/历史检查与join/ready拒绝已成功，最后仅focused使用正确room.hand.start和真实原owner验证CLOSED拒绝。两次脚本原退出1均保留，不改标整体通过。
6. 120局fixture编译后才加入的tamper扩展未被该次执行采用；已恢复并索引实际原版本，未执行扩展另存明示；三项真实篡改验证由单独focused SQL检查完成，不把代码存在当执行证据。
7. 默认完整集成没有开启DOUDIZHU_SCALE_PROFILE，只有官方100局；独立receipt profile虽有1000局，仍未冒充官方用例。收尾发现后仅临时启动自有测试MySQL，显式开启开关定向执行官方hands-1000并停止，完整结果另列official-scale-1000.log，没有重跑整套。
8. game2-terminal-loss及settling-failed-complete-game的退出1是受控丢结果/故障注入，父级恢复/补偿断言最后成功；原始Web启动FAIL也保留，仅已有后处理完整配方PASS。

## 12. BLOCKED、未覆盖与正式发布边界

- C09/C10：完整journal HMAC换根/new-only恢复、全量key库存/迁移/核准退休缺合格操作流程/环境证据；AES seed/checkpoint重包与错根拒绝不能替代。
- E03：自动化实际先私有外部备份再commit；真实人类备份确认、手机重启/重装找回和UX没有验证。
- H05：固定Linux4工具链没有当前可执行的Douyin exporter；旧3.8报告不构成本轮构建。Douyin真实SDK/熵/设备亦未资格化。
- GOAL8：微信已登录、AppID/AppSecret和官方换证实际通过；缺少的是核准生产域名/TLS/平台白名单、实际手机和适用发布条件。不能再写成“微信未登录”或缺换证凭据。
- 未完整重启微信客户端后穷举终局恢复、未验证三台手机/移动网络/真实前后台投递/分享/长期soak与生产SLA；部分版本/成员失效/未知手局排列只有正式组件/端口测试，未称全部原生GUI通过。
- 真实SDK/native touch不同于真机；beta engine/2D实验exporter不资格化3D、remote bundle、分包、审核/上传或正式发布。Web调试包、SDK缩放小字及工具性能提示均不作发布合格证据。

## 13. 结论与交接

**内部代码/集成产品FAIL，外部资格BLOCKED，正式发布NO，封板NO。** 8个Round9原问题只关闭7个；R9-N01/P1仍在，另有R10-N01/P2。即使其他可执行大门禁、正式Linux导出、实际SDK两局和CI通过，也不满足内部PASS条件。

交回ChatGPT修复以上两项及相应真实回归，完成全面自测并push新的固定main候选后再交Codex复验受影响项和完整门禁。Codex未改产品、正式测试、构建脚本、协议/schema或CI。Goal0045/0046继续 `in_progress`，保留历史证据；本轮独立验收任务交付不等于产品Goal完成。

## 14. 保护、清理与交付复查

本轮实际测试最后收于2026-11-06 01:07:11 UTC；自有backend/GUI/TLS/MySQL/Redis及production runtime均停止，自有测试端口无监听，数据库卷/证据和工具缓存保留。曾仅为补跑官方1000局短暂重启自有MySQL，已再次停止并核对；没有全局Docker清理、swap、生产DB/Kubernetes/既有凭据变更。`cleanup.json`记录实际动作、剩余服务和资源。

私有根目录0700，实际文件0600/必要执行文件0700，符号链接不递归处理外部工具目录。已知AppSecret、签名/存储/HMAC密钥、JWT/密码/kit/TLS私钥的精确值扫描覆盖拟提交文档和微信/Web产物；`secret-scan.json`保留范围与结果，不输出原值、不冒称覆盖所有编码/运行内存。私有截图不提交。`evidence-sha256.json`、`matrix-round10.json`、`closeout-summary.json`索引当前证据；产物公开key/AppID不与秘密混淆。

Web原始manifest `2885fb73fd90373c8fa56df284420d1082e6b399fbe9d346f0b277850fa2b680`，既有后处理运行包manifest `c3b80fb3a7c72d6d8de9b9506eed5e4ade798d2f903750b6abe0cbacb1a38771`，调试包23545246B；SDK操作后的微信包变化和编译app/maintenance二进制hash另存`artifact-binding-final.json`。调试HUD已在portrait截图人工查看，不称可发布资源包。清理后可用内存约4459MiB、swap0、根盘48%/数据盘70%（约21/12GiB余量）。


报告复查覆盖：114项ID/要求及状态计数、本轮源码/argv/实际退出、已知/新增问题前提和优先级、原生与controller/SDK与真机、正式构建与Web既有后处理、fixture错误与产品FAIL的区别。此为报告自审，未称额外第三方审查。最终diff仅本报告/current获准状态段，原文及历史报告保留；文档commit与固定产品SHA分开，main-only正常提交推送，无强推。
