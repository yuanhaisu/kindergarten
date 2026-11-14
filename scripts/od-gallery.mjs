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

// Optional <batch>/gallery.json lets a batch carry its own title and palette.
// Every key falls back to the 2026-10-25 orange batch, so existing galleries
// regenerate byte-for-byte the same way.
const defaults = {
  title: '养成星球 · 全量页面重设计（2026-10-25）',
  heroTitle: '养成星球 · 全量页面重设计',
  subtitle: '老师端 + 家长端全部页面 · 微信小程序风格 · 每页独立 HTML 原型',
  paletteLink: '_shared/themes.html',
  paletteLabel: '查看三套配色方案对比（亮橙 / 深橙 AA / 亮橙+深墨字）→',
  themeNote: '',
  themeFollowup: '',
  bg: '#f7f3ea',
  surface: '#fff',
  fg: '#1e293b',
  fg2: '#475569',
  muted: '#64748b',
  accent: '#f97316',
  accentDeep: '#9a3412',
  accentWash: '#fff1e6',
  border: '#e7e0d4',
  teacher: '#ff8a3d',
  parent: '#f97316',
  ok: '#16a34a',
  bad: '#dc2626',
  shotBg: '#f1ece2',
  shadow: '0 8px 24px rgba(178, 122, 52, .12)',
  teacherBadgeBg: '#fff2e6',
  teacherBadgeFg: '#b45309',
  parentBadgeBg: '#ffecd9',
  parentBadgeFg: '#c2410c',
  teacherHint: '无 tabBar · 暖奶白底',
  parentHint: '底部 tabBar · 冷雾白底',
  readmeTitle: '全量页面重设计（2026-10-25）',
  readmeLead: '老师端 + 家长端所有页面，每页一份独立 HTML 原型。先看 [index.html](index.html) 画廊。',
  readmeShared: '- 共享约束：[_shared/brief.md](_shared/brief.md)',
  designSystem: 'user:nurture-planet-local',
};
let galleryMeta = {};
try {
  galleryMeta = JSON.parse(fs.readFileSync(path.join(root, 'gallery.json'), 'utf8'));
} catch {}
const M = { ...defaults, ...galleryMeta };

const surfaces = [
  { id: 'teacher', label: '老师端', hint: M.teacherHint },
  { id: 'parent', label: '家长端', hint: M.parentHint },
];

function esc(value) {
  return String(value).replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]));
}

// Optional illustration atlas: gallery.json may declare one or more groups.
// Each group renders as a section between the toolbar and the page grids.
const illustrationGroups = Array.isArray(galleryMeta.illustrations) ? galleryMeta.illustrations : [];

const renderIllustrations = () => illustrationGroups.map((group, index) => `
  <section class="pp-art-group" aria-labelledby="art-group-${index}">
    <header class="group__head">
      <h2 id="art-group-${index}">${esc(group.title)}</h2>
      <span>${esc(group.note || '')}</span>
    </header>
    <div class="pp-art-grid pp-art-grid--${group.layout === 'bands' ? 'bands' : 'scenes'}">
      ${(group.items || []).map((item) => `
      <figure class="pp-art-card">
        <img src="${esc(item.src)}" alt="${esc(item.title)}" loading="lazy" />
        <figcaption>
          <b>${esc(item.title)}</b>
          ${item.guest ? `<span>${esc(item.guest)}</span>` : ''}
          ${item.quote ? `<em>「${esc(item.quote)}」</em>` : ''}
        </figcaption>
      </figure>`).join('')}
    </div>
  </section>`).join('');

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
<title>${M.title}</title>
<style>
  :root {
    --bg: ${M.bg}; --surface: ${M.surface}; --fg: ${M.fg}; --fg-2: ${M.fg2}; --muted: ${M.muted};
    --accent: ${M.accent}; --accent-deep: ${M.accentDeep}; --accent-wash: ${M.accentWash}; --border: ${M.border};
    --teacher: ${M.teacher}; --parent: ${M.parent}; --ok: ${M.ok}; --bad: ${M.bad};
    --radius: 16px; --shadow: ${M.shadow};
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
  .hero__art { width: 300px; max-width: 100%; height: auto; border-radius: 14px; border: 1px solid var(--border); background: #fbfaf6; }
  .hero--art .hero__copy { flex: 1 1 320px; min-width: 0; }
  .pp-art-group { margin-top: 34px; }
  .pp-art-grid { display: grid; gap: 16px; }
  .pp-art-grid--scenes { grid-template-columns: repeat(auto-fill, minmax(170px, 1fr)); }
  .pp-art-grid--bands { grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); }
  .pp-art-card { margin: 0; background: var(--surface); border: 1px solid var(--border); border-radius: 14px; overflow: hidden; }
  .pp-art-card img { display: block; width: 100%; height: auto; background: #fbfaf6; }
  .pp-art-card figcaption { padding: 10px 12px 12px; display: flex; flex-direction: column; gap: 2px; }
  .pp-art-card b { font-size: 13px; }
  .pp-art-card span { color: var(--muted); font-size: 11px; }
  .pp-art-card em { color: var(--accent-deep); font-style: normal; font-size: 12px; }
  .stats { display: flex; gap: 24px; }
  .stat b { display: block; font-size: 26px; line-height: 1.2; }
  .stat span { color: var(--muted); font-size: 12px; }
  .toolbar { display: flex; gap: 8px; margin: 24px 0 4px; flex-wrap: wrap; }
  .toolbar button {
    border: 1px solid var(--border); background: var(--surface); color: var(--fg-2);
    border-radius: 999px; padding: 8px 16px; font: inherit; cursor: pointer;
  }
  .toolbar button[aria-pressed="true"] { background: var(--accent-wash); border-color: var(--accent); color: ${M.accentDeep}; }
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
    display: block; aspect-ratio: 390 / 560; background: ${M.shotBg}; overflow: hidden;
    border-bottom: 1px solid var(--border);
  }
  .shot img { width: 100%; height: 100%; object-fit: cover; object-position: top; display: block; }
  .shot__missing { display: grid; place-items: center; height: 100%; color: var(--muted); font-size: 12px; }
  .body { padding: 12px 14px 14px; }
  .meta { display: flex; align-items: center; gap: 6px; margin-bottom: 6px; }
  .badge { font-size: 11px; padding: 2px 8px; border-radius: 999px; }
  .badge--teacher { background: ${M.teacherBadgeBg}; color: ${M.teacherBadgeFg}; }
  .badge--parent { background: ${M.parentBadgeBg}; color: ${M.parentBadgeFg}; }
  .badge--ok { background: #e8f7ee; color: var(--ok); }
  .badge--bad { background: #fdeaea; color: var(--bad); }
  .time { margin-left: auto; color: var(--muted); font-size: 11px; }
  .card h3 { margin: 0; font-size: 15px; }
  .slug { margin: 2px 0 10px; color: var(--muted); font-size: 11px; font-family: ui-monospace, monospace; }
  .links { display: flex; gap: 12px; }
  .links a { color: var(--accent); text-decoration: none; font-size: 12px; font-weight: 500; }
  .links a:hover { text-decoration: underline; }
  .empty { padding: 40px; text-align: center; color: var(--muted); }
  footer { margin-top: 34px; padding-top: 18px; border-top: 1px solid var(--border); color: var(--muted); font-size: 12px; line-height: 1.9; }
  footer p { margin: 0; }
  footer code { background: var(--accent-wash); padding: 1px 5px; border-radius: 5px; color: var(--accent-deep); }
</style>
</head>
<body>
<div class="wrap">
  <header class="hero${M.heroImage ? ' hero--art' : ''}">
    <div class="hero__copy">
      <h1>${M.heroTitle}</h1>
      <p>${M.subtitle}</p>
      <p style="margin-top:8px"><a href="${M.paletteLink}" style="color:var(--accent-deep);font-weight:600;text-decoration:none">${M.paletteLabel}</a></p>
    </div>
    ${M.heroImage ? `<img class="hero__art" src="${esc(M.heroImage)}" alt="" />` : ''}
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
${illustrationGroups.length ? renderIllustrations() : ''}
${surfaces.map(section).join('\n')}

  <p class="empty" id="empty" hidden>该端还没有页面产物。</p>
${M.themeNote || M.themeFollowup ? `  <footer>
    ${M.themeNote ? `<p>${M.themeNote}</p>` : ''}
    ${M.themeFollowup ? `<p>${M.themeFollowup}</p>` : ''}
  </footer>` : ''}
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

const artReadme = illustrationGroups.map((group) => {
  const rows = (group.items || [])
    .map((item, i) => `| ${i + 1} | ${item.title} | ${item.guest || '—'} | ${item.quote ? `「${item.quote}」` : '—'} | [${item.src}](${item.src}) |`)
    .join('\n');
  const guestLabel = group.layout === 'bands' ? '使用位置' : '客人';
  return [
    `## ${group.title}`,
    '',
    group.note ? `${group.note}\n` : '',
    `| # | 画面 | ${guestLabel} | 金句 | 文件 |`,
    '|---|---|---|---|---|',
    rows,
  ].join('\n');
}).join('\n\n');

const readme = `# ${M.readmeTitle}

${M.readmeLead}

- 页面总数：${total}
- 已出稿：${ready}
${M.readmeShared}
- 设计系统：\`${M.designSystem}\`
${artReadme ? `\n${artReadme}\n` : ''}
${table('teacher')}

${table('parent')}
`;

fs.writeFileSync(path.join(root, 'README.md'), readme, 'utf8');
console.log(`[gallery] wrote ${path.join(root, 'index.html')} (${ready}/${total} ready)`);
