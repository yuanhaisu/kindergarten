import test from 'node:test';
import assert from 'node:assert/strict';
import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import {
  BATCH,
  buildManifest,
  manifestText,
  sha256File,
  prototypeTitle,
  codeTabbarRoutes,
  loadLock,
  loadPreviewLock,
  prototypeAssetsSha,
  previewIsStale,
  staleReasons,
  depsSha,
} from './design-sync.mjs';

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const MANIFEST = path.join(BATCH, '_shared/pages.json');
const GALLERY = path.join(BATCH, 'index.html');

const entries = buildManifest();
const routeOf = (entry) => entry.path.replace(`${entry.surface}/src/`, '').replace(/\.vue$/, '');

test('manifest matches the committed pages.json', () => {
  const committed = JSON.parse(fs.readFileSync(MANIFEST, 'utf8'));
  assert.deepEqual(committed, entries);
  assert.equal(fs.readFileSync(MANIFEST, 'utf8').replace(/\r\n/g, '\n'), manifestText(entries));
});

test('every page has a lock entry with current source and dependency hashes', () => {
  const lock = loadLock();
  for (const entry of entries) {
    const saved = lock.pages[`${entry.surface}/${entry.slug}`];
    assert.ok(saved, `missing lock entry: ${entry.surface}/${entry.slug}`);
    assert.equal(saved.source, entry.path);
    assert.equal(saved.sha256, sha256File(path.join(repoRoot, entry.path)));
    assert.equal(saved.depsSha, depsSha(entry));
    assert.equal(saved.title, entry.title);
    assert.equal(saved.tabbar, entry.tabbar);
    assert.ok(saved.reviewedAt);
  }
});

test('prototype navigation titles match pages.json', () => {
  for (const entry of entries) {
    assert.equal(prototypeTitle(entry), entry.title, `${entry.surface}/${entry.slug}`);
  }
});

test('tabBar ownership is derived from code', () => {
  for (const entry of entries) {
    assert.equal(entry.tabbar, codeTabbarRoutes(entry.surface).has(routeOf(entry)));
  }
});

test('gallery links every page', () => {
  const html = fs.readFileSync(GALLERY, 'utf8');
  for (const entry of entries) {
    assert.ok(html.includes(`${entry.surface}/${entry.slug}/`), `gallery missing ${entry.surface}/${entry.slug}`);
  }
});

test('preview screenshots match the current prototype assets', () => {
  const preview = loadPreviewLock();
  for (const entry of entries) {
    const saved = preview.pages[`${entry.surface}/${entry.slug}`];
    assert.ok(saved, `missing preview baseline: ${entry.surface}/${entry.slug}`);
    assert.equal(saved.assetsSha, prototypeAssetsSha(entry), `stale screenshot: ${entry.surface}/${entry.slug}`);
  }
});

test('staleReasons accepts the current state and rejects every drift kind', () => {
  const entry = entries[0];
  const current = {
    source: entry.path,
    sha256: sha256File(path.join(repoRoot, entry.path)),
    depsSha: depsSha(entry),
    title: entry.title,
    tabbar: entry.tabbar,
    reviewedAt: '2026-11-06',
  };
  assert.deepEqual(staleReasons(entry, current), []);
  assert.ok(staleReasons(entry, { ...current, sha256: 'x' }).includes('页面源码变了'));
  assert.ok(staleReasons(entry, { ...current, depsSha: 'x' }).includes('依赖的 utils 变了'));
  assert.ok(staleReasons(entry, { ...current, title: 'x' }).some((reason) => reason.startsWith('导航标题变了')));
  assert.ok(staleReasons(entry, { ...current, tabbar: !entry.tabbar }).includes('tabBar 归属变了'));
  assert.ok(staleReasons(entry, { ...current, reviewedAt: null }).includes('新页面未复核'));
  assert.deepEqual(staleReasons(entry, null), ['未登记']);
});

test('previewIsStale detects both asset and image drift', () => {
  const entry = entries[0];
  const key = `${entry.surface}/${entry.slug}`;
  const previewFile = path.join(BATCH, entry.surface, entry.slug, 'preview.png');
  const current = {
    assetsSha: prototypeAssetsSha(entry),
    previewSha: crypto.createHash('sha256').update(fs.readFileSync(previewFile)).digest('hex'),
  };
  assert.equal(previewIsStale(entry, { pages: { [key]: current } }), false);
  assert.equal(previewIsStale(entry, { pages: { [key]: { ...current, assetsSha: 'x' } } }), true);
  assert.equal(previewIsStale(entry, { pages: { [key]: { ...current, previewSha: 'x' } } }), true);
});
