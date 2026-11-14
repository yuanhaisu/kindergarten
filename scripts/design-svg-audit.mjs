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
    const own = parseAttrs(rawAttrs);
    const merged = { ...stack[stack.length - 1], ...own };
    elements.push({
      tag,
      attrs: own,
      stroke: merged.stroke || 'none',
      strokeWidth: merged['stroke-width'] || null,
      fill: merged.fill === undefined ? '#000000' : merged.fill,
      linecap: merged['stroke-linecap'] || null,
      linejoin: merged['stroke-linejoin'] || null,
    });
  }
  return { root, elements };
}

// Approximate geometry bbox. Path parsing covers M/L/H/V/C/S/Q/T/A with
// absolute and relative commands; control points are included, so curves may
// overestimate by a pixel or two — acceptable for a range check.
function pathBBox(d) {
  const tokens = String(d).match(/[a-zA-Z]|-?\d*\.?\d+(?:e[-+]?\d+)?/gi) || [];
  let i = 0;
  let cmd = '';
  let cx = 0;
  let cy = 0;
  let sx = 0;
  let sy = 0;
  let minX = Infinity;
  let minY = Infinity;
  let maxX = -Infinity;
  let maxY = -Infinity;
  const add = (x, y) => {
    minX = Math.min(minX, x);
    minY = Math.min(minY, y);
    maxX = Math.max(maxX, x);
    maxY = Math.max(maxY, y);
  };
  const num = () => Number(tokens[i++]);
  const isCmd = (t) => /^[a-zA-Z]$/.test(t);
  while (i < tokens.length) {
    if (isCmd(tokens[i])) cmd = tokens[i++];
    if (!cmd) break;
    const rel = cmd === cmd.toLowerCase();
    const C = cmd.toUpperCase();
    if (C === 'M' || C === 'L') {
      const x = num() + (rel ? cx : 0);
      const y = num() + (rel ? cy : 0);
      cx = x;
      cy = y;
      if (C === 'M') {
        sx = x;
        sy = y;
        cmd = rel ? 'l' : 'L';
      }
      add(x, y);
    } else if (C === 'H') {
      cx = num() + (rel ? cx : 0);
      add(cx, cy);
    } else if (C === 'V') {
      cy = num() + (rel ? cy : 0);
      add(cx, cy);
    } else if (C === 'C' || C === 'S' || C === 'Q') {
      const pairs = C === 'C' ? 3 : 2;
      const points = [];
      for (let p = 0; p < pairs; p += 1) {
        points.push([num() + (rel ? cx : 0), num() + (rel ? cy : 0)]);
      }
      for (const [x, y] of points) add(x, y);
      [cx, cy] = points[points.length - 1];
    } else if (C === 'T') {
      cx = num() + (rel ? cx : 0);
      cy = num() + (rel ? cy : 0);
      add(cx, cy);
    } else if (C === 'A') {
      num();
      num();
      num();
      num();
      num();
      const x = num() + (rel ? cx : 0);
      const y = num() + (rel ? cy : 0);
      add(cx, cy);
      add(x, y);
      cx = x;
      cy = y;
    } else if (C === 'Z') {
      cx = sx;
      cy = sy;
    } else if (i < tokens.length && !isCmd(tokens[i])) {
      i += 1;
    }
  }
  if (minX === Infinity) return null;
  return { x: minX, y: minY, w: maxX - minX, h: maxY - minY };
}

function geometryBBox(element) {
  const attrs = element.attrs || {};
  if (element.tag === 'path') return pathBBox(attrs.d);
  if (element.tag === 'rect') {
    const x = Number(attrs.x || 0);
    const y = Number(attrs.y || 0);
    return { x, y, w: Number(attrs.width || 0), h: Number(attrs.height || 0) };
  }
  if (element.tag === 'circle') {
    const cx = Number(attrs.cx || 0);
    const cy = Number(attrs.cy || 0);
    const r = Number(attrs.r || 0);
    return { x: cx - r, y: cy - r, w: r * 2, h: r * 2 };
  }
  if (element.tag === 'line') {
    const x1 = Number(attrs.x1 || 0);
    const y1 = Number(attrs.y1 || 0);
    const x2 = Number(attrs.x2 || 0);
    const y2 = Number(attrs.y2 || 0);
    return { x: Math.min(x1, x2), y: Math.min(y1, y2), w: Math.abs(x2 - x1), h: Math.abs(y2 - y1) };
  }
  if (element.tag === 'polyline' || element.tag === 'polygon') {
    const nums = (attrs.points || '').match(/-?\d*\.?\d+/g)?.map(Number) || [];
    const xs = [];
    const ys = [];
    for (let i = 0; i + 1 < nums.length; i += 2) {
      xs.push(nums[i]);
      ys.push(nums[i + 1]);
    }
    if (xs.length === 0) return null;
    return {
      x: Math.min(...xs),
      y: Math.min(...ys),
      w: Math.max(...xs) - Math.min(...xs),
      h: Math.max(...ys) - Math.min(...ys),
    };
  }
  return null;
}

function unionBBox(boxes) {
  const valid = boxes.filter(Boolean);
  if (valid.length === 0) return null;
  const x = Math.min(...valid.map((b) => b.x));
  const y = Math.min(...valid.map((b) => b.y));
  const maxX = Math.max(...valid.map((b) => b.x + b.w));
  const maxY = Math.max(...valid.map((b) => b.y + b.h));
  return { x, y, w: maxX - x, h: maxY - y };
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

  const bbox = unionBBox(elements.map(geometryBBox));

  if (elements.length < rule.elements.min || elements.length > rule.elements.max) {
    issues.push(`元素数 ${elements.length} 超出 ${rule.elements.min}–${rule.elements.max}`);
  }

  if (!bbox) {
    issues.push('无法计算几何范围（bbox）');
  } else if (type === 'icon') {
    if (bbox.x < 1.5 || bbox.y < 1.5 || bbox.x + bbox.w > 22.5 || bbox.y + bbox.h > 22.5) {
      issues.push(
        `内容 bbox ${bbox.x.toFixed(1)},${bbox.y.toFixed(1)} ${bbox.w.toFixed(1)}×${bbox.h.toFixed(1)} 超出 2–22 安全区`
      );
    }
    if (bbox.w < 8 || bbox.h < 8) {
      issues.push(`内容过小：${bbox.w.toFixed(1)}×${bbox.h.toFixed(1)}，至少 8×8`);
    }
  } else {
    if (bbox.x < 4 || bbox.y < 4 || bbox.x + bbox.w > 116 || bbox.y + bbox.h > 116) {
      issues.push(`插画 bbox 超出 4–116 网格留白`);
    }
    if (bbox.w < 40 || bbox.w > 90 || bbox.h < 30 || bbox.h > 90) {
      issues.push(`插画主体 ${bbox.w.toFixed(1)}×${bbox.h.toFixed(1)} 超出实测区间（宽 40–90 / 高 30–90）`);
    }
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
      bbox: bbox
        ? {
            x: Number(bbox.x.toFixed(1)),
            y: Number(bbox.y.toFixed(1)),
            w: Number(bbox.w.toFixed(1)),
            h: Number(bbox.h.toFixed(1)),
          }
        : null,
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
        `      viewBox=${r.stats.viewBox}  elements=${r.stats.elements}  stroke=${r.stats.strokeWidths.join('/')}  fills=${r.stats.fills.join(',')}  bbox=${r.stats.bbox ? `${r.stats.bbox.x},${r.stats.bbox.y} ${r.stats.bbox.w}×${r.stats.bbox.h}` : 'n/a'}`
      );
    }
    for (const issue of r.issues) console.log(`      ✗ ${issue}`);
    for (const warning of r.warnings) console.log(`      ! ${warning}`);
  }
}

process.exit(results.every((r) => r.pass) ? 0 : 1);
