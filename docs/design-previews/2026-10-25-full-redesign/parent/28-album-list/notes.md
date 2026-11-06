# 班级相册（家长端）

> 生成方式：从代码现状补齐（2026-11-06）

- 源码：`parent/src/pages/album/index.vue`
- 接口：`backend/api/api-parent/`（班级相册列表）
- 设计系统：`user:nurture-planet-local`，surface=parent

## 信息层级

1. 范围切换：我的孩子 / 全班。只有服务端明确下发 `classwide_enabled=true` 才出现；关闭时整块不渲染。
2. 列表头：全部相册 + 共 N 本。
3. 相册卡：封面拼图（1/2/3 张时对应 1/2/3 列）、标题、可见张数、日期、分类标签。
4. 分页：已展示 N 本 · 共 M 本 + 加载更多；到底后按钮置灰为「已经到底了」。
5. 外壳：返回「成长时光」，无 tabBar。

## 与代码一致的口径

- 计数用 `visible_photo_count`（家长可见张数），不是老师端的 `photo_count`。
- 封面张数决定拼图列数；没有封面时显示相册占位图标。
- 日期用 `M月D日`，与家长端其他页面一致。

## 覆盖状态

| 状态 | 表现 |
|---|---|
| 正常 | 3 本相册 + 加载更多；封面分别 3 / 2 / 1 张 |
| 空 | 「本班还没有已发布相册」+「刷新看看」 |
| 加载 | 3 张与卡片同形状的骨架，不跳动 |
| 失败 | 「班级相册加载失败」+「网络不太稳定，请稍后再试」+ 重试 |

四态由 `_shared/shell.js` 的演示 chip 切换；页面用 `[data-state]` 接管。

## Token 映射

- 背景 / 卡片：`--bg`、`--surface`、`--border`
- 文字：`--fg`、`--fg-2`、`--muted`
- 品牌橙：`--accent`、`--accent-deep`、`--accent-wash`、`--accent-gradient`、`--on-accent`
- 圆角：`--radius-card`、`--radius-pill`
- 阴影：`--shadow-sm`
- 间距 / 字号：`--space-xs…xl`、`--text-xs…xl`

## 已知差异

- 封面占位用 CSS 渐变 + 线性图标，真实图片由接口下发；原型不引入外网资源。
- 代码里的 `PageShell` 会处理安全区与最大宽度，原型由 `_shared/shell.css` 提供等价外壳。
