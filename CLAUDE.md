# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.


# Claude Code - Project Guidelines (Karpathy Style)

## 🎯 Highest Execution Principles (最高行动纲领)
1. **Never Blind Guess (拒绝盲猜)**: Do not attempt to fix errors or refactor code without understanding the full context. If you encounter an unclear state, STOP immediately and ask the user to clarify.
2. **Read Before Writing (先读后写)**: Before modifying any function or file, you MUST read the adjacent functions and files to ensure consistency. Do not duplicate existing logic.
3. **Surgical Precision (外科手术式修改)**: Minimize lines changed. Do not rewrite whole files or modify unrelated code snippets for "aesthetic" reasons.
4. **Test Real Logic (测试真逻辑)**: Passing tests is NOT the only goal. Do not write shallow test overrides just to make a pipeline green while breaking actual production logic.

## 🛠️ Command Rules (指令约束)
- **Checkpoints**: For multi-step refactoring, create a summary checkpoint after each step. Do not proceed to step N+1 if step N fails or behaves unexpectedly.
- **Build & Lint**: Run localized build/lint commands (`npm run build` or equivalent) *before* declaring a task complete to catch syntax/type errors silently.
- **Silent Failures**: Always add explicit error boundaries and logs. Never catch exceptions silently.

## ✍️ Coding Style & Preferences
- **Language**: TypeScript / Python (根据你项目实际情况修改)
- **Patterns**: Prefer composition over inheritance. Keep functions small and pure.
- **Comments**: Write why, not what. Avoid obvious comments.

## Project Overview

**养成星球 (Nurture Planet)** — a kindergarten digital management platform. Three client apps served by one Go backend, with `kindergarten_id`-based multi-tenant isolation throughout.

| App                  | Directory  | Tech                                    | Status      |
| -------------------- | ---------- | --------------------------------------- | ----------- |
| Teacher mini-program | `teacher/` | uni-app + Vue 3 + Vite (WeChat)         | Active      |
| Admin web            | `admin/`   | Nuxt 3 + Element Plus + Pinia + ECharts | Active      |
| Parent mini-program  | `parent/`  | uni-app (WeChat)                        | Not started |
| Backend server       | `server/`  | Go + Gin + GORM + Redis + Casbin        | Active      |

API contracts live in `backend/api/` (spec-first). Data models live in `backend/model/`. Design data JSON files live in `design-data/`.

## Spec Workflow (功能规格化)

**Before implementing any non-trivial feature**, follow this workflow:

1. Write a spec file in `docs/specs/` using the template `docs/specs/TEMPLATE.md`
2. Name the file `YYYY-MM-DD-feature-name.md`
3. Fill in: context → data model → API design → frontend changes → edge cases → acceptance criteria
4. Get user approval on the spec before writing code
5. Update the spec status to ✅ completed after implementation

Existing specs live in `docs/specs/`. Refer to them when revisiting features built under this system.

Project-specific knowledge skills live in `.claude/skills/`:
- `growth-records` — Growth records module business context
- `system-enums` — All hardcoded enums across the platform
- `golang-wechat-backend` — Backend architecture patterns

## Commands

### Go backend (`server/`)

```bash
make run               # Start dev server (CONFIG=configs/app.ini go run cmd/api/main.go)
make build             # Build binary to bin/
make wire              # Regenerate Wire DI code (cd internal/app && wire)
make test              # Run all tests with race detector
make test-cover        # Run tests + coverage HTML report
make lint              # golangci-lint run ./...
make tidy              # go mod tidy
make check             # vet + lint + test
```

Config file path via env: `CONFIG_PATH=configs/app.local.ini make run`. Sensitive values can be overridden via env vars: `MYSQL_DSN`, `REDIS_ADDR`, `JWT_SECRET`, `WECHAT_APP_ID`, `WECHAT_SECRET`, `OSS_*`, etc.

### Admin web (`admin/`)

```bash
npm run dev            # Start dev server with HMR (Vite proxies /api/v1 to :8080)
npm run build          # Production build
npm run preview        # Preview production build
```

Copy `.env.example` to `.env` to configure `NUXT_PUBLIC_API_BASE`. In production, use Nginx to reverse-proxy `/api/v1` to the Go backend.

### Teacher mini-program (`teacher/`)

```bash
npm run dev:mp-weixin         # Dev build for WeChat
npm run build:mp-weixin       # Production build
npm run build:mp-weixin:test  # Test env build
```

Open `dist/dev/mp-weixin` in WeChat Developer Tools. Update `src/manifest.json` with real AppID before release.

## Architecture

### Backend layer (server/)

```
cmd/api/main.go          # Entry: load config → init slog → Wire inject → gin.Run
configs/
  app.ini                # INI config (DSN, Redis, JWT, OSS, Casbin)
  rbac_model.conf        # Casbin RBAC model definition
  rbac_policy.csv        # Casbin initial policy
internal/
  app/                   # Wire injector, providers, router registration, wire_gen.go
  config/                # INI loader with env var override
  auth/                  # JWT issue/parse, WeChat code2session, refresh token
  middleware/             # RequestID → CORS → Logger → AuthJWT → CasbinRBAC
  rbac/                  # Casbin enforcer init
  domain/                # GORM entity models (~40 models covering 6 business layers)
  repository/            # Data access layer (one repo file per domain)
  service/               # Business logic layer (one service file per domain)
  handler/http/          # HTTP handlers: Handler struct aggregates all *Service fields
  storage/oss/           # Alibaba Cloud OSS (presigned URLs, upload)
pkg/
  log/                   # slog init + context propagation
  response/              # Unified JSON envelope (code/message/data/request_id)
  pagination/            # Page/PageSize helpers
```

**Middleware chain**: `gin.Recovery → CORS → RequestID → RequestContextLogger → [AuthJWT → CasbinRBAC on authed groups]`

**Auth flow (mini-program)**: Client calls `wx.login()` → gets code → `POST /api/v1/auth/login/wechat` with code + role → backend calls WeChat code2session → issues JWT access + refresh tokens.

**Auth flow (admin)**: `POST /api/v1/auth/login/password` with username/password → JWT access + refresh tokens. Tokens stored in cookies (`admin_token`, `admin_refresh`).

**Casbin RBAC**: Enforcer checks `(role, route_path, http_method)` for every authenticated request. `CasbinRBAC` middleware iterates user's roles and allows if any role matches. Admin-specific RBAC (roles/permissions) also stored in DB (`admin_rbac.go`).

**Wire DI**: After adding a new repository/service/handler, run `make wire` in `server/internal/app` to regenerate `wire_gen.go`. The handler struct aggregates all services; a new module requires updates to handler struct, constructor, provider set, and router registration.

### API route structure (`/api/v1`)

- `POST /auth/login/wechat|password|sms`, `POST /auth/refresh`, `POST /auth/logout`
- `GET|PUT /common/user/profile`, `POST /common/upload`
- `GET|PUT /common/messages`, `/common/messages/unread-count`
- `/admin/*` — 20+ resource groups (dashboard, classes, students, teachers, parents, attendance, leaves, pickup, growth, tasks, points, achievements, notices, announcements, audit, admins, roles, settings, export, logs)
- `/teacher/*` — 8 resource groups (home, notices, leaves, growth, pickup, calendar, class, tasks)

### Data model layers (from `backend/model/README.md`)

1. **Organization & People**: Kindergarten, Class, User (+ UserRole), Student, StudentParent, ClassTeacher
2. **Notices & Communication**: Notice, NoticeReadLog, NoticeTemplate, LeaveRequest, Announcement
3. **Attendance & Pickup**: Attendance, PickupRecord, PickupAuthorization, PickupQRCode
4. **Growth Records**: GrowthDailyLog, GrowthMedia, GrowthComment, GrowthLike, GrowthReport
5. **Tasks & Points**: ParentTask, TaskCheckIn, PointAccount, PointTransaction, Achievement
6. **Admin-only**: ContentAudit, AdminRole, AdminPermission, OperationLog

All business tables include `kindergarten_id` for multi-tenancy. Core business data uses `SoftDeleteModel`.

### Admin web structure (`admin/`)

- SPA mode (`ssr: false`), cookie-based auth with global middleware `auth.global.ts` redirecting to `/login`
- `api/` — per-module API functions built on `useHttp` composable
- `stores/` — Pinia stores: `auth` (session), `permission` (RBAC codes, defaults to `['*']`), `app` (sidebar collapse)
- `middleware/auth.global.ts` — checks `admin_token` cookie on every route
- `composables/` — `useHttp` (Axios wrapper with auto token refresh), `useAuth`, `usePagination`, `usePermission`
- Sidebar menu defined in `constants/menus.ts` with permission codes
- Vite dev proxy forwards `/api/v1` to `http://127.0.0.1:8080`

### Teacher mini-program structure (`teacher/`)

- `src/pages.json` — uni-app page routing (15 pages: index, auth, notice, growth, pickup, tasks, calendar, leave, class, album, messages, more, profile, webview)
- `src/utils/login.js` — `ensureLogin()` calls `wx.login()` → `POST /auth/login/wechat` with `role=teacher`
- `src/utils/auth.js` — localStorage-based token storage (uni.setStorageSync)
- `src/api/` — per-module API wrappers using a shared `http.js` (uni.request interceptor with auto token refresh)
- Login is silent at app launch; `App.vue` `onLaunch` calls `ensureLogin()`

## Key constraints

- Never commit `configs/app.local.ini`, `.env` files, or credentials in `app.ini`
- Wire-generated file `internal/app/wire_gen.go` must be regenerated (`make wire`) after DI changes
- API specs in `backend/api/` are the source of truth; handlers and services must conform to them
- Parent mini-program (`parent/`) is a planned uni-app project but has not been scaffolded yet
