# 星空蓝 · Starry Blue（2026-10-27）

第五套配色方案面向客户评审：客户认为蓝色更符合「星球」概念。上一版「星海蓝」走的是白天浅蓝；这一版把主色压到**深空海军蓝**，用夜里的天空讲同一个星球。

- 主题名：`starry`
- 预览：任意页面默认深空蓝；`?theme=celestial` 切到星海蓝，`?theme=bright` 切回原亮橙
- 视觉说明：`_shared/palette.html`
- 色值入口：`shared-design-tokens/themes.json` → `starry`
- 生成：`node scripts/build-theme-css.mjs`

## 色板

| 角色 | 老师端 | 家长端 |
|---|---|---|
| 页面底 | `#F8F5EE` 暖米白 | `#F1F6FD` 冷雾白 |
| 卡片 | `#FFFFFF` | `#FFFFFF` |
| 主色（文字/图标/填充） | `#1C3F73` 夜空蓝 | `#1F4E8C` 深空蓝 |
| 品牌渐变 | `#3568A4 → #1B3E6E` | `#3A70A9 → #1D4573` |
| 浅底 | `#D8E5F4` 月光蓝 | `#DAE8F8` 冷月蓝 |
| 深色文字 | `#0B1B33` | `#0B1B33` |
| 渐变上的文字 | `#FFFFFF` | `#FFFFFF` |
| 星砂金（奖章/提醒） | `#B45309` / `#FEF3C7` | 同左 |

## 为什么这次用白字压深蓝

亮橙原稿的「白字压亮橙」只有 2.35–2.80:1，星海蓝改用「深墨字压浅蓝渐变」。星空蓝把渐变整体压深，文字翻成白色：

- 老师端渐变最浅端 `#3568A4` 上的白字：**5.73:1**
- 家长端渐变最浅端 `#3A70A9` 上的白字：**5.16:1**
- 老师端主按钮 `#1C3F73` 上的白字：**10.46:1**
- 夜空蓝字压月光蓝浅底：**8.18:1**

深墨字只留在浅色纸面与浅蓝浅底上。两种路径都过 WCAG AA。

## 三条用色规则

1. **行动与状态用夜空蓝**：主按钮、发布、提交、进度、选中态。
2. **奖章与提醒用金**：小小艺术家、Lv.5 小星星、安排有变更；金色面积始终小于蓝。
3. **成长与结果用绿**：已同步、已通过、积分增加。语义色不参与品牌色统一。

## 相对 10-25 版改了什么

- 颜色 token：仅通过 `themes.json` 的 `starry` 覆盖，不改页面布局与文案。
- 主题级修正（`_shared/starry.css`，用 `[data-theme="starry"]` 限定，不影响 `?theme=bright` / `celestial`）：
  - 奖章条、等级/徽章、加分：引入星砂金与成功绿。
  - 11 处「深色字压强调色填充」改为白字：`submit-btn`、`task-btn--primary`、`state-action`、`primary-button`、`btn-primary`、`publish-btn`、`btn-gradient`，以及老师端筛选计数。
  - 两端同名 `.btn-primary` 的禁用态契约不同：老师端禁用后换成浅底（排除），家长端禁用后仍保留渐变（保留白字）。
- 页面级修正：`parent/24-feedback` 主按钮沿用 `--on-accent`；星空蓝下为白字，原亮橙下仍是白字，不改变原稿。
- `_shared/theme.js` 默认保持 `starry`；加 `?theme=celestial` 可对照星海蓝，`?theme=bright` 对照原亮橙。

## 复验

```bash
node scripts/build-theme-css.mjs
node scripts/od-shots.mjs docs/design-previews/2026-10-27-starry-blue --width 390 --dsf 2
node scripts/od-audit.mjs docs/design-previews/2026-10-27-starry-blue
```

单页布局检查：

```bash
node scripts/od-check.mjs docs/design-previews/2026-10-27-starry-blue/parent/01-home/index.html --width 390
```
