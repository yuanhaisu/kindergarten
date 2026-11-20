#!/usr/bin/env node
/**
 * OpenDesign DNA 批次静态审计。
 *
 *   node _shared/audit.mjs              # baseline 31 页（parent/）
 *   node _shared/audit.mjs --chosen     # chosen 31 页（parent-chosen/）
 *
 * 只检查机械可判定的项；视觉与信息密度仍需人看。
 * 退出码 1 表示有 error。
 */
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const CHOSEN = process.argv.includes("--chosen");
const PAGES_ROOT = path.join(root, CHOSEN ? "parent-chosen" : "parent");
const TAB_DIRS = new Set(["01-home", "11-growth", "17-pickup", "31-profile"]);

const dirs = fs.existsSync(PAGES_ROOT)
  ? fs
      .readdirSync(PAGES_ROOT, { withFileTypes: true })
      .filter((d) => d.isDirectory() && /^\d{2}-/.test(d.name))
      .map((d) => d.name)
      .sort()
  : [];

const rows = [];
let errors = 0;
let warnings = 0;

function stripCssComments(css) {
  return css.replace(/\/\*[\s\S]*?\*\//g, "");
}

function checkCss(rel, css, problems, warns) {
  const clean = stripCssComments(css);
  if (/@import\s/.test(clean)) problems.push("CSS 使用 @import（禁止外部依赖）");
  if (/https?:\/\//.test(clean)) problems.push("CSS 包含外部 URL");

  // 橙色主按钮 / 大面积橙底
  for (const m of clean.matchAll(/[^;{}]*background(?:-color)?\s*:\s*([^;{}]+)[;}]/gi)) {
    const value = m[1].trim();
    if (/#f97316|#F97316|249\s*,\s*115\s*,\s*22/.test(value)) {
      if (/rgba?\([^)]*0?\.(0\d|1\d|2\d)\s*\)/.test(value)) continue;
      problems.push(`橙色作为背景：${value}`);
    }
  }

  // 卡片阴影：只允许悬浮导航的 --shadow-float
  for (const m of clean.matchAll(/box-shadow\s*:\s*([^;{}]+)[;}]/gi)) {
    const value = m[1].trim();
    if (/^none$/i.test(value)) continue;
    if (/var\(--shadow-(float|card)\)/.test(value)) continue;
    warns.push(`box-shadow 使用：${value}`);
  }

  // 衬线加粗
  for (const block of clean.matchAll(/\{([^}]+)\}/g)) {
    const body = block[1];
    if (/(?:--font-display|--font-serif|Instrument Serif|Noto Serif SC)/.test(body)) {
      const weight = body.match(/font-weight\s*:\s*(\d+|bold)/i);
      if (weight && (weight[1] === "bold" || Number(weight[1]) > 400)) {
        problems.push(`衬线块使用字重 ${weight[1]}`);
      }
    }
  }

  // 小字用装饰色
  for (const block of clean.matchAll(/\{([^}]+)\}/g)) {
    const body = block[1];
    const size = body.match(/font-size\s*:\s*([\d.]+)px/i);
    if (!size || Number(size[1]) > 13) continue;
    const color = body.match(/(?:^|[;\s])color\s*:\s*([^;]+)/i);
    if (!color) continue;
    if (/#728173|#8a94a3|#8A94A3|var\(--icon-soft\)|var\(--decor-blue\)/i.test(color[1])) {
      problems.push(`小字使用装饰色 ${color[1].trim()}`);
    }
  }

  if (/\brpx\b/.test(clean)) problems.push("CSS 使用 rpx（原型应为 px）");
}

for (const dir of dirs) {
  const dirPath = path.join(PAGES_ROOT, dir);
  const indexPath = path.join(dirPath, "index.html");
  if (!fs.existsSync(indexPath)) {
    errors += 1;
    rows.push({ rel: `${dir}/index.html`, problems: ["缺少 index.html"], warns: [] });
    continue;
  }
  const html = fs.readFileSync(indexPath, "utf8");
  const cssPath = path.join(dirPath, "styles.css");
  const css = fs.existsSync(cssPath) ? fs.readFileSync(cssPath, "utf8") : "";
  const jsPath = path.join(dirPath, "app.js");
  const js = fs.existsSync(jsPath) ? fs.readFileSync(jsPath, "utf8") : "";

  const problems = [];
  const warns = [];

  for (const required of ["styles.css", "app.js"]) {
    if (!fs.existsSync(path.join(dirPath, required))) problems.push(`缺少 ${required}`);
  }
  for (const [name, content] of [["index.html", html], ["styles.css", css], ["app.js", js]]) {
    if (Buffer.byteLength(content) > 20 * 1024) problems.push(`${name} 超过 20KB`);
  }

  const h1 = (html.match(/<h1[\s>]/g) || []).length;
  if (h1 !== 1) problems.push(`h1 数量为 ${h1}（应为 1）`);

  const all = `${html}\n${css}\n${js}`;
  // xmlns 是 SVG 命名空间，不是外部资源。
  const resourceRefs = all.replace(/xmlns="http:\/\/www\.w3\.org\/2000\/svg"/g, "");
  if (/https?:\/\//.test(resourceRefs)) problems.push("包含外部 URL");
  if (/lorem|Lorem|TODO|FIXME|待补充/.test(all)) problems.push("包含占位文案");
  if (/<img[\s>]/i.test(html)) warns.push("使用 <img>（原型应使用内联 SVG 或色块）");
  if (/rpx/.test(all)) problems.push("包含 rpx");
  if (/\bfetch\s*\(|XMLHttpRequest|axios/.test(js)) problems.push("app.js 发起网络请求");
  if (/fonts\.googleapis|fonts\.gstatic/.test(all)) problems.push("引用 Google Fonts");

  // 托盘导航：tab 页必须有 4 项，非 tab 页不得有
  const hasNav =
    /<nav[\s>]/.test(html) &&
    ["首页", "成长", "接送", "我的"].every((label) => html.includes(label));
  const isTab = TAB_DIRS.has(dir);
  if (isTab && !hasNav) problems.push("tab 页缺少 4 项悬浮导航");
  if (!isTab && hasNav) problems.push("非 tab 页出现底部导航");

  // 行内高亮：内容页应至少出现 .hl（登录/政策/设置等纯表单页可豁免）
  const exempt = new Set(["02-login", "03-bind", "04-consent", "05-policy", "09-leave-create", "20-pickup-authorized-create", "28-feedback"]);
  if (!exempt.has(dir) && !/class="[^"]*\bhl\b/.test(html) && !/class='[^']*\bhl\b/.test(html)) {
    warns.push("未发现行内高亮 .hl");
  }

  // 橙色：主按钮规则里出现橙色
  if (/(?:class="[^"]*\bbtn[^"]*primary[^"]*"|\.btn--primary)[\s\S]{0,300}/.test(html)) {
    if (/#f97316|#F97316/i.test(css.slice(css.indexOf(".btn--primary"), css.indexOf(".btn--primary") + 400))) {
      problems.push("主按钮使用橙色");
    }
  }

  if (dir === "29-settings") {
    const groupTitle = css.match(/\.group-title\s*\{([^}]*)\}/);
    const rule = groupTitle ? groupTitle[1] : "";
    if (!/font-family\s*:\s*var\(--font-display\)/.test(rule)) {
      problems.push("29 分组标题未使用 DNA 衬线字体");
    }
    if (!/font-size\s*:\s*21px/.test(rule) || !/font-weight\s*:\s*400/.test(rule)) {
      problems.push("29 分组标题未使用 H2 21px/400");
    }
  }

  if (dir === "11-growth") {
    const footer = css.match(/\.feed-footer\s*\{([^}]*)\}/);
    const rule = footer ? footer[1] : "";
    if (/--muted-on-surface|#5E6B7A/i.test(rule) || !/--muted\b|#55616F/i.test(rule)) {
      problems.push("11 页脚未使用 on-bone muted 色（#55616F）");
    }
  }

  checkCss(`${dir}/styles.css`, css, problems, warns);

  if (problems.length) errors += problems.length;
  if (warns.length) warnings += warns.length;
  rows.push({ rel: `${dir}/index.html`, problems, warns });
}

for (const row of rows) {
  if (!row.problems.length && !row.warns.length) continue;
  console.log(`\n${row.rel}`);
  for (const p of row.problems) console.log(`  ERROR ${p}`);
  for (const w of row.warns) console.log(`  WARN  ${w}`);
}

console.log(`\n共检查 ${rows.length} 个页面：${errors} 个 error，${warnings} 个 warning。`);
if (errors) process.exitCode = 1;
