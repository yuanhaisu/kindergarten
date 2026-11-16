# 并行开发与 worktree 纪律

多个 Claude Code 会话（或人 + 会话）同时改这个仓库时，按本文执行。违反纪律的代价不是冲突本身，而是冲突被发现得太晚。

## 为什么要有这份文档

三次真实事故：

1. **mega PR / 387 项脏工作区**：多个不相关的改动挤在同一个分支上，PR 无法评审，CI 红灯时定位不到是哪一处。收尾时工作区有 387 项未提交改动，既不敢提交也不知道该丢弃哪些。
2. **token 目录在 worktree 清理时被误删**：清理一个 worktree 时连带删掉了共享的 `shared-design-tokens/build/`，三端样式同时失效；因为它是生成产物，一开始没人意识到「重新 build 就能恢复」。
3. **并发改设计预览**：两个会话同时跑 `design:sync` / `design:accept`，后跑的覆盖了先跑的 `design.lock.json`，页面 sha 与源码对不上，`design:check` 失败但说不清是谁改的。

三条教训指向同一件事：**共享的东西必须有明确的串行规则，独占的东西必须物理隔离**。

## 核心原则

> **1 session = 1 branch = 1 worktree。主工作树始终留在 `main` 且保持干净。**

- 每个会话在自己的 worktree 里开发，互不看见对方的未提交改动。
- 主工作树（仓库根目录）只用来：`git fetch`、切 `main`、看日志、跑跨分支的只读命令。不在主工作树写代码。
- 一个分支只服务一件事。发现自己在一个分支上做第二件事，停下来开新分支。

## 标准流程

```powershell
# 1. 同步远端（在主工作树做）
git fetch origin --prune

# 2. 建 worktree + 分支（默认 feat/<name>，基 origin/main）
#    <name> 由执行 agent 按业务自命名（见「Agent 会话规则」）
powershell -ExecutionPolicy Bypass -File scripts/worktree-new.ps1 -Name pickup-polish
#    docs / fix 类切片显式指定分支名：
#   ... -Name parallel-development -Branch docs/parallel-development

# 3. 进入 worktree 开发
cd ..\kg-pickup-polish
npm ci            # 每个 worktree 自己装依赖，不要共享 node_modules
# 开发 / 构建 / 验证……

# 4. 提交（精确 add，见下方纪律）
git add -- path/to/a path/to/b
git commit -m "..."

# 5. 推送并开 PR
$env:https_proxy = "http://127.0.0.1:7890"; $env:http_proxy = "http://127.0.0.1:7890"
git push -u origin feat/pickup-polish
& "C:\Program Files\GitHub CLI\gh.exe" pr create --base main --head feat/pickup-polish --title "..." --body-file <file>

# 5.1 起只读 subagent 监听 CI/合并（见「Agent 会话规则」第 2 条）

# 6. 等 CI 绿；合并由 Grok bot 执行。
#    并行 worker 会话不执行 gh pr merge。

# 7. 合并成功后，Grok bot 立即回收 worktree / 本地分支 / 远端分支：
#    cd ..\kindergarten
#    powershell -ExecutionPolicy Bypass -File scripts/worktree-clean.ps1 -Pr <PR号> -DeleteRemote
#    bot 不可用或清理失败时，才按「worktree 清理纪律」由属主处理。
```

`worktree-clean.ps1` 以 PR 终态为唯一判据：PR 必须 MERGED（CLOSED 未合并需 `-AllowClosed` 且属主确认），worktree 必须干净，本地 HEAD 必须等于 PR 的 `headRefOid`。**不查 `origin/main` 祖先**——本仓存在 squash/rebase 合并，祖先检查会把已合并分支误判为未合并。拒绝是保护，不要用 `git worktree remove --force` 绕过；脚本内部的 `git branch -D` 只在 PR 终态 + `headRefOid` 校验通过后使用。

`worktree-new.ps1` / `worktree-clean.ps1` 都支持 `-WhatIf`，先看它准备做什么再执行；`worktree-audit.ps1` 是只读审计。

## Agent 会话规则（2026-11-15）

以下三条对所有并行开发 agent 生效；与「标准流程」冲突时，以本节为准。

### 1. Topic 由执行 agent 自命名

开工前先读 brief / 文档确定切片边界，agent 自己为切片定一个能描述其意义的 topic，**不要**每次向用户索取。

- 命名：kebab-case，`领域+动作`，如 `deploy-script-safety`、`seed-history`。
- 分支 `fix|feat|docs/<topic>`、worktree `kg-<topic>`、PR 标题三者表达同一含义。
- 命名后直接执行；只有**切片范围本身不清**时才问用户。

### 2. 提交 PR 后起只读 subagent 监听 CI/合并

推送 PR 后**立即**起一个只读 subagent，持续监听结果并反馈；主会话收到终态后再提下一项工作。

- **一个 subagent 只监听一个 PR**。不要把两个 PR 塞进同一个监听任务；多个 PR 并行等待时，每个 PR 各起一个 subagent，提示里写死各自的 PR 号，回报也只针对该 PR。
- 使用 `gh` + 本地代理（PowerShell 用 `gh.exe`，Bash 用 `~/bin/gh`）：

  ```powershell
  $env:https_proxy = "http://127.0.0.1:7890"; $env:http_proxy = "http://127.0.0.1:7890"
  & "C:\Program Files\GitHub CLI\gh.exe" pr checks <n> --watch
  & "C:\Program Files\GitHub CLI\gh.exe" pr view <n> --json state,mergeStateStatus,mergedAt,statusCheckRollup
  ```

- 轮询间隔、重起档位、终态定义以「[CI 与合并等待](#ci-与合并等待)」为唯一口径，本节不另立策略。
- subagent **只报告**，不 merge、不改文件、不推送、不清理。
- subagent 回报需要修改的建议时，按建议再次执行，然后再起一轮监听。

### 3. 并行 worker 只提交 PR，不合并、不清理

并行 worktree 会话（本流程）**只做到 push 分支 + 提交 PR**；合并、worktree/远端分支清理由 **Grok bot** 负责。bot 的例行任务、报告格式与确认协议见 `docs/agents/worktree-bot-routine.md`。

- 不要执行 `gh pr merge`，即使 CI 全绿也不自行合并。
- 不要执行 `worktree-clean.ps1`，不要删远端分支。
- 收工时工作区必须干净（`git status --porcelain` 为空）；脏工作区会让 bot 拒绝清理。
- **一个切片一个 PR**；等当前 PR 合并后再开下一个切片，不并发开新切片。
- 与主会话的 `subagent-driven-slicing` 区分：主会话编排 subagent 时可在 CI 绿后合并；本条只约束并行 worker 会话。

完成切片后按「汇报格式」报告，然后停手。

## 共享文件串行清单

下列路径**同一时刻只允许一个会话修改**。动它们之前，先在会话群里声明；改完立即提交推送，缩短窗口。

| 路径 | 为什么串行 |
| --- | --- |
| `shared-design-tokens/tokens.json`、`shared-design-tokens/build/**` | 三端样式的唯一真相源；两个会话同时 build 会互相覆盖生成产物 |
| `shared-ui/**` | 两端 `components/` 由 `sync-ui.js` 从这里生成；改两端组件会在这里被覆盖 |
| `design.lock.json`、`design.config.json`、`design.preview.lock.json` | `design:sync` / `design:accept` 会整份重写，并发执行必丢条目 |
| `CLAUDE.md` | 规则入口，多人同时改会产生语义冲突而非文本冲突 |
| `docs/agents/parallel-development.md` | 并行纪律的详细规则，改前先在会话群声明 |
| `.github/workflows/**` | CI 行为影响所有分支；改坏了所有人一起红 |
| `server/migrations/**` | goose 版本号必须线性递增；并发新增会同号或乱序 |
| `deploy/**` | 部署脚本与配置，串行评审 |
| 根级 `package.json` / `package-lock.json`、`scripts/**` | 锁文件与共享脚本，并发改动易产生不可复现的 `npm ci` |

规则：

- 在自己的 worktree 里改共享文件，然后**单独开一个 PR**，不要混进功能 PR。
- 改 token 只改 `tokens.json`，然后 `cd shared-design-tokens && npm run build`，产物必须一起提交。**不要手改 `build/` 下的文件。**
- 改完设计相关共享文件后跑 `npm run design:sync`，需要时 `npm run design:accept -- <page> --reason "..."`，并确认 `npm run design:check` 通过。

## 共享运行时与测试资源

代码可以并行，**有状态的运行时不行**。

| 资源 | 规则 |
| --- | --- |
| 开发库迁移 / 种子 | `make migrate`、`make seedbase`、`seed*` 命令**串行**执行。一个会话在跑迁移时，另一个会话不要连库写数据。 |
| 后端 8080 | 同时只跑一个实例。识别队列的 worker 会抢走测试任务，跑队列相关测试前先停掉 8080（见项目记忆 `running-backend-steals-queue-jobs`）。 |
| 微信开发者工具 | 单 IDE 实例时 MCP 验证**串行**；需要双端并行时走 `scripts/dual-ide.ps1`（双实例 + 双微信账号）。端口固定：家长端 CDP `9222` / 自动化 `9420`，老师端 CDP `9223` / 自动化 `9421`；调用 MCP 时显式传端口，别依赖默认值。 |
| 本机端口 | admin dev `5173`、website dev `3000`、server `8080`。并行起多个前端时改端口，不要抢占。 |
| 构建产物 | 两个 uni-app 项目的 `npm run dev:mp-weixin` 是 watch 进程，会把**自己的** `dist/` 写热。不同 worktree 的 dist 互不影响，但同一个 worktree 里不要起两份。 |

## CI 与合并等待

PR 推送后，主会话必须为**每个 PR** 各起一个只读 subagent 监听结果，不能空转或提前开始下一项。**一个 subagent 只监听一个 PR**：提示里必须有且只有一个 PR 号，不允许一个 subagent 同时轮询多个 PR；多个 PR 就起多个 subagent。

1. CI：`gh pr checks <n> --watch`，等所有 job 终态。
2. 合并：`gh pr view <n> --json state,mergeStateStatus,mergedAt,statusCheckRollup`。
3. 若 CI 绿但 PR 仍 OPEN，按阶梯轮询，**直到 MERGED / CLOSED**：
   - 0–5 分钟：15 秒
   - 5–15 分钟：30 秒
   - 15–60 分钟：60 秒
   - 1–4 小时：5 分钟
   - 4–12 小时：15 分钟
   - 12 小时后：只报告“仍未合并”并立即重新起监听，不设短超时
   - **正常走满 12 小时而重起**：新一轮从 15 秒重新走阶梯
   - **单轮因工具/会话硬限制被打断**（subagent 运行上限、单次命令超时等）：续用本轮已持久化的开始时间继续当前档位，不回到 15 秒；只有开始时间丢失才从 15 秒开始
4. subagent 只读：不 merge、不 push、不改文件。
5. 主会话收到终态反馈后，才提出下一项工作。

> 单轮监听不设短超时，但**单轮结束不等于等待结束**：到运行上限就续起下一轮，整体仍等到 `MERGED` / `CLOSED`。起监听时把 PR 号与本轮开始时间（UTC）写进 subagent 提示，避免重起后算错档位。

> PowerShell 不能执行 `~/bin/gh` 的 POSIX wrapper；PowerShell 里用 `& "C:\Program Files\GitHub CLI\gh.exe"`。Bash 里用 `~/bin/gh`。

## 提交与 PR 纪律

**只允许精确暂存**：

```powershell
git add -- docs/agents/parallel-development.md scripts/worktree-new.ps1
```

**禁止**：`git add -A`、`git add .`、`git stash`、`git reset`、`git checkout -- .`、`git clean`。这些命令会把别人的工作或你不理解的改动一起吞掉。发现工作区有不该出现的文件，先查来源，不要清理。

提交前按改动范围跑验证：

| 改动 | 必跑 |
| --- | --- |
| Go 后端 | `make build` / `make test` / `make lint`（至少 build + 相关包测试） |
| admin | `npm run build` |
| teacher / parent | `npm run build:mp-weixin`（或 `dev:mp-weixin` 后看编译错误） |
| 设计 token / 页面 | `npm run design:sync`、必要时 `npm run design:accept`、`npm run design:check` |
| 任意改动 | `node scripts/check-secrets.mjs`（或 `--files <改动文件>`） |

PR 描述必须包含**「不在本 PR」**一节，显式列出被切出去、留给其他 PR 的内容。这是防止 mega PR 复发的唯一有效手段：读 PR 的人能立刻知道边界在哪。

## worktree 清理纪律

> **主路径**：合并成功后由 Grok bot 立即清理。并行 worker 不执行清理。
> **兜底**：bot 每天早晨运行只读审计并出报告；非合并项的清理必须经属主确认。
> 例行任务、报告格式与确认协议见 `docs/agents/worktree-bot-routine.md`。

清理是危险操作，按顺序确认：

1. **先看干净**：`git -C ..\kg-<name> status --porcelain` 必须为空（含未跟踪文件）。
2. **先确认 PR 终态**：PR 必须 MERGED；CLOSED 未合并只能在属主明确放弃后加 `-AllowClosed`。判定用 `gh pr view <n> --json state,headRefOid`，**不要**用 `git merge-base --is-ancestor`——squash/rebase 合并会让已合并分支不是 `origin/main` 的祖先。无 PR 但已确认并入 `origin/main` 的项由审计报告单列，`worktree-clean.ps1` 不处理，删前需属主逐项确认。
3. **HEAD 必须对上**：本地 HEAD 必须等于 PR 的 `headRefOid`；不一致说明可能有没进 PR 的提交，拒绝删除。
4. **执行者**：合并后的即时清理由 bot 执行；每日兜底的批量清理由 bot 在属主确认后执行。属主只处理异常项。
5. **确认没有进程占用**：IDE、`vite` watcher、后端进程如果以该目录为 cwd，先停掉再删。
6. **不要用符号链接共享 `node_modules`**：不同 worktree 的原生依赖与 postinstall 会互相污染，升级依赖时直接炸。每个 worktree 自己 `npm ci`。

误删恢复：worktree 里的**已提交**内容都在分支上（`git worktree remove` 不会删分支）；**未提交**内容无法从 git 恢复。所以第 1 条最重要。

## 事故处理

**共享目录被清空 / 被并发修改时，立即停止当前操作。** 不要盲目 `git restore`、`git checkout -- .`、`git clean`、`npm run build`——你可能正在覆盖另一个会话尚未提交的改动。

按顺序确认：

1. `git status`：被改/被删的是已提交文件还是未跟踪文件？
2. `git worktree list`：有没有别的 worktree 正在改同一批文件？
3. 直接问：在会话群里确认是否有其他会话正在写这个目录。
4. 确认无人占用后再决定：已提交内容用 `git restore -- <精确路径>`；生成产物（`shared-design-tokens/build/**`）用 `npm run build` 重建，**不要手工补**；未跟踪内容丢失则接受损失并记录。

拿不准就停在那里说明情况，不要「先修好再说」。

## 汇报格式

每个切片收工时按此格式汇报，便于主会话汇总：

```
branch=<分支名>
commit=<sha>
files=<文件数> +<新增行>/-<删除行>
验证退出码=<build/test/lint/design:check 等，逐项>
PR=<url>
remaining_dirty=<git status --porcelain 行数>
notes=<异常或需要人工确认的事，无则写 无>
```

报告后停手：等当前 PR 合并后再开下一个切片（见「Agent 会话规则」第 3 条）。
