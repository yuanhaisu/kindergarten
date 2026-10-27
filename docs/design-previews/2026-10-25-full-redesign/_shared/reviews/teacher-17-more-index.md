# teacher/17-more-index · 对抗性单页审查

- verdict：**PASS**（score 88/100）
- 源码覆盖：源码 9 个固定入口 + 1 个 doctor 条件入口逐条覆盖，文案与跳转去向一致；唯一未覆盖的是普通老师（非 doctor）的 9 行视图——默认直接渲染 doctor 视图，其余无缺项。

## P2 · 打磨项（4）

### 1. [logic] 默认「正常」态把 doctor 专属的「入园晨检」按所有老师可见渲染，并加入源码不存在的「仅保健医」徽标；普通老师的 9 行视图在演示中不可达，页面唯一真实的角色分支只能看到一半

- 证据：index.html:220-237（#group-health 与 .role-badge「仅保健医」无条件渲染，无角色判断）；teacher/src/pages/more/index.vue:28-30（仅 isDoctor 为真时 list.push 入园晨检，普通老师整行不渲染）；notes.md:66 自认「真实产品应整组隐藏」
- 建议：演示条增加「普通老师 / 保健医」角色切换，默认普通老师（9 行、无保健组）；「仅保健医」徽标只作为评审标注，不进入产品 UI

### 2. [redundancy] 分组标题「接送管理」与组内第一行标题「接送管理」完全同名，同一屏上下相邻重复同一信息

- 证据：index.html:156 <h2 class="group-title" id="group-pickup">接送管理</h2>；index.html:163-164 首个 .menu-row 的 .menu-label 文本同为「接送管理」
- 建议：组标题改为「接送」或「接送与签到」，或把行标题改为「接送登记」，消除同名重复

### 3. [notes-accuracy] notes 声称 10 个入口在 390×844 上「约一屏半」，实测页面内容只有约 1.1 屏

- 证据：notes.md:67「10 个入口在 390×844 上约一屏半」；Chrome 390×844 实测 document.documentElement.scrollHeight=929、.mp-app 高度=929，929/844≈1.10 屏（最大滚动 85px，非半屏）
- 建议：改为「约一屏多一点（多出约 85px）」；若要引用 1.5 屏，应注明是 390×620 视口

### 4. [design-core] 骨架屏尾部箭头使用裸值 border-radius: 2px，低于设计核心要求的「无直角，最小 --radius-sm（4px）」

- 证据：styles.css:415 `.skeleton-tail { ... border-radius: 2px; }`；tokens.css:53 `--radius-sm: 4px`
- 建议：改用 var(--radius-sm)，或删除圆角、以纯装饰线条并在注释说明为何不适用最小圆角
