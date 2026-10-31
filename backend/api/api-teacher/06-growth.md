# 模块 6：成长记录 API

> 对应开发计划：模块 6：成长记录
>
> 基础路径：`/api/v1/teacher`
>
> 认证：`Authorization: Bearer <token>`，用户角色为 `teacher`
>
> 以下 Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。

## 接口清单

- `GET /teacher/growth/logs`
- `GET /teacher/growth/logs/:id`
- `POST /teacher/growth/logs`
- `PUT /teacher/growth/logs/:id`
- `DELETE /teacher/growth/logs/:id`
- `GET /teacher/growth/logs/:id/comments`
- `POST /teacher/growth/logs/:id/comments`
- `POST /teacher/growth/comments/:id/like`
- `DELETE /teacher/growth/comments/:id/like`
- `GET /teacher/growth/logs/:id/likes`
- `GET /teacher/growth/logs/:id/related-students`
- `GET /teacher/growth/logs/:id/tagging-status`
- `POST /teacher/growth/logs/:id/suggestions/confirm-high`
- `PUT /teacher/growth/media/:id/students/:sid`
- `POST /teacher/growth/media/:id/class-roster`
- `POST /teacher/growth/media/upload`

---

## 5. 成长记录

### GET `/teacher/growth/logs`

> 成长日志列表

```json
// Query: ?date=2026-03-19&page=1&page_size=20

// Response
{
  "list": [
    {
      "id": 1,
      "date": "2026-03-19",
      "content": "今天的美术课小朋友们都太棒了！...",
      "performance_tags": ["乐于助人", "积极发言"],
      "media_preview": [
        { "thumb_url": "https://cdn.example.com/thumb/1.jpg", "type": "image" },
        { "thumb_url": "https://cdn.example.com/thumb/2.jpg", "type": "image" }
      ],
      "likes_count": 28,
      "comments_count": 12,
      "related_students_count": 0,
      "pending_suggestions_count": 3,
      "sync_to_feed": true,
      "target_type": 1,
      "creator": {
        "id": 1,
        "nickname": "李老师",
        "avatar": "https://cdn.example.com/avatar/1.jpg"
      },
      "created_at": "2026-03-19T11:30:00Z"
    }
  ],
  "total": 236
}
```

> `media_preview` 只含卡片渲染所需的最小字段（仅缩略图），无附件的记录为 `[]`；
> `related_students_count` 仅对个体记录（`target_type=2`）有非零值。
> `pending_suggestions_count` 为该记录照片上待确认的 AI 建议数（已处理为 0）——
> 老师不必打开详情即可知道哪条记录有待处理建议（见 `confirm-high` 端点）。

---

### GET `/teacher/growth/logs/:id`

> 成长日志详情

```json
// Response
{
  "id": 1,
  "date": "2026-03-19",
  "breakfast_rating": 3,
  "lunch_rating": 2,
  "dinner_rating": 3,
  "nap_duration": 120,
  "sleep_quality": 3,
  "morning_mood": 3,
  "afternoon_mood": 2,
  "performance_tags": ["乐于助人", "积极发言", "遵守纪律"],
  "content": "今天的美术课主题是《我心中的春天》，小朋友们都发挥了自己的想象力...",
  "media": [
    {
      "id": 1,
      "file_url": "https://cdn.example.com/growth/1.jpg",
      "thumb_url": "https://cdn.example.com/growth/1_thumb.jpg",
      "file_type": "image",
      "file_name": "IMG_001.jpg",
      "file_size": 1024000,
      "duration": 0,
      "ref_desc": "小明在搭积木",
      "student_ids": [3, 5],
      "student_suggestions": [
        { "student_id": 7, "name": "王小明", "avatar": "https://cdn.example.com/avatar/7.jpg", "confidence": 91.5, "tier": "high" },
        { "student_id": 9, "name": "李小红", "avatar": "https://cdn.example.com/avatar/9.jpg", "confidence": 73.2, "tier": "low" }
      ]
    },
    {
      "id": 2,
      "file_url": "https://cdn.example.com/growth/2.jpg",
      "thumb_url": "https://cdn.example.com/growth/2_thumb.jpg",
      "file_type": "image",
      "file_name": "IMG_002.jpg",
      "file_size": 980000,
      "duration": 0,
      "ref_desc": "午餐光盘啦",
      "student_ids": [],
      "student_suggestions": []
    }
  ],
  "sync_to_feed": true,
  "target_type": 1,
  "related_students": [],
  "related_students_count": 0,
  "likes_count": 28,
  "comments_count": 12,
  "creator": {
    "id": 1,
    "nickname": "李老师",
    "avatar": "https://cdn.example.com/avatar/1.jpg"
  },
  "created_at": "2026-03-19T11:30:00Z"
}
```

> `related_students` 只对个体记录（`target_type=2`）有值，全班记录恒为 `[]`；
> `related_students_count` 恒等于 `related_students` 的长度。
> 视频媒体的 `duration` 为秒数，图片恒为 0。
>
> **照片点名（media 项）**：
> `student_ids` 为该照片**已确认**点名的学生（人工点名与老师确认过的识别结果；恒为数组，空为 `[]`）；
> `student_suggestions` 为**待老师处理的 AI 识别建议**（异步到达，见 `tagging-status`）：
> `tier=high`（`confidence ≥ suggest_high`，前端预勾）与 `tier=low`（`≥ suggest_low`，仅提示）。
> 建议须经确认端点转为点名后才生效，且**绝不出现在家长端出参**。

---

### POST `/teacher/growth/logs`

> 创建成长日志

```json
// Request
{
  "date": "2026-03-19",
  "breakfast_rating": 3,
  "lunch_rating": 2,
  "dinner_rating": 3,
  "nap_duration": 120,
  "sleep_quality": 3,
  "morning_mood": 3,
  "afternoon_mood": 2,
  "performance_tags": ["乐于助人", "积极发言", "遵守纪律"],
  "content": "今天的美术课小朋友们都太棒了！...",
  "media": [
    { "file_id": 7, "ref_desc": "小明在搭积木", "student_ids": [3] },
    { "file_id": 9, "ref_desc": "午餐光盘啦", "student_ids": [3, 5] }
  ],
  "sync_to_feed": true,
  "target_type": 1,
  "student_ids": []
}

// Response
{
  "id": 1,
  "created_at": "2026-03-19T11:30:00Z"
}
```

> `media` 语义（素材管理基线 L2，见 `docs/media-management-design.md`）：
> 元素只传引用——`file_id` 指向上传通道返回的素材（URL 等物理属性由服务端反查，客户端不得自拼 URL）；
> `ref_desc` 为引用层描述（这次为什么用），可空。
> 空数组 = 无附件（合法）；上限 20；`file_id` 必须存在且未被其他记录占用，否则整条请求被拒绝。
>
> **`media[].student_ids`（照片点名）**：提交即该照片的最终点名名单（人工点名直接生效）。
> 缺省（不传该字段）不触碰既有点名（兼容旧客户端）；传空数组 = 清除该照片点名；
> 点名对象必须是本记录范围内的学生（全班记录→本班在读；个体记录→该记录关联学生），越界整条请求被拒绝（422 参数校验失败）。

---

### PUT `/teacher/growth/logs/:id`

> 编辑成长日志

```json
// Request（同 POST，字段可选更新）
{
  "content": "今天的美术课小朋友们都太棒了！（已更新）",
  "performance_tags": ["乐于助人", "积极发言", "遵守纪律", "动手能力强"],
  "media": [
    { "id": 12, "file_id": 7, "ref_desc": "保留的既有图片（改描述）", "student_ids": [3, 5] },
    { "file_id": 9, "ref_desc": "本次新增", "student_ids": [] }
  ]
}

// Response
{
  "id": 1,
  "updated_at": "2026-03-19T12:00:00Z"
}
```

> `student_ids` 语义：`target_type=2` 时提交的目标学生名单**替换**该记录的既有学生关联（先删后插，重复提交同名单幂等）；`target_type=1`（全班）时清除既有学生关联。`target_type=2` 且 `student_ids` 为空时请求被拒绝。
> `media` 语义：**全量替换**——提交的 `media` 即最终附件集合。带 `id` 的为保留的既有媒体（必须属于本记录当前媒体，可改 `ref_desc`）；不带 `id` 的为本次新增（`file_id` 必须存在且未被其他记录占用）；未出现在清单中的既有媒体**解除绑定**（不删文件本体——基线 §4）。空数组 = 清空附件。
> `media[].student_ids` 同 POST 的点名语义；**提交的名单即最终名单**——该照片上未提交的点名（含尚未处理的 AI 建议）被移除。不传该字段（缺省）时保持该照片既有点名不变。
> 记录日期（`date`）在编辑时不可改——请求体不含该字段，提交后保持原值。

---

### DELETE `/teacher/growth/logs/:id`

> 删除成长日志

```json
// Response
{}
```

---

### GET `/teacher/growth/logs/:id/comments`

> 成长日志评论列表（顶层评论 + 一层 `replies`）

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
      "content": "谢谢李老师的表扬和鼓励！小明回家特别开心，说今天画画特别有意思，还要继续努力呢 😊",
      "parent_id": null,
      "likes_count": 3,
      "liked": false,
      "replies": [
        {
          "id": 5,
          "user": {
            "id": 1,
            "nickname": "李老师",
            "avatar": "https://cdn.example.com/avatar/1.jpg"
          },
          "content": "小明今天表现确实很棒，超级优秀，想象力很强！",
          "parent_id": 1,
          "likes_count": 0,
          "liked": true,
          "created_at": "2026-03-19T13:10:00Z"
        }
      ],
      "created_at": "2026-03-19T13:05:00Z"
    },
    {
      "id": 2,
      "user": {
        "id": 11,
        "nickname": "李小花妈妈",
        "avatar": "https://cdn.example.com/avatar/11.jpg",
        "relation": "mother",
        "student_name": "李小花"
      },
      "content": "看到小花的作品了，画的真好看！谢谢老师拍照记录～",
      "parent_id": null,
      "likes_count": 0,
      "liked": false,
      "replies": [],
      "created_at": "2026-03-19T13:12:00Z"
    }
  ],
  "total": 12,
  "root_total": 9,
  "has_more": false
}
```

> **嵌套语义（与家长端 `GET /parent/growth/feed/:id/comments` 同口径）**：
> `list[]` 只放顶层评论（`parent_id` 为空），其回复放 `replies[]`，只展开一层；
> 多级回复归到**最近的顶层祖先**下，父评论已被删除的孤立回复**提升为顶层**（不丢评论）。
> 排序：顶层与回复各自按 `created_at ASC, id ASC`。
>
> `user.relation` / `user.student_name` 只有**家长作者**才有（取与这条记录相关的那个孩子）：
> 个体记录优先记录点名到的孩子，全班记录优先同班孩子。老师/员工作者**不含这两个字段**。
>
> **分页作用于顶层**：`page` / `page_size` 约束 `list[]`（顶层，含被提升的孤立回复），
> 回复始终随所属顶层一起返回，不单独分页。`root_total` 是顶层条数（`all` 的长度，
> 不是当前页长度）；`has_more` 表示当前页之后是否还有顶层；越界页返回空 `list` 且
> `has_more=false`。
>
> **`total` 是该记录下的评论总数，含 `replies`**（不是 `list` 的长度，也不是 `root_total`）。
>
> `likes_count` 为该条评论的点赞数；`liked` 为**当前登录老师**是否已点赞该条评论。
> 顶层的 `replies[]` 与顶层评论同形（同样带 `likes_count` / `liked`）。

---

### POST `/teacher/growth/logs/:id/comments`

> 发表评论 / 回复评论

```json
// Request
{
  "content": "小明今天表现确实很棒，继续加油！",
  "parent_id": 1                // 回复某条评论，传 null 或不传为一级评论
}

// Response
{
  "id": 5,
  "created_at": "2026-03-19T13:10:00Z"
}
```

> `parent_id` 只接受**这条记录下的顶层评论**（其自身 `parent_id` 为空）——回复只有一层。
> 父评论不存在、不属于该记录、或本身已是回复时返回 400，且**不落库**。
> `content` 去空白后为空同样 400。

---

### POST `/teacher/growth/comments/:id/like`

> 点赞一条评论（幂等）

```json
// Request
{}

// Response
{
  "liked": true,
  "likes_count": 4
}
```

> **幂等**：重复点赞不报错、`likes_count` 不重复计数（`growth_comment_likes` 上
> `uk_comment_like_user(comment_id, user_id)` 唯一，重复写入被吞掉）。
> 评论所属记录必须在本教师班级内，否则 404。
> `likes_count` 为该评论的**最新**点赞总数（含其他用户），客户端直接以它渲染。

---

### DELETE `/teacher/growth/comments/:id/like`

> 取消点赞一条评论（幂等）

```json
// Response
{
  "liked": false,
  "likes_count": 3
}
```

> 未点过赞时调用同样成功（不报错），`likes_count` 不变。
> 取消赞是**删行**，不留软删记录（软删行会占住 `uk_comment_like_user` 唯一键，导致再次点赞撞键）。

---

### GET `/teacher/growth/logs/:id/likes`

> 点赞列表

```json
// Query: ?page=1&page_size=50

// Response
{
  "list": [
    {
      "user_id": 10,
      "nickname": "王小明妈妈",
      "avatar": "https://cdn.example.com/avatar/10.jpg",
      "created_at": "2026-03-19T12:00:00Z"
    },
    {
      "user_id": 11,
      "nickname": "李小花爸爸",
      "avatar": "https://cdn.example.com/avatar/11.jpg",
      "created_at": "2026-03-19T12:05:00Z"
    }
  ],
  "total": 28
}
```

---

### GET `/teacher/growth/logs/:id/related-students`

> 相关孩子列表

```json
// Response
{
  "list": [
    { "id": 1, "name": "王小明", "avatar": "https://cdn.example.com/avatar/s1.jpg" },
    { "id": 2, "name": "李小花", "avatar": "https://cdn.example.com/avatar/s2.jpg" },
    { "id": 3, "name": "张小强", "avatar": "https://cdn.example.com/avatar/s3.jpg" }
  ],
  "total": 28
}
```

---

### GET `/teacher/growth/logs/:id/tagging-status`

> 照片识别状态与建议（详情页轮询用；识别在发布后异步进行，不阻塞发布）

```json
// Response
{
  "pending": true,
  "media": [
    {
      "id": 1,
      "recognition_status": "done",
      "last_error": "",
      "face_count": 1,
      "is_group_photo": false,
      "suggestions": [
        { "student_id": 7, "name": "王小明", "avatar": "https://cdn.example.com/avatar/7.jpg", "confidence": 91.5, "tier": "high" }
      ]
    },
    {
      "id": 2,
      "recognition_status": "done",
      "last_error": "",
      "face_count": 8,
      "is_group_photo": true,
      "suggestions": []
    },
    {
      "id": 3,
      "recognition_status": "failed",
      "last_error": "识别失败: 取图 404",
      "face_count": 0,
      "is_group_photo": false,
      "suggestions": []
    }
  ]
}
```

> `recognition_status`：`none`（未入队，如该园未开启识别）/`pending`/`processing`/`done`/`failed`。
> `pending=true` 表示仍有任务未完成，客户端可间隔轮询（建议 3–5s）；`done` 后可停止。
> `suggestions` 与详情响应 `media[].student_suggestions` **同形**——轮询到新建议可直接就地更新详情页建议区。
> 识别未开启的园所（`[face].enabled=false`）全部为 `none`，客户端据此不展示识别相关 UI。
>
> `face_count` / `is_group_photo`（照片分型）：识别检测到的人脸数与**大合影判定**。
> 人脸数达到 `[face].group_photo_min_faces`（默认 6）即 `is_group_photo=true`——
> **全班记录**的大合影不产生逐人建议（逐人确认一张 29 人的合影注定不完整且是伪工作），
> 客户端据此呈现「大合影 · 一键全班」入口（见下方 `class-roster`）。
> ⚠️ 该入口**仅对全班记录（`target_type=1`）呈现**：个体记录不做全班（服务端会拒绝），
> 客户端须以记录的 `target_type` 抑制入口（`is_group_photo` 本身是分型事实，不区分记录类型）。
> ⚠️ 分型抑制**仅对全班记录生效**：个体记录里的大合影**照常产生建议**（其建议本就限定在该记录的目标孩子内）。
> 分型阈值由服务端判定，**客户端不得硬编码**。`face_count=0` 表示未识别/无法识别。

---

### POST `/teacher/growth/logs/:id/suggestions/confirm-high`

> 一键确认该记录的全部**高置信**建议（`confidence ≥ [face].suggest_high`）

```json
// Request
{}

// Response
{ "confirmed": 3 }
```

> 省力路径：一次活动的合影可产出数十条建议，逐条确认会把「AI 加速」变成负担。
> 仅 `status=0` 且高置信的建议转为生效（`status=1`）；**低置信建议保持待确认**（仍逐条处理）；
> 已确认/已拒绝的行不变化。`confirmed` 为本次转为生效的条数（可能为 0）。
> 记录必须属于本教师班级，否则 404。

---

### PUT `/teacher/growth/media/:id/students/:sid`

> 识别建议确认 / 拒绝（建议异步到达时老师不在编辑页，故有本轻量端点）

```json
// Request
{ "action": "confirm" }

// Response
{ "media_id": 1, "student_id": 7, "status": 1 }
```

> `action`：`confirm` → `status=1`（点名生效，家长端随即按此筛选）；`reject` → `status=2`（不再提示）。
> 仅允许对**建议态**（`status=0`）的行流转；重复提交同一动作幂等成功；
> 已确认/已拒绝的行调用返回 400（调整点名走编辑端点改 `media[].student_ids`）。
> `:id` 为媒体 ID（`media[].id`），必须属于本教师班级的某条已发布记录。

---

### POST `/teacher/growth/media/:id/class-roster`

> 大合影「一键全班」：把该照片所属记录**当日到场**的学生写为这张照片的确认点名

```json
// Request
{}

// Response
{ "present": 28, "named": [1, 2, 3, 5] }
```

> 名单口径与 `GET /teacher/class/present-students` 一致（在读 ∧ 未被明确排除；无数据保守全员），
> 日期取**记录自身的日期**（不是请求当天）。
> 并集语义：既有确认不受影响；**已拒绝的行不复活**（那是老师此前的明确表态）；
> 重复调用幂等。写入的点名与人工点名同样直接生效（家长端立即按此筛选）。
> `named` 为**操作完成后该照片的最终确认名单**（升序）——客户端须直接以它渲染，
> 不要自行做「本地并集」（本地算不出「已拒绝不复活」，也看不见点名与到场名单之间的时序差）。
> 仅对**全班记录**开放——个体记录（`target_type=2`）的照片本就属于那几位孩子，调用返回 400。
> `:id` 为媒体 ID（`media[].id`），必须属于本教师班级的某条已发布记录，否则 404。

---

### ~~POST `/teacher/growth/media/upload`~~（已废弃，勿使用）

> **本接口不再使用，勿在新代码中调用。**
>
> 废弃原因：其契约与实现形状不一致（契约曾声明 multipart `files[]`，实现是按 JSON 建行且无 `log_id` 挂载入参），
> 且全端零调用。上传统一走 `POST /common/upload`（multipart，`biz_type=growth`），
> 返回的 `id` 即成长记录 `media[].file_id`（见「创建成长日志」的 `media` 语义）。
>
> 接口路由暂保留（避免破坏未知调用方），下一轮清理时移除。
> ⚠️ 该废弃声明曾注明「AI 自动识别学生属独立变更、实现前不承诺」——**该独立变更已落地**
> （照片点名与人脸识别：`student_suggestions` + 确认端点 + `tagging-status`，见上）。
