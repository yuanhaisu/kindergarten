# OS-ALB-6a · 对抗性实现审查

- verdict: **PASS**
- score: 90/100
- 源码覆盖: `teacher/src/pages/album/index.vue`、`teacher/src/pages/class/index.vue`、`teacher/src/api/teacher-albums.js`
- 审查范围: diff vs d8dc567 + untracked + build 产物 + 后端契约（10-album.md / 09-class-and-tasks.md）
- 基线: d8dc567 → d8dc567（未提交）

## DevTools 证据

- 构建: `cd teacher && npm run dev:mp-weixin` → `DONE Build complete`，dist `2026-11-01 12:04:00` 晚于源码；`npm run build:mp-weixin` → exit 0；watcher 已清理
- Token: `node scripts/check-tokens.mjs teacher` → 3 violations（全部既有 `CalendarEventCard.vue:79-80`，基线 3）
- Secret scan: `node scripts/check-secrets.mjs --files ...` → 0 findings
- 目标端: CDP 9222 / automator 9420，live 栈 `pages/more/index`（老师端独有页）
- Mock 口径: 后端 8080 **未监听**（两次探测），按主会话指示不重启。Mock `/teacher/albums*` 与
  `/teacher/class/students/:id`（允许范围）；因后端整体不可用，额外 mock 班级页读接口
  `/teacher/class/info`、`/teacher/class/students`、`/teacher/pickup/overview`、`/teacher/notices`。
  其余路径委派真实后端（工作台 `/teacher/home/overview` 因此 ERR_CONNECTION_REFUSED）。
  原生 `wx.showModal` 用记录型 stub 捕获文案并驱动 confirm/cancel；点击为真实 Input 事件。
- 正常态: 统计 17/12/2；5 册含草稿/已发布/已下架徽标
- 空态: 草稿筛选无结果 →「这个状态下还没有相册」+「查看全部」；点击后双筛选复位
- 错误态: 注入 stats/list 网络失败 →「相册加载失败」+ 重试；恢复后重试回到 24/2；
  家长详情注入 500 → 内联「家长联系方式加载失败」+ 重试，重试后两行恢复
- 交互: 状态筛选 4 项切换→列表与请求参数正确；草稿详情→发布/删除；发布会话 cancel→无请求、
  confirm→POST publish + toast + 列表/统计刷新（29/3）；已发布→下架确认→POST unpublish；
  已下架→重新发布（再次弹未授权确认）；删除确认→DELETE、卡片移除、弹层关闭、统计 24/2；
  publish 注入 500→toast 后端 message、视图保持草稿。未授权文案精确为
  「2 张照片含未授权孩子，发布后其他家长不可见」
- 家长联系方式: 王明/爸爸/138****8888 与 李丽/妈妈/「—」（is-empty）；无家长学生整块隐藏
- Console: album/class 重新导航采集 → 0 error / 0 warning；全量缓冲仅工作台
  `/teacher/home/overview` 连接拒绝（后端不可用，非本切片页面）
- 路由: 老师端无 tabBar；本切片无新增跳转；相册页由 more 页真实 tap 进入
- 截图: `wechat_screenshot` 90s daemon 超时（IDE 已置前台仍失败）→ 改用 DOM/文本断言

## P1 · 阻塞问题 (0)

无。

## P2 · 建议 (2)

### 1. [ux] 学生详情接口非 silent 导致重复提示
- 证据: `teacher/src/api/teacher-class.js:13-15` 未透传 silent；`teacher/src/api/http.js:168-170`
  在 silent=false 时弹全局 toast；`teacher/src/pages/class/index.vue:289-292` 同时渲染内联错误
- 影响: 家长联系方式加载失败时出现两个错误提示
- 建议: wrapper 增加 options 并在班级页传 `{ silent: true }`（该文件不在本切片 Allowed 列表，留作后续）

### 2. [ux] 动作成功 toast 可能被刷新失败 toast 覆盖
- 证据: `teacher/src/pages/album/index.vue:415` 在动作成功后 `await loadAlbums({skeleton:false})`；
  `loadAlbums` 内部吞错并 toast「刷新失败，请重试」（`:147-157`）
- 影响: 服务端已成功、本地状态已修正，但提示可能误导为失败
- 建议: 动作后静默刷新，或在有动作 toast 时抑制刷新失败提示

## 范围与残留

- `git status -- teacher/` 仅 3 个 Allowed 文件；无 commit、无 pages.json/后端/parent/admin/shared-ui/tokens 源改动。
- `shared-design-tokens/build/scss/tokens-*.scss` 仅 `Generated on` 时间戳行变化（强制 build 副产物）。
- 未跟踪 `backend/api/api-parent/13-class-albums.md` mtime 12:09（晚于 worker 结束），属并发切片。
- 残留: 真实后端 E2E（发布落库/未授权计数/家长手机号真值）需新后端；原生 modal 外观未验；
  截图缺失；状态分段控件与分类胶囊的视觉间距仅 DOM 断言。
