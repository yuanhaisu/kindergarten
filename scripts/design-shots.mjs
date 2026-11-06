#!/usr/bin/env node
/**
 * Re-shoot every prototype and record a content-based screenshot baseline.
 *
 * mtime is unreliable across git checkouts, so the freshness gate compares a
 * hash of the prototype assets instead: if index.html / styles.css / app.js /
 * notes.md / tokens.css change, design-sync check warns until this script runs.
 */
import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { BATCH, prototypeAssetsSha } from './design-sync.mjs';

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const PREVIEW_LOCK = path.join(BATCH, 'design.preview.lock.json');

function sha256Buffer(buffer) {
  return crypto.createHash('sha256').update(buffer).digest('hex');
}

const pageArg = process.argv.indexOf('--page');
const only = pageArg > -1
  ? new Set(String(process.argv[pageArg + 1] || '').split(',').map((item) => item.trim()).filter(Boolean))
  : null;

const shotArgs = [
  path.join(repoRoot, 'scripts/od-shots.mjs'),
  BATCH,
  '--width', '390',
  '--dsf', '2',
];
if (only) shotArgs.push('--page', [...only].join(','));
execFileSync(process.execPath, shotArgs, { cwd: repoRoot, stdio: 'inherit' });

const pages = only && fs.existsSync(PREVIEW_LOCK)
  ? { ...JSON.parse(fs.readFileSync(PREVIEW_LOCK, 'utf8')).pages }
  : {};
for (const surface of ['teacher', 'parent']) {
  const root = path.join(BATCH, surface);
  if (!fs.existsSync(root)) continue;
  for (const slug of fs.readdirSync(root).sort()) {
    if (only && !only.has(slug)) continue;
    const dir = path.join(root, slug);
    if (!fs.statSync(dir).isDirectory()) continue;
    const preview = path.join(dir, 'preview.png');
    if (!fs.existsSync(path.join(dir, 'index.html')) || !fs.existsSync(preview)) continue;
    pages[`${surface}/${slug}`] = {
      assetsSha: prototypeAssetsSha({ surface, slug }),
      previewSha: sha256Buffer(fs.readFileSync(preview)),
    };
  }
}

const sorted = Object.fromEntries(Object.entries(pages).sort(([a], [b]) => a.localeCompare(b)));
fs.writeFileSync(PREVIEW_LOCK, JSON.stringify({ version: 1, pages: sorted }, null, 2) + '\n');
console.log(`[design-shots] 基线已写入 ${path.relative(repoRoot, PREVIEW_LOCK)}（${Object.keys(sorted).length} 页）`);
