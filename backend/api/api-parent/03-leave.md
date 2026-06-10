# 家长端 API — 请假

> 基础路径：`/api/v1/parent` | 鉴权：`Authorization: Bearer <token>`（角色 `parent`）
>
> Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。

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
  "total": 5,
  "page": 1,
  "page_size": 20
}
```

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
  "reason": "感冒发烧，身体不适",
  "start_date": "2026-03-19",
  "end_date": "2026-03-21",
  "days": 3,
  "attachments": [
    "https://cdn.example.com/uploads/leave/diagnosis.jpg"
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

---

### POST `/parent/leaves`

> 提交请假申请

```json
// Request
{
  "student_id": 1,
  "reason": "感冒发烧，身体不适",
  "start_date": "2026-03-19",
  "end_date": "2026-03-21",
  "attachments": [
    "https://cdn.example.com/uploads/leave/diagnosis.jpg"
  ],
  "emergency_contact": {
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

> 撤销请假申请（仅待审批状态可撤销）

```json
// Request
{}

// Response
{
  "id": 101,
  "status": 3,
  "status_label": "已撤销"
}
```
