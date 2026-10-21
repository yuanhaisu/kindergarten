# OpenSpec 使用指南

> 本项目的规格化工具是 **OpenSpec**。本文是日常查阅手册。
>
> - 语义所有权与工作流总览 → 根目录 `CLAUDE.md` 的「Spec Workflow」一节
> - 项目约束与 artifact 规则 → `openspec/config.yaml`
> - 历史规格（已冻结）→ `docs/specs/`

---

## 1. 两套命令，跑在两个地方

**这是新手最容易栽的坑。**

| 场所 | 命令 | 作用 | 频率 |
| --- | --- | --- | --- |
| **Claude Code 聊天框** | `/opsx:...` | **日常干活**：提议、实现、归档 | 天天用 |
| **PowerShell** | `openspec ...` | **引擎维护**：看状态、体检、校验 | 偶尔用 |

```
        PowerShell                              Claude Code 聊天框
   ┌──────────────────────┐               ┌──────────────────────────────┐
   │  $ openspec list     │    提供       │  /opsx:propose add-xxx       │
   │  $ openspec doctor   │   ────────►  │  /opsx:apply                 │
   │  $ openspec view     │   引擎能力    │  /opsx:archive               │
   └──────────────────────┘               └──────────────────────────────┘
        跑 openspec 这里                       跑 /opsx:* 这里
```

> ⚠️ 在 PowerShell 里打 `/opsx:propose` **没反应**——斜杠命令是 AI 聊天指令，不是 CLI。
> 这是 OpenSpec 官方文档点名的「最常见栽跟头点」。

**Claude Code 的调用形式**：`/opsx:propose`（带冒号）。命令文件在 `.claude/commands/opsx/`。

---

## 2. 先判断要不要走 OpenSpec

在聊天框里**直接描述需求**，Claude 会按 `CLAUDE.md` 的路由表自己判断。但你心里有杆秤更稳：

```
这次改动会碰到几层？

  Go domain / repo / service / handler
  Casbin policy / router.go
  backend/api/ 契约
  teacher 页面
  parent 页面
  admin 页面
  design token
  ─────────────────────
  数一下「会改」的层数：

  1–2 层      → 直接说需求，别碰 /opsx:*
  ≥ 3 层      → /opsx:propose
  原因不明    → 先 systematic-debugging，定位清楚再决定
  做不做没想清 → /opsx:explore
```

### 路由表

| 变更类型 | 用什么 | 产物落在哪 |
| --- | --- | --- |
| 触点 < 3 层（单点 bug、typo、样式、配置） | **不写规格**，直接改 | 代码 + commit message |
| 诊断困难的 bug | `superpowers:systematic-debugging` | 代码 + commit message |
| 触点 ≥ 3 层的功能切片 | **OpenSpec**：`/opsx:propose → apply → archive` | `openspec/changes/` → `openspec/specs/` |
| 做不做的概念决策 | `/opsx:explore` | 无（思考对话） |

---

## 3. 大改示例（触点 ≥ 3 层）

**场景**：家长端积分商城——家长用积分兑换奖励，老师端核销，后台配置奖品。

**触点清点**：Go domain/repo/service/handler + Casbin + router + `backend/api/` + parent 页面 + teacher 页面 + admin 页面 + token = **8 层** ✅ 走 OpenSpec

### 聊天框完整流程

```text
# ── 第 1 步（可选）：想清楚 ──────────────────────────
/opsx:explore

你：家长端积分体系现在只有 PointAccount / PointTransaction，
    我想加兑换商城，但不确定奖品库存和核销链路怎么设计。

Claude：读代码 → 给 2–3 个方案 + 权衡 → 你选定一个

# ── 第 2 步：提议，产出规划工件（不写代码）──────────
/opsx:propose add-points-mall

Claude 生成：
  openspec/changes/add-points-mall/
    proposal.md      ← 为什么做、明确不做什么
    specs/...        ← SHALL + Scenario，带 TC ID
    design.md        ← 必含「跨端触点清单」（rules.design 强制）
    tasks.md         ← 按触点分组 + [P] 并行标记

# ── 第 3 步：你审阅、批准 ───────────────────────────
你：proposal 的 out-of-scope 划掉了「积分过期」，对。
    design 的触点清单漏了 Casbin policy，补上。批准实现。

# ── 第 4 步：实现 ───────────────────────────────────
/opsx:apply

Claude 按 tasks.md 分组实现，每组做 checkpoint
（rules.tasks：实现完必须逐条对照 specs 验证）

# ── 第 5 步：归档，delta 合并进活基线 ────────────────
/opsx:archive

产物落到 openspec/specs/points-mall/spec.md，成为活文档
```

### 关键约束（已注入 config.yaml，自动生成时生效）

- `propose` **只产规划工件，不动代码**（skill 内置 planning boundary）。写完会停下来等你批准
- 中途发现规格有问题 → `/opsx:update` 改工件，**不要带着错误规格硬写代码**
- `design.md` 必然包含「跨端触点清单」，每项写「会改+改什么」或「不改+为何不受影响」
- `specs` 里每条验收场景必须引用 `docs/tests/TEST_CATALOG.md` 的 TC ID
- 涉及跳转必须核对 tabBar 差异（家长端 4 tab / 老师端无 tabBar）

---

## 4. 小改示例（触点 < 3 层）

**场景**：家长端首页切换孩子没生效（对应提交 `42079cc`）。

**触点清点**：parent 一个页面的状态绑定 = **1 层** ❌ 不走 OpenSpec

### 聊天框就这么说

```text
你：家长端首页切换孩子没生效，帮我查一下
```

然后按情况走：

| 情况 | 走什么 |
| --- | --- |
| 原因清楚、改动小 | Claude 直接改，commit message 说清 |
| 症状诡异、原因不明 | 触发 `superpowers:systematic-debugging` |
| 改着改着发现触点变多 | 停下来，转 `/opsx:propose` |

**全程不碰 `/opsx:*`，不开 PowerShell。**

### 其它不用 OpenSpec 的例子

| 改动 | 为什么不用 |
| --- | --- |
| token 用错（`$spacing-md` 写成硬编码） | 单文件样式 |
| 按钮文案、图标换掉 | 单点 |
| `pages.json` 加个页面 | 配置 |
| 修 `wire.Bind` 漏绑定 | 单点，CLAUDE.md 已有指引 |
| 某列表空状态没显示 | 单页面 |

---

## 5. 中间流程命令

| 我想… | 命令 | 说明 |
| --- | --- | --- |
| 想清楚一个模糊想法 | `/opsx:explore` | 零成本思考伙伴，不写代码不建工件 |
| 提一个大改 | `/opsx:propose <变更名>` | 一次产出 proposal + specs + design + tasks |
| 改已有的规划工件 | `/opsx:update` | 发现规格有误时用这个，不要硬写代码 |
| 按批准的规格开工 | `/opsx:apply` | 实现 tasks.md |
| 把 delta 同步进活基线 | `/opsx:sync` | 可选，archive 时通常会做 |
| 归档完成的变更 | `/opsx:archive` | delta 合并进 `openspec/specs/` |

### PowerShell 侧（引擎维护）

| 命令 | 用途 |
| --- | --- |
| `openspec list` | 看有哪些变更在飞 |
| `openspec list --specs` | 看活基线里已有哪些能力规格 |
| `openspec status --change <名>` | 看某变更的工件完成度 |
| `openspec doctor` | 体检：root 健康、配置是否被认 |
| `openspec validate --all` | 校验工件是否符合 schema |
| `openspec show <名>` | 查看某变更/规格全文 |
| `openspec view` | 浏览器仪表盘（唯一真正交互式） |
| `openspec context --json` | 打印生效的项目上下文 |
| `openspec update` | 升级 OpenSpec 后刷新 skill 文件 |
| `openspec config profile` | 切换 core / expanded 工作流 |

**收尾时建议跑一遍**：

```powershell
openspec list          # 确认变更状态
openspec status --change <变更名>
openspec validate --all
openspec doctor
```

---

## 6. 扩展命令（当前未启用）

当前装的是 **core profile**，只有 6 个命令：`propose / explore / apply / update / sync / archive`。

需要 `new` / `continue` / `ff` / `verify` / `bulk-archive` / `onboard` 时：

```powershell
openspec config profile     # 选 expanded
openspec update             # 应用到本项目
```

其中 `verify`（对照工件校验实现）和 `onboard`（带讲解的端到端演示）对新人有价值。

---

## 7. 语义所有权（谁是真相源）

规格层**只占「行为 + 验收」这一格**，不得侵入下列已饱和的语义：

| 语义 | 唯一 SoT | 规格里怎么写 |
| --- | --- | --- |
| 接口形状（路由/字段/包络） | `backend/api/` | 只写「调哪个端点、期望什么行为」，不复述字段清单 |
| 数据结构（表/字段/关联） | `server/internal/domain/` | 只写「需要什么数据约束」，不复述 struct |
| 架构铁律 / 硬规则 / 已知坑 | `CLAUDE.md` | 不复述，引用即可 |
| 样式 / 间距 / 色值 | `shared-design-tokens/tokens.json` | 写意图（「用语义成功色」），不写色值 |
| 测试用例编号 | `docs/tests/TEST_CATALOG.md` | 引用 TC ID |
| **行为需求 + 验收标准** | **`openspec/specs/`** | **这是规格层的地盘** |

---

## 8. 踩坑清单

| 坑 | 症状 | 解法 |
| --- | --- | --- |
| 在 PowerShell 打 `/opsx:propose` | 没反应 | 去 Claude Code 聊天框打 |
| 期望 Claude 自动跑 `openspec list` | 它不跑 | 说「跑一下 openspec list」，或自己开 PowerShell |
| 小修也走 `/opsx:propose` | 仪式感劝退 | 路由表：触点 < 3 层直接改 |
| propose 完直接写代码 | 跳过审批 | propose 是规划边界，等用户批准才 apply |
| 规格写错硬着头皮实现 | 产出没人信 | `/opsx:update` 先改规格 |
| archive 前没验证 | 又一处腐烂 | `rules.tasks`：未验证不得 archive |
| 引用 `backend/model/` | 引用了停更层 | 用 `server/internal/domain/` |
| 假设「MCP 导航过了 = 功能对了」 | 验证结论失真 | 交互验证必须真实点击，见 CLAUDE.md 测试纪律 |

---

## 9. 为什么是 OpenSpec（决策摘要）

**结论**：用 OpenSpec，不引入 GitHub Spec Kit。

**撑住结论的三条**：

1. **防腐烂（R1）**——本项目失败模式是「写了就烂」（`docs/specs/`、`backend/model/`、`backend/model/README.md` 三处实证）。OpenSpec 的 delta→archive 机械同步是工具化的防腐机制；Spec Kit 的 evolving-specs 靠手改，会重蹈覆辙
2. **不制造新真相源（R2）**——SoT 已饱和。Spec Kit 默认产出 `contracts/` / `data-model.md` 会侵入 `backend/api/` 与 `domain/` 两格
3. **摩擦低（R4）**——solo + AI 驱动，动作式（随时 propose/apply/archive）比阶段序列（specify→plan→tasks→implement→converge）贴合实际节奏

**Spec Kit 的真优势已移植进 `openspec/config.yaml`**：

| Spec Kit 强项 | 移植到 |
| --- | --- |
| `/speckit-plan` 的跨端触点结构 | `rules.design` 强制「跨端触点清单」 |
| `implement→converge` 验证收敛 | `rules.tasks`「未验证不得 archive」 |

**翻盘条件**（任一命中就重新评估 Spec Kit）：

- 功能切片远多于小修
- 决定让规划步骤直接产出 `backend/api/` 契约
- 出现第二位协作者，需要宪法 + 评审流程
- OpenSpec 用 2–3 个 change 后规格照样腐烂

---

## 10. 首次实战验证记录（2026-10-04）

配置上线后用一个假想变更 `add-parent-growth-filter`（家长端成长时光按类型筛选）跑了一遍完整 `propose` 流程，目的有两个：验证 rules 真的生效；验证需求本身是否站得住。结论如下。

### 10.1 rules 注入：全部生效

`openspec instructions <artifact> --json` 的响应里，`context` 与 `rules` 字段**逐条带着 `config.yaml` 的内容**。工件产出后逐项核对：

| 规则 | 注入 | 落地 | 证据 |
| --- | --- | --- | --- |
| context 职责边界 | ✅ | ✅ | spec 只写行为，未复述字段清单/struct |
| proposal 写清不做 | ✅ | ✅ | 独立「明确不做什么」6 条 |
| proposal 点名端 | ✅ | ✅ | 「涉及端：teacher、parent。不涉及 admin」 |
| proposal 契约变更声明 | ✅ | ✅ | 显式「BREAKING（契约变更）」 |
| specs SHALL + `#### Scenario:` | ✅ | ✅ | 5 Requirement / 11 Scenario |
| specs 引 TC ID | ✅ | ✅ | 每场景挂 TC-NEW-GRW-*，tasks 登记补录 |
| design 跨端触点清单 | ✅ | ✅ | 14 行表，每行「会改+改什么」或「不改+为何」 |
| design tabBar 核对 | ✅ | ✅ | 独立小节，结论「页内交互，不涉及跳转」 |
| design AutoMigrate + dbcheck | ✅ | ✅ | 含 `go run ./cmd/dbcheck` 与手工 SQL 兜底 |
| design 用 ASCII 不用 Mermaid | ✅ | ✅ | 两张 ASCII 流程图 |
| tasks 按触点分组 | ✅ | ✅ | 6 组不混排 |
| tasks `[P]` 并行标记 | ✅ | ✅ | 4 个任务标 `[P]` |
| tasks 每组验证 | ✅ | ✅ | 每组末尾是验证任务 |
| tasks 真实点击验证 | ✅ | ✅ | 明写 `wechat_automator` tap/input，禁用 navigate 替代 |
| tasks 杀 uni watcher | ✅ | ✅ | 按 PID 杀 node 的 PowerShell 片段 |

`openspec status` → 4/4 artifacts complete；`openspec validate --changes --strict` → 全绿。

### 10.2 需求本身：方向真实，方案不合理 → 已否决并删除

对抗性审查发现三处硬伤，该 change 已删除，**不作为真实需求推进**：

| # | 问题 | 证据 |
| --- | --- | --- |
| 1 | **分类法是凭空拟的** | 设计稿 `design-data/02-家长端-成长时光页.json` 的筛选区是「校园生活/活动照片/作品展示/获奖记录」；假想需求写的是「日常观察/里程碑/作品展示」；数据模型里**没有任何类型字段**。三套互不相同，spec 里却已固化成 11 条带 TC 的验收场景 |
| 2 | **老师端打标入口在设计里不存在** | `design-data/teacher-04-成长记录页.json` 创建表单只有 饮食/睡眠/情绪/表现/照片/内容，**无类型选择器**。设计稿自身不自洽：家长端画了分类筛选，老师端没画分类来源 |
| 3 | **「硬编码 3 个常量」违背已有架构意图** | `server/internal/domain/dict.go` 已实现完整的 `SysDictType`/`SysDictItem` 体系，注释点名 `growth_category`，且明写「每个幼儿园独立配置」。拿「当前只有 3 个取值」否定「本来就要可配置」是把实现现状当设计约束 |

**站得住的部分**（不是伪需求，留给未来）：设计稿明画了筛选区；首页显示 `236 成长记录 / 1286 精彩照片`，这个量级下不做浏览/筛选家长翻不动；现有那排假 tab 是 `c596073`（stop showing fabricated data）修掉的问题的残留，必须处理。

**当下动作**：只做 1 层清理——删掉假 tab（触点 < 3 层，按路由表不走 OpenSpec）。
**阻塞待定**：分类法到底用哪套？设计稿那套 / 别的 / 走 `dict.go` 的 `growth_category` 让园所自配？这个问题只有产品/人能回答，定下来之前不写新 spec。

### 10.3 暴露出的 rules 漏洞（已修）

**失败模式**：`rules.proposal` 的「触点 < 3 层不走 OpenSpec」在探索阶段确实拦住了范围蔓延（发现类型字段不存在时停下来问了），但**没有任何规则拦住把随手拟的分类法写进 spec**。`rules.specs` 管了格式（SHALL/Scenario/TC ID），没管**取值从哪来**。

**已补进 `openspec/config.yaml`**：

- `rules.proposal` 新增：涉及新增分类/类型/标签/状态等枚举语义时，必须先查三处权威来源（`design-data/` 设计稿、`dict.go` 字典体系、代码已有常量）；三处都没有 → 列为待产品决策的阻塞项并停下
- `rules.specs` 新增**取值溯源**：规格里任何枚举取值必须能指到权威来源，禁止凭空拟取值再写成 SHALL + Scenario；无来源的取值标 `[待定]` 且不得写入验收场景

**教训一句话**：rules 能管住格式，管不住事实。枚举取值这类「必须有出处」的东西，要在规则里显式要求溯源——否则 AI 会把「写得像真的」当成「是真的」。

---

## 11. 第二个 rules 漏洞：契约字段必要性（2026-10-04 补修）

### 11.1 漏洞是什么

第 10 节的「取值溯源」管住了**枚举取值**（分类名/状态名），但**管不住字段必要性**。

同日做家长端 `GET /parent/growth/feed` 的字段审查时暴露：契约 14 个字段里有 **5 个是纯噪音**（36%）——设计稿没画、业务说不清、实现没返回、前端没渲染，但契约里躺着：

| 多余字段 | 为什么多余 |
| --- | --- |
| `likes_count` | 设计稿互动栏只有 `点赞`/`已点赞` 状态，**从不显示数字**；评论才有数字 |
| `media_count` | 设计用真图网格，`media.length` 即得，是冗余派生值 |
| `target_type` | 服务端可见性过滤的内部态，卡片不渲染 |
| `class_name` | 卡片不显示班级（班级只在孩子信息卡） |
| `related_students*` | 详情页语义（「相关孩子（28人）」），塞进列表是接口粒度画错 |

**根因**：契约按「社交 feed 接口惯例」加字段，没人回头对设计稿。这是**「看起来专业的接口」的症状**——惯性加上去，四方都不用。

### 11.2 已补进 `openspec/config.yaml`

| 位置 | 规则 |
| --- | --- |
| `rules.proposal`【契约字段溯源】 | 契约变更不得夹带私货。新增/修改的响应字段必须点名设计稿出处或明确业务场景。指认不出的视为多余。「社交接口惯例上会有」「以后可能用得上」「加上没坏处」都不是理由 |
| `rules.design`【契约字段溯源】 | 触点清单里标注「会改（契约变更）」的，必须给出**字段级清单**，每字段指到 `design-data/` 的具体 UI 元素；还要说明为什么不能由已有字段派生 |
| `rules.design`【接口粒度】 | 只有详情页渲染的字段不得塞进列表接口 |

### 11.3 与第 10 节漏洞的关系

| 漏洞 | 症状 | 修法 |
| --- | --- | --- |
| 第 10 节：枚举取值无出处 | 凭空拟分类法，写成 11 条假 TC | 取值必须溯源到设计稿/字典/代码常量 |
| 本节：字段无出处 | 契约塞了设计不渲染的字段 | 字段必须溯源到设计稿元素或业务场景 |

**同一个病**：AI（和人）会把「写得像真的」当成「是真的」。格式规则挡不住，**溯源规则才挡得住**。

### 11.4 审查详情

完整逐字段假设映射、对抗性拷问、实施清单见 **`docs/growth-feed-field-review.md`**。

> ⚠️ 该审查的实施（补 handler / 砍契约 / 挪字段 / 修模板映射）触点 ≥3 层，**未执行**。按第 2 节路由表应走 `/opsx:propose`，审查文档是 propose 的输入材料。另有 2 项设计超前（`badge` 奖章、`category` 活动上下文）阻塞于产品决策。
