# AQR-S7 · 对抗性实现审查

- verdict: **PASS**
- score: 92/100
- 轮次: Round 2（P1 修复复验；Round 1 判 FAIL）
- 源码覆盖: `service/diet_analysis_service.go`、`diet_analysis_conclusion.go`、`diet_analysis_export.go`、`diet_analysis_test.go`、`diet_analysis_export_test.go`（均新增）、`domain/diet.go`、`domain/diet_food.go`、`handler/http/diet.go`、`handler/http/handler.go`、`app/router.go`、`app/wire_gen.go`、`middleware/admin_perm.go`、`service/wire.go`、6 个测试/调用点文件
- 审查范围: diff + untracked + 调用链（router → handler → service → repo → domain 快照）+ 权限表 + 契约 §7 + spec TC-0044..0064 + api-samples + 导出
- 基线: `b2b6faab435e8ce64bc98822abdd9d0b318b3468` → 工作区未提交（后端 slice，无 DevTools）

## 后端证据（无 DevTools）

- 构建: `cd server && go build ./...` → exit 0，无输出
- vet: `go vet ./...` → exit 0，无输出
- service 全量回归: `go test ./internal/service -run 'DietAnalysis|DietPlan' -count=1 -v -timeout 600s` → exit 0，`ok 223.131s`，0 FAIL；**DietAnalysis 23 top-level PASS**（含真实 MySQL 的 TC-0063 10.94s、跨园/空态 8.87s，未 skip）+ S6 DietPlan 全绿
- 路由: `go test ./internal/app -run 'RouterDiet|RouterPerm' -count=1 -v` → exit 0，RouterDiet 13 PASS，diet 精确 27 条
- 权限: `go test ./internal/middleware -run AdminPerm -count=1 -v` → exit 0，15 PASS（view/export 码互不替代）
- 仓储: `go test ./internal/repository -run 'DietPlan|MealPlan' -count=1` → exit 0，`ok 8.448s`
- gofmt: 18 个 manifest Go 文件 → 无输出；Secret scan: 18 个 manifest 文件 → 0 findings
- Wire: `make wire` → exit 0，`wire_gen.go` sha256 前后不变（幂等）；`go.mod`/`go.sum` 未改
- Token: n/a（纯后端，无 UI/token 面）
- 正常态: 纯函数黄金回归 + 真实 MySQL 保存/分析；12 项营养、三个能量总量（1341.75/1274.65/1380.7）、85.88%、126% 钙、3/2 脂肪、钙/锌话术均复算一致
- 空态: service `ErrDietAnalysisEmpty` → handler 422「该食谱暂无明细，无法生成营养分析」（测试 + 代码核对；无 HTTP 级测试，见 P2）
- 错误态: 跨园 `ErrDietPlanNotFound` → 404；kgID 0 → 403（同上，无 HTTP 级测试）
- 交互: 真实 `SavePlan` → 断言快照含 `grain`/`plant` → 改食物 `group16`/`source_flags`/6 项营养 → 重分析，整份 7 组报表 JSON 全等；旧快照回退测试通过
- 导出: excelize 重开生成物 → 5 sheet；营养表 15 cells（标签+14 列）；构成 11 cells（类别+10 类）；脂肪来源仅 2 列；Sheet2 13 列；全簿无 retinol/carotin
- Console/截图/switchTab: N/A（纯后端）

## 范围

- 实际内容改动 = 13 modified + 5 untracked = 18 文件，与 worker manifest 完全一致。
- fix-brief 允许的 domain 改动仅两个文件，且只新增快照字段与复制逻辑；无 schema 变更（JSON 列）。
- brief 外 2 个文件判必要最小：`app/router_test.go`、`service/admin_album_test.go` 各 +1 `nil`（`NewHandler` 签名 +1），manifest 已声明。
- `server/` 其余 ` M` 为 Windows stat 脏位（`git diff --raw` 为空，内容 == HEAD）；非 server 脏文件/untracked 早于本 slice。
- 未改 `backend/api/`、`configs/`、`admin/`、`teacher/`、`parent/`、`go.mod`、`go.sum`、`.mcp.json`。

## P1 修复核验（Round 1 唯一 P1）

Round 1 P1：构成/钙铁/蛋白/脂肪来源表读食物库实时 `group16`/`source_flags`，改食物分类会改写历史报表。

- `DietNutritionSnapshot` 新增 `Group16`、`SourceFlags`（`domain/diet.go:200-203`）；`Food.NutritionSnapshot()` 复制（`domain/diet_food.go:66-74`）；S6 保存/复制/模板路径经 `dietNutritionSnapshotJSON` 自动携带，无迁移。
- `aggregateDietAnalysisItems` 优先读快照，逐字段空值才回退当前食物库（`service/diet_analysis_service.go:476-496`）；回退在代码/domain 注释、self-review、known_gaps 均写明。
- TC-0063 已用真实保存路径重写并覆盖分类+来源+营养值，`assert.JSONEq` 整份结果（7 组 + 结论）全等；`DietFoodRepo.Update` 用 `Select("*")` 真落库，测试非空转。
- 旧行快照无分类时回退食物库，有独立测试 `TestDietAnalysis_LegacySnapshotFallsBackToFood`。
- 结论：**P1 已修复，且有可失败回归测试。**

## P1 · 阻塞问题 (0)

无。

## P2 · 建议 (4)

### 1. [快照边界] 合法空分类仍走实时回退

- 证据: `service/diet_analysis_service.go:478-496`；`service/diet_food_service.go:656-673`（`小吃、甜饼 / 速食食品 / 饮料类 / 调味品类 / 其他` 无默认 source_flags）
- 说明: 空值既可能是旧行，也可能是保存时本来没有分类；这类食物日后补填 flags 会改变历史来源表。worker known_gaps #4「新建/重存完全抵抗分类漂移」对该子集略有夸大。建议后续用指针字段（nil=旧行缺失，""=合法空）或快照版本区分；非本轮 fix-brief 要求。

### 2. [契约] 响应快照字段未同步 SoT

- 证据: `backend/api/api-admin/23-diet.md:669-674` 仍写 `nutrition_snapshot` 为「27 项，同食物详情；缺测为 null」，现额外含 `group16`/`source_flags`
- 建议: additive 变更，fix-brief 禁止改 `backend/api/`；后续补契约说明或标记 internal。

### 3. [test gap] 无 HTTP 级分析/导出测试

- 证据: `app/router_diet_test.go:609-650` 只覆盖注册/权限；handler 的 200 二进制头、422、404 未走 HTTP
- 建议: 新增 `TestRouterDiet_AnalysisRealBehavior`（真实 plan：200 JSON / 200 xlsx headers / 422 / 404）。

### 4. [cross-slice] macro_energy 现状值需页面派生

- 证据: `backend/api/api-admin/23-diet.md:834-849`（契约示例为 DRIs 派生「要求」值）；`service/diet_analysis_export.go:348-382`（导出已同时写要求/现状）
- 建议: S8/S10 brief 写明 `nutrition.intake ×4/9/4` 派生，或后续扩契约；非 P1（契约示例本身如此）。

> 另：结论高值话术缺失属「无证据、正确省略」（样本铁「高」无建议）；图表 sheet「其他」与参考导出的 0 差异属上游不一致、契约未要求保留；旧行兼容路径与 `dri_targets` 缺失均为已文档化限制。以上均维持 Round 1 的 P2 分类。

## 结论

P1 已完整修复并通过真实 MySQL 回归证明；build/vet/23 条分析测试/S6 DietPlan 回归/13 条路由测试/27 条权限集合/15 条 AdminPerm/仓储/导出 5-14-10-2/黄金数值/gofmt/secrets/wire 幂等全部独立复跑通过，manifest 干净。仅余 P2 级观察项。**PASS。**
