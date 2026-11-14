# 小王子主题 · 配色与插图

主题名：`petitprince`。色值入口 `shared-design-tokens/themes.json`；生成 `node scripts/build-theme-css.mjs`。

## 一句话

羊皮纸是纸，墨是字，朱砂是那一笔。页面顶部的故事带是这套主题的签名，不抢内容，只提示这一页在讲什么。

## 主色

| 角色 | 老师端 | 家长端 | 用途 |
|---|---|---|---|
| 页面底 | `#F6F0E3` | `#F7F1E6` | 羊皮纸 |
| 卡片 | `#FFFDF6` | `#FFFEFA` | 暖白 |
| 主色 | `#B33A2B` | `#A63A45` | 主按钮、未读、待办 |
| 渐变起点 | `#C9593F` | `#C25A64` | 品牌渐变 |
| 深色 | `#8C2A1E` | `#7E2833` | 浅底上的强调字 |
| 浅底 | `#F5E0D8` | `#F4DEE1` | 选中态、标签底 |
| 正文 | `#23201A` | 同左 | 墨 |
| 次级文字 | `#6B6354` | 同左 | 说明 |
| 星金 | `#8A5A0B` | 同左 | 奖章、等级、积分 |
| 星金浅底 | `#F6E9C8` | 同左 | 奖励底 |

## 对比度（WCAG 2.1）

| 组合 | 用途 | 对比度 |
|---|---|---|
| `#23201A` / `#FFFDF6` | 正文 / 暖白卡 | 15.96 : 1 |
| `#23201A` / `#F6F0E3` | 正文 / 羊皮纸 | 14.30 : 1 |
| `#FFFFFF` / `#B33A2B` | 白字 / 老师端主按钮 | 5.90 : 1 |
| `#FFFFFF` / `#A63A45` | 白字 / 家长端主按钮 | 6.34 : 1 |
| `#8C2A1E` / `#F5E0D8` | 深朱砂字 / 朱砂洗 | 6.71 : 1 |
| `#8A5A0B` / `#F6E9C8` | 星金字 / 星金浅底 | 4.91 : 1 |
| `#6B6354` / `#FFFDF6` | 次级文字 / 暖白卡 | 5.83 : 1 |

## 插图

源文件：`_shared/illustrations/`，共 21 个 SVG。

- **8 幅分镜**：`scene-01` 启程 / `scene-02` 玫瑰 / `scene-03` 驯养 / `scene-04` 来信 / `scene-05` 告别 / `scene-06` 值守 / `scene-07` 收好 / `scene-08` 星球。
- **12 条故事带**：`band-journey`、`band-candle`、`band-letter`、`band-growth`、`band-mirror`、`band-knot`、`band-kite`、`band-bell`、`band-planets`、`band-stars`、`band-lamp`、`band-constellation`。
- **1 张封面**：`hero-gallery.svg`。

规则（来自 `.claude/skills/petit-prince-story`）：

- line 画风：描边 `#141414`，纸底 `#fbfaf6`。
- 每幅分镜恰好一个朱砂焦点 `#C0392B`；金句 / 标题保持墨色。
- 星点 ≤12，小行星 ≤2；每位迎客角色在同一套分镜里只出现一次。
- 故事带是 390×84 的无客人版本；每条一个朱砂点，同模块子页面复用同一条。

## 页面集成

`_shared/apply-theme.mjs` 做三件事（幂等）：

1. `<html data-theme="starry">` → `data-theme="petitprince"`。
2. `_shared/starry.css` → `_shared/petit-prince.css`。
3. 在导航栏之后插入 `<figure class="pp-band">`，按 `surface/slug` 映射到 12 条故事带之一。

`?theme=bright` 会移除 `data-theme` 并隐藏故事带，回到 2026-10-25 原亮橙版本。

## 重新生成

```bash
node scripts/build-theme-css.mjs
node docs/design-previews/2026-10-27-petit-prince/_shared/apply-theme.mjs
node scripts/od-gallery.mjs docs/design-previews/2026-10-27-petit-prince
node scripts/od-shots.mjs docs/design-previews/2026-10-27-petit-prince
node scripts/od-shots.mjs docs/design-previews/2026-10-27-petit-prince --query "theme=bright" --suffix -bright
```
