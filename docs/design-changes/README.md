# 设计变更单

设计稿是视觉参考，代码是 UI 的唯一真相源。当你要**主动优化已实现页面的设计**（设计领先），用这里的变更单把「设计 → 实现 → 收口」串起来。

## 什么时候用

| 场景 | 用什么 |
|---|---|
| 探索方向、不一定落地 | `docs/design-explorations/`，不进 51 页批次，不进 CI |
| 已实现页面的视觉/交互改版 | **本文的变更单** |
| 只改 token（主色、字号、圆角） | 直接改 `tokens.json` + `design:sync` + `design:shots`，不需要变更单 |
| 新增页面 | 先补原型 + `design.config.json`，走 `design:sync`，实现后 `accept` |

## 创建

```bash
npm run design:change -- pickup-polish "接送页三态与完成态收口"
```

会在 `docs/design-changes/YYYY-MM-DD-pickup-polish.md` 生成模板。手写也可以，文件名必须是 `<date>-<kebab-slug>.md`。

## 流程

1. **定基线**：记录当前代码 commit，跑 `design:check` 必须绿。
2. **改稿**：只改原型 HTML/CSS/notes；跑 `od-check` 和 `design:shots -- --page <slug>`。
3. **设计评审**：看 before/after，把 `status` 改为 `approved`。
4. **实现**：改 `.vue` / `shared-ui`，必要时 `design:sync`。
5. **收口**：`design:accept -- <surface>/<slug> --reason "<本变更单文件名 + 实现 commit>"`，重出截图，`design:check`，代码 + lock + 截图同批提交。
6. **归档**：`status: implemented`，`git mv` 到 `docs/design-changes/archive/`（或保留在原位，历史更清楚）。

## 规则

- `accept --reason` 必填；reason 必须能指向变更单或实现 commit，不允许“看一眼没问题”。
- 设计 commit 与实现 commit 分开，设计 diff 可单独评审。
- 实现偏离设计时，在「差异」段写清楚，再 accept。
- 变更单是临时工件，落地后归档；不要把它变成永久合同。
- 只闭环本次涉及的页面；不要为了“全绿”批量 accept 无关页面。
