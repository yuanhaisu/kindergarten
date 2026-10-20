# OpenDesign → Impeccable 交接卡

## 0. 元信息

- 日期：2026-10-19（两模式版本追加于 2026-10-20）
- 目标页面 / 路由：`parent/src/pages/growth/index.vue`（家长端「成长时光」列表）
- 端：`parent`
- OpenDesign 项目（初始方向）：`Pilot · Parent Growth Direction`（project id `2839f692-c94a-408c-b802-8bd449a81e91`）
- OpenDesign 项目（两模式）：`Parent Growth - Two Modes - Split Files`（project id `efe7dad8-168c-4981-a2f1-4d988cf72caa`）
- 设计系统：`user:nurture-planet`（适配层，源 `shared-design-tokens/tokens.json`）；两模式稿使用本地副本 `user:nurture-planet-local`
- 方向名：收敛版 —— 压缩首屏橙色面积，强化时间线层级；追加「只看孩子 / 看全部」两种查看方式
- 状态：approved（visual scope）

## 1. 目标与成功标准

- 用户是谁：家长
- 使用场景：深夜或通勤时查看孩子在园的成长记录
- 成功后用户能做什么：一屏内看清「今天发生了什么」，并能快速找到历史记录
- 失败定义：视觉更花哨但没有提升可读性；或为了视觉改动破坏现有数据流与行为

## 2. 方向选择

- 选定方向：收敛版 —— 保留 Tender Cosmos 的暖橙渐变与胶囊语言，但把 hero 卡压缩到更紧凑的比例，把视觉重心让给时间线内容
- 为什么选它：现有 critique 的 P0 是首屏橙色渐变占约 36%（超出 10% 预算）；先把品牌表达收敛到 tokens 契约，再谈方向探索
- 放弃了哪些方向：A/B 两套探索方向（初次运行因 `deepseek-flash[1m]` 推理过度未产出；改用 `deepseek-chat` + 干净 CLI 配置后，单方向产物已成功生成）
- 两模式追加：在收敛版基础上增加「只看孩子 / 看全部」分段控件；默认只看孩子，看全部包含班级其他孩子的动态；具体见 3.1

## 3. 视觉契约

- surface：`parent`（`<html data-surface="parent">`）
- token 映射：

  | OpenDesign | kindergarten token | 当前值 |
  |---|---|---|
  | `var(--bg)` | `parent.color.bg.page` | `#F8FAFC` |
  | `var(--surface)` | `parent.color.bg.card` | `#FFFFFF` |
  | `var(--accent)` | `parent.color.brand.primary` | `#F97316` |
  | `var(--fg)` | `shared.color.text.primary` | `#1E293B` |
  | `var(--fg-2)` | `shared.color.text.body` | `#334155` |
  | `var(--radius-card)` | `shared.component.card.radius` | `12px` |

- 组件行为：孩子信息卡、月份分组、时间线行、底部导航；不改交互逻辑
- 动效：只用于列表进入与点赞反馈，尊重 `prefers-reduced-motion`
- 状态覆盖：加载 / 空 / 错误必须有可区分的视觉形态；本试点不改错误处理的业务逻辑
- 响应式断点：390px 手机优先；768px 平板；1200px 桌面（如适用）

## 3.1 两种查看方式（2026-10-20 追加）

- 默认「只看孩子」：只显示与当前孩子相关的成长内容；孩子信息卡下方放「只看孩子 / 看全部」分段控件，显示两边的条数与照片数。
- 照片过滤：按 `media[].student_ids.includes(当前孩子ID)` 过滤；孩子专属记录（详情 `target_type=2`）不按照片点名过滤，整条记录的照片都属于孩子，标「专属记录」。
- 「看全部」：显示班级全部记录和照片，包括其他孩子；其他孩子标「其他孩子」/「班级活动」，集体照片标「集体」；默认模式只显示含当前孩子的集体照片。
- 集体照片：两种模式都可解释；默认模式显示含当前孩子的照片，未点名时提示「还有 N 张班级照片，切到「看全部」查看」。
- 空态：「只看孩子」无匹配时显示「还没有小明的专属记录」+「看全部班级动态」；「看全部」无内容时显示「班级还没有发布成长记录」；加载态骨架屏，错误态重试。
- 隐私：只显示已确认点名；其他孩子用「小朋友 A」等匿名标识，不暴露真实姓名；「导出相册」仍只导出当前孩子的相册。
- 实现依赖：当前列表契约只保证 `media[].student_ids`；`target_type` / `related_students` 只在详情接口。实现两模式时需要在列表补一个最小的 `scope` 字段（`individual` / `class`），否则客户端无法区分个体记录与全班记录；不要把 `target_type` 当作列表已返回字段。

## 4. 参考物

- OpenDesign artifact（单方向历史稿）：`open-design/.od/projects/370dba5d-1e97-47ca-a463-fde621fa33e7/index.html`（Retest4，39,454 bytes；截图 `C:\Users\33139\Downloads\retest4-parent-growth-2026-10-19T13-06-30-210Z.png`）
- OpenDesign artifact（两模式稿）：`kindergarten/docs/design-previews/2026-10-20-parent-growth-two-modes/index.html`（源项目 `open-design/.od/projects/efe7dad8-168c-4981-a2f1-4d988cf72caa/`；含 `styles.css` / `app.js` / `tokens.css`；截图 `preview-child.png` / `preview-all.png`）
- `components.html`：`open-design/.od/design-systems/nurture-planet/components.html`
- 只取意图、不取像素：紧凑 hero、琥珀阴影、胶囊按钮与时间线行比例；不逐像素复制 fixture

## 5. 非目标 / 不可变项

- 产品事实：成长记录数据来自 `fetchGrowthFeed`，点赞走 `likeGrowth` / `unlikeGrowth`；不改接口
- 文案：现有中文文案与信息结构保留；可改进层级，不新增无依据的营销文案
- 数据与接口：不改 `parent/src/api/*` 与后端
- 现有流程与导航：保留 `BottomNav`、详情页跳转、相册与报告入口
- 明确不动的文件：`parent/src/api/*`、`server/*`、tabBar 配置

## 6. 验收标准

- 视觉层级：首屏能区分「孩子信息 / 今日记录 / 历史记录」；橙色渐变不超过首屏 10%
- 两模式：默认「只看孩子」；切换「看全部」后记录/照片/计数/归属标识同步；看全部包含其他孩子与集体内容；「导出相册」仍只针对当前孩子
- 可访问性：正文对比度 WCAG AA；焦点可见；交互目标 ≥ 44px
- 响应式：390px 无横向溢出；文本不截断；时间线不重叠
- 性能：不新增重型依赖；动效不阻塞首屏
- 需要的证据：桌面与移动截图、`pnpm check-tokens:parent` 输出、改动文件列表

## 7. 实施记录（Impeccable 填）

- 使用的命令：`impeccable context --target parent/src/pages/growth/index.vue`；`node scripts/check-tokens.mjs parent`；`npm run build:mp-weixin`
- 改动文件：`parent/src/pages/growth/index.vue`（本次仅改 `<style>`；该文件工作区此前已有未提交的行为层改动，diff 会同时显示）
- 关键决策：
  - hero 卡改用 `$gradient-brand`，内边距 `$component-card-padding`，头像 160rpx → 120rpx，姓名 48rpx → 40rpx，统计数字 40rpx → 36rpx
  - 卡片阴影从硬编码黑阴影改为 `$shadow-sm`，列表卡改用 `$component-card-radius` / `$component-card-padding` / `$component-card-gap`
  - 删除本地 `$orange-100`、`$slate-200`、`$slate-600`，改用语义 token；渐变上的半透明白叠加保留并注明为装饰例外
  - 时间戳与互动文案从 `$color-text-hint` 提升到 `$color-text-muted`，改善可读性
  - 补充卡片与互动项的 `:active` 反馈，使用 `$animation-duration-fast`
- 证据：`check-tokens` 在该文件 0 违规；`npm run build:mp-weixin` 构建通过（exit 0，`DONE Build complete.`，仅有 Sass @import 弃用警告）
- 遗留问题与后续：首屏筛选/年份硬编码/错误态业务逻辑属于行为层，未纳入本次视觉范围

### 7.1 两模式 UI 实施（2026-10-20）

- 改动文件：`parent/src/pages/growth/index.vue`
- 查看方式：新增 `viewMode`（默认 `child`）与「只看孩子 / 看全部」分段控件，显示两边的记录/照片计数；`visibleFeeds` 驱动列表与月份分组。
- 照片过滤：`mediaFor(item)` 对 `scope=individual` 返回全部照片，不按照片点名过滤；`scope=class` 按 `media[].student_ids.includes(currentStudentId)` 过滤；「看全部」不过滤。班级记录中有照片被隐藏时显示「还有 N 张班级照片，切到「看全部」查看」。
- 归属标识：`ownerKind/ownerLabel` 对个体记录输出「专属记录」，对全班记录输出「含孩子 / 集体 · 含孩子 / 其他孩子 / 班级记录」。
- 空态：「只看孩子」过滤后为空时显示模式空态 +「看全部班级动态」；原始空列表仍用 `StateBlock`；加载/错误态保持原逻辑。
- 导航：保留原生导航栏，在 `loadData` 里用 `uni.setNavigationBarTitle` 动态设置「{孩子名}的成长时光」；未引入 `navigationStyle: custom`。
- 契约补齐：OpenSpec `add-parent-growth-feed-scope` 为列表新增 `scope`（`individual` / `class`）。个体记录在「只看孩子」展示全部照片并标「专属记录」；全班记录继续按 `media[].student_ids` 过滤。其他孩子的个体记录仍不可见。
- 验证（代码层）：`go build ./...`、`go vet ./...`、`go test ./internal/service/ -run 'TestFeedScope|TestAssembleFeedItems'` 通过；`npm run build:mp-weixin` 通过；`check-tokens` 针对 `growth/index.vue` 无新增违规（仓库整体仍有其他文件的历史违规 161 处）；`openspec validate --all` 通过。
- 测试环境实测（2026-10-20）：真实 HTTP 返回 7 条，id 21/22/27/28/29 为 `class`、id 23/24 为 `individual`；id 23（个体、1 张未点名照片）在只看孩子可见并标「专属记录」；id 21（全班、2 张未点名照片）在只看孩子隐藏、看全部可见；只看孩子 6 条/1 张，看全部 7 条/3 张，TC-GRW-099/100/101/102/104 通过。
- 混合点名 fixture：log `991000522`（content 前缀 `R6| 混合点名验证`，media `991000522` / `991000523`，仅前者 `growth_media_students` 确认到学生 27）。真实鼠标点击（新 IDE 实例，屏幕坐标 `501,391`）切到只看孩子：`runtime-child.png` 显示仅 1 张照片并出现「还有 1 张班级照片，切到「看全部」查看」；`runtime-all.png` 显示 2 张照片。TC-GRW-103 通过。
- 证据截图：`docs/design-previews/2026-10-20-parent-growth-two-modes/runtime-child.png` / `runtime-all.png`。

## 8. 风险与回滚

- 主要风险：视觉方向与现有信息架构冲突；改动范围蔓延到行为逻辑
- 回滚方式：单个 `.vue` 文件的 git diff 回滚
- 需要同步更新的源：`shared-design-tokens/tokens.json`（如新增 token）/ `DESIGN.md`（如视觉世界改变）/ 本卡
- OpenDesign 侧遗留：初次失败根因是 `deepseek-flash[1m]` 推理型模型在 OpenDesign 设计规范下长时间思考、不调用工具；已通过 app config 将 Claude CLI 模型切换为 `deepseek-chat`、使用干净 `CLAUDE_CONFIG_DIR`，并在 `claude.ts` 传 `--effort low`。
- Opus 状态：AMR 的 `claude-opus-5.5` / `claude-opus-5` 在运行时均返回 `no longer available in AMR`，会回退到 `deepseek-v4.1-flash`；本地 Claude OAuth 也未登录。两模式稿实际由 `deepseek-chat` 生成，并拆成 `index.html` / `styles.css` / `app.js` 三个文件写入，避免单次大 `Write` 触发 600s 不活动超时。要真正用 Opus，需在 OpenDesign Settings → Models & providers 配置可用的 Anthropic 兼容 provider（base URL + key + model），或在会话里执行 `! claude login` 完成本地 Claude 登录；不要把 AMR 当作 Opus 入口。
- 设计系统说明：原 `user:nurture-planet` 被 workspace 绑定，headerless 列表不可见；两模式稿使用本地副本 `user:nurture-planet-local`（同源 tokens），生产仍以原设计系统为准。
