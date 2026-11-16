# 部署 Runbook

配套设计：[部署架构](../deployment-architecture.md)；决策：[ADR-0007 ~ ADR-0010](../adr/)。

> **当前阶段（2026-11-15）**：只落地了 staging；§1–§5 是 prod 目标流程，尚未启用。staging 实际步骤见 §1.0，访问门禁见 §1.1，详细状态见 `docs/handoffs/2026-11-15-staging-bootstrap.md`。
> 本文中的服务器 IP、bucket 名等已脱敏为占位符；真实值见运维私密记录，占位符约定见 `docs/handoffs/README.md`。

## 0. 约定与前置

| 项 | 值 |
|---|---|
| 生产域名 | `api.nurtureplanet.cn`、`admin.nurtureplanet.cn`、`www.nurtureplanet.cn`（apex 301 → `www`）；**未创建** |
| 预发域名 | `test-api.nurtureplanet.cn`、`test-admin.nurtureplanet.cn`、`test-www.nurtureplanet.cn` |
| 生产 API 端口 | ECS 本机 `127.0.0.1:8080`（未启用） |
| 预发 API 端口 | ECS 本机 `127.0.0.1:8081` |
| 当前服务器 | A `np-staging` `<staging-public-ip>`（staging）；B `np-idle` `<idle-public-ip>`（闲置） |
| 目录 | `/opt/kindergarten/{prod,staging}`；当前只有 `staging` |
| 配置 | `/etc/kindergarten/{prod,staging}/app.ini`（0600）；当前只有 `staging` |
| 部署密钥 | `/etc/kindergarten/{prod,staging}/deploy.env`（0600，含 `MIGRATE_MYSQL_DSN`） |

前置条件（缺一不可）：

1. 域名已完成 ICP 备案，DNS 指向 ECS EIP；`www` / `test-www` 也要有 A 记录。
2. 证书 SAN 覆盖 `api` / `admin` / `test-api` / `test-admin` / `www` / `test-www`（以及 apex）；少一个主机名会让对应的 server block 握手失败。
3. 两个小程序后台已配置 request / upload / download 合法域名（`api` 与 `test-api`）。
4. GitHub Environments `staging`、`production` 已配置 Secrets；`production` 已配置 required reviewer。
5. ECS 已安装 Docker + Compose 插件、Nginx、curl；安全组仅开 80/443，SSH 限管理员 IP。
6. RDS 已创建 `kg_app`（DML）与 `kg_migrate`（DDL）账号；`kg_migrate` 凭据只放在 `deploy.env` 与 GitHub Secrets。

## 1.0 staging 首次部署（2026-11-15 实际）

> 详细状态与待办见 `docs/handoffs/2026-11-15-staging-bootstrap.md`。以下为当前 staging 的实际流程；prod 尚未使用。
> 平台管理员、测试园所与 `kg_migrate` 账号的初始化/幂等重置步骤见
> [staging-initialization.md](staging-initialization.md)。

1. **DNS**：`test-api` / `test-admin` / `test-www` → `<staging-public-ip>`，TTL 600。解析线路必须是「默认」或额外加「境外」，否则 Let's Encrypt 会看到 NXDOMAIN。
2. **安全组**：A（`sg-np-staging`）开放 80/443，22 限管理员 IP；B（`sg-np-idle`）只开 22。
3. **安装**：Docker 24 + Compose、Nginx 1.30、acme.sh 3.1.5、jq/curl/git/rsync/httpd-tools。
4. **配置**：`/etc/kindergarten/staging/app.ini`（0600）。MySQL/Redis 指向 compose 服务名；OSS 用 `<oss-internal-endpoint>` + `<bucket-staging>`；微信 teacher/parent AppID/Secret 填真实值。
5. **容器**：`docker compose -f docker-compose.yml -f docker-compose.local.yml up -d mysql redis` → `run --rm migrate` → `run --rm --entrypoint /app/kindergarten-seedbase api` → `up -d api`。
6. **静态包**：解压到 `admin/releases/<tag>`、`website/releases/<tag>`，`current` 指向该 release（当前 `staging-20261115-1`）。

   > **构建即检查（管理端）**：`npm run generate` 的 `postgenerate` 会自动执行
   > `node scripts/check-admin-static.mjs .output/public`，校验 `index.html`、
   > `_nuxt` 下的 JS/CSS、远程字体（`fonts.googleapis.com` / `fonts.gstatic.com`）
   > 与 `localhost` / `127.0.0.1` 地址（仅放行 axios 浏览器适配器的兜底表达式，见脚本白名单）；
   > 任一命中即构建失败，并打印规则与命中文件。
   > 通过后日志末尾打印入口、资源数与体积摘要。CI 与 release 流水线都跑
   > `npm run generate`，因此无需额外步骤；打包/上传前也不必再人工抽查产物内容。
   >
   > **构建期环境变量**：`NUXT_PUBLIC_SITE_URL` 决定管理端同意页链接的官网地址，
   > 未注入时回退到生产默认值 `https://www.nurtureplanet.cn`（`admin/nuxt.config.ts`）；
   > staging 若需指向 `test-www`，构建时注入
   > `NUXT_PUBLIC_SITE_URL=https://test-www.nurtureplanet.cn`。生产留空即用默认值。
   > `NUXT_PUBLIC_API_BASE` 同理，默认同源 `/api/v1`。
7. **证书**：
   ```bash
   /root/.acme.sh/acme.sh --issue --server letsencrypt \
     -d test-api.nurtureplanet.cn -d test-admin.nurtureplanet.cn -d test-www.nurtureplanet.cn \
     -w /var/www/acme --keylength 2048 --request-v4
   /root/.acme.sh/acme.sh --install-cert -d test-api.nurtureplanet.cn \
     --key-file /etc/nginx/ssl/nurtureplanet.cn/privkey.pem \
     --fullchain-file /etc/nginx/ssl/nurtureplanet.cn/fullchain.pem \
     --reloadcmd "systemctl reload nginx"
   ```
   `ACME_USE_IPV4_REQUESTS=1` 必须保留（服务器 IPv6 出口不通）。
8. **验证**：`curl -s https://test-api.nurtureplanet.cn/ready`、`https://test-admin.nurtureplanet.cn/`、`https://test-www.nurtureplanet.cn/`（后两者自 2026-11-15 起需要 staging Basic Auth，见 §1.1；`/privacy` 与 `/terms` 匿名可访问）。

> 当前 staging 不走 ACR：API 镜像在本地构建后 `docker save | gzip | ssh ... docker load` 传到 A。GitHub Actions tag 流水线尚未启用。

## 1.1 staging 访问门禁（test-admin / test-www Basic Auth，2026-11-15）

对应 ticket #53。门禁只作用于 staging 的两个静态站点；`test-api` 保持公开（微信测试包匿名调用）。

| 站点 | 需要凭据 | 匿名可用 |
|---|---|---|
| `test-admin` | `/`、`/_nuxt/`（SPA 页面与静态资源） | `/api/v1/`、`/health`、`/ready`、`/.well-known/acme-challenge/` |
| `test-www` | 未列出的页面 | `/privacy`、`/terms`（家长端 web-view 匿名打开）、`/_nuxt/`、`/screens/`、`/brand/`、常见静态文件、`/404.html`、`/api/v1/`、`/health`、`/ready`、`/.well-known/acme-challenge/` |
| `test-api` | 无 | 全部 |

- 两个 staging 站点共用一份 realm（`Kindergarten staging`）与一份 htpasswd，用户名固定 `staging`；口令只在服务器生成。
- 生产 `admin` / `www` 不启用门禁（模板中只在 staging 两个 server 块加了指令；生产是否启用另行决策）。

> **test-www 匿名豁免范围（对 #53 US4 的有意偏离）**：test-www 与生产共用同一份官网静态产物。
> 家长端 web-view 会匿名打开 `/privacy`、`/terms`，这两个页面及其引用的 `/_nuxt/`、`/brand/`
> （以及 `screens/` 图片、字体等常见静态文件）必须匿名可读，否则 web-view 里版面与插图会裂开。
> `/404.html` 也必须匿名：匿名请求在政策页缺失时会走 `error_page 404`，若被门禁拦成 401
> 就看不到真正的 404。`/api/v1/`、`/health`、`/ready` 在该 host 显式 `return 404`
> （官网静态站没有后端），保持匿名是为了将来接入后端时不被门禁拦截。**除上表列出的路径外，
> 其余页面一律需要凭据。**

### 1.1.1 凭据文件（只在服务器）

| 项 | 值 |
|---|---|
| htpasswd | `/etc/nginx/htpasswd/kindergarten-staging`（**`0640 root:nginx`**） |
| 口令备份 | `/root/kindergarten-staging-basic-auth.txt`（`0600 root:root`） |
| 仓库 | 不含任何凭据；不要把 htpasswd 或口令写进仓库、ticket 或聊天 |

> 必须是 `0640 root:nginx`：Nginx worker 以 `nginx` 用户读 htpasswd，`0600 root:root` 会让所有带凭据的请求返回 **500**（`error.log` 报 `open() ... failed (13: Permission denied)`）。目录 `/etc/nginx/htpasswd` 为 `0755 root:root`。已在 nginx 1.30.4 上实测。

### 1.1.2 首次创建

```bash
sudo install -d -m 0755 /etc/nginx/htpasswd
PW="$(openssl rand -base64 24)"
sudo htpasswd -bc /etc/nginx/htpasswd/kindergarten-staging staging "$PW"
sudo chown root:nginx /etc/nginx/htpasswd/kindergarten-staging
sudo chmod 640 /etc/nginx/htpasswd/kindergarten-staging
printf 'sites: test-admin.nurtureplanet.cn, test-www.nurtureplanet.cn\nusername: staging\npassword: %s\n' "$PW" \
  | sudo tee /root/kindergarten-staging-basic-auth.txt >/dev/null
sudo chmod 600 /root/kindergarten-staging-basic-auth.txt
unset PW
```

### 1.1.3 同步 Nginx

线上 staging 使用 staging 专用配置 `/etc/nginx/conf.d/kindergarten-staging.conf`；仓库模板 `deploy/nginx/kindergarten.conf.tmpl` 的 staging server 块保持同一套指令。只改 auth 相关指令时可直接编辑线上 conf，**改动前必须备份**：

```bash
sudo cp -a /etc/nginx/conf.d/kindergarten-staging.conf \
  /etc/nginx/conf.d/kindergarten-staging.conf.bak-$(date +%Y%m%d-%H%M)
sudo nginx -t && sudo systemctl reload nginx
```

`nginx -t` 失败时立即用备份恢复并 reload。渲染模板（新环境/整文件替换）用：

```bash
sudo envsubst '$DOMAIN' < deploy/nginx/kindergarten.conf.tmpl > /etc/nginx/conf.d/kindergarten.conf
```

### 1.1.4 验证

先做结构化契约检查（无需服务器，纯模板断言）：

```bash
node scripts/check-nginx-staging-auth.mjs
# 期望：OK ...；staging 的 test-admin / test-www 含 auth_basic，
#       prod admin / www / apex 与 test-api 不含，80 端口块仅 ACME + 301
```

口令从服务器文件读取，不要贴进命令历史或聊天：

```bash
PW="$(sudo sed -n 's/^password: //p' /root/kindergarten-staging-basic-auth.txt)"

# 无凭据 / 错误凭据 → 401；正确凭据 → 200
curl -sI https://test-admin.nurtureplanet.cn/ -o /dev/null -w '%{http_code}\n'                      # 401
curl -sI -u staging:wrong https://test-admin.nurtureplanet.cn/ -o /dev/null -w '%{http_code}\n'     # 401
curl -sI -u "staging:$PW" https://test-admin.nurtureplanet.cn/ -o /dev/null -w '%{http_code}\n'     # 200
curl -sI -u "staging:$PW" https://test-admin.nurtureplanet.cn/_nuxt/ -o /dev/null -w '%{http_code}\n'

# _nuxt 长缓存：取首页引用的第一个 JS 资源，确认 Cache-Control 含 immutable
ASSET="$(curl -s -u "staging:$PW" https://test-admin.nurtureplanet.cn/ | grep -o '/_nuxt/[^"]*\.js' | head -1)"
echo "$ASSET"
curl -sI -u "staging:$PW" "https://test-admin.nurtureplanet.cn$ASSET" | grep -i '^cache-control'
# 期望：cache-control 含 "public, immutable"（expires 另会生成 max-age）

# 401 必须带 WWW-Authenticate
curl -sI https://test-admin.nurtureplanet.cn/ | grep -i '^www-authenticate'

# test-www：同一套凭据；政策页与静态资源匿名可用
curl -sI https://test-www.nurtureplanet.cn/ -o /dev/null -w '%{http_code}\n'                        # 401
curl -sI -u "staging:$PW" https://test-www.nurtureplanet.cn/ -o /dev/null -w '%{http_code}\n'       # 200
curl -sI https://test-www.nurtureplanet.cn/privacy -o /dev/null -w '%{http_code}\n'                 # 200
curl -sI https://test-www.nurtureplanet.cn/terms -o /dev/null -w '%{http_code}\n'                   # 200

# 不受影响：API / 健康检查 / ACME
curl -fsS https://test-api.nurtureplanet.cn/health -o /dev/null -w '%{http_code}\n'                 # 200
curl -sI https://test-admin.nurtureplanet.cn/api/v1/system/status -o /dev/null -w '%{http_code}\n'  # 非 401
curl -sI https://test-admin.nurtureplanet.cn/.well-known/acme-challenge/<file> -o /dev/null -w '%{http_code}\n'  # 非 401
unset PW
```

浏览器验收：带凭据打开 `test-admin`，确认登录页与 `_nuxt` 资源加载正常，刷新深链接仍返回 SPA；官网首页同样带凭据打开。

### 1.1.5 轮换

不改应用代码，重写 htpasswd 后 reload 即可：

```bash
PW="$(openssl rand -base64 24)"
sudo htpasswd -bc /etc/nginx/htpasswd/kindergarten-staging staging "$PW"
sudo chown root:nginx /etc/nginx/htpasswd/kindergarten-staging
sudo chmod 640 /etc/nginx/htpasswd/kindergarten-staging
printf 'sites: test-admin.nurtureplanet.cn, test-www.nurtureplanet.cn\nusername: staging\npassword: %s\n' "$PW" \
  | sudo tee /root/kindergarten-staging-basic-auth.txt >/dev/null
sudo chmod 600 /root/kindergarten-staging-basic-auth.txt
unset PW
sudo nginx -t && sudo systemctl reload nginx
```

- `htpasswd -c` 会覆盖整份文件；`-b` 把口令放进进程参数，只能在服务器本机执行，不要写进脚本仓库。
- 轮换后必须通知所有 staging 使用者，否则他们下次访问会看到 401。
- 不要重启应用容器：Basic Auth 完全在 Nginx 层。

### 1.1.6 回滚 / 临时关闭

```bash
sudo cp -a /etc/nginx/conf.d/kindergarten-staging.conf.bak-<时间戳> \
  /etc/nginx/conf.d/kindergarten-staging.conf
sudo nginx -t && sudo systemctl reload nginx
```

只临时关闭：删掉两个 staging server 块中的 `auth_basic` / `auth_basic_user_file` 后 reload；保留 htpasswd 与口令文件，恢复时无需重新生成。

## 1. 首次部署

> 以下为 prod 目标流程，尚未实施；当前 staging 见 §1.0。

```bash
# 1) 目录
sudo mkdir -p /opt/kindergarten/{prod,staging}/{admin,website}/releases
sudo mkdir -p /etc/kindergarten/{prod,staging}
sudo chown -R deploy:deploy /opt/kindergarten /etc/kindergarten

# 2) 配置：从模板渲染，填入真实密钥
sudo -u deploy cp deploy/config/app.ini.tmpl /tmp/app.ini
sudo -u deploy envsubst < /tmp/app.ini > /etc/kindergarten/prod/app.ini
sudo chmod 600 /etc/kindergarten/prod/app.ini
# deploy.env 至少包含（DSN 必须用单引号包住，否则 source 时 & 会被 shell 当成后台执行）：
#   MIGRATE_MYSQL_DSN='kg_migrate:***@tcp(<rds-host>:3306)/kindergarten?charset=utf8mb4&parseTime=True&loc=Asia%2FShanghai'
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
   - 执行迁移与基础种子（`seedbase`）、启动 API 并等 `/ready`、再切 admin/website symlink（顺序不能反）。
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

`rollback.sh` 的顺序与发布相反：先切静态、再切 API。旧静态 + 新 API 在 ADR-0009 下向后兼容；反过来会出现新静态页面打旧 API。

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
- **IPv4 与 DNS 线路**：本机 IPv6 出口不通时必须加 `--request-v4`；acme.sh 已持久化 `ACME_USE_IPV4_REQUESTS=1`。A 记录若不在「默认/境外」线路，Let's Encrypt 的解析器会看到 NXDOMAIN。

## 7. 常用命令

```bash
# staging（当前）
cd /opt/kindergarten/staging
docker compose -f docker-compose.yml -f docker-compose.local.yml ps
docker compose -f docker-compose.yml -f docker-compose.local.yml logs -f --tail=100 api
curl -s https://test-api.nurtureplanet.cn/ready

# prod（未来）
cd /opt/kindergarten/prod && docker compose ps
docker compose logs -f --tail=200 api

# 迁移状态 / 执行
docker compose --profile migrate run --rm --entrypoint /app/kindergarten-migrate migrate status
docker compose --profile migrate run --rm migrate   # 默认 entrypoint 是 up

# 只读核对表结构
cd /path/to/repo/server
MYSQL_DSN='<app-dsn>' go run ./cmd/dbcheck task_check_ins
MYSQL_DSN='<app-dsn>' go run ./cmd/dbcheck --query "SELECT COUNT(*) FROM files"

# 供给园所管理员（幂等；园所必须已存在，软删除视为不存在）
# -phone / -kg 必填；口令走环境变量，避免留在 shell 历史与进程参数里
CONFIG_PATH=/etc/kindergarten/prod/app.ini SETADMIN_PASSWORD='<强密码>' \
  go run ./cmd/setadmin -phone <手机号> -kg <园所ID>

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
| acme.sh 报 NXDOMAIN 但国内能解析 | 检查 A 记录的解析线路是否为「默认/境外」；用 `1.1.1.1` 验证 |
| acme.sh 连接 CA 超时 | 服务器 IPv6 出口不通；加 `--request-v4`，确认 `ACME_USE_IPV4_REQUESTS=1` |
| `docker pull` 超时 | Docker Hub 从广州不可达；本地构建后 `docker save` / `ssh docker load`，或配镜像加速 |
| `np-staging` 实例到期 | 2026-12-14；提前续费并开自动续费 |

## 9. 安全清单（每次上线前）

- [ ] 没有新增明文凭据；`node scripts/check-secrets.mjs` 通过
- [ ] 生产 `app.ini` 0600，属主 deploy
- [ ] 安全组仅 80/443；SSH 限管理员 IP
- [ ] RDS/Redis 无公网地址
- [ ] `cors_origins` 精确匹配本环境管理端：staging = `https://test-admin.nurtureplanet.cn`，prod = `https://admin.<domain>`；不含 `*`（见 §9.1）
- [ ] 发布前快照已成功
- [ ] 回滚 tag / admin symlink 已知
- [ ] MCP 使用专用 RAM 用户，AK 未进入任何被跟踪文件
- [ ] staging 安全组 22 已限管理员 IP（当前仍为 `0.0.0.0/0`，待收）
- [ ] staging Basic Auth 生效：`test-admin` / `test-www` 无凭据 401、`/privacy` 匿名 200；htpasswd 为 `0640 root:nginx`，口令文件 0600（见 §1.1）
- [ ] `np-staging` 续费已确认（2026-12-14 到期）

### 9.1 CORS / 反向代理 / 上传上限验收

CORS 由 `server/internal/middleware/cors.go` 实现：白名单精确匹配、留空默认拒绝、`OPTIONS` 返回 204、响应带 `Vary: Origin`。`cors_origins` 由 `deploy/config/app.ini.tmpl` 从 `CORS_ORIGINS` 渲染：

| 环境 | `cors_origins` |
|---|---|
| staging | `https://test-admin.nurtureplanet.cn` |
| prod | `https://admin.<domain>` |
| 本地 | `http://localhost:3000`（admin dev 走 Vite 代理时可不填） |

小程序请求不带 Origin、官网不调用 API，因此两者都不需要跨域来源；管理端经 Nginx 同源访问，同样不需要 CORS。staging 与 prod 仍显式配置上表来源，作为白名单与验收值（纵深防御与未来跨域）。留空是安全默认值。release 模式下误填 `*` 会在启动日志记录 WARN（不阻断启动），看到即应改回显式来源。

```bash
API=https://test-api.nurtureplanet.cn   # prod 换成 https://api.<domain>

# 1) 白名单来源：逐字回显 + Vary: Origin
curl -sS -o /dev/null -D - -H 'Origin: https://test-admin.nurtureplanet.cn' "$API/health" \
  | grep -iE '^(access-control-allow-origin|vary):'
# 期望：access-control-allow-origin: https://test-admin.nurtureplanet.cn
#       vary: Origin

# 2) 非白名单来源：无放行头，但仍有 Vary（防止缓存把 A 来源的响应复用到 B 来源）
curl -sS -o /dev/null -D - -H 'Origin: https://evil.example.com' "$API/health" \
  | grep -iE '^(access-control-allow-origin|vary):'
# 期望：只有 vary: Origin，没有 access-control-allow-origin

# 3) 预检：204，允许头含 terminal_from
curl -sS -o /dev/null -D - -X OPTIONS -H 'Origin: https://test-admin.nurtureplanet.cn' \
  -H 'Access-Control-Request-Method: GET' "$API/health" \
  | grep -iE '^(HTTP/|access-control-allow-origin|access-control-allow-headers):'
# 期望：HTTP/... 204；allow-headers 含 terminal_from
```

**代理头**：应用只信任来自 `127.0.0.1` / `::1`（宿主机 Nginx）的 `X-Forwarded-For`，见 `server/internal/app/router.go` 的 `SetTrustedProxies`；Nginx 用 `$proxy_add_x_forwarded_for` 追加而非透传，客户端伪造值只能排在最左。

```bash
# 1) 每个反向代理 location 都转发了三个代理头（每个块应有 3 行）
sudo grep -nE 'X-Real-IP|X-Forwarded-For|X-Forwarded-Proto' /etc/nginx/conf.d/kindergarten-staging.conf

# 2) 应用只信任宿主机回环地址（在仓库根目录执行）
grep -n 'SetTrustedProxies' server/internal/app/router.go
# 期望：[]string{"127.0.0.1", "::1"}；出现 0.0.0.0/0 或删掉该行都视为回归
```

**端到端核对**：`X-Forwarded-For` 只有经过一次真实写操作才会落进操作日志。用管理员账号带凭据打开 `test-admin`，登录后修改任意班级信息，然后在「操作日志」（侧栏审计 → 操作日志，路由 `/logs`）核对最新一条的 IP 为你的公网出口 IP；或直接查库：

```bash
MYSQL_DSN='<app-dsn>' go run ./cmd/dbcheck --query \
  "SELECT id, module, action, ip, created_at FROM operation_logs WHERE ip <> '' ORDER BY id DESC LIMIT 5"
```

要验证伪造头被忽略：用 `curl` 直接对**写接口**带 `X-Forwarded-For: 1.2.3.4`（需先取得管理员 JWT），再查上面同一条记录——IP 仍应是真实出口 IP，`1.2.3.4` 不应出现。只读接口（如 `/health`）不写操作日志，不能作为验收依据。

**上传上限**：`client_max_body_size 25m` 只出现在四个反向代理或静态块——`api`、`test-api`、`admin`、`test-admin`；`www` / `test-www` 是纯静态站，没有上传需求，不设该指令。后端业务层单文件限制仍是 20MB（按 `FileSize` 校验），两者不要混为一谈。用 26MB 与 24MB 各试一次：

```bash
head -c 26M /dev/zero > /tmp/26mb.bin
head -c 24M /dev/zero > /tmp/24mb.bin
# 超过 Nginx 上限：413，不进入应用（未带凭据也应先看到 413）
curl -sS -o /dev/null -w '%{http_code}\n' -X POST --data-binary @/tmp/26mb.bin "$API/api/v1/common/upload"
# 低于 Nginx 上限：不是 413（通常是 401/400，由应用层处理）
curl -sS -o /dev/null -w '%{http_code}\n' -X POST --data-binary @/tmp/24mb.bin "$API/api/v1/common/upload"
rm -f /tmp/26mb.bin /tmp/24mb.bin
```

## 10. MCP 运维凭据（alibaba-cloud-ops-mcp-server）

阿里云运维 MCP 配置在**用户级**（`~/.claude.json`，通过 `claude mcp add --scope user` 写入），**不进入仓库**：项目 `.mcp.json` 是 git 跟踪文件，且被 `scripts/check-secrets.mjs` 的 `sensitive-file` 规则整文件拦截。版本固定在 `0.9.27`，`--visible-tools` 只开放只读工具（ECS/RDS 查询、OSS 列举、CMS 指标）。**配置本身不存任何 AK。**

### 10.1 凭据来源

该 MCP 使用阿里云 SDK 默认凭据链，按顺序取：

1. 环境变量 `ALIBABA_CLOUD_ACCESS_KEY_ID` / `ALIBABA_CLOUD_ACCESS_KEY_SECRET`（STS 临时凭据再加 `ALIBABA_CLOUD_SECURITY_TOKEN`）
2. `ALIBABA_CLOUD_CREDENTIALS_URI`
3. `ALIBABA_CLOUD_PROFILE`（对应 `~/.aliyun/config.json`）
4. ECS 实例 RAM 角色（`ALIBABA_CLOUD_ECS_METADATA=<角色名>`）

### 10.2 本机（Claude Code）配置

1. 创建专用 RAM 用户（如 `mcp-ops-readonly`）：仅编程访问，不开控制台登录；只授只读策略，写操作按需单独加并限定资源 ARN。
2. 安装固定版本（不要用 `uvx`：每次启动都要解析/下载，冷启动会超过 30 秒的 MCP 握手超时）：
   ```bash
   uv tool install alibaba-cloud-ops-mcp-server==0.9.27
   ```
3. 在用户级注册 MCP（只需执行一次，配置写入 `~/.claude.json`，不进仓库）：
   ```bash
   claude mcp add --scope user alibaba-cloud-ops-mcp-server -- \
     "C:/Users/<你>/.local/bin/alibaba-cloud-ops-mcp-server.exe" \
     --services ecs,rds,oss,vpc \
     --visible-tools "ECS_DescribeInstances,RDS_DescribeDBInstances,OSS_ListBuckets,OSS_ListObjects,CMS_GetCpuUsageData,CMS_GetCpuLoadavgData,CMS_GetMemUsageData,CMS_GetDiskUsageData"
   ```
4. AK 写入 **Windows 用户环境变量**，不要写进项目文件：
   ```powershell
   setx ALIBABA_CLOUD_ACCESS_KEY_ID "LTAI..."
   setx ALIBABA_CLOUD_ACCESS_KEY_SECRET "..."
   ```
5. 重启终端与 Claude Code（`setx` 只对新进程生效），在 Claude Code 里用 `/mcp` 确认服务器已连接。
6. 验证：调用 `ECS_DescribeInstances`，能返回 `cn-guangzhou` 的实例列表即通。

> 动态 API 工具（`ECS_*` / `RDS_*`）启动时会从 `api.aliyun.com/meta/v1` 拉取 API 元数据，每个约 4 次请求；白名单里的动态工具越多，握手越慢，超过 30 秒就会被判为连接失败。当前只保留 `ECS_DescribeInstances` 和 `RDS_DescribeDBInstances`。需要更多工具时逐个加，并用 `/mcp` 验证启动时间；必要时用 `MCP_TIMEOUT`（毫秒）放宽 Claude Code 的启动超时。

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

### 10.5 方案 C：小 MCP + `aliyun` CLI（2026-11-14）

**架构**：MCP 只保留当前 8 个只读工具；安全组、VPC、Redis、SLS、ACR、账单、ActionTrail 等长尾操作走 `aliyun` CLI；服务器内部操作走 SSH MCP；发布仍走 GitHub Actions。

**为什么 MCP 不再加工具**：实测本机（2026-11-14，0.9.27）：

| 工具数 | 启动耗时 |
| --- | --- |
| 8（当前） | 15.5 秒 |
| 10（+2 个动态工具） | 29.0 秒 |

Claude Code 的 MCP 握手默认 30 秒，加到 10 个就贴着超时线，再加一个必挂。因此**不要往 `--visible-tools` 里加动态工具**；需要新查询时优先用 CLI。改配置后跑 `python scripts/verify-aliyun-mcp.py` 验证启动时间。

**凭据与 profile**：

- `mcp-readonly`：只读，绑定 `deploy/ram/mcp-readonly-policy.json`
- `mcp-operator`：写操作，绑定 `deploy/ram/mcp-readonly-policy.json` + `deploy/ram/cli-operator-policy.json`
- MCP 用户级配置已加 `ALIBABA_CLOUD_PROFILE=mcp-readonly`
- CLI 通过 `--profile mcp-readonly` / `--profile mcp-operator` 选择身份
- AK 只写入 `~/.aliyun/config.json`，不进入仓库、`.mcp.json` 或对话

**CLI 安装位置**：

- 官方 Go CLI v3.5.3：`D:\tools\aliyun-cli\aliyun.exe`
- Bash 入口：`~/bin/aliyun`
- PowerShell 入口：`~\.local\bin\aliyun.cmd`
- 不要用 PyPI 的 `aliyuncli`（2017 年旧包，依赖老 SDK）

**操作说明与占位符**：见 `deploy/ram/README.md`。

**边界纪律**：MCP 与 operator CLI 都不授予删机、重置密码、换系统盘、删 RDS、Flush Redis、删桶、改桶 ACL、`ram:*`；这些动作保留在控制台人工执行。ActionTrail 开启，AK 90 天轮换。

## 11. 存量 trial 延长（一次性，2026-12-06 前完成）

决策依据：ADR-0018 #1。命令：`server/cmd/extendtrials/`（规格 #62、工单 #65/#66）。
范围仅限 **2026-11-15（含）前开通、状态为 `active` 且未删除、仍有 `source='trial' AND order_id IS NULL` 授权、无在期订单、无在期订阅**的园所；目标到期日统一为 **2027-01-31 23:59:59 Asia/Shanghai**。

「在期」口径：

- **在期订单**：`status <> 'cancelled'` 且 `expires_at > now`，不看 `starts_at`——尚未到起算日的已付费订单同样阻断（否则试用会叠加在已付费授权上）。
- **在期订阅**：`source='subscription'` 且 `starts_at <= now AND expires_at > now`（与 `EntitlementRepo.ListActiveByKindergarten` 的领域惯例一致）。
- 边界均为严格大于：恰好 `expires_at = now` 的订单/订阅不再阻断。

### 11.1 流程（dry-run → 运营确认 → apply → 抽检 → revert 演练）

```bash
cd server

# 1) 只读预演：不设 EXTENDTRIALS_ALLOW，数据库不会发生任何变化
go run ./cmd/extendtrials -manifest ./extendtrials-2026-11-16.csv
```

- dry-run **默认拒绝覆盖已存在的 manifest（及其 `.sha256` 侧车）**：覆盖会毁掉运营已确认、用于回滚的清单。确需重新生成时显式传 `-force`；计划为空时不落盘，无需 `-force`（apply 后重跑 dry-run 的常见场景）。
- 输出末尾会列出**排除原因**（政策日后开通 / 园所非在营 / 无 trial / 在期订单 / 在期订阅 / 无需延长），以及**孤儿 trial 授权**（所属园所不存在或已删除，不在扫描集合内）：这类行不纳入本次延长，需运营确认是否另行处理。

产物三个文件，必须一起归档、一起保留在原目录：

| 文件 | 用途 |
|---|---|
| `extendtrials-2026-11-16.csv` | 逐条授权清单（园所、授权 ID、原到期日），apply/revert 的唯一输入 |
| `extendtrials-2026-11-16.csv.sha256` | 校验值侧车文件；apply/revert 只认它，没有手工覆盖哈希的参数 |
| `extendtrials-2026-11-16.csv.summary.csv` | 园所汇总（名称、trial 条数），供运营核对 |

```bash
# 2) 运营逐行确认清单与汇总：园所名单、授权条数、原到期日
#    确认后把 manifest 与 sha256 一起交给执行人（建议只读副本/staging 先跑一遍）

# 3) 写入。缺少 EXTENDTRIALS_ALLOW=1 或 -operator 都会被拒绝
EXTENDTRIALS_ALLOW=1 go run ./cmd/extendtrials \
  -manifest ./extendtrials-2026-11-16.csv \
  -apply -operator "张三"

# PowerShell: $env:EXTENDTRIALS_ALLOW="1" 后再执行同一命令
```

apply 只处理 manifest 中的授权：清单外的数据不变；已是目标日或更晚的行跳过（只延长不回退）；重复执行影响 0 行。写入前命令会先打印目标库（`host:port/库名`，不含凭据）供人工核对；成功后输出执行人、开始/结束时间、影响园所数与行数，并在 `platform_operation_logs` 落一条 `module=billing, action=extend_trials`，`detail` 含影响园所数、园所 ID、行数、目标日与 manifest SHA256。

**写入事务内会按最终口径复核清单内每一所园所**（未删除、状态 `active`、无在期订单、无在期订阅）：dry-run 与 apply 之间可能相隔数天，期间园所签约、停用或删除都会命中；任一园所不再符合即**整体拒绝**并在错误里点名园所 ID，不修改任何行。被拒绝时重新跑 dry-run、确认清单后再执行。

### 11.2 抽检

```sql
-- 期望：清单内园所的 trial 授权 max(expires_at) = 2027-01-31 23:59:59
SELECT kindergarten_id, COUNT(*) AS rows,
       MIN(expires_at) AS min_expires, MAX(expires_at) AS max_expires
FROM entitlements
WHERE deleted_at IS NULL AND source = 'trial' AND order_id IS NULL
  AND kindergarten_id IN (<manifest 中的园所 ID>)
GROUP BY kindergarten_id;

-- 期望：能看到本次执行的 operator / detail；失败的回滚不在其中
SELECT id, operator, module, action, detail, created_at
FROM platform_operation_logs
WHERE action IN ('extend_trials', 'revert_trials')
ORDER BY id DESC LIMIT 5;
```

### 11.3 回滚与演练

回滚使用**同一份 manifest 与 sha256 侧车文件**，不需要重新 dry-run：命令按清单里的 `old_expires_at` 逐条写回，属于**秒级近似恢复**（见下）。

```bash
EXTENDTRIALS_ALLOW=1 go run ./cmd/extendtrials \
  -manifest ./extendtrials-2026-11-16.csv \
  -revert -operator "张三"
```

- 回滚前逐行校验：当前 `expires_at` 必须等于 apply 目标日（或已等于 manifest 原值）。**任一行不等于两者即整体拒绝并点名**，不会覆盖 apply 之后的合法续期/调整。
- 重复回滚是 no-op（0 行），同样落一条 `action=revert_trials` 日志。
- 演练顺序：staging/只读副本 dry-run → apply → 抽检 → revert → 再次抽检确认恢复 → 重新 apply；生产执行前把演练结果与 manifest 一并归档。
- manifest 以**秒**精度记录原到期日，回滚是**秒级近似恢复**而非位级还原：若库里原值带毫秒，revert 只能恢复到秒（业务等价）。需要绝对精确值时先导出原值留档，人工核对。

### 11.4 拒绝条件（命令级护栏）

| 场景 | 行为 |
|---|---|
| `-manifest` 缺失/文件不存在/空文件/只有表头 | 拒绝，并给出明确错误 |
| dry-run 时目标 manifest（或 `.sha256` 侧车）已存在且未传 `-force` | 拒绝覆盖，既有文件原样保留 |
| manifest 版本行或 SHA256 不符 | 拒绝，不写库 |
| **apply 时复核不通过：清单内园所已删除 / 状态非 `active` / 出现 `expires_at > now` 的非取消订单（含未来起算）/ 出现在期订阅** | 整体拒绝，点名园所 ID，事务内不修改任何行 |
| apply/revert 时清单行与库中不一致：授权不存在 / 授权已软删 / 园所 ID 不符 / 园所已软删 / 已不是存量 trial（`source` 变了或已挂 `order_id`） | 整体拒绝，点名具体授权，事务内不修改任何行 |
| 未设 `EXTENDTRIALS_ALLOW=1` | 拒绝（dry-run 不需要该开关） |
| 未传 `-operator` | 拒绝 |
| `-apply` 与 `-revert` 同时使用 | 拒绝 |
| apply 时清单行已 ≥ 目标日 | 跳过，计入「未变更」 |
| revert 时清单行既非目标日也非原值 | 整体拒绝，列出授权 ID 与当前值 |
| 事务内任一写入失败 | 全部回滚，不留下部分更新 |

> revert **不做**口径复核：回滚是撤销已发生的事实，apply 之后园所签约/停用不应阻断回滚；但仍逐行校验当前值未漂移（上一行）。

### 11.5 测试

```bash
cd server
CONFIG_PATH=$(pwd)/configs/app.ini go test ./cmd/extendtrials -count=1
```

`CONFIG_PATH` 用绝对路径：`go test` 的工作目录是包目录（`cmd/extendtrials`），相对路径会找不到 `configs/app.ini` 而让全部数据库用例静默 skip。

测试自动创建并迁移独立 scratch 库 `kindergarten_extendtrials_test_<PID>`（按测试进程 PID 加后缀，并发跑多个 `go test` 进程互不干扰；不碰共享开发库），覆盖 apply/revert/幂等/SHA 与版本校验/写开关/防覆盖/事务回滚。可用 `EXTENDTRIALS_TEST_DSN` 指定其它测试实例。
