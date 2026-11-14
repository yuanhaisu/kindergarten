# DGO-S3b · 对抗性实现审查

- verdict: PASS
- score: 90/100
- 源码覆盖: `server/internal/handler/http/diet.go`、`handler.go`、`app/router.go`、`app/router_diet_test.go`、`app/router_perm_test.go`、`app/wire_gen.go`、`middleware/admin_perm.go`、`middleware/admin_perm_test.go`、`repository/wire.go`、`service/diet_plan_service.go`、`service/diet_plan_test.go`、`service/wire.go`、新增 `service/diet_meal_config.go`；删除旧 config service/repo 及测试 4 个文件
- 审查范围: 内容 diff + untracked + wire 生成物 + 路由/权限表 + handler 调用链 + 计划创建入参 + 真实 MySQL 测试；后端 slice，页面/DevTools 不适用
- 基线: `aac8739` → 工作区未提交（HEAD 仍为 `aac8739`）

## DevTools 证据

- 构建: `cd server && make wire` → exit 0，`wire_gen.go` sha256 前后一致；`go build ./...` → exit 0；`go vet ./...` → exit 0
- Token: N/A（无样式/token 文件变更）
- Secret scan: `node scripts/check-secrets.mjs --files <13 files>` → 0 findings
- 正常态: 新增 201 + 6 餐次/系数 0.95/班级 join；列表按学年+关键词裁剪且只返回启用餐次名；详情与创建一致；PUT 返回保存后详情；周食谱创建带 `group_object_id`，快照携带对象 ID/名称/班级
- 空态: service TC0065 / ListWithoutCurrentYearReturnsEmpty —— 空学年返回空列表，读取路径不写库
- 错误态: 重名/无班级 422；不存在/跨园/跨学年对象拒绝且不落库；旧 GET/PUT `/diet/config` 均 404
- 交互: N/A（无页面）
- Console: N/A；路由: 4 条 group-object 路由注册、精确 diet 路由集 29 条、权限登记自检通过；截图: N/A（未驱动 IDE，未取锁）
- 实时 curl: 未执行。端口 8080 由 slice 前启动的 `bin/api.exe`（PID 40284，21:16 创建）持有；brief 允许端口占用时记 known_gap。等效证据为真实 gin engine + MySQL + RequirePermission 的 httptest。

## 附加验证

- `go test ./internal/app -run Router -count=1 -timeout 600s` → ok 41.588s；4 个新路由用例全部 PASS
- `go test ./internal/app -run 'TestAdminPermissionRegistryCoversAllRoutes|TestRouterPerm_DietConfigRoutesRegistered'` → PASS
- `go test ./internal/middleware -run AdminPerm -count=1` → ok 0.223s
- `go test ./internal/handler/http -count=1` → ok 0.199s
- `go test ./internal/service -run 'DietPlan|DietGroupObject|DietAnalysis' -count=1 -timeout 900s` → ok 477.181s（隔离运行；此前一次 DietAnalysis 失败因 supervisor 并发跑了同一 fixture 园所 990995，隔离复跑通过）
- `go test ./internal/repository -run DietGroupObject -count=1` → ok 17.581s
- `gofmt -l` 所有变更/新增文件 → 无输出；仓库基线 43 个 `internal/` 未格式化文件不在本 diff
- Scope: `server/` 内容 diff 精确为 12 M + 4 D + 1 untracked，与 worker manifest 一致；其余 56 个工作区差异 mtime 早于 slice，非本 slice 产生

## 语义核对

- 路由：GET/POST `/group-objects` + GET/PUT `/group-objects/:id` 真实注册；旧 `/diet/config` handler 与路由删除，404 已断言。
- 权限：`admin_perm.go` 4 条登记（view/manage 各 2），旧 2 条移除；fail-closed 自检通过；Casbin `admin` 通配无需改。
- Handler：仅绑定、ID 解析与错误映射；学年/班级/餐次校验仍在 service；422 文案原样透出、404/500 不泄露内部。
- 周食谱：创建入参新增 `group_object_id` 并透传；缺对象/不存在 422 路由层验证，跨园/跨学年 service 层验证。
- Wire：旧 config repo/service provider 与构造参数全部移除；`NewHandler` 聚合 `dietGroupObjectSvc`；生成物与提交内容一致。
- 校验抽离：新旧文件 diff 仅删除单例专属类型/方法，保留校验逻辑逐字一致，错误文案未漂移；残留引用 grep 无命中。

## P1 · 阻塞问题 (0)

无。

## P2 · 建议 (5)

### 1. [error-code] 重名返回 422，brief 首选 409
- 证据: `server/internal/handler/http/diet.go:155`；`router_diet_test.go` 断言重名 422；service 以通用校验错误报重名。
- 建议: S4 契约明确采用 422，或增加 typed conflict error 后映射 409。

### 2. [brief-manifest] 两处相关文件在 brief 允许清单外
- 证据: `app/router_perm_test.go`（精确路由集合 gate 必需）与新增 `service/diet_meal_config.go`（design D8）；均已在 worker manifest 申报。
- 建议: main session 按 worker manifest 提交，修正后续 brief Scope。

### 3. [test-coverage] 删除 `diet_meal_config_test.go` 丢失存活校验边界用例
- 证据: 被删用例覆盖 HH:mm、时间递增、相邻上限、未知比例、无启用餐次、餐次缺失/重复；这些校验仍被 group-object/analysis 使用。
- 建议: S4/S6 迁移纯校验用例，锁定 TC-NEW-DIE-0003..0007。

### 4. [staging] admin 基本参数页与契约文档仍指向已删端点
- 证据: `admin/api/diet.ts:44,48`、`admin/pages/diet/config.vue:193,240`、`backend/api/api-admin/23-diet.md:21-22,166,193`。
- 建议: commit message 注明中间态；S4/S5 同批落地前不发布。

### 5. [contract-gap] 周食谱列表缺可选 `group_object_id` 筛选
- 证据: `listDietPlansQuery` 无该字段；design D3 标为可选，仓储不在 manifest。
- 建议: S4 按可选处理或后续补仓储筛选。
