# 老师端 API — 成就徽章

> 基础路径：`/api/v1/teacher` | 鉴权：`Authorization: Bearer <token>`（角色 `teacher`）
>
> Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。
>
> ⚠️ **2026-10 增量变更**：本文件为**新增**（此前老师端无成就相关端点）。纯增量，不影响既有接口。

---

### GET `/teacher/achievements`

> 授予入口的徽章选择器：只列**本园已启用**的徽章（停用的不出现在选择器里）。
>
> 不返回 `condition` / `target_value` / `progress` 等后台配置——那是 admin 与 P2 规则引擎的事。

```json
// Response
{
  "list": [
    { "id": 1, "name": "全勤小明星", "icon": "https://cdn.example.com/badges/attendance.png", "category": "attendance" },
    { "id": 2, "name": "小小艺术家", "icon": "https://cdn.example.com/badges/art.png", "category": "growth" }
  ],
  "total": 2
}
```

---

### POST `/teacher/achievements/:id/grant`

> 老师把徽章授予学生，可选绑定一条成长记录作为**获奖见证**。
>
> 设计依据：设计稿里那条「恭喜小明获得全勤小明星称号……」的记录，正文即获奖贺词、卡片挂同名徽章——
> 该记录就是这次授予的见证。

```json
// Request
{
  "student_ids": [21, 22],
  "log_id": 25            // 可选：获奖见证的成长记录 ID
}

// Response 201
{
  "achievement_id": 1,
  "granted_student_ids": [21, 22],
  "granted_count": 2,
  "log_id": 25
}
```

**字段说明**

| 字段 | 说明 |
| --- | --- |
| `student_ids` | **必填**，至少 1 个。只能是**自己班级**的学生 |
| `log_id` | **可选**。给出时把该成长记录标为获奖见证，家长在该记录卡片上看到徽章；不给时徽章只体现在已获数里，不出现在任何卡片上 |

**错误码**

| HTTP | code | 场景 |
| --- | --- | --- |
| 400 | 40010 | 徽章不存在或未启用 |
| 400 | 40011 | 获奖见证记录无效，或该学生对它不可见 |
| 400 | 40012 | 学生不存在或参数为空 |
| 403 | 40300 | 跨园所 / 非自己班级的学生 |
| 409 | 40910 | 该学生已获得过这枚徽章（重复授予） |
| 422 | 42200 | 参数校验失败（如 `student_ids` 为空） |

**校验顺序**（实现与之一致）：徽章启用 → 跨园所（徽章/学生/班级）→ 见证记录学生可见 → 重复授予。

**见证记录可见性规则**：记录必须同园，且对该学生可见——`target_type=1`（全班）即可见；`target_type=2`（个体）需在该记录的关联学生里。否则拒绝，避免「见证」成为假关联。
