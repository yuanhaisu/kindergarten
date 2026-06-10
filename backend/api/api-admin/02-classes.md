# 02. 班级管理

> 对应前端页面：`admin/pages/classes/index.vue`
>
> 基础路径：`/api/v1/admin`，需 `Authorization: Bearer <token>`，角色 `admin`。
>
> 状态：🚫 6 个接口全部待后端实现。

---

## GET `/admin/classes`

> 班级列表（支持分页、年级筛选；对应前端 `getClasses`）

```json
// Query: ?grade=小班&keyword=&page=1&page_size=20
// grade: 小班 / 中班 / 大班

// Response
{
  "list": [
    {
      "id": 1,
      "name": "小一班",
      "grade": "小班",
      "academic_year": "2025-2026",
      "capacity": 35,
      "student_count": 30,
      "teachers": [
        { "id": 1, "nickname": "李老师", "title": "head", "title_label": "主班老师" },
        { "id": 2, "nickname": "张老师", "title": "assistant", "title_label": "副班老师" },
        { "id": 3, "nickname": "刘老师", "title": "nursery", "title_label": "生活老师" }
      ],
      "status": 1,
      "created_at": "2025-09-01T08:00:00Z"
    }
  ],
  "total": 10,
  "page": 1,
  "page_size": 20
}
```

---

## GET `/admin/classes/:id`

> 班级详情（对应前端 `getClassDetail`）

```json
// Response
{
  "id": 1,
  "name": "小一班",
  "grade": "小班",
  "academic_year": "2025-2026",
  "capacity": 35,
  "student_count": 30,
  "status": 1,
  "teachers": [
    {
      "id": 1,
      "nickname": "李老师",
      "avatar": "https://cdn.example.com/avatar/1.jpg",
      "title": "head",
      "title_label": "主班老师",
      "phone": "138****1234"
    },
    {
      "id": 2,
      "nickname": "张老师",
      "avatar": "https://cdn.example.com/avatar/2.jpg",
      "title": "assistant",
      "title_label": "副班老师",
      "phone": "139****5678"
    }
  ],
  "students_preview": [
    { "id": 1, "name": "王小明", "avatar": "..." },
    { "id": 2, "name": "李小花", "avatar": "..." }
  ],
  "created_at": "2025-09-01T08:00:00Z"
}
```

---

## POST `/admin/classes`

> 创建班级（对应前端 `createClass`）

```json
// Request
{
  "name": "小三班",
  "grade": "小班",
  "academic_year": "2025-2026",
  "capacity": 35,
  "teacher_ids": [
    { "user_id": 10, "title": "head" },
    { "user_id": 11, "title": "assistant" },
    { "user_id": 12, "title": "nursery" }
  ]
}

// Response
{
  "id": 11,
  "created_at": "2026-03-19T10:00:00Z"
}
```

---

## PUT `/admin/classes/:id`

> 编辑班级（对应前端 `updateClass`）

```json
// Request
{
  "name": "小三班",
  "grade": "小班",
  "capacity": 35,
  "teacher_ids": [
    { "user_id": 10, "title": "head" },
    { "user_id": 13, "title": "assistant" },
    { "user_id": 12, "title": "nursery" }
  ]
}

// Response
{
  "id": 11,
  "updated_at": "2026-03-19T10:30:00Z"
}
```

---

## DELETE `/admin/classes/:id`

> 删除班级（仅空班级可删除；对应前端 `deleteClass`）

```json
// Response
{}
```

---

## POST `/admin/classes/:id/adjust`

> 调班（将学生调入 / 调出；对应前端 `adjustClassStudents`）

```json
// Request
{
  "action": "move_in",              // move_in=调入 / move_out=调出
  "student_ids": [1, 2, 3],
  "from_class_id": 2                // move_in 时需指定来源班级
}

// Response
{
  "adjusted_count": 3,
  "message": "成功将3位学生从小二班调入小三班"
}
```
