#!/usr/bin/env node
/**
 * Re-shoot every prototype and record a content-based screenshot baseline.
 *
 * mtime is unreliable across git checkouts, so the freshness gate compares a
 * hash of the prototype assets instead: if the page files or the shared shell /
 * theme assets change, design-sync check warns until this script runs.
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

function fail(message) {
  console.error(`[design-shots] ${message}`);
  process.exit(1);
}

const pageArg = process.argv.indexOf('--page');
const only = pageArg > -1
  ? new Set(String(process.argv[pageArg + 1] || '').split(',').map((item) => item.trim()).filter(Boolean))
  : null;
if (pageArg > -1 && !only.size) fail('--page 需要至少一个 slug');
if (only && !fs.existsSync(PREVIEW_LOCK)) fail('--page 需要先跑一次全量 design:shots 建立基线');

const shotArgs = [
  path.join(repoRoot, 'scripts/od-shots.mjs'),
  BATCH,
  '--width', '390',
  '--dsf', '2',
];
if (only) shotArgs.push('--page', [...only].join(','));

let stdout;
try {
  stdout = execFileSync(process.execPath, shotArgs, { cwd: repoRoot, encoding: 'utf8' });
} catch (error) {
  if (error.stdout) process.stdout.write(error.stdout);
  fail(`od-shots 失败：${error.message}`);
}

const lines = String(stdout).trim().split('\n').filter(Boolean);
let summary;
try {
  summary = JSON.parse(lines.at(-1));
} catch {
  fail('无法解析 od-shots 汇总输出');
}
if (!summary || summary.failed > 0) fail(`${summary?.failed ?? '?'} 页截图失败`);
if (!summary.total) fail('od-shots 没有生成任何截图');
if (only && summary.total < only.size) {
  fail(`--page 只命中 ${summary.total}/${only.size} 页，检查 slug`);
}
console.log(`[design-shots] od-shots: ${summary.total} 页，失败 ${summary.failed}`);

const pages = only
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

if (!Object.keys(pages).length) fail('没有可写入基线的页面');
const sorted = Object.fromEntries(Object.entries(pages).sort(([a], [b]) => a.localeCompare(b)));
fs.writeFileSync(PREVIEW_LOCK, JSON.stringify({ version: 1, pages: sorted }, null, 2) + '\n');
console.log(`[design-shots] 基线已写入 ${path.relative(repoRoot, PREVIEW_LOCK)}（${Object.keys(sorted).length} 页）`);
