# Go 代码片段（Gin + GORM + Redis + OSS + INI + Casbin + slog）

`go.mod` 中典型依赖：

```text
github.com/gin-gonic/gin
github.com/google/wire
github.com/google/uuid
gopkg.in/ini.v1
github.com/casbin/casbin/v2
gorm.io/gorm
gorm.io/driver/mysql
github.com/golang-jwt/jwt/v5
github.com/redis/go-redis/v9
github.com/aliyun/aliyun-oss-go-sdk/oss
```

安装代码生成工具（一次性）：

```bash
go install github.com/google/wire/cmd/wire@latest
```

在 **`internal/app`** 目录执行 **`wire`**。可选：新增 `generate.go`（无 `wireinject` tag）仅承载指令：

```go
package app

//go:generate go run github.com/google/wire/cmd/wire
```

然后在仓库根目录执行 **`go generate ./internal/app`**。

生产环境将 Gin 设为 **`gin.ReleaseMode`**，由 INI `[app] gin_mode` 或环境变量控制。

---

## 应用配置（INI）

```go
package config

import (
	"os"

	"gopkg.in/ini.v1"
)

type Config struct {
	App struct {
		HTTPAddr string `ini:"http_addr"`
		GinMode  string `ini:"gin_mode"`
	} `ini:"app"`
	Casbin struct {
		ModelPath  string `ini:"model_path"`
		PolicyPath string `ini:"policy_path"`
		Enable     bool   `ini:"enable"`
	} `ini:"casbin"`
	Log struct {
		Level  string `ini:"level"`
		Format string `ini:"format"`
	} `ini:"log"`
	MySQL struct {
		DSN string `ini:"dsn"`
	} `ini:"mysql"`
}

func Load(path string) (*Config, error) {
	f, err := ini.Load(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := f.MapTo(&c); err != nil {
		return nil, err
	}
	if v := os.Getenv("MYSQL_DSN"); v != "" {
		c.MySQL.DSN = v
	}
	return &c, nil
}
```

---

## 统一日志（`pkg/log` + `slog`）

```go
package log

import (
	"context"
	"log/slog"
	"os"
)

type ctxKey struct{}

func Init(level slog.Level, json bool) {
	opts := &slog.HandlerOptions{Level: level}
	var h slog.Handler
	if json {
		h = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		h = slog.NewTextHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(h))
}

func With(ctx context.Context, args ...any) context.Context {
	parent := L(ctx)
	return context.WithValue(ctx, ctxKey{}, parent.With(args...))
}

func L(ctx context.Context) *slog.Logger {
	if v := ctx.Value(ctxKey{}); v != nil {
		return v.(*slog.Logger)
	}
	return slog.Default()
}
```

### Gin：注入 `request_id` / `user_id`

```go
package middleware

import (
	"github.com/gin-gonic/gin"
	"your/module/pkg/log"
)

func RequestContextLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := log.With(c.Request.Context(),
			"request_id", c.GetString("request_id"),
		)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// 鉴权成功后：ctx = log.With(c.Request.Context(), "user_id", uid, "role", role)
```

业务代码：`log.L(ctx).Info("created order", "order_id", id)`。

---

## Casbin + Gin

```go
package rbac

import (
	"net/http"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"

	"your/module/pkg/log"
	"your/module/pkg/response"
)

func NewEnforcer(modelPath, policyPath string) (*casbin.Enforcer, error) {
	return casbin.NewEnforcer(modelPath, policyPath)
}

// CasbinMiddleware sub 为 Casbin 的 subject：常与策略一致，例如用户 ID（配合 g, user, role）
func CasbinMiddleware(e *casbin.Enforcer, subjectFn func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if e == nil {
			c.Next()
			return
		}
		sub := subjectFn(c)
		if sub == "" {
			c.Next()
			return
		}
		obj := c.FullPath()
		act := c.Request.Method
		ok, err := e.Enforce(sub, obj, act)
		if err != nil {
			log.L(c.Request.Context()).Error("casbin enforce", "err", err)
			response.JSON(c, http.StatusInternalServerError, response.Envelope{Success: false,
				Error: &response.ErrorBody{Code: "INTERNAL", Message: "权限校验失败"}})
			c.Abort()
			return
		}
		if !ok {
			response.JSON(c, http.StatusForbidden, response.Envelope{Success: false,
				Error: &response.ErrorBody{Code: "PERMISSION_DENIED", Message: "无权限"}})
			c.Abort()
			return
		}
		c.Next()
	}
}
```

**注意**：`subjectFn` 在 **JWT 中间件之后** 执行；匿名路由不要挂本中间件或使用白名单。策略与模型见 `references/rbac-casbin.md`。

---

## Google Wire 依赖注入

### 原则

- **构造函数注入**：`NewX(dep *Dep)`，由 Wire 解析依赖顺序。
- **`wire_gen.go` 仅由 `wire` 生成**，不要手写修改；改依赖后重新 `wire`。
- **`wire.go` 使用 build tag `wireinject`**，避免与正常编译冲突。
- 需要 **同一接口多种实现** 时，用 `wire.Bind`；需要 **从 struct 取字段** 时，用 `wire.FieldsOf`。

### `internal/app/wire.go`（注入器）

```go
//go:build wireinject
// +build wireinject

package app

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"

	"your/module/internal/config"
	handler "your/module/internal/handler/http"
	"your/module/internal/middleware"
	"your/module/internal/repository"
	"your/module/internal/service"
)

// InjectEngine 由 wire 生成实现，写入 wire_gen.go
func InjectEngine(cfg *config.Config) (*gin.Engine, error) {
	wire.Build(
		ProvideGorm,
		ProvideRedis,
		middleware.ProvideJWTSecret,
		middleware.ProvideAuth,
		repository.Set,
		service.Set,
		handler.Set,
		NewRouter,
	)
	return nil, nil
}
```

### 基础设施 Provider（可与 `OpenMySQL` 合并）

```go
package app

import (
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"your/module/internal/config"
)

func ProvideGorm(cfg *config.Config) (*gorm.DB, error) {
	return OpenMySQL(cfg.MySQLDSN)
}

func ProvideRedis(cfg *config.Config) *redis.Client {
	return redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
}
```

### 分层 `ProviderSet` 示例

```go
// internal/repository/wire.go（或 set.go）
package repository

import "github.com/google/wire"

var Set = wire.NewSet(
	NewUserRepo,
)

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}
```

```go
// internal/service/wire.go
package service

import "github.com/google/wire"

var Set = wire.NewSet(
	NewUserService,
)

func NewUserService(r *repository.UserRepo, rdb *redis.Client) *UserService {
	return &UserService{repo: r, redis: rdb}
}
```

```go
// internal/handler/http/wire.go
package handler

import "github.com/google/wire"

var Set = wire.NewSet(
	NewHandler,
)
```

```go
// internal/middleware/wire.go — 将 []byte secret 注入鉴权中间件
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"your/module/internal/config"
)

func ProvideJWTSecret(cfg *config.Config) []byte {
	return []byte(cfg.JWTSecret)
}

func ProvideAuth(secret []byte) gin.HandlerFunc {
	return AuthJWT(secret)
}
```

### `cmd/api/main.go`

```go
package main

import (
	"log"

	"your/module/internal/app"
	"your/module/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	engine, err := app.InjectEngine(cfg)
	if err != nil {
		log.Fatal(err)
	}
	if err := engine.Run(cfg.HTTPAddr); err != nil {
		log.Fatal(err)
	}
}
```

生成后会出现 **`internal/app/wire_gen.go`**，内含 `func InjectEngine(cfg *config.Config) (*gin.Engine, error) { ... }` 的具体实现。

### `internal/app/router.go`（普通源文件，汇总路由）

与「Gin 路由与中间件骨架」一致，由 Wire 注入 `*handler.Handler` 与 `gin.HandlerFunc`（鉴权）：

```go
package app

import (
	"github.com/gin-gonic/gin"
	handler "your/module/internal/handler/http"
	"your/module/internal/middleware"
)

func NewRouter(h *handler.Handler, auth gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())

	v1 := r.Group("/api/v1")
	{
		v1.POST("/auth/wechat/login", h.WeChatLogin)
		authed := v1.Group("")
		authed.Use(auth)
		authed.GET("/me", h.Me)
	}
	return r
}
```

### 接口与测试

```go
wire.Bind(new(UserReader), new(*UserRepo))
```

测试可手写 `NewHandler(NewUserService(fakeRepo))`，不必每次跑 Wire；集成测试可对 `InjectEngine` 使用测试配置与测试库。

---

## GORM 连接 MySQL 与连接池

```go
import (
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func OpenMySQL(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)
	return db, nil
}
```

业务里统一 **`db.WithContext(c.Request.Context())`**，便于超时与 tracing 传递。

---

## Gin 路由与中间件骨架

与 Wire 配合时，将 **`NewRouter` 放在 `internal/app/router.go`**（见上文 Wire 一节），由注入器组装 `*handler.Handler` 与鉴权 `gin.HandlerFunc`。

```go
func NewRouter(h *handler.Handler, auth gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	// r.Use(gin.Logger()) 或接入 zap / slog

	v1 := r.Group("/api/v1")
	{
		v1.POST("/auth/wechat/login", h.WeChatLogin)
		v1.POST("/auth/refresh", h.Refresh)

		authed := v1.Group("")
		authed.Use(auth)
		{
			authed.GET("/me", h.Me)
		}
	}
	return r
}
```

### RequestID 中间件（写入 `c` 与 `response` 信封）

```go
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			rid = uuid.NewString() // github.com/google/uuid
		}
		c.Set("request_id", rid)
		c.Writer.Header().Set("X-Request-ID", rid)
		c.Next()
	}
}
```

### JWT 鉴权中间件（Bearer）

```go
const ctxClaimsKey = "jwt_claims"

func AuthJWT(secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		const p = "Bearer "
		if len(h) < len(p) || h[:len(p)] != p {
			// 使用与项目一致的 JSON 信封，例如 response.JSON(c, 401, Envelope{...})
			c.Abort()
			return
		}
		claims, err := auth.ParseAccess(secret, h[len(p):])
		if err != nil {
			c.Abort()
			return
		}
		c.Set(ctxClaimsKey, claims)
		c.Next()
	}
}
```

Handler 内取 claims：`claims := c.MustGet(ctxClaimsKey).(*auth.AccessClaims)`（或安全类型断言）。

---

## GORM 事务（推荐封装在 service / repository）

以下 `Wallet`、`WalletLedger` 为示意模型，按实际表结构定义。

```go
func Transfer(ctx context.Context, db *gorm.DB, fromID, toID int64, amount int64) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&Wallet{}).Where("user_id = ? AND balance >= ?", fromID, amount).
			UpdateColumn("balance", gorm.Expr("balance - ?", amount)).Error; err != nil {
			return err
		}
		if err := tx.Model(&Wallet{}).Where("user_id = ?", toID).
			UpdateColumn("balance", gorm.Expr("balance + ?", amount)).Error; err != nil {
			return err
		}
		return tx.Create(&WalletLedger{FromUserID: fromID, ToUserID: toID, Amount: amount}).Error
	})
}
```

---

## 列表查询：Preload 避免 N+1

```go
var rows []Order
err := db.WithContext(ctx).
	Preload("Items").
	Where("user_id = ?", uid).
	Order("created_at DESC").
	Limit(20).Find(&rows).Error
```

多条件关联用 **`Joins`** 时注意只选必要字段，避免宽结果集。

---

## JWT 签发与解析（access）

```go
package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AccessClaims struct {
	UserID   string   `json:"uid"`
	Roles    []string `json:"roles"`
	TenantID string   `json:"tid,omitempty"`
	jwt.RegisteredClaims
}

func SignAccess(secret []byte, userID string, roles []string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := AccessClaims{
		UserID: userID,
		Roles:  roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "kindergarten-api",
			Subject:   userID,
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(secret)
}

func ParseAccess(secret []byte, tokenStr string) (*AccessClaims, error) {
	t, err := jwt.ParseWithClaims(tokenStr, &AccessClaims{}, func(t *jwt.Token) (any, error) {
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	c, ok := t.Claims.(*AccessClaims)
	if !ok || !t.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return c, nil
}
```

---

## Redis 存储 refresh（可撤销）

```go
func SaveRefresh(ctx context.Context, rdb *redis.Client, jti, userID string, ttl time.Duration) error {
	return rdb.Set(ctx, "refresh:"+jti, userID, ttl).Err()
}

func ConsumeRefresh(ctx context.Context, rdb *redis.Client, jti string) (string, error) {
	key := "refresh:" + jti
	userID, err := rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", err
	}
	if err != nil {
		return "", err
	}
	_ = rdb.Del(ctx, key).Err()
	return userID, nil
}
```

---

## 阿里云 OSS（图片/附件）

`endpoint` 形如 `https://oss-cn-hangzhou.aliyuncs.com`（与 Bucket 地域一致）；凭证优先 **RAM 子账号**，权限收敛到目标 Bucket。

### 初始化 Client 与 Bucket

```go
package ossx

import (
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type Config struct {
	Endpoint        string
	AccessKeyID     string
	AccessKeySecret string
	BucketName      string
}

type Storage struct {
	bucket *oss.Bucket
}

func NewStorage(cfg Config) (*Storage, error) {
	client, err := oss.New(cfg.Endpoint, cfg.AccessKeyID, cfg.AccessKeySecret)
	if err != nil {
		return nil, err
	}
	bucket, err := client.Bucket(cfg.BucketName)
	if err != nil {
		return nil, err
	}
	return &Storage{bucket: bucket}, nil
}
```

### 预签名 PUT（直传，适合支持 PUT 的 Web 端）

```go
import "time"

func (s *Storage) PresignPut(objectKey, contentType string, ttl time.Duration) (string, error) {
	// 实际上传请求的 Content-Type（及 Meta）须与签名时一致，否则易 SignatureDoesNotMatch
	return s.bucket.SignURL(objectKey, oss.HTTPPut, int64(ttl.Seconds()),
		oss.ContentType(contentType),
	)
}
```

前端使用返回的 URL，**PUT** 原始二进制 body，`Content-Type` 须与签名一致。

### 预签名 GET（私有 Bucket 临时访问）

```go
func (s *Storage) PresignGet(objectKey string, ttl time.Duration) (string, error) {
	return s.bucket.SignURL(objectKey, oss.HTTPGet, int64(ttl.Seconds()))
}
```

### 服务端转存（Gin `multipart` → OSS）

```go
import "github.com/gin-gonic/gin"

func (s *Storage) PutFromMultipart(c *gin.Context, formKey, objectKey string) error {
	fh, err := c.FormFile(formKey)
	if err != nil {
		return err
	}
	src, err := fh.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	// 生产：校验 fh.Size、MIME、白名单扩展名
	return s.bucket.PutObject(objectKey, src,
		oss.ContentType(fh.Header.Get("Content-Type")),
	)
}
```

### 微信小程序：表单 POST（PostObject）

`wx.uploadFile` 走 **multipart POST**，需服务端下发 **Policy、Signature、host、key、OSSAccessKeyId** 等（由 OSS Post 签名规则生成）。具体字段与 SDK 辅助函数以 [阿里云 OSS PostObject 文档](https://help.aliyun.com/zh/oss/developer-reference/postobject) 为准；Policy 中务必限制 **`content-length-range`**、`key` 前缀、`Content-Type` 前缀（如 `image/`）。

### Wire

将 `ossx.NewStorage` 纳入 `wire.Build`，Handler/Service 注入 `*ossx.Storage`；**勿**把 AK/SK 放进 DB。

---

## 微信 code2session

调用 `GET https://api.weixin.qq.com/sns/jscode2session`，参数：`appid`、`secret`、`js_code`、`grant_type=authorization_code`。

- 使用 **`http.Client` + `context` 超时**（如 5s）。
- 失败映射为 **`WECHAT_CODE_INVALID`** 等业务码，不把微信原始 body 透传给前端。
