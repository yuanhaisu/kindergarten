# 请假申请（parent）

- 源码：`parent/src/pages/leave/create.vue`（`createLeave` → `POST /parent/leaves`；提交字段 `student_id / reason / start_date / end_date / attachments / emergency_contact`）
- 设计系统：`user:nurture-planet-local`，surface=parent
- 生成方式：OpenDesign（首次 run 因 API 402 余额中断，仅产出 `index.html`；本次 run 392s，daemon 最终 `succeeded`、三件套齐全。包装脚本在 `index.html` 稳定 30s 时提前推断完成并退出，`styles.css` / `app.js` 由 run 真正结束后手工拷入，见 `_run.json.note`）
- 方向：**「一张安静的表单，所有规则都提前说清楚」**——冷雾白底、白色软卡分组，橙色只落在选中的病假卡、日期值胶囊和底部提交按钮；必填与阻断在按钮上方常驻，不靠事后 toast。

## 信息层级

1. 微信外壳：20px 状态栏 + 44px 导航栏（返回箭头，标题「请假申请」，右侧胶囊），本页无 tabBar
2. 孩子上下文条：40px 首字头像「乐」+ `为 乐乐 · 小一班`，只读、不可切换
3. 请假类型：三张等宽卡（病假 / 事假 / 其他），选中 = 品牌色描边 + `--accent-wash` + 右上角对勾
4. 请假时间：开始 / 结束两行，右侧日期胶囊 + 星期 + `›`；底部一行 `共 2 天 · 含首尾两天`
5. 请假原因：多行输入 + 右下角实时 `44/500`；接送人姓名 / 手机号两张堆叠输入；附件 4 列方格（已选缩略 + 添加照片）
6. 底部固定提交栏：常驻说明行（阻断文案或「共 N 天 · 提交后进入待审批」）+ 48px 主按钮，安全区兜底

## 覆盖状态

| 状态 | 表现 |
| --- | --- |
| 正常（默认） | 病假选中，开始 10-28 / 结束 10-29，`共 2 天`，原因 44 字，接送人王芳 / 13812345678，附件 1 张「病假条」；常驻行「共 2 天 · 提交后进入待审批」，主按钮品牌渐变可点 |
| 必填未填 | 类型未选，两个日期「请选择」，原因空；三处就地红字（请选择开始日期 / 请选择结束日期 / 请填写请假原因）；常驻行红色「还需填写：开始日期、结束日期、请假原因」；按钮灰底 + `aria-disabled="true"`，点击不提交、不弹 toast |
| 加载 | 表单整体替换为 5 段骨架（类型三格 / 日期两行 / 原因多行 / 接送人两行 / 附件方格，`sk-pulse` 呼吸）；常驻行「正在加载…」，按钮禁用 |
| 提交中 | 表单 `opacity:.62` + `pointer-events:none`，所有控件 `disabled`；按钮保持品牌渐变 + 小转圈 +「提交中…」；常驻行「正在提交，请稍候」 |
| 失败 | 表单保持完整内容；提交栏上方出现 `--danger-wash` 失败条「提交失败，请重试」+ 44px「重试」按钮；点重试收起失败条、toast「已重新提交」、恢复可提交 |
| 禁用 | ① 未填齐：按钮灰底 + `aria-disabled`，点击只做就地校验；② 加载 / 提交中：类型卡、日期行、原因、接送人、附件删除与添加全部 `disabled`；③ 日期弹层「确定」在未选日期时禁用；④ 结束日期弹层里早于开始日期（或早于今天）的日期置灰 |

演示芯片：正常 / 必填未填 / 加载 / 提交中 / 失败（`data-mp-state` = normal / empty / loading / submitting / failed），五态都走真实 `app.js` 分支，不是静态截图。

## token 映射

| 元素 | Token |
| --- | --- |
| 页面底 / 卡片面 | `--bg` / `--surface` |
| 分组标题 / 说明 / 字数 | `--muted`、`--text-sm`、`--text-xs`（未用对比度不足的 `--hint` 作正文） |
| 孩子头像 / 选中类型卡 / 日期胶囊 | `--accent-wash` + `--accent-deep` / `--accent` 描边 |
| 卡片 / 输入框 / 弹层圆角 | `--radius-card` / `--radius-xl` / `--radius-2xl`（顶部） |
| 卡片阴影 / 底栏阴影 / toast | `--shadow-sm` / `--shadow-lg` |
| 主按钮 / 选中对勾 / 日期弹层确定 | `--accent-gradient` / `--accent-deep` + `--on-brand` |
| 禁用按钮 | `--surface-warm` 底 + `--border` 边 + `--muted` 字 |
| 就地错误 / 失败条 | `--danger` / `--danger-wash` |
| 天数汇总、日期、星期 | `--fg` + `--weight-semibold`、`--muted` + `tabular-nums` |
| 骨架 | `--border` + `sk-pulse 1.6s var(--ease)` |
| 间距 / 字号 / 动效 | `--space-*`、`--text-*`、`--weight-*`、`--page-pad-x`、`--motion-fast`、`--ease` |

## 与现状的差异

- **提交按钮从表单末尾改为底部固定栏**：源码按钮在最后一张卡片下、随页面滚走；原型固定吸底，并在按钮上方常驻一行说明，安全区 `calc(12px + env(safe-area-inset-bottom))`，内容区留 126px / 失败态 190px 防遮挡。
- **校验从 toast 前移到界面**：源码只有 `uni.showToast("请填写完整信息")`；原型三处就地红字 + 常驻阻断文案 + 灰色按钮，填齐即恢复，点击禁用态不再弹任何提示。
- **日期从原生 picker 改为底部月历弹层**：源码 `mode="date"` 是系统滚轮；原型给 7 列月历、今天之后可选、结束日期不早于开始日期、开始日期后移时结束日期自动跟随（与源码 `onDateChange` 规则一致），选中后写回行内并重算天数。
- **请假类型选中态加强**：源码只有浅底 + 描边；原型加右上角对勾、品牌色描边与 `aria-pressed`，仍是单选。
- **新增孩子上下文条**：源码只把 `getCurrentStudentId()` 塞进 payload，页面不显示请假对象；原型补一行 `为 乐乐 · 小一班` 的只读上下文（见遗留 3）。
- **新增附件上传区**：源码 form 里有 `attachments: []`，但模板没有任何上传入口；原型补「+ 添加照片」方格与最多 3 张的缩略管理（API 契约支持 `attachments`，见遗留 2）。
- **失败态从一次性 toast 改为持久失败条**：源码 catch 里只弹 toast 并保留表单；原型在提交栏上方挂常驻失败条 + 重试，表单内容不丢。
- **加载态补骨架屏**：源码没有 loading 分支；原型给 5 段骨架，避免表单白屏。
- **提交成功行为改为表单复位**：源码 toast「提交成功」后 1s `navigateBack()`；原型按 brief 留在本页并复位，便于连续审阅（见遗留 4）。

## 遗留

1. **「请假类型」仍不进 payload，需产品确认**：源码 `activeType` 只做前端视觉，`createLeave` 从未携带；`backend/api/api-parent/03-leave.md` 的创建请求也没有类型字段。原型保留三卡（本页主要选择动作），但不加必填星号、不参与阻断。要落地须先补契约与提交字段。
2. **附件 UI 源码不存在，需产品确认**：`form.attachments` 与 API 请求/详情都支持附件数组，但模板没有 chooseImage 入口。原型按批次要求补了上传方格；落地需补 `uni.chooseImage`、上传与回填逻辑（当前只是原型占位，不接真实文件）。
3. **孩子姓名 / 班级本页拿不到**：`getCurrentStudentId()` 只存 id；要显示「乐乐 · 小一班」需页面额外拉 overview 或把当前孩子上下文传进来。后端还会校验孩子归属，不属于该家长直接 403。
4. **提交成功后的去向**：源码 `navigateBack()` 回到请假记录；原型复位表单留在本页。真实实现建议保留 `navigateBack()`，否则用户会看到一个已清空但仍在原地的表单。
5. **失败态目前只能由演示芯片触发**：原型没有「模拟失败」开关，真实提交永远成功。若要审阅失败路径，用演示芯片切到「失败」；落地时把 catch 分支映射到这条持久失败条即可。
6. **天数口径**：后端详情返回 `days`（含首尾），原型用前端日期差 +1 复算，二者在跨月/跨年时也一致；真实实现应以后端返回为准，避免时区差异。
7. **未做的**：切换孩子、请假类型自定义、草稿箱、审批人选择、剩余假期、撤销/撤回、上传视频——源码均无，原型也不发明。

## 验收

- `node scripts/od-check.mjs …/07-leave-create/index.html --width 390` → `problems: []`（`scrollWidth 390 = clientWidth 390`，`overflowing: []`，`bodyHeight 1195`；滚动后 `statusTop: 0 / navTop: 20`，吸顶正常，无 console error）
- `node scripts/od-shot.mjs … --width 390 --dsf 2` → `preview.png`（390 × 1195，约 196 KB）
- `node scripts/od-audit.mjs docs/design-previews/2026-10-25-full-redesign` → 本页无 ✗（无裸色值、无 nowrap、无 `.mp-*` 重定义、已链接 `../../_shared/shell.css`、无 `_shared/` 副本、无 `.mcp.json`）；仅 `styles.css 25KB / app.js 21KB` 体积提示
- 视觉复核（mcp-vision 读 `preview.png`）：状态栏 / 导航栏 / 返回箭头 / 胶囊齐全，无 tabBar；卡片与按钮全圆角；文字无裁切；橙色只出现在选中病假卡、日期胶囊与提交按钮；演示条默认收起为「原型演示」小胶囊
- 交互复核（Chrome CDP）：五态芯片齐全且真实切换；类型单选、日期弹层（今天前置灰 / 结束日期下限 / 改开始日期自动跟随）、字数统计、附件增删（最多 3 张）、空表单提交就地报错、提交成功 toast + 复位、失败条重试全部通过；console 无报错
