# 家长端 API — 晨检记录

> 基础路径：`/api/v1/parent` | 鉴权：`Authorization: Bearer <token>`（角色 `parent`）
>
> Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。

---

### GET `/parent/health/records`

> 查看孩子最近 7 天已完成的晨检记录（按日期倒序）

```json
// Query: ?student_id=1

// Response
{
  "list": [
    {
      "check_date": "2026-10-18",
      "overall_status": 1,
      "overall_status_label": "需观察",
      "temperature": 37.6,
      "abnormal_items": [
        { "part": "throat", "part_label": "喉咙", "status_label": "发红" },
        { "part": "hand", "part_label": "手部", "status_label": "疱疹" }
      ],
      "remark": "喉咙略红，提醒多喝水",
      "is_isolated": false
    }
  ]
}
```

说明：

- 只能查看本人已绑定孩子的记录；`student_id` 未绑定返回 `403`，未传返回 `422`。
- 只返回已完成（`check_status = 1`）的记录；待检与已跳过不展示。
- `overall_status`：`0` 健康 / `1` 需观察 / `2` 劝返就医。
- `abnormal_items` 是服务端翻译后的可读项，包含 `3=其他` 这类没有统计标签的取值；
  客户端不需要也不应自行翻译内部代码。
- 仅最近 7 天（含当日）。更长的健康档案不在本期范围。
