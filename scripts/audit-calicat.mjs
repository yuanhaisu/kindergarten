#!/usr/bin/env node

/**
 * audit-calicat.mjs
 * Scans design-data/ JSON files extracted from Calicat, extracts visual values,
 * and compares them against shared-design-tokens/tokens.json.
 *
 * Usage:
 *   node scripts/audit-calicat.mjs              # full audit
 *   node scripts/audit-calicat.mjs --colors     # colors only
 *   node scripts/audit-calicat.mjs --summary    # summary only
 */

import { readFileSync, readdirSync, existsSync } from 'fs';
import { join, resolve, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const ROOT = resolve(__dirname, '..');

const DESIGN_DATA_DIR = join(ROOT, 'design-data');
const TOKENS_FILE = join(ROOT, 'shared-design-tokens', 'tokens.json');

// Load tokens
const tokens = JSON.parse(readFileSync(TOKENS_FILE, 'utf-8'));

// Flatten shared + all platform tokens into lookup sets
const tokenColorSet = new Set();
const tokenFontSizeSet = new Set();   // raw values (pre-transform)
const tokenRadiusSet = new Set();     // raw values
const tokenSpacingSet = new Set();    // raw values
const tokenShadowSet = new Set();     // string values

function collectTokenValues(obj) {
  if (!obj || typeof obj !== 'object') return;
  if (obj.hasOwnProperty('value')) {
    const val = obj.value;
    const attrs = obj.attributes || {};
    const unit = attrs.unit;

    if (unit === 'fontSize') tokenFontSizeSet.add(val);
    else if (unit === 'borderRadius') tokenRadiusSet.add(val);
    else if (unit === 'spacing') tokenSpacingSet.add(val);
    else if (typeof val === 'string' && (val.startsWith('#') || val.startsWith('rgba'))) {
      tokenColorSet.add(normalizeColor(val));
    }
    else if (typeof val === 'string' && val.includes('rgba') && val.includes('blur')) {
      tokenShadowSet.add(val);
    }
  }
  for (const key in obj) {
    if (typeof obj[key] === 'object') collectTokenValues(obj[key]);
  }
}

// Collect from shared AND each platform independently (avoid merge overwrite)
collectTokenValues(tokens.shared || {});
collectTokenValues(tokens.teacher || {});
collectTokenValues(tokens.parent || {});
collectTokenValues(tokens.admin || {});

// ============================================================
// Color normalization
// ============================================================

function rgbaToHex(r, g, b, a) {
  if (a < 1) return null; // transparent colors not comparable
  const toHex = (n) => n.toString(16).padStart(2, '0');
  return '#' + toHex(r) + toHex(g) + toHex(b);
}

function parseRgba(str) {
  const m = str.match(/rgba?\((\d+),\s*(\d+),\s*(\d+)(?:,\s*([\d.]+))?\)/);
  if (!m) return null;
  return {
    r: parseInt(m[1]), g: parseInt(m[2]), b: parseInt(m[3]),
    a: m[4] !== undefined ? parseFloat(m[4]) : 1,
  };
}

function normalizeColor(str) {
  if (!str) return null;
  const trimmed = str.trim();

  // Already hex
  if (/^#([0-9a-fA-F]{3,8})$/.test(trimmed)) {
    let h = trimmed.toUpperCase();
    if (h.length === 4) h = '#' + h[1]+h[1] + h[2]+h[2] + h[3]+h[3];
    return h;
  }

  // rgba
  const rgba = parseRgba(trimmed);
  if (rgba) return rgbaToHex(rgba.r, rgba.g, rgba.b, rgba.a);

  return trimmed;
}

// ============================================================
// Design data extraction
// ============================================================

function walkLayers(layer, collector, parentLayout) {
  if (!layer) return;

  // Colors from fills
  if (layer.fills) {
    const fills = typeof layer.fills === 'string' ? [layer.fills] :
      Array.isArray(layer.fills) ? layer.fills : [layer.fills];
    for (const f of fills) {
      if (typeof f === 'string') {
        collector.addColor(normalizeColor(f), layer.name, layer.type);
      } else if (f.colorStops) {
        // Gradients
        for (const stop of f.colorStops) {
          collector.addColor(normalizeColor(stop.color), layer.name, layer.type + '(gradient)');
        }
      }
    }
  }

  // Colors from fontFill
  if (layer.fontFill) {
    collector.addColor(normalizeColor(layer.fontFill), layer.name, 'text');
  }

  // Colors from effects (shadows)
  if (layer.effects) {
    for (const eff of layer.effects) {
      if (eff.color) {
        collector.addShadow(eff, layer.name);
      }
    }
  }

  // Font sizes
  if (layer.fontSize) {
    collector.addFontSize(layer.fontSize, layer.name);
  }

  // Border radii
  if (layer.cornerRadius !== undefined && layer.cornerRadius > 0) {
    collector.addRadius(layer.cornerRadius, layer.name);
  }

  // Spacing (padding)
  if (layer.padding) {
    const pads = Array.isArray(layer.padding) ? layer.padding : [layer.padding];
    for (const p of pads) {
      if (p > 0) collector.addSpacing(p, layer.name);
    }
  }

  // Gap from "spacer" elements (only small gaps < 100px are tokenizable)
  if (layer.type === 'rectangle' && layer.name === 'spacer') {
    if (layer.width > 0 && layer.width < 100) collector.addSpacing(layer.width, 'spacer');
    if (layer.height > 0 && layer.height < 100) collector.addSpacing(layer.height, 'spacer');
  }

  // Children
  if (layer.children) {
    for (const child of layer.children) {
      walkLayers(child, collector, layer.layout || parentLayout);
    }
  }
}

class ValueCollector {
  constructor() {
    this.colors = new Map();      // normalized hex -> { count, sources[], types[] }
    this.fontSizes = new Map();   // px value -> { count, sources[] }
    this.radii = new Map();       // px value -> { count, sources[] }
    this.spacings = new Map();    // px value -> { count, sources[] }
    this.shadows = new Map();     // key -> { count, sources[] }
  }

  addColor(color, source, type) {
    if (!color) return;
    // Normalize to uppercase for case-insensitive matching
    const normalized = color.toUpperCase();
    if (!this.colors.has(normalized)) this.colors.set(normalized, { count: 0, sources: new Set(), types: new Set() });
    const entry = this.colors.get(normalized);
    entry.count++;
    entry.sources.add(source);
    entry.types.add(type);
  }

  addFontSize(size, source) {
    if (!this.fontSizes.has(size)) this.fontSizes.set(size, { count: 0, sources: new Set() });
    const entry = this.fontSizes.get(size);
    entry.count++;
    entry.sources.add(source);
  }

  addRadius(radius, source) {
    if (!this.radii.has(radius)) this.radii.set(radius, { count: 0, sources: new Set() });
    const entry = this.radii.get(radius);
    entry.count++;
    entry.sources.add(source);
  }

  addSpacing(value, source) {
    if (!this.spacings.has(value)) this.spacings.set(value, { count: 0, sources: new Set() });
    const entry = this.spacings.get(value);
    entry.count++;
    entry.sources.add(source);
  }

  addShadow(effect, source) {
    const key = `${effect.offsetX},${effect.offsetY},${effect.blur},${effect.spread || 0},${normalizeColor(effect.color)}`;
    if (!this.shadows.has(key)) this.shadows.set(key, { count: 0, effect, sources: new Set() });
    const entry = this.shadows.get(key);
    entry.count++;
    entry.sources.add(source);
  }
}

// ============================================================
// Token matching
// ============================================================

function nearestToken(value, tokenSet, name) {
  // Exact match
  if (tokenSet.has(value)) return { match: 'exact', value };
  // Try value * 2 (px to rpx conversion for mini-programs)
  if (tokenSet.has(value * 2)) return { match: 'exact', value: value * 2, actual: value };

  // Find nearest for approximate suggestions
  let nearest = null, minDist = Infinity;
  for (const tv of tokenSet) {
    const dist = Math.abs(tv - value);
    if (dist < minDist) { minDist = dist; nearest = tv; }
  }
  return { match: 'mismatch', value, nearest, distance: minDist };
}

// ============================================================
// Main audit
// ============================================================

function main() {
  const flag = process.argv[2];
  const summaryOnly = flag === '--summary';
  const colorsOnly = flag === '--colors';

  if (!existsSync(DESIGN_DATA_DIR)) {
    console.error('design-data/ directory not found. Run Calicat get_design_data first.');
    process.exit(1);
  }

  const files = readdirSync(DESIGN_DATA_DIR).filter(f => f.endsWith('.json'));
  if (files.length === 0) {
    console.error('No JSON files in design-data/.');
    process.exit(1);
  }

  console.log('🔍 Calicat Design Token Audit');
  console.log(`   Scanning ${files.length} design files...\n`);

  const collector = new ValueCollector();

  for (const file of files) {
    const content = readFileSync(join(DESIGN_DATA_DIR, file), 'utf-8');
    let data;
    try { data = JSON.parse(content); } catch { continue; }

    const layers = data.result || (Array.isArray(data) ? data : [data]);
    for (const item of layers) {
      if (item.layerData) walkLayers(item.layerData, collector);
      else walkLayers(item, collector);
    }
  }

  // ==============================
  // Colors
  // ==============================
  const showColors = !summaryOnly || colorsOnly;
  if (showColors) {
    console.log('═══ Colors ═══');
    const colorEntries = [...collector.colors.entries()]
      .sort((a, b) => b[1].count - a[1].count);

    let colorMatches = 0, colorMismatches = 0, colorNoToken = 0;

    for (const [color, info] of colorEntries) {
      const inTokens = tokenColorSet.has(color);
      const status = inTokens ? '✅' : '⚠️';

      if (inTokens) colorMatches++;
      else colorNoToken++;

      console.log(`  ${status} ${color} (${info.count}×) — ${[...info.sources].slice(0, 3).join(', ')}${info.sources.size > 3 ? ` +${info.sources.size - 3} more` : ''}`);
    }

    console.log(`\n  Matched: ${colorMatches}  |  Not in tokens: ${colorNoToken}\n`);
  }

  // ==============================
  // Font Sizes
  // ==============================
  if (!colorsOnly) {
    console.log('═══ Font Sizes (= design px, → rpx = ×2) ═══');
    const fsEntries = [...collector.fontSizes.entries()]
      .sort((a, b) => a[0] - b[0]);

    for (const [size, info] of fsEntries) {
      const rpx = size * 2;
      const { match, nearest: n } = nearestToken(size, tokenFontSizeSet);
      const status = match === 'exact' ? '✅' : '⚠️';

      console.log(`  ${status} ${size}px → ${rpx}rpx (${info.count}×)` +
        (match !== 'exact' && n !== undefined
          ? ` — closest token: ${n} (${n * 2}rpx)`
          : '') +
        ` — ${[...info.sources].slice(0, 2).join(', ')}`);
    }
    console.log('');

    // ==============================
    // Border Radii
    // ==============================
    console.log('═══ Border Radii (= design px, → rpx = ×2) ═══');
    const radEntries = [...collector.radii.entries()]
      .sort((a, b) => a[0] - b[0]);

    for (const [radius, info] of radEntries) {
      const rpx = radius * 2;
      const { match, nearest: n } = nearestToken(radius, tokenRadiusSet);
      const status = match === 'exact' ? '✅' : '⚠️';

      console.log(`  ${status} ${radius}px → ${rpx}rpx (${info.count}×)` +
        (match !== 'exact' && n !== undefined
          ? ` — closest token: ${n} (${n * 2}rpx)`
          : '') +
        ` — ${[...info.sources].slice(0, 2).join(', ')}`);
    }
    console.log('');

    // ==============================
    // Spacing
    // ==============================
    console.log('═══ Spacing / Gap (design px, → rpx = ×2) ═══');
    const spEntries = [...collector.spacings.entries()]
      .sort((a, b) => a[0] - b[0]);

    for (const [spacing, info] of spEntries) {
      const rpx = spacing * 2;
      const { match, nearest: n } = nearestToken(spacing, tokenSpacingSet);
      const status = match === 'exact' ? '✅' : '⚠️';

      console.log(`  ${status} ${spacing}px → ${rpx}rpx (${info.count}×)` +
        (match !== 'exact' && n !== undefined
          ? ` — closest token: ${n} (${n * 2}rpx)`
          : '') +
        ` — ${[...info.sources].slice(0, 2).join(', ')}`);
    }
    console.log('');

    // ==============================
    // Shadows
    // ==============================
    console.log('═══ Shadows ═══');
    if (collector.shadows.size === 0) {
      console.log('  (none found in design files)\n');
    } else {
      for (const [key, info] of collector.shadows) {
        const eff = info.effect;
        const shadowStr = `${eff.offsetX}px ${eff.offsetY}px ${eff.blur}px ${eff.color}`;
        console.log(`  📐 ${shadowStr} (${info.count}×) — ${[...info.sources].slice(0, 2).join(', ')}`);
      }
      console.log('');
    }
  }

  // ==============================
  // Summary
  // ==============================
  console.log('═══ Summary ═══');
  const totalColors = collector.colors.size;
  const matchedColors = [...collector.colors.keys()].filter(c => tokenColorSet.has(c)).length;

  console.log(`  Colors:     ${matchedColors}/${totalColors} matched in tokens`);
  console.log(`  Font sizes: ${collector.fontSizes.size} unique values`);
  console.log(`  Border radii: ${collector.radii.size} unique values`);
  console.log(`  Spacings:   ${collector.spacings.size} unique values`);
  console.log(`  Shadows:    ${collector.shadows.size} unique values`);

  if (matchedColors < totalColors) {
    const unmatched = [...collector.colors.keys()].filter(c => !tokenColorSet.has(c));
    console.log(`\n  ⚠️  Colors NOT in tokens: ${unmatched.join(', ')}`);
  }

  console.log('');
}

main();
