# 模块 7：接送管理 API

> 对应开发计划：模块 7：接送管理
>
> 基础路径：`/api/v1/teacher`
>
> 认证：`Authorization: Bearer <token>`，用户角色为 `teacher`
>
> 以下 Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。

## 接口清单

- `GET /teacher/pickup/overview`
- `GET /teacher/pickup/students`
- `POST /teacher/pickup/scan`
- `POST /teacher/pickup/manual`
- `GET /teacher/pickup/records`
- `GET /teacher/pickup/records/:id`
- `PUT /teacher/pickup/records/:id/abnormal`

---

## 6. 接送管理

### GET `/teacher/pickup/overview`

> 今日接送统计

```json
// Response
{
  "date": "2026-03-19",
  "arrived": 28,
  "picked_up": 12,
  "not_picked_up": 16,
  "abnormal": 0
}
```

---

### GET `/teacher/pickup/students`

> 未接走 / 全部学生接送状态列表

```json
// Query: ?status=not_picked_up
// status: arrived=已入园 / picked_up=已接走 / not_picked_up=未接走，不传查全部

// Response
{
  "list": [
    {
      "student_id": 1,
      "name": "王小明",
      "avatar": "https://cdn.example.com/avatar/s1.jpg",
      "authorized_persons": "爸爸、妈妈、爷爷",
      "arrive_time": "07:52",
      "pickup_status": "not_picked_up"
    },
    {
      "student_id": 2,
      "name": "李小花",
      "avatar": "https://cdn.example.com/avatar/s2.jpg",
      "authorized_persons": "妈妈、奶奶",
      "arrive_time": "08:05",
      "pickup_status": "not_picked_up"
    }
  ],
  "total": 16
}
```

---

### POST `/teacher/pickup/scan`

> 扫码接送登记

```json
// Request
{
  "qr_code": "PICKUP_CODE_xxx"
}

// Response
{
  "record_id": 301,
  "student": { "id": 1, "name": "王小明", "avatar": "..." },
  "pickup_person": {
    "name": "爸爸（王明）",
    "relation": "father",
    "phone": "138****8888"
  },
  "type": 2,
  "type_label": "接走",
  "is_authorized": true,
  "recorded_at": "2026-03-19T17:30:25Z"
}
```

---

### POST `/teacher/pickup/manual`

> 手动接送登记

```json
// Request
{
  "student_id": 1,
  "type": 2,                     // 1=送入园 2=接走
  "pickup_name": "爸爸（王明）",
  "pickup_relation": "father",
  "remark": ""
}

// Response
{
  "record_id": 302,
  "student": { "id": 1, "name": "王小明" },
  "type": 2,
  "type_label": "接走",
  "is_authorized": true,
  "recorded_at": "2026-03-19T17:35:00Z"
}
```

---

### GET `/teacher/pickup/records`

> 接送记录列表

```json
// Query: ?date=2026-03-19&student_id=&page=1&page_size=20

// Response
{
  "list": [
    {
      "id": 301,
      "student": { "id": 1, "name": "王小明", "avatar": "..." },
      "type": 2,
      "type_label": "接走",
      "method": 1,
      "method_label": "扫码",
      "pickup_name": "爸爸（王明）",
      "pickup_relation": "father",
      "operator": { "id": 5, "nickname": "李师傅" },
      "is_abnormal": false,
      "recorded_at": "2026-03-19T17:30:25Z"
    },
    {
      "id": 300,
      "student": { "id": 1, "name": "王小明", "avatar": "..." },
      "type": 1,
      "type_label": "入园",
      "method": 1,
      "method_label": "扫码",
      "pickup_name": "妈妈（李丽）",
      "pickup_relation": "mother",
      "operator": { "id": 5, "nickname": "李师傅" },
      "is_abnormal": false,
      "recorded_at": "2026-03-19T07:52:18Z"
    }
  ],
  "total": 56
}
```

---

### GET `/teacher/pickup/records/:id`

> 接送记录详情

```json
// Response
{
  "id": 301,
  "student": {
    "id": 1, "name": "王小明", "avatar": "...",
    "class_name": "小一班"
  },
  "date": "2026-03-19",
  "type": 2,
  "type_label": "接走",
  "method": 1,
  "method_label": "扫码",
  "pickup_name": "爸爸（王明）",
  "pickup_relation": "father",
  "pickup_phone": "138****8888",
  "operator": {
    "id": 5,
    "nickname": "李师傅",
    "title": "门卫"
  },
  "is_abnormal": false,
  "abnormal_reason": "",
  "remark": "",
  "recorded_at": "2026-03-19T17:30:25Z"
}
```

---

### PUT `/teacher/pickup/records/:id/abnormal`

> 标记接送异常

```json
// Request
{
  "is_abnormal": true,
  "abnormal_reason": "超时未接走，已联系家长"
}

// Response
{
  "id": 301,
  "is_abnormal": true,
  "abnormal_reason": "超时未接走，已联系家长",
  "updated_at": "2026-03-19T18:30:00Z"
}
```
