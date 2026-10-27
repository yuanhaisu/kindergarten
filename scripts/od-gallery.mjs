#!/usr/bin/env node
/**
 * Build a review gallery for a design-preview batch.
 *
 *   node scripts/od-gallery.mjs docs/design-previews/2026-10-25-full-redesign
 *
 * Scans <batch>/<surface>/<NN-slug>/ for index.html / preview.png / notes.md /
 * _run.json and writes <batch>/index.html + <batch>/README.md.
 */
import fs from 'node:fs';
import path from 'node:path';

const batch = process.argv[2];
if (!batch) {
  console.error('usage: node scripts/od-gallery.mjs <batch-dir>');
  process.exit(2);
}
const root = path.resolve(batch);
const surfaces = [
  { id: 'teacher', label: '老师端', hint: '无 tabBar · 暖奶白底' },
  { id: 'parent', label: '家长端', hint: '底部 tabBar · 冷雾白底' },
];

function esc(value) {
  return String(value).replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]));
}

function readPages(surface) {
  const dir = path.join(root, surface);
  if (!fs.existsSync(dir)) return [];
  return fs
    .readdirSync(dir, { withFileTypes: true })
    .filter((e) => e.isDirectory())
    .map((e) => e.name)
    .sort()
    .map((name) => {
      const dirPath = path.join(dir, name);
      const runPath = path.join(dirPath, '_run.json');
      let run = null;
      try {
        run = JSON.parse(fs.readFileSync(runPath, 'utf8'));
      } catch {}
      const files = fs.readdirSync(dirPath);
      return {
        slug: name,
        surface,
        title: run?.title || name.replace(/^\d+-/, ''),
        html: files.includes('index.html'),
        png: files.includes('preview.png'),
        notes: files.includes('notes.md'),
        status: run?.status || (files.includes('index.html') ? 'unknown' : 'missing'),
        elapsedSec: run?.elapsedSec ?? null,
        rel: `${surface}/${name}`,
      };
    });
}

const pages = surfaces.flatMap((s) => readPages(s.id));
const total = pages.length;
const ready = pages.filter((p) => p.html && p.png).length;

const card = (p) => `
    <article class="card" data-surface="${p.surface}" data-state="${p.html ? 'ok' : 'missing'}">
      <a class="shot" href="${p.rel}/index.html" target="_blank" rel="noreferrer">
        ${p.png
          ? `<img src="${p.rel}/preview.png" alt="${esc(p.title)} 预览" loading="lazy" />`
          : `<span class="shot__missing">未生成预览</span>`}
      </a>
      <div class="body">
        <div class="meta">
          <span class="badge badge--${p.surface}">${p.surface === 'teacher' ? '老师' : '家长'}</span>
          <span class="badge badge--${p.html ? 'ok' : 'bad'}">${p.html ? '已出稿' : '缺产物'}</span>
          ${p.elapsedSec ? `<span class="time">${Math.round(p.elapsedSec / 60)} 分</span>` : ''}
        </div>
        <h3>${esc(p.title)}</h3>
        <p class="slug">${esc(p.slug)}</p>
        <div class="links">
          <a href="${p.rel}/index.html" target="_blank" rel="noreferrer">原型</a>
          ${p.notes ? `<a href="${p.rel}/notes.md">说明</a>` : ''}
          ${p.png ? `<a href="${p.rel}/preview.png" target="_blank" rel="noreferrer">长图</a>` : ''}
        </div>
      </div>
    </article>`;

const section = (s) => {
  const list = pages.filter((p) => p.surface === s.id);
  return `
  <section class="group" data-group="${s.id}">
    <header class="group__head">
      <h2>${s.label}</h2>
      <span>${list.length} 页 · ${s.hint}</span>
    </header>
    <div class="grid">${list.map(card).join('')}
    </div>
  </section>`;
};

const html = `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<title>养成星球 · 全量页面重设计（2026-10-25）</title>
<style>
  :root {
    --bg: #f7f3ea; --surface: #fff; --fg: #1e293b; --fg-2: #475569; --muted: #64748b;
    --accent: #f97316; --accent-wash: #fff1e6; --border: #e7e0d4;
    --teacher: #ff8a3d; --parent: #f97316; --ok: #16a34a; --bad: #dc2626;
    --radius: 16px; --shadow: 0 8px 24px rgba(178, 122, 52, .12);
  }
  * { box-sizing: border-box; }
  body {
    margin: 0; background: var(--bg); color: var(--fg);
    font: 14px/1.6 "Noto Sans SC", system-ui, -apple-system, "Segoe UI", sans-serif;
  }
  .wrap { max-width: 1180px; margin: 0 auto; padding: 32px 20px 64px; }
  .hero {
    background: var(--surface); border-radius: 24px; padding: 28px 28px 24px;
    box-shadow: var(--shadow); display: flex; flex-wrap: wrap; gap: 20px;
    align-items: flex-end; justify-content: space-between;
  }
  .hero h1 { margin: 0 0 6px; font-size: 24px; letter-spacing: .2px; }
  .hero p { margin: 0; color: var(--muted); }
  .stats { display: flex; gap: 24px; }
  .stat b { display: block; font-size: 26px; line-height: 1.2; }
  .stat span { color: var(--muted); font-size: 12px; }
  .toolbar { display: flex; gap: 8px; margin: 24px 0 4px; flex-wrap: wrap; }
  .toolbar button {
    border: 1px solid var(--border); background: var(--surface); color: var(--fg-2);
    border-radius: 999px; padding: 8px 16px; font: inherit; cursor: pointer;
  }
  .toolbar button[aria-pressed="true"] { background: var(--accent-wash); border-color: var(--accent); color: #9a3412; }
  .group { margin-top: 28px; }
  .group__head { display: flex; align-items: baseline; gap: 12px; margin-bottom: 14px; }
  .group__head h2 { margin: 0; font-size: 18px; }
  .group__head span { color: var(--muted); font-size: 12px; }
  .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(230px, 1fr)); gap: 18px; }
  .card {
    background: var(--surface); border-radius: var(--radius); overflow: hidden;
    box-shadow: var(--shadow); display: flex; flex-direction: column;
  }
  .shot {
    display: block; aspect-ratio: 390 / 560; background: #f1ece2; overflow: hidden;
    border-bottom: 1px solid var(--border);
  }
  .shot img { width: 100%; height: 100%; object-fit: cover; object-position: top; display: block; }
  .shot__missing { display: grid; place-items: center; height: 100%; color: var(--muted); font-size: 12px; }
  .body { padding: 12px 14px 14px; }
  .meta { display: flex; align-items: center; gap: 6px; margin-bottom: 6px; }
  .badge { font-size: 11px; padding: 2px 8px; border-radius: 999px; }
  .badge--teacher { background: #fff2e6; color: #b45309; }
  .badge--parent { background: #ffecd9; color: #c2410c; }
  .badge--ok { background: #e8f7ee; color: var(--ok); }
  .badge--bad { background: #fdeaea; color: var(--bad); }
  .time { margin-left: auto; color: var(--muted); font-size: 11px; }
  .card h3 { margin: 0; font-size: 15px; }
  .slug { margin: 2px 0 10px; color: var(--muted); font-size: 11px; font-family: ui-monospace, monospace; }
  .links { display: flex; gap: 12px; }
  .links a { color: var(--accent); text-decoration: none; font-size: 12px; font-weight: 500; }
  .links a:hover { text-decoration: underline; }
  .empty { padding: 40px; text-align: center; color: var(--muted); }
</style>
</head>
<body>
<div class="wrap">
  <header class="hero">
    <div>
      <h1>养成星球 · 全量页面重设计</h1>
      <p>老师端 + 家长端全部页面 · 微信小程序风格 · 每页独立 HTML 原型</p>
      <p style="margin-top:8px"><a href="_shared/themes.html" style="color:var(--accent-deep);font-weight:600;text-decoration:none">查看三套配色方案对比（亮橙 / 深橙 AA / 亮橙+深墨字）→</a></p>
    </div>
    <div class="stats">
      <div class="stat"><b>${total}</b><span>页面</span></div>
      <div class="stat"><b>${ready}</b><span>已出稿</span></div>
      <div class="stat"><b>${total - ready}</b><span>待补</span></div>
    </div>
  </header>

  <div class="toolbar" role="group" aria-label="按端筛选">
    <button type="button" data-filter="all" aria-pressed="true">全部</button>
    <button type="button" data-filter="teacher" aria-pressed="false">老师端</button>
    <button type="button" data-filter="parent" aria-pressed="false">家长端</button>
  </div>
${surfaces.map(section).join('\n')}

  <p class="empty" id="empty" hidden>该端还没有页面产物。</p>
</div>

<script>
  const buttons = [...document.querySelectorAll('[data-filter]')];
  const groups = [...document.querySelectorAll('[data-group]')];
  buttons.forEach((button) => {
    button.addEventListener('click', () => {
      const filter = button.dataset.filter;
      buttons.forEach((b) => b.setAttribute('aria-pressed', String(b === button)));
      groups.forEach((g) => { g.hidden = filter !== 'all' && g.dataset.group !== filter; });
    });
  });
</script>
</body>
</html>
`;

fs.writeFileSync(path.join(root, 'index.html'), html, 'utf8');

const table = (surface) => {
  const list = pages.filter((p) => p.surface === surface);
  return [
    `### ${surface === 'teacher' ? '老师端' : '家长端'}（${list.length}）`,
    '',
    '| # | 页面 | slug | 原型 | 说明 |',
    '|---|---|---|---|---|',
    ...list.map((p, i) => `| ${i + 1} | ${p.title} | \`${p.slug}\` | ${p.html ? `[打开](${p.rel}/index.html)` : '—'} | ${p.notes ? `[notes](${p.rel}/notes.md)` : '—'} |`),
  ].join('\n');
};

const readme = `# 全量页面重设计（2026-10-25）

老师端 + 家长端所有页面，每页一份独立 HTML 原型。先看 [index.html](index.html) 画廊。

- 页面总数：${total}
- 已出稿：${ready}
- 共享约束：[_shared/brief.md](_shared/brief.md)
- 设计系统：\`user:nurture-planet-local\`

${table('teacher')}

${table('parent')}
`;

fs.writeFileSync(path.join(root, 'README.md'), readme, 'utf8');
console.log(`[gallery] wrote ${path.join(root, 'index.html')} (${ready}/${total} ready)`);
