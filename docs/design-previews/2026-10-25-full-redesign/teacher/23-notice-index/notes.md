# 通知列表（老师端）

- **源码**：`teacher/src/pages/notice/index.vue` 已补（2026-10-27），该页由 `teacher/src/pages/index/index.vue:141` 的分享路径 `/pages/notice/index` 指向，并在 `pages.json` 注册；字段与状态按 `backend/api/api-teacher/04-notices.md` 的 `GET /teacher/notices` 设计。
- **设计系统**：user:nurture-planet-local，surface=teacher
- **方向**：把「发出去的通知」做成可追踪的工作列表，而不是第二个消息中心。首屏先回答「发了多少、家长看没看」，再按状态和类型筛选。
- **生成方式**：手写兜底（`handAuthored: true`）。OpenDesign 未参与；批次规则要求手写页在 notes 与 `_run.json` 中标注。

## 信息层级

1. 外壳：状态栏 + 导航栏「通知」+ 返回 + 胶囊；子页，无底部 tabBar。
2. 概要卡：全部通知 36 / 已发布 28 / 平均已读 84%；右上「发布通知」直达 `03-notice-create`。
3. 筛选：状态 5 段（全部 / 已发布 / 定时 / 草稿 / 已撤回）+ 类型 6 个胶囊（全部 / 作业 / 带物 / 活动 / 健康 / 其他）。
4. 列表：每条显示类型、状态、时间、标题、两行摘要、班级、附件数、已读进度；已发布显示「已读 25/30 · 83%」，草稿显示「仅自己可见」。
5. 详情弹层：完整正文、已读/未读/已确认、进度条、查看未读家长、再次提醒；草稿/定时改为「继续编辑 / 编辑定时」。

## 覆盖状态

- **正常**：5 条示例（已发布 2 / 定时 1 / 草稿 1 / 已撤回 1），筛选与详情可点。
- **空态**：整页无通知 → 「还没有发布过通知」+「去发布通知」。
- **筛选空**：某状态/类型下无结果 → 「这个筛选下没有通知」+「清除筛选」。
- **加载**：3 张骨架卡。
- **失败**：一句话 +「重试」；重试先回到加载态再恢复。
- **禁用/只读**：定时与草稿的已读统计显示「发送后开始统计」；已撤回标注「家长已不可见」。

## Token 映射

- 背景：`var(--bg)`、`var(--surface)`、`var(--surface-warm)`
- 品牌：`var(--accent-gradient)`、`var(--accent)`、`var(--accent-deep)`、`var(--accent-wash)`、`var(--on-brand)`、`var(--on-accent)`
- 语义：`var(--success)` / `--success-wash`、`var(--warning)` / `--warning-wash`、`var(--danger)` / `--danger-wash`、`var(--info)` / `--info-wash`
- 文字：`var(--fg)`、`var(--fg-2)`、`var(--muted)`
- 结构：`var(--radius-card)`、`var(--radius-pill)`、`var(--shadow-sm/md/lg)`、`var(--space-*)`、`var(--text-*)`

## 与现状的差异

- 这是**补设计的页**：原源码没有对应 `.vue`，`pages.json` 也没有注册；2026-10-27 已补源码与注册，本设计作为实现参考。
- 源码只有「发布通知」页；本页补上发布之后的列表、状态筛选、已读率与快速查看，接口已支持这些字段（`status` / `category` / `read_rate` / `read_count` / `total_count` / `attachments_count`）。
- 没有画独立的「通知详情」页：详情以列表内底部弹层承载；源码实现同样使用弹层，完整已读名单分页仍可后续补。
- 「再次提醒」与「查看未读家长」对应契约里的 `POST /teacher/notices/:id/remind`、`GET /teacher/notices/:id/read-status`；原型只做 toast / 名单展开，源码实现已接真实接口。

## 遗留

1. **源码已补**：`teacher/src/pages/notice/index.vue` 已新建并在 `pages.json` 注册；分享路径已通。源码实现与设计的差异：底栏图标用文字（小程序不支持内联 SVG）、详情用弹层、草稿暂不支持编辑。
2. **完整详情页**：若产品需要独立通知详情（含附件预览、完整已读名单分页），需要再补一页；当前弹层是列表内的快速查看。
3. **筛选与分页**：源码已接 `status` / `category` 服务端筛选；分页仍是 20 条「加载更多」，未做总数分页器。
4. **多人协作**：列表目前按「当前老师创建的通知」呈现（消息接口口径）；若园长/年级组长要查看全园通知，需要额外权限与接口参数。
5. **演示数据**：全部通知 36 / 已发布 28 / 平均已读 84% 为示例值；源码以接口返回为准。
6. **草稿编辑未实现**：源码列表/详情当前只能查看草稿；设计稿的「继续编辑」需要 `notice/create` 支持 `id` 预填 + `updateTeacherNotice`。

## 验证

- `od-check --width 390`：`problems: []`（`scrollWidth` 390 / `clientWidth` 390，无越界、无 console 报错）。
- `od-shot --width 390`：`preview.png` 780×3160；底部演示条与内容无遮挡。
- DOM 实测：5 条默认列表；状态「草稿」→ 1 条；状态「定时」+ 类型「作业」→ 筛选空态；「清除筛选」→ 恢复 5 条；详情弹层可开/关；草稿显示「继续编辑」且隐藏已读统计；定时显示「编辑定时」；已发布展开 5 位未读家长；「再次提醒」toast 正常；空态切换后列表隐藏、空态显示；失败态「重试」→ 加载 → 正常；无 console error。
- 未做真实接口联调：本页是原型，`POST /teacher/notices/:id/remind` 与 `GET /teacher/notices/:id/read-status` 仅以 toast / 名单展开表现。
