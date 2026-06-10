# 20. 数据导出

> 对应前端页面：`admin/pages/export/index.vue`
>
> 基础路径：`/api/v1/admin`，需 `Authorization: Bearer <token>`，角色 `admin`。
>
> 状态：✅ 4 个接口已实现（下载走 `GET /admin/export/download/:task_id`，需 Bearer）。

> 💡 本模块提供"异步导出"方案（创建任务 → 轮询状态 → 下载）。另外在各业务模块也存在同步 Blob 下载接口，例如：
> - [`03-students.md`](./03-students.md) `GET /admin/students/export`
> - [`06-attendance.md`](./06-attendance.md) `GET /admin/attendance/report`

---

## POST `/admin/export/attendance`

> 出勤报表异步导出（对应前端 `exportAttendance`）

```json
// Request
{
  "class_id": null,             // null=全园
  "start_date": "2026-03-01",
  "end_date": "2026-03-31",
  "format": "xlsx"
}

// Response
{
  "task_id": "export_001",
  "status": "processing",
  "message": "报表生成中，完成后可下载"
}
```

---

## POST `/admin/export/students`

> 学生名册异步导出（对应前端 `exportStudentsTask`）

```json
// Request
{
  "class_id": null,
  "status": 1,
  "format": "xlsx"
}

// Response
{
  "task_id": "export_002",
  "status": "processing",
  "message": "报表生成中，完成后可下载"
}
```

---

## POST `/admin/export/growth`

> 成长记录异步导出（对应前端 `exportGrowthTask`）

```json
// Request
{
  "class_id": 1,
  "start_date": "2026-03-01",
  "end_date": "2026-03-31",
  "format": "xlsx"
}

// Response
{
  "task_id": "export_003",
  "status": "processing",
  "message": "报表生成中，完成后可下载"
}
```

---

## GET `/admin/export/:task_id`

> 查询导出任务状态 & 下载（对应前端 `getExportTask`）

```json
// Response（处理中）
{
  "task_id": "export_001",
  "status": "processing",
  "progress": 65,
  "created_at": "2026-03-19T10:00:00Z"
}

// Response（已完成）
{
  "task_id": "export_001",
  "status": "completed",
  "download_url": "https://cdn.example.com/exports/attendance_202603.xlsx",
  "file_name": "出勤报表_2026年3月.xlsx",
  "file_size": 128000,
  "expire_at": "2026-03-22T10:00:00Z",
  "created_at": "2026-03-19T10:00:00Z",
  "completed_at": "2026-03-19T10:01:30Z"
}
```
