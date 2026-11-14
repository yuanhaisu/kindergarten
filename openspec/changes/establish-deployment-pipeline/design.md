# Design

## Context

动机见 `proposal.md`。影响方案的关键现状：

- 仓库没有任何可执行的部署资产；生产域名仍是占位符。
- 后端启动即执行 `AutoMigrate` 与种子；`[log]`、`[casbin]`、`[face]`、`redis.db`、`jwt.refresh_ttl_day` 等配置无法用现有环境变量覆盖，因此必须渲染完整的 `app.ini`，不能只注入几个环境变量。
- 媒体 URL 以完整公开 URL 存于 `files`，并被 `growth_media`、`album_photos`、`notices`、`leaves`、`health_check_media`、`tasks`、`users` / `students` / `pickup` 头像等表缓存；三端有大量读取路径。
- admin 是无 Nitro server route 的 SPA，可改为 `nuxt generate` 静态产物。
- Redis 是认证链路强依赖；后端含进程内定时任务、识别队列、课表同步与本地导出临时文件，当前按单实例运行。
- 开发环境为 Windows；仓库在 GitHub private；生产目标为阿里云。

## Goals / Non-Goals

**Goals:**

- 用不可变 tag 驱动可重复的后端 / admin / 小程序构建。
- 预发与生产隔离；生产部署有人工闸门与可执行回滚。
- 密钥不进入仓库；泄漏凭据在上线前完成轮换。
- 幼儿媒体默认私有，响应只下发短期签名 URL。
- schema 变更有版本记录，应用启动不再自愈建表。
- 健康检查、优雅停机、日志、告警、备份恢复达到可运维的最低线。
- 单实例边界明确，避免后续误扩。

**Non-Goals:**

- 多实例 / 水平扩展 / SLB / K8s / CDN / WAF / KMS。
- Terraform 等完整 IaC；云资源仍以控制台 + runbook 管理。
- 业务功能、RBAC 策略与接口形状变更。
- 小程序审核提交的全自动 API。

## Decisions

### D1 拓扑：单 ECS + 宿主机 Nginx + Docker Compose

- **选择**：一台 ECS 运行宿主机 Nginx 与两套 Compose；生产仅 `api`，预发 `api + mysql + redis`；生产 MySQL/Redis 使用阿里云托管服务。
- **理由**：规模小、单人运维；Compose 提供可复现运行环境与按镜像 tag 回滚；staging 与 prod 同机不同端口/数据，成本最低。
- **备选**：ACK/K8s（运维与费用过重）；systemd 原生二进制（回滚与环境一致性更差）；多实例 + SLB（需先改造导出/下载与进程内任务）。

### D2 构建产物：镜像 + 静态目录 + 小程序包

- **选择**：后端产出一个不可变镜像；admin 产出静态目录，按 `releases/<tag>` 存放并用 `current` symlink 切换；两个小程序产出测试/生产包，由 `miniprogram-ci` 上传体验版。
- **理由**：后端回滚 = 切镜像 tag；admin 回滚 = 切 symlink；小程序审核与发布保留人工闸门。
- **备选**：admin 继续跑 Nitro node-server（多一个进程、回滚更复杂）；小程序手工上传（无法在 CI 中复现）。

### D3 配置与密钥：模板 + GitHub Environment Secrets + 部署时渲染

- **选择**：仓库只提交 `app.ini` 模板；staging / production 的密钥存 GitHub Environment Secrets；部署时渲染到 `/etc/kindergarten/{env}/app.ini`（0600），容器只读挂载；`CONFIG_PATH` 指向该文件。Casbin 两个文件随镜像发布，路径改为容器内绝对路径。
- **理由**：必须生成完整 ini 才能覆盖不可环境变量化的配置；模板可评审，密钥只存在 GitHub 与服务器 0600 文件。
- **备选**：服务器手工维护 ini（不可复现）；KMS/Secrets Manager（当前规模过重）。

### D4 迁移：goose 版本化 + 迁移 job

- **选择**：以**实际数据库 schema**（包含 `docs/migrations/` 已执行的手工 SQL）导出 `server/migrations/00001_baseline.sql`，用 goose 管理；发布流水线在应用重启前跑一次性迁移 job；应用运行账号只有 DML，迁移账号凭据仅存在于 CI；启动期不再 `AutoMigrate`、不再播种。
- **理由**：`AutoMigrate` 只增不改、启动即执行、无版本记录；绿地阶段引入基线的成本最低。
- **备选**：保留 `AutoMigrate`（无版本记录、DDL 常驻）；仅手工 SQL（无执行器，易漂移）；Atlas（当前规模过重）。

### D5 媒体：私有桶 + 响应层签名

- **选择**：生产/预发桶私有；新写入的媒体引用保存为 object key（缩略图保存 key + `x-oss-process` 参数），历史完整 OSS URL 由解析器兼容；**所有**读取响应在序列化时统一签发 1 小时 GET URL；`thumb_url` 的图片/视频处理参数一并参与签名；首期不启用 CDN。
- **理由**：私有桶让「忘记签名」的路径失败关闭（返回不可访问的 URL），而不是泄露；用解析器兼容历史 URL，避免对 10+ 张表做 schema 变更。
- **备选**：公共读 + 不可枚举 key（泄露即公开）；后端代理媒体（稳定 URL，但占 ECS 带宽且需鉴权改造）；私有桶 + CDN 回源签名（首期不需要）。

### D6 运行时：健康检查、优雅停机、单实例

- **选择**：`/health` 只报进程存活；`/ready` 检查 MySQL/Redis；收到 SIGTERM/SIGINT 后停止接收新请求并在有界超时内完成在途请求；CORS 收紧到 admin 域名；Gin 设置可信代理；导出临时文件启动清理 + 每日清理；生产 `[face].enabled=false`。
- **理由**：部署重启不能掐断在途请求；`/ready` 让流水线能判断「新版本是否真的可用」；单实例是当前进程内任务与本地导出文件的正确边界。
- **备选**：不做优雅停机（发布即报错）；把 `/ready` 做成依赖探测（会把 Redis 抖动放大成全站不可用）。

### D7 发布：tag 驱动、预发自动、生产审批

- **选择**：`vYYYY.MM.DD-N` tag 触发；staging 自动部署；production 需 GitHub Environment 人工批准；生产发布前确认 RDS 快照；就绪失败自动回滚镜像与静态目录。
- **理由**：小程序审核周期使「服务端与客户端同步发布」不可行，必须把发布拆成可回滚的独立步骤。
- **备选**：分支合并即发布（缺少审批与版本标识）；纯手动 Run workflow（无法保证 tag 与产物一一对应）。

### D8 可观测性与恢复：SLS + CloudMonitor + 备份

- **选择**：应用 JSON 日志到 stdout，Docker 轮转，SLS 采集 30 天；Nginx 日志同样进 SLS；CloudMonitor 每分钟探测 `/health`，资源告警走短信/邮件；RDS 每日备份 + binlog、每周逻辑备份到 OSS、发布前快照、季度恢复演练。
- **理由**：单 ECS 没有自愈能力，必须能「发现、定位、恢复」。
- **备选**：自建 Prometheus/Grafana（当前规模过重）；只看本地日志（无法跨发布定位）。

### D9 网络与安全

- **选择**：VPC 内网连接 RDS/Redis；安全组仅 80/443 入站；SSH 22 限管理员 IP、仅密钥；Nginx TLS 1.2+/HSTS/限流/25MB 上传；CI 用 RAM 子账号。
- **理由**：把公网暴露面压到最小；数据库与缓存不暴露公网。
- **备选**：先全开后收紧（容易形成长期债）；WAF（当前规模不需要）。

### D10 管理端静态化

- **选择**：`nuxt generate` 出静态站点，Nginx `try_files ... /index.html`；`NUXT_PUBLIC_API_BASE` 保持 `/api/v1` 同源；移除 `fonts.googleapis.com`。
- **理由**：SPA 不需要 Node 运行时；静态目录回滚简单；同源反代避免 CORS 与构建期固化环境地址。
- **备选**：保留 Nitro node-server（多一个进程与端口）；构建期写死完整后端域名（环境迁移需重建）。

### D11 小程序发布

- **选择**：两个小程序分别构建 `test` / `prod` 包；`miniprogram-ci` 上传体验版；「提交审核 / 发布」人工；域名白名单在各自后台配置；上传私钥进 GitHub Secrets，暂不启用 IP 白名单。
- **理由**：审核与正式发布是平台人工流程，自动化到「上传体验版」即可覆盖重复劳动。
- **备选**：开发者工具手工上传（不可在 CI 复现）；启用 IP 白名单 + 自托管 runner（后续可选）。

## 跨端触点清单

| 层 | 结论 | 说明 |
|---|---|---|
| Go domain | **会改（仅注释/取值语义，不改结构）** | 不新增表/字段；`File.FileURL` / `FileRecord.FileURL` 等字段名保留，新写入值为 object key；需补注释说明「字段保存对象引用，响应层签名」。 |
| repository | **不改** | 字段与查询不变；媒体引用仍通过既有列读写。 |
| service | **会改** | `file_service.go`（上传写 object key、返回签名 URL）、`storage/oss`（签名支持 `x-oss-process`、object key 解析）、`teacher_portal_service.go`（成长媒体读写签名；废弃上传入口的 URL 归一化）、growth/album/notice/leave/health/task/pickup 等服务的读取映射、`export_task_service.go`（临时文件清理）、`recognition_queue.go`（生产关闭）、`provider.go`（移除启动迁移/种子）。 |
| handler/http | **会改** | 所有返回媒体 URL 的响应路径统一走签名；新增 `/ready`；`/health` 保持；管理端文件列表签名。 |
| middleware | **会改** | CORS 允许列表改为 `https://admin.<domain>`；可信代理配置。JWT/Casbin 逻辑不变。 |
| Casbin policy | **不改** | 无新增受保护业务路由；`/health`、`/ready` 在认证组外。 |
| `router.go` | **会改** | 注册 `/ready`；不新增业务路由。 |
| `backend/api/` 契约 | **会改（语义，不改形状）** | `api-common.md` 的 `file_url` / `thumb_url` 说明改为短期签名 URL；各业务契约中示例 URL 同步加注。无新增/删除字段。 |
| teacher 页面 | **会改** | `pages/growth/create.vue`、`pages/growth/detail.vue`、`pages/growth/index.vue`、`pages/album/index.vue`、`pages/notice/index.vue`、`pages/notice/create.vue`、`pages/index/index.vue`、`pages/doctor/checkin.vue`、`pages/leave/index.vue`、`pages/tasks/detail.vue`、`components/AvatarBadge.vue`：图片加载失败时重新拉取数据以刷新签名 URL；提交仍用 `file_id`。 |
| parent 页面 | **会改** | `pages/growth/index.vue`、`pages/growth/detail.vue`、`pages/growth/album.vue`、`pages/growth/report.vue`、`pages/album/index.vue`、`pages/album/detail.vue`、`pages/class/index.vue`、`pages/pickup/qrcode.vue`、`pages/tasks/detail.vue`、`pages/tasks/wall.vue`、`pages/leave/create.vue`、`pages/index/index.vue`、`pages/student/index.vue`、`pages/points/index.vue`、`pages/profile/index.vue`、`components/AvatarBadge.vue`：同上。 |
| admin 页面 | **会改** | 所有渲染 `file_url` / `thumb_url` / `cover_url` / `avatar` 的页面，至少 `pages/albums/index.vue`、`pages/announcements/index.vue`、`pages/teachers/index.vue`、`pages/students/index.vue`；`types/api.d.ts` 注释同步。图片加载失败时触发刷新。 |
| design token | **不改** | 无样式/间距/色值变更。 |
| 小程序 env/config | **会改** | `teacher/.env.test`、`teacher/.env.production`、`parent/.env.test`、`parent/.env.production` 与两端 `src/config/env.js`：替换真实域名；构建时内联，改域名必须重建。 |
| CI / 基础设施 | **会改（新增）** | `.github/workflows/`、`deploy/`、`server/Dockerfile`、`server/migrations/`、`deploy/config/app.ini.tmpl`、Nginx 配置、runbook。 |

### 导航与 tabBar

本次**不改任何页面跳转**：没有新增入口、没有 `navigateTo` / `switchTab` 变更。家长端 tabBar 4 项、老师端无 tabBar 的现状不受影响。图片刷新只重建 URL，不触发路由。

### 数据库迁移与 AutoMigrate

- 本次**不新增表、不新增字段**，因此没有手工 `ALTER TABLE`。
- 但迁移机制改变：基线 SQL 必须从**已执行完全部 `docs/migrations/*.sql` 的实际开发库**导出（`mysqldump --no-data`），而不是仅按 GORM 模型推断；否则会漏掉手工索引/列。
- 基线在 staging 空库执行 `goose up` 后，必须用 `go run ./cmd/dbcheck <table>` 核对关键表（`files`、`growth_media`、`album_photos`、`notices`、`leaves`、`health_check_media`、`tasks`、`users`、`students`）的索引与行数。
- `AutoMigrate` 只增不改的局限由此被版本化迁移取代；后续任何 schema 变更都必须写 goose 迁移，并在 tasks 中登记 `dbcheck` 验证。

### 契约字段溯源

本次**不新增响应字段**。修改的是既有字段的取值语义（长期公开 URL → 1 小时签名 URL），字段名、类型、包络不变：

| 字段 | 出现位置 | 变更 | 业务场景 / 溯源 |
|---|---|---|---|
| `file_url` | `api-common.md` 上传响应；`api-teacher/06-growth.md`、`api-parent/04-growth.md`、`api-teacher/10-album.md`、`api-parent/13-class-albums.md`、通知/请假/晨检等 | 值改为 1 小时签名 URL | 成长记录详情与相册网格、通知附件、请假凭证、晨检照片、任务媒体；均为既有 UI，无新增 UI |
| `thumb_url` | 同上 | 值改为 1 小时签名 URL | 列表卡片缩略图；既有 UI |
| `cover_url` | `api-admin/22-albums.md`、公告 | 值改为 1 小时签名 URL | 相册封面、公告封面；既有 UI |
| `avatar` | 用户/学生/接送人 | 值改为 1 小时签名 URL | 头像；既有 UI |
| `face_url` | 学生人脸注册（管理/老师端） | 值改为 1 小时签名 URL | 人脸注册照片；既有 UI |

- 没有设计稿新增元素，因此不涉及 `design-data/` 变更。
- `media_url`（亲子任务）同样按上述规则处理。
- 不新增字段的理由：本次只改变「如何访问已存在的媒体」，不引入新的媒体语义；object key 是内部表示，不暴露为新字段。

### 接口粒度

无列表/详情字段增删；列表接口仍只返回既有字段，不因签名改造夹带详情字段。

## 发布时序

```
维护者            GitHub Actions                 ECS / 阿里云             微信后台
  |  push tag v...      |                             |                       |
  |-------------------->|                             |                       |
  |                     | 1. PR 门禁 / 构建后端镜像    |                       |
  |                     |--------------------------->| ACR push              |
  |                     | 2. 构建 admin 静态包         |                       |
  |                     | 3. 构建小程序 test/prod 包   |                       |
  |                     | 4. 自动部署 staging          |                       |
  |                     |--------------------------->| compose pull/up       |
  |                     |<--------------------------| /ready OK             |
  |  人工批准 production |                             |                       |
  |-------------------->| 5. 确认 RDS 快照             |                       |
  |                     | 6. goose up                  |                       |
  |                     | 7. 部署 prod                  |                       |
  |                     |--------------------------->| compose pull/up       |
  |                     |<--------------------------| /ready OK / 失败回滚   |
  |                     | 8. miniprogram-ci 上传体验版  |                       |
  |                     |-------------------------------------------------->|
  |                     |                             |                       | 人工审核/发布
```

回滚状态：

```
[staging 就绪] --批准--> [prod 部署中] --/ready 失败--> [切回上一镜像 + 上一 admin symlink]
                                             |
                                             +-- 成功 --> [prod 新版本]
```

## Migration Plan

1. **前置**：轮换 RDS/Redis/模型 key/JWT secret；删除已跟踪明文凭据；创建 ECS/VPC/RDS/Redis/OSS/ACR/SLS；启动 ICP 备案；配置 RAM 子账号与 GitHub Environments。
2. **代码加固 + 容器化**：优雅停机、`/health` & `/ready`、CORS、可信代理、临时文件清理、face 关闭、Dockerfile；先在本地与 staging 验证。
3. **迁移基线**：从实际开发库导出 schema，生成 `00001_baseline.sql`；staging 空库 `goose up` + `dbcheck`；确认后再用于生产。
4. **媒体私有化**：切换预发桶为私有，落地签名解析器与三端图片刷新；用 staging 全量遍历媒体读取路径。
5. **admin 静态化 + Nginx**：`nuxt generate`、字体替换、Nginx 站点与 TLS。
6. **CI/CD**：GitHub Actions 构建/推送/部署；staging 自动、production 审批；演练回滚。
7. **小程序**：替换域名、配置白名单、`miniprogram-ci` 上传测试包；生产包在 prod 后端可用后上传。
8. **可观测性与备份**：SLS、CloudMonitor 告警、RDS 备份策略；执行一次恢复演练并记录耗时。

**回滚策略**：后端镜像 tag 回退；admin symlink 回退；数据库迁移只向前，破坏性变更靠发布前快照恢复；小程序在微信后台回退版本。

## Risks / Trade-offs

| 风险 | 缓解 |
|---|---|
| 单 ECS 单点故障 | 托管数据层 + 每日备份 + 季度恢复演练；接受本期可用性上限 |
| GitHub Actions → ACR 跨境推送慢/失败 | 首次实测；不可用则评估云效 Flow（先 PoC），不改变发布契约 |
| 媒体签名路径遗漏导致图片不可见 | 私有桶失败关闭；staging 遍历三端所有媒体页面；日志告警 403 |
| 历史完整 URL 与新 object key 混存 | 解析器同时接受两种形态；字段重命名留作后续清理 |
| 移除 AutoMigrate 后 schema 漂移 | 基线从实际库导出；`dbcheck` 核对；后续变更必须写 goose 迁移 |
| staging 与 prod 同机争内存 | 监控内存；不足时把 staging MySQL/Redis 迁出或升配 |
| 微信审核周期 | API 向后兼容纪律（ADR-0009）；测试包与生产包分离 |
| admin 静态化破坏 Nitro 特性 | 已确认无 server route；staging 全页面回归 |
| ICP 备案周期 | 立项即启动；未完成不发小程序生产版 |

## Open Questions

- ACR 命名空间与镜像仓库名（可在实施时按账号规范确定，不影响 specs）。
- SLS 日志保留期与付费档位（默认 30 天；可按实际账单调整）。
- 未来是否引入 CDN（本期明确不引入；引入时需支持回源签名）。
