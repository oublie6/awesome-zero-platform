# Goal 0051 — 房间恢复一致性与客户端可用性独立验收

执行日期：2026-11-08 UTC。执行者：Codex。独立增量验收；未修改产品、正式测试、Schema、部署配置或历史报告。

## 1. 结论

**最终 FAIL。** 六项指定修复——G50-N01、G50-N02以及额外A/B/C/D——均通过本轮真实服务器复验。新增 **G51-N01/P3：换账号后旧房间对账失败回调覆盖新账号成功提示**，未满足任务明确要求的旧回调UI隔离条件，因此不判整个Goal0051 PASS。此缺陷仅实证为提示污染，没有新账号业务状态或服务端记录损坏，也不是新的房间版本回退。

协议49/49、shell85/85、typecheck通过，精确候选的三套CI总体成功。CI和六项修复通过不替代剩余必需条件。ChatGPT负责修复；Goal0045/0046继续 `in_progress`，E03不完成，不封版、不发布。

## 2. 固定版本与完整变更范围

受测产品 **`e57f95af6baea6019890cae967c3e5a3eb937150`**。干净main执行 `git pull --ff-only`，从上轮报告提交 `82499735ceeb7b93626e3a914bacee6a377c80d9` 快进至候选；报告前 `git ls-remote origin refs/heads/main` 再次核对相同。文档提交与受测产品SHA分开。

已审阅AGENTS、原则、current、[Goal0049](0049-independent-acceptance.md)、[Goal0050](0050-independent-acceptance.md)和候选完整差异：7文件，241行新增/27行删除。

| 文件 | 变更及独立审查重点 |
|---|---|
| GameBootstrap.ts | 可见RetryUnknownRoom菜单、普通reconnect先精确对账、refresh统一采用校验及选择代次、最终恢复不直接补写版本；错误回调隔离另见G51-N01 |
| GameOnlineController.ts | member join命令代次围栏、公开adoptAuthorizedRoomRefresh、retryUnknownRoom及连接就绪检查、reconcile先结算原请求 |
| GameClientShell.ts | 暴露commandTransportReady，只读查询实际连接状态 |
| doudizhu-client/src/room.ts | 每房间knownVersions、命令/重试完成finally推进代次、join序号在本地验证通过后分配、低版本receipt保护 |
| assets.test.mjs、online.test.mjs、room.test.mjs | 对应菜单、生命周期、序号、版本及恢复测试；独立检查断言，正式测试未被验收方改写 |

server tree仍为 `06209f0fc5a8f9bb8a50682c69cc6f195f354e18`；GameFairnessActions和GameHandActions未变。没有新增平台/Domain依赖或客户端权威牌局备份。

## 3. 精确候选CI与本地命令

独立读取固定SHA的三个run及全部job，均已completed；未重复触发工作流。

| 工作流 | 最终结果 |
|---|---|
| [主CI 42429595761](https://github.com/oublie6/awesome-zero-platform/actions/runs/42429595761) | SUCCESS，secure_envelope/admin_web/unit/integration/runtime/report 6/6 |
| [Fair Doudizhu 42429595608](https://github.com/oublie6/awesome-zero-platform/actions/runs/42429595608) | SUCCESS，application/mysql_integration/client_foundation/manifest_client 4/4 |
| [Cocos Linux 42429595655](https://github.com/oublie6/awesome-zero-platform/actions/runs/42429595655) | 总体SUCCESS，linux_source SUCCESS，**linux_native_export SKIPPED** |

本地新执行并通过：

```sh
npm --prefix clients/packages/doudizhu-client test       # 49/49
npm --prefix clients/fair-doudizhu-cocos test             # 85/85
npm --prefix clients/fair-doudizhu-cocos run typecheck
```

未将SKIPPED或TS节点夹具记为本候选原生导出、实际SDK或真机通过。

## 4. 环境与证据边界

Linux、Node22.23.4、真实Go账号认证/会话、MySQL5.7.44、Redis8.8.0、HTTPS/WSS、真实CSPRNG及Quicknet。复用同server tree的API二进制，SHA256为 `e3901e2e4b776da28eb8608affac804598233ebeafdbd975755c1af1dfefc1d1`，本轮重新核对摘要和树身份，没有重复编译后端。只延长复制夹具的测试registry有效期；保护根、生产配置不变。构建和运行验证顺序执行。

实际候选TS的Bootstrap菜单及方法经转译执行，布局/基础Cocos节点由私有夹具替代；Controller、协议、HTTP/WSS、真实命令接受/拒绝、MySQL持久化和公平贡献均走产品实现。延迟的是实际读结果、实际熵源交付或实际ACK，未伪造版本/随机/accepted结果。账号切换采用真实已认证controller替换及产品的dispose/代次清理顺序，不冒称执行微信SDK换号或服务器token撤销。

私有证据目录 `E=/root/.config/awesome-zero-platform/acceptance-0051-e57f95a`：runs.jsonl记录argv/cwd/SHA/UTC/退出码；ci.json、provenance.json、reused-evidence.json、场景JSON/日志、shutdown.json、completion-audit.json及sha256.json保存结果。私有账号、密钥、贡献和手牌不提交。

## 5. G50-N01及G50-N02原始反例

**G50-N01(a) PASS：** member join的最终合法v1被扣留，同controller真实ready接受并ACK至v2，释放旧读后join不能采用或报告过期成功，Controller保持v2。额外命令在途ACK、明确拒绝、精确重试交错也通过。证据：generation.json。

**G50-N01(b) PASS：** ready先开始，延迟真实CSPRNG返回；随后恢复两次真实读取v1，扣留最终结果；放行ready实际持久化并ACK至v2，再放行旧读，恢复失效，不覆盖v2。证据：inflight-version.json。源码确认perform/retry开始及finally完成均推进命令代次；普通member join和reconnect捕获代次，普通refresh使用同一采用规则。

**G50-N02 PASS：** 实际Bootstrap建房已接受但ACK丢失，roomID为空而pendingRoomId保留。重新渲染后的RetryUnknownRoom菜单可见、可达，实际调用该菜单action，普通Scene reconnect发送完全相同原请求并恢复A。RequestID、payload、clientSeq、expectedVersion逐字段相同，没有新create。之后正常ready/unready/Scene leave成功。证据：scene-room-retry.json。

另四类room.join/ready/hand.start/leave均真实接受后丢ACK，通过Scene reconnect精确对账，再恢复本人B；原请求wire相同，MySQL原整行摘要不变。普通同/跨房间读不能清除UNKNOWN，切换先受保护；leave后不再恢复已退出A。证据：scene-unknown.json。这里读取原协调器仅用于断言，恢复操作不依赖用户访问底层retry方法。

## 6. 额外四项修复

| 项目 | 独立验证结果 |
|---|---|
| A：refresh统一前置校验 | PASS。Room UNKNOWN、Fairness UNKNOWN/执行中、Hand UNKNOWN时真实刷新不能替换状态/原请求；room.hand.start实际已接受但ACK扣留时不抢先采用HTTP中的新H；旧读跨命令ACK被拒绝，最新刷新可继续 |
| B：被拒join污染序号 | PASS。A已有身份时冲突join(B,seq800)及非法join(A,v0,seq900)本地拒绝，后续真实A.ready仍seq2；之后B首个ready仍seq2，未跨房间污染。真正提交ACK丢失的序号在五类原请求精确重试中保持不变 |
| C：A→B→A水位 | PASS。真实A/v3读先保留，确认A/v4后切B；向公开采用入口交付先前真实v3、使用当前命令代次隔离验证水位，拒绝且B不变；另一个真实客户端推进A/v5后Scene正常恢复A。没有伪造服务器在v5时返回v3。明确拒绝不滞留UNKNOWN；leave确认A/v7后旧v3仍不可采用，B水位独立 |
| D：Scene旧刷新与新选择 | PASS。等待房间A旧刷新与B开始恢复交错不写旧提示；再用真实active H验证旧等待房间刷新不能清除新H、Fairness实例或authoritativeHand，最新刷新仍成功 |

证据：extra-four.json、hand-matrix.json、cold-active-refresh.json。命令已获明确拒绝时按协议消耗其合法已提交序号，不能与本地非法join不应分配序号混淆。

## 7. 交叉矩阵及正常业务

| 真实状态/时序 | 结果与证据 |
|---|---|
| 无未决，等待/active恢复 | PASS；cold-waiting、cold-active-refresh |
| Room UNKNOWN，其他无未决 | PASS；五类房间命令Scene对账及Room UNKNOWN普通刷新 |
| Fairness UNKNOWN，其他无未决 | PASS；贡献真实接受/丢ACK，刷新及跨房间采用被拒，Scene公平重试后可继续 |
| Hand UNKNOWN，其他无未决 | PASS；实际Bid(3)接受/丢ACK，Scene原Hand请求重试后恢复A/B，live持久化整行摘要不变 |
| Hand UNKNOWN + Fairness UNKNOWN | PASS；两原请求保留，没有部分采用 |
| Room UNKNOWN + Hand UNKNOWN | PASS；Hand未知时另一个room.ready实际得到持久拒绝但丢响应，两UNKNOWN均保留；原Room拒绝receipt结算后Hand仍受保护 |
| Room命令开始/ACK/拒绝/重试与旧读 | PASS；generation、inflight-version及extra-four，代次不复位 |
| start已接受、ACK等待 | PASS；Scene刷新不提前切H，ACK完成可正常继续 |
| Fairness执行中/Hand执行中 | PASS；真实贡献/Bid已提交但扣留返回，禁止跨房间替换；Fairness执行中的普通刷新也拒绝 |
| 仍UNKNOWN/明确rejected/accepted | PASS；完整原请求保留或按原receipt结算，随后正常ready/unready；receipt摘要不变 |
| WSS断开、未就绪时双击核对 | PASS；重建真实连接，只发1次原wire请求，最终可继续发送正常命令，无永久卡死 |
| 旧账号Room对账失败迟到 | **FAIL，G51-N01**；新账号业务未变但状态提示被旧错误覆盖 |
| Scene销毁后旧对账结果 | PASS；status/online已清空，无旧UI写入 |

正常对照实跑：账号登录/服务器发现、创建、非成员加入、成员重新进入、准备/取消准备、三人公平开局、三份真实托管贡献、断线恢复、三类UNKNOWN精确对账、退出/切换、全新controller仅持账号认证冷恢复等待房间及active H。没有本地牌局备份。active冷恢复沿用产品规则：唯一active房间优先自动恢复，多房间需选择的历史用例单独验证。

本轮只新建一局推进至真实Bid(3)，保留H=`g51-hand-38c04c782d2a0309`、A=`room-cmd-213a6c2c8ef46f63e1df0bc1d47436a7`于私有验收卷。没有为了清理而重打完整牌局；后续夹具必须考虑它与Goal0050留存局。其余自有等待/已取消测试房间通过正常授权接口关闭。

## 8. 相关历史回归与执行纠偏

| 历史项目 | 本轮判断 |
|---|---|
| R18-N01/N02 | PASS：等待/active账号冷恢复新执行；mode-isolation现代模式菜单、四种历史core操作拒绝、旧H1回调隔离及新贡献通过 |
| R19-N01/N02 | PASS（受影响子集）：多房间明确选择、重复恢复合并、恢复中CLOSED、采用后CLOSED撤销、旧locator重新检查新执行；未重复创建33个房间或再造两局active，容量/隐私负例沿用历史证据 |
| G47-N01 | PASS：旧member join迟到不能覆盖后选B，零新room命令 |
| G48-N02/N03 | PASS（原有场景）：成员/公平转换保护及六个登出/销毁/换账号交错；新增Room对账异常回调另FAIL，不扩大旧PASS含义 |
| G49-N01/N02 | PASS（受影响交叉）：同房间读不能清Room UNKNOWN、旧读与命令ACK/精确重试交错、普通刷新UNKNOWN保护新执行；G49-N01原始“旧读开始后新命令丢ACK”精确时序沿用Goal0050通过证据。Fairness UNKNOWN跨房间原子拒绝及精确结算后切换新执行 |
| G50-N01/N02 | PASS：两个真实版本反例及实际Scene菜单原始反例，见第5节 |

诚实保留非产品失败：旧mode夹具在Fairness执行中先调用peer.reconcileRoom，现有统一前置校验拒绝；改用另一个真实同账号controller推进服务器，旧客户端刷新仍必须拒绝，操作结束后再采用。该次残留房间导致后续6个历史用例在房间集合前提断言即失败；清理自有夹具后才逐项执行并通过，没有把前提失败算产品缺陷。最初跨VM数组原型比较错误已改为值比较，仅补跑未完成换账号子例。hand-matrix的9个行为断言通过后，冷恢复夹具错误要求roomID为空，而产品正确自动选择唯一active A；保留原日志，独立cold-active-refresh以空状态新controller确认自动恢复及active隔离，不重跑已通过矩阵。未更改正式测试或放宽产品预期。

## 9. 新发现 G51-N01 / P3 — 旧账号对账异常覆盖新账号提示

位置：`GameBootstrap.ts:363–386`。`reconnect()`在await之后的成功分支检查online身份/content，但外层catch直接notice；内层明确拒绝notice也在身份检查前。已真实验证的是外层catch路径。

最小复现：

1. 真实账号X创建A，ready接受并持久化，丢ACK，保留原UNKNOWN。
2. 实际Scene reconnect精确重试原RequestID，真实Go返回已接受重复receipt，扣留交付。
3. 按login中的dispose/代次/清理顺序替换为另一个真实已认证账号Y，完成Y的服务器房间恢复，提示为“已通过服务器恢复本人房间和牌局”。本探针不模拟wx.login。
4. 放行旧X请求并模拟该次返回仍丢失，旧reconnect catch将Y提示改为“重连失败：old account duplicate ACK lost”。

A=`room-cmd-9d20d9a6a3e683959ff053ed3477dd54`，原RequestID=`cmd-eab63a3bd8b6877f8756c194c0da980f`。证据retry-lifecycle.json记录前后提示、新账号Controller状态逐值不变、MySQL原请求整行摘要不变；没有新命令或新的业务影响。配套销毁场景通过。

预期旧X结果完全不写Y界面；实际仅错误提示越过账号生命周期。影响等级P3：可被用户后续正常操作覆盖，不是P1/P2数据损坏、权限泄漏或永久阻塞；仍违反本任务明确必需条件，故本轮整体FAIL。最小建议：在整个reconnect入口捕获online及Scene请求/选择生命周期，所有await之后及catch/notice之前统一验证；不要靠清UNKNOWN、生成新ID或禁止正常账号切换解决。增加原accepted与原rejected的迟到异常、换账号和销毁对照正式回归。其他异步入口同类风险仅建议审查，不以源码猜测另开已证实缺陷。

## 10. 新执行、复用及未验证项目

新执行项目如上；所有真实场景的原命令都经授权产品调用产生。readonly MySQL oracle只确认receipt和断言，不向产品恢复入口传手牌或改库。

未变化的HPKE/AEAD、drand验证、洗牌/最终证明、完整游戏及服务端故障恢复，沿用Goal0047/0049/0050对应报告与私有证据；本轮核对API二进制、server tree、相关未变源码并复核13个历史文件摘要。新局使用真实贡献/信标不等于重做完整密码学验收。不声称新执行本地全Go/race、全游戏、SIGKILL、规模或原生SDK测试；这些在本次客户端增量之外。

本候选原生导出CI为SKIPPED，未执行新的Linux导出/SDK/真机/生产平台与正式发布验证。外部资格沿用既有未解除限制，不能据此宣称发布。此轮没有因环境问题跳过六项指定修复的真实服务器验证，也没有将未执行项目记为PASS。

交付范围仅本报告和current允许状态段；保留所有历史FAIL及原报告。验收服务停止、私有证据保留；当前验收任务报告交付完成不表示产品Goal完成。ChatGPT应修复G51-N01后提交新SHA，再独立复验受影响回调和正常恢复门槛。
