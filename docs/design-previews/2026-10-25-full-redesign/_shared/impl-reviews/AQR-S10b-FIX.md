# AQR-S10b-FIX · 对抗性实现审查

- verdict: **PASS**（修复轮）
- score: 92/100
- 源码覆盖: `RecipeConflictDialog.vue`、`plans/[id].vue`、`plans/index.vue`、`diet_seed_service.go`、`diet_seed_service_test.go`、`design.md`（6 文件）
- 审查范围: 完整 diff + 全仓事件名扫描 + 运行时控制组复现 + 构建产物 + 调用链 + 定向测试 + manifest/untracked
- 基线: `f4a66a6` → 工作区未提交（HEAD 未变，staged 0）

## 运行证据

- 构建: `cd admin && npm run build` → exit 0，`Build complete!`，Total 2.09 MB（16:28:35）；产物 chunk `Bwzzx5Im.js` 含 `onApplyOnce` ×1、`onOnce` ×0
- 类型: slice `vue-tsc -p .claude/runs/aqr-s10b/tsconfig.check.json` → exit 0
- Token: `node scripts/check-tokens.mjs admin` → exit 1，8 条存量（teachers/index.vue 6、_variables.scss 2），diet 新增 0
- Secret: `check-secrets.mjs --files <6>` → 0 findings
- Go: `go build ./...` exit 0；`go test ./internal/service -run 'DietSeed|DietTemplate'` → ok 43.967s；`TestDietSeedGlobals -v` → PASS 13.98s
- OpenSpec: `validate add-quantified-recipe --strict` → exit 0
- Console / 截图 / 真实点击: 本环境无浏览器驱动；按编排器指示交 E2E 重跑，未据此判失败

## F5（P0）事件名

- 根因独立复现：Vue 3.5.30 `runtime-core.cjs.js:4478-4484`，`onOnce` 剥 `Once` 得空串后 `key[0].toLowerCase()` 抛错。
- 修复：`emit('applyOnce')` + `defineEmits.applyOnce` + `@apply-once`；编译产物 `onApplyOnce`；三分支 testid 与 handler 全部接线。
- 独立临时 vitest（跑后删除）：控制组抛 `/toLowerCase/`；真实组件挂载成功，三个 testid 点击分别触发 `applyOnce/replace/cancel` 各 1 次。worker 的临时控制组结论可信。
- 全仓 grep `@once` / `emit('once')` = 0。

## F6（P1）二次确认

- 列表删除 `index.vue:548-563`、编辑器清空 `[id].vue:894-911`：均为「第一步确认 → 第二步最终确认 → 调 API」；任一步取消即 `return`。与既有 `removePlan` 风格一致。
- 真实浏览器双击待 E2E 步骤 16。

## F3（P1）叶子分类

- `SeedDietGlobals` 为 20 个一级分类幂等种 `Level=2`「其他」：`Unscoped` 按 `(parent_id,name)` 查重、软删复活、并发创建回查、`sort=9999`；新增 `LeafCategoriesAdded`。
- 测试断言：重复执行不新增、每根 ≥1 个二级、占位 `Level==2`、`requireLeafCategory` 返回成功且 `rootName==根名`。定向执行 PASS。
- UI：`foods/index.vue` 级联按根分组，占位可选；E2E 步骤 4 不再需要 `cmd/dietfood` 合成 CSV 绕行。
- design D10 已补「占位叶子、导入后共存」说明。

## F1 判定

spec `diet-food-library:131-134` 的菜谱库管理无「新增」；新增由「保存菜谱」重名分支（`:83-102`）承担。编辑器 `saveDishRecipe` 已实现。**E2E 重跑用编辑器创建菜谱，菜谱库页只验列表/详情/改名/删除/合并**。

## 范围

修复窗口 mtime 仅命中 6 个 manifest 文件；其余 150+ 脏文件为会话前存量；HEAD 未变、无 staged；token 生成文件的 prebuild 时间戳改动已还原。

## P1 · 阻塞问题

无。

## P2 · 建议

1. `server/internal/app/seeder.go:160` 与 `cmd/dietfood/main.go:106-109` 未输出 `leaf_categories_added`，存量库升级时日志会显得「什么都没种」。建议补打。
2. `diet_seed_service_test.go:137` 只断言第二次为 0；共享开发库可能预先已有占位，首次创建未被断言。建议用独立测试根断言首次 `==1`、再次 `0`。

## 下一步

E2E 重跑（新后端 + 新 admin）：步骤 4 直接 UI 建食物；步骤 5 从编辑器保存菜谱并验 409 替换/仅本次；步骤 7–11、16 为回归重点；步骤 17 console 未捕获异常应从 3 降为 0。7.5 许可门仍需用户签署授权协议。
