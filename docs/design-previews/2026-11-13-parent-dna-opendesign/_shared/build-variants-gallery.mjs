#!/usr/bin/env node
/**
 * 8 组变体对照页：baseline vs variant。
 *
 *   node _shared/build-variants-gallery.mjs
 *   node _shared/build-variants-gallery.mjs --check
 */
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");

const groups = [
  {
    id: "v1-leave-day",
    title: "V1 · 11/12 请假在家",
    note: "乐乐发烧请假在家（待审批）；01/17 改请假态，与 baseline 的“在园”并存。",
    pages: [
      { slug: "01-home", title: "首页", baseline: "parent/01-home", label: "请假态" },
      { slug: "17-pickup", title: "接送", baseline: "parent/17-pickup", label: "请假中" },
    ],
  },
  {
    id: "v2-task-points",
    title: "V2 · 任务积分提案",
    note: "恢复 +10/+15 与连续打卡；需 06-task 契约同步，标“未来提案”。",
    pages: [
      { slug: "21-tasks", title: "亲子任务", baseline: "parent/21-tasks", label: "积分提案" },
      { slug: "24-points", title: "积分与成就", baseline: "parent/24-points", label: "任务打卡 +10" },
    ],
  },
  {
    id: "v3-notice-signup",
    title: "V3 · 通知报名提案",
    note: "“我要报名/已报名”；需新增报名 API，标“未来提案”。",
    pages: [{ slug: "07-notice-detail", title: "通知详情", baseline: "parent/07-notice-detail", label: "报名提案" }],
  },
  {
    id: "v4-settings-simple",
    title: "V4 · 设置简化版",
    note: "2 个通用开关 + 免打扰；免打扰需后端补字段。",
    pages: [{ slug: "29-settings", title: "设置", baseline: "parent/29-settings", label: "简化版" }],
  },
  {
    id: "v5-report-month",
    title: "V5 · 报告月度版",
    note: "本月 +180，与 24 的“本月获得”一致。",
    pages: [{ slug: "14-growth-report", title: "成长报告", baseline: "parent/14-growth-report", label: "本月版" }],
  },
  {
    id: "v6-female-child",
    title: "V6 · 女性 / 5 岁数据集",
    note: "27 性别女、5 岁 2 个月、小一班；11/12 的“她”为同数据集说明。",
    pages: [{ slug: "27-student", title: "幼儿信息", baseline: "parent/27-student", label: "女 / 5 岁" }],
  },
  {
    id: "v7-age5-class",
    title: "V7 · 5 岁 / 中班数据集",
    note: "27 5 岁 2 个月、中一班；01/25/31 的班级标签为同数据集说明。",
    pages: [{ slug: "27-student", title: "幼儿信息", baseline: "parent/27-student", label: "5 岁 / 中班" }],
  },
  {
    id: "v8-single-child",
    title: "V8 · 单孩子",
    note: "只显示乐乐，隐藏“切换”；31 的统计改用累计口径。",
    pages: [{ slug: "31-profile", title: "我的", baseline: "parent/31-profile", label: "单孩子" }],
  },
];

const missing = [];
for (const group of groups) {
  for (const page of group.pages) {
    const base = path.join(root, "parent-variants", group.id, page.slug);
    if (!fs.existsSync(path.join(base, "index.html"))) missing.push(`parent-variants/${group.id}/${page.slug}/index.html`);
    if (!fs.existsSync(path.join(base, "preview.png"))) missing.push(`parent-variants/${group.id}/${page.slug}/preview.png`);
    if (!fs.existsSync(path.join(root, page.baseline, "preview.png"))) missing.push(`${page.baseline}/preview.png`);
  }
}

if (process.argv.includes("--check")) {
  if (missing.length) {
    console.log(`缺少 ${missing.length} 个文件：`);
    for (const m of missing) console.log(`  - ${m}`);
    process.exitCode = 1;
  } else {
    console.log("变体画廊所需文件齐全。");
  }
  process.exit();
}

const esc = (s) => s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
const totalPages = groups.reduce((n, g) => n + g.pages.length, 0);

const groupHtml = groups
  .map(
    (group) => `
    <section class="group" id="${group.id}">
      <div class="group-head">
        <h2>${esc(group.title)}</h2>
        <p>${esc(group.note)}</p>
      </div>
      ${group.pages
        .map((page) => {
          const vBase = `parent-variants/${group.id}/${page.slug}`;
          return `
          <article class="pair">
            <div class="pair-head"><h3>${esc(page.title)}</h3><span>${esc(page.label)}</span></div>
            <div class="shots">
              <figure>
                <a class="shot" href="${esc(page.baseline)}/index.html"><img src="${esc(page.baseline)}/preview.png" alt="baseline ${esc(page.title)}" loading="lazy" /></a>
                <figcaption>baseline</figcaption>
              </figure>
              <figure>
                <a class="shot" href="${vBase}/index.html"><img src="${vBase}/preview.png" alt="${esc(group.title)} ${esc(page.title)}" loading="lazy" /></a>
                <figcaption>${esc(group.title)}</figcaption>
              </figure>
            </div>
          </article>`;
        })
        .join("")}
    </section>`,
  )
  .join("");

const html = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <meta name="color-scheme" content="light" />
  <title>家长端 · 8 组变体对照</title>
  <link rel="stylesheet" href="_shared/fonts.css" />
  <style>
    :root { --bg:#ECE6D7; --surface:#fff; --fg:#173220; --muted:#55616F; --muted-card:#5E6B7A; --hair:rgba(23,50,32,.1); --serif:"Instrument Serif","Noto Serif SC","Songti SC","STSong","SimSun",Georgia,serif; --sans:"Geist","Inter",-apple-system,"PingFang SC","Microsoft YaHei",system-ui,sans-serif; }
    * { box-sizing:border-box; }
    html,body { margin:0; background:var(--bg); color:var(--fg); font-family:var(--sans); -webkit-font-smoothing:antialiased; }
    body { padding:48px 40px 80px; }
    .top { max-width:1100px; margin:0 auto 40px; }
    .eyebrow { margin:0 0 8px; font-size:11px; font-weight:600; letter-spacing:.16em; color:var(--muted); }
    .top h1 { margin:0; font-family:var(--serif); font-size:40px; font-weight:400; line-height:1.2; }
    .lede { margin:12px 0 0; max-width:680px; font-family:var(--serif); font-size:17px; line-height:1.8; color:#3D5141; }
    .lede a { color:var(--fg); }
    .group { max-width:1100px; margin:48px auto 0; padding-top:18px; border-top:1px solid var(--hair); }
    .group-head h2 { margin:0; font-family:var(--serif); font-size:26px; font-weight:400; }
    .group-head p { margin:6px 0 0; font-size:12px; color:var(--muted-card); }
    .pair { margin-top:18px; }
    .pair-head { display:flex; align-items:baseline; gap:10px; }
    .pair-head h3 { margin:0; font-family:var(--serif); font-size:20px; font-weight:400; }
    .pair-head span { font-size:11px; color:var(--muted-card); }
    .shots { display:grid; grid-template-columns:1fr 1fr; gap:20px; margin-top:10px; }
    figure { margin:0; }
    .shot { display:block; max-width:390px; border:1px solid var(--hair); border-radius:24px; overflow:hidden; background:#DED6C6; }
    .shot img { display:block; width:100%; }
    figcaption { margin-top:7px; font-size:11.5px; color:var(--muted); }
    footer { max-width:1100px; margin:64px auto 0; padding-top:16px; border-top:1px solid var(--hair); font-size:11px; color:var(--muted-card); }
    @media (max-width:760px) { body { padding:28px 18px 56px; } .top h1 { font-size:30px; } .shots { grid-template-columns:1fr; } }
  </style>
</head>
<body>
  <header class="top">
    <p class="eyebrow">养成星球 · 家长端 · VARIANTS</p>
    <h1>8 组变体对照</h1>
    <p class="lede">左边是 31 页 baseline，右边是同一页的变体/提案。变体不替换 baseline；带“提案”的页面需要先改后端契约。共 ${groups.length} 组 / ${totalPages} 页。<a href="index.html">31 页 baseline 画廊 →</a></p>
  </header>
  ${groupHtml}
  <footer>来源：docs/design-previews/2026-11-13-parent-dna-opendesign · 变体 brief：briefs-variants/ · 生成器：_shared/run-variants.mjs</footer>
</body>
</html>`;

fs.writeFileSync(path.join(root, "variants.html"), html);
console.log(`已生成 ${path.join(root, "variants.html")}（${groups.length} 组 / ${totalPages} 页）`);
if (missing.length) {
  console.log(`注意：仍有 ${missing.length} 个文件缺失`);
  for (const m of missing) console.log(`  - ${m}`);
  process.exitCode = 1;
}
