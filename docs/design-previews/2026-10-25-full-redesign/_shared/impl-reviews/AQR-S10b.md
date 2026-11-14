# AQR-S10b · 最终对抗性实现审查

- verdict: **PASS**（含显式豁免 W1/W2；7.5 为发布阻断，不是代码失败）
- score: 93/100
- 源码覆盖: `RecipeConflictDialog.vue`、`plans/[id].vue`、`plans/index.vue`、`diet_seed_service.go`、`diet_seed_service_test.go`、`design.md`
- 审查范围: Round 2 E2E 证据 + 6 文件 diff + 构建/类型/秘密/token/openspec + 全量 `make test` 一次 + manifest
- 基线: `f4a66a6` → 工作区未提交（HEAD 未变，staged 0）

## 运行证据

- 构建: `cd admin && npm run build` → exit 0，`Build complete!`，2.09 MB；产物 chunk 含 `onApplyOnce` ×1、`onOnce` ×0
- 类型: `vue-tsc --noEmit -p .claude/runs/aqr-s10b/tsconfig.check.json` → exit 0
- Secret: `check-secrets.mjs --files <6>` → 0 findings
- Token: `check-tokens.mjs` → exit 1，14 条存量（teacher/parent CalendarEventCard、admin teachers 6、admin `_variables.scss` 2）；diet 0
- OpenSpec: `validate add-quantified-recipe --strict` → exit 0
- Go: `go build ./...` exit 0；`make test` exit 2 —— `internal/service` 600.257s 超时，**diet 103 PASS / 0 FAIL**，唯一 FAIL 为 notice 日期炸弹，卡点为 notice fixture 的 MySQL Commit/Pluck（三轮卡点不同，预存在抖动）
- 正常态: Round 2 真实浏览器，编辑器网格 / 小计 134.1→101.5→108.3 kcal，刷新保持；截图 `r2-05`
- 空态: 「该食谱暂无明细，无法生成营养分析」+ 编辑器/列表 CTA；截图 `r2-07`
- 错误态: 0g 保存被拒；409「仅本次 / 替换」两分支按预期（步骤 10–11）
- 交互: 编辑器打开、加菜、改克数、同类替换、冲突弹窗、打印 stub、清空/删除二次确认 → 均真实点击通过（步骤 7–11、14、16）
- Console: Round 2 清缓冲后 0 未捕获异常、0 console error；后端 0 ERROR / 0 HTTP 500
- 路由: admin Nuxt，无 tabBar/switchTab；本 diff 不新增路由
- 截图: `.claude/runs/aqr-s10b/screenshots/r2-01..r2-07`；r2-03/04/05/06/07 经 mcp-vision 独立复核

## F5（P0）事件名白屏

- 根因: `@once` → `onOnce`，Vue `isEmitListener` 剥 `Once` 得空串后 `key[0].toLowerCase()` 抛错（Vue 3.5.30）。
- 修复: `emit('applyOnce')` + `defineEmits.applyOnce` + `@apply-once`；三分支 testid 未改。
- 复核: 全仓 grep `@once|emit('once')|onOnce` = 0；构建产物 `onApplyOnce` ×1 / `onOnce` ×0；Round 2 编辑器与冲突弹窗正常。

## F6（P1）二次确认

- 列表删除 `index.vue:550,555`、编辑器清空 `[id].vue:897,902`：第一步 → 最终确认 → 才调 API；任一步取消即返回。Round 2 步骤 16 真实点击验证。

## F3（P1）叶子分类

- `SeedDietGlobals` 为 20 个一级幂等补二级「其他」（软删复活、并发回查、sort 9999），新增 `LeafCategoriesAdded`；测试断言每根 ≥1 个二级、`requireLeafCategory` 接受占位。
- 运行时 `r2-cats.json`：20 roots，全部有 level-2「其他」（21 children，含 R1 合成叶子 587）；UI 直接建食物成功。
- design D10 已补「占位叶子，授权 CSV 导入后与真实二级共存」。

## 验收与豁免

- 4 条命令: build PASS、openspec PASS、Go test FAIL、check-tokens FAIL（2/4）。
- **W1 Go test**: 预存在 notice 日期炸弹（`notice_drafts_test.go:194` 写死 2026-11-01）+ service 套件 MySQL 卡死；8080 无后端、无 API 进程、DSN localhost，三轮卡点不同；diet 0 失败。需用户明确接受为基线豁免。
- **W2 check-tokens**: 14 条存量，diet 0。需用户明确接受。
- **7.5**: 数据许可门，用户/法务签署授权协议并导入授权 CSV 前为**发布阻断**；合成 fixture 不得作为正式数据。

## P1 · 阻塞问题

无。

## P2 · 建议

1. `tasks.md` 1/42（仅 5.1 勾选）；7.4/7.5 未勾，archive 前需如实对账。
2. `server/internal/app/seeder.go:160-163`、`cmd/dietfood/main.go:106-109` 未输出 `leaf_categories_added`。
3. `diet_seed_service_test.go:137` 仅断言第二次不新增；首次创建未在干净状态覆盖。
4. E2E 合成数据残留在共享开发库（kg=0 食物已复制到 6 个园所、二级分类 587），无删除入口；正式导入前清理。
5. `openspec/changes/add-quantified-recipe/` 在 base 即未跟踪；6 文件提交只含 design.md，其余提案/规格/任务仍未跟踪，编排器需决定。

## 下一步

- commit 6 文件（清单见 `.claude/runs/aqr-s10b/verdict.json`）；PASS 以 W1/W2 被用户明确接受为前提。
- 独立后续 chore：notice 日期炸弹、service 套件 MySQL 卡死、tasks.md 对账、E2E 合成数据清理。
