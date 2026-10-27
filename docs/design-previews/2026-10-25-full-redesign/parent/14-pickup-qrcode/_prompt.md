你是幼儿园管理平台「养成星球」的资深移动端设计师。为家长端微信小程序做**接送二维码**页的高保真 HTML 原型。

## 唯一任务
页面：接送二维码（源码：parent/src/pages/pickup/qrcode.vue）
只做一个方向，不要 A/B 变体，不要多页面。写完 index.html / styles.css / app.js 后立刻停止：不读回产物，不自查循环，不提问，不联网。

## 这个页面真实的产品事实（照此设计，不要发明）

家长从「接送记录」页右上角「接送二维码」按钮 `uni.navigateTo` 进入的**子页**。核心是一张「当日接走凭证」：到园接孩子时把码出示给老师扫，核验后一次有效，当日过期。

自上而下 3 个区块：

1. **凭证卡**
   - 卡内标题：「{孩子姓名} 的接送二维码」
   - **人证对照行**：孩子头像 + 姓名 + 小标签「孩子」；中间指向箭头；今日接走人头像 + 姓名 + 关系标签（如「奶奶」）。未指定时姓名「待指定」、标签「接走人」。
   - **二维码**：源码用 canvas 本地绘制 240px 码，白底深色模块。
   - 两条短说明：「请向老师出示此码，核验后一次有效」；「有效期至 10月26日 23:59」（源码把 `expire_at` 截断为「10月26日 23:59」）。
2. **两个按钮**：「刷新二维码」（主，重新请求凭证）、「更换今日接走人」（次，跳授权接送人页 `/pages/pickup/authorized`）。
3. **业务前提**：必须先「指定今日接走人」才能拿码。未指定时后端返回可读错误，页面展示错误与「去指定今日接走人」按钮（同一跳转）。授权失效时文案「所选接走人授权已失效，请重新指定」。

契约 `GET /parent/pickup/qrcode` 真实字段：`qr_code`、`student{ name, photo }`、`person{ name, relation, photo }`、`expire_at`。

**源码没有、不要画**：倒计时秒数、下载/保存图片、分享、扫码记录、历史凭证、多孩子切换、底部 tabBar（子页）。

## 示例数据（中文，幼儿园语境；今天 = 2026年10月26日 星期一）
- 孩子：**乐乐**（小一班）；今日接走人：**奶奶**
- 有效期至：**10月26日 23:59**
- 二维码：装饰性占位图形，不可扫描

## 视觉与结构硬约束

- `<html lang="zh-CN" data-surface="parent">`，390px 手机视口，单列纵向流。
- **外壳原样用共享 shell**：读 `D:/yuanhaisu/personal_projects/kindergarten/docs/design-previews/2026-10-25-full-redesign/_shared/shell.md` 与 `shell.css`，照抄 `mp-statusbar`/`mp-time`/`mp-status-icons`/`mp-navbar`/`mp-back`/`mp-title`/`mp-capsule`/`mp-main`/`mp-demo` 的标记与 SVG。
- 引入顺序：`tokens.css` → `../../_shared/shell.css` → `styles.css`；末尾先 `<script src="../../_shared/shell.js"></script>` 再 `<script src="app.js"></script>`。
- **禁止复制 shell.css、禁止改写状态栏 SVG**；只写 `<link href="../../_shared/shell.css">`，文件在当前项目里不存在也没关系。
- **styles.css 里不许出现任何 `.mp-*` 选择器**（唯一例外是 `[data-state]` 选择器）。
- **子页：导航栏必须有返回箭头**；标题就是 `pages.json` 的「接送二维码」。
- **不画 tabBar**（家长端只有首页/成长/接送/我的四个页面画）。

### 设计方向：一张当日通行证

整页安静中性底，凭证卡像登机牌——上半人证对照、下半大方留白的二维码；暖橙只出现在状态胶囊、主按钮、步骤序号上，面积 ≤ 10%。

- 整页底 `var(--bg)`；卡片 `var(--surface)` + `var(--radius-card)` + `var(--shadow-md)`。
- **凭证卡**：左右外边距 14px，内边距 18px 16px 16px。
  - 卡顶一行：左 11px/500 `var(--muted)`「接送凭证」；右状态胶囊（有效 `var(--success-wash)`/`var(--success)`，已过期 `var(--danger-wash)`/`var(--danger)`，刷新中 `var(--info-wash)`/`var(--info)`），高 24px、内边距 0 10px、11px/600。
  - 人证对照行：`display: grid; grid-template-columns: minmax(0,1fr) 28px minmax(0,1fr)`。两侧纵向堆叠：48px 圆形头像位（原型用姓名首字「乐」「奶」，`var(--accent-wash)` 底 + `var(--accent-deep)` 字 15px/600）+ 14px/600 姓名 + 11px `var(--muted)` 关系标签（「孩子」/「奶奶」）。中间 28px 宽水平虚线 + 小箭头 SVG（`var(--hint)`）。
  - **票根缺口**：人证对照行下方 1px 虚线 `var(--border)` 分隔，在卡片左右边缘各画一个 14px `var(--bg)` 圆形缺口，垂直居中于分隔线，营造登机牌撕口。
  - **二维码区留白充足**：分隔线下方留 18px，放**居中 232px 方形面板**：`var(--surface)` 底、1px `var(--border)` 描边、`var(--radius-xl)` 圆角、内边距 16px；面板内 200×200 二维码占位。
  - **二维码占位（硬性）**：内联 SVG 画装饰性二维码——三个定位角（大方块套空心方块）+ 伪随机 10px 模块 + 中心 40px 白色圆底 logo（内嵌 `var(--accent)` 小星球图标）。**绝不要生成真实可扫二维码，不要引 qrcode 库，不要用在线二维码图片**；中心 logo 保证不可扫。加 `role="img"` 与 `aria-label="接送二维码（原型占位）"`。
  - 面板下方居中：13px `var(--fg-2)`「有效期至 10月26日 23:59」（前置 14px 时钟图标 `var(--muted)`）；下一行 12px `var(--muted)`「请向老师出示此码，核验后一次有效」。
- **操作区**：主按钮「刷新二维码」100% 宽、高 48px、`var(--radius-pill)`、`var(--accent-gradient)` 底、`var(--on-brand)` 字 15px/600；次按钮「更换今日接走人」100% 宽、高 48px、透明底、1px `var(--accent)` 描边、`var(--accent-deep)` 字 15px/500；两按钮间距 10px。
- **使用说明卡**：白卡 + `var(--radius-card)` + `var(--shadow-sm)`，内边距 14px 16px。标题 12px/500 `var(--muted)`「使用说明」；3 行，每行 `min-height:40px`，左侧 20px 圆形序号（`var(--accent-wash)`/`var(--accent-deep)` 11px/600），右侧 13px/1.6 `var(--fg-2)`，行间 1px `var(--border)`：
  1. 到园接孩子时，把本页二维码出示给老师。
  2. 老师扫码核验通过后，本凭证立即失效。
  3. 凭证当日 23:59 过期，过期或需换人时点「刷新二维码」。
- 页面底部 `padding-bottom: calc(28px + env(safe-area-inset-bottom))`；卡片间距 12px。
- **禁止 `white-space: nowrap`**（本页一律不写）；所有容器 `max-width:100%`；多列用 `minmax(0,1fr)`；页面不得横向滚动。
- 颜色只写 `var(--*)`，禁止裸 hex/rgb；无直角（禁止 `border-radius: 0`）；阴影只用 `var(--shadow-*)`；触控目标 ≥ 44px；正文 ≥ 13px；`:active` 必须有反馈。
- 不用 `<select>`、alert、prompt；无 console 报错；不引用外部资源；图标用内联 SVG。

## 状态覆盖（`data-state` 驱动，`mp-demo` 放 **5 个芯片**）

芯片 `data-mp-state`：**normal 正常 / expired 已过期 / empty 空态 / loading 加载 / error 失败**，默认正常。

- **正常**：凭证卡 + 两个按钮 + 使用说明完整渲染（默认）。
- **已过期 expired**：① 状态胶囊「已过期」；② 二维码模块 `opacity:.16`，面板中央浮 `var(--danger-wash)`/`var(--danger)` 胶囊「已过期 · 刷新后可用」（带小刷新图标）；③ 有效期行变 `var(--danger)`「已于 10月26日 23:59 过期」；④ 主按钮仍为「刷新二维码」，其下多一行 12px `var(--muted)`「刷新后即可继续使用」。
- **空态 empty**（源码「未指定今日接走人」业务错误）：隐藏凭证卡/操作区/使用说明，换状态卡：56px 圆底 `var(--bg)` + `var(--hint)` 线框图标 +「还没有接送凭证」+「请先指定今日接走人，指定后即可获取二维码」+ 主按钮「去指定今日接走人」。
- **加载 loading**：凭证卡换骨架（两个头像圆 + 两条线 + 232px 方形骨架 + 两行文字骨架，`var(--bg)` 呼吸动画）；两个按钮 `opacity:.5` + `pointer-events:none`；使用说明保留。
- **失败 error**：隐藏凭证卡与操作区，换状态卡：`var(--danger-wash)` 圆底 + `var(--danger)` 图标 +「凭证加载失败」+「网络不太稳定，请稍后再试」+ 主按钮「重试」；使用说明保留。
- **禁用态（刷新中）**：主按钮 `disabled`、文案「刷新中…」、左侧 14px 转圈 spinner、降到 `var(--muted)`；二维码面板盖一层 `var(--surface)` 半透明遮罩 + 居中 20px `var(--accent)` spinner；1.4 秒后恢复并 toast「二维码已刷新」。

## 交互（app.js，全部真实可点）

- 点「刷新二维码」：进入刷新中禁用态，1.4 秒后回正常态 + toast「二维码已刷新」；若当前过期态，刷新后回正常态（同步对应芯片 `aria-pressed`）。
- 点「更换今日接走人」/ 空态「去指定今日接走人」：toast「已打开授权接送人」（不真跳页）。
- 点「重试」：按钮禁用、文案「重试中…」，1 秒后恢复 + toast「已重新加载」+ 回正常态。
- 二维码占位由 app.js 用确定性伪随机生成（每次结果一致），渲染进面板内 `<svg>`；不依赖任何库。
- 演示芯片由 `_shared/shell.js` 写 `data-state` 并派发 `mp:state`；app.js 监听该事件：进 loading 时重置刷新按钮，进 normal 时清掉过期/刷新残留。
- toast 自绘（`var(--fg)` 底、`var(--on-brand)` 字、圆角胶囊、固定中下部、`z-index` 高于演示条），2 秒消失；不用 alert。
- 所有按钮有 `:active`；无 console 报错。

## 输出文件（项目根目录，每个 < 20KB）
1. index.html —— 结构
2. styles.css —— 样式（引用 tokens.css）
3. app.js —— 交互

tokens.css 已由流程提供，**不要重写，不要覆盖**。不要生成 preview.png 或 notes.md。
