# 家长端 API — 亲子任务

> 基础路径：`/api/v1/parent` | 鉴权：`Authorization: Bearer <token>`（角色 `parent`）
>
> A 切片：班级/个体任务、提交、老师回应、无积分。B 切片已补：补充回应（`responses[].supplements`）与家长反馈（`reaction`）。公开墙、未读读标记仍待补。
>
> 所有任务接口必须带 `student_id` query；缺失或不属于当前家长时拒绝。

---

### GET `/parent/tasks`

> 当前孩子的任务列表

```json
// Query: ?student_id=27&page=1&page_size=20

// Response
{
  "list": [
    {
      "id": 1,
      "title": "每日阅读10分钟",
      "description": "每天陪孩子阅读10分钟绘本",
      "intro": "任务详细介绍",
      "learning_link": "延续本周班级的春天主题",
      "frequency": "daily",
      "target_student_id": 0,
      "allow_public": false,
      "recognition_ritual": "sticker",
      "star_scope": "task",
      "star_goal": 20,
      "start_date": "2026-03-19",
      "end_date": "2026-03-25",
      "status": 1,
      "checked_in_today": false,
      "submitted_current_period": false,
      "checkin_count": 2,
      "has_unread_supplement": false,
      "created_at": "2026-03-18T10:00:00Z"
    }
  ],
  "total": 5,
  "page": 1,
  "page_size": 20
}
```

- `target_student_id=0` 表示班级任务；`>0` 表示个体任务。
- `checkin_count` 是**当前孩子**的提交次数，不是全班累计。
- `submitted_current_period`：每日=今天、每周=滚动 7 天、一次性=周期内是否已提交；前端用它显示「已提交/待完成」。
- `points` 已移除；任务不发放积分。

---

### GET `/parent/tasks/:id`

> 任务详情

```json
// Query: ?student_id=27

// Response
{
  "id": 1,
  "title": "每日阅读10分钟",
  "description": "每天陪孩子阅读10分钟绘本",
  "intro": "任务详细介绍",
  "requirements": ["每天阅读10分钟", "上传一张阅读照片"],
  "learning_link": "延续本周班级的春天主题",
  "frequency": "daily",
  "target_student_id": 0,
  "allow_public": false,
  "recognition_ritual": "sticker",
  "star_scope": "task",
  "star_goal": 20,
  "start_date": "2026-03-19",
  "end_date": "2026-03-25",
  "status": 1,
  "checked_in_today": false,
  "has_unread_supplement": false,
  "responses": [
    {
      "id": 10,
      "ritual": "voice",
      "sticker_key": "",
      "content": "今天读得真认真",
      "media_url": "https://cdn.example.com/task/voice/10.mp3",
      "response_type": "single",
      "my_reaction": "",
      "supplements": [
        {
          "id": 12,
          "checkin_id": 1,
          "ritual": "sticker",
          "sticker_key": "flower",
          "content": "单独夸一句",
          "media_url": "",
          "my_reaction": "received",
          "created_at": "2026-03-19T20:40:00Z"
        }
      ],
      "created_at": "2026-03-19T20:30:00Z"
    }
  ],
  "created_at": "2026-03-18T10:00:00Z"
}
```

- `requirements` 是数组，不再返回 JSON 字符串。
- `responses` 返回当前孩子提交关联的**主回应**；`supplements` 是老师在主回应之后对**本孩子的这条提交**追加的个别补充，按创建时间升序，条数不限。
- 补充回应只包含当前孩子提交上的那些：一条周期汇总回应可能给多个孩子各补一条，跨孩子的补充不会出现在这里。
- `my_reaction`：当前家长账号对该条回应的反馈，未反馈时为空串；主回应与补充回应各自一个。
- `points`、`streak_days`、`my_completion_rate` 已移除。

---

### POST `/parent/tasks/:id/checkin`

> 提交互动记录

```json
// Query: ?student_id=27
// Request
{
  "content": "今天和爸爸一起读了《小王子》第三章",
  "media_file_id": 1001
}

// Response（HTTP 201）
{
  "id": 1,
  "message": "已提交，等待老师回应",
  "created_at": "2026-03-19T20:30:00Z"
}
```

- `media_file_id` 可选；通过 `/common/upload`（`biz_type=task`）上传后获得。
- 频率：每日任务当天一次；每周任务滚动 7 天内一次；一次性任务周期内一次。
- 未开始、已结束、跨班/非目标孩子、重复提交都会被拒绝。
- 不写积分流水、不返回 `points_earned`。

---

### GET `/parent/tasks/:id/checkin-history`

> 当前孩子在该任务下的提交历史

```json
// Query: ?student_id=27&page=1&page_size=20

// Response
{
  "list": [
    {
      "id": 1,
      "date": "2026-03-19",
      "content": "今天和爸爸一起读了《小王子》第三章",
      "media_url": "https://cdn.example.com/task/checkin/1.jpg",
      "response_id": 10,
      "student_name": "王小明",
      "created_at": "2026-03-19T20:30:00Z"
    }
  ],
  "total": 1
}
```

---

### POST `/parent/tasks/responses/:response_id/reaction`

> 家长对老师回应的反馈

```json
// Query: ?student_id=27
// Request
{ "reaction": "child_happy" }

// Response（HTTP 200）
{ "response_id": 10, "reaction": "child_happy" }
```

- `reaction`：`child_happy`（孩子很开心，孩子感知质量信号）/ `received`（收到了）/ `thanks`（谢谢老师）。其它取值 → 422。
- 按 `(回应, 家长)` 保存：同一家长再次提交是**更新**自己那一行（200），不会新增；同一回应可被多个家庭分别反馈，互不覆盖。
- 只允许对「与自己孩子提交相关」的回应反馈：主回应按提交的 `response_id` 判定，补充回应按 `task_response_supplements` 关系判定；**同一孩子的另一位家长也算**（有绑定关系即可，不要求是他本人提交的）。与该回应无任何关系的账号 → 403。
- `student_id` 必带（与其它家长端任务接口一致），但反馈记在**家长**维度，不受该参数限制——一条周期汇总回应可能同时覆盖同一家长两个孩子的提交；同一条回应允许父母双方各反馈一次。
- 补充回应的反馈单独存在，不进入主回应的 `feedback_coverage` / `child_happy_rate` / `ack_rate` 指标。
- 反馈是「给老师的确认」，不是评分或排名，契约不提供家长维度排序。
- 与任务删除并发时可能返回 **503**「操作冲突，请稍后重试」（行锁等待超时/死锁），可重试。

---

## 说明

- 公开申请/审核、未读读标记、班级墙在 B 切片补齐。
- 积分、排行榜、任务奖励在本功能中不实现。
