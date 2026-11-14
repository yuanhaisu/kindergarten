# 养成星球 · Logo 使用规范（A1 + P1）

## 色板

| Token | 值 | 用途 |
|---|---|---|
| Ink | `#1B2A41` | 弧、字标、深色底 |
| Amber | `#E8A23A` | 种子点，唯一暖色 |
| Cream | `#F7F3E8` | 徽章底、纸感留白 |

不要新增第 4 个品牌色；不要把琥珀点换成品牌橙 `#FF8A3D` / `#F97316`，那会失去「种子」的点睛。

## 最小尺寸

- 图形：≥ 16px
- 横排锁图（纯中文）：≥ 280px 宽
- **横排锁图（中英双语）：≥ 400px 宽**——英文 32px 在更小的宽度下读不清
- 竖排锁图：≥ 240px 宽
- 奶油 / 深墨徽章：≥ 48px
- app icon：144px（微信）/ 512 / 1024
- favicon：16 / 32 / 48

## 安全距离

- 图形四周留白 ≥ 图形宽度的 25%（256 画布上约 64）。
- 徽章四周留白 ≥ 徽章直径的 1/8。

## 背景

- **奶油 / 白**：彩色图形或横排锁图。
- **品牌橙**：用**奶油徽章**（透明图形 + 琥珀点在橙色上会融）。
- **深色**：深墨徽章或反白横排 / 反白图形。
- **favicon**：奶油方块 + 彩色图形。

## 不要做

- 不要合上弧的缺口、不要删掉或改色琥珀点。
- 不要给弧加星、行星环、嫩芽、房子、小孩或吉祥物。
- 不要拉伸、旋转、加阴影 / 渐变 / 描边。
- 不要把字标缩小到 24px 以下；不要用另一款字体重排字标（字标已轮廓化）。
- 不要把透明锁图直接放在品牌橙上。

## 字体

- 字标：**Noto Sans SC Bold**（SIL OFL），已轮廓化。
- 正式商用建议做定制中文字标，以获得独占字形；当前版本可先用。

## 三端接入清单

1. **老师端 / 家长端**：`teacher/src/static/logo.png`、`parent/src/static/logo.png` 替换为 `exports/png/badge-cream-320.png`；`shared-ui/BrandBlock.vue` 的 160rpx 圆形不用改结构。
2. **管理端侧栏**：用 `exports/png/badge-cream-128.png`（64px 显示）替换原来的渐变方块 + `StarFilled`，保留「养成星球」文字。
3. **管理端 favicon**：把 `exports/web/favicon.ico` 放到 `admin/public/`，在 `admin/nuxt.config.ts` head 里加 `head-snippet.html` 的 link。
4. **微信 app icon**：在 mp.weixin.qq.com 后台上传 `exports/app-icon/app-icon-144.png`；两端各自上传。
5. **PWA / web**：`exports/web/` 的图标 + `site.webmanifest` + `head-snippet.html`。
6. 旧 `assets/logo.*` 保留到验收后再归档；`PRODUCT.md` / `assets/logo-design-philosophy.md` 在接入时再切到本 kit。

## 验收

- 16 / 32 / 81 / 144 / 512 目视
- 奶油、品牌橙、深墨三种背景
- 单色 / 反白
- 图形在 16px 仍读成「弧 + 点」；字标在 24px 以上可读
- 上线前建议做商标检索；本文件不构成法律意见
