# 通知公告（parent）

- 源码：parent/src/pages/notice/index.vue（接口 `GET /parent/notices`，契约见 backend/api/api-parent/02-notice.md）
- 设计系统：user:nurture-planet-local，surface=parent
- 方向：**安静的收件箱** —— 三个数字先说明「有多少、几条没看、今天几条」，类型只负责筛选，颜色只负责标记未读，不做任何整行染色。
- 信息层级（从上到下）：
  1. 状态栏 + 导航栏「通知公告」（`navigateTo` 子页，带返回箭头，无 tabBar）
  2. 统计条：全部通知 8 / 未读 1 / 今日新增 1
  3. 类型分段：全部 · 任务 · 活动 · 通知（源码 4 个 tab，选中填充品牌色）
  4. 通知卡片流：类型图标 → 标题（未读跟品牌色圆点）→ 两行预览 → 「类型 · 发布人 · 时间」→ 右侧 `›`
- 覆盖状态：
  - **正常**：8 条数据；只有「秋季亲子运动会报名通知」未读（与家长首页原型的「通知公告 1」角标对齐），未读项图标位用 `--accent-wash`/`--accent-deep`，其余中性。
  - **空**：统计 0/0/0；46vh 白卡内居中「暂无通知」+ 一句说明 +「刷新」主按钮（对应源码 `onPullDownRefresh`）。
  - **加载**：统计数字换骨架条；3 张骨架卡呼吸动画；筛选胶囊降透明度并禁用。
  - **失败**：统计显示「—」且转灰；46vh 错误卡（`--danger-wash` 图标位 + 「通知加载失败」+「重试」）；筛选同样禁用。点重试 →「重试中…」禁用 1 秒 → 自动回到正常态并 toast。
  - **禁用**：加载/失败态的筛选胶囊（`opacity:.5` + 原生 `disabled`）、失败态重试按钮（`--hint`/`--bg` 降级）。
- token 映射：页面底 `--bg`；卡片/统计/空态 `--surface` + `--radius-card` + `--shadow-sm`；选中胶囊/未读数字/未读圆点/错误态外主按钮 `--accent`（按钮用 `--accent-gradient`）；未读图标位 `--accent-wash` + `--accent-deep`；正文 `--fg`、次要 `--muted`、箭头 `--hint`；分隔线/描边 `--border`；错误图标位 `--danger-wash` + `--danger`；字号 `--text-xs/sm/md/3xl`，间距 `--space-md` + `--page-pad-x`。
- 与现状的差异：
  - 未读标记：源码是整卡 `border-left` + 红色圆点；原型按批次要求改为**品牌色小圆点**（紧贴标题末尾，不整行染色、不描边）。
  - 加载态：源码「加载中...」文字 → 骨架屏；错误态：源码只有 toast → 错误卡 + 重试。
  - 空态补了「刷新」主按钮（源码只有下拉刷新手势，原型无法手势，用按钮对应同一 handler）。
  - 列表新增类型标签与类型图标（列表接口返回 `category` / `category_label`，源码 tab 已按 category 分组，只是列表未渲染标签）；类型标签是展示层中文映射：task 任务 / activity 活动 / homework 作业 / bring 带物 / health 健康 / other 其他。
  - 统计数字在切换分类时按当前可见列表重算（源码是接口 `total` + 当前列表统计的组合），原型保持一致语义。
  - 点击即置已读并提示「已读 · 打开通知详情」：源码实际是详情页 `onLoad` 调 `markNoticeRead`，列表本身不置读；原型把「进详情→置读」合并成一次点击以便演示。
- 遗留 / 需产品确认：
  - **置顶**：列表接口无置顶字段（置顶只在管理端公告），本页未画，不视为缺陷。
  - **需确认 / 已确认 / 附件数**：接口返回 `need_confirm`、`is_confirmed`、`attachments_count`，但源码列表不展示；原型未加，避免发明优先级。
  - 「通知」tab 的过滤是前端按 homework+bring+health+other 过滤，`total` 仍是接口返回的全量 total；原型按可见列表重算，真实接口上线后需确认统计口径是否要改成后端返回。
  - 家长端 `category_label` 在 `parent_portal_service.go` 里直接回原始英文 key，未本地化；如果后续列表要显示中文类型，需要后端补映射。
