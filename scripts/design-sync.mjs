#!/usr/bin/env node
/**
 * Keep the design-preview batch in sync with the mini-program sources.
 *
 * The code is the single source of truth for UI. The HTML prototypes under
 * docs/design-previews/2026-10-25-full-redesign are a visual reference, so the
 * parts that can be derived are generated here instead of hand-maintained:
 *
 *   - tokens.css           <- shared-design-tokens/tokens.json
 *   - _shared/pages.json   <- teacher|parent/src/pages.json + design.config.json
 *   - index.html / README  <- od-gallery.mjs
 *   - design.lock.json     <- sha256 of every page's .vue + title / tabBar
 *
 * Commands:
 *   node scripts/design-sync.mjs sync                 regenerate tokens + manifest + gallery; add unreviewed lock entries
 *   node scripts/design-sync.mjs check                fail when tokens / manifest / lock / gallery drift
 *   node scripts/design-sync.mjs check --quick        warn only, for the pre-commit hook
 *   node scripts/design-sync.mjs accept <slug>        mark one page reviewed
 *   node scripts/design-sync.mjs accept --shared      acknowledge shared-ui changes only
 *   node scripts/design-sync.mjs accept --all         bulk-accept, only when no page content changed
 *
 * Exit codes: 0 ok / 1 drift or usage error / 2 infrastructure error.
 */
import fs from 'node:fs';
import path from 'node:path';
import os from 'node:os';
import crypto from 'node:crypto';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { renderThemesCss, writeThemesCss } from './build-theme-css.mjs';

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const BATCH = path.join(repoRoot, 'docs/design-previews/2026-10-25-full-redesign');
const CONFIG = path.join(BATCH, 'design.config.json');
const MANIFEST = path.join(BATCH, '_shared/pages.json');
const LOCK = path.join(BATCH, 'design.lock.json');
const GALLERY = path.join(BATCH, 'index.html');
const README = path.join(BATCH, 'README.md');
const SURFACES = ['teacher', 'parent'];

const argv = process.argv.slice(2);
const command = argv[0] || 'check';
const flags = new Set(argv.filter((a) => a.startsWith('--')));
const positional = argv.slice(1).filter((a) => !a.startsWith('--'));

function localDate() {
  const date = new Date();
  const pad = (value) => String(value).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
}

const TODAY = localDate();

function fail(message, code = 1) {
  console.error(message);
  process.exit(code);
}

function readJson(file) {
  try {
    return JSON.parse(fs.readFileSync(file, 'utf8'));
  } catch (error) {
    fail(`无法读取 ${path.relative(repoRoot, file)}: ${error.message}`, 2);
  }
}

function normalize(text) {
  return String(text).replace(/\r\n/g, '\n');
}

function sha256Buffer(buffer) {
  return crypto.createHash('sha256').update(buffer).digest('hex');
}

function sha256File(file) {
  if (!fs.existsSync(file)) return null;
  return sha256Buffer(normalize(fs.readFileSync(file, 'utf8')));
}

function rel(file) {
  return path.relative(repoRoot, file).split(path.sep).join('/');
}

function codePages(surface) {
  const file = path.join(repoRoot, surface, 'src/pages.json');
  const json = readJson(file);
  const pages = new Map();
  for (const node of json.pages || []) {
    pages.set(node.path, node.style?.navigationBarTitleText || '');
  }
  for (const group of json.subPackages || json.subpackages || []) {
    const root = String(group.root || '').replace(/\/$/, '');
    for (const node of group.pages || []) {
      const route = root ? `${root}/${node.path}` : node.path;
      pages.set(route, node.style?.navigationBarTitleText || '');
    }
  }
  return pages;
}

function codeTabbarRoutes(surface) {
  if (surface === 'parent') {
    const json = readJson(path.join(repoRoot, 'parent/src/pages.json'));
    return new Set((json.tabBar?.list || []).map((item) => item.pagePath));
  }
  // 老师端 pages.json 没有 tabBar，4 个常驻页由 BottomNav 决定。
  const file = path.join(repoRoot, 'teacher/src/components/BottomNav.vue');
  if (!fs.existsSync(file)) return new Set();
  const text = fs.readFileSync(file, 'utf8');
  return new Set([...text.matchAll(/path:\s*["']\/?(pages\/[^"']+)["']/g)].map((match) => match[1]));
}

function buildManifest() {
  const config = readJson(CONFIG);
  const entries = [];
  const errors = [];

  for (const surface of SURFACES) {
    const pages = codePages(surface);
    const meta = config[surface] || {};
    const tabbarRoutes = codeTabbarRoutes(surface);
    const seenSlugs = new Set();
    const seenNumbers = new Set();

    for (const [route, item] of Object.entries(meta)) {
      if (seenSlugs.has(item.slug)) errors.push(`slug 重复: ${surface}/${item.slug}`);
      if (seenNumbers.has(item.n)) errors.push(`编号重复: ${surface}/${item.n}`);
      if (Boolean(item.tabbar) !== tabbarRoutes.has(route)) {
        errors.push(`design.config.json 的 tabbar 与代码不一致: ${surface}/${route}` +
          `（代码 ${tabbarRoutes.has(route)}）`);
      }
      seenSlugs.add(item.slug);
      seenNumbers.add(item.n);
    }

    for (const [route, item] of Object.entries(meta)) {
      if (!pages.has(route)) {
        errors.push(`design.config.json 登记了不存在的页面: ${surface}/${route}`);
        continue;
      }
      const source = `${surface}/src/${route}.vue`;
      if (!fs.existsSync(path.join(repoRoot, source))) {
        errors.push(`pages.json 登记了 ${surface}/${route}，但源码 ${source} 不存在`);
        continue;
      }
      entries.push({
        n: item.n,
        surface,
        slug: item.slug,
        path: source,
        title: pages.get(route),
        tabbar: tabbarRoutes.has(route),
      });
    }

    for (const route of pages.keys()) {
      if (!meta[route]) {
        errors.push(`代码新增页面未登记到 design.config.json: ${surface}/${route}`);
      }
    }
  }

  if (errors.length) {
    fail(`[design-sync] 页面清单无法生成:\n- ${errors.join('\n- ')}\n` +
      '  新页面请先在 design.config.json 里补 slug / n / tabbar，并补对应原型目录。');
  }

  const order = new Map(SURFACES.map((surface, index) => [surface, index]));
  entries.sort((a, b) => order.get(a.surface) - order.get(b.surface) || a.n - b.n || a.slug.localeCompare(b.slug));
  return entries;
}

function manifestText(entries) {
  const groups = [];
  for (const surface of SURFACES) {
    const rows = entries.filter((entry) => entry.surface === surface);
    if (!rows.length) continue;
    groups.push(rows.map((entry) => (
      `  { "n": ${entry.n}, "surface": ${JSON.stringify(entry.surface)}, ` +
      `"slug": ${JSON.stringify(entry.slug)}, "path": ${JSON.stringify(entry.path)}, ` +
      `"title": ${JSON.stringify(entry.title)}, "tabbar": ${entry.tabbar} }`
    )).join(',\n'));
  }
  return '[\n' + groups.join(',\n\n') + '\n]\n';
}

function sameJson(a, b) {
  return JSON.stringify(a) === JSON.stringify(b);
}

function prototypeDir(entry) {
  return path.join(BATCH, entry.surface, entry.slug);
}

function walkFiles(dir, base = dir) {
  if (!fs.existsSync(dir)) return [];
  const out = [];
  for (const name of fs.readdirSync(dir)) {
    if (name === 'node_modules' || name === '.git') continue;
    const full = path.join(dir, name);
    const stat = fs.statSync(full);
    if (stat.isDirectory()) out.push(...walkFiles(full, base));
    else out.push(rel(full));
  }
  return out;
}

function supportHash() {
  // 页面表现不只由 .vue 决定：共享组件、utils 里的状态文案、api 层的字段映射都会影响原型。
  // 这些目录跨页面复用，逐页归因噪声太大，因此做一个全局指纹并只告警。
  const roots = [
    'shared-ui',
    'parent/src/utils', 'parent/src/api', 'parent/src/components', 'parent/src/custom-tab-bar',
    'teacher/src/utils', 'teacher/src/api', 'teacher/src/components',
  ];
  const files = roots.flatMap((root) => walkFiles(path.join(repoRoot, root))).sort();
  const hash = crypto.createHash('sha256');
  for (const file of files) {
    hash.update(file);
    hash.update('\0');
    hash.update(normalize(fs.readFileSync(path.join(repoRoot, file), 'utf8')));
    hash.update('\n');
  }
  return { hash: hash.digest('hex'), files: files.length };
}

function sourceCommit(file) {
  try {
    return execFileSync('git', ['log', '-1', '--format=%h', '--', file], { cwd: repoRoot, encoding: 'utf8' }).trim();
  } catch {
    return '';
  }
}

function isDirty(file) {
  try {
    return execFileSync('git', ['status', '--porcelain', '--', file], { cwd: repoRoot, encoding: 'utf8' }).trim() !== '';
  } catch {
    return false;
  }
}

function generateTokens() {
  const tempRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'design-sync-'));
  try {
    execFileSync(process.execPath, [path.join(repoRoot, 'scripts/sync-open-design-adapter.mjs')], {
      cwd: repoRoot,
      env: { ...process.env, OPEN_DESIGN_ROOT: tempRoot },
      stdio: 'pipe',
    });
    const file = path.join(tempRoot, '.od/design-systems/nurture-planet/tokens.css');
    return normalize(fs.readFileSync(file, 'utf8'));
  } finally {
    fs.rmSync(tempRoot, { recursive: true, force: true });
  }
}

// 这些 utils 是登录/请求/格式化等基础设施，几乎每页都引用；放进逐页指纹只会制造噪声。
// 内容型 utils（leave / relation / tasks / notice / calendar / albums…）才逐页归因。
const DEP_DENY = new Set([
  'utils/auth', 'utils/login', 'utils/mediaRefresh', 'utils/time', 'utils/tabbar',
  'utils/currentStudent', 'utils/pendingRoute', 'utils/media', 'utils/nav',
]);

function resolveModule(base) {
  const candidates = [base, `${base}.js`, `${base}.ts`, path.join(base, 'index.js'), path.join(base, 'index.ts')];
  for (const candidate of candidates) {
    if (fs.existsSync(candidate) && fs.statSync(candidate).isFile()) return candidate;
  }
  return null;
}

function collectUtilDeps(file, surface, seen = new Set()) {
  if (seen.has(file)) return seen;
  seen.add(file);
  const text = fs.readFileSync(file, 'utf8');
  const srcRoot = path.join(repoRoot, surface, 'src');
  for (const match of text.matchAll(/(?:from\s+|import\s+)["']([^"']+)["']/g)) {
    const spec = match[1];
    let base = null;
    if (spec.startsWith('@/')) base = path.join(srcRoot, spec.slice(2));
    else if (spec.startsWith('./') || spec.startsWith('../')) base = path.resolve(path.dirname(file), spec);
    if (!base) continue;

    const relPath = path.relative(srcRoot, base).split(path.sep).join('/').replace(/\.(js|ts)$/, '');
    if (!relPath.startsWith('utils/')) continue;
    if (DEP_DENY.has(relPath)) continue;

    const resolved = resolveModule(base);
    if (resolved && (resolved.endsWith('.js') || resolved.endsWith('.ts'))) {
      collectUtilDeps(resolved, surface, seen);
    }
  }
  return seen;
}

function depsSha(entry) {
  const deps = [...collectUtilDeps(path.join(repoRoot, entry.path), entry.surface)].sort();
  const hash = crypto.createHash('sha256');
  for (const dep of deps) {
    hash.update(rel(dep));
    hash.update('\0');
    hash.update(normalize(fs.readFileSync(dep, 'utf8')));
    hash.update('\n');
  }
  return hash.digest('hex');
}

function lockKey(entry) {
  return `${entry.surface}/${entry.slug}`;
}

function loadLock() {
  if (!fs.existsSync(LOCK)) return { version: 1, supportHash: '', pages: {} };
  return readJson(LOCK);
}

function writeLock(lock) {
  const pages = Object.fromEntries(Object.entries(lock.pages).sort(([a], [b]) => a.localeCompare(b)));
  fs.writeFileSync(LOCK, JSON.stringify({ version: 1, supportHash: lock.supportHash, pages }, null, 2) + '\n');
}

function lockEntry(entry) {
  const commit = sourceCommit(entry.path);
  return {
    source: entry.path,
    sha256: sha256File(path.join(repoRoot, entry.path)),
    depsSha: depsSha(entry),
    title: entry.title,
    tabbar: entry.tabbar,
    reviewedAt: TODAY,
    codeCommit: commit ? (isDirty(entry.path) ? `${commit}-dirty` : commit) : '',
  };
}

function staleReasons(entry, saved) {
  if (!saved) return ['未登记'];
  const reasons = [];
  if (!saved.reviewedAt) reasons.push('新页面未复核');
  if (saved.source !== entry.path) reasons.push(`source 从 ${saved.source} 变为 ${entry.path}`);
  if (saved.sha256 !== sha256File(path.join(repoRoot, entry.path))) reasons.push('页面源码变了');
  if (saved.depsSha === undefined) reasons.push('依赖指纹未登记');
  else if (saved.depsSha !== depsSha(entry)) reasons.push('依赖的 utils 变了');
  if (saved.title !== entry.title) reasons.push(`导航标题变了: ${saved.title ?? '(空)'} → ${entry.title}`);
  if (saved.tabbar !== entry.tabbar) reasons.push('tabBar 归属变了');
  return reasons;
}

function checkPrototypeInventory(entries) {
  const errors = [];
  const known = new Set(entries.map((entry) => `${entry.surface}/${entry.slug}`));

  for (const entry of entries) {
    const dir = prototypeDir(entry);
    if (!fs.existsSync(path.join(dir, 'index.html'))) {
      errors.push(`缺少原型页面: ${rel(dir)}/index.html`);
    }
  }

  for (const surface of SURFACES) {
    const root = path.join(BATCH, surface);
    if (!fs.existsSync(root)) continue;
    for (const name of fs.readdirSync(root)) {
      const dir = path.join(root, name);
      if (!fs.statSync(dir).isDirectory()) continue;
      // 只有带 index.html 的目录才算页面原型，避免把 .file-versions 等辅助目录误报。
      if (!fs.existsSync(path.join(dir, 'index.html'))) continue;
      if (!known.has(`${surface}/${name}`)) {
        errors.push(`设计稿有页面但代码清单没有: ${surface}/${name}`);
      }
    }
  }
  return errors;
}

function prototypeTitle(entry) {
  const file = path.join(prototypeDir(entry), 'index.html');
  if (!fs.existsSync(file)) return null;
  const html = fs.readFileSync(file, 'utf8');
  const match = html.match(/<h1[^>]*class="[^"]*\bmp-title\b[^"]*"[^>]*>([\s\S]*?)<\/h1>/);
  if (!match) return null;
  return match[1]
    .replace(/<[^>]+>/g, '')
    .replace(/&amp;/g, '&')
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/\s+/g, ' ')
    .trim();
}

function checkPrototypeTitles(entries) {
  const errors = [];
  for (const entry of entries) {
    const title = prototypeTitle(entry);
    if (title === null) {
      errors.push(`原型缺少 .mp-title: ${entry.surface}/${entry.slug}`);
    } else if (title !== entry.title) {
      errors.push(`原型导航标题与 pages.json 不一致: ${entry.surface}/${entry.slug} ` +
        `proto="${title}" manifest="${entry.title}"`);
    }
  }
  return errors;
}

function checkGallery(entries) {
  const errors = [];
  if (!fs.existsSync(GALLERY)) return ['缺少画廊 index.html → 运行 npm run design:sync'];
  const html = fs.readFileSync(GALLERY, 'utf8');
  const readme = fs.existsSync(README) ? fs.readFileSync(README, 'utf8') : '';
  const known = new Set(entries.map((entry) => `${entry.surface}/${entry.slug}`));
  for (const entry of entries) {
    const needle = `${entry.surface}/${entry.slug}/`;
    if (!html.includes(needle)) errors.push(`画廊缺少页面: ${entry.surface}/${entry.slug}`);
    if (readme && !readme.includes(needle)) errors.push(`README 缺少页面: ${entry.surface}/${entry.slug}`);
  }
  for (const match of html.matchAll(/href="(teacher|parent)\/([^/"]+)\/index\.html"/g)) {
    const key = `${match[1]}/${match[2]}`;
    if (!known.has(key)) errors.push(`画廊有不存在页面: ${key}`);
  }
  return errors;
}

function previewIsStale(entry) {
  const dir = prototypeDir(entry);
  const preview = path.join(dir, 'preview.png');
  if (!fs.existsSync(preview)) return true;
  const previewMtime = fs.statSync(preview).mtimeMs;
  const assets = ['index.html', 'styles.css', 'app.js', 'tokens.css']
    .map((name) => path.join(dir, name))
    .filter((file) => fs.existsSync(file));
  return assets.some((file) => fs.statSync(file).mtimeMs > previewMtime + 2000);
}

function regenerateGallery() {
  execFileSync(process.execPath, [path.join(repoRoot, 'scripts/od-gallery.mjs'), BATCH], {
    cwd: repoRoot,
    stdio: 'pipe',
  });
}

function commandSync() {
  const entries = buildManifest();
  fs.writeFileSync(MANIFEST, manifestText(entries));

  const tokens = generateTokens();
  let written = 0;
  for (const entry of entries) {
    const file = path.join(prototypeDir(entry), 'tokens.css');
    if (!fs.existsSync(path.dirname(file))) continue;
    const current = fs.existsSync(file) ? normalize(fs.readFileSync(file, 'utf8')) : '';
    if (current !== tokens) {
      fs.writeFileSync(file, tokens);
      written += 1;
    }
  }

  const themeCount = writeThemesCss({ quiet: true });

  const lock = loadLock();
  const known = new Set(entries.map(lockKey));
  let pruned = 0;
  for (const key of Object.keys(lock.pages)) {
    if (!known.has(key)) {
      delete lock.pages[key];
      pruned += 1;
    }
  }
  let added = 0;
  for (const entry of entries) {
    const key = lockKey(entry);
    if (lock.pages[key]) continue;
    lock.pages[key] = {
      source: entry.path,
      sha256: null,
      depsSha: null,
      title: null,
      tabbar: null,
      reviewedAt: null,
      codeCommit: '',
    };
    added += 1;
  }
  writeLock(lock);

  try {
    regenerateGallery();
  } catch (error) {
    fail(`[design-sync] 画廊重建失败: ${error.message}`);
  }

  const stale = entries
    .map((entry) => ({ entry, reasons: staleReasons(entry, lock.pages[lockKey(entry)]) }))
    .filter((item) => item.reasons.length);

  console.log(`[design-sync] manifest: ${entries.length} 页`);
  console.log(`[design-sync] tokens.css: 更新 ${written} 个`);
  console.log(`[design-sync] themes.css: 更新 ${themeCount} 份`);
  console.log(`[design-sync] lock: 新增 ${added} 页（未复核），清理 ${pruned} 条失效条目`);
  console.log('[design-sync] 画廊: 已重建');
  const titleDrift = checkPrototypeTitles(entries);
  if (titleDrift.length) {
    console.log('[design-sync] 以下原型导航标题需手动同步到 pages.json:');
    for (const item of titleDrift) console.log(`  - ${item}`);
  }
  if (stale.length) {
    console.log('[design-sync] 以下页面需要复核：');
    for (const item of stale) console.log(`  - ${lockKey(item.entry)}: ${item.reasons.join('；')}`);
    console.log('  复核后运行: npm run design:accept -- <surface>/<slug>');
  } else {
    console.log('[design-sync] 所有页面标题 / tabBar / 源码指纹一致');
  }
}

function commandCheck() {
  const quick = flags.has('--quick');
  const entries = buildManifest();

  if (quick) {
    const staged = new Set(
      execFileSync('git', ['diff', '--cached', '--name-only'], { cwd: repoRoot, encoding: 'utf8' })
        .split('\n').map((line) => line.trim()).filter(Boolean),
    );
    const lock = loadLock();
    const stale = entries.filter((entry) => {
      if (!staged.has(entry.path) && !staged.has(`${entry.surface}/src/pages.json`)) return false;
      return staleReasons(entry, lock.pages[lockKey(entry)]).length > 0;
    });
    if (stale.length) {
      console.warn('[design-sync] 提醒：以下页面源码已改，设计稿指纹未更新（CI 会阻断）：');
      for (const entry of stale) console.warn(`  - ${lockKey(entry)}  (${entry.path})`);
      console.warn('  复核后运行: npm run design:accept -- <surface>/<slug>');
    }
    return;
  }

  const errors = [];
  const warnings = [];

  const current = fs.existsSync(MANIFEST) ? readJson(MANIFEST) : null;
  if (!sameJson(current, entries)) {
    errors.push('_shared/pages.json 与代码页面清单不一致 → 运行 npm run design:sync');
  }

  errors.push(...checkPrototypeInventory(entries));
  errors.push(...checkPrototypeTitles(entries));
  errors.push(...checkGallery(entries));

  const lock = loadLock();
  for (const entry of entries) {
    const reasons = staleReasons(entry, lock.pages[lockKey(entry)]);
    if (reasons.length) errors.push(`${lockKey(entry)}: ${reasons.join('；')}`);
  }
  const known = new Set(entries.map(lockKey));
  const extras = Object.keys(lock.pages).filter((key) => !known.has(key));
  if (extras.length) {
    errors.push(`design.lock.json 有不存在页面的条目（${extras.join(', ')}）→ 运行 npm run design:sync`);
  }

  const tokens = generateTokens();
  const tokenDrift = entries.filter((entry) => {
    const file = path.join(prototypeDir(entry), 'tokens.css');
    return !fs.existsSync(file) || normalize(fs.readFileSync(file, 'utf8')) !== tokens;
  });
  if (tokenDrift.length) {
    errors.push(`tokens.css 与 tokens.json 不一致（${tokenDrift.length} 页）→ 运行 npm run design:sync`);
  }

  const themesFile = path.join(BATCH, '_shared/themes.css');
  if (!fs.existsSync(themesFile) ||
      normalize(fs.readFileSync(themesFile, 'utf8')) !== normalize(renderThemesCss())) {
    errors.push('_shared/themes.css 与 themes.json 不一致 → 运行 npm run design:sync');
  }

  const support = supportHash();
  if (!lock.supportHash) {
    warnings.push('支撑代码尚未登记基线：运行 npm run design:accept -- --shared');
  } else if (lock.supportHash !== support.hash) {
    warnings.push('shared-ui / utils / api / components 已变更：复核影响后运行 npm run design:accept -- --shared');
  }

  const staleShots = entries.filter(previewIsStale);
  if (staleShots.length) {
    warnings.push(`${staleShots.length} 页原型资源比 preview.png 新：运行 npm run design:shots 重出截图`);
  }

  for (const warning of warnings) console.warn(`[design-sync] 警告: ${warning}`);
  if (errors.length) {
    console.error('[design-sync] 检查失败:');
    for (const error of errors) console.error(`- ${error}`);
    process.exit(1);
  }
  console.log(`[design-sync] 通过：${entries.length} 页，tokens / manifest / lock / 画廊一致`);
}

function commandAccept() {
  const entries = buildManifest();
  const lock = loadLock();

  // --shared alone只确认共享组件；与 --all 同时出现时走批量分支，避免静默忽略 --all。
  if (flags.has('--shared') && !flags.has('--all')) {
    lock.supportHash = supportHash().hash;
    writeLock(lock);
    console.log('[design-sync] 已登记支撑代码（shared-ui / utils / api / components）当前状态');
    return;
  }

  const all = flags.has('--all');
  if (all) {
    const changed = entries
      .map((entry) => ({ entry, reasons: staleReasons(entry, lock.pages[lockKey(entry)]) }))
      .filter((item) => item.reasons.length);
    const support = supportHash();
    const supportChanged = Boolean(lock.supportHash) && lock.supportHash !== support.hash;
    const blockers = changed.map((item) => `- ${lockKey(item.entry)}: ${item.reasons.join('；')}`);
    if (supportChanged && !flags.has('--shared')) blockers.push('- shared-ui / utils / api / components 已变更');
    if (blockers.length && !flags.has('--force')) {
      fail('[design-sync] 拒绝 --all：以下内容需要显式确认。\n' + blockers.join('\n') +
        '\n  页面变化请逐页 accept；只确认 shared-ui 加 --shared；确需覆盖全部加 --force。');
    }
  }

  const targets = all
    ? entries
    : entries.filter((entry) => positional.includes(lockKey(entry)) || positional.includes(entry.slug));

  if (!targets.length) {
    fail('用法: npm run design:accept -- <surface>/<slug>  或  --all / --shared');
  }

  for (const entry of targets) {
    lock.pages[lockKey(entry)] = lockEntry(entry);
  }
  if (all) lock.supportHash = supportHash().hash;
  writeLock(lock);

  console.log(`[design-sync] 已登记 ${targets.length} 页:`);
  for (const entry of targets) console.log(`  - ${lockKey(entry)}`);
  if (all) {
    console.log('  --all 已同时登记支撑代码（shared-ui / utils / api / components）当前状态。');
  }
}

switch (command) {
  case 'sync':
    commandSync();
    break;
  case 'check':
    commandCheck();
    break;
  case 'accept':
    commandAccept();
    break;
  case '--help':
  case 'help':
    console.log('用法: node scripts/design-sync.mjs <sync|check|accept> [--quick|--all|--force|--shared] [slug]');
    break;
  default:
    fail(`未知命令: ${command}\n用法: node scripts/design-sync.mjs <sync|check|accept>`, 2);
}
