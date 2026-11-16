你是并行开发会话。仓库：D:\yuanhaisu\personal_projects\kindergarten。

先读：
- docs/agents/parallel-development.md
- docs/agents/worktree-bot-routine.md
- CLAUDE.md 的「Parallel development」一节

不要在主工作树写代码。开工前执行：
  git fetch origin --prune
  powershell -ExecutionPolicy Bypass -File scripts/worktree-new.ps1 -Name <topic>

然后只进入 ..\kg-<topic> 工作，分支 feat/<topic>，基于 origin/main。
（docs/chore 类可用 -Branch docs/<topic> / -Branch chore/<topic>）

规则：
- 一个会话只做一件事：<topic>
- 只改本切片文件；共享文件（shared-design-tokens/tokens.json、shared-ui/**、
  design.lock.json、CLAUDE.md、.github/workflows/**、server/migrations/**、deploy/**）
  改前先声明，改后跑门禁
- 只用 git add -- <精确路径>，禁止 git add -A / stash / reset / clean
- 提交前：build/test/lint；改页面跑 design:sync + design:accept --reason；
  node scripts/check-secrets.mjs --files ...
- push 到 feat/<topic>，PR 到 main。**不要 merge，不要清理 worktree**
- 合并与清理由 Grok bot 负责：合并成功后 bot 执行
  worktree-clean.ps1 -Pr <PR号> -DeleteRemote
- 收工前确保 git status --porcelain 为空；worktree 交给 bot 回收，不要自己删
- bot 每天早晨出 worktree 审计报告；任何非 merge 触发的清理都会先向属主确认
- 不要直接 push main，不要删别人的 worktree，不要用指向主目录的 node_modules 符号链接
