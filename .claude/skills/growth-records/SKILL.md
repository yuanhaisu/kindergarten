---
name: growth-records
description: >
  Business context and implementation guide for the 成长记录 (Growth Records) feature module 
  of the 养成星球 kindergarten platform. Use this skill whenever working on growth-related 
  code — adding APIs, modifying the growth data model, fixing growth bugs, reviewing PRs 
  that touch growth files, or planning growth module iterations. Triggers on mentions of: 
  成长记录, growth records, 成长日记, growth daily log, 成长报告, growth report, 成长相册, 
  growth album, 成长时光, growth feed, GrowthDailyLog, GrowthMedia, GrowthComment, GrowthLike, 
  GrowthReport, growth stats, growth dashboard, or changes to files under 
  server/internal/domain/growth*, server/internal/service/growth*, 
  server/internal/repository/growth*, admin/pages/growth/, teacher/src/pages/growth/.
---

# Growth Records (成长记录) Module

## Module Purpose

The growth records module is the heart of the kindergarten's daily observation system. Teachers log children's daily meals, sleep, mood, and performance, upload photos/videos, and share these moments with parents. Parents engage via likes and comments. Periodically, the system generates comprehensive growth reports with radar-chart assessments across seven developmental dimensions.

**Three-portal architecture**: Teacher mini-program (creator), Admin web (overseer/auditor), Parent mini-program (consumer).

---

## Data Models

### Primary Tables

**`growth_daily_logs`** — Core entity. One record per class per day (or multiple).

| Field | Type | Notes |
|---|---|---|
| `kindergarten_id` | uint | Multi-tenant scope |
| `class_id` | uint | Owner class |
| `creator_id` | uint | Teacher who wrote it |
| `date` | DATE | The day being recorded |
| `breakfast_rating` / `lunch_rating` / `dinner_rating` | int | 3=Good 😊, 2=Normal 😐, 1=Poor 😞, 0=Not rated |
| `nap_duration` | int | Minutes |
| `sleep_quality` | int | 3/2/1/0 (same scale) |
| `morning_mood` / `afternoon_mood` | int | 3=Happy, 2=Normal, 1=Sad, 0=Not rated |
| `performance_tags` | string | Comma-separated, e.g. `"乐于助人,积极发言"`. 9 preset tags available |
| `content` | text | Free-text narrative |
| `sync_to_feed` | bool | Whether visible in parent feed |
| `target_type` | int | `1` = entire class; `2` = specific students only |
| SoftDeleteModel | embedded | `deleted_at`, `created_at`, `updated_at` |

**`growth_log_students`** — Join table for `target_type=2` records.

| Field | Type |
|---|---|
| `log_id` | uint |
| `student_id` | uint |

**`growth_media`** — Photos/videos attached to a log.

| Field | Type | Notes |
|---|---|---|
| `log_id` | uint (nullable) | NULL = unattached media (upload-before-attach pattern) |
| `file_url` | string | Full OSS URL |
| `thumb_url` | string | Thumbnail URL |
| `file_type` | string | `"image"` or `"video"` |
| `file_name` / `file_size` | string / int64 | |
| `duration` | int | Video duration in seconds |
| `sort_num` | int | Display order |

**`growth_media_students`** — AI-tagged or manually tagged students in photos.

| Field | Type |
|---|---|
| `media_id` | uint |
| `student_id` | uint |
| `is_auto` | bool | AI recognition flag (placeholder) |

**`growth_comments`** — Parent comments, threaded via `parent_id`.

| Field | Type | Notes |
|---|---|---|
| `log_id` | uint | |
| `user_id` | uint | Commenter |
| `content` | string | Max 500 chars |
| `parent_id` | uint (nullable) | For threaded replies |

**`growth_likes`** — One like per user per log.

| Field | Type |
|---|---|
| `log_id` | uint |
| `user_id` | uint |

Unique constraint on `(log_id, user_id)`.

**`growth_reports`** — AI-generated periodic reports per student.

| Field | Type | Notes |
|---|---|---|
| `student_id` | uint | |
| `report_type` | string | `"weekly"` or `"monthly"` |
| `period_start` / `period_end` | DATE | |
| `content` | text | Full report text |
| `highlights` | text | |
| `concerns` | text | |
| `status` | int | `1`=generated, `0`=generating, `-1`=failed |

**`growth_report_dimensions`** — Radar chart scores.

| Field | Type |
|---|---|
| `report_id` | uint |
| `dim_name` | string | 语言/认知/社交/运动/艺术/自理/情绪 |
| `score` | float | |
| `max_score` | float | Default 100 |
| `comment` | text | |

### Shared: `content_audits`

Growth logs use the shared audit table with `target_type = "growth"`:

| Field | Notes |
|---|---|
| `target_type` | `"growth"` |
| `target_id` | Growth log ID |
| `status` | `0`=pending, `1`=approved, `2`=rejected |
| `submitter_id` / `reviewer_id` | |
| `reject_reason` | |

---

## API Endpoints

### Admin (`/api/v1/admin`)

| Method | Path | Purpose |
|---|---|---|
| GET | `/growth/stats` | Period stats (totals, by-class, daily trend) |
| GET | `/growth/logs` | Paginated list with class/teacher/date/audit filters |
| GET | `/growth/logs/:id` | Detail with media, audit info, counts |
| DELETE | `/growth/logs/:id` | Soft-delete with operation log |
| PUT | `/growth/logs/:id/audit` | Approve/reject (body: `{status, remark}`) |
| POST | `/export/growth` | Async XLSX export (columns: date, class, content, teacher, created_at) |

### Teacher (`/api/v1/teacher`)

| Method | Path | Purpose |
|---|---|---|
| GET | `/growth/logs` | List teacher's own class logs |
| POST | `/growth/logs` | Create log with meal/sleep/mood/tags/media/students |
| GET | `/growth/logs/:id` | Detail |
| PUT | `/growth/logs/:id` | Update |
| DELETE | `/growth/logs/:id` | Soft-delete |
| GET | `/growth/logs/:id/comments` | List comments |
| POST | `/growth/logs/:id/comments` | Add comment |
| GET | `/growth/logs/:id/likes` | Like count |
| POST | `/growth/logs/:id/like` | Like (**STUB** — returns hardcoded) |
| DELETE | `/growth/logs/:id/like` | Unlike (**STUB**) |
| GET | `/growth/logs/:id/related-students` | Linked students for target_type=2 |
| POST | `/growth/media/upload` | Upload media (file → OSS presigned URL → create media record) |

### Parent (`/api/v1/parent`)

| Method | Path | Status |
|---|---|---|
| GET | `/growth/feed` | **Implemented** — merged feed (class-wide + student-specific) |
| GET | `/growth/feed/:id` | STUB |
| POST | `/growth/feed/:id/like` | STUB |
| DELETE | `/growth/feed/:id/like` | STUB |
| GET | `/growth/feed/:id/comments` | STUB |
| POST | `/growth/feed/:id/comments` | STUB |
| GET | `/growth/album` | STUB |
| GET | `/growth/report` | STUB |

### Dashboard

`GET /admin/dashboard/growth-stats` — Returns 7d/30d/12m aggregated data + top classes ranking.

---

## Business Rules

### Scoping
- **Multi-tenancy**: Every query MUST filter by `kindergarten_id` (from JWT context).
- **Teacher scope**: Teacher operations are additionally scoped to their assigned `class_id`. A teacher can never access another class's growth logs.

### Target Type
- `target_type = 1` (全班): Visible to ALL parents of the class. Meal/sleep/mood templates available.
- `target_type = 2` (指定学生): Only visible to parents of linked students. `growth_log_students` stores the join. Individual-student templates available (no meal/sleep batch rating).

### Audit Flow
1. Teacher creates log → auto-submitted to `content_audits` with `status=0` (pending)
2. Admin reviews → approves (`status=1`) or rejects (`status=2`) with optional `reject_reason`
3. Audit status is JOINed into log list queries via `FindLatestAudit`

### Enum Values
- Meal/Sleep/Mood ratings: `3`=Good, `2`=Normal, `1`=Poor, `0`=Not rated (default)
- Performance tags: stored as comma-separated string `"tag1,tag2"`. Convert to/from `[]string` at the API boundary.

### Media Pattern
- Media can exist independently (`log_id = NULL`) before being attached to a log
- Enables "upload photos first, write log later" workflow
- Media sorting via `sort_num`

### Soft Delete
- All primary entities (logs, comments, media, likes) use GORM soft delete
- Deleted records remain in DB but are filtered by GORM's default scope
- Admin delete creates an `OperationLog` entry

### Sync to Feed
- Controlled by `sync_to_feed` boolean
- When false, the log exists for internal records but does NOT appear in the parent feed
- Teacher sets this per log

### AI Placeholder
- `GrowthMediaStudent.is_auto` field exists for future AI face recognition
- No AI service is currently implemented
- Manual student tagging is supported

---

## Frontend Pages

### Teacher Mini-Program (`teacher/src/pages/growth/`)

| Page | Key UX |
|---|---|
| `index.vue` | Monthly-grouped card list, search bar, category filters (全部/今日/学习/体能/艺术/其他), pull-to-refresh, FAB to create |
| `create.vue` | Date picker with day nav, quick templates (饮食/睡眠/情绪/表现), student selector for target_type=2, photo grid (max 20 photos / 5 videos), content textarea, sync toggle, class/individual mode switcher |
| `detail.vue` | Photo grid with tap-to-preview, related students chip list (max 6 + "+N"), stats bar, threaded comments, bottom comment input |

### Admin Web (`admin/pages/growth/`)

| Page | Key UX |
|---|---|
| `index.vue` | DataTable: date, class, content preview, media count, likes, audit status, actions (approve/reject/delete). Uses `usePagination()`. |
| `statistics/index.vue` | Growth stats alongside leave/task stats. Uses `getGrowthStats()`. |
| `audit/index.vue` | Centralized audit page with type=growth filter. |

Dashboard card: `admin/components/dashboard/GrowthStatsCard.vue` — 4 KPI cards (logs, media, likes, comments).

### Parent Mini-Program (`parent/`) — Not Yet Scaffolded

Design data exists in `design-data/`:
- `02-家长端-成长时光页.json` — Feed design
- `06-家长端-成长报告页.json` — Report design

---

## Key Files

### Backend (Go)
- `server/internal/domain/growth.go` — Entity models
- `server/internal/repository/growth_repo.go` — Data access (complex JOIN queries)
- `server/internal/service/growth_service.go` — Admin service (stats, audit, export)
- `server/internal/service/teacher_portal_service.go` — Teacher CRUD + comments + media
- `server/internal/service/parent_portal_service.go` — Parent feed + stubs
- `server/internal/service/dashboard_service.go` — Dashboard growth stats
- `server/internal/service/export_task_service.go` — Async XLSX export
- `server/internal/handler/http/growth.go` — Admin HTTP handlers
- `server/internal/handler/http/teacher_portal.go` — Teacher HTTP handlers (growth methods)
- `server/internal/handler/http/parent_portal.go` — Parent HTTP handlers (growth methods)
- `server/internal/handler/http/dashboard.go` — Dashboard handlers
- `server/internal/handler/http/export.go` — Export handlers
- `server/internal/app/router.go` — Route registration

### API Specs
- `backend/api/api-admin/09-growth.md`
- `backend/api/api-teacher/06-growth.md`
- `backend/api/api-parent/04-growth.md`

### Admin Frontend
- `admin/api/growth.ts` — API functions
- `admin/pages/growth/index.vue` — List page
- `admin/types/api.d.ts` — TypeScript types

### Teacher Mini-Program
- `teacher/src/api/teacher-growth.js` — API wrappers
- `teacher/src/pages/growth/index.vue` — List
- `teacher/src/pages/growth/create.vue` — Create/Edit
- `teacher/src/pages/growth/detail.vue` — Detail

### Design Data
- `design-data/teacher-04-成长记录页.json`
- `design-data/teacher-08-成长记录详情页.json`
- `design-data/02-家长端-成长时光页.json`
- `design-data/06-家长端-成长报告页.json`

---

## Coding Conventions for This Module

### Adding a New API Endpoint
1. Define request/response structs in the handler file
2. Add a method to the appropriate service (or create a new service method)
3. Add a repository method if new queries are needed
4. Register the route in `server/internal/app/router.go` under the correct group (admin/teacher/parent)
5. Add RBAC policy CSV entry if it's a new permission
6. Run `make wire` in `server/internal/app` if DI changed
7. Update the API spec in `backend/api/`

### Service Pattern
- `GrowthService` — admin-only operations (stats, audit, export). Check `rbac.Enforce(role, "growth:view", "*")`.
- `TeacherPortalService` — teacher operations. Always scope to `teacher.ClassID` from JWT.
- `ParentPortalService` — parent operations. Always scope to children's classes via `StudentParent` join.

### Repository Pattern
- `GrowthRepo` methods take `*gorm.DB` (pre-scoped with kindergarten_id at the handler level)
- Complex list queries use subqueries for counts (media, likes, comments) and latest audit status
- Always use `Preload` for associations, not N+1 queries
- Export queries bypass pagination — use `ListLogsForExportDateRange` with date bounds

### RBAC
- Permission key: `growth:view` (admin menu visibility)
- Audit operations: `audit:view`
- Teacher endpoints: authenticated via JWT (role=teacher), scoped to class
- Parent endpoints: authenticated via JWT (role=parent), scoped to their children

### Frontend Patterns
- Admin: Use `useHttp` composable for all API calls (auto token refresh). Pinia stores for state.
- Teacher: Use `uni.request` interceptor in `src/utils/http.js`. LocalStorage for tokens.
- Always handle the three rating enums with semantic constants, not magic numbers.

---

## Implementation Status

| Feature | Status |
|---|---|
| Teacher CRUD + media upload + comments | ✅ Done |
| Admin list/detail/delete/audit/stats | ✅ Done |
| Admin dashboard growth stats | ✅ Done |
| Growth data export (XLSX) | ✅ Done |
| Teacher like/unlike | ⏳ Stub (hardcoded response) |
| Parent growth feed | ✅ Done |
| Parent detail/like/comments/album/report | ⏳ Stub |
| AI face recognition tagging | ⏳ Schema ready, no service |
| Parent mini-program frontend | ❌ Not scaffolded |
