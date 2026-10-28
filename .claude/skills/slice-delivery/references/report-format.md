# Report Format

## Verdicts

### PASS

All of the following are true:

- Every validator item is checked, or explicitly waived by the user and recorded in `PROGRESS.md`.
- No P1 findings remain.
- Build, token check, secret scan, and console checks pass.
- Normal, empty, and error states have evidence.
- Every claimed interaction was actually performed.
- The changed-file manifest matches `git status` against the recorded baseline.
- The design review's P1 items for this page are resolved or explicitly deferred by the user.

### FAIL

Any of the following:

- A P1 finding.
- A failed validator, build, token check, secret scan, or console check.
- A missing normal/empty/error state.
- A claimed interaction that was not performed.
- A changed file outside the manifest.
- A contract violation: missing JSON, over the size cap, source edit by the supervisor, self-commit by the worker.
- The worker or supervisor guessed a product decision.

### BLOCKED

The environment prevents verification, not the code:

- DevTools will not open, the port is occupied, or the simulator is stuck.
- Backend is down and cannot be restored.
- Required seed data cannot be created or cleaned up.
- Build toolchain is missing.

`BLOCKED` requires two documented attempts and a concrete blocker. It is never `PASS`.

## Review document

Write to `docs/design-previews/2026-10-25-full-redesign/_shared/impl-reviews/<slice-id>.md`. Keep it under 200 lines. Follow the existing `_shared/reviews/` style, plus a DevTools evidence section.

```markdown
# <slice-id> · 对抗性实现审查

- verdict: PASS | FAIL | BLOCKED
- score: NN/100
- 源码覆盖: <files reviewed>
- 审查范围: diff + untracked + build config + affected call chain + routes + shared components
- 基线: <base_sha> → <head_sha>

## DevTools 证据

- 构建: `<cmd>` → exit 0，产物 `<path>`，mtime `<time>`
- Token: `<cmd>` → 0 violations
- Secret scan: `<cmd>` → 0 findings
- 正常态: <data used, page state>
- 空态: <method, observed>
- 错误态: <method, observed>
- 交互: <action> → <observed>
- Console: 0 error
- 路由: <switchTab / navigateTo checks>
- 截图: <attempted / failed reason / fallback>

## P1 · 阻塞问题 (N)

### 1. [category] <issue>
- 证据: `file:line` or command output
- 影响: <what breaks>
- 建议: <fix>

## P2 · 建议 (N)

### 1. [category] <issue>
- 证据: `file:line`
- 建议: <fix or defer reason>
```

## Evidence rules

- Every claim needs a command, a file/line, or an observed UI state.
- Quote command output summaries, not full logs.
- Never print secret values. Report `file:line` and the rule name only.
- Screenshots are best-effort. If the IDE is not foregrounded or capture fails, record the reason and use DOM/CSS assertions plus a manual-device note.
- `wechat_navigate` proves rendering only. Any interaction claim must use `wechat_automator` or `_verify/act.py` and must say which action was performed.
- A page loaded with stale `dist` is not evidence. Record the build time and `dist` mtime.

## Finding severity

| Severity | Meaning | Gate |
| --- | --- | --- |
| P1 | Breaks the slice's acceptance, a user flow, a route, a build, or a security rule | Must fix before PASS |
| P2 | Quality, polish, naming, minor visual difference, non-blocking debt | May defer with a written reason |

## Score

Optional. Use it to communicate overall quality, never to override the verdict. A slice with a P1 cannot score above 79.
