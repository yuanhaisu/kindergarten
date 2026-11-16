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
- **Language**: Go / TypeScript / Vue 3 / SCSS
- **Patterns**: Prefer composition over inheritance. Keep functions small and pure.
- **Comments**: Write why, not what. Avoid obvious comments.

## Project Overview

**养成星球 (Nurture Planet)** — a kindergarten digital management platform. Three client apps served by one Go backend, with `kindergarten_id`-based multi-tenant isolation throughout.

| App                  | Directory  | Tech                                    | Status |
| -------------------- | ---------- | --------------------------------------- | ------ |
| Teacher mini-program | `teacher/` | uni-app + Vue 3 + Vite (WeChat)         | Active |
| Admin web            | `admin/`   | Nuxt 3 + Element Plus + Pinia + ECharts | Active |
| Parent mini-program  | `parent/`  | uni-app + Vue 3 + Vite (WeChat)         | Active |
| Official website     | `website/` | Nuxt 3 SSG + design tokens              | Active |
| Backend server       | `server/`  | Go + Gin + GORM + Redis + Casbin        | Active |

API contracts live in `backend/api/` (spec-first). Data models live in `server/internal/domain/`（**不是** `backend/model/`，见下方说明）. Design data JSON files live in `design-data/`.

> `backend/` 下**两个子目录状态不同**，别看错：
> - `backend/api/`（`api-admin/`、`api-teacher/`、`api-parent/`、`api-doctor/`、`api-platform/`、`api-common.md`）— **仍在维护**，是接口契约的 source of truth，改接口先改这里。
> - `backend/model/`（18 个 .go 文件，全部停留在 2026-03，最新 2026-03-22）— **早期模型草案，已停更**，被 `server/internal/domain/`（42 个模型文件 + 1 个测试 / 108 个 struct）取代。**以 `server/internal/domain/` 为准**。该目录的 `README.md` 本身也已漂移（例如它列出 `leave.go`，但该文件并不存在）。

## Spec Workflow (功能规格化)

**规格化工具：OpenSpec**（`openspec/` + `/opsx:*` 命令）。曾评估过 GitHub Spec Kit，结论是不引入——理由与翻盘条件见下方「为什么不是 Spec Kit」。

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
make run               # Start dev server (CONFIG_PATH=$(CONFIG) go run cmd/api/main.go，CONFIG 默认 configs/app.ini)
make build             # Build binary to bin/
make wire              # Regenerate Wire DI code (cd internal/app && wire)
make test              # Run all tests with race detector
make test-cover        # Run tests + coverage HTML report
make lint              # golangci-lint run ./...
make tidy              # go mod tidy
make check             # vet + lint + test
```

Config file path: `CONFIG=configs/app.local.ini make run`（Makefile 的 `CONFIG` 变量会转成 `CONFIG_PATH` 传给进程；直接 `go run` 时用 `CONFIG_PATH`）。 Sensitive values can be overridden via env vars: `MYSQL_DSN`, `REDIS_ADDR`, `JWT_SECRET`, `WECHAT_TEACHER_APP_ID`, `WECHAT_TEACHER_SECRET`, `WECHAT_PARENT_APP_ID`, `WECHAT_PARENT_SECRET`, `OSS_*`, etc.（旧的 `WECHAT_APP_ID`/`WECHAT_SECRET` 仍被识别，仅用于回滚到共用 AppID）

### 数据库迁移（`server/`）

```bash
make migrate           # goose up（连接 configs/app.ini）
make seedbase          # 幂等初始化字典 / 膳食 / 园长角色
go run ./cmd/migrate status
go run ./cmd/migrate -database <db> up
```

- `server/migrations/00001_baseline.sql` 由 `cmd/schemadump` 从**实际开发库**导出，**不要手改**。
- 任何 schema 变更都要新增 `server/migrations/0000N_xxx.sql`（goose `-- +goose Up` / `-- +goose Down`）；生产不再依赖 `AutoMigrate`。
- 新环境首次部署顺序：`make migrate` → `make seedbase`。
- `cmd/dbcheck` 仍是迁移后核对索引与行数的只读工具。

### Admin web (`admin/`)

```bash
npm run dev            # Start dev server with HMR (Vite proxies /api/v1 to :8080)
npm run build          # Production build
npm run preview        # Preview production build
```

### Official website (`website/`)

```bash
npm run dev            # http://localhost:3000
npm run generate       # 静态产物 → .output/public，由 nginx 托管
npm run preview        # 预览构建产物
```

- 纯静态营销站，只面向民办园园长；没有后端调用。
- 文案集中在 `website/content/site.ts`；改文案不动模板。
- 颜色/圆角/阴影/字号走 `shared-design-tokens`（Ink / Amber / Cream 已登记为 `shared.color.identity`）。
- 首屏构图是「手机墙」：老师端、家长端、后台三块界面并置；唯一交互动效是一日五时刻的滚动切换。
- ICP 号待备案后补；主体已是「湘潭市岳塘区稚序软件开发工作室」；联系邮箱已是 `contact@nurtureplanet.cn`；企业微信二维码暂缓（认证费 300 元），当前以邮件为唯一联系方式。

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

### Parent mini-program (`parent/`)

与老师端同构，命令相同；构建产物在 `parent/dist/dev/mp-weixin`：

```bash
npm run dev:mp-weixin
npm run build:mp-weixin
npm run build:mp-weixin:test
```

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

**1. `success: true` 不代表"验证的是我想要的那个项目"。** MCP 只保留一个 `wechat-devtools` server，它**不绑定任何一端**——目标端完全由 `auto_port` / `cdp_port` / `project_path` 决定。同一个 server 会出现：

- 用同一个 `auto_port` 连续调用，`page_data` 返回**逐字节相同**的数据 —— 说明在驱动同一个会话
- 端口传错时，`wechat_navigate` 能"成功"导航到只在另一端存在的 `pages/more/index`

**判别是否真的双端**：每次都显式传端口（家长端 9420/9222，老师端 9421/9223）+ 探该端**独有**页面 + 比对 `page_data` 业务字段是否不同。详见下方「🔀 双端同时验证」。

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
| 单 IDE 双项目窗口 | ✅ | ⚠️ 截图可按 `--target-id` 精确指定；导航与交互仍串行 | 逐端截图对照 |
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

> 只有**一个** `wechat-devtools` server，它不绑定端。单 server 的默认 `WECHAT_PROJECT_PATH` 是老师端：家长端调用 `wechat_file` / `wechat_build` / `wechat_automator(start)` / `wechat_navigate` 时必须显式传 `project_path=parent/dist/dev/mp-weixin`；`wechat_screenshot` 要落到家长端目录时传 `output_path`。

⚠️ **CLI 端口发现写死读** `%LOCALAPPDATA%\微信开发者工具\User Data\<hash>\Default\.ide`，**不认 `--user-data-dir`**。要用 `cli.bat` 驱动实例 B，先改写该文件内容为 `44500`（`dual-ide.ps1` 的 `Set-IdePort` 已封装）。`.ide` 切换只在 `cli auto`/`open` 等启动期调用时需要；automator 走 ws 直连，不受影响。

⚠️ **残留 `cli login` 进程会占 3799 端口并弄死后启动的实例**。启动前先 `-Kill`，它会连带清掉 `node.exe ...cli...login` 残留。

**验证双实例确实各自独立**（不要只看 `success:true`）：

- 两端 `page_data` 返回的字段应明显不同（老师端有「发布通知/成长记录」，家长端有「我要请假/接送管理」）
- 探独有页面：老师端 `pages/more/index`，家长端 `pages/points/index`

#### 单 IDE 双项目窗口（可按 target 精确截图）

同一 IDE 实例可同时打开两个项目窗口（窗口标题分别显示「幼儿园老师端」「幼儿园家长端」）。

**2026-11-04 实测**：两个项目会各占一组 `__pageframe__` target，**共用同一个 CDP 端口**（默认 9222）。因此「一次只能驱动一个」只对**导航与交互**成立，**截图和读 DOM 可以按 `--target-id` 精确指定**：

```bash
# 1. 列出全部 pageframe（同一项目的 pageframe 共享一个端口前缀，用它区分端）
python -c "import json,urllib.request; [print(t['id'], t['url']) for t in json.load(urllib.request.urlopen('http://127.0.0.1:9222/json/list')) if '__pageframe__' in t['url']]"

# 2. 按 id 截图（不要用 --title 猜端口）
python scripts\shot.py out.png --port 9222 --target-id <id>
```

判别归属：看 `url` 里的端口前缀（如 `64495/...` 与 `56245/...`），或看页面文案。实测中**同一项目的所有 pageframe 共用一个内部端口**，不同项目用不同端口。

⚠️ `shot.py --title 老师端` 会按窗口标题猜端口，双窗口下容易猜错并报 `ConnectionRefused`——**始终显式传 `--port` 与 `--target-id`**。

⚠️ 导航仍会串：`cdp.py nav` 只会命中它找到的那个 route 的 pageframe，无法指定端。要驱动指定端，用双实例方案。

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
  middleware/             # Recovery/CORS/RequestID/Logger → AuthJWT → 园所状态 → CasbinRBAC → ModuleGate → consent（按端）→ RequirePermission（admin）
  rbac/                  # Casbin enforcer init
  domain/                # GORM entity models (108 structs / 42 files)
  repository/            # Data access layer (one repo file per domain)
  service/               # Business logic layer (one service file per domain)
  handler/http/          # HTTP handlers: Handler struct aggregates all *Service fields
  ai/face/               # 腾讯云人脸识别（照片点名 / 底库）
  storage/oss/           # Alibaba Cloud OSS (presigned URLs, upload)
pkg/
  log/                   # slog init + context propagation
  response/              # Unified JSON envelope (code/message/data/request_id)
  pagination/            # Page/PageSize helpers
```

**辅助命令（`cmd/` 下的独立 `main.go`）**：不进 HTTP 服务，用 `go run ./cmd/<name>` 单独执行。均读取 `configs/app.ini`（可用 `CONFIG_PATH` / `MYSQL_DSN` 覆盖）。

| 命令 | 作用 | 性质 |
| ------------ | ---------------------------------------------- | ---- |
| `chkgrowth`  | 打印 kindergarten 4 的成长记录总数与最近 3 条 | 只读 |
| `dbcheck`    | 打印指定表的索引定义与行数，用于核对迁移是否生效 | 只读 |
| `fixpwd`     | 把两个硬编码演示手机号（13900010001/10002）的密码重置为 `Parent@123` | 写入 |
| `linkwx`     | 打印上述两个硬编码手机号的 openid 绑定情况 | 只读 |
| `openid`     | 打印最近 5 个用户 openid，并把最新 openid 绑给上述两个演示家长账号 | 写入 |
| `seedgrowth` | 灌入成长记录种子数据 | 写入 |
| `seedattendance` | 灌入出勤演示数据（3 个工作日 × 全体在读学生） | 写入 |
| `seedleaves` | 灌入 6 条请假演示数据 | 写入 |
| `seedpickup` | 灌入当日接送记录 + 待审核授权人 | 写入 |
| `refreshtasks` | **删除** kindergarten 4 的全部 `ParentTask` / `TaskCheckIn` 后重建种子任务 | ⚠️ **破坏性** |
| `initcheck`  | 创建测试园所+管理员并打印计数，随后 **`Unscoped().Where("1=1")` 清空 `users` / `user_roles` / `kindergartens`** | ⚠️ **破坏性** |
| `mktoken`    | 为指定手机号+角色签发本地 JWT（验证辅助） | 只读/测试 |
| `setadmin` / `setpwd` | `setadmin` 按 `-phone/-kg` 供给租户管理员（园所须已存在），口令从 `-password` 或 `SETADMIN_PASSWORD` 读取，**无默认口令**；`setpwd` 重置硬编码用户 id 61/65 | 写入 |
| `platformadmin` | 创建或重置首个平台方账号 | 写入 |
| `redactlogs` | 回填脱敏 `operation_logs.detail` 中的手机号/身份证 | 写入 |
| `albumretention` | **硬删**软删已满 180 天的相册及关联（支持 `-dry-run`） | ⚠️ **破坏性** |
| `dietfood`   | 导入食物成分 CSV 到平台预置食物库（kg=0） | 写入 |
| `facecheck`  | 腾讯云人脸识别接入验证与质量实测 | 读取/诊断 |
| `seedgrowthfeed` | 灌入家长端成长 feed 验证所需最小数据 | 写入 |
| `seedqrcode` | 补全接送二维码链路种子数据 | 写入 |

> `migrate` / `seedbase` / `schemadump` 见上方「数据库迁移」一节，此处不重复。

> ✅ **三个 `seed*` 命令的由来**：出勤 / 请假 / 接送的演示数据原先由 service 层的
> `ensureBootstrap*` 函数在**读取路径**上按需生成。那意味着打开一次后台看板或列表页
> 就会往生产表写入编造数据。现已全部移出，只能由这些命令显式触发。
> **新增种子数据时请循此例，不要再往读取路径挂写操作。**
>
> 三者共用 `cmd/internal/seedutil`（配置、连库、园所、时区），
> 园所默认 4，可用 `SEED_ALLOW_DEMO=1 SEED_KG_ID=7 go run ./cmd/seedleaves` 覆盖；均幂等。

> 🔒 **演示/破坏性命令必须显式 opt-in**（`server/cmd/internal/seedutil`）：
> `seedgrowth` / `seedgrowthfeed` / `seedattendance` / `seedleaves` / `seedpickup` / `seedqrcode` 需要 `SEED_ALLOW_DEMO=1`；
> `refreshtasks` / `initcheck` 还需 `SEED_ALLOW_DESTRUCTIVE=1`（`seedgrowthfeed -reset` 同）。
> 拒绝时打印脱敏后的目标库 `host:port/db`。所有 `cmd` 命令统一按 `CONFIG_PATH`（未设置回落 `configs/app.ini`）解析配置。

> ⚠️ 表中 `chkgrowth` / `fixpwd` / `linkwx` / `openid` / `refreshtasks` 及多数 `seed*` 是**针对特定 kindergarten（id=4）或硬编码手机号的一次性调试脚本**，不是通用工具：幼儿园 ID、班级 ID、手机号、任务文案全部写死。**不要在生产库上执行。**
>
> `migrate` / `seedbase` / `schemadump` / `setadmin` / `platformadmin` / `albumretention` / `mktoken` 是环境级工具，可按需复用；仍需遵守各自的权限与备份要求。
>
> ⚠️ 破坏性命令：`refreshtasks` 清空该园全部任务；`initcheck` 更彻底——`Unscoped().Where("1=1")` 会**无视软删除、跨租户清空全部用户、角色与园所**；`albumretention` 会**硬删**到期相册及关联。前两者只适用于一次性测试库自检。
>
> `initcheck` 的连接串已改为读 `CONFIG_PATH`（默认 `configs/app.ini`，可用 `MYSQL_DSN` 覆盖），管理员口令改为读环境变量 `INIT_ADMIN_PASSWORD`（未设置则直接退出）。**不要再把凭据写回源码**——即使后续删除，git 历史仍可追溯。

> 🔧 **`dbcheck` 的用途**：schema 由 **goose** 版本化迁移管理（`server/migrations/`），应用启动不再执行 `AutoMigrate`。手工核对某张表的索引与行数：
>
> ```bash
> go run ./cmd/dbcheck task_check_ins
> ```
>
> 会打印该表的全部索引（标注唯一/普通、列出列序）与行数。跑完 `make migrate` 或手工 SQL 后，用它确认线上库真的变了，而不是假设迁移会按预期执行。

**Middleware chain**: `gin.Recovery → CORS → RequestID → RequestContextLogger`；authed 组：`AuthJWT → RequireActiveKindergarten → CasbinRBAC → ModuleGate`，再按端叠加 `RequireConsent`（admin/teacher/parent）与 `RequirePermission`（仅 admin）；`/platform` 走 `PlatformAuth → CasbinRBAC`

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
> **老师端是个人号的两个后果**：个人号不支持 `<web-view>`（原 `webview/index` 占位页已删除，老师端不再保留内嵌网页入口）；`code2session` 不返回 `unionid`，跨端同人只能靠手机号关联。
>
> **跨端同人是两条 User + 关联**（不是一条 User 多角色）：unionid 优先、手机号兜底，见 `server/internal/service/cross_end_identity.go`。
>
> 微信侧调用各自选端：`GetPhoneNumber`（家长绑定手机号）走家长端、`GetWXACodeUnlimit`（教师邀请码二维码）走老师端、`GetAccessToken` 的 Redis 缓存键带 appid。改 AppID 后**必须 `npm run dev:mp-weixin` 重建**（IDE 读 `dist/dev/mp-weixin/project.config.json`，不是 `src/manifest.json`）。

**Auth flow (admin)**: `POST /api/v1/auth/login/password` with username/password → JWT access + refresh tokens. Tokens stored in cookies (`admin_token`, `admin_refresh`).

**Casbin RBAC**: Enforcer checks `(role, route_path, http_method)` for every authenticated request. `CasbinRBAC` middleware iterates user's roles and allows if any role matches. Admin-specific RBAC (roles/permissions) also stored in DB（见 `service/admin_authz_service.go`、`role_service.go`、`role_catalog.go`）。

**Wire DI**: After adding a new repository/service/handler, run `make wire` in `server/` (it runs `cd internal/app && wire`) to regenerate `wire_gen.go`. The handler struct aggregates all services; a new module requires updates to handler struct, constructor, provider set, and router registration.

> ⚠️ **Wire 不会隐式把具体类型匹配到接口。** service 层为可测试性把依赖声明成接口（`ClassRepository`、`GrowthRepository`、`OpLogger`、`TeacherLookup`），每新增一个都必须在上面的 `wire.go` 里补一条 `wire.Bind`，否则报 `no provider found for <iface>`。
>
> ⚠️ **wire 二进制必须用 ≥ 模块 go 指令的版本构建。** 本仓库要求 go 1.25；用旧工具链构建的 wire 会直接拒绝解析源码，报 `package requires newer Go version go1.25` —— 这个错误看起来像依赖问题，其实不是。重建：
>
> ```bash
> GOTOOLCHAIN=go1.25.0 go install github.com/google/wire/cmd/wire@latest
> ```

### API route structure (`/api/v1`)

- `POST /auth/login/wechat|password|sms`, `POST /auth/sms/send`, `POST /auth/refresh`, `POST /auth/logout`
- `POST /platform/auth/login`；`GET /system/status`、`POST /system/init`（初始化/健康自检）
- `GET|PUT /common/user/profile`, `POST /common/upload`
- `GET /common/messages`、`GET /common/messages/unread-count`、`PUT /common/messages/:id/read`、`PUT /common/messages/read-all`
- `/admin/*` — 28 个资源组：billing, dashboard, classes, students, teachers, parents, settings, kindergarten, admins, roles, attendance, leaves, pickup, growth, albums, tasks, task-activities, points, achievements, notices, announcements, audit, export, health, calendar, timetable, diet, dicts；另有单条路由 `GET /admin/logs`（操作日志，不构成组）
- `/teacher/*` — 12 个资源组：home, notices, leaves, growth, achievements, pickup, calendar, class, albums, messages, tasks, dicts
- `/parent/*` — 11 个资源组：home, notices, leaves, growth, pickup, tasks, points, class, class-albums, student, health
- `/doctor/*` — 2 个资源组：calendar, health（晨检/保健医）
- `/platform/*` — 24 条路由（园所入驻 / 套餐 / 订单 / 分销 / 佣金），独立信任域：`PlatformAuth` + `CasbinRBAC`，凭证 `scope=platform`，不走 JWT / 园所门禁（ADR-0014）

> 五个路由分组（admin / teacher / parent / doctor / platform）见 `server/internal/app/router.go`。新增端或分组时必须同步此列表与 Casbin policy。

### Data model layers

> 概念分层引自 `backend/model/README.md`（该文件已停更，**具体 struct 以 `server/internal/domain/` 为准**，共 108 个 struct / 42 个模型文件）。

1. **Organization & People**: Kindergarten, Class, User (+ UserRole), Student, StudentParent, ClassTeacher
2. **Notices & Communication**: Notice, NoticeReadLog, NoticeTemplate, LeaveRequest, Announcement
3. **Attendance & Pickup**: Attendance, PickupRecord, PickupAuthorization, PickupQRCode
4. **Growth Records**: GrowthDailyLog, GrowthMedia, GrowthComment, GrowthLike, GrowthReport
5. **Tasks & Points**: ParentTask, TaskCheckIn, PointAccount, PointTransaction, Achievement
6. **Admin-only**: ContentAudit, AdminRole, AdminPermission, OperationLog
7. **Health & Face**: MorningHealthCheck, HealthCheckMedia, StudentFace, RecognitionJob
8. **Diet & Nutrition**: DietConfigSnapshot, MealConfig, Food, FoodCategory, DriAgeTarget, DietGroupObject, MealPlan, Recipe 等 19 个 struct
9. **Albums**: Album, AlbumPhoto, AlbumStudent, AlbumPhotoStudent
10. **SaaS Commercial**: Plan, PlanModule, PlanTier, Order, Payment, Entitlement, Distributor, CommissionLedger, PlatformAdmin, PlatformOperationLog

`server/internal/domain/` 中另有上述分层未涵盖的域：`academic_year`、`calendar`（班级日历）、`consent`（隐私同意）、`daily_summary`、`dict`、`export_task`、`feedback`、`message`、`parent_setting`、`teacher_invitation`、`teacher_message`、`kindergarten_app_setting`、`audit` / `rbac`（管理端权限）；`base.go` / `file.go` 为共享基础模型（`BaseModel` / `SoftDeleteModel` / 文件表），不属于业务域。

All tenant business tables include `kindergarten_id` for multi-tenancy; platform-level tables (`platform_admins` / `distributors` / `plans` / `plan_modules` / `plan_tiers`) are intentionally global (ADR-0014). Core business data uses `SoftDeleteModel`.

### Admin web structure (`admin/`)

- SPA mode (`ssr: false`), cookie-based auth with global middleware `auth.global.ts` redirecting to `/login`
- `api/` — per-module API functions built on `useHttp` composable
- `stores/` — Pinia stores: `auth` (session), `permission` (RBAC codes, defaults to `['*']`), `app` (sidebar collapse)
- `middleware/auth.global.ts` — checks `admin_token` cookie on every route
- `composables/` — `useHttp` (Axios wrapper with auto token refresh), `useAuth`, `usePagination`, `usePermission`
- Sidebar menu defined in `constants/menus.ts` with permission codes
- Vite dev proxy forwards `/api/v1` to `http://127.0.0.1:8080`

### Teacher mini-program structure (`teacher/`)

- `src/pages.json` — uni-app page routing (**23 pages**, 无 `tabBar` 字段；工作台/班级/消息/我的 4 页用自绘底栏):

  | 模块 | 页面 |
  | --------- | ------------------------------------------------------ |
  | 工作台    | `index/index`                                          |
  | 登录/同意 | `auth/login`、`auth/consent`                           |
  | 通知      | `notice/index`、`notice/create`                        |
  | 成长记录  | `growth/index`、`growth/create`、`growth/detail`       |
  | 接送      | `pickup/index`、`pickup/records`                       |
  | 亲子任务  | `tasks/index`、`tasks/detail`                          |
  | 班级日历  | `calendar/index`、`calendar/create`、`calendar/detail` |
  | 请假      | `leave/index`                                          |
  | 班级      | `class/index`                                          |
  | 相册      | `album/index`                                          |
  | 消息      | `messages/index`                                       |
  | 我的/更多 | `profile/index`、`more/index`                          |
  | 晨检      | `doctor/checkin`、`doctor/calendar`                    |

- `src/utils/login.js` — 导出 `silentLogin` / `bindPhoneLogin` / `ensureLogin`；`ensureLogin()` 调 `uni.login()` 拿 code，转交 `src/api/auth.js` 的 `wxCodeLogin`（`role` 在此处硬编码为 `"teacher"`）
- `src/utils/auth.js` — localStorage-based token storage (uni.setStorageSync)
- `src/api/` — per-module API wrappers using a shared `http.js` (uni.request interceptor with auto token refresh)
- Login is silent at app launch; `App.vue` `onLaunch` calls `ensureLogin()`

### Parent mini-program structure (`parent/`)

与老师端同构（同版本 uni-app、同样的 `utils/`+`api/` 分层与静默登录），差异如下：

- `src/pages.json` — **31 pages，有 tabBar**（老师端没有）
- tabBar 4 项：`index/index` 首页、`growth/index` 成长时光、`pickup/index` 接送、`profile/index` 我的
- 页面模块：首页、登录/绑定/同意（`auth/login`、`auth/bind`、`auth/consent`、`auth/policy`）、通知（列表+详情）、请假（列表+新建+详情）、成长（列表+详情+相册+**报告**）、**班级相册**（`album/*`）、接送（+**二维码**、**授权人管理**）、亲子任务（列表+详情+**任务墙**）、**积分**、班级、日历、**幼儿档案**、**反馈**、设置、消息、我的
- 相对老师端多出的能力：绑定关系、成长报告、接送二维码与授权人、积分、反馈
- 登录同样静默：`App.vue` `onLaunch` → `ensureLogin()`，`role` 在 `src/api/auth.js` 硬编码为 `"parent"`

> 两端登录逻辑各自独立实现（各有一份 `utils/login.js`，导出签名一致），**不是共享包**。修改登录流程时两端都要改。

### Shared frontend assets（teacher / parent）

- `shared-ui/` — 两端共用组件的**唯一真相源**；两端的 `src/components/` 由 `scripts/sync-ui.js` 生成，**禁止手改**（`predev` / `prebuild` 自动同步）。
- `shared-design-tokens/` — 三端样式 token 唯一真相源，详见下方「Design Token 规范」。
- `website/` 复用 `kindergarten-design-tokens/tokens-admin.css`（`predev` / `prebuild` 会先构建 tokens）。

## ⚠️ Known Issues（已知问题，动手前先看）

1. **阿里云 RDS 凭据已随 git 历史泄漏（源码已修，历史未清）** — 初始提交 `1611c61` 中，`server/cmd/initcheck/main.go` 硬编码了 RDS 连接串（账号、明文密码与实例地址均已从当前文档中打码；具体值只存在于 git 历史与既有泄漏记录中）。
   - ✅ **源码已修复**：改为 `config.Load()` 读 `configs/app.ini`，口令走 `INIT_ADMIN_PASSWORD` 环境变量。
   - ❌ **仍未处理**：该凭据**依然存在于 git 历史中**（`git log -S` 可查到），克隆仓库即可读到。
   - **待办（需人工执行）**：
     1. **轮换该 RDS 密码** —— 这是唯一真正有效的补救，历史清理只是辅助
     2. 清理历史：`git filter-repo` 或 BFG，之后所有协作者必须重新克隆
     3. 检查仓库托管方是否有该仓库的公开/共享访问
   - 在密码轮换前，**不要把该仓库公开或分享给外部**。

2. **`backend/model/` 已停更，与 `server/internal/domain/` 漂移** — `backend/model/` 18 个 .go 文件全部停在 2026-03（最新 2026-03-22）；`server/internal/domain/` 有 42 个模型文件 / 108 个 struct，是唯一实现。`backend/api/` 则相反（最近更新 2026-11-15），仍是接口契约的 source of truth。**写 model 看 `server/`，写 API 看 `backend/api/`。**

3. **`server/cmd/` 下多为一次性调试脚本** — 见「辅助命令」一节的警告，`refreshtasks` 具有破坏性。

4. **`.gitignore` 的 `*.exe` 匹配不到 Delve 调试产物** — `__debug_bin.exe<数字>` 的扩展名是 `.exe<数字>`，不匹配 `*.exe`，曾导致 3 个调试二进制被提交进仓库。✅ 已在 `server/.gitignore` 增加 `__debug_bin*`，当前无跟踪产物。

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
  font-size: $typography-font-size-base;
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
| 文字 | `$color-text-primary` / `-body` / `-muted` / `-hint` | 共享 |
| 功能色 | `$color-semantic-success`, `warning`, `danger`, `info` + 对应的 `-bg` 浅底 | 共享 |
| 圆角 | `$border-radius-sm`(8rpx) ~ `2xl`(48rpx), `full`(19998rpx) | 共享 |
| 阴影 | `$shadow-sm`, `md`, `lg`（多层琥珀暖调） | 共享 |
| 渐变 | `$gradient-brand`, `-brand-soft`, `-page` | 共享 |
| 间距 | `$spacing-xs`(8rpx) ~ `xl`(32rpx) | 共享 |
| 字号 | `$typography-font-size-xs`(22rpx) ~ `3xl`(40rpx) | 共享 |
| 字重 | `$typography-font-weight-normal`(400) ~ `bold`(700) | 共享 |
| 动画 | `$animation-duration-fast`(150ms), `base`(300ms), `slow`(500ms) | 共享 |

> 📐 **单位换算**：`tokens.json` 的数值是 **px 设计稿值**，`sd.config.js` 构建时 `×2` 输出 rpx。
> 例：`borderRadius.xl = 12` → 构建产物 `$border-radius-xl: 24rpx`。**改 token 时写 px，不是 rpx。**

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
| | `$component-badge-radius` | 19998rpx | 标签圆角（胶囊） |
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

### 设计稿同步门禁（必读）

代码是 UI 的唯一真相源；`docs/design-previews/2026-10-25-full-redesign` 是派生的视觉参考。**不要手改 `tokens.css` 或 `_shared/pages.json`**，它们由脚本生成。

改动 `parent/src/pages/**`、`teacher/src/pages/**`、`shared-design-tokens/tokens.json` 或 `shared-ui/**` 后：

```bash
npm run design:sync                                  # 重生成 tokens.css / pages.json / 画廊，并列出需复核的页面
npm run design:change -- pickup-polish "接送页三态收口"  # 设计领先的改版：生成变更单
npm run design:accept -- parent/13-pickup-index --reason "2026-11-09-pickup-polish 已实现 <commit>"
npm run design:accept -- --shared --reason "共享组件变更说明"
npm run design:shots                                 # 重出 51 页 preview.png（需要 Chrome）
npm run design:check                                 # 必须通过；CI 的 design-sync job 会跑同一检查
npm run design:test                                  # 门禁自身的结构断言（node --test）
```

- `design.lock.json` 记录每页 `.vue` 的 sha256、依赖的内容型 utils、导航标题与 tabBar 归属；源码/依赖/标题/tabBar 变了但没 `accept`，`check` 会失败并列出原因。
- 新增页面必须先在 `design.config.json` 登记 `slug / n / tabbar` 并补原型目录；`sync` 只会写“待复核”条目，必须显式 `accept`。
- 纯 token 改动由 `design:sync` 自动覆盖；`accept --all` 默认拒绝，只有确认没有任何页面内容变化或显式加 `--force` 才执行。
- `accept` 必须带 `--reason`，指向变更单或实现 commit；设计领先的改版走 `docs/design-changes/`。
- `shared-ui / utils / api / components` 变更不会阻断，但会持续告警，直到 `accept --shared`。
- `design.preview.lock.json` 记录的截图基线与原型内容不一致时，`check` 会给 warning，跑 `design:shots` 重出。
- pre-commit 只做提醒；真正的阻断在 CI。

## Key constraints

- Never commit `configs/app.local.ini`, `.env` files, or credentials in `app.ini`
- Wire-generated file `internal/app/wire_gen.go` must be regenerated (`make wire`) after DI changes
- API specs in `backend/api/` are the source of truth; handlers and services must conform to them
- Design tokens source: `shared-design-tokens/tokens.json`. Generated `build/` files must be committed alongside source changes
- 新增 `/admin` 路由必须在 `server/internal/middleware/admin_perm.go` 的 `adminRoutePermission` 登记权限码，否则启动自检直接失败（`router.go` 拒绝启动）
- 新增 premium 模块必须在 `server/internal/middleware/module_gate.go` 的 `premiumRouteRules` 命中至少一条已注册路由，否则启动失败
- `teacher/src/components/`、`parent/src/components/` **禁止手改**：唯一真相源是 `shared-ui/`，由 `scripts/sync-ui.js` 在 `predev` / `prebuild` 同步

## Agent skills

### Issue tracker

Issues and specs live as GitHub issues on `yuanhaisu/kindergarten`, driven by the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Domain docs

single-context: `GLOSSARY.md` at the repo root plus `docs/adr/`. See `docs/agents/domain.md`.

### Parallel development

多个会话并行时，一个会话 = 一个分支 = 一个独立 worktree；主工作树始终留在 `main` 且保持干净。纪律与脚本见 `docs/agents/parallel-development.md`。

### CI / 合并等待

推送 PR 后，不要在主会话空转或提前开始下一项。起一个只读 subagent 监听 CI 与合并结果：`gh pr checks <n> --watch` 等 CI 终态，`gh pr view <n> --json state,mergeStateStatus,mergedAt,statusCheckRollup` 看合并与检查汇总；收到 CI/合并终态后再提出下一项工作。subagent 不得 merge、push 或改文件。**一个 subagent 只监听一个 PR**：多个 PR 并行等待时，每个 PR 各起一个 subagent，不得用一个 subagent 同时轮询多个 PR。

**合并必须等到 `MERGED` 或 `CLOSED`，不能因为短时间没合并就结束监听。** 轮询用阶梯：0–5 分钟 15 秒，5–15 分钟 30 秒，15–60 分钟 60 秒，1–4 小时 5 分钟，4–12 小时 15 分钟；12 小时后只报告“仍未合并”并立即重新起监听。

**重起后的档位**：正常走满 12 小时而重起的，**新一轮从 15 秒重新走阶梯**；若只是单轮因工具/会话硬限制（如 subagent 运行上限、命令超时）被打断，则**续用本轮已持久化的开始时间**继续当前档位，只有开始时间丢失才回到 15 秒。起监听时把 PR 号与本轮开始时间（UTC）写进 subagent 提示，避免重起后算错档位。
