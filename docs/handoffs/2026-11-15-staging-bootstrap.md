# Staging 环境交接（2026-11-15）

> ⚠️ 已脱敏：公网/内网 IP、实例 ID、安全组/VPC 标识、SSH 用户名与密钥路径均替换为占位符；真实值见运维私密记录。
> 用途：记录 staging-only 环境的当前状态、关键路径和明天待办。prod 暂缓。
> 相关文档：`docs/deployment-architecture.md`、`docs/runbook/deployment.md`、`deploy/ram/README.md`。
> 历史提交：`b1d1e08`（部署架构 + runbook）、`5d7e272`（MCP 修正）、`b2678d7`（方案 C / RAM 策略）。

## 0. 结论

- staging 已可用：`test-api` ready，`test-admin` / `test-www` 200，Let's Encrypt 证书受信任。
- 仍未完成：微信 AppID/Secret、staging 平台管理员、Basic Auth、安全组 22 限制、A 续费、小程序测试包域名。
- prod 未创建：没有 RDS、托管 Redis、EIP、ACR、`api` / `admin` 生产域名。

## 1. 服务器与网络

| 项 | A（staging） | B（idle） |
|---|---|---|
| 实例 | `<ecs-instance-id-staging>` / `np-staging` | `<ecs-instance-id-idle>` / `np-idle` |
| 规格 | 2C4G / 40G ESSD | 2C4G / 40G ESSD |
| 系统 | Alibaba Cloud Linux 4 LTS 容器优化版 | CentOS 7.9（EOL，未重装） |
| 公网 | `<staging-public-ip>`（固定公网 IP，非 EIP） | `<idle-public-ip>`（固定公网 IP） |
| 内网 | `<staging-private-ip>` | `<idle-private-ip>` |
| 带宽 | 1 Mbps 出方向，固定带宽 | 1 Mbps 出方向 |
| 到期 | **2026-12-14** | 2027-07-31 |
| 安全组 | `sg-np-staging` = `<security-group-staging>` | `sg-np-idle` = `<security-group-idle>` |

- VPC `<vpc-id>` / vSwitch `<vswitch-id>` / 可用区 `cn-guangzhou-b`。
- 旧共享安全组 `<security-group-legacy>` 已从两台实例移出，暂未删除。
- SSH：`<ssh-user>@<staging-public-ip>`，密钥路径由运维保管（不入库）。
- Git Bash helper：`<local-bin>/np-prd.sh`、`np-test.sh`；打开命令：
  `Start-Process -FilePath "<Git Bash 路径>" -ArgumentList '-li','-c','<local-bin>/np-prd.sh'`

## 2. 已完成

### 2.1 基础软件

- Docker `24.0.9` + Compose `2.26.1`（来自 `alinux4-updates`），已 enable + active。
- Nginx `1.30.4`，`nginx -t` 通过，已 enable + active。
- acme.sh `v3.1.5`（从 Gitee 镜像安装）、jq、git、rsync、httpd-tools。
- **Docker Hub 从 A 不可达**（`registry-1.docker.io` 超时）。镜像通过本地 `docker save | gzip | ssh ... docker load` 传输；以后更新镜像仍需本地构建/传输，或配镜像加速/ACR。

### 2.2 数据层

- `kindergarten-mysql-staging`（mysql:8.0，1g / 0.8 CPU）与 `kindergarten-redis-staging`（redis:7-alpine，256m / 0.3 CPU），均 `Up (healthy)`。
- `migrate` 已应用 5 个 goose 版本，schema 101 张表。
- `seedbase` 已完成：2 个字典类型 / 14 项，42 条 `admin_permissions`；当前没有园所和租户管理员（`admin_roles` 为 0，符合预期）。
- 资源上限在 `/opt/kindergarten/staging/docker-compose.local.yml`；MySQL `innodb-buffer-pool-size=256M`、`performance-schema=OFF`、`max-connections=100`，Redis `maxmemory 128mb` + `allkeys-lru`。

### 2.3 API

- `kindergarten-api-staging` 运行中，宿主 `127.0.0.1:8081`。
- `https://test-api.nurtureplanet.cn/health` 与 `/ready` 均通过。
- **尚不能用于微信登录**：`app.ini` 的 teacher/parent AppID/Secret 仍是 `CHANGE_ME`。

### 2.4 DNS 与证书

- `test-api` / `test-admin` / `test-www` → `<staging-public-ip>`，TTL 600，解析线路 **默认**。
- ⚠️ 解析线路必须是「默认」或额外加「境外」线路；如果放在电信/联通等具体线路，Let's Encrypt 的境外解析器会看到 NXDOMAIN。
- 证书：Let's Encrypt（issuer `CN=YR1`），SAN 覆盖三个 test 域名，`notBefore` 2026-11-14 16:55 UTC，`notAfter` **2027-02-12 16:55 UTC**。
- 证书路径：`/etc/nginx/ssl/nurtureplanet.cn/fullchain.pem` + `privkey.pem`。
- acme.sh cron：`7 0,6,12,18 * * *`；`ACME_USE_IPV4_REQUESTS='1'` 已持久化（A 的 IPv6 出口不通，必须走 IPv4）。

### 2.5 静态包

- Release：`staging-20261115-1`。
- Admin：`/opt/kindergarten/staging/admin/releases/staging-20261115-1`（251 文件，3.0M）。
- Website：`/opt/kindergarten/staging/website/releases/staging-20261115-1`（42 文件，2.7M）。
- `current` 符号链接已指向该 release；`test-admin` 与 `test-www` 均返回 200。

### 2.6 关键路径与权限

| 路径 | 权限 |
|---|---|
| `/opt/kindergarten/staging/docker-compose.yml` | 644 |
| `/opt/kindergarten/staging/docker-compose.local.yml` | 644 |
| `/opt/kindergarten/staging/.env` | 600 |
| `/etc/kindergarten/staging/app.ini` | 600 |
| `/etc/kindergarten/staging/app.ini.tmpl` | 644 |
| `/etc/kindergarten/staging/deploy.env` | 600 |
| `/etc/nginx/conf.d/kindergarten-staging.conf` | 644 |
| `/etc/nginx/ssl/nurtureplanet.cn/fullchain.pem` | 644 |
| `/etc/nginx/ssl/nurtureplanet.cn/privkey.pem` | 600 |
| `/var/www/acme/` | 755 |

- `app.ini` 的 OSS 字段已填（bucket `<bucket-staging>`，endpoint `<oss-internal-endpoint>`）；`[face]` 的 `secret_id` / `secret_key` 仍是 `CHANGE_ME`，因 `enabled = false` 无害。
- `deploy.env` 的 `MIGRATE_MYSQL_DSN` 仍是 `CHANGE_ME`；本次迁移用的是 `.env` 里的 root DSN，接 CI 前要换成独立迁移账号。

## 3. 明天待办（按顺序）

1. **填微信字段**：`/etc/kindergarten/staging/app.ini` 的 `[wechat]` 四个字段（teacher/parent AppID/Secret）。不要发聊天，直接在服务器改。
2. **更新小程序测试域名**：`teacher/.env.test`、`parent/.env.test` 目前仍是 `https://test-api.xxx.com/...`，改成 `https://test-api.nurtureplanet.cn/api/v1` 与对应 upload URL，再 `npm run build:mp-weixin:test`。
3. **建 staging 平台管理员**：
   - 本地构建 `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o /tmp/platformadmin ./cmd/platformadmin`；
   - scp 到 A，运行：`CONFIG_PATH=/etc/kindergarten/staging/app.ini PLATFORM_ADMIN_USERNAME=ops PLATFORM_ADMIN_PASSWORD='<强密码>' PLATFORM_ADMIN_NAME='Ops' /root/platformadmin`；
   - 登录 `https://test-admin.nurtureplanet.cn/platform/login`，再创建测试园所与租户管理员。
   - 也可绕过后台，直接在服务器上供给租户管理员（园所已存在时；幂等，重复执行只更新口令与绑定）：
     ```bash
     # 本地构建：GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o /tmp/setadmin ./cmd/setadmin
     CONFIG_PATH=/etc/kindergarten/staging/app.ini SETADMIN_PASSWORD='<强密码>' \
       /root/setadmin -phone <手机号> -kg <园所ID>
     ```
     `-phone` 与 `-kg` 必填；园所不存在或已软删除会直接失败；口令只从 `SETADMIN_PASSWORD` 或 `-password` 读取，不会回显。
4. **Basic Auth**：给 `test-admin` / `test-www` 加 `auth_basic`（htpasswd），避免公开暴露；`test-api` 保持公网，供微信测试包访问。
5. **安全组 22**：`sg-np-staging` 与 `sg-np-idle` 当前仍是 `22 → 0.0.0.0/0`，改成 `<你的固定公网 IP>/32`（或你的当前公网 IP）。
6. **A 续费**：2026-12-14 到期，建议 12 月初前续 1 年并开自动续费。
7. **微信后台域名白名单**：两个小程序分别配置 `https://test-api.nurtureplanet.cn` 的 request / upload / download 合法域名，再上传测试包。
8. **媒体上传验证**：用测试包上传一张图片，确认对象落在 `<bucket-staging>`，响应 URL 是广州内网签名的 1 小时 URL。
9. **后续（prod 前）**：RDS / 托管 Redis / EIP / ACR / `api` / `admin` 生产域名；修 `deploy.sh` 先切静态后启 api、`snapshot-rds.sh` 假成功、`deploy.env` DSN 引号等问题。

## 4. 常用命令

```bash
# SSH
ssh -i <ssh-key-path> <ssh-user>@<staging-public-ip>

# 容器状态
cd /opt/kindergarten/staging
docker compose -f docker-compose.yml -f docker-compose.local.yml ps

# 启动 / 重启 API
docker compose -f docker-compose.yml -f docker-compose.local.yml up -d api
docker logs -f --tail=100 kindergarten-api-staging

# API 健康
curl -s https://test-api.nurtureplanet.cn/ready
curl -s https://test-api.nurtureplanet.cn/health

# 证书
/root/.acme.sh/acme.sh --list
/root/.acme.sh/acme.sh --cron --home /root/.acme.sh

# Nginx
nginx -t && systemctl reload nginx
```

## 5. 已知坑

- **IPv6 出口不通**：acme.sh 必须用 `--request-v4`；`ACME_USE_IPV4_REQUESTS='1'` 已写入 `account.conf`，不要删。
- **DNS 线路**：Let's Encrypt 的境外解析器只看「默认」或「境外」线路；具体运营商线路会导致 NXDOMAIN。
- **Docker Hub 不可达**：A 上无法直接 pull；镜像需本地构建后 `docker save | gzip | ssh ... docker load`。
- **1 Mbps 出方向**：静态资源和 API 响应会受限；媒体读走 OSS，上传到广州桶走内网，不受这个限制。
- **B 是 CentOS 7.9 EOL**：作为未来 prod/冷备前必须重装为 ALinux 4。
- **`deploy.env` 的 DSN**：必须加单引号，否则 `&parseTime=True` 会被 shell 截断。
- **发布脚本顺序**：`deploy.sh` 先切静态再拉/启 api；`snapshot-rds.sh` 可能读到旧快照的 Success。prod 前必须修。
- **不要改项目 `.mcp.json`**：阿里云 MCP 使用用户级配置；项目 `.mcp.json` 是 git 跟踪文件且被 secret 扫描器整文件拦截。

## 6. 明天会话的起点

- 分支：`main`，工作区干净。
- 服务器：A `<staging-public-ip>`；配置 `/etc/kindergarten/staging/app.ini`。
- 先跑 `docker compose ... ps` 确认三个容器仍健康，再填微信字段。
- 本文档未包含任何密钥，实例与网络标识已脱敏；真实值的位置见 `docs/handoffs/README.md`。
