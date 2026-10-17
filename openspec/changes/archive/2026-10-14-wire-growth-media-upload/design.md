# Design

## Context

动机见 proposal.md。影响设计的关键现状（均为本次实测）：

1. **老师端无 multipart 能力**：`teacher/src/api/http.js` 只有 `uni.request`（JSON）封装，无 `uni.uploadFile`；源码与编译产物中 `common/upload` 只出现在配置值里、无消费代码。
2. **上传基础设施是完整的**：`/common/upload`（`handler/http/upload.go:12`）经 `FileService`（`service/file_service.go`）：单文件上限 20MB、批量上限 20、扩展名白名单（jpg/jpeg/png/gif/webp/mp4/mov/avi 等）、OSS `PutObject`、对象键 `{bizType}/{year}/{month}/{uuid}.{ext}`、落 `files` 表、OSS 参数式缩略图（`?x-oss-process=image/resize,w_200`）。
3. **成长媒体表已有先传后挂设计**：`growth_media.log_id` 可空——素材先独立存在，挂载发生在其后有明确的模型支持。
4. **上传基础设施是现成的，且已有通用通道**：`/common/upload`（`handler/http/upload.go:12`）→ `FileService`（`service/file_service.go`）：单文件上限 20MB、扩展名白名单（jpg/jpeg/png/gif/webp/mp4/mov/avi 等）、OSS `PutObject`、对象键 `uploads/{biz_type}/{year}/{month}/{uuid}.{ext}`、落 `files` 表（含 `biz_type` 列）、OSS 参数式缩略图（`?x-oss-process=image/resize,w_200`）、返回 `{id, file_url, thumb_url, file_name, file_size, file_type}`。`biz_type` 无白名单校验（`file_service.go:56-92`）。
5. **其他模块的媒体能力盘点**（详见 proposal「Why」的全景表）：数据表/字段多数已备（`notice_attachments`、`health_check_media`、`CalendarEvent.Cover`、`feedback.Images`），但「选图 → 上传 → 存引用 → 渲染」这条链**全平台只有 admin 的学生导入接通了**（且走专用端点）；成长记录、通知、日历、晨检都断在不同位置。
6. **既有 `POST /teacher/growth/media/upload` 是形状错的半成品**：handler 用 `ShouldBindJSON`（实现为 JSON `{file_url,...}`），契约写 multipart `files[]`；`CreateGrowthMedia` 无 `log_id` 入参。前端从未调用（封装已作死代码删除）。**没有保留价值**。
7. **家长端已就绪**：`growth_media` 是按 `log_id` 聚合的，`parent_portal_service` 的 feed/详情已消费 `media`——管道接通即自动生效，无需家长端改动。

## Goals / Non-Goals

**Goals:**

- 选图后真实上传，媒体凭证随记录持久化并绑定 `log_id`
- 编辑态回显、追加、移除可保存
- 素材归属可校验（跨班/跨记录引用被拒）
- 不新增冗余接口；消灭既有半成品接口的形状漂移

**Non-Goals:**

- 照片级学生关联与 AI 识别（`growth_media_students` 写入路径）——后续变更
- 孤儿素材清理策略（先保证正常路径不产孤儿；上传即弃的残留治理另案）
- 媒体排序拖拽、视频封面、压缩与转码
- 其他业务的媒体接线

## Decisions

### D1：上传走平台统一的 `/common/upload` 通道（成长记录成为第一个接入方）

**决定**：成长记录的上传复用**既有** `POST /common/upload`（multipart），`biz_type=growth`；前端新增 multipart 能力后调用它，拿到 `file_id` 等凭证。**不改造** `POST /teacher/growth/media/upload`——该接口契约与实现形状不符、零调用方，改造它只会把成长记录变成「专用端点」的特例。

**为什么这条是「每模块重造一遍」的解法**：全平台盘点显示，各模块的数据表都在，断的永远是同一条链。若本变更为成长记录单开 multipart 通道，通知/晨检/日历接入时每个都要各造一遍上传+端点+wire 依赖。统一通道后，**「上传」只存在一份**（`FileService` 已在），每个模块只需做两件事：业务表存引用、UI 渲染。成长记录是**第一个接入样板**。

**对外部共享面的影响**：`/common/upload` 的参数与响应形状**保持不变**（含既有的 `biz_type` 约定）；上传能力只在前端 `teacher/src/api/http.js` 新增封装（三端同构，家长端后续直接复制）。

**替代方案**（未采纳）：改造成长专用 multipart 端点（原 D1）。开发量相近，但把特例固化成平台范式。

**Wire 影响**：**无**（`TeacherPortalService` 不新增依赖）。这是统一通道相较专用端点的直接收益。

**`biz_type` 约定**：`FileService` 无白名单校验，值直接进入对象键与 `files.biz_type`。成长记录统一使用 `growth`。后续模块的 `biz_type` 在各自接入时新增（如 `notice`、`health`、`calendar`），`files.biz_type` 即成为全平台素材的分类索引（审计与清理的钩子）。

### D2：媒体凭证以 `file_id` 为主键随记录提交

**决定**：`TeacherGrowthInput` 增加 `media` 数组，元素两类形态：

```json
"media": [
  { "id": 12, "file_id": 7, "ref_desc": "小明在搭积木" },  // 编辑态：既有媒体（id=growth_media 行）
  { "file_id": 9, "ref_desc": "午餐光盘啦" }               // 新上传：无 id，仅指向刚上传的素材
]
```

- **只传 `file_id`**（+编辑态的 `media.id`，+引用层描述 `ref_desc`），**不传 URL 等展示字段**——`file_url`/`thumb_url`/`file_type`（L1 物理属性）由服务端按 `file_id` 查 `files` 表获取，杜绝客户端伪造 URL
- `file_id` 即 `/common/upload` 响应里的 `id`
- **描述按基线双层归属**：`ref_desc` 存 `growth_media`（L2，这次使用的上下文）；素材固有描述是 `files.caption`（L1），本变更建字段但成长记录 UI 暂不编辑它（素材库启动时接入）

**语义为「全量替换」**：提交的 `media` 即最终附件集合，未出现在其中的既有媒体从记录移除。

**为何全量替换而非增量 patch**：与 `student_ids` 的编辑语义一致（提交即最终名单），前端无需 diff，幂等天然成立。

**校验（`file_id` 维度）**：
- 每个 `file_id` 必须存在于 `files` 表且在提交者可见范围内
- 同一 `file_id` 不得已绑定到**其他**记录（`growth_media` 中该 `file_id` 存在的行、且 `log_id ≠ 本记录` → 拒绝）；同一文件理论上不应同时挂在多条记录上，由服务层校验而非新加数据库约束（存量表不加唯一索引）
- `media` 空数组合法；数组长度上限 20（与批量上传上限同口径）

### D3：媒体行与记录的绑定在创建/更新的事务内完成

**决定**：`CreateGrowthLog` 与 `UpdateGrowthLog` 在既有事务（更新路径已有）中：

```
创建: 校验每个 file_id（存在于 files、未被其他记录绑定）
      INSERT growth_media (log_id=本次记录, file_id, class_id=老师班级, uploader_id=老师) × N
更新: ① 校验提交的 media：
        带 id 的必须 ∈ 本记录当前媒体（否则拒绝）
        每个 file_id 必须存在于 files 且未被其他记录占用
      ② UPDATE 旧媒体 SET log_id=NULL → 不在新集合中的从记录移除
      ③ 新 file_id 建行并绑定 log_id；带 id 的沿用原行（可更新 sort_num）
```

**删除的形态**：从记录移除 = **解除绑定**（`log_id` 置空）而非删除媒体行——素材可能被后续复用，且 OSS 对象不做物理删除（避免误删不可恢复）。

**边界**：`media` 为空数组 = 无附件（合法）；单条记录媒体数上限沿用批量上限 20。

### D4：归属校验采用「file_id 有效性 + 记录域白名单」而非「类目校验」

**决定**：提交的每个 `file_id` 必须（1）存在于 `files` 表；（2）未被其他记录绑定（`growth_media` 中该 `file_id` 的行 `log_id` 为空或等于本记录）。带 `media.id` 的项必须命中「该记录当前关联的媒体集」。

**为何不做「班级内媒体」校验**：新上传素材此时 `log_id = NULL` 且无 class 归属，无法用类目反查；`file_id` 是上传时由服务端签发的、不可伪造的标识，比 URL 域校验更严格，记录域白名单则覆盖编辑场景。

**替代方案**（未采纳）：给 `growth_media` 加 `status` 字段做严格生命周期。代价大（迁移 + 状态机），当前需求不值。

### D5：前端上传走 `uni.uploadFile` 调用统一通道，逐张上传、逐张反馈

**决定**：`http.js` 新增 `uploadFile({ filePath, name, formData, auth })`（基于 `uni.uploadFile`，复用 token 注入与错误归一化约定，multipart 含 `biz_type` 参数）；`teacher-growth.js` 暴露 `uploadGrowthMedia(filePath)` 调用 `/common/upload`（`biz_type=growth`），返回 `file_id` 与展示用凭证。

**前端流程**：

```
选图 (chooseImage)
  → 每张：uploadFile → 成功则 form.media.push({ file_id, file_url, thumb_url, file_type })，失败则该张标记 failed 可重试
  → 发布/保存：form.media 全量随记录提交（file_id + 编辑态 media.id）
  → 编辑回显：loadExistingLog 把 rec.media 写入 form.media（带 id）
  → 移除：从 form.media 中删项（提交时服务端解除绑定）
```

**为何逐张上传而非批量**：单张失败可独立重试；发布时不必等待整批；`/common/upload/batch` 的 20 张批量会放大失败面。

## 跨端触点清单

| 层 | 会改 / 不改 | 说明 |
|---|---|---|
| Go domain | **不改** | `GrowthMedia` 表结构已支持先传后挂；`files` 表已存在；不加新字段 |
| Go repository | **不改** | 媒体与 `files` 的读校验、`growth_media` 的写走 `db` 直写（与既有 `enrichGrowthLogsBatch` 等模式一致） |
| Go service | **会改** | `TeacherPortalService`：`TeacherGrowthInput` 加 `media`、`CreateGrowthLog`/`UpdateGrowthLog` 的 media 事务与 `file_id` 校验辅助；**不新增 OSS 依赖** |
| Go handler | **不改** | 媒体入参随 `TeacherGrowthLogs` 的既有 JSON body 进入；上传端点复用既有 `POST /common/upload` |
| Casbin policy | **不改** | `POST /common/upload` 的 teacher 通配已覆盖 |
| router.go | **不改** | 无新增路由 |
| Wire DI | **不改** | `TeacherPortalService` 不新增依赖构造参数 |
| `backend/api/` 契约 | **会改（契约变更）** | `06-growth.md`：`POST/PUT` 的记录入参补 `media`（`file_id` 维度）语义；`media/upload` 段标记废弃（形状错、零消费，保留路径指向 `/common/upload`） |
| 老师端页面 | **会改** | `create.vue`（选图→上传→提交、编辑态回显与移除）、`api/http.js`（新增通用 multipart 能力）、`api/teacher-growth.js`（上传封装，`biz_type=growth`） |
| 家长端页面 | **不改** | feed/详情已消费 `media`，管道接通即生效 |
| admin 页面 | **不改** | 不涉及 |
| design token | **不改** | 媒体格与上传按钮已有样式 |

**tabBar 核对**：全部改在 `create.vue` 页内与后端实现，不新增页面、不涉及跳转。

**迁移核对**：本变更新增三个列（`files.caption`/`files.tags`/`growth_media.ref_desc`）。仓库无迁移工具、`AutoMigrate` 只增不改，故按既有惯例写手工迁移 SQL（`docs/migrations/`）并用 `go run ./cmd/dbcheck <table>` 核实；无数据回填——存量行新列为 NULL，读取端按空值处理。

**平台级意义（供后续模块接引）**：本变更建立「统一上传通道 + `file_id` 随记录提交」的范式。通知/晨检/日历/反馈接入时，各自只需：定义业务表的引用字段 + 复用前端 `uploadFile` 封装 + 后端按 `file_id` 校验落库。上传端点、OSS 依赖、契约漂移面均不再重复产生。

## Risks / Trade-offs

- **上传成功但未发布 → 孤儿素材**（`log_id NULL` 的残留） → 缓解：素材无 `log_id` 时不进任何 feed；清理策略列入 Non-Goals，后续用定时任务或运维清理（当前量级可忽略）。
- **`/common/upload` 返回 20MB 限制在三端共享**：成长记录沿用该限制（图片经压缩远小于此），无冲突。
- **`media` 全量替换语义可能误删**（前端漏传已有项即视为移除） → 缓解：前端加载详情后以服务端返回为基准初始化 `form.media`，编辑期间只在用户显式移除时才减项；任务中覆盖「仅改文字不动照片」的回归场景。
- **同一文件被多记录引用的边界** → 缓解：服务层校验 `file_id` 未被其他记录绑定（D2/D4），在事务内保证不会出现双重挂载。
- **测试环境 OSS 凭据** → 缓解：本地 `app.ini` 已含 OSS 配置（既有 `ProvideOSSStorage` 在用），若 CI 无凭据，集成验证以「`file_id` 落库与绑定」断言为准，不真实触达 OSS 的对象校验。

## Open Questions

- **「实时照片」是否要做**（老师在活动现场拍完即传、家长实时可见）：涉及「发布前可见性」的产品决策，本变更不实现；管道接通后它是纯前端与策略问题，可独立成案。
- **孤儿素材清理**：由谁定期清（定时任务 / 运维脚本 / 上传时配额），取决于线上量级，暂不决策。
