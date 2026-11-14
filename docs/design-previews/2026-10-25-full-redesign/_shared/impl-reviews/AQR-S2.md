# AQR-S2 · 对抗性实现审查

- verdict: **PASS**
- score: 92/100
- 源码覆盖: `domain/diet_config.go`、`domain/diet.go`（+1 行快照字段）、`repository/diet_config_repo.go`、`service/diet_config_service.go`、`service/diet_meal_config_test.go`（新增）、`handler/http/diet.go`、`app/router_diet_test.go`
- 审查范围: diff vs base + untracked + manifest + spec/design/`api-admin/23-diet.md`（2026-11-02 决策 B 版）+ OpenSpec apply guidance + 路由/权限/Wire + DB 调用链
- 基线: `0a306248ec091c797bff1be7c37125970be250fb`（HEAD 未前移）
- 环境: 后端切片；按编排器指示未使用微信 DevTools

## 证据（独立复跑）

- 构建: `cd server && go build ./...` → exit 0
- Service: `go test ./internal/service -run DietMealConfig -count=1 -v` → exit 0，15 顶层测试 PASS / 0 SKIP（真实 MySQL 45.1s）
- Router: `go test ./internal/app -run RouterDiet -count=1 -v` → exit 0，3 PASS（真实 gin + 权限中间件 + MySQL）
- 回归: repository `-run 'DietConfig|MealConfig'`、middleware `-run AdminPerm`、app `-run Router`、handler 全包 → 全部 ok
- Vet/Gofmt: `go vet ./...` → exit 0；7 个 slice 文件 `gofmt -l` 无输出
- Wire: `cd internal/app && wire diff` → exit 0 无 diff（构造签名未变）
- Secret scan: `node scripts/check-secrets.mjs --files <7 files>` → 0 findings
- 502 清理: `grep -n "501\|NotImplemented"` 两个实现文件 → exit 1（无匹配）
- Token / Console / 截图: N/A（后端切片，无页面）

## 行为核对

- **单例 + 默认值**：无配置行时返回日托/standard/5 餐启用/0.95，读取零写入、重复读取一致；保存走事务，按 kg 更新或插入，`Unscoped` 硬删重插 6 餐次。
- **决策 B**：standard 跳过相邻上限、只要求已启用餐次等于固定 25/5/35/5/25/5（因此 35+5=40 与默认 0.95 合法）；custom 校验占比>0 与 29/39/30 上限（日托早+早点 29，全托 30）。
- **通用校验**：时间格式 HH:mm、已启用时间严格递增、至少一餐启用、6 键齐全唯一、`market_weight` 唯一口径、`day_care/full_care`、`standard/custom`。
- **系数与快照**：系数 = 已启用占比和 ÷100；`BuildConfigSnapshot` 含 care/food/ratio/coefficient/meals，DRIs 留空给 S6/S7。
- **不回溯**：`DietPlanRepo` 落一条 `config_snapshot` 后改配置，读回 JSONEq 不变。
- **契约/权限**：响应字段与契约一致（含 labels，不含 kindergarten_id）；GET=`diet:view`、PUT=`diet:manage`，互不替代；错误统一 422/42200 或 500/50000。

## TC 映射

TC-0001..0008 均有同名测试并通过：首读默认 / 标准保存读回 / 两种模式时间倒序拒绝 / custom 29·30·39 超限 / 0.95 / 停晚餐 0.70 / 可食口径拒绝 / 快照不回溯。

## P1 · 阻塞问题 (0)

无。

## P2 · 建议 (3)

1. [scope] `domain/diet.go:161` 为 brief 清单外 1 行；`DietConfigSnapshot.RatioMode` 是 Required behavior 8 的必需字段，判 necessary-minimal、已披露，接受。
2. [semantics] `service/diet_config_service.go:352` 的 custom 相邻上限把停用餐次占比也计入；契约第 6 条未限定「已启用」，不算违约，建议契约明确口径。
3. [test] 晚餐+晚点 ≤30 无独立用例；实现与其它两组同构，可延后补。

## 结论

10 条 acceptance 独立复跑全绿；manifest 与 diff 一致；唯一清单外改动为必要最小且已披露。判 **PASS**，可 commit。
