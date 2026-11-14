#!/usr/bin/env node
/**
 * 统一底部导航的「成长」图标为 V1 的叶子图标。
 *
 *   node _shared/unify-growth-icon.mjs
 *   node _shared/unify-growth-icon.mjs --check
 *
 * 只改底部 <nav> 里的成长 tab，不动快捷入口/正文里的成长图标。
 */
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");

const LEAF = '<svg class="icon" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M20 4C12 4 5 8 5 16c0 2 1 4 3 4 8 0 12-7 12-16Z"/><path d="M5 20c2-5 6-8 11-10"/></svg>';

function pages() {
  const out = [];
  for (const slug of fs.readdirSync(path.join(root, "parent"))) {
    const f = path.join(root, "parent", slug, "index.html");
    if (fs.existsSync(f)) out.push(f);
  }
  const variants = path.join(root, "parent-variants");
  if (fs.existsSync(variants)) {
    for (const group of fs.readdirSync(variants)) {
      const gdir = path.join(variants, group);
      if (!fs.statSync(gdir).isDirectory()) continue;
      for (const slug of fs.readdirSync(gdir)) {
        const f = path.join(gdir, slug, "index.html");
        if (fs.existsSync(f)) out.push(f);
      }
    }
  }
  return out;
}

const check = process.argv.includes("--check");
let changed = 0;
let skipped = 0;
let failed = 0;

for (const file of pages()) {
  const rel = path.relative(root, file);
  const html = fs.readFileSync(file, "utf8");
  const navMatch =
    html.match(/<nav\b[^>]*class="[^"]*mp-nav[^"]*"[\s\S]*?<\/nav>/i) ||
    html.match(/<nav\b[^>]*aria-label="[^"]*(?:主导航|底部导航)[^"]*"[\s\S]*?<\/nav>/i);
  if (!navMatch) {
    failed += 1;
    console.log(`SKIP ${rel}（找不到底部 nav）`);
    continue;
  }
  const nav = navMatch[0];
  const itemMatch = nav.match(
    /<(a|button)\b[^>]*(?:data-tab="(?:growth|成长)"|data-od-id="tab-growth")[^>]*>[\s\S]*?<\/\1>/i,
  );
  if (!itemMatch) {
    failed += 1;
    console.log(`SKIP ${rel}（找不到成长 tab）`);
    continue;
  }
  const item = itemMatch[0];
  if (item.includes("M20 4C12 4")) {
    skipped += 1;
    continue;
  }
  const svgMatch = item.match(/<svg\b[\s\S]*?<\/svg>/i);
  if (!svgMatch) {
    failed += 1;
    console.log(`SKIP ${rel}（成长 tab 没有 svg）`);
    continue;
  }
  const newItem = item.replace(svgMatch[0], LEAF);
  const newHtml = html.replace(nav, nav.replace(item, newItem));
  if (check) {
    console.log(`WOULD ${rel}`);
    changed += 1;
    continue;
  }
  fs.writeFileSync(file, newHtml, "utf8");
  console.log(`PATCH ${rel}`);
  changed += 1;
}

console.log(`${check ? "待改" : "已改"} ${changed} 页；已是 V1 叶子 ${skipped} 页；异常 ${failed} 页。`);
if (failed) process.exitCode = 1;
