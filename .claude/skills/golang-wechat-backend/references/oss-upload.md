# 阿里云 OSS 上传（补充）

与 `SKILL.md`、`api-io-spec.md`、`go-snippets.md` 配合使用。

## 配置项（环境变量示例）

- `OSS_ENDPOINT`：如 `https://oss-cn-hangzhou.aliyuncs.com`
- `OSS_BUCKET`
- `OSS_ACCESS_KEY_ID` / `OSS_ACCESS_KEY_SECRET`（或改用 **STS**，不在进程中长期放主账号 AK）

## Object Key 建议

- 格式：`{env}/{tenant_id}/{yyyy}/{mm}/{uuid}.ext`，`ext` 与校验后的类型一致。
- 由服务端生成 **最终 key** 写入预签名或 Post Policy，避免客户端随意写路径（防覆盖、防遍历）。

## 预签名 PUT（管理后台 / 支持 PUT 的客户端）

1. 客户端请求：`content_type`、`byte_size`（可选，用于业务校验）、业务场景（头像/成长册等）。
2. 服务端校验角色与配额 → 生成 `object_key` → `SignURL(..., HTTPPut, ttl, oss.ContentType(...))`。
3. 客户端对该 URL 发 **PUT**，body 为文件二进制，**Header `Content-Type` 必须与签名一致**。
4. 成功后客户端再调业务 API，把 **`object_key` 或 CDN URL 策略** 写入 MySQL（如 `media.url` 存 key，展示时再签 GET）。

## PostObject（微信小程序）

1. 服务端构造 **Policy**（过期时间、`conditions`：`bucket`、`key` 前缀、`content-length-range`、`Content-Type` 前缀等）。
2. 返回 `host`、`policy`（Base64）、`signature`、`OSSAccessKeyId`、`key`（或 `dir` + 前端拼文件名，须同样受 Policy 约束）。
3. 小程序 `wx.uploadFile`：`filePath` + `formData` 带上上述字段；**仍要对单用户限流**（见主 skill）。

Policy 字段细则以阿里云 **PostObject** 文档为准。

## 私有读与 CDN

- **私有 Bucket**：列表/详情展示用 **短 TTL 签名 GET URL**，或由 **CDN + URL 鉴权** 统一对外。
- **公共读**：仅适合可公开资源；需配 **防盗链**（Referer）等。

## CORS（Web 直传）

在 OSS 控制台为 Bucket 配置：

- **来源**：管理后台实际域名（勿 `*` 生产环境）。
- **方法**：含 `PUT` 或 `POST`（与方案一致）。
- **暴露头**：按需 `ETag`、`x-oss-request-id` 等。

## 错误码（示例，与总表合并维护）

| code | HTTP | 说明 |
|------|------|------|
| `UPLOAD_NOT_ALLOWED` | 403 | 无上传权限或场景不允许 |
| `UPLOAD_TOO_LARGE` | 413 / 422 | 超过大小上限 |
| `UPLOAD_INVALID_TYPE` | 422 | MIME/扩展名不在白名单 |
| `OSS_SIGN_FAILED` | 500 | 签名/调用 OSS 失败（日志记详情，响应脱敏） |

## 安全备忘

- 预签名接口 **强限流**；**短 TTL**（如 60～300s）。
- 日志禁止打印 AccessKeySecret、完整预签名 URL（可打 object_key + request_id）。
- 定期轮转 RAM 密钥；生产使用 **最小权限策略**（仅 `oss:PutObject` / `oss:GetObject` 等到指定 `Resource`）。
