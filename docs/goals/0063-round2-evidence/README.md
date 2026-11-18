# Goal0063 Round2 筛选证据

固定候选：`099af4eaa117628a6337841928cbbbc5f13697c1`。总体 **FAIL**；参见[完整报告](../0063-round2-independent-acceptance.md)。

- `no-landlord-faults.json`：原始夹具曾仅凭归档存在标PASS，语义审计已撤回；正确结果FAIL。
- `completed-faults.json`：终局live ACK损失导致归档缺失；修复中崩溃/接管与domain ACK损失的首启失败、二启恢复均保留。
- `cancel-cursor.json`：正常取消后的残留重投游标。
- `nominal-business-native.json`、`native-metrics.json`及9张PNG：真实业务、原生交互、字体/触摸与短程资源证据。未宣称三Cocos客户端全手或真机资格。
- CI/job状态、输入/产物清单、实际Go日志及退出码、调度资源和保护审计分别独立保存。
- `fixture-notes.md`记录原始探针失败与纠正，不将夹具失败冒充产品缺陷，也不以旧脚本总标签覆盖最终审计。

完整原始证据及外部夹具仍在本机私有 `/root/goal0063-round2-099af4e`；本目录不包含密码、JWT、私钥、服务配置、浏览器profile、原始socket或完整终局秘密payload。脚本/配置路径只是复现定位，不代表已把全部本机材料上传。

`sha256.json`校验本目录其他文件；它不包含自身。
