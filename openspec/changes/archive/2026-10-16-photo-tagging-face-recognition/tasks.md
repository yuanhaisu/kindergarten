# Tasks

## 1. 接入验证：腾讯云实测（决定后续阈值与可行性，优先做）

- [x] 1.1 腾讯云控制台准备核对：开通人脸识别（IAI）、AK 权限（子账号最小权限）、确认免费额度与 QPS 档位；验证：用 CLI/Postman 成功调用一次 `CreateGroup`（记录到 `[face]` 配置说明文档）
- [x] 1.2 `internal/ai/face/tencent` 最小可用实现（`EnsurePerson` + `Search` + `RemovePerson`），并写一次性验证命令 `cmd/facecheck`（读本地照片目录 → 建库 → 注册 → 搜索 → 打印每张照片的命中与分数）；验证：`go run ./cmd/facecheck <dir>` 输出真实分数
- [x] 1.3 **儿童脸质量实测**：用 20-30 张真实照片（含合影/不同光照）跑 facecheck，统计召回与误报，产出实测报告（`docs/face-quality-report.md`：分数分布 + 建议的 `suggest_high`/`suggest_low` 取值）；验证：报告落盘且阈值有数据依据

## 2. 数据地基（Go domain + 迁移）

- [x] 2.1 新表 `student_faces`（student_id/face_url/provider/person_ref/status/deleted_at；联合唯一 `(student_id, face_url)`）；新表 `recognition_jobs`（media_id 唯一/kindergarten_id/status/retries/last_error/next_run_at/worker_id）；验证：结构性迁移 SQL + `AutoMigrate` + `go run ./cmd/dbcheck student_faces` / `recognition_jobs` 核实
- [x] 2.2 `growth_media_students` 加 `status`（0建议/1确认/2拒绝，默认 NULL 兼容存量）与 `confidence` 列；验证：`go run ./cmd/dbcheck growth_media_students` 显示新列

## 3. FaceProvider 抽象与配置（Go 新包）

- [x] 3.1 `internal/ai/face`：`Provider` 接口 + `Match` 结构 + provider 工厂（按 `[face].provider` 选择）；验证：`go build ./...` 通过
- [x] 3.2 `internal/ai/face/mock`：按可配置规则返回假匹配（供无凭证环境跑通全链路）；验证：单测断言 mock 的确定性输出
- [x] 3.3 `config` 增 `[face]` 段（provider/enabled/qps/daily_budget/suggest_high/suggest_low/secret_id/secret_key/region）；`configs/app.ini.example` 同步；验证：配置加载单测通过

## 4. 识别队列（Go service）

- [x] 4.1 worker 池：原子 CAS 认领（`UPDATE ... WHERE status='pending'`）→ base64 送 `Provider.Search` → 写建议行（`status=0`，`ON DUPLICATE` 不覆盖已确认）；验证：`go build` + 单测（认领互斥、建议写入不覆盖）
- [x] 4.2 限流器（`golang.org/x/time/rate` 全局 bucket，`[face].qps` 驱动）；验证：单测断言调用节奏不超配置
- [x] 4.3 租户公平认领（按 kindergarten_id 轮转取任务）；验证：单测（两园积压时交替消费）
- [x] 4.4 预算护栏（Redis 日计数：全局+按园；触顶暂停认领并记告警日志，次日恢复）；验证：单测/集成验证（触顶后不再发起调用且任务保留）
- [x] 4.5 重试与降级（网络类错误退避重试 ≤3；业务类结果不算失败；`[face].enabled` 白名单未开启不建任务）；验证：单测覆盖各分支
- [x] 4.6 发布路径接线：记录创建/更新事务提交后，为**含照片且该园已开启识别**的记录建任务（每照片一条）；验证：集成验证（发布后任务存在；未开启园所无任务）

## 5. 点名与建议确认（Go service + handler + 契约）

- [x] 5.1 点名写入：`media[]` 扩展 `student_ids`；创建/更新事务内重建关联（白名单=本记录涉及学生，越界拒绝）；验证：`go test ./internal/service/` 通过
- [x] 5.2 媒体出参扩展：老师端详情、家长端卡片/详情三个出参函数补 `student_ids`（**仅 `status=1`**）；验证：单测（建议态不出现在家长出参）
- [x] 5.3 建议确认端点：`PUT /teacher/growth/media/:id/students/:sid`（body `{action: confirm|reject}`，仅允许 0→1/0→2）；识别状态查询端点（老师端轮询用）；验证：`go build` + 单测状态迁移
- [x] 5.4 底库端点（admin）：学生人脸照片上传/列表/重试注册/删除（调 `EnsurePerson`/`RemovePerson`）；验证：`go build` + 路由注册核对
- [x] 5.5 契约更新：`api-teacher/06-growth.md`（media 入参 `student_ids`、建议确认、识别状态）、`api-parent/04-growth.md`（media 项 `student_ids`）、admin 学生接口（底库）；验证：文档与实现逐字段核对

## 6. 三端 UI

- [x] 6.1 [P] admin 学生管理：人脸照片上传（1-5 张）+ 底库状态列 + 重试按钮；验证：`npm run build`（admin）通过
- [x] 6.2 [P] 老师端 create.vue：照片点名面板（多选照片 → 批量勾选；单张微调）；验证：`npm run build:mp-weixin` 通过
- [x] 6.3 老师端 detail.vue：照片标签（点名呈现）+ 建议态（高置信预勾标记、低置信提示、确认/拒绝按钮）+ 轮询刷新建议；验证：构建通过
- [x] 6.4 家长端 detail.vue：「只看我的孩子（默认）/ 看全部」切换 + 照片标识 + 无点名空态引导；个体记录不筛选；验证：构建通过
- [x] 6.5 组内验证：三端构建全通过；对照 specs/photo-tagging 与 specs/growth-recognition-queue 交互场景自查

## 7. 测试补录与集成验收

- [x] 7.1 `docs/tests/TEST_CATALOG.md` 补录 TC-NEW-PTG-0001~0014、TC-NEW-FRG-0001~0006、TC-NEW-RCQ-0001~0008
- [x] 7.2 [P] 后端单测补强（点名白名单、状态机迁移、认领互斥、限流节奏、公平轮转）；验证：`go test ./internal/service/ -run "TestPhotoTagging|TestRecognition|TestFace"` 通过
- [x] 7.3 后端整体验证：`make build` + `go test -count=1 ./...` + `make lint` 无新增；`make wire` 后启动正常
- [x] 7.4 API 级集成验证（mock provider）：发布带照片记录 → 任务生成 → 建议写入 → 确认/拒绝 → 家长端出参只含确认项；白名单越界拒绝；未开启园所不建任务（TC-NEW-RCQ-0001/0008、PTG 系列）
- [x] 7.5 UI 走查（CDP 真实点击）：老师端点名面板批量/单张、建议确认；家长端筛选切换与空态（TC-NEW-PTG-0010~0013）
- [x] 7.6 逐条对照三个 specs 验收；产出归档说明（含质量实测结论、阈值取值依据、腾讯云配置清单）→ `docs/face-tagging-delivery-notes.md`
- [x] 7.7 收工清理：临时数据/照片清理（21 条验证记录 + 关联媒体/点名/任务 + mock 照片服务）；uni watcher 按 PID 清理

## 8. 省力路径（D10：对抗性审查后纳入）

- [x] 8.1 后端：`POST /teacher/growth/logs/:id/suggestions/confirm-high`（批量 0→1，仅 confidence ≥ suggest_high；归属校验）；验证：单测（高置信转正/低置信不动/跨班拒绝）
- [x] 8.2 后端：记录列表出参 `pending_suggestions_count`（status=0 计数，按 log 聚合）；验证：单测/API 断言
- [x] 8.3 老师端详情页：建议区加「全部确认（N 条高置信）」按钮（无高置信时不出现）；验证：UI 走查
- [x] 8.4 老师端列表：卡片呈现「AI建议 N 待确认」角标（0 不呈现）；批量面板加「全选照片」；验证：UI 走查
- [x] 8.5 契约与用例：`api-teacher/06-growth.md` 补端点与字段；TEST_CATALOG 补 TC-NEW-PTG-0015~0017
## 9. 大合影与全班一键（D11：二次审查后纳入）

- [x] 9.1 Provider 扩展：`Search` 返回 `FaceNum`（tencent 解析响应字段 + `FaceMatchThreshold=suggest_low`；mock 支持 `MOCK_FACES:n` 指令）；`[face].group_photo_min_faces` 配置；验证：单测（mock 分型指令）+ `go build`
- [x] 9.2 队列：人脸数 ≥ 阈值 → **不写逐人建议**，`face_count` 落任务行；写建议前按记录范围过滤（全班→本班在读；个体→关联学生）；验证：单测（大合影无建议 / 跨班命中被滤）
- [x] 9.3 后端：`GET /teacher/class/present-students?date=`（到场 = 在读 ∧ 无明确不在；无数据保守全员）+ `POST /teacher/growth/media/:id/class-roster`（一键全班，落确认点名）；迁移 SQL 补 `recognition_jobs.face_count`；验证：单测（请假排除/无数据保守/跨班拒绝）
- [x] 9.4 老师端 UI：详情页大合影 CTA（「大合影 · 一键全班（当日到场 N 人）」）+ 编辑页面板「全班」按钮（预勾可增减）+ 发布页引导文案；验证：真机走查
- [x] 9.5 契约与用例：`api-teacher/06-growth.md`（tagging-status 出参 `face_count`、两个新端点）；TEST_CATALOG 补 TC-NEW-PTG-0018~0020、TC-NEW-RCQ-0009；`docs/teacher-naming-guidelines.md`（老师拍照与点名规范）
- [x] 9.6 端到端验证：mock `MOCK_FACES:8` 大合影 → 无建议 + face_count=8 → 一键全班 → 家长端只含确认项、请假者不可见；单人照路径回归

- [x] 9.7 **对抗性审查修复（2026-10-16 复审）**：① 个体记录的大合影 CTA 点击 400（违反「仅对全班记录开放」）→ 客户端按 `target_type` 抑制入口 + spec/契约补条款；② 客户端本地并集 ≠ 服务端真相（已拒绝不复活 / 点名与到场名单 TOCTOU）→ `class-roster` 回读最终名单 `named` 下发，客户端直接渲染；③ 名单未加载时 CTA 不显示人数、批量面板区分「新增 N 人 / 到场共 M 人」。复验：单测 + 真机真实点击 + `go test ./...` + lint 全绿。
- [x] 9.8 **复审裁定（2026-10-16，用户拍板）**：① 家长端默认栏保持现状（集体照片漏标时经「看全部」可见，不改为无条件进默认栏）——内容模型「集体/孩子专属/其他孩子的」记入交付说明；② 大合影分型**仅对全班记录生效**，个体记录的大合影保留建议（`handle` 按 `targetType` 判定 + 新单测 TC-NEW-PTG-0021；spec/契约/SOP 同步）。
- [x] 9.9 **复审 D 纳入（2026-10-16）**：建议轮询遇瞬时失败不再永久停摆——改 setTimeout 调度 + 退避重试（4s→8s→…→30s 封顶）+ in-flight 去重，识别完成后无需重进页面即可看到建议（TC-NEW-RCQ-0011）。验证：真机——轮询期间停后端复启，建议自动出现（未重进页面）。

- [x] 8.6 **手动测试缺陷修复（2026-10-15 用户实测）**：批量面板不显示"照片上已点的孩子"→「已点 2 人 + 新增 1 人 = 点名 3」不可预期。修复：① 面板显性标注「已有」并不可取消；② 底部按钮预告总人数（确定（共 N 人））；③ tip 说明追加+总数。**连带修复**：批量只选 1 张时曾掉入「替换」分支（会清掉已有名单）——改为显式 `namingIsBatch` 标记。验证：Node 逐行复刻逻辑断言 6 个场景 + **真机复验**：面板打开即显示「已有×2 / 确定（共 2 人）」→ 选吴桐桐 →「确定（共 3 人）」预告 → 确认后徽标「点名 3」与预告一致
