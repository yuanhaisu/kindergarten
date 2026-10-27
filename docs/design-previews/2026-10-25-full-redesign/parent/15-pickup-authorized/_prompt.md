你是幼儿园管理平台「养成星球」的资深移动端设计师。为家长端微信小程序做「授权接送人」列表页的高保真 HTML 原型。

## 唯一任务
页面：授权接送人（源码：`parent/src/pages/pickup/authorized.vue`）
只做一个方向，不 A/B，不多页面。写完 index.html / styles.css / app.js 立刻停止：不读回产物，不自查循环，不提问，不联网。

## 这个页面真实的产品事实（照此设计，不要发明）

家长从「接送」页 `navigateTo` 进入的子页，`onShow` 重拉。核心任务：管理「谁能接走孩子」，并把其中一位设为「今日接走人」（指定后才能获取接送二维码）。

自上而下 5 个区块：

1. **安全提示卡**：「安全提示」+「授权接送人信息仅供门卫核验身份，请确保信息真实有效。」
2. **工具行**：左「5 位授权接送人」，右「＋ 添加」（去 `/pages/pickup/authorized-create`）。
3. **今日接走人横幅**：已指定 →「今日接走人：王建国」；未指定 →「今日尚未指定接走人，指定后才能获取接送二维码」。
4. **人员卡片**（每张）：姓名首字头像；姓名；授权类型「长期授权 / 临时授权」；已指定者加「今日接走人」；「关系 · 脱敏手机号」（如「爷爷 · 138****6621」）；权限标签「可接送 / 看记录 / 看成长」（按 `allow_pickup` / `allow_view_record` / `allow_growth_view` 为真才显示）；操作「设为今日接走人」+「删除」。
5. **页脚说明**：「新添加的接送人需园所审核通过，才能指定为今日接走人。」

真实字段：`name`、`relation`、`phone`（后端已脱敏）、`auth_type_label`、`allow_pickup`、`allow_view_record`、`allow_growth_view`、`status`（1 生效 / 0 已撤销）、`review_status`（0 待审核 / 1 已通过 / 2 已拒绝）及 `review_status_label`。

规则：只有 `review_status===1 && status===1` 的人可被指定；其余按钮**视觉降级但可点**，点按 toast「该接送人尚未通过园所审核」或「该接送人授权已撤销」。审核状态必须可见：待审核 / 已拒绝徽标；已通过不加徽标，避免绿色泛滥。

二次确认（源码原话）：
- 指定且当前无人：「确定指定 李秀兰 为今日接走人吗？」
- 指定且已有今日接走人：「换为 李秀兰 后，今日已生成的接送二维码将全部失效，需要重新获取。确定继续？」
- 删除：微信风格居中对话框，标题「确认删除」，正文「确定删除 王建国 吗？」，按钮「取消 / 删除」。

**源码没有、不要画**：搜索、编辑入口、分享、批量操作、接送记录、底部 tabBar。

## 示例数据（中文，幼儿园语境）
- 王建国 · 爷爷 · 138****6621 · 长期授权 · 已通过 · 生效 · 可接送 / 看记录 —— **今日接走人**
- 李秀兰 · 奶奶 · 139****3382 · 长期授权 · 已通过 · 生效 · 可接送 / 看记录 / 看成长
- 陈梅 · 阿姨 · 137****9045 · 临时授权 · 已通过 · 生效 · 可接送
- 张建军 · 叔叔 · 136****1187 · 长期授权 · **待审核** · 可接送（不可指定）
- 刘芳 · 外婆 · 135****5508 · 长期授权 · 已通过 · **已撤销**（不可指定）

## 视觉方向：「可信名单」
整页安静中性底；可信赖感来自**清晰的分层**而不是装饰。橙色只落在添加按钮、今日接走人徽标、被指定卡片的 1px 描边和主 CTA 上，面积 ≤ 10%。

- `<html lang="zh-CN" data-surface="parent">`，390px 视口，单列纵向流。
- **外壳原样用共享 shell**：读 `D:/yuanhaisu/personal_projects/kindergarten/docs/design-previews/2026-10-25-full-redesign/_shared/shell.md` 与 `shell.css`，照抄 `mp-statusbar` / `mp-time` / `mp-status-icons` / `mp-navbar` / `mp-back` / `mp-title` / `mp-capsule` / `mp-main` / `mp-demo` 的标记与 SVG。
- 引入顺序：`tokens.css` → `../../_shared/shell.css` → `styles.css`；末尾先 `../../_shared/shell.js` 再 `app.js`。
- **子页：导航栏左侧必须有返回箭头**，标题「授权接送人」；**不画 tabBar**；演示条用默认位置。
- 禁止复制 shell.css、禁止改写状态栏 SVG；**styles.css 里不得出现任何 `.mp-*` 选择器**（唯一例外 `[data-state]`）。

### 结构细节
- 页面左右内边距 `var(--page-pad-x)`，顶部 12px，底部 `calc(28px + env(safe-area-inset-bottom))`；卡片间距 12px。
- **安全提示卡**：`var(--surface)` + 1px `var(--border)` + `var(--radius-card)`，内边距 12px 14px；左侧 30px 圆 `var(--warning-wash)` 内 16px `var(--warning)` 盾牌图标；标题 13px/600 `var(--fg)`，正文 12px `var(--fg-2)` 行高 1.55。
- **工具行**：左 13px/500 `var(--muted)`「5 位授权接送人」；右侧「＋ 添加」为 44px 高、`var(--radius-pill)`、`var(--accent-gradient)` 底、`var(--on-brand)` 字 13px/600、左右内边距 16px、`var(--shadow-sm)`。
- **横幅**：已指定用 `var(--success-wash)` + 18px `var(--success)` 对勾；上行 11px/500 `var(--muted)`「今日接走人」，下行 14px/600 `var(--fg)` 姓名。未指定用 `var(--warning-wash)` + 警示图标，主句 13px/600 `var(--fg-2)`，次句 12px `var(--muted)`。内边距 12px 14px，`var(--radius-card)`。
- **人员卡片**：`var(--surface)` + `var(--radius-card)` + `var(--shadow-sm)`，内边距 14px 16px；被指定卡片改 `box-shadow: inset 0 0 0 1px var(--accent), var(--shadow-md)`。
  - 头行：40px 圆形首字头像（普通 `var(--bg)` 底 / `var(--fg-2)` 字；被指定 `var(--accent-wash)` 底 / `var(--accent-deep)` 字，15px/600）；姓名 15px/600 `var(--fg)`；姓名右侧徽标可换行。
  - 徽标：授权类型 `var(--bg)`+`var(--fg-2)`；待审核 `var(--warning-wash)`+`var(--warning)`；已拒绝 `var(--danger-wash)`+`var(--danger)`；已撤销 `var(--bg)`+`var(--muted)`；今日接走人 `var(--accent)` 底 + `var(--on-brand)`。统一 11px/500、高 20px、内边距 0 8px、`var(--radius-pill)`。
  - 元信息：「爷爷 · 138****6621」12px `var(--muted)`，允许换行。
  - 权限标签：11px，高 20px；「可接送」用 `var(--success-wash)`+`var(--success)`，「看记录」用 `var(--info-wash)`+`var(--info)`，「看成长」用 `var(--bg)`+`var(--fg-2)`（tokens 无紫色）。
  - 操作行与上方用 1px `var(--border)` 分隔，右对齐：「设为今日接走人」为 44px 高描边胶囊（1px `var(--accent)`、`var(--accent-deep)` 13px/500、透明底）；不可指定时改 `var(--bg)` 底、`var(--hint)` 字、无描边。「删除」为 44px 高纯文字按钮，`var(--muted)` 13px/500 + 15px 垃圾桶线性图标，点开确认框时确认键才用 `var(--danger)`。
- **页脚说明**：12px `var(--muted)` + 13px 信息图标，左对齐，上边距 4px。
- 交互反馈只用 `:active`（背景压到 `var(--bg)` 或 `scale(.98)`）。**禁止 `white-space: nowrap`**；所有容器 `max-width: 100%`；多列用 `repeat(N, minmax(0, 1fr))`；页面不得横向滚动。
- 颜色只写 `var(--*)`（遮罩用 `var(--sidebar)` 加透明度，或 `color-mix`），禁止裸 hex/rgb；无直角（禁止 `border-radius: 0`）；阴影只用 `var(--shadow-*)`；正文 ≥ 13px（徽标可 11px）；触控目标 ≥ 44px；图标一律内联 SVG。
- 不用 `<select>`、alert、prompt；不引用外部资源；无 console 报错。

## 状态覆盖（`data-state` 驱动，`mp-demo` 放 4 个芯片）
`data-mp-state`：normal 正常 / empty 空态 / loading 加载 / error 失败，默认正常。

- **正常**：工具行、横幅、5 张卡片、页脚说明完整（默认）。
- **空态**：隐藏工具行、横幅、列表、页脚；换 56px `var(--bg)` 圆 + `var(--hint)` 线框「添加联系人」图标 +「还没有授权接送人」+「添加家人后，才能指定今日接走人并获取接送二维码」+ 主按钮「添加接送人」。
- **加载**：横幅与列表换骨架（横幅一条、3 张卡：40px 圆 + 三条线 + 两个标签块），`var(--bg)` 呼吸动画 1.4s；工具行计数换短骨架条，添加按钮保留可用；页脚保留。
- **失败**：隐藏横幅、列表、页脚；换 `var(--danger-wash)` 圆 + `var(--danger)` 感叹号 +「授权接送人加载失败」+「网络不太稳定，请稍后再试」+ 主按钮「重试」；工具行保留。
- **禁用态**：待审核 / 已撤销人员的「设为今日接走人」降级（见上），`aria-disabled="true"`，点按 toast 解释原因；删除按钮始终可用。

## 交互（app.js，全部真实可点）
- 「＋ 添加」与空态「添加接送人」：toast「已打开添加授权接送人」。
- 「设为今日接走人」：打开居中确认框，按是否已有指定切换两种文案；点「确定」后更新横幅姓名、把该卡改为被指定样式、移除原被指定者的徽标、toast「今日接走人：李秀兰」；点「取消」关闭。
- 「删除」：打开居中对话框（标题「确认删除」，正文带姓名）；「删除」后卡片从列表移除并 toast「已删除」；若删的是今日接走人，横幅切为未指定态；若无剩余卡片，自动进入空态并同步芯片。取消则关闭。
- 「重试」：按钮进禁用态「重试中…」，1 秒后回正常并 toast「已重新加载」。
- 返回箭头：toast「已返回接送」。点遮罩关闭对话框。
- 弹层打开时切换 `aria-hidden`；Esc 关闭。toast 自绘（`var(--fg)` 底、`var(--surface)` 字、胶囊、`z-index` 高于演示条），2 秒消失。
- 芯片切换由 `_shared/shell.js` 负责，app.js 监听 `mp:state`：进别态时关闭弹层；回正常态时恢复列表 DOM。

## 输出文件（项目根目录）
1. `index.html` 2. `styles.css` 3. `app.js`，每个 < 20KB；`tokens.css` 已由流程提供，不要重写或覆盖。
