#!/usr/bin/env node
/**
 * 把「变体 delta」与「baseline brief」拼成一份可交给 od-design-page 的完整 brief。
 *
 *   node _shared/compose-variant-briefs.mjs
 *
 * 输出：briefs-variants/<group>/<slug>.composed.md
 * 变体 delta 在前，baseline 在后；delta 里已写明「冲突时以本文件为准」。
 */
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const variantsDir = path.join(root, "briefs-variants");
const baselineDir = path.join(root, "briefs");

const pages = [
  ["v1-leave-day", "01-home"],
  ["v1-leave-day", "17-pickup"],
  ["v2-task-points", "21-tasks"],
  ["v2-task-points", "24-points"],
  ["v3-notice-signup", "07-notice-detail"],
  ["v4-settings-simple", "29-settings"],
  ["v5-report-month", "14-growth-report"],
  ["v6-female-child", "27-student"],
  ["v7-age5-class", "27-student"],
  ["v8-single-child", "31-profile"],
];

let count = 0;
for (const [group, slug] of pages) {
  const variantPath = path.join(variantsDir, group, `${slug}.md`);
  const baselinePath = path.join(baselineDir, `${slug}.md`);
  if (!fs.existsSync(variantPath) || !fs.existsSync(baselinePath)) {
    console.error(`跳过（缺文件）：${group}/${slug}`);
    continue;
  }
  const variant = fs.readFileSync(variantPath, "utf8").trim();
  const baseline = fs.readFileSync(baselinePath, "utf8").trim();
  const composed = `${variant}

---

# 以下为 baseline brief 原文（未覆盖处按此）

${baseline}
`;
  fs.writeFileSync(path.join(variantsDir, group, `${slug}.composed.md`), composed, "utf8");
  count += 1;
  console.log(`composed ${group}/${slug}`);
}
console.log(`完成，生成 ${count} 份 composed brief。`);
