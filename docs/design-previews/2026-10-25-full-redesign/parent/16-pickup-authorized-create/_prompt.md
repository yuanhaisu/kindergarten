你是幼儿园管理平台「养成星球」的资深移动端设计师。为**家长端**微信小程序做一个页面的高保真 HTML 原型。

## 唯一任务
页面：添加授权接送人（源码：`parent/src/pages/pickup/authorized-create.vue`）
只做一个方向，不要 A/B 变体，不要多页面。写完三个文件立刻停止：不读回产物、不自查循环、不提问、不联网。

## 这个页面真实的产品事实（照此设计，不要发明字段）

家长在「授权接送人」列表页点「＋ 添加」进入本页（`uni.navigateTo`，**是子页，导航栏必须有返回箭头**）。页面 = 一张登记表 + 底部固定提交栏。自上而下：

1. **孩子上下文（只读）**：`student_id` 取自本地「当前孩子」，本页**没有切换入口**，不要画下拉/切换器/班级选择。只用一行轻量上下文：`为 乐乐 添加接送人`。
2. **基本信息卡**（4 项，顺序固定）：
   - 姓名 `name`，**必填**
   - 手机号 `phone`，**必填**，11 位数字
   - 与孩子关系 `relation`，**必填**，默认「爸爸」；选项固定 9 个：爸爸 / 妈妈 / 爷爷 / 奶奶 / 外公 / 外婆 / 叔叔 / 阿姨 / 其他。源码是原生 picker，本原型改成**底部选择面板**。
   - 身份证号 `id_card_number`，**选填**，占位「选填，便于门卫核验」，最多 18 位。
3. **权限卡**（5 个开关，顺序与语义固定）：
   - 可接送孩子 `allow_pickup`（默认开）
   - 可查看接送记录 `allow_view_record`（默认关）
   - 可查看在园状态 `allow_view_status`（默认关）
   - 可查看成长记录 `allow_growth_view`（默认关）
   - 接送时通知我 `notify_on_pickup`（默认开）
4. **审核说明**（源码原文）：「提交后需园所审核通过，授权才会生效。」可补一句事实：「审核通过前，该接送人不能接走孩子。」
5. **底部固定提交栏**：按钮上方常驻一行说明 + 主按钮「提交」。

校验（源码）：姓名 / 手机号 / 关系任一为空即中止。**原型不要用事后 toast**：未填齐时按钮上方常驻阻断文案 + 字段就地错误 + 按钮置灰。关系有默认值，实际阻断通常只涉及姓名与手机号。
提交：`POST /parent/pickup/authorized-persons`；提交中按钮「提交中...」不可点；成功提示「已提交，待园所审核」并返回列表；失败**不 toast**，用失败条。

**源码里不存在、绝对不要画**：临时/长期授权类型选择、短信验证码、头像上传、有效期、停用/启用、审批人、备注、多个孩子切换、tabBar。

## 示例数据（中文，幼儿园语境）
- 孩子：乐乐 · 小一班
- 姓名：王建国
- 手机号：13812345678
- 关系：爷爷
- 身份证号：310101195503124012
- 权限：可接送开、查看接送记录关、查看在园状态关、查看成长记录关、接送时通知我开
- 填齐后的常驻行：`提交后需园所审核，通过后才可接送`

## 视觉方向：「核验登记表」
安静、可信、可核验：先「这个人是谁」，再「他能做什么」；橙色只落在开启的开关、关系值、主按钮与选中项，面积 ≤ 10%。

## 视觉与结构硬约束
- `<html lang="zh-CN" data-surface="parent">`，390px 手机视口，单列纵向流，不做桌面布局。
- **外壳必须原样使用共享 shell**：先读 `D:/yuanhaisu/personal_projects/kindergarten/docs/design-previews/2026-10-25-full-redesign/_shared/shell.md` 与 `shell.css`，照抄 `mp-statusbar` / `mp-time` / `mp-status-icons` / `mp-navbar` / `mp-back` / `mp-title` / `mp-capsule` / `mp-main` / `mp-demo` 的标记与 SVG，不许自造状态栏。
- 引入顺序：`tokens.css` → `../../_shared/shell.css` → `styles.css`；末尾先 `<script src="../../_shared/shell.js"></script>` 再 `<script src="app.js"></script>`。
- **禁止把 shell.css 复制进项目**，禁止改写状态栏 SVG；**styles.css 里绝对不许出现任何 `.mp-*` 选择器或覆盖**（唯一例外是 `[data-state]` 状态选择器，可写 `.mp-app[data-state="failed"] .xxx`）。
- 子页导航栏标题「添加授权接送人」，左侧**保留返回箭头**；家长端本页**不画 tabBar**。
- 页面内边距 `var(--page-pad-x)`，顶部 12px，底部 `calc(120px + env(safe-area-inset-bottom))`，末卡不被固定栏遮住。

### 结构细节
- **孩子上下文**：28px 圆形 `--accent-wash` 底 + `--accent-deep` 字「乐」13px/600；右侧 13px `--fg-2`「为 乐乐 添加接送人」；整行高 40px，无边框、无箭头。
- **分组标题在卡片外**：13px/500 `--muted`；右侧 11px `--muted`「3 项必填」。必填字段标签后加一个 `--danger` 的 `*`；选填字段标签右侧 11px `--muted`「选填」。
- **卡片**：`--surface` + `--radius-card` + `--shadow-sm`，内边距 14px 16px；字段间距 12px。
- **输入字段（堆叠式）**：标签 13px/500 `--fg-2` 在上；输入框 44px 高、`--bg` 底、`--radius-xl`、13px `--fg`、占位 `--hint`；聚焦 `box-shadow: inset 0 0 0 1px var(--accent)`；错误 `inset 0 0 0 1px var(--danger)` + 下方 12px `--danger` 说明。
- **关系行**：整行可点，52px 高，`--bg` 底 + `--radius-xl`，左侧值 13px `--fg`（或未选时的 `--hint`），右侧 12px `--hint` 的 `›`。
- **开关**：51×31 轨道、27px 圆钮（`--surface` + `--shadow-sm`），开 `--accent` / 关 `--border`；整行高 ≥ 52px 可点；主标签 13px/500 `--fg`，副文案 11px `--muted` 单行，只解释既有能力，如「允许门卫核验身份后接走孩子」「查看孩子每次被接走的时间」「孩子被接走时通知本账号」。
- **审核说明条**：`--warning-wash` 底 + 1px `--border`，`--radius-card`，内边距 10px 14px；左侧 16px `--warning` 盾牌线性图标；正文 12px `--fg-2` 行高 1.55。
- **底部栏**：`position: fixed; left: 0; right: 0; bottom: 0; margin: 0 auto; max-width: 390px`；`--surface` 底、上边框 1px `--border`、`--shadow-lg`；内边距 `10px 14px calc(12px + env(safe-area-inset-bottom))`。
  - 常驻说明行：13px/500；未填齐 `--danger`（如「还需填写：姓名、手机号」），填齐 `--muted`（示例数据那句）。
  - 主按钮「提交」：48px 高、`--radius-pill`、`--accent-gradient`、`--on-brand`、15px/600、`--shadow-sm`；未填齐时 `--surface-warm` 底 + 1px `--border` + `--muted` 字 + `aria-disabled="true"`，**点了不提交、不弹 toast**；提交中内嵌 16px 小转圈 +「提交中…」。
  - 失败条（仅失败态/模拟失败时）在常驻行上方：`--danger-wash` 底、`--radius-xl`、`--danger` 13px、右侧 44px 高「重试」文字按钮。
- 交互反馈只用 `:active`（压暗到 `--bg` 或 `scale(.98)`）。**禁止 `white-space: nowrap`**；所有容器 `max-width: 100%`；多列用 `repeat(N, minmax(0, 1fr))`；长姓名/说明自然折行；页面不得横向滚动。
- 颜色只写 `var(--*)`（遮罩用 `color-mix(in srgb, var(--sidebar) 44%, transparent)`），禁止裸 hex/rgb/hsl；**无直角**（禁止 `border-radius: 0`）；阴影只用 `var(--shadow-*)`；正文 ≥ 13px（副文案 11px 可）；触控目标 ≥ 44×44px；图标一律内联 SVG，不引用外部资源。
- 不用 `<select>`、alert、prompt；无 console 报错。

## 状态覆盖（`data-state` 驱动，`mp-demo` 放 5 个芯片）
`data-mp-state`：`normal` 正常 / `empty` 必填未完成 / `loading` 加载 / `submitting` 提交中 / `failed` 失败，默认正常。
- **正常**：示例数据填满；常驻行「提交后需园所审核，通过后才可接送」；按钮品牌渐变可点。
- **必填未完成**：姓名、手机号清空（关系保留「爷爷」）；对应输入框红色内描边 + 下方红字「请填写姓名」「请填写 11 位手机号」；常驻行「还需填写：姓名、手机号」；按钮置灰禁用。
- **加载**：表单区换骨架（上下文条一行、两张卡共 5–6 条呼吸条），常驻行「正在加载…」，按钮禁用，尊重 reduced-motion。
- **提交中**：表单保持正常内容但 `opacity: .62` + `pointer-events: none`，所有输入/关系行/开关 `disabled`；按钮小转圈 +「提交中…」；常驻行「正在提交，请稍候」。
- **失败**：表单保持正常内容；常驻行上方出现失败条「提交失败，请重试」+「重试」；点重试 → 收起失败条、进入提交中约 1.2s、toast「已提交，待园所审核」。

## 交互（app.js，全部真实可点）
- **关系行 → 底部选择面板**：半透明遮罩 + 从底部升起 `--surface` 面板（顶部 `--radius-2xl`），标题「与孩子关系」，9 个选项各 48px 高、整行可点、选中项右侧 `--accent-deep` 对勾并加粗；「取消」44px 关闭；点遮罩关闭。选中后写回行内、关闭面板、刷新阻断行。
- 开关点击切换 `aria-checked` 与开合样式，未开启的权限不改变任何布局。
- 输入实时更新：已有错误时输入即清除对应错误并重算阻断行。
- 点「提交」：未填齐只做就地提示；填齐后按钮进入提交中约 1.4s，随后 toast「已提交，待园所审核」，按钮恢复、表单保留。
- 返回箭头：toast「已返回授权接送人」。
- `app.js` 监听 `mp:state`：进非正常态时关闭关系面板，回正常态恢复表单；芯片切换由 `_shared/shell.js` 负责，页面不重复绑定。
- 演示条用行内 `style="bottom: calc(112px + env(safe-area-inset-bottom))"` 避让底部固定栏（唯一允许的行内外壳覆盖）。
- toast 自绘：`--fg` 底、`--surface` 字、`--radius-pill`、`z-index` 高于演示条，2 秒消失。

## 输出文件（项目根目录，每个 < 20KB）
1. `index.html` —— 结构
2. `styles.css` —— 只写内容区样式（不写外壳、不重写 tokens）
3. `app.js` —— 上述交互

`tokens.css` 已由流程提供，不要重写、不要覆盖。不要引用任何外部资源或网络字体；不要内联大段 `<style>`；不要 base64 内联图片。
