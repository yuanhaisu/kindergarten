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

## B03 相关决策（2026-10-29）

- 拆两片：B03a 落库 + 老师端 UI；B03b 定时发布 worker。
- 定时到点由**新增 worker 扫表**触发（status=1 且 scheduled_at<=now → 发布）。
- 创建页增加「存草稿」次按钮；草稿允许标题或正文不完整暂存。
- 附件**仅图片 ≤9 张、单张 ≤20MB**，仅老师端展示；家长端附件延后。

## T02 相关决策（2026-10-29）

- 授权接送人：**先做 B11**，返回 `authorized_person_list[{name, relation, relation_label}]`（保留旧字符串兼容）。
- 概览「异常」：用后端 `overview.abnormal`（今日异常记录数），不改口径。
- 扫码核验失败：**通用失败弹层 + 后端 message**，不显示学生姓名/班级。
- 例外登记语义：**后端改为同时置异常**（`is_authorized_override` → 同时写 abnormal/原因）。

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
| B11 | W1* | 老师接送：授权人结构 + 例外登记同时置异常 | passed | 0079e04 | d0a67b3 | impl-reviews/B11.md | 0 P1 / 4 P2；数组与旧串同源同序；override 置异常 |
| B12 | W1* | 修复接送记录 date 过滤（UTC→本地） | passed | 4686d8e | 1d41578 | impl-reviews/B12.md | 0 P1 / 3 P2；HTTP 旧二进制 0 → 新 6 条 |
| T02 | W1 | 老师接送列表/记录 07/08 | passed | d0a67b3 | b808659 | impl-reviews/T02.md | 0 P1 / 2 P2；复审通过（B12 修复后） |
| B01 | W2 | 成长报告四格统计 | passed | 722054a | c398c0c | impl-reviews/B01.md | 0 P1 / 5 P2；四格实测 67%/45/8-12/15；已归档 |
| B02 | W2 | 班级家长人数 | passed | e3ab84a | f846bf3 | impl-reviews/B02.md | 0 P1 / 3 P2；实测 5/4 人，底部 2 班 9 人；已归档 |
| B03a | W2 | 草稿/定时/附件落库 + 老师端 UI | passed | b839890 | 1cbabcb | impl-reviews/B03a.md | 0 P1 / 4 P2；HTTP 34/34；UI 复验延后 V01（IDE 登录失效） |
| B03b | W2 | 定时发布 worker 扫表 | passed | 96fc2fc | 1fa2128 | impl-reviews/B03b.md | 0 P1 / 4 P2；并发仅 1 次成功；已归档 |
| B04 | W1* | 点赞落库复核 + 两端防连点 | passed | e3b39cc | 8011f8d | impl-reviews/B04.md | 0 P1 / 5 P2；原症状已由 97f4dd8f 修复，本次修并发 1062 |
| B05 | W1* | 首页 feed 按 student_id + relation + 契约键名 | passed | 0c63e14 | ee5dcb5 | impl-reviews/B05.md | 0 P1 / 6 P2 |
| B13 | W3* | 家长通知接口补齐契约字段 | passed | 3ce9efc | 2de7b6d | impl-reviews/B13.md | 0 P1 / 3 P2；HTTP 47 项；无 N+1 |
| P05 | W3 | 家长通知列表/详情 04/05 | passed | 2de7b6d | a7bd080 | impl-reviews/P05.md | 0 P1 / 6 P2；UI 延后 V01；token 74 |
| B14 | W3* | 家长请假：类型/附件/审批人/统计/撤销守卫 | passed | 2b8c2c5 | 726eb44 | impl-reviews/B14.md | 0 P1 / 5 P2；HTTP 47/47；已归档 |
| P06 | W3 | 家长请假列表/创建/详情 06/07/08 | passed | 4ea5153 | db0d03c | impl-reviews/P06.md | 0 P1 / 8 P2；DevTools 实测；token 71 |
| B15 | W3* | 家长任务：scope/已结束详情/current_checkin/老师昵称 | passed | d0d2039 | 10d169c | impl-reviews/B15.md | 0 P1 / 4 P2；HTTP 34/34 |
| P07 | W3 | 家长任务列表/详情/任务墙 17/18/19 | passed | 10d169c | 091342a | impl-reviews/P07.md | 0 P1 / 8 P2；DevTools 全交互；token 71 |
| B16 | W3* | 积分概览/排行/流水 + 反馈标签 | passed | 8505a52 | 5dc57e4 | impl-reviews/B16.md | 0 P1 / 5 P2；HTTP 68；等级进度降级 |
| P08 | W3 | 家长积分 + 反馈 20/24 | passed | 5dc57e4 | 05f409a | impl-reviews/P08.md | 0 P1 / 8 P2；DevTools 实测；token 55 |
| B17 | W3* | 班级老师 label/phone/排序 + 档案日期格式 | passed | 9e98536 | e956b5f | impl-reviews/B17.md | 0 P1 / 5 P2；HTTP 验证通过 |
| P09 | W3 | 家长班级 + 幼儿档案 21/23 | passed | e956b5f | fd260ed | impl-reviews/P09.md | 0 P1 / 8 P2；多孩串修修复后复审通过；token 53 |
| P10 | W3 | 家长班级日历 22 | passed | 5e2a40f | - | impl-reviews/P10.md | 0 P1 / 5 P2；DevTools 全交互；token 53 |
| P11 | W3 | 家长消息 26 | in_progress | - | - | - | 依赖 F0、X00 |
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
  "slice_id": "B02",
  "phase": "committed",
  "commit_sha": "f846bf3",
  "next_action": "P05 brief or B03 decision",
  "updated_at": "2026-10-29T22:45:00+08:00"
}
```

## 交接块（handoff）

### Handoff 2026-10-29T22:45+08:00

- 当前切片：**W2 后端片完成**（B01、B02 均已提交并归档）；**W1 全部完成**（F0/X00/B04–B12/S01/P01–P04/T01–T02）
- 下一步：**W3 页面片 P05**（家长通知列表/详情 04/05），或先做可选 B03（草稿编辑定时/附件，需产品确认）
- 必须读：
  - `PROGRESS.md`、`_shared/IMPLEMENTATION-PLAN.md` P05 行
  - `_shared/IMPLEMENTATION-WORKFLOW.md`、`_shared/VALIDATORS.md`
  - `parent/04-notice-index/`、`05-notice-detail/`
  - `_shared/reviews/parent-04-*.md`、`parent-05-*.md`
  - `_shared/impl-reviews/T01.md`（四态/时间工具先例）、`P02.md`（家长端列表先例）
  - 活基线规格：`openspec/specs/growth-report/`、`openspec/specs/class-roster/`
- 家长端有 tabBar（4 项），tab 页跳转用 `switchTab`；老师端无 tabBar。
- 已知阻塞：无。后端 `8080` 运行中（B02 新码）；DevTools 锁已释放。
- B01 带入 P2：报告正文未按周期过滤、term 未配置时页面显示通用网络错误、任务可见性 SQL 内联重复、测试数记 10 实为 11。
- B02 带入 P2：TC 映射与 spec 漂移、弹层文案含职务前缀、null 失败分支无单测。
- T02 带入 P2：授权人未过滤 expire_at、token 15 分钟无 refresh。
- T01 带入 P2：详情小节计数被 WXML 丢弃、创建同步开关无效、表现标签空字典不回退、授徽章守卫不可达、大合影多按钮、视觉 1–2px。
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
