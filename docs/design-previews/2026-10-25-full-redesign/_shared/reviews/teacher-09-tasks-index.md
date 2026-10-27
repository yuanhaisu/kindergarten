# teacher/09-tasks-index · 对抗性单页审查

- verdict：**WARN**（score 64/100）
- 源码覆盖：列表正常/空/加载/错误四态与创建表单主体齐全，但「范围」退化为 6 个硬编码姓名且没有学生名单加载失败分支，星星罐未配置/读取失败两态不可达或缺失，加载失败时「保留旧列表 + 错误条」的源码分支未画（已参与 N/30 的 N+1 依赖见 A4，未重复计）。

## P1 · 明显缺陷（2）

### 1. [completeness] 「范围」只剩 7 个硬编码 pill（班级 + 6 个孩子），源码是从接口全量渲染的本班学生（本页自设 CLASS_SIZE=30），且没有「学生名单加载失败，点击重试；恢复前只能创建班级任务」分支。按真实班级 30 人算，多数孩子无法被指派个体任务。

- 证据：index.html:197-204 仅 班级/乐乐/小雨/朵朵/果果/豆豆/米米；app.js:55 CLASS_SIZE=30；teacher/src/pages/tasks/index.vue:32、211-220、309-312；backend/api/api-teacher/09-class-and-tasks.md:68「一次全量返回（无分页）」
- 建议：改用底部 picker/action sheet 承载完整学生列表（可滚动或带搜索），「班级」固定为首项；补学生名单加载失败的提示与重试；notes 记录该差异。

### 2. [notes-accuracy] notes 声称覆盖「星星罐未配置时『按班级』置灰」，但该状态不可达也不可见：jar.goal 初始为 60 且保存只接受 >0；.jar-empty-note 恒为 display:none（无任何显示规则），.is-unset 无对应 CSS，保存按钮永远写「更新目标」而非源码按 configured 切换的「设置目标」。

- 证据：notes.md:17；app.js:58、208、456-465；index.html:158、161；styles.css:1（`.jar-empty-note{display:none…}` 且无 `.jar-card.is-unset` 规则）；teacher/src/pages/tasks/index.vue:348-359
- 建议：演示条增加「未配置」态或提供切换；补 `.jar-card.is-unset .jar-live{display:none}` 与 `.jar-empty-note{display:block}`；按钮文案按 jarConfigured 切换；同步修正 notes。

## P2 · 打磨项（6）

### 1. [logic] 加载失败态被简化成「隐藏列表 + 网络异常，未获取到任务列表」，而源码是「保留旧列表 + 顶部错误卡（显示的是上次拉取的结果）」。更严重的是在错误态新建任务后，app.js 直接把 data-state 切回 normal，错误卡消失、旧列表被当成正常数据展示，期间没有重新拉取。

- 证据：index.html:134-139；styles.css:1 `[data-state="error"] .task-list{display:none}`；app.js:393；teacher/src/pages/tasks/index.vue:362-366；浏览器实测：错误态创建后 data-state=normal、error-state display:none、列表复现且「共 5 个任务」
- 建议：错误态保留已加载卡片并在列表上方插入可重试错误条；创建成功后只触发加载函数，不直接清除错误态。

### 2. [logic] 弹层打开只给 body 加 is-locked，页面与 shell 都没有对应 CSS，遮罩上滑动仍会滚动背景页，滚动锁定是空操作。

- 证据：app.js:287、297；全目录 grep 无 `body.is-locked` 样式规则；浏览器实测 `getComputedStyle(document.body).overflowY === "visible"`，弹层打开时 window.scrollY 从 200 变为 400
- 建议：补 `body.is-locked{overflow:hidden}`（或锁 .mp-app 并保留滚动位置），关闭时恢复。

### 3. [logic] 取消/关闭按钮、点遮罩、Esc 全部走 closeCompose → resetCompose，已填的标题/说明/日期/范围/公开开关被清空且无确认，再次打开是空表单。

- 证据：app.js:293-303、258-280；浏览器实测：输入标题后关闭再打开，`#taskTitle.value` 为空（草稿丢失）
- 建议：仅在创建成功后 reset；取消/遮罩关闭保留草稿，或对丢弃做二次确认。

### 4. [design-core] 筛选条写了 position:sticky;top:0，但 .mp-main 的 overflow-x:hidden 使其成为滚动容器（computed overflow-y:auto，scrollHeight=clientHeight），sticky 从不生效，页面一滚动筛选条就随内容滚走；即便修好滚动容器，top:0 也会被 20px 状态栏 + 44px 导航栏遮挡。

- 证据：styles.css:1 `.filters{position:sticky;top:0;z-index:8}`；_shared/shell.css:130-135 `.mp-main{overflow-x:hidden}`；浏览器实测 scrollY=0/80/120 时 .filters top=74/−6/−46（从未停在 0）
- 建议：让筛选条真正吸附（不要落在 overflow 容器内，或让 .mp-main 内部滚动），并把 top 设为 64px 以避开吸顶外壳。

### 5. [redundancy] 空态/错误态只隐藏筛选计数，筛选条本身仍显示且可点，但列表被隐藏，点「进行中/已结束」没有任何可见结果；空态首屏还留着三个没有对象的筛选项。

- 证据：styles.css:1 仅隐藏 `[data-state="empty"|error"] .filter-count`；index.html:35-47；浏览器实测：empty 下 .filters display=block、三个 count display=none；error 下 chips cursor=pointer 而 #taskList display=none
- 建议：空/错误态整体隐藏 `.filters`（或降级为只读摘要），与 list-head 同步显隐。

### 6. [design-core] 固定底栏与底部弹层宽度写死 430px，超出 shell 的 390px 手机框：视口宽于 390 时底栏与遮罩左右各多出 20px，遮罩还会把手机框外区域压暗。

- 证据：styles.css:1 `.action-bar{max-width:430px}`、`.sheet-layer{max-width:430px}`；_shared/shell.css:9 `.mp-app{max-width:390px}`；浏览器实测 500px 视口：mp-app 47.5–437.5（390px），action-bar 与 sheet-layer 27.5–457.5（430px）
- 建议：两处改为 max-width:390px（与 mp-app 一致），或改为 100% 并约束在 .mp-app 内。

## 2026-10-27 P1 修复记录

- P1-1 [completeness] 已修：把 7 个硬编码 pill 换成弹层内可滚动选择器（「班级」固定首项 + 全班 30 人演示名单，选项 44px 高、最多 236px 滚动），并补源码的学生名单失败分支——范围只剩「班级」+「班级学生加载失败，点击重试；恢复前只能创建班级任务」，演示条「名单失败」可切换、重试 600ms 恢复；notes 同步记录该差异与演示数据口径。
- P1-2 [notes-accuracy] 已修：演示条新增「星星罐未设」切换；未配置时隐藏进度区与百分比、显示 `.jar-empty-note`、保存按钮切为「设置目标」，`.jar-card.is-unset` 规则补齐；保存目标后恢复进度并把按钮切回「更新目标」；notes 覆盖状态同步修正。
- 说明：P1-2 未按建议整块隐藏 `.jar-live`——目标表单就在该容器内，整块隐藏会让「未配置」态无法设置目标（死路）。改为未配置时只隐藏 `.jar-stats` 与 `.jar-note`，保留目标表单。
- 验证：`od-check --width 390` problems `[]`；浏览器实测选择器 31 项、Esc 先收列表再关弹层、名单失败/重试恢复、未配置/保存目标互切、创建个体任务均通过，控制台无 error/warning。
