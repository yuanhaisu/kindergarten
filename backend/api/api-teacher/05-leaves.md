# 模块 5：请假审批 API

> 对应开发计划：模块 5：请假审批
>
> 基础路径：`/api/v1/teacher`
>
> 认证：`Authorization: Bearer <token>`，用户角色为 `teacher`
>
> 以下 Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。

## 接口清单

- `GET /teacher/leaves`
- `GET /teacher/leaves/stats`
- `GET /teacher/leaves/:id`
- `PUT /teacher/leaves/:id/approve`
- `PUT /teacher/leaves/:id/reject`

---

## 3. 请假审批

### GET `/teacher/leaves`

> 请假申请列表

```json
// Query: ?status=pending&page=1&page_size=20
// status: pending=待审批 / approved=已同意 / rejected=已拒绝，不传查全部

// Response
{
  "list": [
    {
      "id": 101,
      "student": {
        "id": 1,
        "name": "王小明",
        "avatar": "https://cdn.example.com/avatar/s1.jpg"
      },
      "applicant": {
        "id": 10,
        "nickname": "王小明妈妈",
        "relation": "mother"
      },
      "reason": "感冒发烧，身体不适",
      "start_date": "2026-03-19",
      "end_date": "2026-03-21",
      "days": 3,
      "status": 0,
      "status_label": "待审批",
      "created_at": "2026-03-19T07:30:00Z"
    },
    {
      "id": 102,
      "student": {
        "id": 2,
        "name": "李小花",
        "avatar": "https://cdn.example.com/avatar/s2.jpg"
      },
      "applicant": {
        "id": 11,
        "nickname": "李小花爸爸",
        "relation": "father"
      },
      "reason": "家中有事，需带孩子外出",
      "start_date": "2026-03-20",
      "end_date": "2026-03-20",
      "days": 1,
      "status": 0,
      "status_label": "待审批",
      "created_at": "2026-03-18T20:15:00Z"
    }
  ],
  "total": 33,
  "page": 1,
  "page_size": 20
}
```

---

### GET `/teacher/leaves/stats`

> 请假统计（顶部统计卡片）

```json
// Response
{
  "pending": 3,
  "approved": 28,
  "rejected": 2,
  "monthly_total": 33
}
```

---

### GET `/teacher/leaves/:id`

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
  "applicant": {
    "id": 10,
    "nickname": "王小明妈妈",
    "avatar": "https://cdn.example.com/avatar/10.jpg",
    "relation": "mother",
    "phone": "139****6666"
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
  "status": 0,
  "status_label": "待审批",
  "reviewer": null,
  "reviewed_at": null,
  "review_remark": "",
  "created_at": "2026-03-19T07:30:00Z"
}
```

---

### PUT `/teacher/leaves/:id/approve`

> 同意请假

```json
// Request
{
  "remark": "注意休息，祝早日康复"    // 可选
}

// Response
{
  "id": 101,
  "status": 1,
  "status_label": "已同意",
  "reviewed_at": "2026-03-19T08:00:00Z"
}
```

---

### PUT `/teacher/leaves/:id/reject`

> 拒绝请假

```json
// Request
{
  "remark": "请假天数过长，建议调整"   // 必填
}

// Response
{
  "id": 101,
  "status": 2,
  "status_label": "已拒绝",
  "reviewed_at": "2026-03-19T08:00:00Z"
}
```
