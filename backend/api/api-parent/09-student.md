# 家长端 API — 学生信息

> 基础路径：`/api/v1/parent` | 鉴权：`Authorization: Bearer <token>`（角色 `parent`）
>
> Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。

---

### GET `/parent/student/profile`

> 获取孩子信息

```json
// Query: ?student_id=1

// Response
{
  "id": 1,
  "name": "王小明",
  "avatar": "https://cdn.example.com/avatar/s1.jpg",
  "gender": 1,
  "birthday": "2021-09-15",
  "age": "4岁半",
  "student_no": "2025001",
  "health_note": "对花生过敏",
  "class_name": "小一班",
  "enroll_date": "2025-09-01",
  "parents": [
    { "name": "王明", "relation": "father", "relation_label": "爸爸", "is_primary": true },
    { "name": "李丽", "relation": "mother", "relation_label": "妈妈", "is_primary": false }
  ]
}
```

---

### PUT `/parent/student/profile`

> 更新孩子信息（家长可修改有限字段）

```json
// Request
{
  "student_id": 1,
  "avatar": "https://cdn.example.com/uploads/student_new.jpg",
  "health_note": "对花生和虾过敏"
}

// Response
{
  "id": 1,
  "avatar": "https://cdn.example.com/uploads/student_new.jpg",
  "health_note": "对花生和虾过敏",
  "updated_at": "2026-03-19T10:00:00Z"
}
```

---

### GET `/parent/students`

> 家长绑定的孩子列表（多孩）

```json
// Response
{
  "list": [
    {
      "id": 1,
      "name": "王小明",
      "avatar": "https://cdn.example.com/avatar/s1.jpg",
      "gender": 1,
      "class_name": "小一班",
      "relation": "father",
      "is_current": true
    },
    {
      "id": 5,
      "name": "王小花",
      "avatar": "https://cdn.example.com/avatar/s5.jpg",
      "gender": 2,
      "class_name": "大二班",
      "relation": "father",
      "is_current": false
    }
  ]
}
```

---

### POST `/parent/students/bind`

> 绑定孩子（通过邀请码或扫码）

```json
// Request
{
  "invite_code": "ABC12345"
}

// Response
{
  "student": {
    "id": 3,
    "name": "王小华",
    "avatar": "https://cdn.example.com/avatar/s3.jpg",
    "class_name": "小二班",
    "relation": "father"
  },
  "created_at": "2026-03-20T10:00:00Z"
}
```

---

### DELETE `/parent/students/:id`

> 解除孩子绑定

```json
// Response
{}
```
