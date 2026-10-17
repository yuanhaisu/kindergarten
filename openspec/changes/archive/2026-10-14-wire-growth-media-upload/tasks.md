# Tasks

## 0. 素材管理设计基线（先行落地）

- [x] 0.1 `docs/media-management-design.md` 与 `openspec/specs/media-assets/spec.md` 已写入（三层模型/标签体系/版本替换/删除治理/可见性/接入范式）——本变更按基线实施，无需再改基线文件本身；验证：`openspec validate --all` 通过

## 1. 后端：数据地基与媒体引用持久化（Go service）

- [x] 1.1 **L1 地基**：`files` 表新增 `caption`（varchar 255）与 `tags`（varchar 255，逗号分隔）两列，新增 `domain.File` 字段 + 手工迁移 SQL（仓库无迁移工具，`AutoMigrate` 只增不改）；验证：`go run ./cmd/dbcheck files` 显示两列已存在
- [x] 1.2 **L2 地基**：`growth_media` 新增 `ref_desc`（varchar 255）列 + `domain.GrowthMedia` 字段 + 手工迁移 SQL；验证：`go run ./cmd/dbcheck growth_media` 显示新列
- [x] 1.3 `TeacherGrowthInput` 增加 `media` 数组（元素 `{ id?, file_id, ref_desc? }`）；`MediaInput` 结构体定义；验证：`go build ./...` 通过
- [x] 1.4 `file_id` 校验辅助：存在性（`files` 表）、未被其他记录绑定（`growth_media.file_id` 行 `log_id` 为空或等于本记录）、编辑态 `media.id` ∈ 本记录当前媒体；`media` 空数组合法、上限 20；验证：单测覆盖各拒绝分支
- [x] 1.5 `CreateGrowthLog` 事务路径：按 `file_id` 查 `files` 取展示字段，写 `growth_media`（`log_id=本次记录, class_id, uploader_id, ref_desc`）并绑定；验证：`go build` + 单测通过
- [x] 1.6 `UpdateGrowthLog` 事务内「全量替换」：带 `id` 沿用原行（可更新 `sort_num`/`ref_desc`）；不在新集合中的既有媒体解除绑定（`log_id=NULL`，**不删文件行**）；新 `file_id` 建行绑定；验证：`go test ./internal/service/ -run TestGrowth` 通过

## 2. 契约文档（backend/api/）

- [x] 2.1 更新 `backend/api/api-teacher/06-growth.md`：`POST /teacher/growth/logs` 与 `PUT /teacher/growth/logs/:id` 补 `media` 入参语义（`file_id` 维度、全量替换、编辑态保留 `id`、上限 20）；验证：文档与实现字段逐条核对
- [x] 2.2 `media/upload` 段标记废弃（形状错、零调用），指引改走 `POST /common/upload`（`biz_type=growth`）；验证：文档中不再宣称 `auto_tag`/`tagged_students`（AI 属另案）

## 3. 前端：上传能力（teacher/）

- [x] 3.1 [P] `teacher/src/api/http.js` 新增 `uploadFile({ filePath, name, formData, auth })`：基于 `uni.uploadFile`，复用 token 注入与错误归一化约定（成功判定与 `request()` 一致：2xx + `code === 0`）；验证：`npm run build:mp-weixin` 通过
- [x] 3.2 [P] `teacher/src/api/teacher-growth.js` 新增 `uploadGrowthMedia(filePath)`：调 `POST /common/upload`（`biz_type=growth`），返回 `{ id, file_url, thumb_url, file_type, file_name, file_size }`；验证：被 create.vue 引用（编译通过）

## 4. 前端：发布与编辑链路（teacher/）

- [x] 4.1 `create.vue` 选图改造：`chooseImage` 成功后逐张上传，`form.media` 存 `{ file_id, file_url, thumb_url, file_type }`（不再存临时路径）；上传中状态、失败张标记与重试；验证：`npm run build:mp-weixin` 通过
- [x] 4.2 `create.vue` 编辑态：`loadExistingLog` 把详情 `media` 写入 `form.media`（含 `media.id` 与 `file_id`）；照片区区分「已有」与「新上传」；移除按钮从 `form.media` 删项；验证：构建通过 + 集成验证组用真实数据回显
- [x] 4.3 `create.vue` 提交：`publish()` 随记录提交 `media`（`{ id?, file_id }` 数组）；存在上传未完成的张时阻断提交并提示；页面「自动识别孩子」文案按实现调整（不承诺未实现能力）；验证：构建通过
- [x] 4.4 组内验证：`npm run build:mp-weixin`（teacher）通过；对照 specs/growth-media-pipeline 的交互场景自查

## 5. 测试补录（docs/tests/）

- [x] 5.1 在 `docs/tests/TEST_CATALOG.md` 补录 TC-NEW-GMP-0001~0011（见 specs/growth-media-pipeline）
- [x] 5.2 [P] 媒体校验纯函数单测（`file_id` MISS/被占、`media.id` 越界、空数组合法、上限 20）；验证：`go test ./internal/service/ -run TestGrowthMedia` 通过

## 6. 集成验收（跨组）

- [x] 6.1 后端整体验证：`make build` 与 `go test -count=1 ./...` 通过；`make lint` 无新增问题
- [x] 6.2 API 级验证（真实数据 + JWT，multipart 直传验证通道）：
      `POST /common/upload` 上传 2 张图（`biz_type=growth`）→ 返回 `file_id` 且 `files` 表落库；
      带 `file_id` 发布记录 → `growth_media.log_id` 绑定、详情 `media` 可见（TC-NEW-GMP-0005）；
      编辑保留 1 张 + 新增 1 张 → 记录照片最终为 2 张、被移除者 `log_id` 置空（TC-NEW-GMP-0008/0009）；
      跨记录引用 `file_id` → 拒绝且无写入（TC-NEW-GMP-0010）；超限/非法格式上传被拒（TC-NEW-GMP-0003/0004）
- [x] 6.3 老师端 UI 验证（CDP 真实点击）：**部分完成**——已验：新构建渲染（新文案入、旧「自动识别孩子」承诺消失）、点击「添加」真实触发 `wx.chooseImage`、上传状态机全链路（上传中→失败·重试→点击重试→再上传中）、失败项可移除；期间补了真实缺口**上传超时护栏**（先前的挂起会永远停在「上传中」，现 30s 判失败可重试）。**未验**：真实选图→上传成功的完整 UI 回路——模拟器无法合成文件选择器，且对本机 localhost 的 upload 请求不可达（环境限制，与代码无关）；该路径由 API 级真实 OSS 上传验证覆盖（见 verification-notes.md）。
- [x] 6.4 逐条对照 specs/growth-media-pipeline 验收，核对 TC 引用与实际验证的对应关系
- [x] 6.5 收工清理：删除验证产生的临时记录与媒体行（解除绑定 + 处理孤儿 `files` 行）；按 PID 杀掉残留 uni watcher
