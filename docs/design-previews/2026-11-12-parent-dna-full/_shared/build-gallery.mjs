#!/usr/bin/env node
/**
 * 家长端全量设计稿画廊生成器。
 *
 *   node _shared/build-gallery.mjs           # 生成 index.html
 *   node _shared/build-gallery.mjs --check   # 只报告缺失的页面/截图
 *
 * 页面截图由 scripts/od-shot.mjs 产出（390×844，dsf 2）。
 */
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");

const groups = [
  {
    title: "核心页面",
    note: "四个 tab，先看这四页的基调",
    pages: [
      { dir: "01-home", title: "首页", tab: true },
      { dir: "11-growth", title: "成长时光", tab: true, variants: [{ file: "empty.html", img: "preview-empty.png", label: "空态" }] },
      { dir: "17-pickup", title: "接送", tab: true, variants: [
        { file: "index.html?state=away", img: "preview-away.png", label: "未入园" },
        { file: "index.html?state=done", img: "preview-done.png", label: "已接走" },
      ] },
      { dir: "31-profile", title: "我的", tab: true },
    ],
  },
  {
    title: "登录与协议",
    note: "首次进入的完整路径",
    pages: [
      { dir: "02-login", title: "登录" },
      { dir: "03-bind", title: "绑定孩子" },
      { dir: "04-consent", title: "同意与协议" },
      { dir: "05-policy", title: "政策全文" },
    ],
  },
  {
    title: "通知与请假",
    note: "列表页含空态；表单含校验错误态",
    pages: [
      { dir: "06-notice-index", title: "通知列表", variants: [{ file: "empty.html", img: "preview-empty.png", label: "空态" }] },
      { dir: "07-notice-detail", title: "通知详情" },
      { dir: "08-leave-index", title: "请假记录", variants: [{ file: "empty.html", img: "preview-empty.png", label: "空态" }] },
      { dir: "09-leave-create", title: "新建请假", variants: [{ file: "index.html?state=error", img: "preview-error.png", label: "校验错误" }] },
      { dir: "10-leave-detail", title: "请假详情" },
    ],
  },
  {
    title: "成长与相册",
    note: "记录、报告与影像",
    pages: [
      { dir: "12-growth-detail", title: "成长详情" },
      { dir: "13-growth-album", title: "成长相册" },
      { dir: "14-growth-report", title: "成长报告" },
      { dir: "15-album-index", title: "班级相册", variants: [{ file: "empty.html", img: "preview-empty.png", label: "空态" }] },
      { dir: "16-album-detail", title: "相册详情" },
    ],
  },
  {
    title: "接送与积分",
    note: "门口场景与激励",
    pages: [
      { dir: "18-pickup-qrcode", title: "接送二维码" },
      { dir: "19-pickup-authorized", title: "可接送人", variants: [{ file: "empty.html", img: "preview-empty.png", label: "空态" }] },
      { dir: "20-pickup-authorized-create", title: "新增授权人" },
      { dir: "24-points", title: "积分与成就" },
    ],
  },
  {
    title: "任务与班级",
    note: "亲子任务、班级墙、班级信息与日历",
    pages: [
      { dir: "21-tasks", title: "亲子任务", variants: [{ file: "empty.html", img: "preview-empty.png", label: "空态" }] },
      { dir: "22-tasks-detail", title: "任务详情" },
      { dir: "23-tasks-wall", title: "班级墙" },
      { dir: "25-class", title: "班级信息" },
      { dir: "26-calendar", title: "班级日历" },
    ],
  },
  {
    title: "档案与设置",
    note: "资料、反馈与消息",
    pages: [
      { dir: "27-student", title: "幼儿信息" },
      { dir: "28-feedback", title: "意见反馈" },
      { dir: "29-settings", title: "设置" },
      { dir: "30-messages", title: "消息", variants: [{ file: "empty.html", img: "preview-empty.png", label: "空态" }] },
    ],
  },
];

const missing = [];
for (const group of groups) {
  for (const page of group.pages) {
    if (!fs.existsSync(path.join(root, page.dir, "index.html"))) missing.push(`${page.dir}/index.html`);
    if (!fs.existsSync(path.join(root, page.dir, "preview.png"))) missing.push(`${page.dir}/preview.png`);
    for (const v of page.variants ?? []) {
      if (!v.file.startsWith("index.html?")) {
        if (!fs.existsSync(path.join(root, page.dir, v.file))) missing.push(`${page.dir}/${v.file}`);
      }
      if (!fs.existsSync(path.join(root, page.dir, v.img))) missing.push(`${page.dir}/${v.img}`);
    }
  }
}

if (process.argv.includes("--check")) {
  if (missing.length) {
    console.log(`缺少 ${missing.length} 个文件：`);
    for (const m of missing) console.log(`  - ${m}`);
    process.exitCode = 1;
  } else {
    console.log("画廊所需文件齐全。");
  }
  process.exit();
}

const total = groups.reduce((n, g) => n + g.pages.length, 0);
const variants = groups.reduce((n, g) => n + g.pages.reduce((m, p) => m + (p.variants?.length ?? 0), 0), 0);

const esc = (s) => s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");

const groupHtml = groups
  .map(
    (group, gi) => `
    <section class="group" id="g${gi + 1}">
      <div class="group-head">
        <span class="group-num">${String(gi + 1).padStart(2, "0")}</span>
        <div class="group-title">
          <h2>${esc(group.title)}</h2>
          <p>${esc(group.note)}</p>
        </div>
        <span class="group-count">${group.pages.length} 页</span>
      </div>
      <div class="cards">
        ${group.pages
          .map((page) => {
            const n = String(
              Number(page.dir.slice(0, 2)),
            ).padStart(2, "0");
            const variantHtml = (page.variants ?? [])
              .map(
                (v) => `<a class="variant" href="${page.dir}/${v.file}"><img src="${page.dir}/${v.img}" alt="${esc(page.title)} · ${esc(v.label)}" loading="lazy" /><span>${esc(v.label)}</span></a>`,
              )
              .join("");
            return `
          <article class="card">
            <a class="shot" href="${page.dir}/index.html" aria-label="${esc(page.title)}">
              <img src="${page.dir}/preview.png" alt="${esc(page.title)}" loading="lazy" />
            </a>
            <div class="meta">
              <span class="idx">${n}</span>
              <h3>${esc(page.title)}</h3>
              <p>${page.tab ? "tab 页" : "二级页"}</p>
            </div>
            ${variantHtml ? `<div class="variants">${variantHtml}</div>` : ""}
          </article>`;
          })
          .join("")}
      </div>
    </section>`,
  )
  .join("");

const html = `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<meta name="color-scheme" content="light" />
<title>养成星球 · 家长端全量设计稿</title>
<style>
  @font-face { font-family: "Instrument Serif"; src: url("./_shared/fonts/instrument-serif-regular.woff2") format("woff2"); font-weight: 400; font-style: normal; font-display: swap; }
  @font-face { font-family: "Noto Serif SC"; src: url("./_shared/fonts/noto-serif-sc-subset.woff2") format("woff2"); font-weight: 400; font-style: normal; font-display: swap; }
  :root {
    --bone: #ece6d7; --bone-cool: #f2edde; --card: #fff; --pine: #173220; --ink: #55616f; --ink-card: #5e6b7a;
    --sage: #8da386; --sage-soft: #cfd9c5; --marigold: #ddb14a; --terracotta: #c36540; --hair: rgba(23,50,32,.12);
    --serif: "Instrument Serif", "Noto Serif SC", "Songti SC", "STSong", "SimSun", Georgia, serif;
    --sans: "Geist", "Inter", -apple-system, "PingFang SC", "Microsoft YaHei", system-ui, sans-serif;
  }
  * { box-sizing: border-box; }
  html, body { margin: 0; background: var(--bone); color: var(--pine); font-family: var(--sans); -webkit-font-smoothing: antialiased; }
  body { padding: 48px 40px 96px; }
  header.top { max-width: 1360px; margin: 0 auto 56px; }
  .eyebrow { margin: 0 0 10px; font-size: 11px; font-weight: 600; letter-spacing: .16em; color: #55616f; }
  h1 { margin: 0; font-family: var(--serif); font-size: 46px; font-weight: 400; line-height: 1.25; letter-spacing: 0; }
  .lede { margin: 14px 0 0; max-width: 620px; font-family: var(--serif); font-size: 17px; line-height: 1.8; color: #3d5141; }
  .facts { display: flex; flex-wrap: wrap; gap: 28px; margin-top: 30px; }
  .fact b { display: block; font-family: var(--serif); font-size: 28px; font-weight: 400; line-height: 1; }
  .fact span { display: block; margin-top: 5px; font-size: 11px; color: var(--ink-card); }
  .swatches { display: flex; gap: 10px; margin-top: 26px; align-items: center; }
  .swatch { width: 30px; height: 30px; border-radius: 50%; border: 1px solid rgba(23,50,32,.12); }
  .swatch-note { margin-left: 6px; font-size: 11px; color: var(--ink-card); }
  nav.toc { max-width: 1360px; margin: 0 auto 24px; display: flex; flex-wrap: wrap; gap: 8px; }
  nav.toc a { display: inline-flex; height: 30px; align-items: center; padding: 0 13px; border: 1px solid rgba(23,50,32,.22); border-radius: 999px; font-size: 12px; color: var(--pine); text-decoration: none; }
  nav.toc a:hover { background: var(--bone-cool); }
  main { max-width: 1360px; margin: 0 auto; }
  .group { margin-top: 64px; }
  .group-head { display: flex; align-items: baseline; gap: 14px; padding-bottom: 14px; border-bottom: 1px solid var(--hair); margin-bottom: 22px; }
  .group-num { font-family: var(--serif); font-size: 15px; color: #728173; }
  .group-title { flex: 1; }
  .group-title h2 { margin: 0; font-family: var(--serif); font-size: 26px; font-weight: 400; line-height: 1.2; }
  .group-title p { margin: 4px 0 0; font-size: 11.5px; color: var(--ink-card); }
  .group-count { font-size: 11px; color: var(--ink-card); }
  .cards { display: grid; grid-template-columns: repeat(auto-fill, minmax(236px, 1fr)); gap: 26px; }
  .card { background: transparent; }
  .shot { display: block; aspect-ratio: 390 / 844; overflow: hidden; border-radius: 26px; border: 1px solid var(--hair); background: #ded6c6; box-shadow: 0 18px 36px -26px rgba(23,50,32,.45); }
  .shot img { display: block; width: 100%; height: 100%; object-fit: cover; object-position: top; }
  .shot:hover img { transform: translateY(-4px); transition: transform .3s cubic-bezier(.2,.8,.2,1); }
  .meta { display: flex; align-items: baseline; gap: 8px; margin-top: 12px; }
  .idx { font-family: var(--serif); font-size: 13px; color: #728173; }
  .meta h3 { margin: 0; flex: 1; font-size: 14px; font-weight: 600; }
  .meta p { margin: 0; font-size: 10.5px; color: var(--ink-card); }
  .variants { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 10px; }
  .variant { display: inline-flex; align-items: center; gap: 6px; padding: 3px 9px 3px 3px; border: 1px solid var(--hair); border-radius: 999px; background: var(--card); font-size: 10.5px; color: var(--pine); text-decoration: none; }
  .variant img { width: 22px; height: 40px; object-fit: cover; object-position: top; border-radius: 9px; }
  footer { max-width: 1360px; margin: 80px auto 0; padding-top: 18px; border-top: 1px solid var(--hair); display: flex; justify-content: space-between; gap: 20px; font-size: 11px; color: var(--ink-card); }
  @media (max-width: 720px) { body { padding: 28px 18px 64px; } h1 { font-size: 34px; } .cards { grid-template-columns: repeat(2, 1fr); gap: 14px; } .group { margin-top: 44px; } }
</style>
</head>
<body>
  <header class="top">
    <p class="eyebrow">养成星球 · 家长端 · DESIGN PREVIEW</p>
    <h1>家长端全量设计稿</h1>
    <p class="lede">DNA v1（2026-11-12 冻结）在 31 个真实页面上的落地。先确认设计，再进入移植；本批次不改产品代码。</p>
    <div class="facts">
      <div class="fact"><b>${total}</b><span>页面</span></div>
      <div class="fact"><b>${variants}</b><span>状态变体</span></div>
      <div class="fact"><b>390</b><span>设计宽度</span></div>
      <div class="fact"><b>v1.0</b><span>DNA 版本</span></div>
    </div>
    <div class="swatches" aria-label="DNA 色板">
      <span class="swatch" style="background:#ece6d7" title="bone"></span>
      <span class="swatch" style="background:#173220" title="pine"></span>
      <span class="swatch" style="background:#8da386" title="sage"></span>
      <span class="swatch" style="background:#ddb14a" title="marigold"></span>
      <span class="swatch" style="background:#c36540" title="terracotta"></span>
      <span class="swatch" style="background:#f2edde" title="bone-cool"></span>
      <span class="swatch-note">暖纸底 · 松绿墨 · 鼠尾草 / 金盏 / 赤陶</span>
    </div>
  </header>

  <nav class="toc" aria-label="分组">
    ${groups.map((g, i) => `<a href="#g${i + 1}">${esc(g.title)}</a>`).join("")}
  </nav>

  <main>${groupHtml}</main>

  <footer>
    <span>来源：docs/design-previews/2026-11-12-parent-dna-full · 规则：docs/design-dna/2026-11-11-kinderly/DESIGN.md</span>
    <span>截图 390×844 · 字体 Instrument Serif + Noto Serif SC</span>
  </footer>
</body>
</html>`;

fs.writeFileSync(path.join(root, "index.html"), html);
console.log(`已生成 ${path.join(root, "index.html")}（${total} 页 / ${variants} 个状态变体）`);
if (missing.length) {
  console.log(`注意：仍有 ${missing.length} 个文件缺失`);
  for (const m of missing) console.log(`  - ${m}`);
  process.exitCode = 1;
}
