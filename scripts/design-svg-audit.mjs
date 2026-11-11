#!/usr/bin/env node
/**
 * Audit SVG assets against the Kinderly-derived grammar.
 *
 * Measured from /picks (2026-11-11):
 *   - 24-grid UI icons: 27 samples, stroke 1.6–2.0 (per-icon optical), no fill,
 *     1–5 elements, bbox roughly 4–20 with 2–5 asymmetric padding.
 *   - 120-grid illustrations: 12 samples, outline stroke 3, detail 2, accent 4–6,
 *     1–3 fills from the earth palette, 2–8 elements, subject bbox 52–80 wide.
 *
 * This script checks the mechanical rules only (viewBox / stroke / fill / count).
 * Optical quality and 14px legibility still need a human or vision review.
 *
 * Usage:
 *   node scripts/design-svg-audit.mjs <file.svg> [more.svg...] [--json]
 * Exit code 0 = all pass, 1 = at least one failure.
 */
import fs from 'node:fs';
import path from 'node:path';

const GRAMMAR = {
  icon: {
    label: 'UI icon (24 grid)',
    viewBox: '0 0 24 24',
    stroke: { min: 1.6, max: 2.0 },
    fills: ['none'],
    elements: { min: 1, max: 5 },
  },
  illustration: {
    label: 'illustration (120 grid)',
    viewBox: '0 0 120 120',
    stroke: { allowed: [2, 3, 4, 6], required: 3 },
    fills: ['#c36540', '#8da386', '#ddb14a', '#fcfaf4', '#173220'],
    elements: { min: 2, max: 8 },
  },
};

const TAG_RE = /<(\/?)(g|path|rect|circle|line|polyline|polygon|svg)\b([^>]*?)(\/?)>/g;

function parseAttrs(raw) {
  const out = {};
  for (const m of String(raw).matchAll(/([\w:-]+)="([^"]*)"/g)) out[m[1]] = m[2];
  return out;
}

function normalizeColor(value) {
  const v = String(value || '').trim().toLowerCase();
  const rgb = v.match(/^rgb\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*\)$/);
  if (!rgb) return v;
  return (
    '#' +
    [rgb[1], rgb[2], rgb[3]]
      .map((n) => Number(n).toString(16).padStart(2, '0'))
      .join('')
  );
}

function collect(file) {
  const text = fs.readFileSync(file, 'utf8');
  const svgMatch = text.match(/<svg\b([^>]*)>/);
  if (!svgMatch) throw new Error('no <svg> element');
  const root = parseAttrs(svgMatch[1]);
  const elements = [];
  const stack = [root];
  TAG_RE.lastIndex = 0;
  for (const m of text.matchAll(TAG_RE)) {
    const [, closing, tag, rawAttrs, selfClosing] = m;
    if (tag === 'svg') continue;
    if (tag === 'g') {
      if (closing) {
        if (stack.length > 1) stack.pop();
      } else if (!selfClosing) {
        stack.push({ ...stack[stack.length - 1], ...parseAttrs(rawAttrs) });
      }
      continue;
    }
    if (closing) continue;
    const merged = { ...stack[stack.length - 1], ...parseAttrs(rawAttrs) };
    elements.push({
      tag,
      stroke: merged.stroke || 'none',
      strokeWidth: merged['stroke-width'] || null,
      fill: merged.fill === undefined ? '#000000' : merged.fill,
      linecap: merged['stroke-linecap'] || null,
      linejoin: merged['stroke-linejoin'] || null,
    });
  }
  return { root, elements };
}

function audit(file) {
  const { root, elements } = collect(file);
  const viewBox = (root.viewBox || '').trim();
  const type = viewBox === GRAMMAR.icon.viewBox ? 'icon' : viewBox === GRAMMAR.illustration.viewBox ? 'illustration' : null;
  const issues = [];
  const warnings = [];

  if (!type) {
    return {
      file,
      pass: false,
      type: 'unknown',
      issues: [`viewBox 不在语法内：${viewBox || '(missing)'}，只允许 0 0 24 24 或 0 0 120 120`],
      warnings,
    };
  }

  const rule = GRAMMAR[type];
  const strokeWidths = [];
  const fills = new Set();

  for (const el of elements) {
    if (el.stroke !== 'none') {
      const width = Number.parseFloat(el.strokeWidth ?? '1');
      if (!Number.isNaN(width)) strokeWidths.push(width);
    }
    const fill = normalizeColor(el.fill);
    fills.add(fill);
  }

  if (elements.length < rule.elements.min || elements.length > rule.elements.max) {
    issues.push(`元素数 ${elements.length} 超出 ${rule.elements.min}–${rule.elements.max}`);
  }

  if (type === 'icon') {
    for (const w of strokeWidths) {
      if (w < rule.stroke.min - 0.001 || w > rule.stroke.max + 0.001) {
        issues.push(`线宽 ${w} 超出 1.6–2.0（逐个图标光学修正，但不得越界）`);
      }
    }
    for (const fill of fills) {
      if (fill !== 'none') issues.push(`图标必须无填充，发现 fill=${fill}`);
    }
  } else {
    for (const w of strokeWidths) {
      if (!rule.stroke.allowed.includes(w)) {
        issues.push(`线宽 ${w} 不在允许集合 ${rule.stroke.allowed.join('/')} 内`);
      }
    }
    if (!strokeWidths.includes(rule.stroke.required)) {
      issues.push('缺少 3px 主体轮廓线');
    }
    for (const fill of fills) {
      if (fill !== 'none' && !rule.fills.includes(fill)) {
        issues.push(`填充色 ${fill} 不在白名单 [${rule.fills.join(', ')}]`);
      }
    }
    const colorFills = [...fills].filter((f) => f !== 'none');
    if (colorFills.length > 3) issues.push(`填充色 ${colorFills.length} 个，超过 3 个`);
  }

  for (const el of elements) {
    if (el.stroke !== 'none' && el.linecap && el.linecap !== 'round') {
      warnings.push(`stroke-linecap=${el.linecap}，语法要求 round`);
    }
    if (el.stroke !== 'none' && el.linejoin && el.linejoin !== 'round') {
      warnings.push(`stroke-linejoin=${el.linejoin}，语法要求 round`);
    }
  }

  return {
    file,
    pass: issues.length === 0,
    type,
    label: rule.label,
    stats: {
      viewBox,
      elements: elements.length,
      strokeWidths: [...new Set(strokeWidths)].sort((a, b) => a - b),
      fills: [...fills],
    },
    issues,
    warnings,
  };
}

const args = process.argv.slice(2);
const json = args.includes('--json');
const files = args.filter((a) => a !== '--json');
if (files.length === 0) {
  console.error('usage: node scripts/design-svg-audit.mjs <file.svg> [...] [--json]');
  process.exit(1);
}

const results = files.map((file) => {
  try {
    return audit(path.resolve(file));
  } catch (e) {
    return { file, pass: false, type: 'error', issues: [e.message], warnings: [] };
  }
});

if (json) {
  console.log(JSON.stringify(results, null, 2));
} else {
  for (const r of results) {
    const mark = r.pass ? 'PASS' : 'FAIL';
    console.log(`${mark}  ${path.relative(process.cwd(), r.file)}  [${r.label || r.type}]`);
    if (r.stats) {
      console.log(
        `      viewBox=${r.stats.viewBox}  elements=${r.stats.elements}  stroke=${r.stats.strokeWidths.join('/')}  fills=${r.stats.fills.join(',')}`
      );
    }
    for (const issue of r.issues) console.log(`      ✗ ${issue}`);
    for (const warning of r.warnings) console.log(`      ! ${warning}`);
  }
}

process.exit(results.every((r) => r.pass) ? 0 : 1);
