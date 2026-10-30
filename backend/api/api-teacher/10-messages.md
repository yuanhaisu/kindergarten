# 模块 10：消息聚合列表 API

> 基础路径：`/api/v1/teacher`
>
> 认证：`Authorization: Bearer <token>`，用户角色为 `teacher`
>
> 以下 Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。

## 接口清单

- `GET /teacher/messages` — 老师端消息聚合列表（含每条读状态与全量未读数）
- `PUT /teacher/messages/:id/read` — 单条消息标记已读
- `PUT /teacher/messages/read-all` — 全部消息标记已读

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
      "is_read": false,
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
      "is_read": true,
      "created_at": "2026-03-19T10:30:00+08:00"
    },
    {
      "type": "system",
      "title": "系统维护通知",
      "content": "系统将于今晚22:00进行维护升级，预计持续2小时",
      "reference_id": 88,
      "extra": {},
      "is_read": false,
      "created_at": "2026-03-18T20:00:00+08:00"
    }
  ],
  "unread_count": 2,
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
| reference_id | uint64 | 消息标识（notice → notices.id, leave → leave_requests.id, system → messages.id）。同时是已读接口路径里的 `:id` |
| extra | object | 按类型携带的附加数据 |
| is_read | bool | 当前老师是否已读。来自老师维度读记录，与通知已读率（家长读状态）无关 |
| created_at | string | 创建时间（RFC 3339） |

**响应顶层**

| 字段 | 类型 | 说明 |
|------|------|------|
| unread_count | int | 当前老师**全量**聚合消息中的未读条数（不是本页） |
| total | int | 当前老师可见的聚合消息总数 |
| page / page_size | int | 分页信息 |

> 系统消息没有业务对象，`reference_id` 取 `messages.id`：`messages.reference_id` 常为 0，
> 用它做标识会让多条系统消息共享同一条已读状态。

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

---

## PUT `/teacher/messages/:id/read`

> 标记单条聚合消息为已读（老师视角）。

读状态独立存储于 `teacher_message_reads`（`user_id + message_type + reference_id` 唯一），
**不写** `messages.is_read`，也**不写** `notice_read_logs`——通知已读率统计的是家长读状态，
不能被老师端的「点开即读」污染。

### 请求参数

| 参数 | 位置 | 类型 | 必填 | 说明 |
|------|------|------|------|------|
| id | path | uint64 | 是 | 消息标识：`type=notice` → notices.id；`type=leave` → leave_requests.id；`type=system` → messages.id |
| type | query | string | 是 | `notice` / `leave` / `system` |

### 响应

```json
{
  "id": 1,
  "type": "notice",
  "is_read": true
}
```

### 行为

- **幂等**：重复标记返回同样结果，不产生重复读记录。
- **可见性**：只能标记当前老师可见的消息（通知限本班且本人发布；请假限本班；系统消息限本人），
  不在可见范围或不存在 → 404。
- `type` 非法 → 422。

---

## PUT `/teacher/messages/read-all`

> 把当前老师可见的**全部**聚合消息（通知发布记录 + 本班请假审批 + 本人系统消息）记为已读。
> 作用域是全量，不是当前分类。

### 响应

```json
{
  "marked_count": 3
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| marked_count | int | 本次**新增**的已读条数；此前已读的不计，重复调用返回 0 |
