# 配色主题说明（2026-10-27）

三套方案共用同一份 HTML/CSS，只切换颜色 token。默认是「亮橙」，另外两套通过 `?theme=` 或 `data-theme` 启用。

| 主题 | 说明 | 白字压品牌橙对比度 | 合规 |
|---|---|---|---|
| `bright`（默认） | 原设计稿的亮橙，视觉优先 | 2.35 / 2.80:1 | ✗ 不达 WCAG AA |
| `deep` | 白字保留，品牌渐变加深为焦糖橙 | 5.18–7.31:1 | ✓ AA |
| `ink` | 保留亮橙渐变，按钮文字改深墨色 | 5.9–7.4:1 | ✓ AA |

## 唯一入口

- 基础 token（亮橙）：`shared-design-tokens/tokens.json`
- 主题覆盖（deep / ink）：`shared-design-tokens/themes.json`
- 生成主题 CSS：`node scripts/build-theme-css.mjs` → `_shared/themes.css` + `shared-design-tokens/build/css/themes.css`
- 同步 OpenDesign 适配层：`node scripts/sync-open-design-adapter.mjs`
- 把 theme 链接/脚本接入原型：`node scripts/apply-themes.mjs`

## 运行时切换

- 页面地址：`index.html?theme=deep` / `index.html?theme=ink` / `index.html?theme=bright`
- 或直接给 `<html>` 加 `data-theme="deep|ink"`（`_shared/theme.js` 会读取 query 并写入该属性）
- 无 `theme` 参数 = `bright`

## 改了哪些 token

`deep` / `ink` 共同加深：

| token | 亮橙默认 | deep / ink |
|---|---|---|
| `--muted` | `#64748B` | `#475569` |
| `--hint` | `#94A3B8` | `#5F6B7A` |
| `--success` | `#16A34A` | `#15803D` |
| `--warning` | `#CA8A04` | `#92400E` |
| `--danger` | `#EF4444` | `#B91C1C` |
| `--info` | `#2563EB` | `#1D4ED8` |
| `--accent-deep`（老师） | `#e6762a` | `#9A3412` |
| `--accent-deep`（家长） | `#EA580C` | `#9A3412` |

品牌渐变：

| 主题 | 老师 | 家长 |
|---|---|---|
| `bright` / `ink` | `#ffa866 → #ff8a3d` | `#FB923C → #F97316` |
| `deep` | `#C2410C → #9A3412` | `#C2410C → #9A3412` |

`ink` 额外把 `--on-accent` 设为 `#1E293B`（深墨字）；`bright` / `deep` 为 `#FFFFFF`。

## 后续按园所偏好调整

当前是原型层方案，后端未接。正式落地时建议：

1. `kindergarten_app_setting` 增加 `theme` 字段（`bright|deep|ink`）
2. 客户端启动时读取该设置，给根节点写 `data-theme`
3. 管理端园所设置页提供三选一，并显示对比度说明
4. token 继续以 `tokens.json` + `themes.json` 为唯一入口，不逐页改 CSS

在未接后端前，园所可以通过 `?theme=` 预览，或由部署方在入口页固定 `data-theme`。
