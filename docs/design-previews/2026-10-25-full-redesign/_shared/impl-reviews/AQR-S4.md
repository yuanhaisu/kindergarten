# AQR-S4 · 对抗性实现审查

- verdict: **PASS**
- score: 93/100
- 源码覆盖: `service/diet_recipe_service.go`、`service/diet_recipe_test.go`、`service/wire.go`、`handler/http/diet.go`、`handler/http/handler.go`、`app/router.go`、`app/router_diet_test.go`、`app/router_perm_test.go`、`app/router_test.go`、`app/wire_gen.go`、`middleware/admin_perm.go`、`middleware/admin_perm_test.go`、`repository/diet_recipe_repo.go`、`repository/diet_recipe_repo_test.go`、`service/admin_album_test.go`
- 审查范围: diff + untracked + 调用链 + 路由/权限 + 契约 §四 + design D2/D4/D7 + spec TC-0017..0027 + base 红灯归因
- 基线: `74da9bc5` → `74da9bc5`（未提交）

## 验证证据（后端 slice，无 DevTools）

- 构建: `cd server && go build ./...` → exit 0，无输出
- 菜谱 service: `go test ./internal/service -run DietRecipe -count=1 -v -timeout 600s` → exit 0，13 顶层 PASS + 8 校验子测试（63.1s，真实 MySQL）
- 路由: `go test ./internal/app -run 'RouterDiet|RouterPerm' -count=1 -v` → exit 0，9 PASS（7 条菜谱路由；diet 精确 17 条）
- app 全包: `go test ./internal/app -count=1` → exit 0（`ok 14.0s`）
- 权限: `go test ./internal/middleware -run AdminPerm -count=1 -v` → exit 0，11 PASS（本 slice 新增 3 条）
- 仓储回归: `go test ./internal/repository -run 'DietRecipe|Recipe' -count=1 -v` → exit 0，3 PASS（含 `MergeInto`）
- vet: `go vet ./...` → exit 0，输出 0 字节
- gofmt: 15 个 slice 文件 → 无输出
- Secret scan: `node scripts/check-secrets.mjs --files <15 文件>` → 0 findings
- Wire: `make wire` → exit 0；执行前后 `wire_gen.go` md5 `f80ea3b69c0248c49e1a3028533bfc99` 不变（幂等）
- 正常态: 真实 MySQL 建食物+菜谱，断言 201 新增 / 409 同名 / 200 替换 / 详情 food_name+grams+energy / 列表 total / 联想 / 422 自合并 / 删除后 404
- 空态: 列表/联想/详情 items 均为 `make([]T, 0)` → JSON `[]`；删除后列表 total 0
- 错误态: 409/40900「菜谱名称已存在」、404/40400、422/42200 均经真实 HTTP + service 断言
- 路由: 后端路由，无 switchTab 场景；Casbin `p, admin, /api/v1/*, *` 通配 → 不改进 `rbac_policy.csv`
- 截图: N/A（纯后端）

## 范围

- `git diff --name-only <base> -- server/` = 13 文件 + untracked 2 文件，与 worker manifest 完全一致。
- 60+ 条额外 ` M` server 路径为 stat 脏位：抽查 blob hash 与 `HEAD:path` 相同，无内容改动。
- 两个 brief 外测试调用点（`router_test.go`、`admin_album_test.go`）各 +1 个 `nil`，是允许的 `NewHandler` 签名变更的必要机械连带，不改行为，已披露 → 不构成越界。
- `domain/`、`backend/api/`、`configs/`、`admin/`、`teacher/`、`parent/` 无改动；`shared-design-tokens/` 的既有脏改动早于本 slice。
- `find server -newermt '2026-11-02 11:20'` 只命中 manifest 文件（+ 审查重跑生成的 `wire_gen.go`）。

## base 红灯归因

- base 的 `router_perm_test.go` 仍断言 diet 只有 2 条配置路由，而 S3 已注册 10 条（2 配置 + 8 食物）→ `ElementsMatch` 必红。
- brief Allowed 明确列入 `router_perm_test.go` 并给出 `go test ./internal/app -run RouterPerm` gate，验收又要求 app 全包绿；S4 更新为精确 17 条属于合法修复，非顺手改无关测试。

## 契约核对

- 7 条路由与请求/响应形状逐项对齐契约 §四：list（`list/total/page/page_size` + `id/name/spell_code/item_count`）、create（201/200 + 同字段）、detail（`food_id/food_name/grams/sort/nutrients`）、update（`{id}`）、delete（`{}`）、merge（`{target_id,merged_item_count}`）、suggest（`{list}`）。静态段 `suggest`/`merge` 先于 `:id`，无 501/stub。
- `replace=false` 同名 → 409/40900 且不写库；`replace=true` → 同 ID 覆盖、200；无同名时按新增 201。同菜同食物在新增/修改/合并三路径均合并克数。
- spell_code 服务端生成；改名撞他菜谱 409；删除软删 + 配料硬删；合并同食物累加后删源（单事务）；suggest 名称包含或拼音码前缀、limit 默认 18/上限 100；跨园统一 404。
- kg=0 平台食物不进入详情营养（FindByIDs 租户 scope + 测试直接插入 kg=0 行证明 null）；历史 MealPlanItem 的菜名/克数/营养快照不随菜谱改删变化。
- 名称唯一由服务层保证，`domain` 只有非唯一索引（design D2），软删后同名可重建。
- 权限 7 条 fail-closed 登记：GET→`diet:view`，写→`diet:manage`；view/manage 互斥测试通过。

## TC 覆盖

0017、0018、0019、0020、0021、0022、0023、0024、0025、0026、0027 均有真实 DB 断言；测试数据用 990993/990994 专属园所并 `t.Cleanup` 硬删清理。

## P1 · 阻塞问题 (0)

无。

## P2 · 建议 (5)

1. **[concurrency]** 名称唯一是读后写，无唯一索引/串行化；并发同名保存仍可能落两条。design D2 明确选择服务层保证 → 接受；建议 S8 前端防连点。
   - 证据: `service/diet_recipe_service.go:220-240`、`:284-295`
2. **[semantics]** 详情对引用已删/kg=0 食物的脏配料保留空名、全 null 营养的行；不泄漏但契约未定义。worker 已披露。
   - 证据: `service/diet_recipe_service.go:178-197`
3. **[concurrency]** 合并的源/目标在事务外读取，事务内不复查；并发改配方会被覆盖，并发删源时 0 行删除也当成功。
   - 证据: `service/diet_recipe_service.go:341-357`、`repository/diet_recipe_repo.go:230-263`
4. **[validation]** 名称允许 128 字符而 `spell_code` 列 64；65+ 汉字名在严格模式下变 500 而非 422（沿用 S3 食物库现状）。
   - 证据: `service/diet_recipe_service.go:474-483`、`domain/diet_recipe.go:11-12`
5. **[test]** `diet_recipe_test.go:299` 注释误写「核桃(鲜) 5g + 5g = 10g」，实际断言面粉；不影响断言。

## 结论

10 条验收门独立复现全部通过；manifest 与工作区一致；无 P1、无越界、无 stub、无密钥；base 的 RouterPerm 红灯已按 brief 合法修复。**PASS**，可提交（提交清单 = worker manifest 15 文件 + 本文件）。
