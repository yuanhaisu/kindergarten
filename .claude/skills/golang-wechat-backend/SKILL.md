---
name: golang-wechat-backend
description: Design and implement Go backends for 微信小程序 and 管理后台 using Gin, GORM (MySQL), Redis, Casbin RBAC, INI app config, unified slog-based logging, Alibaba Cloud OSS, Google Wire, REST APIs, JWT + WeChat code2session + Redis session. Use when the user mentions casbin, RBAC, INI configuration, structured logging, gin, gorm, wire, MySQL, Redis, OSS, WeChat login, admin BFF, REST, rate limiting, or API errors.
---

# Go 后端（微信小程序 + 管理后台）

面向 **微信小程序** 与 **Web 管理后台** 的 Go 服务：**HTTP 框架 Gin**，**MySQL 通过 GORM**，Redis，**权限 [Casbin](https://casbin.org)（RBAC）**，**应用配置 INI**，**统一日志（`log/slog` 封装）**，**图片/附件走 [阿里云 OSS](https://www.aliyun.com/product/oss)**，**依赖注入 [Google Wire](https://github.com/google/wire)**；RESTful API、JWT、微信 OAuth（code2session）、统一错误与版本化。目标是可维护的分层架构与一致的契约（请求/响应/错误）。

## 技术栈（固定选型）

| 层级 | 选型 | 说明 |
|------|------|------|
| HTTP | **[gin-gonic/gin](https://github.com/gin-gonic/gin)** | 路由组、中间件、`ShouldBindJSON`、与 `context` 集成 |
| ORM | **[gorm.io/gorm](https://gorm.io)** + `gorm.io/driver/mysql` | 模型迁移、关联预加载、事务；复杂报表可 `Raw` + 占位符 |
| 缓存/会话 | **go-redis v9** | refresh、限流、验证码 |
| 鉴权 | **golang-jwt/jwt/v5** | **认证**（你是谁）；与 Casbin **鉴权**（你能做什么）分离 |
| 权限 | **[casbin/casbin/v2](https://github.com/casbin/casbin)** | RBAC（可扩展 domain/租户）；模型 `.conf` + 策略 CSV 或 DB Adapter |
| 应用配置 | **[gopkg.in/ini.v1](https://gopkg.in/ini.v1)** | **INI 文件**加载非敏感项；密码/密钥仍建议 **环境变量覆盖** |
| 日志 | **`log/slog`（stdlib）+ `pkg/log` 封装** | JSON/文本、级别、**request_id / user_id** 贯穿 context |
| 依赖注入 | **[google/wire](https://github.com/google/wire)** | 编译期生成 `wire_gen.go`，`main` 只调用注入器 |
| 对象存储 | **[aliyun-oss-go-sdk](https://github.com/aliyun/aliyun-oss-go-sdk)** | 图片/文件；预签名直传或服务端 `PutObject` |

## 推荐目录与分层

```
cmd/api/main.go              # 加载 INI、初始化日志、调用注入器、Run
configs/
  app.ini                    # 应用配置（端口、路径、日志级别等）
  rbac_model.conf            # Casbin 模型（非 INI 语法，见 Casbin 文档）
  rbac_policy.csv            # 初始策略（或改 DB Adapter）
internal/
  app/
    wire.go                  # //go:build wireinject — InjectApp + wire.Build（勿提交前忘 generate）
    wire_gen.go              # wire 生成，勿手改
  config/                    # LoadINI → *Config（供 Wire）；敏感项 Env 覆盖
  rbac/                      # NewEnforcer、同步策略、Gin 中间件封装（可选）
  domain/                    # 实体与领域错误（可选）
  repository/              # MySQL/Redis 实现；构造函数 + wire.NewSet
  service/                   # 业务逻辑；构造函数 + wire.NewSet
  handler/http/              # Gin：ShouldBind、调用 service、JSON 信封
  middleware/                # JWT、Casbin、RequestID、日志注入 context
  auth/                      # JWT 签发/校验、微信 token 交换封装
  storage/oss/               # OSS Client 封装：预签名、PutObject、访问 URL（可选）
pkg/
  log/                       # slog 初始化、LoggerFromContext、Gin 绑定
  response/                  # 统一 JSON 信封（可选，可被多服务复用时再抽）
```

**依赖方向**：`handler → service → repository`；`repository` 不引用 `handler`。跨层传递用 **context.Context**（超时、取消、request_id）。

**为何这样分**：Handler 薄（Gin 绑定与状态码），Service 承载规则与编排，Repository 封装 **GORM `*gorm.DB`** 与 Redis，避免在 handler 里直接操作 DB。**Wire** 把构造函数串成一张图，`main` 不出现冗长的 `NewA(NewB(NewC(...)))`，且依赖在编译期可检查。

### 依赖注入（Wire）约定

1. **注入器**：在 `internal/app/wire.go`（build tag `wireinject`）里声明 `func InjectApp(*config.Config) (*gin.Engine, error)`（或返回自定义 `App` 结构体，内含 `Engine`、清理函数等）。
2. **Provider**：每层用 **`wire.NewSet`** 聚合：`ProvideDB`、`NewUserRepository`、`NewUserService`、`NewHandler` 等；可导出 `var RepositorySet = wire.NewSet(...)` 供 `wire.Build` 引用。
3. **生成**：安装 `go install github.com/google/wire/cmd/wire@latest`；在 **`internal/app` 目录执行 `wire`**（会读取带 `wireinject` 的 `wire.go`）。也可在该包增加仅含 `//go:generate go run github.com/google/wire/cmd/wire` 的源文件，便于 `go generate ./internal/app`。
4. **`wire_gen.go`**：仅由工具生成，**禁止手改**；CI 中可校验 `wire` 后再 `git diff --exit-code`。
5. **接口绑定**：对需要 mock 的依赖，用 `wire.Bind(new(Reader), new(*MySQLReader))` 把接口与实现绑在一起。
6. **错误与清理**：`OpenMySQL` 等返回 `error` 时 Wire 会正确传播；需要优雅关闭时可在注入器返回 `(app *App, cleanup func(), err error)` 或在 `App` 上挂 `Close()`。

### 应用配置（INI）

- **文件**：默认 `configs/app.ini`（路径可由启动参数或环境变量 `CONFIG_PATH` 指定）。
- **库**：`gopkg.in/ini.v1` 映射到 `struct`；支持 `MustString`、`MapTo` 等。
- **敏感信息**：DSN、JWT secret、OSS AK/SK **不要写死进仓库**；INI 可留空占位，**启动时用环境变量覆盖**（在 `Load` 里 `os.Getenv` 优先）。
- **多环境**：`app.ini` + `app.local.ini` merge（可选），或分 `configs/dev.ini` / `prod.ini`。

Casbin 的 **model / policy 路径**、日志 **level/format**、HTTP **addr** 等放在 INI 的 `[app]`、`[casbin]`、`[log]` 节（示例见 `references/config-ini.md`）。

### 统一日志（`pkg/log`）

- **实现**：Go **`log/slog`**，封装在 **`pkg/log`**：`Init(level, json)`、`L(ctx)` / `With(ctx, key, val)`，用 **context** 传递带字段的 Logger。
- **Gin**：`RequestID` 之后挂中间件，把 `request_id`（及鉴权后的 `user_id`、`role`）写入 `c.Request.Context()`，handler/service/repo 统一 **`log.L(ctx).Info(...)`**。
- **约束**：禁止业务层直接 `fmt.Println` / 多处新建互不相干的 `slog.Logger`；错误栈仅写服务端日志，**响应仍走统一 JSON 信封**。
- **脱敏**：手机号、token、密码不打完整明文（截断或哈希展示）。

### RBAC（Casbin）

- **职责划分**：**JWT** 完成身份认证；**Casbin** 对「角色 / 用户 → 资源 → 操作」做鉴权。未登录先 **401**，已登录无权限 **403** + 业务码如 `PERMISSION_DENIED`。
- **模型文件**：`rbac_model.conf`（Casbin 自有语法，**不是** INI）；常用 `rbac_model.conf` 或带 **租户** 的 `rbac_with_domains_model.conf`。
- **策略**：开发期可用 **`rbac_policy.csv`**；线上推荐 **GORM Adapter** 或官方持久化 Adapter，策略变更可后台管理并 **`e.LoadPolicy()`** 热载（INI 里 `auto_load` / 定时或订阅通知）。
- **与 Gin 对齐**：中间件内 `e.Enforce(sub, obj, act)` —— `sub` 建议为 **用户 ID** 或 **角色**（与模型一致）；`obj` 可为 **路由模板**（`c.FullPath()`）或 **资源标识**；`act` 为 **HTTP Method** 或业务动作 `read|write|delete`（全项目统一一套）。
- **Wire**：`NewEnforcer(modelPath, policyPath)` 由配置注入；同一 `*casbin.Enforcer` 进程单例。

详细模型/策略示例与中间件片段见 `references/rbac-casbin.md` 与 `go-snippets.md`。

## REST 与版本

- **资源名用复数名词**：`/api/v1/users`、`/api/v1/children/{id}`。
- **动作型接口**（如微信登录）可用 **RPC 风格子路径**：`POST /api/v1/auth/wechat/login`（body 带 `code`），仍保持动词在路径、名词表资源。
- **版本**：URL 前缀 `/api/v1`；破坏性变更进 `v2`。
- **方法语义**：`GET` 安全幂等、`PUT` 全量替换、`PATCH` 部分更新、`POST` 创建或非幂等操作、`DELETE` 删除。
- **分页**：`?page=1&page_size=20` 或 `?cursor=xxx&limit=20`；列表响应带 `items` + `next_cursor` / `total`（二选一或组合，项目内统一即可）。

## HTTP 状态码（约定）

| 场景 | 状态码 |
|------|--------|
| 成功返回 body | 200 |
| 创建成功 | 201 + `Location`（可选） |
| 无 body 成功 | 204 |
| 参数/校验错误 | 422（或 400，**全项目统一一种**） |
| 未认证 | 401 |
| 已登录无权限 | 403 |
| 资源不存在 | 404 |
| 冲突（重复注册等） | 409 |
| 限流 | 429 |
| 服务器错误 | 500 |

## 统一错误响应（全 API 一致）

业务错误用 **稳定机器可读 `code`** + **人类可读 `message`**；生产环境避免把 SQL/堆栈返回给客户端。

```json
{
  "success": false,
  "error": {
    "code": "AUTH_TOKEN_EXPIRED",
    "message": "登录已过期，请重新登录",
    "details": [
      { "field": "refresh_token", "issue": "required" }
    ]
  },
  "request_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

成功时 **`success`: true**，数据放在 `data`（见下一节）。

## API 输入输出规范（契约）

### 通用响应信封

**成功：**

```json
{
  "success": true,
  "data": { },
  "meta": {
    "request_id": "uuid"
  }
}
```

**列表可附加：**

```json
{
  "success": true,
  "data": {
    "items": [],
    "page": 1,
    "page_size": 20,
    "total": 100
  },
  "meta": { "request_id": "uuid" }
}
```

### 请求约定

- **Content-Type**：`application/json; charset=utf-8`。
- **时间**：API 层统一 **RFC3339** 字符串，如 `2025-03-20T08:00:00Z`；DB 存 UTC。
- **ID**：对外统一 string（UUID 或雪花 ID 字符串），避免 JS 大整数精度问题。
- **校验**：Gin `binding` tag（如 `binding:"required,email"`）配合 **validator**；复杂规则在 service 层校验；**禁止**信任客户端传来的权限相关字段（角色、园所 ID 等从 token/session 推导）。

### 认证传递

- **小程序 / 移动端**：`Authorization: Bearer <access_token>`。
- **管理后台（浏览器）**：可选 **HttpOnly Cookie** 存 refresh + 短生命周期 access；或纯 Bearer，视 XSS 面权衡。**敏感操作**配合 CSRF（Cookie 方案时）。

---

## 认证体系分工

| 机制 | 用途 |
|------|------|
| **JWT** | **认证**：无状态访问凭证；claims 含 `sub`(user_id)、**角色**、租户/园所 id、签发时间、过期时间 |
| **Casbin** | **鉴权**：按角色/用户与策略判定接口或资源是否允许；策略与模型路径由 **INI** 配置 |
| **微信 OAuth** | `wx.login` 得 `code` → 服务端 `code2session` → `openid`/`unionid` → 绑定用户 → 签发自有 JWT |
| **Redis Session** | refresh token 轮转、踢下线、后台「单端登录」、短期权限缓存；key 如 `session:{user_id}:{jti}` |

**实践**：access JWT 短（如 15m），refresh 长（如 7d）仅存服务端 Redis + 哈希比对，或 refresh 也为 JWT 但 id 在 Redis 可撤销。**角色以服务端为准**：JWT 内角色仅作提示，关键权限以 DB 角色 + Casbin 策略为准时可于登录后 `e.AddRoleForUser` 同步（策略需与之一致）。

---

## 安全清单

1. **校验所有输入**：路径/query/body；**图片上传**校验扩展名/MIME/文件头（可选）、**单文件大小上限**；OSS 侧用 **Bucket 策略 + 预签名条件**（`Content-Type`、最大长度）缩小滥用面。
2. **清理与编码**：输出到 HTML/PDF 等场景做转义；**统一经 `pkg/log`** 打日志，敏感字段脱敏。
3. **参数化查询**：GORM 的 `Where("id = ?", id)`、`Create` 等默认安全；**禁止** `Raw(fmt.Sprintf(...))` 或拼接字符串；原生 SQL 必须用 `?` 占位符。
4. **限流**：按 IP + 用户 ID（登录后）双维度；登录、发短信等接口更严。
5. **生产仅 HTTPS**；HSTS；Cookie 设 `Secure`、`SameSite`。
6. **密钥**：JWT secret、MySQL、Redis、微信 secret、**OSS AccessKey（优先 RAM 子账号 + 最小权限）** 来自环境变量或密钥管理，不入库不入 Git；INI 中仅存占位或本地开发默认值（**勿提交生产机密**）。
7. **权限**：Casbin 策略变更需审计；管理端接口对「改策略」本身再做 **超级管理员** 或双因子保护。

---

## 图片与文件存储（阿里云 OSS）

**官方 Go SDK**：`github.com/aliyun/aliyun-oss-go-sdk/oss`。

### 上传路径选型

| 方式 | 适用 | 说明 |
|------|------|------|
| **预签名 URL（PUT）** | 管理后台（XHR/fetch PUT）、部分 App | 服务端只签发 URL 与 headers；**不经过**应用服务器带宽；需控制过期时间与可写 **object key** |
| **PostObject + Policy** | 微信小程序 `wx.uploadFile`（表单 POST）等 | 服务端返回 `policy`、`signature`、`host`、`key` 等字段；Policy 限制 **max size**、路径前缀、`Content-Type` |
| **服务端转存** | 小文件、需病毒扫描/压缩/水印 | Gin `multipart` → `bucket.PutObject`；注意 **body 大小限制** 与超时 |

**推荐**：大流量图片优先 **直传 OSS**（预签名或 Post Policy）；敏感场景再经服务端处理。

### 对象 Key 与元数据

- **Key 规范**：`{env}/{tenant_or_school_id}/{yyyy}/{mm}/{uuid}.jpg` 等，避免可猜、避免覆盖；**禁止**把用户原始文件名直接当 key。
- **访问控制**：默认 **Bucket 私有**；展示用 **签名 GET URL**（短 TTL）或 **CDN + URL 鉴权**；公开头像类再评估 `public-read` 与盗链防护。
- **元数据**：`Content-Type` 必设；可写 `x-oss-meta-*` 存业务 id（仅作辅助，权威数据仍在 MySQL）。

### 运维与合规

- **CORS**：后台 Web 直传时，在 OSS 控制台为 Bucket 配置对应 **Origin**、**Allowed Methods**（PUT/POST）、**ExposeHeaders**（如 `ETag`）。
- **限流**：`/uploads/presign` 等接口按用户 **严格限流**，防刷签名链接。
- **RAM**：使用子账号 + **仅 OSS 必要 Action**；移动端长期凭证可用 **STS 临时凭证**（AssumeRole），由服务端签发。

实现细节、API 字段与错误码见 `references/oss-upload.md`；代码片段见 `references/go-snippets.md` 中 OSS 一节。

---

## 数据库（MySQL + GORM）

- **索引**：在模型上用 `gorm:"index"` / `uniqueIndex` 声明常用查询列；迁移（AutoMigrate 或独立 SQL）与线上索引变更要有评审。
- **避免 N+1**：列表带关联时用 **`Preload("Association")`** 或 **`Joins`** 一次拉齐；勿在 `for` 里对每条记录再 `Find`。
- **事务**：`db.Transaction(func(tx *gorm.DB) error { ... })` 或 `tx := db.Begin()` … `Commit`；同一事务内完成扣费 + 流水等多表写入。
- **连接池**：`sqlDB, _ := db.DB()` 后配置 `SetMaxOpenConns`、`SetMaxIdleConns`、`SetConnMaxLifetime`；避免默认值过小导致高并发排队。
- **只读查询**：热点列表可用 `Session(&gorm.Session{QueryFields: true})` 或明确 `Select` 字段，减少宽表回表（按需）。
- **敏感批量**：`Delete`/`Updates` 必须带 `Where`，防止误删全表；生产慎用全表 `AutoMigrate`，优先显式迁移脚本。

## Redis

- 会话、验证码、限流计数器、热点缓存；**缓存要有 TTL 与淘汰策略**。
- Key 命名空间：`app:v1:ratelimit:{ip}`、`app:v1:session:{id}`。
- 防缓存击穿：热点 key 互斥重建或短期空值缓存（视业务）。

---

## 微信小程序登录（服务端要点）

1. 客户端 `wx.login` 获取 **临时 `code`**（短有效）。
2. 服务端调用微信接口换取 **`openid`**（及 `session_key`，勿下发给前端）。
3. 用 `openid`/`unionid` 查找或创建用户，写入 MySQL。
4. 签发 **自有 JWT**（及 refresh 存 Redis），返回给客户端。

**注意**：`session_key` 仅服务端用于解密敏感数据等业务；不要放进 JWT。

---

## 基础代码示例（Gin + GORM）

### 1. 统一 JSON 信封（Gin）

```go
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorBody struct {
	Code    string           `json:"code"`
	Message string           `json:"message"`
	Details []map[string]any `json:"details,omitempty"`
}

type Envelope struct {
	Success   bool       `json:"success"`
	Data      any        `json:"data,omitempty"`
	Error     *ErrorBody `json:"error,omitempty"`
	RequestID string     `json:"request_id,omitempty"`
}

func JSON(c *gin.Context, status int, env Envelope) {
	if env.RequestID == "" {
		env.RequestID = c.GetString("request_id")
	}
	c.JSON(status, env)
}
```

### 2. Handler：绑定 + 业务错误映射

```go
type wechatLoginReq struct {
	Code  string `json:"code" binding:"required"`
	AppID string `json:"app_id"`
}

func (h *Handler) WeChatLogin(c *gin.Context) {
	var req wechatLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSON(c, http.StatusUnprocessableEntity, response.Envelope{
			Success: false,
			Error: &response.ErrorBody{
				Code:    "VALIDATION_FAILED",
				Message: err.Error(),
			},
		})
		return
	}
	// 调用 service，成功则 response.JSON(c, 200, Envelope{Success: true, Data: ...})
}
```

### 3. GORM 查询（安全 Where）

```go
var u User
if err := db.WithContext(c.Request.Context()).Where("openid = ?", openid).First(&u).Error; err != nil {
	// errors.Is(err, gorm.ErrRecordNotFound) -> 404
}
```

### 4. GORM 事务

```go
err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
	if err := tx.Model(&Wallet{}).Where("user_id = ? AND balance >= ?", uid, amount).
		UpdateColumn("balance", gorm.Expr("balance - ?", amount)).Error; err != nil {
		return err
	}
	return tx.Create(&WalletLedger{UserID: uid, Amount: -amount}).Error
})
```

### 5. 列表 + 预加载（避免 N+1）

```go
var list []Child
err := db.WithContext(ctx).
	Preload("Classroom").
	Where("tenant_id = ?", tid).
	Order("created_at DESC").
	Limit(20).Offset((page - 1) * 20).
	Find(&list).Error
```

### 6. Redis 限流

```go
// INCR key + EXPIRE；超限 c.AbortWithStatusJSON(429, Envelope{...})
// 或引入 github.com/go-redis/redis_rate 做令牌桶
```

---

## 实现清单（接到需求时按序做）

1. 定稿 **错误码表**（与前端/小程序对齐）。
2. 定稿 **版本前缀** 与分页字段名。
3. 定义 **`config.Config`** 与各层 **构造函数 + `wire.NewSet`**，在 `internal/app` 写 **`wire.go` + `wire.Build`**，生成 **`wire_gen.go`**。
4. **`main`**：**`pkg/log.Init`** → **加载 INI**（`config.Load`）→ `app.InjectApp(cfg)` → `engine.Run`；禁止在 `main` 里手动拼装长依赖链。
5. 搭 **Gin Engine**（由注入器得到）：`gin.Recovery()`、`RequestID`、**日志 context**、CORS；`r.Group("/api/v1")`。
6. **auth**：JWT + 微信 code2session + Redis refresh。
7. **Casbin**：`rbac_model.conf` + 策略源（CSV 或 DB）；**JWT 之后**挂 `Enforce` 中间件；后台与小程序路由可按前缀拆分不同策略。
8. **Repository** 接收 `*gorm.DB`（或接口），只暴露领域方法名；handler 不出现 GORM 链式调用。
9. 为核心列表接口检查 **索引与 N+1**。
10. 对登录、短信、**获取上传凭证/预签名** 接 **限流**。
11. **OSS**：配置 `endpoint`、`bucket`、凭证（或 STS）；实现 **预签名或 Post Policy**（或转存）；DB 存 **object key** 与 **CDN/访问 URL 策略**，不存 AK/SK。

---

## 与前端协作

- 提供 **OpenAPI 3** 或等价文档：`paths`、`schemas`、`securitySchemes`（bearerJWT）。
- 错误码文档含：**code、HTTP 状态、含义、前端处理建议**（如跳转登录）。

按需加载参考文件：

- `references/config-ini.md` — INI 节示例、与环境变量覆盖约定
- `references/rbac-casbin.md` — Casbin 模型/策略示例、与 Gin 的 sub/obj/act 约定
- `references/api-io-spec.md` — 字段习惯、登录/刷新、**OSS 预签名/上传**、错误码表模板
- `references/oss-upload.md` — OSS 策略、Key 规范、小程序与后台协作要点
- `references/go-snippets.md` — INI 加载、**pkg/log**、**Casbin + Gin**、Wire、JWT、Redis、GORM、OSS、code2session
