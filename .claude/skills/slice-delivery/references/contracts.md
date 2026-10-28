# Slice Contracts

All paths below are relative to the repo root unless stated otherwise. Runtime artifacts live under `.claude/runs/<slice-id>/` and are gitignored.

## 1. `brief.md`

The main session writes this before spawning the worker. It is the single source of truth for the slice. Keep it under 150 lines.

```markdown
# <slice-id> · <slice name>

## Goal
One paragraph. What user-visible or system-visible outcome this slice delivers.

## Inputs (read these first)
- Design: docs/design-previews/2026-10-25-full-redesign/<end>/<page>/
- Existing source: <paths>
- Design review: docs/design-previews/2026-10-25-full-redesign/_shared/reviews/<end>-<page>.md
- Project adapter: docs/design-previews/2026-10-25-full-redesign/_shared/IMPLEMENTATION-WORKFLOW.md
- Validators: docs/design-previews/2026-10-25-full-redesign/_shared/VALIDATORS.md

## Scope
- Allowed files: <explicit paths/globs>
- Allowed shared files: <only if this slice owns them>
- Forbidden: unrelated pages, `pages.json` unless listed, tokens unless listed, `.claude/` runtime, `configs/app.local.ini`, `.env`, `.mcp.json`

## Acceptance
Copy the relevant validator items. Every item is a gate.

## Required evidence
- Build command + exit code
- Token check command + violations
- Route/`switchTab` check
- Empty/error state method
- Interaction list
- P1/P2 review items addressed or deferred

## Output
- `.claude/runs/<slice-id>/worker-result.json`
- `.claude/runs/<slice-id>/self-review.md`
- Final response ≤250 words

## Human decisions
- None, or the exact question the worker must not guess.
```

## 2. `worker-result.json`

```json
{
  "slice_id": "P01",
  "status": "DONE",
  "base_sha": "abc1234",
  "files_changed": ["parent/src/pages/index/index.vue"],
  "files_added": [],
  "files_deleted": [],
  "commands": [
    {"cmd": "cd parent && npm run dev:mp-weixin", "exit": 0, "summary": "DONE Build complete"},
    {"cmd": "node scripts/check-tokens.mjs parent", "exit": 0, "summary": "0 violations"}
  ],
  "self_review": {
    "findings": 0,
    "resolved": 0,
    "notes": "Checked against design, review P1/P2, switchTab rules."
  },
  "known_gaps": [],
  "needs_decision": null,
  "artifacts": {
    "result": ".claude/runs/P01/worker-result.json",
    "self_review": ".claude/runs/P01/self-review.md"
  },
  "next_action": "supervisor"
}
```

`status` values:

- `DONE` — implemented, self-tested, self-reviewed, ready for supervisor.
- `FAILED` — could not complete; include evidence and `next_action`.
- `BLOCKED` — environment or missing input; include the exact blocker.

`needs_decision` is either `null` or `{ "question": "...", "options": [...], "recommendation": "..." }`. A worker must never guess a product decision.

## 3. `verdict.json`

```json
{
  "slice_id": "P01",
  "verdict": "PASS",
  "score": 92,
  "base_sha": "abc1234",
  "scope_check": {
    "manifest_mismatch": false,
    "unrelated_changes": []
  },
  "evidence": {
    "build": {"cmd": "cd parent && npm run dev:mp-weixin", "exit": 0, "dist_mtime": "2026-10-28T..."},
    "tokens": {"cmd": "node scripts/check-tokens.mjs parent", "violations": 0},
    "secrets": {"cmd": "node scripts/check-secrets.mjs --files ...", "findings": 0},
    "normal": {"ok": true, "notes": "real data: ..."},
    "empty": {"ok": true, "notes": "..."},
    "error": {"ok": true, "method": "failure-build", "notes": "..."},
    "interactions": [
      {"action": "tap 入口 -> 页面", "ok": true},
      {"action": "返回 -> 列表刷新", "ok": true}
    ],
    "console_errors": 0,
    "routes": {"switch_tab_ok": true, "notes": "..."},
    "screenshots": {"attempted": true, "ok": false, "reason": "IDE background; DOM/CSS assertions used"}
  },
  "findings": [
    {
      "severity": "P2",
      "file": "parent/src/pages/index/index.vue",
      "line": 42,
      "issue": "Hardcoded color in a non-token gradient",
      "evidence": "line 42: background: #fff",
      "suggestion": "Use $color-bg-card or add a token"
    }
  ],
  "review_path": "docs/design-previews/2026-10-25-full-redesign/_shared/impl-reviews/P01.md",
  "next_action": "commit"
}
```

`verdict` values:

- `PASS` — every validator item is checked or explicitly waived by the user; no P1 findings; evidence complete.
- `FAIL` — any P1, failed validator, unrelated change, missing evidence, or build/tokens/console failure.
- `BLOCKED` — environment or missing data; include `blocker` and the two retry attempts.

`score` is optional but recommended (0–100). It must not override the verdict.

## 4. `checkpoint.json`

Written by the main session at four phases.

```json
{
  "slice_id": "P01",
  "phase": "supervisor_done",
  "base_sha": "abc1234",
  "head_sha": "def5678",
  "commit_sha": null,
  "retries": {"worker": 0, "supervisor": 0},
  "artifacts": {
    "brief": ".claude/runs/P01/brief.md",
    "worker_result": ".claude/runs/P01/worker-result.json",
    "verdict": ".claude/runs/P01/verdict.json",
    "review": "docs/design-previews/2026-10-25-full-redesign/_shared/impl-reviews/P01.md"
  },
  "next_action": "commit",
  "updated_at": "2026-10-28T04:00:00+08:00"
}
```

`phase` values: `briefed`, `worker_done`, `supervisor_done`, `committed`, `blocked`.

On `blocked`, add:

```json
{
  "blocker": "...",
  "attempts": ["...", "..."],
  "escalation": "user"
}
```

## 5. `PROGRESS.md` row

```markdown
| P01 | W1 | 家长首页 | passed | abc1234 | def5678 | impl-reviews/P01.md | - |
```

Status values: `pending`, `briefed`, `working`, `review`, `fix`, `passed`, `blocked`, `waived`.

## 6. Size caps

| Artifact | Cap |
| --- | --- |
| `brief.md` | 150 lines |
| `worker-result.json` | 80 lines |
| `verdict.json` | 120 lines |
| `review.md` | 200 lines |
| Worker final response | 250 words |
| Supervisor final response | 300 words |

If a cap is exceeded, the agent must write the full content to disk and return a pointer, not paste it.
