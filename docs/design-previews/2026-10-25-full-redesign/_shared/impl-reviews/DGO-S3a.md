# DGO-S3a · 对抗性实现审查

- verdict: PASS
- score: 91/100
- 源码覆盖: `server/internal/service/diet_plan_service.go`、`diet_plan_snapshot.go`、`diet_plan_test.go`、`server/internal/app/wire_gen.go`
- 审查范围: 内容 diff + untracked + wire 生成物 + 调用链（ListPlans/GetPlan/CreatePlan/buildDietConfigSnapshot/dietDriWeights）+ S2 快照契约；后端 slice，页面/DevTools 不适用
- 基线: 4dcff90 → 4dcff90（工作区未提交）

## DevTools 证据

- 构建: `cd server && go build ./...` → exit 0
- Wire: `cd server && make wire` → exit 0，`wire_gen.go` sha256 前后一致（ee2d27de…40e0f）
- Token: N/A（无样式/token 文件变更）
- Secret scan: `node scripts/check-secrets.mjs --files <4 files>` → 0 findings
- 正常态: TC-NEW-DIE-0073 列表/详情按行返回对象 ID + 名称；TC-NEW-DIE-0075 对象改名后旧食谱详情保留快照原名，列表显示当前名
- 空态: 软删后列表 total=0 且 slice 为空；对象校验失败不落库
- 错误态: TC-NEW-DIE-0071/0072 缺对象、对象不存在、跨园、跨学年均拒绝且零写入；既有周期/方式/学年校验回归通过
- 交互: N/A（无页面）
- Console: N/A
- 路由: N/A（handler/router 归 S3b）
- 截图: N/A（后端 slice；未驱动 IDE，未取 wechat-devtools 锁）

## 附加验证

- `go test ./internal/service -run 'DietPlan' -count=1 -timeout 900s -v` → ok 330.399s，23 个顶层测试 PASS；新增 4 个测试（TC-0069/0071/0072/0073/0075/0076 覆盖）全部 PASS
- `go test ./internal/service -run 'DietGroupObject' -count=1 -timeout 600s` → ok 122.498s
- `DietAnalysis` → ok 25.076s；`DietConfig|DietMealConfig` → ok 46.442s
- `go vet ./internal/service` clean；`gofmt -l` 4 个变更文件无输出
- Scope: `server/` 内容 diff 仅 4 个 manifest 文件（3 service + wire_gen）；无新增/删除/untracked；其余工作区改动 mtime 早于 slice 开始，非本 slice 产生

## 语义核对

- 创建：`group_object_id` 必填；本园 + 同学年双重校验；错误文案与 brief 一致；校验先于写库。
- 快照：改用 `DietGroupObjectService.BuildConfigSnapshot`，对象 ID/名称/就餐班级随快照固化；DRIs 只按快照班级集合加权，空集合回退全部班级，无可用学生回退 0.267/0.326/0.408/0。
- 列表：对象名按去重 ID 回查；页大小上限 100，查询有界；对象缺失时留空不阻断。
- 详情：名称优先快照（改名后仍显示原名），快照缺失再回查当前对象；旧快照零值解析向后兼容。
- `MealPlan.GroupObjectID` 落库；`DietPlanDetail`/`DietPlanListItem` 新字段带 JSON tag，handler 直出即可。

## P1 · 阻塞问题 (0)

无。

## P2 · 建议 (3)

### 1. [cross-slice] `internal/app/router_diet_test.go:116` 仍用 7 参构造器

- 证据: `go vet ./internal/app` 报参数不足；`go build ./...` 通过。
- 建议: S3b 同批修正构造器调用并补 `group_object_id`，跑 `go test ./internal/app -run Router`。本 slice manifest 不含该文件。

### 2. [staging] handler 未透传 `group_object_id`

- 证据: `server/internal/handler/http/diet.go:638-660` 的请求结构体与 `toInput()` 无该字段，真实创建接口暂返 422。
- 建议: S3b 接字段；commit message 注明中间态。

### 3. [spec-drift] 回退权重算例数值不一致（既有问题）

- 证据: `openspec/changes/add-diet-group-objects/specs/diet-nutrition-analysis/spec.md:12` 为 1484.26/50.7/746.61，实际种子+公式为 1485.95/50.76/747.4；AQR-S6 已记录。
- 建议: DGO-S6 最终验收时校准 spec 或种子，不在本 slice 扩围。
