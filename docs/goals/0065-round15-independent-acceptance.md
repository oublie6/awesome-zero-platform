# Goal0065 Round15 清理异常隔离及收尾独立验收

**PASS：G65-R14-N01/P3关闭，适用源码/异常/浏览器/部署门禁通过；Goal0065正式completed。** 固定候选 `efc0699b0d62ae1284db86121619a746ce4f37ba`。未修改产品代码；最终API→Web已部署至现有测试集群，包含此前尚未上线的服务端/客户端改造。Goal0045/0046保持in_progress；不是移动真机、WeChat或完整产品Goal0046的发布资格证明。

执行日期2026-11-27 UTC；干净main经 `git pull --ff-only` 同步，审阅 `2165a2a2f7df57be384461dff0a83cf0ef81ef17...efc0699b0d62ae1284db86121619a746ce4f37ba` 全部六文件增量、AGENTS、相关原则、当前状态、Goal0065与Round14报告。测试限定项目自有环境，无手机黑屏调查或真机测试。本报告保留三次部署尝试、两次逆序回滚及观察器修正，最终PASS不覆盖失败历史。

## 固定 CI 与产物身份

固定候选三项适用工作流、全部job均SUCCESS：[Vue](https://github.com/oublie6/awesome-zero-platform/actions/runs/45923418552)、[Fair Doudizhu](https://github.com/oublie6/awesome-zero-platform/actions/runs/45923418549)、[CI](https://github.com/oublie6/awesome-zero-platform/actions/runs/45923418533)。见 [ci.json](evidence/0065-round15/ci.json)。Cocos按授权复用Round13成功结果；不声称本SHA重新执行native export。

只重新构建Vue一次，`npm test` **40PASS/0FAIL/0SKIP**，包括本轮新增7组（6组auth、1组App呈现约束）；`npm run build` 包含vue-tsc --noEmit及正式Vite构建，通过。命令/结果和原始测试输出见 [checks.json](evidence/0065-round15/checks.json)、vue-tests.log、vue-build.log。源码来自固定SHA的git archive，1654个普通tracked文件全部字节一致，见 [source-artifact-identity.json](evidence/0065-round15/source-artifact-identity.json)。

Cocos源/配置、共享游戏协议、server、scripts、deploy及workflow相对Round14未变。直接复用其核验的Round13官方Cocos4 Web产物：24文件SHA256逐项一致，4.0.0-beta.0，CLI f506fa3de403b48b4833334fa48dddd27bb26364、engine d0a696cfe63fd8d19c70d7913ba11aa641c2e3c7。API复用server tree `a8f14fec9d7703c0fe68ac286bbd4aa9081fefbe` 的二进制 `dc50adc3bdc3fabe04b64e0a66f417df44a1fb56fc35a2f3f225e9cdb714c000`。本轮Cocos导出0、Go构建0；镜像打包及运行库修正详见部署段，不混称Go重建。[artifact-reuse.json](evidence/0065-round15/artifact-reuse.json)。

## 原26场景复验：全部PASS

复用Round14 `storage-audit.mjs`，只替换源路径/SHA，直接加载生产WebAuth，未复制实现或调整26个业务期望。[storage-audit.json](evidence/0065-round15/storage-audit.json) **26PASS/0FAIL**。

| 原六个失败场景 | 本轮实际结果 |
| --- | --- |
| logout清理getItem抛错 | 本地身份失效，捕获old的服务端注销实际调用一次，清理错误不再替代结果。 |
| logout remove前抛错 | 同上；允许底层拒绝删除时保留本地old，但仍撤销捕获凭据。 |
| logout remove后抛错 | 本地已删仍报告清理不确定，服务端注销不被跳过。 |
| restore验证失败后清理getItem抛错 | 返回null、身份无效，清理异常不再逃出。 |
| restore验证失败后remove前抛错 | 返回null；不伪称物理删除成功。 |
| restore验证失败后remove后抛错 | 返回null，本地/待定状态释放，后续新登录正常。 |

其余20场景保持通过：读写前后同步dispose/logout/restore/同实例与共享实例新login、getter读异常、部分写入、原错误对象保留、精确write lease释放及新凭据隔离。原Round13五个失权反例另作直接复验5PASS；没有把logout结果扩展为撤销替换身份或撤回已派发业务。

## 关联异常专项

Vue WebAuth拥有本地身份/存储与HTTP生命周期，App拥有呈现，Go仍是身份权限权威。没有新storage key、额外正常HTTP、无界队列、后端/schema/协议改造。清理函数只接收步骤及静态诊断；业务请求、已确认/未确认结果和清理状态分开，常数级所有权机制保持原样。

| 要求 | 实际证据 |
| --- | --- |
| 存储与诊断同时抛错 | get/remove各覆盖成功、原服务端错误、未确认结果；每步仍独立，warning仅一个固定字符串参数，不带原错误/凭据。 |
| 清理与注销错误同时发生 | 原服务端错误对象原样拒绝；成功才返回storageCleanupFailed，清理警告不替代业务结果。 |
| 取消与HTTP资源清理错误 | abort抛错时多个取消步骤仍尝试，旧交换迟到返回null、最新身份保留；clearTimeout/off/诊断均抛错时，业务原值/原错误保留，其他清理仍执行。底层拒绝操作时不保证物理释放。 |
| 清理中同步启动新登录 | listener移除、存储比较、诊断回调重入新任务；旧任务不能再验证/保存旧身份，注销只用原old，不删除/撤销new。 |
| 服务端确认/失败/未确认 | 仅OK envelope且loggedOut:true确认为注销；HTTP拒绝、请求错误、loggedOut:false/缺失及非OK不被包装成成功；无捕获token时不认领存储值。 |
| 临时write及补偿均失败 | 原write错误对象保留，诊断失败隔离，精确lease释放，后续新登录可成功。 |
| generation/epoch/write机制 | 原26场景和5个旧反例通过；未提交token不被restore提前认领，旧补偿不删除替换凭据。 |
| App呈现及业务隔离 | 新增App测试约束加上实际浏览器两种结果文案验证；先撤销当前呈现，再按服务端确认及本地清理结果提示。 |

新增7组全量测试通过；额外独立 [exception-audit.json](evidence/0065-round15/exception-audit.json) **12PASS/0FAIL**。HTTP移除/计时器故障注入在实际清理后抛错，用于验证异常不会跳过后续步骤；不把它扩写成底层拒绝清理也必然物理释放。没有重跑无关连接管理/同步重入全量矩阵。

## 实际注销、持续删除故障与刷新恢复

用实际Vue构建和匹配真实API，在自有浏览器context注入Storage.removeItem异常，不修改产品源码：

1. 正常登录后点击实际退出按钮，服务端注销请求1次，实际旧凭据 `/auth/session` **401**；残留存储仍为old。
2. Vue显示“服务端已注销，但浏览器本地凭据清理未能确认。”，无旧iframe；没有声称本地已物理删除。
3. 保持删除故障跨刷新持续存在：再次实际验证401，待initialize结束后首页可见、未认证且无iframe，残留凭据不能恢复身份。
4. 恢复正常存储后再次刷新：失效凭据清除，随后新登录和正常注销成功。
5. 对浏览器拦截返回503或OK/loggedOut:false的受控对照，显示“当前页面已退出，服务端注销状态未确认，浏览器可能仍保留凭据。”；真实API验证旧凭据仍200，符合该对照没有撤销的事实。未声称真实Go自行产生了这两个故障响应。

源码环境确认路径1PASS，失败/未确认对照2PASS；[browser-logout.json](evidence/0065-round15/browser-logout.json)及初始观察记录。部署后又完成了更强的同一关键路径：[deployed/browser-logout.json](evidence/0065-round15/deployed/browser-logout.json)明确等待刷新真实401响应及initialize结束的hero呈现，再判断失效身份，避免只看初始登录按钮产生假阳性。最终实际集群同样通过全部确认/残留/刷新/重新登录步骤。

普通原生sessionStorage路径没有注入异常，正常登录、恢复、注销均通过；受控删除失败不被扩大解释成普通浏览器故障。

## 正常业务与历史复用

隔离候选Web/API与最终部署版本分别完成真实登录、官方游戏加载、新建自有房间、网站/游戏导航、断线同房恢复、正常leave、注销/换号。两处均同一iframe复用、新增静态请求0，注销后iframe0、旧token401，换号无旧room/private hand/fairness。见 [smoke.json](evidence/0065-round15/smoke.json)、[deployed/smoke.json](evidence/0065-round15/deployed/smoke.json)。创建房间与离房是实际Go关联业务命令，未新开完整牌局。

两处实际HTTPS缓存均通过：HTML revalidate、配置/manifest no-store、指纹JS immutable、未指纹cc.js条件304/0字节、身份no-store。最终实际30个站点文件逐一HTTP下载核对SHA256，包含当前Vue、Cocos、公开运行时配置及清单；精确CSP与实际加载相符。[deployed/cache-smoke.json](evidence/0065-round15/deployed/cache-smoke.json)、running-image-proof.json。

复用Round9—14仍有效的数据库/性能/崩溃恢复/缓存内存、官方Cocos与连接/清理专项；完整三账号v61、300秒自动操作、UNKNOWN原RequestID重试、终局归档及个人字符串来自 [Round11](0065-round11-independent-acceptance.md)。启动、失联、布局及性能证据沿 [Round9](0065-round9-independent-acceptance.md) 的有效证据链与 [Round13](0065-round13-independent-acceptance.md) 实际产物启动/清理检查保留。对应布局/构建、Go/协议/缓存路径本轮无变化，未把历史数值当作当前重新测量，也未重复全局牌局、Go构建、压力/heap或300秒等待。

## 条件部署：最终PASS，保留失败及回滚

部署前每次确认未完成hand0、deadline0；制作新的私有一致性备份，保留schema hash及原镜像。仅修改既有app-api和game-web的image，保持API Recreate策略及既有Kubernetes ConfigMap/Secret/Service/PVC等；Web镜像内按Goal授权更新CSP和缓存配置。[deploy-gate.json](evidence/0065-round15/deploy-gate.json)。

- **第一次实际部署失败，逆序回滚成功。** 复用API二进制是动态ELF，原静态镜像缺少 `/lib64/ld-linux-x86-64.so.2`，容器报exec no such file、未能运行。Web尚未升级；仍按Web→API恢复原镜像。见attempt1结果/回滚。
- **修正镜像运行库，未改Go源码或重新构建。** 保留原镜像/config及nonroot用户，打包隔离API已使用的libc/loader（2.35-0ubuntu3.18，逐文件SHA256），相同Go二进制hash不变。先以最终镜像在隔离环境nonroot运行并通过/health/ready，再重试。运行中的实际库hash及UID65532也已验证。另一次初始BuildKit将FROM配置ID误作远端仓库的打包失败已保留；改用经配置ID核对的本地base及legacy builder，最终镜像按不可变manifest部署。见api-runtime-packaging.json、container-api-smoke.json、Dockerfile及构建日志。
- **第二次API Pod Ready后业务代理首次502，逆序回滚成功。** 直接观察为旧Web连接同一API ClusterIP被拒绝；不推断已证明Go业务缺陷。部署脚本当时仅等静态Web `/health`，不能证明代理API已可用。修正为有界等待真实代理 `/health/ready` 的ready JSON，再继续业务门禁。没有再改镜像或产品代码，见attempt2与rollout-gate-correction.json。
- **第三次完整API→Web、部署后门禁PASS。** 先确认API Ready及真实代理健康，原捕获QA会话升级后仍200，并显式注销；再升级Web、完成实际业务/缓存/镜像文件验收。[rollout-result.json](evidence/0065-round15/rollout-result.json)、rollout-events.json、cluster-session-after-api.json。

最终运行镜像、OCI revision均指向固定候选efc0699：

| 组件 | 最终不可变镜像 |
| --- | --- |
| API | `docker.io/awesome-zero-platform/goal0065-api@sha256:78083cc8ede1b176b179de55e7bdde7680714cd76fc131c83430a6c5a2d82dd6` |
| Web | `docker.io/awesome-zero-platform/goal0065-web@sha256:af8fdd096931c95f72ddc126521242607e4e5b90a6ee9b63cbf233d78105e5d9` |

原回滚API04614d0b…、Web44995360…及导入别名保留。镜像仅在指定测试节点本地导入，没有发布外部registry。API运行库SHA与native测试环境明确绑定，这是实际测试环境打包，不是另一OS工具链或通用生产镜像资格。实际Pod imageID、OCI标签、运行API二进制、Nginx/cache配置、所有站点文件均核对通过：[running-image-proof.json](evidence/0065-round15/running-image-proof.json)。

历史QA账号仍有可恢复房间，未为测试清理或改写它们。最终通过公开注册创建2个自有QA账号，测试只建并正常关闭一间新房、注销其会话；测试账号保留，凭据仅私有留存。没有SQL删除历史业务或还原备份覆盖共享库。

## 保护资源与收尾

[final-integrity.json](evidence/0065-round15/final-integrity.json)通过：6 Pod/2节点Ready，新API/Web重启0；Deployment仅授权image变化，UID和其余spec保持；依赖Pod UID/重启数不变，ConfigMap/Service/StatefulSet/PVC、Secret值及完整列schema hash不变。集群ABORTED14/COMPLETED11、未完成hand0/deadline0；隔离库ABORTED21/COMPLETED14/未完成0。仓库产品代码、schema与既有Kubernetes配置无验收者修改；Web镜像内CSP/缓存配置是本Goal授权部署内容。

自有API/镜像sanity容器、浏览器、Nginx及4个隔离依赖均停止，18870/19181/30471关闭；临时CA昵称删除且NSS列表恢复，隔离g65r15残留key0。最终用户29457服务active、HTTPS健康200、代理Go readiness200，候选测试部署保留运行。[post-cleanup.json](evidence/0065-round15/post-cleanup.json)。私有备份/原始日志、产物和新QA凭据保留；公开证据经白名单、已知秘密/JWT/PEM扫描，无口令、token、私钥、手牌或个人字符串。

## 完成审计与第二遍证据自审

| 完成要求 | 结论与依据 |
| --- | --- |
| 原缺陷与关联异常 | 26+12专项、40 Vue测试及5旧失权反例通过，原始错误/成功/未知、清理隔离、精确凭据所有权及静态诊断有直接证据。 |
| 真实注销/持续故障/刷新 | 真实401、等待初始化结算后的未认证呈现、恢复删除和重新登录通过；503/未确认结果为明确标识的受控对照。 |
| 原Goal0065启动/布局/失败及性能要求 | 按授权复用Round9—14有效且未受本轮改动影响的证据链；本轮当前构建的加载/认证/复用/缓存/故障文案实际通过，不新增性能或真机声明。 |
| 原Goal0065房间/恢复/UNKNOWN及完整局 | 复用Round11同server tree完整局/UNKNOWN/归档证据；本轮匹配API与最终部署均另做实际创建/同房重连/离房/注销回归。 |
| CI、实际构建与版本 | 固定候选CI成功；Vue构建一次，官方Cocos和API按源/哈希复用；最终实际运行镜像、二进制、运行库、30文件全部匹配。 |
| 条件部署与保护资源 | API→Web最终通过；前两次失败均执行Web→API回滚；备份、schema/Secret/PVC等保护及用户29457检查通过。 |

重新审视了所有增量及直接关联调用、故障组合的预期与实际结果、第一轮networkidle观察超时、源与镜像身份、两次真实部署失败、native依赖修正、代理健康门禁以及后续实际业务。初始确认路径因额外networkidle等待超时，改为业务状态谓词后通过；最终又补充部署后真实401+initialize结算观察，防止读取初始登录按钮造成假通过。原始观察和回滚结果保留，未用进程exit0替代case结果。

此为证据/报告第二遍自审，不冒称另一位独立审阅者。无未解决的适用FAIL/BLOCKED项；未重复的历史矩阵按本轮授权复用。手机真机、WeChat/Douyin资格、完整Goal0045/0046及通用生产发布均不在本轮关闭范围。**Goal0065在本固定候选及最终测试部署范围completed，Goal0045/0046状态不变。**
