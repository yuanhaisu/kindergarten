# RAM 策略与阿里云 CLI（方案 C）

方案 C = **小 MCP（只读）+ `aliyun` CLI（长尾）+ SSH MCP（服务器内操作）**。
目标不是让 MCP 接管一切，而是：常用查询走结构化工具，低频/写操作走 CLI，服务器内部走 SSH，发布仍走 GitHub Actions。

## 文件

| 文件 | 用途 |
|---|---|
| `mcp-readonly-policy.json` | 只读策略，绑定给 `mcp-readonly` RAM 用户；MCP 使用这个 profile |
| `cli-operator-policy.json` | 写操作策略，绑定给 `mcp-operator` RAM 用户；CLI 使用这个 profile |

`mcp-operator` 用户需要**同时**绑定上面两个策略：只读策略 + 写操作策略。
写操作策略里有一条显式 `Deny`，会覆盖掉高危动作。

## 一次性配置

> **不需要用户组**。两个用户的权限集合不同（一个只读、一个读写），直接给用户绑策略最直观。只有以后要加 3 个以上共享同一权限的用户（或 CI 用户）时，才值得引入用户组；可选做法见本节末尾。

### 1. 创建 RAM 用户

RAM 控制台 → 身份管理 → 用户 → 创建用户。

- 登录名称：`mcp-readonly`
- 显示名称：`MCP ReadOnly`
- 访问方式：只勾选 **OpenAPI 调用访问**（编程访问），不要勾选控制台访问
- 创建后先不急着建 AK，等策略绑完再建

重复一次创建 `mcp-operator`（显示名称 `MCP Operator`）。

不要用主账号 AccessKey。

### 2. 创建自定义策略

RAM 控制台 → 权限管理 → 权限策略 → 创建策略。

- 策略名称：`mcp-readonly-policy`
- 备注：`MCP read-only for kindergarten`
- 配置模式：脚本编辑（JSON）
- 粘贴 `mcp-readonly-policy.json`，替换/删除占位符
- 点“校验语法”确认无报错，再创建

重复一次创建 `cli-operator-policy`。

粘贴前先替换这些占位符。**如果某个资源还没创建，先把对应的整个 Statement 删掉**，等资源创建后再加回来；不要原样保存带 `<...>` 的策略，避免 RAM 校验或匹配出问题。

| 占位符 | 替换为 |
|---|---|
| `<bucket-prod>` / `<bucket-staging>` / `<bucket-dev>` | OSS 桶名 |
| `<rds-instance-id>` | `rm-` 开头的 RDS 实例 ID |
| `<redis-instance-id>` | `r-` 开头的 Redis/Tair 实例 ID |
| `<sls-project>` | SLS Project 名 |
| `<acr-instance-id>` / `<acr-namespace>` | ACR 实例 ID 与命名空间 |
| `<domain>` | 你的域名，例如 `nurtureplanet.cn` |
| `<security-group-id>` | 两台 ECS 共用 `sg-7xvjc9ug5rtvox03hy6w`（已探明） |

### 3. 绑定策略

用户列表 → 点用户名 → 权限管理 → 添加权限：

- `mcp-readonly` ← `mcp-readonly-policy`
- `mcp-operator` ← `mcp-readonly-policy` + `cli-operator-policy`

不要选 `AliyunCloudOpsMCPAccess`（Resource 全 `*`，含删机/重置密码/换系统盘）。

### 4. 创建 AccessKey 并配置 profile

每个用户创建 AccessKey。**不要把 AK 贴进聊天、仓库或 `.mcp.json`。**

推荐用仓库里的脚本，在**真实的 PowerShell 窗口**里跑：

```powershell
powershell -ExecutionPolicy Bypass -File scripts\set-aliyun-profile.ps1 -Profile mcp-readonly
powershell -ExecutionPolicy Bypass -File scripts\set-aliyun-profile.ps1 -Profile mcp-operator
```

脚本会提示输入 AccessKey ID 和 Secret（Secret 不回显），写入 `~/.aliyun/config.json`，并把文件 ACL 收紧到当前用户。

> 不要在 Claude Code 的 `!` 里跑 `aliyun configure`：`!` 走的是 Git Bash，CLI 的交互提示读不到输入，会存一个空 profile（实测会报 `AccessKeyId/AccessKeySecret is empty`）。也不要在 `!` 里直接写 `D:\tools\...`，反斜杠会被 Bash 当成转义符。

配置完成后，如果希望 CLI 默认使用只读身份，再跑一次：

```powershell
aliyun configure switch --profile mcp-readonly
```

也可以手工写 `C:\Users\<你>\.aliyun\config.json`（经典格式，CLI 与 MCP SDK 都认）：

```json
{
  "current": "mcp-readonly",
  "profiles": [
    {
      "name": "mcp-readonly",
      "mode": "AK",
      "access_key_id": "LTAI...",
      "access_key_secret": "...",
      "region_id": "cn-guangzhou",
      "output_format": "json"
    },
    {
      "name": "mcp-operator",
      "mode": "AK",
      "access_key_id": "LTAI...",
      "access_key_secret": "...",
      "region_id": "cn-guangzhou",
      "output_format": "json"
    }
  ]
}
```

Windows ACL 收紧到当前用户（可选但建议）：

```powershell
icacls "$env:USERPROFILE\.aliyun\config.json" /inheritance:r /grant:r "$env:USERNAME:(F)"
```

### 5. 验证

```powershell
aliyun sts GetCallerIdentity --profile mcp-readonly
aliyun ecs DescribeInstances --RegionId cn-guangzhou `
  --InstanceIds '["i-7xv06thh5tsvzwnigrjz","i-7xvbga12t596lzkbjt4r"]' `
  --profile mcp-readonly
```

写权限可用一次可逆的 Tag/Untag 验证（不要用删机、重启之类的动作做测试）：

```powershell
# 1) 打标签
aliyun ecs TagResources --RegionId cn-guangzhou --ResourceType instance `
  --ResourceId.1 i-7xv06thh5tsvzwnigrjz --Tag.1.Key MCPVerify --Tag.1.Value 1 `
  --profile mcp-operator

# 2) 用只读身份确认标签存在（应返回 TotalCount: 1）
aliyun ecs DescribeInstances --RegionId cn-guangzhou `
  --Tag.1.Key MCPVerify --Tag.1.Value 1 --profile mcp-readonly

# 3) 删标签
aliyun ecs UntagResources --RegionId cn-guangzhou --ResourceType instance `
  --ResourceId.1 i-7xv06thh5tsvzwnigrjz --TagKey.1 MCPVerify `
  --profile mcp-operator

# 4) 再次确认（应返回 TotalCount: 0）
aliyun ecs DescribeInstances --RegionId cn-guangzhou `
  --Tag.1.Key MCPVerify --Tag.1.Value 1 --profile mcp-readonly
```

MCP 侧已配置 `ALIBABA_CLOUD_PROFILE=mcp-readonly`，重启 Claude Code 后即可用。

### 6. 可选：用用户组管理

如果你以后要加更多 MCP/CI 用户，可以改成用户组：

1. 创建用户组 `mcp-readonly-group`，绑定 `mcp-readonly-policy`，把 `mcp-readonly` 加进去。
2. 创建用户组 `mcp-operator-group`，绑定 `mcp-readonly-policy` + `cli-operator-policy`，把 `mcp-operator` 加进去。
3. 用户不再直接绑策略。

用户最终权限 = 用户直接策略 ∪ 所属用户组策略；显式 `Deny` 优先于 `Allow`。

## 工具边界（重要）

### MCP 保持 8 个工具，不要再加动态工具

实测（本机 2026-11-14）：

| 工具数 | 启动耗时 |
|---|---|
| 8（当前） | 15.5 秒 |
| 10（+2 个动态工具） | 29.0 秒 |

Claude Code 的 MCP 握手默认 30 秒。**加到 10 个就贴着超时线，再加一个必挂。**
所以安全组、VPC、Redis、SLS、ACR、账单、ActionTrail 全部走 `aliyun` CLI，不要塞进 MCP。

改完 MCP 配置先跑 `python scripts/verify-aliyun-mcp.py`，确认启动 < 30 秒且工具数符合预期。

### 走 CLI 的典型操作

```powershell
# 安全组
aliyun ecs DescribeSecurityGroups --RegionId cn-guangzhou --profile mcp-readonly

# VPC / vSwitch
aliyun vpc DescribeVpcs --RegionId cn-guangzhou --profile mcp-readonly
aliyun vpc DescribeVSwitches --RegionId cn-guangzhou --profile mcp-readonly

# Redis / Tair
aliyun r-kvstore DescribeInstances --RegionId cn-guangzhou --profile mcp-readonly

# SLS（Project 级 SQL；CLI 产品码是 sls，不是 log）
aliyun sls GET /logs --project <sls-project> `
  --query "SELECT * FROM <logstore> LIMIT 20" `
  --region cn-guangzhou --profile mcp-readonly

# 账单 / 审计
aliyun bssopenapi QueryAccountBill --BillingCycle 2026-11 --profile mcp-readonly
aliyun actiontrail LookupEvents --RegionId cn-guangzhou --profile mcp-readonly
```

> CLI 的 `--query` 是 API 参数（例如 SLS 的 SQL）；过滤输出用 `--cli-query`（JMESPath）。

### 走 SSH MCP 的操作

容器日志、`docker compose`、`nginx -t`、文件查看、服务重启，全部用已配置的 `execute-command`（`connectionName=np-test` / `np-prd`）。
不要再为这些操作开 `OOS_RunCommand`，避免同一能力有两套凭据。

### 不要给 MCP 的权限

`ecs:DeleteInstances`、`ecs:ResetPassword`、`ecs:ReplaceSystemDisk`、`rds:DeleteDBInstance`、`kvstore:FlushInstance`、`oss:DeleteBucket`、`oss:PutBucketAcl`、`oss:PutBucketPolicy`、`ram:*`、`log:Delete*`、`cr:Delete*`、`vpc:Delete*`。

## 已知坑

- **不要直接用 `AliyunCloudOpsMCPAccess`**：Resource 全是 `*`，且包含删机、重置密码、换系统盘。
- **不要用 PyPI 的 `aliyuncli`**：2017 年的旧包，依赖老 SDK，不适合 Python 3.12。用官方 Go CLI。
- **CLI 位置**：`D:\tools\aliyun-cli\aliyun.exe`（v3.5.3）；Bash 入口 `~/bin/aliyun`，PowerShell 入口 `~\.local\bin\aliyun.cmd`。
- **动态工具一个都不能写错**：`sls:GetLogs`、`sls:GetHistograms`、`ecs:DescribeDisks`、`ecs:DescribeSnapshots`、`vpc:DescribeEipAddresses`、`cms:PutResourceMetricRule` 都会让 MCP 启动崩溃。改 `--visible-tools` 前先跑一次启动验证。
- **OOS CodeDeploy 不适合本项目**：`region_id_oss` 写死 `cn-hangzhou`，广州实例会跨地域下载产物；且绕开现有 GitHub Actions 发布链路。
- **发布不交给 MCP**：仍走 tag → GitHub Actions → production 人工批准。
