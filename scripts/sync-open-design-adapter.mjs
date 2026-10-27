#!/usr/bin/env node
/**
 * Generate the OpenDesign adapter package from the canonical kindergarten tokens.
 *
 * Source of truth:
 *   shared-design-tokens/tokens.json
 *
 * Generated output (never edit by hand):
 *   <open-design>/.od/design-systems/nurture-planet/{tokens.css,components.html,metadata.json}
 *
 * DESIGN.md and USAGE.md in the adapter folder are hand-curated prose. This
 * script validates that every `var(--token)` they reference exists in the
 * generated stylesheet.
 *
 * Usage:
 *   node scripts/sync-open-design-adapter.mjs
 *   OPEN_DESIGN_ROOT=/path/to/open-design node scripts/sync-open-design-adapter.mjs
 */

import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const openDesignRoot = process.env.OPEN_DESIGN_ROOT
  ? resolve(process.env.OPEN_DESIGN_ROOT)
  : resolve(repoRoot, '..', 'open-design');
const outDir = join(openDesignRoot, '.od', 'design-systems', 'nurture-planet');
const tokensPath = join(repoRoot, 'shared-design-tokens', 'tokens.json');

const raw = JSON.parse(readFileSync(tokensPath, 'utf8'));
const flat = new Map();

(function walk(node, prefix) {
  for (const [key, value] of Object.entries(node)) {
    const path = prefix ? `${prefix}.${key}` : key;
    if (value && typeof value === 'object' && Object.prototype.hasOwnProperty.call(value, 'value')) {
      flat.set(path, value.value);
    } else if (value && typeof value === 'object') {
      walk(value, path);
    }
  }
})(raw, '');

function get(path) {
  if (!flat.has(path)) throw new Error(`Missing token: ${path}`);
  return flat.get(path);
}

function resolveValue(value, seen = new Set()) {
  if (Array.isArray(value)) return value.join(', ');
  if (typeof value !== 'string') return value;
  return value.replace(/\{([^}]+)\}/g, (_, ref) => {
    const key = flat.has(`shared.${ref}`) ? `shared.${ref}` : ref;
    if (seen.has(key)) throw new Error(`Circular token reference: ${key}`);
    seen.add(key);
    const resolved = resolveValue(get(key), seen);
    return String(resolved);
  });
}

function px(path) {
  const value = resolveValue(get(path));
  if (typeof value === 'number') return `${value}px`;
  // Aliases such as {spacing.lg} resolve to a bare numeric string.
  if (/^-?\d+(?:\.\d+)?$/.test(String(value))) return `${value}px`;
  return String(value);
}

function color(path) {
  return String(resolveValue(get(path)));
}

function fontStack() {
  const families = get('shared.typography.fontFamily.base');
  if (!Array.isArray(families)) return String(families);
  return families
    .map((family) => (family.includes(' ') ? `"${family}"` : family))
    .join(', ');
}

const vars = [
  ['--bg', color('parent.color.bg.page')],
  ['--bg-gradient', color('parent.gradient.page')],
  ['--surface', color('parent.color.bg.card')],
  ['--surface-warm', color('teacher.color.bg.card')],
  ['--accent-gradient-soft', color('parent.gradient.brandSoft')],
  ['--fg', color('shared.color.text.primary')],
  ['--fg-2', color('shared.color.text.body')],
  ['--muted', color('shared.color.text.muted')],
  ['--hint', color('shared.color.text.hint')],
  ['--on-brand', color('shared.color.text.onBrand')],
  ['--on-accent', color('shared.color.text.onAccent')],
  ['--border', color('shared.color.border.hairline')],
  ['--accent', color('parent.color.brand.primary')],
  ['--accent-soft', color('parent.color.brand.primaryLight')],
  ['--accent-deep', color('parent.color.brand.primaryDark')],
  ['--accent-wash', color('parent.color.brand.primaryBg')],
  ['--accent-gradient', color('parent.gradient.brand')],
  ['--brand-teacher', color('teacher.color.brand.primary')],
  ['--brand-parent', color('parent.color.brand.primary')],
  ['--brand-admin', color('admin.color.brand.primary')],
  ['--success', color('shared.color.semantic.success')],
  ['--success-wash', color('shared.color.semantic.successBg')],
  ['--warning', color('shared.color.semantic.warning')],
  ['--warning-wash', color('shared.color.semantic.warningBg')],
  ['--danger', color('shared.color.semantic.danger')],
  ['--danger-wash', color('shared.color.semantic.dangerBg')],
  ['--info', color('shared.color.semantic.info')],
  ['--info-wash', color('shared.color.semantic.infoBg')],
  ['--sidebar', color('admin.color.brand.sidebar')],
  ['--sidebar-hover', color('admin.color.brand.sidebarHover')],
  ['--font-sans', fontStack()],
  ['--text-xs', px('shared.typography.fontSize.xs')],
  ['--text-sm', px('shared.typography.fontSize.sm')],
  ['--text-base', px('shared.typography.fontSize.base')],
  ['--text-md', px('shared.typography.fontSize.md')],
  ['--text-lg', px('shared.typography.fontSize.lg')],
  ['--text-xl', px('shared.typography.fontSize.xl')],
  ['--text-2xl', px('shared.typography.fontSize.2xl')],
  ['--text-3xl', px('shared.typography.fontSize.3xl')],
  ['--weight-normal', String(resolveValue(get('shared.typography.fontWeight.normal')))],
  ['--weight-medium', String(resolveValue(get('shared.typography.fontWeight.medium')))],
  ['--weight-semibold', String(resolveValue(get('shared.typography.fontWeight.semibold')))],
  ['--weight-bold', String(resolveValue(get('shared.typography.fontWeight.bold')))],
  ['--leading-tight', '1.4'],
  ['--leading-body', '1.6'],
  ['--radius-sm', px('shared.borderRadius.sm')],
  ['--radius-md', px('shared.borderRadius.md')],
  ['--radius-lg', px('shared.borderRadius.lg')],
  ['--radius-xl', px('shared.borderRadius.xl')],
  ['--radius-2xl', px('shared.borderRadius.2xl')],
  ['--radius-pill', px('shared.borderRadius.full')],
  ['--radius-card', px('shared.component.card.radius')],
  ['--space-xs', px('shared.spacing.xs')],
  ['--space-sm', px('shared.spacing.sm')],
  ['--space-md', px('shared.spacing.md')],
  ['--space-lg', px('shared.spacing.lg')],
  ['--space-xl', px('shared.spacing.xl')],
  ['--shadow-sm', color('shared.shadow.sm')],
  ['--shadow-md', color('shared.shadow.md')],
  ['--shadow-lg', color('shared.shadow.lg')],
  ['--motion-fast', String(resolveValue(get('shared.animation.duration.fast')))],
  ['--motion-base', String(resolveValue(get('shared.animation.duration.base')))],
  ['--motion-slow', String(resolveValue(get('shared.animation.duration.slow')))],
  ['--ease', color('shared.animation.easing.default')],
  ['--page-pad-x', px('shared.component.page.paddingX')],
  ['--page-pad-y', px('shared.component.page.paddingY')],
  ['--avatar-sm', px('shared.component.avatar.sm')],
  ['--avatar-md', px('shared.component.avatar.md')],
  ['--avatar-lg', px('shared.component.avatar.lg')],
  ['--input-height', px('shared.component.input.height')],
  ['--badge-pad-x', px('shared.component.badge.paddingX')],
];

const varNames = new Set(vars.map(([name]) => name));

function block(lines, indent = '  ') {
  return lines.map(([name, value]) => `${indent}${name}: ${value};`).join('\n');
}

function surfaceBlock(surface, overrides) {
  const lines = [];
  for (const [name, value] of Object.entries(overrides)) {
    if (!varNames.has(name)) throw new Error(`Unknown scoped variable: ${name}`);
    lines.push([name, value]);
  }
  return `:root[data-surface="${surface}"] {\n${block(lines)}\n}`;
}

const tokensCss = `/* Generated by kindergarten/scripts/sync-open-design-adapter.mjs.
 * Source of truth: shared-design-tokens/tokens.json
 * Do not edit this file by hand — rerun the sync script instead.
 *
 * Default surface: parent. Override with <html data-surface="teacher|parent|admin">.
 */

:root {
${block(vars)}
}

${surfaceBlock('teacher', {
  '--bg': color('teacher.color.bg.page'),
  '--bg-gradient': color('teacher.gradient.page'),
  '--surface': color('teacher.color.bg.card'),
  '--accent': color('teacher.color.brand.primary'),
  '--accent-soft': color('teacher.color.brand.primaryLight'),
  '--accent-deep': color('teacher.color.brand.primaryDark'),
  '--accent-wash': color('teacher.color.brand.primaryBg'),
  '--accent-gradient': color('teacher.gradient.brand'),
  '--accent-gradient-soft': color('teacher.gradient.brandSoft'),
})}

${surfaceBlock('parent', {
  '--bg': color('parent.color.bg.page'),
  '--bg-gradient': color('parent.gradient.page'),
  '--surface': color('parent.color.bg.card'),
  '--accent': color('parent.color.brand.primary'),
  '--accent-soft': color('parent.color.brand.primaryLight'),
  '--accent-deep': color('parent.color.brand.primaryDark'),
  '--accent-wash': color('parent.color.brand.primaryBg'),
  '--accent-gradient': color('parent.gradient.brand'),
  '--accent-gradient-soft': color('parent.gradient.brandSoft'),
})}

${surfaceBlock('admin', {
  '--bg': color('admin.color.bg.page'),
  '--bg-gradient': color('admin.gradient.page'),
  '--surface': color('admin.color.bg.card'),
  '--accent': color('admin.color.brand.primary'),
  '--accent-soft': color('admin.color.brand.primaryLight'),
  '--accent-deep': color('admin.color.brand.primaryDark'),
  '--accent-wash': color('admin.color.brand.primaryBg'),
  '--accent-gradient': color('admin.gradient.brand'),
  '--accent-gradient-soft': color('admin.gradient.brandSoft'),
  '--sidebar': color('admin.color.brand.sidebar'),
  '--sidebar-hover': color('admin.color.brand.sidebarHover'),
})}
`;

const componentsHtml = `<!doctype html>
<html lang="zh-CN" data-surface="parent">
<!-- Generated reference fixture. Source: kindergarten/shared-design-tokens/tokens.json -->
<head>
<meta charset="utf-8" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<title>Nurture Planet component reference</title>
<style>
${tokensCss}
* { box-sizing: border-box; }
body {
  margin: 0;
  min-height: 100vh;
  padding: var(--page-pad-y) var(--page-pad-x);
  background: var(--bg-gradient);
  color: var(--fg-2);
  font-family: var(--font-sans);
  font-size: var(--text-base);
  line-height: var(--leading-body);
}
.stack { display: grid; gap: var(--space-lg); max-width: 420px; margin: 0 auto; }
.card {
  background: var(--surface);
  border-radius: var(--radius-card);
  padding: var(--space-xl);
  box-shadow: var(--shadow-md);
}
.child-card {
  background: var(--accent-gradient);
  color: var(--on-brand);
  border-radius: var(--radius-card);
  padding: var(--space-xl);
  box-shadow: var(--shadow-md);
}
.title { font-size: var(--text-xl); font-weight: var(--weight-semibold); color: var(--fg); }
.label { font-size: var(--text-xs); font-weight: var(--weight-medium); color: var(--muted); line-height: var(--leading-tight); }
.timeline { display: grid; gap: var(--space-lg); }
.timeline-item { display: grid; grid-template-columns: var(--avatar-md) 1fr; gap: var(--space-lg); }
.avatar {
  width: var(--avatar-md); height: var(--avatar-md);
  border-radius: var(--radius-pill);
  background: var(--accent-wash);
  display: grid; place-items: center;
  color: var(--accent-deep); font-weight: var(--weight-semibold);
}
.badge {
  display: inline-flex; align-items: center;
  height: 24px; padding: 0 var(--badge-pad-x);
  border-radius: var(--radius-pill);
  background: var(--accent-wash); color: var(--accent-deep);
  font-size: var(--text-xs); font-weight: var(--weight-medium);
}
.btn {
  height: var(--input-height); padding: 0 var(--space-xl);
  border: 0; border-radius: var(--radius-pill);
  background: var(--accent-gradient); color: var(--on-brand);
  font-family: inherit; font-size: var(--text-base); font-weight: var(--weight-semibold);
  box-shadow: var(--shadow-md);
}
.btn-ghost {
  background: var(--accent-wash); color: var(--accent-deep); box-shadow: var(--shadow-sm);
}
.input {
  width: 100%; height: var(--input-height); padding: 0 var(--space-lg);
  border: 1px solid var(--border); border-radius: var(--radius-xl);
  background: var(--surface); color: var(--fg); font-family: inherit; font-size: var(--text-base);
}
.row { display: flex; gap: var(--space-lg); align-items: center; }
</style>
</head>
<body>
  <main class="stack">
    <section class="child-card">
      <div class="label" style="color: var(--on-brand); opacity: .8;">小一班</div>
      <div class="title" style="color: var(--on-brand);">今天的成长时光</div>
      <div class="label" style="color: var(--on-brand); opacity: .8;">4 条新记录</div>
    </section>

    <section class="card">
      <div class="row" style="justify-content: space-between; margin-bottom: var(--space-lg);">
        <span class="title">成长记录</span>
        <span class="badge">已同步</span>
      </div>
      <div class="timeline">
        <article class="timeline-item">
          <div class="avatar">朵</div>
          <div>
            <div class="title" style="font-size: var(--text-md);">第一次独立完成拼图</div>
            <div class="label">3月19日 11:30 · 益智活动</div>
          </div>
        </article>
        <article class="timeline-item">
          <div class="avatar">朵</div>
          <div>
            <div class="title" style="font-size: var(--text-md);">午睡时长 1 小时 20 分</div>
            <div class="label">3月19日 14:10 · 生活照护</div>
          </div>
        </article>
      </div>
    </section>

    <section class="card">
      <div class="label" style="margin-bottom: var(--space-md);">快速记录</div>
      <input class="input" placeholder="今天发生了什么？" />
      <div class="row" style="margin-top: var(--space-lg);">
        <button class="btn">发布记录</button>
        <button class="btn btn-ghost">存草稿</button>
      </div>
    </section>
  </main>
</body>
</html>
`;

const metadataJson = `${JSON.stringify({
  title: '养成星球 Nurture Planet',
  category: 'Education',
  surface: 'web',
  status: 'published',
}, null, 2)}\n`;

mkdirSync(outDir, { recursive: true });
writeFileSync(join(outDir, 'tokens.css'), tokensCss, 'utf8');
writeFileSync(join(outDir, 'components.html'), componentsHtml, 'utf8');
writeFileSync(join(outDir, 'metadata.json'), metadataJson, 'utf8');

let validationErrors = 0;
for (const file of ['DESIGN.md', 'USAGE.md']) {
  const path = join(outDir, file);
  if (!existsSync(path)) continue;
  const text = readFileSync(path, 'utf8');
  for (const match of text.matchAll(/var\(\s*(--[a-z0-9-]+)\s*\)/gi)) {
    if (!varNames.has(match[1])) {
      validationErrors += 1;
      console.error(`[adapter] ${file}: unknown variable ${match[1]}`);
    }
  }
}

if (validationErrors > 0) process.exitCode = 1;

console.log(`[adapter] wrote ${outDir}`);
console.log(`[adapter] tokens: ${vars.length} variables`);
console.log(`[adapter] validation: ${validationErrors === 0 ? 'ok' : `${validationErrors} error(s)`}`);
