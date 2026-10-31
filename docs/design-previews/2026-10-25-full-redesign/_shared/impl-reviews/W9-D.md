# W9-D · 对抗性实现审查

- verdict: **PASS**
- score: 92/100
- 源码覆盖: `teacher/src/pages/growth/detail.vue`、`parent/src/pages/growth/detail.vue`、`teacher/src/api/teacher-growth.js`
- 审查范围: diff + untracked + 构建 + 调用链 + 路由 + 共享组件 + DevTools 运行时
- 基线: `300dd15`

## DevTools 证据

- 构建: supervisor 重跑 `cd teacher && npm run dev:mp-weixin` → `DONE  Build complete`（8166ms）；`cd parent && npm run dev:mp-weixin` → `DONE  Build complete`（10874ms）；watcher 已按 PID 清理。dist 13:31:54 / 13:35:04，晚于源码 13:28:04。产物含 `COMMENT_PAGE_SIZE = 20`、`加载更多评论`、`root_total`。
- Token: `node scripts/check-tokens.mjs teacher|parent` → exit 1，两端各 3 条既有 `CalendarEventCard.vue:79-80` 违规，与基线逐条一致，改动文件 0 新增。
- Secret scan: `node scripts/check-secrets.mjs --files ...` → exit 0，0 findings。
- 正常态（**评论 mock，已标注**）: 运行中的 `:8080` 早于 W9-C（评论响应仅 `list,total`）。仅在 AppService 包装 `wx.request` 拦截目标 log 的评论 GET/POST（及教师端 mock id 的评论赞），其余委派真实后端，响应 JSON 克隆模拟原生桥。状态：21 顶层 + 1 回复。
  - 教师：首屏 20 根 + 1 回复，`page=1,page_size=20`，计数 22（`total`）；加载更多 `page=2,page_size=20` → 21 根；双击仅 1 次 page2；发根评论 → POST 无 parent_id + 重取 page1+2 → 22 根、新评论末尾、计数 23；回复 → parent_id=9001 + 重取 1+2 → 首楼 2 回复；点赞 → 本地 赞1/is-on，评论 GET 不变。
  - 家长：20 根但标题 **评论 (21)**、互动条 21（`root_total`，非已加载长度）；加载更多 → 21，标题仍 21；发根评论 → 22，标题 22；回复 → parent_id=9001，首楼 2 回复，标题 22；整页点赞走真实后端并已点回原状。
- 空态（真实后端）: 教师 log 26 → `还没有家长评论` / `0 条评论`；家长 log 27 → `评论 (0)` / `还没有评论` / 互动条 0；均无加载更多。
- 错误态（两端 mock 一次性 page2 500）: 列表保留 20 根、按钮保留；`wx.showToast` 探针记录 `mock failure`；再次点击成功 → 21 根，重试可恢复。
- 交互: 全部用 `_verify/act.py` / CDP 指针或触摸事件执行；`navigateTo` 仅用于装载页面。
- Console: 教师三段采集（100s/20s/50s）、家长 120s，errors 0；仅平台 `getSystemInfo` 弃用 warning。
- 路由: 详情页非 tabBar；未改跳转；老师端无 tabBar。端确认：教师 `pages/more/index`、家长 `pages/points/index`（appid 各自匹配）。
- 截图: 尝试失败（daemon 90s 超时，automator 绑定另一端关闭窗口）；以 DOM/计算样式断言为准。
- 清理: mock 已 restore，项目关闭丢弃运行时；家长点赞已恢复，真实 API 复核 `is_liked=false, likes_count=0`、log 28 评论 0。

## P1 · 阻塞问题 (0)

无。

## P2 · 建议 (3)

1. **[pre-existing] 教师端 `家长评论（N条）` 计数不渲染** — 嵌套 `<text class="section-count">` 被 DevTools 丢弃（`.section-count` 为 null；互动条 `N 条评论` 正常）。基线同样存在，本片未改；建议改为单插值。
2. **[polish] 加载更多按钮字号跨端不一致** — teacher `base` 26rpx vs parent `sm` 24rpx；均来自 token，无设计稿约束，可延后统一。
3. **[residual-risk] 旧后端兜底可能重复取页** — `root_total` 缺失时按页长兜底；仅前端先于 W9-C 上线时出现。W9-C 已随 `300dd15` 提交，应同批部署；后续可去兜底或按 id 去重。

## 说明

- 分页运行时证据基于 mock（旧二进制无 `root_total`/`has_more`，团队明示）；后端分页由 W9-C Go 测试覆盖。
- worker-result 的 `base_sha 3f16d74` 是 W9-C 提交前的父提交；对 `300dd15` 的 diff 已复核，范围无越界。
