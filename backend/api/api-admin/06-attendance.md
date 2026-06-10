# 06. 出勤管理

> 对应前端页面：`admin/pages/attendance/index.vue`
>
> 基础路径：`/api/v1/admin`，需 `Authorization: Bearer <token>`，角色 `admin`。
>
> 状态：✅ 后端已实现（2026-04-21）。

---

## GET `/admin/attendance/overview`

> 全园出勤概览（按日期、按班级；对应前端 `getAttendanceOverview`）

```json
// Query: ?date=2026-03-19

// Response
{
  "date": "2026-03-19",
  "total_students": 300,
  "present": 280,
  "leave": 12,
  "absent": 8,
  "attendance_rate": "93.3%",
  "by_class": [
    {
      "class_id": 1,
      "class_name": "小一班",
      "total": 30,
      "present": 28,
      "leave": 2,
      "absent": 0,
      "rate": "93.3%"
    },
    {
      "class_id": 2,
      "class_name": "小二班",
      "total": 30,
      "present": 29,
      "leave": 1,
      "absent": 0,
      "rate": "96.7%"
    }
  ]
}
```

---

## GET `/admin/attendance/records`

> 出勤记录明细（对应前端 `getAttendanceRecords`）

```json
// Query: ?class_id=1&date=2026-03-19&status=leave&page=1&page_size=20
// status: present / leave / absent

// Response
{
  "list": [
    {
      "id": 1,
      "student": { "id": 1, "name": "王小明", "avatar": "...", "class_name": "小一班" },
      "date": "2026-03-19",
      "status": "present",
      "status_label": "出勤",
      "arrive_time": "07:52",
      "pickup_time": "17:30",
      "remark": ""
    }
  ],
  "total": 300,
  "page": 1,
  "page_size": 20
}
```

---

## GET `/admin/attendance/report`

> 出勤报表导出（Blob；对应前端 `exportAttendanceReport`）

```json
// Query: ?class_id=&start_date=2026-03-01&end_date=2026-03-31&format=xlsx
// 返回文件流
```

> 💡 另见 [`20-export.md`](./20-export.md) 的 `POST /admin/export/attendance` 异步导出方式。
