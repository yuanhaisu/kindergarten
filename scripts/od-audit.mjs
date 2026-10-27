#!/usr/bin/env node
/**
 * Batch-level consistency audit for prototype pages.
 *
 *   node scripts/od-audit.mjs docs/design-previews/2026-10-25-full-redesign [--json]
 *
 * Checks every <surface>/<NN-slug>/index.html for:
 *  - data-surface matches its folder
 *  - hardcoded hex / rgb colors outside tokens.css
 *  - external network references
 *  - white-space: nowrap (can overflow narrow rows)
 *  - zero border-radius
 *  - files over the 20KB guideline
 *  - mini-program chrome markers (status bar, navbar, capsule)
 */
import fs from 'node:fs';
import path from 'node:path';

const batch = process.argv[2];
if (!batch) {
  console.error('usage: node scripts/od-audit.mjs <batch-dir> [--json]');
  process.exit(2);
}
const root = path.resolve(batch);
const jsonOnly = process.argv.includes('--json');

const skipFiles = new Set(['tokens.css']);
const colorRe = /#[0-9a-fA-F]{3,8}\b|\brgba?\(/g;

function pageDirs() {
  const out = [];
  for (const surface of ['teacher', 'parent']) {
    const dir = path.join(root, surface);
    if (!fs.existsSync(dir)) continue;
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      if (!entry.isDirectory()) continue;
      const dirPath = path.join(dir, entry.name);
      if (fs.existsSync(path.join(dirPath, 'index.html'))) {
        out.push({ surface, slug: entry.name, dir: dirPath });
      }
    }
  }
  return out.sort((a, b) => (a.surface + a.slug).localeCompare(b.surface + b.slug));
}

function read(dirPath, name) {
  const file = path.join(dirPath, name);
  return fs.existsSync(file) ? fs.readFileSync(file, 'utf8') : '';
}

function countUnallowed(text, pattern, allowTag) {
  return text
    .split(/\r?\n/)
    .filter((line) => pattern.test(line) && !line.includes(`od-audit-allow: ${allowTag}`))
    .length;
}

const rows = pageDirs().map(({ surface, slug, dir }) => {
  const html = read(dir, 'index.html');
  const css = read(dir, 'styles.css');
  const js = read(dir, 'app.js');
  // <meta name="theme-color"> cannot take a CSS variable; it is not a style leak.
  const htmlForColors = html.replace(/<meta\b[^>]*>/gi, '');
  const code = `${htmlForColors}\n${css}\n${js}`;
  const issues = [];

  const surfaceMatch = html.match(/data-surface="([^"]+)"/);
  if (!surfaceMatch) issues.push('缺少 data-surface');
  else if (surfaceMatch[1] !== surface) issues.push(`data-surface=${surfaceMatch[1]} 与目录 ${surface} 不符`);

  const hexes = [...htmlForColors.matchAll(colorRe), ...css.matchAll(colorRe), ...js.matchAll(colorRe)];
  if (hexes.length) issues.push(`裸色值 ${hexes.length} 处（如 ${hexes[0][0]}）`);

  const external = [...code.matchAll(/url\(\s*['"]?https?:/g), ...code.matchAll(/<script[^>]+src=["']https?:/g), ...code.matchAll(/@import\s+url\(\s*['"]?https?:/g)];
  if (external.length) issues.push(`外链资源 ${external.length} 处`);

  const nowrap = countUnallowed(css, /white-space:\s*nowrap/, 'nowrap');
  if (nowrap) issues.push(`未标注的 nowrap ${nowrap} 处`);

  const zeroRadius = countUnallowed(css, /border-radius:\s*0(px|rpx)?\s*;/, 'radius0');
  if (zeroRadius) issues.push(`未标注的 border-radius:0 ${zeroRadius} 处`);

  const sizes = ['index.html', 'styles.css', 'app.js']
    .map((name) => ({ name, size: Buffer.byteLength(read(dir, name)) }))
    .filter((f) => f.size > 20480);
  const warnings = [];
  if (sizes.length) warnings.push(`超 20KB：${sizes.map((f) => `${f.name} ${Math.round(f.size / 1024)}KB`).join('、')}`);

  const chrome = {
    statusbar: /9:41/.test(html),
    navbar: /mp-navbar|navbar|nav-bar/i.test(html),
    capsule: /mp-capsule|capsule/i.test(html),
  };
  const missingChrome = Object.entries(chrome).filter(([, ok]) => !ok).map(([k]) => k);
  if (missingChrome.length) issues.push(`缺小程序外壳：${missingChrome.join('、')}`);

  // Shell compliance: the shared shell is the only implementation allowed.
  if (!/href="[^"]*_shared\/shell\.css"/.test(html)) issues.push('未链接 _shared/shell.css');
  else if (!/href="\.\.\/\.\.\/_shared\/shell\.css"/.test(html)) issues.push('shell.css 未指向 ../../_shared（疑似本地副本）');
  if (fs.existsSync(path.join(dir, '_shared'))) issues.push('页面目录里有 _shared/ 副本，应链接共享文件');
  if (fs.existsSync(path.join(dir, '.mcp.json'))) issues.push('.mcp.json 泄漏到页面目录（含 API key，必须删除）');
  if (!/class="mp-navbar"/.test(html)) issues.push('未使用 mp-navbar');
  if (!/class="mp-capsule"/.test(html)) issues.push('未使用 mp-capsule');
  const shellRedef = css
    .split(/\r?\n/)
    .filter((line) => /\.mp-[a-z0-9-]+\s*[,{]/.test(line) && !line.includes('[data-state'))
    .flatMap((line) => (line.match(/\.mp-[a-z0-9-]+/g) || []));
  if (shellRedef.length) issues.push(`页面重定义了外壳类：${[...new Set(shellRedef)].slice(0, 3).join('、')}`);

  const notes = fs.existsSync(path.join(dir, 'notes.md'));
  const preview = fs.existsSync(path.join(dir, 'preview.png'));
  if (!notes) issues.push('缺 notes.md');
  if (!preview) issues.push('缺 preview.png');

  // Relative assets must actually exist (e.g. a run that references
  // ../../_shared/tokens.css, which is not a real file, renders unstyled).
  const refs = [
    ...html.matchAll(/(?:href|src)="([^"]+)"/g),
  ]
    .map((m) => m[1])
    .filter((ref) => !/^https?:|^#|^data:/.test(ref));
  const missing = [...new Set(refs)].filter((ref) => {
    const resolved = path.resolve(dir, ref);
    return !fs.existsSync(resolved) && !fs.existsSync(path.join(dir, ref.split('?')[0]));
  });
  if (missing.length) issues.push(`引用不存在的资源：${missing.slice(0, 4).join('、')}`);

  return { surface, slug, issues, warnings, notes, preview, chrome };
});

if (jsonOnly) {
  console.log(JSON.stringify(rows, null, 2));
} else {
  const clean = rows.filter((r) => r.issues.length === 0).length;
  console.log(`\n批次审计：${rows.length} 页，通过 ${clean}，有问题 ${rows.length - clean}\n`);
  for (const row of rows) {
    if (row.issues.length) {
      console.log(`✗ ${row.surface}/${row.slug}`);
      for (const issue of row.issues) console.log(`    - ${issue}`);
    }
    if (row.warnings?.length) {
      console.log(`  ⚠ ${row.surface}/${row.slug}`);
      for (const warning of row.warnings) console.log(`    - ${warning}`);
    }
  }
  if (clean === rows.length) console.log('结构全部通过（体积提示不计失败）。');
}
process.exitCode = rows.some((r) => r.issues.length) ? 1 : 0;
