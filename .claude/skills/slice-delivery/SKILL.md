---
name: slice-delivery
description: Execute a multi-slice implementation plan with a thin orchestrator, one worker subagent per slice, and a fresh adversarial supervisor that verifies in WeChat DevTools. Enforces checkpoints, validators, secret scanning, and context-safe handoffs.
---

# Slice Delivery

Use this skill when a plan has been sliced into independently verifiable units and each unit must be implemented, adversarially reviewed, and verified before the next one starts.

Do not use it for one-off fixes, exploratory work, or work without a slice definition.

## Hard rules

1. **The main session is a thin orchestrator.** It never reads source files, full diffs, full logs, or full review reports. Allowed reads: `PROGRESS.md`, `worker-result.json`, `verdict.json`, `git diff --stat`, `git diff --cached --name-only`, and the slice row in the plan.
2. **Worker implements; supervisor judges.** The worker implements, self-tests, and self-reviews. The supervisor is always a **fresh** instance; it adversarially reviews the worker's full output and verifies in WeChat DevTools. The supervisor never edits source and never commits.
3. **Every slice ends PASS, FAIL, or BLOCKED.** No PASS without evidence. `BLOCKED` is not `PASS`.
4. **Every slice has four checkpoints:** briefed, worker done, supervisor done, committed. State lives in `PROGRESS.md` plus `.claude/runs/<slice-id>/checkpoint.json`.
5. **Human checkpoints are mandatory** for product decisions, OpenSpec approvals, design ambiguity, and the third consecutive FAIL.
6. **No ruflo runtime.** Do not call, depend on, or restore claude-flow/ruflo hooks, MCP tools, memory, or agent definitions. The checkpoint, validator, and secret-scan patterns are reimplemented in project Markdown and scripts.

## Inputs

| Input | Path |
| --- | --- |
| Plan | `docs/design-previews/2026-10-25-full-redesign/_shared/IMPLEMENTATION-PLAN.md` |
| Progress board | `PROGRESS.md` |
| Project adapter | `docs/design-previews/2026-10-25-full-redesign/_shared/IMPLEMENTATION-WORKFLOW.md` |
| Validators | `docs/design-previews/2026-10-25-full-redesign/_shared/VALIDATORS.md` |
| Design reviews | `docs/design-previews/2026-10-25-full-redesign/_shared/reviews/` |
| Contracts | `references/contracts.md` |
| Report format | `references/report-format.md` |
| Validator templates | `references/validators.md` |

## Roles

| Role | Subagent | May do | Must not do |
| --- | --- | --- | --- |
| Orchestrator | main session | Write brief, spawn agents, read verdict, commit, checkpoint | Read source/diff/logs; touch DevTools; decide product scope |
| Worker | `slice-worker` | Read, edit, build, test, self-review | Use WeChat MCP; commit; edit outside manifest |
| Supervisor | `slice-supervisor` | Read, build, run tests, Drive DevTools, write review/verdict | Edit source; commit; reuse another supervisor's context |

## Per-slice procedure

### 0. Preflight (main session)

1. Read the slice row in the plan and its row in `PROGRESS.md`.
2. If the slice needs a human decision (`X00`, OpenSpec approval, design ambiguity), stop and ask the user. Record the answer in `PROGRESS.md`.
3. Record the current `base_sha` (`git rev-parse HEAD`).
4. Write `.claude/runs/<slice-id>/brief.md` from `references/contracts.md`.
5. Write `checkpoint.json` with `phase: "briefed"`.
6. Update the task to `in_progress`.

### 1. Spawn the worker

Call `Agent` with `subagent_type: "slice-worker"`. The prompt is a short pointer, not a copy of the brief:

```
slice: <slice-id>
brief: .claude/runs/<slice-id>/brief.md
base_sha: <sha>
```

Read only its `worker-result.json` and its final summary (≤250 words). Do not read its full logs.

### 2. Spawn the supervisor

Only after the worker returns `DONE` or `FAILED` with artifacts. Call `Agent` with `subagent_type: "slice-supervisor"`:

```
slice: <slice-id>
brief: .claude/runs/<slice-id>/brief.md
worker_result: .claude/runs/<slice-id>/worker-result.json
base_sha: <sha>
```

The supervisor must acquire the DevTools lock, rebuild, review all outputs, verify in DevTools, and write `verdict.json`.

Read only `verdict.json` and its final summary (≤300 words).

### 3. Decide

| Verdict | Action |
| --- | --- |
| `PASS` | Go to commit. |
| `FAIL` | Spawn a **new** worker with the supervisor's findings file; then a **new** supervisor. Max 2 fix rounds. The 3rd FAIL escalates to the user. |
| `BLOCKED` | Let the supervisor retry twice with evidence. Then do one environment repair (restart backend, reopen IDE). If still blocked, escalate to the user. |

Never let the same worker or supervisor handle the fix/re-review. Never mark `BLOCKED` as `PASS`.

### 4. Commit (main session only)

1. Read `worker-result.json` for the manifest.
2. Stage exactly the manifest paths:
   ```bash
   git add -- <path1> <path2> ...
   ```
3. Run the secret scan on the staged diff:
   ```bash
   node scripts/check-secrets.mjs --staged
   ```
4. Compare:
   ```bash
   git diff --cached --name-only
   git diff --cached --stat
   ```
   The names must equal the manifest. Any extra path is a FAIL.
5. Commit one slice per commit. Do not use `git add -A`.
6. Write `checkpoint.json` with `phase: "committed"` and `commit_sha`.
7. Update `PROGRESS.md` and the task to `completed`.

Commit message template:

```
<type>(<scope>): <slice-id> <short description>

- <what changed>
- Verified: <build/tokens/DevTools evidence pointer>
```

### 5. Checkpoint

Write `checkpoint.json` at four points: `briefed`, `worker_done`, `supervisor_done`, `committed`. On `FAIL`/`BLOCKED`, write the same file with `phase: "blocked"` and `next_action`.

`PROGRESS.md` is the human-readable index; `checkpoint.json` is the machine-readable resume state.

## Context budget

- Target ≤2k new tokens per slice in the main session.
- Never run `cat`, `git diff`, or `git show` on source files. Use `git diff --stat` and the agent summaries.
- Worker final response ≤250 words; supervisor final response ≤300 words. Full reports go to disk.
- At each wave boundary, check `/context`. If usage is >40% or auto-compact has occurred, write a handoff block in `PROGRESS.md` and stop; the user starts a fresh session that reads `PROGRESS.md` + the current `checkpoint.json`.

## Resume protocol

1. Read `PROGRESS.md` and the current slice's `checkpoint.json`.
2. Verify `git rev-parse HEAD` matches the checkpoint's `head_sha` (or the `base_sha` plus the recorded commit).
3. Continue from `next_action`. Do not re-run a completed phase.

## Human checkpoints

Stop and ask the user when:

- A slice needs a product-scope decision (`X00`).
- A backend slice needs an OpenSpec proposal approved (`B01`–`B03`).
- A worker or supervisor reports `needs_decision` / design ambiguity.
- A slice reaches the third consecutive `FAIL`.

The main session owns the question, the recommendation, and the record of the answer. Subagents must not decide product scope or approve their own specs.

## Failure policy

| State | Meaning | Owner |
| --- | --- | --- |
| `PASS` | Validators pass, evidence complete, no P1 findings | Supervisor |
| `FAIL` | Code/design issue: P1 finding, failed validator, unrelated changes, build/tokens/console failure | Worker fixes; new supervisor re-reviews |
| `BLOCKED` | Environment issue: DevTools, backend, build tool, missing data | Supervisor retries twice; main session repairs once; then user |

## References

- `references/contracts.md` — brief, worker-result, verdict, checkpoint schemas.
- `references/report-format.md` — PASS/FAIL/BLOCKED and review document format.
- `references/validators.md` — the four validator templates.
- `docs/design-previews/2026-10-25-full-redesign/_shared/IMPLEMENTATION-WORKFLOW.md` — project adapter: build commands, DevTools runbook, ports, error-state method, test data.
