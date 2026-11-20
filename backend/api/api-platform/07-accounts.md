# 07. 平台账号管理

> 对应前端页面：`admin/pages/platform/accounts`（前端切片 T6 落地）
>
> 基础路径：`/api/v1/platform`
>
> 平台账号独立于园所账号（ADR-0014）。账号响应**不包含** `password_hash` 与 `role`；
> `role` 是预留列，当前固定为 `platform`，不参与鉴权。

---

## 通用规则

- **可用账号口径**：`status = enabled` 且未被删除。停用账号不计入。
- **守卫**：
  - 不能停用当前登录账号（后端以令牌中的账号 ID 判定），返回 `409`。
  - 停用前统计可用账号，只剩一个时拒绝，返回 `409`，避免平台被锁死。
- **口令策略**：至少 8 位，且同时包含大写字母、小写字母和数字。新建与重置由操作者提供口令，
  响应不回显；修改本人密码需要校验原密码。
- **凭证失效**：停用、重置他人密码、修改本人密码都会使该账号此前签发的平台访问令牌立即失效
  （服务端写入账号级失效时间）。失效存储不可用（Redis 故障）时操作**失败并返回 `503`**，
  数据库保持不变。
- **审计**：新建、停用/启用、重置密码与账号变更写同一数据库事务，审计失败则操作整体回滚；
  修改本人密码的审计是 best-effort，不阻断改密。

---

## GET `/platform/admins`

> 平台账号列表

```json
// GET /platform/admins?page=1&page_size=20&keyword=ops

// Response
{
  "list": [
    {
      "id": 3,
      "username": "ops",
      "name": "运营",
      "status": "enabled",
      "created_at": "2026-11-18T10:00:00+08:00"
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 20
}
```

### 查询参数

| 参数 | 类型 | 说明 |
|---|---|---|
| `page` / `page_size` | int | 分页，默认 `1` / `20`，`page_size` 上限 100 |
| `keyword` | string | 模糊匹配用户名或姓名 |

### 响应字段

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | uint64 | 账号 ID |
| `username` | string | 登录用户名 |
| `name` | string | 显示名 |
| `status` | string | `enabled` 启用 / `disabled` 停用 |
| `created_at` | RFC 3339 | 创建时间 |

---

## POST `/platform/admins`

> 新建平台账号。新账号权限与既有账号一致，用户名全局唯一。

```json
// Request
{
  "username": "finance",
  "name": "财务",
  "password": "Abc12345"
}

// Response 201
{
  "id": 4,
  "username": "finance",
  "name": "财务",
  "status": "enabled",
  "created_at": "2026-11-18T10:05:00+08:00"
}
```

### 请求字段

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `username` | string | 是 | 3–64 字符，去除首尾空格后不能为空 |
| `name` | string | 否 | 显示名，≤ 64 字符 |
| `password` | string | 是 | 满足口令策略 |

### 错误

| HTTP | 场景 |
|---|---|
| `409` | 用户名已存在 |
| `422` | 口令强度不足 / 用户名为空 |

---

## PATCH `/platform/admins/:id/status`

> 启用或停用平台账号。停用后该账号立即无法登录，既有令牌同时失效。

```json
// Request
{ "status": "disabled" }

// Response
{
  "id": 4,
  "status": "disabled",
  "updated_at": "2026-11-18T10:10:00+08:00"
}
```

### 请求字段

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `status` | string | 是 | `enabled` 或 `disabled` |

### 错误

| HTTP | 场景 |
|---|---|
| `404` | 账号不存在 |
| `409` | 不能停用当前登录账号 / 这是最后一个可用账号 |
| `422` | 状态取值不合法 |
| `503` | 凭证失效存储不可用（停用时） |

> 状态未变化时幂等返回成功，不重复写审计，也不重复失效凭证。

---

## POST `/platform/admins/:id/reset-password`

> 运营重置指定账号口令。口令由操作者提供，响应不回显。
> 该账号此前签发的令牌立即失效。

```json
// Request
{ "password": "Xyz98765" }

// Response
{
  "id": 4,
  "updated_at": "2026-11-18T10:15:00+08:00"
}
```

### 请求字段

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `password` | string | 是 | 满足口令策略 |

### 错误

| HTTP | 场景 |
|---|---|
| `404` | 账号不存在 |
| `422` | 口令强度不足 |
| `503` | 凭证失效存储不可用，口令未变更 |

---

## PUT `/platform/admins/me/password`

> 当前登录账号修改本人密码。成功后该账号所有平台令牌（包括本次请求的令牌）立即失效，
> 客户端必须回到登录页。

```json
// Request
{
  "old_password": "Abc12345",
  "new_password": "NewPass123"
}

// Response
{ "changed": true }
```

### 请求字段

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `old_password` | string | 是 | 当前口令 |
| `new_password` | string | 是 | 满足口令策略 |

### 错误

| HTTP | 场景 |
|---|---|
| `404` | 账号不存在 |
| `422` | 原密码不正确 / 新口令强度不足 |
| `503` | 凭证失效存储不可用，口令未变更 |

---

## 审计动作对照

| 动作 | `module` | `action` | 一致性 |
|---|---|---|---|
| 新建账号 | `account` | `create` | 与账号写入同事务 |
| 停用账号 | `account` | `disable` | 与状态变更同事务 |
| 启用账号 | `account` | `enable` | 与状态变更同事务 |
| 重置他人密码 | `account` | `reset_password` | 与口令变更同事务 |
| 修改本人密码 | `account` | `change_password` | best-effort |

账号类审计记录的 `target_type = platform_admin`、`target_id = 账号 ID`，
`kindergarten_id = 0`，`request_id` 与响应头 `X-Request-ID` 一致。
