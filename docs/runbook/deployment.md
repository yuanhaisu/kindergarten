# 部署 Runbook

配套设计：[部署架构](../deployment-architecture.md)；决策：[ADR-0007 ~ ADR-0010](../adr/)。

## 0. 约定与前置

| 项 | 值 |
|---|---|
| 生产域名 | `api.<domain>`、`admin.<domain>`、`www.<domain>`（`<domain>` 301 → `www`） |
| 预发域名 | `test-api.<domain>`、`test-admin.<domain>`、`test-www.<domain>` |
| 生产 API 端口 | ECS 本机 `127.0.0.1:8080` |
| 预发 API 端口 | ECS 本机 `127.0.0.1:8081` |
| 目录 | `/opt/kindergarten/{prod,staging}` |
| 配置 | `/etc/kindergarten/{prod,staging}/app.ini`（0600） |
| 部署密钥 | `/etc/kindergarten/{prod,staging}/deploy.env`（0600，含 `MIGRATE_MYSQL_DSN`） |

前置条件（缺一不可）：

1. 域名已完成 ICP 备案，DNS 指向 ECS EIP；`www` / `test-www` 也要有 A 记录。
2. 证书 SAN 覆盖 `api` / `admin` / `test-api` / `test-admin` / `www` / `test-www`（以及 apex）；少一个主机名会让对应的 server block 握手失败。
3. 两个小程序后台已配置 request / upload / download 合法域名（`api` 与 `test-api`）。
4. GitHub Environments `staging`、`production` 已配置 Secrets；`production` 已配置 required reviewer。
5. ECS 已安装 Docker + Compose 插件、Nginx、curl；安全组仅开 80/443，SSH 限管理员 IP。
6. RDS 已创建 `kg_app`（DML）与 `kg_migrate`（DDL）账号；`kg_migrate` 凭据只放在 `deploy.env` 与 GitHub Secrets。

## 1. 首次部署

```bash
# 1) 目录
sudo mkdir -p /opt/kindergarten/{prod,staging}/{admin,website}/releases
sudo mkdir -p /etc/kindergarten/{prod,staging}
sudo chown -R deploy:deploy /opt/kindergarten /etc/kindergarten

# 2) 配置：从模板渲染，填入真实密钥
sudo -u deploy cp deploy/config/app.ini.tmpl /tmp/app.ini
sudo -u deploy envsubst < /tmp/app.ini > /etc/kindergarten/prod/app.ini
sudo chmod 600 /etc/kindergarten/prod/app.ini
# deploy.env 至少包含：
#   MIGRATE_MYSQL_DSN=kg_migrate:***@tcp(<rds-host>:3306)/kindergarten?...
#   RDS_INSTANCE_ID=rm-***
sudo chmod 600 /etc/kindergarten/prod/deploy.env

# 3) Docker Compose
cd /opt/kindergarten/prod
# 放入 docker-compose.prod.yml 与 .env（ACR_REGISTRY / ACR_NAMESPACE / IMAGE_TAG / MIGRATE_MYSQL_DSN）

# 4) Nginx
sudo envsubst '$DOMAIN' < deploy/nginx/kindergarten.conf.tmpl > /etc/nginx/conf.d/kindergarten.conf
sudo nginx -t && sudo systemctl reload nginx

# 5) 首次迁移 + 基础种子
cd /opt/kindergarten/prod
docker compose --profile migrate run --rm migrate
# 基础数据（幂等；使用 app.ini 里的 DML 账号）
docker compose run --rm --entrypoint /app/kindergarten-seedbase api
```

> 镜像内置三个二进制：`kindergarten-server`、`kindergarten-migrate`、`kindergarten-seedbase`。
> **不要**用应用启动代替迁移或种子。

## 2. 日常发布

1. 合并代码，打 tag：`git tag v2026.11.04-1 && git push origin v2026.11.04-1`。
2. GitHub Actions `release`：
   - 构建后端镜像并推 ACR；
   - 构建 admin 与 website 静态包；
   - 自动部署 staging；
   - 等待 `production` 环境人工批准；
   - 生产发布前调用 `snapshot-rds.sh`；
   - 执行迁移、切换 admin/website symlink、`/ready` 探测。
3. 生产验证：
   ```bash
   curl -fsS https://api.<domain>/ready
   curl -fsS https://api.<domain>/health
   curl -fsSI https://www.<domain>/ | head -1
   # 管理端登录、相册/成长页图片可加载
   # 官网：首屏截图、/privacy、/terms 均可打开
   ```
4. 小程序：在微信后台用测试包验收 staging，正式包走审核后发布。

## 2.1 隐私政策同意门禁上线（一次性）

1. 执行 `00005_user_consents.sql`（`make migrate`），再用 `go run ./cmd/dbcheck user_consents` 核对唯一键。
2. 部署服务端，保持 `app.ini` 的 `[consent] enforce = false`；确认登录响应带 `consent_required` / `current_policy_version`。
3. 发布家长端、老师端、管理端带同意阻断页的版本。
4. 观察一周，确认旧版小程序已退出后，把 `enforce` 改为 `true` 并重启 API。
5. 回填历史日志：先 `go run ./cmd/redactlogs -dry-run` 核对行数，确认后再 `-dry-run=false` 执行；命令可重复执行。生产执行前先做一次 RDS 快照。
6. 回滚：把 `enforce` 改回 `false` 并回滚镜像；`user_consents` 表保留，不删数据。

> 顺序不能反：先开 `enforce=true` 再发客户端，会让未升级的线上用户全部收到 403（违反 ADR-0009 的向后兼容约束）。

## 3. 回滚

| 层 | 操作 |
|---|---|
| 后端 | 把 `IMAGE_TAG` 改为上一个 tag，`docker compose up -d api`；或运行 `deploy/scripts/rollback.sh <env> <previous-tag>` |
| admin | `ln -sfn /opt/kindergarten/<env>/admin/releases/<previous-tag> /opt/kindergarten/<env>/admin/current` |
| website | `ln -sfn /opt/kindergarten/<env>/website/releases/<previous-tag> /opt/kindergarten/<env>/website/current` |
| 数据库 | 迁移只向前；破坏性变更用发布前 RDS 快照恢复到新实例，再切 DSN |
| 小程序 | 微信后台回退版本（受平台能力限制） |

回滚后必须再探 `/ready`，并检查 SLS 中的 5xx。

## 4. 备份与恢复

- RDS：每日自动备份 + binlog，保留 14 天；发布前手动快照。
- 每周逻辑备份：`mysqldump --single-transaction --routines --triggers` 到 OSS，保留 8 周。
- 恢复演练（每季度）：
  1. 恢复快照到临时实例；
  2. `MYSQL_DSN=<临时实例> go run ./cmd/dbcheck files` 等关键表；
  3. 记录实际 RTO；
  4. 确认后销毁临时实例。

## 5. 告警处置

| 告警 | 首先看 | 处理 |
|---|---|---|
| `/ready` 失败 | `docker compose ps`、容器日志、RDS/Redis 连接 | 依赖恢复后自动转 ready；持续失败按回滚处理 |
| ECS CPU/内存高 | `docker stats` | 确认是否 staging MySQL 争内存；必要时迁出 staging DB 或升配 |
| RDS 连接数高 | RDS 监控、慢查询 | 检查连接池/慢查询；不要盲目重启应用 |
| 图片 403 | OSS 桶 ACL、签名过期 | 确认桶为私有、响应 URL 带 `Signature`；客户端应重取数据 |
| 发布后 5xx | SLS 按 `request_id` 检索 | 先回滚镜像，再定位 |

## 6. 证书续期

- 使用 acme.sh + Let's Encrypt；续期 cron 由 acme.sh 安装。
- 续期后 reload：
  ```bash
  acme.sh --install-cert -d <domain> \
    --key-file /etc/nginx/ssl/<domain>/privkey.pem \
    --fullchain-file /etc/nginx/ssl/<domain>/fullchain.pem \
    --reloadcmd "systemctl reload nginx"
  ```
- 检查到期：`acme.sh --list`。

## 7. 常用命令

```bash
# 查看服务
cd /opt/kindergarten/prod && docker compose ps
docker compose logs -f --tail=200 api

# 迁移状态 / 执行
docker compose --profile migrate run --rm --entrypoint /app/kindergarten-migrate migrate status
docker compose --profile migrate run --rm migrate   # 默认 entrypoint 是 up

# 只读核对表结构
cd /path/to/repo/server
MYSQL_DSN='<app-dsn>' go run ./cmd/dbcheck task_check_ins
MYSQL_DSN='<app-dsn>' go run ./cmd/dbcheck --query "SELECT COUNT(*) FROM files"

# Nginx
sudo nginx -t && sudo systemctl reload nginx
```

## 8. 常见故障

| 现象 | 排查 |
|---|---|
| `migrate` 报 `Unknown database` | 检查 DSN 的库名；RDS 白名单是否包含 ECS 私网 IP |
| `migrate` 报 DDL 权限不足 | `kg_migrate` 是否被授予该库的 DDL；`deploy.env` 是否被正确 source |
| 应用启动即退出 | `docker compose logs api`；`configs/` 是否打进镜像、`CONFIG_PATH` 是否挂载 |
| `/ready` 503 但 `/health` 200 | MySQL/Redis 不可达；检查 VPC/白名单/密码 |
| 管理端刷新 404 | Nginx 缺少 `try_files $uri $uri/ /index.html` |
| 图片全部裂开 | 桶是否私有但响应未签名；检查响应层 `SignRef` 与 OSS endpoint |
| 小程序请求合法域名报错 | 两个小程序后台分别配置 `https://api` / `https://test-api` |
| GitHub Actions 推 ACR 超时 | 重试；持续失败则评估云效 Flow（见 Q11 备选） |

## 9. 安全清单（每次上线前）

- [ ] 没有新增明文凭据；`node scripts/check-secrets.mjs` 通过
- [ ] 生产 `app.ini` 0600，属主 deploy
- [ ] 安全组仅 80/443；SSH 限管理员 IP
- [ ] RDS/Redis 无公网地址
- [ ] `cors_origins` 只含 `https://admin.<domain>`
- [ ] 发布前快照已成功
- [ ] 回滚 tag / admin symlink 已知
- [ ] MCP 使用专用 RAM 用户，AK 未进入任何被跟踪文件

## 10. MCP 运维凭据（alibaba-cloud-ops-mcp-server）

阿里云运维 MCP 配置在**用户级**（`~/.claude.json`，通过 `claude mcp add --scope user` 写入），**不进入仓库**：项目 `.mcp.json` 是 git 跟踪文件，且被 `scripts/check-secrets.mjs` 的 `sensitive-file` 规则整文件拦截。版本固定在 `0.9.27`，`--visible-tools` 只开放只读工具（ECS/VPC/RDS 查询、OSS 列举、CMS 指标）。**配置本身不存任何 AK。**

### 10.1 凭据来源

该 MCP 使用阿里云 SDK 默认凭据链，按顺序取：

1. 环境变量 `ALIBABA_CLOUD_ACCESS_KEY_ID` / `ALIBABA_CLOUD_ACCESS_KEY_SECRET`（STS 临时凭据再加 `ALIBABA_CLOUD_SECURITY_TOKEN`）
2. `ALIBABA_CLOUD_CREDENTIALS_URI`
3. `ALIBABA_CLOUD_PROFILE`（对应 `~/.aliyun/config.json`）
4. ECS 实例 RAM 角色（`ALIBABA_CLOUD_ECS_METADATA=<角色名>`）

### 10.2 本机（Claude Code）配置

1. 创建专用 RAM 用户（如 `mcp-ops-readonly`）：仅编程访问，不开控制台登录；只授只读策略，写操作按需单独加并限定资源 ARN。
2. 在用户级注册 MCP（只需执行一次，配置写入 `~/.claude.json`，不进仓库）：
   ```bash
   claude mcp add --scope user alibaba-cloud-ops-mcp-server -- \
     uvx alibaba-cloud-ops-mcp-server@0.9.27 \
     --services ecs,rds,oss,vpc \
     --visible-tools "ECS_DescribeInstances,ECS_DescribeRegions,ECS_DescribeZones,ECS_DescribeImages,ECS_DescribeSecurityGroups,VPC_DescribeVpcs,VPC_DescribeVSwitches,RDS_DescribeDBInstances,OSS_ListBuckets,OSS_ListObjects,CMS_GetCpuUsageData,CMS_GetCpuLoadavgData,CMS_GetMemUsageData,CMS_GetDiskUsageData"
   ```
3. AK 写入 **Windows 用户环境变量**，不要写进项目文件：
   ```powershell
   setx ALIBABA_CLOUD_ACCESS_KEY_ID "LTAI..."
   setx ALIBABA_CLOUD_ACCESS_KEY_SECRET "..."
   ```
4. 重启终端与 Claude Code（`setx` 只对新进程生效），在 Claude Code 里用 `/mcp` 确认服务器已连接。
5. 验证：调用 `ECS_DescribeInstances`，能返回 `cn-guangzhou` 的实例列表即通。

> ⚠️ 项目 `.mcp.json` 是 git 跟踪文件，且被 secret 扫描器整文件拦截；本 MCP 使用用户级配置，**不要改项目 `.mcp.json`**。**禁止**把真实 AK 写入 `.mcp.json` 或任何被跟踪文件。`scripts/check-secrets.mjs` 的 `LTAI` 规则只扫变更文件，不能当兜底。
>
> 需要写操作时，必须同时改两处：`--visible-tools` 白名单与 RAM 策略。只改一处等于给自己留后门。

### 10.3 在 ECS 上运行（可选）

- 优先给 ECS 绑定实例 RAM 角色，设 `ALIBABA_CLOUD_ECS_METADATA=<角色名>`，**不落长期 AK**。
- 只监听 `127.0.0.1`，本机通过 SSH 隧道访问；禁止 `--host 0.0.0.0`。
- 用专用低权限系统用户运行，不要 root，不要加入 docker 组。
- `LOCAL_RunShellScript` / `OOS_RunCommand` 默认不得加入白名单——它们等于在 MCP 宿主上执行任意命令。

### 10.4 轮换与审计

| 项 | 要求 |
|---|---|
| 轮换周期 | 90 天；换机或怀疑泄漏立即停用 |
| 轮换步骤 | 新建 AK → 更新环境变量并重启 → 验证 → 停用旧 AK → 观察 ActionTrail 一周后删除 |
| 审计 | 开启 ActionTrail；RAM 控制台核对 AK「最后使用时间」 |
| 环境隔离 | 本机 MCP / staging / production 使用不同 RAM 用户，不复用 AK |
| 生产边界 | MCP 不参与生产发布；发布仍走 tag → GitHub Actions → 人工批准 |
