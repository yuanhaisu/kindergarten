# Proposal

## Why

产品方案承诺「老师批量上传照片，**系统自动按孩子分类推送给对应家长**」（家长信任核心的兑现点）。现状是：照片与孩子的关联（`growth_media_students` 表）**从未有写入路径**——家长打开一条班级活动记录，只能在 20 张合影里自己翻找孩子。这是「成长记录」从「班级流水账」升级为「我的孩子」体验的最后一块拼图。

同时，SaaS 多园所的业务节奏高度重叠（户外课后集中上传、午睡前批量发布），识别服务必须按**异步队列 + 限流**设计，否则峰值会撞穿云服务配额并拖慢发布路径。

**接入断言已验证**（规划期完成，含一次选型翻盘）：

| 断言 | 结论 |
|---|---|
| 云供应商选型 | 阿里云 facebody **已退市**（2026-10-11 停止新开通）→ 改选**腾讯云 IAI 人脸识别**（长期产品、每月 1 万次免费额度、URL/Base64 双输入、分库搜索） |
| 密钥边界 | **人脸服务的密钥、签名、调用一律不出后端**——客户端只上传照片、只看结果 |
| 图片输入 | 腾讯云支持 Base64 输入——后端从 OSS 取图后直接送识别，无跨云可达性问题 |
| 儿童脸质量 | **待实测**：官方建议阈值（70/80/90 分档）是成人脸调校的，实施第一步必须用 20-30 张真实儿童照片校准 |

## What Changes

### 照片点名（人工通路，先于 AI 可用）

- **老师端**：照片区支持批量点名（多选照片 → 一次勾选学生）+ 单张微调；不强制、不点名的照片为「通用照片」
- **家长端**：记录详情照片区**默认「只看我的孩子」+ 可切换「看全部」**；含自家孩子的照片呈现标识；列表卡片不做照片级筛选（保持现状）
- **可见性**：只有**确认**过的点名对家长生效

### 学生人脸底库（admin 端）

- 学生录入/编辑支持上传 1-5 张人脸照片（走统一上传通道）；后端经 `FaceProvider` 注册到云端底库
- 底库状态可视化（未注册/已注册/失败重试）；按园所隔离（云侧 `GroupId = kg_{id}`）

### 人脸识别（腾讯云 IAI，密钥不出后端）

- **`FaceProvider` 抽象**：`tencent` 实现 + `mock` 实现（mock 让无凭证环境也能开发/测试全链路）；密钥与签名只存在于后端
- **异步识别队列**：记录发布后照片入队（`recognition_jobs`）；worker 池消费；**发布路径零等待**——老师发布完立刻可继续操作，建议稍后（正常负载 < 2 分钟）就绪
- **AI 建议**：识别结果写为 `status=0`（建议态），老师端与编辑页呈现「建议：某某（置信度）」；**高置信预勾、低置信提示不预勾、可拒绝**
- **SaaS 峰值防护**：
  - 限流器：全局调用速率受 `[face].qps` 控制（保护配额）
  - 租户公平：任务认领按园所轮转，防单园饿死其他租户
  - 预算护栏：日调用上限（`[face].daily_budget`），触顶暂停并告警
  - 降级：云服务故障/超限 → 任务留队重试；**人工点名路径完全不受影响**
  - 灰度：`[face].enabled` 按园所白名单上线

### 明确不做什么（Out of scope）

- **不做**家长端的人脸授权流程（合规必需，但属独立变更——底库采集上线前必须完成，本变更在文档中标注该前置依赖）
- **不做**「识别自动确认」（跳过老师确认直接对家长生效）——信任模型是「AI 建议、老师背书」
- **不做**人脸特征值本地存储（特征在腾讯云侧，我方库只存照片 URL 与云端 PersonId 映射）
- **不做**在线实时识别（识别一律走队列；发布时不做同步等待）
- **不做**其他模块的人脸能力（接送核对等，另案）

**契约变更声明**

本次将修改：`backend/api/api-teacher/06-growth.md`（media 入参/出参补 `student_ids` 与识别状态）、`backend/api/api-admin/` 学生相关（底库照片上传与注册状态）、`backend/api/api-parent/04-growth.md`（详情 media 项补点名信息）。新增老师端识别状态查询端点。供应商接入配置（密钥）落在 `configs/app.ini` 的 `[face]` 段（已 gitignore）。

## Capabilities

### New Capabilities

- `photo-tagging`: 照片与孩子的关联矩阵——人工点名（批量+单张）、AI 建议的确认/拒绝工作流、以及家长端按点名筛选的呈现规则
- `student-face-registry`: 学生人脸底库——底库照片的上传、云端注册、状态管理与多租户隔离
- `growth-recognition-queue`: 识别任务的异步处理——任务入队与消费、限流、租户公平、预算护栏、降级与重试

### Modified Capabilities

（无。点名数据是新增语义；`growth-media-pipeline` 已确立的媒体管道行为不变，本变更在其上叠加识别与点名。）

## Impact

**后端（Go）**

- `server/internal/domain/` — 新表：`student_faces`（底库注册）、`recognition_jobs`（识别任务）；`growth_media_students` 增加 `status`/`confidence` 列
- `server/internal/ai/face/` — **新包**：`Provider` 接口 + `tencent`（腾讯云 IAI SDK）+ `mock` 实现
- `server/internal/service/` — 底库管理服务、识别 worker 池与限流器、点名写入/读取（三个媒体出参函数扩展）
- `server/internal/handler/http/` + `router.go` — 底库端点（admin）、识别状态端点（teacher）
- `server/internal/config/` + `configs/app.ini` — `[face]` 段（provider/enabled/qps/daily_budget/阈值/密钥）
- Go 依赖：新增腾讯云 SDK（`tencentcloud-sdk-go/tencentcloud/iai`）
- Wire DI：新服务的构造注入 → `make wire`

**契约**

- `backend/api/api-teacher/06-growth.md`、`backend/api/api-admin/`（学生）、`backend/api/api-parent/04-growth.md`

**老师端小程序**

- `create.vue`（点名面板：批量+单张；建议态呈现）、`detail.vue`（照片标签）、`teacher-growth.js`（识别状态轮询）

**admin web**

- 学生管理页（`admin/pages/students/`）：人脸照片上传与底库状态

**家长端小程序**

- `pages/growth/detail.vue`：筛选切换（只看我的/全部）+ 照片标识

**测试**

- `docs/tests/TEST_CATALOG.md` — 补录 TC-NEW-PTG-*（点名/建议）、TC-NEW-FRG-*（底库）、TC-NEW-RCQ-*（队列）
- 真实照片质量实测报告（阈值校准依据）

**不改动**

- `growth-media-pipeline` 的既有行为、上传通道、OSS 配置、其他端其他模块
