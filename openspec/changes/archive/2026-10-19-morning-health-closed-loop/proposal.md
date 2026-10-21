# Proposal

## Why

晨检目前只完成了「记录」：保健医提交「健康 / 需观察 / 劝返」后流程即结束。家长对孩子的晨检结果完全不可见，劝返与需观察没有任何通知；异常照片的相机入口不会上传任何内容；症状标签由客户端拼字符串、服务端不校验。结果是晨检产生的信息只在园内流转——最需要它的家长拿不到，最需要留证的异常场景没有证据。

本变更补齐「异常 → 家长」这条闭环，并修掉让闭环无法成立的三个前置缺陷（照片链路、症状标签可靠性、跨园所边界），同时为 `/doctor` 补上缺失的接口契约。

## What Changes

- **家长可见晨检结果**（parent）：家长可查看孩子在指定日期的晨检结论、异常项与保健医备注。范围限本人孩子、限已完成记录。
- **异常结论通知家长**（parent + 后端）：结论为「需观察」或「劝返」时，向该生全部已绑定家长投递站内消息。同一记录对同一家长至多一条；投递失败必须在提交结果中体现，不得静默成功。劝返且已隔离时消息标题标注紧急。
- **异常照片真正落库**（teacher 医生模块 + 后端）：任一异常部位可拍照，经统一上传通道 `POST /common/upload` 上传后关联到晨检记录；关联接口必须校验记录属于当前园所（当前实现缺失该校验）。**BREAKING**：照片引用表对齐 `media-assets` 的统一引用契约，需要手工迁移存量数据。
- **症状标签改为服务端派生**（后端）：写入契约不再接收客户端拼装的 `symptom_tags`，改由服务端根据体温与五项检查值按既有 11 个标签映射派生，并校验各项取值在允许范围内。**BREAKING**（PUT 契约字段变更；唯一客户端同步修改）。
- **补齐 `/doctor` 接口契约**（契约）：**本次将修改 `backend/api/`，属于契约变更**。新增 doctor 端接口规格（队列 / 详情 / 提交 / 照片关联 / 当日统计），并在 parent 端补晨检查询端点。管理端现有统计页行为不变。

## Out of Scope

- 全日观察 / 午检：需要新的观察记录模型与工作流，单独 change
- 缺勤原因与因病缺勤追踪：同上
- 传染病聚集性预警、趋势与热力图（旧设计 Phase 2/3）
- 隔离闭环（隔离室登记、解除时间、疾控上报）
- 队列与入园登记的耦合（园车 / 迟到 / 漏登记导致漏检）：属于覆盖问题，涉及队列来源重设计，单独 change
- 医生侧的晨检补录、历史查询、日期选择器
- 家长侧仅开放最近 7 天，更长的健康档案不在本期
- 家长查看晨检照片：涉及未成年人医疗影像可见性，待产品裁决
- 微信订阅消息：本期只做站内消息
- 医生页面的 UI token 化与视觉改版

## Capabilities

### New Capabilities

- `morning-health`: 晨检记录从待检到结论、异常留证、结论对家长可见与通知，以及跨园所的数据边界。

### Modified Capabilities

（无。照片归属遵守既有 `media-assets` 约束，不修改其需求；接送登记语义遵守既有 `pickup-custody`，不修改其需求。）

## Open Decisions / Conflicts

1. **照片引用表与 `media-assets` 契约冲突**：现有 `health_check_media` 直接存 `file_url/thumb_url`，没有 `file_id/ref_desc/sort_num`，不符合 `media-assets` 的统一引用契约。本变更倾向按契约改造并迁移存量；若不同意，必须在 design 中显式记录偏离理由。
2. **家长入口位置**：旧设计与 `design-data/` 都没有规定家长在哪个页面查看晨检结果（幼儿档案 / 首页 / 新增页面）。行为规格只约束「可见与可见范围」，具体入口由 design 决定。
3. **通知类型**：不新增 `MessageType` 枚举，复用既有 `system`；若产品希望消息中心单列健康分类，需要另行裁决。
4. **与 `pickup-custody` 的冲突**：现有实现会在「劝返」时自动把对应接送记录标为异常；而 `pickup-custody` 规定「接送记录的异常标记 MUST 由老师显式开启，系统 MUST NOT 自动置为异常」。本变更倾向移除该自动标记——劝返是健康结论，不是接送异常，家长已由健康通知获知。若保留，则需要修改 `pickup-custody` 的口径，属于另一个 capability 的变更。

## Impact

- **后端**：`domain/health_check.go`（照片引用契约）、`repository/health_check_repo.go`（园所范围校验、派生标签落库）、`service/health_check_service.go`（标签派生、通知、照片关联）、`service/parent_portal_service.go`（家长查询）、`handler/http/health_check.go`、parent 端 handler、`router.go`、`configs/rbac_policy.csv`
- **契约**：新增 `backend/api/api-doctor/`；修改 `backend/api/api-parent/`；响应包络沿用 `pkg/response`
- **数据**：`morning_health_checks` 与 `health_check_media` 已存在；照片引用表改动需要手工 SQL，并用 `go run ./cmd/dbcheck health_check_media` 核实（AutoMigrate 只增不改）
- **客户端**：`teacher/src/pages/doctor/checkin.vue`、`teacher/src/api/doctor-health.js`、`parent/src/pages/*`（晨检结果入口）、`parent/src/api/*`
- **外部依赖**：无新增；不接微信订阅消息
