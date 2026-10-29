# 设计落地进度板

> 计划：`docs/design-previews/2026-10-25-full-redesign/_shared/IMPLEMENTATION-PLAN.md`
> 工作流：`.claude/skills/slice-delivery/SKILL.md`
> 适配层：`docs/design-previews/2026-10-25-full-redesign/_shared/IMPLEMENTATION-WORKFLOW.md`
> 检查表：`docs/design-previews/2026-10-25-full-redesign/_shared/VALIDATORS.md`

- 工作流工件：**已就绪**（skill、两个 agent、check-secrets、validators、adapter）
- 当前波次：**W0**
- 下一动作：**B07 worker**
- 后端：`bcxw6cwn2` 已因系统低内存停止；supervisor 验证前由用户确认重启
- 最后更新：2026-10-28

## X00 零入边决策（2026-10-28）

- 老师端 `pages/webview/index`：隐藏入口，保留空态页。
- 家长端 `pages/auth/bind`：仅保留首次绑定流程，不单独放入口。
- 家长端 `pages/messages/index`：在首页/我的补消息入口。
- 家长端 `pages/webview/index`：隐藏入口，保留空态页，等业务域名配置。

## 切片状态

| ID | 波次 | 范围 | 状态 | Base | Commit | Review | 备注 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| F0 | W0 | 落地基线：AppIcon / PageShell / StateBlock / CSS 兼容 / secret scan | passed | 30f1982 | cabe031 | impl-reviews/F0.md | 0 P1 / 6 P2；README D01/D02 已修 e3b39cc |
| X00 | W0 | 零入边页面决策：老师 webview、家长 bind/messages/webview | passed | - | - | - | 4 项决策已记录 |
| S01 | W1* | 原生 custom-tab-bar + 4 tab 页去 BottomNav | passed | 0434244 | 48cb569 | impl-reviews/S01.md | 0 P1 / 5 P2；原生底栏四页闭环 |
| P01 | W1 | 家长首页 01 | passed | 6d1a2f9 | 516c591 | impl-reviews/P01.md | 0 P1 / 5 P2；S01 原生底栏后复审通过 |
| B06 | W1* | 成长相册 groups/student_id + 评论 replies/relation | passed | ee4aa37 | 8804b0a | impl-reviews/B06.md | 0 P1 / 6 P2；契约对齐，6/6 测试通过 |
| P02 | W1 | 家长成长列表/详情/相册 09/10/11 | passed | 56c01ac | ecd7a05 | impl-reviews/P02.md | 0 P1 / 5 P2；三页/四态/嵌套评论/album groups |
| B07 | W1* | 接送记录/授权人按孩子过滤 + 关系枚举 9 值 | briefed | bbe7afe | - | - | P03 前置 |
| P03 | W1 | 家长接送首页/授权人/新增 13/15/16 | pending | - | - | - | 依赖 F0、B07 |
| P04 | W1 | 家长我的+设置 27/25 | pending | - | - | - | 依赖 F0 |
| T01 | W1 | 老师成长列表/创建/详情 04/05/06 | pending | - | - | - | 依赖 F0；保留既有发布工作台行为 |
| T02 | W1 | 老师接送列表/记录 07/08 | pending | - | - | - | 依赖 F0 |
| B01 | W2 | 成长报告四格统计 | pending | - | - | - | OpenSpec；页面 12 已存在 |
| B02 | W2 | 班级家长人数 | pending | - | - | - | OpenSpec；页面 03 已存在 |
| B03 | W2 | 草稿编辑定时/附件 | pending | - | - | - | OpenSpec；可选 |
| B04 | W1* | 点赞落库复核 + 两端防连点 | passed | e3b39cc | 8011f8d | impl-reviews/B04.md | 0 P1 / 5 P2；原症状已由 97f4dd8f 修复，本次修并发 1062 |
| B05 | W1* | 首页 feed 按 student_id + relation + 契约键名 | passed | 0c63e14 | ee5dcb5 | impl-reviews/B05.md | 0 P1 / 6 P2 |
| P05 | W3 | 家长通知列表/详情 04/05 | pending | - | - | - | 依赖 F0 |
| P06 | W3 | 家长请假列表/创建/详情 06/07/08 | pending | - | - | - | 依赖 F0 |
| P07 | W3 | 家长任务列表/详情/任务墙 17/18/19 | pending | - | - | - | 依赖 F0 |
| P08 | W3 | 家长积分+反馈 20/24 | pending | - | - | - | 依赖 F0 |
| P09 | W3 | 家长班级+幼儿档案 21/23 | pending | - | - | - | 依赖 F0 |
| T03 | W3 | 老师任务列表/详情 09/10 | pending | - | - | - | 依赖 F0 |
| T04 | W3 | 老师请假 14 | pending | - | - | - | 依赖 F0 |
| T06 | W4 | 老师班级+相册 15/16 | pending | - | - | - | 依赖 F0 |
| T07 | W4 | 老师日历三页 11/12/13 | pending | - | - | - | 依赖 F0 |
| T08 | W4 | 老师更多+我的+消息 17/19/18 | pending | - | - | - | 依赖 F0 |
| P10 | W4 | 家长日历 22 | pending | - | - | - | 依赖 F0 |
| P11 | W4 | 家长消息 26 | pending | - | - | - | 依赖 F0、X00 |
| T09 | W5 | 老师登录 02 | pending | - | - | - | 依赖 F0 |
| T10 | W5 | 老师晨检+保健日历 20/21 | pending | - | - | - | 依赖 F0 |
| P12 | W5 | 家长登录+绑定 02/03 | pending | - | - | - | 依赖 F0、X00 |
| X01 | W6 | 老师 webview 22 | pending | - | - | - | 个人主体不可用；依赖 X00 |
| X02 | W6 | 家长 webview 28 | pending | - | - | - | 需业务域名；依赖 X00 |
| V01 | W7 | 51 页遍历回归 + 真实交互 + 真机 | pending | - | - | - | 依赖全部页面片 |
| V02 | W7 | 截图视觉比对与差异清单 | pending | - | - | - | 依赖 V01 |
| D01 | W7 | 共享逻辑收口 | pending | - | - | - | 依赖 W3 |
| D02 | W7 | token/样式债终检 | pending | - | - | - | 依赖全部页面片 |

## 当前检查点

```json
{
  "slice_id": "B07",
  "phase": "briefed",
  "commit_sha": null,
  "next_action": "spawn B07 worker",
  "updated_at": "2026-10-29T09:40:00+08:00"
}
```

## 交接块（handoff）

### Handoff 2026-10-29T09:15+08:00

- 当前切片：P02 已 passed/committed；下一片 **P03 家长接送三页**
- 已完成：W0（F0、X00）；W1 前置（B04、B05、B06、S01）；P01、P02
- 下一步：新会话读 `PROGRESS.md` + `IMPLEMENTATION-PLAN.md` 的 P03 行，写 `.claude/runs/P03/brief.md`，派 `slice-worker`
- 必须读：
  - `PROGRESS.md`、`_shared/IMPLEMENTATION-PLAN.md` P03 行
  - `_shared/IMPLEMENTATION-WORKFLOW.md`、`_shared/VALIDATORS.md`
  - `parent/13-pickup-index/`、`15-pickup-authorized/`、`16-pickup-authorized-create/`
  - `_shared/reviews/parent-13-pickup-index.md`、`parent-15-pickup-authorized.md`、`parent-16-pickup-authorized-create.md`
  - S01 commit `48cb569`（原生底栏）、P02 commit `ecd7a05`（三页模式）、B06 commit `8804b0a`
- 已知阻塞：无。后端 `8080` PID 13712 运行（可能被低内存停）；DevTools 锁空闲。
- P02 带入 P03 的 P2：album 下拉刷新未开启、评论计数口径、video 原生层级、P01 `formatTime` 未迁移、album 错误态冻结换月。
- 上下文：主会话已约 60%，按约定 handoff；新会话只读上述文件，不要重读旧对话。

## 最近完成

| Slice | Commit | 日期 | 备注 |
| --- | --- | --- | --- |
| F0 | cabe031 | 2026-10-28 | 0 P1 / 6 P2；AppIcon/PageShell/CSS-COMPAT；5 页回归通过 |
| B04 | 8011f8d | 2026-10-28 | 0 P1 / 5 P2；点赞并发 1062 修复 + 回归测试 |
| B05 | ee5dcb5 | 2026-10-28 | 0 P1 / 6 P2；feed 按孩子过滤 + relation/relation_label |
| S01 | 48cb569 | 2026-10-28 | 0 P1 / 5 P2；原生 custom-tab-bar，四页闭环 |
| P01 | 516c591 | 2026-10-28 | 0 P1 / 5 P2；家长首页六块/四态/多孩/feed 隔离 |
| B06 | 8804b0a | 2026-10-28 | 0 P1 / 6 P2；album groups + comments replies |
| P02 | ecd7a05 | 2026-10-29 | 0 P1 / 5 P2；成长三页/四态/嵌套评论/album groups |
