---
name: system-enums
description: >
  Complete catalog of all hardcoded enum values, option lists, tag categories, 
  and constant mappings across the 养成星球 platform (teacher mini-program, admin web, 
  parent mini-program, and Go backend). Use this skill whenever adding or modifying 
  dropdown options, filter tabs, tag clouds, rating scales, status labels, category 
  selectors, or any UI element that presents a fixed set of choices. Also use when 
  migrating hardcoded enums to the sys_dict module, or when asked about "表现标签",
  "通知分类", "出勤状态", "日历事件类型", "请假类型", "教师职称", "年级", "学期",
  "家庭关系", "任务频率", "审核类型", "积分类型", "成长维度", "消息类型", "相册分类".
---

# System Enums Catalog (系统枚举目录)

## Purpose

Every dropdown, chip selector, tab bar, and option list in this project is currently hardcoded in frontend `.vue` files. Only `performance_tag` has been migrated to the dict module. This skill is the single source of truth for all 22 enum categories — where they live, what values they hold, and whether they've been migrated.

## Enum Index

| # | Category | Status | Dict Key |
|---|----------|--------|----------|
| 1 | 表现标签 | ✅ Dict | `performance_tag` |
| 2 | 成长记录分类 | ❌ Hardcoded | — |
| 3 | 饮食/睡眠/情绪评分 | ❌ Hardcoded | — |
| 4 | 相册分类 | ❌ Hardcoded | — |
| 5 | 出勤状态 | ❌ Hardcoded | — |
| 6 | 日历事件类型 | ✅ Dict | `calendar_event_type` |
| 7 | 消息类型 | ❌ Hardcoded | — |
| 8 | 通知分类 | ❌ Hardcoded | — |
| 9 | 请假状态/类型 | ❌ Hardcoded | — |
| 10 | 教师职称 | ❌ Hardcoded | — |
| 11 | 学生状态 | ❌ Hardcoded | — |
| 12 | 年级 | ❌ Hardcoded | — |
| 13 | 学期 | ❌ Hardcoded | — |
| 14 | 家庭关系 | ❌ Hardcoded | — |
| 15 | 任务频率 | ❌ Hardcoded | — |
| 16 | 审核类型/状态 | ❌ Hardcoded | — |
| 17 | 积分类型 | ❌ Hardcoded | — |
| 18 | 操作日志模块 | ❌ Hardcoded | — |
| 19 | 成长报告类型/维度 | ❌ Hardcoded | — |
| 20 | 家长任务标签 | ❌ Hardcoded | — |
| 21 | 家长通知标签 | ❌ Hardcoded | — |
| 22 | 日历硬编码班级 | ❌ Hardcoded | — |

---

## Detailed Enum Catalog

### 1. 表现标签 (Performance Tags) ✅ DICT

- **Dict key**: `performance_tag`
- **Items**: 乐于助人, 积极发言, 遵守纪律, 动手能力强, 专注力强, 有礼貌, 爱干净, 团结友善, 勇于尝试
- **Used in**: `teacher/src/pages/growth/create.vue` → tag cloud for growth records
- **Fetch**: `GET /teacher/dicts/performance_tag`
- **Backend storage**: `growth_daily_logs.performance_tags` — comma-separated string, no validation
- **Backend model**: `server/internal/domain/growth.go` — `PerformanceTags string`

### 2. 成长记录分类 (Growth Record Categories)

- **File**: `teacher/src/pages/growth/index.vue` lines 13-22
- **Values + Colors**:
  - 全部 → `bg: #EFF6FF, fg: #3B82F6, border: #BFDBFE`
  - 今日 → `bg: #DCFCE7, fg: #16A34A, border: #BBF7D0`
  - 学习 → `bg: #DBEAFE, fg: #2563EB, border: #BFDBFE`
  - 体能 → `bg: #FFF7ED, fg: #F97316, border: #FED7AA`
  - 艺术 → `bg: #F3E8FF, fg: #9333EA, border: #E9D5FF`
  - 其他 → `bg: #F1F5F9, fg: #64748B, border: #E2E8F0`
- **Also in parent app**: `parent/src/pages/growth/index.vue` line 13 — 全部, 校园生活, 活动照片, 作品展示, 获奖记录
- **Note**: Categories have per-item color metadata → use `extra` JSON field in dict items

### 3. 饮食/睡眠/情绪评分 (Meal/Sleep/Mood Ratings)

- **File**: `teacher/src/pages/growth/create.vue` lines 119-124
- **Meal ratings** (早餐/午餐/晚餐): `3` = 😊好, `2` = 😐一般, `1` = 😞差
- **Sleep duration**: `[1, 2, 3]` hours (rendered as chip selectors)
- **Sleep quality**: `3` = 很好, `2` = 一般, `1` = 不好
- **Mood** (上午/下午): `3` = 😊开心, `2` = 😐一般, `1` = 😞低落
- **Backend types** (`server/internal/domain/growth.go` lines 5-27):
  - `MealRating`: 3=Good, 2=Normal, 1=Poor
  - `SleepQuality`: 3=Good, 2=Normal, 1=Poor
  - `MoodRating`: 3=Happy, 2=Normal, 1=Sad
- **Note**: These are semantic enums with backend type definitions. Changes must sync frontend labels + backend constants.

### 4. 相册分类 (Album Categories)

- **File**: `teacher/src/pages/album/index.vue` lines 8-15
- **Values**: 全部, 活动, 日常, 作品, 节日
- **CSS class mapping**: 活动→`.tag-activity`(green), 日常→`.tag-daily`(blue), 作品→`.tag-artwork`(purple), 节日→`.tag-festival`(yellow)

### 5. 出勤状态 (Attendance Status)

- **File**: `teacher/src/pages/class/index.vue` lines 13-19
- **Values**: 全部, 出勤, 请假, 缺勤, 迟到
- **Status class map**: 出勤→green dot, 请假→blue dot, 缺勤→red dot, 迟到→orange dot

### 6. 日历事件类型 (Calendar Event Types) ✅ DICT（2026-10-17 起，align-class-calendar）

- **Dict key**: `calendar_event_type`（园所级；内置 5 类 `IsSystem=true` 不可删、可改名/停用；园所可扩展自定义类型；颜色存字典项 `Extra={"color":...}`）
- **内置项与色值**（seed 自 `design-data/teacher-06` 图例）:
  - `notice`(通知) → `#3B82F6`
  - `activity`(活动) → `#22C55E`
  - `holiday`(假期) → `#F97316`
  - `meeting`(家长会) → `#A855F7`
  - `checkup`(体检) → `#EF4444`
- **Backend SoT**: `server/internal/domain/calendar.go`（`EventType` 常量）+ `server/internal/app/seeder.go`（启动播种，幂等，含 kgID=0 模板）
- **Rendering**: 两端一律按字典解析——家长端由服务端解析下发 `event_type_label`/`color`；老师端类型选择器经 `GET /teacher/dicts/calendar_event_type`。旧前端硬编码映射（`eventDotColors` 等）随本变更删除
- **Importance levels**: `normal`(普通), `important`(重要), `urgent`(紧急) —— 仍为硬编码，产品未裁决（见 change 阻塞清单）
- **Hardcoded classes**: 中一班, 中二班, 大一班, 大二班, 小一班 (create.vue line 35) —— 待本变更端页面改造时修复

### 7. 消息类型 (Message Types)

- **File**: `teacher/src/pages/messages/index.vue` lines 8, 18-22
- **Tabs**: 全部, 通知, 审批, 系统
- **TYPE_CONFIG**:
  - `notice` → sender: 班级通知, iconBg: `#DBEAFE`, iconColor: `#3B82F6`
  - `leave` → sender: 审批提醒, iconBg: `#FEF9C3`, iconColor: `#CA8A04`
  - `system` → sender: 系统消息, iconBg: `#F1F5F9`, iconColor: `#64748B`

### 8. 通知分类 (Notice Categories)

- **Files**: `teacher/src/pages/notice/create.vue` (6-12), `admin/pages/notices/index.vue` (90-96)
- **Values**: `homework`(作业), `bring`(带物), `activity`(活动), `health`(健康), `other`(其他)
- **Backend** (`server/internal/domain/notice.go` lines 5-13): also includes `task` — `NoticeCategoryTask`
- **Note**: Backend has 6 categories but both frontends only expose 5. `task` is backend-only.

### 9. 请假状态与类型 (Leave Statuses & Types)

- **Files**: `parent/src/pages/leave/index.vue` (27), `create.vue` (13-17)
- **Statuses**: `pending`(待审批)=0, `approved`(已同意)=1, `rejected`(已拒绝)=2
- **Types**: 病假, 事假, 其他 (each with key + label + description)

### 10. 教师职称 (Teacher Titles)

- **File**: `admin/pages/teachers/index.vue` lines 219-223
- **Values**: `head`(班主任), `assistant`(配班老师), `nursery`(保育员)
- **Also**: `teacher/src/pages/profile/index.vue` line 32 — `principal`(园长)

### 11. 学生状态 (Student Status)

- **File**: `admin/pages/students/index.vue` lines 22-24
- **Values**: `1`(在读), `2`(毕业), `3`(退学)

### 12. 年级 (Grade Levels)

- **File**: `admin/pages/classes/index.vue` lines 11-13, 54-56
- **Values**: 小班, 中班, 大班

### 13. 学期 (Semester)

- **File**: `admin/pages/settings/index.vue` lines 10-11
- **Values**: `spring`(春季), `fall`(秋季)
- **Backend**: `server/internal/domain/kindergarten_app_setting.go` — `CurrentSemester string`

### 14. 家庭关系 (Family Relations)

- **File**: `admin/pages/parents/index.vue` lines 54-58
- **Values**: `father`(爸爸), `mother`(妈妈), `grandfather`(爷爷), `grandmother`(奶奶), `other`(其他)

### 15. 任务频率 (Task Frequency)

- **File**: `teacher/src/pages/tasks/index.vue` line 7
- **Values**: `once`(一次), `daily`(每天), `weekly`(每周)
- **Backend** (`server/internal/domain/task.go` lines 8-10): `TaskFrequency` type with constants

### 16. 审核类型与状态 (Audit Types & Statuses)

- **File**: `admin/pages/audit/index.vue` lines 7-14
- **Content types**: `notice`(通知), `growth`(成长), `comment`(评论)
- **Statuses**: `pending`(待审核), `passed`(已通过), `rejected`(已驳回)
- **Backend**: `server/internal/domain/audit.go` — `ContentAudit` model

### 17. 积分类型 (Point Types)

- **File**: `admin/pages/points/index.vue` lines 43-44
- **Values**: `earn`(增加), `spend`(扣减)
- **Backend**: `server/internal/domain/point.go` — `PointTransaction` model

### 18. 操作日志模块 (Operation Log Modules)

- **File**: `admin/pages/logs/index.vue` lines 7-10
- **Values**: `student`, `teacher`, `parent`, `system`

### 19. 成长报告类型与维度 (Growth Report Types & Dimensions)

- **File**: `parent/src/pages/growth/report.vue` lines 11-15
- **Report types**: `weekly`(周报), `monthly`(月报), `term`(学期)
- **7 Dimensions** (`server/internal/domain/growth.go` line 141):
  - 语言 (Language), 认知 (Cognition), 社交 (Social)
  - 运动 (Motor), 艺术 (Art), 自理 (Self-care), 情绪 (Emotion)
- Each dimension: `score`, `max_score`(default 100), `comment`

### 20. 家长任务标签 (Parent Task Tabs)

- **File**: `parent/src/pages/tasks/index.vue` line 12
- **Values**: 进行中, 已打卡, 全部

### 21. 家长通知标签 (Parent Notice Tabs)

- **File**: `parent/src/pages/notice/index.vue` lines 12-17
- **Values**: `""`(全部), `task`(任务), `activity`(活动), `other`(通知)

### 22. 日历硬编码班级

- **File**: `teacher/src/pages/calendar/create.vue` line 35
- **Values**: 中一班, 中二班, 大一班, 大二班, 小一班
- **Note**: This should come from the teacher's actual class assignment via `GET /teacher/class/list`, not be hardcoded. This is a data bug, not an enum issue.

---

## Dict Module Integration

### Architecture

```
sys_dict_types (字典类型)          sys_dict_items (字典项)
├── kindergarten_id                ├── type_id (FK)
├── type_key (unique per kg)       ├── item_key
├── type_name                      ├── item_label
├── is_system (built-in, can't     ├── extra (JSON: colors, icons)
│   delete)                        ├── sort_order
└── sort_order                     └── is_enabled
```

### API Reference

**Admin** (`/api/v1/admin/dicts`):
| Method | Path | Purpose |
|--------|------|---------|
| GET | `/types` | List types with item_count |
| POST | `/types` | Create type |
| PUT | `/types/:id` | Update type |
| DELETE | `/types/:id` | Delete (rejects if is_system or has items) |
| GET | `/types/:id/items` | List items |
| POST | `/types/:id/items` | Create item |
| PUT | `/items/:id` | Update item |
| DELETE | `/items/:id` | Delete item |
| PUT | `/items/sort` | Batch reorder |

**Teacher** (`/api/v1/teacher/dicts`):
| Method | Path | Purpose |
|--------|------|---------|
| GET | `/:type_key` | Get enabled items sorted |
| GET | `?types=a,b,c` | Batch fetch |

**Files**:
- Backend: `server/internal/domain/dict.go`, `repository/dict_repo.go`, `service/dict_service.go`, `handler/http/dict.go`
- Seeder: `server/internal/app/seeder.go` — `SeedDictsForAll(db)`
- Admin UI: `admin/pages/settings/dict.vue`, `admin/api/dict.ts`
- Teacher API: `teacher/src/api/dict.js` — `fetchDict(typeKey)`, `fetchDictsBatch(typeKeys)`

### Migration Pattern

To migrate enum #N from hardcoded to dict:

1. **Create dict type** via admin UI or seeder:
   ```go
   // In seeder.go SeedDicts()
   dictType := &domain.SysDictType{
       KindergartenID: kgID,
       TypeKey:        "your_type_key",
       TypeName:       "中文名称",
       IsSystem:       true,
   }
   ```

2. **Seed items** with existing hardcoded values

3. **Replace frontend hardcoded array**:
   ```javascript
   // Before
   const options = ["值1", "值2", "值3"]
   
   // After
   import { fetchDict } from "@/api/dict"
   const options = ref([])
   async function load() {
     try {
       const res = await fetchDict("your_type_key")
       options.value = (res.data?.items || []).map(i => i.item_label)
     } catch (e) {
       options.value = ["值1", "值2", "值3"] // fallback
     }
   }
   ```

4. **Mark as migrated** in this skill's index table

### Enums with Color Metadata

These enums need colors/icons per item → use the `extra` JSON field:
- #2 成长记录分类 — `{ bg, fg, border }` colors
- #4 相册分类 — CSS class mappings
- #5 出勤状态 — status color dots
- #6 日历事件类型 — event colors + icon letters
- #7 消息类型 — iconBg + iconColor

Example dict item with extra:
```json
{
  "item_key": "study",
  "item_label": "学习",
  "extra": { "bg": "#DBEAFE", "fg": "#2563EB", "border": "#BFDBFE" }
}
```

### Enums with Backend Type Definitions

These have Go type constants — frontend labels must stay in sync:
- #3 Meal/Sleep/Mood — `server/internal/domain/growth.go` (MealRating, SleepQuality, MoodRating)
- #8 Notice categories — `server/internal/domain/notice.go` (NoticeCategory*)
- #15 Task frequency — `server/internal/domain/task.go` (TaskFrequency*)
- #19 Growth dimensions — `server/internal/domain/growth.go` (DimName comment)
