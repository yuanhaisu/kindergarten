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

### POST `/auth/sms/send`

> 发送短信验证码

```json
// Request
{
  "phone": "13800001234",
  "scene": "login"            // login=登录 | bind=绑定手机号 | authorization=添加授权接送人
}

// Response
{
  "expire_seconds": 300       // 验证码有效期（秒）
}
```

---

### POST `/auth/login/sms`

> 手机号 + 验证码登录（老师端 / 家长端 / 管理后台通用）

```json
// Request
{
  "phone": "13800001234",
  "code": "123456",
  "role": "teacher"           // teacher | parent | admin（用于确定登录端）
}

// Response
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": "2026-03-20T12:00:00Z",
  "user": {
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
        "title": "head"
      }
    ]
  }
}
```

---

### POST `/auth/login/wechat`

> 微信小程序登录（老师端 / 家长端）

```json
// Request
{
  "code": "0b3Khv000aXCfR1xxx000",   // wx.login() 获取的 code
  "role": "teacher",                   // teacher | parent
  "encrypted_data": "...",             // 可选，获取手机号时传入
  "iv": "..."                          // 可选
}

// Response（同 sms 登录格式）
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": "2026-03-20T12:00:00Z",
  "user": {
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
        "title": "head"
      }
    ]
  }
}
```

---

### POST `/auth/login/password`

> 账号密码登录（管理后台专用）

```json
// Request
{
  "phone": "13800001234",
  "password": "Abc123456"
}

// Response（同 sms 登录格式）
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": "2026-03-20T12:00:00Z",
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
  "expires_at": "2026-03-21T12:00:00Z"
}
```

---

### POST `/auth/logout`

> 退出登录（使当前 token 失效）

```json
// Request
{}

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
// type 可选值: notice / leave / pickup / growth / task / system，不传则查全部

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
- `reference_id`：关联业务对象 id。任务发布消息为 `task_id`，主回应消息为 `response_id`。

---

### GET `/common/messages/unread-count`

> 未读消息数量（按类型分组，用于各端底部 tab 红点和首页通知铃铛）

```json
// Response
{
  "total": 7,
  "by_type": {
    "notice": 2,
    "leave": 1,
    "pickup": 0,
    "growth": 3,
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
