# 养成星球 · 部署架构

- Status: Proposed（设计定稿，待实施）
- Date: 2026-11-03
- Scope: 端到端——Go 后端、管理端、两个微信小程序、发布与运维
- Related: [ADR-0007](adr/0007-single-ecs-compose-single-instance.md)、[ADR-0008](adr/0008-versioned-migrations-goose.md)、[ADR-0009](adr/0009-miniprogram-api-backward-compatibility.md)、[ADR-0010](adr/0010-private-oss-presigned-urls.md)

## 1. 目标与约束

| 项 | 结论 |
|---|---|
| 项目阶段 | 绿地：尚无生产环境，可推倒重来 |
| 规模 | ≤10 所园、≤3000 注册用户、峰值 ≤50 QPS |
| 预算 | ≤ ¥500/月（估算见 §10） |
| 租户模型 | 单部署、共享 MySQL、`kindergarten_id` 隔离 |
| 环境 | 本地开发 + 预发（staging） + 生产（prod） |
| 扩展性 | **单实例**；首期不引入 SLB、K8s、CDN、WAF |
| 地域 | 暂定阿里云华东1（杭州）；目标园所在北方则改华北2（北京） |
| 域名/备案 | 新注册主域名 + 公司主体 ICP 备案；本文用 `<domain>` 占位 |

**上线关键路径**：ICP 备案通常需要 1–3 周，必须先于小程序生产发布启动。

## 2. 目标拓扑

```
                        用户 / 微信小程序
                              │ HTTPS
                    ┌─────────▼─────────┐
                    │ 阿里云 DNS + EIP   │
                    └─────────┬─────────┘
                              │ 80/443
        ┌─────────────────────▼──────────────────────────────┐
        │ ECS 2C4G · 40–60GB ESSD（VPC · 华东1）              │
        │                                                     │
        │ 宿主机 Nginx                                        │
        │  ├ api.<domain>        → 127.0.0.1:8080  prod api  │
        │  ├ admin.<domain>      → /opt/kg/prod/admin         │
        │  ├ www.<domain>        → /opt/kg/prod/website       │
        │  ├ test-api.<domain>   → 127.0.0.1:8081  staging   │
        │  ├ test-admin.<domain> → /opt/kg/staging/admin      │
        │  └ test-www.<domain>   → /opt/kg/staging/website    │
        │                                                     │
        │ Docker Compose · prod                               │
        │  └ api（Go 镜像）                                   │
        │ Docker Compose · staging                            │
        │  ├ api（Go 镜像）                                   │
        │  ├ mysql:8（容器卷）                                │
        │  └ redis:7（容器卷）                                │
        └───────────────┬─────────────────────────────────────┘
                        │ VPC 内网
        ┌───────────────┼───────────────────┐
        │               │                   │
┌───────▼──────┐ ┌──────▼────────┐ ┌────────▼─────────┐
│ RDS MySQL    │ │ ApsaraDB Redis│ │ OSS prod/staging │
│ （仅生产）    │ │ （仅生产）     │ │ 私有桶 + 签名 URL │
└──────────────┘ └───────────────┘ └──────────────────┘
```

发布链路：

```
GitHub tag ─→ GitHub Actions ─→ ACR 镜像 ──SSH──→ ECS compose
                              ├→ admin 静态包 ──→ /opt/kg/<env>/admin
                              ├→ website 静态包 ─→ /opt/kg/<env>/website
                              └→ miniprogram-ci ─→ 微信后台（体验版）
```

## 3. 环境矩阵

| 维度 | 本地开发 | 预发 staging | 生产 prod |
|---|---|---|---|
| 后端 | `go run` :8080 | 容器 :8081 | 容器 :8080 |
| 管理端 | `npm run dev` | 静态 `test-admin.<domain>` | 静态 `admin.<domain>` |
| 官网 | `npm run dev` | 静态 `test-www.<domain>` | 静态 `www.<domain>` |
| MySQL | 本机 | 容器 `mysql:8` | RDS MySQL |
| Redis | 本机 | 容器 `redis:7` | 托管 Redis |
| OSS | dev 桶 | staging 桶 | prod 桶（私有） |
| 小程序构建 | 开发者工具 | `build:mp-weixin:test` | `build:mp-weixin:prod` |
| 后端域名 | `127.0.0.1:8080` | `test-api.<domain>` | `api.<domain>` |
| 数据 | 任意 | **仅合成/种子数据** | 真实数据 |
| 配置来源 | 本地 `app.ini` | GitHub Secrets 渲染 | GitHub Secrets 渲染 |

## 4. 组件与规格

| 组件 | 规格 | 说明 |
|---|---|---|
| ECS | 2 vCPU / 4 GiB / 40–60 GiB ESSD / EIP | 按流量计费，峰值 100 Mbps |
| 宿主机 Nginx | 系统包 | TLS 终止、反向代理、静态站点、限流 |
| 生产后端 | Docker Compose 单容器 | Go 静态二进制镜像 |
| 预发后端 | Docker Compose 三容器 | api + mysql:8 + redis:7 |
| 生产 MySQL | RDS MySQL 基础版 1C2G / 20GB | 同地域、同 VPC |
| 生产 Redis | ApsaraDB Redis 1GB | 同地域、同 VPC |
| 对象存储 | OSS prod / staging 桶 | 私有、版本控制、同地域 |
| 镜像仓库 | ACR 个人版 | 按版本 tag 保存镜像 |
| 日志 | SLS | 采集容器 stdout 与 Nginx 日志，保留 30 天 |
| 监控 | CloudMonitor | 站点探测 + 资源告警 |
| 证书 | acme.sh + Let's Encrypt | DNS/HTTP 校验，自动续期 |

**首期不引入**：SLB、ACK/K8s、CDN、WAF、Prometheus/Grafana、KMS/Secrets Manager。

## 5. 网络与安全

- ECS、RDS、Redis 置于同一 VPC；RDS/Redis 不开放公网，白名单仅 ECS 私网 IP。
- 安全组入站仅 80/443 对公网；SSH 22 仅管理员固定 IP。
- SSH 仅密钥登录，禁用密码与 root 登录；部署使用独立 `deploy` 账号。
- Nginx：TLS 1.2+、HSTS、HTTP→HTTPS 跳转、`limit_req`、`client_max_body_size 25m`。
- 后端 CORS 仅允许 `https://admin.<domain>`；小程序请求无 Origin，不受影响。
- 管理端 Cookie 使用 `Secure` + `SameSite=Lax`。
- CI 与日常运维使用阿里云 RAM 子账号（ACR 推送、ECS 部署各自最小权限），不使用主账号 AccessKey。
- 生产 `[face].enabled=false`，人脸识别未接真实供应商前不启 worker。

## 6. 配置与密钥

- 仓库只提交模板 `deploy/config/app.ini.tmpl`；**任何真实密钥不入库**。
- 密钥存放在 GitHub Environment Secrets：`staging` 与 `production` 各自独立。
- 部署时渲染到 `/etc/kindergarten/{env}/app.ini`（`0600`，属主 `deploy`），容器只读挂载，`CONFIG_PATH` 指向该文件。
- `[casbin]` 的 `rbac_model.conf` / `rbac_policy.csv` 随镜像发布，路径改为容器内绝对路径（如 `/app/configs/...`）。
- 容器与数据库统一 `Asia/Shanghai`：容器 `TZ=Asia/Shanghai`，DSN `loc=Asia%2FShanghai`。
- 生产配置示例：`gin_mode=release`、`log.format=json`、`log.level=info`。

### 6.1 上线前必须轮换/清理的凭据

| 项 | 位置 | 动作 |
|---|---|---|
| RDS 密码 | git 历史、`.githooks/pre-commit` | 轮换；历史清理另行处理 |
| Redis 口令 | `start.bat`、`app.ini` | 轮换；删除明文文件 |
| 模型 API key | `cc.ps1` | 轮换；删除或改为本地环境变量 |
| JWT secret | `app.ini` | 轮换（会使现有会话失效，选停机窗口） |
| 微信 / OSS / 腾讯云密钥 | `app.ini` | 确认未被跟踪；按环境重新签发 |

## 7. 数据、迁移与存储

### 7.1 数据库迁移

- 引入 **goose** 版本化迁移，取代启动期 `AutoMigrate`（见 ADR-0008）。
- `server/migrations/00001_baseline.sql` 由 `go run ./cmd/schemadump` 从**已执行完全部 `docs/migrations/` 手工 SQL 的开发库**导出；在空库上 `make migrate` 验证。
- 发布流水线在应用重启前运行一次性迁移容器：`/app/kindergarten-migrate up`（迁移文件已编进二进制）；失败即中止发布。
- 应用运行账号只有 DML 权限；迁移账号具备 DDL 权限，凭据仅存在于 CI。
- 启动期种子（字典、膳食、园长角色）改为显式、幂等命令 `make seedbase`；不再挂在读取/启动路径。

### 7.2 备份与恢复

| 项 | 策略 |
|---|---|
| RDS 自动备份 | 每日全量 + binlog，保留 14 天 |
| 逻辑备份 | 每周 `mysqldump` 到 OSS，保留 8 周 |
| 发布前快照 | 每次生产发布前手动快照 |
| OSS | 生产桶开启版本控制；90 天后转低频存储 |
| Redis | 视为可丢失（会话失效重登），不备份 |
| staging | 不备份 |
| 恢复目标 | RPO ≤ 5 分钟；RTO ≤ 2 小时 |
| 演练 | 每季度恢复到临时实例并跑 `dbcheck` |

### 7.3 对象存储

- 生产与预发桶均为**私有**；不使用公共读。
- 数据库保存对象引用：新写入为 object key，历史完整 OSS URL 由解析器兼容；响应层不得直接下发该引用。
- 读取接口按需签发 **1 小时有效**的 GET URL；小程序端遇到过期 URL 重新拉取。
- 首期不启用 CDN；媒体量上来后再评估「私有桶 + CDN 回源签名」。

## 8. 发布流程

### 8.1 CI（GitHub Actions）

1. **PR 门禁**：`make check`（vet + lint + test）、`check-tokens`、`check-secrets`、admin 与 website 构建。
2. **打 tag**：`vYYYY.MM.DD-N`。
3. **构建**：
   - 后端：多阶段构建静态镜像，推送 ACR（tag + commit sha）。
   - 管理端：`nuxt generate` 产出静态包。
   - 官网：`nuxt generate` 产出静态包。
   - 小程序：分别构建测试包（`test-api`）与生产包（`api`）。
4. **自动发预发**：渲染配置 → `goose up` → `compose pull/up` → `/ready` 探测 → 冒烟。
5. **人工批准**：GitHub Environment `production` 配置 required reviewer。
6. **发生产**：RDS 快照 → `goose up` → 部署 → `/ready` 探测 → 冒烟；失败切回上一个镜像 tag。
7. **小程序**：`miniprogram-ci` 上传体验版；「提交审核 / 发布」在微信公众平台人工操作。

### 8.2 回滚

| 层 | 方式 |
|---|---|
| 后端 | `docker compose up -d` 指定上一个镜像 tag |
| 管理端 | `current` symlink 切回上一个 `releases/<tag>` |
| 数据库 | 迁移只向前；破坏性变更靠发布前快照恢复 |
| 小程序 | 微信后台回退版本（受平台能力限制） |

### 8.3 API 兼容纪律

后端 API 必须向后兼容「仍在线上运行的最旧小程序版本」（见 ADR-0009）：

- 先加字段/接口，后删旧字段/接口。
- 破坏性语义变更必须拆两次发布：先兼容双写/双读，再清理。
- 契约文档（`backend/api/`）标注弃用时间与最低客户端版本。

## 9. 小程序发布

- 两个小程序使用各自真实 AppID；老师端为个人号，`web-view` 业务域名不可用，已划出范围。
- 域名白名单（两个后台分别配置）：`https://api.<domain>`、`https://test-api.<domain>` 作为 request / upload / download 合法域名。
- 测试包指向 `test-api.<domain>`，生产包指向 `api.<domain>`；域名在构建时内联，**改域名必须重新构建**。
- `miniprogram-ci` 上传私钥存 GitHub Secrets；暂不启用 IP 白名单（与 GitHub 动态出口 IP 冲突）。
- 审核与正式发布保留人工操作。

## 10. 可观测性

| 项 | 方案 |
|---|---|
| 应用日志 | stdout JSON；Docker `json-file` 轮转 50MB × 5 |
| 日志采集 | SLS Logtail 采集容器 stdout + Nginx access/error，保留 30 天 |
| 关联 | 全链路 `request_id` |
| 站点监控 | CloudMonitor 每分钟探测 `https://api.<domain>/health`，连续 2 次失败告警 |
| 资源监控 | ECS CPU/内存/磁盘、RDS 连接数/磁盘、Redis 内存/连接数 |
| 告警通道 | 短信 + 邮件（阿里云联系人组） |
| 健康检查 | `/health` 存活；`/ready` 检查 MySQL/Redis |

## 11. 成本估算

> 估算值，以实际询价为准；年付通常更便宜。

| 项 | 规格 | 约 ¥/月 |
|---|---|---|
| ECS | 2C4G、40–60GB ESSD、按流量计费 | 150 |
| RDS MySQL | 1C2G、20GB、基础版 | 120 |
| Redis | 1GB | 60 |
| OSS + 流量 + SLS | 媒体/日志 | 60 |
| 域名 + 证书 | — | 10 |
| **合计** | | **≈ 400** |

## 12. 上线前代码加固清单

1. 优雅停机：`signal.NotifyContext` + `http.Server.Shutdown`。
2. 健康检查拆分：`/health` 存活；`/ready` 检查 MySQL/Redis。
3. CORS 收紧到 `https://admin.<domain>`。
4. 移除启动期 `AutoMigrate` / 种子；改为 goose + 显式幂等种子命令。
5. OSS 私有桶 + 签名 GET：数据库存 object key，读取时签名。
6. 生产 `[face].enabled=false`。
7. 导出临时文件清理（启动清理 + 每日 cron）。
8. Nginx `client_max_body_size 25m`，对齐后端 20MB 单文件限制。
9. Gin `SetTrustedProxies`，保证 Nginx 后真实 IP。
10. 管理端改 `nuxt generate`，并把 `fonts.googleapis.com` 换成本地/国内可达字体。
11. 管理端 Cookie 加 `Secure` / `SameSite`。
12. 两个小程序替换真实域名，配置微信服务器域名白名单。

## 13. 服务器目录与端口约定

```
/opt/kindergarten/
  prod/
    docker-compose.yml
    admin/
      releases/<tag>/
      current -> releases/<tag>
  staging/
    docker-compose.yml
    admin/
      releases/<tag>/
      current -> releases/<tag>

/etc/kindergarten/
  prod/app.ini       # 0600, deploy:deploy
  staging/app.ini    # 0600, deploy:deploy

/etc/nginx/conf.d/kindergarten.conf
/etc/nginx/ssl/<domain>/  # acme.sh 签发
```

| 服务 | 监听 |
|---|---|
| Nginx | 0.0.0.0:80 / 443 |
| prod api | 127.0.0.1:8080 |
| staging api | 127.0.0.1:8081 |
| staging mysql | compose 内网（不映射宿主端口） |
| staging redis | compose 内网（不映射宿主端口） |

## 14. 风险与开放项

| 风险 | 影响 | 缓解 |
|---|---|---|
| ICP 备案周期 | 阻塞小程序生产发布 | 立项即启动备案 |
| 地域/域名/备案主体假设 | 资源选型返工 | 实施前最终确认 |
| GitHub Actions → ACR 跨境推送 | 发布慢/失败 | 实测；不可用再评估云效 Flow（先 PoC） |
| 单 ECS 单点故障 | 全站不可用 | 托管数据层 + 备份 + 季度恢复演练 |
| 微信审核延迟 | 客户端版本不同步 | API 向后兼容纪律（ADR-0009） |
| 人脸识别为 mock | 生产误启 worker | `[face].enabled=false` |
| 2C4G 内存余量 | staging MySQL/Redis 与 prod api 争内存 | 监控；不足则 staging DB 迁出或升配 |
| 无 CDN | 媒体加载慢/流量成本 | 首期接受；量级上来再评估 |

## 15. 未纳入本期

- 多实例 / 水平扩展 / SLB / K8s。
- CDN、WAF、Prometheus/Grafana。
- 阿里云 KMS / Secrets Manager。
- 小程序审核提交的全自动 API（平台能力与合规限制）。
- 跨地域容灾。

## 16. 变更记录

| 日期 | 变更 |
|---|---|
| 2026-11-03 | 初版设计定稿（21 项决策） |
