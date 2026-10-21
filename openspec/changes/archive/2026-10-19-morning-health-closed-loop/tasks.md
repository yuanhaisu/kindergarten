# Tasks

## 1. 后端 · 领域与数据迁移

- [x] 1.1 在 `server/internal/domain/health_check.go` 为 `HealthCheckMedia` 增加 `FileID`、`RefDesc`、`SortNum` 字段；验证：`go build ./...` 通过
- [x] 1.2 新增 `docs/migrations/<日期>-health-check-media-file-ref.sql`（加列 + 索引 + 按 URL 回填 `file_id`）；验证：对本地库执行后 `go run ./cmd/dbcheck health_check_media` 显示新列与索引
- [x] 1.3 验证：迁移 SQL 可在不删数据的前提下重复执行（或提供幂等保护），回滚语句已在 design 中登记

## 2. 后端 · 提交、标签派生与通知闭环

- [x] 2.1 在 `health_check_service.go` 为五项检查值增加范围校验（喉 0–2、手/口/皮 0–3、眼 0–2），越界返回明确错误；验证：新增单测覆盖每个部位的边界值
- [x] 2.2 实现服务端症状标签派生（11 个既有标签，`3=其他` 不产标签），忽略请求中的 `symptom_tags`；验证：单测覆盖多部位并集、体温阈值、`3=其他`
- [x] 2.3 提交改为条件更新（`WHERE check_status <> completed`），并发双击只有一个请求成功；验证：单测模拟两次提交，断言第二次拒绝且通知只发一次
- [x] 2.4 将 `pickup_notify.go` 的发送函数参数化消息类型，保持接送通知行为不变；验证：既有 pickup 通知测试全部通过
- [x] 2.5 提交成功后向该生全部已绑定家长写站内消息（`type=system`、标题含「晨检提醒」、`reference_id=check_id`），发送前按 `(user_id,type,reference_id)` 查重；验证：单测断言需观察/劝返各生成一条、健康不生成
- [x] 2.6 提交响应增加 `notified_count` / `notify_failed_count`，发送失败不回滚结论；验证：单测令某位家长写入失败，断言结论仍为已完成且失败计数为 1
- [x] 2.7 移除劝返时自动标记接送记录异常的副作用；验证：单测断言劝返后 `pickup_records.is_abnormal` 保持原值
- [x] 2.8 运行 `make wire` 更新 DI（`HealthCheckService` 新增 `MessageRepo` 依赖）；验证：`go build ./...` 通过，无 `no provider found`
- [x] 2.9 验证：`go test ./internal/service -run 'HealthCheck|GrowthLike' -timeout 600s` 通过；逐条对照 specs 的 0005–0009、0013–0021

## 3. 后端 · 照片关联与园所隔离

- [x] 3.1 改造照片关联：入参改为 `file_id` + `category` + 可选 `ref_desc`，服务端读 `files` 行填充 URL/缩略图/大小；验证：单测断言关联后详情能读回照片且 `file_id` 正确
- [x] 3.2 关联前校验晨检记录属于当前园所，并校验 `file_id` 存在；验证：单测跨园所关联被拒，不产生引用行
- [x] 3.3 复核所有 doctor 端读写（队列、详情、提交、媒体、统计）都带 `kindergarten_id` 条件；验证：单测覆盖跨园所读取与提交被拒
- [x] 3.4 验证：`go build ./...` 与 `go test ./internal/service -run HealthCheck -timeout 600s` 通过；对照 specs 的 0010–0012、0017–0019

## 4. 后端 · 家长晨检查询

- [x] 4.1 在 `parent_portal_service.go` 实现按 `student_id` 查询最近 7 天已完成晨检，先校验 `student_parents(user_id, student_id)` 绑定；验证：单测断言未绑定家长被拒、只返回已完成记录
- [x] 4.2 服务端把五项检查值翻译为 `abnormal_items`（部位 + 状态可读文案，含 `3=其他`），并返回结论 label；验证：单测覆盖每个部位的 0/1/2/3
- [x] 4.3 新增 parent handler 与 `router.go` 路由 `GET /api/v1/parent/health/records`；确认 Casbin 现有 `/api/v1/parent/*` 通配已覆盖；验证：`go build ./...` 通过；200/401 的实请求验证并入 8.2 集成验证
- [x] 4.4 验证：`go test ./internal/service -run ParentHealth -timeout 600s` 通过；对照 specs 的 0001–0004

## 5. 契约

- [x] 5.1 新增 `backend/api/api-doctor/README.md` 与 `01-health.md`，覆盖队列/详情/提交/照片关联/当日统计，并与实现字段一致；验证：逐字段比对 handler 与 service DTO，无多余或缺失字段
- [x] 5.2 新增 `backend/api/api-parent/12-health.md` 并在 `api-parent/README.md` 索引登记；验证：字段清单与 design 的溯源表一致
- [x] 5.3 验证：契约文档中不再出现已移除的 `symptom_tags`（提交请求）与 `file_url/thumb_url`（媒体关联请求）；响应新增字段均有溯源

## 6. 老师端 · 医生模块

- [x] 6.1 更新 `teacher/src/api/doctor-health.js`：媒体关联入参改为 `file_id/category/ref_desc`，提交返回类型增加通知计数；验证：调用方编译无类型/字段错误
- [x] 6.2 在 `checkin.vue` 为每个异常部位提供拍照入口：先 `uni.uploadFile` 到 `/common/upload`（`biz_type=health`），拿 `id` 后调用关联接口；验证：真实点击拍照 → 上传 → 详情能读回照片
- [x] 6.3 提交后根据 `notified_count/notify_failed_count` 给出结果提示（全部送达 / 部分未送达），不静默；验证：构造失败场景提示出现
- [x] 6.4 新增 UI 元素只使用既有语义 token；验证：`node scripts/check-tokens.mjs teacher` 中 `checkin.vue` 的违规均为改动前既有项，新增样式无硬编码
- [x] 6.5 验证：`npm run build:mp-weixin` 通过；用真实点击走完「打开检查卡 → 标异常 → 拍照 → 提交」，不依赖 MCP 导航（实测：原生文件选择器用 `_verify/select-file-dialog.ps1`（Win32 枚举 + SendKeys）自动完成；真实点击「拍」→ 选图 → 上传 → 关联 → 卡片预览 → 提交全链路通过，详情接口可读回照片）

## 7. 家长端

- [x] 7.1 新增 `parent/src/api/parent-health.js` 封装家长晨检查询；验证：调用路径与契约一致
- [x] 7.2 在 `parent/src/pages/student/index.vue` 增加「最近晨检」区块：按日期倒序展示 7 天内记录、结论、异常项与备注；验证：多孩切换后展示对应孩子的记录
- [x] 7.3 页面跳转核对：本期在幼儿档案页内嵌区块，未新增跳转；该页非 tabBar 页，既有入口无 `switchTab` 误用
- [x] 7.4 验证：`npm run build:mp-weixin` 通过；真实点击进入幼儿档案 → 看到晨检区块 → 切换孩子后数据变化（实测：小俊彦显示「需观察/喉咙红肿/37.8°C」，切换到王小花显示「健康/36.5°C」；父提交后家长端即时出现「劝返就医/已隔离/38.1°C」）

## 8. 测试补录与集成验证

- [x] 8.1 在 `docs/tests/TEST_CATALOG.md` 新增「晨检 (HTH)」模块，登记 TC-NEW-HTH-0001 至 0022；验证：22 条编号与 specs 引用一一对应
- [x] 8.2 集成验证：老师登记入园 → 保健医提交「需观察」→ 家长收到站内消息且在幼儿档案看到记录 → 提交通知失败场景能显示未送达；验证：在本地库查 `messages` 与家长接口返回一致（实测：提交「劝返+隔离」→ 3 位家长各收一条「【紧急】晨检提醒」，家长端幼儿档案立即出现该记录）
- [x] 8.3 跨租户验证：用另一园所账号访问本园晨检详情/提交/照片关联，全部被拒；验证：响应与 DB 均无变化（实测 404，媒体行数不变）
- [x] 8.4 回归验证：管理端「晨检统计」页症状分布仍正确（标签改为服务端派生后口径一致）；验证：对比提交前后的统计数值（实测 `red_throat=2, swollen_throat=1, fever=2`，与库内一致）
- [x] 8.5 收工验证：`make check` 通过；杀掉残留的 `dev:mp-weixin` watcher；逐条对照 specs 全部 22 个场景，未通过项不得归档（实测：`go vet`、`golangci-lint` 0 issues、`make check` 全量 `-race` 通过（service 281s）；已安装 WinLibs 64 位 gcc 并设置用户级 `CC`，race detector 恢复可用；清理已完成）
