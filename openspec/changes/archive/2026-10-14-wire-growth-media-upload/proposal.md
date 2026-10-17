# Proposal

## Why

成长记录的照片链路**从未接通**：`files` 表 0 行（历史上没有任何一次上传落库）。现状是四个断点叠加——老师端选图只存本地临时路径、发布请求不带媒体字段（后端结构体无对应字段，字段被丢弃）、后端媒体接口没有 `log_id` 入参（无法挂到记录）、三端前端无人调用上传接口。

后果直接打击产品定位：家长端照片区永远空白，而老师端页面仍写着「最多可上传20张照片/5个视频，系统将自动识别孩子并推送」——**承诺与实现的落差比功能缺失更伤家长信任**（详见「家长信任核心」的定位分析）。

## What Changes

**老师端（teacher/）**

- `http.js` 新增通用 multipart 上传能力（`uni.uploadFile`，复用 token 注入与错误归一化约定）——**三端可复用的平台能力**
- 选图后**立即上传**到统一通道 `POST /common/upload`（`biz_type=growth`），拿回 `file_id` 等凭证；携带上传进度与失败重试提示
- 发布/保存时把文件引用随记录提交；编辑态**回显已有照片**、可增可删
- 页面承诺文案与实现对齐（「系统自动识别孩子」属另案，本轮先不承诺）

**后端（server/）**

- `TeacherGrowthInput` 新增 `media` 入参（`file_id` 维度，含引用层描述 `ref_desc`；编辑态保留清单 + 新上传引用）
- 创建记录：按 `file_id` 查 `files` 表取展示字段，写 `growth_media`（含 `ref_desc`）并绑定 `log_id`
- 更新记录：新引用落库、保留清单外的既有媒体从记录移除、全程事务
- 校验：`file_id` 必须存在、未被其他记录绑定；编辑态 `media.id` 必须属于本记录；跨记录引用被拒
- **不新增上传端点、不新增 OSS 依赖、不改 Wire**——复用既有 `POST /common/upload`
- 既有 `POST /teacher/growth/media/upload`（形状错、零调用）在契约中标记废弃

**素材管理设计基线落地（`docs/media-management-design.md` + `openspec/specs/media-assets/`）**

本变更是首个遵循素材管理设计基线的接入方，同时落地基线的数据地基：

- **L1 文件层**：`files` 新增 `caption`（素材固有描述）与 `tags`（受控标签，字典驱动）；分类字段（`category`）延后至素材库功能启动时加（口径未定，不抢跑）
- **L2 引用层**：`growth_media` 新增 `ref_desc`（本次使用的描述）；照片描述在发布页可填
- **生命周期分工**（按基线 §3/§4/§5）：编辑移除照片 = 解除绑定（不删文件行）；替换 = 新文件行 + 引用转向；可见性随 `growth-visibility` 既有规则，文件层不承载可见性；孤儿清理机制不在本变更（基线已定规则，清理实现待素材库启动）

**契约（backend/api/）**

- `api-teacher/06-growth.md`：`POST/PUT /teacher/growth/logs` 的 `media` 入参语义（`file_id` 维度、全量替换、编辑态保留清单）；`media/upload` 段标记废弃并指向 `/common/upload`

**明确不做什么（Out of scope）**

- **不做**照片级学生关联（`growth_media_students` 写入路径）与 AI 识别——「自动按孩子分发」的承诺属后续变更
- **不做**家长端新增页面/改动（家长 feed 与详情已消费 `media`，管道接通即生效）
- **不做**其他业务场景（请假凭证、接送照片、反馈）的媒体接线
- **不做**孤儿素材（`log_id IS NULL`）的清理机制（先确保正常路径不留孤儿；清理策略另案）
- **不做**媒体排序拖拽、视频封面自定义
- **不做**「实时照片」（活动现场即时上传供家长实时看到）——涉及发布前可见性，属独立产品决策，见 design Open Questions

**契约变更声明**

本次将修改 `backend/api/api-teacher/06-growth.md`，属于契约变更：为 `POST/PUT /teacher/growth/logs` 补记媒体入参语义（新增响应不涉及的请求字段，不新增任何响应字段）、按实现决策校正 `media/upload` 段形状。

## Capabilities

### New Capabilities

- `growth-media-pipeline`: 成长记录媒体的上传与持久化——选图后真实上传、随记录落库并挂载、编辑态回显与增删、素材归属校验

### Modified Capabilities

（无。本变更不改变既有需求的行为——家长端可见性与呈现规则已在 `growth-visibility`/`growth-feed` 确立且不变。）

## Impact

**后端（Go）**

- `server/internal/service/teacher_portal_service.go` — `TeacherGrowthInput` 的 `media` 入参、`CreateGrowthLog`/`UpdateGrowthLog` 的 `file_id` 校验与 `growth_media` 事务
- `server/internal/handler/http/teacher_portal.go` — 不改（媒体入参随既有 JSON body 进入；上传复用 `/common/upload`）
- `server/internal/app/router.go` — 不改（无新增/移除路由）

**契约**

- `backend/api/api-teacher/06-growth.md` — 媒体入参与 `media/upload` 形状

**老师端小程序**

- `teacher/src/api/http.js` — 新增 multipart 上传能力（基于 `uni.uploadFile`）
- `teacher/src/api/teacher-growth.js` — 上传封装
- `teacher/src/pages/growth/create.vue` — 选图→上传→提交的完整链路；编辑态媒体回显与增删

**测试**

- `docs/tests/TEST_CATALOG.md` — 补录 TC-NEW-GMP-*（上传落库 / 挂载 / 编辑增删 / 归属拒绝）

**不改动**

- `server/internal/domain/` — `GrowthMedia` 表结构不变（`log_id` 可空设计已支持先传后挂）
- `parent/`、`admin/` — 不涉及
