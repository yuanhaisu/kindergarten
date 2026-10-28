# Validator Templates

The project adapter (`_shared/VALIDATORS.md`) instantiates these templates with concrete commands and paths. The worker fills a copy; the supervisor independently re-runs it.

## Gate rule

An unchecked item is a `FAIL`, unless the user has explicitly waived it and the waiver is recorded in `PROGRESS.md`. A waiver must include the reason and the follow-up slice ID.

## Template A — UI page

- [ ] Route exists in `pages.json`; no page added without an approved spec.
- [ ] Target end identified (teacher or parent); correct ports/project path used.
- [ ] Dev build rebuilt after the last source edit; `dist` mtime is newer than the source.
- [ ] `npm run dev:mp-weixin` exits 0.
- [ ] `node scripts/check-tokens.mjs <end>` reports 0 new violations.
- [ ] No inline SVG in WXML; icons are assetized.
- [ ] No unsupported CSS (`color-mix`, `dvh`, `aspect-ratio`, `sticky`, `backdrop-filter`, `scroll-snap`) unless replaced by the adapter's fallback.
- [ ] Normal state renders with real data.
- [ ] Empty state renders with the specified copy.
- [ ] Error state renders with retry.
- [ ] Every claimed interaction was performed with `wechat_automator` / `_verify/act.py`.
- [ ] Console has 0 errors.
- [ ] Tab navigation uses `switchTab`; non-tab uses `navigateTo` / `redirectTo`.
- [ ] Design diff list (font, spacing, color, radius, icon) is recorded; P1 diffs fixed.
- [ ] The page's P1 design-review findings are resolved or user-deferred.

## Template B — Backend

- [ ] `go build ./...` exits 0.
- [ ] Targeted `go test` for the touched package exits 0 (or a documented reason why no test exists).
- [ ] `backend/api/` contract is updated first; handler/service conform.
- [ ] Router and Casbin policy updated if a route or role changed.
- [ ] `AutoMigrate` ran or the migration SQL + `cmd/dbcheck` confirms the schema.
- [ ] API response shape verified with a real request.
- [ ] Affected mini-program page still renders and the new field is visible.
- [ ] OpenSpec change is proposed, approved, applied, and (if complete) archived.
- [ ] No secrets in the diff; `check-secrets.mjs` passes.

## Template C — Boundary / asset

- [ ] Icons are files or iconfont; no inline SVG.
- [ ] The page has a defined entry or is explicitly hidden; no dead link.
- [ ] Empty/error state is meaningful, especially where the platform cannot render the real feature.
- [ ] `web-view` constraints are documented (personal-subject teacher app cannot use it; parent app needs a business domain).
- [ ] Build and console checks pass.
- [ ] The user has approved the entry/hide decision.

## Template D — F0 baseline

- [ ] `AppIcon` covers the agreed icon set.
- [ ] `PageShell` / safe-area / bottom-nav conventions are documented and used by the pilot pages.
- [ ] `StateBlock` covers loading, empty, error, retry.
- [ ] CSS compatibility guide covers the known unsupported features.
- [ ] `.gitignore` includes `.claude/runs/` and `.claude/locks/`.
- [ ] `scripts/check-secrets.mjs` exists, has a self-test, and passes on a clean tree.
- [ ] The five pilot pages (teacher workbench, teacher notice create, teacher notice list, parent growth report, parent pickup QR) still build and render without regression.
- [ ] Both ends build; token check has no new violations.

## Worker responsibilities

1. Copy the relevant template into `.claude/runs/<slice-id>/validators.md`.
2. Check each item with evidence.
3. Leave unchecked items visibly unchecked and explain why.

## Supervisor responsibilities

1. Do not trust the worker's checked boxes.
2. Re-run every command and interaction.
3. Mark any unverified item as `FAIL`.
4. Compare the worker's checked list with the actual evidence; a false claim is a P1 finding.
