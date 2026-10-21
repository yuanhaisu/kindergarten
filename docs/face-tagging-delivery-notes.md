# 照片点名与人脸识别 — 交付与验收说明

> 变更：`photo-tagging-face-recognition`（OpenSpec，specs 三份：photo-tagging / growth-recognition-queue / student-face-registry）。
> 本文是**逐条验收的结论记录 + 运维配置清单**，随变更归档保留。
> 老师侧使用说明见 `docs/teacher-naming-guidelines.md`；识别质量实测见 `docs/face-quality-report.md`。

## 一、验收结论（逐条对照 specs）

三段规格的全部 Requirement / Scenario 均有对应实现与验证证据；TC 状态见 `docs/tests/TEST_CATALOG.md`（PTG-0001~0020 / FRG-0001~0006 / RCQ-0001~0009 全绿）。

| 规格域 | 关键场景 | 验证层级 |
|---|---|---|
| photo-tagging | 点名关联/批量/单张微调、建议确认流、高置信一键、待确认角标、**大合影全班**、家长筛选、白名单 | 单测 + API 集成 + 真机走查（CDP 真实点击） |
| growth-recognition-queue | 不阻塞发布、限流/预算、租户公平、故障降级、**范围过滤**、灰度 | 单测 + 真实栈（本机后端 + mock provider） |
| student-face-registry | 录入/注册/重试/状态、按园隔离、密钥不出后端、底库变更不影响记录 | 单测 + admin 端走查 + 腾讯云实测 |

**真实栈端到端（9.6，2026-10-16）**：本机后端（`[face].provider=mock`）+ 老师端 IDE

1. 8 张人脸的照片（`MOCK_FACES:8`）→ 队列分型：`face_count=8`、`is_group_photo=true`、**不产逐人建议**；
2. 老师端详情页呈现「大合影 · 一键全班（当日到场 2 人）」，真实点击 → 2 名到场孩子落确认点名，**当日已批准请假的 1 名孩子被排除**；
3. 家长端出参：被点名孩子的家长在「只看我的」可见合影；请假孩子的家长默认视图看不到（`student_ids` 不含其孩子）；
4. 单人照路径回归：1 张人脸 → 高置信建议照常产出与确认；
5. 范围过滤实证：命中学号落在班级范围外时（模拟跨班相似脸），建议被滤除、不落库。

## 二、阈值取值依据

| 配置（`configs/app.ini` `[face]`） | 取值 | 依据 |
|---|---|---|
| `suggest_high` | 80 | 「可一键采纳」的置信门槛。质量初测（28 张证件照风格样本）中同人跨图分值落在 42–57，样本不足且场景不覆盖；按保守取值——高置信集合宁可小而准，避免把误认喂给「全部确认」 |
| `suggest_low` | 70 | 建议落库门槛，同时作为下发腾讯云的 `FaceMatchThreshold`（服务端过滤与云端过滤同源，避免云端低分噪音占建议位）。低于此分的命中不提示老师 |
| `group_photo_min_faces` | 6 | 人脸数 ≥ 此值即判「大合影」。**仅全班记录**走「不产建议 + 一键全班」；个体记录保留建议（建议本就限定在目标孩子）。取值在「两三个孩子的活动照仍可逐人核对」与「一张合影逐人确认注定不完整」之间取 6 |
| `qps` | 10 | 令牌桶全局限速（保护云端配额） |
| `daily_budget` | 10000 | 日调用预算护栏（触顶暂停消费、任务回滚 pending） |

> 阈值全部配置化：质量数据充分后调整只改配置，不改代码与交互。分型阈值同样**由服务端判定**并随 `tagging-status` 下发 `is_group_photo`，客户端不硬编码。

## 三、腾讯云 IAI 配置清单

| 项 | 值/约定 | 说明 |
|---|---|---|
| 产品/接口 | 腾讯云人脸识别 IAI `SearchFacesReturnsByGroup` | 按组搜索，支持多园隔离 |
| 人员库（GroupId） | `kg_{kindergarten_id}` | 一园一组，**跨园不匹配**（FRG-0005） |
| 人员（PersonId） | `kg{kgID}_stu{studentID}` | 可从 PersonId 反解学生；非本系统格式的 PersonId 防御性跳过 |
| 密钥存放 | 仅后端 `configs/app.ini` `[face].secret_id/secret_key`（可用 `FACE_SECRET_ID/FACE_SECRET_KEY` 覆盖） | **绝不下发任何客户端**；`provider=tencent` 但缺密钥时启动即报错（不静默降级 mock） |
| `MaxFaceNum` | **硬上限 10** | 云端单次最多返回 10 张人脸——即「大合影逐人名单注定不完整」的根因，D11 分型即为此而设（≥ 阈值不产建议）。`face_count` 触顶即「≥10」 |
| `MinFaceSize` | 40 | 过小人脸不返回（远景合影中后排孩子可能不入选，属预期） |
| `FaceMatchThreshold` | 取 `suggest_low`（70） | 服务端与云端同阈值 |
| 底库照片 | 每人 1–5 张（`maxFacesPerStudent=5`），经统一上传通道存 OSS | 底库变更（增删）不影响既有记录与点名（FRG-0006） |
| 待接入（独立后继变更） | `CopyPerson` / `DeletePersonFromGroup` 班级组搜索 | 设计见 design.md D11：按班建组可缩小搜索空间，但需处理升班/转班生命周期，**独立变更**切走 |

## 四、家长端内容模型（复审裁定 2026-10-16）

家长端记录详情按**点名结果**筛选，内容在家长心里的三类归属：

| 内容 | 家长端呈现 |
|---|---|
| 集体照片（含大合影） | 与「孩子专属」同栏可解释；点到自家孩子即进默认栏，漏标/请假时切「看全部」可见 |
| 孩子个人专属 | 默认栏（「只看我的孩子」） |
| 其他孩子的 | 仅「看全部」可见 |

推论：**「一键全班」的保守（无数据全员）与请假排除都不会让家长「看不到活动」**——集体照片始终可通过「看全部」看到，点名只决定它是否进默认筛选。因此一键全班不设二次确认/撤销（误标属可解释内容），旧记录一键全班按当前在读名单也无需日期守卫。

## 五、实际触点与设计阶段的差异（归档回溯用）

设计阶段（design.md D1–D11）与实际实现/交付的差异，均为**经复审裁定**后落地：

1. **大合影分型的适用范围**：设计表述为「人脸数达阈值即不产建议」；复审裁定改为**仅全班记录**（个体记录的建议本就限定在目标孩子，逐人确认不是伪工作）——队列 `handle` 按记录类型判定，spec/契约/SOP 同步（TC-NEW-PTG-0021）。
2. **一键全班的响应形状**：设计未要求；实现后审查发现客户端「本地并集」与服务端真相不一致（已拒绝不复活、点名与到场名单的 TOCTOU），补 `named` 最终名单回读下发（契约新增字段）。
3. **建议轮询的容错**：设计未覆盖；复审判定轮询 MUST 退避自愈（原「遇错即停」会让建议永不到达），替换为 setTimeout 调度 + 退避（TC-NEW-RCQ-0011）。
4. **家长端内容模型**：交付期裁定——「集体/孩子专属」同栏可解释、「其他孩子的」另栏；据此明确一键全班不设二次确认/撤销、旧记录不设日期守卫，家长端呈现保持现状（漏标时经「看全部」可见）。
5. **实际触点清单**（相对设计「跨端触点清单」的落点核对）：`server/internal/ai/face/`（face+mock+tencent）· `service/{recognition_queue,teacher_portal_service,face_registry_service}.go` · `handler/http/{teacher_portal,face_registry}.go` · `app/router.go` + `middleware/admin_perm.go` · `domain/face.go` + `config` · `cmd/{facecheck,dbcheck}` · 契约 `api-teacher/{06,09}` / `api-parent/04` / `api-admin/03` · 老师端 `growth/{create,detail,index}.vue` + api · 家长端 `growth/detail.vue` · admin `pages/students` + api/types · `docs/migrations/2026-10-15-face-tagging-tables.sql` · `docs/tests/TEST_CATALOG.md`（PTG 21 / FRG 6 / RCQ 11）。设计清单中「token」触点未改动（本轮不涉及鉴权形状）。

## 六、已知边界（交付时与老师说明）

1. **识别只覆盖已录入底库的孩子**；未录入者任何照片都不会被建议，老师按画面手动点名即可（点名不依赖识别）。
2. **单人照是 AI 点名的好输入**；集体照走「一键全班」（当日到场），个别不在画面的孩子由老师增减。
3. 建议**不确认不生效**；拒绝过的建议不再提示（一键全班也不会复活被拒绝的项）。
4. 家长端默认「只看我的孩子」；被点名的照片才进默认视图，其余照片可切「看全部」。
