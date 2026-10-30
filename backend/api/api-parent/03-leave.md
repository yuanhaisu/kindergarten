# 家长端 API — 请假

> 基础路径：`/api/v1/parent` | 鉴权：`Authorization: Bearer <token>`（角色 `parent`）
>
> Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。
>
> `status_label` 全端统一：`0 待审批` / `1 已同意` / `2 已拒绝` / `3 已取消`。
> 请假类型 `leave_type`：`sick 病假` / `personal 事假` / `other 其他`。

---

### GET `/parent/leaves`

> 请假记录列表

```json
// Query: ?status=&page=1&page_size=20
// status: pending / approved / rejected

// Response
{
  "list": [
    {
      "id": 101,
      "student": { "id": 1, "name": "王小明", "avatar": "..." },
      "leave_type": "sick",
      "leave_type_label": "病假",
      "reason": "感冒发烧，身体不适",
      "start_date": "2026-03-19",
      "end_date": "2026-03-21",
      "days": 3,
      "status": 1,
      "status_label": "已同意",
      "reviewer": { "nickname": "李老师" },
      "reviewed_at": "2026-03-19T08:00:00Z",
      "created_at": "2026-03-19T07:30:00Z"
    }
  ],
  "stats": { "all": 6, "pending": 1, "approved": 2 },
  "total": 5,
  "page": 1,
  "page_size": 20
}
```

> `stats` 为该家长全部请假的全量计数（不随 `status` 筛选变化）；`reviewer` 无审批时为 `null`；`leave_type` 缺省为 `other`。

---

### GET `/parent/leaves/:id`

> 请假详情

```json
// Response
{
  "id": 101,
  "student": {
    "id": 1,
    "name": "王小明",
    "avatar": "https://cdn.example.com/avatar/s1.jpg",
    "class_name": "小一班"
  },
  "leave_type": "sick",
  "leave_type_label": "病假",
  "reason": "感冒发烧，身体不适",
  "start_date": "2026-03-19",
  "end_date": "2026-03-21",
  "days": 3,
  "attachments": [
    {
      "id": 1,
      "file_url": "https://cdn.example.com/uploads/leave/diagnosis.jpg",
      "file_type": "image",
      "file_name": "diagnosis.jpg",
      "file_size": 512000
    }
  ],
  "emergency_contact": {
    "name": "爸爸（王明）",
    "phone": "138****8888"
  },
  "status": 1,
  "status_label": "已同意",
  "reviewer": {
    "id": 1,
    "nickname": "李老师",
    "avatar": "https://cdn.example.com/avatar/1.jpg"
  },
  "reviewed_at": "2026-03-19T08:00:00Z",
  "review_remark": "注意休息，祝早日康复",
  "created_at": "2026-03-19T07:30:00Z"
}
```

> `attachments` 按上传顺序返回（`sort_num, id` 升序），最多 3 张；无附件时为 `[]`。
> `emergency_contact` 选填，未填写时为 `null`；`reviewer` 无审批时为 `null`。

---

### POST `/parent/leaves`

> 提交请假申请

```json
// Request
{
  "student_id": 1,
  "leave_type": "sick",          // 可选：sick / personal / other，缺省 other
  "reason": "感冒发烧，身体不适",
  "start_date": "2026-03-19",
  "end_date": "2026-03-21",
  "attachments": [
    {
      "file_url": "https://cdn.example.com/uploads/leave/diagnosis.jpg",
      "file_type": "image",
      "file_name": "diagnosis.jpg",
      "file_size": 512000
    }
  ],
  "emergency_contact": {         // 选填
    "name": "爸爸（王明）",
    "phone": "13888888888"
  }
}

// Response
{
  "id": 101,
  "status": 0,
  "status_label": "待审批",
  "created_at": "2026-03-19T07:30:00Z"
}
```

> `attachments` 最多 3 张、仅图片（`file_type=image`）、单张不超过 20MB；任一不满足返回参数错误（422）且整单不创建。

---

### PUT `/parent/leaves/:id`

> 修改请假申请（仅待审批状态可修改）

```json
// Request
{
  "reason": "感冒发烧，高烧不退",
  "end_date": "2026-03-22",
  "attachments": [
    "https://cdn.example.com/uploads/leave/diagnosis.jpg",
    "https://cdn.example.com/uploads/leave/prescription.jpg"
  ]
}

// Response
{
  "id": 101,
  "updated_at": "2026-03-19T07:45:00Z"
}
```

---

### PUT `/parent/leaves/:id/cancel`

> 撤销请假申请（仅待审批状态可撤销；已同意 / 已拒绝 / 已取消返回参数错误，状态不变）

```json
// Request
{}

// Response
{
  "id": 101,
  "status": 3,
  "status_label": "已取消"
}
```
