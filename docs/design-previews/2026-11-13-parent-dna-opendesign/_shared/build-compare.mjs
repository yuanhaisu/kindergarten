#!/usr/bin/env node
/**
 * 生成 31 页的并排对照页（手工稿 vs OpenDesign）。
 *
 *   node _shared/build-compare.mjs
 */
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const groups = [
  ["核心页面", ["01-home|首页", "11-growth|成长时光|empty", "17-pickup|接送|away,done", "31-profile|我的"]],
  ["登录与协议", ["02-login|登录", "03-bind|绑定孩子", "04-consent|同意与协议", "05-policy|政策全文"]],
  ["通知与请假", ["06-notice-index|通知列表|empty", "07-notice-detail|通知详情", "08-leave-index|请假记录|empty", "09-leave-create|新建请假|error", "10-leave-detail|请假详情"]],
  ["成长与相册", ["12-growth-detail|成长详情", "13-growth-album|成长相册", "14-growth-report|成长报告", "15-album-index|班级相册|empty", "16-album-detail|相册详情"]],
  ["接送与积分", ["18-pickup-qrcode|接送码", "19-pickup-authorized|可接送人|empty", "20-pickup-authorized-create|添加授权人", "24-points|积分与成就"]],
  ["任务与班级", ["21-tasks|亲子任务|empty", "22-tasks-detail|任务详情", "23-tasks-wall|班级墙", "25-class|班级信息", "26-calendar|班级日历"]],
  ["档案与设置", ["27-student|幼儿信息", "28-feedback|意见反馈", "29-settings|设置", "30-messages|消息|empty"]],
];

const esc = (s) => s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
const stateLabel = { empty: "空态", error: "错误态", away: "未入园", done: "已接走" };

function pageHtml(spec) {
  const [dir, title, states] = spec.split("|");
  const variants = (states ? states.split(",") : [])
    .map((s) => `<a class="mini" href="parent/${dir}/index.html?state=${s}"><img src="parent/${dir}/preview-${s}.png" alt="${esc(title)} · ${stateLabel[s] || s}" loading="lazy" /><span>${stateLabel[s] || s}</span></a>`)
    .join("");
  return `
    <section class="pair">
      <div class="pair-head"><h2>${esc(title)}</h2><span>${dir}</span></div>
      <div class="shots">
        <figure>
          <a class="shot" href="../2026-11-12-parent-dna-full/${dir}/index.html"><img src="../2026-11-12-parent-dna-full/${dir}/preview.png" alt="手工版 ${esc(title)}" loading="lazy" /></a>
          <figcaption>手工版</figcaption>
        </figure>
        <figure>
          <a class="shot" href="parent/${dir}/index.html"><img src="parent/${dir}/preview.png" alt="OpenDesign ${esc(title)}" loading="lazy" /></a>
          <figcaption>OpenDesign 版</figcaption>
        </figure>
      </div>
      ${variants ? `<div class="variants">${variants}</div>` : ""}
    </section>`;
}

const groupHtml = groups
  .map(([title, pages]) => `<h1 class="group-title">${esc(title)}</h1>${pages.map(pageHtml).join("")}`)
  .join("");

const html = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <meta name="color-scheme" content="light" />
  <title>手工稿 vs OpenDesign · 31 页对照</title>
  <link rel="stylesheet" href="_shared/fonts.css" />
  <style>
    :root { --bg:#ECE6D7; --surface:#fff; --fg:#173220; --muted:#55616F; --muted-card:#5E6B7A; --hair:rgba(23,50,32,.1); --serif:"Instrument Serif","Noto Serif SC","Songti SC","STSong","SimSun",Georgia,serif; --sans:"Geist","Inter",-apple-system,"PingFang SC","Microsoft YaHei",system-ui,sans-serif; }
    * { box-sizing:border-box; }
    html,body { margin:0; background:var(--bg); color:var(--fg); font-family:var(--sans); -webkit-font-smoothing:antialiased; }
    body { padding:48px 40px 80px; }
    .top { max-width:1180px; margin:0 auto 40px; }
    .eyebrow { margin:0 0 8px; font-size:11px; font-weight:600; letter-spacing:.16em; color:var(--muted); }
    .top h1 { margin:0; font-family:var(--serif); font-size:40px; font-weight:400; line-height:1.2; }
    .lede { margin:12px 0 0; max-width:680px; font-family:var(--serif); font-size:17px; line-height:1.8; color:#3D5141; }
    .lede a { color:var(--fg); }
    .group-title { max-width:1180px; margin:48px auto 0; padding-top:16px; border-top:1px solid var(--hair); font-family:var(--serif); font-size:26px; font-weight:400; }
    .pair { max-width:1180px; margin:18px auto 0; }
    .pair-head { display:flex; align-items:baseline; gap:10px; }
    .pair-head h2 { margin:0; font-family:var(--serif); font-size:20px; font-weight:400; }
    .pair-head span { font-size:11px; color:var(--muted-card); }
    .shots { display:grid; grid-template-columns:1fr 1fr; gap:20px; margin-top:10px; }
    figure { margin:0; }
    .shot { display:block; max-width:390px; border:1px solid var(--hair); border-radius:24px; overflow:hidden; background:#DED6C6; }
    .shot img { display:block; width:100%; }
    figcaption { margin-top:7px; font-size:11.5px; color:var(--muted); }
    .variants { display:flex; gap:8px; margin-top:8px; }
    .mini { display:grid; gap:3px; justify-items:center; text-decoration:none; color:var(--muted-card); font-size:10.5px; }
    .mini img { width:48px; height:104px; object-fit:cover; object-position:top; border:1px solid var(--hair); border-radius:9px; }
    @media (max-width:760px) { body { padding:28px 18px 56px; } .top h1 { font-size:30px; } .shots { grid-template-columns:1fr; } }
  </style>
</head>
<body>
  <header class="top">
    <p class="eyebrow">养成星球 · 家长端 · SIDE BY SIDE</p>
    <h1>手工稿 vs OpenDesign</h1>
    <p class="lede">31 页逐页并排。左为手工全量稿，右为同一 DNA / 同一 brief 的 OpenDesign 版本；状态变体放在右侧下方。</p>
  </header>
  ${groupHtml}
</body>
</html>`;

fs.writeFileSync(path.join(root, "compare.html"), html);
console.log(`已生成 ${path.join(root, "compare.html")}`);
