---
name: admin-run
description: >
  Launch and drive the 养成星球 admin web (Nuxt 3) on localhost, end to end:
  ensure the Go backend is on :8080, start `npm run dev`, open the page with the
  Playwright MCP, screenshot it, and prove the /api/v1 proxy actually reaches Gin.
  Use this skill whenever the user wants to 打开后台 / 启动后台 / 跑起来 admin /
  run the admin web / start the Nuxt dev server / see the admin UI locally /
  本地打开管理后台 / open admin locally, or asks to screenshot or poke at the
  admin pages. Also use it when someone says "后台起不来", "admin 打不开", or
  "admin 页面是白的" — the gotchas below are exactly what those symptoms are made of.
  Triggers on: admin, 后台, 管理后台, Nuxt, nuxt dev, NUXT_PUBLIC_API_BASE,
  localhost:3000, admin/pages/, admin/nuxt.config.ts.
---

# Admin 后台：本地启动与驱动

管理后台是 `admin/` 下的 Nuxt 3 SPA（`ssr: false`），靠 Vite 代理把 `/api/v1`
转给 Go 后端。**「跑起来」的判据不是进程活着，是页面渲染正常 + API 链路打穿。**

## 🔴 浏览器控制只用 Playwright MCP

驱动浏览器一律走 `mcp__playwright__*`：

| 动作 | 工具 |
|---|---|
| 打开页面 | `playwright_navigate` |
| 截图 | `playwright_screenshot`（`savePng: true`） |
| 读页面文字 | `playwright_get_visible_text` |
| 看报错 | `playwright_console_logs` |
| 跑 JS / 探接口 | `playwright_evaluate` |

**不要**用 `chrome-devtools` MCP，也不要写 Playwright 脚本或建议 `chromium-cli`。
截图后**必须用 Read 工具打开那张 PNG 看一眼**——空白帧等于启动失败，别只看文件生成了。

## 启动步骤

### 1. 确认 Go 后端在 :8080

`admin/nuxt.config.ts` 的 `vite.server.proxy` 把 `/api/v1` → `http://127.0.0.1:8080`。
后端不在，后台能开但所有请求都会失败。

```bash
netstat -ano | grep -E ":8080\s" | grep LISTENING
```

已有进程在听就别重复起（`go run` 会端口冲突）。没跑就：

```bash
cd server && make run          # = CONFIG=configs/app.ini go run cmd/api/main.go
```

### 2. 确保 `admin/.env` 存在

`admin/.env` 常常只有 `.env.example` 那一份模板。补一行：

```
NUXT_PUBLIC_API_BASE=http://localhost:8080/api/v1
```

dev 下 `nuxt.config.ts` 的 Vite 代理已经覆盖了这件事，`.env` 不是必需的；
但生产/preview 场景要靠它，照 CLAUDE.md 补上无害。

### 3. 起 dev server

```bash
cd admin && npm run dev
```

⚠️ **`predev` 会先跑 `cd ../shared-design-tokens && npm run build`**（构建 design
token），首次启动比预期慢，别误判成卡死。放到后台跑，然后等端口。

⚠️ **Nuxt 只绑 `[::1]:3000`（仅 IPv6 loopback）**。所以：

- 浏览器用 `http://localhost:3000` ✅
- 直接填 `http://127.0.0.1:3000` ❌ 可能连不上
- 探端口别 `grep "0.0.0.0:3000"`，匹配不到。用：

```powershell
Get-NetTCPConnection -State Listen | Where-Object { $_.LocalPort -eq 3000 }
# 或 netstat -ano | grep "\[::1\]:3000"
```

### 4. 启动日志里的两类噪音（都不是故障）

刷这两条不影响运行，页面照常渲染，别去修：

```
Pre-transform error: Failed to resolve import "#app-manifest"
```
→ Nuxt 的 vite 预变换告警，对应 `node_modules/nuxt/dist/app/composables/manifest.js`
  里的 `if (false)` 死分支。**页面是好的。**

```
WARN  Browserslist: browsers data (caniuse-lite) is 7 months old
```
→ 与功能无关。想清掉就 `npx update-browserslist-db@latest`。

## 验证（两件事都要做）

### A. 页面渲染 —— Playwright + 看图

```
playwright_navigate  →  http://localhost:3000/   (width 1280, height 800)
playwright_screenshot → savePng: true
Read 那张 PNG
```

登录页的正常形态：卡片标题「管理后台登录」，「手机号」「密码」两个必填项
（红 `*`），蓝色「登录」按钮，浅蓝灰底。

**空白帧 = 启动失败**，哪怕导航返回成功也要当失败处理。

### B. API 链路 —— 必须打穿到 Gin

只看进程活着是不够的，那是「typechecking with extra steps」。要证明请求真的
穿过 Vite 代理到了后端：

```bash
curl -sS -X POST http://localhost:3000/api/v1/auth/login/password \
  -H "Content-Type: application/json" -d '{}'
```

预期：

```
HTTP 422
{"code":42200,"message":"Key: 'passwordLoginReq.Phone' ... 'required' tag",
 "data":null,"request_id":"e4d6b6a3-..."}
```

**判据是响应体里有 `request_id`** ——那是后端 RequestID 中间件打的，能证明请求
真的到了 Gin，而不是被静态页或 Vite 吞掉。`code` 是后端统一包络（`pkg/response`）。

若返回 502 / ECONNREFUSED / HTML，说明代理链断了：先查 8080 是否还在。

## 停止

`npm run dev` 是 watch 进程，`TaskStop` 只终止 shell 包装，vite 会残留。按 PID 杀：

```powershell
Get-NetTCPConnection -State Listen |
  Where-Object { $_.LocalPort -eq 3000 } |
  ForEach-Object { Stop-Process -Id $_.OwningProcess -Force }
```

后端（8080）如果是你自己起的才动；**先看是谁的进程**，别误杀别的服务：

```powershell
Get-CimInstance Win32_Process -Filter "ProcessId=<pid>" |
  Select-Object ProcessId, Name, CommandLine
```

## 常见症状对照

| 症状 | 原因 | 处置 |
|---|---|---|
| 浏览器连 `127.0.0.1:3000` 连不上 | Nuxt 只绑 `[::1]` | 改用 `localhost:3000` |
| 页面白 / 截图空白帧 | 启动没完成，或 `predev` 的 token 构建还在跑 | 等端口 LISTENING 再导航 |
| 登录后接口全失败 | 8080 后端没起 | `cd server && make run` |
| 端口探测说没起，但浏览器能开 | 用 `0.0.0.0:3000` 去 grep 了 | 匹配 `[::1]:3000` |
| 日志刷 `#app-manifest` | Nuxt 已知噪音 | **忽略** |
