#!/usr/bin/env node
/**
 * 统一二级页返回按钮的形态：page-bar 内一律使用无边框 chevron（icon-btn--plain）。
 * 设计稿评审反馈：圆形返回钮与文字型页头并置时视觉过重，且各页不一致。
 *
 *   node _shared/normalize.mjs
 */
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const dirs = fs
  .readdirSync(root, { withFileTypes: true })
  .filter((d) => d.isDirectory() && /^\d{2}-/.test(d.name))
  .map((d) => d.name);

let changed = 0;
for (const dir of dirs) {
  const dirPath = path.join(root, dir);
  for (const file of fs.readdirSync(dirPath).filter((f) => f.endsWith(".html"))) {
    const filePath = path.join(dirPath, file);
    const html = fs.readFileSync(filePath, "utf8");
    const next = html.replace(/<(div|header) class="page-bar">([\s\S]*?)<\/\1>/g, (block) =>
      block.replace(/class="icon-btn"/, 'class="icon-btn icon-btn--plain"'),
    );
    if (next !== html) {
      fs.writeFileSync(filePath, next);
      changed += 1;
      console.log(`normalized ${dir}/${file}`);
    }
  }
}
console.log(`共调整 ${changed} 个文件。`);
