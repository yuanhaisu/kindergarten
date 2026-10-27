# teacher/20-doctor-checkin · 对抗性单页审查

> 2026-10-27 token 对比度已统一修复（主按钮/语义色全部 ≥4.5:1），见 `_shared/REVIEW.md` §3.2。

- verdict：**WARN**（score 72/100）
- 源码覆盖：统计、政策、园级事件入口、班级筛选、待检/已完成列表、体温与五项三态、异常拍照、备注、隔离、三档结论和四种状态都已覆盖，缺的是通知失败分支不可达、体温输入缺少源码的 maxlength=4/数字约束，以及空态 18/3 的静态数字不会随提交更新。

## P1 · 明显缺陷（3）

### 1. [logic] 空态「回看已完成」与已渲染内容自相矛盾：进入空态时 18 条完成记录和 completedCount=18 已经全部显示，点击后仍是 18 条，只是把 5 条原本待检的记录打上「新完成」并提示「已补齐 5 条完成记录」；再点一次提示「已完成记录已置顶」但顺序不变。空态行由静态 SEED 生成，不读 records，提交过的结论会被重置；空态统计也写死 18/0/18。

- 证据：app.js:141-152 emptyRows() 用 SEED，!emptyReviewed 时直接返回全部 18 条；app.js:318-333 renderCompleted() 在空态渲染 emptyRows() 并给 wasPending 行加 is-fresh；app.js:525-536 首次点击 toast「已补齐 5 条完成记录」，二次点击「已完成记录已置顶」；app.js:181 空态统计硬编码 {18,0,18}。运行时空态点击前 #completedCount=18、#completedList li=18、.is-fresh=0；点击后仍 18 条、.is-fresh=5。
- 建议：空态不要预先渲染完成列表，改为点击后再渲染；或把 CTA 改成「查看已完成」并删掉「补齐/置顶」文案；emptyRows 改为从 records 派生，renderStats 不写死 18/0/18。

### 2. [redundancy] 园级事件入口被从源码的筛选栏上方移到页面最底部，位于 13–18 条已完成记录之后；它是进入 pages/doctor/calendar 的唯一入口，正常态距页顶约 1344px、空态约 1394px，首屏 844px 内完全不可见，等于把唯一跳转入口埋掉，且 notes 的信息层级把它列为第 6 项、差异清单未提这次重排。

- 证据：源码 teacher/src/pages/doctor/checkin.vue:207-212 garden-entry 在 filter 214-225 之前，goGardenCalendar 在 :185；原型 index.html:113-120 eventEntry 在 completedSection 103-111 之后；运行时 #eventEntry 绝对 top 正常态 1344、空态 1394，视口 844；notes.md:15 把入口列为第 6 项，notes.md:34-44 未提重排。
- 建议：移回今日卡下方、班级筛选上方；若产品坚持置底，至少在首屏给快捷入口，或把入口并入今日卡并在 notes 标注这次重排。

### 3. [design-core] 关键操作与状态文字对比度远低于 WCAG AA：主按钮白字压暖橙渐变约 1.90–2.35:1；「需观察」「劝返」和三档状态标签用语义色压同色浅底约 2.74–3.08:1；「待晨检」约 2.83:1，政策胶囊约 4.24:1，字号仅 11–14px。低对比是品牌方向，但这些文字承载操作结果与幼儿状态，已低于可读阈值。

- 证据：styles.css:104 .submit-primary、:107-109 .submit-warn/.submit-danger、:132 .primary-btn；styles.css:49 .child-check、:150-157 .status-tag、:13 .policy-pill；tokens.css:18 --on-brand:#FFFFFF、:24 默认 accent-gradient #FB923C→#F97316、:89 teacher 渐变 #ffa866→#ff8a3d、:28-35 语义色。按实际色值计算：白字压 #ffa866=1.90、压 #ff8a3d=2.35；#16A34A/#DCFCE7=3.00；#CA8A04/#FEF9C3=2.74；#EF4444/#FEE2E2=3.07；#e6762a/#fff7f0=2.83；#2563EB/#DBEAFE=4.24。
- 建议：状态文字改用 --fg/--fg-2 压浅底，或把语义色加深到 ≥4.5:1；主按钮改深色字或加深渐变；在 tokens.json 统一修，不要逐页打补丁。

## P2 · 打磨项（5）

### 1. [notes-accuracy] notes.md 多处与实现/源码不符：提交接口写成 POST /api/v1/doctor/health/checks，实际是 PUT /doctor/health/check/:id；源码错误态被描述成 toast，实际是内联错误 + 重试；把「异常项才拍照」列为与源码的差异，但源码每个检查项也有 v-if="isAbnormal(...)"；token 映射声称卡片用 --surface-warm、数字用 tabular-nums，styles.css 实际用 --surface 且没有 tabular-nums；「顶部 toast」实际固定在底部 84px。

- 证据：notes.md:3 vs teacher/src/api/doctor-health.js:14 与 backend/api/api-doctor/01-health.md:85；notes.md:43 vs teacher/src/pages/doctor/checkin.vue:231-234；notes.md:37 vs checkin.vue:290,302；notes.md:28,30,32 vs styles.css:9,29,39,159（无 --surface-warm、无 tabular-nums）；notes.md:24 vs styles.css:175 bottom:84px。
- 建议：改正接口方法/路径与错误态描述；删除与源码相同的「差异」；token 名改成实际使用值或补上 tabular-nums；toast 位置改为「底部」。

### 2. [navigation] 返回按钮的 aria-label 和点击 toast 都写「返回工作台」，但该页在源码中的唯一入口是 pages/more/index（更多），工作台没有晨检入口；返回意图应指向「更多」，或泛化为「返回上一页」。

- 证据：index.html:24 aria-label="返回工作台"；app.js:508 showToast('返回工作台')；teacher/src/pages/more/index.vue:29 是唯一 push 入园晨检的地方；grep teacher/src 只有 pages.json 与 more/index.vue 命中 pages/doctor/checkin，工作台无入口。
- 建议：改为「返回更多」或「返回上一页」；若支持通知/深链进入，用 uni.navigateBack 语义而不是固定页面。

### 3. [design-core] 主行动按钮「健康入园」和三档结论按钮用 --radius-lg（8px），不是品牌规定的 --radius-pill；同页 chips、状态标签、头像都是 pill，最重要的三个按钮反而成了方角，视觉语言不一致。

- 证据：styles.css:104 .submit-primary border-radius:var(--radius-lg)；styles.css:107 .submit-warn,.submit-danger 同；styles.css:132 .primary-btn 同；tokens.css:55 --radius-lg:8px、:58 --radius-pill:9999px；运行时 computed borderRadius 为 8px；对比 styles.css:28-29 的 chips 使用 --radius-pill。
- 建议：三处改用 var(--radius-pill)，与批次内其他 pill 按钮一致（如 03-notice-create 的发布按钮）；若 8px 是有意例外，需在 notes 标注并全站统一。

### 4. [completeness] 体温输入丢失源码约束：源码是 type="digit" maxlength="4"，原型是 type="text" inputmode="decimal" 且没有 maxlength；非数字只在提交时 toast 拦截，长串可以输入；notes 只记了备注 200 字，没有记录这处差异。

- 证据：teacher/src/pages/doctor/checkin.vue:279 type="digit" maxlength="4"；app.js:251 type="text" inputmode="decimal" 无 maxlength；app.js:433-437 只在提交时校验非数字；notes.md:51 只提备注 maxlength。
- 建议：补 maxlength="4" 与输入过滤（或等价的 digit 约束），并在 notes 的差异/遗留里注明。

### 5. [completeness] 源码的「家长通知失败」分支不可达：源码在 notify_failed_count > 0 时弹模态提示，原型把 notify 写死为 {notified_count:1, notify_failed_count:0}，失败文案永远不出现，notes 也没有把它列为未覆盖分支。

- 证据：teacher/src/pages/doctor/checkin.vue:154-160 通知失败 showModal；app.js:499-503 notify 写死、失败分支不可达；notes.md:24 只写成功 toast。
- 建议：增加一次可触发的演示（例如首次提交成功但通知失败，或单独演示态），或至少在 notes 标为未覆盖的源码分支；不要只留不可达代码。

## 2026-10-27 P1 修复记录

- **P1-1 [logic] 空态「回看已完成」自相矛盾 → 已修**。`emptyRows()` 改为从 `records` 派生（不再读静态 SEED），`renderStats()` 空态计数与新增的 `renderEmptyCopy()` 空态文案数字（18 / 3）都从 records 计算；删掉 `emptyReviewed` / `PENDING_SEED` / `wasPending` 与「补齐 / 置顶」toast，CTA 改为「查看已完成」且只滚动定位，`.is-fresh`、`.fresh-tag` 样式随之移除（已无标记可打）。
- **P1-2 [redundancy] 园级事件入口被埋到页底 → 已修**。`#eventEntry` 从 `completedSection` 之后移回今日卡与班级筛选之间，恢复源码 `garden-entry` 的次序；`.filter-wrap` 上边距 `6px → var(--space-lg)`，避免筛选胶囊贴住事件卡；notes 信息层级同步把入口列为第 2 项。
- **P1-3 [design-core] 关键文字对比度不足 → 暂缓（批次级）**。主按钮白字压品牌橙渐变、语义色压同色浅底都来自 `tokens.css` 的全局 token；审查建议与 `_shared/REVIEW.md` §6.4 均为「回 `tokens.json` 一次修」，本页不能手改生成产物、也不能改 `_shared/` 或其它页，逐页改色会造成批次内不一致。已在 notes「遗留」登记继承该问题。

**验证**：`od-check` `problems: []`；运行时正常态 18 / 5 / 13、已完成 13 行，空态 18 / 0 / 18、已完成 18 行且点击 CTA 前后行数/顺序/标记不变、无 toast；提交一人（劝返）后切空态仍为 18 行、该生显示「劝返」且需留意数变 4（结论不被重置）；`#eventEntry` 四态 top≈210（视口 844，原 1344/1394）；`od-shot` 已重出 `preview.png`（390×1499）；批次 `od-audit` 50/50 通过，本页仅剩既有 app.js 25KB 体积提示。

