# 模块 3：首页工作台 API

> 对应开发计划：模块 3：首页工作台
>
> 基础路径：`/api/v1/teacher`
>
> 认证：`Authorization: Bearer <token>`，用户角色为 `teacher`
>
> 以下 Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。

## 接口清单

- `GET /teacher/home/overview`
- `GET /teacher/home/todos`
- `PUT /teacher/home/todos/:id/action`

---

## 1. 首页

### GET `/teacher/home/overview`

> 首页概览数据（今日班级概况卡片）

```json
// Response
{
  "date": "2026-03-19",
  "weekday": "星期四",
  "class_info": {
    "id": 1,
    "name": "小一班",
    "title": "head",
    "title_label": "主班老师"
  },
  "attendance": {
    "present": 28,
    "leave": 2
  },
  "pending_approvals": 3,
  "unread_notices": 5
}
```

---

### GET `/teacher/home/todos`

> 待办事项列表

```json
// Query: ?page=1&page_size=10

// Response
{
  "list": [
    {
      "id": 1,
      "type": "leave_approval",
      "title": "王小明妈妈提交的请假申请",
      "description": "感冒发烧，请假3天（3月19日-3月21日）",
      "reference_id": 101,
      "actions": ["approve", "reject"],
      "priority": "high",
      "created_at": "2026-03-19T07:30:00Z"
    },
    {
      "id": 2,
      "type": "notice_unread_remind",
      "title": "\"春季出游活动\"通知未读提醒",
      "description": "还有5位家长未读，可一键提醒",
      "reference_id": 201,
      "actions": ["remind"],
      "priority": "normal",
      "created_at": "2026-03-19T09:00:00Z"
    },
    {
      "id": 3,
      "type": "growth_pending",
      "title": "今日成长记录待提交",
      "description": "还未上传今日班级活动照片和记录",
      "reference_id": null,
      "actions": ["go_record"],
      "priority": "normal",
      "created_at": "2026-03-19T10:00:00Z"
    }
  ],
  "total": 3
}
```

---

### PUT `/teacher/home/todos/:id/action`

> 处理待办事项（快捷操作）

```json
// Request
{
  "action": "approve",          // approve | reject | remind | go_record
  "remark": "注意休息"           // 可选，审批备注
}

// Response
{
  "id": 1,
  "status": "done",
  "message": "已同意请假申请"
}
```
