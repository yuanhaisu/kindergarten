#!/usr/bin/env node
/**
 * 把官方圆形 logo 加到登录页与「我的」页脚。
 *
 *   node _shared/add-brand-logo.mjs
 *
 * 只改展示层；logo 资源是 _shared/logo.png（来自 parent/src/static/logo.png）。
 */
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");

function patch(rel, replacements) {
  const file = path.join(root, rel);
  let text = fs.readFileSync(file, "utf8");
  for (const [from, to, optional] of replacements) {
    if (!text.includes(from)) {
      if (optional) continue;
      throw new Error(`missing in ${rel}: ${from.slice(0, 80)}`);
    }
    text = text.split(from).join(to);
  }
  fs.writeFileSync(file, text, "utf8");
  console.log(`patched ${rel}`);
}

const loginMarkup = '      <section class="brand" aria-labelledby="login-title" data-od-id="brand-header">\n        <p class="brand__overline">养成星球 · NURTURE PLANET</p>';
const loginNew = '      <section class="brand" aria-labelledby="login-title" data-od-id="brand-header">\n        <img class="brand-logo" src="../../_shared/logo.png" alt="养成星球" width="56" height="56" />\n        <p class="brand__overline">养成星球 · NURTURE PLANET</p>';

const brandCss = '.brand {\n  flex: 0 0 auto;\n}';
const brandCssNew = '.brand {\n  flex: 0 0 auto;\n}\n\n.brand-logo {\n  display: block;\n  width: 56px;\n  height: 56px;\n  margin-bottom: 18px;\n  border: 1px solid var(--hair);\n  border-radius: 50%;\n}';

for (const rel of ["parent/02-login", "parent-chosen/02-login"]) {
  patch(`${rel}/index.html`, [[loginMarkup, loginNew]]);
  patch(`${rel}/styles.css`, [[brandCss, brandCssNew]]);
}

const versionMarkup = '<p class="version" data-od-id="profile-version">养成星球 · 家长端 v1.0.0</p>';
const versionCss = '.version {\n  margin: 10px 0 0;\n  color: var(--muted);\n  font-size: 11px;\n  line-height: 1.5;\n  letter-spacing: 0.02em;\n  text-wrap: pretty;\n}';
const versionCssNew = `${versionCss}\n\n.version-logo {\n  width: 16px;\n  height: 16px;\n  margin-right: 6px;\n  vertical-align: -3px;\n  border-radius: 50%;\n}`;

for (const rel of ["parent/31-profile", "parent-chosen/31-profile"]) {
  patch(`${rel}/index.html`, [[versionMarkup, '<p class="version" data-od-id="profile-version"><img class="version-logo" src="../../_shared/logo.png" alt="" width="16" height="16" />养成星球 · 家长端 v1.0.0</p>']]);
  patch(`${rel}/styles.css`, [[versionCss, versionCssNew]]);
}

patch("parent-variants/v8-single-child/31-profile/index.html", [
  [versionMarkup, '<p class="version" data-od-id="profile-version"><img class="version-logo" src="../../../_shared/logo.png" alt="" width="16" height="16" />养成星球 · 家长端 v1.0.0</p>'],
]);
patch("parent-variants/v8-single-child/31-profile/styles.css", [[versionCss, versionCssNew]]);

console.log("done");
