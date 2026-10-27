# 授权接送人（parent）

- 源码：`parent/src/pages/pickup/authorized.vue`（`onShow` 重拉；`GET /parent/pickup/authorized-persons` + `GET /parent/pickup/designation` 并行；指定走 `POST /parent/pickup/designate`，删除走 `DELETE /parent/pickup/authorized-persons/:id`；`goAdd` → `/pages/pickup/authorized-create`）
- 设计系统：`user:nurture-planet-local`，surface=parent
- 生成方式：OpenDesign（首跑成功，542s；项目 `b76ee413` / run `ab7f7732`，三件套齐全，未走手写兜底。脚本以「产物稳定」推断终态，daemon 状态未落定，交付目录与项目终态已逐字节核对一致）
- 方向：**「可信名单」**——不靠装饰制造安全感，而靠清晰的分层：一条「今日接走人」全局状态 + 每张卡的授权类型/审核/生效状态三轨可读；橙色只标记「可操作」和「被选中」，面积 ≤ 10%。

## 信息层级

1. 微信外壳：20px 状态栏 + 44px 导航栏（**返回箭头** + 标题「授权接送人」+ 胶囊）；**无 tabBar**（`navigateTo` 子页）
2. 安全提示白卡：30px `--warning-wash` 盾牌 + 「安全提示 / 授权接送人信息仅供门卫核验身份，请确保信息真实有效。」
3. 工具行：左「5 位授权接送人」；右 44px「＋ 添加」渐变胶囊
4. 今日接走人横幅：已指定 `--success-wash`（kicker「今日接走人」+ 姓名）；未指定 `--warning-wash`（「今日尚未指定接走人 / 指定后才能获取接送二维码」）
5. 人员卡 ×5：40px 姓名首字头像 → 姓名 + 授权类型徽标 +（今日接走人 / 待审核 / 已撤销）→「关系 · 脱敏手机号」→ 权限标签（仅生效授权显示）→ 1px 分隔线下的「设为今日接走人」（44px 描边）+「删除」（文字）
6. 页脚说明：「新添加的接送人需园所审核通过，才能指定为今日接走人。」

## 覆盖状态

| 状态 | 表现 |
| --- | --- |
| 正常（默认） | 5 张卡：王建国（今日接走人，橙描边）/ 李秀兰（长期）/ 陈梅（临时）/ 张建军（待审核，指定按钮降级，无权限标签）/ 刘芳（已撤销，指定按钮降级，无权限标签）；横幅显示「今日接走人：王建国」 |
| 空态 | 隐藏工具行、横幅、列表、页脚；56px `--bg` 圆 + 线框「添加联系人」+「还没有授权接送人」+「添加家人后，才能指定今日接走人并获取接送二维码」+ 主按钮「添加接送人」 |
| 加载 | 横幅与列表换 3 张骨架卡（头像圆 + 名称/元信息/标签/操作四层呼吸条）；计数换胶囊骨架；添加按钮保留可用；页脚保留；骨架在 `prefers-reduced-motion` 下停止呼吸 |
| 失败 | 隐藏横幅、列表、页脚；`--danger-wash` 圆 + 感叹号 +「授权接送人加载失败」+「网络不太稳定，请稍后再试」+「重试」（点击禁用并变「重试中…」1s，回正常 + toast） |
| 禁用 | 待审核 / 已撤销人员的「设为今日接走人」改 `--bg` 底、`--hint` 字、无描边；`aria-disabled="true"`，**点按仍会给原因 toast**（源码行为：`该接送人尚未通过园所审核` / `该接送人授权已撤销`）；两张卡都不渲染权限标签（权限只在生效授权上展示）；删除始终可用 |
| 今日未指定 | 删除当前「今日接走人」后横幅自动切为 `--warning-wash` 未指定态；全部删完则自动进入空态并同步芯片 |
| 确认中 | 指定与删除都走微信风格居中对话框（标题 + 说明 + 取消/确定）；删除确认键 `--danger`；指定换人时文案明确「今日已生成的接送二维码将全部失效」 |

演示芯片 4 个：正常 / 空态 / 加载 / 失败（`normal|empty|loading|error`）。

## token 映射

| 元素 | Token |
| --- | --- |
| 页面底 / 卡片面 / 分隔线 | `--bg` / `--surface` / `--border` |
| 安全提示图标 / 标题 / 正文 | `--warning-wash`+`--warning` / `--fg` / `--fg-2`（正文不用 `--hint`，保证可读） |
| 工具行计数 / 添加按钮 | `--muted` / `--accent-gradient`+`--on-brand`+`--shadow-sm` |
| 已指定横幅 / 未指定横幅 | `--success-wash`+`--success` + `--fg` / `--warning-wash`+`--warning` + `--fg-2` |
| 卡片圆角 / 阴影 / 选中描边 | `--radius-card` / `--shadow-sm`（选中 `--shadow-md` + 1px `--accent` 内描边） |
| 头像 | `--bg`+`--fg-2`；被指定 `--accent-wash`+`--accent-deep` |
| 徽标 | 类型 `--bg`+`--fg-2`；今日接走人 `--accent`+`--on-brand`；待审核 `--warning-wash`+`--warning`；已撤销 `--bg`+`--muted`；已拒绝预留 `--danger-wash`+`--danger` |
| 权限标签 | 可接送 `--success-wash`+`--success`；看记录 `--info-wash`+`--info`；看成长 `--bg`+`--fg-2`（tokens 无紫色） |
| 主操作 / 删除 | `--accent` 描边 + `--accent-deep` 字 / `--muted` 文字 + 垃圾桶；确认键 `--danger` |
| 状态卡 / CTA | `--danger-wash`+`--danger`；`--accent-gradient`+`--on-brand` |
| 遮罩 / toast | `color-mix(in srgb, var(--sidebar) 44%, transparent)` / `--fg`+`--surface` |
| 间距 / 字号 / 动效 | `--space-*`、`--text-xs…xl`、`--weight-*`、`--motion-fast`、`--ease` |

## 与现状的差异

1. **页面内重复标题去掉**：源码导航栏已有「授权接送人」，内容区又写一遍 `page-title`；原型让导航栏承担标题，内容区改成「5 位授权接送人 + 添加」的工具行，纵向省 40px。
2. **安全提示从橙色底改为白卡 + 浅黄圆图标**：源码 `$color-brand-primary-bg` 整块铺橙、正文还用 warning 色，与其他卡片的视觉权重失衡；原型只保留一个 30px 图标块，正文 `--fg-2`。
3. **头图从空占位改姓名首字**：列表接口 `avatar` 普遍为空，源码会渲染空圆圈；原型用姓氏首字（王/李/陈/张/刘），被指定者染 `--accent-wash`。
4. **操作权重反转**：源码「设为今日接走人」与「删除」都是实心色块（橙 + 红），一张卡上两个重色按钮；原型把主操作降为描边胶囊、删除降为灰字 + 垃圾桶，危险色只在确认框出现一次。
5. **审核状态前置到卡片**：源码只有点了被禁用的指定按钮才 toast「尚未通过园所审核」；原型把「待审核 / 已撤销」做成卡片徽标，用户不用试探。已通过不加绿徽标，避免整屏绿色噪声。权限标签只表达**生效**权限：仅 `review_status=1 && status=1` 的卡片显示「可接送 / 看记录 / 看成长」，待审核与已撤销卡片不显示，避免「待审核」与绿色「可接送」同卡矛盾。
6. **「已撤销」可见**：源码 `status=0` 的人只是指定按钮变灰，卡片上看不出原因；原型加中性徽标。
7. **横幅双态分色**：源码两个分支共用 `.designated-banner`，未指定只是换成 warning 背景；原型把已指定/未指定做成两套结构（kicker + 姓名 vs 主句 + 次句），未指定态更有行动指向。
8. **三态从一行字升级为完整状态**：源码空态/加载/失败都只是居中一行文字（失败仅 toast）；原型给骨架卡、带 CTA 的空态、带重试的错误态。
9. **失败态不谎报数量**：列表拉取失败时工具行隐藏计数（保留添加入口），避免「5 位」与错误卡同时出现。

## 遗留

1. **「停用」在契约与源码里都不存在（需批次决策）**：`PUT /parent/pickup/authorized-persons/:id` 只接受 `name/phone/relation/avatar/auth_type/allow_*/expire_at`，没有 `status`；源码 UI 只有「删除」（DELETE），没有停用/恢复。原型按源码用「删除 + 居中确认」，**没有发明停用按钮**。若产品确实要「停用/启用」，需先改 `backend/api/api-parent/05-pickup.md` + `UpdateAuthorizedPersonInput` + service，再把卡片操作换成停用/启用开关。
2. **状态 0 的文案口径**：domain 注释与源码 toast 都写「已撤销」，本次调度要求写「已停用」。原型采用源码/domain 的「已撤销」，需产品统一（列表徽标、toast、审核说明三处必须一致）。
3. **「已拒绝」只有样式没有样例**：5 张卡已覆盖主要状态，`badge--rejected` 已就位；真实数据出现 `review_status=2` 时直接套用，无需改结构。
4. **编辑入口缺失**：契约有 PUT，但源码没有编辑入口，原型不发明；若要支持修改手机号/关系，得先补页面与入口。
5. **多孩子账号未分组**：列表 payload 带 `student_name` / `class_name`，源码不渲染；原型也没加。一个家长有多个孩子时名单会混在一起，是否按孩子分组需产品确认。
6. **头像仍是占位**：原型禁用外链图片，统一用姓名首字；落地时若有真实 `avatar` 应替换（保留首字兜底）。
7. **分页未体现**：接口支持 `page/page_size`（默认 20），源码只拉第一页；原型不画「加载更多」。
8. **静态页无法真正作废二维码**：换人确认文案已按源码写明「今日已生成的接送二维码将全部失效」，但原型只更新横幅与徽标。
9. **安全图标属视觉补充**：源码用 remixicon 字体字符（空字形），原型改为内联盾牌线稿，语义一致。

## 验收

- `node scripts/od-check.mjs …/15-pickup-authorized/index.html --width 390` → `problems: []`（`scrollWidth 390 = clientWidth 390`，`overflowing: []`，`bodyHeight 1186`；滚动 286px 后 `statusTop: 0 / navTop: 20`，吸顶正常，无 console error）
- `node scripts/od-shot.mjs … --width 390 --dsf 2` → `preview.png`（390 × 1186，209,721 bytes）
- `node scripts/od-audit.mjs docs/design-previews/2026-10-25-full-redesign` → 本页无 ✗（无裸色值、无 nowrap、无 `.mp-*` 重定义、已链接 `../../_shared/shell.css`、无 `_shared/` 副本、无 `.mcp.json`）
- 视觉复核（mcp-vision 读 `preview.png`）：状态栏/导航栏/返回箭头/胶囊齐全，无 tabBar；5 张卡文字完整无裁切、无横向滚动；卡片/标签/按钮全圆角；橙色只在添加按钮、今日接走人徽标、被指定卡描边与可点的指定按钮上；「原型演示」胶囊未遮挡内容
- 未修改 `teacher/` 与 `parent/` 源码；交付目录无 `.mcp.json`、无 `_shared/` 副本
