# 星海蓝 · Celestial Blue（2026-10-27）

第四套配色方案面向客户评审：把主色从亮橙切成 **logo 星球自带的暖蔚蓝**。不是换一套品牌，而是把 logo 里被橙色压住的那一面放回产品。

- 主题名：`celestial`
- 预览：任意页面默认蓝色；`?theme=bright` 切回原亮橙
- 视觉说明：`_shared/palette.html`
- 色值入口：`shared-design-tokens/themes.json` → `celestial`
- 生成：`node scripts/build-theme-css.mjs`

## 色板

| 角色 | 老师端 | 家长端 |
|---|---|---|
| 页面底 | `#F8F5EE` 暖米白 | `#F5FAFE` 冷雾白 |
| 卡片 | `#FFFFFF` | `#FFFFFF` |
| 主色（文字/图标/填充） | `#25609A` | `#2A6AA5` |
| 品牌渐变 | `#8FC4EF → #5B9BD5` | `#93C8F1 → #5FA2DE` |
| 浅底 | `#DCEBF8` | `#DDEBF8` |
| 深色文字 | `#12243D` | `#12243D` |
| 渐变上的文字 | `#0F2A4A` | `#0F2A4A` |
| 星砂金（奖章/提醒） | `#B45309` / `#FEF3C7` | 同左 |

## 为什么不用白字压蓝

亮橙原稿的「白字压亮橙」只有 2.35–2.80:1。星海蓝改成 **深墨字压浅蓝渐变**，最深端 `#12243D / #5B9BD5` 为 5.27:1；主按钮的白字版本（`#FFFFFF / #25609A`）为 6.52:1。两种都达 WCAG AA。

## 三条用色规则

1. **行动与状态用蓝**：主按钮、选中态、进度、关键图标。
2. **奖章与提醒用金**：小小艺术家、Lv.5 小星星、安排有变更；金色面积始终小于蓝。
3. **成长与结果用绿**：已同步、已通过、积分增加。语义色不参与品牌色统一。

## 相对 10-25 版改了什么

- 颜色 token：仅通过 `themes.json` 的 `celestial` 覆盖，不改页面布局与文案。
- 主题级修正（`_shared/celestial.css`，用 `[data-theme="celestial"]` 限定，不影响 `?theme=bright`）：
  - `parent/01-home` 奖章条、`parent/20-points` 等级/徽章/加分、`teacher/01-workbench` / `teacher/19-profile-index` 待审批 pill：引入星砂金与成功绿。
  - `parent/23-student-index` 主按钮、`teacher/09-tasks-index` 激活筛选计数：深蓝填充上改白字（原亮橙下深字是达标的）。
- 页面级修正：
  - `parent/24-feedback` 主按钮：白字改 `--on-accent`；星海蓝下是深墨字，原亮橙下仍是白字，不改变原稿。
- `_shared/theme.js` 默认保持 `celestial`；加 `?theme=bright` 可对照原亮橙。

## 复验

```bash
node scripts/build-theme-css.mjs
node scripts/od-shots.mjs docs/design-previews/2026-10-27-celestial-blue --width 390 --dsf 2
node scripts/od-audit.mjs docs/design-previews/2026-10-27-celestial-blue
```

单页布局检查：

```bash
node scripts/od-check.mjs docs/design-previews/2026-10-27-celestial-blue/parent/01-home/index.html --width 390
```
