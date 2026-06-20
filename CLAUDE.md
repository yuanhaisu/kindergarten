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

**WeChat Developer Tools**: Open `teacher/dist/dev/mp-weixin` (compiled output), NOT the `teacher/` source directory. MCP config in `.mcp.json` points to this path.

> ⚠️ WeChat DevTools MCP (`wechat-devtools`) requires Node.js in PATH for screenshot/automator features. The IDE bundles its own Node at `D:\Program Files (x86)\微信web开发者工具\node.exe`.

## 🧪 小程序测试验证纪律

**MCP 导航 ≠ 用户操作验证。** `wechat_navigate` 通过 CDP 直接调用 IDE 的路由跳转 API（`reLaunch` / `switchTab`），完全绕过了页面内的业务代码。因此：

1. **页面渲染验证**：用 `wechat_navigate` 跳转 + `wechat_inspector` 采集 console → 验证页面能正常加载、无 JS 报错 ✅
2. **UI 交互验证**：点击按钮、填写表单、触发事件 → **必须**通过 `wechat_automator`（tap / input）或真机操作，不能用 MCP 导航替代 ❌
3. **导航跳转验证**：`uni.navigateTo` vs `uni.switchTab` 的区别只在用户点击触发时才会暴露——MCP 导航不存在此限制。自查要点：
   - 所有跳转到 tabBar 页面的地方必须用 `switchTab`，不能用 `navigateTo`
   - tabBar 页面列表在 `pages.json` → `tabBar.list[].pagePath` 中定义
   - 常见的 tabBar 页面：首页、成长时光、亲子任务、我的
4. **每次功能变更后**，除了 MCP 导航遍历全部页面外，还必须检查新增/修改的函数入口（按钮 `@click`、`onLoad` 等）是否在真实交互场景下会触发错误

### MCP 工具能力矩阵

| 工具 | 能力 | 需要 Node.js |
|------|------|-------------|
| `wechat_navigate` | 跳转页面 + CDP 日志采集 | 否 |
| `wechat_inspector` | 控制台日志/异常采集 | 否 |
| `wechat_ide` | IDE 生命周期（打开/关闭项目） | 否 |
| `wechat_automator` | tap / input / evaluate / page_data | **是** |
| `wechat_screenshot` | 截图 | **是** |

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

## 🎨 Design Token 规范（三端必须遵守）

养成星球使用统一的 design token 系统管理三端样式。**所有颜色、圆角、阴影、间距、字号必须通过 token 引用，禁止硬编码。**

单一数据源: `shared-design-tokens/tokens.json` → 构建产出三端各自格式的文件。

### 决策树

遇到任何样式值，先查 token：
- `tokens.json` 里有的 → **必须用 token 变量**，禁止硬编码
- 没有的 → 评估是否「品牌基因」级别（会被多处使用） → 在 tokens.json 中新增 → `cd shared-design-tokens && npm run build` → 引用
- 仅一处使用的特殊值（渐变、遮罩）→ 允许局部硬编码，**加注释说明原因**

### 小程序端（teacher / parent）— SCSS

```scss
// ✅ 正确：引用 token 变量
.card {
  background: $color-bg-card;
  border-radius: $border-radius-lg;
  padding: $spacing-md $spacing-lg;
  box-shadow: $shadow-sm;
  color: $color-text-primary;
  font-size: $font-size-base;
}

// ✅ 正确：语义色
.tag-success { background: $color-semantic-success-bg; color: $color-semantic-success; }

// ❌ 错误：硬编码值（这些值在 tokens.json 中都有对应 token）
.card {
  background: #fff;          // 应使用 $color-bg-card
  border-radius: 12rpx;      // 应使用 $border-radius-md
  padding: 16rpx 24rpx;      // 应使用 $spacing-md $spacing-lg
  color: #1E293B;            // 应使用 $color-text-primary
}
```

### 管理端（admin）— Tailwind

```html
<!-- ✅ 正确：使用 token 类名 -->
<div class="bg-bg-card rounded-lg shadow-sm p-4 text-text-primary">

<!-- ✅ 正确：语义色 -->
<span class="bg-semantic-successBg text-semantic-success">已通过</span>

<!-- ❌ 错误：Tailwind 任意值 -->
<div class="bg-[#ffffff] rounded-[12px] p-[16px]">
```

### 禁止的行为

- 组件 `<style>` 中**重复定义 token 已有变量**（如 `$blue-500: #3B82F6;`）——应直接使用 token 变量
- 使用 Tailwind 任意值替代 token 类名（如 `bg-[#3B82F6]` 替代 `bg-brand-primary`）
- 修改 token 值不更新 `tokens.json`（应改源头，不是改生成产物）
- 在 `<style scoped>` 中重新声明 `$slate-50: #F8FAFC` 等已在 token 中定义的值

### 新增 / 修改 Token 流程

1. 编辑 `shared-design-tokens/tokens.json`（按 `category.type.item` 命名规范）
2. `cd shared-design-tokens && npm run build`
3. `git diff build/` 验证生成产物
4. 提交 `tokens.json` + `build/`（产物必须提交）

### Token 速查

| 分类 | Token 示例 | 端差异 |
|---|---|---|
| 品牌主色 | `$color-brand-primary` | 老师端 #ff8a3d / 家长端 #F97316 / 管理端 #3B82F6 |
| 背景 | `$color-bg-page`, `$color-bg-card` | 端差异 |
| 文字 | `$color-text-primary`, `body`, `muted`, `hint` | 共享 |
| 功能色 | `$color-semantic-success`, `warning`, `danger`, `info` + 对应的 `-bg` 浅底 | 共享 |
| 圆角 | `$border-radius-sm`(8rpx) ~ `xl`(48rpx), `full`(9999rpx) | 共享 |
| 阴影 | `$shadow-sm`, `md`, `lg` | 共享 |
| 间距 | `$spacing-xs`(8rpx) ~ `xl`(64rpx) | 共享 |
| 字号 | `$font-size-xs`(22rpx) ~ `3xl`(40rpx) | 共享 |
| 字重 | `$font-weight-normal`(400) ~ `bold`(700) | 共享 |
| 动画 | `$animation-duration-fast`(150ms), `base`(300ms), `slow`(500ms) | 共享 |

### 组件级 Token（原子 token 之上的语义层）

优先使用组件级 token 描述 **UI 模式意图**，而非直接使用原始 token：

```scss
// ✅ 推荐：组件级 token 表达意图
.card { padding: $component-card-padding; border-radius: $component-card-radius; }
.page { padding: $component-page-padding-y $component-page-padding-x; }

// ⚠️ 可接受但意图不明确
.card { padding: $spacing-xl; border-radius: $border-radius-xl; }
```

| 分类 | Token | 值 | 说明 |
|---|---|---|---|
| **卡片** | `$component-card-padding` | 32rpx | 卡内边距 |
| | `$component-card-radius` | 24rpx | 卡片圆角 |
| | `$component-card-gap` | 24rpx | 卡片列表间距 |
| **页面** | `$component-page-padding-x` | 28rpx | 页面水平内边距 |
| | `$component-page-padding-y` | 24rpx | 页面垂直内边距 |
| **头像** | `$component-avatar-sm` | 80rpx | 小头像（作者/评论） |
| | `$component-avatar-md` | 96rpx | 中头像（学生列表） |
| | `$component-avatar-lg` | 128rpx | 大头像（个人主页） |
| **输入框** | `$component-input-height` | 88rpx | 输入框高度 |
| | `$component-input-radius` | 24rpx | 输入框圆角 |
| **标签** | `$component-badge-padding-x` | 20rpx | 标签水平内边距 |
| | `$component-badge-radius` | 9999rpx | 标签圆角（胶囊） |
| | `$component-badge-font-size` | 22rpx | 标签字号 |

**产物路径:**
- Teacher SCSS: `kindergarten-design-tokens/tokens-teacher`
- Parent SCSS: `kindergarten-design-tokens/tokens-parent`
- Admin TS: `import tokens from 'kindergarten-design-tokens'`
- CSS 变量 (Teacher): `kindergarten-design-tokens/tokens-teacher.css`
- CSS 变量 (Parent): `kindergarten-design-tokens/tokens-parent.css`

### 运行时主题切换

所有 token 同时生成为 CSS 自定义属性（`--color-brand-primary` 等），在 `App.vue` 中自动注入。可通过 `theme.js` 运行时切换：

```js
import { applyTheme, resetTheme, brandToTheme, BRANDS } from 'kindergarten-design-tokens/theme'

// 切换到预设品牌色
applyTheme(brandToTheme(BRANDS.ocean))

// 自定义主题
applyTheme({ '--color-brand-primary': '#FF6B35', '--color-bg-page': '#FFFBF5' })

// 恢复默认
resetTheme()
```

### Lint 检查

```bash
node scripts/check-tokens.mjs          # 扫描三端
node scripts/check-tokens.mjs teacher  # 只扫描老师端
```

在提交代码前运行，确保无 token 违规。

## Key constraints

- Never commit `configs/app.local.ini`, `.env` files, or credentials in `app.ini`
- Wire-generated file `internal/app/wire_gen.go` must be regenerated (`make wire`) after DI changes
- API specs in `backend/api/` are the source of truth; handlers and services must conform to them
- Design tokens source: `shared-design-tokens/tokens.json`. Generated `build/` files must be committed alongside source changes
