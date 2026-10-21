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

| App                  | Directory  | Tech                                    | Status |
| -------------------- | ---------- | --------------------------------------- | ------ |
| Teacher mini-program | `teacher/` | uni-app + Vue 3 + Vite (WeChat)         | Active |
| Admin web            | `admin/`   | Nuxt 3 + Element Plus + Pinia + ECharts | Active |
| Parent mini-program  | `parent/`  | uni-app + Vue 3 + Vite (WeChat)         | Active |
| Backend server       | `server/`  | Go + Gin + GORM + Redis + Casbin        | Active |

API contracts live in `backend/api/` (spec-first). Data models live in `server/internal/domain/`（**不是** `backend/model/`，见下方说明）. Design data JSON files live in `design-data/`.

> `backend/` 下**两个子目录状态不同**，别看错：
> - `backend/api/`（`api-admin/`、`api-teacher/`、`api-parent/`、`api-common.md`）— **仍在维护**，是接口契约的 source of truth，改接口先改这里。
> - `backend/model/`（18 个 .go 文件，全部停留在 2026-03-22）— **早期模型草案，已停更**，被 `server/internal/domain/`（29 个文件 / 58 个 struct）取代。**以 `server/internal/domain/` 为准**。该目录的 `README.md` 本身也已漂移（例如它列出 `leave.go`，但该文件并不存在）。

## Spec Workflow (功能规格化)

**规格化工具：OpenSpec**（`openspec/` + `/opsx:*` 命令）。曾评估过 GitHub Spec Kit，结论是不引入——理由与翻盘条件见文末。

### 先判断要不要走规格

| 变更类型 | 用什么 |
| --- | --- |
| 触点 < 3 层（单点 bug、typo、样式微调、配置） | **不写规格**，直接改，commit message 说清 |
| 诊断困难的 bug | `superpowers:systematic-debugging`，不写规格 |
| 触点 ≥ 3 层的功能切片（跨 Go/契约/端页面/token） | **走 OpenSpec** |
| 做不做的概念决策 | `/opsx:explore`，想清楚再决定要不要 propose |

### 走 OpenSpec 的流程

```
/opsx:explore（可选，想清楚）→ /opsx:propose → 用户批准 → /opsx:apply → 验证 → /opsx:archive
```

1. **propose 只产规划工件**（proposal / specs / design / tasks），不动代码。等用户明确批准后再 apply
2. **design 必须给出「跨端触点清单」**：Go domain/repo/service/handler + Casbin + router + `backend/api/` + 各端页面 + token。每一项写「会改」或「不改」
3. **specs 只写行为与验收**（`SHALL` + Scenario），每条场景引用 `docs/tests/TEST_CATALOG.md` 的 TC ID
4. **archive 前必须逐条对照 specs 验证**，并把 delta 合并进 `openspec/specs/` 活基线

详细约束已注入 `openspec/config.yaml` 的 `context` / `rules`，生成工件时自动生效。
**日常使用手册（含大改/小改示例、命令速查、踩坑清单）见 `docs/openspec-usage.md`。**

### 语义所有权（谁是真相源）

规格层**只占「行为 + 验收」这一格**，不得侵入下列已饱和的语义：

| 语义 | 唯一 SoT |
| --- | --- |
| 接口形状（路由/字段/包络） | `backend/api/`（改接口先改这里） |
| 数据结构（表/字段/关联） | `server/internal/domain/` |
| 架构铁律 / 硬规则 / 已知坑 | 本文件（CLAUDE.md） |
| 样式 / 间距 / 色值 | `shared-design-tokens/tokens.json` |
| 测试用例编号 | `docs/tests/TEST_CATALOG.md` |
| **行为需求 + 验收标准** | **`openspec/specs/`** |

### 历史规格

`docs/specs/` 是 **2026-06 之前的旧工作流产物，已冻结**，仅作回溯参考，不要再新增文件。新变更一律走 OpenSpec。

### 为什么不是 Spec Kit

OpenSpec 的 delta→archive 机械同步机制对上本项目「写了就烂」的失败模式（`docs/specs/`、`backend/model/`、`backend/model/README.md` 三处腐烂是实证）；Spec Kit 默认产出 `contracts/` / `data-model.md` 会制造新的真相源。其强项（跨端触点结构、验证收敛门）已写进 `openspec/config.yaml` 的 `rules.design` / `rules.tasks`，不必为此换工具链。

**翻盘条件**（任一命中就重新评估 Spec Kit）：功能切片远多于小修；决定让规划步骤直接产出 `backend/api/` 契约；出现第二位协作者需要宪法+评审流程；OpenSpec 用 2–3 个 change 后规格照样腐烂。

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

Config file path via env: `CONFIG_PATH=configs/app.local.ini make run`. Sensitive values can be overridden via env vars: `MYSQL_DSN`, `REDIS_ADDR`, `JWT_SECRET`, `WECHAT_TEACHER_APP_ID`, `WECHAT_TEACHER_SECRET`, `WECHAT_PARENT_APP_ID`, `WECHAT_PARENT_SECRET`, `OSS_*`, etc.（旧的 `WECHAT_APP_ID`/`WECHAT_SECRET` 仍被识别，仅用于回滚到共用 AppID）

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

> ⚠️ **`wechat_automator` / `wechat_screenshot` 需要 Node.js，但开关是 `NODE_PATH`，不是 `PATH`。**
> `wechat-devtools-mcp` 检测 Node 时**不做 PATH 查找**（源码里没有 `shutil.which`），只按顺序试 `os.environ["NODE_PATH"]` 和几个硬编码路径（`C:\Program Files\nodejs\node.exe`、`%APPDATA%\nvm\current\node.exe` 等）。
> 本机 Node 装在 `E:\software\nodejs`，不在候选里，所以必须在 `.mcp.json` 的 `env` 中显式指定：
>
> ```json
> "NODE_PATH": "E:\\software\\nodejs\\node.exe"
> ```
>
> 改完**必须重启 Claude Code**（env 在进程 spawn 时固定，热改无效）。判定是否生效：`wechat_ide(action='status')` 返回的 `node_available` 字段。

## 🧪 小程序测试验证纪律

**MCP 导航 ≠ 用户操作验证。** `wechat_navigate` 通过 CDP 直接调用 IDE 的路由跳转 API（`reLaunch` / `switchTab`），完全绕过了页面内的业务代码。因此：

1. **页面渲染验证**：用 `wechat_navigate` 跳转 + `wechat_inspector` 采集 console → 验证页面能正常加载、无 JS 报错 ✅
2. **UI 交互验证**：点击按钮、填写表单、触发事件 → **必须**通过 `wechat_automator`（tap / input）或真机操作，不能用 MCP 导航替代 ❌
3. **导航跳转验证**：`uni.navigateTo` vs `uni.switchTab` 的区别只在用户点击触发时才会暴露——MCP 导航不存在此限制。自查要点：
   - 所有跳转到 tabBar 页面的地方必须用 `switchTab`，不能用 `navigateTo`
   - tabBar 页面列表以 `pages.json` → `tabBar.list[].pagePath` 为**唯一依据**，两端的 tabBar 配置不同：
     - **家长端**（`parent/`）**有** tabBar，4 个 tab：`pages/index/index`（首页）、`pages/growth/index`（成长时光）、`pages/pickup/index`（接送）、`pages/profile/index`（我的）
     - **老师端**（`teacher/`）**没有** tabBar（`pages.json` 中无 `tabBar` 字段），入口靠 `pages/index/index` 工作台 + `pages/more/index` 功能聚合页。因此老师端**不存在** `switchTab` 场景
   - ⚠️ 老师端没有 tabBar 这一点容易误判：新增跳转时不要"照着家长端的样子"补 `switchTab`
4. **每次功能变更后**，除了 MCP 导航遍历全部页面外，还必须检查新增/修改的函数入口（按钮 `@click`、`onLoad` 等）是否在真实交互场景下会触发错误

### MCP 工具能力矩阵

| 工具 | 能力 | 依赖 |
|------|------|------|
| `wechat_file` | 读取编译产物（project_info / 页面列表） | 无 |
| `wechat_ide` | IDE 生命周期、环境诊断 | 无 |
| `wechat_navigate` | 跳转页面 + **CDP** 日志采集 | 无（走 CDP 9222） |
| `wechat_inspector` | 控制台日志/异常采集 | 无 |
| `wechat_automator` | tap / input / evaluate / page_data | **Node + 自动化端口** |
| `wechat_screenshot` | 截图（支持长图拼接） | **Node + 自动化端口** |

### ⚠️ 三个会让结论失真的陷阱

**1. `success: true` 不代表"验证的是我想要的那个项目"。** 两个 MCP server（`wechat-devtools` 老师端 / `wechat-devtools-parent` 家长端）共用一个 IDE 会话时，会出现：

- 两个 server 各自 `automator start` 都返回 `success: true`，随后 `page_data` 却返回**逐字节相同**的数据 —— 在驱动同一个会话
- parent server 能"成功"导航到只在老师端存在的 `pages/more/index`

**判别是否真的双端**：探该端**独有**页面 + 比对 `page_data` 业务字段是否不同。详见下方「🔀 双端同时验证」。

> 🔴 **`success: true` 不代表"验证的是我想要的那个项目"。**

**2. appid 必须是真实值。** 占位符 `请替换为真实小程序AppID` 会让 `cli auto` 在 prepare 阶段以 `code 10 AppID 不合法` 中止 → 端口 9420 永不监听 → automator/screenshot 全废。MCP 把这个报成"CLI auto 连续 3 次返回成功但端口 9420 未监听"——**CLI 退出码是 0**，极具误导性。改完 appid **必须重新构建**（IDE 读 `dist/dev/mp-weixin/project.config.json`，不是 `src/manifest.json`）。

**3. 构建永远可并行；MCP 验证能否并行取决于 IDE 拓扑。** 两个 uni-app 项目的写代码/构建互不干扰。验证环节：

- **单 IDE 实例**（默认）：MCP 验证必须串行，一次驱动一个端
- **双 IDE 实例 + 双微信账号**（`scripts/dual-ide.ps1`）：两端可同时 MCP 驱动，跨端数据流可自动化验证 —— 见下方「🔀 双端同时验证」

### ✅ 可复用清单（每次用 MCP 验证前逐条过）

| # | 检查项 | 怎么做 |
|---|--------|--------|
| 1 | **确认目标端** | 看 IDE 窗口标题，并探一个该端**独有**页面（见下表） |
| 2 | **确认产物是新的** | 源码改过就先 `npm run dev:mp-weixin`；否则验的是陈旧产物 |
| 3 | **确认 Node 可用** | `wechat_ide(action='status')` → `node_available: true` |
| 4 | **确认端口就绪** | `wechat_automator(action='start')` → `tcp_ready` 与 `ws_ready` 均为 true |
| 5 | **开始验证** | 渲染用 `wechat_navigate`；交互**必须**用 `wechat_automator` |
| 6 | **收工清理** | 杀掉残留的 uni watcher（见下） |

**判别目标端的独有页面：**

| 端 | 独有页面 |
|----|----------|
| 老师端 | `pages/more/index`、`pages/doctor/checkin`、`pages/album/index`、`pages/notice/create` |
| 家长端 | `pages/points/index`、`pages/student/index`、`pages/growth/report`、`pages/feedback/index` |

**收工清理**：`npm run dev:mp-weixin` 是 watch 进程，`TaskStop` 只终止 shell 包装，`@dcloudio/vite-plugin-uni` 会残留并随时重新构建、弄脏工作区。按 PID 杀：

```powershell
Get-CimInstance Win32_Process -Filter "Name='node.exe'" |
  Where-Object { $_.CommandLine -match "dev:mp-weixin|kindergarten\\(teacher|parent)\\node_modules" } |
  ForEach-Object { Stop-Process -Id $_.ProcessId -Force }
```

### 🔀 双端同时验证（跨端数据流联调）

要验证「老师端操作 → 家长端响应」这类跨端数据流，需要两端**同时在线**。有三种形态：

| 形态 | 双端同时在线 | 可自动化 | 适用 |
|---|---|---|---|
| **双 IDE 实例 + 双微信账号** | ✅ | ✅ 两端都能 MCP 驱动 | 跨端数据流自动化验证 |
| 单 IDE 双项目窗口 | ✅ | ❌ 只能串行 MCP | 人工对照观察 |
| 一端 MCP + 一端真机预览 | ✅ | 半 | 一端自动化、一端人工点 |

#### 双 IDE 实例方案（已实测可行，2026-10-05）

**原理**：微信开发者工具的登录态是「**单账号单客户端**」。同一账号开两个实例会互踢（`islogin` 立即 `false`）。用**两个不同微信号**登录两个实例（各自独立 `--user-data-dir`），登录态不互踢，即可双端并行。

**前置（一次性人工配置）**：

1. 第二个微信号加为老师端小程序的开发者（[mp.weixin.qq.com](https://mp.weixin.qq.com) → 成员管理 → 开发者）
2. 两个实例各自登录对应微信号（首次会弹二维码）

**端口分配**：

| 实例 | 账号 | 项目 | CDP | 服务端口 | 自动化 |
|---|---|---|---|---|---|
| A | 主账号 | 家长端 | 9222 | 43534 | 9420 |
| B | 第二账号 | 老师端 | 9223 | 44500 | 9421 |

**启动**：

```powershell
powershell -ExecutionPolicy Bypass -File scripts/dual-ide.ps1 -Kill   # 先清干净
powershell -ExecutionPolicy Bypass -File scripts/dual-ide.ps1 -Auto   # 启动两端 + 开自动化
powershell -ExecutionPolicy Bypass -File scripts/dual-ide.ps1 -Status # 看端口状态
```

**MCP 调用时必须显式指定端口**（否则默认连 9222/9420 会驱动错端）：

```
老师端: wechat_automator(auto_port=9421)  wechat_navigate(cdp_port=9223)
家长端: wechat_automator(auto_port=9420)  wechat_navigate(cdp_port=9222)
```

⚠️ **CLI 端口发现写死读** `%LOCALAPPDATA%\微信开发者工具\User Data\<hash>\Default\.ide`，**不认 `--user-data-dir`**。要用 `cli.bat` 驱动实例 B，先改写该文件内容为 `44500`（`dual-ide.ps1` 的 `Set-IdePort` 已封装）。`.ide` 切换只在 `cli auto`/`open` 等启动期调用时需要；automator 走 ws 直连，不受影响。

⚠️ **残留 `cli login` 进程会占 3799 端口并弄死后启动的实例**。启动前先 `-Kill`，它会连带清掉 `node.exe ...cli...login` 残留。

**验证双实例确实各自独立**（不要只看 `success:true`）：

- 两端 `page_data` 返回的字段应明显不同（老师端有「发布通知/成长记录」，家长端有「我要请假/接送管理」）
- 探独有页面：老师端 `pages/more/index`，家长端 `pages/points/index`

#### 单 IDE 双项目窗口（人工对照）

同一 IDE 实例可同时打开两个项目窗口（窗口标题分别显示「幼儿园老师端」「幼儿园家长端」）。适合人工对照观察，但 MCP 一次只能驱动一个窗口。

打开方式：IDE 菜单或项目列表里打开第二个项目即可（`cli open-other --project X` **无效**——它忽略 `--project` 参数，只弹项目选择界面）。

## Architecture

### Backend layer (server/)

```
cmd/
  api/main.go            # Entry: load config → init slog → Wire inject → gin.Run
  <helper>/main.go       # 一次性/运维辅助命令，见下方「辅助命令」
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
  domain/                # GORM entity models (58 structs covering 6 business layers)
  repository/            # Data access layer (one repo file per domain)
  service/               # Business logic layer (one service file per domain)
  handler/http/          # HTTP handlers: Handler struct aggregates all *Service fields
  storage/oss/           # Alibaba Cloud OSS (presigned URLs, upload)
pkg/
  log/                   # slog init + context propagation
  response/              # Unified JSON envelope (code/message/data/request_id)
  pagination/            # Page/PageSize helpers
```

**辅助命令（`cmd/` 下的独立 `main.go`）**：不进 HTTP 服务，用 `go run ./cmd/<name>` 单独执行。除 `initcheck` 外均读取 `configs/app.ini`。

| 命令 | 作用 | 性质 |
| ------------ | ---------------------------------------------- | ---- |
| `chkgrowth`  | 打印 kindergarten 4 的成长记录总数与最近 3 条 | 只读 |
| `dbcheck`    | 打印指定表的索引定义与行数，用于核对迁移是否生效 | 只读 |
| `fixpwd`     | 把指定手机号家长的密码重置为固定值 | 写入 |
| `linkwx`     | 打印指定手机号的 openid 绑定情况 | 只读 |
| `openid`     | 打印最近 5 个用户 openid，并把最新 openid 绑给家长账号 | 写入 |
| `seedgrowth` | 灌入成长记录种子数据 | 写入 |
| `seedattendance` | 灌入出勤演示数据（3 个工作日 × 全体在读学生） | 写入 |
| `seedleaves` | 灌入 6 条请假演示数据 | 写入 |
| `seedpickup` | 灌入当日接送记录 + 待审核授权人 | 写入 |
| `refreshtasks` | **删除** kindergarten 4 的全部 `ParentTask` / `TaskCheckIn` 后重建种子任务 | ⚠️ **破坏性** |
| `initcheck`  | 创建测试园所+管理员并打印计数，随后 **`Unscoped().Where("1=1")` 清空 `users` / `user_roles` / `kindergartens`** | ⚠️ **破坏性** |

> ✅ **三个 `seed*` 命令的由来**：出勤 / 请假 / 接送的演示数据原先由 service 层的
> `ensureBootstrap*` 函数在**读取路径**上按需生成。那意味着打开一次后台看板或列表页
> 就会往生产表写入编造数据。现已全部移出，只能由这些命令显式触发。
> **新增种子数据时请循此例，不要再往读取路径挂写操作。**
>
> 三者共用 `cmd/internal/seedutil`（配置、连库、园所、时区），
> 园所默认 4，可用 `SEED_KG_ID=7 go run ./cmd/seedleaves` 覆盖；均幂等。

> ⚠️ 这批命令是**针对特定 kindergarten（id=4）的一次性调试脚本**，不是通用工具：幼儿园 ID、班级 ID、手机号、任务文案全部硬编码。**不要在生产库上执行。**
>
> ⚠️ 两个破坏性命令：`refreshtasks` 清空该园全部任务；`initcheck` 更彻底——`Unscoped().Where("1=1")` 会**无视软删除、跨租户清空全部用户、角色与园所**。后者只适用于一次性的测试库自检。
>
> `initcheck` 的连接串已改为读 `configs/app.ini`（可用 `MYSQL_DSN` 覆盖），管理员口令改为读环境变量 `INIT_ADMIN_PASSWORD`（未设置则直接退出）。**不要再把凭据写回源码**——即使后续删除，git 历史仍可追溯。

> 🔧 **`dbcheck` 的用途**：本仓库**没有迁移工具**（只有 `AutoMigrate`，而它只增不改——同名索引已存在就会被跳过）。所以手工迁移后需要一个核实手段：
>
> ```bash
> go run ./cmd/dbcheck task_check_ins
> ```
>
> 会打印该表的全部索引（标注唯一/普通、列出列序）与行数。改完 struct tag 或跑完 `docs/migrations/` 下的 SQL 后，用它确认线上库真的变了，而不是假设 `AutoMigrate` 会处理。

**Middleware chain**: `gin.Recovery → CORS → RequestID → RequestContextLogger → [AuthJWT → CasbinRBAC on authed groups]`

**Auth flow (mini-program)**: Client calls `wx.login()` → gets code → `POST /api/v1/auth/login/wechat` with code + role → backend calls WeChat code2session → issues JWT access + refresh tokens.

> ⚠️ **两端用两个不同的 AppID**，后端 `WeChatConfig` 是按端双对（`teacher_*` / `parent_*`），`Code2Session` 按登录 `role` 选凭据。
>
> | 端 | AppID | 主体 |
> |---|---|---|
> | 家长端 `parent/` | `wxdf989efff8796c53` | 非个人（`getPhoneNumber` 只在家长端用，个人号不支持） |
> | 老师端 `teacher/` | `wxbfb5f81a87f73a61` | **个人号** |
>
> **为什么必须拆开**：openid 是「用户×小程序」二元的。共用 AppID 会让同一人在两端拿到同一个 openid，`FindByOpenID` 只命中一条 User → 另一端角色丢失 → 接口全 403。
>
> **老师端是个人号的两个后果**：`teacher/src/pages/webview/index.vue` 的 `<web-view>` 会白屏（个人号不支持，**已划出验收范围**）；`code2session` 不返回 `unionid`，跨端同人只能靠手机号关联。
>
> **跨端同人是两条 User + 关联**（不是一条 User 多角色）：unionid 优先、手机号兜底，见 `server/internal/service/cross_end_identity.go`。
>
> 微信侧调用各自选端：`GetPhoneNumber`（家长绑定手机号）走家长端、`GetWXACodeUnlimit`（教师邀请码二维码）走老师端、`GetAccessToken` 的 Redis 缓存键带 appid。改 AppID 后**必须 `npm run dev:mp-weixin` 重建**（IDE 读 `dist/dev/mp-weixin/project.config.json`，不是 `src/manifest.json`）。

**Auth flow (admin)**: `POST /api/v1/auth/login/password` with username/password → JWT access + refresh tokens. Tokens stored in cookies (`admin_token`, `admin_refresh`).

**Casbin RBAC**: Enforcer checks `(role, route_path, http_method)` for every authenticated request. `CasbinRBAC` middleware iterates user's roles and allows if any role matches. Admin-specific RBAC (roles/permissions) also stored in DB (`admin_rbac.go`).

**Wire DI**: After adding a new repository/service/handler, run `make wire` in `server/internal/app` to regenerate `wire_gen.go`. The handler struct aggregates all services; a new module requires updates to handler struct, constructor, provider set, and router registration.

> ⚠️ **Wire 不会隐式把具体类型匹配到接口。** service 层为可测试性把依赖声明成接口（`ClassRepository`、`GrowthRepository`、`OpLogger`、`TeacherLookup`），每新增一个都必须在上面的 `wire.go` 里补一条 `wire.Bind`，否则报 `no provider found for <iface>`。
>
> ⚠️ **wire 二进制必须用 ≥ 模块 go 指令的版本构建。** 本仓库要求 go 1.25；用旧工具链构建的 wire 会直接拒绝解析源码，报 `package requires newer Go version go1.25` —— 这个错误看起来像依赖问题，其实不是。重建：
>
> ```bash
> GOTOOLCHAIN=go1.25.0 go install github.com/google/wire/cmd/wire@latest
> ```

### API route structure (`/api/v1`)

- `POST /auth/login/wechat|password|sms`, `POST /auth/refresh`, `POST /auth/logout`
- `GET|PUT /common/user/profile`, `POST /common/upload`
- `GET|PUT /common/messages`, `/common/messages/unread-count`
- `/admin/*` — 22 个资源组：dashboard, classes, students, teachers, parents, settings, kindergarten, admins, roles, attendance, leaves, pickup, growth, tasks, points, achievements, notices, announcements, audit, export, health, dicts；另有单条路由 `GET /admin/logs`（操作日志，不构成组）
- `/teacher/*` — 10 个资源组：home, notices, leaves, growth, pickup, calendar, class, messages, tasks, dicts
- `/parent/*` — 9 个资源组：home, notices, leaves, growth, pickup, tasks, points, class, student
- `/doctor/*` — 1 个资源组：health（晨检/保健医）

> 四端路由分组见 `server/internal/app/router.go`。新增端或分组时必须同步此列表与 Casbin policy。

### Data model layers

> 概念分层引自 `backend/model/README.md`（该文件已停更，**具体 struct 以 `server/internal/domain/` 为准**，共 58 个 struct / 29 个文件）。

1. **Organization & People**: Kindergarten, Class, User (+ UserRole), Student, StudentParent, ClassTeacher
2. **Notices & Communication**: Notice, NoticeReadLog, NoticeTemplate, LeaveRequest, Announcement
3. **Attendance & Pickup**: Attendance, PickupRecord, PickupAuthorization, PickupQRCode
4. **Growth Records**: GrowthDailyLog, GrowthMedia, GrowthComment, GrowthLike, GrowthReport
5. **Tasks & Points**: ParentTask, TaskCheckIn, PointAccount, PointTransaction, Achievement
6. **Admin-only**: ContentAudit, AdminRole, AdminPermission, OperationLog

`server/internal/domain/` 中另有上述分层未涵盖的域：`academic_year`、`calendar`（班级日历）、`daily_summary`、`dict`、`export_task`、`feedback`、`health_check`、`message`、`teacher_invitation`、`teacher_message`、`kindergarten_app_setting`；`base.go` / `file.go` 为共享基础模型（`BaseModel` / `SoftDeleteModel` / 文件表），不属于业务域。

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

- `src/pages.json` — uni-app page routing (**20 pages**, 无 tabBar):

  | 模块 | 页面 |
  | --------- | ------------------------------------------------------ |
  | 工作台    | `index/index`                                          |
  | 登录      | `auth/login`                                           |
  | 通知      | `notice/create`                                        |
  | 成长记录  | `growth/index`、`growth/create`、`growth/detail`       |
  | 接送      | `pickup/index`、`pickup/records`                       |
  | 亲子任务  | `tasks/index`                                          |
  | 班级日历  | `calendar/index`、`calendar/create`、`calendar/detail` |
  | 请假      | `leave/index`                                          |
  | 班级      | `class/index`                                          |
  | 相册      | `album/index`                                          |
  | 消息      | `messages/index`                                       |
  | 我的/更多 | `profile/index`、`more/index`                          |
  | 晨检      | `doctor/checkin`                                       |
  | 内嵌网页  | `webview/index`                                        |

- `src/utils/login.js` — 导出 `silentLogin` / `bindPhoneLogin` / `ensureLogin`；`ensureLogin()` 调 `uni.login()` 拿 code，转交 `src/api/auth.js` 的 `wxCodeLogin`（`role` 在此处硬编码为 `"teacher"`）
- `src/utils/auth.js` — localStorage-based token storage (uni.setStorageSync)
- `src/api/` — per-module API wrappers using a shared `http.js` (uni.request interceptor with auto token refresh)
- Login is silent at app launch; `App.vue` `onLaunch` calls `ensureLogin()`

### Parent mini-program structure (`parent/`)

与老师端同构（同版本 uni-app、同样的 `utils/`+`api/` 分层与静默登录），差异如下：

- `src/pages.json` — **26 pages，有 tabBar**（老师端没有）
- tabBar 4 项：`index/index` 首页、`growth/index` 成长时光、`pickup/index` 接送、`profile/index` 我的
- 页面模块：首页、登录/绑定（`auth/login`、`auth/bind`）、通知（列表+详情）、请假（列表+新建+详情）、成长（列表+详情+相册+**报告**）、接送（+**二维码**、**授权人管理**）、亲子任务（列表+详情）、**积分**、班级、日历、**幼儿档案**、**反馈**、设置、消息、我的、webview
- 相对老师端多出的能力：绑定关系、成长报告、接送二维码与授权人、积分、反馈
- 登录同样静默：`App.vue` `onLaunch` → `ensureLogin()`，`role` 在 `src/api/auth.js` 硬编码为 `"parent"`

> 两端登录逻辑各自独立实现（各有一份 `utils/login.js`，导出签名一致），**不是共享包**。修改登录流程时两端都要改。

## ⚠️ Known Issues（已知问题，动手前先看）

1. **阿里云 RDS 凭据已随 git 历史泄漏（源码已修，历史未清）** — 初始提交 `1611c61` 中，`server/cmd/initcheck/main.go` 硬编码了 RDS 连接串（账号 `test24yangshi` + 明文密码 + 实例地址 `rm-bp1w012d612w344y7po`）。
   - ✅ **源码已修复**：改为 `config.Load()` 读 `configs/app.ini`，口令走 `INIT_ADMIN_PASSWORD` 环境变量。
   - ❌ **仍未处理**：该凭据**依然存在于 git 历史中**（`git log -S` 可查到），克隆仓库即可读到。
   - **待办（需人工执行）**：
     1. **轮换该 RDS 密码** —— 这是唯一真正有效的补救，历史清理只是辅助
     2. 清理历史：`git filter-repo` 或 BFG，之后所有协作者必须重新克隆
     3. 检查仓库托管方是否有该仓库的公开/共享访问
   - 在密码轮换前，**不要把该仓库公开或分享给外部**。

2. **`backend/model/` 已停更，与 `server/internal/domain/` 漂移** — `backend/model/` 18 个 .go 文件全部停在 2026-03-22；`server/internal/domain/` 有 29 个文件 / 58 个 struct，是唯一实现。`backend/api/` 则相反（最近更新 2026-06-02），仍是接口契约的 source of truth。**写 model 看 `server/`，写 API 看 `backend/api/`。**

3. **`server/cmd/` 下多为一次性调试脚本** — 见「辅助命令」一节的警告，`refreshtasks` 具有破坏性。

4. **`.gitignore` 的 `*.exe` 匹配不到 Delve 调试产物** — `__debug_bin.exe<数字>` 的扩展名是 `.exe<数字>`，不匹配 `*.exe`，导致 3 个调试二进制被提交进仓库。修复方式见 `server/.gitignore`。

## 🎨 Design Token 规范（三端必须遵守）

养成星球使用统一的 design token 系统管理三端样式。**所有颜色、圆角、阴影、间距、字号必须通过 token 引用，禁止硬编码。**

单一数据源: `shared-design-tokens/tokens.json` → 构建产出三端各自格式的文件。

**前端设计探索与落地流程见 `docs/open-design-impeccable-playbook.md`**（OpenDesign 发散 → 交接卡 → Impeccable 收敛 → 验证归档）。

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
| 圆角 | `$border-radius-sm`(12rpx) ~ `2xl`(48rpx), `full`(19998rpx) | 共享 |
| 阴影 | `$shadow-sm`, `md`, `lg`（多层琥珀暖调） | 共享 |
| 渐变 | `$gradient-brand`, `-brand-soft`, `-page` | 共享 |
| 间距 | `$spacing-xs`(8rpx) ~ `xl`(32rpx) | 共享 |
| 字号 | `$font-size-xs`(22rpx) ~ `3xl`(40rpx) | 共享 |
| 字重 | `$font-weight-normal`(400) ~ `bold`(700) | 共享 |
| 动画 | `$animation-duration-fast`(150ms), `base`(300ms), `slow`(500ms) | 共享 |

> 📐 **单位换算**：`tokens.json` 的数值是 **px 设计稿值**，`sd.config.js` 构建时 `×2` 输出 rpx。
> 例：`borderRadius.xl = 16` → 构建产物 `$border-radius-xl: 32rpx`。**改 token 时写 px，不是 rpx。**

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
