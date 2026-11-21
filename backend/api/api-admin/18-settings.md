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
  },
  "health": {
    "temperature_required": false,
    "fever_threshold": 37.3
  },
  "album": {
    "classwide_enabled": false
  }
}
```

> `health.temperature_required` 默认 `false`（按需测温）；`health.fever_threshold` 默认 `37.3`，允许范围 `35.0–42.0`。
>
> `album.classwide_enabled` 是园所级「全班视图」开关，默认 `false`；它与家长端
> `GET /parent/class-albums?scope=all` 的放行判断同源（`kindergarten_app_settings.class_album_classwide_enabled`）。

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
  },
  "health": {
    "temperature_required": true,
    "fever_threshold": 37.5
  },
  "album": {
    "classwide_enabled": true
  }
}

// Response
{
  "updated_at": "2026-03-19T10:00:00Z"
}
```

> `health` / `album` 块为整块替换；`fever_threshold` 越界（<35.0 或 >42.0）返回 422，原策略不变。
> 更新 `album` 会写 `update_settings` 审计（detail 同时包含 `extra_config` 与 `album`）；
> 无 `system:kindergarten` 权限的管理员由现有 RBAC 拒绝，开关值不变。

---

## GET `/admin/settings/academic-years`

> 学年列表（对应前端 `getAcademicYears`）

```json
// Response
{
  "list": [
    { "id": 31846, "year": "2025-2026", "is_current": true },
    { "id": 31845, "year": "2024-2025", "is_current": false }
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
  "brand_color": "#F97316",
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
>
> 字段均为可选：不传表示不修改；传空串表示清空。
> `brand_color` 只接受预设色板或对比度足够的 `#RRGGBB`，非法值返回 422（`code=42200`）；
> 空串表示清空，家长端/老师端回落默认主题。保存后公开品牌接口立即返回新值。

```json
// Request
{
  "name": "阳光幼儿园",
  "address": "北京市朝阳区阳光路123号",
  "phone": "010-12345678",
  "principal": "张园长",
  "logo_url": "https://cdn.example.com/logo/1.png",
  "brand_color": "#F97316",
  "description": "阳光幼儿园创办于2010年..."
}

// Response
{
  "id": 1,
  "updated_at": "2026-03-19T10:00:00Z"
}
```

---

## GET `/admin/kindergarten/parent-qrcode`

> 家长端园所入口二维码（#176）：当前登录园所的对外物料。权限 `system:kindergarten`。
>
> 码属于园所，**不接受 `kindergarten_id` 参数**——目标园由 JWT 决定，避免越权为任意园生成入口。

```json
// Request（可选 query）
// env_version=release | trial | develop（不传回落到服务端配置的 code_env_version）

// Response
{
  "public_code": "k3m9x2qa7t",
  "scene": "kg_k3m9x2qa7t",
  "page": "pages/index/index",
  "env_version": "release",
  "image_base64": "iVBORw0KGgoAAAANSUhEUg..."
}
```

- `scene` 固定为 `kg_<public_code>`，长度上限 32（微信 scene 限制）；`public_code` 为 10 位 Crockford base32。
- 园所尚无 `public_code` 时首次生成自动分配并落库，重复生成复用同一码。
- 微信调用走**家长端**小程序凭据（`GetParentWXACodeUnlimit`）；`page` 必须真实存在于 `parent/src/pages.json`。
- 错误：园所不存在 `404`；`env_version` 非法 `422`；园所通道与家长端客户端作用域不匹配 `409`；家长端客户端未注册/停用 `503`。
