# 家长端 API — 意见反馈

> 基础路径：`/api/v1/parent` | 鉴权：`Authorization: Bearer <token>`（角色 `parent`）
>
> Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。

---

### POST `/parent/feedback`

> 提交意见反馈

```json
// Request
{
  "category": "suggestion",          // suggestion=建议 / complaint=投诉 / bug=问题
  "content": "希望成长日志里能看到孩子的饮食情况评分",
  "contact": "13800001234",           // 可选，联系方式
  "images": [
    "https://cdn.example.com/uploads/feedback/1.jpg"
  ]
}

// Response
{
  "id": 1,
  "status": 0,
  "status_label": "已提交",
  "created_at": "2026-03-19T10:00:00Z"
}
```

---

### GET `/parent/feedback`

> 反馈列表

```json
// Query: ?page=1&page_size=20

// Response
{
  "list": [
    {
      "id": 1,
      "category": "suggestion",
      "category_label": "建议",
      "content": "希望成长日志里能看到孩子的饮食情况评分",
      "status": 1,
      "status_label": "已回复",
      "reply": "感谢您的建议！我们已在最新版本中加入了饮食评分功能，请更新查看。",
      "replied_at": "2026-03-20T10:00:00Z",
      "created_at": "2026-03-19T10:00:00Z"
    }
  ],
  "total": 3
}
```
