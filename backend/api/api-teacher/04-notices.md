# 模块 4：通知管理 API

> 对应开发计划：模块 4：通知管理
>
> 基础路径：`/api/v1/teacher`
>
> 认证：`Authorization: Bearer <token>`，用户角色为 `teacher`
>
> 以下 Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。

> ⚠️ **契约变更（notice-multitarget，2026-10-09）**：
> - 发布/编辑入参新增 `class_ids`（班级目标集合，必填非空，均须为任教班）；单值 `class_id` **删除**——多目标下保留会歧义
> - 列表项/详情新增 `class_names`（目标班级名列表）作为「接收范围」展示字段
> - `read_count` / `total_count` / `read_rate` / `read_stats`：**形状不变，语义变**——分母统一为「目标范围去重家长数」（同一家长多孩只计一人），不再是单班学生数

> ⚠️ **契约变更（notice-drafts-attachments，2026-10-29）**：
> - `POST` / `PUT` 新增**可选**入参 `status`（0 草稿 / 1 定时 / 2 发布）。`POST` 省略等价 2（保持旧调用兼容），`PUT` 省略表示保持当前状态；`PUT` 显式传 2 = 发布草稿 / 立即发布。`status=3` 只能由撤回接口产生。
> - 完整性：草稿（0）要求标题或正文至少一项非空，`class_ids` 可为空；定时（1）与发布（2）要求标题、正文、`class_ids` 均非空。
> - `scheduled_at` 必须是**带时区的 RFC3339**（如 `2026-11-01T09:00:00+08:00`）且晚于当前时间；缺失、无时区或已过期一律 400。
> - 附件：`file_type` 仅接受 `image`，单条通知最多 9 张，单张 `file_size` ≤ 20MB；不合法整单拒绝。创建/编辑按提交列表**整体替换**。
> - 可见性：草稿仅创建者可见（列表与详情）；`GET /teacher/notices` 不传 `status` 时不返回他人草稿，传 `status=0` 只返回自己的草稿。
> - 状态机：仅 0/1 可编辑；仅 2 可撤回；仅 0 可删除；家长确认只对 `status=2` 生效。
> - 响应字段落实：详情返回 `attachments` 与 `scheduled_at`；列表 `attachments_count` 为真实计数（此前恒为 0）。

## 接口清单

- `GET /teacher/notices`
- `GET /teacher/notices/:id`
- `POST /teacher/notices`
- `PUT /teacher/notices/:id`
- `DELETE /teacher/notices/:id`
- `PUT /teacher/notices/:id/revoke`
- `POST /teacher/notices/:id/remind`
- `GET /teacher/notices/:id/read-status`
- `GET /teacher/notices/templates`
- `POST /teacher/notices/templates`
- `PUT /teacher/notices/templates/:id`
- `DELETE /teacher/notices/templates/:id`

---

## 2. 通知管理

### GET `/teacher/notices`

> 通知列表

```json
// Query: ?status=2&category=homework&page=1&page_size=20
// status: 0=草稿 1=定时待发 2=已发布 3=已撤回，不传查全部
// category: homework / bring / activity / health / other，不传查全部
// 可见性：草稿仅创建者可见——不传 status 时排除他人草稿；传 status=0 时只返回自己的草稿
// attachments_count: 该通知实际附件条数

// Response
{
  "list": [
    {
      "id": 1,
      "category": "homework",
      "category_label": "作业",
      "title": "周末作业通知",
      "content": "各位家长您好，本周作业如下：1. 复习本周学过的儿歌...",
      "class_names": ["小一班", "小二班"],
      "need_confirm": true,
      "status": 2,
      "status_label": "已发布",
      "read_count": 25,
      "total_count": 30,
      "read_rate": "83%",
      "attachments_count": 2,
      "published_at": "2026-03-19T09:00:00Z",
      "created_at": "2026-03-19T08:50:00Z"
    }
  ],
  "total": 36,
  "page": 1,
  "page_size": 20
}
```

---

### GET `/teacher/notices/:id`

> 通知详情（含已读统计）。草稿仅创建者可读，其他老师得到 404。

```json
// Response
{
  "id": 1,
  "category": "homework",
  "category_label": "作业",
  "title": "周末作业通知",
  "content": "各位家长您好，本周作业如下：1. 复习本周学过的儿歌；2. 完成手工制作《我的家》...",
  "class_names": ["小一班", "小二班"],
  "need_confirm": true,
  "status": 2,
  "scheduled_at": null,
  "published_at": "2026-03-19T09:00:00Z",
  "attachments": [
    {
      "id": 10,
      "file_url": "https://cdn.example.com/uploads/notice/1.jpg",
      "file_type": "image",
      "file_name": "活动照片1.jpg",
      "file_size": 512000
    }
  ],
  "read_stats": {
    "read_count": 25,
    "unread_count": 5,
    "confirmed_count": 20,
    "total_count": 30,            // 分母=目标范围去重家长数（多孩只计一人）；形状不变，口径已变
    "read_rate": "83%"
  },
  "creator": {
    "id": 1,
    "nickname": "李老师",
    "avatar": "https://cdn.example.com/avatar/1.jpg"
  },
  "created_at": "2026-03-19T08:50:00Z"
}
```

---

### POST `/teacher/notices`

> 创建 / 发布通知

```json
// Request
{
  "category": "homework",
  "title": "周末作业通知",
  "content": "各位家长您好，本周作业如下...",
  "class_ids": [1, 2],            // 接收班级；status=0 草稿可为空，status=1/2 必填非空；均须为任教班级，否则拒绝
  "status": 2,                    // 可选：0 草稿 / 1 定时待发 / 2 立即发布；省略等价 2
  "attachments": [
    {
      "file_url": "https://cdn.example.com/uploads/notice/1.jpg",
      "file_type": "image",       // 仅 image；最多 9 张；单张 file_size ≤ 20971520
      "file_name": "活动照片1.jpg",
      "file_size": 512000
    }
  ],
  "need_confirm": true,
  "scheduled_at": null,           // status=1 必填：带时区 RFC3339 且晚于当前时间；其余状态忽略
  "template_id": null             // 使用的模板 ID，可选
}

// 状态与完整性
// status=0 草稿：标题或正文至少一项非空；class_ids 可为空
// status=1 定时：标题、正文、class_ids 均非空；scheduled_at 必须是 `2026-11-01T09:00:00+08:00` 形式
// status=2 发布：标题、正文、class_ids 均非空
// 附件校验先于落库：非 image / 超过 9 张 / 单张超 20MB → 400，整单不保存

// Response
{
  "id": 1,
  "status": 2,
  "status_label": "已发布",
  "created_at": "2026-03-19T09:00:00Z"
}
```

---

### PUT `/teacher/notices/:id`

> 编辑通知（仅草稿 / 定时待发状态可编辑；已发布/已撤回返回业务错误）
>
> `status` 省略 = 保持当前状态；显式传 2 = 发布草稿 / 立即发布（校验完整性并写 `published_at`）；显式传 0/1 可把草稿与定时互相切换。`attachments` 按提交列表整体替换（不传即清空）。

```json
// Request
{
  "category": "homework",
  "title": "周末作业通知（修改版）",
  "content": "各位家长您好，本周作业更新如下...",
  "class_ids": [1, 2],            // status=0 可为空（草稿可先不选班），status=1/2 必填非空
  "status": 2,                    // 可选：0 草稿 / 1 定时待发 / 2 发布；省略保持当前状态
  "attachments": [
    {
      "file_url": "https://cdn.example.com/uploads/notice/1.jpg",
      "file_type": "image",
      "file_name": "活动照片1.jpg",
      "file_size": 512000
    }
  ],
  "need_confirm": true,
  "scheduled_at": null            // status=1 必填：带时区 RFC3339 且晚于当前时间
}

// 完整性：status=0 要求标题或正文至少一项非空；status=1/2 要求标题、正文、class_ids 均非空。
// 发布失败（如正文为空）返回 400 且草稿内容不变。

// Response
{
  "id": 1,
  "status": 0,
  "updated_at": "2026-03-19T09:10:00Z"
}
```

---

### DELETE `/teacher/notices/:id`

> 删除通知（仅草稿可删除；非草稿返回 400 业务错误，需先撤回/等待发送）

```json
// Response
{}
```

---

### PUT `/teacher/notices/:id/revoke`

> 撤回已发布的通知（仅 `status=2` 可撤回；草稿/定时返回 400 业务错误）

```json
// Request
{}

// Response
{
  "id": 1,
  "status": 3,
  "status_label": "已撤回"
}
```

---

### POST `/teacher/notices/:id/remind`

> 一键提醒未读家长

```json
// Request
{}

// Response
{
  "reminded_count": 5,
  "message": "已向5位未读家长发送提醒"
}
```

---

### GET `/teacher/notices/:id/read-status`

> 已读 / 未读家长名单

```json
// Query: ?status=read&page=1&page_size=50
// status: read=已读 / unread=未读 / confirmed=已确认，不传查全部

// Response
{
  "list": [
    {
      "user_id": 10,
      "parent_name": "王小明妈妈",
      "avatar": "https://cdn.example.com/avatar/10.jpg",
      "student_name": "王小明",
      "relation": "mother",
      "is_read": true,
      "read_at": "2026-03-19T09:15:00Z",
      "is_confirmed": true,
      "confirmed_at": "2026-03-19T09:16:00Z"
    },
    {
      "user_id": 11,
      "parent_name": "李小花爸爸",
      "avatar": "https://cdn.example.com/avatar/11.jpg",
      "student_name": "李小花",
      "relation": "father",
      "is_read": false,
      "read_at": null,
      "is_confirmed": false,
      "confirmed_at": null
    }
  ],
  "total": 30,
  "read_count": 25,
  "unread_count": 5,
  "confirmed_count": 20
}
```

---

### GET `/teacher/notices/templates`

> 通知模板列表

```json
// Query: ?category=homework&page=1&page_size=20

// Response
{
  "list": [
    {
      "id": 1,
      "category": "homework",
      "category_label": "作业",
      "title": "周末作业通知",
      "content": "各位家长您好，本周作业如下：1. 复习本周学过的儿歌；2. 完成手工制作《我的家》；3. 请家长和孩子一起阅读绘本30分钟。请于下周一前完成，感谢配合！",
      "usage_count": 12,
      "created_at": "2026-01-15T10:00:00Z"
    },
    {
      "id": 2,
      "category": "activity",
      "category_label": "活动",
      "title": "春季出游通知",
      "content": "各位家长您好，我们将于3月25日组织春季出游活动...",
      "usage_count": 3,
      "created_at": "2026-02-20T14:00:00Z"
    }
  ],
  "total": 8
}
```

---

### POST `/teacher/notices/templates`

> 创建通知模板

```json
// Request
{
  "category": "homework",
  "title": "每日作业通知",
  "content": "各位家长您好，今日作业如下：..."
}

// Response
{
  "id": 3,
  "created_at": "2026-03-19T10:00:00Z"
}
```

---

### PUT `/teacher/notices/templates/:id`

> 编辑模板

```json
// Request
{
  "category": "homework",
  "title": "每日作业通知（更新版）",
  "content": "各位家长您好，今日作业更新如下：..."
}

// Response
{
  "id": 3,
  "updated_at": "2026-03-19T10:30:00Z"
}
```

---

### DELETE `/teacher/notices/templates/:id`

> 删除模板

```json
// Response
{}
```
