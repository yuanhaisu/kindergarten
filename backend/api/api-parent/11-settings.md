# 家长端 API — 设置

> 基础路径：`/api/v1/parent` | 鉴权：`Authorization: Bearer <token>`（角色 `parent`）
>
> Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。
>
> 设置按 **`(当前用户, 当前园所)`** 独立存储：家长 A 的开关不影响家长 B；同一家长在
> 不同园所可以有不同的设置口径。没有记录时按下方默认值返回（不创建记录）。

---

### GET `/parent/settings`

> 获取设置项。8 个键始终全部返回布尔值，不会返回 `null`。

```json
// Response（默认值）
{
  "notification": {
    "notice_push": true,
    "leave_push": true,
    "pickup_push": true,
    "growth_push": true,
    "task_push": false,
    "announcement_push": true
  },
  "privacy": {
    "show_phone_to_teacher": true,
    "allow_growth_share": false
  }
}
```

#### 字段与默认值

| 分组 | 键 | 含义 | 默认值 |
| --- | --- | --- | --- |
| `notification` | `notice_push` | 通知推送 | `true` |
| `notification` | `leave_push` | 请假通知 | `true` |
| `notification` | `pickup_push` | 接送通知 | `true` |
| `notification` | `growth_push` | 成长记录通知 | `true` |
| `notification` | `task_push` | 任务提醒 | `false` |
| `notification` | `announcement_push` | 公告通知 | `true` |
| `privacy` | `show_phone_to_teacher` | 向老师展示手机号 | `true` |
| `privacy` | `allow_growth_share` | 允许展示成长记录 | `false` |

> `task_push` / `allow_growth_share` 默认关，以原型 `parent/25-settings` 为准；
> 其余默认开。原型是这两项默认值的唯一依据，契约此前写全 `true`，与原型冲突。

---

### PUT `/parent/settings`

> 更新设置。支持**部分更新**：只传要改的分组/键，未出现的键保持原值。
> 同一值重复写入幂等，不报错、不产生额外记录。空对象是空操作。

```json
// Request（完整写入）
{
  "notification": {
    "notice_push": true,
    "leave_push": true,
    "pickup_push": true,
    "growth_push": false,
    "task_push": true,
    "announcement_push": true
  },
  "privacy": {
    "show_phone_to_teacher": false,
    "allow_growth_share": true
  }
}
```

```json
// Request（部分更新：只改任务提醒）
{
  "notification": { "task_push": true }
}
```

```json
// Response
{
  "updated_at": "2026-03-19T10:00:00Z"
}
```

#### 校验

- 每个键的值必须是布尔（`true` / `false`）。字符串、数字、`null` 等一律拒绝，不落库。
- 只接受 `notification` / `privacy` 两个分组；未知分组、未知键一律拒绝，避免拼错字段名后
  「保存成功但值没变」。
- 分组值回传 `null` 视为该分组不更新（兼容前端尚未渲染的分组）。
- 校验失败返回统一错误包络（`code` 非 0），设置保持原值。
