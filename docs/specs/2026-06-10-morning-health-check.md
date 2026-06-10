# 入园晨检与传染病防控

> **状态**: ✏️设计中
> **创建**: 2026-06-10
> **关联 Skill**: 无

## 背景与动机

幼儿园每天早晨由**园医（保健医生）**逐个检查入园孩子的体温、喉咙、手部、口腔、眼睛、皮肤。当前全是纸质流程，痛点：数据不可追溯、无影像留证、传染病响应滞后。

## 核心设计：入园与晨检分离

```
入园登记                      晨检
────────                    ──────
老师扫码/手动登记              园医检查孩子
      ↓                           ↓
生成 PickupRecord            更新 MorningHealthCheck
(type=1 送入)                status: pending → completed
      ↓                           ↓
自动生成待晨检记录              healthy / observe / reject
(status=pending)
      ↓
园医待检队列中出现该孩子
```

**规则**：有一个入园登记（PickupRecord type=1）→ 就自动生成一条待晨检记录（MorningHealthCheck status=pending）。入园和晨检是两个独立动作，通过 `pickup_record_id` 关联。

## 角色体系

新增 **园医（doctor）** 角色：

| 角色 | 职责 | 操作端 |
|------|------|--------|
| teacher | 扫码/手动登记孩子入园 → 生成 PickupRecord | 教师小程序 |
| **doctor** | 从待检队列逐一完成晨检 → 更新 MorningHealthCheck | 教师小程序·医生模块 |
| admin | 查看全园晨检统计 + 传染病预警 | 管理后台 Web |
| parent | 接收劝返通知 + 查看孩子晨检结果 | 家长小程序 |

## 数据模型

### morning_health_checks

```
morning_health_checks
├── kindergarten_id     uint64       多租户
├── class_id            uint64       冗余班级
├── student_id          uint64       学生 FK
├── pickup_record_id    uint64       关联入园记录 FK（NOT NULL）
├── date                DATE         检查日期
├── check_status        int8         0=待检 1=已完成 2=已跳过(未到园)
├── temperature         decimal(3,1) 体温 °C（待检时为 NULL）
├── throat_status       int8         0=正常 1=发红 2=红肿（待检时为 0）
├── hand_status         int8         0=正常 1=红疹 2=疱疹 3=其他
├── mouth_status        int8         0=正常 1=溃疡 2=疱疹 3=其他
├── eye_status          int8         0=正常 1=发红 2=分泌物
├── skin_status         int8         0=正常 1=红疹 2=水疱 3=其他
├── overall_status      int8         0=健康 1=需观察 2=劝返就医（待检时为 0）
├── is_isolated         bool         是否在隔离室等待家长
├── symptom_tags        varchar(300) 逗号分隔症状标签
├── remark              TEXT         园医备注
├── operator_id         uint64       园医 FK → users
└── SoftDeleteModel
```

**check_status 状态流转**：

```
pending (0) ──→ completed (1)  园医完成了检查
           ──→ skipped (2)     孩子今天没来（老师手动跳过）
```

### health_check_media（异常拍照留证）

```
health_check_media
├── check_id            uint64       关联晨检 FK
├── file_url            varchar(500) OSS URL
├── thumb_url           varchar(500) 缩略图
├── category            varchar(32)  拍摄部位：throat/hand/mouth/eye/skin
├── file_name           varchar(200)
├── file_size           int64
└── SoftDeleteModel
```

### 自动生成逻辑

在 TeacherPortalService 的 `CreatePickupRecord` 方法中，当 `type=PickupTypeDropOff (1)` 时，事务内追加：

```go
// 入园登记 → 自动生成待晨检
healthCheck := &domain.MorningHealthCheck{
    KindergartenID:  record.KindergartenID,
    ClassID:         record.ClassID,
    StudentID:       record.StudentID,
    PickupRecordID:  record.ID,
    Date:            record.Date,
    CheckStatus:     0, // pending
}
db.Create(healthCheck)
```

## 症状标签标准化

| 标签 | 含义 | 关联检查项 |
|------|------|-----------|
| fever | 发烧 (≥37.3°C) | 体温 |
| red_throat | 喉咙发红 | 喉咙 |
| swollen_throat | 喉咙红肿 | 喉咙 |
| hand_rash | 手部红疹 | 手部 |
| hand_blister | 手部疱疹 (手足口疑似) | 手部 |
| mouth_ulcer | 口腔溃疡 | 口腔 |
| mouth_blister | 口腔疱疹 (手足口疑似) | 口腔 |
| red_eye | 眼睛发红 | 眼睛 |
| eye_discharge | 眼睛分泌物 (结膜炎疑似) | 眼睛 |
| skin_rash | 皮肤红疹 | 皮肤 |
| skin_blister | 皮肤水疱 (水痘疑似) | 皮肤 |

园医勾选症状项 → 系统自动生成 symptom_tags，无需手动填写。

## API 设计

### 园医端（`/api/v1/doctor`）

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/doctor/health/queue?date=&class_id=` | 今日待检/已完成队列 |
| GET | `/doctor/health/check/:id` | 查看单条待检详情（含学生信息） |
| PUT | `/doctor/health/check/:id` | 提交晨检结果（更新 check_status → completed） |
| POST | `/doctor/health/check/:id/media` | 为晨检上传照片 |
| GET | `/doctor/health/today/stats` | 今日实时统计 |

### 老师端（`/api/v1/teacher`）

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/teacher/health/today` | 本班今日晨检结果摘要 |
| GET | `/teacher/health/student/:id` | 某学生近期晨检记录 |

### 管理后台（`/api/v1/admin`）

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/admin/health/stats?date=` | 全园统计 + 分班 + 症状排行 |
| GET | `/admin/health/stats/trend?days=7` | 劝返率趋势 |
| GET | `/admin/health/alerts` | 当前传染病预警 |
| GET | `/admin/health/records?date=&class_id=&status=&page=` | 记录列表 |
| GET | `/admin/health/records/:id` | 单条详情（含照片） |

## 前端交互

### 园医晨检页面（`teacher/src/pages/doctor/checkin.vue`）

园医登录 → 看到今日待检队列：

```
┌─────────────────────────────────┐
│  🏥 入园晨检  待检 82 / 已检 46  │
│                                  │
│  班级: [全部▾]                    │
│                                  │
│  ⏳ 待检查                        │
│  ┌─────────────────────────────┐ │
│  │ ○ 张三  中一班  7:32入园 等2分│ │  ← 等待时间
│  │ ○ 李四  中一班  7:33入园 等1分│ │
│  │ ○ 王五  中二班  7:35入园  刚刚│ │
│  └─────────────────────────────┘ │
│                                  │
│  ✅ 已完成（折叠）                 │
│  ┌─────────────────────────────┐ │
│  │ ● 赵六  中一班  ✅ 健康 7:31 │ │
│  │ ● 钱七  大一班  ⚠️ 观察 7:30 │ │
│  └─────────────────────────────┘ │
└─────────────────────────────────┘
```

点击待检学生 → 检查卡片（与之前设计相同，不重复）

### 关键流程：老师入园登记 → 医生立即看到

```
时间线：

7:32  老师扫码登记「张三」入园
      → POST /teacher/pickup/scan
      → 生成 PickupRecord(type=1)
      → 自动生成 MorningHealthCheck(status=pending, pickup_record_id=xxx)

7:32  园医手机上刷新队列 → 「张三」出现在待检列表，显示「7:32入园」

7:33  园医叫张三过来检查 → 点击张三 → 检查卡片
      → 体温 36.5 → 全部正常 → 点「✅ 健康」
      → PUT /doctor/health/check/xxx { check_status: 1, overall_status: 0 }
      → 张三从待检 → 已完成
```

## 传染病预警规则

| 预警 | 规则 | 动作 |
|------|------|------|
| 🔴 手足口疑似 | 同班 ≥2 人手/口/足疱疹 | 通知园长+班主任；建议消毒停课 |
| 🔴 聚集性发热 | 同班 ≥3 人发烧 | 通知园长+班主任 |
| 🟡 结膜炎聚集 | 同班 ≥3 人红眼/分泌物 | 通知园长 |
| 🟡 劝返率异常 | 某班劝返率 > 全园均值 2x | 推送园长 |

## 边界与异常

- **自动生成时机**：仅 PickupRecord type=1（送入）时生成待晨检；type=2（接走）不生成
- **修改限制**：当日可改，隔天只读
- **孩子没来**：老师可在本班页面将待检记录标记为「未到园」（check_status=2 skipped）
- **高峰期**：单次 PUT 更新一条晨检 < 50ms，无并发问题

## RBAC 变更

```csv
p, doctor, /api/v1/auth/*, *
p, doctor, /api/v1/common/*, *
p, doctor, /api/v1/doctor/*, *
p, doctor, /api/v1/classes/*, GET
p, doctor, /api/v1/children/*, GET
```

## 分阶段实施

| Phase | 内容 |
|---|---|
| Phase 1 | 数据模型 + 园医角色 + RBAC + 入园时自动生成待检 + 晨检页面 + 拍照 |
| Phase 2 | 症状推导标签 + 预警规则 + 通知推送 |
| Phase 3 | 趋势面板 + 症状热力图 + 数据导出 |

## 验收标准

- [ ] 老师登记入园 → 自动生成待晨检记录
- [ ] 园医队列中按入园时间排序显示待检学生
- [ ] 园医完成检查 → 状态 pending → completed
- [ ] 健康 → 仅更新 health_check；劝返 → 同时标记 pickup_record.is_abnormal
- [ ] 异常拍照 → 上传 OSS → 关联 health_check_media
- [ ] 管理后台可查看统计 + 分班明细 + 照片
- [ ] 后端 + 两前端编译通过
