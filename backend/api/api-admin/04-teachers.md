# 04. 老师管理

> 对应前端页面：`admin/pages/teachers/index.vue`
>
> 基础路径：`/api/v1/admin`，需 `Authorization: Bearer <token>`，角色 `admin`。
>
> 状态：✅ 6 个接口已在 `server` 实现（老师 ID 为 `users.id`，档案表 `teacher_profiles`）。

---

## GET `/admin/teachers`

> 老师列表（对应前端 `getTeachers`）

```json
// Query: ?keyword=李&class_id=&status=1&page=1&page_size=20

// Response
{
  "list": [
    {
      "id": 1,
      "nickname": "李老师",
      "avatar": "https://cdn.example.com/avatar/1.jpg",
      "phone": "138****1234",
      "gender": 2,
      "classes": [
        { "class_id": 1, "class_name": "小一班", "title": "head", "title_label": "主班老师" }
      ],
      "status": 1,
      "status_label": "在职",
      "join_date": "2020-09-01",
      "created_at": "2020-09-01T08:00:00Z"
    }
  ],
  "total": 35,
  "page": 1,
  "page_size": 20
}
```

---

## GET `/admin/teachers/:id`

> 老师详情（对应前端 `getTeacher`）

```json
// Response
{
  "id": 1,
  "nickname": "李老师",
  "real_name": "李明明",
  "avatar": "https://cdn.example.com/avatar/1.jpg",
  "phone": "13812341234",
  "gender": 2,
  "id_card_number": "310101198801010001",
  "education": "本科",
  "major": "学前教育",
  "certificate_no": "教师资格证20180001",
  "classes": [
    { "class_id": 1, "class_name": "小一班", "title": "head", "title_label": "主班老师" }
  ],
  "status": 1,
  "status_label": "在职",
  "join_date": "2020-09-01",
  "stats": {
    "total_growth_logs": 580,
    "total_notices": 120,
    "avg_parent_satisfaction": 4.8
  },
  "created_at": "2020-09-01T08:00:00Z"
}
```

---

## POST `/admin/teachers`

> 添加老师（对应前端 `createTeacher`）

```json
// Request
{
  "real_name": "赵红",
  "nickname": "赵老师",
  "phone": "13700002222",
  "gender": 2,
  "education": "本科",
  "major": "学前教育",
  "certificate_no": "教师资格证20260001",
  "join_date": "2026-03-01",
  "class_assignments": [
    { "class_id": 3, "title": "assistant" }
  ]
}

// Response
{
  "id": 36,
  "created_at": "2026-03-19T10:00:00Z"
}
```

---

## PUT `/admin/teachers/:id`

> 编辑老师（对应前端 `updateTeacher`）

```json
// Request
{
  "nickname": "赵老师",
  "education": "硕士",
  "class_assignments": [
    { "class_id": 3, "title": "head" }
  ]
}

// Response
{
  "id": 36,
  "updated_at": "2026-03-19T10:30:00Z"
}
```

---

## DELETE `/admin/teachers/:id`

> 删除老师（将同时解除班级关联；对应前端 `deleteTeacher`）

```json
// Response
{}
```

---

## PUT `/admin/teachers/:id/status`

> 修改老师状态（离职 / 复职；对应前端 `updateTeacherStatus`）

```json
// Request
{
  "status": 2,                  // 1=在职 2=离职
  "reason": "个人原因"
}

// Response
{
  "id": 1,
  "status": 2,
  "status_label": "离职",
  "updated_at": "2026-03-19T10:00:00Z"
}
```
