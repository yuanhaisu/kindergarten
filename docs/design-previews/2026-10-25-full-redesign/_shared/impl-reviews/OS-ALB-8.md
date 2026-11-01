# OS-ALB-8 · 对抗性实现审查

- verdict: PASS
- score: 92/100
- 源码覆盖: `admin/pages/albums/index.vue`（新）、`admin/api/albums.ts`（新）、`admin/constants/menus.ts`
- 审查范围: diff + untracked + 构建配置 + 调用链（useHttp/DataTable/usePagination/permission 守卫）+ 路由 + 契约
- 基线: `66e45c3` → `ebc2100`（OS-ALB-5 后端落库；管理端 UI 仍为工作树改动，未提交）

## DevTools 证据

- 构建: `cd admin && npm run build` → exit 0，产物 `admin/.output`（2026-11-01 13:03 +08:00）
- Token: `node scripts/check-tokens.mjs admin` → 8 violations（全部既有：teachers/index.vue x6、_variables.scss x2），本切片 3 文件 0 命中
- Secret scan: `node scripts/check-secrets.mjs --files <3 files>` → 0 findings
- 单测: `cd admin && npx vitest run tests/unit` → 2 files / 18 tests passed
- 正常态: 构建产物 + 无头 Chrome CDP + `/api/v1` 请求拦截；3 行（已发布/草稿/已下架）、8 列、3 筛选、分页、菜单入口齐全
- 空态: mock 空列表 → 表格「暂无相册」
- 错误态: 列表 HTTP 500 `{code:50000,message:'服务器开小差了'}` → ElMessage 原样提示
- 交互: 状态筛选→`status=published` 请求；重置→无筛选参请求；分页→`page=2`；详情抽屉→参与幼儿/2 张照片/无编辑；下架二次确认→恰一次 POST → 抽屉与列表状态同步刷新；权限负例→菜单隐藏并重定向 `/attendance`
- Console: 本切片页面 0 error / 0 未捕获异常
- 路由: 管理端无 tabBar/switchTab；`/albums` 可达，菜单派生权限守卫双向验证
- 截图: 8 张 PNG（`.claude/runs/OS-ALB-8/evidence/`）

## P1 · 阻塞问题 (0)

无。

## P2 · 建议 (3)

### 1. [scope] 列表缺「封面」列
- 证据: `admin/pages/albums/index.vue:41`；契约 `backend/api/api-admin/22-albums.md:93` 明确列表不返回照片/参与者以避免 N+1
- 建议: 按 worker 已登记的口径延后；若产品坚持，扩契约+后端返回封面缩略图，禁止前端逐行补请求

### 2. [docs] 注释引用过期契约文件名
- 证据: `admin/constants/menus.ts:76` 写「契约 21-albums.md」，实际为 `22-albums.md`（21 已被 calendar 占用，见契约第 10 行）
- 建议: 改为 22-albums.md（`.claude/runs/OS-ALB-8/validators.md` 同步修正）

### 3. [edge] 图片预览索引可能漂移
- 证据: `admin/pages/albums/index.vue:264` 的 `previewUrls` 过滤空 URL，而 `:initial-index="index"`（第 123 行）用未过滤下标
- 建议: 用「照片 → 预览索引」映射，或为空 URL 提供占位以保证两数组等长

## 备注

- 未做真实后端 E2E：8080 未运行且按要求未启动；接口形状以契约 + 拦截 mock 校验，登录与真实数据留待联调。
- 「详情」按钮因 Element Plus 固定右列覆盖，自动化用元素 `click()` 事件触发（同一 Vue 处理器），非页面缺陷。
