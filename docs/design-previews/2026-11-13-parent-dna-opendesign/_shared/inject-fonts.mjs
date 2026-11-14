#!/usr/bin/env node
/**
 * 给 OpenDesign 产物挂上 DNA 自托管字体（Instrument Serif + Noto Serif SC）。
 *
 *   node _shared/inject-fonts.mjs parent/01-home [...]
 *
 * 字体声明放在共享的 _shared/fonts.css，页面只加一行 <link>，避免把
 * styles.css 撑过 20KB；幂等。
 */
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const dirs = process.argv.slice(2);
if (!dirs.length) {
  console.error("usage: node _shared/inject-fonts.mjs parent/01-home [...]");
  process.exit(2);
}

const MARKER = "/* DNA self-hosted fonts (injected by _shared/inject-fonts.mjs) */";

function linkFor(rel) {
  const pageDir = path.resolve(root, rel);
  const sharedFonts = path.join(root, "_shared", "fonts.css");
  const relPath = path.relative(pageDir, sharedFonts).split(path.sep).join("/");
  return `<link rel="stylesheet" href="${relPath}" />`;
}

function stripInjectedFonts(css) {
  if (!css.startsWith(MARKER)) return css;
  const end = css.indexOf("/* 家长端");
  return end === -1 ? css : css.slice(end);
}

let changed = 0;
for (const rel of dirs) {
  const dir = path.resolve(root, rel);
  const cssPath = path.join(dir, "styles.css");
  const htmlPath = path.join(dir, "index.html");
  if (!fs.existsSync(htmlPath)) {
    console.error(`跳过（无 index.html）：${rel}`);
    continue;
  }

  if (fs.existsSync(cssPath)) {
    const css = fs.readFileSync(cssPath, "utf8");
    const stripped = stripInjectedFonts(css);
    if (stripped !== css) {
      fs.writeFileSync(cssPath, stripped, "utf8");
      console.log(`清理 styles.css 内联字体：${rel}`);
      changed += 1;
    }
  }

  const html = fs.readFileSync(htmlPath, "utf8");
  if (!html.includes("_shared/fonts.css")) {
    const link = linkFor(rel);
    const patched = html.includes('<link rel="stylesheet" href="styles.css"')
      ? html.replace(
          '<link rel="stylesheet" href="styles.css"',
          `${link}\n  <link rel="stylesheet" href="styles.css"`,
        )
      : html.replace("</head>", `  ${link}\n</head>`);
    fs.writeFileSync(htmlPath, patched, "utf8");
    console.log(`挂载共享字体：${rel}`);
    changed += 1;
  }
}
console.log(`完成，修改 ${changed} 个文件。`);
