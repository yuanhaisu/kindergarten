# 设计批次基线 · 2026-10-27

> Git tag：`design-baseline-2026-10-27`（本地提交，未 push）
> 范围：`docs/design-previews/2026-10-25-full-redesign/` + `shared-design-tokens/` 亮橙默认与主题层。

## 1. 冻结内容

| 项 | 数量 / 状态 |
|---|---|
| 原型页面 | 50 页（老师端 22 / 家长端 28） |
| 每页交付 | `index.html` / `styles.css` / `app.js` / `tokens.css` / `notes.md` / `preview.png` |
| 配色主题 | `bright`（默认，亮橙）/ `deep`（深橙 AA）/ `ink`（亮橙+深墨字 AA） |
| 主题对比图 | 7 个代表页 × 3 方案 = 21 张（含默认），主题图 14 张 |
| 审查记录 | `_shared/REVIEW.md` + `_shared/reviews/` 50 份单页审查 |
| 接口/设计缺口 | `_shared/findings.md`（A1–A42 + 原有 9 接口 + 3 token） |
| 主题说明 | `_shared/THEMES.md` + `_shared/themes.html` |

## 2. 冻结时的质量状态

- **P0**：5 条全部修复并实测。
- **P1**：85 项已修；4 条对比度项在 `deep` / `ink` 主题下解决；剩余 4 条暂缓（bind 入口、notice 缺字段、growth 空态芯片、登录默认芯片）。
- **P2**：253 条未处理，留待 Impeccable 落地阶段。
- **跨页决策**：演示芯片规范、家长端首页/接送页今日叙事、老师端工作台/我的重复统计，仍待产品确认。
- **批次门禁**：`od-audit` 50/50 通过，0 问题；64 张预览均 780px；无 `.mcp.json`；无 `teacher/` / `parent/` 源码改动。

## 3. 主题与对比度

| 主题 | 白字压品牌橙 | 合规 |
|---|---|---|
| `bright`（默认） | 2.35 / 2.80:1 | ✗ 视觉参考 |
| `deep` | 5.18–7.31:1 | ✓ AA |
| `ink` | 5.9–7.4:1 | ✓ AA |

- 入口：`shared-design-tokens/tokens.json`（亮橙默认）+ `shared-design-tokens/themes.json`（deep / ink）。
- 切换：`index.html?theme=deep` / `?theme=ink`；无参数 = 亮橙。
- 对比页：`_shared/themes.html`。

## 4. 复验命令

```bash
node scripts/od-audit.mjs docs/design-previews/2026-10-25-full-redesign
node scripts/od-check.mjs docs/design-previews/2026-10-25-full-redesign/parent/01-home/index.html --width 390
```

文件级校验见同目录 `baseline.sha256`（交付文件 + `_shared` 文档 + 主题文件）。

## 5. 明确不在基线内

- `teacher/`、`parent/` 业务源码（本批次只出原型）。
- P2 打磨项与跨页设计决策。
- 后端 `kindergarten_app_setting.theme` 字段（仅文档中给出接法建议）。
