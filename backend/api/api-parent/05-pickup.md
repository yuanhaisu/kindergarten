# 家长端 API — 接送

> 基础路径：`/api/v1/parent` | 鉴权：`Authorization: Bearer <token>`（角色 `parent`）
>
> Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。

---

### GET `/parent/pickup/qrcode`

> 获取接送二维码

```json
// Query: ?student_id=1

// Response
{
  "qr_code": "PICKUP_CODE_xxx_20260319_s1",
  "qr_image_url": "https://cdn.example.com/qr/pickup_s1.png",
  "student": { "id": 1, "name": "王小明" },
  "expire_at": "2026-03-19T23:59:59Z"
}
```

---

### GET `/parent/pickup/records`

> 接送记录

```json
// Query: ?student_id=1&page=1&page_size=20

// Response
{
  "list": [
    {
      "id": 301,
      "date": "2026-03-19",
      "type": 2,
      "type_label": "接走",
      "pickup_name": "爸爸（王明）",
      "pickup_relation": "father",
      "recorded_at": "2026-03-19T17:30:25Z",
      "is_abnormal": false
    },
    {
      "id": 300,
      "date": "2026-03-19",
      "type": 1,
      "type_label": "入园",
      "pickup_name": "妈妈（李丽）",
      "pickup_relation": "mother",
      "recorded_at": "2026-03-19T07:52:18Z",
      "is_abnormal": false
    }
  ],
  "total": 88,
  "page": 1,
  "page_size": 20
}
```

---

### GET `/parent/pickup/authorized-persons`

> 授权接送人列表

```json
// Query: ?student_id=1

// Response
{
  "list": [
    {
      "id": 1,
      "name": "爸爸（王明）",
      "relation": "father",
      "phone": "138****8888",
      "id_card_tail": "****1234",
      "avatar": "https://cdn.example.com/pickup/person/1.jpg",
      "auth_type": "permanent",
      "auth_type_label": "长期授权",
      "allow_pickup": true,
      "allow_view_record": true,
      "allow_view_status": false,
      "allow_growth_view": false,
      "notify_on_pickup": true,
      "status": 1,
      "status_label": "已审核"
    }
  ]
}
```

---

### POST `/parent/pickup/authorized-persons`

> 添加授权接送人

```json
// Request
{
  "student_id": 1,
  "name": "爷爷（王建国）",
  "relation": "grandfather",
  "phone": "13600005555",
  "sms_code": "123456",
  "id_card_number": "3101...",
  "avatar": "https://cdn.example.com/uploads/pickup/person_new.jpg",
  "auth_type": "permanent",
  "allow_pickup": true,
  "allow_view_record": true,
  "allow_view_status": false,
  "allow_growth_view": false,
  "notify_on_pickup": true
}

// Response
{
  "id": 4,
  "status": 0,
  "status_label": "待审核",
  "created_at": "2026-03-19T10:00:00Z"
}
```

---

### PUT `/parent/pickup/authorized-persons/:id`

> 编辑授权接送人

```json
// Request
{
  "name": "二姨（陈梅）",
  "phone": "137****3333",
  "avatar": "https://cdn.example.com/uploads/pickup/person_new.jpg"
}

// Response
{
  "id": 4,
  "updated_at": "2026-03-19T10:30:00Z"
}
```

---

### DELETE `/parent/pickup/authorized-persons/:id`

> 删除授权接送人

```json
// Response
{}
```
