# 保健医端 API — 入园晨检

> 基础路径：`/api/v1/doctor` | 鉴权：`Authorization: Bearer <token>`（角色 `doctor`）
>
> Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。

---

### GET `/doctor/health/queue`

> 当日晨检队列（待检 + 已完成）。队列由「送入登记」自动生成。

```json
// Query: ?date=2026-10-18&class_id=3
// class_id 可选；date 缺省时前端必须显式传值

// Response
{
  "list": [
    {
      "id": 101,
      "pickup_record_id": 5001,
      "student_id": 1,
      "student_name": "王小明",
      "class_name": "小一班",
      "class_id": 3,
      "check_status": 0,
      "overall_status": 0,
      "pickup_time": "07:32",
      "avatar": "https://cdn.example.com/avatar/s1.jpg"
    }
  ],
  "policy": {
    "temperature_required": false,
    "fever_threshold": 37.3
  }
}
```

`check_status`：`0` 待检 / `1` 已完成 / `2` 已跳过
`overall_status`：`0` 健康 / `1` 需观察 / `2` 劝返就医
`policy`：当前园所的体温策略。`temperature_required=true` 时提交必须带体温（`skip` 豁免）；`fever_threshold` 用于发烧判定与前端提示。

---

### GET `/doctor/health/check/:id`

> 单条晨检详情（含已关联的照片）

```json
// Response
{
  "id": 101,
  "pickup_record_id": 5001,
  "student_id": 1,
  "student_name": "王小明",
  "class_name": "小一班",
  "check_status": 1,
  "temperature": 37.6,
  "throat_status": 1,
  "hand_status": 0,
  "mouth_status": 0,
  "eye_status": 0,
  "skin_status": 0,
  "overall_status": 1,
  "is_isolated": false,
  "symptom_tags": "fever,red_throat",
  "remark": "喉咙略红，提醒多喝水",
  "media": [
    {
      "id": 11,
      "file_url": "https://cdn.example.com/health/throat.jpg",
      "thumb_url": "https://cdn.example.com/health/throat.jpg?x-oss-process=image/resize,w_200",
      "category": "throat"
    }
  ]
}
```

检查项取值：喉咙 `0–2`、手部 `0–3`、口腔 `0–3`、眼睛 `0–2`、皮肤 `0–3`（`0` 正常）。
`symptom_tags` 由服务端派生，客户端只读。

---

### PUT `/doctor/health/check/:id`

> 提交晨检结论。已完成的记录不可再次提交。

```json
// Request（未提交的字段保持原值）
{
  "temperature": 37.6,
  "throat_status": 1,
  "hand_status": 0,
  "mouth_status": 0,
  "eye_status": 0,
  "skin_status": 0,
  "overall_status": 1,
  "is_isolated": false,
  "remark": "喉咙略红，提醒多喝水",
  "skip": false
}
```

```json
// Response：详情字段同上，另加通知投递结果
{
  "id": 101,
  "...": "（同 GET 详情的字段）",
  "notified_count": 2,
  "notify_failed_count": 0
}
```

说明：

- `symptom_tags` **不再是请求字段**：服务端按体温与五项检查值派生；客户端传了也会被忽略。
- 检查项越界返回 `422`，晨检记录保持原状态。
- 园所要求全员测温时，未录体温返回 `422`（「本园要求全员测温，请先录入体温」）；`skip=true` 豁免。
- `overall_status` 为 `1`（需观察）或 `2`（劝返）时，系统向该生全部已绑定家长投递站内消息；
  同一记录对同一家长至多一条。投递失败不回滚结论，通过 `notify_failed_count` 暴露。
- `skip = true` 表示标记为未到园，不触发家长通知。

---

### POST `/doctor/health/check/:id/media`

> 为异常部位关联照片。文件必须先经统一上传通道上传。

```json
// 1) 先上传
// POST /api/v1/common/upload   (multipart: file, biz_type=health)
// Response: { "id": 9001, "file_url": "...", "thumb_url": "...", ... }

// 2) 再关联
// POST /api/v1/doctor/health/check/101/media
{
  "file_id": 9001,
  "category": "throat",
  "ref_desc": "咽部红肿"
}
```

```json
// Response
{
  "id": 11,
  "file_url": "https://cdn.example.com/health/throat.jpg",
  "thumb_url": "https://cdn.example.com/health/throat.jpg?x-oss-process=image/resize,w_200",
  "category": "throat"
}
```

- `category` 必须是 `throat` / `hand` / `mouth` / `eye` / `skin` 之一。
- `file_id` 必须是当前保健医上传、`biz_type=health` 的文件。
- 晨检记录必须属于当前园所，否则返回 `404`。

---

### GET `/doctor/health/today/stats`

> 当日晨检统计（分班 + 症状分布）

```json
// Query: ?date=2026-10-18

// Response
{
  "date": "2026-10-18",
  "by_class": [
    {
      "class_id": 3,
      "class_name": "小一班",
      "total": 25,
      "pending": 2,
      "completed": 23,
      "healthy": 20,
      "observe": 2,
      "reject": 1
    }
  ],
  "symptoms": [
    { "tag": "hand_blister", "count": 2 }
  ],
  "total_students": 120
}
```
