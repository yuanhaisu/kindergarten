# add-quantified-recipe 进度板

> 计划：`openspec/changes/add-quantified-recipe/SLICE-PLAN.md`
> 工作流：`.claude/skills/slice-delivery/SKILL.md`（适配 admin + Go，无微信/DevTools）
> 变更：`openspec/changes/add-quantified-recipe`
> base_sha：`8774bf5b4c12a214bd98a233ead8817a7004b051`
> 提交策略：每个 slice PASS 后自动 commit + push（2026-11-01 用户授权）；分支 `feat/add-quantified-recipe`
> 推进策略：PASS + commit + push 后自动进入下一 slice，不等用户确认；仅在 FAIL/BLOCKED、需产品决策、推送失败或上下文预算超限时停下

| Slice | Tasks | 目标 | 状态 | base | head | verdict | commit |
|---|---|---|---|---|---|---|---|
| AQR-S1a | 1.1–1.3 | 契约 + 模型 + AutoMigrate | passed | 8774bf5 | ebe8f21 | impl-reviews/AQR-S1a.md | ebe8f21 |
| AQR-S1b | 1.4–1.6, 2.3 | repository + Wire + 路由/权限 | passed | ebe8f21 | 0a30624 | impl-reviews/AQR-S1b.md | 0a30624 |
| AQR-S2 | 2.1, 2.2, 2.4 | 供餐配置后端 | passed | 0a30624 | 04197b0 | impl-reviews/AQR-S2.md | 04197b0 |
| AQR-S3 | 3.1, 3.5 | 食物库后端 | passed | 04197b0 | 74da9bc | impl-reviews/AQR-S3.md | 74da9bc |
| AQR-S4 | 3.2, 3.5 | 菜谱库后端 | passed | 74da9bc | d771b3e | impl-reviews/AQR-S4.md | d771b3e |
| AQR-S5 | 3.3, 3.4, 3.6 | 导入 + 种子 + 新园 hook | passed | d771b3e | 338c496 | impl-reviews/AQR-S5.md | 338c496 |
| AQR-S6 | 4.1–4.5 | 周食谱后端 | passed | 338c496 | b2b6faa | impl-reviews/AQR-S6.md | b2b6faa |
| AQR-S7 | 5.2–5.6 | 营养分析 + 导出 | passed | b2b6faa | 3dd5967 | impl-reviews/AQR-S7.md | 3dd5967 |
| AQR-S8 | 6.1–6.5, 6.9 | admin API/菜单/三个页面 | passed | 3dd5967 | f20560a | impl-reviews/AQR-S8.md | f20560a |
| AQR-S9a | S9 前置 | 学年接口补数值 id | passed | f20560a | fcec4f1 | impl-reviews/AQR-S9a.md | fcec4f1 |
| AQR-S9 | 6.6, 6.7 | admin 列表 + 编辑器 | passed | fcec4f1 | b29f0e1 | impl-reviews/AQR-S9.md | b29f0e1 |
| AQR-S10a | 6.8, 7.1 | 分析页 + TEST_CATALOG | passed | b29f0e1 | f4a66a6 | impl-reviews/AQR-S10a.md | f4a66a6 |
| AQR-S10b | 6.10, 7.2–7.5 | E2E + 最终验收 | passed | f4a66a6 | 2e6afd7 | impl-reviews/AQR-S10b.md | 2e6afd7 |
| AQR-S10 | 6.8, 6.10, 7.x | 分析页 + E2E + 验收 | pending | | | | - |

## Handoff（2026-11-01）

- **S1a PASS**：commit `ebe8f21`，已 push 到 `origin/feat/add-quantified-recipe`
- **下一 slice**：AQR-S1b（tasks 1.4–1.6, 2.3）——repository + Wire + 路由/权限骨架
- **恢复**：新会话读本文件 + `.claude/runs/aqr-s1a/checkpoint.json`；不要重跑 S1a
- **规则**：一次一个 slice；PASS → commit + push；每轮新 worker / 新 supervisor；每个 brief 必读 `openspec instructions apply --change add-quantified-recipe --json` 的 `context` / `operationGuidance`
- **Deferred P2**：S2 必须按 `meal_key` upsert 或硬删子行，并补「连续保存两次」测试；批量删除部分成功、`week_of_year` 默认 0 已接受
- **⚠️ 共享工作树冲突**：并发相册会话把工作区分支切回 `feat/parent-growth-scope`，S1a commit `ebe8f21` 也落在该本地分支上。**不要直接 push 默认分支**；继续前请在干净工作树或专用 worktree 上切到 `feat/add-quantified-recipe`。
- **上下文**：本会话已 64%，按 slice-delivery 规则在此 handoff。

## 决策记录

- **2026-11-02 · AQR-S2 供餐配置 spec 矛盾**：默认占比 25/5/35/5/25（95%）与「早+早点≤29（日托）、午+午点≤39」无法同时成立。用户选 **B 标准/自定双模式**（对齐参考产品 `precentType`）：`ratio_mode=standard`（默认）固定标准占比并跳过相邻上限；`custom` 才校验 >0、时间递增与 29/39/30 上限。同步修改 domain（MealConfig 加 `ratio_mode`）、契约 `23-diet.md`、spec、design。理由：保住 0.95 折算与 TC-0005/0006。

## 最终状态（2026-11-03）

- **全部 slice PASS**：S1a / S1b / S2 / S3 / S4 / S5 / S6 / S7 / S8 / S9 / S9a / S10a / S10b；HEAD `2e6afd7`，已 push `origin/feat/add-quantified-recipe`。
- **E2E Round 2 PASS**：真实 Playwright 点击覆盖配置→食物→菜谱→周食谱→编辑器（白屏修复后）→同类替换→409 替换/仅本次→清空/删除两次确认→分析/导出/打印/空态；0 未捕获异常、0 500。
- **豁免 W1（2026-11-03 部分修复）**：notice 日期炸弹已修（改用 2099-01-01，定向测试 PASS）；`make test` 仍可能因 service 套件 MySQL COMMIT 抖动中断。diet 103 测试 PASS / 0 FAIL。
- **豁免 W2（2026-11-03 已修复）**：14 条存量 token 违规已清零（`check-tokens` 0）；CalendarEventCard 改用 `previewParent` token，admin 教师页改用 Tailwind token 类。
- **发布阻断 7.5**：上线前必须与中国疾控中心营养与健康所签署授权协议并导入授权 CSV；Sanotsu 数据仅本地开发验证，未提交、未分发。
- **残留 dev 数据（非仓库）**：`e2e-s10b-rice`（kg=0 及 6 园副本）、分类 587；上线前清理。
- **临时关闭**：`/tmp/aqr-s10b-*` 日志与 `.claude/runs/` 运行工件不入库。
