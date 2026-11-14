#!/usr/bin/env node
/**
 * 家长端全量设计稿静态审计。
 *
 *   node _shared/audit.mjs
 *
 * 只做机械检查；对比度、视觉与信息密度仍需人看。退出码 1 表示有 error。
 */
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const TAB_DIRS = new Set(["01-home", "11-growth", "17-pickup", "31-profile"]);

const dirs = fs
  .readdirSync(root, { withFileTypes: true })
  .filter((d) => d.isDirectory() && /^\d{2}-/.test(d.name))
  .map((d) => d.name)
  .sort();

const rows = [];
let errors = 0;
let warnings = 0;

for (const dir of dirs) {
  const dirPath = path.join(root, dir);
  const files = fs.readdirSync(dirPath).filter((f) => f.endsWith(".html")).sort();
  for (const file of files) {
    const rel = `${dir}/${file}`;
    const html = fs.readFileSync(path.join(dirPath, file), "utf8");
    const problems = [];
    const warns = [];

    const h1 = (html.match(/<h1[\s>]/g) || []).length;
    const isTab = TAB_DIRS.has(dir);
    const hasNav = html.includes('class="mp-nav"');
    const hasPageBar = html.includes('class="page-bar"');
    const isIndex = file === "index.html";

    if (!html.includes('class="mp-app"')) problems.push("缺少 .mp-app 外壳");
    if (isIndex && h1 !== 1) problems.push(`h1 数量为 ${h1}（应为 1）`);
    if (isTab && !hasNav) problems.push("tab 页缺少 .mp-nav");
    if (!isTab && hasNav) problems.push("非 tab 页出现 .mp-nav");
    if (!isTab && dir !== "02-login" && !hasPageBar) problems.push("非 tab 页缺少 .page-bar");

    if (/fonts\.googleapis|fonts\.gstatic/.test(html)) problems.push("引用了 Google Fonts");
    if (/https?:\/\//.test(html)) problems.push("包含外部 URL");
    if (/lorem|Lorem|TODO|FIXME/.test(html)) problems.push("包含占位文案");
    if (/<img[\s>]/i.test(html)) problems.push("使用 <img>（设计稿不允许外链图片）");

    // 文字色误用：color: #728173 / #8A94A3 / #C36540
    const textColorRe = /(?:^|[;{\s])color:\s*(#728173|#8a94a3|#c36540)\b/gi;
    if (textColorRe.test(html)) warns.push("使用了装饰色作为文字色（需人工确认字号≥24px 或为图标）");

    // 衬线 + 粗体：逐条 CSS 规则检查
    for (const block of html.matchAll(/\{([^}]+)\}/g)) {
      const body = block[1];
      if (/font-family:\s*var\(--serif\)/.test(body)) {
        const weight = body.match(/font-weight:\s*(\d+|bold)/);
        if (weight && Number(weight[1]) > 400) warnings.push(`衬线块使用字重 ${weight[1]}`);
      }
    }

    if (/box-shadow/.test(html)) warns.push("页面内使用 box-shadow（仅共享悬浮导航允许）");

    // 小字检查：11–12px 的文字不要用 --ink-soft（本批次静态写法）
    if (/font-size:\s*1[12](?:\.5)?px[^}]*color:\s*var\(--ink-soft\)/.test(html)) {
      problems.push("小字使用 --ink-soft");
    }

    if (problems.length) errors += problems.length;
    if (warns.length) warnings += warns.length;
    rows.push({ rel, problems, warns });
  }
}

for (const row of rows) {
  if (!row.problems.length && !row.warns.length) continue;
  console.log(`\n${row.rel}`);
  for (const p of row.problems) console.log(`  ERROR ${p}`);
  for (const w of row.warns) console.log(`  WARN  ${w}`);
}

console.log(`\n共检查 ${rows.length} 个页面文件：${errors} 个 error，${warnings} 个 warning。`);
if (errors) process.exitCode = 1;
