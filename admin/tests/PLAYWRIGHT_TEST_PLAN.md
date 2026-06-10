# 养成星球 · 管理后台 Playwright 测试计划

> 目标：通过 Playwright 自动化验证 admin 端所有页面功能是否正常，覆盖登录、菜单导航、核心 CRUD、权限与边界。
>
> 技术栈：Nuxt 3 + Vue 3 + Element Plus + Pinia + ECharts

---

## 一、测试环境准备

### 前置条件

| 项目 | 说明 |
|---|---|
| 后端服务 | `server/cmd/api/main.go` 启动，监听 `http://localhost:8080` |
| 前端服务 | `admin/` 下 `npm run dev`，默认 `http://localhost:3000` |
| 测试账号 | 管理员手机号 + 密码（需在执行前提供一组有效账号） |
| API 基址 | `http://localhost:8080/api/v1`（见 `admin/nuxt.config.ts`） |

### 基础配置

- 浏览器：Chromium（默认）
- 视口：1440 × 900（桌面后台）
- 启动后预跑健康检查：访问 `/login` 并断言页面 200、无 console 致命错误

---

## 二、测试范围

admin 共 **21 个业务页面 + 1 个登录页**（见 `admin/constants/menus.ts`），按模块分为 7 大类。

### P0 · 认证与框架（必测）

| # | 测试项 | 覆盖页面 / 模块 |
|---|---|---|
| 1 | 未登录访问任意页面 → 跳转 `/login`（`middleware/auth.global.ts`） | 全局 |
| 2 | 已登录访问 `/login` → 跳转首页（`middleware/guest.ts`） | `/login` |
| 3 | 空表单校验（手机号 / 密码必填） | `/login` |
| 4 | 错误密码 → ElMessage 报错 | `/login` |
| 5 | 正确账号 → 登录成功跳转 `/` | `/login` |
| 6 | 刷新页面后 token 保持（Pinia + localStorage） | 全局 |
| 7 | 退出登录 → 清除 session 并回到 `/login` | Topbar |
| 8 | 侧边栏菜单渲染、折叠 / 展开 | `components/layout/Sidebar.vue` |
| 9 | 面包屑与路由联动 | `components/layout/Breadcrumb.vue` |

### P0 · 数据看板（`/`）

| # | 测试项 |
|---|---|
| 10 | 4 个统计卡片数据渲染（在园 / 请假 / 出勤率 / 已接走） |
| 11 | 近 30 日出勤趋势图表加载（ECharts） |
| 12 | 今日待办数量渲染 |
| 13 | 班级出勤排行表、成长记录统计卡加载 |
| 14 | 4 个并发接口（`dashboard/overview`、`trend`、`ranking`、`growth/stats`）均 200 |

### P1 · 组织架构类 CRUD

| # | 页面 | 核心用例 |
|---|---|---|
| 15 | `/classes` 班级管理 | 列表加载、搜索、新建、编辑、删除（二次确认） |
| 16 | `/students` 学生管理 | 列表、搜索、筛选班级、新增学生、关联家长、禁用 / 启用 |
| 17 | `/teachers` 老师管理 | 列表、新增老师、分配班级、重置密码 |
| 18 | `/parents` 家长管理 | 列表、搜索、关联学生、解除关联 |

### P1 · 日常业务

| # | 页面 | 核心用例 |
|---|---|---|
| 19 | `/attendance` 出勤 | 日期筛选、按班级查询、导出 |
| 20 | `/leaves` 请假审批 | 待审列表、通过 / 驳回、审批理由 |
| 21 | `/pickup` 接送记录 | 日期筛选、记录列表、异常标记 |
| 22 | `/growth` 成长记录 | 列表、查看详情、分页 |
| 23 | `/tasks` 亲子任务 | 任务列表、新增任务、指派班级、截止时间 |
| 24 | `/points` 积分管理 | 积分规则、手动调整、积分流水 |
| 25 | `/achievements` 成就管理 | 成就列表、创建 / 编辑成就、启用开关 |

### P1 · 内容与通知

| # | 页面 | 核心用例 |
|---|---|---|
| 26 | `/notices` 通知管理 | 发布通知、接收范围选择、撤回 |
| 27 | `/announcements` 园所公告 | 富文本发布、置顶、删除 |
| 28 | `/audit` 内容审核 | 待审列表、通过 / 驳回 |

### P2 · 统计与系统

| # | 页面 | 核心用例 |
|---|---|---|
| 29 | `/statistics` 数据统计 | 各维度图表切换、时间范围 |
| 30 | `/admins` 管理员 | 列表、新增、分配角色 |
| 31 | `/roles` 角色权限 | 角色列表、权限勾选树、保存 |
| 32 | `/settings` 系统设置 | 园所信息编辑、保存 |
| 33 | `/logs` 操作日志 | 列表、筛选时间 / 操作人 / 类型、分页 |
| 34 | `/export` 数据导出 | 选择类型导出、下载文件校验 |

---

## 三、每个业务页面的通用测试模板

```text
1. 页面可访问（返回 200，无 JS 错误）
2. 列表接口正常返回，表格渲染行数 > 0（或空态 el-empty 正确展示）
3. 搜索 / 筛选 → 接口带上查询参数 → 列表刷新
4. 分页切换 → 接口 page 参数变化 → 数据刷新
5. 新增 / 编辑表单：
   - 必填校验
   - 取消按钮关闭弹窗
   - 保存成功提示 + 列表刷新
6. 删除操作：确认弹窗 → 接口调用 → toast 提示
7. 无权限用户访问 → permission 中间件拦截或菜单不显示
```

---

## 四、Playwright 可用的关键技巧（对应 MCP 工具）

| 场景 | 工具 | 用法 |
|---|---|---|
| 打开页面 | `playwright_navigate` | 首次进入 + 设置视口 |
| 表单登录 | `playwright_fill` + `playwright_click` | 填手机号 / 密码 → 点击登录 |
| 等待接口 | `playwright_expect_response` + `playwright_assert_response` | 断言 `/auth/login/password` 返回 200 |
| 校验渲染 | `playwright_get_visible_text` | 断言页面标题、统计数字存在 |
| 抓错误 | `playwright_console_logs`（type: `error`） | 捕获前端报错 |
| 保留现场 | `playwright_screenshot` | 失败截图便于排查 |
| 执行断言 | `playwright_evaluate` | 读取 `localStorage.token`、DOM 状态 |

---

## 五、建议的执行分层（由浅入深）

### 第 1 轮 · 冒烟（约 5 分钟）

- 登录页 → 正确登录 → 看板渲染成功 → 控制台无 error

### 第 2 轮 · 导航冒烟（约 10 分钟）

- 登录后依次点击侧边栏 21 个菜单项，每个页面：
  - 断言路由跳转成功
  - 断言列表或空态正确渲染
  - 收集 console 错误与失败的 XHR

### 第 3 轮 · 核心业务 CRUD（约 30 分钟）

- 选取 P1 里 3~5 个高频模块（建议：`students`、`leaves`、`notices`、`tasks`、`logs`）做完整 CRUD 流程。

### 第 4 轮 · 权限与边界

- 准备一个受限账号，验证菜单项过滤、越权访问被拦截。

---

## 六、执行前需要确认的事项

1. **后端和前端是否已经在本机起好？** 端口分别是什么？（后端默认 `:8080`，Nuxt dev 默认 `:3000`）
2. **一组可登录的管理员账号**（手机号 + 密码）。
3. **本轮想跑哪个层级？**
   - A. 只跑冒烟（第 1 + 2 轮）
   - B. 冒烟 + 2~3 个重点模块 CRUD（第 1 + 2 + 3 轮部分）
   - C. 全量（按页面逐个覆盖，耗时较长）
4. **失败截图和报告保存目录**：建议 `admin/tests/playwright-reports/`。

---

## 七、产出物规划

- `admin/tests/playwright-reports/YYYYMMDD-HHmm/` 每轮测试单独目录
  - `summary.md` 用例通过 / 失败汇总
  - `screenshots/` 关键步骤截图，失败必留
  - `console-errors.log` 浏览器控制台错误聚合
  - `network-failures.log` 4xx / 5xx 接口列表
