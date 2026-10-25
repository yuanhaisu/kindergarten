#!/usr/bin/env node

/**
 * check-tokens.mjs
 * Lightweight design token violation scanner for 养成星球.
 * Zero external dependencies — uses only Node.js builtins (fs, path).
 *
 * Usage:
 *   node scripts/check-tokens.mjs              # scan all projects
 *   node scripts/check-tokens.mjs teacher       # scan only teacher/
 *   node scripts/check-tokens.mjs admin         # scan only admin/
 *   node scripts/check-tokens.mjs parent        # scan only parent/
 *
 * Exit code: 0 = no violations, 1 = violations found (CI-friendly)
 */

import { readFileSync, readdirSync, statSync, existsSync } from 'fs';
import { join, relative, resolve, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const ROOT = resolve(__dirname, '..');

// ============================================================
// Configuration
// ============================================================

const SCAN_DIRS = {
  teacher: join(ROOT, 'teacher', 'src'),
  parent: join(ROOT, 'parent', 'src'),
  admin: join(ROOT, 'admin'),
};

// Directories to skip
const EXCLUDE_DIRS = new Set([
  'node_modules', '.nuxt', '.output', 'dist', '.git',
]);

// CSS properties whose numeric values should come from tokens
const NUMERIC_PROP_TOKENS = new Set([
  'border-radius', 'borderRadius',
  'padding', 'margin', 'gap', 'grid-gap',
  'font-size', 'fontSize',
]);

// rpx values with EXACT token matches, by property type
const TOKEN_RPX_BY_PROP = {
  spacing:   new Set([8, 12, 16, 24, 32]),                // spacing tokens
  radius:    new Set([8, 12, 16, 24, 19998]),              // border-radius tokens
  fontSize:  new Set([22, 24, 26, 28, 30, 32, 36, 40]),   // font-size tokens
};

function isTokenRpxValue(rpxVal, propName) {
  const prop = propName.toLowerCase();
  if (prop === 'font-size' || prop === 'fontsize') return TOKEN_RPX_BY_PROP.fontSize.has(rpxVal);
  if (prop === 'border-radius' || prop === 'borderradius') return TOKEN_RPX_BY_PROP.radius.has(rpxVal);
  // padding, margin, gap, grid-gap
  return TOKEN_RPX_BY_PROP.spacing.has(rpxVal);
}

// Values that are only *close* to a token. They stay hand-listed because no
// single token owns them; everything else is derived from tokens.json below.
const CLOSE_MATCH_ALIASES = new Map([
  ['#10B981', '$color-semantic-success (close match)'],
  ['#22C55E', '$color-semantic-success (close match)'],
  ['#F59E0B', '$color-semantic-warning (close match)'],
]);

// Token color → suggested replacement (used for helpful messages).
//
// Derived from tokens.json at startup — do NOT hand-maintain this again. The
// previous literal map silently drifted from the token source: it was missing
// `border.hairline` (#CBD5E1), so every correct use of the border token was
// reported as "No matching token". Adding a token now teaches the checker
// about it automatically.
const COLOR_TO_TOKEN = buildColorTokenMap();

function buildColorTokenMap() {
  const map = new Map();
  let tokens;
  try {
    tokens = JSON.parse(readFileSafe(join(ROOT, 'shared-design-tokens', 'tokens.json')));
  } catch {
    // Unreadable tokens.json — hints degrade to "No matching token" rather
    // than crashing the scan.
    return map;
  }

  const ENDS = new Set(['teacher', 'parent', 'admin']);

  // style-dictionary emits kebab-case: `successBg` -> `$color-semantic-success-bg`.
  // Hints must match the generated variable exactly, or copy-paste fails.
  const kebab = (s) => s.replace(/([a-z0-9])([A-Z])/g, '$1-$2').toLowerCase();

  // Path arrives as e.g. ['shared', 'color', 'border', 'hairline'].
  const walk = (node, path) => {
    if (!node || typeof node !== 'object') return;
    const raw = node.value;
    if (typeof raw === 'string' && /^#[0-9a-fA-F]{3,8}$/.test(raw.trim())) {
      const key = normalizeHex(raw.trim());
      if (map.has(key)) return; // first occurrence wins; shared scope comes first
      const [scope, ...rest] = path;
      const varName = `$${rest.map(kebab).join('-')}`;
      map.set(key, ENDS.has(scope) ? `${varName} (${scope})` : varName);
      return;
    }
    for (const [k, v] of Object.entries(node)) walk(v, [...path, k]);
  };

  for (const [scope, body] of Object.entries(tokens)) walk(body, [scope]);
  for (const [k, v] of CLOSE_MATCH_ALIASES) if (!map.has(k)) map.set(k, v);
  return map;
}

// Track statistics
let violations = 0;
let scannedFiles = 0;

// ============================================================
// Helpers
// ============================================================

function warn(file, line, msg, suggestion) {
  violations++;
  const loc = line ? `:${line}` : '';
  console.log(`  ${file}${loc}`);
  console.log(`    ❌ ${msg}`);
  if (suggestion) console.log(`    ↪ ${suggestion}`);
  console.log('');
}

function readFileSafe(filePath) {
  try { return readFileSync(filePath, 'utf-8'); }
  catch { return ''; }
}

function collectFiles(dir, exts) {
  const results = [];
  try {
    for (const entry of readdirSync(dir, { withFileTypes: true })) {
      const full = join(dir, entry.name);
      if (entry.isDirectory()) {
        if (EXCLUDE_DIRS.has(entry.name)) continue;
        results.push(...collectFiles(full, exts));
      } else if (entry.isFile()) {
        const ext = entry.name.split('.').pop();
        if (exts.has(ext)) results.push(full);
      }
    }
  } catch { /* dir missing — skip */ }
  return results;
}

function normalizeHex(hex) {
  let h = hex.toUpperCase();
  // Normalize 3-char hex to 6-char for matching
  if (h.length === 3 || h.length === 4) {
    h = '#' + h.slice(1).split('').map(c => c + c).join('');
  }
  return h;
}

// ============================================================
// SCSS Scanner (for teacher/ parent mini-programs)
// ============================================================

function scanSCSS(filePath, content) {
  const lines = content.split('\n');
  // Standalone .scss files (uni.scss 等) have no <style> wrapper — scan them
  // wholesale. Previously they were skipped entirely unless some comment
  // happened to contain the literal "<style", which is how parent/src/uni.scss
  // was partially scanned by accident.
  const isVue = filePath.toLowerCase().endsWith('.vue');
  let inStyle = !isVue;

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const lineNum = i + 1;

    // Track <style> block. Anchor to line start so a comment mentioning
    // "<style>" cannot switch scanning on early.
    if (/^\s*<style\b/i.test(line)) { inStyle = true; continue; }
    if (/^\s*<\/style>/i.test(line)) { inStyle = false; continue; }
    if (!inStyle) continue;

    // Skip comment-only lines
    const trimmed = line.trim();
    if (trimmed.startsWith('//') || trimmed.startsWith('/*')) continue;

    // ===== Check 1: Local variable redefinition =====
    const varDef = trimmed.match(/^\$([a-zA-Z][\w-]+)\s*:\s*(.+?);$/);
    if (varDef) {
      const [, varName, varValue] = varDef;
      const colorKey = varValue.trim().toUpperCase();
      const suggestion = COLOR_TO_TOKEN.get(colorKey);
      if (suggestion) {
        warn(relative(ROOT, filePath), lineNum,
          `Local SCSS variable redefines token value: \`$${varName}: ${varValue.trim()}\``,
          `Remove this definition; use \`${suggestion}\` directly`
        );
      }
    }

    // ===== Check 2: Hex colors =====
    // (?<!#) 只用来避免匹配 "##..."。这里**不能**再加 \b：CSS 里 # 前永远是
    // 空格 / ':' / '(' / ',' 这类非词字符，`\b` 要求前一个字符是词字符，
    // 边界在真实样式里根本不存在 —— 加了它整条正则会静默永不命中（2026-10 修复）。
    const hexRegex = /(?<!#)#([0-9a-fA-F]{3,8})\b(?![-\w])/g;
    let hexMatch;
    while ((hexMatch = hexRegex.exec(line)) !== null) {
      const hex = hexMatch[0];
      const normalized = normalizeHex(hex);

      // Allow white/black in certain contexts
      if (hex.toLowerCase() === '#ffffff' || hex.toLowerCase() === '#fff') {
        // Only flag if it's explicitly a background or color property
        if (/\b(background|color)\s*:/i.test(line)) {
          warn(relative(ROOT, filePath), lineNum,
            `Hardcoded white color: \`${hex}\``,
            `Consider using \`$color-bg-card\` or \`$color-bg-page\``
          );
        }
        continue;
      }

      // Allow hex inside gradient functions (single-use patterns)
      if (/linear-gradient|radial-gradient|conic-gradient/i.test(line)) {
        // Still note it, but as info
        const sugg = COLOR_TO_TOKEN.get(normalized);
        warn(relative(ROOT, filePath), lineNum,
          `Hardcoded color in gradient: \`${hex}\``,
          sugg
            ? `Consider replacing with token: \`${sugg}\``
            : 'If single-use gradient, allowed — add comment explaining why'
        );
        continue;
      }

      // Allow hex inside rgba() — not a direct hex usage
      if (/rgba?\s*\(/.test(line)) continue;

      // Otherwise: violation
      const suggestion = COLOR_TO_TOKEN.get(normalized);
      warn(relative(ROOT, filePath), lineNum,
        `Hardcoded color: \`${hex}\``,
        suggestion
          ? `Use token: \`${suggestion}\``
          : 'No matching token. Add to tokens.json first, or add comment if single-use exception'
      );
    }

    // ===== Check 3: Hardcoded rpx values on token-covered properties =====
    if (filePath.includes('teacher') || filePath.includes('parent')) {
      for (const prop of NUMERIC_PROP_TOKENS) {
        const propRegex = new RegExp(
          `\\b${prop.replace('-', '\\-')}\\s*:\\s*(\\d+)rpx`,
          'i'
        );
        const m = line.match(propRegex);
        if (m) {
          const rpxVal = parseInt(m[1]);
          // Only flag if this rpx value has an exact token match for THIS property
          if (isTokenRpxValue(rpxVal, prop)) {
            const tokenHint = prop.toLowerCase().includes('font-size')
              ? '$typography-font-size-*'
              : prop.toLowerCase().includes('border-radius')
              ? '$border-radius-*'
              : '$spacing-*';
            warn(relative(ROOT, filePath), lineNum,
              `Hardcoded \`${m[1]}rpx\` for \`${prop}\``,
              `Use the corresponding \`${tokenHint}\` token`
            );
          }
        }
      }
    }
  }
}

// ============================================================
// Inline Style Scanner (for Vue inline :style bindings)
// ============================================================

function scanInlineStyles(filePath, content) {
  // Vue `:style` / `v-bind:style` bindings are computed at runtime (often from
  // API data such as `item.badge.color`), so they are intentionally NOT scanned
  // — flagging them would produce noise on data-driven values.
  //
  // Static `style=""` attributes ARE authored literals and must obey the same
  // token rules as <style> blocks. `placeholder-style` is reported as info
  // rather than a violation: the mini-program API only accepts a literal there
  // and cannot reference a SCSS variable.
  const lines = content.split('\n');
  let templateDepth = 0;

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const lineNum = i + 1;

    // Only scan inside <template> — avoids matching `style = "..."` in <script>.
    if (/<template\b/i.test(line)) { templateDepth++; continue; }
    if (/<\/template>/i.test(line)) { templateDepth = Math.max(0, templateDepth - 1); continue; }
    if (templateDepth === 0) continue;

    const attrRe = /([:\w-]*style)\s*=\s*"([^"]*)"/g;
    let attrMatch;

    while ((attrMatch = attrRe.exec(line)) !== null) {
      const attrName = attrMatch[1];
      if (attrName.includes(':')) continue; // Vue binding — runtime value
      const body = attrMatch[2];
      const literalOnly = attrName === 'placeholder-style';

      const hexRegex = /(?<!#)#([0-9a-fA-F]{3,8})\b(?![-\w])/g;
      let hexMatch;
      while ((hexMatch = hexRegex.exec(body)) !== null) {
        const hex = hexMatch[0];
        const suggestion = COLOR_TO_TOKEN.get(normalizeHex(hex));

        if (literalOnly) {
          console.log(`  ${relative(ROOT, filePath)}:${lineNum}`);
          console.log(`    ℹ️  Literal-only \`${attrName}\` uses \`${hex}\`${suggestion ? ` (= ${suggestion})` : ''}`);
          console.log('    ↪ Mini-program API accepts literals only — keep this value in sync with tokens.json manually');
          console.log('');
          continue;
        }

        warn(relative(ROOT, filePath), lineNum,
          `Hardcoded color in inline style: \`${hex}\``,
          suggestion
            ? `Use token: \`${suggestion}\``
            : 'No matching token. Add to tokens.json first, or add comment if single-use exception'
        );
      }

      if (literalOnly) continue;
      for (const prop of NUMERIC_PROP_TOKENS) {
        const propRegex = new RegExp(`\\b${prop.replace('-', '\\-')}\\s*:\\s*(\\d+)rpx`, 'i');
        const m = body.match(propRegex);
        if (!m) continue;
        if (!isTokenRpxValue(parseInt(m[1]), prop)) continue;
        const tokenHint = prop.toLowerCase().includes('font-size')
          ? '$typography-font-size-*'
          : prop.toLowerCase().includes('border-radius')
          ? '$border-radius-*'
          : '$spacing-*';
        warn(relative(ROOT, filePath), lineNum,
          `Hardcoded \`${m[1]}rpx\` for \`${prop}\` in inline style`,
          `Use the corresponding \`${tokenHint}\` token`
        );
      }
    }
  }
}

// ============================================================
// Tailwind Scanner (for admin project)
// ============================================================

function scanTailwind(filePath, content) {
  const lines = content.split('\n');

  const arbitraryPatterns = [
    { regex: /(?:bg|text|border|ring|outline|shadow|fill|stroke)-\[#([^\]]+)\]/g, type: 'color' },
    { regex: /rounded-\[(\d+)(px|rpx)?\]/g, type: 'borderRadius' },
    { regex: /\b[pm]x?[trblxy]?-\[(\d+)(px|rpx)?\]/g, type: 'spacing' },
    { regex: /gap-[xy]?-\[(\d+)(px|rpx)?\]/g, type: 'spacing' },
    { regex: /text-\[(\d+)(px)\]/g, type: 'fontSize' },
    { regex: /shadow-\[(.+?)\]/g, type: 'shadow' },
    { regex: /leading-\[(\d+)(px)?\]/g, type: 'lineHeight' },
    { regex: /tracking-\[(.+?)\]/g, type: 'letterSpacing' },
  ];

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const lineNum = i + 1;

    // Skip if it's commented out
    if (/^\s*\/\//.test(line) || /^\s*<!--/.test(line)) continue;

    for (const { regex, type } of arbitraryPatterns) {
      const regexCopy = new RegExp(regex.source, regex.flags);
      let match;
      while ((match = regexCopy.exec(line)) !== null) {
        // Allow layout dimensions (w-[], h-[], min-w-[], etc) — not token-covered
        const classStr = match[0];
        if (/^[wh](?:-full|-screen|-min|-max|-fit|-auto)?-\[/.test(classStr)) continue;
        if (/^(?:top|bottom|left|right|inset)-\[/.test(classStr)) continue;

        warn(relative(ROOT, filePath), lineNum,
          `Tailwind arbitrary value: \`${classStr}\``,
          type === 'color'
            ? 'Replace with Tailwind token class (e.g. bg-brand-primary, text-text-muted)'
            : type === 'borderRadius'
            ? 'Replace with token class (e.g. rounded-sm, rounded-lg, rounded-xl)'
            : type === 'spacing'
            ? 'Replace with token class (e.g. p-4, m-6, gap-3)'
            : `Replace with the corresponding Tailwind token class for ${type}`
        );
      }
    }
  }
}

// ============================================================
// Build Freshness Check
// ============================================================

function checkBuildFreshness() {
  const tokensFile = join(ROOT, 'shared-design-tokens', 'tokens.json');
  const buildDir = join(ROOT, 'shared-design-tokens', 'build');

  if (!existsSync(tokensFile)) {
    console.error('  ❌ tokens.json not found at shared-design-tokens/tokens.json\n');
    return false;
  }

  if (!existsSync(buildDir)) {
    console.error('  ❌ build/ directory not found. Run: cd shared-design-tokens && npm run build\n');
    return false;
  }

  const tokenMtime = statSync(tokensFile).mtimeMs;
  let allFresh = true;

  function checkRecursive(dir) {
    if (!existsSync(dir)) return;
    for (const entry of readdirSync(dir, { withFileTypes: true })) {
      const full = join(dir, entry.name);
      if (entry.isDirectory()) {
        if (EXCLUDE_DIRS.has(entry.name)) continue;
        checkRecursive(full);
      } else if (entry.isFile() && /\.(scss|ts)$/i.test(entry.name)) {
        if (statSync(full).mtimeMs < tokenMtime) {
          console.error(`  ⚠️  STALE: ${relative(ROOT, full)} (older than tokens.json)`);
          allFresh = false;
        }
      }
    }
  }

  checkRecursive(buildDir);
  if (!allFresh) {
    console.error('  ↪ Run: cd shared-design-tokens && npm run build\n');
    violations++;
  }
  return allFresh;
}

// ============================================================
// Main
// ============================================================

function main() {
  const target = process.argv[2] || 'all';
  const targets = target === 'all' ? ['teacher', 'parent', 'admin'] : [target];

  console.log('\n🔍 check-tokens — scanning for design token violations...\n');

  // Check build freshness for mini-program targets
  if (target === 'all' || targets.includes('teacher') || targets.includes('parent')) {
    checkBuildFreshness();
  }

  const VALID_TARGETS = ['teacher', 'parent', 'admin'];

  for (const t of targets) {
    if (!VALID_TARGETS.includes(t)) {
      console.error(`  ❌ Unknown target: ${t}. Use: teacher, parent, admin, or all`);
      process.exit(1);
    }

    const srcDir = SCAN_DIRS[t];
    if (!existsSync(srcDir)) {
      console.log(`  [SKIP] ${t}/src/ not found\n`);
      continue;
    }

    // Collect files
    const vueFiles = collectFiles(srcDir, new Set(['vue']));
    const scssFiles = collectFiles(srcDir, new Set(['scss']));

    // Scan Vue files (SCSS blocks + inline styles)
    for (const file of vueFiles) {
      scannedFiles++;
      const content = readFileSafe(file);
      scanSCSS(file, content);
      scanInlineStyles(file, content);
      if (t === 'admin') scanTailwind(file, content);
    }

    // Scan standalone SCSS files
    for (const file of scssFiles) {
      scannedFiles++;
      const content = readFileSafe(file);
      scanSCSS(file, content);
    }
  }

  // ===== Report =====
  console.log(`  Scanned ${scannedFiles} files.\n`);

  if (violations === 0) {
    console.log('✅ No design token violations found!\n');
    process.exit(0);
  }

  console.log(`❗ ${violations} violation(s) found.\n`);
  process.exit(1);
}

main();
