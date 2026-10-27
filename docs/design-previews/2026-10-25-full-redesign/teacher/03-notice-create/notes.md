# 发布通知（老师端）

- **源码**：`teacher/src/pages/notice/create.vue`（另读 `teacher/src/api/teacher-notices.js`、`server/internal/domain/notice.go`、`backend/api/`）
- **设计系统**：`user:nurture-planet-local`，`surface=teacher`；外壳用 `_shared/shell.css` 的 canonical shell
- **方向**：「填一张会发到 28 位家长手机上的告知单」——把散落的 8 段卡片收成一条 **类型 → 快捷填写 → 内容 → 受众 → 发送设置** 的表单流，选择类字段全部沉到小程序底部弹层，底部一条常驻提交栏负责回答「发给谁 / 什么时候发 / 还差什么」。

## 信息层级（从上到下）

1. **系统外壳**（`_shared/shell.css`）：20px 状态栏 + 44px 导航栏（「发布通知」居中、**左侧返回箭头**、右侧 `···`｜`◎` 胶囊）。子页，返回箭头对应源码 `uni.navigateTo` 入栈。
2. **通知类型**：5 个互斥胶囊（作业/带物/活动/健康/其他），`grid-template-columns: repeat(5, minmax(0,1fr))`，选中用 `--accent-wash` + `--accent-deep`。
3. **快捷填写**：cell 行「常用模板 · 3 条可用 ›」→ 底部模板列表弹层（标题 + 两行预览 + 使用）。点「使用」把 `title / content / category` 写入表单。
4. **通知内容**（一张卡内 3 个 cell）：标题（右端 `14 / 80`）→ 正文（右端 `38 字`；源码无上限，只给字数不设限）→ 图片 / 视频（`2/9`，两个 CSS 抽象缩略图 + 添加位）。
5. **接收范围**：cell 行「接收班级 → 小一班全体家长 ›」，脚下「1 个班级 · 28 位家长将收到通知」。点开为**底部多选弹层**（班级名 + 人数 + 圆形勾选，顶部 取消 / 完成）。
6. **发送设置**：两行开关 —— 需要家长确认收到（默认开，副文案沿用源码「开启后可查看已读未读状态」）、定时发送（默认关，副文案「立即发送」；开启后展开发送日期 / 发送时间，各自打开日期、时间**滚轮弹层**）。
7. **底部固定提交栏**（`position: fixed`，宽 `min(390px, 100%)`）：一行灰摘要（「小一班 · 立即发送 · 需家长确认」）+ 全宽主按钮「发布」，`padding-bottom: calc(12px + env(safe-area-inset-bottom))`；`.page` 留 `calc(112px + env(safe-area-inset-bottom))` 底距避免遮挡。老师端**无 tabBar**。
8. **确认弹层**：点发布先弹微信居中对话框（标题 + 「发布后小一班全体家长会收到微信提醒。」+ 类型 / 发送方式两行）+ 取消 / 确认发布。

## 覆盖状态

状态由 `_shared/shell.js` 写到 `.mp-app[data-state]`，页面用属性选择器接管显隐、用 `mp:state` 事件接管数据重置：

| 演示芯片 | 对应真实状态 | 表现 |
|---|---|---|
| 正常 | 表单已填、班级已选中 | 5 个区块完整；发布按钮 `--accent-gradient` 可点 |
| 空态 | 标题、正文、附件皆空（源码 `validate()` 前两道） | 标题 / 正文下方各一条 `--danger-wash` 提示；摘要「还差标题和正文」；发布按钮 `--surface-warm` 底 + `--border` 边 + `--muted` 字并 `disabled` |
| 加载 | `GET /teacher/class/list` + `/notices/templates` 未回 | 接收范围值、模板条数变 shimmer 骨架；接收班级行不可点；摘要「正在加载班级信息…」；按钮禁用 |
| 失败 | 班级列表请求失败（源码 `catch { /* */ }` 静默吞掉，原型改为显式暴露） | 接收范围行右侧 `--danger` 文案「班级列表加载失败」+「重试」；摘要「班级信息未加载，暂不能发布」；按钮禁用；重试回正常并 toast |
| （独立）正文为空 | 正常态下清空正文直接点发布 | 正文下出现同一条 `--danger` 提示 + toast「请输入通知内容」，**不弹确认框**（对应源码 `validate()` 直接 return） |
| （独立）发布成功 | 确认发布（源码 toast「已发布」后 600ms `navigateBack()`） | 单页原型**留页复位**：toast「已发布，返回老师工作台」，标题 / 正文 / 附件清空、其余字段回默认，摘要「通知已发布 · 表单已复位」，发布按钮禁用并显示「已发布」；开始写新草稿（标题 / 正文 / 附件非空）或切演示芯片后解锁，同一份通知无法重复群发 |

## token 映射

| 元素 | token |
|---|---|
| 页面底 / 卡片 | `--bg` / `--surface` |
| 文字层级 | `--fg`（标题、输入值）/ `--fg-2`（区块标题）/ `--muted`（说明、占位、禁用）/ `--hint` |
| 主操作 | `--accent-gradient`（仅底部「发布」）+ `--on-brand` |
| 选中态 | `--accent-wash` 底 + `--accent` 边 + `--accent-deep` 字（类型胶囊、开关、勾选圈） |
| 分割与骨架 | `--border` |
| 错误 | `--danger` / `--danger-wash` |
| 圆角 | `--radius-card`、`--radius-pill`、`--radius-lg`（缩略图）、`--radius-2xl`（弹层上缘） |
| 阴影 | `--shadow-md`（卡片）、`--shadow-lg`（弹层 / 对话框） |
| 间距 / 字号 | `--page-pad-x`、`--space-*`、`--text-xs…--text-2xl`；层级只靠字重 + 字号 |

## 与现状的差异

1. **8 段卡片合并为 5 组**：源码「类型 / 标题 / 内容 / 附件 / 接收范围 / 确认 / 定时 / 模板」各占一卡；现把标题、正文、附件并进「通知内容」一张卡（卡内细分隔）。
2. **常用模板上移到类型之后，并从内联卡片改为 cell + 底部弹层**：源码把模板平铺在表单最底部（3 条就拉长一屏），而它的用途是「先套模板再改」，放在填完发送设置之后失去意义；且标题旁「查看更多」是无 `@click` 的死链。数据与「使用」行为（写入 title/content/category）完全不变。
3. **接收班级从内联勾选胶囊改为底部多选弹层**：源码仅在 `classList.length > 1` 时展示内联勾选，单班时无法确认发往哪；改成常驻 cell 行，值始终可见。
4. **新增发布前确认弹层**（源码无，点发布直接请求）。仅增加一次确认，不新增业务能力。
5. **类型选中色由信息蓝改品牌暖橙**：源码 `$color-semantic-info` 蓝，与老师端琥珀体系冲突。
6. **新增失败态**：源码 `onMounted` 的 `catch { /* */ }` 会静默失败，老师看到「请选择班级」而不是「加载失败」。
7. **附件计数与上限呈现**：保留 `9 - attachments.length` 的上限，补 `2/9` 可见计数。
8. **发布成功后的收尾**：源码 `submit()` 成功后 toast「已发布」，600ms 后 `navigateBack()`（无上一页则 `reLaunch('/pages/index/index')`），本页随即销毁。原型是单页静态预览，改为**留页复位**：清空标题 / 正文 / 附件、其余字段回默认，发布按钮禁用并显示「已发布」，摘要「通知已发布 · 表单已复位」；点开新草稿才解锁。这样既不虚构工作台页面，也不会让同一份已发出的通知被二次发布。

## 外壳回归

按 `_shared/shell.md` 把自造外壳换成 canonical shell，只换实现、不重设计：

- **结构**：`div.mp-app[data-state]` → `div.mp-statusbar / header.mp-navbar（mp-back + mp-title + mp-capsule）` → `main.mp-main`；子页保留 `mp-back`。
- **引入**：`tokens.css` → `../../_shared/shell.css` → `styles.css`；页尾 `app.js` 之后引入 `../../_shared/shell.js`。
- **删除**：原 `.topbar / .statusbar / .navbar / .nav-back / .nav-title / .nav-capsule / .cap-btn` 与整套 `.demo*` 样式全部移除；`styles.css` 中不再有 `.mp-*` 类定义，只保留 `.mp-app[data-state="…"]` 状态选择器（shell.md §状态约定 的写法）。
- **四态**：原来的自定义演示条 JS（`demoToggle / demoBody / setDemoOpen`）整段删除；`shell.js` 负责折叠与写 `data-state`，页面监听 `mp:state` 事件同步数据（`setView`），并在 `renderChrome()` 里回写 `data-state` + 同步 chip 的 `aria-pressed`，保证「打字把空态解除」「重试回正常」时演示条不脱节。
- **连带调整**（原实现依赖固定高度 + 内部滚动，换成文档流后必须改）：
  - 提交栏 `position: sticky`（原在内部滚动容器里）→ `position: fixed` + `left: 50%` 居中；
  - `.page` 补 `calc(112px + safe-area)` 底部留白；
  - 弹层 `.layer` 与 toast 从 `absolute` 改 `fixed`（否则给整个文档高度定位）；
  - `.sheet` 加 `max-width: 390px`，宽窗口下不铺满；
  - `scroll.scrollTo` 改 `window.scrollTo`（`.mp-main` 已不是滚动容器）。
- **演示条抬升**：`mp-demo` 用行内 `style="bottom: calc(104px + env(safe-area-inset-bottom))"` 抬到固定提交栏之上。shell.md 给的是 72px，但本页提交栏约 95px 高，72px 会压住「发布」按钮。

## 遗留

1. **「是否置顶」未画**：调度说明里提到该字段，但源码 `form` 无此项；`is_top` 只存在于 `server/internal/domain/announcement.go`（管理端公告），`notice.go` 没有。按「不虚构字段」处理，需产品确认通知是否要加置顶能力。
2. **确认弹层属新增交互**，需产品确认是否保留（源码为「点发布即发」）。
3. **附件选择**：原型用 CSS 抽象缩略图占位。真实实现走 `uni.chooseImage`（原生对话框，CDP 拦不到），Impeccable 落地时需真机验证计数与删除。
4. **正文无长度上限**：源码 `textarea` 未设 `maxlength`，原型只显示字数；若后端有上限应回到 `backend/api/` 明确后再加提示。
5. **定时发送的时间语义**：源码把日期与时间拼成 `YYYY-MM-DD HH:mm` 直接提交，未做「不能选过去时间」校验；原型同样不校验，落地时建议补。
6. **预览图长度 1130px**：整页一次性展示（提交栏是 fixed，viewport 低于内容时会被截在中段），不代表真机单屏。
7. **主按钮 / 选中胶囊 / 错误文案的对比度不达 WCAG AA（token 层，批次级）**：`--on-brand` #FFFFFF 压 `--accent-gradient`（#ffa866→#ff8a3d）仅 1.90–2.35:1；`--accent-deep` #e6762a 压 `--accent-wash` #fff7f0 为 2.83:1（类型选中胶囊、弹层「完成」「使用」同一组）；`--danger` #EF4444 压 `--danger-wash` #FEE2E2 为 3.08:1。`tokens.css` 是 `scripts/sync-open-design-adapter.mjs` 的生成产物（文件头注明「Do not edit by hand」），需回 `shared-design-tokens/tokens.json` 压深品牌橙与语义红并同步三端，不在本页目录内解决。
