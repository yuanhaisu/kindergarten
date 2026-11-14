# Tasks

## 0. 前置与安全基线

- [ ] 0.1 轮换 RDS 密码、Redis 口令、模型 API key、JWT secret；验证旧凭据失效、新凭据可连接
- [x] 0.2 删除已跟踪的明文凭据（`cc.ps1`、`start.bat`）并清理 `.githooks/pre-commit` 中内联的凭据片段；验证 `node scripts/check-secrets.mjs` 通过且 `git grep` 不再命中
- [ ] 0.3 确认生产地域、主域名与 ICP 备案主体；启动备案；创建 ACR/ECS 的 RAM 子账号并验证最小权限
- [ ] 0.4 创建 ECS/VPC/RDS/Redis/OSS/ACR/SLS 资源与 DNS；验证 ECS 可内网访问 RDS/Redis、公网仅 80/443 可达

## 1. 后端：运行时加固与容器化

- [x] 1.1 在 `cmd/api/main.go` 增加 SIGTERM/SIGINT 优雅停机（停止接收新请求 + 有界超时）；用单元测试或本地发送 SIGTERM 验证在途请求完成后再退出（TC-NEW-OPS-0003/0004）
- [x] 1.2 增加 `/ready`（检查 MySQL/Redis），保留 `/health` 为纯存活检查；用测试验证依赖缺失时 `/ready` 失败、`/health` 成功（TC-NEW-OPS-0001/0002）
- [x] 1.3 [P] 收紧 CORS 到 `https://admin.<domain>`，配置 Gin `SetTrustedProxies`；验证跨域预检仅放行 admin 域名，日志中的 ClientIP 来自 `X-Forwarded-For`
- [x] 1.4 [P] 生产日志改为 JSON + 脱敏手机号/openid；导出临时文件增加启动清理与每日清理；生产配置 `[face].enabled=false`；验证日志样例无明文、临时目录清理生效、识别 worker 不启动（TC-NEW-OPS-0007/0008）
- [ ] 1.5 增加 `server/Dockerfile`（多阶段、`CGO_ENABLED=0`、`TZ=Asia/Shanghai`）、`deploy/config/app.ini.tmpl`、Casbin 容器内绝对路径与 compose 文件；验证 `docker build` 成功、容器以渲染后的 `app.ini` 启动并通过 `/ready`
- [ ] 1.6 后端验证：`go build ./...`、`go test ./...`、`make lint` 全部通过；逐条核对 runtime-operations 的 OPS-0001~0006、0008 场景

## 2. 后端：goose 迁移基线

- [x] 2.1 从已执行完全部 `docs/migrations/*.sql` 的开发库导出 `server/migrations/00001_baseline.sql`；在空 staging 库执行 `goose up` 并验证成功
- [x] 2.2 移除启动期 `AutoMigrate` 与种子；新增显式、幂等的 `cmd/seedbase`；验证应用启动不执行 DDL、种子命令重复执行不产生重复数据
- [x] 2.3 创建迁移账号（DDL）与应用账号（DML）；验证应用账号无法建表/改表，迁移账号可执行 `goose up`
- [x] 2.4 迁移验证：staging 空库 `goose up` 后用 `go run ./cmd/dbcheck <table>` 核对 `files`、`growth_media`、`album_photos`、`notices`、`leaves`、`health_check_media`、`tasks`、`users`、`students` 的索引与行数；核对 OPS-0005/0006

## 3. 后端：媒体私有化与签名

- [x] 3.1 扩展 `storage/oss`：支持从 object key 或历史完整 URL 解析对象引用，`PresignGet` 支持 `x-oss-process` 参数；用单元测试覆盖 key/URL 两种输入与缩略图签名
- [x] 3.2 修改 `file_service.go`：新上传写 object key（缩略图写 key + 处理参数），响应返回 1 小时签名 URL；验证上传响应 URL 可访问、去掉签名后 403（TC-NEW-MAS-0012/0014）
- [x] 3.3 [P] 成长与相册读取路径统一签名（growth feed/detail/media、album list/detail/cover）；用 API 级测试验证响应 URL 带签名且能访问
- [x] 3.4 [P] 通知/请假/晨检/任务/接送/用户与学生头像/管理端文件列表的读取路径统一签名；逐端点验证响应 URL 带签名且旧数据（完整 URL 形态）也能签名
- [x] 3.5 处理已废弃的 `POST /teacher/growth/media/upload` 入口：归一化输入并返回签名 URL；补充媒体签名测试，覆盖 TC-NEW-MAS-0012~0014

## 4. 契约：backend/api 语义更新

- [x] 4.1 [P] 更新 `backend/api/api-common.md`：`file_url` / `thumb_url` 说明改为 1 小时签名 URL，字段形状与包络不变
- [x] 4.2 [P] 更新引用媒体 URL 的业务契约（成长、相册、通知、请假、晨检、任务、管理端文件）示例与说明；不新增/删除字段
- [x] 4.3 契约验证：`git diff backend/api/` 仅包含注释/示例变化；对照 design 的字段溯源表确认无新增字段；`openspec validate establish-deployment-pipeline` 通过

## 5. admin：静态化与媒体刷新

- [x] 5.1 将 admin 构建改为 `nuxt generate`，Nginx 配置 SPA fallback 与同源 `/api/v1` 反代；验证产物包含 `index.html`、页面刷新不 404
- [x] 5.2 移除 `nuxt.config.ts` 中的 `fonts.googleapis.com`，改用本地/国内可达字体；验证页面无外部字体请求、视觉无明显变化
- [x] 5.3 在 `pages/albums/index.vue`、`pages/announcements/index.vue`、`pages/teachers/index.vue`、`pages/students/index.vue` 等媒体页面增加图片加载失败后的数据刷新；用 Playwright 真实点击进入页面并验证图片可加载（TC-NEW-MAS-0012/0013）
- [ ] 5.4 admin 验证：`npm run generate` 成功；Playwright 冒烟覆盖登录、相册、公告、学生/教师头像；逐条核对 media-assets 相关场景

## 6. teacher 小程序

- [x] 6.1 在 `pages/growth/create.vue`、`pages/growth/detail.vue`、`pages/growth/index.vue`、`pages/album/index.vue`、`pages/notice/index.vue`、`pages/notice/create.vue`、`pages/index/index.vue`、`pages/doctor/checkin.vue`、`pages/leave/index.vue`、`pages/tasks/detail.vue`、`components/AvatarBadge.vue` 增加图片失败刷新；提交仍使用 `file_id`，不得改用 URL
- [ ] 6.2 更新 `teacher/.env.test`、`teacher/.env.production` 与 `src/config/env.js` 的真实域名；验证构建产物中的 `VITE_API_BASE_URL` / `VITE_UPLOAD_URL` 指向对应环境
- [ ] 6.3 teacher 验证：`npm run build:mp-weixin` 成功；用 `wechat_automator` 的 tap/input 真实走一遍成长记录发布与详情、相册、通知、请假、头像加载（不得用 `wechat_navigate` 替代交互）；核对 TC-NEW-MAS-0012/0013

## 7. parent 小程序

- [x] 7.1 [P] 在 `pages/growth/index.vue`、`pages/growth/detail.vue`、`pages/growth/album.vue`、`pages/growth/report.vue`、`pages/album/index.vue`、`pages/album/detail.vue`、`pages/class/index.vue`、`pages/pickup/qrcode.vue`、`pages/tasks/detail.vue`、`pages/tasks/wall.vue`、`pages/leave/create.vue`、`pages/index/index.vue`、`pages/student/index.vue`、`pages/points/index.vue`、`pages/profile/index.vue`、`components/AvatarBadge.vue` 增加图片失败刷新
- [ ] 7.2 [P] 更新 `parent/.env.test`、`parent/.env.production` 与 `src/config/env.js` 的真实域名；验证构建产物指向对应环境
- [ ] 7.3 parent 验证：`npm run build:mp-weixin` 成功；用 `wechat_automator` 真实点击走一遍成长时光、相册、接送二维码、任务、头像加载；核对 TC-NEW-MAS-0012/0013

## 8. CI/CD 与基础设施

- [ ] 8.1 编写宿主机 Nginx 配置：TLS 1.2+/HSTS/HTTP 跳转/`limit_req`/`client_max_body_size 25m`/静态 SPA/`/api/v1`、`/health`、`/ready` 反代；`nginx -t` 通过
- [ ] 8.2 编写 prod/staging 两套 Compose 与目录/端口/symlink 约定；`docker compose config` 通过，staging 能按约定启动
- [ ] 8.3 编写 PR 门禁 workflow：`make check`、`check-tokens`、`check-secrets`、admin 构建、两个小程序构建；在 PR 上验证全部通过
- [ ] 8.4 编写 tag 发布 workflow：构建并推送 ACR、自动部署 staging、production 人工批准、发布前快照、`/ready` 失败自动回滚；用一次 staging 发布与一次模拟失败验证回滚（TC-NEW-DEP-0001~0006、0009/0010）
- [ ] 8.5 接入 `miniprogram-ci` 上传测试/生产体验版，私钥存 GitHub Secrets；验证测试包上传成功且指向 `test-api`
- [ ] 8.6 基础设施验证：staging 端到端发布成功；production 流程在批准前不部署；回滚演练恢复上一镜像与上一 admin symlink；核对 DEP-0007/0008 的兼容规则已写入发布检查

## 9. 可观测性、备份与安全 runbook

- [ ] 9.1 [P] 接入 SLS：采集容器 stdout 与 Nginx 日志，保留 30 天；验证可按 `request_id` 检索一次请求
- [ ] 9.2 [P] 配置 CloudMonitor：每分钟探测 `/health`、连续 2 次失败告警；ECS/RDS/Redis 资源告警到短信/邮件；模拟一次探测失败验证告警到达
- [ ] 9.3 配置 RDS 每日备份 + binlog、每周逻辑备份到 OSS、发布前快照；恢复到临时实例并通过 `dbcheck`，记录 RTO（TC-NEW-OPS-0011/0012）
- [ ] 9.4 按安全清单核对：安全组仅 80/443、SSH 限管理员 IP、RDS/Redis 无公网、TLS 1.2+/HSTS、RAM 子账号；用外部端口扫描验证无多余暴露
- [ ] 9.5 编写 `docs/runbook/deployment.md`：发布、回滚、恢复、告警处置、证书续期、常见故障；按文档原样执行一次并修正不可执行步骤
- [ ] 9.6 运维验证：逐条核对 runtime-operations 的 OPS-0007~0012 与 release-pipeline 的 DEP-0009/0010

## 10. 集成验收

- [x] 10.1 在 `docs/tests/TEST_CATALOG.md` 登记新 TC：`TC-NEW-DEP-0001~0010`、`TC-NEW-OPS-0001~0012`、`TC-NEW-MAS-0012~0014`；验证 ID 可在目录中检索到
- [ ] 10.2 后端集成测试：优雅停机、`/health` & `/ready`、CORS、签名 URL（含过期）、迁移顺序；`go test ./...` 通过
- [ ] 10.3 管理端集成验证：静态构建 + Playwright 真实点击相册/公告/头像页面，图片可加载；无外部字体请求
- [ ] 10.4 小程序集成验证：两端真实 tap/input 走媒体页面；关闭页面后重新进入验证签名 URL 刷新；核对 tabBar 差异（家长端 4 tab、老师端无 tabBar），本次无跳转变更
- [ ] 10.5 逐条对照 `specs/` 的每条 Requirement/Scenario 记录验证证据；未通过项不得宣布完成或 archive
- [ ] 10.6 收工清理：按 PID 杀掉残留的 `dev:mp-weixin` watcher 与 MCP 自动化进程；确认工作区无构建产物脏文件
