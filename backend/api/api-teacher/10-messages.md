# 模块 10：消息聚合列表 API

> 基础路径：`/api/v1/teacher`
>
> 认证：`Authorization: Bearer <token>`，用户角色为 `teacher`
>
> 以下 Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。

## 接口清单

- `GET /teacher/messages` — 老师端消息聚合列表

---

## GET `/teacher/messages`

> 聚合三类数据按时间倒序返回：通知发布记录 + 请假审批 + 系统消息。

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |

### 响应

```json
{
  "list": [
    {
      "type": "notice",
      "title": "周末作业通知",
      "content": "各位家长您好，本周作业如下...",
      "reference_id": 1,
      "extra": {
        "category": "homework",
        "category_label": "作业",
        "status": 2,
        "status_label": "已发布",
        "read_count": 25,
        "total_count": 30,
        "read_rate": "83%"
      },
      "created_at": "2026-03-19T09:00:00+08:00"
    },
    {
      "type": "leave",
      "title": "王小明妈妈提交的请假申请",
      "content": "王小明：身体不适（2026-03-20 至 2026-03-21）",
      "reference_id": 5,
      "extra": {
        "student_name": "王小明",
        "applicant_name": "王小明妈妈",
        "leave_dates": "2026-03-20 至 2026-03-21",
        "status": 0,
        "status_label": "待审批"
      },
      "created_at": "2026-03-19T10:30:00+08:00"
    },
    {
      "type": "system",
      "title": "系统维护通知",
      "content": "系统将于今晚22:00进行维护升级，预计持续2小时",
      "reference_id": 0,
      "extra": {},
      "created_at": "2026-03-18T20:00:00+08:00"
    }
  ],
  "total": 45,
  "page": 1,
  "page_size": 20
}
```

### 字段说明

**通用字段**

| 字段 | 类型 | 说明 |
|------|------|------|
| type | string | 消息类型：`notice` / `leave` / `system` |
| title | string | 消息标题 |
| content | string | 消息内容摘要（最多 200 字） |
| reference_id | uint64 | 关联业务记录 ID（notice → notices.id, leave → leave_requests.id） |
| extra | object | 按类型携带的附加数据 |
| created_at | string | 创建时间（RFC 3339） |

**notice 类型 extra**

| 字段 | 说明 |
|------|------|
| category | 通知分类：`homework` / `bring` / `activity` / `health` / `other` |
| category_label | 分类中文标签 |
| status | 通知状态：1=定时待发, 2=已发布 |
| status_label | 状态中文标签 |
| read_count | 已读家长数 |
| total_count | 班级学生总数 |
| read_rate | 已读率 |

**leave 类型 extra**

| 字段 | 说明 |
|------|------|
| student_name | 学生姓名 |
| applicant_name | 申请人姓名 |
| leave_dates | 请假日期范围 |
| status | 请假状态：0=待审批, 1=已同意, 2=已拒绝, 3=已取消 |
| status_label | 状态中文标签 |

**system 类型 extra**

空对象（当前无额外数据）。

### 数据来源

| 类型 | 来源表 | 筛选条件 |
|------|--------|----------|
| notice | notices | kindergarten_id + class_id + creator_id = 当前老师，status IN (1, 2) |
| leave | leave_requests | kindergarten_id + class_id = 当前班级 |
| system | messages | user_id = 当前老师，type = "system" |

三类数据统一按 `created_at DESC` 排序后分页。
