# Goal 0046 — Round 14：E03 恢复凭证与交付范围独立验收

2026-11-06 UTC，Codex。固定产品 **`dc99ad558decbb7b91a4f9203a5ff722a4627fa6`**。

| 独立判定 | 结论 |
|---|---|
| A：E03 实现与安全 | **FAIL**：正常备份门槛和恢复路径通过，房间/牌局切换存在两项 P2 生命周期问题 |
| B：当前 Goal0046 内部完成资格 | **FAIL**：需修复 E03，不能因 C09/C10 延期直接封版 |
| C：外部平台及发布资格 | **BLOCKED**：原生键盘/人工备份体验、抖音、官方工具/域名/手机及发布资格未全部完成 |

Goal0045/0046保持 `in_progress`。本轮新增发现 **R14-N01/P2、R14-N02/P2**，不修改产品。当前增量状态 **106 PASS / 4 FAIL / 2 BLOCKED / 2 DEFERRED = 114**；四个FAIL是两个缺陷对关联要求的映射，不是四个独立缺陷。

## 1. 版本、CI与范围

从 clean main 的 Round13 报告提交 `ca8344362ed53fcce21bbe741ecd915fffc15d04` 执行 `git pull --ff-only origin main`，同步到上述候选。重新查询 [CI 42193277232](https://github.com/oublie6/awesome-zero-platform/actions/runs/42193277232) **6/6 SUCCESS**、[Fair Doudizhu 42193277212](https://github.com/oublie6/awesome-zero-platform/actions/runs/42193277212) **4/4 SUCCESS**；两者均为本候选，见 `ci.json`，未复跑CI或混用报告提交CI。

按需核查 AGENTS、相关原则、current、0045/0046要求、Round10–13及密钥/客户端文档。本轮六文件差异中，生产代码仅 **GameBootstrap.ts**：增加清空/重入校验入口、hand/controller绑定的已验证标记和commit门槛。其余为源码匹配测试及范围/使用说明；GameFairnessActions、GameOnlineController、平台适配、协议和后端均未改。

影响范围是凭证UI、异步结果、客户端公平会话的跨房间/牌局生命周期；新增测试优先覆盖这些路径。后端整个 `server` Git tree 在候选与 Round13 相同：`bafabbb8cb75baec79bf94b8b968b297391e83e2`。因此复用 Round13 API 二进制（编译源14748096，SHA256见 provenance-summary.json）及自有隔离产品库；**不声称重新编译了 dc99ad5 的后端**。本轮客户端源码、dist测试和原生导出均取当前候选。

私有证据根 **`E=/root/.config/awesome-zero-platform/acceptance-0046-r14-dc99ad5`**，目录0700、证据0600；`runs.jsonl` 保存实际命令、cwd、目标SHA、UTC、耗时、退出码，`sha256.json`索引。配置、凭据、恢复套件、截图及临时探针不提交。复用Linux/Node22.23.4、已有MySQL5.7.44/Redis8.8.0及SDK；测试重型构建顺序执行，没有下载引擎、Wine或其他OS。

## 2. 备份门槛与安全检查

实际调用当前源转译的 GameBootstrap 方法、真实 GameFairnessActions（Cocos基础节点在Node逻辑探针中替代），独立验证以下行为，不只采用 assets.test.mjs 的正则匹配断言：

| 场景 | 结果与证据级别 |
|---|---|
| 只生成、直接commit、只点击确认、空输入 | PASS：没有合法重入校验就不能从UI提交；确认按钮清空显示而不是设置已备份 |
| 截断、损坏、非法格式、其他Hand/Seat、错误digest | PASS：校验失败关闭、未设置verified标记；错误后可重新输入并重试 |
| 重复生成/确认 | PASS：核心inFlight防止并行生成；重复清空不等于完成校验 |
| 正确外部副本重入 | PASS：清空→重新输入→认证projection校验后才绑定当前controller/hand，成功后输入再次清空 |
| import等待期间离房、新手局、换controller、销毁 | UI verified标记隔离PASS；但公平会话旧secret的释放另有N02，不能据此说整个生命周期通过 |
| prepare等待期间离房 | **FAIL N01**：缺少相同的上下文围栏，旧凭证回填 |
| 取消/结束旧手局后进入下一局 | **FAIL N02**：旧secret仍阻塞新的凭证准备 |
| 请求失败/超时、未知commit结果、重复提交 | PASS：失败验证不设置proof；可重试；未知commit维持原requestId/完整请求，重复commit不创建新请求 |
| 已验证后server版本变化 | PASS：commit重新读权威版本；独立探针验证expectedVersion更新，真实三人序列提交通过 |
| 将完整kit作为房间号 | PASS：实际HTTP客户端在transport前拒绝（0请求），不会把完整凭证放进房间URL |

Kit绑定Hand/Seat，完整性由贡献/承诺计算和权威projection共同验证。**未承诺时**只有自身digest及当前认证座位可校验；**已承诺后**必须精确匹配服务端commitment。不是服务器签名的备份证明，也不能用“匹配服务端承诺”的统一文案暗示未承诺时已有权威commitment。账号绑定依赖认证API的手局席位授权，而非kit内一个AccountID字符串。

生产流程无自动kit上传、平台明文持久化、自动剪贴板保存或Math.random替代。Kit只在本机输入框/内存展示，commit发送公开commitment，随机贡献经既有Manifest/AAD/HPKE链路揭示。真实kit未出现在app、SDK工具和原生探针常规输出日志中。共享输入框的完整kit长于合法room locator上限，实际API负向测试确认本地拒绝。

**重新粘贴仅证明校验时持有副本，不证明已经安全持久保存。** 操作系统剪贴板、输入法、调试器和外部备份介质不受此UI自动清除保证。本轮外部文件仅是受控测试备份，不代替人类实际保管体验。遗失后的可行动中文指引、各设备复制/粘贴和可读性仍须补用户验证，不能仅凭说明文字PASS。

## 3. 真实恢复及原生证据

### 真实HTTPS/WSS/数据库恢复

`real-recovery.mjs commit` / `restore` 两个**独立Node进程**均exit0：

1. 四个正常Go账号真实登录，三个不同账号实际建房、加入、准备、开局。
2. 三个当前Bootstrap流程分别生成kit，保存到游戏进程外私有文件。未验证commit实际不被提交；清空后从外部文件重入、验证，再提交成功。
3. 第一进程dispose所有控制器并退出。第二进程新建会话及Fairness实例，原secret不存在；从真实server房间projection恢复hand，从外部文件导入各自kit。
4. 三人通过正常Manifest/AAD/HPKE加密揭示，server revealed=true、客户端secret清除；正常结束测试手局并离房。

没有保留旧实例内存来完成恢复，没有替换服务端、公平签名或加密校验。见 `real-commit.log`、`real-restore.log` 及对应JSON。

### 当前源码Linux原生导出/社区SDK

使用现成官方Linux cocos-cli `f506fa3de403b48b4833334fa48dddd27bb26364` / engine `d0a696cfe63fd8d19c70d7913ba11aa641c2e3c7`（4.0 beta基线），仅重导出受影响的WECHAT目标。仓库 `scripts/cocos-linux-wechat/build.cjs` exit0，约264.6s；没有重建或下载引擎。26文件原始manifest SHA256 **`a583dc2982cdfb9b202e33237ad609be7fe4062ac17084578ce9404d782bc442`**。模拟器兼容的projectname/baselib设置另记，原始manifest保留。

Linux社区微信工具2.02.2609240-2/SDK3.17.5，实际引擎报告4.0.0：真实wx登录、三人房间、当前E03按钮触摸、未验证commit拒绝、清空、空导入拒绝、外部副本校验、commit、重新登录替换controller并销毁原Fairness实例、从外部副本恢复及实际加密揭示，共 **16项操作检查通过**，`native-e03` exit0。当前新增按钮和门槛确实在新导出中运行，不是引用旧包截图。

**输入证据限制：** 焦点EditBox的SDK `Input.insertText`返回sent，但未观察到string改变；采用从私有外部文件回填真实EditBox后再实际触摸验证/commit，379字符组件往返未截断。故这证明真实控件和门槛/协议，不证明人工剪贴板、系统键盘或真实手机UX通过。输入maxLength2048、完整kit可容纳不等于实际粘贴入口已资格化。

已目视复核 `native-after-reveal.png`：按钮可触达，输入区仍有默认Label及内容拥挤现象，需输入/渲染适配专项核查；不把它直接认定为本轮新增P2，也不宣称所有布局与文字清晰。社区工具不是官方真机/发布资格。

## 4. 新发现缺陷

### R14-N01 / P2：旧prepare结果在权威离房后回填恢复秘密

- 位置：`GameBootstrap.ts:368–378`，尤其`:371` await及`:373–376`无上下文复核的状态/输入写入。
- 最短真实复现：有效手局开始prepare；让认证fairness HTTP响应真实到达后延迟交给调用方；另一合法成员取消手局，其他成员离房，owner最后权威离房；此时roomID/handID均空；释放旧响应。
- 预期：旧generation/controller/hand结果不能改变当前UI或保留终止手局秘密。
- 实际：新的空输入框被填回 **356字符旧kit**，core仍持有旧secret；退出后的当前界面再次显示旧凭证。没有证明跨账号越权或服务端数据泄漏，不按P1夸大。
- 证据：`ui-probe.mjs`、`independent-ui-corrected.log`、`real-stale-prepare.log/.json`。后者是当前源Bootstrap控制流+Cocos节点替代+真实认证服务/数据库，不冒称该延迟注入已在原生中复现。
- 建议：prepare和其他会写秘密的await捕获并复核controller、hand及room generation；失效结果不写入新输入框，同时按权威生命周期处理旧秘密，而非只隐藏文本。

### R14-N02 / P2：取消/超时后旧公平秘密阻止下一局

- 位置：`GameOnlineController.ts:107–120`成功leave只清shell；`GameFairnessActions.ts`的`secret`生命周期及`prepareRecoveryKit`的different-hand保护；Bootstrap复用同一online/Fairness对象。
- 最短真实复现：三人开H1，owner生成kit但不commit；合法取消H1（server ABORTED），其他成员先离，owner成功离房；同一控制器建新房并三人准备/start H2；生成H2kit。
- 预期：已确认终止的旧hand不阻塞H2；应有新的独立恢复材料。
- 实际：`oldSecretRetained=true`，server H2已为 `FAIRNESS_COMMITTING`，但客户端无kit，提示 **`unrevealed contribution for a different hand`**；下一局无法完成备份/承诺。重新登录可替换控制器，但不是完整流程修复。
- 证据：`real-leave-after-cancel.log`、`real-leave-repro.json`。初次直接退出活跃局被服务端正确拒绝，不算缺陷；有效反例明确遵循取消/终止和owner最后离房约束。
- 建议：在权威终局/成功离房/切换新局时安全释放或替换Fairness会话；保留unknown-result必须精确重试和Hand/Seat不匹配拒绝，不能通过删除安全检查、重用旧贡献或无条件丢弃活跃秘密“修复”。

两项是**本轮新发现的既存生命周期缺口**；关键根因在此前已有逻辑/未改动类，不能据发现日期声称首次由dc99ad5引入。新增E03门槛未覆盖这些失败路径，故整体FAIL。修复交回ChatGPT，验收方未改实现。

## 5. 自动化、回归与范围记账

| 本轮真实执行 | 结果 |
|---|---|
| `npm --prefix clients/fair-doudizhu-cocos test` | 42 PASS/0 FAIL，exit0；含shell编译 |
| `npm --prefix clients/fair-doudizhu-cocos run typecheck` | exit0，包含当前Bootstrap静态类型 |
| 独立当前源UI状态探针 | 12项合计10 PASS/2 FAIL：主组7/2，unknown/timeout两项PASS，kit非room一项PASS；不把不同轮次重复计数 |
| 外部备份→新Node进程→真实恢复 | commit/restore均exit0；三人实际加密揭示通过 |
| 两个真实服务失败反例 | `real-leave-after-cancel` / `real-stale-prepare`均exit1，保留FAIL |
| 当前WECHAT原生导出/16项SDK操作 | 均exit0；键盘/人工粘贴限制见§3 |

完整argv、cwd、SHA和耗时见 `runs.jsonl`；临时程序均在E。初始Cocos节点替代缺destroy方法、直接退出活跃局的错误前提及SDK首次open相对路径ENOENT均保留为夹具/环境记录。修正后不改产品或安全断言；SDK确认进程/CDP/CLI端口就绪后一次重开成功，不以初次环境失败否定最终操作结果。

复用 [Round10](0046-round10-independent-acceptance.md)、[Round11](0046-round11-independent-acceptance.md)、[Round12](0046-round12-independent-acceptance.md)、[Round13](0046-round13-independent-acceptance.md) 中与未变实现兼容的后端、完整Bid/Play/Pass、WSS、持久化/冷恢复、终局、私牌、稳定根/禁用rotate及规模性能证据。未重跑全后端SIGKILL矩阵、100/1000或其他平台全构建；不将这些历史结果冒充此次新执行。当前CI仅作支持，不能覆盖N01/N02。

### C09/C10正式延期

用户批准的正式Goal开头scope amendment、current最新状态及operations保护根文档一致：**C09/C10为DEFERRED、未来可选Goal，不是PASS，不再阻塞当前版本**。仍要求稳定管理员根、安全备份、HMAC、Seed/Checkpoint AES-GCM、错误根失败关闭、原根恢复和旧rotate禁用；这些实现未变，Round13证据仍适用。历史Round12/13矩阵原样保留，不重新解释其当时FAIL/BLOCKED。

| 相对调整后基线的状态变化 | 理由 |
|---|---|
| E03：BLOCKED→FAIL | 两项当前可复现生命周期缺陷，而非仅缺用户体验证据 |
| A04、H08、GOAL3：历史PASS→FAIL | 分别对应新局公平承诺受阻、旧秘密污染当前UI、完整客户端公平链路不能持续使用 |
| H05、GOAL8：BLOCKED不变 | 内部适配及外部资格均未完成，见下一节 |
| C09、C10：DEFERRED | 用户正式范围调整，不代表功能实现 |

其他106项按影响核查后引用历史PASS；本轮不声称逐项完整重跑。**当前范围112项=106PASS+4FAIL+2BLOCKED，另2DEFERRED；总计114。** 受同一反例影响的关联FAIL不是独立缺陷数量。私有 `scope-delta.json` 保存记账依据。

## 6. H05/GOAL8及最小开发交接

**H05：BLOCKED，不能全部归为外部资质。** 已有target检测、纯TS协议及注入式mini SDK的HTTP/WSS/session/random/lifecycle/share基础；缺熵能力时失败关闭。当前Linux导出扩展仅wechatgame，**没有合格抖音导出/SDK证据**；Bootstrap登录仍只读取wx并使用微信身份交换，缺实际抖音启动/业务登录接入。这些是内部工程/适配缺口；正式tt小游戏熵、生命周期、网络及设备SDK验证则另需平台环境。本轮未反复尝试无资质平台或下载别的OS。

**GOAL8：BLOCKED。** 本轮新原生导出、真实wx登录及SDK操作证明了可用开发路径，已有AppID/AppSecret不再被记为缺失。但Linux社区工具、beta/实验导出不是正式发布资格；核准HTTPS/WSS域名/生产TLS、官方受支持工具、真实手机、多用户前后台/网络/分享及发布授权尚未完整验证。另有当前E03代码缺陷和键盘输入/可读性待核查，不能全部解释成外部配置问题。

交回ChatGPT的最小清单：

1. 修复N01的异步上下文围栏及N02的权威生命周期清理，新增真实取消/结束→离房→新局回归，保留未知结果/旧hand拒绝保护。
2. 补实际EditBox键盘/粘贴与长凭证可读性验证，精确区分“未承诺自校验”“已承诺匹配server”和“持有副本/长期安全保存”；补遗失后的可行动指引。
3. 按现有范围补抖音内部入口/导出/身份适配及SDK资格；外部域名/真机/发布条件单列推进。**本轮不开发C09/C10**，以后另经授权Goal再做。

Goal0045与0046均继续 `in_progress`。不建议最终封版；先进入针对上述缺陷的开发修复及受影响/必要关联复验。验收完成不等于产品通过。

仅提交本报告及current允许状态文字；报告提交SHA与产品SHA分开，最终总结及 `git log -1 --format=%H -- docs/goals/0046-round14-independent-acceptance.md` 可查询。私有证据/备份保留；本轮GUI、API、TLS、自有数据库容器已停止，未清理全局Docker或修改其他部署。
