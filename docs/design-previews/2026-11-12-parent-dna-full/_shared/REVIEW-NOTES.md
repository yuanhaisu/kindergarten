# 全量设计稿 · 评审待修项

> 内部清单，修完即删。来源：逐页首屏 + 全页截图审阅、`backend/api/api-parent/` 口径核对。

## 全局

- [x] 二级页返回钮统一为无边框 chevron（`_shared/normalize.mjs`，已跑）
- [ ] 全部页面重出 390×844 截图（agent 的截图可能早于修订）
- [ ] `24-points/_fullcheck.html`、`_fullcheck.png` 是 agent 的临时 QA 文件，交付前删除

## 逐页

- [x] `08-leave-index`：状态文案改为 API 口径「待审批 / 已同意 / 已拒绝 / 已取消」；
      统计条「已通过」→「已同意」；筛选 chips 同步
- [x] `10-leave-detail`：当前为待审批，符合「仅待审批可撤回」，保留
- [x] `19-pickup-authorized`：授权人状态「已授权」→「已通过」（review_status_label）；
      另可保留「长期授权」字样
- [x] `24-points`：改为累计积分口径 —— 总分 1245、本周 +45、本月 +180、班级排名 3/28；
      去掉「可用/已使用」；流水显示 amount + balance + source_label；
      成就 earned 8 / total 20，未获得只显示描述与锁态（进度字段 API 未实现，不编数字）
- [x] `12-growth-detail`：照片 4 张 + 「+2」在 3 列网格里落单，改 2 列 2×2
- [x] `14-growth-report`：补 API 的四格统计（出勤 / 积分 / 任务 / 成长记录）
- [x] `22-tasks-detail`：h1 现折成「…开心的 / 事」单字孤行，手动断成两行更均衡
- [x] `15-album-index`：page-bar 小标题「首页」→「小一班」；相册按日期倒序
      （手工课 11-06、音乐课 11-01、生日会 10-30、运动会 10-18、种植角 09-26、午餐时光 09-18）
- [x] `28-feedback`：联系方式预填尾号与「我的」页不一致（138****6721），改为 138 0013 6721
- [x] `07-notice-detail`：844 首屏确认底部操作条正常，附件卡完整
- [x] `30-messages`：空态 CTA 已改为 `btn--primary`

## 收尾

- [ ] `node _shared/normalize.mjs`
- [ ] 逐页重跑 `od-shot`（含 17 两态、09 错误态、各 empty）
- [ ] `node _shared/audit.mjs`（0 error）
- [ ] `node _shared/font-check.mjs`（全部 OK）
- [ ] `node _shared/build-gallery.mjs` + 画廊截图
- [ ] README 的「9 个状态变体」应为 10
- [ ] 写 `REVIEW.md`、更新 `PLAN.md` 状态、提交
