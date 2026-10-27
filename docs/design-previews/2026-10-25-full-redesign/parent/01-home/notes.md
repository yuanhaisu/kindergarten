# 家长首页（parent）

- 源码：parent/src/pages/index/index.vue（tabBar 首页；`onLoad` 并行拉 `HomeOverview` + `ListHomeFeeds`）
- 设计系统：user:nurture-planet-local，surface=parent
- 生成方式：OpenDesign（首跑仅产出 index.html，缺 styles.css / app.js；第二次 run 缩短 brief 后四件套齐全，未走手写兜底）
- 方向：**「清晨第一眼」——孩子 → 今天 → 该做什么**。整页冷雾白底，唯一的暖色块是孩子卡这一张紧凑 hero，暖色面积约 3–5%，把视觉重量压在「今日在校情况」这张白卡上。
- 信息层级：
  1. 孩子卡：52px 头像（姓名首字占位）、姓名、班级胶囊、问候语、44px 设置按钮
  2. 今日在校情况：日期 + 4 列状态（出勤/进餐/午睡/情绪）+ 表现条
  3. 快捷入口 4 格一行：通知公告（1）/ 接送管理 / 亲子任务（3）/ 我要请假
  4. 今晚一件事（有未提交任务时）
  5. 最新动态：通知与成长混合卡片流（含未读红点、类型胶囊、相对时间）
  6. 底部 tabBar 4 项，首页选中

## 覆盖状态

| 状态 | 表现 |
| --- | --- |
| 正常 | 示例数据完整渲染（乐乐 / 小一班 / 妈妈 您好~ / 已入园 07:52 / 很好 / 120分钟 / 小小艺术家） |
| 空态 | 只把动态列表换成空态块（线性枝叶图标 + 「这周还没有新的通知和成长记录」+ 主按钮「看看成长时光」）；「今晚一件事」按今日任务保留，通知/任务角标隐藏；孩子卡、今日卡、快捷入口保留 |
| 加载 | 孩子卡保留姓名/班级/问候（缓存态）；今日卡 4 列换灰色骨架圆 + 骨架短条；动态区换 2 条骨架卡（呼吸动画，`prefers-reduced-motion` 关闭）；今晚一件事隐藏 |
| 错误 | 今日卡、今晚一件事、动态区整体换成错误卡（`--danger-wash` 图标 + 「首页数据加载失败」+ 重试）；孩子卡与快捷入口保留 |
| 禁用 | ① 当前孩子的胶囊 `aria-current="true"`，点击直接 return、无按压反馈；② 错误态点「重试」后按钮立刻 `disabled`、文案「重试中…」、降到 `--border`/`--hint` 视觉，约 1s 后恢复并 toast「已重新加载」 |

## token 映射

| 元素 | Token |
| --- | --- |
| 页面底 / 卡片底 | `--bg` / `--surface` |
| 孩子卡 | `--accent-gradient-soft`、`--radius-2xl`、`--shadow-sm`、`--fg`、`--fg-2` |
| 班级胶囊 / 设置按钮 | `--surface` 底 + `--accent-deep` 字；按钮 44px = `--input-height` |
| 今日卡 | `--radius-card`、`--shadow-md`、`--text-xl` 标题 |
| 状态列圆底 | `--success-wash`+`--success`（出勤）、`--info-wash`+`--info`（进餐）、`--warning-wash`+`--warning`（午睡）、`--accent-wash`+`--accent-deep`（情绪），均 38px `--radius-pill` |
| 表现条 | `--accent-wash` 底 + `--accent-deep` 星标 + `--text-sm` |
| 快捷入口 | 46px `--accent-wash` 圆角方（`--radius-xl`）+ `--accent-deep` 线性图标；角标 `--danger` 底 `--on-brand` 字 |
| 动态类型胶囊 | 通知 `--info-wash`/`--info`，成长 `--success-wash`/`--success`；未读点 `--danger` |
| 辅助文字 | `--muted`（时间、副行、区块标题），未用 `--hint`（对比度不足） |
| 骨架 / 分隔 | `--border` |
| 主按钮 | `--accent-gradient` + `--on-brand` + `--radius-pill` + `--shadow-md` |
| 动效 | `--motion-fast` + `--ease`；`--radius-pill`/`--radius-xl`/`--radius-lg`/`--radius-sm` |

## 与现状的差异

- **去掉半屏橙色渐变 header**：现状整块渐变占过半屏，改为冷雾白底 + 紧凑 hero，暖色只留在孩子卡与 wash 圆点上。
- **首屏密度**：孩子卡 + 今日卡 + 快捷入口三块压进第一屏（390×844 可见），中间不插大标题与大留白。
- **4 张 PNG 插画换成线性 SVG**：入口图标 22px `currentColor`，不再依赖位图资源。
- **状态信息结构化**：出勤/进餐/午睡/情绪从平铺文字改为 4 列等宽 grid（`repeat(4, minmax(0,1fr))`），圆底 wash 色区分语义。
- **最新动态从列表改为卡片流**：类型胶囊 + 未读红点 + 两行预览 + 右对齐相对时间（1 小时内 N 分钟前 / 24 小时内 N 小时前 / 更早日期，与源码 `formatTime` 一致）。
- **多孩切换**：源码 `handleSwitchStudent` 提为一排孩子胶囊，当前孩子高亮；切换同时刷新孩子卡、4 个状态值，以及「今晚一件事」文案与亲子任务角标（乐乐、小雨各有独立的任务数据，切换后不会残留上一个孩子的任务）。

## 遗留

1. **`students[].relation` 从未下发**：`HomeOverview` 只返回 `current_student{id,name,avatar,class_name,class_id}`，`students[]` 里没有 `relation`，所以源码 `relationLabel` 恒走兜底「家长」。原型按设计意图画「妈妈 您好~」，需后端补字段或产品改文案（属契约缺口，已并入 `_shared/findings.md` 的口径）。
2. **契约文档漂移**：`backend/api/api-parent/01-home.md` 把动态流写成 `growth_log`，实际 `ListHomeFeeds` 返回 `growth`。以代码为准，文档需回写。
3. **`needBindPhone` 分支未画**：未绑定手机号时整页换绑定引导卡，按本页约定不画，留给产品单独定页面。
4. **空态 CTA「看看成长时光」是新增引导**：源码空列表没有该按钮，属设计补充，需产品确认是否落实现（当前指向成长时光 tab）。
5. **未做**：下拉刷新控件（`enablePullDownRefresh` 为系统行为，原型不画）、消息中心、积分、成长报告入口（首页源码均无）。

## 验收

- `od-check`：`problems: []`，390/390 无横向溢出，吸顶验证 `scrollY=296 / statusTop=0 / navTop=20`。
- `od-shot`：390×1196，preview.png 237KB。
- `od-audit`：本页仅「缺 notes.md」一项，本文补齐；无其它 ✗（无裸色值、无外部资源、外壳未复制、`styles.css` 无 `.mp-*` 定义）。
- mcp-vision 逐项核对：状态栏 + 导航栏「养成星球」+ 胶囊齐全，无返回箭头；tabBar 4 项且首页高亮；首屏三块齐全；4 入口同排未换行；无文字裁切、无直角、暖色 3–5%。
