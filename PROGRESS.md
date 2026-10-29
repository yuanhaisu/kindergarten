# 设计落地进度板

> 计划：`docs/design-previews/2026-10-25-full-redesign/_shared/IMPLEMENTATION-PLAN.md`
> 工作流：`.claude/skills/slice-delivery/SKILL.md`
> 适配层：`docs/design-previews/2026-10-25-full-redesign/_shared/IMPLEMENTATION-WORKFLOW.md`
> 检查表：`docs/design-previews/2026-10-25-full-redesign/_shared/VALIDATORS.md`

- 工作流工件：**已就绪**（skill、两个 agent、check-secrets、validators、adapter）
- 当前波次：**W0**
- 下一动作：**T01（新会话接手）**
- 后端：`bcxw6cwn2` 已因系统低内存停止；supervisor 验证前由用户确认重启
- 最后更新：2026-10-28

## X00 零入边决策（2026-10-28）

- 老师端 `pages/webview/index`：隐藏入口，保留空态页。
- 家长端 `pages/auth/bind`：仅保留首次绑定流程，不单独放入口。
- 家长端 `pages/messages/index`：在首页/我的补消息入口。
- 家长端 `pages/webview/index`：隐藏入口，保留空态页，等业务域名配置。

## T01 相关决策（2026-10-29）

- 详情页评论区要「关系标签 + 老师回复（嵌套）+ 评论赞」：**先做 B10 后端补齐**，再做 T01。
- 列表「全部班级 ▾」：老师端单班，**降级为静态班级名**（不可点），保留「全部时间」页内筛选与搜索。
- 全班记录「相关 N 位小朋友」：后端无该口径，**全班记录只显示同步状态**；个体记录保留相关人数。
- 创建页「照片 / 视频」：**仅照片**，不渲染视频入口，差异记录待后续切片。

## 切片状态

| ID | 波次 | 范围 | 状态 | Base | Commit | Review | 备注 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| F0 | W0 | 落地基线：AppIcon / PageShell / StateBlock / CSS 兼容 / secret scan | passed | 30f1982 | cabe031 | impl-reviews/F0.md | 0 P1 / 6 P2；README D01/D02 已修 e3b39cc |
| X00 | W0 | 零入边页面决策：老师 webview、家长 bind/messages/webview | passed | - | - | - | 4 项决策已记录 |
| S01 | W1* | 原生 custom-tab-bar + 4 tab 页去 BottomNav | passed | 0434244 | 48cb569 | impl-reviews/S01.md | 0 P1 / 5 P2；原生底栏四页闭环 |
| P01 | W1 | 家长首页 01 | passed | 6d1a2f9 | 516c591 | impl-reviews/P01.md | 0 P1 / 5 P2；S01 原生底栏后复审通过 |
| B06 | W1* | 成长相册 groups/student_id + 评论 replies/relation | passed | ee4aa37 | 8804b0a | impl-reviews/B06.md | 0 P1 / 6 P2；契约对齐，6/6 测试通过 |
| P02 | W1 | 家长成长列表/详情/相册 09/10/11 | passed | 56c01ac | ecd7a05 | impl-reviews/P02.md | 0 P1 / 5 P2；三页/四态/嵌套评论/album groups |
| B07 | W1* | 接送记录/授权人按孩子过滤 + 关系枚举 9 值 | passed | bbe7afe | f7e1c6c | impl-reviews/B07.md | 0 P1 / 7 P2；A/B 隔离 + 9 值校验 |
| P03 | W1 | 家长接送首页/授权人/新增 13/15/16 | passed | 49f036d | f8a1eda | impl-reviews/P03.md | 0 P1 / 7 P2；三页/三态/9 关系/入口 |
| B08 | W1* | 家长设置开关持久化（6 通知 + 2 隐私） | passed | 4a8bbb9 | f9f841e | impl-reviews/B08.md | 0 P1 / 5 P2；默认/部分更新/隔离 |
| P04 | W1 | 家长我的+设置 27/25 | passed | 86c56c8 | e3f315d | impl-reviews/P04.md | 0 P1 / 5 P2；logout 路由待 B09 |
| B09 | W1* | 恢复 `/auth/logout` + token 吊销（B09-full） | passed | e3f315d | 9433a9e | impl-reviews/B09.md | 0 P1 / 7 P2；access/refresh 吊销闭环 |
| B10 | W1* | 老师评论补齐：relation/student_name/likes_count/parent_id + 回复 | passed | f9df5df | fb3dba6 | impl-reviews/B10.md | 0 P1 / 5 P2；嵌套评论/关系标签/评论赞幂等 |
| T01 | W1 | 老师成长列表/创建/详情 04/05/06 | passed | fb3dba6 | 1f78208 | impl-reviews/T01.md | 0 P1 / 7 P2；三页四态+真实交互，token 175 |
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
  "slice_id": "T01",
  "phase": "committed",
  "commit_sha": "1f78208",
  "next_action": "T02 brief",
  "updated_at": "2026-10-29T16:12:00+08:00"
}
```

## 交接块（handoff）

### Handoff 2026-10-29T16:12+08:00

- 当前切片：**T02 老师接送两页**（`pickup/index`、`pickup/records`，设计 07/08）
- 已完成：W0（F0、X00）；B04–B10；S01；P01–P04；T01
- 下一步：读 `IMPLEMENTATION-PLAN.md` T02 行、设计 `teacher/07-pickup-index/`、`08-pickup-records/`、对应 reviews；写 `.claude/runs/T02/brief.md`，派 `slice-worker` → 全新 `slice-supervisor`
- 必须读：
  - `PROGRESS.md`、`_shared/IMPLEMENTATION-PLAN.md` T02 行
  - `_shared/IMPLEMENTATION-WORKFLOW.md`、`_shared/VALIDATORS.md`
  - `teacher/07-pickup-index/`、`08-pickup-records/`
  - `_shared/reviews/teacher-07-*.md`、`teacher-08-*.md`
  - `_shared/impl-reviews/T01.md`（同端外壳/四态/时间工具先例）
  - P03 commit `f8a1eda`（家长端接送三页，可参考）
- 老师端无 tabBar；4 个常驻页仍用 BottomNav；原生导航栏；不要引入 `switchTab`。
- 已知阻塞：无。后端 `8080` 运行中（T01 supervisor 已恢复）；DevTools 锁已释放。
- T01 带入 P2：详情两处小节计数被 WXML 丢弃、创建同步开关无效（后端零值省略）、表现标签空字典不回退、授徽章守卫不可达、大合影多按钮、视觉 1–2px、15 分钟 token 无 refresh 拦截。
- B10 带入 P2：契约写 400 实为 422、relation 按家长关联判定、可见范围谓词重复、GET 契约残留分页提示。
- B09 带入的 P2：每请求 Redis EXISTS、Redis 故障 503、旧无 jti token 过期前不可吊销、teacher/admin 不调用 logout、refresh 归属未校验、E2E 绕过 Casbin。

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
| B07 | f7e1c6c | 2026-10-29 | 0 P1 / 7 P2；接送多孩过滤 + 关系 9 值 |
| P03 | f8a1eda | 2026-10-29 | 0 P1 / 7 P2；接送三页/三态/9 关系/入口 |
| B08 | f9f841e | 2026-10-29 | 0 P1 / 5 P2；设置 8 键持久化 |
| P04 | e3f315d | 2026-10-29 | 0 P1 / 5 P2；我的/设置/消息入口/退出 |
| B09 | 9433a9e | 2026-10-29 | 0 P1 / 7 P2；logout + access/refresh 吊销 |
| B10 | fb3dba6 | 2026-10-29 | 0 P1 / 5 P2；老师评论嵌套/关系/赞 |
| T01 | 1f78208 | 2026-10-29 | 0 P1 / 7 P2；老师成长三页，token 175 |
