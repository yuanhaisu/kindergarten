# 养成星球 · API 接口文档 — 公共接口

> 三端（老师端、家长端、管理后台）共用的公共接口。
>
> 基础路径：`/api/v1`
>
> 鉴权：除 `auth` 开头的接口外，均需携带 `Authorization: Bearer <token>`

---

## 通用约定

### 请求格式
- `Content-Type: application/json`
- 分页参数：`?page=1&page_size=20`
- 时间格式：`2006-01-02T15:04:05Z07:00`（RFC 3339）
- 日期格式：`2006-01-02`

### 统一响应结构

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

错误响应：
```json
{
  "code": 40001,
  "message": "验证码错误或已过期",
  "data": null
}
```

### 分页响应结构

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [],
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}
```

> 以下接口示例中，仅展示 `data` 部分内容，省略外层 `code` / `message` 包装。

---

## 1. 认证

> ⛔ **v1 关闭的端点**：`POST /auth/sms/send` 与 `POST /auth/login/sms` 在 v1 固定返回
> `403`（`code=40300`），不发送验证码、不签发任何 token。真实短信 + 频控 + 角色校验
> 落地前不要接入。

> 登录与刷新响应包含 `consent_required` 与 `current_policy_version`（增量字段，旧客户端忽略）。
> `consent_required=true` 时，客户端 MUST 先调用 `POST /common/consent` 完成同意，再访问业务接口；
> 服务端在 `[consent] enforce=true` 时会对未同意的业务请求返回 `403` + `code=40301`。

### POST `/auth/sms/send` — v1 不可用

> ⛔ 短信通道在 v1 关闭，该端点固定返回 `403`，不会发送验证码；请求格式仅作将来恢复时的参考。

```json
// Request（v1 仍会被拒绝）
{
  "phone": "13800001234",
  "scene": "login"            // login=登录 | bind=绑定手机号 | authorization=添加授权接送人
}

// Response（v1 固定为 403）
{
  "code": 40300,
  "message": "短信功能在 v1 未开放",
  "data": null
}
```

---

### POST `/auth/login/sms` — v1 不可用

> ⛔ 手机号 + 验证码登录在 v1 关闭，该端点固定返回 `403`，**不签发 `token` / `refresh_token`**；
> 任何 `role`（teacher / parent / admin）结果相同。恢复前必须先补齐真实短信、频控与角色校验。

```json
// Request（v1 仍会被拒绝）
{
  "phone": "13800001234",
  "code": "123456",
  "role": "teacher"           // teacher | parent | admin
}

// Response（v1 固定为 403）
{
  "code": 40300,
  "message": "短信功能在 v1 未开放",
  "data": null
}
```

---

### POST `/auth/login/wechat` — v1 关闭

> ⛔ 旧微信登录在 v1 关闭（#164）：`open_id` 不再直接建账号。服务端固定返回
> `403`（`code=40300`），不调用微信 `code2session`。新登录（`app_id` + `code`
> → 账号 + 客户端身份）由 #175 落定后再更新本节；在此之前旧客户端应引导用户升级。

```json
// Request（v1 仍会被拒绝；旧格式仅作参考）
{
  "code": "0b3Khv000aXCfR1xxx000",
  "role": "teacher"                    // teacher | parent
}

// Response（v1 固定为 403）
{
  "code": 40300,
  "message": "微信登录已升级，请更新小程序后重试",
  "data": null
}
```

---

### POST `/common/user/bind-invite`、`POST /common/user/bind-parent-phone` — 已移除

> ⛔ 两个旧绑定端点已在 v1 移除，路由不存在（`404`），无需登录态。教师邀请码绑定
> 与家长手机号换绑分别由 #168 / #167 重写后重新登记。

---

### POST `/auth/login/password`

> 账号密码登录（管理后台专用）

```json
// Request
{
  "phone": "13800001234",
  "password": "Abc123456"
}

// Response（登录响应）
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": "2026-03-20T12:00:00Z",
  "consent_required": true,
  "current_policy_version": "v1",
  "user": {
    "id": 100,
    "phone": "138****1234",
    "nickname": "张园长",
    "avatar": "https://cdn.example.com/avatar/100.jpg",
    "gender": 2,
    "roles": [
      {
        "role": "admin",
        "kindergarten_id": 1,
        "kindergarten_name": "阳光幼儿园",
        "admin_role": "系统管理员"
      }
    ]
  }
}
```

---

### POST `/auth/refresh`

> 刷新 token

```json
// Request
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}

// Response
{
  "token": "eyJhbGciOiJIUzI1NiIs...(new)",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...(new)",
  "expires_at": "2026-03-21T12:00:00Z",
  "consent_required": true,
  "current_policy_version": "v1"
}
```

---

### POST `/auth/logout`

> 退出登录：吊销当前 access token；`refresh_token` 可选，携带时一并吊销该 refresh 会话。

```json
// Request
{
  "refresh_token": "<登录返回的 refresh_token>"  // 可选；不带则只吊销 access token
}

// Response
{}
```

---

## 2. 文件上传

### POST `/common/upload`

> 通用文件上传（单个文件）

```
Content-Type: multipart/form-data
字段: file       — 文件（必填）
字段: biz_type   — 业务类型（必填）：notice / growth / avatar / leave / task
```

```json
// Response
{
  "id": 1001,
  "file_url": "https://cdn.example.com/uploads/2026/03/19/abc123.jpg",
  "thumb_url": "https://cdn.example.com/uploads/2026/03/19/abc123_thumb.jpg",
  "file_name": "photo.jpg",
  "file_size": 1024000,
  "file_type": "image",
  "duration": 0
}
```

> **媒体 URL 语义**：`file_url` / `thumb_url` 为**短期签名 URL**（默认 1 小时有效），
> 不是长期公开地址。客户端应以 `id` / `file_id` 作为媒体身份；URL 过期后重新拉取。
> 数据库内部保存 object key，响应层统一签名。

---

### POST `/common/upload/batch`

> 批量上传文件（最多 20 个）

```
Content-Type: multipart/form-data
字段: files[]    — 文件数组（必填）
字段: biz_type   — 业务类型（必填）：growth / notice
```

```json
// Response
{
  "files": [
    {
      "id": 1001,
      "file_url": "https://cdn.example.com/uploads/2026/03/19/abc123.jpg",
      "thumb_url": "https://cdn.example.com/uploads/2026/03/19/abc123_thumb.jpg",
      "file_name": "IMG_001.jpg",
      "file_size": 1024000,
      "file_type": "image",
      "duration": 0
    },
    {
      "id": 1002,
      "file_url": "https://cdn.example.com/uploads/2026/03/19/def456.mp4",
      "thumb_url": "https://cdn.example.com/uploads/2026/03/19/def456_thumb.jpg",
      "file_name": "VID_001.mp4",
      "file_size": 5120000,
      "file_type": "video",
      "duration": 15
    }
  ],
  "success_count": 2,
  "fail_count": 0
}
```

---

## 3. 用户信息

### GET `/common/user/profile`

> 获取当前登录用户信息

```json
// Response
{
  "id": 1,
  "phone": "138****1234",
  "nickname": "李老师",
  "avatar": "https://cdn.example.com/avatar/1.jpg",
  "gender": 1,
  "roles": [
    {
      "role": "teacher",
      "kindergarten_id": 1,
      "kindergarten_name": "阳光幼儿园",
      "class_id": 1,
      "class_name": "小一班",
      "title": "head",
      "title_label": "主班老师"
    },
    {
      "role": "admin",
      "kindergarten_id": 1,
      "kindergarten_name": "阳光幼儿园",
      "admin_role": "园所管理员",
      "permissions": ["dashboard:view", "student:view", "student:manage"]
    }
  ],
  "created_at": "2025-09-01T08:00:00Z"
}
```

**字段说明**：

- `roles[]` 为当前用户全部角色条目；字段随 `role` 取值不同。
- `role: "admin"` 的条目承载管理端角色信息：
  - `admin_role`：管理端角色标记/名称（超管账号为 `super_admin` 标记）。
  - `permissions`：该账号在对应园所的权限码集合，供管理端渲染菜单与拦截路由使用；**超管为通配时返回 `["*"]`**。权限码目录见 `api-admin/17-roles.md` 的 `GET /admin/permissions`。
- 其他角色的条目不含 `admin_role` / `permissions`。

---

### PUT `/common/user/profile`

> 更新个人信息

```json
// Request
{
  "nickname": "李老师",
  "gender": 1
}

// Response
{
  "id": 1,
  "phone": "138****1234",
  "nickname": "李老师",
  "avatar": "https://cdn.example.com/avatar/1.jpg",
  "gender": 1
}
```

---

### PUT `/common/user/avatar`

> 更新头像

```json
// Request
{
  "avatar": "https://cdn.example.com/uploads/2026/03/19/new_avatar.jpg"
}

// Response
{
  "avatar": "https://cdn.example.com/uploads/2026/03/19/new_avatar.jpg"
}
```

---

## 4. 消息

### GET `/common/messages`

> 站内消息列表

```json
// Query: ?type=notice&is_read=false&page=1&page_size=20
// type 可选值: notice / leave / pickup / growth / album / task / system，不传则查全部

// Response
{
  "list": [
    {
      "id": 1,
      "type": "leave",
      "title": "请假申请已通过",
      "content": "王小明的请假申请（3月19日-3月21日）已被李老师批准",
      "reference_id": 101,
      "action_path": "",
      "is_read": false,
      "created_at": "2026-03-19T07:35:00Z",
      "read_at": null
    },
    {
      "id": 2,
      "type": "pickup",
      "title": "孩子已被接走",
      "content": "王小明已于17:30被爸爸（王明）接走，门卫李师傅核验",
      "reference_id": 201,
      "action_path": "",
      "is_read": true,
      "created_at": "2026-03-18T17:30:25Z",
      "read_at": "2026-03-18T17:31:00Z"
    }
  ],
  "total": 45,
  "page": 1,
  "page_size": 20
}
```

**字段说明（消息列表）**：

- `action_path`：站内跳转路径，**只允许内部 `/pages/` 开头的路径**；空串表示该消息不可跳转。
  任务发布与主回应消息形如 `/pages/tasks/detail?id=<task_id>&student_id=<student_id>`：
  `id` 定位任务，`student_id` 指定进入时应选中的孩子（多孩家庭必须带上，否则会落到默认孩子）。
  客户端必须先按下当前孩子与绑定关系校验再跳转；非 `/pages/` 开头的路径一律忽略，不得当外链打开。
- `reference_id`：关联业务对象 id。任务发布消息为 `task_id`，主回应消息为 `response_id`，
  班级相册发布消息为 `album_id`。
- `type=album`（班级相册发布）：`action_path` 形如
  `/pages/album/detail?id=<album_id>&student_id=<该家长参与的孩子>`；同一相册对同一家长只发一条，
  重复发布不重复发；监护人本人关闭班内共享时不投递。

---

### GET `/common/messages/unread-count`

> 未读消息数量（按类型分组，用于各端底部 tab 红点和首页通知铃铛）

```json
// Response
{
  "total": 8,
  "by_type": {
    "notice": 2,
    "leave": 1,
    "pickup": 0,
    "growth": 3,
    "album": 1,
    "task": 1,
    "system": 0
  }
}
```

---

### PUT `/common/messages/:id/read`

> 标记单条消息已读

```json
// Request
{}

// Response
{
  "id": 1,
  "is_read": true,
  "read_at": "2026-03-19T10:00:00Z"
}
```

---

### PUT `/common/messages/read-all`

> 全部标记已读

```json
// Request
{
  "type": "notice"    // 可选，指定只标记某类型；不传则全部标记
}

// Response
{
  "marked_count": 5
}
```

---

## 5. 隐私政策同意

> 三端通用。记录用户对当前隐私政策版本的同意。客户端 MUST 在请求头带
> `terminal_from: parent | teacher | admin`；服务端校验该端出现在令牌角色中后写入 `source`，
> 不接受与角色不符的自称。旧客户端未带该头时，服务端回退为按令牌角色推导（兼容期行为）。
> 同一用户同一版本重复提交幂等，返回首次同意时间。

### GET `/common/consent`

```json
// Response
{
  "policy_version": "v1",
  "consented": false,
  "consented_at": null
}
```

### POST `/common/consent`

```json
// Request
{
  "policy_version": "v1"
}

// Response
{
  "policy_version": "v1",
  "consented": true,
  "consented_at": "2026-11-09T10:00:00+08:00"
}
```

| code | HTTP | 含义 |
| --- | --- | --- |
| 40030 | 400 | 政策版本已更新，客户端需刷新后重试 |
| 40031 | 400 | 无法识别账号来源端 |
| 40301 | 403 | 未同意当前政策，业务接口被拦截；客户端应跳转同意页 |

> 当前政策版本为 `v1`，与官网 `website/content/site.ts` 的 `policyVersion` 保持一致；
> 备案通过、文本定稿后两处一起改为生效日期。

---

## 8.1 自助改密

### PUT `/common/user/password`

> 修改当前账号密码。平台方开通园所时创建的园长账号，登录响应中
> `user.must_change_password` 为 `true`，改密成功后清零。

```json
// Request
{
  "old_password": "初始密码",
  "new_password": "新密码（含大写、小写与数字，至少 8 位）"
}

// Response
{
  "must_change_password": false
}
```

> 原密码错误返回 422；新密码不满足强度要求返回 422。

---

## 9. 功能授权

### GET `/common/entitlements`

> 读取当前园所的功能模块授权。两端小程序在启动时调用，用于隐藏未购买的模块入口。

```json
// Response
{
  "modules": ["attendance", "notice", "pickup", "calendar", "growth"],
  "expires_at": "2027-05-05"
}
```

> 只返回当前可用的模块；已到期并超过宽限期的 premium 模块不会出现在 `modules` 中。
> `expires_at` 取所有在期授权中最早的到期日，仅用于展示提醒；为 `null` 表示无在期授权。
> 客户端隐藏入口**不能替代**服务端校验：直接调用未授权模块的接口仍会被拒绝。
