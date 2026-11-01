# 设计落地进度板

> 计划：`docs/design-previews/2026-10-25-full-redesign/_shared/IMPLEMENTATION-PLAN.md`
> 工作流：`.claude/skills/slice-delivery/SKILL.md`
> 适配层：`docs/design-previews/2026-10-25-full-redesign/_shared/IMPLEMENTATION-WORKFLOW.md`
> 检查表：`docs/design-previews/2026-10-25-full-redesign/_shared/VALIDATORS.md`

- 工作流工件：**已就绪**（skill、两个 agent、check-secrets、validators、adapter）
- 当前波次：**Phase 2-A 全班视图 + Phase 2-B 发布通知已完成（真实后端 + DevTools 验证）**
- 下一动作：Phase 2-C 视频（`PHASE2-PLAN.md`）
- 后端：8080 运行中（PID 44216，含 Phase 2-A/B 代码；可用内存 ~2.7GB）
- 最后更新：2026-11-01

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
| P11 | W3 | 家长消息 26 | passed | 114675b | - | impl-reviews/P11.md | 0 P1 / 8 P2；DevTools 61 条/分页；token 53 |
| B18 | W3* | 老师任务列表 stats/筛选 + 参与人数 | passed | e986835 | a5795bb | impl-reviews/B18.md | 0 P1 / 4 P2；HTTP 62/62 |
| T03 | W3 | 老师任务列表/详情 09/10 | passed | a5795bb | - | impl-reviews/T03.md | 0 P1 / 10 P2；DevTools 全交互；token 164 |
| B19 | W3* | 修复老师请假统计条件累积 | passed | 6820b2d | f121fc4 | impl-reviews/B19.md | 0 P1 / 3 P2；统计与列表一致 |
| T04 | W3 | 老师请假 14 | passed | - | - | impl-reviews/T04.md | 0 P1 / 3 P2；复审 92 分；token 144 |
| T06a | W4 | 老师我的班级 15 | passed | 6412f47 | c1a0758 | impl-reviews/T06a.md | 0 P1 / 8 P2；DevTools 实测；token 118 |
| B20 | W4* | 班级相册后端 | passed | a1653ec | 78026d4 | impl-reviews/B20.md | 0 P1 / 5 P2；HTTP 全过；已归档 |
| B21 | W4* | 相册列表返回前三张封面 | passed | f53b09b | 2b51787 | impl-reviews/B21.md | 0 P1 / 4 P2；HTTP 实测 |
| T06b | W4 | 老师班级相册 16 | passed | c1a0758 | - | impl-reviews/T06b.md | 0 P1 / 8 P2；上传真实落库；token 97 |
| T07 | W4 | 老师日历三页 11/12/13 | passed | 37d9502 | - | impl-reviews/T07.md | 0 P1 / 7 P2；DevTools 全交互；token 93 |
| B22 | W4* | 老师消息未读态/已读接口 | passed | f7dc14a | cea2639 | impl-reviews/B22.md | 0 P1 / 4 P2；未读 11→10→0 |
| T08 | W4 | 老师更多/我的/消息 17/19/18 | passed | cea2639 | - | impl-reviews/T08.md | 0 P1 / 6 P2；多角色修复后复审；token 73 |
| T09 | W5 | 老师登录 02 | passed | 25111c2 | - | impl-reviews/T09.md | 0 P1 / 7 P2；登录全链路实测；token 73 |
| T10 | W5 | 老师晨检 + 保健日历 20/21 | passed | 66239f1 | - | impl-reviews/T10.md | 0 P1 / 10 P2；DevTools 全交互；token 22 |
| P12 | W5 | 家长登录 + 绑定 02/03 | passed | b35c96c | - | impl-reviews/P12.md | 0 P1 / 7 P2；登录态恢复；token 53 |
| X01X02 | W6 | 两端 webview 边界页 | passed | 068774f | - | impl-reviews/X01X02.md | 0 P1 / 2 P2；两端实测；已收口 |
| D01 | W7 | 共享逻辑收口（4 项） | passed | dd9a3cc | - | impl-reviews/D01.md | 0 P1 / 4 P2；两端回归；token 22/53 |
| V01a | W7 | 老师端 23 页遍历回归 | passed | - | - | _shared/impl-reviews/V01.md | 0 P0 / 0 P1 / 2 P2；23/23 页 |
| V01b | W7 | 家长端 28 页遍历回归 | passed | - | - | _shared/impl-reviews/V01.md | 0 P0 / 0 P1 / 1 P2；28/28 页 |
| D02 | W7 | token/样式债终检 | passed | 5b57000 | - | impl-reviews/D02.md | 0 P1 / 3 P2；token 3/3；像素等价 |
| V02 | W7 | 截图视觉比对 | passed | - | - | .claude/runs/V02/report.md | 24/24 截图；0 P1 / 5 类 P2 |
| V03 | W7 | V02 视觉差异小修 | passed | 89cf31b | - | impl-reviews/V03.md | 0 P1 / 6 P2；接送/通知修复 |
| V04 | W7 | 班级学年数据修正 | passed | cecd705 | - | impl-reviews/V04.md | 0 P1 / 3 P2；6 班+罐+学年对齐 2026-2027 |
| W8-A | W8 | 双端 access token 静默续期 | passed | 9bd9d9d | 7a4e843 | impl-reviews/W8-A.md | 0 P1 / 2 P2；单飞刷新/轮换/重放；DevTools 双端实测 |
| W8-A2 | W8 | 静默续期收口：迟到 401 + 家长上传 | passed | 7a4e843 | 6076d56 | impl-reviews/W8-A2.md | 0 P1 / 2 P2；真实上传 401→刷新→重放 200 |
| W8-B | W8 | 后端 P2：422 映射 + 老师点赞幂等 | passed | 9bd9d9d | 38f8169 | impl-reviews/W8-B.md | 0 P1 / 2 P2；4/4 并发用例，反事实验证 1062 |
| W8-C | W8 | 家长二维码空态 + 相册文案 | passed | 6076d56 | 2b5a4c0 | impl-reviews/W8-C.md | 0 P1 / 4 P2；qrcode 请求 0；文案对齐契约 |
| W9-A | W9 | 前端债：登录页回跳守卫/死组件/设计文案 | passed | e9dc87b | 6863a95 | impl-reviews/W9-A.md | 0 P1 / 2 P2；mock 401 实测 |
| W9-B | W9 | 后端债：报告正文周期过滤 + 指定接走人 422 | passed | e9dc87b | 2658de4 | impl-reviews/W9-B.md | 0 P1 / 3 P2；反事实验证取错周期 |
| W9-C | W9 | 成长评论分页（后端） | passed | 2658de4 | 3f16d74,300dd15 | impl-reviews/W9-C.md | 0 P1 / 0 P2；MaxInt 溢出修复 |
| W9-D | W9 | 成长评论分页（前端 + D2 小修） | passed | 300dd15 | 8a0d2cf,f307227 | impl-reviews/W9-D.md | 0 P1 / 3 P2；mock 分页 20→21 |
| OS-ALB-1 | ALB-P1 | 数据模型与迁移 | passed | 703ca04 | 2759bc8 | impl-reviews/OS-ALB-1.md | 0 P1 / 4 P2；历史 0 行，回填未在真实数据验证 |
| OS-ALB-2 | ALB-P1 | 老师相册生命周期 + 幼儿关联 | passed | 2759bc8 | 0a6e9bf | impl-reviews/OS-ALB-2.md | 0 P1 / 5 P2；24 用例、并发软删守卫 |
| OS-ALB-3 | ALB-P1 | 隐私设置拆分 + 老师端联系方式 | passed | 0a6e9bf | d8dc567 | impl-reviews/OS-ALB-3.md | 0 P1 / 5 P2；默认/迁移/逐人过滤 |
| OS-ALB-4 | ALB-P1 | 家长端班级相册后端 | passed | d8dc567 | 66e45c3 | impl-reviews/OS-ALB-4.md | 0 P1 / 3 P2；{A,E} any-off 反事实 |
| OS-ALB-5 | ALB-P1 | 管理端相册后端 | passed | 66e45c3 | ebc2100 | impl-reviews/OS-ALB-5.md | 0 P1 / 3 P2；权限登记 + 审计 |
| OS-ALB-6a | ALB-P1 | 老师端状态/发布/下架 UI | passed | ebc2100 | 19c2115 | impl-reviews/OS-ALB-6a.md | 0 P1 / 2 P2；DevTools mock |
| OS-ALB-6b | ALB-P1 | 老师端编辑/参与者 UI | passed | 19c2115 | 35326bb | impl-reviews/OS-ALB-6b.md | 0 P1 / 2 P2；离班参与者阻断 |
| OS-ALB-7 | ALB-P1 | 家长端相册页面 + 设置 | passed | 19c2115 | 753be99 | impl-reviews/OS-ALB-7.md | 0 P1 / 3 P2；DevTools mock |
| OS-ALB-8 | ALB-P1 | 管理端相册页面 | passed | ebc2100 | 0bf01b2 | impl-reviews/OS-ALB-8.md | 0 P1 / 3 P2；无头 Chrome mock |
| OS-ALB-9 | ALB-P1 | 留存清理 + 状态时间 + 指标 | passed | ebc2100 | 7f3e72d | impl-reviews/OS-ALB-9.md | 0 P1 / 4 P2；真实调用点验收 |
## 当前检查点

```json
{
  "slice_id": "P2-B",
  "phase": "verified",
  "commit_sha": "07e878e",
  "next_action": "commit + archive class-album-publish-notice; then Phase 2-C",
  "updated_at": "2026-11-01T23:15:00+08:00"
}
```

## 交接块（handoff）

### Handoff 2026-11-01T23:15+08:00 — Phase 2-B 发布通知完成（验证通过）

- OpenSpec change `class-album-publish-notice` 全部落地，**0 P1**。
  - 新增站内消息类型 `album`；`PublishAlbum` 成功后在事务外通知参与者监护人，
    逐人按 `allow_class_share` 过滤（无设置行默认允许），同一相册幂等，
    `action_path=/pages/album/detail?id=<album>&student_id=<孩子>`；失败只记日志。
  - 家长端消息中心识别 `album`（标签「相册」），点击沿用既有已读/跳转。
- 验证：
  1. Go 单测：`TestTeacherAlbum|TestAlbumPublish|TestAlbumRepublish` 130s 全绿；
     覆盖发布发消息、opt-out 不收、共同监护人照收、非参与者不收、重复发布不重复、
     msgRepo 未接线仍发布成功。
  2. 真实后端：发布后允许的监护人各收 1 条；关闭班内共享的监护人 0 条；
     下架重发仍 1 条；消息列表/未读数/标记已读/详情跳转全部 200。
  3. DevTools 真实点击：消息中心显示「班级相册已更新 / 相册」→ 点击进入相册详情
     （真实后端 1 张照片）→ 消息 `is_read=1`。
  4. 测试相册/照片/消息/审计/文件零残留；测试期间临时开启的家长 61
     班内共享已恢复为关闭。
- 说明：本轮 subagent 仍受 API 402 余额限制，由主会话直接实现与验证。
- 后端：8080 运行中（PID 44216）。

### Handoff 2026-11-01T22:50+08:00 — Phase 2-A 全班视图完成（验证通过）

- OpenSpec change `class-album-classwide` 的后端/管理端/家长端全部落地，**0 P1**。
  - 后端：`kindergarten_app_settings.class_album_classwide_enabled`（默认 false）+
    幂等迁移 SQL；`GET/PUT /admin/settings` 增 `album.classwide_enabled`（审计同时记录
    `extra_config` 与 `album`）；家长相册列表/详情支持 `scope=mine|all`，关闭时 403、
    非法 422、跨班 404；列表返回 `classwide_enabled`。
  - 管理端：设置页「班级相册」区块 + 开关，复用 `system:kindergarten` 权限。
  - 家长端：`classwide_enabled=true` 时显示「我的孩子 / 全班」胶囊，切换重置分页；
    详情链接携带 scope；关闭/缺字段时不渲染入口。
- 验证：
  1. `go test ./internal/service/ -run 'TestParentClassAlbum|TestSettings.*Classwide|TestClasswide'`
     → ok（150s，CAW-0001–0007/0011 + Phase 1 回归）。
  2. 真实后端 E2E：老师创建 3 照片相册（A/B）→ B 关闭班内共享 → 家长 mine 1 张；
     B 恢复后 mine 2 张 → 管理员开启 → 家长切全班 3 张（真实 DevTools 点击，
     列表计数 2→3，详情「共 3 张照片」）→ 切回 mine 2 张 → 管理员关闭 → 入口消失。
     测试相册/照片/关联/审计/设置零残留。
  3. 管理员 API：GET/PUT 读写、家长端同源生效、教学主任（无 `system:kindergarten`）PUT 403、
     审计 detail 含 album；测试审计行已按 id 清理。
  4. 管理端 `npm run build` exit 0；无头 Chrome 渲染设置页确认「班级相册」区块与开关。
  5. `check-tokens`：parent 3 / admin 8（基线不变）；secret scan 0。
- 残余：
  1. 管理端未做「点击保存」的真实浏览器提交（已用真实 API PUT 覆盖同一路径）。
  2. 本轮 subagent 因 API 402 余额中断，复核由主会话直接完成（非独立 supervisor）。
  3. Phase 2-B–F 未做。
- 后端：8080 运行中（PID 41380，`GOGC=50 GOMEMLIMIT=768MiB`）。

### Handoff 2026-11-01T19:35+08:00 — 班级相册 Phase 1 完成并归档

- OpenSpec change `class-album-parent-sharing` Phase 1 的 10 个切片 + 真实 E2E
  全部通过，**0 P1**；规格 `--strict` 通过，已归档为
  `openspec/changes/archive/2026-11-01-class-album-parent-sharing/`，
  活基线新增 `parent-class-album`、`parent-privacy-settings`，更新
  `class-album`、`growth-visibility`。
- 已交付：隐私设置拆分（班内共享默认开/园外分享默认关）、老师相册
  草稿→发布→下架/编辑/软删/审计、照片 file_id 与幼儿混合关联、
  家长端「我的孩子」照片级过滤 + opt-out 整张隐藏、管理端只读/下架、
  毕业后 90 天/软删 180 天留存、三项服务端指标。
- 真实 E2E（`OS-ALB-E2E`）：老师创建草稿（3 张 file_id、A/B 参与者）→ 发布 →
  家长 visible_photo_count=1（B-only 过滤、A+B opt-out 整张隐藏）→
  下架 404 → 恢复/重发 → 编辑 → 软删 404；审计 publish/unpublish/edit/delete；
  测试数据与设置零残留。Go 回归 265s 全绿。
- 残余：
  1. admin 真实 E2E（无管理员凭据）、B 家长本人视角、毕业 90 天真实边界、
     真机保存原图未做；由服务层测试或 mock 覆盖。
  2. 3 个测试 PNG 的 OSS 对象未删（仅测试资产）。
  3. Phase 2 未做：全班视图开关、视频、园外分享/下载、发布通知/周报、打包导出、人脸模糊。
  4. 残余 P2：历史相册 0 行导致回填未实测；`students:[]` 与显式空覆盖不可区分；
     admin 列表无封面列；`album_complaint` 无入口。
- 后端：8080 运行中（PID 11076，当前源码）；可用内存约 2GB，如被低内存守卫停止，
  由用户确认后再重启。

### Handoff 2026-10-31T14:10+08:00 — W9 技术债收口

- 状态：W0–W8 交付不变；W9 完成 4 片（A/B/C/D + C2/D2 小修），**0 P1**。
- W9-A：家长端 rawRequest 401 不再把登录页写进 `pending_route`；删除无引用的
  家长端 BottomNav；老师相册设计工件文案与实现/契约同步。
- W9-B：成长报告正文按请求周期过滤（不再跨周期取最新）；指定接走人非法参数 422。
- W9-C/C2：成长评论按顶层分页（page/page_size、root_total/has_more，total 仍含回复）；
  极端 page 溢出修复；老师/家长 page_size 上限统一 50。
- W9-D/D2：两端成长详情接入分页——首屏 20 条、加载更多追加、发评论/回复保留已加载页、
  点赞不重拉；家长计数用 root_total；老师「家长评论（N条）」内联渲染修复。
- 验证：Go 定向 12/12 PASS（含 MaxInt 用例）；DevTools 以「仅评论接口 mock」验证
  前端分页交互（首屏 20 → 加载更多 21、发评论 22/23、错误重试）。**8080 仍是旧二进制**。
- 仍需处理：
  1. 重启后端加载 W9-C，再做一次真实评论分页 E2E（当前仅 mock + Go 测试覆盖）。
  2. 数据/产品：`current_semester=spring` 与秋季窗口不一致；班级 2–6 无星星罐行。
  3. 低影响后端：定时通知 worker 的 `Stop()` 未接进程关闭；请假附件无删除路径（不可达）。
  4. 人工：RDS 凭据轮换 + 历史清理；品牌素材；对比度 tokens；真机补验。
- 后端：8080 仍运行 03:49 旧二进制；如被低内存守卫停止，由用户确认后再重启。

### Handoff 2026-10-31T11:05+08:00 — W8 遗留收口（历史；评论分页/rawRequest/死组件/设计文案已由 W9 处理）

- 状态：W0–W7 交付不变；W8 完成 4 片（A/A2/B/C），**0 P1**。
- W8-A/A2：两端 401 静默续期——单飞刷新（refresh 一次性轮换）、落盘新对并重放一次；
  仅刷新失败才清会话回登录；迟到 401 不再清新会话；家长请假附件/任务照片走同一续期链路。
  DevTools 实测真实上传 401 → 1 次刷新 → 重放 200。
- W8-B：家长接送授权人/设置非法参数 → 422（原 500）；老师 LikeGrowth 并发 1062 幂等。
- W8-C：未指定今日接走人时不再请求 `/parent/pickup/qrcode`（消除设计内 422 红字）；
  老师相册空态文案对齐「本期不支持家长端浏览」契约。
- 仍需人工/产品（不阻塞）：RDS 凭据轮换 + 历史清理；品牌素材决策；对比度 tokens；
  真机补验（原生选图、语音、滚动锁）。15 分钟 token 续期已由 W8-A 解决。
- 剩余技术债（不阻塞）：评论分页未实现；`rawRequest` 401 仍无条件写 pending_route；
  班级 2–6 无星星罐行；`current_semester=spring` 与秋季窗口不一致；死组件 BottomNav；
  `custom-tab-bar` 字面量颜色；设计工件相册旧文案。
- 后端：8080 当前源码运行；如被低内存守卫停止，由用户确认后再重启。

### Handoff 2026-10-31T07:05+08:00 — 全部完成（历史；第 5 项 token 续期已由 W8-A 解决）

- 状态：**W0–W7 全部切片完成**；51 页设计全部落地并通过逐片对抗性审查 + V01 全量遍历（51/51，0 P0/0 P1）。
- 终验：V01a 老师 23/23、V01b 家长 28/28；V02 截图 24/24；V03 修视觉差异；V04 修正班级学年数据；D01 共享逻辑收口；D02 token 债 teacher 22→3、parent 53→3。
- 仍需人工（不阻塞交付）：
  1. 阿里云 RDS 凭据轮换 + git 历史清理（仓库必须保持 private）。
  2. 品牌素材：登录 logo 仍是蓝色位图、入口图标仍是彩色 PNG，设计为橙色/线性体系，属品牌决策。
  3. 对比度：主按钮白字压橙 1.9–2.8:1、hint/muted、语义色压浅底；属 tokens 品牌级决策（会同时影响 admin）。
  4. 真机补验：原生选图对话框、语音播放、滚动锁等 CDP 无法稳定自动化的交互。
  5. 15 分钟 access token 无静默续期（401 直接跳登录），既有债务。
- 已知 P2 汇总见各 `_shared/impl-reviews/*.md` 末尾。
- 后端：8080 当前源码运行；如被低内存守卫停止，由用户确认后再重启。

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
