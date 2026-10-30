# 家长端 API — 积分与成就

> 基础路径：`/api/v1/parent` | 鉴权：`Authorization: Bearer <token>`（角色 `parent`）
>
> Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。

---

### GET `/parent/points/overview`

> 积分总览

```json
// Query: ?student_id=1

// Response
{
  "student": { "id": 1, "name": "王小明", "avatar": "..." },
  "total_points": 1245,
  "level": 5,
  "level_name": "超级小星星",
  "level_icon": "https://cdn.example.com/levels/5.png",
  "next_level_points": 1500,
  "rank_in_class": 3,
  "class_total": 30,
  "this_week": 45,
  "this_month": 180
}
```

**字段口径（2026-10 B16 补齐）**

| 字段 | 口径 |
| --- | --- |
| `rank_in_class` | 本班在读学生中按「本周获得积分」降序的名次，同分并列取最小名次；无班级或不在读时为 0 |
| `class_total` | 本班在读学生数（`students.status = 1`） |
| `this_week` | 本周（周一 00:00 起，园所时区）正向积分合计 |
| `this_month` | 本月正向积分合计（B16 前恒 0，现为真实值） |
| `next_level_points` | **暂未启用**：仓库无等级模型与阈值来源，固定返回 0；前端据 0 隐藏等级进度条 |
| `level_icon` | **暂未启用**：固定空串 |
| `level` / `level_name` | 历史遗留的总分粗分 1–3 档，不是等级表 |

---

### GET `/parent/points/history`

> 积分流水

```json
// Query: ?student_id=1&type=earn&page=1&page_size=20
// type: earn=获取 / spend=消费 / all=全部

// Response
{
  "list": [
    {
      "id": 1,
      "type": "earn",
      "type_label": "获得",
      "source": "task_checkin",
      "source_label": "任务打卡",
      "amount": 5,
      "balance": 1245,
      "description": "完成任务【每日阅读10分钟】",
      "reference_id": 1,
      "created_at": "2026-03-19T20:30:00Z"
    }
  ],
  "total": 128,
  "page": 1,
  "page_size": 20
}
```

**字段口径（2026-10 B16 补齐）**

- `type` 由 `amount` 符号推导：`earn` / `spend`；`type_label` 为 获得 / 消耗。
- `amount` 带符号（获得为正、消耗为负），`balance` 为该笔发生后的余额。
- `source_label` 复用管理端同一张映射（任务打卡 / 管理员调整 / 周全勤 / 月全勤 / 成长日志被推荐），未知 `source` 原样返回。
- `type` 查询参数支持 `earn|spend|all`，缺省或未知值按全部处理；分页 `total` 为过滤后的总数。

---

### GET `/parent/points/leaderboard`

> 积分排行榜

```json
// Query: ?student_id=1&period=weekly
// period: weekly / monthly / all

// Response
{
  "period": "weekly",
  "my_rank": 3,
  "my_points": 45,
  "class_total": 28,
  "list": [
    {
      "rank": 1,
      "student": { "id": 2, "name": "李小花", "avatar": "..." },
      "points": 60,
      "is_self": false
    },
    {
      "rank": 2,
      "student": { "id": 3, "name": "张小强", "avatar": "..." },
      "points": 50,
      "is_self": false
    },
    {
      "rank": 3,
      "student": { "id": 1, "name": "王小明", "avatar": "..." },
      "points": 45,
      "is_self": true
    }
  ]
}
```

**字段口径（2026-10 B16 补齐）**

- `period` 固定 `weekly`（契约与设计均无月榜/总榜）。
- 按本周获得积分降序、同分按学生 id 升序；并列取最小名次（1,2,2,4）。
- 仅返回前 20 名；`my_rank` / `my_points` 按全班全量计算，自己排在 20 名外也有真实名次。
- `class_total` 为本班在读学生数；零积分学生也在榜内。
- `student` 为嵌套对象（id/name/avatar），`is_self` 标记当前家长所选孩子。

---

### GET `/parent/achievements`

> 成就勋章列表
>
> ⚠️ **2026-10 增量变更**：`earned[]` 新增 `log_id`（可空——未绑定见证的授予不含此键）。
> 有 `log_id` 表示该徽章有一条成长记录作为获奖见证，家长会在那条记录的卡片上看到徽章。

```json
// Query: ?student_id=1

// Response
{
  "earned": [
    {
      "id": 1,
      "name": "全勤之星",
      "description": "连续一个月全勤",
      "icon_url": "https://cdn.example.com/badges/attendance.png",
      "category": "attendance",
      "points": 10,
      "earned_at": "2026-03-01T08:00:00Z",
      "log_id": 25
    },
    {
      "id": 2,
      "name": "小小艺术家",
      "icon_url": "https://cdn.example.com/badges/art.png",
      "category": "growth",
      "earned_at": "2026-03-05T09:00:00Z"
    }
  ],
  "not_earned": [
    {
      "id": 3,
      "name": "艺术小天才",
      "description": "参与10次美术/手工活动",
      "icon_url": "https://cdn.example.com/badges/art_locked.png",
      "category": "growth",
      "progress": {
        "current": 6,
        "target": 10,
        "rate": "60%"
      }
    }
  ],
  "earned_count": 8,
  "total_count": 20
}
```

**字段说明**

| 字段 | 说明 |
| --- | --- |
| `earned[].log_id` | **可空**。获奖见证的成长记录 ID；未绑定见证的授予不含此键 |
| `earned[].earned_at` | 获得时刻 |
| `not_earned[].progress` | **未实现**——属 P2 规则引擎（`Condition`/`TargetValue` 进度制），本次不实现 |

> ✅ **已对齐（2026-10-06）**：实现返回 `description`（此前缺失，家长端 `points/index.vue` 渲染它但拿到空），
> 字段名统一为 `icon_url`（与 admin 侧一致），并补记 `points`。
