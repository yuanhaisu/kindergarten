# 新会话开场提示词（Kickoff Prompt）

> 这是"一模块一会话"工作流的入口。**每次开新会话，你只需复制下面的一段话发给 AI 即可**。
>
> AI 会自己读文档、确认状态、建工作簿、开工、收尾。

---

## 🚀 最常用：按计划自动继续（推荐）

复制这一句发给 AI：

```
请按 backend/api/api-admin/KICKOFF_PROMPT.md 的 "自动续接流程" 继续执行 admin 接口计划。
```

AI 将自动走完下面的"自动续接流程"。

---

## 🎯 指定模块开工

如果你想明确指定本次会话做哪个模块，用这一句：

```
请按 backend/api/api-admin/KICKOFF_PROMPT.md 开始做 **02 classes** 模块（只做这一个模块，完成后结账）。
```

把 `02 classes` 换成目标模块即可：
- `01 dashboard`, `02 classes`, `03 students`
- `04 teachers`, `05 parents`, `16 admins`, `17 roles`, `18 settings`
- `06 attendance`, `07 leaves`, `08 pickup`
- `09 growth`, `10 tasks`, `11 points`, `12 achievements`
- `13 notices`, `14 announcements`, `15 audit`, `19 logs`, `20 export`

---

## 🔧 S0 脚手架（第一次启动时用）

```
请按 backend/api/api-admin/KICKOFF_PROMPT.md 执行 S0 脚手架（8 项一次性做完）。
```

---

# 自动续接流程（AI 读到这里后按此执行）

**你（AI）收到上面任一提示词后，必须严格按照以下步骤执行，不要跳过任何一步。**

## 步骤 0 · 打开钥匙

读以下 3 份文件（**必读，按顺序**）：

1. `backend/api/api-admin/SESSION_HANDOFF.md` — 确认 `Current Sprint`、`Current Module`、`Next Action`、`Blockers`
2. `backend/api/api-admin/PROGRESS.md` — 确认当前 Sprint 未完成项、S0 脚手架是否已完成
3. `backend/api/api-admin/EXECUTION_PLAN.md` 的 §5（Recipe）+ §6（接口 DoD）+ §7（模块 DoD）+ §9（约定）

## 步骤 1 · 确定本次会话目标

**优先级判定**：

1. 如果 `SESSION_HANDOFF.md` 里有 `Blockers` 未解决 → 先汇报 blocker，等用户决策
2. 如果用户在提示词里显式指定了模块（如"做 02 classes 模块"）→ 目标 = 该模块
3. 如果用户说"S0 脚手架" → 目标 = 完成 `PROGRESS.md` 里 S0 剩余项
4. 如果用户说"自动续接" → 目标 = `SESSION_HANDOFF.md` 的 `Next Action`
5. 如果 S0 未完成但用户要做业务模块 → 警告并询问：是否先做 S0？（强烈建议先做）

**汇报给用户**（必须）：
- 本会话目标：`<Sprint> / <模块> / <接口数量>`
- 预期产出：`<接口列表>`
- 预计改动文件：`<文件清单>`
- 等待用户确认（简短"好/继续"即可开工）

## 步骤 2 · 开工前准备（按业务模块时）

**仅在本次要做业务模块时执行此步**：

1. 读接口规格：`backend/api/api-admin/<XX>-<module>.md`（整份）
2. 读前端封装：`admin/api/<module>.ts`（整份）
3. 读前端类型：`admin/types/api.ts` 中与该模块相关的类型
4. 读前端页面：`admin/pages/<module>/index.vue`（主要看接口用在哪、字段怎么展示）
5. 读 domain：`server/internal/domain/<相关>.go`（看字段是否够用）

**创建本模块工作簿**：

1. 读 `backend/api/api-admin/workbooks/_template.md`
2. 在 `backend/api/api-admin/workbooks/<XX>-<module>.md` 创建工作簿实例（若已存在则读取继续）
3. 填入模块元信息、接口清单、domain 核对结果

## 步骤 3 · 按 Recipe 实现

- 对模块内每个接口，严格按 `EXECUTION_PLAN.md` §5 的 9 步走。
- 顺序：`GET list → GET detail → POST → PUT → DELETE → 业务动作`。
- **每完成 1 个接口**：
  1. 运行 `go build ./...` 确认编译通过
  2. 在 `PROGRESS.md` 对应行打勾 + 写日期
  3. 在工作簿里更新"接口完成进度"
  4. （可选）用 `curl` 或写 `.http` 文件在 `samples/` 里留样

## 步骤 4 · 模块收尾

当模块所有接口做完：

1. 跑一次 `go generate ./internal/app` 确认 Wire 生成无误
2. 跑 `go build ./...`
3. 提示用户启动 `server` + `admin` 跑一次页面 CRUD（用户手动确认）
4. 在 `PROGRESS.md` 把模块状态从 🚫 改为 ✅
5. 在 `README.md` 的模块索引表把状态列也改成 ✅
6. 在工作簿里填 DoD checklist 并写 `Status: Done`
7. 建议 commit：`feat(admin): complete <module> module (x/87)`

## 步骤 5 · 会话结账（最重要）

**不做这一步不能结束会话**。更新 `SESSION_HANDOFF.md`：

1. 追加 `Session Log` 条目（日期、Completed、Decisions、Next Action、Blockers、Last Modified Files）
2. 更新顶部 `Current Status`（Sprint / Module / Completed So Far / Overall Progress）
3. 更新 `🎯 Next Action` 为下一个模块的第一个接口
4. 如果有新 blocker，写入 `🚧 Blockers` 并@用户

## 步骤 6 · 告知用户下次怎么继续

在会话最后输出类似这样的"交接语"：

```
本次会话已结账：
- 完成：<module> 模块（x 个接口）
- 进度：y / 87
- 下次会话用这一句启动：
  > 请按 backend/api/api-admin/KICKOFF_PROMPT.md 的 "自动续接流程" 继续执行 admin 接口计划。
- 或指定下一个模块：
  > 请按 KICKOFF_PROMPT.md 开始做 <next-module> 模块。
```

---

# 给 AI 的硬性约束

1. ❌ **不得跳过步骤 0**（读 3 份文档），否则状态会错乱。
2. ❌ **不得跳过步骤 5**（更新 `SESSION_HANDOFF.md`），否则下次会话无法续接。
3. ❌ **不得并行做多个模块**，一次会话一个模块（或 S0 脚手架）。
4. ❌ **不得改动 `EXECUTION_PLAN.md` 的方法论部分**（§0–§9），除非用户明确要求。
5. ✅ **必须对照 `XX-*.md` 的 JSON 字段**，不得随意改名。
6. ✅ **必须在 Service 层加 `kindergarten_id` 过滤**（多租户兜底）。
7. ✅ **必须 `go build ./...` 通过**才算完成一个接口。
8. ✅ **遇到规格不清晰的地方**，写入工作簿 `Questions` 并在收尾时向用户汇报。

---

# 常见场景

## Q1：新会话第一句应该说什么？
A：复制本文件顶部"最常用"或"指定模块"那一句。

## Q2：上次没做完，怎么接着做？
A：AI 读 `SESSION_HANDOFF.md` 的 `Next Action` 会知道上次停在哪。你只需说"继续"。

## Q3：模块做到一半会话超时了？
A：下次开会话 AI 会读该模块工作簿（`workbooks/XX-*.md`）的"接口完成进度"表，从未打勾的接口继续。

## Q4：突然想插入某个优先级高的模块？
A：直接用"指定模块"那一句，但要求 AI 先把当前模块的未完成接口在工作簿里标记 `Paused`，并把 `SESSION_HANDOFF.md` 的 `Next Action` 指向"回来继续 X 模块"。

## Q5：想并行做 2 个模块（不推荐）？
A：本计划约束串行。若确实需要并行，请为另一个模块开独立 git worktree，并在 `SESSION_HANDOFF.md` 里显式声明"并行模式"。
