# 家长端 API — 亲子任务

> 基础路径：`/api/v1/parent` | 鉴权：`Authorization: Bearer <token>`（角色 `parent`）
>
> A 切片：班级/个体任务、提交、老师回应、无积分。公开墙、补充回应未读属于 B 切片。
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
      "created_at": "2026-03-19T20:30:00Z"
    }
  ],
  "created_at": "2026-03-18T10:00:00Z"
}
```

- `requirements` 是数组，不再返回 JSON 字符串。
- `responses` 只返回当前孩子提交关联的主回应；`my_reaction` 在 B 切片接入。
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

## 说明

- 公开申请/审核、补充回应、未读读标记、班级墙在 B 切片补充。
- 积分、排行榜、任务奖励在本功能中不实现。
