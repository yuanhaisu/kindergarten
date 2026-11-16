# Worktree 生命周期：Grok bot 例行任务

这份文档是给 Grok bot 的执行说明，不是给并行 worker 的。bot 有本机读写权限和自带定时任务，承担两件事：

1. **主路径**：PR 合并成功后立即回收该切片的 worktree。
2. **兜底**：每天早晨运行只读审计，生成残余报告和清理方案，经属主确认后执行。

bot 不修改代码。审计本身失败时，在 PR 上留结构化改进意见，由当前开发会话修复。

## 环境

| 项 | 值 |
| --- | --- |
| 仓库（主工作树） | `D:\yuanhaisu\personal_projects\kindergarten` |
| Shell | Windows PowerShell 5.1（`powershell -ExecutionPolicy Bypass`） |
| gh | `C:\Program Files\GitHub CLI\gh.exe`（脚本会自动探测） |
| 代理 | `http://127.0.0.1:7890`，通过 `-Proxy` 传入；也可预先设置 `HTTPS_PROXY` |
| 脚本 | `scripts\worktree-audit.ps1`、`scripts\worktree-clean.ps1` |

所有命令都从**主工作树**执行。不要在某个 worktree 内部执行 `-DeleteRemote` 之类的清理命令。

## 任务 A：合并成功后立即清理（主路径）

清理前确认主工作树有最新的脚本版本：

```powershell
git -C D:\yuanhaisu\personal_projects\kindergarten fetch origin main
git -C D:\yuanhaisu\personal_projects\kindergarten merge --ff-only origin/main
```

如果无法快进（主工作树有本地改动或分叉），**不要强推**；改用临时 worktree 跑脚本：

```powershell
git -C D:\yuanhaisu\personal_projects\kindergarten worktree add <临时目录> origin/main
powershell -ExecutionPolicy Bypass -File <临时目录>\scripts\worktree-clean.ps1 -Pr <PR号> -DeleteRemote -Json -Proxy http://127.0.0.1:7890
git -C D:\yuanhaisu\personal_projects\kindergarten worktree remove <临时目录>
```

正常情况直接在主工作树执行：

```powershell
powershell -ExecutionPolicy Bypass -File scripts\worktree-clean.ps1 -Pr <PR号> -DeleteRemote -Json -Proxy http://127.0.0.1:7890
```

判定：

- **退出码 0**：成功，或本来就没有 worktree/本地分支（`already-clean`）。
- **退出码 1**：拒绝或失败。**不要重试、不要 `--force`、不要手工删目录。** 读取 JSON 的 `stage` 与 `error`，记录并报告给属主。
- 只有属主明确说「这个 PR 放弃合并」时，才允许追加 `-AllowClosed`。

清理是合并动作的一部分，已经由「合并」本身授权，不需要再向属主确认。

## 任务 B：合并门禁（PR 验收回路）

在合并任何 PR 之前，**用 PR head 上的脚本**运行审计。在主工作树直接跑只会验证 main 上的旧版本；这个 PR 本身就是要新增脚本，所以必须从 PR head 取代码。

执行位置：

1. 读取 PR 的 `headRefOid` 与 `headRefName`。
2. 找已注册 worktree：分支等于 headRefName、HEAD 等于 headRefOid、工作区干净——直接用它。
3. 找不到就建临时 worktree，跑完删掉：

   ```powershell
   git -C D:\yuanhaisu\personal_projects\kindergarten worktree add <临时目录> <headRefOid>
   powershell -ExecutionPolicy Bypass -File <临时目录>\scripts\worktree-audit.ps1 -Json -Fetch -SkipSize -Proxy http://127.0.0.1:7890
   git -C D:\yuanhaisu\personal_projects\kindergarten worktree remove <临时目录>
   ```

目的是验证脚本本身可执行、输出可解析，而不是要求「零残余」。

判定：

| 退出码 | 含义 | 动作 |
| --- | --- | --- |
| 0 | 审计执行成功，JSON 合法 | 允许合并；残余数量与合并无关 |
| 1 | 环境/执行失败（git、gh、认证、代理） | 不合并，发 PR 评论反馈 |
| 2 | 输出不合法（JSON 解析失败、字段缺失、分类计数不自洽） | 不合并，发 PR 评论反馈 |

**不要把「有残余」当成失败。** 退出码 0 就是通过；`autoClean > 0` 只是报告内容。

### 失败反馈格式（PR 评论）

```text
<!-- worktree-audit:failed -->
stage=<stage>
exit=<1 或 2>
command=<完整可复现命令>
error=<原样贴 stderr 或 JSON.error>
expected=exit 0 + valid JSON
suggestion=<可执行的修改建议；不猜测，只写能验证的>
```

然后等待开发会话推新提交。**同一个失败签名连续 3 轮没有变化**时，停止自动重试，升级给属主。

审计通过后：合并 PR → 执行任务 A。

## 任务 C：每天早晨的残余报告（兜底）

bot 自带定时任务，每天固定时间（建议 08:30 本地时间）执行。日报保留大小统计（只对候选清理项递归计算，通常十几秒）：

```powershell
powershell -ExecutionPolicy Bypass -File scripts\worktree-audit.ps1 -Json -Fetch -Proxy http://127.0.0.1:7890
```

合并门禁只验脚本可执行性，不需要大小，使用 `-SkipSize` 更快。

### 失败时

- 退出码 1 或 2：**不要**把空结果或上一次的结果当作「无残余」。
- 给属主发失败摘要：`stage`、`error`、完整命令。可以顺手开一条 issue 记录，但不要自己改代码。

### 成功时

把 JSON 转成中文报告发给属主，**不删除任何东西**。报告必须包含：

1. 概览：总 worktree、可立即清理（有 PR / 无 PR 分开计数）、需确认、活跃、孤儿目录、无 worktree 的本地分支。
2. **可立即清理（有 PR）**：路径、分支、PR、可释放空间、`worktree-clean.ps1 -Pr <n>` 命令。
3. **可立即清理（无 PR）**：路径、分支、可释放空间、显式 `git worktree remove` / `git branch -D` 命令。这类项 `worktree-clean.ps1` 会拒绝，只能逐项确认后执行。
4. 需确认：路径、分支、PR 状态、原因（CLOSED 未合并 / 脏 / HEAD 不一致 / 无 PR 且过期 / detached / locked）。
5. 孤儿目录：路径、大小、抽样内容；提醒先看内容再删。
6. 无 worktree 的本地分支：分支、PR 状态、建议动作。
7. 结尾给确认口令示例：「清理 A、C」「全部清理可立即清理项」。

### 确认协议

- **用户没有明确说「清理」时，一律不动。** 沉默 = 不清理。
- 「全部清理」的范围**只包括可立即清理且有 PR 的项**（`cleanupMode=pr`）；需确认项、无 PR 项、孤儿目录必须逐项点名。
- 收到确认后：
  1. **重新运行一次审计**（早上的报告是快照，状态可能已变）。
  2. 只对仍然处于 `auto-clean` 且 `cleanupMode=pr` 的项执行 `worktree-clean.ps1 -Pr <n> -DeleteRemote -Json`。
  3. `CLOSED` 未合并项：只有用户明确说「放弃」时才加 `-AllowClosed`。
  4. 无 PR 的 `auto-clean` 项、需确认项和孤儿目录：按用户确认的显式命令逐项执行；执行前再核对一次路径。
  5. 逐项回报结果：成功 / 跳过 / 失败原因。
- 执行前重校验不通过（变脏、HEAD 变了、PR 状态变了）就跳过并说明。

## 报告模板

```text
worktree 残余报告 — 2026-11-16 08:30

概览：总 17 | 可立即清理 13（有 PR 13 / 无 PR 0） | 需确认 1 | 活跃 3 | 孤儿 1 | 残留分支 19

【可立即清理（有 PR，bot 可直接执行）】
1. kg-commercial-closeout（feat/commercial-closeout，PR #33 MERGED，释放 1.2 GB）
   建议：worktree-clean.ps1 -Pr 33 -DeleteRemote
...

【可立即清理（无 PR，需按命令手动执行）】
（本次无）

【需确认】
1. kg-parallel-dev-rules（docs/parallel-dev-rules，PR #36 CLOSED 未合并）
   原因：PR 关闭未合并；如确认放弃，使用 -AllowClosed
...

【孤儿目录】
1. kg-issue-3（4.2 MB；抽样：shared-design-tokens、teacher、产品设计文档）
   建议：先抽查 thoughts.txt 等内容，确认后删除

【无 worktree 的本地分支】
...

回复「清理 1、3」或「全部清理可立即清理项」；不回复则不做任何改动。
「全部清理」只涵盖有 PR 的可立即清理项；其余项请逐项点名。
```

## 硬规则

1. **每日兜底未确认不得删除**；合并后的即时清理是预授权的例外。
2. **执行前必须重校验**：PR 终态、工作区干净、本地 HEAD == `headRefOid`。
3. **绝不使用 `git worktree remove --force`**；删除失败就保留并报告。`worktree-clean.ps1` 内部的 `git branch -D` 是 PR 终态 + `headRefOid` 校验通过后的受控强删，不是绕过检查。
4. **不碰主工作树**（`D:\yuanhaisu\personal_projects\kindergarten`）。
5. **不删别人正在用的 worktree**；脏检查和 HEAD 比对就是为此设置的。
6. **审计失败如实报告**，不得把空 JSON、旧报告或部分结果当作成功。
7. **不修改代码**；发现问题通过 PR 评论反馈。

## 退出码速查

| 脚本 | 退出码 | 含义 |
| --- | --- | --- |
| `worktree-audit.ps1` | 0 | 审计执行成功（有无残余都算成功） |
| | 1 | 环境/执行失败（git、gh、认证、代理、fetch） |
| | 2 | 输出不合法（JSON 解析失败、字段缺失、计数不自洽、数组契约违反） |
| `worktree-clean.ps1` | 0 | 已清理或本来就没有可清理项 |
| | 1 | 拒绝或失败（查看 JSON 的 `stage` / `error`） |
