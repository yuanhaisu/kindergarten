# teacher/22-webview · 对抗性单页审查

- verdict：**WARN**（score 76/100）
- 源码覆盖：源码的 `v-if="src"`（web-view）与 `v-else`（空态）两条分支均已覆盖、空态文案与源码一致；缺口是 `decodeURIComponent` 解析失败这条真实异常路径没有对应错误态，且加载/失败/地址条/刷新均为源码没有的提案，其中地址条与刷新在真机 `<web-view>` 全屏覆盖下不可呈现。

## P1 · 明显缺陷（1）

### 1. [design-core] 页内地址条/重新加载/容器脚注与微信 `<web-view>` 全屏覆盖的硬约束冲突：真实 web-view 会自动铺满整个页面并盖住页内其他元素，原型却把地址卡（`index.html:37-55`）、网页容器内的加载/失败面板（`57-100`）、脚注（`102-105`）画成与网页画布同页共存；真实实现时这些 UI 会被 web-view 覆盖。地址条本身也是浏览器语汇，与批次「一眼看出这是小程序、不是移动网页」的约束相悖。

- 证据：源码 `teacher/src/pages/webview/index.vue:13` 以 `<web-view>` 作为整页唯一内容；原型 `docs/design-previews/2026-10-25-full-redesign/teacher/22-webview/index.html:37-55`（地址条）、`:57-100`（web-card 内加载/失败面板）、`:102-105`（脚注）均在同一页面；`notes.md:38-40` 只把地址条/脚注标为「新增、需产品确认」，未说明 web-view 全屏覆盖导致其不可实现。家长端 28-webview 同构（A38 已记来源条重复），同一约束同样适用。
- 建议：删除页内地址条、刷新按钮与容器脚注，只保留「容器占位 + 空态」；若必须让用户知道链接，改由原生导航栏/入口页承担，并保持源码 `<web-view>` 独占页面。若产品坚持地址条，需把实现方案从 `<web-view>` 改成自绘 H5 容器，并在 notes 明确标注该平台限制。

## P2 · 打磨项（5）

### 1. [logic] 失败态的「重新加载」永远成功：`startReload()` 无论从哪个失败原因触发，1.2 秒后都切回 normal 并 toast「网页已重新加载」；但失败文案把「当前小程序未开通内嵌网页能力」列为原因之一（老师端个人号正是这种情况，重试不可能修复），notes 也明确说不应假装页面正常。

- 证据：`docs/design-previews/2026-10-25-full-redesign/teacher/22-webview/app.js:67-76` 的 `startReload` 固定 `applyState('normal')`；`index.html:95-97` 失败文案含「未开通内嵌网页能力」；`notes.md:41` 写明「原型需要把这个不可用原因讲清楚，而不是假装页面正常」。
- 建议：把「未开通内嵌网页能力」从可重试错误中拆出（禁用重试，只给「返回」/「了解」）；或让演示中的重试保留失败分支，至少不要在永久原因下提示「网页已重新加载」。

### 2. [completeness] 错误态没有覆盖源码唯一可抛出的失败路径：`onLoad` 直接 `decodeURIComponent(query.url || "")`，url 含裸 `%` 时会抛异常，`src` 保持空串，页面落到「未提供页面地址」空态而非错误态；原型的失败文案只写网络/未开通两种原因，空态文案仍是「未提供」，都会误导入口方。

- 证据：源码 `teacher/src/pages/webview/index.vue:7-9`（无 try/catch）、`:13-16`（`v-if="src"` / `v-else`）；原型 `index.html:95-118` 的错误态与空态文案均未覆盖「链接无效」。
- 建议：源码给解码加 try/catch 并进入独立「链接无效」错误态；原型补「链接无效」原因，或把空态文案改为「地址无效或未提供」。

### 3. [redundancy] 正常态把「这是外部/第三方网页、且当前是原型占位」说了三遍：占位块内说明、容器脚注文案、脚注标签；其中「原型占位」是原型脚手架却放在产品脚注位置，挤占唯一的内容区，也让正常态读起来像错误/占位态。

- 证据：`docs/design-previews/2026-10-25-full-redesign/teacher/22-webview/index.html:70-71`「此处由外部网页渲染，原型以中性占位块代替」；`:102-105`「网页内容由第三方站点提供」+「原型占位」；`preview.png` 中同一卡片内三层说明同时可见。
- 建议：正常态只保留一处第三方内容说明（建议容器脚注），删除占位块内的原型说明与「原型占位」胶囊，或把它们收进右下角「原型演示」面板。

### 4. [notes-accuracy] notes 声称本页「由 navigateTo 进入」，并把空态不可达归因于「入口是否传 url」；但老师端源码里没有任何页面 navigateTo 到 `/pages/webview/index`，全仓只有 pages.json 一条路由注册，页面是孤儿路由，正常态与空态在真实运行中都不可达（只能靠演示条）。家长端 28 页同问题已记 A38，老师端这里未记且 notes 反向陈述。

- 证据：`docs/design-previews/2026-10-25-full-redesign/teacher/22-webview/notes.md:9`「由 `navigateTo` 进入」、`:49`「空态目前不可达……真实场景由入口是否传 url 决定」；`teacher/src/pages.json:126` 是 `pages/webview/index` 在 `teacher/src` 内的唯一命中，无任何 navigateTo 引用；A38 记录家长端同类问题。
- 建议：notes 改为「当前无任何入口（孤儿路由），各状态只能通过演示条触发」，删除 navigateTo 表述；上线前补入口或从 pages.json 移除该页。

### 5. [design-core] 即便保留地址条，其两行截断 + `title` 兜底也无法在移动端兑现：长 URL 的尾部被截断后没有展开/复制入口，`title` tooltip 在触屏（以及小程序运行时）不显示，地址条「说清打开的是哪个链接」的目标对真实长链接失效。

- 证据：`docs/design-previews/2026-10-25-full-redesign/teacher/22-webview/styles.css:62-75` 使用 `-webkit-line-clamp: 2`；`index.html:47` 仅靠 `title` 属性兜底；`notes.md:10` 宣称「两行截断，`title` 兜底」。
- 建议：地址条改为只显示域名 + 可展开/可复制的完整地址，或去掉两行截断；不要依赖 `title` 作为移动端兜底。

## 2026-10-27 P1 修复记录

- **P1-1 [design-core] 已修**：删除页内地址条（含其重新加载按钮 `#reloadBtn`）与容器脚注；页面不再有卡片外壳、页边距或任何常驻 chrome，网页区域（`#webSurface` / `.web-surface`）改为导航栏以下整屏铺满，与源码 `<web-view>` 独占整页一致。加载/失败两态改为整屏替代态，与网页区域互斥渲染，不再是「卡片内面板与网页画布同页共存」；`aria-busy` 移到整屏区域，随之删除常驻刷新按钮的禁用逻辑（`app.js`）。
- notes 同步：方向、信息层级、覆盖状态、token 映射、与现状的差异均已改写；「遗留」新增平台硬约束——真实 `<web-view>` 铺满整页并覆盖其他组件，不得再加地址条/来源条/脚注/浮动刷新，链接与来源只能放入口页或打开前的确认层。
- 验证：`od-check --width 390` → `problems: []`；`od-shot` 已重出 `preview.png`（390 / dsf 2）；`od-audit` 批次结构全部通过，本页无 ✗。DOM 实测四态切换互斥：正常/加载/失败时网页区域整屏可见、空态隐藏，`aria-busy` 仅在加载态为 `true`。
- P2 五项未主动处理：P2-1（重试恒成功，仍保留错误态「重新加载」）、P2-2（`decodeURIComponent` 异常路径）、P2-4（notes 的 `navigateTo` 表述）仍在；P2-3 的脚注重复与 P2-5 的地址条截断随 P1 删除载体自然消失，不计为已按 P2 修复。
