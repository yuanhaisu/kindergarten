# extendtrials 演练记录（2026-11-16）

本记录对应工单 #81，规格 #62，命令 `server/cmd/extendtrials`（工单 #65/#66）。
演练**只在开发库的一次性只读副本上执行**，未在预发或生产库写入任何数据。

## 1. 结论

| 项目 | 结果 |
| --- | --- |
| dry-run | 通过：6 所园 / 24 条授权命中，清单与 SHA256 齐全，数据库无变化 |
| apply | 通过：24 条授权全部变为 `2027-01-31 23:59:59`（Asia/Shanghai） |
| 抽检 | 通过：6 所园各 4 条，`MIN = MAX = 2027-01-31 23:59:59`；另按授权 ID 抽检 3 条均一致 |
| revert | 通过：24 条授权恢复为清单原始到期日（秒级） |
| 再次 apply | 通过：24 条重新延长 |
| 幂等重跑 | 通过：影响 0 所园 / 0 条授权，24 条计入「未变更」 |
| 清单外数据 | 未变化：8 条 trial 授权（被排除园所 4 条 + 无对应园所的历史孤儿行 4 条）及订阅授权保持原值 |

## 2. 环境与版本

| 项目 | 值 |
| --- | --- |
| 执行日期 | 2026-11-16（Asia/Shanghai） |
| 执行人 | `rehearsal:t81`（演练占位，非真实运营人员） |
| 目标库 | `kindergarten_extendtrials_rehearsal`（开发库 `kindergarten` 的 mysqldump 副本） |
| 库快照 | 101 张表；7 所未删除园所；42 条授权 |
| 命令源码 | commit `d1798b4bc6e9a57e03b965d52c500c24c898cdea`（#66），合并后 HEAD `67b8a26ab1d584f16f0a8d24b5c9523508dcb38b` |
| 工具链 | Go 1.25.0 windows/amd64 |
| 连接方式 | `CONFIG_PATH=configs/app.ini` + `MYSQL_DSN` 指向副本库（连接串不归档） |

副本制作与销毁：

```text
mysqldump --single-transaction --routines --triggers --set-gtid-purged=OFF kindergarten
  | mysql kindergarten_extendtrials_rehearsal
# 演练与证据采集完成后 DROP DATABASE kindergarten_extendtrials_rehearsal
```

开发库 `kindergarten` 全程只被读取（dump），命令本身未连过开发库。

## 3. 演练范围（manifest）

- 扫描 7 所园所；命中 **6 所 / 24 条** trial 授权；排除 1 所（存在未取消订单，园所 ID `998113`）；`created_at` 为空 0 所。
- 原到期日区间：`2026-12-06 14:25:56.746` ～ `2026-12-09 16:09:58.118`。
- manifest SHA256：`dcfdf81ddc8f436ec055b53fc326247f0f89735d41f2d8804bf1d7372eaf3811`（24 行逐条授权）。

| 园所 ID | trial 条数 | 原到期日 |
| --- | --- | --- |
| 4 | 4 | 2026-12-06 14:25:56 |
| 990001 | 4 | 2026-12-06 14:25:56 |
| 990002 | 4 | 2026-12-06 14:25:56 |
| 990101 | 4 | 2026-12-06 14:25:56 |
| 990201 | 4 | 2026-12-06 14:25:56 |
| 998143 | 4 | 2026-12-09 16:09:58 |
| **合计** | **24** | 2026-12-06 ～ 2026-12-09 |

未纳入延长的 trial 授权共 8 条：4 条属于被排除园所 `998113`；4 条所属园所 ID `990301` 在 `kindergartens` 表中不存在（历史孤儿行，不在园所扫描集合内）。

> 本 manifest 只对应演练副本，**不得复用于真实环境**。预发/生产必须在目标库重新 dry-run 生成自己的清单。

## 4. 执行步骤与结果

命令均在 `server/` 下执行；`EXTENDTRIALS_ALLOW=1` 仅写操作时设置。

### 4.1 dry-run

```text
go run ./cmd/extendtrials -manifest ./extendtrials-2026-11-16.csv
```

输出：命中 6 所 / 24 条；生成 manifest、`.sha256`、`.summary.csv`（汇总含园所名称，按要求不归档进仓库）。
dry-run 前后到期日与「目标日行数」一致（32 条 trial 行仍为原值，目标日行数 0），未写库。

### 4.2 apply

```text
EXTENDTRIALS_ALLOW=1 go run ./cmd/extendtrials \
  -manifest ./extendtrials-2026-11-16.csv -apply -operator "rehearsal:t81"
```

结果：影响 **6 所园 / 24 条授权**；`platform_operation_logs#1752`
（`module=billing, action=extend_trials, target_type=entitlement_batch`）。

抽检（全量 + 指定授权 ID）：

| 授权 ID | 园所 ID | apply 后到期日 |
| --- | --- | --- |
| 12 | 990001 | 2027-01-31 23:59:59 |
| 13 | 4 | 2027-01-31 23:59:59 |
| 138 | 998143 | 2027-01-31 23:59:59 |

6 所园 24 条授权的 `MIN(expires_at) = MAX(expires_at) = 2027-01-31 23:59:59`。
清单外 8 条 trial 授权与 2 条 `source='subscription'` 授权（园所 `998113`）保持原值。

### 4.3 revert

```text
EXTENDTRIALS_ALLOW=1 go run ./cmd/extendtrials \
  -manifest ./extendtrials-2026-11-16.csv -revert -operator "rehearsal:t81"
```

结果：影响 **6 所园 / 24 条授权**；`platform_operation_logs#1753`（`action=revert_trials`）。
24 条全部恢复为 manifest 原始值（秒级）：`2026-12-06 14:25:56` / `2026-12-09 16:09:58`；
目标日行数回到 0。

**精度说明**：副本原值带毫秒（`.746` / `.118`），manifest 按秒记录，revert 后为 `.000`。这是
`docs/runbook/deployment.md` §11.3 已声明的业务等价行为；若真实环境需要绝对精确恢复，
apply 前须另行导出带毫秒的原值留档。

### 4.4 再次 apply

结果：影响 **6 所园 / 24 条授权**；`platform_operation_logs#1754`。日期重新变为 `2027-01-31 23:59:59`。

### 4.5 幂等重跑（紧接着第二次 apply）

结果：影响 **0 所园 / 0 条授权**（「幂等重跑，无实际变更」），未变更 **24 条**；
`platform_operation_logs#1755` 的 `detail` 中 `rows=0, unchanged=24`。
最终库内 24 条目标日授权，8 条清单外 trial 授权保持原值。

另在最终状态重跑 dry-run：命中 0 所 / 0 条，未生成 manifest，库指纹前后一致
（`82e9929fb919461817603c2f9f5a548d`）。

## 5. 回滚路径

1. 使用**同一份** manifest 与 `.sha256` 侧车文件执行 `-revert`（需要 `EXTENDTRIALS_ALLOW=1` 与 `-operator`）。
2. 命令逐行校验：当前值等于目标日 → 恢复原值；已等于原值 → 跳过；其他值 → **整体拒绝**并点名授权 ID，
   不会覆盖 apply 之后的合法续期/调整。
3. 本次演练已验证「apply → revert → 再 apply」闭环；重复 revert 的 no-op 行为由命令测试覆盖
   （`go test ./cmd/extendtrials`），本次未重复执行。
4. 事务语义：apply/revert 均为单事务，任一行失败整体回滚；失败不会留下部分更新。

## 6. 风险与后续

- **真实环境未执行**：本演练使用开发库副本，数据规模与形状不等于预发/生产。预发/生产执行前仍需
  运营逐行确认 manifest 与汇总，并由运维确认维护窗口与备份。
- **孤儿授权**：`entitlements` 中可能存在所属园所已不存在的 trial 行（本次 4 条，园所 ID `990301`），
  命令按园所维度扫描不会纳入。真实库执行前建议先核对是否存在同类行，运营确认是否另行处理。
- **毫秒截断**：见 §4.3。真实执行若要求逐字节恢复，apply 前导出原值。
- **证据归档**：演练日志与 manifest 保留在执行人证据目录，清单 SHA256 已记录在本文件；
  副本库在证据采集后已删除。生产执行的 manifest、sha256、summary 与操作日志应按 §11 要求一并归档。
- **权限与凭据**：命令不读入凭据，DSN 来自配置或环境变量；本记录不包含 DSN、口令或园所名称。

## 7. 复现命令摘要

```bash
cd server
export CONFIG_PATH=configs/app.ini
export MYSQL_DSN='root:<password>@tcp(127.0.0.1:3316)/kindergarten_extendtrials_rehearsal?charset=utf8mb4&parseTime=True&loc=Local'

go run ./cmd/extendtrials -manifest ./extendtrials-2026-11-16.csv
EXTENDTRIALS_ALLOW=1 go run ./cmd/extendtrials -manifest ./extendtrials-2026-11-16.csv -apply  -operator "rehearsal:t81"
EXTENDTRIALS_ALLOW=1 go run ./cmd/extendtrials -manifest ./extendtrials-2026-11-16.csv -revert -operator "rehearsal:t81"
EXTENDTRIALS_ALLOW=1 go run ./cmd/extendtrials -manifest ./extendtrials-2026-11-16.csv -apply  -operator "rehearsal:t81"
EXTENDTRIALS_ALLOW=1 go run ./cmd/extendtrials -manifest ./extendtrials-2026-11-16.csv -apply  -operator "rehearsal:t81"
```
