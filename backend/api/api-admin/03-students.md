# 03. 学生管理

> 对应前端页面：`admin/pages/students/index.vue`
>
> 基础路径：`/api/v1/admin`，需 `Authorization: Bearer <token>`，角色 `admin`。
>
> 状态：✅ 8 个接口已在 `server` 实现（导出为文件流，无 JSON 信封）。

---

## GET `/admin/students`

> 学生列表（分页、班级、状态筛选；对应前端 `getStudents`）

```json
// Query: ?class_id=1&keyword=王小明&status=1&page=1&page_size=20
// status: 1=在读 2=毕业 3=退学 0=待入学

// Response
{
  "list": [
    {
      "id": 1,
      "name": "王小明",
      "avatar": "https://cdn.example.com/avatar/s1.jpg",
      "gender": 1,
      "birthday": "2021-09-15",
      "age": "4岁半",
      "student_no": "2025001",
      "class_name": "小一班",
      "class_id": 1,
      "status": 1,
      "status_label": "在读",
      "enroll_date": "2025-09-01",
      "parents_count": 2,
      "created_at": "2025-09-01T08:00:00Z"
    }
  ],
  "total": 300,
  "page": 1,
  "page_size": 20
}
```

---

## GET `/admin/students/:id`

> 学生详情（对应前端 `getStudent`）

```json
// Response
{
  "id": 1,
  "name": "王小明",
  "avatar": "https://cdn.example.com/avatar/s1.jpg",
  "gender": 1,
  "birthday": "2021-09-15",
  "age": "4岁半",
  "student_no": "2025001",
  "id_card_number": "310101202109150001",
  "health_note": "对花生过敏",
  "address": "北京市朝阳区XXX小区",
  "class_id": 1,
  "class_name": "小一班",
  "enroll_date": "2025-09-01",
  "status": 1,
  "status_label": "在读",
  "parents": [
    {
      "user_id": 10,
      "name": "王明",
      "phone": "13888888888",
      "relation": "father",
      "relation_label": "爸爸",
      "is_primary": true,
      "avatar": "https://cdn.example.com/avatar/10.jpg",
      "id_card_number": "310101199001010001"
    },
    {
      "user_id": 11,
      "name": "李丽",
      "phone": "13966666666",
      "relation": "mother",
      "relation_label": "妈妈",
      "is_primary": false,
      "avatar": "https://cdn.example.com/avatar/11.jpg",
      "id_card_number": "310101199205050002"
    }
  ],
  "pickup_authorizations": [
    {
      "id": 1,
      "name": "爷爷（王建国）",
      "relation": "grandfather",
      "phone": "13655555555",
      "status": 1
    }
  ],
  "created_at": "2025-09-01T08:00:00Z"
}
```

---

## POST `/admin/students`

> 添加学生（对应前端 `createStudent`）

```json
// Request
{
  "name": "赵小刚",
  "gender": 1,
  "birthday": "2022-01-20",
  "id_card_number": "310101202201200001",
  "health_note": "",
  "address": "北京市海淀区XXX小区",
  "class_id": 1,
  "enroll_date": "2026-03-01",
  "parents": [
    {
      "name": "赵大刚",
      "phone": "13700001111",
      "relation": "father",
      "is_primary": true,
      "id_card_number": "310101199001010099"
    }
  ]
}

// Response
{
  "id": 301,
  "student_no": "2026301",
  "created_at": "2026-03-19T10:00:00Z"
}
```

---

## PUT `/admin/students/:id`

> 编辑学生（对应前端 `updateStudent`）

```json
// Request
{
  "name": "赵小刚",
  "health_note": "轻度近视",
  "address": "北京市海淀区XXX小区2单元",
  "class_id": 1
}

// Response
{
  "id": 301,
  "updated_at": "2026-03-19T10:30:00Z"
}
```

---

## DELETE `/admin/students/:id`

> 删除学生（对应前端 `deleteStudent`）

```json
// Response
{}
```

---

## PUT `/admin/students/:id/status`

> 修改学生状态（毕业 / 退学 / 恢复；对应前端 `updateStudentStatus`）

```json
// Request
{
  "status": 2,                  // 1=在读 2=毕业 3=退学
  "reason": "已达毕业年龄"
}

// Response
{
  "id": 1,
  "status": 2,
  "status_label": "毕业",
  "updated_at": "2026-07-01T00:00:00Z"
}
```

---

## POST `/admin/students/import`

> 批量导入学生（Excel；对应前端 `importStudents`）

```
Content-Type: multipart/form-data
字段: file — Excel 文件（.xlsx）
```

```json
// Response
{
  "total": 30,
  "success_count": 28,
  "fail_count": 2,
  "fail_details": [
    { "row": 15, "name": "张三", "reason": "手机号格式错误" },
    { "row": 22, "name": "李四", "reason": "班级不存在" }
  ]
}
```

---

## GET `/admin/students/export`

> 导出学生名册（Blob；对应前端 `exportStudentsBlob`）

```json
// Query: ?class_id=1&status=1&format=xlsx
// 返回文件流，Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
```
