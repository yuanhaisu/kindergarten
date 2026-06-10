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

### Wire 代码生成

```bash
# 安装 wire 工具（一次性）
go install github.com/google/wire/cmd/wire@latest

# 生成注入代码
cd internal/app && wire
# 或
go generate ./internal/app
```

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
