# 测试与临时账号台账

> 本文件只记录**账号清单与操作方式**，不记录任何密码、令牌或密钥。
> 本仓库可能被公开（见 CLAUDE.md「已知问题」），明文凭据一律不得提交。

- 本文件（可提交）：`docs/runbooks/test-accounts.md`
- 口令等敏感信息（**不提交**）：`~/.kindergarten/test-accounts.local.md`
  - Windows：`%USERPROFILE%\.kindergarten\test-accounts.local.md`
- 服务器侧凭据位置（**不复制到本文件**）：`/etc/kindergarten/staging/app.ini`、`/etc/nginx/htpasswd/kindergarten-staging`

## 1. 环境入口

| 环境 | 平台控制台 | 园所后台 | API |
|---|---|---|---|
| staging | https://test-admin.nurtureplanet.cn/platform/login | https://test-admin.nurtureplanet.cn/login | https://test-api.nurtureplanet.cn/api/v1 |
| local | http://localhost:3000/platform/login | http://localhost:3000/login | http://127.0.0.1:8080/api/v1 |

## 2. 平台运营账号（`/platform/*`）

平台账号与园所账号是两套独立信任域，不能混用（ADR-0014）。

| 环境 | 用户名 | 显示名 | 状态 | 用途 / 备注 | 口令位置 |
|---|---|---|---|---|---|
| staging | `ops` | — | 启用 | 平台运营主账号 | 负责人保管 |
| staging | `stg_verify_ops` | Staging Verify | 启用 | 2026-11-19 平台控制台 TailAdmin 验收临时账号；验收完成后应停用 | 本地口令文件 |
| staging | `stg_ip_probe` | — | 停用 | 审计真实 IP / `request_id` 探针，已验证后停用 | 无（已停用） |
| local | `it_verify_ops` | — | 停用 | 本地平台控制台集成验证账号 | 本地口令文件 / 用命令重置 |

账号状态字段：`platform_admins.status`，`1=启用`、`0=停用`。停用或改密后，该账号已登录的会话会因 epoch 失效而立即退出。

### 创建 / 重置

首个平台账号（幂等：已存在时只重置口令）：

```bash
cd server
PLATFORM_ADMIN_USERNAME=ops \
PLATFORM_ADMIN_PASSWORD='<至少 8 位，含大小写与数字>' \
PLATFORM_ADMIN_NAME='运营负责人' \
CONFIG_PATH=configs/app.ini \
go run ./cmd/platformadmin
```

本地开发库：

```bash
CONFIG_PATH=configs/app.local.ini go run ./cmd/platformadmin   # 同样需要上面三个 PLATFORM_ADMIN_* 变量
```

### 停用 / 启用

优先用控制台：平台控制台 → 账号 → 停用 / 启用。

需要直接操作数据库时（表名与字段已按 `server/internal/domain/billing.go`）：

```sql
UPDATE platform_admins SET status = 0 WHERE username = 'stg_verify_ops';
```

> 停用不会删除审计记录；`platform_operation_logs` 仍保留操作人与时间。

## 3. 园所后台 / 小程序演示账号

| 端 | 账号 | 口令 | 说明 |
|---|---|---|---|
| 家长端 / 园所后台 | `13900010001`、`13900010002` | 见 CLAUDE.md「辅助命令 → `fixpwd`」 | 仅演示库；`go run ./cmd/fixpwd` 可重置为文档中的演示口令 |

园所租户管理员由 `setadmin` 供给（口令从 `-password` 或 `SETADMIN_PASSWORD` 读取，无默认口令）：

```bash
cd server
SETADMIN_PASSWORD='<口令>' go run ./cmd/setadmin -phone 13800000000 -kg 7
```

## 4. staging Nginx Basic Auth

| 站点 | 状态 | 说明 |
|---|---|---|
| `test-admin.nurtureplanet.cn` | 已关闭（2026-11-19） | 匿名可访问；临时 Basic Auth 用户 `stgcheck` 已从 htpasswd 删除，备份保留 |
| `test-www.nurtureplanet.cn` | 仍启用 | 官网测试入口继续使用 Basic Auth；账号在服务器 htpasswd 中，不在仓库 |

> `deploy/nginx/kindergarten.conf.tmpl` 仍含 staging 门禁模板；如果未来重新渲染部署，`test-admin` 的 Basic Auth 会回来，需要同步修改模板或再次关闭。

## 5. 临时账号使用纪律

1. 临时账号必须写清**用途 + 创建日期 + 计划停用时间**，用完当天停用。
2. 不把平台账号用于园所后台测试，反之亦然。
3. 不把口令写进 issue、PR、聊天截图、构建产物或本文件。
4. 口令只放本地 `test-accounts.local.md` 或密码管理器；服务器侧凭据只放服务器配置文件。
5. 发现凭据进入 git 历史：先轮换，再清理历史（见 CLAUDE.md「已知问题」）。

## 6. 变更记录

| 日期 | 变更 |
|---|---|
| 2026-11-20 | 建立台账；登记 staging `stg_verify_ops`、`stg_ip_probe` 与本地 `it_verify_ops`；记录 `test-admin` Basic Auth 已关闭 |
