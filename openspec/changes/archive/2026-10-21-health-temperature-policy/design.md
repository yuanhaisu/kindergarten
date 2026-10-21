# Design

## Context

See `proposal.md` — Why.

现有基础设施与约束：

- 园所级配置已存在：`KindergartenAppSetting.ExtraConfig` 是一个 JSON 文本，`SettingsService` 在其中维护 `attendance / notice / growth / pickup / points` 五个块；管理端 `GET/PUT /admin/settings` 与设置页已就绪。
- 晨检服务当前把发烧阈值写死为 `feverThresholdCelsius = 37.3`；`SubmitHealthCheckReq` 的体温是可空指针，数据库列可空。
- 医生队列由 `HealthCheckService.GetQueue` 返回，handler 包成 `{ list }`；`Handler` 已持有 `settingsSvc`。
- `HealthCheckService` 当前依赖 `db / HealthCheckRepo / MessageRepo / FileRepo`，通过 Wire 注入。
- 上一个变更 `morning-health-closed-loop` 仍是活跃变更，其 `morning-health` 能力尚未进入 `openspec/specs/` 活基线；本变更以新能力叠加，不改它的 delta。

## Goals / Non-Goals

**Goals**

- 园所可开关「全员必测」，默认关闭（按需测温）。
- 发烧阈值园所可配，默认 37.3°C。
- 医生端能看到当前策略，并按策略校验与提示。
- 必填策略由服务端强制执行，客户端预校验只是体验层。

**Non-Goals**

- 按期（迎检期/传染病期）自动切换策略
- 体温计/热成像设备集成
- 晨检一键健康、单卡步进等交互重构
- 测温覆盖率统计
- 已完成记录的回溯重算或当日修正

## Decisions

### D1：用布尔 `temperature_required`，不用 `optional|required|scheduled` 枚举

用户明确先按 optional 落地。布尔足够表达当前两态，也避免为尚未决定的 `scheduled` 发明枚举值。未来要做按期必填时，再演进为带时间窗的策略对象；届时才是真正的产品决策点。

### D2：存进 `ExtraConfig` JSON，不新增表/列

`settingsPayload` 增加：

```go
type settingsHealth struct {
    TemperatureRequired bool    `json:"temperature_required"`
    FeverThreshold      float64 `json:"fever_threshold"`
}
```

默认 `false / 37.3`。存量行的 JSON 没有 `health` 块，`parsePayload` 补默认值，行为不变。无迁移、无 AutoMigrate 风险。

### D3：医生队列响应携带 policy

在 `GET /doctor/health/queue` 的 data 包络里增加：

```json
"policy": { "temperature_required": false, "fever_threshold": 37.3 }
```

理由：医生页打开队列时必须同时知道策略，否则要么多一次请求、要么出现「策略已改但页面还是旧值」。policy 是页面级元数据，不是列表项字段，放在 `list` 同级。列表项形状不变。

### D4：`HealthPolicyProvider` 接口 + Wire Bind

`HealthCheckService` 不直接依赖 `*SettingsService`，而是依赖一个小接口：

```go
type HealthPolicyProvider interface {
    GetHealthPolicy(ctx context.Context, kindergartenID uint64) (HealthPolicy, error)
}
```

`SettingsService` 实现它。按项目既有约定，在 `internal/service/wire.go` 增加 `wire.Bind(new(HealthPolicyProvider), new(*SettingsService))`，否则 Wire 报 `no provider found`。

好处：提交路径的测试可以用一个 stub 直接控制策略，不必构造依赖众多的 `SettingsService`。

### D5：`GetHealthPolicy` 只读，缺失即默认

现有的 `ensureAppSetting` 会在读取时建行；晨检提交是高频写路径，不应为了读策略而插入配置行。`GetHealthPolicy` 只做 `FindByKindergartenID`：

- 行不存在 → 返回默认 `false / 37.3`
- 行存在 → 解析 JSON，缺失/非法值归一化为默认
- 数据库错误 → 向上返回，由 handler 报错

### D6：服务端权威校验，客户端预校验

- 服务端在合并请求字段后、条件更新前判断：`required && check.Temperature == nil` → 拒绝（skip 提前返回，天然豁免）。
- 老师端在 `doSubmit` 前做同样的预校验，避免一次无谓请求；但服务端的 422 才是最终裁决。
- 新增错误 `ErrHealthCheckTemperatureRequired`，handler 映射为 422。

### D7：阈值参与标签派生，不回溯旧记录

`deriveSymptomTags(check)` 改为 `deriveSymptomTags(check, threshold)`；`SubmitCheck` 先取策略再派生。已完成的旧记录 `symptom_tags` 不重算——阈值变更只影响新提交，避免在读写路径上做批量重算。

### D8：按需测温时的空值是 NULL，不是「正常」

- 可选模式下未录入 → `temperature` 存 NULL，绝不预填 36.5。
- 医生端体温输入在可选模式下标注「选填」；必填模式下标注「必填」。
- 家长端现有逻辑对 NULL 不展示体温，本期不改。

## 跨端触点清单

| 层 | 是否修改 | 说明 |
|---|---|---|
| Go domain | **不改** | `ExtraConfig` 是 JSON 文本，不新增 struct 字段；`MorningHealthCheck` 不变 |
| repository | **不改** | 复用 `KindergartenAppSettingRepo.FindByKindergartenID` |
| service | **会改** | `settings_service.go`：`settingsHealth`、默认值、解析归一化、更新校验、`GetHealthPolicy`；`health_check_service.go`：策略接口、阈值参数、必填校验 |
| handler | **会改** | `health_check.go`：队列响应增加 `policy`；提交错误映射增加体温必填 |
| Casbin policy | **不改** | 管理端 settings 与医生队列的既有策略已覆盖，无新增路由 |
| router.go | **不改** | 不新增端点 |
| backend/api/ 契约 | **会改（契约变更）** | `api-admin/18-settings.md` 增加 health 块；`api-doctor/01-health.md` 队列响应增加 policy、提交错误补充 |
| 端页面 admin | **会改** | `pages/settings/index.vue` 增加开关与阈值；`types/api.d.ts` 增加 health 类型 |
| 端页面 teacher | **会改** | `pages/doctor/checkin.vue` 显示策略、阈值提示、必填校验；`api/doctor-health.js` 注释响应形状 |
| 端页面 parent | **不改** | 空体温已不展示，无新字段 |
| design token | **不改** | 策略提示复用现有语义色与字号，不新增 token |
| 跳转 | **不改** | 本期不新增页面跳转；医生页仍在老师端非 tabBar 体系内，不涉及 switchTab |

## 契约字段清单（溯源）

无设计稿覆盖园所设置页与医生晨检页，以下字段以明确业务场景溯源。

### `GET/PUT /admin/settings` 新增 `health` 块

| 字段 | 变化 | 溯源 / 为什么不能派生 |
|---|---|---|
| `health.temperature_required` | 新增 bool | 管理端设置页「要求全员测温」开关；用户明确先按 optional（false）落地 |
| `health.fever_threshold` | 新增 number | 管理端「发烧阈值」输入；默认值 37.3 来自既有常量 `feverThresholdCelsius` |

配置是用户设置的状态，无法从晨检记录派生，必须显式存储。

### `GET /doctor/health/queue` 新增 `policy`

| 字段 | 变化 | 溯源 / 为什么不能派生 |
|---|---|---|
| `policy.temperature_required` | 新增 bool | 医生页要显示「本园按需测温 / 全员必测」并做预校验；客户端无法从队列数据推断 |
| `policy.fever_threshold` | 新增 number | 医生页发烧提示使用园所阈值；客户端不应硬编码 37.3 |

不新增其他字段；列表项 `list[]` 形状不变，避免把策略复制到每一行。

## 数据与迁移

- 无表结构变更，无 `AutoMigrate` 需求。
- `ExtraConfig` 旧值缺少 `health` 时按默认解析；管理员首次保存后写入新块。
- 若 JSON 损坏或 `fever_threshold` 越界，读取时归一化为 37.3；管理端保存时显式拒绝越界值。

## 流程

```
管理员设置页
  → PUT /admin/settings { health: { temperature_required, fever_threshold } }
  → SettingsService 校验范围 → 写入 ExtraConfig

医生打开晨检页
  → GET /doctor/health/queue
  → { list, policy }
  → 页面显示「按需测温 / 全员必测」，体温框标选填/必填

医生提交
  → PUT /doctor/health/check/:id
  → 读取 policy
  → skip? → 直接标记未到园（豁免体温）
  → required && temperature == nil? → 422，不落库
  → deriveSymptomTags(check, fever_threshold)
  → 条件更新 → 家长通知
```

## Risks / Trade-offs

- **中途切换策略**：当天已打开但未提交的记录会在提交时按新策略校验。若从 optional 切到 required，未录体温的记录会被拒。UI 已显示当前策略；管理员应在检查开始前切换。
- **阈值变更不回溯**：同一天内阈值调整后，新记录用新阈值、旧记录标签不变，统计可能混合两种阈值。这是为了避免写路径上的批量重算；需要在报表口径中说明。
- **旧版小程序**：客户端若忽略 `policy`，必填模式下会收到 422。老师端与后端同批发布，错误文案明确。
- **策略读取失败**：队列接口返回 500，而不是带着错误策略继续，避免医生按错误策略操作。
- **布尔到枚举的演进**：未来做按期必填时需要数据迁移与契约变更，当前不做。

## Open Questions

- 是否需要在管理端展示「本园本月未测温记录数」：属于覆盖率统计，已列入 Non-Goals，后续单独评估。
- 家长端是否需要显式「未测温」：当前不显示空值，已列入 Non-Goals。
