# Goal 0056 — 微信构建兼容性与真机启动独立验收

日期：2026-11-08 UTC。执行者：Codex。固定产品 SHA：**`06bb6a79caf9b0908693e015cae82d4d76866872`**。

## 1. 当前结论

**BLOCKED：构建、微信真实处理、模拟器和 HPKE 验证通过，尚未取得本轮荣耀手机扫码后的实际启动与操作反馈。**

- **G55-N01：PASS，可按本候选关闭。** 新产物所有项目 JS 完成语法降级和 AST 审计，默认配置实际开发预览成功，模拟器场景及触控通过；不再复现 `Unexpected token ?`。
- **G55-N02：未完整关闭。** 最终处理后 BigInt、真实微信 SDK 熵和 HPKE、同预览配置模拟器均通过；必需的实际手机首页及交互证据仍缺，不能以工具结果替代。
- **G55-E01：继续 BLOCKED，单独评估。** 合法公网 HTTPS/WSS 完整联调未证明，不与启动兼容性混为一个问题。

本轮未确认新的产品兼容性缺陷。报告记录当前阶段，不能据此将 Goal0056 判为最终 PASS；收到本轮真机反馈后继续同一轮验收。Goal0045/0046、E03、完整微信三人局及正式发布资格均未解除，**Goal0045/0046 保持 `in_progress`，C09/C10 保持 DEFERRED**。Goal0055 FAIL 和 Goal0054 专项 PASS 原文保留。

## 2. 固定版本、差异与环境

clean `main` 执行 `git pull --ff-only`，候选为上述 SHA，续验前再次同步仍相同。完整比较上轮产品 `a4551b77d3751bf6afbdeeb14d8cde1be30fb803`：实际实现/测试/操作文档改动6文件，集中在 `hooks.cjs`、`audit.cjs`、`build.cjs`、`test.cjs`、assets测试和构建文档；上轮报告/current变化仅为已提交历史证据。未变更业务、协议、Scene、Crypto实现、Go或Schema。本轮只新增本报告和current允许状态文字，没有修补产品或降低正式断言。

复核AGENTS、原则、current、Goal0054/0055、交付缺口审计、Linux构建文档和全部相关构建/配置/审计脚本。新审计对上轮两份真实失败产物分别拒绝 `assets/main/index.js` 的 `??` 和微信处理后的 BigInt `Math.pow`，不是仅采用开发方修复结论。

- Linux x86_64，Node22.23.4，约7.75GiB RAM，无swap；单次构建3GiB/CPU100%限制。无新安装、Wine、其他OS或Android模拟器。
- CLI：`/data/awesome-zero-platform-tools/round7-cocos-cli`，revision `f506fa3de403b48b4833334fa48dddd27bb26364`。
- Engine：上述CLI的 `packages/engine`，revision `d0a696cfe63fd8d19c70d7913ba11aa641c2e3c7`，包4.0.0-beta.0；实际运行字符串4.0.0，不冒称正式稳定版。
- 复用Linux社区微信开发者工具2.02.2609240及既有登录profile，实际 `islogin=true`；基础库3.17.5。工具是Linux ELF，兼容层的Windows UA不代表Windows虚拟机。
- 模拟器iPhone12/13(Pro)，844×390，DPR3，safe-area来自真实SDK。它不是荣耀手机实测。

私有证据目录：`/root/.config/awesome-zero-platform/acceptance-0056-06bb6a7/`，含原始日志、脚本、产物、完整manifest、处理后源码、截图与 `SHA256SUMS`。登录会话、AppSecret、Token、二维码和私有设备材料不提交。

## 3. 精确候选CI及本地执行

独立查询全部job及其head_sha/status/conclusion，未重触发成功CI：

| 工作流 | 最终结果 |
| --- | --- |
| [主CI 42459681085](https://github.com/oublie6/awesome-zero-platform/actions/runs/42459681085) | SUCCESS，unit/integration/admin_web/secure_envelope/runtime/report，6/6 |
| [Fair 42459681106](https://github.com/oublie6/awesome-zero-platform/actions/runs/42459681106) | SUCCESS，application/mysql_integration/client_foundation/manifest_client，4/4 |
| [Cocos Linux 42459681149](https://github.com/oublie6/awesome-zero-platform/actions/runs/42459681149) | linux_source SUCCESS；linux_native_export SKIPPED |

实际执行现有命令（私有目录简称以完整日志为准）：

```sh
export COCOS_CLI_ROOT=/data/awesome-zero-platform-tools/round7-cocos-cli
export COCOS_ENGINE_ROOT="$COCOS_CLI_ROOT/packages/engine"
node scripts/cocos-linux-wechat/verify-toolchain.cjs
node --test scripts/cocos-linux-wechat/test.cjs
bash scripts/verify-client-foundation.sh
node scripts/cocos-linux-wechat/build.cjs /private/goal0056-native
node scripts/cocos-linux-wechat/audit.cjs /private/actual-wechat-processed
```

Node22.23.4路径、离线npm缓存和已授权AppID在私有执行配置中指定，不传AppSecret到客户端。工具6/6、通用客户端31/31、斗地主协议49/49、场景/客户端106/106，**192/192**；相关类型检查和真实CLI生成引擎声明的严格检查通过。静态Creator3.8.8类型辅助不等于历史Creator产物。

08:13:59–08:24:45 UTC完成全流程，原生 `build.exit=0`，CLI实际wechatgame阶段自报7min。真实builder日志两处记录 `targets={chrome:"79"}`，不只是读取配置源码。新target触发引擎重编译，过程持续CPU工作，没有把长编译误判为挂起。未使用旧产物或将Web包冒充WECHAT。

内建未使用3D资源的cmft/FBX工具限制保留在日志中，2D构建成功；不由此宣称3D资格。分包/远程资源仍不具备实现和验证，当前是本地不分包小游戏。

## 4. 原始产物、配置与AST

输出：私有 `native/fair-doudizhu-wechat/`，**26文件、3,113,852字节**，其中11份JS全部AST通过。核对game.js/game.json、实际Entry.scene资源、CommonJS/SystemJS引导、官方adapter、业务/依赖/引擎chunk。默认 `es6=false,minified=true,urlCheck=true,enhance=false`，没有私有配置覆盖。只设置独立本地projectname；AppID由构建环境提供，没有改JS或为通过预览更改设置。

完整本地产物确定性归档SHA256：`f00adf3582859cfe89686fdbaf2bedcc3af47263a9ea3c8ca3f4642016f7f780`。

| 原始文件 | 字节 | SHA256 |
| --- | ---: | --- |
| `application.js` | 1269 | `60633140454acc1070f4eac53f89828e86f2c8471bb8f80e77e42fac68b4ff3a` |
| `assets/internal/config.json` | 2439 | `e021b9fe38aebffd2ebd0382f1fd960eb98a01e810db747426b30e1b567a9226` |
| `assets/internal/import/07/078d1fd7a.json` | 197930 | `139094c6a6e40219d1547ebfa516abd4c9c91d9a9cc921ed32e281d9c2b0e55c` |
| `assets/internal/index.js` | 43779 | `6d232cacaf7b8e9a617e0d2498304b683a2deee65a760c5777ccabe46aa389fd` |
| `assets/internal/native/08/0835f102-5471-47a3-9a76-01c07ac9cdb2/OpenSans-Regular.ttf` | 217360 | `e64e508b2aa2880f907e470c4550980ec4c0694d103a43f36150ac3f93189bee` |
| `assets/internal/native/0e/0ed97c56-390e-4dd1-96b7-e7f2d93a98ed/OpenSans-Italic.ttf` | 212896 | `a54dc8488f8193bf30c3820cf6f261f911f9d328d699e1a1b8042641554cec70` |
| `assets/internal/native/b2/b23391b6-52eb-46a6-8da1-6244d9d315fb/OpenSans-BoldItalic.ttf` | 213292 | `3ca680f2444cc9e50447d057c006464566e92f2e77b0b6e26491e5bd757ed4e7` |
| `assets/internal/native/b5/b5475517-23b9-4873-bc1a-968d96616081/OpenSans-Bold.ttf` | 224592 | `5894a3649b213cf5b2d673b6e7a871815fd1d120fa68a463592f27db14eae323` |
| `assets/main/config.json` | 582 | `51723a5b4c662d43b5eb9264df1effdc92e9140bfa4eb03cd7098a97f4d846c9` |
| `assets/main/import/ba/ba21476f-2866-4f81-9c4d-6e359316e448.json` | 192 | `0a34e9263b215bb661d6334db6a255497c6cdac1f1c40b130f50bf9d91f4dd39` |
| `assets/main/import/e9/e9a5f2e4-6d64-4363-814e-dab771327440.json` | 1350 | `a7879d77c700c29297c1070d677dffdf8176944f0f1a2fdb72674c2285ec4ea4` |
| `assets/main/import/fd/fd8ec536-a354-4a17-9c74-4f3883c378c8.json` | 480 | `c2eee4552a460ed97591f527e127b95c811d5f796b9448f3f705974af9beba9d` |
| `assets/main/index.js` | 80427 | `f83e66ca0da6f450eb37f644ab9e5d928c966020fc75ed68feee84c2a957e750` |
| `cocos-js/cc.js` | 1390036 | `c873f149dfdc342fdcfb69c129ac0edfd752f424fb10d516949618f3f2513097` |
| `engine-adapter.js` | 25141 | `4f3d6a3148449b899ae1b61e6b1387f8dcde8bbf4ce7422d2a8de0fff129942e` |
| `game.js` | 645 | `4635cf7530900863cd82f6b6221b8d18b8651e490347499a1001fcfc51934f21` |
| `game.json` | 141 | `bd42e4367355c19fe493ea9cb6e27f0132690c2d45341665bd43cdd435200ecf` |
| `linux-builder.json` | 152 | `b5aed182a6679894bb1bf5779ff8b36736227688631b2751e183a2728ddb6c5c` |
| `project.config.json` | 233 | `50b36697905b4f11d8f2b7f03f5fd16a3ef6b2eb95724e28081c2fa4c9684e96` |
| `src/chunks/bundle.js` | 160029 | `c9be3d64a33451fc8e34bbfb5cfd6aee4e20476fa77c4a89ad6ce32b5398c714` |
| `src/effect.bin` | 5755 | `b0d92f8307e3109abe8af8a863d68feb58cf4f49cf000b1d48fbb3dc3ad45556` |
| `src/import-map.js` | 200 | `ab69617983b59f99feb82086911e3717317e3c40d51676a783ad3cb6f5f01b9c` |
| `src/polyfills.bundle.js` | 13117 | `44edd4b2861183676aad27df3844cb18cd97802a4d256496c6dc94452c89b45d` |
| `src/settings.json` | 206267 | `ffea6cb67b323a310c0ce17331ef687308c0b8c16faba34385e74f40b44a2f8c` |
| `src/system.bundle.js` | 8246 | `f7a5d86e9025c2f9d1dd72f8875bc6373db9daa7a9b99529a4a0a2081f93765b` |
| `web-adapter.js` | 107302 | `c0c19caf7480d07268580f9492d6376b88359eae74c5c865f01cd4cd47bbbdad` |

## 5. 微信实际处理后代码与预览

对新项目进行真实导入、编译，从SDK Debugger.getScriptSource取得**全部11份原始项目JS的实际处理结果，加1份工具生成入口包装，共12份，无缺失**。包含引擎、两个adapter、业务及 `src/chunks/bundle.js` 依赖，未仅检查业务chunk。平台WAGame SDK和IDE自身不是项目上传包，不冒称审计了它们。

实际执行仓库 `audit.cjs` 的CLI入口，12份全部通过。处理后快照的确定性归档SHA256：`5d2cf03a8ede6655d14d8b643e0002ebc43e8e9c64fc401d807b4a8a010b5cea`。CLI只返回预览体积，没有提供远端上传包整体二进制；以下是实际工具编译执行的源码哈希，不伪称远端包整体哈希。

| 实际处理后文件 | 字节 | SHA256 |
| --- | ---: | --- |
| `__wechat_wrapper__/game.js` | 442 | `1411337474ddb914dbf868e2bf8d56fc5d5d49b80e6a85710bbfb39b6f91c7ed` |
| `application.js` | 3575 | `b1f49d415cb2f5d02e1ac1727331817a4ab6462d9fc74cf8d587ab2b226e4c36` |
| `assets/internal/index.js` | 103511 | `6426d0184a440c674d31b4a194d02e63e046ae02d4761b374f0f4f088d9d2b20` |
| `assets/main/index.js` | 190423 | `332545be9d4afd22bfa7b5693ed72b60953391ee4ec22cbd38e5b7b7f4eca855` |
| `cocos-js/cc.js` | 3257334 | `656b0c3f0f7b6a7218be1bf9bc842bd13cb51f9417dace7da65ab06dc8aeaf0c` |
| `engine-adapter.js` | 59054 | `8176a59ff8aacb421ba491b8bd1de4ce10f9662dbc717f271940522fdfac2c35` |
| `game.js` | 1928 | `7fce76b556011470efd7ebfd25725543aea72dbc8e8e2610a507248d846294be` |
| `src/chunks/bundle.js` | 379129 | `3cf1686126a9cfab12b4e65765cbab755e9011618714b7e292b6feabc3f95870` |
| `src/import-map.js` | 837 | `73683b3b42159a23cadb76bd549137f23c7119cb0f6fc39de6329906893e9ee0` |
| `src/polyfills.bundle.js` | 31236 | `df46566a0b5a505fd06dd4a316c80b3691ad59a63bd281e8fbe01812e5e3cb19` |
| `src/system.bundle.js` | 19730 | `812e83a98cb326af702a87cd91e38e26884706d8550404ccfa282890d7caad7b` |
| `web-adapter.js` | 251708 | `57ce7283f43406c566161632b0bc762b7de323ab2c62be65a86a0c80d5d34dc3` |

| AST项目 | 原始11份 | 微信处理后12份 |
| --- | ---: | ---: |
| nullish `??` | 0 | 0 |
| optional member/call | 0/0 | 0/0 |
| BigIntLiteral | 71 | 71 |
| 带BigInt字面量的原生指数表达式 | 1 | 1 |
| Math.pow调用 | 3 | 3 |

3处Math.pow已查看，属于字节宽度与引擎数值缩放，未发现BigInt参数；不能将所有Math.pow都视为缺陷。另执行10项AST正负控制，覆盖直接BigInt、BigInt调用、计算属性pow、变量/常量多级别名、真实原生幂、字符串中的符号和普通数值运算；均符合预期。AST审计不是任意JS程序的完整类型证明，结合实际处理代码和运行结果判断。

**默认配置开发预览SUCCESS**，实际报告 **3,091,558字节**；不复用上轮3.38MB数字。新QR已生成并私下提供，来源、设置、时间、QR/配置哈希存于私有device-preview.json，且与Goal0055旧码不同。只有开发预览，没有审核、生产上传或发布。没有复现原 `Unexpected token ?` 或包大小超限。

## 6. 模拟器、SDK、HPKE与退出

- 对本轮新产物运行smoke.cjs；真实Cocos事件触摸首页→游戏说明→返回→房间成功。标题“公平斗地主”、场景及模块加载正常，原生 `2n ** 256n` 正确，wx安全随机返回32字节。
- 实际wx.login临时代码、临时session存储/清理、前后台监听注册/移除通过。分享仅确认接口存在，没有发送消息或将送达记为通过。
- 14项实际房间菜单均经原生翻页可访问；前后翻页保留EditBox标记。标记通过程序赋值，不冒充软键盘实测。登录入口响应并显示缺少公网HTTPS/WSS配置；SDK取码与完整后端登录分开计证据。
- 实际导出的HPKE模块使用真实wx SDK熵封装，独立可信Node接收端成功解密，错误AAD拒绝，未改用假SDK或禁用验证。依赖chunk实际参与加载；BigInt语义不是只靠直接eval一个算式推断。
- 当前源代码未删减密码学实现，完整BLS/牌序证明沿用未变服务端历史证据，不声称本轮重跑整套证明或手机HPKE业务提交。
- 真实控制台递归读取Shadow DOM：35个shadow roots、9条消息、0条error；初始body文本无法读取其内容，已补证。未观察到导致白屏的初始化Promise拒绝、语法/BigInt异常或资源加载失败。
- CDP页面截图出现高DPR合成裁切，与真实X11帧缓冲不一致；以X11截图核对可见布局，不将抓图失真当产品故障。
- 最后实际销毁Bootstrap：component/content/全部按钮失效，resize和resolution监听从存在变为不存在，退出清理通过。

首次工具实例按1小时上限于09:25停止（Result=timeout，非OOM）。续验前核实终态，复用原profile/同产物重启；未重复构建和192项测试。续验再次确认login=true、场景和运行检查通过。已停止本轮工具进程，避免等待手机时继续占用资源；登录profile保留。

## 7. 真机证据缺口与外部网络

用户已有荣耀100Pro测试条件，前轮提供MagicOS10/Android16；本轮已提供**新候选新QR**并请求：加载结束、首页标题/按钮、规则进入返回、房间与分页、输入框软键盘/长文字；正常后再前后台和关闭重入。**截至本次记录，没有收到本轮实际扫码、页面或操作反馈，也没有新真机截图**。现有上轮加载失败图片不能替代本轮证据。

因此DEVICE为 **NOT RUN/待用户反馈**；微信版本、手机基础库、网络和本轮实际启动结果未取得，不编造。N02在这一步之前不得完整关闭。未声明无手机、未登录或缺AppSecret；这次缺的是本轮真实执行反馈。

G55-E01单独BLOCKED：既有客户端测试配置仍是127.0.0.1加自签TLS，当前连接集群ingress为0，未取得可供小游戏使用的批准公网测试端点/有效TLS。本轮做了配置和环境再审，没有把旧TLS失败说成新运行结果。没有关闭urlCheck/TLS、加入忽略证书参数、修改AppID权限或把AppSecret置于客户端；没有将SDK临时代码等同于完整微信换证→HTTPS/WSS→房间成功。

## 8. 复用、收尾与后续

服务端树仍为 `06209f0fc5a8f9bb8a50682c69cc6f195f354e18`，7份Goal0055真实Go完整局/v60/SIGKILL/本人字符串证据哈希一致。本轮未开启Go/MySQL、未重跑完整局、数据库故障矩阵、整套密码学或压测；历史证据作用域不升级为当前原生/真机业务E2E。

私有manifest和产物设置已复核无变化；7个已知服务端秘密值在原始和处理后客户端无匹配。没有产品/正式测试/Schema/生产数据更改。资源采样及进程日志保留，工具3GiB限制、OOM计数0，均不当作手机堆或生产性能SLA。

| 层次 | 当前结果 |
| --- | --- |
| SOURCE | PASS：固定差异、192/192、类型及AST正负控制 |
| NATIVE EXPORT | PASS：新WECHAT包、真实Chrome79、默认设置、11份AST |
| WECHAT PROCESSED | PASS：12份完整实际处理代码、AST、预览体积与成功结果 |
| SIMULATOR | PASS：场景、真实触摸/分页、SDK熵、HPKE/错误AAD、清理 |
| DEVICE | NOT RUN/待反馈，不计PASS |

**剩余必需工作：取得本轮荣耀手机真实启动和最小UI反馈，必要时采集其失败现象/日志，再更新同一轮报告和current。** 本阶段报告按main-only提交；不将阶段BLOCKED当作整个目标完成。即使后续本专项PASS，Goal0045/0046、E03、合法网络、完整微信对局和正式发布仍按各自门槛判断。
