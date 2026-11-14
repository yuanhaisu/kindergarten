# AQR-S6 · 对抗性实现审查

- verdict: **PASS**
- score: 89/100
- 源码覆盖: `service/diet_plan_service.go`、`service/diet_plan_snapshot.go`、`service/diet_plan_test.go`（均新增）、`handler/http/diet.go`、`handler/http/handler.go`、`app/router.go`、`app/wire_gen.go`、`middleware/admin_perm.go`、`repository/diet_plan_repo.go`、`service/wire.go`、4 个测试文件
- 审查范围: diff + untracked + 调用链（router → handler → service → repo）+ 权限表 + D2/D3/D5 + TC-0028..0043 + 独立算例
- 基线: `338c496395721ad28e24ae7985d7f7985970db26` → 工作区未提交（后端 slice，无 DevTools）

## 验证证据（后端 slice，无 DevTools）

- 构建: `cd server && go build ./...` → exit 0，无输出
- vet: `go vet ./...` → exit 0，无输出（含测试文件类型检查）
- service: `go test ./internal/service -run DietPlan -count=1 -v` → exit 0，`ok 204.017s`；19 顶层 PASS + 15 子测试，真实 MySQL 未 skip
- 路由: `go test ./internal/app -run RouterDiet -count=1 -v` → exit 0，11 PASS；8 条新路由真实注册，diet 精确 25 条
- 权限: `go test ./internal/app -run RouterPerm -count=1` → exit 0；另跑 `TestAdminPermissionRegistryCoversAllRoutes` → PASS（fail-closed 启动校验含新路由）
- 中间件: `go test ./internal/middleware -run AdminPerm -count=1 -v` → exit 0，13 PASS（新增 2 条）
- 仓储: `go test ./internal/repository -run 'DietPlan|MealPlan|DietTemplate' -count=1 -v` → exit 0，3 PASS
- handler 包回归: `go test ./internal/handler/... -count=1` → exit 0
- gofmt: 15 个 slice Go 文件 → 无输出；Secret scan: 15 个 manifest 文件 → 0 findings
- Wire: `make wire` → exit 0；`wire_gen.go` sha256 `344305e0…049e7b` 执行前后不变（幂等）
- 正常态: 真实 MySQL 覆盖自定义/模板/复制三种创建、列表、详情、整单保存、清空、软删除；HTTP 201/200/404/409 实测
- 空态: custom `day_num=0`；无当前学年列表返回 `[]`；空明细/模板返回 `[]` 而非 null
- 错误态: 422 克数 0/负、菜名空、餐次未启用、日期越界、跨园食物；404 跨园/已删；409 同学年重名
- 交互: 路由级真实请求 create → get → clear → delete；保存/合并/排序由 service 集成测试覆盖
- Console/截图/switchTab: N/A（纯后端，无页面）

## 范围

- `git diff --name-only 338c496 -- server/` = 12 个修改；untracked server = 3 个新增；无 staged；与 worker manifest 完全一致。
- brief 外 2 个文件判为必要最小：`app/router_test.go`、`service/admin_album_test.go` 各 +1 个 `nil` 实参（`NewHandler` 签名 +1），manifest 已声明，符合 brief 的「因签名变化必需的测试调用点」。
- 51 条 server ` M` 为 stat 脏位（内容 hash 与 HEAD 相同）；非 server 脏文件（GLOSSARY/package.json/docs/tokens）mtime 均早于本 slice 窗口，确认非 S6 改动。
- 未改 `domain/`、`backend/api/`、`configs/`、`admin/`、`teacher/`、`parent/`、`.env`、`.mcp.json`。

## 行为与 spec 核对

- 列表: 缺省学年取 `is_current`；无当前学年返回空列表；keyword/分页；字段含 `created_method_label`。
- 创建: 周一 + 结束严格 +6；学年本园；同名校验；三种方式附加字段互斥；custom `day_num=0`。
- 模板: 复制条目；缺失/停用/非法克数食物跳过并写 warnings；模板只读，生成结果修改后回读模板不变（TC-0033）。
- 复制: 按 `weekday`（缺失时由 date 推导）映射；用当前食物库重算浓度快照；已删食物跳过并 warnings。
- 详情: config_snapshot 只读；weekday 由 date 推导；items 按「日期 → 餐次 → sort → id」稳定排序。
- 保存: `items` 缺失 422、空数组清空；日期区间/启用餐次/菜名/克数/本园食物校验；同 `(date,meal_key,recipe_name)` 同食物合并；服务端 food_name；sort 按数组顺序在 `(date,meal_key)` 内重写；day_num=有明细的不同日期数；改名撞同学年 409。
- 清空/删除: 清空保留食谱、day_num=0；软删除后列表/详情 404；明细硬删避免软删行累积。
- 权限: 8 条逐条登记；GET→`diet:view`，写操作→`diet:manage`；view/manage 互斥；fail-closed 启动校验通过。
- 快照冻结: 唯一写入点是 `CreatePlan`；`SavePlan/ClearPlan/UpdateWithItems/Update` 均不写 `config_snapshot`（代码核对）。
- DRI 算例（独立复算）: 3/4/5 人 → 权重 0.25/0.3333/0.4167/0；能量 1487.505→1487.51；蛋白 50.8335→50.83；钙 750。无学生回退 0.267/0.326/0.408/0 → 能量 1485.95、蛋白 50.76、钙 747.4；11 项齐全。
- 营养公式（独立复算）: 30g × 食部 100% × 344/100g ÷ 100 = 103.2；20g → 68.8；浓度快照保持 344；`protein` nil 序列化为 `null`，未转 0。

## P1 · 阻塞问题 (0)

无。

## P2 · 建议 (9)

1. **[transaction]** 保存的读取/校验/合并/快照在事务外，事务只包住「表头更新 + 明细替换」；design D5 字面要求全流程在一个事务内。无部分写入，最后写入胜出；仅并发删除被引用食物/计划时有窗口。`service/diet_plan_service.go:422-437,529`。
2. **[concurrency]** 名称唯一仍为读后写，无唯一索引；并发创建/改名可同名。design D2 明确取舍，与 S3/S4 同源。`service/diet_plan_service.go:278-287,517-525`；`repository/diet_plan_repo.go:50-73`。
3. **[data]** DRI 表未播种/播种失败时 `config_snapshot.dri_targets=[]`，权重仍回退；启动 seed 失败只记日志。`service/diet_plan_snapshot.go:46-47,56-73`；`app/provider.go:189`。
4. **[contract edge]** 模板创建 `day_num` 取实际有明细的不同日期数，而非 `TemplatePlan.day_num`；某天条目全被跳过时偏小。`service/diet_plan_service.go:323`；`service/diet_plan_test.go:467`。
5. **[dead code / S7]** `dietNutritionAmount` 仅测试使用；生产保存存每 100g 浓度快照（与契约 detail 一致），S7 必须复用该口径。`service/diet_plan_snapshot.go:161`；调用点仅 `service/diet_plan_test.go:923,944,954`。
6. **[message]** self-review 的「always include dish + food」偏宽：空菜名/超长菜名/日期格式错误消息不含两者；TC-0037 克数用例已含。`service/diet_plan_service.go:448,460,463`。
7. **[test gap]** 无 HTTP 级 PUT 保存测试（422/409/成功包络）、无「创建后改配置/名册仍读旧快照」回归、无 keyword/学年过滤测试。`app/router_diet_test.go:549` 只覆盖 create/get/clear/delete。
8. **[cross-slice]** S7 spec 的回退期望值（能量 1484.26/蛋白 50.7/钙 746.61）与 S5 种子 + S6 加权结果（1485.95/50.76/747.4）不一致；S7 需校准期望值或种子。`openspec/changes/add-quantified-recipe/specs/diet-nutrition-analysis/spec.md:48`；`server/internal/service/diet_seed_service.go:428`。
9. **[test assertion]** `TestDietPlanSaveRecomputesSnapshotAndFoodName` 的客户端 `FoodName` 与服务端更新后名称相同，无法证明「以服务端为准」；蛋白断言期望 50.84，独立复算 50.83（0.02 容差掩盖 0.01 偏差）。`service/diet_plan_test.go:756-786`；`service/diet_plan_test.go:1000` 附近。

## 结论

全部 acceptance gate 独立复跑通过；manifest 与工作区一致；三种创建方式、模板只读、星期映射、同食物合并、排序持久化、清空/软删、422/409/404、DRI 11 项与营养黄金值均有真实 MySQL 断言。无 P1；9 条 P2 均为已披露取舍、测试补强或跨 slice 校准项，判 **PASS**，不阻塞提交。
