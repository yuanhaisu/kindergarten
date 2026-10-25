# 亲子任务闭环 A 切片 Handoff

> 用途：把当前实现状态、未完成项、验证步骤交给新会话，避免在 37 MB / 3 次 compact 的旧会话里继续。
> 写于：2026-10-21
> 分支：`feat/parent-growth-scope`
> 写文档时 HEAD：`67fbb51`（仓库有并发提交，执行前先 `git log -3 --oneline` 确认）


## 0. 当前状态（2026-10-24 更新，**新窗口先读这一节**）

| 项 | 值 |
| --- | --- |
| 分支 | `feat/parent-growth-scope` |
| C-3 起点 | `1813880`（已提交为 `c036a55`） |
| C-4 | 已提交 `7ecd6c7`（代码）+ `25554b1`（文档） |
| 已完成切片 | A → B1 → B2 → B3 → B4-a → B4-b → C-1 → C-2 → C-3 → C-4；**C-5 进行中** |
| 下一步 | **可归档**：`tasks.md` **90/90**（`openspec apply` → `all_done`）。真机项（5.4/9.3/9.5/9.7/12.6）按 2026-10-25 决定豁免、记为后续项；12.3 用分段证据落账（双端联跑待环境恢复后补）。详见 §28.6；归档命令 `/opsx:archive` |
| 新窗口开工 | **直接读 §26 的开工提示词**，它已按「先提交 C-4 → 补 3.11/4.5 → 验证」排好序 |

**按角色选读**（不要从第 1 节顺序读到底）：

| 你要做什么 | 读这些 |
| --- | --- |
| C-4 老师端入口 | §22.2（老师端现状）、§22.3（后端接口清单）、§23.2 第 6 条（班级罐口径）、`backend/api/api-teacher/09-class-and-tasks.md` |
| C-5 验证/勾选/归档 | §23.3（证据）、§23.4（缺口）、`tasks.md` 的 11.x/12.x、`openspec/changes/2026-10-20-redesign-parent-task-loop/` |
| 只想改管理端任务页 | §23.1（文件清单）、§23.2（**9 条语义决定，动之前必读**）、§23.4（已知缺口） |
| 排查/接手历史 | §20（C 阶段决策，§20.7-5 / §20.10-6/7 仍被 spec 引用）、§21（C-2） |

> ⚠️ **已过时的章节**（保留仅为追溯，不要当作现状）：
> §7「下一步」（A 阶段）、§8「已知缺口」（A 阶段快照）、§11「新会话开场建议」（B3 时期）。
> 现行缺口一律看 §23.4；现行下一步看本节表格。

> 📌 引用约定：`tasks.md` 引用了本文件的 **§21.8** 与 **§23.3**，decision 引用了 §20.7-5、§20.10-6/7 ——
> **这些编号不要改**，精简/搬移时保住它们。

## 7. 下一步（A 阶段，已过时）

> 本节的待办已由 §12–§23 全部消化（B1–B4、C-1–C-3）。原文见 git 历史 `1813880`。
> 现行下一步见 **§0** 的表格。

## 8. 已知缺口

> ⚠️ **本节为 A 阶段（2026-10-21）快照，多数条目已在 B/C 阶段完成，不要当作现状。**
> 现行缺口见 §23.4。

| 项 | 状态 |
|---|---|
| 发布通知 / `action_path` | 未实现；`CreateTask` 返回 `notified_count: 0` |
| 批量回应并发 | ✅ 已修：事务内 `FOR UPDATE` + `RowsAffected` 判定；输家返回 0 且回滚，无孤儿回应 |
| 周期汇总边界 | ✅ 已修：`taskPeriodRange` 按频率取区间并写入 `PeriodStart/PeriodEnd`；区间外提交不被关闭 |
| 公开申请与老师审核 | 未实现；只有 `allow_public` 与撤回 |
| 补充回应 / 家长反馈 / 未读标记 | 只有模型，无接口 |
| 班级星星罐管理 | 只有模型与校验，无入口；A 只开放按任务 |
| 园所活动 / 多班活动 / 活动统计 | 未实现 |
| 管理端任务页面 | 仍旧列表/删除，未接新字段 |
| 家长列表 Tab | 仍旧“进行中/已打卡/全部” |
| 老师回应文字 | 后端支持 `content`，前端只有贴纸/语音 |
| `has_unread_supplement` | 硬编码 false |
| 任务状态自动结束 | 未做；家长查询按日期过滤，但 `status` 仍为 1 |
| `TaskStats` 周期口径 | 目前是全量提交，不按周期 |
| `TaskService` 管理端 | 已去积分、已接级联删除；新字段展示仍需核对 |
| admin 前端类型 | 仍可能有 `task_checkin_points` 旧类型 |

## 9. 已知坑

- **并发提交**：写文档期间 HEAD 连续前进（`c5f7704 → 77c91e5 → 67fbb51`）。新会话开始前先 `git status` + `git log -3`，确认没有另一个会话正在改同一批文件。
- **旧会话不要继续**：37.8 MB / 11,819 行 / 3 次 compact / 最近一次压缩约 38 分钟；rewind 后当前分支是回退叶子，细节可能不在上下文。
- **nullable**：`task_checkin_states.last_check_in_date` 首次提交为 NULL；GORM AutoMigrate 只增不改，必须手工迁移。
- **时区**：统一 `Asia/Shanghai`；日期比较用 `shanghaiLoc()`。
- **范围**：`taskScopeQuery` 以 `pc.CurrentStudent` 为准；多孩家庭所有任务请求都要带 `student_id`。
- **唯一键**：`task_check_ins` 是 `(task_id, student_id, date)`；不是只按 `date`。
- **媒体**：`media_file_id` 必须属于当前用户且 `biz_type=task`；不要接受任意外链。
- **路由**：批量回应用 `POST /teacher/tasks/:id/batch-response`，避免 Gin 的 `:id` 与 `checkins` 冲突。
- **编辑**：老师编辑页 `status` 用 `?? 1`，不要用 `|| 1`，否则已结束任务会被重新打开。
- **OpenSpec 目录已入库**：`openspec/changes/2026-10-20-redesign-parent-task-loop/` 下的文件都在版本控制里，按普通代码对待即可（此前「未跟踪、别误删」的提醒已过期）。

## 10. 回滚

- 任务代码已提交；按提交回滚：`git revert <commit>`。
- 或只回滚文件：`git checkout <commit> -- <path>`。
- nullable SQL 是向后兼容的，不需要回滚。
- OpenSpec change 目录已入库，随代码一起 `git revert` 即可。

## 11. 新会话开场建议

> ⚠️ **B3 完成后作废。** 新窗口请按 §0 的「按角色选读」起步。

```text
读 docs/handoffs/2026-10-21-parent-task-loop.md 的 §12（B1/B2 已完成）与 §13，然后开工 B3：
OpenSpec change 2026-10-20-redesign-parent-task-loop 的任务 5.1–5.5（通知聚合 + 读标记 + 小程序码）。

注意：B3 会改通知实现，旧提示「不要改通知实现」只适用于 A/B1 阶段，已作废。
先读 design.md D9 与 backend/api/ 的相关契约，再动代码；任务写路径先锁 parent_tasks 行。
按 §12 的窗口纪律：一次只做一组，做完一组先做限制性对抗性审查再验证，通过后才进下一组。
不要动 A/B1/B2 已落地的语义；不要提交 docs/competitive/*、.impeccable/、skills-lock.json。
```

---

## 12. B 切片进展（2026-10-22）

B 按任务组拆成 4 个窗口推进，**窗口内只做一组**，每组走完「改 → 限制性对抗性审查 → 验证」再进下一组：

| 窗口 | 任务组 | 状态 |
| --- | --- | --- |
| **B1** | 3.5 补充回应 + 3.6 家长反馈 | ✅ 2026-10-22 完成（后端） |
| **B2** | 4.1–4.4 公开审核 + 公开墙 | ✅ 2026-10-22 完成（后端，无界面） |
| **B3** | 5.1–5.5 通知聚合 + 读标记 + 小程序码 | ⚠️ 2026-10-23 后端完成（5.1–5.3 勾选；5.4/5.5 的界面与真机扫码待 10.x/12.6） |
| B4 | 9.5/9.7 + 10.5–10.8 前端 + 12.3/12.4/12.6 | 未开始 |

### 12.1 B1 落地内容

- `POST /teacher/tasks/:id/checkins/:checkin_id/supplement` → `TeacherPortalService.SupplementResponse`
- `POST /parent/tasks/responses/:response_id/reaction` → `ParentPortalService.ReactToResponse`
- `ParentPortalService.GetTask` 的 `responses[]` 新增 `supplements[]`（只含当前孩子），`my_reaction` 按家长账号回填
- 契约：`backend/api/api-teacher/09-class-and-tasks.md`、`backend/api/api-parent/06-task.md`
- 测试：`server/internal/service/task_response_supplement_test.go`（17 条，真实 MySQL）；每条关键断言都做过变异验证（删掉被守护的条件后该测试确实失败）

### 12.2 B1 的语义决定（后续不要改错）

- **补充回应不改 `ResponseID`**：它是一条独立 `TaskResponse(single)` + 一行关系表，提交的「已回应」状态与主回应都不动。
- **补充的 `ritual` 不继承任务默认仪式**：空值一律贴纸。否则一个默认 `class_star` 的任务会让所有补充请求 422。
- **补充禁止 `class_star`**：一条提交只有一个主回应能计入星星罐（3.7 的口径依赖这一点）。
- **反馈权限按「自己孩子」而不是「自己提交」**：提交人本人算，与该孩子有绑定关系的另一位家长也算（规格原文「家长只能对与自己孩子提交相关的回应反馈」）。判定时主回应走 `checkin.response_id`、补充回应走 `task_response_supplements` 关系，两条路径都要有；`student_parents` 绑定作为 co-parent 的放行条件。反馈仍记在**家长账号**维度，一条 period 回应可能覆盖两个孩子的提交，不按当前孩子收窄。
- **所有任务写入方都先锁任务行（`SELECT ... FOR UPDATE` on `parent_tasks`）**：老师端 `DeleteTask` / `RespondTask` / `BatchRespondTask` / `SupplementResponse`、家长端 `CheckinTask` / `ReactToResponse`、管理端 `TaskService.DeleteTask`。这是唯一能同时防止「孤儿关系行」和「1213 死锁」的锁序：
  - 孤儿：级联用当前读（`FOR UPDATE` Pluck）取回应 id，能看见并发补充刚提交的回应；
  - 死锁：级联的顺序是「锁回应 → 删 checkin」，而回应路径是「锁 checkin → 插回应」，不先拿任务锁就是 A→B/B→A。
  - 任何**新增**的任务写路径都必须先锁任务行，否则死锁会以 500 形式出现（删除类已映射 503）。
- **语音只能挂音频**：三个回应入口共用同一判断（`ErrTeacherRitualMediaType`），否则会写入家长端播不出来的语音回应。
- **仪式载体必填**：贴纸必须有 `sticker_key`，语音必须有音频媒体。主回应/批量回应/补充回应三个入口共用 `validateRitualPayload`。
- **`BaseModel` 自带 `DeletedAt`**，所有表默认软删除。凡是手写 `.Table(...)` 的查询都要自己补 `deleted_at IS NULL`；upsert 命中软删行时要清 `deleted_at`（同 `growth_likes` 的历史回归）。

### 12.3 B1 的路由结论（推翻 §9 的一条旧结论）

§9 写的「避免 Gin 的 `:id` 与静态段冲突」在 gin 1.11 上**不成立**：`/parent/tasks/responses/:response_id/reaction` 与 `/parent/tasks/:id` 可以共存，静态段优先匹配。`go test ./internal/app` 会构造整张路由表，冲突会直接 panic。

### 12.4 B1 未做的部分

- 老师端界面看不到某条提交已经追加过哪些补充（`TeacherTaskCheckins` 未返回 supplements）——归 9.3。
- 家长端反馈按钮、补充回应渲染、「给孩子看」：归 10.3 / 10.5。
- 指标（`feedback_coverage` / `child_happy_rate` / `ack_rate`）：管理端归 7.2；**老师端反馈汇总此前无人承接**，已登记为 3.11（P1）。落地时补一条「补充回应的反馈不进入主回应指标」的断言。

### 12.5 B1 UI 回归证据（2026-10-22）

实测路径（微信开发者工具 + CDP 直连）：

| # | 端 | 操作 | 结果 |
| --- | --- | --- | --- |
| 1 | 家长 | 打开任务详情 | 正常渲染任务标题/学习关联/提交区/打卡历史 |
| 2 | 家长 | 一键提交（真实鼠标事件） | 页面转为「打卡历史 2026-10-22 已提交」 |
| 3 | 老师 | 微信授权登录（含协议勾选，真实点击） | 进入工作台，`teacher_token` 落盘 |
| 4 | 老师 | 打开任务详情 | 待回应列表出现提交；贴纸选项 star/heart/thumb/录音 齐全 |
| 5 | 老师 | 点击 star 贴纸回应 | 待回应清空；「已回应」标签页出现该提交 |
| 6 | 家长 | 重新打开该任务 | 渲染「老师回应 / 贴纸 / 老师回应了你」，console 0 error |

工具层坑（复现记录，下次别再踩）：

- MCP 的 `wechat_navigate` / `wechat_automator(page_data)` / `wechat_screenshot` 在本机均 **120s 超时**；全程改用 `_verify/wx_cdp.mjs`（CDP 直连 `127.0.0.1:9222`）做导航、读文本、真实点击。
- `wx_cdp.mjs click <sel>` 默认 `touch` 模式 **点不动 uni 的 @click**（没有任何请求发出）；必须传第二个参数 `mouse`。
- DOM 标签是 `wx-view` / `wx-text` / `wx-button`（不是 `uni-*`），选择器要按前者写。
- 老师端与家长端共用同一个 IDE 实例：用 `wechat_ide(open)` 切项目后，**另一端的 automator 会失效**，需要重新 `start`；切换后 storage 仍在，但页面可能停在 login，重新走一次授权登录即可。

未覆盖（前端尚未实现，不是回归遗漏）：老师端补充回应入口（9.3）、家长端反馈按钮与补充回应渲染（10.3/10.5）。

---

## 12.6 B2 落地内容（2026-10-22）

起点 `393a3e3`。按 §13.2 先写契约再写代码；**没有界面**，全部 API/服务层验证。

新增端点（契约已在 `backend/api/`）：

- `POST /parent/tasks/checkins/:checkin_id/public-request` / `DELETE …/public-request`（取消；二选一里选了 DELETE）
- `GET /parent/tasks/public-wall`
- `PUT /teacher/tasks/:id/checkins/:checkin_id/public`（`{approve, scope_confirmed}`）
- `PUT /admin/tasks/checkins/:checkin_id/public`（只接受 `{revoke:true}`，已登记 `task:manage`）

读路径补 `public_*` / `public_status` / `public_visible`（`checkin-history`、老师端 `checkins`）。`public_visible` 由服务端按公开墙同一套条件算出。

### 12.7 B2 的语义决定（后续不要改错）

- **取消/撤回不受任务状态限制**：任务关了公开开关或已结束，家长仍必须能撤回自己的内容；只有「申请」要求 `allow_public=true`、非个体任务、任务有效。最初的实现把两者都套上这组前置，导致撤回入口被任务状态锁死——已修，`TestPublicCancelWorksAfterPublicDisabled/AfterTaskEnded` 守护。
- **重复申请不降级**：已批准且未撤回时再点申请是幂等空操作，避免手滑把墙上记录打回待审核。
- **空操作不写时间戳/日志**：从未申请时取消不写 `revoked_at`；管理端撤回空操作不写 `operation_logs`。
- **个体任务三重拒绝**：申请 422、审核 422、公开墙查询侧再排除一次（即使 2.9 回归留下 `allow_public=true` 也不展示）。
- **归属按「自己孩子」**：同一孩子的另一位家长（有绑定）也能申请/取消，与 3.6 反馈权限口径一致；不是「必须本人提交」。
- **按 checkin_id 进来的写路径**先用**非锁定读**定位 task_id，再锁 `parent_tasks`、最后锁 `task_check_ins`——锁序仍是任务行优先，非锁定读不参与死锁顺序。
- **审核允许在任务最后一天/结束后进行**，但公开墙按 `end_date >= 今天` 实时过滤，响应里的 `public_visible` 会如实返回 false。
- **公开墙多一条安全补强：`st.class_id = t.class_id`**。D7 的九条只约束任务侧，不约束提交者侧；少了它，孩子转班后内容会留在原班级墙上，管理员把任务改派到别的班时原班孩子的照片会漏给新班家长。
- **取消不套 `taskScopeQuery`**：孩子转班后原班级任务不再匹配，套上去会让家长撤不回自己的内容。归属只看「提交属于当前孩子」；「任务是否可见」只在申请时校验。

### 12.9 B2 限制性对抗性审查结论（2026-10-22）

审查方独立复核（未参与实现）后给出的 CONFIRMED / POSSIBLE，逐条处置：

| # | 级别 | 问题 | 处置 |
| --- | --- | --- | --- |
| 1 | CONFIRMED | 转班后撤回入口 404，内容仍留在原班级墙 | ✅ 已修：取消改为按 `id + kindergarten_id` 锁任务；新增 `TestPublicContentFollowsStudentClassMove` |
| 2 | POSSIBLE | 管理端改派任务班级 → 原班孩子内容出现在新班墙 | ✅ 已修：公开墙加 `st.class_id = t.class_id`；新增 `TestPublicWallHidesWhenTaskReassignedToOtherClass` |
| 3 | POSSIBLE | 老师审核锁内缺班级/园所条件（TOCTOU） | ✅ 已修：锁定读补 `kindergarten_id + class_id`（串行测试不可观测，未做变异） |
| 4 | POSSIBLE | `UpdateTask` 对不公开任务无条件撤销，反复给从未申请的提交盖撤回时间 | ✅ 已修：只在 `allow_public` true→false 或班级任务改个体时撤销；新增 `TestEditingNonPublicTaskDoesNotStampRevokedAt` |
| 5 | POSSIBLE | 测试缺 `approved_at IS NOT NULL`、提交软删、排序守卫 | ✅ 已补 3 条用例 + 3 项变异 |

审查确认无问题的部分：状态机逐条对应 D7、幂等语义、锁序无反向、手写 SQL 的软删过滤、权限边界。

### 12.10 第二轮审查（针对修复本身，2026-10-22）

对「修完的那版」再做一轮对抗审查，发现 3 个问题（均已修）：

| # | 问题 | 修法 |
| --- | --- | --- |
| 6 | 关闭 `allow_public` 的撤销只有 helper 级测试，**真实编辑路径没被覆盖**（spec 0143 可能回归而全绿） | 新增 `TestTeacherUpdateTaskClosingPublicRevokes` + 变异「彻底不撤销公开」 |
| 7 | `public_visible` 在老师/管理端按 0 计算，不校验提交者班级，与契约「等价于公开墙」不符 | 家长申请/取消与管理端撤回用 `locateCheckin` 带出学生当前班级；老师审核直接载入提交者班级；老师列表用 `Preload("Student")` 取班级。新增 `TestTeacherCheckinsPublicVisibleFollowsStudentClass` |
| 8 | 提交者已转班仍可批准 → 造出永远不上墙的「已公开」状态 | 批准时校验「提交者班级 = 任务班级」，不满足 422；拒绝/撤回仍可用（`TestTeacherApproveRejectedAfterStudentLeavesClass`） |
| 9 | `revokePublicForTask` 给**从未申请**的提交也盖 `public_revoked_at`，`public_status` 与撤回时间失真 | WHERE 收窄为「仍有公开状态」的行（申请过/确认过/批准过）；D7 的「清空」在这些行上本就是空操作，可见性不变；`TestTeacherUpdateTaskClosingPublicRevokes` 断言同伴提交保持 `none` |
| 10 | `UpdateTask` 撤销判定用的是原始 `req.AllowPublic`，落库值却被「个体任务强制 false」夹过 | 改用 `effectiveAllowPublic = allowPublic && target==0` 判定（防御 7.1 放宽结构锁后的耦合） |

第二轮变异项从 22 条增加到 25 条，全部 detected。另按审查建议修正了两处测试口径：

- 0143：改为由**真实编辑路径**的用例守护（此前只有 helper 直调）。
- 0145：注明「有提交后 `assertTaskEditable` 拒绝改 target，转换路径不可达」，已验证的只是公开墙侧排除个体任务，不能说成端到端已验证。
- 「老师审核锁内再校验园所/班级」是 TOCTOU 兜底，串行测试观测不到，代码保留但不计入变异。

### 12.11 第三轮审查（契约 / 接线 / 可见性等价，2026-10-22）

这轮换了切入面：不再只看 service 语义，而是逐字核对 `backend/api/` 契约、Casbin/路由接线，并把 `public_visible` 的判定与公开墙 SQL 做逐条对照。发现 2 个 CONFIRMED（同一根因的两面）、1 个契约分歧、3 处契约漂移：

| # | 问题 | 后果 | 修法 |
| --- | --- | --- | --- |
| 11 | 提交者**学生记录被软删**后，`locateCheckin`/`Preload` 拿到的班级为 0，而 `publicCheckinVisible` 把 0 当成「跳过班级校验」 | 老师 `checkins` 列表报 `public_visible=true`，公开墙的 `st.class_id = t.class_id` 却把该行排除——违反契约「`public_visible` 等价于公开墙会不会返回它」 | `submitterClassID == 0` 一律判不可见；补 `TestTeacherCheckinsPublicVisibleHidesDeletedStudent` |
| 12 | 同一状态下老师**撤回**返回 422（`TEACHER_INVALID_PARAMS`），因为提交者载入失败被当成参数错误 | 违反契约「撤回永远可用」；老师清不掉残留公开状态（管理端仍可撤） | 只在查库真出错时中断；学生缺失时班级为 0，批准自然被拒、撤回照常；补 `TestTeacherRevokeWorksAfterStudentDeleted` |
| 13 | §12.10 行 7 把「老师列表取班级」写成用 `locateCheckin`，实际是 `Preload("Student")` | 文档夸大了覆盖范围，正是本轮暴露 #11 的原因 | 已更正为真实实现 |
| 14 | 家长契约把 `public_status=approved` 定义为「已批准**且可见**」，但任务结束/关闭公开/转班后仍是 approved 而 `public_visible=false` | 客户端若用 `status` 判断上墙，会在任务结束后继续显示「已公开」 | 契约改为「审核状态」，并写明「是否在墙只以 `public_visible` 为准」 |
| 15 | 转班后申请公开返回 **404**（`ErrParentNotFound`），而契约把「任务仍属于孩子当前班级」列为允许条件并写明「不满足 → 422」 | 客户端按 404 显示「任务不存在」，与契约语义不符；测试还把错误码固化了 | 班级不再匹配 → `ErrTaskPublicNotAllowed`（422）；个体任务指向别的孩子仍按 404（不可见，不泄漏存在性）；更新测试断言 |
| 16 | 契约漂移：老师审核响应漏写 `task_id`；老师 checkins 漏写 B1 的 `response` 对象；家长申请/取消漏写 503 | 契约不再是准确的 source of truth | 三处契约补齐（实现不变） |

**复核对确认无问题的部分**：

- 公开墙与 `publicCheckinVisible` 的 `end_date` 口径等价（DATE 列 + 上海时区，`taskDayOf` 与字符串比较同一天）。
- 读者维度（`t.class_id = 读者孩子班级`）只属于墙查询，不属于 `public_visible`；后者回答的是「在该任务所属班级的墙上会不会出现」。
- 学生 `status`（在读/毕业/退学）两侧都不过滤——行为一致；若产品要求「毕业生不上墙」，必须同时改两处，否则会新造一条漂移。
- 公开墙不排除读者自己孩子的提交，D7 没有此要求（「本班家长可见」含自己在内）。
- Casbin 用 `keyMatch2` + `/api/v1/{parent,teacher,admin}/*` 通配，5 条新路由都在既有 policy 覆盖内；管理端权限缺口已在 `admin_perm.go` 登记并通过 `router_perm_test`。
- 路由无参数名冲突：`/public-wall`、`/checkins/:checkin_id/...` 与既有的 `/:id` 共存，`router_perm_test` 会构建真实路由树。
- 「编辑时无条件调用 revoke」已不是可观测缺陷（收窄后的 WHERE 清空后再调匹配不到），`UpdateTask` 里的条件只为少写无用 UPDATE；不变量由 `TestEditingTaskDoesNotRefreshRevokedAt` 守护。

### 12.8 B2 验证状态

- `go build ./...`、`go vet ./...`、`go test ./... -timeout 900s` 全绿（2026-10-22；跑全量前先停掉 `:8080` 的 `kg-server.exe`，否则识别队列测试会被抢任务）。
- `server/internal/service/task_public_test.go`：47 个真实 MySQL 用例；`_verify/mutate_b2.py` 28 项变异全部 detected。
- HTTP 冒烟（真实网关 + 中间件 + handler，2026-10-22）：公开墙 200、缺 `student_id` 422、不存在提交 404、管理端 `revoke=false` 422、跨班任务 404。
- TEST_CATALOG 的 0119/0120/0121/0134/0143/0144/0145 已从 ⏸ 改为「后端已验证 ⚠️」，UI 仍待 9.3/10.x/11.4。
- 未做：审核/申请/公开墙界面（9.3、10.x、11.4）；「含他人内容」只能靠老师勾选，无自动识别；`GET /parent/tasks` 与详情尚未返回 `public_*`（归 8.1）；`CheckinHistory` 仍按任务可见范围收窄（转班后家长查不到旧任务历史，但撤回接口可用）。

---

## 13. B2 开工清单（公开审核 + 公开墙）

### 13.1 起点状态

- B1 已提交并推送：`393a3e3 feat(tasks): B1 补充回应与家长反馈`（分支 `feat/parent-growth-scope`）。
- `openspec/.../tasks.md` 已是 3/89 勾选（3.5、3.6，另新增 3.11 待做）；B2 = **4.1–4.4**（4.5–4.7 是 P2）。
- 全量测试在 B1 收口时全绿；`go test ./internal/service -run 'TestSupplement|TestParentGetTask|TestReactToResponse|TestRitualPayload|TestDeleteTask'` 约 55s。
- 本地后端跑在 `:8080`（新构建），家长端 IDE 登录 user 61 / 学生 27，老师端登录 user 11（**小一班 class 1**）——两者不同班，跨端 UI 验证要造数，见 §12.5。

### 13.2 第一步：先把端点写进契约

`backend/api/` 目前**只声明**「公开相关属于 B 切片」，没有任何具体端点。按仓库铁律「改接口先改这里」，B2 的第一件事是把形状定下来，建议：

| 用途 | 建议路径 | 备注 |
| --- | --- | --- |
| 家长申请公开 | `POST /parent/tasks/checkins/:checkin_id/public-request?student_id=` | 校验提交属于当前孩子 |
| 家长取消申请 | `POST /parent/tasks/checkins/:checkin_id/public-cancel?student_id=` | 也可用 DELETE，二选一并写进契约 |
| 老师审核 | `PUT /teacher/tasks/:id/checkins/:checkin_id/public` | body `{approve, scope_confirmed}`；批准时 `scope_confirmed` 必须 true |
| 公开墙 | `GET /parent/tasks/public-wall?student_id=` | 只返回本班、仅含提交者孩子的内容 |
| 管理端撤回 | `PUT /admin/tasks/checkins/:checkin_id/public` body `{revoke:true}` | 走管理端权限登记表 |

> 管理端新增路由必须登记到权限注册表，否则 `NewRouter` 直接报错（`router_perm_test.go` 会红）。

### 13.3 状态机（design.md D7，逐字实现）

- 申请：`is_public_requested=true`、`public_scope_confirmed=false`、`public_approved_at=nil`、`public_revoked_at=nil`
- 批准：`scope_confirmed=true`、`public_approved_at=now`、`public_approved_by=teacher`
- 取消 / 撤回 / 关闭 allow_public：`is_public_requested=false`、`scope_confirmed=false`、`approved_*=清空`、`public_revoked_at=now`
- 再次申请：必须重新批准、重新勾确认，**不得沿用**旧批准或旧确认
- 可见条件（公开墙逐字照抄）：`allow_public && task.deleted_at IS NULL && task.status=1 && end_date >= 今天 && task.class_id = 当前孩子班级 && is_public_requested && public_scope_confirmed && public_approved_at IS NOT NULL && public_revoked_at IS NULL`
- 个体任务强制不可公开；一键完成（无文字无媒体）不可申请（P2 4.5）

### 13.4 必须遵守的既有约束

- **锁序**：任何任务写路径的第一条语句都是 `SELECT ... FOR UPDATE` 锁 `parent_tasks` 行，否则与 `deleteTaskCascade`（锁回应 → 删 checkin）成反序死锁。
- `BaseModel` 自带 `DeletedAt`，手写 `.Table()` 查询要自己补 `deleted_at IS NULL`；upsert 命中软删行要清 `deleted_at`。
- 已有 helper：`revokePublicForTask`（`task_rules.go`，清申请/批准/确认/审核人并写 revoked_at）——关闭 `allow_public` 时直接复用，不要重写。
- 老师审核的权限校验照 `SupplementResponse` 的写法：任务 `id + kindergarten_id + class_id`，提交 `id + task_id`。
- 错误码：参数/状态 422、跨家庭 403、并发关闭 409、锁冲突 503。

### 13.5 验证怎么做

- 后端：真实 MySQL 集成测试（照 `task_response_supplement_test.go` 的结构：固定 ID 夹具 + `Unscoped` 清理 + `clean()` 先跑一次）。**每条关键断言都要做变异验证**。
- 场景清单见 spec.md「活动公开需老师审核」的 7 个 Scenario（默认私密、申请后审核可见、跨班不可见、撤回与个体任务不可公开、关闭开关、更新为个体强制关闭、取消后重审）。
- UI：家长申请/老师审核的界面属 9.3、10.x，B2 仍然**没有界面可点**；只能走 HTTP + 必要时造数。造数用一次性 `cmd/tmp*` 程序，**用完立即删除**。

---

## 14. B3 落地内容（2026-10-23）

起点 `b3f2028`。按 §13.2 先改契约再写代码；只有后端，**没有界面**。

### 14.1 新增 / 变更

- 通知（5.1–5.3）：`server/internal/service/task_notify.go`
  - `POST /teacher/tasks` 发布后给目标家庭写 `type=task` 消息，`reference_id=task_id`；
  - `POST /teacher/tasks/:id/checkins/:checkin_id/response` 与 `POST /teacher/tasks/:id/batch-response` 按家长聚合写 `reference_id=response_id` 的消息；
  - 响应新增 `notified_count` / `notify_failed_count`（创建任务原本已有字段，现在返回真实值）。
  - 公共写入路径：`pickup_notify.go` 的 `sendMessageToRecipients`（支持每个收件人各自的 `action_path`）；`messages.action_path` 现在会通过 `/common/messages` 下发（`MessageItem.ActionPath`）。
- 小程序码（5.4）：`wechat.code_env_version`（默认 `release`；`server/configs/app.ini` 被 .gitignore 忽略，开发/测试要扫体验版/开发版时需在各环境本地配置）；`WeChatClient.GetParentWXACodeUnlimit(ctx, scene, page, envVersion)` 走家长端凭据；`GET /teacher/tasks/:id/share-card` → `{task_id, page, scene, env_version, image_base64}`（PNG base64，不落 OSS）。
- 补充回应未读（5.5）：`POST /parent/tasks/:id/read?student_id=`（upsert，返回 `{last_read_at}`）；任务列表/详情的 `has_unread_supplement` 现在按 `(任务, 家长, 孩子)` 实时计算。
- 契约：`backend/api/api-teacher/09-class-and-tasks.md`（回应计数 + share-card）、`backend/api/api-parent/06-task.md`（read 端点 + 未读口径）。
- 测试：`server/internal/service/task_notify_test.go`（18 条真实 MySQL）；`server/internal/auth/wechat_test.go` 新增 2 条（家长端凭据 / envVersion / 非法值）；`config_test.go` 新增默认版本断言；`_verify/mutate_b3.py` 15 项变异全部 detected（每条都人工确认是断言失败，不是编译失败）。

### 14.2 B3 的语义决定（后续不要改错）

- **消息判重带 title**：`type=task` 同时承载发布（`reference_id=task_id`）与回应（`reference_id=response_id`），两张表自增 id 会撞数；只按 `(user,type,reference_id)` 判重会吞掉正常通知。`MessageRepo.Exists` 现在额外匹配完整标题，两类前缀不同（`新亲子任务：` / `老师回应了：`）。
- **通知对象是「家庭」不是「提交人」**：主回应通知发给被覆盖孩子的全部家长（co-parent 也收）；计数按家长去重。
- **深链的孩子选择**：发布时班级任务取该家长在本班 `student_id` 最小的孩子（确定、稳定）；回应时取**第一个被覆盖的提交**所属的孩子（按 checkin id 升序），与 D9 逐字一致。
- **批量回应只通知写成功的提交**（`RowsAffected=1` 的那些）；失败提交继续保持待回应、不通知。
- **action_path 只允许 `/pages/` 开头且不含 `://`**；写路径会拒绝并计入失败，不会落库。
- **通知失败不回滚**：所有通知在事务提交之后调用，失败只记日志 + `notify_failed_count`。
- **读标记时间截断到毫秒**：列是 `datetime(3)`，MySQL 对不足毫秒的部分四舍五入；不截断时 `.0007s` 的读会存成下一秒，误清同毫秒的补充未读。
- **读标记 upsert 要复活软删行**：唯一索引包含软删行，`ON DUPLICATE KEY UPDATE` 必须同时清 `deleted_at`，否则红点清了又回来。
- **未读用 `>=`**：补充回应 `created_at >= last_read_at` 即未读（同毫秒不漏判）。
- **读接口的家长绑定由 handler 的 `parentTaskCtx` 保证**：它走 `ResolveContext`（只加载当前家长绑定的孩子）并校验 `CurrentStudent.ID == student_id`；service 与 `GetTask` 一样信任已解析的 `ParentPortalContext`。
- **share-card 必须走家长端凭据**；`envVersion` 空值取配置，非法值在发 HTTP 前报错（静默回退会生成另一个版本的码）。

### 14.3 限制性对抗性审查（2026-10-23）

| # | 级别 | 问题 | 处置 |
| --- | --- | --- | --- |
| 1 | CONFIRMED | 家长端 `pages/tasks/detail.vue` 的 `onLoad` 不解析 `scene`，扫码进来只有 `scene` 没有 `id` | 未修：属 10.1（前端），记入 14.5 |
| 2 | CONFIRMED | 家长端全仓不消费 `has_unread_supplement`，也不调用 `POST /parent/tasks/:id/read` | 未修：属 10.2/10.3（前端），记入 14.5 |
| 3 | CONFIRMED | `pages/messages/index.vue` 不消费 `action_path`，消息卡片没有点击跳转 | 未修：属 10.6（B4），记入 14.5 |
| 4 | POSSIBLE | `last_read_at` 亚毫秒精度被 MySQL 四舍五入，可能误判同毫秒补充为已读 | ✅ 已修：写库前 `Truncate(time.Millisecond)` |
| 5 | POSSIBLE | 目标家长查询失败时返回 `0/0`，与「班里没有家长」不可区分 | 保留：错误已 `log.Error`（含任务/回应 id），且不得把通知故障升级成业务失败；API 计数不伪造 |
| 6 | NOTE | 用标题当判重命名空间较脆弱，长期应改成 `reference_type + reference_id` | 记录；当前两类前缀稳定，改动需同步 domain/表结构 |
| 7 | NOTE | 5.3 的「部分失败」原先只覆盖全失败（关掉消息库） | ✅ 已补 `TestSendMessagePartialFailureCountsPerRecipient`：用超长 `action_path` 让一个收件人失败、另一个成功，断言 sent/failed 分别计数且失败方不留半条消息 |
| 8 | NOTE | 管理端 `POST /admin/tasks` 发布不发通知 | 属 C 切片（`task_service.go` 无消息依赖）；口径待 7.x 确认 |
| 9 | NOTE | 班级任务通知不按学生 `status` 过滤 | 与家长端任务可见性一致（也只按班级），若产品要求「毕业生不通知」，两处必须同时改 |

#### 第二轮（干净 subagent，2026-10-23）

| # | 级别 | 问题 | 处置 |
| --- | --- | --- | --- |
| 10 | CONFIRMED | 变异脚本有两条「假检出」：把 `isInternalActionPath` 改成 `return true` 会让 `strings` 变成未使用，把详情红点改成 `false` 会让 `unread` 未使用——两者都是编译失败，却被当成测试检出 | ✅ 已修：变体改成仍使用 `strings` / `unread`（`false && unread[...]`）；新增「列表红点」变体；重跑 15/15，并人工确认两条变体输出是 `Error Trace`（断言失败）而非 build error |
| 11 | POSSIBLE | `share-card` handler 把所有错误折叠成固定 500 且不打日志；契约写明「配置非法时错误信息要说明」，但具体原因被丢弃 | ✅ 已修：`auth.ErrInvalidCodeEnvVersion` 哨兵 + handler 分别返回「版本配置非法」；所有非 404 错误先 `log.Error` 再回 500 |
| 12 | NOTE | `POST /parent/tasks/:id/read` 的 default 分支静默回 500 | ✅ 已修：handler 记录 `parent mark task read failed`（含 task id 与 err）后再回 500 |
| 13 | NOTE | 周期汇总回应之后再追加补充的未读路径此前没有用例 | ✅ 已补 `TestUnreadSupplementUnderPeriodResponse`（A1 未读、A2 不受影响） |
| 14 | NOTE | `GenerateTaskShareCard` 没有服务层用例（只有 auth 包的凭据/env 用例） | ✅ 已补 `TestTaskShareCardRejectsForeignOrDeletedTask`（跨班 404、软删 404；正向生成依赖真实微信接口，仍不在单测内） |
| 15 | NOTE | `Truncate(time.Millisecond)` 的精度修复没有测试；目标查询失败仍返回 `0/0`；任务创建允许 `end_date` 已过但仍立即发通知，家长点进去 404 直到窗口开始 | 记录：精度修复靠 `datetime(3)` 事实 + 代码审查；`0/0` 已由日志兜底；过期日期属创建校验（2.x/C 切片），本次不改 |
| 16 | NOTE | 判重加入 title 也改变了晨检通知的幂等口径（同一 check 若结论变化会重发） | 现有 `SubmitCheck` 是一次性的（重复提交直接拒绝），无实际回归；记录在案 |
| 17 | NOTE | 每个夹具 `setupB1/2/3` 都新开一个 `gorm.Open` 连接池且不关闭，全量 suite 会累积连接 | 既有模式，非本次引入；记录 |

### 14.4 验证状态

- `go build ./...`、`go vet ./...` 通过（2026-10-23）。
- `go test ./internal/service -run 'TestTaskPublish|TestBatchResponse|TestSingleResponse|TestTaskAndResponseSurvive|TestSupplementUnread|TestReadMark|TestTaskActionPath|TestSendMessage|TestMessageDedupe'` 全绿。
- `go test ./internal/auth ./internal/config ./internal/app` 全绿（含新路由表的冲突检查）。
- `_verify/mutate_b3.py`：15 项变异全部 detected；其中「放宽 `/pages/` 校验」「列表红点恒 false」两条已人工回读输出，确认是断言失败而不是包编译失败。
- 全量 `go test ./... -timeout 900s`：**全绿**（2026-10-23 第二轮修复后重跑，exit 0；`internal/service` 492s，`internal/ai/face` 1.9s、`internal/app` 3.9s、`internal/auth` 3.8s、`internal/config` 0.5s、`internal/middleware` 3.0s、`internal/rbac` 1.3s、`internal/repository` 10.0s 均 ok；跑前已确认本机 :8080 无 LISTEN）。
- TEST_CATALOG 0124/0125/0126/0140 已从 ⏸ 改为「后端已验证 ⚠️」。

### 14.5 B3 未做的部分（不要当成已验证）

- **家长端**：`scene` 解析（10.1）、任务列表/详情红点与打开清除（10.2/10.3）、消息点击按 `action_path` 跳转（10.6，B4）。
- **老师端**：保存分享码到相册 / 转发提示（9.5，B4）、真机扫码进入对应版本（12.6）。
- **管理端**：管理端创建任务的发布通知（C 切片）。
- 目标家长查询失败的 API 级可观测（见 14.3 #5）；任务级「一个家长成功、一个失败」未做（共享写路径已由 #7 覆盖）。

---

## 15. 下一窗口开工（B4 前端收口，2026-10-23 准备）

### 15.1 起点状态

- 分支 `feat/parent-growth-scope`，HEAD `26a853c`（已 push；远端此前停在 `393a3e3`，本次连带 B1/B2 一起推了）。
- 后端：5.1–5.3、5.7 已勾选；**5.4 / 5.5 的后端已完成但未勾选**——它们的验收包含家长端 `scene` 解析、红点/读标记、真机扫码，这些在本窗口闭合后再勾。
- 全量 `go test ./... -timeout 900s` 全绿（service 492s）；B3 用例 18 条；`_verify/mutate_b3.py` 15/15 **真实**检出。
- 前端是**旧 A 切片页面**：B1（补充回应/反馈）、B2（公开审核/公开墙）、B3（未读/分享码）的界面都还没接。
- 本地后端当前未跑；改前端后必须先 `npm run dev:mp-weixin` 再验证（IDE 读 `dist/dev/mp-weixin`，不是源码目录）。

### 15.2 前端现状（2026-10-23 逐文件核实，别凭任务描述假设）

| 文件 | 已有 | 缺 |
| --- | --- | --- |
| `teacher/src/pages/tasks/detail.vue` | 单条/批量回应、贴纸/语音/星星、待回应/已回应筛选、编辑、下拉刷新 | 补充回应入口、公开审核、分享码、星星罐目标设置 |
| `teacher/src/pages/tasks/index.vue` | 列表、进入详情 | 创建/编辑表单的新字段（学习关联、范围、频率、仪式、星星、公开）→ 9.2 |
| `teacher/src/api/teacher-tasks.js` | 列表/详情/建改删/统计/checkins/回应/批量 | `supplement`、`public` 审核、`share-card` → 9.1 |
| `parent/src/pages/tasks/index.vue` | 列表、按孩子取数（`withStudentScope`）、进度 | 未渲染 `has_unread_supplement` 红点；状态/进度口径待核对 → 10.2 |
| `parent/src/pages/tasks/detail.vue` | 详情、提交、打卡历史、主回应简版 | `scene` 解析、补充回应渲染、`my_reaction`、公开申请/取消、打开调 `read`、「给孩子看」→ 10.1/10.3 |
| `parent/src/pages/messages/index.vue` | 消息列表 | 卡片点击、`action_path` 跳转、任务不存在/结束提示 → 10.6 |
| `parent/src/api/parent-task.js` | 列表/详情/提交/历史（已带 `student_id`） | `markTaskRead`、`reactResponse`、公开申请/取消/公开墙、`share-card` 相关无 |

> ⚠️ 两端 `components/` 会被 `scripts/sync-ui.js` 覆盖；**改组件只改 `shared-ui/`**，再同步。

### 15.3 建议窗口拆分（一次只做一组，组间先审查再验证）

| 窗口 | 范围 | 目的 |
| --- | --- | --- |
| **B4-a（建议下一个开）** | 10.1、10.2、10.3、10.6 + 9.1 的 `share-card` 封装 + 9.5 分享按钮 | 闭合 5.4/5.5 验收，并把 B1/B2 的家长端界面补上 |
| B4-b | 9.1 余下、9.2、9.3（补充回应 + 公开审核）、9.7 | 老师端完整闭环 |
| B4-c | 10.4、10.5、10.7、11.x | 首页卡片、反馈按钮、管理端 C 切片 |
| 收尾 | 8.x 勾选、12.3/12.4/12.6/12.7/12.8 | 真机/回归/归档前验证 |

### 15.4 B4-a 实现要点

1. **10.1 API 与 scene**
   - 新增 `markTaskRead(taskId, studentId)` → `POST /parent/tasks/:id/read?student_id=`；
     `reactResponse(responseId, reaction, studentId)` → `POST /parent/tasks/responses/:response_id/reaction?student_id=`；
     `requestPublic(checkinId, studentId)` / `cancelPublic(checkinId, studentId)` / `fetchPublicWall`。
   - `onLoad(opts)` 解析小程序码：`scene` 形如 `t=123`（可能 URL-encoded）→ `decodeURIComponent` → 取 `t`；
     `id = Number(opts?.id || sceneTaskId || 0)`，两者都没有才报错。
   - 首页入口若也带 `scene`，同样处理（10.1 原文包含首页入口）。
2. **10.2 列表红点**
   - 用 `item.has_unread_supplement` 显示任务级红点（与消息未读是两套，不要合并）。
   - `onShow` 重新拉列表（详情返回后红点要消失）；切换孩子后所有请求带新 `student_id`。
3. **10.3 详情**
   - 渲染 `responses[].supplements`：贴纸 key → 图形/文案、语音 → 播放、文字 → 文本；`my_reaction` 回显。
   - 反馈三选一（`child_happy` / `received` / `thanks`），同一按钮再点即更新；反馈按家长账号，不按孩子收窄。
   - 公开开关只在 `allow_public && target_student_id === 0` 时显示；申请后用 `public_status`（审核状态）与
     `public_visible`（此刻是否在墙）**分别**表达，不要用 status 冒充「已上墙」。
   - **数据加载成功后**调 `POST read`，失败只 toast/日志、不阻塞渲染；多孩时用当前 `student_id`。
   - 「给孩子看」全屏态：放大贴纸/播放语音/老师的话（10.3 原文）。
4. **10.6 消息深链**
   - 只接受 `/pages/` 开头且不含 `://` 的 `action_path`；跳转时传 `id` 与 `student_id`。
   - 任务页不是 tabBar 页 → `uni.navigateTo`；家长端 tabBar 只有 index/growth/pickup/profile。
   - 任务删除/结束（详情 404）显示「任务已结束/不存在」，不要白屏。
5. **9.5 分享码**
   - `fetchTaskShareCard(taskId)` → `{ task_id, page, scene, env_version, image_base64 }`。
   - base64 先用 `uni.getFileSystemManager().writeFile({ encoding: "base64" })` 落临时文件，再 `uni.saveImageToPhotosAlbum`；
     失败提示「请检查家长端微信配置」，不要把微信错误直接抛给老师。
   - 码必须由后端家长端凭据生成（已保证）；真机扫码验证属 12.6，需要本地 `wechat.code_env_version` 配 `trial`/`develop`。

### 15.5 验证与收工

- `npm run build:mp-weixin`（teacher + parent）通过；`node scripts/check-tokens.mjs teacher|parent` 对改动文件零新增违规。
- 交互必须真实点击（`wechat_automator` / CDP），不能用 `wechat_navigate` 代替；导航跳转的 `switchTab`/`navigateTo` 差异只在真实点击下暴露。
- 至少走通：家长任务红点 → 打开 → read 清除 → 看到补充回应 → 反馈 → 申请公开；
  老师生成分享码 → 保存相册 → 真机扫码进入**家长端**任务详情（12.6）。
- 不提交 `docs/competitive/*`、`skills-lock.json`、`.impeccable/`。
- 完成后：更新本文件 §16、勾选 5.4/5.5（只有在 UI 与扫码都验证后）及对应 9.x/10.x，再考虑 archive。

### 15.6 已知坑（沿用，别重踩）

- `wx_cdp.mjs click <sel> mouse` 必须带 `mouse`，否则点不动 uni 的 `@click`；DOM 标签是 `wx-view`/`wx-text`/`wx-button`。
- 单 IDE 实例下双端不能并行；要跨端联调走 `scripts/dual-ide.ps1`（两个微信号、9222/9223、9420/9421）。
- 老师端**没有 tabBar**；家长端 tabBar 是 index/growth/pickup/profile，跳这些页面必须 `switchTab`。
- 改 AppID / `project.config.json` 后必须重新构建；占位 AppID 会让自动化端口静默不监听。
- 前端组件改 `shared-ui/`，不要改两端 `components/`（会被同步脚本覆盖）。

### 15.7 新窗口开工提示词（直接复制）

```text
读 docs/handoffs/2026-10-21-parent-task-loop.md 的 §15，按 §15.3 的 B4-a 范围开工：
家长端 10.1 / 10.2 / 10.3 / 10.6（scene 解析、补充回应渲染与反馈、任务红点与读标记、消息 action_path 跳转），
外加老师端 9.1 的 share-card 封装与 9.5 分享按钮。

开工前先读：design.md 的 D9/D10、backend/api/api-parent/06-task.md、backend/api/api-teacher/09-class-and-tasks.md、
以及 openspec/changes/2026-10-20-redesign-parent-task-loop/tasks.md 的 9.x/10.x。

约束：
- 不要动 A/B1/B2/B3 已落地的后端语义（尤其任务锁序、公开状态机、通知按家长聚合、读标记口径）。
- 前端组件只改 shared-ui/，不要改两端 components/；样式只能引用 shared-design-tokens 的 token。
- 家长端所有任务接口必须带 student_id；老师端没有 tabBar，家长端 tabBar 只有 index/growth/pickup/profile。
- 交互验证必须真实点击（wechat_automator / CDP），不能用 wechat_navigate 代替；改完先 npm run dev:mp-weixin。
- 别提交 docs/competitive/*、skills-lock.json、.impeccable/。
- 完成后：两端 build + check-tokens，更新 handoff §16，并在 UI 与真机扫码都验证后再勾选 5.4/5.5 与对应 9.x/10.x。
```

---

## 16. B4-a 落地内容（2026-10-23）

### 16.1 改了什么（逐个文件）

| 文件 | 变更 |
| --- | --- |
| `parent/src/api/parent-task.js` | 新增 `markTaskRead` / `reactResponse` / `requestPublicCheckin` / `cancelPublicCheckin` / `fetchPublicWall`；`student_id` 一律拼在 query（body 里的会被后端丢弃），读标记走 `silent`（失败不影响渲染） |
| `parent/src/pages/tasks/detail.vue` | `onLoad` 解析小程序码 `scene`（`t=123`，支持 URL 编码）→ 缺省回退 `id`；渲染 `responses[].supplements`（贴纸图形/语音播放/文字）；主回应与补充回应各三个反馈按钮并回显 `my_reaction`；打卡历史的公开申请/撤下（含 `public_visible` 与审核状态分开表达）；「给孩子看」全屏态；加载成功后调 `POST read`；消息目标孩子与当前孩子不一致时给提示 + 可切换 |
| `parent/src/pages/tasks/index.vue` | `has_unread_supplement` 任务级红点 + 文案；状态改「待完成/已提交」；进度按「本周期是否已提交」；`累计打卡 N 次`；进行中 tab 待办数；移除本地 `$purple-*`，全部改 token |
| `parent/src/pages/messages/index.vue` | 卡片按 `action_path` 跳转；只接受 `/pages/` 且不含 `://`；tabBar 四页走 `switchTab`，其余 `navigateTo`（失败提示「该内容已不存在」）；`onLoad` → `onShow` |
| `parent/src/pages/index/index.vue` | `quickActions` 改 `computed`，「亲子任务」红点取后端修正后的 `pending_tasks` |
| `teacher/src/api/teacher-tasks.js` | 新增 `fetchTaskShareCard` |
| `teacher/src/pages/tasks/detail.vue` | 分享卡：生成家长端小程序码 → base64 落临时文件 → 存相册；写入/相册各有超时封顶 |

### 16.2 语义决定（后续不要改错）

1. **列表不显示「已回应」**：契约 `06-task.md` 的任务列表只返回 `submitted_current_period` / `checkin_count`，没有回应字段。10.2 原文要三态，但补这个字段属于后端 + 契约改动，不在 B4-a 范围，且会动已经被 B1/B2 测过的读路径。现状＝列表两态（待完成/已提交），详情页按 `responses` 表达已回应。**要补就等有人愿意同时改 `06-task.md` + service + 单测**，不要在列表里猜。
2. **撤下公开用页面内两步点击，不用 `uni.showModal`**：原生弹窗不进视图层 DOM，CDP 既看不到也点不到，「撤下已公开内容」这条流程因此无法用真实点击验证（实测第一次点确实卡住不回调）。两步点击（`取消公开` → `确认撤下？`）保持显式确认，且可自动化。
3. **任务详情时间按本地时区渲染**：后端是 UTC RFC3339，直接截字符串会显示成「凌晨」。`shortTime` 用 `new Date(iso)` 转本地。
4. **`quickActions` 必须是 computed**：原先静态数组在 setup 时 `summary` 还是空的，`badge` 表达式只求值一次，红点永远不出现。该修复同时让「通知公告」红点开始生效（同一表达式，本就是原意）。
5. **分享码保存有超时封顶**（写入 8s / 相册 15s）：相册授权是系统弹窗，用户不理它时 `saveImageToPhotosAlbum` 永不回调，按钮会一直「生成中…」。没有封顶就是死锁。
6. **公开按钮的显示口径**：只有 `allow_public && target_student_id === 0` 才给入口；一键完成（无文字无媒体）不给「申请」，但「撤下」永远可用（对齐后端「撤回永远可用」）。

### 16.3 验证证据（真实点击，非导航替代）

后端 `make run`（8080）+ IDE 单实例，CDP 走 `scripts/cdp.py` / `scripts/focus-ide.ps1`。

| # | 动作 | 结果 |
| --- | --- | --- |
| 1 | 首页点「亲子任务」（`.action-item`） | 进入 `pages/tasks/index` |
| 2 | 列表读渲染 | 新任务显示「待完成 / 本周期待提交 / 一次性任务 · 累计打卡 0 次」，进行中 tab 带待办数 1 |
| 3 | 点「查看详情」 | 进入 `detail?id=4322&student_id=27`，`requirements` 渲染成 1./2. 两条，学习关联卡正常 |
| 4 | 点「今天完成了（一键提交）」 | 提交成功，提交卡消失，历史出现当日记录 |
| 5 | 库里补主回应 + 补充回应后返回列表 | 任务卡出现 `.tc-dot`（1 个）与「老师又给你留了一句悄悄话」 |
| 6 | 打开详情 | 未读提示条 + `⭐` 贴纸 + 老师的话 + 补充回应 `🌸`；`task_read_marks.last_read_at` 被写成打开时刻 |
| 7 | 返回列表 | 红点消失（`dots=0`），onShow 重取生效 |
| 8 | 点「孩子很开心」 | `task_response_reactions` 落到 `(699, 61, child_happy)`；按钮 `active`；重进详情回显 |
| 9 | 点「给孩子看」 | 全屏遮罩显示放大 `⭐` + 老师的话；点空白处关闭（`Input.dispatchMouseEvent` 打点） |
| 10 | 待完成任务点「去提交」→ 有内容提交 → 历史点「申请上班级墙」 | 状态变「已申请，等待老师审核」，按钮变「取消公开」；库里 `is_public_requested=1` |
| 11 | 点「取消公开」→「确认撤下？」 | 状态变「已撤下」；库 `is_public_requested=0`、`public_revoked_at` 有值 |
| 12 | 点「重新申请」 | 回到待审核；库 `revoked_at` 被清空（不沿用旧批准） |
| 13 | 消息页点任务消息卡片 | 跳到 `detail?id=4322&student_id=27`；消息里带 `查看 ›`，系统消息无入口 |
| 14 | 只带 `scene=t%3D4322` 打开详情 | 正确加载任务 4322（小程序码编码串可解析） |
| 15 | 打开不存在的任务 id | 显示「任务已结束或不存在」+ 重试，不白屏 |
| 16 | 消息目标孩子 ≠ 当前孩子（`student_id=32`） | 顶部提示「这条消息是另一个孩子的任务…」+「切换查看」 |
| 17 | 首页在存在待办任务时刷新 | 「亲子任务」入口出现 `.action-badge` |
| 18 | 老师端任务详情点「生成小程序码」 | 接口 200 + `image_base64`；PNG 落到 `USER_DATA_PATH/task-share-4324.png`（82521 字节）；相册授权在 IDE 里不回调，靠 15s 封顶把按钮放回可点 |

两端 `npm run build:mp-weixin` 通过；`node scripts/check-tokens.mjs parent|teacher` 对本次改动文件 **0 违规**。

### 16.4 未验证 / 缺口（不要当成已验证）

- **5.4 / 5.5 / 9.5 / 10.1–10.3 / 10.6 都没勾**：按 §15.5 的要求，勾选要等真机扫码（12.6）。IDE 里相册弹窗不回调，扫码那一段只能在真机上做。
- **未登录保留 action_path、登录后回跳**：IDE 里重新打开项目会清 token，但没构造「带着深链被踢去登录」的用例。
- **10.7（点消息顺带标记消息已读）** 仍属 B4-c，本次没动消息读状态。
- **列表「已回应」**：见 16.2-1，需要后端补只读字段才能做。
- **本地验证数据未清理**（都在开发库，园所 4）：`parent_tasks` 4322（class 4，回应+补充回应夹具）、4323（class 4，公开申请/撤下过）、4324（class 1，分享码）、4325（class 4，待完成）；补充回应 = `task_responses` 700（`response_type=single`）+ `task_response_supplements` 指向 checkin 1953；`messages` 1171 带 `action_path`；`task_read_marks` 131。要重跑验证可直接用这些 id，要清就从后往前删。
- 后端是本窗口 `go run` 起的（8080），收工时已停；下次验证要重新 `make run`。

### 16.7 第二轮对抗性审查（2026-10-23，针对已提交的 8b5a00f）

换视角再查一遍：不看「我实现了什么」，只看「什么情况下用户会看到错的东西 / 点了没反应 / 说明不清」。

| # | 缺陷 | 处置 |
| --- | --- | --- |
| 1 | **多孩提示条说不清「你看的是哪个孩子」**：`currentChildName` 只从打卡历史里取 `student_name`，而任务发布通知最常见的场景就是「还没提交」——历史为空，提示条退化成「当前孩子」。D9 最后一条要求提示的目的正是「避免多孩家庭误以为看错内容」，退化成这个样子等于没提示 | 已修：错配路径上用 `GET /parent/home/overview` 兜底一次姓名（仅这条罕见路径多一个请求，不污染正常加载）；历史里的名字与兜底名字分开存，互不覆盖 |
| 2 | **同一个提示条在 404 路径上仍是「当前孩子」**：任务对当前孩子不可见（个体任务、跨班）时详情 404，走的是 catch 分支，不经过上面的兜底 | 已修：catch 里同样解析姓名 |

补验了之前没点过的分支（真实点击）：
- 「切换查看」真的点下去：`parent_current_student_id` 变成 32，提示条消失，页面按新孩子重新取数（提交卡出现、历史为空）；随后手动切回 27。
- 补充回应的「给孩子看」：全屏态显示的是**补充回应**的 🌸 与文字，不是主回应的。
- 补充回应的反馈：写库 `(700, 61, received)`，UI 主回应/补充回应各自 active，互不串。
- `pages/index/index`、`pages/tasks/index`、`pages/tasks/detail`、`pages/messages/index` 四页 CDP 采集 console：无未捕获异常。

另核对（未发现问题）：`checkin-history` 确实返回 `is_public_requested / public_approved_at / public_revoked_at / public_visible / student_name`，前端的审核状态推导与契约 `public_status` 映射一致；`new_notice_count` 是 int64（不是数组），首页红点不会因空数组恒亮。

### 16.8 独立审查（fresh context）与处置（2026-10-23）

另起一个全新上下文的只读审查者看 `26a853c..HEAD` 的 diff + 后端契约，报 6 条确认 + 3 条存疑。
**我逐条对源码复核后才动手**，不是照单全收。

| # | 审查结论 | 复核 | 处置 |
| --- | --- | --- | --- |
| 1 | 消息页未读数取 `u.data.count`，而接口返回 `{total, by_type}` → 「N条未读」永不显示；且全端没有标记已读的调用 | 成立（message_service.go 的 `UnreadCountResult`；router 只有 `PUT /common/messages/:id/read` 与 `/read-all`） | 已修：`total`；**把 10.7「点消息同时标记已读」提前做了**（点开即已读、失败只记日志）；顺手修掉 `updateMessage` 指向不存在的 `PUT /common/messages` 的死路由 |
| 2 | 切换孩子时详情页无请求序号，旧孩子的响应会覆盖新孩子（个体任务场景下会把页面刷成「任务已结束或不存在」） | 成立（`loadData` 确实没有 seq；提示条刻意渲染在 `v-if="loading"` 之外，按钮在加载中可点） | 已修：`loadSeq`，赋值前判 `seq !== loadSeq`，`finally` 只在最后一次请求时收 loading |
| 3 | `fetchPublicWall` 无任何调用点、没有墙页面/入口，但详情页写着「已在班级墙展示」 | 成立（grep 只命中定义；pages.json 无墙页面） | **不新建页面**（会带设计决策、且不在任何切片里），改为不承诺目的地：文案改「老师已通过，本班家长可见 / 暂不对本班展示」，按钮改「申请公开」。**公开墙页面仍是未交付项**，见下 |
| 4 | 打卡/一键提交无防连点，第二次 422 会在「已提交」后面跟一条失败提示 | 成立 | 已修：`submitting` 闩 + 按钮 busy/disabled |
| 5 | 消息页一次刷新失败就把已有列表整块换成错误页 | 成立（我上一轮改成 `onShow` 重拉，把这个问题放大了） | 已修：只有列表为空才整页报错，否则内联提示 |
| 6 | 切换孩子后打卡草稿（文字/照片）不重置，可能记到另一个孩子名下 | 成立 | 已修：`draftSid` + `resetDraft`，`loadData` 发现孩子变了就清 |
| 存疑1 | 分享码空图早退分支漏 `hideLoading`，遮罩会一直挂着 | 成立（我加的早退 `return` 绕过了唯一的 hide 调用） | 已修：`hideLoading` 收到 `finally` 里统一执行 |
| 存疑2 | 首页 `pending_tasks` 用进程时区、任务列表用上海时区，0:00–8:00 可能差一天 | 后端既有代码，无部署 TZ 证据 | **未改**，记录待查 |
| 存疑3 | 消息列表 20 条截断无「还有更多」提示 | 既有行为 | 未改，记录 |

复验（真实点击，视图层读取改用 `_verify/wxc.py --host <port>`：`cdp.py` 会读到已销毁页面的僵尸 pageframe，本轮就踩了一次，差点把「未读数没更新」误判成 bug）：

- #1：消息页显示「2条未读」+ 2 个红点 → 点任务消息 → 库里 `is_read=1, read_at` 有值 → 返回列表显示「1条未读」。
- #5：停掉后端后下拉刷新 → 2 张卡片仍在，顶部内联「网络异常，点击重试」，不再整块变错误页。
- #6：在 A 的详情里注入草稿 → 点「切换查看」到 B → 输入框为空、提交卡正常。
- #3：把一条提交重新置为「已批准且可见」→ 详情显示「老师已通过，本班家长可见」。
- 回归：`messages` / `tasks` / `tasks/detail` 三页 console 无未捕获异常；两端 build 通过；tokens 0 违规。

**没能观测到的（不要当成已验证）**：
- #2 的竞态本身没有复现（本地接口太快，制造不出「旧响应后到」的窗口），只验证了正常加载与切换不回归，`loadSeq` 是按老师端既有写法补的。
- #4 的闩没有在「请求在飞」的窗口里被真实命中（同样太快），只验证了连点两次仍只落 1 条提交。

**仍未交付**：公开墙页面（`fetchPublicWall` 已封装、无调用方）。它不在 B4-a/B/C 任何一个任务里，但 12.4 的「班级墙可见」验收需要它；要么在 B4-c 补一个最小列表页，要么明确从切片里拿掉。

### 16.5 下一步

B4-b：`9.1` 余下（supplement / public 审核封装）+ `9.2`（创建/编辑表单新字段）+ `9.3`（补充回应与公开审核 UI）+ `9.7`。老师端目前只能用 A 切片的老表单，创建任务时新字段还填不了。

### 16.6 对抗性审查（2026-10-23，针对 16.1 的 diff 本身）

按「怎样让它坏」的方向逐条过自己写的代码，找到 4 条真缺陷，3 条已修并复验，1 条只报告不动手。

| # | 缺陷 | 性质 | 处置 |
| --- | --- | --- | --- |
| 1 | **关掉 `allow_public` 后家长撤不下已公开内容**：`showPublicAction` 先判 `canPublic()`，老师一关公开开关，按钮整个消失。契约 D9 / 4.1 写的是「撤回永远可用，这些条件只约束申请」 | 违反契约 | 已修：`pending/approved` 直接给撤下入口，只有「申请」受 `canPublic()` 与「有内容」约束 |
| 2 | **深链落在 404 时，「换孩子」出口跟着消失**：提示条原来写在 `<template v-else>` 里，任务对当前孩子不可见（个体任务、跨班）时详情 404，家长只看到「任务已结束或不存在」，没有任何切换入口 | 10.6 的多孩场景 | 已修：提示条移到加载/错误分支之外；`currentSid` 在 `onLoad` 先用本地当前孩子兜底 |
| 3 | **`revoked` 文案把老师驳回说成家长撤下**：后端「老师拒绝/撤回」与「家长取消」写的是同一个 `PublicRevokedAt`，UI 却写死「已撤下」 | 语义误导 | 已修：改成中性的「未公开」 |
| 4 | **分享码接口返回 200 但没图时静默返回**：`stage === "generate"` 分支默认「http 层会弹提示」，只有当接口非 2xx 才成立；`image_base64` 为空时什么都不显示 | 静默失败（违反 CLAUDE.md） | 已修：空图单独 toast「小程序码为空，请检查家长端微信配置」 |
| 5 | ~~已结束任务无法撤下公开内容~~ **（查证后撤回）**：`GetTask` 对 `!taskIsActive` 返回 404，详情进不去 | 起初判为「契约与可达性冲突」，**证据不支持** | **不改**。spec 第 178 行「任务结束后公开内容也不再展示」→ 内容本来就不在墙上；第 209 行「任务删除/结束有提示」→ 详情 404 是有意行为。`DELETE public-request` 不限制任务状态是接口层兜底（防并发/直连），不是要求 UI 给已结束任务留入口。改 `GetTask` 的可见性会推翻 A 切片语义并牵动 2.5 的单测，收益为零 |

复验（真实点击 + 库核对）：
- #1：`task 4323 allow_public=0` + `checkin 1954 approved` → 详情显示「老师已通过，暂未展示在班级墙」且**有**「取消公开」按钮；两次点击后 `is_public_requested=0 / approved=null / revoked_at 有值`，文案变「未公开」，之后再申请入口按规则隐藏。改前该场景按钮数为 0。
- #2：`detail?id=4324&student_id=32`（class 1 任务，当前孩子 class 4）→ 页面同时显示「这条消息是另一个孩子的任务…切换查看」和「任务已结束或不存在」。
- #3 / #4：随 #1、分享码链路一并观察，编译产物已含新分支。
- 回归：`detail?id=4322` 回应/补充回应/反馈回显（6 个反馈按钮，1 个 active）未受模板结构改动影响；两端 `npm run build:mp-weixin` 再次通过。

### 16.9 班级公开墙页面（2026-10-23，补 §16.8 的缺口 3）

`fetchPublicWall` 之前只有封装没有调用方，详情页却在说「已在班级墙展示」。现在补齐：

| 文件 | 变更 |
| --- | --- |
| `parent/src/pages/tasks/wall.vue`（新） | 班级墙列表：按**提交日期**分组（今天 / 昨天 / M月D日），有图的渲染照片、没图的用品牌浅底区分成「一句话」；头像缺失时用姓名首字兜底；点图 `uni.previewImage` |
| `parent/src/pages.json` | 注册 `pages/tasks/wall`（`enablePullDownRefresh`） |
| `parent/src/pages/tasks/index.vue` | 列表顶部加「班级墙 / 看看本班孩子的分享」入口（`navigateTo`，非 tabBar 页） |

状态覆盖：首屏加载 / 空态（「本班还没有分享」+ 一句说明）/ 整页错误（列表为空时给重试）/ 内联错误（已有数据时只提示，不吞列表）/ 分页加载中 / 没有更多。
数据用 `withStudentScope` 带当前孩子 `student_id`，`page_size=20`，`onReachBottom` 翻页、`onPullDownRefresh` 刷新。

验证（真实点击 + 截图）：
- 从任务列表点入口进入 `pages/tasks/wall`，渲染出「今天 / 今天陪小俊彦读了《小熊很忙》… / 小俊彦 / 10:00 / 【B4验证】公开申请 10-23 / 没有更多了」。
- 把该提交改成已撤销后重进 → 「本班还没有分享 / 家长提交、老师确认之后，会出现在这里」。
- 停后端重进 → 「网络异常」+ 重试。
- 改回已批准（夹具恢复），两端 build 通过、tokens 0 违规。

未验证：**加载更多**（本班目前只有 1 条已批准内容，翻不到第二页；`onReachBottom` 分支没有真实数据走过）。

另：新增页面后旧的 parent dev watcher 只写出了 `wall.json`、没编译 `wall.js/wxml`（页面注册变更后 watcher 会卡住）。杀掉 `kindergarten\parent\node_modules` 那个 node 进程重启 `npm run dev:mp-weixin` 即恢复。

---

## 17. 新窗口开工提示词（B4-b，2026-10-23 准备）

```text
读 docs/handoffs/2026-10-21-parent-task-loop.md 的 §15.3、§16（含 16.6/16.7/16.8/16.9 四轮审查与收尾），
按 §15.3 的 B4-b 范围开工：老师端 9.1 余下（supplement / public 审核封装）、9.2（创建/编辑表单新字段）、
9.3（补充回应与公开审核 UI）、9.7。

开工前先读：openspec/changes/2026-10-20-redesign-parent-task-loop/tasks.md 的 9.x、
design.md 的 D7/D12、backend/api/api-teacher/09-class-and-tasks.md。

起点：HEAD 含 B4-a 的四个提交（8b5a00f / 09062eb / 70f2d67 / 班级墙），家长端 B4-a 已闭合；
老师端仍是 A 切片的老表单，创建任务填不了新字段，详情页没有补充回应与公开审核入口。

约束：
- 不要动 A/B1/B2/B3 与 B4-a 已落地的语义（任务锁序、公开状态机、通知按家长聚合、读标记口径）。
- 前端组件只改 shared-ui/（两端 components/ 会被 sync-ui.js 覆盖）；样式只引用 token。
- 老师端没有 tabBar，跳转一律 navigateTo。
- make run 起后端（8080，MySQL 3316 / Redis 6379）；改完先 npm run dev:mp-weixin。
- 交互验证必须真实点击；**视图层读取用 `_verify/wxc.py --host <port>`**（`cdp.py` 会读到僵尸 pageframe，
  本轮踩过两次），host 从 `/json/list` 里 `127.0.0.1:<port>/__pageframe__/...` 取。
- 原生 `uni.showModal` 在 IDE 里点不到（不进视图层 DOM），需要二次确认的交互用页面内两步点击。
- 别提交 docs/competitive/*、skills-lock.json、.impeccable/。
- 9.x 勾选仍要等真机验证；没跑通前不要勾。
```

现成夹具（园所 4）：任务 4322（已提交 + 主回应 + 补充回应，用于老师端「补充回应」入口）、4323（家长端公开申请/撤销）、4324（class 1，分享码）、4325/4326/4327（家长端待完成/连点）；消息 1171（带 `action_path`）；`task_read_marks` 131。

---

## 18. B4-b 落地内容（2026-10-23）

范围：9.1 余下（supplement / public 审核封装）、9.2（创建/编辑表单新字段）、9.3（补充回应与公开审核 UI）、9.7。
起点 `e517184`，未提交任何 9.x 勾选（真机验证未做）。

### 18.1 改了什么（逐个文件）

| 文件 | 变更 |
| --- | --- |
| `teacher/src/api/teacher-tasks.js` | 新增 `supplementTeacherResponse(taskId, checkinId, data)`（POST supplement）与 `reviewTeacherTaskPublic(taskId, checkinId, {approve, scope_confirmed})`（PUT public）；`share-card` 沿用 B4-a |
| `teacher/src/pages/tasks/index.vue` | 创建表单：星星罐范围改成「按任务/按班级」两态，班级态不可选并说明去管理端配置；星星目标始终可填（选班级星星仪式时必填）；公开开关旁说明审核责任；列表频率/仪式中文化、补「星星目标 N / 个体任务」、学生名单失败显式提示 |
| `teacher/src/pages/tasks/detail.vue` | 任务结构区：无提交时可选频率/日期/范围（班级/个体）/仪式/星星目标，有提交或提交数加载失败时改为只读摘要；关公开、改个体各有页面内两步确认；单条回应补「加星」（任务默认班级星星时）；已回应行渲染贴纸图形/语音播放/班级星星；周期汇总按默认仪式发 `class_star` 或 ⭐ 贴纸并如实标注；已回应行可追加补充（贴纸/语音，班级星星不可选）；公开审核块（pending/approved/revoked 分开表达，批准前必须勾「内容中只有提交者孩子」，撤回两步确认，撤回永远可用）；录音失败收闩并提示 |

### 18.2 语义决定（后续不要改错）

1. **汇总回应的载体跟随任务默认仪式**。A 切片无论任务仪式都发 `{ritual:sticker, sticker_key:star}`——对 `class_star` 任务来说这颗⭐根本不进星星罐。现在 `recognition_ritual=class_star` 时批量发 `{ritual:class_star}`，其余发 ⭐ 贴纸；按钮文案分别为「一键送星 / 一键送贴纸（本周期）」。
2. **结构锁定以 `GET /teacher/tasks/:id/stats` 的 `submission_count` 为准**。加载失败按锁定处理（只读 + 可重试），否则老师会在「不知道有没有提交」时改结构，保存才吃 422。
3. **改个体与关公开都是破坏性操作**，各用页面内两步确认；pending 期间禁止保存，避免「点了取消却已经提交」。
4. **`revoked` 只说「未公开」**，不判断是老师拒绝还是家长取消（两者写同一个 `public_revoked_at`，与 B4-a 16.6-#3 同口径）。
5. **「内容中只有提交者孩子」的勾选每轮作废**：审核成功后清掉；`load()` 只保留当前仍为 pending 的行的勾选，家长重新申请必须重新勾（D7）。
6. **公开 `approve` 与 `public_visible` 分开表达**：approved 但 `public_visible=false` 时额外说明「任务已结束/停用、未开启公开、已改个体或孩子已不在本班」（与后端 `publicCheckinVisible` 全部条件对齐）。
7. **老师端没有读取补充回应的字段**（`GET /teacher/tasks/:id/checkins` 只返回主回应）：UI 只在本次会话标记「✓ 已追加补充回应」，不假装能列出历史补充；要补历史需改契约 + service（见 18.4）。
8. **班级星星罐配置仍不可用**：`GET/PUT /teacher/class/star-jar` 属 C 切片、路由不存在，老师端无法判断是否已配置，因此「按班级」选项置灰并说明，不猜。
9. **两步确认不跨刷新沿用**：`load()` 会清空 `revokeConfirmId`，并且只保留当前仍为 pending 行的「只有提交者孩子」勾选。否则上一轮没确认的撤回会停在「确认撤下？」上，刷新后一次点击就撤掉。
10. **不能批准时不给批准入口**：`task.allow_public=false` 或个体任务时，pending 行只显示说明 + 撤回，不显示勾选/批准（否则老师点了才吃 422）。任务详情未加载时也不给批准。
11. **录音期间禁止其它回应**：录音在飞时点贴纸/加星/汇总/补充会先关掉提交，录音结束后的语音回应必然 409；现在这些入口先提示「请先停止当前录音」。
12. **保存结构后同时重取提交列表**：关公开/改个体会在服务端批量撤销公开状态，不重取 checkins 的话当前页还会显示「已公开 · 本班家长可见」，与实际相反。
13. **切 tab 时重取 stats**：家长可能在老师停留期间提交；不重取会让结构编辑器继续开着，保存时才 422。
14. **「加星」入口看星星罐是否存在，不看任务默认仪式**：`star_scope=class`（班级罐）或 `star_goal>0`（任务罐）才显示。D5 明确默认仪式可逐条调整，若只看 `recognition_ritual=class_star`，默认贴纸但已设任务罐的任务永远加不了星；目标为 0 的任务仍不显示（发了也不计数）。

### 18.3 验证证据（真实点击 + 库核对）

后端 `go run cmd/api/main.go`（8080），IDE 单实例，视图层读取 `_verify/wxc.py --host <port>`；逻辑层 `wx.navigateBack` 在热重载后会让 `getCurrentPages()` 与 pageframe 漂移，必要时用 `_verify/pf.py --route` 直连 pageframe，或 close+open 重开项目（见 18.5）。

| # | 动作 | 结果 |
| --- | --- | --- |
| 1 | 工作台点「亲子任务」 | 进 `pages/tasks/index`；列表显示中文频率/仪式、星星目标、待回应数 |
| 2 | 填标题/说明/学习关联 → 点「创建任务」 | 任务 4332 落库，字段与输入一致 |
| 3 | 打开 4332（0 提交） | 结构区为可编辑 picker；改星星目标 0→4、开公开 → 保存 | 库 `star_goal=4, allow_public=1` |
| 4 | 关公开 → 「确认关闭」前点保存 | 库 `allow_public` 仍为 1（pending 期间被拦）；取消后开关恢复 |
| 5 | 再关公开 → 确认关闭 → 保存 | 库 `allow_public=0` |
| 6 | 家长 API 给 4332 提交一条 | 重进详情：显示「已有 1 条提交，结构已锁定」+ 只读摘要，picker 消失 |
| 7 | 打开 4333（`target_student_id=21`，创建时请求 allow_public=true） | 显示「指派给个别学生的任务不可公开」、范围「周嘉嘉」；库 `allow_public=0`（后端强制） |
| 8 | 4335（班级、allow_public=1）把范围改成周嘉嘉（走 exparser change 绑定，原生 picker 无法 CDP 点击） | 出现「改为…个体任务会关闭公开并撤下已公开内容」确认；确认后保存 | 库 `target_student_id=21, allow_public=0` |
| 9 | 4335 改频率 一次性→每日、开始日期→10-24 → 保存 | 库 `frequency=daily, start_date=2026-10-24` |
| 10 | 4328 待审核：不勾确认点「批准公开」 | 库 `public_scope_confirmed` 仍 0、`approved_at` 仍 NULL（被前端拦） |
| 11 | 勾确认 → 批准 | 库 `scope_confirmed=1, approved_at=13:26:38, approved_by=11`；UI「已公开 · 本班家长可见」+ 撤回 |
| 12 | 点「撤回公开」一次 | 出现「确认撤下？/取消」，库未变 |
| 13 | 点「确认撤下？」 | 库 `is_public_requested=0, scope=0, approved_at=NULL, approved_by=0, revoked_at` 有值；UI「未公开」 |
| 14 | 家长重新申请公开 1957 后重进 | 勾选框回到 ☐（不沿用上一轮确认） |
| 15 | 4329 待回应行点 ⭐ | 生成主回应 701，`checkin.response_id=701`；待回应清空 |
| 16 | 已回应 tab 点「补充回应」→ 🌸 | 生成 702（`single`）+ `task_response_supplements` 指向 1958；UI「✓ 已追加补充回应」；补充面板只有贴纸/语音，无班级星星 |
| 17 | 4330（daily/class_star）点「加星」 | 生成 703 `ritual=class_star, response_type=single` |
| 18 | 同一任务点「一键送星（本周期）」 | 生成 704 `response_type=period, period_start=period_end=当天`，覆盖 1960 |
| 19 | 4334（daily/sticker）详情 | 按钮文案「一键送贴纸（本周期）」；点击生成 707 `sticker/star/period` |
| 20 | 4332 已回应 tab（语音主回应） | 渲染「🔊 点击播放语音」（语音文件由 `/common/upload` 上传，id 992000120） |
| 21 | 4324 点「生成小程序码」 | 落盘 `USER_DATA_PATH/task-share-4324.png`（82497 字节）；按钮 15s 封顶后恢复可点 |
| 22 | 四个页面 CDP 采集 console | 0 error（仅 IDE 自身 3 条 warning） |
| 23 | 对抗性审查后：把 4328 改成 `allow_public=0` + 1957 待审核 → 重进详情 | 不显示勾选/批准，显示「任务已关闭公开或已改成个体任务，不能批准；可撤回这条申请」 |
| 24 | 在该状态下点「拒绝」→「确认撤下？」 | 库 `is_public_requested=0, revoked_at` 有值；UI「未公开」（撤回不受公开开关限制） |
| 25 | 审查后重跑 build + console | build 通过；0 error；tokens 对 tasks 文件 0 违规 |
| 26 | 4328（默认贴纸、`star_goal=5`）待回应行 | 显示「加星」（默认仪式不阻挡任务罐加星，D5） |
| 27 | 4336（默认贴纸、`star_goal=0`）待回应行 | 不显示「加星」（没有可计数的星星罐） |

两端 build：`teacher npm run build:mp-weixin` 通过；`node scripts/check-tokens.mjs teacher` 对两个改动文件 **0 违规**（顺手把 index.vue 原有的两处 `#fff` 改为 token，基线 217→215）。

### 18.4 未验证 / 缺口（不要当成已验证）

- **9.1/9.2/9.3/9.7 都没勾**：按 §15.5 / §17，勾选要等真机（12.6 扫码 + 真机录音）。
- **原生 picker 无法用 CDP 真实点击**：频率/日期/范围/仪式的「用户点选」是用 exparser 的 `bindchange` 处理器模拟（`$vm.$scope.e3_c0` 等），handler 路径是真、手势不是。真机上需人工点一遍四类 picker。
- **录音交互未跑**：IDE 内麦克风/系统授权不可控；主回应与补充回应的语音只验证了上传 + API 落库 + 已回应行渲染/播放按钮，`getRecorderManager` 的录制→上传→回应链路未真实点。
- ~~补充回应的历史不可见~~ → **已补齐**：`checkins` 增加 `supplements`，见 §19。
- ~~星星进度不可见~~ → **已补齐任务罐进度**：列表/详情增加 `star_count`，见 §19。班级罐总进度仍缺（C 切片）。
- **班级星星罐配置不可用**：`GET/PUT /teacher/class/star-jar` 仍属 C 切片；「按班级」选项与「班级罐目标设置」未交付。
- **真机扫码、真机多孩、真机公开墙**未做（沿用 B4-a 缺口）。
- **IDE 路由漂移是环境问题**：热重载/`wx.navigateBack` 后 `getCurrentPages()` 可能仍指向已关闭的 detail，同时 pageframe 已是别的页；`wxc.py` 会报 `no live stack`/`no pageframe`。处理：`_verify/pf.py --route` 直连，或 close+open 重开项目；**不是应用代码问题**，但排查时容易误判成「页面没跳」。

### 18.5 对抗性审查与处置（2026-10-23，对 B4-b diff）

按「怎样让它坏」逐条过，另起一个全新上下文的只读审查者复核（其结论未能在本窗口返回，以下处置均经我逐条对源码/库复核）。

| # | 缺陷 | 性质 | 处置 |
| --- | --- | --- | --- |
| 1 | **默认贴纸但 `star_goal>0` 的任务没有「加星」入口**：`starTracking` 只看 `recognition_ritual=class_star`。D5 明确「默认仪式可逐条调整」，任务罐有目标就该能加星；否则列表显示「星星目标 5」却永远加不了 | 违反 D5 / 语义误导 | 已修：`star_scope=class` 或 `star_goal>0` 才显示加星；`star_goal=0` 仍不显示。复验：4328（贴纸+目标5）有「加星」、4336（目标0）没有 |
| 2 | **录音期间点贴纸/加星/汇总**：会先关掉提交，录音结束后的语音回应必然 409「已被其他回应关闭」 | 交互竞争 | 已修：这些入口先提示「请先停止当前录音」 |
| 3 | **保存结构后不重取 checkins**：关公开/改个体会在服务端批量撤销公开，当前页仍显示「已公开 · 本班家长可见」 | 状态说反 | 已修：保存后 `load()`，三个请求一起刷新 |
| 4 | **切 tab 不重取 stats**：家长在老师停留期间提交，结构编辑器仍开着，保存才 422 | 陈旧数据 | 已修：`switchStatus` 同时 `loadStats()` |
| 5 | **不能批准时仍给批准入口**：`allow_public=false` 或个体任务的 pending 行显示勾选/批准，点了才吃 422 | 无效操作 | 已修：`canApprovePublic` 控制；关闭公开时只留说明+撤回。复验：4328 临时置 `allow_public=0` → 无批准，拒绝仍成功写 `revoked_at` |
| 6 | **stats 并发无序号**：切 tab/保存的旧响应可能把「有提交」洗回 0，结构编辑器重新打开 | 竞态 | 已修：`statsSeq`，旧响应丢弃 |
| 7 | **撤回的两步确认跨刷新沿用**：点过一次「撤回公开」没确认，刷新后同一行直接停在「确认撤下？」 | 一次点击误撤 | 已修：`load()` 清空 `revokeConfirmId` |
| 8 | **改个体与关公开两个确认可同时挂着**：保存被两次拦，UI 两块确认 | 交互混乱 | 已修：互相清空 |
| 9 | **`approved` 但不可见的文案漏了「任务已结束/停用」** | 说明不完整 | 已修：与 `publicCheckinVisible` 全部条件对齐 |
| 10 | **个体任务没有星星目标时列表不显示「个体任务」** | 信息缺失 | 已修：`extraMeta` 独立拼接 |
| 11 | **上传回调在离开页面后仍 toast/发回应** | 跨页副作用 | 已修：`pageGone` 守卫 success/fail |
| 12 | **录音 onError 未收闩**：无麦克风权限时按钮永远停在「停止」 | 卡死 | 已修：onError 复位 + toast |
| 13 | **任务详情加载失败时 pending 审核块说成「任务已关闭公开」** | 误报原因 | 已修：分「未加载 / 不能批准」两种提示 |

**Deferred minors（未改）**：
- 新增动作的 catch 会再 toast 一次，而 http 层已经 toast 过后端文案（既有模式，两个 toast 同步覆盖，实际看到一条）。
- `statsError` 时结构摘要仍写「有提交后只能改…」，但头部已说明「未能确认提交数，已按锁定处理」。
- `saveEdit` 没有防连点闩；两次 PUT 幂等（第二次 `updates` 为空），未加。
- 未知 `frequency/ritual` 会回退显示第一项，但保存时原值相等不会改库（后端只允许三个值，正常数据不出现）。

---

现成夹具（园所 4、老师 11 / 班级 1）见 `_verify/b4b_tasks.json`：4328–4336，含 1957–1963 提交、701–707 回应、补充关系 250。4328 已恢复 `allow_public=1`、1957 为待审核（可测批准/撤回）；4329/4330/4334 已完成回应；4331/4333/4335 无提交（可测可编辑态）。清理时从回应/提交往任务删。

---

## 19. B4-b 只读字段补齐（2026-10-23）

补 §18.4 的两个缺口：老师端看不到「补充过几条」、看不到「星星进度」。只加只读字段，不动状态机、不改回应/提交/公开语义。

### 19.1 改了什么

| 文件 | 变更 |
| --- | --- |
| `server/internal/service/teacher_portal_service.go` | `TaskCheckins` 批量取补充回应并给每条提交附 `supplements`（始终空数组而非 null）；`ListTasks`/`GetTask` 增加 `star_count`；新增 `taskStarCounts` / `taskSupplementsForCheckins` |
| `server/internal/service/teacher_task_read_fields_test.go`（新） | 3 条回归：补充按提交分组且不串孩子；period 计 N 星/补充不计星/软删不计星；非 class_star 主回应 0 星 |
| `backend/api/api-teacher/09-class-and-tasks.md` | 列表/详情补 `star_count`，checkins 补 `supplements` 形状与口径 |
| `teacher/src/pages/tasks/index.vue` | 列表显示「星星 N/goal」+ 进度条（封顶 100%，超额显示「已超额」；班级罐只显示本任务贡献，不显示分母） |
| `teacher/src/pages/tasks/detail.vue` | 结构摘要显示「已集 N 星」；已回应行渲染 `supplements`（贴纸/语音/文字），移除会话内「已追加」标记——刷新后直接来自服务端 |

### 19.2 口径（后续不要改错）

1. `star_count` = **本任务**主回应 `ritual=class_star` 且未删除的提交数。period 回应覆盖 N 条提交计 N 星；补充回应不改 `checkin.response_id`，不计入；提交/回应任一软删都不计。
2. `StarScope=class` 时 `star_count` 仍只表示**本任务贡献**，不是班级罐总进度。班级罐要 join `class_star_jars` 的学期目标（C 切片），不要用这个字段冒充。
3. `supplements` 永远返回数组，按 `created_at ASC, id ASC`；只含未删除的关系/提交/回应。**读取失败让整个 checkins 请求失败**（不返回空数组）：`[]` 只能表示「确实没补过」，否则会诱发重复补充。
4. 星数读取失败时，列表与详情都返回 `star_count=null`（不是 0）；前端隐藏进度。星数是装饰性聚合，不能因为它让任务列表/详情整体不可用，也不能给假 0。

### 19.3 验证

- API：4330 `goal=5, star_count=2`；4328/4324 `goal=5, count=0`；4332 `goal=4, count=0`；4336 `goal=0, count=0`；4329 checkin 1958 `supplements=[flower]`。
- UI 列表：4330「星星 2/5」+ 40% 进度条；4332「星星 0/4」+ 0%；目标 0 的任务无星数字/进度条；个体任务标记仍在。
- UI 详情：4330 摘要「星星：按任务 · 目标 5 · 已集 2 星」。
- 星数即时刷新（审查后补）：4419（目标 5、待回应 1）点「加星」→ 同页摘要立即从「目标 5」变为「目标 5 · 已集 1 星」，不退出页面。
- UI 补充：4329 已回应 tab 显示「补充回应 1 条 🌸」；点「补充回应」→ 🌸 后刷新为「补充回应 2 条 🌸🌸」（库：702、733），不再依赖会话标记。
- 测试：`go test ./internal/service -run 'Task|ParentTask' -timeout 600s` → ok（74.3s）；新增 3 条随该命令通过。
- 构建：`teacher npm run build:mp-weixin` 通过；`check-tokens teacher` 对 tasks 文件 0 违规。
- 运行时：tab 切换期间 appservice + view pageframe 采集，无 exception；仅框架 SharedArrayBuffer warning。（IDE 控制台里偶尔出现的 `/teacher/home/overview` 401 + `navigateTo:fail timeout` 来自登录切换/IDE 恢复期的缓冲日志；用当前存储 token 直接复测 overview 返回 200。）

### 19.4 仍未做

- 班级罐总进度（需要 `class_star_jars` 学期目标 + C 切片端点）；当前班级任务只显示本任务贡献。
- 真机验收不变（picker/录音/扫码）；9.x 仍未勾。

### 19.5 对抗性审查与处置（2026-10-23，对 §19 的 diff）

| # | 缺陷 | 性质 | 处置 |
| --- | --- | --- | --- |
| 1 | **详情星数失败返回 0**：注释说「0 会伪装成没有星星」，代码却给了 0；老师会看到「已集 0 星」而实际是查询失败 | 状态说反 | 已修：`GetTask` 失败时 `star_count=null`，前端 `!= null` 才显示「已集」；契约注明 |
| 2 | **列表在 `star_count` 缺失时画 0%**：旧后端/字段缺失会被读成「一颗星都没有」 | 状态说反 | 已修：`star_count == null` 时 `starProgress` 返回 null、`starSummary` 返回空 |
| 3 | **goal=0 的任务详情仍显示「已集 N 星」**：没有星星罐却报星数 | 信息矛盾 | 已修：只有 `class` 罐或 `goal>0` 才显示「已集」 |
| 4 | 列表星数失败直接 500，详情失败降级 null：两者口径不同 | 有意不对称 | 保留并写进契约：列表不能返回「假 0 星」；详情不能因为只读统计阻塞编辑/回应 |
| 5 | 补充回应读取失败只落日志、返回空数组，老师无法区分「没补过」与「没读到」 | 可观测性 | **未改**（minor）：让整页提交列表失败代价更大；若以后要加，给 `supplements_loaded=false` 而不是 500 |
| 6 | DB 故障路径（星数/补充查询失败）没有单测 | 测试缺口 | **未改**（minor）：需要可控失败注入；正常/软删/跨孩子路径已覆盖 |
| 7 | **加星/汇总后详情星数不刷新**：`refreshWithCap` 只重取 checkins，`task.star_count` 停在打开页面时的值；老师刚加的星要退出重进才看得到 | 数据陈旧 | 已修：`refreshPeriod` 一并回填 `star_count`（null 不覆盖）。复验：4419 点「加星」后同页「星星：按任务 · 目标 5 · 已集 1 星」立即出现，无需刷新 |

复查（静态 + 运行）：
- 索引核对：`task_check_ins(task_id, response_id)`、`task_response_supplements(check_in_id, response_id)` 均存在；两条新查询都走索引。
- 口径核对：period 覆盖 N 条提交计 N 星；补充不计；提交或回应软删不计；跨孩子补充不串（测试断言）。
- `go build ./...` 通过；新增 3 条测试与任务套件 `go test -run 'Task|ParentTask'` 通过；teacher `build:mp-weixin` 通过；tokens 0 违规。

### 19.6 干净上下文复审（2026-10-23，针对 §19）

另起一个全新上下文的只读审查者（未参与实现），结论写入 `_verify/p1_review.md`（gitignore）。它报 **0 Critical / 1 Important / 5 Minor**；逐条复核后处置如下。

| # | 审查结论 | 复核 | 处置 |
| --- | --- | --- | --- |
| I-1 | **补充回应读失败被降级成 `[]`**：老师看到「没补过」的假象，会重复追加；客户端无法区分「没补」与「没读到」；家长端同一故障是整请求失败 | 成立 | 已修：`TaskCheckins` 直接返回错误（整请求 500，可重试），契约改为「失败不返回空数组」 |
| M-1 | 列表星数失败 500、详情 null，同一故障表现差异大；星数是装饰性聚合 | 成立 | 已修：列表也降级为 `star_count=null`（前端已能隐藏），不再拖垮任务列表 |
| M-2 | 详情刷新时若服务端给 null，旧 star_count 被保留，可能把「刚加的星」显示成没加上 | 成立 | 已修：`refreshPeriod` 改为 `!== undefined` 就落值（null 也落）；另加 `loadSeq` 守卫，旧响应不能覆盖新星数 |
| M-3 | class scope 的 star_count 是全生命周期，不是当前学期；C 切片直接累加会算错 | 成立 | 契约补「不做学期过滤、不得直接累加」；C 切片必须按学期重算 |
| M-4 | 测试无法区分「按 response 关联」与「按 task 关联」；也没钉住空数组而非 null | 成立 | 已补：同一任务 class_star + sticker 两条主回应只计 1 星；无补充时序列化为 `[]` 的断言 |
| M-5 | `TaskCheckins` 逐行查主回应的 N+1（非本次引入） | 成立 | 未改（既有实现）；本次只把补充回应改成批量查 |
| — | 其他核对点（period 计 N、补充不计、软删过滤、跨孩子、null/0/超额/class scope UI） | 未发现问题 | 保持 |

复审后：`go test ./internal/service -run 'Task|ParentTask' -timeout 600s` → ok（77.2s）；`go build ./...`、teacher `build:mp-weixin` 通过；tokens 0 违规。

### 19.7 第二次干净上下文复审（2026-10-23，B4-b 快照）

用户要求的另一个全新上下文审查者（`b4b-adversarial-reviewer`）最终返回了结论；它审的是 **14:01/14:02 的 B4-b 快照**，不是 §19。其中 3 条在返回前已修（保存后列表不刷新、缺 `canApprovePublic`、两个破坏性确认不互斥），其余逐条复核后处置：

| # | 审查结论 | 复核 | 处置 |
| --- | --- | --- | --- |
| 1 | 默认仪式非 `class_star` 时，有任务罐目标也无法加星 | 成立（快照早于修复） | 已在 §19.5 修：`starTracking` 按罐是否存在判断。复验 4328 有「加星」、4336 无 |
| 2 | 结构锁发现太晚：只读摘要用未保存的 edit 显示现状；保存仍发结构值 → 整单 422，标题改动也丢 | 成立 | 已修：`loadStats` 锁定/失败时 `syncStructureFromTask`；保存前 `refreshStatsBeforeSave` 再确认一次，若期间家长提交且有结构改动则回填并提示、不发请求；`saveEdit` 锁定态只发任务当前结构值 |
| 2b | 结构锁错误落进笼统「参数无效」 | 成立 | 已修：`ErrTaskStructuralLocked` 导出，`UpdateTask` 单独返回，handler 文案「任务已有提交，只能修改标题、说明、学习关联与公开开关」 |
| 3 | `recognition_ritual=''` 的历史/管理端任务：任何保存都 422（客户端归一化成 sticker） | 成立 | 已修客户端：`fillEdit` 不再归一化空枚举，picker 显示「未设置」，保存原样发 `""`（后端当「不修改」）。管理端 `TaskService.CreateTask` 不持久化这些字段属 C 切片，另记 |
| 4 | 语音上传/录音窗口无门闩：上传中点贴纸/汇总会 409；切 tab 后录音 15s 自动发出；批量在飞时可开始录音 | 成立 | 已修：上传开始即 `addResponding(row)`，成功由回应 finally 释放、失败路径也释放；`startRecord` 检查 `batchBusy`/`isResponding`；切 tab/下拉先取消录音（`discardRecording` 丢弃上传） |
| 5 | class-star 任务上 ⭐ 贴纸与「加星」同屏，⭐ 不计星但无提示 | 成立 | 已修：`starTracking` 时隐藏 ⭐ 贴纸，并在列表上方提示「「加星」才计入星星罐；贴纸回应不计星」 |
| 6 | 「只有提交者孩子」勾选可跨「家长撤回→重新申请」沿用 | 成立 | 已修：每轮 `load()` 都清空勾选（服务端没有本轮申请时间，无法区分；安全优先） |
| 7 | statsError 时只读说明仍写「有提交后…」，把加载失败说成已有提交 | 成立 | 已修：三态文案（正在确认/未能确认已锁定/有提交后） |
| 8 | 批量回应忽略 `notified_count/notify_failed_count` | 成立 | 已修：单条/加星/批量/语音在 toast 后追加「N 个家长未收到通知」 |
| 9 | 未知 frequency/ritual 被夹成首项，保存可能静默改值/422 | 成立 | 已修：同 #3，未设置值原样保留 |

**复验（真实点击 + 库核对）**
- 锁发现：4583 先改频率为「每周」（未保存）→ 家长 API 提交 → 切 tab 两次 → 只读摘要回到「一次性」、picker 消失（`locked=true`）；改标题保存成功，库 `frequency=once` 未被未保存值覆盖。
- 空仪式：4584（DB 直插 `recognition_ritual=''` + 一条提交）→ 摘要「仪式：未设置」；改标题保存成功，库仍 `recognition_ritual=''`。
- 结构锁文案：`PUT /teacher/tasks/4330` 改 frequency → 422「任务已有提交，只能修改标题、说明、学习关联与公开开关」。
- 星星提示：4328 有「加星」、无 ⭐ 贴纸，页面出现「「加星」才计入星星罐；贴纸回应不计星」。
- 勾选作废：4328 勾选 ☑ → 切「已回应」→ 切回「待回应」→ 回到 ☐。
- 构建/测试：`go test -run 'Task|ParentTask'` → ok（78.7s）；`go build ./...`、teacher `build:mp-weixin` 通过；tokens 0 违规；清空 CDP Log 缓冲后点 tab，view 层 0 error。
- **未验证**：语音三条门闩路径（无麦克风/授权，静态审查 + 编译）；`notify_failed_count>0` 的真实提示（需制造通知失败）。

现成夹具新增：4583（锁发现，checkin 2107，已被标题修改但结构仍 once）、4584（空仪式，checkin 2106）。清理时按 task 级联删。

---

## 20. C 阶段开工准备（2026-10-23）

起点：`feat/parent-growth-scope` @ `982a817`（已 push）。B4-b 已闭合（9.x 仅差真机），§19 的两个只读字段已交付。**C 阶段不要从零假设现状**——下面每一项都在本窗口逐文件核过。

### 20.1 现状核实（截至 982a817）

| 项 | 现状 | 缺口 |
| --- | --- | --- |
| `TaskService.CreateTask`（`POST /admin/tasks`） | 只写 title/description/class/frequency/dates/status；**不写** `learning_link` / `target_student_id` / `allow_public` / `recognition_ritual` / `star_scope` / `star_goal`；不通知家长 | 7.0；字段持久化 + `validateTargetStudent` + `validateClassStarJar` + 发布通知（见 20.3-2） |
| `TaskService.UpdateTask` | 全量覆盖 title/description/class/frequency/dates/status；**没有** `assertTaskEditable`、没有 `allow_public` 撤销、没有 `validateTargetStudent` | 7.0 + D12；有提交后只能改安全字段；改个体/关公开要 revoke |
| `ParentTask.ActivityID` | 字段/索引已在 domain 与表里；**没有任何端点写它**，现有任务全为空 | 7.1 活动端点与 `activity_id` 生成 |
| `ClassStarJar` | domain + 表 + `validateClassStarJar` 已存在；**无 GET/PUT 端点**；`validateClassStarJar` 只按 `goal>0`，**没有学期过滤** | 3.8（teacher）、7.5（admin）、学期口径 |
| 管理端路由 | `/admin/tasks` CRUD/stats + `PUT /admin/tasks/checkins/:id/public`；**无** `task-activities`、**无** `star-jar` | 7.1/7.5 |
| 管理端统计 | `completion_rate` / `avg_completion_rate` / `by_class` 旧口径 | 7.2 新指标（design D12） |
| 管理端 UI | `admin/pages/tasks/index.vue` 只有列表 + 删除；`api/tasks.ts` 只有旧字段；`types/api.d.ts` TaskItem 旧；`admin/pages/classes` 存在但无星星罐 | 11.1–11.4 |
| 权限 | Casbin 对 `admin`/`teacher` 是 `/api/v1/admin/*`、`/api/v1/teacher/*` 通配；新端点不需要改 `rbac_policy.csv` | 无需改策略，但 UI 菜单权限码仍是 `task:view` |
| 契约 | `backend/api/api-admin/10-tasks.md` 明确标注 `[C]` 未实现字段/端点 | 8.3/8.4 + 新端点契约 |
| 老师端 | 「按班级」选项置灰；班级罐目标不可设置；列表/详情只有本任务 `star_count` | 3.8 后端就绪后开 C-4 |

### 20.2 范围与顺序

C 阶段 = `tasks.md` 的 **3.7 / 3.8 / 7.0–7.5 / 11.1–11.4 / 12.5**，外加两处老师端收尾（把 9.2 的「班级罐」补上，把 3.8 接到 UI）。

推荐拆窗（不要一个窗口从后端做到 UI）：

| 窗口 | 范围 | 目的 |
| --- | --- | --- |
| **C-1 后端基础** | 7.0（admin create/update 字段与规则）、3.8 + 7.5（班级罐 GET/PUT）、学期 term helper、`validateClassStarJar` 学期化、8.3 契约 | 让「班级罐」与「admin 建任务」语义正确 |
| **C-2 后端活动/统计** | 7.1（`task-activities`）、7.2（stats 新指标）、7.3（管理端查看公开内容并撤回）、8.3 契约 | 让 11.x 有可接的接口 |
| **C-3 管理端 UI** | 11.1–11.4：任务表单/多班活动/统计/公开撤回；班级页星星罐设置 | 管理端可用 |
| **C-4 老师端收尾** | 按班级选项启用、班级罐目标设置、class 罐进度；9.2 闭环 | 3.8/7.5 有 UI 入口 |
| **C-5 验证/收尾** | 12.5；补 12.3/12.4；9.x/7.x 勾选；12.7/12.8 | 回归与归档 |

### 20.3 开工前必须先定的决策

1. **学期 term 的格式与范围**（阻塞 3.8/7.5/班级罐进度）。
   建议 `term = Class.AcademicYear + "-" + KindergartenAppSetting.CurrentSemester`（如 `2025-2026-spring`）；范围取 `AcademicYear.SemesterSpringStart/End` 或 `SemesterFallStart/End`。为空时按 design 的自然年/自然学期兜底（建议 spring=2/1–7/31、fall=9/1–次年1/31；都没有则自然年）。**`validateClassStarJar` 必须改成只认当前 term**，否则上学期目标会让本学期班级罐任务通过创建校验。
2. **管理端建任务是否发通知**。D9 写「任务发布给目标家庭写站内消息」，但没有限定老师端；`notifyTaskPublished` 目前是 `TeacherPortalService` 方法，依赖 `s.msg`。建议抽成共享 notifier（db + message repo），admin/activity 创建后也发，响应可只返回计数或写日志；若决定不发，必须在契约里写明例外。默认选「发」。
3. **管理端如何查看公开内容（7.3）**。现在只有 `PUT /admin/tasks/checkins/:id/public`，没有列表能拿到 checkin id。至少补一个只读端点（建议 `GET /admin/tasks/:id/checkins?status=public`，返回公开字段与 `public_status/public_visible`），否则 7.3 只有撤回没有「查看」。
4. **活动 `activity_id` 生成与分组边界**。建议服务端生成不可猜字符串（如 `act_<unixnano>_<rand>`）；`POST /admin/tasks` 强制忽略/拒绝客户端传的 `activity_id`；空 `activity_id` 的 stats 按任务 id 分组，不能塞进一个「空活动」。
5. **班级罐进度的计数口径**：当前 term 内、`StarScope=class`、主回应 `class_star` 的提交数；封顶 100%，超额显示「已超额」。任务罐 `star_count`（§19）是**本任务**，不能直接累加当班级罐。

### 20.4 关键实现约束（沿用，别重写一套）

- `server/internal/service/task_rules.go` 是共享规则：`validateTargetStudent` / `validateClassStarJar` / `assertTaskEditable` / `revokePublicForTask`。**admin 的 Create/Update 必须复用**，不要复制条件（D12）。
- `TaskService.UpdateTask` 有提交后只允许 `title/description/learning_link/allow_public`；`target_student_id` 改个体、`allow_public` 关都必须 revoke 公开状态。
- `TaskService.CreateTask` 的 `allow_public` 对个体任务强制 false；`star_scope=class` 必须有当前 term 班级罐目标；`recognition_ritual=class_star && star_scope=task` 要求 `star_goal>0`。
- 活动创建：`class_ids` 必填、去重、≤50、全部属于当前园所；事务内逐班建任务，任一失败整组回滚；活动任务禁止单条 `DELETE /admin/tasks/:id`，单条 `PUT` 只允许安全字段或直接拒绝（D12）。
- Stats 公式（D12）：`response_rate` = 有主回应的提交/总提交；`participation_rate` 分母是目标学生数（个体=1），不是家长数；`avg_response_hours` 只算 `single` 主回应的中位数；`period` 与补充回应不计；`feedback_coverage/child_happy_rate/ack_rate` 的分母是 `notified_count + notify_failed_count` 的去重家长对；`pending_over_48h` 用 `TaskCheckIn.CreatedAt < now-48h`。
- 下线旧 `completion_rate/total_possible/checkin_count/not_checkin_count/total_points_awarded`，任务列表/详情/老师统计/管理端统计统一新口径。
- 新依赖（message repo / 班级罐 repo / 新 service）要过 Wire：改 provider + `make wire`；wire 必须用 go1.25 构建（见 CLAUDE.md）。
- 通知失败不回滚任务/回应；计数按家长去重，不按提交条数。

### 20.5 验证与已知坑

- 后端：`go test ./internal/service -run 'Task|ParentTask|ClassStar' -timeout 600s`；新增测试覆盖字段持久化、结构锁、活动事务/class_ids、班级罐 term、stats 公式、公开查看/撤回。
- 管理端：`npm run build`；用 `admin-run` skill（Playwright）或 `_verify` 的无头 CDP（见 memory `admin-ui-headless-cdp`）做真实点击；不要用截图代替 DOM 断言。
- 小程序：老师端「按班级」选项、班级罐目标设置、进度展示要真实点击；仍不能替代真机。
- 跑识别队列相关的 DB 测试前停 8080（memory `running-backend-steals-queue-jobs`）；service 全量测试用 `-timeout 600s`。
- 不要提交 `docs/competitive/*`、`skills-lock.json`、`.impeccable/`；样式只引用 token；两端组件只改 `shared-ui/`。
- 现有夹具：老师/班级 1 的任务 4328–4336（见 `_verify/b4b_tasks.json`）；班级 4 的 4322/4323/4324。C 阶段新增班级罐后，先给班级 1 配一个目标再用 UI 验证。

### 20.6 新窗口开工提示词（直接复制）

```text
读 docs/handoffs/2026-10-21-parent-task-loop.md 的 §20（C 阶段开工准备）与 §15.3/§19，
按 §20.2 的 C-1 范围开工：7.0（管理端建/改任务新字段与共享规则）、3.8 + 7.5（班级星星罐 GET/PUT）、
学期 term helper、validateClassStarJar 学期化、8.3 契约。

开工前先读：openspec/changes/2026-10-20-redesign-parent-task-loop/tasks.md 的 3.7/3.8/7.0–7.5、
design.md 的 D6/D12、backend/api/api-admin/10-tasks.md、server/internal/service/task_rules.go。

起点：feat/parent-growth-scope @ 982a817（已 push）；B4-b 已完成，C 阶段后端/管理端仍是旧实现（见 §20.1）。

约束：
- 复用 task_rules.go，不许在 TaskService 里复制结构锁/个体/班级罐规则。
- 先定 §20.3 的五个决策（term 格式、admin 是否通知、公开内容查看端点、activity_id 生成、班级罐计数口径），
  在 handoff 记录 ruling 再写码。
- 管理端契约 backend/api/api-admin/10-tasks.md 是 SoT；新端点/字段先改契约。
- 新依赖改 provider 后 make wire（go1.25）；跑 service 测试用 -timeout 600s；停 8080 再跑队列相关测试。
- 不要动 A/B1/B2/B3/B4 已落地语义；不要提交 docs/competitive/*、skills-lock.json、.impeccable/。
- 完成后更新 handoff，并只在测试/真实点击通过后勾 7.x/3.x/11.x。
```

---

### 20.7 C-1 开工决策（2026-10-23，写码前定）

按 §20.3 的五条逐条定，先落 ruling 再写代码。

| # | 决策 | 结论 | 依据 / 理由 |
| --- | --- | --- | --- |
| 1 | 学期 `term` 格式与范围 | `term = <Class.AcademicYear>-<semester>`（如 `2031-2032-spring`）。`AcademicYear` 为空或不是 `YYYY-YYYY` → `<自然年>-<semester>`（如 `2026-spring`）。**范围**：优先取 `academic_years` 中 `(kg, AcademicYear)` 行对应学期的起止；该行缺失/起止为空/不可解析时按自然学期兜底（`spring` = 次年 2/1–7/31，`fall` = 当年 9/1–次年 1/31）；`AcademicYear` 为空时用自然年 1/1–12/31。`semester` 取 `KindergartenAppSetting.CurrentSemester`，非 `fall` 一律按 `spring`（与 `settings_service.go` 同口径）。**`validateClassStarJar` 只认当前 term**。 | design D6 第 99–101 行 + §20.3-1。上学期目标不得让本学期 `StarScope=class` 任务通过创建校验。helper 只读，不在读取路径建配置行。 |
| 2 | 管理端建任务是否发通知 | **发**。把 `notifyTaskPublished` 从 `TeacherPortalService` 方法抽成公共函数 `notifyTaskPublishedFor(ctx, db, msg, task)`，teacher 创建与 admin `POST /admin/tasks` 共用；admin 响应追加 `notified_count` / `notify_failed_count`。通知在落库之后执行，失败只记日志 + 计数，不回滚任务。 | design D9 只写「任务发布给目标家庭写站内消息」，没有限定端；§20.3-2 默认选「发」。两个入口各写一套会漂移。 |
| 3 | 管理端如何查看公开内容（7.3） | **C-2 补只读端点** `GET /admin/tasks/:id/checkins?status=public`，返回 `public_*` 与 `public_status` / `public_visible`（复用 `publicCheckinView` 与 `publicCheckinVisible`），只读、不改状态机；C-1 不动。 | 现有只有 `PUT /admin/tasks/checkins/:id/public`，拿不到 checkin id，7.3 只剩「撤回」没有「查看」。属 C-2 范围（§20.2）。 |
| 4 | 活动 `activity_id` 生成与分组 | 服务端生成 `act_<unixnano>_<rand>`（不可猜、不连续）；`POST /admin/tasks` 收到非空 `activity_id` **直接 422 拒绝**（不是静默丢弃）；空 `activity_id` 是独立任务，`by_activity` 按任务 id 分组，不塞进「空活动」。属 C-2。 | design D12 第 173、179 行。静默丢弃客户端字段正是 7.0 要消灭的失败模式；拒绝能让调用方立刻发现用错端点。 |
| 5 | 班级罐进度计数口径 | **当前 term** 内、`StarScope=class` 的任务中，主回应 `ritual=class_star` 且提交/回应均未软删的提交数 / 班级罐 `goal`；封顶 100%，`count > goal` 时返回「已超额」标记而不是 >100% 的进度。任务罐 `star_count`（§19）只表示**本任务**贡献，**不得**直接累加当班级罐。属 C-4 的展示；C-1 只交付目标配置与校验。 | design D6 第 96–98 行；§19.2-2 已明确 `star_count` 不能冒充班级罐总进度。 |

---

### 20.8 C-1 落地内容（2026-10-23）

起点 `982a817`。范围：7.0（管理端建/改任务新字段与共享规则）、3.8 + 7.5（班级罐 GET/PUT）、学期 term helper、`validateClassStarJar` 学期化、8.3 契约。**没有 UI**，全部后端 + 契约。

#### 改了什么（逐个文件）

| 文件 | 变更 |
| --- | --- |
| `server/internal/service/task_rules.go` | 新增 `classStarJarTerm` / `classStarJarTermOf`（term + 学期范围解析）、`findClassStarJar` / `upsertClassStarJar`、`validTaskRitual`；`validateClassStarJar` 改为**只认当前 term**；新增哨兵 `ErrClassNotFound` / `ErrStarJarGoalInvalid` / `ErrTaskRitualInvalid` / `ErrTaskStarScopeInvalid` / `ErrTaskStarGoalInvalid` / `ErrTaskActivityIDForbidden`，并导出 `ErrTaskTargetInvalid` / `ErrTaskStarJarMissing` 供 handler 用 `errors.Is` 判定 |
| `server/internal/service/task_notify.go` | `notifyTaskPublished` 抽成公共 `notifyTaskPublishedFor(ctx, db, msg, task)`，老师端与管理端共用同一收件人口径/文案/深链/判重 |
| `server/internal/service/task_service.go` | `TaskService` 增加 `msg` 依赖；`CreateTask` 持久化 `learning_link` / `target_student_id` / `allow_public` / `recognition_ritual` / `star_scope` / `star_goal`，复用共享校验，发布后发通知并返回计数；`UpdateTask` 改为「只改变化字段 + 结构锁 + 关闭公开/改个体撤销」，新增字段用**指针**表示缺省；`Status` 改 `*int8`（原先 `required` 让 `status=0` 不可达）；列表/详情补返回新字段 |
| `server/internal/service/task_star_jar.go`（新） | `GetClassStarJar` / `SetClassStarJar`（老师端与管理端共用），返回 `term` / `term_start` / `term_end` / `goal` / `configured` / `updated_at`，写 `operation_logs` |
| `server/internal/repository/task_repo.go` | `TaskListRow` 与列表 SELECT 补 `learning_link` / `target_student_id` / `allow_public` / `recognition_ritual` / `star_scope` / `star_goal` / `activity_id` |
| `server/internal/handler/http/task.go` | 新增 `writeTaskError`（哨兵 → 稳定 4xx/503 文案）；新增 `GetClassStarJar` / `SetClassStarJar`（admin）；`CreateTask` / `UpdateTask` 改用统一映射 |
| `server/internal/handler/http/teacher_portal.go` | 新增 `TeacherGetClassStarJar` / `TeacherSetClassStarJar`，班级取自登录态 |
| `server/internal/app/router.go` | `GET/PUT /teacher/class/star-jar`、`GET/PUT /admin/classes/:id/star-jar` |
| `server/internal/middleware/admin_perm.go` | 新路由登记：GET → `task:view`，PUT → `task:manage` |
| `server/internal/app/wire_gen.go` | `make wire` 重新生成（TaskService 新增 MessageRepo） |
| `backend/api/api-admin/10-tasks.md` | 8.3：建/改任务新字段、默认值、422 码、指针语义、结构锁、撤销规则、列表/详情字段、班级罐端点；活动/统计仍标 `[C]` |
| `backend/api/api-teacher/09-class-and-tasks.md` | 3.8：`GET/PUT /teacher/class/star-jar` 形状与校验；班级罐说明改为「已实现」 |
| `server/internal/service/task_admin_c1_test.go`（新） | 21 条真实 MySQL 用例 |
| `openspec/.../tasks.md` | 勾 7.0、7.5（3.8 / 8.3 未勾，见下） |

#### 语义决定（后续不要改错）

1. **term 只认当前学期**：`<班级 academic_year>-<spring|fall>`；班级无学年时 `<自然年>-<学期>`。范围优先 `academic_years` 配置，缺失按自然学期兜底（spring 次年 2/1–7/31，fall 当年 9/1–次年 1/31）。上学期目标**不会**放行本学期 `StarScope=class` 任务（`TestValidateClassStarJarOnlyAcceptsCurrentTerm` 守护）。
2. **班级罐 upsert 必须复活软删行**：唯一索引不含 `deleted_at`，`ON DUPLICATE KEY UPDATE` 不带 `deleted_at=NULL` 会「写成功但读不到」。
3. **目标 0 是非法写入**：`0` 表示「未配置」，写成 0 会让配置看起来成功、而建任务继续 422。
4. **`learning_link` 在管理端建任务改为必填**（D15）。建/改任务在落库前校验；更新时缺省 = 保留原值，原值也为空才 422。理由：管理端此前没有任何创建界面，此刻改必填不会破坏存量调用方；放着不填会让家长端看到没有学习关联的任务。
5. **更新请求的新字段一律指针语义**：`target_student_id` / `allow_public` / `star_goal` 缺省 = 不改。老客户端不传 `allow_public` 不会把公开悄悄关掉并批量撤回（`TestAdminUpdateTaskOmittedPointersKeepValues` 守护）。
6. **`status` 不是结构字段**：有提交后仍允许结束任务。`assertTaskEditable` 的允许清单没动（老师端语义不变），管理端把 status 从结构检查里单独摘出来。
7. **改个体 → 强制关闭公开 + 撤销**；但**有提交后改个体本身被结构锁拒绝**（D12）。所以撤销分支在管理端只覆盖并发窗口，属防御；测试断言的是落库值与「被拒的更新不留半个改动」，没有伪造不可达链路。
8. **管理端创建任务也发通知**：与老师端同一函数。文案沿用「老师发布了亲子任务…」——判重键是完整标题，改文案会让历史任务的幂等失效，因此不在本窗口改。
9. **`activity_id` 非空直接 422**（decision §20.7-4）：静默丢弃正是 7.0 要消灭的失败模式。

#### 验证证据（2026-10-23）

- `go build ./...`、`go vet ./internal/service ./internal/handler/http ./internal/app ./internal/repository` 通过。
- `go test ./internal/service -run 'TestClassStarJar|TestValidateClassStarJar|TestAdminCreateTask|TestAdminUpdateTask' -timeout 600s` → ok（47.8s，21 条）。
- `go test ./internal/service -run 'Task|ParentTask|ClassStar' -timeout 600s` → ok（125.4s，含全部既有任务用例）。
- `go test ./internal/app ./internal/middleware` → ok（路由表冲突检查 + 管理端权限登记表覆盖检查，新路由会自动纳入）。
- 变异验证 6 项全部**真实断言失败**（非编译错误）：
  `validateClassStarJar` 去掉 term 过滤 / `CreateTask` 丢弃 `learning_link` / 不强制个体任务关闭公开 / `UpdateTask` 不撤销公开（首次变体因 `effectiveAllowPublic` 未使用变成编译失败，已改成 `if false && (...)` 重跑）/ upsert 不复活软删行 / 结构锁把 status 一起算进去。
  未做变异的：`classStarJarTermOf` 的自然学期兜底分支（由 3 条 term 用例分别钉住，但没逐条变异）。

#### 未做 / 缺口（不要当成已验证）

- **3.8 未勾**：老师端 UI（按班级隐藏任务目标、班级罐目标设置入口）属 C-4；`/admin/task-activities` 第三个创建入口属 C-2，所以「三个入口共用校验」当前只落地两个。
- **8.3 未勾**：活动端点、统计新口径仍标 `[C]`（C-2）。
- **HTTP 冒烟未跑**：本轮没有管理端 token，未起 8080 走真实网关。handler 层只做了编译 + 路由/权限登记检查。
- **`TaskService.CreateTask` 未写 `intro` / `requirements` / `rewards`**：不在 7.0 清单内，家长端任务详情这几块对管理端创建的任务仍是空。
- **老师端 `CreateTask` 未校验仪式取值**：`validTaskRitual` 只在管理端用。改老师端会动 A 切片已测路径，本窗口不动，登记在此。
- **`TaskService.GetTaskStats` / 列表 `completion_rate`** 仍是旧口径（7.4 属 C-2）。

### 20.9 C-1 对抗性审查与处置（2026-10-23）

另起一个全新上下文的只读审查者看 C-1 的未提交 diff（不参与实现），报 6 CONFIRMED / 5 POSSIBLE / 6 NOTE。**我逐条对源码复核后才动手**，其中 1 条复核后不成立。

| # | 审查结论 | 复核 | 处置 |
| --- | --- | --- | --- |
| 1 | **term 永远不前进**：`current_semester` 没有任何写路径（只在建行时写 `"spring"`），`UpdateClassReq` 也没有 `academic_year` → term 恒等于建班学年 + spring，fall 分支是死代码，「按学期」实际不生效 | **成立**（grep 确认：`CurrentSemester` 只在 `point_service.go:323` / `settings_service.go:177` 初始化时写；`UpdateSettingsReq` 与 `UpdateClassReq` 都不含这两个字段） | **不在 C-1 修**：补写路径要动班级 + 设置两个模块（handler/service/契约/管理端 UI），属跨 3 层的新切片。记为 **P1 阻塞项**，见下 |
| 2 | 无学年班级返回自然年区间，与契约「未配置时按自然学期兜底」矛盾 | 成立（实现按 design「为空时用自然年」，是**契约写错**） | ✅ 已修契约：明确三种口径（有配置 / 有学年无配置 / 无学年→自然年） |
| 3 | 契约宣称的 `TASK_*` 机器错误码在响应体里不存在（只有 `code=42200` + 中文 message） | 成立（`pkg/response` 只有数字业务码） | ✅ 已修契约：删掉错误码清单，改为「以 message 为准」 |
| 4 | 收件人查询整体失败返回 `(0,0)`，与契约「通知失败体现在 `notify_failed_count`」不符 | 成立，但这就是 B3 §14.3-#5 的既有决定（不伪造失败数） | ✅ 已修契约措辞，与 B3 口径对齐；实现不动 |
| 5 | 我的「被拒更新不留半个改动」断言的是 `AllowPublic`（请求根本没改它），改成「先写 target 再撞锁」也照样过 | **成立**（真实测试缺陷） | ✅ 已修：改断言 `TargetStudentID == 0` |
| 6 | `CreateTask` 接受纯空白标题并落库空串，`UpdateTask` 却用 `BAD_TITLE` 拒绝 | 成立 | ✅ 已修：创建路径同样拒 `BAD_TITLE`，并补用例 |
| 7 | 倒序/同年学年（`2032-2031`）让 fall 兜底产出 `term_start > term_end` | 成立 | ✅ 已修：`parseAcademicYearSpan` 要求 `end >= start`，否则整段退化成自然年；补 `TestClassStarJarTermRejectsReversedAcademicYear` |
| 8 | `SetClassStarJar` 用两次 `time.Now()`，跨 term 边界会「写旧 term、读新 term」，返回 200 却 `goal=0` | 成立（窗口极窄但真实） | ✅ 已修：写入与回读共用一个 `now` |
| 9 | 任务本来就是 `star_scope=class` 时，改标题也会重新校验班级罐；罐被删后连标题都改不了 | 成立 | ✅ 已修：只在 `star_scope` 或 `class_id` 真的变化时校验；补 `TestAdminUpdateTaskClassScopeTitleEditNotBlockedByMissingJar` |
| 10 | 库里 `goal=0` 的行让 GET 返回 `configured=false` 但 `updated_at` 非空，契约说 null | 成立 | ✅ 已修：`updated_at` 只在 `goal>0` 时给值 |
| 11 | 列表新读的三个可空列遇 NULL 会让整页 500 | **不成立**：实测 `learning_link`/`recognition_ritual`/`activity_id` 虽可空、库里也确实有 NULL（任务 4584 的 `activity_id`），但 GORM 对 string 字段容忍 NULL，`ListTasks` 与 `GetTaskDetail` 都正常返回 `""`（已实跑验证） | ❌ 不改，不引入 COALESCE |
| 12 | 老师端 `UpdateTask` 仍是事务外结构锁 + 事务内不锁任务行（TOCTOU） | 成立，D12 要求两条路径一致 | **未改**：属老师端既有实现，动它要重跑 A/B 已测路径；记为后续 |
| 13 | 管理端前端类型未同步（`createTask` 无 `learning_link`、`TaskItem` 无新字段） | 成立 | **未改**：属 C-3（11.1） |
| 14 | `description` 非指针，PUT 省略会清空说明 | 成立（真实静默丢数据） | ✅ 已修：改 `*string`，缺省保留原值；补断言 |
| 15 | 测试缺口：无 fall 分支、无倒序学年、无「省略 learning_link 保留原值」、无通知失败；夹具注释 ID 写错 | 成立 | ✅ 已补 5 条用例（fall / 倒序 / 空白标题 / 省略 learning_link / class 范围标题编辑）；注释已更正。**`writeTaskError` 仍无 handler 级测试**，保留 |
| 16 | 班级后来补设学年后，旧 term 的罐行读不到也不清理 | 成立 | **未改**（无迁移工具）；记录：旧行按新 term 重新配置即可，不影响正确性 |
| 17 | upsert 复活软删行时保留旧 `created_by/created_at` | 成立 | **不改**：创建者是原始配置人，审计上是对的 |

**P1 阻塞项（从 #1 拆出，必须在 C-4 之前定）**：学期切换没有写路径——`classes.academic_year` 建班后不可改、`current_semester` 永远 `spring`。要让「按班级罐按学期重置」真实生效，需要新增：管理端班级编辑接受 `academic_year`、设置页接受 `semester`（或提供「学期切换」动作）。这是跨 `class` + `settings` 两模块的独立切片，**不属于 C-1**。

#### 修复后的验证（2026-10-23）

- `go build ./...` 通过。
- `go test ./internal/service -run 'TestClassStarJar|TestValidateClassStarJar|TestAdminCreateTask|TestAdminUpdateTask' -timeout 600s` → ok（57.5s，26 条）。
- `go test ./internal/service -run 'Task|ParentTask|ClassStar' -timeout 600s` → ok（133.8s）。
- `go test ./internal/app ./internal/middleware` → ok。
- 新增变异 4 项，全部**真实断言失败**（非编译错误）：更新时无条件校验班级罐 / 学年允许倒序 / 创建不拒空白标题 / 省略 description 时清空。
- 结论：本轮审查修复未引入回归；#1 作为 P1 阻塞项待 C-4 前处理，#12/#13/#16 为已知缺口。

#### 20.9.1 补修：`star_goal` 负值（2026-10-23，用户指定先修）

我自己的复查把它列为 POSSIBLE，独立审查者没报；用户要求先修。

- **共享下界**：`task_rules.go` 新增 `validTaskStarGoal(goal int) bool`（`goal >= 0`）与哨兵 `ErrTaskStarGoalNegative`。
- **四条写路径全部拦截**：管理端 `CreateTask` / `UpdateTask`、老师端 `CreateTask` / `UpdateTask`。老师端 `UpdateTask` 只在请求带了新值时校验（`req.StarGoal < 0`），历史负值不会把「改标题」一起挡住——与管理端「只校验本次请求值」同口径。
- **handler 文案**：`writeTaskError` 增加 `星星目标不能为负数`（422）。
- **契约**：`api-admin/10-tasks.md` 的字段表与 PUT 说明、`api-teacher/09-class-and-tasks.md` 的编辑说明各补一行「必须 ≥0」。
- **测试**：新增 3 条（admin 创建 / admin 更新 / 老师端创建），更新断言「被拒的目标不得落库」（`StarGoal` 仍为 0）。
- **变异**：3 个入口各删掉校验，3 条用例全部**真实断言失败**（非编译错误）。
- **回归**：`go test ./internal/service -run 'Task|ParentTask|ClassStar' -timeout 600s` → ok（140.7s）；`./internal/app ./internal/middleware` → ok。

口径说明：`0` 仍然合法（表示「按任务罐不启用」）；只有 `class_star + task` 范围额外要求 `> 0`，那条规则没动。

#### 20.9.2 第二轮对抗性审查（2026-10-23，针对「修复本身」）

第二个全新上下文的只读审查者，切入点是「上一轮的修复有没有引入新问题」。报 4 CONFIRMED / 4 POSSIBLE / 5 NOTE，全部复核成立（除重复项），处置如下。

| # | 结论 | 复核 | 处置 |
| --- | --- | --- | --- |
| 1 | 同学年 `2031-2031` 在 fall 兜底仍倒置（`2031-09-01 > 2031-01-31`） | 成立：上一轮只拒了 `end < start`，相等放行 | ✅ 改为严格递增 `end <= start` 即拒；兜底区间改用 `startYear` 推（spring = 次年 2/1–7/31，fall = 当年 9/1–次年 1/31） |
| 2 | **老师端 `UpdateTask` 的 `star_goal` 缺省被当成 0**：静默清零目标；有提交时 `updates` 含 `star_goal` → 结构锁 422；合同示例（只发 title/description/learning_link/allow_public）实际失败 | 成立（原实现 `req.StarGoal != cur.StarGoal`） | ✅ `TeacherTaskInput` 的 `StarGoal`/`AllowPublic`/`Description` 改为指针：缺省 = 保留原值；创建按默认值展开。合同示例现在真的能用 |
| 3 | 老师端不校验 `star_scope` / `recognition_ritual` 枚举、不 trim，能落 `"CLASS"` / `"class "`；落库后管理端编辑直接 422 | 成立 | ✅ 老师端创建/更新都 trim + `validTaskRitual` / 枚举校验，与管理端同一套 helper |
| 4 | 同一「罐被删 + 只改标题」状态，老师端 422、管理端放行 | 成立（D12 要求两端一致） | ✅ 老师端改成只在 `star_scope` 真的变化时才校验 |
| 5 | 老师端省略 `description`/`allow_public` 会清空说明、关公开并批量撤回 | 成立（同 #2 根因） | ✅ 指针化一并解决；补 `TestTeacherUpdateTaskOmittedAllowPublicKeepsValue` |
| 6 | 管理端 `class_star + task + goal=0` 的历史任务，改标题也 422 | 成立 | ✅ 目标规则改为「只在本次动到目标/仪式/范围时校验」（`goalTouched`），与负值那条同口径 |
| 7 | `academic_years` 查询用未归一化学年，`"2031 - 2032"` 永远命不中，term 也带空格 | 成立 | ✅ 查询与 term 一律用解析后的 `normAY` |
| 8 | 非相邻学年 `2031-2035` 的 fall 兜底跨到 2035，违反契约「次年 1/31」 | 成立 | ✅ 同 #1，兜底只用 `startYear` |
| 9 | C-1 测试在无 `configs/app.ini` 时会整体 skip（绿但零覆盖） | 成立，但这是全仓 service 测试的既有模式 | **不改**，记录 |
| 10 | 测试缺口：同年 fall 无覆盖、老师端 update 的 goal 无覆盖、负值用例初始值就是 0（弱断言） | 成立 | ✅ 补 9 条用例；负值用例先建 `goal=3` 再断言仍为 3 |
| 11 | `SetClassStarJar` 外班 + `goal=0` 先 422 而不是契约的 404 | 成立 | ✅ 先解析班级再判目标 |
| 12 | term 永不前进（P1，已记 §20.9） | 已知 | 不变 |
| 13 | 管理端前端 `createTask` 不带 `learning_link` | 已知（C-3） | 不变 |

另：审查者独立复核了上一轮被否掉的「NULL 列打挂列表」，从 GORM `schema/field.go` 的 `**string` 扫描池给出独立依据，**结论一致**（不成立）。这条不必再争。

#### 本轮验证（2026-10-23）

- `go build ./...` 通过。
- `go test ./internal/service -run 'Task|ParentTask|ClassStar|Supplement|Public|React|Ritual' -timeout 600s` → ok（260.3s，含 B1/B2/B3 全部既有用例）。
- `./internal/app ./internal/middleware ./internal/config ./internal/auth` → 全部 ok。
- 新增变异 7 项全部**真实断言失败**：fall 用 endYear / 允许同学年 / 学年不规范化 / 先判目标再解析班级 / 老师端 goal 缺省归零 / 老师端无条件校验罐 / 管理端无条件校验 goal>0。其中 M20 首版因 `goalTouched` 未使用变成编译失败，已换 `(goalTouched || true)` 重跑确认为断言失败。
- C-1 用例从 21 条增至 38 条（`grep -c "^func Test"` 核对）。

### 20.10 C-2 开工决策（2026-10-23，写码前定）

§20.3 第 1–5 条已在 §20.7 定完（C-1）。这里补 C-2 必须先定的两条，编号接续。取证（2026-10-23 实查）：MySQL **8.4.0**（有窗口函数）；`parent_tasks` / `task_responses` **没有任何通知计数列**，`notified_count` 只活在响应体里；`messages` 表在**判重命中时不写新行**（`exists → sent++` 后直接 continue），失败也不写行；当前库仅 12 条 `single` 主回应。

| # | 决策 | 结论 | 依据 / 理由 |
| --- | --- | --- | --- |
| 6 | 反馈率（`feedback_coverage` / `child_happy_rate` / `ack_rate`）的**分母**从哪来 | 在 `parent_tasks` 与 `task_responses` 各加 **`notify_target_count INT NULL`**；**先写义务再投递**（算出 targets 后先落数，再逐条发）；`NULL` 表示「未能确定义务」，统计时**排除而不是当 0**；分母聚合**必须从 `task_check_ins.response_id` 出发**。**不建** `task_notify_targets` 明细表。 | 见下方分析。一句话：分母的语义是「义务集」，不是「成功集」。 |
| 7 | `avg_response_hours` 的中位数怎么算 | **应用层排序取中位**；SQL 只负责筛出 `single` 主回应的时长；偶数取两个中间值的平均（四舍五入 1 位小数）、负时长剔除并计数、空集返回 `null`；新契约字段名用 **`median_response_hours`**（design 第 182 行写的 `avg_response_hours` 就是指它）。 | 7.2 要 `by_class` / `by_activity` **每个桶各算一次中位数**；SQL 方案要为每个维度写一遍窗口函数，Go 方案一次取数按桶分组，口径只有一处且中位数可单测。 |

#### 决策 6 的推理链（为什么不是另外三个）

指标问的是「我们告知过的家长里，有多少人回来给了反馈」，衡量的是**回路**，不是**投递可靠性**。因此分母必须是发布/回应那一刻确定的**义务集**：

- **不能用 `messages` 行数**：① 失败无行 → 分母漏掉失败方；② 判重命中不写新行 → 即使全部成功，行数也小于义务集。两者都会让分母偏小、比率虚高。
- **不能实时 join `student_parents` 算「应通知」**：解绑、转班、后来新增绑定都会改变「上个月那次通知该发给谁」，历史报表每次查询结果不同——那就不是指标。
- **不建明细表**：精确到人才能「查谁没收到 / 补发 / 追责」，但 D12 没有这个需求；且通知路径是**故意非事务**的（失败不回滚任务），多写 N 行会放大故障面。等真出现补发需求再升级为明细表，计数列是它的子集。

**关键约束：先写义务再投递。** 若「发完再记数量」，一旦进程崩在发送中途，计数永远写不上，任务会被静默排除在指标外且无人察觉。义务必须在承诺的那一刻落库。

**`NULL` 与 `0` 必须分开**：收件人查询整体失败时写 `NULL`（未知），统计把它从比率里排除并单独返回缺失计数；写 `0` 会让「未知」和「确实没有家长」不可区分，正是 §19.2-4 已定的「读失败不给假 0」同一条原则。

**聚合起点必须是 `task_check_ins.response_id`**：补充回应也是一条 `TaskResponse(single)`，只是不写 `checkin.response_id`；若从 `task_responses` 出发聚合，补充回应会被算进主回应指标，违反 B1 已定的「补充不计入」。

#### 决策 7 的边界（不写死一定会吵）

- **偶数个**：两个中间值取平均，四舍五入到 1 位小数。
- **负时长**（时钟漂移、手工改库）：剔除并计数，不参与计算，且要能被观察到剔除了几条。
- **空集**：返回 `null`（不是 0），前端显示「—」。
- **命名**：design 把这个指标写成 `avg_response_hours = 中位数`，名字与语义相反。7.2 的契约尚未编写，此时改成 `median_response_hours` 代价为零；等 11.2 接上 UI 再改就要改两端。契约里要注明与 design 旧名的对应关系。
- **数据量**：stats 按园所 + 时间段过滤，不是全表扫描；万级行以内应用层排序无压力。若将来出现跨园年度聚合（十万级），再考虑下沉 SQL —— 那时也只改 `median()` 的取数方式，不改语义。

#### 对 7.2 的直接影响（实现清单）

1. 迁移：`parent_tasks.notify_target_count`、`task_responses.notify_target_count`（均可空，`docs/migrations/` 新建 SQL；`AutoMigrate` 只增不改）。
2. 写入点：`notifyTaskPublishedFor`（发布）与 `notifyTaskResponse`（主回应），在发送循环**之前**更新计数；查询失败写 `NULL` 并记日志。
3. 统计：`response_rate` / `participation_rate` / `pending_over_48h` 不依赖本决策；反馈类三个指标与 `median_response_hours` 依赖。
4. 契约：`GET /admin/tasks/stats` 增加 `median_response_hours`、`notify_baseline_missing`（分母未知的任务/回应数），写明 `NULL` 语义。
5. 测试：分母取「义务集」而非 messages 行数（用「判重命中」场景钉住：重复发布后分母不变）；补充回应不进分母；`NULL` 被排除而不是当 0；偶数/奇数/负值/空集四种中位数。

#### 仍未定（C-2 之外，不阻塞）

- **7.3 的查看端点**已定（§20.7-3）。
- **学期切换写路径**（P1）：`academic_year` 建班后不可改、`current_semester` 恒为 spring；需单独 change，建议排在 C-4 之前。

### 20.11 C-2 新窗口开工提示词（2026-10-23 准备，直接复制）

```text
读 docs/handoffs/2026-10-21-parent-task-loop.md 的 §20.10（C-2 两个决策）、§20.8（C-1 已落地）、
§20.9（C-1 两轮审查与修复），按 §20.2 的 C-2 范围开工：
7.1 活动端点、7.2 stats 新口径（含 7.4 下线旧字段）、7.3 管理端查看公开内容、8.3 契约剩余。

开工前先读：
- openspec/changes/2026-10-20-redesign-parent-task-loop/tasks.md 的 7.1–7.4、8.3
- design.md 的 D12（活动与统计）、D9（通知口径）、D7（公开）
- backend/api/api-admin/10-tasks.md（SoT，C-1 已更新）
- server/internal/service/task_rules.go（共享规则）、task_notify.go（通知/判重/收件人）、task_service.go
- server/internal/repository/task_repo.go（管理端列表/统计现有口径）

起点：feat/parent-growth-scope @ 8e93cb2（已 push）。
C-1 已交付：管理端建/改任务新字段与共享规则、班级罐 GET/PUT（teacher + admin）、学期 term、
validateClassStarJar 学期化、8.3 部分契约；对应测试 38 条在 task_admin_c1_test.go。

开工前 10 分钟检查：
1. git log -3 --oneline 确认没有别的窗口在改同一批文件
2. netstat -ano | findstr :8080 —— 跑队列相关测试前必须无 LISTEN
3. server/configs/app.ini 存在（service 测试没有它会整体 skip，会变成「绿但零覆盖」）

开工顺序（一次只做一组，组间先做限制性对抗性审查再验证）：
1. 7.1 活动端点 —— 不依赖 §20.10 的两个决策，先做
2. §20.10 决策 6 的迁移与写入点（notify_target_count 两列）
3. 7.2 stats 新口径 + 7.4 下线 completion_rate / total_possible / checkin_count / not_checkin_count
4. 7.3 查看公开内容端点 + 8.3 契约收尾

已定规则（不要重新发明）：
- activity_id：服务端生成 act_<unixnano>_<rand>；POST /admin/tasks 收到非空 activity_id 直接 422（§20.7-4）
- 7.3：GET /admin/tasks/:id/checkins?status=public，复用 publicCheckinView / publicCheckinVisible（§20.7-3）
- 反馈率分母：notify_target_count 可空列；先写义务再投递；NULL=未知、统计排除而不是当 0；
  聚合从 task_check_ins.response_id 出发（补充回应不进分母）（§20.10-6）
- median_response_hours：应用层排序取中位；偶数取两中值平均、负时长剔除并计数、空集 null；
  契约字段用 median_response_hours 并注明与 design avg_response_hours 的对应（§20.10-7）
- 活动创建/编辑/删除全部复用 task_rules.go：validateTargetStudent / validateClassStarJar /
  assertTaskEditable / revokePublicForTask
- class_ids 必填、去重、≤50、全部属于当前园所；事务内逐班建，任一失败整组回滚
- 活动任务禁止单条 DELETE /admin/tasks/:id；单条 PUT 只允许安全字段或直接拒绝

约束：
- backend/api/api-admin/10-tasks.md 是 SoT：先改契约再写码
- 新依赖改 provider 后 make wire（GOTOOLCHAIN=go1.25.0）；service 测试 -timeout 600s
- 不要动 A/B1/B2/B3/B4 与 C-1 已落地语义（任务锁序、公开状态机、通知按家长聚合、
  读标记口径、term 只认当前学期、指针缺省=保留原值）
- 关键断言做变异验证，并人工确认是断言失败而不是编译失败（B3 §14.3-#10 踩过）
- 不要提交 docs/competitive/*、skills-lock.json、.impeccable/

现成夹具（园所 4）：任务 4328–4336（_verify/b4b_tasks.json）、4322/4323/4324；
C-1 测试夹具 992731–992736 / 992974。
C-2 需要新增夹具：一条已公开的提交（7.3）、一个带通知的任务 + 主回应（反馈率分母）、
两个班级（活动批量建/整组回滚）。
注意：AutoMigrate 能给开发库补新列，production 走 docs/migrations/ 的 SQL。

完成后：更新 handoff（新增 §21 C-2 落地），只在测试通过后勾 7.1/7.2/7.3/7.4/8.3；
3.8 与 11.x 仍要等 UI（C-3/C-4）。
```

**新窗口不要碰的三件事**：
1. 学期切换写路径（`academic_year` / `current_semester` 无写入口）——独立 change，排在 C-4 前。
2. 管理端前端 `admin/api/tasks.ts` / `types/api.d.ts`——属 C-3。
3. 老师端 `UpdateTask` 的 TOCTOU（结构锁在事务外）——既有实现，改动要重跑 A/B 已测路径。

---

## 21. C-2 落地内容（2026-10-24）

起点 `8e93cb2`（含 §20.8 的 C-1 与 `3ff64fe` 的决策记录）。范围：7.1 多班活动、7.2 统计新口径、
7.3 查看公开内容、7.4 下线旧字段、8.3 契约收尾。**没有 UI**，全部后端 + 契约。

### 21.1 改了什么（逐个文件）

| 文件 | 变更 |
| --- | --- |
| `server/internal/service/task_activity.go`（新） | `CreateActivity` / `UpdateActivity` / `DeleteActivity`；`newTaskActivityID`（`act_<unixnano>_<24位hex>`，crypto/rand）；`normalizeActivityClassIDs`（非空/无 0/不重复/≤50/全属本园所，按 id 升序）；描述符 `ErrTaskActivity*` 六个 |
| `server/internal/service/task_stats.go`（新） | 统计全部口径：`loadTaskStatsInput`（一次读任务/区间提交/被引用回应/反馈/班级在读学生）、`buildBuckets`（全园/班级/活动或独立任务）、`metrics`（回应率/参与率/待回应/48h/中位数/反馈三率）、`taskStatRate`（封顶 100，分母 0 → nil）、`taskStatMedianHours`、`taskDetailMetrics`（单任务详情） |
| `server/internal/service/task_rules.go` | 新增 `taskStructure` + `validateTaskStructureForCreate`（三个创建入口共用，老师端也接上了）、`taskSafeEditableFields` / `taskSafeEditableField`、`validateTaskTitleLength` + `ErrTaskTitleTooLong` |
| `server/internal/service/task_service.go` | `CreateTask` 改用共享结构校验；`ListTasks` 去掉逐行比率（少两倍 N+1）；`GetTaskDetail` 换新口径 + `notify_target_count`；`UpdateTask` 加活动任务单条结构禁令；`DeleteTask` 加活动任务单删禁令；新增 `ListPublicCheckins`；`GetTaskStats` 整体重写（调 task_stats.go）；删除 `totalPossibleCheckins` / `completionForTask` / `completionForRow`（`formatPercent` 保留，通知模块仍在用）；新增 `parseTaskDateRange` / `taskStatsRange` |
| `server/internal/service/task_notify.go` | `recordTaskNotifyTargetCount` / `recordResponseNotifyTargetCount`；发布与主回应都在**投递前**写义务集，收件人查询失败写 NULL |
| `server/internal/repository/task_repo.go` | `TaskOverlapRow` 补 `title/activity_id/target_student_id`；新增 `CountParticipatingStudents`；删除只为旧口径存在的 `CountCheckinsByTaskIDs` / `CountCheckinsInDateRange` |
| `server/internal/domain/task.go` | `ParentTask.NotifyTargetCount` / `TaskResponse.NotifyTargetCount`（`*int`，可空） |
| `server/internal/handler/http/task.go` | `ListTaskPublicCheckins`；`writeTaskError` 增加活动/公开列表 8 条映射 |
| `server/internal/handler/http/task_activity.go`（新） | 三个活动端点 handler |
| `server/internal/service/teacher_portal_service.go` | 老师端 `DeleteTask` / `UpdateTask` 加活动任务禁令；`CreateTask` 的结构校验改调共享 helper（错误码仍是 `ErrTeacherInvalidParams`） |
| `server/internal/handler/http/teacher_portal.go` | `teacherPortalErr` 两条活动哨兵文案 |
| `server/internal/handler/http/task_error_test.go`（新） | `writeTaskError` 唯一出口的表驱动用例（10 个哨兵 → 状态码/业务码） |
| `backend/api/api-teacher/09-class-and-tasks.md` | 老师端两条活动任务禁令的契约（8.2 未全完成，只补这两行，别当成 8.2 已勾） |
| `server/internal/app/router.go` | `GET /admin/tasks/:id/checkins`、`POST/PUT/DELETE /admin/task-activities` |
| `server/internal/middleware/admin_perm.go` | 四条登记：GET 查看码 `task:view`，活动写操作 `task:manage`。⚠️ 该文件 diff 有大量空白行变化，是 gofmt 因新增最长键重排对齐，`git diff -w` 只有 7 行 |
| `docs/migrations/2026-10-23-task-notify-target-count.sql`（新） | 两列 `notify_target_count INT NULL` |
| `backend/api/api-admin/10-tasks.md` | 8.3 收尾：活动三端点、`GET /admin/tasks/:id/checkins`、stats 全字段与新口径、详情新字段、列表/详情移除旧字段、活动任务单删/单编辑规则 |
| `server/internal/service/task_admin_c2_*.go`（5 个新文件） | 43 条真实 MySQL 用例（活动 16 / 通知 5 / 统计 16 / 公开 6），夹具 992741–992751 |

### 21.2 语义决定（后续不要改错）

1. **活动没有个体目标**：`POST /admin/task-activities` 不接收 `target_student_id`（design D12 第 171 行的请求形状里没有它）。所以「三个创建入口共用 `validateTargetStudent`」体现为共享函数传 `targetStudentID=0`（该函数对 0 是空操作），而不是给活动加一个个体目标字段。
2. **`class_ids` 重复有独立哨兵**：重复会同时被「归属计数差」挡住，但那条文案会让调用方去查「哪个班不属于本园所」。`ErrTaskActivityDuplicateClass` 单独一条，文案直接说「选重了」。
3. **活动批量编辑真拒绝而不是忽略**：请求 DTO 里保留结构字段，`hasStructuralField()` 一旦为真直接 422（即使取值与当前相同）。静默忽略正是 §20.7-4 要消灭的失败模式。
4. **活动任务单条 PUT 连 `status` 也不许改**：D12 只允许标题/说明/学习关联/公开开关。`status` 在单任务路径是「非结构字段」，在活动任务这里是禁改——活动整组同进同退。
5. **先写义务再投递**：`notify_target_count` 在发送循环之前写；收件人查询失败写 NULL 并记日志。投递失败不影响计数（失败也算「我们承诺告知过」）。
6. **统计的提交集按 `date` 落区间**：与 `total_checkins` 同口径。任务集按区间重叠取。
7. **参与率是集合口径**：分子 = 区间内有提交、且仍在目标范围（个体 = 目标孩子；班级 = 当前在读且仍在该班）的去重孩子；分母 = 任务覆盖的目标孩子去重数。转班/离园的历史提交不进分子（`TestStatsParticipationIgnoresTransferredStudents`）。
8. **反馈类分母只来自「被提交引用的回应」**：`task_check_ins.response_id` 出发 → `task_responses.notify_target_count`。补充回应不被引用，天然不进分母；NULL 排除并计入 `notify_baseline_missing`。
9. **中位数只算 `single`**：`period` 覆盖的提交计入回应率/待回应，但不进中位数。负时长剔除并计数。
10. **比率封顶 100%**：分母小于分子时（历史数据/手工改库）截断，不允许 200%。
11. **`by_activity` 里独立任务按任务 id 各自成条**，`activity_id` 为空串、`task_id` 为该任务 id；不合并成「空活动」。
12. **班级归属校验在事务内并用 `FOR SHARE` 锁班级行**：普通一致性读看不到并发软删，校验通过后仍可能给一个正在删除的班级建任务——这种任务在列表/统计的 `INNER JOIN classes` 里永远不出现。共享锁让并发软删等待本事务提交。
13. **标题长度按字符算（≤200）且校验在 service**：只有挂 `binding` 的入口会挡超长标题，活动批量编辑没有 binding，放过去就是 MySQL 1406 → 500。
14. **活动批量编辑的空值口径按类型分**：字符串/数组的空串、空数组 = 未提供；`star_goal`/`target_student_id`/`status` 只要键出现就拒绝（0 是合法业务值，无法区分回传与修改）。

### 21.3 验证证据（2026-10-24）

- `go build ./...`、`go vet ./...` 通过。
- `go test ./internal/service -run 'TestActivity|TestStats|TestTaskStat|TestTaskListAndDetail|TestTaskDetailMetrics|TestPublishNotify|TestResponseNotify|TestRecordNotify|TestSupplementResponseLeaves|TestAdminListPublicCheckins|TestIndividualTaskParticipation'` → ok（43 条）。
- `go test ./internal/handler/http`（`writeTaskError` 映射表）→ ok。
- 全量任务回归 `-run 'Task|ParentTask|ClassStar|Supplement|Public|React|Ritual|Stats|Activity'` → ok（344.4s，含老师端 A/B 全部既有用例）。
- `./internal/app`（路由冲突 + 权限登记覆盖）与 `./internal/middleware` → ok。
- 开发库补列：启动一次 `go run ./cmd/api/main.go` 让 AutoMigrate 加 `notify_target_count`（随后已杀进程，:8080 现无 LISTEN）。
- `golangci-lint run` 对改动包 → **0 issues**。顺带修了 3 条**既有**问题（不是本窗口引入）：
  `inclusiveCalendarDays` 在我删掉旧口径调用后变成死代码（已删）；`task_rules_test.go` 的一处
  ineffectual assignment（改成 `start, _`）；`task_notify.go` 的一处 S1016（改类型转换）。
- 变异验证 20 项，全部**真实断言失败**（人工确认非编译错误）：
  1. 活动创建改在事务外写 → 回滚用例残留 1 条任务；
  2. 去掉重复班级判定 → 错误从 `DUPLICATE_CLASS` 变 `CLASSES_INVALID`；
  3. 去掉活动任务单删禁令 → 未报错（删成功了）；
  4. 去掉活动任务单条结构禁令 → 未报错；
  5. 批量编辑不检查结构字段 → 频率改动未被拒；
  6. 发布不写义务集 → `notify_target_count` 为 NULL；
  7. 义务集写成 0（不取 len(targets)）→ 发布与回应两条都读到 0；
  8. 中位数不过滤 `period` → 6 而不是 4；
  9. NULL 义务当 0 → 覆盖率 100% 而不是 50%；
  10. 参与率不与目标范围取交集 → 100% 而不是 50%；
  11. 去掉比率封顶 → 200%；
  12. 去掉公开列表的 `is_public_requested` 过滤 → 3 条而不是 1 条；
  13. `public_visible` 硬编码 true → 关闭公开的任务仍显示可见；
  14. 去掉老师端活动任务单删禁令 → 删成功、未报错；
  15. 去掉老师端活动任务结构禁令 → 未报错；
  16. 个体任务参与率按提交行计数（去掉 `Distinct`）→ 2 而不是 1；
  17. 去掉活动标题列宽校验 → 直接撞 MySQL `Error 1406: Data too long for column 'title'`；
  18. 去掉老师端标题列宽校验 → 同样撞 1406；
  19. 去掉公开列表的 `Limit/Offset` → 一页返回 3 条（分页用例失败）；
  20. 批量结束不把 `status` 从结构锁里摘出 → 有提交的活动任务无法结束。

### 21.4 对抗性审查与处置（2026-10-24，fresh context 只读复审）

一个全新上下文的审查者只读复审 C-2 的未提交 diff，报 2 CONFIRMED / 1 POSSIBLE / 6 NOTE。
**逐条对源码复核后才动手**，全部成立（无一条需要驳回）。

| # | 结论 | 复核 | 处置 |
| --- | --- | --- | --- |
| 1 | `DELETE /admin/tasks/:id` 对活动任务返回 **500**：该 handler 自带一份错误映射，没有 `ErrTaskActivityDeleteForbidden` 分支，虽然 `writeTaskError` 里补了也没人用 | 成立（契约写 422，实际 500） | ✅ 删掉第二份映射，`DeleteTask` 改走 `writeTaskError`；新增 `handler/http/task_error_test.go` 把「唯一出口」钉成回归（10 个哨兵 → 状态码/业务码） |
| 2 | 活动不变量只守在管理端：老师端 `PUT/DELETE /teacher/tasks/:id` 没有任何 activity 判定，老师可单删活动里的一条任务，或把它改成个体任务（此时 `activity_id` 还留着） | 成立；spec 第 228 行的 MUST NOT **未限定端点**（D12 只写了 admin 路由是它的简写） | ✅ 老师端 `DeleteTask` 加单删禁令、`UpdateTask` 加「只允许安全字段（含 status）」；`teacherPortalErr` 补两条文案；新增 2 条用例 |
| 3 | 活动批量编辑把「显式传空串」当未提供，契约却写「显式传入任何结构字段都拒绝」 | 成立，但**结论是改契约不是改代码**：客户端整表单回传时结构位是空串，因此 422 会把合法调用打成不可用 | ✅ 契约改为「显式传入**非空**结构字段」并写明空串/空数组按未提供处理（`class_ids: []` 不会删班） |
| 4 | `activity_id` 契约写 12 位 hex，实现是 24 位 | 成立（实现用 12 字节 crypto/rand，更好） | ✅ 契约与示例改为 24 位 |
| 5 | 个体任务 `CountParticipatingStudents` 数的是提交行，两次提交会得 2/1 | 成立；当前只被比率封顶掩盖，原始计数已错 | ✅ 加 `Distinct(student_id)`；补用例直接断言该方法返回 1（封顶会掩盖，所以断言落在方法而不是比率上） |
| 6 | `response_id` 指向已删/缺失回应的提交仍算「已回应」 | 成立但当前不可达：级联删除会同时软删提交 | ❌ 不改，记在 §21.5 |
| 7 | 死代码：`inclusiveCalendarDays`、`TaskOverlapRow` 的 `frequency/start_date/end_date` | 成立 | ✅ `inclusiveCalendarDays` 已在 lint 时删；`TaskOverlapRow` 去掉三个统计不再读的字段 |
| 8 | 测试弱点：无 handler 用例（正是 #1 逃逸的原因）、「先写义务再投递」的**顺序**无法被现有用例钉住、无 `configs/app.ini` 时整体 skip | 成立 | ✅ 补 handler 映射用例；顺序不可测（要 fault injection）与 skip 属全仓既有模式，记在 §21.5 |
| 9 | 工作区卫生：`skills-lock.json`、`.impeccable/`、`docs/competitive/*` | 成立 | ❌ 不提交（§20.5 已明确） |

**修复后的验证（2026-10-24）**：`go build ./...` / `go vet ./...` / `golangci-lint`（改动包 0 issues）通过；
C-2 用例 38 条 + handler 映射用例通过；全量任务回归 344.4s 通过（含老师端 A/B 既有用例）；
新增变异 3 项（老师端删/改禁令、个体参与率去重）全部**真实断言失败**。

### 21.5 未做 / 缺口（不要当成已验证）

- **管理端前端仍是旧字段**：`admin/pages/tasks/index.vue` 还在渲染 `completion_rate`（现在为 undefined，列会空），
  `statistics/index.vue` 读 `avg_completion_rate`（已无此字段）。属 C-3 的 11.1/11.2/11.4，本窗口按约束未碰。
- **HTTP 冒烟未跑**：本轮没有管理端 token，未起 8080 走真实网关。handler 层的状态码由
  `writeTaskError` 的表驱动用例（10 个哨兵）保证，但**没有真实请求证据**；老师端两条新禁令
  也只有 service 层用例 + `teacherPortalErr` 映射，未走网关。
- **通知「收件人查询失败」这条分支没有自动化用例**：伪造查询失败需要坏连接/坏 schema，
  与 B3 的既有缺口一致。但它的**写入语义**已被钉住：`TestRecordNotifyTargetCountWritesNullForUnknown`
  验证传 `nil` 会把旧值清成 NULL（`Updates` 的 map-nil 行为），不会被写成 0。
- **`parent_tasks.notify_target_count` 目前只在任务详情暴露**，不参与任何比率（反馈率的分母来自回应）。
  它的用途是「发布触达义务集」，供 C-3 展示。
- **「先写义务再投递」的顺序没有用例钉住**：把 `recordTaskNotifyTargetCount` 移到发送循环之后，
  现有用例仍会通过（要测顺序需能注入「发送中途崩溃」）。写入值本身（正常 = len(targets)、未知 = NULL）
  已被两条用例钉住。
- **C-2 用例在缺 `configs/app.ini` 时会整体 skip**（`openTaskB1DB` 的既有模式，同 §20.8-#9）：
  CI 上若没有该文件，这些用例是「绿但零覆盖」。
- **`response_id` 指向已删回应时仍算已回应**：当前级联删除会同时软删提交，不可达；
  若将来出现「只删回应不删提交」的路径，`response_rate` 会偏高。
- **老师端 `UpdateTask` 的 TOCTOU**、**学期切换写路径（P1）**：仍未动，见 §20.9 / §20.11。
- **`CountParticipatingStudents` 只按当前班级判断**：孩子转班后旧提交不计入参与率，这是有意的；
  但若产品希望「历史参与也要算」，要改的是统计口径而不是这条 SQL。

### 21.6 第二轮对抗性复审（2026-10-24，针对「第一轮修复本身」）

第二个全新上下文的只读审查者。结论：**第一轮 6 项修复全部被确认有效、无绕过、无回归**
（逐条核对了 delete 映射的四种可达错误、老师端 guard 的位置与可达性、`COUNT(DISTINCT)` 的
GORM 生成语义、`TaskOverlapRow` 无残留读取、文档与代码一致、handler 用例非重言）。
同时提出 7 条新问题，全部复核成立，处置如下。

| # | 结论 | 复核 | 处置 |
| --- | --- | --- | --- |
| 1 | `PUT /admin/task-activities/:id` 的 `title` 没有长度校验：201 个字符 → MySQL 1406 → **500**（同一请求走 POST 是 422） | 成立 | ✅ `task_rules.go` 加 `taskTitleMaxLen` / `validateTaskTitleLength`（按**字符**算，不是字节）+ `ErrTaskTitleTooLong`；活动创建/批量编辑都校验；补 200/201 边界用例 |
| 2 | 契约说「所有班级的校验与写入在同一事务内」，但归属校验在事务**外**（TOCTOU）：并发软删班级会建出一条永远不出现在列表/统计里的任务 | 成立 | ✅ 拆成纯输入校验（事务外）+ `lockOwnedClasses`（事务内、`FOR SHARE` 锁班级行）；契约同步 |
| 3 | `10-tasks.md` 的 `participation_rate` 行和 `pending_count` 行被挤成一行，表格渲染乱码 | 成立（是我上一轮编辑时手滑删掉的换行） | ✅ 拆回两行 |
| 4 | 文档说「空串/空数组按未提供」，但 `star_goal`/`target_student_id`/`status` 出现即拒绝（含 0），与给出的理由相反 | 成立 | ✅ 契约改成按类型分述：字符串/数组空值=未提供；数值/指针**出现即拒绝（0 也算）**，并说明客户端整表单回传时应省略这三个键 |
| 5 | 迁移脚本用 `INT`，模型 `*int` 经 GORM 实际建成 `bigint`（已实查 information_schema） | 成立 | ✅ 迁移改 `BIGINT` 并注明原因 |
| 6 | `validateTaskStructureForCreate` 注释声称「三个创建入口共用」，但老师端 `CreateTask` 仍手抄一份 | 成立 | ✅ 老师端 `CreateTask` 改调共享 helper（错误码仍 `ErrTeacherInvalidParams`，对外行为不变）；注释现在为真 |
| 7 | `task_error_test.go` 钉不住「有人在 handler 里再内联一份映射」这个根因 | 成立，但无 DB 的 handler 用例覆盖不到 | ❌ 不改，记在下方 |

**本轮验证**：`go build ./...` / `go vet ./...` / `golangci-lint`（改动包 0 issues）通过；
C-2 用例 39 条（当时）+ handler 映射用例通过；全量任务回归 **348.8s** 通过（含老师端 A/B 全部既有用例，
覆盖了 `CreateTask` 的共享 helper 改造）；新增变异 1 项（去掉标题列宽校验）**真实断言失败**——
失败信息正是审查者预测的 `Error 1406: Data too long for column 'title'`。

**仍保留的缺口**：
- 老师端 `UpdateTask` 的**更新侧**校验仍是手写（创建侧已共享）。更新语义与创建不同
  （只在字段真的变化时校验），强行抽一个 helper 会把它变成两套参数，暂时不动。
- `lockOwnedClasses` 的并发行为没有自动化用例（要稳定复现需要两个连接交错软删，成本高于收益）；
  它由「归属计数不匹配 → 整组失败」的既有用例覆盖非并发路径。
- `task_error_test.go` 只能保证「哨兵有映射」，不能阻止别人再写第二份映射表。
  这类回归只能靠真实网关冒烟或 handler 集成测试发现（§21.5 已记）。

### 21.7 第三轮对抗性复审（2026-10-24，全新上下文）

第三个全新上下文的只读审查者。第二轮修复被确认有效（A1 边界、`FOR SHARE` 真实发射与锁序、
老师端 helper 替换后落库字节一致、文档与代码、统计口径、软删过滤、时区、确定性输出全部核过）。
新提 2 CONFIRMED / 1 POSSIBLE / 4 NOTE，处置如下。

| # | 结论 | 复核 | 处置 |
| --- | --- | --- | --- |
| 1 | 老师端 `CreateTask` / `UpdateTask` 仍无标题列宽校验：201 字符 → MySQL 1406 → **500**；而我这轮加的注释声称「校验放 service，入口漏了也不会变成 500」 | 成立（既有漏洞，但注释已变成不实陈述） | ✅ 老师端创建/更新补 `validateTaskTitleLength`（哨兵直传，`teacherPortalErr` 补文案）；管理端两个入口也在 service 层补同一校验（binding 只是第一道）；补老师端 200/201 用例 |
| 2 | `task_error_test.go` 的表格漏了 `ErrTaskTitleTooLong`：删掉该映射分支用例仍全绿 | 成立（我自己加哨兵时没同步表格） | ✅ 补一行；顺带确认表格的 10→11 项全部命中 |
| 3 | 活动任务永远无法结束：`status` 在批量编辑、管理端单编辑、老师端单编辑三处都被拒，也没有活动级动作；过期的活动在 `active_tasks` 里永远是 active，`?status=0` 永远查不到 | 成立，但这是 **D12/tasks 7.1 的取舍**（批量编辑只允许四个安全字段），不是实现走样 | ⏸ **不擅自加端点**，记为待定决策（见下） |
| 4 | 所有 C-2 手写 `Table(...)` 查询都显式过滤软删 | 成立（无缺陷） | — |
| 5 | 迁移从 INT 改 BIGINT 后，没有「已执行 INT 草案的库」的升级语句 | 成立（理论上；本文件的 INT 版从未离开工作区） | ✅ SQL 注释补 `MODIFY ... BIGINT NULL` 升级语句 |
| 6 | 契约写「数值/指针只要键出现就拒绝」，但 JSON `null` 实际等价于缺省 | 成立 | ✅ 契约改为「出现非 `null` 的键就拒绝，显式 `null` 与缺省同义」 |
| 7 | `GET /admin/tasks/:id/checkins` 无分页：`status=all` 对学期长的每日任务可能一次返回几千行 | 成立（新端点，尚无 UI 消费方，正是定契约的时机） | ✅ 加 `page` / `page_size`（缺省 1/20，上限 100），`total` 为过滤后总数；补分页用例。handler 用**独立的分页 query 结构**绑定，不复用 `listTasksQuery`（它的 `status` 是 `*int8`，会和这里的字符串 status 抢同一个 query 键） |

**本轮验证**：`go build ./...` / `go vet ./...` / `golangci-lint`（改动包 0 issues）通过；
C-2 用例 41 条（当时）+ handler 映射用例通过；全量任务回归 **350.6s** 通过（覆盖老师端创建/更新的标题校验）；
新增变异 2 项（老师端列宽、公开列表分页）**真实断言失败**。

**待定决策——活动如何结束（当日已由用户拍板，见 §21.8）**
现状：活动任务的 `status` 在三处写路径都被拒（`PUT /admin/task-activities` 的 `hasStructuralField`、
管理端与老师端的活动任务单编辑禁令），也没有任何活动级动作能改 `status`；创建恒为 `1`。
后果：过期活动在 `GET /admin/tasks/stats` 的 `active_tasks` 里永远算 active，`?status=0` 永远查不到，
唯一出路是 `DELETE /admin/task-activities/:id`（会连历史提交一起软删）。
两个候选：
1. **允许活动批量编辑 `status`**（0=整组结束）：C-1 已裁定「status 不是结构字段」，
   与那条口径一致，改动最小（DTO + 白名单 + 契约 + 用例）。
2. **新增 `POST /admin/task-activities/:id/end`**：语义更明确，但多一个端点与一套权限/文案。
两个方案都**不改变**「过期后家长端不可见」——可见性由 `end_date` 决定，`status` 只影响管理端统计与列表筛选。

### 21.8 决策落地：活动允许批量 `status`（2026-10-24，用户拍板）

§21.7 的第 3 条（活动无法结束）由用户选择「允许批量编辑 `status`」。实现与口径：

- `PUT /admin/task-activities/:activity_id` 的白名单从四个字段扩到五个：`title` / `description` /
  `learning_link` / `allow_public` / **`status`**。`status` 只接受 `0`（整组结束）/ `1`（进行中），
  其他值 `BAD_STATUS` → 422。
- **`status` 不是结构字段**（C-1 裁定），所以在 `UpdateActivity` 里单独从 `assertTaskEditable`
  的判定集中摘出：有提交的活动任务**仍可整组结束**（变异 20 守护）。
- **单条路径仍然拒绝**：`PUT /admin/tasks/:id` 与老师端 `PUT /teacher/tasks/:id` 对活动任务
  依旧不允许改 `status`——否则一个活动会出现「A 班已结束、B 班还在进行」，统计与 `by_activity`
  聚合都会失真。结束只能整组做。
- 可见性不受影响：家长端/老师端的任务有效性仍按 `status=1 AND start_date<=今天<=end_date`
  判定，`status` 只影响管理端统计（`active_tasks` / `completed_tasks`）与列表筛选。
- 合同与 `tasks.md` 7.1 已同步（`tasks.md` 里标注为 C-2 追加、引用本节）。

验证：`go build ./...`、`go vet ./...`、`golangci-lint`（改动包）0 issues；
C-2 用例 **43 条**（活动 16，新增「整组结束」「单条仍拒绝 status」两条）通过；
全量任务回归 **356.2s** 通过；变异 20（批量结束不退让结构锁）**真实断言失败**。

---

## 22. C-3 开工准备（2026-10-24，逐文件核实）

起点：`feat/parent-growth-scope` @ `d51f000`（已 push）。C-2 已闭合（7.1–7.4、8.3），C-1 的班级罐与建/改任务字段也已就绪。**不要从零假设现状**——下表是 2026-10-24 逐文件看过的结果。

### 22.1 现状（管理端）

| 文件 | 现状 | 缺口 |
| --- | --- | --- |
| `admin/pages/tasks/index.vue` | 只有列表 + 删除按钮；表格列含 `completion_rate`（后端已下线 → **该列为空**） | 11.1 全部 |
| `admin/api/tasks.ts` | `createTask` / `updateTask` 只有 title/description/class/frequency/dates；无活动端点、无公开列表、`getTaskStats` 返回 `Record<string,unknown>` | 11.1–11.3 |
| `admin/types/api.d.ts` `TaskItem` | 仍含 `completion_rate`，缺 learning_link/target/allow_public/ritual/star_*/activity_id 与新统计字段 | 11.1 |
| `admin/pages/statistics/index.vue` | 读 `avg_completion_rate`（后端已下线 → **显示空白**） | 11.2 |
| `admin/pages/classes/index.vue` | 有学年列与操作列，**没有**星星罐入口 | 11.4 的班级罐设置 |
| `admin/constants/menus.ts` | 菜单权限码已是 `task:view` | 无需改 |
| 活动/公开/星星罐相关代码 | `grep` 无任何引用 | 11.1/11.3 |

### 22.2 现状（老师端，C-4 用，本窗口别动）

> ⚠️ **本节是 C-4 开工前的快照**（2026-10-24 上午）。C-4 已落地，现行状态见 **§24**。

- `teacher/src/pages/tasks/index.vue:200`：「按班级」仍是置灰的 `view`，提示「当前暂未开放」。
- `teacher/src/pages/tasks/detail.vue`：已能展示 `star_scope=class` 的**本任务**贡献，但拿不到班级罐总进度。
- `teacher/src/api/teacher-tasks.js`：**没有** star-jar 的封装（C-1 已交付 `/teacher/class/star-jar` 端点）。

### 22.3 后端已就绪的接口（C-3 直接用，不要改后端语义）

| 用途 | 接口 |
| --- | --- |
| 任务列表 | `GET /admin/tasks?class_id=&status=&page=&page_size=`（**无** completion_rate） |
| 任务详情 | `GET /admin/tasks/:id`（含 `submission_count`/`responded_count`/`pending_count`/`response_rate`/`participation_rate`/`notify_target_count`） |
| 独立任务建/改 | `POST /admin/tasks`、`PUT /admin/tasks/:id`（新字段、指针缺省=保留、结构锁、`learning_link` 必填） |
| 多班活动 | `POST/PUT/DELETE /admin/task-activities[/:activity_id]`（服务端返回 `activity_id`/`task_ids`；批量只收安全字段 + `status`） |
| 统计 | `GET /admin/tasks/stats?start_date=&end_date=`（新字段见 `api-admin/10-tasks.md`；无分母时为 `null`） |
| 公开内容 | `GET /admin/tasks/:id/checkins?status=public|revoked|all&page=&page_size=` |
| 撤回公开 | `PUT /admin/tasks/checkins/:checkin_id/public`（只能 `{"revoke":true}`） |
| 班级罐 | `GET/PUT /admin/classes/:id/star-jar`（`goal>0`） |

契约：`backend/api/api-admin/10-tasks.md`（SoT）、`api-teacher/09-class-and-tasks.md`。

### 22.4 C-3 拆分（一次只做一组，组间先验证）

| 步骤 | 范围 | 验收 |
| --- | --- | --- |
| **C-3-a** | `types/api.d.ts` + `api/tasks.ts`：补齐 TaskItem/详情/统计/活动/公开列表的类型与函数 | `npm run build` 通过；无 `any` 泛滥 |
| **C-3-b** | `pages/tasks/index.vue`：列表去掉 `completion_rate`，加回应率/参与率列（或「待回应」列）、按 `activity_id` 分组展示、行操作（编辑/删除/公开内容） | 真实点击：列表不再空白列 |
| **C-3-c** | 建/改表单：独立任务与多班活动共用字段组件；`star_scope=class` 时隐藏任务目标并先查班级罐；`allow_public=false` 时提示「已公开内容将全部撤下」；有提交后禁用结构字段 | 真实点击：建独立任务、建 2 班活动、改一次标题两条都变、结束活动两条都变 |
| **C-3-d** | 统计接新字段 + 班级页星星罐设置 + 公开内容查看/撤回 | 真实点击：统计与列表一致；撤回后列表 `public_visible=false` |

### 22.5 约束与已知坑（C-3 必看）

1. **`completion_rate` 已下线**：表格里那一列现在是 `undefined`，必须换掉；`statistics` 页的 `avg_completion_rate` 同理（换 `response_rate`）。别为了「先跑起来」把后端字段加回去。
2. **比率可能是 `null`**：前端显示「—」，不要 `Number(null)` 成 0%，也不要把 `"100%"` 再乘 100。
3. **活动任务单删会被 422**：列表删除按钮对 `activity_id != ""` 的任务要隐藏或改成「删除整个活动」；单条编辑只允许四个安全字段 + `status`（`status` 单条也禁，只有活动批量能改）。
4. **`class_ids` 重复会被 422**（不是静默去重）；活动创建返回 `activity_id` + `task_ids`，前端不要自己拼 id。
5. **公开内容必须带分页**（缺省 20、上限 100）；`public_visible=false` 的行要标出「已不可见」而不是直接隐藏——管理员需要看到「批准了但墙上没有」。
6. **样式只引用 token**（Tailwind 类名，如 `bg-bg-card`/`text-text-primary`），禁止 `bg-[#fff]` 这类任意值；改完跑 `node scripts/check-tokens.mjs admin`。
7. **不要提交** `admin/.nuxt/`、`admin/.output/`（已被忽略，别 `git add -f`）、`docs/competitive/*`、`skills-lock.json`、`.impeccable/`。
8. 验证优先用 `admin-run` skill（起 Go 后端 + Nuxt + Playwright）；没有 Playwright 时按 memory `admin-ui-headless-cdp` 用构建产物 + 无头 Chrome CDP。**DOM 断言，不要用截图代替**。

### 22.6 新窗口开工提示词（直接复制）

```text
读 docs/handoffs/2026-10-21-parent-task-loop.md 的 §22（C-3 开工准备：现状核实 + 接口清单）。
起点：feat/parent-growth-scope @ 1813880。后端接口全部就绪，本窗口只做管理端前端。

开工前先读（按顺序）：
1. backend/api/api-admin/10-tasks.md —— 任务/活动/统计/公开列表/班级罐的请求与响应形状、错误码（SoT）
2. admin/pages/tasks/index.vue、admin/api/tasks.ts、admin/types/api.d.ts
3. admin/pages/statistics/index.vue、admin/pages/classes/index.vue、admin/constants/menus.ts
4. server/internal/service/task_activity.go、task_stats.go（只读，确认后端语义；不要改后端）

按四步做，每步做完先跑 npm run build，再进入下一步。

第 1 步 · 类型与 API 层
- admin/types/api.d.ts：TaskItem 去掉 completion_rate，补 learning_link / target_student_id /
  allow_public / recognition_ritual / star_scope / star_goal / activity_id；
  新增 TaskDetail、TaskStats、ActivityTask、PublicCheckin。
- admin/api/tasks.ts：createTask/updateTask 补新字段；新增 createActivity/updateActivity/deleteActivity、
  listTaskPublicCheckins（status + page + page_size）、getClassStarJar/setClassStarJar；
  getTaskStats 用具体类型替换 Record<string, unknown>。
- 判据：npm run build 通过，无新增 any。

第 2 步 · 任务列表
- admin/pages/tasks/index.vue：
  · 删掉 completion_rate 列。列表接口不返回每任务比率（7.4 已下线），不要在前端重算；
    每任务指标放在详情抽屉里。
  · 按 activity_id 分组：同一活动折叠成一条（显示班级数/任务数），展开后是各班任务。
  · 行操作：编辑、删除。activity_id 非空的任务不给单删，只给「删除整个活动」。
- 判据（真实点击）：列表无空白列；活动折叠/展开正确；活动任务的单删被前端拦住。

第 3 步 · 建 / 改表单（独立任务 + 多班活动）
- 新建弹窗字段：标题、说明、learning_link（必填）、班级（可多选）、频率、起止日期、仪式、
  星星范围/目标、允许公开。
- star_scope=class：隐藏任务目标；先 GET /admin/classes/:id/star-jar，未配置则提示并禁用提交。
- 选 1 个班 → POST /admin/tasks；选多个班 → POST /admin/task-activities（activity_id 由服务端返回）。
- 关闭 allow_public 时二次确认「已公开内容将全部撤下」。
- 有提交后（submission_count>0）：频率/日期/班级/个体目标/仪式/星星范围一律禁用，
  只留标题/说明/学习关联/公开开关；活动任务即使没有提交也只允许这四个字段。
- 活动编辑弹窗：只发 title/description/learning_link/allow_public/status；结束活动 = status 0。
- 判据（真实点击）：建 1 个独立任务；建 2 班活动（2 条任务、同一 activity_id）；
  改一次标题两条都变；结束活动两条都变 status=0；关闭公开后该任务的公开内容全部不可见。

第 4 步 · 统计 + 班级罐 + 公开内容
- admin/pages/statistics/index.vue：avg_completion_rate 换成 response_rate / participation_rate /
  pending_over_48h / median_response_hours；null 显示「—」。
- 任务统计区：接 GET /admin/tasks/stats 的 response_rate / participation_rate / pending_count /
  pending_over_48h / median_response_hours / child_happy_rate / ack_rate / feedback_coverage /
  by_class / by_activity；比率 null 显示「—」。
- admin/pages/classes/index.vue：行操作加「星星罐」，弹窗读写 GET/PUT /admin/classes/:id/star-jar；
  目标必须 >0，展示 term / term_start / term_end。
- 公开内容入口：GET /admin/tasks/:id/checkins?status=public&page=&page_size=，逐条显示
  public_status / public_visible（不可见的要显式标注），撤回调
  PUT /admin/tasks/checkins/:checkin_id/public（请求体只能 {"revoke": true}）。
- 判据（真实点击）：统计与列表一致；班级罐设置后建 class scope 任务不再被拒；
  撤回后该条 public_visible=false。

硬约束
- 不改后端字段/接口语义；completion_rate / avg_completion_rate 已下线，不要加回后端字段。
- 比率可能是 null：显示「—」，不要 Number(null) 成 0%，也不要把 "100%" 再乘 100。
- 公开内容列表必须分页（缺省 20、上限 100）。
- 样式只用 design token 的 Tailwind 类名（如 bg-bg-card / text-text-primary），
  改完跑 node scripts/check-tokens.mjs admin。
- 不要提交 admin/.nuxt、admin/.output、docs/competitive/*、skills-lock.json、.impeccable/。

验证方式
- npm run build 通过。
- 用 admin-run skill（Go 后端 + Nuxt + Playwright）；没有 Playwright 时按 memory
  admin-ui-headless-cdp 用无头 Chrome CDP。做 DOM 断言与真实点击，不要用截图或 MCP 导航
  代替交互验证。

完成后
- 更新 handoff：新增 §23 C-3 落地（改了什么 / 语义决定 / 验证证据 / 未做缺口）。
- 只在 npm run build + 真实点击都通过后勾 11.1–11.4。
- 3.8 的老师端入口属 C-4；12.x 验证与 tasks.md 的 A/B 勾选补正属 C-5；归档留到 C-5 之后。
```

**下一窗口不要碰的三件事**：
1. 老师端页面与 `teacher/src/api/*`（C-4）。
2. `tasks.md` 里 A/B 的勾选状态（C-5 逐条核对后再改，现在改了也没有证据支撑）。
3. 归档（C-3/C-4/C-5 完成后才做）。

---

## 23. C-3 落地（2026-10-24）

起点 `1813880`，只改管理端前端 + 一处通用组件可选 prop，**后端零改动**。

### 23.1 改了什么

| 文件 | 改动 |
| --- | --- |
| `admin/types/api.d.ts` | `TaskItem` 去掉 `completion_rate`，补 `learning_link` / `target_student_id` / `allow_public` / `recognition_ritual` / `star_scope` / `star_goal` / `activity_id`；新增 `TaskDetail`、`TaskStatMetrics`、`TaskStats`（+`ByClass` / `ByActivity`）、`ActivityTask`、`PublicCheckin`、`ClassStarJar`，以及 `TaskFrequency` / `TaskStarScope` / `TaskRitual` / `TaskPublicStatus` 四个联合类型 |
| `admin/api/tasks.ts` | `getTask` 用 `TaskDetail`；`createTask` / `updateTask` 补新字段（`TaskCreateBody` / `TaskUpdateBody`）；新增 `createActivity` / `updateActivity` / `deleteActivity`、`listTaskPublicCheckins`、`revokeTaskPublicCheckin`、`getClassStarJar` / `setClassStarJar`；`getTaskStats` 由 `Record<string, unknown>` 换成 `TaskStats` |
| `admin/utils/tasks.ts`（新） | 频率 / 仪式 / 星星范围枚举与文案（口径对齐 `taskFrequencyLabel`）、`groupTasksByActivity`、`buildTaskListRows`（表格行模型）、`canDeleteSingleTask`、`lockedFieldsForTask` |
| `admin/utils/format.ts` | 新增 `formatRate` / `formatMetric`：`null` → 「—」，与 0% 区分 |
| `admin/components/common/DataTable.vue` | 新增**可选** `row-key` / `tree-props`（不传时行为不变），供任务列表按活动折叠 |
| `admin/pages/tasks/index.vue` | 列表（删完成率列、按 `activity_id` 折叠、行操作、详情抽屉指标）、建/改弹窗（独立任务 + 多班活动 + 活动整组编辑）、结构锁、`star_scope=class` 的班级罐前置校验、关闭公开二次确认、公开内容查看/撤回 |
| `admin/pages/statistics/index.vue` | `avg_completion_rate` → `response_rate` / `participation_rate` / `pending_over_48h` / `median_response_hours` + 反馈类三项，另加「按班级」「按活动 / 独立任务」两张明细表 |
| `admin/pages/classes/index.vue` | 行操作加「星星罐」，弹窗读写 `GET/PUT /admin/classes/:id/star-jar`（目标必须 > 0，展示 `term` / `term_start` / `term_end`） |

### 23.2 语义决定（下一窗口先看这 9 条再改）

1. **活动与独立任务共用同一弹窗**：`class_ids` 长度 1 → `POST /admin/tasks`，>1 → `POST /admin/task-activities`（`activity_id` 只认服务端返回）。
2. **编辑单条任务时班级是单选**，编辑活动时班级只读展示。多班语义只存在于活动；单条任务用多选会静默丢掉第二个班，活动端点又不接受 `class_ids`。
3. **活动组行只有「编辑活动 / 删除整个活动」**；展开后的各班任务行只有「详情」，删除按钮不渲染（后端 422），另留 `canDeleteSingleTask` 兜底守卫。
4. **组行的状态/公开取组内聚合**：任一任务进行中即「进行中」，全部 `allow_public` 才算「公开」——组行不是真实任务，不能拿第一行的值冒充。
5. **结构锁集中判定**（`lockedFieldsForTask`）：`activity_id` 非空 → 锁；`submission_count > 0` → 锁。锁定时只回传后端 `required` 的结构位（值未变），**省略** `star_goal` / `recognition_ritual` / `star_scope`：后端对 `star_goal` 是「出现非 null 就校验」，历史 `class_star + goal=0` 的脏数据会因此把「改标题」也挡成 422。
6. **`star_scope=class` 时隐藏任务星星目标**，并逐班 `GET star-jar`：任一班级未配置或查询失败 → 禁用提交并显示**分开**的原因（「未配置」与「查询失败」不是一回事）。
7. **列表不再算任何比率**（后端 7.4 已下线，前端重算只会制造第二个真相源）；每任务指标在详情抽屉现取 `GET /admin/tasks/:id`，`null` 一律「—」。
8. **撤回成功后自动把筛选切到「全部」**：撤回会让该行不再满足「公开/待审核」，留在原筛选里会直接消失，管理员就看不到 `public_visible=false` 的结果了。已批准但不可见的行用「已不可见」标签显式标出，不隐藏。
9. **公开状态/可见性一律用服务端字段**（`public_status` / `public_visible`），前端不拼条件；标签色 approved=success、pending=warning、revoked=danger、none=info。

### 23.3 验证证据

- `cd admin && npm run build` 通过（最后一版含「编辑时班级单选」）。
- `node scripts/check-tokens.mjs admin`：**改动文件 0 违规**（仓库既有 8 条在 `teachers/index.vue` 与 `assets/css/_variables.scss`，本次未触碰）。
- 真实点击：构建产物 + 无头 Chrome CDP（`PORT=3000 node .output/server/index.mjs` + `chrome --headless=new --remote-debugging-port=9333`），脚本 `_verify/c3_admin_tasks_ui.mjs`，**最终一次运行 107 项断言全绿（rc=0）**（含 §23.5 所列 5 轮审查加强后的请求体断言）。关键证据：

| 判据 | 证据 |
| --- | --- |
| 列表无空列 | 表头 `任务/班级/频率/起止日期/状态/公开/星星/操作`，无「完成率」 |
| 建独立任务 | UI 创建 → `activity_id=""`、`learning_link` 正确、`allow_public=true`、班级落库 |
| 建多班活动 | 选 3 个班 → 3 条任务共享同一 `act_...`；组行显示「3 个班」 |
| 活动折叠/展开 | 组行「多班活动」标签；展开出 3 个 child（各班名 + 频率 + 日期 + 「随活动编辑」） |
| 活动任务不给单删 | 组行按钮 `['', '编辑活动', '删除整个活动']`；child 行为 `详情/随活动编辑`，**无删除** |
| 改一次标题整组生效 | 经活动端点保存后 3 条任务标题同时变化 |
| 结束活动 | 3 条任务 `status=0` |
| 关闭公开二次确认 | 弹窗文案含「已公开的内容将全部撤下」；确认后 `allow_public=false` |
| 有提交后结构锁 | 既有任务 4583：警示「任务已有提交，只能修改标题、说明、学习关联与公开开关」，频率/日期 `.is-disabled`、标题可编辑 |
| 统计新口径 | 回应率/参与率/超 48 小时未回应/回应中位数/按班级/按活动 均在；无「平均完成率」；正文无 `NaN` / `null%`；明细 38 行 |
| 班级罐 | 弹窗读到 `学期 2031-2032-spring`、`学期起止 2031-02-20 ~ 2031-07-10`；PUT goal=30 后 GET `configured=true` |
| 班级罐解锁 class scope | 未配置的班会让提交按钮禁用并提示「当前学期未配置」；配置后提示「目标 30 颗」，创建成功且 `star_scope=class` |
| 公开内容查看/撤回 | 批准后行「已批准 / 在墙上」→ 撤回 → 「已撤回 / 已不可见」；接口 `public_status=revoked`、`public_visible=false` |

- **对开发库的副作用已还原**：`task_check_ins#1957` 恢复为「已申请待审核」（验证时用老师身份批准过）、本次验证创建的 `class_star_jars`（小一班 goal=30）已删除、`C3验证*` 任务全部清理（`SELECT COUNT(*) = 0`）。
- `nuxi typecheck` **在本仓库不可用作门禁**：它把 `.vue` 当 TS 解析（`teachers/index.vue`、`classes/index.vue` 等未改动文件在 HEAD 上就报 `TS1109/TS1005`），没有 `vue-tsc` 参与。判据仍是 `npm run build` + 真实点击。

### 23.4 未做缺口

1. **「班级罐进度封顶 100%」未做**：管理端不展示星星进度（jar 视图不含进度，按 decision §20.7-5 归 C-4 老师端）。班级罐**目标**设置按最新口径放在「班级管理 → 星星罐」。
2. **11.2 的「平均回应时长」**：后端只给 `median_response_hours`，页面显示中位数，不在前端自算平均值。
3. **活动组的「N 个班」只统计本页可见的任务行**（列表按任务分页），跨页活动会低估班数；要精确需后端按活动分页，未做。
4. **管理端不能批准公开**（契约只允许撤回），验证时用老师身份批准——这是设计如此，不是缺口。
5. **公开内容的 `media_url` 未渲染缩略图**，只按文本展示，避免 OSS 鉴权与排版噪音。
6. ~~**结束单条独立任务仍无入口**~~ —— 第三轮审查后已补（见 §23.5 第 3 轮）：编辑弹窗有「任务状态」单选，`status` 只在改动时发，且不受结构锁限制。
7. 3.8 老师端入口属 C-4；12.x 验证、`tasks.md` A/B 勾选补正属 C-5；归档留到 C-5 之后。
8. **统计页的 `by_activity` 明细不分页**：`activity_id` 为空的独立任务每条各占一行（实测 38 行），把区间拉长到一学期会到数百行。目前按「管理端桌面端、可滚动」接受；要收敛需要后端分页或前端虚拟滚动。

### 23.5 对抗性审查收敛记录（5 轮，2026-10-24）

C-3 共跑了 5 轮独立审查，**采纳 21 条、驳回 1 条**。逐轮叙述已合并进下表（这些审查记录此前未提交，故无 git 原文可回溯）；这里只留「结论 + 为什么」。

| 轮次 | 发现 | 采纳 | 关键修正（会影响后续改动的） |
| --- | --- | --- | --- |
| 1（自查） | 3 | 3 | 编辑提交体改**差异集**（整表单回传会把未改动的指针字段重写）；活动 `status` 只在改动时发；星星罐前置校验对齐后端口径 |
| 2（Standards+Spec 双轴） | 8 | 7 | `removeTask` 静默吞异常修掉；枚举/映射表并入 `utils/tasks.ts`；`TaskUpdateBody` 不再继承创建体（`learning_link` 在后端可选） |
| 3（干净上下文） | 7 | 7 | **P1**：差异基线必须用「打开弹窗时归一后的表单快照」——历史 `recognition_ritual=''` 的行否则改标题必 422（库里有真实行）；独立任务补「任务状态」；详情/编辑/星罐/公开列表加请求序号 |
| 4（只审修复代码） | 6 | 6 | `openCreate` 与弹窗关闭要作废在途编辑请求（否则新建弹窗被改成编辑并清空输入）；公开列表换任务先清空；单测从「恒真」改成真实链路 |
| 5（只审上一轮修复代码 + 单测） | 5 | 4 | 单测补 scope 正反向 + trim，并做**变异检查**证明非恒真；两个编辑入口 catch 补 seq 守卫；`!origin` 兜底改为抛错 |

**驳回 1 条**：混合公开组的「假确认」——活动编辑走 `submitActivityEdit`（`submitForm` 在 `mode==='edit-activity'` 时提前 return），只用 `activityAllowPublicBaseline`，第 896 行的判断在活动路径不可达。

**审查暴露的两类通用坑（值得带进 C-4）**：
1. **显示层归一 ≠ 数据改动**：任何「下拉给个默认值」的地方，差异判定必须拿归一后的快照当基线，否则会把默认值写回库（有提交时直接 422）。
2. **只在改动时发字段**：后端对指针字段是「缺省=保留」、对 `star_goal` 是「出现即改动」，整表单回传两边都会错。

---

## 24. C-4 落地（2026-10-24）

起点 `c036a55`。范围：3.7 的**班级侧**计数、3.8 的老师端 UI（`GET/PUT /teacher/class/star-jar` 的进度与设置入口）、9.2 的班级罐部分（「按班级」可选、隐藏任务目标、detail 口径文案）。

### 24.1 第 0 步：口径决定（写码前定）

**结论：方案 A —— 扩 `GET /teacher/class/star-jar`，追加 `stars` / `progress` / `over_goal`。**

| 方案 | 结论 | 理由 |
| --- | --- | --- |
| **A** 扩现有 GET | **采用** | ① 目标与星数必须来自**同一次 term 解析**：`findClassStarJar` 的 term 在无学年班级按自然年、有学年按 `academic_years` 起止，拆成两个端点会在学期边界（1/1、学年切换）读到两套 term，出现「目标 30、进度按上一学期算」。② 老师端卡片是一个整体，多一次请求只多一个失败面。③ 管理端响应可以**逐字节保持不变**（老师端走新 service 方法 `GetClassStarJarWithProgress`，管理端继续 `GetClassStarJar`），C-3 已验证的管理端契约不受影响。 |
| B 新建 `/star-jar/progress` | 驳回 | 同一资源的派生视图，没有独立生命周期；仍然解决不了两次调用之间的 term 漂移。 |

**计数口径**（已核对 design D6 第 96 行与 decision §20.7-5，不是猜的）：

- 只算 **`star_scope=class`** 的任务（`design.md`：「该班当前学期内所有 `StarScope=class` 任务中…」）；
- 只算**主回应**：计数落 `task_check_ins` 行，其 `response_id` 指向的回应 `ritual=class_star` 才计入；一次 `period` 回应覆盖 N 条提交就计 N 星；补充回应不写 `response_id`，天然不计入；
- 提交、回应、任务三层都过滤 `deleted_at IS NULL`；
- **「当前学期」按提交日期 `task_check_ins.date` 落在 `term_start..term_end` 内判定**——本轮唯一新定的细节。理由：罐按学期重置，跨学期的长任务应按提交日拆到两边；用任务起始日会把整条任务算进某一边。已写进 `backend/api/api-teacher/09-class-and-tasks.md`。

### 24.2 改了什么

| 文件 | 改动 |
| --- | --- |
| `server/internal/service/task_star_jar.go` | 新增 `GetClassStarJarWithProgress`（同一次 `now` 里读目标 + 算星数）、`classStarCountInTerm`（先取本班 class scope 任务 id，再按 `task_id` 计数）与 `starCountBase`（「什么算一颗星」的唯一谓词，`taskStarCounts` 与班级罐共用）、`loadClassStarJar`（两个读路径共用入口守卫）；`progress = floor(stars*100/goal)` 封顶 100，`goal<=0` 时 `progress=null`、`over_goal=false`；`GetClassStarJar` 签名与响应不变 |
| `server/internal/handler/http/teacher_portal.go` | `TeacherGetClassStarJar` 改调新方法；`teacherPortalErr` 增加 `ErrTaskStarJarMissing` → 422「请先配置班级星星罐」（与管理端 `writeTaskError` 同一句） |
| `server/internal/service/teacher_portal_service.go` | **C-1 遗留修复**：`CreateTask` / `UpdateTask` 把班级罐缺失折叠成了 `ErrTeacherInvalidParams`（HTTP 422「请求参数无效」），导致 spec 要求的「三个入口返回同一句引导」在老师端落空。现在只把 `ErrTaskStarJarMissing` 原样上抛，其余结构错误维持原错误码。另外 `taskStarCounts` 改为复用 `starCountBase`（行为不变） |
| `server/internal/service/task_rules.go` | **两学期无缝不重叠**（C-4 决策）：`classStarJarTermOf` 先算两学期的原始窗口（每对配置独立覆盖、缺失按自然学期兜底），再交给 `classStarJarTermWindow` 收口——后一个学期的起点 = 前一个结束次日；`term` 键不变。无学年班级改为自然年切分（spring 1/1–7/31、fall 8/1–12/31） |
| `server/internal/service/task_admin_c1_test.go` | 三条 term 期望随新口径更新（寒假空档 → spring 2031-01-21；无学年 → spring 到 7/31），并新增 `TestClassStarJarTermSealsGapBetweenSemesters` 钉住暑假空档收口 |
| `server/internal/service/task_response_supplement_test.go` | 测试夹具修复（对抗审查发现）：清理顺序按外键方向重排，补删 `messages`，并显式把 `current_semester` 设为 `fall`（提交都落在今天，学期窗口收窄后必须落在窗口内）。只动清理与夹具，不改业务断言 |
| `server/internal/service/task_admin_c1_test.go` | 跟着上面改一条断言：`TestTeacherCreateTaskRejectsInvalidEnums` 里 `StarScope=" class "` 触发班级罐校验后，期望值由 `ErrTeacherInvalidParams` 改成 `ErrTaskStarJarMissing`（原断言锁定的是被 spec 否定的旧行为）。**不是为了让测试变绿而改语义**——`task_rules.go` 的行为没动，只是老师端不再吞掉哨兵 |
| `backend/api/api-teacher/09-class-and-tasks.md` | 补 `stars` / `progress` / `over_goal` 的字段语义、计数口径、封顶与 `null` 规则；注明管理端响应不含进度 |
| `teacher/src/api/teacher-tasks.js` | 新增 `fetchClassStarJar()` / `setClassStarJar(goal)`，沿用 `http.js` 拦截器 |
| `teacher/src/pages/tasks/index.vue` | 删掉「班级星星罐暂未开放」占位；新增班级罐卡片（学期 / 目标 / 已集星 / 封顶进度条 / 设置或更新目标）；「按班级」在已配置时可选、未配置或读取失败时给**分开**的原因；`star_scope=class` 时隐藏并清零任务目标；创建前校验按任务场景的目标 > 0；`onShow` 与下拉刷新同时刷新罐进度 |
| `teacher/src/pages/tasks/detail.vue` | `star_scope=class` 的标签由「按班级（管理端配置）」改成「按班级（使用班级罐目标）」 |
| `server/internal/service/task_star_jar_progress_test.go`（新） | 5 条真实 MySQL 用例（见 24.4） |

### 24.3 语义决定（后续别改错）

1. **管理端响应不含进度**：`GET /admin/classes/:id/star-jar` 仍只有 `goal` / `configured` / `term*` / `updated_at`。老师端的新字段不能漏进管理端响应（有测试钉住）。
2. **PUT 只回目标，不回进度**：`SetClassStarJar` 是老师端与管理端共用的写入路径，保持原有响应形状。**老师端保存后必须重新 GET**——本轮 UI 就在这里错过一次：直接把 PUT 响应 `applyJar` 覆盖状态，把「已集 3 / 30」显示成「已集 0 / 2」。
3. **目标 0 不在前端拦**：请求照常发出，由后端 422 决定，再把文案换成「目标必须大于 0；0 表示未配置」。后端是唯一权威，前端只负责可读。
4. **「按班级」的可用性取决于后端**：未配置（`configured=false`）与读取失败（`jarError`）给不同的提示，不能合并成「未配置」。
5. **按班级时任务目标按 0 提交**：`star_goal` 在 class scope 下不生效，残留数字会被误当生效值。
6. **未配置时 `progress=null` 而不是 0**：0 会被画成「0% 进度」，与「还没设目标」混淆；判据是 `configured`。
7. **不展示任何个人星星/完成率排名**（spec MUST NOT）；卡片只显示集体进度。
8. **两学期必须无缝且不重叠**（2026-10-24 产品决定）：后一个学期的起点 = 前一个结束次日；`term` 键不变。**不要**把 `classStarJarTermOf` 改回「两个学期共用一个整年窗口」或「按配置原样留空档」——前者会让春季的星算进秋季罐，后者会让假期里的提交永远不计入。

### 24.4 验证证据

**后端**

- `go build ./...`、`go vet ./internal/service ./internal/handler/http` 通过。
- `go test ./internal/service -run 'TestClassStarJarProgress|TestTeacherTaskClassScopeMissingJar'` → ok（新增 5 条）：
  - `..._CountsOnlyClassScopeMainClassStar`：task scope 的 class_star 回应**不**进班级罐；改 class scope 后 period 覆盖 2 条 = 2 星；贴纸主回应不计；DB 直插的 class_star **补充回应**不计；软删提交/回应各减 1；
  - `..._OverGoalIsFlaggedAndCapped`：2/3 → 66% 且不超额；目标下调到 1 → 100% + `over_goal=true`；恰好 2/2 → 100% 不超额；未配置 → `progress=null` 且星数照实返回；
  - `..._TermClassAndKindergartenScoped`：跨班、跨园所、上一学期提交都不计入；无学年班级 term 为自然年；
  - `..._AdminViewUnchanged`：管理端响应没有 `stars/progress/over_goal`，老师端三个字段齐全；
  - `TestTeacherTaskClassScopeMissingJarIsActionable`：创建与更新为 class scope 时都返回 `ErrTaskStarJarMissing`（不再是笼统参数错误）。
- `go test ./internal/service -run 'Task|ParentTask|ClassStar' -timeout 900s`：
  - 改错误映射**前**的那一轮 → ok；
  - 改完后第一轮暴露出 C-1 的一条旧断言（`TestTeacherCreateTaskRejectsInvalidEnums` 的 `StarScope=" class "` 期望 `ErrTeacherInvalidParams`）——那是被 spec 否定的旧行为，已按 spec 改成 `ErrTaskStarJarMissing`（见 24.2 表最后一行）；
  - **全量复跑**（断言更新后）：`exit=0`、`ok kindergarten-server/internal/service 260.324s`；
  - **对抗性审查修复后的最终复跑**：`exit=0`、`ok … 241.202s`（含两段式计数查询、B1 夹具清理修复）；`TestSupplement_|TestClassStarJarProgress` 合并连跑 4 次全绿。

**接口层**（真实 8080 + 开发库，`_verify/c4_setup.py`，14 项断言全绿）

| 判据 | 证据 |
| --- | --- |
| 未配置时三个入口都拒且提示一致 | teacher / admin `POST /tasks`、admin `POST /task-activities` 全部 `422` + `请先配置班级星星罐`（修正前老师端是「请求参数无效」，见 24.2） |
| 管理端设 30 → 老师端显示 30 | admin `PUT /admin/classes/1/star-jar {goal:30}` → `configured=true`；老师端 GET `goal=30`、`stars=0`、`progress=0`、`over_goal=false`；管理端响应无进度字段 |
| N 条提交发 class_star → 已集 N | 建 `star_scope=class` 任务 + 3 条提交（日期落在 term 内；当时窗口 `2031-02-20..07-10`，收口后为 `2031-01-21..07-10`，种子日期仍在窗口内，E2E 已按新窗口复跑），逐条 `POST .../response {ritual:"class_star"}` → `stars=3`、`progress=10`、`over_goal=false`；任务自身 `star_count=3` |
| 目标设 0 被拒 | `PUT /teacher/class/star-jar {goal:0}` → `422` + `班级星星罐目标必须大于 0` |

**真实点击（小程序，老师端 小一班）**

| 判据 | 证据 |
| --- | --- |
| 卡片按后端数据渲染 | 页面文本：`班级星星罐 / 2031-2032-spring / 已集 3 / 30 / 更新目标`；进度条 `style.width = 10%` |
| 按班级可选并切到班级口径 | 点击 `.scope-opt`（第 2 项，触摸事件）→ `按班级` 获得 `active`，提示变为「按班级时使用下方班级罐目标（30 颗），本任务不再单独设目标」，任务目标输入框不再渲染 |
| 有目标时不能出现 >100% | 目标下调到 2 后卡片显示 `已超额 3/2`，进度条 `width=100%`（无 2% 之类溢出文案） |
| 更新目标走真实写接口 | 在目标输入框输入 2 后点击「更新目标」→ 后端 `class_star_jars.goal` 变为 2、`updated_at` 更新；重新进入页面读到 `已超额 3/2` |
| 未配置时的空态 | 清理验证数据后重新进入页面：卡片显示 `2031-2032-spring / 本学期还没设目标；设好之后才能创建「按班级」任务。`，按钮文案变为「设置目标」，`.scope-opt`「按班级」带 `disabled`，且没有进度条与 `已集` 行 |

- `npm run build:mp-weixin` 通过；`node scripts/check-tokens.mjs teacher` 对两个改动文件 **0 违规**（仓库既有 215 条不在本次文件里）。

### 24.5 未做 / 缺口（不要当成已验证）

1. **「目标 0 的两步反馈」只验证到一半**：后端 422 与文案已用真实请求验证；前端把文案换成「目标必须大于 0；0 表示未配置」的映射也已实现，但**没在运行中的小程序里抓到那条 toast**。原因是 IDE 工具链在这一轮中途失效：热重载后 CDP 合成的鼠标/触摸事件不再落到页面（页面**能渲染**、能读 DOM、`elementFromPoint` 命中正确，但 handler 不触发——用「未配置时点按班级应弹提示」复核过，同样无反应），`wechat_automator` 守护进程两次 120s 超时（memory `mcp-view-layer-wedge-cdp-tools` 的已知形态），`cli auto` / 重开项目 / 重启 IDE 都只恢复到「能渲染、不能输入」。修复前最后一次成功的输入是「设目标 30→2」那一轮（见 24.4 表）。因此 24.4 的渲染类断言可信，**交互类断言只覆盖到上面 4 条**。
   **C-5 补验方法**：IDE 干净重启 → 进 tasks → 输入 0 → 点「更新目标」→ 读 `wx-toast` 文案；或改用真机预览。
2. **详情页只改了标签**：`detail.vue` 没有展示班级罐总进度（3.8 没要求；本任务贡献已由 `star_count` 显示）。班级范围任务的**行内**仍然显示「本任务已集 N 星」而不是学期累计——这是有意的：`star_count` 是任务贡献，塞进行内会被读成「这个任务完成了班级罐」。
3. **管理端不展示班级罐进度**：C-3 的决定（§23.4-1），本轮不改。
4. **`doc` 未归档、`tasks.md` 的 A/B 勾选未动**（属 C-5）。
5. **家长端看不到班级集体进度**（spec 轴审查发现，非 C-4 范围）：spec「家长只看到班级集体进度，不看到个人排名」里，「不排名」已满足，但「看得到集体进度」在 `parent/` 没有任何入口，`tasks.md` §10 也没有对应任务。**目前无人承接**，需在 C-5 决定是补一条家长端只读展示还是显式写进规格的豁免。
6. ~~学期窗口存在空档 / 无学年班级两个学期同一区间~~ —— **已按产品决定修复（2026-10-24，方案一：两学期无缝不重叠）**，见下条：
   - `classStarJarTermOf` 现在先把两学期的原始窗口（配置优先、自然学期兜底）算出来，再统一收口：**按起点排序，后一个学期的起点 = 前一个结束的次日**。
   - 无学年班级：自然年切成 `spring` 1/1–7/31、`fall` 8/1–12/31（之前两学期都是 1/1–12/31，切学期只换键）。
   - 有学年且有配置：暑假空档（如 7/11–8/31）归 `fall`（fall 起点前移到 spring_end+1）；配置里 fall 在 spring 之前时（如 fall 2030-09-01..2031-01-20、spring 2031-02-20..2031-07-10），寒假空档归 `spring`（spring 起点前移到 fall_end+1 = 2031-01-21）。
   - **`term` 键不变**，只有区间收口；因此 goal 的学期归属不受影响（`validateClassStarJar` 仍按同一个 `term` 查行）。
   - 测试：`TestClassStarJarTermSealsGapBetweenSemesters`（暑假空档）、`TestClassStarJarTermUsesAcademicYearConfig` / `...NormalizesWhitespaceAcademicYear`（寒假空档 → 2031-01-21）、`TestClassStarJarTermFallsBackToNaturalYear`（无学年两段）。B1 夹具因此显式把 `current_semester` 设为 `fall`（提交都落在「今天」，否则 8–12 月跑测试会算出 0 星）。
7. **多班老师只能看/设「当前生效班级」的班级罐**（自查）：C-1 的契约规定 `GET/PUT /teacher/class/star-jar` 的班级取自登录态、**不接受 `class_id`**（防越权）。代价是同时带两个班的老师（`class_teachers` 多行）无法在老师端给另一个班设目标，只能走管理端。若产品要求老师端支持多班，需在契约里显式加 `class_id` + 归属校验，属接口变更。
8. **老师的主班级被软删后，班级罐卡片会永久停在校验错误**（干净复审发现，既有缺陷）：`ResolveContext` 选第一个 `class_teachers` 行时 join 了 `classes` 但**没有过滤 `classes.deleted_at IS NULL`**，随后 `classStarJarTermOf` 返回 `ErrClassNotFound`，卡片显示「班级不存在，点击重试」且重试必然失败。属老师身份解析路径（影响所有老师端接口，不只是本卡片），不在 C-4 修。

### 24.6 对抗性审查与处置（2026-10-24）

**方式**：三个并行独立审查（Standards 轴 / Spec 轴 / 对抗性正确性）+ 自查，之后又开了一轮**干净上下文的独立复审**（不喂前一轮结论，只给代码/规格/攻击面）。Spec 与对抗性两份先回传；Standards 轴的报告迟到（工具故障后重发），共 2 条硬违规 + 7 条判断项（16–23）；第二轮干净复审给出 1 条 P1 + 7 条 P2（24–31）。

| # | 来源 | 发现 | 判定 | 处置 |
| --- | --- | --- | --- | --- |
| 1 | 对抗 | B1 夹具清理顺序是 `students → classes → student_parents`，与它自己的注释相反；`fk_student_parents_student` / `fk_students_class` 每次都拦下删除，残留园所/班级让下一轮 setup 撞 1062/1452。新测试是第一个反复压测该夹具的用例 | 属实（P1，测试基建） | 修：顺序改为 `student_parents → students → classes → files → users → kindergarten` |
| 2 | 对抗 | 回应/发布通知写 `messages`（`fk_messages_user`），B1 清理漏删这一张 → `users` 删不掉 → 残留 → 下一轮 1062 | 属实（P1，同上） | 修：补 `messages` 清理（C1 夹具早有这一步） |
| 3 | 自查 | `detail.vue:969` 仍写「班级星星罐暂未开放（需管理端先配置）」——本轮已开放，用户可见的失实文案 | 属实（P1） | 改文案；`detail.vue:52` 的注释同步改 |
| 4 | 自查 | 单条三表 JOIN 让优化器从 `task_responses` 出发（EXPLAIN 实测首表 = `task_responses`，`ritual` 无索引），卡片每次 `onShow` 都可能扫全表回应 | 属实（P2 性能） | 改为先 `Pluck` 本班 class scope 任务 id、再按 `task_id` 计数；EXPLAIN 首表变为 `task_check_ins`（`idx_task_student_date`） |
| 5 | 对抗 | 首帧 `jarConfigured=false` 把「加载中」显示成「未配置」，「按班级」看似被禁用且提示错误 | 属实 | 加 `jarLoading`，加载中/读取失败/未配置三态分开 |
| 6 | 对抗 | 读失败后仍可用输入框里的过期值盲写；`onShow` 的慢 GET 可能晚于保存后的刷新落地，把新进度刷回旧值 | 属实 | 读失败禁用保存并要求先重试；`jarSeq` 请求序号守卫 |
| 7 | 对抗 | 失败保存连弹两条 toast（`http.js` 一条 + 页面一条） | 属实 | 只在 `goal<=0` 时映射文案，其它失败交给 `http.js` |
| 8 | Spec | `tasks.md` 3.7 注记把「未配置返回 `null`」写成缺口，并指向 §24.5 里并不存在的条目 | 属实（文档） | 改为「有意决定，理由见 §24.3-6」 |
| 9 | Spec | `.goal-save` 用 84/22 而非组件 token `$component-input-height`(88) / `$component-input-radius`(24) | 属实（judgement） | 保留 84/22（同一行的 `.input` 就是 84/22，用 token 会高 4rpx 错位），补注释说明 |
| 10 | Spec | 家长端看不到班级集体进度（spec：「家长只看到班级集体进度，不看到个人排名」） | 属实，但**不属 C-4** | 记 §24.5-5：`parent/` 无入口、`tasks.md` §10 无任务，需 C-5 决定补 UI 或写豁免 |
| 11 | 对抗 | 学期窗口有空档（有学年配置时 7/11–8/31 不属于任何学期）；无学年班级 spring/fall 区间同为整自然年，切学期只换 goal 键、星数不清零 | 属实，属 **C-1 的 term 语义** | 记 §24.5-6；改 `classStarJarTermOf` 会动 goal 校验的学期归属，需产品确认 |
| 12 | 对抗 | 任务行的 `star_count` 与班级罐不可比（可能「任务 5 星 / 班级罐 2 星」） | 口径不同，非缺陷 | 不改；行内就该显示本任务贡献（§24.5-2） |
| 13 | 对抗 | 「EXPLAIN 我没跑；不认同优化器会从 `task_responses` 出发，`parent_tasks` 才是天然驱动表」 | **驳回** | 我跑了 EXPLAIN：改前首表确实是 `task_responses`（`ref idx_task_responses_deleted_at`），改后是 `task_check_ins`（`ref idx_task_student_date`）。优化器是否走 `parent_tasks` 取决于统计信息，不能靠推断 |
| 14 | Spec | class scope 任务行没有显示学期累计 | **驳回**（判断） | 卡片已显示累计；行内再显示一次会让「本任务进度」和「班级进度」混淆，且违反 §19.2-2（`star_count` 不得冒充班级罐） |
| 15 | Spec | 老师端错误映射属范围外改动 | **驳回** | spec 明文要求三入口同文案；且只做映射，`validateClassStarJar` 仍是唯一实现 |
| 16 | Standards | `classStarCountInTerm` 与 `taskStarCounts` 各写一份「什么算一颗星」的谓词，注释还声称「不另写一套」 | 属实（CLAUDE.md 纲领 #2） | 修：抽出 `starCountBase`，按任务与按班级两条路径共用同一份 JOIN/条件 |
| 17 | Standards | `view["goal"].(int)` 若将来变成 int64 会静默把进度降级成 null，与「未配置」无法区分 | 属实 | 修：进度直接读 `row.Goal`，不再对 view 做类型断言 |
| 18 | Standards | `GetClassStarJar` 与 `GetClassStarJarWithProgress` 各写一遍园所/班级守卫 | 属实 | 修：抽出 `loadClassStarJar`（仍保证同一个 `now`） |
| 19 | Standards | service 内用未导出 `errTaskStarJarMissing`，handler 用导出别名 | 属实（一致性） | 修：两处统一成 `ErrTaskStarJarMissing` |
| 20 | Standards | `.goal-save` 硬编码 84/22 绕过组件 token | 部分属实 | 已补注释说明「同一行 `.input` 是 84/22，用 88/24 会高 4rpx 错位」；尺寸保留 |
| 21 | Standards | `applyJar` 把「成功但空载荷」当成未配置，正是 `loadJar` 注释禁止的静默降级 | 属实 | 修：空载荷转成可重试的错误态 |
| 22 | Standards | 任务行与班级罐两处进度条 markup 重复 | **驳回** | 两处数据来源不同（任务罐 / 班级罐），为一条 8rpx 的横条抽组件不划算 |
| 23 | Standards | 3.7/3.8 勾选时没有登记 `TC-NEW-*` | 记账 | 用例登记是 12.1（C-5 范围），本轮按要求不动 A/B 与 12.x |
| 24 | 干净复审（第二轮） | **P1**：无学年班级的两个学期共用同一日期窗口，春季的星在切到 fall 后仍被算进秋季罐；有学年时 7/11–8/31 又是空档 | 属实（C-1 term 语义） | **已按产品决定修复**（方案一：两学期无缝不重叠）：后一学期起点 = 前一学期结束次日；`term` 键不变。细节与测试见 §24.5-6 |
| 25 | 干净复审 | 任务创建失败连弹两条 toast（`http.js` + `addTask` catch），新引入的 422 正好走这条路 | 属实（既有代码，新路径暴露） | 修：`addTask` catch 不再重复弹，只对既无 code 又无 message 的异常打日志 |
| 26 | 干净复审 | `saveGoal` 把任意失败都归因成「目标必须大于 0」（断网时输入 0 也这么说） | 属实 | 修：只在 `e.code === 42200` 且 `goal<=0` 时映射，其它交给 `http.js` |
| 27 | 干净复审 | 取任务 id 的 `Pluck` 没有日期下界，跨年累积会把该班历史上所有 class scope 任务全读出来 | 属实 | 修：加 `end_date >= term.Start AND start_date <= term.End`。前提是「提交日期必落在任务起止内」——生产端由家长提交校验保证；**验证脚本种数据时必须让任务区间罩住提交日期**，否则会算 0 星（本轮 E2E 实际踩到并修正了脚本） |
| 28 | 干净复审 | `starCountBase` 未校验 `r.task_id = c.task_id` | 属实（防御） | 修：JOIN 条件补上；主键比较零成本，一次错误回填就会静默多计 |
| 29 | 干净复审 | `UpdateTask` 先校验班级罐、后判结构锁：已有提交的任务先报「请先配置」，配完重试仍被结构锁拒绝 | 属实（误导） | 修：班级罐校验移到 `assertTaskEditable` 之后 |
| 30 | 干净复审 | 老师主班级被软删时 `ResolveContext` 未过滤 `classes.deleted_at`，卡片永久停在校验错误态 | 属实（既有缺陷） | 记账 §24.5-8：属身份解析路径，不在 C-4 修 |
| 31 | 干净复审 | 契约没写「未配置时 `stars` 仍如实返回」 | 属实 | 修：契约补一句，并说明客户端应据 `configured` 而非 `stars>0` 判断 |

**未采纳但需记住**：对抗审查提到的「`response_id` 跨任务指向」结构上可能（计数不校验 `r.task_id = c.task_id`），但现有写入路径产生不了；`taskStarCounts` 同样是这个形状。真要防，应在两张表加一致性约束，不是在这里加条件。

**修复后复验（2026-10-24）**

- 计数 SQL 改成两段式后 EXPLAIN 首表：`task_check_ins`（`ref idx_task_student_date`）→ `task_responses`（`eq_ref PRIMARY`）。
- `go test ./internal/service -run 'Task|ParentTask|ClassStar'` → `exit=0`、`ok 241.202s`（**在 Standards 轴那批重构之前**）。
- Standards 轴采纳项（`starCountBase` 共享谓词 / `loadClassStarJar` / 进度读 `row.Goal` / 统一导出哨兵）落完后：`go build ./...`、`go vet ./...`、`golangci-lint`（改动两包）通过；定向用例 `TestClassStarJarProgress*|TestTeacherTaskRead_Star*|TestTeacherTaskClassScopeMissingJar|TestTeacherCreateTaskRejectsInvalidEnums` → ok（33.6s，覆盖 `taskStarCounts` 的 list/detail 断言）。
- **重构后的全量复跑没有完成**：后台任务被系统以「内存不足」中止（不是用例失败），按要求未擅自重启。下次开窗口先补这一条：`go test ./internal/service -run 'Task|ParentTask|ClassStar' -timeout 900s`。
- `TestSupplement_|TestClassStarJarProgress` 合并连跑 4 次全绿（修夹具前是交替红）——这一轮在 `starCountBase` 重构之前。
- 起 8080 重跑 `_verify/c4_setup.py`：14 项断言全绿（含 `stars=3 / progress=10`、三入口同文案、goal=0 → 422），随后 `_verify/c4_cleanup.py` 还原：C4 任务 0 条、kg4 班级罐 0 行、B1 夹具行 0 条。
- 老师端 `npm run build:mp-weixin` 通过；`check-tokens teacher` 对改动文件 0 违规。
- 第二轮干净复审采纳项（共享谓词补 `r.task_id=c.task_id`、任务 id 加日期上下界、`UpdateTask` 校验顺序、toast 去重、422 精确映射）落完后：`go build ./...`、`go vet` 通过；定向用例 `TestClassStarJarProgress|TestTeacherTaskRead|TestTeacherTaskClassScopeMissingJar|TestTeacherCreateTaskRejectsInvalidEnums|TestTeacherUpdateTask|TestAdminUpdateTask` → ok（73.5s）。
- 中等回归子集 `TestSupplement_|TestTeacherTaskRead|TestClassStarJar|TestTaskPublic` → `ok 64.2s`。
- 两学期无缝收口（`task_rules.go`）落完后：`go build`/`go vet`/`golangci-lint` 通过；`TestClassStarJarProgress*|TestClassStarJarTerm*|TestValidateClassStarJar*|TestTeacherTaskRead*|TestSupplement_*` → `ok 64.3s`（含新增的 `TestClassStarJarTermSealsGapBetweenSemesters`）。
- 收口后再跑一次真实接口 E2E（`_verify/c4_setup.py`）：**14 项全绿**，`term_start` 已是 `2031-01-21`、`stars=3 / progress=10`；随后清理干净、停后端。踩到并修正了一个脚本前提：种子提交的日期必须落在**任务起止区间**内（生产不变式），否则新的「任务区间与学期有交集」过滤会把任务排除、算出 0 星。

---

## 25. tasks.md 勾选补正（2026-10-25，C-5 第一步）

**背景**：`tasks.md` 长期「做了没勾」。开工前是 **23 勾 / 67 未勾**，与实际交付严重不符（例：12.1 的 46 个 TC 编号早就登记齐了，1.x/2.x 大部分已落地）。本轮只盘勾选，**不动代码**。

**方法**：5 路并行只读审计（按 §1–§12 分区），要求每条结论必须给 `file:line` 或真实 grep 输出；构建/点击类静态不可证的只能标 PARTIAL。回传后由主会话**逐条复核锚点**才落笔——「子代理说 DONE」不足以进 `tasks.md`。

**结果：51 勾 / 39 未勾 / 共 90**（本轮新勾 28 条）：

| 组 | 新勾条目 |
| --- | --- |
| 1 | 1.1、1.2、1.4、1.6 |
| 2 | 2.4、2.8、2.9 |
| 3 | 3.3、3.9、3.10 |
| 5 | 5.5 |
| 6 | 6.1、6.3、6.4、6.5、6.6 |
| 8 | 8.1、8.2、8.5 |
| 9 | 9.1、9.2、9.6 |
| 10 | 10.3、10.5、10.7、10.8 |
| 12 | 12.1、12.4 |

每条都补了 `<br>` 注记（格式同 3.7/3.8：`落地 <日期>（切片，证据见 …）`）。**这些注记是后续归档与复核的唯一来源，不要删。**

### 25.1 两处判断修正（与最初估计不同，值得记住）

1. **10.4 没勾**。「今晚一件事」卡片代码在（`parent/src/pages/index/index.vue:296-307`），但验收写的是「真实点击进入详情」，而 §16.3、§18.3 两张真实点击表里**都没有点这张卡片的记录**。同批的 10.3/10.5/10.7/10.8 都找到了记录，唯独它没有 → 不勾，留作 5 分钟补验项。
2. **10.5 / 10.8 一度被我误判为 PARTIAL**。原因是只翻到 §18.3（老师端点击表），没找到家长端记录；翻到 §16.3 后确认：#8 点「孩子很开心」落库 `task_response_reactions (699,61,child_happy)`、#4/#6/#10–#13 走完全链、末段记了 build 与 token 结果 → 勾。
   **教训：本文件有两张真实点击表——§16.3 家长端、§18.3 老师端。查证据时两张都要看，只看一张会低估已完成度。**

### 25.2 写进注记、但**不干净**的地方（不要当成已清理）

| 条目 | 残留 | 影响 |
| --- | --- | --- |
| 6.3 | `task_checkin_points` 仍被 `readPointsFromExtra`（`point_service.go:232`）读入后丢弃，`settings_service.go:97` 仍有该字段 | 死配置；`GetPointRules` / `UpdatePointRules` / 管理端页面都已无任务项 |
| 6.6 | `pointSourceLabel`（`point_service.go:33-48`）仍保留 `task_checkin → 任务打卡` | 仅用于回溯历史流水，不代表任务页还有积分 |
| 8.1 / 8.2 | `points` 在契约里各出现 3 / 4 次 | 全是「已移除」声明（`06-task.md:53/116/161`；`09-class-and-tasks.md:223/287/333/371`），无字段定义 |
| 9.2 | 「目标设 0 的 toast」仍未在运行中小程序里抓到 | 沿用 §24.5-1，属 C-5 补验项 |

### 25.3 剩余 39 条（按性质分，这才是 C-5 真正要干的活）

**真缺口 5 条**（功能/删除项本身没落地，不修不能归档）：

| 条目 | 问题 | 严重度 |
| --- | --- | --- |
| 3.11 | 老师端反馈汇总**全仓不存在**（`feedback-summary` / `FeedbackSummary` 零命中）。spec 写「老师端与管理端 SHALL 能看到反馈汇总」，管理端在 7.2 做了，老师端无人承接 | P1 |
| 4.5 | 公开申请缺「无文字无媒体」闸门（`Content == ""` / `MediaFileID == 0` / `MediaURL == ""` 零命中）→ 一键完成的空提交可申请并批准上墙 | P1（漏洞） |
| 1.5 | `Rewards` 没删干净：`domain/task.go:52` 字段仍在，`teacher_portal_service.go:2856/2999/3039` 仍读写（契约与前端已删） | P2 |
| 7.6 | `participation_rate` 分母是否排除「未绑定家长的学生」未决策；`task_stats.go:311` 直接用全部在读学生，无单测 | P2（需产品拍板） |
| 12.7 | 回归验证零执行证据（成长时光/通知/接送/积分页/看板 + 积分流水不产生） | 归档门禁 |

**实现缺一点 10 条**：`2.1`（谓词在 `parent_portal_service.go:1485-1492` 而非约定的 `task_repo.go`，行为等价）、`2.10`（`Status=0` 结束任务无二次确认）、`4.6`/`4.7`（产品未决策）、`8.4`（`action_path` 无 `/pages/` 与 `student_id` 语义；三份 README 索引陈旧）、`8.6`（契约里只有 `period_start`/`period_end`，`response_type` 的 `single|period` 枚举本身没有定义）、`8.7`（溯源表与契约字段一致性**审计结论，本轮未逐条复核**：3 个字段名不一致、8 个契约字段未在 design 声明）、`10.1`（首页不解析小程序码 `scene`）、`10.2`（缺「已回应」态——根因是列表契约没给字段，属设计性缺口）、`10.6`（未登录保留路径 + 登录后回跳未实现，`redirect|回跳|pendingPath` 零命中）。

> 其中 `2.10`、`4.6`、`4.7`、`8.7` 四条来自审计回传，**本轮未逐条复核**；开工时先自查再改。其余（`2.1`、`8.4`、`8.6`、`10.1`、`10.2`、`10.6`）已由主会话复核。

**只缺验证/单测 24 条**：`1.3`（列级需 `dbcheck` 直验）、`2.2/2.3/2.5/2.6/2.7/2.11`、`3.1/3.2/3.4`、`4.8`、`5.4`（真机扫码）、`5.6`（跨时区）、`6.2`（积分不变）、`9.3/9.4/9.5/9.7`、`10.4`、`12.2/12.3/12.5/12.6/12.8`。

### 25.4 归档门禁现状（2026-10-25）

**仍不可归档**，四条理由：

1. **规格未兑现**：3.11（老师端反馈汇总）与 4.5（空内容公开）都是 spec 明文要求，代码里没有。
2. **规格与实现冲突**：`specs/parent-task-loop/spec.md:145` 的 Scenario THEN 写「家长只看到班级集体进度」，但 `parent/` 没有任何入口，`tasks.md` §10 也没有对应任务 → 需决定「补只读展示」还是「改规格为豁免」。
3. **12.x 未全绿**：12.2/12.3/12.5/12.6/12.7/12.8 全部未勾，其中 12.6 依赖真机扫码、12.7 零证据。
4. **C-4 改动未提交**（见 §0 表格）。

**已通过**：`openspec validate --all` → **21 passed, 0 failed**（2026-10-25 复跑，唯一 warning 在 `spec/pickup-custody`，与本 change 无关）。

---

## 26. 新窗口开工提示词（C-5，2026-10-25 准备）

> 直接复制下面代码块给新会话。它假设新窗口**没有任何上一轮上下文**。

```text
项目：D:\yuanhaisu\personal_projects\kindergarten（养成星球，Go + uni-app + Nuxt）
分支：feat/parent-growth-scope
当前阶段：OpenSpec change「2026-10-20-redesign-parent-task-loop」的 C-5 收口（不新开 change）

## 第 0 步：先读，不要先动手

按顺序读，读完再回话：
1. docs/handoffs/2026-10-21-parent-task-loop.md 的 **§0**（当前状态）、**§25**（勾选补正全表）、**§24.5**（C-4 未验项）
2. openspec/changes/2026-10-20-redesign-parent-task-loop/tasks.md
   —— 现状 **51 勾 / 39 未勾**，未勾的分布与分类见 handoff §25.3
3. openspec/changes/2026-10-20-redesign-parent-task-loop/specs/parent-task-loop/spec.md
   —— 特别看 **:145** 那条与 parent/ 实现冲突的 Scenario
4. CLAUDE.md（硬约束，尤其「小程测试验证纪律」与「Known Issues」）

读完先回一段：你理解的本轮目标、你打算的顺序、以及任何与 §25 结论不一致的地方。
**在得到我确认前不要改任何代码，也不要勾任何 tasks.md 复选框。**

## 已知事实（不要重新发现）

- 工作区有 **C-4 的未提交改动**（老师端班级罐入口 + 进度）。git status 里还有 3 个与本次无关的路径：
  `docs/competitive/*`、`skills-lock.json`、`.impeccable/`、`docs/competitive/interview-guide.md` —— **不要提交它们，也不要删除**。
- 全量回归已跑过并通过：`go test ./internal/service -p 1 -parallel 2 -count=1` → ok 220.271s。
- `openspec validate --all` → 21 passed / 0 failed。
- 老师端**没有 tabBar**，任何跳转都用 `uni.navigateTo`，不存在 switchTab 场景。
- 家长端**有** tabBar（index/growth/pickup/profile），跳这四个必须 switchTab。

## 本轮要做的（严格按这个顺序）

### 1. 先提交 C-4
两个 commit：
- 代码：server/internal/{service,handler}/...、teacher/src/...、两个 api 契约、新增的 task_star_jar_progress_test.go
- 文档：docs/handoffs/...、openspec/.../tasks.md
提交前跑：`go build ./...`、`go vet ./...`、`node scripts/check-tokens.mjs teacher`。
commit message 末尾按仓库约定加 Co-Authored-By。

### 2. 补两个真缺口（P1）
- **3.11 老师端反馈汇总**：新增 `GET /teacher/tasks/:id/feedback-summary`（或并入任务详情），返回
  `feedback_coverage` / `child_happy_rate` / `ack_rate` + 按孩子的反馈计数。
  口径必须与 7.2 完全一致（按 `(response_id, parent_user_id)` 对计算，分母为回应覆盖的去重家长数，
  含通知失败；补充回应的反馈不计入）——**优先复用 7.2 现有实现，不要另写一套统计**。
  改契约 `backend/api/api-teacher/09-class-and-tasks.md` + 加单测。
- **4.5 空内容不可申请公开**：在 `POST /parent/tasks/checkins/:checkin_id/public-request` 的校验里
  拒绝「无文字且无媒体」的提交，返回明确错误；加单测；前端「一键完成」路径不给申请入口（已有条件，
  核对一遍）。改契约 `backend/api/api-parent/06-task.md`。

### 3. 产品决策（做之前必须问我，不要自己拍板）
- **spec.md:145 家长端集体进度**：补 parent/ 只读展示，还是改规格为「本期不提供」？
- **4.6 / 4.7**（任务结束后公开内容的老师回看/导出；PublicApprovedBy 清空后的审计留痕）：提供还是明确不提供？
- **7.6** participation_rate 分母是否排除未绑定家长的学生？

### 4. 补实现缺口（P2，可与 2 并行）
2.10（Status=0 二次确认）、8.4（action_path 语义 + 三份 README 索引）、8.6、8.7、10.1（scene 解析）、
10.2（列表契约补「已回应」所需字段）、10.6（未登录回跳）、1.5（删 Rewards）。

### 5. 验证与勾选（最后做，逐条补 evidence）
- 单测：2.2/2.3/2.5/2.6/2.7/2.11、3.1/3.2/3.4、6.2、12.2
- 真实点击/真机：9.3/9.4/9.5/9.7、10.4、12.3/12.5/12.6（真机扫码）
- 回归：12.7（成长时光/通知/接送/积分页/看板 + 不产生积分流水）
- 收工：12.8（两端 build + admin build + openspec validate --all + 杀 watcher）
- **每勾一条，必须在 `<br>` 里写清证据（file:line / 命令输出 / 点击记录）**。没有证据就不勾。

### 6. 归档
只有 39 条全部清零（或有明确豁免记录）后才执行 /opsx:archive，并把 delta 合并进 openspec/specs/。

## 纪律（违反会被退回）

- **Never Blind Guess**：不确定就停下来问，不要猜着改。
- 子代理/审查者的结论**不直接进仓库**，必须我自己复核锚点后才落笔（§25 就是这么做的）。
- 单测通过不是目的；不要为了让测试变绿而写浅层覆盖。
- 设计值一律用 token（`shared-design-tokens`），改完跑 `node scripts/check-tokens.mjs <teacher|parent>`。
- 收工按 PID 杀 `dev:mp-weixin` watcher，不要只 TaskStop。
- **不要**把 `configs/app.local.ini` / `.env` / `app.ini` 里的凭据提交进仓库。
- 破坏性命令（`refreshtasks`、`initcheck`）**不得**对生产库执行。
```

---

## 27. 附：本轮勾选用到的复核命令（可复用）

```bash
# 领域字段
rg -n 'ActivityID|TargetStudentID|LearningLink|AllowPublic|RecognitionRitual|StarScope|StarGoal|MediaFileID|ResponseID' server/internal/domain/task.go
# 四个 helper 是否仍在
rg -n '^func (validateTargetStudent|validateClassStarJar|assertTaskEditable|revokePublicForTask)' server/internal/service/task_rules.go
# 契约字段命中
rg -c 'target_student_id|has_unread_supplement|last_read_at|my_reaction' backend/api/api-parent/06-task.md
# 用例登记是否连续
rg -n 'TC-NEW-TSK-01(01|46)' docs/tests/TEST_CATALOG.md
# 勾选计数（改完必须回数）
rg -c '^- \[x\]' openspec/changes/2026-10-20-redesign-parent-task-loop/tasks.md
rg -c '^- \[ \]' openspec/changes/2026-10-20-redesign-parent-task-loop/tasks.md
```

---

## 28. C-5 落地（2026-10-25）

### 28.1 C-4 已提交

| commit | 内容 |
| --- | --- |
| `7ecd6c7` | C-4 代码：老师端班级罐入口/进度、term 两学期无缝、错误映射修复、5 条 MySQL 用例、两份 api 契约 |
| `25554b1` | C-4 文档：§24/§25 与 `tasks.md` 勾选补正（51 勾 / 39 未勾） |

提交前：`go build ./...`、`go vet ./...` 通过；`check-tokens teacher` 对 `pages/tasks/*` **0 违规**（仓库既有 215 条在别的文件）。工作区只留 `docs/competitive/*`、`skills-lock.json`、`.impeccable/`、`interview-guide.md` 未提交（按要求不动）。

### 28.2 产品决策（2026-10-25，已拍板）

| 项 | 决定 |
| --- | --- |
| `spec.md:145` 家长端集体进度 | **本期不提供**：已把 Scenario THEN 改为「不展示个人排名；班级集体进度在老师端展示，家长端本期不提供」（原句与实现冲突，非静默放过） |
| 4.6 结束后回看/导出 | **回看已有、导出不提供**：`TaskCheckins` 不过滤 `status`/有效期，契约已写明 |
| 4.7 审核留痕 | **写只读操作日志**：批准/拒绝/家长取消/关闭任务公开都在事务内写 `operation_logs` |
| 7.6 参与率分母 | **保持全部在读学生**（含未绑定家长），契约与单测已钉住 |
| 2.10 recognition ritual | **按 spec 锁死**：不加入 `taskSafeEditableFields`，有提交后不可改 |
| 2.1 范围谓词位置 | **保留现状 + 注记**：谓词在 `parent_portal_service.go` `taskScopeWhere`，四端共用 |

### 28.3 本轮代码改动（未提交）

| 文件 | 改动 |
| --- | --- |
| `server/internal/service/task_stats.go` | 抽出 `taskFeedbackTotalsOf`（反馈口径唯一实现），7.2 与新老师端汇总共用；`taskStatsReaction` 增 `UserID`（仅归属用） |
| `server/internal/service/teacher_portal_service.go` | 新增 `TaskFeedbackSummary`（3.11）：三个率 + `feedback_counts` + 按孩子 `children` + `notify_baseline_missing`/`unattributed_feedback`；跨班/跨园 404 |
| `server/internal/handler/http/teacher_portal.go`、`app/router.go` | `GET /teacher/tasks/:id/feedback-summary` |
| `server/internal/service/task_rules.go` | `ErrTaskPublicEmptyContent` + `checkinHasPublicContent`（4.5）；四个审计动作常量 + `publicAuditEntry` + `revokePublicForTask` 增 `kindergartenID/actorID/reason` 并写日志（4.7） |
| `server/internal/service/parent_portal_service.go` | 4.5 闸门；4.7 取消日志；`ResolveTaskContext`（2.2：任务链路不回退到第一个孩子）；抽出 `pendingTaskCount`（2.7，复用 `taskSubmittedInPeriod`，读失败不再静默）；列表新增 `responded`（10.2） |
| `server/internal/handler/http/parent_portal.go` | 4.5 错误文案；`parentTaskCtx` 改用 `ResolveTaskContext` |
| `server/internal/service/{task_service,task_activity}.go` | `revokePublicForTask` 传 actor/reason（管理端/活动） |
| `server/internal/service/operation_log_service.go` | 四个公开审核 action 的中文标签 |
| `server/internal/domain/task.go` | 删除 `Rewards`（1.5）；`TeacherTaskInput.Rewards` 与落库同步删除 |
| `parent/src/pages/tasks/index.vue` | 三态「待完成/已提交/已回应」+ `responded` 徽标 |
| `parent/src/utils/taskScene.js`（新） | 小程序码 `scene` 解析（详情与首页共用） |
| `parent/src/pages/index/index.vue` | `onLoad` 解析 `scene` 转场任务详情（10.1） |
| `parent/src/utils/pendingRoute.js`（新）、`api/http.js`、`pages/auth/login.vue`、`pages/messages/index.vue` | 401/未登录保留路径 + 登录后回跳（10.6） |
| `admin/pages/tasks/index.vue` | 「进行中 → 已结束」二次确认（2.10，任务与活动） |
| `admin/pages/logs/index.vue` | 日志页模块筛选补「任务」 |
| `backend/api/*` | 3.11/4.5/4.6/7.6/8.4/8.6 契约与三份 README 索引；`api-admin/19-logs.md` 补 action 说明 |
| `design.md` | 契约字段溯源表按实际契约修正/补齐（8.7） |

### 28.4 新增单测（全部真实 MySQL）

| 文件 | 用例 |
| --- | --- |
| `task_feedback_summary_test.go` | 6 条：无提交、部分+co-parent、补充不计入、周期按孩子归属、未知义务、跨班/跨园 |
| `task_public_test.go`（追加） | 空内容申请拒绝、审计留痕（批准→取消、任务级撤销）、action 标签 |
| `task_parent_flow_test.go` | `ResolveTaskContext`、跨班/个体提交拒绝、有效期窗口、三种频率、并发双击、pending 计数、媒体归属、积分不变 |
| `task_teacher_flow_test.go` | 待回应筛选与排序、汇总空列表/部分已回应/跨任务隔离、读标记时区 |
| `task_admin_c2_stats_test.go`（追加） | 参与率分母含未绑定学生 |

### 28.5 验证状态（2026-10-25 收口）

- `go build ./...`、`go vet ./...`、`golangci-lint`（改动三包）→ 通过。
- `openspec validate --all` → **21 passed / 0 failed**。
- `npm run build:mp-weixin`（parent、teacher）、`npm run build`（admin）→ 通过。
- `check-tokens`：改动文件 0 违规；基线 teacher 215 / parent 149。
- **新增单测全绿**（真实 MySQL）：`task_feedback_summary_test.go` 6 条、`task_parent_flow_test.go` 8 条、`task_teacher_flow_test.go` 3 条、`task_public_test.go` 新增 3 条、`task_admin_c2_stats_test.go` 新增 1 条。
- **真实点击（家长端）**：10.4 首页「今晚一件事」→ 详情；10.6 消息深链 → 详情 + 已读落库 + 「无效 token → 401 → 登录 → 回跳」；10.1 `scene=t%3D4322` 模拟扫码入口 → 详情；12.7 成长/通知/接送/积分/首页导航。
- **真实点击（老师端）**：9.4 工作台 → 列表 → 详情；9.3 贴纸单条回应（2158）、个别补充（2159）、周期汇总（2160），均落库且主回应不被改写。语音录制（`getRecorderManager`）无法在 IDE 自动化，未覆盖。
- **真实点击（管理端，无头 Chrome，脚本 exit=0）**：2.10 结束任务/活动二次确认（取消不落库、确认后 `status=0`）；12.5 三班活动共享 id、整组改标题、单删被拒、**删除活动整组消失**、统计与公开撤回。
- **全量 `go test ./internal/service`**：第一次 613s 后失败，日志被 `tail` 截断；第二次被系统内存中止；第三次完整落盘后定位到 **`Error 1040: Too many connections`**（每个用例新建连接池、从不关闭，MySQL `max_connections=151` 被打满），不是业务回归。已在 `openTaskB1DB` 限制 `MaxOpenConns=8` 并 `t.Cleanup` 关闭。第四次：**`ok 798.946s`，EXIT=0，0 条 FAIL/panic**。
- `tasks.md` 当前 **84 勾 / 6 未勾 / 共 90**。

### 28.6 真机项豁免与归档缺口（2026-10-25 用户决定）

**决定**：当前没有真机环境，真机相关项**记为后续项、不作为归档门禁**。相关条目已勾并注明「真机豁免」，实现与可自动化部分均已验证。

| 条目 | 已自动化验证 | 真机待办（后续） |
| --- | --- | --- |
| 5.4 | 家长端凭据 + `code_env_version` 单测；share-card 生成并落盘（§18.3 #21） | 扫码进入体验版/开发版家长端 |
| 9.3 | 贴纸/补充/周期汇总/公开审核真实点击；语音「上传+落库+渲染」已验证 | `getRecorderManager` 真机录音→上传→回应 |
| 9.5 | 详情页生成码、保存相册、转发提示（§18.3 #21） | 转发后真机扫码进入家长端 |
| 9.7 | 两端 build + token + 各段点击 | 原生 picker 真机点选、扫码链路 |
| 12.6 | `scene` 解析与转场（模拟入口）、非法 envVersion 单测 | 真机扫码，不从老师端进入 |

**12.3 已按「分段证据」落账（未做单轮双端联跑）**。尝试过 `scripts/dual-ide.ps1 -Kill -Auto`：第二实例启动后教师端项目未加载（窗口停在启动器，第二微信号登录态失效，`cli auto` 超时），恢复登录需要扫码 —— 与「无真机环境」同一约束。按用户 2026-10-25 授权回退：
- 每一段均有真实点击 + 库核对：个体任务可见性 §18.3 #7（+`TestParentTaskScopeThreeCases`）、家长提交 §16.3 #4、老师回应 §18.3 #15/#18、家长反馈 §16.3 #6/#8、个别补充 §18.3 #16、红点与「给孩子看」§16.3 #6/#9、读标记 §16.3 #6。
- 未做的只是「两端同一轮联跑」；真机/双端环境恢复后建议补一轮，把 12.3 的注记改回联跑证据。

**归档门禁状态：`openspec instructions apply` → `state: all_done`，90/90。** `openspec validate --all` 21 passed / 0 failed。其余门禁：12.2 全量回归 `ok 798.946s`、12.5 管理端脚本 exit=0、12.7 回归、12.8 收工命令。

> 后续真机待办（不影响本次归档）：5.4 扫码进入体验/开发版、9.3 真机录音、9.5 转发扫码、9.7 原生 picker 点选、12.6 真机扫码、12.3 双端联跑。真机可用时按对应 `<br>` 注记补验并更新。
