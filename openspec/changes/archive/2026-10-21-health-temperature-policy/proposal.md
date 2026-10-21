# Proposal

## Why

当前晨检体温策略是写死的：服务端 `feverThresholdCelsius = 37.3`，体温对所有园所都是选填。但不同园所的流程和迎检要求不同——有的要求每个孩子都录体温，有的只在摸额头异常时才测。单一策略要么逼所有人做无谓录入、拖慢早高峰，要么让要求全员测温的园所拿不出完整记录。

体温策略必须是**园所级配置**，默认选填（optional），需要时由园所管理员打开全员必测。

## What Changes

- **园所设置增加 `health` 块**（`KindergartenAppSetting.ExtraConfig` JSON，无需迁移）：
  - `temperature_required`：布尔，默认 `false`（按需测温）
  - `fever_threshold`：数值，默认 `37.3`，允许范围 `35.0–42.0`
- **管理端**：`GET/PUT /admin/settings` 读写该块；设置页增加「要求全员测温」开关与「发烧阈值」输入。
- **医生端**：`GET /doctor/health/queue` 响应增加 `policy` 对象（与设置同字段），医生页据此显示「本园按需测温 / 全员必测」，并把体温字段标为选填或必填。
- **服务端判定**：
  - 发烧标签按园所阈值派生，不再写死 37.3。
  - `temperature_required=true` 时，已完成的晨检必须带体温，否则拒绝；`skip`（未到园）豁免。
  - `temperature_required=false` 时允许不录体温，存 NULL，**不得伪造正常值**。
  - 策略变更只影响新提交，不回溯已完成的记录。
- **契约变更声明**：本次将修改 `backend/api/api-admin/18-settings.md` 与 `backend/api/api-doctor/01-health.md`，属于契约变更。

## Out of Scope

- 按期必填（scheduled：迎检期/传染病期自动强制），本期只做常驻开关
- 蓝牙体温计、热成像等设备集成
- 一键健康 / 单卡步进等晨检交互重构
- 测温覆盖率统计与报表
- 家长端「未测温」文案（家长页已对空体温不展示，本期不改）
- 已完成记录的当日修正/审计
- 对既有 `symptom_tags` 的回填重算

## Capabilities

### New Capabilities

- `health-temperature-policy`: 园所级晨检体温策略（是否必填、发烧阈值）、策略下发与提交校验。

### Modified Capabilities

（无。`morning-health-closed-loop` 仍在活跃变更中，尚未进入 `openspec/specs/` 活基线；本变更以新能力方式叠加，不修改其 delta。）

## Open Decisions / Assumptions

1. **用布尔 `temperature_required` 而不是 `optional|required|scheduled` 枚举**：用户明确先按 optional 落地；布尔足够表达当前两态，也避免为未来 scheduled 提前发明枚举值。后续要按期必填时再演进为带时间窗的策略。
2. **队列响应携带 policy，而不是新增独立端点**：医生页打开队列时一次拿到，避免策略与队列不一致；属于对既有响应包络的追加字段。
3. **旧的 `ExtraConfig` 无 `health` 块时按默认值解析**（false / 37.3），存量园所行为不变。

## Impact

- **后端**：`service/settings_service.go`（payload/默认值/校验/`GetHealthPolicy`）、`service/health_check_service.go`（阈值、必填校验、策略依赖）、`handler/http/health_check.go`（队列下发 policy）、`internal/app/wire.go` + `wire_gen.go`
- **契约**：`backend/api/api-admin/18-settings.md`、`backend/api/api-doctor/01-health.md`
- **管理端**：`admin/pages/settings/index.vue`、`admin/types/api.d.ts`
- **老师端**：`teacher/src/pages/doctor/checkin.vue`、`teacher/src/api/doctor-health.js`
- **家长端**：不改
- **数据**：无迁移；`ExtraConfig` 为 JSON，新增块向后兼容
