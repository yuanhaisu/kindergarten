# 家长端 API — 亲子任务

> 基础路径：`/api/v1/parent` | 鉴权：`Authorization: Bearer <token>`（角色 `parent`）
>
> A 切片：班级/个体任务、提交、老师回应、无积分。B 切片已补：补充回应（`responses[].supplements`）、家长反馈（`reaction`）、公开申请/取消与班级公开墙。未读读标记仍待补。
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
      "is_public_requested": false,
      "public_scope_confirmed": false,
      "public_approved_at": null,
      "public_revoked_at": null,
      "public_visible": false,
      "created_at": "2026-03-19T20:30:00Z"
    }
  ],
  "total": 1
}
```

- `public_*` 是该条提交当前的公开状态；`public_visible` 是**服务端算好的结果**（等价于公开墙会不会返回它），客户端不要自己拼条件。
- 只有本家庭能拿到这些字段：`checkin-history` 按 `student_id` 收窄。

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

### POST `/parent/tasks/checkins/:checkin_id/public-request`

> 家长申请把自己的这条提交公开到班级墙（**申请不等于公开**，必须老师审核通过）

```json
// Query: ?student_id=27

// Response（HTTP 200）
{
  "checkin_id": 1,
  "task_id": 1,
  "is_public_requested": true,
  "public_scope_confirmed": false,
  "public_approved_at": null,
  "public_approved_by": 0,
  "public_revoked_at": null,
  "public_status": "pending",
  "public_visible": false
}
```

- 允许条件：任务 `allow_public=true`、**不是**个体任务（`target_student_id=0`）、任务已开始未结束且 `status=1`、提交属于 `student_id` 这个孩子且未被删除，且该任务仍属于该孩子当前班级（转班后就不能再申请原班级任务的公开）。
- 不满足 → **422**；提交不属于当前孩子（跨家庭）→ **403**；提交或任务不存在 → **404**；行锁等待超时/死锁 → **503**（可重试）。
- 申请时**清空**旧批准与旧确认：`public_scope_confirmed=false`、`public_approved_at=null`、`public_approved_by=0`、`public_revoked_at=null`。所以「撤回/取消后再次申请」必然回到待审核，不能沿用旧批准。
- 已处于「已批准且未撤回」状态时重复调用是**幂等空操作**：不会把已公开的内容打回待审核（`public_status=approved`）。
- 待审核期间重复调用同样幂等，返回同一状态。

---

### DELETE `/parent/tasks/checkins/:checkin_id/public-request`

> 家长取消公开申请（也用于撤下已公开的内容）

```json
// Query: ?student_id=27

// Response（HTTP 200）
{
  "checkin_id": 1,
  "task_id": 1,
  "is_public_requested": false,
  "public_scope_confirmed": false,
  "public_approved_at": null,
  "public_approved_by": 0,
  "public_revoked_at": "2026-03-19T21:00:00Z",
  "public_status": "revoked",
  "public_visible": false
}
```

- 取消后 `public_scope_confirmed`、`public_approved_at`、`public_approved_by` 同时清空，并写 `public_revoked_at`；公开墙下一次读取即不可见（实时过滤，无缓存）。
- **撤回永远可用**：任务关掉公开开关、已结束、甚至孩子已经转班，家长仍必须能撤下自己的内容；这些条件只约束「申请」，不约束「取消」。
- 从未申请过时调用是**幂等空操作**：不改写 `public_revoked_at`（不会凭空产生一条「撤回」记录），`public_status` 返回 `none`。
- 跨家庭 → **403**；提交或任务不存在 → **404**；行锁等待超时/死锁 → **503**（可重试）。

> `public_status` 是**审核状态**：`none`（未申请）/ `pending`（已申请待审核）/ `approved`（已批准且未撤回）/ `revoked`（已取消或撤回）。它不等于「此刻在墙上」——任务结束、关闭公开、孩子转班或学生记录删除后仍可能是 `approved`，但 `public_visible=false`。是否真的会出现在公开墙，**只以 `public_visible` 为准**。

---

### GET `/parent/tasks/public-wall`

> 本班公开墙：只有**老师审核通过**的提交才会出现

```json
// Query: ?student_id=27&page=1&page_size=20

// Response
{
  "list": [
    {
      "checkin_id": 42,
      "task_id": 1,
      "task_title": "本周亲子共读",
      "student": { "id": 28, "name": "李小雨", "avatar": "..." },
      "date": "2026-03-20",
      "content": "今天读了《小熊很忙》",
      "media_url": "https://cdn.example.com/task/checkin/42.jpg",
      "created_at": "2026-03-20T19:30:00Z"
    }
  ],
  "total": 3,
  "page": 1,
  "page_size": 20
}
```

- 返回条件（逐条实时过滤，缺一不可）：`task.allow_public=true`、任务未删除、`task.status=1`、`task.end_date >= 今天`、`task.class_id = 当前孩子班级`、`is_public_requested`、`public_scope_confirmed`、`public_approved_at IS NOT NULL`、`public_revoked_at IS NULL`；另加一条安全补强：**提交者孩子当前所在班级必须等于任务班级**——孩子转班后原班级墙不再展示他的内容，管理员把任务改派到别的班时原班孩子的内容也不会漏到新班。
- 只返回**本班**内容，跨班不可见；任务结束后立即不展示。
- 每条只包含**提交者孩子**的内容与姓名，不返回其他孩子、家长联系方式或家庭信息；不做集体合照墙。
- 排序：`created_at DESC`（最新的在前）。
- `student_id` 必带（与其它家长端任务接口一致）：它决定「本班」是哪一班。

---

## 说明

- 公开申请/审核、班级墙见上方三个端点；未读读标记在 B 切片剩余部分补齐。
- 公开内容默认私密：家长申请、老师确认「内容中只有提交者孩子」后才可见；取消/撤回后立即消失。
- 积分、排行榜、任务奖励在本功能中不实现。
