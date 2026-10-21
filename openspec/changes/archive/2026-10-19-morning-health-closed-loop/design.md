# Design

## Context

See `proposal.md` — Why.

Current constraints that shape the approach：

- 晨检记录的创建仍由「入园登记」触发（`teacher_portal_service.go`），本期**不改队列来源**，只在既有记录上补闭环。
- `morning_health_checks` 已包含体温、五项检查值、三档结论、隔离标记、备注与 `symptom_tags`，不需要新表。
- `HealthCheckMedia` 已存在，但直接存 `file_url/thumb_url`，不符合 `media-assets` 的统一引用契约。
- 站内消息走 `messages` 表（`domain.Message`），已有按学生找家长的辅助函数（`pickup_notify.go`），但发送函数把消息类型写死为 pickup。
- 家长端已有「当前孩子」选择机制（`utils/currentStudent`），幼儿档案页 `pages/student/index.vue` 是现成入口。
- 所有表带 `kindergarten_id`；现有 `AddMedia` 缺园所校验。

## Goals / Non-Goals

**Goals**

- 补上「异常 → 家长」闭环：家长看得到、异常收得到通知。
- 让照片从「选了不上传」变成「上传—关联—可读」的完整链路，并对齐 `media-assets`。
- 把症状标签的权威从客户端搬回服务端，写入有校验。
- 补齐 doctor 健康接口契约与 parent 晨检查询契约。

**Non-Goals**

- 不重设计晨检队列来源（入园登记解耦）。
- 不新增全日观察/午检、缺勤追踪、传染病预警。
- 不新增消息类型枚举、不引入微信订阅消息。
- 不对 `pages/doctor/checkin.vue` 做全页 token 化改造（新增元素按 token 写，旧有硬编码不动）。

## Decisions

### D1：家长视图返回「服务端派生后的可读项」，不返回原始枚举

家长需要的是「哪里异常」，不是 `throat_status=2`。服务端把五项检查值翻译成可读项（含 `3=其他` 这类没有对应症状标签的取值），家长端直接渲染。

- 备选：客户端拿原始值自行翻译。否决——会把 11 个标签的映射复制到第二个客户端，且暴露内部编码。
- 备选：只返回 `symptom_tags`。否决——`其他` 没有标签，家长会漏看到异常。

响应只含家长需要的字段，不含照片、不含原始部位代码、不含 `student_name`（请求已按 `student_id` 限定，当前孩子由前端上下文决定）。

### D2：通知复用 `system` 类型 + 条件更新防并发

- 不新增 `MessageType`：现有枚举（notice/leave/pickup/growth/task/system）没有健康类，新增需要产品裁决；本期复用 `system`，标题固定前缀「晨检提醒」。
- 通知接收方为该生全部 `student_parents` 关联用户，复用 `pickup_notify.go` 的「按学生找家长」查询。
- 幂等：提交改为**条件更新**（`WHERE check_status <> completed`），只有更新成功的请求才发通知；并发双击只有一个赢家。发送前再按 `(user_id, type, reference_id)` 查重，作为兜底。
- 发送失败不回滚晨检结论；逐条收集结果，提交响应返回成功/失败人数。

### D3：照片引用表对齐 `media-assets`

`health_check_media` 增加 `file_id`、`ref_desc`、`sort_num`，满足 `media-assets` 的统一引用契约：

- 上传走 `POST /common/upload`（`biz_type=health`），拿到 `id` 即 `files.id`。
- 关联接口只接收 `file_id` + `category`（+ 可选 `ref_desc`），服务端读 `files` 行取 URL/缩略图/大小。
- 保留 `file_url/thumb_url` 作为读取缓存，避免详情查询回表；**文件内容与元数据的真相源仍是 `files`**。
- 关联前校验：晨检记录属于当前园所；`file_id` 存在且属于当前上传者/园所可达范围。

### D4：症状标签服务端派生

映射沿用既有 11 个标签（来源：`docs/specs/2026-06-10-morning-health-check.md` 与 `admin/pages/health/index.vue` 的展示映射）：

| 条件 | 标签 |
|---|---|
| 体温 ≥ 37.3 | fever |
| 喉咙 1 / 2 | red_throat / swollen_throat |
| 手部 1 / 2 | hand_rash / hand_blister |
| 口腔 1 / 2 | mouth_ulcer / mouth_blister |
| 眼睛 1 / 2 | red_eye / eye_discharge |
| 皮肤 1 / 2 | skin_rash / skin_blister |

- 手/口/皮的 `3=其他` 合法，但**不产生标签**（无权威标签值，不臆造）；家长视图通过 D1 的可读项仍能看到「手部其他异常」。
- 写入校验取值范围：喉 0–2、手 0–3、口 0–3、眼 0–2、皮 0–3；越界拒绝。
- 客户端提交的 `symptom_tags` 字段从写入契约移除，服务端不再读取。

### D5：劝返不再改写接送异常标记

`pickup-custody` 规定接送异常只能由老师显式标记。现有 `SubmitCheck` 在劝返时自动 `pr.IsAbnormal = true`，与该规定冲突。本变更移除该副作用：劝返是健康结论，由健康通知触达家长，不复用接送异常语义。

### D6：家长入口放在幼儿档案页

`parent/src/pages/student/index.vue` 已承载「当前孩子」上下文，新增「最近晨检」区块，按日期倒序展示最近 7 天已完成记录。该页不是 tabBar 页（tab 为 index/growth/pickup/profile），跳转用 `uni.navigateTo`；本期不新增页面。

- 备选：新增独立健康页。否决——P0 多一个页面与入口，收益不抵成本。
- 通知暂不支持点击深链（`messages` 表没有链接字段），消息正文提示「可在「我的 → 幼儿档案」查看」；深链作为后续改进。

### D7：契约补齐方式

- 新增 `backend/api/api-doctor/`，覆盖健康模块 5 个既有端点；doctor 日历端点不在本期（另案补）。
- 在 `backend/api/api-parent/` 新增 `12-health.md`，并在 `README.md` 索引中登记。
- 本次修改 `backend/api/`，属于契约变更（proposal 已声明）。

## 跨端触点清单

| 层 | 是否修改 | 说明 |
|---|---|---|
| Go domain | **会改** | `health_check.go`：`HealthCheckMedia` 增加 `FileID/RefDesc/SortNum`；`morning_health_checks` 不改 |
| repository | **会改** | `health_check_repo.go`：条件更新、按学生+日期范围查询、媒体关联（读 `files`）；消息写入复用 `MessageRepo` |
| service | **会改** | `health_check_service.go`：取值校验、标签派生、条件更新、通知投递与结果统计；`pickup_notify.go`：发送函数参数化消息类型；`parent_portal_service.go`：家长晨检查询与绑定校验 |
| handler | **会改** | `handler/http/health_check.go`：提交响应增加通知统计、媒体入参改 `file_id`；新增 parent 晨检 handler |
| Casbin policy | **不改** | `parent` 已有 `/api/v1/parent/*, *`；`doctor` 已有 `/api/v1/doctor/*, *`，新端点被现有通配覆盖 |
| router.go | **会改** | parent 组新增 `GET /health/records` |
| backend/api/ 契约 | **会改（契约变更）** | 新增 `api-doctor/README.md` + `01-health.md`；新增 `api-parent/12-health.md` 并更新索引 |
| 端页面 teacher | **会改** | `pages/doctor/checkin.vue`：各异常部位拍照、提交后展示通知结果；`api/doctor-health.js`：媒体入参改 `file_id` |
| 端页面 parent | **会改** | `pages/student/index.vue`：新增「最近晨检」区块；新增 `api/parent-health.js` |
| 端页面 admin | **不改** | 统计页消费的 `symptom_tags` 结构不变，标签改为服务端派生后统计口径一致 |
| design token | **会改** | 新增 UI 元素只用既有语义 token，不新增 token；不改造旧有硬编码 |

## 契约变更（字段级溯源）

无设计稿覆盖晨检（`design-data/` 中无对应文件），以下字段以明确业务场景溯源。

### 修改 `PUT /api/v1/doctor/health/check/:id`（BREAKING）

| 字段 | 变化 | 溯源 |
|---|---|---|
| `symptom_tags` | **移除** | 规格「症状标签由服务端派生」；客户端提交值不再生效 |
| `media_ids` | **移除** | 现有实现从未消费该字段（死字段），照片走独立关联端点 |
| `notified_count` / `notify_failed_count`（响应新增） | 新增 | TC-NEW-HTH-0009「通知失败可见」；无法由已有字段派生，消息结果不在晨检记录中 |

其余请求字段（体温、五项、结论、隔离、备注）不变。

### 修改 `POST /api/v1/doctor/health/check/:id/media`（BREAKING）

| 字段 | 变化 | 溯源 |
|---|---|---|
| `file_url` / `thumb_url` / `file_name` / `file_size` | **移除** | 改由服务端按 `file_id` 读取，避免客户端伪造 URL |
| `file_id` | 新增（必填） | `media-assets` 统一引用契约；`POST /common/upload` 返回的 `id` |
| `category` | 保留 | 拍摄部位（throat/hand/mouth/eye/skin），既有业务语义 |
| `ref_desc` | 新增（可选） | `media-assets` 引用层描述字段 |

### 新增 `GET /api/v1/parent/health/records?student_id=<id>`（响应字段）

| 字段 | 溯源 |
|---|---|
| `check_date` | 家长按日期查看最近 7 天记录（规格要求） |
| `overall_status` + `overall_status_label` | 规格「健康/需观察/劝返」结论；label 供直接渲染 |
| `temperature` | 规格「包括体温」；可为空 |
| `abnormal_items[]`（部位标签 + 状态标签） | TC-NEW-HTH-0004「可读文字呈现」；含 `3=其他` |
| `remark` | 规格「保健医备注」 |
| `is_isolated` | TC-NEW-HTH-0006「劝返且隔离」；家长需要据此接走孩子 |

不返回：照片（proposal 明确排除）、原始部位代码（由 `abnormal_items` 覆盖）、`student_name`（请求已限定 `student_id`，避免冗余派生）。

### 既有 3 个 doctor 端点

`GET /doctor/health/queue`、`GET /doctor/health/check/:id`、`GET /doctor/health/today/stats` 响应不变，仅补文档。

## 数据迁移

`health_check_media` 需要显式迁移（AutoMigrate 只增不改，且不会修正既有索引/数据）：

```sql
ALTER TABLE health_check_media
  ADD COLUMN file_id BIGINT UNSIGNED NULL AFTER check_id,
  ADD COLUMN ref_desc VARCHAR(255) NOT NULL DEFAULT '' AFTER category,
  ADD COLUMN sort_num INT NOT NULL DEFAULT 0 AFTER ref_desc,
  ADD INDEX idx_health_media_file_id (file_id);

-- 存量回填：按 URL 命中 files 表
UPDATE health_check_media m
JOIN files f ON f.file_url = m.file_url AND f.deleted_at IS NULL
SET m.file_id = f.id
WHERE m.file_id IS NULL AND m.deleted_at IS NULL;

-- 核实
-- go run ./cmd/dbcheck health_check_media
```

- 回填不到 `file_id` 的历史行保留 URL 缓存，详情仍可展示；不删除任何行。
- 回滚：`ALTER TABLE health_check_media DROP INDEX idx_health_media_file_id, DROP COLUMN file_id, DROP COLUMN ref_desc, DROP COLUMN sort_num;`。旧代码不读取新列，回滚安全。
- `orders`/索引变更后必须用 `dbcheck` 核实，不假设 AutoMigrate 生效。

## 流程

提交与通知：

```
保健医点「需观察/劝返」
        │
        ▼
PUT /doctor/health/check/:id
        │
        ├─ 校验取值范围（越界 → 422，不落库）
        ├─ 服务端派生 symptom_tags
        ├─ UPDATE ... WHERE check_status <> 1   ← 并发只有一个赢家
        │      └─ 影响行数 0 → 已完成后提交，拒绝
        ├─ 查 student_parents 得家长列表
        ├─ 逐条写 messages（type=system，reference_id=check_id，先查重）
        └─ 返回 { detail, notified_count, notify_failed_count }
```

家长查看：

```
家长进入「幼儿档案」（当前孩子）
        │
        ▼
GET /parent/health/records?student_id=
        │
        ├─ 校验 student_parents(user_id, student_id) 绑定
        ├─ 查最近 7 天、check_status=1 的记录
        └─ 服务端把五项值翻译为 abnormal_items + 结论 label
```

## Risks / Trade-offs

- **通知部分失败无自动重试** → 提交响应暴露 `notify_failed_count`，操作者可见；补齐动作留待后续（可加「重发通知」）。
- **消息类型复用 `system`** → 消息中心会把晨检提醒归入系统类；P0 可接受，产品若要求单列需新增枚举。
- **家长入口依赖家长自行返回幼儿档案**（无深链）→ 消息正文明确路径；深链改造超出本期。
- **照片存量回填可能无匹配** → 保留 URL 缓存，不影响读取；`file_id` 为空的行在后续清理时处理。
- **条件更新与详情读取非同一事务** → 更新成功后再读详情；期间被其他请求改动（本期无其他写路径）风险可忽略。

## Open Questions

- 家长是否应看到异常照片：涉及未成年人医疗影像可见性，待产品裁决。
- 消息中心是否需要健康分类与点击深链：影响 `MessageType` 与 `messages` 表结构，另案处理。
- 医生页整体 token 化：与本次功能改动解耦，后续 UI 专项处理。
