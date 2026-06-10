# API 输入输出补充约定

本文档与 `SKILL.md` 主技能配合使用：定义字段级习惯与示例，便于与小程序、管理后台对齐。

## Gin 请求绑定（与 JSON 字段对齐）

- **Body**：`c.ShouldBindJSON(&req)`；结构体字段用 `` `json:"snake_case"` `` 与 API 文档一致。
- **校验**：`binding:"required"`、`email`、`min=1,max=100` 等；绑定失败返回 **422** + `VALIDATION_FAILED`，`message` 可映射为可读文案（勿把内部栈暴露给客户端）。
- **Query / URI**：`ShouldBindQuery`、`ShouldBindUri` 用于分页与 `:id` 路径参数。
- **重复绑定**：`ShouldBindJSON` 仅读一次 body；需要原始 body 做验签时，用 `ShouldBindBodyWith` 或自定义 `io.TeeReader`。

## 命名风格

- JSON 字段：**snake_case**（与 Go `json` tag 一致，小程序端友好）或 **camelCase**（若团队已统一 TS 前端，则全栈统一一种）。
- 布尔：`is_active`、`has_paid`。
- 枚举：字符串枚举，如 `role: "teacher" | "parent" | "admin"`。

## 认证相关接口示例

### POST /api/v1/auth/wechat/login

**Request**

```json
{
  "code": "wx_login_temp_code",
  "app_id": "optional_if_multi_mini_program"
}
```

**Response 200**

```json
{
  "success": true,
  "data": {
    "access_token": "jwt...",
    "expires_in": 900,
    "refresh_token": "opaque_or_jwt",
    "token_type": "Bearer",
    "user": {
      "id": "usr_xxx",
      "nickname": "家长A",
      "roles": ["parent"]
    }
  },
  "meta": { "request_id": "..." }
}
```

### POST /api/v1/auth/refresh

**Request**

```json
{ "refresh_token": "..." }
```

**Response**：同 login，返回新 access（及可选轮转 refresh）。

### POST /api/v1/auth/logout

**Headers**：`Authorization: Bearer ...`

**Response 204** 或 200 + `{ "success": true, "data": null }`（全项目统一）。

## 管理后台登录（密码 + 可选 2FA）

### POST /api/v1/admin/auth/login

**Request**

```json
{
  "username": "admin",
  "password": "***",
  "captcha_id": "optional",
  "captcha_code": "optional"
}
```

**Response**：返回 JWT + 用户权限列表；refresh 建议 Redis 可撤销。

## 阿里云 OSS 上传

### POST /api/v1/uploads/presign — 预签名 PUT

**Request**

```json
{
  "purpose": "avatar",
  "content_type": "image/jpeg",
  "size_bytes": 204800
}
```

**Response 200**

```json
{
  "success": true,
  "data": {
    "method": "PUT",
    "url": "https://bucket.oss-cn-xxx.aliyuncs.com/prod/tenant/2025/03/uuid.jpg?...",
    "object_key": "prod/tenant/2025/03/uuid.jpg",
    "headers": {
      "Content-Type": "image/jpeg"
    },
    "expires_at": "2025-03-20T08:05:00Z"
  },
  "meta": { "request_id": "..." }
}
```

客户端对 `url` 发 **PUT**，body 为文件字节，并带上 `headers` 中与签名一致的项。

### POST /api/v1/uploads/post-policy — 小程序表单上传凭证

**Response 200**（字段名以阿里云 PostObject 为准，示例）

```json
{
  "success": true,
  "data": {
    "host": "https://bucket.oss-cn-xxx.aliyuncs.com",
    "policy": "base64...",
    "access_key_id": "STS 或 RAM 的 AccessKeyId",
    "signature": "...",
    "key": "prod/tenant/2025/03/uuid.jpg",
    "security_token": "使用 STS 时必填，对应 x-oss-security-token"
  },
  "meta": { "request_id": "..." }
}
```

小程序 `wx.uploadFile` 的 `formData` 填入上述字段；**不要**在响应中返回 `AccessKeySecret`。

### POST /api/v1/uploads/complete（可选）

直传成功后，客户端上报 `object_key` 与业务关联（如 `child_id`），服务端写入 DB 并做病毒扫描/审核任务入队等。

## 错误码示例（需维护成表）

| code | HTTP | 说明 |
|------|------|------|
| `VALIDATION_FAILED` | 422 | 参数校验失败，`details` 带字段 |
| `UNAUTHORIZED` | 401 | 未登录或 token 无效 |
| `FORBIDDEN` | 403 | 无权限（与 Casbin `PERMISSION_DENIED` 二选一统一） |
| `PERMISSION_DENIED` | 403 | Casbin 鉴权失败（已登录但策略不允许） |
| `INTERNAL` | 500 | 服务端内部错误（如 Casbin `Enforce` 返回 error，勿暴露细节） |
| `NOT_FOUND` | 404 | 资源不存在 |
| `WECHAT_CODE_INVALID` | 400 | code 无效或已使用 |
| `RATE_LIMITED` | 429 | 请求过于频繁 |
| `UPLOAD_TOO_LARGE` | 422 | 超过允许大小 |
| `UPLOAD_INVALID_TYPE` | 422 | 类型不在白名单 |
| `OSS_SIGN_FAILED` | 500 | OSS 签名失败（对外脱敏） |

## Idempotency（可选进阶）

对 **POST 支付、创建订单** 等：支持 header `Idempotency-Key`，服务端 Redis 记录首次结果，重复请求返回相同响应，避免重复扣款。

## 日期与分页

- 过滤：`created_after`、`created_before`（RFC3339）。
- 游标分页：`next_cursor` 为 opaque string（如 base64 编码的 `(created_at,id)`），客户端原样回传。
