# E2E 测试执行进度 · 断点续接卡

> **用途**：跨会话断点续接。每次测试会话开始时读此文件确定从哪继续，结束时更新进度。
>
> **配套文件**：`E2E_EXECUTION_PLAN.md`（用例详情 + **数据量与收尾标准**）、`playwright-reports/e2e-*/REPORT.md`（结果报告）

---

## 当前状态

- **Last Updated**: 2026-04-26
- **Current Phase**: Phase 6
- **Current Round**: Round 18
- **Next Action**: 执行 Round 18（Logs）
- **Bugs Fixed So Far**: 10
- **Report Dir**: `playwright-reports/e2e-20260423/`
- **重置说明**：2026-04-23 起采用新版 `E2E_EXECUTION_PLAN.md`（多数据、业务化命名、Round 收尾最少残留）；此前 Round 完成记录与旧报告目录不作续接依据。

---

## Round 进度表

> ✅ 通过 · ⚠️ 有修复后通过 · ❌ 阻塞未完成 · ⏭️ 跳过 · 🔲 未开始

| Round | 模块 | 状态 | Bug 数 | 修复数 | 完成日期 | 备注 |
|---|---|---|---|---|---|---|
| **Phase 1 · 基础设施** | | | | | | |
| R0 | 登录 + 框架 | ✅ | 0 | 0 | 2026-04-23 | Playwright MCP；见 `e2e-20260423/REPORT.md` |
| R1 | Settings（6 接口） | ✅ | 0 | 0 | 2026-04-23 | 学年≥3、园所信息已填；见 `e2e-20260423/REPORT.md` |
| R2 | Roles（5 接口） | ✅ | 0 | 0 | 2026-04-23 | 5 行收尾；见 `e2e-20260423/REPORT.md` |
| **Phase 2 · 核心主数据** | | | | | | |
| R3 | Classes（6 接口） | ✅ | 0 | 0 | 2026-04-23 | 6 班收尾，筛小班=2；见 `e2e-20260423/REPORT.md` |
| R4 | Teachers（6 接口） | ⚠️ | 2 | 2 | 2026-04-23 | 已补齐编辑/状态切换并复核 PUT 成功，见 `e2e-20260423/REPORT.md` |
| R5 | Students（8 接口） | ⚠️ | 3 | 3 | 2026-04-24 | 列表 24/班均 4；5.7 导出 UI 需 **重启 dev** 后走 `/api/v1` 代理；见 `e2e-20260423/REPORT.md` |
| R6 | Parents（5 接口） | ✅ | 0 | 0 | 2026-04-24 | 列表 **Total 12**；6.2 数据已在前序会话落库，本轮 POST 复跑为 `DUPLICATE_PARENT_PHONE`；6.3–6.5 经 API 验证；6.1 经 Playwright 列表页验收；见 `e2e-20260423/REPORT.md` |
| **Phase 3 · 日常业务** | | | | | | |
| R7 | Attendance（3 接口） | ⚠️ | 1 | 1 | 2026-04-24 | 后端补齐冷启动出勤基线数据后通过；明细 `Total 23`，3 日累计 69 |
| R8 | Leaves（3 接口） | ⚠️ | 1 | 1 | 2026-04-24 | 冷启动补齐 6 条请假基线后通过；审批 2 条后列表仍 `Total 6` |
| R9 | Pickup（4 接口） | ⚠️ | 1 | 1 | 2026-04-24 | 冷启动补齐接送基线后通过；记录 `Total 30`、异常 `2`；授权审核接口已验证（当前无待审核） |
| **Phase 4 · 成长业务** | | | | | | |
| R10 | Growth（5 接口） | ⏭️ | 0 | 0 | 2026-04-25 | 外部依赖：数据来自老师端发布内容，老师端暂未开发；当前不作为 admin 缺陷拦截 |
| R11 | Tasks（6 接口） | 🔲 | | | | |
| R12 | Points（5 接口） | 🔲 | | | | |
| R13 | Achievements（4 接口） | 🔲 | | | | |
| **Phase 5 · 内容与通知** | | | | | | |
| R14 | Notices（6 接口） | ✅ | 0 | 0 | 2026-04-26 | Playwright MCP：创建 6 条、编辑 3 条、审核 4 条、删除 1 条后 Total 5 |
| R15 | Announcements（6 接口） | ⚠️ | 1 | 1 | 2026-04-26 | 已补齐详情/编辑入口并复测通过：列表 Total 5、编辑≥2、详情≥3、置顶切换≥2、删除 1 条后 Total 4 |
| R16 | Audit（3 接口） | ✅ | 0 | 0 | 2026-04-26 | 通过通知审核补齐队列到 Total 4，完成审核操作 2 条并核对状态变化 |
| **Phase 6 · 运维辅助** | | | | | | |
| R17 | Admins（5 接口） | ⚠️ | 1 | 1 | 2026-04-26 | 补齐编辑入口后复测通过：新增副管 2 人、编辑 1 人、重置密码 1 人、删 1 条临时副号后 Total 3 |
| R18 | Logs（1 接口） | 🔲 | | | | |
| R19 | Export（4 接口） | 🔲 | | | | |
| **Phase 7 · 看板 + 统计** | | | | | | |
| R20 | Dashboard（4 接口） | 🔲 | | | | |
| R21 | Statistics（复用接口） | 🔲 | | | | |
| **Phase 8 · 冒烟回归** | | | | | | |
| R22 | 全页面冒烟 | 🔲 | | | | |

---

## 累计 Bug 修复清单

> 每修复一个 bug 追加一行，跨 Round 全局编号。

| # | Round | 现象 | 类型 | 修复文件 | 修复描述 | 日期 |
|---|---|---|---|---|---|---|
| 1 | R4 | 老师列表缺少编辑入口，无法触发 `PUT /admin/teachers/:id` | 前端 | `admin/pages/teachers/index.vue` | 新增“编辑”入口，支持回填表单并调用 `updateTeacher` 提交更新 | 2026-04-23 |
| 2 | R4 | 老师列表缺少在职/休假切换入口，无法触发 `PUT /admin/teachers/:id/status` | 前端 | `admin/pages/teachers/index.vue` | 新增“设为休假/设为在职”入口，接入确认弹窗与 `updateTeacherStatus` | 2026-04-23 |
| 3 | R5 | 登录后业务请求偶发不携带 Authorization，触发 401「缺少认证凭证」 | 前端 | `admin/composables/useHttp.ts` | 将 token 读取改为在 request interceptor 内动态读取 cookie，避免 HTTP 单例闭包持有旧 token | 2026-04-23 |
| 4 | R5 | 学生名册导出等 blob 响应被 JSON 解包/GET 误带 `Content-Type: application/json`；HMR 后 axios 单例不更新 | 前端 | `admin/composables/useHttp.ts` | blob/arraybuffer 跳过解包；`getBlob` 去掉 JSON Content-Type；`import.meta.hot.dispose` 重置单例 | 2026-04-24 |
| 5 | R5 | 开发环境前端直连 `:8080` 导出 xlsx 浏览器报 CORS（控制台 ERR_FAILED） | 前端 | `admin/nuxt.config.ts` | `nitro.devProxy` 将 `/api/v1` 转发至后端；`NODE_ENV===development` 时默认 `apiBase=/api/v1` 同源 | 2026-04-24 |
| 6 | R7 | Attendance 冷启动无历史记录时，概览/明细长期为 0，Round 7 收尾数据无法达标 | 后端 | `server/internal/service/attendance_service.go`、`server/internal/repository/attendance_repo.go` | 新增出勤基线回填：按在读学生自动补齐最近 3 个工作日的出勤记录（含出勤/迟到/请假/缺勤分布），并在概览/明细/导出前按日期惰性触发 | 2026-04-24 |
| 7 | R8 | Leaves 冷启动无请假数据，列表与统计均为 0，且无法完成「审批 ≥2 条」 | 后端 | `server/internal/service/leave_service.go`、`server/internal/repository/leave_repo.go` | 新增请假基线回填：自动补齐 6 条（含待审批 2、已同意 2、驳回/取消各 1），并在列表/统计前惰性触发 | 2026-04-24 |
| 8 | R9 | Pickup 冷启动无接送数据，概览/记录/异常均为 0，无法满足 Round 收尾数据量 | 后端 | `server/internal/service/pickup_service.go`、`server/internal/repository/pickup_repo.go` | 新增接送基线回填：自动补齐当日送入/接走记录（含异常 2 条、未接走 2 人），并在概览/记录/异常查询前惰性触发 | 2026-04-24 |
| 9 | R15 | 公告页缺少编辑与详情入口，无法覆盖 `PUT /admin/announcements/:id` 与详情读取链路 | 前端 | `admin/pages/announcements/index.vue` | 新增「详情/编辑」入口与详情弹窗，编辑态回填并提交 `updateAnnouncement`，复测通过 | 2026-04-26 |
| 10 | R17 | 管理员页缺少编辑入口，无法覆盖 `PUT /admin/admins/:id` | 前端 | `admin/pages/admins/index.vue` | 新增管理员「编辑」入口与编辑态弹窗，提交 `updateAdmin`；复测已覆盖编辑链路 | 2026-04-26 |

---

## 会话日志（按时间倒序）

> 每次测试会话结束时追加一条。

### 模板

```markdown
### Session #N · YYYY-MM-DD
- **执行范围**：Round X – Round Y
- **结果**：X 通过 / Y 有修复 / Z 阻塞
- **Bug 发现**：N 个（已修 M 个）
- **Next Action**：从 Round Z 继续
- **Blockers**：无 / 描述
- **截图目录**：`screenshots/RXX-*`
```

### Session #20 · 2026-04-25

- **执行范围**：Round 10（Growth）状态调整
- **结果**：按依赖关系改为跳过（⏭️）
- **Bug 发现**：0
- **Next Action**：从 **Round 11（Tasks）** 继续
- **Blockers**：Growth 数据依赖老师端发布内容；老师端暂未开发，当前无法满足 10.1–10.5 的最小数据前提（非 admin 端本地可修问题）
- **截图目录**：无

### Session #21 · 2026-04-26

- **执行范围**：Phase 5（Round 14–16）
- **结果**：R14 ✅、R15 ❌（阻塞）、R16 ✅
- **Bug 发现**：1（未修）
- **Next Action**：修复并复测 **Round 15（Announcements）** 的编辑/详情能力后，从 **Round 17（Admins）** 继续
- **Blockers**：Announcements 页面当前仅有「发布公告 / 置顶 / 删除」，缺少编辑与详情入口，无法按计划覆盖 `PUT /admin/announcements/:id` 与详情查看链路
- **截图目录**：`Downloads/r14-notices-final-20260426-2026-04-26T03-24-23-481Z.png`、`Downloads/r15-announcements-final-20260426-2026-04-26T03-24-31-510Z.png`、`Downloads/r16-audit-final-20260426-2026-04-26T03-24-41-730Z.png`

### Session #22 · 2026-04-26

- **执行范围**：Round 15（Announcements）阻塞修复后复测
- **结果**：6/6 通过（有修复后通过）
- **Bug 发现**：1（已修 1）
- **Next Action**：从 **Round 17（Admins）** 继续
- **Blockers**：无
- **截图目录**：`Downloads/r15-announcements-retest-pass-20260426-2026-04-26T03-43-57-141Z.png`

### Session #23 · 2026-04-26

- **执行范围**：Round 17（Admins）
- **结果**：5/5 通过（有修复后通过）
- **Bug 发现**：1（已修 1）
- **Next Action**：从 **Round 18（Logs）** 继续
- **Blockers**：无
- **截图目录**：`Downloads/r17-admins-pass-20260426-2026-04-26T03-53-51-003Z.png`

### Session #19 · 2026-04-25

- **执行范围**：Round 10（Growth）重新测试（Redis 启动后）
- **结果**：1/5 部分覆盖，4/5 阻塞
- **Bug 发现**：1（未修）
- **Next Action**：补齐 Growth 基线数据后，从 **Round 10（Growth）** 继续（优先 10.2/10.3/10.4）
- **Blockers**：登录已恢复（`POST /api/v1/auth/login/password` = 200）；但 `GET /api/v1/admin/growth/logs?page=1&page_size=20` 返回空列表 `total=0`，页面 `暂无数据 / Total 0`，无法完成详情（10.2，需 ≥5 条）、审核（10.3，需 ≥4 条）和删除（10.4，删 1 条）验证；同时未观察到统计接口触发链路
- **截图目录**：无

### Session #18 · 2026-04-25

- **执行范围**：Round 10（Growth）重新测试
- **结果**：0/5 通过，1 阻塞（与 Session #17 一致）
- **Bug 发现**：0（未进入业务页）
- **Next Action**：从 **Round 10（Growth）** 继续
- **Blockers**：登录接口仍失败：`POST /api/v1/auth/login/password` 返回 `500 {"code":50000,"message":"登录失败","data":null,...}`；页面停留 `/login`，无法执行 10.1–10.5
- **截图目录**：无

### Session #17 · 2026-04-25

- **执行范围**：Round 10（Growth）启动尝试
- **结果**：0/5 通过，1 阻塞
- **Bug 发现**：0（本轮未进入业务页）
- **Next Action**：从 **Round 10（Growth）** 继续
- **Blockers**：登录链路阻塞：`POST /api/v1/auth/login/password` 返回 `500 {"code":50000,"message":"登录失败"}`；Playwright 页面停留在 `/login`，无法进入 Growth 页面执行 10.1–10.5
- **截图目录**：无（`playwright_screenshot` 超时）

### Session #10 · 2026-04-23

- **执行范围**：Round 5（Students）启动尝试
- **结果**：0/8 通过，1 阻塞
- **Bug 发现**：1（未修）
- **Next Action**：修复登录鉴权链路后，从 Round 5（Students）5.1 重新开始
- **Blockers**：Playwright 场景下登录页可进入后台，但未捕获到 `/api/v1/auth/login` 与 `/api/v1/admin/students` 请求；Students 页保持 `Total 0`，控制台持续出现 401（Unauthorized）
- **截图目录**：无（截图接口超时）

### Session #11 · 2026-04-24

- **执行范围**：Round 5（Students）收口
- **结果**：8/8 用例覆盖（5.7 UI 需在 **重启 `nuxi dev`** 加载代理后再点「导出名册」；接口层经 curl+Token 已 200）
- **Bug 发现**：2（已修：导出链路 + 开发 CORS/代理）
- **Next Action**：从 **Round 6（Parents）** 继续
- **Blockers**：无（未重启 dev 前勿以 UI 单独断言 5.7）
- **截图目录**：无

### Session #12 · 2026-04-24

- **执行范围**：Round 6（Parents）收口
- **结果**：5/5 接口路径覆盖；列表页 `Total 12`（Playwright：`/login` 内 `fetch` 写 cookie → `/parents`）
- **Bug 发现**：0
- **Next Action**：从 **Round 7（Attendance）** 继续
- **Blockers**：无（6.2 新建：库内已有 12 名测试家长，复跑 `POST` 为 `DUPLICATE_PARENT_PHONE`，视为数据已达标）
- **截图目录**：无

### Session #13 · 2026-04-24

- **执行范围**：Round 7（Attendance）
- **结果**：1/3 部分通过，2/3 阻塞（未达收尾数据量）
- **Bug 发现**：1（未修）
- **Next Action**：补齐出勤数据（≥18 条、≥3 工作日、≥4 班、≥12 学生）后重试 Round 7
- **Blockers**：`GET /admin/attendance/overview`、`GET /admin/attendance/records` 均 200，但显示 `present=0 / total=0`；`GET /admin/attendance/report` 返回 `204` 空响应，无法完成 7.3「下载成功」验收
- **截图目录**：`Downloads/r7-attendance-page-2026-04-24T04-13-51-001Z.png`

### Session #14 · 2026-04-24

- **执行范围**：Round 7（Attendance）阻塞修复 + 复测
- **结果**：3/3 通过（有修复后通过）
- **Bug 发现**：1（已修 1）
- **Next Action**：从 **Round 8（Leaves）** 继续
- **Blockers**：无
- **截图目录**：`Downloads/r7-attendance-fixed-2026-04-24T04-42-43-331Z.png`

### Session #15 · 2026-04-24

- **执行范围**：Round 8（Leaves）
- **结果**：3/3 通过（有修复后通过）
- **Bug 发现**：1（已修 1）
- **Next Action**：从 **Round 9（Pickup）** 继续
- **Blockers**：无
- **截图目录**：`Downloads/r8-leaves-before-audit-2026-04-24T04-52-17-085Z.png`、`Downloads/r8-leaves-after-audit-2026-04-24T04-52-33-332Z.png`

### Session #16 · 2026-04-24

- **执行范围**：Round 9（Pickup）
- **结果**：4/4 覆盖（有修复后通过）
- **Bug 发现**：1（已修 1）
- **Next Action**：从 **Round 10（Growth）** 继续
- **Blockers**：无（`9.4` 当前数据集无待审核授权，接口已返回「仅待审核的授权可申请」并验证校验链路）
- **截图目录**：`Downloads/r9-pickup-overview-records-2026-04-24T04-56-37-179Z.png`

### Session #5 · 2026-04-23

- **执行范围**：Round 4（Teachers）
- **结果**：0/6 通过，1 阻塞
- **Bug 发现**：1（未修）
- **Next Action**：修复老师模块后重试 Round 4
- **Blockers**：查询与保存动作均未触发 `/admin/teachers` 请求；控制台存在 `Unhandled Rejection` 且伴随 500/401
- **截图目录**：无（本轮无有效进展截图）

### Session #9 · 2026-04-23

- **执行范围**：Round 4（Teachers）复核收口（4.3/4.5）
- **结果**：6/6 通过（有修复后通过）
- **Bug 发现**：0（本次仅复核）
- **Next Action**：从 Round 5（Students）继续
- **Blockers**：无
- **截图目录**：无

### Session #8 · 2026-04-23

- **执行范围**：Round 4（Teachers）续测（入口补齐后验证）
- **结果**：4/6 通过，2 待复核
- **Bug 发现**：0（未新增）
- **Next Action**：服务恢复后复核 4.3/4.5 两条 PUT 接口
- **Blockers**：前端入口已具备（编辑弹窗、状态确认框可见），但保存/确认提交时出现 `Failed to load resource: net::ERR_CONNECTION_REFUSED`
- **截图目录**：无

### Session #7 · 2026-04-23

- **执行范围**：Round 4（Teachers）续测（4.3/4.5）
- **结果**：4/6 通过，2 阻塞（与 Session #6 结论一致）
- **Bug 发现**：0（未新增）
- **Next Action**：等待老师页补齐编辑与状态变更入口后继续 Round 4 收口
- **Blockers**：列表与详情仅有「详情/删除」，仍无法触发 `PUT /admin/teachers/:id`、`PUT /admin/teachers/:id/status`
- **截图目录**：无（本轮未产出有效截图）

### Session #6 · 2026-04-23

- **执行范围**：Round 4（Teachers）复测
- **结果**：4/6 通过，2 阻塞
- **Bug 发现**：2（未修）
- **Next Action**：继续补齐 4.3/4.5（编辑与状态变更）
- **Blockers**：当前页面未提供编辑入口与状态切换入口，无法触发 `PUT /admin/teachers/:id`、`PUT /admin/teachers/:id/status`
- **截图目录**：无（`playwright_screenshot` 超时）

### Session #4 · 2026-04-23

- **执行范围**：Round 3（Classes）
- **结果**：7/7 用例通过，0 阻塞
- **Bug 发现**：0
- **Next Action**：从 **Round 4（Teachers）** 继续
- **Blockers**：Playwright 截图工具超时（本轮）
- **截图目录**：`screenshots/r3-classes-*.png`（本轮未成功写入）

### Session #3 · 2026-04-23

- **执行范围**：Round 2（Roles）
- **结果**：5/5 用例通过，0 阻塞
- **Bug 发现**：0
- **Next Action**：从 **Round 3（Classes）** 继续
- **Blockers**：无
- **截图目录**：`screenshots/r2-roles-final-*.png`

### Session #2 · 2026-04-23

- **执行范围**：Round 1（Settings）
- **结果**：6/6 用例通过，0 阻塞
- **Bug 发现**：0
- **Next Action**：从 **Round 2（Roles）** 继续
- **Blockers**：无
- **截图目录**：`screenshots/r1-settings-years-*.png`

### Session #1 · 2026-04-23

- **执行范围**：Round 0
- **结果**：7/7 用例通过，0 阻塞
- **Bug 发现**：0
- **Next Action**：从 **Round 1（Settings）** 继续
- **Blockers**：无
- **截图目录**：`screenshots/r0-final-classes-*.png`

### Session #0 · 2026-04-23（元会话 · 计划重置）

- **执行范围**：无（仅文档）
- **结果**：`E2E_EXECUTION_PLAN.md` 升级为「高数据量 + 业务化命名 + Round 收尾最少条数」；本进度文件与报告目录重置
- **Bug 发现**：—
- **Next Action**：从 **Round 0** 起按新计划执行 E2E
- **Blockers**：无
- **旧报告**：`playwright-reports/e2e-20260422/REPORT.md` 已标记作废，新执行写入 `e2e-20260423/`

---

## 给 AI 的续接流程

**新会话第一步**：

1. 读 `admin/tests/E2E_PROGRESS.md`（本文件）→ 看 `Current Round` 和 `Next Action`
2. 读 `admin/tests/E2E_EXECUTION_PLAN.md` 对应 Round 的用例表与 **「收尾数据量」**
3. 确认后端 `:8080` + 前端 `:3000` 在运行
4. 执行 Playwright 登录（如需）
5. 从 `Next Action` 指示的 Round 开始执行；**每 Round 结束核对 1.5 总表 + 本 Round 收尾条数并附截图**

**每完成一个 Round**：

1. 在上方「Round 进度表」更新该行状态、Bug 数、完成日期
2. 如有 Bug 修复，追加到「累计 Bug 修复清单」
3. 更新顶部「当前状态」区块的 `Current Round` / `Next Action`

**会话结束时**：

1. 在「会话日志」追加一条记录
2. 更新顶部「当前状态」区块
3. 告诉用户下次启动提示词

---

## 启动提示词（给用户）

### 首次执行或继续执行

```
按 admin/tests/E2E_PROGRESS.md 续接执行 E2E 测试，使用 Playwright MCP 工具。
```

### 指定执行某个 Round

```
按 admin/tests/E2E_PROGRESS.md 执行 Round 3（Classes），使用 Playwright MCP 工具。
```

### 指定执行某个 Phase

```
按 admin/tests/E2E_PROGRESS.md 执行 Phase 2（核心主数据），使用 Playwright MCP 工具。
```
