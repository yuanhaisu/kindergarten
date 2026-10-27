# 全量页面重设计 · 共享约束（每个 subagent 必须先读）

> 批次：2026-10-25-full-redesign
> 目标：老师端 + 家长端全部页面，每页一份可独立打开的 HTML 原型，供用户一次性审阅。
> 本文件只写**所有页面都一样的硬约束**；每页的具体内容由该页 subagent 从源码里读出来。

---

## 1. 交付物（缺一不可）

每个页面一个目录，例如 `docs/design-previews/2026-10-25-full-redesign/teacher/01-workbench/`：

| 文件 | 要求 |
|---|---|
| `index.html` | 单页原型，390px 手机视口，`<html lang="zh-CN" data-surface="teacher|parent">` |
| `styles.css` | 所有样式；**不得**内联大段 `<style>` 里重复 tokens |
| `app.js` | 所有交互（tab 切换、下拉、空态切换、底部弹层等） |
| `tokens.css` | 从设计系统原样复制：`D:/yuanhaisu/personal_projects/open-design/.od/design-systems/nurture-planet-local/tokens.css` |
| `notes.md` | 设计说明（见第 7 节） |
| `preview.png` | 由脚本自动生成，不用手写 |

禁止：多方向 A/B 变体、多页面拼在一份文件里、引用外网资源、base64 内联大图。

---

## 2. 微信小程序风格（本批次最高优先级）

> **外壳不许自己画。** 状态栏、导航栏、胶囊、返回箭头、tabBar、演示状态条，一律用
> `_shared/shell.css` + `_shared/shell.md` 里的标记与类名（`mp-statusbar`、`mp-navbar`、`mp-capsule`…）。
> 页面只写内容区；`styles.css` 里出现 `.mp-*` 定义即违规。引入顺序：`tokens.css` → `../../_shared/shell.css` → `styles.css`。

原型要让人一眼看出「这是小程序」，不是移动网页：

1. **微信导航栏**：顶部固定 44px 导航栏，标题居中、14px 加粗；右上角画胶囊按钮（`···` + `◎`，圆角胶囊，1px 边框）。状态栏区域留 20px（时间 `9:41` + 信号/wifi/电池的小图标，用 CSS/SVG 画，不用图片）。
2. **页面结构**：`状态栏 + 导航栏 + 可滚动内容区`。内容区背景用页面底色，卡片浮在上面。
3. **tabBar**：家长端 `pages/index/index`、`pages/growth/index`、`pages/pickup/index`、`pages/profile/index` 这四个页面**必须**画底部 tabBar（4 项：首页/成长/接送/我的，选中项用品牌色）。老师端虽然 `pages.json` 没有 `tabBar`，但源码在工作台 / 班级 / 消息 / 我的 4 个常驻页自绘了 4 项底栏；这 4 页必须画同一套底栏（首页/班级/消息/我的），其余老师端页面不画。统一使用 `_shared/shell.css` 的 `.mp-tabbar` / `.mp-tab`，不要每页重画。
4. **页面内返回**：非 tabBar 页面在导航栏左侧画返回箭头（`‹`），标题相应变成该页标题。
5. **交互语汇用小程序原生件**，不要用 Web 桌面件：
   - 列表行 = cell：左图标 + 主文案 + 右侧值 + `›` 箭头
   - 选择用底部 action sheet / picker 风格面板，不用 `<select>`、不用右上角下拉菜单
   - 提示用 toast / 轻提示条，不用 alert / 右上角 notification
   - 危险操作二次确认用微信风格居中对话框（标题 + 说明 + 取消/确定）
   - 加载用骨架屏或 `loading` 小圆环，不用进度条
6. **点击反馈**：只做 `:active` 缩放/变暗（`transform: scale(.98)` 或背景压暗），**不要**做 hover 才出现的信息。桌面鼠标 hover 可以给轻微反馈，但信息不能只在 hover 时可见。
7. **触控目标** ≥ 44×44px；正文 ≥ 13px；对比度满足 WCAG AA。
8. **安全区**：底部内容留 `env(safe-area-inset-bottom)`（用 `padding-bottom: calc(24px + env(safe-area-inset-bottom))` 这种写法兜底）。
9. **单列纵向流**，不做横向滚动、不做侧边栏、不做 hover 浮层菜单、不做桌面多栏。
10. **禁止 `white-space: nowrap`** 用于可能超宽的文本；多列统计用 `grid-template-columns: repeat(N, minmax(0, 1fr))`。
11. **门禁**：每页必须跑 `node scripts/od-check.mjs <out>/index.html --width 390`，`problems` 必须为空数组（横向溢出、元素越界、console error 都算失败）。

---

## 3. 品牌与 token 契约

品牌北极星：**Tender Cosmos 温柔宇宙** —— 低对比、无锐角、圆润边界、琥珀色柔光。

- 颜色**只能**写 `var(--*)`，不得写裸 hex / rgb。需要新颜色就说明「建议回到 tokens.json 新增」，原型里先用最接近的 token。
- 老师端 `data-surface="teacher"`：暖奶白底 `var(--bg)`，主色偏奶暖橙 `var(--accent)`。
- 家长端 `data-surface="parent"`：冷雾白底，主色鲜橙。
- **暖色占比 ≤ 10% 面积**：大面积永远是中性底；渐变只用于一个紧凑 hero 或主按钮。
- **无直角**：最扁平元素 ≥ `var(--radius-sm)`；按钮/标签/头像 `var(--radius-pill)`；卡片 `var(--radius-card)`。
- **阴影只用 `var(--shadow-*)`**，不写黑色投影。
- 层级只靠字重 + 字号：标题 600 / 正文 400 / 标签 500。

主要变量（完整见 `tokens.css`）：
`--bg --bg-gradient --surface --surface-warm --fg --fg-2 --muted --hint --accent --accent-soft --accent-deep --accent-wash --accent-gradient --border --success --warning --danger --info` 及各自 `-wash`；`--shadow-sm/md/lg`；`--radius-sm/xl/2xl/card/pill`；`--space-xs/sm/md/lg/xl`；`--text-xs/sm/base/md/lg/xl/2xl/3xl`；`--input-height --page-pad-x --page-pad-y --avatar-sm/md/lg --badge-pad-x`。

---

## 4. 内容必须来自真实产品

1. **先读源码再设计**：读该页的 `src/pages/<path>.vue`，以及它调用的 `src/api/*.js`，以真实字段、真实按钮、真实跳转为准。
2. **不虚构功能**：源码里没有的入口/字段不要发明。可以做「状态覆盖」，但不能新增业务能力。
3. **数据用中文示例**，符合幼儿园语境：孩子「乐乐 / 小雨 / 朵朵」，班级「小一班 / 中二班」，老师「王老师 / 李老师」，日期用 `10月24日 周五` 这种写法。
4. **文案克制**：不写营销话术（「记录每一个精彩瞬间」这类最多用在空态，且只一句）。按钮文案用动词，2–4 字。
5. **不虚构数字**：统计类数字要与页面语境自洽（比如全班 28 人、今日出勤 26 人）。

---

## 5. 状态覆盖（数据区域必须有）

每页至少覆盖：

- **正常态**：有数据的完整展示（默认渲染这个）
- **空态**：图标/插画位 + 一句说明 + 一个主行动按钮
- **加载态**：骨架屏 2–3 行（用 `app.js` 里的演示开关切换，默认不显示）
- **错误态**：一句话 + 「重试」
- **禁用态**：不可点的按钮/表单项要有明显降级

页面上放一个**演示用的状态切换条**（固定在右下角或底部上方，半透明小胶囊，仅原型可见，标注「原型演示」），可以在 正常/空/加载/错误 之间切换。它不能干扰主体视觉，用户审阅时能关掉更好。

---

## 6. 交互必须真的能点

`app.js` 里至少做到：

- 所有 tab / 分段控件能切换
- 列表项点击有反馈（toast「已打开…」或切换详情态）
- 底部弹层能打开/关闭
- 状态切换条能切换四种状态
- 无 JS 报错（`console` 干净）

不要用 `alert()`。不要用 `window.prompt`。

---

## 7. notes.md 必填

```markdown
# <页面中文名>（<端>）

- 源码：teacher/src/pages/xxx/index.vue
- 设计系统：user:nurture-planet-local，surface=teacher
- 方向：一句话
- 信息层级：从上到下 3–6 条
- 覆盖状态：正常 / 空 / 加载 / 错误 / 禁用（各自怎么表现）
- token 映射：关键元素用了哪些 var(--*)
- 与现状的差异：改了哪些、为什么
- 遗留：没解决或需要产品确认的点
```

---

## 8. 红线

- 不改 `teacher/` `parent/` 源码，本批次只出原型。
- 不联网抓图，不生成带水印图片进原型。
- 不把原型 CSS 直接抄进仓库源码（那是下一步 Impeccable 的事）。
- 不在一个 subagent 里做第二个页面。

---

## 9. `_prompt.md` 骨架（交给 OpenDesign 内层 agent 的指令）

> 直接按这个结构填，别自由发挥。文件写到该页输出目录，然后作为 `--brief` 传给 `scripts/od-design-page.mjs`。

```markdown
你是幼儿园管理平台「养成星球」的资深移动端设计师。为<端>做一个页面的高保真 HTML 原型。

## 唯一任务
页面：<中文名>（源码：<path>）
只做一个方向，不要 A/B 变体，不要多页面。写完文件即停止，不要提问，不要联网。

## 这个页面真实的产品事实
<从源码里读出来的：页面目标、区块顺序、字段、按钮、点击后去哪、权限/角色差异>

## 示例数据（中文，幼儿园语境）
<3–8 条真实感数据>

## 视觉与结构硬约束
- <html lang="zh-CN" data-surface="<teacher|parent>">
- 390px 手机视口；单列纵向流；不做桌面布局
- 微信小程序外壳：20px 状态栏（9:41 + 图标）+ 44px 导航栏（标题居中 + 右侧胶囊）；<需要返回箭头 / 不需要>
- <家长端 tabBar：画/不画；老师端：仅 01/15/18/19 四个常驻页画，其余不画>
- 颜色只写 var(--*)，禁止裸 hex/rgb；无直角；阴影只用 var(--shadow-*)
- 暖色面积 ≤ 10%；字号层级靠字重+字号
- 触控目标 ≥ 44px；正文 ≥ 13px
- 数据区必须覆盖：正常 / 空 / 加载骨架 / 错误；另做一个「原型演示」状态切换胶囊
- 点击必须有 :active 反馈；不用 alert/prompt；无 JS 报错

## 输出文件（项目根目录）
1. index.html —— 结构
2. styles.css —— 样式（引用 tokens.css）
3. app.js —— 交互
tokens.css 已由流程提供，不要重写，不要覆盖。
每个文件 < 20KB。不要引用任何外部资源。
```

