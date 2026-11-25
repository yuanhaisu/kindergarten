# Kindergarten Server

幼儿园管理系统后端服务，基于 Go 语言构建，服务于微信小程序（家长端 / 教师端）与 Web 管理后台。

## 技术栈

| 层级 | 选型 |
|------|------|
| HTTP 框架 | Gin |
| ORM | GORM + MySQL |
| 缓存/会话 | go-redis v9 |
| 认证 | JWT (golang-jwt/v5) + 微信 code2session |
| 鉴权 | Casbin RBAC |
| 配置 | INI (gopkg.in/ini.v1) |
| 日志 | log/slog 封装 |
| 依赖注入 | Google Wire |
| 对象存储 | 阿里云 OSS |

## 目录结构

```
cmd/api/main.go              # 入口：加载配置 → 初始化日志 → Wire 注入 → Run
configs/
  app.ini                    # 应用配置（端口、DSN、日志级别等）
  rbac_model.conf            # Casbin RBAC 模型
  rbac_policy.csv            # Casbin 初始策略
internal/
  app/                       # Wire 注入器、Provider、路由组装
  config/                    # INI 配置加载，环境变量覆盖
  auth/                      # JWT 签发/解析、微信 code2session、refresh token
  middleware/                # RequestID、CORS、JWT 鉴权、Casbin、日志 context
  rbac/                      # Casbin Enforcer 初始化
  domain/                    # 实体模型（User 等）
  repository/                # 数据访问层（GORM）
  service/                   # 业务逻辑层
  handler/http/              # HTTP Handler（Gin 绑定 + 响应）
  storage/oss/               # 阿里云 OSS 封装（预签名、上传）
pkg/
  log/                       # slog 初始化与 context 传递
  response/                  # 统一 JSON 信封
```

## 快速开始

### 前置条件

- Go 1.25+
- MySQL 8.0+
- Redis 6+

### 配置

1. 复制并编辑配置文件：

```bash
cp configs/app.ini configs/app.local.ini
# 编辑 app.local.ini 填入实际的 DSN、密钥等
```

2. 或通过环境变量覆盖敏感配置：

```bash
export MYSQL_DSN="user:pass@tcp(127.0.0.1:3306)/kindergarten?charset=utf8mb4&parseTime=True&loc=Local"
export JWT_SECRET="your-jwt-secret"
export JWT_PLATFORM_SECRET="your-platform-jwt-secret"   # 必须与 JWT_SECRET 不同
export REDIS_ADDR="127.0.0.1:6379"
export WECHAT_APP_ID="your-wechat-app-id"
export WECHAT_SECRET="your-wechat-secret"
```

### 运行

```bash
# 下载依赖
go mod tidy

# 启动服务
go run cmd/api/main.go

# 或指定配置文件路径
CONFIG_PATH=configs/app.local.ini go run cmd/api/main.go
```

### 测试

无数据库依赖的包（CI 每次 PR 同样会跑）：

```bash
go test ./internal/app ./internal/config ./internal/middleware ./pkg/... ./cmd/internal/seedutil ./cmd/setadmin -count=1
```

#### 认证测试（authharness，需要 MySQL 8）

`internal/testutil/authharness` 是登录/刷新/切换/绑定/身份支持的单一 HTTP 缝：
真实 Gin + 真实 MySQL 迁移 + 假微信 + 假 Redis，**不需要 Redis 服务**。

CI 的 `authharness` job 会起 MySQL 8 service，先跑 `migrate` 与 `seedbase`，
再跑测试。本地等价命令（Git Bash / Linux）：

```bash
# 仓库根目录：准备配置模板（已有 configs/app.ini 时跳过这行）
cp server/configs/app.ini.example server/configs/app.ini

# 可选：没有现成 MySQL 8 时，用与 CI 相同的镜像起一个（自动创建测试库）
docker run --rm -d --name kdg-authharness-mysql \
  -e MYSQL_ROOT_PASSWORD=root -e MYSQL_DATABASE=kindergarten_test \
  -p 3306:3306 mysql:8.0
# 已有 MySQL 但库不存在时，先执行：CREATE DATABASE kindergarten_test;

cd server
export CONFIG_PATH=configs/app.ini
export KDG_TEST_REQUIRE_MYSQL=1   # 缺库/漏配直接失败；CI 必须设置
export MYSQL_DSN='root:root@tcp(127.0.0.1:3306)/kindergarten_test?charset=utf8mb4&parseTime=True&loc=Local'
go run ./cmd/migrate up
go run ./cmd/seedbase

# 测试步骤不要设置 MYSQL_DSN：harness 把配置里的 DSN 视为「开发库」，
# 显式测试库与它同名会被拒绝（防止误跑开发库）。
unset MYSQL_DSN
export KDG_TEST_MYSQL_DSN='root:root@tcp(127.0.0.1:3306)/kindergarten_test?charset=utf8mb4&parseTime=True&loc=Local'
go test ./internal/testutil/authharness -count=1 -timeout 600s
```

- CI 显式测试库名为 `kindergarten_test`：库名须带 `test` 标记且不同于开发库（#207
  的测试库守卫），测试内建库需要 CREATE DATABASE 权限（CI 使用 root 账号，已具备）。
- 未设置 `KDG_TEST_MYSQL_DSN` 时，harness 会从 `CONFIG_PATH` 的 DSN 派生
  `<库名>_test_<分支+测试包>` 临时库并自动跑 goose 迁移；与开发库同名会被拒绝。
- 未设置 `KDG_TEST_REQUIRE_MYSQL` 且 MySQL 不可达时测试会 skip（本地无 DB 也能开发）；
  CI 一律设置该变量，任何 skip 都视为失败。
- 运行时长基线：本地（Windows + 本机 MySQL）冷启动含 goose 迁移约 255s；按 CI 顺序
  （migrate + seedbase 完成后）测试约 185s，新建并迁移空测试库另约 21s。CI job 另含
  MySQL service 启动，整体约 4 分钟（超时上限 15 分钟）。明显变慢时按测试文件拆
  job，不降低覆盖。

#### 首批 service 测试（#208，需要 MySQL 8）

`backend` job 只跑无数据库依赖的包，`internal/service` 此前整体缺席 CI。这里先接入
**一小批命名测试**，按三个业务域分组（CI 用显式测试名，不用通配扩大批次）：

| 域 | 测试（CI 选择器） | 依赖 |
| --- | --- | --- |
| 认证 / 园所 | `TestKindergartenStatusService_*`、`TestKindergartenEntryFieldsDefaultToSharedChannel`、`TestKindergartenFindByPublicCodeReturnsActiveOnly`、`TestGeneratePublicCode*`、`TestBackfillPublicCodesFillsMissingOnly`、`TestSetPublicCodeIfEmpty*`、`TestEntryServiceFindByPublicCode` | authharness / testdb |
| 家长档案 | `TestParentClassInfo_*`、`TestParentStudentProfile_*`、`TestParentClassProfile_*` | testdb |
| 平台服务 | `TestPlatformService_*`、`TestPlatformOnboard_*` | testdb + fakeredis |

- CI 复用 `authharness` job 的 MySQL 8 service 与 `migrate`/`seedbase` bootstrap；
  testdb 在测试内派生 `kindergarten_test_<分支|包哈希>` 库，与 `kindergarten_test`
  相互隔离。**不新增 Redis service**：平台服务测试用 `fakeredis`，园所状态测试不碰 Redis。
- 本地等价命令（Git Bash / Linux）：

  ```bash
  cd server
  export CONFIG_PATH=configs/app.ini
  export KDG_TEST_REQUIRE_MYSQL=1
  export KDG_TEST_MYSQL_DSN='root:root@tcp(127.0.0.1:3306)/kindergarten_test?charset=utf8mb4&parseTime=True&loc=Local'
  # 与 CI 步骤的 -run 选择器逐字一致；改批次时两处同步，避免本地多跑/少跑。
  go test ./internal/service -count=1 -timeout 300s -p 1 \
    -run '^(TestKindergartenStatusService_SuspendedReadsDB|TestKindergartenEntryFieldsDefaultToSharedChannel|TestKindergartenFindByPublicCodeReturnsActiveOnly|TestGeneratePublicCode\w*|TestBackfillPublicCodesFillsMissingOnly|TestSetPublicCodeIfEmpty\w*|TestEntryServiceFindByPublicCode|TestParentClassInfo_\w+|TestParentStudentProfile_\w+|TestParentClassProfile_\w+|TestPlatformService_\w+|TestPlatformOnboard_\w+)$'
  ```

- 门禁：步骤以 `-json` 采集，出现任何 `"Action":"skip"` 即失败（缺库/漏配不允许静默
  跳过）；没有任何 `"Action":"pass".*"Test":` 也失败（防止选择器与测试名漂移后空跑）。
- 串行与清理：`-p 1` 串行；测试只硬删自己写入的行（`Unscoped`/按固定夹具 ID），
  不整表清空、不 DROP 库。
- 超时与 flake：`-timeout 300s`（本地基线 ~10 倍余量），**不重试**。出现偶发失败按
  真实缺陷处理（修复或显式隔离并记录 ticket），禁止加 retry 掩盖抖动。
- 运行时长基线（2026-11-23，Windows + 本机 MySQL 8.4，26 个测试全绿）：
  `ok kindergarten-server/internal/service 25.5s`；含冷编译 wall time 约 44s，
  其中首个 testdb 用例 19.6s 含派生库 goose 迁移。CI 在同 job 已有镜像与迁移缓存，
  预期增量 < 1 分钟；批次变慢时优先拆 job，而不是删除测试。
- 本批次刻意封顶：其余 service 测试仍留在本地/后续 ticket，**不要**用
  `go test ./internal/service` 无选择器地整体搬进 CI。

### Wire 代码生成

```bash
# 安装 wire 工具（一次性）
go install github.com/google/wire/cmd/wire@latest

# 生成注入代码
cd internal/app && wire
# 或
go generate ./internal/app
```

### 测试数据库

需要真实 MySQL 的测试统一调用 `testdb.Open(t)`，不要各自读配置连库：

```bash
# 指向一台独立测试 MySQL；库名必须含 test，账号需要 CREATE 权限
KDG_TEST_MYSQL_DSN='user:pass@tcp(127.0.0.1:3306)/kindergarten_test?charset=utf8mb4&parseTime=True' \
  go test ./internal/service/ -run TestParentGrowthLike
```

- 设置 `KDG_TEST_MYSQL_DSN` 时，helper 会派生「分支 + 测试包」隔离的测试库并自动执行 goose 迁移，不改动开发库；
- 不设置时保持原有本地行为（读 `configs/app.ini`），无配置则跳过；设置
  `KDG_TEST_REQUIRE_MYSQL=1` 后跳过改为失败（CI 用，缺库不得静默变绿；上面的
  service 批次命令依赖这一点）；
- 源码、行为矩阵与护栏规则见 [`internal/testutil/testdb/README.md`](internal/testutil/testdb/README.md)。

## API 概览

| 方法 | 路径 | 说明 | 认证 |
|------|------|------|------|
| GET | /health | 健康检查 | 否 |
| POST | /api/v1/auth/wechat/login | 微信小程序登录 | 否 |
| POST | /api/v1/auth/refresh | 刷新令牌 | 否 |
| GET | /api/v1/me | 获取当前用户信息 | Bearer JWT |

## 统一响应格式

**成功：**

```json
{
  "success": true,
  "data": { },
  "request_id": "uuid"
}
```

**失败：**

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "人类可读描述"
  },
  "request_id": "uuid"
}
```
