#!/usr/bin/env node
/**
 * Export colour variants of the DNA icon masters.
 *
 * Masters live in parent/src/static/icons/dna/*.svg and use currentColor.
 * WXML <image> cannot inherit currentColor, so each icon is exported per tone.
 *   pine  #173220 — light surfaces
 *   muted #728173 — secondary/disabled
 *   bone  #ECE6D7 — only icons that sit on pine fills
 */
import fs from 'node:fs';
import path from 'node:path';

const ROOT = path.resolve(process.cwd(), 'parent/src/static/icons/dna');
const TONES = { pine: '#173220', muted: '#728173', bone: '#ECE6D7' };
const BONE_ICONS = new Set(['check', 'qr']);

const masters = fs
  .readdirSync(ROOT)
  .filter((f) => f.startsWith('icon-') && f.endsWith('.svg'));

let written = 0;
for (const file of masters) {
  const name = file.replace(/\.svg$/, '');
  const source = fs.readFileSync(path.join(ROOT, file), 'utf8');
  for (const [tone, color] of Object.entries(TONES)) {
    if (tone === 'bone' && !BONE_ICONS.has(name.replace(/^icon-/, ''))) continue;
    const dir = path.join(ROOT, tone);
    fs.mkdirSync(dir, { recursive: true });
    fs.writeFileSync(path.join(dir, file), source.replaceAll('currentColor', color));
    written += 1;
  }
}
console.log(`exported ${written} variants from ${masters.length} masters`);
