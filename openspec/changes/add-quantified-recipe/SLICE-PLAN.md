# add-quantified-recipe · Slice Plan

- Change: `openspec/changes/add-quantified-recipe`
- Base: `8774bf5b4c12a214bd98a233ead8817a7004b051` (`feat/parent-growth-scope`)
- Workflow: slice-delivery（适配：无微信/DevTools；验证器为 `go build/test`、`make wire`、`npm run build`、`check-tokens`、Playwright MCP）
- Commit policy: 每个 slice PASS 后**自动 commit + push**（用户 2026-11-01 授权）；分支 `feat/add-quantified-recipe`；commit message 末尾附 `Co-Authored-By: Claude Code <noreply@anthropic.com>`。
- Human approval: 用户已确认 7 组报表 + 非标准口径 + DRI 年龄加权（2026-11-01）

| Slice | OpenSpec tasks | Goal | Depends | Verifier |
|---|---|---|---|---|
| AQR-S1a | 1.1–1.3 | 契约 + 11 领域模型 + AutoMigrate | — | go build, contract review, dbcheck |
| AQR-S1b | 1.4–1.6, 2.3 | repository + Wire + 路由/权限骨架 | S1a | make wire, go build, router perm |
| AQR-S2 | 2.1, 2.2, 2.4 | 供餐配置后端 | S1b | go test, route |
| AQR-S3 | 3.1, 3.5 | 食物库后端 | S1b | go test, route |
| AQR-S4 | 3.2, 3.5 | 菜谱库后端 | S1b | go test, route |
| AQR-S5 | 3.3, 3.4, 3.6 | 导入命令 + 种子 + 新园 hook | S3, S4 | import sample, idempotent seed |
| AQR-S6 | 4.1–4.5 | 周食谱后端 | S3, S4, S5 | go test, route |
| AQR-S7 | 5.2–5.6 | 营养分析 + 7 组表 + Excel | S6 | golden fixture, go test |
| AQR-S8 | 6.1–6.5, 6.9 | admin api/types/menu + 配置/食物/菜谱页 | S2, S3, S4 | npm build, tokens |
| AQR-S9 | 6.6, 6.7 | admin 周食谱列表 + 网格编辑器 | S6, S8 | npm build, Playwright |
| AQR-S10 | 6.8, 6.10, 7.1–7.5 | 营养分析页 + E2E + 最终验收 | S7, S9 | Playwright, make test, final validators |

## Notes

- **每个 brief 必须要求 worker 运行** `openspec instructions apply --change add-quantified-recipe --json`，阅读 `context` 与 `operationGuidance`，按项目约束与「spec 有误先停/先改」规则执行；Supervisor 同样复核。
- 同一时刻只推进一个 slice；PASS 后才进入下一个。
- **PASS + commit + push 后自动进入下一 slice，不等用户确认**；仅在 FAIL/BLOCKED、需产品决策、推送失败、或上下文预算超限时停下。
- Worker 只改 brief 允许的文件；Supervisor 只读/验证/写 verdict，不改源码、不 commit。
- `.claude/runs/<slice-id>/` 与 `.claude/locks/` 已在 `.gitignore`。
- 3 个后台调研 agent 已完成任务，不再新增。
