# 18. 系统设置 / 园所信息

> 对应前端页面：`admin/pages/settings/index.vue`
>
> 基础路径：`/api/v1/admin`，需 `Authorization: Bearer <token>`，角色 `admin`。
>
> 状态：✅ 6 个接口已实现（后端）。

本模块包含三部分：
1. 系统设置项（出勤时间、通知审核、积分规则等）
2. 学年管理
3. 园所基础信息

---

## GET `/admin/settings`

> 系统设置项（对应前端 `getSettings`）

```json
// Response
{
  "basic": {
    "academic_year": "2025-2026",
    "semester": "spring",
    "semester_start": "2026-02-20",
    "semester_end": "2026-07-10"
  },
  "attendance": {
    "arrive_start_time": "07:30",
    "arrive_end_time": "09:00",
    "pickup_start_time": "16:30",
    "late_threshold_minutes": 15
  },
  "notice": {
    "need_audit": true,
    "auto_remind_hours": 24
  },
  "growth": {
    "need_audit": false,
    "allow_parent_comment": true,
    "allow_parent_like": true
  },
  "pickup": {
    "qr_code_expire_hours": 24,
    "need_authorization_audit": true
  },
  "points": {
    "task_checkin_points": 5,
    "weekly_attendance_bonus": 10,
    "monthly_attendance_bonus": 50
  }
}
```

---

## PUT `/admin/settings`

> 更新系统设置（对应前端 `updateSettings`）

```json
// Request（可部分更新）
{
  "attendance": {
    "arrive_start_time": "07:30",
    "arrive_end_time": "09:00",
    "pickup_start_time": "17:00",
    "late_threshold_minutes": 20
  },
  "notice": {
    "need_audit": false
  }
}

// Response
{
  "updated_at": "2026-03-19T10:00:00Z"
}
```

---

## GET `/admin/settings/academic-years`

> 学年列表（对应前端 `getAcademicYears`）

```json
// Response
{
  "list": [
    { "year": "2025-2026", "is_current": true },
    { "year": "2024-2025", "is_current": false }
  ]
}
```

---

## POST `/admin/settings/academic-years`

> 创建新学年（对应前端 `createAcademicYear`）

```json
// Request
{
  "year": "2026-2027",
  "semester_spring_start": "2027-02-20",
  "semester_spring_end": "2027-07-10",
  "semester_fall_start": "2026-09-01",
  "semester_fall_end": "2027-01-20"
}

// Response
{
  "year": "2026-2027",
  "created_at": "2026-03-19T10:00:00Z"
}
```

---

## GET `/admin/kindergarten/info`

> 园所基础信息（对应前端 `getKindergartenInfo`）

```json
// Response
{
  "id": 1,
  "name": "阳光幼儿园",
  "address": "北京市朝阳区阳光路123号",
  "phone": "010-12345678",
  "principal": "张园长",
  "logo_url": "https://cdn.example.com/logo/1.png",
  "license_no": "教办字[2020]001号",
  "capacity": 350,
  "current_count": 300,
  "class_count": 10,
  "teacher_count": 35,
  "established_at": "2010-09-01",
  "description": "阳光幼儿园创办于2010年，是一所以培养孩子综合素质为目标的现代化幼儿园..."
}
```

---

## PUT `/admin/kindergarten/info`

> 更新园所信息（对应前端 `updateKindergartenInfo`）

```json
// Request
{
  "name": "阳光幼儿园",
  "address": "北京市朝阳区阳光路123号",
  "phone": "010-12345678",
  "principal": "张园长",
  "logo_url": "https://cdn.example.com/logo/1.png",
  "description": "阳光幼儿园创办于2010年..."
}

// Response
{
  "id": 1,
  "updated_at": "2026-03-19T10:00:00Z"
}
```
