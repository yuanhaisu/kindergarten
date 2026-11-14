# 养成星球 · Logo Kit（A1 + P1）

**状态**：2026-11-02 定稿 **A1 弧巢 + 色板 1**；**teacher 已接入**（`teacher/src/static/logo.png`），**parent 待接入**；旧 `assets/logo.*` 保留归档。

## 主文件

- `masters/A1-symbol-color.svg` — 主图形：深墨弧 + 琥珀种子点
- `masters/A1-symbol-mono.svg` / `-reversed.svg` / `-reversed-mono.svg`
- `masters/wordmark.svg` — 「养成星球」轮廓化字标（Noto Sans SC Bold，无 `<text>`）
- `masters/A1-horizontal-color.svg` / `-reversed.svg` — 横排锁图（纯中文）
- `masters/A1-horizontal-bilingual.svg` / `-reversed.svg` — 横排锁图（中英双语，英文 Nurture Planet 在下一行）
- `masters/A1-stacked-color.svg` — 竖排锁图
- `masters/A1-badge-cream.svg` / `-ink.svg` — 奶油 / 深墨徽章
- `exports/app-icon/` — 1024 / 512 / 144；`app-icon-badge` 为带环备用
- `exports/web/` — favicon.ico / svg、16/32/48 PNG、apple-touch、192/512、maskable、webmanifest、head-snippet
- `exports/png/` — 图形 512–16、横排 960/480、竖排、徽章、字标、单色 / 反白
- `exports/kit-overview.png` — 一页总览

## 研究依据

- `docs/research/2026-11-02-kindergarten-logo-landscape/REPORT.md`
- 方向对比：`.../directions/compare-A1-A2-color.png`
- **A2 双弧守护保留为备选**；本次按用户选择做 A1。

## 字体与许可

- 字标轮廓来自 **Noto Sans SC Bold**（SIL OFL），已烘焙为路径，不依赖 `<text>`。
- 正式商用若需要独占字形，建议在 kit 基础上做定制中文字标；符号本身为 A1 原始几何。
- 未描摹任何竞品。

## 下一步

1. 按 `USAGE.md` 接入 parent / admin（teacher 已完成）。
2. 微信 app icon 在 mp.weixin.qq.com 后台人工上传 `exports/app-icon/app-icon-144.png`。
3. 旧 `assets/logo.*` 已保留归档，不再更新。
