# parent/11-growth-album · 对抗性单页审查

- verdict：**WARN**（score 75/100）
- 源码覆盖：源码的月份条（上/下月箭头 + 年月 + N张照片）、按日期分组与组内张数、三列网格、按当天分组预览、月内空态均已覆盖，未体现 onPullDownRefresh 下拉刷新；新增的日期范围行、骨架屏、错误卡片、整页空态在 notes.md:40-43 中标为提案（其中整页空态缺少数据来源，见 finding）。

## P1 · 明显缺陷（1）

### 1. [logic] 整页空态下月份条没有隐藏、箭头仍可切换：页面停在「还没有照片」，标题却能一路从 2026年10月 翻到 12 月；此时点「刷新看看」落到的是 2026年12月的月内空态「本月暂无照片」，原本有 18 张的 10 月回不去。空态描述的状态与可用操作互相矛盾，也与 notes 的声明相反。

- 证据：styles.css:338-339 在 empty 只隐藏 .mb-meta，全文件没有隐藏 .monthbar 的规则（.monthbar 定义在 styles.css:54）；app.js:78-80 的 locked 仅含 loading/error，empty 下 #nextMonth 仍 enabled；DOM 实测：data-state=empty 时 .monthbar display:block，连点两次后 #monthTitle=2026年12月 而 .state-empty 仍显示「还没有照片」，再点 #emptyRefresh → data-state=normal、标题 2026年12月、[data-month-empty] 显示「本月暂无照片」并 toast「已刷新」；notes.md:21 声称空态「月份条与网格隐藏」。
- 建议：二选一并让实现与 notes 一致：a) 保留整页空态时补 .mp-app[data-state="empty"] .monthbar { display:none; }，并在 empty 下冻结 goMonth（需先有全量照片数据，见另一条 finding）；b) 删除整页空态，把演示 empty 芯片映射为月内空态，月份条保持可用。

## P2 · 打磨项（5）

### 1. [redundancy] 切月 toast 固定在 top:72px，正好压在月份条上：遮住标题尾部（「2026年11月」的“11月”）和右箭头，用户切月后第一眼看到的是被盖住的标题。

- 证据：styles.css:814-819（top:72px; right: max(12px, calc(50vw - 195px + 12px))）；500px 视口实测 #toast rect (288,64)-(433,102) 与 #monthTitle (194,80)-(306,108)、#nextMonth (310,72)-(354,116) 均相交；Temp/rev11/month-empty-toast.png 中 toast 覆盖标题右半与右箭头，页面同时显示「已切换到 2026年11月」。
- 建议：把 toast 移到月份条下方（如 top: calc(64px + 64px)）或改为贴底提示（bottom 加 env(safe-area-inset-bottom)），避免与月份条和预览弹层同区。

### 2. [redundancy] 预览弹层把总张数说了两遍：标题「10月24日 · 第 1 / 3 张」已含总数，脚注又写「共 3 张 · 左右切换查看」；单张时变成「第 1 / 1 张」+「共 1 张」。

- 证据：app.js:174-176 设置 #pvTitle="第 i / n 张"，app.js:187-191 设置 #pvMeta="共 n 张 · 左右切换查看"（单张为「共 n 张」）；index.html:213 与 index.html:224；截图 Temp/rev11/preview-open.png 中两行同屏可见。
- 建议：脚注只保留「左右切换查看」，单张时隐藏整行；总数只由「第 i / n 张」表达。

### 3. [redundancy] 月内空态同一屏说两遍「没有照片」：月份条 meta 的「0 张照片」正下方就是空态块「本月暂无照片」，读起来像两个不同状态。

- 证据：app.js:89 用 info.count 拼「N 张照片」（空月即「0 张照片」）；app.js:100 在 count=0 时显示 [data-month-empty]；index.html:46-47（#monthMeta/#monthCount）与 index.html:196-202（.month-empty / 「本月暂无照片」）；截图 Temp/rev11/month-empty-toast.png 中「0 张照片」与「本月暂无照片」同屏。
- 建议：count=0 时隐藏 .mb-meta（或 .mb-count），空态信息只由空态块承担；_prompt.md:67 要求显示「0 张照片」，若保留该行则删掉空态块标题，二者只留一个。

### 4. [notes-accuracy] notes.md 的网格间距依据写错了：声称源码 $spacing-xs 是 8px 设计值 → 16rpx、原型 4px 是在「收紧」；实际生成 token 是 8rpx（=4px），与原型 --space-xs 完全相同，不存在收紧。

- 证据：notes.md:45；shared-design-tokens/build/scss/tokens-parent.scss:59 `$spacing-xs: 8rpx;`；parent/src/pages/growth/album.vue:88 `gap: $spacing-xs;`；原型 styles.css:192 网格 gap 用 var(--space-xs)（4px）。
- 建议：把 notes.md:45 改为「源码与原型同为 4px（8rpx）」或直接删除这条差异；否则落地者会照 note 去「还原 16rpx」，把网格间距改宽一倍。

### 5. [completeness] 整页空态「还没有照片」在当前数据模型下不可判定：源码与契约都只按 year+month 查询，total 是该月张数，无法区分「从未有过照片」与「这个月没有」。照现状落地，要么把某个月的空误报成整页空态（并按设计隐藏月份条把用户锁死），要么整页空态永远不出现。notes 已标为提案，但提案本身缺少数据前提。

- 证据：parent/src/pages/growth/album.vue:19-21 只取 {year, month} 与当月 total，album.vue:59 当月 groups 为空即「本月暂无照片」；backend/api/api-parent/04-growth.md:229-254 契约也只有 year/month + groups/total（groups 与实现不一致已见 A12，不重复计）；notes.md:43 明确整页空态是分流提案；原型中整页空态只能靠演示芯片触发（app.js 的 setDemoState），月内空态则由 app.js:100 的 count=0 驱动，两条路径共存正说明语义未定。
- 建议：整页空态需新增「是否有过照片」的数据源（如 total_all / first_date / 最近上传时间），否则删除整页空态、只保留月内空态；在数据源确定前不要隐藏月份条，避免用户被困在「还没有照片」且无法换月。

## 2026-10-27 P1 修复记录

- ✅ **已修 P1-1 [logic]** 采纳建议 a（让实现与 notes.md:21「月份条与网格隐藏」一致）：`styles.css` 增加 `.mp-app[data-state="empty"] .monthbar { display: none; }`；`app.js` 把 `empty` 并入 `syncArrows()` 的 `locked`，并在 `goMonth()` 入口对 `empty` 直接 return。改动 3 处（css 1 条 + js 2 处），未动其他状态与视觉。
- **实测**（headless Chrome 390×844）：`data-state=empty` 时 `.monthbar` computed display=none、`#monthTitle` 冻结在 2026年10月、两个月箭头 disabled/aria-disabled=true；强行 `next.click()/prev.click()` 标题不再变化；点「刷新看看」回到 normal、标题仍为 2026年10月、`18 张照片`、不再出现 12 月「本月暂无照片」；正常态与月内空态（11/12 月）的月份条与箭头照常可用。
- **notes.md 同步**：覆盖状态表的「空」「禁用」两行改为「月份条整体隐藏、换月冻结（刷新回到进入空态前的月份）」，空态分流条目补充该约束与「整页空态是强制演示态」。
- **关联但未做**：本清单 P2 全部条目按要求不动。其中 P2-5（整页空态缺「是否曾经有过照片」的数据源）决定的是**何时进入**该状态，与本条修复的**态内自洽**无关；若产品最终采纳 P2-5 的「删除整页空态」路线，本条隐藏/冻结可随该状态一起回退。
- **验证**：`od-check --width 390` → problems `[]`；`od-audit` → 本页 issues 为空、无 ✗；`preview.png` 已按 390/dsf2 重出。
