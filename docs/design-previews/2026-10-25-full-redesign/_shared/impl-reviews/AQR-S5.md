# AQR-S5 · 对抗性实现审查

- verdict: **PASS**
- score: 92/100
- 源码覆盖: `cmd/dietfood/main.go`、`cmd/dietfood/main_test.go`、`cmd/initcheck/main.go`、`app/provider.go`、`app/seeder.go`、`app/wire_gen.go`、`repository/diet_dri_repo.go`、`repository/diet_food_repo.go`、`repository/diet_template_repo.go`、`service/diet_seed_service.go`、`service/diet_seed_service_test.go`、`service/diet_template_seed.go`、`service/system_service.go`、`service/wire.go`
- 审查范围: diff + untracked + 调用链 + design D2/D6/D10 + spec TC-0009/0016 + DRI 溯源 + 许可门 + 启动 wiring
- 基线: `d771b3e57de7c98e704108c7a7ed93d194e35805` → 工作区未提交

## 验证证据（后端 slice，无 DevTools）

- 构建: `cd server && go build ./...` → exit 0，无输出
- vet: `go vet ./...` → exit 0，无输出
- 导入命令: `go test ./cmd/dietfood -count=1 -v` → exit 0；5 顶层 PASS（11.78s，真实 MySQL，未 skip）
- 种子 service: `go test ./internal/service -run 'DietSeed|DietTemplate' -count=1 -v` → exit 0；6 顶层 PASS（38.10s，未 skip）
- 新园 hook 回归: `go test ./internal/app -run 'Seed|Router' -count=1` → exit 0（13.97s）；app 全包 `-count=1` → exit 0（14.04s）
- 仓储: `go test ./internal/repository -run 'Diet' -count=1` → exit 0（16.18s）
- 既有膳食回归: `go test ./internal/service -run 'DietFood|DietRecipe|DietMealConfig' -count=1` → exit 0（163.27s）
- gofmt: 14 个 slice Go 文件 → 无输出
- Secret scan: `node scripts/check-secrets.mjs --files <14 文件>` → 0 findings
- Wire: `make wire` 连续两次 → exit 0；`wire_gen.go` sha256 `73ba2f20…2bc24` 不变（幂等）
- 数据库: `go run ./cmd/dbcheck` → `diet_foods` 0、`diet_food_categories` 20、`diet_dri_age_targets` 44、`diet_template_plans` 0；分类/DRI 唯一索引存在，食物 preset/模板 key 为普通索引（符合 design D2）
- 正常态: 真实 MySQL 覆盖新增/更新/跳过/错误四类计数、NULL 不转 0、二次导入幂等、`--kindergarten` 分发、TC-0016 冲突报告
- 空态: 无食物园所跳过 2 套模板并 `slog.Warn`；只有表头的 CSV 明确提示且退出码 1
- 错误态: 非法数字/未知一级分类/重复 key → 行级错误、整批继续；写库错误 → 退出码 2；模板缺失食物 → 逐条跳过并汇总告警
- 路由: 后端 slice，无 tabBar/switchTab；未改路由或 Casbin
- 截图: N/A（纯后端）

## 范围

- `git diff --name-status HEAD -- server` = 9 个修改；untracked server = 5 个新增，与 worker manifest 完全一致；无 staged 内容。
- 50+ 条额外 ` M` server 路径为 stat 脏位：抽查 `app/service_provider.go`、`domain/attendance.go` 的 blob hash 与 `HEAD:path` 相同，无内容改动。
- brief 外 2 个文件判为必要最小：`app/provider.go` 在 AutoMigrate 后新增 1 行 `SeedDietAfterMigration(db)`（紧邻 `SeedDictsAfterMigration` 先例，否则新 seeder 是死代码）；`cmd/initcheck/main.go` 因 `NewSystemService` 签名 +1 实参，机械连带。
- 未改 `domain/`、`backend/api/`、`configs/`、`admin/`、`teacher/`、`parent/`；无 `.csv`/`.xlsx` 新增。

## 行为与 spec 核对

- `cmd/dietfood`：表头 33 必需列（6 固定 + 27 营养）与 `--help` 一致；空/`-`/`—` → NULL；非法数字/负数/NaN/Inf/未知一级分类/重复 `preset_key` → 行级错误；四类计数真实断言；`--kindergarten` 目标园存在性检查 + 已修改行不覆盖；`--dry-run` 在 `seedutil.Open()` 之前返回，不连库。
- `SeedDietGlobals`：20 个一级分类名称/排序逐条对照 `food-getFoodTypeList.json`（foodTypeID 1–20）；44 行 = 3/4/5/6 岁 × 11 项，唯一 `(age_band,nutrient_key)`，二次播种 0 新增；软删行复活。
- `SeedDietForAll` / `EnsureKindergartenSeeded`：按 `preset_key` 只补缺失；ID/时间戳/软删位清零，NULL 营养列保留；二次调用 0 复制；单园失败 warning 后继续。
- 模板：2 套（均衡周/清淡周）× 5 工作日 × 3 餐 × 每餐 1 菜 × 每菜 1–2 食物；按本园食物名解析 `food_id`；无食物跳过并告警；缺失食物逐条跳过并汇总；整块解析不到不建模板。
- 新园 hook：`SystemService.Initialize` 在园所事务提交后调用 `EnsureKindergartenSeeded`，失败仅 `slog.Error`；`dietSeed == nil` 有保护。启动接线在 AutoMigrate 之后，顺序正确。
- Wire 幂等；`initcheck` 调用点编译通过。

## 数据准确性与许可

- DRI 明确来源：能量男值 1350/1450/1600/1700、蛋白 45/50/55/55、VA 500/600/600/600、VC 60/70/70/70、钙 600/800/800/800、锌 9/12/12/12、4–6 岁 B1/B2 0.7 与铁 12（notes §5.1/§5.4）；脂肪/碳水按 §5.4 派生关系。
- 3 岁 B1/B2=0.6：notes 无原表单列，由观测 0.67、w3=26.7%、4–6 岁 0.7 反解，一位小数唯一解 0.6；3 岁铁=12 由观测 12 与 4–6 岁 12 代数强制。**不是编造**；代码注释、self-review §4、worker `known_gaps` 三处已标注「上线前用 DRIs 2000 原书核对」。
- 许可：无任何真实食物成分 CSV/数据集；测试 CSV 全在 `t.TempDir()` 合成；模板初稿只有常见食物名与克数，无营养数据；DRI 为标准值而非《中国食物成分表》数据。
- 模板初稿标记：`service/diet_template_seed.go:8-9` 明确「上线前必须由园所营养师审核」。

## P1 · 阻塞问题 (0)

无。

## P2 · 建议 (5)

1. **[data provenance]** 3 岁 B1/B2=0.6 是 §5.4 一位小数反解值（铁 12 为代数强制），notes 无 2000 版 1–3 岁原表行；数值自洽且已三处 caveat，上线前仍应拿原书核对。
   - 证据: `service/diet_seed_service.go:421-423,447-453`；`notes.md:284-300`。
2. **[test gap]** `SeedDietForAll`/`SeedDietAfterMigration` 的遍历、id=0 跳过、warning 汇总无直接 DB 断言；单园逻辑已覆盖，直接测试会改写开发库所有园所，判为可接受的已知缺口。
   - 证据: `service/diet_seed_service.go:186-212`；`app/seeder.go:146-178`。
3. **[idempotency edge]** 模板按 `template_key` 存在即跳过：首次只解析到部分食物时，后续导入补齐食物后不会补全模板；未来初稿升级也不会应用到已建模板。
   - 证据: `service/diet_seed_service.go:275-281`。
4. **[exit code doc]** `seedutil.Open()` 对读配置/连库失败调用 `log.Fatal`（退出码 1），与 self-review/`--help` 的「参数/读文件/写库失败退出码 2」不完全一致；brief 只要求非 0，故不阻塞。
   - 证据: `cmd/internal/seedutil/seedutil.go:28,31,37`；`cmd/dietfood/main.go:15,175`。
5. **[concurrency]** `copyPresetFoods` 为查后插，`diet_foods` 按 design D2 刻意无 `(kindergarten_id,preset_key)` 唯一索引；多实例同时启动可能重复复制同一 `preset_key`（顺序重跑幂等）。
   - 证据: `service/diet_seed_service.go:320-360`；`domain/diet_food.go:30-32`。

## TC 覆盖

S5 新增 `TestDietSeedCopiesPresetFoods`（TC-0009）与 `TestDietFoodImport`（TC-0016），均为真实 MySQL 断言；self-review 覆盖表引用的 14 个 S3/S4 测试名全部存在。`TEST_CATALOG.md` 尚无 `TC-NEW-DIE-*`（0 处），S10 统一补录，与本 slice known gap 一致。

## 结论

全部 acceptance gate 独立复跑通过；manifest 与工作区一致；无真实数据集入库；DRI 反推值有出处且已 caveat；模板初稿已标记营养师审核。判 **PASS**，5 条 P2 不阻塞提交。
