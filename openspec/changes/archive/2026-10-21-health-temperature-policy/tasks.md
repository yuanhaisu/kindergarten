# Tasks

## 1. 后端 · 园所策略存储与读取

- [x] 1.1 在 `settings_service.go` 增加 `settingsHealth`（`temperature_required` / `fever_threshold`），并入 `settingsPayload`、`defaultSettingsPayload`（false / 37.3）与 `parsePayload`（缺失/非法归一化）；验证：单测覆盖空 JSON、旧五块 JSON、`health` 为 0 值时都得到默认策略
- [x] 1.2 `SystemSettingsResp` 与 `UpdateSettingsReq` 暴露 `health` 块；`UpdateSettings` 校验阈值范围 35.0–42.0，越界返回明确错误；验证：单测覆盖有效保存与 34.9/42.1 被拒
- [x] 1.3 增加只读 `GetHealthPolicy(ctx, kgID)`：无配置行返回默认，不写库；验证：单测断言读取前后 `kindergarten_app_settings` 行数不变
- [x] 1.4 验证：`go test ./internal/service -run Settings -count=1 -timeout 600s` 通过；对照 specs 的 HTP-0001~0003

## 2. 后端 · 晨检阈值与必填校验

- [x] 2.1 定义 `HealthPolicyProvider` 接口，`SettingsService` 实现；在 `service/wire.go` 增加 `wire.Bind`；验证：`make wire` + `go build ./...` 无 `no provider found`
- [x] 2.2 `HealthCheckService` 依赖该接口；`deriveSymptomTags` 增加 threshold 参数，`SubmitCheck` 按园所阈值派生标签；验证：单测覆盖 37.2→发烧（阈值 37.2）与 37.2→不发烧（阈值 37.5）
- [x] 2.3 必填策略校验：`required && temperature == nil && !skip` 返回 `ErrHealthCheckTemperatureRequired`；handler 映射 422；验证：单测覆盖缺体温被拒、有体温通过、skip 豁免
- [x] 2.4 可选策略下允许不录体温且保持 NULL，不写默认正常值；验证：单测断言 `temperature IS NULL`
- [x] 2.5 `DoctorGetQueue` 响应增加 `policy`（读取失败返回 500，不返回错误策略）；验证：单测/接口断言 `list` 与 `policy` 同时返回
- [x] 2.6 验证：`go build ./...`、`go test ./internal/service -run "HealthCheck|Settings" -count=1 -timeout 600s` 通过；对照 specs 的 HTP-0006~0012

## 3. 契约

- [x] 3.1 更新 `backend/api/api-admin/18-settings.md`：GET/PUT 增加 `health` 块，注明默认值与阈值范围；验证：字段与 `settingsHealth` 一致
- [x] 3.2 更新 `backend/api/api-doctor/01-health.md`：队列响应增加 `policy`，PUT 补充必填模式的 422；验证：字段与 handler 响应一致
- [x] 3.3 验证：契约变更已在 proposal 声明；未新增无法溯源的字段

## 4. 管理端

- [x] 4.1 在 `admin/types/api.d.ts` 的 `SystemSettings` 增加 `health` 类型；验证：`npm run build` 类型检查通过
- [x] 4.2 在 `admin/pages/settings/index.vue` 增加「要求全员测温」开关与「发烧阈值」输入（35–42，步长 0.1）；验证：保存后重新进入页面值保持
- [x] 4.3 验证：`npm --prefix admin run build` 通过；不新增硬编码色值

## 5. 老师端

- [x] 5.1 `teacher/src/api/doctor-health.js` 注释队列响应包含 `policy`；验证：调用方字段名一致
- [x] 5.2 `checkin.vue` 的 `loadQueue` 保存 `policy`，默认 `false / 37.3`；页面显示「本园按需测温」或「本园要求全员测温」；验证：两种策略下文案切换
- [x] 5.3 体温字段按策略标「选填/必填」，发烧提示改用 `policy.fever_threshold`；`doSubmit` 在必填且未录体温时先拦截；验证：必填模式下空体温不发出请求
- [x] 5.4 新增 UI 只用既有 token；验证：`node scripts/check-tokens.mjs teacher` 中 `checkin.vue` 无新增违规
- [x] 5.5 验证：`npm --prefix teacher run build:mp-weixin` 通过；真实点击验证两种策略下的提示与提交（实测：required 下页面显示「本园要求全员测温」、体温标「必填」，空体温点健康被拦截并提示「本园要求录入体温」，记录保持待检；optional 下显示「本园按需测温（体温选填）」、体温标「选填」，空体温提交成功且 temperature 为 NULL）

## 6. 测试补录与集成验证

- [x] 6.1 在 `docs/tests/TEST_CATALOG.md` 的晨检模块下登记 `TC-NEW-HTP-0001` 至 `0012`；验证：编号与 specs 引用一一对应
- [x] 6.2 集成验证：管理端打开全员必测 → 医生队列下发 required → 不录体温提交返回 422 → 录入后提交成功；关闭后空体温可提交且存 NULL（实测 422「本园要求全员测温，请先录入体温」；37.6 提交成功并派生 fever；optional 下 temperature 为 NULL）
- [x] 6.3 阈值验证：把阈值改到 37.2/37.5，分别提交 37.2°C，断言发烧标签有无符合预期；旧记录标签不变（实测 37.2→fever、37.5→无标签；旧记录 995020 仍为 fever）
- [x] 6.4 验证：`make -C server check`（含 `-race`，使用 64 位 `CC`）通过；杀掉残留 watcher；逐条对照 specs 全部 12 个场景（实测 service 285s 全 PASS；顺带修正一条日期相关的课表测试：手动事件与课表实例同日导致 dots 断言误判）
