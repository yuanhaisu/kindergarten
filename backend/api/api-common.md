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

### POST `/auth/login/wechat` — 小程序登录（#175）

> 登录以 `app_id` 为入口：服务端先查客户端注册表，再用该客户端角色对应的
> 微信凭据调用 `code2session`。授权角色只来自注册表，请求体里的 `role`
> 仅用于一致性校验，不能改变授权范围。

```json
// Request
{
  "app_id": "wxdf989efff8796c53",      // 必填：小程序 AppID
  "code": "0b3Khv000aXCfR1xxx000",     // 必填：wx.login 的 code
  "role": "parent",                    // 可选：与客户端角色不一致时 422
  "invite_code": "T8K2..."             // 可选：仅老师端客户端可携带（绑定属于 #168）
}

// Response（登录响应）
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "…",
  "expires_at": "2026-11-21T12:00:00Z",
  "consent_required": true,
  "current_policy_version": "v1",
  "user": {
    "id": 100,
    "phone": "",                       // 待绑定账号为空
    "nickname": "",
    "avatar": "",
    "gender": 0,
    "must_change_password": false,
    "roles": []                        // 只含当前客户端角色族；待绑定账号为空
  }
}
```

- **身份**：按 `(app_id, open_id)` 查 `user_identities`；没有身份时创建「待绑定账号 +
  身份」（手机号为空、无角色、不加入任何园所）。同一身份再次登录复用同一账号。
- **角色裁剪**：`user.roles` 与 token `roles` 都是「当前客户端角色族 ∩ 该账号实际角色」。
  同一账号同时是家长与老师时，家长端只出 `parent`、老师端只出 `teacher`，
  `kindergarten_id` 也由裁剪后的角色推导。
- **待绑定限制**：待绑定 token（`app_id` 存在且角色为空）只能访问
  `GET|PUT /common/user/profile`、`PUT /common/user/avatar`、`GET|POST /common/consent`、
  `POST /auth/logout`、`GET /common/user/kindergartens` 与
  `POST /common/user/bind-parent-phone`（#167）；其余业务接口返回
  `403` + `code=40302`（客户端应跳转绑定页，而不是提示无权限）。
- **错误**：缺 `app_id` / 角色与客户端不一致 / 家长端携带 `invite_code` → `422`（`42200`）；
  客户端未注册或已停用 → `403`（`40300`）；`code` 无效 → `400`（`40002`）。
- `must_change_password` 只对管理端密码登录生效，小程序登录恒为 `false`。

---

### POST `/common/user/bind-invite` — 已移除

> ⛔ 教师邀请码绑定旧端点已在 v1 移除，路由不存在（`404`），无需登录态。
> 由 #168 重写后重新登记。

---

### POST `/common/user/bind-parent-phone` — 家长绑定手机号（#167）

> 家长端小程序专用。用微信手机号授权（`wx.getPhoneNumber` 的 `code`）换出手机号，
> 把当前「待绑定账号」的客户端身份并入**园所预创建**的家长账号；**绝不新建家长账号**。
> 目标园所只用于校验「该园已录入此手机号」和「绑定后的初始落点」，不是角色边界——
> 该账号在其他共用园所已有的成员关系继续生效，并在响应 `kindergartens` 中一次返回
> （不依赖成员切换接口 #169）。
>
> 待绑定 token 可访问本端点（见登录一节的白名单）；已绑定 token 重复调用用于幂等重试。

```json
// Request
{
  "wx_phone_code": "e31a2b...",        // 必填：wx.getPhoneNumber 返回的一次性 code
  "public_code": "kg1001",             // 可选：园所入口码；不传则按手机号在全部共用园所中查找
  "kindergarten_id": 4                 // 可选：public_code 的等价兜底；两者都传时 public_code 优先
}

// Response（200）
{
  "token": "eyJhbGciOiJIUzI1NiIs...",   // 新会话凭证，立即具备家长角色
  "refresh_token": "…",
  "expires_at": "2026-11-21T12:00:00Z",
  "user": {
    "id": 100,
    "phone": "138****1234",             // 脱敏返回
    "nickname": "张妈妈",
    "avatar": "",
    "gender": 0,
    "must_change_password": false,
    "roles": [
      { "role": "parent", "kindergarten_id": 4, "kindergarten_name": "阳光幼儿园" },
      { "role": "parent", "kindergarten_id": 7, "kindergarten_name": "月亮幼儿园" }
    ]
  },
  "consent_required": true,
  "current_policy_version": "v1",
  "kindergartens": [
    { "kindergarten_id": 4, "kindergarten_name": "阳光幼儿园" },
    { "kindergarten_id": 7, "kindergarten_name": "月亮幼儿园" }
  ]
}
```

- **绑定方向**：`待绑定账号 → 预创建账号`。保留预创建账号的手机号、角色、孩子与家长档案，
  只把客户端身份 `(app_id, open_id)` 挂过去；待绑定账号**硬删除**（不留软删行占用唯一键）。
- **已录入校验**：手机号必须已由某园所录入（预创建家长账号存在），否则返回 `40031`，
  不会创建账号。家长账号永远由园所先录入。
- **无 scene**：不带 `public_code` / `kindergarten_id` 时，按手机号在全部共用园所中查找；
  找到即绑定，`kindergarten_id` 取最小园所 ID，`kindergartens` 列出全部共用成员关系。
- **目标园校验顺序**：入口码无效 → `404`；客户端与园所通道不匹配（见 #177 通道矩阵）→
  `40033`；手机号未被该园录入 → `40031`。入口码与通道先于手机号判定，避免用伪造入口码
  探测手机号是否已录入。
- **家长档案**：绑定后保证预创建账号在落点园所有一条家长档案（缺则补，已有则不动）。
- **幂等**：同一微信重复绑定返回 `200`，账号、档案、身份都不重复创建。
- **限流**：按 `app_id + IP + 手机号`，20 次/分钟；Redis 故障失败开放并告警。
- **错误**：

| HTTP | code | 场景 |
| --- | --- | --- |
| 400 | 40030 | `wx_phone_code` 无效或已使用 |
| 400 | 40031 | 园所尚未录入该家长手机号（目标园未录入，或无 scene 时全部共用园所都未命中） |
| 400 | 40032 | 该手机号已绑定其他微信（同一客户端一个账号只允许一个微信） |
| 400 | 40033 | 当前小程序不支持该园所（客户端通道与园所通道不匹配） |
| 401 | — | token 缺失/无效，或 token 不含客户端身份（需重新走小程序登录） |
| 403 | — | 客户端未注册/已停用，或当前小程序不是家长端 |
| 404 | — | `public_code` / `kindergarten_id` 找不到园所 |
| 429 | — | 绑定请求过于频繁 |

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

### GET `/common/user/kindergartens`

> 我的园所（#169）：只返回「当前客户端角色族 ∩ 园所通道」过滤后的成员园，供客户端展示与切换；不提供跨租户园所目录。

```json
// Response
[
  { "kindergarten_id": 4, "name": "阳光幼儿园", "role": "parent" },
  { "kindergarten_id": 7, "name": "星光幼儿园", "role": "parent" }
]
```

- 家长端客户端只返回本人的 `parent` 园；老师端只返回 `teacher` 园。
- 共用（shared）客户端不返回 `app_channel=dedicated/group` 的园；专属客户端只返回自己 scope 的园。
- 管理端凭证（无 `app_id`）→ `403`（`40300`）；`app_id` 指向的客户端未注册或已停用 → `403`。
- 待绑定 token 在白名单内：返回空数组，不报错。

---

### POST `/auth/switch-kindergarten`

> 切换园所（#169）：仅 `parent` / `teacher` 小程序客户端。校验目标园在过滤后的成员列表内，重新签发绑定目标园的 access + refresh，并原子消费旧 refresh。

```json
// Request（Authorization: Bearer <当前 access token>）
{
  "kindergarten_id": 7,
  "refresh_token": "<当前 refresh token>"
}

// Response：与登录响应同形
{
  "token": "eyJhbGciOiJIUzI1NiIs...(new)",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...(new)",
  "expires_at": "2026-11-21T12:00:00Z",
  "consent_required": false,
  "current_policy_version": "v1",
  "user": { "id": 100, "roles": [{ "role": "parent", "kindergarten_id": 7 }] }
}
```

- `refresh_token` 必须属于同一账号、同一客户端；跨客户端 → `401`（`40100`）。
- 目标园不在成员过滤结果内（角色族/通道不匹配或非成员）→ `403`（`40300`）。
- 管理端 token（无 `app_id`）→ `403`；待绑定 token → `403` + `code=40302`。
- 旧 refresh 一次性消费（GETDEL）：并发「切换 + 续期」只有一个成功，另一个 `401`；成功后旧 refresh 不能重放。
- 响应中的 `roles` 与 token 一致，只含目标园所在客户端的角色族。

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
