# shared-ui — 两端共用组件

`teacher/` 与 `parent/` 的 `src/components/` **不接受手改**：唯一真相源在本目录，
构建前由 `node scripts/sync-ui.js` 复制到两端。

```bash
node scripts/sync-ui.js
```

- 脚本只扫本目录**顶层**的 `*.vue`，所以组件放在平铺一层，不要建子目录。
- `npm run dev:mp-weixin` / `build:mp-weixin` 的 `predev` / `prebuild` 会自动执行同步。
- 页面里统一从 `@/components/<Name>.vue` 引入，两端路径一致。

## 组件清单

| 组件 | 用途 |
| --- | --- |
| `AppButton.vue` | 主/次按钮 |
| `AppCard.vue` | 卡片容器 |
| `AppIcon.vue` | 统一图标入口（`/static/icons/<name>.png`） |
| `AvatarBadge.vue` | 头像 + 身份角标 |
| `CalendarEventCard.vue` | 班级日历事件卡 |
| `PageShell.vue` | 页面骨架（padding / 安全区 / 底栏占位） |
| `StateBlock.vue` | loading / empty / error / retry 四态 |
| `StatusPill.vue` | 状态标签 |

> `BottomNav.vue` 是**端内组件**（两端的 4 个 tab 不同），不在本目录。

---

## AppIcon

```html
<AppIcon name="notice" :size="72" label="发布通知" />
<AppIcon name="growth" size="48rpx" />
```

| prop | 类型 | 默认 | 说明 |
| --- | --- | --- | --- |
| `name` | String | `""` | 图标名，映射到 `/static/icons/<name>.png`；可带 `.png` 后缀 |
| `size` | String \| Number | `48` | 数字按 `rpx`；字符串写 `"72rpx"` / `"24px"` |
| `label` | String | `""` | 无障碍名称；留空表示装饰性图标 |

行为：

1. `name` 只接受 `[a-z0-9_-]` 形态；非法或空 name 直接渲染占位块，不拼路径、不抛错。
2. 图片加载失败（例如该端还没有这张资产）时自动降级为中性圆点占位；`name` 变化后重试。
3. 纯 WXML `<image>`，**不生成内联 SVG**。新增图标只需把 PNG 放进两端的
   `src/static/icons/`，页面改 `name` 即可，不需要改组件。

当前覆盖：老师工作台 8 个入口（`notice` / `growth` / `pickup` / `task` /
`calendar` / `leave` / `album` / `more`）。导航与状态图标尚未资产化：老师端底栏
仍用文字字形，家长端用 remixicon 字体；后续切片补 PNG 后可直接用同一个入口。

> 不要重画这 8 张图；它们来自 `teacher/src/static/icons/README.md` 记录的生成流程。
> 家长端 `parent/src/static/icons/` 已持有同名副本。

---

## PageShell

```html
<PageShell bottom-space="180rpx" max-width="430px" flush-top>
  <view class="content">…</view>

  <template #bottom>
    <BottomNav :active="0" :unread-count="unreadCount" />
  </template>
</PageShell>
```

| prop | 类型 | 默认 | 说明 |
| --- | --- | --- | --- |
| `padded` | Boolean | `true` | 套用页面级 padding：水平 `$component-page-padding-x` + 顶部 `$component-page-padding-y` |
| `flushTop` | Boolean | `false` | 去掉顶部 padding（首块自己带间距时用，如老师工作台问候区） |
| `background` | String | `""` | 覆盖页面底色；默认 `$color-bg-page`。可传具体色值，或 `App.vue` 已注入的 `var(--color-*)` |
| `maxWidth` | String | `""` | PC / 平板上限宽度，如 `"430px"`；不传则占满 |
| `bottomSpace` | String | `""` | 固定底栏高度，如 `"180rpx"`；body 会预留 |
| `safeBottom` | Boolean | `true` | 在 `bottomSpace` 之外再让出 `env(safe-area-inset-bottom)` |

slot：

- `default` — 页面内容，走原生页面滚动。
- `bottom` — 固定底栏（BottomNav / 提交条）。组件只做 `position: fixed` 定位，
  **不画背景、不补安全区**；底栏组件自己负责外观。

### 约定

1. **原生导航栏**：两端 `pages.json` 不设 `navigationStyle: custom`。导航栏在页面区域
   之外，页面顶部**不要再加** `statusBarHeight` / 64px 假外壳；页内 `top` 从内容间距算。
   详见 `_shared/SHELL-DECISION.md`。
2. **水平 padding**：用 `$component-page-padding-x`（28rpx），不再写裸 `28rpx`
   （历史页 `teacher/src/pages/notice/index.vue` 仍是裸值，留给 D01 收口）。
3. **底部安全区只算一次**：用 PageShell 时页面不要再写
   `padding-bottom: calc(... + env(safe-area-inset-bottom))`。底栏自身的安全区属于底栏高度，
   `bottomSpace` 负责让内容滚出底栏。
4. **BottomNav 占位**：老师端底栏按样式估算约 147rpx + 安全区，现有页面预留 180rpx。
   新页面统一 `bottom-space="180rpx"`；若底栏高度改了，改页面传参，不改组件。
5. **内层滚动**：需要 `scroll-view` 的页面（如发通知）可继续自己写
   `height: 100vh; display: flex; flex-direction: column` + `scroll-view`，但底部预留仍走
   `bottomSpace`，别在 `scroll-view` 的 `padding` 里再算一次安全区。写法见
   `_shared/CSS-COMPAT.md` §2。

### 试点页根容器核对（F0，不重构）

| 试点页 | 根容器 | 结论 |
| --- | --- | --- |
| 老师工作台 `pages/index/index.vue` | 已改 `PageShell` | ✅ 参考页：`flushTop` + `bottom-space="180rpx"` + bottom slot |
| 老师发通知 `pages/notice/create.vue` | `.page` 100vh flex + `scroll-view` | ✅ 符合约定（内层滚动、底部 224rpx+safe 由页面预留）；未迁 PageShell，留给 D01 |
| 老师通知列表 `pages/notice/index.vue` | `.page` min-height 100vh | ⚠️ 基本符合；水平 padding 写死 `28rpx`（等于 token 值），待 D02 换 token |
| 家长成长报告 `pages/growth/report.vue` | `.page` min-height 100vh | ✅ 符合约定（page padding token + 56rpx+safe）；未迁 PageShell |
| 家长接送二维码 `pages/pickup/qrcode.vue` | `.page` min-height 100vh | ✅ 同上 |

四个未迁移页都不在 F0 重构范围内；它们没有 BottomNav，不存在底栏双重留白。

---

## StateBlock

```html
<StateBlock
  :loading="loading"
  :error="loadError"
  loading-text="加载中…"
  empty-text="暂无数据"
  @retry="load"
/>
```

| prop | 类型 | 默认 | 说明 |
| --- | --- | --- | --- |
| `loading` | Boolean | `false` | 加载态：转圈 + `loadingText` |
| `error` | String | `""` | 非空即错误态：`!` 图标 + 文案 + 重试 |
| `loadingText` | String | `"加载中..."` | 加载文案 |
| `emptyText` | String | `"暂无数据"` | 空态文案 |
| `retryable` | Boolean | `true` | 错误态是否显示「重试」 |

事件：`retry` — 点「重试」时触发。
slots：无（加载/空/错误文案都走 props，避免各页各写一套）。

判定顺序固定为 **loading → error → empty**。错误先于空态，避免「加载失败」被误当成
「今天没有数据」这类危险语义（接送页空列表 = 孩子都接走了）。组件没有「正常态」：
正常内容不进 `StateBlock`，只在 `loading || error || 列表为空` 时挂载。

F0 审计结论：现有实现已覆盖 loading / empty / error / retry 四态，**未改动**。
使用方：`teacher/src/pages/growth/index.vue`、`teacher/src/pages/notice/index.vue`、
`parent/src/pages/growth/index.vue`。

---

## 改共享组件的流程

1. 改 `shared-ui/*.vue`。
2. `node scripts/sync-ui.js`（或直接跑构建的 predev）。
3. 两端各自构建 + `node scripts/check-tokens.mjs <end>`。
4. 不要直接改 `teacher/src/components/` 或 `parent/src/components/`：下次同步会覆盖。

CSS 兼容性见 `docs/design-previews/2026-10-25-full-redesign/_shared/CSS-COMPAT.md`。
