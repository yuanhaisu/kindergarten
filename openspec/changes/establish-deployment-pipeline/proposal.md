# Proposal

## Why

仓库目前没有任何可执行的部署资产：没有 Dockerfile、没有 CI、没有 Nginx 配置、没有进程守护、没有部署脚本；生产域名仍是 `xxx.com` 占位符，部署只存在于两份 draft 文档里。项目处于绿地阶段（尚无正式生产园所），现在是把「构建 → 发布 → 回滚 → 恢复」变成可重复流程的最低成本时点。

同时，仓库 HEAD 仍包含明文凭据（`cc.ps1`、`start.bat`、`.githooks/pre-commit`），且 RDS 凭据已进入 git 历史。首次上线前必须完成轮换与清理，否则部署体系会把已知泄漏固化进生产。

## What Changes

- 建立**单台 ECS + 宿主机 Nginx + Docker Compose** 的部署拓扑（ADR-0007）：生产仅 `api`，预发为 `api + mysql + redis`；生产 MySQL/Redis 使用阿里云托管服务；首期不引入 SLB、K8s、CDN、WAF。
- 后端容器化，并完成上线前代码加固：优雅停机、`/health` 存活与 `/ready` 依赖检查、CORS 收紧、Gin 可信代理、导出临时文件清理、生产关闭 `[face]` worker。
- 引入 **goose 版本化迁移**（ADR-0008），以当前模型生成基线；移除启动期 `AutoMigrate` 与种子；迁移 job 与运行账号权限分离（DML / DDL 分离）。
- **BREAKING** 幼儿媒体改为**私有 OSS 桶 + 1 小时签名 GET URL**（ADR-0010）：数据库存 object key，读取时签名；`backend/api/api-common.md` 中 `file_url` 的语义从长期公开 URL 变为短期签名 URL（字段形状不变）。teacher / parent / admin 三端都必须处理 URL 过期与刷新。
- admin 改为 `nuxt generate` 纯静态产物，由 Nginx 托管；移除 `fonts.googleapis.com` 依赖；`NUXT_PUBLIC_API_BASE` 保持默认 `/api/v1` 同源。
- 宿主机 Nginx：TLS 1.2+ / HSTS / HTTP→HTTPS / `limit_req` / 静态站点 / 反向代理 / `client_max_body_size 25m`。
- GitHub Actions：tag 驱动发布（`vYYYY.MM.DD-N`）；staging 自动部署，production 需人工批准；后端按镜像 tag 回滚，admin 按静态版本 symlink 回滚。
- 配置模板 + GitHub Environment Secrets：部署时渲染 `/etc/kindergarten/{env}/app.ini`（0600）；轮换泄漏凭据并移除已跟踪的明文凭据文件。
- 两个小程序：替换真实域名、配置微信服务器域名白名单、`miniprogram-ci` 上传体验版；「提交审核 / 发布」保留人工。
- 可观测性与恢复：SLS 采集 JSON 日志与 Nginx 日志；CloudMonitor 站点与资源告警；RDS 自动备份 + binlog + 每周逻辑备份 + 季度恢复演练。
- 确立 **API 向后兼容最旧线上小程序版本** 的发布纪律（ADR-0009）。

## Capabilities

### New Capabilities

- `release-pipeline`: 版本发布的触发、环境隔离、人工审批、回滚，以及「后端 API 向后兼容最旧线上小程序版本」的发布约束。
- `runtime-operations`: 单实例运行约束、健康检查与优雅停机、版本化数据库迁移的执行顺序、日志/监控/备份恢复的最低运行要求。

### Modified Capabilities

- `media-assets`: 媒体文件的存储与访问从公共 URL 改为私有桶 + 短期签名 URL；可见性仍由业务引用决定，但 URL 有效期与刷新成为契约行为。

## Impact

- **本次将修改 `backend/api/`，属于契约语义变更**：`backend/api/api-common.md` 的 `file_url` 从「长期公开 URL」改为「短期签名 URL」，字段形状与包络不变；各业务契约中引用 `file_url` 的示例需同步说明有效期。不新增/删除字段。
- **Go 后端**：`cmd/api/main.go`（优雅停机）、`internal/app/provider.go`（移除启动迁移/种子、迁移 job 入口）、`internal/app/router.go`（`/ready`）、`internal/middleware/`（CORS、可信代理）、`internal/storage/oss/`（签名 GET）、`internal/service/file_service.go`（存 object key）、`internal/service/export_task_service.go`（临时文件清理）、`internal/service/recognition_queue.go`（生产关闭）、`configs/`（模板与 Casbin 路径）、`migrations/`（goose 基线）、`cmd/`（显式种子命令）。
- **admin**：`nuxt.config.ts`（静态生成、字体）、构建与发布产物布局；页面业务逻辑不改。
- **teacher / parent**：`.env.test` / `.env.production` 真实域名；媒体 URL 过期后的刷新；`miniprogram-ci` 上传配置；微信后台服务器域名白名单。
- **基础设施**：阿里云 ECS / VPC / 安全组 / RDS / Redis / OSS / ACR / SLS / CloudMonitor、DNS 与证书、GitHub Environments 与 Secrets。云资源在控制台创建，本期不引入 Terraform。
- **测试**：健康检查、优雅停机、迁移顺序、签名 URL 过期、admin 静态构建、两个小程序测试/生产构建。
- **不涉及**：业务功能、RBAC 策略、design token、数据库表结构（除迁移执行方式外）。

## Out of Scope

- 多实例 / 水平扩展 / SLB / K8s / CDN / WAF / KMS。
- Terraform 等完整 IaC；云资源仍以控制台 + runbook 管理。
- 小程序审核提交的全自动 API。
- ICP 备案本身（外部流程，作为上线前置跟踪）。
- git 历史的凭据清理（轮换与删除已跟踪明文文件在范围内，历史重写另行处理）。
- 业务功能与接口形状变更。

## 前置条件与阻塞项

| 项 | 性质 | 处理 |
|---|---|---|
| ICP 备案 | 上线关键路径（1–3 周） | 立项即启动；未完成不得发小程序生产版 |
| 地域 / 域名 / 备案主体确认 | 资源选型 | 当前按华东1（杭州）+ 公司主体设计，实施前确认 |
| 凭据轮换 | 安全前置 | RDS、Redis、模型 key、JWT secret 在第一套生产环境创建前完成 |
| 阿里云 RAM 子账号 | 权限前置 | ACR 推送、ECS 部署各自最小权限 |
| 微信服务器域名白名单 | 发布前置 | 两个小程序后台分别配置 request/upload/download |
