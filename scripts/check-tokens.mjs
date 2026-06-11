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

// Token color → suggested replacement (used for helpful messages)
const COLOR_TO_TOKEN = new Map([
  // Shared text colors
  ['#1E293B', '$color-text-primary'],
  ['#334155', '$color-text-body'],
  ['#64748B', '$color-text-muted'],
  ['#94A3B8', '$color-text-hint'],
  // Shared semantic colors
  ['#16A34A', '$color-semantic-success'],
  ['#DCFCE7', '$color-semantic-success-bg'],
  ['#CA8A04', '$color-semantic-warning'],
  ['#FEF9C3', '$color-semantic-warning-bg'],
  ['#EF4444', '$color-semantic-danger'],
  ['#FEE2E2', '$color-semantic-danger-bg'],
  ['#2563EB', '$color-semantic-info'],
  ['#DBEAFE', '$color-semantic-info-bg'],
  // Brand colors — teacher
  ['#ff8a3d', '$color-brand-primary (teacher)'],
  ['#ffa866', '$color-brand-primary-light (teacher)'],
  ['#e6762a', '$color-brand-primary-dark (teacher)'],
  ['#fff7f0', '$color-brand-primary-bg (teacher)'],
  ['#f7f3ea', '$color-bg-page (teacher)'],
  ['#fffaf0', '$color-bg-card (teacher)'],
  // Brand colors — parent
  ['#F97316', '$color-brand-primary (parent)'],
  ['#FB923C', '$color-brand-primary-light (parent)'],
  ['#EA580C', '$color-brand-primary-dark (parent)'],
  ['#FFF7ED', '$color-brand-primary-bg (parent)'],
  ['#F8FAFC', '$color-bg-page (parent)'],
  ['#FFFFFF', '$color-bg-card (parent)'],
  // Brand colors — admin
  ['#3B82F6', '$color-brand-primary (admin)'],
  ['#60A5FA', '$color-brand-primary-light (admin)'],
  ['#1D4ED8', '$color-brand-primary-dark (admin)'],
  ['#EFF6FF', '$color-brand-primary-bg (admin)'],
  ['#F1F5F9', '$color-bg-page (admin)'],
  // Close matches (common tailwind aliases that appear in code)
  ['#10B981', '$color-semantic-success (close match)'],
  ['#22C55E', '$color-semantic-success (close match)'],
  ['#F59E0B', '$color-semantic-warning (close match)'],
]);

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
  let inStyle = false;

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const lineNum = i + 1;

    // Track <style> block
    if (/<style\b/i.test(line)) { inStyle = true; continue; }
    if (/<\/style>/i.test(line)) { inStyle = false; continue; }
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
    const hexRegex = /(?<!#)\b#([0-9a-fA-F]{3,8})\b(?![-\w])/g;
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
  // Inline `:style` bindings with hex colors are typically data-driven
  // (e.g., `:style="{ backgroundColor: item.bg, color: item.fg }"` where
  // colors come from JS arrays). Per the hardcoded whitelist, these are
  // legitimate exceptions — skip them.
  //
  // We only flag inline styles within actual `style=""` HTML attributes,
  // not Vue `:style` bindings which are computed at runtime.
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
