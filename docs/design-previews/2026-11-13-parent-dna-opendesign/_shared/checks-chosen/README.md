# chosen 检查入口

`node _shared/check-variants.mjs --chosen` 的断言不写死在共享检查脚本里。
本目录每个 `.mjs` 文件（`_` 开头的除外，可放共享辅助模块）都是一个入口，
按文件名排序加载。**新增页面/票的断言时只加文件，不改 `check-variants.mjs`。**

## 运行

```bash
node _shared/check-variants.mjs --chosen
```

脚本会：

1. 读取本目录全部入口，合并 `pages` / `states` / `checks`；
2. 按清单核对 31 页的 `index.html` / `styles.css` / `app.js` / `preview.png`
   与 13 个状态截图；
3. 用无头 Chrome 渲染每页（带 `state` 时加 `?state=`），再按 `must` / `mustNot` 断言；
4. 逐页输出 `OK` / `FAIL`，任一失败退出码 1。

## 入口形状

默认导出一个对象，三个字段都可省略：

```js
export default {
  // 可选：声明 chosen 页集的页面目录（清单的真相源，全目录合并）
  pages: ["01-home", "02-login"],

  // 可选：预期状态截图（只核对文件存在，不渲染）
  states: [{ slug: "10-leave-detail", state: "cancelled", file: "preview-cancelled.png" }],

  // 可选：内容断言
  checks: [
    { slug: "10-leave-detail", must: ["请假详情"], mustNot: ["已取消"] },
    { slug: "10-leave-detail", state: "cancelled", must: ["已取消"] },
  ],

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

- 一个文件对应一页或一张票（如 `43-onboarding.mjs`、`06-notice-index.mjs`），
  不要把所有页面塞进同一个文件。
- 只断言用户可见事实：文案、字段有无、默认状态、状态页查询。
- 不要断言 CSS 类名、内部 DOM 结构或 `data-*` 实现细节。
- 页面文案会随票改，断言写该票的验收口径；共享冒烟断言（`smoke.mjs`）
  只锁定页面身份，不锁定业务口径。
