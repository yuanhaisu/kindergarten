# 接送记录（老师端）

- 源码：`teacher/src/pages/pickup/records.vue`
- 设计系统：`user:nurture-planet-local`，`data-surface="teacher"`
- 外壳：`_shared/shell.css` + `shell.js`（外链引用，未复制、未改 SVG）
- 方向：园门口的接送流水账——一天一张表，时间倒序，谁接走、怎么核验、哪条有例外，一眼扫完。

## 信息层级

1. 日期卡：左箭头 / 中间日期（点击开底部日期弹层）/ 右箭头；非今天显示「今天」胶囊，右箭头到今天即禁用
2. 统计条：记录 / 接走 / 入园签到，单位为「条」（一条 = 一次接送事件）
3. 组头：「10月26日 周一 · 共 13 条」，右侧班级名
4. 时间轴：左侧时间列、竖向轨道、行内学生 + 接送人（含关系）、核验方式轻量标签、例外/异常标签；理由与异常备注完整换行
5. 行详情弹层：接送类型、记录时间、接送人、核验方式、授权状态、异常备注、理由
6. 空 / 加载 / 失败三态

## 状态覆盖

- 正常：10/26 当天 13 条（4 接走 / 9 入园签到），含 1 条提前接走异常、1 条授权例外
- 空：10/25 周日，统计 0 条，空态文案 +「回到今天」；10/24 周六同样为空
- 加载：统计与列表换骨架，列表 4 行占位
- 失败：整页错误 + 重试，统计条三项改为「—」占位（不留可能过期的条数，避免与「没取到」文案自相矛盾）
- 局部：日期弹层（最近 7 天，每天标条数 / 无记录）、行详情弹层（异常 / 例外 / 入园签到三类内容）、返回与选择日期的 toast

## token 映射

- 页面 / 卡片：`var(--bg)` / `var(--surface)` / `var(--radius-card)` / `var(--shadow-sm)`；分隔线 `var(--border)`
- 主色：选中日期与「今天」用 `var(--accent)`，主按钮 `var(--accent-gradient)`
- 核验方式：扫码 `var(--info)` + `var(--info-wash)`；手动走中性描边（`var(--border)` + `var(--muted)`）
- 例外：`var(--warning)` + `var(--warning-wash)`；异常：`var(--danger)` + `var(--danger-wash)`；入园签到 `var(--success)`
- 文本：`var(--fg)` / `var(--fg-2)` / `var(--muted)` / `var(--hint)`；数字用 `font-variant-numeric: tabular-nums`

## 与现状的差异

- 日期：源码是「◂ / ▸ + 一个不可点的 ▾」；改为「左箭头 / 可点日期（底部弹层）/ 右箭头」。弹层列最近 7 个连续日期（10/20–10/26）及每天条数，周末显示「无记录」，未来日不可选
- 移除「筛选」按钮：源码 `openFilter()` 只弹「筛选功能开发中」，没有对应参数。同时**没有**新增「只看异常」chip——`GET /teacher/pickup/records` 的 Query 只有 `date` / `student_id` / `page` / `page_size`，无异常过滤
- 统计单位「人」→「条」：同一学生一天可多次进出（入园 + 接走），按人计数会与记录条数对不上
- 列表：源码是平铺卡片，每卡重复头像与状态徽章；改为日期组头 + 时间轴单列，密度更高、扫描更快
- 核验方式：源码 `method_color` 硬编码蓝 / 紫（`#9333EA` 不在 token 内）；改为扫码 = info 浅底、手动 = 中性描边
- 异常：源码只有白字红底的「异常」小标签，且不展示 `abnormal_reason`；改为 danger 浅底标签 + 行内异常备注 + 详情弹层
- 理由：源码 `white-space: nowrap` + 省略号截断；改为完整换行
- 新增行详情：源码没有详情入口；弹层只用列表字段与 `GET /teacher/pickup/records/:id` 已有语义，未展示 `pickup_phone` / `operator`（列表契约不返回，原型也不发详情请求）
- 空 / 加载 / 失败：源码分别是一行文字、一行「加载中...」、toast；改为完整三态，失败态可重试
- 未加底部导出 / 统计入口：源码没有

## 遗留

- 数据为原型内固定数据：10/26 当天 13 条，10/20–10/23 四个工作日各 20 条；「加载」是本地 380ms 定时，不接真实接口
- 返回按钮与日期选择只做 toast / 本地状态，不触发真实路由与请求
- 详情弹层未接 `GET /teacher/pickup/records/:id`，因此没有核验人（`operator`）与接送人手机号（`pickup_phone`）
- 为修 `od-check` 的横向溢出，在 `styles.css` 顶部补了三条基础规则：全局 `box-sizing: border-box`、`html, body` 去默认 margin、`[hidden] { display: none !important }`；不改 token
- `styles.css` 约 23KB、`app.js` 约 21KB，超过 20KB 建议值（审计仅提示，不阻断）
