# 家长端 API — 成长时光

> 基础路径：`/api/v1/parent` | 鉴权：`Authorization: Bearer <token>`（角色 `parent`）
>
> Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。
>
> ⚠️ **2026-10 破坏性变更**：列表响应移除了 `likes_count` / `media_count` / `target_type` / `class_name` / `performance_tags`，`related_students` 移至详情；详情移除了 `media_count`。设计稿从未渲染这些字段，前端也未消费。字段溯源见 `docs/growth-feed-field-review.md`。
>
> ⚠️ **2026-10 增量变更**：列表与详情条目新增 `badge`（可空，获奖见证徽章）。纯增量，旧客户端忽略即可。设计依据见 `docs/achievement-badge-review.md`。
>
> ⚠️ **2026-10 增量变更**：列表条目新增 `scope`（`individual` / `class`），由原 `target_type` 派生，供家长端「只看孩子 / 看全部」筛选。纯增量，非破坏性；不是把 `target_type` 重新暴露给客户端。

---

### GET `/parent/growth/feed`

> 成长时光流（家长端时间线）。**列表只携带卡片实际渲染的内容**；仅详情页需要的信息（如 `related_students`）见 `GET /parent/growth/feed/:id`。

```json
// Query: ?student_id=1&year=2026&month=3&page=1&page_size=10

// Response
{
  "list": [
    {
      "id": 1,
      "scope": "class",
      "date": "2026-03-19",
      "content": "今天的美术课主题是《我心中的春天》，小朋友们都发挥了自己的想象力...",
      "media": [
        {
          "id": 1,
          "file_url": "https://cdn.example.com/growth/1.jpg",
          "thumb_url": "https://cdn.example.com/growth/1_thumb.jpg",
          "file_type": "image",
          "student_ids": [3, 5]
        },
        {
          "id": 2,
          "file_url": "https://cdn.example.com/growth/2.mp4",
          "thumb_url": "https://cdn.example.com/growth/2_thumb.jpg",
          "file_type": "video",
          "duration": 30,
          "student_ids": []
        }
      ],
      "comments_count": 12,
      "is_liked": true,
      "creator": {
        "id": 1,
        "nickname": "李老师",
        "avatar": "https://cdn.example.com/avatar/1.jpg"
      },
      "created_at": "2026-03-19T11:30:00Z"
    }
  ],
  "total": 236,
  "page": 1,
  "page_size": 10
}
```

**字段说明**

| 字段 | 说明 |
| --- | --- |
| `id` | 记录 ID，点卡片进详情用 |
| `scope` | 记录范围：`individual`（个体记录，原 `target_type=2`）/ `class`（全班记录，原 `target_type=1`）。由 `target_type` 派生，供家长端「只看孩子 / 看全部」筛选与归属标识；不是把内部 `target_type` 暴露给客户端 |
| `date` | 记录所属日期 `YYYY-MM-DD`，前端据此派生月份分组 |
| `content` | 正文 |
| `media` | 附件数组，无附件时为 `[]`（**不是 `null`**），前端按 `length` 判断是否渲染照片区 |
| `media[].student_ids` | 该照片上**已确认**点名的孩子 ID（恒为数组，无点名为 `[]`）。家长端「只看我的孩子」按 `includes(当前孩子)` 筛选；**未确认的 AI 建议不会出现在此字段**（信任模型：AI 建议、老师背书） |
| `comments_count` | 评论条数，0 时前端渲染「评论」 |
| `is_liked` | 当前家长是否已点赞 |
| `creator` | 发布者 `{id, nickname, avatar}` |
| `created_at` | 发布时刻 ISO8601 |
| `badge` | **可空**，仅当本条记录是某次获奖的见证时存在。形状 `{name, icon, color_kind}`，`color_kind` 取 `warning`/`info`（前端据此映射 design token 语义色，不落库） |

---

### GET `/parent/growth/feed/:id`

> 成长日志详情

```json
// Response
{
  "id": 1,
  "date": "2026-03-19",
  "target_type": 1,
  "breakfast_rating": 3,
  "lunch_rating": 2,
  "dinner_rating": 3,
  "nap_duration": 120,
  "sleep_quality": 3,
  "morning_mood": 3,
  "afternoon_mood": 2,
  "performance_tags": ["乐于助人", "积极发言", "遵守纪律"],
  "content": "今天的美术课主题是《我心中的春天》，小朋友们都发挥了自己的想象力...",
  "sync_to_feed": true,
  "media": [
    {
      "id": 1,
      "file_url": "https://cdn.example.com/growth/1.jpg",
      "thumb_url": "https://cdn.example.com/growth/1_thumb.jpg",
      "file_type": "image",
      "student_ids": [3]
    }
  ],
  "likes_count": 28,
  "comments_count": 12,
  "is_liked": true,
  "related_students": [
    { "id": 1, "name": "王小明", "avatar": "..." }
  ],
  "creator": {
    "id": 1,
    "nickname": "李老师",
    "avatar": "https://cdn.example.com/avatar/1.jpg"
  },
  "created_at": "2026-03-19T11:30:00Z"
}
```

**字段说明**

| 字段 | 说明 |
| --- | --- |
| `performance_tags` | **数组**（不是逗号串），库内为逗号分隔字符串，服务端解析后返回。无标签时为 `[]` |
| `related_students` | **仅发给指定学生的记录非空**；发给全班的记录不含此键。形状 `[{id, name, avatar}]` |
| `target_type` / `sync_to_feed` | 服务端实现已返回，此处如实记录。**尚无家长端详情设计稿**可判定是否渲染，见 `openspec/changes/align-parent-growth-feed/design.md` 的「证据缺口」 |
| `badge` | **可空**，仅当本条记录是当前孩子某次获奖的见证时存在。形状 `{name, icon, color_kind}` |
| `media[].student_ids` | 同列表语义（仅已确认点名）。**个体记录（`target_type=2`）不做「只看我的」筛选**——照片本就属于这些孩子，仅呈现标识 |
| ~~`media_count`~~ | **已移除**：可由 `media.length` 派生，且实现从未返回 |
| 其余 | 与列表同名字段语义一致 |

---

### POST `/parent/growth/feed/:id/like`

> 点赞成长日志

```json
// Request
{}

// Response
{
  "is_liked": true,
  "likes_count": 29
}
```

---

### DELETE `/parent/growth/feed/:id/like`

> 取消点赞

```json
// Response
{
  "is_liked": false,
  "likes_count": 28
}
```

---

### GET `/parent/growth/feed/:id/comments`

> 评论列表
>
> `replies` 只嵌一层：更深的回复归属到其最近的顶层评论，`parent_id` 仍指向直接父级。`user.relation` / `user.student_name` 仅在作者是本园家长时返回，教师/员工省略这两个字段。
>
> **分页作用于顶层**：`page` / `page_size` 约束 `list[]`（顶层，含被提升的孤立回复），回复始终随所属顶层一起返回，不单独分页。`root_total` 是顶层条数；`has_more` 表示当前页之后是否还有顶层；越界页返回空 `list` 且 `has_more=false`。`total` 为该记录下的评论总数（含回复），不随分页变化。

```json
// Query: ?page=1&page_size=20

// Response
{
  "list": [
    {
      "id": 1,
      "user": {
        "id": 10,
        "nickname": "王小明妈妈",
        "avatar": "https://cdn.example.com/avatar/10.jpg",
        "relation": "mother",
        "student_name": "王小明"
      },
      "content": "谢谢李老师的表扬和鼓励！小明回家特别开心 😊",
      "parent_id": null,
      "replies": [
        {
          "id": 5,
          "user": { "id": 1, "nickname": "李老师", "avatar": "..." },
          "content": "小明今天表现确实很棒！",
          "parent_id": 1,
          "created_at": "2026-03-19T13:10:00Z"
        }
      ],
      "created_at": "2026-03-19T13:05:00Z"
    }
  ],
  "total": 12,
  "root_total": 9,
  "has_more": false
}
```

---

### POST `/parent/growth/feed/:id/comments`

> 发表评论

```json
// Request
{
  "content": "谢谢李老师的表扬和鼓励！小明回家特别开心 😊",
  "parent_id": null            // 回复某条评论传 ID，一级评论传 null
}

// Response
{
  "id": 1,
  "created_at": "2026-03-19T13:05:00Z"
}
```

---

### GET `/parent/growth/album`

> 成长相册（按月查询、按**记录日期**分组的照片视频列表）
>
> 分组口径：`groups[].date` 取成长记录所属日期（`growth_daily_logs.date`，`YYYY-MM-DD`），**不是**媒体上传时刻；`year` / `month` 只是查询范围。`total` 为本次返回的媒体条数（与 `media` 元素总数一致）。

```json
// Query: ?student_id=1&year=2026&month=3

// Response
{
  "year": 2026,
  "month": 3,
  "groups": [
    {
      "date": "2026-03-19",
      "media": [
        {
          "id": 1,
          "file_url": "https://cdn.example.com/growth/1.jpg",
          "thumb_url": "https://cdn.example.com/growth/1_thumb.jpg",
          "file_type": "image",
          "log_id": 1
        }
      ]
    }
  ],
  "total": 28
}
```

---

### GET `/parent/growth/report`

> 成长报告（周报 / 月报 / 学期报告）。四格统计为**实时聚合**：即使周期的 `growth_reports` 尚无正文行，也照常返回，此时 `report` 缺省、`dimensions` 为 `[]`。
>
> 周期口径：`weekly` = `date` 所在周（周一 00:00 – 周日 23:59，本地时区）；`monthly` = `date` 所在自然月；`term` = `academic_years` 中覆盖 `date` 的春季/秋季学期区间。`date` 缺省为今天；`type` 只接受 `weekly` / `monthly` / `term`，非法值或 `term` 无学年配置返回参数错误，不静默回退。

```json
// Query: ?student_id=1&type=weekly&date=2026-03-17
// type: weekly / monthly / term（必填）
// date: YYYY-MM-DD，可落在周期内任意一天；缺省今天

// Response
{
  "student": { "id": 1, "name": "王小明", "avatar": "..." },
  "period": {
    "type": "weekly",
    "start_date": "2026-03-16",
    "end_date": "2026-03-22"
  },
  "attendance": {
    "present_days": 2,
    "leave_days": 1,
    "total_days": 3,
    "rate": "67%"
  },
  "points_earned": 45,
  "tasks_completed": 8,
  "tasks_total": 12,
  "growth_record_count": 15,
  "report": {
    "id": 1,
    "report_type": "weekly",
    "period_start": "2026-03-16T00:00:00+08:00",
    "period_end": "2026-03-22T00:00:00+08:00",
    "content": "...",
    "highlights": "...",
    "concerns": "...",
    "student": { "id": 1, "name": "王小明", "avatar": "..." }
  },
  "dimensions": [
    { "name": "语言", "score": 85, "max_score": 100, "comment": "表达能力强，词汇量丰富" }
  ]
}
```

**字段说明**

| 字段 | 说明 |
| --- | --- |
| `student` | 当前孩子 `{id, name, avatar}` |
| `period` | 实际统计周期 `{type, start_date, end_date}`，日期为 `YYYY-MM-DD` |
| `attendance.total_days` | 周期内**孩子所在班级**实际产生考勤记录的去重天数；老师未记考勤的日子不计入分母 |
| `attendance.present_days` | 孩子 status ∈ {1 出勤, 4 迟到} 的去重天数 |
| `attendance.leave_days` | 孩子 status = 3 请假的去重天数 |
| `attendance.rate` | `present_days / total_days` 的整数百分比字符串；分母为 0 时为 `"0%"` |
| `points_earned` | 周期内该孩子正向积分流水（`points > 0`）合计，扣减流水不计 |
| `tasks_total` | 周期内发布给该孩子的任务条数（班级任务 + 只发给该孩子的个体任务，按 `start_date` 落在周期内） |
| `tasks_completed` | 该孩子在上方任务集合中已打卡的去重任务条数 |
| `growth_record_count` | 周期内该孩子可见的成长记录条数（`growth_daily_logs` 条数，不是媒体数） |
| `report` | 报告正文行，保持既有结构；周期内无正文行时**不返回该键** |
| `dimensions` | 报告维度数组，保持既有结构；无正文行时为 `[]` |

> ⚠️ **2026-10 契约变更**：`growth_media_count` 更名 `growth_record_count`（该字段从未实现，语义是可见记录条数）；四格统计由「已声明未实现」变为真实返回；`date` 参数生效，`term` 增加取值校验。字段溯源见 `openspec/changes/growth-report-stats/design.md`。
