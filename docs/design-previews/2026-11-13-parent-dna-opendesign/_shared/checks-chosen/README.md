# chosen 检查入口

`node _shared/check-variants.mjs --chosen` 的断言不写死在共享检查脚本里。
本目录每个 `.mjs` 文件（`_` 开头的除外，可放共享辅助模块）都是一个入口，
按文件名排序加载。**新增页面/票的断言时只加文件，不改 `check-variants.mjs`。**
例外（spec #88 / ticket #98）：引擎允许增加 `clock` measure 与 `relation`（`eq/gte/lte/gt/lt`）以支持时间顺序断言；该扩展由 #99 终检统一维护。

## 运行

```bash
node _shared/check-variants.mjs             # 8 组变体断言（parent-variants/）
node _shared/check-variants.mjs --chosen    # 31 页 chosen 页集（parent-chosen/）
```

同一个脚本、同一个退出码服务两种模式：不带 `--chosen` 时读
`parent-variants/` 的静态 HTML，跑脚本里写死的 8 组断言；带 `--chosen` 时
读本目录的入口文件并渲染 `parent-chosen/`。**chosen 模式没有第二条验收脚本**，
logo PNG、交互、跨页一致性与文案断言都在这一个入口里。

脚本会：

1. 读取本目录全部入口，合并 `pages` / `states` / `checks` / `crossPage` / `assets`；
2. 按清单核对 31 页的 `index.html` / `styles.css` / `app.js` / `preview.png`
   与 13 个状态截图；
3. 用无头 Chrome 渲染每页（带 `state` 时加 `?state=`），断言 `must` / `mustNot`、
   DOM 属性（`props`）、一次真实点击（`interaction`）与跨页数值（`crossPage`）；
4. 解码 `assets` 指定的 PNG，核对尺寸 / 透明 / 取样色 / 不裁切；
5. 逐页输出 `OK` / `FAIL`，任一失败退出码 1。

## 入口形状

默认导出一个对象，五个字段都可省略：

```js
export default {
  // 可选：声明 chosen 页集的页面目录（清单的真相源，全目录合并）
  pages: ["01-home", "02-login"],

  // 可选：预期状态截图（只核对文件存在，不渲染）
  states: [{ slug: "10-leave-detail", state: "cancelled", file: "preview-cancelled.png" }],

  // 可选：内容 / 属性 / 交互断言
  checks: [
    { slug: "10-leave-detail", must: ["请假详情"], mustNot: ["已取消"] },
    { slug: "10-leave-detail", state: "cancelled", must: ["已取消"] },
    { slug: "04-consent", props: [{ selector: "#consent-privacy", checked: false }] },
    { slug: "04-consent", state: "decline", props: [{ selector: "#decline-sheet", bottomSheet: true }] },
    {
      slug: "02-login",
      interaction: { click: "#wechat-login", expectStayOn: "02-login", expectVisible: "#consent-error" },
    },
  ],

  // 可选：不同页面上同一事实必须一致
  crossPage: [
    {
      label: "通知未读数",
      a: { slug: "06-notice-index", selector: '.notice-card[data-unread="true"]', measure: "count" },
      b: { slug: "01-home", selector: ".unread-badge", measure: "number" },
    },
  ],

  // 可选：PNG 资产（file 相对 _shared/）
  assets: [{ file: "logo-dna-168.png", width: 168, height: 168, transparentCorners: true, samples: [] }],

  // 可选：页数/状态数契约（一般只在 inventory.mjs 里写）
  expected: { pages: 31, states: 13 },
};
```

### check 字段

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `slug` | 是 | 页面目录名，如 `06-notice-index` |
| `must` | 否 | 必须出现的文本（字符串数组） |
| `mustNot` | 否 | 不得出现的文本（字符串数组） |
| `within` | 否 | CSS 选择器；断言只作用于该容器的可见文本 |
| `state` | 否 | 状态值；渲染时打开 `index.html?state=<值>` |
| `props` | 否 | DOM 属性断言数组，见下 |
| `interaction` | 否 | 点击交互断言，见下 |

`props` 目前支持 `checked`（原生 checkbox）与 `bottomSheet`（元素是否贴底、横向占满：
危险确认必须是底部确认区，不是居中弹窗）。`selector` 未命中算失败，不静默跳过。

`interaction` 与 `must` / `mustNot` / `within` / `props` 互斥；字段：

| 字段 | 说明 |
| --- | --- |
| `steps` | 可选，点击前的准备动作 `{ click }[]` |
| `click` | 必填，最后点击的元素选择器 |
| `expectNavigateTo` | 点击后 `location.href` 应包含的路径 |
| `expectStayOn` | 点击后 `location.href` 应仍包含的路径（不能有延迟跳转） |
| `expectVisible` | 点击后应可见的元素选择器 |
| `expectVisibleText` | 与 `expectVisible` 配套的文案包含断言 |
| `expectHidden` | 点击后应不可见的元素选择器（如关闭后的 sheet） |
| `settleMs` | 不应跳转时等待多久再判定，默认 1500ms |

`crossPage` 的两端各是 `{ slug, selector, measure }`；`measure` 为 `count`（元素数量）、
`number`（容器文本里的第一个数字）或 `text`（容器文本）。两端取值必须相等。

`assets` 的 `samples` 每项为 `{ name, svg: [x, y], rgb: [r, g, b] }`，坐标按
`viewBox`（默认 256）换算到导出像素；默认要求取样点不透明，写 `opaque: false` 可跳过。

### 断言的文本是什么

- 整页断言：`document.title + " " + document.body.innerText`，空白折叠成一个空格。
- `within` 断言：只取 `document.querySelector(within).innerText`。
- 用 `innerText` 而不是静态 HTML：`hidden` / `display:none` 的内容不算，
  所以 `?state=` 的隐藏分支不会在默认态误命中。
- 断言的文本是**渲染后的可见文案**，不是 CSS 类名或 DOM 结构。

`within` 用来避免正文里的同名词造成误报。例：06 的通知筛选组：

```js
{ slug: "06-notice-index", within: ".filter-row", must: ["全部", "未读"], mustNot: ["园所"] }
```

### 写一条新断言的规矩

- 一个文件对应一页或一张票（如 `43-onboarding.mjs`、`46-classification.mjs`），
  不要把所有页面塞进同一个文件。
- 只断言用户可见事实：文案、字段有无、默认状态、真实点击后果、跨页一致的数字。
- 不要断言 CSS 类名、内部 DOM 结构或 `data-*` 实现细节。
- `data-od-id` 是原型评审钩子，可作为 `within` 的稳定容器定位；不要把它的存在或取值当作业务事实断言。
- 页面文案会随票改，断言写该票的验收口径；共享冒烟断言（`smoke.mjs`）
  只锁定页面身份，不锁定业务口径。
